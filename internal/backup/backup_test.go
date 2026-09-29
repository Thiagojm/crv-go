package backup

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Thiagojm/crv-go/internal/catalog"
	"github.com/Thiagojm/crv-go/internal/store"
)

func setupInstalled(t *testing.T) (dataDir string, st *store.Store, revID int64, sessionCode string) {
	t.Helper()
	root := t.TempDir()
	cat := filepath.Join(root, "farsight")
	if _, err := catalog.WriteSynthetic(cat, 4); err != nil {
		t.Fatal(err)
	}
	dataDir = filepath.Join(root, "data")
	st, err := store.Open(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	res, err := catalog.EnsureInstalled(st, cat)
	if err != nil || !res.Report.Ready {
		t.Fatalf("install: %+v %v", res, err)
	}
	revID = res.RevisionID
	if err := st.SetPreference("theme", "dark"); err != nil {
		t.Fatal(err)
	}
	ts := time.Now().UTC().Format(time.RFC3339)
	sessionCode = "1000–4242"
	hit := 1
	row := &store.SessionRow{
		ID: "sess-backup-1", Code: sessionCode, CreateOpID: "op-backup-1",
		State: "completed", Revision: 2, CatalogRevisionID: revID,
		TargetSHA: "x", PosA: "a", PosB: "b", PosC: "c", PosD: "d",
		ProtocolJSON: "{}", HelpVersion: "v", RecordJSON: "{}", LockedRecordJSON: "{}",
		Step: 4, ConfirmedChoice: "A", Hit: &hit, CreatedAt: ts, UpdatedAt: ts, CompletedAt: ts,
		OpenedExamplesJSON: "[]",
	}
	// Fill real assignment SHAs from revision images.
	imgs, err := st.RevisionImages(revID)
	if err != nil || len(imgs) < 4 {
		t.Fatalf("images: %v", err)
	}
	row.TargetSHA = imgs[0].SHA256
	row.PosA, row.PosB, row.PosC, row.PosD = imgs[0].SHA256, imgs[1].SHA256, imgs[2].SHA256, imgs[3].SHA256
	if err := st.InsertSession(row); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return dataDir, st, revID, sessionCode
}

func TestCreateRestoreRoundTrip(t *testing.T) {
	dataDir, st, revID, sessionCode := setupInstalled(t)

	imgs, err := st.RevisionImages(revID)
	if err != nil {
		t.Fatal(err)
	}
	displayPath := filepath.Join(store.RevisionDir(dataDir, revID), filepath.FromSlash(imgs[0].DisplayRelpath))
	wantHash, err := sha256File(displayPath)
	if err != nil {
		t.Fatal(err)
	}

	out := filepath.Join(dataDir, "backups", "roundtrip.zip")
	if err := Create(st, out); err != nil {
		t.Fatalf("Create: %v", err)
	}
	fi, err := os.Stat(out)
	if err != nil || fi.Size() == 0 {
		t.Fatalf("backup missing: %v", err)
	}

	// Mutate preferences while catalog files are still intact so pre-backup can succeed.
	if err := st.SetPreference("theme", "light"); err != nil {
		t.Fatal(err)
	}
	pre := filepath.Join(dataDir, "backups", "pre-restore.zip")
	if err := Create(st, pre); err != nil {
		t.Fatalf("pre-backup: %v", err)
	}
	_ = os.Remove(displayPath)

	staging := DefaultStagingDir(dataDir)
	if err := ValidateArchive(out, staging); err != nil {
		t.Fatalf("ValidateArchive: %v", err)
	}
	if err := PrepareSwap(dataDir, staging, pre); err != nil {
		t.Fatalf("PrepareSwap: %v", err)
	}
	if err := ApplySwap(dataDir, st.Close); err != nil {
		t.Fatalf("ApplySwap: %v", err)
	}
	if err := CommitSwap(dataDir); err != nil {
		t.Fatalf("CommitSwap: %v", err)
	}

	st2, err := store.Open(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer st2.Close()

	got, err := st2.GetSession("sess-backup-1")
	if err != nil || got == nil {
		t.Fatalf("session missing after restore: %v", err)
	}
	if got.Code != sessionCode {
		t.Fatalf("code=%q want %q", got.Code, sessionCode)
	}
	theme, ok, err := st2.GetPreference("theme")
	if err != nil || !ok || theme != "dark" {
		t.Fatalf("preference theme=%q ok=%v err=%v", theme, ok, err)
	}
	gotHash, err := sha256File(displayPath)
	if err != nil {
		t.Fatalf("display after restore: %v", err)
	}
	if gotHash != wantHash {
		t.Fatalf("display hash mismatch")
	}
	if _, err := os.Stat(MarkerPath(dataDir)); !os.IsNotExist(err) {
		t.Fatalf("marker should be gone, err=%v", err)
	}
}

func TestValidateArchiveRejectsCorruptAndWrongFormat(t *testing.T) {
	dir := t.TempDir()

	corrupt := filepath.Join(dir, "corrupt.zip")
	if err := os.WriteFile(corrupt, []byte("not-a-zip"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := ValidateArchive(corrupt, filepath.Join(dir, "stg-corrupt")); err == nil {
		t.Fatal("expected corrupt zip rejection")
	}

	badFmt := filepath.Join(dir, "badfmt.zip")
	if err := writeRawZip(badFmt, map[string][]byte{
		"manifest.json":    []byte(`{"format":"crv-backup-v999","createdAt":"x","files":{},"revisionIds":[],"note":""}`),
		"crv.sqlite":       []byte("x"),
		"preferences.json": []byte(`{}`),
	}); err != nil {
		t.Fatal(err)
	}
	if err := ValidateArchive(badFmt, filepath.Join(dir, "stg-fmt")); err == nil || !strings.Contains(err.Error(), "não suportado") {
		t.Fatalf("want unsupported format, got %v", err)
	}
}

func TestValidateArchiveRejectsTraversalManifest(t *testing.T) {
	dir := t.TempDir()
	// Build a minimal valid-looking sqlite via store, then craft evil manifest.
	st, err := store.Open(filepath.Join(dir, "seed"))
	if err != nil {
		t.Fatal(err)
	}
	snap := filepath.Join(dir, "crv.sqlite")
	if err := SnapshotDB(st, snap); err != nil {
		t.Fatal(err)
	}
	_ = st.Close()
	sum, err := sha256File(snap)
	if err != nil {
		t.Fatal(err)
	}
	dbBytes, err := os.ReadFile(snap)
	if err != nil {
		t.Fatal(err)
	}
	man := Manifest{
		Format:      FormatVersion,
		CreatedAt:   time.Now().UTC().Format(time.RFC3339),
		Files:       map[string]string{"crv.sqlite": sum, "../evil.txt": sum},
		RevisionIDs: nil,
		Note:        manifestNote,
	}
	raw, _ := json.Marshal(man)
	zipPath := filepath.Join(dir, "trav.zip")
	if err := writeRawZip(zipPath, map[string][]byte{
		"manifest.json":    raw,
		"crv.sqlite":       dbBytes,
		"preferences.json": []byte(`{}`),
		"catalog/.keep":    []byte("x"),
	}); err != nil {
		t.Fatal(err)
	}
	// Either ExtractZip or manifest path check must reject.
	err = ValidateArchive(zipPath, filepath.Join(dir, "stg-trav"))
	if err == nil {
		t.Fatal("expected traversal rejection")
	}
}

func TestValidateArchiveRejectsHashMismatch(t *testing.T) {
	dataDir, st, _, _ := setupInstalled(t)
	out := filepath.Join(dataDir, "backups", "good.zip")
	if err := Create(st, out); err != nil {
		t.Fatal(err)
	}
	_ = st.Close()

	// Tamper with zip entry bytes while keeping manifest hashes.
	tampered := filepath.Join(dataDir, "backups", "tampered.zip")
	if err := tamperZipFile(out, tampered, "preferences.json", []byte(`{"hacked":"1"}`)); err != nil {
		t.Fatal(err)
	}
	err := ValidateArchive(tampered, filepath.Join(dataDir, "stg-badhash"))
	if err == nil || !strings.Contains(err.Error(), "hash") {
		t.Fatalf("want hash mismatch, got %v", err)
	}
}

func TestResolveInterruptedPrebackupClearsStaging(t *testing.T) {
	dataDir, st, _, _ := setupInstalled(t)
	out := filepath.Join(dataDir, "backups", "b.zip")
	if err := Create(st, out); err != nil {
		t.Fatal(err)
	}
	staging := DefaultStagingDir(dataDir)
	if err := ValidateArchive(out, staging); err != nil {
		t.Fatal(err)
	}
	pre := filepath.Join(dataDir, "backups", "pre.zip")
	if err := Create(st, pre); err != nil {
		t.Fatal(err)
	}
	if err := PrepareSwap(dataDir, staging, pre); err != nil {
		t.Fatal(err)
	}
	_ = st.Close()

	if err := ResolveInterrupted(dataDir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(MarkerPath(dataDir)); !os.IsNotExist(err) {
		t.Fatal("marker should be removed")
	}
	if _, err := os.Stat(staging); !os.IsNotExist(err) {
		t.Fatal("staging should be removed")
	}
	// Live DB still opens.
	st2, err := store.Open(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer st2.Close()
	if got, err := st2.GetSession("sess-backup-1"); err != nil || got == nil {
		t.Fatalf("live session lost: %v", err)
	}
}

func TestResolveInterruptedLiveMovedGoodNewCommits(t *testing.T) {
	dataDir, st, revID, sessionCode := setupInstalled(t)
	out := filepath.Join(dataDir, "backups", "b.zip")
	if err := Create(st, out); err != nil {
		t.Fatal(err)
	}

	// Change preference so we can detect restore content.
	_ = st.SetPreference("theme", "light")

	staging := DefaultStagingDir(dataDir)
	if err := ValidateArchive(out, staging); err != nil {
		t.Fatal(err)
	}
	pre := filepath.Join(dataDir, "backups", "pre.zip")
	if err := Create(st, pre); err != nil {
		t.Fatal(err)
	}
	if err := PrepareSwap(dataDir, staging, pre); err != nil {
		t.Fatal(err)
	}
	if err := ApplySwap(dataDir, st.Close); err != nil {
		t.Fatal(err)
	}
	// Leave marker at new_inplace (skip CommitSwap).
	m, err := readMarker(dataDir)
	if err != nil || m.Stage != StageNewInPlace {
		t.Fatalf("marker: %+v %v", m, err)
	}

	if err := ResolveInterrupted(dataDir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(MarkerPath(dataDir)); !os.IsNotExist(err) {
		t.Fatal("marker should be gone after commit")
	}
	st2, err := store.Open(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer st2.Close()
	got, err := st2.GetSession("sess-backup-1")
	if err != nil || got == nil || got.Code != sessionCode {
		t.Fatalf("session after resolve: %+v %v", got, err)
	}
	theme, ok, _ := st2.GetPreference("theme")
	if !ok || theme != "dark" {
		t.Fatalf("theme=%q (restored backup should be dark)", theme)
	}
	if _, err := os.Stat(store.RevisionDir(dataDir, revID)); err != nil {
		t.Fatal(err)
	}
}

func TestResolveInterruptedLiveMovedWithGoodNewCommits(t *testing.T) {
	dataDir, st, _, sessionCode := setupInstalled(t)
	out := filepath.Join(dataDir, "backups", "b.zip")
	if err := Create(st, out); err != nil {
		t.Fatal(err)
	}
	staging := DefaultStagingDir(dataDir)
	if err := ValidateArchive(out, staging); err != nil {
		t.Fatal(err)
	}
	pre := filepath.Join(dataDir, "backups", "pre.zip")
	if err := Create(st, pre); err != nil {
		t.Fatal(err)
	}
	if err := PrepareSwap(dataDir, staging, pre); err != nil {
		t.Fatal(err)
	}
	if err := ApplySwap(dataDir, st.Close); err != nil {
		t.Fatal(err)
	}
	// Simulate crash after payload install but before new_inplace was persisted.
	m, err := readMarker(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	m.Stage = StageLiveMoved
	if err := writeMarker(dataDir, *m); err != nil {
		t.Fatal(err)
	}
	if err := ResolveInterrupted(dataDir); err != nil {
		t.Fatal(err)
	}
	st2, err := store.Open(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer st2.Close()
	got, err := st2.GetSession("sess-backup-1")
	if err != nil || got == nil || got.Code != sessionCode {
		t.Fatalf("session after live_moved resolve: %+v %v", got, err)
	}
}

func TestResolveInterruptedBadNewRollsBack(t *testing.T) {
	dataDir, st, _, sessionCode := setupInstalled(t)
	out := filepath.Join(dataDir, "backups", "b.zip")
	if err := Create(st, out); err != nil {
		t.Fatal(err)
	}
	if err := st.SetPreference("theme", "keep-me"); err != nil {
		t.Fatal(err)
	}

	staging := DefaultStagingDir(dataDir)
	if err := ValidateArchive(out, staging); err != nil {
		t.Fatal(err)
	}
	pre := filepath.Join(dataDir, "backups", "pre.zip")
	if err := Create(st, pre); err != nil {
		t.Fatal(err)
	}
	if err := PrepareSwap(dataDir, staging, pre); err != nil {
		t.Fatal(err)
	}
	if err := ApplySwap(dataDir, st.Close); err != nil {
		t.Fatal(err)
	}

	// Corrupt the newly installed database.
	if err := os.WriteFile(filepath.Join(dataDir, "crv.sqlite"), []byte("broken"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := ResolveInterrupted(dataDir); err != nil {
		t.Fatal(err)
	}
	st2, err := store.Open(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer st2.Close()
	got, err := st2.GetSession("sess-backup-1")
	if err != nil || got == nil || got.Code != sessionCode {
		t.Fatalf("rollback lost session: %+v %v", got, err)
	}
	theme, ok, _ := st2.GetPreference("theme")
	if !ok || theme != "keep-me" {
		t.Fatalf("expected rolled-back theme keep-me, got %q", theme)
	}
}

func TestSnapshotDBAndCreateRefuseMissingRevisionDir(t *testing.T) {
	dataDir, st, revID, _ := setupInstalled(t)

	snap := filepath.Join(t.TempDir(), "snap.sqlite")
	if err := SnapshotDB(st, snap); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(snap); err != nil {
		t.Fatal(err)
	}

	_ = os.RemoveAll(store.RevisionDir(dataDir, revID))
	err := Create(st, filepath.Join(dataDir, "backups", "missing-rev.zip"))
	if err == nil {
		t.Fatal("expected missing revision failure")
	}
}

func TestCreateRejectsMissingCatalogImage(t *testing.T) {
	dataDir, st, revID, _ := setupInstalled(t)
	imgs, err := st.RevisionImages(revID)
	if err != nil || len(imgs) == 0 {
		t.Fatalf("images: %v", err)
	}
	missing := filepath.Join(store.RevisionDir(dataDir, revID), filepath.FromSlash(imgs[0].DisplayRelpath))
	if err := os.Remove(missing); err != nil {
		t.Fatal(err)
	}
	err = Create(st, filepath.Join(dataDir, "backups", "missing-img.zip"))
	if err == nil || !strings.Contains(err.Error(), "ausente") {
		t.Fatalf("want missing catalog file error, got %v", err)
	}
}

func TestValidateArchiveRejectsMissingCatalogImage(t *testing.T) {
	dataDir, st, revID, _ := setupInstalled(t)
	out := filepath.Join(dataDir, "backups", "good.zip")
	if err := Create(st, out); err != nil {
		t.Fatal(err)
	}
	imgs, err := st.RevisionImages(revID)
	if err != nil || len(imgs) == 0 {
		t.Fatalf("images: %v", err)
	}
	_ = st.Close()

	extractDir := filepath.Join(dataDir, "extract-good")
	if err := ValidateArchive(out, extractDir); err != nil {
		t.Fatal(err)
	}
	// Remove a DB-referenced display file and drop it from the manifest so only the
	// staged catalog-path check can catch the damage.
	dropRel := filepath.ToSlash(imgs[0].DisplayRelpath)
	dropPath := filepath.Join(extractDir, "catalog", fmt.Sprintf("rev-%d", revID), filepath.FromSlash(dropRel))
	if err := os.Remove(dropPath); err != nil {
		t.Fatal(err)
	}
	manPath := filepath.Join(extractDir, "manifest.json")
	man, err := readManifestFile(manPath)
	if err != nil {
		t.Fatal(err)
	}
	dropZip := path.Join(fmt.Sprintf("catalog/rev-%d", revID), dropRel)
	delete(man.Files, dropZip)
	raw, err := json.MarshalIndent(man, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manPath, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	stripped := filepath.Join(dataDir, "backups", "stripped.zip")
	if err := zipDirectory(extractDir, stripped); err != nil {
		t.Fatal(err)
	}
	err = ValidateArchive(stripped, filepath.Join(dataDir, "stg-missing-img"))
	if err == nil || !strings.Contains(err.Error(), "ausente") {
		t.Fatalf("want missing catalog file error, got %v", err)
	}
}

func TestValidateArchiveRejectsManifestOmittingRequiredRevision(t *testing.T) {
	dataDir, st, revID, _ := setupInstalled(t)
	out := filepath.Join(dataDir, "backups", "rev-omit-src.zip")
	if err := Create(st, out); err != nil {
		t.Fatal(err)
	}
	_ = st.Close()

	extractDir := filepath.Join(dataDir, "extract-rev-omit")
	if err := ValidateArchive(out, extractDir); err != nil {
		t.Fatal(err)
	}
	manPath := filepath.Join(extractDir, "manifest.json")
	man, err := readManifestFile(manPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(man.RevisionIDs) == 0 || man.RevisionIDs[0] != revID {
		t.Fatalf("expected manifest to list rev %d, got %v", revID, man.RevisionIDs)
	}
	man.RevisionIDs = nil
	raw, err := json.MarshalIndent(man, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manPath, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	stripped := filepath.Join(dataDir, "backups", "rev-omit.zip")
	if err := zipDirectory(extractDir, stripped); err != nil {
		t.Fatal(err)
	}
	err = ValidateArchive(stripped, filepath.Join(dataDir, "stg-rev-omit"))
	if err == nil || !strings.Contains(err.Error(), "manifesto") {
		t.Fatalf("want omitted-revision error, got %v", err)
	}
}

func TestApplySwapPreservesOldDirWhenMoveAndUndoFail(t *testing.T) {
	dataDir, st, _, _ := setupInstalled(t)
	out := filepath.Join(dataDir, "backups", "swap-fail.zip")
	if err := Create(st, out); err != nil {
		t.Fatal(err)
	}
	staging := DefaultStagingDir(dataDir)
	if err := ValidateArchive(out, staging); err != nil {
		t.Fatal(err)
	}
	pre := filepath.Join(dataDir, "backups", "pre-swap-fail.zip")
	if err := Create(st, pre); err != nil {
		t.Fatal(err)
	}
	if err := PrepareSwap(dataDir, staging, pre); err != nil {
		t.Fatal(err)
	}

	calls := 0
	moveLivePayloadFn = func(from, to string) error {
		calls++
		if calls == 1 {
			// Move succeeds on disk, then the operation reports failure; undo also fails.
			if err := moveLivePayload(from, to); err != nil {
				return err
			}
			return fmt.Errorf("simulated move failure")
		}
		return fmt.Errorf("simulated undo failure")
	}
	t.Cleanup(func() { moveLivePayloadFn = moveLivePayload })

	err := ApplySwap(dataDir, st.Close)
	if err == nil {
		t.Fatal("expected ApplySwap failure")
	}
	m, err := readMarker(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	if m.Stage != StageLiveMoved {
		t.Fatalf("stage=%s want live_moved", m.Stage)
	}
	if m.OldDir == "" {
		t.Fatal("OldDir must be preserved after failed undo")
	}
	if _, err := os.Stat(filepath.Join(m.OldDir, "crv.sqlite")); err != nil {
		t.Fatalf("old sqlite missing under OldDir: %v", err)
	}
	if _, err := os.Stat(filepath.Join(m.OldDir, "catalog")); err != nil {
		t.Fatalf("old catalog missing under OldDir: %v", err)
	}
	// Recovery must still be possible from the preserved marker + OldDir.
	if err := ResolveInterrupted(dataDir); err != nil {
		t.Fatalf("ResolveInterrupted: %v", err)
	}
	if _, err := os.Stat(MarkerPath(dataDir)); !os.IsNotExist(err) {
		t.Fatalf("marker should be cleared after recovery, err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(dataDir, "crv.sqlite")); err != nil {
		t.Fatalf("live sqlite not restored: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dataDir, "catalog")); err != nil {
		t.Fatalf("live catalog not restored: %v", err)
	}
}

func TestResolveInterruptedAfterPartialMoveAndFailedUndo(t *testing.T) {
	dataDir, st, _, sessionCode := setupInstalled(t)
	out := filepath.Join(dataDir, "backups", "partial-move.zip")
	if err := Create(st, out); err != nil {
		t.Fatal(err)
	}
	staging := DefaultStagingDir(dataDir)
	if err := ValidateArchive(out, staging); err != nil {
		t.Fatal(err)
	}
	if err := PrepareSwap(dataDir, staging, out); err != nil {
		t.Fatal(err)
	}

	calls := 0
	moveLivePayloadFn = func(from, to string) error {
		calls++
		if calls == 1 {
			if err := moveFile(filepath.Join(from, "crv.sqlite"), filepath.Join(to, "crv.sqlite")); err != nil {
				return err
			}
			return fmt.Errorf("simulated failure before moving catalog")
		}
		return fmt.Errorf("simulated undo failure")
	}
	t.Cleanup(func() { moveLivePayloadFn = moveLivePayload })
	if err := ApplySwap(dataDir, st.Close); err == nil {
		t.Fatal("expected partial move failure")
	}
	if _, err := os.Stat(filepath.Join(dataDir, "catalog")); err != nil {
		t.Fatalf("unmoved live catalog missing before recovery: %v", err)
	}
	if err := ResolveInterrupted(dataDir); err != nil {
		t.Fatal(err)
	}
	st2, err := store.Open(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer st2.Close()
	got, err := st2.GetSession("sess-backup-1")
	if err != nil || got == nil || got.Code != sessionCode {
		t.Fatalf("recovery lost session: %+v %v", got, err)
	}
	if _, err := os.Stat(filepath.Join(dataDir, "catalog")); err != nil {
		t.Fatalf("recovery lost unmoved catalog: %v", err)
	}
	if _, err := os.Stat(MarkerPath(dataDir)); !os.IsNotExist(err) {
		t.Fatalf("marker should be cleared after recovery, err=%v", err)
	}
}

func TestResolveInterruptedAfterLiveMovedMarkerWithPayloadAside(t *testing.T) {
	// Simulates crash after live_moved was persisted and live payload moved aside,
	// before staged data was installed — must roll back to OldDir.
	dataDir, st, _, sessionCode := setupInstalled(t)
	out := filepath.Join(dataDir, "backups", "b.zip")
	if err := Create(st, out); err != nil {
		t.Fatal(err)
	}
	if err := st.SetPreference("theme", "keep-aside"); err != nil {
		t.Fatal(err)
	}
	staging := DefaultStagingDir(dataDir)
	if err := ValidateArchive(out, staging); err != nil {
		t.Fatal(err)
	}
	pre := filepath.Join(dataDir, "backups", "pre.zip")
	if err := Create(st, pre); err != nil {
		t.Fatal(err)
	}
	if err := PrepareSwap(dataDir, staging, pre); err != nil {
		t.Fatal(err)
	}
	_ = st.Close()

	oldDir := filepath.Join(dataDir, "restore-old-test")
	if err := os.MkdirAll(oldDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := moveLivePayload(dataDir, oldDir); err != nil {
		t.Fatal(err)
	}
	if err := writeMarker(dataDir, Marker{
		Stage:     StageLiveMoved,
		PreBackup: pre,
		Staging:   staging,
		OldDir:    oldDir,
	}); err != nil {
		t.Fatal(err)
	}

	if err := ResolveInterrupted(dataDir); err != nil {
		t.Fatal(err)
	}
	st2, err := store.Open(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer st2.Close()
	got, err := st2.GetSession("sess-backup-1")
	if err != nil || got == nil || got.Code != sessionCode {
		t.Fatalf("rollback lost session: %+v %v", got, err)
	}
	theme, ok, _ := st2.GetPreference("theme")
	if !ok || theme != "keep-aside" {
		t.Fatalf("expected rolled-back theme keep-aside, got %q", theme)
	}
}

func zipDirectory(root, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	zw := zip.NewWriter(out)
	err = filepath.Walk(root, func(p string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		name := filepath.ToSlash(rel)
		w, err := zw.Create(name)
		if err != nil {
			return err
		}
		f, err := os.Open(p)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(w, f)
		_ = f.Close()
		return copyErr
	})
	if err != nil {
		_ = zw.Close()
		return err
	}
	return zw.Close()
}

func writeRawZip(path string, files map[string][]byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	zw := zip.NewWriter(f)
	for name, body := range files {
		w, err := zw.Create(name)
		if err != nil {
			_ = zw.Close()
			return err
		}
		if _, err := w.Write(body); err != nil {
			_ = zw.Close()
			return err
		}
	}
	return zw.Close()
}

func tamperZipFile(src, dst, entry string, newBody []byte) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()
	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)
	for _, f := range r.File {
		var body []byte
		if f.Name == entry {
			body = newBody
		} else {
			rc, err := f.Open()
			if err != nil {
				_ = zw.Close()
				return err
			}
			var b bytes.Buffer
			_, err = b.ReadFrom(rc)
			_ = rc.Close()
			if err != nil {
				_ = zw.Close()
				return err
			}
			body = b.Bytes()
		}
		w, err := zw.Create(f.Name)
		if err != nil {
			_ = zw.Close()
			return err
		}
		if _, err := w.Write(body); err != nil {
			_ = zw.Close()
			return err
		}
	}
	if err := zw.Close(); err != nil {
		return err
	}
	return os.WriteFile(dst, buf.Bytes(), 0o644)
}

func TestSha256File(t *testing.T) {
	p := filepath.Join(t.TempDir(), "f.bin")
	body := []byte("abc")
	if err := os.WriteFile(p, body, 0o644); err != nil {
		t.Fatal(err)
	}
	sum, err := sha256File(p)
	if err != nil {
		t.Fatal(err)
	}
	want := sha256.Sum256(body)
	if sum != hex.EncodeToString(want[:]) {
		t.Fatalf("got %s", sum)
	}
}
