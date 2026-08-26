# plazum: la guía definitiva de construcción

**24 de agosto de 2026. Undécima ronda, y la última.** Este documento sustituye a `grc-guia-construccion.md` y a las secciones de plan de `grc-definitivo.md`: **es la fuente única del plan**. Donde el diseño y esta guía difieran, manda esta guía.

**Qué la hace definitiva.** La versión anterior pasó por dos revisores más: un cazador de fallos del plan, que encontró **42** (6 bloqueantes, 17 graves, 19 menores), y un panel de tres compradores simulados con criterio duro (CISO español de 300 empleados, responsable alemán de 800 bajo NIS2, consultora española de 8 personas con 14 clientes) que puntuó **cada feature por su peso en la decisión de pago** y dejó la nota en un 6/10 con el camino exacto al 9-10. Esta versión integra los 42 arreglos y las 5 adiciones del panel. Ningún fallo conocido queda sin arreglo asignado.

---

## 0. Lo que las dos revisiones cambiaron, en una página

**Los seis bloqueantes del plan anterior y su arreglo, ya integrado:**

| # | Fallo | Arreglo (sección) |
|---|---|---|
| 1 | No existía almacén de ficheros de evidencia; Litestream solo replica SQLite | blobs content-addressed DENTRO de SQLite, cifrados por entrada: un solo fichero que respaldar (E1) |
| 2 | El borrado por destrucción de clave era reversible vía backups/WAL, o ilegible tras restore | keystore separado con su política de réplica y retención de backups alineada con el borrado (E1) |
| 3 | Primera venta en el mes 5-6, capacidad de facturar en el mes 14-16 | venta desde autónomo con RC profesional y contrato con tope desde la etapa 3; SL antes del primer tenant Cloud (E3, §12) |
| 4 | Altas del Cloud antes que el DPA, la SL y la automatización | los 5 primeros tenants son pilotos gratuitos con acuerdo escrito y datos mínimos; GA solo tras la etapa 8 (E8) |
| 5 | El problema del vigilante: la instancia muere y los relojes vencen en silencio | latido opt-in (si tu instancia calla 24h, te avisamos) + smoke test del canal de notificación + estado del planificador en "Hoy" (E2) |
| 6 | Calendario optimista sin mantenimiento | replanificado con factor 1,5-2x: **24-27 meses** (45-67 fines de semana), con el 25% de cada fin de semana reservado a mantenimiento desde el primer release público (§13) |

**Lo que el panel de compradores enseñó, y que reordena el plan:**

La tabla de "peso en la decisión de PAGO" (0-10, media de los tres compradores):

| Lo que paga | Media | | Lo que no paga | Media |
|---|---|---|---|---|
| Motor de plazos con citas | **8,3** | | Expediente verificable offline | 4,0 |
| Acta 9.3 autogenerada | **8,0** | | Escalado (per se) | 4,0 |
| Cross-framework con lista de huecos | **7,7** | | Portal de confianza | 3,7 |
| UAR con snapshot | **7,0** | | MCP | 1,3 |
| Atestación de políticas | **6,7** | | RoI DORA validado | **0,7** |

Cuatro decisiones salen de ahí:

1. **La demo y el pitch son la capa aburrida**: relojes con cita, acta 9.3, lista de huecos, UAR, atestación. El expediente Merkle y el MCP son confianza e infraestructura: se construyen igual (sostienen el producto y el 9,7), pero **no encabezan ninguna venta**.
2. **El RoI de DORA se va al año 2.** Peso 0,7 en la decisión de pago del ICP y era la pieza más infraestimada del plan (las reglas EBA completas en local son un proyecto entero). El paquete DORA (obligaciones y relojes, transcrito) se queda en el año 1; el generador validado, no. Libera 4-6 fines de semana.
3. **Teams antes que Slack.** El CISO español de 300 empleados vive en Teams. Orden de canales de notificación: email, Teams, Slack, Jira.
4. **La consola de cartera para el partner entra en el plan.** La consultora es el comprador con más probabilidad (55% a 12 meses) y su dealbreaker era exactamente este: 14 clientes son 14 instancias sin vista agregada. V1 de solo lectura en la etapa 8.

**Y la lección que manda sobre todas:** la nota de "¿pagarán al salir?" era un 6/10 y **lo que falta no se compila**. Componente de confianza: 3/10 (mantenedor único, sin entidad, sin referencias, sin pentest). Por eso esta guía añade el **programa de confianza** (§12) como carril paralelo con el mismo rango que el código: plan de continuidad verificable, design partners con nombre, SL y DPA en el primer euro, pentest externo publicado, y certificar el propio Cloud usando plazum.

---

## 1. La regla del 9,7, actualizada

Igual que antes (cada dimensión sube cuando su puerta está en verde), con dos honestidades nuevas que impuso el revisor:

- **D14 (open core)** ya no se declara con extrapolaciones: su puerta es "presupuesto de horas por tenant proyectado con **al menos 3 meses de medición real** y automatización demostrada en drill". Se cierra hacia el mes 22-24, no antes.
- **D11 (intuitividad)** cierra su último tramo con telemetría real de TTFV de usuarios en producción, no solo con el sintético de CI.

El resto de puertas no cambia. **El 9,7 global se verifica alrededor del mes 24**, con la mayoría de dimensiones cerradas entre los meses 14 y 20.

