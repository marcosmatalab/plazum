---
description: Ejecuta todas las puertas de la etapa en curso y reporta qué casillas de ETAPAS.md se pueden marcar
---
Ejecuta **`./comprobar.sh`**, y nada más para afirmar. Es el objetivo único: hace `gofmt` sobre `git ls-files '*.go'` (no sobre `.`, que entra en los worktrees de los agentes bajo `.claude/`), `go vet ./...`, `go build ./...`, y después lee **todas** las invocaciones de `puerta()` de `.github/workflows/*.yml` y las corre a través de `.github/puerta.sh`, que cuenta los casos ejecutados y exige el mínimo declarado.

**Ningún número que no salga de ahí vale para un informe.** `go test` con `-run` a mano sale con código 0 cuando el patrón no casa con nada y cuando el glob de paquetes no tiene tests: sirve para depurar mientras arreglas algo, nunca para decir que algo está hecho. Ya mordió tres veces en este proyecto, la última en el lazo local.

Si `comprobar.sh` sale en rojo, la salida ya dice qué puerta y con qué arreglo: no hace falta reejecutar nada suelto para averiguarlo.

Después abre ETAPAS.md, localiza la etapa en curso, y para cada casilla sin marcar di: si su test-puerta existe y está en verde (propón marcarla), si existe y está en rojo (di qué falla), o si aún no existe (di qué test habría que escribir según docs/guia.md). No marques nada tú: propón el diff de ETAPAS.md y espera confirmación.
