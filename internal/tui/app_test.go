package tui

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"wsdash/internal/config"
	"wsdash/internal/layout"
	"wsdash/internal/project"
)

func newModel(t *testing.T, projs []project.Project, launches *[]string) Model {
	t.Helper()
	cfg := &config.Config{DefaultBackend: "fake", BackendOrder: []string{"fake", "herdr"}}
	return New(cfg, projs, "fake",
		func(p project.Project, backendName string) error {
			*launches = append(*launches, p.Name+"@"+backendName)
			return nil
		},
		func(p project.Project) project.Status {
			return project.Status{Branch: "main", Dirty: p.Name == "dirty"}
		},
	)
}

func key(s string) tea.KeyPressMsg {
	switch s {
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "tab":
		return tea.KeyPressMsg{Code: tea.KeyTab}
	case "down":
		return tea.KeyPressMsg{Code: tea.KeyDown}
	default:
		return tea.KeyPressMsg{Code: rune(s[0]), Text: s}
	}
}

func sample() []project.Project {
	return []project.Project{
		{Name: "beta", Path: "/b", Source: project.SourceConfig},
		{Name: "alpha", Path: "/a", Source: project.SourceConfig},
		{Name: "dirty", Path: "/d", Source: project.SourceConfig},
		{Name: "scanned", Path: "/s", Source: project.SourceScan},
	}
}

func TestVisibleHidesScannedUntilTab(t *testing.T) {
	var launches []string
	m := newModel(t, sample(), &launches)
	if got := len(m.Visible()); got != 3 {
		t.Fatalf("visible = %d, want 3", got)
	}
	next, _ := m.Update(key("tab"))
	m = next.(Model)
	if got := len(m.Visible()); got != 4 {
		t.Fatalf("after tab visible = %d, want 4", got)
	}
	next, _ = m.Update(key("tab"))
	m = next.(Model)
	if got := len(m.Visible()); got != 3 {
		t.Fatalf("tab twice visible = %d, want 3", got)
	}
}

func TestListIsSortedByName(t *testing.T) {
	var launches []string
	m := newModel(t, sample(), &launches)
	var names []string
	for _, p := range m.Visible() {
		names = append(names, p.Name)
	}
	if strings.Join(names, ",") != "alpha,beta,dirty" {
		t.Fatalf("order = %v", names)
	}
}

func TestCursorMovesAndIsClamped(t *testing.T) {
	var launches []string
	m := newModel(t, sample(), &launches)
	for i := 0; i < 10; i++ {
		next, _ := m.Update(key("j"))
		m = next.(Model)
	}
	if m.cursor != 2 {
		t.Fatalf("cursor = %d, want 2", m.cursor)
	}
	for i := 0; i < 10; i++ {
		next, _ := m.Update(key("k"))
		m = next.(Model)
	}
	if m.cursor != 0 {
		t.Fatalf("cursor = %d, want 0", m.cursor)
	}
}

func TestEnterLaunchesSelectionOnce(t *testing.T) {
	var launches []string
	m := newModel(t, sample(), &launches)
	_, cmd := m.Update(key("enter"))
	if cmd == nil {
		t.Fatal("enter returned no command")
	}
	msg := cmd()
	lm, ok := msg.(launchMsg)
	if !ok {
		t.Fatalf("got %T", msg)
	}
	next, _ := m.Update(lm)
	m = next.(Model)
	if len(launches) != 1 || launches[0] != "alpha@fake" {
		t.Fatalf("launches = %v", launches)
	}
	if !strings.Contains(m.toast, "launched alpha") {
		t.Fatalf("toast = %q", m.toast)
	}
}

func TestLaunchErrorIsShown(t *testing.T) {
	cfg := &config.Config{DefaultBackend: "fake"}
	m := New(cfg, sample(), "fake",
		func(project.Project, string) error { return errors.New("boom") },
		func(project.Project) project.Status { return project.Status{} },
	)
	next, _ := m.Update(launchMsg{name: "alpha", err: errors.New("boom")})
	m = next.(Model)
	if !m.toastErr || !strings.Contains(m.toast, "boom") {
		t.Fatalf("toast = %q (err=%v)", m.toast, m.toastErr)
	}
}

func TestStatusMsgFillsDetail(t *testing.T) {
	var launches []string
	m := newModel(t, sample(), &launches)
	next, _ := m.Update(statusMsg{name: "alpha", st: project.Status{Branch: "feat", Dirty: true}})
	m = next.(Model)
	view := m.View().Content
	if !strings.Contains(view, "feat") {
		t.Fatalf("view misses branch: %s", view)
	}
	if !strings.Contains(view, "dirty") {
		t.Fatalf("view misses state: %s", view)
	}
}

func TestBackendCycle(t *testing.T) {
	var launches []string
	m := newModel(t, sample(), &launches)
	next, _ := m.Update(key("L"))
	m = next.(Model)
	if m.backend != "herdr" {
		t.Fatalf("backend = %q", m.backend)
	}
}

func TestRefreshReturnsCommands(t *testing.T) {
	var launches []string
	m := newModel(t, sample(), &launches)
	_, cmd := m.Update(key("r"))
	if cmd == nil {
		t.Fatal("refresh returned no command")
	}
	if len(m.loading) == 0 {
		t.Fatal("refresh did not mark projects loading")
	}
}

func TestViewMentionsKeysAndHiddenCount(t *testing.T) {
	var launches []string
	m := newModel(t, sample(), &launches)
	view := m.View().Content
	for _, want := range []string{"wsdash", "PROJECTS (3)", "+1 scanned", "enter launch"} {
		if !strings.Contains(view, want) {
			t.Errorf("view misses %q", want)
		}
	}
}

func TestLayoutSummary(t *testing.T) {
	p := project.Project{Workspaces: map[string]layout.Workspace{
		"dev": {Tabs: []layout.Tab{{Label: "editor", Panes: []layout.Pane{{}, {}}}, {Label: "server", Panes: []layout.Pane{{}}}}},
	}}
	if got := layoutSummary(p); got != "editor(2) + server(1)" {
		t.Errorf("got %q", got)
	}
	if got := layoutSummary(project.Project{Launch: "vroom"}); got != "shell: vroom" {
		t.Errorf("got %q", got)
	}
	if got := layoutSummary(project.Project{}); got != "no layout" {
		t.Errorf("got %q", got)
	}
}

func TestEmptyProjectListIsSafe(t *testing.T) {
	var launches []string
	m := newModel(t, nil, &launches)
	_, cmd := m.Update(key("enter"))
	if cmd != nil {
		t.Fatal("enter on empty list should not launch")
	}
	m.View()
}
