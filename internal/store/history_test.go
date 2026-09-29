package store

import (
	"math"
	"testing"
	"time"
)

func seedCatalog(t *testing.T, st *Store) int64 {
	t.Helper()
	id, err := st.ActivateRevision("t", []ImageRow{{
		SHA256: "aa", OriginalRelpath: "originals/aa.jpg", DisplayRelpath: "display/aa.png",
		Width: 1, Height: 1, Bytes: 10, SourcesJSON: "[]",
	}, {
		SHA256: "bb", OriginalRelpath: "originals/bb.jpg", DisplayRelpath: "display/bb.png",
		Width: 1, Height: 1, Bytes: 10, SourcesJSON: "[]",
	}, {
		SHA256: "cc", OriginalRelpath: "originals/cc.jpg", DisplayRelpath: "display/cc.png",
		Width: 1, Height: 1, Bytes: 10, SourcesJSON: "[]",
	}, {
		SHA256: "dd", OriginalRelpath: "originals/dd.jpg", DisplayRelpath: "display/dd.png",
		Width: 1, Height: 1, Bytes: 10, SourcesJSON: "[]",
	}}, nil, map[string]any{"ready": true})
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func baseRow(id, code, op string, revID int64, ts string) *SessionRow {
	return &SessionRow{
		ID: id, Code: code, CreateOpID: op, State: "collecting", Revision: 1,
		CatalogRevisionID: revID, TargetSHA: "aa", PosA: "aa", PosB: "bb", PosC: "cc", PosD: "dd",
		ProtocolJSON: "{}", HelpVersion: "v", RecordJSON: "{}", Step: 1,
		CreatedAt: ts, UpdatedAt: ts, OpenedExamplesJSON: "[]",
	}
}

func TestStatsEmpty(t *testing.T) {
	st, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	_ = seedCatalog(t, st)
	stats, err := st.Stats()
	if err != nil {
		t.Fatal(err)
	}
	if stats.Initiated != 0 || stats.Active != 0 || stats.Completed != 0 ||
		stats.AbandonedBefore != 0 || stats.AbandonedAfter != 0 ||
		stats.ConfirmedChoices != 0 || stats.Hits != 0 || stats.HitRate != nil {
		t.Fatalf("empty stats: %+v", stats)
	}
	if len(stats.Chart) != 0 {
		t.Fatalf("chart: %v", stats.Chart)
	}
}

func TestStatsA9Example(t *testing.T) {
	st, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	rev := seedCatalog(t, st)
	ts := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC).Format(time.RFC3339)

	hit := 1
	miss := 0
	done1 := baseRow("h1", "1000–0001", "op-h1", rev, ts)
	done1.State = "completed"
	done1.LockedRecordJSON = "{}"
	done1.ConfirmedChoice = "A"
	done1.Hit = &hit
	done1.CompletedAt = "2026-09-01T12:10:00Z"
	if err := st.InsertSession(done1); err != nil {
		t.Fatal(err)
	}

	done2 := baseRow("h2", "1000–0002", "op-h2", rev, "2026-09-01T13:00:00Z")
	done2.State = "completed"
	done2.LockedRecordJSON = "{}"
	done2.ConfirmedChoice = "B"
	done2.Hit = &miss
	done2.CompletedAt = "2026-09-01T13:10:00Z"
	if err := st.InsertSession(done2); err != nil {
		t.Fatal(err)
	}

	ab := baseRow("h3", "1000–0003", "op-h3", rev, "2026-09-01T14:00:00Z")
	ab.State = "abandoned"
	ab.AbandonedAt = "2026-09-01T14:05:00Z"
	ab.AbandonedAfterAlts = false
	if err := st.InsertSession(ab); err != nil {
		t.Fatal(err)
	}

	stats, err := st.Stats()
	if err != nil {
		t.Fatal(err)
	}
	if stats.Initiated != 3 || stats.Completed != 2 || stats.AbandonedBefore != 1 ||
		stats.AbandonedAfter != 0 || stats.ConfirmedChoices != 2 || stats.Hits != 1 || stats.Active != 0 {
		t.Fatalf("counts: %+v", stats)
	}
	if stats.HitRate == nil || math.Abs(*stats.HitRate-0.5) > 1e-9 {
		t.Fatalf("hitRate: %v", stats.HitRate)
	}
	if len(stats.Chart) != 2 {
		t.Fatalf("chart len %d", len(stats.Chart))
	}
	if stats.Chart[0].N != 1 || stats.Chart[0].Hits != 1 || stats.Chart[0].Reference != 0.25 ||
		!stats.Chart[0].Hit || stats.Chart[0].Code != "1000–0001" {
		t.Fatalf("chart0: %+v", stats.Chart[0])
	}
	if stats.Chart[1].N != 2 || stats.Chart[1].Hits != 1 || stats.Chart[1].Reference != 0.5 ||
		stats.Chart[1].Hit || stats.Chart[1].Code != "1000–0002" {
		t.Fatalf("chart1: %+v", stats.Chart[1])
	}
}

