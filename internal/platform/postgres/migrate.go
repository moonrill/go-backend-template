package postgres

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/moonrill/go-backend-template/migrations"
)

// Advisory lock key used to serialize concurrent migration operations.
// Chosen to match a deterministic namespace hash for go-backend-template.
const migrationAdvisoryLockID int64 = 0x676f5f626b6e64 // "go_bknd"

// ErrNoMigrations indicates there are no migrations left to roll back.
var ErrNoMigrations = errors.New("no applied migrations to roll back")

type migrationPair struct {
	version  int64
	name     string
	upFile   string
	downFile string
}

// ApplyMigrations applies all unapplied embedded *.up.sql migrations in version order.
// It acquires a PostgreSQL transaction-level advisory lock per migration, records applied
// versions in the schema_migrations table, and is idempotent when called repeatedly.
func ApplyMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	if pool == nil {
		return errors.New("database pool is not initialized")
	}

	pairs, err := loadMigrationPairs()
	if err != nil {
		return err
	}

	conn, err := pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire migration connection: %w", sanitizeError(err))
	}
	defer conn.Release()

	if err := acquireAdvisoryLock(ctx, conn); err != nil {
		return err
	}
	defer releaseAdvisoryLock(context.Background(), conn)

	if err := ensureSchemaMigrationsTable(ctx, conn); err != nil {
		return err
	}

	applied, err := getAppliedVersions(ctx, conn)
	if err != nil {
		return err
	}

	for _, m := range pairs {
		if applied[m.version] {
			continue
		}
		sqlBytes, err := migrations.FS.ReadFile(m.upFile)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", m.upFile, err)
		}

		if err := applySingleMigration(ctx, conn, m.version, m.upFile, string(sqlBytes)); err != nil {
			return err
		}
	}

	return nil
}

// RollbackMigration rolls back the single highest applied migration version using its
// corresponding *.down.sql file and removes its version entry from schema_migrations.
// Returns ErrNoMigrations if no migrations are currently applied.
func RollbackMigration(ctx context.Context, pool *pgxpool.Pool) error {
	if pool == nil {
		return errors.New("database pool is not initialized")
	}

	pairs, err := loadMigrationPairs()
	if err != nil {
		return err
	}
	pairByVersion := make(map[int64]migrationPair, len(pairs))
	for _, p := range pairs {
		pairByVersion[p.version] = p
	}

	conn, err := pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire migration connection: %w", sanitizeError(err))
	}
	defer conn.Release()

	if err := acquireAdvisoryLock(ctx, conn); err != nil {
		return err
	}
	defer releaseAdvisoryLock(context.Background(), conn)

	if err := ensureSchemaMigrationsTable(ctx, conn); err != nil {
		return err
	}

	var latestVersion int64
	var dirty bool
	err = conn.QueryRow(ctx, `
		SELECT version, dirty
		FROM schema_migrations
		ORDER BY version DESC
		LIMIT 1
	`).Scan(&latestVersion, &dirty)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNoMigrations
	}
	if err != nil {
		return fmt.Errorf("query current migration version: %w", sanitizeError(err))
	}
	if dirty {
		return fmt.Errorf("database migration version %d is marked dirty; manual intervention required", latestVersion)
	}

	m, ok := pairByVersion[latestVersion]
	if !ok || m.downFile == "" {
		return fmt.Errorf("missing down migration file for version %d", latestVersion)
	}

	sqlBytes, err := migrations.FS.ReadFile(m.downFile)
	if err != nil {
		return fmt.Errorf("read down migration %s: %w", m.downFile, err)
	}

	tx, err := conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin rollback transaction: %w", sanitizeError(err))
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if _, err := tx.Exec(ctx, string(sqlBytes)); err != nil {
		return fmt.Errorf("execute down migration %s: %w", m.downFile, sanitizeError(err))
	}
	if _, err := tx.Exec(ctx, `DELETE FROM schema_migrations WHERE version = $1`, latestVersion); err != nil {
		return fmt.Errorf("delete migration version %d: %w", latestVersion, sanitizeError(err))
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit rollback %s: %w", m.downFile, sanitizeError(err))
	}
	return nil
}

