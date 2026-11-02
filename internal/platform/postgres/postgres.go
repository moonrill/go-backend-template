package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/moonrill/go-backend-template/internal/config"
)

const (
	pgUniqueViolation     = "23505"
	pgForeignKeyViolation = "23503"
)

// NewPool initializes and verifies a PostgreSQL connection pool using pgxpool.
// Returned errors never include the raw connection string or credentials.
func NewPool(ctx context.Context, cfg config.DatabaseConfig) (*pgxpool.Pool, error) {
	if !cfg.Enabled {
		return nil, errors.New("database is not enabled")
	}

	poolCfg, err := pgxpool.ParseConfig(cfg.URL)
	if err != nil {
		return nil, errors.New("invalid database configuration")
	}

	if cfg.MaxConns > 0 {
		poolCfg.MaxConns = cfg.MaxConns
	}
	if cfg.MinConns >= 0 {
		poolCfg.MinConns = cfg.MinConns
	}
	if cfg.MaxConnLifetime > 0 {
		poolCfg.MaxConnLifetime = cfg.MaxConnLifetime
	}
	if cfg.MaxConnIdleTime > 0 {
		poolCfg.MaxConnIdleTime = cfg.MaxConnIdleTime
	}
	if cfg.HealthCheckPeriod > 0 {
		poolCfg.HealthCheckPeriod = cfg.HealthCheckPeriod
	}
	connectTimeout := cfg.ConnectTimeout
	if connectTimeout <= 0 {
		connectTimeout = 3 * time.Second
	}
	poolCfg.ConnConfig.ConnectTimeout = connectTimeout

	connectCtx, cancel := context.WithTimeout(ctx, connectTimeout)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(connectCtx, poolCfg)
	if err != nil {
		return nil, errors.New("failed to initialize database pool")
	}

	if err := pool.Ping(connectCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("database ping failed: %w", sanitizeError(err))
	}

	return pool, nil
}

// HealthCheck returns a readiness check function that pings the database pool
// with a bounded timeout.
func HealthCheck(pool *pgxpool.Pool, timeout time.Duration) func(ctx context.Context) error {
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	return func(ctx context.Context) error {
		if pool == nil {
			return errors.New("database pool is not initialized")
		}
		pingCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		if err := pool.Ping(pingCtx); err != nil {
			return errors.New("database ping failed")
		}
		return nil
	}
}

// WithTx executes fn inside a database transaction, rolling back on error or panic
// and committing only when fn returns nil.
func WithTx(ctx context.Context, pool *pgxpool.Pool, fn func(pgx.Tx) error) error {
	if pool == nil {
		return errors.New("database pool is not initialized")
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", sanitizeError(err))
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if err := fn(tx); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", sanitizeError(err))
	}
	return nil
}

// IsNotFound reports whether err is pgx.ErrNoRows.
func IsNotFound(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}

// IsUniqueViolation reports whether err is a PostgreSQL unique constraint violation (SQLSTATE 23505).
func IsUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation
}

// IsForeignKeyViolation reports whether err is a PostgreSQL foreign key violation (SQLSTATE 23503).
func IsForeignKeyViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == pgForeignKeyViolation
}

func sanitizeError(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return fmt.Errorf("postgres error (code %s)", pgErr.Code)
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return context.DeadlineExceeded
	}
	if errors.Is(err, context.Canceled) {
		return context.Canceled
	}
	return errors.New("database operation failed")
}
