package streambridge

import (
	"context"
	"encoding/json"
	"testing"
)

func TestPublishMarketTokenUpdateMapsTopics(t *testing.T) {
	publisher := NewMemoryPublisher()
	svc := NewService(publisher)

	result, err := svc.Publish(context.Background(), Event{
		Type:    EventMarketTokenUpdated,
		Source:  "market-data",
		ChainID: "SOLANA",
		Token:   "So111",
		Payload: map[string]any{"price": "1.23"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Topics) != 2 {
		t.Fatalf("topics = %+v", result.Topics)
	}
	if result.Topics[0] != "public/meme/token_info/501/So111" {
		t.Fatalf("topic[0] = %s", result.Topics[0])
	}
	if result.Topics[1] != "public/token_statistic/501/So111" {
		t.Fatalf("topic[1] = %s", result.Topics[1])
	}

	messages := publisher.Messages()
	if len(messages) != 2 {
		t.Fatalf("messages = %+v", messages)
	}
	var envelope Envelope
	if err := json.Unmarshal(messages[0].Payload, &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Seq != 1 || envelope.Payload["price"] != "1.23" {
		t.Fatalf("envelope = %+v", envelope)
	}
}

func TestPublishTradingOrderUpdateMapsUserTopic(t *testing.T) {
	svc := NewService(NewMemoryPublisher())

	result, err := svc.Publish(context.Background(), Event{
		Type:    EventTradingOrderUpdated,
		UserID:  "user-1",
		Payload: map[string]any{"orderId": "order-1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Topics) != 1 || result.Topics[0] != "users/user-1/order_updated" {
		t.Fatalf("topics = %+v", result.Topics)
	}

	recent := svc.Recent("users/user-1/order_updated", 10)
	if len(recent) != 1 || recent[0].Payload["orderId"] != "order-1" {
		t.Fatalf("recent = %+v", recent)
	}
}

func TestPublishHypertraderEventsMapUserTopics(t *testing.T) {
	cases := []struct {
		eventType string
		topic     string
	}{
		{EventHypertraderOrderUpdated, "users/0xabc/hypertrader/order_updated"},
		{EventHypertraderFillCreated, "users/0xabc/hypertrader/fill_created"},
		{EventHypertraderOpenOrders, "users/0xabc/hypertrader/open_orders"},
		{EventHypertraderAccountUpdated, "users/0xabc/hypertrader/account_updated"},
		{EventHypertraderPositionUpdated, "users/0xabc/hypertrader/position_updated"},
		{EventHypertraderFundingUpdated, "users/0xabc/hypertrader/funding_updated"},
		{EventHypertraderLedgerUpdated, "users/0xabc/hypertrader/ledger_updated"},
		{EventHypertraderRawEvent, "users/0xabc/hypertrader/event"},
	}
	for _, tc := range cases {
		svc := NewService(NewMemoryPublisher())
		result, err := svc.Publish(context.Background(), Event{
			Type:    tc.eventType,
			UserID:  "0xabc",
			Payload: map[string]any{"ok": true},
		})
		if err != nil {
			t.Fatalf("%s publish: %v", tc.eventType, err)
		}
		if len(result.Topics) != 1 || result.Topics[0] != tc.topic {
			t.Fatalf("%s topics = %+v", tc.eventType, result.Topics)
		}
	}
}

func TestPublishRejectsUnknownEventWithoutTopic(t *testing.T) {
	svc := NewService(NewMemoryPublisher())
	if _, err := svc.Publish(context.Background(), Event{Type: "unknown"}); err == nil {
		t.Fatalf("expected error")
	}
}
