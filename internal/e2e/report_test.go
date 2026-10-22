package e2e

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/ihsan-sa/hive-standalone-build/internal/api"
	"github.com/ihsan-sa/hive-standalone-build/internal/mod/report"
	"github.com/ihsan-sa/hive-standalone-build/internal/mod/work"
)

// Sharing failures: while sharing is off a blocked task sends nothing; once
// it's on, the next blocked task goes out as one batch with a local copy, the
// earlier one stays home, and a second send with nothing new sends nothing.
func TestReportSharing(t *testing.T) {
	t.Parallel()
	var mu sync.Mutex
	var got [][]byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		mu.Lock()
		got = append(got, raw)
		mu.Unlock()
	}))
	defer srv.Close()
	batches := func() int { mu.Lock(); defer mu.Unlock(); return len(got) }

	h := start(t)
	cfg := h.ws.LoadConfig()
	cfg.ShareURL = srv.URL
	h.ws.SaveConfig(cfg)
	h.seed("me/rep", map[string]string{".hive/project.json": `{"test": "true"}`, ".hive/goals.md": "# Goals\n"})
	h.post("/api/project", map[string]any{"name": "rep", "repo": "me/rep"}, nil)
	send := func() {
		t.Helper()
		before := time.Now()
		h.post("/api/job", map[string]string{"name": "report.send"}, nil)
		h.wait("report.send to run", minute, func(st api.State) bool {
			for _, j := range st.Jobs {
				if j.Name == "report.send" && j.Last.After(before) && !j.Running {
					return true
				}
			}
			return false
		})
	}

	h.post("/api/ask", map[string]string{"project": "rep", "title": "rename the widget", "known": "FAKE:block"}, nil)
	h.wait("the first task to block", 2*minute, taskStage("rep", "rename the widget", work.StageBlocked))
	send()
	if n := batches(); n != 0 || len(report.Sent(h.ws)) != 0 {
		t.Fatalf("sharing is off, yet %d batches went out", n)
	}

	time.Sleep(1100 * time.Millisecond) // sharing starts on the next second
	cfg = h.ws.LoadConfig()
	cfg.Share = time.Now().UTC().Format(time.RFC3339)
	h.ws.SaveConfig(cfg)
	h.post("/api/ask", map[string]string{"project": "rep", "title": "port the parser", "known": "FAKE:block"}, nil)
	h.wait("the second task to block", 2*minute, taskStage("rep", "port the parser", work.StageBlocked))
	send()
	if n := batches(); n != 1 {
		t.Fatalf("want one batch, got %d", n)
	}
	var p report.Payload
	if err := json.Unmarshal(got[0], &p); err != nil {
		t.Fatal(err)
	}
	blocked := 0
	for _, f := range p.Failures {
		if f.Kind == "task-blocked" {
			blocked++
		}
	}
	if p.Schema != 1 || p.Install == "" || p.Harness != "fake" || blocked != 1 || p.Usage.Runs == 0 {
		t.Fatalf("payload: %s", got[0])
	}
	sent := report.Sent(h.ws)
	if raw, _ := os.ReadFile(sent[len(sent)-1]); len(sent) != 1 || string(raw) != string(got[0]) {
		t.Fatalf("local copies %v don't match what was sent", sent)
	}
	send()
	if n := batches(); n != 1 {
		t.Fatalf("nothing new happened, yet %d batches went out", n)
	}
}