---

## 2. Semana 0: fundaciones (ampliada con lo que faltaba)

Todo lo de antes (estructura del repo, política de dependencias con test, CLA con concesión de relicenciar, AGPL con cabeceras, registro de imágenes Docker para la imagen del producto en E2, dominio provisional) más lo que las revisiones exigieron:

**Seguridad desde el commit uno:**
- `SECURITY.md` con divulgación coordinada y el private vulnerability reporting de GitHub activado. Un producto de seguridad público no puede pasar 14 meses sin canal de reporte, y el CRA lo exige a quien comercializa, que es plazum desde el mes 5-6.
- CI de seguridad: **govulncheck, gosec, CodeQL y dependabot** (una hora de configuración), y **fuzzing** del parser de corpus, del ledger y del verificador como puerta de la etapa 1: son exactamente las tres piezas que se fuzzean.
- El propio cumplimiento CRA de plazum (SBOM, CVD, ventana de soporte declarada) arranca aquí, no en la etapa 8.

**La marca, antes de invertir en ella:** búsqueda de anterioridades y solicitud EUIPO (clases 9 y 42, ~850 €) en la semana 0. Estado real a fecha de hoy: la búsqueda sobre "Plazum" está pendiente y no ha arrojado hallazgos, de modo que no consta anterioridad ni consta vía libre. (Corrección: el aviso que figuraba aquí sobre una fintech homónima y un término financiero común en alemán se refería al nombre anterior del proyecto; el renombrado a Plazum lo convirtió en una afirmación falsa y se retira. No debe reintroducirse sin una búsqueda hecha sobre "Plazum".) **Decisión de nombre definitivo antes del primer release público**, con alternativa preparada si la búsqueda pinta mal.

**La propiedad intelectual del autor:** revisar las cláusulas de PI del contrato de empleo o de los contratos de consultoría activos **antes del primer release público**, y documentar la cadena de titularidad para la futura aportación a la SL. Es un fallo de los que no se ven hasta que cuestan el proyecto.

**Dependencias, lista cerrada y ampliada:** a las cinco de antes (modernc/sqlite, cel-go, extism-go-sdk, x/crypto, x/oauth2) se añaden, con su porqué en DEPENDENCIAS.md: una librería de verificación RFC 3161 (el ASN.1/CMS a mano son semanas), y **nada más**. Decisión que simplifica dos etapas: los paquetes de corpus **se firman con Ed25519 propio** (la misma criptografía del ledger y de la licencia), no con cosign: fuera la dependencia de sigstore y fuera el cliente OCI; la distribución del corpus es **descarga HTTP firmada autenticada contra la licencia** (§13), no GHCR con tokens por cliente.

---

## 3. Etapa 1 (4-6 fines de semana): el núcleo probatorio, completo de verdad

Lo de antes (ledger v2 con compromiso de clave, lápidas firmadas, historia bitemporal, Certificado, perímetros, anclaje RFC 3161) más los dos bloqueantes técnicos:

### 3.1. El almacén de evidencia: blobs dentro de SQLite

Toda evidencia con fichero (PDF del acta, captura, informe del auditor) se guarda **content-addressed dentro de SQLite**: tabla `blobs(hash, datos_cifrados, tamano)`, cifrada por entrada con el mismo régimen del ledger (AEAD con compromiso de clave), chunking a partir de 32 MB. Por qué dentro y no en disco: **un solo fichero que respaldar**: Litestream replica la base y con ella viaja todo (evidencias incluidas), `plazum update` hace backup de una cosa, el restore drill restaura una cosa, y el runbook del Cloud no tiene una segunda ruta de backup que olvidar. El coste (base más gorda) se acepta y se mide: presupuesto en CI de tamaño de base con la demo.

### 3.2. El keystore: dónde viven las claves para que borrar sea borrar

Las claves por entrada viven en un **keystore separado de la base principal** (fichero propio, cifrado con la clave maestra del operador), con su política explícita:

- El keystore se replica aparte y con **retención corta y declarada** (por defecto 35 días de generaciones).
- La retención de backups de la base queda **alineada con el borrado**: cuando se destruye una clave, el borrado es efectivo en el momento para la instancia viva, y efectivo para el mundo cuando expira la última generación de backup que la contenía. Ese plazo (35 días por defecto) se declara en la política de privacidad y en la lápida.
- La clave maestra del operador (la que firma lápidas y cierra checkpoints) tiene ciclo de vida escrito: generación en el primer arranque, copia de recuperación impresa (frase o QR), rotación documentada, y qué pasa si se pierde (el histórico verifica, no se puede firmar nuevo hasta rotar).
- El restore drill de CI restaura **base más keystore** y verifica tres cosas: la cadena entera, que una entrada borrada sigue ilegible, y que su lápida lista la base legal.

### 3.3. El resto, como estaba, con dos matices

El anclaje RFC 3161 sale con **cadena de reserva**: dos TSAs gratuitas configuradas más cola local con reintento, y el QTSP cualificado documentado como opción (la promesa probatoria no puede colgar de un servicio de aficionado, que fue el aviso del revisor). Y los tests de fuzzing del §2 son puerta de esta etapa.

**Hito público:** release v0.2 firmada, con el post del ledger (el ataque de los invisible salamanders explicado y resuelto viaja solo entre gente técnica).

---

