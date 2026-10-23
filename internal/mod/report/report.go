// Package report shares an install's failures and usage with the Hive's
// developer, only when the person has said yes. It reads the journal, scrubs
// secrets on this machine, sends one batch to the configured address and keeps
// a copy of every batch it sent. The payload is docs/report.md.
package report

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/ihsan-sa/hive-standalone-build/internal/core"
)

// DefaultURL receives reports unless the config names another.
const DefaultURL = "https://hive.ihsan.cc/api/report"

// Size caps, as docs/report.md states them.
const (
	Schema      = 1
	maxFailures = 50
	maxText     = 2000
	maxExcerpt  = 4000
	maxBytes    = 256 << 10
)

// Payload is one batch.
type Payload struct {
	Schema   int       `json:"schema"`
	Install  string    `json:"install_id"`
	Sent     time.Time `json:"sent_at"`
	From     time.Time `json:"from"`
	To       time.Time `json:"to"`
	Version  string    `json:"version"`
	OS       string    `json:"os"`
	Arch     string    `json:"arch"`
	Harness  string    `json:"harness"`
	Usage    Usage     `json:"usage"`
	Failures []Failure `json:"failures"`
}

// Usage counts what happened in the batch's period.
type Usage struct {
	Runs    int     `json:"runs"`
	Turns   int     `json:"turns"`
	Merges  int     `json:"merges"`
	Reverts int     `json:"reverts"`
	CostUSD float64 `json:"cost_usd"`
}

// Failure is one thing that went wrong.
type Failure struct {
	Kind    string    `json:"kind"` // task-blocked | task-stopped | run-error | alert | crash | doctor
	At      time.Time `json:"at"`
	Role    string    `json:"role,omitempty"`
	Text    string    `json:"text"`
	Excerpt string    `json:"excerpt,omitempty"`
}

// Module sends reports.
type Module struct{ C *core.Core }

// Dir holds the copies of what was sent and doctor failures still to send.
func Dir(ws core.Workspace) string { return filepath.Join(ws.Hive(), "reports") }

func outbox(ws core.Workspace) string { return filepath.Join(Dir(ws), "outbox.jsonl") }

// Note keeps a failure seen outside the daemon (the doctor) for the next batch.
func Note(ws core.Workspace, f Failure) {
	if _, on := ws.LoadConfig().Sharing(); on {
		f.At = time.Now().UTC()
		core.AppendJSONL(outbox(ws), f)
	}
}

// URL is where reports go.
func URL(cfg core.Config) string {
	if cfg.ShareURL != "" {
		return cfg.ShareURL
	}
	return DefaultURL
}

