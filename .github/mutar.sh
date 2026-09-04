#!/usr/bin/env bash
# MUTAR UNA PUERTA EN DOS PASOS SEPARADOS, CON EL CONTADOR COMPROBADO ANTES Y
# DESPUES.
#
# # Por que existe
#
# La pasada 2 exige romper a proposito lo que una puerta vigila y demostrar el
# rojo. Hacerlo a mano tiene cuatro trampas, y las cuatro se han cometido en
# este repositorio:
#
#  1. EL ARBOL SUCIO. La mutacion se aplica sobre trabajo sin commitear y
#     restaurar se lo lleva por delante. Paso el 27-08-2026, dos veces el mismo
#     dia.
#  2. LA MUTACION QUE NO SE APLICA. Un `sed` cuyo patron no casa deja el fichero
#     igual, la puerta sale verde, y ese verde se lee como «no lo caza». Paso el
#     04-09-2026 con la M8 de la puerta del ensayo y con la M5 de la del
#     marcador: dos falsos hallazgos en un dia.
#  3. LA MUTACION QUE NO COMPILA. Un fallo de build no produce lineas `--- FAIL`,
#     asi que parece que la puerta no la caza. Y no vale con `grep cannot|
#     undefined`: `imported and not used` no lleva ninguna de esas palabras.
#  4. RESTAURAR CON `git checkout`. Restaura a HEAD, no al estado previo: si
#     habia trabajo sin commitear, se pierde. Se restaura con la COPIA.
#
# Ninguna de las cuatro se arregla con cuidado. Se arreglan con guardas, que es
# lo que este script es.
#
# # Los dos pasos, y por que van separados
#
#   .github/mutar.sh preparar <fichero> [<fichero>...]
#   ...aplicas la mutacion a mano, con el editor o con sed...
#   .github/mutar.sh comprobar "<orden de test>"
#   .github/mutar.sh restaurar
#
# Separados porque la mutacion la escribe una persona y tiene que poder ser
# cualquier cosa. Lo que el script garantiza es el ANTES y el DESPUES: que el
# arbol estaba limpio, que el cambio se aplico de verdad, que compila, y que se
# restaura desde la copia y no desde git.
set -uo pipefail

cd "$(dirname "$0")/.." || exit 1

# DONDE SE GUARDA LA COPIA. Se pregunta a git en vez de escribir ".git", Y ESE
# ES UN FALLO QUE ESTE SCRIPT TUVO EN SU PRIMER DIA: en un WORKTREE, `.git` no
# es un directorio sino un FICHERO con una linea `gitdir: ...` dentro, asi que
# `mkdir -p .git/mutaciones` falla y el script no arranca.
#
# Y ESA ERA LA UNICA FORMA EN QUE NO PODIA FALLAR SIN QUE SE NOTARA: las cuatro
# rebanadas de una campana trabajan EN WORKTREES, o sea que la herramienta
# escrita para que las mutaciones no se hagan a ojo estaba rota exactamente
# donde se iba a usar, y funcionaba solo en el checkout del integrador, que es
# el unico sitio donde casi no se usa. Lo encontro un frente al intentar usarla,
# no una lectura.
#
# Y LO ENCONTRARON DOS A LA VEZ, cada uno en su worktree, con lo que el arreglo
# se escribio dos veces. Esta version es la del integrador mas la guarda de
# abajo, que traia la otra: si `git rev-parse --git-dir` NO contesta (porque
# esto no se ejecuta dentro de un repositorio), la sustitucion se queda vacia y
# el deposito seria `/mutaciones`, o sea la raiz del sistema de ficheros. Ahi
# `mkdir -p` puede fallar o, peor, funcionar.
#
# Es la misma regla que ya lleva escrita .github/esperar-ci.sh: una medida que
# no se puede tomar NO es «todavia no», es un error, y se dice. La nada que sale
# de una sustitucion fallida no puede convertirse en una ruta plausible.
deposito="$(git rev-parse --git-dir 2>/dev/null)/mutaciones"
if [ "$deposito" = "/mutaciones" ]; then
  echo "PARADA: no estoy dentro de un repositorio git (git rev-parse --git-dir" >&2
  echo "  no contesta), asi que el deposito habria salido en la raiz del sistema" >&2
  echo "  de ficheros. Sin deposito no hay copia, y restaurar desde git es justo" >&2
  echo "  lo que este script existe para no hacer." >&2
  exit 2
fi
manifiesto="$deposito/manifiesto"

uso() {
  echo "uso:" >&2
  echo "  .github/mutar.sh preparar <fichero> [<fichero>...]" >&2
  echo "  .github/mutar.sh comprobar \"<orden de test>\"" >&2
  echo "  .github/mutar.sh restaurar" >&2
  exit 2
}

