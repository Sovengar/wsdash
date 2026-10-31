package gitstatus

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func gitAvailable(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
}

func initRepo(t *testing.T) (string, string) {
	t.Helper()
	dir := t.TempDir()
	remote := filepath.Join(t.TempDir(), "remote.git")
	runGit(t, dir, "init", "--bare", remote)
	runGit(t, dir, "init", "-q", "-b", "main")
	runGit(t, dir, "config", "user.email", "t@example.com")
	runGit(t, dir, "config", "user.name", "t")
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-q", "-m", "feat: first commit")
	runGit(t, dir, "remote", "add", "origin", remote)
	runGit(t, dir, "push", "-q", "-u", "origin", "main")
	return dir, remote
}

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v: %s", args, err, out)
	}
	return string(out)
}

func TestStatusCleanRepo(t *testing.T) {
	gitAvailable(t)
	dir, _ := initRepo(t)
	st := Status(context.Background(), dir)
	if st.Err != nil {
		t.Fatalf("err: %v", st.Err)
	}
	if st.Branch != "main" {
		t.Errorf("branch = %q", st.Branch)
	}
	if st.Dirty {
		t.Error("repo should be clean")
	}
	if !strings.Contains(st.LastCommit, "feat: first commit") {
		t.Errorf("last commit = %q", st.LastCommit)
	}
}

func TestStatusDirtyRepo(t *testing.T) {
	gitAvailable(t)
	dir, _ := initRepo(t)
	if err := os.WriteFile(filepath.Join(dir, "b.txt"), []byte("b\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	st := Status(context.Background(), dir)
	if !st.Dirty {
		t.Error("repo should be dirty")
	}
}

func TestStatusAheadOfUpstream(t *testing.T) {
	gitAvailable(t)
	dir, _ := initRepo(t)
	if err := os.WriteFile(filepath.Join(dir, "c.txt"), []byte("c\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-q", "-m", "local only")
	st := Status(context.Background(), dir)
	if !strings.Contains(st.AheadBehind, "↑1") {
		t.Errorf("ahead/behind = %q", st.AheadBehind)
	}
}

func TestStatusMissingPath(t *testing.T) {
	gitAvailable(t)
	st := Status(context.Background(), filepath.Join(t.TempDir(), "absent"))
	if st.Err == nil {
		t.Fatal("want error for missing path")
	}
}
