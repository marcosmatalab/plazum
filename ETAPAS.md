# ETAPAS.md: el plan ejecutable

Fuente única del diseño: `docs/guia.md` (con sus Anexos A y B). Este fichero concreta números y detalles operativos: en conflicto de diseño manda la guía, en concreción operativa manda este fichero. Cada casilla es una puerta: se marca cuando su test corre en verde en CI, no antes. Estado global objetivo: 9,7 en las 17 dimensiones hacia el mes 24-27.

> **D-19 (01-09-2026) recorta este plan, y recorta ANCHURA DE CORPUS, no profundidad de nada.** La v1 sale con **12 marcos** (ISO 27001, RGPD+LOPDGDD, ENS, NIS2, DORA, AI Act, ISO 42001, CRA, Ley 2/2023, SOC 2, PCI DSS, TISAX) y sale **como plataforma guiada**, no como motor con pantallas. **E6 (conectores) y E7 (riesgos) dejan de bloquear la salida** y pasan a post-v1, sin salirse del 9,7. Lo construido de MiCA, eIDAS2, MDR, PSD2-ES y ENI **se queda en el corpus como extra**: medido el 01-09-2026, los **90 relojes escritos están todos dentro de los 12**, así que el recorte no deja fuera ni un reloj hecho. El porqué entero, con los números y la corrección al censo de `nis2-tecnica`, en `docs/decisiones.md` D-19.

> **D-20 (01-09-2026) mueve DÓNDE ESTÁ EL VALOR, no la anchura.** El sistema es el producto y el corpus es combustible: los 12 marcos se terminan igual pero como **corpus community-grade** (gratis, sin garantía, con el descargo cargando la honestidad), **la IA de adopción entra en la v1** (piezas 1, 2, 3, 4 y 7 de `docs/ia.md`, con FTS5 y el verificador de citas adelantados de E5), el tier de 1.490 pasa a ser **suscripción de vigilancia** sin lenguaje de garantía jurídica, y **la revisión jurídica externa y los design partners dejan de ser puertas** y pasan a aceleradores opcionales, con dos puertas que los sustituyen y que son más caras de falsificar. La vara sigue en más de 9,7, y esta decisión **lo hace más caro, no más fácil**: al mover peso hacia D5 y D12, que hoy están a 2,0 y 1,5, el global honesto **baja 0,0505**. El porqué entero, con la aritmética, en `docs/decisiones.md` D-20.

<!-- estado:inicio -->
> **EL ESTADO VA CON DOS NÚMEROS, y ninguno de los dos solo dice la verdad** (regla del 03-09-2026). Contados del árbol por `estado_del_plan_test.go`, no estimados:
>
> **78 de 144 casillas** cerradas, y **266 relojes escritos** en el corpus.
>
> **Y LA TERCERA, QUE ES LA QUE DECIDE LA FECHA**: **25 de 66 abiertas** en el conjunto que bloquea la v1, o sea las etapas 3 y 4 y el bloque de la v1. Las dos de arriba cuentan el plan ENTERO, que llega hasta la etapa 8, asi que ninguna dice cuanto falta para salir. Esta vivia en la prosa de un informe y se contaba a mano: el 08-09-2026 dos recuentos del mismo dia dieron **30 y 31**, y el arbol dice **31**. El denominador, 65, coincidia en los dos, que es lo que hace peor al fallo, porque cuando el denominador cuadra nadie vuelve a mirar el numerador. La deriva `estado_del_plan_test.go` por seccion, no por rango de lineas.
>
> **LA DERIVA, que las dos cifras de arriba no ensenan.** Instantanea anterior: **31 de 100, el 25-08-2026**, en `a4df425`. Esta es del **10-09-2026**, o sea **16 dias** despues: **+47 cerradas** y **+44 abiertas**, **2,9 al dia contra 2,8**, y las pendientes de **69 a 66**. El plan crece un poco mas deprisa de lo que se cierra, y **eso no es malo por si solo**: D-22 abrio siete casillas porque destapo trabajo que ya estaba dentro de otras dos, y el trabajo estaba igual antes de contarlo. Lo que si era malo es que no se viera, porque con dos cifras que solo suben **un re-corte que ensancha el plan se lee igual que uno que lo estrecha**. El ancla es un hecho del pasado y no caduca; los deltas, las tasas y las pendientes los deriva `estado_del_plan_test.go` de ella y del arbol.
>
> **El total sube de 135 a 142 el 07-09-2026 y el numerador no se mueve, que es lo que tiene que pasar.** D-22 no cierra nada: **abre siete casillas** (las seis del eslabón de la prueba más la del techo de `observable` en la etapa 3) y **mueve una** desde la etapa 4. Un re-corte que subiera el numerador sería un re-corte que se está puntuando, y el trabajo que estas casillas destapan ya estaba dentro de las piezas 3 y 4 de la IA: lo que cambia no es cuánto queda, es que ahora se ve.
>
> El contador de casillas **no mide el trabajo de corpus, y tampoco mide el de producto**. La campaña del 04-09-2026 hizo que **por primera vez se pueda entrar en plazum y recorrer los seis pasos del camino**, construyó la ley de conservación del calendario, bajó la entrevista de 41 preguntas a 19 y las vigencias sin contrastar de 196 a 39, **y mueve CERO casillas**. Las tres D11 que seguían abiertas siguen abiertas, cada una con su número en su línea: 2 órdenes de terminal, 5 cifras huérfanas de 14, y 51 segundos de más.
>
> **BARRIDO DE CASILLAS FALSAMENTE ABIERTAS (04-09-2026): 77 recorridas, 0 resultan ya ciertas.** Y el cero vale más que un puñado de casillas, porque dice DÓNDE está el riesgo. Las tres que se cerraron el 03-09-2026 sin construir nada (repo público, private vulnerability reporting, CodeQL) eran **las tres estado de plataforma**: cosas ciertas fuera del repositorio, que cambian sin dejar rastro aquí. **Una casilla de código no envejece en silencio**, porque el repositorio es su fuente de verdad y hay una puerta que la vigila; una cuyo estado vive en GitHub, en un despacho o en un contrato **sí**, y nadie va a mirarla porque no depende de ti. Las que quedan de esa clase, y que este barrido NO puede resolver porque exigen a una persona: las cláusulas de PI del contrato activo, la revisión del CLA por abogado, la venta legal y las horas fijas de vigilancia normativa. Comprobadas y descartadas por estar realmente pendientes, con su evidencia: la política N-1, la página pública de vigilancia, el verificador de citas por hash y la puerta del descargo por pantalla (la frase está en cuatro sitios y **ninguna puerta las enumera**, que es la forma que tenía D11-b antes de cerrarse).
>
> **Y EL BARRIDO MIRABA HACIA EL LADO EQUIVOCADO.** Buscaba casillas falsamente ABIERTAS, encontró cero, y había una falsamente **CERRADA**: la del camino guiado de punta a punta, marcada mientras tres de sus seis pasos contestaban 401. Una casilla abierta de más hace planificar trabajo que no existe; **una cerrada de más hace medir sobre una maqueta y creerse el número**, que es lo que llevaban haciendo las cinco puertas de D11. El próximo barrido recorre las **58 cerradas**, no las abiertas.
>
> **BARRIDO DE LAS 58 CERRADAS (04-09-2026): 58 verificadas ciertas, 0 desmarcadas, 1 que no se puede comprobar por ejecución.** La evidencia de cada una, en `docs/hallazgos/barrido.md`. **54 de las 58 se verificaron con una puerta, con el binario ejecutado o contra la API de GitHub**; las otras cuatro descansan sólo en que un fichero está donde dice. La que no se puede comprobar es el **workflow de release**: está escrito y **no se ha ejecutado nunca**, ni una vez, y el remoto no tiene ni una etiqueta.
>
> **Y el cero se dice en voz alta, porque este cero llega un día después del barrido que corrigió la falsamente cerrada**, o sea sobre un tablero recién repasado. Un cero así vale mucho menos que un cero sobre uno que lleva un mes sin mirarse, y el próximo barrido toca dentro de semanas, no mañana. **Lo que sí salió fueron 5 casillas cerradas cuya prosa ya no describe el árbol** (codeql decía «una alerta ABIERTA» con cero abiertas; el release decía 4 plataformas con seis destinos; los «30 marcos» son 33 con 252 relojes; axe decía 16 auditorías y hace 26; y la UAR declaraba un hueco de accesibilidad que llevaba nueve horas cerrado). Las cinco están corregidas. **Y el hallazgo no son las cinco correcciones, es que hicieron falta**: una casilla tiene dos mitades, el corchete y la prosa, **y sólo la primera tiene puerta**.
<!-- estado:fin -->
>
> **Recuento anterior**: **50 hechas de 135** (eran 50 de 121). Once del re-corte de D-20: **+10 en v1** (el bloque IA de adopción, con FTS5 y el verificador adelantados de E5), **+1 en E3** (la puerta de D15 que sustituye a la revisión jurídica), **+1 en E8** (la puerta de D14 que sustituye a los design partners) y **−1 en E5**, que se queda en 7 porque cede tres casillas y recupera dos. Y tres de las correcciones de rumbo del 02-09-2026: **+1 en E3** (`nis2-es` preparado para el día D), **+1 en v1** (la demo alojada, que sube desde E3) y **+1 en E6** (los cuatro conectores partidos en dos que se construyen y dos que esperan a que alguien los pida). **Lo hecho no se mueve: 50 sigue siendo 50.** Un re-corte que subiera el numerador sería un re-corte que se está puntuando.

