package backend

import (
	"context"
	"errors"
	"sort"
	"testing"

	"wsdash/internal/layout"
	"wsdash/internal/project"
)

type stub struct{ name string }

func (s stub) Name() string                  { return s.name }
func (stub) Available(context.Context) error { return nil }
func (stub) Plan(p project.Project) (layout.Workspace, error) {
	return layout.Workspace{Name: p.Name, Root: p.Path}, nil
}
func (stub) Launch(context.Context, project.Project, layout.Workspace) error {
	return ErrNotImplemented
}

func TestRegisterAndGet(t *testing.T) {
	Register(stub{name: "test-one"})
	defer delete(registry, "test-one")
	l, ok := Get("test-one")
	if !ok || l.Name() != "test-one" {
		t.Fatalf("got %v, %v", l, ok)
	}
	if _, ok := Get("nope"); ok {
		t.Fatal("unknown backend must not resolve")
	}
	names := Names()
	sort.Strings(names)
	found := false
	for _, n := range names {
		if n == "test-one" {
			found = true
		}
	}
	if !found {
		t.Errorf("Names() misses the registered backend: %v", names)
	}
}

func TestErrNotImplementedIsSentinel(t *testing.T) {
	err := stub{name: "x"}.Launch(context.Background(), project.Project{}, layout.Workspace{})
	if !errors.Is(err, ErrNotImplemented) {
		t.Fatalf("got %v", err)
	}
}
