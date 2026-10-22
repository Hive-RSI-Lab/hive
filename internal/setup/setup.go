// Package setup is the first run, `hive login`, `hive doctor`, starting with
// the machine, and uninstalling. The Hive never stores, copies or prints a
// login: it runs the tools' own logins in the person's terminal.
package setup

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/ihsan-sa/hive-standalone-build/internal/core"
	"github.com/ihsan-sa/hive-standalone-build/internal/sandbox"
)

// Tool is a prerequisite.
type Tool struct {
	Name       string
	Found      bool
	LoggedIn   bool
	NeedsLogin bool
	Get        string // what to get and where
	Login      []string
}

// Check finds git, Claude Code and the GitHub CLI, and whether they're logged in.
func Check() []Tool {
	tools := []Tool{
		{Name: "git", Get: "Install git: https://git-scm.com/downloads"},
		{Name: "claude", NeedsLogin: true, Login: []string{"claude", "auth", "login"},
			Get: "Install Claude Code: https://claude.com/claude-code  (macOS/Linux: curl -fsSL https://claude.ai/install.sh | bash   Windows: irm https://claude.ai/install.ps1 | iex)"},
		{Name: "gh", NeedsLogin: true, Login: []string{"gh", "auth", "login"},
			Get: "Install the GitHub CLI: https://cli.github.com"},
	}
	for i := range tools {
		t := &tools[i]
		_, err := exec.LookPath(t.Name)
		t.Found = err == nil
		if !t.Found || !t.NeedsLogin {
			t.LoggedIn = t.Found
			continue
		}
		switch t.Name {
		case "gh":
			t.LoggedIn = exec.Command("gh", "auth", "status").Run() == nil
		case "claude":
			out, err := exec.Command("claude", "auth", "status", "--json").Output()
			t.LoggedIn = err == nil && regexp.MustCompile(`"loggedIn"\s*:\s*true`).Match(out)
		}
	}
	return tools
}

// Login runs each missing login in the person's terminal.
func Login(all bool) error {
	for _, t := range Check() {
		if !t.Found || !t.NeedsLogin || (t.LoggedIn && !all) {
			continue
		}
		fmt.Printf("\nLogging in to %s with its own login. Follow its link or code in your browser.\n", t.Name)
		cmd := exec.Command(t.Login[0], t.Login[1:]...)
		cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("%s login: %v", t.Name, err)
		}
	}
	return nil
}

// Prompt asks a question on the terminal.
func Prompt(q, def string) string {
	if def != "" {
		fmt.Printf("%s [%s]: ", q, def)
	} else {
		fmt.Printf("%s: ", q)
	}
	line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		return def
	}
	return line
}

// SandboxReport explains the sandbox state and what to do about it.
func SandboxReport(ws core.Workspace, hiveLinux string) (string, error) {
	cfg := ws.LoadConfig()
	sb, err := sandbox.Select(sandbox.Options{Mode: "auto", WSLDistro: cfg.WSLDistro, HiveLinux: hiveLinux})
	if err == nil {
		return "sandbox: " + sb.Name(), nil
	}
	hint := ""
	switch runtime.GOOS {
	case "windows":
		if strings.Contains(err.Error(), "Linux hive binary") {
			hint = "The installer puts hive-linux-amd64 next to hive.exe; run the install line again."
			break
		}
		hint = "Agents run sandboxed inside WSL 2. Install it with `wsl --install -d Ubuntu`, then install Claude Code inside it:\n    wsl -d Ubuntu -- bash -lc \"curl -fsSL https://claude.ai/install.sh | bash\"\nand run `hive doctor` again."
	case "linux":
		hint = "Agents run in unprivileged user namespaces. If your distribution restricts them (Ubuntu 24.04+: kernel.apparmor_restrict_unprivileged_userns=1), allow them for the hive binary or system-wide, then run `hive doctor` again."
	case "darwin":
		hint = "Agents run under sandbox-exec, which ships with macOS."
	}
	return "", fmt.Errorf("%v\n%s", err, hint)
}

// AcceptBare records that the person read the warning and opted into bare runs.
func AcceptBare(ws core.Workspace) {
	cfg := ws.LoadConfig()
	cfg.Sandbox, cfg.BareAccepted = "bare", time.Now().UTC().Format(time.RFC3339)
	ws.SaveConfig(cfg)
}

// WSLClaude reports whether Claude Code is installed inside the WSL distro.
func WSLClaude(distro string) bool {
	if runtime.GOOS != "windows" {
		return true
	}
	args := []string{"--exec", "sh", "-c", "command -v claude >/dev/null || test -x \"$HOME/.local/bin/claude\""}
	if distro != "" {
		args = append([]string{"-d", distro}, args...)
	}
	return exec.Command("wsl.exe", args...).Run() == nil
}

// Executable returns the running binary's path.
func Executable() string {
	exe, _ := os.Executable()
	exe, _ = filepath.EvalSymlinks(exe)
	return exe
}
