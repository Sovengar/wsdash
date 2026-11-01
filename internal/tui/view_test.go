package tui

import (
	"fmt"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"wsdash/internal/agents"
	"wsdash/internal/config"
	"wsdash/internal/layout"
	"wsdash/internal/project"
)

func fixtureModel(t *testing.T) Model {
	t.Helper()
	cfg := &config.Config{DefaultBackend: "fake"}
	m := New(cfg, []project.Project{
		{
			Name: "alpha", Path: "/home/u/dev/work/vsocial/alpha", Description: "alpha project",
			StartWS: "api",
			Workspaces: map[string]layout.Workspace{
				"api": {Name: "api", Root: "/home/u/dev/work/vsocial/alpha", Tabs: []layout.Tab{
					{Label: "one", Panes: []layout.Pane{{Command: "nvim"}}},
					{Label: "two", Panes: []layout.Pane{{Command: "nvim"}}},
				}},
			},
		},
		{Name: "beta", Path: "/tmp/beta", Description: "beta project"},
	}, "fake", nil, nil, nil, nil)
	m.status = map[string]project.Status{
		"alpha": {Branch: "main", AheadBehind: "↑1 ↓2", LastCommit: "abc123 fix it"},
		"beta":  {},
	}
	m.agents = map[string][]agents.Agent{
		"alpha": {
			{Agent: "pi", Status: "working", TabID: "w45:t0", PaneID: "w45:p1", AgentSession: agents.SessionRef{Value: "/s/a"}},
			{Agent: "pi", Status: "done", TabID: "w45:t12", PaneID: "w45:p1F", AgentSession: agents.SessionRef{Value: "/s/b"}},
		},
	}
	m.sessionNames = map[string]string{"/s/a": "Add tabs to sidebar rail", "/s/b": "Hybrid review"}
	return m
}

func assertFrame(t *testing.T, out string, w, h int) []string {
	t.Helper()
	plain := ansi.Strip(out)
	lines := strings.Split(plain, "\n")
	if len(lines) != h {
		t.Fatalf("frame has %d lines, want %d:\n%s", len(lines), h, plain)
	}
	for i, line := range lines {
		if got := ansi.StringWidth(line); got > w {
			t.Errorf("line %d measures %d columns, want <= %d: %q", i, got, w, line)
		}
	}
	return lines
}

func TestKeybindsRenderKeysInAccentAndDescriptionsPlain(t *testing.T) {
	keyStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	plainKeys := "enter launch · tab section · f filter · r refresh · L backend · q quit"
	for _, size := range []struct{ w, h int }{{100, 30}, {60, 20}} {
		t.Run(fmt.Sprintf("%dx%d", size.w, size.h), func(t *testing.T) {
			m := fixtureModel(t)
			m.width, m.height = size.w, size.h
			out := renderView(m)

			var keyLine string
			for _, line := range strings.Split(out, "\n") {
				if strings.Contains(ansi.Strip(line), "enter launch") {
					keyLine = line
					break
				}
			}
			if keyLine == "" {
				t.Fatalf("keybind line not found:\n%s", ansi.Strip(out))
			}

			for _, seg := range strings.Split(plainKeys, " · ") {
				key, desc, _ := strings.Cut(seg, " ")
				if !strings.Contains(ansi.Strip(keyLine), key+" ") {
					continue
				}
				if !strings.Contains(keyLine, keyStyle.Render(key)+" ") {
					t.Errorf("segment %q should render key %q in the accent, got %q", seg, key, keyLine)
				}
				if size.w >= 100 && !strings.Contains(keyLine, keyStyle.Render(key)+" "+desc) {
					t.Errorf("segment %q should keep its description plain, got %q", seg, keyLine)
				}
			}
			if size.w >= 100 && !strings.Contains(ansi.Strip(keyLine), plainKeys) {
				t.Errorf("plain keybind text changed: %q", ansi.Strip(keyLine))
			}
		})
	}
}

func TestKeybindsToastStaysUnstyled(t *testing.T) {
	m := fixtureModel(t)
	m.width, m.height = 100, 30
	m.toast = "refreshed"
	out := renderView(m)
	keyStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))

	lines := strings.Split(out, "\n")
	for i, line := range lines {
		if strings.Contains(ansi.Strip(line), " Keybinds ") {
			toastLine := lines[i+2]
			if !strings.Contains(ansi.Strip(toastLine), "refreshed") {
				t.Fatalf("toast line not found below the keybinds: %q", ansi.Strip(toastLine))
			}
			if strings.Contains(toastLine, keyStyle.Render("refreshed")) {
				t.Errorf("toast should keep its current unstyled look: %q", toastLine)
			}
			return
		}
	}
	t.Fatal("keybinds box not found")
}

