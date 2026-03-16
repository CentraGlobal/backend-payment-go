// Package pcibooking provides an HTTP client for the PCI Booking API.
// API docs: https://developers.pcibooking.net/
package pcibooking

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/CentraGlobal/backend-payment-go/internal/processor"
)

var _ processor.Processor = (*Client)(nil)

type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

func NewClient(apiKey, baseURL string) *Client {
	return &Client{
		apiKey:  apiKey,
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) Name() string {
	return "pcibooking"
}

func (c *Client) do(ctx context.Context, method, path string, queryParams url.Values, body any) ([]byte, int, error) {
	endpoint := c.baseURL + path
	if queryParams == nil {
		queryParams = url.Values{}
	}
	queryParams.Set("api_key", c.apiKey)

	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, 0, fmt.Errorf("pcibooking: invalid URL %q: %w", endpoint, err)
	}
	u.RawQuery = queryParams.Encode()

	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, 0, fmt.Errorf("pcibooking: marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), reqBody)
	if err != nil {
		return nil, 0, fmt.Errorf("pcibooking: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("pcibooking: http do: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("pcibooking: read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, resp.StatusCode, fmt.Errorf("pcibooking: API error %d: %s", resp.StatusCode, string(data))
	}

	return data, resp.StatusCode, nil
}

type tokenizationRequest struct {
	Paycard paycardData `json:"paycard"`
}

type paycardData struct {
	CardNumber      string `json:"CardNumber"`
	CardholderName  string `json:"CardholderName"`
	ExpirationMonth string `json:"ExpirationMM"`
	ExpirationYear  string `json:"ExpirationYYYY"`
	CardType        string `json:"CardType,omitempty"`
}

type tokenizationResponse struct {
	Paycard paycardResponse `json:"paycard"`
}

type paycardResponse struct {
	Token           string `json:"Token"`
	CardNumberMask  string `json:"CardNumberMask"`
	CardType        string `json:"CardType"`
	CardholderName  string `json:"CardholderName"`
	ExpirationMonth string `json:"ExpirationMM"`
	ExpirationYear  string `json:"ExpirationYYYY"`
}

type retrievePaycardResponse struct {
	Paycard retrievePaycardData `json:"paycard"`
}

type retrievePaycardData struct {
	Token           string `json:"Token"`
	CardNumberMask  string `json:"CardNumberMask"`
	CardType        string `json:"CardType"`
	CardholderName  string `json:"CardholderName"`
	ExpirationMonth string `json:"ExpirationMM"`
	ExpirationYear  string `json:"ExpirationYYYY"`
}

type relayRequest struct {
	CardToken string            `json:"cardToken"`
	Method    string            `json:"method"`
	URL       string            `json:"url"`
	Headers   map[string]string `json:"headers,omitempty"`
	Body      string            `json:"body,omitempty"`
}

type relayResponse struct {
	StatusCode int               `json:"statusCode"`
	Headers    map[string]string `json:"headers,omitempty"`
	Body       json.RawMessage   `json:"body"`
}

type sessionTokenRequest struct {
	SessionToken struct {
		Scope string `json:"scope"`
	} `json:"session_token"`
}

type sessionTokenResponse struct {
	SessionToken struct {
		Token string `json:"token"`
		Scope string `json:"scope"`
	} `json:"session_token"`
}

func (c *Client) CreateCard(ctx context.Context, card processor.Card) (*processor.CardResponse, error) {
	req := tokenizationRequest{
		Paycard: paycardData{
			CardNumber:      card.CardNumber,
			CardholderName:  card.CardholderName,
			CardType:        card.CardType,
			ExpirationMonth: card.ExpirationMonth,
			ExpirationYear:  card.ExpirationYear,
		},
	}

	data, _, err := c.do(ctx, http.MethodPost, "/api/payments/paycard/capture", nil, req)
	if err != nil {
		return nil, err
	}

	var resp tokenizationResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("pcibooking: decode tokenization response: %w", err)
	}

	return &processor.CardResponse{
		CardToken:       resp.Paycard.Token,
		CardMask:        resp.Paycard.CardNumberMask,
		CardType:        resp.Paycard.CardType,
		CardholderName:  resp.Paycard.CardholderName,
		ExpirationMonth: resp.Paycard.ExpirationMonth,
		ExpirationYear:  resp.Paycard.ExpirationYear,
	}, nil
}

func (c *Client) GetCard(ctx context.Context, cardToken string) (*processor.CardResponse, error) {
	params := url.Values{}
	params.Set("token", cardToken)

	data, _, err := c.do(ctx, http.MethodGet, "/api/payments/paycard", params, nil)
	if err != nil {
		return nil, err
	}

	var resp retrievePaycardResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("pcibooking: decode retrieve paycard response: %w", err)
	}

	return &processor.CardResponse{
		CardToken:       resp.Paycard.Token,
		CardMask:        resp.Paycard.CardNumberMask,
		CardType:        resp.Paycard.CardType,
		CardholderName:  resp.Paycard.CardholderName,
		ExpirationMonth: resp.Paycard.ExpirationMonth,
		ExpirationYear:  resp.Paycard.ExpirationYear,
	}, nil
}

func (c *Client) DeleteCard(ctx context.Context, cardToken string) error {
	params := url.Values{}
	params.Set("token", cardToken)

	_, _, err := c.do(ctx, http.MethodDelete, "/api/payments/paycard", params, nil)
	return err
}

func (c *Client) SendCard(ctx context.Context, cardToken string, req processor.SendRequest) (*processor.SendResponse, error) {
	relayReq := relayRequest{
		CardToken: cardToken,
		Method:    req.Method,
		URL:       req.URL,
		Headers:   req.Headers,
		Body:      req.Body,
	}

	data, _, err := c.do(ctx, http.MethodPost, "/api/payments/paycard/relay", nil, relayReq)
	if err != nil {
		return nil, err
	}

	var resp relayResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("pcibooking: decode relay response: %w", err)
	}

	return &processor.SendResponse{
		StatusCode: resp.StatusCode,
		Headers:    resp.Headers,
		Body:       resp.Body,
	}, nil
}

func (c *Client) CreateSessionToken(ctx context.Context, scope string) (*processor.SessionTokenResponse, error) {
	req := sessionTokenRequest{}
	req.SessionToken.Scope = scope

	data, _, err := c.do(ctx, http.MethodPost, "/api/payments/session_tokens", nil, req)
	if err != nil {
		return nil, err
	}

	var resp sessionTokenResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("pcibooking: decode session token response: %w", err)
	}

	return &processor.SessionTokenResponse{
		Token: resp.SessionToken.Token,
		Scope: resp.SessionToken.Scope,
	}, nil
}

func (c *Client) CaptureFormURL(sessionToken string) string {
	params := url.Values{}
	params.Set("session_token", sessionToken)
	return c.baseURL + "/api/payments/paycard/ui?" + params.Encode()
}
