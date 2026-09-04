#!/usr/bin/env bash
# LA PRUEBA DE LA MAQUINA LIMPIA.
#
# EL CRITERIO DE EXITO NO ES QUE EL WORKFLOW SALGA VERDE. Un workflow verde dice
# que se construyo un fichero; no dice que ese fichero sirva para nada en manos
# de otro. El criterio es este, y solo este:
#
#     coger el binario PUBLICADO, en una maquina que no tiene el repositorio,
#     y llegar al calendario.
#
# «Sin tocar el repositorio» es literal: sin clonar, sin `go build`, sin copiar
# `paquetes/` al lado. Todo lo que haga falta, o viene dentro del binario o el
# binario dice donde conseguirlo. Si en algun paso hay que volver al repositorio,
# la prueba ha fallado aunque el comando salga con 0, porque el comprador no
# tiene el repositorio: tiene un fichero que se ha bajado de una pagina.
#
# POR QUE ES UN GUION Y NO UNA LISTA EN UN DOCUMENTO. Una lista de pasos en
# prosa se «ejecuta» leyendola y asintiendo, que es como se dan por buenas las
# cosas que no se han probado. Esto se ejecuta y sale con 0 o con 1.
#
# Uso:
#   docs/lanzamiento/maquina-limpia.sh --binario /ruta/a/plazum
#   docs/lanzamiento/maquina-limpia.sh --desde-release v0.1.0-rc1
#
# El segundo modo se baja el artefacto de la release con `gh` y comprueba su
# suma y su firma antes de ejecutarlo. Es el que vale como prueba de verdad: el
# primero sirve para ensayar el guion antes de que exista ninguna release.
#
# NO SE EJECUTA DENTRO DEL REPOSITORIO, y se niega a hacerlo. Correrlo ahi
# dentro daria verde por el peor motivo posible: que `paquetes/` estaba al lado
# sin que nadie lo pidiera, o sea midiendo justo lo contrario de lo que se
# quiere medir.

set -uo pipefail

# -e APAGADO a proposito, igual que en .github/puerta.sh y por lo mismo: aqui se
# ejecutan comandos que TIENEN que fallar (un `calendario` sin corpus sale con 1
# y eso es un acierto, no un fallo). Con -e, el primero de esos mata el guion
# antes de imprimir el motivo. El codigo de salida lo pone `cerrar` al final.

BINARIO=""
ETIQUETA=""
CORPUS_TAR=""
MANTENER=no

while [ $# -gt 0 ]; do
  case "$1" in
    --binario)        BINARIO="${2:-}"; shift 2 ;;
    --corpus)         CORPUS_TAR="${2:-}"; shift 2 ;;
    --desde-release)  ETIQUETA="${2:-}"; shift 2 ;;
    --mantener)       MANTENER=si; shift ;;
    *) echo "opcion desconocida: $1" >&2
       echo "uso: $0 --binario <ruta> --corpus <fichero.tar.gz>" >&2
       echo "     $0 --desde-release <vX.Y.Z> [--mantener]" >&2
       exit 2 ;;
  esac
done

if [ -z "$BINARIO" ] && [ -z "$ETIQUETA" ]; then
  echo "hace falta --binario <ruta> o --desde-release <vX.Y.Z>." >&2
  echo "  Sin ninguno de los dos no hay nada que probar, y un guion que no prueba" >&2
  echo "  nada y sale con 0 es el verde mas falso que hay." >&2
  exit 2
fi

# EL ENSAYO SIN CORPUS SE RECHAZA EN LA PUERTA, y no es rigidez.
#
# El modo --binario existe para ensayar este guion antes de que haya release. Si
# ademas dejara ensayar SIN corpus, lo que se estaria ensayando es el mundo
# viejo: el binario solo, llegando al calendario con la demo, saliendo con 0. O
# sea justo el verde que este tramo existe para no volver a dar. Se pide el
# fichero, o no se ensaya.
if [ -n "$BINARIO" ] && [ -z "$CORPUS_TAR" ]; then
  echo "hace falta --corpus <fichero.tar.gz> junto a --binario." >&2
  echo "  El criterio de esta prueba es llegar al calendario CON EL CORPUS REAL." >&2
  echo "  Un binario sin corpus llega al calendario igual, con la demostracion, y" >&2
  echo "  sale con 0: ese es exactamente el falso verde que esta prueba dio hasta" >&2
  echo "  el 04-09-2026." >&2
  echo "  Sacalo del repositorio con:" >&2
  echo "    go run ./cmd/plazum corpus --empaquetar paquetes --salida /tmp/plazum-corpus.tar.gz" >&2
  exit 2
