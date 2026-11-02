package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/moonrill/go-backend-template/internal/config"
)

func TestNewPoolDisabled(t *testing.T) {
	_, err := NewPool(context.Background(), config.DatabaseConfig{Enabled: false})
	if err == nil {
		t.Fatal("expected error when database is disabled")
	}
}

func TestNewPoolDoesNotLeakCredentialsInErrors(t *testing.T) {
	secret := "super-secret-password-999"
	cfg := config.DatabaseConfig{
		Enabled:        true,
		URL:            "postgres://appuser:" + secret + "@127.0.0.1:1/nonexistent?sslmode=disable",
		MaxConns:       2,
		MinConns:       0,
		ConnectTimeout: 100 * time.Millisecond,
	}

	_, err := NewPool(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected connection error")
	}
	if strings.Contains(err.Error(), secret) || strings.Contains(err.Error(), "appuser") {
		t.Fatalf("error leaked credentials: %v", err)
	}
}

func TestHealthCheckNilPool(t *testing.T) {
	check := HealthCheck(nil, time.Second)
	if err := check(context.Background()); err == nil {
		t.Fatal("expected error when pool is nil")
	}
}

func TestErrorClassificationHelpers(t *testing.T) {
	if !IsNotFound(pgx.ErrNoRows) {
		t.Error("expected IsNotFound(pgx.ErrNoRows) == true")
	}
	if IsNotFound(errors.New("other")) {
		t.Error("expected IsNotFound(other) == false")
	}

	uniqueErr := &pgconn.PgError{Code: pgUniqueViolation}
	if !IsUniqueViolation(uniqueErr) {
		t.Error("expected IsUniqueViolation == true for 23505")
	}
	if IsForeignKeyViolation(uniqueErr) {
		t.Error("expected IsForeignKeyViolation == false for 23505")
	}

	fkErr := &pgconn.PgError{Code: pgForeignKeyViolation}
	if !IsForeignKeyViolation(fkErr) {
		t.Error("expected IsForeignKeyViolation == true for 23503")
	}
}

func TestWithTxAndMigrationsNilPool(t *testing.T) {
	if err := WithTx(context.Background(), nil, func(tx pgx.Tx) error { return nil }); err == nil {
		t.Error("expected error from WithTx when pool is nil")
	}
	if err := ApplyMigrations(context.Background(), nil); err == nil {
		t.Error("expected error from ApplyMigrations when pool is nil")
	}
	if err := RollbackMigration(context.Background(), nil); err == nil {
		t.Error("expected error from RollbackMigration when pool is nil")
	}
}

func TestLoadMigrationPairs(t *testing.T) {
	pairs, err := loadMigrationPairs()
	if err != nil {
		t.Fatalf("loadMigrationPairs() error = %v", err)
	}
	if len(pairs) == 0 {
		t.Fatal("expected at least one embedded migration pair")
	}
	if pairs[0].version != 1 || pairs[0].upFile == "" || pairs[0].downFile == "" {
		t.Fatalf("unexpected first migration pair: %+v", pairs[0])
	}
}
