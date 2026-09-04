# Instantánea de plazum, 4 de septiembre de 2026

> **Para qué sirve este documento.** Es la foto del proyecto para alguien de fuera que no ha leído nada más. Es autocontenido: no hace falta abrir el repositorio para entenderlo, y todos los números salen de ejecutar comandos, no de memoria.
>
> **Qué NO es.** No es una presentación. La autoevaluación del final está hecha para que se pueda discutir, así que las notas bajas están donde están y sin acolchar.
>
> **Por qué está rehecha entera y no retocada.** `docs/diseno.md` §14 dejó escrita la regla que gobierna este fichero: *«se vuelve a hacer entera o no se hace; retocarle una celda la convertiría en una foto que finge estar viva»*.
>
> **Y esta foto se ha tomado DOS VECES el mismo día, que es la lección que trae.** La primera fue el 04-09-2026 entre las 00:47 y las 02:22. Las cuatro rebanadas del tramo 3 aterrizaron entre las **14:47 y las 15:35**, o sea que aquella foto quedó falsa doce horas después de tomarse, y con ella el marcador que la copia: se publicó y se citó durante un tramo entero un subíndice medido **antes** del trabajo que decía medir. Ésta es la de la tarde, y todos sus números se han vuelto a medir después de que aterrizara todo.
>
> **Los que habían dejado de ser ciertos en doce horas**, contados: 230 relojes (hoy **252**), 51,4 % de cobertura de la v1 (hoy **56,7 %**), 24 puertas de CI (hoy **25**), 2.331 casos de test (hoy **2.748**), 60 paquetes Go (hoy **65**), un binario de 11.788.580 bytes (hoy **12.714.146**), un TTFV de 15m51s (hoy **20m27s**) y una tabla de cobertura a la que le faltaban **cuatro paquetes enteros** que el tramo había creado.
>
> **Y lo que salió de arreglarlo**, porque es más importante que cualquiera de esas cifras: la puerta del marcador comparaba tres documentos entre sí y ninguno miraba el árbol. Coherencia y frescura no son lo mismo. Desde hoy `instantanea_test.go` ata los cardinales de esta foto a quien los computa del repositorio.
>
> **Y una que no se copia aquí a propósito: el peso de cada dimensión.** Vive en `docs/diseno.md` §14 y en ningún otro sitio. La foto anterior llevaba una copia, con los pesos anteriores a D-20, y esa copia es exactamente el tipo de segunda lista que se queda vieja.

## Qué es esto

Un GRC de continuidad de cumplimiento. Motor determinista de obligaciones con **reloj legal**, corpus normativo como **paquetes de datos firmados**, y un **expediente verificable offline**: un tercero con el fichero y el binario, sin red y sin fiarse de quien lo emitió, recalcula la aplicabilidad, los plazos y los estados de control y obtiene lo mismo, o le dicen dónde no coincide.

Go puro, AGPL-3.0, una persona construyéndolo por etapas. **El repositorio es público desde el 03-09-2026** y no hay ninguna release publicada.

## Los números, medidos hoy

