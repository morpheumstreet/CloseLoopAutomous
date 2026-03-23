package task

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/closeloopautomous/arms/internal/domain"
	"github.com/closeloopautomous/arms/internal/ports"
)

// Service owns task lifecycle up to dispatch; execution is delegated to AgentGateway.
type Service struct {
	Tasks          ports.TaskRepository
	Products       ports.ProductRepository
	Ideas          ports.IdeaRepository
	Gateway        ports.AgentGateway
	Budget         ports.BudgetPolicy
	Checkpt        ports.CheckpointRepository
	Clock          ports.Clock
	IDs            ports.IdentityGenerator
	Events         ports.LiveActivityPublisher // optional: live activity / outbox
	LiveTX         ports.LiveActivityTX        // optional: SQLite same-transaction outbox with domain writes
	Gate           *ProductGate                // optional: per-product mutex (e.g. completion)
	Ship           ports.PullRequestPublisher  // GitHub / noop
	AgentHealth    ports.AgentHealthRepository // optional: stall nudge heartbeats
	MergeShip      ports.MergeQueueShipper     // optional: auto merge-queue completion on task done (full_auto always; semi_auto when GitHub gates pass)
	AutoStallNudge AutoStallNudgeSettings      // optional: periodic auto stall nudge (#83); zero value disables
	ExecAgents     ports.ExecutionAgentRegistry // optional: execution agent registry (#107 binding + auto-reassign)
	AutoStallReassign AutoStallReassignSettings // optional: auto re-dispatch to another agent on stall (#107)
	// KnowledgeAutoIngest optional (#90); after successful task completion, best-effort knowledge row (summary may be empty).
	KnowledgeAutoIngest func(ctx context.Context, task *domain.Task, source string, knowledgeSummary string)
}

// taskAndActiveProduct loads the task and its product. The product must exist and not be soft-deleted.
func (s *Service) taskAndActiveProduct(ctx context.Context, taskID domain.TaskID) (*domain.Task, *domain.Product, error) {
	raw, err := s.Tasks.ByID(ctx, taskID)
	if err != nil {
		return nil, nil, err
	}
	p, err := s.Products.ByID(ctx, raw.ProductID)
	if err != nil {
		return nil, nil, err
	}
	return raw, p, nil
}

func (s *Service) taskWithActiveProduct(ctx context.Context, taskID domain.TaskID) (*domain.Task, error) {
	t, _, err := s.taskAndActiveProduct(ctx, taskID)
	return t, err
}

// TaskByIDForAPI returns a task only when its product is active (soft-delete cascade for HTTP reads).
func (s *Service) TaskByIDForAPI(ctx context.Context, id domain.TaskID) (*domain.Task, error) {
	return s.taskWithActiveProduct(ctx, id)
}

// CreateFromApprovedIdea starts the Kanban in planning until ApprovePlan moves to inbox.
func (s *Service) CreateFromApprovedIdea(ctx context.Context, ideaID domain.IdeaID, spec string) (*domain.Task, error) {
	idea, err := s.Ideas.ByID(ctx, ideaID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			extra := ""
			if strings.HasPrefix(strings.ToLower(strings.TrimSpace(string(ideaID))), "initial-") {
				extra = ` On this server, new ideas get ids "idea-1", "idea-2", … (see GET …/ideas → "id"). There is no "initial-*" prefix unless you inserted that row yourself.`
			}
			return nil, fmt.Errorf(
				`%w: no idea with id %q — use GET /api/products/{product_id}/ideas and send JSON field "id" as idea_id (not "task_id" or product id).%s`,
				domain.ErrNotFound, ideaID, extra,
			)
		}
		return nil, err
	}
	if !idea.Decided || !idea.Decision.Approved() {
		return nil, fmt.Errorf("%w: idea not approved", domain.ErrInvalidTransition)
	}
	if err := ports.RequireActiveProduct(ctx, s.Products, idea.ProductID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, fmt.Errorf("%w: product %q for this idea is missing or soft-deleted", domain.ErrNotFound, idea.ProductID)
		}
		return nil, err
	}
	now := s.Clock.Now()
	t := &domain.Task{
		ID:           s.IDs.NewTaskID(),
		ProductID:    idea.ProductID,
		IdeaID:       ideaID,
		Spec:         spec,
		Status:       domain.StatusPlanning,
		PlanApproved: false,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := s.Tasks.Save(ctx, t); err != nil {
		return nil, err
	}
	idea.LinkedTaskID = t.ID
	idea.Status = domain.IdeaStatusBuilding
	idea.UpdatedAt = now
	if err := s.Ideas.Save(ctx, idea); err != nil {
		return nil, err
	}
	return t, nil
}