func TestChartOrderByCompletedAt(t *testing.T) {
	st, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	rev := seedCatalog(t, st)
	hit := 1
	miss := 0
	// Insert later-created first, but earlier completed second — chart must follow completed_at.
	later := baseRow("c2", "code-later", "op-c2", rev, "2026-09-02T10:00:00Z")
	later.State = "completed"
	later.LockedRecordJSON = "{}"
	later.ConfirmedChoice = "A"
	later.Hit = &miss
	later.CompletedAt = "2026-09-02T12:00:00Z"
	if err := st.InsertSession(later); err != nil {
		t.Fatal(err)
	}
	earlier := baseRow("c1", "code-earlier", "op-c1", rev, "2026-09-02T11:00:00Z")
	earlier.State = "completed"
	earlier.LockedRecordJSON = "{}"
	earlier.ConfirmedChoice = "A"
	earlier.Hit = &hit
	earlier.CompletedAt = "2026-09-02T11:30:00Z"
	if err := st.InsertSession(earlier); err != nil {
		t.Fatal(err)
	}
	stats, err := st.Stats()
	if err != nil {
		t.Fatal(err)
	}
	if len(stats.Chart) != 2 || stats.Chart[0].Code != "code-earlier" || stats.Chart[1].Code != "code-later" {
		t.Fatalf("order: %+v", stats.Chart)
	}
	if stats.Chart[0].Hits != 1 || stats.Chart[1].Hits != 1 {
		t.Fatalf("cum hits: %+v", stats.Chart)
	}
}

func TestListHistoryOrderFilterBounds(t *testing.T) {
	st, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	rev := seedCatalog(t, st)
	hit := 1
	rows := []*SessionRow{
		baseRow("l1", "old", "op-l1", rev, "2026-09-10T08:00:00Z"),
		baseRow("l2", "mid", "op-l2", rev, "2026-09-10T12:00:00Z"),
		baseRow("l3", "new", "op-l3", rev, "2026-09-10T18:00:00Z"),
	}
	rows[0].State = "completed"
	rows[0].LockedRecordJSON = "{}"
	rows[0].ConfirmedChoice = "A"
	rows[0].Hit = &hit
	rows[0].CompletedAt = "2026-09-10T08:30:00Z"
	rows[1].State = "abandoned"
	rows[1].AbandonedAt = "2026-09-10T12:10:00Z"
	rows[2].State = "completed"
	rows[2].LockedRecordJSON = "{}"
	rows[2].ConfirmedChoice = "B"
	rows[2].Hit = &hit
	rows[2].CompletedAt = "2026-09-10T18:30:00Z"
	for _, r := range rows {
		if err := st.InsertSession(r); err != nil {
			t.Fatal(err)
		}
	}

	page, err := st.ListHistory(HistoryFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 3 || len(page.Items) != 3 {
		t.Fatalf("all: %+v", page)
	}
	if page.Items[0].ID != "l3" || page.Items[1].ID != "l2" || page.Items[2].ID != "l1" {
		t.Fatalf("newest first: %+v", page.Items)
	}

	page, err = st.ListHistory(HistoryFilter{States: []string{"completed", "bogus"}})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 2 || len(page.Items) != 2 {
		t.Fatalf("state filter: %+v", page)
	}
	for _, it := range page.Items {
		if it.State != "completed" {
			t.Fatalf("unexpected state %s", it.State)
		}
		if it.Hit == nil || it.ConfirmedChoice == "" {
			t.Fatalf("completed summary incomplete: %+v", it)
		}
	}

	page, err = st.ListHistory(HistoryFilter{
		From: "2026-09-10T10:00:00Z",
		To:   "2026-09-10T15:00:00Z",
	})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 1 || page.Items[0].ID != "l2" {
		t.Fatalf("date bounds: %+v", page)
	}

	page, err = st.ListHistory(HistoryFilter{States: []string{"abandoned"}})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 1 || page.Items[0].ConfirmedChoice != "" || page.Items[0].Hit != nil {
		t.Fatalf("abandoned must hide choice/hit: %+v", page.Items[0])
	}

	ok, err := st.HasNonterminalSession()
	if err != nil || ok {
		t.Fatalf("HasNonterminalSession: %v %v", ok, err)
	}
	active := baseRow("act", "active", "op-act", rev, "2026-09-11T00:00:00Z")
	if err := st.InsertSession(active); err != nil {
		t.Fatal(err)
	}
	ok, err = st.HasNonterminalSession()
	if err != nil || !ok {
		t.Fatalf("want nonterminal: %v %v", ok, err)
	}
}