fi

# ---------------------------------------------------------------------------
# El contador. Mismo patron que .github/puerta.sh: se cuenta lo ejecutado y se
# exige un minimo, porque un guion que se salta la mitad de los pasos y sale con
# 0 no se distingue de uno que los paso todos.
# ---------------------------------------------------------------------------
# SUBIO DE 8 A 14 EL 04-09-2026, con el corpus real entrando en la prueba. El
# numero se deriva contando los `paso` y los `_corridos` del camino base (sin los
# tres que solo existen en modo --desde-release: suma, firma y presencia del
# activo del corpus). Bajarlo se hace en el mismo commit que recorte pasos y
# diciendo por que, nunca para que un rojo se vuelva verde.
PASOS_MINIMOS=14
_corridos=0
_fallos=0
_saltados=0

# paso NOMBRE RC_ESPERADO COMANDO...
#
# RC_ESPERADO es parte de la afirmacion, no un detalle: la mitad de esta prueba
# consiste en que ciertos comandos fallen BIEN. `calendario` sin corpus tiene que
# salir con 1 y decir como arreglarlo; si saliera con 0 estaria ensenando un
# calendario vacio, que es peor que un error.
paso() {
  local nombre="$1" esperado="$2"
  shift 2
  _corridos=$((_corridos + 1))
  echo "== $nombre"
  echo "   \$ $*"
  local salida rc
  salida=$("$@" 2>&1)
  rc=$?
  echo "$salida" | sed 's/^/   | /'
  echo "   rc=$rc (esperado $esperado)"
  if [ "$rc" != "$esperado" ]; then
    echo "   PASO ROTO: $nombre"
    _fallos=$((_fallos + 1))
    return 1
  fi
  echo
  return 0
}

# saltar NOMBRE MOTIVO
#
# «No se pudo ejecutar» y «no encontro nada» son cosas distintas, y confundirlas
# hace que una maquina sin red se lea como una maquina limpia. Un paso saltado se
# dice, con su motivo, y cuenta aparte.
saltar() {
  echo "== $1"
  echo "   SALTADO: $2"
  echo "   Esto NO es un aprobado: es un paso que no se ha podido ejecutar."
  echo
  _saltados=$((_saltados + 1))
}

# ---------------------------------------------------------------------------
# La guarda: esto no corre dentro del repositorio.
# ---------------------------------------------------------------------------
if [ -d .git ] || [ -f go.mod ] || [ -d paquetes ]; then
  echo "ESTOY DENTRO DEL REPOSITORIO (hay .git, go.mod o paquetes/ aqui)." >&2
  echo "  La prueba de la maquina limpia mide si el binario SOLO llega al" >&2
  echo "  calendario. Corriendola aqui, 'paquetes/' esta al lado sin que nadie" >&2
  echo "  lo pida y el resultado seria verde por el motivo contrario al que" >&2
  echo "  se quiere medir." >&2
  echo "  Arreglo: cd a un directorio vacio fuera del repositorio y vuelve a" >&2
  echo "  llamar a este guion por su ruta absoluta." >&2
  exit 2
fi

TALLER="$(mktemp -d)"
echo "taller (maquina limpia simulada): $TALLER"
echo

limpiar() {
  if [ "$MANTENER" = si ]; then
    echo "taller conservado en $TALLER (--mantener)"
  else
    rm -rf "$TALLER"
  fi
}
trap limpiar EXIT

