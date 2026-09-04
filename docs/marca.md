# Marca: el expediente de UTIQ y la elección de PLAZUM

> **Para qué sirve este documento.** Es la entrada del agente de la propiedad industrial cuando toque consultarle. Recoge lo que se ha comprobado, con números y fechas, y separa lo que es hecho de lo que es opinión. Aquí no hay dictamen jurídico y no lo va a haber: el juicio de riesgo de confusión lo hace un profesional.
>
> **Estado a 26-08-2026: PLAZUM, decidido, IMPLANTADO y con el candado ABIERTO.** El apartado "La criba de agosto de 2026" al final tiene la criba entera y el porqué. El renombrado está hecho de punta a punta: módulo Go, CLI, marca, documentos, web, dominio de compromiso del ledger (`plazum/commit/v1`) y expediente de demostración regenerado y **resellado contra una TSA real**. El candado se abrió el 26-08-2026 borrando `.github/marca-congelada`, y ese fichero es la única fuente de la verdad sobre este estado: **no lo preguntes a este documento, mira si el fichero está** (`ls .github/marca-congelada`). Lo que queda pendiente ya no es un permiso, es un acto: **la primera etiqueta `v*` que se empuje publica de verdad**, y publicar en Rekor es irreversible.

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

> **Aviso de caducidad, escrito el 04-09-2026.** Esta sección se escribió cuando el repositorio era privado y el candado estaba puesto, y **dos de sus cuatro puntos habían dejado de ser ciertos sin que nadie los tocara**. Es la enfermedad que este documento ya se ha cobrado dos veces hoy: el dato tiene puerta y la explicación no, así que el dato se corrige solo y la prosa se queda describiendo un mundo anterior. Los puntos que se pueden comprobar dicen ahora **con qué comando**, para que la próxima vez el lector no tenga que creerse el documento.

- El tag `v0.2.0` está creado en local y **no se empuja**. Comprobable: `git ls-remote --tags origin` no devuelve nada.
- **No hay release firmada.** La firma keyless de cosign publica la identidad del repositorio en el log público de Rekor, que es append-only y no se borra. Comprobable: `gh release list` sale vacío.
- ~~El repositorio sigue privado, lo que a su vez mantiene desactivados el workflow de CodeQL y el private vulnerability reporting.~~ **FALSO desde que el repositorio se hizo público, y falso dos veces.** `gh repo view --json isPrivate` devuelve `false`, y **CodeQL no está desactivado: corre y sale verde** (run `33855507365` sobre `a7c43b2`). Quien leyera esta línea daría por apagado un análisis que lleva tiempo funcionando, y daría por cerrada una exposición que está abierta.
- **El post del ledger** (`docs/post-ledger-salamanders.md`) está escrito y sin publicar.

~~Nada de esto es trabajo pendiente. Todo se desbloquea con una decisión, no con código.~~

**Y esa frase también era falsa, y la refutó el primer intento de usar el mecanismo.** El 04-09-2026 se lanzó por primera vez `release.yml` con `workflow_dispatch` y el trabajo `imagen` murió en su primer paso: `Multi-platform build is not supported for the docker driver`. Un runner limpio trae buildx con driver `docker`, que construye una sola arquitectura, y `publicar` depende de `imagen`. **Una etiqueta empujada ese día no habría publicado nada: habría dado una ejecución roja.** Faltaba código (`setup-qemu-action`, `setup-buildx-action` y un `--load` explícito), y faltaba desde siempre.

La lección es la que ya llevaba escrita la cabecera de `release.yml` sin que nadie la ejerciera: **un workflow de release que solo se ha ejecutado el día de la release es un workflow que se estrena el peor día.** Y el detalle que la hace inapelable: el driver de buildx **no está en el YAML, está en el runner**, así que ninguna cantidad de leer el fichero lo habría encontrado.

Lo que queda por decidir sigue siendo una decisión y no código: **empujar la primera etiqueta**. Con el candado abierto, eso firma en Rekor y publica en ghcr.io, y no se deshace.

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

## La prueba de PLAZUM, pegada y no resumida

Después de UTIQ, un nombre no se da por bueno por informe. Esto es la salida literal del cribador el 26-08-2026, con `-todo`, o sea **sin ocultar lo que queda bajo el umbral**.

