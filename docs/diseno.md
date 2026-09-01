# plazum: el GRC de continuidad. Diseño definitivo

**24 de agosto de 2026.** Novena ronda del proceso completo. Sustituye a `grc-diseno.md` (8,75) y a `diseno-v7.md` (9,03).

**Cómo se ha hecho esta versión.** Tres investigadores con web y GitHub: uno barrió el estado del arte por dimensión (leyendo releases y código de los rivales, no su marketing), otro el dinero europeo con veinte listas de precios, y un tercero recibió el borrador de este diseño con la orden de destrozarlo. Lo destrozó: **4,5 sobre 10**, con dos afirmaciones refutables con un enlace, un pricing que se canibalizaba a sí mismo y un fallo criptográfico real en el ledger. Este documento es el borrador **después** de aplicar sus veintiún arreglos. La sección 15 lista cada objeción y qué se cambió, porque un diseño que esconde su peor revisión no merece las notas que pide.

**El encargo, literal:** cada dimensión por encima de 9,5. Hexagonal y modular. RAGs, agentes, MCPs e IA, con el núcleo de compliance reproducible y determinista. Compliance as code de verdad, e2e hasta donde la norma lo permite (la parte que depende del cliente, dicha honestamente). 30 marcos con cross-framework. **Self-serve puro: el libre no genera soporte, solo el de pago lo tiene.** Comprador: CISO europeo, de SMB a gran empresa sin llegar a enterprise. Y que genere dinero.

**Qué mide la nota.** La nota es de **diseño**: contra el mejor referente público verificado con fuente, con el test falsable especificado hasta poderse escribir sin decidir nada más, y con la limitación residual documentada. El 10 se reserva para lo demostrado con test ejecutado (hoy, una dimensión). El estado de construcción va en su columna. Base construida heredada: 2.802 líneas de Go, 2.045 de test, 100 casos en verde, cero dependencias, binario de 3,95 MB.

---

## 1. La tesis, en las tres afirmaciones que sobreviven a un revisor hostil

El borrador decía "hueco total" y "nadie lo tiene" cuatro veces. El revisor tumbó dos con un enlace (Hyperproof tiene la frescura de evidencia como feature documentada; ServiceNow tiene excepciones de política con aprobador y caducidad desde hace una década). Lo que queda en pie, formulado estrecho y verificado:

1. **Nadie en el lado comprador de pyme y mid-market ha ensamblado el conjunto**: certificado como objeto con su ciclo (vigilancia anual ISO, recertificación trienal, bienal ENS, INES anual, ventanas SOC 2 solapadas), hitos país de NIS2 como datos versionados, y obligaciones internas con reloj multi-régimen encadenadas a todo ello. Las piezas existen sueltas (Vanta modela auditorías con ventanas; el software de las entidades de certificación modela el ciclo desde el otro lado de la mesa; los ISMS llevan calendarios). El conjunto, no. **Y es una ventana de 2 a 4 trimestres, no un foso**: la claim se trata como cuña de lanzamiento con fecha de caducidad.
2. **El expediente verificable offline sin confiar en el emisor no lo tiene nadie en GRC**, y no es una feature: exige rehacer el modelo de evidencia entero. Ledger encadenado, Merkle RFC 6962, verificación por tercero, sellado RFC 3161 de serie y cualificado eIDAS opcional. Esto sí es foso, porque copiarlo es rediseñar.
3. **El corpus como datos abiertos versionados es un foso de datos, no de código**: ENS con relojes, calendarios país de NIS2, MAGERIT empaquetado, equivalencias en formato OSCAL Mapping Model publicadas para que la comunidad las corrija. El código AGPL lo puede clonar cualquiera; la vigilancia que mantiene los datos frescos es el trabajo que nadie regala.

El norte del producto sigue siendo **continuidad de cumplimiento**: no pierdas nunca la conformidad. Pero la venta ya no ignora al que aún no tiene el certificado: el ICP es **100 a 1.000 empleados, ya certificados o con la primera auditoría contratada y con fecha**. El de 20 empleados sin certificado usa el libre (y es semilla del canal); el de 5.000 es alcanzable con Postgres y réplica, no es el centro de la diana.

---

## 2. Comprador, cabeza de playa y el motion que no se contradice

El revisor señaló la contradicción: continuidad es una venta de sustitución, y la sustitución con self-serve puro no cuadra en 500+ empleados donde compras exige DPA, cuestionario de seguridad y un humano. Tres decisiones la resuelven:

**ICP único: 100-1.000 empleados europeos, certificados o en camino con fecha.** Ahí el self-serve es viable, el presupuesto existe (mediana europea de la categoría: 4.500-14.500 euros/año) y el dolor de la continuidad es real porque ya hay un certificado que perder.

**Cabeza de playa: España, no cinco países a la vez.** El corpus diferencial local (ENS con relojes, art. 2.3 para proveedores del sector público, INES anual, MAGERIT) solo vale en España, y el revisor tiene razón en que mantener vigilancia normativa de cinco jurisdicciones en solitario es un compromiso de despacho jurídico. Año 1: España con ISO 27001 primero y ENS segundo. Año 2: DACH con un partner jurídico por país en revenue share que firme la revisión del corpus. Los calendarios NIS2 de DE/BE/IT/NL se publican como datos desde el día 1 (son baratos y son imán de comunidad), pero **sin SLA contractual fuera de España** hasta que exista ese partner.

