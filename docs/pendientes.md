# Backlog

Lo que está abierto, entero y en una pantalla. **80 elementos: 2 P0, 29 P1 y 49 P2.**
Los tres cardinales los deriva de esta misma tabla
`TestLosCardinalesDelBacklogSalenDeLaTabla`, no se escriben a mano.

| Nivel | Significado | Efecto |
|---|---|---|
| **P0** | bloquea una casilla del plan | la casilla no se marca hasta cerrarlo |
| **P1** | entra en la etapa en curso | se cierra dentro de la etapa |
| **P2** | deuda conocida | se cierra cuando toque, o se decide que no |

Un elemento se cierra en el commit que lo arregla, y sale de aquí en ese mismo
commit. El relato de cada familia, con su fecha y su cardinal, está archivado:
ver [El relato completo](#el-relato-completo) abajo.

## P0, bloqueantes

| # | Qué bloquea | Relato |
|---|---|---|
| 1 | La entrevista asistida no distingue el «sí» del «no»: sobre 64 preguntas del corpus, la negación puntúa **más** que la afirmación (5,54 frente a 4,48), así que BM25 devuelve el párrafo que dice lo contrario | [medición](https://github.com/marcosmatalab/plazum/blob/a1ef407850fba1887977cc255e7eddfd9173d962/docs/bitacora/pendientes-historico.md) |
| 2 | El plan de TTFV llevaba dentro una pieza sin costear: **35 de 68**. Bloquea D11-e, que es la fila que decide la fecha de la v1 | [medición](https://github.com/marcosmatalab/plazum/blob/a1ef407850fba1887977cc255e7eddfd9173d962/docs/bitacora/pendientes-historico.md) |

## P1, dentro de la etapa

Numerados de forma estable: hay código que cita el número (`P1 10`, `P1 12`,
`P1 20`), así que no se renumeran en bloque.

| # | Qué |
|---|---|
| 2 | `Obligacion.Vigencia` no la usa nadie |
| 3 | Falta `Obligacion.Titulo` |
| 4 | `Temporalidad` no sabe de prorrogas |
| 5 | No hay forma de ver un paquete |
| 6 | `rgpd` y `cra` llevan el texto transcrito sin tildes |
| 7 | `corpus.EsquemaUI` pierde citas |
| 9 | `paquetes/ens` no tiene entidades `informacion` ni `servicio` |
| 11 | El generador del demo vive en `nucleo/` y ya no puede regenerarlo |
| 12 | El ataque 10 no lo caza la comprobacion que su comentario promete |
| 10 | Tras un borrado legal queda un estado de control huerfano |
| 13 | El formulario del esquema se pinta en solo lectura |
| 14 | La derivacion de la pantalla no es el motor de aplicabilidad |
| 15 | El texto del corpus se pinta sin declarar su idioma |
| 26 | `nucleo/corpus` no exporta la traducción de `Temporalidad` a primitiva de `ventana` |
| 27 | La tabla de caducidades de las raíces de TSA está declarada en `adaptadores/diagnostico`, no leída |
| 16 | No existe `plazum scim token` |
| 17 | No hay pantalla de Personas |
| 18 | El directorio SCIM vive en memoria |
| 19 | El `state`, el `nonce` y el verificador PKCE viven en memoria del proceso |
| 20 | El middleware de seguridad tiene que cubrir `/scim/v2` |
| 21 | La superficie sigue pintando con su borrador de catalogo |
| 22 | El CLI habla un solo idioma |
| 23 | Nadie ensena todavia `aviso.idioma_del_corpus` |
| 24 | Las fechas no tienen formato acordado, y aqui eso es un riesgo |
| 25 | No existe todavia la eleccion de idioma |
| 28 | Las plantillas de `superficies/serve` llevan `lang="es"` cableado |
| 29 | Dentro de un contenedor, `plazum serve` dice una direccion que no sirve |
| 30 | `ber2der` amplifica hasta x482 medido, y el arreglo no esta a nuestro alcance hoy |
| 31 | El motor de fuzzing no corre en CI, en ningun objetivo del repositorio |

## P2, deuda conocida

| # | Qué |
|---|---|
| 1 | `nombresDeConfianza` es una lista cerrada |
| 2 | `ledger`: la clave publica malformada no tiene centinela |
| 3 | Un `paquete.json` corrupto y uno ausente se tratan distinto |
| 4 | Lectura del reloj por via indirecta |
| 5 | `time.Now()` en los `_test.go` de `nucleo/` |
| 6 | El linter del corpus no acota la longitud de etiqueta ni de ayuda |
| 7 | Las paginas no llevan cache HTTP |
| 8 | La accesibilidad esta cuidada a mano, no verificada por herramienta |
| 9 | Las formas del plural las tiene que resolver el catalogo |
| 10 | No se resalta que cambio con la ultima respuesta |
| 11 | No hay siguiente paso al terminar la entrevista |
| 12 | El limitador se vacia entero al llegar al techo de claves |
| 13 | Las sesiones viven en memoria y reiniciar echa a todo el mundo |
| 14 | El limite de intentos de autenticacion es por direccion, no por cuenta |
| 15 | `Origin` ausente no se rechaza |
| 16 | HSTS se manda tambien sobre http, y RFC 6797 §7.2 dice que no |
| 17 | La cookie usa `SameSite=Lax` y no `Strict` |
| 18 | El diagnostico no ve todavia el estado del servidor |
| 19 | La politica de contrasena del primer administrador es una longitud minima |
| 20 | El cierre ordenado bajo senal solo se comprueba en Linux |
| 21 | Las dos pantallas de arranque van sin estilo |
| 22 | `plazum serve` no existe todavia como orden |
| 23 | Los estaticos no traen `ETag` ni contenido precomprimido |
| 24 | El filtro de un atributo multivaluado en la ruta de un PATCH se ignora |
| 25 | No hay cierre de sesion federado |
| 26 | `meta.version` se emite y `/ServiceProviderConfig` declara `etag` no soportado |
| 27 | No hay SAML: solo OIDC. Apuntado para el ano 2 y dicho en `docs/identidad.md` |
| 28 | `web/index.html` tiene una violacion de axe y no entra en la puerta |
| 29 | El contraste con el corpus no ve una parafrasis |
| 30 | La web publica solo esta en castellano |
| 31 | El formateo de duraciones y cantidades no es del catalogo |
| 32 | `/primer-admin` no entra en la auditoria de accesibilidad |
| 33 | Los tres presupuestos viven en un fichero que se llama `etapa2-accesibilidad.yml` |
| 34 | `plazum --help` sale con codigo 2 |
| 35 | La puerta de reproducibilidad de la imagen no prueba `-trimpath` |
| 38 | El planificador de la etapa 2 es el cron del operador |
| 39 | `plazum doctor` no comprueba el planificador |
| 40 | "Su ultimo ciclo termino hace 0 horas" |
| 41 | `latido.json` se escribe sin candado |
| 42 | El receptor del pulso no existe |
| 43 | Litestream no se ejercita en CI, y por eso la casilla dice "documentado" y no "probado" |
| 44 | La retencion de 35 dias es una politica, no una comprobacion |
| 45 | El keystore del ensayo se escribe en claro |
| 46 | `plazum doctor` no sabe verificar una restauracion |
| 47 | La cadena que siembra el ensayo no lleva checkpoints |
| 48 | El manifiesto de la copia no es integridad frente a un adversario |
| 49 | La copia vendorizada sigue aceptando SHA-1 para la firma del token |
| 50 | La rama de cabeza de `pkcs7` quita del todo la verificacion de firmas DSA |
| 51 | `govulncheck` no ve el directorio vendorizado |

## El relato completo

Cada elemento de arriba tiene detrás su relato: qué lo encontró, qué se midió y
por qué se decidió lo que se decidió. Son **13.221 líneas** y no se leen para
trabajar, así que no viajan en el árbol: viven en el commit
[`a1ef407`](https://github.com/marcosmatalab/plazum/blob/a1ef407850fba1887977cc255e7eddfd9173d962), que es inmutable.

| Qué | Dónde |
|---|---|
| Las familias de fallo, con su relato y su cardinal | [`docs/bitacora/pendientes-historico.md`](https://github.com/marcosmatalab/plazum/blob/a1ef407850fba1887977cc255e7eddfd9173d962/docs/bitacora/pendientes-historico.md) |
| Los cuadernos de auditoría, uno por campaña | [`docs/hallazgos/`](https://github.com/marcosmatalab/plazum/blob/a1ef407850fba1887977cc255e7eddfd9173d962/docs/hallazgos/LEEME.md) |
| El censo de relojes que decide el orden de autoría del corpus | [`docs/censo-relojes.md`](https://github.com/marcosmatalab/plazum/blob/a1ef407850fba1887977cc255e7eddfd9173d962/docs/censo-relojes.md) |

**Por qué se archivan en vez de borrarse o quedarse.** Quedarse costaba que
`docs/` fuera mayoritariamente bitácora: 13.221 de 24.266 líneas de markdown eran
relato, y quien abría el directorio no podía distinguir la documentación del
diario. Borrarse costaba el material con el que se reconoce la séptima aparición
de una familia, que es lo que ha cerrado casi todos los hallazgos de este
repositorio. Un commit fijado no cuesta ninguna de las dos cosas: el contenido
sigue completo y recuperable con `git show`, y las referencias apuntan a un SHA
que no se mueve.

Lo vigila `TestTodaReferenciaAlArchivoApuntaAlMismoCommitYExiste`, que comprueba
que **todas** las referencias citan el mismo commit y que la ruta existe ahí.
