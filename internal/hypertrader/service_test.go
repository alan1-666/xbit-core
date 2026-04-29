package hypertrader

import (
	"context"
	"errors"
	"testing"
)

func TestServiceContractScope(t *testing.T) {
	service := NewService(nil)
	ctx := context.Background()

	symbols, err := service.ListSymbols(ctx, "", "", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(symbols) < 3 {
		t.Fatalf("expected seeded futures symbols, got %d", len(symbols))
	}
	if symbols[0].Type != "PERP" {
		t.Fatalf("expected perp symbol type, got %q", symbols[0].Type)
	}

	account, err := service.Account(ctx, "0xuser")
	if err != nil {
		t.Fatal(err)
	}
	if account.Balance == "" || len(account.Positions) == 0 {
		t.Fatalf("expected account positions, got %+v", account)
	}

	group, err := service.CreateGroup(ctx, "VIP", "user-1", false)
	if err != nil {
		t.Fatal(err)
	}
	address, err := service.CreateAddress(ctx, "0xabc", "Alpha", []string{group.ID}, "user-1")
	if err != nil {
		t.Fatal(err)
	}
	updated, err := service.UpdateAddress(ctx, address.ID, "Alpha Updated", []string{group.ID})
	if err != nil {
		t.Fatal(err)
	}
	if updated.RemarkName != "Alpha Updated" {
		t.Fatalf("unexpected updated address: %+v", updated)
	}
}

func TestServiceOrderLifecycle(t *testing.T) {
	service := NewService(nil)
	ctx := context.Background()

	order, err := service.CreateOrder(ctx, CreateOrderInput{
		UserID:          "user-1",
		UserAddress:     "0xuser",
		Symbol:          "btc",
		Side:            "buy",
		OrderType:       "market",
		Size:            "0.1",
		ClientRequestID: "req-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if order.Status != "submitted" || order.ProviderOrderID == "" {
		t.Fatalf("unexpected submitted order: %+v", order)
	}

	duplicate, err := service.CreateOrder(ctx, CreateOrderInput{
		UserID:          "user-1",
		UserAddress:     "0xuser",
		Symbol:          "BTC",
		Side:            "buy",
		OrderType:       "market",
		Size:            "0.1",
		ClientRequestID: "req-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if duplicate.ID != order.ID {
		t.Fatalf("idempotent order mismatch: got %s want %s", duplicate.ID, order.ID)
	}

	synced, err := service.SyncOrderStatus(ctx, OrderStatusInput{OrderID: order.ID})
	if err != nil {
		t.Fatal(err)
	}
	if synced.Status != "submitted" {
		t.Fatalf("unexpected synced order: %+v", synced)
	}

	cancelled, err := service.CancelOrder(ctx, CancelOrderInput{OrderID: order.ID, UserID: "user-1"})
	if err != nil {
		t.Fatal(err)
	}
	if cancelled.Status != "cancelled" || cancelled.CancelledAt == nil {
		t.Fatalf("unexpected cancelled order: %+v", cancelled)
	}

	resynced, err := service.SyncOrderStatus(ctx, OrderStatusInput{OrderID: order.ID})
	if err != nil {
		t.Fatal(err)
	}
	if resynced.Status != "cancelled" {
		t.Fatalf("terminal status should not be downgraded: %+v", resynced)
	}

	if _, err := service.UpdateLeverage(ctx, UpdateLeverageInput{UserID: "user-1", Symbol: "BTC", Leverage: 10, IsCross: true}); err != nil {
		t.Fatal(err)
	}
	events, err := service.AuditEvents(ctx, "user-1", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) < 5 {
		t.Fatalf("expected audit events, got %+v", events)
	}
}

func TestServiceOpenOrdersUsesEmptySnapshotFallback(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	if err := store.SaveOpenOrdersSnapshot(ctx, "0xempty", nil); err != nil {
		t.Fatal(err)
	}
	service := NewServiceWithProvider(store, failingOpenOrdersProvider{LocalProvider: NewLocalProvider()})

	orders, err := service.OpenOrders(ctx, "0xempty")
	if err != nil {
		t.Fatal(err)
	}
	if len(orders) != 0 {
		t.Fatalf("expected empty snapshot fallback, got %+v", orders)
	}
}

type failingOpenOrdersProvider struct {
	*LocalProvider
}

func (p failingOpenOrdersProvider) OpenOrders(context.Context, string) ([]OpenOrder, error) {
	return nil, errors.New("provider unavailable")
}

func TestServiceAgentSignerLifecycle(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	signer := NewAgentSigner(store, AgentSignerConfig{
		Enabled: true,
		DefaultPolicy: AgentPolicy{
			AllowedActions: []string{"order", "cancel", "updateLeverage"},
			AllowedSymbols: []string{"BTC"},
			MaxLeverage:    5,
		},
	})
	service := NewServiceWithProviderAndSigner(store, NewLocalProvider(), signer)

	approval, err := service.CreateAgentWallet(ctx, CreateAgentWalletInput{UserID: "user-1", UserAddress: "0xabc", AgentName: "Desk Agent"})
	if err != nil {
		t.Fatal(err)
	}
	if approval.Wallet.Status != "pending_approval" || approval.ApprovalPayload["action"] == nil {
		t.Fatalf("unexpected approval: %+v", approval)
	}
	if _, err := service.AgentSign(ctx, AgentSignInput{UserID: "user-1", UserAddress: "0xabc", Action: "order", Symbol: "BTC", ExchangeAction: map[string]any{"type": "order", "orders": []any{}}}); err == nil {
		t.Fatal("expected inactive agent wallet to be rejected")
	}
	wallet, err := service.ActivateAgentWallet(ctx, ActivateAgentWalletInput{UserAddress: "0xabc", AgentAddress: approval.Wallet.AgentAddress})
	if err != nil {
		t.Fatal(err)
	}
	if wallet.Status != "active" || wallet.ApprovedAt == nil {
		t.Fatalf("unexpected active wallet: %+v", wallet)
	}
	signed, err := service.AgentSign(ctx, AgentSignInput{UserID: "user-1", UserAddress: "0xabc", Action: "order", Symbol: "BTC", ExchangeAction: map[string]any{"type": "order", "orders": []any{}}})
	if err != nil {
		t.Fatal(err)
	}
	if signed.ExchangePayload["nonce"] == nil || signed.Signature.R == "" {
		t.Fatalf("unexpected signed payload: %+v", signed)
	}
	if _, err := service.AgentSign(ctx, AgentSignInput{UserID: "user-1", UserAddress: "0xabc", Action: "updateLeverage", Symbol: "BTC", Leverage: 10, ExchangeAction: map[string]any{"type": "updateLeverage"}}); err == nil {
		t.Fatal("expected leverage policy rejection")
	}
	order, err := service.CreateOrder(ctx, CreateOrderInput{
		UserID:         "user-1",
		UserAddress:    "0xabc",
		Symbol:         "BTC",
		Side:           "buy",
		OrderType:      "limit",
		Price:          "95000",
		Size:           "0.1",
		ExchangeAction: map[string]any{"type": "order", "orders": []any{}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if order.ResponsePayload["agentSigner"] == nil || order.RawPayload["exchangePayload"] == nil {
		t.Fatalf("expected managed signer payload on order: %+v", order)
	}
}
