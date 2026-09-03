# Hallazgos: las tres fechas de una norma

Frente D de la campana del 03-09-2026. Columna: `herramientas/ingestanorma/`,
`corpus-vigilancia/`, `vigencias_test.go`, este fichero.

Todo dato normativo de aqui esta verificado contra fuente primaria (Cellar, el
servicio de datos de la Oficina de Publicaciones de la UE) el **03-09-2026**, con
`Accept: application/xml;notice=branch` sobre
`publications.europa.eu/resource/celex/{CELEX}`.

## El numero

| momento | relojes contrastables | expuestos | de ellos, instantanea muda | de ellos, sin instantanea |
|---|---|---|---|---|
| antes (03-09-2026, manana) | 34 | **196** | 154 | 42 |
| despues | 191 | **39** | 0 | 39 |

Los dos numeros salen de la puerta de CI «suite completa», que corre
`vigencias_test.go` dentro de `./...`.

**Los 39 que quedan no bajan nunca**, y por eso el techo dejo de ser una tarea
pendiente y paso a ser una frontera:

| paquete | relojes | por que no puede tener instantanea |
|---|---|---|
| `urn:iso-iec:42001:2023` | 10 | invariante 3: el texto de ISO no se transcribe |
| `urn:iso-iec:27001:2022` | 9 | idem |
| `urn:pcissc:dss:4` | 7 | idem |
| `urn:aicpa:tsc:2017` | 5 | idem |
| `urn:enx:tisax:6` | 5 | idem |
| `urn:plazum:demo:empresa` | 3 | es la empresa de demostracion, sintetica |

## Por que no hizo falta ningun parser de prosa

El encargo daba por hecho que la entrada en vigor vive en la prosa del ultimo
articulo («entrara en vigor a los veinte dias de su publicacion») y que habria que
resolver esa formula. **Se miro antes la otra puerta y el dato estaba como dato.**

La ficha `notice=branch` de Cellar trae, al nivel de la OBRA:

| elemento | que es |
|---|---|
| `WORK_DATE_DOCUMENT` | la fecha DEL ACTO |
| `DATE_PUBLICATION` | la de publicacion, para el Diario nuevo (desde 2023-10) |
| `RESOURCE_LEGAL_PUBLISHED_IN_OFFICIAL-JOURNAL` → `EMBEDDED_NOTICE/WORK/DATE_PUBLICATION` | la de publicacion, para el Diario viejo |
| `RESOURCE_LEGAL_DATE_ENTRY-INTO-FORCE` | un hito por cada fecha, con su papel |

Cada hito lleva `ANNOTATION/TYPE_OF_DATE` con el codigo de la autoridad `fd_335`
(`EV` = entrada en vigor, `MA` = aplicacion, `MA/PART` = de una parte) y
`COMMENT_ON_DATE` con **la regla que la produjo**, en la forma `DATPUB +20`.

Eso permite algo que un parser de prosa no permite: **contrastar la fecha contra
la regla de la propia fuente**. Es el unico sitio del recorrido donde dos datos
independientes de la ficha se miran el uno al otro. Las once normas de la UE del
corpus cuadran hoy; y esa comprobacion **nacio verde sobre las once**, lo cual se
dice en voz alta: o vigila poco o llego tarde, y aqui es lo segundo, porque su
valor es que la doceava no pueda entrar torcida en silencio.

## Lo que se descubrio y no se sabia

### H-D1. Una fecha puede hacer DOS papeles, y Cellar lo escribe como dos anotaciones

Un mismo `RESOURCE_LEGAL_DATE_ENTRY-INTO-FORCE` puede llevar **varias**
`ANNOTATION`. El 29-06-2023 de MiCA es a la vez la entrada en vigor (art. 149.1)
y el escalon de aplicacion de unos articulos (art. 149.4). Lo mismo el 17-09-2014
de eIDAS, y el 26-05-2021 del MDR lleva dos anotaciones de aplicacion.

