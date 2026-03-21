package migrate

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Run applies all SQL migrations in the given directory.
// Migration 001 is always executed (idempotent by design via IF NOT EXISTS).
// Subsequent migrations are tracked via the schema_migrations table.
func Run(ctx context.Context, pool *pgxpool.Pool, dir string) error {
	files, err := filepath.Glob(filepath.Join(dir, "*.sql"))
	if err != nil || len(files) == 0 {
		slog.Warn("no migration files found", "dir", dir)
		return nil
	}
	sort.Strings(files)

	for _, file := range files {
		version := parseVersion(filepath.Base(file))

		// Migration 001 bootstraps the schema_migrations table, always run it
		if version == 1 {
			sql, err := os.ReadFile(file)
			if err != nil {
				return fmt.Errorf("read migration %s: %w", file, err)
			}
			if _, err := pool.Exec(ctx, string(sql)); err != nil {
				return fmt.Errorf("run migration %s: %w", file, err)
			}
			slog.Info("migration applied", "version", version)
			continue
		}

		// Check if already applied
		var applied bool
		_ = pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)`, version).Scan(&applied)
		if applied {
			continue
		}

		sql, err := os.ReadFile(file)
		if err != nil {
			slog.Warn("migration file not found, skipping", "file", file, "error", err)
			continue
		}

		if _, err := pool.Exec(ctx, string(sql)); err != nil {
			return fmt.Errorf("run migration %s: %w", file, err)
		}

		if _, err := pool.Exec(ctx, `INSERT INTO schema_migrations (version) VALUES ($1) ON CONFLICT DO NOTHING`, version); err != nil {
			slog.Warn("failed to record migration", "version", version, "error", err)
		}

		slog.Info("migration applied", "version", version)
	}

	return nil
}

func parseVersion(filename string) int {
	parts := strings.SplitN(filename, "_", 2)
	if len(parts) == 0 {
		return 0
	}
	v, _ := strconv.Atoi(parts[0])
	return v
}
