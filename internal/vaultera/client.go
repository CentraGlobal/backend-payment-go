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
	return "vaultera"
}

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

type cardRequest struct {
	CardNumber      string `json:"card_number"`
	CardType        string `json:"card_type,omitempty"`
	CardholderName  string `json:"cardholder_name,omitempty"`
	ServiceCode     string `json:"service_code,omitempty"`
	ExpirationMonth string `json:"expiration_month"`
	ExpirationYear  string `json:"expiration_year"`
}

type cardResponseWrapper struct {
	Data struct {
		Type       string `json:"type"`
		ID         string `json:"id"`
		Attributes struct {
			CardNumber      string `json:"card_number"`
			CardToken       string `json:"card_token"`
			CardType        string `json:"card_type"`
			CardholderName  string `json:"cardholder_name"`
			ExpirationMonth string `json:"expiration_month"`
			ExpirationYear  string `json:"expiration_year"`
			ServiceCode     string `json:"service_code"`
		} `json:"attributes"`
	} `json:"data"`
}

type cardResponse struct {
	CardToken       string `json:"card_token"`
	CardNumberMask  string `json:"card_number_mask"`
	CardType        string `json:"card_type"`
	CardholderName  string `json:"cardholder_name"`
	ExpirationMonth string `json:"expiration_month"`
	ExpirationYear  string `json:"expiration_year"`
}

type sendRequest struct {
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    string            `json:"body,omitempty"`
}

type sessionTokenRequest struct {
	SessionToken struct {
		Scope string `json:"scope"`
	} `json:"session_token"`
}

type sessionTokenResponseWrapper struct {
	Data struct {
		Type       string `json:"type"`
		ID         string `json:"id"`
		Attributes struct {
			SessionToken string `json:"session_token"`
			Scope        string `json:"scope"`
		} `json:"attributes"`
	} `json:"data"`
}

func (c *Client) CreateCard(ctx context.Context, card processor.Card) (*processor.CardResponse, error) {
	payload := map[string]any{
		"card": cardRequest{
			CardNumber:      card.CardNumber,
			CardType:        card.CardType,
			CardholderName:  card.CardholderName,
			ServiceCode:     card.ServiceCode,
			ExpirationMonth: card.ExpirationMonth,
			ExpirationYear:  card.ExpirationYear,
		},
	}
	data, _, err := c.do(ctx, http.MethodPost, "/cards", nil, payload)
	if err != nil {
		return nil, err
	}
	var wrapper cardResponseWrapper
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return nil, fmt.Errorf("vaultera: decode card response: %w", err)
	}
	return &processor.CardResponse{
		CardToken:       wrapper.Data.Attributes.CardToken,
		CardMask:        wrapper.Data.Attributes.CardNumber,
		CardType:        wrapper.Data.Attributes.CardType,
		CardholderName:  wrapper.Data.Attributes.CardholderName,
		ExpirationMonth: wrapper.Data.Attributes.ExpirationMonth,
		ExpirationYear:  wrapper.Data.Attributes.ExpirationYear,
	}, nil
}

func (c *Client) GetCard(ctx context.Context, cardToken string) (*processor.CardResponse, error) {
	data, _, err := c.do(ctx, http.MethodGet, "/cards/"+cardToken, nil, nil)
	if err != nil {
		return nil, err
	}
	var wrapper cardResponseWrapper
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return nil, fmt.Errorf("vaultera: decode card response: %w", err)
	}
	return &processor.CardResponse{
		CardToken:       wrapper.Data.Attributes.CardToken,
		CardMask:        wrapper.Data.Attributes.CardNumber,
		CardType:        wrapper.Data.Attributes.CardType,
		CardholderName:  wrapper.Data.Attributes.CardholderName,
		ExpirationMonth: wrapper.Data.Attributes.ExpirationMonth,
		ExpirationYear:  wrapper.Data.Attributes.ExpirationYear,
	}, nil
}

func (c *Client) DeleteCard(ctx context.Context, cardToken string) error {
	_, _, err := c.do(ctx, http.MethodDelete, "/cards/"+cardToken, nil, nil)
	return err
}

func (c *Client) SendCard(ctx context.Context, cardToken string, req processor.SendRequest) (*processor.SendResponse, error) {
	vreq := sendRequest{
		Method:  req.Method,
		URL:     req.URL,
		Headers: req.Headers,
		Body:    req.Body,
	}
	data, _, err := c.do(ctx, http.MethodPost, "/cards/"+cardToken+"/send", nil, vreq)
	if err != nil {
		return nil, err
	}
	var resp struct {
		StatusCode int               `json:"status_code"`
		Headers    map[string]string `json:"headers,omitempty"`
		Body       json.RawMessage   `json:"body"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("vaultera: decode send response: %w", err)
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

	data, _, err := c.do(ctx, http.MethodPost, "/session_tokens", nil, req)
	if err != nil {
		return nil, err
	}
	var wrapper sessionTokenResponseWrapper
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return nil, fmt.Errorf("vaultera: decode session_token response: %w", err)
	}
	return &processor.SessionTokenResponse{
		Token: wrapper.Data.Attributes.SessionToken,
		Scope: wrapper.Data.Attributes.Scope,
	}, nil
}

func (c *Client) CaptureFormURL(sessionToken string) string {
	params := url.Values{}
	params.Set("session_token", sessionToken)
	return c.baseURL + "/capture_form?" + params.Encode()
}
