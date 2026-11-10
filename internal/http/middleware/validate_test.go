package middleware_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	api "github.com/Vllad3/tripgo-avito/internal/generated"
	"github.com/Vllad3/tripgo-avito/internal/http/middleware"
)

const createPath = "/api/v1/trips"

type probe struct {
	called bool
	body   string
}

func (p *probe) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p.called = true
	b, _ := io.ReadAll(r.Body)
	p.body = string(b)
	w.WriteHeader(http.StatusOK)
}

func newValidator(t *testing.T) func(http.Handler) http.Handler {
	t.Helper()

	spec, err := api.GetSpec()
	if err != nil {
		t.Fatalf("load spec: %v", err)
	}
	validate, err := middleware.ValidateRequest(spec)
	if err != nil {
		t.Fatalf("build validator: %v", err)
	}
	return validate
}

func TestValidateRequest(t *testing.T) {
	const (
		userID   = `"user_id": "5cb72c04-7650-45c9-a79b-bcdba0631e0c"`
		driverID = `"driver_id": "8860b315-ec86-42eb-a17c-7c163d721ff5"`
		start    = `"start_point": {"latitude": 59.94, "longitude": 30.31}`
		end      = `"end_point": {"latitude": 59.93, "longitude": 30.36}`
	)

	tests := []struct {
		name       string
		method     string
		path       string
		body       string
		wantStatus int
		wantNext   bool
	}{
		{
			name: "valid create", method: http.MethodPost, path: createPath,
			body:       `{` + userID + `,` + driverID + `,` + start + `,` + end + `, "price": 1450}`,
			wantStatus: http.StatusOK, wantNext: true,
		},
		{
			name: "missing price", method: http.MethodPost, path: createPath,
			body:       `{` + userID + `,` + driverID + `,` + start + `,` + end + `}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "null price", method: http.MethodPost, path: createPath,
			body:       `{` + userID + `,` + driverID + `,` + start + `,` + end + `, "price": null}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "missing nested latitude", method: http.MethodPost, path: createPath,
			body:       `{` + userID + `,` + driverID + `, "start_point": {"longitude": 30.31},` + end + `, "price": 1450}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "get passes through", method: http.MethodGet,
			path:       "/api/v1/trips/1f0a9c62-4a1c-4f2e-9d33-2a4bb0f0b111",
			wantStatus: http.StatusOK, wantNext: true,
		},
		{
			name: "path not in contract passes through", method: http.MethodGet, path: "/metrics",
			wantStatus: http.StatusOK, wantNext: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			next := &probe{}
			h := newValidator(t)(next)

			req := httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d; body: %s", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if next.called != tt.wantNext {
				t.Errorf("next called = %v, want %v", next.called, tt.wantNext)
			}

			if tt.wantStatus == http.StatusBadRequest {
				var p api.Problem
				if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
					t.Fatalf("decode problem: %v", err)
				}
				if p.Code != "invalid_request" {
					t.Errorf("code = %q, want invalid_request", p.Code)
				}
			}
		})
	}
}

func TestValidateRequest_BodyStaysReadable(t *testing.T) {
	body := `{"user_id": "5cb72c04-7650-45c9-a79b-bcdba0631e0c", "driver_id": "8860b315-ec86-42eb-a17c-7c163d721ff5",
		"start_point": {"latitude": 59.94, "longitude": 30.31}, "end_point": {"latitude": 59.93, "longitude": 30.36}, "price": 1450}`
	next := &probe{}

	req := httptest.NewRequest(http.MethodPost, createPath, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	newValidator(t)(next).ServeHTTP(httptest.NewRecorder(), req)

	if !next.called {
		t.Fatal("request must reach the handler")
	}
	if next.body != body {
		t.Errorf("handler got body %q, want the original body", next.body)
	}
}