// splitSpecTitleBody uses the first non-empty line as title; remainder is description.
func splitSpecTitleBody(spec string) (title, description string) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return "", ""
	}
	first, rest, ok := strings.Cut(spec, "\n")
	first = strings.TrimSpace(first)
	rest = strings.TrimSpace(rest)
	if !ok || rest == "" {
		if first != "" {
			return first, ""
		}
		one := strings.ReplaceAll(strings.TrimSpace(spec), "\n", " ")
		if len(one) > 160 {
			one = one[:160] + "…"
		}
		return one, spec
	}
	if first == "" {
		one := strings.ReplaceAll(strings.TrimSpace(spec), "\n", " ")
		if len(one) > 160 {
			one = one[:160] + "…"
		}
		return one, spec
	}
	return first, rest
}

// CreateFromSpecWithNewIdea inserts an auto-approved manual idea (title/description from spec), then creates the planning task linked to it.
// preferredIdeaID, when non-empty, is used as the new row's id (must not already exist). Typical source: POST …/nlp/suggest-idea-id.
// ideaCategory is normalized via domain.NormalizeIdeaCategory (empty → feature).
func (s *Service) CreateFromSpecWithNewIdea(ctx context.Context, productID domain.ProductID, spec string, preferredIdeaID string, ideaCategory string) (*domain.Task, error) {
	spec = strings.TrimSpace(spec)
	ideaCategory = strings.TrimSpace(ideaCategory)
	if spec == "" {
		return nil, fmt.Errorf("%w: spec is required", domain.ErrInvalidInput)
	}
	preferredIdeaID = strings.TrimSpace(preferredIdeaID)
	if len(preferredIdeaID) > 200 {
		return nil, fmt.Errorf("%w: new_idea_id too long", domain.ErrInvalidInput)
	}
	if err := ports.RequireActiveProduct(ctx, s.Products, productID); err != nil {
		return nil, err
	}
	title, desc := splitSpecTitleBody(spec)
	now := s.Clock.Now()
	var ideaID domain.IdeaID
	if preferredIdeaID != "" {
		if _, err := s.Ideas.ByID(ctx, domain.IdeaID(preferredIdeaID)); err == nil {
			return nil, fmt.Errorf("%w: idea id %q already exists", domain.ErrConflict, preferredIdeaID)
		} else if !errors.Is(err, domain.ErrNotFound) {
			return nil, err
		}
		ideaID = domain.IdeaID(preferredIdeaID)
	} else {
		ideaID = s.IDs.NewIdeaID()
	}
	idea := domain.Idea{
		ID:          ideaID,
		ProductID:   productID,
		Title:       title,
		Description: desc,
		Reasoning:   "",
		Decided:     true,
		Decision:    domain.DecisionYes,
		CreatedAt:   now,
		UpdatedAt:   now,
		Source:      "manual",
		Category:    ideaCategory,
	}
	domain.SyncIdeaStatusFromSwipe(&idea, domain.DecisionYes, now)
	domain.NormalizeIdeaForSave(&idea, now)
	if err := s.Ideas.Save(ctx, &idea); err != nil {
		return nil, err
	}
	return s.CreateFromApprovedIdea(ctx, idea.ID, spec)
}

