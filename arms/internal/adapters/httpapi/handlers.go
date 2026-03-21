package httpapi

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	workgit "github.com/closeloopautomous/arms/internal/adapters/workspace"
	agentapp "github.com/closeloopautomous/arms/internal/application/agent"
	"github.com/closeloopautomous/arms/internal/application/autopilot"
	"github.com/closeloopautomous/arms/internal/application/convoy"
	"github.com/closeloopautomous/arms/internal/application/cost"
	"github.com/closeloopautomous/arms/internal/application/mergequeue"
	productapp "github.com/closeloopautomous/arms/internal/application/product"
	"github.com/closeloopautomous/arms/internal/application/task"
	"github.com/closeloopautomous/arms/internal/domain"
	"github.com/closeloopautomous/arms/internal/ports"
)

const webhookSigHeader = "X-Arms-Signature"

var gitBranchTokenRE = regexp.MustCompile(`^[a-zA-Z0-9/._-]+$`)

// Handlers holds application services for the HTTP adapter.
type Handlers struct {
	Config    Config
	Product   *productapp.Service
	Autopilot *autopilot.Service
	Task      *task.Service
	Convoy    *convoy.Service
	Agent     *agentapp.Service
	Cost           *cost.Service
	Live           ports.ActivityStream // SSE subscribers; required for live routes
	WorkspacePorts ports.WorkspacePortRepository
	MergeQueue     ports.WorkspaceMergeQueueRepository
	MergeShip      *mergequeue.Service // optional; real merge ship when ARMS_MERGE_* set
	AgentHealth    ports.AgentHealthRepository // optional; nil keeps legacy agent listing stub
	PrefModel      ports.PreferenceModelRepository
	OperationsLog  ports.OperationsLogRepository
	// AutopilotScheduleReconcile re-scans products and enqueues Asynq per-product ticks (set from cmd/arms when Redis is configured).
	AutopilotScheduleReconcile func(context.Context)
	// ResyncProductSchedule re-enqueues Asynq product:schedule:tick for one product after PATCH (set when Redis is configured).
	ResyncProductSchedule func(context.Context, domain.ProductID)
}

func (h *Handlers) maybeReconcileAutopilotSchedule(ctx context.Context) {
	if h.AutopilotScheduleReconcile == nil {
		return
	}
	h.AutopilotScheduleReconcile(ctx)
}

func (h *Handlers) maybeResyncProductSchedule(ctx context.Context, pid domain.ProductID) {
	if h.ResyncProductSchedule == nil {
		return
	}
	h.ResyncProductSchedule(ctx, pid)
}

func (h *Handlers) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handlers) routesDoc(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"routes": routeCatalog()})
}

func (h *Handlers) createProduct(w http.ResponseWriter, r *http.Request) {
	var req createProductReq
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if err := req.validate(); err != nil {
		writeError(w, http.StatusBadRequest, "validation", err.Error())
		return
	}
	p, err := h.Product.Register(r.Context(), productapp.RegistrationInput{
		Name:                 req.Name,
		WorkspaceID:          req.WorkspaceID,
		RepoURL:              req.RepoURL,
		RepoClonePath:        strings.TrimSpace(req.RepoClonePath),
		RepoBranch:           req.RepoBranch,
		Description:          req.Description,
		ProgramDocument:      req.ProgramDocument,
		SettingsJSON:         req.SettingsJSON,
		IconURL:              req.IconURL,
		ResearchCadenceSec:   req.ResearchCadenceSec,
		IdeationCadenceSec:   req.IdeationCadenceSec,
		AutomationTier:       req.AutomationTier,
		AutoDispatchEnabled:  req.AutoDispatchEnabled,
		MergePolicyJSON:      strings.TrimSpace(req.MergePolicyJSON),
	})
	if err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	detail, _ := json.Marshal(map[string]string{"name": p.Name})
	h.logOperation(r.Context(), "http", "product.create", "product", string(p.ID), string(detail), p.ID)
	h.maybeReconcileAutopilotSchedule(r.Context())
	writeJSON(w, http.StatusCreated, productToJSON(p))
}

func (h *Handlers) listProducts(w http.ResponseWriter, r *http.Request) {
	list, err := h.Autopilot.Products.ListAll(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	out := make([]any, 0, len(list))
	for i := range list {
		out = append(out, productToJSON(&list[i]))
	}
	writeJSON(w, http.StatusOK, map[string]any{"products": out})
}

func (h *Handlers) patchProduct(w http.ResponseWriter, r *http.Request) {
	var req patchProductReq
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if err := req.validate(); err != nil {
		writeError(w, http.StatusBadRequest, "validation", err.Error())
		return
	}
	id := domain.ProductID(r.PathValue("id"))
	patch := productapp.MetadataPatch{
		Name:                 req.Name,
		RepoURL:              req.RepoURL,
		RepoClonePath:        req.RepoClonePath,
		RepoBranch:           req.RepoBranch,
		Description:          req.Description,
		ProgramDocument:      req.ProgramDocument,
		SettingsJSON:         req.SettingsJSON,
		IconURL:              req.IconURL,
		MergePolicyJSON:      req.MergePolicyJSON,
		ResearchCadenceSec:   req.ResearchCadenceSec,
		IdeationCadenceSec:   req.IdeationCadenceSec,
		AutomationTier:       req.AutomationTier,
		AutoDispatchEnabled:  req.AutoDispatchEnabled,
	}
	p, err := h.Product.PatchMetadata(r.Context(), id, patch)
	if err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, productToJSON(p))
}

func (h *Handlers) getProduct(w http.ResponseWriter, r *http.Request) {
	id := domain.ProductID(r.PathValue("id"))
	p, err := h.Autopilot.Products.ByID(r.Context(), id)
	if err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	m := productToJSON(p)
	if h.MergeQueue != nil {
		if n, qerr := h.MergeQueue.CountPendingByProduct(r.Context(), id); qerr == nil {
			m["merge_queue_pending"] = n
		}
	}
	writeJSON(w, http.StatusOK, m)
}

func (h *Handlers) runResearch(w http.ResponseWriter, r *http.Request) {
	id := domain.ProductID(r.PathValue("id"))
	if err := h.Autopilot.RunResearch(r.Context(), id); err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	p, err := h.Autopilot.Products.ByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, productToJSON(p))
}

func (h *Handlers) runIdeation(w http.ResponseWriter, r *http.Request) {
	id := domain.ProductID(r.PathValue("id"))
	if err := h.Autopilot.RunIdeation(r.Context(), id); err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	p, err := h.Autopilot.Products.ByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, productToJSON(p))
}

func (h *Handlers) listIdeas(w http.ResponseWriter, r *http.Request) {
	id := domain.ProductID(r.PathValue("id"))
	list, err := h.Autopilot.Ideas.ListByProduct(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	out := make([]any, 0, len(list))
	for i := range list {
		out = append(out, ideaToJSON(&list[i]))
	}
	writeJSON(w, http.StatusOK, map[string]any{"ideas": out})
}

func (h *Handlers) listProductTasks(w http.ResponseWriter, r *http.Request) {
	id := domain.ProductID(r.PathValue("id"))
	list, err := h.Task.ListByProduct(r.Context(), id)
	if err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	out := make([]any, 0, len(list))
	for i := range list {
		out = append(out, taskToJSON(&list[i]))
	}
	writeJSON(w, http.StatusOK, map[string]any{"tasks": out})
}

func (h *Handlers) listProductConvoys(w http.ResponseWriter, r *http.Request) {
	id := domain.ProductID(r.PathValue("id"))
	list, err := h.Convoy.ListByProduct(r.Context(), id)
	if err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	out := make([]any, 0, len(list))
	for i := range list {
		out = append(out, convoyToJSON(&list[i]))
	}
	writeJSON(w, http.StatusOK, map[string]any{"convoys": out})
}

func (h *Handlers) swipe(w http.ResponseWriter, r *http.Request) {
	var req swipeReq
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	dec, err := parseSwipe(req.Decision)
	if err != nil {
		writeError(w, http.StatusBadRequest, "validation", err.Error())
		return
	}
	id := domain.IdeaID(r.PathValue("id"))
	if err := h.Autopilot.SubmitSwipe(r.Context(), id, dec); err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	idea, err := h.Autopilot.Ideas.ByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, ideaToJSON(idea))
}

