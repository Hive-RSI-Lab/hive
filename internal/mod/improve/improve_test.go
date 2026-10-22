package improve

import (
	"strings"
	"testing"
)

func TestMissingTool(t *testing.T) {
	got := MissingTool("textkit-2", "tests red: /bin/sh: 1: -m: not found")
	if !strings.Contains(got, "no program") || strings.Contains(got, "-m") {
		t.Errorf("empty interpreter: %q", got)
	}
	if got := MissingTool("p", "sh: 1: python3: not found"); got != "python3 isn't installed, so p's tests can't run" {
		t.Errorf("named tool: %q", got)
	}
	if got := MissingTool("p", "FAILED test_x"); got != "" {
		t.Errorf("plain failure: %q", got)
	}
}
