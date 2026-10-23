package tui

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"github.com/ihsan-sa/hive-standalone-build/internal/api"
	"github.com/ihsan-sa/hive-standalone-build/internal/core"
	"github.com/ihsan-sa/hive-standalone-build/internal/harness"
	"github.com/ihsan-sa/hive-standalone-build/internal/mod/chat"
	"github.com/ihsan-sa/hive-standalone-build/internal/mod/improve"
	"github.com/ihsan-sa/hive-standalone-build/internal/mod/work"
)

func sample() api.State {
	now := time.Now()
	long := strings.Repeat("a very long line that would run off any screen ", 10)
	five, four := 5.0, 4.0
	return api.State{
		Version: "test", Workspace: "/home/x/hive", Sandbox: "wsl:Ubuntu",
		Projects: []api.ProjectView{{
			Project: core.Project{Name: "textkit", Repo: "me/textkit"},
			Goals:   "# Goals\n\n- " + long,
			Tasks: []work.Task{
				{ID: "t1", Title: "implement " + long, Stage: work.StageWorking, Worker: "w-t1", Updated: now, PR: core.PR{Number: 3, URL: "https://github.com/me/textkit/pull/3"}},
				{ID: "t2", Title: "done thing", Stage: work.StageDone, Updated: now},
			},
			Scores: []improve.Measurement{{SHA: "abcdef123456", Score: &four, Kept: true, At: now}, {SHA: "bcdef1234567", Score: &five, Kept: true, At: now, Note: long}},
			Best:   &five,
		}},
		Runs:    []core.Run{{RunSpec: core.RunSpec{ID: "w-t1", Role: "worker", Project: "textkit", Task: "t1", Brief: long}, Status: core.StatusRunning, Progress: long, Updated: now}},
		Plans:   []core.Plan{{Harness: "claude", Limit: harness.RateLimit{Status: "allowed_warning", Utilization: 0.89, Window: "seven_day", Windows: map[string]harness.Window{"five_hour": {Utilization: 0.38}}}}},
		Asks:    []core.Ask{{ID: "a1", Title: "Merge? " + long, Detail: long, Options: []string{"merge", "leave open"}, Asked: now}},
		Alerts:  []api.Alert{{At: now, Kind: "quiet", Text: long}},
		Modules: []api.ModuleView{{Name: "agents", Description: long, Source: "builtin", Release: "test", Projects: []string{"*"}}},
		Chats: []api.ChatView{
			{Chat: chat.Chat{ID: "lead", Title: "lead", Kind: "lead"}, Unread: 2, LeadStatus: "idle"},
			{Chat: chat.Chat{ID: "project:textkit", Title: "textkit", Kind: "project", Project: "textkit"}},
		},
		Config: core.DefaultConfig(),
	}
}

func testModel(w, h int) *model {
	m := &model{open: map[string]bool{}, closed: map[string]bool{}, msgs: map[string][]chat.Message{},
		runs: map[string]api.RunDetail{}, tasks: map[string]work.Task{}, input: textinput.New()}
	m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	m.Update(stateMsg{st: sample()})
	now := time.Now()
	long := strings.Repeat("words that wrap ", 40)
	m.msgs["lead"] = []chat.Message{
		{ID: "1", Thread: "1", From: "person", Name: "you", Text: "please do " + long, At: now},
		{ID: "2", Thread: "1", From: "lead-x", Name: "lead", Text: "Understood. " + long, At: now},
		{ID: "3", Thread: "1", From: "hive", Name: "hive", Text: "Merged: https://github.com/me/textkit/pull/3", At: now},
	}
	return m
}

func checkWidth(t *testing.T, view string, w, h int, where string) {
	t.Helper()
	lines := strings.Split(view, "\n")
	if len(lines) > h {
		t.Fatalf("%s: %d lines on a %d-line screen", where, len(lines), h)
	}
	for i, l := range lines {
		if ansi.StringWidth(l) > w {
			t.Fatalf("%s: line %d is %d wide on a %d-wide screen: %q", where, i, ansi.StringWidth(l), w, l)
		}
	}
}

