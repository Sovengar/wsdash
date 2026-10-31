# wsdash — design (v0, prototype)

Standalone TUI dashboard for local projects: see their state and start each one's
workspace (tabs + panes) on a terminal multiplexer.

Replaces the `prefix+pp` Herdr plugin flow (`herdr-fuzzy-projects` + `herdr-spreader`)
with one binary that is **backend agnostic**.

## Ideas that drive the design

1. **The layout is declarative and portable.** A project describes workspaces →
   tabs → panes with a split tree, ratios, cwd, env, command and focus. It holds
   no Herdr-specific or tmux-specific token.
2. **Each multiplexer compiles the plan its own way.** Herdr gets the full
   potential (ratios, `focus`, `wait_for` via `pane wait-output`); tmux and tuios
   get their own compiler later. v0 ships Herdr only.
3. **Configured projects are first class; scanned projects are hidden.** The
   dashboard shows the projects from the config. Repos found by discovery sit
   behind a keybind (`tab`), so the list stays short.
4. **The TUI never blocks on the launcher.** Launch runs in a `tea.Cmd`; the
   dashboard refreshes status afterwards.

## UX (v0)

```
┌ wsdash ──────────────────────────────────────────────────────────┐
│ PROJECTS (4)            │ prdash                                  │
│ ▸ prdash                │ multi-forge PR inbox                    │
│   gitdash               │                                         │
│   tsk                   │ path     ~/dev/projects/prdash          │
│   vroom                 │ branch   main                           │
│                         │ state    clean · up to date             │
│ ── hidden (5) tab ──    │ last     a1b2c3d fix: retarget overlay  │
│                         │ backend  herdr                          │
│                         │ layout   editor(2 panes) + review       │
├─────────────────────────┴─────────────────────────────────────────┤
│ enter launch  tab section  f filter  r refresh  q quit             │
└───────────────────────────────────────────────────────────────────┘
```

Keys: `j/k` move · `enter` launch/jump · `tab` switch section (Projects ↔ Agents) ·
`f` toggle scanned filter · `r` refresh · `L` cycle backend · `q` quit.
Exit codes: `0` quit, `10` launched a workspace.

## Config (`${XDG_CONFIG_HOME:-~/.config}/wsdash/config.toml`)

```toml
default_backend = "herdr"
backend_order = ["herdr", "tmux", "tuios"]

[scan]
roots = ["~/dev"]
depth = 3
exclude = ["node_modules", ".git", "worktrees"]

[projects.prdash]                 # table key = project name
path = "~/dev/projects/prdash"
description = "multi-forge PR inbox"
tags = ["go", "tui"]

[projects.prdash.workspaces.review]
root = "~/dev/projects/prdash"

[[projects.prdash.workspaces.review.tabs]]
label = "editor"
[[projects.prdash.workspaces.review.tabs.panes]]
command = "nvim"
[[projects.prdash.workspaces.review.tabs.panes]]
split = "down"
ratio = 0.3
command = "make test"
wait_for = "PASS"
```

The layout block is the spreader schema (already familiar), reused as our own
declarative format. Optional keys per project: `backend` (override),
`launch` (shell command instead of a layout), `start` (chosen workspace name).

## Domain contracts

```go
// internal/project
type Project struct {
    Name        string
    Path        string
    Description string
    Tags        []string
    Backend     string   // "" = config default
    StartWS     string   // "" = first workspace
    Workspaces  map[string]layout.Workspace
    Source      Source   // SourceConfig | SourceScan
}

type Status struct {           // computed lazily, cached with TTL
    Branch      string
    Dirty       bool
    AheadBehind string        // "↑2 ↓0"
    LastCommit  string        // short hash + subject
    Err         error         // missing path, not a repo
}

// internal/layout — the portable plan
type Workspace struct {
    Name string
    Root string
    Env  map[string]string
    Tabs []Tab
}
type Tab struct { Label, Cwd string; Panes []Pane }
type Pane struct {
    Command string
    Cwd     string
    Env     map[string]string
    Split   string  // "" (first pane) | "right" | "down"
    Ratio   float64 // 0 = backend default
    Focus   bool
    WaitFor string
}

// internal/backend — one implementation per multiplexer
type Launcher interface {
    Name() string
    Available(ctx context.Context) error      // binary in PATH / socket alive
    Plan(p project.Project) (layout.Workspace, error)   // resolve default ws
    Launch(ctx context.Context, p project.Project, ws layout.Workspace) error
}
```

`Launch` returns as soon as the workspace exists; it does not wait for the user.

## Herdr compiler (v0, the only real one)

Compilation is a pure function `[]Step` (testable without a socket):

| Plan element | Step |
| --- | --- |
| workspace | `herdr workspace create --cwd <root> --label <name> [--env K=V]` |
| tab | `herdr tab create --workspace <id> --cwd <cwd> --label <label>` |
| first pane | `herdr pane run <pane-id> <command>` |
| split pane | `herdr pane split --pane <id> --direction <right\|down> --ratio <r> --cwd <cwd>` then `herdr pane run <new-id> <command>` |
| `wait_for` | `herdr pane wait-output <id> --match <text>` (best effort, timeout) |
| `focus` | `herdr pane focus` / `herdr tab create --focus` |

Ids come back in the JSON printed by each command; the compiler parses them.
Every `exec` goes through a small `runner` interface so tests assert on argv, not
on a live server.

tmux and tuios are registered in the registry but return
`ErrNotImplemented` from `Launch`; `Available()` reports the binary. This proves
the seam without the cost of three compilers.

## File map

```
cmd/wsdash/main.go          flags (--print, --config, --backend), wiring, tea.NewProgram
internal/config/config.go    TOML load + defaults + validation + XDG path
internal/config/expand.go   ~ and env-variable expansion
internal/discovery/scan.go   walk roots, depth, exclude, find .git
internal/project/project.go  Project, Status, merge(config, scanned)
internal/gitstatus/status.go git branch/dirty/ahead-behind/last commit
internal/layout/layout.go    portable plan types + validation
internal/backend/backend.go  Launcher interface + registry + ErrNotImplemented
internal/backend/herdr/…     compiler (pure steps) + client (runner) + launcher
internal/backend/fake/…      in-memory launcher recording launches (tests + --backend fake)
internal/tui/…               app.go, list.go, detail.go, styles.go, keys.go, update.go
internal/testutil/…          temp HOME, fake git repos
docs/DESIGN.md               this file
docs/FEATURES.md             inventory of keys/flags/config keys
```

## Test strategy

- `config`: parse a fixture TOML, defaults, invalid backend, expansion, missing file.
- `discovery`: fixture tree in `t.TempDir()` with fake `.git` dirs, depth and exclude.
- `gitstatus`: real `git init` repos in `t.TempDir()` (clean, dirty, ahead).
- `layout`: validation rejects bad split/ratio, keeps portability (no backend token).
- `herdr`: golden test of `Steps()` for a two-tab plan; runner stub asserts argv.
- `backend`: registry, unknown name, `ErrNotImplemented` for tmux/tuios.
- `tui`: `tea.Model` update tests (keys move selection, `tab` toggles hidden,
  `enter` calls the fake launcher once, `r` refreshes).
- The suite must pass with no `herdr`, `tmux` or `git` server available
  (`git` itself is needed for `gitstatus` tests; skip cleanly if absent).

## Non-goals for v0

- Attaching to a workspace once launched (user keeps Herdr's own UI).
- Editing layouts inside the TUI (only `e` to open the config in `$EDITOR`).
- Remote machines, worktree management, per-project health checks.
- tmux/tuios compilers, notifications, persistence of the last selection.
