//go:build integration

package testutil

import (
	"context"
	"os"
	"testing"

	"camp-scheduler/internal/config"
	"camp-scheduler/internal/db"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// MustOpenDB creates a connection pool to the test database. It prefers
// APP_DATABASE_URL_TEST when set; otherwise it loads config from the
// DATABASE_* env vars and appends "_test" to DATABASE_NAME, mirroring the
// CLI tooling's idea of a "test" database.
func MustOpenDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dsn := os.Getenv("APP_DATABASE_URL_TEST")
	if dsn == "" {
		cfg, err := config.Load()
		if err != nil {
			t.Fatalf("loading config: %v", err)
		}
		cfg.Database.URL = "" // force DSN to be built from discrete fields
		cfg.Database.Name = cfg.Database.Name + "_test"
		dsn = cfg.Database.DSN()
	}

	pool, err := pgxpool.New(context.Background(), dsn)
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
//
// TRUNCATE requires a privilege the runtime app_user role intentionally does
// not have, so this opens a one-shot connection as the privileged
// camp_scheduler role (via DATABASE_URL_TEST) for the truncate only.
func TruncateAll(t *testing.T, _ *pgxpool.Pool) {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL_TEST")
	if dsn == "" {
		t.Fatal("DATABASE_URL_TEST is not set; required for test cleanup")
	}
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("opening privileged connection for truncate: %v", err)
	}
	defer conn.Close(ctx)
	if _, err := conn.Exec(ctx, "TRUNCATE app.camps CASCADE"); err != nil {
		t.Fatalf("truncating tables: %v", err)
	}
}
