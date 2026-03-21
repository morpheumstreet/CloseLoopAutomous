package sqlite

import (
	"context"
	"testing"
)

func TestMigrateIdempotent(t *testing.T) {
	ctx := context.Background()
	db, err := Open(ctx, ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	if err := Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	var v int
	if err := db.QueryRowContext(ctx, `SELECT version FROM arms_schema_version WHERE singleton = 1`).Scan(&v); err != nil {
		t.Fatal(err)
	}
	if v != ExpectedSchemaVersion {
		t.Fatalf("version %d", v)
	}
}
