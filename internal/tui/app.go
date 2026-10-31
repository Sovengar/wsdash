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

type focusArea int

const (
	focusProjects focusArea = iota
	focusAgents
)

type Model struct {
	cfg             *config.Config
	projects        []project.Project
	backend         string
	cursor          int
	agentCursor     int
	focus           focusArea
	showScan        bool
	status          map[string]project.Status
	toast           string
	agents          map[string][]agents.Agent
	sessionNames    map[string]string
	rows            []row
	width           int
	height          int
	LaunchFunc      func(p project.Project) error
	StatusFunc      func(p project.Project) project.Status
	AgentsFunc      func(p project.Project) []agents.Agent
	SessionNameFunc func(a agents.Agent) string
	JumpFunc        func(a agents.Agent) error
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
	m := Model{
		cfg: cfg, projects: sorted, backend: backend,
		status: map[string]project.Status{}, agents: map[string][]agents.Agent{},
		sessionNames: map[string]string{},
		LaunchFunc:   launchFunc, StatusFunc: statusFunc, AgentsFunc: agentsFunc, JumpFunc: jumpFunc,
	}
	m.rebuildRows()
	return m
}

func (m *Model) rebuildRows() {
	oldName := m.selectedProjectName()
	m.rows = nil
	for i := range m.projects {
		if m.projects[i].Source == project.SourceScan && !m.showScan {
			continue
		}
		m.rows = append(m.rows, row{kind: rowProject, project: i})
	}

	newCursor := 0
	if oldName != "" {
		for idx, r := range m.rows {
			if m.projects[r.project].Name == oldName {
				newCursor = idx
				break
			}
		}
	}
	if newCursor >= len(m.rows) {
		newCursor = len(m.rows) - 1
	}
	if newCursor < 0 {
		newCursor = 0
	}
	m.cursor = newCursor
	if m.selectedProjectName() != oldName {
		m.agentCursor = 0
	}
	m.clampAgentCursor()
}

func (m Model) selectedProjectName() string {
	if len(m.rows) == 0 || m.cursor < 0 || m.cursor >= len(m.rows) {
		return ""
	}
	return m.projects[m.rows[m.cursor].project].Name
}

func (m Model) selectedProject() (project.Project, bool) {
	if len(m.rows) == 0 || m.cursor < 0 || m.cursor >= len(m.rows) {
		return project.Project{}, false
	}
	return m.projects[m.rows[m.cursor].project], true
}

func (m *Model) clampAgentCursor() {
	p, ok := m.selectedProject()
	if !ok {
		m.agentCursor = 0
		return
	}
	n := len(m.agents[p.Name])
	if m.agentCursor >= n {
		m.agentCursor = n - 1
	}
	if m.agentCursor < 0 {
		m.agentCursor = 0
	}
}

func (m Model) agentsFor(p project.Project) []agents.Agent {
	return m.agents[p.Name]
}

func (m Model) Init() tea.Cmd { return m.refreshCmd() }

func (m Model) refreshAll() tea.Cmd { return m.refreshCmd() }

func (m Model) refreshCmd() tea.Cmd {
	projects := append([]project.Project(nil), m.projects...)
	statusFunc := m.StatusFunc
	agentsFunc := m.AgentsFunc
	sessionNameFunc := m.SessionNameFunc
	return func() tea.Msg {
		data := refreshDataMsg{
			status:   map[string]project.Status{},
			agents:   map[string][]agents.Agent{},
			sessions: map[string]string{},
		}
		for _, p := range projects {
			if statusFunc != nil {
				data.status[p.Name] = statusFunc(p)
			}
			if agentsFunc != nil {
				ags := agentsFunc(p)
				data.agents[p.Name] = ags
				if sessionNameFunc != nil {
					for _, a := range ags {
						path := a.AgentSession.Value
						if path == "" {
							continue
						}
						if _, ok := data.sessions[path]; !ok {
							data.sessions[path] = sessionNameFunc(a)
						}
					}
				}
			}
		}
		return data
	}
}

type refreshMsg struct{}
type refreshDataMsg struct {
	status   map[string]project.Status
	agents   map[string][]agents.Agent
	sessions map[string]string
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case refreshMsg:
		return m, m.refreshCmd()
	case refreshDataMsg:
		if msg.status != nil {
			m.status = msg.status
		}
		if msg.agents != nil {
			m.agents = msg.agents
		}
		if msg.sessions != nil {
			m.sessionNames = msg.sessions
		}
		m.rebuildRows()
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "j", "down":
			if m.focus == focusAgents {
				p, ok := m.selectedProject()
				if ok && m.agentCursor < len(m.agents[p.Name])-1 {
					m.agentCursor++
				}
			} else if m.cursor < len(m.rows)-1 {
				m.cursor++
				m.agentCursor = 0
			}
		case "k", "up":
			if m.focus == focusAgents {
				if m.agentCursor > 0 {
					m.agentCursor--
				}
			} else if m.cursor > 0 {
				m.cursor--
				m.agentCursor = 0
			}
		case "g":
			if m.focus == focusAgents {
				m.agentCursor = 0
			} else {
				m.cursor = 0
				m.agentCursor = 0
			}
		case "G":
			if m.focus == focusAgents {
				if p, ok := m.selectedProject(); ok && len(m.agents[p.Name]) > 0 {
					m.agentCursor = len(m.agents[p.Name]) - 1
				}
			} else if len(m.rows) > 0 {
				m.cursor = len(m.rows) - 1
				m.agentCursor = 0
			}
		case "h", "left":
			m.focus = focusProjects
		case "l", "right":
			m.focus = focusAgents
		case "tab":
			m.showScan = !m.showScan
			m.rebuildRows()
		case "r":
			m.toast = "refreshed"
			return m, m.refreshCmd()
		case "L":
			m.toast = "backend: " + m.backend
		case "enter":
			p, ok := m.selectedProject()
			if !ok {
				return m, nil
			}
			if m.focus == focusProjects {
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
				ags := m.agents[p.Name]
				if m.agentCursor < len(ags) && m.JumpFunc != nil {
					a := ags[m.agentCursor]
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
