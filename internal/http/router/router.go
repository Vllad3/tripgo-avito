package router

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	api "github.com/Vllad3/tripgo-avito/internal/generated"
	"github.com/Vllad3/tripgo-avito/internal/http/domainerror"
	"github.com/Vllad3/tripgo-avito/internal/http/middleware"
)

func New(
	create createHandler,
	get getHandler,
	finish finishHandler,
	health healthHandler,
	log *slog.Logger,
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
	r.Use(middleware.Recover(log), validate)

	api.HandlerWithOptions(&server{create: create, get: get, finish: finish, health: health}, api.ChiServerOptions{
		BaseRouter:       r,
		ErrorHandlerFunc: invalidParam,
	})

	return r, nil
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

func (s *server) GetTrip(w http.ResponseWriter, r *http.Request, tripID api.TripId) {
	s.get.Handle(w, r, tripID)
}

func (s *server) FinishTrip(w http.ResponseWriter, r *http.Request, tripID api.TripId) {
	s.finish.Handle(w, r, tripID)
}

func (s *server) Health(w http.ResponseWriter, r *http.Request) {
	s.health.Health(w, r)
}

func (s *server) Ready(w http.ResponseWriter, r *http.Request) {
	s.health.Ready(w, r)
}
