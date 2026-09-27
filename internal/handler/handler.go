package handler

import (
	"encoding/json"
	"net/http"

	api "github.com/Vllad3/tripgo-avito/internal/generated"
)

type Handler struct {
	api.Unimplemented
}

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	if err := writeJSON(w, http.StatusOK, api.HealthResponse{Status: api.Ok}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	// еще подкрутить unavailable, как появится бд
	if err := writeJSON(w, http.StatusOK, api.HealthResponse{Status: api.Ok}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func writeJSON(w http.ResponseWriter, statusCode int, data any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	return json.NewEncoder(w).Encode(data)
}
