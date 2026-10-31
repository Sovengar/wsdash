# AGENTS.md — wsdash

## What it is

A standalone TUI dashboard for local projects. It shows each repo's state (branch,
dirty, ahead/behind, last commit) and launches the project's workspace (tabs +
panes) on a terminal multiplexer. It replaces the `prefix+pp` Herdr plugin flow
(`herdr-fuzzy-projects` + `herdr-spreader`) with one backend-agnostic binary.

Read `docs/DESIGN.md` for the contracts and `docs/DECISIONS.md` for the why.

## Stack

- Go (`module wsdash`), `charm.land/bubbletea/v2`, `charm.land/bubbles/v2`,
  `charm.land/lipgloss/v2` — import paths are `charm.land`, NOT `github.com/...`
- `github.com/BurntSushi/toml`
- At runtime: the `herdr` binary for real launches; `git` for status

## Commands

```bash
make check    # build + gofmt check + go vet + go test -race — the gate
make test
make install  # ~/.local/bin/wsdash  (run it after a code change)
make run
```

`make` picks the mise-managed Go because `/usr/bin/go` and `GOROOT` disagree on
this machine. Override with `make GO=/path/to/go`.

## Layout

```
cmd/wsdash/            flags, wiring, resolve(config + scan), launch
internal/config/       TOML load, validation, ~/env expansion, backend cycling
internal/discovery/    walk scan roots, find .git, derive names
internal/gitstatus/    branch/dirty/ahead-behind/last commit
internal/layout/       portable plan types + validation
internal/backend/      Launcher interface + registry + ErrNotImplemented
internal/backend/herdr/  pure Compile (placeholders) + Launch (real ids) + Runner
internal/backend/fake/   in-memory launcher for tests
internal/tui/          Model/Update/View, injected LaunchFunc + StatusFunc
```

## Rules that matter

- **The layout never holds backend tokens.** Backend specifics live in the
  compiler (`internal/backend/<name>`), never in `internal/layout` or the config.
- **Every `exec` goes through a `Runner`/injected func.** Tests assert argv; they
  never need a live Herdr server, and the suite passes without `herdr` installed.
- **`--print PROJECT` is the debug path**: it compiles the plan with placeholders
  and touches nothing.
- A configured project always wins over a discovered one (same name or path).
- New user-visible key, flag or config key → update `docs/FEATURES.md` and
  `README.md` in the same change.
- After a code change: `make check`, then `make install`.
