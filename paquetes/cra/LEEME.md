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

## Cuatro papeles, y el tuyo decide qué ves

El reglamento no obliga igual a todo el mundo, y el paquete lo declara con
`papel_cra`. Si no dices cuál eres, no se enciende nada.

| papel | qué ve |
|---|---|
| `fabricante` | las dos cadenas del art. 14, el informe provisional del 14.6, el aviso a usuarios del 14.8, las retenciones de los arts. 13.9, 13.13 y 13.18 y los cinco módulos del anexo VIII, las medidas correctoras del 13.21 y del 57.2, el aviso de fin de soporte del 13.19 y el examen periódico del anexo I |
| `importador` | la retención del art. 19.6 y los tres deberes del art. 19.5 (medidas correctoras, aviso al fabricante, aviso a las autoridades) |
| `distribuidor` | los tres deberes del art. 20.4, con la misma forma que los del importador |
| `representante_autorizado` | la retención del art. 18.3, letra a) |

## Lo que este paquete NO hace

- **No decide si tu producto es "producto con elementos digitales"** ni si es
  importante o crítico (anexos III y IV). Esa clasificación la haces tú.
- **No lleva el aviso previo al cese de actividades** (art. 13.23), que obliga a
  informar a las autoridades y a los usuarios *antes de que dicho cese surta
  efecto*. Es un plazo que corre hacia atrás desde una fecha que eliges tú, o sea
  la primitiva `preaviso` del motor, y hoy un paquete no puede declararla: el
  ejecutor de casos dorados exige el hecho del `disparador` y el linter le
  prohíbe a un `preaviso` tener disparador. Está medido y escrito, con la línea
  exacta que lo abre, en `docs/hallazgos-cra-nis2.md`. **1 reloj esperando.**
- **No lleva otros once relojes por evento sin cifra** de los arts. 13.6, 13.22,
  18.3, 19.3, 19.7, 19.8, 20.3, 20.5 y 20.6: avisar al mantenedor de un
  componente vulnerable, atender un requerimiento motivado de la autoridad, e
  informar del cese de actividades del fabricante cuando quien se entera es el
  importador o el distribuidor. Están contados uno a uno en
  `docs/hallazgos-cra-nis2.md`.
- **No lleva los deberes permanentes sin cifra** (identificación del producto,
  datos de contacto, punto de contacto único, documentación técnica, copia de la
  declaración UE, la fecha de fin de soporte en el momento de la compra y sus
  gemelos de los arts. 19 y 20). Son deberes que no vencen y que se escriben como
  `continua`, y van en una sola pasada para que no salgan desiguales.
  **24 identificados en los arts. 13, 18, 19 y 20**, contados en
  `docs/hallazgos-cra-nis2.md`.
- **No cubre el art. 22** a propósito, y no es un hueco: los apartados 1 y 2
  dicen que se considerará fabricante a quien haga una modificación sustancial y
  comercialice el producto. Eso cambia **quién** está obligado, no **cuándo**.

## Comprobado

64 casos dorados derivados del texto, sobre 29 hitos de 24 obligaciones. El
detalle de cada fecha, en `COMPUTO.md`.