func (h *Handlers) listSwipeHistory(w http.ResponseWriter, r *http.Request) {
	pid := domain.ProductID(r.PathValue("id"))
	if _, err := h.Autopilot.Products.ByID(r.Context(), pid); err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	limit := 100
	if q := strings.TrimSpace(r.URL.Query().Get("limit")); q != "" {
		n, err := strconv.Atoi(q)
		if err != nil || n < 1 {
			writeError(w, http.StatusBadRequest, "validation", "limit must be a positive integer")
			return
		}
		limit = n
	}
	list, err := h.Autopilot.ListSwipeHistory(r.Context(), pid, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	out := make([]any, 0, len(list))
	for i := range list {
		e := &list[i]
		out = append(out, map[string]any{
			"id":         e.ID,
			"idea_id":    string(e.IdeaID),
			"product_id": string(e.ProductID),
			"decision":   e.Decision,
			"created_at": e.CreatedAt.UTC().Format(time.RFC3339Nano),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"swipes": out})
}

func (h *Handlers) listMaybePool(w http.ResponseWriter, r *http.Request) {
	pid := domain.ProductID(r.PathValue("id"))
	if _, err := h.Autopilot.Products.ByID(r.Context(), pid); err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	ids, err := h.Autopilot.MaybePool.ListIdeaIDsByProduct(r.Context(), pid)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	out := make([]any, 0, len(ids))
	for _, iid := range ids {
		idea, err := h.Autopilot.Ideas.ByID(r.Context(), iid)
		if err != nil {
			if mapDomainErr(w, err) {
				return
			}
			writeError(w, http.StatusInternalServerError, "internal", err.Error())
			return
		}
		out = append(out, ideaToJSON(idea))
	}
	writeJSON(w, http.StatusOK, map[string]any{"ideas": out})
}

func (h *Handlers) promoteMaybe(w http.ResponseWriter, r *http.Request) {
	id := domain.IdeaID(r.PathValue("id"))
	if err := h.Autopilot.PromoteMaybe(r.Context(), id); err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	idea, err := h.Autopilot.Ideas.ByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, ideaToJSON(idea))
}

func (h *Handlers) createTask(w http.ResponseWriter, r *http.Request) {
	var req createTaskReq
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if err := req.validate(); err != nil {
		writeError(w, http.StatusBadRequest, "validation", err.Error())
		return
	}
	t, err := h.Task.CreateFromApprovedIdea(r.Context(), domain.IdeaID(req.IdeaID), req.Spec)
	if err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	td, _ := json.Marshal(map[string]string{"idea_id": req.IdeaID})
	h.logOperation(r.Context(), "http", "task.create", "task", string(t.ID), string(td), t.ProductID)
	writeJSON(w, http.StatusCreated, taskToJSON(t))
}

func (h *Handlers) getTask(w http.ResponseWriter, r *http.Request) {
	id := domain.TaskID(r.PathValue("id"))
	t, err := h.Task.Tasks.ByID(r.Context(), id)
	if err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, taskToJSON(t))
}

func (h *Handlers) patchTask(w http.ResponseWriter, r *http.Request) {
	var req patchTaskReq
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if err := req.validate(); err != nil {
		writeError(w, http.StatusBadRequest, "validation", err.Error())
		return
	}
	ctx := r.Context()
	id := domain.TaskID(r.PathValue("id"))
	if req.ClarificationsJSON != nil {
		if err := h.Task.UpdatePlanningArtifacts(ctx, id, *req.ClarificationsJSON); err != nil {
			if mapDomainErr(w, err) {
				return
			}
			writeError(w, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
	}
	if req.Status != nil {
		to, err := domain.ParseTaskStatus(*req.Status)
		if err != nil {
			writeError(w, http.StatusBadRequest, "validation", err.Error())
			return
		}
		reason := ""
		if req.StatusReason != nil {
			reason = *req.StatusReason
		}
		if err := h.Task.SetKanbanStatus(ctx, id, to, reason); err != nil {
			if mapDomainErr(w, err) {
				return
			}
			writeError(w, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
	} else if req.StatusReason != nil {
		if err := h.Task.SetStatusReasonOnly(ctx, id, *req.StatusReason); err != nil {
			if mapDomainErr(w, err) {
				return
			}
			writeError(w, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
	}
	if req.SandboxPath != nil || req.WorktreePath != nil {
		if err := h.Task.PatchWorkspacePaths(ctx, id, req.SandboxPath, req.WorktreePath); err != nil {
			if mapDomainErr(w, err) {
				return
			}
			writeError(w, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
	}
	t, err := h.Task.Tasks.ByID(ctx, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, taskToJSON(t))
}

func (h *Handlers) approvePlan(w http.ResponseWriter, r *http.Request) {
	var req approvePlanReq
	if err := decodeJSON(r, &req); err != nil && !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	id := domain.TaskID(r.PathValue("id"))
	if err := h.Task.ApprovePlan(r.Context(), id, req.Spec); err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	t, err := h.Task.Tasks.ByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, taskToJSON(t))
}

func (h *Handlers) rejectPlan(w http.ResponseWriter, r *http.Request) {
	var req rejectPlanReq
	if err := decodeJSON(r, &req); err != nil && !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	id := domain.TaskID(r.PathValue("id"))
	if err := h.Task.ReturnToPlanning(r.Context(), id, req.StatusReason); err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	t, err := h.Task.Tasks.ByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, taskToJSON(t))
}

func (h *Handlers) openPullRequest(w http.ResponseWriter, r *http.Request) {
	var req openPullRequestReq
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if err := req.validate(); err != nil {
		writeError(w, http.StatusBadRequest, "validation", err.Error())
		return
	}
	id := domain.TaskID(r.PathValue("id"))
	url, prNum, err := h.Task.OpenPullRequest(r.Context(), id, req.HeadBranch, req.Title, req.Body)
	if err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	out := map[string]any{"pr_url": url}
	if prNum > 0 {
		out["pr_number"] = prNum
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handlers) dispatchTask(w http.ResponseWriter, r *http.Request) {
	var req dispatchReq
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	id := domain.TaskID(r.PathValue("id"))
	if err := h.Task.Dispatch(r.Context(), id, req.EstimatedCost); err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	t, err := h.Task.Tasks.ByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	dd, _ := json.Marshal(map[string]float64{"estimated_cost": req.EstimatedCost})
	h.logOperation(r.Context(), "http", "task.dispatch", "task", string(id), string(dd), t.ProductID)
	writeJSON(w, http.StatusOK, taskToJSON(t))
}

func (h *Handlers) checkpoint(w http.ResponseWriter, r *http.Request) {
	var req checkpointReq
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if err := req.validate(); err != nil {
		writeError(w, http.StatusBadRequest, "validation", err.Error())
		return
	}
	id := domain.TaskID(r.PathValue("id"))
	if err := h.Task.RecordCheckpoint(r.Context(), id, req.Payload); err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	t, err := h.Task.Tasks.ByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, taskToJSON(t))
}

func (h *Handlers) nudgeStallTask(w http.ResponseWriter, r *http.Request) {
	var req stallNudgeReq
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<14))
	if err != nil {
		writeError(w, http.StatusBadRequest, "body", err.Error())
		return
	}
	if s := strings.TrimSpace(string(body)); s != "" {
		if err := json.Unmarshal([]byte(s), &req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
	}
	id := domain.TaskID(r.PathValue("id"))
	if err := h.Task.NudgeStall(r.Context(), id, req.Note); err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	t, err := h.Task.Tasks.ByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, taskToJSON(t))
}

func (h *Handlers) completeTask(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := domain.TaskID(r.PathValue("id"))
	if err := h.Task.CompleteWithLiveActivity(ctx, id, "api_task_complete"); err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if !h.Task.UsesLiveActivityTX() {
		h.recordAgentHealthCompletion(ctx, id, "api_task_complete")
	}
	t, err := h.Task.Tasks.ByID(ctx, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, taskToJSON(t))
}

func (h *Handlers) createConvoy(w http.ResponseWriter, r *http.Request) {
	var req createConvoyReq
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if err := req.validate(); err != nil {
		writeError(w, http.StatusBadRequest, "validation", err.Error())
		return
	}
	subs := make([]domain.Subtask, len(req.Subtasks))
	for i := range req.Subtasks {
		deps := make([]domain.SubtaskID, len(req.Subtasks[i].DependsOn))
		for j, d := range req.Subtasks[i].DependsOn {
			deps[j] = domain.SubtaskID(d)
		}
		subs[i] = domain.Subtask{
			ID:        domain.SubtaskID(req.Subtasks[i].ID),
			DependsOn: deps,
			AgentRole: req.Subtasks[i].AgentRole,
		}
	}
	c, err := h.Convoy.Create(r.Context(), domain.TaskID(req.ParentTaskID), domain.ProductID(req.ProductID), subs)
	if err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, convoyToJSON(c))
}

func (h *Handlers) getConvoy(w http.ResponseWriter, r *http.Request) {
	id := domain.ConvoyID(r.PathValue("id"))
	c, err := h.Convoy.Get(r.Context(), id)
	if err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, convoyToJSON(c))
}

func (h *Handlers) dispatchConvoy(w http.ResponseWriter, r *http.Request) {
	var req dispatchReq
	if err := decodeJSON(r, &req); err != nil && !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	id := domain.ConvoyID(r.PathValue("id"))
	if err := h.Convoy.DispatchReady(r.Context(), id, req.EstimatedCost); err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	c, err := h.Convoy.Get(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, convoyToJSON(c))
}

func (h *Handlers) recordCost(w http.ResponseWriter, r *http.Request) {
	var req recordCostReq
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if err := req.validate(); err != nil {
		writeError(w, http.StatusBadRequest, "validation", err.Error())
		return
	}
	if err := h.Cost.Record(r.Context(), domain.ProductID(req.ProductID), domain.TaskID(req.TaskID), req.Amount, req.Note, req.Agent, req.Model); err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"status": "recorded"})
}

