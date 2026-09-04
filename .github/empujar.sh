#!/usr/bin/env bash
# EMPUJAR SOLO CON EL LAZO EN VERDE Y EL ARBOL LIMPIO. EL LAZO LO DICE, Y DECIR
# NO ES IMPEDIR.
#
# # Por que existe, y por que llega tarde
#
# `comprobar.sh` lleva semanas diciendo la verdad sobre el arbol. Nunca ha
# impedido nada. El 04-09-2026 el integrador empujo un tramo entero leyendo el
# codigo de salida del mandato EQUIVOCADO tres veces seguidas:
#
#   ./comprobar.sh | tail -5        el codigo que sobrevive es el de tail
#   gh release create ... | sed ... el codigo que sobrevive es el de sed
#   grep -q X fichero && git push   el push cuelga de si el grep caso
#
# Las tres salieron verdes sin haber comprobado lo que decian. Y la de mas
# arriba es la peor de las tres, porque `comprobar.sh` SI se ejecuto y SI habria
# hablado: su codigo de salida se lo comio una tuberia puesta para ver menos
# texto. La herramienta funcionaba; lo que fallo fue el sitio donde se leyo.
#
# LA REGLA QUE SALE DE AHI, y esta en CLAUDE.md: **ninguna orden que decida algo
# va conectada por tuberia.** Se ejecuta, se guarda el codigo de salida en una
# variable, y se lee la variable. En bash un `cmd | filtro` devuelve el codigo
# del ULTIMO tramo, y `set -o pipefail` no basta porque hay que acordarse de
# ponerlo, que es justo lo que no pasa a las tres de la tarde.
#
# Y la segunda leccion, que es la que convierte esto en script: el mismo dia se
# arreglo esa familia DENTRO de un script (`esperar-ci.sh`, guarda del conjunto
# vacio) y se cometio A MANO en la orden con la que se empujaba ese arreglo. La
# disciplina que vive en el producto y no en las manos se pierde en cuanto se
# teclea. Asi que el paso peligroso deja de teclearse.
#
# # Las seis formas de parada, y ninguna se confunde con exito
#
#   1. arbol sucio                     salida 2, sin correr nada
#   2. HEAD desprendido                salida 2
#   3. el lazo en rojo                 salida 1, con las lineas rotas
#   4. el lazo no se pudo ejecutar     salida 2  (no saberlo no es verde)
#   5. el remoto se movio por debajo   salida 3, con la receta exacta
#   6. el push falla                   salida del push, sin adornar
#
# LA CUARTA ES LA HERMANA DE LAS OTRAS TRES y es la que casi no se escribe: un
# `comprobar.sh` que no arranca (permisos, bash ausente, ruta mala) no es un
# lazo limpio. Es el invariante 8 aplicado al proceso: de las dos formas de la
# nada, la peligrosa es la que sale por descuido.
#
# # Uso
#
#   .github/empujar.sh                 empuja la rama actual a origin
#   .github/empujar.sh --sin-lazo      SOLO para ramas de trabajo que no son
#                                      main y cuyo lazo ya paso en esta sesion;
#                                      lo dice en voz alta y lo deja escrito
#
# Al terminar imprime el SHA de origin/<rama>, que es el unico ancla que vale
# para un informe: trabajo sin empujar es trabajo que no existe para quien
# revisa.
set -uo pipefail

cd "$(dirname "$0")/.." || exit 1

sin_lazo=0
case "${1:-}" in
  --sin-lazo) sin_lazo=1 ;;
  "") ;;
  *)
    echo "uso: .github/empujar.sh [--sin-lazo]" >&2
    exit 2
    ;;
esac

# ---------------------------------------------------------------------------
# GUARDA 1: EL ARBOL LIMPIO.
#
# No es higiene. Empujar con el arbol sucio publica un commit cuyo contenido no
# es lo que acaba de comprobarse, y el informe que cite ese SHA estara hablando
# de un arbol que solo existe en esta maquina.
# ---------------------------------------------------------------------------
sucio="$(git status --porcelain)"
codigo=$?
if [ "$codigo" -ne 0 ]; then
  echo "PARADA: no he podido preguntar por el estado del arbol (codigo $codigo)." >&2
  echo "  No saber si esta limpio NO es que este limpio." >&2
  exit 2
fi
if [ -n "$sucio" ]; then
  echo "PARADA: el arbol no esta limpio." >&2
  printf '%s\n' "$sucio" | sed 's/^/    /' >&2
  echo >&2
  echo "  Lo que se empuja tiene que ser lo que se ha comprobado. Commitea, o" >&2
  echo "  aparta con git stash lo que no sea tuyo: una edicion de otro autor no" >&2
  echo "  entra nunca en un commit propio." >&2
  exit 2
fi

# ---------------------------------------------------------------------------
# GUARDA 2: HEAD ESTA EN UNA RAMA.
#
# Con HEAD desprendido, `git push origin HEAD` empuja a algo que no se llama
# como nadie cree. Se para antes de averiguarlo.
# ---------------------------------------------------------------------------
rama="$(git symbolic-ref --quiet --short HEAD)"
codigo=$?
if [ "$codigo" -ne 0 ] || [ -z "$rama" ]; then
  echo "PARADA: HEAD esta desprendido, no hay rama que empujar." >&2
  echo "  git switch -c <rama>  antes de empujar nada." >&2
  exit 2
fi

