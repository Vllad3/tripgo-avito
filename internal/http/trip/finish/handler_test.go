package finish_test

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	"github.com/Vllad3/tripgo-avito/internal/domain/trip"
	api "github.com/Vllad3/tripgo-avito/internal/generated"
	"github.com/Vllad3/tripgo-avito/internal/http/domainerror"
	"github.com/Vllad3/tripgo-avito/internal/http/trip/finish"
)

var tripID = uuid.MustParse("1f0a9c62-4a1c-4f2e-9d33-2a4bb0f0b111")

func newHandler(t *testing.T) (*finish.Handler, *MockuseCase) {
	t.Helper()
	uc := NewMockuseCase(gomock.NewController(t))
	errs := domainerror.NewWriter(slog.New(slog.NewTextHandler(io.Discard, nil)))
	return finish.New(uc, errs), uc
}

func call(h *finish.Handler) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/trips/"+tripID.String()+"/finish", nil)
	rec := httptest.NewRecorder()
	h.Handle(rec, req, tripID)
	return rec
}

func TestHandler_Success(t *testing.T) {
	h, uc := newHandler(t)
	startedAt := time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)
	finishedAt := startedAt.Add(18 * time.Minute)
	uc.EXPECT().Execute(gomock.Any(), tripID).Return(&trip.Trip{
		ID:         tripID,
		Status:     trip.StatusCompleted,
		StartedAt:  startedAt,
		FinishedAt: &finishedAt,
	}, nil)

	rec := call(h)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var got api.Trip
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode trip: %v", err)
	}
	if got.Status != api.TripStatus(trip.StatusCompleted) {
		t.Errorf("status = %q, want completed", got.Status)
	}
	if got.FinishedAt == nil || !got.FinishedAt.Equal(finishedAt) {
		t.Errorf("finished_at = %v, want %v", got.FinishedAt, finishedAt)
	}
}

func TestHandler_Errors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{"not found", trip.ErrTripNotFound, http.StatusNotFound, "trip_not_found"},
		{"already completed", trip.ErrTripCompleted, http.StatusConflict, "trip_completed"},
		{"internal", errors.New("pq: password authentication failed"), http.StatusInternalServerError, "internal_error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, uc := newHandler(t)
			uc.EXPECT().Execute(gomock.Any(), tripID).Return(nil, tt.err)

			rec := call(h)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			var p api.Problem
			if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
				t.Fatalf("decode problem: %v", err)
			}
			if p.Code != tt.wantCode {
				t.Errorf("code = %q, want %q", p.Code, tt.wantCode)
			}
			if strings.Contains(rec.Body.String(), "password") {
				t.Errorf("body leaks internal error: %s", rec.Body.String())
			}
		})
	}
}