## 4. Etapa 2 (8-12 fines de semana): serve, la UI y el autoservicio, con la seguridad que faltaba

La estimación anterior (4-6) era irreal para lo que contiene, dijo el revisor, y tenía razón: esta etapa es el producto visible. Se reestima a 8-12 e incorpora seis arreglos:

**4.1. Seguridad web como puerta, no como intención.** CSRF en todo POST (htmx con cookie era vulnerable de libro), rate limiting en login y API, cabeceras (CSP, HSTS, X-Frame-Options) verificadas en CI, flujo de primer administrador documentado (token de un solo uso impreso en el arranque, como ya hacía el diseño de `serve`), TLS terminado por el proxy del cliente con guía, o autofirmado con aviso. Test-puerta propio.

**4.2. El latido (el arreglo del vigilante).** Opt-in en la instalación: la instancia manda un pulso diario mínimo (ID de instancia y marca de tiempo, nada más; política de privacidad publicada) a `plazum.dev/latido` (dominio provisional: ver la casilla de marca de la semana 0); si calla 24 horas, aviso al email del operador. Más el smoke test periódico del canal de notificación ("este canal funcionó por última vez hace X") y el estado del planificador visible en "Hoy". Un producto que vende "no pierdas nunca la conformidad" no puede morir en silencio.

**4.3. SCIM con la extensión enterprise.** El atributo `manager` incluido (la jerarquía del escalado sale de ahí, no de los grupos), con mapeo manual en la UI como alternativa para quien no lo puebla.

**4.4. Export a SIEM.** El log de auditoría exportable en JSON líneas desde el día uno (era promesa del diseño sin etapa; es barato y es table stakes).

**4.5. Idiomas: es/en, y el mecanismo.** La promesa de alemán se recorta por escrito hasta que exista el partner DACH que lo revise; el **mecanismo** de i18n de la UI generada (catálogo de cadenas por clave, incluido lo que viene de los paquetes) se diseña ahora aunque cargue dos idiomas.

**4.6. La demo alojada, definida.** Además del `plazum demo` local: una instancia pública efímera con reset horario, sin LLM expuesto y sin registro, con presupuesto (~10 €/mes). Es el "pruébalo lleno en dos minutos" del que depende el self-serve.

**Sistemas operativos, declarado:** Linux de primera clase (systemd y Docker), macOS para evaluación, Windows Server vía Docker; matrix build en CI. Y el descargo "no es asesoramiento jurídico" entra ya en el pie de la UI y en la salida de `explain`.

**Tests-puerta:** los de antes (TTFV sintético, axe cero violaciones, presupuestos, restore drill ahora con keystore y blobs) más el de seguridad web y el del latido.

**Hito:** v0.3, demo alojada pública, lista de espera del Cloud abierta (con política de privacidad y responsable identificado: el fundador como persona física hasta la SL, documentado).

---

## 5. Etapa 3 (6-8 fines de semana): el corpus, la venta legal y los design partners

La guía de autoría no cambia (el pipeline con el ejemplo del art. 31, los casos dorados obligatorios por reloj, la frontera del LLM, ISO referencial sin texto, las equivalencias OSCAL con su álgebra). Se añaden los cinco arreglos que la convierten en una etapa vendible sin bombas:

**5.1. La revisión jurídica externa del corpus español.** El plan exigía jurista para DACH y dejaba el corpus español firmado por una sola persona. Antes de la v0.4: un despacho o el primer consultor-partner revisa el paquete ENS y los relojes del RGPD (pocos miles de euros o margen a cambio), y su revisión consta en el changelog. El "firmado" pasa de firma criptográfica a firma con respaldo.

**5.2. La política de compatibilidad, escrita y testada.** Desde el momento en que alguien paga por paquetes: el paquete de versión N carga en el binario N y N-1; un expediente lo verifica cualquier verificador de su versión o superior; la API se versiona. Test de compatibilidad en CI con artefactos de la release anterior. Sin esto, la primera evolución del formato rompe a los clientes que pagan.

**5.3. Vender desde el mes 5-6, legalmente y sin SL.** Alta de actividad como autónomo (ya existe por la consultoría), **seguro de RC profesional contratado ahora** (con atención a que cubra PI de terceros), Stripe Payment Link (sin checkout completo todavía), y contrato de suscripción del corpus con descargo, tope de responsabilidad al importe de 12 meses y la política de compatibilidad anexa. La SL llega con el primer tenant Cloud o al superar 5.000 € de ingreso, lo que ocurra antes (§12).

**5.4. El programa de design partners, la máquina de referencias.** Cinco organizaciones con nombre (dos vía el consultor-partner, una del entorno ENS art. 2.3, dos de la lista de espera): precio fundador del 50% de por vida a cambio de logo, llamada de referencia y feedback quincenal. Empieza aquí, no al final: las referencias son la feature número uno que ningún sprint compila, dijo el panel, y tenía razón.

**5.5. La vigilancia normativa entra en el calendario.** Desde esta etapa: **2-3 horas semanales fijas** de leer BOE/DOUE y actualizar paquetes, restadas de la capacidad de las etapas siguientes (ya reflejado en el calendario del §13). El foso autodeclarado deja de ser gratis.

**Tests-puerta:** los de antes más compatibilidad N-1 en verde y el linter sobre todos los paquetes publicados.

