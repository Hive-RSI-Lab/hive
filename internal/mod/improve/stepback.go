package improve

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ihsan-sa/hive-standalone-build/internal/core"
	"github.com/ihsan-sa/hive-standalone-build/internal/mod/work"
)

// A step-back asks whether a project's whole approach is right and what the
// simplest setup is that might beat it, then runs the setup the agent designs
// in a copy of the project and scores it. A win is a fault in the Hive's own
// setup, filed with both scores and what ran.

// Setup is the experiment a step-back designs: one harness session in a copy
// of the project at its start, with System appended to the harness's own
// prompt and Asks sent in turn. Nothing else is added.
type Setup struct {
	Why    string   `json:"why"`
	System string   `json:"system"`
	Asks   []string `json:"asks"`
	Model  string   `json:"model"`
}

// StepBack is one step-back on one project.
type StepBack struct {
	ID      string    `json:"id"`
	Project string    `json:"project"`
	Why     string    `json:"why"`
	Run     string    `json:"run"`           // the agent that steps back
	Exp     string    `json:"exp,omitempty"` // the experiment it designed
	Setup   *Setup    `json:"setup,omitempty"`
	Asked   int       `json:"asked,omitempty"`
	From    string    `json:"from"` // the project's start
	Best    float64   `json:"best"` // the Hive's best kept score when it began
	Budget  float64   `json:"budget,omitempty"`
	Score   *float64  `json:"score,omitempty"`
	Status  string    `json:"status"` // thinking | experimenting | scoring | won | lost | none | failed
	Note    string    `json:"note,omitempty"`
	Fault   string    `json:"fault,omitempty"` // the task or person's ask it filed
	Started time.Time `json:"started"`
	Ended   time.Time `json:"ended,omitempty"`
}

func (s StepBack) active() bool {
	return s.Status == "thinking" || s.Status == "experimenting" || s.Status == "scoring"
}

// StepBacks returns a project's step-backs ("" for all), oldest first.
func (m *Module) StepBacks(project string) []StepBack {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []StepBack
	for _, s := range m.stepbacks {
		if project == "" || s.Project == project {
			out = append(out, *s)
		}
	}
	return out
}

func (m *Module) saveStepBack(s StepBack) { m.C.J.MustAppend("stepback.saved", s) }

// start is the project's first kept measure: the seed every setup starts from.
func (m *Module) start(project string) (Measurement, bool) {
	for _, x := range m.Scores(project) {
		if x.Kept && x.Score != nil {
			return x, true
		}
	}
	return Measurement{}, false
}

// RotateStepBacks starts a step-back on each scored project with new
// measures since its last one (the improve.stepback schedule).
func (m *Module) RotateStepBacks() string {
	var started []string
	for _, p := range m.C.Projects.List() {
		ms, last := m.Scores(p.Name), m.lastStepBack(p.Name)
		if len(ms) > 0 && ms[len(ms)-1].At.After(last) {
			if err := m.StartStepBack(p, "it is the regular step-back"); err == nil {
				started = append(started, p.Name)
			}
		}
	}
	return "stepped back on: " + strings.Join(started, ", ")
}

func (m *Module) lastStepBack(project string) time.Time {
	var t time.Time
	for _, s := range m.StepBacks(project) {
		if s.Started.After(t) {
			t = s.Started
		}
	}
	return t
}

// plateau counts the merges measured since the best kept score last moved
// and since the last step-back.
func (m *Module) plateau(project string) int {
	n, since := 0, m.lastStepBack(project)
	var best *float64
	for _, x := range m.Scores(project) {
		if !x.Kept || x.Best == nil {
			continue
		}
		if best == nil || *x.Best != *best {
			n = 0
		} else if x.Task != "" && x.At.After(since) {
			n++
		}
		best = x.Best
	}
	return n
}