// Send builds and sends one batch; it is the report.send job. It never fails
// the Hive: an unreachable address leaves the batch for next time.
func (m *Module) Send() (string, error) {
	cfg := m.C.WS.LoadConfig()
	if _, on := cfg.Sharing(); !on {
		return "sharing is off", nil
	}
	if cfg.InstallID == "" {
		return "sharing has no install id; `" + core.Cmd + " report on` makes one", nil
	}
	p, seq, read := m.Build(cfg)
	if p.Usage == (Usage{}) && len(p.Failures) == 0 {
		return "nothing new", nil
	}
	raw := encode(&p)
	req, err := http.NewRequest("POST", URL(cfg), bytes.NewReader(raw))
	if err != nil {
		return "", fmt.Errorf("share_url: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "hive/"+core.Version)
	// The copy is kept before the batch leaves, so nothing goes out unrecorded.
	name := filepath.Join(Dir(m.C.WS), "sent", core.NewID()+".json")
	if err := core.WriteFileAtomic(name, raw, 0o600); err != nil {
		return "", fmt.Errorf("keeping a copy of the batch: %v", err)
	}
	res := "couldn't reach " + URL(cfg) + "; will try again"
	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err == nil {
		resp.Body.Close()
		res = fmt.Sprintf("%s answered %d; will try again", URL(cfg), resp.StatusCode)
	}
	if err != nil || resp.StatusCode/100 != 2 {
		os.Remove(name)
		return res, nil
	}
	trim(outbox(m.C.WS), read)
	m.C.J.MustAppend("report.sent", map[string]any{"seq": seq, "file": filepath.Base(name), "failures": len(p.Failures)})
	return fmt.Sprintf("sent %d failures", len(p.Failures)), nil
}

// trim drops the first n bytes of the outbox, the part a batch carried, and
// keeps what was noted since.
func trim(path string, n int) {
	raw, _ := os.ReadFile(path)
	if len(raw) <= n {
		os.Remove(path)
		return
	}
	core.WriteFileAtomic(path, raw[n:], 0o600)
}

// encode marshals p, dropping the newest failures until it fits the cap.
func encode(p *Payload) []byte {
	for {
		raw, _ := json.Marshal(p)
		if len(raw) <= maxBytes || len(p.Failures) == 0 {
			return raw
		}
		p.Failures = p.Failures[:len(p.Failures)-1]
	}
}

// Build reads the journal since the last batch, from when sharing was turned
// on, and returns the batch, the last event seq it covers and how many bytes
// of the outbox it read.
func (m *Module) Build(cfg core.Config) (Payload, int64, int) {
	since, _ := cfg.Sharing()
	p := Payload{Schema: Schema, Install: cfg.InstallID, Sent: time.Now().UTC(), Version: core.Version, OS: runtime.GOOS, Arch: runtime.GOARCH, Harness: cfg.Harness, Failures: []Failure{}}
	var last, seq int64
	roles := map[string]string{}
	stages := map[string]string{}
	running := false
	var events []core.Event
	m.C.J.Replay(func(e core.Event) {
		if e.Kind == "report.sent" {
			var s struct{ Seq int64 }
			e.Decode(&s)
			last = s.Seq
		}
		events = append(events, e)
	})
	root := m.C.WS.Root
	add := func(at time.Time, kind, role, text, excerpt string) {
		p.Failures = append(p.Failures, Failure{Kind: kind, At: at, Role: role, Text: core.Clip(Scrub(text, root), maxText), Excerpt: core.Clip(Scrub(excerpt, root), maxExcerpt)})
	}
	for _, e := range events {
		seq = e.Seq
		fresh := e.Seq > last && !e.At.Before(since)
		if fresh && p.From.IsZero() {
			p.From = e.At
		}
		if fresh {
			p.To = e.At
		}
		switch e.Kind {
		case "run.created":
			var r core.RunSpec
			e.Decode(&r)
			roles[r.ID] = r.Role
			if fresh {
				p.Usage.Runs++
			}
		case "run.turn_ended":
			var t struct {
				ID, Status, Reason string
				Result             *struct {
					Text    string
					IsError bool    `json:"is_error"`
					Limited bool    `json:"limited"`
					Cost    float64 `json:"cost_usd"`
				}
			}
			e.Decode(&t)
			if !fresh {
				continue
			}
			p.Usage.Turns++
			if t.Result != nil {
				p.Usage.CostUSD += t.Result.Cost
			}
			if t.Status == core.StatusParked && (t.Result == nil || t.Result.IsError && !t.Result.Limited) {
				text := ""
				if t.Result != nil {
					text = t.Result.Text
				}
				add(e.At, "run-error", roles[t.ID], t.Reason, text)
			}
		case "task.saved":
			var t struct {
				ID, Kind, Stage, Reason string
				Log                     []struct{ Text string }
			}
			e.Decode(&t)
			prev := stages[t.ID]
			stages[t.ID] = t.Stage
			if !fresh || prev == t.Stage {
				continue
			}
			switch t.Stage {
			case "merged":
				p.Usage.Merges++
				if t.Kind == "revert" {
					p.Usage.Reverts++
				}
			case "blocked", "stopped":
				var log []string
				for _, l := range t.Log[max(0, len(t.Log)-8):] {
					log = append(log, l.Text)
				}
				add(e.At, "task-"+t.Stage, "", t.Reason, strings.Join(log, "\n"))
			}
		case "alert.raised":
			var a map[string]string
			e.Decode(&a)
			if fresh {
				add(e.At, "alert", "", a["kind"]+": "+a["text"], "")
			}
		case "hive.started":
			if running && fresh {
				add(e.At, "crash", "", "the Hive stopped without shutting down", "")
			}
			running = true
		case "hive.stopped":
			running = false
		}
	}
	box, _ := os.ReadFile(outbox(m.C.WS))
	box = box[:bytes.LastIndexByte(box, '\n')+1] // a line still being written waits
	for _, line := range bytes.Split(box, []byte("\n")) {
		var f Failure
		if json.Unmarshal(line, &f) == nil {
			add(f.At, f.Kind, "", f.Text, f.Excerpt)
		}
	}
	sort.SliceStable(p.Failures, func(i, j int) bool { return p.Failures[i].At.Before(p.Failures[j].At) })
	if len(p.Failures) > maxFailures {
		p.Failures = p.Failures[len(p.Failures)-maxFailures:]
	}
	return p, seq, len(box)
}

// Sent lists the copies of sent batches, oldest first.
func Sent(ws core.Workspace) []string {
	files, _ := filepath.Glob(filepath.Join(Dir(ws), "sent", "*.json"))
	sort.Strings(files)
	return files
}
