package trading

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestTradingHTTPFlow(t *testing.T) {
	handler := NewHandler(NewService(nil))
	router := chi.NewRouter()
	handler.RegisterRoutes(router)

	quoteReq := httptest.NewRequest(http.MethodPost, "/v1/trading/quote", strings.NewReader(`{"userId":"user-1","chainType":"EVM","inputToken":"ETH","outputToken":"USDC","inputAmount":"10","slippageBps":100}`))
	quoteRec := httptest.NewRecorder()
	router.ServeHTTP(quoteRec, quoteReq)
	if quoteRec.Code != http.StatusOK {
		t.Fatalf("quote status = %d body = %s", quoteRec.Code, quoteRec.Body.String())
	}
	if !strings.Contains(quoteRec.Body.String(), `"outputAmount":"9.97"`) {
		t.Fatalf("unexpected quote body: %s", quoteRec.Body.String())
	}

	createReq := httptest.NewRequest(http.MethodPost, "/v1/trading/orders", strings.NewReader(`{"userId":"user-1","chainType":"EVM","walletAddress":"0xabc","orderType":"market","side":"buy","inputToken":"ETH","outputToken":"USDC","inputAmount":"10","clientRequestId":"req-1"}`))
	createRec := httptest.NewRecorder()
	router.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusOK {
		t.Fatalf("create status = %d body = %s", createRec.Code, createRec.Body.String())
	}

	var createBody struct {
		Data Order `json:"data"`
	}
	if err := json.Unmarshal(createRec.Body.Bytes(), &createBody); err != nil {
		t.Fatal(err)
	}
	if createBody.Data.ID == "" {
		t.Fatalf("order id missing: %s", createRec.Body.String())
	}

	statusReq := httptest.NewRequest(http.MethodPost, "/v1/trading/orders/"+createBody.Data.ID+"/status", strings.NewReader(`{"status":"submitted","txHash":"0xtx"}`))
	statusRec := httptest.NewRecorder()
	router.ServeHTTP(statusRec, statusReq)
	if statusRec.Code != http.StatusOK {
		t.Fatalf("status update status = %d body = %s", statusRec.Code, statusRec.Body.String())
	}
	if !strings.Contains(statusRec.Body.String(), `"status":"submitted"`) {
		t.Fatalf("unexpected status body: %s", statusRec.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/v1/trading/orders?userId=user-1&status=submitted", nil)
	listRec := httptest.NewRecorder()
	router.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list status = %d body = %s", listRec.Code, listRec.Body.String())
	}
	if !strings.Contains(listRec.Body.String(), createBody.Data.ID) {
		t.Fatalf("order not listed: %s", listRec.Body.String())
	}

	feeReq := httptest.NewRequest(http.MethodGet, "/v1/trading/network-fee?chainType=SOLANA", nil)
	feeRec := httptest.NewRecorder()
	router.ServeHTTP(feeRec, feeReq)
	if feeRec.Code != http.StatusOK {
		t.Fatalf("fee status = %d body = %s", feeRec.Code, feeRec.Body.String())
	}
	if !strings.Contains(feeRec.Body.String(), `"platformFeeBps":30`) {
		t.Fatalf("unexpected fee body: %s", feeRec.Body.String())
	}
}

func TestTradingGraphQLFacadeFlow(t *testing.T) {
	handler := NewHandler(NewService(nil))
	router := chi.NewRouter()
	handler.RegisterRoutes(router)

	createBody := postGraphQL(t, router, `{
		"operationName":"createOrder",
		"query":"mutation createOrder($input: CreateOrderInput!) { createOrder(input: $input) { id status } }",
		"variables":{"input":{"userAddress":"0xuser","chainId":"EVM","baseAddress":"BASE","quoteAddress":"QUOTE","quoteAmount":"5","transactionType":"Buy","type":"Market","slippage":"1"}}
	}`)
	order := createBody["data"].(map[string]any)["createOrder"].(map[string]any)
	if order["id"] == "" || order["status"] != "Pending" {
		t.Fatalf("unexpected createOrder payload: %+v", order)
	}

	pendingBody := postGraphQL(t, router, `{
		"operationName":"getPendingOrders",
		"query":"query getPendingOrders($input: SearchOrderInput!) { getPendingOrders(input: $input) { total orders { id } } }",
		"variables":{"input":{"userAddress":"0xuser","limit":10}}
	}`)
	pending := pendingBody["data"].(map[string]any)["getPendingOrders"].(map[string]any)
	if pending["total"].(float64) != 1 {
		t.Fatalf("unexpected pending payload: %+v", pending)
	}

	cancelBody := postGraphQL(t, router, `{
		"operationName":"cancelOrder",
		"query":"mutation cancelOrder($id: ID!) { cancelOrder(id: $id) { id status } }",
		"variables":{"id":"`+order["id"].(string)+`"}
	}`)
	cancelled := cancelBody["data"].(map[string]any)["cancelOrder"].(map[string]any)
	if cancelled["status"] != "Canceled" {
		t.Fatalf("unexpected cancel payload: %+v", cancelled)
	}

	feeBody := postGraphQL(t, router, `{
		"operationName":"getNetworkFee",
		"query":"query getNetworkFee($input: NetworkFeeInput!) { getNetworkFee(input: $input) { solana { platformFee maxComputeUnits } } }",
		"variables":{"input":{"chain":"SOLANA"}}
	}`)
	solana := feeBody["data"].(map[string]any)["getNetworkFee"].(map[string]any)["solana"].(map[string]any)
	if solana["platformFee"].(float64) != 30 {
		t.Fatalf("unexpected fee payload: %+v", solana)
	}
}

func postGraphQL(t *testing.T, router http.Handler, body string) map[string]any {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/trading/trading-gql", strings.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["errors"] != nil {
		t.Fatalf("graphql errors: %s", rec.Body.String())
	}
	return payload
}
