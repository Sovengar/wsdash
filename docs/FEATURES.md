# Features

Inventario de lo que el usuario puede disparar. Si algo no está aquí, no existe.

## Dashboard

Tres cajas con borde redondeado: **Projects** (izquierda), detalle del proyecto
(derecha) y **Keybinds** (abajo, ancho completo).

| Qué | Cómo |
| --- | --- |
| Ver proyectos configurados | `wsdash` |
| Ver repos encontrados por escaneo | `tab` (ocultos por defecto; contador `+N scanned`) |
| Mover selección (proyectos y agentes) | `j` `k` `↓` `↑` `g` `G` |
| Ver detalle | selección: path, branch, estado, último commit, backend, agentes |
| Ver agentes de un proyecto | filas `↳ <agent> <status> <tab>/<pane>` bajo el proyecto, badge `N●`/`N✓`/`N!` en la fila |
| Arrancar workspace | `enter` sobre un proyecto → crea el workspace, le da el foco y **cierra la TUI** |
| Saltar a un agente | `enter` sobre una fila de agente → `workspace focus` + `tab focus`, y **cierra la TUI** |
| Refrescar estado git + agentes | `r` |
| Cambiar backend | `L` (cicla `backend_order`) |
| Salir | `q` / `ctrl+c` |

Estados de agente: `working` amarillo, `idle`/`done` verde, `blocked` rojo,
`unknown` atenuado. Un agente pertenece al proyecto cuando su `cwd` o
`foreground_cwd` es la ruta del proyecto o está dentro de ella.

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

| Backend | Compila | Ejecuta | Agentes |
| --- | --- | --- | --- |
| `herdr` | sí (`workspace create`, `tab create`, `pane split`, `pane run`, `pane wait-output`, `workspace focus`) | sí | sí (`herdr agent list`) |
| `tmux` | no | no (`ErrNotImplemented`) | no |
| `tuios` | no | no (`ErrNotImplemented`) | no |
| `fake` | n/a | registra en memoria (tests) | no (lista vacía) |

## Limitaciones conocidas

- Saltar a un agente lleva a su **tab**, no al pane exacto: Herdr solo tiene foco
  direccional de panes, no `pane focus <id>`.
- Los backends sin implementar aparecen en el ciclo de `L` pero fallan al lanzar.
