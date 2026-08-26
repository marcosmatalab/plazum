# Instantánea de plazum, 26 de agosto de 2026

> **Para qué sirve este documento.** Es la foto del proyecto para alguien de fuera que no ha leído nada más. Es autocontenido: no hace falta abrir el repositorio para entenderlo, y todos los números salen de ejecutar comandos, no de memoria.
>
> **Qué NO es.** No es una presentación. La autoevaluación del final está hecha para que se pueda discutir, así que las notas bajas están donde están y sin acolchar.

## Qué es esto

Un GRC de continuidad de cumplimiento. Motor determinista de obligaciones con **reloj legal**, corpus normativo como **paquetes de datos firmados**, y un **expediente verificable offline**: un tercero con el fichero y el binario, sin red y sin fiarse de quien lo emitió, recalcula la aplicabilidad, los plazos y los estados de control y obtiene lo mismo, o le dicen dónde no coincide.

Go puro, AGPL-3.0, una persona construyéndolo por etapas. El repositorio es privado y no hay ninguna release publicada.

## Los números, medidos hoy

| | |
|---|---|
| Paquetes Go | **39** |
| Líneas de Go de producción | **30.464** |
| Líneas de Go de test | **31.194** |
| Otras líneas (datos del corpus, workflows, documentación) | 22.370 |
| Casos de test ejecutados (subtests incluidos) | **1.199** |
| Cobertura de sentencias, total | **77,1 %** |
| Dependencias externas | **2** |
| Binario (`-s -w -trimpath`) | **10,51 MB** de un presupuesto de 25 |
| Arranque hasta la primera respuesta | **637 ms** de un presupuesto de 3.000 |
| Tiempo hasta el valor, un comando en un directorio vacío | **315 ms** |
| Paquetes de corpus | **31** |
| Obligaciones con reloj censadas | **360** |
| Puertas de CI | 7 workflows, **todas en verde en `main`** |

### `go list -m all`

```
plazum
github.com/digitorus/pkcs7 v0.0.0-20250729175123-57bd227bfa2f
github.com/digitorus/timestamp v0.0.0-20250524132541-c45532741eea
```

Dos dependencias, las dos en el mismo sitio (`adaptadores/tsa`, sellado RFC 3161) y **las dos vendorizadas o en camino de estarlo**. `nucleo/` tiene cero, y eso lo vigila un test que lee el AST.

## El árbol, a dos niveles

```
(raiz)/          29 ficheros: go.mod, ETAPAS.md, CLAUDE.md, los tests de arquitectura
.github/         puerta.sh, presupuesto.sh, marca-congelada, workflows/
adaptadores/     actualizador catalogo diagnostico latido oidc plantilla scim secretos tsa
cmd/             plazum
docs/            16 documentos, entre ellos guia.md (fuente única del plan), diseno.md,
                 modelo-de-amenaza.md, censo-relojes.md, decisiones.md, pendientes.md
evals/           conjuntos dorados de IA (etapa 5, vacío)
herramientas/    cribamarca ensayocopia generardemo ingestanorma sellardemo
nucleo/          aplicabilidad blobs certificado corpus estado expediente historia
                 ledger pantalla perimetro ventana
paquetes/        31 marcos: ai-act cis cra csrd data-act demo-empresa dga dora eidas2
                 eni ens iso22301 iso27001 iso27002 iso27701 iso42001 ley2-2023 lopdgdd
                 magerit mdr mica nis2-tecnica nis2-ue nist-800-53 nist-csf pci-dss
                 psd2 rgpd soc2 stig tisax
puertos/         las 9 interfaces hexagonales, congeladas, con suites de contrato
superficies/     export pantallas scim serve
web/             la web del open core, estática, sin build
```

## Cobertura por paquete

El núcleo es lo que más cubierto está, y es donde tiene que estarlo: es lo único que un tercero recomputa.

