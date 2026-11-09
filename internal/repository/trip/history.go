package trip

import (
	"context"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"

	domain "github.com/Vllad3/tripgo-avito/internal/domain/trip"
)

func (r *Repository) Add(ctx context.Context, tripID uuid.UUID, from *domain.Status, to domain.Status) error {
	ctx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	var fromStatus *string
	if from != nil {
		s := string(*from)
		fromStatus = &s
	}

	query, args, err := squirrel.
		Insert(historyTable).
		Columns("trip_id", "from_status", "to_status").
		Values(tripID, fromStatus, string(to)).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert history: %w", err)
	}

	if _, err := r.db(ctx).Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("insert history: %w", err)
	}

	return nil
}
