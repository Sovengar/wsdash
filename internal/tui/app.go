package tui

import (
	"fmt"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"

	"wsdash/internal/config"
	"wsdash/internal/layout"
	"wsdash/internal/project"
)

// LaunchFunc starts the workspace of a project. Injected so the model is
// testable without a multiplexer.
type LaunchFunc func(p project.Project, backendName string) error

// StatusFunc reads the git state of a project.
type StatusFunc func(p project.Project) project.Status

type statusMsg struct {
	name string
	st   project.Status
}

type launchMsg struct {
	name string
	err  error
}

// Model is the dashboard: a list of projects plus the detail of the selected one.
type Model struct {
	cfg       *config.Config
	all       []project.Project
	backend   string
	cursor    int
	showScan  bool
	width     int
	height    int
	status    map[string]project.Status
	loading   map[string]bool
	toast     string
	toastErr  bool
	launch    LaunchFunc
	readState StatusFunc
}

func New(cfg *config.Config, projs []project.Project, backendName string, launch LaunchFunc, readState StatusFunc) Model {
	if backendName == "" {
		backendName = cfg.DefaultBackend
	}
	sort.SliceStable(projs, func(i, j int) bool {
		if projs[i].Source != projs[j].Source {
			return projs[i].Source < projs[j].Source
		}
		return projs[i].Name < projs[j].Name
	})
	return Model{
		cfg:       cfg,
		all:       projs,
		backend:   backendName,
		width:     100,
		height:    30,
		status:    map[string]project.Status{},
		loading:   map[string]bool{},
		launch:    launch,
		readState: readState,
	}
}

// Visible returns the projects the list shows for the current filter.
func (m Model) Visible() []project.Project {
	out := make([]project.Project, 0, len(m.all))
	for _, p := range m.all {
		if p.Source == project.SourceScan && !m.showScan {
			continue
		}
		out = append(out, p)
	}
	return out
}

func (m Model) Selected() (project.Project, bool) {
	vis := m.Visible()
	if m.cursor < 0 || m.cursor >= len(vis) {
		return project.Project{}, false
	}
	return vis[m.cursor], true
}

func (m Model) Init() tea.Cmd {
	return m.refreshAll()
}

func (m Model) refreshAll() tea.Cmd {
	cmds := make([]tea.Cmd, 0, len(m.all))
	for _, p := range m.all {
		cmds = append(cmds, m.readCmd(p))
	}
	return tea.Batch(cmds...)
}

func (m Model) readCmd(p project.Project) tea.Cmd {
	read := m.readState
	return func() tea.Msg {
		return statusMsg{name: p.Name, st: read(p)}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil
	case statusMsg:
		m.status[msg.name] = msg.st
		delete(m.loading, msg.name)
		return m, nil
	case launchMsg:
		m.toastErr = msg.err != nil
		if msg.err != nil {
			m.toast = "launch failed: " + msg.err.Error()
		} else {
			m.toast = "launched " + msg.name + " on " + m.backend
		}
		return m, nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "j", "down":
		if m.cursor < len(m.Visible())-1 {
			m.cursor++
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}
	case "g", "home":
		m.cursor = 0
	case "G", "end":
		if n := len(m.Visible()); n > 0 {
			m.cursor = n - 1
		}
	case "tab":
		m.showScan = !m.showScan
		m.clampCursor()
	case "r":
		for _, p := range m.all {
			m.loading[p.Name] = true
		}
		m.toast, m.toastErr = "refreshing", false
		return m, m.refreshAll()
	case "L":
		m.backend = m.cfg.NextBackend(m.backend)
		m.toast, m.toastErr = "backend: "+m.backend, false
	case "enter":
		p, ok := m.Selected()
		if !ok {
			return m, nil
		}
		launch, name := m.launch, p.Name
		return m, func() tea.Msg { return launchMsg{name: name, err: launch(p, m.backend)} }
	}
	return m, nil
}

func (m *Model) clampCursor() {
	if n := len(m.Visible()); m.cursor >= n {
		m.cursor = max(0, n-1)
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// layoutSummary is a one-line portable description of the selected workspace.
func layoutSummary(p project.Project) string {
	ws, ok := p.Workspaces[p.StartWS]
	if !ok {
		for _, w := range p.Workspaces {
			ws = w
			break
		}
	}
	if len(ws.Tabs) == 0 {
		if p.Launch != "" {
			return "shell: " + p.Launch
		}
		return "no layout"
	}
	parts := make([]string, 0, len(ws.Tabs))
	for _, t := range ws.Tabs {
		label := t.Label
		if label == "" {
			label = "tab"
		}
		parts = append(parts, fmt.Sprintf("%s(%d)", label, len(t.Panes)))
	}
	return strings.Join(parts, " + ")
}

func absPath(path, home string) string {
	if home != "" && strings.HasPrefix(path, home) {
		return "~" + strings.TrimPrefix(path, home)
	}
	return path
}

var _ = layout.Workspace{}
