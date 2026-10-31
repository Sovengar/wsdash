package tui

import (
	"fmt"
	"os"
	"strings"

	"charm.land/lipgloss/v2"

	"wsdash/internal/agents"
	"wsdash/internal/project"
	"wsdash/internal/tui/bordered"
)

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
	detailW := w - sidebarW
	if detailW < 10 {
		detailW = 10
	}
	home, _ := os.UserHomeDir()

	// sidebar content
	var sb strings.Builder
	for i, r := range m.rows {
		if r.kind == rowProject {
			p := m.projects[r.project]
			ags := m.agents[p.Name]
			marker := ""
			if len(ags) > 0 {
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
				marker = fmt.Sprintf("  %d%s", len(ags), sym)
			}
			line := p.Name + marker
			if i == m.cursor {
				line = "> " + line
			} else {
				line = "  " + line
			}
			sb.WriteString(line + "\n")
		} else {
			p := m.projects[r.project]
			ags := m.agents[p.Name]
			a := ags[r.agent]
			line := fmt.Sprintf("    ↳ %s %s", a.Agent, a.Status)
			if len(line) > sidebarW-2 {
				line = line[:sidebarW-2]
			}
			if i == m.cursor {
				line = "> " + strings.TrimPrefix(line, "  ")
			}
			sb.WriteString(line + "\n")
		}
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
	sidebarContent := strings.TrimRight(sb.String(), "\n")

	// detail content
	var detail strings.Builder
	if len(m.projects) > 0 && m.cursor < len(m.rows) {
		p := m.projects[m.rows[m.cursor].project]
		st := m.status[p.Name]
		state := "clean"
		if st.Dirty {
			state = "dirty"
		}
		if st.AheadBehind != "" {
			state += " " + st.AheadBehind
		}
		path := p.Path
		if strings.HasPrefix(path, home) {
			path = "~" + path[len(home):]
		}
		detail.WriteString(p.Description + "\n")
		detail.WriteString("path " + path + "\n")
		detail.WriteString("branch " + st.Branch + "\n")
		detail.WriteString("state " + state + "\n")
		detail.WriteString("last " + st.LastCommit + "\n")
		detail.WriteString("backend " + m.backend + "\n")
		ags := m.agents[p.Name]
		detail.WriteString(fmt.Sprintf("Agents (%d)\n", len(ags)))
		if len(ags) == 0 {
			detail.WriteString(lipgloss.NewStyle().Faint(true).Render("no agents") + "\n")
		}
		for _, a := range ags {
			col := lipgloss.Color("8")
			switch a.Status {
			case "working":
				col = lipgloss.Color("11")
			case "idle", "done":
				col = lipgloss.Color("10")
			case "blocked":
				col = lipgloss.Color("9")
			}
			styled := lipgloss.NewStyle().Foreground(col).Render(a.Status)
			detail.WriteString(fmt.Sprintf("↳ %s %s %s/%s\n", a.Agent, styled, a.TabID, a.PaneID))
		}
	}
	detailContent := strings.TrimRight(detail.String(), "\n")

	// keybinds
	keys := "enter launch · tab hidden · r refresh · L backend · q quit"
	kbContent := keys
	if m.toast != "" {
		kbContent += "\n" + m.toast
	}
	keyBoxHeight := 3
	if m.toast != "" {
		keyBoxHeight = 4
	}
	avail := h - keyBoxHeight
	if avail < 4 {
		avail = 4
	}
	topHeight := avail
	// pad/trim content to topHeight-2 lines
	sidebarContent = truncateLines(sidebarContent, sidebarW-2)
	detailContent = truncateLines(detailContent, detailW-2)
	sidebarContent = padTrim(sidebarContent, topHeight-2)
	detailContent = padTrim(detailContent, topHeight-2)

	borderFg := lipgloss.Color("8")
	projFg := lipgloss.Color("12")
	detFg := lipgloss.Color("12")

	projectsBox := bordered.RenderWithTitleEx(lipgloss.RoundedBorder(), projFg, bordered.AlignLeft, " Projects ", sidebarContent, sidebarW)
	detailBox := bordered.RenderWithTitleEx(lipgloss.RoundedBorder(), detFg, bordered.AlignLeft, projectName(m), detailContent, detailW)

	top := lipgloss.JoinHorizontal(lipgloss.Top, projectsBox, detailBox)
	keyBox := bordered.RenderWithTitleEx(lipgloss.RoundedBorder(), borderFg, bordered.AlignLeft, " Keybinds ", kbContent, w)

	out := lipgloss.JoinVertical(lipgloss.Left, top, keyBox)
	_ = agents.Agent{}
	return out
}

func truncateLines(content string, w int) string {
	lines := strings.Split(content, "\n")
	out := []string{}
	for _, l := range lines {
		if len([]rune(l)) > w && w > 0 {
			// simple truncate with ellipsis
			r := []rune(l)
			if w > 1 {
				l = string(r[:w-1]) + "…"
			} else {
				l = string(r[:w])
			}
		}
		out = append(out, l)
	}
	return strings.Join(out, "\n")
}

func padTrim(content string, n int) string {
	lines := strings.Split(content, "\n")
	if len(lines) > n {
		lines = lines[:n]
	}
	for len(lines) < n {
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

func projectName(m Model) string {
	if len(m.projects) == 0 || m.cursor >= len(m.rows) {
		return " detail "
	}
	return " " + m.projects[m.rows[m.cursor].project].Name + " "
}
