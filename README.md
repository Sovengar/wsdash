# wsdash

Standalone TUI dashboard. Install: make install. Usage: wsdash --backend fake. Config at ~/.config/wsdash/config.toml. Roots ~/dev; projects prdash, gitdash, tsk, vroom, dbx, cdx.
EOF
## Layout de la pantalla

Tres cajas con borde redondeado: **Projects** (izquierda, ~30%), detalle del
proyecto (derecha, titulado con el nombre) y **Keybinds** (abajo, ancho completo).
`enter` sobre un proyecto lo arranca y cierra la TUI; `enter` sobre un agente
salta a su tab y también cierra. Ver `docs/FEATURES.md`.
