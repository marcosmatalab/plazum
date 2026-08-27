# mdr: de dónde sale cada fecha

## 1. Régimen

Plazos en **días**, así que rige el Reglamento (CEE, Euratom) 1182/71 entero: el
día del suceso no se computa (art. 3.1), el plazo termina al expirar la última
hora del último día (art. 3.2.b) y se traslada al hábil siguiente si cae en
inhábil (art. 3.4). Cómputo natural, cierre a fin de día, traslado al hábil
siguiente.

**Aviso de alcance**: el calendario del corpus solo conoce fines de semana. Los
feriados nacionales del Estado miembro de la autoridad competente no están, y
cuando entren habrá que revisar estos dorados uno a uno.

## 2. Los tres límites y quién los elige

Disparador único: `conocimiento_incidente_grave`.

| Hito | Límite | Clase (hecho) | Artículo |
|---|---|---|---|
| `notificacion_general` | P15D | `incidente_general` | 87.3 |
| `notificacion_amenaza_salud_publica` | P2D | `amenaza_grave_salud_publica` | 87.4 |
| `notificacion_muerte_o_deterioro` | P10D | `muerte_o_deterioro_grave` | 87.5 |

**Los tres llevan clase, el general incluido.** Si el general no la llevara,
regiría siempre y un incidente con muerte produciría dos fechas para la misma
obligación. Los cuatro casos dorados de este paquete seguirían en verde con ese
fallo puesto, porque cada dorado mira su hito: lo que lo caza es
`relojes_corpus_test.go`, que comprueba la dirección contraria.

**El 87.2 no es un plazo.** Dice que *como norma general, el plazo de la
notificación dependerá de la gravedad*, que es la regla de la que salen los tres
apartados siguientes, no un cuarto plazo.

**El 87.3 exige además notificar "inmediatamente" después de establecer la
causalidad.** Los quince días son el techo, no el objetivo, y lo mismo pasa con
los otros dos. El paquete calcula el techo, que es lo computable, y las notas de
los hitos dicen lo otro.

## 3. Las cuentas de los dorados

- **General.** Conocimiento el martes 1 de septiembre de 2026. Quince días sin
  contar el día del suceso: miércoles 16, hábil.
- **Amenaza para la salud pública.** Mismo conocimiento, dos días: jueves 3.
- **Muerte.** Conocimiento el sábado 5 de septiembre. Diez días: martes 15.
- **Reclasificación.** General el día 1 a las 15:00, muerte constatada el día 3 a
  las 11:00. Manda la más reciente, así que rige el 87.5: diez días **desde el
  conocimiento del incidente** (el día 1), no desde la reclasificación. Vence el
  11, no el 13. Un fabricante que siguiera viendo quince días vería el 16.
