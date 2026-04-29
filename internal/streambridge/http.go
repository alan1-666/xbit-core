package streambridge

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
	router.Route("/v1/stream", func(r chi.Router) {
		r.Get("/topics", h.topics)
		r.Get("/events", h.recent)
		r.Post("/events", h.publish)
		r.Post("/events/batch", h.publishBatch)
	})
}

func (h *Handler) publish(w http.ResponseWriter, r *http.Request) {
	var req Event
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, response{Error: "invalid json body"})
		return
	}
	result, err := h.service.Publish(r.Context(), req)
	h.writeResult(w, result, err)
}

func (h *Handler) publishBatch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Events []Event `json:"events"`
	}
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, response{Error: "invalid json body"})
		return
	}
	result, err := h.service.PublishBatch(r.Context(), req.Events)
	h.writeResult(w, result, err)
}

func (h *Handler) recent(w http.ResponseWriter, r *http.Request) {
	topic := r.URL.Query().Get("topic")
	if topic == "" {
		httpx.WriteJSON(w, http.StatusBadRequest, response{Error: "topic is required"})
		return
	}
	h.writeResult(w, h.service.Recent(topic, intQuery(r, "limit")), nil)
}

func (h *Handler) topics(w http.ResponseWriter, _ *http.Request) {
	h.writeResult(w, h.service.Topics(), nil)
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
