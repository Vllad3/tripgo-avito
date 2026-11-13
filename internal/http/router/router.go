package router

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	api "github.com/Vllad3/tripgo-avito/internal/generated"
	"github.com/Vllad3/tripgo-avito/internal/http/domainerror"
	"github.com/Vllad3/tripgo-avito/internal/http/middleware"
)

type Options struct {
	Log             *slog.Logger
	TracerProvider  trace.TracerProvider
	RequestDuration metric.Float64Histogram
	ActiveRequests  metric.Int64UpDownCounter
}

func New(
	create createHandler,
	get getHandler,
	finish finishHandler,
	health healthHandler,
	opts Options,
) (http.Handler, error) {
	spec, err := api.GetSpec()
	if err != nil {
		return nil, fmt.Errorf("load openapi spec: %w", err)
	}

	validate, err := middleware.ValidateRequest(spec)
	if err != nil {
		return nil, fmt.Errorf("build request validator: %w", err)
	}

	r := chi.NewRouter()
	resolver := routeResolver(r)

	r.Use(
		middleware.Recover(opts.Log),
		middleware.Tracing(opts.TracerProvider, resolver),
		middleware.Logging(opts.Log, resolver),
		middleware.Metrics(opts.RequestDuration, opts.ActiveRequests, resolver),
		validate,
	)

	api.HandlerWithOptions(
		&server{create: create, get: get, finish: finish, health: health},
		api.ChiServerOptions{
			BaseRouter:       r,
			ErrorHandlerFunc: invalidParam,
		},
	)

	return r, nil
}

func routeResolver(mux *chi.Mux) func(*http.Request) string {
	return func(r *http.Request) string {
		rctx := chi.NewRouteContext()
		if !mux.Match(rctx, r.Method, r.URL.Path) {
			return "unmatched"
		}
		return rctx.RoutePattern()
	}
}

func invalidParam(w http.ResponseWriter, r *http.Request, _ error) {
	status, body := domainerror.InvalidRequest("Invalid path or header parameter")
	domainerror.Write(w, r, status, body)
}

type server struct {
	create createHandler
	get    getHandler
	finish finishHandler
	health healthHandler
}

var _ api.ServerInterface = (*server)(nil)

func (s *server) CreateTrip(w http.ResponseWriter, r *http.Request, _ api.CreateTripParams) {
	s.create.Handle(w, r)
}

func (s *server) GetTrip(w http.ResponseWriter, r *http.Request, id api.TripId) {
	s.get.Handle(w, r, id)
}

func (s *server) FinishTrip(w http.ResponseWriter, r *http.Request, id api.TripId) {
	s.finish.Handle(w, r, id)
}

func (s *server) Health(w http.ResponseWriter, r *http.Request) {
	s.health.Health(w, r)
}

func (s *server) Ready(w http.ResponseWriter, r *http.Request) {
	s.health.Ready(w, r)
}
