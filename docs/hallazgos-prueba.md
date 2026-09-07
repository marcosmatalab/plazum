# Hallazgos del eslabón de la prueba (D-22, A1)

> **Fecha: 07-09-2026.** Todo número de aquí sale de la salida de un comando de
> esa sesión, no de memoria. El punto de partida es `258d06f`.

---

## La verificación del paso cero, y lo que no cuadraba

Las siete cifras del encargo se reprodujeron una a una y **todas cuadran**: 549
obligaciones, 34 `observable` por faceta, el reparto de recursos entero,
`Recolectar` sin implementaciones, `estado.Calcular` llamado sólo desde
`nucleo/expediente/expediente.go:717`, ninguna clave `pruebas` en el esquema, y
12 paquetes vacíos.

**Lo que no cuadraba no era una cifra: era el techo que se derivaba de una.**

| dónde vive `observable` | obligaciones |
|---|---|
| clase primaria (`clase_e2e`) | **99** |
| faceta (`facetas[]`) | **34** |
| en las dos a la vez | **0** |
| **techo real** | **133 de 549** |

El linter valida los dos campos contra el mismo vocabulario
(`nucleo/corpus/paquete.go:1833` y `:1839`) y los conjuntos son disjuntos. La
orden del encargo contaba sólo las facetas, así que su 34 es cierto y mide
facetas; lo que no se sostenía era *«tienen 34 obligaciones donde
engancharse»*, que habría publicado un techo **cuatro veces más bajo que el
real**, y además en la dirección que hace el argumento más dramático.

**El argumento sobrevive entero** —133 de 549 sigue siendo menos de una cuarta
parte— y la lección es de forma y no de aritmética: **un cardinal sobre un
vocabulario que vive en dos sitios se cuenta en los dos o no se cuenta.**

Y hay una segunda corrección, de numeración: la entrada se pidió como D-21 y
**D-21 existe** desde el 02-09-2026. Va como D-22.

---

## Lo que A1 cambia, medido

- El corpus real declara **1 prueba** (`paquetes/ens`, art. 20, mínimo
  privilegio, recurso `Privilegio`, TTL `P30D`, SLA `P7D`). Una, y se dice que
  es una: la campaña de los 12 marcos es la casilla de la etapa 3.
- **133 obligaciones observables, 132 sin prueba declarada.** El cardinal lo
  deriva `TestElTechoDeLaRecoleccionSeDerivaYNoSeEscribe`, que **no afirma un
  techo ni un suelo** —eso sería una promesa sobre el contenido del corpus, y el
  corpus crece— sino que el contador cuenta.

## Las tres puertas existentes que pararon esto, las tres con razón

1. **La frontera legal** exigió clasificar los cinco campos de texto nuevos. El
   `predicado` quedó como **prosa**, que es lo correcto y lo importante: es
   exactamente el sitio donde alguien pega el enunciado de un control de un
   catálogo de pago creyendo que ayuda.
2. **El vocabulario cerrado de `licencia_fuente`** rechazó `del_proyecto` (es
   `del-proyecto`).
3. **El de `identificador`** rechazó `sin_identificador` y, después, un
   `registro` que sólo usa el esquema de ISO.

Un paquete sintético de test tuvo que pasar por las mismas tres puertas que un
paquete real, que es la señal de que las puertas no tienen puerta de servicio.

---

## Las mutaciones (M25 a M30)

Todas sobre árbol limpio, con `.github/mutar.sh`, restauradas desde la copia.

| # | Qué se rompió | Puerta | Resultado |
|---|---|---|---|
| M25 | un campo `Cumple bool` añadido a `estado.Observacion` | `TestUnRecolectorNoPuedeDevolverUnVeredicto` | **cazada** |
| M26 | `Satisfecho` renombrado a `Conforme` en todo el árbol | la misma | **cazada** (a la segunda: ver abajo) |
| M27 | la guarda de `observable` apagada (`if false`) | `TestLasFormasDeRomperUnaPrueba` | **cazada** |
| M28 | el contraste `sla > ttl` aflojado ×1000 | la misma | **cazada** (a la segunda: la primera no compilaba) |
| M29 | el TTL vacío pasa a significar «no caduca» | la misma | **cazada** |
| M30 | el recurso deja de contrastarse contra la obligación | la misma | **cazada** |

### M26 sobrevivió la primera vez, y el motivo es doctrina

El `sed` que renombraba `Satisfecho` alcanzó también a `veredicto_test.go`, o
sea que **la justificación siguió al campo**: se renombró la excusa a la vez que
lo excusado. La regla de la casa lo dice con estas palabras: *si el test prueba
contra una lista escrita a su lado, muta **fuera** de esa lista.* Repetida
excluyendo el fichero de la puerta, se cazó en las dos direcciones (el campo sin
justificar y la justificación huérfana).

### M28 no compiló, y `mutar.sh` se negó a leer su rojo

`if okTTL && okSLA && false` deja `ttl` declarada y sin usar. La guarda 3 del
script hizo su trabajo: **un fallo de compilación no produce líneas `--- FAIL`**,
así que su rojo no demuestra que la puerta cace nada. Se rehízo como `sla >
ttl*1000`, que compila y afloja de verdad.

---

## Lo que A1 NO cierra, dicho

- **La pantalla de controles no enseña estado de evidencia.** Lo que pinta hoy
  es aplicabilidad derivada de la entrevista (`aplica` / `no aplica` / `falta
  contestar`), en `superficies/pantallas/derivacion.go:470`. Así que la pregunta
  de la pasada 3 —*un CISO ve un control en verde: ¿sabe por qué, desde cuándo y
  de dónde salió el dato?*— **hoy ni siquiera se puede formular**, y A1 no la
  arregla: la hace contestable, porque el dato ya existe (el `predicado` dice por
  qué, el `ttl` con `Recolectada` dice desde cuándo, y `Recolector` con
  `HashCarga` dicen de dónde). Ponerlo en pantalla es **A2**.
- **Y un riesgo para A2, apuntado ahora que se ve**: el verde de hoy significa
  **aplica** y el verde de mañana significará **consta**, y los dos van a estar
  en la misma pantalla. Dos verdes que significan cosas distintas a un palmo uno
  del otro es como se construye una pantalla que engaña sin mentir en ningún
  campo.
- **La obligatoriedad del predicado ya es exigible; su CONTENIDO no se
  comprueba.** El linter exige que haya predicado, no que el predicado diga algo
  evaluable. Eso último no lo puede hacer un linter sin un lenguaje de
  predicados (CEL, que está planeado en `DEPENDENCIAS.md` y no ha entrado).
