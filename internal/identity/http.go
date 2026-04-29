package identity

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/xbit/xbit-backend/internal/httpx"
)

type Handler struct {
	service        *Service
	devAuthEnabled bool
}

type response struct {
	Data  any    `json:"data,omitempty"`
	Error string `json:"error,omitempty"`
}

type nonceRequest struct {
	WalletAddress string `json:"walletAddress"`
	ChainType     string `json:"chainType"`
}

func NewHandler(service *Service, devAuthEnabled bool) *Handler {
	return &Handler{service: service, devAuthEnabled: devAuthEnabled}
}

func (h *Handler) RegisterRoutes(router chi.Router) {
	router.Route("/v1/auth", func(r chi.Router) {
		r.Post("/nonce", h.createNonce)
		r.Post("/dev-login", h.devLogin)
		r.Post("/refresh", h.refresh)
		r.Post("/logout", h.logout)
		r.Get("/me", h.me)
	})
}

func (h *Handler) createNonce(w http.ResponseWriter, r *http.Request) {
	var req nonceRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, response{Error: "invalid json body"})
		return
	}
	challenge, err := h.service.CreateNonceChallenge(r.Context(), req.WalletAddress, req.ChainType)
	if err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, response{Error: err.Error()})
		return
	}
	httpx.WriteJSON(w, http.StatusOK, response{Data: challenge})
}

func (h *Handler) devLogin(w http.ResponseWriter, r *http.Request) {
	if !h.devAuthEnabled {
		httpx.WriteJSON(w, http.StatusForbidden, response{Error: "dev auth is disabled"})
		return
	}
	var req DevLoginInput
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, response{Error: "invalid json body"})
		return
	}
	req.UserAgent = r.UserAgent()
	req.IPAddress = clientIP(r)
	pair, err := h.service.IssueDevLogin(r.Context(), req)
	if err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, response{Error: err.Error()})
		return
	}
	httpx.WriteJSON(w, http.StatusOK, response{Data: pair})
}

func (h *Handler) refresh(w http.ResponseWriter, r *http.Request) {
	var req RefreshInput
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, response{Error: "invalid json body"})
		return
	}
	pair, err := h.service.Refresh(r.Context(), req)
	if err != nil {
		httpx.WriteJSON(w, http.StatusUnauthorized, response{Error: err.Error()})
		return
	}
	httpx.WriteJSON(w, http.StatusOK, response{Data: pair})
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	var req LogoutInput
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, response{Error: "invalid json body"})
		return
	}
	if err := h.service.Logout(r.Context(), req); err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, response{Error: err.Error()})
		return
	}
	httpx.WriteJSON(w, http.StatusOK, response{Data: map[string]bool{"ok": true}})
}

func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	token := bearerToken(r.Header.Get("Authorization"))
	if token == "" {
		httpx.WriteJSON(w, http.StatusUnauthorized, response{Error: "missing bearer token"})
		return
	}
	claims, err := h.service.VerifyAccessToken(token)
	if err != nil {
		httpx.WriteJSON(w, http.StatusUnauthorized, response{Error: "invalid bearer token"})
		return
	}
	httpx.WriteJSON(w, http.StatusOK, response{Data: claims})
}

func bearerToken(header string) string {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(header, prefix))
}

func clientIP(r *http.Request) string {
	if value := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); value != "" {
		parts := strings.Split(value, ",")
		return strings.TrimSpace(parts[0])
	}
	return strings.TrimSpace(strings.Split(r.RemoteAddr, ":")[0])
}
