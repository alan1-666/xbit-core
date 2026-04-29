package streambridge

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestStreamBridgeHTTPFlow(t *testing.T) {
	router := chi.NewRouter()
	NewHandler(NewService(NewMemoryPublisher())).RegisterRoutes(router)

	body := `{"type":"market.ohlc.updated","chainId":"501","token":"xbit-demo-token","bucket":"1m","payload":{"close":"0.042"}}`
	publishReq := httptest.NewRequest(http.MethodPost, "/v1/stream/events", strings.NewReader(body))
	publishRec := httptest.NewRecorder()
	router.ServeHTTP(publishRec, publishReq)
	if publishRec.Code != http.StatusOK {
		t.Fatalf("publish status = %d body = %s", publishRec.Code, publishRec.Body.String())
	}
	if !strings.Contains(publishRec.Body.String(), "public/kline/ohlc_1m/xbit-demo-token") {
		t.Fatalf("unexpected publish body: %s", publishRec.Body.String())
	}

	topicsRec := httptest.NewRecorder()
	router.ServeHTTP(topicsRec, httptest.NewRequest(http.MethodGet, "/v1/stream/topics", nil))
	if topicsRec.Code != http.StatusOK {
		t.Fatalf("topics status = %d body = %s", topicsRec.Code, topicsRec.Body.String())
	}

	recentRec := httptest.NewRecorder()
	router.ServeHTTP(recentRec, httptest.NewRequest(http.MethodGet, "/v1/stream/events?topic=public/kline/ohlc_1m/xbit-demo-token", nil))
	if recentRec.Code != http.StatusOK {
		t.Fatalf("recent status = %d body = %s", recentRec.Code, recentRec.Body.String())
	}
	var recentBody struct {
		Data []Envelope `json:"data"`
	}
	if err := json.Unmarshal(recentRec.Body.Bytes(), &recentBody); err != nil {
		t.Fatal(err)
	}
	if len(recentBody.Data) != 1 || recentBody.Data[0].Payload["close"] != "0.042" {
		t.Fatalf("recent = %+v", recentBody)
	}
}
