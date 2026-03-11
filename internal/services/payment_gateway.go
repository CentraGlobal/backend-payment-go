// Package services provides domain service interfaces and their PostgreSQL
// implementations for shared payment entities.
package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/CentraGlobal/backend-payment-go/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PaymentGatewayService supports reading active/default gateway configuration
// for a hotel from the shared payment_gateways table.
type PaymentGatewayService interface {
	// GetActiveDefault returns the active default gateway for the given org/hotel.
	GetActiveDefault(ctx context.Context, orgID, hotelID string) (*domain.PaymentGateway, error)
	// GetByID returns a single gateway by its primary key.
	GetByID(ctx context.Context, id string) (*domain.PaymentGateway, error)
}

// PostgresPaymentGatewayService implements PaymentGatewayService against
// a live PostgreSQL connection pool.
type PostgresPaymentGatewayService struct {
	db *pgxpool.Pool
}

// NewPaymentGatewayService creates a PostgresPaymentGatewayService. db may be
// nil; service methods will return an error in that case.
func NewPaymentGatewayService(db *pgxpool.Pool) *PostgresPaymentGatewayService {
	return &PostgresPaymentGatewayService{db: db}
}

// GetActiveDefault returns the single active+default gateway for the hotel.
func (s *PostgresPaymentGatewayService) GetActiveDefault(ctx context.Context, orgID, hotelID string) (*domain.PaymentGateway, error) {
	if s.db == nil {
		return nil, errors.New("payment_gateway service: database not available")
	}
	const q = `
		SELECT id, org_id, hotel_id, provider, mode, status, is_default,
		       public_config, secret_refs, routing_config, webhook_config,
		       capabilities, metadata, created_at, updated_at
		FROM payment_gateways
		WHERE org_id = $1 AND hotel_id = $2 AND status = 'active' AND is_default = true
		LIMIT 1`
	return scanGateway(s.db.QueryRow(ctx, q, orgID, hotelID))
}

// GetByID returns a gateway row by its UUID primary key.
func (s *PostgresPaymentGatewayService) GetByID(ctx context.Context, id string) (*domain.PaymentGateway, error) {
	if s.db == nil {
		return nil, errors.New("payment_gateway service: database not available")
	}
	const q = `
		SELECT id, org_id, hotel_id, provider, mode, status, is_default,
		       public_config, secret_refs, routing_config, webhook_config,
		       capabilities, metadata, created_at, updated_at
		FROM payment_gateways
		WHERE id = $1`
	return scanGateway(s.db.QueryRow(ctx, q, id))
}

// scanGateway decodes a single row into a PaymentGateway struct.
func scanGateway(row pgx.Row) (*domain.PaymentGateway, error) {
	var gw domain.PaymentGateway
	// Nullable JSONB columns are scanned as *[]byte so that SQL NULL becomes nil.
	var (
		publicConfig  []byte
		secretRefs    []byte
		routingConfig []byte
		webhookConfig []byte
		capabilities  []byte
		metadata      []byte
	)
	err := row.Scan(
		&gw.ID, &gw.OrgID, &gw.HotelID,
		&gw.Provider, &gw.Mode, &gw.Status, &gw.IsDefault,
		&publicConfig, &secretRefs, &routingConfig,
		&webhookConfig, &capabilities, &metadata,
		&gw.CreatedAt, &gw.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("payment_gateway service: %w", err)
	}
	gw.PublicConfig = publicConfig
	gw.SecretRefs = secretRefs
	gw.RoutingConfig = routingConfig
	gw.WebhookConfig = webhookConfig
	gw.Capabilities = capabilities
	gw.Metadata = metadata
	return &gw, nil
}
