package hypertrader

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestHypertraderHTTPFlow(t *testing.T) {
	router := chi.NewRouter()
	NewHandler(NewService(nil)).RegisterRoutes(router)

	symbolRec := httptest.NewRecorder()
	router.ServeHTTP(symbolRec, httptest.NewRequest(http.MethodGet, "/v1/futures/symbols?limit=2", nil))
	if symbolRec.Code != http.StatusOK {
		t.Fatalf("symbols status = %d body = %s", symbolRec.Code, symbolRec.Body.String())
	}
	if !strings.Contains(symbolRec.Body.String(), `"type":"PERP"`) {
		t.Fatalf("unexpected symbols body: %s", symbolRec.Body.String())
	}

	accountRec := httptest.NewRecorder()
	router.ServeHTTP(accountRec, httptest.NewRequest(http.MethodGet, "/v1/futures/account?userAddress=0xuser", nil))
	if accountRec.Code != http.StatusOK || !strings.Contains(accountRec.Body.String(), `"positions"`) {
		t.Fatalf("account status = %d body = %s", accountRec.Code, accountRec.Body.String())
	}

	createRec := httptest.NewRecorder()
	router.ServeHTTP(createRec, httptest.NewRequest(http.MethodPost, "/v1/futures/orders", strings.NewReader(`{"userId":"user-1","userAddress":"0xuser","symbol":"BTC","side":"buy","orderType":"market","size":"0.1","clientRequestId":"http-req-1"}`)))
	if createRec.Code != http.StatusOK {
		t.Fatalf("create order status = %d body = %s", createRec.Code, createRec.Body.String())
	}
	var createBody struct {
		Data FuturesOrder `json:"data"`
	}
	if err := json.Unmarshal(createRec.Body.Bytes(), &createBody); err != nil {
		t.Fatal(err)
	}
	if createBody.Data.Status != "submitted" {
		t.Fatalf("unexpected order body: %s", createRec.Body.String())
	}

	cancelRec := httptest.NewRecorder()
	router.ServeHTTP(cancelRec, httptest.NewRequest(http.MethodPost, "/v1/futures/orders/"+createBody.Data.ID+"/cancel", strings.NewReader(`{"userId":"user-1"}`)))
	if cancelRec.Code != http.StatusOK || !strings.Contains(cancelRec.Body.String(), `"status":"cancelled"`) {
		t.Fatalf("cancel status = %d body = %s", cancelRec.Code, cancelRec.Body.String())
	}

	ratesRec := httptest.NewRecorder()
	router.ServeHTTP(ratesRec, httptest.NewRequest(http.MethodGet, "/v1/futures/funding-rates?limit=2", nil))
	if ratesRec.Code != http.StatusOK || !strings.Contains(ratesRec.Body.String(), `"fundingRate"`) {
		t.Fatalf("rates status = %d body = %s", ratesRec.Code, ratesRec.Body.String())
	}
}

func TestHypertraderGraphQLFacades(t *testing.T) {
	router := chi.NewRouter()
	NewHandler(NewService(nil)).RegisterRoutes(router)

	symbols := postHyperGraphQL(t, router, "/api/graphql-dex", `{
		"operationName":"GetSymbolList",
		"query":"query GetSymbolList { getSymbolList { list { symbol type } } }",
		"variables":{"input":{"limit":3}}
	}`)
	list := symbols["data"].(map[string]any)["getSymbolList"].(map[string]any)["list"].([]any)
	if len(list) == 0 {
		t.Fatalf("symbol list empty: %+v", symbols)
	}

	position := postHyperGraphQL(t, router, "/api/graphql-dex", `{
		"operationName":"GetUserPosition",
		"query":"query GetUserPosition { getUserPosition { rawUSD positions { coin } } }",
		"variables":{"userAddress":"0xuser"}
	}`)
	positions := position["data"].(map[string]any)["getUserPosition"].(map[string]any)["positions"].([]any)
	if len(positions) == 0 {
		t.Fatalf("positions empty: %+v", position)
	}

	smartMoney := postHyperGraphQL(t, router, "/api/dex-hypertrader/graphql", `{
		"operationName":"GetActiveSmartMoney",
		"query":"query GetActiveSmartMoney { getActiveSmartMoney { data { userAddress roi } } }",
		"variables":{"pageSize":2}
	}`)
	traders := smartMoney["data"].(map[string]any)["getActiveSmartMoney"].(map[string]any)["data"].([]any)
	if len(traders) == 0 {
		t.Fatalf("smart money empty: %+v", smartMoney)
	}

	group := postHyperGraphQL(t, router, "/api/dex-hypertrader/graphql", `{
		"operationName":"CreateAddressGroup",
		"query":"mutation CreateAddressGroup($input: CreateAddressGroupInput!) { createAddressGroup(input: $input) { id name } }",
		"variables":{"input":{"name":"Desk A","userId":"user-1"}}
	}`)
	created := group["data"].(map[string]any)["createAddressGroup"].(map[string]any)
	if created["id"] == "" || created["name"] != "Desk A" {
		t.Fatalf("unexpected group: %+v", created)
	}

	wallet := postHyperGraphQL(t, router, "/api/user/user-gql", `{
		"operationName":"CheckHyperLiquidWallet",
		"query":"query CheckHyperLiquidWallet { checkHyperLiquidWallet { approvedAgent } }",
		"variables":{"userId":"user-1"}
	}`)
	status := wallet["data"].(map[string]any)["checkHyperLiquidWallet"].(map[string]any)
	if status["approvedAgent"] != true {
		t.Fatalf("unexpected wallet status: %+v", status)
	}

	signature := postHyperGraphQL(t, router, "/api/user/user-gql", `{
		"operationName":"signHyperLiquidCreateOrder",
		"query":"mutation signHyperLiquidCreateOrder($input: SignInput!) { signHyperLiquidCreateOrder(input: $input) { signature { r s v } } }",
		"variables":{"input":{"userId":"user-1","symbol":"BTC","size":"0.1"}}
	}`)
	signed := signature["data"].(map[string]any)["signHyperLiquidCreateOrder"].(map[string]any)
	if signed["signature"] == nil {
		t.Fatalf("signature missing: %+v", signed)
	}

	orderBody := postHyperGraphQL(t, router, "/api/dex-hypertrader/graphql", `{
		"operationName":"CreateHyperLiquidOrder",
		"query":"mutation CreateHyperLiquidOrder($input: OrderInput!) { createHyperLiquidOrder(input: $input) { id status providerOrderId } }",
		"variables":{"input":{"userId":"user-1","userAddress":"0xuser","symbol":"BTC","side":"buy","orderType":"market","size":"0.1","clientRequestId":"gql-req-1"}}
	}`)
	order := orderBody["data"].(map[string]any)["createHyperLiquidOrder"].(map[string]any)
	if order["status"] != "submitted" || order["providerOrderId"] == "" {
		t.Fatalf("unexpected gql order: %+v", order)
	}

	rates := postHyperGraphQL(t, router, "/api/dex-hypertrader/graphql", `{
		"operationName":"GetFundingRates",
		"query":"query GetFundingRates { getFundingRates { symbol fundingRate } }",
		"variables":{"input":{"limit":2}}
	}`)
	if len(rates["data"].(map[string]any)["getFundingRates"].([]any)) == 0 {
		t.Fatalf("funding rates empty: %+v", rates)
	}
}

func postHyperGraphQL(t *testing.T, router http.Handler, path string, body string) map[string]any {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
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
