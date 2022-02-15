package hermes

import (
	"context"
	"sync"

	"github.com/jackc/pgtype"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

var dataTypes []pgtype.DataType
var dtMutex sync.RWMutex

// Connect creates a pgx database connection pool and returns it.
func Connect(uri string) (*DB, error) {
	config, err := pgxpool.ParseConfig(uri)
	if err != nil {
		return nil, err
	}

	return ConnectConfig(config)
}

// ConnectConfig creates a pgx database connection pool based on a pool configuration and returns
// it.
func ConnectConfig(config *pgxpool.Config) (*DB, error) {
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
