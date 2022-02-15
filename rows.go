package hermes

import (
	"database/sql"
	"errors"

	"github.com/jackc/pgx/v4"
)

// NoRows returns true if the supplied error is one of the NoRows indicators
func NoRows(err error) bool {
	return errors.Is(err, sql.ErrNoRows) || errors.Is(err, pgx.ErrNoRows)
}

// RowScanner is a shared interface between pgx.Rows and pgx.Row
type RowScanner interface {
	Scan(dest ...interface{}) error
}
