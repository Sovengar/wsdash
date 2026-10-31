package herdr

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"wsdash/internal/backend"
	"wsdash/internal/layout"
	"wsdash/internal/project"
)

// Runner executes one herdr invocation and returns its stdout. Tests inject a
// stub so no live server is needed.
type Runner interface {
	Run(ctx context.Context, argv []string) (string, error)
}

type execRunner struct{}

func (execRunner) Run(ctx context.Context, argv []string) (string, error) {
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			msg := strings.TrimSpace(string(ee.Stderr))
			if msg != "" {
				return string(out), fmt.Errorf("%s: %w: %s", strings.Join(argv, " "), err, msg)
			}
		}
		return string(out), fmt.Errorf("%s: %w", strings.Join(argv, " "), err)
	}
	return string(out), nil
}

// Launcher materialises a portable layout on Herdr.
type Launcher struct {
	runner Runner
	sleep  func(time.Duration)
}

func New(r Runner) *Launcher {
	if r == nil {
		r = execRunner{}
	}
	return &Launcher{runner: r, sleep: time.Sleep}
}

func (l *Launcher) setSleep(fn func(time.Duration)) { l.sleep = fn }

func (l *Launcher) Name() string { return "herdr" }

func (l *Launcher) Available(context.Context) error {
	_, err := exec.LookPath("herdr")
	return err
}

// Plan resolves the workspace a launch should use for p.
func (l *Launcher) Plan(p project.Project) (layout.Workspace, error) {
	if p.StartWS != "" {
		ws, ok := p.Workspaces[p.StartWS]
		if !ok {
			return layout.Workspace{}, fmt.Errorf("project %s: start workspace %q not found", p.Name, p.StartWS)
		}
		ws.Name = p.StartWS
		return ws, nil
	}
	if ws, ok := firstWorkspace(p.Workspaces); ok {
		return ws, nil
	}
	return layout.Workspace{Name: p.Name, Root: p.Path}, nil
}

func firstWorkspace(m map[string]layout.Workspace) (layout.Workspace, bool) {
	if len(m) == 0 {
		return layout.Workspace{}, false
	}
	names := make([]string, 0, len(m))
	for n := range m {
		names = append(names, n)
	}
	sortStrings(names)
	ws := m[names[0]]
	ws.Name = names[0]
	return ws, true
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}

// Launch creates the workspace, its tabs and its panes, then returns. It does
// not wait for the user and focuses a workspace only when the plan has tabs.
func (l *Launcher) Launch(ctx context.Context, p project.Project, ws layout.Workspace) error {
	if err := ws.Validate(); err != nil {
		return fmt.Errorf("project %s: %w", p.Name, err)
	}
	root := ws.Root
	if root == "" {
		root = p.Path
	}

	argv := []string{"herdr", "workspace", "create", "--cwd", root, "--label", wsName(ws, p.Name), "--focus"}
	for _, k := range sortedKeys(ws.Env) {
		argv = append(argv, "--env", k+"="+ws.Env[k])
	}
	out, err := l.runner.Run(ctx, argv)
	if err != nil {
		return fmt.Errorf("create workspace: %w", err)
	}
	wsID := jsonID(out, "result", "workspace", "workspace_id")
	rootPane := jsonID(out, "result", "root_pane", "pane_id")
	if wsID == "" || rootPane == "" {
		return fmt.Errorf("create workspace: unexpected response %q", strings.TrimSpace(out))
	}

	// A workspace with no tabs in the plan keeps Herdr's default tab and root pane.
	for i, tab := range ws.Tabs {
		if i > 0 {
			out, err = l.runner.Run(ctx, tabArgv(wsID, root, tab))
			if err != nil {
				return fmt.Errorf("create tab %q: %w", tab.Label, err)
			}
			rootPane = jsonID(out, "result", "root_pane", "pane_id")
			if rootPane == "" {
				return fmt.Errorf("create tab %q: unexpected response %q", tab.Label, strings.TrimSpace(out))
			}
		}
		if err := l.fillTab(ctx, root, tab, rootPane); err != nil {
			return err
		}
	}

	if len(ws.Tabs) > 0 {
		return l.focusGuaranteed(ctx, wsID)
	}
	return nil
}

