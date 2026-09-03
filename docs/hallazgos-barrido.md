# El barrido de las 58 casillas CERRADAS (04-09-2026)

> **Qué se buscaba, y es lo contrario de lo que buscó el barrido anterior.** El del 04-09-2026 por la mañana recorrió las **77 abiertas** buscando casillas falsamente ABIERTAS y encontró cero. Y había una falsamente **CERRADA**: el camino guiado, marcado mientras tres de sus seis pasos contestaban 401. Una casilla abierta de más hace planificar trabajo que no existe; **una cerrada de más hace medir sobre una maqueta y creerse el número**. Este barrido mira hacia el lado que faltaba.

## El resultado, con su cardinal

| | |
|---|---|
| Casillas cerradas recorridas | **58** |
| Verificadas ciertas hoy, con su evidencia | **58** |
| **Desmarcadas** | **0** |
| Cuya afirmación no se puede comprobar por ejecución | **1** (la 58, el workflow de release) |
| Con **prosa que ya no describe el árbol** | **5** |

**Y el cero se dice en voz alta, porque un cero no es una buena noticia por sí solo.** Hay dos lecturas y las dos son ciertas a la vez:

1. La buena: las casillas de este repositorio están escritas como puertas, y una puerta que se cierra con un test no envejece en silencio porque el repositorio es su fuente de verdad. **54 de las 58 se verificaron con una puerta, con el binario ejecutado o con la API de GitHub**, no leyendo el árbol. **Las cuatro restantes descansan sólo en que un fichero está donde dice**, y se nombran para que se sepa cuáles son: la de los cinco documentos de gobierno, la del texto canónico de la AGPL, la de la decisión de marca y la del workflow de release.
2. La incómoda: **este barrido llega un día después del que encontró la falsamente cerrada**, o sea sobre un árbol que alguien acaba de corregir. Un cero sobre un tablero recién barrido dice menos que un cero sobre uno que lleva un mes sin mirarse. **La próxima vez que valga la pena hacerlo es dentro de varias semanas, no mañana.**

## Cómo se miró cada una

Tres reglas, y son las que separan esto de una lectura:

- **La prosa de una casilla no es evidencia de sí misma.** Donde la casilla dice «con test», se comprobó que el test existe **y** que corre dentro de una puerta con recuento. `go test -run` a mano no cuenta.
- **Las casillas cuyo estado vive FUERA del repositorio se miraron con más cuidado, no con menos**, porque nadie las relee. Son las tres de plataforma (repo público, private vulnerability reporting, CodeQL), y se comprobaron contra la API de GitHub, no contra su propia prosa.
- **«No se pudo comprobar» y «es falsa» son cosas distintas**, y confundirlas es el invariante 8 aplicado al barrido: la nada peligrosa es la que se lee como inocua.

## Lo que sí salió: cinco casillas cerradas cuya prosa ha caducado

Ninguna hace falsa a su casilla, y por eso ninguna se desmarca. Pero las cinco afirman algo que hoy no es cierto, y **una casilla tiene dos mitades —el corchete y la prosa— y sólo la primera tiene puerta**. Es la misma lección que la casilla de «hacer público el repo» dejó escrita el 03-09-2026, y vuelve a aparecer entera.

| Casilla | Lo que dice | Lo que se mide hoy |
|---|---|---|
| Reactivar el workflow codeql | «había 2 alertas, una `go/bad-redirect-check` de severidad media **ABIERTA**» | **0 alertas abiertas**: la nº 1 quedó `fixed` el 28-08-2026 y la nº 2 el 03-09-2026 a las 18:34. La deuda que la casilla declara (ninguna puerta lee las alertas) **sigue viva** |
| Workflow de release | «**4 plataformas**» | la matriz da **6 destinos**: 3 sistemas × 2 arquitecturas (amd64 y arm64) |
| Los 30 marcos montados como paquetes | «los **30** marcos» y «**12 dorados**» | **33** paquetes, **21** con obligaciones escritas, **230** relojes |
| TTFV sintético, axe-core y presupuestos | «axe-core sobre **16 auditorías** (8 rutas por es y en)», «arranque 637 ms», «binario 9,45 MB» | **26 auditorías** (13 rutas × 2 idiomas) más 1 control negativo; arranque **101 ms**; binario **11.788.580 bytes** |
| UAR con snapshot firmado | «**Hueco declarado**: la puerta de axe-core NO la audita todavía, porque exige 200 y esta pantalla contesta 401 sin sesión» | **la audita desde el 04-09-2026**: `/uar/` sale en el log con `[es]` y `[en]`, 26 reglas en verde y cero violaciones. El hueco se cerró y su declaración se quedó |