> **Tres correcciones de rumbo del 02-09-2026, de la investigación de demanda 2027. No tocan la v1, ordenan lo que va detrás.**
>
> 1. **El post-v1 se reordena a E6 → E8 → E7.** «Cumplimiento continuo» sin conectores se lee en 2027 como herramienta de documentación, y la frase de D-20 —*que haga cosas dentro del sistema del cliente*— es literalmente E6. Y los conectores propios bajan de cuatro a **dos en profundidad**, Entra ID y GitHub, que son los que dan evidencia de accesos y de cambio de código, o sea justo lo que alimentan la UAR y la auditoría que ya están construidas.
> 2. **La demo alojada sube a la v1.** Deja de ser una casilla del carril de confianza en E3 y pasa a bloquear el hito de la v1: el self-hosted gana en regulado y en ENS, que es la cabeza de playa, pero a quien espera SaaS **sólo lo convence probarlo sin instalar**, y «ultra intuitivo» sin sitio donde comprobarlo no lo comprueba nadie de fuera.
> 3. **El día D de NIS2 en España se planifica como evento, no como sorpresa.** Se prepara `nis2-es` como release de vigilancia antes de que exista la ley, para publicar en horas y no en semanas. Casilla en E3.

## Semana 0: fundaciones
- [x] Estructura del repo (nucleo/puertos/adaptadores/superficies/paquetes)
- [x] Núcleo construido y en verde (ventana, aplicabilidad, estado, ledger, expediente, corpus)
- [x] Tests de arquitectura: AST del núcleo, normas no cableadas, linter sobre paquetes/
- [x] CLAUDE.md, DEPENDENCIAS.md, SECURITY.md, CONTRIBUTING.md, CLA.md
- [x] CI: build, test, gofmt, vet, cobertura con puerta dura 85%, govulncheck y gosec bloqueantes con versión fijada, CodeQL, dependabot
- [x] Descargar el texto canónico de AGPL-3.0 a LICENSE (gnu.org/licenses/agpl-3.0.txt)
- [x] DECISIÓN DE MARCA, tomada e implantada el 26-08-2026. **El producto se llamaba DUTIQ y ahora se llama PLAZUM.** Lo que mató a DUTIQ: TMview, 25-08-2026, DOS EUTM de "Utiq" REGISTRADAS (no solicitadas), del titular Utiq SA/NV, **018838934** denominativa y **018838908** figurativa, vigentes hasta el 21-02-2033, en clases **9, 25, 35, 38 y 42**. ([por qué](docs/casillas.md#decision-de-marca-tomada-e-implantada-el-26))
- [x] HACER PÚBLICO EL REPO. **Hecho, y el plan no se había enterado.** Comprobado el 03-09-2026 contra la API (`gh api repos/marcosmatalab/plazum --jq .private` devuelve `false`). ([por qué](docs/casillas.md#hacer-publico-el-repo-hecho-y-el-plan))
- [ ] Revisar cláusulas de PI del contrato de empleo/consultoría activo
- [x] Activar private vulnerability reporting en GitHub. **Activo**, comprobado el 03-09-2026: `gh api repos/marcosmatalab/plazum/private-vulnerability-reporting` devuelve `{"enabled": true}`, donde antes devolvía 404. La casilla decía «BLOQUEADA por plataforma» y el bloqueo se había levantado solo, que es lo peor que le puede pasar a un motivo escrito: **un impedimento externo no se vuelve a mirar precisamente porque no depende de ti**
- [x] Reactivar el workflow codeql. **Reactivado** en el commit 49507c1 («lo que dormía era el nombre del fichero») y corriendo: el análisis de main del 03-09-2026 subió su SARIF y el code scanning tiene análisis con resultados. ([por qué](docs/casillas.md#reactivar-el-workflow-codeql-reactivado-en-el-commit))
- [x] Endurecer gosec a bloqueante tras triar sus hallazgos en el primer push (quitar continue-on-error, anotar #nosec justificados)
- [ ] Revisión por abogado del texto del CLA antes de la primera contribución externa
## Etapa 1 (4-6 FdS): el núcleo probatorio completo

> **Etapa 1 técnicamente completa: 12 de 13.** La revisión hostil del 25-08-2026 encontró 7 hallazgos, uno bloqueante, y los tres bloques de arreglo están cerrados con el código delante. El bloqueante era de clase, no de caso borde: todo lo que el receptor debe aportar estaba guardado como campo que escribe el emisor. Arreglo: `ContextoReceptor` y `ledger.Confianza` entran por parámetro, los campos del fichero pasan a `AnclasDeclaradas` y `ClavesDeclaradas` y solo se contrastan, el `Checkpoint` lleva `Token` que se verifica de verdad, y el `Expediente` pasa a `CadenaV2`. Lo vigila `TestLaConfianzaNoViajaEnElFichero` (AST, con control negativo). Los `hostil_test.go` de cada paquete pasaron de rojo a regresión. Barrido de mutación sobre el verificador: de 16 comprobaciones sin control negativo aislado a 1, y esa queda anotada en el código como estructuralmente redundante. **Lo único que falta es el HITO, y no depende del código sino de la comprobación de UTIQ en TMview.**

## Etapa 1 (4-6 FdS): el núcleo probatorio completo
- [x] Ledger v2: AEAD con compromiso de clave, con control negativo de clave sustituida (nucleo/ledger/v2.go). Y ahora SÍ en el camino del tercero: el Expediente lleva `CadenaV2`, con las entradas cifradas, las claves divulgadas por entrada y checkpoints propios. Una entrada sin clave y sin lápida es discrepancia: el emisor no puede ocultar contenido sin decir por qué
- [x] Lápidas firmadas con base legal; verificar informa "suprimida con base legal X", nunca "manipulada" (v2.go). `contenidoFirmado()` ata la lápida al hash de la entrada y a la raíz de la cadena, con dominio de firma propio, así que ya no se transplanta. Se rechazan índices fuera de rango y duplicados, y la clave del operador se comprueba de tamaño antes de usarla, que hacía panic
- [x] Keystore separado con destrucción de clave (v2.go). El borrado legal compone con la verificación de un tercero: el ciclo e2e borra sobre el expediente ya verificado y este sigue verificando informando la supresión. Pendiente operativo: retención 35 días y ciclo de la clave maestra en el runbook del adaptador
- [x] Blobs content-addressed cifrados con compromiso y detección de sustitución (nucleo/blobs); la tabla SQLite y el chunking >32 MB van con el adaptador de almacén
- [x] Historia bitemporal: EstadoEn, Ventana SOC 2, PrimerConocimiento (art. 33) y MTTR (nucleo/historia); pendiente: re-ejecutar los 10 ataques del expediente sobre historia al integrarla en el expediente
- [x] Objeto Certificado con hitos sobre el motor de ventana, con los TRES dorados en verde (nucleo/certificado)
- [x] Perímetros multi-entidad: herencia, roll-up, ciclos rechazados al cargar (nucleo/perimetro)
- [x] Anclaje RFC 3161 con cadena de reserva (2 TSAs + cola local) y verificación offline (adaptadores/tsa). `Checkpoint` lleva `Token` y la verificación llama de verdad a `VerificarSello`, inyectado porque nucleo/ no importa el adaptador. La firma del checkpoint cubre ahora el anclaje y el digest del token, que antes no. Sin raíces de TSA, `plazum verify` avisa y NO verifica, en vez de dar por bueno un anclaje que nadie comprobó
- [x] `plazum verify` funciona out of the box sobre el demo. El demo se sella UNA VEZ contra una TSA real (`herramientas/sellardemo`, que sale a la red a mano y nunca en CI) y el binario trae embebidas las raíces de FreeTSA y Certum, que son certificados públicos y redistribuibles: un verificador que no trae raíces no es usable offline, y offline es su razón de existir. ([por qué](docs/casillas.md#plazum-verify-funciona-out-of-the-box-sobre))
- [x] Fuzzing nativo de Go del linter de corpus y del verificador comprometido (semillas corren en cada go test)
- [x] Workflow de release: **6 destinos** (3 sistemas × amd64 y arm64; la casilla decía «4 plataformas» y la matriz da seis, corregido en el barrido del 04-09-2026), SHA256SUMS, SBOM CycloneDX, firma keyless cosign (.github/workflows/release.yml). ([por qué](docs/casillas.md#workflow-de-release-6-destinos-3-sistemas-amd64))
- [~] HITO: v0.2 firmada. **BLOQUEADO por decisión de marca, no por trabajo pendiente.** El tag v0.2.0 está creado en local y NO se empuja. Razón: el workflow de release firma con cosign keyless, que sube el certificado al log público de Rekor con la identidad del repositorio dentro, y Rekor es append-only. La primera release firmada publicaría el nombre de forma irreversible. **El motivo original ya no existe**: se llamaba DUTIQ, DUTIQ contenía UTIQ, y desde el 26-08-2026 se llama PLAZUM, implantado de punta a punta y con el expediente de demostración regenerado y resellado. El candado (`.github/marca-congelada`) sigue puesto por una razón distinta y más pequeña: publicar es irreversible y la decisión es de una persona, no de un renombrado que salió bien. Se abre borrando ese fichero en un commit propio. **La etapa 1 se cierra en 12 de 13 con este hito diferido** **CANDADO ABIERTO el 26-08-2026**: `.github/marca-congelada` borrado en commit propio, después de repetir la criba con el cribador **paginado** (el anterior podía contestar "sin hallazgos" con el transporte roto, ver `docs/marca.md`) y de comprobar a mano los vecinos a una letra, que ninguna herramienta de subcadenas ve. Lo que queda NO es trabajo ni decisión de marca: es un `git push origin v0.2.0`, y no se hace solo porque publicar en Rekor es de un solo sentido y ese empujón lo da una persona
- [x] Post del ledger escrito (docs/post-ledger-salamanders.md): los invisible salamanders explicados y resueltos, con lo que el compromiso de clave NO arregla dicho también. SIN PUBLICAR hasta que el repo sea público

## Etapa 2 (8-12 FdS): serve, UI generada y autoservicio

> **Puertos congelados el 25-08-2026, antes de implementar nada.** `puertos/etapa2.go` define las interfaces contra las que compilan todos los frentes, `congelacion_test.go` las congela leyendo el AST y `puertos/contrato/` son las suites de comportamiento que toda implementación tiene que pasar. Congelar la firma no congela el significado: una `Sesion` que devuelve nil en `ComprobarCSRF` satisface la interfaz y no protege de nada, por eso existen las suites. Verificado por mutación que tienen dientes, y la mutación enseñó además que cambiar una firma a secas lo caza el compilador vía los dobles, no el test de congelación: para aislarlo hay que cambiar la firma Y arreglar el doble, que es lo que haría un frente.

> **Invariante 2 cerrado antes de abrir los frentes (25-08-2026).** Las reglas de aplicabilidad las declara el paquete, en el dialecto Datalog estratificado, y `progENS` ha dejado de existir como código Go. El motivo no es de higiene: la suscripción del corpus y el canal consultor solo existen si actualizar el corpus es un fichero de datos firmado y no una release del binario. Con 2 paquetes autorizados esto era una tarde; con 12 habría sido una migración. Lo que hizo falta: sintaxis de superficie del dialecto con su parser (4 hallazgos del fuzzing, todos de la familia "acepta lo que su propio escritor no puede reemitir"), el bloque `aplicabilidad` en el formato de paquete con su linter, las 29 reglas del ENS como datos con cita artículo por artículo, y un test que las EJECUTA contra el motor, porque el linter solo dice que una regla se parsea. Y el test AST que vigila el invariante pasa a mirar los `_test.go` de `nucleo/` y `adaptadores/`: excluirlos era el agujero por el que `progENS` vivió meses con el test en verde. Y el aislamiento por espacio de nombres, que estaba escrito en un godoc y no existía en el código: dos paquetes que declararan `en_ambito` con significados distintos derivaban el uno sobre los hechos del otro en silencio. Ahora lo que un paquete define y no exporta es suyo, exportar es una promesa con tres formas de romperla y las tres son error al cargar, y un hecho del sujeto que choca con un predicado propio de un paquete se denuncia en vez de alimentar el vacío.


- [x] Puertos de la etapa 2 definidos y congelados, con godoc y suites de contrato. ([por qué](docs/casillas.md#puertos-de-la-etapa-2-definidos-y-congelados))
- [x] `nucleo/pantalla`: derivación determinista de las 6 pantallas desde el corpus, con caso dorado comparado byte a byte y control negativo. Cazó de paso que `corpus.EsquemaUI` no era determinista: con dos paquetes declarando el mismo atributo, el primero recorrido fijaba etiqueta, ayuda y cita, y ese "primero" lo decidía el orden del directorio
- [x] serve con html/template + htmx vendorizado, go:embed, sesiones, y la orden `plazum serve` que lo levanta. ([por qué](docs/casillas.md#serve-con-html-template-htmx-vendorizado-go-embed))
- [x] Seguridad web como puerta (`superficies/serve`). CSRF exigido por MÉTODO, no por lista: la puerta pregunta al enrutador cuáles son sus rutas y falla si alguna mutante se atiende sin token, con control negativo sobre la cadena desnuda. ([por qué](docs/casillas.md#seguridad-web-como-puerta-superficies-serve-csrf-exigido))
- [x] Las 6 pantallas con derivación a un clic (`superficies/pantallas`), sobre `nucleo/pantalla`. Un P0 real cerrado en la pasada del atacante: un paquete con bytes que no son UTF-8 válido producía una página que tampoco lo era, y ante una secuencia inválida cada navegador resincroniza por donde quiere. `html/template` escapaba perfectamente y no lo arreglaba, porque no era un problema de escapado. Todas las rutas son GET a propósito: no hay dónde guardar todavía, y un botón que no guarda es la peor mentira posible en esta pantalla
- [x] UI generada desde `corpus.EsquemaUI` y `corpus.Entrevista`. La frontera de licencias es ejecutable: `TestElContenidoDelCorpusNoPasaPorElCatalogo` apunta cada clave que se le pide al catálogo y exige que ninguna sea texto del corpus, porque traducir texto del BOE crea obra derivada. Verificado por mutación (metiendo `{{ t .Texto }}` en una plantilla, dos tests en rojo)
- [x] `plazum demo`, `plazum doctor` y `plazum update` con vuelta atrás. **TTFV medido: 315 ms y UN comando**, sin flags, en un directorio vacío. La pantalla trae el alcance respondido, las obligaciones que aplican con su regla del dialecto y su cita, **una que NO aplica** (que es la mitad que importa: sin ella la derivación no se distingue de un volcado de catálogo), los relojes ordenados por urgencia y los dorados recalculados contra el motor. `update` verifica el punto de retorno releyéndolo antes de tocar nada, y deshacer un punto inventado falla en vez de fingir
- [x] El latido: pulso diario opt-in a plazum.dev/latido (dominio provisional) + aviso si calla 24h + smoke test del canal + estado del planificador en Hoy. ([por qué](docs/casillas.md#el-latido-pulso-diario-opt-in-a-plazum))
- [x] OIDC + SCIM con extensión enterprise y mapeo manual (`adaptadores/oidc`, `adaptadores/scim`, `superficies/scim`). **Cero dependencias nuevas**: descubrimiento, JWKS y verificación del ID token con la biblioteca estándar, porque esto está en la frontera de confianza. 25 formas de falsificar un ID token comprobadas una a una, cada una con su control negativo dentro del subtest, incluida la confusión de algoritmo (`alg: HS256` firmado con la clave pública RSA). `golang.org/x/oauth2` se retira de `DEPENDENCIAS.md`: estaba planeada y no hizo falta
- [x] Export del log de auditoría a SIEM (JSON líneas) (`superficies/export`, `plazum export`). ([por qué](docs/casillas.md#export-del-log-de-auditoria-a-siem-json))
- [x] i18n es/en con mecanismo de catálogo (`adaptadores/catalogo`), consumido ya por la interfaz. ([por qué](docs/casillas.md#i18n-es-en-con-mecanismo-de-catalogo-adaptadores))
- [x] Litestream documentado + restore drill en CI (base + keystore + blobs, verifica cadena y lápidas). ([por qué](docs/casillas.md#litestream-documentado-restore-drill-en-ci-base-keystore))
- [~] Matrix build Linux/macOS/Windows-Docker + imagen Docker publicada; descargo "no es asesoramiento jurídico" en pie y explain. **Hecho todo menos PUBLICAR la imagen.** Ya NO lo bloquea la marca: el candado se abrió el 26-08-2026. Lo que queda es que el repositorio es privado, así que un paquete subido a `ghcr` nacería privado y "imagen publicada" no sería cierto para ningún comprador. O sea que esta casilla depende de hacer público el repositorio, que es una decisión aparte y no se toma aquí. Matriz de las tres plataformas con la suite entera y el binario ARRANCADO en cada una (`etapa2-distribucion.yml`): compilar no es arrancar, y lo que el comprador de macOS descarga tiene que haberse ejecutado en macOS antes de llevar una firma. `Dockerfile` multietapa sobre `scratch`, sin privilegios, sin intérprete de órdenes, imagen base fijada por digest, 15 MB, con el corpus y el expediente dentro para que `docker run --rm plazum` enseñe algo sin montar nada. **Reproducible medido**: dos construcciones con `--no-cache` dan el mismo sha256 del binario. La imagen se construye y se prueba entera en CI y NO se sube a ningún registro: `.github/marca-congelada` es el candado, `release.yml` pregunta por él antes de cada paso que sale de la máquina, y `distribucion_test.go` se pone rojo si alguien añade uno que no pregunte. Un P0 real cerrado por el camino: en `scratch` no hay `/usr/share/zoneinfo`, así que `plazum verify` respondía **NO VERIFICA** sobre un expediente correcto, o sea acusaba al emisor de un fallo del receptor, que es el peor fallo posible en un producto cuya tesis es que el receptor no se fía. La base de zonas viaja ya dentro del binario (`cmd/plazum/zonas.go`) y la puerta que lo mide ejecuta `verify` DENTRO de la imagen. El descargo: en el pie de las seis pantallas (por catálogo, es y en) y al cierre de `plazum explain`, con puerta propia en los dos sitios y control negativo
- [x] Vendorizar pkcs7 en adaptadores/tsa/internal/pkcs7 con fuzzing propio. ([por qué](docs/casillas.md#vendorizar-pkcs7-en-adaptadores-tsa-internal-pkcs7-con))
- [x] TTFV sintético en CI <15 min; axe-core cero violaciones; presupuestos (binario <25 MB, arranque <3 s, RAM <256 MB). ([por qué](docs/casillas.md#ttfv-sintetico-en-ci-15-min-axe-core))
- [ ] HITO: v0.3 + lista de espera con política de privacidad. **La demo alojada sale de aquí y sube a la v1** (corrección de rumbo del 02-09-2026): a quien espera SaaS sólo lo convence probarlo sin instalar

## Etapa 3 (6-8 FdS): corpus, entrega firmada y vigilancia

> **Se llamaba «corpus y venta legal» hasta el 08-09-2026.** D-23 se llevo la venta legal a E8 y el titulo se queda con lo que hay dentro: transcribir corpus, poder entregarlo verificado (licencia, descarga firmada, compatibilidad N-1) y vigilar el BOE. Lo que se fue es constituir el negocio; lo que se queda es poder entregar el producto. El titulo se corrige en el mismo commit que mueve las casillas, porque un encabezado que nombra algo que ya no esta dentro es prosa caducada en el sitio donde mas se lee.
- [x] Extensión Anexo B construida: clase_e2e con facetas, temporalidad con régimen, escalado, pruebas/ de dorados, linter con controles negativos, Y el ejecutor de dorados contra el motor real (nucleo/corpus/dorados.go)
- [x] Los 30 marcos montados como paquetes con su estratificación legal correcta y linter en CI (paquetes/CORPUS.md): ens con art. 31 bienal + INES anual, rgpd con art. 33 (72 h), cra con art. 14.1 (24 h, vigente 11-09-2026); 12 dorados ejecutándose contra el motor. ([por qué](docs/casillas.md#los-30-marcos-montados-como-paquetes-con-su))
- [x] Test de integración del ciclo e2e (ciclo_e2e_test.go). La flecha que faltaba existe: el paso 9 hace el borrado legal SOBRE el expediente que se acaba de verificar, retira la clave divulgada, pone la lápida y comprueba que sigue verificando e informa la supresión con su base legal. Lo que aún no encadena sigue dicho al final del test, ahora completo

### El orden de la etapa 3 cambia el 26-08-2026: primero el motor, luego el corpus

**NO SE ESCRIBE UNA LÍNEA DE CORPUS hasta que las dos primitivas de abajo existan y tengan dorados.** El orden pasa a ser:

**primitivas → dorados de las primitivas → censo terminado → corpus**

**El porqué, sin adornos:** escribir los 31 relojes del CRA sobre un motor incompleto es escribirlos mal y volver a escribirlos. No es una cuestión de elegancia: un reloj que el motor no sabe calcular se transcribe **aproximado**, el dorado se escribe contra la aproximación, y el día que la primitiva llega hay que revisar los 31 sin ninguna puerta que diga cuáles estaban mal.

**Y esto queda escrito como mérito del censo, porque para esto servía:** midiendo el corpus se encontró un **hueco del motor**. El censo (`docs/censo-relojes.md`) no era contenido, era la medición que decide el orden de todo lo que queda, y lo primero que ha decidido es que el corpus no empieza todavía.

- [x] **Primitiva: el máximo de dos duraciones** ([por qué](docs/casillas.md#primitiva-el-maximo-de-dos-duraciones))
- [x] **Primitiva: el preaviso contractual, que se calcula AL REVÉS** ([por qué](docs/casillas.md#primitiva-el-preaviso-contractual-que-se-calcula-al))
- [x] **Comprobado antes de escribir corpus: qué necesita la familia A del motor** ([por qué](docs/casillas.md#comprobado-antes-de-escribir-corpus-que-necesita-la))
- [ ] Censo de relojes terminado contra las dos primitivas nuevas: reclasificar lo que se había anotado como "el motor no lo cubre". **Las dos primitivas ya están (26-08-2026), así que esto es reclasificar, no construir**

- [x] **Paquete `nis1-es` (RD 43/2021), la tabla 3 del anexo transcrita con sus relojes.** ([por qué](docs/casillas.md#paquete-nis1-es-rd-43-2021-la-tabla))
- [x] **Paquete `ai-act` (Reglamento (UE) 2024/1689): el art. 50 primero, y el art. 73 con su vigencia en discusión.** ([por qué](docs/casillas.md#paquete-ai-act-reglamento-ue-2024-1689-el))
- [x] **Paquete `iso42001` referencial (ISO/IEC 42001:2023), misma forma y coste marginal que `iso27001`.** ([por qué](docs/casillas.md#paquete-iso42001-referencial-iso-iec-42001-2023-misma))
- [ ] **Equivalencias `iso42001`↔`ai-act` en formato propio.** ([por qué](docs/casillas.md#equivalencias-iso42001-ai-act-en-formato-propio))
- [x] **La familia A del censo, cerrada de una tacada** ([por qué](docs/casillas.md#la-familia-a-del-censo-cerrada-de-una))
- [x] **`plazum calendario`: los relojes existen y ahora se ven** ([por qué](docs/casillas.md#plazum-calendario-los-relojes-existen-y-ahora-se))
- [ ] **Familia B del censo: la periódica anual, ocho marcos.** ([por qué](docs/casillas.md#familia-b-del-censo-la-periodica-anual-ocho))

> **D-19 acota el corpus de esta etapa a los 12 del escaparate** (01-09-2026). Las casillas de autoría de abajo se cierran **para los 12** y no para los 30: `mica`, `eidas2`, `mdr`, `psd2-es` y `eni` se quedan como extra con lo que ya tienen, y el resto del censo pasa a autoría continua post-v1. **El orden deja de ser por marco y pasa a ser por familia de reloj**, empezando por los dos que tienen fecha encima y que el propio corpus ya trae verificados con su cita: **AI Act art. 111.4, límite 02-12-2026** (lo fija el apartado, añadido por el Reglamento (UE) 2026/1744) y **CRA art. 14, aplicable desde el 11-09-2026**. Después LOPDGDD y Ley 2/2023, y después los rituales de las tres referenciales nuevas (SOC 2, PCI DSS, TISAX), que el censo tiene en «no verificado»: **primero se cuentan con cita** (invariante 10), luego se montan.

- [x] Paquete ISO 27001 referencial completo (id + título corto, rituales, cadencias). **Cerrado el 03-09-2026**: 132 obligaciones (30 cláusulas + los 93 controles del anexo A + 9 rituales de plazum), 9 relojes y 27 dorados, con cero texto normativo. Las cadencias son rituales con `origen_del_intervalo: propuesto`, su justificación y su `cuando_cambiarlo`
- [ ] Paquete ENS transcrito completo con dorados por reloj (partir de paquetes/ens semilla)
- [ ] Equivalencias ENS↔ISO **en formato propio** + la lista de huecos computada. OSCAL no: su modelo no tiene dónde poner un plazo y hacerle ida y vuelta borra el diferenciador (`docs/decisiones.md` D-1)
- [ ] **PUERTA D15 sin firma externa (D-20)**: la legalidad del corpus se verifica con los **estratos ejecutables** (el linter, que ya corre en CI), **fuentes primarias con el invariante 10** (cada dato con qué se miró, dónde y qué día, en el cuerpo del commit) y **el descargo**. Es la puerta que sustituye a la revisión jurídica externa, y es más cara de falsificar: una firma es un PDF, un historial de verificaciones fechadas no
- [ ] **`nis2-es` preparado como release de vigilancia, ANTES de que exista la ley.** ([por qué](docs/casillas.md#nis2-es-preparado-como-release-de-vigilancia-antes))
- [ ] **Marcar `observable` y declarar su prueba en los 12 marcos de la v1.** ([por qué](docs/casillas.md#marcar-observable-y-declarar-su-prueba-en-los))
  **Y la casilla trae su propia prueba, porque «preparado» sin medida es una palabra**: se escribe en ella **cuánto se tarda desde la publicación en el BOE hasta el release**, medido en un ensayo con una norma cualquiera ya transcrita. **Si la respuesta no es «horas», el esqueleto no está lo bastante preparado** y la casilla no se marca. El día D puede caer entre este otoño y bien entrado 2027, así que esto no es una apuesta a largo plazo: es una póliza que puede cobrarse pronto
- [ ] Política de compatibilidad N-1 escrita y con test contra artefactos de la release anterior
- [ ] Formato del fichero de licencia Ed25519 y su verificación (emisión manual; el checkout llega en E8)
- [ ] Entrega del corpus: descarga HTTP firmada autenticada contra esa licencia
- [ ] Vigilancia normativa: 2-3 h/semana fijas desde aquí (restadas del calendario)
- [ ] Página de vigilancia pública: tabla fecha-BOE → fecha-paquete autogenerada
- [ ] Plan de continuidad v1 publicado: la página "si me pasa algo, ocurre esto" + segundo juego de llaves de release en custodia

## Etapa 4 (6-10 FdS): continuidad, personas e incidentes
- [x] Ingesta manual firmada (adelantada aquí: fuente de UAR y formación): `nucleo/censo`, con la ley de conservación sobre líneas y un contador independiente que NO sabe de comillas, porque uno que las respetara se tragaría las mismas filas que el parser. `plazum accesos` la sube, dice qué ha entendido antes de dar un número y la anota en el ledger
- [x] Objeto Incidente mínimo: registro + timeline bitemporal + obligaciones notificatorias; payload de notificación del art. 33.3 (el mapeo al formulario concreto de la AEPD necesita ese formulario como fuente primaria y queda pendiente: decir que es «el de la AEPD» sin haberlo mirado sería afirmar algo que no se ha comprobado)
- [ ] Escalado (email + Teams) con jerarquía SCIM y colapso de niveles
- [ ] Ventanas de silencio auditadas + cambio material con diff de paquetes
- [ ] Atestación de políticas (obligación-persona anual, registro al ledger)
- [ ] Formación: tracking + quizzes SOLO de normas transcritas (ENS, RGPD, NIS2)
- [ ] On/offboarding por evento SCIM con SLAs
- [x] **UAR con snapshot firmado** ([por qué](docs/casillas.md#uar-con-snapshot-firmado))
- [x] **Auditoría interna 9.2 con arrastre entre ciclos** ([por qué](docs/casillas.md#auditoria-interna-9-2-con-arrastre-entre-ciclos))
- [x] **Acta 9.3 autogenerada + board pack (LA demo)**. ([por qué](docs/casillas.md#acta-9-3-autogenerada-board-pack-la-demo))
- [ ] HITO: demo de venta (acta 9.3 + UAR + relojes, 2 min) + calendarios pais NIS2 publicados. **PARTIDO el 08-09-2026 por D-23, y partido en vez de movido a proposito**: «primer cliente del corpus» exige a alguien de fuera y se va a E8, y las otras dos mitades son producto y se quedan. Mover el hito entero habria sacado del conjunto bloqueante los calendarios pais de NIS2, que no dependen de nadie

## v1: la plataforma guiada (D-19). Lo que bloquea la salida, y nada más

> **La v1 no es "el motor con pantallas": es el camino guiado de punta a punta sobre `plazum serve`.** Lo que ya está hecho y la sostiene: las 6 pantallas con derivación a un clic, OIDC, SCIM, export SIEM, el mecanismo de catálogo es/en, el calendario y el escalado. Lo que falta es lo de abajo. Las casillas de corpus, licencia y entrega firmada viven en la etapa 3 y las de acta y UAR en la etapa 4: aquí no se duplican, se exigen.

- [x] **El camino guiado, de punta a punta y sin salir de `plazum serve`** ([por qué](docs/casillas.md#el-camino-guiado-de-punta-a-punta-y))
- [x] **Catálogo de interfaz completo en EN.** El mecanismo está desde la etapa 2 (`adaptadores/catalogo`) y lo que falta es el contenido, con el inventario en verde
- [x] **Selector de idioma en el armazón, visible en las ocho plantillas y con la elección persistida.** Hasta hoy el idioma lo decidía SOLO `Accept-Language`, así que un inglés con el navegador en castellano no podía leer plazum en inglés y las 589 claves traducidas no le servían de nada. Es enlace y no formulario (la CSP no admite nada inline y las pantallas son GET-only), y la elección se recuerda en cookie. Lo vigila `TestLasOchoPantallasRespondenEnElIdiomaElegido`, en las dos direcciones y con las tres formas de la nada del invariante 8
- [ ] **El derecho de la UE en inglés, transcrito de la versión oficial vía Cellar** (paquete o variante con su propia fuente), jamás traducido por nosotros (D-11). El derecho nacional (ENS, LOPDGDD, Ley 2/2023, RD 43/2021) **se queda en español y la interfaz inglesa lo dice honestamente**, no lo disimula
- [x] **PUERTA D11-a, cero formaciones.** ([por qué](docs/casillas.md#puerta-d11-a-cero-formaciones))
- [x] **PUERTA D11-b, todo estado vacío trae su siguiente paso, con test.** ([por qué](docs/casillas.md#puerta-d11-b-todo-estado-vacio-trae-su))
- [x] **PUERTA D11-c, cada número clicable hasta su derivación.** ([por qué](docs/casillas.md#puerta-d11-c-cada-numero-clicable-hasta-su))
- [x] **PUERTA D11-d, el camino guiado es determinista.** La IA propone **detrás**, jamás delante (invariante 9). Se comprueba con el mismo arnés que `PLAZUM_SIN_IA=1`: el camino entero tiene que funcionar con la IA apagada
- [x] **PUERTA D11-e, TTFV por debajo de 15 minutos medido sobre el camino COMPLETO.** ([por qué](docs/casillas.md#puerta-d11-e-ttfv-por-debajo-de-15))
- [x] **Las familias de guardas del núcleo alcanzan a las pantallas.** ([por qué](docs/casillas.md#las-familias-de-guardas-del-nucleo-alcanzan))

### El bloque IA de adopción, dentro de la v1 (D-20)

> **Entra en la v1 porque es donde está la fricción, no porque toque.** Las cinco piezas son las de `docs/ia.md` §4.1 y §4.2, y las dos primeras casillas se **adelantan de E5** por dependencia dura: sin búsqueda no hay dónde buscar la cita, y sin verificador una propuesta no se puede enseñar. Todo lo demás de E5 se queda detrás.

- [x] **Búsqueda BM25 sobre el corpus y sobre lo que sube el cliente.** Los embeddings salen de la casilla por D-24: un índice que necesita un modelo para construirse rompe que la búsqueda funcione con `PLAZUM_SIN_IA=1` ([por qué](docs/casillas.md#busqueda-fts5-bm25))
- [x] **Verificador de citas por hash**, determinista, corriendo en cada PR con sus adversariales. **Adelantado de E5** ([por qué](docs/casillas.md#verificador-de-citas-por-hash-determinista-corriendo-en))
- [ ] **Pieza 1, entrevista asistida**: el cliente suelta su política, su inventario y su Excel de controles, y el sistema propone cada respuesta **con la cita del documento y la página**. Es la mayor mejora de tiempo hasta el primer valor de todo el producto
- [x] **Pieza 2, la pregunta con su consecuencia al lado** ([por qué](docs/casillas.md#pieza-2-la-pregunta-con-su-consecuencia-al))
- [x] **Pieza 3, mapeo de la evidencia que ya tiene**: qué documento suyo satisface qué obligación, con cita ([por qué](docs/casillas.md#pieza-3-mapeo-de-la-evidencia-que-ya))
- [x] **Pieza 4, plan de los primeros 30 días** en la pantalla Hoy: 130 obligaciones en rojo es un muro, y la IA agrupa por trabajo (*«estas catorce se cierran con una sola auditoría»*) ([por qué](docs/casillas.md#pieza-4-plan-de-los-primeros-30))
- [x] **Pieza 7, extracción de metadatos de la evidencia**: sube un PDF y se proponen fecha, alcance, firmante y caducidad ([por qué](docs/casillas.md#pieza-7-extraccion-de-metadatos-de-la))
- [x] **PUERTA: el camino completo en verde con `PLAZUM_SIN_IA=1`.** ([por qué](docs/casillas.md#puerta-el-camino-completo-en-verde-con-plazum))
- [x] **PUERTA: evals adversariales del subconjunto, con la inyección vía documento incluida.** ([por qué](docs/casillas.md#puerta-evals-adversariales-del-subconjunto-con-la-inyeccion))
- [x] **PUERTA: ni una pestaña de chat.** ([por qué](docs/casillas.md#puerta-ni-una-pestana-de-chat))


### El eslabón de la prueba (D-22)

> **Está construido y desconectado, y el eslabón que falta es de DATOS.** `puertos.Recoleccion`, `estado.Observacion`, `estado.Prueba`, `estado.Excepcion` y `estado.Calcular` existen desde la etapa 1; el corpus ya declara `facetas` y `recursos`. Lo que no hay es forma de que un paquete diga **qué prueba comprueba qué obligación**, así que `estado.Calcular` lo llama un solo sitio (`nucleo/expediente/expediente.go:717`) y `puertos.Recoleccion` no lo implementa nadie. El porqué entero, con las siete piezas y sus cifras, en `docs/decisiones.md` D-22.
>
> **Entra en la v1 porque destapa trabajo que ya estaba dentro de dos casillas de la v1**, las piezas 3 y 4 de la IA: las dos necesitan saber qué se comprueba, y hoy están escritas como si mapear evidencia contra una obligación fuese lo mismo que mapearla contra una prueba. La obligación dice qué exige la ley; la prueba dice qué se mira para saber si consta.

- [x] **A1. Bloque `pruebas` en el esquema de paquete, con su linter** ([por qué](docs/casillas.md#a1-bloque-pruebas-en-el-esquema-de-paquete))
- [x] **A2. `estado.Calcular` cableado a la pantalla de controles** ([por qué](docs/casillas.md#a2-estado-calcular-cableado-a-la-pantalla-de))
- [x] **A3. `adaptadores/recoleccion/manual`** ([por qué](docs/casillas.md#a3-adaptadores-recoleccion-manual))
- [x] **A4. PUERTA de extremo a extremo: una observación metida a mano llega hasta `verify` offline y sale verificable.** ([por qué](docs/casillas.md#a4-puerta-de-extremo-a-extremo-una-observacion))
- [x] **A5. `ErrorRecol` como la tercera forma de la nada aplicada a la recolección** ([por qué](docs/casillas.md#a5-errorrecol-como-la-tercera-forma-de-la))
- [x] **A6. Los 12 paquetes vacíos fuera de `paquetes/`.** ([por qué](docs/casillas.md#a6-los-12-paquetes-vacios-fuera-de-paquetes))
- [ ] **Frescura de evidencia como segunda familia de relojes.** *(Se mueve aquí desde la etapa 4 el 07-09-2026: no es de personas, es de esto. El TTL del que sale la frescura vive en la prueba, y sin bloque `pruebas` no hay de dónde sacarlo.)*

- [ ] **Demo alojada pública, efímera, con reset horario y sin registro** (~10 €/mes, sin LLM expuesto; definida en `docs/guia.md` §4.6). **Sube desde E3 y bloquea el hito**: el self-hosted gana en regulado y en ENS, pero a quien espera SaaS sólo lo convence probarlo sin instalar, y una puerta de D11 que dice «cero formaciones» no la puede comprobar nadie de fuera si no hay dónde entrar
- [ ] HITO: **v1 publicada** con los 12 marcos, en español e inglés, con el bloque IA de adopción dentro **y con la demo alojada en pie**. ([por qué](docs/casillas.md#hito-v1-publicada-con-los-12-marcos-en))

## Etapa 5 (post-v1): la IA de análisis y agentes — **PARTIDA por D-20**

> **Lo adelantado a la v1 NO está aquí, está en el bloque IA de la sección v1**: FTS5, el verificador de citas por hash y las cinco piezas de adopción (1, 2, 3, 4 y 7). Se dice para que no se cuenten dos veces ni se den por pendientes de esta etapa.
>
> **Lo que queda aquí sirve a quien YA adoptó**, y por eso va detrás: las de arriba consiguen que adopte (`docs/ia.md` §6).

- [ ] Propuestas con revisión por trozos para el redactor documental y la remediación (piezas 8 y 9 de `docs/ia.md`); nunca se aplican solas
- [ ] Runtime de agentes: acciones tipadas, presupuesto, allowlist, transcript cifrado al ledger
- [ ] Agente 1: contradicciones; Agente 2: huecos de evidencia; Agente 3: cuestionarios entrantes
- [ ] MCP server de solo lectura con tokens de alcance; corpus como skills
- [ ] **Pieza 12, notas de alcance de la vigilancia normativa** (borrador desde el diff del BOE, verificado por una persona). **Sube de prioridad por D-20**: es lo que hace sostenible la suscripción de vigilancia por una sola persona, y es IA aplicada a nuestro coste, no a la experiencia del cliente
- [ ] Evals: nightly + release con modelo fijado y media de N; publicados en release notes
- [ ] HITO: "el primer GRC que publica la precisión de su IA"

## Etapa 6 (5-7 FdS): conectores — **LA PRIMERA DESPUÉS DE LA v1** (D-19 la sacó de la v1, la corrección de rumbo del 02-09-2026 la pone delante de E8 y E7)
- [ ] SDK WASM (Extism): ABI v1 (describe/collect/health; http_fetch con allowlist, secret_get, log)
- [ ] Suite de conformidad pública y gratuita + plantilla Go→WASM
- [ ] **Delegados: Prowler, OpenSCAP, Trivy, ScubaGear vía OCSF.** ([por qué](docs/casillas.md#delegados-prowler-openscap-trivy-scubagear-via-ocsf))
- [ ] **2 propios y en profundidad: Entra ID y GitHub** (sin agente propio). Son los que dan **evidencia de accesos** y **evidencia de cambio de código**, o sea las dos fuentes que alimentan la UAR y la auditoría interna que ya están construidas. Dos conectores que cierran un ciclo completo valen más que cuatro que dejan cuatro ciclos a medias
- [ ] **Google Workspace e Intune/Jamf**: no se construyen hasta que **haya un usuario que los pida**. No están aplazados por falta de tiempo, están fuera hasta que exista demanda con nombre
- [ ] Evidencia con procedencia y NO corroborada por defecto; corroboración exigible en críticas
- [ ] Canario diario contra cuentas sandbox reales (fuera del pipeline de PR)
- [ ] Slack + Jira como canales; MCP client por Recolección
- [ ] Export OSCAL **con pérdidas declaradas** (catalog/group/control/part no tiene campo de plazo, así que el reloj no sale) + Mapping Model publicado a partir de las equivalencias en formato propio
- [ ] HITO: pilotos Cloud (máx. 5, gratuitos, acuerdo escrito, datos mínimos, horas/tenant medidas)

## Etapa 7 (4-6 FdS): riesgos y MAGERIT — **POST-V1 por D-19, y la ÚLTIMA del orden post-v1** (E6 → E8 → E7)

> **Depende del modelo de control que nace en el bloque `pruebas` (D-22).** Hacer MAGERIT antes obliga a **inventar el modelo de control dos veces** y luego reconciliarlos: el análisis de riesgos cuelga los riesgos de controles, y un control sin prueba declarada es un nombre. Es el mismo argumento por el que E7 ya iba la última del orden post-v1, ahora con su causa concreta.
- [ ] MAGERIT v3 + taxonomía ENISA como paquetes + crosswalk ENS/ISO 27005
- [ ] 3 niveles de análisis con semilla fija; aceptación caducable; tratamiento genera obligaciones
- [ ] Paquete DORA transcrito (obligaciones y relojes; el RoI validado es AÑO 2)
- [ ] Paquete AI Act (inventario, art. 4, art. 50) + plantilla SRP del CRA
- [ ] HITO: paquetes publicados con dorados

## Etapa 8 (5-7 FdS): el dinero y la confianza — **segunda del orden post-v1** (E6 → E8 → E7)
- [ ] SL constituida (disparador: primer piloto Cloud o 5.000 € acumulados, lo primero)
- [ ] Checkout Stripe con Stripe Tax doméstico ES primero; licencia Ed25519 offline
- [ ] Cloud GA con el runbook de 8 piezas (bóveda secretos, OIDC/SCIM por tenant, incidentes+status, brechas, drill por tenant, baja con certificado de borrado, email transaccional, SLA de horario laboral)
- [ ] DPA con subencargados nominados
- [ ] Carpeta de compras autogenerada + portal de confianza con clickwrap
- [ ] Consola de cartera para partners v1 (solo lectura, N instancias)
- [ ] Pentest externo publicado (4-8k € del primer ingreso)
- [ ] Plan de continuidad completo: escrow formal del corpus y claves, 12 meses de fin de vida contractual, extensión automática de suscripciones (la v1 se publicó en E3)
- [ ] **PUERTA D14 sin referencias de partners (D-20)**: el open core self-serve se verifica con **tres meses de medición real de uso** más el **checkout operando**. Es la puerta que sustituye al programa de design partners, y mide lo que pasa, no lo que alguien dice que pasó
- [ ] **ACELERADOR, ya no puerta (D-20)** ([por qué](docs/casillas.md#acelerador-ya-no-puerta-d-20))
- [ ] Venta legal: autónomo + seguro RC profesional + Stripe Payment Link + contrato con tope 12 meses **Sale del conjunto bloqueante de la v1 el 08-09-2026, por D-23**: es constituir el negocio, no construir el producto. Nada de esto cambia una linea de plazum ni lo que un evaluador puede hacer con el, y E8 se llama justamente «el dinero y la confianza» y ya tiene dentro el checkout y la licencia
- [ ] **ACELERADOR, ya no puerta (D-20)**: programa de design partners, 5 con nombre, 50% de por vida, logo + llamada de referencia. La puerta que lo sustituye es la de D14 en la etapa 8, tres meses de medición real de uso **Sale del conjunto bloqueante de la v1 el 08-09-2026, por D-23**: exige **cinco clientes con nombre**, o sea que la fecha de la v1 la decidirian cinco personas que todavia no saben que existimos. Su propia linea ya dice que la puerta que lo sustituye vive en E8: esto es mudarlo al lado de ella
- [ ] HITO: v0.4 + primera venta posible + 5 consultores contactados **Sale del conjunto bloqueante de la v1 el 08-09-2026, por D-23**: **exige un cliente contactado**, y un hito asi no se cierra programando. La mitad que si depende de nosotros —que v0.4 exista y se pueda comprar— la cubren las casillas de licencia y de entrega firmada, que se quedan
- [ ] Kit mínimo de partner: acuerdo de margen 40% + demo grabada **Sale del conjunto bloqueante de la v1 el 08-09-2026, por D-23**: es material de canal. Un acuerdo de margen es un documento comercial, y la demo grabada se hace cuando hay a quien ensenarsela
- [ ] Primer cliente del corpus. **Sale del conjunto bloqueante de la v1 el 08-09-2026, por D-23**: era la mitad del hito de venta de la etapa 4 que exige a alguien de fuera; las otras dos mitades se quedaron alli
- [ ] HITO: Cloud GA + 9,7 en camino de verificación (D14/D11 con 3 meses de medición real)

## Año 2 (apuntado, sin casillas)
Postgres, SAML, RoI DORA con subconjunto de reglas EBA + Arelle, resto del catálogo (NIST importado, ISO 22301/42001, SOC 2, PCI, TISAX referenciales, CIS/STIG delegados), consola de cartera con marca blanca, certificar el propio Cloud usando plazum, partner jurídico DACH y alemán.
