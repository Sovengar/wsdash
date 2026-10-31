# Features

Inventario de lo que el usuario puede disparar. Si algo no está aquí, no existe.

## Dashboard

Cuatro cajas con borde redondeado: **Projects** (izquierda), **Detail** (derecha
arriba, altura fija de 6 líneas interiores), **Agents** (derecha abajo, ocupa el
resto de la columna) y **Keybinds** (abajo, ancho completo).

Cada fila de proyecto muestra `<name>  <branch>`: la rama va atenuada y se
trunca con `…` si no cabe; mientras el estado es desconocido no se dibuja nada
tras el nombre. El badge de agentes `N●`/`N✓`/`N!` y la línea
`+N scanned (tab)` se mantienen.

| Qué | Cómo |
| --- | --- |
| Ver proyectos configurados | `wsdash` |
| Ver repos encontrados por escaneo | `tab` (ocultos por defecto; contador `+N scanned`) |
| Cambiar de columna | `h` / `left` (Projects), `l` / `right` (Agents) |
| Mover selección dentro de la columna | `j` `k` `↓` `↑` `g` `G` |
| Ver detalle | descripción, path, estado, último commit, backend y resumen del layout |
| Ver agentes del proyecto seleccionado | caja **Agents**, filas `↳ <agent> <status> <sessionName> <tab>/<pane>`, badge `N●`/`N✓`/`N!` en la fila del proyecto |
| Arrancar workspace | `enter` en Projects sobre un proyecto → crea el workspace, le da el foco y **cierra la TUI** |
| Saltar a un agente | `enter` en Agents sobre una fila → `workspace focus` + `tab focus`, y **cierra la TUI** |
| Refrescar estado git + agentes | `r` |
| Cambiar backend | `L` (cicla `backend_order`) |
| Salir | `q` / `ctrl+c` |

La línea de Keybinds es `enter launch · h/l column · tab hidden · r refresh · L backend · q quit`.

Estados de agente: `working` amarillo, `idle`/`done` verde, `blocked` rojo,
`unknown` atenuado. Un agente pertenece al proyecto cuando su `cwd` o
`foreground_cwd` es la ruta del proyecto o está dentro de ella.

Al cambiar de proyecto el cursor de agentes vuelve a 0; si la lista de agentes
cambia, el cursor se recorta para no quedar fuera de rango.

## Nombre de sesión de los agentes

La caja Agents muestra el nombre humano de la sesión de pi del agente, leído de
`agent_session.value` (un `.jsonl`). Se lee como máximo el último 512 KiB y gana
el último `session_info`. Si no hay nombre, se usan en orden:
`terminal_title_stripped` (sin `π - ` ni ` - <usuario>`), los primeros 8 hex del
uuid del fichero, o cadena vacía. El resultado se cachea por ruta para que el
render no haga IO; la lectura ocurre en el `tea.Cmd` de refresco de agentes.

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

`config.example.toml` y `~/.config/wsdash/config.toml` incluyen el proyecto
`vsocial` (`path = "~/dev/work/vsocial"`, `start = "api"`):

- workspace `api`: un tab por servicio backend (`actuacions-api`, `agenda-api`,
  `formularis-api`, `assignador-api`, `integracions-api`), un pane con `nvim` y
  `cwd` en cada repo.
- workspace `front`: un tab por repo frontend (`valorador-spa-vsocial`,
  `agenda-wc-vsocial`, `calculadora-spa-vsocial`, `valorador-wc-vsocial`), un
  pane con `nvim` y `cwd` en cada repo.

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