| | |
|---|---|
| Paquetes Go | **65** |
| Líneas de Go de producción | **64.079** |
| Líneas de Go de test | **83.634** |
| Otras líneas (datos del corpus, workflows, documentación), en 532 ficheros | 110.933 |
| Casos de test ejecutados (subtests incluidos) | **2.748**, contra un suelo declarado de 800 |
| Cobertura de sentencias del núcleo | **89,4 %**, contra una puerta dura de 85 % |
| Dependencias externas | **0** |
| Binario Linux (`-s -w -trimpath`) | **12.714.146 bytes**, 12,1 MB de un presupuesto de 25. **Sube 0,9 MB desde la foto de la mañana** y se dice por qué: la release ahora lleva el corpus dentro, que es lo que hace que una máquina limpia pase de 3 relojes a 222 |
| Arranque hasta la primera respuesta | **101 ms** de un presupuesto de 3.000 |
| RAM de `plazum serve` tras 200 peticiones | **6 MB** de un presupuesto de 256 |
| Tiempo hasta el valor, un comando en un directorio vacío (`plazum demo`) | **86 ms** |
| Tiempo hasta el valor sobre el **camino guiado completo** | **20 m 27 s** de un presupuesto de 15 m, **NO cumple, y se aleja**. El número SUBE 4m36s respecto a la foto de la mañana y **no es que el producto haya empeorado: es que la medida dejó de ser ciega**. Antes no cobraba las órdenes de terminal de los estados vacíos; ahora sí, y son **7m30s de 20m27s, el 37 %**, que es el cuello derivado del reparto y no escrito a mano |
| Auditorías de accesibilidad con cero violaciones | **26** (13 rutas × 2 idiomas), más 1 control negativo con 5 violaciones |
| Paquetes de corpus | **33**, de los cuales **21 con obligaciones escritas** |
| Obligaciones con reloj escritas | **252** |
| Puertas de CI | **25**, en 12 workflows. La 25 es la suite entera con `PLAZUM_SIN_IA=1`, que convierte «el núcleo es determinista» en hecho comprobable en dos minutos |

### `go list -m all`

```
github.com/marcosmatalab/plazum
```

**Una sola línea, que es el cambio más visible desde la foto anterior.** Aquella tenía dos dependencias (`digitorus/pkcs7` y `digitorus/timestamp`); hoy `go.mod` no tiene ni una directiva `require` y **`go.sum` ya no existe**: el 04-09-2026 se sacó del índice y se puso en `.gitignore`, porque estaba rastreado y vacío, que es la cicatriz que dejó la contaminación de cosign y una invitación a que la próxima herramienta lo volviera a llenar sin que nadie lo notara. La criptografía RFC 3161 vive vendorizada bajo `adaptadores/tsa/internal/`, con su fuzzing propio y una puerta de procedencia que recalcula el `sha256` de cada fichero ajeno. Lo vigila `TestElBinarioNoLlevaNingunaDependenciaExterna`: el día que entre la primera hay que cambiar ese test a propósito.

## El árbol, a dos niveles

```
(raiz)/          go.mod, ETAPAS.md, CLAUDE.md y 36 tests de arquitectura y de plan
.github/         puerta.sh, presupuesto.sh, frontera.sh, workflows/ (12)
adaptadores/     actualizador busqueda canal catalogo diagnostico escalador ia
                 latido oidc plantilla scim secretos tsa usuarios
cmd/             plazum
docs/            39 documentos, entre ellos guia.md (fuente única del plan), diseno.md,
                 modelo-de-amenaza.md, censo-relojes.md, decisiones.md, pendientes.md,
                 marcador.md
evals/           arnés de evaluación y conjunto dorado de citas (ya NO vacío)
herramientas/    cotejapkcs7 cribamarca ensayocopia generardemo ingestanorma sellardemo
nucleo/          accesos acta aplicabilidad auditoria blobs censo certificado corpus
                 entregable escalado estado estricto expediente historia incidente
                 ledger metrica pantalla perimetro ventana
paquetes/        33 marcos (ver paquetes/CORPUS.md) + marcos-v1.json
puertos/         las 9 interfaces hexagonales, congeladas, con suites de contrato
superficies/     acta calendario camino escalado export pantallas scim serve uar
web/             la web del open core, estática, sin build
```

## Cobertura por paquete

El núcleo es lo que más cubierto está, y es donde tiene que estarlo: es lo único que un tercero recomputa. Sólo tres números tienen puerta propia (núcleo 85 %, SCIM 75 %, cribador 80 %); el resto se publican y no se vigilan, y eso también hay que saberlo.

**A la foto de la mañana le faltaban cuatro paquetes enteros**, y no por descuido: los creó el tramo unas horas después de medirla. Son `adaptadores/ia`, `adaptadores/ia/ollama`, `adaptadores/busqueda` y `evals`, o sea justo los cimientos de la IA, que era el titular del tramo. Una tabla de cobertura a la que le falta lo recién construido dice exactamente lo contrario de lo que parece decir.