# ---------------------------------------------------------------------------
# 0. Conseguir el binario
# ---------------------------------------------------------------------------
if [ -n "$ETIQUETA" ]; then
  if ! command -v gh > /dev/null; then
    echo "no hay 'gh' y hace falta para bajarse la release $ETIQUETA." >&2
    echo "  Esto es un 'no se pudo ejecutar', no un 'esta roto'." >&2
    exit 2
  fi
  echo "== bajando los artefactos de $ETIQUETA"
  if ! gh release download "$ETIQUETA" --dir "$TALLER" 2>&1 | sed 's/^/   | /'; then
    echo "   no se ha podido bajar la release $ETIQUETA." >&2
    echo "   Si aun no existe, esto es lo ESPERADO: el guion esta listo y la" >&2
    echo "   etiqueta no. Vuelve cuando exista." >&2
    exit 2
  fi
  echo

  # Nombre del artefacto segun el sistema. Sale del patron del workflow:
  # dist/plazum-${goos}-${arch}${sufijo}
  case "$(uname -s)" in
    Linux)  goos=linux;  sufijo="" ;;
    Darwin) goos=darwin; sufijo="" ;;
    *)      goos=windows; sufijo=".exe" ;;
  esac
  case "$(uname -m)" in
    x86_64|amd64) arch=amd64 ;;
    arm64|aarch64) arch=arm64 ;;
    *) arch=amd64 ;;
  esac
  BINARIO="$TALLER/plazum-${goos}-${arch}${sufijo}"

  # EL CORPUS ES UN ACTIVO DE LA RELEASE Y AQUI SE COMPRUEBA QUE ESTE.
  #
  # `gh release download` sin --pattern se baja TODO, asi que si el corpus
  # existe ya esta en el taller. Que no este no es un descuido de esta prueba:
  # es la release publicandose sin corpus, que es el P0 entero. Se dice como
  # PASO ROTO y no como paso saltado, porque «no se pudo ejecutar» y «no esta»
  # son cosas distintas y solo la segunda es un fallo del producto.
  CORPUS_TAR="$TALLER/plazum-corpus.tar.gz"
  _corridos=$((_corridos + 1))
  echo "== la release trae el corpus real, y no solo binarios"
  if [ -f "$CORPUS_TAR" ]; then
    echo "   ok: $CORPUS_TAR"
    if [ -f "$TALLER/plazum-corpus.huella" ]; then
      echo "   huella publicada: $(cat "$TALLER/plazum-corpus.huella")"
    else
      echo "   AVISO: no viene plazum-corpus.huella. Sin ella, quien tenga un binario"
      echo "   mas viejo que el corpus no puede instalarlo: --huella-esperada se queda"
      echo "   sin fuente y el ancla del binario pasa de comprobacion a muro."
    fi
    echo
  else
    echo "   PASO ROTO: la release $ETIQUETA no trae plazum-corpus.tar.gz."
    echo "   Son binarios sin corpus. Quien se los baje llega al calendario con la"
    echo "   demostracion (un paquete, tres relojes) y se va pensando que plazum no"
    echo "   trae nada. Arreglo: mira el trabajo 'corpus' de release.yml."
    _fallos=$((_fallos + 1))
    CORPUS_TAR=""
  fi

  # LA SUMA, ANTES DE EJECUTAR NADA. Un binario que no cuadra con su
  # SHA256SUMS no se ejecuta: se para aqui.
  if [ -f "$TALLER/SHA256SUMS-${goos}" ]; then
    if command -v sha256sum > /dev/null; then
      esperada=$(grep -F "plazum-${goos}-${arch}${sufijo}" "$TALLER/SHA256SUMS-${goos}" | awk '{print $1}')
      real=$(sha256sum "$BINARIO" | awk '{print $1}')
      paso "la suma del binario cuadra con SHA256SUMS-${goos}" 0 \
        test "$esperada" = "$real"
    else
      saltar "la suma del binario" "no hay sha256sum en esta maquina"
    fi
  else
    saltar "la suma del binario" "la release no trae SHA256SUMS-${goos}"
  fi

  # LA FIRMA. cosign publica el certificado en Rekor al firmar; verificar no
  # publica nada. Si no hay cosign, se dice y se sigue: es un 'no se pudo
  # comprobar', que no es lo mismo que 'la firma esta mal'.
  if command -v cosign > /dev/null && [ -f "$BINARIO.sig" ] && [ -f "$BINARIO.pem" ]; then
    paso "la firma del binario verifica contra su certificado" 0 \
      cosign verify-blob \
        --certificate "$BINARIO.pem" \
        --signature "$BINARIO.sig" \
        --certificate-identity-regexp '.*' \
        --certificate-oidc-issuer-regexp '.*' \
        "$BINARIO"
  else
    saltar "la firma del binario" \
      "no hay cosign instalado, o la release no trae $BINARIO.sig y .pem"
  fi
fi

if [ ! -f "$BINARIO" ]; then
  echo "no encuentro el binario en $BINARIO" >&2
  exit 2
fi

# Se copia al taller y se ejecuta DESDE ALLI, con nombre `plazum`: si se
# ejecutara desde su sitio original, el directorio de trabajo podria tener al
# lado cosas que en una maquina limpia no estan.
cp "$BINARIO" "$TALLER/plazum" 2>/dev/null || cp "$BINARIO" "$TALLER/plazum.exe"
PLAZUM="$TALLER/plazum"
[ -f "$PLAZUM" ] || PLAZUM="$TALLER/plazum.exe"
chmod +x "$PLAZUM" 2>/dev/null

