# wsdash

Standalone TUI dashboard. Install: make install. Usage: wsdash --backend fake. Config at ~/.config/wsdash/config.toml. Roots ~/dev; projects prdash, gitdash, tsk, vroom, dbx, cdx.
## Layout de la pantalla

Cuatro cajas con borde redondeado: **Projects** (izquierda, ~30%), **Detail**
(derecha arriba, 6 líneas fijas), **Agents** (derecha abajo) y **Keybinds**
(abajo, ancho completo). Cada proyecto muestra su rama junto al nombre.

`h`/`l` cambian de columna; `j`/`k`/`g`/`G` mueven dentro de la columna. `enter`
sobre un proyecto lo arranca y cierra la TUI; `enter` sobre un agente salta a su
tab y también cierra. Ver `docs/FEATURES.md`.
