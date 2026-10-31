package tui

import (
	"fmt"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"wsdash/internal/project"
)

var (
	styleTitle  = lipgloss.NewStyle().Bold(true)
	styleDim    = lipgloss.NewStyle().Faint(true)
	styleSel    = lipgloss.NewStyle().Bold(true).Reverse(true)
	styleErr    = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	styleOK     = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	styleBorder = lipgloss.NewStyle().Faint(true)
)

const keysLine = "enter launch  tab hidden  r refresh  L backend  q quit"

func (m Model) View() tea.View {
	var b strings.Builder
	b.WriteString(styleTitle.Render("wsdash") + "  " + styleDim.Render("backend: "+m.backend) + "\n")
	b.WriteString(m.body())
	b.WriteString("\n")
	b.WriteString(styleDim.Render(keysLine))
	if m.toast != "" {
		style := styleOK
		if m.toastErr {
			style = styleErr
		}
		b.WriteString("\n" + style.Render(m.toast))
	}
	return tea.NewView(b.String())
}

func (m Model) body() string {
	leftW := min(32, max(18, m.width/3))
	left := m.list(leftW)
	right := m.detail(max(20, m.width-leftW-3))
	return lipgloss.JoinHorizontal(lipgloss.Top, left, styleBorder.Render(" │ "), right)
}

func (m Model) list(w int) string {
	var b strings.Builder
	vis := m.Visible()
	hidden := len(m.all) - len(vis)
	b.WriteString(styleTitle.Render(fmt.Sprintf("PROJECTS (%d)", len(vis))) + "\n")
	for i, p := range vis {
		marker := "  "
		if p.Source == project.SourceScan {
			marker = "· "
		}
		line := marker + p.Name
		if st, ok := m.status[p.Name]; ok && st.Dirty {
			line += " *"
		}
		if m.loading[p.Name] {
			line += " …"
		}
		if i == m.cursor {
			b.WriteString(styleSel.Render(pad(line, w)))
		} else {
			b.WriteString(pad(line, w))
		}
		b.WriteString("\n")
	}
	if hidden > 0 {
		b.WriteString(styleDim.Render(fmt.Sprintf("+%d scanned (tab)", hidden)))
	}
	return lipgloss.NewStyle().Width(w).Render(b.String())
}

func (m Model) detail(w int) string {
	p, ok := m.Selected()
	if !ok {
		return styleDim.Render("no project")
	}
	home, _ := os.UserHomeDir()
	var b strings.Builder
	b.WriteString(styleTitle.Render(p.Name) + "\n")
	if p.Description != "" {
		b.WriteString(p.Description + "\n")
	}
	b.WriteString("\n")
	rows := [][2]string{{"path", absPath(p.Path, home)}}
	if st, ok := m.status[p.Name]; ok {
		if st.Err != nil {
			rows = append(rows, [2]string{"state", styleErr.Render(st.Err.Error())})
		} else {
			state := "clean"
			if st.Dirty {
				state = "dirty"
			}
			if st.AheadBehind != "" {
				state += " " + st.AheadBehind
			}
			rows = append(rows, [2]string{"branch", st.Branch}, [2]string{"state", state}, [2]string{"last", st.LastCommit})
		}
	} else {
		rows = append(rows, [2]string{"branch", "…"})
	}
	backend := p.Backend
	if backend == "" {
		backend = m.backend
	}
	rows = append(rows,
		[2]string{"backend", backend},
		[2]string{"layout", layoutSummary(p)},
	)
	if len(p.Tags) > 0 {
		rows = append(rows, [2]string{"tags", strings.Join(p.Tags, ", ")})
	}
	for _, r := range rows {
		b.WriteString(styleDim.Render(padRight(r[0], 9)) + r[1] + "\n")
	}
	return lipgloss.NewStyle().Width(w).Render(strings.TrimRight(b.String(), "\n"))
}

func pad(s string, w int) string {
	n := w - lipgloss.Width(s)
	if n <= 0 {
		return s
	}
	return s + strings.Repeat(" ", n)
}

func padRight(s string, w int) string {
	if len(s) >= w {
		return s
	}
	return s + strings.Repeat(" ", w-len(s))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