**Hito comercial:** v0.4 "ISO 27001 + ENS con relojes", primeros 5 consultores contactados con la demo de la lista de huecos, primera venta posible del corpus (autónomo + RC + contrato), y los 5 design partners reclutándose.

---

## 6. Etapa 4 (6-10 fines de semana): continuidad, personas e incidentes

La estimación anterior (3-4) ignoraba que esta etapa absorbió nueve features. Se reestima a 6-10 y se arreglan las tres dependencias rotas:

**6.1. La UAR tiene fuente de datos desde el primer día.** El arreglo del fallo grave: la **ingesta manual firmada se adelanta aquí** (CSV o export del IdP, subido y firmado: era de la etapa 6) y el SCIM de la etapa 2 ya aporta usuarios y grupos. La campaña UAR del año 1 funciona sobre datos importados a mano más SCIM, y se documenta así, sin vergüenza; los conectores de la etapa 6 la vuelven automática. Lo mismo para formación: tracking sobre import manual, KnowBe4/Moodle cuando haya conector.

**6.2. El objeto Incidente, mínimo pero existente.** Tres piezas del plan lo consumían (el acta 9.3, el cronómetro del art. 33, la clase notificatoria) y ninguna etapa lo construía: registro con clasificación, línea de tiempo bitemporal (la de la etapa 1), y vínculo a las obligaciones notificatorias que dispara. El payload del formulario de brecha AEPD se construye aquí con el paquete RGPD; la plantilla SRP del CRA, en la etapa 7 con su paquete.

**6.3. El resto de la capa, como estaba, más lo que faltaba nombrar:** atestación de políticas, formación (quizzes **solo de normas transcritas**: ENS, RGPD, NIS2; los de ISO serían derivados que su licencia veta, y el contenido pedagógico tiene su fila propia en el presupuesto de autoría), on/offboarding por evento SCIM, **auditoría interna 9.2 con arrastre** (estaba anunciada y sin cuerpo: variante de ritual plurianual que re-instancia alcance y expediente), acta 9.3 autogenerada, frescura, escalado (canales de esta etapa: **email y Teams**; Slack y Jira en la 6), ventanas de silencio y cambio material.

**6.4. El kit mínimo de partner, aquí y no en el mes 14.** El canal se contactaba en el mes 5-6 y su kit llegaba nueve meses después: canal enfriado. Ahora: acuerdo de margen sobre el corpus por escrito más la demo grabada del acta 9.3 y la UAR, en cuanto existen.

**Tests-puerta:** los de antes más: incidente que dispara el reloj del art. 33 desde `InstanteHecho`; UAR completa sobre import manual con snapshot verificado; quiz de ENS como datos con su resultado por persona en el expediente.

**Hito:** la demo de venta completa (acta 9.3 + UAR + relojes, 2 minutos), primer cliente de pago del corpus, calendarios país NIS2 publicados como datos abiertos.

---

## 7. Etapa 5 (6-9 fines de semana): la IA, con los evals donde pueden vivir

Contenido como estaba (FTS5 siempre, embeddings opcionales vía Ollama, verificador de citas por hash, propuestas con revisión por trozos, contradicciones, huecos de evidencia, cuestionarios entrantes, runtime de agentes con presupuesto y transcript al ledger, MCP server con tokens de alcance, corpus como skills), con dos arreglos:

**7.1. Los evals, partidos en dos cadencias.** El verificador de citas es determinista y corre en cada PR. Los evals con LLM (extracción, contradicciones) **no pueden ser puerta de cada push** (coste, secretos expuestos a forks, no determinismo que produce rojos aleatorios): corren en nightly y obligatoriamente en release, con modelo y versión fijados y umbral sobre la media de N ejecuciones. Publicados en cada release igual que antes.

**7.2. El MCP client se va a la etapa 6.** Consumía la normalización OCSF y la corroboración que se construyen con los conectores; ahora vive donde sus dependencias.

**Hito:** "el primer GRC que publica la precisión de su IA", con los números de la nightly en la release.

---

## 8. Etapa 6 (5-7 fines de semana): conectores, con canario

Como estaba (SDK WASM con Extism, ABI v1, sandbox por capacidades, suite de conformidad pública, evidencia no corroborada por defecto, 4 propios: Entra ID, Google Workspace, GitHub, Intune/Jamf; delegados Prowler, OpenSCAP, Trivy y **ScubaGear**, que estaba en el diseño y se había caído de la guía), más:

- **Slack y Jira** como canales de notificación (Teams ya está desde la 4).
- **El canario diario**: los conectores se validan en PR contra mocks grabados, y además un job diario contra cuentas sandbox reales de los cuatro proveedores (gratuitas), fuera del pipeline de PR. Los mocks no detectan que Microsoft cambió la API el martes.
- **El MCP client**, que llega de la etapa 5 y entra por Recolección normalizando a OCSF.

**Hito:** con conectores, los **pilotos** del Cloud: máximo 5, gratuitos, con acuerdo escrito y datos mínimos (el arreglo del bloqueante 4: sin SL ni DPA no hay tenant de pago), midiendo horas por tenant desde el primer día.

---

## 9. Etapa 7 (4-6 fines de semana): riesgos y el recorte honesto de DORA

MAGERIT v3 empaquetado con el crosswalk a ENS e ISO 27005, los tres niveles con semilla fija, aceptación caducable, tratamiento que genera obligaciones, proveedores, y el paquete del AI Act (inventario, art. 4 y 50, preparación de diciembre de 2027). Con el recorte que el panel y el revisor impusieron a la vez:

