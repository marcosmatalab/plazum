#!/usr/bin/env bash
# Genera las tres capturas del lanzamiento, DEL BINARIO DE VERDAD.
#
# POR QUE UN SCRIPT Y NO TRES PEGADOS A MANO. Una captura pegada a mano en un
# post es una afirmacion sobre lo que el producto hace, y las afirmaciones se
# verifican. Con esto, cualquiera reproduce las tres en un comando y compara.
#
# EL INSTANTE VA CABLEADO (--ahora), y es lo que las hace reproducibles: sin el,
# la captura del estreno del CRA deja de tener sentido en cuanto pasa el 11 de
# septiembre de 2026, y nadie sabria si el post mentia o si el mundo avanzo.
#
# Uso:
#   ./docs/lanzamiento/generar.sh
set -uo pipefail
cd "$(dirname "$0")/../.." || exit 1

AHORA="2026-09-01T09:00:00Z"
SALIDA="docs/lanzamiento"

# La cabecera que hace auditable cada captura: de que commit sale y con que
# instante. Sin esto, una captura vieja es indistinguible de una actual.
cabecera() {
  echo "\$ plazum $*"
  echo "#"
  echo "# generado por docs/lanzamiento/generar.sh"
  echo "# commit:   $(git rev-parse --short HEAD)"
  echo "# instante: $AHORA (cableado, para que la captura sea reproducible)"
  echo "#"
  echo
}

# 1. EL ESTRENO DEL CRA. Es la fila que ningun catalogo de controles imprime:
#    una obligacion que TODAVIA no obliga y empieza el 11 de septiembre de 2026.
{
  cabecera "calendario --pais=ES --sector=fabricante-software --empleados=200"
  go run ./cmd/plazum calendario --ahora="$AHORA" --corpus=paquetes \
    --pais=ES --sector=fabricante-software --empleados=200
} > "$SALIDA/1-estreno-del-cra.txt" 2>&1

# 2. LAS SENTADAS. La composicion entre marcos, computada: dos obligaciones de
#    marcos distintos que se despachan en la misma sesion.
{
  cabecera "calendario --pais=ES --sector=servicios-digitales --empleados=200 --sentadas"
  go run ./cmd/plazum calendario --ahora="$AHORA" --corpus=paquetes \
    --pais=ES --sector=servicios-digitales --empleados=200 --sentadas
} > "$SALIDA/2-las-sentadas.txt" 2>&1

# 3. LO VENCIDO Y LA CUENTA ENTERA, con su descargo. Hace falta un alcance con
#    una fecha vieja: un perfil de arranque no puede tener incumplimientos,
#    porque sus fechas son de ejemplo y se siembran recientes.
ALCANCE=$(mktemp)
cat > "$ALCANCE" <<'JSON'
{
  "organizacion": "Ejemplo, S.L.",
  "sujeto": "ejemplo",
  "hechos": [{"pred": "papel_nis2_tecnica", "args": ["ejemplo", "entidad_pertinente"]}],
  "fechas": {
    "ultima_revision_de_roles_y_responsabilidades": "2025-01-15",
    "ultima_revision_de_la_politica": "2026-03-01"
  }
}
JSON
{
  cabecera "calendario --alcance alcance.json"
  echo "# el alcance de esta captura:"
  sed 's/^/#   /' "$ALCANCE"
  echo
  go run ./cmd/plazum calendario --ahora="$AHORA" --corpus=paquetes --alcance="$ALCANCE"
} > "$SALIDA/3-lo-vencido-y-la-cuenta.txt" 2>&1
rm -f "$ALCANCE"

echo "tres capturas en $SALIDA:"
wc -l "$SALIDA"/*.txt
