package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite" // pure Go driver
)

// Open opens a SQLite database (file path or ":memory:") with sane pragmas.
func Open(ctx context.Context, path string) (*sql.DB, error) {
	dsn := path
	if path == ":memory:" {
		dsn = "file:mem?mode=memory&cache=shared"
	}
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	if _, err := db.ExecContext(ctx, `PRAGMA foreign_keys = ON; PRAGMA busy_timeout = 5000;`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("pragma: %w", err)
	}
	return db, nil
}
