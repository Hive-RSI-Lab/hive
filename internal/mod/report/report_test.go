package report

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ihsan-sa/hive-standalone-build/internal/core"
)

// gen makes a token body of n characters from alphabet, so no key-shaped
// literal sits in the source (push protection refuses those, fake or not).
func gen(n int, alphabet string) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = alphabet[(i*7+3)%len(alphabet)]
	}
	return string(b)
}

const (
	alnum = "aB3cD4eF5gH6iJ7kL8mN9oP0qR1sT2uVwXyZ"
	upper = "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567"
)

// Each line holds a secret (the second field) that must not survive Scrub.
var leaks = func() []struct{ in, secret string } {
	tok := func(prefix string, n int) string { return prefix + gen(n, alnum) }
	ghp, gho, pat := tok("gh"+"p_", 36), tok("gh"+"o_", 36), tok("github_"+"pat_11", 70)
	ant, oat, oai := tok("sk"+"-ant-api03-", 32), tok("sk"+"-ant-oat01-", 40), tok("sk"+"-proj-", 30)
	stripe, slack := tok("sk"+"_live_", 24), "xo"+"xb-2048-1234567890-"+gen(24, alnum)
	aws, awsSecret := "AK"+"IA"+gen(16, upper), gen(16, alnum)+"/K7MDENG/"+gen(16, alnum)
	google, gitlab := tok("AI"+"za", 35), tok("gl"+"pat-", 20)
	npm, hf := tok("np"+"m_", 36), tok("h"+"f_", 34)
	jwtSig := gen(43, alnum)
	return []struct{ in, secret string }{
		{"push failed: remote: Invalid username or token " + ghp, ghp},
		{"GH_TOKEN=" + gho + " gh pr create", gho},
		{"using " + pat, pat[:40]},
		{"ANTHROPIC_API_KEY: " + ant + "-jK2lQwErTy_AA", ant},
		{"CLAUDE_CODE_OAUTH_TOKEN=" + oat, oat[:32]},
		{"openai error for key " + oai, oai},
		{"stripe: " + stripe, stripe},
		{"slack " + slack + " said 401", slack},
		{"aws_access_key_id = " + aws, aws},
		{"export AWS_SECRET_ACCESS_KEY=" + awsSecret, awsSecret},
		{"maps key " + google + " in config", google},
		{"gitlab " + gitlab, gitlab},
		{"//registry.npmjs.org/:_authToken=" + npm, npm},
		{"HF " + hf, hf},
		{"Authorization: Bearer eyJ" + gen(20, alnum) + ".eyJ" + gen(24, alnum) + "." + jwtSig, jwtSig},
		{"Authorization: Basic dXNlcjpodW50ZXIy", "dXNlcjpodW50ZXIy"},
		{"curl -H 'Authorization: token 7f3a9c2e1b8d4f6a0e5c9b2d7a1f8e3c' api", "7f3a9c2e1b8d4f6a0e5c9b2d7a1f8e3c"},
		{"fatal: could not read from https://ihsan:hunter2pass@github.com/me/repo.git", "hunter2pass"},
		{"DATABASE_URL=postgres://admin:S3cr3tP%40ss@db.internal:5432/app", "S3cr3tP%40ss"},
		{`{"password": "correct-horse-battery", "user": "x"}`, "correct-horse-battery"},
		{"db_password=Tr0ub4dor&3", "Tr0ub4dor&3"},
		{"client_secret: 'q8w7e6r5t4y3'", "q8w7e6r5t4y3"},
		{"Set-Cookie: session=4f9a1c7e2b; Path=/", "4f9a1c7e2b"},
		{"api-key: 0123abcd", "0123abcd"},
		{"-----BEGIN OPENSSH PRIVATE KEY-----\nb3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQ\nAAAAMwAAAAtzc2gtZW\n-----END OPENSSH PRIVATE KEY-----", "AAAAMwAAAAtzc2gtZW"},
		{"-----BEGIN RSA PRIVATE KEY-----\nMIIEpAIBAAKCAQEA7bq9 (clipped", "MIIEpAIBAAKCAQEA7bq9"},
		{"commit by Ihsan Salari <ihsan.salari@example.com>", "ihsan.salari@example.com"},
		{"mail me at a.b+hive@mail.co.uk please", "a.b+hive@mail.co.uk"},
		{"open /home/ihsan/hive/projects/x/main.go: no such file", "ihsan"},
		{"stat /Users/jane.doe/Library/hive: denied", "jane.doe"},
		{`C:\Users\Marta\hive\.hive\runs\x failed`, "Marta"},
		{"token at /root/.config/gh/hosts.yml", "/root"},
		{"daemon token 9b1e4c7a2f6d8e3b5a0c9f1e7d2b4a6c in daemon.json", "9b1e4c7a2f6d8e3b5a0c9f1e7d2b4a6c"},
		{"random blob Zm9vYmFyYmF6cXV4MTIzNDU2Nzg5MGFiY2RlZmdo end", "Zm9vYmFyYmF6cXV4MTIzNDU2Nzg5MGFiY2RlZmdo"},
	}
}()

func TestScrubLeavesNoSecret(t *testing.T) {
	for _, c := range leaks {
		if out := Scrub(c.in, ""); strings.Contains(out, c.secret) {
			t.Errorf("secret survived:\n  in:  %s\n  out: %s", c.in, out)
		}
	}
}

func TestScrubHomeAndRoot(t *testing.T) {
	home := core.HomeDir()
	out := Scrub("open "+filepath.Join(home, "x")+" and /srv/hive/.hive/events.jsonl", "/srv/hive")
	if strings.Contains(out, home) || strings.Contains(out, "/srv/hive") || !strings.Contains(out, "<hive>/.hive/events.jsonl") {
		t.Fatalf("home or Hive folder left: %s", out)
	}
}