```
$ go run ./herramientas/cribamarca -candidatos plazum -clases 9,42 -oficina EM -todo
criba de marca, clases 9,42, EUIPO (marca de la Union Europea)

== PLAZUM  [sin hallazgos]
   COLISIONES (se llaman igual): 0
   CONTENEDORAS (una marca contiene el candidato): 0
   CONTENIDAS (el candidato contiene una marca): 1
      [ruido  50%] 016915233  AZU  Word  Registered  clases 9  GUANGDONG SIRUI OPTICAL CO.

$ go run ./herramientas/cribamarca -candidatos plazum -clases 9,42 -oficina ES -todo
criba de marca, clases 9,42, OEPM (marca nacional espanola)

== PLAZUM  [sin hallazgos]
   COLISIONES (se llaman igual): 0
   CONTENEDORAS (una marca contiene el candidato): 0
   CONTENIDAS (el candidato contiene una marca): 1
      [ruido  50%] M1928953  AZU  Combined  Registered  clases 42  JESUS MARIA ZUGARRONDO
```

Las tres lentes en cero. Lo único que aparece es **AZU** dentro de plaz-**azu**-m, tres letras que cubren el 50% del nombre: por debajo de los dos umbrales, y de la misma naturaleza que los acrónimos que hacían que el semáforo dijera ROJO siempre. Se lista aquí a propósito, porque un umbral que descarta en silencio hace que "sin hallazgos" se lea como "se ha mirado todo".

**Un fallo del propio cribador, encontrado al preparar esta prueba.** La cabecera decía `oficina EUIPO` cableado, mirara donde mirara: con `-oficina ES` la consulta iba de verdad a la OEPM (los números que salen son españoles, `M1928953`) y el rótulo seguía diciendo EUIPO. En una herramienta cuyo único producto es la prueba, una cabecera que miente sobre el registro consultado deja la prueba sin valor: quien la lea dentro de un año no puede saber dónde se buscó. Corregido, con test y con mutación en rojo.

**Un segundo fallo del cribador, encontrado al volver a pedir esta prueba (26-08-2026).** TMview devuelve como mucho `pageSize` registros por página y **no dice cuántos hay en total**: ni `totalResults`, ni cabecera de conteo. El cribador pedía una sola página de 50 y daba el término por agotado. Con más de 50 anterioridades, la 51 era invisible y la tabla se imprimía igual.

Y hay algo peor debajo, que es lo que convierte esto en puerta y no en mejora:

```
pageSize=50   azu -> 50 marcas
pageSize=100  azu -> 100 marcas
pageSize=200  azu ->   0 marcas      <-- HTTP 200, sin error, lista vacia
```

**Doscientos es más de lo que la API sirve, y contesta que no hay nada.** Con la versión anterior de este fichero, tocar esa constante habría hecho que *todo* candidato saliera "sin hallazgos" y la herramienta habría seguido imprimiendo su tabla con la misma cara de siempre. Un transporte roto y un nombre limpio se leían **exactamente igual**, que es la definición de falso verde y la razón por la que este proyecto no se fía de un verde que nunca se ha visto rojo.

Arreglado con tres puertas, cada una con su mutación en rojo demostrada:

| puerta | mutación | qué se puso rojo |
|---|---|---|
| canario de transporte antes de cribar nada, sin pasar por caché | `if len(ms) == 0` → `if len(ms) < 0` | `TestElCanarioCazaUnTransporteQueContestaVacio` |
| paginar hasta que una página venga corta | `if len(ms) < paginaTamano` → `if len(ms) >= 0` | `TestLaBusquedaRecogeLasPaginasQueSiguen`, `TestUnTerminoQueNoSeAgotaEsUnError` |
| tope de páginas que escupe en vez de recortar en silencio | `return nil, fmt.Errorf(...)` → `return todas, fmt.Errorf(...)` | `TestUnTerminoQueNoSeAgotaEsUnError` |

**¿Cambia el veredicto de PLAZUM? No, y está medido, no supuesto.** Los términos que deciden son los de 4 letras o más, porque por debajo de eso es acrónimo y es ruido por umbral. Ninguno se acercaba al tope:

```
EUIPO                       OEPM
plazum ->  0 resultados     plazum ->  0
plazu  ->  0                plazu  ->  0
lazum  ->  0                lazum  ->  0
plaz   ->  4                plaz   ->  1
lazu   ->  3                lazu   ->  0
azum   ->  3                azum   ->  3
--- por debajo del umbral, y aqui si tocaban techo ---
pla    -> 50  <<< tope      pla    -> 50  <<< tope
laz    -> 50  <<< tope      laz    -> 37
azu    -> 50  <<< tope      azu    -> 50  <<< tope
zum    -> 50  <<< tope      zum    -> 44
```

