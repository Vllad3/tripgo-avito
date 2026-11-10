package domainerror_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Vllad3/tripgo-avito/internal/domain/trip"
	api "github.com/Vllad3/tripgo-avito/internal/generated"
	"github.com/Vllad3/tripgo-avito/internal/http/domainerror"
)

const secret = "pq: password authentication failed for user tripgo"

func TestProblem(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{"validation", &trip.ValidationError{Field: "price", Message: "must be >= 0"}, http.StatusBadRequest, "invalid_request"},
		{"validation wrapped", fmt.Errorf("create: %w", &trip.ValidationError{Field: "price", Message: "x"}), http.StatusBadRequest, "invalid_request"},
		{"not found", trip.ErrTripNotFound, http.StatusNotFound, "trip_not_found"},
		{"not found wrapped", fmt.Errorf("get trip: %w", trip.ErrTripNotFound), http.StatusNotFound, "trip_not_found"},
		{"completed", trip.ErrTripCompleted, http.StatusConflict, "trip_completed"},
		{"driver busy", trip.ErrDriverBusy, http.StatusConflict, "driver_busy"},
		{"unknown", errors.New(secret), http.StatusInternalServerError, "internal_error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, body := domainerror.Problem(tt.err)

			if status != tt.wantStatus || int(body.Status) != tt.wantStatus {
				t.Errorf("status = %d/%d, want %d", status, body.Status, tt.wantStatus)
			}
			if body.Code != tt.wantCode {
				t.Errorf("code = %q, want %q", body.Code, tt.wantCode)
			}
			if body.Detail != nil && strings.Contains(*body.Detail, "password") {
				t.Errorf("detail leaks internal error: %q", *body.Detail)
			}
		})
	}
}

func TestWriter_Error(t *testing.T) {
	w := domainerror.NewWriter(slog.New(slog.NewTextHandler(io.Discard, nil)))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/trips", nil)
	rec := httptest.NewRecorder()

	w.Error(rec, req, errors.New(secret))

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/problem+json" {
		t.Errorf("content-type = %q, want application/problem+json", ct)
	}
	if strings.Contains(rec.Body.String(), "password") {
		t.Errorf("body leaks internal error: %s", rec.Body.String())
	}

	var got api.Problem
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if got.Instance == nil || *got.Instance != "/api/v1/trips" {
		t.Errorf("instance = %v, want /api/v1/trips", got.Instance)
	}
}
