# La bitácora

Lo que ya pasó: relatos fechados de fallos, familias y decisiones. **Nada de
aquí hace falta para usar plazum ni para trabajar en él.** Está para una sola
cosa, y es la que ha cerrado casi todos los hallazgos de este repositorio:
**reconocer la séptima aparición de una familia**, que sobre un resumen no se
puede hacer.

| Dónde | Qué es |
|---|---|
| [`pendientes-historico.md`](pendientes-historico.md) | El cuerpo de `docs/pendientes.md`, mudado entero el 22-09-2026. Las familias con su relato, su cardinal y su fecha. Lo que sigue **abierto** se lee en [`../pendientes.md`](../pendientes.md), que es una tabla. |
| [`../hallazgos/`](../hallazgos/LEEME.md) | Los cuadernos de auditoría, uno por campaña, con su propio índice y su puerta en las dos direcciones. |

## Por qué los cuadernos de hallazgos no se fundieron aquí

Se midió antes de decidir, el 22-09-2026: **47 referencias en 35 ficheros**
apuntan a un cuaderno concreto, y **cinco de ellas son datos del corpus**
(`paquetes/ai-act/paquete.json`, `paquetes/cra/paquete.json`,
`paquetes/mica/paquete.json`, `paquetes/psd2-es/paquete.json` y
`paquetes/marcos-v1.json`), que es exactamente lo que no se toca por comodidad
documental. Fundir los diecinueve cuadernos en un resumen convertiría 47 citas
precisas en 47 enlaces a una línea, que es perder el dato y quedarse con la
forma.

Lo que sí se hizo fue darles esta puerta de entrada, para que la bitácora se lea
como una y no como dos montones separados.
