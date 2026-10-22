package tools

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// tgz builds a gzipped tar of name -> content (a "->" prefix is a symlink).
func tgz(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(zw)
	for name, body := range files {
		h := &tar.Header{Name: name, Mode: 0o755, Size: int64(len(body)), Typeflag: tar.TypeReg}
		if target, ok := strings.CutPrefix(body, "->"); ok {
			h = &tar.Header{Name: name, Linkname: target, Typeflag: tar.TypeSymlink}
		}
		if err := tw.WriteHeader(h); err != nil {
			t.Fatal(err)
		}
		if h.Typeflag == tar.TypeReg {
			tw.Write([]byte(body))
		}
	}
	tw.Close()
	zw.Close()
	return buf.Bytes()
}

// serve answers path -> body; an archive's .sha256 is served beside it.
func serve(t *testing.T, files map[string][]byte) string {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, ok := files[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Write(body)
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

func sum(b []byte) string { return fmt.Sprintf("%x", sha256.Sum256(b)) }

func TestUnpackVerifiesAndInstalls(t *testing.T) {
	arc := tgz(t, map[string]string{"node-v1/bin/node": "#!/bin/sh\necho node\n", "node-v1/bin/npm": "->../lib/npm.js", "node-v1/lib/npm.js": "x"})
	url := serve(t, map[string][]byte{"/node.tar.gz": arc, "/node.tar.gz.sha256": []byte(sum(arc) + "  node.tar.gz\n")})
	folder := t.TempDir()
	v, err := unpack(folder, "node", "1.2.3", url+"/node.tar.gz", "")
	if err != nil || v != "1.2.3" {
		t.Fatalf("unpack: %q %v", v, err)
	}
	fi, err := os.Stat(filepath.Join(folder, "node", "bin", "node"))
	if err != nil || fi.Mode().Perm()&0o100 == 0 {
		t.Fatalf("node not installed executable: %v", err)
	}
	if target, _ := os.Readlink(filepath.Join(folder, "node", "bin", "npm")); target != "../lib/npm.js" {
		t.Fatalf("symlink: %q", target)
	}
	if got := Installed(folder); len(got) != 1 || got[0] != "node 1.2.3" {
		t.Fatalf("Installed: %q", got)
	}
}

func TestBadChecksumInstallsNothing(t *testing.T) {
	old := tgz(t, map[string]string{"go/bin/go": "old"})
	evil := tgz(t, map[string]string{"go/bin/go": "evil"})
	url := serve(t, map[string][]byte{"/go.tar.gz": evil})
	folder := t.TempDir()
	if _, err := unpackSum(folder, "go", "1", url+"/go.tar.gz", "", sum(old)); err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("want a checksum mismatch, got %v", err)
	}
	if _, err := unpackSum(folder, "go", "1", url+"/go.tar.gz", "", ""); err == nil {
		t.Fatal("an archive with no checksum was installed")
	}
	if entries, _ := os.ReadDir(folder); len(entries) != 0 {
		t.Fatalf("folder not left empty: %v", entries)
	}
}

func TestArchiveCannotLeaveItsFolder(t *testing.T) {
	arc := tgz(t, map[string]string{"top/../../escaped": "x"})
	url := serve(t, map[string][]byte{"/x.tar.gz": arc})
	folder := t.TempDir()
	if _, err := unpackSum(folder, "uv", "1", url+"/x.tar.gz", "bin", sum(arc)); err == nil {
		t.Fatal("an entry outside the folder was written")
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(folder), "escaped")); err == nil {
		t.Fatal("escaped file exists")
	}
}

func TestFetchRefusesOddInput(t *testing.T) {
	if _, err := Fetch(t.TempDir(), "python", "3.12; rm -rf /"); err == nil {
		t.Fatal("a version with a command in it was accepted")
	}
	if _, err := Fetch(t.TempDir(), "rust", ""); err == nil {
		t.Fatal("an unknown tool was accepted")
	}
}

func TestSumFor(t *testing.T) {
	sums := []byte("aaa  node-v1-linux-x64.tar.gz\nbbb  node-v1-darwin-arm64.tar.gz\n")
	if got := sumFor(sums, "node-v1-darwin-arm64.tar.gz"); got != "bbb" {
		t.Fatalf("got %q", got)
	}
	if got := sumFor(sums, "node-v1-linux-arm64.tar.gz"); got != "" {
		t.Fatalf("got %q for a missing name", got)
	}
}
