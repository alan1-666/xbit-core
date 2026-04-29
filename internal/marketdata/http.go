package marketdata

import (
	"errors"
	"net/http"
	"strconv"
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
	router.Post("/graphql", h.graphql)
	router.Post("/meme-gql", h.graphql)
	router.Post("/api/meme2/meme-gql", h.graphql)
	router.Post("/api/meme/graphql", h.graphql)

	router.Route("/v1/market", func(r chi.Router) {
		r.Get("/tokens", h.listTokens)
		r.Get("/tokens/search", h.searchTokens)
		r.Post("/tokens", h.upsertToken)
		r.Get("/tokens/{chainID}/{address}", h.getToken)
		r.Get("/tokens/{chainID}/{address}/ohlc", h.ohlc)
		r.Get("/tokens/{chainID}/{address}/transactions", h.transactions)
		r.Get("/tokens/{chainID}/{address}/pools", h.pools)
		r.Get("/categories", h.categories)
	})

	router.Route("/v1/indexer", func(r chi.Router) {
		r.Post("/tokens", h.upsertToken)
		r.Post("/transactions", h.appendTransaction)
		r.Get("/checkpoints/{source}", h.getCheckpoint)
		r.Put("/checkpoints/{source}", h.saveCheckpoint)
	})
}

func (h *Handler) listTokens(w http.ResponseWriter, r *http.Request) {
	list, err := h.service.ListTokens(r.Context(), TokenFilter{
		Query:    r.URL.Query().Get("q"),
		Category: r.URL.Query().Get("category"),
		ChainID:  intQuery(r, "chainId"),
		Page:     intQuery(r, "page"),
		Limit:    intQuery(r, "limit"),
	})
	h.writeResult(w, list, err)
}

func (h *Handler) searchTokens(w http.ResponseWriter, r *http.Request) {
	tokens, err := h.service.SearchTokens(r.Context(), r.URL.Query().Get("q"), intQuery(r, "limit"))
	h.writeResult(w, tokens, err)
}

func (h *Handler) getToken(w http.ResponseWriter, r *http.Request) {
	token, err := h.service.GetToken(r.Context(), intPath(r, "chainID"), chi.URLParam(r, "address"))
	h.writeResult(w, token, err)
}

func (h *Handler) ohlc(w http.ResponseWriter, r *http.Request) {
	points, err := h.service.OHLC(r.Context(), intPath(r, "chainID"), chi.URLParam(r, "address"), r.URL.Query().Get("bucket"), intQuery(r, "limit"))
	h.writeResult(w, points, err)
}

func (h *Handler) transactions(w http.ResponseWriter, r *http.Request) {
	txs, err := h.service.Transactions(r.Context(), intPath(r, "chainID"), chi.URLParam(r, "address"), intQuery(r, "limit"))
	h.writeResult(w, map[string]any{"data": txs}, err)
}

func (h *Handler) pools(w http.ResponseWriter, r *http.Request) {
	pools, err := h.service.Pools(r.Context(), intPath(r, "chainID"), chi.URLParam(r, "address"))
	h.writeResult(w, pools, err)
}

func (h *Handler) categories(w http.ResponseWriter, r *http.Request) {
	categories, err := h.service.Categories(r.Context(), intQuery(r, "limit"))
	h.writeResult(w, categories, err)
}

func (h *Handler) upsertToken(w http.ResponseWriter, r *http.Request) {
	var req UpsertTokenInput
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, response{Error: "invalid json body"})
		return
	}
	token, err := h.service.UpsertToken(r.Context(), req)
	h.writeResult(w, token, err)
}

func (h *Handler) appendTransaction(w http.ResponseWriter, r *http.Request) {
	var req AppendTransactionInput
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, response{Error: "invalid json body"})
		return
	}
	tx, err := h.service.AppendTransaction(r.Context(), req)
	h.writeResult(w, tx, err)
}

func (h *Handler) saveCheckpoint(w http.ResponseWriter, r *http.Request) {
	var req Checkpoint
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, response{Error: "invalid json body"})
		return
	}
	if strings.TrimSpace(req.Source) == "" {
		req.Source = chi.URLParam(r, "source")
	}
	checkpoint, err := h.service.SaveCheckpoint(r.Context(), req)
	h.writeResult(w, checkpoint, err)
}

func (h *Handler) getCheckpoint(w http.ResponseWriter, r *http.Request) {
	checkpoint, err := h.service.GetCheckpoint(r.Context(), chi.URLParam(r, "source"))
	h.writeResult(w, checkpoint, err)
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

func intQuery(r *http.Request, key string) int {
	value, _ := strconv.Atoi(strings.TrimSpace(r.URL.Query().Get(key)))
	return value
}

func intPath(r *http.Request, key string) int {
	value, _ := strconv.Atoi(strings.TrimSpace(chi.URLParam(r, key)))
	return value
}
