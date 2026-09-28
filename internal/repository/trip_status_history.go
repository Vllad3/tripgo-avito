package repository

import (
	"context"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/Vllad3/tripgo-avito/internal/database"
	"github.com/Vllad3/tripgo-avito/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TripStatusHistoryRepository interface {
	Create(ctx context.Context, tripID uuid.UUID, from *domain.TripStatus, to domain.TripStatus, reason *string) error
}

type tripStatusHistoryRepository struct {
	pool *pgxpool.Pool
}

func NewTripStatusHistoryRepository(pool *pgxpool.Pool) TripStatusHistoryRepository {
	return &tripStatusHistoryRepository{pool: pool}
}

func (r *tripStatusHistoryRepository) Create(ctx context.Context, tripID uuid.UUID, from *domain.TripStatus, to domain.TripStatus, reason *string) error {
	query, args, err := squirrel.
		Insert("trip_status_history").
		Columns("trip_id", "from_status", "to_status", "reason").
		Values(tripID, from, to, reason).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return fmt.Errorf("build query: %w", err)
	}

	if _, err := database.ExecutorFromContext(ctx, r.pool).Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("exec query: %w", err)
	}
	return nil
}
