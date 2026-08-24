# ETAPAS.md: el plan ejecutable

Fuente única: `docs/guia.md`. Cada casilla es una puerta: se marca cuando su test corre en verde en CI, no antes. Estado global objetivo: 9,7 en las 17 dimensiones hacia el mes 24-27.

## Semana 0: fundaciones
- [x] Estructura del repo (nucleo/puertos/adaptadores/superficies/paquetes)
- [x] Núcleo construido y en verde (ventana, aplicabilidad, estado, ledger, expediente, corpus)
- [x] Tests de arquitectura: AST del núcleo, normas no cableadas, linter sobre paquetes/
- [x] CLAUDE.md, DEPENDENCIAS.md, SECURITY.md, CONTRIBUTING.md, CLA.md
- [x] CI: build, test, gofmt, vet, cobertura, gosec, govulncheck
- [ ] Descargar el texto canónico de AGPL-3.0 a LICENSE (gnu.org/licenses/agpl-3.0.txt)
- [ ] Búsqueda de anterioridades de marca + solicitud EUIPO (clases 9 y 42). OJO: existe una fintech "Obligo" y es término financiero común en alemán. Decidir nombre definitivo ANTES del primer release público
- [ ] Revisar cláusulas de PI del contrato de empleo/consultoría activo
- [ ] Activar private vulnerability reporting en GitHub

## Etapa 1 (4-6 FdS): el núcleo probatorio completo
- [ ] Ledger v2: AEAD con compromiso de clave (HMAC de la clave junto al cifrado)
- [ ] Lápidas firmadas con base legal, en el formato público del expediente
- [ ] Keystore separado: réplica propia, retención 35 días declarada, ciclo de vida de la clave maestra
- [ ] Almacén de blobs content-addressed DENTRO de SQLite, cifrado por entrada, chunking >32 MB
- [ ] Historia bitemporal (CambioEstado con instante_hecho/instante_registro) + los 10 ataques sobre historia
- [ ] Objeto Certificado con hitos (dorados: ISO trienal+vigilancias, ENS bienal+INES, SOC 2 solapadas)
- [ ] Perímetros multi-entidad con herencia y roll-up computado
- [ ] Anclaje RFC 3161 con cadena de reserva (2 TSAs + cola local) y verificación offline
- [ ] Fuzzing de parser de corpus, ledger y verificador en CI
- [ ] HITO: v0.2 firmada + post del ledger

## Etapa 2 (8-12 FdS): serve, UI generada y autoservicio
- [ ] serve con html/template + htmx vendorizado, go:embed, sesiones
- [ ] Seguridad web como puerta: CSRF en todo POST, rate limit auth/API, CSP/HSTS/XFO en CI, primer admin por token de un solo uso, guía TLS
- [ ] Las 6 pantallas (Alcance, Hoy, Controles, Certificados, Personas-esqueleto, Estado) con derivación a un clic
- [ ] UI generada desde corpus.EsquemaUI y corpus.Entrevista
- [ ] obligo demo (paquete demo-empresa), obligo doctor, obligo update con rollback
- [ ] El latido: pulso diario opt-in a obligo.dev/latido + aviso si calla 24h + smoke test del canal + estado del planificador en Hoy
- [ ] OIDC + SCIM con extensión enterprise (atributo manager) + mapeo manual alternativo
- [ ] Export del log de auditoría a SIEM (JSON líneas)
- [ ] i18n es/en con mecanismo de catálogo (de: cuando haya partner DACH)
- [ ] Litestream documentado + restore drill en CI (base + keystore + blobs, verifica cadena y lápidas)
- [ ] Matrix build Linux/macOS/Windows-Docker; descargo "no es asesoramiento jurídico" en pie y explain
- [ ] TTFV sintético en CI <15 min; axe-core cero violaciones; presupuestos (binario <25 MB, arranque <3 s, RAM <256 MB)
- [ ] HITO: v0.3 + demo alojada (efímera, reset horario, ~10 €/mes) + lista de espera con política de privacidad

## Etapa 3 (6-8 FdS): corpus, venta legal y design partners
- [ ] Paquete ISO 27001 referencial completo (id + título corto, rituales, cadencias)
- [ ] Paquete ENS transcrito completo con dorados por reloj (partir de paquetes/ens semilla)
- [ ] Equivalencias ENS↔ISO en formato OSCAL Mapping Model + la lista de huecos computada
- [ ] Revisión jurídica externa del corpus español (despacho o consultor-partner, consta en changelog)
- [ ] Política de compatibilidad N-1 escrita y con test contra artefactos de la release anterior
- [ ] Entrega del corpus: descarga HTTP firmada autenticada contra licencia Ed25519
- [ ] Venta legal: autónomo + seguro RC profesional + Stripe Payment Link + contrato con tope 12 meses
- [ ] Programa de design partners: 5 con nombre, 50% de por vida, logo + llamada de referencia
- [ ] Vigilancia normativa: 2-3 h/semana fijas desde aquí (restadas del calendario)
- [ ] Página de vigilancia pública: tabla fecha-BOE → fecha-paquete autogenerada
- [ ] HITO: v0.4 + primera venta posible + 5 consultores contactados

