package hypertrader

import (
	"context"
	"encoding/json"
	"testing"

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
	bridge := NewHyperliquidStreamBridge(StreamBridgeConfig{Users: []string{"0xabc"}}, streams)
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

func decodeStreamEnvelope(t *testing.T, payload []byte) streambridge.Envelope {
	t.Helper()
	var envelope streambridge.Envelope
	if err := json.Unmarshal(payload, &envelope); err != nil {
		t.Fatal(err)
	}
	return envelope
}
