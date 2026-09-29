package export

import (
	"encoding/csv"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Thiagojm/crv-go/internal/catalog"
	"github.com/Thiagojm/crv-go/internal/session"
	"github.com/Thiagojm/crv-go/internal/store"
)

func TestNeutralizeCSV(t *testing.T) {
	if got := NeutralizeCSV("=1+1"); got != "'=1+1" {
		t.Fatalf("formula: %q", got)
	}
	if got := NeutralizeCSV("+x"); got != "'+x" {
		t.Fatalf("plus: %q", got)
	}
	if got := NeutralizeCSV("-1"); got != "'-1" {
		t.Fatalf("minus: %q", got)
	}
	if got := NeutralizeCSV("@cmd"); got != "'@cmd" {
		t.Fatalf("at: %q", got)
	}
	if got := NeutralizeCSV("texto normal"); got != "texto normal" {
		t.Fatalf("plain: %q", got)
	}
}

func TestBuildCSVQuotingAndBlindness(t *testing.T) {
	hit := true
	conf := 70
	rows := []SessionReport{{
		Code:            "1000–0001",
		CreatedAt:       "2026-09-01T12:00:00Z",
		CompletedAt:     "2026-09-01T12:10:00Z",
		State:           "completed",
		CollectionMs:    1000,
		ChoiceMs:        200,
		ConfirmedChoice: "A",
		Hit:             &hit,
		Confidence:      &conf,
		Comment:         "=SUM(A1)\nlinha, com vírgula",
		TargetLabel:     "should-not-appear",
		TargetCredit:    "fixture",
		TargetDescription: "farsight secret",
		CorrectPosition: "A",
		RecordJSON:      `{"version":"crv-record-v1"}`,
	}, {
		Code:        "1000–0002",
		CreatedAt:   "2026-09-01T13:00:00Z",
		AbandonedAt: "2026-09-01T13:05:00Z",
		State:       "abandoned",
		Comment:     "+hack",
	}}

	raw, err := BuildCSV(rows)
	if err != nil {
		t.Fatal(err)
	}
	out := string(raw)
	for _, bad := range []string{"sha256", "farsight", "originals/", "display/", "should-not-appear", "TargetLabel", "target_sha"} {
		if strings.Contains(strings.ToLower(out), strings.ToLower(bad)) {
			t.Fatalf("CSV leaked %q:\n%s", bad, out)
		}
	}

	r := csv.NewReader(strings.NewReader(out))
	recs, err := r.ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 3 {
		t.Fatalf("rows: %d", len(recs))
	}
	completed := recs[1]
	if completed[8] != "A" || completed[9] != "true" {
		t.Fatalf("completed choice/hit: %#v", completed)
	}
	if !strings.HasPrefix(completed[11], "'=SUM") {
		t.Fatalf("comment not neutralized: %q", completed[11])
	}
	if !strings.Contains(completed[11], "\n") || !strings.Contains(completed[11], ",") {
		t.Fatalf("comment lost newline/comma: %q", completed[11])
	}
	abandoned := recs[2]
	if abandoned[8] != "" || abandoned[9] != "" {
		t.Fatalf("abandoned should have empty choice/hit: %#v", abandoned)
	}
	if abandoned[11] != "'+hack" {
		t.Fatalf("abandoned comment: %q", abandoned[11])
	}
}

func seedSynthStore(t *testing.T) (*store.Store, int64, []string) {
	t.Helper()
	dataDir := t.TempDir()
	catDir := filepath.Join(dataDir, "synth-cat")
	hashes, err := catalog.WriteSynthetic(catDir, 4)
	if err != nil {
		t.Fatal(err)
	}
	st, err := store.Open(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	res, err := catalog.EnsureInstalled(st, catDir)
	if err != nil {
		t.Fatal(err)
	}
	if res.RevisionID == 0 || !res.Report.Ready {
		t.Fatalf("catalog not ready: %+v", res)
	}
	return st, res.RevisionID, hashes
}

func baseSession(id, code, op string, rev int64, hashes []string, ts string) *store.SessionRow {
	rec := session.EmptyRecord()
	rec.Disposition = "calmo"
	rec.Concentration = "Moderada"
	rec.AOL1 = "hipótese"
	rec.Drawings.Ideogram = []session.Stroke{{
		Width:  3,
		Points: []session.Point{{X: 10, Y: 10}, {X: 40, Y: 50}},
	}}
	raw, _ := json.Marshal(rec)
	return &store.SessionRow{
		ID: id, Code: code, CreateOpID: op, State: "collecting", Revision: 1,
		CatalogRevisionID: rev,
		TargetSHA:         hashes[0],
		PosA:              hashes[0], PosB: hashes[1], PosC: hashes[2], PosD: hashes[3],
		ProtocolJSON:      `{"version":"crv-record-v1"}`, HelpVersion: session.HelpVersion,
		RecordJSON: string(raw), Step: 1,
		CreatedAt: ts, UpdatedAt: ts, OpenedExamplesJSON: "[]",
	}
}

func TestBuildSessionReportCompletedVsAbandoned(t *testing.T) {
	st, rev, hashes := seedSynthStore(t)
	ts := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC).Format(time.RFC3339)

	hit := 1
	conf := 80
	done := baseSession("e1", "2000–0001", "op-e1", rev, hashes, ts)
	done.State = "completed"
	done.LockedRecordJSON = done.RecordJSON
	done.ConfirmedChoice = "A"
	done.ConfirmedConfidence = &conf
	done.Hit = &hit
	done.CompletedAt = "2026-09-28T12:10:00Z"
	done.Comment = "ok"
	if err := st.InsertSession(done); err != nil {
		t.Fatal(err)
	}

	ab := baseSession("e2", "2000–0002", "op-e2", rev, hashes, "2026-09-28T13:00:00Z")
	ab.State = "abandoned"
	ab.AbandonedAt = "2026-09-28T13:05:00Z"
	if err := st.InsertSession(ab); err != nil {
		t.Fatal(err)
	}

	gotDone, err := BuildSessionReport(done, st)
	if err != nil {
		t.Fatal(err)
	}
	if gotDone.TargetCredit != "fixture" {
		t.Fatalf("completed credit: %q", gotDone.TargetCredit)
	}
	if gotDone.TargetLabel == "" || !strings.Contains(gotDone.TargetLabel, "Alvo sintético") {
		t.Fatalf("completed label: %q", gotDone.TargetLabel)
	}
	if gotDone.CorrectPosition != "A" || gotDone.ConfirmedChoice != "A" || gotDone.Hit == nil || !*gotDone.Hit {
		t.Fatalf("completed fields: %+v", gotDone)
	}
	if gotDone.TargetImageDataURI == "" || !strings.HasPrefix(gotDone.TargetImageDataURI, "data:image/png;base64,") {
		t.Fatalf("expected data URI, got %q", gotDone.TargetImageDataURI)
	}

	gotAb, err := BuildSessionReport(ab, st)
	if err != nil {
		t.Fatal(err)
	}
	if gotAb.TargetCredit != "" || gotAb.TargetLabel != "" || gotAb.TargetImageDataURI != "" ||
		gotAb.ConfirmedChoice != "" || gotAb.Hit != nil || gotAb.CorrectPosition != "" {
		t.Fatalf("abandoned leaked target fields: %+v", gotAb)
	}
}

