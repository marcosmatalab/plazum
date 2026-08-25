# Arranque en local: VS Code + GitHub (marcosmatalab)

Una sola vez, en tu maquina:

## 1. Requisitos

- Go 1.24+ (`go version` para comprobar; si no: https://go.dev/dl/)
- git y la CLI de GitHub `gh` (https://cli.github.com/), con sesion iniciada: `gh auth login`
- VS Code con las extensiones recomendadas (este repo las sugiere solo al abrirlo: Go y Claude Code)
- Claude Code: `npm install -g @anthropic-ai/claude-code` o el instalador de https://claude.com/claude-code

## 2. Crear el repo privado y subirlo (desde este directorio)

```bash
go test ./... -count=1          # linea base: todo verde antes de tocar nada
gh repo create marcosmatalab/plazum --private --source=. --push
```

Con eso el historial completo (5 commits) queda en GitHub y el CI arranca solo
en el primer push (build, tests, puertas, govulncheck).

## 3. Ajustes del repo en GitHub (2 minutos, una vez)

- Settings -> Security -> activar "Private vulnerability reporting" (casilla de la semana 0)
- Settings -> General -> Default branch: `main` (ya lo es)
- Nada mas mientras sea privado y trabajes solo: las branch protections llegan al hacerlo publico

## 4. El dia a dia en VS Code

Abre la carpeta en VS Code. El repo ya trae la configuracion:

- **Guardar = formatear** (gofmt) y **commitear = push automatico** a GitHub
  (`git.postCommitCommand: push`): sincronizado todo el rato, sin pensar en ello.
- Terminal integrada -> `claude` -> `/etapa` te situa y propone la casilla del dia.
- Ctrl/Cmd+Shift+B no esta mapeado a build sino a tests: la tarea por defecto es
  `go test ./... -count=1`. Las otras dos tareas: puertas de raiz y cobertura del corpus.
- El bucle: /etapa -> plan -> implementar con su test -> /puerta -> commit (y el
  push sale solo) -> /clear y siguiente casilla. Al cerrar etapa: /adversarial.

## 5. Regla de oro de la sincronizacion

Se commitea SOLO en verde (`go test ./...` limpio), y cada commit viaja solo a
GitHub. Si algo se rompe a medias, no se commitea: se arregla o se hace stash.
Asi el remoto siempre es un estado sano que puedes clonar en cualquier maquina.

Cuando la v0.2 este lista y decidas la casa definitiva: crear la organizacion,
Settings -> Transfer ownership (GitHub redirige las URLs viejas solo), y en ese
mismo momento cambiar el module path a `github.com/ORG/plazum` en un commit
(go.mod + sed de imports) para que `go install` funcione a los usuarios.
