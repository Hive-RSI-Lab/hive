package daemon

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/ihsan-sa/hive-standalone-build/internal/api"
	"github.com/ihsan-sa/hive-standalone-build/internal/app"
	"github.com/ihsan-sa/hive-standalone-build/internal/core"
)

// After an update renames the running binary to hive.old, the restart runs
// the new hive, not the old code under its new name, and daemon.json tells
// the control center to wait for it.
func TestRestartRunsNewBinary(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the stand-in binaries are shell scripts")
	}
	dir := t.TempDir()
	ws := core.Workspace{Root: filepath.Join(dir, "ws")}
	if err := os.MkdirAll(ws.Hive(), 0o755); err != nil {
		t.Fatal(err)
	}
	ran := filepath.Join(dir, "ran")
	for name, says := range map[string]string{"hive": `"$*"`, "hive.old": "old"} {
		script := fmt.Sprintf("#!/bin/sh\necho %s > '%s.tmp' && mv '%s.tmp' '%s'\n", says, ran, ran, ran)
		if err := os.WriteFile(filepath.Join(dir, name), []byte(script), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	defer func(f func() (string, error)) { executable = f }(executable)
	executable = func() (string, error) { return filepath.Join(dir, "hive.old"), nil }
	stopped := false
	s := &server{a: &app.App{C: &core.Core{WS: ws}}, info: api.DaemonInfo{PID: os.Getpid(), Token: "t"}, cancel: func() { stopped = true }}
	s.restart()
	if !stopped {
		t.Fatal("the old daemon didn't stop")
	}
	if c, err := api.Dial(ws); err != nil || !c.Info.Restarting {
		t.Fatalf("daemon.json doesn't say a restart is under way: %v", err)
	}
	want := fmt.Sprintf("daemon --wait-for %d", os.Getpid())
	got := ""
	for i := 0; i < 50 && got != want; i++ {
		time.Sleep(100 * time.Millisecond)
		raw, _ := os.ReadFile(ran)
		got = strings.TrimSpace(string(raw))
	}
	if got != want {
		t.Fatalf("the restart ran %q, want the new binary with %q", got, want)
	}
}

// Five identical failures are one alert with a count; a different one is a
// second alert. A repeat with fresh output refreshes the text and moves to the
// newest place, and the same failure on another project is its own alert.
func TestAlertsDedup(t *testing.T) {
	s := &server{}
	raise := func(project, key, text string) {
		raw, _ := json.Marshal(map[string]string{"kind": "baseline", "project": project, "key": key, "text": text})
		s.onEvent(core.Event{Kind: "alert.raised", At: time.Now(), Data: raw})
	}
	for i := 0; i < 5; i++ {
		raise("p", "python3 isn't installed", "python3 isn't installed")
	}
	raise("p", "git isn't installed", "git isn't installed")
	if len(s.alerts) != 2 || s.alerts[0].Count != 5 || s.alerts[1].Count != 1 {
		t.Fatalf("alerts: %+v", s.alerts)
	}
	raise("p", "python3 isn't installed", "python3 isn't installed, run 6")
	raise("q", "python3 isn't installed", "python3 isn't installed")
	if len(s.alerts) != 3 || s.alerts[1].Count != 6 || s.alerts[1].Text != "python3 isn't installed, run 6" || s.alerts[2].Count != 1 {
		t.Fatalf("refresh and per project: %+v", s.alerts)
	}
}
