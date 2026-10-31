package herdr

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"wsdash/internal/layout"
	"wsdash/internal/project"
)

type stubRunner struct {
	splits      []string
	calls       [][]string
	err         error
	createOut   string
	listFocused []bool
}

func (s *stubRunner) Run(_ context.Context, argv []string) (string, error) {
	s.calls = append(s.calls, append([]string(nil), argv...))
	if s.err != nil {
		return "", s.err
	}
	joined := strings.Join(argv, " ")
	switch {
	case strings.Contains(joined, "workspace create"):
		if s.createOut != "" {
			return s.createOut, nil
		}
		return `{"result":{"workspace":{"workspace_id":"w1"},"tab":{"tab_id":"w1:t1"},"root_pane":{"pane_id":"w1:p1"}}}`, nil
	case strings.Contains(joined, "tab create"):
		return `{"result":{"tab":{"tab_id":"w1:t2"},"root_pane":{"pane_id":"w1:p9"}}}`, nil
	case strings.Contains(joined, "workspace list"):
		focused := true
		if len(s.listFocused) > 0 {
			focused = s.listFocused[0]
			if len(s.listFocused) > 1 {
				s.listFocused = s.listFocused[1:]
			}
		}
		return fmt.Sprintf(`{"result":{"workspaces":[{"workspace_id":"w1","focused":%t}]}}`, focused), nil
	case strings.Contains(joined, "pane split"):
		if len(s.splits) == 0 {
			return `{"result":{}}`, nil
		}
		out := s.splits[0]
		s.splits = s.splits[1:]
		return out, nil
	default:
		return `{"result":{}}`, nil
	}
}

func twoTabPlan() layout.Workspace {
	return layout.Workspace{
		Name: "review",
		Root: "/repo",
		Env:  map[string]string{"NODE_ENV": "development"},
		Tabs: []layout.Tab{
			{Label: "editor", Panes: []layout.Pane{
				{Command: "nvim"},
				{Split: "down", Ratio: 0.3, Command: "make test", WaitFor: "PASS"},
			}},
			{Label: "review", Cwd: "/repo/src", Panes: []layout.Pane{{Command: "gh pr list"}}},
		},
	}
}

func TestCompileIsPureAndOrdered(t *testing.T) {
	steps := Compile(twoTabPlan())
	want := [][]string{
		{"herdr", "workspace", "create", "--cwd", "/repo", "--label", "review", "--focus", "--env", "NODE_ENV=development"},
		{"herdr", "pane", "run", "<pane>", "nvim"},
		{"herdr", "pane", "split", "--pane", "<pane>", "--direction", "down", "--ratio", "0.3", "--cwd", "/repo", "--no-focus"},
		{"herdr", "pane", "run", "<pane>.p1", "make test"},
		{"herdr", "pane", "wait-output", "<pane>.p1", "--match", "PASS", "--timeout", "15000"},
		{"herdr", "tab", "create", "--workspace", "<ws>", "--cwd", "/repo/src", "--no-focus", "--label", "review"},
		{"herdr", "pane", "run", "<tab2>", "gh pr list"},
		{"herdr", "workspace", "focus", "<ws>"},
	}
	if len(steps) != len(want) {
		t.Fatalf("got %d steps, want %d: %v", len(steps), len(want), steps)
	}
	for i := range want {
		if !reflect.DeepEqual(steps[i].Argv, want[i]) {
			t.Errorf("step %d\n got %v\nwant %v", i, steps[i].Argv, want[i])
		}
	}
}

func TestCompileIsStable(t *testing.T) {
	ws := twoTabPlan()
	ws.Env = map[string]string{"B": "2", "A": "1", "C": "3"}
	first := Compile(ws)
	second := Compile(ws)
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("compile is not deterministic")
	}
	if !strings.Contains(first[0].String(), "--env A=1 --env B=2 --env C=3") {
		t.Errorf("env is not sorted: %s", first[0])
	}
}

