# Features

Inventario de lo que el usuario puede disparar. Si algo no está aquí, no existe.

## Dashboard

Cuatro cajas con borde redondeado: **Projects** (izquierda), **Detail** (derecha
arriba, altura fija de 6 líneas interiores), **Agents** (derecha abajo, ocupa el
resto de la columna) y **Keybinds** (abajo, ancho completo).

Cada fila de proyecto muestra `<name>  <branch>`: la rama va atenuada y se
trunca con `…` si no cabe; mientras el estado es desconocido no se dibuja nada
tras el nombre. El badge de agentes `N●`/`N✓`/`N!` y la línea
`+N scanned (f)` se mantienen.

Los **worktrees enlazados** de un proyecto aparecen justo debajo de su proyecto
padre, indentados con `  └ ` y atenuados, con su propia rama. Se muestran siempre
que el padre sea visible, aunque el filtro (`f`) esté apagado: no forman parte
del filtro de escaneo. Si el padre está oculto por el filtro, el worktree también
se oculta. La línea `+N scanned (f)` no cuenta los worktrees ya colgados de un
padre. Si la lista no cabe en la caja, se desplaza para mantener visible la fila
seleccionada.

| Qué | Cómo |
| --- | --- |
| Ver proyectos configurados | `wsdash` |
| Buscar proyectos (fuzzy) | `/` abre el prompt; lo escrito filtra por nombre, descripción, tags y ruta. `enter` mantiene el filtro, `esc` lo limpia. Mientras hay búsqueda la lista es plana y ordenada por puntuación, e incluye también los proyectos descubiertos por escaneo |
| Ver repos encontrados por escaneo | `f` (ocultos por defecto; contador `+N scanned (f)`) |
| Cambiar de columna | `tab` / `shift+tab` alterna Projects y Agents; `h` / `left` (Projects), `l` / `right` (Agents) siguen funcionando como alias |
| Mover selección dentro de la columna | `j` `k` `↓` `↑` `g` `G` |
| Ver detalle | descripción, path, estado, último commit, backend y resumen del layout; en un worktree muestra `worktree of <padre>` y su propio path, rama, estado y último commit |
| Ver agentes del proyecto seleccionado | caja **Agents**, filas `↳ <agent> <status> <sessionName> <tab>/<pane>`, badge `N●`/`N✓`/`N!` en la fila del proyecto |
| Arrancar workspace | `enter` en Projects sobre un proyecto → crea el workspace, le da el foco y **cierra la TUI** con código de salida 10 |
| Arrancar un worktree | `enter` sobre una fila de worktree → lanza el workspace `start` del **padre** con `root` sustituido por la ruta del worktree; los tabs/panes son los del padre, sin entrada de config extra |
| Saltar a un agente | `enter` en Agents sobre una fila → `workspace focus` + `tab focus`, y **cierra la TUI** con código 0 (un salto no es un lanzamiento) |
| Refrescar estado git + agentes | `r` |
| Cambiar backend | `L` (cicla `backend_order`) |
| Salir | `q` / `ctrl+c` |

La línea de Keybinds es `enter launch · / search · tab section · f filter · r refresh · L backend · q quit`.

## Buscador

`/` abre un prompt en la caja **Keybinds**. Cada carácter filtra con fuzzy
matching (subsecuencia, sin distinguir mayúsculas): `wsd` encuentra `wsdash`.
También busca en descripción, tags y ruta, pero el nombre puntúa más alto. La
lista pasa a ser plana y ordenada por puntuación para que el mejor resultado
quede primero.

La búsqueda recorre **todos** los proyectos, incluidos los que `f` mantiene
ocultos; la línea `+N scanned` no aparece mientras hay filtro. `↓`/`↑` mueven la
selección, `enter` cierra el prompt y conserva el filtro, `esc` lo borra.

## Códigos de salida

| Código | Qué significa |
| --- | --- |
| `0` | salida normal: `q` / `ctrl+c`, o tras saltar a un agente |
| `10` | la TUI cerró porque un proyecto se lanzó correctamente |
| `1` | error de arranque, config, backend o del propio programa |

Un lanzamiento fallido no cierra la TUI: muestra el toast y sigue abierta.

Estados de agente: `working` amarillo, `idle`/`done` verde, `blocked` rojo,
`unknown` atenuado. Un agente pertenece al proyecto cuando su `cwd` o
`foreground_cwd` es la ruta del proyecto o está dentro de ella. Si encaja con
varios proyectos, gana el de **ruta más larga**, así un agente dentro de un
worktree aparece bajo la fila del worktree y no bajo el padre.

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
| `--list` | lista proyectos resueltos con su origen (`config` / `scan` / `worktree`); los worktrees añaden `← <padre>` |
| `--no-scan` | desactiva discovery |

## Claves de config

`default_backend`, `backend_order`, `scan.roots`, `scan.depth`, `scan.exclude`,
`projects.<name>.path|description|tags|backend|start|launch`,
`projects.<name>.workspaces.<ws>.root|env`,
`.tabs[].label|cwd`, `.tabs[].panes[].command|cwd|env|split|ratio|focus|wait_for`.

`config.example.toml` y `~/.config/wsdash/config.toml` incluyen DOS proyectos sobre
`~/dev/work/vsocial`, uno por layout (cada uno con su workspace, así los dos son
alcanzables desde la TUI):

- `vsocial-api` → workspace `api`: un tab por servicio backend
  (`actuacions-api`, `agenda-api`, `formularis-api`, `assignador-api`,
  `integracions-api`), un pane shell con `cwd` en cada repo.
- `vsocial-front` → workspace `front`: un tab por repo frontend
  (`valorador-spa-vsocial`, `agenda-wc-vsocial`, `calculadora-spa-vsocial`,
  `valorador-wc-vsocial`), un pane shell con `cwd` en cada repo.

Los layouts son solo shell: no lanzan `nvim` ni ningún comando.

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
