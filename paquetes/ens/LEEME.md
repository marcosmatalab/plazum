# ens: el Esquema Nacional de Seguridad, transcrito

**Estrato: transcrito.** El ENS es un real decreto publicado en el BOE, asi que
aqui esta su texto entero, palabra por palabra, con enlace a la fuente oficial en
cada obligacion. Lo permite el articulo 13 del texto refundido de la Ley de
Propiedad Intelectual, que deja fuera de la proteccion las disposiciones legales,
y lo exigen las condiciones de reutilizacion del BOE, que piden citar la fuente.

El texto no se ha tecleado: se extrae del XML del texto consolidado del BOE,
bloque a bloque y parrafo a parrafo, y cada obligacion guarda de que apartado
sale. Una errata de transcripcion es imposible por construccion.

## Que te obliga y desde cuando

| Norma | BOE | Te obliga desde |
|---|---|---|
| RD 311/2022, ENS | BOE-A-2022-7191 | 5 de mayo de 2022 |
| ITS de Conformidad con el ENS | BOE-A-2016-10109 | 3 de noviembre de 2016 |
| ITS de Informe del Estado de la Seguridad | BOE-A-2016-10108 | 22 de noviembre de 2016 |
| ITS de Notificacion de Incidentes | BOE-A-2018-5370 | 20 de abril de 2018 |

El paquete trae **132 obligaciones**: el articulado completo dirigido a la
entidad titular, el anexo I, las 73 medidas del anexo II, la disposicion
transitoria y las tres instrucciones tecnicas.

Cada obligacion lleva su propia fecha de vigencia, que es la que vincula. Hay una
obligacion **caducada** en el paquete y esta ahi a proposito: la disposicion
transitoria unica dio veinticuatro meses de adaptacion y vencio el 5 de mayo de
2024. Se conserva con `vigencia.hasta` porque un expediente de 2023 tiene que
poder explicarse con el derecho de 2023, no con el de hoy.

**Aviso para quien construya la pantalla**: hoy nada en el producto lee
`vigencia` obligacion a obligacion. Si una superficie ensena esta lista sin
filtrar, la primera linea que vera un CISO en 2026 sera un vencimiento en rojo de
mayo de 2024 por una obligacion que ya no le alcanza. Filtrar por `vigencia.desde`
y `vigencia.hasta` contra la fecha de evaluacion no es un adorno.

**A quien alcanza**: a todo el sector publico (art. 2.1) y, por contrato, a las
entidades privadas que le prestan servicios o le proveen soluciones, incluida la
obligacion de tener politica de seguridad (art. 2.3). Si eres un contratista,
esto te obliga a ti tambien, y el paquete lo pregunta.

## Los ocho relojes, con su fuente

| Que | Cada cuanto | De donde sale |
|---|---|---|
| Auditoria regular ordinaria | 2 anos | RD 311/2022 art. 31.1 |
| Reevaluacion de la categoria del sistema | 1 ano | RD 311/2022 anexo I.1 |
| Informe del estado de la seguridad (INES) | 1 ano | ITS INES, III.2 |
| Estadisticas de incidentes al CCN | 1 ano | ITS Incidentes, VI |
| Notificacion al CCN de incidente ALTO o superior | inmediata | ITS Incidentes, IV.3 |
| Autoevaluacion de conformidad, categoria BASICA | 2 anos | ITS Conformidad, III.2 |
| Certificacion de conformidad, MEDIA o ALTA | 2 anos | ITS Conformidad, III.3 |
| Plena adecuacion al ENS (ya vencido) | 24 meses | Disposicion transitoria unica |

Cada uno trae tres casos dorados en `pruebas/` derivados del texto legal, con la
cita de la que sale la fecha esperada, y se recalculan contra el motor en cada
ejecucion de los tests. Si el motor y un caso discrepan, gana el caso.

