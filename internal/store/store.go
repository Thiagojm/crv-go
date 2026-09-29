package store

import (
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

type Store struct {
	DB      *sql.DB
	DataDir string
}

type ActiveCatalog struct {
	RevisionID    int64
	SourceLabel   string
	ActivatedAt   string
	EligibleCount int
	ExcludedCount int
	ReportJSON    string
}

func Open(dataDir string) (*Store, error) {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, err
	}
	dbPath := filepath.Join(dataDir, "crv.sqlite")
	db, err := sql.Open("sqlite", dbPath+"?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	s := &Store{DB: db, DataDir: dataDir}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error {
	if s == nil || s.DB == nil {
		return nil
	}
	return s.DB.Close()
}

func (s *Store) migrate() error {
	if _, err := s.DB.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY,
		applied_at TEXT NOT NULL
	)`); err != nil {
		return err
	}
	entries, err := migrationFS.ReadDir("migrations")
	if err != nil {
		return err
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	for _, name := range names {
		ver, err := strconv.Atoi(strings.SplitN(name, "_", 2)[0])
		if err != nil {
			return fmt.Errorf("migration name %s: %w", name, err)
		}
		var exists int
		if err := s.DB.QueryRow(`SELECT COUNT(1) FROM schema_migrations WHERE version = ?`, ver).Scan(&exists); err != nil {
			return err
		}
		if exists > 0 {
			continue
		}
		body, err := migrationFS.ReadFile("migrations/" + name)
		if err != nil {
			return err
		}
		tx, err := s.DB.Begin()
		if err != nil {
			return err
		}
		if _, err := tx.Exec(string(body)); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("apply %s: %w", name, err)
		}
		if _, err := tx.Exec(`INSERT INTO schema_migrations(version, applied_at) VALUES (?, ?)`, ver, time.Now().UTC().Format(time.RFC3339)); err != nil {
			_ = tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) GetPreference(key string) (string, bool, error) {
	var v string
	err := s.DB.QueryRow(`SELECT value FROM preferences WHERE key = ?`, key).Scan(&v)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	return v, err == nil, err
}

func (s *Store) SetPreference(key, value string) error {
	_, err := s.DB.Exec(`INSERT INTO preferences(key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, value)
	return err
}

func (s *Store) ActiveCatalog() (*ActiveCatalog, error) {
	row := s.DB.QueryRow(`SELECT id, source_label, activated_at, eligible_count, excluded_count, report_json
		FROM catalog_revisions WHERE active = 1`)
	var c ActiveCatalog
	err := row.Scan(&c.RevisionID, &c.SourceLabel, &c.ActivatedAt, &c.EligibleCount, &c.ExcludedCount, &c.ReportJSON)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// AllRevisionIDs returns every catalog revision id, oldest first.
func (s *Store) AllRevisionIDs() ([]int64, error) {
	rows, err := s.DB.Query(`SELECT id FROM catalog_revisions ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (s *Store) RevisionImages(revisionID int64) ([]ImageRow, error) {
	rows, err := s.DB.Query(`SELECT sha256, original_relpath, display_relpath, width, height, bytes, COALESCE(primary_source_id, ''), sources_json
		FROM catalog_images WHERE revision_id = ?`, revisionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ImageRow
	for rows.Next() {
		var img ImageRow
		if err := rows.Scan(&img.SHA256, &img.OriginalRelpath, &img.DisplayRelpath, &img.Width, &img.Height, &img.Bytes, &img.PrimarySourceID, &img.SourcesJSON); err != nil {
			return nil, err
		}
		out = append(out, img)
	}
	return out, rows.Err()
}

func (s *Store) DeactivateRevision(revisionID int64) error {
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.Exec(`UPDATE catalog_revisions SET active = 0 WHERE id = ?`, revisionID); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) DeleteRevision(revisionID int64) error {
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.Exec(`DELETE FROM catalog_exclusions WHERE revision_id = ?`, revisionID); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM catalog_images WHERE revision_id = ?`, revisionID); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM catalog_revisions WHERE id = ?`, revisionID); err != nil {
		return err
	}
	return tx.Commit()
}

func RevisionDir(dataDir string, revisionID int64) string {
	return filepath.Join(dataDir, "catalog", fmt.Sprintf("rev-%d", revisionID))
}

func (s *Store) UpdateRevisionReport(revisionID int64, report any) error {
	raw, err := json.Marshal(report)
	if err != nil {
		return err
	}
	_, err = s.DB.Exec(`UPDATE catalog_revisions SET report_json = ? WHERE id = ?`, string(raw), revisionID)
	return err
}

type ImageRow struct {
	SHA256          string
	OriginalRelpath string
	DisplayRelpath  string
	Width           int
	Height          int
	Bytes           int64
	PrimarySourceID string
	SourcesJSON     string
}

type ExclusionRow struct {
	SHA256 string
	Path   string
	Reason string
}

// InsertRevisionInactive stores catalog rows with active=0 so files can be finalized first.
func (s *Store) InsertRevisionInactive(sourceLabel string, eligible []ImageRow, exclusions []ExclusionRow, report any) (int64, error) {
	reportJSON, err := json.Marshal(report)
	if err != nil {
		return 0, err
	}
	tx, err := s.DB.Begin()
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()

	res, err := tx.Exec(`INSERT INTO catalog_revisions(source_label, activated_at, eligible_count, excluded_count, report_json, active)
		VALUES (?, ?, ?, ?, ?, 0)`, sourceLabel, time.Now().UTC().Format(time.RFC3339), len(eligible), len(exclusions), string(reportJSON))
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	for _, img := range eligible {
		if _, err := tx.Exec(`INSERT INTO catalog_images(revision_id, sha256, original_relpath, display_relpath, width, height, bytes, primary_source_id, sources_json)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			id, img.SHA256, img.OriginalRelpath, img.DisplayRelpath, img.Width, img.Height, img.Bytes, nullIfEmpty(img.PrimarySourceID), img.SourcesJSON); err != nil {
			return 0, err
		}
	}
	for _, ex := range exclusions {
		if _, err := tx.Exec(`INSERT INTO catalog_exclusions(revision_id, sha256, path, reason) VALUES (?, ?, ?, ?)`,
			id, nullIfEmpty(ex.SHA256), nullIfEmpty(ex.Path), ex.Reason); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return id, nil
}

func (s *Store) MarkRevisionActive(revisionID int64, report any) error {
	reportJSON, err := json.Marshal(report)
	if err != nil {
		return err
	}
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.Exec(`UPDATE catalog_revisions SET active = 0 WHERE active = 1`); err != nil {
		return err
	}
	res, err := tx.Exec(`UPDATE catalog_revisions SET active = 1, report_json = ?, activated_at = ? WHERE id = ?`,
		string(reportJSON), time.Now().UTC().Format(time.RFC3339), revisionID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return fmt.Errorf("revisão %d não encontrada para ativação", revisionID)
	}
	return tx.Commit()
}

// ActivateRevision inserts and activates in one step (tests / simple callers).
func (s *Store) ActivateRevision(sourceLabel string, eligible []ImageRow, exclusions []ExclusionRow, report any) (int64, error) {
	id, err := s.InsertRevisionInactive(sourceLabel, eligible, exclusions, report)
	if err != nil {
		return 0, err
	}
	if err := s.MarkRevisionActive(id, report); err != nil {
		_ = s.DeleteRevision(id)
		return 0, err
	}
	return id, nil
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
