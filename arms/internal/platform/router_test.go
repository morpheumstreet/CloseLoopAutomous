package platform_test

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/closeloopautomous/arms/internal/adapters/httpapi"
	"github.com/closeloopautomous/arms/internal/config"
	"github.com/closeloopautomous/arms/internal/domain"
	"github.com/closeloopautomous/arms/internal/platform"
)

func TestWebhookAgentCompletion(t *testing.T) {
	cfg := httpapi.Config{WebhookSecret: "testsecret"}
	app := platform.NewInMemoryApp(cfg)
	ctx := context.Background()
	now := time.Unix(1700000000, 0)
	err := app.Tasks.Save(ctx, &domain.Task{
		ID:           "task-1",
		ProductID:    "prod-1",
		IdeaID:       "idea-1",
		Spec:         "s",
		Status:       domain.StatusInProgress,
		PlanApproved: true,
		CreatedAt:    now,
		UpdatedAt:    now,
	})
	if err != nil {
		t.Fatal(err)
	}

	body := []byte(`{"task_id":"task-1"}`)
	mac := hmac.New(sha256.New, []byte(cfg.WebhookSecret))
	_, _ = mac.Write(body)
	sig := hex.EncodeToString(mac.Sum(nil))

	req := httptest.NewRequest(http.MethodPost, "/api/webhooks/agent-completion", bytes.NewReader(body))
	req.Header.Set("X-Arms-Signature", sig)
	rec := httptest.NewRecorder()
	httpapi.NewRouter(cfg, app.Handlers).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}

	tt, err := app.Tasks.ByID(ctx, "task-1")
	if err != nil {
		t.Fatal(err)
	}
	if tt.Status != domain.StatusDone {
		t.Fatalf("want completed got %s", tt.Status.String())
	}
}

func TestWebhookAgentCompletionConvoyFieldsPartialRejected(t *testing.T) {
	cfg := httpapi.Config{WebhookSecret: "testsecret"}
	app := platform.NewInMemoryApp(cfg)
	body := []byte(`{"task_id":"task-parent","convoy_id":"c1"}`)
	mac := hmac.New(sha256.New, []byte(cfg.WebhookSecret))
	_, _ = mac.Write(body)
	sig := hex.EncodeToString(mac.Sum(nil))
	req := httptest.NewRequest(http.MethodPost, "/api/webhooks/agent-completion", bytes.NewReader(body))
	req.Header.Set("X-Arms-Signature", sig)
	rec := httptest.NewRecorder()
	httpapi.NewRouter(cfg, app.Handlers).ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
}

func TestWebhookInvalidHMAC(t *testing.T) {
	cfg := httpapi.Config{WebhookSecret: "s"}
	app := platform.NewInMemoryApp(cfg)
	body := []byte(`{"task_id":"x"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/webhooks/agent-completion", bytes.NewReader(body))
	req.Header.Set("X-Arms-Signature", "deadbeef")
	rec := httptest.NewRecorder()
	httpapi.NewRouter(cfg, app.Handlers).ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status %d", rec.Code)
	}
}

func TestBearerAuthRequired(t *testing.T) {
	cfg := httpapi.Config{MCAPIToken: "tok", WebhookSecret: "s"}
	app := platform.NewInMemoryApp(cfg)
	req := httptest.NewRequest(http.MethodGet, "/api/agents", nil)
	rec := httptest.NewRecorder()
	httpapi.NewRouter(cfg, app.Handlers).ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status %d", rec.Code)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/api/agents", nil)
	req2.Header.Set("Authorization", "Bearer tok")
	rec2 := httptest.NewRecorder()
	httpapi.NewRouter(cfg, app.Handlers).ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("status %d", rec2.Code)
	}
}

func TestBearerOrACLBasicWhenBothConfigured(t *testing.T) {
	cfg := httpapi.Config{
		MCAPIToken: "tok",
		ACLUsers: []config.ACLUser{
			{UserID: "u", Password: "p", Role: "admin"},
		},
	}
	app := platform.NewInMemoryApp(cfg)
	req := httptest.NewRequest(http.MethodGet, "/api/agents", nil)
	req.SetBasicAuth("u", "p")
	rec := httptest.NewRecorder()
	httpapi.NewRouter(cfg, app.Handlers).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
}

func TestHealthNoAuth(t *testing.T) {
	cfg := httpapi.Config{MCAPIToken: "tok"}
	app := platform.NewInMemoryApp(cfg)
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()
	httpapi.NewRouter(cfg, app.Handlers).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
}

func TestACLBasicAdmin(t *testing.T) {
	cfg := httpapi.Config{
		ACLUsers: []config.ACLUser{
			{UserID: "admin", Password: "secret", Role: "admin"},
		},
	}
	app := platform.NewInMemoryApp(cfg)
	req := httptest.NewRequest(http.MethodPost, "/api/products", bytes.NewReader([]byte(`{"name":"n","workspace_id":"w"}`)))
	req.SetBasicAuth("admin", "secret")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	httpapi.NewRouter(cfg, app.Handlers).ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
}

func TestACLReadCannotPost(t *testing.T) {
	cfg := httpapi.Config{
		ACLUsers: []config.ACLUser{
			{UserID: "v", Password: "p", Role: "read"},
		},
	}
	app := platform.NewInMemoryApp(cfg)
	req := httptest.NewRequest(http.MethodPost, "/api/products", bytes.NewReader([]byte(`{"name":"n","workspace_id":"w"}`)))
	req.SetBasicAuth("v", "p")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	httpapi.NewRouter(cfg, app.Handlers).ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status %d want 403", rec.Code)
	}
}

func TestACLReadCanGet(t *testing.T) {
	cfg := httpapi.Config{
		ACLUsers: []config.ACLUser{
			{UserID: "v", Password: "p", Role: "read"},
		},
	}
	app := platform.NewInMemoryApp(cfg)
	req := httptest.NewRequest(http.MethodGet, "/api/products", nil)
	req.SetBasicAuth("v", "p")
	rec := httptest.NewRecorder()
	httpapi.NewRouter(cfg, app.Handlers).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
}

func TestACLSSEBasicQuery(t *testing.T) {
	cfg := httpapi.Config{
		ACLUsers: []config.ACLUser{
			{UserID: "v", Password: "p", Role: "read"},
		},
	}
	app := platform.NewInMemoryApp(cfg)
	basic := base64.StdEncoding.EncodeToString([]byte("v:p"))
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	req := httptest.NewRequest(http.MethodGet, "/api/live/events?basic="+basic, nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	httpapi.NewRouter(cfg, app.Handlers).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
}

func TestSSEBearerAuth(t *testing.T) {
	cfg := httpapi.Config{MCAPIToken: "sse-tok"}
	app := platform.NewInMemoryApp(cfg)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	req := httptest.NewRequest(http.MethodGet, "/api/live/events", nil).WithContext(ctx)
	req.Header.Set("Authorization", "Bearer sse-tok")
	rec := httptest.NewRecorder()
	httpapi.NewRouter(cfg, app.Handlers).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
}
