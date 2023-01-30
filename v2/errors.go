package hermes

import "github.com/jackc/pgx/v5/pgconn"

// PostgreSQL disconnect errors - https://www.postgresql.org/docs/current/errcodes-appendix.html
const (
	OperatorIntervention = "57000"
	QueryCanceled        = "57014"
	AdminShutdown        = "57P01"
	CrashShutdown        = "57P02"
	CannotConnectNow     = "57P03"
	DatabaseDropped      = "57P04"
	IdleSessionTimeout   = "57P05"
)

var (
	// Disconnects is the list of PostgreSQL error codes that indicate the connection failed.
	Disconnects = []string{
		OperatorIntervention,
		QueryCanceled,
		AdminShutdown,
		CrashShutdown,
		CannotConnectNow,
		DatabaseDropped,
		IdleSessionTimeout,
	}
)

// IsDisconnected returns true if the error is a PostgreSQL disconnect error (SQLSTATE 57P01).
func IsDisconnected(err error) bool {
	if err == nil {
		return false
	}

	pgErr, ok := err.(*pgconn.PgError)
	if !ok {
		return false
	}

	for _, code := range Disconnects {
		if pgErr.Code == code {
			return true
		}
	}

	return false
}
