package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Thiagojm/crv-go/internal/catalog"
	"github.com/Thiagojm/crv-go/internal/store"
)

func testServer(t *testing.T) *Server {
	t.Helper()
	dir := t.TempDir()
	st, err := store.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	sec, err := NewSecurity()
	if err != nil {
		t.Fatal(err)
	}
	return &Server{
		Store:     st,
		Security:  sec,
		Port:      34567,
		Ready:     true,
		Report:    catalog.Report{Ready: true, EligibleCount: 10, ExcludedCount: 1, SourceLabel: "test"},
	}
}

func authClient(t *testing.T, s *Server) (csrf string, cookie *http.Cookie) {
	t.Helper()
	token := s.Security.BootstrapToken()
	req := httptest.NewRequest(http.MethodGet, "/api/bootstrap?token="+token, nil)
	req.Host = "127.0.0.1:34567"
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("bootstrap %d %s", rec.Code, rec.Body.String())
	}
	var body map[string]string
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	cookies := rec.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("missing session cookie")
	}
	return body["csrf"], cookies[0]
}

func TestRejectBadHostAndOrigin(t *testing.T) {
	s := testServer(t)
	h := s.Handler()

	req := httptest.NewRequest(http.MethodGet, "/api/ready", nil)
	req.Host = "evil.example"
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("host: %d", rec.Code)
	}

	csrf, cookie := authClient(t, s)
	req = httptest.NewRequest(http.MethodPost, "/api/shutdown", nil)
	req.Host = "127.0.0.1:34567"
	req.Header.Set("Origin", "http://evil.example")
	req.Header.Set("X-CSRF-Token", csrf)
	req.AddCookie(cookie)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("origin: %d", rec.Code)
	}
}

func TestAuthCSRFAndNoStaticBank(t *testing.T) {
	s := testServer(t)
	h := s.Handler()

	req := httptest.NewRequest(http.MethodGet, "/api/ready", nil)
	req.Host = "127.0.0.1:34567"
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unauth ready: %d", rec.Code)
	}

	csrf, cookie := authClient(t, s)
	req = httptest.NewRequest(http.MethodGet, "/api/ready", nil)
	req.Host = "127.0.0.1:34567"
	req.AddCookie(cookie)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("ready: %d", rec.Code)
	}
	var ready map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &ready)
	if ready["phase"] != "2" {
		t.Fatalf("phase: %v", ready["phase"])
	}

	req = httptest.NewRequest(http.MethodPost, "/api/shutdown", nil)
	req.Host = "127.0.0.1:34567"
	req.Header.Set("Origin", "http://127.0.0.1:34567")
	req.AddCookie(cookie)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("missing csrf: %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/shutdown", nil)
	req.Host = "127.0.0.1:34567"
	req.Header.Set("Origin", "http://127.0.0.1:34567")
	req.Header.Set("X-CSRF-Token", csrf)
	req.AddCookie(cookie)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("shutdown: %d %s", rec.Code, rec.Body.String())
	}

	for _, path := range []string{"/farsight/images/x.jpg", "/images/secret.jpg", "/api/images/guess"} {
		req = httptest.NewRequest(http.MethodGet, path, nil)
		req.Host = "127.0.0.1:34567"
		req.AddCookie(cookie)
		rec = httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("%s expected 404 got %d", path, rec.Code)
		}
	}
}

func TestInstanceLockExclusive(t *testing.T) {
	if runtime.GOOS != "windows" && runtime.GOOS != "linux" {
		t.Skip("lock only on windows/linux")
	}
	dir := t.TempDir()
	a, err := AcquireInstance(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Release()
	if _, err := AcquireInstance(dir); err == nil {
		t.Fatal("second lock should fail")
	}
	_ = a.Release()
	b, err := AcquireInstance(dir)
	if err != nil {
		t.Fatal(err)
	}
	_ = b.Release()
}

func TestBundledCatalogEligibleCount(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	cat := filepath.Join(root, "farsight")
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	res, err := catalog.Install(st, cat)
	if err != nil {
		t.Fatalf("bundled install: %v report=%+v", err, res)
	}
	t.Logf("bundled eligible=%d excluded=%d multi=%d empty=%d",
		res.Report.EligibleCount, res.Report.ExcludedCount, res.Report.MultiImageSources, res.Report.EmptySources)
	if res.Report.EligibleCount < 4 {
		t.Fatalf("eligible too low: %d", res.Report.EligibleCount)
	}
	if res.Report.MultiImageSources != 2 || res.Report.EmptySources != 1 {
		t.Fatalf("inventory summary counts multi=%d empty=%d want 2/1", res.Report.MultiImageSources, res.Report.EmptySources)
	}
	if len(res.Report.Provenance) == 0 || len(res.Report.ReviewEntries) == 0 {
		t.Fatal("expected preserved provenance and reviewEntries")
	}
}
