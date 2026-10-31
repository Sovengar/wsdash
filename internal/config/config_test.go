package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeConfig(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadDefaultsAndProjects(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	path := writeConfig(t, `
[scan]
roots = ["~/dev"]

[projects.alpha]
path = "~/dev/alpha"
description = "alpha project"
tags = ["go"]

[projects.alpha.workspaces.dev]
root = "~/dev/alpha"

[[projects.alpha.workspaces.dev.tabs]]
label = "editor"

[[projects.alpha.workspaces.dev.tabs.panes]]
command = "nvim"

[[projects.alpha.workspaces.dev.tabs.panes]]
split = "down"
ratio = 0.3
command = "make test"
wait_for = "PASS"
`)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.DefaultBackend != "herdr" {
		t.Errorf("default backend = %q, want herdr", cfg.DefaultBackend)
	}
	if cfg.Scan.Depth != 3 {
		t.Errorf("scan depth = %d, want 3", cfg.Scan.Depth)
	}
	projs := cfg.ToProjects()
	if len(projs) != 1 {
		t.Fatalf("got %d projects, want 1", len(projs))
	}
	p := projs[0]
	if p.Name != "alpha" || p.Path != filepath.Join(home, "dev/alpha") {
		t.Errorf("unexpected project %+v", p)
	}
	ws, ok := p.Workspaces["dev"]
	if !ok {
		t.Fatalf("workspace dev missing")
	}
	if ws.Root != filepath.Join(home, "dev/alpha") {
		t.Errorf("root = %q", ws.Root)
	}
	if len(ws.Tabs) != 1 || len(ws.Tabs[0].Panes) != 2 {
		t.Fatalf("unexpected layout %+v", ws)
	}
	if got := ws.Tabs[0].Panes[1]; got.Split != "down" || got.Ratio != 0.3 || got.WaitFor != "PASS" {
		t.Errorf("unexpected split pane %+v", got)
	}
}

func TestLoadMissingConfigExplainsExample(t *testing.T) {
	_, err := Load(filepath.Join(t.TempDir(), "nope.toml"))
	if err == nil {
		t.Fatal("want error for missing config")
	}
	if !contains0(err.Error(), "example") {
		t.Errorf("error should include an example: %v", err)
	}
}

func TestLoadRejectsBadPane(t *testing.T) {
	path := writeConfig(t, `
[projects.bad]
path = "/tmp/bad"

[projects.bad.workspaces.w]

[[projects.bad.workspaces.w.tabs]]
[[projects.bad.workspaces.w.tabs.panes]]
split = "sideways"
`)
	if _, err := Load(path); err == nil {
		t.Fatal("want error for invalid split")
	}
}

func TestLoadRejectsProjectWithoutPath(t *testing.T) {
	path := writeConfig(t, "[projects.nopath]\ndescription = \"x\"\n")
	if _, err := Load(path); err == nil {
		t.Fatal("want error for missing path")
	}
}

func TestNextBackend(t *testing.T) {
	cfg := &Config{DefaultBackend: "herdr", BackendOrder: []string{"herdr", "tmux"}}
	if got := cfg.NextBackend("herdr"); got != "tmux" {
		t.Errorf("got %q", got)
	}
	if got := cfg.NextBackend("tmux"); got != "herdr" {
		t.Errorf("wraps to %q", got)
	}
	if got := cfg.NextBackend("unknown"); got != "herdr" {
		t.Errorf("unknown restarts at %q", got)
	}
	empty := &Config{}
	if got := empty.NextBackend("fake"); got != "herdr" {
		t.Errorf("no order starts at %q", got)
	}
}

func TestExpandHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if got := Expand("~/x/y"); got != filepath.Join(home, "x/y") {
		t.Errorf("got %q", got)
	}
	if got := Expand("/abs/path"); got != "/abs/path" {
		t.Errorf("absolute path changed: %q", got)
	}
}

func contains0(s, sub string) bool {
	return len(s) >= len(sub) && (len(sub) == 0 || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
