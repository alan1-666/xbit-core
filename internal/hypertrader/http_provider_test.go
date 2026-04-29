package hypertrader

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHTTPProviderInfoEndpoints(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/info" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		switch req["type"] {
		case "fundingHistory":
			writeProviderJSON(t, w, []map[string]any{{"coin": req["coin"], "fundingRate": "0.00012", "premium": "0.00001", "time": float64(1710000000000)}})
		case "userFills":
			writeProviderJSON(t, w, []map[string]any{{
				"coin": "BTC", "time": float64(1710000000000), "closedPnl": "12", "dir": "Close Long", "hash": "0xfill",
				"oid": float64(12345), "px": "95100", "startPosition": "0.1", "sz": "0.1", "fee": "0.2", "feeToken": "USDC", "tid": float64(99),
			}})
		case "frontendOpenOrders":
			writeProviderJSON(t, w, []map[string]any{{
				"coin": "BTC", "limitPx": "95000", "oid": float64(12345), "orderType": "Limit", "origSz": "0.1",
				"reduceOnly": false, "side": "B", "sz": "0.04", "timestamp": float64(1710000000000), "tif": "Gtc", "cloid": "0xabc",
			}})
		case "orderStatus":
			if req["user"] != "0xuser" || req["oid"] == nil {
				t.Fatalf("unexpected order status request: %+v", req)
			}
			writeProviderJSON(t, w, map[string]any{
				"status": "order",
				"order": map[string]any{
					"status":          "open",
					"statusTimestamp": float64(1710000000000),
					"order": map[string]any{
						"coin":    "BTC",
						"limitPx": "95000",
						"oid":     float64(12345),
						"origSz":  "0.1",
						"sz":      "0.04",
						"cloid":   "0xabc",
					},
				},
			})
		case "clearinghouseState":
			writeProviderJSON(t, w, map[string]any{
				"marginSummary": map[string]any{"accountValue": "1200", "totalRawUsd": "1190"},
				"assetPositions": []map[string]any{{"position": map[string]any{
					"coin": "BTC", "szi": "0.1", "entryPx": "95000", "positionValue": "9500", "unrealizedPnl": "10",
					"returnOnEquity": "0.01", "liquidationPx": "60000", "marginUsed": "500", "maxLeverage": float64(50),
					"leverage":   map[string]any{"type": "cross", "value": float64(5)},
					"cumFunding": map[string]any{"allTime": "1", "sinceOpen": "0.5", "sinceChange": "0.2"},
				}}},
			})
		default:
			t.Fatalf("unexpected info type: %v", req["type"])
		}
	}))
	defer server.Close()

	provider := NewHTTPProvider(server.URL, time.Second)
	rates, err := provider.FundingRates(t.Context(), "BTC", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(rates) != 1 || rates[0].Symbol != "BTC" || rates[0].FundingRate != "0.00012" {
		t.Fatalf("unexpected funding rates: %+v", rates)
	}

	account, err := provider.Account(t.Context(), "0xuser")
	if err != nil {
		t.Fatal(err)
	}
	if account.Balance != "1200" || len(account.Positions) != 1 || account.Positions[0].Coin != "BTC" {
		t.Fatalf("unexpected account: %+v", account)
	}

	trades, err := provider.TradeHistory(t.Context(), "0xuser", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(trades) != 1 || trades[0].Hash != "0xfill" || trades[0].Symbol != "BTC" {
		t.Fatalf("unexpected trades: %+v", trades)
	}

	openOrders, err := provider.OpenOrders(t.Context(), "0xuser")
	if err != nil {
		t.Fatal(err)
	}
	if len(openOrders) != 1 || openOrders[0].Side != "buy" || openOrders[0].ProviderOrderID != "12345" {
		t.Fatalf("unexpected open orders: %+v", openOrders)
	}

	status, err := provider.OrderStatus(t.Context(), OrderStatusInput{UserAddress: "0xuser", ProviderOrderID: "12345"})
	if err != nil {
		t.Fatal(err)
	}
	if status.Status != "open" || status.ProviderOrderID != "12345" || status.FilledSize != "0.06" {
		t.Fatalf("unexpected order status: %+v", status)
	}
}

func TestHTTPProviderExchangeRequiresSignedPayload(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.URL.Path != "/exchange" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		if req["action"] == nil || req["signature"] == nil || req["nonce"] == nil {
			t.Fatalf("unsigned exchange request: %+v", req)
		}
		writeProviderJSON(t, w, map[string]any{"status": "ok", "hash": "0xprovider"})
	}))
	defer server.Close()

	provider := NewHTTPProvider(server.URL, time.Second)
	_, err := provider.SubmitOrder(t.Context(), FuturesOrder{RawPayload: map[string]any{}})
	if err == nil {
		t.Fatal("expected unsigned submit to fail")
	}
	if requests != 0 {
		t.Fatalf("unsigned request should not hit provider, got %d requests", requests)
	}

	result, err := provider.SubmitOrder(t.Context(), FuturesOrder{RawPayload: map[string]any{"exchangePayload": signedProviderPayload()}})
	if err != nil {
		t.Fatal(err)
	}
	if result.RequestID != "0xprovider" || result.Status != "ok" {
		t.Fatalf("unexpected exchange result: %+v", result)
	}
}

func signedProviderPayload() map[string]any {
	return map[string]any{
		"action":    map[string]any{"type": "order"},
		"signature": map[string]any{"r": "0x1", "s": "0x2", "v": 27},
		"nonce":     1710000000000,
	}
}

func writeProviderJSON(t *testing.T, w http.ResponseWriter, payload any) {
	t.Helper()
	w.Header().Set("content-type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Fatal(err)
	}
}