# En el modo de ensayo el corpus lo trae el operador; en el de release ya esta
# en el taller porque `gh release download` se lo bajo con todo lo demas.
if [ -n "$CORPUS_TAR" ] && [ "$(dirname "$CORPUS_TAR")" != "$TALLER" ]; then
  if [ ! -f "$CORPUS_TAR" ]; then
    echo "no encuentro el corpus en $CORPUS_TAR" >&2
    exit 2
  fi
  cp "$CORPUS_TAR" "$TALLER/plazum-corpus.tar.gz"
  CORPUS_TAR="$TALLER/plazum-corpus.tar.gz"
fi

cd "$TALLER" || exit 2
echo "en el taller solo esta lo que se baja el comprador:"
ls -A | sed 's/^/   /'
echo

# ---------------------------------------------------------------------------
# 1. El primer minuto: que ve quien acaba de bajarse esto
# ---------------------------------------------------------------------------
# rc=2 es lo correcto para `plazum` a secas: imprime el menu y no hace nada. Un
# 0 aqui diria que ejecutar plazum sin argumentos es una operacion valida.
paso "plazum a secas imprime por donde empezar" 2 "$PLAZUM"

# ---------------------------------------------------------------------------
# 2. La pared, que tiene que ser una pared con instrucciones
# ---------------------------------------------------------------------------
# En una maquina limpia NO hay corpus: `calendario` tiene que fallar, y tiene que
# fallar diciendo que hacer. Un calendario vacio con rc=0 seria mucho peor.
paso "calendario sin corpus falla, y dice como arreglarlo" 1 "$PLAZUM" calendario

# ---------------------------------------------------------------------------
# 3. EL CORPUS REAL, comprobado antes de tocar el disco
#
# Este es el paso que el tramo 3 anadio y es el que convierte una descarga en el
# producto. `--instalar` compara la huella del .tar.gz contra la que el binario
# lleva dentro ANTES de dejarlo caer en su sitio: si no cuadra, no instala nada.
#
# Que salga con 0 aqui prueba las dos mitades a la vez: que el corpus publicado
# es el que este binario espera, y que el ancla entro de verdad en el binario. Un
# `-ldflags -X` que no hubiera entrado (porque alguien renombro el simbolo, y el
# enlazador no se queja de eso) daria un binario sin ancla, y sin ancla este
# comando se niega.
# ---------------------------------------------------------------------------
paso "el binario dice que corpus espera, antes de tener ninguno" 0 "$PLAZUM" corpus

paso "EL CORPUS REAL se instala, comprobado contra la huella del binario" 0 \
  "$PLAZUM" corpus --instalar plazum-corpus.tar.gz

paso "el corpus instalado cuadra con el ancla del binario" 0 \
  "$PLAZUM" corpus --verificar paquetes

# LA VUELTA DE LA COMPROBACION, y sin ella lo de arriba no demuestra nada. Un
# --instalar que dijera que si a todo pasaria los tres pasos anteriores. Se le da
# un fichero que NO es el corpus publicado y tiene que negarse.
_corridos=$((_corridos + 1))
echo "== un corpus que no es el publicado NO entra"
printf 'esto no es un corpus' > falso.tar.gz
if "$PLAZUM" corpus --instalar falso.tar.gz --destino paquetes-falso > falso.txt 2>&1; then
  echo "   PASO ROTO: ha instalado un fichero que no es el corpus publicado."
  echo "   Un corpus que entra sin comprobar es peor que no tener corpus: son"
  echo "   fechas legales en las que alguien va a confiar."
  sed 's/^/   | /' falso.txt
  _fallos=$((_fallos + 1))
elif [ -d paquetes-falso ]; then
  echo "   PASO ROTO: se nego y aun asi ha dejado el destino puesto."
  _fallos=$((_fallos + 1))
else
  echo "   ok: se niega y no deja nada. Motivo que da:"
  sed 's/^/   | /' falso.txt
fi
rm -f falso.tar.gz falso.txt
echo

# El corpus con el que se contesta el calendario, y el alcance. Van en variables
# porque el paso 5.bis los vuelve a usar para contar, y dos copias de la ruta son
# dos copias que se separan.
CORPUS_USADO=paquetes
ALCANCE_USADO="--pais=ES --sector=fabricante-software --empleados=200"

