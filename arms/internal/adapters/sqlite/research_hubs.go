package sqlite

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/domain"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/ports"
)

type ResearchHubStore struct{ db *sql.DB }

func NewResearchHubStore(db *sql.DB) *ResearchHubStore { return &ResearchHubStore{db: db} }

var _ ports.ResearchHubRegistry = (*ResearchHubStore)(nil)

func (s *ResearchHubStore) Save(ctx context.Context, h *domain.ResearchHub) error {
	_, err := s.db.ExecContext(ctx, `
INSERT INTO research_hubs (id, display_name, base_url, api_key, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?)`,
		h.ID, h.DisplayName, h.BaseURL, h.APIKey,
		h.CreatedAt.UTC().Format(time.RFC3339Nano), h.UpdatedAt.UTC().Format(time.RFC3339Nano))
	return err
}

func (s *ResearchHubStore) ByID(ctx context.Context, id string) (*domain.ResearchHub, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, display_name, base_url, api_key, created_at, updated_at FROM research_hubs WHERE id = ?`, id)
	var h domain.ResearchHub
	var c, u string
	if err := row.Scan(&h.ID, &h.DisplayName, &h.BaseURL, &h.APIKey, &c, &u); err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	h.CreatedAt = parseSQLiteTime(c)
	h.UpdatedAt = parseSQLiteTime(u)
	return &h, nil
}

func (s *ResearchHubStore) List(ctx context.Context, limit int) ([]domain.ResearchHub, error) {
	if limit <= 0 {
		limit = 200
	}
	if limit > 500 {
		limit = 500
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT id, display_name, base_url, api_key, created_at, updated_at FROM research_hubs ORDER BY created_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanResearchHubRows(rows)
}

func scanResearchHubRows(rows *sql.Rows) ([]domain.ResearchHub, error) {
	var out []domain.ResearchHub
	for rows.Next() {
		var h domain.ResearchHub
		var c, u string
		if err := rows.Scan(&h.ID, &h.DisplayName, &h.BaseURL, &h.APIKey, &c, &u); err != nil {
			return nil, err
		}
		h.CreatedAt = parseSQLiteTime(c)
		h.UpdatedAt = parseSQLiteTime(u)
		out = append(out, h)
	}
	return out, rows.Err()
}

func (s *ResearchHubStore) Update(ctx context.Context, h *domain.ResearchHub) error {
	res, err := s.db.ExecContext(ctx, `
UPDATE research_hubs SET display_name=?, base_url=?, api_key=?, updated_at=? WHERE id=?`,
		h.DisplayName, h.BaseURL, h.APIKey, h.UpdatedAt.UTC().Format(time.RFC3339Nano), h.ID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *ResearchHubStore) Delete(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM research_hubs WHERE id=?`, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

type ResearchSystemSettingsStore struct{ db *sql.DB }

func NewResearchSystemSettingsStore(db *sql.DB) *ResearchSystemSettingsStore {
	return &ResearchSystemSettingsStore{db: db}
}

var _ ports.ResearchSystemSettingsRepository = (*ResearchSystemSettingsStore)(nil)

func (s *ResearchSystemSettingsStore) Get(ctx context.Context) (domain.ResearchSystemSettings, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT auto_research_claw_enabled, default_research_hub_id FROM research_system_settings WHERE singleton = 1`)
	var enabled int
	var hid sql.NullString
	if err := row.Scan(&enabled, &hid); err != nil {
		return domain.ResearchSystemSettings{}, err
	}
	out := domain.ResearchSystemSettings{AutoResearchClawEnabled: enabled != 0}
	if hid.Valid {
		out.DefaultResearchHubID = strings.TrimSpace(hid.String)
	}
	return out, nil
}

func (s *ResearchSystemSettingsStore) Upsert(ctx context.Context, st domain.ResearchSystemSettings) error {
	var hid sql.NullString
	if v := strings.TrimSpace(st.DefaultResearchHubID); v != "" {
		hid.String = v
		hid.Valid = true
	}
	en := 0
	if st.AutoResearchClawEnabled {
		en = 1
	}
	res, err := s.db.ExecContext(ctx, `
UPDATE research_system_settings SET auto_research_claw_enabled = ?, default_research_hub_id = ? WHERE singleton = 1`,
		en, hid)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		_, err = s.db.ExecContext(ctx, `
INSERT INTO research_system_settings (singleton, auto_research_claw_enabled, default_research_hub_id) VALUES (1, ?, ?)`,
			en, hid)
	}
	return err
}

func parseSQLiteTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		t, _ = time.Parse(time.RFC3339, s)
	}
	return t.UTC()
}