O sea que el veredicto no estaba truncado, **pero por suerte del candidato, no por una propiedad de la herramienta**. La criba se repitió entera con el cribador paginado (19 consultas en EUIPO, 13 en OEPM, frente a 10 y 10) y sale lo mismo: tres lentes en cero, AZU como único ruido.

Y lo que devuelven esos términos, para que no haya que creerse el "0":

```
EUIPO   plaz -> IPLAZ (cl 35), EZPLAZ (cl 5), NIRPLAZ (cl 5), AROPLAZ (cl 1,2)
        lazu -> BELAZU x3 (cl 29,30,31), dos de ellas caducadas
        azum -> FUN AZUM (cl 3), Vitazum (cl 32,35), KANGAZUM (cl 5)
OEPM    plaz -> PLAZ, M4190316, clase 14 (joyeria)
        azum -> JAZUM (cl 32), lavaZum M4293761 (cl 9,42), ALDUSAZUM (cl 30)
```

Ninguno es coincidencia exacta con una subcadena de plazum, que es lo que la lente 3 busca. El único que está **en nuestras clases exactas y en nuestro país** es `lavaZum` (OEPM, clases 9 y 42): comparte la terminación `azum`, pero el elemento distintivo va delante y es `lava` frente a `pla`. Se anota aquí porque callarlo sería exactamente lo que el `-todo` existe para impedir, no porque cambie la decisión.

### Los vecinos a una letra, que el cribador no puede ver

Esta es la parte que **ninguna herramienta de subcadenas alcanza** y que tumbó a Deontia: un competidor a una sustitución de distancia no es subcadena de nada. Se hace a mano y se pega:

- **Plazus Technologies Inc** (Vancouver, Canadá). Agencia de desarrollo software con paquete de ciberseguridad; entre sus servicios, cumplimiento de ISO 27001, RGPD e HIPAA. **Sin marca registrada**: `plazus` devuelve cero resultados en EUIPO con las tres modalidades de búsqueda. Es una consultora de servicios sin producto con nombre, fuera de la Unión, y no vende un motor de obligaciones. Riesgo bajo, anotado.
- **Plazo Technologies** (subsidiaria de ingeniería de ID Finance, fintech de préstamo alternativo). `plazo` sí tiene registros, pero es **palabra del diccionario español** y por eso mismo débil como signo, y el sector es concesión de crédito, no cumplimiento.
- **plazum.com** está registrado desde 2014-04-25 (Namecheap, servidores de nombres `owlbits.com`) y **no sirve nada por HTTP**. Hay una página de Facebook "Plazum Paraguay" sin web enlazada. No hay empresa en activo en nuestro sector con este nombre.

La diferencia con Deontia, dicha explícitamente: Deontic era un **producto de IA para cumplimiento regulatorio, en la Unión, a una letra**. Plazus es una **consultora generalista, en Canadá, sin marca y sin producto con nombre**. No es el mismo hallazgo con distinto color.

### El dominio

`plazum.dev` está **libre** a 26-08-2026. Comprobado por RDAP contra dos servidores, uno de ellos el del registro autoritativo:

```
https://rdap.org/domain/plazum.dev                    HTTP 404
https://www.registry.google/rdap/domain/plazum.dev    HTTP 404
nslookup -type=NS plazum.dev                          Non-existent domain
```

Con su control negativo, porque un 404 puede significar "el método no funciona":

```
https://rdap.org/domain/google.dev   HTTP 200  ldhName=google.dev
https://rdap.org/domain/web.dev      HTTP 200  ldhName=web.dev
```

### El separador de dominio del ledger

No basta con que la constante diga `plazum/commit/v1`: hay que demostrar que el expediente de demostración **commiteado** está atado a ella. Se devuelve la etiqueta al valor viejo y se pide verificar:

```
$ sed -i 's|plazum/commit/v1|dutiq/commit/v1|' nucleo/ledger/v2.go
$ go test . -run TestLaDemoVerificaConElVerificadorDeVerdad
--- FAIL
  entrada 0 de la cadena: la clave no compromete este cifrado: clave equivocada o sustituida
  entrada 1 de la cadena: la clave no compromete este cifrado: clave equivocada o sustituida
  entrada 2 de la cadena: la clave no compromete este cifrado: clave equivocada o sustituida
  observacion backup.restauracion/sede-electronica: no aparece en la cadena, o aparece con otro contenido
  ...
```

Las tres entradas dejan de abrirse. Eso sólo puede pasar si la demo se **regeneró** con la etiqueta nueva, no si sólo se cambió la constante.

## La implantación, hecha el 26-08-2026

El renombrado es de punta a punta, 152 ficheros y 662 ocurrencias. Lo que no es mecánico y conviene saber:

