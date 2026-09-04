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

deposito=".git/mutaciones"
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

    # GUARDA 3: COMPILA. Un fallo de build no produce `--- FAIL`, asi que
    # parece que la puerta no caza la mutacion. Se comprueba con `go build`
    # ENTERO y por su codigo de salida, nunca por grep de un mensaje.
    if ! salida_build="$(go build ./... 2>&1)"; then
      echo "PARADA: la mutacion NO COMPILA." >&2
      printf '%s\n' "$salida_build" | sed 's/^/    /' >&2
      echo "  Un fallo de build no produce lineas --- FAIL, asi que su rojo NO" >&2
      echo "  demuestra que la puerta cace nada. Arregla la mutacion." >&2
      exit 2
    fi
    echo "compila."

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
