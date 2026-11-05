package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"syscall"

	"github.com/Vllad3/tripgo-avito/internal/config"
	"github.com/Vllad3/tripgo-avito/internal/database"
	"github.com/Vllad3/tripgo-avito/internal/handler"
	"github.com/Vllad3/tripgo-avito/internal/httpserver"
	"github.com/Vllad3/tripgo-avito/internal/repository"
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

	dbPool, err := database.NewPool(ctx, cfg.DatabaseURL, int32(cfg.DatabaseMaxConns), int32(cfg.DatabaseMinConns), cfg.DatabaseMaxConnLifetime, cfg.DatabaseConnectTimeout)
	if err != nil {
		log.Fatalf("database: %v", err)
	}

	defer dbPool.Close()

	txManager := database.NewTxManager(dbPool, cfg.DatabaseQueryTimeout)

	trips := repository.NewTripRepository(dbPool, cfg.DatabaseQueryTimeout)
	history := repository.NewTripStatusHistoryRepository(dbPool, cfg.DatabaseQueryTimeout)

	h := handler.NewHandler(dbPool, txManager, trips, history, cfg.DatabaseQueryTimeout)
	srv, err := httpserver.NewServer(cfg.HTTPAddr, httpserver.Timeouts{
		ReadHeader: cfg.HTTPReadHeaderTimeout,
		Read:       cfg.HTTPReadTimeout,
		Write:      cfg.HTTPWriteTimeout,
		Idle:       cfg.HTTPIdleTimeout,
	}, h)

	if err != nil {
		log.Fatalf("http server: %v", err)
	}

	serverErr := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	select {
	case <-ctx.Done():
		log.Println("shutdown signal received")
	case err := <-serverErr:
		log.Printf("http server: %v", err)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	} else {
		log.Println("server stopped")
	}
}
