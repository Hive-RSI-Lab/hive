package improve

import (
	"strings"
	"testing"
)

func TestMissingTool(t *testing.T) {
	got := MissingTool("textkit-2", "test", "tests red: /bin/sh: 1: -m: not found")
	if !strings.Contains(got, "no program") || strings.Contains(got, "-m") {
		t.Errorf("empty interpreter: %q", got)
	}
	if got := MissingTool("p", "test", "sh: 1: python3: not found"); got != "python3 isn't installed, so p's tests can't run; install it where the Hive runs its checks" {
		t.Errorf("named tool: %q", got)
	}
	if got := MissingTool("p", "test", "bash: line 1: node: command not found"); !strings.HasPrefix(got, "node isn't installed") {
		t.Errorf("bash form: %q", got)
	}
	if got := MissingTool("p", "test", "tests red: hive: missing program: python or python3\n"); !strings.HasPrefix(got, "python or python3 isn't installed, so p's tests can't run") {
		t.Errorf("guard line: %q", got)
	}
	// Tests passed and the score command's tool is missing: the score can't be measured.
	if got := MissingTool("p", "score", "score did not produce a number: sh: 1: python3: not found"); !strings.HasPrefix(got, "python3 isn't installed, so p's score can't be measured") {
		t.Errorf("score tool: %q", got)
	}
	for _, out := range []string{"FAILED test_x", "AssertionError: key: not found", "lookup: not found in cache"} {
		if got := MissingTool("p", "test", out); got != "" {
			t.Errorf("not a missing program %q: %q", out, got)
		}
	}
}
