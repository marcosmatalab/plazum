# Arranque en local: VS Code + GitHub (marcosmatalab)

Una sola vez, en tu maquina:

## 1. Requisitos

- Go 1.24+ (`go version` para comprobar; si no: https://go.dev/dl/)
- git y la CLI de GitHub `gh` (https://cli.github.com/), con sesion iniciada: `gh auth login`
- VS Code con las extensiones recomendadas (este repo las sugiere solo al abrirlo: Go y Claude Code)
- Claude Code: `npm install -g @anthropic-ai/claude-code` o el instalador de https://claude.com/claude-code

## 2. Crear el repo privado y subirlo (desde este directorio)

```bash
./comprobar.sh                 # linea base: todo verde antes de tocar nada
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
- Ctrl/Cmd+Shift+B no esta mapeado a build sino a la comprobacion entera: la
  tarea por defecto es `./comprobar.sh`, el objetivo unico. Las otras tareas:
  puertas de raiz y cobertura del corpus, que son para DEPURAR.
- El bucle: /etapa -> plan -> implementar con su test -> /puerta -> commit (y el
  push sale solo) -> /clear y siguiente casilla. Al cerrar etapa: /adversarial.

## 5. Regla de oro de la sincronizacion

Se commitea SOLO en verde, y verde significa **`./comprobar.sh` limpio**, no
`go test` suelto: `go test` sale con codigo 0 cuando el patron `-run` no casa
con nada y cuando el glob de paquetes no tiene tests, asi que un `ok` de un
`-run` a mano no dice nada. `comprobar.sh` lee las puertas de los workflows y
las corre con su recuento. Ningun numero que no salga de ahi vale para un
informe. Y cada commit viaja solo a GitHub. Si algo se rompe a medias, no se commitea: se arregla o se hace stash.
Asi el remoto siempre es un estado sano que puedes clonar en cualquier maquina.

HECHO EL 28-08-2026, la mitad del plan que quedaba: el repositorio se hizo
publico, se renombro de `dutiq` a `plazum` (GitHub redirige el nombre viejo) y
el module path paso de `plazum` a `github.com/marcosmatalab/plazum`, que es lo
que hace que `go install github.com/marcosmatalab/plazum/cmd/plazum@latest`
funcione. Un modulo desnudo no se puede instalar, y en publico eso importa.

El renombrado del modulo se llevo por delante cinco puertas que llevaban la
ruta vieja escrita a mano, y DOS DE ELLAS SE QUEDARON VERDES vigilando el
vacio. La leccion, con la medida, en `internal/modulo`; la puerta que impide
que vuelva a pasar es `TestNadieCableaLaRutaDelModulo`.

Lo que sigue pendiente es la casa definitiva: si algun dia se crea la
organizacion, Settings -> Transfer ownership (GitHub redirige las URLs viejas
solo) y el module path vuelve a cambiar. Esa vez sera un `go mod edit` y una
reescritura de imports, y NINGUN test habra que tocarlo.