func (h *Handlers) listAgents(w http.ResponseWriter, r *http.Request) {
	out := map[string]any{}
	if h.Agent != nil {
		reg, err := h.Agent.List(r.Context(), 200)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal", err.Error())
			return
		}
		regJSON := make([]any, 0, len(reg))
		for i := range reg {
			a := &reg[i]
			row := map[string]any{
				"id": a.ID, "display_name": a.DisplayName, "source": a.Source,
				"external_ref": a.ExternalRef, "created_at": a.CreatedAt.UTC().Format(time.RFC3339Nano),
			}
			if a.ProductID != "" {
				row["product_id"] = string(a.ProductID)
			}
			regJSON = append(regJSON, row)
		}
		out["registry"] = regJSON
	}
	if h.AgentHealth == nil {
		out["items"] = []any{}
		out["stub"] = true
		writeJSON(w, http.StatusOK, out)
		return
	}
	list, err := h.AgentHealth.ListRecent(r.Context(), 200)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	items := make([]any, 0, len(list))
	for i := range list {
		items = append(items, agentHealthToJSON(&list[i], h.agentHeartbeatStale(list[i].LastHeartbeatAt)))
	}
	out["items"] = items
	writeJSON(w, http.StatusOK, out)
}

func (h *Handlers) registerExecutionAgent(w http.ResponseWriter, r *http.Request) {
	if h.Agent == nil {
		writeError(w, http.StatusServiceUnavailable, "not_configured", "agent registry not available")
		return
	}
	var req registerAgentReq
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if err := req.validate(); err != nil {
		writeError(w, http.StatusBadRequest, "validation", err.Error())
		return
	}
	pid := domain.ProductID(strings.TrimSpace(req.ProductID))
	a, err := h.Agent.Register(r.Context(), req.DisplayName, pid, req.Source, req.ExternalRef)
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, executionAgentToJSON(a))
}

func (h *Handlers) listAgentMailbox(w http.ResponseWriter, r *http.Request) {
	if h.Agent == nil {
		writeError(w, http.StatusServiceUnavailable, "not_configured", "agent mailbox not available")
		return
	}
	aid := strings.TrimSpace(r.PathValue("id"))
	limit := 50
	if q := strings.TrimSpace(r.URL.Query().Get("limit")); q != "" {
		n, err := strconv.Atoi(q)
		if err != nil || n < 1 {
			writeError(w, http.StatusBadRequest, "validation", "limit must be a positive integer")
			return
		}
		limit = n
	}
	msgs, err := h.Agent.ListMailbox(r.Context(), aid, limit)
	if err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	out := make([]any, 0, len(msgs))
	for i := range msgs {
		m := &msgs[i]
		row := map[string]any{
			"id": m.ID, "agent_id": m.AgentID, "body": m.Body,
			"created_at": m.CreatedAt.UTC().Format(time.RFC3339Nano),
		}
		if m.TaskID != "" {
			row["task_id"] = string(m.TaskID)
		}
		out = append(out, row)
	}
	writeJSON(w, http.StatusOK, map[string]any{"messages": out})
}

func (h *Handlers) postAgentMailbox(w http.ResponseWriter, r *http.Request) {
	if h.Agent == nil {
		writeError(w, http.StatusServiceUnavailable, "not_configured", "agent mailbox not available")
		return
	}
	var req postAgentMailReq
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if err := req.validate(); err != nil {
		writeError(w, http.StatusBadRequest, "validation", err.Error())
		return
	}
	aid := strings.TrimSpace(r.PathValue("id"))
	if err := h.Agent.PostMailbox(r.Context(), aid, domain.TaskID(strings.TrimSpace(req.TaskID)), req.Body); err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"status": "queued"})
}

func (h *Handlers) listProductResearchCycles(w http.ResponseWriter, r *http.Request) {
	pid := domain.ProductID(r.PathValue("id"))
	limit := 50
	if q := strings.TrimSpace(r.URL.Query().Get("limit")); q != "" {
		n, err := strconv.Atoi(q)
		if err != nil || n < 1 {
			writeError(w, http.StatusBadRequest, "validation", "limit must be a positive integer")
			return
		}
		limit = n
	}
	list, err := h.Autopilot.ListResearchHistory(r.Context(), pid, limit)
	if err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	out := make([]any, 0, len(list))
	for i := range list {
		c := &list[i]
		out = append(out, map[string]any{
			"id": c.ID, "product_id": string(c.ProductID),
			"summary_snapshot": c.SummarySnapshot,
			"created_at":       c.CreatedAt.UTC().Format(time.RFC3339Nano),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"research_cycles": out})
}

func (h *Handlers) getTaskAgentHealth(w http.ResponseWriter, r *http.Request) {
	if h.AgentHealth == nil {
		writeError(w, http.StatusServiceUnavailable, "not_configured", "agent health not available")
		return
	}
	id := domain.TaskID(r.PathValue("id"))
	row, err := h.AgentHealth.ByTask(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	if row == nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"task_id":             string(id),
			"status":              "unknown",
			"detail":              map[string]any{},
			"last_heartbeat_at":   nil,
			"heartbeat_stale":     false,
		})
		return
	}
	writeJSON(w, http.StatusOK, agentHealthToJSON(row, h.agentHeartbeatStale(row.LastHeartbeatAt)))
}

