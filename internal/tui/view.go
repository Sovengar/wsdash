package tui

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"wsdash/internal/agents"
	"wsdash/internal/project"
	"wsdash/internal/tui/bordered"
)

const detailInteriorHeight = 6

func renderView(m Model) string {
	w := m.width
	if w == 0 {
		w = 80
	}
	h := m.height
	if h == 0 {
		h = 24
	}
	sidebarW := w * 30 / 100
	if sidebarW < 22 {
		sidebarW = 22
	}
	if sidebarW > w-10 {
		sidebarW = w - 10
	}
	if sidebarW < 2 {
		sidebarW = 2
	}
	detailW := w - sidebarW
	if detailW < 2 {
		detailW = 2
	}
	home, _ := os.UserHomeDir()

	// keybinds
	keys := "enter launch · h/l column · tab hidden · r refresh · L backend · q quit"
	kbContent := keys
	if m.toast != "" {
		kbContent += "\n" + m.toast
	}
	keyBoxHeight := 3
	if m.toast != "" {
		keyBoxHeight = 4
	}
	kbContent = truncateLines(kbContent, w-2)
	kbContent = padTrim(kbContent, keyBoxHeight-2)

	avail := h - keyBoxHeight
	if avail < 4 {
		avail = 4
	}
	detailBoxHeight := detailInteriorHeight + 2
	agentsBoxHeight := avail - detailBoxHeight
	if agentsBoxHeight < 3 {
		agentsBoxHeight = 3
	}
	topHeight := detailBoxHeight + agentsBoxHeight

	// Projects content
	var sb strings.Builder
	for _, r := range m.rows {
		p := m.projects[r.project]
		st := m.status[p.Name]
		marker := agentBadge(m.agents[p.Name])
		selected := len(m.rows) > 0 && m.cursor < len(m.rows) && m.rows[m.cursor].project == r.project
		prefix := "  "
		if selected {
			prefix = "> "
		}
		line := prefix + p.Name
		if st.Branch != "" {
			room := sidebarW - 2 - ansi.StringWidth(line) - ansi.StringWidth(marker) - 2
			if room > 0 {
				branch := ansi.Truncate(st.Branch, room, "…")
				line += "  " + lipgloss.NewStyle().Faint(true).Render(branch)
			}
		}
		line += marker
		sb.WriteString(line + "\n")
	}
	if !m.showScan {
		n := 0
		for _, p := range m.projects {
			if p.Source == project.SourceScan {
				n++
			}
		}
		if n > 0 {
			sb.WriteString(fmt.Sprintf("+%d scanned (tab)\n", n))
		}
	}
	sidebarContent := padTrim(truncateLines(strings.TrimRight(sb.String(), "\n"), sidebarW-2), topHeight-2)

	// Detail shape: fixed six rows.
	p, hasProject := m.selectedProject()
	var detailLines []string
	agentCount := 0
	projectName := ""
	if hasProject {
		projectName = p.Name
		st := m.status[p.Name]
		state := "clean"
		if st.Dirty {
			state = "dirty"
		}
		if st.AheadBehind != "" {
			state += " " + st.AheadBehind
		}
		path := shortPath(p.Path, home)
		detailLines = []string{
			p.Description,
			"path " + path,
			"state " + state,
			"last " + st.LastCommit,
			"backend " + m.backend,
			layoutSummary(p),
		}
		agentCount = len(m.agents[p.Name])
	}
	detailContent := padTrim(truncateLines(strings.Join(detailLines, "\n"), detailW-2), detailInteriorHeight)

	// Agents content
	var ab strings.Builder
	if hasProject {
		ags := m.agents[p.Name]
		if len(ags) == 0 {
			ab.WriteString(lipgloss.NewStyle().Faint(true).Render("no agents") + "\n")
		}
		for _, a := range ags {
			line := fmt.Sprintf("↳ %s %s", a.Agent, styledStatus(a.Status))
			if name := m.sessionNames[a.AgentSession.Value]; name != "" {
				line += " " + name
			}
			line += " " + a.TabID + "/" + a.PaneID
			ab.WriteString(line + "\n")
		}
	} else {
		ab.WriteString(lipgloss.NewStyle().Faint(true).Render("no agents") + "\n")
	}
	agentsContent := padTrim(truncateLines(strings.TrimRight(ab.String(), "\n"), detailW-2), agentsBoxHeight-2)

	borderFg := lipgloss.Color("8")
	projFg := borderFg
	if m.focus == focusProjects {
		projFg = lipgloss.Color("12")
	}
	agentsFg := borderFg
	if m.focus == focusAgents {
		agentsFg = lipgloss.Color("12")
	}

	projectsBox := bordered.RenderWithTitleEx(lipgloss.RoundedBorder(), projFg, bordered.AlignLeft, " Projects ", sidebarContent, sidebarW)
	detailTitle := " detail "
	if projectName != "" {
		detailTitle = " " + projectName + " "
	}
	detailBox := bordered.RenderWithTitleEx(lipgloss.RoundedBorder(), borderFg, bordered.AlignLeft, detailTitle, detailContent, detailW)
	agentsTitle := " Agents (0) "
	if projectName != "" {
		agentsTitle = fmt.Sprintf(" Agents · %s (%d) ", projectName, agentCount)
	}
	agentsBox := bordered.RenderWithTitleEx(lipgloss.RoundedBorder(), agentsFg, bordered.AlignLeft, agentsTitle, agentsContent, detailW)

	right := lipgloss.JoinVertical(lipgloss.Left, detailBox, agentsBox)
	top := lipgloss.JoinHorizontal(lipgloss.Top, projectsBox, right)
	keyBox := bordered.RenderWithTitleEx(lipgloss.RoundedBorder(), borderFg, bordered.AlignLeft, " Keybinds ", kbContent, w)

	out := lipgloss.JoinVertical(lipgloss.Left, top, keyBox)
	_ = agents.Agent{}
	return out
}