Las cinco se corrigen en `ETAPAS.md` en el mismo commit que este documento. **La corrección no es el hallazgo: el hallazgo es que hicieron falta cinco correcciones y ninguna la iba a encontrar nadie**, porque la prosa de una casilla cerrada no la vuelve a leer nadie por definición.

## La única que no se puede comprobar por ejecución

**«Workflow de release: 4 plataformas, SHA256SUMS, SBOM CycloneDX, firma keyless cosign».** El fichero existe y declara las cuatro cosas; se leyó entero y están. Lo que no existe es una sola prueba de que funcione:

```
$ gh run list --workflow release.yml --limit 10
(sin salida)
$ git ls-remote --tags origin
(sin salida)
```

**Cero ejecuciones en toda la historia del repositorio y cero etiquetas en el remoto.** La casilla es cierta al nivel que afirma —el workflow está escrito— y **se queda marcada**, porque la publicación es el HITO v0.2, que está `[~]` y declarado como diferido a propósito. Pero la casilla más cara del repositorio, la que produce lo que un tercero descarga y verifica, es la única que **nunca se ha visto correr**, y eso encaja mal con la regla de la casa de que una puerta que nunca se ha visto fallar no es una puerta.

**Y hay un cambio de estado que nadie ha anotado:** el candado `.github/marca-congelada` **ya no existe** (se borró el 26-08-2026), así que el trabajo `candado` de `release.yml` resolvería hoy «Esta ejecución PUBLICA». La prosa de la casilla de la imagen Docker (`[~]`, no es de las 58) sigue diciendo que el candado es lo que impide subir la imagen. No lo es desde hace nueve días. Va aquí y no en la tabla de arriba porque esa casilla no está entre las cerradas.

## La evidencia, casilla a casilla

Abreviaturas: **P** = verificada dentro de una puerta con recuento (`./comprobar.sh`, 24 puertas leídas de los workflows, 21 ejecutadas aquí, 2.331 casos); **CI** = verificada en el log de la ejecución sobre `main`; **API** = verificada contra la API de GitHub; **BIN** = verificada ejecutando el binario; **A** = verificada leyendo el árbol.

### Semana 0, once cerradas

| Casilla | Cómo | Evidencia |
|---|---|---|
| Estructura del repo | A | los cinco directorios existen; `arquitectura_test.go` y `frontera_test.go` los vigilan por AST |
| Núcleo construido y en verde | P | 20 paquetes bajo `nucleo/`; puerta «cobertura del nucleo», 823 casos, 89,4 % |
| Tests de arquitectura | P | `arquitectura_test.go`, `extensibilidad_test.go`, `paquetes_test.go` dentro de la puerta «suite completa» |
| CLAUDE.md, DEPENDENCIAS.md, SECURITY.md, CONTRIBUTING.md, CLA.md | A | los cinco ficheros están |
| CI completo | CI | `ci.yml`: formato, vet, build, cobertura del núcleo con puerta dura 85 %, `govulncheck@v1.7.0` y `gosec@v2.28.0` bloqueantes con versión fijada, `codeql.yml`, `.github/dependabot.yml` |
| LICENSE AGPL-3.0 canónica | A | 678 líneas, cabecera «GNU AFFERO GENERAL PUBLIC LICENSE, Version 3, 19 November 2007» |
| Decisión de marca | A | `docs/marca.md` y D-4; el producto se llama plazum de punta a punta |
| Hacer público el repo | **API** | `gh api repos/marcosmatalab/plazum --jq .private` devuelve `false` |
| Private vulnerability reporting | **API** | `gh api .../private-vulnerability-reporting` devuelve `{"enabled":true}` |
| Reactivar codeql | **API** | último análisis 03-09-2026 20:52 UTC, `tool: CodeQL`. Prosa caducada, arriba |
| gosec bloqueante | CI | ninguna aparición de `continue-on-error` en los 12 workflows; 114 `#nosec` con motivo, vigilados por `supresiones_test.go` |