// ListByProduct returns tasks for a product (newest first), or ErrNotFound if the product does not exist.
func (s *Service) ListByProduct(ctx context.Context, productID domain.ProductID) ([]domain.Task, error) {
	if _, err := s.Products.ByID(ctx, productID); err != nil {
		return nil, err
	}
	return s.Tasks.ListByProduct(ctx, productID)
}

// ApprovePlan clears the planning gate and moves the task to inbox (MC-style).
func (s *Service) ApprovePlan(ctx context.Context, taskID domain.TaskID, spec string) error {
	t, err := s.taskWithActiveProduct(ctx, taskID)
	if err != nil {
		return err
	}
	if t.Status != domain.StatusPlanning {
		return fmt.Errorf("%w: task not in planning", domain.ErrInvalidTransition)
	}
	t.PlanApproved = true
	if strings.TrimSpace(spec) != "" {
		t.Spec = spec
	}
	t.Status = domain.StatusInbox
	t.StatusReason = ""
	t.UpdatedAt = s.Clock.Now()
	return s.Tasks.Save(ctx, t)
}

// ReturnToPlanning revokes plan approval and moves the task back to planning from inbox or assigned (before dispatch).
func (s *Service) ReturnToPlanning(ctx context.Context, taskID domain.TaskID, statusReason string) error {
	t, err := s.taskWithActiveProduct(ctx, taskID)
	if err != nil {
		return err
	}
	reason := strings.TrimSpace(statusReason)
	switch t.Status {
	case domain.StatusPlanning:
		return fmt.Errorf("%w: already in planning", domain.ErrInvalidTransition)
	case domain.StatusInbox:
		if !domain.AllowedKanbanTransition(t.Status, domain.StatusPlanning) {
			return fmt.Errorf("%w: inbox -> planning", domain.ErrInvalidTransition)
		}
		t.Status = domain.StatusPlanning
		t.PlanApproved = false
		t.StatusReason = reason
	case domain.StatusAssigned:
		if t.ExternalRef != "" {
			return fmt.Errorf("%w: cannot recall after dispatch", domain.ErrInvalidTransition)
		}
		if !domain.AllowedKanbanTransition(domain.StatusAssigned, domain.StatusInbox) ||
			!domain.AllowedKanbanTransition(domain.StatusInbox, domain.StatusPlanning) {
			return fmt.Errorf("%w: assigned -> planning", domain.ErrInvalidTransition)
		}
		t.Status = domain.StatusPlanning
		t.PlanApproved = false
		t.StatusReason = reason
	default:
		return fmt.Errorf("%w: return to planning from %s", domain.ErrInvalidTransition, t.Status)
	}
	t.UpdatedAt = s.Clock.Now()
	return s.Tasks.Save(ctx, t)
}

// SetKanbanStatus moves the task on the board when AllowedKanbanTransition permits it.
func (s *Service) SetKanbanStatus(ctx context.Context, taskID domain.TaskID, to domain.TaskStatus, statusReason string) error {
	t, err := s.taskWithActiveProduct(ctx, taskID)
	if err != nil {
		return err
	}
	if !domain.AllowedKanbanTransition(t.Status, to) {
		return fmt.Errorf("%w: %s -> %s", domain.ErrInvalidTransition, t.Status, to)
	}
	if to == domain.StatusAssigned && !t.PlanApproved {
		return fmt.Errorf("%w: assign requires approved plan", domain.ErrInvalidTransition)
	}
	from := t.Status
	t.Status = to
	t.StatusReason = strings.TrimSpace(statusReason)
	t.UpdatedAt = s.Clock.Now()
	if err := s.Tasks.Save(ctx, t); err != nil {
		return err
	}
	_ = s.tryAutoOpenPRIfApplicable(ctx, taskID, from, to)
	if to == domain.StatusDone {
		s.maybeAutoMergeShip(ctx, taskID)
	}
	return nil
}

