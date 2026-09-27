package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"syscall"

	"github.com/Vllad3/tripgo-avito/internal/config"
	"github.com/Vllad3/tripgo-avito/internal/handler"
	"github.com/Vllad3/tripgo-avito/internal/httpserver"
)

func main() {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer stop()
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	h := handler.NewHandler()
	srv := httpserver.NewServer(cfg.HTTPAddr, h)

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	}
}
