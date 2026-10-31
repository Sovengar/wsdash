# wsdash

Standalone TUI dashboard. Install: make install. Usage: wsdash --backend fake. Config at ~/.config/wsdash/config.toml. Roots ~/dev; projects prdash, gitdash, tsk, vroom, dbx, cdx.
## Layout de la pantalla

Cuatro cajas con borde redondeado: **Projects** (izquierda, ~30%), **Detail**
(derecha arriba, 6 líneas fijas), **Agents** (derecha abajo) y **Keybinds**
(abajo, ancho completo). Cada proyecto muestra su rama junto al nombre.

`tab`/`shift+tab` alternan la columna activa (Projects ↔ Agents); `h`/`l` y
`←`/`→` siguen funcionando como alias. `f` muestra u oculta los proyectos
descubiertos por escaneo. `j`/`k`/`g`/`G` mueven dentro de la columna. `enter`
sobre un proyecto lo arranca y cierra la TUI; `enter` sobre un agente salta a su
tab y también cierra. Ver `docs/FEATURES.md`.

Al salir, `wsdash` devuelve **código 10** cuando la TUI se cerró porque lanzó un
proyecto, y **0** en una salida normal (`q`/`ctrl+c`, o tras saltar a un agente).
Ese 10 es lo que usa el wrapper de fish para distinguir un lanzamiento de un
cierre normal; los errores siguen saliendo con 1.
