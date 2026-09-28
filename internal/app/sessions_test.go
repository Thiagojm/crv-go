package app

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/Thiagojm/crv-go/internal/catalog"
	"github.com/Thiagojm/crv-go/internal/session"
	"github.com/Thiagojm/crv-go/internal/store"
)

func catalogServer(t *testing.T) *Server {
	t.Helper()
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
	return &Server{
		Store:    st,
		Security: sec,
		Sessions: session.New(st),
		Port:     34567,
		Ready:    true,
		Report:   res.Report,
	}
}

func apiJSON(t *testing.T, s *Server, method, path, csrf string, cookie *http.Cookie, lease, body string) *httptest.ResponseRecorder {
	t.Helper()
	var rdr io.Reader
	if body != "" {
		rdr = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, rdr)
	req.Host = "127.0.0.1:34567"
	if cookie != nil {
		req.AddCookie(cookie)
	}
	if csrf != "" {
		req.Header.Set("X-CSRF-Token", csrf)
		req.Header.Set("Origin", "http://127.0.0.1:34567")
	}
	if lease != "" {
		req.Header.Set("X-Lease-Token", lease)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	return rec
}

func createSession(t *testing.T, s *Server, csrf string, cookie *http.Cookie, op string) (id, lease string, rev int) {
	t.Helper()
	rec := apiJSON(t, s, http.MethodPost, "/api/sessions", csrf, cookie, "", `{"operationId":"`+op+`"}`)
	if rec.Code != 200 {
		t.Fatalf("create %d %s", rec.Code, rec.Body.String())
	}
	var env struct {
		LeaseToken string `json:"leaseToken"`
		Session    struct {
			ID       string `json:"id"`
			Revision int    `json:"revision"`
			State    string `json:"state"`
		} `json:"session"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatal(err)
	}
	return env.Session.ID, env.LeaseToken, env.Session.Revision
}

func TestSessionBlindingAndImages(t *testing.T) {
	s := catalogServer(t)
	csrf, cookie := authClient(t, s)
	id, lease, rev := createSession(t, s, csrf, cookie, "op-blind")

	got := apiJSON(t, s, http.MethodGet, "/api/sessions/"+id, "", cookie, lease, "")
	if got.Code != 200 {
		t.Fatal(got.Body.String())
	}
	raw := got.Body.String()
	for _, leak := range []string{"target_sha", "targetSha", "sha256", "pos_a", "farsight", "originals/", "credit"} {
		if strings.Contains(strings.ToLower(raw), strings.ToLower(leak)) {
			t.Fatalf("leak %q in collecting: %s", leak, raw)
		}
	}
	var env map[string]any
	_ = json.Unmarshal(got.Body.Bytes(), &env)
	sess := env["session"].(map[string]any)
	if sess["alternatives"] != nil {
		t.Fatal("alternatives during collection")
	}

	img := apiJSON(t, s, http.MethodGet, "/api/sessions/"+id+"/images/A", "", cookie, lease, "")
	if img.Code != http.StatusNotFound {
		t.Fatalf("image before lock: %d", img.Code)
	}
	guess := apiJSON(t, s, http.MethodGet, "/api/sessions/"+id+"/images/secret", "", cookie, lease, "")
	if guess.Code != http.StatusNotFound {
		t.Fatalf("guessed pos: %d", guess.Code)
	}

	lock := apiJSON(t, s, http.MethodPost, "/api/sessions/"+id+"/lock", csrf, cookie, lease, `{"expectedRevision":`+strconv.Itoa(rev)+`}`)
	if lock.Code != 200 {
		t.Fatalf("lock %d %s", lock.Code, lock.Body.String())
	}
	raw = lock.Body.String()
	for _, leak := range []string{"sha256", "targetSha", "credit", "description"} {
		if strings.Contains(strings.ToLower(raw), strings.ToLower(leak)) {
			t.Fatalf("leak %q after lock: %s", leak, raw)
		}
	}
	_ = json.Unmarshal(lock.Body.Bytes(), &env)
	sess = env["session"].(map[string]any)
	alts := sess["alternatives"].([]any)
	if len(alts) != 4 {
		t.Fatal("need 4 alternatives")
	}
	rev = int(sess["revision"].(float64))

	img = apiJSON(t, s, http.MethodGet, "/api/sessions/"+id+"/images/A", "", cookie, lease, "")
	if img.Code != 200 || !strings.Contains(img.Header().Get("Content-Type"), "image/png") {
		t.Fatalf("locked image %d %s", img.Code, img.Header().Get("Content-Type"))
	}
	if img.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("cache")
	}

	confirm := apiJSON(t, s, http.MethodPost, "/api/sessions/"+id+"/confirm", csrf, cookie, lease,
		`{"expectedRevision":`+strconv.Itoa(rev)+`,"choice":"A"}`)
	if confirm.Code != 200 {
		t.Fatalf("confirm %d %s", confirm.Code, confirm.Body.String())
	}
	_ = json.Unmarshal(confirm.Body.Bytes(), &env)
	sess = env["session"].(map[string]any)
	if sess["state"] != "completed" || sess["feedback"] == nil || sess["hit"] == nil {
		t.Fatalf("feedback missing: %s", confirm.Body.String())
	}
	replay := apiJSON(t, s, http.MethodPost, "/api/sessions/"+id+"/confirm", csrf, cookie, lease,
		`{"expectedRevision":`+strconv.Itoa(rev)+`,"choice":"A"}`)
	if replay.Code != 200 {
		t.Fatalf("replay %d %s", replay.Code, replay.Body.String())
	}
	conflict := apiJSON(t, s, http.MethodPost, "/api/sessions/"+id+"/confirm", csrf, cookie, lease,
		`{"expectedRevision":`+strconv.Itoa(rev)+`,"choice":"B"}`)
	if conflict.Code != http.StatusConflict {
		t.Fatalf("choice conflict %d", conflict.Code)
	}
}

func TestConcurrentCreateAndAbandon(t *testing.T) {
	s := catalogServer(t)
	csrf, cookie := authClient(t, s)
	var wg sync.WaitGroup
	codes := make([]int, 2)
	bodies := make([]string, 2)
	wg.Add(2)
	for i := 0; i < 2; i++ {
		i := i
		go func() {
			defer wg.Done()
			rec := apiJSON(t, s, http.MethodPost, "/api/sessions", csrf, cookie, "", `{"operationId":"op-`+strconv.Itoa(i)+`"}`)
			codes[i] = rec.Code
			bodies[i] = rec.Body.String()
		}()
	}
	wg.Wait()
	ok, conflict := 0, 0
	for i, c := range codes {
		switch c {
		case 200:
			ok++
		case http.StatusConflict:
			conflict++
		default:
			t.Fatalf("create %d: %d %s", i, c, bodies[i])
		}
	}
	if ok != 1 || conflict != 1 {
		t.Fatalf("concurrent create ok=%d conflict=%d bodies=%v", ok, conflict, bodies)
	}
}

func TestAbandonBeforeAndAfterAlts(t *testing.T) {
	s := catalogServer(t)
	csrf, cookie := authClient(t, s)
	id, lease, rev := createSession(t, s, csrf, cookie, "ab1")
	ab := apiJSON(t, s, http.MethodPost, "/api/sessions/"+id+"/abandon", csrf, cookie, lease, `{"expectedRevision":`+strconv.Itoa(rev)+`}`)
	if ab.Code != 200 {
		t.Fatal(ab.Body.String())
	}
	img := apiJSON(t, s, http.MethodGet, "/api/sessions/"+id+"/images/A", "", cookie, lease, "")
	if img.Code != 404 {
		t.Fatalf("abandoned before alts image %d", img.Code)
	}
	raw := strings.ToLower(ab.Body.String())
	if strings.Contains(raw, "feedback") && strings.Contains(raw, `"hit"`) {
		t.Fatal("abandoned collecting leaked result")
	}

	id, lease, rev = createSession(t, s, csrf, cookie, "ab2")
	lock := apiJSON(t, s, http.MethodPost, "/api/sessions/"+id+"/lock", csrf, cookie, lease, `{"expectedRevision":`+strconv.Itoa(rev)+`}`)
	if lock.Code != 200 {
		t.Fatal(lock.Body.String())
	}
	var env struct {
		Session struct {
			Revision int `json:"revision"`
		} `json:"session"`
	}
	_ = json.Unmarshal(lock.Body.Bytes(), &env)
	ab = apiJSON(t, s, http.MethodPost, "/api/sessions/"+id+"/abandon", csrf, cookie, lease, `{"expectedRevision":`+strconv.Itoa(env.Session.Revision)+`}`)
	if ab.Code != 200 {
		t.Fatal(ab.Body.String())
	}
	if strings.Contains(strings.ToLower(ab.Body.String()), `"correct"`) {
		t.Fatal("abandoned after alts leaked correct")
	}
	img = apiJSON(t, s, http.MethodGet, "/api/sessions/"+id+"/images/B", "", cookie, lease, "")
	if img.Code != 200 {
		t.Fatalf("alts remain after abandon: %d", img.Code)
	}
}

func TestStaleRevisionAndLease(t *testing.T) {
	s := catalogServer(t)
	csrf, cookie := authClient(t, s)
	id, lease, rev := createSession(t, s, csrf, cookie, "stale")
	body := `{"expectedRevision":` + strconv.Itoa(rev) + `,"record":{"version":"crv-record-v1","groups":{},"drawings":{"ideogram":[],"sketch":[]},"summary":["","","","",""],"step":1}}`
	save := apiJSON(t, s, http.MethodPut, "/api/sessions/"+id+"/record", csrf, cookie, lease, body)
	if save.Code != 200 {
		t.Fatal(save.Body.String())
	}
	stale := apiJSON(t, s, http.MethodPut, "/api/sessions/"+id+"/record", csrf, cookie, lease, body)
	if stale.Code != http.StatusConflict {
		t.Fatalf("stale %d %s", stale.Code, stale.Body.String())
	}
	other := apiJSON(t, s, http.MethodPut, "/api/sessions/"+id+"/record", csrf, cookie, "nope", body)
	if other.Code != http.StatusConflict {
		t.Fatalf("lease %d", other.Code)
	}
}

func TestFailedSaveDoesNotLock(t *testing.T) {
	s := catalogServer(t)
	csrf, cookie := authClient(t, s)
	id, lease, rev := createSession(t, s, csrf, cookie, "fail")
	bad := apiJSON(t, s, http.MethodPut, "/api/sessions/"+id+"/record", csrf, cookie, lease,
		`{"expectedRevision":`+strconv.Itoa(rev)+`,"record":{"groups":{"movement":{"ids":["nope"]}}}}`)
	if bad.Code != http.StatusBadRequest {
		t.Fatalf("invalid record %d %s", bad.Code, bad.Body.String())
	}
	lock := apiJSON(t, s, http.MethodPost, "/api/sessions/"+id+"/lock", csrf, cookie, lease, `{"expectedRevision":0}`)
	if lock.Code != http.StatusConflict {
		t.Fatalf("lock with bad rev %d", lock.Code)
	}
	img := apiJSON(t, s, http.MethodGet, "/api/sessions/"+id+"/images/A", "", cookie, lease, "")
	if img.Code != 404 {
		t.Fatal("must not unlock images")
	}
}

func TestRestartPausesActiveSession(t *testing.T) {
	s := catalogServer(t)
	csrf, cookie := authClient(t, s)
	id, lease, _ := createSession(t, s, csrf, cookie, "pause")
	s2 := &Server{Store: s.Store, Security: s.Security, Sessions: session.New(s.Store), Port: 34567, Ready: true, Report: s.Report}
	got := apiJSON(t, s2, http.MethodGet, "/api/sessions/"+id, "", cookie, lease, "")
	if got.Code != 200 {
		t.Fatal(got.Body.String())
	}
	var env struct {
		Session struct {
			Paused bool `json:"paused"`
		} `json:"session"`
	}
	_ = json.Unmarshal(got.Body.Bytes(), &env)
	if !env.Session.Paused {
		t.Fatal("restart should pause")
	}
}

func TestReadyOpensSessionsInPhase2(t *testing.T) {
	s := catalogServer(t)
	csrf, cookie := authClient(t, s)
	_ = csrf
	got := apiJSON(t, s, http.MethodGet, "/api/ready", "", cookie, "", "")
	var body map[string]any
	_ = json.Unmarshal(got.Body.Bytes(), &body)
	if body["sessionsOpen"] != true || body["phase"] != "2" {
		t.Fatalf("%v", body)
	}
}
