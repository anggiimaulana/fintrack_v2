package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TxManager runs a function inside a single database transaction and exposes
// per-user advisory locks. Repositories constructed with the same pool
// automatically use the transaction when it is present in the context.
type TxManager struct {
	pool *pgxpool.Pool
}

func NewTxManager(pool *pgxpool.Pool) *TxManager {
	return &TxManager{pool: pool}
}

// RunInTx begins a transaction, invokes fn with a tx-scoped context, and
// commits (or rolls back on error/panic).
func (m *TxManager) RunInTx(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	if err := fn(withTx(ctx, tx)); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	committed = true
	return nil
}

// AdvisoryXactLock takes a transaction-scoped advisory lock keyed by the given
// string. It must be called inside RunInTx; it serializes concurrent work per
// key and is released automatically at commit/rollback.
func (m *TxManager) AdvisoryXactLock(ctx context.Context, key string) error {
	_, err := q(ctx, m.pool).Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, key)
	if err != nil {
		return fmt.Errorf("failed to acquire advisory lock: %w", err)
	}
	return nil
}