**DORA en el año 1 es el paquete transcrito** (obligaciones, relojes, registro de proveedores como datos). **El generador del RoI validado se va al año 2**, y cuando llegue será con un subconjunto declarado de reglas EBA (las de rechazo frecuente, listadas nominalmente) y Arelle como herramienta externa documentada, no una reimplementación total en Go. Pesaba 0,7 sobre 10 en la decisión de pago del ICP: era orgullo de ingeniería, no producto.

La plantilla SRP del CRA (clase notificatoria) se construye aquí con su paquete.

---

## 10. Etapa 8 (5-7 fines de semana): el dinero, con el runbook completo

Checkout Stripe (con **Stripe Tax configurado primero para el caso doméstico español**, IVA 21%, que es la cabeza de playa; el reverse charge intracomunitario documentado detrás), licencia Ed25519 con activación offline, y el Cloud en GA con el runbook que ahora sí contiene las ocho piezas que el revisor exigió para creerse las 4 horas por tenant:

1. bóveda de secretos multi-tenant (tokens OAuth de cada cliente: cifrado, rotación, revocación)
2. configuración OIDC/SCIM por tenant con guion (la fuente clásica de horas)
3. gestión de incidentes y página de estado
4. comunicación de brechas como encargado (el art. 33 aplica a plazum)
5. restore drill periódico **por tenant real**, no solo el sintético de CI
6. baja de tenant: export completo más certificado de borrado (lo exige cualquier DPA)
7. email transaccional con proveedor y SPF/DKIM (los escalados dependen de que el correo llegue)
8. **el SLA que una persona puede firmar**: respuesta en horario laboral europeo definido, sin 24/7, escrito en el contrato

Más lo que se movió aquí y lo nuevo: la carpeta de compras, el portal de confianza con clickwrap, y la **consola de cartera para partners v1**: vista agregada de solo lectura de N instancias (cada cliente sigue soberano; la consola lee resúmenes vía API con tokens de alcance), que era el dealbreaker de la consultora, el comprador más probable. Marca blanca en el año 2.

**La SL ya existe al llegar aquí** (se constituyó con el primer piloto Cloud o los primeros 5.000 €, §12), así que esta etapa firma DPAs con subencargados nominados (hosting UE, email transaccional, Stripe) y convierte pilotos en clientes.

**Tests-puerta:** compra end-to-end en test; licencia offline; alta de tenant cronometrada; carpeta generada de la instancia propia; drill de baja de tenant con certificado.

---

## 11. La matriz corpus libre / corpus de pago, por fin definida

Era la única línea de ingreso self-hosted y su producto no estaba delimitado (fallo grave 18). La matriz, definitiva:

| | Gratis (todo el mundo) | Suscripción "Corpus firmado" (1.490 €/año) |
|---|---|---|
| Los paquetes | **todos, inmediatos, sin retraso** | los mismos |
| Licencia de datos | Apache-2.0 (adopción máxima; el foso es la vigilancia, no la licencia) | igual |
| Plazo de actualización | mejor esfuerzo, sin compromiso | **plazo objetivo contractual** publicado, con histórico verificable |
| Changelog | técnico | **curado con notas de alcance** (qué te cambia y por qué), sellado RFC 3161 |
| Cambio material | lo calcula tu instancia al actualizar | **aviso proactivo** por email/Teams cuando se publica el paquete |
| Sello de tiempo | en los checkpoints propios | **cada release del corpus, sellada**, enseñable al auditor |
| Preguntas sobre el corpus | Discussions, sin SLA | canal con respuesta en horario definido |
| Track record | público para todos | es el argumento: la **página de vigilancia** con la tabla fecha-BOE → fecha-paquete, generada automáticamente |

La decisión de fondo, coherente con todas las rondas anteriores: **el contenido es gratis e inmediato para todos** (el retraso tipo Snort murió en la ronda 6: el contenido normativo caduca en años). Lo que se paga es el **contrato de servicio sobre el contenido**: el compromiso de plazo con histórico público, el aviso, el sello, las notas y alguien que responde. Es el modelo Red Hat aplicado a datos, y la página de vigilancia con el track record es un artefacto de confianza que ningún competidor tiene ni puede fingir.

La entrega técnica (fallo 38): descarga HTTP firmada, autenticada contra la licencia Ed25519, con la misma activación offline. Sin GHCR, sin tokens por cliente, sin fricción.

---

## 12. El programa de confianza: el carril que no se compila

El panel lo dijo sin anestesia: el producto convence (encaje feature-dolor 8,5/10) y el proveedor no (confianza vendible 3/10). Este carril corre en paralelo a las etapas y tiene el mismo rango que el código:

