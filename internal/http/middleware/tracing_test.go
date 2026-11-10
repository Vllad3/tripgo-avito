package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"

	"github.com/Vllad3/tripgo-avito/internal/http/middleware"
)

const routeTemplate = "/api/v1/trips/{tripId}"

func templateRoute(*http.Request) string { return routeTemplate }

func newSpanRecorder(t *testing.T) (*sdktrace.TracerProvider, *tracetest.InMemoryExporter) {
	t.Helper()

	exp := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exp))
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })

	return tp, exp
}

func attrValue(span tracetest.SpanStub, key string) (attribute.Value, bool) {
	for _, kv := range span.Attributes {
		if string(kv.Key) == key {
			return kv.Value, true
		}
	}
	return attribute.Value{}, false
}

func traced(tp trace.TracerProvider, h http.HandlerFunc) http.Handler {
	return middleware.Tracing(tp, templateRoute)(h)
}

func TestTracing_SpanIsNamedByRouteTemplate(t *testing.T) {
	tp, exp := newSpanRecorder(t)
	h := traced(tp, func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/v1/trips/1f0a9c62", nil))

	spans := exp.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("spans = %d, want 1", len(spans))
	}
	span := spans[0]

	if want := "GET " + routeTemplate; span.Name != want {
		t.Errorf("name = %q, want %q", span.Name, want)
	}
	if span.SpanKind != trace.SpanKindServer {
		t.Errorf("kind = %v, want server", span.SpanKind)
	}
	if v, ok := attrValue(span, "http.route"); !ok || v.AsString() != routeTemplate {
		t.Errorf("http.route = %v, want %q", v, routeTemplate)
	}
	if v, ok := attrValue(span, "http.response.status_code"); !ok || v.AsInt64() != http.StatusOK {
		t.Errorf("status code attribute = %v, want 200", v)
	}
	if span.Status.Code != codes.Unset {
		t.Errorf("status = %v, want unset", span.Status.Code)
	}
}

func TestTracing_OnlyServerErrorsMarkSpan(t *testing.T) {
	tests := []struct {
		name   string
		status int
		want   codes.Code
	}{
		{"not found is not an error", http.StatusNotFound, codes.Unset},
		{"conflict is not an error", http.StatusConflict, codes.Unset},
		{"server error is an error", http.StatusInternalServerError, codes.Error},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tp, exp := newSpanRecorder(t)
			h := traced(tp, func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(tt.status) })

			h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/x", nil))

			spans := exp.GetSpans()
			if len(spans) != 1 {
				t.Fatalf("spans = %d, want 1", len(spans))
			}
			if spans[0].Status.Code != tt.want {
				t.Errorf("status = %v, want %v", spans[0].Status.Code, tt.want)
			}
			if _, has := attrValue(spans[0], "error.type"); has != (tt.want == codes.Error) {
				t.Errorf("error.type present = %v, want %v", has, tt.want == codes.Error)
			}
		})
	}
}

func TestTracing_ContinuesIncomingTrace(t *testing.T) {
	tp, exp := newSpanRecorder(t)
	h := traced(tp, func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
	h.ServeHTTP(httptest.NewRecorder(), req)

	spans := exp.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("spans = %d, want 1", len(spans))
	}
	if got := spans[0].SpanContext.TraceID().String(); got != "4bf92f3577b34da6a3ce929d0e0e4736" {
		t.Errorf("trace id = %s, want the one from traceparent", got)
	}
	if got := spans[0].Parent.SpanID().String(); got != "00f067aa0ba902b7" {
		t.Errorf("parent span id = %s, want the one from traceparent", got)
	}
}

func TestTracing_PanicIsRecordedAndRethrown(t *testing.T) {
	tp, exp := newSpanRecorder(t)
	h := traced(tp, func(http.ResponseWriter, *http.Request) { panic("boom") })

	func() {
		defer func() {
			if recover() == nil {
				t.Error("panic must be re-thrown so Recover can answer the client")
			}
		}()
		h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/x", nil))
	}()

	spans := exp.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("spans = %d, want 1: запрос с паникой не должен пропадать из трейсов", len(spans))
	}
	if v, ok := attrValue(spans[0], "http.response.status_code"); !ok || v.AsInt64() != http.StatusInternalServerError {
		t.Errorf("status code attribute = %v, want 500", v)
	}
	if spans[0].Status.Code != codes.Error {
		t.Errorf("status = %v, want error", spans[0].Status.Code)
	}
}
