package hermes

import (
	"context"
	"sync"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgtype"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

var dataTypes []pgtype.DataType
var dtMutex sync.RWMutex

// Conn abstracts the *pgxpool.Pool struct and the pgx.Tx interface into a common interface.  This
// can be useful for building domain models more functionally, i.e the same function could be used
// for a single database query outside of a transaction, or included in a transaction with other
// function calls.
//
// It's also useful for testing, as you can pass a transaction into any database-related function,
// don't commit, and simply Close() at the end of the test to clean up the database.
type Conn interface {
	// Begin starts a transaction.  If Conn already represents a transaction, pgx will create a
	// savepoint instead.
	Begin(ctx context.Context) (Conn, error)

	// Commit the transaction.  Does nothing if Conn is a *pgxpool.Pool.  If the transaction is
	// a psuedo-transaction, i.e. a savepoint, releases the savepoint.  Otherwise commits the
	// transaction.
	Commit(ctx context.Context) error

	// Close rolls back the transaction if this is a real transaction or rolls back to the
	// savepoint if this is a pseudo nested transaction.  For a *pgxpool.Pool, this call is
	// ignored.
	//
	// Returns ErrTxClosed if the Conn is already closed, but is otherwise safe to call multiple
	// times. Hence, a defer conn.Close() is safe even if conn.Commit() will be called first in
	// a non-error condition.
	//
	// Any other failure of a real transaction will result in the connection being closed.
	Close(ctx context.Context) error

	CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error)
	SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults

	// TODO: Implement Prepare on *DB?
	// Prepare(ctx context.Context, name, sql string) (*pgconn.StatementDescription, error)

	Exec(ctx context.Context, sql string, arguments ...interface{}) (commandTag pgconn.CommandTag, err error)
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
}

func Connect(uri string) (Conn, error) {
	config, err := pgxpool.ParseConfig(uri)
	if err != nil {
		return nil, err
	}
	
	return ConnectConfig(config)
}
// Connect creates a pgx database connection pool and returns it.
func ConnectConfig(config *pgxpool.Config) (Conn, error) {
	config.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		dtMutex.RLock()
		defer dtMutex.RUnlock()

		for _, dt := range dataTypes {
			conn.ConnInfo().RegisterDataType(dt)
		}

		return nil
	}

	pool, err := pgxpool.ConnectConfig(context.Background(), config)
	if err != nil {
		return nil, err
	}

	return &DB{pool}, nil
}

// Register a new datatype to be associated with connections, such as a custom UUID or time data
// types.  Best to call this before calling Connect.
func Register(dataType pgtype.DataType) {
	dtMutex.Lock()
	defer dtMutex.Unlock()

	dataTypes = append(dataTypes, dataType)
}

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

// Close does nothing.
func (db *DB) Close(context.Context) error {
	return nil
}

// Tx wraps the pgx.Tx interface and provides the missing hermes function wrappers.
type Tx struct {
	pgx.Tx
}

// Begin starts a pseudo nested transaction.
func (tx *Tx) Begin(ctx context.Context) (Conn, error) {
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
	return tx.Tx.Rollback(ctx)
}
