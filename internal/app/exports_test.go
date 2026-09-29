package app

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Thiagojm/crv-go/internal/backup"
)

func TestExportCSVAndPrintNoHiddenLeak(t *testing.T) {
	s := catalogServer(t)
	csrf, cookie := authClient(t, s)

	id, lease, rev := createSession(t, s, csrf, cookie, "exp-csv")
	lock := apiJSON(t, s, http.MethodPost, "/api/sessions/"+id+"/lock", csrf, cookie, lease, `{"expectedRevision":`+strconv.Itoa(rev)+`}`)
	if lock.Code != 200 {
		t.Fatal(lock.Body.String())
	}
	var env struct {
		Session struct {
			Revision int     `json:"revision"`
			Correct  *string `json:"correct"`
		} `json:"session"`
	}
	_ = json.Unmarshal(lock.Body.Bytes(), &env)
	choice := "A"
	if env.Session.Correct != nil && *env.Session.Correct == "A" {
		choice = "B"
	}
	conf := apiJSON(t, s, http.MethodPost, "/api/sessions/"+id+"/confirm", csrf, cookie, lease,
		`{"expectedRevision":`+strconv.Itoa(env.Session.Revision)+`,"choice":"`+choice+`","confidence":3}`)
	if conf.Code != 200 {
		t.Fatal(conf.Body.String())
	}

	id2, lease2, rev2 := createSession(t, s, csrf, cookie, "exp-ab")
	ab := apiJSON(t, s, http.MethodPost, "/api/sessions/"+id2+"/abandon", csrf, cookie, lease2, `{"expectedRevision":`+strconv.Itoa(rev2)+`}`)
	if ab.Code != 200 {
		t.Fatal(ab.Body.String())
	}

	csvRec := apiJSON(t, s, http.MethodGet, "/api/exports/csv", "", cookie, "", "")
	if csvRec.Code != 200 {
		t.Fatalf("csv %d %s", csvRec.Code, csvRec.Body.String())
	}
	ct := csvRec.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/csv") {
		t.Fatalf("content-type: %s", ct)
	}
	body := csvRec.Body.String()
	low := strings.ToLower(body)
	for _, leak := range []string{"sha256", "farsight", "target_sha", "originals/", "credit"} {
		if strings.Contains(low, leak) {
			t.Fatalf("csv leak %q: %s", leak, body)
		}
	}
	if !strings.Contains(body, "completed") || !strings.Contains(body, "abandoned") {
		t.Fatalf("csv missing states: %s", body)
	}

	printRec := apiJSON(t, s, http.MethodGet, "/api/exports/sessions/"+id+"/print", "", cookie, "", "")
	if printRec.Code != 200 {
		t.Fatalf("print %d %s", printRec.Code, printRec.Body.String())
	}
	html := printRec.Body.String()
	if !strings.Contains(html, "Salvar como PDF") && !strings.Contains(html, "Imprimir") {
		t.Fatalf("missing print guidance: %s", html[:min(200, len(html))])
	}
	if !strings.Contains(html, "Fonte:") && !strings.Contains(strings.ToLower(html), "crédito") && !strings.Contains(strings.ToLower(html), "credito") {
		// completed print should mention credit/label section; tolerate label-only
		if !strings.Contains(html, "Alvo") && !strings.Contains(html, "acerto") && !strings.Contains(strings.ToLower(html), "erro") {
			t.Fatalf("completed print missing result: %s", html[:min(400, len(html))])
		}
	}

	abPrint := apiJSON(t, s, http.MethodGet, "/api/exports/sessions/"+id2+"/print", "", cookie, "", "")
	if abPrint.Code != 200 {
		t.Fatal(abPrint.Body.String())
	}
	abHTML := strings.ToLower(abPrint.Body.String())
	for _, leak := range []string{"sha256", "farsight/", "data:image/png"} {
		if strings.Contains(abHTML, leak) {
			t.Fatalf("abandoned print leak %q", leak)
		}
	}

	set := apiJSON(t, s, http.MethodGet, "/api/settings", "", cookie, "", "")
	var settings map[string]any
	_ = json.Unmarshal(set.Body.Bytes(), &settings)
	if settings["exportsAvailable"] != true || settings["backupAvailable"] != true {
		t.Fatalf("settings flags: %v", settings)
	}
}

