// Package api is the control center's view of the daemon: the types it
// returns and a client for its local HTTP endpoint.
package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/ihsan-sa/hive-standalone-build/internal/core"
	"github.com/ihsan-sa/hive-standalone-build/internal/harness"
	"github.com/ihsan-sa/hive-standalone-build/internal/mod/chat"
	"github.com/ihsan-sa/hive-standalone-build/internal/mod/improve"
	"github.com/ihsan-sa/hive-standalone-build/internal/mod/risk"
	"github.com/ihsan-sa/hive-standalone-build/internal/mod/system"
	"github.com/ihsan-sa/hive-standalone-build/internal/mod/work"
)

// DaemonInfo is .hive/daemon.json.
type DaemonInfo struct {
	PID     int       `json:"pid"`
	Started time.Time `json:"started"`
	Port    int       `json:"port"`
	Token   string    `json:"token"`
	Version string    `json:"version"`
}

// State is everything the control center shows.
type State struct {
	Version    string            `json:"version"`
	Seq        int64             `json:"seq"`
	Workspace  string            `json:"workspace"`
	Sandbox    string            `json:"sandbox"`
	SandboxErr string            `json:"sandbox_err,omitempty"`
	Projects   []ProjectView     `json:"projects"`
	Runs       []core.Run        `json:"runs"`
	Plans      []core.Plan       `json:"plans"`
	Sweeps     []improve.Sweep   `json:"sweeps"`
	Asks       []core.Ask        `json:"asks"`
	Alerts     []Alert           `json:"alerts"`
	Jobs       []system.JobState `json:"jobs"`
	Wakeups    []system.Wakeup   `json:"wakeups"`
	Modules    []ModuleView      `json:"modules"`
	Chats      []ChatView        `json:"chats"`
	Requests   []risk.Request    `json:"requests"`
	Config     core.Config       `json:"config"`
}

// ProjectView is one project on the dashboard.
type ProjectView struct {
	core.Project
	Goals  string                `json:"goals"`
	Tasks  []work.Task           `json:"tasks"`
	Scores []improve.Measurement `json:"scores"`
	Best   *float64              `json:"best,omitempty"`
}

// Alert is a ledger alert.
type Alert struct {
	At   time.Time `json:"at"`
	Kind string    `json:"kind"`
	Text string    `json:"text"`
}

// ModuleView is one installed module.
type ModuleView struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Source      string   `json:"source"`
	Release     string   `json:"release"`
	Projects    []string `json:"projects"`
}

// ChatView is one chat in the list.
type ChatView struct {
	chat.Chat
	Unread     int    `json:"unread"`
	LeadStatus string `json:"lead_status,omitempty"`
}

// RunDetail is an agent's step-by-step log.
type RunDetail struct {
	Run   core.Run `json:"run"`
	Steps []Step   `json:"steps"`
}

// Step is one line of an agent's log.
type Step struct {
	Turn int    `json:"turn"`
	Kind string `json:"kind"`
	Tool string `json:"tool,omitempty"`
	Text string `json:"text"`
}

// StepsFrom converts harness events.
func StepsFrom(turn int, evs []harness.Event) []Step {
	var out []Step
	for _, e := range evs {
		out = append(out, Step{Turn: turn, Kind: e.Kind, Tool: e.Tool, Text: e.Text})
	}
	return out
}

// Client talks to a running daemon.
type Client struct {
	Info DaemonInfo
	http *http.Client
}

// Dial reads daemon.json and returns a client (without checking liveness).
func Dial(ws core.Workspace) (*Client, error) {
	raw, err := os.ReadFile(ws.DaemonFile())
	if err != nil {
		return nil, errors.New("the Hive isn't running")
	}
	var info DaemonInfo
	if err := json.Unmarshal(raw, &info); err != nil {
		return nil, err
	}
	return &Client{Info: info, http: &http.Client{Timeout: 60 * time.Second}}, nil
}

// Do calls an endpoint; body may be nil; out may be nil.
func (c *Client) Do(method, path string, body, out any) error {
	var rd io.Reader
	if body != nil {
		raw, _ := json.Marshal(body)
		rd = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(method, fmt.Sprintf("http://127.0.0.1:%d%s", c.Info.Port, path), rd)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.Info.Token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		var e struct{ Error string }
		if json.Unmarshal(raw, &e) == nil && e.Error != "" {
			return errors.New(e.Error)
		}
		return fmt.Errorf("%s %s: %s", method, path, resp.Status)
	}
	if out != nil {
		return json.Unmarshal(raw, out)
	}
	return nil
}

// Ping reports whether the daemon answers.
func (c *Client) Ping() bool {
	var v map[string]any
	return c.Do("GET", "/api/ping", nil, &v) == nil
}

// State fetches the dashboard state.
func (c *Client) State() (State, error) {
	var s State
	return s, c.Do("GET", "/api/state", nil, &s)
}
