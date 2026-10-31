package tui

import (
	"fmt"
	"strings"
	"testing"

	"wsdash/internal/agents"
	"wsdash/internal/config"
	"wsdash/internal/layout"
	"wsdash/internal/project"
)

const (
	wtParentPath = "/p/parent"
	wtChildPath  = "/p/parent.oct26"
)

func worktreeModel(t *testing.T) Model {
	t.Helper()
	cfg := &config.Config{DefaultBackend: "fake"}
	m := New(cfg, []project.Project{
		{
			Name: "parent", Path: wtParentPath, Description: "parent project", StartWS: "api",
			Workspaces: map[string]layout.Workspace{
				"api": {
					Name: "api", Root: wtParentPath,
					Tabs: []layout.Tab{{Label: "editor", Panes: []layout.Pane{{Command: "nvim"}}}},
				},
			},
		},
		{
			Name: "a-child", Path: wtChildPath, Source: project.SourceScan,
			ParentPath: wtParentPath,
		},
		{Name: "z-other", Path: "/p/z-other"},
	}, "fake", nil, nil, nil, nil)
	m.status = map[string]project.Status{
		"parent":  {Branch: "deploy/SEP26", LastCommit: "sep111 parent commit"},
		"a-child": {Branch: "deploy/OCT26", LastCommit: "oct222 child commit"},
	}
	return m
}

func rowNames(m Model) []string {
	out := make([]string, 0, len(m.rows))
	for _, r := range m.rows {
		out = append(out, m.projects[r.project].Name)
	}
	return out
}

func TestRowsPlaceWorktreeDirectlyUnderParent(t *testing.T) {
	m := worktreeModel(t)
	got := strings.Join(rowNames(m), ",")
	if got != "parent,a-child,z-other" {
		t.Fatalf("rows = %s, want parent,a-child,z-other", got)
	}
}

func TestWorktreeHiddenWhenParentIsNotVisible(t *testing.T) {
	cfg := &config.Config{DefaultBackend: "fake"}
	m := New(cfg, []project.Project{
		{Name: "parent", Path: wtParentPath, Source: project.SourceScan},
		{Name: "a-child", Path: wtChildPath, Source: project.SourceScan, ParentPath: wtParentPath},
		{Name: "z-other", Path: "/p/z-other"},
	}, "fake", nil, nil, nil, nil)

	if got := strings.Join(rowNames(m), ","); got != "z-other" {
		t.Fatalf("rows with scan off = %s, want only z-other", got)
	}
	next, _ := pressKey(m, "f")
	m = next.(Model)
	if got := strings.Join(rowNames(m), ","); got != "parent,a-child,z-other" {
		t.Fatalf("rows with scan on = %s, want parent,a-child,z-other", got)
	}
}

func TestEnterOnWorktreeLaunchesParentWorkspaceAtWorktreeRoot(t *testing.T) {
	m := worktreeModel(t)
	var launched []project.Project
	m.LaunchFunc = func(p project.Project) error {
		launched = append(launched, p)
		return nil
	}
	next, _ := pressKey(m, "j") // parent -> child
	m = next.(Model)
	if m.selectedProjectName() != "a-child" {
		t.Fatalf("selection = %q, want a-child", m.selectedProjectName())
	}
	_, cmd := pressKey(m, "enter")
	if len(launched) != 1 {
		t.Fatalf("launched %d projects, want 1", len(launched))
	}
	got := launched[0]
	if got.Path != wtChildPath {
		t.Fatalf("launched path = %q, want worktree %q", got.Path, wtChildPath)
	}
	if got.StartWS != "api" {
		t.Errorf("launched start workspace = %q, want parent's api", got.StartWS)
	}
	if ws := got.Workspaces["api"]; ws.Root != wtChildPath {
		t.Errorf("workspace root = %q, want worktree %q", ws.Root, wtChildPath)
	}
	if cmd == nil {
		t.Fatal("worktree enter should quit")
	}
}

func TestAgentsAssignToLongestMatchingProjectPath(t *testing.T) {
	m := worktreeModel(t)
	all := []agents.Agent{
		{Agent: "pi", Status: "working", Cwd: wtChildPath + "/src", TabID: "w1:t0", PaneID: "w1:p0"},
		{Agent: "pi", Status: "idle", Cwd: wtParentPath + "/other", TabID: "w2:t0", PaneID: "w2:p0"},
	}
	m.AgentsFunc = func(p project.Project) []agents.Agent {
		return agents.ForProject(all, p.Path)
	}
	next, _ := m.Update(m.Init()())
	m = next.(Model)

	if got := len(m.agents["a-child"]); got != 1 {
		t.Fatalf("worktree has %d agents, want 1 (%v)", got, m.agents)
	}
	if got := m.agents["a-child"][0].TabID; got != "w1:t0" {
		t.Fatalf("worktree agent = %q, want w1:t0", got)
	}
	if got := len(m.agents["parent"]); got != 1 {
		t.Fatalf("parent has %d agents, want 1 (%v)", got, m.agents)
	}
	if got := m.agents["parent"][0].TabID; got != "w2:t0" {
		t.Fatalf("parent agent = %q, want w2:t0", got)
	}
}

func TestWorktreeFrameAt100x30And60x20(t *testing.T) {
	for _, size := range []struct{ w, h int }{{100, 30}, {60, 20}} {
		t.Run(fmt.Sprintf("%dx%d", size.w, size.h), func(t *testing.T) {
			m := worktreeModel(t)
			m.width, m.height = size.w, size.h
			next, _ := pressKey(m, "j") // select the worktree row
			m = next.(Model)
			lines := assertFrame(t, renderView(m), size.w, size.h)
			joined := strings.Join(lines, "\n")
			for _, want := range []string{"└ a-child", "deploy/OCT26", "worktree of parent", "child commit"} {
				if !strings.Contains(joined, want) {
					t.Errorf("frame is missing %q:\n%s", want, joined)
				}
			}
		})
	}
}

func TestProjectListScrollsToSelectedWorktree(t *testing.T) {
	projs := []project.Project{
		{
			Name: "parent", Path: wtParentPath, StartWS: "api",
			Workspaces: map[string]layout.Workspace{
				"api": {Name: "api", Root: wtParentPath},
			},
		},
		{Name: "a-child", Path: wtChildPath, Source: project.SourceScan, ParentPath: wtParentPath},
		{Name: "z-other", Path: "/p/z-other"},
	}
	for i := 0; i < 25; i++ {
		name := fmt.Sprintf("a%02d", i)
		projs = append(projs, project.Project{Name: name, Path: "/p/" + name})
	}
	m := New(&config.Config{DefaultBackend: "fake"}, projs, "fake", nil, nil, nil, nil)
	m.status = map[string]project.Status{"a-child": {Branch: "deploy/OCT26"}}
	m.width, m.height = 60, 20

	next, _ := pressKey(m, "G")
	m = next.(Model)
	next, _ = pressKey(m, "k")
	m = next.(Model)
	if m.selectedProjectName() != "a-child" {
		t.Fatalf("selection = %q, want a-child", m.selectedProjectName())
	}
	lines := assertFrame(t, renderView(m), 60, 20)
	joined := strings.Join(lines, "\n")
	for _, want := range []string{"└ a-child", "worktree of parent"} {
		if !strings.Contains(joined, want) {
			t.Errorf("scrolled frame is missing %q:\n%s", want, joined)
		}
	}
}
