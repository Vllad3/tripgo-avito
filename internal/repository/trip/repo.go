package trip

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Vllad3/tripgo-avito/internal/infra/postgres/transaction"
)

const (
	tripsTable   = "trips"
	historyTable = "trip_status_history"
)

type Repository struct {
	pool         *pgxpool.Pool
	queryTimeout time.Duration
}

func New(pool *pgxpool.Pool, queryTimeout time.Duration) *Repository {
	return &Repository{pool: pool, queryTimeout: queryTimeout}
}

func (r *Repository) db(ctx context.Context) executor {
	return transaction.ExecutorFromContext(ctx, r.pool)
}
