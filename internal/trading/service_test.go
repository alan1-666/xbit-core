package trading

import (
	"context"
	"testing"
	"time"
)

func TestQuoteCalculatesFeesAndSlippage(t *testing.T) {
	svc := NewService(nil)
	svc.now = func() time.Time { return time.Date(2026, 4, 28, 12, 0, 0, 0, time.UTC) }

	quote, err := svc.Quote(context.Background(), QuoteRequest{
		UserID:      "user-1",
		ChainType:   "SOLANA",
		InputToken:  "SOL",
		OutputToken: "USDC",
		InputAmount: "100",
		SlippageBps: 100,
	})
	if err != nil {
		t.Fatal(err)
	}
	if quote.ID == "" || quote.RouteID == "" {
		t.Fatalf("quote identifiers missing: %+v", quote)
	}
	if quote.PlatformFeeAmount != "0.3" {
		t.Fatalf("platform fee = %s", quote.PlatformFeeAmount)
	}
	if quote.OutputAmount != "99.7" {
		t.Fatalf("output amount = %s", quote.OutputAmount)
	}
	if quote.MinOutputAmount != "98.703" {
		t.Fatalf("min output amount = %s", quote.MinOutputAmount)
	}
	if !quote.ExpiresAt.Equal(quote.CreatedAt.Add(30 * time.Second)) {
		t.Fatalf("expires at = %s created at = %s", quote.ExpiresAt, quote.CreatedAt)
	}
}

func TestCreateOrderIsIdempotentByClientRequestID(t *testing.T) {
	svc := NewService(nil)
	ctx := context.Background()

	input := CreateOrderInput{
		UserID:          "user-1",
		ChainType:       "EVM",
		WalletAddress:   "0xabc",
		OrderType:       OrderTypeMarket,
		Side:            SideBuy,
		InputToken:      "ETH",
		OutputToken:     "USDC",
		InputAmount:     "1.5",
		ClientRequestID: "req-1",
	}
	first, err := svc.CreateOrder(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	second, err := svc.CreateOrder(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != second.ID {
		t.Fatalf("idempotent order IDs differ: %s vs %s", first.ID, second.ID)
	}
	if first.Status != OrderStatusPending {
		t.Fatalf("status = %s", first.Status)
	}
	if first.ExpectedOutputAmount != "1.4955" {
		t.Fatalf("expected output = %s", first.ExpectedOutputAmount)
	}

	orders, err := svc.ListOrders(ctx, SearchOrdersInput{UserID: "user-1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(orders) != 1 {
		t.Fatalf("orders = %+v", orders)
	}
}

func TestUpdateAndCancelOrder(t *testing.T) {
	svc := NewService(nil)
	ctx := context.Background()

	order, err := svc.CreateOrder(ctx, CreateOrderInput{
		UserID:        "user-1",
		ChainType:     "SOLANA",
		WalletAddress: "sol-wallet",
		OrderType:     OrderTypeMarket,
		Side:          SideSell,
		InputToken:    "SOL",
		OutputToken:   "USDC",
		InputAmount:   "2",
	})
	if err != nil {
		t.Fatal(err)
	}

	confirmed, err := svc.UpdateOrderStatus(ctx, order.ID, UpdateOrderStatusInput{
		Status: OrderStatusConfirmed,
		TxHash: "0xtx",
	})
	if err != nil {
		t.Fatal(err)
	}
	if confirmed.Status != OrderStatusConfirmed || confirmed.TxHash != "0xtx" || confirmed.FilledAt == nil {
		t.Fatalf("confirmed order = %+v", confirmed)
	}

	cancelled, err := svc.CancelOrder(ctx, order.ID)
	if err != nil {
		t.Fatal(err)
	}
	if cancelled.Status != OrderStatusCancelled {
		t.Fatalf("cancelled status = %s", cancelled.Status)
	}
}

func TestGetNetworkFeeFallsBackToDefault(t *testing.T) {
	svc := NewService(nil)

	fee, err := svc.GetNetworkFee(context.Background(), "solana")
	if err != nil {
		t.Fatal(err)
	}
	if fee.PlatformFeeBps != 30 || fee.MaxComputeUnits == 0 || fee.Source == "" {
		t.Fatalf("fee = %+v", fee)
	}
}