func TestLaunchResolvesIDsFromResponses(t *testing.T) {
	r := &stubRunner{
		splits:      []string{`{"result":{"pane":{"pane_id":"w1:p2"}}}`},
		listFocused: []bool{true},
	}
	l := New(r)
	p := project.Project{Name: "repo", Path: "/repo"}
	if err := l.Launch(context.Background(), p, twoTabPlan()); err != nil {
		t.Fatalf("launch: %v", err)
	}

	want := [][]string{
		{"herdr", "workspace", "create", "--cwd", "/repo", "--label", "review", "--focus", "--env", "NODE_ENV=development"},
		{"herdr", "pane", "run", "w1:p1", "nvim"},
		{"herdr", "pane", "split", "--pane", "w1:p1", "--direction", "down", "--ratio", "0.3", "--cwd", "/repo", "--no-focus"},
		{"herdr", "pane", "run", "w1:p2", "make test"},
		{"herdr", "pane", "wait-output", "w1:p2", "--match", "PASS", "--timeout", "15000"},
		{"herdr", "tab", "create", "--workspace", "w1", "--cwd", "/repo/src", "--no-focus", "--label", "review"},
		{"herdr", "pane", "run", "w1:p9", "gh pr list"},
		{"herdr", "workspace", "focus", "w1"},
		{"herdr", "workspace", "list"},
	}
	if len(r.calls) != len(want) {
		t.Fatalf("got %d calls, want %d:\n%v", len(r.calls), len(want), r.calls)
	}
	for i := range want {
		if !reflect.DeepEqual(r.calls[i], want[i]) {
			t.Errorf("call %d\n got %v\nwant %v", i, r.calls[i], want[i])
		}
		for _, arg := range r.calls[i] {
			if strings.Contains(arg, "<") {
				t.Errorf("call %d contains a placeholder: %v", i, r.calls[i])
			}
		}
	}
}

func TestLaunchRequestsFocusAtCreate(t *testing.T) {
	r := &stubRunner{
		splits:      []string{`{"result":{"pane":{"pane_id":"w1:p2"}}}`},
		listFocused: []bool{true},
	}
	if err := New(r).Launch(context.Background(), project.Project{Name: "repo", Path: "/repo"}, twoTabPlan()); err != nil {
		t.Fatalf("launch: %v", err)
	}
	want := []string{"herdr", "workspace", "create", "--cwd", "/repo", "--label", "review", "--focus", "--env", "NODE_ENV=development"}
	if len(r.calls) == 0 || !reflect.DeepEqual(r.calls[0], want) {
		t.Fatalf("create call = %v, want %v", r.calls[0], want)
	}
}

func TestLaunchFocusStopsWhenListConfirms(t *testing.T) {
	r := &stubRunner{
		splits:      []string{`{"result":{"pane":{"pane_id":"w1:p2"}}}`},
		listFocused: []bool{true},
	}
	l := New(r)
	sleeps := 0
	l.setSleep(func(time.Duration) { sleeps++ })
	if err := l.Launch(context.Background(), project.Project{Name: "repo", Path: "/repo"}, twoTabPlan()); err != nil {
		t.Fatalf("launch: %v", err)
	}
	if got := countExact(r.calls, "herdr", "workspace", "focus", "w1"); got != 1 {
		t.Errorf("focus calls = %d, want 1", got)
	}
	if got := countExact(r.calls, "herdr", "workspace", "list"); got != 1 {
		t.Errorf("list calls = %d, want 1", got)
	}
	if sleeps != 0 {
		t.Errorf("sleeps = %d, want 0", sleeps)
	}
}

func TestLaunchFocusRetriesUntilConfirmed(t *testing.T) {
	r := &stubRunner{
		splits:      []string{`{"result":{"pane":{"pane_id":"w1:p2"}}}`},
		listFocused: []bool{false, false, true},
	}
	l := New(r)
	sleeps := 0
	l.setSleep(func(time.Duration) { sleeps++ })
	if err := l.Launch(context.Background(), project.Project{Name: "repo", Path: "/repo"}, twoTabPlan()); err != nil {
		t.Fatalf("launch: %v", err)
	}
	if got := countExact(r.calls, "herdr", "workspace", "focus", "w1"); got != 3 {
		t.Errorf("focus calls = %d, want 3", got)
	}
	if got := countExact(r.calls, "herdr", "workspace", "list"); got != 3 {
		t.Errorf("list calls = %d, want 3", got)
	}
	if sleeps != 2 {
		t.Errorf("sleeps = %d, want 2", sleeps)
	}
}

