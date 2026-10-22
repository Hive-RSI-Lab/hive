package sandbox

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// Every agent finds what the Hive installed on PATH: inside the sandbox from
// InsideEnv, and on host paths (bare, Seatbelt) from the tools folder's mount.
func TestToolsOnPath(t *testing.T) {
	var path string
	for _, kv := range InsideEnv("/hive/home", false) {
		if v, ok := strings.CutPrefix(kv, "PATH="); ok {
			path = v
		}
	}
	for _, n := range ToolNames {
		if !strings.Contains(path, "/hive/tools/"+n+"/bin:") {
			t.Fatalf("PATH %q lacks %s", path, n)
		}
	}
	folder := filepath.Join(t.TempDir(), "tools")
	dirs := pathDirs(Spec{Mounts: []Mount{{Host: "/elsewhere", Inside: "/hive/kb"}, {Host: folder, Inside: ToolsInside}}})
	if !slices.Contains(dirs, filepath.Join(folder, "python", "bin")) || slices.Contains(dirs, filepath.Join("/elsewhere", "python", "bin")) {
		t.Fatalf("host PATH dirs: %q", dirs)
	}
}