// maybeAutoMergeShip runs merge-queue ship when a task reaches done (best-effort): full_auto always attempts ship; semi_auto only if merge gates pass.
// For full_auto and semi_auto, opens a PR first when pull_request_head_branch is set but no PR URL yet (e.g. agent-completion jumps straight to done without a review transition).
func (s *Service) maybeAutoMergeShip(ctx context.Context, taskID domain.TaskID) {
	if s.MergeShip == nil {
		return
	}
	t, err := s.taskWithActiveProduct(ctx, taskID)
	if err != nil || t.Status != domain.StatusDone {
		return
	}
	p, err := s.Products.ByID(ctx, t.ProductID)
	if err != nil {
		return
	}
	switch p.AutomationTier {
	case domain.TierFullAuto:
		_ = s.ensurePullRequestForAutoMerge(ctx, taskID, p.AutomationTier)
		_ = s.MergeShip.Complete(ctx, taskID, false)
	case domain.TierSemiAuto:
		_ = s.ensurePullRequestForAutoMerge(ctx, taskID, p.AutomationTier)
		_ = s.MergeShip.CompleteIfPolicyAllowsAuto(ctx, taskID)
	default:
		// supervised: merge queue completion stays manual
	}
}

func (s *Service) ensurePullRequestForAutoMerge(ctx context.Context, taskID domain.TaskID, tier domain.AutomationTier) error {
	return s.openPullRequestWhenEligible(ctx, taskID, tier, mergePrepAutoPRTiers)
}

// tryAutoOpenPRIfApplicable opens a PR when entering review from execution columns under full_auto (best-effort; errors ignored).
func (s *Service) tryAutoOpenPRIfApplicable(ctx context.Context, taskID domain.TaskID, from, to domain.TaskStatus) error {
	if to != domain.StatusReview {
		return nil
	}
	switch from {
	case domain.StatusTesting, domain.StatusInProgress, domain.StatusConvoyActive:
	default:
		return nil
	}
	p, err := s.productForTask(ctx, taskID)
	if err != nil {
		return nil
	}
	_ = s.openPullRequestWhenEligible(ctx, taskID, p.AutomationTier, reviewColumnAutoPRTiers)
	return nil
}

func (s *Service) productForTask(ctx context.Context, taskID domain.TaskID) (*domain.Product, error) {
	_, p, err := s.taskAndActiveProduct(ctx, taskID)
	return p, err
}

// UpdatePlanningArtifacts stores opaque planning JSON (e.g. clarifying Q&A) while in planning.
func (s *Service) UpdatePlanningArtifacts(ctx context.Context, taskID domain.TaskID, clarificationsJSON string) error {
	t, err := s.taskWithActiveProduct(ctx, taskID)
	if err != nil {
		return err
	}
	if t.Status != domain.StatusPlanning {
		return fmt.Errorf("%w: not in planning", domain.ErrInvalidTransition)
	}
	t.ClarificationsJSON = clarificationsJSON
	t.UpdatedAt = s.Clock.Now()
	return s.Tasks.Save(ctx, t)
}

// SetStatusReasonOnly updates the free-text reason without moving the Kanban column.
func (s *Service) SetStatusReasonOnly(ctx context.Context, taskID domain.TaskID, statusReason string) error {
	t, err := s.taskWithActiveProduct(ctx, taskID)
	if err != nil {
		return err
	}
	t.StatusReason = strings.TrimSpace(statusReason)
	t.UpdatedAt = s.Clock.Now()
	return s.Tasks.Save(ctx, t)
}

