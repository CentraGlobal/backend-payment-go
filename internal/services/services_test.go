package services_test

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/CentraGlobal/backend-payment-go/internal/services"
	"github.com/CentraGlobal/backend-payment-go/internal/types"
)

// ---------------------------------------------------------------------------
// Interface compliance – ensure concrete types satisfy the interface.
// ---------------------------------------------------------------------------

var _ services.PaymentGatewayService = (*services.PostgresPaymentGatewayService)(nil)
var _ services.CardTokenService = (*services.PostgresCardTokenService)(nil)
var _ services.TransactionService = (*services.PostgresTransactionService)(nil)

// ---------------------------------------------------------------------------
// In-memory stub implementations used in handler / service-level tests.
// ---------------------------------------------------------------------------

// StubPaymentGatewayService is an in-memory PaymentGatewayService for tests.
type StubPaymentGatewayService struct {
	Gateways map[string]*types.PaymentGateway
}

func NewStubPaymentGatewayService() *StubPaymentGatewayService {
	return &StubPaymentGatewayService{Gateways: map[string]*types.PaymentGateway{}}
}

func (s *StubPaymentGatewayService) GetActiveDefault(_ context.Context, orgID, hotelID string) (*types.PaymentGateway, error) {
	for _, gw := range s.Gateways {
		if gw.OrgID == orgID && gw.HotelID == hotelID &&
			gw.Status == types.GatewayStatusActive && gw.IsDefault {
			return gw, nil
		}
	}
	return nil, nil
}

func (s *StubPaymentGatewayService) GetByID(_ context.Context, id string) (*types.PaymentGateway, error) {
	gw, ok := s.Gateways[id]
	if !ok {
		return nil, nil
	}
	return gw, nil
}

// StubCardTokenService is an in-memory CardTokenService for tests.
type StubCardTokenService struct {
	tokens map[string]*types.CardToken
	nextID int
}

func NewStubCardTokenService() *StubCardTokenService {
	return &StubCardTokenService{tokens: map[string]*types.CardToken{}}
}

func (s *StubCardTokenService) Create(_ context.Context, token *types.CardToken) (*types.CardToken, error) {
	s.nextID++
	token.ID = "ct_stub_" + strconv.Itoa(s.nextID)
	token.CreatedAt = time.Now()
	token.UpdatedAt = time.Now()
	s.tokens[token.ID] = token
	return token, nil
}

func (s *StubCardTokenService) GetByID(_ context.Context, id string) (*types.CardToken, error) {
	t, ok := s.tokens[id]
	if !ok {
		return nil, nil
	}
	return t, nil
}

func (s *StubCardTokenService) GetByVaultToken(_ context.Context, gatewayID, vaultToken string) (*types.CardToken, error) {
	for _, t := range s.tokens {
		if t.GatewayID == gatewayID && t.VaultToken == vaultToken {
			return t, nil
		}
	}
	return nil, nil
}

func (s *StubCardTokenService) UpdateStatus(_ context.Context, id string, status types.CardTokenStatus) error {
	if t, ok := s.tokens[id]; ok {
		t.Status = status
		t.UpdatedAt = time.Now()
	}
	return nil
}

// StubTransactionService is an in-memory TransactionService for tests.
type StubTransactionService struct {
	transactions map[string]*types.Transaction
	nextID       int
	LastUpdate   *types.TransactionStatusUpdate
}

func NewStubTransactionService() *StubTransactionService {
	return &StubTransactionService{transactions: map[string]*types.Transaction{}}
}

func (s *StubTransactionService) CreatePending(_ context.Context, tx *types.Transaction) (*types.Transaction, error) {
	s.nextID++
	tx.ID = "tx_stub_" + strconv.Itoa(s.nextID)
	tx.Status = types.TxStatusPending
	tx.CreatedAt = time.Now()
	s.transactions[tx.ID] = tx
	return tx, nil
}

func (s *StubTransactionService) UpdateStatus(_ context.Context, id string, u types.TransactionStatusUpdate) error {
	s.LastUpdate = &u
	if tx, ok := s.transactions[id]; ok {
		tx.Status = u.Status
	}
	return nil
}

func (s *StubTransactionService) GetByID(_ context.Context, id string) (*types.Transaction, error) {
	tx, ok := s.transactions[id]
	if !ok {
		return nil, nil
	}
	return tx, nil
}

