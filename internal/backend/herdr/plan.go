package herdr

import (
	"strconv"
	"strings"

	"wsdash/internal/layout"
)

// Placeholders used by Compile. They mark where a live id goes, so --print can
// show the exact command sequence without touching a server.
const (
	phWorkspace = "<ws>"
	phPane      = "<pane>"
)

// Step is one herdr invocation. Compile output is a pure function of the plan:
// it never resolves ids and never talks to a server.
type Step struct {
	Argv []string
}

func (s Step) String() string {
	return strings.Join(quoteAll(s.Argv), " ")
}

func quoteAll(argv []string) []string {
	out := make([]string, len(argv))
	for i, a := range argv {
		if a == "" || strings.ContainsAny(a, " \t'\"<>$") {
			out[i] = "'" + strings.ReplaceAll(a, "'", `'\''`) + "'"
			continue
		}
		out[i] = a
	}
	return out
}

// Compile renders the command sequence a launch would run, ids excluded.
func Compile(ws layout.Workspace) []Step {
	var steps []Step
	root := ws.Root
	name := ws.Name
	if name == "" {
		name = "workspace"
	}
	argv := []string{"herdr", "workspace", "create", "--cwd", root, "--label", name}
	for _, k := range sortedKeys(ws.Env) {
		argv = append(argv, "--env", k+"="+ws.Env[k])
	}
	steps = append(steps, Step{Argv: argv})

	for i, tab := range ws.Tabs {
		pane := phPane
		if i > 0 {
			steps = append(steps, Step{Argv: tabArgv(phWorkspace, root, tab)})
			pane = "<tab" + strconv.Itoa(i+1) + ">"
		}
		steps = append(steps, fillTabSteps(root, tab, pane)...)
	}
	if len(ws.Tabs) > 0 {
		steps = append(steps, Step{Argv: []string{"herdr", "workspace", "focus", phWorkspace}})
	}
	return steps
}

func tabArgv(wsID, root string, tab layout.Tab) []string {
	argv := []string{"herdr", "tab", "create", "--workspace", wsID, "--cwd", cwdOr(tab.Cwd, root), "--no-focus"}
	if tab.Label != "" {
		argv = append(argv, "--label", tab.Label)
	}
	return argv
}

func fillTabSteps(root string, tab layout.Tab, firstPane string) []Step {
	var steps []Step
	cwd := cwdOr(tab.Cwd, root)
	pane := firstPane
	for i, p := range tab.Panes {
		if i > 0 {
			argv := []string{"herdr", "pane", "split", "--pane", pane, "--direction", splitDir(p)}
			if p.Ratio > 0 {
				argv = append(argv, "--ratio", strconv.FormatFloat(p.Ratio, 'g', -1, 64))
			}
			argv = append(argv, "--cwd", cwdOr(p.Cwd, cwd), "--no-focus")
			for _, k := range sortedKeys(p.Env) {
				argv = append(argv, "--env", k+"="+p.Env[k])
			}
			steps = append(steps, Step{Argv: argv})
			pane = firstPane + ".p" + strconv.Itoa(i)
		}
		if p.Command == "" {
			continue
		}
		steps = append(steps, Step{Argv: []string{"herdr", "pane", "run", pane, p.Command}})
		if p.WaitFor != "" {
			steps = append(steps, Step{Argv: []string{
				"herdr", "pane", "wait-output", pane,
				"--match", p.WaitFor, "--timeout", strconv.Itoa(defaultWaitOutput),
			}})
		}
	}
	return steps
}