// Dispatch sends work to the execution plane when the task is assigned (MC dispatch gate).
func (s *Service) Dispatch(ctx context.Context, taskID domain.TaskID, estimatedCost float64) error {
	t, err := s.taskWithActiveProduct(ctx, taskID)
	if err != nil {
		return err
	}
	if t.Status != domain.StatusAssigned {
		return fmt.Errorf("%w: dispatch requires status assigned (got %s)", domain.ErrInvalidTransition, t.Status)
	}
	if !t.PlanApproved {
		return fmt.Errorf("%w: plan not approved", domain.ErrInvalidTransition)
	}
	if err := s.Budget.AssertWithinBudget(ctx, t.ProductID, estimatedCost); err != nil {
		return err
	}
	if s.ExecAgents != nil && strings.TrimSpace(t.CurrentExecutionAgentID) == "" {
		if aid, aerr := s.pickLeastLoadedExecutionAgent(ctx, t.ProductID, ""); aerr == nil && aid != "" {
			t.CurrentExecutionAgentID = aid
		}
	}
	ref, err := s.Gateway.DispatchTask(ctx, *t)
	if err != nil {
		if errors.Is(err, domain.ErrNoDispatchTarget) {
			return err
		}
		return fmt.Errorf("%w: %v", domain.ErrGateway, err)
	}
	t.Status = domain.StatusInProgress
	t.ExternalRef = ref
	t.UpdatedAt = s.Clock.Now()
	ev := ports.LiveActivityEvent{
		Type:      "task_dispatched",
		Ts:        s.Clock.Now().UTC().Format(time.RFC3339Nano),
		ProductID: string(t.ProductID),
		TaskID:    string(t.ID),
		Data: map[string]any{
			"external_ref": ref,
		},
	}
	if s.LiveTX != nil {
		if err := s.LiveTX.SaveTaskWithEvent(ctx, t, ev); err != nil {
			return err
		}
		return nil
	}
	if err := s.Tasks.Save(ctx, t); err != nil {
		return err
	}
	if s.Events != nil {
		_ = s.Events.Publish(ctx, ev)
	}
	return nil
}

// RecordCheckpoint persists crash-recovery state from the gateway stream.
func (s *Service) RecordCheckpoint(ctx context.Context, taskID domain.TaskID, payload string) error {
	t, err := s.taskWithActiveProduct(ctx, taskID)
	if err != nil {
		return err
	}
	switch t.Status {
	case domain.StatusInProgress, domain.StatusTesting, domain.StatusConvoyActive:
	default:
		return fmt.Errorf("%w: checkpoint not allowed in %s", domain.ErrInvalidTransition, t.Status)
	}
	t.Checkpoint = payload
	t.Status = domain.StatusInProgress
	t.UpdatedAt = s.Clock.Now()
	ev := ports.LiveActivityEvent{
		Type:      "checkpoint_saved",
		Ts:        s.Clock.Now().UTC().Format(time.RFC3339Nano),
		ProductID: string(t.ProductID),
		TaskID:    string(taskID),
		Data:      map[string]any{"payload_len": len(payload)},
	}
	if s.LiveTX != nil {
		return s.LiveTX.RecordCheckpointWithEvent(ctx, taskID, payload, t, ev)
	}
	if err := s.Checkpt.Save(ctx, taskID, payload); err != nil {
		return err
	}
	if err := s.Tasks.Save(ctx, t); err != nil {
		return err
	}
	if s.Events != nil {
		_ = s.Events.Publish(ctx, ev)
	}
	return nil
}

// ListCheckpointHistory returns recent checkpoint revisions newest-first.
func (s *Service) ListCheckpointHistory(ctx context.Context, taskID domain.TaskID, limit int) ([]domain.CheckpointHistoryEntry, error) {
	if _, err := s.taskWithActiveProduct(ctx, taskID); err != nil {
		return nil, err
	}
	return s.Checkpt.ListHistory(ctx, taskID, limit)
}

// RestoreCheckpoint applies a historical payload through the same gate as RecordCheckpoint.
func (s *Service) RestoreCheckpoint(ctx context.Context, taskID domain.TaskID, historyID int64) error {
	e, err := s.Checkpt.HistoryByID(ctx, historyID)
	if err != nil {
		return err
	}
	if e.TaskID != taskID {
		return domain.ErrNotFound
	}
	return s.RecordCheckpoint(ctx, taskID, e.Payload)
}

