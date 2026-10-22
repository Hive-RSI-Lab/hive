package kb

import (
	"fmt"
	"go/doc"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ihsan-sa/hive-standalone-build/internal/core"
)

// SelfDir is where the Hive's own source sits in the knowledge base.
const SelfDir = "hive"

// seedSelf writes the Hive's source and design into the knowledge base at the
// running version, with a source map, so every agent can read the code it
// runs. It runs at every start, so an update (a new binary) reseeds it, and a
// commit is made only when something changed.
func (m *Module) seedSelf(src fs.FS) {
	dir := filepath.Join(m.C.WS.KB(), SelfDir)
	os.RemoveAll(dir)
	fs.WalkDir(src, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		raw, err := fs.ReadFile(src, p)
		if err == nil {
			os.MkdirAll(filepath.Join(dir, filepath.Dir(p)), 0o755)
			os.WriteFile(filepath.Join(dir, p), raw, 0o644)
		}
		return nil
	})
	os.WriteFile(filepath.Join(dir, "VERSION"), []byte(core.Version+"\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "SOURCE-MAP.md"), []byte(sourceMap(src)), 0o644)
	m.mu.Lock()
	defer m.mu.Unlock()
	core.Git(m.C.WS.KB(), "add", "-A", SelfDir)
	core.Git(m.C.WS.KB(), "commit", "-q", "-m", "The Hive's own source at "+core.Version, "--", SelfDir)
}

// sourceMap lists the design docs and each Go package with its doc comment.
func sourceMap(src fs.FS) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# The Hive's own source, at %s\n\n", core.Version)
	b.WriteString("This folder is the Hive's repository at the version this install runs, rewritten at every start. Read the code here to answer how the Hive works, and the live settings in `hive agent status` (`config`). The design: `docs/spec.md`, `docs/architecture.md`, `docs/risk.md`, `docs/status.md`. The default modules (role prompts) are in `modules/`.\n\n## Packages\n\n")
	synopsis := map[string]string{}
	fs.WalkDir(src, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
			return err
		}
		raw, _ := fs.ReadFile(src, p)
		f, err := parser.ParseFile(token.NewFileSet(), p, raw, parser.PackageClauseOnly|parser.ParseComments)
		if dirName := path.Dir(p); err == nil && f.Doc != nil {
			synopsis[dirName] = new(doc.Package).Synopsis(f.Doc.Text())
		} else if _, ok := synopsis[dirName]; !ok {
			synopsis[dirName] = ""
		}
		return nil
	})
	var dirs []string
	for d := range synopsis {
		dirs = append(dirs, d)
	}
	sort.Strings(dirs)
	for _, d := range dirs {
		fmt.Fprintf(&b, "- `%s`: %s\n", d, synopsis[d])
	}
	return b.String()
}
