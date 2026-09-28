package httpserver

import (
	"net/http"
	"time"

	api "github.com/Vllad3/tripgo-avito/internal/generated"
	"github.com/Vllad3/tripgo-avito/internal/handler"
	"github.com/go-chi/chi/v5"
)

type Timeouts struct {
	ReadHeader time.Duration
	Read       time.Duration
	Write      time.Duration
	Idle       time.Duration
}

func NewServer(addr string, t Timeouts, h api.ServerInterface) *http.Server {
	r := chi.NewRouter()
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
	}
}
