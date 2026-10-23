package tui

import (
	"fmt"
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

func protectedAsk() core.Ask {
	var diff strings.Builder
	diff.WriteString("diff --git a/.hive/goals.md b/.hive/goals.md\n--- a/.hive/goals.md\n+++ b/.hive/goals.md\n@@ -1,1 +1,60 @@\n")
	for i := 0; i < 60; i++ {
		fmt.Fprintf(&diff, "+- goal %d\n", i)
	}
	return core.Ask{ID: "a2", Kind: "merge", Ref: "t1@abc", Title: `Merge "implement"? It changes .hive/goals.md`,
		Detail: ".hive/goals.md are yours to merge.", Options: []string{"merge", "leave open"}, Diff: diff.String(), Asked: time.Now()}
}

func TestWaitingShowsTheDiff(t *testing.T) {
	for _, size := range [][2]int{{120, 40}, {80, 24}, {50, 16}} {
		w, h := size[0], size[1]
		m := testModel(w, h)
		m.st.Asks = []core.Ask{protectedAsk()}
		m.section = secWaiting
		v := ansi.Strip(m.View())
		checkWidth(t, m.View(), w, h, "waiting with a diff")
		if !strings.Contains(v, "+- goal 0") || strings.Contains(v, "+- goal 59") || !strings.Contains(v, "d the whole diff") || !strings.Contains(v, "leave open") {
			t.Fatalf("%dx%d: Waiting should show the start of the diff, clipped, and the options:\n%s", w, h, v)
		}
		m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
		if len(m.stack) != 1 || m.stack[0].kind != "diff" {
			t.Fatalf("d should open the whole diff: %+v", m.stack)
		}
		for i := 0; i < 80; i++ {
			m.Update(tea.KeyMsg{Type: tea.KeyDown})
		}
		checkWidth(t, m.View(), w, h, "the whole diff")
		if v := ansi.Strip(m.View()); !strings.Contains(v, "+- goal 59") {
			t.Fatalf("%dx%d: scrolling the whole diff should reach its end:\n%s", w, h, v)
		}
	}
}

func TestChatJumpsToItsAsk(t *testing.T) {
	m := testModel(80, 24)
	m.st.Projects[0].Tasks[0].Chat = "lead"
	m.st.Asks = append(m.st.Asks, protectedAsk()) // a1 is someone else's; a2 is about t1, a task of this chat
	if v := ansi.Strip(m.View()); !strings.Contains(v, "2 waiting on you · → then a to answer") {
		t.Fatalf("the chat should say what's waiting:\n%s", v)
	}
	m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	if m.section != secWaiting || m.st.Asks[m.waitSel].ID != "a2" || m.optSel != 0 {
		t.Fatalf("a should open Waiting on this chat's ask: section %d, ask %d", m.section, m.waitSel)
	}
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.composing {
		t.Fatal("Enter in Waiting should answer, not write")
	}
}
