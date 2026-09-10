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

Lo llamamos `eidas2` porque es el nombre con el que se conoce la reforma, pero el
acto es el **Reglamento (UE) n.o 910/2014**: ahi viven las obligaciones, y el
Reglamento (UE) 2024/1183 es el que lo **modifica**. Por eso el `urn` y el
`identificador` dicen los dos `910/2014`. El censo de relojes se hizo sobre el
consolidado `02014R0910-20241018`, y ese es el texto contra el que se escribe
cada obligacion.

**Hasta el 10-09-2026 el `urn` decia `urn:eu:reg:2024:1183`**, o sea el
modificativo, y este fichero lo tenia anotado como *«lo que sigue mal»* dejandolo
«para la autoria». No impidio nada: `art. 24.3`, que es texto base (marca ▼B),
heredo de esa identidad el **20-05-2024**, que es el vigor del modificativo,
cuando lo que le obliga es el **01-07-2016** del art. 52.2. Y la puerta de
vigencias dijo que **casaba**, porque emparejaba por el `urn`. El porque, con las
tres salidas que se sopesaron, en D-26.

**De donde sale la fecha de cada una de las tres**, que ahora no es la misma para
todas:

| obligacion | marca | vigencia | de donde |
|---|---|---|---|
| art. 19 bis.1.b | ▼M2 | 2024-05-20 | vigor del Reglamento (UE) 2024/1183, art. 2 |
| art. 24.2.f ter | ▼M2 | 2024-05-20 | idem |
| art. 24.3 | ▼B | 2016-07-01 | aplicacion del 910/2014, art. 52.2 |

Las dos primeras llevan `origen: "propia"` porque salen de otro acto; la tercera
hereda la del paquete, y ahora la hereda **porque es verdad**.

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
