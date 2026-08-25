# Censo de relojes de los 30 marcos

Fecha del censo: 26-08-2026. Este documento **cuenta**, no transcribe. Lo que hay
son números de obligación con reloj y el número de artículo que respalda cada
número. Las pocas expresiones entrecomilladas son marcadores de búsqueda de dos o
tres palabras, siempre de textos de BOE o DOUE, que son reutilizables citando la
fuente. De ISO, PCI DSS, SOC 2, TISAX y CIS no aparece ni una palabra, y por eso
sus filas de la tabla están vacías.

Sirve para una sola cosa: **decidir el orden de autoría del corpus**. La
conclusión operativa está al final, en "Orden de autoría propuesto", y no está
ordenada por marco sino por familia de reloj, porque la unidad de trabajo real
no es el marco, es la primitiva temporal.

## 1. Qué se cuenta y qué no

**Unidad de cuenta**: una obligación con reloj, identificada por el par
(artículo, apartado), **cuyo destinatario es la organización obligada**. Se
descartan de la cuenta las obligaciones dirigidas a la Comisión, a los Estados
miembros, a las autoridades competentes, a las AES, a ENISA, a los CSIRT, a los
organismos notificados y a los tribunales. Son la mayoría de los plazos de un
reglamento europeo y contarlas infla el censo sin darle una sola obligación al
cliente. En los marcos grandes la sección correspondiente dice cuántas líneas
candidatas había antes de ese filtro, para que se vea el tamaño del descarte.

Las tres columnas, con la definición exacta que se ha aplicado:

1. **Plazo**: fecha límite o duración computable desde un hecho. "72 horas desde
   que se tenga constancia", "en el plazo de un mes", "durante diez años".
2. **Periodicidad**: cadencia declarada. Se distingue siempre **cadencia con
   número** ("al menos cada dos años", "anualmente") de **cadencia sin
   cuantificar** ("periódicamente", "a intervalos planificados"). Las dos cuentan
   como periodicidad, pero solo la primera es computable por el motor sin que el
   cliente ponga un número.
3. **Evento disparador**: la obligación nace cuando ocurre un hecho. Brecha,
   incidente, cambio sustancial, entrada en un umbral, solicitud del interesado.

Hay una **cuarta categoría que aparece en casi todos los marcos y que no es
ninguna de las tres**: el plazo abierto, "sin demora indebida", "sin dilación
indebida", "inmediatamente". No es computable y no se cuenta como plazo, pero se
señala donde es relevante porque el motor va a necesitar decidir qué hace con
ella y porque el cliente la va a preguntar.

## 2. Método, para que se pueda repetir

- **Fuentes**: solo primarias. DOUE por el servicio de la Oficina de
  Publicaciones (`https://publications.europa.eu/resource/celex/<CELEX>` con
  `Accept: application/xhtml+xml` y `Accept-Language: spa`), BOE por su ELI
  consolidado. Ningún repositorio de terceros, ningún GitHub, ninguna
  recopilación comercial.
- **Extracción**: el texto se parte por artículo y por apartado, y sobre cada
  apartado se pasan tres juegos de marcadores en español (plazo, periodicidad,
  evento) más el de plazo abierto. Eso produce una lista de candidatos con su
  fragmento.
