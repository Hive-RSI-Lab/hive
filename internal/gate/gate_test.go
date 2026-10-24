package gate

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func res(probe string, pass bool, ms ...Metric) Result { return Result{Probe: probe, Pass: pass, Metrics: ms} }

func TestDecide(t *testing.T) {
	score := func(v float64) Metric { return Metric{"score", v, true} }
	secs := func(v float64) Metric { return Metric{"seconds", v, false} }
	cases := []struct {
		name       string
		cand, ctrl []Result
		opt        Options
		keep       bool
		reason     string
	}{
		{"better score", []Result{res("bench", true, score(8))}, []Result{res("bench", true, score(5))}, Options{}, true, "better"},
		{"same", []Result{res("bench", true, score(5))}, []Result{res("bench", true, score(5))}, Options{}, false, "not better"},
		{"same, ties allowed", []Result{res("bench", true, score(5))}, []Result{res("bench", true, score(5))}, Options{AllowTie: true}, true, "ties"},
		{"worse score", []Result{res("bench", true, score(3))}, []Result{res("bench", true, score(5))}, Options{}, false, "worse"},
		{"probe regresses though the score gains", []Result{res("e2e", false), res("bench", true, score(9))}, []Result{res("e2e", true), res("bench", true, score(5))}, Options{}, false, "e2e passes 0/1"},
		{"candidate fixes a probe", []Result{res("e2e", true)}, []Result{res("e2e", false)}, Options{}, true, "e2e passes 1/1"},
		{"slower is worse", []Result{res("bench", true, secs(130))}, []Result{res("bench", true, secs(100))}, Options{}, false, "seconds"},
		{"faster is better", []Result{res("bench", true, secs(60))}, []Result{res("bench", true, secs(100))}, Options{}, true, "seconds"},
		{"inside the floor is nothing", []Result{res("bench", true, secs(101))}, []Result{res("bench", true, secs(100))}, Options{}, false, "not better"},
		{"candidate missing a probe", nil, []Result{res("e2e", true)}, Options{}, false, "did not run"},
	}
	for _, c := range cases {
		v := Decide(c.cand, c.ctrl, c.opt)
		if v.Keep != c.keep || !strings.Contains(v.Reason+strings.Join(v.Gains, "|")+strings.Join(v.Regressions, "|"), c.reason) {
			t.Errorf("%s: keep=%v %q gains=%v regs=%v", c.name, v.Keep, v.Reason, v.Gains, v.Regressions)
		}
	}
}

// Noise decides: the same 5% gain is real when the runs agree and nothing when they scatter.
func TestDecideNoise(t *testing.T) {
	mk := func(vs ...float64) []Result {
		var rs []Result
		for _, v := range vs {
			rs = append(rs, res("bench", true, Metric{"score", v, true}))
		}
		return rs
	}
	if v := Decide(mk(10.5, 10.5, 10.5), mk(10, 10, 10), Options{}); !v.Keep {
		t.Errorf("steady 5%% gain not kept: %v", v)
	}
	if v := Decide(mk(8, 13, 10.5), mk(10, 7, 13), Options{}); v.Keep {
		t.Errorf("a 5%% gain inside the scatter was kept: %v", v)
	}
}

func fakeBin(t *testing.T, dir, name, version string, ok bool) string {
	p := filepath.Join(dir, name)
	body := "#!/bin/sh\necho " + version + "\n"
	if !ok {
		body = "#!/bin/sh\necho broken >&2\nexit 3\n"
	}
	if err := os.WriteFile(p, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestInstallRollsBack(t *testing.T) {
	dir := t.TempDir()
	target := fakeBin(t, dir, "hive", "v1", true)
	if err := Install(fakeBin(t, dir, "good", "v2", true), target, "v2"); err != nil {
		t.Fatal(err)
	}
	if out, _ := exec.Command(target).Output(); strings.TrimSpace(string(out)) != "v2" {
		t.Fatalf("not installed: %s", out)
	}
	if out, _ := exec.Command(target + ".prev").Output(); strings.TrimSpace(string(out)) != "v1" {
		t.Fatalf("previous not kept: %s", out)
	}
	err := Install(fakeBin(t, dir, "bad", "v3", false), target, "v3")
	if err == nil || !strings.Contains(err.Error(), "rolled back") {
		t.Fatalf("a broken install was accepted: %v", err)
	}
	if out, _ := exec.Command(target).Output(); strings.TrimSpace(string(out)) != "v2" {
		t.Fatalf("not rolled back to v2: %s", out)
	}
	if err := Install(fakeBin(t, dir, "wrong", "v9", true), target, "v4"); err == nil {
		t.Fatal("a binary reporting the wrong version was accepted")
	}
}

// The whole run on stand-in binaries and a local org: the candidate that wins is kept,
// the one that loses is rolled back with the lesson recorded, and the org repos are gone.
func TestRunKeepsAndRollsBack(t *testing.T) {
	dir := t.TempDir()
	good, bad := fakeBin(t, dir, "good", "v2", true), fakeBin(t, dir, "bad", "v2", true)
	probe := func(score map[string]float64) ProbeFunc {
		return func(cfg *Config, c *Child) Result {
			return Result{Pass: true, Metrics: []Metric{{"score", score[filepath.Base(c.Bin)], true}}}
		}
	}
	ctrl := fakeBin(t, dir, "ctrl", "v1", true)
	run := func(cand string, scores map[string]float64) (Report, string) {
		org := filepath.Join(dir, "org-"+filepath.Base(cand))
		lessons := filepath.Join(dir, "lessons-"+filepath.Base(cand), ".hive", "lessons.md")
		rep, err := Run(Config{CandBin: cand, CtrlBin: ctrl, Out: filepath.Join(dir, "out-"+filepath.Base(cand)+"-x", "run"), Reps: 2, Sandbox: "bare",
			Probes: []string{"org", "score"}, Probe: map[string]ProbeFunc{"score": probe(scores)}, Org: Local{Dir: org}, Lessons: lessons})
		if err != nil {
			t.Fatal(err)
		}
		if len(rep.Repos) != 2 || len(rep.Cleaned) != 2 || len(rep.Leaked) != 0 {
			t.Errorf("org repos: made %v cleaned %v leaked %v", rep.Repos, rep.Cleaned, rep.Leaked)
		}
		if ents, _ := os.ReadDir(org); len(ents) != 0 {
			t.Errorf("repos left in the org: %v", ents)
		}
		raw, _ := os.ReadFile(lessons)
		return rep, string(raw)
	}
	scores := map[string]float64{"good": 9, "bad": 2, "ctrl": 5}
	if rep, lessons := run(good, scores); !rep.Verdict.Keep || lessons != "" {
		t.Errorf("good candidate: %+v lessons %q", rep.Verdict, lessons)
	}
	rep, lessons := run(bad, scores)
	if rep.Verdict.Keep || !strings.Contains(lessons, "rolled back") || !strings.Contains(lessons, "score") {
		t.Errorf("bad candidate: %+v lessons %q", rep.Verdict, lessons)
	}
}
