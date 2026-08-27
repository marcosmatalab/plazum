# ai-act: de dónde sale cada fecha

Este documento es la fuente que citan los casos dorados de `pruebas/`. No
describe la implementación: describe la lectura del texto. Si el motor y esto
discrepan, gana esto.

## 1. El régimen de cómputo de los plazos en días del artículo 73

El Reglamento (UE) 2024/1689 no trae reglas propias de cómputo, así que rige el
**Reglamento (CEE, Euratom) 1182/71**, por el que se determinan las normas
aplicables a los plazos, fechas y términos. Tres reglas suyas se aplican aquí, y
las tres se declaran en el paquete:

1. **Art. 3.1**: el día en cuyo curso se produce el suceso **no se computa**
   dentro del plazo. Consecuencia práctica: la hora a la que se conoce el
   incidente no acorta el plazo. Conocerlo a las 00:05 y conocerlo a las 23:30
   del mismo día dan la misma fecha límite. Hay un caso dorado para esto, porque
   es la diferencia con un plazo en horas como el del artículo 14.2.a del CRA,
   donde la hora sí manda.
2. **Art. 3.2, letra b**: el plazo expresado en días comienza al principio de la
   primera hora del primer día y **termina al expirar la última hora del último
   día**. De ahí `cierre: fin_de_dia`.
3. **Art. 3.4**: si el último día cae en **sábado, domingo o día feriado**, el
   plazo termina al expirar la última hora del día hábil siguiente. De ahí
   `traslado: siguiente_habil`.

**Aviso de alcance, y es una limitación real de hoy**: el calendario que usa el
ejecutor de dorados solo conoce fines de semana. Los días feriados de las
instituciones de la Unión no están en el corpus, así que hoy el traslado del art.
3.4 solo se calcula por sábado y domingo. Cuando entre el calendario de la Unión,
las fechas que caigan en feriado se moverán y los dorados de este paquete habrá
que revisarlos uno a uno. Está dicho aquí para que no sea una sorpresa.

**Lectura declarada, y se dice en voz alta**: el artículo 1 del Reglamento
1182/71 se refiere a los actos del Consejo y de la Comisión. Se aplica aquí a un
reglamento del Parlamento Europeo y del Consejo porque es el régimen general de
cómputo del Derecho de la Unión y así lo trata la práctica. Es una lectura, no una
cita literal, y por eso está escrita aquí y no escondida en el código.

## 2. Artículo 73: los tres límites y quién los elige

| Hito del paquete | Límite | Clase (hecho) | Artículo |
|---|---|---|---|
| `notificacion_general` | P15D | `incidente_general` | 73.2 |
| `notificacion_infraccion_generalizada` | P2D | `incidente_infraccion_generalizada` | 73.3 |
| `notificacion_fallecimiento` | P10D | `incidente_fallecimiento` | 73.4 |

Disparador único: `conocimiento_incidente_grave`. Los tres cuentan desde ahí; no
hay encadenamiento (a diferencia de la tabla 3 del RD 43/2021, donde la
intermedia cuenta desde la remisión de la inicial).

**Los tres hitos llevan clase, incluido el general, y eso no es simetría
decorativa.** Los apartados 3 y 4 empiezan con *no obstante lo dispuesto en el
apartado 2*: desplazan al plazo general, no se suman a él. Si el hito general no
llevara clase, regiría siempre, y un incidente con fallecimiento produciría dos
fechas para la misma obligación. Los nueve casos dorados de este paquete siguen
en verde con ese fallo puesto, porque cada dorado mira su hito; lo caza
`relojes_corpus_test.go`.

**Sin clasificar no se calla.** Con el incidente conocido y sin clasificar, los
tres hitos salen como pendientes de un hecho, diciendo qué falta. Una lista vacía
se leería como "nada que hacer" cuando lo que pasa es que falta un dato que pone
el obligado.

### Las cuentas de los casos dorados

- **Caso general.** Conocimiento el martes 1 de septiembre de 2026. Quince días
  sin contar el día del suceso: el último día es el miércoles 16, hábil. Vence el
  16 a las 23:59:59.
- **Traslado.** Conocimiento el viernes 4 de septiembre de 2026. El día quince es
  el **sábado 19**, así que el plazo termina al expirar el lunes 21.
- **Infracción generalizada.** Mismo conocimiento que el caso general, otra
  clasificación: dos días, jueves 3, hábil.