### Etapa 1, doce cerradas

Las doce son de código y las doce corren dentro de la puerta «suite completa». Comprobado además, una a una: `nucleo/ledger/v2.go` con `contenidoFirmado()` y `Lapida` (ledger v2, lápidas, keystore); `nucleo/blobs`; `nucleo/historia` con `EstadoEn`, `Ventana`, `PrimerConocimiento` y `MTTR`; `nucleo/certificado`; `nucleo/perimetro`; `adaptadores/tsa` con `raices/freetsa.pem` y `raices/certum.pem`; `nucleo/corpus/fuzz_test.go` y `nucleo/ledger/fuzz_test.go`; `docs/post-ledger-salamanders.md`.

**Y la más importante se comprobó ejecutándola** (BIN), no leyendo su test: `plazum verify expediente-demo.json contexto-demo.json` desde un directorio que sólo contiene el binario y los dos ficheros responde `VERIFICADO. Recalculado desde cero sin red y sin confiar en el emisor`, con las ocho comprobaciones en `ok`, incluida `1 checkpoint(s) con sello verificado`.

El workflow de release es la excepción, tratada arriba.

### Etapa 2, catorce cerradas

| Casilla | Cómo | Evidencia |
|---|---|---|
| Puertos congelados | A/P | `puertos/etapa2.go` con 7 interfaces, `congelacion_test.go`, `puertos/contrato/` |
| `nucleo/pantalla` determinista | P | paquete con `testdata` de dorados, 91,4 % de cobertura |
| serve + htmx + `plazum serve` | CI/BIN | arranque medido **101 ms** de 3.000 en `etapa2-accesibilidad.yml`; la orden existe en la ayuda del binario |
| Seguridad web como puerta | CI | `etapa2-seguridad-web.yml` en verde: 412 casos con detector de carreras **y** las comprobaciones con curl contra el binario arrancado |
| Las 6 pantallas, GET-only | A/CI | `superficies/pantallas` no tiene ni una aparición de `http.MethodPost`; el menú que sirve el producto ofrece 6 pantallas y la puerta de axe falla si ofrece menos |
| UI generada desde el corpus | P | `TestElContenidoDelCorpusNoPasaPorElCatalogo` en `superficies/pantallas/pantallas_test.go` |
| `demo`, `doctor`, `update` | **BIN** | `plazum demo` en un directorio vacío: **86 ms**, un comando, sin banderas; `plazum doctor` sale con 0 y da tres avisos con su arreglo |
| El latido | **BIN** | `plazum latido` responde con el veredicto del planificador y dice que el pulso está apagado de fábrica; `docs/latido.md` |
| OIDC + SCIM | CI | puerta propia en `ci.yml`: 79 casos, cobertura de SCIM **81,8 %** contra un mínimo de 75 % |
| Export a SIEM | CI | `etapa2-siem.yml` en verde; `plazum export` en la ayuda |
| i18n es/en | P | `cadenas/` tiene exactamente `es.json` y `en.json`, **392 claves cada uno, sin huecos ni sobrantes** |
| Litestream + restore drill | CI | `etapa2-copias.yml` en verde con sus **nueve** pasadas: una sana y ocho copias rotas |
| pkcs7 vendorizado | CI | puerta propia: 123 casos contra un suelo de 112, sobre `adaptadores/tsa` y `herramientas/cotejapkcs7` |
| TTFV, axe-core y presupuestos | CI | TTFV 9 s de 900; `plazum demo` 0 s de 10; binario 11.788.580 bytes de 26.214.400; arranque 101 ms de 3.000; RAM 6 MB de 256 tras 200 peticiones; **26 auditorías de axe con cero violaciones y el control negativo con 5** |

