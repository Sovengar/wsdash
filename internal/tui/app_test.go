package tui

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"wsdash/internal/agents"
	"wsdash/internal/config"
	"wsdash/internal/project"
)

func TestNew(t *testing.T) {
	cfg := &config.Config{DefaultBackend: "fake"}
	m := New(cfg, []project.Project{{Name: "p", Path: "/p"}}, "fake", nil, nil, nil, nil)
	if m.backend != "fake" {
		t.Fatal("backend")
	}
}

func pressKey(m Model, s string) (tea.Model, tea.Cmd) {
	var msg tea.KeyMsg
	switch s {
	case "enter":
		msg = tea.KeyPressMsg{Code: tea.KeyEnter}
	case "tab":
		msg = tea.KeyPressMsg{Code: tea.KeyTab}
	case "shift+tab":
		msg = tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift}
	case "left":
		msg = tea.KeyPressMsg{Code: tea.KeyLeft}
	case "right":
		msg = tea.KeyPressMsg{Code: tea.KeyRight}
	case "up":
		msg = tea.KeyPressMsg{Code: tea.KeyUp}
	case "down":
		msg = tea.KeyPressMsg{Code: tea.KeyDown}
	default:
		r := []rune(s)[0]
		msg = tea.KeyPressMsg{Code: r, Text: s}
	}
	return m.Update(msg)
}

func agentModel(t *testing.T) Model {
	t.Helper()
	cfg := &config.Config{DefaultBackend: "fake"}
	m := New(cfg, []project.Project{
		{Name: "alpha", Path: "/a"},
		{Name: "beta", Path: "/b"},
	}, "fake", nil, nil, nil, nil)
	m.agents = map[string][]agents.Agent{
		"alpha": {
			{Agent: "pi", Status: "working", TabID: "w0:t0", AgentSession: agents.SessionRef{Value: "/s/0"}},
			{Agent: "pi", Status: "idle", TabID: "w0:t1", AgentSession: agents.SessionRef{Value: "/s/1"}},
		},
		"beta": {{Agent: "pi", Status: "done", TabID: "w1:t0", AgentSession: agents.SessionRef{Value: "/s/2"}}},
	}
	m.rebuildRows()
	return m
}

func TestFocusAndMovementIsPerColumn(t *testing.T) {
	m := agentModel(t)
	next, _ := pressKey(m, "l")
	mm := next.(Model)
	if mm.focus != focusAgents {
		t.Fatal("l should focus the agents column")
	}
	next, _ = pressKey(mm, "j")
	mm = next.(Model)
	if mm.cursor != 0 || mm.agentCursor != 1 {
		t.Errorf("j in agents moved project=%d agent=%d, want 0/1", mm.cursor, mm.agentCursor)
	}
	next, _ = pressKey(mm, "h")
	mm = next.(Model)
	if mm.focus != focusProjects {
		t.Fatal("h should focus projects")
	}
	next, _ = pressKey(mm, "j")
	mm = next.(Model)
	if mm.cursor != 1 || mm.agentCursor != 0 {
		t.Errorf("j in projects moved project=%d agent=%d, want 1/0", mm.cursor, mm.agentCursor)
	}
	next, _ = pressKey(mm, "G")
	mm = next.(Model)
	if mm.cursor != 1 {
		t.Errorf("G selected project %d, want 1", mm.cursor)
	}
	next, _ = pressKey(mm, "g")
	mm = next.(Model)
	if mm.cursor != 0 {
		t.Errorf("g selected project %d, want 0", mm.cursor)
	}
}

func TestTabSwitchesSectionAndFTogglesFilter(t *testing.T) {
	m := New(&config.Config{DefaultBackend: "fake"}, []project.Project{
		{Name: "alpha", Path: "/a"},
		{Name: "beta", Path: "/b", Source: project.SourceScan},
	}, "fake", nil, nil, nil, nil)
	if m.focus != focusProjects {
		t.Fatal("initial focus should be projects")
	}
	next, _ := pressKey(m, "tab")
	m = next.(Model)
	if m.focus != focusAgents {
		t.Fatal("tab should focus agents")
	}
	next, _ = pressKey(m, "shift+tab")
	m = next.(Model)
	if m.focus != focusProjects {
		t.Fatal("shift+tab should focus projects")
	}

	if got := strings.Join(rowNames(m), ","); got != "alpha" {
		t.Fatalf("rows before filter toggle = %s, want alpha", got)
	}
	next, _ = pressKey(m, "f")
	m = next.(Model)
	if !m.showScan {
		t.Fatal("f should show scanned projects")
	}
	if got := strings.Join(rowNames(m), ","); got != "alpha,beta" {
		t.Fatalf("rows after filter toggle = %s, want alpha,beta", got)
	}
	next, _ = pressKey(m, "f")
	m = next.(Model)
	if m.showScan {
		t.Fatal("f should hide scanned projects again")
	}
}

