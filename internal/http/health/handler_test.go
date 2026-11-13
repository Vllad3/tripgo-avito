package health_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	api "github.com/Vllad3/tripgo-avito/internal/generated"
	"github.com/Vllad3/tripgo-avito/internal/http/health"
)

func newHandler(t *testing.T) (*health.Handler, *Mockpinger) {
	t.Helper()
	db := NewMockpinger(gomock.NewController(t))
	return health.New(db, time.Second), db
}

func status(t *testing.T, rec *httptest.ResponseRecorder) api.HealthResponseStatus {
	t.Helper()

	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("content-type = %q, want application/json", ct)
	}
	var body api.HealthResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	return body.Status
}

func TestHandler_Health_DoesNotTouchDatabase(t *testing.T) {
	h, _ := newHandler(t)
	rec := httptest.NewRecorder()
	h.Health(rec, httptest.NewRequest(http.MethodGet, "/health", nil))

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	if got := status(t, rec); got != api.Ok {
		t.Errorf("status field = %q, want ok", got)
	}
}

func TestHandler_Ready_DatabaseUp(t *testing.T) {
	h, db := newHandler(t)
	db.EXPECT().Ping(gomock.Any()).DoAndReturn(func(ctx context.Context) error {
		if _, ok := ctx.Deadline(); !ok {
			t.Error("ping context must have a deadline")
		}
		return nil
	})

	rec := httptest.NewRecorder()
	h.Ready(rec, httptest.NewRequest(http.MethodGet, "/ready", nil))

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	if got := status(t, rec); got != api.Ok {
		t.Errorf("status field = %q, want ok", got)
	}
}

func TestHandler_Ready_DatabaseDown(t *testing.T) {
	h, db := newHandler(t)
	db.EXPECT().Ping(gomock.Any()).Return(errors.New("connection refused: 10.0.0.5:5432"))

	rec := httptest.NewRecorder()
	h.Ready(rec, httptest.NewRequest(http.MethodGet, "/ready", nil))

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", rec.Code)
	}
	if got := status(t, rec); got != api.Unavailable {
		t.Errorf("status field = %q, want unavailable", got)
	}
	if strings.Contains(rec.Body.String(), "10.0.0.5") {
		t.Errorf("body leaks internal error: %s", rec.Body.String())
	}
}