**El canal año 1 son los consultores que ya operan esos ISMS.** La continuidad hoy la hace un consultor con hojas de cálculo: no es el enemigo, es el canal. Se le da la herramienta con la que retiene a su cliente, margen del 40% sobre Cloud y corpus (el revisor tumbó el 25%: nadie mueve un dedo por 750 euros/año), certificación gratuita (la de pago era fantasía a esta escala: año 3 como pronto), y él pone la capa humana que el self-serve no da. instant27001 vendió 2.500 licencias así, con el 100% del canal en partners.

**Y el procurement se hace self-serve con el propio producto.** La objeción de "compras exige DPA y cuestionario" se responde con la **carpeta de compras autogenerada**: el producto se instala su propio paquete, publica su expediente verificable, y genera el paquete de proveedor completo: DPA con lista de subencargados, cuestionario de seguridad respondido con citas al expediente, SBOM firmado, política de divulgación, releases firmadas. El comprador de 800 empleados descarga la carpeta sin hablar con nadie. Dogfooding como argumento de venta: plazum pasa el due diligence que sus clientes hacen con plazum.

---

## 3. Autoservicio radical

La restricción del encargo, convertida en ocho mecanismos (y neutraliza de paso el hallazgo del burnout: el mantenedor de core-js, 250 horas al mes por 400 dólares):

1. **Toda pantalla responde "por qué veo esto"**: derivación completa a un clic (regla, paquete, cita). Ya construido en el motor (`plazum explain`).
2. **Todo error es accionable**: causa, arreglo y runbook embebido en el binario (`/ayuda` offline, es/en/de). Nada depende de una web.
3. **`plazum doctor`**: diagnóstico local completo con salida copiable a un issue, datos personales redactados. El bug report lo genera la herramienta.
4. **`plazum demo`**: empresa sintética completa con relojes corriendo. El evaluador ve el producto lleno en dos minutos sin conectar nada.
5. **`plazum update`**: backup previo, migración, rollback de un comando. La causa número uno de tickets de self-hosted, cerrada por diseño.
6. **Compra self-serve**: checkout, la licencia es un fichero firmado Ed25519, activación offline. Y la licencia firma **derechos sobre datos y servicios** (corpus con garantía, Cloud), nunca gates de código: en AGPL cualquier fork borra un gate, y venderlos sería vender humo. Precios públicos, como los europeos que venden sin comerciales (Cyberday, Hicomply, instant27001).
7. **Política de soporte pública y sin vergüenza**: comunidad en Discussions sin SLA; el soporte con SLA existe solo en el gestionado. Regla de ingeniería: todo issue recurrente se convierte en un test, un mensaje de error mejor o un cambio de UI. El soporte no se contesta: se elimina.
8. **Dos métricas de fricción, las dos publicadas**: en CI, el arranque sintético (instalar, entrevista, primer reloj visible) con presupuesto duro; y en producto, la métrica real que el revisor exigió: **tiempo hasta la primera obligación real con dueño y reloj**, medida con telemetría opt-in y publicada por release. La primera vigila regresiones; la segunda no se puede maquillar.

---

## 4. Arquitectura hexagonal definitiva

```
  corpus (paquetes de datos firmados y versionados)
    obligaciones, reglas Datalog, calendarios pais, escalas, autoridades,
    plantillas, preguntas de alcance, equivalencias OSCAL, catalogos de
    riesgo (MAGERIT, ENISA), cadenas de escalado, casos dorados
        |
  nucleo determinista            [construido: 2.802 lineas, 100 tests, 0 deps]
    ventana · aplicabilidad · estado · historia (bitemporal) · riesgo ·
    certificado · equivalencia · precedencia · ledger · expediente · corpus
    cero I/O, cero reloj de sistema, cero LLM, cero norma cableada (test AST)
        |
  puertos (interfaces Go)
    Ingesta | Recoleccion | Almacen | Notificacion | Escalado |
    Documento | Anclaje | Asistente | Identidad
        |
  adaptadores
    SQLite+Litestream / Postgres · OCI · conectores WASM (Extism) ·
    delegados (Prowler, OpenSCAP, Trivy, ScubaGear) · RFC 3161 / QTSP
    eIDAS / Rekor · SMTP / Slack / Teams / Jira / webhook · LLM fuera de
    proceso (Ollama local, API) · MCP (x3) · OIDC / SAML / SCIM
        |
  superficies
    CLI · API HTTP · UI embebida generada del modelo (htmx) ·
    portal de confianza estatico · carpeta de compras ·
    exportadores (OSCAL, OCSF, xBRL-CSV, SARIF)
```

