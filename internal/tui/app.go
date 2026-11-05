package tui

import (
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"wsdash/internal/agents"
	"wsdash/internal/config"
	"wsdash/internal/layout"
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
}

type focusArea int

const (
	focusProjects focusArea = iota
	focusAgents
)

// ExitReason records why the TUI quit so the caller can map it to an exit code.
type ExitReason int

const (
	ExitNone ExitReason = iota
	ExitQuit
	ExitLaunched
)

type Model struct {
	cfg             *config.Config
	projects        []project.Project
	backend         string
	cursor          int
	agentCursor     int
	focus           focusArea
	showScan        bool
	searching       bool
	query           string
	status          map[string]project.Status
	toast           string
	agents          map[string][]agents.Agent
	sessionNames    map[string]string
	rows            []row
	width           int
	height          int
	ExitReason      *ExitReason
	LaunchFunc      func(p project.Project) error
	StatusFunc      func(p project.Project) project.Status
	AgentsFunc      func(p project.Project) []agents.Agent
	SessionNameFunc func(a agents.Agent) string
	JumpFunc        func(a agents.Agent) error
}

func (m Model) setExitReason(reason ExitReason) {
	if m.ExitReason != nil {
		*m.ExitReason = reason
	}
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
		ExitReason:   new(ExitReason),
		LaunchFunc:   launchFunc, StatusFunc: statusFunc, AgentsFunc: agentsFunc, JumpFunc: jumpFunc,
	}
	m.rebuildRows()
	return m
}

func (m *Model) rebuildRows() {
	oldName := m.selectedProjectName()
	m.normalizeAgentAssignments()
	m.rows = nil

	if m.query != "" {
		m.rows = m.filteredRows()
		m.finishRows(oldName)
		return
	}

	byPath := map[string]int{}
	for i := range m.projects {
		byPath[filepath.Clean(m.projects[i].Path)] = i
	}
	children := map[int][]int{}
	top := []int{}
	for i := range m.projects {
		p := m.projects[i]
		if p.IsWorktree() {
			if parent, ok := byPath[filepath.Clean(p.ParentPath)]; ok && parent != i {
				children[parent] = append(children[parent], i)
				continue
			}
		}
		top = append(top, i)
	}

	visible := make(map[int]bool, len(m.projects))
	visiting := make(map[int]bool, len(m.projects))
	var isVisible func(int) bool
	isVisible = func(i int) bool {
		if v, ok := visible[i]; ok {
			return v
		}
		if visiting[i] {
			return false
		}
		visiting[i] = true
		p := m.projects[i]
		v := p.Source != project.SourceScan || m.showScan
		if p.IsWorktree() {
			if parent, ok := byPath[filepath.Clean(p.ParentPath)]; ok && parent != i {
				v = isVisible(parent)
			}
		}
		visiting[i] = false
		visible[i] = v
		return v
	}
	var appendTree func(int)
	appendTree = func(i int) {
		if !isVisible(i) {
			return
		}
		m.rows = append(m.rows, row{kind: rowProject, project: i})
		for _, child := range children[i] {
			appendTree(child)
		}
	}
	for _, i := range top {
		appendTree(i)
	}

	m.finishRows(oldName)
}

