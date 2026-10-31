package backend

import (
	"context"
	"errors"
	"wsdash/internal/layout"
	"wsdash/internal/project"
)

var ErrNotImplemented = errors.New("not implemented")

type Launcher interface {
	Name() string
	Available(ctx context.Context) error
	Plan(p project.Project) (layout.Workspace, error)
	Launch(ctx context.Context, p project.Project, ws layout.Workspace) error
}

var registry = map[string]Launcher{}

func Register(l Launcher) { registry[l.Name()] = l }

func Get(name string) (Launcher, bool) { l, ok := registry[name]; return l, ok }

func Names() []string {
	out := make([]string, 0, len(registry))
	for k := range registry {
		out = append(out, k)
	}
	return out
}
