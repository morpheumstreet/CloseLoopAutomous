//go:build integration

// Package integration holds opt-in HTTP stack tests (stub gateway, in-memory stores).
// Run: go test -tags=integration ./internal/integration/...
package integration_test

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/closeloopautomous/arms/internal/adapters/httpapi"
	"github.com/closeloopautomous/arms/internal/config"
	"github.com/closeloopautomous/arms/internal/platform"
)

func TestHTTP_ProductToTaskDispatch(t *testing.T) {
	cfg := config.Config{
		AccessLog: false,
	}
	app := platform.NewInMemoryApp(cfg, platform.Build{})
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

	var productsList struct {
		Products []map[string]any `json:"products"`
	}
	mustJSON(t, cli, http.MethodGet, base+"/api/products", nil, http.StatusOK, &productsList)
	if len(productsList.Products) < 1 {
		t.Fatalf("GET /api/products: want >=1 product, got %#v", productsList)
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

	var swipeHist struct {
		Swipes []map[string]any `json:"swipes"`
	}
	mustJSON(t, cli, http.MethodGet, base+"/api/products/"+pid+"/swipe-history", nil, http.StatusOK, &swipeHist)
	if len(swipeHist.Swipes) != 1 {
		t.Fatalf("expected one swipe history row: %#v", swipeHist)
	}
	if swipeHist.Swipes[0]["decision"] != "yes" {
		t.Fatalf("want decision yes: %#v", swipeHist.Swipes[0])
	}

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

func TestHTTP_LiveSSEOnDispatch(t *testing.T) {
	cfg := config.Config{AccessLog: false}
	app := platform.NewInMemoryApp(cfg, platform.Build{})
	t.Cleanup(func() { _ = app.Close() })

	srv := httptest.NewServer(httpapi.NewRouter(cfg, app.Handlers))
	t.Cleanup(srv.Close)
	base := srv.URL
	cli := srv.Client()

	done := make(chan string, 1)
	go func() {
		req, err := http.NewRequest(http.MethodGet, base+"/api/live/events", nil)
		if err != nil {
			done <- ""
			return
		}
		res, err := cli.Do(req)
		if err != nil {
			done <- ""
			return
		}
		defer res.Body.Close()
		buf := make([]byte, 16384)
		var acc strings.Builder
		for {
			n, err := res.Body.Read(buf)
			acc.Write(buf[:n])
			s := acc.String()
			if strings.Contains(s, `"type":"task_dispatched"`) {
				done <- s
				return
			}
			if err != nil {
				done <- ""
				return
			}
		}
	}()

	// Same pipeline as TestHTTP_ProductToTaskDispatch up to dispatch
	var prod map[string]any
	var created map[string]any
	mustJSON(t, cli, http.MethodPost, base+"/api/products", []byte(`{"name":"sse-p","workspace_id":"ws-sse"}`), http.StatusCreated, &created)
	pid, _ := created["id"].(string)
	mustJSON(t, cli, http.MethodPost, base+"/api/products/"+pid+"/research", nil, http.StatusOK, &prod)
	mustJSON(t, cli, http.MethodPost, base+"/api/products/"+pid+"/ideation", nil, http.StatusOK, &prod)
	var ideasWrap struct {
		Ideas []map[string]any `json:"ideas"`
	}
	mustJSON(t, cli, http.MethodGet, base+"/api/products/"+pid+"/ideas", nil, http.StatusOK, &ideasWrap)
	iid, _ := ideasWrap.Ideas[0]["id"].(string)
	mustJSON(t, cli, http.MethodPost, base+"/api/ideas/"+iid+"/swipe", []byte(`{"decision":"yes"}`), http.StatusOK, nil)
	taskCreate := []byte(fmt.Sprintf(`{"idea_id":%q,"spec":"sse spec"}`, iid))
	var task map[string]any
	mustJSON(t, cli, http.MethodPost, base+"/api/tasks", taskCreate, http.StatusCreated, &task)
	tid, _ := task["id"].(string)
	mustJSON(t, cli, http.MethodPost, base+"/api/tasks/"+tid+"/plan/approve", []byte(`{}`), http.StatusOK, &task)
	mustJSON(t, cli, http.MethodPatch, base+"/api/tasks/"+tid, []byte(`{"status":"assigned"}`), http.StatusOK, &task)
	mustJSON(t, cli, http.MethodPost, base+"/api/tasks/"+tid+"/dispatch", []byte(`{"estimated_cost":1}`), http.StatusOK, &task)

	select {
	case body := <-done:
		if body == "" || !strings.Contains(body, tid) {
			t.Fatalf("expected task_dispatched with task id in SSE buffer, got len=%d", len(body))
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for SSE task_dispatched")
	}
}

func TestHTTP_MergeQueueEnqueueAndList(t *testing.T) {
	cfg := config.Config{AccessLog: false}
	app := platform.NewInMemoryApp(cfg, platform.Build{})
	t.Cleanup(func() { _ = app.Close() })

	srv := httptest.NewServer(httpapi.NewRouter(cfg, app.Handlers))
	t.Cleanup(srv.Close)
	cli := srv.Client()
	base := srv.URL

	var prod map[string]any
	mustJSON(t, cli, http.MethodPost, base+"/api/products", []byte(`{"name":"mq-p","workspace_id":"ws-mq"}`), http.StatusCreated, &prod)
	pid, _ := prod["id"].(string)
	mustJSON(t, cli, http.MethodPost, base+"/api/products/"+pid+"/research", nil, http.StatusOK, &prod)
	mustJSON(t, cli, http.MethodPost, base+"/api/products/"+pid+"/ideation", nil, http.StatusOK, &prod)

	var ideasWrap struct {
		Ideas []map[string]any `json:"ideas"`
	}
	mustJSON(t, cli, http.MethodGet, base+"/api/products/"+pid+"/ideas", nil, http.StatusOK, &ideasWrap)
	iid, _ := ideasWrap.Ideas[0]["id"].(string)
	mustJSON(t, cli, http.MethodPost, base+"/api/ideas/"+iid+"/swipe", []byte(`{"decision":"yes"}`), http.StatusOK, nil)

	taskCreate := []byte(fmt.Sprintf(`{"idea_id":%q,"spec":"merge queue spec"}`, iid))
	var task map[string]any
	mustJSON(t, cli, http.MethodPost, base+"/api/tasks", taskCreate, http.StatusCreated, &task)
	tid, _ := task["id"].(string)

	var emptyQ struct {
		MergeQueue []map[string]any `json:"merge_queue"`
	}
	mustJSON(t, cli, http.MethodGet, base+"/api/products/"+pid+"/merge-queue", nil, http.StatusOK, &emptyQ)
	if len(emptyQ.MergeQueue) != 0 {
		t.Fatalf("want empty merge queue: %#v", emptyQ)
	}

	mustJSON(t, cli, http.MethodPost, base+"/api/tasks/"+tid+"/merge-queue", nil, http.StatusCreated, nil)

	var pWithQ struct {
		MergeQueuePending int64 `json:"merge_queue_pending"`
		MergePolicy       struct {
			MergeMethod string `json:"merge_method"`
		} `json:"merge_policy"`
	}
	mustJSON(t, cli, http.MethodGet, base+"/api/products/"+pid, nil, http.StatusOK, &pWithQ)
	if pWithQ.MergeQueuePending != 1 {
		t.Fatalf("product merge_queue_pending: want 1 got %d", pWithQ.MergeQueuePending)
	}
	if pWithQ.MergePolicy.MergeMethod != "merge" {
		t.Fatalf("merge_policy.merge_method: %#v", pWithQ.MergePolicy)
	}

	var ws struct {
		MergeQueuePending int64 `json:"merge_queue_pending"`
	}
	mustJSON(t, cli, http.MethodGet, base+"/api/workspaces", nil, http.StatusOK, &ws)
	if ws.MergeQueuePending != 1 {
		t.Fatalf("workspaces merge_queue_pending: want 1 got %d", ws.MergeQueuePending)
	}

	var q1 struct {
		MergeQueue []struct {
			TaskID        string `json:"task_id"`
			QueuePosition int    `json:"queue_position"`
			IsHead        bool   `json:"is_head"`
		} `json:"merge_queue"`
		PendingCount int64  `json:"pending_count"`
		HeadTaskID   string `json:"head_task_id"`
	}
	mustJSON(t, cli, http.MethodGet, base+"/api/products/"+pid+"/merge-queue", nil, http.StatusOK, &q1)
	if q1.PendingCount != 1 || q1.HeadTaskID != tid || len(q1.MergeQueue) != 1 {
		t.Fatalf("merge queue meta: %#v", q1)
	}
	r0 := q1.MergeQueue[0]
	if r0.TaskID != tid || r0.QueuePosition != 1 || !r0.IsHead {
		t.Fatalf("merge queue row: %#v", r0)
	}

	mustJSON(t, cli, http.MethodPost, base+"/api/tasks/"+tid+"/merge-queue", nil, http.StatusConflict, nil)

	mustJSON(t, cli, http.MethodPost, base+"/api/tasks/"+tid+"/merge-queue/complete", nil, http.StatusOK, nil)

	var ws2 struct {
		MergeQueuePending int64 `json:"merge_queue_pending"`
	}
	mustJSON(t, cli, http.MethodGet, base+"/api/workspaces", nil, http.StatusOK, &ws2)
	if ws2.MergeQueuePending != 0 {
		t.Fatalf("after complete want merge_queue_pending 0 got %d", ws2.MergeQueuePending)
	}
	var q2 struct {
		MergeQueue []map[string]any `json:"merge_queue"`
	}
	mustJSON(t, cli, http.MethodGet, base+"/api/products/"+pid+"/merge-queue", nil, http.StatusOK, &q2)
	if len(q2.MergeQueue) != 0 {
		t.Fatalf("after complete want empty pending list: %#v", q2)
	}

	mustJSON(t, cli, http.MethodPost, base+"/api/tasks/"+tid+"/merge-queue/complete", nil, http.StatusNotFound, nil)
	mustJSON(t, cli, http.MethodPost, base+"/api/tasks/"+tid+"/merge-queue", nil, http.StatusCreated, nil)
}

func TestHTTP_MergeQueueCancelRemovesPending(t *testing.T) {
	cfg := config.Config{AccessLog: false}
	app := platform.NewInMemoryApp(cfg, platform.Build{})
	t.Cleanup(func() { _ = app.Close() })
	srv := httptest.NewServer(httpapi.NewRouter(cfg, app.Handlers))
	t.Cleanup(srv.Close)
	cli := srv.Client()
	base := srv.URL

	var prod map[string]any
	mustJSON(t, cli, http.MethodPost, base+"/api/products", []byte(`{"name":"mq-cancel","workspace_id":"ws-mqc"}`), http.StatusCreated, &prod)
	pid, _ := prod["id"].(string)
	mustJSON(t, cli, http.MethodPost, base+"/api/products/"+pid+"/research", nil, http.StatusOK, &prod)
	mustJSON(t, cli, http.MethodPost, base+"/api/products/"+pid+"/ideation", nil, http.StatusOK, &prod)
	var ideasWrap struct {
		Ideas []map[string]any `json:"ideas"`
	}
	mustJSON(t, cli, http.MethodGet, base+"/api/products/"+pid+"/ideas", nil, http.StatusOK, &ideasWrap)
	iid, _ := ideasWrap.Ideas[0]["id"].(string)
	mustJSON(t, cli, http.MethodPost, base+"/api/ideas/"+iid+"/swipe", []byte(`{"decision":"yes"}`), http.StatusOK, nil)
	var task map[string]any
	mustJSON(t, cli, http.MethodPost, base+"/api/tasks", []byte(fmt.Sprintf(`{"idea_id":%q,"spec":"c1"}`, iid)), http.StatusCreated, &task)
	tid, _ := task["id"].(string)
	mustJSON(t, cli, http.MethodPost, base+"/api/tasks/"+tid+"/merge-queue", nil, http.StatusCreated, nil)
	mustJSON(t, cli, http.MethodDelete, base+"/api/tasks/"+tid+"/merge-queue", nil, http.StatusOK, nil)
	var q struct {
		MergeQueue   []any `json:"merge_queue"`
		PendingCount int64 `json:"pending_count"`
	}
	mustJSON(t, cli, http.MethodGet, base+"/api/products/"+pid+"/merge-queue", nil, http.StatusOK, &q)
	if q.PendingCount != 0 || len(q.MergeQueue) != 0 {
		t.Fatalf("after cancel want empty queue: %#v", q)
	}
}

func TestHTTP_MergeQueueCancelNonHead(t *testing.T) {
	cfg := config.Config{AccessLog: false}
	app := platform.NewInMemoryApp(cfg, platform.Build{})
	t.Cleanup(func() { _ = app.Close() })
	srv := httptest.NewServer(httpapi.NewRouter(cfg, app.Handlers))
	t.Cleanup(srv.Close)
	cli := srv.Client()
	base := srv.URL

	var prod map[string]any
	mustJSON(t, cli, http.MethodPost, base+"/api/products", []byte(`{"name":"mq-two","workspace_id":"ws-mq2"}`), http.StatusCreated, &prod)
	pid, _ := prod["id"].(string)
	mustJSON(t, cli, http.MethodPost, base+"/api/products/"+pid+"/research", nil, http.StatusOK, &prod)
	mustJSON(t, cli, http.MethodPost, base+"/api/products/"+pid+"/ideation", nil, http.StatusOK, &prod)
	var ideasWrap struct {
		Ideas []map[string]any `json:"ideas"`
	}
	mustJSON(t, cli, http.MethodGet, base+"/api/products/"+pid+"/ideas", nil, http.StatusOK, &ideasWrap)
	iid, _ := ideasWrap.Ideas[0]["id"].(string)
	mustJSON(t, cli, http.MethodPost, base+"/api/ideas/"+iid+"/swipe", []byte(`{"decision":"yes"}`), http.StatusOK, nil)
	var t1, t2 map[string]any
	mustJSON(t, cli, http.MethodPost, base+"/api/tasks", []byte(fmt.Sprintf(`{"idea_id":%q,"spec":"first"}`, iid)), http.StatusCreated, &t1)
	mustJSON(t, cli, http.MethodPost, base+"/api/tasks", []byte(fmt.Sprintf(`{"idea_id":%q,"spec":"second"}`, iid)), http.StatusCreated, &t2)
	tid1, _ := t1["id"].(string)
	tid2, _ := t2["id"].(string)
	mustJSON(t, cli, http.MethodPost, base+"/api/tasks/"+tid1+"/merge-queue", nil, http.StatusCreated, nil)
	mustJSON(t, cli, http.MethodPost, base+"/api/tasks/"+tid2+"/merge-queue", nil, http.StatusCreated, nil)
	var q2 struct {
		PendingCount int64  `json:"pending_count"`
		HeadTaskID   string `json:"head_task_id"`
	}
	mustJSON(t, cli, http.MethodGet, base+"/api/products/"+pid+"/merge-queue", nil, http.StatusOK, &q2)
	if q2.PendingCount != 2 || q2.HeadTaskID != tid1 {
		t.Fatalf("want 2 pending head %q: %#v", tid1, q2)
	}
	mustJSON(t, cli, http.MethodDelete, base+"/api/tasks/"+tid2+"/merge-queue", nil, http.StatusOK, nil)
	var q3 struct {
		PendingCount int64  `json:"pending_count"`
		HeadTaskID   string `json:"head_task_id"`
	}
	mustJSON(t, cli, http.MethodGet, base+"/api/products/"+pid+"/merge-queue", nil, http.StatusOK, &q3)
	if q3.PendingCount != 1 || q3.HeadTaskID != tid1 {
		t.Fatalf("after tail cancel: %#v", q3)
	}
}

func TestHTTP_ProductMergePolicyJSONInvalid(t *testing.T) {
	cfg := config.Config{AccessLog: false}
	app := platform.NewInMemoryApp(cfg, platform.Build{})
	t.Cleanup(func() { _ = app.Close() })
	srv := httptest.NewServer(httpapi.NewRouter(cfg, app.Handlers))
	t.Cleanup(srv.Close)
	cli := srv.Client()
	base := srv.URL
	mustJSON(t, cli, http.MethodPost, base+"/api/products",
		[]byte(`{"name":"bad-mpj","workspace_id":"ws-bad","merge_policy_json":"{"}`),
		http.StatusBadRequest, nil)
}

func TestHTTP_ConvoySubtaskWebhookAndSecondDispatch(t *testing.T) {
	const whSecret = "int-wh-secret"
	cfg := config.Config{AccessLog: false, WebhookSecret: whSecret}
	app := platform.NewInMemoryApp(cfg, platform.Build{})
	t.Cleanup(func() { _ = app.Close() })

	srv := httptest.NewServer(httpapi.NewRouter(cfg, app.Handlers))
	t.Cleanup(srv.Close)
	cli := srv.Client()
	base := srv.URL

	var prod map[string]any
	mustJSON(t, cli, http.MethodPost, base+"/api/products", []byte(`{"name":"cv-p","workspace_id":"ws-cv"}`), http.StatusCreated, &prod)
	pid, _ := prod["id"].(string)
	mustJSON(t, cli, http.MethodPost, base+"/api/products/"+pid+"/research", nil, http.StatusOK, &prod)
	mustJSON(t, cli, http.MethodPost, base+"/api/products/"+pid+"/ideation", nil, http.StatusOK, &prod)

	var ideasWrap struct {
		Ideas []map[string]any `json:"ideas"`
	}
	mustJSON(t, cli, http.MethodGet, base+"/api/products/"+pid+"/ideas", nil, http.StatusOK, &ideasWrap)
	iid, _ := ideasWrap.Ideas[0]["id"].(string)
	mustJSON(t, cli, http.MethodPost, base+"/api/ideas/"+iid+"/swipe", []byte(`{"decision":"yes"}`), http.StatusOK, nil)

	taskCreate := []byte(fmt.Sprintf(`{"idea_id":%q,"spec":"convoy parent"}`, iid))
	var task map[string]any
	mustJSON(t, cli, http.MethodPost, base+"/api/tasks", taskCreate, http.StatusCreated, &task)
	tid, _ := task["id"].(string)
	mustJSON(t, cli, http.MethodPost, base+"/api/tasks/"+tid+"/plan/approve", []byte(`{}`), http.StatusOK, &task)
	mustJSON(t, cli, http.MethodPatch, base+"/api/tasks/"+tid, []byte(`{"status":"assigned"}`), http.StatusOK, &task)
	mustJSON(t, cli, http.MethodPost, base+"/api/tasks/"+tid+"/dispatch", []byte(`{"estimated_cost":1}`), http.StatusOK, &task)

	convBody := []byte(fmt.Sprintf(
		`{"parent_task_id":%q,"product_id":%q,"subtasks":[{"id":"b1","agent_role":"builder"},{"id":"t1","agent_role":"tester","depends_on":["b1"]}]}`,
		tid, pid,
	))
	var conv map[string]any
	mustJSON(t, cli, http.MethodPost, base+"/api/convoys", convBody, http.StatusCreated, &conv)
	cid, _ := conv["id"].(string)
	if cid == "" {
		t.Fatalf("convoy id missing: %#v", conv)
	}

	mustJSON(t, cli, http.MethodPost, base+"/api/convoys/"+cid+"/dispatch-ready", []byte(`{"estimated_cost":1}`), http.StatusOK, nil)
	var c1 map[string]any
	mustJSON(t, cli, http.MethodGet, base+"/api/convoys/"+cid, nil, http.StatusOK, &c1)
	edges1, _ := c1["edges"].([]any)
	if len(edges1) != 1 {
		t.Fatalf("edges: %#v", edges1)
	}
	e0 := edges1[0].(map[string]any)
	if e0["from"] != "b1" || e0["to"] != "t1" {
		t.Fatalf("edge want b1->t1 got %#v", e0)
	}
	subs1, _ := c1["subtasks"].([]any)
	if len(subs1) != 2 {
		t.Fatalf("subtasks: %#v", subs1)
	}
	b0 := subs1[0].(map[string]any)
	t0 := subs1[1].(map[string]any)
	if b0["dispatched"] != true || t0["dispatched"] != false {
		t.Fatalf("first dispatch wave: %#v / %#v", b0, t0)
	}

	whPayload := []byte(fmt.Sprintf(`{"task_id":%q,"convoy_id":%q,"subtask_id":"b1"}`, tid, cid))
	mustWebhook(t, cli, base, whSecret, whPayload, http.StatusOK, nil)

	mustJSON(t, cli, http.MethodPost, base+"/api/convoys/"+cid+"/dispatch-ready", []byte(`{"estimated_cost":1}`), http.StatusOK, nil)
	var c2 map[string]any
	mustJSON(t, cli, http.MethodGet, base+"/api/convoys/"+cid, nil, http.StatusOK, &c2)
	subs2, _ := c2["subtasks"].([]any)
	b1 := subs2[0].(map[string]any)
	t1 := subs2[1].(map[string]any)
	if b1["completed"] != true {
		t.Fatalf("builder should be completed after webhook: %#v", b1)
	}
	if t1["dispatched"] != true || t1["completed"] != false {
		t.Fatalf("tester after second dispatch: %#v", t1)
	}
}

func TestHTTP_TaskConvoyMCShape(t *testing.T) {
	cfg := config.Config{AccessLog: false}
	app := platform.NewInMemoryApp(cfg, platform.Build{})
	t.Cleanup(func() { _ = app.Close() })
	srv := httptest.NewServer(httpapi.NewRouter(cfg, app.Handlers))
	t.Cleanup(srv.Close)
	cli := srv.Client()
	base := srv.URL

	var prod map[string]any
	mustJSON(t, cli, http.MethodPost, base+"/api/products", []byte(`{"name":"mc-shape","workspace_id":"ws-mc"}`), http.StatusCreated, &prod)
	pid, _ := prod["id"].(string)
	mustJSON(t, cli, http.MethodPost, base+"/api/products/"+pid+"/research", nil, http.StatusOK, &prod)
	mustJSON(t, cli, http.MethodPost, base+"/api/products/"+pid+"/ideation", nil, http.StatusOK, &prod)
	var ideasWrap struct {
		Ideas []map[string]any `json:"ideas"`
	}
	mustJSON(t, cli, http.MethodGet, base+"/api/products/"+pid+"/ideas", nil, http.StatusOK, &ideasWrap)
	iid, _ := ideasWrap.Ideas[0]["id"].(string)
	mustJSON(t, cli, http.MethodPost, base+"/api/ideas/"+iid+"/swipe", []byte(`{"decision":"yes"}`), http.StatusOK, nil)
	var task map[string]any
	mustJSON(t, cli, http.MethodPost, base+"/api/tasks", []byte(fmt.Sprintf(`{"idea_id":%q,"spec":"parent for mc convoy"}`, iid)), http.StatusCreated, &task)
	tid, _ := task["id"].(string)
	mustJSON(t, cli, http.MethodPost, base+"/api/tasks/"+tid+"/plan/approve", []byte(`{}`), http.StatusOK, &task)
	mustJSON(t, cli, http.MethodPatch, base+"/api/tasks/"+tid, []byte(`{"status":"assigned"}`), http.StatusOK, &task)
	mustJSON(t, cli, http.MethodPost, base+"/api/tasks/"+tid+"/dispatch", []byte(`{"estimated_cost":1}`), http.StatusOK, &task)

	mcBody := []byte(`{"strategy":"manual","name":"MC Convoy","subtasks":[{"title":"Step A","suggested_role":"builder","description":"d1"}]}`)
	var mc map[string]any
	mustJSON(t, cli, http.MethodPost, base+"/api/tasks/"+tid+"/convoy", mcBody, http.StatusCreated, &mc)
	if mc["name"] != "MC Convoy" || mc["strategy"] != "manual" {
		t.Fatalf("mc create: %#v", mc)
	}
	subs, _ := mc["subtasks"].([]any)
	if len(subs) != 1 {
		t.Fatalf("subtasks: %#v", subs)
	}
	st0 := subs[0].(map[string]any)
	tk, _ := st0["task"].(map[string]any)
	if tk["title"] != "Step A" || tk["status"] != "inbox" {
		t.Fatalf("nested task: %#v", tk)
	}

	var mc2 map[string]any
	mustJSON(t, cli, http.MethodGet, base+"/api/tasks/"+tid+"/convoy", nil, http.StatusOK, &mc2)
	if mc2["parent_task_id"] != tid {
		t.Fatalf("parent_task_id: %#v", mc2)
	}
	var prog map[string]any
	mustJSON(t, cli, http.MethodGet, base+"/api/tasks/"+tid+"/convoy/progress", nil, http.StatusOK, &prog)
	if prog["total"].(float64) != 1 {
		t.Fatalf("progress total: %#v", prog)
	}
}

func mustWebhook(t *testing.T, cli *http.Client, base, secret string, body []byte, wantStatus int, out any) {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, base+"/api/webhooks/agent-completion", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	req.Header.Set("X-Arms-Signature", hex.EncodeToString(mac.Sum(nil)))
	res, err := cli.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	if res.StatusCode != wantStatus {
		t.Fatalf("webhook: status %d body %s", res.StatusCode, string(b))
	}
	if out != nil && len(b) > 0 {
		if err := json.Unmarshal(b, out); err != nil {
			t.Fatalf("webhook decode: %v body %s", err, string(b))
		}
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
