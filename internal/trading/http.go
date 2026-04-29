package trading

import (
	"errors"
	"net/http"
	"strconv"

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
	router.Post("/graphql", h.graphql)
	router.Post("/trading-gql", h.graphql)
	router.Post("/api/trading/trading-gql", h.graphql)

	router.Route("/v1/trading", func(r chi.Router) {
		r.Get("/exchange-meta", h.exchangeMeta)
		r.Post("/quote", h.quote)
		r.Get("/network-fee", h.networkFee)
		r.Get("/orders", h.listOrders)
		r.Post("/orders", h.createOrder)
		r.Get("/orders/{orderID}", h.getOrder)
		r.Post("/orders/{orderID}/status", h.updateStatus)
		r.Post("/orders/{orderID}/cancel", h.cancelOrder)
	})
}

func (h *Handler) exchangeMeta(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, response{Data: map[string]any{
		"providers": []string{"internal-mvp"},
		"chains":    []string{"SOLANA", "EVM", "BSC", "ARB", "MON"},
		"orderTypes": []string{
			OrderTypeMarket,
			OrderTypeLimit,
		},
		"sides": []string{SideBuy, SideSell},
	}})
}

func (h *Handler) quote(w http.ResponseWriter, r *http.Request) {
	var req QuoteRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, response{Error: "invalid json body"})
		return
	}
	quote, err := h.service.Quote(r.Context(), req)
	h.writeResult(w, quote, err)
}

func (h *Handler) networkFee(w http.ResponseWriter, r *http.Request) {
	fee, err := h.service.GetNetworkFee(r.Context(), r.URL.Query().Get("chainType"))
	h.writeResult(w, fee, err)
}

func (h *Handler) createOrder(w http.ResponseWriter, r *http.Request) {
	var req CreateOrderInput
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, response{Error: "invalid json body"})
		return
	}
	order, err := h.service.CreateOrder(r.Context(), req)
	h.writeResult(w, order, err)
}

func (h *Handler) getOrder(w http.ResponseWriter, r *http.Request) {
	order, err := h.service.GetOrder(r.Context(), chi.URLParam(r, "orderID"))
	h.writeResult(w, order, err)
}

func (h *Handler) listOrders(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	orders, err := h.service.ListOrders(r.Context(), SearchOrdersInput{
		UserID: r.URL.Query().Get("userId"),
		Status: r.URL.Query().Get("status"),
		Limit:  limit,
	})
	h.writeResult(w, orders, err)
}

func (h *Handler) updateStatus(w http.ResponseWriter, r *http.Request) {
	var req UpdateOrderStatusInput
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, response{Error: "invalid json body"})
		return
	}
	order, err := h.service.UpdateOrderStatus(r.Context(), chi.URLParam(r, "orderID"), req)
	h.writeResult(w, order, err)
}

func (h *Handler) cancelOrder(w http.ResponseWriter, r *http.Request) {
	order, err := h.service.CancelOrder(r.Context(), chi.URLParam(r, "orderID"))
	h.writeResult(w, order, err)
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