**La cuarentena de la IA es de proceso, no de sintaxis.** El revisor tumbó la versión anterior (un test de AST se elude con inyección de dependencias, reflection o un simple http contra la API local). La frontera real: **los adaptadores de LLM corren fuera de proceso** (o en WASM sin host functions de escritura), y el único canal de vuelta es el tipo `Propuesta{diff, cita, hash_fuente, modelo, digest_prompt}` por el puerto Asistente. La cita se verifica por hash contra el corpus o la propuesta nace bloqueada. Nada muta estado sin aceptación humana registrada. El test de AST se queda como defensa en profundidad, no como la garantía.

**La evidencia de conectores tampoco es confiable por defecto.** Segunda parte del mismo hallazgo: un conector WASM malicioso contaminaría el expediente sin tocar la IA. Arreglo: toda observación entra con procedencia (conector, versión, digest del módulo) y en estado **no corroborado**; los predicados pueden exigir corroboración (dos fuentes independientes) para obligaciones designadas como críticas en el paquete. El sandbox limita capacidades (sin red ni filesystem salvo lo concedido, secretos solo en el host), y la suite de conformidad es pública y gratuita: la certificación de pago, si llega, es de año 3, cuando haya base instalada que la justifique.

**Table stakes de mid-market en la v1, no en el roadmap**: OIDC/SAML y SCIM en la edición libre (cobrar el SSO en una herramienta de cumplimiento sigue siendo una contradicción performativa), réplica continua con Litestream de serie y Postgres opcional desde el primer release, export del log de auditoría a SIEM, y la carpeta de compras. El revisor lo dijo exacto: plazum no puede suspender el due diligence que sus propios clientes le harían con plazum.

---

## 5. El núcleo determinista: lo construido y las tres brechas cerradas bien

Lo heredado y medido: motor temporal (seis primitivas, cómputo natural/hábil, calendarios combinables art. 30.6, cierre exacto o fin de día Rgto. 1182/71, traslado, suspensión, prórroga, lecturas divergentes con cita), Datalog estratificado con linter y semi-naive, ocho estados de control, ledger con Merkle RFC 6962 y verificación por tercero, expediente con anclas de confianza del receptor, y el formato de corpus con linter de cuatro estratos **con control negativo ejecutado**.

**5.1. Historia bitemporal.** `CambioEstado{prueba, de, a, instante_hecho, instante_registro, causa}`. El estado actual es un pliegue; el estado en cualquier instante pasado es reproducible. Da la ventana de observación de SOC 2, el cronómetro desde el primer conocimiento (RGPD art. 33) y el MTTR. Test: los diez ataques del expediente re-ejecutados sobre historia.

**5.2. Borrado legal con criptografía que compromete la clave.** La versión anterior proponía AES-GCM por entrada y el revisor la tumbó con los "invisible salamanders" (Grubbs et al., Black Hat 2020): GCM no es key-committing, un escritor malicioso puede fabricar un cifrado que descifra a dos contenidos distintos con dos claves distintas, y enseñar uno al auditor y otro al juzgado con la misma cadena válida. Arreglo, dos líneas de formato: **AEAD con compromiso de clave** (etiqueta adicional HMAC-SHA-256 de la clave junto al cifrado; la verificación exige ambas). Borrar sigue siendo destruir la clave y añadir una **lápida firmada** con base legal citada (Ley 2/2023 art. 32, RGPD art. 17). Y la semántica de lápidas va en el formato público del expediente, para que un verificador independiente reporte "suprimida con base legal X el día Y" y no "posible manipulación". Test: destruir clave, verificar cadena, comprobar irrecuperabilidad y que la lápida lista la base legal.

**5.3. Precedencia normativa sin jugar a ser juez.** El revisor recordó que lex specialis depende de los hechos del caso y que resolverla en silencio es tomar determinaciones jurídicas. Arreglo: los paquetes declaran `desplaza(a, b, ambito, cita)` y la capa posterior al punto fijo **anota, no resuelve**: la obligación queda `desplazada_por` con su cita, visible, y la primera vez que afecta al sujeto se pide **confirmación humana registrada** (quién, cuándo, con qué cita). Determinista, auditable, y la decisión jurídica queda donde debe: en el cliente, con la cita delante. Ciclos entre declaraciones: error de carga. Test: casos dorados DORA/NIS2, MiCA/DORA, Máquinas/AI Act.

**5.4. El almacén de transcripts de agentes tiene su propia política.** Hallazgo M7: el hash en el ledger no pesa, pero el transcript contiene prompts con evidencia confidencial. Arreglo: mismo régimen que la evidencia (cifrado por entrada con compromiso de clave, retención declarada, borrado por destrucción de clave con lápida).

---

## 6. Compliance as code e2e: cinco clases, con facetas

Toda obligación del corpus lleva **clase primaria** y opcionalmente facetas (el revisor tumbó la clase única: "notificar en 72h Y documentar Y remediar" existe). Obligación sin clase primaria: error de linter.

