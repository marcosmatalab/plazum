# Erratas: lo que un commit afirmó de más y no se puede corregir donde lo dijo

> **Por qué existe este fichero.** La historia de git es **append-only**: el cuerpo de un commit se puede corregir en un commit posterior, y el **asunto no se puede corregir nunca**. Cuando una sobreafirmación vive en el asunto, quien lea el log la va a ver **antes** que su corrección, y a veces en lugar de ella, porque un `git log --oneline` no enseña cuerpos.
>
> Éste es el único sitio donde una historia append-only puede decir la verdad sobre sí misma.

## Cómo se usa

Una entrada por sobreafirmación detectada **después** de commitear, con:

- el **SHA** del commit que la contiene, sacado de `git rev-parse` y no escrito de memoria,
- **qué afirma de más**, citado,
- **la corrección**, con su cardinal si lo tiene,
- y **dónde vive la corrección** en el árbol, para que no haya que fiarse de este fichero tampoco.

**Lo que NO entra aquí**: una afirmación que era cierta al escribirse y caducó después. Eso no es una errata, es una foto vieja, y su sitio es el fichero que la publica. La errata es lo que **ya era falso cuando se escribió**.

## La regla preventiva, que es la mitad barata

**Un asunto de commit no lleva jamás un cardinal ni una totalidad.** Nada de «las tres», «todas», «cierra X», «arregla la familia». El asunto dice la **intención**; el cuerpo dice la **cuenta**.

El motivo es exactamente la asimetría de arriba y no una cuestión de estilo: **el cuerpo se puede corregir en un commit posterior y el asunto no**, así que todo lo que pueda resultar falso va donde se puede arreglar. Está en `CLAUDE.md`, y este fichero es lo que queda para cuando la regla llega tarde.

## Las entradas

### 1. `e1abac0` — «apaga las TRES ordenes»

**Asunto entero**: `el paso 2 del P0 tambien tenia una pieza sin costear, y apaga las TRES ordenes`

**Qué afirma de más.** Que el almacén de identidad de la instalación (el sujeto y el nombre de la organización, que hoy entran por bandera de arranque) **apaga** las tres órdenes de terminal que dependen de él. Es falso: esa pieza es **común** a las tres, no **suficiente** para ellas.

**La corrección, con el reparto real de las 5 órdenes que cobra el TTFV:**

| orden | qué hace falta para que la pantalla deje de pedirla |
|---|---|
| `plazum alcance` + `plazum serve --alcance` (calendario y plan, **2 cobradas**) | identidad **+** el cable de las respuestas de la cuenta. `fuenteCal`/`fuenteEsc` sólo existen si hay un alcance |
| `plazum serve --acta-organizacion …` (**1**) | organización **+** periodo (desde y hasta) **+ la campaña de accesos**. `fuenteDelActa` devuelve error sin `HayCampana` |
| `plazum accesos ver` + `plazum serve --accesos-fichero` (**2**) | los datos de accesos de la campaña |

**Dónde vive la corrección**: `docs/pendientes.md`, en el P0 del tramo 4, corregida en `ea103bd`. El cuerpo del propio `e1abac0` también la afirmaba de más y quedó igualmente sin corregir en su sitio.

**Cómo se detectó**: releyendo el commit ya escrito, antes de que nadie lo señalara. No la cazó ninguna puerta, y no hay puerta que pueda cazarla: un asunto de commit no lo lee ningún test.

### 1-bis. La primera corrección de esta errata TAMBIÉN estaba mal

**Y esto es lo que hay que aprender, más que la errata.** La versión anterior de la tabla de arriba decía que **1 de las 5 órdenes depende «sólo de la identidad»**, y era la del acta. Es falso: `cmd/plazum/serve.go` calcula `hayCampana` de `--accesos-fichero` y `--accesos-ledger`, y `fuenteDelActa` **se niega a componer sin ella**. O sea que **la orden del acta depende del frente de accesos**, que es justo el que la tabla ponía como independiente.

**Ninguna de las 5 órdenes cae con la identidad sola.** El reparto verificado es el de la tabla de arriba, y sale de leer las tres funciones que deciden si cada fuente existe, no de leer una.

**El patrón, que ya va tres veces en dos días**: escribí un cardinal de dependencias **leyendo una función y suponiendo el resto**. La primera vez fue «apaga las tres», la segunda «1 sólo-identidad», y las dos veces el error fue **hacia abajo en el coste**, o sea a favor de que el trabajo pareciera más pequeño. Regla que sale: **un cardinal de dependencias se traza leyendo TODAS las funciones que deciden, y se escribe con el nombre de cada una al lado**, para que quien lo lea pueda ir a mirar. Las tres de aquí son `fuenteDelActa`, la construcción de `fuenteCal`/`fuenteEsc` en `serve.go`, y `construirUAR`.
