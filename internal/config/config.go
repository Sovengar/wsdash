package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"wsdash/internal/layout"
	"wsdash/internal/project"
)

type Scan struct {
	Roots   []string `toml:"roots"`
	Depth   int      `toml:"depth"`
	Exclude []string `toml:"exclude"`
}

type RawProject struct {
	Path        string           `toml:"path"`
	Description string           `toml:"description"`
	Tags        []string         `toml:"tags"`
	Backend     string           `toml:"backend"`
	Start       string           `toml:"start"`
	Launch      string           `toml:"launch"`
	Workspaces  map[string]RawWS `toml:"workspaces"`
}

type RawWS struct {
	Root string            `toml:"root"`
	Env  map[string]string `toml:"env"`
	Tabs []RawTab          `toml:"tabs"`
}

type RawTab struct {
	Label string    `toml:"label"`
	Cwd   string    `toml:"cwd"`
	Panes []RawPane `toml:"panes"`
}

type RawPane struct {
	Command string            `toml:"command"`
	Cwd     string            `toml:"cwd"`
	Env     map[string]string `toml:"env"`
	Split   string            `toml:"split"`
	Ratio   float64           `toml:"ratio"`
	Focus   bool              `toml:"focus"`
	WaitFor string            `toml:"wait_for"`
}

type Config struct {
	DefaultBackend string                `toml:"default_backend"`
	BackendOrder   []string              `toml:"backend_order"`
	Scan           Scan                  `toml:"scan"`
	Projects       map[string]RawProject `toml:"projects"`
}

func DefaultPath() string {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		if home, err := os.UserHomeDir(); err == nil {
			base = filepath.Join(home, ".config")
		}
	}
	return filepath.Join(base, "wsdash", "config.toml")
}

func Load(path string) (*Config, error) {
	if path == "" {
		path = DefaultPath()
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("config missing at %s\nexample:\n%s", path, exampleConfig())
		}
		return nil, err
	}
	var c Config
	if _, err := toml.Decode(string(data), &c); err != nil {
		return nil, err
	}
	if c.DefaultBackend == "" {
		c.DefaultBackend = "herdr"
	}
	if c.Scan.Depth == 0 {
		c.Scan.Depth = 3
	}
	for name, p := range c.Projects {
		if p.Path == "" {
			return nil, fmt.Errorf("project %s: missing path", name)
		}
		for wname, ws := range p.Workspaces {
			for i, tab := range ws.Tabs {
				for j, pane := range tab.Panes {
					if err := (layout.Pane{Split: pane.Split, Ratio: pane.Ratio}).Validate(); err != nil {
						return nil, fmt.Errorf("project %s workspace %s tab %d pane %d: %w", name, wname, i, j, err)
					}
				}
			}
		}
	}
	return &c, nil
}

// NextBackend cycles the backend list, honouring backend_order and then the
// remaining registered names. Unknown current values restart the list.
func (c *Config) NextBackend(current string) string {
	order := c.BackendOrder
	if len(order) == 0 {
		order = []string{"herdr", "tmux", "tuios", "fake"}
	}
	if c.DefaultBackend != "" && !contains(order, c.DefaultBackend) {
		order = append([]string{c.DefaultBackend}, order...)
	}
	for i, name := range order {
		if name == current {
			return order[(i+1)%len(order)]
		}
	}
	return order[0]
}

func contains(list []string, v string) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}

func (c *Config) ToProjects() []project.Project {
	out := []project.Project{}
	for name, rp := range c.Projects {
		ws := map[string]layout.Workspace{}
		for wname, rw := range rp.Workspaces {
			tabs := []layout.Tab{}
			for _, rt := range rw.Tabs {
				panes := []layout.Pane{}
				for _, rp2 := range rt.Panes {
					panes = append(panes, layout.Pane{
						Command: rp2.Command, Cwd: Expand(rp2.Cwd),
						Env: rp2.Env, Split: rp2.Split, Ratio: rp2.Ratio,
						Focus: rp2.Focus, WaitFor: rp2.WaitFor,
					})
				}
				tabs = append(tabs, layout.Tab{Label: rt.Label, Cwd: Expand(rt.Cwd), Panes: panes})
			}
			root := Expand(rw.Root)
			if root == "" {
				root = Expand(rp.Path)
			}
			ws[wname] = layout.Workspace{Name: wname, Root: root, Env: rw.Env, Tabs: tabs}
		}
		out = append(out, project.Project{
			Name: name, Path: Expand(rp.Path), Description: rp.Description,
			Tags: rp.Tags, Backend: rp.Backend, StartWS: rp.Start, Launch: rp.Launch,
			Workspaces: ws, Source: project.SourceConfig,
		})
	}
	return out
}

func exampleConfig() string {
	return `[projects.example]
path = "~/dev/example"
description = "example project"
`
}
