// Package db verbindet die Datenbank und führt Migrationen aus.
package db

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"sort"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrations embed.FS

// Advisory Lock, damit bei mehreren Instanzen nur eine migriert.
const migrationLock = 7428301

func Connect(ctx context.Context, url string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("Datenbank nicht erreichbar: %w", err)
	}
	return pool, nil
}

// Migrate führt alle noch nicht ausgeführten SQL-Dateien in alphabetischer Reihenfolge aus.
func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()

	if _, err := conn.Exec(ctx, `SELECT pg_advisory_lock($1)`, migrationLock); err != nil {
		return err
	}
	defer conn.Exec(context.Background(), `SELECT pg_advisory_unlock($1)`, migrationLock)

	if _, err := conn.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version TEXT PRIMARY KEY,
		applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
	)`); err != nil {
		return err
	}

	files, err := fs.Glob(migrations, "migrations/*.sql")
	if err != nil {
		return err
	}
	sort.Strings(files)

	// Kennt die App eine bereits ausgeführte Migration nicht, passt die Datenbank nicht zum Code
	var unknown []string
	if err := conn.QueryRow(ctx,
		`SELECT COALESCE(array_agg(version ORDER BY version), '{}') FROM schema_migrations WHERE NOT version = ANY($1)`,
		files).Scan(&unknown); err != nil {
		return err
	}
	if len(unknown) > 0 {
		return fmt.Errorf("Datenbank enthält unbekannte Migrationen %v, sie passt nicht zu dieser Version der App "+
			"(lokal: make db-reset, löscht alle Daten)", unknown)
	}

	for _, f := range files {
		var exists bool
		if err := conn.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version = $1)`, f).Scan(&exists); err != nil {
			return err
		}
		if exists {
			continue
		}
		sql, err := migrations.ReadFile(f)
		if err != nil {
			return err
		}
		tx, err := conn.Begin(ctx)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, string(sql)); err != nil {
			tx.Rollback(ctx)
			return fmt.Errorf("Migration %s: %w", f, err)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, f); err != nil {
			tx.Rollback(ctx)
			return err
		}
		if err := tx.Commit(ctx); err != nil {
			return err
		}
	}
	return nil
}
