# Marca: el expediente de UTIQ

> **Para qué sirve este documento.** Es la entrada del agente de la propiedad industrial cuando toque consultarle. Recoge lo que se ha comprobado, con números y fechas, y separa lo que es hecho de lo que es opinión. Aquí no hay dictamen jurídico y no lo va a haber: el juicio de riesgo de confusión lo hace un profesional.
>
> **Estado a 25-08-2026: marca congelada.** No se empuja el tag v0.2.0, no hay release firmada, nada va a Rekor y el repositorio sigue privado.

## El hallazgo

La comprobación en TMview del 25-08-2026 devolvió dos marcas de la Unión Europea **registradas**, no solicitadas, del mismo titular y con el mismo alcance.

| | EUTM 018838934 | EUTM 018838908 |
|---|---|---|
| Denominación | Utiq | Utiq |
| Tipo | **Denominativa** | Figurativa |
| Estado | Registrada | Registrada |
| Solicitada | 21-02-2023 | 21-02-2023 |
| Registrada | 13-06-2023 | 13-06-2023 |
| Vigente hasta | 21-02-2033 | 21-02-2033 |
| Clases Niza | 9, 25, 35, 38, 42 | 9, 25, 35, 38, 42 |

**Titular:** Utiq SA/NV, persona jurídica belga, Rue aux Laines, 1000 Bruselas.
**Representante ante la EUIPO:** SIRIUS LEGAL.
**Origen del dato:** API de TMview (`tmdn.org`), oficina EM, consultada el 25-08-2026. Reproducible con `go run ./herramientas/cribamarca -candidatos dutiq`.

Utiq es la empresa conjunta de Deutsche Telekom, Orange, Telefónica y Vodafone para publicidad digital con consentimiento.

## Por qué esto importa más de lo que parecía

El primer aviso interno decía "clases 9 y 42". Eso se quedaba corto en dos sentidos.

**Primero, el alcance es mayor:** son cinco clases, 9, 25, 35, 38 y 42.

**Segundo, y esto es lo que cambia el análisis, el solape no es de número de clase sino de descripción.** Una clase Niza es una casilla administrativa muy ancha; dos productos pueden compartir clase y no parecerse en nada. Aquí no es el caso. Estas son las descripciones literales de las dos clases que nos afectan.

### Clase 42 (servicios)

> Data migration services; Data mining; Development of systems for the processing of data; Development of systems for the storage of data; Development of systems for the transmission of data; **Platform as a service [PaaS]**; **Software as a service [SaaS]**; Technical advisory services relating to data processing.

### Clase 9 (productos), extracto

> Computer programmes for data processing; Artificial intelligence software for analysis; **Application software for mobile devices**; Application software for mobile phones; Application software for televisions; Application software for wireless devices; **Application software for cloud computing services**; **Application software**; Communication software for connecting computer network users; Application software for social networking services via internet.

"Software as a service" y "Development of systems for the processing of data" no describen un vecino de dutiq. Describen dutiq.

**Y la que pesa es la denominativa (018838934).** Una marca denominativa protege el signo con independencia de cómo se dibuje, así que cambiar la tipografía, el color o el logotipo no cambia nada del análisis.

## La forma del riesgo

Dicho como hecho, no como dictamen:

- **DUTIQ contiene UTIQ entero.** Es el caso que casi nadie comprueba: se busca si el candidato colisiona con algo, y no si el candidato *contiene* una marca ajena. Por eso `herramientas/cribamarca` mira las tres direcciones.
- **El solape de clases es total** en lo que nos afecta: 9 y 42 están las dos.
- **El solape de descripción es literal**, no analógico.
- **Enfrente hay cuatro operadoras de telecomunicaciones** con presupuesto y con un despacho ya designado.

Lo que un profesional tendrá que valorar, y que aquí no se resuelve: si el prefijo "D" basta para distinguir los signos a ojos del consumidor medio, cómo pesa la coincidencia de servicios en el riesgo de confusión, y qué probabilidad real hay de oposición.

## Una fecha que conviene tener anotada

Las dos marcas se registraron el **13-06-2023**. En la Unión Europea, una marca registrada queda expuesta a la exigencia de prueba de uso efectivo a los cinco años de su registro. Es decir, **desde junio de 2028** se le puede pedir a Utiq que acredite uso real para cada clase.

Eso importa por dos razones prácticas:

1. Si Utiq no usa la marca en alguna de las cinco clases, esa parte se vuelve atacable a partir de esa fecha.
2. En una eventual oposición posterior a junio de 2028, se le puede exigir que pruebe el uso en las clases que invoque, en vez de darlas por buenas.

La clase 25 (ropa y calzado) parece la más difícil de sostener para una empresa de publicidad digital, aunque no es una de las que nos afectan. Las que nos afectan, 9 y 42, son las que con más probabilidad sí usan.

Esto **no es una estrategia recomendada**, es un dato de calendario para que el profesional lo tenga delante.

## Qué preguntar al agente de la propiedad industrial

Por orden de lo que bloquea antes:

1. ¿Es asumible el riesgo de registrar DUTIQ en clases 9 y 42 existiendo la denominativa 018838934? ¿Con qué probabilidad estimada de oposición?
2. Si la respuesta es "arriesgado", ¿hay alguna variante del nombre que se distancie lo suficiente sin perder la identidad?
3. ¿Cambia algo el hecho de que dutiq sea software libre AGPL y que el uso comercial inicial sea pequeño?
4. ¿Conviene una búsqueda de anterioridades formal, más allá de esta criba automática, antes de presentar?
5. ¿Y las marcas nacionales españolas y el nombre de dominio? Esta comprobación solo miró la EUIPO.

## Lo que está congelado mientras tanto

- El tag `v0.2.0` está creado en local y **no se empuja**.
- **No hay release firmada.** La firma keyless de cosign publica la identidad del repositorio en el log público de Rekor, que es append-only y no se borra.
- **El repositorio sigue privado**, lo que a su vez mantiene desactivados el workflow de CodeQL y el private vulnerability reporting.
- **El post del ledger** (`docs/post-ledger-salamanders.md`) está escrito y sin publicar.

Nada de esto es trabajo pendiente. Todo se desbloquea con una decisión, no con código.

## Cómo repetir la comprobación

```bash
go run ./herramientas/cribamarca -candidatos dutiq,otronombre -clases 9,42
```

La herramienta consulta TMview, cachea en disco y compara en las tres direcciones (colisión exacta, marcas que contienen el candidato, y candidatos que contienen una marca). Detalles en su propia documentación.
