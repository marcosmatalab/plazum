# pci-dss: que trae este paquete y que no

**Estrato: referencial. Aqui NO esta el texto de PCI DSS y no va a estarlo.** El
marco lo publica su consejo de normas con sus propias condiciones, y este proyecto
no redistribuye su contenido. Lo que hay aqui son **rituales de plazum**: siete
ceremonias con reloj, con el intervalo puesto por nosotros y el argumento escrito
al lado.

El linter del proyecto lo hace cumplir por ti: un campo de prosa de un paquete
referencial con mas de 120 caracteres no carga, y el paquete entero se rechaza.

## Lo primero, porque es lo que mas confunde

**Ninguno de estos siete relojes es un requisito del marco.** Sus identificadores
empiezan por `pcidss.ritual.` para que nadie los confunda, y su campo `articulo`
dice literalmente `ritual plazum sobre ...`. Si un auditor te pregunta de donde
sale una de estas fechas, la respuesta es *«la pone plazum, aqui esta el porque, y
la puedo mover»*, no *«lo exige el marco»*.

**Y ninguno dice a que requisito sirve.** Ese anclaje exige la copia licenciada
delante, y este paquete no la tiene. Lo pones tu en tu instancia: es una linea por
ritual. El hueco esta contado, no escondido: **7 de 7**, en
`docs/hallazgos-censo-a.md`.

## Que hay dentro

| Ritual | Reloj | Clase |
|---|---|---|
| Revision de accesos | cada 3 meses | procedimental |
| Analisis de vulnerabilidades | cada 3 meses | observable |
| Prueba de intrusion | cada 12 meses | procedimental |
| Revision de las politicas | cada 12 meses, reabre por cambio significativo | documental |
| Formacion de concienciacion | cada 12 meses | procedimental |
| Revision de proveedores | cada 12 meses | procedimental |
| Remediacion de vulnerabilidad critica | 30 dias naturales desde la deteccion | remediacion |

Los seis primeros son cadencias con `origen_del_intervalo: "propuesto"`, o sea que
**el numero es nuestro y lo puedes mover en las dos direcciones**. Cada uno trae
en el propio paquete su `justificacion_del_intervalo` (por que ese numero y no
otro) y su `cuando_cambiarlo` (una condicion para acortarlo y una para alargarlo,
cada una con el supuesto que la hace cierta). El septimo es un plazo.

Cada reloj trae al menos tres casos dorados en `pruebas/`, derivados de
`RITUALES.md` y no de la implementacion, y se recalculan contra el motor en cada
ejecucion de los tests. Si el motor y un caso discrepan, gana el caso.

## A quien alcanza

A quien declare que adopta el marco. Las reglas de `aplicabilidad` derivan
`aplica(<ritual>, E)` desde `adopta(E, "pci-dss")` y desde nada mas: este paquete
no afirma nada sobre tu sector, tu tamano ni tu volumen de transacciones, porque
esos umbrales estan en el marco y el marco no se puede leer desde aqui.

## Lo que este paquete NO hace

- **No sustituye al marco.** Sin tu copia, aqui hay rituales, no requisitos.
- **No te valida ni te certifica.** Eso lo hace un evaluador cualificado o un
  cuestionario de autoevaluacion, segun tu nivel de comerciante o proveedor.
- **No mapea ritual a requisito.** Ver arriba: 7 de 7 sin anclar, a proposito.
- **No cubre la revision de reglas del cortafuegos, la de la segmentacion ni la
  rotacion de claves.** Estan identificadas y no escritas, con su motivo uno a
  uno, en `RITUALES.md` seccion 3 y en `docs/hallazgos-censo-a.md`.

## Aviso

Esto no es asesoramiento juridico ni una interpretacion autorizada del marco. Las
cadencias de `RITUALES.md` son criterio de plazum y no proceden de PCI DSS.
