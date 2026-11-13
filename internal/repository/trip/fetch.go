package trip

import (
	"context"
	"errors"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	domain "github.com/Vllad3/tripgo-avito/internal/domain/trip"
)

func (r *Repository) Get(ctx context.Context, id uuid.UUID) (*domain.Trip, error) {
	return r.fetch(ctx, id, false)
}

func (r *Repository) GetForUpdate(ctx context.Context, id uuid.UUID) (*domain.Trip, error) {
	return r.fetch(ctx, id, true)
}

func (r *Repository) fetch(ctx context.Context, id uuid.UUID, forUpdate bool) (*domain.Trip, error) {
	ctx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	builder := squirrel.
		Select(tripColumns...).
		From(tripsTable).
		Where(squirrel.Eq{"id": id}).
		PlaceholderFormat(squirrel.Dollar)
	if forUpdate {
		builder = builder.Suffix("FOR UPDATE")
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build select trip: %w", err)
	}

	var row tripRow
	if err := r.db(ctx).QueryRow(ctx, query, args...).Scan(row.dest()...); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrTripNotFound
		}
		return nil, fmt.Errorf("select trip: %w", err)
	}

	return row.toDomain(), nil
}
