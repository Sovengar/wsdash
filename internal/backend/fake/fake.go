package fake

import (
	"context"

	"wsdash/internal/backend"
	"wsdash/internal/layout"
	"wsdash/internal/project"
)

type Launcher struct {
	Launches []struct {
		Project project.Project
		WS      layout.Workspace
	}
}

func (f *Launcher) Name() string                        { return "fake" }
func (f *Launcher) Available(ctx context.Context) error { return nil }
func (f *Launcher) Plan(p project.Project) (layout.Workspace, error) {
	if len(p.Workspaces) > 0 {
		for name, ws := range p.Workspaces {
			ws.Name = name
			return ws, nil
		}
	}
	return layout.Workspace{Name: "default", Root: p.Path}, nil
}
func (f *Launcher) Launch(ctx context.Context, p project.Project, ws layout.Workspace) error {
	f.Launches = append(f.Launches, struct {
		Project project.Project
		WS      layout.Workspace
	}{p, ws})
	return nil
}

func init() { backend.Register(&Launcher{}) }