func (h *Handlers) patchTaskAgentHealth(w http.ResponseWriter, r *http.Request) {
	if h.AgentHealth == nil {
		writeError(w, http.StatusServiceUnavailable, "not_configured", "agent health not available")
		return
	}
	var req patchAgentHealthReq
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if err := req.validate(); err != nil {
		writeError(w, http.StatusBadRequest, "validation", err.Error())
		return
	}
	ctx := r.Context()
	id := domain.TaskID(r.PathValue("id"))
	t, err := h.Task.Tasks.ByID(ctx, id)
	if err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	detail := string(req.Detail)
	if detail == "" {
		detail = "{}"
	}
	if !json.Valid([]byte(detail)) {
		writeError(w, http.StatusBadRequest, "validation", "detail must be JSON")
		return
	}
	if err := h.AgentHealth.UpsertHeartbeat(ctx, id, t.ProductID, strings.TrimSpace(req.Status), detail, time.Now().UTC()); err != nil {
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	row, err := h.AgentHealth.ByTask(ctx, id)
	if err != nil || row == nil {
		writeError(w, http.StatusInternalServerError, "internal", "heartbeat persist failed")
		return
	}
	writeJSON(w, http.StatusOK, agentHealthToJSON(row, h.agentHeartbeatStale(row.LastHeartbeatAt)))
}

func (h *Handlers) listProductAgentHealth(w http.ResponseWriter, r *http.Request) {
	if h.AgentHealth == nil {
		writeError(w, http.StatusServiceUnavailable, "not_configured", "agent health not available")
		return
	}
	pid := domain.ProductID(r.PathValue("id"))
	if _, err := h.Autopilot.Products.ByID(r.Context(), pid); err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	limit := 100
	if q := strings.TrimSpace(r.URL.Query().Get("limit")); q != "" {
		n, err := strconv.Atoi(q)
		if err != nil || n < 1 {
			writeError(w, http.StatusBadRequest, "validation", "limit must be a positive integer")
			return
		}
		limit = n
	}
	list, err := h.AgentHealth.ListByProduct(r.Context(), pid, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	items := make([]any, 0, len(list))
	for i := range list {
		items = append(items, agentHealthToJSON(&list[i], h.agentHeartbeatStale(list[i].LastHeartbeatAt)))
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handlers) listStalledTasks(w http.ResponseWriter, r *http.Request) {
	if h.AgentHealth == nil {
		writeError(w, http.StatusServiceUnavailable, "not_configured", "agent health not available")
		return
	}
	ctx := r.Context()
	pid := domain.ProductID(r.PathValue("id"))
	if _, err := h.Autopilot.Products.ByID(ctx, pid); err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	staleSec := h.Config.AgentStaleSec
	if staleSec <= 0 {
		staleSec = 300
	}
	if q := strings.TrimSpace(r.URL.Query().Get("stale_sec")); q != "" {
		n, err := strconv.Atoi(q)
		if err != nil || n < 1 {
			writeError(w, http.StatusBadRequest, "validation", "stale_sec must be a positive integer")
			return
		}
		staleSec = n
	}
	threshold := time.Duration(staleSec) * time.Second

	tasks, err := h.Task.ListByProduct(ctx, pid)
	if err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	stalled := make([]map[string]any, 0)
	for i := range tasks {
		t := &tasks[i]
		if !taskExpectsAgentHeartbeat(t.Status) {
			continue
		}
		row, err := h.AgentHealth.ByTask(ctx, t.ID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal", err.Error())
			return
		}
		var reason string
		var lastAny any
		if row == nil {
			reason = "no_heartbeat"
			lastAny = nil
		} else if time.Since(row.LastHeartbeatAt) > threshold {
			reason = "heartbeat_stale"
			lastAny = row.LastHeartbeatAt.UTC().Format(time.RFC3339Nano)
		} else {
			continue
		}
		stalled = append(stalled, map[string]any{
			"task_id":           string(t.ID),
			"status":            t.Status.String(),
			"reason":            reason,
			"last_heartbeat_at": lastAny,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"product_id": string(pid),
		"stale_sec":  staleSec,
		"stalled":    stalled,
	})
}

func taskExpectsAgentHeartbeat(st domain.TaskStatus) bool {
	switch st {
	case domain.StatusInProgress, domain.StatusTesting, domain.StatusReview, domain.StatusConvoyActive:
		return true
	default:
		return false
	}
}

func (h *Handlers) prepareGitWorktree(w http.ResponseWriter, r *http.Request) {
	if !h.Config.EnableGitWorktrees {
		writeError(w, http.StatusServiceUnavailable, "not_enabled", "set ARMS_ENABLE_GIT_WORKTREES=1 and ARMS_WORKSPACE_ROOT")
		return
	}
	root := strings.TrimSpace(h.Config.WorkspaceRoot)
	if root == "" {
		writeError(w, http.StatusServiceUnavailable, "not_configured", "ARMS_WORKSPACE_ROOT is required")
		return
	}
	var req gitWorktreeReq
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if err := req.validate(); err != nil {
		writeError(w, http.StatusBadRequest, "validation", err.Error())
		return
	}
	branch := strings.TrimSpace(req.Branch)
	if utf8.RuneCountInString(branch) > 200 || !gitBranchTokenRE.MatchString(branch) {
		writeError(w, http.StatusBadRequest, "validation", "branch must match [a-zA-Z0-9/._-]+ and len <= 200")
		return
	}
	ctx := r.Context()
	id := domain.TaskID(r.PathValue("id"))
	t, err := h.Task.Tasks.ByID(ctx, id)
	if err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	p, err := h.Autopilot.Products.ByID(ctx, t.ProductID)
	if err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	mainRepo := strings.TrimSpace(p.RepoClonePath)
	if mainRepo == "" {
		writeError(w, http.StatusBadRequest, "invalid_input", "product.repo_clone_path must be set (PATCH /api/products/{id})")
		return
	}
	wtPath := filepath.Join(root, string(t.ProductID), string(t.ID))
	cleanWT, err := workgit.EnsurePathUnderRoot(root, wtPath)
	if err != nil {
		writeError(w, http.StatusBadRequest, "validation", err.Error())
		return
	}
	execCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	if err := workgit.PrepareGitWorktree(execCtx, h.Config.GitBin, filepath.Clean(mainRepo), cleanWT, branch); err != nil {
		writeError(w, http.StatusBadGateway, "git_error", err.Error())
		return
	}
	pathStr := cleanWT
	if err := h.Task.PatchWorkspacePaths(ctx, id, nil, &pathStr); err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"worktree_path": cleanWT, "branch": branch})
}

func (h *Handlers) recordAgentHealthCompletion(ctx context.Context, taskID domain.TaskID, source string) {
	if h.AgentHealth == nil {
		return
	}
	t, err := h.Task.Tasks.ByID(ctx, taskID)
	if err != nil {
		return
	}
	detail, err := json.Marshal(map[string]string{"source": source})
	if err != nil {
		return
	}
	_ = h.AgentHealth.UpsertHeartbeat(ctx, taskID, t.ProductID, "completed", string(detail), time.Now().UTC())
}

func (h *Handlers) agentHeartbeatStale(last time.Time) bool {
	if h.Config.AgentStaleSec <= 0 || last.IsZero() {
		return false
	}
	return time.Since(last) > time.Duration(h.Config.AgentStaleSec)*time.Second
}

func agentHealthToJSON(h *domain.TaskAgentHealth, stale bool) map[string]any {
	m := map[string]any{
		"task_id":           string(h.TaskID),
		"product_id":        string(h.ProductID),
		"status":            h.Status,
		"heartbeat_stale":   stale,
		"last_heartbeat_at": h.LastHeartbeatAt.UTC().Format(time.RFC3339Nano),
	}
	var detail any
	if strings.TrimSpace(h.DetailJSON) != "" && json.Valid([]byte(h.DetailJSON)) {
		_ = json.Unmarshal([]byte(h.DetailJSON), &detail)
	}
	if detail == nil {
		detail = map[string]any{}
	}
	m["detail"] = detail
	return m
}

func (h *Handlers) openclawStub(w http.ResponseWriter, _ *http.Request) {
	writeError(w, http.StatusNotImplemented, "not_implemented", "openclaw proxy not implemented; use AgentGateway adapter")
}

func (h *Handlers) workspacesView(w http.ResponseWriter, r *http.Request) {
	if h.WorkspacePorts == nil || h.MergeQueue == nil {
		writeJSON(w, http.StatusOK, map[string]any{"ports": []any{}, "merge_queue_pending": 0, "stub": true})
		return
	}
	ctx := r.Context()
	portsList, err := h.WorkspacePorts.ListAll(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	n, err := h.MergeQueue.CountPending(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	items := make([]map[string]any, 0, len(portsList))
	for _, a := range portsList {
		items = append(items, map[string]any{
			"port":         a.Port,
			"product_id":   string(a.ProductID),
			"task_id":      string(a.TaskID),
			"allocated_at": a.AllocatedAt.UTC().Format(time.RFC3339Nano),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ports":               items,
		"merge_queue_pending": n,
	})
}

func (h *Handlers) enqueueMergeQueue(w http.ResponseWriter, r *http.Request) {
	if h.MergeQueue == nil {
		writeError(w, http.StatusServiceUnavailable, "not_configured", "merge queue not available")
		return
	}
	id := domain.TaskID(r.PathValue("id"))
	t, err := h.Task.Tasks.ByID(r.Context(), id)
	if err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	if err := h.MergeQueue.Enqueue(r.Context(), t.ProductID, t.ID, time.Now().UTC()); err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	h.logOperation(r.Context(), "http", "merge_queue.enqueue", "task", string(id), "{}", t.ProductID)
	writeJSON(w, http.StatusCreated, map[string]string{"status": "queued"})
}

func (h *Handlers) completeMergeQueue(w http.ResponseWriter, r *http.Request) {
	if h.MergeQueue == nil {
		writeError(w, http.StatusServiceUnavailable, "not_configured", "merge queue not available")
		return
	}
	id := domain.TaskID(r.PathValue("id"))
	t, err := h.Task.Tasks.ByID(r.Context(), id)
	if err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	skip := strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("skip_ship")), "1") ||
		strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("skip_real_merge")), "1")
	if h.MergeShip != nil {
		err = h.MergeShip.Complete(r.Context(), id, skip)
	} else {
		err = h.MergeQueue.CompletePendingForTask(r.Context(), id)
	}
	if err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	detail, _ := json.Marshal(map[string]any{"skip_ship": skip})
	h.logOperation(r.Context(), "http", "merge_queue.complete", "task", string(id), string(detail), t.ProductID)
	writeJSON(w, http.StatusOK, map[string]string{"status": "completed"})
}

func (h *Handlers) resolveMergeQueueByTask(w http.ResponseWriter, r *http.Request) {
	if h.MergeShip == nil {
		writeError(w, http.StatusServiceUnavailable, "not_configured", "merge ship not available")
		return
	}
	id := domain.TaskID(r.PathValue("id"))
	t, err := h.Task.Tasks.ByID(r.Context(), id)
	if err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	var req mergeQueueResolveReq
	if b, rerr := io.ReadAll(r.Body); rerr != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", rerr.Error())
		return
	} else if s := strings.TrimSpace(string(b)); s != "" {
		if err := json.Unmarshal([]byte(s), &req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
	}
	action := strings.ToLower(strings.TrimSpace(req.Action))
	if action == "" {
		action = "retry_merge"
	}
	var skip bool
	switch action {
	case "retry_merge":
		skip = false
	case "skip_ship":
		skip = true
	default:
		writeError(w, http.StatusBadRequest, "validation", `action must be "retry_merge" or "skip_ship"`)
		return
	}
	if err := h.MergeShip.Complete(r.Context(), id, skip); err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	detail, _ := json.Marshal(map[string]any{"action": action})
	h.logOperation(r.Context(), "http", "merge_queue.resolve", "task", string(id), string(detail), t.ProductID)
	writeJSON(w, http.StatusOK, map[string]string{"status": "resolved"})
}

func (h *Handlers) resolveMergeQueueByRow(w http.ResponseWriter, r *http.Request) {
	if h.MergeShip == nil || h.MergeQueue == nil {
		writeError(w, http.StatusServiceUnavailable, "not_configured", "merge queue not available")
		return
	}
	rowID, err := strconv.ParseInt(strings.TrimSpace(r.PathValue("id")), 10, 64)
	if err != nil || rowID < 1 {
		writeError(w, http.StatusBadRequest, "validation", "merge queue id must be a positive integer")
		return
	}
	ctx := r.Context()
	e, err := h.MergeQueue.GetPendingMergeQueueEntry(ctx, rowID)
	if err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	var req mergeQueueResolveReq
	if b, rerr := io.ReadAll(r.Body); rerr != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", rerr.Error())
		return
	} else if s := strings.TrimSpace(string(b)); s != "" {
		if err := json.Unmarshal([]byte(s), &req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
	}
	action := strings.ToLower(strings.TrimSpace(req.Action))
	if action == "" {
		action = "retry_merge"
	}
	var skip bool
	switch action {
	case "retry_merge":
		skip = false
	case "skip_ship":
		skip = true
	default:
		writeError(w, http.StatusBadRequest, "validation", `action must be "retry_merge" or "skip_ship"`)
		return
	}
	if err := h.MergeShip.Complete(ctx, e.TaskID, skip); err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	detail, _ := json.Marshal(map[string]any{"merge_queue_row_id": rowID, "action": action})
	h.logOperation(ctx, "http", "merge_queue.resolve", "merge_queue_row", strconv.FormatInt(rowID, 10), string(detail), e.ProductID)
	writeJSON(w, http.StatusOK, map[string]string{"status": "resolved"})
}

func (h *Handlers) listProductMergeQueue(w http.ResponseWriter, r *http.Request) {
	if h.MergeQueue == nil {
		writeError(w, http.StatusServiceUnavailable, "not_configured", "merge queue not available")
		return
	}
	pid := domain.ProductID(r.PathValue("id"))
	if _, err := h.Autopilot.Products.ByID(r.Context(), pid); err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	limit := 50
	if q := strings.TrimSpace(r.URL.Query().Get("limit")); q != "" {
		n, err := strconv.Atoi(q)
		if err != nil || n < 1 {
			writeError(w, http.StatusBadRequest, "validation", "limit must be a positive integer")
			return
		}
		limit = n
	}
	ctx := r.Context()
	list, err := h.MergeQueue.ListPendingByProduct(ctx, pid, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	pendingTotal, _ := h.MergeQueue.CountPendingByProduct(ctx, pid)
	out := make([]any, 0, len(list))
	for i := range list {
		e := &list[i]
		row := map[string]any{
			"id":         e.ID,
			"product_id": string(e.ProductID),
			"task_id":    string(e.TaskID),
			"status":     e.Status,
			"created_at": e.CreatedAt.UTC().Format(time.RFC3339Nano),
		}
		if e.LeaseOwner != "" {
			row["lease_owner"] = e.LeaseOwner
		}
		if !e.LeaseExpiresAt.IsZero() {
			row["lease_expires_at"] = e.LeaseExpiresAt.UTC().Format(time.RFC3339Nano)
		}
		if e.MergeShipState != "" {
			row["merge_ship_state"] = string(e.MergeShipState)
		}
		if e.MergedSHA != "" {
			row["merged_sha"] = e.MergedSHA
		}
		if e.MergeError != "" {
			row["merge_error"] = e.MergeError
		}
		if strings.TrimSpace(e.ConflictFilesJSON) != "" {
			var cf []string
			if json.Unmarshal([]byte(e.ConflictFilesJSON), &cf) == nil {
				row["conflict_files"] = cf
			}
		}
		row["queue_position"] = i + 1
		if i == 0 {
			row["is_head"] = true
		} else {
			row["is_head"] = false
		}
		out = append(out, row)
	}
	resp := map[string]any{"merge_queue": out, "pending_count": pendingTotal}
	if len(list) > 0 {
		resp["head_task_id"] = string(list[0].TaskID)
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handlers) cancelMergeQueue(w http.ResponseWriter, r *http.Request) {
	if h.MergeQueue == nil {
		writeError(w, http.StatusServiceUnavailable, "not_configured", "merge queue not available")
		return
	}
	id := domain.TaskID(r.PathValue("id"))
	t, err := h.Task.Tasks.ByID(r.Context(), id)
	if err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	if err := h.MergeQueue.CancelPendingForTask(r.Context(), id); err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	h.logOperation(r.Context(), "http", "merge_queue.cancel", "task", string(id), "{}", t.ProductID)
	writeJSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
}

func (h *Handlers) allocateWorkspacePort(w http.ResponseWriter, r *http.Request) {
	if h.WorkspacePorts == nil {
		writeError(w, http.StatusServiceUnavailable, "not_configured", "workspace ports not available")
		return
	}
	var req allocatePortReq
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if err := req.validate(); err != nil {
		writeError(w, http.StatusBadRequest, "validation", err.Error())
		return
	}
	port, err := h.WorkspacePorts.Allocate(r.Context(), domain.ProductID(req.ProductID), domain.TaskID(req.TaskID), time.Now().UTC())
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "ports_exhausted", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"port": port})
}

func (h *Handlers) releaseWorkspacePort(w http.ResponseWriter, r *http.Request) {
	if h.WorkspacePorts == nil {
		writeError(w, http.StatusServiceUnavailable, "not_configured", "workspace ports not available")
		return
	}
	pstr := r.PathValue("port")
	p, err := strconv.Atoi(strings.TrimSpace(pstr))
	if err != nil || p < 4200 || p > 4299 {
		writeError(w, http.StatusBadRequest, "validation", "port must be 4200-4299")
		return
	}
	if err := h.WorkspacePorts.Release(r.Context(), p); err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "released"})
}