# ---------------------------------------------------------------------------
# 4. La maquina, revisada por el propio producto
# ---------------------------------------------------------------------------
paso "doctor da el parte de esta maquina" 0 \
  "$PLAZUM" doctor --corpus "${CORPUS_USADO}"

# ---------------------------------------------------------------------------
# 5. EL CALENDARIO. Es el criterio entero.
#
# Y AHORA SE LLEGA CON EL CORPUS REAL. Se usa el perfil de arranque
# (--pais/--sector/--empleados) y no un alcance, porque una maquina limpia no
# tiene respuestas de nadie: es exactamente lo que hace quien acaba de bajarse
# esto. Cada fila sale marcada como [supuesto], que es lo correcto.
# ---------------------------------------------------------------------------
# shellcheck disable=SC2086 -- ALCANCE_USADO son tres banderas, no una ruta
paso "EL CALENDARIO, con el corpus real recien instalado" 0 \
  "$PLAZUM" calendario --corpus "${CORPUS_USADO}" ${ALCANCE_USADO}

# Y que lo que salio es un calendario y no un mensaje amable con rc=0.
salida_cal=$("$PLAZUM" calendario --corpus "${CORPUS_USADO}" ${ALCANCE_USADO} 2>&1)
_corridos=$((_corridos + 1))
echo "== el calendario trae fechas de verdad, no una pantalla vacia"
if printf '%s' "$salida_cal" | grep -qF "PROXIMOS DOCE MESES"; then
  echo "   ok: la cabecera del calendario esta"
  echo
else
  echo "   PASO ROTO: la salida tiene rc=0 y no parece un calendario."
  echo "   Un rc=0 con la pantalla vacia es exactamente el verde que este"
  echo "   guion existe para no dar."
  printf '%s' "$salida_cal" | sed 's/^/   | /'
  _fallos=$((_fallos + 1))
fi


# ---------------------------------------------------------------------------
# 5.bis. CON QUE CORPUS SE HA LLEGADO AL CALENDARIO. Es la mitad que faltaba.
#
# LA GUARDA DE ESTE PASO SALTO, Y ESO ES LA SENAL DE QUE FUNCIONO.
#
# Aqui vivia un aviso que decia que se habia llegado al calendario con el corpus
# de DEMOSTRACION (un paquete, tres relojes) porque los treinta marcos vivian en
# el repositorio y no viajaban en la release. Ese aviso llevaba dentro una guarda
# que lo mataba si algun dia aparecian mas de cinco paquetes, para que el dia que
# el corpus SI viajara el aviso no se quedara mintiendo al reves.
#
# Ese dia fue el 04-09-2026: el corpus real se publica ahora como activo firmado
# de la release, la guarda salto con 33 paquetes, y este texto es el mundo nuevo.
# Un aviso que se rompe solo cuando caduca vale mas que uno correcto que nadie
# vuelve a leer.
#
# LO QUE SE MIDE AHORA, Y SON DOS NUMEROS PORQUE NINGUNO SOLO DICE LA VERDAD:
#
#   paquetes  cuantos marcos han llegado. Un corpus con un paquete es la demo.
#   relojes   cuantas obligaciones con fecha trae. Un corpus con 33 paquetes
#             vacios contaria 33 y no serviria para nada, asi que el numero de
#             paquetes solo no basta.
#
# Y LOS DOS SE IMPRIMEN. Antes, `_marcos` se calculaba en este mismo sitio y NO
# SE IMPRIMIA NUNCA: la variable se asignaba y se tiraba. La mitad contable de la
# afirmacion estaba escrita y muerta, y la transcripcion se leia como «el
# producto funciona entero» sin que ninguna linea lo dijera. Una medida que se
# toma y no se saca por pantalla es peor que no tomarla, porque el guion parece
# que lo comprueba.
# ---------------------------------------------------------------------------
_corridos=$((_corridos + 1))
echo "== con QUE corpus se ha llegado al calendario"

_paquetes=$(find "${CORPUS_USADO}" -name paquete.json 2>/dev/null | wc -l | tr -d ' ')
# Los relojes se cuentan sobre el calendario SIN filtrar por aplicabilidad: es la
# medida del corpus instalado, no la de lo que le toca a un perfil concreto.
_relojes=$("$PLAZUM" calendario --corpus "${CORPUS_USADO}" --todos-los-relojes \
  ${ALCANCE_USADO} 2>/dev/null | grep -cE '^[[:space:]]*urn:' || true)

echo "   corpus usado          ${CORPUS_USADO}"
echo "   paquetes instalados   ${_paquetes}"
echo "   relojes en el corpus  ${_relojes}"
echo

