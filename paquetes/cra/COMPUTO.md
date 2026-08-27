# cra: de dónde sale cada fecha

Fuente de los casos dorados de `pruebas/`. Describe la lectura del texto, no la
implementación. Si el motor y esto discrepan, gana esto.

## 1. Dos regímenes dentro de la misma obligación

El artículo 14 mezcla **horas** (24 y 72) con **días** (14) y **meses** (1), y
el régimen de cómputo no es el mismo para las tres cosas. Lo decide el
Reglamento (CEE, Euratom) 1182/71:

- **Horas**: de hora a hora. El artículo 3.4 traslada al hábil siguiente el
  vencimiento que cae en inhábil *expresado de cualquier modo, salvo en horas*,
  así que a los plazos en horas **no les alcanza**. Régimen: cierre exacto,
  traslado ninguno.
- **Días y meses**: el día del suceso no se computa (art. 3.1), el plazo termina
  al expirar la última hora del último día (art. 3.2.b) y se traslada al hábil
  siguiente si cae en inhábil (art. 3.4). Régimen: cierre a fin de día, traslado
  al hábil siguiente.

Por eso los hitos de este paquete llevan **régimen propio**: los de horas heredan
el de la obligación y los de días y meses declaran el suyo. Antes de esto había
que elegir entre partir la obligación en dos (y perder el encadenamiento del
informe final) o darle a los meses el régimen de las horas, que produce una fecha
más temprana que la legal.

**Aviso de alcance**: el calendario del corpus solo conoce fines de semana. Los
días feriados de las instituciones de la Unión no están, así que hoy el traslado
del art. 3.4 solo se calcula por sábado y domingo.

## 2. Vulnerabilidad aprovechada activamente (art. 14.1 y 14.2)

Disparador: `conocimiento_vulnerabilidad`.

| Hito | Límite | Desde | Artículo |
|---|---|---|---|
| `alerta_temprana` | 24 h | disparador | 14.2.a |
| `notificacion_vulnerabilidad` | 72 h | disparador | 14.2.b |
| `medida_correctora_disponible` | sin plazo legal | disparador | (no es del art. 14) |
| `informe_final` | 14 días | cumplimiento de `medida_correctora_disponible` | 14.2.c |

**Por qué `medida_correctora_disponible` es un hito sin número.** El artículo
14.2.c cuelga el informe final de un hecho —que exista la medida correctora— que
el propio artículo 14 no fecha. El artículo 13.8 obliga a subsanar *sin demora*,
sin número. Así que el hito existe (hay obligación), sale como *sin plazo legal*
y el motor mide el tiempo transcurrido. Inventar aquí un plazo sería inventar
derecho; borrar el hito dejaría el informe final sin nada de donde colgar.

**Las cuentas de los dorados**:

- Alerta temprana: conocimiento el 17 de septiembre de 2026 a las 20:00, vence
  el 18 a las 20:00. Y los dos bordes clásicos: fin de mes y cambio de año, que
  no mueven nada porque el plazo está en horas.
- Notificación: el fabricante remitió la alerta 23 horas después, y eso **no
  mueve** el plazo de las 72 horas, que cuenta desde el mismo conocimiento.
  Encadenarla a la alerta le daría casi un día de más.
- Informe final: conocimiento el 17 de septiembre, medida correctora el 5 de
  octubre. Catorce días desde el 5 son el 19 de octubre (lunes). Contado desde
  el conocimiento habría vencido el 1 de octubre, **cuatro días antes de que
  existiera el parche**.
- Y el control negativo del traslado: con la medida disponible el viernes 2 de
  octubre, el día catorce es el viernes 16, hábil, y el traslado no se dispara.

## 3. Incidente grave de seguridad del producto (art. 14.3 y 14.4)

Disparador: `conocimiento_incidente_producto`.

| Hito | Límite | Desde | Artículo |
|---|---|---|---|
| `alerta_temprana_incidente` | 24 h | disparador | 14.4.a |
| `notificacion_incidente` | 72 h | disparador | 14.4.b |
| `informe_final_incidente` | 1 mes | cumplimiento de `notificacion_incidente` | 14.4.c |

**Las cuentas**:

- Informe final: notificación presentada el 20 de septiembre de 2026 a las
  18:00. Un mes de fecha a fecha es el 20 de octubre (martes, hábil), no el 17
  de octubre que saldría contando desde el conocimiento.
- Traslado: notificación presentada el viernes 18 de septiembre. Un mes cae el
  domingo 18 de octubre, así que el plazo termina al expirar el **lunes 19**.