func (l *Launcher) focusGuaranteed(ctx context.Context, wsID string) error {
	const attempts = 5
	const pause = 100 * time.Millisecond
	for i := 0; i < attempts; i++ {
		if _, err := l.runner.Run(ctx, []string{"herdr", "workspace", "focus", wsID}); err != nil {
			return fmt.Errorf("focus workspace: %w", err)
		}
		if focused, err := l.workspaceFocused(ctx, wsID); err == nil && focused {
			return nil
		}
		if i < attempts-1 {
			l.sleep(pause)
		}
	}
	return fmt.Errorf("workspace %s did not take focus after %d attempts", wsID, attempts)
}

func (l *Launcher) workspaceFocused(ctx context.Context, wsID string) (bool, error) {
	out, err := l.runner.Run(ctx, []string{"herdr", "workspace", "list"})
	if err != nil {
		return false, err
	}
	var resp struct {
		Result struct {
			Workspaces []struct {
				WorkspaceID string `json:"workspace_id"`
				Focused     bool   `json:"focused"`
			} `json:"workspaces"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		return false, err
	}
	for _, w := range resp.Result.Workspaces {
		if w.WorkspaceID == wsID {
			return w.Focused, nil
		}
	}
	return false, nil
}

func (l *Launcher) fillTab(ctx context.Context, root string, tab layout.Tab, paneID string) error {
	cwd := cwdOr(tab.Cwd, root)
	for i, pane := range tab.Panes {
		if i > 0 {
			argv := []string{"herdr", "pane", "split", "--pane", paneID, "--direction", splitDir(pane)}
			if pane.Ratio > 0 {
				argv = append(argv, "--ratio", strconv.FormatFloat(pane.Ratio, 'g', -1, 64))
			}
			argv = append(argv, "--cwd", cwdOr(pane.Cwd, cwd), "--no-focus")
			for _, k := range sortedKeys(pane.Env) {
				argv = append(argv, "--env", k+"="+pane.Env[k])
			}
			out, err := l.runner.Run(ctx, argv)
			if err != nil {
				return fmt.Errorf("split pane: %w", err)
			}
			paneID = jsonID(out, "result", "pane", "pane_id")
			if paneID == "" {
				return fmt.Errorf("split pane: unexpected response %q", strings.TrimSpace(out))
			}
		}
		if pane.Command == "" {
			continue
		}
		if _, err := l.runner.Run(ctx, []string{"herdr", "pane", "run", paneID, pane.Command}); err != nil {
			return fmt.Errorf("run %q: %w", pane.Command, err)
		}
		if pane.WaitFor != "" {
			_, _ = l.runner.Run(ctx, []string{
				"herdr", "pane", "wait-output", paneID,
				"--match", pane.WaitFor,
				"--timeout", strconv.Itoa(defaultWaitOutput),
			})
		}
	}
	return nil
}

func wsName(ws layout.Workspace, fallback string) string {
	if ws.Name != "" {
		return ws.Name
	}
	return fallback
}

func cwdOr(c, def string) string {
	if c != "" {
		return c
	}
	return def
}

// jsonID walks nested objects looking for the last key of path, then for the
// conventional "id" fallback. Herdr answers with {"result": {...}} envelopes.
func jsonID(raw string, path ...string) string {
	var v any
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &v); err != nil {
		return ""
	}
	cur := v
	for _, key := range path {
		obj, ok := cur.(map[string]any)
		if !ok {
			return ""
		}
		cur = obj[key]
	}
	if s, ok := cur.(string); ok {
		return s
	}
	if obj, ok := cur.(map[string]any); ok {
		if s, ok := obj["id"].(string); ok {
			return s
		}
	}
	return ""
}

func init() { backend.Register(New(nil)) }
