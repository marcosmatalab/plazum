# Entorno de desarrollo

## Requisitos

| Qué | Versión | Por qué |
|---|---|---|
| Go | 1.24+ | `go version` |
| git | cualquiera | el lazo local lee el índice, no el disco |
| Un compilador de C | opcional, ver abajo | `go test -race` exige cgo |

No hay npm, ni Makefile, ni generadores en el producto. `go build ./...` construye
todo y el módulo no tiene ni una dependencia externa.

## El objetivo único

```bash
./comprobar.sh
```

Formato, `vet`, `build`, **las puertas de CI con su recuento** y las herramientas
de seguridad bloqueantes. Las puertas **no se declaran ahí**: se leen de
`.github/workflows/*.yml`, porque una segunda lista es una lista que se queda
vieja. Tarda unos 15 minutos.

**Ningún resultado de test cuenta en un informe si no salió de ahí.** `go test`
sale con código 0 en dos situaciones donde no ha comprobado nada —un patrón
`-run` que no casa, un glob de paquetes sin tests— y en las dos el verde es
indistinguible de uno de verdad. `go test -run` queda para **depurar**, nunca
para **afirmar**.

Para empujar:

```bash
.github/empujar.sh
```

Se niega a empujar con el árbol sucio, con `HEAD` desprendido, con el lazo en
rojo, con el lazo que **no llegó a dar veredicto** (que no es lo mismo) y con el
remoto movido por debajo. Después espera a CI y cuelga su código de salida de lo
que CI diga.

## El detector de carreras y cgo

Tres de las 26 puertas corren con `go test -race`, y el detector exige cgo. Sin
un compilador de C en el PATH, `comprobar.sh` **las salta y lo dice** —no las
esconde— y el lazo local corre 23 de 26. En CI corren las 26, sobre
`ubuntu-latest`.

Ese hueco costó caro una vez: una versión de `nombraA` que compilaba una
expresión regular por comparación dejaba `plazum serve` sin responder dentro de
los 5 s que le da su test, y lo cazó la puerta de carreras **de CI**. El único
sitio donde se veía era el único donde no se miraba en local.

En Windows, sin permisos de administrador:

```powershell
winget install --id BrechtSanders.WinLibs.POSIX.UCRT --scope user
```

Deja `gcc` bajo `%LOCALAPPDATA%/Microsoft/WinGet/Packages/.../mingw64/bin`.
Añade ese directorio al PATH del usuario y abre una consola **nueva**: la que ya
estaba abierta conserva el PATH viejo. `go env CGO_ENABLED` pasa a `1`.

La otra vía es correr la suite desde WSL sobre el mismo árbol (`/mnt/c/...`),
donde `gcc` ya viene; hay que instalar Go dentro.

## Cómo entra un cambio

1. Implementar **con su test-puerta en el mismo cambio**. Sin puerta no entra.
2. La puerta nace con **su fallo demostrado**: se rompe a propósito lo que
   vigila y la salida roja va en el cuerpo del commit. Una comprobación que
   nunca se ha visto fallar no es una puerta.
3. `./comprobar.sh` en verde. Nunca se commitea con tests en rojo.
4. Commits pequeños, con el **porqué** en el cuerpo.

Las reglas completas, cada una con la fecha del día en que algo se rompió por no
tenerla, están en [`invariantes.md`](invariantes.md). El corte de capas y el test
que vigila cada flecha, en [`arquitectura.md`](arquitectura.md).

<a id="como-se-construyo"></a>

## Cómo se construyó

Lo escribió una persona con asistencia intensiva de IA, y eso es lo de menos:
importa el régimen bajo el que se escribió, porque es lo que se puede ir a
comprobar. **Nada se da por bueno sin una comprobación que se haya visto fallar**,
y las tres reglas de arriba no son una aspiración, son las que dejaron el rastro
que hay en el historial.

Las consecuencias, que son las que se miran y no la autoría:

- **Cada cifra publicada la deriva un test del árbol.** Ninguna se escribe a
  mano, en ningún documento: el tamaño del binario, la cobertura del núcleo, los
  casos de test, los paquetes de corpus, los hitos, los dorados y el porcentaje
  de la v1 tienen todos su puerta, y CI se pone rojo si el documento y el árbol
  se separan en cualquiera de los dos sentidos.
- **Toda puerta nació con su fallo demostrado**, y el commit que la trajo lleva
  la salida roja pegada. Las que nacieron verdes se anotan como tales, con fecha
  de revisión, en [`puertas-nacidas-verdes.md`](puertas-nacidas-verdes.md):
  nacer verde no es lo mismo que vigilar, y las dos cosas hay que saberlas.
- **Lo que está mal o a medias no se disimula.** Hay dos P0 abiertos declarados
  en el [backlog](pendientes.md), con sus cardinales derivados de la tabla.
- **Los errores de medida están escritos con su fecha**, incluidos los que
  favorecían: un presupuesto que subió al volverse honesta la medida y no se
  movió, tres cifras que se cayeron el mismo día por mezclar bancos de medida, y
  un recuento corregido dos veces. Están en
  [`erratas.md`](erratas.md) y en [`deslices.md`](deslices.md).

El historial es denso porque el ciclo fue corto, y no es eso lo que sostiene el
resultado: lo sostiene que `./comprobar.sh` salga en verde con 26 puertas y que
`GOPROXY=off go test ./...` pase sin tocar la red. Las dos cosas se ejecutan en
quince minutos sin pedir permiso a nadie.

## Trabajo en paralelo

Los frentes que no comparten ficheros se construyen a la vez en worktrees contra
interfaces congeladas. Un worktree **no cambia un puerto por su cuenta**: escribe
la propuesta en [`puertos-propuestas.md`](puertos-propuestas.md), sigue contra el
interfaz actual y no para. Las decisiones de puertos se resuelven en lote.

Un checkout tiene **un solo integrador**, y `main` es suyo: todo lo demás entrega
rama y SHA.
