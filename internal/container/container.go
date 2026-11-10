package container

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Vllad3/tripgo-avito/internal/config"
	"github.com/Vllad3/tripgo-avito/internal/http/domainerror"
	"github.com/Vllad3/tripgo-avito/internal/http/health"
	"github.com/Vllad3/tripgo-avito/internal/http/router"
	"github.com/Vllad3/tripgo-avito/internal/http/server"
	createhttp "github.com/Vllad3/tripgo-avito/internal/http/trip/create"
	finishhttp "github.com/Vllad3/tripgo-avito/internal/http/trip/finish"
	gethttp "github.com/Vllad3/tripgo-avito/internal/http/trip/get"
	"github.com/Vllad3/tripgo-avito/internal/infra/postgres"
	"github.com/Vllad3/tripgo-avito/internal/infra/postgres/transaction"
	"github.com/Vllad3/tripgo-avito/internal/infra/telemetry"
	triprepo "github.com/Vllad3/tripgo-avito/internal/repository/trip"
	createuc "github.com/Vllad3/tripgo-avito/internal/usecase/trip/create"
	finishuc "github.com/Vllad3/tripgo-avito/internal/usecase/trip/finish"
	getuc "github.com/Vllad3/tripgo-avito/internal/usecase/trip/get"
)

// Container создаёт все зависимости и связывает их между собой.
// Это единственное место, где встречаются конкретные реализации.
type Container struct {
	log    *slog.Logger
	tel    *telemetry.Telemetry
	pool   *pgxpool.Pool
	server *http.Server
}

func New(ctx context.Context, cfg *config.Config, log *slog.Logger) (*Container, error) {
	tel, err := telemetry.Setup(ctx, log)
	if err != nil {
		return nil, fmt.Errorf("setup telemetry: %w", err)
	}

	c := &Container{log: log, tel: tel}

	if err := c.build(ctx, cfg); err != nil {
		closeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cfg.ShutdownTimeout)
		defer cancel()
		c.Close(closeCtx)
		return nil, err
	}

	return c, nil
}

func (c *Container) build(ctx context.Context, cfg *config.Config) error {
	pool, err := postgres.NewPool(ctx, postgres.Config{
		URL:             cfg.DatabaseURL,
		MaxConns:        cfg.DatabaseMaxConns,
		MinConns:        cfg.DatabaseMinConns,
		MaxConnLifetime: cfg.DatabaseMaxConnLifetime,
		ConnectTimeout:  cfg.DatabaseConnectTimeout,
	}, c.tel.TracerProvider(), c.tel.MeterProvider())
	if err != nil {
		return fmt.Errorf("connect to postgres: %w", err)
	}
	c.pool = pool

	httpMetrics, err := telemetry.NewHTTPMetrics(c.tel.MeterProvider())
	if err != nil {
		return fmt.Errorf("create http metrics: %w", err)
	}
	tripMetrics, err := telemetry.NewTripMetrics(c.tel.MeterProvider())
	if err != nil {
		return fmt.Errorf("create trip metrics: %w", err)
	}

	tx := transaction.New(pool, cfg.DatabaseQueryTimeout)
	repo := triprepo.New(pool, cfg.DatabaseQueryTimeout)
	errs := domainerror.NewWriter(c.log)

	createUC := createuc.New(repo, repo, tx, c.log, tripMetrics, time.Now, uuid.New)
	finishUC := finishuc.New(repo, repo, tx, c.log, tripMetrics, time.Now)
	getUC := getuc.New(repo)

	handler, err := router.New(
		createhttp.New(createUC, errs),
		gethttp.New(getUC, errs),
		finishhttp.New(finishUC, errs),
		health.New(pool, cfg.DatabaseQueryTimeout),
		router.Options{
			Log:             c.log,
			TracerProvider:  c.tel.TracerProvider(),
			RequestDuration: httpMetrics.RequestDuration,
			ActiveRequests:  httpMetrics.ActiveRequests,
		},
	)
	if err != nil {
		return fmt.Errorf("build router: %w", err)
	}

	c.server = server.New(cfg.HTTPAddr, server.Timeouts{
		ReadHeader: cfg.HTTPReadHeaderTimeout,
		Read:       cfg.HTTPReadTimeout,
		Write:      cfg.HTTPWriteTimeout,
		Idle:       cfg.HTTPIdleTimeout,
	}, handler)

	return nil
}

func (c *Container) Server() *http.Server {
	return c.server
}

func (c *Container) Close(ctx context.Context) {
	if c.pool != nil {
		c.pool.Close()
	}
	if err := c.tel.Shutdown(ctx); err != nil {
		c.log.Warn("telemetry shutdown failed", slog.Any("error", err))
	}
}
