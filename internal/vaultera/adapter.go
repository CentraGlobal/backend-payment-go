package vaultera

import (
	"context"

	"github.com/CentraGlobal/backend-payment-go/internal/processor"
)

var _ processor.Processor = (*Adapter)(nil)

type Adapter struct {
	client *Client
}

func NewAdapter(apiKey, baseURL string) *Adapter {
	return &Adapter{
		client: NewClient(apiKey, baseURL),
	}
}

func (a *Adapter) Name() string {
	return "vaultera"
}

func (a *Adapter) CreateCard(ctx context.Context, card processor.Card) (*processor.CardResponse, error) {
	vcard := Card{
		CardNumber:      card.CardNumber,
		CardType:        card.CardType,
		CardholderName:  card.CardholderName,
		ServiceCode:     card.ServiceCode,
		ExpirationMonth: card.ExpirationMonth,
		ExpirationYear:  card.ExpirationYear,
	}
	resp, err := a.client.CreateCard(ctx, vcard)
	if err != nil {
		return nil, err
	}
	return &processor.CardResponse{
		CardToken:       resp.CardToken,
		CardMask:        resp.CardNumberMask,
		CardType:        resp.CardType,
		CardholderName:  resp.CardholderName,
		ExpirationMonth: resp.ExpirationMonth,
		ExpirationYear:  resp.ExpirationYear,
	}, nil
}

func (a *Adapter) GetCard(ctx context.Context, cardToken string) (*processor.CardResponse, error) {
	resp, err := a.client.GetCard(ctx, cardToken)
	if err != nil {
		return nil, err
	}
	return &processor.CardResponse{
		CardToken:       resp.CardToken,
		CardMask:        resp.CardNumberMask,
		CardType:        resp.CardType,
		CardholderName:  resp.CardholderName,
		ExpirationMonth: resp.ExpirationMonth,
		ExpirationYear:  resp.ExpirationYear,
	}, nil
}

func (a *Adapter) DeleteCard(ctx context.Context, cardToken string) error {
	return a.client.DeleteCard(ctx, cardToken)
}

func (a *Adapter) SendCard(ctx context.Context, cardToken string, req processor.SendRequest) (*processor.SendResponse, error) {
	vreq := SendRequest{
		Method:  req.Method,
		URL:     req.URL,
		Headers: req.Headers,
		Body:    req.Body,
	}
	resp, err := a.client.SendCard(ctx, cardToken, vreq)
	if err != nil {
		return nil, err
	}
	return &processor.SendResponse{
		StatusCode: resp.StatusCode,
		Headers:    resp.Headers,
		Body:       resp.Body,
	}, nil
}

func (a *Adapter) CreateSessionToken(ctx context.Context, scope string) (*processor.SessionTokenResponse, error) {
	resp, err := a.client.CreateSessionToken(ctx, scope)
	if err != nil {
		return nil, err
	}
	return &processor.SessionTokenResponse{
		Token: resp.Token,
		Scope: resp.Scope,
	}, nil
}

func (a *Adapter) CaptureFormURL(sessionToken string) string {
	return a.client.CaptureFormURL(sessionToken)
}
