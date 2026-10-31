package main

import (
	"errors"
	"strings"
	"testing"

	"wsdash/internal/config"
	"wsdash/internal/discovery"
	"wsdash/internal/project"
)

type fakeRepo struct {
	common string
	list   string
}

func fakeGitRepos(repos map[string]fakeRepo) discovery.Runner {
	return func(dir string, args ...string) (string, error) {
		r, ok := repos[dir]
		if !ok {
			return "", errors.New("not a repository")
		}
		switch strings.Join(args, " ") {
		case "rev-parse --git-common-dir":
			return r.common, nil
		case "worktree list --porcelain":
			return r.list, nil
		default:
			return "", errors.New("unexpected git args")
		}
	}
}

func configWithProjects(projects map[string]config.RawProject) *config.Config {
	return &config.Config{Projects: projects}
}

func findProject(t *testing.T, projs []project.Project, name string) project.Project {
	t.Helper()
	for _, p := range projs {
		if p.Name == name {
			return p
		}
	}
	t.Fatalf("project %q not found in %v", name, projs)
	return project.Project{}
}

func TestResolveNestsWorktreeUnderConfiguredParent(t *testing.T) {
	parentPath := "/projects/vsocial"
	childPath := "/projects/vsocial.oct26"
	cfg := configWithProjects(map[string]config.RawProject{
		"vsocial": {Path: parentPath},
		"oct26":   {Path: childPath},
	})
	run := fakeGitRepos(map[string]fakeRepo{
		childPath: {
			common: parentPath + "/.git",
			list:   "worktree " + parentPath + "\nHEAD abc\nbranch refs/heads/main\n\nworktree " + childPath + "\n",
		},
	})

	projs := resolveWithRunner(cfg, true, run)
	child := findProject(t, projs, "oct26")
	if child.ParentPath != parentPath {
		t.Fatalf("child ParentPath = %q, want %q", child.ParentPath, parentPath)
	}
	if !child.IsWorktree() {
		t.Fatal("child should report IsWorktree")
	}
	if parent := findProject(t, projs, "vsocial"); parent.ParentPath != "" {
		t.Fatalf("parent should not be a worktree, got ParentPath %q", parent.ParentPath)
	}
}

func TestResolveNestsWorktreeByBasenameWhenMainCheckoutMissing(t *testing.T) {
	parentPath := "/projects/vsocial"
	childPath := "/projects/vsocial.oct26"
	mainPath := "/somewhere/else/vsocial"
	cfg := configWithProjects(map[string]config.RawProject{
		"vsocial": {Path: parentPath},
		"oct26":   {Path: childPath},
	})
	run := fakeGitRepos(map[string]fakeRepo{
		childPath: {
			common: mainPath + "/.git",
			list:   "worktree " + mainPath + "\nHEAD abc\nbranch refs/heads/main\n\nworktree " + childPath + "\n",
		},
	})

	projs := resolveWithRunner(cfg, true, run)
	child := findProject(t, projs, "oct26")
	if child.ParentPath != parentPath {
		t.Fatalf("child ParentPath = %q, want basename match %q", child.ParentPath, parentPath)
	}
}

func TestResolveLeavesOrphanWorktreeAsNormalProject(t *testing.T) {
	parentPath := "/projects/unrelated"
	childPath := "/projects/orphan.oct26"
	mainPath := "/somewhere/else/orphan"
	cfg := configWithProjects(map[string]config.RawProject{
		"unrelated": {Path: parentPath},
		"orphan":    {Path: childPath},
	})
	run := fakeGitRepos(map[string]fakeRepo{
		childPath: {
			common: mainPath + "/.git",
			list:   "worktree " + mainPath + "\n",
		},
	})

	projs := resolveWithRunner(cfg, true, run)
	child := findProject(t, projs, "orphan")
	if child.ParentPath != "" {
		t.Fatalf("orphan ParentPath = %q, want empty", child.ParentPath)
	}
	if child.IsWorktree() {
		t.Fatal("orphan should stay an ordinary project")
	}
	if len(projs) != 2 {
		t.Fatalf("configured projects were dropped: %v", projs)
	}
}
