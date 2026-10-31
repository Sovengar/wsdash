package discovery

import (
	"os"
	"path/filepath"
	"testing"
)

func mk(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}

func TestScanFindsReposAndRespectsDepthAndExclude(t *testing.T) {
	root := t.TempDir()
	mk(t, filepath.Join(root, "a", ".git"))
	mk(t, filepath.Join(root, "group", "b", ".git"))
	mk(t, filepath.Join(root, "group", "b", "nested", ".git"))
	mk(t, filepath.Join(root, "group", "b", "node_modules", ".git"))
	mk(t, filepath.Join(root, "deep", "1", "2", "3", ".git"))
	mk(t, filepath.Join(root, "plain"))

	got := Scan([]string{root}, 2, nil)
	want := map[string]bool{
		filepath.Join(root, "a"):          true,
		filepath.Join(root, "group", "b"): true,
	}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for _, g := range got {
		if !want[g] {
			t.Errorf("unexpected %q", g)
		}
	}

	filtered := Scan([]string{root}, 3, []string{"node_modules"})
	for _, g := range filtered {
		if filepath.Base(g) == "node_modules" {
			t.Errorf("excluded dir returned: %q", g)
		}
	}

	deep := Scan([]string{root}, 4, nil)
	found := false
	for _, g := range deep {
		if g == filepath.Join(root, "deep", "1", "2", "3") {
			found = true
		}
	}
	if !found {
		t.Errorf("depth 4 should reach the deep repo, got %v", deep)
	}
}

func TestScanMissingRootIsQuiet(t *testing.T) {
	got := Scan([]string{filepath.Join(t.TempDir(), "absent")}, 3, nil)
	if len(got) != 0 {
		t.Errorf("got %v, want none", got)
	}
}

func TestName(t *testing.T) {
	if got := Name("/home/x/dev/projects/wsdash"); got != "wsdash" {
		t.Errorf("got %q", got)
	}
}
