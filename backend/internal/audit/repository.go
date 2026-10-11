package audit

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresRepository persists audit events into the append-only audit_logs
// table. The table itself rejects UPDATE/DELETE via trigger (see migration
// 0005), so this repository only ever INSERTs.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository builds a repository backed by the given pool.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// InsertBatch writes all events in a single multi-row INSERT.
func (r *PostgresRepository) InsertBatch(ctx context.Context, events []Event) error {
	if len(events) == 0 {
		return nil
	}

	const cols = 9
	var sb strings.Builder
	sb.WriteString(`INSERT INTO audit_logs
        (actor_id, action, entity, entity_id, before, after, ip, user_agent, request_id)
        VALUES `)

	args := make([]any, 0, len(events)*cols)
	for i, e := range events {
		if i > 0 {
			sb.WriteByte(',')
		}
		base := i * cols
		fmt.Fprintf(&sb, "($%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d)",
			base+1, base+2, base+3, base+4, base+5, base+6, base+7, base+8, base+9)

		args = append(args,
			e.ActorID, e.Action, e.Entity, nullIfEmpty(e.EntityID),
			jsonOrNil(e.Before), jsonOrNil(e.After),
			nullIfEmpty(e.IP), nullIfEmpty(e.UserAgent), nullIfEmpty(e.RequestID),
		)
	}

	_, err := r.pool.Exec(ctx, sb.String(), args...)
	return err
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func jsonOrNil(m map[string]any) any {
	if len(m) == 0 {
		return nil
	}
	return m
}
