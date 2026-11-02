package postgres_test

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/moonrill/go-backend-template/internal/config"
	"github.com/moonrill/go-backend-template/internal/platform/postgres"
	"github.com/moonrill/go-backend-template/internal/platform/postgres/sqlc"
)

func TestPostgresIntegration(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("skipping PostgreSQL integration test: TEST_DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// Create an admin pool solely to create and drop the isolated test schema.
	adminPool, err := postgres.NewPool(ctx, config.DatabaseConfig{
		Enabled:        true,
		URL:            dsn,
		MaxConns:       2,
		MinConns:       0,
		ConnectTimeout: 3 * time.Second,
	})
	if err != nil {
		t.Fatalf("admin NewPool() error = %v", err)
	}
	defer adminPool.Close()

	// Isolate the test run inside a temporary schema so existing database objects are untouched.
	schemaName := fmt.Sprintf("test_phase2_%d", time.Now().UnixNano())
	if _, err := adminPool.Exec(ctx, "CREATE SCHEMA "+schemaName); err != nil {
		t.Fatalf("create test schema: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		_, _ = adminPool.Exec(cleanupCtx, "DROP SCHEMA IF EXISTS "+schemaName+" CASCADE")
	})

	// Configure a dedicated pool whose every connection uses search_path=<schemaName>,public.
	isolatedDSN, err := withSearchPath(dsn, schemaName+",public")
	if err != nil {
		t.Fatalf("withSearchPath() error = %v", err)
	}

	pool, err := postgres.NewPool(ctx, config.DatabaseConfig{
		Enabled:           true,
		URL:               isolatedDSN,
		MaxConns:          5,
		MinConns:          1,
		ConnectTimeout:    3 * time.Second,
		MaxConnLifetime:   5 * time.Minute,
		MaxConnIdleTime:   1 * time.Minute,
		HealthCheckPeriod: 30 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewPool() error = %v", err)
	}
	defer pool.Close()

	// Apply migrations and verify idempotency when called a second time.
	if err := postgres.ApplyMigrations(ctx, pool); err != nil {
		t.Fatalf("first ApplyMigrations() error = %v", err)
	}
	if err := postgres.ApplyMigrations(ctx, pool); err != nil {
		t.Fatalf("second ApplyMigrations() (idempotency) error = %v", err)
	}

	check := postgres.HealthCheck(pool, 2*time.Second)
	if err := check(ctx); err != nil {
		t.Fatalf("HealthCheck() error = %v", err)
	}

	queries := sqlc.New(pool)

	created, err := queries.CreateUser(ctx, sqlc.CreateUserParams{
		Email:    "alice@example.com",
		FullName: "Alice Example",
		Role:     "admin",
		IsActive: true,
	})
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	if created.ID == "" || created.Email != "alice@example.com" || created.Role != "admin" {
		t.Fatalf("unexpected created user: %+v", created)
	}

	fetched, err := queries.GetUserByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetUserByID() error = %v", err)
	}
	if fetched.ID != created.ID || fetched.FullName != created.FullName {
		t.Fatalf("fetched = %+v, want %+v", fetched, created)
	}

	newName := "Alice Updated"
	newRole := "member"
	updated, err := queries.UpdateUser(ctx, sqlc.UpdateUserParams{
		ID:       created.ID,
		FullName: &newName,
		Role:     &newRole,
	})
	if err != nil {
		t.Fatalf("UpdateUser() error = %v", err)
	}
	if updated.FullName != newName || updated.Role != newRole {
		t.Fatalf("updated = %+v, want full_name=%q role=%q", updated, newName, newRole)
	}

	// Verify transaction rollback leaves data intact on error.
	_ = postgres.WithTx(ctx, pool, func(tx pgx.Tx) error {
		qtx := queries.WithTx(tx)
		if _, err := qtx.DeleteUser(ctx, created.ID); err != nil {
			return err
		}
		return fmt.Errorf("abort transaction")
	})

	if _, err := queries.GetUserByID(ctx, created.ID); err != nil {
		t.Fatalf("expected user to remain after rolled-back tx, got error: %v", err)
	}

	rows, err := queries.DeleteUser(ctx, created.ID)
	if err != nil {
		t.Fatalf("DeleteUser() error = %v", err)
	}
	if rows != 1 {
		t.Fatalf("DeleteUser() rows = %d, want 1", rows)
	}

	_, err = queries.GetUserByID(ctx, created.ID)
	if !postgres.IsNotFound(err) {
		t.Fatalf("expected IsNotFound after delete, got: %v", err)
	}

	// Verify RollbackMigration rolls back the latest version and subsequent rollback returns ErrNoMigrations.
	if err := postgres.RollbackMigration(ctx, pool); err != nil {
		t.Fatalf("RollbackMigration() error = %v", err)
	}
	if err := postgres.RollbackMigration(ctx, pool); !errors.Is(err, postgres.ErrNoMigrations) {
		t.Fatalf("second RollbackMigration() error = %v, want ErrNoMigrations", err)
	}

	// Re-apply migrations after rollback to verify full up -> down -> up cycle.
	if err := postgres.ApplyMigrations(ctx, pool); err != nil {
		t.Fatalf("re-ApplyMigrations() after rollback error = %v", err)
	}
}

func withSearchPath(dsn, searchPath string) (string, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("search_path", searchPath)
	u.RawQuery = q.Encode()
	return u.String(), nil
}
