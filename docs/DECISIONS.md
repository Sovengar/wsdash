- **Launching (or jumping) closes the TUI.** Both actions change Herdr's active
  view, so keeping the dashboard alive in a now-hidden pane is pointless: after a
  successful launch or jump the model returns `tea.Quit`. On failure it keeps the
  toast and stays open.
- **The exit code carries the outcome.** A normal quit (`q`/`ctrl+c`) exits 0, a
  successful project launch exits 10, a jump exits 0 (it is not a launch), and
  errors exit 1. The model writes the reason through a pointer the caller reads
  after `tea.Program.Run()` returns, because the final model is otherwise
  discarded. The exit code is the only signal left once the TUI closes, so the
  fish wrapper keys on 10 to tell a launch from a plain quit.
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
- **Projects and Agents are two focusable columns.** `tab`/`shift+tab` toggle
  the focus; `h`/`left` and `l`/`right` remain aliases. `j`/`k`/`g`/`G` operate
  inside the focused column. `f` toggles the scanned-project filter.
  The focused box border is blue (`12`), the others grey (`8`). Selecting a
  project resets the agent cursor to 0 and clamps it when the agent list changes.
- **Fonts/borders**: the sections use `lipgloss.RoundedBorder()` through a
  vendored copy of tsk's `internal/tui/bordered` helper (same API, copied, not
  imported) so both dashboards look alike.
- Focus requested at workspace creation (--focus) and verified with bounded retries, because a single trailing workspace focus raced with TUI exit.

- **Worktrees se detectan con git, detrás de un runner inyectable.** Primero se
  ejecuta `git -C <path> rev-parse --git-common-dir`; si el resultado resuelve a
  `<path>/.git`, es un checkout normal. Si no, se lee
  `git worktree list --porcelain` y se toma la primera línea `worktree <path>`
  como worktree principal. Cualquier error, falta de `git` o ruta que no sea
  repositorio devuelve `("", false)`, nunca un fallo. `resolve` marca
  `ParentPath` cuando el principal coincide con otro proyecto; si no está en el
  conjunto, prueba por basename y, si tampoco, deja el worktree como proyecto
  normal. Los proyectos configurados nunca se descartan.
- **La visibilidad del worktree la decide su padre.** La fila del worktree no
  entra en el filtro de escaneo: se dibuja justo debajo del padre, indentada y
  atenuada, siempre que el padre sea visible. Si el padre está oculto por `f`,
  el worktree también. El contador `+N scanned` excluye los worktrees ya
  colgados de un padre.
- **Lanzar un worktree usa el workspace del padre con otro root.** `enter` sobre
  la fila del worktree clona el proyecto padre, conserva `start`, tabs y panes,
  y sustituye `Path` y el `root` de cada workspace por la ruta del worktree. No
  hace falta una segunda entrada en la config. Los agentes se asignan al
  proyecto con la ruta coincidente más larga para que el worktree no herede los
  del padre.