// finishRows restores the selection by project name after rows are rebuilt
// and keeps the agent cursor in range.
func (m *Model) finishRows(oldName string) {
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

func (m *Model) normalizeAgentAssignments() {
	if len(m.agents) == 0 || len(m.projects) == 0 {
		return
	}
	assigned := map[string][]agents.Agent{}
	seen := map[string]bool{}
	for i := range m.projects {
		src := m.projects[i]
		for _, a := range m.agents[src.Name] {
			key := agentKey(a)
			if seen[key] {
				continue
			}
			best := src
			bestLen := -1
			for j := range m.projects {
				candidate := m.projects[j]
				if agentMatchesPath(a, candidate.Path) && len(candidate.Path) > bestLen {
					best = candidate
					bestLen = len(candidate.Path)
				}
			}
			seen[key] = true
			assigned[best.Name] = append(assigned[best.Name], a)
		}
	}
	m.agents = assigned
}

func agentKey(a agents.Agent) string {
	return strings.Join([]string{a.Agent, a.WSID, a.TabID, a.PaneID, a.Cwd, a.FgCwd, a.AgentSession.Value}, "\x00")
}

func agentMatchesPath(a agents.Agent, path string) bool {
	if path == "" {
		return false
	}
	for _, cwd := range []string{a.Cwd, a.FgCwd} {
		if cwd == path || strings.HasPrefix(cwd, path+"/") {
			return true
		}
	}
	return false
}

func (m Model) parentProject(p project.Project) (project.Project, bool) {
	if !p.IsWorktree() {
		return project.Project{}, false
	}
	for _, candidate := range m.projects {
		if filepath.Clean(candidate.Path) == filepath.Clean(p.ParentPath) {
			return candidate, true
		}
	}
	return project.Project{}, false
}

func launchProjectFor(parent, worktree project.Project) project.Project {
	out := parent
	out.Path = worktree.Path
	out.ParentPath = worktree.ParentPath
	if len(parent.Workspaces) > 0 {
		workspaces := make(map[string]layout.Workspace, len(parent.Workspaces))
		for name, ws := range parent.Workspaces {
			ws.Root = worktree.Path
			workspaces[name] = ws
		}
		out.Workspaces = workspaces
	}
	return out
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

func (m Model) Init() tea.Cmd { return m.refreshCmd() }

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
		if m.searching {
			return m.updateSearch(msg)
		}
		switch msg.String() {
		case "/":
			m.searching = true
			m.focus = focusProjects
			m.query = ""
			m.rebuildRows()
		case "q", "ctrl+c":
			m.setExitReason(ExitQuit)
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
		case "tab", "shift+tab":
			if m.focus == focusProjects {
				m.focus = focusAgents
			} else {
				m.focus = focusProjects
			}
		case "f":
			m.showScan = !m.showScan
			m.rebuildRows()
		case "esc":
			if m.query != "" {
				m.query = ""
				m.rebuildRows()
			}
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
				launchP := p
				if parent, ok := m.parentProject(p); ok {
					launchP = launchProjectFor(parent, p)
				}
				if m.LaunchFunc != nil {
					err := m.LaunchFunc(launchP)
					if err != nil {
						m.toast = "launch failed: " + err.Error()
					} else {
						m.toast = "launched " + p.Name
						m.setExitReason(ExitLaunched)
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
						m.setExitReason(ExitQuit)
						return m, tea.Quit
					}
				}
			}
		}
	}
	return m, nil
}

// updateSearch handles keys while the "/" prompt is open. Printable text
// extends the fuzzy query, enter keeps the filter, esc clears it.
func (m Model) updateSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	switch key {
	case "ctrl+c":
		m.setExitReason(ExitQuit)
		return m, tea.Quit
	case "esc":
		m.searching = false
		m.query = ""
		m.rebuildRows()
	case "enter":
		m.searching = false
	case "backspace":
		if r := []rune(m.query); len(r) > 0 {
			m.query = string(r[:len(r)-1])
		}
		m.rebuildRows()
		m.cursor = 0
		m.agentCursor = 0
	case "down":
		if m.cursor < len(m.rows)-1 {
			m.cursor++
			m.agentCursor = 0
		}
	case "up":
		if m.cursor > 0 {
			m.cursor--
			m.agentCursor = 0
		}
	default:
		if kp, ok := msg.(tea.KeyPressMsg); ok && kp.Text != "" {
			m.query += kp.Text
			m.rebuildRows()
			m.cursor = 0
			m.agentCursor = 0
		}
	}
	return m, nil
}

func (m Model) View() tea.View {
	// delegate to view.go helper
	return tea.NewView(renderView(m))
}
