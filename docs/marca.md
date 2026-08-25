# Marca: el expediente de UTIQ y la elección de PLAZUM

> **Para qué sirve este documento.** Es la entrada del agente de la propiedad industrial cuando toque consultarle. Recoge lo que se ha comprobado, con números y fechas, y separa lo que es hecho de lo que es opinión. Aquí no hay dictamen jurídico y no lo va a haber: el juicio de riesgo de confusión lo hace un profesional.
>
> **Estado a 26-08-2026: nombre decidido, PLAZUM.** El apartado "La criba de agosto de 2026" al final tiene la criba entera y el porqué. La congelación de publicación sigue vigente hasta que el nombre esté implantado: no se empuja tag, no hay release firmada, nada va a Rekor y el repositorio sigue privado.

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

---

# La criba de agosto de 2026: cómo se eligió PLAZUM

## Lo que se cribó

Dos candidatos propuestos, `vencia` y `preceptum`, y veinte generados después. Todos contra TMview, oficina EUIPO, clases 9 y 42, con las tres lentes.

Reproducible entero:

```bash
go run ./herramientas/cribamarca -clases 9,42 -candidatos vencia,preceptum
```

## Antes de cribar hubo que arreglar el cribador

**El semáforo decía ROJO siempre.** Con subcadenas de tres letras contra la base entera de la Unión, todo candidato lleva dentro algún acrónimo registrado en la clase 9: VEN, ENC, NCI, CIA, REC, ECE, CEP, EPT. Un semáforo que siempre dice lo mismo no dice nada, que es la regla de las puertas de este proyecto vista del revés.

Lo que separó a DUTIQ de una casualidad no fue que UTIQ existiera, fue **cuánto** de DUTIQ era UTIQ: cuatro letras de cinco, el 80%. Así que ahora pesa la **cobertura**, no la presencia. Umbrales, con el caso real que los calibra:

| Lente | Rojo | Ámbar | Calibrado con |
|---|---|---|---|
| La marca va dentro del candidato | cobertura ≥ 60% y ≥ 4 letras | ≥ 50% | UTIQ en DUTIQ = 80%, PRECEPT en PRECEPTUM = 78% |
| El candidato va dentro de la marca | ≥ 70% | ≥ 50% | VENCIA en AVENCIA = 86% |

Lo que queda por debajo del umbral **se cuenta y se dice**, no se descarta en silencio, y con `-todo` se lista. Un umbral que tira cosas sin decirlo hace que "sin hallazgos" se lea como "se ha mirado todo".

## Los dos propuestos: los dos rojos, y por la forma exacta de UTIQ

| Candidato | Veredicto | Por qué |
|---|---|---|
| **vencia** | ROJO | **AVENCIA**, EUTM 019216770, denominativa, registrada, clases 9, 35, 36, 41, 42, 45, de Oliver James Associates Group. El candidato es el 86% de la marca y sólo cambia la letra inicial. Toca las dos clases que nos importan. También SAVENCIA (012178778) al 75%, en clase 42 |
| **preceptum** | ROJO | **PRECEPT**, EUTM 018314665, denominativa, registrada, clases 9, 12, 35, 38, 39, 42, de **Polestar Holding AB**. La marca ajena es el 78% del candidato, en nuestras dos clases exactas, y enfrente hay un fabricante de coches |

Las dos son la misma forma que costó DUTIQ: una marca ajena ocupando la mayor parte del signo, en clases idénticas. No es ruido de acrónimo.

## Los veinte generados

Familia del gerundivo latino, "lo que debe hacerse", que es literalmente lo que calcula un motor de obligaciones, más algunas alternativas.

Cinco salieron limpias en EUIPO: **plazum, deontia, vinculum, peremptia, statuenda**. Las cinco se volvieron a cribar contra la **OEPM** (`-oficina ES`), que era un hueco anotado en este mismo documento, y las cinco siguen limpias.

Las quince restantes cayeron por cobertura real, no por ruido. Las más instructivas: PROBANDA lleva dentro **PROBAND**, denominativa de **Apple Inc.** en clase 9, al 88%. REGENDA es **colisión exacta** con Regenda AB en clase 42. ONERUM es colisión exacta con una marca china en clase 9. OBLIGIA lleva **oblig** al 71% en clases 35, 36, 37, 42 y 45.

## El paso que faltaba, y que descartó al finalista mejor

**TMview es un registro de marcas. No sabe nada de empresas en activo que operan sin registrar.**

`deontia` era, sobre el papel, la mejor de las cinco: *lógica deóntica* es literalmente la lógica de la obligación, y como signo es más distintiva que `plazum`, que roza lo descriptivo en español. Salió limpia en EUIPO y en OEPM.

Y está descartada. Existe **Deontic** (deontic.ai, Lovaina, fundada en 2022), plataforma de IA para gestión de cumplimiento regulatorio. Mismo sector, una letra de diferencia. Hay además **Deontics Ltd** (IA clínica, escisión de Oxford y UCL) y **Deontic Data** (Londres).

El registro estaba limpio y el mercado no. Es la misma clase de fallo que UTIQ, en otro registro: **buscar sólo donde es cómodo buscar**. Ahora el cribador lo dice en su propia salida, porque no se puede automatizar de forma fiable y una automatización a medias daría justo el falso verde que la herramienta existe para no dar.

## La decisión: PLAZUM

De `plazo`, con terminación latina. Seis letras.

**Por qué esta y no otra de las cinco:**

- Es la única de las cinco que sale limpia en las tres comprobaciones: EUIPO, OEPM y búsqueda de empresa en activo. Lo más cercano es Plazo Technologies, fintech española, otro sector y otro nombre.
- Dice lo que hace el producto a un CISO español, que es el comprador de la etapa 3, sin necesitar que nadie le explique una etimología.
- Es la más corta de las cinco.
- `vinculum` tiene homónimos operando (software de cadena de suministro), `peremptia` y `statuenda` son largas y opacas.

**Lo que hay que preguntarle al agente de la propiedad industrial**, y es lo único abierto: `plazo` es una palabra descriptiva del servicio en español, y el artículo 7.1.c del Reglamento de Marca de la Unión rechaza los signos descriptivos. `Plazum` no es una palabra española y exige un paso mental, que es la zona en la que un signo sugestivo sí se registra. Es una pregunta de dictamen, no de criba, y por eso no la resuelve este documento.

## Lo que queda congelado hasta que el nombre esté implantado

Lo mismo de antes: no se empuja tag, no hay release firmada, nada va a Rekor, el repositorio sigue privado. Se desbloquea cuando el módulo, el CLI, la marca, los documentos, la web, el dominio de compromiso del ledger y el expediente de demostración lleven el nombre nuevo.