1. **El plan de continuidad verificable, publicado desde la etapa 3.** Una página: qué pasa si el mantenedor desaparece. Segundo juego de llaves de release en custodia (una persona de confianza con acuerdo escrito), el corpus y las claves de firma en escrow, compromiso contractual de 12 meses de fin de vida ordenado para los clientes de pago, y extensión automática de suscripciones si la vigilancia se pausa más de N semanas. Convierte el bus factor de dealbreaker en riesgo gestionado: es lo que separa el 5% del alemán del 40% del español.
2. **Los 5 design partners con nombre** (etapa 3, §5.4): la máquina de logos y llamadas de referencia.
3. **La SL en el primer euro serio**: con el primer piloto Cloud o al superar 5.000 € de ingreso acumulado, lo que llegue antes. Capital de 1 €, ~400 € de notaría, y desde ese día los DPAs se firman con entidad. El seguro de RC profesional, antes: con la primera venta del corpus (etapa 3).
4. **El pentest externo publicado** (etapa 8, presupuesto 4-8k € del primer ingreso): la carpeta de compras es autodeclaración; el informe de un tercero no.
5. **Certificarse a sí mismo usando plazum** (año 2): el Cloud con su ENS o ISO 27001, con el expediente público. Es a la vez la prueba del producto, el contenido de marketing definitivo y lo que desbloquea al comprador alemán en 2028.

---

## 13. El calendario honesto

Con el factor 1,5-2x del revisor, el 25% de mantenimiento desde el primer release público y las 2-3 horas semanales de vigilancia desde la etapa 3:

| Fase | Fines de semana | Mes (ritmo realista) | Hito |
|---|---|---|---|
| Semana 0 | 1-2 | 1 | repo, CI de seguridad, SECURITY.md, marca |
| E1 núcleo probatorio | 4-6 | 2-4 | v0.2, post del ledger |
| E2 serve + UI + autoservicio | 8-12 | 5-9 | v0.3, demo alojada, lista de espera |
| E3 corpus + venta + partners | 6-8 | 9-12 | v0.4, primera venta (autónomo+RC), design partners |
| E4 continuidad + personas | 6-10 | 12-16 | la demo de venta, kit de partner |
| E5 IA | 6-9 | 16-19 | evals publicados |
| E6 conectores | 5-7 | 19-22 | pilotos Cloud (gratuitos, máx. 5) |
| E7 riesgos + MAGERIT | 4-6 | 22-24 | paquete AI Act, SRP del CRA |
| E8 dinero + confianza | 5-7 | 24-27 | SL→DPA→Cloud GA, pentest, consola de cartera |
| **Total** | **45-67** | **~24-27 meses** | 9,7 verificado ~mes 24-27 |

Sí: es un año más que la versión anterior. La versión anterior era la estimación optimista de cada etapa a la vez, con cero horas de mantenimiento, de comunidad y de vigilancia. Esta aguanta que la vida pase. Y los hitos comerciales no esperan al final: primera venta en el mes 9-12, la demo de venta en el 12-16, pilotos en el 19-22.

**Los disparadores de pivote, actualizados:**

- Si CISO Assistant publica un motor de plazos real: acelerar expediente y corpus español (siguen siendo lo que no pueden copiar sin rediseñar).
- Si en el mes 14 no hay ni una venta de corpus ni 10 consultores interesados ni 3 design partners: el producto sigue como proyecto, el plan comercial se replantea antes de construir el Cloud.
- Si los pilotos no bajan de 4 horas por tenant y año proyectadas: el Cloud se congela y el dinero es corpus más partners.
- El orden de sacrificio si el ritmo no se sostiene: nivel cuantitativo de riesgos, dos de los cuatro conectores, el tercer agente, la consola de cartera v1. **Nunca**: el corpus, los dorados, el expediente, el latido ni el programa de confianza.

---

## 14. El veredicto final: cerca de qué 10 estamos

**En diseño y features (la nota de las 17 dimensiones):** el plan lleva cada dimensión a su puerta de 9,7 con test ejecutable, global proyectado **~9,74 verificado hacia el mes 24-27**. Tras once rondas, no queda ninguna decisión de diseño abierta ni ningún fallo conocido sin arreglo asignado. De un 1 a 10 en "el plan está completo y sin errores a priori": **9,5**. El medio punto que falta es el que ningún plan tiene: los errores que solo aparecen construyendo, y para esos están las puertas de CI, que es exactamente para lo que se diseñaron.

**En "¿pagarán los CISOs por esto como open core?":** el panel puntuó el plan anterior con un **6/10** y esta versión integra sus cinco correcciones (continuidad verificable, design partners, SL y RC en el primer euro, pentest, consola de cartera). Con eso, la salida comercial se proyecta en un **8/10**: el encaje feature-dolor es 8,5, el precio cabe en los tres presupuestos, el canal consultor tiene su herramienta, y la confianza pasa de 3 a 6-7 con el programa del §12. Los dos puntos restantes **no se compilan**: uno es logos y llamadas de referencia (los design partners lo fabrican entre los meses 12 y 20), y el otro es tiempo existiendo (el comprador alemán firma en 2028 si en 2027 el track record de vigilancia y el pentest están publicados). Nadie en el mundo lanza un open core con esos dos puntos puestos: se ganan operando.

**Y la frase que resume las once rondas:** el producto ya no es la apuesta; la apuesta es la constancia. Todo lo que dependía de decidir bien está decidido y revisado por siete adversarios; lo que queda depende de ejecutar 45-67 fines de semana sin abandonar, y ese riesgo no lo cubre ninguna ronda más.


---

## Anexo A: formatos de la etapa 1 (heredados de la décima ronda, fuente para E1)

Los tipos exactos que E1 construye. Nombres orientativos; los campos y las reglas, no.

