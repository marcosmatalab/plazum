#!/usr/bin/env bash
# presupuesto.sh: los tres presupuestos de la etapa 2, comparados de UNA sola
# forma y con su control negativo dentro.
#
# POR QUE EXISTE. ETAPAS.md fija tres numeros (binario <25 MB, arranque <3 s,
# RAM <256 MB) y hoy los tres se cumplen con muchisimo margen: el binario mide
# 9,5 MB de 25. Un presupuesto con ese margen NO SE VE FALLAR NUNCA, y una
# puerta que nunca se ha visto fallar no es una puerta: es un numero decorativo
# del que nadie sabe si compara algo. Exactamente el mismo problema que tenia
# `go test` saliendo con 0 sin haber ejecutado nada, y por eso esto vive al lado
# de puerta.sh.
#
# El arreglo es el mismo: cada medida pasa DOS veces por la misma comparacion.
#
#   la de verdad   contra su limite de ETAPAS.md. Tiene que pasar.
#   la de control  contra un limite imposible. Tiene que FALLAR.
#
# Si la segunda pasa, la comparacion no compara y el paso se pone rojo aunque la
# medida fuera excelente. Asi la puerta se ve fallar en cada ejecucion, sin
# esperar al dia en que el binario engorde 15 MB.
#
# Uso:
#   source .github/presupuesto.sh
#   presupuesto "binario (bytes)" "$bytes" "$limite"
#   ...
#   cerrar_presupuestos      # sale con 1 si no se midio nada, o si algo fallo

set -uo pipefail

_PRESUPUESTOS_FALLOS=0
_PRESUPUESTOS_CORRIDOS=0

# _comparar sale con 0 si la medida CABE en el limite y con 1 si lo alcanza o lo
# pasa. Es la unica comparacion del fichero: la usan la medida de verdad y su
# control negativo, que es lo que hace que el control valga para algo.
#
# Se hace con awk y no con [ -lt ] porque las medidas son bytes y milisegundos y
# se pasan de lo que shell aritmetico maneja comodo.
_comparar() {
  awk -v m="$1" -v l="$2" 'BEGIN { exit (m+0 >= l+0) ? 1 : 0 }'
}

presupuesto() {
  local que="$1" medida="$2" limite="$3"
  _PRESUPUESTOS_CORRIDOS=$((_PRESUPUESTOS_CORRIDOS + 1))
  echo "  $que: $medida (presupuesto $limite)"

  # CONTROL NEGATIVO, en cada llamada y no una vez al ano: la MISMA comparacion
  # contra un limite imposible tiene que decir que no cabe.
  if _comparar "$medida" 0; then
    echo "PUERTA ROTA: la comparacion de presupuestos da por buena una medida de"
    echo "  $medida contra un limite de 0. No esta comparando nada, asi que el verde"
    echo "  de '$que' no significa nada tampoco."
    echo "  Arreglo: mirar _comparar en .github/presupuesto.sh."
    _PRESUPUESTOS_FALLOS=$((_PRESUPUESTOS_FALLOS + 1))
    return 1
  fi

  if ! _comparar "$medida" "$limite"; then
    echo "PUERTA ROTA: $que vale $medida y el presupuesto de ETAPAS.md es $limite."
    echo "  No se sube el limite sin decir por que EN EL MISMO COMMIT: el numero"
    echo "  es una promesa de compra, no una preferencia."
    _PRESUPUESTOS_FALLOS=$((_PRESUPUESTOS_FALLOS + 1))
    return 1
  fi

  echo "  presupuesto ok: $que"
  return 0
}

cerrar_presupuestos() {
  if [ "$_PRESUPUESTOS_CORRIDOS" -eq 0 ]; then
    echo "PUERTA ROTA: no se ha medido NINGUN presupuesto en este paso."
    echo "  Un paso que no mide nada y sale verde es el verde mas falso de todos."
    return 1
  fi
  if [ "$_PRESUPUESTOS_FALLOS" -gt 0 ]; then
    echo "$_PRESUPUESTOS_FALLOS de $_PRESUPUESTOS_CORRIDOS presupuestos rotos."
    return 1
  fi
  echo "$_PRESUPUESTOS_CORRIDOS presupuestos medidos, todos dentro."
  return 0
}
