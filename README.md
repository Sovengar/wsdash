# wsdash

Dashboard TUI standalone de proyectos: ves el estado de cada repo y arrancas su
workspace (tabs + panes) en tu multiplexor.

Sustituye el flujo `prefix+pp` (plugin `herdr-fuzzy-projects` + `herdr-spreader`)
por un único binario **agnóstico de backend**: hoy Herdr, mañana tmux o tuios.

## Instalación

```bash
cd ~/dev/projects/wsdash
make install          # binario en ~/.local/bin/wsdash
```

Config en `~/.config/wsdash/config.toml` (ejemplo: `config.example.toml`).

## Uso

```bash
wsdash                     # dashboard
wsdash --list              # proyectos resueltos (config + escaneados)
wsdash --print prdash      # comandos que ejecutaría el backend
wsdash --launch prdash     # arranca sin abrir la TUI
wsdash --backend fake      # launcher en memoria (no toca el multiplexor)
wsdash --no-scan           # solo proyectos de la config
```

Teclas: `j/k` o `↑/↓` mover · `enter` arrancar · `tab` mostrar/ocultar
escaneados · `r` refrescar · `L` cambiar backend · `q` salir.

## Config

Los proyectos configurados son los principales. Los repos que encuentra el
escaneo (`scan.roots`, `scan.depth`) quedan ocultos hasta pulsar `tab`. Un
proyecto configurado siempre gana: se descarta el escaneado con el mismo nombre
o la misma ruta.

```toml
default_backend = "herdr"
backend_order = ["herdr", "tmux", "tuios", "fake"]

[scan]
roots = ["~/dev"]
depth = 3
exclude = ["node_modules", ".git", "worktrees"]

[projects.prdash]
path = "~/dev/projects/prdash"
description = "multi-forge PR inbox"
tags = ["go", "tui"]

[projects.prdash.workspaces.dev]
root = "~/dev/projects/prdash"

[[projects.prdash.workspaces.dev.tabs]]
label = "editor"
[[projects.prdash.workspaces.dev.tabs.panes]]
command = "nvim"
[[projects.prdash.workspaces.dev.tabs.panes]]
split = "down"          # right | down
ratio = 0.3
command = "make test"
wait_for = "PASS"       # opcional, solo backends que lo soportan

[[projects.prdash.workspaces.dev.tabs]]
label = "shell"
[[projects.prdash.workspaces.dev.tabs.panes]]
```

El layout es **portable**: no contiene ningún token de Herdr, tmux ni tuios. Cada
backend lo compila a sus propios comandos (ver `docs/DESIGN.md`).

## Estado

MVP. Herdr es el único backend que compila y ejecuta de verdad. tmux y tuios
están registrados (`Available` comprueba el binario) y `Launch` devuelve
`ErrNotImplemented`.

## Desarrollo

```bash
make check    # build + gofmt + vet + test -race
make test
make run
```
