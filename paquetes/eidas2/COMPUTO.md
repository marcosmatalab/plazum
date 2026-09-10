# eidas2: de dónde sale cada fecha

## 1. Dos plazos de 24 horas del mismo reglamento con disparadores distintos

Es el mejor ejemplo de todo el corpus de por qué un catálogo de controles no
sirve y hace falta un motor de plazos. El Reglamento 910/2014, en la redacción
del Reglamento (UE) 2024/1183, tiene dos obligaciones de notificar en 24 horas
que **no cuentan desde el mismo hecho**:

| Obligación | Quién | 24 horas desde | Artículo |
|---|---|---|---|
| Notificar una violación de seguridad o interrupción | prestador **no** cualificado | **haber tenido conocimiento** | 19 bis, apartado 1, letra b |
| Notificar una violación de seguridad o interrupción | prestador **cualificado** | **haberse producido el incidente** | 24, apartado 2, letra f ter |

La diferencia no es de matiz. Un prestador cualificado que se entera de un
incidente veintidós horas después de que ocurriera **tiene dos horas**, no
veinticuatro. Y si se entera a las cuarenta y ocho horas, su plazo ya venció:
esa es una situación que hay que **enseñar**, no esconder, y por eso hay un caso
dorado con el vencimiento en el pasado.

Un producto que tratara las dos como "24 horas desde el incidente" o como "24
horas desde que te enteras" le daría a la mitad de sus clientes una fecha
equivocada, y en la dirección cara.

## 2. El tercer plazo de 24 horas: la revocación

El artículo 24.3 obliga a publicar el estado de revocación de un certificado
cualificado *en todo caso, en un plazo de 24 horas después de la recepción de la
solicitud*. Disparador: la recepción de la solicitud.

Hay un dorado con la solicitud recibida un **sábado**, y no es decorativo: el
artículo 3.4 del Reglamento (CEE, Euratom) 1182/71 traslada al hábil siguiente
los plazos expresados *de cualquier modo, salvo en horas*, así que a este **no
le alcanza**. Una autoridad de certificación que solo revoque en horario de
oficina incumple, y el paquete lo dice con una fecha.

## 3. Régimen

Los tres plazos están en horas: cómputo natural, **cierre exacto** y **traslado
ninguno**, por el artículo 3.4 a contrario. Ni el cambio de mes ni el cambio de
año los mueven, y hay dorados para los dos bordes.

## 4. Vigencia, que no es una sino dos

El texto que se ha leído es el consolidado del 910/2014 a 18-10-2024, y **los tres
plazos no empiezan a obligar el mismo día**, porque no salen del mismo acto. La
consolidada lo marca en el margen y ese margen es el dato:

| obligación | marca | vigencia | de dónde sale |
|---|---|---|---|
| art. 19 bis.1.b | ▼M2 | **2024-05-20** | vigor del Reglamento (UE) 2024/1183 (art. 2, veinte días desde la publicación de 30-04-2024) |
| art. 24.2.f ter | ▼M2 | **2024-05-20** | ídem |
| art. 24.3 | ▼B | **2016-07-01** | aplicación del propio 910/2014 (art. 52.2) |

El artículo 24.3 es **texto base**: no lo introduce ni lo toca eIDAS 2. Su fecha
es la de aplicación general del Reglamento 910/2014, y no la de entrada en vigor
(17-09-2014), porque el apartado 3 **no** está entre las excepciones del art.
52.2, letra a), donde sí está el apartado 5 del mismo artículo. Lo que obliga es
la aplicación, que es la convención de toda la casa (`rgpd`, `dora`, `mdr`,
`mica`).

**Y el 18-10-2024 del enlace no es de eIDAS 2**, aunque lo parezca: es la fecha
con la que se rehace la consolidada porque el art. 42 de la Directiva (UE)
2022/2555 (NIS2) suprime el art. 19 del 910/2014 *con efectos a partir del 18 de
octubre de 2024*. El 2024/1183 no tiene aplicación diferida.

**Hasta el 10-09-2026 las tres decían `2024-05-20`**, heredado de un `urn` que
nombraba al reglamento modificativo. Siete años y diez meses de más en la fila
del art. 24.3. El porqué y las tres salidas que se sopesaron, en D-26.

## 5. Lo que no está

- El **artículo 20.1**, la auditoría de conformidad cada 24 meses del prestador
  cualificado. Es familia C del censo.
- Las obligaciones del **organismo de supervisión**, que no son del obligado.
- La **cartera europea de identidad digital** y sus plazos propios.