| Paquete | Cobertura | | Paquete | Cobertura |
|---|---|---|---|---|
| `nucleo/expediente` | **97,7 %** | | `adaptadores/busqueda` | **97,6 %** |
| `nucleo/historia` | 97,6 % | | `adaptadores/ia/ollama` | **95,7 %** |
| `nucleo/escalado` | 96,5 % | | `superficies/pantallas` | 95,6 % |
| `nucleo/metrica` | 96,2 % | | `superficies/export` | 94,4 % |
| `nucleo/ventana` | 94,5 % | | `superficies/camino` | 94,1 % |
| `nucleo/aplicabilidad` | 93,1 % | | `evals` | **93,9 %** |
| `nucleo/estado` | 92,9 % | | `superficies/calendario` | 92,5 % |
| `nucleo/blobs` | 91,7 % | | `adaptadores/plantilla` | 92,2 % |
| `nucleo/pantalla` | 91,5 % | | `adaptadores/catalogo` | 90,2 % |
| `nucleo/acta` | 91,0 % | | `superficies/acta` | 90,0 % |
| `nucleo/censo` | 89,9 % | | `adaptadores/escalador` | 89,7 % |
| `nucleo/accesos` | 89,6 % | | `superficies/escalado` | 87,4 % |
| `nucleo/perimetro` | 89,5 % | | `adaptadores/ia` | **87,3 %** |
| `nucleo/corpus` | 87,7 % | | `superficies/scim` | 86,5 % |
| `nucleo/ledger` | 86,8 % | | `superficies/serve` | 83,8 % |
| `nucleo/auditoria` | 86,4 % | | `adaptadores/secretos` | 83,3 % |
| `nucleo/certificado` | 85,7 % | | `adaptadores/usuarios/alcances` | 83,2 % |
| `nucleo/incidente` | 85,5 % | | `adaptadores/latido` | 82,4 % |
| `nucleo/estricto` | 84,2 % | | `adaptadores/tsa` | 81,8 % |
| `nucleo/entregable` | 81,6 % | | `adaptadores/oidc` | 81,5 % |
| `cmd/plazum` | 70,3 % | | `adaptadores/usuarios` | 80,0 % |
| `puertos/contrato` | 66,0 % | | `adaptadores/scim` | 79,5 % |
| `herramientas/cribamarca` | 89,5 % | | **`superficies/uar`** | **78,3 %** |
| `herramientas/cotejapkcs7` | 82,5 % | | `adaptadores/canal` | 77,4 % |
| `herramientas/ingestanorma` | 82,3 % | | `adaptadores/diagnostico` | 76,8 % |
| `herramientas/ensayocopia` | 76,7 % | | `adaptadores/actualizador` | 75,3 % |
| `herramientas/generardemo` | 68,6 % | | `adaptadores/tsa/internal/pkcs7` | 72,9 % |
| **`herramientas/sellardemo`** | **24,5 %** | | | |

**Los dos números bajos, dichos y no escondidos.** `sellardemo` sale a internet a sellar el demo contra una TSA real y lo que tiene sin cubrir es exactamente la parte que sale a internet, que no se prueba en CI a propósito. Y `superficies/uar` al 78,3 % es el que hay que mirar: es la primera superficie que muta y la única cuyo contenido son nombres de personas.

**Y tres paquetes salen a 0,0 %, que aquí no significa «sin probar».** `puertos` son las nueve interfaces hexagonales: no tiene sentencias que cubrir. `internal/modulo` e `internal/redactado` **sí están probados, desde fuera**: sus tests viven en los paquetes que los usan (`arquitectura_test.go`, y los `frontera_test.go` de `adaptadores/canal`, `adaptadores/latido`, `adaptadores/ia` y `adaptadores/ia/ollama`, que son los centinelas del invariante 11). La cobertura por paquete no sabe contar eso, y un 0,0 % que se lee como «nadie lo prueba» es peor que no publicarlo: por eso va con su explicación al lado y no en una nota al pie.