- **`dutiq/commit/v1` pasa a `plazum/commit/v1`**, que es el separador de dominio del compromiso de clave del ledger, y **entra en el hash de cada entrada**. Con la semilla del operador de la demo, que también cambia, eso mueve la raíz Merkle de la cadena, así que el expediente de demostración hubo que **resellarlo contra una TSA real**, no regenerarlo y ya.
- Eso destapó un huevo y una gallina: `generardemo` se niega a escribir un expediente que no verifica, y no verifica sin un sello que cubra su raíz nueva, que sólo existe después de construirlo. Se rompió con una bandera `-raiz` en `sellardemo`, que además vino con su propio riesgo (la raíz se copia a mano de un mensaje de error) y por tanto con su comprobación de longitud y su test.
- **Tres ficheros quedan fuera del renombrado a propósito**, porque en ellos `dutiq` es historia y no marca: `herramientas/cribamarca` (su test se llama "las subcadenas incluyen la que nos mordió" y el caso ES `dutiq`), este documento, y `docs/decisiones.md`. Renombrarlos falsearía el expediente de la marca.
- El barrido mecánico sí falsificó cuatro documentos que narran la historia antes de que nadie lo mirase: `.github/marca-congelada` llegó a decir "PLAZUM contiene UTIQ entero", que es falso. Corregidos a mano. **Es la lección de este renombrado**: un `sed` global sobre un repositorio que documenta sus propias decisiones convierte el registro histórico en mentira, y lo hace en silencio.

## El candado, abierto el 26-08-2026

`.github/marca-congelada` se borra en su propio commit, que es como el propio fichero decía que tenía que abrirse: **"en un commit propio que diga quién lo decidió y cuándo"**.

- **Quién**: Marcos Mata, a la vista de la prueba pegada arriba.
- **Cuándo**: 26-08-2026, después de repetir la criba con el cribador paginado y de la comprobación manual de vecinos a una letra.
- **Qué se comprobó antes**: tres lentes en cero en EUIPO y en OEPM, `plazum.dev` libre por RDAP contra el registro autoritativo, ninguna empresa en activo en el sector con este nombre, y el separador de dominio del ledger demostrado atado al expediente commiteado.

**Lo que el borrado abre, dicho sin suavizar.** El mecanismo NO se ha tocado: `release.yml` sigue preguntando por el fichero antes de cada paso que sale de la máquina, y `distribucion_test.go` sigue poniéndose rojo si alguien añade uno que no pregunte. Lo que cambia es la respuesta a esa pregunta. A partir de aquí:

- empujar el tag `v0.2.0` dispara la release firmada con cosign keyless, y eso **publica en Rekor**, que es un log de solo añadir: el nombre y la identidad del repositorio quedan ahí para siempre;
- el trabajo de imagen puede subir a `ghcr`.

Ninguna de las dos cosas se hace en este commit. Borrar el candado es dar el permiso; disparar la publicación es un acto aparte y deliberado, y la lección de UTIQ es exactamente que lo irreversible se separa de lo reversible.

## Lo que queda por hacer, que ya no es abrir nada

**Esta sección decía «lo que queda congelado» y describía un mundo que dejó de existir el 26-08-2026**, cuando el candado se abrió dos secciones más arriba. El dato tenía puerta (`.github/marca-congelada` existe o no existe, y `release.yml` lo pregunta en cada paso que sale de la máquina) y la explicación no la tenía, así que el dato se corrigió solo y la explicación se quedó diciendo «se abre borrando el fichero» sobre un fichero ya borrado. Se corrige el 04-09-2026, y se deja escrito el fallo en vez de disimularlo: es la familia de la **afirmación acompañada**, en su forma más cara, que es cuando quien miente es la prosa y no el número. Un número falso se contrasta; una explicación falsa se cree.

**El estado NO se lee aquí, se lee del árbol.** El candado está abierto si `.github/marca-congelada` no existe, y esa comprobación la hace el trabajo `candado` de `release.yml` en cada ejecución, diciendo en voz alta cuál de las dos respuestas dio.

Lo que sigue sin hacerse, y es un acto deliberado y no un permiso:

- **no se ha empujado ninguna etiqueta `v*`**, así que nada se ha firmado, nada ha ido a Rekor y nada se ha subido a `ghcr.io`. La primera que se empuje hace las tres cosas;
- por eso la primera etiqueta es un **candidato** (`v0.1.0-rc1`) y no una `v1`: si la primera ejecución real de un workflow que nunca se ha ejecutado sale torcida, se quema un `rc` y no el número de versión con el que se publica el producto.

