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

## El artículo 73, y por qué su vigencia lleva dos lecturas

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

### La vigencia, y la corrección del 27-08-2026

**El ómnibus digital se publicó, y este paquete decía lo contrario.** Es el
**Reglamento (UE) 2026/1744 del Parlamento Europeo y del Consejo, de 8 de julio
de 2026**, CELEX `32026R1744`, que modifica el AI Act en 34 puntos. Durante un
día, este paquete afirmó que las dos fechas del ómnibus «NO VINCULAN» cuando ya
vinculaban. Es el error más caro que puede cometer un producto de cumplimiento y
está corregido; el item de vigilancia `publicacion-del-omnibus-en-el-doue` existe
para que no vuelva a hacer falta que lo cace una persona.

**Lo que el ómnibus cambia, verificado artículo por artículo contra EUR-Lex:**

- **Sustituye la letra c del párrafo tercero del artículo 113** (art. 1, punto
  40, letra b). Donde decía «el artículo 6.1 y las obligaciones correspondientes,
  desde el 02-08-2027» ahora dice: **el capítulo III, secciones 1, 2 y 3** se
  aplica desde el **02-12-2027** para el alto riesgo del artículo 6.2 y el anexo
  III, y desde el **02-08-2028** para el del artículo 6.1 y el anexo I. La fecha
  del 02-08-2027 **ya no existe en la ley**, y por eso ha desaparecido del
  paquete en vez de quedarse como divergencia.
- **No toca el párrafo segundo del artículo 113**, que es el que dice «será
  aplicable a partir del 2 de agosto de 2026». Comprobado: el ómnibus solo
  modifica las letras a y c del párrafo TERCERO.
- **No menciona el capítulo IV** ni una sola vez. El artículo 50 sigue vigente
  desde el 02-08-2026, sin cambios en sus apartados 1 a 5.
- **Añade el artículo 111, apartado 4**, que es una obligación nueva con fecha
  cierta y está en este paquete (ver abajo).
- Añade prohibiciones al artículo 5 aplicables desde el **02-12-2026**, que este
  paquete todavía no transcribe.

**Por qué el artículo 73 sigue con `desde: 2026-08-02`.** El artículo 73 está en
el capítulo IX, y el aplazamiento alcanza al capítulo III. Su fecha de aplicación
no se ha movido: sigue siendo la general del párrafo segundo. Lo que se aplaza es
la **clasificación** que le da destinatarios, y esa es una lectura, no un hecho,
así que viaja como divergencia:

1. **`capitulo-iii-anexo-iii`, 02-12-2027.** Reglamento (UE) 2026/1744, art. 1,
   punto 40, letra b. **Publicado y vinculante.**
2. **`capitulo-iii-anexo-i`, 02-08-2028.** Lo mismo para el anexo I.

**Lo que manda sigue siendo lo declarado.** El artículo 73 está en vigor y el
paquete lo dice; que la clasificación de alto riesgo del capítulo III no se
aplique hasta 2027 o 2028 se enseña al lado, con su cita, y quien decide es quien
lee. Decirle a un cliente que puede ignorar un artículo en vigor porque su
capítulo instrumental está aplazado es una lectura defendible, y por eso está
escrita; no es la que el producto aplica por su cuenta.

### El artículo 111.4: fecha cierta y a tres meses

> Los proveedores de sistemas de IA [...] que generen contenido sintético de
> audio, imagen, vídeo o texto y **se hayan introducido en el mercado antes del 2
> de agosto de 2026** adoptarán las medidas necesarias para cumplir lo dispuesto
> en el artículo 50, apartado 2, **a más tardar el 2 de diciembre de 2026**.

Es la primera obligación del corpus con **fecha fijada por la norma** y no
calculada desde un hecho: primitiva `puntual`. La fecha de comercialización
decide **a quién** alcanza, no **cuándo** vence, y hay un caso dorado que lo fija.

