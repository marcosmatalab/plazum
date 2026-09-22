#!/usr/bin/env bash
# gif.sh: regenera docs/demo.gif desde docs/demo.tape, en un contenedor.
#
# POR QUE ESTE SCRIPT EXISTE Y NO BASTA `vhs docs/demo.tape`
#
# Por tres cosas medidas el 22-09-2026, y las tres dan un fallo SILENCIOSO, que
# es lo que hace que merezca la pena escribirlas en vez de recordarlas:
#
#   1. `vhs` necesita `ttyd`, que no existe en Windows. Y su contenedor oficial
#      (`ghcr.io/charmbracelet/vhs`, `charmcli/vhs`) contesta `denied` desde una
#      maquina sin credenciales de registro. Por eso la imagen se construye aqui
#      desde `vhs.Dockerfile`, con ttyd, ffmpeg y chromium dentro.
#
#   2. `vhs` CAPTURA bien y NO CODIFICA: sale con codigo 0, imprime «Creating
#      demo.gif...» y no escribe ningun fichero ni llega a invocar a ffmpeg
#      (comprobado sustituyendo ffmpeg por un envoltorio que registra sus
#      argumentos: no se le llamo ni una vez). Asi que el `.tape` pide
#      `Output frames/` y la codificacion la hace este script, que es lo que
#      `vhs` haria.
#
#   3. El montaje de Windows NO admite las 888 escrituras de los frames: con el
#      directorio de trabajo sobre el bind mount no se escribe ni un PNG, y
#      tampoco se dice. Por eso se copia el arbol DENTRO del contenedor y solo
#      sale un fichero, el GIF terminado.
#
# LA COMPROBACION QUE NO SE PUEDE SALTAR, y esta arriba porque ya mordio: la
# primera version de este GIF ensenaba un ERROR («el corpus de paquetes no
# carga»), porque dentro del contenedor faltaba `paquetes/`. El guion salio con
# codigo 0, el GIF se genero, pesaba lo esperado y era inservible. Un GIF no se
# publica sin MIRAR un fotograma: por eso este script extrae uno al final y dice
# donde esta.
#
# Uso:
#   docs/lanzamiento/gif.sh
set -uo pipefail

cd "$(dirname "$0")/../.." || exit 1
raiz="$PWD"

echo "== 1. el binario de Linux, el mismo que mide la puerta del presupuesto"
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
  go build -ldflags="-s -w" -trimpath -o docs/lanzamiento/plazum-linux ./cmd/plazum || exit 1

echo "== 2. la imagen con ttyd, ffmpeg y chromium"
docker build -q -t plazum-vhs -f docs/lanzamiento/vhs.Dockerfile docs/lanzamiento || exit 1

echo "== 3. grabar y codificar, todo dentro del contenedor"
salida="$(mktemp -d)"
MSYS_NO_PATHCONV=1 docker run --rm --shm-size=1g \
  -v "$raiz":/repo:ro -v "$salida":/salida \
  --entrypoint bash plazum-vhs -c '
set -e
mkdir -p /vhs
# paquetes/ ENTERO, que es lo que faltaba la primera vez y convirtio el GIF en
# una captura de un mensaje de error.
cp -r /repo/docs /repo/paquetes /repo/expediente-demo.json /repo/contexto-demo.json /vhs/
chmod +x /vhs/docs/lanzamiento/plazum-linux
cd /vhs

# Y ANTES DE GRABAR, QUE LA ORDEN FUNCIONE AQUI. Grabar primero y mirar despues
# es como salio el GIF del error.
if ! docs/lanzamiento/plazum-linux calendario --pais=ES --sector=servicios-digitales --empleados=200 \
     > /salida/ensayo.txt 2>&1; then
  echo "PARADA: la orden que se va a grabar falla dentro del contenedor." >&2
  tail -5 /salida/ensayo.txt >&2
  exit 1
fi

vhs docs/demo.tape > /salida/vhs.log 2>&1 || true
n=$(ls frames/frame-text-*.png 2>/dev/null | wc -l)
if [ "$n" -lt 100 ]; then
  echo "PARADA: solo $n fotogramas capturados. vhs no ha grabado nada util." >&2
  tail -20 /salida/vhs.log >&2
  exit 1
fi
echo "   $n fotogramas"

# La superposicion texto + cursor es lo que hace vhs: sin ella no se ve escribir.
ffmpeg -y -framerate 50 -i "frames/frame-text-%05d.png" \
           -framerate 50 -i "frames/frame-cursor-%05d.png" \
  -filter_complex "[0][1]overlay,fps=16,scale=1100:-1:flags=lanczos,split[s0][s1];[s0]palettegen=stats_mode=diff[p];[s1][p]paletteuse=dither=bayer:bayer_scale=3" \
  -loop 0 /salida/demo.gif 2>/dev/null
# Un fotograma para mirar, que es el paso que no se salta.
ffmpeg -y -i /salida/demo.gif -vf "select=eq(n\,100)" -vframes 1 /salida/fotograma.png 2>/dev/null
' || { echo "el contenedor ha fallado"; rm -rf "$salida"; exit 1; }

if [ ! -s "$salida/demo.gif" ]; then
  echo "PARADA: no ha salido ningun GIF." >&2
  rm -rf "$salida"
  exit 1
fi

cp "$salida/demo.gif" docs/demo.gif
cp "$salida/fotograma.png" "$salida/para-mirar.png" 2>/dev/null

bytes=$(wc -c < docs/demo.gif)
echo
echo "docs/demo.gif: $bytes bytes"
echo "MIRA UN FOTOGRAMA ANTES DE PUBLICARLO: $salida/fotograma.png"
echo "  La primera version de este GIF ensenaba un mensaje de error y pesaba lo"
echo "  esperado. El tamano no dice si el contenido sirve."

# El binario es un artefacto de esta grabacion, no del repositorio.
rm -f docs/lanzamiento/plazum-linux
