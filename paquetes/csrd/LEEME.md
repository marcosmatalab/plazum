# csrd: informacion de sostenibilidad de las empresas

**Estado: esqueleto.** Metadatos correctos y cero obligaciones. La transcripcion
esta en el plan de autoria, no hecha.

## El instrumento que hay que leer, que no es el que da nombre al paquete

Lo llamamos `csrd` y su `urn` dice `urn:eu:dir:2022:2464`, que es la **directiva
modificativa**. Las obligaciones **no viven ahi**. Viven en la **Directiva
2013/34/UE**, la directiva contable, que es la que la CSRD modifica y la que hay
que transcribir.

Por eso la `fuente` apunta a la 2013/34/UE y no a la 2022/2464.

**Lo que sigue mal**: el `urn` todavia nombra a la modificativa. Cambiarlo cambia
la identidad del paquete y queda para la autoria.

**Aviso de version**: hay que leer el consolidado vigente el dia de la
transcripcion. La 2013/34/UE se ha modificado despues de la CSRD, asi que fijar
hoy una version concreta seria fijar una que ya no es la ultima.

## Que vincula de verdad en Espana

Nada de este paquete, todavia. La CSRD es una **directiva**: no vincula por si
misma a una empresa espanola, vincula la norma que la transpone. La Ley de
Sostenibilidad Empresarial seguia en tramitacion parlamentaria en 2026 segun
fuentes secundarias, y no se ha localizado publicacion en el BOE.

Y hay un segundo motivo para no escribir todavia los relojes: las fechas de
aplicacion las movio dos anos la **Directiva (UE) 2025/794**, la llamada "stop
the clock". Escribir hoy los relojes de csrd es escribir relojes que van a
cambiar de fecha.

## Donde estan los relojes, segun el censo

- Plazo (3): art. 30.1 (publicacion de las cuentas anuales y el informe de
  gestion, doce meses desde la fecha del balance), art. 40 quinquies y art. 48
  quinquies (doce meses).
- Periodicidad (5, todas anuales): art. 4, art. 19 bis, art. 29 bis, art. 40 bis
  y art. 48 ter.
- Evento: ninguno. La CSRD es puro calendario.

Detalle completo en `docs/censo-relojes.md`.

## Derechos

Texto del DOUE. La Decision 2011/833/UE autoriza la reutilizacion **con
atribucion**, y el aviso literal viaja en el campo `atribucion` del paquete y
sale en la pantalla del producto.

## Aviso

Esto no es asesoramiento juridico.