## Estado real de las casillas

Contado sobre `ETAPAS.md` por `estado_del_plan_test.go`, no de memoria. `[~]` significa hecha salvo una parte declarada, y no cuenta ni como hecha ni como abierta.

| Etapa | Hechas | A medias | Abiertas |
|---|---|---|---|
| Semana 0 | **11** | 0 | 2 |
| Etapa 1, núcleo probatorio | **12** | 1 | 0 |
| Etapa 2, serve, UI generada y autoservicio | **14** | 1 | 1 |
| Etapa 3, corpus y venta legal | **12** | 0 | 17 |
| Etapa 4, continuidad, personas e incidentes | **5** | 0 | 8 |
| v1, la plataforma guiada | **4** | 0 | 17 |
| Etapa 5, IA verificable | 0 | 0 | 8 |
| Etapa 6, conectores | 0 | 0 | 9 |
| Etapa 7, riesgos y MAGERIT | 0 | 0 | 5 |
| Etapa 8, el dinero y la confianza | 0 | 0 | 10 |
| **TOTAL** | **58** | **2** | **77** |

**El 04-09-2026 se recorrieron las 58 cerradas una a una buscando casillas falsamente CERRADAS, y salieron cero.** El resultado está en `docs/hallazgos-barrido.md` con la evidencia de cada una, y lo que sí salió fueron **cinco casillas cuya prosa ya no describe el árbol**, que es la mitad de una casilla que nadie vigila.

## La lista de pendientes

`docs/pendientes.md` recoge cada hallazgo con su porqué y su arreglo. **El recuento de P1 y P2 no se publica aquí porque no se puede contar mecánicamente hoy** (las prioridades viven en prosa, no en una columna), y un número escrito a ojo en esta foto sería exactamente lo que esta foto no debe traer.

Lo que sí se cuenta, porque está en una tabla con una fila por caso: **la familia «guardas que no guardaban», con dieciséis entradas**. Cada una es una comprobación que parecía funcionar y no comprobaba nada, con cuánto llevaba así y cómo se cazó. **Diez de las dieciséis son emparejamientos** hechos por índice, posición u orden en vez de por una identidad firmada, que es el invariante 7 del proyecto.

## Lo que hay de corpus, sin maquillar

33 paquetes con metadatos correctos y linter legal en verde. **Con obligaciones escritas, 21**; los otros 12 son esqueletos (`cis`, `csrd`, `data-act`, `dga`, `iso22301`, `iso27002`, `iso27701`, `magerit`, `nist-800-53`, `nist-csf`, `psd2`, `stig`).

**252 obligaciones con reloj escritas**, contadas por `estado_del_plan_test.go` con el cargador del producto (eran 230 por la mañana: la rebanada de corpus del tramo 3 puso 22). Los más grandes: `ens` (133 obligaciones, 12 relojes), `iso27001` (132 y 9), `iso42001` (51 y 10), `dora` (30 y 30), `ai-act` (25 y 20), `cra` (24 y 24), `nis2-ue` (12 y 12).

**Y la cifra que importa para la venta, computada por puerta y no a mano: 56,7 % de cobertura estricta de la v1**, o sea 89 relojes *cuyo intervalo lo escribe la norma* sobre 157 puntos censados, más 69 rituales de plazum que salen al lado y **nunca dentro** del porcentaje. **7 de los 15 marcos de la v1 quedan fuera de ese denominador**, con su motivo escrito, y para ellos la cifra honesta es sin denominador: 29 rituales y 42 relojes escritos. El detalle entero está en el bloque `cobertura-v1` del `README.md`.

**Los dos huecos del corpus, con su cardinal:** **39 relojes** cuya vigencia nadie puede contrastar porque su norma no tiene instantánea guardada (los seis referenciales y el demo), y **17 vigencias** que no casan con ninguna de las tres fechas que declara su fuente y que hay que poder explicar una a una.

