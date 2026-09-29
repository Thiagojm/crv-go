package app

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/Thiagojm/crv-go/internal/catalog"
	"github.com/Thiagojm/crv-go/internal/store"
)

func TestCatalogImportFolderAndRejectActiveSession(t *testing.T) {
	s := catalogServer(t)
	csrf, cookie := authClient(t, s)

	root := t.TempDir()
	pkg := filepath.Join(root, "replacement")
	if _, err := catalog.WriteSynthetic(pkg, 5); err != nil {
		t.Fatal(err)
	}

	activeBefore, err := s.Store.ActiveCatalog()
	if err != nil || activeBefore == nil {
		t.Fatal(err)
	}
	oldDir := store.RevisionDir(s.Store.DataDir, activeBefore.RevisionID)

	body, _ := json.Marshal(map[string]string{"path": pkg})
	rec := apiJSON(t, s, http.MethodPost, "/api/catalog/import/folder", csrf, cookie, "", string(body))
	if rec.Code != 200 {
		t.Fatalf("import folder %d %s", rec.Code, rec.Body.String())
	}
	var out struct {
		RevisionID int64          `json:"revisionId"`
		Ready      bool           `json:"ready"`
		Report     catalog.Report `json:"report"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if !out.Ready || out.RevisionID == activeBefore.RevisionID {
		t.Fatalf("unexpected import result: %+v", out)
	}
	if _, err := os.Stat(oldDir); err != nil {
		t.Fatalf("old revision must remain: %v", err)
	}
	if s.Report.SourceLabel == "" || !s.Ready {
		t.Fatalf("server report not updated: %+v ready=%v", s.Report, s.Ready)
	}

	createSession(t, s, csrf, cookie, "op-block-replace")
	rec = apiJSON(t, s, http.MethodPost, "/api/catalog/import/folder", csrf, cookie, "", string(body))
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409 with active session, got %d %s", rec.Code, rec.Body.String())
	}
	if !stringsContains(rec.Body.String(), "substitução indisponível") {
		t.Fatalf("message: %s", rec.Body.String())
	}
}

func TestCatalogImportZipRawAndRepair(t *testing.T) {
	root := t.TempDir()
	cat := filepath.Join(root, "farsight")
	if _, err := catalog.WriteSynthetic(cat, 4); err != nil {
		t.Fatal(err)
	}
	st, err := store.Open(filepath.Join(root, "data"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	res, err := catalog.Install(st, cat)
	if err != nil {
		t.Fatal(err)
	}
	sec, err := NewSecurity()
	if err != nil {
		t.Fatal(err)
	}
	s := &Server{
		Store: st, Security: sec, Port: 34567, Ready: true, Report: res.Report, CatalogDir: cat,
	}
	csrf, cookie := authClient(t, s)

	pkg := filepath.Join(root, "pkg")
	if _, err := catalog.WriteSynthetic(pkg, 4); err != nil {
		t.Fatal(err)
	}
	var zipBuf bytes.Buffer
	if err := writeTestZip(&zipBuf, pkg); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/catalog/import/zip", bytes.NewReader(zipBuf.Bytes()))
	req.Host = "127.0.0.1:34567"
	req.Header.Set("Content-Type", "application/zip")
	req.Header.Set("X-CSRF-Token", csrf)
	req.Header.Set("Origin", "http://127.0.0.1:34567")
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("zip import %d %s", rec.Code, rec.Body.String())
	}

	rec = apiJSON(t, s, http.MethodPost, "/api/catalog/repair", csrf, cookie, "", "")
	if rec.Code != 200 {
		t.Fatalf("repair %d %s", rec.Code, rec.Body.String())
	}

	// Historical revision loses files; repair from the matching bundled source restores them
	// without changing the active imported revision id.
	oldID := res.RevisionID
	_ = os.RemoveAll(store.RevisionDir(st.DataDir, oldID))
	s.CatalogDir = cat
	rec = apiJSON(t, s, http.MethodPost, "/api/catalog/repair", csrf, cookie, "", "")
	if rec.Code != 200 {
		t.Fatalf("repair historical %d %s", rec.Code, rec.Body.String())
	}
	if _, err := os.Stat(filepath.Join(store.RevisionDir(st.DataDir, oldID), "display")); err != nil {
		t.Fatalf("historical revision files not restored: %v", err)
	}
	active, err := st.ActiveCatalog()
	if err != nil || active == nil || active.RevisionID == oldID {
		t.Fatalf("active must remain imported revision, got %+v", active)
	}
}

func TestCatalogImportZipMultipart(t *testing.T) {
	s := catalogServer(t)
	csrf, cookie := authClient(t, s)

	pkg := filepath.Join(t.TempDir(), "pkg")
	if _, err := catalog.WriteSynthetic(pkg, 4); err != nil {
		t.Fatal(err)
	}
	var zipBuf bytes.Buffer
	if err := writeTestZip(&zipBuf, pkg); err != nil {
		t.Fatal(err)
	}
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, err := mw.CreateFormFile("archive", "bank.zip")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fw.Write(zipBuf.Bytes()); err != nil {
		t.Fatal(err)
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/catalog/import/zip", &body)
	req.Host = "127.0.0.1:34567"
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("X-CSRF-Token", csrf)
	req.Header.Set("Origin", "http://127.0.0.1:34567")
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("multipart zip %d %s", rec.Code, rec.Body.String())
	}
}

func TestCatalogCorruptZipPreservesActive(t *testing.T) {
	s := catalogServer(t)
	csrf, cookie := authClient(t, s)
	before, err := s.Store.ActiveCatalog()
	if err != nil || before == nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/catalog/import/zip", bytes.NewReader([]byte("broken")))
	req.Host = "127.0.0.1:34567"
	req.Header.Set("Content-Type", "application/zip")
	req.Header.Set("X-CSRF-Token", csrf)
	req.Header.Set("Origin", "http://127.0.0.1:34567")
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code == 200 {
		t.Fatal("expected corrupt zip failure")
	}
	after, err := s.Store.ActiveCatalog()
	if err != nil || after == nil || after.RevisionID != before.RevisionID {
		t.Fatalf("active changed: before=%+v after=%+v", before, after)
	}
}

func writeTestZip(buf *bytes.Buffer, dir string) error {
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

func stringsContains(s, sub string) bool {
	return bytes.Contains([]byte(s), []byte(sub))
}
