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
# TRAMO 2 (dias 4 a 6). LA UNIDAD DE PARTICION ES LA FUNCIONALIDAD, NO EL
# DIRECTORIO, y ese es el cambio que trae este tramo.
#
# El tramo 1 partio por carpetas y fallo tres veces por la misma causa: una
# clave de catalogo y el codigo que la pinta son UNA SOLA COSA que vive en dos
# carpetas, y repartirlas por carpeta las separa. Igual `ttfv_camino_test.go` y
# los tres ficheros de raiz del puente, que congelan medidas de un trabajo que
# hacia otro. Asi que cada rebanada es VERTICAL: se le da su funcionalidad
# entera con todos los ficheros que necesita, esten en cuatro carpetas o en
# una, y si dos rebanadas se tocan la particion esta mal hecha y se rehace
# ANTES de empezar.
#
# Los ficheros de raiz dejan de ser de nadie: cada uno se asigna a la rebanada
# que MUEVE EL NUMERO que ese fichero congela. Un cardinal con igualdad exacta
# es parte de la funcionalidad que lo mueve, no un fichero de infraestructura.
rebanada_1="adaptadores/catalogo/cadenas/ superficies/calendario/ superficies/escalado/ docs/hallazgos-d11.md"
rebanada_2="paquetes/ nucleo/corpus/ puente_piloto_test.go entrevista_alcanza_al_motor_test.go docs/hallazgos-puente.md docs/censo-relojes.md"
rebanada_3="superficies/pantallas/ superficies/serve/ cmd/plazum/ docs/hallazgos-entrevista.md adaptadores/catalogo/cadenas/"
rebanada_4=".github/workflows/release.yml distribucion_test.go docs/lanzamiento/ docs/hallazgos-release.md"

# EL UNICO SOLAPE DECLARADO, Y SE RESUELVE EN EL TIEMPO, NO EN EL ESPACIO.
#
# `adaptadores/catalogo/cadenas/` sale en la 1 y en la 3. No es un descuido: es
# el limite de esta matriz, y se dice en vez de disimularlo.
#
# La regla del tramo 1 («el catalogo va con quien tenga las pantallas») no
# alcanza aqui porque las dos rebanadas tienen pantalla: la 1 pinta la nota del
# calendario y los cubos del escalado, la 3 pinta los campos de valor de la
# entrevista. Y las dos necesitan clave propia, porque la puerta del catalogo
# cruza en las dos direcciones: una clave que nadie pide es tan roja como una
# que falta, asi que ninguna de las dos puede entregar sus claves para que las
# ponga otro.
#
# Partir el fichero por espacio de nombres tampoco vale: es UN es.json y UN
# en.json, nombrados uno a uno en el go:embed, con un test que exige que el
# directorio tenga exactamente esos dos.
#
# Asi que se serializa: LA REBANADA 1 ENTRA EN main ANTES DE QUE ARRANQUE LA 3,
# y la 3 nace rebasada sobre ella. Es barato porque la 1 es un dia y es la
# primera del tramo por decision de producto (la nota es lo mas barato que deja
# de absolver en falso). Cuesta runway a la 3, y se dice: la 3 tiene dos dias,
# no tres.
#
# ESTO ES LO UNICO QUE ESTE SCRIPT NO PUEDE COMPROBAR POR MI. `--cruce` mira
# conjuntos de ficheros, no calendarios. Se verifica a mano de una sola forma:
# `--cruce` con las cuatro ramas vivas tiene que salir limpio, y si saca este
# fichero es que la 3 arranco antes de tiempo.
#
# LO QUE NO ES DE NADIE, y por que:
#
#   ETAPAS.md, README.md       las casillas y los numeros publicados los mueve
#     docs/marcador.md         quien integra, cuando el trabajo ya esta dentro.
#   conservacion_calendario_test.go  censo topado de TODO el corpus. Ninguna
#                              rebanada de este tramo anade obligaciones, asi
#                              que no deberia moverse; si alguna lo pone rojo,
#                              se dice en el informe y NO se toca.
#   comprobar.sh, ci.yml       el lazo local y las puertas. Cambiarlos a mitad
#     .github/puerta.sh        de campana caduca lo que ya se valido.
#   CLAUDE.md, este fichero    los escribe el integrador, y solo el.

# LAS REBANADAS SE NOMBRAN POR NUMERO Y POR LO QUE HACEN. El nombre largo no es
# adorno: `A` no dice nada y `nota` dice de que responde quien la tiene, que es
# lo que hace que un fichero fuera de columna se vea raro al leerlo.
columnas_de() {
  case "$1" in
    1|nota)      echo "$rebanada_1" ;;
    2|puente)    echo "$rebanada_2" ;;
    3|valores)   echo "$rebanada_3" ;;
    4|release)   echo "$rebanada_4" ;;
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
  echo "rebanada desconocida: $frente (son 1|nota, 2|puente, 3|valores, 4|release)" >&2
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
