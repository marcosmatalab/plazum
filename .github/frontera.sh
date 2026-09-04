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
# TRAMO 3 (dias 7 a 10). CUATRO REBANADAS DISJUNTAS DE VERDAD.
#
# La unidad de particion sigue siendo la FUNCIONALIDAD y no el directorio (esa
# leccion es del tramo 2 y no se toca). Lo que cambia aqui es que el tramo 2
# dejo UN solape declarado y resuelto en el tiempo, y este no deja ninguno: las
# cuatro columnas son disjuntas y `--cruce` tiene que salir vacio con las cuatro
# ramas vivas, sin excepciones que recordar.
#
# COMO SE CONSIGUIO, porque no salio solo:
#
#   el catalogo de cadenas se va ENTERO con quien tiene las pantallas (R1), y
#     este tramo tiene un solo frente de pantallas a proposito. R3 construye los
#     cimientos de la IA y NO toca pantalla, dicho en su encargo: las cinco
#     piezas de adopcion y su integracion visual van al tramo 4, encima.
#   cmd/plazum/ se va con el empaquetado (R0), que es quien lo publica.
#   ttfv_camino_test.go se va con R1, que es quien mueve el TTFV. Es la regla
#     del tramo 2: cada fichero de raiz a la rebanada que MUEVE EL NUMERO que
#     ese fichero congela.
rebanada_0=".github/workflows/release.yml .github/esperar-ci.sh .github/mutar.sh Dockerfile cmd/plazum/ distribucion_test.go docs/lanzamiento/ docs/instalacion.md docs/hallazgos-release.md"
rebanada_1="superficies/pantallas/ superficies/camino/ superficies/acta/ superficies/uar/plantillas/ superficies/calendario/ superficies/escalado/ adaptadores/catalogo/cadenas/ ttfv_camino_test.go docs/hallazgos-d11.md"
rebanada_2="paquetes/ docs/censo-relojes.md docs/hallazgos-corpus-t3.md"
rebanada_3="adaptadores/ia/ adaptadores/busqueda/ puertos/ evals/ herramientas/ ia_test.go docs/ia.md docs/hallazgos-ia.md"

# LO QUE NO ES DE NADIE, y por que:
#
#   ETAPAS.md, README.md       las casillas y los numeros publicados los mueve
#     docs/marcador.md         quien integra, cuando el trabajo ya esta dentro.
#     docs/instantanea.md      ademas esta CONGELADA: se rehace entera en el
#                              tramo 4 y hasta entonces sus numeros no salen de
#                              ella (instantanea_congelada_test.go).
#   nucleo/                    NADIE lo toca en este tramo. R3 construye la IA
#                              en adaptadores y puertos, y el invariante 9 dice
#                              que el nucleo no conoce el puerto de IA y NI
#                              SIQUIERA LO NOMBRA. Un frente que necesite tocar
#                              nucleo/ para su trabajo de IA esta rompiendo el
#                              invariante y tiene que parar y decirlo.
#   ci.yml, .github/puerta.sh  las puertas compartidas. Cambiarlas a mitad de
#     comprobar.sh             campana caduca lo que ya se valido. La EXCEPCION
#                              es el paso de PLAZUM_SIN_IA=1 que pide R3: se
#                              escribe en su informe y lo mete el integrador, en
#                              un commit propio, cuando su rama este dentro.
#   CLAUDE.md, este fichero    los escribe el integrador, y solo el.
#
# Y LA REGLA QUE ESTE TRAMO ESTRENA, sacada de que el anterior no lo hizo: la
# matriz se empuja Y SE PASA EL LAZO ENTERO antes de lanzar los frentes. En el
# tramo 2 se empujo sin lazo, el renombrado rompio dos puertas de raiz, y los
# tres frentes gastaron tiempo diagnosticando un rojo del integrador.

columnas_de() {
  case "$1" in
    0|release)  echo "$rebanada_0" ;;
    1|visual)   echo "$rebanada_1" ;;
    2|corpus)   echo "$rebanada_2" ;;
    3|ia)       echo "$rebanada_3" ;;
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
  # RAMAS APILADAS: UNA CONTIENE A LA OTRA, Y ESO NO ES UN CRUCE.
  #
  # El 04-09-2026 el tramo 2 tuvo que apilar la rebanada de los valores SOBRE la
  # del puente, porque una pantalla que manda valores solo significa algo contra
  # un corpus que declara que hecho produce cada respuesta. Con las dos vivas,
  # `--cruce` acuso 26 ficheros compartidos.
  #
  # ERAN TODOS FALSOS. `ficheros_de` calcula el diff contra la rama de
  # INTEGRACION, y la rama de arriba todavia no tenia dentro a la de abajo, asi
  # que su diff contra `main` incluia el de la otra ENTERO. El script sabia
  # distinguir una rama rebasada de una que invade, y no sabia distinguir una
  # rama APILADA. La limitacion la creo quien integra al apilarlas, no el frente.
  #
  # POR QUE NO BASTA CON CALLARSE EL PAR. Un falso positivo que se ignora a mano
  # se convierte en «esta puerta siempre grita, no la mires», y entonces el cruce
  # DE VERDAD del proximo tramo pasa por delante de los ojos de alguien que ya
  # decidio no creersela. Asi que se detecta, se dice en voz alta, y se compara
  # el conjunto que si significa algo.
  #
  # Y NO EXCUSA NADA: si la de arriba vuelve a tocar un fichero que la de abajo
  # ya habia tocado, ese fichero SIGUE saliendo en los dos conjuntos y sigue
  # siendo un cruce. Lo unico que se quita del conjunto de arriba es lo que
  # heredo sin tocarlo.
  base_del_par() {
    # Devuelve contra que hay que diffear la rama $2 cuando se la compara con
    # $1: la propia $1 si $1 es antepasada de $2, y la integracion si no.
    if git merge-base --is-ancestor "$1" "$2" 2>/dev/null; then
      echo "$1"
    else
      echo "$INTEGRACION"
    fi
  }
  ficheros_contra() {
    # $1 = referencia contra la que diffear, $2 = rama
    local b
    b=$(git merge-base "$1" "$2" 2>/dev/null) || b="$1"
    git diff --name-only "$b" "$2"
  }
  for ((i = 0; i < ${#ramas[@]}; i++)); do
    for ((j = i + 1; j < ${#ramas[@]}; j++)); do
      a="${ramas[$i]}"; z="${ramas[$j]}"
      base_a=$(base_del_par "$z" "$a")
      base_z=$(base_del_par "$a" "$z")
      if [ "$base_a" != "$INTEGRACION" ] || [ "$base_z" != "$INTEGRACION" ]; then
        echo "APILADAS: $a y $z. Una contiene a la otra, asi que la de arriba se"
        echo "  compara contra la de abajo y no contra $INTEGRACION. Lo que hereda"
        echo "  sin tocar no cuenta; lo que vuelve a tocar, si."
      fi
      comunes=$(comm -12 \
        <(ficheros_contra "$base_a" "$a" | sort) \
        <(ficheros_contra "$base_z" "$z" | sort))
      if [ -n "$comunes" ]; then
        echo "CRUCE entre $a y $z:"
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
  echo "uso: .github/frontera.sh <rebanada|1|2|3|4> <base> <rama>" >&2
  exit 2
fi
columnas=$(columnas_de "$frente")
if [ -z "$columnas" ]; then
  echo "rebanada desconocida: $frente (son 0|release, 1|visual, 2|corpus, 3|ia)" >&2
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