**Esto costo una medida falsa en esta misma sesion.** La primera exploracion leyo
solo la primera anotacion de cada hito y concluyo que **dos de las diez** normas
de la UE del corpus (eIDAS 1 y MiCA) no tenian entrada en vigor declarada. Las
once la tienen. No lo cazo ninguna mutacion: lo cazo el test contra el fixture
REAL de MiCA, que se puso rojo diciendo que salia una fecha donde el autor
esperaba un hueco. El error no estaba en el codigo, estaba en lo que se creia que
decia la fuente, y contra eso solo sirve el dato real.

### H-D2. La fecha de publicacion vive en dos sitios distintos segun el ano

El Diario Oficial cambio de formato en octubre de 2023. Los actos posteriores
llevan `DATE_PUBLICATION` en la obra; los anteriores no lo llevan, y hay que
sacarlo del enlace al numero del Diario. **Seis de las once** normas de la UE del
corpus son del formato viejo: leer solo la obra las habria dejado sin fecha de
publicacion, y con ella se pierde el unico contraste que caza la conflacion del
invariante 10 (confundir «de 8 de julio» con «publicado el 8 de julio»).

### H-D3. Cellar escribe `1001-01-01` donde no consta la fecha

Pasa hoy en el Reglamento 2017/745 (MDR), art. 123.3(d), (e) y (ea). Tragarselo
pondria en el calendario un vencimiento del ano 1001; descartarlo callando
perderia un escalon de aplicacion sin que nadie lo echara de menos. Se guarda el
hito **con la fecha vacia y la nota que dice por que**, y el articulo de apoyo,
que es por donde hay que empezar a mirarlo a mano.

### H-D4. La ficha del acto base no recoge lo que le movio un omnibus

La ficha de Cellar del Reglamento 2024/1689 (AI Act) sigue declarando el
02-08-2026 como fecha de aplicacion general, que es lo que decia antes de que el
Reglamento 2026/1744 moviera dieciseis meses el capitulo III. Las dos fechas del
paquete `ai-act` que salen de ahi (`2026-07-27` y `2027-12-02`) **no** figuran
entre las que declara la ficha del acto base.

Consecuencia de proceso: la comprobacion «la vigencia de un paquete es una de las
fechas que declara su fuente» **no puede ser una acusacion**, porque pondria rojo
un paquete correcto. Es un recuento con techo (`TestSeCuentanLasVigenciasQueNoSonNingunaFechaDeLaFuente`,
17 de 336 hoy), y ese es el sitio donde hay que mirar cuando el numero suba: una
fecha que la ficha del acto base no declara puede venir de un acto modificador
(bien) o de nadie (mal), y desde fuera se ven igual.

## El limite que NO se cerro, con su cardinal

**La aplicacion escalonada se fecha, pero no se sabe QUE alcanza cada escalon.**

Cellar nombra el ARTICULO de la disposicion final que fija el escalon (el 71.2 del
CRA, el 113(b) del AI Act), no el ambito. **17 escalones parciales** del corpus
estan hoy fechados sin saber, desde el dato, a que capitulo o articulos alcanzan:

| norma | escalones parciales |
|---|---|
| Reglamento 2017/745 (MDR) | 9 (uno de ellos sin fecha, ver H-D3) |
| Reglamento 2024/1689 (AI Act) | 3 |
| Reglamento 2024/2847 (CRA) | 2 |
| Reglamento 2023/1114 (MiCA) | 2 |
| Reglamento 910/2014 (eIDAS) | 1 |

Es el hallazgo H8 con otro nombre: la instantanea no trae los rotulos de capitulo
ni de seccion, asi que no hay con que cruzar «capitulo IV» contra un escalon. La
salida de la herramienta lo dice donde se lee, no en una nota al pie: *«La fuente
dice el articulo que fija cada escalon, no que capitulo alcanza: eso hay que
leerlo en el articulo»*.

## Las seis mutaciones, con lo que se puso rojo

Una puerta que nunca se ha visto fallar no es una puerta. Las seis se aplicaron y
se restauraron **con copia**, en comandos separados, sobre arbol commiteado, y se
comprobo aparte que el arbol mutado compilaba (`go build ./...`).