func (h *Handlers) patchProductCostCaps(w http.ResponseWriter, r *http.Request) {
	var req patchCostCapsReq
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if err := req.validate(); err != nil {
		writeError(w, http.StatusBadRequest, "validation", err.Error())
		return
	}
	pid := domain.ProductID(r.PathValue("id"))
	if err := h.Cost.PatchCaps(r.Context(), pid, req.DailyCap, req.MonthlyCap, req.CumulativeCap); err != nil {
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handlers) productCostBreakdown(w http.ResponseWriter, r *http.Request) {
	pid := domain.ProductID(r.PathValue("id"))
	from, to, err := parseTimeRangeQuery(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "validation", err.Error())
		return
	}
	if _, err := h.Autopilot.Products.ByID(r.Context(), pid); err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	out, err := h.Cost.Breakdown(r.Context(), pid, from, to)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handlers) listTaskCheckpoints(w http.ResponseWriter, r *http.Request) {
	id := domain.TaskID(r.PathValue("id"))
	limit := 50
	if s := strings.TrimSpace(r.URL.Query().Get("limit")); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 && n <= 500 {
			limit = n
		}
	}
	list, err := h.Task.ListCheckpointHistory(r.Context(), id, limit)
	if err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	items := make([]map[string]any, 0, len(list))
	for _, e := range list {
		items = append(items, map[string]any{
			"id":         e.ID,
			"task_id":    string(e.TaskID),
			"payload":    e.Payload,
			"created_at": e.CreatedAt.UTC().Format(time.RFC3339Nano),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"checkpoints": items})
}

