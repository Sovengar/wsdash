package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"wsdash/internal/config"
	"wsdash/internal/project"
)

func TestFuzzyScoreMatchesSubsequence(t *testing.T) {
	cases := []struct {
		pattern string
		target  string
		want    bool
	}{
		{"wsh", "wsdash", true},
		{"WSH", "wsdash", true},
		{"xyz", "wsdash", false},
		{"wshd", "wsdash", false},
		{"", "anything", true},
		{"a", "A", true},
	}
	for _, tc := range cases {
		if _, got := fuzzyScore(tc.pattern, tc.target); got != tc.want {
			t.Errorf("fuzzyScore(%q, %q) = %v, want %v", tc.pattern, tc.target, got, tc.want)
		}
	}
}

func searchModel(t *testing.T) Model {
	t.Helper()
	m := New(&config.Config{DefaultBackend: "fake"}, []project.Project{
		{Name: "alpha", Path: "/a", Description: "first"},
		{Name: "beta", Path: "/b", Description: "second"},
		{Name: "gamma", Path: "/c", Description: "third"},
		{Name: "delta", Path: "/d", Source: project.SourceScan},
	}, "fake", nil, nil, nil, nil)
	return m
}

func TestSlashSearchFiltersFuzzyAndEscClears(t *testing.T) {
	m := searchModel(t)
	if got := strings.Join(rowNames(m), ","); got != "alpha,beta,gamma" {
		t.Fatalf("initial rows = %s", got)
	}

	next, _ := pressKey(m, "/")
	m = next.(Model)
	if !m.searching {
		t.Fatal("/ should open the search prompt")
	}
	for _, key := range []string{"b", "t"} {
		next, _ = pressKey(m, key)
		m = next.(Model)
	}
	if m.query != "bt" {
		t.Fatalf("query = %q, want bt", m.query)
	}
	if got := strings.Join(rowNames(m), ","); got != "beta" {
		t.Fatalf("rows for bt = %s, want beta", got)
	}

	next, _ = pressKey(m, "enter")
	m = next.(Model)
	if m.searching {
		t.Fatal("enter should close the prompt")
	}
	if m.query != "bt" {
		t.Fatalf("enter should keep the filter, query = %q", m.query)
	}

	next, _ = pressKey(m, "esc")
	m = next.(Model)
	if m.query != "" {
		t.Fatalf("esc should clear the query, got %q", m.query)
	}
	if got := strings.Join(rowNames(m), ","); got != "alpha,beta,gamma" {
		t.Fatalf("rows after esc = %s", got)
	}
}

func TestSlashSearchBackspaceAndEmptyResult(t *testing.T) {
	m := searchModel(t)
	next, _ := pressKey(m, "/")
	m = next.(Model)
	for _, key := range []string{"b", "t", "a"} {
		next, _ = pressKey(m, key)
		m = next.(Model)
	}
	next, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	m = next.(Model)
	if m.query != "bt" {
		t.Fatalf("backspace left query %q, want bt", m.query)
	}

	next, _ = pressKey(m, "z")
	m = next.(Model)
	if len(m.rows) != 0 {
		t.Fatalf("rows for z = %v, want none", rowNames(m))
	}
}

func TestSlashSearchSeesScannedProjects(t *testing.T) {
	m := searchModel(t)
	next, _ := pressKey(m, "/")
	m = next.(Model)
	for _, key := range []string{"d", "l", "t"} {
		next, _ = pressKey(m, key)
		m = next.(Model)
	}
	if got := strings.Join(rowNames(m), ","); got != "delta" {
		t.Fatalf("rows for dlt = %s, want delta", got)
	}
}

func TestEnterInSearchDoesNotLaunch(t *testing.T) {
	m := searchModel(t)
	launched := false
	m.LaunchFunc = func(project.Project) error { launched = true; return nil }
	next, _ := pressKey(m, "/")
	next, cmd := pressKey(next.(Model), "enter")
	if cmd != nil {
		t.Fatal("enter in search should not quit")
	}
	if launched {
		t.Fatal("enter in search should not launch")
	}
}

func TestViewShowsSearchPrompt(t *testing.T) {
	m := searchModel(t)
	m.width, m.height = 100, 30
	next, _ := pressKey(m, "/")
	m = next.(Model)
	for _, key := range []string{"b", "e"} {
		next, _ = pressKey(m, key)
		m = next.(Model)
	}
	out := renderView(m)
	if !strings.Contains(out, "/be") {
		t.Fatalf("search prompt missing from view:\n%s", out)
	}
	assertFrame(t, out, 100, 30)
}
