# eni — Real Decreto 4/2010 (Esquema Nacional de Interoperabilidad)

Texto del BOE, transcrito (art. 13 TRLPI). Fuente: instantánea con huella del BOE, `BOE-A-2010-1331`, consolidado.

## El resultado, y es un resultado negativo

**El ENI no tiene ni un reloj periódico.** No es que falte por transcribir: es que **no los hay**.

Se recorrieron sus 42 artículos buscando cualquier lenguaje temporal (plazo, meses, años, días, periódico, actualizar, revisar, vigencia, caducidad, renovación). Lo que hay es:

| tipo | dónde | qué es |
|---|---|---|
| **deberes permanentes** | arts. 9, 10, 29 | *«mantendrá actualizado»*, *«se mantendrá actualizada»*. Sin cadencia y sin plazo |
| **deberes de establecimiento** | arts. 8, 11, 21, 27, 28 | establecer, publicar, adoptar medidas, dar publicidad. Se hacen una vez y se mantienen |
| **un plazo transitorio, agotado** | DT única | doce meses desde la entrada en vigor (30-01-2010) para disponer de un plan si no cabía la plena aplicación |
| **deberes del Estado** | arts. 10.1, 29 y DA primera | mantener el propio Esquema y la Relación de modelos de datos. Obligan a quien regula, no a quien cumple |

**Ese resultado se escribe porque es información.** Un paquete vacío sin explicación se lee como *«falta por hacer»* y hace que alguien vuelva a buscar dentro dentro de seis meses. Un paquete que dice *«aquí no hay relojes, y este es el barrido»* cierra la pregunta.

## Lo único que sí entra: el art. 9.1

*«Cada Administración Pública **mantendrá actualizado** el conjunto de sus inventarios de información administrativa»*.

Es un deber **permanente** y va como tal: primitiva `continua`, sin plazo legal, y el motor mide el tiempo transcurrido. **No se le pone número**, porque el artículo no lo da; inventarle un trimestre habría sido poner una cifra que el texto no tiene. El porqué de tratarlo así, en `docs/decisiones.md` D-17.

**A quién alcanza.** Al sector público, por el art. 3 del propio Real Decreto: el hecho `ambito(E, "sector_publico")`.

## Por qué el ENI aporta poco a un calendario y mucho a otra cosa

El ENI es una norma de **arquitectura**, no de ritmo: dice cómo tienen que hablarse los sistemas, no cada cuánto hay que revisarlos. Su vecino en el mismo Real Decreto —el **Esquema Nacional de Seguridad**— es lo contrario: relojes por todas partes, y por eso el paquete `ens` tiene doce y éste uno.

Confundir los dos y esperar del ENI un calendario es un error de expectativa que este documento existe para evitar.

## Lo que este paquete NO hace

- **La DT única**, el plazo de doce meses desde el 30-01-2010. Está agotado hace más de quince años: escribirlo produciría una fila que dejó de obligar antes de cualquier ventana que alguien vaya a mirar. Se deja anotado aquí en vez de en el corpus.
- **Las NTI** (Normas Técnicas de Interoperabilidad) que desarrollan el Esquema. Son el sitio donde sí podría haber ritmo, y no están transcritas.
