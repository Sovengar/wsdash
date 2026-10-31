# Prototype brief — wsdash iterate: agents + rounded-border UI

Status: iterate the existing wsdash MVP (already committed on the repo).
Location: /home/buble/dev/projects/wsdash
Branch: main (shared environment, no isolation, no remote)

Two changes. Both are features of the existing dashboard; do not restructure the
package layout and do not touch the backend compilers.

## 1. Agents on a project, and jump to their tab

Herdr already knows who is working where. `herdr agent list` returns
`{"result":{"agents":[...]}}` and each agent carries:

```
agent, agent_status (working|idle|blocked|done|unknown), cwd, foreground_cwd,
pane_id ("w45:p1B"), tab_id ("w45:t0"), workspace_id ("w45"),
terminal_title / terminal_title_stripped, focused
```

Requirements:

- A new read-only source `internal/agents` that shells out to the `herdr` binary
  through an injected `Runner` func (`type Runner func(ctx, argv) (string, error)`)
  and parses that JSON. No socket, no new dependency. Injectable so tests use a
  fixture string and never a live server.
- An agent belongs to a project when its `cwd` or `foreground_cwd` is the project
  path or is inside it (`strings.HasPrefix(path, projectPath+"/")`). Agents whose
  cwd matches no project are not shown.
- Refresh agents together with git status: on `Init` and on `r`. After a
  successful launch, refresh too. A refresh is a `tea.Cmd`; the TUI never blocks.
  When the `herdr` binary is missing, agents are simply empty and the dashboard
  still works (no error toast).
- Detail pane gets an `agents` block: one line per agent with the status word
  colour-coded (`working` yellow, `idle`/`done` green, `blocked` red,
  `unknown` dim), the agent kind, and `tab/pane` (e.g. `w45:t12 / w45:p1D`).
  Count in the section title, e.g. `Agents (2)`.
- Sidebar project rows show a compact marker when the project has agents: the
  count plus the worst status, e.g. `prdash  2●` (working) / `prdash  1✓` (idle).
  Keep it short: it must not push names out of the column.
- Selection: the list becomes a flat list of rows — every project, and directly
  under the selected project its agent rows, indented with `↳ `. `j/k/↑/↓` walk
  the whole flat list (so a user can land on an agent row). `enter` on a project
  launches it (unchanged); `enter` on an agent row jumps to it.
- Jump = `herdr workspace focus <workspace_id>` then `herdr tab focus <tab_id>`.
  Herdr has **no** `pane focus <id>` (only directional focus), so document in
  DECISIONS.md that the jump lands on the agent's tab, not on the exact pane.
  The jump is also a `tea.Cmd` and reports a toast.

## 2. UI: rounded borders for the three sections

The current UI is one flat text block. Wrap the three sections in rounded boxes,
the way `tsk` does.

Reference implementation to copy (vendored, same API, keep the tests if you copy
them): `~/dev/projects/tsk/internal/tui/bordered/` — `RenderWithTitleEx(border,
borderFg, align, title, content, width)`. Copy it into
`internal/tui/bordered/` (package `bordered`), do not import tsk.

Layout of the screen:

```
╭ Projects ─────────────────╮╭ prdash ──────────────────────────────╮
│ ...                       ││ ...                                  │
╰───────────────────────────╯╰──────────────────────────────────────╯
╭ Keybinds ──────────────────────────────────────────────────────────╮
│ ...                                                                │
╰────────────────────────────────────────────────────────────────────╯
```

- Three boxes: **Projects** (left), the project detail (right, titled with the
  project name), **Keybinds** (full width, bottom).
- `lipgloss.RoundedBorder()`, border colour `lipgloss.Color("8")` normally and
  `lipgloss.Color("12")` for the focused box. Focus follows `tab`/`shift+tab`
  between Projects and Keybinds? No: keep one focus model — Projects focused
  (blue) by default; the Keybinds box is always grey; the detail box is blue
  while the list has a selection.
- Widths: sidebar ~30% (min 22), detail the rest, keybinds full width. Heights
  must fill the terminal and must not overflow: nothing may be drawn past the
  last row or past the terminal width, including when the terminal is 60x20.
  Clip content to the box interior (use the `bordered` helper, it pads).
- Keys line goes inside the Keybinds box as the title bar content, e.g.
  `enter launch · tab hidden · r refresh · L backend · q quit`. The toast, if
  any, is the last line inside that box.
- No emoji beyond the existing `↳`, `*`, `…`; the status words stay text.

## Tests (minimum that proves it works)

- `internal/agents`: fixture JSON → parsed agents, project matching (exact path,
  nested path, unrelated), and the missing-binary case returning empty.
- herdr jump: a fake runner asserts the `workspace focus` + `tab focus` argv.
- `internal/tui`: an agent row is reachable with `j`, `enter` there triggers the
  jump instead of a launch, badges render, and the frame stays inside 60x20
  (assert no line is longer than the width and the line count fits).
- `make check` must stay green.

## Out of scope

tmux/tuios compilers, live agent output preview, attaching to a pane, mutating
agents, changing the layout/backend contracts, replacing the config format.