func (h *Handlers) restoreTaskCheckpoint(w http.ResponseWriter, r *http.Request) {
	var req restoreCheckpointReq
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if err := req.validate(); err != nil {
		writeError(w, http.StatusBadRequest, "validation", err.Error())
		return
	}
	id := domain.TaskID(r.PathValue("id"))
	if err := h.Task.RestoreCheckpoint(r.Context(), id, req.HistoryID); err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	t, err := h.Task.Tasks.ByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, taskToJSON(t))
}

func (h *Handlers) settingsStub(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{})
}

func (h *Handlers) agentCompletionWebhook(w http.ResponseWriter, r *http.Request) {
	if h.Config.WebhookSecret == "" {
		writeError(w, http.StatusServiceUnavailable, "not_configured", "WEBHOOK_SECRET is not set")
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, "body", err.Error())
		return
	}
	sig := r.Header.Get(webhookSigHeader)
	if !hmacSHA256Equal(h.Config.WebhookSecret, body, sig) {
		writeError(w, http.StatusUnauthorized, "invalid_signature", "HMAC verification failed")
		return
	}
	var req agentCompletionReq
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if err := req.validate(); err != nil {
		writeError(w, http.StatusBadRequest, "validation", err.Error())
		return
	}
	ctx := r.Context()
	tid := domain.TaskID(req.TaskID)
	if strings.TrimSpace(req.ConvoyID) != "" && strings.TrimSpace(req.SubtaskID) != "" {
		if h.Convoy == nil {
			writeError(w, http.StatusServiceUnavailable, "not_configured", "convoy service not available")
			return
		}
		if err := h.Convoy.CompleteSubtask(ctx, domain.ConvoyID(strings.TrimSpace(req.ConvoyID)), domain.SubtaskID(strings.TrimSpace(req.SubtaskID)), tid); err != nil {
			if mapDomainErr(w, err) {
				return
			}
			writeError(w, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		return
	}
	if err := h.Task.CompleteWithLiveActivity(ctx, tid, "agent_completion_webhook"); err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if !h.Task.UsesLiveActivityTX() {
		h.recordAgentHealthCompletion(ctx, tid, "agent_completion_webhook")
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func hmacSHA256Equal(secret string, body []byte, sigHex string) bool {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	want := mac.Sum(nil)
	got, err := hex.DecodeString(strings.TrimSpace(sigHex))
	if err != nil || len(got) != len(want) {
		return false
	}
	return subtle.ConstantTimeCompare(want, got) == 1
}

func (h *Handlers) liveSSE(w http.ResponseWriter, r *http.Request) {
	if h.Live == nil {
		writeError(w, http.StatusInternalServerError, "streaming", "live activity stream not configured")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	fl, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming", "response writer does not flush")
		return
	}
	enc := func(v any) {
		b, _ := json.Marshal(v)
		_, _ = fmt.Fprintf(w, "data: %s\n\n", b)
		fl.Flush()
	}
	filterProduct := strings.TrimSpace(r.URL.Query().Get("product_id"))
	enc(map[string]any{"event": "hello", "ts": time.Now().UTC().Format(time.RFC3339Nano)})
	ch, unsub := h.Live.Subscribe()
	defer unsub()
	tick := time.NewTicker(25 * time.Second)
	defer tick.Stop()
	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case payload := <-ch:
			if filterProduct != "" && !ssePayloadMatchesProduct(filterProduct, payload) {
				continue
			}
			_, _ = fmt.Fprintf(w, "data: %s\n\n", payload)
			fl.Flush()
		case <-tick.C:
			_, _ = fmt.Fprintf(w, ": ping\n\n")
			fl.Flush()
		}
	}
}

func ssePayloadMatchesProduct(wantProduct string, payload []byte) bool {
	var m struct {
		ProductID string `json:"product_id"`
	}
	if json.Unmarshal(payload, &m) != nil || m.ProductID == "" {
		return true
	}
	return m.ProductID == wantProduct
}

func parseTimeRangeQuery(r *http.Request) (from, to time.Time, err error) {
	if s := strings.TrimSpace(r.URL.Query().Get("from")); s != "" {
		from, err = time.Parse(time.RFC3339Nano, s)
		if err != nil {
			from, err = time.Parse(time.RFC3339, s)
		}
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid from time")
		}
	}
	if s := strings.TrimSpace(r.URL.Query().Get("to")); s != "" {
		to, err = time.Parse(time.RFC3339Nano, s)
		if err != nil {
			to, err = time.Parse(time.RFC3339, s)
		}
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid to time")
		}
	}
	return from, to, nil
}

