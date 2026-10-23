package main

import (
	"path/filepath"
	"testing"

	"github.com/ihsan-sa/hive-standalone-build/internal/core"
)

// The installer answers before there is a Hive; first run takes that answer
// without asking, and `report asked` tells a reinstall not to ask again.
func TestInstallerAnswerReachesFirstRun(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "config"))
	t.Setenv("HOME", dir)
	t.Setenv("HIVE_HOME", filepath.Join(dir, "hive"))
	if cmdReport([]string{"asked"}) != 1 {
		t.Fatal("nothing answered yet, but asked says yes")
	}
	if cmdReport([]string{"on"}) != 0 || cmdReport([]string{"asked"}) != 0 {
		t.Fatal("the installer's answer wasn't kept")
	}
	ws := core.Workspace{Root: filepath.Join(dir, "hive")}
	if err := ws.Init(); err != nil {
		t.Fatal(err)
	}
	if err := askShare(ws); err != nil {
		t.Fatal(err)
	}
	if cfg := ws.LoadConfig(); cfg.InstallID == "" {
		t.Fatalf("first run didn't take the installer's yes: %+v", cfg)
	} else if _, on := cfg.Sharing(); !on {
		t.Fatalf("first run didn't take the installer's yes: %q", cfg.Share)
	}
	if cmdReport([]string{"off"}) != 0 {
		t.Fatal("report off failed")
	}
	if err := askShare(ws); err != nil { // answered: no prompt, nothing changes
		t.Fatal(err)
	}
	if cfg := ws.LoadConfig(); cfg.Share != "off" || cmdReport([]string{"asked"}) != 0 {
		t.Fatalf("want off and answered, got %q", cfg.Share)
	}
}
