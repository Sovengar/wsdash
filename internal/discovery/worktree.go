package discovery

import (
	"os/exec"
	"path/filepath"
	"strings"
)

// Runner executes one git invocation in dir and returns its stdout. Tests
// inject a stub so the pure detection logic does not need a real repository.
type Runner func(dir string, args ...string) (string, error)

// GitRunner is the real runner used by IsLinkedWorktree.
var GitRunner Runner = func(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	b, err := cmd.Output()
	return string(b), err
}

// IsLinkedWorktree reports whether path is a linked git worktree. When it is,
// mainPath is the path of the repository's main worktree (the first entry of
// `git worktree list --porcelain`). Any error, missing git, or non-repository
// path is reported as ("", false).
func IsLinkedWorktree(path string) (mainPath string, ok bool) {
	return IsLinkedWorktreeWithRunner(path, GitRunner)
}

// IsLinkedWorktreeWithRunner is IsLinkedWorktree with an injected git runner.
func IsLinkedWorktreeWithRunner(path string, run Runner) (mainPath string, ok bool) {
	if run == nil || path == "" {
		return "", false
	}

	out, err := run(path, "rev-parse", "--git-common-dir")
	if err != nil {
		return "", false
	}
	common := strings.TrimSpace(out)
	if common == "" {
		return "", false
	}
	common = resolveGitPath(path, common)
	if samePath(common, filepath.Join(path, ".git")) {
		return "", false
	}

	out, err = run(path, "worktree", "list", "--porcelain")
	if err != nil {
		return "", false
	}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimRight(line, "\r")
		if !strings.HasPrefix(line, "worktree ") {
			continue
		}
		main := strings.TrimSpace(strings.TrimPrefix(line, "worktree "))
		if main == "" {
			return "", false
		}
		if !filepath.IsAbs(main) {
			main = filepath.Join(path, main)
		}
		return filepath.Clean(main), true
	}
	return "", false
}

func resolveGitPath(base, p string) string {
	p = strings.TrimSpace(p)
	if !filepath.IsAbs(p) {
		p = filepath.Join(base, p)
	}
	return filepath.Clean(p)
}

func samePath(a, b string) bool {
	a, b = filepath.Clean(a), filepath.Clean(b)
	if a == b {
		return true
	}
	ae, aerr := filepath.EvalSymlinks(a)
	be, berr := filepath.EvalSymlinks(b)
	return aerr == nil && berr == nil && filepath.Clean(ae) == filepath.Clean(be)
}
