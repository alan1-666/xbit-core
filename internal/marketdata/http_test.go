package marketdata

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestMarketDataHTTPFlow(t *testing.T) {
	router := chi.NewRouter()
	NewHandler(NewService(nil)).RegisterRoutes(router)

	listRec := httptest.NewRecorder()
	router.ServeHTTP(listRec, httptest.NewRequest(http.MethodGet, "/v1/market/tokens?limit=2", nil))
	if listRec.Code != http.StatusOK {
		t.Fatalf("list status = %d body = %s", listRec.Code, listRec.Body.String())
	}

	var listBody struct {
		Data TokenList `json:"data"`
	}
	if err := json.Unmarshal(listRec.Body.Bytes(), &listBody); err != nil {
		t.Fatal(err)
	}
	if len(listBody.Data.Data) == 0 {
		t.Fatalf("empty list: %s", listRec.Body.String())
	}
	token := listBody.Data.Data[0]

	detailRec := httptest.NewRecorder()
	router.ServeHTTP(detailRec, httptest.NewRequest(http.MethodGet, "/v1/market/tokens/"+strconvItoa(token.ChainID)+"/"+token.Address, nil))
	if detailRec.Code != http.StatusOK {
		t.Fatalf("detail status = %d body = %s", detailRec.Code, detailRec.Body.String())
	}

	ohlcRec := httptest.NewRecorder()
	router.ServeHTTP(ohlcRec, httptest.NewRequest(http.MethodGet, "/v1/market/tokens/"+strconvItoa(token.ChainID)+"/"+token.Address+"/ohlc?limit=3", nil))
	if ohlcRec.Code != http.StatusOK || !strings.Contains(ohlcRec.Body.String(), `"close"`) {
		t.Fatalf("ohlc status = %d body = %s", ohlcRec.Code, ohlcRec.Body.String())
	}
}

func TestMarketDataGraphQLFacade(t *testing.T) {
	router := chi.NewRouter()
	NewHandler(NewService(nil)).RegisterRoutes(router)

	trending := postMarketGraphQL(t, router, `{
		"operationName":"GetTokenTrending",
		"query":"query GetTokenTrending($input: TokenTrendingInput!) { getTokenTrending(input: $input) { data { token symbol price } } }",
		"variables":{"input":{"limit":2}}
	}`)
	data := trending["data"].(map[string]any)["getTokenTrending"].(map[string]any)["data"].([]any)
	if len(data) == 0 {
		t.Fatalf("trending empty: %+v", trending)
	}
	first := data[0].(map[string]any)

	detail := postMarketGraphQL(t, router, `{
		"operationName":"GetTokenDetail",
		"query":"query GetTokenDetail($input: TokenDetailInput!) { getTokenDetail(token: $input) { address symbol price } }",
		"variables":{"input":{"chainId":`+strconvFloat(first["chainId"])+`,"token":"`+first["token"].(string)+`"}}
	}`)
	if detail["data"].(map[string]any)["getTokenDetail"].(map[string]any)["symbol"] == "" {
		t.Fatalf("detail missing symbol: %+v", detail)
	}

	search := postMarketGraphQL(t, router, `{
		"operationName":"SearchUniversal",
		"query":"query SearchUniversal($input: SearchInput!) { searchUniversal(input: $input) { data { ... on SearchData { token symbol } } } }",
		"variables":{"input":{"keyword":"sol"}}
	}`)
	searchData := search["data"].(map[string]any)["searchUniversal"].(map[string]any)["data"].([]any)
	if len(searchData) == 0 {
		t.Fatalf("search empty: %+v", search)
	}

	ohlc := postMarketGraphQL(t, router, `{
		"operationName":"GetOHLC",
		"query":"query GetOHLC($input: OHLCInput!) { getOHLC(input: $input) { ts close } }",
		"variables":{"input":{"chainId":`+strconvFloat(first["chainId"])+`,"token":"`+first["token"].(string)+`","limit":4}}
	}`)
	points := ohlc["data"].(map[string]any)["getOHLC"].([]any)
	if len(points) != 4 {
		t.Fatalf("ohlc = %+v", ohlc)
	}
}

func postMarketGraphQL(t *testing.T, router http.Handler, body string) map[string]any {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/meme2/meme-gql", strings.NewReader(body))
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

func strconvItoa(value int) string {
	return strconv.FormatInt(int64(value), 10)
}

func strconvFloat(value any) string {
	switch v := value.(type) {
	case float64:
		return strconv.FormatFloat(v, 'f', 0, 64)
	default:
		return strconv.FormatInt(int64(v.(int)), 10)
	}
}