| # | mutacion | quien se puso rojo |
|---|---|---|
| 1 | la instantanea del RGPD declara como publicacion la fecha que el paquete usa | `TestNingunaVigenciaEsLaFechaDePublicacionDeSuNorma`, 11 veces (paquete + 10 obligaciones), nombrando la conflacion |
| 2 | se le quita `fecha_vigencia` a esa misma instantanea | `TestSeDiceCuantoAlcanzaElContrasteDeFechas` (1 > 0) y `TestSeCuentanLosRelojesQueNadiePuedeContrastar` (47 > 39) |
| 3 | no se llama a `comprobarLaRegla` | `TestUnaFechaDeVigorQueNoCuadraConLaReglaDeLaFuenteEsUnError` y los tres subcasos de `TestUnaReglaDeEntradaEnVigorQueNoSeSabeResolverEsUnError` |
| 4 | se lee solo la primera anotacion de cada hito (la regresion que produjo la medida falsa) | `TestUnaFechaQueHaceDosPapelesSeLeeEnLosDos` |
| 5 | la comprobacion de identidad del CELEX pasa siempre | `TestLaFichaDeOtraNormaNoPasaPorLaDeEsta` |
| 6 | se acepta el centinela `1001-01-01` como fecha | `TestUnaFechaCentinelaDeLaFuenteNoEsUnaFechaNiSeTiraEnSilencio` |

La 3 es la que el encargo pedia en las dos direcciones: la fecha que no cuadra da
error, **y** la regla que no se sabe resolver da error en vez de dejar pasar la
fecha plausible que venia al lado.

## Lo que un CISO sigue sin ver (pasada 3), y no es mi columna

Un CISO abre el calendario y ve una fila con su fecha. **Sigue sin poder ver de
donde sale esa fecha.** Las tres fechas y los escalones de aplicacion viven hoy en
`corpus-vigilancia/*/instantanea.json` y en la salida de la herramienta de
ingesta; lo unico que llega a una pantalla es el `vigencia.desde` del paquete, una
fecha suelta sin procedencia. Con el dato ya en la instantanea, la fila podria
decir *«te obliga desde el 11-09-2026, que es el escalon del art. 71.2 del
Reglamento 2024/2847, publicado el 20-11-2024»*, y hoy no lo dice.

`superficies/calendario/` no es la columna de este frente, asi que queda anotado y
no tocado. Es el hallazgo mas caro de la pasada 3 y es de D11.

## Dos instantaneas que no usa ningun paquete, contadas

`urn:eu:reg:2014:910` (eIDAS base, cuyo paquete es el 2024/1183 que lo modifica) y
`urn:eu:reg:2026:1744` (el omnibus, que no tiene paquete propio). Las dos son
legitimas: son actos que modifican a otro. Se cuentan en
`TestUnPaqueteYSuInstantaneaLlamanIgualALaMismaNorma` porque es **la direccion que
nadie recorria** (invariante 7), y porque el dia que ese numero crezca sin
explicacion lo que hay detras es una norma ingerida y olvidada.

## Vigencias de `paquetes/` que NO se han corregido

Ninguna esta mal. La revision de las 336 fechas del corpus (paquete y obligacion,
en las 17 normas con instantanea) no encontro **ni una** conflacion: ninguna es la
fecha de publicacion ni la del acto de su norma.

Lo que si conviene mirar, y no es un error:

| paquete | escribe | es | nota |
|---|---|---|---|
| `rgpd` | 2018-05-25 | la fecha de APLICACION (art. 99), no la de vigor (2016-05-24) | correcto para plazum: lo que obliga es la aplicacion |
| `dora` | 2025-01-17 | idem (art. 64), vigor 2023-01-16 | idem |
| `mdr` | 2021-05-26 | idem (art. 123.2), vigor 2017-05-25 | idem |
| `mica` | 2024-12-30 | idem (art. 149.2), vigor 2023-06-29 | idem |

Las cuatro son la eleccion correcta y ahora se puede demostrar que lo son, porque
la instantanea trae las dos fechas y se ve cual se eligio. Antes solo se veia una
fecha suelta que no se podia contrastar con nada.