func applySingleMigration(ctx context.Context, conn *pgxpool.Conn, version int64, fileName, sqlText string) error {
	tx, err := conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin migration %s: %w", fileName, sanitizeError(err))
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if _, err := tx.Exec(ctx, sqlText); err != nil {
		return fmt.Errorf("execute migration %s: %w", fileName, sanitizeError(err))
	}
	if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations (version, dirty) VALUES ($1, FALSE)`, version); err != nil {
		return fmt.Errorf("record migration version %d: %w", version, sanitizeError(err))
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit migration %s: %w", fileName, sanitizeError(err))
	}
	return nil
}

func acquireAdvisoryLock(ctx context.Context, conn *pgxpool.Conn) error {
	if _, err := conn.Exec(ctx, `SELECT pg_advisory_lock($1)`, migrationAdvisoryLockID); err != nil {
		return fmt.Errorf("acquire migration advisory lock: %w", sanitizeError(err))
	}
	return nil
}

func releaseAdvisoryLock(ctx context.Context, conn *pgxpool.Conn) {
	_, _ = conn.Exec(ctx, `SELECT pg_advisory_unlock($1)`, migrationAdvisoryLockID)
}

func ensureSchemaMigrationsTable(ctx context.Context, conn *pgxpool.Conn) error {
	_, err := conn.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version BIGINT PRIMARY KEY,
			dirty BOOLEAN NOT NULL DEFAULT FALSE
		)
	`)
	if err != nil {
		return fmt.Errorf("ensure schema_migrations table: %w", sanitizeError(err))
	}
	return nil
}

func getAppliedVersions(ctx context.Context, conn *pgxpool.Conn) (map[int64]bool, error) {
	rows, err := conn.Query(ctx, `SELECT version, dirty FROM schema_migrations`)
	if err != nil {
		return nil, fmt.Errorf("query applied migrations: %w", sanitizeError(err))
	}
	defer rows.Close()

	applied := make(map[int64]bool)
	for rows.Next() {
		var v int64
		var dirty bool
		if err := rows.Scan(&v, &dirty); err != nil {
			return nil, fmt.Errorf("scan migration version: %w", sanitizeError(err))
		}
		if dirty {
			return nil, fmt.Errorf("database migration version %d is marked dirty; manual intervention required", v)
		}
		applied[v] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate migration versions: %w", sanitizeError(err))
	}
	return applied, nil
}

func loadMigrationPairs() ([]migrationPair, error) {
	entries, err := fs.ReadDir(migrations.FS, ".")
	if err != nil {
		return nil, fmt.Errorf("read embedded migrations: %w", err)
	}

	byVersion := make(map[int64]*migrationPair)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		fileName := entry.Name()
		var suffix string
		switch {
		case strings.HasSuffix(fileName, ".up.sql"):
			suffix = ".up.sql"
		case strings.HasSuffix(fileName, ".down.sql"):
			suffix = ".down.sql"
		default:
			continue
		}

		base := strings.TrimSuffix(fileName, suffix)
		parts := strings.SplitN(base, "_", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return nil, fmt.Errorf("invalid migration filename %q: expected NNNNNN_description%s", fileName, suffix)
		}
		version, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil || version <= 0 {
			return nil, fmt.Errorf("invalid migration version in %q", fileName)
		}

		pair, ok := byVersion[version]
		if !ok {
			pair = &migrationPair{version: version, name: parts[1]}
			byVersion[version] = pair
		}
		if suffix == ".up.sql" {
			pair.upFile = fileName
		} else {
			pair.downFile = fileName
		}
	}

	pairs := make([]migrationPair, 0, len(byVersion))
	for _, p := range byVersion {
		if p.upFile == "" || p.downFile == "" {
			return nil, fmt.Errorf("migration version %d (%s) must have both .up.sql and .down.sql files", p.version, p.name)
		}
		pairs = append(pairs, *p)
	}
	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].version < pairs[j].version
	})
	return pairs, nil
}
