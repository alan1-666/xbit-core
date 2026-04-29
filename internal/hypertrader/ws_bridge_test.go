package hypertrader

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/xbit/xbit-backend/internal/streambridge"
)

func TestHyperliquidStreamBridgePublishesOrderAndFillEvents(t *testing.T) {
	publisher := streambridge.NewMemoryPublisher()
	streams := streambridge.NewService(publisher)
	bridge := NewHyperliquidStreamBridge(StreamBridgeConfig{Users: []string{"0xabc"}}, streams)
	ctx := context.Background()

	if err := bridge.handleMessage(ctx, "0xabc", map[string]any{
		"channel": "orderUpdates",
		"data": []any{map[string]any{
			"status":          "open",
			"statusTimestamp": float64(1710000000000),
			"order": map[string]any{
				"coin": "BTC", "side": "B", "limitPx": "95000", "sz": "0.1", "origSz": "0.1", "oid": float64(12345), "cloid": "0xcloid",
			},
		}},
	}); err != nil {
		t.Fatal(err)
	}
	if err := bridge.handleMessage(ctx, "0xabc", map[string]any{
		"channel": "userFills",
		"data": map[string]any{
			"user":       "0xabc",
			"isSnapshot": true,
			"fills": []any{map[string]any{
				"coin": "BTC", "side": "B", "px": "95100", "sz": "0.1", "time": float64(1710000000100), "closedPnl": "12", "hash": "0xfill", "oid": float64(12345), "tid": float64(99), "fee": "0.2", "feeToken": "USDC",
			}},
		},
	}); err != nil {
		t.Fatal(err)
	}

	messages := publisher.Messages()
	if len(messages) != 2 {
		t.Fatalf("messages = %+v", messages)
	}
	orderEnvelope := decodeStreamEnvelope(t, messages[0].Payload)
	if orderEnvelope.Topic != "users/0xabc/hypertrader/order_updated" || orderEnvelope.Payload["status"] != "submitted" {
		t.Fatalf("unexpected order envelope: %+v", orderEnvelope)
	}
	fillEnvelope := decodeStreamEnvelope(t, messages[1].Payload)
	if fillEnvelope.Topic != "users/0xabc/hypertrader/fill_created" || fillEnvelope.Payload["hash"] != "0xfill" || fillEnvelope.Payload["isSnapshot"] != true {
		t.Fatalf("unexpected fill envelope: %+v", fillEnvelope)
	}
}

func TestHyperliquidStreamBridgePublishesSnapshots(t *testing.T) {
	publisher := streambridge.NewMemoryPublisher()
	streams := streambridge.NewService(publisher)
	store := NewMemoryStore()
	bridge := NewHyperliquidStreamBridge(StreamBridgeConfig{Users: []string{"0xabc"}, StateStore: store}, streams)
	ctx := context.Background()

	if err := bridge.handleMessage(ctx, "0xabc", map[string]any{
		"channel": "openOrders",
		"data": map[string]any{
			"user": "0xabc",
			"dex":  "",
			"orders": []any{map[string]any{
				"coin": "BTC", "side": "A", "limitPx": "95200", "sz": "0.2", "origSz": "0.2", "oid": float64(111), "timestamp": float64(1710000000000),
			}},
		},
	}); err != nil {
		t.Fatal(err)
	}
	if err := bridge.handleMessage(ctx, "0xabc", map[string]any{
		"channel": "clearinghouseState",
		"data": map[string]any{
			"marginSummary": map[string]any{"accountValue": "1200", "totalRawUsd": "1190"},
			"assetPositions": []any{map[string]any{"position": map[string]any{
				"coin": "BTC", "szi": "0.1", "entryPx": "95000", "positionValue": "9500", "unrealizedPnl": "10",
				"returnOnEquity": "0.01", "marginUsed": "500", "maxLeverage": float64(50), "leverage": map[string]any{"type": "cross", "value": float64(5)},
			}}},
		},
	}); err != nil {
		t.Fatal(err)
	}

	messages := publisher.Messages()
	if len(messages) != 3 {
		t.Fatalf("messages = %+v", messages)
	}
	openOrders := decodeStreamEnvelope(t, messages[0].Payload)
	if openOrders.Topic != "users/0xabc/hypertrader/open_orders" {
		t.Fatalf("unexpected open order topic: %+v", openOrders)
	}
	account := decodeStreamEnvelope(t, messages[1].Payload)
	if account.Topic != "users/0xabc/hypertrader/account_updated" || account.Payload["balance"] != "1200" {
		t.Fatalf("unexpected account envelope: %+v", account)
	}
	position := decodeStreamEnvelope(t, messages[2].Payload)
	if position.Topic != "users/0xabc/hypertrader/position_updated" {
		t.Fatalf("unexpected position envelope: %+v", position)
	}
	savedOrders, err := store.ListOpenOrdersSnapshot(ctx, "0xabc")
	if err != nil {
		t.Fatal(err)
	}
	if len(savedOrders) != 1 || savedOrders[0].ProviderOrderID != "111" {
		t.Fatalf("unexpected saved orders: %+v", savedOrders)
	}
	accountSnapshot, err := store.GetAccountSnapshot(ctx, "0xabc")
	if err != nil {
		t.Fatal(err)
	}
	if accountSnapshot.Balance != "1200" || len(accountSnapshot.Positions) != 1 {
		t.Fatalf("unexpected account snapshot: %+v", accountSnapshot)
	}
}

