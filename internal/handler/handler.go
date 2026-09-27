package handler

import (
	"encoding/json"
	"net/http"

	api "github.com/Vllad3/tripgo-avito/internal/generated"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Handler struct {
	api.Unimplemented
	pool *pgxpool.Pool
}

func NewHandler(pool *pgxpool.Pool) *Handler {
	return &Handler{pool: pool}
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	if err := writeJSON(w, http.StatusOK, api.HealthResponse{Status: api.Ok}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	if err := h.pool.Ping(r.Context()); err != nil {
		if writeErr := writeJSON(w, http.StatusServiceUnavailable, api.HealthResponse{Status: api.Unavailable}); writeErr != nil {
			http.Error(w, writeErr.Error(), http.StatusInternalServerError)
		}
		return
	}

	if err := writeJSON(w, http.StatusOK, api.HealthResponse{Status: api.Ok}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func writeJSON(w http.ResponseWriter, statusCode int, data any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	return json.NewEncoder(w).Encode(data)
}
