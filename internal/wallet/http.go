package wallet

import (
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/xbit/xbit-backend/internal/httpx"
)

type Handler struct {
	service *Service
}

type response struct {
	Data  any    `json:"data,omitempty"`
	Error string `json:"error,omitempty"`
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(router chi.Router) {
	router.Route("/v1", func(r chi.Router) {
		r.Get("/wallets", h.listWallets)
		r.Post("/wallets", h.createWallet)
		r.Patch("/wallets/order", h.updateWalletOrder)
		r.Patch("/wallets/{walletID}", h.updateWalletName)
		r.Get("/wallet-whitelist", h.listWhitelist)
		r.Post("/wallet-whitelist", h.addWhitelist)
		r.Post("/wallet-security-events", h.recordSecurityEvent)
	})
}

func (h *Handler) createWallet(w http.ResponseWriter, r *http.Request) {
	var req CreateWalletInput
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, response{Error: "invalid json body"})
		return
	}
	wallet, err := h.service.CreateWallet(r.Context(), req)
	h.writeResult(w, wallet, err)
}

func (h *Handler) listWallets(w http.ResponseWriter, r *http.Request) {
	wallets, err := h.service.ListWallets(r.Context(), userIDFromRequest(r))
	h.writeResult(w, wallets, err)
}

func (h *Handler) updateWalletName(w http.ResponseWriter, r *http.Request) {
	var req UpdateWalletNameInput
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, response{Error: "invalid json body"})
		return
	}
	if req.UserID == "" {
		req.UserID = userIDFromRequest(r)
	}
	wallet, err := h.service.UpdateWalletName(r.Context(), req.UserID, chi.URLParam(r, "walletID"), req.Name)
	h.writeResult(w, wallet, err)
}

func (h *Handler) updateWalletOrder(w http.ResponseWriter, r *http.Request) {
	var req UpdateWalletOrderInput
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, response{Error: "invalid json body"})
		return
	}
	if req.UserID == "" {
		req.UserID = userIDFromRequest(r)
	}
	h.writeResult(w, map[string]bool{"ok": true}, h.service.UpdateWalletOrder(r.Context(), req))
}

func (h *Handler) addWhitelist(w http.ResponseWriter, r *http.Request) {
	var req AddWhitelistInput
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, response{Error: "invalid json body"})
		return
	}
	entry, err := h.service.AddWhitelist(r.Context(), req)
	h.writeResult(w, entry, err)
}

func (h *Handler) listWhitelist(w http.ResponseWriter, r *http.Request) {
	entries, err := h.service.ListWhitelist(r.Context(), userIDFromRequest(r))
	h.writeResult(w, entries, err)
}

func (h *Handler) recordSecurityEvent(w http.ResponseWriter, r *http.Request) {
	var req RecordSecurityEventInput
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, response{Error: "invalid json body"})
		return
	}
	event, err := h.service.RecordSecurityEvent(r.Context(), req)
	h.writeResult(w, event, err)
}

func (h *Handler) writeResult(w http.ResponseWriter, data any, err error) {
	if err == nil {
		httpx.WriteJSON(w, http.StatusOK, response{Data: data})
		return
	}
	if errors.Is(err, ErrNotFound) {
		httpx.WriteJSON(w, http.StatusNotFound, response{Error: err.Error()})
		return
	}
	httpx.WriteJSON(w, http.StatusBadRequest, response{Error: err.Error()})
}

func userIDFromRequest(r *http.Request) string {
	if userID := strings.TrimSpace(r.Header.Get("X-User-Id")); userID != "" {
		return userID
	}
	return strings.TrimSpace(r.URL.Query().Get("userId"))
}