---

# Autoevaluación de las 17 dimensiones

**El criterio, y es distinto del de `docs/diseno.md`.** Allí la nota es de **diseño**: si la decisión está tomada, es defendible y tiene un test falsable especificado. Aquí la nota es de **lo construido y medido hoy**. Son dos preguntas distintas y por eso las dos columnas se enseñan juntas.

Una dimensión con el diseño cerrado y cero código sacará un 1,5, y eso **no es un defecto del diseño**: es que le toca en la etapa 5 o en la 7 y estamos entrando en la v1.

**Los pesos no están en esta tabla a propósito.** Viven en `docs/diseno.md` §14, en un solo sitio, y la ponderación se computa desde allí. La foto anterior los copiaba y la copia se quedó con los pesos anteriores a D-20.

| # | Dimensión | Diseño | Hoy | Qué sostiene la nota de hoy |
|---|---|---|---|---|
| D1 | Modelo de obligación y temporalidad | 9,7 | **9,0** | **230 relojes** escritos con dorados derivados del texto legal, calendario que los publica con `--ics`, régimen de cómputo por norma y ley de conservación. **Las ocho primitivas del motor están construidas Y encendidas para el corpus**: `primitivas_alcanzables_test.go` informa que hoy no hay ninguna apagada ni sin cablear. No llega más arriba por dos huecos contados: 39 relojes cuya vigencia nadie puede contrastar y 17 vigencias que no casan con la fecha de su fuente |
| D2 | Determinismo y reproducibilidad | 9,6 | **9,5** | 14 ataques al expediente, modelo de amenaza escrito con lo que NO demuestra, demo que verifica offline con sello real (`VERIFICADO` sobre 8 comprobaciones desde un directorio vacío), imagen Docker con reproducibilidad medida. **Sube porque «el núcleo es determinista» dejó de ser un eslogan**: la puerta 25 corre la **suite entera** con `PLAZUM_SIN_IA=1`, así que la afirmación se comprueba en dos minutos en vez de creerse. Residual, el mismo: sin prueba de consistencia entre checkpoints |
| D3 | Cobertura por estratos y calendarios país | 9,5 | **7,0** | De **4 paquetes con contenido de 31 a 21 de 33**, con **252 relojes** y un **56,7 %** de cobertura estricta de la v1 computado por puerta en las dos direcciones (por la mañana eran 230 y 51,4 %). Sube medio punto y **no más, porque la razón que la frenaba sigue entera**: **7 de los 15** marcos de la v1 están fuera del denominador y hay **46 relojes identificados y sin escribir** que ningún censo cuenta todavía. Y el denominador va a crecer, así que este porcentaje va a bajar |
| D4 | Implantación e2e, 5 clases con facetas | 9,6 | **7,5** | `clase_e2e` con facetas construido y `plazum cobertura` lo publica marco a marco. Sube medio punto y no más: el mecanismo no ha cambiado, lo que ha cambiado es su base, de 4 paquetes medibles a 21 |
| D5 | Conectores WASM con conformidad | 9,5 | **2,0** | Nada construido. Es la etapa 6 |
| D6 | Continuidad: certificado, escalado, silencio | 9,5 | **7,5** | Certificado, estados, escalado y latido construidos, con `superficies/escalado` ya auditada por axe. **No sube**, porque la razón que la bajaba sigue en pie palabra por palabra: falta el planificador propio, y hoy quien apunta que ha corrido es un temporizador del operador (`plazum latido` manda a tu cron) |
| D7 | Evidencia y valor probatorio | 9,7 | **9,5** | **Lo más terminado del producto.** Ledger v2 con compromiso de clave, lápidas, borrado legal que no blanquea, 14 ataques, ensayo de restauración que corre nueve veces (una sana y ocho copias rotas) y termina verificando la cadena, y el modelo de amenaza que dice qué queda fuera |
| D8 | Riesgos con MAGERIT | 9,5 | **1,5** | Nada construido. Es la etapa 7 |
| D9 | Ligereza y huella | 9,8 | **9,7** | Binario 11,2 MB de 25, arranque 101 ms de 3.000, 6 MB de RAM tras 200 peticiones de 256, imagen `scratch` sin intérprete de órdenes, **cero dependencias externas**. Todo medido en CI, con la puerta viéndose fallar en cada ejecución contra un límite imposible |
| D10 | Instalación local y datacenter | 9,6 | **9,0** | Docker, matriz en tres sistemas arrancando el binario, Litestream documentado con ensayo de restauración, OIDC y SCIM con cero dependencias. **Sube porque media razón de la nota vieja dejó de ser cierta**: la imagen está publicada, y sobre todo **la release lleva el corpus dentro**, así que una máquina limpia pasa de **3 relojes a 222** sin red y sin pasos extra, que era el agujero real de una instalación de verdad. No llega más arriba porque la otra media sigue en pie: falta el tramo alto (Postgres) |
| D11 | Intuitividad y guiado | 9,5 | **8,0** | **Ya se puede guardar**, que era lo que la bajaba: `superficies/uar` escribe decisiones, `/entrar` y `/primer-admin` son POST, y **los seis pasos del camino guiado contestan 200** medidos contra el binario desde un directorio vacío. 26 auditorías de axe-core con cero violaciones. No sube más porque **3 de sus 5 puertas propias siguen abiertas, cada una con su cardinal**: 2 órdenes de terminal en el camino, 5 cifras huérfanas de enlace de 14, y 51 segundos de más sobre el TTFV |
| D12 | IA verificable | 9,6 | **4,0** | **Era «nada construido» y ya no lo es**, que es el movimiento más grande de la tarde. Están los CIMIENTOS: `adaptadores/ia` con el **verificador de citas por hash** (521 líneas y 750 de test adversarial), el interruptor, el adaptador de Ollama fuera de proceso, `adaptadores/busqueda`, el arnés de evals con su conjunto dorado, y la puerta 25 corriendo la suite entera con `PLAZUM_SIN_IA=1`. La puerta antialucinación es mecánica y está demostrada. **No sube de 4,0 porque ninguna de las cinco piezas de adopción de `docs/ia.md` existe** (entrevista asistida, la pregunta con su consecuencia, mapeo de evidencia, plan de 30 días, extracción de metadatos): hay motor y no hay producto. **Y sigue FUERA del subíndice de plataforma**, que es lo que hay que mirar dos veces: meterla dentro ahora subiría el número publicado sin que un tercero pueda descargar y usar nada de esto |
| D13 | Extensibilidad | 10,0 | **9,8** | Una norma nueva no toca código: las reglas de aplicabilidad las declara el paquete en Datalog estratificado, y el test AST que prohíbe normas cableadas vigila también los `_test.go` desde que se descubrió que ése era el agujero |
| D14 | Open core self-serve | 9,5 | **1,5** | Nada construido. Licencia, cobro y carpeta de compras son etapas 3 y 8 |
| D15 | Legalidad del corpus | 9,6 | **9,0** | Tres techos de texto por tipo de campo, `licencia_fuente` de vocabulario cerrado cruzado con la clase, **lista negra ejecutable**, atribución del DOUE mostrada en producto, y el guardia de arranque que impide que el catálogo de i18n transporte texto normativo |
| D16 | Cross-framework computado | 9,5 | **1,5** | Nada construido. Es la etapa 3 |
| D17 | Autoservicio radical | 9,6 | **6,0** | `demo`, `doctor` y `update` con vuelta atrás comprobada, y un TTFV medido en CI contra el binario. **No sube**: sigue faltando la carpeta de compras y todo el autoservicio de licencia, que es la mitad de esta dimensión |