| Paquete | Cobertura | | Paquete | Cobertura |
|---|---|---|---|---|
| `nucleo/expediente` | **97,7 %** | | `superficies/pantallas` | 95,2 % |
| `nucleo/historia` | 97,6 % | | `superficies/export` | 94,4 % |
| `nucleo/ventana` | 93,6 % | | `superficies/serve` | 83,8 % |
| `nucleo/aplicabilidad` | 93,1 % | | `superficies/scim` | 69,4 % |
| `nucleo/estado` | 92,9 % | | `adaptadores/plantilla` | 92,2 % |
| `nucleo/blobs` | 91,7 % | | `adaptadores/catalogo` | 90,1 % |
| `nucleo/pantalla` | 90,0 % | | `adaptadores/secretos` | 83,3 % |
| `nucleo/perimetro` | 89,5 % | | `adaptadores/latido` | 81,6 % |
| `nucleo/ledger` | 86,8 % | | `adaptadores/oidc` | 81,5 % |
| `nucleo/certificado` | 85,7 % | | `adaptadores/tsa` | 80,7 % |
| `nucleo/corpus` | 81,7 % | | `adaptadores/actualizador` | 75,3 % |
| `cmd/plazum` | 72,3 % | | `adaptadores/diagnostico` | 73,5 % |
| `herramientas/ingestanorma` | 81,2 % | | `adaptadores/tsa/internal/pkcs7` | 69,1 % |
| `herramientas/ensayocopia` | 76,7 % | | **`adaptadores/scim`** | **46,4 %** |
| `herramientas/generardemo` | 68,6 % | | `herramientas/cribamarca` | 30,4 % |
| `puertos/contrato` | 66,0 % | | `herramientas/sellardemo` | 24,5 % |

**Los tres números bajos, dichos y no escondidos.** `adaptadores/scim` al 46,4 % es el único bajo que preocupa: es superficie de red autenticada. `cribamarca` y `sellardemo` son herramientas de un solo uso que salen a internet, y lo que tienen sin cubrir es exactamente la parte que sale a internet, que no se prueba en CI a propósito.

## Estado real de las casillas

Contado sobre `ETAPAS.md`, no de memoria. `[~]` significa hecha salvo una parte declarada.

| Etapa | Hechas | A medias | Abiertas |
|---|---|---|---|
| Semana 0 | 8 | 0 | 5 |
| Etapa 1, núcleo probatorio | **12** | 1 | 0 |
| Etapa 2, serve, UI generada y autoservicio | **14** | 1 | 1 |
| Etapa 3, corpus y venta legal | 3 | 0 | 14 |
| Etapa 4, continuidad, personas e incidentes | 0 | 0 | 13 |
| Etapa 5, IA verificable | 0 | 0 | 8 |
| Etapa 6, conectores | 0 | 0 | 9 |
| Etapa 7, riesgos y MAGERIT | 0 | 0 | 5 |
| Etapa 8, el dinero y la confianza | 0 | 0 | 9 |
| **TOTAL** | **37** | **2** | **64** |

Las dos a medias, y por qué:

- **Etapa 1, HITO v0.2 firmada**: bloqueada por la congelación de publicación, no por trabajo pendiente.
- **Etapa 2, matrix build e imagen Docker**: la casilla pide "imagen **publicada**" y no se publica nada. Todo lo demás está y medido.

La única casilla abierta de la etapa 2 es el **HITO v0.3**, que tampoco es trabajo: demo alojada y lista de espera, o sea una decisión y unos diez euros al mes.

## La lista de pendientes

**30 P1** y **54 P2** anotados en `docs/pendientes.md`, cada uno con su porqué y su arreglo. Un P1 no bloquea una casilla, entra en la etapa; un P2 se arregla cuando toque o se decide que no.

Y una tabla aparte que es la más útil del repositorio: **la familia "guardas que no guardaban", con quince entradas en dos semanas**. Cada una es una comprobación que parecía funcionar y no comprobaba nada, con cuánto llevaba así y cómo se cazó. **Diez de las quince son emparejamientos** hechos por índice, posición u orden en vez de por una identidad firmada, que es ahora el invariante 7 del proyecto.

## Lo que hay de corpus, sin maquillar