// OpenPullRequest opens a GitHub PR (requires product.repo_url; supports github.com and GitHub-like paths on GHES).
// Persists pull_request_url, pull_request_number (when known), and pull_request_head_branch on the task.
// When LiveTX is wired (SQLite), task row and pull_request_opened outbox row commit together; CreatePullRequest is retried a few times on transient shipping errors.
func (s *Service) OpenPullRequest(ctx context.Context, taskID domain.TaskID, headBranch, title, body string) (prURL string, prNumber int, err error) {
	if s.Ship == nil {
		return "", 0, fmt.Errorf("%w: pull request publisher not configured", domain.ErrInvalidInput)
	}
	t, err := s.taskWithActiveProduct(ctx, taskID)
	if err != nil {
		return "", 0, err
	}
	switch t.Status {
	case domain.StatusInProgress, domain.StatusTesting, domain.StatusReview, domain.StatusDone:
	default:
		return "", 0, fmt.Errorf("%w: pull request not allowed in %s", domain.ErrInvalidTransition, t.Status)
	}
	p, err := s.Products.ByID(ctx, t.ProductID)
	if err != nil {
		return "", 0, err
	}
	owner, repo, err := domain.ParseGitHubLikeOwnerRepo(p.RepoURL)
	if err != nil {
		return "", 0, fmt.Errorf("%w: product.repo_url: %v", domain.ErrInvalidInput, err)
	}
	base := strings.TrimSpace(p.RepoBranch)
	if base == "" {
		base = "main"
	}
	head := strings.TrimSpace(headBranch)
	if head == "" {
		return "", 0, fmt.Errorf("%w: head_branch required", domain.ErrInvalidInput)
	}
	ti := strings.TrimSpace(title)
	if ti == "" {
		ti = fmt.Sprintf("[%s] %s", taskID, trimSpecOneLine(t.Spec))
	}
	cre, err := createPullRequestWithRetry(ctx, s.Ship, ports.CreatePullRequestInput{
		ProductID:  t.ProductID,
		TaskID:     taskID,
		Owner:      owner,
		Repo:       repo,
		Title:      ti,
		Body:       body,
		HeadBranch: head,
		BaseBranch: base,
	})
	if err != nil {
		return "", 0, err
	}
	prURL = cre.HTMLURL
	prNumber = cre.Number
	t.PullRequestURL = prURL
	t.PullRequestNumber = prNumber
	t.PullRequestHeadBranch = head
	t.UpdatedAt = s.Clock.Now()
	if err := s.persistPullRequestOpen(ctx, t, taskID, prURL, prNumber, head); err != nil {
		return prURL, prNumber, err
	}
	return prURL, prNumber, nil
}

func (s *Service) persistPullRequestOpen(ctx context.Context, t *domain.Task, taskID domain.TaskID, prURL string, prNumber int, head string) error {
	if prURL == "" {
		return s.Tasks.Save(ctx, t)
	}
	prData := map[string]any{"html_url": prURL, "head": head}
	if prNumber > 0 {
		prData["number"] = prNumber
	}
	ev := ports.LiveActivityEvent{
		Type:      "pull_request_opened",
		Ts:        s.Clock.Now().UTC().Format(time.RFC3339Nano),
		ProductID: string(t.ProductID),
		TaskID:    string(taskID),
		Data:      prData,
	}
	if s.LiveTX != nil {
		return s.LiveTX.SaveTaskWithEvent(ctx, t, ev)
	}
	if err := s.Tasks.Save(ctx, t); err != nil {
		return err
	}
	if s.Events != nil {
		_ = s.Events.Publish(ctx, ev)
	}
	return nil
}

func trimSpecOneLine(spec string) string {
	s := strings.TrimSpace(spec)
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > 80 {
		return s[:77] + "..."
	}
	if s == "" {
		return "update"
	}
	return s
}

