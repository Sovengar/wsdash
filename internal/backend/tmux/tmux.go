package tmux

import (
	"context"
	"os/exec"

	"wsdash/internal/backend"
	"wsdash/internal/layout"
	"wsdash/internal/project"
)

type Launcher struct{}

func (Launcher) Name() string { return "tmux" }
func (Launcher) Available(ctx context.Context) error {
	_, err := exec.LookPath("tmux")
	return err
}
func (Launcher) Plan(p project.Project) (layout.Workspace, error) {
	return layout.Workspace{Name: "default", Root: p.Path}, nil
}
func (Launcher) Launch(ctx context.Context, p project.Project, ws layout.Workspace) error {
	return backend.ErrNotImplemented
}

func init() { backend.Register(Launcher{}) }