// ---------------------------------------------------------------------------
// PaymentGatewayService tests
// ---------------------------------------------------------------------------

func TestStubPaymentGatewayService_GetActiveDefault(t *testing.T) {
	svc := NewStubPaymentGatewayService()
	svc.Gateways["gw_1"] = &types.PaymentGateway{
		ID:        "gw_1",
		OrgID:     "org_1",
		HotelID:   "hotel_1",
		Provider:  types.GatewayProviderStripe,
		Mode:      types.GatewayModeLive,
		Status:    types.GatewayStatusActive,
		IsDefault: true,
	}

	gw, err := svc.GetActiveDefault(context.Background(), "org_1", "hotel_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gw == nil {
		t.Fatal("expected a gateway, got nil")
	}
	if gw.ID != "gw_1" {
		t.Errorf("expected id gw_1, got %q", gw.ID)
	}
}

func TestStubPaymentGatewayService_GetActiveDefault_NotFound(t *testing.T) {
	svc := NewStubPaymentGatewayService()

	gw, err := svc.GetActiveDefault(context.Background(), "org_unknown", "hotel_unknown")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gw != nil {
		t.Error("expected nil for missing gateway")
	}
}

func TestStubPaymentGatewayService_GetByID(t *testing.T) {
	svc := NewStubPaymentGatewayService()
	svc.Gateways["gw_42"] = &types.PaymentGateway{ID: "gw_42", OrgID: "org_1", HotelID: "hotel_1"}

	gw, err := svc.GetByID(context.Background(), "gw_42")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gw == nil || gw.ID != "gw_42" {
		t.Errorf("expected gw_42, got %v", gw)
	}
}

// ---------------------------------------------------------------------------
// CardTokenService tests
// ---------------------------------------------------------------------------

func TestStubCardTokenService_CreateAndGet(t *testing.T) {
	svc := NewStubCardTokenService()
	token := &types.CardToken{
		OrgID:         "org_1",
		HotelID:       "hotel_1",
		GatewayID:     "gw_1",
		VaultProvider: "vaultera",
		VaultToken:    "tok_xyz",
		TokenScope:    types.TokenScopeSingleUse,
		Status:        types.TokenStatusActive,
	}

	created, err := svc.Create(context.Background(), token)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.ID == "" {
		t.Fatal("expected non-empty ID after create")
	}

	fetched, err := svc.GetByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if fetched == nil || fetched.VaultToken != "tok_xyz" {
		t.Errorf("expected vault_token tok_xyz, got %v", fetched)
	}
}

func TestStubCardTokenService_GetByVaultToken(t *testing.T) {
	svc := NewStubCardTokenService()
	svc.Create(context.Background(), &types.CardToken{
		OrgID: "org_1", HotelID: "hotel_1", GatewayID: "gw_1",
		VaultProvider: "vaultera", VaultToken: "tok_abc",
		TokenScope: types.TokenScopeSingleUse, Status: types.TokenStatusActive,
	})

	found, err := svc.GetByVaultToken(context.Background(), "gw_1", "tok_abc")
	if err != nil {
		t.Fatalf("GetByVaultToken: %v", err)
	}
	if found == nil {
		t.Fatal("expected token, got nil")
	}
	if found.VaultToken != "tok_abc" {
		t.Errorf("expected tok_abc, got %q", found.VaultToken)
	}
}

func TestStubCardTokenService_UpdateStatus(t *testing.T) {
	svc := NewStubCardTokenService()
	token, _ := svc.Create(context.Background(), &types.CardToken{
		OrgID: "org_1", HotelID: "hotel_1", GatewayID: "gw_1",
		VaultProvider: "vaultera", VaultToken: "tok_def",
		TokenScope: types.TokenScopeSingleUse, Status: types.TokenStatusActive,
	})

	if err := svc.UpdateStatus(context.Background(), token.ID, types.TokenStatusConsumed); err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}

	fetched, _ := svc.GetByID(context.Background(), token.ID)
	if fetched.Status != types.TokenStatusConsumed {
		t.Errorf("expected consumed, got %q", fetched.Status)
	}
}

// ---------------------------------------------------------------------------
// TransactionService tests
// ---------------------------------------------------------------------------

