package hermes

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DB wraps the *pgxpool.Pool and provides the missing hermes function wrappers.
type DB struct {
	*pgxpool.Pool
	defaultTimeout time.Duration
}

// Begin a new transaction.
func (db *DB) Begin(ctx context.Context) (Conn, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}

	return &Tx{tx, db.defaultTimeout}, nil
}

// Commit does nothing.
func (db *DB) Commit(context.Context) error {
	return nil
}

// Rollback does nothing
func (db *DB) Rollback(context.Context) error {
	return nil
}

// Close does nothing.
func (db *DB) Close(context.Context) error {
	return nil
}