// StartStepBack starts the agent that steps back on p.
func (m *Module) StartStepBack(p core.Project, why string) error {
	for _, s := range m.StepBacks(p.Name) {
		if s.active() {
			return fmt.Errorf("a step-back on %s is already running", p.Name)
		}
	}
	st, ok := m.start(p.Name)
	best, _ := m.Best(p.Name)
	if !ok {
		return fmt.Errorf("%s has no score to beat", p.Name)
	}
	cfg := m.C.Effects.Config(p, st.SHA)
	id := "stepback-" + core.NewID()
	dir, err := m.Agents.Scratch(id)
	if err != nil {
		return err
	}
	h, model := m.Agents.Model("worker", p.Name)
	better := "higher"
	if cfg.Score != nil && cfg.Score.Direction == "lower" {
		better = "lower"
	}
	brief := fmt.Sprintf(`Step back on project %s: %s.

The person's goal, in their words: %s

Its score (%s is better) started at %g, and the best the Hive has kept is %g. The records are in $HIVE_RUN/digest.md, the project is in $HIVE_PROJECTS/%s, and the Hive's own prompts and modules are in $HIVE_MODULES.

Ask two questions: is the whole approach right, and what is the simplest setup that might beat it?

If a setup is worth testing, write it to $HIVE_BOX/experiment.json as {"why": "...", "system": "...", "asks": ["..."], "model": "..."}. The Hive runs it once in a fresh copy of the project at its start, in a worker's sandbox: one %s session, "system" appended to the harness's own system prompt (empty for none), the "asks" sent in turn in that session, on "model" (empty for %s). Nothing else is added. It may spend what the Hive has spent on this project so far. The copy is then scored with the project's score command, and if it beats %g the Hive files a fault against its own setup with both scores.

Write your reasoning to $HIVE_BOX/sweep.md, then finish.`, p.Name, why, p.Goals, better, *st.Score, best, p.Name, h, model, best)
	spec := m.Agents.Spec(id, "stepback", "", dir, brief)
	spec.Inputs = map[string]string{"digest.md": m.Digest(0)}
	if _, err := m.C.Runs.Start(spec); err != nil {
		return err
	}
	m.saveStepBack(StepBack{ID: id, Project: p.Name, Why: why, Run: id, From: st.SHA, Best: best, Status: "thinking", Started: time.Now().UTC()})
	return nil
}

// TickStepBacks starts a step-back where the score has plateaued and moves
// each running one on.
func (m *Module) TickStepBacks() {
	if n := m.C.WS.LoadConfig().PlateauMerges; n > 0 {
		for _, p := range m.C.Projects.List() {
			if k := m.plateau(p.Name); k >= n {
				m.StartStepBack(p, fmt.Sprintf("its best score has not moved in %d merges", k))
			}
		}
	}
	for _, s := range m.StepBacks("") {
		if !s.active() {
			continue
		}
		m.mu.Lock()
		busy := m.busy[s.ID]
		m.busy[s.ID] = true
		m.mu.Unlock()
		if busy {
			continue
		}
		go func() {
			m.advance(s)
			m.mu.Lock()
			delete(m.busy, s.ID)
			m.mu.Unlock()
		}()
	}
}

func (m *Module) advance(s StepBack) {
	run := s.Run
	if s.Status != "thinking" {
		run = s.Exp
	}
	r, ok := m.C.Runs.Get(run)
	if !ok || (!r.Terminal() && s.Status != "scoring") {
		return
	}
	if r.Status != core.StatusDone {
		m.fail(s, fmt.Sprintf("the %s ended %s: %s", map[bool]string{true: "step-back", false: "experiment"}[run == s.Run], r.Status, r.Reason))
		return
	}
	switch s.Status {
	case "thinking":
		raw, err := core.SafeRead(filepath.Join(m.C.WS.RunBox(r.ID), "experiment.json"), 1<<20)
		if errors.Is(err, os.ErrNotExist) {
			s.Status, s.Note, s.Ended = "none", "it designed no experiment", time.Now().UTC()
			m.saveStepBack(s)
			return
		}
		var set Setup
		if err := json.Unmarshal(raw, &set); err != nil || len(set.Asks) == 0 {
			m.fail(s, fmt.Sprintf("its experiment.json is not a setup with at least one ask (%v)", err))
			return
		}
		m.experiment(s, set)
	case "experimenting":
		if s.Asked < len(s.Setup.Asks) && (s.Budget == 0 || r.Cost < s.Budget) {
			s.Asked++
			m.saveStepBack(s)
			m.C.Runs.Reopen(r.ID, s.Setup.Asks[s.Asked-1])
			return
		}
		s.Status = "scoring"
		m.saveStepBack(s)
		fallthrough
	case "scoring":
		m.score(s, r)
	}
}

// experiment starts the setup in a fresh copy of the project at its start,
// as a worker in a worker's sandbox, seeing no other project.
func (m *Module) experiment(s StepBack, set Setup) {
	p, ok := m.C.Projects.Get(s.Project)
	if !ok {
		m.fail(s, "the project is gone")
		return
	}
	id := s.ID + "-exp"
	dir, err := m.C.Effects.NewCheckout(p, id, s.From)
	if err != nil {
		m.fail(s, "could not make a copy of the project: "+err.Error())
		return
	}
	h, model := m.Agents.Model("worker", p.Name)
	if set.Model != "" {
		model = set.Model
	}
	for _, r := range m.C.Runs.List(func(r *core.Run) bool { return r.Project == p.Name && r.Name != "experiment" }) {
		s.Budget += r.Cost
	}
	if _, ok := m.C.Runs.Get(id); !ok {
		spec := core.RunSpec{ID: id, Role: "worker", Name: "experiment", Project: p.Name, Checkout: dir, Harness: h, Model: model,
			System: set.System, Brief: set.Asks[0], Isolated: true, OneTurn: true, Budget: s.Budget}
		if _, err := m.C.Runs.Start(spec); err != nil {
			m.fail(s, "could not start the experiment: "+err.Error())
			return
		}
	}
	s.Exp, s.Setup, s.Asked, s.Status = id, &set, 1, "experimenting"
	m.saveStepBack(s)
}

