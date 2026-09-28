package store

import (
	"database/sql"
	"errors"
	"strings"
	"time"
)

var ErrConflict = errors.New("conflito de revisão")

type SessionRow struct {
	ID                     string
	Code                   string
	CreateOpID             string
	State                  string
	Revision               int
	CatalogRevisionID      int64
	TargetSHA              string
	PosA, PosB, PosC, PosD string
	ProtocolJSON           string
	HelpVersion            string
	RecordJSON             string
	LockedRecordJSON       string
	CollectionMs           int
	ChoiceMs               int
	TimingSeq              int
	Paused                 bool
	Step                   int
	TentativeChoice        string
	TentativeConfidence    *int
	ConfirmedChoice        string
	ConfirmedConfidence    *int
	Hit                    *int
	Comment                string
	CommentUpdatedAt       string
	CreatedAt              string
	UpdatedAt              string
	LockedAt               string
	CompletedAt            string
	AbandonedAt            string
	AbandonedAfterAlts     bool
	AltsAuthorizedAt       string
	LeaseToken             string
	LeaseUntil             string
	OpenedExamplesJSON     string
}

func (s *Store) PauseNonterminal() error {
	_, err := s.DB.Exec(`UPDATE sessions SET paused = 1, lease_token = NULL, lease_until = NULL, updated_at = ?
		WHERE state IN ('collecting', 'locked')`, time.Now().UTC().Format(time.RFC3339))
	return err
}

func (s *Store) Image(revisionID int64, sha string) (*ImageRow, error) {
	row := s.DB.QueryRow(`SELECT sha256, original_relpath, display_relpath, width, height, bytes, COALESCE(primary_source_id, ''), sources_json
		FROM catalog_images WHERE revision_id = ? AND sha256 = ?`, revisionID, sha)
	var img ImageRow
	err := row.Scan(&img.SHA256, &img.OriginalRelpath, &img.DisplayRelpath, &img.Width, &img.Height, &img.Bytes, &img.PrimarySourceID, &img.SourcesJSON)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &img, nil
}

func (s *Store) GetSession(id string) (*SessionRow, error) {
	return scanSession(s.DB.QueryRow(sessionSelect+` WHERE id = ?`, id))
}

func (s *Store) SessionByOpID(opID string) (*SessionRow, error) {
	return scanSession(s.DB.QueryRow(sessionSelect+` WHERE create_op_id = ?`, opID))
}

func (s *Store) ActiveSession() (*SessionRow, error) {
	return scanSession(s.DB.QueryRow(sessionSelect + ` WHERE state IN ('collecting', 'locked') LIMIT 1`))
}

