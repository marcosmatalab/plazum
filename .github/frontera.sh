#!/usr/bin/env bash
# LA MATRIZ DE FRONTERAS DE UNA CAMPANA EN PARALELO, COMPROBADA Y NO MIRADA.
#
# POR QUE EXISTE. Con cuatro frentes a la vez, la frontera se verifica con
# `git diff --name-only` en los dos sentidos antes de cada merge, y un fichero
# fuera de su columna es un merge RECHAZADO, no una excepcion. Hacer eso a ojo
# sobre cuatro listas de treinta ficheros es exactamente como se cuela una
# violacion: la lista larga se lee por encima, y la unica linea que importa es
# la que no esperabas.
#
# LOS DOS SENTIDOS, y ninguno sobra:
#
#   1. QUE TOCO EL FRENTE QUE NO ES SUYO. Es el que se busca, y es el que
#      destruye trabajo ajeno al fusionar.
#   2. QUE FICHEROS COMPARTEN DOS FRENTES. Este NO lo ve el sentido 1: dos
#      frentes pueden estar los dos DENTRO de su columna y aun asi tocar el
#      mismo fichero, si la matriz esta mal escrita. Ese caso no es un frente
#      desobediente, es una particion mal hecha, y se descubre igual de tarde.
#
# Uso:
#   .github/frontera.sh <frente> <base> <rama>      un frente contra su columna
#   .github/frontera.sh --cruce <base> <ramaA> <ramaB> [...]   frentes entre si
#
# Salida 0 si la frontera se respeta; 1 si no, con las lineas infractoras.
set -uo pipefail

cd "$(dirname "$0")/.." || exit 1

# LA MATRIZ. Es DATO y va aqui, en un solo sitio: una segunda copia en la cabeza
# del integrador es la que se queda vieja a mitad de campana.
#
# Cada frente son prefijos de ruta. Un fichero cuenta como suyo si empieza por
# alguno. Se anade el fichero de hallazgos propio de cada frente de corpus,
# porque docs/censo-relojes.md no lo toca nadie durante la campana: dos frentes
# escribiendo en el mismo documento de prosa es un conflicto garantizado que
# ademas no caza ningun test.
frente_A="adaptadores/usuarios/ superficies/serve/ cmd/plazum/ adaptadores/catalogo/cadenas/ docs/hallazgos-entrada.md"
frente_B="superficies/calendario/ nucleo/pantalla/ conservacion_calendario_test.go docs/hallazgos-conservacion.md"
frente_C="superficies/pantallas/ nucleo/corpus/ ttfv_camino_test.go docs/hallazgos-entrevista.md"
frente_D="herramientas/ingestanorma/ corpus-vigilancia/ vigencias_test.go docs/hallazgos-vigencias.md"

# LO QUE NO ES DE NADIE, y por que:
#
#   docs/censo-relojes.md      dos frentes escribiendo en el mismo documento de
#                              prosa es un conflicto garantizado que ademas no
#                              caza ningun test. Cada frente escribe su propio
#                              fichero de hallazgos y el integrador fusiona.
#   nucleo/corpus/             lo toca el integrador y una sola vez: encender
#     primitivas_encendidas.go `preaviso` es una linea, y dos frentes que la
#                              escriben a la vez producen un conflicto en el
#                              unico fichero que dice si el motor esta cableado.
#   ETAPAS.md, README.md       las casillas y los numeros publicados los mueve
#                              quien integra, cuando el trabajo ya esta dentro.

columnas_de() {
  case "$1" in
    A|a) echo "$frente_A" ;;
    B|b) echo "$frente_B" ;;
    C|c) echo "$frente_C" ;;
    D|d) echo "$frente_D" ;;
    *) echo "" ;;
  esac
}

# INTEGRACION es la rama contra la que se fusiona. Se puede fijar por entorno,
# pero por defecto es `main`, y NO se pide como argumento a proposito.
INTEGRACION="${PLAZUM_INTEGRACION:-main}"

# ficheros_de da lo que la rama cambia y que NO esta ya en la rama de
# integracion, calculando el merge-base con ELLA.
#
# POR QUE ASI Y NO CONTRA LA BASE QUE LE PASEN, y costo un falso positivo el
# 03-09-2026: los frentes REBASAN sobre main mientras la campana corre. Si se
# compara contra el inicio de la campana, el diff de un frente rebasado incluye
# TODO lo que otros frentes ya integraron, y la comprobacion acusa de romper la
# frontera a quien no la rompio. Paso con el frente D: 74 ficheros «fuera de su
# columna», y 71 eran de los frentes A y C, ya en main.
#
# El arreglo obvio (merge-base con la referencia que le pasen) NO SIRVE: si esa
# referencia es vieja, el merge-base con ella sigue siendo ella. Lo unico que da
# la respuesta correcta es el merge-base con la rama de integracion de AHORA, y
# eso el script lo sabe solo. Por eso deja de ser un argumento.
#
# Un falso positivo aqui no es ruido: es rechazar el merge de un frente limpio,
# o sea tirar su trabajo por un error de quien integra.
ficheros_de() {
  local base
  base=$(git merge-base "$INTEGRACION" "$2" 2>/dev/null) || base="$1"
  git diff --name-only "$base" "$2"
}

