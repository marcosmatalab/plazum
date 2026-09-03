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
frente_A="paquetes/soc2/ paquetes/pci-dss/ paquetes/tisax/ docs/hallazgos-censo-a.md"
frente_B="paquetes/ai-act/ paquetes/iso42001/ paquetes/ens/ paquetes/iso27001/ paquetes/rgpd/ paquetes/nis2-ue/ paquetes/nis1-es/ docs/hallazgos-censo-b.md"
frente_C="superficies/pantallas/ superficies/camino/ superficies/acta/ superficies/uar/plantillas/ adaptadores/catalogo/cadenas/"
frente_D="cmd/plazum/ superficies/calendario/ superficies/escalado/ perfiles/"

columnas_de() {
  case "$1" in
    A|a) echo "$frente_A" ;;
    B|b) echo "$frente_B" ;;
    C|c) echo "$frente_C" ;;
    D|d) echo "$frente_D" ;;
    *) echo "" ;;
  esac
}

ficheros_de() {
  git diff --name-only "$1" "$2"
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
