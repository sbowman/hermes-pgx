package hermes

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// BeginWithTimeout starts a custom transaction that manages the timeout context for you.
// If Conn already represents a transaction, pgx will create a savepoint instead.  This is
// experimental; use at your own risk!
func (tx *Tx) BeginWithTimeout(ctx context.Context) (*ContextualTx, error) {
	ctx, cancel := tx.WithTimeout(ctx)

	newTx, err := tx.Tx.Begin(ctx)
	if err != nil {
		return nil, err
	}

	return &ContextualTx{newTx, ctx, cancel}, nil
}

// ContextualTx is a prototype for starting a transaction using the default timeout and using the
// context on the transaction for any database calls from then on.
//
// This does not support the hermes.Conn interface.  At this point you can only use this transaction
// in a single function if you stick with hermes.Conn in your function parameters.
type ContextualTx struct {
	pgx.Tx

	ctx    context.Context
	cancel context.CancelFunc
}

// Commit the transaction.  Does nothing if Conn is a *pgxpool.Pool.  If the transaction is
// a psuedo-transaction, i.e. a savepoint, releases the savepoint.  Otherwise commits the
// transaction.
func (tx *ContextualTx) Commit() error {
	return tx.Tx.Commit(tx.ctx)
}

// Rollback the transaction. Does nothing if Conn is a *pgxpool.Pool.
func (tx *ContextualTx) Rollback() error {
	return tx.Tx.Rollback(tx.ctx)
}

// Close rolls back the transaction if this is a real transaction or rolls back to the
// savepoint if this is a pseudo nested transaction.  It also cancels the context for the
// transaction.
//
// Returns ErrTxClosed if the Conn is already closed, but is otherwise safe to call multiple
// times. Hence, a defer conn.Close() is safe even if conn.Commit() will be called first in
// a non-error condition.
//
// Any other failure of a real transaction will result in the connection being closed.
func (tx *ContextualTx) Close() error {
	defer tx.cancel()
	return tx.Rollback()
}

// CopyFrom uses the context on the transaction.
func (tx *ContextualTx) CopyFrom(tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error) {
	return tx.Tx.CopyFrom(tx.ctx, tableName, columnNames, rowSrc)
}

// SendBatch uses the context on the transaction.
func (tx *ContextualTx) SendBatch(b *pgx.Batch) pgx.BatchResults {
	return tx.Tx.SendBatch(tx.ctx, b)
}

// Prepare uses the context on the transaction.
func (tx *ContextualTx) Prepare(name, sql string) (*pgconn.StatementDescription, error) {
	return tx.Tx.Prepare(tx.ctx, name, sql)
}

// Exec uses the context on the transaction.
func (tx *ContextualTx) Exec(sql string, arguments ...interface{}) (commandTag pgconn.CommandTag, err error) {
	return tx.Tx.Exec(tx.ctx, sql, arguments...)
}

// Query uses the context on the transaction.
func (tx *ContextualTx) Query(sql string, args ...interface{}) (pgx.Rows, error) {
	return tx.Tx.Query(tx.ctx, sql, args...)
}

// QueryRow uses the context on the transaction.
func (tx *ContextualTx) QueryRow(sql string, args ...interface{}) pgx.Row {
	return tx.Tx.QueryRow(tx.ctx, sql, args...)
}
