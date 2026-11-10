package domainerror

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/Vllad3/tripgo-avito/internal/domain/trip"
	api "github.com/Vllad3/tripgo-avito/internal/generated"
)

const problemTypeBase = "https://tripgo.example/problems/"

func Problem(err error) (int, api.Problem) {
	var validationErr *trip.ValidationError

	switch {
	case errors.As(err, &validationErr):
		return newProblem(http.StatusBadRequest, "invalid_request", "Invalid request",
			validationErr.Field+": "+validationErr.Message)
	case errors.Is(err, trip.ErrTripNotFound):
		return newProblem(http.StatusNotFound, "trip_not_found", "Trip not found", "Trip not found")
	case errors.Is(err, trip.ErrTripCompleted):
		return newProblem(http.StatusConflict, "trip_completed", "Trip completed", "Trip is already completed")
	case errors.Is(err, trip.ErrDriverBusy):
		return newProblem(http.StatusConflict, "driver_busy", "Driver busy", "Driver already has an active trip")
	default:
		return newProblem(http.StatusInternalServerError, "internal_error", "Internal error", "Internal server error")
	}
}

func Write(w http.ResponseWriter, r *http.Request, status int, body api.Problem) {
	if body.Instance == nil {
		instance := r.URL.Path
		body.Instance = &instance
	}

	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

type Writer struct {
	log *slog.Logger
}

func NewWriter(log *slog.Logger) *Writer {
	return &Writer{log: log}
}

func (e *Writer) Error(w http.ResponseWriter, r *http.Request, err error) {
	status, body := Problem(err)
	if status >= http.StatusInternalServerError {
		e.log.ErrorContext(r.Context(), "request failed", slog.Any("error", err))
	}
	Write(w, r, status, body)
}

func newProblem(status int32, code, title, detail string) (int, api.Problem) {
	return int(status), api.Problem{
		Type:   problemTypeBase + strings.ReplaceAll(code, "_", "-"),
		Title:  title,
		Status: status,
		Detail: &detail,
		Code:   code,
	}
}

func InvalidRequest(detail string) (int, api.Problem) {
	return newProblem(http.StatusBadRequest, "invalid_request", "Invalid request", detail)
}
