package chromemstore

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/philippgille/chromem-go"

	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/config"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/domain"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/ports"
)

const collectionName = "arms_knowledge"

// KnowledgeStore implements [ports.KnowledgeRepository] with chromem-go vector search.
type KnowledgeStore struct {
	coll    *chromem.Collection
	seqMu   sync.Mutex
	seqPath string
}

// NewKnowledgeStore opens a persistent chromem DB and collection.
func NewKnowledgeStore(cfg config.Config) (*KnowledgeStore, error) {
	dir := strings.TrimSpace(cfg.ChromemPersistencePath)
	if dir == "" {
		dir = "./data/chromem-knowledge"
	}
	dir = filepath.Clean(dir)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("chromem knowledge: mkdir %q: %w", dir, err)
	}
	embed, err := NewEmbeddingFunc(cfg)
	if err != nil {
		return nil, err
	}
	db, err := chromem.NewPersistentDB(dir, cfg.ChromemCompress)
	if err != nil {
		return nil, fmt.Errorf("chromem knowledge: %w", err)
	}
	coll, err := db.GetOrCreateCollection(collectionName, nil, embed)
	if err != nil {
		return nil, fmt.Errorf("chromem knowledge collection: %w", err)
	}
	_ = db // no explicit DB close; collection holds persisted state under dir
	s := &KnowledgeStore{coll: coll, seqPath: filepath.Join(dir, "arms_knowledge.seq")}
	if err := s.bootstrapSeq(context.Background()); err != nil {
		return nil, err
	}
	return s, nil
}

var _ ports.KnowledgeRepository = (*KnowledgeStore)(nil)

func readSeqFile(path string) int64 {
	b, err := os.ReadFile(path)
	if err != nil || len(strings.TrimSpace(string(b))) == 0 {
		return 0
	}
	n, _ := strconv.ParseInt(strings.TrimSpace(string(b)), 10, 64)
	if n < 0 {
		return 0
	}
	return n
}

func writeSeqFile(path string, n int64) error {
	return os.WriteFile(path, []byte(strconv.FormatInt(n, 10)+"\n"), 0o600)
}

// bootstrapSeq sets arms_knowledge.seq from existing document IDs when the file is missing or zero.
func (s *KnowledgeStore) bootstrapSeq(ctx context.Context) error {
	s.seqMu.Lock()
	defer s.seqMu.Unlock()
	if readSeqFile(s.seqPath) > 0 {
		return nil
	}
	n := s.coll.Count()
	if n == 0 {
		return nil
	}
	res, err := s.coll.Query(ctx, ".", n, nil, nil)
	if err != nil {
		return fmt.Errorf("chromem knowledge bootstrap seq: %w", err)
	}
	var max int64
	for i := range res {
		id, err := strconv.ParseInt(res[i].ID, 10, 64)
		if err != nil {
			continue
		}
		if id > max {
			max = id
		}
	}
	if max == 0 {
		return nil
	}
	return writeSeqFile(s.seqPath, max)
}

func (s *KnowledgeStore) nextID() (int64, error) {
	s.seqMu.Lock()
	defer s.seqMu.Unlock()
	n := readSeqFile(s.seqPath) + 1
	if err := writeSeqFile(s.seqPath, n); err != nil {
		return 0, err
	}
	return n, nil
}

func docKey(id int64) string {
	return strconv.FormatInt(id, 10)
}

func metaFromEntry(e *domain.KnowledgeEntry) map[string]string {
	m := map[string]string{
		"product_id":    string(e.ProductID),
		"metadata_json": e.MetadataJSON,
		"created_at":    e.CreatedAt.UTC().Format(time.RFC3339Nano),
		"updated_at":    e.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
	if tid := strings.TrimSpace(string(e.TaskID)); tid != "" {
		m["task_id"] = tid
	}
	return m
}

func entryFromDoc(id string, meta map[string]string, content string) (*domain.KnowledgeEntry, error) {
	nid, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("chromem document id: %w", err)
	}
	pid := domain.ProductID(meta["product_id"])
	var tid domain.TaskID
	if v := strings.TrimSpace(meta["task_id"]); v != "" {
		tid = domain.TaskID(v)
	}
	metaj := strings.TrimSpace(meta["metadata_json"])
	if metaj == "" {
		metaj = "{}"
	}
	ct, _ := time.Parse(time.RFC3339Nano, meta["created_at"])
	if ct.IsZero() {
		ct, _ = time.Parse(time.RFC3339, meta["created_at"])
	}
	ut, _ := time.Parse(time.RFC3339Nano, meta["updated_at"])
	if ut.IsZero() {
		ut, _ = time.Parse(time.RFC3339, meta["updated_at"])
	}
	return &domain.KnowledgeEntry{
		ID:           nid,
		ProductID:    pid,
		TaskID:       tid,
		Content:      content,
		MetadataJSON: metaj,
		CreatedAt:    ct.UTC(),
		UpdatedAt:    ut.UTC(),
	}, nil
}

