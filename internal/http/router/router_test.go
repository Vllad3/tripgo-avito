package router_test

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	api "github.com/Vllad3/tripgo-avito/internal/generated"
	"github.com/Vllad3/tripgo-avito/internal/http/router"
)

const tripPath = "/api/v1/trips/1f0a9c62-4a1c-4f2e-9d33-2a4bb0f0b111"

var tripID = uuid.MustParse("1f0a9c62-4a1c-4f2e-9d33-2a4bb0f0b111")

type fakeCreate struct{ calls int }

func (f *fakeCreate) Handle(w http.ResponseWriter, _ *http.Request) {
	f.calls++
	w.WriteHeader(http.StatusCreated)
}

type fakeTrip struct {
	calls     int
	gotID     uuid.UUID
	panicWith any
}

func (f *fakeTrip) Handle(w http.ResponseWriter, _ *http.Request, id uuid.UUID) {
	f.calls++
	f.gotID = id
	if f.panicWith != nil {
		panic(f.panicWith)
	}
	w.WriteHeader(http.StatusOK)
}

type fakeHealth struct{ health, ready int }

func (f *fakeHealth) Health(w http.ResponseWriter, _ *http.Request) {
	f.health++
	w.WriteHeader(http.StatusOK)
}

func (f *fakeHealth) Ready(w http.ResponseWriter, _ *http.Request) {
	f.ready++
	w.WriteHeader(http.StatusOK)
}

type env struct {
	h      http.Handler
	create *fakeCreate
	get    *fakeTrip
	finish *fakeTrip
	health *fakeHealth
}

func newEnv(t *testing.T) *env {
	t.Helper()

	e := &env{create: &fakeCreate{}, get: &fakeTrip{}, finish: &fakeTrip{}, health: &fakeHealth{}}
	h, err := router.New(e.create, e.get, e.finish, e.health, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("build router: %v", err)
	}
	e.h = h
	return e
}

func (e *env) do(method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.h.ServeHTTP(rec, req)
	return rec
}

func assertProblem(t *testing.T, rec *httptest.ResponseRecorder, status int, code string) {
	t.Helper()

	if rec.Code != status {
		t.Errorf("status = %d, want %d; body: %s", rec.Code, status, rec.Body.String())
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

const validBody = `{
	"user_id": "5cb72c04-7650-45c9-a79b-bcdba0631e0c",
	"driver_id": "8860b315-ec86-42eb-a17c-7c163d721ff5",
	"start_point": {"latitude": 59.94, "longitude": 30.31},
	"end_point": {"latitude": 59.93, "longitude": 30.36},
	"price": 1450
}`

func TestRouter_DispatchesToHandlers(t *testing.T) {
	e := newEnv(t)

	if rec := e.do(http.MethodPost, "/api/v1/trips", validBody); rec.Code != http.StatusCreated || e.create.calls != 1 {
		t.Errorf("create: status %d, calls %d", rec.Code, e.create.calls)
	}
	if rec := e.do(http.MethodGet, tripPath, ""); rec.Code != http.StatusOK || e.get.gotID != tripID {
		t.Errorf("get: status %d, id %v", rec.Code, e.get.gotID)
	}
	if rec := e.do(http.MethodPost, tripPath+"/finish", ""); rec.Code != http.StatusOK || e.finish.gotID != tripID {
		t.Errorf("finish: status %d, id %v", rec.Code, e.finish.gotID)
	}
	if rec := e.do(http.MethodGet, "/health", ""); rec.Code != http.StatusOK || e.health.health != 1 {
		t.Errorf("health: status %d, calls %d", rec.Code, e.health.health)
	}
	if rec := e.do(http.MethodGet, "/ready", ""); rec.Code != http.StatusOK || e.health.ready != 1 {
		t.Errorf("ready: status %d, calls %d", rec.Code, e.health.ready)
	}
}

func TestRouter_InvalidTripID(t *testing.T) {
	e := newEnv(t)

	assertProblem(t, e.do(http.MethodGet, "/api/v1/trips/not-a-uuid", ""), http.StatusBadRequest, "invalid_request")
	assertProblem(t, e.do(http.MethodPost, "/api/v1/trips/not-a-uuid/finish", ""), http.StatusBadRequest, "invalid_request")

	if e.get.calls != 0 || e.finish.calls != 0 {
		t.Errorf("handlers called %d/%d times, want 0: до них запрос не должен доходить", e.get.calls, e.finish.calls)
	}
}

func TestRouter_MissingRequiredFieldNeverReachesHandler(t *testing.T) {
	e := newEnv(t)

	rec := e.do(http.MethodPost, "/api/v1/trips", `{"user_id": "5cb72c04-7650-45c9-a79b-bcdba0631e0c"}`)

	assertProblem(t, rec, http.StatusBadRequest, "invalid_request")
	if e.create.calls != 0 {
		t.Errorf("create called %d times, want 0", e.create.calls)
	}
}

func TestRouter_UnknownPath(t *testing.T) {
	e := newEnv(t)

	if rec := e.do(http.MethodGet, "/api/v1/unknown", ""); rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

func TestRouter_PanicInHandlerDoesNotKillService(t *testing.T) {
	e := newEnv(t)
	e.get.panicWith = "boom"

	assertProblem(t, e.do(http.MethodGet, tripPath, ""), http.StatusInternalServerError, "internal_error")

	if rec := e.do(http.MethodGet, "/health", ""); rec.Code != http.StatusOK {
		t.Errorf("health after panic: status = %d, want 200", rec.Code)
	}
}
