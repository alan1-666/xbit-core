package health

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

type Response struct {
	Service string `json:"service"`
	Status  string `json:"status"`
	Time    string `json:"time"`
}

func Register(router chi.Router, service string) {
	handler := func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Response{
			Service: service,
			Status:  "ok",
			Time:    time.Now().UTC().Format(time.RFC3339Nano),
		})
	}

	router.Get("/healthz", handler)
	router.Get("/readyz", handler)
}
