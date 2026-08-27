#!/usr/bin/env bash
# comprobar.sh: el UNICO objetivo local. Todo resultado de test que se cite en un
# informe sale de aqui.
#
# POR QUE EXISTE. `go test` sale con CODIGO 0 en dos situaciones donde no ha
# comprobado nada (patron -run que no casa, glob de paquetes sin tests), y en las
# dos el verde es indistinguible de un verde de verdad. Contra eso se escribio
# `.github/puerta.sh`, que cuenta los casos EJECUTADOS y exige un minimo, y
# `puertas_test.go`, que prohibe a los workflows invocar `go test` a pelo.
#
# Y aun asi la trampa mordio otra vez, la tercera, y esta vez en el lazo de
# desarrollo y no en CI: `go test . -run "Paquetes"` dio `ok` porque el patron no
# casaba con el test que importaba, y ese `ok` se llevo a un informe. La puerta
# estaba puesta en CI, o sea a diez minutos y un empujon de distancia del sitio
# donde se decide si algo esta hecho.
#
# La regla, que esta escrita en CLAUDE.md: **ningun resultado de test cuenta en
# un informe si no salio de la puerta.** El `-run` a mano queda para depurar,
# nunca para afirmar.
#
# COMO NO SE DUPLICA NADA. Las puertas NO se declaran aqui: se LEEN de
# `.github/workflows/*.yml`. Escribir la lista dos veces seria exactamente la
# guarda que este proyecto ya ha visto fallar catorce veces, una copia que se
# queda vieja y sigue dando verde. Si CI gana una puerta, este script la corre
# sin tocarlo; si la cuenta cambia, PUERTAS_ESPERADAS no cuadra y esto se pone
# rojo, y `comprobar_test.go` lo pone rojo tambien dentro de la suite.
#
# Uso:
#   ./comprobar.sh          formato, vet, build y todas las puertas
set -uo pipefail

cd "$(dirname "$0")" || exit 1

# El numero de invocaciones de puerta() que hay hoy en .github/workflows/*.yml.
# Se sube o se baja EN EL MISMO COMMIT que anade o quita una puerta de CI, y
# `comprobar_test.go` comprueba que sigue cuadrando. Es incomodo a proposito:
# obliga a notar cuando el conjunto de puertas MENGUA, que es la unica direccion
# que nadie mira.
PUERTAS_ESPERADAS=21

rojo=0

paso() {
  local nombre="$1"
  shift
  echo "== $nombre"
  if ! "$@"; then
    echo "PASO ROTO: $nombre"
    echo "  Argumentos: $*"
    rojo=1
    return 1
  fi
  return 0
}

# gofmt sobre la lista de git y no sobre `.`: `.` entra en los worktrees de los
# agentes bajo .claude/, que son repos aparte, y ensucia la puerta con ficheros
# que no son del proyecto.
formato() {
  local sucios
  sucios=$(gofmt -l $(git ls-files '*.go'))
  if [ -n "$sucios" ]; then
    echo "sin formatear:"
    echo "$sucios"
    echo "  Arreglo: gofmt -w \$(git ls-files '*.go')"
    return 1
  fi
  return 0
}

paso "formato (gofmt sobre git ls-files)" formato
paso "vet" go vet ./...
paso "build" go build ./...

# ---------------------------------------------------------------------------
# Las puertas, leidas de los workflows.
# ---------------------------------------------------------------------------

source .github/puerta.sh

lineas=()
while IFS= read -r cruda; do
  lineas+=("$cruda")
done < <(grep -hE '^[[:space:]]*puerta "' .github/workflows/*.yml)

encontradas=${#lineas[@]}
if [ "$encontradas" -ne "$PUERTAS_ESPERADAS" ]; then
  echo "PASO ROTO: extraccion de puertas"
  echo "  he encontrado $encontradas invocaciones de puerta() en .github/workflows/*.yml"
  echo "  y PUERTAS_ESPERADAS dice $PUERTAS_ESPERADAS."
  if [ "$encontradas" -eq 0 ]; then
    echo "  CERO. O el directorio de workflows se movio, o la forma de la invocacion"
    echo "  cambio y este script lleva desde entonces corriendo el vacio y saliendo"
    echo "  verde. Es la misma familia que puerta.sh existe para cazar."
  elif [ "$encontradas" -lt "$PUERTAS_ESPERADAS" ]; then
    echo "  Han DESAPARECIDO puertas de CI. Si el recorte es intencionado, baja"
    echo "  PUERTAS_ESPERADAS en el mismo commit y di por que."
  else
    echo "  CI ha ganado puertas y el lazo local no se ha enterado. Arreglo: sube"
    echo "  PUERTAS_ESPERADAS a $encontradas."
  fi
  exit 1
fi

# El detector de carreras exige cgo, y la maquina de desarrollo es Windows sin
# compilador de C (lo dice ci.yml). No se salta a ciegas: se pregunta, y si se
# salta se DICE, porque una puerta saltada en silencio es una puerta que no
# existe.
hay_cgo=0
if [ "$(go env CGO_ENABLED)" = "1" ]; then
  hay_cgo=1
fi

saltadas=0
for linea in "${lineas[@]}"; do
  # Quitar la sangria del YAML.
  linea="${linea#"${linea%%[![:space:]]*}"}"
  # Las expresiones de GitHub (${{ matrix.sistema }}) son parte del NOMBRE de la
  # puerta, no de sus argumentos. En bash son un error de sintaxis, asi que se
  # sustituyen por la etiqueta que corresponde aqui.
  linea=$(sed -E 's/\$\{\{[^}]*\}\}/local/g' <<<"$linea")

  if [[ "$linea" == *" -race"* ]] && [ "$hay_cgo" -ne 1 ]; then
    echo "PUERTA SALTADA: $linea"
    echo "  -race exige cgo y aqui CGO_ENABLED=0. En CI si corre (ubuntu-latest)."
    saltadas=$((saltadas + 1))
    continue
  fi

  # Los pasos de CI que corren una puerta con entorno propio. Si aparece otro,
  # comprobar_test.go se pone rojo hasta que se declare aqui: una puerta que en
  # CI corre con una variable puesta y en local sin ella no es la misma puerta,
  # sale verde comprobando otra cosa.
  #
  # El nombre se casa CON LAS COMILLAS Y EL ESPACIO de detras. "suite completa"
  # es prefijo de otras cuatro puertas ("...sin IA", "...con detector de
  # carreras", "...en local", "...antes de empaquetar"), y un case sin el cierre
  # se las llevaria todas por delante.
  case "$linea" in
  *'puerta "suite completa" '*) export GOPROXY=off ;;
  *'puerta "suite completa sin IA" '*) export PLAZUM_SIN_IA=1 ;;
  esac

  eval "$linea"

  unset PLAZUM_SIN_IA GOPROXY
done

if ! cerrar_puertas; then
  rojo=1
fi

if [ "$saltadas" -gt 0 ]; then
  echo "$saltadas puertas saltadas en esta maquina (dicho arriba, con el motivo)."
fi

if [ "$rojo" -ne 0 ]; then
  echo
  echo "COMPROBACION EN ROJO. No se commitea, y ningun numero de esta salida vale"
  echo "para un informe."
  exit 1
fi

echo
echo "COMPROBACION EN VERDE: $PUERTAS_ESPERADAS puertas leidas de los workflows,"
echo "$((PUERTAS_ESPERADAS - saltadas)) ejecutadas aqui, mas formato, vet y build."
