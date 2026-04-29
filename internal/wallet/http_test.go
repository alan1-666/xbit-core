package wallet

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestWalletHTTPFlow(t *testing.T) {
	handler := NewHandler(NewService(nil))
	router := chi.NewRouter()
	handler.RegisterRoutes(router)

	createReq := httptest.NewRequest(http.MethodPost, "/v1/wallets", strings.NewReader(`{"userId":"user-1","chainType":"EVM","address":"0xabc","walletType":"embedded","name":"Main"}`))
	createRec := httptest.NewRecorder()
	router.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", createRec.Code, createRec.Body.String())
	}

	var createBody struct {
		Data Wallet `json:"data"`
	}
	if err := json.Unmarshal(createRec.Body.Bytes(), &createBody); err != nil {
		t.Fatal(err)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/v1/wallets?userId=user-1", nil)
	listRec := httptest.NewRecorder()
	router.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", listRec.Code, listRec.Body.String())
	}
	if !strings.Contains(listRec.Body.String(), createBody.Data.ID) {
		t.Fatalf("wallet not listed: %s", listRec.Body.String())
	}

	renameReq := httptest.NewRequest(http.MethodPatch, "/v1/wallets/"+createBody.Data.ID, strings.NewReader(`{"userId":"user-1","name":"Trading"}`))
	renameRec := httptest.NewRecorder()
	router.ServeHTTP(renameRec, renameReq)
	if renameRec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", renameRec.Code, renameRec.Body.String())
	}
	if !strings.Contains(renameRec.Body.String(), "Trading") {
		t.Fatalf("unexpected rename response: %s", renameRec.Body.String())
	}
}
