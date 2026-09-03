# Instantánea de plazum, 4 de septiembre de 2026

> **Para qué sirve este documento.** Es la foto del proyecto para alguien de fuera que no ha leído nada más. Es autocontenido: no hace falta abrir el repositorio para entenderlo, y todos los números salen de ejecutar comandos, no de memoria.
>
> **Qué NO es.** No es una presentación. La autoevaluación del final está hecha para que se pueda discutir, así que las notas bajas están donde están y sin acolchar.
>
> **Por qué está rehecha entera y no retocada.** La versión anterior era del 26-08-2026 y `docs/diseno.md` §14 dejó escrita la regla que gobierna este fichero: *«se vuelve a hacer entera o no se hace; retocarle una celda la convertiría en una foto que finge estar viva»*. Se ha hecho entera. Todos los números de abajo se han vuelto a medir el 04-09-2026, y **cinco de los que traía la foto anterior habían dejado de ser ciertos**: dos dependencias (hoy cero), 1.199 casos de test (hoy 2.331), 31 paquetes de corpus (hoy 33), 4 paquetes con contenido (hoy 21) y 8 relojes escritos (hoy 230).
>
> **Y una que no se copia aquí a propósito: el peso de cada dimensión.** Vive en `docs/diseno.md` §14 y en ningún otro sitio. La foto anterior llevaba una copia, con los pesos anteriores a D-20, y esa copia es exactamente el tipo de segunda lista que se queda vieja.

## Qué es esto

Un GRC de continuidad de cumplimiento. Motor determinista de obligaciones con **reloj legal**, corpus normativo como **paquetes de datos firmados**, y un **expediente verificable offline**: un tercero con el fichero y el binario, sin red y sin fiarse de quien lo emitió, recalcula la aplicabilidad, los plazos y los estados de control y obtiene lo mismo, o le dicen dónde no coincide.

Go puro, AGPL-3.0, una persona construyéndolo por etapas. **El repositorio es público desde el 03-09-2026** y no hay ninguna release publicada.

## Los números, medidos hoy

| | |
|---|---|
| Paquetes Go | **60** |
| Líneas de Go de producción | **56.729** |
| Líneas de Go de test | **69.919** |
| Otras líneas (datos del corpus, workflows, documentación), en 470 ficheros | 91.893 |
| Casos de test ejecutados (subtests incluidos) | **2.331**, contra un suelo declarado de 700 |
| Cobertura de sentencias del núcleo | **89,4 %**, contra una puerta dura de 85 % |
| Dependencias externas | **0** |
| Binario Linux (`-s -w -trimpath`) | **11.788.580 bytes**, 11,2 MB de un presupuesto de 25 |
| Arranque hasta la primera respuesta | **101 ms** de un presupuesto de 3.000 |
| RAM de `plazum serve` tras 200 peticiones | **6 MB** de un presupuesto de 256 |
| Tiempo hasta el valor, un comando en un directorio vacío (`plazum demo`) | **86 ms** |
| Tiempo hasta el valor sobre el **camino guiado completo** | **15 m 51 s** de un presupuesto de 15 m, **NO cumple** |
| Auditorías de accesibilidad con cero violaciones | **26** (13 rutas × 2 idiomas), más 1 control negativo con 5 violaciones |
| Paquetes de corpus | **33**, de los cuales **21 con obligaciones escritas** |
| Obligaciones con reloj escritas | **230** |
| Puertas de CI | **24**, en 12 workflows, todas en verde en `main` |

### `go list -m all`

```
github.com/marcosmatalab/plazum
```

**Una sola línea, que es el cambio más visible desde la foto anterior.** Aquella tenía dos dependencias (`digitorus/pkcs7` y `digitorus/timestamp`); hoy `go.mod` no tiene ni una directiva `require` y `go.sum` está vacío. La criptografía RFC 3161 vive vendorizada bajo `adaptadores/tsa/internal/`, con su fuzzing propio y una puerta de procedencia que recalcula el `sha256` de cada fichero ajeno. Lo vigila `TestElBinarioNoLlevaNingunaDependenciaExterna`: el día que entre la primera hay que cambiar ese test a propósito.

## El árbol, a dos niveles