// JSON views

func productToJSON(p *domain.Product) map[string]any {
	pol := domain.ParseMergePolicy(p.MergePolicyJSON)
	gates := domain.EffectiveMergeExecutionGates(p, pol)
	polOut := map[string]any{
		"merge_method":                 pol.MergeMethod,
		"require_approved_review":      gates.RequireApprovedReview,
		"require_clean_mergeable":      gates.RequireCleanMergeable,
	}
	if o := strings.TrimSpace(pol.MergeBackendOverride); o != "" {
		polOut["merge_backend_override"] = o
	}
	m := map[string]any{
		"id":                    string(p.ID),
		"name":                  p.Name,
		"stage":                 p.Stage.String(),
		"research_summary":      p.ResearchSummary,
		"workspace_id":          p.WorkspaceID,
		"repo_url":              p.RepoURL,
		"repo_clone_path":       p.RepoClonePath,
		"repo_branch":           p.RepoBranch,
		"description":           p.Description,
		"program_document":      p.ProgramDocument,
		"settings_json":         p.SettingsJSON,
		"icon_url":              p.IconURL,
		"research_cadence_sec":  p.ResearchCadenceSec,
		"ideation_cadence_sec":  p.IdeationCadenceSec,
		"automation_tier":       p.AutomationTier.String(),
		"auto_dispatch_enabled": p.AutoDispatchEnabled,
		"preference_model_json": p.PreferenceModelJSON,
		"merge_policy_json":     p.MergePolicyJSON,
		"merge_policy":          polOut,
		"updated_at":            p.UpdatedAt.Format(time.RFC3339Nano),
	}
	if !p.LastAutoResearchAt.IsZero() {
		m["last_auto_research_at"] = p.LastAutoResearchAt.Format(time.RFC3339Nano)
	}
	if !p.LastAutoIdeationAt.IsZero() {
		m["last_auto_ideation_at"] = p.LastAutoIdeationAt.Format(time.RFC3339Nano)
	}
	return m
}

func ideaToJSON(i *domain.Idea) map[string]any {
	return map[string]any{
		"id":          string(i.ID),
		"product_id":  string(i.ProductID),
		"title":       i.Title,
		"description": i.Description,
		"impact":      i.Impact,
		"feasibility": i.Feasibility,
		"reasoning":   i.Reasoning,
		"decided":     i.Decided,
		"decision":    swipeString(i.Decision),
		"created_at":  i.CreatedAt.Format(time.RFC3339Nano),
	}
}

func swipeString(d domain.SwipeDecision) string {
	switch d {
	case domain.DecisionPass:
		return "pass"
	case domain.DecisionMaybe:
		return "maybe"
	case domain.DecisionYes:
		return "yes"
	case domain.DecisionNow:
		return "now"
	default:
		return ""
	}
}

func taskToJSON(t *domain.Task) map[string]any {
	m := map[string]any{
		"id":                  string(t.ID),
		"product_id":          string(t.ProductID),
		"idea_id":             string(t.IdeaID),
		"spec":                t.Spec,
		"status":              t.Status.String(),
		"status_reason":       t.StatusReason,
		"plan_approved":       t.PlanApproved,
		"clarifications_json": t.ClarificationsJSON,
		"checkpoint":          t.Checkpoint,
		"external_ref":        t.ExternalRef,
		"sandbox_path":        t.SandboxPath,
		"worktree_path":       t.WorktreePath,
		"created_at":          t.CreatedAt.Format(time.RFC3339Nano),
		"updated_at":          t.UpdatedAt.Format(time.RFC3339Nano),
	}
	if strings.TrimSpace(t.PullRequestURL) != "" {
		m["pull_request_url"] = t.PullRequestURL
	}
	if t.PullRequestNumber > 0 {
		m["pull_request_number"] = t.PullRequestNumber
	}
	if strings.TrimSpace(t.PullRequestHeadBranch) != "" {
		m["pull_request_head_branch"] = t.PullRequestHeadBranch
	}
	return m
}

func (h *Handlers) logOperation(ctx context.Context, actor, action, resType, resID, detailJSON string, pid domain.ProductID) {
	if h.OperationsLog == nil {
		return
	}
	if detailJSON == "" {
		detailJSON = "{}"
	}
	_ = h.OperationsLog.Append(ctx, domain.OperationLogEntry{
		Actor: actor, Action: action, ResourceType: resType, ResourceID: resID,
		DetailJSON: detailJSON, ProductID: pid,
	})
}

func (h *Handlers) getProductPreferenceModel(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	pid := domain.ProductID(r.PathValue("id"))
	p, err := h.Autopilot.Products.ByID(ctx, pid)
	if err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	modelJSON := strings.TrimSpace(p.PreferenceModelJSON)
	if modelJSON == "" {
		modelJSON = "[]"
	}
	source := "legacy"
	updated := p.UpdatedAt
	if mj, at, ok, err := h.PrefModel.Get(ctx, pid); err != nil {
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	} else if ok {
		modelJSON = strings.TrimSpace(mj)
		if modelJSON == "" {
			modelJSON = "{}"
		}
		source = "preference_models"
		updated = at
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"product_id": string(pid),
		"model_json": modelJSON,
		"source":     source,
		"updated_at": updated.UTC().Format(time.RFC3339Nano),
	})
}

func (h *Handlers) putProductPreferenceModel(w http.ResponseWriter, r *http.Request) {
	var req putPreferenceModelReq
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if err := req.validate(); err != nil {
		writeError(w, http.StatusBadRequest, "validation", err.Error())
		return
	}
	ctx := r.Context()
	pid := domain.ProductID(r.PathValue("id"))
	if _, err := h.Autopilot.Products.ByID(ctx, pid); err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	if err := h.PrefModel.Upsert(ctx, pid, strings.TrimSpace(req.ModelJSON), time.Now().UTC()); err != nil {
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	h.logOperation(ctx, "http", "preference_model.upsert", "product", string(pid), "{}", pid)
	mj, at, _, err := h.PrefModel.Get(ctx, pid)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"product_id": string(pid),
		"model_json": mj,
		"source":     "preference_models",
		"updated_at": at.UTC().Format(time.RFC3339Nano),
	})
}

func (h *Handlers) listOperationsLog(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if q := strings.TrimSpace(r.URL.Query().Get("limit")); q != "" {
		n, err := strconv.Atoi(q)
		if err != nil || n < 1 {
			writeError(w, http.StatusBadRequest, "validation", "limit must be a positive integer")
			return
		}
		limit = n
	}
	var pidPtr *domain.ProductID
	if ps := strings.TrimSpace(r.URL.Query().Get("product_id")); ps != "" {
		p := domain.ProductID(ps)
		pidPtr = &p
	}
	var since *time.Time
	if qs := strings.TrimSpace(r.URL.Query().Get("since")); qs != "" {
		t, err := time.Parse(time.RFC3339Nano, qs)
		if err != nil {
			t, err = time.Parse(time.RFC3339, qs)
		}
		if err != nil {
			writeError(w, http.StatusBadRequest, "validation", "since must be RFC3339 or RFC3339Nano")
			return
		}
		u := t.UTC()
		since = &u
	}
	f := ports.OperationsLogFilter{
		Limit:        limit,
		ProductID:    pidPtr,
		Action:       strings.TrimSpace(r.URL.Query().Get("action")),
		ResourceType: strings.TrimSpace(r.URL.Query().Get("resource_type")),
		Since:        since,
	}
	list, err := h.OperationsLog.List(r.Context(), f)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	out := make([]any, 0, len(list))
	for i := range list {
		e := &list[i]
		m := map[string]any{
			"id":             e.ID,
			"created_at":     e.CreatedAt.UTC().Format(time.RFC3339Nano),
			"actor":          e.Actor,
			"action":         e.Action,
			"resource_type":  e.ResourceType,
			"resource_id":    e.ResourceID,
			"detail_json":    e.DetailJSON,
		}
		if e.ProductID != "" {
			m["product_id"] = string(e.ProductID)
		}
		out = append(out, m)
	}
	writeJSON(w, http.StatusOK, map[string]any{"entries": out})
}