Las reglas de computo que el texto legal no fija (a que hora vence, que pasa si
cae en domingo) estan escritas y razonadas en `COMPUTO.md`, con la norma en la
que se apoyan. No estan escondidas en el codigo.

## Lo que este paquete NO hace todavia

Detalle nominal en `COBERTURA.md`. En corto:

- **Estan las 73 medidas del anexo II, pero solo sus requisitos base.** Ningun
  refuerzo (R1, R2, R3...) esta transcrito, porque el formato no puede declarar
  todavia a que nivel de que dimension se activa cada uno. Cada obligacion de
  medida lo dice en su cita.
- **No sabe filtrar por categoria.** El ENS entero se modula por la categoria del
  sistema, y el formato de paquete no puede declarar todavia "esto solo aplica a
  MEDIA y ALTA". Consecuencia practica: veras la autoevaluacion de BASICA y la
  certificacion de MEDIA o ALTA a la vez, y es el texto transcrito el que dice a
  quien alcanza cada una. Es la primera cosa que hay que arreglar.
- **No aplica la prorroga de tres meses** por fuerza mayor del art. 31.1 aunque
  su texto este transcrito, ni reinicia el ciclo solo cuando hay auditoria
  extraordinaria. Ver `COMPUTO.md`, apartado 5.
- **No lo ha revisado un jurista.** Lo ha escrito una sola persona contra el
  texto del BOE. La revision externa es una casilla abierta de la etapa 3.

## Dos avisos de vigilancia normativa

1. Las tres instrucciones tecnicas se dictaron bajo el **RD 3/2010, derogado**.
   Siguen vigentes porque no se oponen al RD 311/2022, pero sus remisiones
   internas apuntan a articulos que ya no existen. El paquete transcribe lo
   publicado sin corregirlo y lo explica en `COMPUTO.md`, apartado 6.
2. Un incidente que afecte a datos personales dispara **dos relojes a la vez**:
   el inmediato de la ITS y las 72 horas del art. 33 del RGPD. Lo dice la propia
   ITS en su apartado X.2. Instala tambien el paquete `rgpd`.

## Aviso

Esto no es asesoramiento juridico. El texto es el del BOE; la clasificacion en
obligaciones, las reglas de computo y los relojes son criterio de dutiq.

## Las reglas de aplicabilidad

Desde el 25 de agosto de 2026 este paquete declara sus propias reglas de
aplicabilidad, en el bloque `aplicabilidad` de `paquete.json` y en el dialecto
Datalog estratificado del motor. Antes vivian en codigo Go, y eso hacia falso el
invariante de que una norma es un fichero de datos.

Son 29 reglas, cada una con su articulo: ambito (art. 2.1 y 2.3), categoria por
agregacion del maximo de las dimensiones (anexo I, apartados 2 y 3), auditoria
bienal del art. 31.1 para MEDIA y ALTA, autoevaluacion para BASICA con sus
publicidades (ITS de Conformidad III.2, III.3, IV.3 y V.3), INES anual, datos
personales (art. 3.2 y mp.info.1), externalizacion (art. 13.5, art. 16.2 y
op.ext.1 a 3) y nube (op.nub.1).

Se ejecutan de verdad contra el motor en `aplicabilidad_corpus_test.go`, con las
dos direcciones comprobadas: lo que TIENE que aplicarle a un sujeto y lo que NO
puede aplicarle, con el articulo de cada exclusion.

### Lo que falta, dicho aqui y no escondido

La regla de la categoria por agregacion es correcta y esta declarada, pero los
hechos que consume (`maneja`, `nivel_dimension`) **no los recoge ninguna
pregunta de este paquete**: el modelo de entidades solo tiene `sistema`, y para
calcular la categoria del anexo I hacen falta `informacion` y `servicio` con el
nivel de cada dimension sobre cada una.

Mientras eso no exista, en el producto la categoria **se declara** y no se
calcula. La regla se ejerce en el test, no en la interfaz. Esta apuntado en
`docs/pendientes.md`.