```
(raiz)/          go.mod, ETAPAS.md, CLAUDE.md y 31 tests de arquitectura y de plan
.github/         puerta.sh, presupuesto.sh, frontera.sh, workflows/ (12)
adaptadores/     actualizador canal catalogo diagnostico escalador latido oidc
                 plantilla scim secretos tsa usuarios
cmd/             plazum
docs/            30 documentos, entre ellos guia.md (fuente única del plan), diseno.md,
                 modelo-de-amenaza.md, censo-relojes.md, decisiones.md, pendientes.md,
                 marcador.md
evals/           conjuntos dorados de IA (etapa 5, vacío)
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

| Paquete | Cobertura | | Paquete | Cobertura |
|---|---|---|---|---|
| `nucleo/expediente` | **97,7 %** | | `superficies/pantallas` | 95,8 % |
| `nucleo/historia` | 97,6 % | | `superficies/camino` | 94,1 % |
| `nucleo/escalado` | 96,5 % | | `superficies/export` | 94,4 % |
| `nucleo/metrica` | 96,2 % | | `superficies/calendario` | 90,8 % |
| `nucleo/ventana` | 94,5 % | | `superficies/acta` | 90,0 % |
| `nucleo/aplicabilidad` | 93,1 % | | `superficies/escalado` | 87,9 % |
| `nucleo/estado` | 92,9 % | | `superficies/scim` | 86,5 % |
| `nucleo/blobs` | 91,7 % | | `superficies/serve` | 83,8 % |
| `nucleo/pantalla` | 91,4 % | | **`superficies/uar`** | **78,3 %** |
| `nucleo/acta` | 91,0 % | | `adaptadores/plantilla` | 92,2 % |
| `nucleo/censo` | 89,9 % | | `adaptadores/catalogo` | 90,2 % |
| `nucleo/accesos` | 89,6 % | | `adaptadores/escalador` | 89,7 % |
| `nucleo/perimetro` | 89,5 % | | `adaptadores/secretos` | 83,3 % |
| `nucleo/ledger` | 86,8 % | | `adaptadores/latido` | 82,4 % |
| `nucleo/auditoria` | 86,4 % | | `adaptadores/tsa` | 81,8 % |
| `nucleo/certificado` | 85,7 % | | `adaptadores/oidc` | 81,5 % |
| `nucleo/corpus` | 85,6 % | | `adaptadores/usuarios` | 80,0 % |
| `nucleo/incidente` | 85,5 % | | `adaptadores/scim` | 79,5 % |
| `nucleo/estricto` | 84,2 % | | `adaptadores/diagnostico` | 76,8 % |
| `nucleo/entregable` | 81,6 % | | `adaptadores/canal` | 77,4 % |
| `cmd/plazum` | 70,6 % | | `adaptadores/actualizador` | 75,3 % |
| `puertos/contrato` | 66,0 % | | `adaptadores/tsa/internal/pkcs7` | 72,9 % |
| `herramientas/cribamarca` | 89,5 % | | `herramientas/ensayocopia` | 76,9 % |
| `herramientas/cotejapkcs7` | 82,5 % | | `herramientas/generardemo` | 68,6 % |
| `herramientas/ingestanorma` | 82,3 % | | **`herramientas/sellardemo`** | **24,5 %** |

**Los dos números bajos, dichos y no escondidos.** `sellardemo` sale a internet a sellar el demo contra una TSA real y lo que tiene sin cubrir es exactamente la parte que sale a internet, que no se prueba en CI a propósito. Y `superficies/uar` al 78,3 % es el que hay que mirar: es la primera superficie que muta y la única cuyo contenido son nombres de personas. **`adaptadores/scim`, que en la foto anterior estaba al 46,4 % y era el que preocupaba, está hoy al 79,5 % con puerta propia al 75 %.**

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

**230 obligaciones con reloj escritas**, contadas por `estado_del_plan_test.go` con el cargador del producto. Los más grandes: `ens` (133 obligaciones, 12 relojes), `iso27001` (132 y 9), `iso42001` (51 y 10), `dora` (30 y 30), `ai-act` (25 y 20), `cra` (24 y 24), `nis2-ue` (12 y 12).

**Y la cifra que importa para la venta, computada por puerta y no a mano: 51,4 % de cobertura estricta de la v1**, o sea 73 relojes *cuyo intervalo lo escribe la norma* sobre 142 puntos censados, más 69 rituales de plazum que salen al lado y **nunca dentro** del porcentaje. **7 de los 15 marcos de la v1 quedan fuera de ese denominador**, con su motivo escrito, y para ellos la cifra honesta es sin denominador: 28 rituales y 37 relojes escritos. El detalle entero está en el bloque `cobertura-v1` del `README.md`.

**Los dos huecos del corpus, con su cardinal:** **39 relojes** cuya vigencia nadie puede contrastar porque su norma no tiene instantánea guardada (los seis referenciales y el demo), y **17 vigencias** que no casan con ninguna de las tres fechas que declara su fuente y que hay que poder explicar una a una.

---

# Autoevaluación de las 17 dimensiones

**El criterio, y es distinto del de `docs/diseno.md`.** Allí la nota es de **diseño**: si la decisión está tomada, es defendible y tiene un test falsable especificado. Aquí la nota es de **lo construido y medido hoy**. Son dos preguntas distintas y por eso las dos columnas se enseñan juntas.

Una dimensión con el diseño cerrado y cero código sacará un 1,5, y eso **no es un defecto del diseño**: es que le toca en la etapa 5 o en la 7 y estamos entrando en la v1.

**Los pesos no están en esta tabla a propósito.** Viven en `docs/diseno.md` §14, en un solo sitio, y la ponderación se computa desde allí. La foto anterior los copiaba y la copia se quedó con los pesos anteriores a D-20.

| # | Dimensión | Diseño | Hoy | Qué sostiene la nota de hoy |
|---|---|---|---|---|
| D1 | Modelo de obligación y temporalidad | 9,7 | **9,0** | **230 relojes** escritos con dorados derivados del texto legal, calendario que los publica con `--ics`, régimen de cómputo por norma y ley de conservación. **Las ocho primitivas del motor están construidas Y encendidas para el corpus**: `primitivas_alcanzables_test.go` informa que hoy no hay ninguna apagada ni sin cablear. No llega más arriba por dos huecos contados: 39 relojes cuya vigencia nadie puede contrastar y 17 vigencias que no casan con la fecha de su fuente |
| D2 | Determinismo y reproducibilidad | 9,6 | **9,3** | 14 ataques al expediente, modelo de amenaza escrito con lo que NO demuestra, demo que verifica offline con sello real (`VERIFICADO` sobre 8 comprobaciones desde un directorio vacío), imagen Docker con reproducibilidad medida. Residual: sin prueba de consistencia entre checkpoints |
| D3 | Cobertura por estratos y calendarios país | 9,5 | **6,5** | De **4 paquetes con contenido de 31 a 21 de 33**, con 230 relojes y un **51,4 %** de cobertura estricta de la v1 computado por puerta en las dos direcciones. Sigue siendo la nota baja de las que importan: **7 de los 15** marcos de la v1 están fuera del denominador y hay **46 relojes identificados y sin escribir** que ningún censo cuenta todavía |
| D4 | Implantación e2e, 5 clases con facetas | 9,6 | **7,5** | `clase_e2e` con facetas construido y `plazum cobertura` lo publica marco a marco. Sube medio punto y no más: el mecanismo no ha cambiado, lo que ha cambiado es su base, de 4 paquetes medibles a 21 |
| D5 | Conectores WASM con conformidad | 9,5 | **2,0** | Nada construido. Es la etapa 6 |
| D6 | Continuidad: certificado, escalado, silencio | 9,5 | **7,5** | Certificado, estados, escalado y latido construidos, con `superficies/escalado` ya auditada por axe. **No sube**, porque la razón que la bajaba sigue en pie palabra por palabra: falta el planificador propio, y hoy quien apunta que ha corrido es un temporizador del operador (`plazum latido` manda a tu cron) |
| D7 | Evidencia y valor probatorio | 9,7 | **9,5** | **Lo más terminado del producto.** Ledger v2 con compromiso de clave, lápidas, borrado legal que no blanquea, 14 ataques, ensayo de restauración que corre nueve veces (una sana y ocho copias rotas) y termina verificando la cadena, y el modelo de amenaza que dice qué queda fuera |
| D8 | Riesgos con MAGERIT | 9,5 | **1,5** | Nada construido. Es la etapa 7 |
| D9 | Ligereza y huella | 9,8 | **9,7** | Binario 11,2 MB de 25, arranque 101 ms de 3.000, 6 MB de RAM tras 200 peticiones de 256, imagen `scratch` sin intérprete de órdenes, **cero dependencias externas**. Todo medido en CI, con la puerta viéndose fallar en cada ejecución contra un límite imposible |
| D10 | Instalación local y datacenter | 9,6 | **8,5** | Docker, matriz en tres sistemas arrancando el binario, Litestream documentado con ensayo de restauración, OIDC y SCIM con cero dependencias. **No sube**: sigue faltando el tramo alto (Postgres) y publicar la imagen |
| D11 | Intuitividad y guiado | 9,5 | **8,0** | **Ya se puede guardar**, que era lo que la bajaba: `superficies/uar` escribe decisiones, `/entrar` y `/primer-admin` son POST, y **los seis pasos del camino guiado contestan 200** medidos contra el binario desde un directorio vacío. 26 auditorías de axe-core con cero violaciones. No sube más porque **3 de sus 5 puertas propias siguen abiertas, cada una con su cardinal**: 2 órdenes de terminal en el camino, 5 cifras huérfanas de enlace de 14, y 51 segundos de más sobre el TTFV |
| D12 | IA verificable | 9,6 | **1,5** | Nada construido. El interruptor (`PLAZUM_SIN_IA`, con su paso de CI sobre la suite entera) existe desde antes que el adaptador, a propósito |
| D13 | Extensibilidad | 10,0 | **9,8** | Una norma nueva no toca código: las reglas de aplicabilidad las declara el paquete en Datalog estratificado, y el test AST que prohíbe normas cableadas vigila también los `_test.go` desde que se descubrió que ése era el agujero |
| D14 | Open core self-serve | 9,5 | **1,5** | Nada construido. Licencia, cobro y carpeta de compras son etapas 3 y 8 |
| D15 | Legalidad del corpus | 9,6 | **9,0** | Tres techos de texto por tipo de campo, `licencia_fuente` de vocabulario cerrado cruzado con la clase, **lista negra ejecutable**, atribución del DOUE mostrada en producto, y el guardia de arranque que impide que el catálogo de i18n transporte texto normativo |
| D16 | Cross-framework computado | 9,5 | **1,5** | Nada construido. Es la etapa 3 |
| D17 | Autoservicio radical | 9,6 | **6,0** | `demo`, `doctor` y `update` con vuelta atrás comprobada, y un TTFV medido en CI contra el binario. **No sube**: sigue faltando la carpeta de compras y todo el autoservicio de licencia, que es la mitad de esta dimensión |

**GLOBAL ponderado, sobre los pesos de `docs/diseno.md` §14: diseño 9,60, hoy 6,41.** La aritmética entera, y el subíndice de plataforma que se publica a su lado, están en `docs/marcador.md`, con una puerta que los recomputa del dato.

## Cómo se lee ese 6,41

**No es una nota de calidad, es una nota de avance.** El diseño está en 9,60 y no ha bajado; lo que mide la columna de hoy es cuánto de ese diseño existe. Venía de **6,13** el 26-08-2026, o sea **+0,29 en nueve días**, y ese es el número que mide trabajo.

Las cinco dimensiones que más separan las dos columnas son **D12 (IA), D8 (MAGERIT), D14 (open core), D16 (cross-framework) y D5 (conectores)**, y las cinco tienen la misma explicación: **les toca más adelante**. Sumadas valen **31 puntos de peso de 109**, o sea que por sí solas explican más de la mitad de la diferencia. Su nota conjunta es **1,61**, y es exactamente la misma que hace nueve días: **en esta campaña no se movió ni una décima de las cinco**.

**La que sí era el aviso, D3, se ha movido de verdad**: de 4,5 a 6,5, con los paquetes con contenido pasando de 4 a 21. Sigue siendo la más baja de las que no están esperando a una etapa lejana.

Y la lectura optimista, que también es real: **las tres dimensiones más terminadas son D13 (9,8), D9 (9,7) y D7 (9,5)**, y son exactamente las tres que un competidor no puede copiar sin rehacer su producto: la extensibilidad sin tocar código, la huella y el valor probatorio.

## Cómo reproducir estos números

```bash
./comprobar.sh                              # las 24 puertas con su recuento
go test ./... -count=1 -cover               # cobertura por paquete
go list -m all                              # una linea: cero dependencias
go build -ldflags="-s -w" -trimpath -o plazum ./cmd/plazum
gh run list --branch main                   # las puertas de CI
go test . -run TestElEstadoDelPlanLoComputaUnTestYNoUnaPersona -v   # casillas y relojes
go test . -run TestElSubindiceDePlataformaLoComputaUnTestYNoUnaPersona -v
```

Los presupuestos (binario, arranque, RAM, TTFV, axe-core) se miden en `etapa2-ttfv.yml` y `etapa2-accesibilidad.yml`, y sus valores salen del log de la ejecución sobre `main`, no de esta máquina.