func TestExitReasonRecordsQuitLaunchAndJump(t *testing.T) {
	t.Run("q quits", func(t *testing.T) {
		m := agentModel(t)
		next, _ := pressKey(m, "q")
		m = next.(Model)
		if m.ExitReason == nil || *m.ExitReason != ExitQuit {
			t.Fatalf("exit reason = %v, want quit", m.ExitReason)
		}
	})

	t.Run("successful launch", func(t *testing.T) {
		m := agentModel(t)
		m.LaunchFunc = func(p project.Project) error { return nil }
		next, _ := pressKey(m, "enter")
		m = next.(Model)
		if m.ExitReason == nil || *m.ExitReason != ExitLaunched {
			t.Fatalf("exit reason = %v, want launched", m.ExitReason)
		}
	})

	t.Run("agent jump is not a launch", func(t *testing.T) {
		m := agentModel(t)
		m.JumpFunc = func(a agents.Agent) error { return nil }
		next, _ := pressKey(m, "l")
		next, _ = pressKey(next.(Model), "enter")
		m = next.(Model)
		if m.ExitReason == nil || *m.ExitReason != ExitQuit {
			t.Fatalf("exit reason = %v, want quit", m.ExitReason)
		}
	})
}

func TestFailedLaunchKeepsTUIAndDoesNotReportLaunched(t *testing.T) {
	m := agentModel(t)
	m.LaunchFunc = func(p project.Project) error { return errors.New("backend down") }
	next, cmd := pressKey(m, "enter")
	m = next.(Model)
	if cmd != nil {
		t.Fatal("failed launch should not quit")
	}
	if m.ExitReason == nil || *m.ExitReason == ExitLaunched {
		t.Fatal("failed launch must not report launched")
	}
	if !strings.Contains(m.toast, "launch failed") {
		t.Fatalf("toast = %q, want launch failure", m.toast)
	}
}

func TestEnterInProjectsLaunchesAndEnterInAgentsJumps(t *testing.T) {
	var launched []project.Project
	var jumped []agents.Agent
	m := agentModel(t)
	m.LaunchFunc = func(p project.Project) error { launched = append(launched, p); return nil }
	m.JumpFunc = func(a agents.Agent) error { jumped = append(jumped, a); return nil }

	_, cmd := pressKey(m, "enter")
	if len(launched) != 1 || launched[0].Name != "alpha" {
		t.Fatalf("project enter launched %v", launched)
	}
	if cmd == nil {
		t.Fatal("project enter should quit")
	}

	next, _ := pressKey(m, "l")
	next, _ = pressKey(next.(Model), "j")
	_, cmd = pressKey(next.(Model), "enter")
	if len(jumped) != 1 || jumped[0].TabID != "w0:t1" {
		t.Fatalf("agent enter jumped %v", jumped)
	}
	if cmd == nil {
		t.Fatal("agent enter should quit")
	}
}

func TestAgentCursorResetsAndClamps(t *testing.T) {
	m := agentModel(t)
	next, _ := pressKey(m, "l")
	next, _ = pressKey(next.(Model), "j")
	mm := next.(Model)
	if mm.agentCursor != 1 {
		t.Fatalf("agent cursor = %d, want 1", mm.agentCursor)
	}
	next, _ = pressKey(mm, "h")
	next, _ = pressKey(next.(Model), "j")
	mm = next.(Model)
	if mm.agentCursor != 0 {
		t.Errorf("changing project should reset agent cursor, got %d", mm.agentCursor)
	}

	// Same project, fewer agents after a refresh: clamp, don't stay out of range.
	next, _ = pressKey(mm, "l")
	next, _ = pressKey(next.(Model), "j")
	mm = next.(Model)
	data := refreshDataMsg{agents: map[string][]agents.Agent{
		"alpha": {{Agent: "pi", Status: "working"}},
		"beta":  {},
	}}
	next, _ = mm.Update(data)
	mm = next.(Model)
	if mm.agentCursor != 0 {
		t.Errorf("agent cursor should clamp to 0, got %d", mm.agentCursor)
	}
}

func TestSessionNamesResolvedInRefreshNotRender(t *testing.T) {
	m := agentModel(t)
	calls := 0
	m.AgentsFunc = func(p project.Project) []agents.Agent {
		if p.Name == "alpha" {
			return m.agents[p.Name][:1]
		}
		return nil
	}
	m.SessionNameFunc = func(a agents.Agent) string {
		calls++
		return "session name"
	}
	msg := m.Init()()
	next, _ := m.Update(msg)
	mm := next.(Model)
	if calls != 1 {
		t.Fatalf("SessionNameFunc called %d times during refresh, want 1", calls)
	}
	mm.width, mm.height = 100, 30
	out := renderView(mm)
	if calls != 1 {
		t.Errorf("render did IO: SessionNameFunc called %d times", calls)
	}
	if !strings.Contains(out, "session name") {
		t.Errorf("resolved session name missing from agents box:\n%s", out)
	}
}
