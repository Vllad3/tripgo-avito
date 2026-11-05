package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/Vllad3/tripgo-avito/internal/database"
	api "github.com/Vllad3/tripgo-avito/internal/generated"
	"github.com/Vllad3/tripgo-avito/internal/repository"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Handler struct {
	api.Unimplemented
	pool         *pgxpool.Pool
	tx           *database.TxManager
	trips        repository.TripRepository
	history      repository.TripStatusHistoryRepository
	queryTimeout time.Duration
}

func NewHandler(pool *pgxpool.Pool, tx *database.TxManager, trips repository.TripRepository, history repository.TripStatusHistoryRepository, queryTimeout time.Duration) *Handler {
	return &Handler{pool: pool, tx: tx, trips: trips, history: history, queryTimeout: queryTimeout}
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	if err := writeJSON(w, http.StatusOK, api.HealthResponse{Status: api.Ok}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), h.queryTimeout)
	defer cancel()

	if err := h.pool.Ping(ctx); err != nil {
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
