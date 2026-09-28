package store

import (
	"errors"
	"testing"
	"time"
)

func TestSessionNonterminalConstraintAndPersist(t *testing.T) {
	st, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
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
	ts := time.Now().UTC().Format(time.RFC3339)
	row := &SessionRow{
		ID: "s1", Code: "1000–2000", CreateOpID: "op1", State: "collecting", Revision: 1,
		CatalogRevisionID: id, TargetSHA: "aa", PosA: "aa", PosB: "bb", PosC: "cc", PosD: "dd",
		ProtocolJSON: "{}", HelpVersion: "v", RecordJSON: "{}", Step: 1,
		CreatedAt: ts, UpdatedAt: ts, OpenedExamplesJSON: "[]",
	}
	if err := st.InsertSession(row); err != nil {
		t.Fatal(err)
	}
	dup := *row
	dup.ID, dup.Code, dup.CreateOpID = "s2", "1000–2001", "op2"
	if err := st.InsertSession(&dup); err == nil || !IsUniqueErr(err) {
		t.Fatalf("expected unique active session, err=%v", err)
	}
	got, err := st.ActiveSession()
	if err != nil || got == nil || got.ID != "s1" {
		t.Fatalf("active: %v %v", got, err)
	}
	row.State = "completed"
	row.LockedRecordJSON = "{}"
	row.ConfirmedChoice = "A"
	hit := 1
	row.Hit = &hit
	row.Revision = 2
	row.CompletedAt = ts
	if err := st.SaveSession(row, 1); err != nil {
		t.Fatal(err)
	}
	if err := st.InsertSession(&dup); err != nil {
		t.Fatal(err)
	}
}

func TestPatchSessionRequiresRevision(t *testing.T) {
	st, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
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
	ts := time.Now().UTC().Format(time.RFC3339)
	row := &SessionRow{
		ID: "p1", Code: "1000–3000", CreateOpID: "opp", State: "collecting", Revision: 1,
		CatalogRevisionID: id, TargetSHA: "aa", PosA: "aa", PosB: "bb", PosC: "cc", PosD: "dd",
		ProtocolJSON: "{}", HelpVersion: "v", RecordJSON: "{}", Step: 1,
		CreatedAt: ts, UpdatedAt: ts, OpenedExamplesJSON: "[]", LeaseToken: "tok", LeaseUntil: ts,
	}
	if err := st.InsertSession(row); err != nil {
		t.Fatal(err)
	}
	row.LeaseToken = "other"
	row.Revision = 2
	if err := st.PatchSession(row, 1); err != nil {
		t.Fatal(err)
	}
	stale := *row
	stale.State = "locked"
	stale.LeaseToken = "tok"
	if err := st.PatchSession(&stale, 1); !errors.Is(err, ErrConflict) {
		t.Fatalf("want conflict on stale revision, got %v", err)
	}
	got, err := st.GetSession("p1")
	if err != nil || got.State != "collecting" || got.LeaseToken != "other" || got.Revision != 2 {
		t.Fatalf("patch clobbered: %+v %v", got, err)
	}
}
