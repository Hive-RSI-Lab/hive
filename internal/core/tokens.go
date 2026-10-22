package core

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

// Tokens hands agents a short-lived credential for their harness, read fresh
// for each turn. Copying a login whole would sign the person out (two holders
// of one refresh token revoke each other), so agents only ever get the access
// token, which expires on its own; the harness's own login on the host keeps
// the refresh token and renews it.
type Tokens struct {
	ws          Workspace
	mu          sync.Mutex
	lastRefresh time.Time
}

// NewTokens builds the token source.
func NewTokens(ws Workspace) *Tokens { return &Tokens{ws: ws} }

// ErrNoLogin means the harness is not logged in on this machine.
var ErrNoLogin = errors.New("Claude Code is not logged in on this machine")

// AgentEnv returns environment entries carrying a short-lived credential.
func (t *Tokens) AgentEnv(h string) ([]string, error) {
	switch h {
	case "fake":
		return nil, nil
	case "", "claude", "claude-code":
		cfg := t.ws.LoadConfig()
		if cfg.APIKeyOptIn {
			if k := os.Getenv("ANTHROPIC_API_KEY"); k != "" {
				return []string{"ANTHROPIC_API_KEY=" + k}, nil
			}
		}
		tok, exp, err := readClaudeAccess()
		if err == nil && time.Until(exp) < 15*time.Minute {
			t.refresh()
			tok, exp, err = readClaudeAccess()
		}
		if err != nil {
			return nil, err
		}
		if time.Until(exp) <= 0 {
			return nil, errors.New("Claude login has expired; open Claude Code once to renew it")
		}
		return []string{"CLAUDE_CODE_OAUTH_TOKEN=" + tok}, nil
	}
	return nil, errors.New("no token source for harness " + h)
}

type claudeCreds struct {
	OAuth *struct {
		AccessToken string `json:"accessToken"`
		ExpiresAt   int64  `json:"expiresAt"` // ms since epoch
	} `json:"claudeAiOauth"`
}

// readClaudeAccess reads only the access token and its expiry from Claude
// Code's own credential store. It is never logged, printed or stored.
func readClaudeAccess() (string, time.Time, error) {
	var raw []byte
	var err error
	if runtime.GOOS == "darwin" {
		raw, err = exec.Command("security", "find-generic-password", "-s", "Claude Code-credentials", "-w").Output()
	} else {
		dir := os.Getenv("CLAUDE_CONFIG_DIR")
		if dir == "" {
			dir = filepath.Join(HomeDir(), ".claude")
		}
		raw, err = os.ReadFile(filepath.Join(dir, ".credentials.json"))
	}
	if err != nil {
		return "", time.Time{}, ErrNoLogin
	}
	var c claudeCreds
	if json.Unmarshal(raw, &c) != nil || c.OAuth == nil || c.OAuth.AccessToken == "" {
		return "", time.Time{}, ErrNoLogin
	}
	return c.OAuth.AccessToken, time.UnixMilli(c.OAuth.ExpiresAt), nil
}

// refresh asks the host's own Claude Code to renew its login with the
// smallest possible call. At most once every ten minutes.
func (t *Tokens) refresh() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if time.Since(t.lastRefresh) < 10*time.Minute {
		return
	}
	t.lastRefresh = time.Now()
	cmd := exec.Command("claude", "-p", "Reply with: ok", "--model", "haiku", "--tools", "", "--no-session-persistence", "--strict-mcp-config", "--setting-sources", "")
	cmd.Dir = os.TempDir()
	cmd.Stdin = nil
	_ = cmd.Run()
}
