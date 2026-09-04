#!/usr/bin/env bash
# ESPERAR A QUE CI TERMINE, SIN QUE EL CONJUNTO VACIO SE LEA COMO EXITO.
#
# # Por que existe, y no es comodidad
#
# El 04-09-2026 el integrador escribio a mano un bucle de espera de CI y le dio
# verde: «pendientes 0, FALLOS: []». Habia CERO ejecuciones, porque GitHub
# todavia no las habia registrado, y un conjunto vacio no tiene pendientes ni
# fallos. Es EXACTAMENTE el verde vacio que este repositorio lleva meses
# vigilando en `puerta.sh` (go test sale 0 cuando el patron no casa) y en
# `frontera.sh` (cero ficheros fuera de columna no es una frontera respetada).
#
# La leccion no es «tener mas cuidado»: la disciplina estaba en el producto y no
# en las manos, y el fallo salio por donde no habia puerta. Asi que el comando
# suelto pasa a ser un script con su guarda, como todo lo demas.
#
# # Las cinco formas de fallar que este script NO confunde con exito
#
#  1. CI no ha arrancado todavia        menos ejecuciones que el minimo: espera
#  2. CI arranco a medias               idem
#  3. una ejecucion fallo               se dice cual, salida 1
#  4. el SHA no esta en el remoto       se para ANTES de esperar, salida 2
#  5. la propia medida se rompe         se para tras varios intentos, salida 2
#
# LA QUINTA ES LA QUE ESTE SCRIPT SE HIZO A SI MISMO en su primer borrador: la
# consulta fallaba, el `case` caia en el comodin y dormia, y el script se
# quedaba esperando para siempre a que una medida rota diera un resultado. Una
# medida que no se puede tomar NO es «todavia no»: es un error, y se dice.
#
# Uso:
#   .github/esperar-ci.sh <sha> [minimo-de-ejecuciones]
#
# El minimo por defecto es 9, que son los workflows que dispara hoy un push a
# main. Se pasa a mano cuando se sabe que van a ser otros.
set -uo pipefail

cd "$(dirname "$0")/.." || exit 1

sha="${1:-}"
minimo="${2:-9}"

if [ -z "$sha" ]; then
  echo "uso: .github/esperar-ci.sh <sha> [minimo-de-ejecuciones]" >&2
  exit 2
fi

# EL SHA TIENE QUE EXISTIR Y ESTAR EN EL REMOTO, comprobado ANTES de esperar
# nada. Esperar ejecuciones de un commit que no se ha empujado es esperar para
# siempre y despues concluir que no hay fallos.
if ! git cat-file -e "${sha}^{commit}" 2>/dev/null; then
  echo "ese SHA no existe ni en local: $sha" >&2
  exit 2
fi
if ! git branch -r --contains "$sha" 2>/dev/null | grep -q .; then
  echo "PARADA: $sha no esta en ninguna rama del remoto." >&2
  echo "  CI no va a ejecutar nada sobre el, asi que esperar daria «cero fallos»" >&2
  echo "  sobre cero ejecuciones, que es el verde vacio. Empuja primero." >&2
  exit 2
fi

corto="$(git rev-parse --short=7 "$sha")"
export CORTO="$corto" MINIMO="$minimo"
echo "esperando CI sobre $corto (minimo $minimo ejecuciones)..."

rotos=0
while :; do
  linea="$(gh run list --limit 30 --json workflowName,status,conclusion,headSha 2>/dev/null |
    python -c '
import json,sys,os
corto=os.environ["CORTO"]; minimo=int(os.environ["MINIMO"])
todas=json.load(sys.stdin)
rs=[r for r in todas if r["headSha"].startswith(corto)]
pend=[r["workflowName"] for r in rs if r["status"]!="completed"]
fall=[r["workflowName"] for r in rs
      if r["status"]=="completed" and r["conclusion"] not in ("success","skipped")]
# EL MINIMO ES LA GUARDA ENTERA. Menos ejecuciones que el minimo NO es verde:
# es que CI no ha arrancado, y decir «cero fallos» ahi es el verde vacio.
if len(rs) < minimo:
    estado="ARRANCANDO"
elif pend:
    estado="CORRIENDO"
elif fall:
    estado="ROJO"
else:
    estado="VERDE"
print("%s|%d|%s|%s" % (estado, len(rs), ",".join(pend), ",".join(fall)))
' 2>/dev/null)"

  if [ -z "$linea" ]; then
    # LA MEDIDA SE HA ROTO. No es «todavia no»: es que no se puede saber.
    rotos=$((rotos + 1))
    if [ "$rotos" -ge 5 ]; then
      echo "PARADA: la consulta a GitHub ha fallado 5 veces seguidas." >&2
      echo "  No se sabe en que estado esta CI, y NO SABERLO NO ES VERDE." >&2
      echo "  Mira: gh auth status, y la red, antes de dar nada por bueno." >&2
      exit 2
    fi
    sleep 10
    continue
  fi
  rotos=0

  estado="${linea%%|*}"
  resto="${linea#*|}"; n="${resto%%|*}"
  resto="${resto#*|}"; fallos="${resto#*|}"

  case "$estado" in
    VERDE)
      echo "CI EN VERDE sobre $corto: $n ejecuciones, ninguna fallida."
      exit 0
      ;;
    ROJO)
      echo "CI EN ROJO sobre $corto: $n ejecuciones."
      echo "  fallan: $fallos"
      echo "  Un rojo NO se reintenta hasta que salga verde sin mirarlo: si es"
      echo "  intermitente, la que esta mal suele ser la MEDIDA, y aflojar el"
      echo "  umbral para que pase es bajar la afirmacion. gh run view --log-failed"
      exit 1
      ;;
    ARRANCANDO) sleep 10 ;;
    CORRIENDO)  sleep 15 ;;
    *)
      # UN ESTADO QUE NO EXISTE NO SE DUERME. Es la misma familia que el
      # comodin que dormia: lo que no se entiende es un error, nunca una espera.
      echo "PARADA: estado desconocido de la medida: $estado" >&2
      exit 2
      ;;
  esac
done