31 paquetes con metadatos correctos y linter legal en verde. Con **contenido de verdad**, cuatro: `ens` (132 obligaciones, 8 relojes, 24 dorados), `iso27001` (129 obligaciones referenciales, 6 relojes, 18 dorados), `rgpd` y `cra` (semillas con su reloj). Los otros 27 son esqueletos.

El **censo de relojes** (`docs/censo-relojes.md`) mide qué hay que escribir: **360 obligaciones con reloj** repartidas en plazo explícito, periodicidad y evento disparador, cada cuenta con la cita del artículo que la respalda, y los siete referenciales marcados "no verificable sin la copia del cliente", que es un resultado y no una excusa.

**El agujero que el censo destapó y que cambia el orden de autoría**: lo que obliga hoy en España para incidentes de red **no es NIS2 sino NIS1** (RDL 12/2018 + RD 43/2021 + sus 12 NTI). NIS2 sigue sin transponer, comprobado contra el índice de legislación consolidada del BOE. Y no hay paquete.

Y dos cosas que el censo le pide **al motor**, no al corpus: falta la primitiva del **máximo de dos duraciones** (una fija y otra declarada por el obligado; el CRA no se puede escribir bien sin ella, y son 31 relojes), y aparece una familia de **preaviso contractual** que el motor calcula al revés, porque la fecha límite es entrada y lo que se calcula es cuándo empezar.

---

# Autoevaluación de las 17 dimensiones

**El criterio, y es distinto del de `docs/diseno.md`.** Allí la nota es de **diseño**: si la decisión está tomada, es defendible y tiene un test falsable especificado. Aquí la nota es de **lo construido y medido hoy**. Son dos preguntas distintas y por eso las dos columnas se enseñan juntas.

Una dimensión con el diseño cerrado y cero código sacará un 1,5, y eso **no es un defecto del diseño**: es que le toca en la etapa 5 o en la 7 y estamos en la 2.