// Complete marks the task finished (e.g. after agent-completion webhook).
func (s *Service) Complete(ctx context.Context, taskID domain.TaskID) error {
	t, err := s.taskWithActiveProduct(ctx, taskID)
	if err != nil {
		return err
	}
	run := func() error {
		return s.Tasks.TryComplete(ctx, taskID, s.Clock.Now())
	}
	var runErr error
	if s.Gate != nil {
		runErr = s.Gate.WithLock(t.ProductID, run)
	} else {
		runErr = run()
	}
	if runErr == nil {
		s.maybeAutoMergeShip(ctx, taskID)
	}
	return runErr
}

// UsesLiveActivityTX is true when task completion can persist agent health + outbox in the same SQLite transaction.
func (s *Service) UsesLiveActivityTX() bool { return s.LiveTX != nil }

// CompleteWithLiveActivity completes the task, records agent health as completed when SQLite LiveTX is wired,
// and emits task_completed (same DB transaction as domain writes, or hub/outbox publish on memory path).
// source is stored in agent-health detail JSON (e.g. api_task_complete, agent_completion_webhook).
// Optional knowledgeSummary (first element only) is passed to KnowledgeAutoIngest when wired.
func (s *Service) CompleteWithLiveActivity(ctx context.Context, taskID domain.TaskID, source string, knowledgeSummary ...string) error {
	t, err := s.taskWithActiveProduct(ctx, taskID)
	if err != nil {
		return err
	}
	productID := t.ProductID
	detailB, err := json.Marshal(map[string]string{"source": source})
	if err != nil {
		return err
	}
	now := s.Clock.Now()
	ev := ports.LiveActivityEvent{
		Type:      "task_completed",
		Ts:        now.UTC().Format(time.RFC3339Nano),
		ProductID: string(productID),
		TaskID:    string(taskID),
		Data:      map[string]any{"source": source},
	}
	run := func() error {
		if s.LiveTX != nil {
			return s.LiveTX.CompleteTaskWithEvent(ctx, taskID, now, "completed", string(detailB), ev)
		}
		if err := s.Tasks.TryComplete(ctx, taskID, now); err != nil {
			return err
		}
		if s.Events != nil {
			_ = s.Events.Publish(ctx, ev)
		}
		return nil
	}
	var runErr error
	if s.Gate != nil {
		runErr = s.Gate.WithLock(productID, run)
	} else {
		runErr = run()
	}
	if runErr == nil {
		s.maybeAutoMergeShip(ctx, taskID)
		s.runKnowledgeAutoIngest(ctx, taskID, source, knowledgeSummary)
	}
	return runErr
}

func (s *Service) runKnowledgeAutoIngest(ctx context.Context, taskID domain.TaskID, source string, knowledgeSummary []string) {
	if s.KnowledgeAutoIngest == nil {
		return
	}
	sum := ""
	if len(knowledgeSummary) > 0 {
		sum = strings.TrimSpace(knowledgeSummary[0])
	}
	t2, err := s.taskWithActiveProduct(ctx, taskID)
	if err != nil || t2 == nil {
		return
	}
	s.KnowledgeAutoIngest(ctx, t2, source, sum)
}

// NudgeStall records an operator nudge for tasks in active execution statuses (Phase A manual policy).
// Updates task_agent_health detail (when wired), prepends a short line to status_reason, and emits task_stall_nudged.
func (s *Service) NudgeStall(ctx context.Context, taskID domain.TaskID, note string) error {
	return s.nudgeStall(ctx, taskID, note, false)
}

