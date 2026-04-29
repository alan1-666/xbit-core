package hypertrader

import (
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
	router.Post("/api/graphql-dex", h.graphql)
	router.Post("/api/dex-hypertrader/graphql", h.graphql)
	router.Post("/api/dex-hypertrader/graphql/", h.graphql)
	router.Post("/api/user/user-gql", h.graphql)

	router.Route("/v1/futures", func(r chi.Router) {
		r.Get("/symbols", h.symbols)
		r.Get("/account", h.account)
		r.Get("/trades", h.trades)
		r.Get("/smart-money", h.smartMoney)
		r.Get("/funding-rates", h.fundingRates)
		r.Get("/orders", h.orders)
		r.Post("/orders", h.createOrder)
		r.Post("/orders/{orderId}/cancel", h.cancelOrder)
		r.Post("/leverage", h.updateLeverage)
		r.Get("/audit-events", h.auditEvents)
	})
}

func (h *Handler) symbols(w http.ResponseWriter, r *http.Request) {
	symbols, err := h.service.ListSymbols(r.Context(), r.URL.Query().Get("q"), r.URL.Query().Get("category"), intQuery(r, "limit"))
	h.writeResult(w, symbols, err)
}

func (h *Handler) account(w http.ResponseWriter, r *http.Request) {
	account, err := h.service.Account(r.Context(), r.URL.Query().Get("userAddress"))
	h.writeResult(w, account, err)
}

func (h *Handler) trades(w http.ResponseWriter, r *http.Request) {
	h.writeResult(w, h.service.TradeHistory(r.Context()), nil)
}

func (h *Handler) smartMoney(w http.ResponseWriter, r *http.Request) {
	traders, err := h.service.SmartMoney(r.Context(), intQuery(r, "limit"))
	h.writeResult(w, traders, err)
}

func (h *Handler) fundingRates(w http.ResponseWriter, r *http.Request) {
	rates, err := h.service.FundingRates(r.Context(), r.URL.Query().Get("symbol"), intQuery(r, "limit"))
	h.writeResult(w, rates, err)
}

func (h *Handler) orders(w http.ResponseWriter, r *http.Request) {
	orders, err := h.service.Orders(r.Context(), OrderFilter{
		UserID:      r.URL.Query().Get("userId"),
		UserAddress: r.URL.Query().Get("userAddress"),
		Status:      r.URL.Query().Get("status"),
		Symbol:      r.URL.Query().Get("symbol"),
		Limit:       intQuery(r, "limit"),
	})
	h.writeResult(w, orders, err)
}

func (h *Handler) createOrder(w http.ResponseWriter, r *http.Request) {
	var input CreateOrderInput
	if err := httpx.DecodeJSON(r, &input); err != nil {
		h.writeResult(w, nil, err)
		return
	}
	order, err := h.service.CreateOrder(r.Context(), input)
	h.writeResult(w, order, err)
}

func (h *Handler) cancelOrder(w http.ResponseWriter, r *http.Request) {
	var input CancelOrderInput
	if err := httpx.DecodeJSON(r, &input); err != nil {
		h.writeResult(w, nil, err)
		return
	}
	input.OrderID = firstNonEmpty(input.OrderID, chi.URLParam(r, "orderId"))
	order, err := h.service.CancelOrder(r.Context(), input)
	h.writeResult(w, order, err)
}

func (h *Handler) updateLeverage(w http.ResponseWriter, r *http.Request) {
	var input UpdateLeverageInput
	if err := httpx.DecodeJSON(r, &input); err != nil {
		h.writeResult(w, nil, err)
		return
	}
	result, err := h.service.UpdateLeverage(r.Context(), input)
	h.writeResult(w, result, err)
}

func (h *Handler) auditEvents(w http.ResponseWriter, r *http.Request) {
	events, err := h.service.AuditEvents(r.Context(), r.URL.Query().Get("userId"), intQuery(r, "limit"))
	h.writeResult(w, events, err)
}

func (h *Handler) writeResult(w http.ResponseWriter, data any, err error) {
	if err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, response{Error: err.Error()})
		return
	}
	httpx.WriteJSON(w, http.StatusOK, response{Data: data})
}

func intQuery(r *http.Request, key string) int {
	value, _ := strconv.Atoi(r.URL.Query().Get(key))
	return value
}
