- Jump lands on agent's tab, not exact pane: Herdr has no pane focus by id.
EOF- Launching/agent-jump quit TUI because Herdr active view changes; staying open in hidden pane is useless.
- **Launching (or jumping) closes the TUI.** Both actions change Herdr's active
  view, so keeping the dashboard alive in a now-hidden pane is pointless: after a
  successful launch or jump the model returns `tea.Quit`. On failure it keeps the
  toast and stays open.
- **Agents come from `herdr agent list`,** parsed by `internal/agents` behind an
  injected `Runner`. An agent belongs to a project when its `cwd` or
  `foreground_cwd` equals the project path or sits under it. A missing `herdr`
  binary degrades to "no agents", never to an error.
- **The jump lands on the agent's tab, not its pane.** Herdr has no
  `pane focus <id>`: only directional focus, `tab focus <id>` and
  `workspace focus <id>`. So the jump is `workspace focus` + `tab focus`, and
  `Pane.Focus` in the portable layout stays unused.
- **Fonts/borders**: the three sections use `lipgloss.RoundedBorder()` through a
  vendored copy of tsk's `internal/tui/bordered` helper (same API, copied, not
  imported) so both dashboards look alike.
- Focus requested at workspace creation (--focus) and verified with bounded retries, because a single trailing workspace focus raced with TUI exit.