// score scores what the experiment left with the project's score command and
// files a fault when it beats the Hive's best.
func (m *Module) score(s StepBack, r core.Run) {
	p, ok := m.C.Projects.Get(s.Project)
	if !ok {
		m.fail(s, "the project is gone")
		return
	}
	main, err := m.C.Effects.Sync(p)
	if err != nil {
		return // tried again next tick
	}
	cfg := m.C.Effects.Config(p, main)
	if cfg.Score == nil {
		m.fail(s, "the project no longer has a score command")
		return
	}
	sha, err := m.C.Effects.Snapshot(p, r, s.From)
	if errors.Is(err, core.ErrNoChanges) {
		sha, err = s.From, nil
	}
	if err != nil {
		m.fail(s, "could not take out what the experiment left: "+err.Error())
		return
	}
	if ch, err := m.C.Effects.Classify(p, s.From, sha); err == nil && len(ch.Protected) > 0 {
		m.fail(s, "the experiment changed what scores it ("+strings.Join(ch.Protected, ", ")+"), so its score can't count")
		return
	}
	c := m.C.Effects.RunCheck(p, sha, "score", cfg.Score.Command)
	if !c.OK || c.Score == nil {
		m.fail(s, "the experiment's copy did not score: "+core.Clip(c.Output, 300))
		return
	}
	s.Score, s.Ended = c.Score, time.Now().UTC()
	s.Status, s.Note = "lost", fmt.Sprintf("the setup scored %g against the Hive's best %g", *c.Score, s.Best)
	if !cfg.Score.WithinTolerance(s.Best, *c.Score) {
		s.Status = "won"
		s.Fault = m.fileFault(s, p, r, *cfg.Score)
	}
	m.saveStepBack(s)
}

// fileFault writes the report and files it against the Hive's own source when
// the Hive has it as a project, or asks the person when it doesn't.
func (m *Module) fileFault(s StepBack, p core.Project, exp core.Run, sc core.Score) string {
	var hive []string
	var spent float64
	for _, r := range m.C.Runs.List(func(r *core.Run) bool { return r.Project == p.Name && r.Name != "experiment" }) {
		hive = append(hive, fmt.Sprintf("- %s %s ($%.2f): %s", r.Role, r.ID, r.Cost, filepath.Join(m.C.WS.RunHost(r.ID), "turn-*.jsonl")))
		spent += r.Cost
	}
	set, _ := json.MarshalIndent(s.Setup, "", "  ")
	title := fmt.Sprintf("The Hive's own setup lost to a simpler one on %s: %g against %g", p.Name, *s.Score, s.Best)
	report := fmt.Sprintf(`# %s

A step-back on %s (%s) asked whether the Hive's approach is right and what the simplest setup is that might beat it, and ran its answer.

From the project's start at %s, the setup scored %g. The best the Hive has kept is %g (%s is better, tolerance %g). The setup cost $%.2f; the Hive's runs on the project cost $%.2f.

The setup:

%s

The setup's transcript: %s

The Hive's transcripts:
%s
`, title, p.Name, s.Why, core.Short(s.From), *s.Score, s.Best, sc.Direction, sc.Tolerance, exp.Cost, spent, set,
		filepath.Join(m.C.WS.RunHost(exp.ID), "turn-*.jsonl"), strings.Join(hive, "\n"))
	path := filepath.Join(m.C.WS.Hive(), "improve", "stepbacks", s.ID+".md")
	os.MkdirAll(filepath.Dir(path), 0o755)
	os.WriteFile(path, []byte(report), 0o644)
	for _, self := range m.C.Projects.List() {
		if self.Self {
			t, err := m.Work.File(work.Brief{Project: self.Name, Title: title, From: s.Run, Known: report + "\nThe full report: " + path,
				Done: "the prompt or module that held the Hive's agents back is changed, from what the transcripts show the simpler setup did that they didn't"})
			if err == nil {
				return t.ID
			}
		}
	}
	return m.C.Person.Ask(core.Ask{Kind: "question", Ref: "stepback:" + s.ID, Project: p.Name, Title: title,
		Detail: report + "\nThe full report: " + path, Options: []string{"ok"}}).ID
}

// fail ends a step-back and says why in one alert per project that updates.
func (m *Module) fail(s StepBack, why string) {
	s.Status, s.Note, s.Ended = "failed", why, time.Now().UTC()
	m.saveStepBack(s)
	m.C.J.MustAppend("alert.raised", map[string]string{"kind": "stepback", "ref": "stepback:" + s.Project, "project": s.Project,
		"text": "the step-back on " + s.Project + " could not finish: " + why})
}
