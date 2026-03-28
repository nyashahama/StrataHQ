package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	dbgen "github.com/stratahq/backend/db/gen"
)

// WithTx runs fn inside a single database transaction. The transaction is
// committed on success and rolled back on any error.
func WithTx(ctx context.Context, pool *Pool, fn func(tx pgx.Tx) error) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			return fmt.Errorf("rollback failed: %v (original error: %w)", rbErr, err)
		}
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// WithTxQueries runs fn inside a single database transaction and supplies a
// *dbgen.Queries bound to that transaction. Use this when multiple sqlc calls
// must succeed or fail together.
//
// Example:
//
//	err := database.WithTxQueries(ctx, db, func(q *dbgen.Queries) error {
//	    if _, err := q.CreateLevyAccount(ctx, ...); err != nil {
//	        return err
//	    }
//	    return q.CreateLevyPayment(ctx, ...)
//	})
func WithTxQueries(ctx context.Context, pool *Pool, fn func(q *dbgen.Queries) error) error {
	return WithTx(ctx, pool, func(tx pgx.Tx) error {
		return fn(dbgen.New(tx))
	})
}
