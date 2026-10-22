package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ihsan-sa/hive-standalone-build/internal/api"
	"github.com/ihsan-sa/hive-standalone-build/internal/core"
	"github.com/ihsan-sa/hive-standalone-build/internal/mod/report"
)

const shareQuestion = "Share failures and usage with the Hive's developer so it gets fixed? Secrets are scrubbed on this machine first, and `%s report` shows what was sent. (y/N)"

// setShare turns sharing on or off.
func setShare(ws core.Workspace, on bool) error {
	cfg := ws.LoadConfig()
	cfg.Share = ""
	if on {
		cfg.Share = time.Now().UTC().Format(time.RFC3339)
	}
	return ws.SaveConfig(cfg)
}

// cmdReport shows what's shared, turns it on or off, prints the last batch
// sent, or sends one now.
func cmdReport(args []string) int {
	ws, err := workspace()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	sub := ""
	if len(args) > 0 {
		sub = args[0]
	}
	switch sub {
	case "on", "off":
		if err := setShare(ws, sub == "on"); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
	case "show":
		sent := report.Sent(ws)
		if len(sent) == 0 {
			fmt.Println("Nothing has been sent.")
			return 0
		}
		raw, _ := os.ReadFile(sent[len(sent)-1])
		fmt.Println(string(raw))
		return 0
	case "send":
		return withClient(func(c *api.Client) error {
			return c.Do("POST", "/api/job", map[string]string{"name": "report.send"}, nil)
		})
	case "":
	default:
		fmt.Fprintln(os.Stderr, "usage: "+core.Cmd+" report [on|off|show|send]")
		return 2
	}
	cfg := ws.LoadConfig()
	if cfg.Share == "" {
		fmt.Println("Sharing failures with the developer is off. `" + core.Cmd + " report on` turns it on.")
	} else {
		fmt.Printf("Sharing failures and usage with %s since %s (install %s).\n", report.URL(cfg), cfg.Share, cfg.InstallID)
	}
	if sent := report.Sent(ws); len(sent) > 0 {
		fmt.Printf("%d batches sent; copies in %s, the last is `%s report show`.\n", len(sent), filepath.Dir(sent[0]), core.Cmd)
	}
	return 0
}