func (h *Handlers) getProductSchedule(w http.ResponseWriter, r *http.Request) {
	pid := domain.ProductID(r.PathValue("id"))
	row, err := h.Autopilot.GetProductSchedule(r.Context(), pid)
	if err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	out := map[string]any{
		"product_id":      string(pid),
		"enabled":         row.Enabled,
		"spec_json":       row.SpecJSON,
		"cron_expr":       row.CronExpr,
		"delay_seconds":   row.DelaySeconds,
		"asynq_task_id":   row.AsynqTaskID,
	}
	if row.LastEnqueuedAt != nil {
		out["last_enqueued_at"] = row.LastEnqueuedAt.UTC().Format(time.RFC3339Nano)
	}
	if row.NextScheduledAt != nil {
		out["next_scheduled_at"] = row.NextScheduledAt.UTC().Format(time.RFC3339Nano)
	}
	if !row.UpdatedAt.IsZero() {
		out["updated_at"] = row.UpdatedAt.UTC().Format(time.RFC3339Nano)
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handlers) patchProductSchedule(w http.ResponseWriter, r *http.Request) {
	var req patchProductScheduleReq
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if err := req.validate(); err != nil {
		writeError(w, http.StatusBadRequest, "validation", err.Error())
		return
	}
	ctx := r.Context()
	pid := domain.ProductID(r.PathValue("id"))
	cur, err := h.Autopilot.GetProductSchedule(ctx, pid)
	if err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	enabled := cur.Enabled
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	spec := cur.SpecJSON
	if req.SpecJSON != nil {
		spec = strings.TrimSpace(*req.SpecJSON)
		if spec == "" {
			spec = "{}"
		}
	}
	cronExpr := cur.CronExpr
	if req.CronExpr != nil {
		cronExpr = strings.TrimSpace(*req.CronExpr)
	}
	delaySec := cur.DelaySeconds
	if req.DelaySeconds != nil {
		delaySec = *req.DelaySeconds
		if delaySec < 0 {
			writeError(w, http.StatusBadRequest, "validation", "delay_seconds must be >= 0")
			return
		}
	}
	row := &domain.ProductSchedule{
		ProductID:    pid,
		Enabled:      enabled,
		SpecJSON:     spec,
		CronExpr:     cronExpr,
		DelaySeconds: delaySec,
	}
	timingPatch := req.CronExpr != nil || req.DelaySeconds != nil
	if timingPatch || !enabled {
		row.AsynqTaskID = ""
		row.LastEnqueuedAt = nil
		row.NextScheduledAt = nil
	} else {
		row.AsynqTaskID = cur.AsynqTaskID
		row.LastEnqueuedAt = cur.LastEnqueuedAt
		row.NextScheduledAt = cur.NextScheduledAt
	}
	row, err = h.Autopilot.SaveProductSchedule(ctx, row)
	if err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	h.logOperation(ctx, "http", "product_schedule.upsert", "product", string(pid), "{}", pid)
	h.maybeReconcileAutopilotSchedule(ctx)
	h.maybeResyncProductSchedule(ctx, pid)
	out := map[string]any{
		"product_id":      string(pid),
		"enabled":         row.Enabled,
		"spec_json":       row.SpecJSON,
		"cron_expr":       row.CronExpr,
		"delay_seconds":   row.DelaySeconds,
		"asynq_task_id":   row.AsynqTaskID,
		"updated_at":      row.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
	if row.LastEnqueuedAt != nil {
		out["last_enqueued_at"] = row.LastEnqueuedAt.UTC().Format(time.RFC3339Nano)
	}
	if row.NextScheduledAt != nil {
		out["next_scheduled_at"] = row.NextScheduledAt.UTC().Format(time.RFC3339Nano)
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handlers) postRecomputePreferenceModel(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	pid := domain.ProductID(r.PathValue("id"))
	limit := 500
	if q := strings.TrimSpace(r.URL.Query().Get("limit")); q != "" {
		n, err := strconv.Atoi(q)
		if err != nil || n < 1 {
			writeError(w, http.StatusBadRequest, "validation", "limit must be a positive integer")
			return
		}
		limit = n
	}
	jsonStr, err := h.Autopilot.RecomputePreferenceModelFromSwipes(ctx, pid, limit)
	if err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	h.logOperation(ctx, "http", "preference_model.recompute", "product", string(pid), "{}", pid)
	writeJSON(w, http.StatusOK, map[string]any{
		"product_id": string(pid),
		"model_json": jsonStr,
		"source":     "preference_models",
	})
}

func (h *Handlers) listConvoyMail(w http.ResponseWriter, r *http.Request) {
	id := domain.ConvoyID(r.PathValue("id"))
	limit := 100
	if q := strings.TrimSpace(r.URL.Query().Get("limit")); q != "" {
		n, err := strconv.Atoi(q)
		if err != nil || n < 1 {
			writeError(w, http.StatusBadRequest, "validation", "limit must be a positive integer")
			return
		}
		limit = n
	}
	list, err := h.Convoy.ListMail(r.Context(), id, limit)
	if err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	out := make([]any, 0, len(list))
	for i := range list {
		m := list[i]
		out = append(out, map[string]any{
			"id": m.ID, "convoy_id": string(m.ConvoyID), "subtask_id": string(m.SubtaskID),
			"body": m.Body, "created_at": m.CreatedAt.UTC().Format(time.RFC3339Nano),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"messages": out})
}

func (h *Handlers) postConvoyMail(w http.ResponseWriter, r *http.Request) {
	var req postConvoyMailReq
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if err := req.validate(); err != nil {
		writeError(w, http.StatusBadRequest, "validation", err.Error())
		return
	}
	ctx := r.Context()
	id := domain.ConvoyID(r.PathValue("id"))
	if err := h.Convoy.PostMail(ctx, id, domain.SubtaskID(req.SubtaskID), req.Body); err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	h.logOperation(ctx, "http", "convoy.mail.append", "convoy", string(id), `{}`, "")
	writeJSON(w, http.StatusCreated, map[string]any{"status": "ok"})
}

func convoyToJSON(c *domain.Convoy) map[string]any {
	subs := make([]map[string]any, len(c.Subtasks))
	for i := range c.Subtasks {
		deps := make([]string, len(c.Subtasks[i].DependsOn))
		for j, d := range c.Subtasks[i].DependsOn {
			deps[j] = string(d)
		}
		subs[i] = map[string]any{
			"id":              string(c.Subtasks[i].ID),
			"agent_role":      c.Subtasks[i].AgentRole,
			"depends_on":      deps,
			"dispatched":      c.Subtasks[i].Dispatched,
			"completed":       c.Subtasks[i].Completed,
			"external_ref":    c.Subtasks[i].ExternalRef,
			"last_checkpoint": c.Subtasks[i].LastCheckpoint,
		}
	}
	return map[string]any{
		"id":         string(c.ID),
		"product_id": string(c.ProductID),
		"parent_id":  string(c.ParentID),
		"subtasks":   subs,
		"created_at": c.CreatedAt.Format(time.RFC3339Nano),
	}
}

func executionAgentToJSON(a *domain.ExecutionAgent) map[string]any {
	m := map[string]any{
		"id": a.ID, "display_name": a.DisplayName, "source": a.Source,
		"external_ref": a.ExternalRef, "created_at": a.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
	if a.ProductID != "" {
		m["product_id"] = string(a.ProductID)
	}
	return m
}
