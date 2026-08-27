# ai-act: Reglamento (UE) 2024/1689 de inteligencia artificial

Fuente: EUR-Lex, ELI `reg/2024/1689/oj`. Texto reproducido al amparo de la
Decisión 2011/833/UE, citando la fuente. Solo es auténtico el texto publicado en
el Diario Oficial.

## Qué obliga hoy, 27-08-2026

**El artículo 50 está vigente desde el 2 de agosto de 2026, y no lo ha movido
nadie.** Es la fecha general del artículo 113, y el capítulo IV no aparece entre
sus excepciones. Alcanza a casi todo el mundo, porque no depende de que el
sistema sea de alto riesgo: basta con que interactúe con personas, genere
contenido sintético, reconozca emociones o publique una ultrasuplantación.

Las cuatro obligaciones del artículo 50 **no son todas del mismo obligado**, y
esa es la diferencia entre enseñarle a alguien cuatro obligaciones o dos:

| Apartado | Quién | Qué |
|---|---|---|
| 50.1 | proveedor | que la persona sepa que habla con una IA |
| 50.2 | proveedor | marcar el contenido sintético en formato legible por máquina |
| 50.3 | responsable del despliegue | informar a quien queda expuesto a reconocimiento de emociones o categorización biométrica |
| 50.4 | responsable del despliegue | hacer público que un contenido es una ultrasuplantación, y que un texto de interés público está generado |

El paquete lo reparte con reglas de aplicabilidad, no con código, y las dos
direcciones se comprueban contra el motor en
`aplicabilidad_corpus_test.go`.

**El reloj del artículo 50 es el 50.5**, y tiene una forma que no había en el
corpus: la información se facilita *a más tardar con ocasión de la primera
interacción o exposición*. El límite no es una duración desde un suceso, **es el
suceso**. Se declara como plazo de `P0D` con cierre exacto, y el efecto práctico
es que la fecha límite coincide con el lanzamiento: la información tiene que
estar puesta antes, no después. Si el cierre se dejara en automático, un plazo
expresado en días vencería a fin de día y el producto regalaría casi
veinticuatro horas que el artículo no da. Hay un caso dorado para eso.

## El artículo 73, y por qué su vigencia lleva tres lecturas

El artículo 73 (notificación de incidentes graves) es de la familia de
notificación escalonada: **tres límites distintos sobre el mismo disparador**, y
lo que decide cuál rige es cómo clasifica el incidente el propio obligado.

| Clasificación | Límite | Artículo |
|---|---|---|
| general | 15 días | 73.2 |
| infracción generalizada, o incidente del art. 3.49.b | 2 días | 73.3 |
| fallecimiento de una persona | 10 días | 73.4 |

Los apartados 3 y 4 empiezan con *no obstante lo dispuesto en el apartado 2*, así
que **desplazan** al plazo general: no conviven con él. El paquete lo expresa
dando clase a los tres hitos, y hay un test que comprueba la dirección que un
caso dorado no sabe expresar, que es la de lo que NO tiene que salir.

Y sin clasificar, el motor no se calla ni se inventa una fecha: dice que falta un
dato que pone el obligado.

### La vigencia, que es lo que hay que mirar antes de presupuestar nada

El paquete declara `desde: 2026-08-02`, que es lo que dice el DOUE, y lleva
**tres lecturas divergentes** con su cita:

1. **`anexo-i-art-113-c`, 02-08-2027.** No es una discrepancia, es el propio
   artículo 113, letra c: el artículo 6.1 y las obligaciones correspondientes se
   aplican desde esa fecha. Si tu sistema es de alto riesgo por la vía del anexo
   I (producto sujeto a legislación de armonización), esa es tu fecha.
2. **`omnibus-digital-anexo-iii`, 02-12-2027.** Acuerdo político sobre el paquete
   ómnibus digital, mayo de 2026, sobre la propuesta de la Comisión de noviembre
   de 2025. **No está publicado en el DOUE, así que no vincula.**
3. **`omnibus-digital-anexo-i`, 02-08-2028.** Lo mismo para el anexo I.

**Lo que manda es lo declarado, siempre.** Una lectura divergente no cambia
nunca lo que el motor calcula ni lo que aparece como exigible: se enseña al lado,
con su cita, y quien decide es quien lee. Si fuera al revés, un acuerdo sin
publicar se habría convertido en derecho aplicado dentro de un producto de
cumplimiento.

Cuando se publique, la vigilancia normativa (`ingestanorma -historial`) lo verá
en EUR-Lex y entonces la fecha se mueve donde tiene que estar: al campo `desde`.

## Qué NO hace este paquete

- **No cubre el capítulo III entero** (los requisitos de los sistemas de alto
  riesgo: gestión de riesgos, datos, documentación técnica, registros,
  transparencia hacia el responsable del despliegue, supervisión humana,
  exactitud y ciberseguridad). El censo cuenta 26 obligaciones con reloj en el
  reglamento y aquí hay seis.
- **No cubre la retención documental** (arts. 18.1, 19.1, 22.3, 23.5, 26.6, 47.1
  y 54.3: diez años y seis meses). Es la familia E del censo y necesita la
  primitiva del máximo de dos duraciones.
- **No cubre los modelos de uso general** (arts. 51 a 56, con el plazo de dos
  semanas del art. 52.1).
- **No cubre las prohibiciones del artículo 5**, vigentes desde el 02-02-2025,
  que no son obligaciones con reloj sino un límite absoluto.
- **No dice si tu sistema es de alto riesgo.** Esa clasificación la haces tú, con
  el artículo 6 y los anexos I y III delante; el paquete la toma como dato y de
  ella deduce fechas.

## Comprobado

Nueve casos dorados derivados del texto, ejecutados contra el motor en cada
ejecución de `./comprobar.sh`. El detalle de cómo se llega a cada fecha está en
`COMPUTO.md`.
