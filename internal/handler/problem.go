package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	api "github.com/Vllad3/tripgo-avito/internal/generated"
	"github.com/Vllad3/tripgo-avito/internal/repository"
)

func writeProblem(w http.ResponseWriter, r *http.Request, status int, code, title, detail string) {
	instance := r.URL.Path
	problem := api.Problem{
		Type:     "https://tripgo.example/problems/" + strings.ReplaceAll(code, "_", "-"),
		Title:    title,
		Status:   int32(status),
		Detail:   &detail,
		Instance: &instance,
		Code:     code,
	}

	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(problem)
}

func writeError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, repository.ErrDriverBusy):
		writeProblem(w, r, http.StatusConflict, "driver_busy", "Driver busy", "Driver already has an active trip")
	case errors.Is(err, repository.ErrTripNotFound):
		writeProblem(w, r, http.StatusNotFound, "trip_not_found", "Trip not found", "Trip not found")
	case errors.Is(err, repository.ErrTripAlreadyCompleted):
		writeProblem(w, r, http.StatusConflict, "trip_completed", "Trip completed", "Trip is already completed")
	default:
		log.Printf("internal error: %v", err)
		writeProblem(w, r, http.StatusInternalServerError, "internal_error", "Internal error", "Internal server error")
	}
}

func InvalidParam(w http.ResponseWriter, r *http.Request, err error) {
	writeProblem(w, r, http.StatusBadRequest, "invalid_request", "Invalid request", "Invalid path or header parameter")
}