Un aviso de honestidad sobre su vigencia, **corregido el 02-09-2026**: este
párrafo decía que el paquete declaraba `desde: 2026-07-24`, que es la fecha de
**publicación** del ómnibus en el DOUE y no la de entrada en vigor. El artículo 4
del ómnibus dice «a los tres días de su publicación», o sea el **27-07-2026**, y
eso es lo que declara hoy el paquete. Es la conflación que el invariante 10
nombra con estas palabras, «de 8 de julio» y «publicado el 8 de julio» no son lo
mismo, cometida dentro del corpus. La fecha límite, que es lo que importa, no
depende de eso.

## Las pruebas en condiciones reales, y el primer plazo que corre hacia atrás

El artículo 60 deja al proveedor probar un sistema de alto riesgo **en
condiciones reales**, fuera de un espacio controlado, y le pone condiciones. Dos
de ellas son relojes y viven en la misma letra f), que es donde está el detalle
que casi se pierde:

> las pruebas en condiciones reales no duran más de lo necesario para lograr sus
> objetivos y, en cualquier caso, **no más de seis meses**, que podrán
> prorrogarse por un período adicional de seis meses, con sujeción al envío de
> una **notificación previa** por parte del proveedor o proveedor potencial a la
> autoridad de vigilancia del mercado

**Son dos verbos y dos relojes, y el segundo es de una forma que no había en el
corpus.** El primero es un techo, seis meses desde que empiezan las pruebas. El
segundo no cuenta hacia adelante desde nada que te ocurra: cuenta **hacia atrás
desde una fecha que eliges tú**, el día en que quieres que la prórroga surta
efecto, y lo que calcula es hasta cuándo puedes seguir callado. Es la primitiva
`preaviso`, y este es el primer paquete del corpus que la usa.

**Cuánta antelación exige el apartado: no lo dice.** Así que el paquete lo
escribe como lo que es, `antelacion: indeterminado`, y el producto contesta *«la
norma exige preaviso y no fija cuánta antelación»* midiendo el tiempo, en vez de
inventarse un número que ninguna autoridad podría exigirte. La consecuencia
práctica que sí importa, y que hay caso dorado para ella: **si mueves la fecha de
efecto, el aviso se mueve contigo**, y puede quedarse en el pasado.

Con ellos van otros dos actos del mismo artículo:

| art. | qué | reloj |
|---|---|---|
| 60.4.f | terminar las pruebas | seis meses desde que empiezan, sin traslado por fin de semana |
| 60.4.f | avisar antes de prorrogar | cuenta atrás desde la fecha que fijas, sin antelación en la norma |
| 60.8 | notificar la suspensión o la terminación y los resultados | obliga y no lleva número |
| 60.7 | informar de un incidente grave durante las pruebas | **no es una fila nueva**: remite al art. 73, que ya está escrito |

**Si no haces pruebas en condiciones reales, estas filas no te vencen nunca**: el
motor dice que el reloj no ha arrancado porque falta el hecho, y eso no es un
incumplimiento, es un dato que no consta.

## Los dos avisos que se parecen y no son el mismo

El representante autorizado que rompe su mandato tiene que decirlo, y **a quién
lo dice depende de qué represente**:

| art. | quién | a quién |
|---|---|---|
| 22.4 | representante de un proveedor de sistema de **alto riesgo** | autoridad de vigilancia del mercado (y el organismo notificado, cuando proceda) |
| 54.5 | representante de un proveedor de **modelo de uso general** | **Oficina de IA** |

Ninguno de los dos lleva número: uno dice «de inmediato» y el otro
«inmediatamente». Las reglas de aplicabilidad los separan y hay dorados que
comprueban las dos direcciones, porque presentarse ante el órgano equivocado es
de los pocos errores de cumplimiento que no se pueden deshacer.

## Qué NO hace este paquete

- **No cubre el capítulo III entero** (los requisitos de los sistemas de alto
  riesgo: gestión de riesgos, datos, documentación técnica, registros,
  transparencia hacia el responsable del despliegue, supervisión humana,
  exactitud y ciberseguridad).
