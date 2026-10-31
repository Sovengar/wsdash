package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
	"wsdash/internal/agents"
	"wsdash/internal/backend"
	_ "wsdash/internal/backend/fake"
	"wsdash/internal/backend/herdr"
	_ "wsdash/internal/backend/herdr"
	_ "wsdash/internal/backend/tmux"
	_ "wsdash/internal/backend/tuios"
	"wsdash/internal/config"
	"wsdash/internal/discovery"
	"wsdash/internal/gitstatus"
	"wsdash/internal/project"
	"wsdash/internal/tui"
)

func main() {
	cfgPath := flag.String("config", "", "config path (default: $XDG_CONFIG_HOME/wsdash/config.toml)")
	backendName := flag.String("backend", "", "backend override (herdr, tmux, tuios, fake)")
	printFlag := flag.String("print", "", "print the compiled plan of PROJECT and exit")
	listFlag := flag.Bool("list", false, "list resolved projects and exit")
	launchFlag := flag.String("launch", "", "launch PROJECT without opening the TUI")
	noScan := flag.Bool("no-scan", false, "skip project discovery")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	projs := resolve(cfg, *noScan)
	backendChoice := pickBackend(cfg, *backendName)

	switch {
	case *listFlag:
		for _, p := range projs {
			src := "config"
			if p.Source == project.SourceScan {
				src = "scan"
			}
			if p.IsWorktree() {
				src = "worktree"
			}
			line := fmt.Sprintf("%-10s %-20s %s", src, p.Name, p.Path)
			if p.IsWorktree() {
				line += "  ← " + parentLabel(projs, p.ParentPath)
			}
			fmt.Println(line)
		}
		return
	case *launchFlag != "":
		p, ok := find(projs, *launchFlag)
		if !ok {
			fmt.Fprintf(os.Stderr, "project %q not found (try --list)\n", *launchFlag)
			os.Exit(1)
		}
		if err := launch(p, backendChoice); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Printf("launched %s on %s\n", p.Name, backendChoice)
		return
	case *printFlag != "":
		if err := printPlan(cfg, projs, *printFlag, backendChoice); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	agentsRunner := func(ctx context.Context, argv []string) (string, error) {
		cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
		b, err := cmd.Output()
		return string(b), err
	}
	app := tui.New(cfg, projs, backendChoice,
		func(p project.Project) error { return launch(p, backendChoice) },
		func(p project.Project) project.Status { return gitstatus.Status(context.Background(), p.Path) },
		func(p project.Project) []agents.Agent {
			all, _ := agents.List(context.Background(), agentsRunner)
			return agents.ForProject(all, p.Path)
		},
		func(a agents.Agent) error { return herdr.JumpAgent(context.Background(), agentsRunner, a) },
	)
	app.SessionNameFunc = agents.SessionName
	_, err = tea.NewProgram(app).Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	reason := tui.ExitNone
	if app.ExitReason != nil {
		reason = *app.ExitReason
	}
	if code := exitCode(err, reason); code != 0 {
		os.Exit(code)
	}
}

// exitCode maps the TUI outcome to the process exit code. A launch exits 10 so
// wrappers can distinguish it from a normal quit (0); errors stay at 1.
func exitCode(err error, reason tui.ExitReason) int {
	if err != nil {
		return 1
	}
	if reason == tui.ExitLaunched {
		return 10
	}
	return 0
}

// resolve merges configured projects with the discovered ones. A configured
// project always wins: a scanned repo with the same path or name is dropped.
func resolve(cfg *config.Config, noScan bool) []project.Project {
	return resolveWithRunner(cfg, noScan, discovery.GitRunner)
}

// resolveWithRunner is resolve with an injected git runner for worktree
// detection, so tests do not need a real repository.
func resolveWithRunner(cfg *config.Config, noScan bool, run discovery.Runner) []project.Project {
	out := cfg.ToProjects()
	if !noScan && len(cfg.Scan.Roots) > 0 {
		byPath := map[string]bool{}
		byName := map[string]bool{}
		for _, p := range out {
			byPath[p.Path] = true
			byName[p.Name] = true
		}
		for _, path := range discovery.Scan(cfg.Scan.Roots, cfg.Scan.Depth, cfg.Scan.Exclude) {
			if byPath[path] {
				continue
			}
			name := discovery.Name(path)
			if byName[name] {
				continue
			}
			byName[name] = true
			out = append(out, project.Project{Name: name, Path: path, Source: project.SourceScan})
		}
	}
	attachWorktrees(out, run)
	return out
}

// attachWorktrees marks every linked worktree whose main checkout belongs to
// the project set. When the main path is not itself a project, the parent is
// matched by directory basename; if that also fails the project stays ordinary.
func attachWorktrees(projs []project.Project, run discovery.Runner) {
	byPath := map[string]int{}
	for i, p := range projs {
		byPath[filepath.Clean(p.Path)] = i
	}
	for i := range projs {
		main, ok := discovery.IsLinkedWorktreeWithRunner(projs[i].Path, run)
		if !ok {
			continue
		}
		parent := -1
		if j, ok := byPath[filepath.Clean(main)]; ok && j != i {
			parent = j
		} else {
			base := filepath.Base(main)
			for j := range projs {
				if j != i && filepath.Base(projs[j].Path) == base {
					parent = j
					break
				}
			}
		}
		if parent >= 0 {
			projs[i].ParentPath = projs[parent].Path
		}
	}
}

func parentLabel(projs []project.Project, parentPath string) string {
	for _, p := range projs {
		if filepath.Clean(p.Path) == filepath.Clean(parentPath) {
			return p.Name
		}
	}
	return filepath.Base(parentPath)
}

func find(projs []project.Project, name string) (project.Project, bool) {
	for _, p := range projs {
		if p.Name == name {
			return p, true
		}
	}
	return project.Project{}, false
}

func pickBackend(cfg *config.Config, override string) string {
	if override != "" {
		return override
	}
	if cfg.DefaultBackend != "" {
		return cfg.DefaultBackend
	}
	return "herdr"
}

func launcherFor(p project.Project, name string) (backend.Launcher, error) {
	if p.Backend != "" {
		name = p.Backend
	}
	l, ok := backend.Get(name)
	if !ok {
		names := backend.Names()
		sort.Strings(names)
		return nil, fmt.Errorf("unknown backend %q (have: %s)", name, strings.Join(names, ", "))
	}
	return l, nil
}

func launch(p project.Project, backendName string) error {
	l, err := launcherFor(p, backendName)
	if err != nil {
		return err
	}
	if err := l.Available(context.Background()); err != nil {
		return fmt.Errorf("backend %s unavailable: %w", l.Name(), err)
	}
	ws, err := l.Plan(p)
	if err != nil {
		return err
	}
	return l.Launch(context.Background(), p, ws)
}

func printPlan(cfg *config.Config, projs []project.Project, name, backendName string) error {
	for _, p := range projs {
		if p.Name != name {
			continue
		}
		l, err := launcherFor(p, backendName)
		if err != nil {
			return err
		}
		ws, err := l.Plan(p)
		if err != nil {
			return err
		}
		fmt.Printf("project:   %s\n", p.Name)
		fmt.Printf("backend:   %s\n", l.Name())
		fmt.Printf("workspace: %s\n", ws.Name)
		fmt.Printf("root:      %s\n", ws.Root)
		if err := ws.Validate(); err != nil {
			return err
		}
		if l.Name() != "herdr" {
			fmt.Printf("plan:      backend %s has no compiler yet\n", l.Name())
			return nil
		}
		fmt.Println("steps:")
		for i, st := range herdr.Compile(ws) {
			fmt.Printf("  %2d. %s\n", i+1, st)
		}
		return nil
	}
	return fmt.Errorf("project %q not found (try --list)", name)
}
