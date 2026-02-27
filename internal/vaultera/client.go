// Package vaultera provides an HTTP client for the Vaultera PCI proxy API.
// API base: https://pci.vaultera.co/api/v1
package vaultera

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Client is an authenticated Vaultera PCI API client.
type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new Vaultera client.
func NewClient(apiKey, baseURL string) *Client {
	return &Client{
		apiKey:  apiKey,
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ---------------------------------------------------------------------------
// Request / Response types
// ---------------------------------------------------------------------------

// Card holds raw card data used for direct tokenization.
type Card struct {
	CardNumber      string `json:"card_number"`
	CardType        string `json:"card_type,omitempty"`
	CardholderName  string `json:"cardholder_name,omitempty"`
	ServiceCode     string `json:"service_code,omitempty"`
	ExpirationMonth string `json:"expiration_month"`
	ExpirationYear  string `json:"expiration_year"`
}

// CardResponse is returned after tokenizing or fetching a card.
type CardResponse struct {
	CardToken      string `json:"card_token"`
	CardNumberMask string `json:"card_number_mask"`
	CardType       string `json:"card_type"`
	CardholderName string `json:"cardholder_name"`
	ExpirationMonth string `json:"expiration_month"`
	ExpirationYear  string `json:"expiration_year"`
}

// CaptureRequest describes a server-to-server capture (detokenize-and-forward) call.
type CaptureRequest struct {
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    string            `json:"body,omitempty"`
	Profile string            `json:"profile,omitempty"`
}

// CaptureResponse is the raw response from the target endpoint after Vaultera
// injects the card data.
type CaptureResponse struct {
	StatusCode int             `json:"status_code"`
	Headers    map[string]string `json:"headers,omitempty"`
	Body       json.RawMessage `json:"body"`
}

// SendRequest describes a detokenize-and-forward call for a stored card.
type SendRequest struct {
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    string            `json:"body,omitempty"`
}

// SendResponse wraps the response from the downstream gateway.
type SendResponse struct {
	StatusCode int             `json:"status_code"`
	Headers    map[string]string `json:"headers,omitempty"`
	Body       json.RawMessage `json:"body"`
}

// SessionTokenRequest is used to create a new session token.
type SessionTokenRequest struct {
	SessionToken struct {
		Scope string `json:"scope"`
	} `json:"session_token"`
}

// SessionTokenResponse contains the newly created session token.
type SessionTokenResponse struct {
	Token string `json:"token"`
	Scope string `json:"scope"`
}

// ---------------------------------------------------------------------------
// Helper
// ---------------------------------------------------------------------------

func (c *Client) do(ctx context.Context, method, path string, queryParams url.Values, body any) ([]byte, int, error) {
	endpoint := c.baseURL + path

	if queryParams == nil {
		queryParams = url.Values{}
	}
	queryParams.Set("api_key", c.apiKey)

	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, 0, fmt.Errorf("vaultera: invalid URL %q: %w", endpoint, err)
	}
	u.RawQuery = queryParams.Encode()

	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, 0, fmt.Errorf("vaultera: marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), reqBody)
	if err != nil {
		return nil, 0, fmt.Errorf("vaultera: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("vaultera: http do: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("vaultera: read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, resp.StatusCode, fmt.Errorf("vaultera: API error %d: %s", resp.StatusCode, string(data))
	}

	return data, resp.StatusCode, nil
}

// ---------------------------------------------------------------------------
// Cards
// ---------------------------------------------------------------------------

// CreateCard tokenizes a credit card and returns the stored card token.
// POST /cards
func (c *Client) CreateCard(ctx context.Context, card Card) (*CardResponse, error) {
	payload := map[string]any{"card": card}
	data, _, err := c.do(ctx, http.MethodPost, "/cards", nil, payload)
	if err != nil {
		return nil, err
	}
	var wrapper struct {
		Card CardResponse `json:"card"`
	}
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return nil, fmt.Errorf("vaultera: decode card response: %w", err)
	}
	return &wrapper.Card, nil
}

// GetCard retrieves masked card information for the given token.
// GET /cards/{card_token}
func (c *Client) GetCard(ctx context.Context, cardToken string) (*CardResponse, error) {
	data, _, err := c.do(ctx, http.MethodGet, "/cards/"+cardToken, nil, nil)
	if err != nil {
		return nil, err
	}
	var wrapper struct {
		Card CardResponse `json:"card"`
	}
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return nil, fmt.Errorf("vaultera: decode card response: %w", err)
	}
	return &wrapper.Card, nil
}

// DeleteCard removes a stored card token.
// DELETE /cards/{card_token}
func (c *Client) DeleteCard(ctx context.Context, cardToken string) error {
	_, _, err := c.do(ctx, http.MethodDelete, "/cards/"+cardToken, nil, nil)
	return err
}

// SendCard detokenizes a stored card and forwards the request to a downstream
// gateway (e.g. Stripe, Payzone). Use placeholders like %CARD_NUMBER% in Body.
// POST /cards/{card_token}/send
func (c *Client) SendCard(ctx context.Context, cardToken string, req SendRequest) (*SendResponse, error) {
	data, _, err := c.do(ctx, http.MethodPost, "/cards/"+cardToken+"/send", nil, req)
	if err != nil {
		return nil, err
	}
	var resp SendResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("vaultera: decode send response: %w", err)
	}
	return &resp, nil
}

// ---------------------------------------------------------------------------
// Capture
// ---------------------------------------------------------------------------

// Capture pulls card data from a third-party HTTPS endpoint and tokenizes it.
// POST /capture
func (c *Client) Capture(ctx context.Context, req CaptureRequest) (*CaptureResponse, error) {
	params := url.Values{}
	if req.Method != "" {
		params.Set("method", req.Method)
	}
	if req.URL != "" {
		params.Set("url", req.URL)
	}
	if req.Profile != "" {
		params.Set("profile", req.Profile)
	}

	data, _, err := c.do(ctx, http.MethodPost, "/capture", params, req)
	if err != nil {
		return nil, err
	}
	var resp CaptureResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("vaultera: decode capture response: %w", err)
	}
	return &resp, nil
}

// CaptureFormURL returns the URL for the embeddable secure card-input iframe.
// The caller must supply a session token with scope "card".
// GET /capture_form?session_token={token}
func (c *Client) CaptureFormURL(sessionToken string) string {
	params := url.Values{}
	params.Set("session_token", sessionToken)
	return c.baseURL + "/capture_form?" + params.Encode()
}

// ---------------------------------------------------------------------------
// Session tokens
// ---------------------------------------------------------------------------

// CreateSessionToken creates a disposable session token with the given scope.
// Valid scopes: "card" (iframe capture), "show_card" (iframe show).
// POST /session_tokens
func (c *Client) CreateSessionToken(ctx context.Context, scope string) (*SessionTokenResponse, error) {
	req := SessionTokenRequest{}
	req.SessionToken.Scope = scope

	data, _, err := c.do(ctx, http.MethodPost, "/session_tokens", nil, req)
	if err != nil {
		return nil, err
	}
	var wrapper struct {
		SessionToken SessionTokenResponse `json:"session_token"`
	}
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return nil, fmt.Errorf("vaultera: decode session_token response: %w", err)
	}
	return &wrapper.SessionToken, nil
}
