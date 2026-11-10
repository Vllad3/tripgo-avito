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
	triprepo "github.com/Vllad3/tripgo-avito/internal/repository/trip"
	createuc "github.com/Vllad3/tripgo-avito/internal/usecase/trip/create"
	finishuc "github.com/Vllad3/tripgo-avito/internal/usecase/trip/finish"
	getuc "github.com/Vllad3/tripgo-avito/internal/usecase/trip/get"
)

type Container struct {
	pool   *pgxpool.Pool
	server *http.Server
}

func New(ctx context.Context, cfg *config.Config, log *slog.Logger) (*Container, error) {
	pool, err := postgres.NewPool(
		ctx,
		cfg.DatabaseURL,
		cfg.DatabaseMaxConns,
		cfg.DatabaseMinConns,
		cfg.DatabaseMaxConnLifetime,
		cfg.DatabaseConnectTimeout,
	)
	if err != nil {
		return nil, fmt.Errorf("connect to postgres: %w", err)
	}

	tx := transaction.New(pool, cfg.DatabaseQueryTimeout)
	repo := triprepo.New(pool, cfg.DatabaseQueryTimeout)
	errs := domainerror.NewWriter(log)

	handler, err := router.New(
		createhttp.New(createuc.New(repo, repo, tx, time.Now, uuid.New), errs),
		gethttp.New(getuc.New(repo), errs),
		finishhttp.New(finishuc.New(repo, repo, tx, time.Now), errs),
		health.New(pool, cfg.DatabaseQueryTimeout),
		log,
	)
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("build router: %w", err)
	}

	srv := server.New(cfg.HTTPAddr, server.Timeouts{
		ReadHeader: cfg.HTTPReadHeaderTimeout,
		Read:       cfg.HTTPReadTimeout,
		Write:      cfg.HTTPWriteTimeout,
		Idle:       cfg.HTTPIdleTimeout,
	}, handler)

	return &Container{pool: pool, server: srv}, nil
}

func (c *Container) Server() *http.Server {
	return c.server
}

func (c *Container) Close() {
	c.pool.Close()
}