# ---------------------------------------------------------------------------
# GUARDA 3: EL LAZO, EJECUTADO ENTERO Y CON SU CODIGO LEIDO DE UNA VARIABLE.
#
# AQUI ESTA EL PUNTO DEL SCRIPT, asi que se escribe explicito: la salida se
# guarda en un fichero y el codigo se guarda en `codigo` EN LA LINEA SIGUIENTE,
# antes de que ningun otro mandato lo pise. No hay tuberia en ninguna parte de
# esta seccion. Filtrar la salida se hace DESPUES, sobre el fichero, cuando el
# veredicto ya esta tomado.
# ---------------------------------------------------------------------------
if [ "$sin_lazo" -eq 1 ]; then
  if [ "$rama" = "main" ]; then
    echo "PARADA: --sin-lazo no vale en main." >&2
    echo "  main es la rama de la que sale la release y de la que se citan los" >&2
    echo "  SHA. Si el lazo tarda, tarda." >&2
    exit 2
  fi
  echo "AVISO: --sin-lazo. El lazo NO se ha ejecutado en este empujon."
  echo "  Lo que se empuja a '$rama' no esta comprobado por este script, y"
  echo "  ningun numero de esta sesion puede citarse como salido de la puerta."
else
  registro="$(git rev-parse --git-dir)/ultimo-lazo.log"
  echo "== corriendo ./comprobar.sh entero (esto tarda)"
  ./comprobar.sh >"$registro" 2>&1
  codigo=$?

  if [ "$codigo" -ne 0 ]; then
    # Se distingue "el lazo dijo que no" de "el lazo no pudo hablar". Confundir
    # las dos hace que una maquina rota se lea como una maquina limpia.
    if grep -q "COMPROBACION EN ROJO" "$registro"; then
      echo "PARADA: el lazo esta EN ROJO (codigo $codigo). No se empuja." >&2
      echo >&2
      grep -E "^PASO ROTO|^--- FAIL|^FAIL|ha encontrado algo|PUERTA SALTADA" "$registro" |
        head -30 | sed 's/^/    /' >&2
      echo >&2
      echo "  Salida entera: $registro" >&2
      exit 1
    fi
    echo "PARADA: ./comprobar.sh no llego a dar veredicto (codigo $codigo)." >&2
    tail -20 "$registro" | sed 's/^/    /' >&2
    echo >&2
    echo "  Esto NO es un lazo limpio: es un lazo que no se pudo ejecutar, y las" >&2
    echo "  dos cosas se parecen solo si no se miran. Salida entera: $registro" >&2
    exit 2
  fi

  # Y el verde tampoco se cree por el codigo: se exige la linea que lo dice. Un
  # script que termine antes de tiempo saldria con 0 sin haber corrido nada.
  if ! grep -q "COMPROBACION EN VERDE" "$registro"; then
    echo "PARADA: ./comprobar.sh salio con 0 y NO imprimio su linea de verde." >&2
    tail -20 "$registro" | sed 's/^/    /' >&2
    echo "  Un cero sin veredicto es el verde vacio de siempre." >&2
    exit 2
  fi
  grep "COMPROBACION EN VERDE" -A 4 "$registro" | sed 's/^/    /'
fi

# ---------------------------------------------------------------------------
# GUARDA 4: EL REMOTO NO SE HA MOVIDO POR DEBAJO.
#
# Si se movio, la respuesta es mecanica y esta escrita en CLAUDE.md: fetch,
# rebase, COMPROBAR OTRA VEZ (lo validado contra el arbol anterior ya no esta
# validado) y empujar. Este script no la ejecuta solo, porque un rebase
# automatico sobre trabajo ajeno es justo lo que no debe hacer una herramienta.
# ---------------------------------------------------------------------------
git fetch origin "$rama" --quiet
codigo=$?
if [ "$codigo" -ne 0 ]; then
  echo "AVISO: no he podido consultar el remoto (codigo $codigo)."
  echo "  Sigo, y el propio push dira si hay conflicto: es el que manda."
else
  remoto="$(git rev-parse --quiet --verify "origin/$rama")"
  if [ -n "$remoto" ]; then
    base="$(git merge-base HEAD "origin/$rama")"
    if [ "$base" != "$remoto" ]; then
      echo "PARADA: origin/$rama se ha movido por debajo." >&2
      echo "  local  $(git rev-parse --short HEAD)" >&2
      echo "  remoto $(git rev-parse --short "origin/$rama")" >&2
      echo >&2
      echo "  Receta, y los tres pasos son obligatorios:" >&2
      echo "    git fetch origin" >&2
      echo "    git rebase origin/$rama" >&2
      echo "    ./comprobar.sh          <- lo validado antes del rebase YA NO lo esta" >&2
      echo "    .github/empujar.sh" >&2
      exit 3
    fi
  fi
fi

# ---------------------------------------------------------------------------
# EL EMPUJON. Sin tuberia, con su codigo leido.
# ---------------------------------------------------------------------------
echo "== git push origin $rama"
git push origin "$rama"
codigo=$?
if [ "$codigo" -ne 0 ]; then
  echo "PARADA: el push ha fallado (codigo $codigo)." >&2
  exit "$codigo"
fi

# EL ANCLA DEL INFORME. Se saca del remoto con rev-parse y no se teclea de
# memoria: un identificador que no salio de la salida de un comando de esta
# sesion tiene la FORMA de lo verificable sin serlo, y esa forma es lo que hace
# que nadie vaya a comprobarlo.
git fetch origin "$rama" --quiet
sha="$(git rev-parse "origin/$rama")"
codigo=$?
if [ "$codigo" -ne 0 ] || [ -z "$sha" ]; then
  echo "AVISO: el push fue bien pero no he podido leer origin/$rama." >&2
  echo "  Saca el SHA a mano con: git rev-parse origin/$rama" >&2
  exit 0
fi

echo
echo "EMPUJADO. El SHA que va al informe, sacado de origin y no de memoria:"
echo "  origin/$rama  $sha"
