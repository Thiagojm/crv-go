package catalog

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/Thiagojm/crv-go/internal/store"
)

func TestImportFolderCreatesSecondRevision(t *testing.T) {
	root := t.TempDir()
	dataDir := filepath.Join(root, "data")
	st, err := store.Open(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	first := filepath.Join(root, "bank1")
	if _, err := WriteSynthetic(first, 4); err != nil {
		t.Fatal(err)
	}
	res1, err := Install(st, first)
	if err != nil {
		t.Fatal(err)
	}
	oldDir := store.RevisionDir(dataDir, res1.RevisionID)
	if _, err := os.Stat(oldDir); err != nil {
		t.Fatal(err)
	}

	second := filepath.Join(root, "bank2")
	if _, err := WriteSynthetic(second, 5); err != nil {
		t.Fatal(err)
	}
	beforeSrc, err := os.ReadFile(filepath.Join(second, "catalog-unified.json"))
	if err != nil {
		t.Fatal(err)
	}
	res2, err := ImportFolder(st, second, "import-folder")
	if err != nil {
		t.Fatal(err)
	}
	if res2.RevisionID == res1.RevisionID {
		t.Fatalf("expected new revision, got %d", res2.RevisionID)
	}
	if !res2.Report.Ready || res2.Report.SourceLabel != "import-folder" {
		t.Fatalf("report: %+v", res2.Report)
	}
	active, err := st.ActiveCatalog()
	if err != nil || active == nil || active.RevisionID != res2.RevisionID {
		t.Fatalf("active=%+v err=%v", active, err)
	}
	if _, err := os.Stat(oldDir); err != nil {
		t.Fatalf("old revision files must remain: %v", err)
	}
	afterSrc, err := os.ReadFile(filepath.Join(second, "catalog-unified.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(beforeSrc, afterSrc) {
		t.Fatal("source folder was modified")
	}
}

func TestImportZipSuccessAndCorruptRejected(t *testing.T) {
	root := t.TempDir()
	dataDir := filepath.Join(root, "data")
	st, err := store.Open(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	base := filepath.Join(root, "base")
	if _, err := WriteSynthetic(base, 4); err != nil {
		t.Fatal(err)
	}
	res1, err := Install(st, base)
	if err != nil {
		t.Fatal(err)
	}

	pkg := filepath.Join(root, "pkg")
	if _, err := WriteSynthetic(pkg, 4); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := zipDir(&buf, pkg); err != nil {
		t.Fatal(err)
	}
	res2, err := ImportZip(st, bytes.NewReader(buf.Bytes()), "import-zip")
	if err != nil {
		t.Fatal(err)
	}
	if res2.RevisionID == res1.RevisionID || !res2.Report.Ready {
		t.Fatalf("zip import failed: %+v", res2)
	}

	_, err = ImportZip(st, bytes.NewReader([]byte("not-a-zip")), "import-zip")
	if err == nil {
		t.Fatal("expected corrupt zip rejection")
	}
	active, err := st.ActiveCatalog()
	if err != nil || active == nil || active.RevisionID != res2.RevisionID {
		t.Fatalf("active changed after corrupt zip: %+v", active)
	}
}

func TestExtractZipRejectsTraversal(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("../escape.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("nope")); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	dest := t.TempDir()
	if err := ExtractZip(bytes.NewReader(buf.Bytes()), dest); err == nil {
		t.Fatal("expected traversal rejection")
	}
	if _, err := os.Stat(filepath.Join(dest, "escape.txt")); err == nil {
		t.Fatal("escaped file written")
	}
}

func TestExtractZipRejectsAbsolutePath(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	hdr := &zip.FileHeader{Name: "C:/Windows/escape.txt", Method: zip.Deflate}
	w, err := zw.CreateHeader(hdr)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("nope")); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := ExtractZip(bytes.NewReader(buf.Bytes()), t.TempDir()); err == nil {
		t.Fatal("expected absolute path rejection")
	}
}

func TestImportFolderRejectsSymlinkRoot(t *testing.T) {
	root := t.TempDir()
	real := filepath.Join(root, "real")
	if _, err := WriteSynthetic(real, 4); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "link")
	if err := os.Symlink(real, link); err != nil {
		t.Skipf("symlink not available: %v", err)
	}
	st, err := store.Open(filepath.Join(root, "data"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if _, err := ImportFolder(st, link, "import-folder"); err == nil {
		t.Fatal("expected symlink folder rejection")
	}
}

func zipDir(buf *bytes.Buffer, dir string) error {
	zw := zip.NewWriter(buf)
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		name := filepath.ToSlash(rel)
		if info.IsDir() {
			_, err := zw.Create(name + "/")
			return err
		}
		w, err := zw.Create(name)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		_, err = w.Write(data)
		return err
	})
	if err != nil {
		_ = zw.Close()
		return err
	}
	return zw.Close()
}

func TestSafeZipEntry(t *testing.T) {
	for _, name := range []string{"../x", "/abs", "C:/x", "..\\y"} {
		if _, _, err := safeZipEntry(name); err == nil {
			t.Fatalf("expected reject %q", name)
		}
	}
	rel, _, err := safeZipEntry("images/a.jpg")
	if err != nil || rel != "images/a.jpg" {
		t.Fatalf("got %q %v", rel, err)
	}
}