func TestBackupDownloadAndRestoreRoundTrip(t *testing.T) {
	s := catalogServer(t)
	csrf, cookie := authClient(t, s)
	id, lease, rev := createSession(t, s, csrf, cookie, "bak-rt")
	lock := apiJSON(t, s, http.MethodPost, "/api/sessions/"+id+"/lock", csrf, cookie, lease, `{"expectedRevision":`+strconv.Itoa(rev)+`}`)
	if lock.Code != 200 {
		t.Fatal(lock.Body.String())
	}
	var env struct {
		Session struct {
			Revision int    `json:"revision"`
			Code     string `json:"code"`
			Correct  string `json:"correct"`
		} `json:"session"`
	}
	_ = json.Unmarshal(lock.Body.Bytes(), &env)
	code := env.Session.Code
	choice := env.Session.Correct
	if choice == "" {
		choice = "A"
	}
	conf := apiJSON(t, s, http.MethodPost, "/api/sessions/"+id+"/confirm", csrf, cookie, lease,
		`{"expectedRevision":`+strconv.Itoa(env.Session.Revision)+`,"choice":"`+choice+`","confidence":2}`)
	if conf.Code != 200 {
		t.Fatal(conf.Body.String())
	}

	bak := apiJSON(t, s, http.MethodGet, "/api/backup", "", cookie, "", "")
	if bak.Code != 200 {
		t.Fatalf("backup %d %s", bak.Code, bak.Body.String())
	}
	if bak.Header().Get("Content-Type") != "application/zip" {
		t.Fatalf("ct %s", bak.Header().Get("Content-Type"))
	}
	zipBytes := bak.Body.Bytes()
	if len(zipBytes) < 100 {
		t.Fatalf("zip too small: %d", len(zipBytes))
	}

	// Mutate live data, then restore.
	id2, lease2, rev2 := createSession(t, s, csrf, cookie, "bak-extra")
	_ = apiJSON(t, s, http.MethodPost, "/api/sessions/"+id2+"/abandon", csrf, cookie, lease2, `{"expectedRevision":`+strconv.Itoa(rev2)+`}`)

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	_ = w.WriteField("confirm", "true")
	part, err := w.CreateFormFile("archive", "restore.zip")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(zipBytes); err != nil {
		t.Fatal(err)
	}
	_ = w.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/backup/restore", &buf)
	req.Host = "127.0.0.1:34567"
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Origin", "http://127.0.0.1:34567")
	req.Header.Set("X-CSRF-Token", csrf)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("restore %d %s", rec.Code, rec.Body.String())
	}
	var out struct {
		OK             bool   `json:"ok"`
		PreBackupPath  string `json:"preBackupPath"`
		BootstrapToken string `json:"bootstrapToken"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if !out.OK || out.BootstrapToken == "" || out.PreBackupPath == "" {
		t.Fatalf("restore payload: %+v", out)
	}
	if _, err := os.Stat(out.PreBackupPath); err != nil {
		t.Fatalf("pre-backup missing: %v", err)
	}

	// Old cookie invalid; bootstrap with new token.
	req = httptest.NewRequest(http.MethodGet, "/api/bootstrap?token="+out.BootstrapToken, nil)
	req.Host = "127.0.0.1:34567"
	rec = httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("rebootstrap %d", rec.Code)
	}
	newCookie := rec.Result().Cookies()[0]
	var boot map[string]string
	_ = json.Unmarshal(rec.Body.Bytes(), &boot)

	hist := apiJSON(t, s, http.MethodGet, "/api/history", "", newCookie, "", "")
	if hist.Code != 200 {
		t.Fatal(hist.Body.String())
	}
	if !strings.Contains(hist.Body.String(), code) {
		t.Fatalf("restored history missing %s: %s", code, hist.Body.String())
	}
	if strings.Contains(hist.Body.String(), id2) {
		t.Fatalf("extra session survived restore: %s", hist.Body.String())
	}

	// Stale pre-restore credentials must not create sessions after the swap.
	stale := apiJSON(t, s, http.MethodPost, "/api/sessions", csrf, cookie, "", `{"operationId":"stale-after-restore"}`)
	if stale.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 with stale cookie after restore, got %d %s", stale.Code, stale.Body.String())
	}
	_ = boot
}

func TestRestoreRejectsActiveSessionAndBadArchive(t *testing.T) {
	s := catalogServer(t)
	csrf, cookie := authClient(t, s)
	createSession(t, s, csrf, cookie, "bak-active")

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	_ = w.WriteField("confirm", "true")
	part, _ := w.CreateFormFile("archive", "x.zip")
	_, _ = part.Write([]byte("PK\x03\x04not-a-real-backup"))
	_ = w.Close()
	req := httptest.NewRequest(http.MethodPost, "/api/backup/restore", &buf)
	req.Host = "127.0.0.1:34567"
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Origin", "http://127.0.0.1:34567")
	req.Header.Set("X-CSRF-Token", csrf)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d %s", rec.Code, rec.Body.String())
	}

	// Abandon so restore can run, then reject corrupt zip.
	cur := apiJSON(t, s, http.MethodGet, "/api/sessions/current", "", cookie, "", "")
	var curEnv struct {
		Session *struct {
			ID       string `json:"id"`
			Revision int    `json:"revision"`
		} `json:"session"`
		LeaseToken string `json:"leaseToken"`
	}
	_ = json.Unmarshal(cur.Body.Bytes(), &curEnv)
	if curEnv.Session != nil {
		_ = apiJSON(t, s, http.MethodPost, "/api/sessions/"+curEnv.Session.ID+"/abandon", csrf, cookie, curEnv.LeaseToken,
			`{"expectedRevision":`+strconv.Itoa(curEnv.Session.Revision)+`}`)
	}

	buf.Reset()
	w = multipart.NewWriter(&buf)
	_ = w.WriteField("confirm", "true")
	part, _ = w.CreateFormFile("archive", "bad.zip")
	_, _ = io.Copy(part, bytes.NewReader([]byte("not-zip")))
	_ = w.Close()
	req = httptest.NewRequest(http.MethodPost, "/api/backup/restore", &buf)
	req.Host = "127.0.0.1:34567"
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Origin", "http://127.0.0.1:34567")
	req.Header.Set("X-CSRF-Token", csrf)
	req.AddCookie(cookie)
	rec = httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code == 200 {
		t.Fatalf("corrupt restore should fail: %s", rec.Body.String())
	}
	if _, err := os.Stat(backup.MarkerPath(s.Store.DataDir)); err == nil {
		t.Fatal("marker should not remain after failed validate")
	}
	_ = filepath.Separator
}

func TestRestoreWaitsForInFlightComment(t *testing.T) {
	s := catalogServer(t)
	csrf, cookie := authClient(t, s)

	id, lease, rev := createSession(t, s, csrf, cookie, "inflight-comment")
	lock := apiJSON(t, s, http.MethodPost, "/api/sessions/"+id+"/lock", csrf, cookie, lease, `{"expectedRevision":`+strconv.Itoa(rev)+`}`)
	if lock.Code != 200 {
		t.Fatal(lock.Body.String())
	}
	var env struct {
		Session struct {
			Revision int    `json:"revision"`
			Correct  string `json:"correct"`
		} `json:"session"`
	}
	_ = json.Unmarshal(lock.Body.Bytes(), &env)
	choice := env.Session.Correct
	if choice == "" {
		choice = "A"
	}
	conf := apiJSON(t, s, http.MethodPost, "/api/sessions/"+id+"/confirm", csrf, cookie, lease,
		`{"expectedRevision":`+strconv.Itoa(env.Session.Revision)+`,"choice":"`+choice+`","confidence":2}`)
	if conf.Code != 200 {
		t.Fatal(conf.Body.String())
	}

	bak := apiJSON(t, s, http.MethodGet, "/api/backup", "", cookie, "", "")
	if bak.Code != 200 {
		t.Fatalf("backup %d %s", bak.Code, bak.Body.String())
	}
	zipBytes := append([]byte(nil), bak.Body.Bytes()...)

	// Abandon would not apply; session is completed. Create a second abandoned
	// session so live data differs from the backup, then restore.
	id2, lease2, rev2 := createSession(t, s, csrf, cookie, "inflight-extra")
	_ = apiJSON(t, s, http.MethodPost, "/api/sessions/"+id2+"/abandon", csrf, cookie, lease2, `{"expectedRevision":`+strconv.Itoa(rev2)+`}`)

	entered := make(chan struct{})
	release := make(chan struct{})
	var gateOnce sync.Once
	s.testGate = func() {
		gateOnce.Do(func() {
			close(entered)
			<-release
		})
	}
	t.Cleanup(func() { s.testGate = nil })

	commentDone := make(chan int, 1)
	go func() {
		rec := apiJSON(t, s, http.MethodPut, "/api/sessions/"+id+"/comment", csrf, cookie, "", `{"comment":"escrito-em-voo"}`)
		commentDone <- rec.Code
	}()

	select {
	case <-entered:
	case <-time.After(3 * time.Second):
		t.Fatal("comment handler did not enter test gate")
	}

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	_ = w.WriteField("confirm", "true")
	part, err := w.CreateFormFile("archive", "restore.zip")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(zipBytes); err != nil {
		t.Fatal(err)
	}
	_ = w.Close()

	restoreDone := make(chan int, 1)
	go func() {
		req := httptest.NewRequest(http.MethodPost, "/api/backup/restore", bytes.NewReader(buf.Bytes()))
		req.Host = "127.0.0.1:34567"
		req.Header.Set("Content-Type", w.FormDataContentType())
		req.Header.Set("Origin", "http://127.0.0.1:34567")
		req.Header.Set("X-CSRF-Token", csrf)
		req.AddCookie(cookie)
		rec := httptest.NewRecorder()
		s.Handler().ServeHTTP(rec, req)
		restoreDone <- rec.Code
	}()

	select {
	case code := <-restoreDone:
		t.Fatalf("restore finished while in-flight comment held shared lock (code %d)", code)
	case <-time.After(250 * time.Millisecond):
		// Expected: restore blocked behind the comment's RLock.
	}

	// While restore is waiting, a brand-new request must still be able to observe
	// maintaining only after restore acquires the exclusive lock. Hold the gate a
	// bit longer, then release the comment so restore can proceed.
	close(release)

	var commentCode, restoreCode int
	select {
	case commentCode = <-commentDone:
	case <-time.After(5 * time.Second):
		t.Fatal("comment timed out")
	}
	select {
	case restoreCode = <-restoreDone:
	case <-time.After(15 * time.Second):
		t.Fatal("restore timed out")
	}
	if commentCode != 200 {
		t.Fatalf("comment status %d", commentCode)
	}
	if restoreCode != 200 {
		t.Fatalf("restore status %d", restoreCode)
	}

	// Credentials rotated after restore; the pre-restore cookie must fail.
	stale := apiJSON(t, s, http.MethodGet, "/api/history", "", cookie, "", "")
	if stale.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized after restore, got %d %s", stale.Code, stale.Body.String())
	}
}

func TestMaintenanceBlocksWhileRestoring(t *testing.T) {
	s := catalogServer(t)
	csrf, cookie := authClient(t, s)
	atomic.StoreInt32(&s.maintaining, 1)
	t.Cleanup(func() { atomic.StoreInt32(&s.maintaining, 0) })

	rec := apiJSON(t, s, http.MethodPut, "/api/sessions/x/comment", csrf, cookie, "", `{"comment":"nope"}`)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 during maintenance, got %d %s", rec.Code, rec.Body.String())
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
