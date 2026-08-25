# demo-empresa: el paquete del `dutiq demo`

**Esto no es una norma y no lo pretende.** Es un paquete de clase `propio`
(clase 4) con datos sintéticos del proyecto, licencia Apache-2.0. Sus seis
obligaciones se llaman "Demo 1" a "Demo 6" y su texto lo hemos escrito
nosotros. Ninguna cita apunta a un boletín oficial, y no debe apuntar nunca.

## Para qué está

Para que un evaluador que acaba de descargar el binario vea el producto lleno
en un comando, sin configurar nada, sin red y sin instalar el corpus real. Lo
consume `dutiq demo`, que lo lleva empotrado (ver `incrustado.go`).

Lo que enseña, y por qué está elegido así:

| Pieza | Qué ilustra |
|---|---|
| `demo.politica_de_seguridad` | obligación documental con entregable y trazabilidad a campos |
| `demo.inventario_de_activos` | obligación observable, la que un conector puede comprobar solo, y una exención escrita con la negación del dialecto |
| `demo.revision_trimestral_de_accesos` | reloj periódico corto, con cierre al final del día |
| `demo.auditoria_bienal` | ciclo largo que se reinstancia, con el borde del 29 de febrero |
| `demo.notificacion_de_incidente` | plazo en horas exactas y cadena de escalado |
| `demo.plan_de_continuidad` | obligación derivada de un **agregado** sobre respuestas por activo |

## Los predicados van prefijados con `demo.` a propósito

El aislamiento por espacio de nombres del motor de aplicabilidad **no está
implementado** (consta como P1 en `docs/pendientes.md`). Mientras no lo esté,
dos paquetes que declaren `en_ambito` colisionan. Este paquete se carga junto al
corpus real cuando el operador lo tiene, así que todos sus predicados propios
llevan el prefijo `demo.` escrito a mano. Si algún día se implementa el
aislamiento, este prefijo sobra y se puede quitar; hasta entonces es lo único
que impide que el demo contamine las conclusiones de una norma de verdad.

## Las fechas del alcance son relativas

`alcance.json` guarda las respuestas de la empresa de ejemplo. Sus fechas van
como desplazamientos desde el instante de ejecución (`-45d`, `-700d`, `-30h`) y
no como fechas fijas. Un demo con fechas fijas envejece, y a los seis meses
enseña tres relojes vencidos, que es lo contrario de lo que tiene que enseñar.
Se admite también una fecha absoluta `AAAA-MM-DD` para quien quiera congelar un
escenario.

## Los casos dorados

Están en `pruebas/` y son tres por reloj, derivados del texto del propio demo
más las reglas de cómputo generales que sí son reales (recorte al último día del
mes, cierre al final del día en plazos por meses, horas exactas). Corren en cada
`go test` con el resto del corpus publicado, y `dutiq demo` los vuelve a correr
delante del operador contra el motor de verdad.