func TestHyperliquidStreamBridgePublishesFundingAndLedgerUpdates(t *testing.T) {
	publisher := streambridge.NewMemoryPublisher()
	streams := streambridge.NewService(publisher)
	bridge := NewHyperliquidStreamBridge(StreamBridgeConfig{Users: []string{"0xabc"}}, streams)
	ctx := context.Background()

	if err := bridge.handleMessage(ctx, "0xabc", map[string]any{
		"channel": "userFundings",
		"data": map[string]any{
			"user":       "0xabc",
			"isSnapshot": true,
			"fundings": []any{map[string]any{
				"time": float64(1710000000000), "coin": "BTC", "usdc": "1.2", "szi": "0.1", "fundingRate": "0.0001",
			}},
		},
	}); err != nil {
		t.Fatal(err)
	}
	if err := bridge.handleMessage(ctx, "0xabc", map[string]any{
		"channel": "userNonFundingLedgerUpdates",
		"data": map[string]any{
			"user": "0xabc",
			"updates": []any{map[string]any{
				"time": float64(1710000001000), "hash": "0xledger", "delta": map[string]any{"type": "deposit", "usdc": "100"},
			}},
		},
	}); err != nil {
		t.Fatal(err)
	}

	messages := publisher.Messages()
	if len(messages) != 2 {
		t.Fatalf("messages = %+v", messages)
	}
	funding := decodeStreamEnvelope(t, messages[0].Payload)
	if funding.Topic != "users/0xabc/hypertrader/funding_updated" || funding.Payload["isSnapshot"] != true {
		t.Fatalf("unexpected funding envelope: %+v", funding)
	}
	ledger := decodeStreamEnvelope(t, messages[1].Payload)
	if ledger.Topic != "users/0xabc/hypertrader/ledger_updated" {
		t.Fatalf("unexpected ledger envelope: %+v", ledger)
	}
}

