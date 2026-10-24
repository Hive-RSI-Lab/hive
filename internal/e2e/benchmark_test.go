package e2e

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/ihsan-sa/hive-standalone-build/internal/api"
	"github.com/ihsan-sa/hive-standalone-build/internal/mod/work"
)

// benchAsks is the benchmark's fixed set of asks on the demo textkit project:
// one per function the demo leaves unwritten.
var benchAsks = []string{
	"In textkit, implement capitalize_words in textkit/__init__.py.",
	"In textkit, implement is_palindrome in textkit/__init__.py.",
	"In textkit, implement char_frequency in textkit/__init__.py.",
	"In textkit, implement truncate in textkit/__init__.py.",
}

// TestBenchmark is what `hive gate` measures: the fixed asks on the scored demo
// project, one after another, then the project's score, how many asks were
// finished, the wall time and what the agents spent. It prints one BENCH line
// of JSON. It runs only when HIVE_BENCH is set (to the number of asks, or 1
// for all), because on the real harness it spends plan usage.
func TestBenchmark(t *testing.T) {
	n, _ := strconv.Atoi(os.Getenv("HIVE_BENCH"))
	if n <= 0 {
		t.Skip("set HIVE_BENCH=<asks> (4 runs the whole set) to run the benchmark")
	}
	n = min(n, len(benchAsks))
	h := start(t)
	h.post("/api/project", map[string]any{"demo": true}, nil)
	best := func(st api.State) float64 {
		for _, p := range st.Projects {
			if p.Name == "textkit" && p.Best != nil {
				return *p.Best
			}
		}
		return 0
	}
	h.wait("the baseline", 5*minute, func(st api.State) bool { return best(st) > 0 || len(st.Projects) > 0 && len(st.Projects[0].Scores) > 0 })
	start := time.Now()
	finished := 0
	for _, ask := range benchAsks[:n] {
		h.post("/api/ask", map[string]string{"project": "textkit", "title": ask, "known": ask}, nil)
		st := h.wait("the ask to finish", patience(t), func(st api.State) bool {
			tk, ok := task(st, "textkit", ask)
			return ok && (tk.Stage == work.StageDone || tk.Stage == work.StageBlocked || tk.Stage == work.StageStopped)
		})
		if tk, _ := task(st, "textkit", ask); tk.Stage == work.StageDone {
			finished++
		}
	}
	st := h.state()
	cost := 0.0
	for _, r := range st.Runs {
		cost += r.Cost
	}
	out, _ := json.Marshal(map[string]any{"score": best(st), "asks_finished": finished, "asks": n, "seconds": time.Since(start).Seconds(), "cost_usd": cost})
	fmt.Println("BENCH " + string(out))
	if finished < n {
		t.Errorf("%d of %d asks finished", finished, n)
	}
}