- **Revisión**: cada candidato se lee y se decide. El recuento final es de la
  revisión, no de la expresión regular. Los números concretos ("24 horas", "cada
  dos años", "diez días hábiles") se han vuelto a comprobar uno a uno contra el
  texto oficial antes de escribirlos aquí.
- **Marcas de honestidad**, visibles en la tabla:
  - **contado**: articulado leído entero por el extractor y candidatos revisados
    uno a uno. Las citas están.
  - **estimado**: núcleo verificado con cita, resto extrapolado del recuento de
    candidatos. Se dice sobre qué se estima.
  - **no verificado**: no se ha podido establecer. Se dice qué haría falta.

Para los tres marcos ya transcritos con relojes (`ens`, `rgpd`, `cra`) el censo
cruza además contra el propio `paquete.json`, que es la transcripción del autor
y manda sobre el extractor.

Este extractor es la misma tubería única de ingesta que decide el punto 5 de la
decisión D-1 (`docs/decisiones.md`): entrada por el ELI del BOE y por EUR-Lex,
reejecutable, porque el mecanismo que produce el censo es el mismo que va a
producir la vigilancia normativa semanal.

## 3. La frontera legal aplicada al censo

Los ocho marcos de estrato referencial (`iso27001`, `iso27002`, `iso22301`,
`iso42001`, `iso27701`, `soc2`, `pci-dss`, `tisax`) **no se pueden censar sin la
copia del cliente**, y eso no es una excusa, es el resultado. Contar cuántas
cláusulas de una norma ISO llevan cadencia exige leer la norma, y leerla para
extraer ese recuento y publicarlo es exactamente lo que la licencia no permite.

Lo que sí se puede afirmar sin abrir el texto, y se afirma:

- La estructura pública de cada una (número de cláusulas, número de controles del
  anexo) no es contenido protegible y está en la tabla.
- Que las normas de sistema de gestión de la familia ISO **no fijan ninguna
  cadencia numérica** y dejan el intervalo a la organización. Este proyecto ya lo
  publica en `paquetes/iso27001/RITUALES.md`, y es la razón por la que los seis
  relojes del paquete `iso27001` son rituales de dutiq y no obligaciones de la
  norma. Ese hecho vale para las cinco ISO del corpus.
- Que ese hecho es información de venta, no una carencia: significa que el valor
  del paquete referencial está en poner el número y defenderlo, no en copiar.

Para `pci-dss` y `tisax` la situación es distinta y peor de verificar: PCI DSS sí
fija cadencias numéricas en su propio texto, y TISAX tiene una vigencia de
etiqueta que publica ENX en su proceso. Ninguna de las dos se cuenta aquí porque
el recuento exacto exige la copia. Se marcan "no verificado" y se dice qué haría
falta.

## 4. La tabla

Ordenada por densidad de reloj, no alfabéticamente. "Total" es la suma de las
tres columnas. "Núm." es cuántas de las periodicidades traen número. El total no
es el criterio de autoría por sí solo: eso se resuelve en la sección 7.

| # | Marco | Estrato | Plazo | Periodicidad (núm.) | Evento | Total | Marca | Alcance para el comprador objetivo |
|---|---|---|---|---|---|---|---|---|
| 1 | nis2-tecnica | transcrito | 0 | 41 (3) | 20 | **61** | contado | alto, es la lista de control operativa de NIS2 |
| 2 | dora | transcrito | 4 | 18 (9) | 10 | **32** | contado | sectorial financiero, denso |
| 3 | ai-act | transcrito | 14 | 0 | 10 | **24** | contado | transversal creciente |
| 4 | cra | transcrito | 7 | 1 (0) | 8 | **16** | contado | alto para quien fabrica software |
| 5 | mica | transcrito | 7 | 5 (5) | 4 | **16** | estimado | sectorial muy estrecho |
| 6 | mdr | transcrito | 5 | 5 (4) | 5 | **15** | estimado | sectorial estrecho |
| 7 | ens | transcrito | 2 | 7 (6) | 4 | **13** | contado | alto en España, ya construido |
| 8 | nis2-ue | transcrito | 5 | 1 (0) | 5 | **11** | contado | alto, pero es directiva sin transponer |
| 9 | psd2 | transcrito | 6 | 2 (2) | 3 | **11** | estimado | sectorial, y en España vincula el RDL |
| 10 | rgpd | transcrito | 4 | 0 | 6 | **10** | contado | máximo, alcanza a todos |
| 11 | ley2-2023 | transcrito | 7 | 0 | 3 | **10** | contado | alto, desde 50 empleados |
| 12 | dga | transcrito | 5 | 1 (1) | 3 | **9** | contado | muy bajo, población estrecha |
| 13 | csrd | transcrito | 3 | 5 (5) | 0 | **8** | contado | medio, y aplazado |
| 14 | lopdgdd | transcrito | 5 | 0 | 3 | **8** | contado | alto en España, complementa RGPD |
| 15 | eidas2 | transcrito | 3 | 2 (2) | 3 | **8** | contado | bajo salvo prestador de confianza |
| 16 | data-act | transcrito | 4 | 0 | 4 | **8** | contado | medio, sube desde 09-2025 |
| 17 | iso27001 | referencial | 0 | 0 | 0 | **0** en la norma | contado (el 0) | máximo, ya construido con 6 rituales de dutiq |
| 18 | eni | transcrito | 0 | 0 | 0 | **0** | contado | nulo, no tiene reloj |
| 19 | iso27002 | referencial | ? | ? | ? | no verificado | no verificado | catálogo de 93 controles, sin cadencia propia |
| 20 | iso22301 | referencial | ? | ? | ? | no verificado | no verificado | estructura armonizada, cláusulas 4 a 10 |
| 21 | iso42001 | referencial | ? | ? | ? | no verificado | no verificado | estructura armonizada más anexo de controles |
| 22 | iso27701 | referencial | ? | ? | ? | no verificado | no verificado | extensión de 27001 y 27002 |
| 23 | soc2 | referencial | ? | ? | ? | no verificado | no verificado | criterios TSC, series CC1 a CC9 |
| 24 | pci-dss | referencial | ? | ? | ? | no verificado | no verificado | 12 requisitos, sí trae cadencias propias |
| 25 | tisax | referencial | ? | ? | ? | no verificado | no verificado | catálogo VDA ISA y vigencia de etiqueta |
| 26 | cis | delegado | 0 | 0 | 0 | **0** | contado | el reloj lo pone la herramienta |
| 27 | stig | delegado | 0 | 0 | 0 | **0** | contado | el reloj lo pone la herramienta |
| 28 | magerit | propio | 0 | 0 | 0 | **0** | contado | metodología, sin obligaciones |
| 29 | nist-800-53 | importado | n/a | n/a | n/a | sin autoría prevista | n/a | fuera por decisión D-1 |
| 30 | nist-csf | importado | n/a | n/a | n/a | sin autoría prevista | n/a | fuera por decisión D-1 |

**Totales de lo verificado**: 81 obligaciones con plazo explícito, 88 con
periodicidad explícita (de las cuales 37 con cadencia numérica y 51 con cadencia
declarada pero sin cuantificar), 91 con evento disparador explícito. Total 260
relojes en 16 marcos.

**Marcos en "no verificado"**: 7, todos de estrato referencial (iso27002,
iso22301, iso42001, iso27701, soc2, pci-dss, tisax). El motivo es el mismo en los
siete y está en la sección 3. Lo que haría falta: la copia licenciada del
cliente, leída dentro de su instancia, sin que el recuento salga del paquete
propio de ese cliente.

## 5. Las citas, marco por marco

Cada línea es una cuenta con su artículo. Si un artículo no está aquí, no está
contado.

### ens (RD 311/2022, BOE-A-2022-7191, más las tres ITS) - contado

Contado contra el `paquete.json`, que ya tiene 132 obligaciones y 8 relojes con
24 casos dorados en verde, y comprobado contra el consolidado del BOE.

- **Plazo (2)**: disposición transitoria única, apartado 1 (adecuación en 24
  meses); ITS de Notificación de Incidentes, apartado IV.3 (notificación al CCN
  sin demora para impacto alto o superior, modelada como PT0H).
- **Periodicidad (7, seis con número)**: art. 31.1 (auditoría ordinaria al menos
  cada dos años); anexo I, apartado 1 (reevaluación anual de la categoría); ITS
  del Informe del Estado de la Seguridad, apartado III.2 (INES anual); ITS de
  Notificación de Incidentes, apartado VI (estadísticas anuales); ITS de
  Conformidad, apartado III.2 (autoevaluación bienal, categoría básica); ITS de
  Conformidad, apartado III.3 (certificación bienal, media y alta); art. 10.3
  (reevaluación periódica de las medidas, sin cuantificar).
- **Evento (4)**: art. 31.1, párrafo segundo (auditoría extraordinaria por
  modificación sustancial, y además reinicia el cómputo de los dos años); anexo
  I, apartado 1 (modificación significativa de los criterios de determinación);
  art. 33.2 (incidente con impacto significativo, sector público, al CCN); art.
  33.7 (sector privado que presta a entidades públicas, al INCIBE-CERT).

Pendiente conocido y ya documentado en `ens/COBERTURA.md`: los refuerzos del
anexo II y la tabla de aplicación por nivel, que esperan a las reglas de
aplicabilidad.

### rgpd (Reglamento (UE) 2016/679) - contado

56 líneas candidatas, 5 marcadas de autoridad por el filtro automático y muchas
más descartadas en la revisión: los artículos 51 a 97 son casi todos plazos entre
autoridades de control y no son del responsable.

- **Plazo (4)**: art. 12.3 (un mes desde la recepción de la solicitud,
  prorrogable dos meses más); art. 12.4 (un mes para informar de la no
  actuación); art. 14.3.a (un mes desde la obtención de los datos); art. 33.1
  (72 horas, ya transcrito con tres dorados en verde).
- **Periodicidad (0)**. Este es el hallazgo del RGPD y conviene decirlo en voz
  alta: **el RGPD no impone al responsable ni al encargado ninguna cadencia**.
  Las cinco periodicidades que aparecen en el texto son de autoridades (art.
  41.2, 45.3, 59, 97.1) o del Comité. Todo lo que un cliente llama "la revisión
  anual del RGPD" lo pone él, no la norma.
- **Evento (6)**: art. 12.3 (solicitud del interesado); art. 19 (comunicación a
  cada destinatario tras rectificación o supresión); art. 33.1 (violación de
  seguridad, responsable); art. 33.2 (violación, encargado hacia responsable);
  art. 34.1 (alto riesgo, comunicación al interesado); art. 35.1, encadenado con
  el 36.1 (tratamiento de alto riesgo, evaluación de impacto y consulta previa).

### cra (Reglamento (UE) 2024/2847) - contado

- **Plazo (7)**: art. 13.8 (período de soporte de al menos cinco años); art.
  14.2.a (alerta temprana de vulnerabilidad aprovechada activamente, 24 horas
  desde el conocimiento, ya transcrito); art. 14.2.b (notificación, 72 horas);
  art. 14.2.c (informe final, 14 días desde que se dispone de medida correctora o
  paliativa); art. 14.4.a (alerta temprana de incidente grave, 24 horas); art.
  14.4.b (notificación, 72 horas); art. 14.4.c (informe final, un mes desde la
  notificación de la letra b).
- **Periodicidad (1, sin número)**: anexo I, parte II, punto 3 (exámenes y
  pruebas periódicos de la seguridad del producto).
- **Evento (8)**: art. 13.21 (saber o tener motivos para creer que hay no
  conformidad); art. 14.1 (vulnerabilidad aprovechada activamente); art. 14.3
  (incidente grave); art. 14.8 (conocimiento de una u otro, informar a los
  usuarios); art. 19.5 (importador); art. 20.4 (distribuidor); art. 22.1 y 22.2
  (modificación sustancial, que convierte a quien la hace en fabricante); art.
  57.2 (requerimiento de la autoridad, medidas correctoras en el plazo fijado).

Nota de vigencia ya recogida en el paquete: el art. 14 aplica desde el
11-09-2026, el capítulo IV desde el 11-06-2026 y el resto desde el 11-12-2027
(art. 71.2).

### nis2-ue (Directiva (UE) 2022/2555) - contado

47 líneas candidatas, 23 marcadas de autoridad. Es una directiva, así que casi
todo va redactado como "los Estados miembros velarán por que las entidades...".
La obligación de la entidad está dentro, pero el reloj que vincula en España es
el de la transposición.

- **Plazo (5)**: art. 23.4.a (alerta temprana, 24 horas); art. 23.4.b
  (notificación del incidente, 72 horas); art. 23.4.d (informe final, un mes);
  art. 27.3 (cambios en la información registrada, tres meses); art. 27.2 (fecha
  límite de registro para proveedores de DNS, nube, centros de datos y afines,
  17-01-2025). El art. 23.4.c (informe intermedio) no lleva número: se pide.
- **Periodicidad (1, sin número)**: art. 20.2 (formación periódica de los órganos
  de dirección).
- **Evento (5)**: art. 23.1 (incidente significativo); art. 23.2 (ciberamenaza
  significativa, comunicación a los destinatarios del servicio); art. 21.4
  (constatar que no se cumplen las medidas del 21.2); art. 27.3 (cambio en la
  información registrada); art. 23.4 en su conjunto, disparado por el
  conocimiento del incidente.

**Estado de la transposición española**: no verificado. La búsqueda en fuentes
secundarias a fecha del censo dice que el Anteproyecto de Ley de Coordinación y
Gobernanza de la Ciberseguridad, aprobado en Consejo de Ministros el 14-01-2025,
seguía en tramitación y no publicado en el BOE. No se ha localizado publicación
en el BOE consolidado. Lo que haría falta: comprobar el BOE del día antes de
escribir el paquete, porque si la ley ha salido, los plazos que vinculan son los
suyos y no los de la directiva.

### nis2-tecnica (Reglamento de Ejecución (UE) 2024/2690) - contado

El marco más denso del corpus y el más barato de escribir, porque todo el reloj
vive en un anexo con numeración estable de 207 puntos.

- **Plazo (0)**. Verificado: cero plazos numéricos para la entidad. Las dos
  únicas apariciones de "en un plazo de" son el art. 4, y no son un plazo sino un
  criterio de agregación (dos incidentes en seis meses cuentan como uno
  significativo).
- **Periodicidad (41, tres con número)**: con número, anexo puntos 1.1.2
  (revisión de la política al menos una vez al año), 2.1.4 (evaluación de riesgos
  y plan de tratamiento, como mínimo anualmente) y 10.1.3 (revisión de la
  asignación de personal a roles, al menos una vez al año). Sin cuantificar, 30
  puntos con "a intervalos planificados" y 8 con "periódicamente", repartidos por
  las trece secciones del anexo.
- **Evento (20)**: 19 puntos del anexo usan la misma fórmula de disparador
  (incidentes significativos o cambios significativos en las operaciones o los
  riesgos), más el art. 4 de incidentes recurrentes.

Lo que hace a este marco excepcional para dutiq: 41 cadencias y 20 disparadores
que salen de **una sola plantilla repetida**, y la mayoría de las cadencias no
traen número, así que el valor del paquete es precisamente poner el número por
defecto y dejar que el cliente lo cambie, que es el patrón ya construido y
probado en `iso27001`.

### dora (Reglamento (UE) 2022/2554, con el Reglamento Delegado (UE) 2025/301) - contado

- **Plazo (4)**: y aquí está el hallazgo que cambia cómo se escribe este paquete.
  El nivel 1 **no tiene ni un solo plazo numérico para la entidad financiera**:
  el art. 19.4 delega los plazos. Los tres plazos reales están en el Reglamento
  Delegado (UE) 2025/301, art. 5.1: letra a, notificación inicial en 4 horas
  desde la clasificación del incidente como grave y como tope 24 horas desde que
  la entidad tuvo conocimiento; letra b, informe intermedio en 72 horas desde la
  notificación inicial; letra c, informe final un mes después del informe
  intermedio. El cuarto plazo sí está en el nivel 1: art. 31.12 (12 meses para
  que el proveedor esencial de tercer país establezca filial en la Unión, ventana
  que la entidad tiene que vigilar).
- **Periodicidad (18, nueve con número)**: con número, art. 6.5 (documentación y
  revisión del marco al menos una vez al año), 8.1 (revisión anual de la
  clasificación de funciones y activos), 8.2 (revisión anual de los escenarios de
  riesgo), 8.7 (evaluación anual del riesgo en sistemas heredados), 11.6 (pruebas
  anuales de los planes de continuidad y de respuesta y recuperación), 13.5
  (informe anual del directivo de TIC al órgano de dirección), 24.6 (pruebas
  anuales de los sistemas que sustentan funciones esenciales), 26.1 (pruebas de
  penetración basadas en amenazas al menos cada tres años), 28.3 (comunicación
  anual a la autoridad sobre los acuerdos de servicios de TIC). Sin cuantificar,
  art. 5.2, 5.4, 6.6, 8.6, 11.4, 12.2, 16.2, 28.2 y 28.8. Corregido durante el
  censo: la revisión anual del marco por el órgano de dirección aparece redactada
  así en un considerando, no en el art. 5.2, y el art. 5.2 solo dice
  "periódicamente".
- **Evento (10)**: art. 6.5 e 16.2 (incidente grave dispara la revisión del
  marco); 8.3 (cambio importante en la infraestructura dispara evaluación de
  riesgo); 8.6 (cambios importantes disparan la actualización del inventario);
  11.6 (cambio sustancial dispara las pruebas); 13.2 (incidente grave dispara la
  revisión posterior); 19.1 (incidente grave dispara la notificación); 19.3
  (consecuencias para los clientes disparan la información a estos); 26.6 (fin de
  la prueba avanzada dispara la entrega); 45.3 (incorporación a un acuerdo de
  intercambio de información dispara la notificación).

### ai-act (Reglamento (UE) 2024/1689) - contado

110 líneas candidatas, 23 marcadas de autoridad, y de nuevo la mayoría de los
plazos son del capítulo de vigilancia del mercado y no del proveedor.

- **Plazo (14 apartados, 15 relojes)**: art. 5.3 (uso urgente de identificación
  biométrica remota en tiempo real, solicitud de autorización a más tardar en 24
  horas); art. 18.1 (conservación de documentación 10 años); art. 19.1
  (conservación de registros al menos 6 meses); art. 22.3 (representante
  autorizado, 10 años); art. 23.5 (importador, 10 años); art. 26.6 (responsable
  del despliegue, registros al menos 6 meses); art. 26.10 (identificación
  biométrica remota en diferido, autorización a más tardar en 48 horas); art. 47.1
  (declaración UE de conformidad a disposición 10 años); art. 52.1 (proveedor de
  modelo de uso general, notificación a la Comisión antes de transcurridas dos
  semanas desde que se cumple o se sabe que se cumplirá el umbral); art. 54.3
  (representante autorizado del proveedor de modelo de uso general, documentación
  del anexo XI a disposición 10 años); art. 60.4 (pruebas en condiciones reales,
  máximo 6 meses prorrogables otros 6, que son dos relojes); art. 73.2 (incidente
  grave, notificación a más tardar 15 días desde el conocimiento); art. 73.3
  (infracción generalizada o incidente del art. 3.49.b, a más tardar 2 días); art.
  73.4 (fallecimiento, plazo no superior a 10 días).
- **Periodicidad (0)**. Segundo hallazgo fuerte del censo: **el AI Act no impone
  al proveedor ni al responsable del despliegue ninguna cadencia numérica**. La
  vigilancia poscomercialización del art. 72 es continua y su plan lo fija un
  acto de ejecución. Todas las periodicidades del texto son de la Comisión, de la
  Oficina de IA, de los Estados miembros o de las autoridades.
- **Evento (10)**: art. 20.1 (no conformidad detectada); 20.2 (riesgo del art.
  79.1 conocido, investigación inmediata); 22.4 y 54.5 (el representante
  autorizado pone fin al mandato); 24.4 (distribuidor); 26.5 (el responsable del
  despliegue detecta riesgo); 43.4 (modificación sustancial, nueva evaluación de
  la conformidad); 51 con 52 (el modelo alcanza el umbral de riesgo sistémico);
  60.7 (incidente grave en pruebas en condiciones reales); 73.1 (incidente
  grave); 111.2 (cambio significativo en el diseño de sistemas ya en el mercado).

### data-act (Reglamento (UE) 2023/2854) - contado

- **Plazo (4 apartados, 5 relojes)**: art. 18.2 (denegar o pedir la modificación
  de la solicitud de un organismo del sector público, a más tardar 5 días hábiles
  si la solicitud responde a una emergencia pública y 30 días hábiles en los demás
  casos de necesidad excepcional, que son dos relojes); art. 25.2.a (período
  transitorio obligatorio máximo de 30 días naturales para el cambio de
  proveedor); art. 25.2.d (plazo máximo de preaviso, que no excederá de dos
  meses); art. 25.4 (notificación de inviabilidad técnica del cambio, 14 días
  hábiles desde la solicitud). Comprobado y corregido durante el censo: el art.
  18.1, que es la obligación de poner los datos a disposición, **no lleva número**,
  solo "sin demora indebida". El plazo numérico es del 18.2 y es para negarse, no
  para cumplir.
- **Periodicidad (0)**.
- **Evento (4)**: art. 4.1 (petición del usuario); art. 5.1 (petición de un
  tercero en nombre del usuario); art. 14 con 18 (solicitud de un organismo del
  sector público por necesidad excepcional); art. 25.2 (solicitud de cambio de
  proveedor).

### dga (Reglamento (UE) 2022/868) - contado

Marco de población muy estrecha: proveedores de servicios de intermediación de
datos y organizaciones reconocidas de gestión de datos con fines altruistas.

- **Plazo (5)**: art. 11.12 (modificación de la información notificada, 14 días
  desde el día de la modificación); art. 11.13 (cese de actividades, 15 días);
  art. 14.3 (30 días para manifestar opinión ante las observaciones de la
  autoridad); art. 19.7 (modificación de la información, 14 días); art. 24.3 (30
  días, equivalente al 14.3 para las organizaciones altruistas).
- **Periodicidad (1, con número)**: art. 20.2 (informe anual de actividad).
- **Evento (3)**: art. 11.12 y 19.7 (modificación de la información); art. 11.13
  (cese de actividades); art. 21.5 (transferencia, acceso o utilización no
  autorizados de datos no personales, informar a los titulares).

### eidas2 (Reglamento (UE) 2024/1183, sobre el consolidado del 910/2014) - contado

Aviso para el autor: el `urn` del paquete apunta al reglamento modificativo
2024/1183, pero **las obligaciones no viven ahí**, viven en el texto consolidado
del Reglamento (UE) 910/2014. El censo se ha hecho sobre el consolidado
(02014R0910-20241018).

- **Plazo (3)**: art. 19 bis.1.b (prestador no cualificado, notificación de
  violación de seguridad o interrupción con impacto significativo, a más tardar
  24 horas desde el conocimiento); art. 24.2.f ter (prestador cualificado, a más
  tardar 24 horas desde que se produce el incidente); art. 12 bis (cancelación de
  la certificación si la vulnerabilidad detectada no se subsana en tres meses).
- **Periodicidad (2, ambas con número)**: art. 20.1 (los prestadores cualificados
  serán auditados al menos cada 24 meses); art. 5 quater y 12 bis (evaluación de
  la vulnerabilidad cada dos años como condición de la validez de la
  certificación, que dura hasta cinco años).
- **Evento (3)**: violación de seguridad o interrupción (19 bis y 24.2);
  detección de vulnerabilidad no subsanada (12 bis); cese de actividad del
  prestador cualificado (art. 24.2.h y siguientes, obligación de conservación).

### lopdgdd (LO 3/2018) - contado

- **Plazo (5)**: art. 20.1.c (informar del derecho a ejercer los arts. 15 a 22
  dentro de los treinta días siguientes a la notificación de la deuda, sistemas
  de información crediticia); art. 22.3 (videovigilancia, supresión en el plazo
  máximo de un mes desde la captación); art. 34.3 (comunicar a la AEPD las
  designaciones, nombramientos y ceses de delegado de protección de datos en el
  plazo de diez días); art. 37.2 (el delegado responde en el plazo de un mes a la
  reclamación remitida por la autoridad); art. 65.4 (el responsable o encargado
  responde a la reclamación remitida en el plazo de un mes).
- **Periodicidad (0)**.
- **Evento (3)**: art. 22.3 (captación de imágenes); art. 34.3 (designación,
  nombramiento o cese del delegado); art. 36.4 (el delegado aprecia una
  vulneración relevante y la comunica inmediatamente a administración y
  dirección).

### ley2-2023 (Ley 2/2023) - contado

Marco barato de escribir y de alcance alto: alcanza a toda entidad privada de 50
o más trabajadores y a todo el sector público.

- **Plazo (6 apartados, 7 relojes)**: art. 7.2 (reunión presencial a solicitud
  del informante, dentro del plazo máximo de siete días); art. 8.3 (notificar a
  la Autoridad el nombramiento y el cese del Responsable del Sistema en los diez
  días hábiles siguientes); art. 9.2.c (acuse de recibo en el plazo de siete días
  naturales desde la recepción); art. 9.2.d (respuesta a las actuaciones de
  investigación en no más de tres meses desde la recepción, o desde el
  vencimiento del plazo de siete días si no hubo acuse, más una ampliación de
  hasta tres meses adicionales en casos de especial complejidad, que son dos
  relojes); art. 26.2 (conservación máxima de diez años en el libro-registro);
  art. 32.4 (supresión de los datos transcurridos tres meses desde la recepción
  sin que se hayan iniciado actuaciones).
- **Periodicidad (0)**. Tercer hallazgo: **la ley no obliga a revisar el sistema
  interno de información con ninguna cadencia**. La revisión trienal del art. 22
  es de la Autoridad Independiente de Protección del Informante, no de la
  empresa.
- **Evento (3)**: recepción de una comunicación (dispara 9.2.c y 9.2.d);
  nombramiento o cese del Responsable del Sistema (8.3); ausencia de actuaciones
  de investigación a los tres meses (32.4).

### eni (RD 4/2010) - contado

- **Plazo, periodicidad y evento: 0**. Es el resultado más limpio del censo. Los
  dos únicos plazos del texto son transitorios y están vencidos hace años: el
  plan de adecuación con ejecución no superior a 48 meses desde la entrada en
  vigor, y los 24 meses de adaptación que remiten a la disposición transitoria
  primera del RD 1671/2009. El ENI **no tiene ningún reloj vivo**.
- No verificado: las Normas Técnicas de Interoperabilidad son resoluciones
  publicadas aparte y no se han censado. Lo que haría falta: censar las NTI una a
  una antes de decidir si el paquete `eni` merece autoría, porque si hay reloj
  está ahí y no en el real decreto.

### csrd (Directiva (UE) 2022/2464, sobre la Directiva 2013/34/UE consolidada) - contado

Igual que con eidas2, el `urn` apunta a la directiva modificativa pero las
obligaciones viven en la Directiva contable 2013/34/UE consolidada, que es sobre
la que se ha censado.

- **Plazo (3)**: art. 30.1 (publicación de los estados financieros anuales y el
  informe de gestión en un plazo razonable no superior a doce meses desde la
  fecha del balance); art. 40 quinquies (publicación del informe de
  sostenibilidad de filiales y sucursales de empresas de terceros países, doce
  meses); art. 48 quinquies (publicación del informe relativo al impuesto sobre
  sociedades, doce meses desde el cierre).
- **Periodicidad (5, todas con número y todas anuales)**: art. 4 (estados
  financieros anuales); art. 19 bis (información de sostenibilidad en el informe
  de gestión); art. 29 bis (información de sostenibilidad consolidada); art. 40
  bis (informes de sostenibilidad de empresas de terceros países); art. 48 ter
  (informe relativo al impuesto sobre sociedades).
- **Evento (0)**. La CSRD es puro calendario.
- **Dos avisos que pesan más que las cuentas**: es una directiva y **España no la
  ha transpuesto**, la Ley de Sostenibilidad Empresarial seguía en tramitación
  parlamentaria en 2026 según fuentes secundarias, y las fechas de aplicación las
  movió dos años la Directiva (UE) 2025/794, la llamada "stop the clock". Escribir
  hoy los relojes de csrd es escribir relojes que van a cambiar de fecha.

### mdr (Reglamento (UE) 2017/745) - estimado

123 líneas candidatas, 28 marcadas de autoridad. Se ha verificado con cita el
núcleo de vigilancia poscomercialización y conservación, que es donde está el
reloj del fabricante. El resto (investigaciones clínicas, organismos notificados,
EUDAMED) no se ha revisado apartado a apartado, de ahí la marca "estimado".

- **Plazo (5 relojes en 4 apartados, verificados)**: art. 10.8 (conservación de la
  documentación técnica y la declaración UE al menos diez años desde la última
  introducción en el mercado, y al menos quince para productos implantables, que
  son dos relojes); art. 87.3 (incidente grave, notificación a más tardar quince
  días desde el conocimiento); art. 87.4 (amenaza grave para la salud pública, a
  más tardar dos días); art. 87.5 (fallecimiento o deterioro grave de un estado de
  salud, a más tardar diez días).
- **Periodicidad (5, cuatro con número)**: art. 31.5 (confirmación de la exactitud
  de los datos un año después de la primera presentación y después cada dos
  años); art. 86.1 (PSUR actualizado como mínimo una vez cada año para clases IIb
  y III); art. 86.1 (PSUR como mínimo cada dos años para clase IIa); art. 61.11
  (actualización de la evaluación clínica durante todo el ciclo de vida, con
  cadencia anual para clase III e implantables); art. 85 (informe de vigilancia
  poscomercialización de clase I, sin cadencia numérica).
- **Evento (5, estimado)**: art. 87.1 (incidente grave); art. 87.1 (acción
  correctiva de seguridad sobre el terreno); art. 88.1 (aumento estadísticamente
  significativo de la frecuencia o gravedad de incidentes no graves, sin plazo
  numérico); modificación que exige nueva evaluación de la conformidad; no
  conformidad detectada.

### mica (Reglamento (UE) 2023/1114) - estimado

209 líneas candidatas y 97 marcadas de autoridad, la proporción más alta del
censo. MiCA es sobre todo un reglamento de procedimiento de autorización, y la
mayoría de sus 137 líneas de plazo son plazos de tramitación de la autoridad
competente, de la ABE o de la AEVM, no del obligado.

- **Plazo (7, verificados del lado del obligado)**: art. 10.1 (publicación del
  resultado de la oferta pública, 20 días hábiles desde el final del plazo de
  suscripción); art. 14.3 (reembolso a los titulares, a más tardar 25 días
  naturales desde la cancelación de la oferta); art. 36.10 (notificación del
  resultado de la auditoría de la reserva, a más tardar seis semanas desde la
  fecha de referencia de la valoración); art. 46.2 (notificación del plan de
  recuperación, seis meses desde la autorización); art. 47.3 (notificación del
  plan de reembolso, seis meses); art. 55 (las dos variantes anteriores para
  emisores de fichas de dinero electrónico, seis meses desde la oferta pública);
  art. 85.2 (notificación al alcanzar el umbral de usuarios activos, dos meses).
- **Periodicidad (5, todas con número)**: art. 10.2 (publicación al menos mensual
  del número de unidades en circulación por los oferentes sin plazo de oferta);
  art. 22.1 (comunicación trimestral a la autoridad para emisores de fichas
  referenciadas a activos con valor de emisión superior a 100 millones de euros);
  art. 30.1 (actualización al menos mensual de la información sobre la cantidad en
  circulación y la reserva de activos); art. 35.1 (revisión anual del importe de
  fondos propios); art. 72.4 (revisión al menos anual de la política de conflictos
  de intereses de los proveedores de servicios de criptoactivos).
- **Evento (4, estimado)**: cruce del umbral de significatividad; cancelación de
  la oferta; reclamación de un titular; incidente que active el plan de
  recuperación.

### psd2 (Directiva (UE) 2015/2366) - estimado

93 líneas candidatas, 28 marcadas de autoridad. Verificado el núcleo de
ejecución, reembolso y reclamaciones.

- **Plazo (6 apartados, 7 relojes, verificados)**: art. 71.1 (el usuario dispone
  de trece meses para notificar, ventana que el proveedor tiene que vigilar); art.
  73.1 (devolución del importe de la operación no autorizada a más tardar al final
  del día hábil siguiente); art. 77.2 (devolución solicitada, diez días hábiles
  desde la recepción de la solicitud); art. 82.2 (plazo más largo para operaciones
  no denominadas en euros, que no excederá de cuatro días hábiles); art. 83.1
  (abono en la cuenta del proveedor del beneficiario a más tardar al final del día
  hábil siguiente); art. 101.2 (respuesta a reclamaciones a más tardar quince días
  hábiles, y como máximo treinta y cinco en situaciones excepcionales, que son dos
  relojes).
- **Periodicidad (2, ambas con número)**: art. 95.2 (evaluación actualizada y
  completa de los riesgos operativos y de seguridad, anualmente o a intervalos más
  breves si la autoridad lo determina); art. 96.6 (datos estadísticos sobre fraude
  a la autoridad competente por lo menos una vez al año).
- **Evento (3)**: art. 96.1 (incidente operativo o de seguridad de carácter
  grave, notificación sin demora indebida, sin número en la directiva); art. 73
  (operación no autorizada observada o notificada); art. 76 y 77 (solicitud de
  devolución).
- **Aviso**: en España lo que vincula es el Real Decreto-ley 19/2018, no la
  directiva. No se ha censado el RDL. Lo que haría falta antes de escribir el
  paquete: censar el RDL 19/2018 y comprobar si las cifras coinciden.

### Los ocho referenciales - no verificado, con lo que sí se sabe

- `iso27001`: **0 cadencias numéricas en la norma**, hecho ya publicado por el
  proyecto en `paquetes/iso27001/RITUALES.md`. Los 6 relojes del paquete (cinco
  cadencias de 12 o 24 meses y un plazo de 10 días hábiles) son rituales de dutiq
  con su justificación escrita, no obligaciones de ISO. El paquete tiene 129
  obligaciones y 18 dorados. Cuántas cláusulas de la norma llevan cadencia
  declarada sin cuantificar: no verificable sin la copia del cliente.
- `iso27002`: 93 controles en cuatro temas. Es un catálogo de guía, no de
  requisitos auditables, y no fija cadencias propias. Recuento: no verificable.
- `iso22301`: estructura armonizada, cláusulas 4 a 10. Recuento: no verificable.
- `iso42001`: estructura armonizada más anexo de controles. Recuento: no
  verificable.
- `iso27701`: extensión de 27001 y 27002 con anexos separados para responsable y
  encargado. Recuento: no verificable. **Aviso adicional**: el paquete declara la
  edición 2019, y conviene comprobar si debe apuntar a una edición posterior antes
  de escribir nada.
- `soc2`: criterios de servicios de confianza, series CC1 a CC9 más las categorías
  adicionales. No es una norma con cadencias, es un marco de atestación cuya
  periodicidad la fija el tipo de informe y el período de cobertura que el cliente
  contrata. Recuento: no verificable, y probablemente no aplicable.
- `pci-dss`: 12 requisitos. **Es el único referencial de los ocho que sí fija
  cadencias numéricas en su propio texto**, y eso lo convierte en el más valioso
  de los ocho y en el único imposible de censar sin la copia. Recuento: no
  verificable sin la copia del cliente. Lo que haría falta: leer la copia dentro
  de la instancia del cliente y que el recuento no salga de su paquete propio.
- `tisax`: catálogo VDA ISA con etiquetas de vigencia limitada. La vigencia de la
  etiqueta es proceso público de ENX, no texto de la norma, pero el recuento de
  controles con cadencia exige el catálogo. Recuento: no verificable.

### Delegados, propio e importados

- `cis` y `stig`: 0 relojes propios. El contenido lo tiene la herramienta
  (OpenSCAP, Trivy, Prowler) y la cadencia de escaneo la pone el operador. No hay
  nada que censar y no hay nada que escribir.
- `magerit`: 0 relojes. Es una metodología de análisis de riesgo, no un cuerpo de
  obligaciones. La cadencia del análisis de riesgo la impone el ENS, no MAGERIT.
- `nist-800-53` y `nist-csf`: **sin autoría prevista**, por la decisión D-1 de
  `docs/decisiones.md`. Mil controles federales estadounidenses no le sirven a un
  CISO europeo de 20 a 5.000 empleados, y el modelo de OSCAL (`catalog > group >
  control > part`) no tiene campo para un plazo, así que un importador tendría que
  inventar el reloj. No hay importador OSCAL y no lo va a haber. Este censo aporta
  a esa decisión el número que le faltaba: de las 260 obligaciones con reloj
  contadas aquí, **ninguna se podría representar en OSCAL sin perder el reloj**.

## 6. Los cinco hallazgos que cambian el plan de autoría

1. **El reloj de un marco no siempre está en el texto del marco.** DORA no tiene
   ni un plazo numérico para la entidad financiera: están en el Reglamento
   Delegado (UE) 2025/301, art. 5.1. Un pipeline de autoría que solo lea el nivel
   1 escribe un paquete DORA sin relojes y no se entera. Consecuencia: el pipeline
   necesita un paso explícito de "buscar los actos de nivel 2 que completan este
   artículo" antes de dar un artículo por transcrito.

2. **Cuatro paquetes apuntan al instrumento equivocado.** `eidas2` apunta al
   modificativo 2024/1183 en vez de al consolidado 910/2014, `csrd` apunta al
   modificativo 2022/2464 en vez de a la Directiva contable consolidada. Y dos
   directivas, `nis2-ue` y `psd2`, no vinculan en España por sí mismas: lo que
   vincula es la transposición, que en PSD2 es el RDL 19/2018 y en NIS2 no existe
   todavía. Consecuencia: antes de escribir, resolver el instrumento.

3. **Tres marcos transversales no imponen ninguna cadencia y todo el mundo cree
   que sí.** RGPD (0), AI Act (0) y Ley 2/2023 (0). Cero cadencias para el
   obligado. Esto es a la vez un hallazgo de producto y un argumento de venta: la
   "revisión anual del RGPD" que el cliente tiene en su plan es un ritual suyo,
   no una obligación, y dutiq es la herramienta que distingue las dos cosas con
   la cita delante.

4. **La densidad de reloj no correlaciona con el tamaño del texto.** El
   Reglamento de Ejecución 2024/2690 tiene 16 artículos y 61 relojes. MiCA tiene
   149 artículos y 16 relojes del obligado, porque 97 de sus 209 líneas
   candidatas son plazos de la autoridad. Ordenar la autoría por "marcos
   importantes" habría puesto MiCA por delante de 2024/2690, que es exactamente
   la decisión equivocada.

5. **El corpus entero tiene 37 cadencias con número y 51 sin cuantificar.** Más
   de la mitad de las periodicidades del corpus no traen número. Eso confirma que
   el patrón de `iso27001` (dutiq pone un valor de partida razonable, lo justifica
   por escrito en `RITUALES.md` o `COMPUTO.md`, y el cliente lo cambia en su copia)
   no es una excepción del estrato referencial: es el modo por defecto de más de la
   mitad del corpus, incluido el transcrito.

## 7. Orden de autoría propuesto

Relojes primero, cruzando marcos. La unidad de trabajo no es el marco, es la
familia de reloj: escribir la misma primitiva temporal en siete marcos a la vez
cuesta menos que escribir siete marcos enteros, y activa más obligaciones por
hora de trabajo.

**Criterio, en una línea**: relojes verificados dividido por número de fuentes
primarias que hay que leer, ponderado por cuántos de los clientes objetivo
(organización europea de 20 a 5.000 empleados) alcanza el marco.

### Familia A: notificación escalonada de incidente. Primera, y con diferencia

Ocho marcos, veinte relojes, una sola primitiva (plazo con disparador
"conocimiento" o "clasificación"), y el motor ya la ejecuta: `rgpd.art33` y
`cra.art14` están en verde. Se escriben todos de una tacada porque comparten
régimen de cómputo y solo cambian el límite y el hito.

| Marco | Relojes | Por qué aquí |
|---|---|---|
| rgpd | art. 33.1 | ya escrito, sirve de patrón |
| cra | art. 14.2.a/b/c y 14.4.a/b/c | ya escrito el primero, faltan cinco de la misma forma |
| dora | Reg. Del. 2025/301 art. 5.1.a/b/c | tres relojes, un solo artículo, y es la venta al sector financiero |
| nis2-ue | art. 23.4.a/b/d | tres relojes, y el más pedido del mercado español |
| ens | ITS de Notificación de Incidentes IV.3 | ya escrito |
| eidas2 | art. 19 bis.1.b y 24.2.f ter | dos relojes de 24 horas, coste casi nulo |
| ai-act | art. 73.2, 73.3 y 73.4 | tres límites distintos sobre el mismo disparador, buen caso de prueba |
| psd2 | art. 96.1 | sin número en la directiva, se aparca hasta censar el RDL 19/2018 |

### Familia B: revisión anual del marco y de la apreciación de riesgos. Segunda

Cinco marcos, una primitiva periódica de 12 meses con disparador "última
ocurrencia". Es la cadencia que más aparece en todo el corpus y la que el cliente
ya tiene en su calendario, así que es la que convierte el producto en algo que se
mira todas las semanas.

1. **nis2-tecnica** (Reg. Ejec. 2024/2690), puntos 1.1.2, 2.1.4 y 10.1.3, más los
   38 puntos de cadencia sin cuantificar. Es el marco con mejor relación entre
   relojes y trabajo de todo el corpus: 61 relojes en un anexo con numeración
   estable.
2. **dora**, los nueve artículos con cadencia numérica (6.5, 8.1, 8.2, 8.7, 11.6,
   13.5, 24.6, 26.1, 28.3).
3. **iso27001**, ya hecho, sirve de referencia de cómputo.
4. **ens**, ya hecho.
5. **psd2** art. 95.2, cuando se resuelva el instrumento español.

### Familia C: auditoría y certificación de ciclo largo. Tercera

Cuatro marcos, primitiva periódica de 24 o 36 meses, y es la que produce el aviso
que un CISO agradece con seis meses de antelación, así que es la que vende el
escalado.

`ens` art. 31.1 e ITS de Conformidad III.2 y III.3 (ya hechos), `eidas2` art. 20.1
(24 meses), `dora` art. 26.1 (36 meses), `mdr` art. 31.5 (24 meses).

### Familia D: disparador por cambio sustancial. Cuarta

Seis marcos con el mismo disparador y ninguno con plazo: `nis2-tecnica` (19
puntos), `ens` art. 31.1 párrafo segundo, `dora` art. 8.3 y 11.6, `cra` art. 22,
`ai-act` art. 43.4, `iso27001` requisito 8.2. Se deja para la cuarta posición
porque **necesita el motor de eventos**, no solo el de ventana, y porque
`iso27001/RITUALES.md` ya avisa de que ese disparador se declara aparte.

### Familia E: retención y conservación. Quinta

Cinco marcos con una primitiva de duración larga que el motor todavía no
ejercita: `ai-act` art. 18.1 (10 años), `mdr` art. 10.8 (10 y 15 años),
`ley2-2023` art. 26.2 (10 años), `cra` art. 13.8 (5 años), `lopdgdd` art. 22.3 (1
mes). Va después porque su valor operativo es bajo (nadie se olvida de un plazo de
diez años) pero su valor de auditoría es alto.

### Familia F: derechos y reclamaciones con plazo. Sexta

`rgpd` art. 12.3 y 12.4, `ley2-2023` art. 9.2.c y 9.2.d, `psd2` art. 101.2 y
77.2, `data-act` art. 18.2, `lopdgdd` art. 37.2 y 65.4. Plazos cortos disparados
por una solicitud externa. Es la familia que más se parece a un ticket y por eso
la que más compite con herramientas que el cliente ya tiene.

### Lo que se deja para después, y por qué en una línea cada uno

- **mica**: 16 relojes reales pero población estrecha, y 97 de sus 209 líneas
  candidatas son plazos de la autoridad, no del obligado.
- **mdr**: densidad decente, población estrecha, y el reloj bueno (PSUR) ya está
  en el calendario de calidad de cualquier fabricante.
- **csrd**: la aplicación la aplazó dos años la Directiva (UE) 2025/794 y España
  no la ha transpuesto, así que hoy se escribirían fechas que van a cambiar.
- **dga**: cinco plazos y una cadencia, pero la población son proveedores de
  intermediación de datos, casi nadie.
- **psd2**: hay que censar antes el RDL 19/2018, porque la directiva no vincula.
- **eni**: cero relojes vivos. No se escribe hasta haber censado las NTI, y si las
  NTI tampoco tienen reloj, no se escribe nunca.
- **iso27002, iso22301, iso42001, iso27701**: cero cadencias numéricas en la
  norma, así que el paquete es una lista de rituales de dutiq. Se escriben cuando
  haya un cliente que los pida, replicando el patrón de `iso27001`, no antes.
- **soc2**: la periodicidad la fija el contrato de atestación, no el marco.
- **pci-dss** y **tisax**: no verificables sin la copia del cliente. Se escriben,
  si se escriben, dentro de la instancia del cliente y con su copia.
- **cis** y **stig**: sin reloj propio, no hay nada que escribir.
- **magerit**: sin reloj propio.
- **nist-800-53** y **nist-csf**: sin autoría prevista, decisión D-1.

## 8. Lo que este censo no ha verificado, en una lista

1. El estado de la transposición española de NIS2 en el BOE a la fecha de escribir
   el paquete. Haría falta: consulta al BOE consolidado ese mismo día.
2. Las Normas Técnicas de Interoperabilidad del ENI. Haría falta: censarlas una a
   una como resoluciones separadas.
3. El RDL 19/2018 (transposición española de PSD2). Haría falta: censarlo como
   marco propio o como capa del paquete `psd2`.
4. La Ley de Sostenibilidad Empresarial (transposición española de CSRD). Haría
   falta: esperar a su publicación.
5. Los recuentos de los siete referenciales sin verificar. Haría falta: la copia
   licenciada del cliente, leída dentro de su instancia.
6. Los apartados de `mdr`, `mica` y `psd2` fuera del núcleo verificado. Haría
   falta: revisar apartado a apartado los candidatos restantes, 123, 209 y 93
   líneas respectivamente.
7. La edición vigente de ISO/IEC 27701 frente a la que declara el paquete.
