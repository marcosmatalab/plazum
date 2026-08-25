#!/usr/bin/env bash
# puerta.sh: la comprobacion que hace que una puerta de CI sea una puerta.
#
# POR QUE EXISTE. `go test` sale con CODIGO 0 en dos situaciones donde no ha
# comprobado nada, y las dos son verdes falsos indistinguibles de un verde de
# verdad:
#
#   go test -run TestQueYaNoSeLlamaAsi ./x   ->  "[no tests to run]", codigo 0
#   go test ./solo/paquetes/sin/tests/...    ->  "[no test files]",   codigo 0
#
# La primera muerde el dia que alguien renombra un test: la puerta se queda
# verde y nadie mira ya lo que decia mirar. La segunda muerde el dia que un
# directorio se mueve. Ninguna de las dos avisa.
#
# Es la TERCERA vez en dos semanas que una guarda de este proyecto no guardaba
# nada. Ver la familia en docs/pendientes.md.
#
# Uso:
#   source .github/puerta.sh
#   puerta "nombre legible" MINIMO ./paquete/... -run 'TestLoQueSea'
#   ...
#   cerrar_puertas          # sale con 1 si alguna fallo
#
# MINIMO es el numero de casos que esa puerta TIENE que ejecutar. No es un
# adorno: puesto en el numero real de hoy, caza tambien que alguien borre la
# mitad de los casos y deje uno. Si de verdad no se sabe, se pone 1 y se anota
# por que, pero 1 solo protege del cero.

set -uo pipefail

_PUERTAS_FALLOS=0
_PUERTAS_CORRIDAS=0

puerta() {
  local nombre="$1" minimo="$2"
  shift 2
  _PUERTAS_CORRIDAS=$((_PUERTAS_CORRIDAS + 1))

  echo "::group::puerta: $nombre"
  local salida estado
  salida=$(go test -v -count=1 "$@" 2>&1)
  estado=$?
  echo "$salida"
  echo "::endgroup::"

  if [ $estado -ne 0 ]; then
    echo "PUERTA ROTA: $nombre"
    echo "  go test salio con $estado. Argumentos: $*"
    _PUERTAS_FALLOS=$((_PUERTAS_FALLOS + 1))
    return 1
  fi

  # Se cuentan los casos EJECUTADOS, contando subtests, que es donde vive la
  # mitad de las comprobaciones de este proyecto.
  local ejecutados
  ejecutados=$(grep -cE '^[[:space:]]*--- (PASS|FAIL|SKIP)' <<<"$salida" || true)

  if [ "$ejecutados" -lt "$minimo" ]; then
    echo "PUERTA ROTA: $nombre"
    echo "  ha ejecutado $ejecutados casos y tenia que ejecutar al menos $minimo."
    if grep -q "no tests to run" <<<"$salida"; then
      echo "  go test dice 'no tests to run': el patron -run no casa con ningun test."
      echo "  Alguien renombro un test y esta puerta lleva desde entonces dando verde"
      echo "  sin comprobar nada. Arreglo: ajusta el patron, o el nombre del test."
    elif grep -q "no test files" <<<"$salida"; then
      echo "  go test dice 'no test files': el glob de paquetes no encuentra tests."
      echo "  Probablemente un directorio se movio. Arreglo: ajusta la ruta."
    else
      echo "  Se han borrado casos, o se han saltado. Arreglo: si el recorte es"
      echo "  intencionado, baja el minimo EN EL MISMO COMMIT y di por que."
    fi
    echo "  Argumentos: $*"
    _PUERTAS_FALLOS=$((_PUERTAS_FALLOS + 1))
    return 1
  fi

  echo "puerta ok: $nombre ($ejecutados casos, minimo $minimo)"
  return 0
}

cerrar_puertas() {
  if [ "$_PUERTAS_CORRIDAS" -eq 0 ]; then
    echo "PUERTA ROTA: no se ha ejecutado NINGUNA puerta en este paso."
    echo "  Un paso que no corre puertas y sale verde es el verde mas falso de todos."
    return 1
  fi
  if [ "$_PUERTAS_FALLOS" -gt 0 ]; then
    echo "$_PUERTAS_FALLOS de $_PUERTAS_CORRIDAS puertas rotas."
    return 1
  fi
  echo "$_PUERTAS_CORRIDAS puertas, todas en verde."
  return 0
}
