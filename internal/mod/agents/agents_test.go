package agents

import (
	"path/filepath"
	"testing"

	"github.com/ihsan-sa/hive-standalone-build/internal/core"
)

// A project's own model is more specific than lead_model, so it runs that
// project's lead; a lead with no project of its own gets lead_model.
func TestProjectModelBeatsLeadModel(t *testing.T) {
	dir := t.TempDir()
	c, err := core.Open(core.Workspace{Root: filepath.Join(dir, "hive")}, &core.Local{Dir: filepath.Join(dir, "forge")})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.Close() })
	cfg := c.WS.LoadConfig()
	cfg.Model, cfg.LeadModel = "sonnet", "opus"
	cfg.Projects = map[string]core.ProjectSettings{"p": {Model: "haiku"}}
	if err := c.WS.SaveConfig(cfg); err != nil {
		t.Fatal(err)
	}
	a := &Agents{C: c}
	for _, tc := range []struct{ role, project, want string }{{"lead", "p", "haiku"}, {"lead", "", "opus"}, {"worker", "q", "sonnet"}} {
		if _, got := a.Model(tc.role, tc.project); got != tc.want {
			t.Errorf("Model(%q, %q) = %q, want %q", tc.role, tc.project, got, tc.want)
		}
	}
}