- **No cubre las prohibiciones del artículo 5**, vigentes desde el 02-02-2025,
  que no son obligaciones con reloj sino un límite absoluto. Y **tampoco las
  nuevas** que el ómnibus añade con aplicación desde el 02-12-2026 (art. 5.1,
  letras b bis y b ter, y apartados 1 bis y 1 ter): eso es autoría pendiente y
  tiene fecha encima.
- **No dice si tu sistema es de alto riesgo.** Esa clasificación la haces tú, con
  el artículo 6 y los anexos I y III delante; el paquete la toma como dato y de
  ella deduce fechas.

## Los dos relojes del alto riesgo (29-08-2026)

Los arts. 9.2 y 72.2 son **la misma pareja vista desde los dos lados**, y el propio Reglamento lo dice: el art. 9.2, letra c), manda evaluar los riesgos *«a partir del análisis de los datos recogidos con el sistema de vigilancia poscomercialización a que se refiere el artículo 72»*.

| art. | primitiva | qué |
|---|---|---|
| **9.2** | `periodica`, **P12M** (`propuesto`) | revisar y actualizar el sistema de gestión de riesgos |
| **72.2** | `continua`, **sin plazo legal** | recopilar, documentar y analizar los datos de funcionamiento |

**El 72.2 no lleva número, y es deliberado.** El apartado dice *«de manera activa y **sistemática**»*, no «periódicamente», y no da cadencia ni plazo. Ponerle un trimestre habría sido inventar un número que el texto no da; dejarlo fuera del corpus, callar un deber que existe. Sale como deber permanente con su motivo, igual que las obligaciones sin número que el corpus ya traía. El porqué completo, en `docs/decisiones.md` D-17.

**Y el 9.2 sí lleva número, puesto por plazum**, porque el apartado dice «periódicas» y nada más. Doce meses porque el propio apartado dice de qué se alimenta la revisión: de los datos que recoge la vigilancia del art. 72, que es **continua**. El intervalo de la revisión es entonces **el tiempo máximo que esos datos pueden acumularse sin que nadie decida nada con ellos**; más allá de un año, la vigilancia recoge para un archivo.

**Y por eso el art. 9.2 comparte sesión con el RGPD y con DORA**: los tres piden verificar la eficacia de lo implantado. `plazum calendario --sentadas` lo dice en una línea, *«3 fechas de 3 marcos»*, con un test que lo exige.

## Comprobado

Medido el 03-09-2026 sobre este mismo directorio: **25 obligaciones, de ellas 20
con reloj, 22 hitos y 58 casos dorados**, todos derivados del texto y ejecutados
contra el motor en cada ejecución de `./comprobar.sh`. Las seis primitivas del
motor que un paquete puede declarar hoy están todas ejercidas aquí (`puntual`,
`periodica`, `continua`, `plazo`, `maximo` y `preaviso`). El detalle de cómo se
llega a cada fecha está en `COMPUTO.md`.

Estos números se cuentan a mano y **ninguna puerta los vigila**: si al leerlos no
cuadran con el directorio, gana el directorio. Los que sí tienen puerta son los
del `README.md` de la raíz y los de `paquetes/CORPUS.md`.

## Lo que el ómnibus cambia y este paquete todavía no recoge

Dicho para que no se pierda, con el punto de modificación de cada cosa:

- **Punto 27**: se inserta un apartado 1 bis que hace que los proveedores bajo
  competencia de la Oficina de IA notifiquen los incidentes graves **a la Oficina
  de IA** y no a la autoridad de vigilancia del mercado, con el artículo 73,
  apartados 2 a 9, aplicable por analogía. Los plazos son los mismos; cambia el
  destinatario, que es dato de la notificación y no del reloj.
- **Punto 20**: nueva redacción del artículo 50.7 (códigos de buenas prácticas).
  Es una obligación de la Comisión, así que queda fuera por la regla del censo.
- **Punto 39, letra a**: nueva redacción del artículo 111.2, con el tope del
  **02-08-2030** para los sistemas de alto riesgo destinados a autoridades
  públicas. Es una segunda fecha cierta y entra cuando se transcriba.
