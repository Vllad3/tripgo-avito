package httpserver

import (
	"fmt"
	"net/http"
	"time"

	api "github.com/Vllad3/tripgo-avito/internal/generated"
	"github.com/Vllad3/tripgo-avito/internal/handler"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/go-chi/chi/v5"
)

type Timeouts struct {
	ReadHeader time.Duration
	Read       time.Duration
	Write      time.Duration
	Idle       time.Duration
}

func NewServer(addr string, t Timeouts, h api.ServerInterface) (*http.Server, error) {
	spec, err := api.GetSpec()
	if err != nil {
		return nil, fmt.Errorf("load openapi spec: %w", err)
	}
	spec.Servers = nil
	openapi3.SchemaErrorDetailsDisabled = true

	validate, err := requestValidator(spec)
	if err != nil {
		return nil, err
	}

	r := chi.NewRouter()
	r.Use(validate)
	api.HandlerWithOptions(h, api.ChiServerOptions{
		BaseRouter:       r,
		ErrorHandlerFunc: handler.InvalidParam,
	})

	return &http.Server{
		Addr:              addr,
		Handler:           r,
		ReadHeaderTimeout: t.ReadHeader,
		ReadTimeout:       t.Read,
		WriteTimeout:      t.Write,
		IdleTimeout:       t.Idle,
	}, nil
}
