//go:build integration

package testutil

import (
	"context"
	"fmt"
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

// TruncateAll removes all data from tables in the correct order for FK constraints.
// Useful as a test cleanup step.
func TruncateAll(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	tables := []string{
		"counselor_cabin_explanations",
		"counselor_cabin_assignments",
		"counselor_cabin_solutions",
		"assignment_runs",
		"counselor_cocounselor_preferences",
		"counselor_age_group_preferences",
		"counselor_session_history",
		"session_age_group_cabins",
		"session_age_groups",
		"sessions",
		"seasons",
		"counselors",
		"cabins",
		"age_groups",
		"camps",
	}
	for _, table := range tables {
		_, err := pool.Exec(context.Background(), fmt.Sprintf("DELETE FROM %s", table))
		if err != nil {
			t.Fatalf("truncating %s: %v", table, err)
		}
	}
}
