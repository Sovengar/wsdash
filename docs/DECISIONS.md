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
- **Session names come from the pi JSONL, tail-first.** The source is the
  `agent_session.value` path returned by `herdr agent list`. We read at most the
  last 512 KiB (seek to `size-512KiB`), drop the partial first line, and keep the
  `name` of the last `session_info`. Fallbacks in order: that name;
  `terminal_title_stripped` without the leading `π - ` and trailing
  ` - <user>`; the first 8 hex characters of the uuid in the path; `""`. Names are
  cached per session path so the render path never does IO; the read happens in
  the agents refresh `tea.Cmd`.
- **The Detail box has a fixed interior height of 6 lines** (description, path,
  state, last commit, backend, layout summary). Each line is truncated to the
  box width with `…`. The Agents box takes the remaining height of the right
  column, so the whole frame stays exactly terminal-height at 100x30 and 60x20.
- **Projects and Agents are two focusable columns.** `h`/`left` and
  `l`/`right` move the focus; `j`/`k`/`g`/`G` operate inside the focused column.
  The focused box border is blue (`12`), the others grey (`8`). Selecting a
  project resets the agent cursor to 0 and clamps it when the agent list changes.
- **Fonts/borders**: the sections use `lipgloss.RoundedBorder()` through a
  vendored copy of tsk's `internal/tui/bordered` helper (same API, copied, not
  imported) so both dashboards look alike.
- Focus requested at workspace creation (--focus) and verified with bounded retries, because a single trailing workspace focus raced with TUI exit.