func agentBadge(ags []agents.Agent) string {
	if len(ags) == 0 {
		return ""
	}
	worst := "idle"
	for _, a := range ags {
		if a.Status == "working" {
			worst = "working"
		}
		if a.Status == "blocked" {
			worst = "blocked"
		}
	}
	sym := "✓"
	if worst == "working" {
		sym = "●"
	}
	if worst == "blocked" {
		sym = "!"
	}
	return fmt.Sprintf("  %d%s", len(ags), sym)
}

func styledStatus(status string) string {
	col := lipgloss.Color("8")
	switch status {
	case "working":
		col = lipgloss.Color("11")
	case "idle", "done":
		col = lipgloss.Color("10")
	case "blocked":
		col = lipgloss.Color("9")
	}
	return lipgloss.NewStyle().Foreground(col).Render(status)
}

func shortPath(path, home string) string {
	if home != "" && strings.HasPrefix(path, home) {
		return "~" + path[len(home):]
	}
	return path
}

func layoutSummary(p project.Project) string {
	if len(p.Workspaces) == 0 {
		if p.Launch != "" {
			return "layout launch " + p.Launch
		}
		return "layout none"
	}
	name := p.StartWS
	if name == "" {
		names := make([]string, 0, len(p.Workspaces))
		for n := range p.Workspaces {
			names = append(names, n)
		}
		sort.Strings(names)
		name = names[0]
	}
	ws, ok := p.Workspaces[name]
	if !ok {
		return "layout " + name
	}
	panes := 0
	for _, tab := range ws.Tabs {
		panes += len(tab.Panes)
	}
	return fmt.Sprintf("layout %s · %d tabs · %d panes", name, len(ws.Tabs), panes)
}

func truncateLines(content string, w int) string {
	if w < 0 {
		w = 0
	}
	lines := strings.Split(content, "\n")
	out := []string{}
	for _, l := range lines {
		out = append(out, ansi.Truncate(l, w, "…"))
	}
	return strings.Join(out, "\n")
}

func padTrim(content string, n int) string {
	if n < 0 {
		n = 0
	}
	lines := strings.Split(content, "\n")
	if len(lines) > n {
		lines = lines[:n]
	}
	for len(lines) < n {
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}
