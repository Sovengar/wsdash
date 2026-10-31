package discovery

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func fakeGit(common, worktreeList string, err error) Runner {
	return func(dir string, args ...string) (string, error) {
		if err != nil {
			return "", err
		}
		key := strings.Join(args, " ")
		switch key {
		case "rev-parse --git-common-dir":
			return common, nil
		case "worktree list --porcelain":
			return worktreeList, nil
		default:
			return "", errors.New("unexpected git args: " + key)
		}
	}
}

func TestCanonicalCheckoutIsNotAWorktree(t *testing.T) {
	repo := filepath.Join(t.TempDir(), "repo")
	run := fakeGit(".git", "", nil)
	if main, ok := IsLinkedWorktreeWithRunner(repo, run); ok || main != "" {
		t.Fatalf("canonical checkout reported as worktree: main=%q ok=%v", main, ok)
	}

	run = fakeGit(filepath.Join(repo, ".git"), "", nil)
	if main, ok := IsLinkedWorktreeWithRunner(repo, run); ok || main != "" {
		t.Fatalf("absolute canonical .git reported as worktree: main=%q ok=%v", main, ok)
	}
}

func TestLinkedWorktreeReturnsMainPath(t *testing.T) {
	repo := filepath.Join(t.TempDir(), "repo")
	wt := filepath.Join(t.TempDir(), "linked")
	list := "worktree " + repo + "\nHEAD abc\nbranch refs/heads/main\n\nworktree " + wt + "\nHEAD def\nbranch refs/heads/feature\n"
	run := fakeGit(filepath.Join(repo, ".git"), list, nil)

	main, ok := IsLinkedWorktreeWithRunner(wt, run)
	if !ok {
		t.Fatal("linked worktree not detected")
	}
	if main != repo {
		t.Fatalf("main = %q, want %q", main, repo)
	}
}

func TestWorktreeDetectionIsQuietOnGitErrors(t *testing.T) {
	repo := t.TempDir()
	cases := []struct {
		name string
		run  Runner
	}{
		{name: "rev-parse fails", run: fakeGit("", "", errors.New("not a repo"))},
		{name: "worktree list fails", run: func(dir string, args ...string) (string, error) {
			if strings.Join(args, " ") == "rev-parse --git-common-dir" {
				return "/somewhere/else/.git", nil
			}
			return "", errors.New("boom")
		}},
		{name: "nil runner", run: nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if main, ok := IsLinkedWorktreeWithRunner(repo, tc.run); ok || main != "" {
				t.Fatalf("got main=%q ok=%v, want quiet false", main, ok)
			}
		})
	}
}

func TestRealGitWorktreeIsDetected(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	root := t.TempDir()
	repo := filepath.Join(root, "repo")
	linked := filepath.Join(root, "linked")
	runGit := func(dir string, args ...string) (string, error) {
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		out, err := cmd.CombinedOutput()
		return string(out), err
	}
	for _, args := range [][]string{
		{"init", "-q", repo},
		{"-C", repo, "config", "user.email", "test@example.com"},
		{"-C", repo, "config", "user.name", "Test"},
	} {
		if out, err := runGit(root, args...); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	if err := os.WriteFile(filepath.Join(repo, "README"), []byte("fixture\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := runGit(repo, "add", "README"); err != nil {
		t.Fatalf("git add: %v\n%s", err, out)
	}
	if out, err := runGit(repo, "commit", "-qm", "initial"); err != nil {
		t.Fatalf("git commit: %v\n%s", err, out)
	}
	if out, err := runGit(repo, "worktree", "add", "-q", linked); err != nil {
		t.Fatalf("git worktree add: %v\n%s", err, out)
	}

	if main, ok := IsLinkedWorktree(repo); ok || main != "" {
		t.Fatalf("main checkout reported as worktree: main=%q ok=%v", main, ok)
	}
	main, ok := IsLinkedWorktree(linked)
	if !ok {
		t.Fatal("real linked worktree not detected")
	}
	if samePath(main, repo) == false {
		t.Fatalf("main = %q, want %q", main, repo)
	}
}
