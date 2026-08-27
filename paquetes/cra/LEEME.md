# cra: Reglamento (UE) 2024/2847 de ciberresiliencia

Fuente: EUR-Lex, ELI `reg/2024/2847/oj`. Texto reproducido al amparo de la
Decisión 2011/833/UE, citando la fuente.

## Qué obliga y desde cuándo

Las obligaciones de notificación del artículo 14 **se aplican desde el 11 de
septiembre de 2026** (artículo 71.2). El resto del reglamento llega el 11 de
diciembre de 2027. El paquete declara esa fecha en cada obligación, no en la
cabecera: la cabecera lleva la entrada en vigor (10-12-2024), que es otra cosa.

## Dos cadenas de notificación, no una

El artículo 14 tiene **dos** cadenas que se parecen mucho y no son la misma, y
confundirlas es el error caro:

| | Vulnerabilidad aprovechada activamente (14.1 y 14.2) | Incidente grave de seguridad del producto (14.3 y 14.4) |
|---|---|---|
| Disparador | conocimiento de la vulnerabilidad | conocimiento del incidente |
| Alerta temprana | 24 h | 24 h |
| Notificación | 72 h **desde el conocimiento** | 72 h **desde el conocimiento** |
| Informe final | 14 días **desde que hay medida correctora** | 1 mes **desde que se presentó la notificación** |

Las dos primeras filas coinciden en el número y no en el hecho que las dispara.
Las dos últimas no coinciden en nada, y ahí está lo que hay que mirar:

- **El informe final de la vulnerabilidad NO cuenta desde el conocimiento.** El
  artículo 14.2.c dice *a más tardar catorce días después de que se disponga de
  una medida correctora o paliativa*. Contarlo desde el conocimiento produce una
  fecha que puede estar vencida antes de que exista el parche. Hay un caso
  dorado que mide exactamente esa diferencia.
- **El informe final del incidente cuenta desde la PRESENTACIÓN de la
  notificación**, no desde el incidente. Un fabricante que presenta la
  notificación el último día se lleva un mes entero a partir de ahí.

## Lo que este paquete NO hace

- **No cubre la retención documental del CRA** (art. 13.8, 13.9, 13.13, 19.6 y
  los cinco puntos del anexo VIII: diez años **o el período de soporte si es más
  largo**). Es la familia E del censo y necesita la primitiva del máximo de dos
  duraciones, que ya existe en el motor (`ventana.Maximo`) pero todavía no está
  expresada en el formato de corpus.
- **No cubre el disparador por cambio sustancial** (art. 22).
- **No decide si tu producto es "producto con elementos digitales"** ni si es
  importante o crítico (anexos III y IV). Esa clasificación la haces tú.
- **No cubre las obligaciones del importador ni del distribuidor** (arts. 19 y
  20), que tienen sus propios plazos.

## Comprobado

Diez casos dorados derivados del texto. El detalle de cada fecha, en
`COMPUTO.md`.
