package hermes

import (
	"context"
	"time"
)

// SetTimeout sets the default timeout for the database connection pool.
func (db *DB) SetTimeout(dur time.Duration) {
	db.defaultTimeout = dur
}

// Used for WithTimeout calls that already have a deadline.
func fakeCancel() {}

// WithTimeout creates a context with a timeout, assigning ctx as the parent of the timeout context.
// Returns the new context and its cancel function.  The timeout is based on the configured
// database pool connection timeout (see `WithDefaultTimeout`).
//
// Defaults to a 1 second timeout.
//
// Be sure to call the cancel function when you're done to clean up any resources in use!
func (db *DB) WithTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}

	if _, ok := ctx.Deadline(); ok {
		return ctx, fakeCancel
	}

	timeout := db.defaultTimeout
	if timeout == 0 {
		timeout = time.Second
	}

	return context.WithTimeout(ctx, timeout)
}

// BeginWithTimeout starts a custom transaction that manages the timeout context for you.
// If Conn already represents a transaction, pgx will create a savepoint instead.  This is
// experimental; use at your own risk!
func (db *DB) BeginWithTimeout(ctx context.Context) (*ContextualTx, error) {
	ctx, cancel := db.WithTimeout(ctx)

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}

	return &ContextualTx{tx, ctx, cancel}, nil
}

// SetTimeout sets the default timeout for a transaction.  If never set, the transaction uses the
// timeout of the connection from the database pool.
func (tx *Tx) SetTimeout(dur time.Duration) {
	tx.defaultTimeout = dur
}

// WithTimeout creates a context with a timeout, assigning ctx as the parent of the timeout context.
// Returns the new context and its cancel function.  The timeout is based on the configured
// database pool connection timeout (see `WithDefaultTimeout`).
//
// Defaults to a 1 second timeout.
//
// Be sure to call the cancel function when you're done to clean up any resources in use!
func (tx *Tx) WithTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}

	if _, ok := ctx.Deadline(); ok {
		return ctx, fakeCancel
	}

	timeout := tx.defaultTimeout
	if timeout == 0 {
		timeout = time.Second
	}

	return context.WithTimeout(ctx, timeout)
}