```go
// Ledger v2: AEAD con compromiso de clave. No existe ledger persistido previo,
// asi que NO hay migracion: v2 es el primer formato en disco.
type EntradaV2 struct {
    Indice     uint64
    Previo     []byte // hash de la entrada anterior
    Nonce      []byte // 12 bytes
    Cifrado    []byte // AES-256-GCM(payload canonico)
    Compromiso []byte // HMAC-SHA256(clave, "plazum/commit/v1" || Nonce)
    Hash       []byte // sha256(Indice || Previo || Nonce || Cifrado || Compromiso)
}

// Borrar = destruir la clave de la entrada en el keystore + anadir lapida.
type Lapida struct {
    EntradaBorrada uint64
    BaseLegal      string // "Ley 2/2023 art. 32" | "RGPD art. 17" | ...
    Instante       string
    Firma          []byte // Ed25519 de la clave maestra del operador
}

// Historia bitemporal: el estado actual es un pliegue de estos eventos.
type CambioEstado struct {
    Prueba           string
    De, A            Estado
    InstanteHecho    string // cuando paso en el mundo
    InstanteRegistro string // cuando lo supo el sistema
    Causa            string // observacion | ritual | excepcion | correccion
}

// El certificado y sus hitos generan obligaciones internas con reloj.
type Certificado struct {
    ID, Marco, Alcance, Emisor, Emision string
    Hitos  []HitoCert
    Estado EstadoCert // vigente | en_vigilancia | suspendido | retirado
}
type HitoCert struct {
    Tipo    string // vigilancia | recertificacion | ventana_observacion | informe
    Ventana Primitiva // del motor de nucleo/ventana
    Genera  []string  // IDs de obligaciones internas que dispara
}
```

Keystore: fichero separado de la base, cifrado con la clave maestra; réplica propia con retención de 35 días declarada; el restore drill restaura base más keystore y verifica cadena, irrecuperabilidad de lo borrado y lápida con base legal. Blobs: tabla content-addressed dentro de SQLite, cifrada por entrada, chunking desde 32 MB.

## Anexo B: el formato de corpus completo y el pipeline de autoría (fuente para E3 y /autoria)

**Primera casilla de E3: extender `nucleo/corpus` con esto**, con su linter y sus tests. Hasta entonces, los paquetes usan el formato v1 vigente (el que carga hoy).

La extensión de `Obligacion`:

```json
{
  "clase_e2e": "procedimental",
  "facetas": ["documental"],
  "temporalidad": {
    "primitiva": "periodica",
    "cadencia": "P2Y",
    "regimen": {"computo": "naturales", "cierre": "fin_de_dia", "traslado": "ninguno"},
    "disparador": {"hito_certificado": "emision_o_ultima_auditoria"}
  },
  "escalado": [
    {"tras": "P60D_antes", "a": "responsable_seguridad"},
    {"tras": "P30D_antes", "a": "responsable_servicio"}
  ]
}
```

Reglas del linter que llegan con la extensión: obligación sin `clase_e2e` no carga (clases: observable, documental, procedimental, notificatoria, remediacion); las facetas son opcionales; toda obligación con `temporalidad` exige al menos 3 casos dorados.

Los casos dorados viven en `pruebas/` dentro del directorio del paquete, un JSON por artículo:

```json
{
  "caso": "bienal desde la ultima auditoria, cierre fin de dia",
  "obligacion": "ens.art31.auditoria_ordinaria",
  "hechos": {"ultima_auditoria": "2025-03-10"},
  "esperado": {"vence": "2027-03-10T23:59:59", "computo": "naturales"},
  "cita_del_esperado": "art. 31.1: al menos cada dos anos; Rgto. 1182/71 art. 3.2.b para el cierre"
}
```

El pipeline de autoría, por artículo: (1) aislar las obligaciones, una por verbo exigible; (2) escribir el JSON con cita exacta y vigencia; (3) mínimo 3 dorados por reloj (normal, borde de calendario, modificado), derivados DEL TEXTO con su `cita_del_esperado`: si motor y dorado discrepan, gana el dorado y se arregla el motor; (4) linter y cobertura (`plazum cobertura paquetes`). Ritmo a medir con las primeras 20 obligaciones y recalibrar el plan con el número real.

La frontera legal por estrato no cambia: BOE/DOUE entero con su `identificador` declarado (ELI o CELEX; el enlace se deriva al pintar, no se guarda); ISO y similares solo identificador más título ≤120 caracteres y JAMÁS procesadas con un modelo; CIS/STIG delegados sin texto; los datos propios (demo, calendarios, equivalencias) con clase `propio` y Apache-2.0.

## Anexo C: el dialecto de aplicabilidad (construido el 25-08-2026, fuente para E3 y /autoria)

Cierra el invariante 2. El motor de Datalog estratificado existía desde la etapa 1, pero solo se podía programar desde Go, así que las reglas del ENS vivían en un fichero de test llamado `progENS`. Con las reglas en código, actualizar el corpus es una release del binario, y sin fichero de datos firmado no hay suscripción del corpus ni canal consultor. Con 2 paquetes autorizados esto era una tarde; con 12 habría sido una migración.

### Dónde va

En el bloque `aplicabilidad` de `paquete.json`, al mismo nivel que `obligaciones`:

