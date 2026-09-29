package app

import (
	"encoding/json"
	"math"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Thiagojm/crv-go/internal/store"
)

func lockAndConfirm(t *testing.T, s *Server, csrf string, cookie *http.Cookie, id, lease string, rev int, choice string) (hit bool, newRev int) {
	t.Helper()
	lock := apiJSON(t, s, http.MethodPost, "/api/sessions/"+id+"/lock", csrf, cookie, lease, `{"expectedRevision":`+strconv.Itoa(rev)+`}`)
	if lock.Code != 200 {
		t.Fatalf("lock %d %s", lock.Code, lock.Body.String())
	}
	var env struct {
		Session struct {
			Revision int `json:"revision"`
		} `json:"session"`
	}
	if err := json.Unmarshal(lock.Body.Bytes(), &env); err != nil {
		t.Fatal(err)
	}
	confirm := apiJSON(t, s, http.MethodPost, "/api/sessions/"+id+"/confirm", csrf, cookie, lease,
		`{"expectedRevision":`+strconv.Itoa(env.Session.Revision)+`,"choice":"`+choice+`"}`)
	if confirm.Code != 200 {
		t.Fatalf("confirm %d %s", confirm.Code, confirm.Body.String())
	}
	var out struct {
		Session struct {
			Revision int   `json:"revision"`
			Hit      *bool `json:"hit"`
		} `json:"session"`
	}
	if err := json.Unmarshal(confirm.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.Session.Hit == nil {
		t.Fatal("missing hit")
	}
	return *out.Session.Hit, out.Session.Revision
}

func TestStatisticsA9AndHistoryFilters(t *testing.T) {
	s := catalogServer(t)
	csrf, cookie := authClient(t, s)

	empty := apiJSON(t, s, http.MethodGet, "/api/statistics", "", cookie, "", "")
	if empty.Code != 200 {
		t.Fatal(empty.Body.String())
	}
	var emptyStats map[string]any
	_ = json.Unmarshal(empty.Body.Bytes(), &emptyStats)
	if emptyStats["hitRate"] != nil || emptyStats["confirmedChoices"].(float64) != 0 {
		t.Fatalf("empty stats: %s", empty.Body.String())
	}

	// Abandon before alternatives.
	id, lease, rev := createSession(t, s, csrf, cookie, "hist-ab")
	ab := apiJSON(t, s, http.MethodPost, "/api/sessions/"+id+"/abandon", csrf, cookie, lease, `{"expectedRevision":`+strconv.Itoa(rev)+`}`)
	if ab.Code != 200 {
		t.Fatal(ab.Body.String())
	}
	abandonedID := id

	var gotHit, gotMiss bool
	for i := 0; i < 24 && !(gotHit && gotMiss); i++ {
		sid, slease, srev := createSession(t, s, csrf, cookie, "hist-c-"+strconv.Itoa(i))
		hit, _ := lockAndConfirm(t, s, csrf, cookie, sid, slease, srev, "A")
		if hit {
			gotHit = true
		} else {
			gotMiss = true
		}
	}
	if !gotHit || !gotMiss {
		t.Fatalf("could not obtain hit and miss via API (hit=%v miss=%v)", gotHit, gotMiss)
	}

	statsRec := apiJSON(t, s, http.MethodGet, "/api/statistics", "", cookie, "", "")
	if statsRec.Code != 200 {
		t.Fatal(statsRec.Body.String())
	}
	var stats struct {
		Initiated        int      `json:"initiated"`
		Completed        int      `json:"completed"`
		AbandonedBefore  int      `json:"abandonedBefore"`
		AbandonedAfter   int      `json:"abandonedAfter"`
		ConfirmedChoices int      `json:"confirmedChoices"`
		Hits             int      `json:"hits"`
		HitRate          *float64 `json:"hitRate"`
		Chart            []struct {
			N         int     `json:"n"`
			Hits      int     `json:"hits"`
			Reference float64 `json:"reference"`
			Hit       bool    `json:"hit"`
		} `json:"chart"`
	}
	if err := json.Unmarshal(statsRec.Body.Bytes(), &stats); err != nil {
		t.Fatal(err)
	}
	if stats.AbandonedBefore != 1 || stats.AbandonedAfter != 0 {
		t.Fatalf("abandon counts: %+v", stats)
	}
	if stats.ConfirmedChoices != stats.Completed || stats.ConfirmedChoices < 2 || stats.Hits < 1 {
		t.Fatalf("choices: %+v", stats)
	}
	if stats.HitRate == nil {
		t.Fatal("hitRate nil with confirmed choices")
	}
	wantRate := float64(stats.Hits) / float64(stats.ConfirmedChoices)
	if math.Abs(*stats.HitRate-wantRate) > 1e-9 {
		t.Fatalf("hitRate %v want %v", *stats.HitRate, wantRate)
	}
	if len(stats.Chart) != stats.Completed {
		t.Fatalf("chart len %d completed %d", len(stats.Chart), stats.Completed)
	}
	cum := 0
	for i, p := range stats.Chart {
		if p.Hit {
			cum++
		}
		if p.N != i+1 || p.Hits != cum || math.Abs(p.Reference-0.25*float64(p.N)) > 1e-9 {
			t.Fatalf("chart[%d]=%+v cum=%d", i, p, cum)
		}
	}

	// Seed one precise A9 trio via store for exact rate check on a fresh server.
	s2 := catalogServer(t)
	_, cookie2 := authClient(t, s2)
	active, err := s2.Store.ActiveCatalog()
	if err != nil || active == nil {
		t.Fatal(err)
	}
	shas, err := s2.Store.EligibleSHAs(active.RevisionID)
	if err != nil || len(shas) < 4 {
		t.Fatal(err)
	}
	hitV, missV := 1, 0
	ts := time.Now().UTC().Format(time.RFC3339)
	seed := []*store.SessionRow{
		{
			ID: "a9-1", Code: "a9-hit", CreateOpID: "a9-1", State: "completed", Revision: 2, Step: 6,
			CatalogRevisionID: active.RevisionID, TargetSHA: shas[0], PosA: shas[0], PosB: shas[1], PosC: shas[2], PosD: shas[3],
			ProtocolJSON: "{}", HelpVersion: "v", RecordJSON: "{}", LockedRecordJSON: "{}",
			ConfirmedChoice: "A", Hit: &hitV, CreatedAt: ts, UpdatedAt: ts, CompletedAt: "2026-09-20T10:00:00Z",
			OpenedExamplesJSON: "[]",
		},
		{
			ID: "a9-2", Code: "a9-miss", CreateOpID: "a9-2", State: "completed", Revision: 2, Step: 6,
			CatalogRevisionID: active.RevisionID, TargetSHA: shas[0], PosA: shas[0], PosB: shas[1], PosC: shas[2], PosD: shas[3],
			ProtocolJSON: "{}", HelpVersion: "v", RecordJSON: "{}", LockedRecordJSON: "{}",
			ConfirmedChoice: "B", Hit: &missV, CreatedAt: ts, UpdatedAt: ts, CompletedAt: "2026-09-20T11:00:00Z",
			OpenedExamplesJSON: "[]",
		},
		{
			ID: "a9-3", Code: "a9-ab", CreateOpID: "a9-3", State: "abandoned", Revision: 2, Step: 1,
			CatalogRevisionID: active.RevisionID, TargetSHA: shas[0], PosA: shas[0], PosB: shas[1], PosC: shas[2], PosD: shas[3],
			ProtocolJSON: "{}", HelpVersion: "v", RecordJSON: "{}", CreatedAt: ts, UpdatedAt: ts,
			AbandonedAt: "2026-09-20T12:00:00Z", OpenedExamplesJSON: "[]",
		},
	}
	for _, row := range seed {
		if err := s2.Store.InsertSession(row); err != nil {
			t.Fatal(err)
		}
	}
	exact := apiJSON(t, s2, http.MethodGet, "/api/statistics", "", cookie2, "", "")
	var a9 struct {
		Initiated        int      `json:"initiated"`
		ConfirmedChoices int      `json:"confirmedChoices"`
		Hits             int      `json:"hits"`
		AbandonedBefore  int      `json:"abandonedBefore"`
		HitRate          *float64 `json:"hitRate"`
	}
	_ = json.Unmarshal(exact.Body.Bytes(), &a9)
	if a9.Initiated != 3 || a9.ConfirmedChoices != 2 || a9.Hits != 1 || a9.AbandonedBefore != 1 ||
		a9.HitRate == nil || math.Abs(*a9.HitRate-0.5) > 1e-9 {
		t.Fatalf("A9 exact: %+v body=%s", a9, exact.Body.String())
	}

	list := apiJSON(t, s2, http.MethodGet, "/api/history?state=completed,abandoned", "", cookie2, "", "")
	if list.Code != 200 {
		t.Fatal(list.Body.String())
	}
	var page struct {
		Items []map[string]any `json:"items"`
		Total int              `json:"total"`
	}
	_ = json.Unmarshal(list.Body.Bytes(), &page)
	if page.Total != 3 {
		t.Fatalf("list total %d", page.Total)
	}
	for _, it := range page.Items {
		raw, _ := json.Marshal(it)
		low := strings.ToLower(string(raw))
		for _, leak := range []string{"target", "sha256", "feedback", "pos_a", "secret"} {
			if strings.Contains(low, leak) {
				t.Fatalf("history item leak %q: %s", leak, raw)
			}
		}
		if it["state"] == "abandoned" {
			if it["confirmedChoice"] != nil && it["confirmedChoice"] != "" {
				t.Fatalf("abandoned choice: %v", it)
			}
			if _, ok := it["hit"]; ok && it["hit"] != nil {
				t.Fatalf("abandoned hit: %v", it)
			}
		}
	}

	onlyCompleted := apiJSON(t, s2, http.MethodGet, "/api/history?state=completed", "", cookie2, "", "")
	_ = json.Unmarshal(onlyCompleted.Body.Bytes(), &page)
	if page.Total != 2 {
		t.Fatalf("completed filter %d", page.Total)
	}

	detail := apiJSON(t, s, http.MethodGet, "/api/sessions/"+abandonedID, "", cookie, "", "")
	if detail.Code != 200 {
		t.Fatal(detail.Body.String())
	}
	raw := strings.ToLower(detail.Body.String())
	for _, leak := range []string{`"feedback"`, `"correct"`, "sha256", "targetsha", "target_sha"} {
		if strings.Contains(raw, leak) {
			t.Fatalf("abandoned detail leak %q: %s", leak, detail.Body.String())
		}
	}
	if strings.Contains(raw, `"hit":`) {
		t.Fatalf("abandoned detail hit: %s", detail.Body.String())
	}

	settings := apiJSON(t, s, http.MethodGet, "/api/settings", "", cookie, "", "")
	var set map[string]any
	_ = json.Unmarshal(settings.Body.Bytes(), &set)
	if set["phase"] != "3" || set["historyAvailable"] != true {
		t.Fatalf("settings: %v", set)
	}
}

func TestAbandonedHistoryDetailNoLeak(t *testing.T) {
	s := catalogServer(t)
	csrf, cookie := authClient(t, s)
	id, lease, rev := createSession(t, s, csrf, cookie, "ab-detail")
	ab := apiJSON(t, s, http.MethodPost, "/api/sessions/"+id+"/abandon", csrf, cookie, lease, `{"expectedRevision":`+strconv.Itoa(rev)+`}`)
	if ab.Code != 200 {
		t.Fatal(ab.Body.String())
	}
	hist := apiJSON(t, s, http.MethodGet, "/api/history?state=abandoned", "", cookie, "", "")
	if hist.Code != 200 || !strings.Contains(hist.Body.String(), id) {
		t.Fatalf("history: %s", hist.Body.String())
	}
	got := apiJSON(t, s, http.MethodGet, "/api/sessions/"+id, "", cookie, "", "")
	raw := got.Body.String()
	low := strings.ToLower(raw)
	for _, leak := range []string{"feedback", `"correct"`, "sha256", "target"} {
		if strings.Contains(low, leak) {
			t.Fatalf("leak %q in %s", leak, raw)
		}
	}
}
