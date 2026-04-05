//go:build integration

package testutil

import (
	"context"
	"testing"

	"camp-scheduler/internal/config"
	"camp-scheduler/internal/db"

	"github.com/jackc/pgx/v5/pgxpool"
)

// MustOpenDB creates a connection pool to the test database. It reads the
// DATABASE_* environment variables and appends "_test" to DATABASE_NAME to
// target a dedicated test database, then fails the test immediately if the
// connection cannot be established.
func MustOpenDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("loading config: %v", err)
	}
	cfg.Database.Name = cfg.Database.Name + "_test"

	pool, err := pgxpool.New(context.Background(), cfg.Database.DSN())
	if err != nil {
		t.Fatalf("connecting to test database: %v", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		t.Fatalf("pinging test database: %v", err)
	}

	t.Cleanup(func() { pool.Close() })
	return pool
}

// MustQueries creates a *db.Queries backed by a test database connection.
func MustQueries(t *testing.T) *db.Queries {
	t.Helper()
	return db.New(MustOpenDB(t))
}

// TruncateAll removes all data from every application table. Because every
// table has a direct or transitive foreign key to camps, truncating camps
// with CASCADE is sufficient.
func TruncateAll(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	_, err := pool.Exec(context.Background(), "TRUNCATE camps CASCADE")
	if err != nil {
		t.Fatalf("truncating tables: %v", err)
	}
}
