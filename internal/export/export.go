package export

import (
	"bytes"
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Thiagojm/crv-go/internal/catalog"
	"github.com/Thiagojm/crv-go/internal/session"
	"github.com/Thiagojm/crv-go/internal/store"
)

const ReportFormat = "crv-export-report-v1"

// SessionReport is a state-filtered model reused by CSV and printable HTML.
type SessionReport struct {
	ID, Code, State, CreatedAt, CompletedAt, AbandonedAt string
	CollectionMs, ChoiceMs                               int
	AbandonedAfterAlts                                   bool
	ProtocolVersion, HelpVersion                         string
	ConfirmedChoice                                      string // only when completed
	Hit                                                  *bool  // only when completed
	Confidence                                           *int   // only when completed (from confirmed confidence)
	Comment                                              string // post-feedback comment when present
	RecordJSON                                           string // raw locked or current record JSON as stored
	// Completed-only disclosure:
	TargetLabel, TargetDescription, TargetCredit string
	TargetImageDataURI                           string // image/png;base64,... from display file; empty if unavailable
	CorrectPosition                              string // A-D when completed
}

// NeutralizeCSV prefixes spreadsheet-formula-like strings with a single quote.
func NeutralizeCSV(s string) string {
	t := strings.TrimLeft(s, " \t")
	if t == "" {
		return s
	}
	switch t[0] {
	case '=', '+', '-', '@', '\t', '\r':
		return "'" + s
	}
	return s
}

// BuildCSV writes one CSV row per session with permitted columns only.
func BuildCSV(rows []SessionReport) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	header := []string{
		"code", "created_at", "completed_at", "abandoned_at", "state",
		"abandoned_after_alts", "collection_ms", "choice_ms",
		"confirmed_choice", "hit", "confidence", "comment",
	}
	if err := w.Write(header); err != nil {
		return nil, err
	}
	for _, r := range rows {
		hit := ""
		if r.Hit != nil {
			if *r.Hit {
				hit = "true"
			} else {
				hit = "false"
			}
		}
		conf := ""
		if r.Confidence != nil {
			conf = strconv.Itoa(*r.Confidence)
		}
		rec := []string{
			NeutralizeCSV(r.Code),
			r.CreatedAt,
			r.CompletedAt,
			r.AbandonedAt,
			r.State,
			strconv.FormatBool(r.AbandonedAfterAlts),
			strconv.Itoa(r.CollectionMs),
			strconv.Itoa(r.ChoiceMs),
			NeutralizeCSV(r.ConfirmedChoice),
			hit,
			conf,
			NeutralizeCSV(r.Comment),
		}
		if err := w.Write(rec); err != nil {
			return nil, err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// BuildSessionReport fills permitted fields from a session row by state.
func BuildSessionReport(row *store.SessionRow, st *store.Store) (SessionReport, error) {
	if row == nil {
		return SessionReport{}, fmt.Errorf("sessão ausente")
	}
	rep := SessionReport{
		ID:                 row.ID,
		Code:               row.Code,
		State:              row.State,
		CreatedAt:          row.CreatedAt,
		CompletedAt:        row.CompletedAt,
		AbandonedAt:        row.AbandonedAt,
		CollectionMs:       row.CollectionMs,
		ChoiceMs:           row.ChoiceMs,
		AbandonedAfterAlts: row.AbandonedAfterAlts,
		HelpVersion:        row.HelpVersion,
		Comment:            row.Comment,
	}
	if row.LockedRecordJSON != "" {
		rep.RecordJSON = row.LockedRecordJSON
	} else {
		rep.RecordJSON = row.RecordJSON
	}
	var proto struct {
		Version string `json:"version"`
	}
	_ = json.Unmarshal([]byte(row.ProtocolJSON), &proto)
	rep.ProtocolVersion = proto.Version

	if row.State != "completed" {
		return rep, nil
	}

	rep.ConfirmedChoice = row.ConfirmedChoice
	if row.Hit != nil {
		b := *row.Hit == 1
		rep.Hit = &b
	}
	rep.Confidence = row.ConfirmedConfidence
	rep.CorrectPosition = session.CorrectPos(row)

	if st == nil {
		return rep, nil
	}
	img, err := st.Image(row.CatalogRevisionID, row.TargetSHA)
	if err != nil {
		return SessionReport{}, err
	}
	if img != nil {
		label, desc, credit := primarySourceMeta(img)
		rep.TargetLabel = label
		rep.TargetDescription = desc
		rep.TargetCredit = credit
		if img.DisplayRelpath != "" {
			path := filepath.Join(store.RevisionDir(st.DataDir, row.CatalogRevisionID), filepath.FromSlash(img.DisplayRelpath))
			if data, err := os.ReadFile(path); err == nil && len(data) > 0 {
				rep.TargetImageDataURI = "data:image/png;base64," + base64.StdEncoding.EncodeToString(data)
			}
		}
	}
	if rep.TargetLabel == "" {
		rep.TargetLabel = "Alvo"
	}
	return rep, nil
}

func primarySourceMeta(img *store.ImageRow) (label, description, credit string) {
	var sources []catalog.SourceRecord
	_ = json.Unmarshal([]byte(img.SourcesJSON), &sources)
	for _, src := range sources {
		if src.ID == img.PrimarySourceID || label == "" {
			label = src.Label
			if label == "" {
				label = src.ID
			}
			description = src.Description
			credit = src.Credit
		}
	}
	return label, description, credit
}

// ListReports builds SessionReport rows for all sessions matching the history filter
// (pages through the store; f.Limit/Offset are ignored for completeness).
func ListReports(st *store.Store, f store.HistoryFilter) ([]SessionReport, error) {
	f.Limit = 200
	f.Offset = 0
	var out []SessionReport
	for {
		page, err := st.ListHistory(f)
		if err != nil {
			return nil, err
		}
		for _, it := range page.Items {
			row, err := st.GetSession(it.ID)
			if err != nil {
				return nil, err
			}
			if row == nil {
				continue
			}
			rep, err := BuildSessionReport(row, st)
			if err != nil {
				return nil, err
			}
			out = append(out, rep)
		}
		f.Offset += len(page.Items)
		if f.Offset >= page.Total || len(page.Items) == 0 {
			break
		}
	}
	return out, nil
}
