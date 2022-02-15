package hermes

import (
	"context"

	"github.com/jackc/pgx/v4/pgxpool"
)

// DB wraps the *pgxpool.Pool and provides the missing hermes function wrappers.
type DB struct {
	*pgxpool.Pool
}

// Begin a new transaction.
func (db *DB) Begin(ctx context.Context) (Conn, error) {
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}

	return &Tx{tx}, nil
}

// Commit does nothing.
func (db *DB) Commit(context.Context) error {
	return nil
}

// Rollback does nothing
func (db *DB) Rollback(_ context.Context) error {
	return nil
}

// Close does nothing.
func (db *DB) Close(context.Context) error {
	return nil
}
