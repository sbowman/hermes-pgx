package hermes

import (
	"context"
	"errors"
	"sync"

	"github.com/jackc/pgx/v5"
)

// ErrLocked returned if you try to acquire an advisory lock and it's already in use.
var ErrLocked = errors.New("advisory lock already acquired")

type AdvisoryLock interface {
	Release() error
}

// SessionAdvisoryLock creates a session-wide advisory lock.
type SessionAdvisoryLock struct {
	mutex sync.Mutex

	ID   uint64
	conn *pgx.Conn
}

// Release the session-wide advisory lock.
func (lock *SessionAdvisoryLock) Release() error {
	lock.mutex.Lock()
	defer lock.mutex.Unlock()

	// The lock was already released
	if lock.conn == nil {
		return nil
	}

	if _, err := lock.conn.Exec(context.Background(), "SELECT pg_advisory_unlock($1)", lock.ID); err != nil {
		return err
	}

	lock.conn = nil

	return nil
}

// Lock creates a session-wide advisory lock in the database.  Call Release() to release the
// advisory lock.
func (db *DB) Lock(ctx context.Context, id uint64) (AdvisoryLock, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	conn, err := db.Acquire(ctx)
	if err != nil {
		return nil, err
	}

	if _, err := conn.Exec(ctx, "SELECT pg_advisory_lock($1)", id); err != nil {
		return nil, err
	}

	return &SessionAdvisoryLock{
		ID:   id,
		conn: conn.Conn(),
	}, nil
}

// TryLock tries to create a session-wide advisory lock in the database.  If successful, returns the
// advisory lock.  If not, returns ErrLocked.  If you acquire the lock, be sure to release it!
func (db *DB) TryLock(ctx context.Context, id uint64) (AdvisoryLock, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	conn, err := db.Acquire(ctx)
	if err != nil {
		return nil, err
	}

	var available bool
	row := conn.QueryRow(ctx, "SELECT pg_try_advisory_lock($1)", id)
	if err := row.Scan(&available); err != nil {
		return nil, err
	}

	if !available {
		return nil, ErrLocked
	}

	return &SessionAdvisoryLock{
		ID:   id,
		conn: conn.Conn(),
	}, nil
}

// TxAdvisoryLock is a placeholder so the Lock/Release functionality is the same for the
// hermes.Conn interface.
type TxAdvisoryLock struct {
	ID uint64
}

// Release does nothing on a transactional advisory lock.
func (lock *TxAdvisoryLock) Release() error {
	return nil
}

// Lock creates an transactional advisory lock in the database.  This lock will be released at the
// end of the transaction, on either commit or rollback.  You may call AdvisoryLock.Release(), but
// it does nothing on this type of advisory lock.
func (tx *Tx) Lock(ctx context.Context, id uint64) (AdvisoryLock, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	if _, err := tx.Conn().Exec(ctx, "SELECT pg_advisory_xact_lock($1)", id); err != nil {
		return nil, err
	}

	return &TxAdvisoryLock{
		ID: id,
	}, nil
}

// TryLock creates an transactional advisory lock in the database.  You may manually call Release() on
// the AdvisoryLock, or the lock will release automatically on commit or rollback.
func (tx *Tx) TryLock(ctx context.Context, id uint64) (AdvisoryLock, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	var available bool
	row := tx.QueryRow(ctx, "SELECT pg_try_advisory_xact_lock($1)", id)
	if err := row.Scan(&available); err != nil {
		return nil, err
	}

	if !available {
		return nil, ErrLocked
	}

	return &TxAdvisoryLock{
		ID: id,
	}, nil
}
