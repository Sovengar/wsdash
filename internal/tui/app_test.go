package tui

import (
	"testing"

	"wsdash/internal/config"
	"wsdash/internal/project"
)

func TestNew(t *testing.T) {
	cfg := &config.Config{DefaultBackend: "fake"}
	m := New(cfg, []project.Project{{Name: "p", Path: "/p"}}, "fake", nil, nil, nil, nil)
	if m.backend != "fake" {
		t.Fatal("backend")
	}
}
