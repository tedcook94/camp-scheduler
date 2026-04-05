//go:build integration

package testutil

import (
	"context"
	"fmt"
	"os"
	"testing"

	"camp-scheduler/internal/db"

	"github.com/jackc/pgx/v5/pgxpool"
)

// MustOpenDB creates a connection pool to the test database. It reads the
// TEST_DATABASE_URL environment variable (falling back to a localhost default)
// and fails the test immediately if the connection cannot be established.
func MustOpenDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://camp_scheduler:p@ss123@localhost:5432/camp_scheduler?sslmode=disable"
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
