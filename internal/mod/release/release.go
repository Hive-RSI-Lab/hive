// Package release is the Hive on itself: installs follow the Hive's releases,
// and when the Hive's own project merges a change that keeps its tests and a
// dry run green, it cuts the next release.
package release

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/ihsan-sa/hive-standalone-build/internal/core"
	"github.com/ihsan-sa/hive-standalone-build/internal/mod/modules"
)

// Module is the release module.
type Module struct {
	C *core.Core
	// Restart is called after a new binary is in place (the daemon re-execs).
	Restart func()
}

// AssetName is the release asset for an OS and architecture.
func AssetName(goos, goarch string) string {
	n := "hive-" + goos + "-" + goarch
	if goos == "windows" {
		n += ".exe"
	}
	return n
}

// Check returns the newest release, and where it is, if it is newer than this
// build. It looks at the Hive's releases and at the Hive's own project, so a
// change the person's Hive released on their fork is the next update.
func (m *Module) Check() (repo, tag string, newer bool, err error) {
	repos := []string{m.C.WS.LoadConfig().ReleaseRepo}
	for _, p := range m.C.Projects.List() {
		if p.Self && p.Repo != repos[0] {
			repos = append(repos, p.Repo)
		}
	}
	for i, r := range repos {
		rel, e := m.C.Effects.Forge.LatestRelease(r)
		if e != nil {
			if i == 0 {
				err = e
			}
			continue
		}
		if tag == "" || modules.Newer(tag, rel.Tag) {
			repo, tag = r, rel.Tag
		}
	}
	if tag != "" {
		err = nil
	}
	return repo, tag, tag != "" && modules.Newer(core.Version, tag), err
}

// Update installs the newest release over this binary. Live runs are separate
// processes and the daemon reattaches to them after it restarts, so live work
// carries across.
func (m *Module) Update() (string, error) {
	if !strings.HasPrefix(core.Version, "v") {
		return "this is a source build (" + core.Version + "); it doesn't update itself", nil
	}
	repo, tag, newer, err := m.Check()
	if err != nil {
		return "", err
	}
	if !newer {
		return "already on " + core.Version, nil
	}
	tmp, err := os.MkdirTemp("", "hive-update-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmp)
	names := []string{AssetName(runtime.GOOS, runtime.GOARCH)}
	if runtime.GOOS == "windows" {
		names = append(names, AssetName("linux", "amd64")) // used inside WSL
	}
	if err := m.C.Effects.Forge.DownloadAsset(repo, tag, "checksums.txt", tmp); err != nil {
		return "", err
	}
	sums, err := readSums(filepath.Join(tmp, "checksums.txt"))
	if err != nil {
		return "", err
	}
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	dir := filepath.Dir(exe)
	for i, name := range names {
		if err := m.C.Effects.Forge.DownloadAsset(repo, tag, name, tmp); err != nil {
			return "", err
		}
		got, err := fileSum(filepath.Join(tmp, name))
		if err != nil {
			return "", err
		}
		if want := sums[name]; want == "" || want != got {
			return "", fmt.Errorf("%s: checksum mismatch; not installing", name)
		}
		target := exe
		if i > 0 {
			target = filepath.Join(dir, name)
		}
		if err := replace(filepath.Join(tmp, name), target); err != nil {
			return "", err
		}
	}
	m.C.J.MustAppend("hive.updated", map[string]string{"from": core.Version, "to": tag})
	if m.Restart != nil {
		go func() { time.Sleep(time.Second); m.Restart() }()
	}
	return "updated to " + tag, nil
}

// AutoUpdate is the scheduled job: update when allowed and nothing is mid-merge.
func (m *Module) AutoUpdate() (string, error) {
	if !m.C.WS.LoadConfig().AutoUpdate {
		return "auto-update is off", nil
	}
	return m.Update()
}

// replace moves src over dst; a running Windows executable is renamed aside first.
func replace(src, dst string) error {
	raw, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	next := dst + ".new"
	if err := os.WriteFile(next, raw, 0o755); err != nil {
		return err
	}
	old := dst + ".old"
	os.Remove(old)
	if core.Exists(dst) {
		if err := os.Rename(dst, old); err != nil {
			return err
		}
	}
	if err := os.Rename(next, dst); err != nil {
		os.Rename(old, dst)
		return err
	}
	return nil
}

func readSums(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	out := map[string]string{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) == 2 {
			out[strings.TrimPrefix(fields[1], "*")] = fields[0]
		}
	}
	if len(out) == 0 {
		return nil, errors.New("empty checksums.txt")
	}
	return out, sc.Err()
}

func fileSum(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// AfterKept cuts a release when a kept merge lands in the Hive's own project
// and its dry run passes on that exact commit. The release numbers on from
// this build, so the install switches to it at its next update. In the
// person's fork it also offers the change upstream, one pull request that
// grows with each kept merge.
func (m *Module) AfterKept(project, sha string) {
	p, ok := m.C.Projects.Get(project)
	if !ok || !p.Self {
		return
	}
	cfg := m.C.Effects.Config(p, sha)
	if cfg.DryRun != "" {
		c := m.C.Effects.RunCheck(p, sha, "dryrun", cfg.DryRun)
		if !c.OK {
			m.C.J.MustAppend("alert.raised", map[string]string{"kind": "release", "project": project, "text": "dry run failed on " + core.Short(sha) + "; no release"})
			return
		}
	}
	last := "v0.0.0"
	if rel, err := m.C.Effects.Forge.LatestRelease(p.Repo); err == nil && rel.Tag != "" {
		last = rel.Tag
	}
	if strings.HasPrefix(core.Version, "v") && modules.Newer(last, core.Version) {
		last = core.Version
	}
	next := bump(last)
	if err := m.C.Effects.Forge.CreateRelease(p.Repo, next, sha, "Released by the Hive after "+core.Short(sha)+" kept its tests and dry run green."); err != nil {
		m.C.J.MustAppend("alert.raised", map[string]string{"kind": "release", "project": project, "text": "release failed: " + err.Error()})
		return
	}
	m.C.J.MustAppend("hive.released", map[string]string{"project": project, "tag": next, "sha": sha})
	f, up := m.C.Effects.Forge, m.C.WS.LoadConfig().ReleaseRepo
	if p.Repo == up {
		return
	}
	head := strings.SplitN(p.Repo, "/", 2)[0] + ":" + p.Branch
	pr, err := f.CreatePR(up, head, f.DefaultBranch(up), "Improvements from "+p.Repo,
		"Changes the Hive made to itself in "+p.Repo+", each kept by its tests, review and dry run. Released there as "+next+".")
	if err != nil {
		m.C.J.MustAppend("alert.raised", map[string]string{"kind": "release", "project": project, "text": "offering the change upstream failed: " + err.Error()})
		return
	}
	m.C.J.MustAppend("hive.offered", map[string]string{"project": project, "pr": pr.URL})
}

func bump(tag string) string {
	var a, b, c int
	fmt.Sscanf(strings.TrimPrefix(tag, "v"), "%d.%d.%d", &a, &b, &c)
	return fmt.Sprintf("v%d.%d.%d", a, b, c+1)
}