func TestRenderSessionHTML(t *testing.T) {
	st, rev, hashes := seedSynthStore(t)
	ts := "2026-09-28T15:00:00Z"
	hit := 1
	done := baseSession("h1", "3000–0001", "op-h1", rev, hashes, ts)
	done.State = "completed"
	done.LockedRecordJSON = done.RecordJSON
	done.ConfirmedChoice = "A"
	done.Hit = &hit
	done.CompletedAt = "2026-09-28T15:10:00Z"
	repDone, err := BuildSessionReport(done, st)
	if err != nil {
		t.Fatal(err)
	}
	htmlDone, err := RenderSessionHTML(repDone)
	if err != nil {
		t.Fatal(err)
	}
	sDone := string(htmlDone)
	if !strings.Contains(sDone, "3000–0001") {
		t.Fatal("missing code")
	}
	if !strings.Contains(sDone, "Use Imprimir / Salvar como PDF do navegador.") {
		t.Fatal("missing print note")
	}
	if !strings.Contains(sDone, "fixture") {
		t.Fatal("completed HTML missing credit")
	}
	if !strings.Contains(sDone, "polyline") {
		t.Fatal("missing drawing svg")
	}
	if !strings.Contains(sDone, `src="data:image/png;base64,`) {
		snippet := sDone
		if len(snippet) > 800 {
			snippet = snippet[:800]
		}
		t.Fatalf("target image data URI missing or escaped away:\n%s", snippet)
	}
	if strings.Contains(sDone, "#ZgotmplZ") {
		t.Fatal("html/template URL filter broke the target image src")
	}
	if !strings.Contains(sDone, `drawing-block-continued`) || !strings.Contains(sDone, "Esboço") {
		t.Fatal("missing continued drawing-block markup for Esboço")
	}

	ab := baseSession("h2", "3000–0002", "op-h2", rev, hashes, ts)
	ab.State = "abandoned"
	ab.AbandonedAt = "2026-09-28T15:20:00Z"
	repAb, err := BuildSessionReport(ab, st)
	if err != nil {
		t.Fatal(err)
	}
	// even if credit field stays empty, HTML must not invent target credit
	repAb.TargetCredit = ""
	htmlAb, err := RenderSessionHTML(repAb)
	if err != nil {
		t.Fatal(err)
	}
	sAb := string(htmlAb)
	if !strings.Contains(sAb, "3000–0002") {
		t.Fatal("abandoned missing code")
	}
	if strings.Contains(sAb, "fixture") || strings.Contains(sAb, "Crédito") {
		t.Fatalf("abandoned HTML leaked credit:\n%s", sAb)
	}
}

func TestListReports(t *testing.T) {
	st, rev, hashes := seedSynthStore(t)
	ts := "2026-09-28T16:00:00Z"
	hit := 0
	done := baseSession("l1", "4000–0001", "op-l1", rev, hashes, ts)
	done.State = "completed"
	done.LockedRecordJSON = done.RecordJSON
	done.ConfirmedChoice = "B"
	done.Hit = &hit
	done.CompletedAt = "2026-09-28T16:10:00Z"
	if err := st.InsertSession(done); err != nil {
		t.Fatal(err)
	}
	reps, err := ListReports(st, store.HistoryFilter{States: []string{"completed"}, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(reps) != 1 || reps[0].Code != "4000–0001" || reps[0].ConfirmedChoice != "B" {
		t.Fatalf("%+v", reps)
	}
}