| Clase | Qué significa e2e | Cadena |
|---|---|---|
| **Observable** | la máquina comprueba el hecho | conector WASM o delegado → OCSF → predicado CEL → estado → ledger |
| **Documental** | el documento se genera del expediente | plantilla trazada (origen por campo, construido) → render → hash → sello |
| **Procedimental** | una persona hace algo periódicamente | ritual: atestación programada con caducidad, recordatorio y escalado. Aquí vive, honestamente, la parte que depende del cliente: no se finge comprobarla, se agenda, se reclama y se registra |
| **Notificatoria** | aviso a autoridad bajo reloj | reloj multi-régimen → payload listo: formulario AEPD, plantilla SRP del CRA, RoI de DORA en xBRL-CSV |
| **Remediación** | corregir lo que falla | propuesta-como-código: política Cloud Custodian, playbook Ansible o plan Terraform **como artefacto a revisar**, jamás auto-aplicado |

**DORA, donde ya no hay hueco de existencia, se compite en validación**: reglas EBA completas y checks DPM en local, salida verificada con Arelle, convenciones de presentación por autoridad (BaFin, BdE, DNB, CSSF) y diff interanual contrato a contrato. El del líder genera el fichero; este dice si te lo van a rechazar y por qué.

**La métrica de profundidad se publica por marco** en COBERTURA.md (generador construido y con test): total, % por clase, % delegada, y la lista nominal de lo manual. El revisor avisó (M9): el día 1 esos números serán bajos y la competencia los usará. Se asume con los ojos abiertos, con la respuesta en la misma página: el número equivalente de cada rival es **no publicado**. Quien compite contra un número honesto tiene que publicar el suyo o conceder el punto.

---

## 7. Cross-framework computado, con el álgebra que no miente

