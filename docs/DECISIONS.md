# Decisions

Append-only log of decisions taken while building wsdash. Newest last.

- **Go + Bubbletea v2** (`charm.land/*`, not `github.com/charmbracelet/*`) to match
  the sibling repos (gitdash, prdash) and ship one static binary.
- **One unified config file** instead of reusing the plugin's `projects.yaml` plus
  the spreader layout. Configured projects are first class; the discovered ones
  are hidden behind `tab`.
- **Layout stays portable.** `wait_for` is carried as data and only the backends
  that can honour it implement it (Herdr via `pane wait-output`). `focus` is kept
  in the type but not used yet: Herdr has no `pane focus <id>`, only directional
  focus and `workspace focus`, so the launcher focuses the workspace at the end.
- **`Pane.WaitFor` uses a fixed 15s timeout** (`defaultWaitOutput`). It is best
  effort: a timeout never fails the launch.
- **Panes split off the previous pane** in the same tab (a chain), not off the tab
  root, so a vertical then horizontal split reads naturally.
- **`--print` compiles with placeholders** (`<ws>`, `<pane>`, `<pane>.p1`,
  `<tab2>`): the real ids only exist at launch time and come back as JSON.
- **No socket client.** The launcher shells out to the `herdr` binary through the
  `Runner` seam, so tests assert argv and never need a live server.
- **`--launch PROJECT`** exists so the flow can be smoke-tested (and scripted)
  without opening the TUI.
- **`make` prefers the mise Go** (`ls ~/.local/share/mise/installs/go/*/bin/go`)
  because on this machine `/usr/bin/go` and `GOROOT` disagree and the build fails
  with "compile: version does not match go tool version". Override with `make GO=`.
- **Copy shipped**: `internal/tui` follows the same shape as prdash (Model,
  `Update`, `View`, injected `LaunchFunc`/`StatusFunc`) so the two read similarly.
