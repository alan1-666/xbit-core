package identity

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandlerNonce(t *testing.T) {
	handler := NewHandler(newTestService(t), true)
	router := newIdentityTestRouter(handler)

	req := httptest.NewRequest(http.MethodPost, "/v1/auth/nonce", strings.NewReader(`{"walletAddress":"0xabc","chainType":"EVM"}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "0xabc") {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

func TestHandlerDevLoginAndMe(t *testing.T) {
	handler := NewHandler(newTestService(t), true)
	router := newIdentityTestRouter(handler)

	loginReq := httptest.NewRequest(http.MethodPost, "/v1/auth/dev-login", strings.NewReader(`{"userId":"user-1","deviceId":"device-1"}`))
	loginRec := httptest.NewRecorder()
	router.ServeHTTP(loginRec, loginReq)

	if loginRec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", loginRec.Code, loginRec.Body.String())
	}

	var body struct {
		Data struct {
			AccessToken string `json:"accessToken"`
		} `json:"data"`
	}
	if err := json.Unmarshal(loginRec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Data.AccessToken == "" {
		t.Fatalf("access token missing: %s", loginRec.Body.String())
	}

	meReq := httptest.NewRequest(http.MethodGet, "/v1/auth/me", nil)
	meReq.Header.Set("Authorization", "Bearer "+body.Data.AccessToken)
	meRec := httptest.NewRecorder()
	router.ServeHTTP(meRec, meReq)

	if meRec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", meRec.Code, meRec.Body.String())
	}
	if !strings.Contains(meRec.Body.String(), "user-1") {
		t.Fatalf("unexpected body: %s", meRec.Body.String())
	}
}

func TestHandlerRefreshAndLogout(t *testing.T) {
	handler := NewHandler(newTestService(t), true)
	router := newIdentityTestRouter(handler)

	loginReq := httptest.NewRequest(http.MethodPost, "/v1/auth/dev-login", strings.NewReader(`{"userId":"user-1"}`))
	loginRec := httptest.NewRecorder()
	router.ServeHTTP(loginRec, loginReq)

	var loginBody struct {
		Data struct {
			RefreshToken string `json:"refreshToken"`
		} `json:"data"`
	}
	if err := json.Unmarshal(loginRec.Body.Bytes(), &loginBody); err != nil {
		t.Fatal(err)
	}

	refreshReq := httptest.NewRequest(http.MethodPost, "/v1/auth/refresh", strings.NewReader(`{"refreshToken":"`+loginBody.Data.RefreshToken+`"}`))
	refreshRec := httptest.NewRecorder()
	router.ServeHTTP(refreshRec, refreshReq)
	if refreshRec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", refreshRec.Code, refreshRec.Body.String())
	}

	var refreshBody struct {
		Data struct {
			RefreshToken string `json:"refreshToken"`
		} `json:"data"`
	}
	if err := json.Unmarshal(refreshRec.Body.Bytes(), &refreshBody); err != nil {
		t.Fatal(err)
	}
	if refreshBody.Data.RefreshToken == "" || refreshBody.Data.RefreshToken == loginBody.Data.RefreshToken {
		t.Fatalf("expected rotated refresh token")
	}

	logoutReq := httptest.NewRequest(http.MethodPost, "/v1/auth/logout", strings.NewReader(`{"refreshToken":"`+refreshBody.Data.RefreshToken+`"}`))
	logoutRec := httptest.NewRecorder()
	router.ServeHTTP(logoutRec, logoutReq)
	if logoutRec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", logoutRec.Code, logoutRec.Body.String())
	}
}

func TestBearerToken(t *testing.T) {
	if got := bearerToken("Bearer abc"); got != "abc" {
		t.Fatalf("got %q", got)
	}
	if got := bearerToken("Basic abc"); got != "" {
		t.Fatalf("got %q", got)
	}
}

func newIdentityTestRouter(handler *Handler) http.Handler {
	router := chiRouter()
	handler.RegisterRoutes(router)
	return router
}
