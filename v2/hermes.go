package hermes

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

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
	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, err
	}

	return &DB{pool}, nil
}