# LA GUARDA NUEVA, Y APUNTA AL REVES QUE LA VIEJA. Antes lo sospechoso era tener
# MUCHOS paquetes (queria decir que el aviso habia caducado). Ahora lo sospechoso
# es tener POCOS: significa que la release ha vuelto a publicarse sin el corpus
# real y que esta prueba esta dando verde sobre la demo otra vez, que es
# exactamente el P0 que se cerro.
#
# Los minimos son cardinales y se derivan de lo que hay hoy (33 paquetes, 222
# relojes), con holgura hacia abajo para que un paquete que se reorganice no
# ponga rojo esto, y nunca tanta como para que quepa la demo.
_MIN_PAQUETES=30
_MIN_RELOJES=150
if [ "${_paquetes}" -lt "${_MIN_PAQUETES}" ] || [ "${_relojes}" -lt "${_MIN_RELOJES}" ]; then
  echo "   PASO ROTO: se ha llegado al calendario con ${_paquetes} paquetes y"
  echo "   ${_relojes} relojes, y hacen falta al menos ${_MIN_PAQUETES} y ${_MIN_RELOJES}."
  echo "   Esto es el corpus de DEMOSTRACION, no el real. Quien se baje este"
  echo "   binario veria una demo vacia y se iria pensando que plazum no trae"
  echo "   nada, que es la unica primera impresion que no se repite."
  echo "   Arreglo: mira el trabajo 'corpus' de la ejecucion de release.yml y"
  echo "   comprueba que plazum-corpus.tar.gz esta entre los activos publicados."
  _fallos=$((_fallos + 1))
else
  echo "   Se ha llegado al calendario con el corpus REAL, no con el de"
  echo "   demostracion, y comprobado contra la huella que el binario lleva"
  echo "   dentro. Esto es lo que ve quien se baja la release."
fi
echo

# ---------------------------------------------------------------------------
# 6. El paseo de dos minutos SIGUE EXISTIENDO, y ahora es lo que dice ser.
#
# `plazum demo` lleva su corpus DENTRO del binario y no necesita ni red ni el
# activo de la release. Eso no ha cambiado y no tenia que cambiar: lo que
# cambia es que ya no es lo unico. Antes era el unico camino al calendario y
# por eso la prueba entera se leia como si el producto fuera eso.
# ---------------------------------------------------------------------------
paso "plazum demo sigue funcionando sin red y sin el activo de la release" 0 "$PLAZUM" demo

paso "el corpus del demo esta donde demo dijo" 0 \
  test -f plazum-demo/paquetes/demo-empresa/paquete.json

# ---------------------------------------------------------------------------
# 7. Que se pueda deshacer. Un producto que no sabe irse no se prueba dos veces.
# ---------------------------------------------------------------------------
paso "demo --deshacer no deja nada en la maquina" 0 "$PLAZUM" demo --deshacer
paso "y de verdad no queda nada" 1 test -d plazum-demo

# ---------------------------------------------------------------------------
# El cierre
# ---------------------------------------------------------------------------
echo "---------------------------------------------------------------"
echo "pasos ejecutados: $_corridos   rotos: $_fallos   saltados: $_saltados"

if [ "$_corridos" -lt "$PASOS_MINIMOS" ]; then
  echo "PRUEBA ROTA: solo se han ejecutado $_corridos pasos y hacen falta $PASOS_MINIMOS."
  echo "  Un guion que se salta la mitad y sale con 0 no se distingue de uno que"
  echo "  paso todo. Si el recorte es intencionado, baja PASOS_MINIMOS en el mismo"
  echo "  commit y di por que."
  exit 1
fi
if [ "$_fallos" -gt 0 ]; then
  echo "PRUEBA ROTA: $_fallos de $_corridos pasos."
  echo "  El binario publicado NO llega al calendario en una maquina limpia."
  exit 1
fi
if [ "$_saltados" -gt 0 ]; then
  echo "LA PRUEBA PASA, con $_saltados paso(s) SIN COMPROBAR (ver arriba el motivo"
  echo "de cada uno). Eso no es lo mismo que comprobados: si los saltados son la"
  echo "suma y la firma, lo que se ha probado es que el binario funciona, no que"
  echo "sea el que se publico."
  exit 0
fi
echo "LA PRUEBA PASA ENTERA: el binario publicado llega al calendario en una"
echo "maquina sin repositorio, y se va sin dejar nada."
exit 0
