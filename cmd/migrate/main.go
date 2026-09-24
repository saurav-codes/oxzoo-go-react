package main

import (
	"context"
	"embed"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
)

// Embedded SQL keeps the migrate step dependency-light: `go run ./cmd/migrate`
// works inside the release with no golang-migrate CLI to download.
//
//go:embed migrations
var migrations embed.FS

func main() {
	if err := run(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, "migrate:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	// ox injects DATABASE_URL from the postgres service; the fallback keeps
	// local `go run ./cmd/migrate` usable without the dashboard env editor.
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres@127.0.0.1:5432/oxzoo_go_react"
	}
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return err
	}
	defer conn.Close(ctx)

	if _, err := conn.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version int PRIMARY KEY,
			applied_at timestamptz NOT NULL DEFAULT now()
		)`); err != nil {
		return err
	}

	entries, err := migrations.ReadDir("migrations")
	if err != nil {
		return err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names) // numeric prefixes apply in order

	for _, name := range names {
		version, err := strconv.Atoi(strings.SplitN(name, "_", 2)[0])
		if err != nil {
			return fmt.Errorf("bad migration filename %s: %w", name, err)
		}
		var applied int
		if err := conn.QueryRow(ctx,
			"SELECT count(*) FROM schema_migrations WHERE version = $1", version,
		).Scan(&applied); err != nil {
			return err
		}
		if applied > 0 {
			continue // idempotent: already-applied versions are skipped
		}
		sql, err := migrations.ReadFile("migrations/" + name)
		if err != nil {
			return err
		}
		tx, err := conn.Begin(ctx)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, string(sql)); err != nil {
			tx.Rollback(ctx)
			return fmt.Errorf("applying %s: %w", name, err)
		}
		if _, err := tx.Exec(ctx,
			"INSERT INTO schema_migrations (version) VALUES ($1)", version); err != nil {
			tx.Rollback(ctx)
			return err
		}
		if err := tx.Commit(ctx); err != nil {
			return err
		}
		fmt.Println("applied", name)
	}
	return nil
}
