package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/CentraGlobal/backend-payment-go/internal/types"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CardTokenService supports creating, fetching, and updating card token records
// backed by the payment_card_tokens table.
type CardTokenService interface {
	// Create inserts a new card token record and returns it with DB-assigned fields.
	Create(ctx context.Context, token *types.CardToken) (*types.CardToken, error)
	// GetByID returns a card token by its UUID primary key.
	GetByID(ctx context.Context, id string) (*types.CardToken, error)
	// GetByVaultToken returns a card token by gateway + vault token pair.
	GetByVaultToken(ctx context.Context, gatewayID, vaultToken string) (*types.CardToken, error)
	// UpdateStatus changes the lifecycle status of a token.
	UpdateStatus(ctx context.Context, id string, status types.CardTokenStatus) error
}

// PostgresCardTokenService implements CardTokenService against PostgreSQL.
type PostgresCardTokenService struct {
	db *pgxpool.Pool
}

// NewCardTokenService creates a PostgresCardTokenService. db may be nil; service
// methods will return an error in that case.
func NewCardTokenService(db *pgxpool.Pool) *PostgresCardTokenService {
	return &PostgresCardTokenService{db: db}
}

// Create inserts a new row into payment_card_tokens using gen_random_uuid() for
// the primary key, and returns the full record including DB-assigned fields.
func (s *PostgresCardTokenService) Create(ctx context.Context, token *types.CardToken) (*types.CardToken, error) {
	if s.db == nil {
		return nil, errors.New("card_token service: database not available")
	}

	const q = `
		INSERT INTO payment_card_tokens (
			id, org_id, hotel_id, reservation_id, person_id, customer_id,
			gateway_id, vault_provider, vault_token, token_scope, status,
			brand, last4, exp_month, exp_year, fingerprint_hash,
			expires_at, metadata,
			created_by, created_by_principal_type,
			updated_by, updated_by_principal_type
		) VALUES (
			gen_random_uuid(), $1, $2, $3, $4, $5,
			$6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15,
			$16, $17,
			$18, $19,
			$20, $21
		) RETURNING id, created_at, updated_at`

	scope := token.TokenScope
	if scope == "" {
		scope = types.TokenScopeSingleUse
	}
	status := token.Status
	if status == "" {
		status = types.TokenStatusActive
	}

	row := s.db.QueryRow(ctx, q,
		token.OrgID, token.HotelID, token.ReservationID, token.PersonID, token.CustomerID,
		token.GatewayID, token.VaultProvider, token.VaultToken, scope, status,
		token.Brand, token.Last4, token.ExpMonth, token.ExpYear, token.Fingerprint,
		token.ExpiresAt, nilIfEmpty(token.Metadata),
		token.CreatedBy, token.CreatedByPrincipalType,
		token.UpdatedBy, token.UpdatedByPrincipalType,
	)
	if err := row.Scan(&token.ID, &token.CreatedAt, &token.UpdatedAt); err != nil {
		return nil, fmt.Errorf("card_token service create: %w", err)
	}
	token.TokenScope = scope
	token.Status = status
	return token, nil
}

// GetByID returns a card token by its primary key.
func (s *PostgresCardTokenService) GetByID(ctx context.Context, id string) (*types.CardToken, error) {
	if s.db == nil {
		return nil, errors.New("card_token service: database not available")
	}
	const q = `
		SELECT id, org_id, hotel_id, reservation_id, person_id, customer_id,
		       gateway_id, vault_provider, vault_token, token_scope, status,
		       brand, last4, exp_month, exp_year, fingerprint_hash,
		       expires_at, consumed_at, revoked_at, metadata, created_at, updated_at
		FROM payment_card_tokens
		WHERE id = $1`
	return scanCardToken(s.db.QueryRow(ctx, q, id))
}

// GetByVaultToken returns a card token by gateway_id + vault_token (unique pair).
func (s *PostgresCardTokenService) GetByVaultToken(ctx context.Context, gatewayID, vaultToken string) (*types.CardToken, error) {
	if s.db == nil {
		return nil, errors.New("card_token service: database not available")
	}
	const q = `
		SELECT id, org_id, hotel_id, reservation_id, person_id, customer_id,
		       gateway_id, vault_provider, vault_token, token_scope, status,
		       brand, last4, exp_month, exp_year, fingerprint_hash,
		       expires_at, consumed_at, revoked_at, metadata, created_at, updated_at
		FROM payment_card_tokens
		WHERE gateway_id = $1 AND vault_token = $2
		LIMIT 1`
	return scanCardToken(s.db.QueryRow(ctx, q, gatewayID, vaultToken))
}

// UpdateStatus sets a new lifecycle status and bumps updated_at.
func (s *PostgresCardTokenService) UpdateStatus(ctx context.Context, id string, status types.CardTokenStatus) error {
	if s.db == nil {
		return errors.New("card_token service: database not available")
	}
	const q = `
		UPDATE payment_card_tokens
		SET status = $2, updated_at = now()
		WHERE id = $1`
	tag, err := s.db.Exec(ctx, q, id, status)
	if err != nil {
		return fmt.Errorf("card_token service update_status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("card_token service update_status: no row with id %s", id)
	}
	return nil
}

// scanCardToken decodes a single result row into a CardToken struct.
func scanCardToken(row pgx.Row) (*types.CardToken, error) {
	var ct types.CardToken
	var metadata []byte
	err := row.Scan(
		&ct.ID, &ct.OrgID, &ct.HotelID,
		&ct.ReservationID, &ct.PersonID, &ct.CustomerID,
		&ct.GatewayID, &ct.VaultProvider, &ct.VaultToken,
		&ct.TokenScope, &ct.Status,
		&ct.Brand, &ct.Last4, &ct.ExpMonth, &ct.ExpYear, &ct.Fingerprint,
		&ct.ExpiresAt, &ct.ConsumedAt, &ct.RevokedAt,
		&metadata, &ct.CreatedAt, &ct.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("card_token service: %w", err)
	}
	ct.Metadata = metadata
	return &ct, nil
}

// nilIfEmpty converts an empty byte slice to nil so that SQL receives NULL
// instead of an empty JSON value for optional JSONB columns.
func nilIfEmpty(b []byte) []byte {
	if len(b) == 0 {
		return nil
	}
	return b
}