**GLOBAL ponderado, sobre los pesos de `docs/diseno.md` §14: diseño 9,60, hoy 6,66.** La aritmética entera, y el subíndice de plataforma que se publica a su lado, están en `docs/marcador.md`, con una puerta que los recomputa del dato.

## Cómo se lee ese 6,66

**No es una nota de calidad, es una nota de avance.** El diseño está en 9,60 y no ha bajado; lo que mide la columna de hoy es cuánto de ese diseño existe. Venía de **6,13** el 26-08-2026, o sea **+0,54 en nueve días**, y ese es el número que mide trabajo.

**Y por la mañana marcaba 6,41.** La diferencia no es que se haya construido nada entre las dos fotos: es que la primera se tomó antes de que aterrizara el tramo. Se dice porque un número que se mueve un cuarto de punto sin trabajo detrás merece explicación, y la explicación es que el instrumento estaba parado.

Las cinco dimensiones que más separan las dos columnas son **D12 (IA), D8 (MAGERIT), D14 (open core), D16 (cross-framework) y D5 (conectores)**, y sumadas valen **31 puntos de peso de 109**. Su nota conjunta es **2,26**, y aquí hay que corregir lo que decía la foto anterior: aquella afirmaba que las cinco llevaban nueve días **clavadas en 1,61 sin moverse ni una décima**, y era cierto cuando se escribió. Hoy no: **D12 pasa de 1,5 a 4,0** porque los cimientos de la IA existen, y las excluidas se mueven **64 céntimos**. Las otras cuatro sí siguen exactamente donde estaban.

