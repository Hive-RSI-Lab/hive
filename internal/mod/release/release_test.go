package release

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/ihsan-sa/hive-standalone-build/internal/core"
)

// releaseAt is the release module in a build stamped version, with v9.9.9 out.
func releaseAt(t *testing.T, version string) (*Module, *bool) {
	dir := t.TempDir()
	forge := &core.Local{Dir: filepath.Join(dir, "forge"), ReleaseAssets: []string{"checksums.txt", AssetName(runtime.GOOS, runtime.GOARCH)}}
	c, err := core.Open(core.Workspace{Root: filepath.Join(dir, "hive")}, forge)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.Close() })
	forge.CreateRelease(c.WS.LoadConfig().ReleaseRepo, "v9.9.9", "", "")
	restarted := false
	m := &Module{C: c, Restart: func() { restarted = true }}
	v := core.Version
	t.Cleanup(func() { core.Version = v })
	core.Version = version
	return m, &restarted
}

// A source build (no release version) never replaces itself with a release.
func TestSourceBuildNeverUpdates(t *testing.T) {
	m, restarted := releaseAt(t, "0.0.0-dev")
	msg, err := m.Update()
	if err != nil || !strings.Contains(msg, "source build") || *restarted {
		t.Fatalf("source build: %q, %v, restarted=%v", msg, err, *restarted)
	}
}

// A release build, stamped with its tag as release.yml does, is offered the
// newer release and goes on to fetch it (the local forge has no assets to give).
func TestReleaseBuildUpdates(t *testing.T) {
	m, _ := releaseAt(t, "v0.1.0")
	if _, tag, ok, err := m.Check(); err != nil || !ok || tag != "v9.9.9" {
		t.Fatalf("check: %q %v %v", tag, ok, err)
	}
	if _, err := m.Update(); err == nil || !strings.Contains(err.Error(), "release assets") {
		t.Fatalf("the update didn't try to download v9.9.9: %v", err)
	}
}

// A release the person's Hive cut on its own fork is the next update when it
// is newer than the Hive's own releases.
func TestCheckFollowsTheFork(t *testing.T) {
	m, _ := releaseAt(t, "v0.1.0")
	if _, err := m.C.Projects.Add(core.Project{Name: "hive", Repo: "local/hive", Self: true}); err != nil {
		t.Fatal(err)
	}
	m.C.Effects.Forge.CreateRelease("local/hive", "v9.9.10", "", "")
	if repo, tag, ok, err := m.Check(); err != nil || !ok || repo != "local/hive" || tag != "v9.9.10" {
		t.Fatalf("check: %q %q %v %v", repo, tag, ok, err)
	}
}

// Only release_repo and the person's fork of it are update sources: a Hive
// project an agent pointed at another repository is never followed, and a
// release without this platform's binary is passed over.
func TestCheckFollowsOnlyTheHivesOwnRepository(t *testing.T) {
	m, _ := releaseAt(t, "v0.1.0")
	forge := m.C.Effects.Forge.(*core.Local)
	if _, err := m.C.Projects.Add(core.Project{Name: "hive", Repo: "evil/hive", Self: true}); err != nil {
		t.Fatal(err)
	}
	forge.CreateRelease("evil/hive", "v99.0.0", "", "")
	if repo, tag, _, err := m.Check(); err != nil || repo != m.C.WS.LoadConfig().ReleaseRepo || tag != "v9.9.9" {
		t.Fatalf("check followed %q %q (%v), want the Hive's own v9.9.9", repo, tag, err)
	}
	if _, err := m.C.Projects.Add(core.Project{Name: "hive-fork", Repo: "local/hive", Self: true}); err != nil {
		t.Fatal(err)
	}
	forge.ReleaseAssets = nil // a fork with Actions off cuts a release with no binaries
	forge.CreateRelease("local/hive", "v9.9.10", "", "")
	if repo, tag, _, err := m.Check(); err != nil || tag != "v9.9.9" {
		t.Fatalf("check took the release with no binaries: %q %q %v", repo, tag, err)
	}
}