# --- sentido 2: dos frentes que tocan el mismo fichero ---
if [ "${1:-}" = "--cruce" ]; then
  shift
  base="${1:-}"; shift
  if [ -z "$base" ] || [ $# -lt 2 ]; then
    echo "uso: .github/frontera.sh --cruce <base> <ramaA> <ramaB> [...]" >&2
    exit 2
  fi
  rojo=0
  ramas=("$@")
  # LA MISMA GUARDA QUE EN EL OTRO SENTIDO, y faltaba: si TODAS las ramas
  # tienen el diff vacio, no hay cruces porque no hay nada, y decir «sin
  # cruces» seria un verde que no significa nada. Se guardo una via y no la
  # otra, que es como se queda medio cerrada una puerta.
  contenido=0
  for r in "${ramas[@]}"; do
    if [ -n "$(ficheros_de "$base" "$r")" ]; then contenido=$((contenido + 1)); fi
  done
  if [ "$contenido" -lt 2 ]; then
    echo "solo $contenido de las ${#ramas[@]} ramas cambian algo respecto de $base." >&2
    echo "  Con menos de dos ramas con contenido no hay cruce que buscar, y" >&2
    echo "  decir «sin cruces» seria un verde vacio." >&2
    exit 1
  fi
  for ((i = 0; i < ${#ramas[@]}; i++)); do
    for ((j = i + 1; j < ${#ramas[@]}; j++)); do
      comunes=$(comm -12 \
        <(ficheros_de "$base" "${ramas[$i]}" | sort) \
        <(ficheros_de "$base" "${ramas[$j]}" | sort))
      if [ -n "$comunes" ]; then
        echo "CRUCE entre ${ramas[$i]} y ${ramas[$j]}:"
        echo "$comunes" | sed 's/^/    /'
        echo "  Los dos frentes pueden estar DENTRO de su columna y aun asi"
        echo "  pisarse: eso no es un frente desobediente, es la matriz mal"
        echo "  escrita. Se resuelve decidiendo de quien es el fichero, no"
        echo "  fusionando y viendo que pasa."
        rojo=1
      fi
    done
  done
  if [ "$rojo" -eq 0 ]; then
    echo "sin cruces: los ${#ramas[@]} frentes tocan conjuntos disjuntos."
  fi
  exit "$rojo"
fi

# --- sentido 1: un frente fuera de su columna ---
frente="${1:-}"; base="${2:-}"; rama="${3:-}"
if [ -z "$frente" ] || [ -z "$base" ] || [ -z "$rama" ]; then
  echo "uso: .github/frontera.sh <frente> <base> <rama>" >&2
  exit 2
fi
columnas=$(columnas_de "$frente")
if [ -z "$columnas" ]; then
  echo "frente desconocido: $frente (son A, B, C, D)" >&2
  exit 2
fi

tocados=$(ficheros_de "$base" "$rama")
if [ -z "$tocados" ]; then
  # CERO FICHEROS FUERA DE LA COLUMNA ES LITERALMENTE CIERTO Y NO SIGNIFICA
  # QUE LA FRONTERA SE RESPETE: significa que no hay trabajo, o que la base
  # esta mal. Misma familia que `go test` saliendo 0 cuando el patron no casa.
  echo "la rama $rama no cambia nada respecto de $base." >&2
  echo "  Eso NO es una frontera respetada: es que no hay trabajo, o que la" >&2
  echo "  base esta mal. Se dice en vez de dar verde." >&2
  exit 1
fi

fuera=""
dentro=0
while IFS= read -r f; do
  [ -z "$f" ] && continue
  suyo=0
  for pref in $columnas; do
    case "$f" in
      "$pref"*) suyo=1; break ;;
    esac
  done
  if [ "$suyo" -eq 1 ]; then
    dentro=$((dentro + 1))
  else
    fuera="$fuera$f"$'\n'
  fi
done <<< "$tocados"

if [ -n "$fuera" ]; then
  echo "FRONTERA ROTA por el frente $frente ($rama):"
  printf '%s' "$fuera" | sed 's/^/    /'
  echo "  Su columna es:"
  for pref in $columnas; do echo "    $pref"; done
  echo
  echo "  Un fichero fuera de su columna es un MERGE RECHAZADO, no una"
  echo "  excepcion. Si el frente lo necesitaba de verdad, la particion estaba"
  echo "  mal hecha y se decide antes de fusionar, no despues."
  exit 1
fi

echo "frontera del frente $frente respetada: $dentro ficheros, todos en su columna."
