# psd2-es: Real Decreto-ley 19/2018, servicios de pago

Fuente: BOE, ELI `es/rdl/2018/11/23/19/con`. Texto consolidado, reproducido al
amparo del artículo 13 del TRLPI y de las condiciones de reutilización del BOE.

## Por qué existe este paquete y no basta con `psd2`

La Directiva (UE) 2015/2366 (PSD2) **no vincula por sí misma a ninguna empresa
española**. Lo que vincula es su transposición, que es este real decreto-ley. El
paquete `psd2` seguirá existiendo con la directiva; este trae lo que se le puede
exigir hoy a un proveedor de servicios de pago en España.

Es el mismo caso que `nis1-es` frente a `nis2-ue`, y la misma decisión: **el
instrumento que obliga tiene paquete propio**, porque un identificador que mezcle
dos instrumentos no se puede citar en un expediente.

## Lo que trae, y el hueco que enseña

| Artículo | Qué | Reloj |
|---|---|---|
| 67.1 | notificar al Banco de España los incidentes operativos o de seguridad graves | **obliga y no tiene número** |
| 66.2 | evaluación actualizada de riesgos operativos y de seguridad | al menos anual |
| 66.1 | marco de gestión de riesgos, con procedimientos de clasificación de incidentes | sin reloj |

**El artículo 67.1 dice "de forma inmediata y en la forma que este determine".**
Hay obligación y no hay número. El paquete lo declara así: el hito sale como *sin
plazo legal* y el motor mide el tiempo transcurrido en vez de inventarse una
fecha. Es la misma decisión que la notificación inicial de la tabla 3 del RD
43/2021.

**Dónde está el número que falta.** Lo ponen las directrices de la Autoridad
Bancaria Europea sobre notificación de incidentes graves, que el Banco de España
aplica. No se transcriben aquí porque este proyecto solo transcribe fuente
primaria (BOE, EUR-Lex, NIST) y las directrices de la ABE no lo son. Si tu
supervisor te ha dado un plazo concreto, se pone en tu copia del paquete
cambiando `indeterminado` por el número, sin tocar código.

**El artículo 66.2 tiene suelo legal y eso cambia quién puede moverlo.** Dice *al
menos una vez al año*: el Banco de España puede apretar la cadencia y el obligado
no puede aflojarla. Es lo contrario de un ritual de plazum, donde el número lo
ponemos nosotros y el cliente lo cambia libremente.

## Lo que NO hace

- No cubre el artículo 69 (resolución de reclamaciones: quince días hábiles, un
  mes en situaciones excepcionales), que es familia F.
- No cubre el artículo 24 (conservación seis años), que es familia E.
- No cubre el artículo 67.4 (datos estadísticos de fraude, al menos anuales).
- No dice si eres proveedor de servicios de pago. Eso lo declaras tú, y la regla
  de aplicabilidad del paquete lo toma como hecho.

## Comprobado

Tres casos dorados sobre el ciclo anual del artículo 66.2. El artículo 67.1 **no
tiene dorados y no puede tenerlos**: un caso dorado fija una fecha esperada y de
un plazo sin número no sale ninguna. El linter lo admite solo porque el hito
lleva su nota explicando qué dice la norma en vez del número que no da.