func TestStubTransactionService_CreatePending(t *testing.T) {
	svc := NewStubTransactionService()
	tx := &types.Transaction{
		OrgID:     "org_1",
		HotelID:   "hotel_1",
		GatewayID: "gw_1",
		Operation: types.TxOpCharge,
		Amount:    5000,
		Currency:  "USD",
	}

	created, err := svc.CreatePending(context.Background(), tx)
	if err != nil {
		t.Fatalf("CreatePending: %v", err)
	}
	if created.ID == "" {
		t.Fatal("expected non-empty ID")
	}
	if created.Status != types.TxStatusPending {
		t.Errorf("expected pending, got %q", created.Status)
	}
}

func TestStubTransactionService_UpdateStatus_Succeeded(t *testing.T) {
	svc := NewStubTransactionService()
	tx, _ := svc.CreatePending(context.Background(), &types.Transaction{
		OrgID: "org_1", HotelID: "hotel_1", GatewayID: "gw_1",
		Operation: types.TxOpCharge, Amount: 1000, Currency: "EUR",
	})

	providerID := "ch_provider_001"
	update := types.TransactionStatusUpdate{
		Status:                types.TxStatusSucceeded,
		ProviderTransactionID: &providerID,
	}
	if err := svc.UpdateStatus(context.Background(), tx.ID, update); err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}

	fetched, _ := svc.GetByID(context.Background(), tx.ID)
	if fetched.Status != types.TxStatusSucceeded {
		t.Errorf("expected succeeded, got %q", fetched.Status)
	}
}

func TestStubTransactionService_UpdateStatus_Failed(t *testing.T) {
	svc := NewStubTransactionService()
	tx, _ := svc.CreatePending(context.Background(), &types.Transaction{
		OrgID: "org_1", HotelID: "hotel_1", GatewayID: "gw_1",
		Operation: types.TxOpCharge, Amount: 2000, Currency: "GBP",
	})

	msg := "card declined"
	update := types.TransactionStatusUpdate{
		Status:         types.TxStatusFailed,
		FailureMessage: &msg,
	}
	if err := svc.UpdateStatus(context.Background(), tx.ID, update); err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}

	if svc.LastUpdate.Status != types.TxStatusFailed {
		t.Errorf("expected failed, got %q", svc.LastUpdate.Status)
	}
	if *svc.LastUpdate.FailureMessage != "card declined" {
		t.Errorf("expected 'card declined', got %q", *svc.LastUpdate.FailureMessage)
	}
}

// ---------------------------------------------------------------------------
// PostgresXxx nil-pool error handling
// ---------------------------------------------------------------------------

func TestPostgresPaymentGatewayService_NilPool(t *testing.T) {
	svc := services.NewPaymentGatewayService(nil)

	if _, err := svc.GetActiveDefault(context.Background(), "o", "h"); err == nil {
		t.Error("expected error with nil pool")
	}
	if _, err := svc.GetByID(context.Background(), "id"); err == nil {
		t.Error("expected error with nil pool")
	}
}

func TestPostgresCardTokenService_NilPool(t *testing.T) {
	svc := services.NewCardTokenService(nil)

	if _, err := svc.Create(context.Background(), &types.CardToken{}); err == nil {
		t.Error("expected error with nil pool")
	}
	if _, err := svc.GetByID(context.Background(), "id"); err == nil {
		t.Error("expected error with nil pool")
	}
	if _, err := svc.GetByVaultToken(context.Background(), "gw", "tok"); err == nil {
		t.Error("expected error with nil pool")
	}
	if err := svc.UpdateStatus(context.Background(), "id", types.TokenStatusRevoked); err == nil {
		t.Error("expected error with nil pool")
	}
}

func TestPostgresTransactionService_NilPool(t *testing.T) {
	svc := services.NewTransactionService(nil)

	if _, err := svc.CreatePending(context.Background(), &types.Transaction{}); err == nil {
		t.Error("expected error with nil pool")
	}
	if err := svc.UpdateStatus(context.Background(), "id", types.TransactionStatusUpdate{}); err == nil {
		t.Error("expected error with nil pool")
	}
	if _, err := svc.GetByID(context.Background(), "id"); err == nil {
		t.Error("expected error with nil pool")
	}
}
