package session

import (
	"bytes"
	"errors"
	"io"
	"path/filepath"
	"testing"

	"github.com/Thiagojm/crv-go/internal/catalog"
	"github.com/Thiagojm/crv-go/internal/store"
)

type failReader struct{}

func (failReader) Read([]byte) (int, error) { return 0, errors.New("no entropy") }

func openWithCatalog(t *testing.T, n int) *store.Store {
	t.Helper()
	root := t.TempDir()
	cat := filepath.Join(root, "farsight")
	if _, err := catalog.WriteSynthetic(cat, n); err != nil {
		t.Fatal(err)
	}
	st, err := store.Open(filepath.Join(root, "data"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	if _, err := catalog.Install(st, cat); err != nil {
		t.Fatal(err)
	}
	return st
}

func TestDrawFourDistinctAndRepeatableBank(t *testing.T) {
	st := openWithCatalog(t, 4)
	svc := New(st)
	first, _, err := svc.Create(CreateInput{OperationID: "a"})
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{first.PosA: true, first.PosB: true, first.PosC: true, first.PosD: true}
	if len(seen) != 4 {
		t.Fatalf("alts not distinct: %+v", first)
	}
	if !seen[first.TargetSHA] {
		t.Fatal("target missing from order")
	}
	first.State = "completed"
	first.LockedRecordJSON = first.RecordJSON
	choice := "A"
	first.ConfirmedChoice = choice
	hit := 0
	first.Hit = &hit
	first.Revision = 2
	if err := st.SaveSession(first, 1); err != nil {
		t.Fatal(err)
	}
	second, _, err := svc.Create(CreateInput{OperationID: "b"})
	if err != nil {
		t.Fatal(err)
	}
	if second.ID == first.ID {
		t.Fatal("expected a new session")
	}
	seen2 := map[string]bool{second.PosA: true, second.PosB: true, second.PosC: true, second.PosD: true}
	if len(seen2) != 4 {
		t.Fatal("second session alts not distinct")
	}
}

func TestCreateIdempotentOpAndRandomFailure(t *testing.T) {
	st := openWithCatalog(t, 4)
	svc := New(st)
	a, _, err := svc.Create(CreateInput{OperationID: "same"})
	if err != nil {
		t.Fatal(err)
	}
	b, _, err := svc.Create(CreateInput{OperationID: "same"})
	if err != nil || b.ID != a.ID {
		t.Fatalf("idempotent create: %v %v", b, err)
	}
	if _, _, err := svc.Create(CreateInput{OperationID: "other"}); !errors.Is(err, ErrActiveExists) {
		t.Fatalf("want active exists, got %v", err)
	}
	st2 := openWithCatalog(t, 4)
	bad := New(st2)
	bad.Rand = failReader{}
	if _, _, err := bad.Create(CreateInput{OperationID: "x"}); !errors.Is(err, ErrRand) {
		t.Fatalf("want rand err, got %v", err)
	}
	active, _ := st2.ActiveSession()
	if active != nil {
		t.Fatal("failed create left a session")
	}
}

func TestRecordValidationRejectsUnknownAndBounds(t *testing.T) {
	rec := EmptyRecord()
	rec.Groups = map[string]GroupValue{"nope": {IDs: []string{"x"}}}
	if err := ValidateRecord(&rec); err == nil {
		t.Fatal("unknown group")
	}
	rec = EmptyRecord()
	rec.Groups = map[string]GroupValue{"movement": {IDs: []string{"nope"}}}
	if err := ValidateRecord(&rec); err == nil {
		t.Fatal("unknown option")
	}
	rec = EmptyRecord()
	rec.Drawings.Ideogram = []Stroke{{Points: []Point{{X: -1, Y: 0}}, Width: 3}}
	if err := ValidateRecord(&rec); err == nil {
		t.Fatal("out of bounds stroke")
	}
	rec = EmptyRecord()
	c := 101
	rec.Confidence = &c
	if err := ValidateRecord(&rec); err == nil {
		t.Fatal("confidence")
	}
}

func TestLockChoiceAbandonStateMachine(t *testing.T) {
	st := openWithCatalog(t, 4)
	svc := New(st)
	row, lease, err := svc.Create(CreateInput{OperationID: "m"})
	if err != nil {
		t.Fatal(err)
	}
	rec := EmptyRecord()
	rec.Groups = map[string]GroupValue{"movement": {IDs: []string{"curved"}, Note: "livre"}}
	saved, err := svc.SaveRecord(row.ID, lease, row.Revision, rec)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SaveRecord(row.ID, lease, row.Revision, rec); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale save: %v", err)
	}
	locked, err := svc.Lock(row.ID, lease, saved.Revision)
	if err != nil || locked.State != "locked" {
		t.Fatalf("lock: %v %+v", err, locked)
	}
	again, err := svc.Lock(row.ID, lease, locked.Revision)
	if err != nil || again.State != "locked" {
		t.Fatal("lock should be idempotent")
	}
	if _, err := svc.SaveRecord(row.ID, lease, locked.Revision, rec); !errors.Is(err, ErrBadState) {
		t.Fatalf("save after lock: %v", err)
	}
	conf := 40
	tent, err := svc.Tentative(row.ID, lease, locked.Revision, "B", &conf)
	if err != nil {
		t.Fatal(err)
	}
	done, err := svc.Confirm(row.ID, lease, tent.Revision, "B", &conf)
	if err != nil || done.State != "completed" || done.ConfirmedChoice != "B" || done.Hit == nil {
		t.Fatalf("confirm: %v %+v", err, done)
	}
	replay, err := svc.Confirm(row.ID, lease, tent.Revision, "B", &conf)
	if err != nil || replay.State != "completed" {
		t.Fatal("confirm replay")
	}
	if _, err := svc.Confirm(row.ID, lease, tent.Revision, "A", nil); !errors.Is(err, ErrChoiceConflict) {
		t.Fatalf("different choice: %v", err)
	}
	if _, err := svc.Abandon(row.ID, lease, done.Revision); !errors.Is(err, ErrBadState) {
		t.Fatalf("abandon completed: %v", err)
	}
}

func TestAbandonHidesTarget(t *testing.T) {
	st := openWithCatalog(t, 4)
	svc := New(st)
	row, lease, err := svc.Create(CreateInput{OperationID: "ab"})
	if err != nil {
		t.Fatal(err)
	}
	gone, err := svc.Abandon(row.ID, lease, row.Revision)
	if err != nil || gone.State != "abandoned" || gone.AbandonedAfterAlts {
		t.Fatalf("abandon collecting: %+v %v", gone, err)
	}
	if ImagesAuthorized(gone) {
		t.Fatal("no images before lock")
	}
	row2, lease2, err := svc.Create(CreateInput{OperationID: "ab2"})
	if err != nil {
		t.Fatal(err)
	}
	locked, err := svc.Lock(row2.ID, lease2, row2.Revision)
	if err != nil {
		t.Fatal(err)
	}
	gone2, err := svc.Abandon(row2.ID, lease2, locked.Revision)
	if err != nil || !gone2.AbandonedAfterAlts || !ImagesAuthorized(gone2) {
		t.Fatalf("abandon locked: %+v %v", gone2, err)
	}
	if gone2.ConfirmedChoice != "" || gone2.Hit != nil {
		t.Fatal("abandoned must not confirm")
	}
}

func TestTimingDoesNotBumpRevision(t *testing.T) {
	st := openWithCatalog(t, 4)
	svc := New(st)
	row, lease, err := svc.Create(CreateInput{OperationID: "tm"})
	if err != nil {
		t.Fatal(err)
	}
	rev := row.Revision
	got, err := svc.Timing(row.ID, lease, 1, 9000)
	if err != nil || got.Revision != rev || got.CollectionMs != 5000 || got.TimingSeq != 1 {
		t.Fatalf("timing: %+v %v", got, err)
	}
	again, err := svc.Timing(row.ID, lease, 1, 100)
	if err != nil || again.CollectionMs != 5000 {
		t.Fatal("timing replay")
	}
}

func TestLeaseTransferAndStaleWriter(t *testing.T) {
	st := openWithCatalog(t, 4)
	svc := New(st)
	row, lease, err := svc.Create(CreateInput{OperationID: "ls"})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.Heartbeat(row.ID, "other", false); !errors.Is(err, ErrLease) {
		t.Fatalf("want lease held, %v", err)
	}
	staleRev := row.Revision
	next, transferred, err := svc.Heartbeat(row.ID, "other", true)
	if err != nil || next == "" || next == lease {
		t.Fatalf("transfer: %s %v", next, err)
	}
	if transferred.Revision <= staleRev {
		t.Fatalf("transfer must bump revision: %d -> %d", staleRev, transferred.Revision)
	}
	if _, err := svc.SaveRecord(row.ID, lease, row.Revision, EmptyRecord()); !errors.Is(err, ErrLease) {
		t.Fatalf("stale lease save: %v", err)
	}
	stale := *row
	stale.LeaseToken = lease
	stale.Revision = staleRev
	if err := st.PatchSession(&stale, staleRev); !errors.Is(err, store.ErrConflict) {
		t.Fatalf("stale patch after transfer: %v", err)
	}
	got, err := svc.Get(row.ID)
	if err != nil || got.LeaseToken != next || got.State != "collecting" {
		t.Fatalf("lease restored? %+v %v", got, err)
	}
}

func TestStaleHeartbeatCannotRevertCompleted(t *testing.T) {
	st := openWithCatalog(t, 4)
	svc := New(st)
	row, lease, err := svc.Create(CreateInput{OperationID: "hb"})
	if err != nil {
		t.Fatal(err)
	}
	locked, err := svc.Lock(row.ID, lease, row.Revision)
	if err != nil {
		t.Fatal(err)
	}
	stale := *locked
	done, err := svc.Confirm(row.ID, lease, locked.Revision, "A", nil)
	if err != nil || done.State != "completed" {
		t.Fatalf("confirm: %+v %v", done, err)
	}
	stale.State = "locked"
	if err := st.PatchSession(&stale, stale.Revision); !errors.Is(err, store.ErrConflict) {
		t.Fatalf("stale locked patch: %v", err)
	}
	got, err := svc.Get(row.ID)
	if err != nil || got.State != "completed" {
		t.Fatalf("state reverted: %+v %v", got, err)
	}
	_, again, err := svc.Heartbeat(row.ID, lease, false)
	if err != nil || again.State != "completed" {
		t.Fatalf("heartbeat after complete: %+v %v", again, err)
	}
}

func TestDrawUsesReaderBytes(t *testing.T) {
	stream := bytes.Repeat([]byte{0, 0, 0, 0, 0, 0, 0, 1}, 64)
	target, order, err := drawAssignment(bytes.NewReader(stream), []string{"a", "b", "c", "d"})
	if err != nil {
		t.Fatal(err)
	}
	if target == "" || len(map[string]bool{order[0]: true, order[1]: true, order[2]: true, order[3]: true}) != 4 {
		t.Fatalf("%s %v", target, order)
	}
	if _, _, err := drawAssignment(bytes.NewReader(nil), []string{"a", "b", "c", "d"}); !errors.Is(err, ErrRand) {
		t.Fatalf("empty reader: %v", err)
	}
}

func TestFailReaderIsEOFShape(t *testing.T) {
	if _, err := io.ReadAll(failReader{}); err == nil {
		t.Fatal("expected error")
	}
}
