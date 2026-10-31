package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"wsdash/internal/agents"
	"wsdash/internal/config"
	"wsdash/internal/project"
)

var _ = lipgloss.NewStyle

type rowKind int

const (
	rowProject rowKind = iota
	rowAgent
)

type row struct {
	kind    rowKind
	project int
	agent   int
}

type Model struct {
	cfg        *config.Config
	projects   []project.Project
	backend    string
	cursor     int
	showScan   bool
	status     map[string]project.Status
	toast      string
	agents     map[string][]agents.Agent
	rows       []row
	width      int
	height     int
	LaunchFunc func(p project.Project) error
	StatusFunc func(p project.Project) project.Status
	AgentsFunc func(p project.Project) []agents.Agent
	JumpFunc   func(a agents.Agent) error
}

func New(cfg *config.Config, projs []project.Project, backend string, launchFunc func(project.Project) error, statusFunc func(project.Project) project.Status, agentsFunc func(project.Project) []agents.Agent, jumpFunc func(agents.Agent) error) Model {
	// sort projects alphabetically (case-insensitive)
	sorted := append([]project.Project(nil), projs...)
	// simple sort
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if strings.ToLower(sorted[j].Name) < strings.ToLower(sorted[i].Name) {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	m := Model{cfg: cfg, projects: sorted, backend: backend, status: map[string]project.Status{}, agents: map[string][]agents.Agent{}, LaunchFunc: launchFunc, StatusFunc: statusFunc, AgentsFunc: agentsFunc, JumpFunc: jumpFunc}
	m.rebuildRows()
	return m
}

func (m *Model) rebuildRows() {
	// remember selected project name
	sel := ""
	if m.cursor < len(m.rows) && m.cursor >= 0 {
		sel = m.projects[m.rows[m.cursor].project].Name
	}
	m.rows = nil
	for i := range m.projects {
		if m.projects[i].Source == project.SourceScan && !m.showScan {
			continue
		}
		m.rows = append(m.rows, row{kind: rowProject, project: i})
		if sel != "" && m.projects[i].Name == sel {
			for ai := range m.agentsFor(m.projects[i]) {
				m.rows = append(m.rows, row{kind: rowAgent, project: i, agent: ai})
			}
		}
	}
	// cursor preserve by project
	newCursor := 0
	if sel != "" {
		for idx, r := range m.rows {
			if r.kind == rowProject && m.projects[r.project].Name == sel {
				newCursor = idx
				break
			}
		}
	}
	m.cursor = newCursor
	if m.cursor >= len(m.rows) {
		m.cursor = len(m.rows) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	// don't also run bottom clamp twice; remove original trailing clamp by marker
	return
}

func (m Model) cursorProjectIndex() int {
	if len(m.rows) == 0 {
		return -1
	}
	if m.cursor < len(m.rows) {
		return m.rows[m.cursor].project
	}
	if len(m.projects) > 0 {
		return 0
	}
	return -1
}

func (m Model) agentsFor(p project.Project) []agents.Agent {
	return m.agents[p.Name]
}

func (m Model) Init() tea.Cmd { return m.refreshAll() }

func (m Model) refreshAll() tea.Cmd {
	return func() tea.Msg { return refreshMsg{} }
}

func (m Model) refreshCmd() tea.Cmd {
	return func() tea.Msg { return refreshMsg{} }
}

type refreshMsg struct{}
type agentsMsg struct {
	name   string
	agents []agents.Agent
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case refreshMsg:
		for i := range m.projects {
			if m.StatusFunc != nil {
				m.status[m.projects[i].Name] = m.StatusFunc(m.projects[i])
			}
			if m.AgentsFunc != nil {
				m.agents[m.projects[i].Name] = m.AgentsFunc(m.projects[i])
			}
		}
		m.rebuildRows()
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "j", "down":
			if m.cursor < len(m.rows)-1 {
				m.cursor++
			}
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "g":
			m.cursor = 0
		case "G":
			if len(m.rows) > 0 {
				m.cursor = len(m.rows) - 1
			}
		case "tab":
			m.showScan = !m.showScan
			m.rebuildRows()
		case "r":
			m.toast = "refreshed"
			return m, m.refreshCmd()
		case "L":
			m.toast = "backend: " + m.backend
		case "enter":
			if len(m.rows) == 0 {
				return m, nil
			}
			r := m.rows[m.cursor]
			if r.kind == rowProject {
				p := m.projects[r.project]
				if m.LaunchFunc != nil {
					err := m.LaunchFunc(p)
					if err != nil {
						m.toast = "launch failed: " + err.Error()
					} else {
						m.toast = "launched " + p.Name
						return m, tea.Quit
					}
				}
			} else {
				p := m.projects[r.project]
				ags := m.agents[p.Name]
				if r.agent < len(ags) && m.JumpFunc != nil {
					a := ags[r.agent]
					err := m.JumpFunc(a)
					if err != nil {
						m.toast = "jump failed: " + err.Error()
					} else {
						m.toast = "jumped to " + a.TabID
						return m, tea.Quit
					}
				}
			}
		}
	}
	return m, nil
}

func (m Model) View() tea.View {
	// delegate to view.go helper
	return tea.NewView(renderView(m))
}
