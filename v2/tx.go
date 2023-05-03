package hermes

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// Tx wraps the pgx.Tx interface and provides the missing hermes function wrappers.
type Tx struct {
	pgx.Tx
}

// Begin starts a pseudo nested transaction.
func (tx *Tx) Begin(ctx context.Context) (Conn, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	newTx, err := tx.Tx.Begin(ctx)
	if err != nil {
		return nil, err
	}

	return &Tx{newTx}, nil
}

// Close rolls back the transaction if this is a real transaction or rolls back to the
// savepoint if this is a pseudo nested transaction.
//
// Returns ErrTxClosed if the Conn is already closed, but is otherwise safe to call multiple
// times. Hence, a defer conn.Close() is safe even if conn.Commit() will be called first in
// a non-error condition.
//
// Any other failure of a real transaction will result in the connection being closed.
func (tx *Tx) Close(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}

	return tx.Tx.Rollback(ctx)
}
