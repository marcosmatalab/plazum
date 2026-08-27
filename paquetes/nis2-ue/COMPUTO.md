# nis2-ue: de dónde sale cada fecha

## 1. Aviso antes que nada: esto es una DIRECTIVA

La Directiva (UE) 2022/2555 **no vincula por sí misma a ninguna empresa**. Obliga
a los Estados miembros a transponerla, y lo que le aplica a una organización son
las normas nacionales de transposición. En España, a día de hoy, no hay
transposición publicada, y lo que vincula sigue siendo el RD 43/2021 (paquete
`nis1-es`).

Este paquete se escribe igual por dos motivos: los plazos del artículo 23.4 son
los que las transposiciones van a llevar dentro, y en otros Estados miembros ya
vinculan. Pero **no se le debe enseñar a una empresa española como si fuera
exigible hoy**. Su `LEEME.md` lo dice y las reglas de aplicabilidad tendrán que
decirlo también cuando el paquete las tenga.

## 2. La cadena del artículo 23.4

Disparador: `constancia_incidente_significativo` (la directiva dice *constancia*,
no *conocimiento*, y no es un matiz de estilo: es el momento en que la entidad
tiene elementos para saber que el incidente es significativo).

| Hito | Límite | Clase | Artículo |
|---|---|---|---|
| `alerta_temprana` | 24 h | (todos) | 23.4.a |
| `notificacion_incidente` | 72 h | entidad esencial o importante | 23.4.b |
| `notificacion_incidente_prestador_de_confianza` | 24 h | prestador de servicios de confianza | 23.4, párrafo segundo |
| `informe_final` | 1 mes desde la presentación de la notificación | entidad esencial o importante | 23.4.d |
| `informe_final_prestador_de_confianza` | 1 mes desde la presentación de la suya | prestador de servicios de confianza | 23.4.d |

**El párrafo segundo del artículo 23.4 baja las 72 horas a 24** para un prestador
de servicios de confianza, respecto de los incidentes que afecten a la prestación
de sus servicios de confianza. Se expresa con la misma primitiva que usa el nivel
del RD 43/2021: **el límite lo decide una categoría que declara el propio
obligado**. Las dos notificaciones no conviven: o eres prestador de confianza o
no lo eres.

La alerta temprana **no** lleva clase: son 24 horas para todos, así que sale con
fecha aunque no se haya declarado el tipo de entidad. Lo que se queda pendiente
es lo que de verdad depende del tipo. Hay un caso dorado para eso, porque una
lista vacía se leería como "nada que hacer" y elegir uno de los dos plazos en
silencio sería peor.

## 3. Las cuentas de los dorados

Constancia del incidente: **1 de septiembre de 2026 a las 14:00 UTC**.

- Alerta temprana: 24 horas exactas, el día 2 a las 14:00.
- Notificación (entidad esencial): 72 horas, el día 4 a las 14:00.
- Prestador de confianza: **el mismo hecho, a la misma hora, la mitad de
  tiempo**: el día 2 a las 14:00.
- Informe final: la notificación se presentó el día 3 a las 09:00, así que un mes
  de fecha a fecha cae el 3 de octubre, que es **sábado**, y el artículo 3.4 del
  Reglamento 1182/71 lo lleva al lunes 5. Contado desde la constancia habría
  salido el 1 de octubre.

## 4. Lo que no está

- El **informe intermedio** del artículo 23.4.c, que se presenta *a instancias
  del CSIRT*: no tiene plazo propio y su disparador es una petición externa.
- El **informe de situación** del artículo 23.4.e, para el incidente que sigue en
  curso, con su informe final **un mes desde que se gestionó el incidente**. Es
  una segunda cadena que se solapa con la primera y merece su propia lectura.
- La **comunicación a los destinatarios** de los apartados 1 y 2, que es *sin
  demora indebida* y no lleva número.
