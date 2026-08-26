# psd2: servicios de pago

**Estado: esqueleto.** Metadatos correctos y cero obligaciones. La transcripcion
esta en el plan de autoria, no hecha.

## Que vincula de verdad, y por que esto importa antes de leer nada mas

**Esta directiva no te obliga a ti.** Una directiva obliga a los Estados
miembros. Lo que obliga a un proveedor de servicios de pago espanol es la norma
de transposicion, que aqui **si existe**:

> **Real Decreto-ley 19/2018, de 23 de noviembre**, de servicios de pago y otras
> medidas urgentes en materia financiera.
> <https://www.boe.es/eli/es/rdl/2018/11/23/19/con>

Los plazos que se pueden exigir en Espana son los del RDL, y son los que hay que
mirar antes de confiar en ninguna cifra de este paquete.

Lo que esta sin hacer, dicho en vez de disimulado:

- **El RDL 19/2018 no esta censado.** Nadie ha comprobado todavia si sus cifras
  coinciden con las de la directiva.
- **Este paquete sigue apuntando a la directiva** en su `urn` y en su `identificador`,
  porque cambiar eso es decidir si el RDL es un marco propio o una capa de este
  paquete, y esa decision es de la autoria. Consta como hueco en
  `docs/censo-relojes.md`.
- Hasta que eso se resuelva, las cifras de mas abajo son **la referencia
  europea**, no la obligacion espanola.

## Donde estan los relojes, segun el censo

Marca de honestidad del censo: **estimado**. Se verifico el nucleo de ejecucion,
reembolso y reclamaciones; el resto no se reviso apartado a apartado.

- Plazo (6 apartados, 7 relojes): art. 71.1 (trece meses del usuario para
  notificar, ventana que el proveedor vigila), art. 73.1 (devolucion de la
  operacion no autorizada, final del dia habil siguiente), art. 77.2 (diez dias
  habiles desde la solicitud), art. 82.2 (hasta cuatro dias habiles fuera del
  euro), art. 83.1 (abono al proveedor del beneficiario, final del dia habil
  siguiente) y art. 101.2 (quince dias habiles, y hasta treinta y cinco en
  situaciones excepcionales, que son dos relojes).
- Periodicidad (2): art. 95.2 (evaluacion anual de riesgos operativos y de
  seguridad) y art. 96.6 (datos de fraude a la autoridad al menos una vez al
  ano).
- Evento (3): art. 96.1 (incidente grave, sin numero en la directiva), art. 73
  (operacion no autorizada) y art. 76 con 77 (solicitud de devolucion).

Detalle completo en `docs/censo-relojes.md`.

## Derechos

Texto del DOUE. La Decision 2011/833/UE autoriza la reutilizacion **con
atribucion**, y el aviso literal viaja en el campo `atribucion` del paquete y
sale en la pantalla del producto. Cuando se transcriba el RDL 19/2018, su texto
es del BOE y va con el regimen del art. 13 TRLPI, que es otro estrato de fuente
y otra atribucion.

## Aviso

Esto no es asesoramiento juridico.