case "${1:-}" in
  preparar)
    shift
    [ $# -ge 1 ] || uso
    # GUARDA 1: EL ARBOL LIMPIO. No es una recomendacion: con el arbol sucio,
    # restaurar se lleva por delante lo que no estaba commiteado.
    sucio="$(git status --porcelain)"
    if [ -n "$sucio" ]; then
      echo "PARADA: el arbol no esta limpio, y una mutacion sobre trabajo sin" >&2
      echo "commitear se lo lleva por delante al restaurar." >&2
      echo "$sucio" | sed 's/^/    /' >&2
      echo "  Commitea o guarda con git stash antes de mutar." >&2
      exit 2
    fi
    if [ -e "$deposito" ]; then
      echo "PARADA: ya hay una mutacion preparada sin restaurar." >&2
      echo "  Ficheros:" >&2
      sed 's/^/    /' "$manifiesto" >&2
      echo "  Restaura con: .github/mutar.sh restaurar" >&2
      exit 2
    fi
    mkdir -p "$deposito" || exit 2
    : > "$manifiesto"
    for f in "$@"; do
      if [ ! -f "$f" ]; then
        echo "PARADA: $f no existe. Mutar un fichero que no esta deja la puerta" >&2
        echo "  intacta y su verde parece un hallazgo." >&2
        rm -rf "$deposito"
        exit 2
      fi
      destino="$deposito/$(printf '%s' "$f" | tr '/' '_')"
      cp "$f" "$destino" || exit 2
      printf '%s\n' "$f" >> "$manifiesto"
    done
    # LA HUELLA DE ANTES. Es lo que permite decir despues si la mutacion se
    # aplico de verdad, sin depender de que git vea el fichero.
    while IFS= read -r f; do
      sha256sum "$f"
    done < "$manifiesto" > "$deposito/antes"
    echo "preparados $(wc -l < "$manifiesto") fichero(s). Copia en $deposito."
    echo "Aplica ahora la mutacion y despues:"
    echo "  .github/mutar.sh comprobar \"go test . -run TuPuerta\""
    ;;

  comprobar)
    orden="${2:-}"
    [ -n "$orden" ] || uso
    [ -f "$manifiesto" ] || { echo "PARADA: no hay ninguna mutacion preparada." >&2; exit 2; }

    # GUARDA 2: LA MUTACION SE APLICO DE VERDAD. Se compara la huella, no el
    # estado de git: un fichero sin seguimiento no aparece en `git diff` y su
    # mutacion pasaria por no aplicada.
    while IFS= read -r f; do sha256sum "$f"; done < "$manifiesto" > "$deposito/despues"
    if cmp -s "$deposito/antes" "$deposito/despues"; then
      echo "PARADA: NINGUN fichero ha cambiado. La mutacion no se ha aplicado." >&2
      echo "  Un sed cuyo patron no casa deja el fichero igual, la puerta sale" >&2
      echo "  verde, y ese verde se lee como «no lo caza». Es un falso hallazgo," >&2
      echo "  y se ha cometido dos veces en un dia en este repositorio." >&2
      exit 2
    fi
    cambiados=$(diff "$deposito/antes" "$deposito/despues" | grep -c '^>' || true)
    echo "mutacion aplicada: $cambiados de $(wc -l < "$manifiesto") fichero(s) cambiados."

    # GUARDA 3: COMPILA, TESTS INCLUIDOS.
    #
    # SE USA `go vet` Y NO `go build`, Y ESTE SCRIPT SE LO HIZO A SI MISMO EL
    # 04-09-2026, EN SU PRIMERA DEMOSTRACION. `go build ./...` NO compila los
    # ficheros _test.go, asi que una mutacion que rompe un test pasaba la guarda
    # con un «compila» y despues el `go test` fallaba con «build failed», que el
    # script anunciaba como CAZADA. O sea: el script tenia dentro exactamente la
    # trampa numero 3 que existe para evitar, y encima en su forma peor, porque
    # decia en voz alta que habia comprobado lo que no habia comprobado.
    #
    # `go vet ./...` si comprueba los tests. Y su codigo de salida es lo que se
    # mira, nunca un grep del mensaje: `imported and not used` no lleva ni
    # «cannot» ni «undefined», que fue como se colo la primera vez.
    if ! salida_build="$(go vet ./... 2>&1)"; then
      echo "PARADA: la mutacion no compila, o no pasa vet." >&2
      printf '%s\n' "$salida_build" | sed 's/^/    /' >&2
      echo "  Un fallo de compilacion no produce lineas --- FAIL, asi que su rojo" >&2
      echo "  NO demuestra que la puerta cace nada: demuestra que no compila." >&2
      echo "  Arregla la mutacion y vuelve a comprobar." >&2
      exit 2
    fi
    echo "compila (tests incluidos: go vet ./...)."

    echo "--- $orden"
    set +e
    salida_test="$(eval "$orden" 2>&1)"
    codigo=$?
    set -e
    printf '%s\n' "$salida_test" | grep -E '^\s*--- FAIL|^--- FAIL|^FAIL|^ok ' | head -20
    if [ "$codigo" -eq 0 ]; then
      echo
      echo "LA MUTACION HA SOBREVIVIDO: la puerta sigue verde con el cambio puesto."
      echo "  O la puerta no vigila lo que crees, o la mutacion no toca lo que vigila."
      echo "  Las dos cosas hay que saberlas, y ninguna es «ya esta»."
      exit 1
    fi
    echo
    echo "CAZADA: la puerta se pone roja con la mutacion (codigo $codigo)."
    ;;

  restaurar)
    [ -f "$manifiesto" ] || { echo "PARADA: no hay ninguna mutacion preparada." >&2; exit 2; }
    # SE RESTAURA DESDE LA COPIA, NUNCA CON git checkout: checkout restaura a
    # HEAD, no al estado previo, y si el fichero era nuevo falla con «pathspec
    # did not match» dejando la mutacion PUESTA.
    while IFS= read -r f; do
      origen="$deposito/$(printf '%s' "$f" | tr '/' '_')"
      cp "$origen" "$f" || exit 2
    done < "$manifiesto"
    n=$(wc -l < "$manifiesto")
    rm -rf "$deposito"
    echo "restaurados $n fichero(s) desde la copia."
    sucio="$(git status --porcelain)"
    if [ -n "$sucio" ]; then
      echo "AVISO: el arbol NO ha quedado limpio despues de restaurar:" >&2
      echo "$sucio" | sed 's/^/    /' >&2
      exit 1
    fi
    echo "arbol limpio."
    ;;

  *) uso ;;
esac
