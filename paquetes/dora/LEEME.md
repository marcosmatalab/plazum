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