// Create assigns a monotonic ID, embeds content, and persists the document.
func (s *KnowledgeStore) Create(ctx context.Context, e *domain.KnowledgeEntry) error {
	if e == nil {
		return domain.ErrInvalidInput
	}
	if strings.TrimSpace(e.MetadataJSON) == "" {
		e.MetadataJSON = "{}"
	}
	id, err := s.nextID()
	if err != nil {
		return err
	}
	e.ID = id
	return s.coll.AddDocument(ctx, chromem.Document{
		ID:       docKey(id),
		Metadata: metaFromEntry(e),
		Content:  e.Content,
	})
}

func (s *KnowledgeStore) Update(ctx context.Context, id int64, productID domain.ProductID, content string, metadataJSON string, at time.Time) error {
	prev, err := s.ByID(ctx, id, productID)
	if err != nil {
		return err
	}
	meta := strings.TrimSpace(metadataJSON)
	if meta == "" {
		meta = "{}"
	}
	prev.Content = content
	prev.MetadataJSON = meta
	prev.UpdatedAt = at.UTC()
	if err := s.coll.Delete(ctx, nil, nil, docKey(id)); err != nil {
		return err
	}
	return s.coll.AddDocument(ctx, chromem.Document{
		ID:       docKey(id),
		Metadata: metaFromEntry(prev),
		Content:  prev.Content,
	})
}

func (s *KnowledgeStore) Delete(ctx context.Context, id int64, productID domain.ProductID) error {
	if _, err := s.ByID(ctx, id, productID); err != nil {
		return err
	}
	return s.coll.Delete(ctx, nil, nil, docKey(id))
}

func (s *KnowledgeStore) ByID(ctx context.Context, id int64, productID domain.ProductID) (*domain.KnowledgeEntry, error) {
	d, err := s.coll.GetByID(ctx, docKey(id))
	if err != nil {
		return nil, domain.ErrNotFound
	}
	e, err := entryFromDoc(d.ID, d.Metadata, d.Content)
	if err != nil {
		return nil, err
	}
	if e.ProductID != productID {
		return nil, domain.ErrNotFound
	}
	return e, nil
}

func (s *KnowledgeStore) ListByProduct(ctx context.Context, productID domain.ProductID, limit int) ([]domain.KnowledgeEntry, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	return s.queryProductOrdered(ctx, productID, limit)
}

// Search runs semantic similarity; query must be plain text (not FTS5 syntax).
func (s *KnowledgeStore) Search(ctx context.Context, productID domain.ProductID, query string, limit int) ([]domain.KnowledgeEntry, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	q := strings.TrimSpace(query)
	if q == "" {
		return s.ListByProduct(ctx, productID, limit)
	}
	total := s.coll.Count()
	if total == 0 {
		return nil, nil
	}
	nResults := limit
	if nResults > total {
		nResults = total
	}
	res, err := s.coll.Query(ctx, q, nResults, map[string]string{"product_id": string(productID)}, nil)
	if err != nil {
		return nil, err
	}
	out := make([]domain.KnowledgeEntry, 0, len(res))
	for i := range res {
		e, err := entryFromDoc(res[i].ID, res[i].Metadata, res[i].Content)
		if err != nil {
			continue
		}
		out = append(out, *e)
	}
	return out, nil
}

func (s *KnowledgeStore) queryProductOrdered(ctx context.Context, productID domain.ProductID, limit int) ([]domain.KnowledgeEntry, error) {
	total := s.coll.Count()
	if total == 0 {
		return nil, nil
	}
	where := map[string]string{"product_id": string(productID)}
	res, err := s.coll.Query(ctx, ".", total, where, nil)
	if err != nil {
		return nil, err
	}
	var out []domain.KnowledgeEntry
	for i := range res {
		e, err := entryFromDoc(res[i].ID, res[i].Metadata, res[i].Content)
		if err != nil {
			continue
		}
		out = append(out, *e)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].UpdatedAt.After(out[j].UpdatedAt)
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}