func TestLaunchFailsWhenFocusNeverSticks(t *testing.T) {
	r := &stubRunner{
		splits:      []string{`{"result":{"pane":{"pane_id":"w1:p2"}}}`},
		listFocused: []bool{false},
	}
	l := New(r)
	sleeps := 0
	l.setSleep(func(time.Duration) { sleeps++ })
	err := l.Launch(context.Background(), project.Project{Name: "repo", Path: "/repo"}, twoTabPlan())
	if err == nil || !strings.Contains(err.Error(), "w1") {
		t.Fatalf("got %v, want error mentioning w1", err)
	}
	if got := countExact(r.calls, "herdr", "workspace", "focus", "w1"); got != 5 {
		t.Errorf("focus calls = %d, want 5", got)
	}
	if got := countExact(r.calls, "herdr", "workspace", "list"); got != 5 {
		t.Errorf("list calls = %d, want 5", got)
	}
	if sleeps != 4 {
		t.Errorf("sleeps = %d, want 4", sleeps)
	}
}

func countExact(calls [][]string, want ...string) int {
	n := 0
	for _, call := range calls {
		if reflect.DeepEqual(call, want) {
			n++
		}
	}
	return n
}

func TestLaunchFailsWithoutWorkspaceID(t *testing.T) {
	r := &stubRunner{createOut: `{"result":{}}`}
	err := New(r).Launch(context.Background(), project.Project{Name: "x", Path: "/x"}, twoTabPlan())
	if err == nil || !strings.Contains(err.Error(), "unexpected response") {
		t.Fatalf("got %v", err)
	}
}

func TestLaunchPropagatesRunnerError(t *testing.T) {
	boom := errors.New("boom")
	err := New(&stubRunner{err: boom}).Launch(context.Background(), project.Project{Name: "x", Path: "/x"}, layout.Workspace{Name: "w"})
	if !errors.Is(err, boom) {
		t.Fatalf("got %v", err)
	}
}

func TestLaunchRejectsInvalidPlan(t *testing.T) {
	ws := layout.Workspace{Name: "w", Tabs: []layout.Tab{{Panes: []layout.Pane{{Split: "up"}}}}}
	if err := New(&stubRunner{}).Launch(context.Background(), project.Project{Name: "x"}, ws); err == nil {
		t.Fatal("want validation error")
	}
}

func TestPlanPicksStartWorkspaceThenFirstAlphabetically(t *testing.T) {
	l := New(&stubRunner{})
	p := project.Project{Name: "x", Path: "/x", Workspaces: map[string]layout.Workspace{
		"zeta": {Root: "/z"},
		"beta": {Root: "/b"},
	}}
	ws, err := l.Plan(p)
	if err != nil {
		t.Fatal(err)
	}
	if ws.Name != "beta" || ws.Root != "/b" {
		t.Errorf("got %+v", ws)
	}
	p.StartWS = "zeta"
	ws, err = l.Plan(p)
	if err != nil || ws.Name != "zeta" {
		t.Fatalf("got %+v, %v", ws, err)
	}
	p.StartWS = "missing"
	if _, err := l.Plan(p); err == nil {
		t.Fatal("want error for missing start workspace")
	}
}

func TestPlanFallsBackToProjectPath(t *testing.T) {
	ws, err := New(&stubRunner{}).Plan(project.Project{Name: "solo", Path: "/solo"})
	if err != nil {
		t.Fatal(err)
	}
	if ws.Name != "solo" || ws.Root != "/solo" {
		t.Errorf("got %+v", ws)
	}
}

func TestJSONID(t *testing.T) {
	cases := []struct {
		raw  string
		path []string
		want string
	}{
		{`{"result":{"workspace":{"workspace_id":"w1"}}}`, []string{"result", "workspace", "workspace_id"}, "w1"},
		{`{"result":{"pane":{"id":"w1:p2"}}}`, []string{"result", "pane"}, "w1:p2"},
		{`not json`, []string{"result"}, ""},
		{`{"result":{}}`, []string{"result", "pane", "pane_id"}, ""},
	}
	for _, c := range cases {
		if got := jsonID(c.raw, c.path...); got != c.want {
			t.Errorf("jsonID(%s) = %q, want %q", c.raw, got, c.want)
		}
	}
}
