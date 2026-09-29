package catalog

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Thiagojm/crv-go/internal/store"
)

func writeJPEG(path string, w, h int, salt byte) (string, error) {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x + int(salt)), G: uint8(y), B: salt, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90}); err != nil {
		return "", err
	}
	sum := sha256.Sum256(buf.Bytes())
	hexSum := hex.EncodeToString(sum[:])
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		return "", err
	}
	return hexSum, nil
}

func writeGIF(path string, w, h int) (string, error) {
	img := image.NewPaletted(image.Rect(0, 0, w, h), color.Palette{color.White, color.Black})
	var buf bytes.Buffer
	if err := gif.Encode(&buf, img, nil); err != nil {
		return "", err
	}
	sum := sha256.Sum256(buf.Bytes())
	hexSum := hex.EncodeToString(sum[:])
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		return "", err
	}
	return hexSum, nil
}

func TestInstallTinyCatalog(t *testing.T) {
	root := t.TempDir()
	cat := filepath.Join(root, "farsight")
	imgDir := filepath.Join(cat, "images")
	if err := os.MkdirAll(imgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	dataDir := filepath.Join(root, "data")
	st, err := store.Open(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	type item struct {
		name string
		hash string
	}
	var items []item
	for i := 0; i < 4; i++ {
		name := filepath.Join("images", "t"+string(rune('a'+i))+".jpg")
		hash, err := writeJPEG(filepath.Join(cat, name), 8, 8, byte(40+i))
		if err != nil {
			t.Fatal(err)
		}
		items = append(items, item{name: filepath.ToSlash(name), hash: hash})
	}
	gifName := filepath.ToSlash(filepath.Join("images", "anim.gif"))
	gifHash, err := writeGIF(filepath.Join(cat, gifName), 4, 4)
	if err != nil {
		t.Fatal(err)
	}

	inv := Inventory{Format: "crv-local-image-inventory-v1"}
	for i, it := range items {
		inv.Images = append(inv.Images, InventoryImage{
			SHA256: it.hash,
			Paths:  []string{it.name},
			Sources: []SourceRecord{{
				ID: "A-T" + string(rune('1'+i)), Pool: "A",
				Images: []SourceImage{{LocalPath: it.name}},
			}},
		})
	}
	inv.Images = append(inv.Images, InventoryImage{
		SHA256: gifHash,
		Paths:  []string{gifName},
		Sources: []SourceRecord{{
			ID: "B-G1", Pool: "B",
			Images: []SourceImage{{LocalPath: gifName}},
		}},
	})
	// multi-image source only
	inv.Images = append(inv.Images, InventoryImage{
		SHA256: strings.Repeat("ab", 32),
		Paths:  []string{"images/multi.jpg"},
		Sources: []SourceRecord{{
			ID: "C-M1", Pool: "C",
			Images: []SourceImage{{LocalPath: "images/a.jpg"}, {LocalPath: "images/b.jpg"}},
		}},
	})
	// traversal
	badHash, err := writeJPEG(filepath.Join(imgDir, "ok.jpg"), 4, 4, 99)
	if err != nil {
		t.Fatal(err)
	}
	inv.Images = append(inv.Images, InventoryImage{
		SHA256: badHash,
		Paths:  []string{"../secrets.jpg"},
		Sources: []SourceRecord{{
			ID: "A-BAD", Pool: "A",
			Images: []SourceImage{{LocalPath: "../secrets.jpg"}},
		}},
	})

	raw, _ := json.Marshal(inv)
	if err := os.WriteFile(filepath.Join(cat, "catalog-unified.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := Install(st, cat)
	if err != nil {
		t.Fatalf("%v report=%+v", err, res.Report)
	}
	if !res.Report.Ready || res.Report.EligibleCount < 4 {
		t.Fatalf("expected ready catalog, got %+v", res.Report)
	}
	if res.Report.ExcludedCount < 1 {
		t.Fatalf("expected exclusions, got %d", res.Report.ExcludedCount)
	}

	// restart must not duplicate revision / mutate source bank
	before, err := os.ReadFile(filepath.Join(cat, "catalog-unified.json"))
	if err != nil {
		t.Fatal(err)
	}
	res2, err := EnsureInstalled(st, cat)
	if err != nil {
		t.Fatal(err)
	}
	if res2.RevisionID != res.RevisionID {
		t.Fatalf("revision changed on reinstall: %d -> %d", res.RevisionID, res2.RevisionID)
	}
	after, err := os.ReadFile(filepath.Join(cat, "catalog-unified.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("source catalog was modified")
	}

	disp := filepath.Join(dataDir, "catalog", fmt.Sprintf("rev-%d", res.RevisionID), "display", items[0].hash+".png")
	if _, err := os.Stat(disp); err != nil {
		t.Fatalf("display png missing: %v", err)
	}
}

func TestSafePathRejects(t *testing.T) {
	for _, p := range []string{"/abs.jpg", "C:/x.jpg", "../x.jpg", "a/../../b.jpg"} {
		if err := assertSafeRelPath(p); err == nil {
			t.Fatalf("expected reject for %q", p)
		}
	}
}

func TestRejectBadFormatAndDuplicateHash(t *testing.T) {
	root := t.TempDir()
	cat := filepath.Join(root, "farsight")
	if err := os.MkdirAll(filepath.Join(cat, "images"), 0o755); err != nil {
		t.Fatal(err)
	}
	st, err := store.Open(filepath.Join(root, "data"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	hash, err := writeJPEG(filepath.Join(cat, "images", "a.jpg"), 4, 4, 1)
	if err != nil {
		t.Fatal(err)
	}
	bad := Inventory{Format: "other", Images: []InventoryImage{{
		SHA256: hash, Paths: []string{"images/a.jpg"},
		Sources: []SourceRecord{{ID: "A-1", Pool: "A", Images: []SourceImage{{LocalPath: "images/a.jpg", SourceURL: "http://example/a.jpg"}}}},
	}}}
	raw, _ := json.Marshal(bad)
	_ = os.WriteFile(filepath.Join(cat, "catalog-unified.json"), raw, 0o644)
	if _, err := Install(st, cat); err == nil {
		t.Fatal("expected format rejection")
	}

	dup := Inventory{Format: expectedFormat, Images: []InventoryImage{
		{SHA256: hash, Paths: []string{"images/a.jpg"}, Sources: []SourceRecord{{ID: "A-1", Pool: "A", Images: []SourceImage{{LocalPath: "images/a.jpg", SourceURL: "http://example/a.jpg"}}}}},
		{SHA256: hash, Paths: []string{"images/a.jpg"}, Sources: []SourceRecord{{ID: "A-2", Pool: "A", Images: []SourceImage{{LocalPath: "images/a.jpg", SourceURL: "http://example/a.jpg"}}}}},
	}}
	raw, _ = json.Marshal(dup)
	_ = os.WriteFile(filepath.Join(cat, "catalog-unified.json"), raw, 0o644)
	if _, err := Install(st, cat); err == nil {
		t.Fatal("expected duplicate identity rejection")
	}
}

func TestEnsureInstalledWithoutDistributionDir(t *testing.T) {
	root := t.TempDir()
	cat := filepath.Join(root, "farsight")
	_ = os.MkdirAll(filepath.Join(cat, "images"), 0o755)
	dataDir := filepath.Join(root, "data")
	st, err := store.Open(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	var images []InventoryImage
	for i := 0; i < 4; i++ {
		name := filepath.ToSlash(filepath.Join("images", fmt.Sprintf("x%d.jpg", i)))
		hash, err := writeJPEG(filepath.Join(cat, filepath.FromSlash(name)), 4, 4, byte(i+1))
		if err != nil {
			t.Fatal(err)
		}
		images = append(images, InventoryImage{
			SHA256: hash, Paths: []string{name},
			Sources: []SourceRecord{{ID: fmt.Sprintf("A-%d", i), Pool: "A", Images: []SourceImage{{LocalPath: name, SourceURL: "http://example/" + name}}}},
		})
	}
	inv := Inventory{
		Format: expectedFormat,
		Summary: &InventorySummary{EntriesWithMultipleImages: 2, EntriesWithoutImages: 1, SingleImageCandidates: 4},
		Provenance: json.RawMessage(`[{"pool":"A"}]`),
		ReviewEntries: json.RawMessage(`[{"id":"M1","images":[{},{}]},{"id":"M2","images":[{},{}]},{"id":"E1","images":[]}]`),
		Images: images,
	}
	raw, _ := json.Marshal(inv)
	_ = os.WriteFile(filepath.Join(cat, "catalog-unified.json"), raw, 0o644)
	res, err := Install(st, cat)
	if err != nil {
		t.Fatal(err)
	}
	if res.Report.MultiImageSources != 2 || res.Report.EmptySources != 1 {
		t.Fatalf("summary counts multi=%d empty=%d", res.Report.MultiImageSources, res.Report.EmptySources)
	}
	if len(res.Report.Provenance) == 0 || !strings.Contains(string(res.Report.Provenance), `"pool":"A"`) {
		t.Fatal("provenance missing")
	}
	var stored []SourceRecord
	imgs, err := st.RevisionImages(res.RevisionID)
	if err != nil || len(imgs) == 0 {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(imgs[0].SourcesJSON), &stored); err != nil {
		t.Fatal(err)
	}
	if stored[0].Images[0].SourceURL == "" {
		t.Fatal("source_url not preserved")
	}

	_ = os.RemoveAll(cat)
	res2, err := EnsureInstalled(st, cat)
	if err != nil {
		t.Fatalf("should reuse installed revision without farsight: %v", err)
	}
	if res2.RevisionID != res.RevisionID || !res2.Report.Ready {
		t.Fatalf("reuse failed: %+v", res2)
	}

	// Simulate activate-without-files: remove revision dir, EnsureInstalled must not report ready.
	_ = os.RemoveAll(store.RevisionDir(dataDir, res.RevisionID))
	res3, err := EnsureInstalled(st, filepath.Join(root, "missing-farsight"))
	if err == nil && res3.Report.Ready {
		t.Fatal("broken revision must not be ready")
	}
}

func TestRepairRestoresSameRevision(t *testing.T) {
	root := t.TempDir()
	cat := filepath.Join(root, "farsight")
	if _, err := WriteSynthetic(cat, 4); err != nil {
		t.Fatal(err)
	}
	dataDir := filepath.Join(root, "data")
	st, err := store.Open(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	res, err := Install(st, cat)
	if err != nil {
		t.Fatal(err)
	}
	revDir := store.RevisionDir(dataDir, res.RevisionID)
	imgs, err := st.RevisionImages(res.RevisionID)
	if err != nil || len(imgs) == 0 {
		t.Fatal(err)
	}
	missing := filepath.Join(revDir, filepath.FromSlash(imgs[0].DisplayRelpath))
	if err := os.Remove(missing); err != nil {
		t.Fatal(err)
	}
	if err := verifyRevisionFiles(st, res.RevisionID); err == nil {
		t.Fatal("expected broken revision before repair")
	}

	repaired, err := Repair(st, cat)
	if err != nil {
		t.Fatalf("repair: %v", err)
	}
	if repaired.RevisionID != res.RevisionID {
		t.Fatalf("repair changed revision id %d -> %d", res.RevisionID, repaired.RevisionID)
	}
	if !repaired.Report.Ready {
		t.Fatalf("repair not ready: %+v", repaired.Report)
	}
	if _, err := os.Stat(missing); err != nil {
		t.Fatalf("display file not restored: %v", err)
	}
	active, err := st.ActiveCatalog()
	if err != nil || active == nil || active.RevisionID != res.RevisionID {
		t.Fatalf("active revision changed: %+v", active)
	}
}

func TestRepairRejectsMismatchedSource(t *testing.T) {
	root := t.TempDir()
	cat := filepath.Join(root, "farsight")
	other := filepath.Join(root, "other")
	if _, err := WriteSynthetic(cat, 4); err != nil {
		t.Fatal(err)
	}
	// Distinct salts so SHA-256 identities diverge from WriteSynthetic defaults.
	_ = os.MkdirAll(filepath.Join(other, "images"), 0o755)
	var images []InventoryImage
	for i := 0; i < 4; i++ {
		rel := fmt.Sprintf("images/o%d.jpg", i)
		hash, err := writeJPEG(filepath.Join(other, rel), 8, 8, byte(200+i))
		if err != nil {
			t.Fatal(err)
		}
		images = append(images, InventoryImage{
			SHA256: hash, Paths: []string{rel},
			Sources: []SourceRecord{{ID: fmt.Sprintf("B-%d", i), Pool: "B", Images: []SourceImage{{LocalPath: rel}}}},
		})
	}
	raw, _ := json.Marshal(Inventory{Format: expectedFormat, Images: images, Summary: &InventorySummary{SingleImageCandidates: 4}})
	_ = os.WriteFile(filepath.Join(other, "catalog-unified.json"), raw, 0o644)

	dataDir := filepath.Join(root, "data")
	st, err := store.Open(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	res, err := Install(st, cat)
	if err != nil {
		t.Fatal(err)
	}
	_ = os.RemoveAll(store.RevisionDir(dataDir, res.RevisionID))

	_, err = Repair(st, other)
	if err == nil {
		t.Fatal("expected mismatched source to fail")
	}
	active, err := st.ActiveCatalog()
	if err != nil || active == nil || active.RevisionID != res.RevisionID {
		t.Fatalf("repair must keep active revision: %+v err=%v", active, err)
	}
}

func TestEnsureInstalledRepairsInPlace(t *testing.T) {
	root := t.TempDir()
	cat := filepath.Join(root, "farsight")
	if _, err := WriteSynthetic(cat, 4); err != nil {
		t.Fatal(err)
	}
	dataDir := filepath.Join(root, "data")
	st, err := store.Open(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	res, err := Install(st, cat)
	if err != nil {
		t.Fatal(err)
	}
	_ = os.RemoveAll(store.RevisionDir(dataDir, res.RevisionID))

	again, err := EnsureInstalled(st, cat)
	if err != nil {
		t.Fatal(err)
	}
	if again.RevisionID != res.RevisionID {
		t.Fatalf("EnsureInstalled should repair in place, got %d want %d", again.RevisionID, res.RevisionID)
	}
	if !again.Report.Ready {
		t.Fatal("expected ready after in-place repair")
	}
}

func TestAncestorSymlinkRejected(t *testing.T) {
	root := t.TempDir()
	cat := filepath.Join(root, "farsight")
	realImages := filepath.Join(root, "real-images")
	_ = os.MkdirAll(realImages, 0o755)
	_ = os.MkdirAll(cat, 0o755)
	hash, err := writeJPEG(filepath.Join(realImages, "a.jpg"), 4, 4, 7)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(realImages, filepath.Join(cat, "images")); err != nil {
		t.Skipf("symlink not available: %v", err)
	}
	stage := t.TempDir()
	_ = os.MkdirAll(filepath.Join(stage, "originals"), 0o755)
	_ = os.MkdirAll(filepath.Join(stage, "display"), 0o755)
	_, ex := validateImage(cat, stage, InventoryImage{
		SHA256: hash, Paths: []string{"images/a.jpg"},
		Sources: []SourceRecord{{ID: "A-1", Pool: "A", Images: []SourceImage{{LocalPath: "images/a.jpg"}}}},
	})
	if ex == nil || !strings.Contains(ex.Reason, "ancestral") {
		t.Fatalf("expected ancestor symlink exclusion, got %#v", ex)
	}
}
