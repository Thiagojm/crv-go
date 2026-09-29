package store

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type HistoryFilter struct {
	States []string // empty = all; allowed: collecting, locked, completed, abandoned
	From   string   // RFC3339 UTC inclusive lower bound on created_at; empty = none
	To     string   // RFC3339 UTC exclusive upper bound on created_at; empty = none
	Limit  int      // default 50, max 200
	Offset int
}

type HistoryListItem struct {
	ID                 string
	Code               string
	State              string
	CreatedAt          string
	CollectionMs       int
	ChoiceMs           int
	AbandonedAfterAlts bool
	ConfirmedChoice    string // empty unless completed
	Hit                *int   // nil unless completed
	CompletedAt        string
	AbandonedAt        string
}

type HistoryPage struct {
	Items []HistoryListItem
	Total int
}

type Stats struct {
	Initiated          int
	Active             int // collecting+locked
	Completed          int
	AbandonedBefore    int // abandoned AND abandoned_after_alts=0
	AbandonedAfter     int // abandoned AND abandoned_after_alts=1
	ConfirmedChoices   int // == Completed
	Hits               int
	HitRate            *float64 // nil when ConfirmedChoices==0; else Hits/ConfirmedChoices as 0..1
	Chart              []ChartPoint
}

type ChartPoint struct {
	N           int // 1-based index
	Hits        int // cumulative hits
	Reference   float64 // 0.25 * N
	Code        string
	CompletedAt string
	Hit         bool
}

var allowedHistoryStates = map[string]struct{}{
	"collecting": {},
	"locked":     {},
	"completed":  {},
	"abandoned":  {},
}

func normalizeHistoryFilter(f HistoryFilter) (HistoryFilter, error) {
	var states []string
	seen := map[string]struct{}{}
	for _, st := range f.States {
		st = strings.TrimSpace(st)
		if st == "" {
			continue
		}
		if _, ok := allowedHistoryStates[st]; !ok {
			continue
		}
		if _, dup := seen[st]; dup {
			continue
		}
		seen[st] = struct{}{}
		states = append(states, st)
	}
	f.States = states
	if f.From != "" {
		if _, err := time.Parse(time.RFC3339, f.From); err != nil {
			return f, fmt.Errorf("from inválido: use RFC3339 UTC")
		}
	}
	if f.To != "" {
		if _, err := time.Parse(time.RFC3339, f.To); err != nil {
			return f, fmt.Errorf("to inválido: use RFC3339 UTC")
		}
	}
	if f.Limit <= 0 {
		f.Limit = 50
	}
	if f.Limit > 200 {
		f.Limit = 200
	}
	if f.Offset < 0 {
		f.Offset = 0
	}
	return f, nil
}

func (s *Store) ListHistory(f HistoryFilter) (HistoryPage, error) {
	f, err := normalizeHistoryFilter(f)
	if err != nil {
		return HistoryPage{}, err
	}
	where, args := historyWhere(f)
	var total int
	if err := s.DB.QueryRow(`SELECT COUNT(1) FROM sessions`+where, args...).Scan(&total); err != nil {
		return HistoryPage{}, err
	}
	q := `SELECT id, code, state, created_at, collection_ms, choice_ms, abandoned_after_alts,
		COALESCE(confirmed_choice, ''), hit, COALESCE(completed_at, ''), COALESCE(abandoned_at, '')
		FROM sessions` + where + ` ORDER BY created_at DESC, id DESC LIMIT ? OFFSET ?`
	args = append(args, f.Limit, f.Offset)
	rows, err := s.DB.Query(q, args...)
	if err != nil {
		return HistoryPage{}, err
	}
	defer rows.Close()
	items := make([]HistoryListItem, 0)
	for rows.Next() {
		var it HistoryListItem
		var afterAlts int
		var hit sql.NullInt64
		if err := rows.Scan(
			&it.ID, &it.Code, &it.State, &it.CreatedAt, &it.CollectionMs, &it.ChoiceMs, &afterAlts,
			&it.ConfirmedChoice, &hit, &it.CompletedAt, &it.AbandonedAt,
		); err != nil {
			return HistoryPage{}, err
		}
		it.AbandonedAfterAlts = afterAlts != 0
		if it.State != "completed" {
			it.ConfirmedChoice = ""
			it.Hit = nil
		} else if hit.Valid {
			v := int(hit.Int64)
			it.Hit = &v
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return HistoryPage{}, err
	}
	return HistoryPage{Items: items, Total: total}, nil
}

func historyWhere(f HistoryFilter) (string, []any) {
	var parts []string
	var args []any
	if len(f.States) > 0 {
		ph := make([]string, len(f.States))
		for i, st := range f.States {
			ph[i] = "?"
			args = append(args, st)
		}
		parts = append(parts, `state IN (`+strings.Join(ph, ",")+`)`)
	}
	if f.From != "" {
		parts = append(parts, `created_at >= ?`)
		args = append(args, f.From)
	}
	if f.To != "" {
		parts = append(parts, `created_at < ?`)
		args = append(args, f.To)
	}
	if len(parts) == 0 {
		return "", args
	}
	return ` WHERE ` + strings.Join(parts, ` AND `), args
}

func (s *Store) Stats() (Stats, error) {
	var st Stats
	err := s.DB.QueryRow(`
		SELECT
			COUNT(1),
			COALESCE(SUM(CASE WHEN state IN ('collecting', 'locked') THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN state = 'completed' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN state = 'abandoned' AND abandoned_after_alts = 0 THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN state = 'abandoned' AND abandoned_after_alts = 1 THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN state = 'completed' AND hit = 1 THEN 1 ELSE 0 END), 0)
		FROM sessions`).Scan(
		&st.Initiated, &st.Active, &st.Completed, &st.AbandonedBefore, &st.AbandonedAfter, &st.Hits,
	)
	if err != nil {
		return Stats{}, err
	}
	st.ConfirmedChoices = st.Completed
	if st.ConfirmedChoices > 0 {
		rate := float64(st.Hits) / float64(st.ConfirmedChoices)
		st.HitRate = &rate
	}
	rows, err := s.DB.Query(`
		SELECT code, completed_at, hit
		FROM sessions
		WHERE state = 'completed'
		ORDER BY completed_at ASC, id ASC`)
	if err != nil {
		return Stats{}, err
	}
	defer rows.Close()
	st.Chart = make([]ChartPoint, 0)
	cum := 0
	n := 0
	for rows.Next() {
		var code, completedAt string
		var hit int
		if err := rows.Scan(&code, &completedAt, &hit); err != nil {
			return Stats{}, err
		}
		n++
		if hit == 1 {
			cum++
		}
		st.Chart = append(st.Chart, ChartPoint{
			N: n, Hits: cum, Reference: 0.25 * float64(n),
			Code: code, CompletedAt: completedAt, Hit: hit == 1,
		})
	}
	if err := rows.Err(); err != nil {
		return Stats{}, err
	}
	return st, nil
}

// HasNonterminalSession reports whether a collecting or locked session exists.
func (s *Store) HasNonterminalSession() (bool, error) {
	var n int
	err := s.DB.QueryRow(`SELECT COUNT(1) FROM sessions WHERE state IN ('collecting', 'locked')`).Scan(&n)
	return n > 0, err
}