### Etapa 3, doce cerradas

`nucleo/corpus/dorados.go`; 33 paquetes bajo `paquetes/`; `ciclo_e2e_test.go`; los paquetes `nis1-es`, `ai-act`, `iso42001`, `cra`, `dora`, `nis2-ue`, `eidas2`, `mdr`, `psd2-es`; `iso27001` con **132 obligaciones y 9 relojes**, exactamente lo que la casilla afirma.

**Las dos primitivas se comprobaron con la puerta que las vigila, no leyendo el código**: `primitivas_alcanzables_test.go` informa *«VERDE VACIO SOBRE EL CENSO: hoy no hay ninguna primitiva apagada ni sin cablear»*. `ventana.Maximo` y `ventana.Preaviso` existen en `nucleo/ventana/primitivas.go` con su `Vencimientos`.

**Y `plazum calendario` se ejecutó** (BIN) contra el corpus real: `--pais ES --sector fabricante-software` devuelve la cuenta entera de once líneas (249 hitos instalados, 218 en vigor, 10 alcanzados, 1 con fecha en la ventana...), que es lo que la casilla promete.

### Etapa 4, cinco cerradas

`nucleo/censo`, `nucleo/incidente`, `nucleo/accesos` con `superficies/uar`, `nucleo/auditoria`, `nucleo/acta` con `superficies/acta`. Las cinco órdenes correspondientes están en la ayuda del binario (`accesos`, `incidentes`, `auditoria`). Las cinco corren dentro de la puerta «suite completa» y las tres superficies dentro de `etapa2-seguridad-web.yml`.

### v1, cuatro cerradas

| Casilla | Cómo | Evidencia |
|---|---|---|
| **El camino guiado, de punta a punta** | **P/BIN** | `TestTTFVDelCaminoCompleto` construye el binario en un directorio temporal, lo arranca y recorre los seis pasos: `alcance /alcance 200`, `calendario /calendario/ 200`, `derivacion /controles 200`, `acta /acta/ 200`, `uar /uar/ 200`, `escalado /escalado/ 200`. «pasos alcanzados 6 de 6; exigen sesion 3» |
| Catálogo de interfaz completo en EN | P | 392 claves en cada idioma, cero huecos y cero sobrantes; `TestPuertaI18nNingunIdiomaTieneHuecos` y `adaptadores/catalogo/inventario_test.go` |
| PUERTA D11-b, estados vacíos | P | `estados_vacios_test.go` con cuatro tests, incluida la enumeración por AST y las dos formas de la nada |
| PUERTA D11-d, camino determinista | P | paso «la suite entera con la IA desactivada» en `ci.yml`, 2.331 casos con `PLAZUM_SIN_IA=1`, y dentro de esos 2.331 va el recorrido de los seis pasos de arriba |

**Sobre D11-d, con honestidad**: la puerta es cierta y **hoy es casi vacía por construcción**, porque no hay adaptador de IA que apagar. Lo dice el propio `ia_test.go` en su godoc, así que no es un hueco escondido; es un interruptor puesto antes que el aparato, a propósito.

## Lo que este barrido NO alcanza

- **No mide si una casilla está bien escrita**, sólo si lo que dice es cierto. Una casilla que promete poco y lo cumple sale igual de verde que una que promete mucho.
- **No alcanza a las dos `[~]`.** El HITO v0.2 y la imagen Docker no están entre las 58 y no se recorrieron; la segunda tiene prosa caducada (el candado), y consta arriba.
- **No hay puerta que impida que esto vuelva a pasar.** La prosa de una casilla no la vigila nada, y escribir una que la vigilara exigiría que cada afirmación de `ETAPAS.md` fuera un dato y no una frase. Eso es un cambio de forma del plan entero, no un test, y no se hace de paso.
