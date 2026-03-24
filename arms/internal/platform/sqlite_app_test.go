package platform_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/adapters/httpapi"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/domain"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/platform"
)

func TestOpenAppSQLiteFileRoundTrip(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	cfg := httpapi.Config{DatabasePath: dbPath}
	app, err := platform.OpenApp(ctx, cfg, platform.Build{})
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	now := time.Unix(1700000000, 0).UTC()
	p := &domain.Product{
		ID: "p1", Name: "x", Stage: domain.StageResearch, WorkspaceID: "w",
		UpdatedAt: now,
	}
	if err := app.Products.Save(ctx, p); err != nil {
		t.Fatal(err)
	}
	p2, err := app.Products.ByID(ctx, "p1")
	if err != nil || p2.Name != "x" {
		t.Fatalf("got %+v err %v", p2, err)
	}

	_ = app.Close()
	app2, err := platform.OpenApp(ctx, cfg, platform.Build{})
	if err != nil {
		t.Fatal(err)
	}
	defer app2.Close()
	p3, err := app2.Products.ByID(ctx, "p1")
	if err != nil || p3.Name != "x" {
		t.Fatalf("reopen got %+v err %v", p3, err)
	}
}