## Etapa 4 (6-10 FdS): continuidad, personas e incidentes
- [ ] Ingesta manual firmada (adelantada aquí: fuente de UAR y formación)
- [ ] Objeto Incidente mínimo: registro + timeline bitemporal + obligaciones notificatorias; payload de brecha AEPD
- [ ] Escalado (email + Teams) con jerarquía SCIM y colapso de niveles
- [ ] Ventanas de silencio auditadas + cambio material con diff de paquetes
- [ ] Atestación de políticas (obligación-persona anual, registro al ledger)
- [ ] Formación: tracking + quizzes SOLO de normas transcritas (ENS, RGPD, NIS2)
- [ ] On/offboarding por evento SCIM con SLAs
- [ ] UAR con snapshot firmado (fuente: import manual + SCIM; conectores en E6)
- [ ] Auditoría interna 9.2 con arrastre entre ciclos
- [ ] Acta 9.3 autogenerada + board pack (LA demo)
- [ ] Frescura de evidencia como segunda familia de relojes
- [ ] Kit mínimo de partner: acuerdo de margen 40% + demo grabada
- [ ] HITO: demo de venta (acta 9.3 + UAR + relojes, 2 min) + primer cliente del corpus + calendarios país NIS2 publicados

## Etapa 5 (6-9 FdS): la IA verificable
- [ ] Búsqueda FTS5 (BM25) siempre; embeddings opcionales vía Ollama
- [ ] Verificador de citas por hash (determinista, corre en cada PR) con adversariales
- [ ] Propuestas con revisión por trozos; botón bloqueado sin cita verificada
- [ ] Runtime de agentes: acciones tipadas, presupuesto, allowlist, transcript cifrado al ledger
- [ ] Agente 1: contradicciones; Agente 2: huecos de evidencia; Agente 3: cuestionarios entrantes
- [ ] MCP server de solo lectura con tokens de alcance; corpus como skills
- [ ] Evals: nightly + release con modelo fijado y media de N; publicados en release notes
- [ ] HITO: "el primer GRC que publica la precisión de su IA"

## Etapa 6 (5-7 FdS): conectores
- [ ] SDK WASM (Extism): ABI v1 (describe/collect/health; http_fetch con allowlist, secret_get, log)
- [ ] Suite de conformidad pública y gratuita + plantilla Go→WASM
- [ ] 4 propios: Entra ID, Google Workspace, GitHub, Intune/Jamf (sin agente propio)
- [ ] Delegados: Prowler, OpenSCAP, Trivy, ScubaGear vía OCSF
- [ ] Evidencia con procedencia y NO corroborada por defecto; corroboración exigible en críticas
- [ ] Canario diario contra cuentas sandbox reales (fuera del pipeline de PR)
- [ ] Slack + Jira como canales; MCP client por Recolección
- [ ] HITO: pilotos Cloud (máx. 5, gratuitos, acuerdo escrito, datos mínimos, horas/tenant medidas)

## Etapa 7 (4-6 FdS): riesgos y MAGERIT
- [ ] MAGERIT v3 + taxonomía ENISA como paquetes + crosswalk ENS/ISO 27005
- [ ] 3 niveles de análisis con semilla fija; aceptación caducable; tratamiento genera obligaciones
- [ ] Paquete DORA transcrito (obligaciones y relojes; el RoI validado es AÑO 2)
- [ ] Paquete AI Act (inventario, art. 4, art. 50) + plantilla SRP del CRA
- [ ] HITO: paquetes publicados con dorados

## Etapa 8 (5-7 FdS): el dinero y la confianza
- [ ] SL constituida (disparador: primer piloto Cloud o 5.000 € acumulados, lo primero)
- [ ] Checkout Stripe con Stripe Tax doméstico ES primero; licencia Ed25519 offline
- [ ] Cloud GA con el runbook de 8 piezas (bóveda secretos, OIDC/SCIM por tenant, incidentes+status, brechas, drill por tenant, baja con certificado de borrado, email transaccional, SLA de horario laboral)
- [ ] DPA con subencargados nominados
- [ ] Carpeta de compras autogenerada + portal de confianza con clickwrap
- [ ] Consola de cartera para partners v1 (solo lectura, N instancias)
- [ ] Pentest externo publicado (4-8k € del primer ingreso)
- [ ] Plan de continuidad público: segundo juego de llaves en custodia, escrow, 12 meses de fin de vida, extensión automática de suscripciones
- [ ] HITO: Cloud GA + 9,7 en camino de verificación (D14/D11 con 3 meses de medición real)

## Año 2 (apuntado, sin casillas)
Postgres, SAML, RoI DORA con subconjunto de reglas EBA + Arelle, resto del catálogo (NIST importado, ISO 22301/42001, SOC 2, PCI, TISAX referenciales, CIS/STIG delegados), consola de cartera con marca blanca, certificar el propio Cloud usando obligo, partner jurídico DACH y alemán.