// nudgeStall implements stall nudge. When preserveHeartbeatAt is true (auto-nudge path), last_heartbeat_at is not
// advanced to now so the task stays eligible for stalled detection and cooldown logic until a real agent heartbeat.
func (s *Service) nudgeStall(ctx context.Context, taskID domain.TaskID, note string, preserveHeartbeatAt bool) error {
	t, err := s.taskWithActiveProduct(ctx, taskID)
	if err != nil {
		return err
	}
	switch t.Status {
	case domain.StatusInProgress, domain.StatusTesting, domain.StatusReview, domain.StatusConvoyActive:
	default:
		return fmt.Errorf("%w: stall nudge only for in_progress, testing, review, convoy_active (got %s)", domain.ErrInvalidTransition, t.Status)
	}
	now := s.Clock.Now()
	note = strings.TrimSpace(note)
	line := fmt.Sprintf("[stall_nudge %s]", now.UTC().Format(time.RFC3339Nano))
	if note != "" {
		line += " " + note
	}
	if len(line) > 500 {
		line = line[:500]
	}
	reason := strings.TrimSpace(t.StatusReason)
	if reason != "" {
		reason = line + "; " + reason
	} else {
		reason = line
	}
	t.StatusReason = reason
	t.UpdatedAt = now
	if err := s.Tasks.Save(ctx, t); err != nil {
		return err
	}
	pid := t.ProductID
	if s.AgentHealth != nil {
		var prev *domain.TaskAgentHealth
		if h, herr := s.AgentHealth.ByTask(ctx, taskID); herr == nil && h != nil {
			prev = h
		}
		detail := mergeStallNudgeDetail("", note, now)
		st := string(t.Status)
		if prev != nil {
			if strings.TrimSpace(prev.DetailJSON) != "" {
				detail = mergeStallNudgeDetail(prev.DetailJSON, note, now)
			}
			if strings.TrimSpace(prev.Status) != "" && prev.Status != "unknown" {
				st = prev.Status
			}
		}
		heartbeatAt := now
		if preserveHeartbeatAt {
			if prev != nil {
				heartbeatAt = prev.LastHeartbeatAt
			} else {
				heartbeatAt = time.Unix(0, 0).UTC()
			}
		}
		_ = s.AgentHealth.UpsertHeartbeat(ctx, taskID, pid, st, detail, heartbeatAt)
	}
	if s.Events != nil {
		evData := map[string]any{"note": note}
		if strings.HasPrefix(strings.TrimSpace(note), "auto:") {
			evData["source"] = "auto"
		}
		_ = s.Events.Publish(ctx, ports.LiveActivityEvent{
			Type:      "task_stall_nudged",
			Ts:        now.UTC().Format(time.RFC3339Nano),
			ProductID: string(pid),
			TaskID:    string(taskID),
			Data:      evData,
		})
	}
	return nil
}

func mergeStallNudgeDetail(existingJSON, note string, at time.Time) string {
	var m map[string]any
	if strings.TrimSpace(existingJSON) != "" && json.Valid([]byte(existingJSON)) {
		_ = json.Unmarshal([]byte(existingJSON), &m)
	}
	if m == nil {
		m = map[string]any{}
	}
	var arr []any
	if raw, ok := m["stall_nudges"].([]any); ok {
		arr = raw
	}
	entry := map[string]any{"at": at.UTC().Format(time.RFC3339Nano)}
	if strings.TrimSpace(note) != "" {
		entry["note"] = note
	}
	arr = append(arr, entry)
	m["stall_nudges"] = arr
	b, err := json.Marshal(m)
	if err != nil {
		return `{"stall_nudges":[]}`
	}
	return string(b)
}

// PatchWorkspacePaths sets optional sandbox / worktree path metadata (operator-managed).
func (s *Service) PatchWorkspacePaths(ctx context.Context, taskID domain.TaskID, sandboxPath, worktreePath *string) error {
	t, err := s.taskWithActiveProduct(ctx, taskID)
	if err != nil {
		return err
	}
	if sandboxPath != nil {
		t.SandboxPath = *sandboxPath
	}
	if worktreePath != nil {
		t.WorktreePath = *worktreePath
	}
	t.UpdatedAt = s.Clock.Now()
	return s.Tasks.Save(ctx, t)
}
