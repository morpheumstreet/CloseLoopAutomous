//go:build integration

// Package integration holds opt-in HTTP stack tests (stub gateway, in-memory stores).
// Run: go test -tags=integration ./internal/integration/...
package integration_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/closeloopautomous/arms/internal/adapters/httpapi"
	"github.com/closeloopautomous/arms/internal/config"
	"github.com/closeloopautomous/arms/internal/platform"
)

func TestHTTP_ProductToTaskDispatch(t *testing.T) {
	cfg := config.Config{
		AccessLog: false,
	}
	app := platform.NewInMemoryApp(cfg)
	t.Cleanup(func() { _ = app.Close() })

	srv := httptest.NewServer(httpapi.NewRouter(cfg, app.Handlers))
	t.Cleanup(srv.Close)

	cli := srv.Client()
	base := srv.URL

	mustJSON(t, cli, http.MethodGet, base+"/api/health", nil, http.StatusOK, nil)

	prodBody := []byte(`{"name":"integration-p","workspace_id":"ws-1"}`)
	var prod map[string]any
	mustJSON(t, cli, http.MethodPost, base+"/api/products", prodBody, http.StatusCreated, &prod)
	pid, _ := prod["id"].(string)
	if pid == "" {
		t.Fatalf("product id missing: %#v", prod)
	}

	mustJSON(t, cli, http.MethodPost, base+"/api/products/"+pid+"/research", nil, http.StatusOK, &prod)
	mustJSON(t, cli, http.MethodPost, base+"/api/products/"+pid+"/ideation", nil, http.StatusOK, &prod)

	var ideasWrap struct {
		Ideas []map[string]any `json:"ideas"`
	}
	mustJSON(t, cli, http.MethodGet, base+"/api/products/"+pid+"/ideas", nil, http.StatusOK, &ideasWrap)
	if len(ideasWrap.Ideas) < 1 {
		t.Fatalf("expected ideas: %#v", ideasWrap)
	}
	iid, _ := ideasWrap.Ideas[0]["id"].(string)
	if iid == "" {
		t.Fatal("idea id missing")
	}

	swipe := []byte(`{"decision":"yes"}`)
	var idea map[string]any
	mustJSON(t, cli, http.MethodPost, base+"/api/ideas/"+iid+"/swipe", swipe, http.StatusOK, &idea)

	taskCreate := []byte(fmt.Sprintf(`{"idea_id":%q,"spec":"integration spec"}`, iid))
	var task map[string]any
	mustJSON(t, cli, http.MethodPost, base+"/api/tasks", taskCreate, http.StatusCreated, &task)
	tid, _ := task["id"].(string)
	if tid == "" {
		t.Fatalf("task id missing: %#v", task)
	}
	if task["status"] != "planning" {
		t.Fatalf("want planning got %#v", task["status"])
	}

	mustJSON(t, cli, http.MethodPost, base+"/api/tasks/"+tid+"/plan/approve", []byte(`{}`), http.StatusOK, &task)
	if task["status"] != "inbox" {
		t.Fatalf("want inbox got %#v", task["status"])
	}

	patch := []byte(`{"status":"assigned"}`)
	mustJSON(t, cli, http.MethodPatch, base+"/api/tasks/"+tid, patch, http.StatusOK, &task)
	if task["status"] != "assigned" {
		t.Fatalf("want assigned got %#v", task["status"])
	}

	dispatch := []byte(`{"estimated_cost":1}`)
	mustJSON(t, cli, http.MethodPost, base+"/api/tasks/"+tid+"/dispatch", dispatch, http.StatusOK, &task)
	if task["status"] != "in_progress" {
		t.Fatalf("want in_progress got %#v", task["status"])
	}
	if task["external_ref"] == nil || task["external_ref"] == "" {
		t.Fatalf("want external_ref from stub gateway: %#v", task["external_ref"])
	}
}

func mustJSON(t *testing.T, cli *http.Client, method, url string, body []byte, wantStatus int, out any) {
	t.Helper()
	var r io.Reader
	if body != nil {
		r = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, url, r)
	if err != nil {
		t.Fatal(err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := cli.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	if res.StatusCode != wantStatus {
		t.Fatalf("%s %s: status %d body %s", method, url, res.StatusCode, string(b))
	}
	if out != nil && len(b) > 0 {
		if err := json.Unmarshal(b, out); err != nil {
			t.Fatalf("%s %s decode: %v body %s", method, url, err, string(b))
		}
	}
}