// What a person needs to read a failure survives.
func TestScrubKeepsPlainText(t *testing.T) {
	for _, s := range []string{
		"stopped after 3 rounds of fixes; what remains: TestParseHeader fails",
		"the worker finished without any commits",
		"go test ./internal/verylongpackagenamewithoutdigitsatall/...",
		"exit status 1",
	} {
		if out := Scrub(s, ""); out != s {
			t.Errorf("changed plain text:\n  in:  %s\n  out: %s", s, out)
		}
	}
}

// open makes a core on a fresh workspace with the given config.
func open(t *testing.T, edit func(*core.Config)) *core.Core {
	t.Helper()
	ws := core.Workspace{Root: t.TempDir()}
	if err := ws.Init(); err != nil {
		t.Fatal(err)
	}
	cfg := ws.LoadConfig()
	edit(&cfg)
	ws.SaveConfig(cfg)
	c, err := core.Open(ws, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.Close() })
	return c
}

func TestOffSendsNothing(t *testing.T) {
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { hits++ }))
	defer srv.Close()
	c := open(t, func(cfg *core.Config) { cfg.ShareURL = srv.URL })
	c.J.MustAppend("task.saved", map[string]any{"id": "t1", "stage": "blocked", "reason": "no"})
	if res, _ := (&Module{C: c}).Send(); res != "sharing is off" || hits != 0 || len(Sent(c.WS)) != 0 {
		t.Fatalf("off: %q, %d requests, %d copies", res, hits, len(Sent(c.WS)))
	}
}

func TestSendOnce(t *testing.T) {
	var got [][]byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		got = append(got, raw)
	}))
	defer srv.Close()
	c := open(t, func(cfg *core.Config) { cfg.ShareURL = srv.URL })
	c.J.MustAppend("task.saved", map[string]any{"id": "old", "stage": "blocked", "reason": "before sharing was on"})
	time.Sleep(1100 * time.Millisecond) // Share has second resolution
	cfg := c.WS.LoadConfig()
	cfg.Share = time.Now().UTC().Format(time.RFC3339)
	c.WS.SaveConfig(cfg)
	c.J.MustAppend("run.created", map[string]any{"id": "w1", "role": "worker"})
	c.J.MustAppend("run.turn_ended", map[string]any{"id": "w1", "status": "parked", "reason": "turn failed", "result": map[string]any{"text": "boom with " + leaks[0].secret, "is_error": true, "cost_usd": 0.5}})
	c.J.MustAppend("task.saved", map[string]any{"id": "t1", "stage": "working"})
	c.J.MustAppend("task.saved", map[string]any{"id": "t1", "stage": "stopped", "reason": "red tests", "log": []map[string]string{{"text": "tests failed"}}})
	c.J.MustAppend("task.saved", map[string]any{"id": "t1", "stage": "stopped", "reason": "red tests"})
	m := &Module{C: c}
	if res, _ := m.Send(); len(got) != 1 {
		t.Fatalf("want one batch, got %d (%s)", len(got), res)
	}
	var p Payload
	if err := json.Unmarshal(got[0], &p); err != nil {
		t.Fatal(err)
	}
	if p.Schema != 1 || p.Install == "" || p.Usage.Runs != 1 || p.Usage.Turns != 1 || p.Usage.CostUSD != 0.5 || len(p.Failures) != 2 {
		t.Fatalf("payload: %s", got[0])
	}
	if p.Failures[0].Kind != "run-error" || p.Failures[0].Role != "worker" || p.Failures[1].Kind != "task-stopped" || p.Failures[1].Excerpt != "tests failed" {
		t.Fatalf("failures: %+v", p.Failures)
	}
	if strings.Contains(string(got[0]), leaks[0].secret) || strings.Contains(string(got[0]), "before sharing") {
		t.Fatalf("a secret or an event from before sharing went out: %s", got[0])
	}
	if sent := Sent(c.WS); len(sent) != 1 {
		t.Fatalf("copies: %v", sent)
	} else if raw, _ := os.ReadFile(sent[0]); string(raw) != string(got[0]) {
		t.Fatal("the copy differs from what was sent")
	}
	if res, _ := m.Send(); res != "nothing new" || len(got) != 1 {
		t.Fatalf("second send: %q, %d batches", res, len(got))
	}
}

func TestUnreachableKeepsTheBatch(t *testing.T) {
	c := open(t, func(cfg *core.Config) {
		cfg.ShareURL, cfg.Share = "http://127.0.0.1:1/report", time.Now().Add(-time.Minute).UTC().Format(time.RFC3339)
	})
	c.J.MustAppend("alert.raised", map[string]string{"kind": "quiet", "text": "x"})
	Note(c.WS, Failure{Kind: "doctor", Text: "✗ sandbox: no userns"})
	m := &Module{C: c}
	if res, err := m.Send(); err != nil || !strings.Contains(res, "try again") || len(Sent(c.WS)) != 0 {
		t.Fatalf("unreachable: %q %v", res, err)
	}
	if p, _ := m.Build(c.WS.LoadConfig()); len(p.Failures) != 2 {
		t.Fatalf("the batch should still hold the alert and the doctor failure: %+v", p.Failures)
	}
}

func TestSizeCap(t *testing.T) {
	p := Payload{}
	for i := 0; i < maxFailures; i++ {
		p.Failures = append(p.Failures, Failure{Text: strings.Repeat("x", maxText), Excerpt: strings.Repeat("y", maxExcerpt)})
	}
	if raw := encode(&p); len(raw) > maxBytes || len(p.Failures) == 0 {
		t.Fatalf("%d bytes, %d failures", len(raw), len(p.Failures))
	}
}