- **Fallecimiento.** Diez días desde el mismo conocimiento: viernes 11, hábil.
- **La hora no acorta.** Conocimiento a las 23:30 del 1 de septiembre: misma
  fecha límite que el caso general, el 16.
- **Reclasificación.** General el día 1 a las 15:00 y fallecimiento el día 2 a
  las 09:00. Manda la clasificación más reciente, así que rige el 73.4: diez días
  desde el **conocimiento del incidente**, no desde la reclasificación. Vence el
  11, no el 12.

## 3. Artículo 50.5: el plazo cuyo límite es su propio disparador

> La información [...] se facilitará [...] **a más tardar con ocasión de la
> primera interacción o exposición**.

No hay duración que sumar. El vencimiento **es** el instante del disparador, y se
declara como `P0D` con `cierre: exacto`.

Las dos decisiones de lectura, dichas en voz alta:

- **Por qué no `indeterminado`.** "Indeterminado" es lo que usa el paquete
  `nis1-es` para la notificación inicial de la tabla 3, que dice "Inmediata": hay
  obligación y no hay número. Aquí sí hay momento, y es exacto: la primera
  interacción. Decir "sin plazo legal" sería perder una fecha que la norma da.
- **Por qué `exacto` y no automático.** El cierre automático manda a fin de día
  todo plazo expresado en días o meses, y este lo está. Con cierre automático,
  una primera interacción a las 00:05 vencería a las 23:59:59 del mismo día: casi
  veinticuatro horas de margen que el artículo 50.5 no da. Hay un caso dorado
  que fija exactamente esa hora.

El disparador se llama `primera_interaccion_o_exposicion` porque el apartado 5
cubre los cuatro anteriores: la interacción directa del 50.1, la salida marcada
del 50.2, la exposición al reconocimiento de emociones del 50.3 y la publicación
de la ultrasuplantación del 50.4.

## 4. El artículo 111.4: la primera fecha que fija la norma

Es la primera obligación del corpus cuya fecha **no se calcula**: está escrita en
el texto. Primitiva `puntual`, campo `en` con el instante completo.

> [...] adoptarán las medidas necesarias para cumplir lo dispuesto en el artículo
> 50, apartado 2, **a más tardar el 2 de diciembre de 2026**.

Dos decisiones de lectura, dichas en voz alta:

- **Por qué el instante lleva la hora dentro** (`2026-12-02T23:59:59Z`). Una
  primitiva `puntual` no tiene régimen y por tanto no sabe cerrar el día: si el
  paquete escribiera solo la fecha, vencería a las 00:00 y el obligado perdería
  un día entero. La hora se escribe en el dato, y el linter rechaza una `puntual`
  sin `en`.
- **Por qué la fecha de comercialización no es el disparador.** El apartado usa
  «introducidos en el mercado antes del 2 de agosto de 2026» para decidir **a
  quién** alcanza, no **cuándo** vence. Un proveedor que comercializó en 2024 y
  otro que comercializó el 1 de agosto de 2026 tienen la misma fecha límite. Hay
  un caso dorado con cada uno, y un tercero que mete un hecho llamado
  `conocimiento` para fijar que la primitiva **no lee los hechos**: si alguien la
  convirtiera en un plazo contado desde un disparador, ese dorado se pone rojo.

## 5. Vigencias

| Obligación | `desde` | De dónde |
|---|---|---|
| las cinco del art. 50 | 2026-08-02 | art. 113, párrafo segundo; el capítulo IV no está entre las excepciones del párrafo tercero, y el ómnibus no lo menciona |
| art. 73 y art. 73.6 | 2026-08-02 | art. 113, párrafo segundo. El ómnibus solo cambia las letras a y c del párrafo TERCERO, y el art. 73 es del capítulo IX |
| art. 111.4 | 2026-07-24 | entrada en vigor del Reglamento (UE) 2026/1744, con el aviso de precisión de `LEEME.md` |

Y las **dos** lecturas divergentes del artículo 73, con el detalle en `LEEME.md`:
el 02-12-2027 y el 02-08-2028 del capítulo III, secciones 1, 2 y 3, que el
Reglamento (UE) 2026/1744 puso en la letra c del párrafo tercero del artículo
113. **Están publicadas y vinculan**; lo que se discute con ellas no es si son
derecho, sino si un artículo del capítulo IX obliga antes de que exista la
clasificación del capítulo III que le da destinatarios.

La entrada en vigor del reglamento (01-08-2024, veinte días después de su
publicación) es la vigencia de la **cabecera** del paquete, no la de ninguna
obligación. Son cosas distintas y las dos están.