func (s *Store) InsertSession(row *SessionRow) error {
	_, err := s.DB.Exec(`INSERT INTO sessions (
		id, code, create_op_id, state, revision, catalog_revision_id,
		target_sha256, pos_a, pos_b, pos_c, pos_d,
		protocol_json, help_version, record_json, locked_record_json,
		collection_ms, choice_ms, timing_seq, paused, step,
		tentative_choice, tentative_confidence, confirmed_choice, confirmed_confidence, hit,
		comment, comment_updated_at, created_at, updated_at, locked_at, completed_at, abandoned_at,
		abandoned_after_alts, alts_authorized_at, lease_token, lease_until, opened_examples_json
	) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		row.ID, row.Code, row.CreateOpID, row.State, row.Revision, row.CatalogRevisionID,
		row.TargetSHA, row.PosA, row.PosB, row.PosC, row.PosD,
		row.ProtocolJSON, row.HelpVersion, row.RecordJSON, nullIfEmpty(row.LockedRecordJSON),
		row.CollectionMs, row.ChoiceMs, row.TimingSeq, boolInt(row.Paused), row.Step,
		nullIfEmpty(row.TentativeChoice), nullInt(row.TentativeConfidence),
		nullIfEmpty(row.ConfirmedChoice), nullInt(row.ConfirmedConfidence), nullInt(row.Hit),
		row.Comment, nullIfEmpty(row.CommentUpdatedAt), row.CreatedAt, row.UpdatedAt,
		nullIfEmpty(row.LockedAt), nullIfEmpty(row.CompletedAt), nullIfEmpty(row.AbandonedAt),
		boolInt(row.AbandonedAfterAlts), nullIfEmpty(row.AltsAuthorizedAt),
		nullIfEmpty(row.LeaseToken), nullIfEmpty(row.LeaseUntil), emptyJSON(row.OpenedExamplesJSON),
	)
	return err
}

func (s *Store) SaveSession(row *SessionRow, expectedRevision int) error {
	res, err := s.DB.Exec(`UPDATE sessions SET
		state=?, revision=?, record_json=?, locked_record_json=?,
		collection_ms=?, choice_ms=?, timing_seq=?, paused=?, step=?,
		tentative_choice=?, tentative_confidence=?, confirmed_choice=?, confirmed_confidence=?, hit=?,
		comment=?, comment_updated_at=?, updated_at=?, locked_at=?, completed_at=?, abandoned_at=?,
		abandoned_after_alts=?, alts_authorized_at=?, lease_token=?, lease_until=?, opened_examples_json=?
		WHERE id=? AND revision=?`,
		row.State, row.Revision, row.RecordJSON, nullIfEmpty(row.LockedRecordJSON),
		row.CollectionMs, row.ChoiceMs, row.TimingSeq, boolInt(row.Paused), row.Step,
		nullIfEmpty(row.TentativeChoice), nullInt(row.TentativeConfidence),
		nullIfEmpty(row.ConfirmedChoice), nullInt(row.ConfirmedConfidence), nullInt(row.Hit),
		row.Comment, nullIfEmpty(row.CommentUpdatedAt), row.UpdatedAt,
		nullIfEmpty(row.LockedAt), nullIfEmpty(row.CompletedAt), nullIfEmpty(row.AbandonedAt),
		boolInt(row.AbandonedAfterAlts), nullIfEmpty(row.AltsAuthorizedAt),
		nullIfEmpty(row.LeaseToken), nullIfEmpty(row.LeaseUntil), emptyJSON(row.OpenedExamplesJSON),
		row.ID, expectedRevision,
	)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return ErrConflict
	}
	return nil
}

func (s *Store) PatchSession(row *SessionRow, expectedRevision int) error {
	res, err := s.DB.Exec(`UPDATE sessions SET
		state=?, revision=?, record_json=?, locked_record_json=?,
		collection_ms=?, choice_ms=?, timing_seq=?, paused=?, step=?,
		tentative_choice=?, tentative_confidence=?, confirmed_choice=?, confirmed_confidence=?, hit=?,
		comment=?, comment_updated_at=?, updated_at=?, locked_at=?, completed_at=?, abandoned_at=?,
		abandoned_after_alts=?, alts_authorized_at=?, lease_token=?, lease_until=?, opened_examples_json=?
		WHERE id=? AND revision=?`,
		row.State, row.Revision, row.RecordJSON, nullIfEmpty(row.LockedRecordJSON),
		row.CollectionMs, row.ChoiceMs, row.TimingSeq, boolInt(row.Paused), row.Step,
		nullIfEmpty(row.TentativeChoice), nullInt(row.TentativeConfidence),
		nullIfEmpty(row.ConfirmedChoice), nullInt(row.ConfirmedConfidence), nullInt(row.Hit),
		row.Comment, nullIfEmpty(row.CommentUpdatedAt), row.UpdatedAt,
		nullIfEmpty(row.LockedAt), nullIfEmpty(row.CompletedAt), nullIfEmpty(row.AbandonedAt),
		boolInt(row.AbandonedAfterAlts), nullIfEmpty(row.AltsAuthorizedAt),
		nullIfEmpty(row.LeaseToken), nullIfEmpty(row.LeaseUntil), emptyJSON(row.OpenedExamplesJSON),
		row.ID, expectedRevision,
	)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return ErrConflict
	}
	return nil
}

func (s *Store) InsertEvent(sessionID, at, kind, payload string) error {
	if payload == "" {
		payload = "{}"
	}
	_, err := s.DB.Exec(`INSERT INTO session_events(session_id, at, kind, payload_json) VALUES (?,?,?,?)`,
		sessionID, at, kind, payload)
	return err
}

const sessionSelect = `SELECT id, code, create_op_id, state, revision, catalog_revision_id,
	target_sha256, pos_a, pos_b, pos_c, pos_d, protocol_json, help_version, record_json,
	COALESCE(locked_record_json, ''), collection_ms, choice_ms, timing_seq, paused, step,
	COALESCE(tentative_choice, ''), tentative_confidence, COALESCE(confirmed_choice, ''),
	confirmed_confidence, hit, comment, COALESCE(comment_updated_at, ''),
	created_at, updated_at, COALESCE(locked_at, ''), COALESCE(completed_at, ''), COALESCE(abandoned_at, ''),
	abandoned_after_alts, COALESCE(alts_authorized_at, ''), COALESCE(lease_token, ''), COALESCE(lease_until, ''),
	opened_examples_json FROM sessions`

func scanSession(row *sql.Row) (*SessionRow, error) {
	var r SessionRow
	var tentConf, confConf, hit sql.NullInt64
	var paused, afterAlts int
	err := row.Scan(
		&r.ID, &r.Code, &r.CreateOpID, &r.State, &r.Revision, &r.CatalogRevisionID,
		&r.TargetSHA, &r.PosA, &r.PosB, &r.PosC, &r.PosD, &r.ProtocolJSON, &r.HelpVersion, &r.RecordJSON,
		&r.LockedRecordJSON, &r.CollectionMs, &r.ChoiceMs, &r.TimingSeq, &paused, &r.Step,
		&r.TentativeChoice, &tentConf, &r.ConfirmedChoice, &confConf, &hit, &r.Comment, &r.CommentUpdatedAt,
		&r.CreatedAt, &r.UpdatedAt, &r.LockedAt, &r.CompletedAt, &r.AbandonedAt,
		&afterAlts, &r.AltsAuthorizedAt, &r.LeaseToken, &r.LeaseUntil, &r.OpenedExamplesJSON,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	r.Paused = paused != 0
	r.AbandonedAfterAlts = afterAlts != 0
	r.TentativeConfidence = nullIntPtr(tentConf)
	r.ConfirmedConfidence = nullIntPtr(confConf)
	r.Hit = nullIntPtr(hit)
	return &r, nil
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func nullInt(v *int) any {
	if v == nil {
		return nil
	}
	return *v
}

func nullIntPtr(v sql.NullInt64) *int {
	if !v.Valid {
		return nil
	}
	i := int(v.Int64)
	return &i
}

func emptyJSON(s string) string {
	if s == "" {
		return "[]"
	}
	return s
}

func IsUniqueErr(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "unique")
}

func (s *Store) EligibleSHAs(revisionID int64) ([]string, error) {
	rows, err := s.DB.Query(`SELECT sha256 FROM catalog_images WHERE revision_id = ? ORDER BY sha256`, revisionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var sha string
		if err := rows.Scan(&sha); err != nil {
			return nil, err
		}
		out = append(out, sha)
	}
	return out, rows.Err()
}
