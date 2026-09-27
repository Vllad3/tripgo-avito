package httpserver

import (
	"net/http"
	"time"

	api "github.com/Vllad3/tripgo-avito/internal/generated"
	"github.com/go-chi/chi/v5"
)

const (
	readHeaderTimeout = 5 * time.Second
	readTimeout       = 10 * time.Second
	writeTimeout      = 15 * time.Second
	idleTimeout       = 60 * time.Second
)

func NewServer(addr string, h api.ServerInterface) *http.Server {
	r := chi.NewRouter()
	api.HandlerFromMux(h, r)

	return &http.Server{
		Addr:              addr,
		Handler:           r,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}
}
