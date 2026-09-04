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

| cuántas | de qué dependen |
|---|---|
| **1** | sólo de la identidad. Es la del acta, que se niega a componerse sin el nombre de la organización: con la identidad guardada, su orden cae |
| **2** | de la identidad **y además** del cable de las respuestas de la cuenta al calendario y al plan, porque su alcance sale de lo contestado y no sólo del nombre |
| **2** | de nada de esto. Son las de la revisión de accesos (`plazum accesos ver` y `plazum serve --accesos-fichero`), que es otro frente |

**Dónde vive la corrección**: `docs/pendientes.md`, en el P0 del tramo 4, corregida en `ea103bd`. El cuerpo del propio `e1abac0` también la afirmaba de más y quedó igualmente sin corregir en su sitio.

**Cómo se detectó**: releyendo el commit ya escrito, antes de que nadie lo señalara. No la cazó ninguna puerta, y no hay puerta que pueda cazarla: un asunto de commit no lo lee ningún test.