func TestHyperliquidStreamBridgePersistsFillsAndOrderStatus(t *testing.T) {
	publisher := streambridge.NewMemoryPublisher()
	streams := streambridge.NewService(publisher)
	store := NewMemoryStore()
	ctx := context.Background()
	order, err := store.CreateOrder(ctx, FuturesOrder{
		ID:              "order-1",
		UserAddress:     "0xabc",
		Symbol:          "BTC",
		Side:            "buy",
		OrderType:       "limit",
		Size:            "0.1",
		Status:          "submitted",
		Provider:        "hyperliquid-http",
		ProviderOrderID: "12345",
		CreatedAt:       timeNowForTest(),
		UpdatedAt:       timeNowForTest(),
	})
	if err != nil {
		t.Fatal(err)
	}
	bridge := NewHyperliquidStreamBridge(StreamBridgeConfig{Users: []string{"0xabc"}, StateStore: store}, streams)

	if err := bridge.handleMessage(ctx, "0xabc", map[string]any{
		"channel": "orderUpdates",
		"data": []any{map[string]any{
			"status": "filled",
			"order":  map[string]any{"coin": "BTC", "side": "B", "oid": float64(12345), "sz": "0"},
		}},
	}); err != nil {
		t.Fatal(err)
	}
	updated, err := store.GetOrder(ctx, order.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != "filled" {
		t.Fatalf("expected filled order, got %+v", updated)
	}

	if err := bridge.handleMessage(ctx, "0xabc", map[string]any{
		"channel": "userFills",
		"data": map[string]any{
			"user": "0xabc",
			"fills": []any{map[string]any{
				"coin": "BTC", "side": "B", "px": "95100", "sz": "0.1", "time": float64(1710000000100), "closedPnl": "12", "hash": "0xfill", "oid": float64(12345), "tid": float64(99), "fee": "0.2", "feeToken": "USDC",
			}},
		},
	}); err != nil {
		t.Fatal(err)
	}
	fills, err := store.ListFills(ctx, "0xabc", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(fills) != 1 || fills[0].Hash != "0xfill" {
		t.Fatalf("unexpected persisted fills: %+v", fills)
	}
}

func TestHyperliquidStreamBridgeReconcilesRecentFills(t *testing.T) {
	publisher := streambridge.NewMemoryPublisher()
	streams := streambridge.NewService(publisher)
	store := NewMemoryStore()
	ctx := context.Background()
	bridge := NewHyperliquidStreamBridge(StreamBridgeConfig{
		Users:      []string{"0xabc"},
		StateStore: store,
		Provider:   reconcileProvider{LocalProvider: NewLocalProvider()},
	}, streams)

	bridge.reconcileUser(ctx, "0xabc")

	messages := publisher.Messages()
	if len(messages) != 3 {
		t.Fatalf("messages = %+v", messages)
	}
	fill := decodeStreamEnvelope(t, messages[2].Payload)
	if fill.Topic != "users/0xabc/hypertrader/fill_created" || fill.Payload["isSnapshot"] != true || fill.Payload["hash"] != "0xreconcilefill" {
		t.Fatalf("unexpected fill snapshot envelope: %+v", fill)
	}
	fills, err := store.ListFills(ctx, "0xabc", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(fills) != 1 || fills[0].Hash != "0xreconcilefill" {
		t.Fatalf("unexpected reconciled fills: %+v", fills)
	}
}

type reconcileProvider struct {
	*LocalProvider
}

func (p reconcileProvider) OpenOrders(context.Context, string) ([]OpenOrder, error) {
	return nil, nil
}

func (p reconcileProvider) Account(context.Context, string) (AccountBalance, error) {
	return AccountBalance{Balance: "1000", RawUSD: "1000"}, nil
}

func (p reconcileProvider) TradeHistory(context.Context, string, int) ([]TradeHistory, error) {
	return []TradeHistory{{Symbol: "BTC", Time: 1710000000100, PnL: "12", Dir: "Close Long", Hash: "0xreconcilefill", Oid: 12345, Px: "95100", Sz: "0.1", Fee: "0.2", FeeToken: "USDC", Tid: 99}}, nil
}

func decodeStreamEnvelope(t *testing.T, payload []byte) streambridge.Envelope {
	t.Helper()
	var envelope streambridge.Envelope
	if err := json.Unmarshal(payload, &envelope); err != nil {
		t.Fatal(err)
	}
	return envelope
}

func timeNowForTest() time.Time {
	return time.Unix(1710000000, 0).UTC()
}