| # | Dimensión | Peso | Diseño | **Hoy** | Qué sostiene la nota de hoy |
|---|---|---|---|---|---|
| D1 | Modelo de obligación y temporalidad | 12 | 9,7 | **8,0** | 8 relojes reales corriendo con dorados derivados del texto legal, calendarios y régimen de cómputo. Baja porque el censo de 360 relojes identificó **dos primitivas que faltan**: el máximo de dos duraciones y el cómputo hacia atrás del preaviso |
| D2 | Determinismo y reproducibilidad | 8 | 9,6 | **9,3** | 14 ataques al expediente, modelo de amenaza escrito con lo que NO demuestra, demo que verifica offline con sello real, imagen Docker con reproducibilidad medida (mismo digest en dos construcciones). Residual: sin prueba de consistencia entre checkpoints |
| D3 | Cobertura por estratos y calendarios país | 8 | 9,5 | **4,5** | **La nota más baja de las que importan.** El formato está entero y el linter legal es ejecutable, pero sólo 4 de 31 paquetes tienen contenido. El censo dice qué falta y en qué orden; escribirlo es la etapa 3 |
| D4 | Implantación e2e, 5 clases con facetas | 8 | 9,6 | **7,0** | `clase_e2e` con facetas construido y `plazum cobertura` lo publica. Baja porque sólo mide sobre los paquetes que existen |
| D5 | Conectores WASM con conformidad | 6 | 9,5 | **2,0** | Nada construido. Es la etapa 6 |
| D6 | Continuidad: certificado, escalado, silencio | 8 | 9,5 | **7,5** | Certificado, estados y escalado construidos; el latido con su vigilante entró hoy. Falta el planificador propio: hoy quien apunta que ha corrido es un temporizador del operador |
| D7 | Evidencia y valor probatorio | 6 | 9,7 | **9,5** | **Lo más terminado del producto.** Ledger v2 con compromiso de clave, lápidas, borrado legal que no blanquea, 14 ataques, ensayo de restauración que termina verificando la cadena, y el modelo de amenaza que dice qué queda fuera |
| D8 | Riesgos con MAGERIT | 6 | 9,5 | **1,5** | Nada construido. Es la etapa 7 |
| D9 | Ligereza y huella | 3 | 9,8 | **9,7** | Binario 10,51 MB de 25, arranque 637 ms de 3.000, RAM dentro de 256 bajo 200 peticiones, imagen `scratch` de 15 MB sin shell. Todo medido en CI, con la puerta viéndose fallar en cada ejecución contra un límite imposible |
| D10 | Instalación local y datacenter | 5 | 9,6 | **8,5** | Docker, matrix en tres sistemas arrancando el binario, Litestream documentado con ensayo de restauración, OIDC y SCIM con cero dependencias nuevas. Falta el tramo alto (Postgres) y publicar la imagen |
| D11 | Intuitividad y guiado | 7 | 9,5 | **7,5** | 6 pantallas derivadas del corpus, entrevista, TTFV de 315 ms con un solo comando, axe-core con cero violaciones sobre 16 auditorías reales. Baja porque **todavía no se puede guardar nada**: todas las rutas son GET, a propósito, y un botón que no guarda sería la peor mentira de esa pantalla |
| D12 | IA verificable | 6 | 9,6 | **1,5** | Nada construido. Es la etapa 5 |
| D13 | Extensibilidad | 4 | 10,0 | **9,8** | Una norma nueva no toca código: las reglas de aplicabilidad las declara el paquete en Datalog estratificado, y el test AST que prohíbe normas cableadas vigila también los `_test.go` desde que se descubrió que ése era el agujero por el que vivían |
| D14 | Open core self-serve | 6 | 9,5 | **1,5** | Nada construido. Licencia, cobro y carpeta de compras son etapas 3 y 8 |
| D15 | Legalidad del corpus | 6 | 9,6 | **9,0** | Tres techos de texto por tipo de campo, `licencia_fuente` de vocabulario cerrado cruzado con la clase, **lista negra ejecutable** (CIS y SCF se rechazan con el motivo dentro del error), atribución del DOUE mostrada en producto y no sólo en un fichero, e identificador como dato con el enlace derivado |
| D16 | Cross-framework computado | 5 | 9,5 | **1,5** | Nada construido. Es la etapa 3 |
| D17 | Autoservicio radical | 5 | 9,6 | **6,0** | `demo`, `doctor` y `update` con vuelta atrás comprobada, y un TTFV que se mide en CI contra el binario. Falta la carpeta de compras y todo el autoservicio de licencia |
| | **GLOBAL ponderado** | 109 | **9,59** | **6,18** | |

## Cómo se lee ese 6,18

**No es una nota de calidad, es una nota de avance.** El diseño está en 9,59 y no ha bajado; lo que mide la columna de hoy es cuánto de ese diseño existe.

Las cinco dimensiones que más separan las dos columnas son **D12 (IA), D8 (MAGERIT), D14 (open core), D5 (conectores) y D16 (cross-framework)**, y las cinco tienen la misma explicación: **les toca en las etapas 5, 6, 7 y 8, y estamos cerrando la 2**. Sumadas valen 29 puntos de peso, o sea que por sí solas explican casi la mitad de la diferencia.

**La que sí es un aviso es D3, cobertura de corpus, en 4,5 con peso 8.** No está esperando a una etapa lejana: es la etapa 3, la siguiente, y es la dimensión que decide si esto se puede vender. El censo existe precisamente para que escribir corpus deje de ser "completar marcos" y pase a ser un orden justificado con números.

Y la lectura optimista, que también es real: **las tres dimensiones más terminadas son D7 (9,5), D13 (9,8) y D9 (9,7)**, y son exactamente las tres que un competidor no puede copiar sin rehacer su producto: el valor probatorio, la extensibilidad sin tocar código y la huella.

## Cómo reproducir estos números

```bash
go build ./...
go vet ./...
go test ./... -count=1                      # 1.199 casos
go test ./... -count=1 -cover               # cobertura por paquete
go list -m all                              # 2 dependencias
go build -ldflags="-s -w" -trimpath -o /tmp/plazum ./cmd/plazum
gh run list --branch main                   # las 7 puertas
```

El estado de las casillas sale de contar los `- [x]`, `- [~]` y `- [ ]` de `ETAPAS.md`, y el censo de `docs/censo-relojes.md`.