func TestViewsFitTheScreen(t *testing.T) {
	keys := func(s ...string) []tea.KeyMsg {
		var out []tea.KeyMsg
		for _, k := range s {
			switch k {
			case "tab", "enter", "esc", "down", "up", "right", "left":
				out = append(out, tea.KeyMsg(map[string]tea.Key{
					"tab": {Type: tea.KeyTab}, "enter": {Type: tea.KeyEnter}, "esc": {Type: tea.KeyEsc},
					"down": {Type: tea.KeyDown}, "up": {Type: tea.KeyUp}, "right": {Type: tea.KeyRight}, "left": {Type: tea.KeyLeft},
				}[k]))
			default:
				out = append(out, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)})
			}
		}
		return out
	}
	for _, size := range [][2]int{{120, 40}, {80, 24}, {50, 16}} {
		w, h := size[0], size[1]
		m := testModel(w, h)
		checkWidth(t, m.View(), w, h, "chats")
		for _, k := range keys("right", "enter") {
			m.Update(k)
		}
		checkWidth(t, m.View(), w, h, "chat threads")
		m.Update(keys("esc")[0])
		for i, name := range []string{"dashboard", "waiting", "modules", "settings"} {
			m.Update(keys("tab")[0])
			checkWidth(t, m.View(), w, h, name)
			if i == 0 { // walk into the dashboard tree
				for _, k := range keys("down", "down", "right", "down", "right") {
					m.Update(k)
					checkWidth(t, m.View(), w, h, "dashboard nav")
				}
			}
		}
		m.Update(keys("?")[0])
		checkWidth(t, m.View(), w, h, "help")
	}
}

func TestSchedulesAreHours(t *testing.T) {
	for stored, shown := range map[string]string{"every 12h": "12 hours", "every 90m": "every 90m", "daily 04:00": "daily 04:00", "off": "off"} {
		if got := scheduleText(stored); got != shown {
			t.Errorf("scheduleText(%q) = %q, want %q", stored, got, shown)
		}
	}
	for typed, stored := range map[string]string{"12": "every 12h", "12 hours": "every 12h", "0": "off", "off": "off",
		scheduleText("daily 04:00"): "daily 04:00", "daily 05:30": "daily 05:30", " every  90m ": "every 90m"} {
		if got, err := scheduleHours(typed); err != nil || got != stored {
			t.Errorf("scheduleHours(%q) = %q, %v, want %q", typed, got, err, stored)
		}
	}
	for _, bad := range []string{"daily", "daily 25:00", "-3"} {
		if _, err := scheduleHours(bad); err == nil {
			t.Errorf("scheduleHours took %q", bad)
		}
	}
}

func TestModelIsAPickList(t *testing.T) {
	if core.DefaultConfig().Model != "opus" || core.DefaultConfig().LeadModel != "opus" {
		t.Fatal("Opus is not the default model")
	}
	c := core.DefaultConfig()
	c.ModelChoices = []string{"sonnet", "my-model"}
	ch := modelChoices(c)
	if ch[0] != "opus" || ch[len(ch)-1] != custom || ch[len(ch)-2] != "my-model" || len(slices.Compact(slices.Sorted(slices.Values(ch)))) != len(ch) {
		t.Fatalf("choices %v: want opus first, the config's own added once, custom last", ch)
	}
	m := testModel(80, 24)
	m.section = secSettings
	for i, s := range m.settings() {
		if s.key == "lead_model" {
			m.setSel = i
		}
	}
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if !m.picking || m.settings()[m.setSel].choices[m.pickSel] != "opus" {
		t.Fatalf("enter on the lead model should open its list at opus (picking %v, sel %d)", m.picking, m.pickSel)
	}
	if !strings.Contains(m.View(), custom) {
		t.Fatal("the open list does not show the custom entry")
	}
	for _, k := range []tea.KeyMsg{{Type: tea.KeyRunes, Runes: []rune("q")}, {Type: tea.KeyTab}} {
		if _, cmd := m.Update(k); m.picking || m.section != secSettings || cmd != nil {
			t.Fatalf("%v should close the open list and stay in Settings", k)
		}
		m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	}
	m.Update(tea.KeyMsg{Type: tea.KeyUp}) // already at the top
	for range len(m.settings()[m.setSel].choices) {
		m.Update(tea.KeyMsg{Type: tea.KeyDown})
	}
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.picking || !m.editing {
		t.Fatal("custom should open a text field")
	}
}