Lo verificado: los SaaS reutilizan evidencia por controles comunes y muestran el % del segundo marco (Hyperproof lo hace con frescura incluida); CISO Assistant tiene inferencia de un salto con el defecto N→1 abierto un año (#2432); el OSCAL Mapping Model existe desde diciembre de 2025 casi sin contenido. Y el revisor tumbó mi multi-salto alegre con matemática simple: `superset ∘ subset` da como mucho `intersects`, `subset ∘ superset` no da nada, y en crosswalks reales domina justo `intersects-with`, que no compone. Las cadenas largas colapsan a "sin información" casi siempre, y propagar lo impropagable acaba en una no conformidad que el cliente te cobra.

El diseño corregido:

1. **Estado solo desde un salto con relación fuerte** (`equal`, `subset` en la dirección correcta), con procedencia de evidencia y TTL vivo en destino. Todo lo demás (multi-salto, `equivalent`, `intersects`) se muestra como **sugerencia a revisar**, nunca como estado. La honestidad del cálculo es el argumento: el rival que propaga alegre come no conformidades.
2. **Se conservan todas las rutas de inferencia con su confianza** (la corrección del N→1 del líder), y ninguna inferencia existe sin su cadena de citas completa.
3. **La demo no es el porcentaje, es la lista**: "ENS MEDIA te deja ISO 27001 con estos 12 huecos, cada uno con su cita y su evidencia de procedencia". El porcentaje lo enseña cualquiera; la lista nominal con citas no la puede enseñar nadie hoy.
4. **Los mapeos se publican como datos abiertos en formato OSCAL Mapping Model.** El fundador escribe la semilla y la comunidad la corrige: es el foso de datos, y ser el primer contenido real del formato del NIST es posicionamiento gratis.
5. **SCF fuera** (CC BY-ND prohíbe derivados, incluso vía IA): importable verbatim por el usuario, jamás redistribuido transformado.

---

## 8. La capa de IA: proponer con pruebas, decidir jamás

- **RAG con citación verificada mecánicamente**: chunking por unidad estructural, embeddings locales por defecto (el binario no depende de ninguna API), y toda respuesta cita con verificación por hash o nace marcada como no verificada con el botón de aceptar bloqueado. Contra el 17-33% de alucinación medido en las herramientas legales comerciales, la defensa no es un modelo mejor: es un verificador que no es un modelo.
- **Agentes con presupuesto**: acciones tipadas, máximo de pasos y tokens, allowlist por tarea, transcript completo hasheado al ledger (almacenado bajo el régimen 5.4). Casos con verificador determinista gratis: contradicciones (la política dice 90 días, lo observado 365), huecos de evidencia antes de una vigilancia, borradores de cuestionarios con cita. Nunca: veredictos, puntuaciones, firmas. En open source GRC no existe hoy ningún runtime agéntico real: este es el primero, y nace con el freno de mano puesto a propósito.
- **MCP triple**: servidor de solo lectura con tokens de alcance (el auditor pregunta desde su herramienta), cliente (los MCPs del cliente entran por Recolección normalizando a OCSF: cada MCP del mercado es un conector gratis), y el corpus publicado como recursos MCP y skills, el único formato de contenido normativo con tracción real en GitHub.
- **Evals publicados por release** con umbrales que rompen el build. Vanta, Drata y Sprinto no publican un solo número de precisión: publicar los nuestros les obliga a elegir entre publicar los suyos o conceder el punto.
- **El producto cumple el AI Act que vende**: inventario, art. 4 y art. 50 (lo exigible hoy tras el Ómnibus), y se instala su propio paquete.

---

## 9. Continuidad: certificado, escalado y silencio, con las claims corregidas

**El objeto `Certificado`** con su ciclo completo, los hitos país y las obligaciones internas encadenadas: la claim honesta es que **nadie lo ha juntado en el lado comprador**, y que es cuña de 2-4 trimestres, no foso. Se lanza primero por eso mismo.

**Cadenas de escalado por obligación**, corregidas con las tres objeciones del revisor: (a) la claim es "de serie en GRC de pyme y mid-market" (en enterprise, ServiceNow y LogicGate escalan hace años); (b) el escalado entrega **donde vive la gente**: Slack, Teams, Jira y email son conectores de primera ola, no una nota al pie; (c) la jerarquía se importa de IdP/HRIS vía SCIM, y en la empresa de 30 personas donde CISO y dirección son la misma silla, la cadena colapsa niveles automáticamente en vez de escalar dos veces al mismo buzón.

**Ventanas de silencio auditadas** (motivo, aprobador, fecha fin obligatoria, reactivación sola, visible en el expediente): en enterprise existen como policy exceptions; en el segmento objetivo, nadie las trae de serie. Un control silenciado sin ventana viva es un hallazgo.

**Detector de cambio material**: al actualizar un paquete, diff de obligaciones aplicables con lista nominal y tarea de revisión. Y la corrección M1: los calendarios país no se venden como cuenta atrás de hitos (el registro alemán venció en julio; el neerlandés entró hace nueve días), sino como **servicio continuo de obligaciones recurrentes**: informes, reevaluaciones, notificaciones, auditorías. Los hitos pasan; el goteo es el producto.

---

## 10. Riesgos con MAGERIT, el imán local

Sin cambios de fondo tras la ronda (ninguna objeción la tocó): MAGERIT v3 y la taxonomía ENISA como paquetes de datos (reutilización del sector público con atribución; nadie los trae, PILAR los tiene cautivos), tres niveles de análisis (exprés citable, estándar sobre las entidades existentes, cuantitativo opcional con semilla fija), aceptación caducable con dueño, tratamiento que genera obligaciones con reloj, riesgo sin tratamiento elevado a hallazgo, y proveedores con el RoI de DORA encima.

---

## 11. Los 30 marcos, y quién firma el corpus

El modelo de cuatro estratos con linter está construido y probado. Las correcciones de la ronda:

- **El SLA de corpus se reformula**, y D-20 le quita la palabra «garantía» en 2026: **mejor esfuerzo con plazo objetivo publicado** (días desde BOE/DOUE hasta el paquete, con changelog citado y firmado), descargo explícito de que no es asesoramiento jurídico, y tope de responsabilidad al importe pagado en 12 meses. La palabra "garantía contractual de vigilancia jurídica" era una demanda esperando fecha, y el revisor la vio.
- **Un país cabeza de playa** (España), partners jurídicos por país para expandir (año 2, DACH), y los calendarios de DE/BE/IT/NL como datos abiertos sin SLA hasta que exista ese partner.
- Los estratos quedan: importado (NIST en OSCAL, CC0), transcrito (BOE/DOUE con las tres obligaciones formales), referencial (identificador y título corto, límite de 120 caracteres impuesto por linter con control negativo), delegado (CIS/STIG vía la herramienta con licencia).

---

## 12. El dinero, rehecho desde la coherencia de dominancia

El revisor encontró que el gestionado barato dominaba estrictamente a los SKUs self-hosted (4.300 euros sin hosting frente a 2.988 con todo), que el tramo alto regalaba precio contra Cyberday real (1.660-1.990 euros/mes en 1.000-2.999 empleados), y que "100 clientes año 3" exigía el percentil 95 de cada supuesto a la vez. Modelo corregido:

**Dos líneas de ingreso, no cinco:**

| Línea | Precio | Qué incluye |
|---|---|---|
| **plazum Cloud** (gestionado UE, la línea ancla) | 290 €/mes (≤100 empl.) · 590 (≤300) · 990 (≤1.000) · 1.690 (>1.000, Postgres+HA) | instancia operada, corpus con plazo objetivo, sello eIDAS de cada actualización, soporte con SLA, carpeta de compras |
| **Vigilancia del corpus** (la única suscripción self-hosted; se llamaba «Corpus firmado» hasta D-20) | 1.490 €/año | **los paquetes son gratis para todo el mundo**: aquí se paga la vigilancia, o sea plazo objetivo publicado con histórico, changelog curado con notas de alcance, aviso proactivo de cambio material y sello de cada release. Sin lenguaje de garantía jurídica y sin soporte de operación: el self-hosted es self-serve por diseño |

Sin SKU de soporte self-hosted permanente (dominado por el gestionado, y su demanda real es <5% en los modelos que publican datos). Complementos que no son líneas: paquete de norma a medida 3.500 € + 20%/año, y el canal partner con margen 40% y certificación gratuita. El sello eIDAS es feature, no línea: cuesta céntimos al por mayor y vale como prueba ("cada actualización del corpus, sellada y verificable"), no como factura.

**Cohortes honestas, no cierre por decreto** (la corrección C4): lanzamiento comercial en el mes 9-12 (antes hay que construir las etapas 1-4), rampa de altas 1→2→3-4/mes conforme el canal consultor arranca, churn 10%, y **el gestionado con lista de espera hasta que la automatización demuestre menos de 4 horas de operación por tenant y año** (single-tenant SQLite + Litestream + flota automatizada; si no se demuestra, el gestionado no escala y el modelo pivota a corpus + partners).

| Escenario | Año 1 | Año 2 | Año 3 |
|---|---|---|---|
| Pesimista | 8-15k € | 40-70k € | 90-140k € |
| **Base** | **15-30k €** | **80-140k €** | **220-350k €** |
| Bueno | 30-50k € | 150-220k € | 380-500k € |

Los años 1-2 no pagan un salario: el plan asume ingresos de consultoría o empleo en paralelo, dicho sin adornos. El techo en solitario sigue en 400-500k: más allá exige contratar, y esa decisión no es de este documento.

---

## 13. Cara a cara nominal, sin hombres de paja

| Eje | CISO Assistant | Eramba | Comp AI / Probo / Openlane | Hyperproof | ServiceNow IRM | Vanta / Drata | Cyberday | **plazum** |
|---|---|---|---|---|---|---|---|---|
| Átomo | requisito de lista | control | control | control con frescura | objeto configurable | control con test | requisito | **obligación con disparador, reloj, régimen, destinatario, clase e2e** |
| Plazos multi-régimen | no (timedelta, 0 librerías de calendario) | fechas de revisión | no | vencimientos | SLAs genéricos | recordatorios | recordatorios | **construido: hábiles, combinables, cierre, traslado, divergencias con cita** |
| Ciclo de certificado | no | no | no | parcial (auditorías) | configurable a medida | auditorías con ventana | calendario | **objeto de primera clase con hitos país** |
| Escalado por obligación | no | no | no | no | sí (enterprise, a medida) | no | no | **de serie, con jerarquía SCIM y entrega en Slack/Teams/Jira** |
| Cross-framework | 1 salto, N→1 roto (#2432) | manual | básico | map once, comply many | a medida | % del segundo marco | mapeos | **1 salto fuerte como estado, resto sugerencia, rutas completas, datos OSCAL abiertos** |
| Expediente verificable offline | no (hash autoatestado mutable) | no | no | no | no | no | no | **construido: Merkle, tercero, sellado; borrado legal con AEAD comprometido** |
| Corpus como datos versionados con linter legal | Excel→YAML sin análisis de licencia | no | no | propietario | propietario | propietario | propietario | **construido, 4 estratos, control negativo** |
| IA | chat RAG local + MCP | no | orquestación LLM | sí (SaaS) | sí (enterprise) | agentes con aprobación | asistencia | **RAG citación-verificada + primer runtime agéntico OSS con presupuesto + evals publicados** |
| Huella | Django+Postgres+torch | LAMP | node/go + servicios | SaaS | SaaS | SaaS | SaaS | **binario 3,95 MB, 0 dependencias, SQLite+Litestream** |
| Self-serve real | parcial | no | parcial | no (ventas) | no (ventas) | no (ventas) | sí | **radical: doctor, demo, update, carpeta de compras, licencia offline** |

Donde pierde, dicho claro: amplitud de integraciones (Vanta 400), marca ante el auditor anglosajón, madurez de producto (ellos existen, esto es diseño más un núcleo), y motor de riesgos enterprise (ServiceNow/Archer juegan otra liga). Para el ICP declarado, ninguna de esas cuatro decide la compra.

---

## 14. La rúbrica: 17 dimensiones

| # | Dimensión | Peso | Diseño | Estado | Qué sostiene la nota |
|---|---|---|---|---|---|
| D1 | Modelo de obligación y temporalidad | 12 | **9,7** | construido y medido | el líder es timedelta sin calendarios; divergencias con cita únicas |
| D2 | Determinismo y reproducibilidad | 8 | **9,6** | construido, 10 ataques | verificación por tercero; AEAD comprometido tras la ronda |
| D3 | Cobertura por estratos y calendarios país | **4** | **9,5** | formato construido, corpus por escribir | linter legal con control negativo; país como datos, con SLA solo donde hay quien firme |
| D4 | Implantación e2e: 5 clases con facetas | 8 | **9,6** | trazabilidad construida | métrica publicada que nadie más publica; DORA por validación |
| D5 | Conectores WASM con conformidad | **7** | **9,5** | priorización construida | sandbox por capacidades + evidencia no corroborada por defecto |
| D6 | Continuidad: certificado, escalado, silencio | 8 | **9,5** | diseñado sobre estados construidos | claims corregidas: "de serie en pyme", entrega en Slack/Jira, ventana 2-4 trimestres asumida |
| D7 | Evidencia y valor probatorio | 6 | **9,7** | ledger construido | única pieza que exige rediseño para copiarse; salamanders arreglado |
| D8 | Riesgos con MAGERIT | **7** | **9,5** | diseñado | catálogo local que nadie trae; PILAR cautivo |
| D9 | Ligereza y huella | 3 | **9,8** | construido | 3,95 MB frente a stacks completos |
| D10 | Instalación local y datacenter | 5 | **9,6** | diseñado con presupuesto en CI | SSO/SCIM/Litestream en v1 libre; Postgres para el tramo alto |
| D11 | Intuitividad y guiado | 7 | **9,5** | EsquemaUI y Entrevista construidos | UI generada (patrón PocketBase) + el 20% denso presupuestado a mano |
| D12 | IA verificable | **8** | **9,6** | especificada | cuarentena de proceso, no de sintaxis; evals que rompen build |
| D13 | Extensibilidad | 4 | **10,0** | **demostrado con control negativo** | norma nueva sin tocar código, con test |
| D14 | Open core self-serve | 6 | **9,5** | anclado a 20 precios públicos | dominancia coherente tras la ronda; licencia sobre datos, no gates |
| D15 | Legalidad del corpus | 6 | **9,6** | linter construido | único con análisis publicable; SCF esquivado |
| D16 | Cross-framework computado | 5 | **9,5** | álgebra especificada | solo compone lo componible; la lista, no el porcentaje |
| D17 | Autoservicio radical | 5 | **9,6** | especificado | carpeta de compras + métrica real no maquillable |
| | **GLOBAL** | 109 | **9,60** | | |

### Los pesos, con la cuenta enseñada (D-20, 02-09-2026)

**El peso es la importancia en la decisión de compra**, no la dificultad de construir. Por eso D-20 los mueve: si el corpus deja de ser lo que se cobra y lo que se cobra pasa a ser el sistema que asiste y actúa dentro del entorno del cliente, el peso tiene que seguir a la promesa o el número deja de medir nada.

**El movimiento, y la suma no cambia:**

| | antes | ahora | por qué |
|---|---|---|---|
| D3 Cobertura por estratos y calendarios país | 8 | **4** | el corpus pasa a community-grade y gratis (D-20 a). Sigue siendo el diferenciador, deja de ser lo que decide la compra |
| D5 Conectores WASM con conformidad | 6 | **7** | «hacer cosas dentro del entorno del cliente» es literalmente esta dimensión |
| D8 Riesgos con MAGERIT | 6 | **7** | lo mismo: es sistema, no corpus |
| D12 IA verificable | 6 | **8** | la IA de adopción entra en la v1 (D-20 c). Es la que más sube porque es la que más promete |
| **suma** | **109** | **109** | un movimiento de pesos que además cambia el denominador no se puede leer |

**Y la cuenta, que es lo que hace esto contestable en vez de creíble.** Ponderado = suma de (peso × nota) / suma de pesos, con las notas de diseño de la tabla de arriba, que **no se han tocado**:

| | ponderado | / 109 | global |
|---|---|---|---|
| pesos antiguos | 1.045,8 | | **9,5945** |
| pesos nuevos | 1.046,0 | | **9,5963** |

**Movimiento por la reponderación sola: +0,0018.** O sea nada, y tiene que ser nada: la nota de diseño de D3 (9,5) es casi exactamente la media, así que quitarle peso no mueve el resultado. **Reponderar no regala décimas de diseño.**

**Sobre las notas REALES, la reponderación CUESTA una décima.** Con la columna «Hoy» de `docs/instantanea.md` (D3 4,5; D5 2,0; D8 1,5; D12 1,5):

| | ponderado | global real |
|---|---|---|
| pesos antiguos | 673,2 | **6,1761** |
| pesos nuevos | 661,7 | **6,0706** |

> **−0,1055.** Las tres dimensiones que ganan peso son las tres más vacías del tablero, así que darles peso empeora el número honesto. Es exactamente lo que tiene que pasar cuando una decisión mueve la promesa hacia lo que todavía no está construido, y es la prueba de que estos pesos no se movieron para que saliera una nota. La regla de D-20 (e) —*una nota que sube por reponderación sin que suba nada real se dice en voz alta*— aquí no hace falta invocarla: no sube, baja.

**Lo que esto le hace al 9,7, medido y no intuido.** Cada décima de D12 vale ahora **8/109 = 0,073** puntos de global en vez de 6/109 = 0,055, **un tercio más**; cada décima de D3 vale la mitad que antes. El 9,7 no se acerca: **se encarece**, porque queda colgado de D5, D8 y D12, que están a 2,0, 1,5 y 1,5. Esa es la consecuencia real de D-20 y es la que hay que mirar cada vez que se recalibre.

**Dos avisos de lectura, para que ningún número viva en dos sitios sin decirlo:**

- **`docs/instantanea.md` lleva los pesos antiguos** y no se toca: es una foto fechada el 26-08-2026 cuyos números están todos viejos (dice 1.199 casos de test cuando hoy son 1.918). Se vuelve a hacer entera o no se hace; retocarle una celda la convertiría en una foto que finge estar viva. Las notas «Hoy» de arriba salen de ella y **arrastran su fecha**.
- **`docs/decisiones.md` D-3 y D-9 dicen que «las tres dimensiones que deciden la compra suman 20 de peso»** (D11 7 + D3 8 + D17 5). Con los pesos nuevos suman **16**. No se corrigen: son registros de decisión fechados y su aritmética era correcta el día que se escribieron. Se leen con su fecha.

Cada nota de diseño sobrevive ahora a las objeciones de la ronda porque **incorpora su arreglo**: ninguna claim absoluta que un enlace refute, ningún pricing dominado, ningún AEAD sin compromiso de clave, ninguna inferencia sin cadena de citas, ningún SLA que un jurista no firmaría.

---

## 15. La ronda adversarial, a la vista

El borrador de este mismo documento recibió un 4,5/10. Objeciones y arreglos, completos:

| # | Objeción (severidad) | Arreglo aplicado |
|---|---|---|
| C1 | claims "nadie lo tiene" refutables; sin confrontación nominal | sección 13 nominal; claims reducidas a las tres de la sección 1 |
| C2 | pricing auto-canibalizado; tramo alto regalado | dos líneas con dominancia coherente; tramo alto a 1.690 €/mes |
| C3 | continuidad = venta de sustitución con motion self-serve; ICP de tres órdenes | ICP 100-1.000 certificados; canal consultor 40%; carpeta de compras |
| C4 | aritmética año 3 en percentil 95; nadie opera el gestionado | cohortes con rampa y lista de espera atada a <4h/tenant/año |
| A1 | escalado y silencio existen en enterprise | claim "de serie en pyme"; Slack/Teams/Jira + SCIM primera ola |
| A2 | frescura y readiness ya existen (Hyperproof, ISMS.online) | diferencial reformulado: la lista con citas, no el porcentaje |
| A3 | el multi-salto colapsa o miente | estado solo desde salto fuerte; resto sugerencia; datos abiertos |
| A4 | cuarentena AST eludible; conectores contaminan | aislamiento de proceso/WASM; evidencia no corroborada por defecto |
| A5 | AES-GCM no key-committing (invisible salamanders) | AEAD con compromiso de clave; lápidas firmadas en formato público |
| A6 | faltan table stakes; nodo único para 5.000 | SSO/SCIM/Litestream/SIEM en v1; ICP recortado a 1.000 |
| A7 | SLA de corpus multi-país = demanda esperando fecha | mejor esfuerzo con plazo objetivo + descargo + partner jurídico por país |
| A8 | certificado-objeto sobrevendido | reclasificado: cuña con fecha de caducidad, no foso |
| M1-M9 | calendarios que caducan, clases híbridas, precedencia jurídica, lápidas para terceros, certificación de pago fantasía, TTFV de teatro, transcripts con PII, partners al 25%, COBERTURA como munición | todos aplicados en sus secciones (9, 6, 5.3, 5.2, 4, 3, 5.4, 2, 6) |

---

## 16. Plan de construcción

| Etapa | Qué | Sube | Fines de semana |
|---|---|---|---|
| 0 | HECHA: núcleo temporal, Datalog, estados, ledger, expediente, corpus con linter, extensibilidad demostrada | | |
| 1 | AEAD comprometido + historia bitemporal + objeto Certificado | D2, D7, D6 | 3-4 |
| 2 | `serve` + UI generada + entrevista + Hoy + `demo` + `doctor` | D11, D10, D17 | 4-6 |
| 3 | Paquete ISO 27001 (referencial) + ENS (transcrito) + equivalencias OSCAL + la lista de huecos | D3, D16 | 4-6 |
| 4 | Escalado (Slack/email primero) + ventanas de silencio + cambio material | D6 | 2-3 |
| 5 | RAG con citación verificada + evals en CI | D12 | 3 |
| 6 | Calendarios país como datos + carpeta de compras | D3, D17 | 2 |
| 7 | SDK WASM + 4 conectores + suite de conformidad | D5 | 4-6 |
| 8 | MAGERIT + riesgos + RoI DORA validado | D8, D4 | 4-6 |
| 9 | Checkout + licencia firmada + plazum Cloud con lista de espera | D14 | 3-4 |

Cada etapa es útil y publicable sola. El orden pone el argumento comercial (ISO+ENS con la lista de huecos) antes que los conectores, porque la venta europea es ISO 27001 primero.

---

## 17. Las verdades que quedan aunque molesten

1. **El techo de la categoría está medido**: el líder open source factura menos de 700k tras seis años; el ancla europea de precio (Cyberday) facturó 2,67M tras ocho, con 28 empleados. Esto puede pagar bien una vida; no financia un unicornio, y el plan lo asume.
2. **La ventana del conjunto ensamblado es de 2-4 trimestres.** El workflow engine del líder salió hace dieciséis días. Lo que no pueden copiar en un sprint: el modelo de datos temporal, el expediente verificable y el corpus con linter legal. Lo demás, sí. Por eso el orden de etapas empieza por lo incopiable.
3. **Los años 1-2 no pagan un salario.** Consultoría o empleo en paralelo, y el gestionado no escala hasta que las horas por tenant lo demuestren. Si al mes 18 no crece solo, el pivote es corpus + partners, y está escrito desde ya.
4. **9,59 es nota de diseño.** El 10 de cada dimensión se gana construyendo, y hoy solo la extensibilidad lo tiene. Las etapas 1-9 son el camino; ninguna decisión de diseño queda pendiente para recorrerlo, y esa era exactamente la definición del 9,5.