func TestFrameAt100x30And60x20(t *testing.T) {
	for _, size := range []struct{ w, h int }{{100, 30}, {60, 20}} {
		t.Run(fmt.Sprintf("%dx%d", size.w, size.h), func(t *testing.T) {
			m := fixtureModel(t)
			m.width, m.height = size.w, size.h
			lines := assertFrame(t, renderView(m), size.w, size.h)
			if len(lines) < 9 {
				t.Fatalf("not enough lines to hold detail and agents:\n%s", strings.Join(lines, "\n"))
			}
			if !strings.Contains(lines[8], "Agents · alpha (2)") {
				t.Errorf("agents title is not on line 8: %q", lines[8])
			}
			joined := strings.Join(lines, "\n")
			for _, want := range []string{"Projects", "Keybinds", "alpha", "main", "↳ pi working"} {
				if !strings.Contains(joined, want) {
					t.Errorf("frame is missing %q:\n%s", want, joined)
				}
			}
			if size.w >= 100 {
				for _, want := range []string{"Add tabs to sidebar rail", "w45:t0/w45:p1"} {
					if !strings.Contains(joined, want) {
						t.Errorf("frame is missing %q:\n%s", want, joined)
					}
				}
			}
		})
	}
}

func TestProjectRowShowsBranchAndTruncatesIt(t *testing.T) {
	m := fixtureModel(t)
	m.width, m.height = 60, 20
	m.status["alpha"] = project.Status{Branch: strings.Repeat("feature", 20)}
	lines := assertFrame(t, renderView(m), 60, 20)
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "alpha") {
		t.Fatalf("project name missing:\n%s", joined)
	}
	if !strings.Contains(joined, "…") {
		t.Errorf("long branch should end with an ellipsis:\n%s", joined)
	}
}

func TestProjectRowOmitsBranchWhileStatusUnknown(t *testing.T) {
	m := fixtureModel(t)
	m.width, m.height = 100, 30
	m.status["alpha"] = project.Status{}
	m.agents["alpha"] = nil
	lines := assertFrame(t, renderView(m), 100, 30)
	for _, line := range lines {
		if strings.Contains(line, "> alpha") {
			if strings.TrimSpace(strings.TrimPrefix(strings.Split(line, "│")[1], "> alpha")) != "" {
				t.Errorf("unknown status should leave nothing after the name: %q", line)
			}
			return
		}
	}
	t.Fatal("selected project row not found")
}

func TestFrameWithToastStillFillsTheTerminal(t *testing.T) {
	for _, size := range []struct{ w, h int }{{100, 30}, {60, 20}} {
		m := fixtureModel(t)
		m.width, m.height = size.w, size.h
		m.toast = "refreshed"
		assertFrame(t, renderView(m), size.w, size.h)
	}
}

func TestDetailBoxHasFixedSixInteriorLines(t *testing.T) {
	m := fixtureModel(t)
	m.width, m.height = 100, 30
	lines := assertFrame(t, renderView(m), 100, 30)
	if !strings.Contains(lines[0], "alpha") {
		t.Fatalf("detail title not on the first line: %q", lines[0])
	}
	if !strings.Contains(lines[7], "╰") {
		t.Errorf("detail bottom border should be line 7: %q", lines[7])
	}
	if !strings.Contains(lines[8], "╭ Agents") {
		t.Errorf("agents top border should be line 8: %q", lines[8])
	}
}

func TestAgentsBoxNoAgentsAndClipping(t *testing.T) {
	m := fixtureModel(t)
	m.width, m.height = 60, 20
	m.agents["alpha"] = nil
	lines := assertFrame(t, renderView(m), 60, 20)
	if !strings.Contains(strings.Join(lines, "\n"), "no agents") {
		t.Errorf("empty agents box should say no agents:\n%s", strings.Join(lines, "\n"))
	}

	m = fixtureModel(t)
	m.width, m.height = 60, 20
	var many []agents.Agent
	for i := 0; i < 30; i++ {
		many = append(many, agents.Agent{
			Agent: "pi", Status: "working", TabID: fmt.Sprintf("w45:t%d", i), PaneID: "w45:p1",
			AgentSession: agents.SessionRef{Value: fmt.Sprintf("/s/%d", i)},
		})
	}
	m.agents["alpha"] = many
	m.sessionNames = map[string]string{}
	lines = assertFrame(t, renderView(m), 60, 20)
	// Interior of agents is lines 9..15 at 60x20, then its bottom border.
	if !strings.HasPrefix(strings.TrimSpace(lines[16]), "╰") {
		t.Errorf("agents rows overflowed their box: line 16 = %q", lines[16])
	}
}
