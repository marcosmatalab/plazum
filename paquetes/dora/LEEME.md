# dora: Reglamento (UE) 2022/2554, con el Delegado 2025/301

Fuente: EUR-Lex. Los plazos no están en DORA: los fija el **Reglamento Delegado
(UE) 2025/301**, artículo 5, y por eso el paquete cita los dos.

## Lo que tiene de raro, y por qué merecía escribirse

**El reloj arranca en la CLASIFICACIÓN, no en el conocimiento.** El artículo
5.1.a del Delegado da cuatro horas *a partir de la clasificación del incidente
relacionado con las TIC como grave* **y a más tardar veinticuatro horas después**
del momento en que la entidad tuvo conocimiento.

Son **dos plazos que vinculan a la vez, desde dos hechos distintos**. Hay que
cumplir los dos, así que la fecha útil es una sola: la primera de las dos.
Enseñar las dos y dejar que el operador elija sería dejarle una cuenta que puede
hacer mal el día que más prisa tiene.

Y el artículo 5.2 añade el remate: si la entidad **no clasificó** el incidente
como grave dentro de esas 24 horas pero lo clasifica después, la notificación
inicial se presenta en 4 horas desde la clasificación. O sea que el tope de 24
horas **caduca**. El paquete lo declara así, con su cita, porque en este motor
un tope vincula siempre salvo que se pida lo contrario.

## La cadena

| Hito | Límite | Desde | Artículo |
|---|---|---|---|
| `notificacion_inicial` | 4 h, con tope de 24 h desde el conocimiento | clasificación como grave | 5.1.a y 5.2 |
| `informe_intermedio` | 72 h | presentación de la notificación inicial | 5.1.b |
| `informe_final` | 1 mes | presentación del informe intermedio | 5.1.c |

El informe intermedio hay que presentarlo **aunque la situación no haya
cambiado** (5.1.b lo dice expresamente), y si hay intermedios actualizados el
mes del informe final cuenta desde el último. El paquete cuenta desde el primero
y lo dice en la nota del hito: si hay actualizados, hay que registrarlos.

## Las nueve cadencias con número (29-08-2026)

DORA da número a nueve deberes recurrentes, y **los nueve dicen «al menos»**: son suelos legales, sólo se pueden apretar.

| art. | cadencia | qué |
|---|---|---|
| 6.5 | P12M | documentar y revisar el marco de gestión del riesgo TIC |
| 8.1 | P12M | revisar la idoneidad de la clasificación de funciones y activos |
| 8.2 | P12M | revisar los escenarios de riesgo |
| 8.7 | P12M | evaluar el riesgo TIC de los sistemas heredados |
| 11.6.a | P12M | probar los planes de continuidad y de respuesta y recuperación |
| 13.5 | P12M | informe del directivo de TIC al órgano de dirección |
| 24.6 | P12M | pruebas de los sistemas que sustentan funciones esenciales |
| **26.1** | **P36M** | pruebas avanzadas de penetración basadas en amenazas |
| 28.3 | P12M | comunicación anual a la autoridad sobre acuerdos con terceros TIC |

**El art. 26.1 es la única cadencia de este corpus que una autoridad puede ALARGAR.** El propio apartado dice que *«la autoridad competente podrá, en caso necesario, solicitar a la entidad financiera que reduzca o aumente esta frecuencia»*. Sigue siendo `suelo_legal` — tres años es el suelo mientras nadie diga otra cosa —, y que la excepción exista está escrito en la cita del intervalo, no en una nota al pie.

### Dos exclusiones, y no son la misma

DORA excluye por **tamaño** y por **régimen**, y es la primera vez que este corpus usa la negación del dialecto sobre una norma real:

- **Microempresas** (art. 3.60: menos de diez personas y ≤ 2 M EUR de volumen o balance, y que no sea centro de negociación, ECC, registro de operaciones o DCV). Los arts. **6.5, 8.7, 24.6 y 26.1** no les alcanzan.
- **Marco simplificado** del art. 16.1 párrafo primero. Sólo el art. **26.1** las excluye, *además* de a las microempresas.

Las dos se comprueban **en las dos direcciones** en `TestDoraExcluyeALaMicroempresaYAlMarcoSimplificado`: que al banco le alcancen y que a la microempresa no. Una exclusión mal escrita no da error: da obligaciones de más, y ésas las paga el cliente sin deberlas.

### Dos reaperturas por evento

Los arts. **6.5** (incidente grave relacionado con las TIC) y **11.6.a** (cambio sustancial en los sistemas de TIC que sustenten funciones esenciales o importantes) piden el trabajo *«así como cuando...»*. Van como `reabre_por`, no como obligación aparte: es un segundo disparador del mismo deber (`docs/decisiones.md` D-16).

### El reloj que NO se ha escrito, y por qué

El art. 6.5 dice *«al menos una vez al año, **o periódicamente en el caso de las microempresas**»*. Para una microempresa la norma **no da número**, así que su reloj sería `propuesto`: un número de plazum con su justificación y sus instrucciones de uso.

No está escrito, y la obligación anual **no les alcanza** — la regla lo dice con el artículo. Es una laguna deliberada y acotada: afecta a un solo apartado y a un solo tipo de entidad, y está aquí en vez de en el silencio, que es lo que la separa de un olvido.

## Lo que este paquete NO hace

- **No clasifica el incidente.** Los criterios están en el artículo 18 de DORA y
  en el Delegado 2024/1772; la clasificación la hace la entidad y el paquete la
  toma como hecho. De ella depende **cuándo arranca el reloj**, así que es el
  dato más caro de tener tarde.
- **No cubre el artículo 5.4 del Delegado** (presentar antes del mediodía del
  siguiente día hábil cuando el plazo cae en fin de semana o festivo). Es una
  **facultad**, no una obligación, y los apartados 5.5 y 5.6 se la quitan
  precisamente a las entidades más grandes. Un plazo condicionado a que la
  autoridad no haya decidido otra cosa no se puede calcular sin saber esa
  decisión. Régimen declarado: `traslado: ninguno`.
- **No cubre las 21 cadencias periódicas de DORA** (familia B del censo) ni la
  auditoría de 36 meses del art. 26.1 (familia C).
- **No cubre la notificación voluntaria de ciberamenazas** (art. 19.2), que es
  voluntaria y no tiene plazo.

## Comprobado

Cinco casos dorados, incluidos los tres que separan el tope del límite
principal. Detalle en `COMPUTO.md`.
