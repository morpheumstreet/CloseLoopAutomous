package knowledge

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/domain"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/ports"
)

// Service is product-scoped knowledge CRUD + dispatch-time markdown (#90).
type Service struct {
	Products ports.ProductRepository
	Repo     ports.KnowledgeRepository
	Clock    ports.Clock
	// DispatchSnippetLimit caps snippets appended per dispatch (default 5 in wiring).
	DispatchSnippetLimit int
	// UseFTSQuerySyntax when true passes SanitizeFTS5Query(q) to Search/List paths that need FTS (SQLite). When false, raw trimmed text is used (chromem, memory).
	UseFTSQuerySyntax bool
	// AutoIngest when true writes knowledge rows from swipes, product feedback, and task completion (default on; disable with ARMS_KNOWLEDGE_AUTO_INGEST=0).
	AutoIngest bool
}

// Get returns one entry scoped to product.
func (s *Service) Get(ctx context.Context, id int64, productID domain.ProductID) (*domain.KnowledgeEntry, error) {
	if s == nil || s.Repo == nil {
		return nil, domain.ErrNotConfigured
	}
	if _, err := s.Products.ByID(ctx, productID); err != nil {
		return nil, err
	}
	return s.Repo.ByID(ctx, id, productID)
}

// Create adds a knowledge entry after validating product exists.
func (s *Service) Create(ctx context.Context, productID domain.ProductID, content string, taskID domain.TaskID, metadata map[string]any) (*domain.KnowledgeEntry, error) {
	if s == nil || s.Repo == nil {
		return nil, domain.ErrNotConfigured
	}
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, fmt.Errorf("%w: content required", domain.ErrInvalidInput)
	}
	if _, err := s.Products.ByID(ctx, productID); err != nil {
		return nil, err
	}
	meta := "{}"
	if metadata != nil {
		b, err := json.Marshal(metadata)
		if err != nil {
			return nil, err
		}
		meta = string(b)
	}
	now := s.Clock.Now().UTC()
	e := &domain.KnowledgeEntry{
		ProductID:    productID,
		TaskID:       domain.TaskID(strings.TrimSpace(string(taskID))),
		Content:      content,
		MetadataJSON: meta,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := s.Repo.Create(ctx, e); err != nil {
		return nil, err
	}
	return e, nil
}

// Update changes content and/or metadata.
func (s *Service) Update(ctx context.Context, id int64, productID domain.ProductID, content *string, metadata map[string]any) (*domain.KnowledgeEntry, error) {
	if s == nil || s.Repo == nil {
		return nil, domain.ErrNotConfigured
	}
	prev, err := s.Repo.ByID(ctx, id, productID)
	if err != nil {
		return nil, err
	}
	newContent := prev.Content
	if content != nil {
		c := strings.TrimSpace(*content)
		if c == "" {
			return nil, fmt.Errorf("%w: content required", domain.ErrInvalidInput)
		}
		newContent = c
	}
	meta := prev.MetadataJSON
	if metadata != nil {
		b, err := json.Marshal(metadata)
		if err != nil {
			return nil, err
		}
		meta = string(b)
	}
	now := s.Clock.Now().UTC()
	if err := s.Repo.Update(ctx, id, productID, newContent, meta, now); err != nil {
		return nil, err
	}
	return s.Repo.ByID(ctx, id, productID)
}

// Delete removes an entry scoped to product.
func (s *Service) Delete(ctx context.Context, id int64, productID domain.ProductID) error {
	if s == nil || s.Repo == nil {
		return domain.ErrNotConfigured
	}
	return s.Repo.Delete(ctx, id, productID)
}

// List returns recent entries for a product.
func (s *Service) List(ctx context.Context, productID domain.ProductID, limit int) ([]domain.KnowledgeEntry, error) {
	if s == nil || s.Repo == nil {
		return nil, domain.ErrNotConfigured
	}
	if _, err := s.Products.ByID(ctx, productID); err != nil {
		return nil, err
	}
	return s.Repo.ListByProduct(ctx, productID, limit)
}

// Search runs FTS (SQLite) or substring scoring (memory).
func (s *Service) Search(ctx context.Context, productID domain.ProductID, q string, limit int) ([]domain.KnowledgeEntry, error) {
	if s == nil || s.Repo == nil {
		return nil, domain.ErrNotConfigured
	}
	if _, err := s.Products.ByID(ctx, productID); err != nil {
		return nil, err
	}
	query := strings.TrimSpace(q)
	if s.UseFTSQuerySyntax {
		query = SanitizeFTS5Query(q)
	}
	return s.Repo.Search(ctx, productID, query, limit)
}

// MarkdownBlockForDispatch returns markdown to append to OpenClaw dispatch bodies.
func (s *Service) MarkdownBlockForDispatch(ctx context.Context, productID domain.ProductID, rawQuery string) (string, error) {
	if s == nil || s.Repo == nil {
		return "", nil
	}
	limit := s.DispatchSnippetLimit
	if limit <= 0 {
		limit = 5
	}
	var query string
	if s.UseFTSQuerySyntax {
		query = SanitizeFTS5Query(rawQuery)
	} else {
		query = strings.TrimSpace(rawQuery)
	}
	var rows []domain.KnowledgeEntry
	var err error
	if query == "" {
		rows, err = s.Repo.ListByProduct(ctx, productID, limit)
	} else {
		rows, err = s.Repo.Search(ctx, productID, query, limit)
	}
	if err != nil || len(rows) == 0 {
		return "", err
	}
	var b strings.Builder
	for i := range rows {
		if i > 0 {
			b.WriteByte('\n')
		}
		line := strings.TrimSpace(rows[i].Content)
		line = strings.ReplaceAll(line, "\n", "\n  ")
		fmt.Fprintf(&b, "- %s", line)
	}
	return b.String(), nil
}

// DispatchHook returns a callback for [openclaw.Options.KnowledgeForDispatch] (nil if service/repo missing).
func (s *Service) DispatchHook() func(ctx context.Context, productID domain.ProductID, query string) (string, error) {
	if s == nil || s.Repo == nil {
		return nil
	}
	return func(ctx context.Context, productID domain.ProductID, query string) (string, error) {
		return s.MarkdownBlockForDispatch(ctx, productID, query)
	}
}
