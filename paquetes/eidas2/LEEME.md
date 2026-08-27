# eidas2: identidad digital europea

**Estado: los tres plazos de 24 horas transcritos, con nueve casos dorados**
(27-08-2026). El resto del articulado sigue sin transcribir.

**Lo que hay que mirar de este paquete**, y es el mejor ejemplo del corpus de por
que un catalogo de controles no sirve: los arts. 19 bis.1.b y 24.2.f ter son los
dos plazos de veinticuatro horas, y **no cuentan desde el mismo hecho**. El del
prestador NO cualificado cuenta desde que se tiene CONOCIMIENTO; el del
CUALIFICADO, desde que el incidente SE PRODUJO. Un prestador cualificado que se
entera veintidos horas despues tiene dos, no veinticuatro. El detalle, en
`COMPUTO.md`.

## El instrumento que hay que leer, que no es el que da nombre al paquete

Lo llamamos `eidas2` y su `urn` dice `urn:eu:reg:2024:1183`, que es el
**reglamento modificativo**. Las obligaciones **no viven ahi**. Viven en el
**Reglamento (UE) 910/2014**, que es el texto que el 2024/1183 modifica, y que
es el que hay que transcribir.

Por eso el `identificador` de este paquete apunta al 910/2014 y no al 2024/1183. El
censo de relojes se hizo sobre el consolidado `02014R0910-20241018`, y ese es el
texto contra el que hay que escribir cada obligacion.

**Lo que sigue mal, dicho aqui y no escondido**: el `urn` todavia nombra al
modificativo. Cambiarlo cambia la identidad del paquete, que es lo que apunta el
expediente y lo que resuelve las equivalencias, asi que no se toca de pasada.
Queda para la autoria, que es quien decide si el paquete pasa a llamarse por el
910/2014.

**Aviso de version**: el consolidado se rehace cada vez que el texto se modifica,
asi que la version consolidada que hay que leer es la vigente el dia en que se
escriba el paquete, no la del censo. Se comprueba antes de transcribir.

## Donde estan los relojes, segun el censo

- Plazo: art. 19 bis.1.b (prestador no cualificado, 24 horas desde el
  conocimiento), art. 24.2.f ter (prestador cualificado, 24 horas), art. 12 bis
  (tres meses para subsanar la vulnerabilidad detectada).
- Periodicidad: art. 20.1 (auditoria de prestadores cualificados al menos cada
  24 meses) y art. 5 quater con 12 bis (evaluacion de vulnerabilidad cada dos
  anos).
- Evento: violacion de seguridad o interrupcion, vulnerabilidad no subsanada y
  cese de actividad del prestador cualificado.

Detalle completo, con la cita de cada uno, en `docs/censo-relojes.md`.

## Derechos

Texto del DOUE. La Decision 2011/833/UE autoriza la reutilizacion **con
atribucion**, y el aviso literal viaja en el campo `atribucion` del paquete y
sale en la pantalla del producto, no solo en este fichero.

## Aviso

Esto no es asesoramiento juridico.
