package create_test

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
	"github.com/Vllad3/tripgo-avito/internal/http/trip/create"
	createuc "github.com/Vllad3/tripgo-avito/internal/usecase/trip/create"
)

const validBody = `{
	"user_id": "5cb72c04-7650-45c9-a79b-bcdba0631e0c",
	"driver_id": "8860b315-ec86-42eb-a17c-7c163d721ff5",
	"start_point": {"latitude": 59.9398, "longitude": 30.3146},
	"end_point": {"latitude": 59.9290, "longitude": 30.3626},
	"price": 1450
}`

var (
	tripID   = uuid.MustParse("1f0a9c62-4a1c-4f2e-9d33-2a4bb0f0b111")
	userID   = uuid.MustParse("5cb72c04-7650-45c9-a79b-bcdba0631e0c")
	driverID = uuid.MustParse("8860b315-ec86-42eb-a17c-7c163d721ff5")
)

func wantInput() createuc.Input {
	return createuc.Input{
		UserID:   userID,
		DriverID: driverID,
		Start:    trip.Point{Latitude: 59.9398, Longitude: 30.3146},
		End:      trip.Point{Latitude: 59.9290, Longitude: 30.3626},
		Price:    1450,
	}
}

func newHandler(t *testing.T) (*create.Handler, *MockuseCase) {
	t.Helper()
	uc := NewMockuseCase(gomock.NewController(t))
	errs := domainerror.NewWriter(slog.New(slog.NewTextHandler(io.Discard, nil)))
	return create.New(uc, errs), uc
}

func post(h *create.Handler, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/trips", strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.Handle(rec, req)
	return rec
}

func assertProblem(t *testing.T, rec *httptest.ResponseRecorder, status int, code string) {
	t.Helper()

	if rec.Code != status {
		t.Errorf("status = %d, want %d", rec.Code, status)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/problem+json" {
		t.Errorf("content-type = %q, want application/problem+json", ct)
	}
	var p api.Problem
	if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
		t.Fatalf("decode problem: %v", err)
	}
	if p.Code != code {
		t.Errorf("code = %q, want %q", p.Code, code)
	}
}

func TestHandler_Success(t *testing.T) {
	h, uc := newHandler(t)
	created := &trip.Trip{
		ID: tripID, UserID: userID, DriverID: driverID,
		Start:     trip.Point{Latitude: 59.9398, Longitude: 30.3146},
		End:       trip.Point{Latitude: 59.9290, Longitude: 30.3626},
		Price:     1450,
		Status:    trip.StatusActive,
		StartedAt: time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC),
	}
	uc.EXPECT().Execute(gomock.Any(), wantInput()).Return(created, nil)

	rec := post(h, validBody)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/api/v1/trips/"+tripID.String() {
		t.Errorf("location = %q", loc)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("content-type = %q, want application/json", ct)
	}

	var got api.Trip
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode trip: %v", err)
	}
	if got.Id != tripID || got.Status != api.TripStatus(trip.StatusActive) || got.Price != 1450 {
		t.Errorf("trip = %+v", got)
	}
}

func TestHandler_InvalidBody(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"malformed json", `{"user_id": `},
		{"empty body", ``},
		{"unknown field", `{"user_id": "5cb72c04-7650-45c9-a79b-bcdba0631e0c", "foo": 1}`},
		{"not a uuid", `{"user_id": "not-a-uuid"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, _ := newHandler(t)

			assertProblem(t, post(h, tt.body), http.StatusBadRequest, "invalid_request")
		})
	}
}

func TestHandler_UseCaseErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{"validation", &trip.ValidationError{Field: "price", Message: "must be >= 0"}, http.StatusBadRequest, "invalid_request"},
		{"driver busy", trip.ErrDriverBusy, http.StatusConflict, "driver_busy"},
		{"internal", errors.New("pq: password authentication failed"), http.StatusInternalServerError, "internal_error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, uc := newHandler(t)
			uc.EXPECT().Execute(gomock.Any(), wantInput()).Return(nil, tt.err)

			rec := post(h, validBody)

			assertProblem(t, rec, tt.wantStatus, tt.wantCode)
			if strings.Contains(rec.Body.String(), "password") {
				t.Errorf("body leaks internal error: %s", rec.Body.String())
			}
		})
	}
}
