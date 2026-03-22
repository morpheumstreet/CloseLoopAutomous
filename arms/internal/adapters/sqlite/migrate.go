package sqlite

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

func bootstrapVersion(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS arms_schema_version (
  singleton INTEGER PRIMARY KEY CHECK (singleton = 1),
  version INTEGER NOT NULL
);
INSERT OR IGNORE INTO arms_schema_version (singleton, version) VALUES (1, 0);
`)
	return err
}

func currentVersion(ctx context.Context, db *sql.DB) (int, error) {
	var v int
	err := db.QueryRowContext(ctx, `SELECT version FROM arms_schema_version WHERE singleton = 1`).Scan(&v)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return v, err
}

// ExpectedSchemaVersion is the highest migration number in migrations/*.sql (bump when adding files).
const ExpectedSchemaVersion = 18

// Migrate applies pending embedded migrations in lexical order (001_, 002_, …).
func Migrate(ctx context.Context, db *sql.DB) error {
	if err := bootstrapVersion(ctx, db); err != nil {
		return err
	}
	cv, err := currentVersion(ctx, db)
	if err != nil {
		return err
	}
	if cv >= ExpectedSchemaVersion {
		return nil
	}
	entries, err := fs.ReadDir(migrationFS, "migrations")
	if err != nil {
		return err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	for _, name := range names {
		ver, err := parseMigrationVersion(name)
		if err != nil {
			return err
		}
		if cv >= ver {
			continue
		}
		b, err := migrationFS.ReadFile(filepath.Join("migrations", name))
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, string(b)); err != nil {
			return fmt.Errorf("migration %s: %w", name, err)
		}
		var nv int
		if err := tx.QueryRowContext(ctx, `SELECT version FROM arms_schema_version WHERE singleton = 1`).Scan(&nv); err != nil {
			return err
		}
		if nv < ver {
			return fmt.Errorf("migration %s: expected version >= %d, got %d", name, ver, nv)
		}
		cv = nv
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	end, err := currentVersion(ctx, db)
	if err != nil {
		return err
	}
	if end != ExpectedSchemaVersion {
		return fmt.Errorf("after migrate: want version %d got %d", ExpectedSchemaVersion, end)
	}
	return nil
}

// BackupVacuum writes a consistent snapshot to destPath using SQLite VACUUM INTO (3.27+).
func BackupVacuum(ctx context.Context, db *sql.DB, destPath string) error {
	esc := strings.ReplaceAll(destPath, "'", "''")
	_, err := db.ExecContext(ctx, "VACUUM INTO '"+esc+"'")
	return err
}

// BackupBeforeMigrate writes diskPath.pre-migrate-{timestamp}.bak when diskPath is a file-backed DB.
func BackupBeforeMigrate(ctx context.Context, db *sql.DB, diskPath string) error {
	if diskPath == "" || diskPath == ":memory:" {
		return nil
	}
	ts := time.Now().UTC().Format("20060102-150405")
	dest := diskPath + ".pre-migrate-" + ts + ".bak"
	return BackupVacuum(ctx, db, dest)
}

func parseMigrationVersion(name string) (int, error) {
	base := filepath.Base(name)
	i := strings.IndexByte(base, '_')
	if i <= 0 {
		return 0, fmt.Errorf("bad migration name %q", base)
	}
	return strconv.Atoi(base[:i])
}
