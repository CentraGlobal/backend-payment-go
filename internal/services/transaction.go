package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/CentraGlobal/backend-payment-go/internal/types"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TransactionService supports creating pending transactions, updating their
// status after provider calls, and fetching them by shared identifiers.
type TransactionService interface {
	// CreatePending inserts a new transaction with status "pending" and returns
	// it with DB-assigned fields.
	CreatePending(ctx context.Context, tx *types.Transaction) (*types.Transaction, error)
	// UpdateStatus sets the final status and provider metadata on a transaction.
	UpdateStatus(ctx context.Context, id string, update types.TransactionStatusUpdate) error
	// GetByID returns a transaction by its UUID primary key.
	GetByID(ctx context.Context, id string) (*types.Transaction, error)
}

// PostgresTransactionService implements TransactionService against PostgreSQL.
type PostgresTransactionService struct {
	db *pgxpool.Pool
}

// NewTransactionService creates a PostgresTransactionService. db may be nil;
// service methods will return an error in that case.
func NewTransactionService(db *pgxpool.Pool) *PostgresTransactionService {
	return &PostgresTransactionService{db: db}
}

// CreatePending inserts a new pending transaction record and returns the full
// row including the DB-generated UUID and created_at timestamp.
func (s *PostgresTransactionService) CreatePending(ctx context.Context, tx *types.Transaction) (*types.Transaction, error) {
	if s.db == nil {
		return nil, errors.New("transaction service: database not available")
	}

	op := tx.Operation
	if op == "" {
		op = types.TxOpCharge
	}

	const q = `
		INSERT INTO payment_transactions (
			id, org_id, hotel_id, reservation_id, person_id, customer_id,
			gateway_id, card_token_id, scheduled_transaction_id,
			operation, status, amount, currency,
			parent_transaction_id, idempotency_key,
			request_payload, metadata,
			created_by, created_by_principal_type
		) VALUES (
			gen_random_uuid(), $1, $2, $3, $4, $5,
			$6, $7, $8,
			$9, 'pending', $10, $11,
			$12, $13,
			$14, $15,
			$16, $17
		) RETURNING id, status, created_at`

	row := s.db.QueryRow(ctx, q,
		tx.OrgID, tx.HotelID, tx.ReservationID, tx.PersonID, tx.CustomerID,
		tx.GatewayID, tx.CardTokenID, tx.ScheduledTxID,
		op, tx.Amount, tx.Currency,
		tx.ParentTxID, tx.IdempotencyKey,
		nilIfEmpty(tx.RequestPayload), nilIfEmpty(tx.Metadata),
		tx.CreatedBy, tx.CreatedByPrincipalType,
	)
	if err := row.Scan(&tx.ID, &tx.Status, &tx.CreatedAt); err != nil {
		return nil, fmt.Errorf("transaction service create: %w", err)
	}
	tx.Operation = op
	return tx, nil
}

// UpdateStatus applies a resolved status and any provider fields to an existing
// transaction. It also stamps processed_at or failed_at based on the outcome.
func (s *PostgresTransactionService) UpdateStatus(ctx context.Context, id string, u types.TransactionStatusUpdate) error {
	if s.db == nil {
		return errors.New("transaction service: database not available")
	}

	now := time.Now().UTC()
	var processedAt, failedAt *time.Time
	switch u.Status {
	case types.TxStatusSucceeded:
		processedAt = &now
	case types.TxStatusFailed:
		failedAt = &now
	}

	const q = `
		UPDATE payment_transactions
		SET status                  = $2,
		    provider_transaction_id = $3,
		    provider_charge_id      = $4,
		    vaultera_request_id     = $5,
		    response_payload        = $6,
		    failure_code            = $7,
		    failure_message         = $8,
		    processed_at            = $9,
		    failed_at               = $10
		WHERE id = $1`

	tag, err := s.db.Exec(ctx, q,
		id, u.Status,
		u.ProviderTransactionID, u.ProviderChargeID, u.VaulteraRequestID,
		nilIfEmpty(u.ResponsePayload),
		u.FailureCode, u.FailureMessage,
		processedAt, failedAt,
	)
	if err != nil {
		return fmt.Errorf("transaction service update_status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("transaction service update_status: no row with id %s", id)
	}
	return nil
}

// GetByID returns a transaction by its UUID primary key.
func (s *PostgresTransactionService) GetByID(ctx context.Context, id string) (*types.Transaction, error) {
	if s.db == nil {
		return nil, errors.New("transaction service: database not available")
	}
	const q = `
		SELECT id, org_id, hotel_id, reservation_id, person_id, customer_id,
		       gateway_id, card_token_id, scheduled_transaction_id,
		       operation, status, amount, currency,
		       parent_transaction_id, idempotency_key,
		       provider_transaction_id, provider_charge_id, vaultera_request_id,
		       request_payload, response_payload,
		       failure_code, failure_message,
		       processed_at, failed_at, metadata, created_at
		FROM payment_transactions
		WHERE id = $1`
	return scanTransaction(s.db.QueryRow(ctx, q, id))
}

// scanTransaction decodes a single result row into a Transaction struct.
func scanTransaction(row pgx.Row) (*types.Transaction, error) {
	var tx types.Transaction
	var requestPayload, responsePayload, metadata []byte
	err := row.Scan(
		&tx.ID, &tx.OrgID, &tx.HotelID,
		&tx.ReservationID, &tx.PersonID, &tx.CustomerID,
		&tx.GatewayID, &tx.CardTokenID, &tx.ScheduledTxID,
		&tx.Operation, &tx.Status, &tx.Amount, &tx.Currency,
		&tx.ParentTxID, &tx.IdempotencyKey,
		&tx.ProviderTransactionID, &tx.ProviderChargeID, &tx.VaulteraRequestID,
		&requestPayload, &responsePayload,
		&tx.FailureCode, &tx.FailureMessage,
		&tx.ProcessedAt, &tx.FailedAt, &metadata, &tx.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("transaction service: %w", err)
	}
	tx.RequestPayload = requestPayload
	tx.ResponsePayload = responsePayload
	tx.Metadata = metadata
	return &tx, nil
}
