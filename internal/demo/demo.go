// Package demo is the small project a first run creates when the person
// gives no repository: a library with tests and a score, so the first run
// has something to improve.
package demo

import (
	"embed"
	"io/fs"
	"strings"
)

//go:embed all:template
var tmpl embed.FS

// Files returns the demo project's files by repo-relative path.
func Files() map[string]string {
	out := map[string]string{}
	fs.WalkDir(tmpl, "template", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		raw, _ := tmpl.ReadFile(p)
		out[strings.TrimPrefix(p, "template/")] = string(raw)
		return nil
	})
	return out
}
