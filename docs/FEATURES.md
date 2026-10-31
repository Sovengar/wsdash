# Features

Inventario de lo que el usuario puede disparar. Si algo no está aquí, no existe.

## Dashboard

| Qué | Cómo |
| --- | --- |
| Ver proyectos configurados | `wsdash` |
| Ver repos encontrados por escaneo | `tab` (ocultos por defecto) |
| Mover selección | `j` `k` `↓` `↑` `g` `G` |
| Ver detalle (path, branch, dirty, ahead/behind, último commit, backend, layout) | selección sobre el proyecto |
| Arrancar workspace | `enter` |
| Refrescar estado git | `r` |
| Cambiar backend | `L` (cicla `backend_order`) |
| Salir | `q` / `ctrl+c` |

## Flags

| Flag | Qué hace |
| --- | --- |
| `--config PATH` | usa otra config |
| `--backend NAME` | fuerza backend (`herdr`, `tmux`, `tuios`, `fake`) |
| `--print PROJECT` | imprime workspace, root y pasos compilados; no toca el multiplexor |
| `--launch PROJECT` | arranca el proyecto y sale; no abre la TUI |
| `--list` | lista proyectos resueltos con su origen (`config` / `scan`) |
| `--no-scan` | desactiva discovery |

## Claves de config

`default_backend`, `backend_order`, `scan.roots`, `scan.depth`, `scan.exclude`,
`projects.<name>.path|description|tags|backend|start|launch`,
`projects.<name>.workspaces.<ws>.root|env`,
`.tabs[].label|cwd`, `.tabs[].panes[].command|cwd|env|split|ratio|focus|wait_for`.

## Backends

| Backend | Compila | Ejecuta |
| --- | --- | --- |
| `herdr` | sí (`workspace create`, `tab create`, `pane split`, `pane run`, `pane wait-output`, `workspace focus`) | sí |
| `tmux` | no | no (`ErrNotImplemented`) |
| `tuios` | no | no (`ErrNotImplemented`) |
| `fake` | n/a | registra en memoria, para tests y demos |