**El 74 % del numerador que añadió este tramo cayó FUERA del subíndice de plataforma**, precisamente por D12. Es la cifra que explica por qué el subíndice sube nueve céntimos y el global veinticinco, y por qué eso no es un defecto: un subíndice que salta mientras el global no se mueve es la firma de una redefinición, y esto es lo contrario.

**La que sí era el aviso, D3, se ha movido de verdad**: de 4,5 a **7,0** desde el 26-08, con los paquetes con contenido pasando de 4 a 21 y la cobertura estricta de la v1 de 0 a 56,7 %. Sigue siendo la más baja de las que no están esperando a una etapa lejana.

**Y la que NO se movió teniendo excusa, que es la que hay que leer con atención: D11.** Detrás tiene la capa visual entera y las cifras huérfanas bajando de 5 a 1, y se queda en 8,0 porque su razón escrita sigue en pie y **una de sus tres puertas se ha alejado**: el TTFV está en **20m27s** contra un presupuesto de **15m0s**. No es que el producto haya empeorado; es que la medida dejó de ser ciega y empezó a cobrar seis órdenes de terminal que no veía. El hueco publicado era de **51 segundos** y el real es de **5m27s**, o sea que la nota se sostenía con un número **seis veces por debajo**. **Ésta es la fila que decide la fecha de la v1.**

Y la lectura optimista, que también es real: **las tres dimensiones más terminadas son D13 (9,8), D9 (9,7) y D7 (9,5)**, y son exactamente las tres que un competidor no puede copiar sin rehacer su producto: la extensibilidad sin tocar código, la huella y el valor probatorio.

## Cómo reproducir estos números

```bash
./comprobar.sh                              # las 25 puertas con su recuento
go test ./... -count=1 -cover               # cobertura por paquete
go list -m all                              # una linea: cero dependencias
go build -ldflags="-s -w" -trimpath -o plazum ./cmd/plazum
gh run list --branch main                   # las puertas de CI
go test . -run TestElEstadoDelPlanLoComputaUnTestYNoUnaPersona -v   # casillas y relojes
go test . -run TestElSubindiceDePlataformaLoComputaUnTestYNoUnaPersona -v
go test . -run TestLaInstantaneaNoPublicaCardinalesQueElArbolYaDesmiente -v  # que esta foto no miente
go test . -run TestElTamanoPublicadoDelBinarioEsElDeHoy -v                   # el tamano del binario
```

Los presupuestos (binario, arranque, RAM, TTFV, axe-core) se miden en `etapa2-ttfv.yml` y `etapa2-accesibilidad.yml`, y sus valores salen del log de la ejecución sobre `main`, no de esta máquina.
