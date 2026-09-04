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
MANTENER=no

while [ $# -gt 0 ]; do
  case "$1" in
    --binario)        BINARIO="${2:-}"; shift 2 ;;
    --desde-release)  ETIQUETA="${2:-}"; shift 2 ;;
    --mantener)       MANTENER=si; shift ;;
    *) echo "opcion desconocida: $1" >&2
       echo "uso: $0 --binario <ruta> | --desde-release <vX.Y.Z> [--mantener]" >&2
       exit 2 ;;
  esac
done

if [ -z "$BINARIO" ] && [ -z "$ETIQUETA" ]; then
  echo "hace falta --binario <ruta> o --desde-release <vX.Y.Z>." >&2
  echo "  Sin ninguno de los dos no hay nada que probar, y un guion que no prueba" >&2
  echo "  nada y sale con 0 es el verde mas falso que hay." >&2
  exit 2
fi

# ---------------------------------------------------------------------------
# El contador. Mismo patron que .github/puerta.sh: se cuenta lo ejecutado y se
# exige un minimo, porque un guion que se salta la mitad de los pasos y sale con
# 0 no se distingue de uno que los paso todos.
# ---------------------------------------------------------------------------
PASOS_MINIMOS=8
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

cd "$TALLER" || exit 2
echo "en el taller solo esta el binario:"
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
# 3. El corpus, sin red y sin repositorio
# ---------------------------------------------------------------------------
# Esto es lo que hace que la prueba se pueda pasar: el corpus de demostracion va
# DENTRO del binario. Si algun dia dejara de ir, este paso es el que lo dice.
paso "plazum demo instala un corpus sin red y sin repositorio" 0 "$PLAZUM" demo

paso "el corpus instalado esta donde demo dijo" 0 \
  test -f plazum-demo/paquetes/demo-empresa/paquete.json

# ---------------------------------------------------------------------------
# 4. La maquina, revisada por el propio producto
# ---------------------------------------------------------------------------
paso "doctor da el parte de esta maquina" 0 \
  "$PLAZUM" doctor --corpus plazum-demo/paquetes

# ---------------------------------------------------------------------------
# 5. EL CALENDARIO. Es el criterio entero.
# ---------------------------------------------------------------------------
paso "EL CALENDARIO, con el corpus y el alcance que instalo demo" 0 \
  "$PLAZUM" calendario \
    --corpus plazum-demo/paquetes \
    --alcance plazum-demo/paquetes/demo-empresa/alcance.json

# Y que lo que salio es un calendario y no un mensaje amable con rc=0.
salida_cal=$("$PLAZUM" calendario --corpus plazum-demo/paquetes \
  --alcance plazum-demo/paquetes/demo-empresa/alcance.json 2>&1)
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
# 6. Que se pueda deshacer. Un producto que no sabe irse no se prueba dos veces.
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