```json
"aplicabilidad": {
  "exporta": ["categoria"],
  "reglas": [
    {
      "id": "auditoria_ordinaria_media",
      "cita": "RD 311/2022 art. 31.1",
      "regla": "aplica(\"ens.art31.auditoria_ordinaria\", S) :- en_ambito(S), categoria(S, \"MEDIA\")"
    },
    {
      "id": "nivel_maximo_de_las_dimensiones",
      "cita": "RD 311/2022 anexo I, apartado 3",
      "regla": "nivel_max(S, N) :- maneja(S, I), nivel_dimension(I, _, N)",
      "agregado": "maximo",
      "sobre": "N",
      "escala": {"nombre": "ens.niveles", "orden": ["BAJO", "MEDIO", "ALTO"]}
    }
  ]
}
```

`id` y `cita` son obligatorios y sin excepción. El `id` es lo que sale en la explicación de por qué aplica una obligación; una regla de aplicabilidad sin artículo es una opinión.

### El léxico, que son tres reglas y una trampa

| | |
|---|---|
| **Variable** | empieza por mayúscula: `S`, `E`, `Nivel` |
| **Anónima** | el guion bajo solo: `_`. Significa "no me importa el valor" |
| **Constante** | todo lo demás: en minúscula sin comillas (`x.art31.auditoria`), o entre comillas cuando lleva mayúsculas o espacios (`"MEDIA"`) |

**La trampa**, y por eso hay un guardia. Quien escriba `categoria(S, MEDIA)` sin comillas no está comparando con la constante MEDIA: está declarando una variable nueva que unifica con cualquier categoría, y la regla deriva de más en silencio. Es el mismo fallo que ya costó una revisión con la variable anónima. Por eso **una variable que aparece una sola vez en la regla es un error**: o escribes `_`, o entrecomillas la constante. El mensaje de error dice las dos salidas.

Reservadas: `not` (es la negación) y `_AGG` (es la variable interna del motor, no se escribe aquí).

### Lo que NO va en la sintaxis

El agregado, la escala y la cita van como campos del fichero de datos, al lado de la regla. Una escala es una lista ordenada, y una lista ordenada se escribe mejor en JSON que en una gramática que habría que fuzzear entera para nada.

Agregados, y no hay más a propósito porque son los tres que las normas necesitan: `maximo` (categoría ENS como máximo del nivel de cada dimensión), `cuenta` (umbrales por número de empleados, proveedores, incidentes), `existe` (basta uno). El `maximo` sobre valores no numéricos exige `escala`: sin ella el motor rechaza en vez de adivinar, porque con orden lexicográfico `MEDIO` gana a `ALTO`.

La variable que se agrega va en la CABEZA con su nombre, y `sobre` la señala. `_AGG` no asoma al dialecto: `nivel_max(S, _AGG)` no dice nada al leerlo.

### El espacio de nombres, que es lo que impide que añadir la norma 31 rompa la 12

Tres clases de predicado, y no se comportan igual:

- **Comunes** (`aplica`, `desplaza`, `equivale`): el vocabulario de salida del corpus entero. Globales siempre.
- **De entrada**: los que el paquete USA y no DEFINE (`ambito`, `maneja`, `nivel_dimension`). Son hechos que declara el sujeto al describir su alcance, y ese vocabulario es compartido por diseño: `corpus.EsquemaUI` ya funde los atributos que se llaman igual, porque un dato pedido por tres normas se pregunta una vez. Globales.
- **Locales**: los que el paquete DEFINE y no exporta (`nivel_max`, `en_ambito`, cualquier paso intermedio). Se prefijan con el paquete y no los ve nadie más.

`exporta` es el acto deliberado de publicar una derivación propia al espacio común. Es una promesa, y las tres formas de romperla son error al cargar: exportar lo que no se define, exportar un común, y exportar un nombre que ya publica otro paquete instalado.

**Regla de modelado que hay que saber antes de escribir la primera regla: un paquete no redefine un predicado que el sujeto aporta como hecho.** Si quiere cerrar transitivamente lo que el sujeto declara, deriva uno propio a partir de él:

```
proveedor_de(A, B) :- provee_a(A, B)
proveedor_de(A, C) :- provee_a(A, B), proveedor_de(B, C)
```

y no `provee_a(A, C) :- provee_a(A, B), provee_a(B, C)`, que convertiría `provee_a` en propiedad del paquete y dejaría sus reglas alimentándose de un predicado vacío. Se denuncia al evaluar, con el paquete dueño y el arreglo escritos en el error.

### Lo que el linter cruza, y que no se puede comprobar de otra forma

Que una regla que declara `aplica("x.art99", S)` apunte a una obligación que el paquete tiene. Esa errata no da error en ningún sitio: deja al sujeto sin la obligación y nadie se entera.

### Cómo se prueba una regla, y por qué el linter no basta

El linter dice que una regla se PARSEA. No dice que DERIVE lo que tiene que derivar: un paquete puede tener veintitantas reglas impecables que no disparan ninguna obligación porque el predicado del cuerpo no lo produce nadie, y el linter da verde igual.

Un paquete con reglas se prueba como `aplicabilidad_corpus_test.go` prueba el ENS: cargando el corpus de verdad, afirmando los hechos de un sujeto y comprobando **las dos direcciones**. Lo que TIENE que aplicarle y lo que NO puede aplicarle, con el artículo de cada exclusión escrito al lado. Un motor que derivara todo pasaría la primera mitad sin despeinarse.

