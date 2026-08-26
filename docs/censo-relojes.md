# Censo de relojes de los 31 paquetes

Fecha del censo: 26-08-2026. Este documento **cuenta**, no transcribe. Lo que hay
son números de obligación con reloj y el número de artículo que respalda cada
número. Las pocas expresiones entrecomilladas son marcadores de búsqueda de dos o
tres palabras, siempre de textos de BOE o DOUE, que son reutilizables citando la
fuente. De ISO, PCI DSS, SOC 2, TISAX y CIS no aparece ni una palabra, y por eso
sus filas de la tabla están vacías.

**Cobertura**: `paquetes/` tiene hoy 31 directorios y aquí hay 31 filas, una por
directorio, sin excepción. La cuenta es literal, `ls -d paquetes/*/`, no la lista
de treinta marcos de `paquetes/CORPUS.md`: el trigésimo primero es
`demo-empresa`, que no es una norma pero sí es un paquete que carga, tiene
relojes y sale en la demo, así que tiene fila. Veintitrés filas llevan número
contado, una lleva número por construcción (`demo-empresa`), siete dicen "no
verificado" con el motivo, y dos dicen "sin autoría prevista" por la decisión
D-1.

Sirve para una sola cosa: **decidir el orden de autoría del corpus**. La
conclusión operativa está al final, en "Orden de autoría propuesto", y no está
ordenada por marco sino por familia de reloj, porque la unidad de trabajo real
no es el marco, es la primitiva temporal.

## 1. Qué se cuenta y qué no

**Unidad de cuenta**: una obligación con reloj, identificada por el par
(artículo, apartado), **cuyo destinatario es la organización obligada**. La tabla
de la sección 4 cuenta **obligaciones**, o sea apartados, no relojes: un apartado
que fija dos límites (diez años, y quince si el producto es implantable) es una
obligación y dos relojes, y la ficha del marco dice las dos cifras. La tercera
pasada encontró tres filas que habían puesto relojes donde la columna pide
obligaciones (`ley2-2023`, `mdr`, `psd2`) y las ha corregido. Se
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
  fragmento. **El juego de periodicidad es el que falla**, y por eso tiene
  sección propia justo debajo, la 2 bis, con el vocabulario completo.
- **Revisión**: cada candidato se lee y se decide. El recuento final es de la
  revisión, no de la expresión regular. Ojo con la trampa: un marcador que no
  está no produce ningún candidato, así que su ausencia no se ve en la revisión.
  Es exactamente lo que pasó en la primera pasada (sección 6 bis). Los números
  concretos ("24 horas", "cada
  dos años", "diez días hábiles") se han vuelto a comprobar uno a uno contra el
  texto oficial antes de escribirlos aquí.
- **Marcas de honestidad**, visibles en la tabla:
  - **contado**: articulado leído entero por el extractor y candidatos revisados
    uno a uno. Las citas están.
  - **estimado**: núcleo verificado con cita, resto extrapolado del recuento de
    candidatos. Se dice sobre qué se estima.
  - **no verificado**: no se ha podido establecer. Se dice qué haría falta.
- **Barrido de la tercera pasada**, el que cerró las tres filas que quedaban en
  "estimado". Sobre el mismo texto se pasan además tres rejillas que la segunda
  pasada no tenía y que están en la sección 2 ter: los **anexos**, los **actores
  distintos del principal** (importador, distribuidor, representante autorizado,
  promotor, agente económico) y la **retención documental**. Las tres son las que
  produjeron los hallazgos nuevos, incluidos cuatro en un marco que ya estaba
  marcado "contado".

Para los tres marcos ya transcritos con relojes (`ens`, `rgpd`, `cra`) el censo
cruza además contra el propio `paquete.json`, que es la transcripción del autor
y manda sobre el extractor.

Este extractor es la misma tubería única de ingesta que decide el punto 5 de la
decisión D-1 (`docs/decisiones.md`): entrada por el ELI del BOE y por EUR-Lex,
reejecutable, porque el mecanismo que produce el censo es el mismo que va a
producir la vigilancia normativa semanal.

## 2 bis. Cómo se busca la periodicidad sin cuantificar

Esta sección existe porque la primera pasada del censo se dejó 11 periodicidades
por no tener este vocabulario. Está aquí para que la siguiente ejecución no
dependa de que alguien se acuerde.

**El error a evitar**: buscar solo el adverbio. Contadas sobre los diecisiete
textos que tenía el censo en su segunda pasada (la tercera añadió el RDL 19/2018,
el RDL 12/2018, el RD 43/2021 y las doce NTI, y no ha rehecho este conteo de
frecuencias porque la conclusión no depende de él), las formas aparecen así:
`periódicamente` 105 veces,
`periódico/a/os/as` 96, `a intervalos` 40, `de forma o manera continua` 18 y
`regular/regulares` 4. O sea que el adjetivo pesa casi tanto como el adverbio,
y quien busca solo el adverbio se deja fuera la mitad del vocabulario.

Y las 4 de `regular(es)` son la lección de verdad: es la forma más rara del
corpus **y una de esas cuatro es el art. 32.1.d del RGPD**, o sea la cadencia
de seguridad de la norma que alcanza a todos los clientes. La frecuencia de
una forma no dice nada sobre lo que cuesta no verla. `regular` además no
comparte raíz con `periodicidad`, así que no cae en ninguna búsqueda ingenua.

**Vocabulario mínimo, en español, que hay que pasar sobre todo el texto:**

| Familia | Formas |
|---|---|
| adverbio | `periódicamente`, `regularmente`, `anualmente`, `mensualmente`, `trimestralmente`, `semestralmente` |
| adjetivo | `periódico`, `periódica`, `periódicos`, `periódicas`, `regular`, `regulares`, `anual`, `anuales`, `bienal`, `trienal`, `mensual`, `trimestral`, `semestral` |
| sintagma de intervalo | `a intervalos planificados`, `a intervalos`, `con carácter periódico`, `con periodicidad`, `con regularidad`, `al menos una vez al año`, `cada N años`, `cada N meses`, `cada ejercicio` |
| continuidad | `de forma continua`, `de manera continua`, `proceso iterativo continuo`, `de manera activa y sistemática`, `continuado`, `continuada` |

**La regla que decide los casos de continuidad**, porque los cuatro últimos son
los que más ruido dan: cuenta como periodicidad la recurrencia que califica **al
acto obligado** (identificar de forma continua, mejorar de forma continua,
analizar de manera activa y sistemática). No cuenta la que califica **al formato
o a la disponibilidad de una prestación** (poner datos a disposición "de forma
continua y en tiempo real" describe cómo se accede, no cada cuánto hay que hacer
algo). Por esa regla quedan fuera los cuatro candidatos del Data Act y el art.
76.9 de MiCA, y quedan dentro DORA art. 13.3, ENS art. 27, AI Act art. 72.2 y el
anexo I del MDR.

**Y dos comprobaciones que no se pueden saltar**, porque este vocabulario tiene
mucho falso positivo:

1. Descartar por destinatario antes de contar. "Auditorías periódicas" del anexo
   VII del AI Act son del organismo notificado, no del proveedor.
2. Descartar el preámbulo y las definiciones. "Perfiles irregulares",
   "facturación periódica", "contactos regulares" y "actividad continuada" son
   los falsos positivos que se repiten, y ninguno es una cadencia.

## 2 ter. Las tres rejillas que la segunda pasada no tenía

La sección 2 bis existe porque la primera pasada se dejó fuera un vocabulario.
Esta existe por la misma razón y en el mismo sitio del método, pero el agujero no
era de palabras sino de **dónde se mira**. La segunda pasada leyó el articulado y
dio por hecho que el reloj vive ahí. Vive ahí y en otros tres sitios.

**Rejilla 1, los anexos.** En un reglamento del nuevo marco legislativo, los
plazos de conservación no están en el articulado, están en el anexo del módulo de
evaluación de la conformidad, repetidos módulo a módulo. El CRA, que esta tabla
daba por contado, tiene cinco apartados de conservación de diez años en el anexo
VIII (puntos 3.2, 4.2, 5.2, 6 y 10) y ninguno estaba en el censo. El MDR tiene
otros dos en los anexos XIII y XV. Regla: **un marco con anexos de módulos de
conformidad no está contado hasta que los anexos se han leído**.

**Rejilla 2, los actores distintos del principal.** Un reglamento de producto
obliga al fabricante, y también al importador, al distribuidor, al representante
autorizado y al agente económico, con plazos propios y distintos. La segunda
pasada contó al fabricante. El MDR obliga además al importador (art. 30.3, dos
semanas), al agente económico (art. 31.4, una semana) y al promotor de una
investigación clínica (arts. 75 y 77, cinco plazos). Regla: **por cada marco,
enumerar los sujetos obligados que define y recorrer los artículos de cada uno**.

**Rejilla 3, la retención documental.** Es la familia que menos se ve porque no
lleva verbo de urgencia: "tendrá a disposición de las autoridades durante un
período de diez años". La búsqueda que la caza no es de plazo, es de estas formas:
`conservar`, `conserven`, `mantendrá a disposición`, `tendrá a disposición`,
`durante al menos`, `durante un período de`, cruzadas con `años` o `meses`.
Pasada sobre los marcos ya contados, solo el AI Act la tenía completa.

**En qué dirección se equivoca esto.** Igual que la sección 2 bis: **por defecto**.
Las tres rejillas solo añaden. Ninguna de las tres quitó una obligación de
ninguna fila.

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
  relojes del paquete `iso27001` son rituales de plazum y no obligaciones de la
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

| # | Paquete | Estrato | Plazo | Periodicidad (núm.) | Evento | Total | Marca | Alcance para el comprador objetivo |
|---|---|---|---|---|---|---|---|---|
| 1 | nis2-tecnica | transcrito | 0 | 41 (3) | 20 | **61** | contado | alto, es la lista de control operativa de NIS2 |
| 2 | mica | transcrito | 21 | 20 (11) | 12 | **53** | contado | sectorial muy estrecho, pero denso |
| 3 | mdr | transcrito | 17 | 8 (4) | 14 | **39** | contado | sectorial estrecho |
| 4 | dora | transcrito | 4 | 21 (9) | 10 | **35** | contado | sectorial financiero, denso |
| 5 | psd2 | transcrito | 11 | 6 (5) | 10 | **27** | contado | sectorial, y en España vincula el RDL 19/2018 |
| 6 | ai-act | transcrito | 14 | 2 (0) | 10 | **26** | contado | transversal creciente |
| 7 | cra | transcrito | 15 | 1 (0) | 8 | **24** | contado (corregido) | alto para quien fabrica software |
| 8 | ens | transcrito | 2 | 8 (6) | 4 | **14** | contado | alto en España, ya construido |
| 9 | nis2-ue | transcrito | 5 | 1 (0) | 5 | **11** | contado | alto, pero es directiva sin transponer |
| 10 | rgpd | transcrito | 4 | 1 (0) | 6 | **11** | contado | máximo, alcanza a todos |
| 11 | ley2-2023 | transcrito | 6 | 0 | 3 | **9** | contado | alto, desde 50 empleados |
| 12 | dga | transcrito | 5 | 1 (1) | 3 | **9** | contado | muy bajo, población estrecha |
| 13 | csrd | transcrito | 3 | 5 (5) | 0 | **8** | contado | medio, y aplazado |
| 14 | lopdgdd | transcrito | 5 | 0 | 3 | **8** | contado | alto en España, complementa RGPD |
| 15 | eidas2 | transcrito | 3 | 2 (2) | 3 | **8** | contado | bajo salvo prestador de confianza |
| 16 | data-act | transcrito | 4 | 0 | 4 | **8** | contado | medio, sube desde 09-2025 |
| 17 | eni | transcrito | 0 | 3 (0) | 2 | **5** | contado (con las 12 NTI) | bajo, y el reloj no está en el real decreto |
| 18 | demo-empresa | propio | 1 | 2 (2) | 1 | **4** | contado (por construcción) | ninguno, es la demo; el reloj es sintético |
| 19 | iso27001 | referencial | 0 | 0 | 0 | **0** en la norma | contado (el 0) | máximo, ya construido con 6 rituales de plazum |
| 20 | iso27002 | referencial | ? | ? | ? | no verificado | no verificado | catálogo de 93 controles, sin cadencia propia |
| 21 | iso22301 | referencial | ? | ? | ? | no verificado | no verificado | estructura armonizada, cláusulas 4 a 10 |
| 22 | iso42001 | referencial | ? | ? | ? | no verificado | no verificado | estructura armonizada más anexo de controles |
| 23 | iso27701 | referencial | ? | ? | ? | no verificado | no verificado | extensión de 27001 y 27002 |
| 24 | soc2 | referencial | ? | ? | ? | no verificado | no verificado | criterios TSC, series CC1 a CC9 |
| 25 | pci-dss | referencial | ? | ? | ? | no verificado | no verificado | 12 requisitos, sí trae cadencias propias |
| 26 | tisax | referencial | ? | ? | ? | no verificado | no verificado | catálogo VDA ISA y vigencia de etiqueta |
| 27 | cis | delegado | 0 | 0 | 0 | **0** | contado | el reloj lo pone la herramienta |
| 28 | stig | delegado | 0 | 0 | 0 | **0** | contado | el reloj lo pone la herramienta |
| 29 | magerit | propio | 0 | 0 | 0 | **0** | contado | metodología, sin obligaciones |
| 30 | nist-800-53 | importado | n/a | n/a | n/a | sin autoría prevista | n/a | fuera por decisión D-1 |
| 31 | nist-csf | importado | n/a | n/a | n/a | sin autoría prevista | n/a | fuera por decisión D-1 |

**Totales de lo verificado**: 120 obligaciones con plazo explícito, 122 con
periodicidad explícita (de las cuales 48 con cadencia numérica y 74 con cadencia
declarada pero sin cuantificar), 118 con evento disparador explícito. Total 360
obligaciones con reloj en 18 paquetes, más un paquete (`iso27001`) cuyo cero está
contado y defendido.

Estos números son los de la **tercera pasada**, que hizo tres cosas: contó los
tres marcos que estaban en "estimado" (`mica`, `mdr`, `psd2`), abrió las tres
rejillas de la sección 2 ter sobre todos los marcos ya contados, y cerró la fila
que faltaba (`demo-empresa`) y la que estaba a cero por mirar el instrumento
equivocado (`eni`). La sección 6 bis dice qué cambió y en qué dirección, y
conviene leerla antes de fiarse de cualquier cifra de esta tabla.

**Paquetes en "no verificado"**: 7, todos de estrato referencial (iso27002,
iso22301, iso42001, iso27701, soc2, pci-dss, tisax). El motivo es el mismo en los
siete y está en la sección 3. Lo que haría falta: la copia licenciada del
cliente, leída dentro de su instancia, sin que el recuento salga del paquete
propio de ese cliente. **Estas siete filas no las ha tocado la tercera pasada**:
su "no verificado" no es un hueco de trabajo, es el resultado, y sigue siéndolo.

**Dos capas españolas censadas que no tienen paquete**, y que salen en la sección
5 porque cambian lo que hay que escribir en dos filas de esta tabla: el Real
Decreto-ley 19/2018 (transposición de PSD2, con **cuatro relojes que no coinciden
con los de la directiva**) y el Real Decreto-ley 12/2018 con el Real Decreto
43/2021 (NIS1 en España, **el único reloj de notificación de incidentes de red
que vincula hoy en España**, porque NIS2 sigue sin transponer).

## 5. Las citas, marco por marco

Cada línea es una cuenta con su artículo. Si un artículo no está aquí, no está
contado.

### ens (RD 311/2022, BOE-A-2022-7191, más las tres ITS) - contado

Contado contra el `paquete.json`, que ya tiene 132 obligaciones y 8 relojes con
24 casos dorados en verde, y comprobado contra el consolidado del BOE.

- **Plazo (2)**: disposición transitoria única, apartado 1 (adecuación en 24
  meses); ITS de Notificación de Incidentes, apartado IV.3 (notificación al CCN
  sin demora para impacto alto o superior, modelada como PT0H).
- **Periodicidad (8, seis con número)**: art. 31.1 (auditoría ordinaria al menos
  cada dos años); anexo I, apartado 1 (reevaluación anual de la categoría); ITS
  del Informe del Estado de la Seguridad, apartado III.2 (INES anual); ITS de
  Notificación de Incidentes, apartado VI (estadísticas anuales); ITS de
  Conformidad, apartado III.2 (autoevaluación bienal, categoría básica); ITS de
  Conformidad, apartado III.3 (certificación bienal, media y alta); art. 10.3
  (reevaluación periódica de las medidas, sin cuantificar); art. 27 (el proceso
  integral de seguridad se actualiza y mejora de forma continua, sin cuantificar,
  añadido en la segunda pasada).
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
- **Periodicidad (1, sin número)**: art. 32.1.d, que exige al responsable y al
  encargado (así arranca el 32.1) un proceso de verificación, evaluación y
  valoración **regulares** de la eficacia de las medidas técnicas y
  organizativas. Es cadencia declarada sin cuantificar, la misma categoría que
  el punto 3 del anexo I parte II del CRA.

  La primera pasada de este censo puso aquí un 0 y lo escribió en negrita. Era
  falso y el motivo está en la sección 6 bis. Lo que sí sigue en pie, y sigue
  siendo el argumento: **el RGPD no pone ni un número**. No dice cada cuánto es
  "regulares". El número lo pone la organización, y ahí es donde entra plazum, no
  transcribiendo una cadencia que no existe sino declarando y defendiendo la que
  el cliente elige. Las otras cinco periodicidades del texto sí son de
  autoridades (art. 41.2, 45.3, 57.1, 59, 70.1, 97.1) o del Comité, y siguen
  fuera de la cuenta. También queda fuera el art. 24.1 ("se revisarán y
  actualizarán cuando sea necesario"), que no es cadencia ni es evento definido.
- **Evento (6)**: art. 12.3 (solicitud del interesado); art. 19 (comunicación a
  cada destinatario tras rectificación o supresión); art. 33.1 (violación de
  seguridad, responsable); art. 33.2 (violación, encargado hacia responsable);
  art. 34.1 (alto riesgo, comunicación al interesado); art. 35.1, encadenado con
  el 36.1 (tratamiento de alto riesgo, evaluación de impacto y consulta previa).

### cra (Reglamento (UE) 2024/2847) - contado, corregido en la tercera pasada

Este marco estaba marcado "contado" y no lo estaba: le faltaban ocho plazos, y
los ocho son de la familia de retención documental, que es la rejilla 3 de la
sección 2 ter. Siete de ellos viven en el anexo VIII, que la segunda pasada no
abrió. Es el ejemplo que justifica la sección entera.

- **Plazo (15)**. Los siete de la segunda pasada: art. 13.8 (período de soporte de
  al menos cinco años); art. 14.2.a (alerta temprana de vulnerabilidad aprovechada
  activamente, 24 horas desde el conocimiento, ya transcrito); art. 14.2.b
  (notificación, 72 horas); art. 14.2.c (informe final, 14 días desde que se
  dispone de medida correctora o paliativa); art. 14.4.a (alerta temprana de
  incidente grave, 24 horas); art. 14.4.b (notificación, 72 horas); art. 14.4.c
  (informe final, un mes desde la notificación de la letra b).

  Los ocho que faltaban: art. 13.9 (cada actualización de seguridad sigue estando
  disponible un mínimo de diez años tras su publicación, o el resto del período de
  soporte si es más largo); art. 13.13 (documentación técnica y declaración UE a
  disposición de las autoridades de vigilancia del mercado, mínimo diez años desde
  la introducción en el mercado, o el período de soporte si es más largo); art.
  19.6 (lo mismo para el importador); y en el anexo VIII, los puntos 3.2, 4.2,
  5.2, 6 y 10, cada uno con el mismo período de diez años o de soporte, uno por
  módulo de evaluación de la conformidad. **Aviso para el autor**: los cinco
  puntos del anexo VIII son módulos alternativos, así que un fabricante concreto
  ve uno solo. Se cuentan cinco porque el paquete tiene que llevar los cinco, no
  porque nadie tenga cinco.

  Los tres de art. 13.9, 13.13 y 19.6 comparten una primitiva que el motor no
  ejercita todavía: **el máximo de dos duraciones**, una fija (diez años) y otra
  declarada por el propio obligado (el período de soporte del art. 13.8). No es
  una retención de diez años, es una de al menos diez años.
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

**Estado de la transposición española**: verificado el 26-08-2026 contra fuente
primaria, y **sigue sin transponer**. La segunda pasada dejó esto en "no
verificado" apoyándose en fuentes secundarias. La tercera lo ha consultado en el
índice de legislación consolidada del BOE, que es donde entra una ley desde su
publicación, con tres búsquedas: `titulo:ciberseguridad` devuelve tres normas y
ninguna es la transposición (Ley 15/2017 de la Agencia de Ciberseguridad de
Cataluña, Orden PCI/487/2019 de la Estrategia Nacional y Orden PRA/33/2018 del
Consejo Nacional de Ciberseguridad); `titulo:(seguridad AND redes AND sistemas)`
devuelve el Real Decreto-ley 12/2018 y el Real Decreto 43/2021, que son NIS1;
`titulo:2022/2555` no devuelve nada.

Límite del método, dicho para que se pueda repetir: la consulta mira legislación
**consolidada**. Una norma publicada y todavía no consolidada no saldría. La
comprobación sigue teniendo que rehacerse el día de escribir el paquete, pero
ahora se rehace con un comando y no con una búsqueda en prensa.

**Lo que sí vincula hoy en España, y no está en el corpus**: el Real Decreto-ley
12/2018 y su desarrollo, el Real Decreto 43/2021. Censados en la ficha siguiente.

### nis1-es (RDL 12/2018 y RD 43/2021) - contado (no es un paquete, es lo que vincula)

Aparece aquí porque el censo lo encontró buscando NIS2 y porque es el agujero más
grande que ha salido en las tres pasadas: **el único reloj de notificación de
incidentes de red y sistemas que obliga hoy a un operador de servicios esenciales
español no está en ningún paquete de `paquetes/`**.

- **El reloj bueno está en el anexo del RD 43/2021**, la "Instrucción nacional de
  notificación y gestión de ciberincidentes", tabla 3, ventana temporal de
  reporte. Es una notificación escalonada de tres hitos con límites por nivel de
  peligrosidad o impacto: nivel CRÍTICO, notificación inicial inmediata,
  intermedia a las 24 o 48 horas, final a los 20 días; nivel MUY ALTO, inicial
  inmediata, intermedia a las 72 horas, final a los 40 días; nivel ALTO, solo
  inicial inmediata. Los tiempos de la intermedia y la final se cuentan **desde la
  remisión de la notificación inicial**, no desde el incidente, y el propio anexo
  lo dice.
- **Plazo (8 relojes)**: los cinco de la tabla 3, más el art. 7.1 del RD 43/2021
  (designación del responsable de la seguridad de la información, con su plazo),
  el art. 7.2 (comunicación de nombramientos y ceses en el plazo de un mes) y el
  art. 7 del RDL 12/2018 (los proveedores de servicios digitales comunican su
  actividad a la autoridad en el plazo de tres meses desde que la inician).
- **Periodicidad (1, con número)**: art. 6 del RDL 12/2018, actualización bienal
  de la relación de servicios esenciales y de sus operadores. Es de la autoridad,
  pero se anota porque es la que reabre la aplicabilidad del paquete entero.
- **Por qué importa para el plan**: el nivel de peligrosidad es un dato que
  clasifica el propio obligado, y de esa clasificación cuelga qué relojes se
  encienden. Es el primer caso del corpus donde **el disparador no es un hecho
  sino una categoría que el cliente asigna**, y el motor tiene que poder decir de
  dónde salió esa categoría. Escribirlo antes que NIS2 tiene además la ventaja de
  que no caduca: cuando salga la transposición, el escalonado de tres hitos ya
  estará construido y solo cambiarán los límites.

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

Lo que hace a este marco excepcional para plazum: 41 cadencias y 20 disparadores
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
- **Periodicidad (21, nueve con número)**: con número, art. 6.5 (documentación y
  revisión del marco al menos una vez al año), 8.1 (revisión anual de la
  clasificación de funciones y activos), 8.2 (revisión anual de los escenarios de
  riesgo), 8.7 (evaluación anual del riesgo en sistemas heredados), 11.6 (pruebas
  anuales de los planes de continuidad y de respuesta y recuperación), 13.5
  (informe anual del directivo de TIC al órgano de dirección), 24.6 (pruebas
  anuales de los sistemas que sustentan funciones esenciales), 26.1 (pruebas de
  penetración basadas en amenazas al menos cada tres años), 28.3 (comunicación
  anual a la autoridad sobre los acuerdos de servicios de TIC). Sin cuantificar,
  art. 5.2, 5.4, 6.6, 8.6, 10.1 (pruebas periódicas de los mecanismos de
  detección), 11.4, 12.2, 13.3 (incorporación continua de los hallazgos al
  proceso de evaluación del riesgo), 16.1.g (pruebas periódicas de los planes en
  el marco simplificado), 16.2, 28.2 y 28.8. Los tres subrayados, 10.1, 13.3 y
  16.1.g, los añadió la segunda pasada. Corregido durante el
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
- **Periodicidad (2, ninguna con número)**: art. 9.2 (el sistema de gestión de
  riesgos del proveedor es un proceso iterativo continuo que requerirá
  "revisiones y actualizaciones sistemáticas periódicas"); art. 72.2 (el sistema
  de vigilancia poscomercialización recopilará, documentará y analizará "de
  manera activa y sistemática" los datos pertinentes). Las dos son del proveedor
  y las dos se escaparon en la primera pasada, que puso aquí un 0.

  Lo que sobrevive de aquella afirmación, corregido: **el AI Act no fija ni una
  cadencia numérica al proveedor ni al responsable del despliegue**. El plan de
  vigilancia poscomercialización lo fija un acto de ejecución. Las auditorías
  periódicas del anexo VII, punto 5.3, son del organismo notificado y quedan
  fuera de la cuenta.
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
- **Periodicidad (0)**, y aquí la segunda pasada tuvo que decidir una regla. El
  texto dice "de forma continua y en tiempo real" cuatro veces (art. 3.2, 4.1,
  5.1 y 33.1), pero en las cuatro esa continuidad califica **el formato y la
  disponibilidad de los datos**, no cada cuánto hay que ejecutar un acto. La
  regla aplicada, y aplicada igual en todo el censo: cuenta como periodicidad la
  recurrencia que califica al acto obligado, no la que califica a la prestación.
  Por la misma regla queda fuera el art. 76.9 de MiCA.
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
- **Periodicidad (0)**, comprobado en la segunda pasada con el vocabulario
  ampliado: los dos únicos candidatos son falsos positivos, "perfiles
  irregulares" en el preámbulo y "facturación periódica" como tipo de contrato
  en el art. 20.
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
- **Periodicidad (0)**, y este 0 sí resiste la segunda pasada: **la ley no obliga
  a la entidad a revisar su sistema interno de información con ninguna
  cadencia**. Rebuscado con el vocabulario ampliado de la sección 2 bis, las dos
  únicas revisiones periódicas de la ley son la trienal del art. 22 y la del
  art. 68 sobre canales externos, y las dos son de las autoridades, no de la
  empresa. El único "periódica" del preámbulo se refiere a esas mismas
  autoridades.
- **Evento (3)**: recepción de una comunicación (dispara 9.2.c y 9.2.d);
  nombramiento o cese del Responsable del Sistema (8.3); ausencia de actuaciones
  de investigación a los tres meses (32.4).

### eni (RD 4/2010 más las 12 Normas Técnicas de Interoperabilidad) - contado

La segunda pasada dejó este marco a cero y anotó que faltaban las NTI. La tercera
las ha censado, las doce, y **el cero era del real decreto, no del marco**. La
apuesta que hacía aquella nota (si hay reloj, está en las NTI) era la correcta.

- **Plazo (0)**. El real decreto sigue sin plazo vivo: sus dos únicos plazos son
  transitorios y están vencidos hace años, el plan de adecuación con ejecución no
  superior a 48 meses desde la entrada en vigor y los 24 meses que remiten a la
  disposición transitoria primera del RD 1671/2009. Las doce NTI no añaden
  ninguno: el único candidato, en la NTI de Modelo de Datos para el Intercambio de
  asientos (BOE-A-2011-13174), es el año de adaptación desde el 10-08-2021 a la
  versión nueva, también vencido.
- **Periodicidad (3, ninguna con número)**: NTI de Política de gestión de
  documentos electrónicos (Res. de 28-06-2012, BOE-A-2012-10048), apartado X.2
  (las organizaciones realizan evaluaciones o auditorías periódicas documentadas
  de la política de gestión documental) y apartado V.2 (el programa de tratamiento
  se aplica de manera continua sobre todas las etapas del ciclo de vida, que entra
  por la regla de continuidad de la sección 2 bis porque califica al acto
  obligado); NTI de Protocolos de intermediación de datos (BOE-A-2012-10049),
  apartado II.b.3 (auditorías periódicas sobre el uso del sistema de consulta de
  datos). Queda fuera por destinatario la actualización anual del Catálogo de
  estándares (BOE-A-2012-13501, apartado V.1): la hace el órgano que mantiene el
  catálogo, no la organización obligada.
- **Evento (2)**: NTI de Documento Electrónico (BOE-A-2011-13169), apartado VII.5,
  y NTI de Expediente Electrónico (BOE-A-2011-13170), apartado V.6. Los dos tienen
  el mismo disparador, el intercambio entre Administraciones que supone
  transferencia de custodia o traspaso de responsabilidad sobre documentos de
  conservación permanente, y la misma obligación, que el órgano transferidor
  verifique autenticidad e integridad en el momento del intercambio.
- **Lo que esto cambia**: la recomendación de la segunda pasada era "no se escribe
  nunca si las NTI tampoco tienen reloj". Sí lo tienen, aunque poco y sin número,
  así que `eni` deja de ser un cero y pasa a ser un marco pequeño de la familia B
  (cadencia sin cuantificar que plazum propone y el cliente defiende). Sigue sin
  ser prioritario: son tres cadencias y dos eventos para el sector público.
- **Aviso de identidad**: el paquete `eni` apunta al RD 4/2010 y las NTI son
  resoluciones con identificador propio del BOE. Si se escriben sus relojes, la
  `fuente` de cada obligación es la resolución, no el real decreto, y ahí hay una
  decisión de `urn` de la misma familia que la de `eidas2` y `csrd`.

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
  ha transpuesto**, y las fechas de aplicación las movió dos años la Directiva (UE)
  2025/794, la llamada "stop the clock". Escribir hoy los relojes de csrd es
  escribir relojes que van a cambiar de fecha.
- **La transposición, verificada el 26-08-2026 contra fuente primaria**, no contra
  prensa: en el índice de legislación consolidada del BOE, `titulo:sostenibilidad`
  devuelve veinte normas y ninguna es la Ley de Sostenibilidad Empresarial (son de
  sostenibilidad financiera, ambiental, turística y del sistema de pensiones), y
  `titulo:2022/2464` no devuelve nada. Mismo límite que en NIS2: la consulta mira
  legislación consolidada, así que una publicación reciente sin consolidar no
  saldría.

### mdr (Reglamento (UE) 2017/745) - contado en la tercera pasada

388 apartados candidatos sobre el texto entero, anexos incluidos. La segunda
pasada verificó el núcleo de vigilancia poscomercialización y conservación del
fabricante y extrapoló el resto; el resto no se parecía a la extrapolación. De 16
obligaciones estimadas se pasa a 39 contadas, y **la diferencia entera está en las
dos rejillas que faltaban**: el promotor de investigaciones clínicas (arts. 75, 77
y 80) y los anexos (VI, XIII y XV).

- **Plazo (17 apartados, 20 relojes)**: art. 10.8 (documentación técnica y
  declaración UE a disposición al menos diez años desde la última introducción en
  el mercado, y al menos quince para implantables, que son dos relojes); art. 30.3
  (el importador comprueba en las dos semanas siguientes a la introducción en el
  mercado que el fabricante transmitió los datos al sistema electrónico); art.
  31.4 (el agente económico actualiza los datos en el plazo de una semana desde la
  modificación); art. 31.5 (seis meses de gracia antes de la suspensión del
  certificado si no confirma la exactitud); art. 56.2 (validez del certificado,
  tope de cinco años prorrogables por períodos de hasta cinco, ventana que el
  fabricante vigila); art. 75.1 (el promotor comunica la modificación sustancial
  de una investigación clínica en el plazo de una semana); art. 75.3 (y no puede
  aplicarla hasta 38 días después de esa notificación); art. 77.1 (paralización
  temporal o finalización anticipada, quince días); art. 77.3 (finalización en un
  Estado miembro, quince días desde el cese); art. 77.4 (finalización definitiva
  en todos, quince días); art. 77.5 (informe de la investigación clínica, un año
  desde la finalización o tres meses desde la finalización anticipada, que son dos
  relojes); art. 87.3 (incidente grave, a más tardar quince días desde el
  conocimiento); art. 87.4 (amenaza grave para la salud pública, a más tardar dos
  días); art. 87.5 (fallecimiento o deterioro grave imprevisto, a más tardar diez
  días); anexo VI, punto 5.8 (cambio que no exige UDI-DI nuevo, treinta días para
  actualizar la base de datos UDI); anexo XIII, sección 4 (declaración de producto
  a medida, diez años desde la introducción en el mercado, quince para
  implantables, dos relojes); anexo XV, capítulo III, sección 3 (documentación de
  la investigación clínica, diez años desde su fin).

  No se cuentan aparte, porque remiten al reloj de otro artículo y no crean uno
  nuevo: art. 11.3.c y art. 13.9 (representante autorizado e importador, que
  conservan "durante el período del art. 10, apartado 8") y art. 87.7 (la duda
  sobre si un incidente es notificable no abre plazo, obliga a usar el del
  apartado 3, 4 o 5). Son obligaciones distintas con el mismo reloj, y el paquete
  las llevará; el censo cuenta relojes.
- **Periodicidad (8, cuatro con número)**: art. 31.5 (confirmación de la exactitud
  de los datos un año después de la primera presentación y después cada dos años);
  art. 61.11, párrafo segundo (informe de evaluación del seguimiento clínico
  poscomercialización y resumen de seguridad y funcionamiento clínico, al menos
  una vez al año para clase III e implantables); art. 86.1 (PSUR actualizado como
  mínimo una vez cada año para clases IIb y III); art. 86.1 (PSUR como mínimo cada
  dos años para clase IIa). Sin cuantificar: art. 83.2 (el sistema de seguimiento
  poscomercialización recaba, conserva y analiza "activa y sistemáticamente"
  durante todo el ciclo de vida, añadido en la tercera pasada); art. 85 (informe
  de seguimiento poscomercialización de clase I); anexo I, capítulo I, punto 3 (la
  gestión de riesgos es un proceso iterativo continuo con actualizaciones
  sistemáticas periódicas); anexo VI, punto 5.4 (los fabricantes verifican
  periódicamente la exactitud de los datos de los productos comercializados,
  añadido en la tercera pasada). Sigue fuera el art. 87.9, porque los informes
  resumidos periódicos son una opción del fabricante acordada con la autoridad, no
  una obligación.
- **Evento (14)**: art. 10.12, párrafo primero (el fabricante considera o tiene
  motivos para creer que un producto introducido no es conforme); art. 10.12,
  párrafo segundo (el producto presenta un riesgo grave, información inmediata a
  las autoridades); art. 13.7 (lo mismo para el importador); art. 14.4 (lo mismo
  para el distribuidor); art. 14.5 (el distribuidor recibe una reclamación o un
  informe sobre un supuesto incidente); art. 16.1 (el distribuidor o el importador
  realiza una de las actividades que le hacen asumir las obligaciones del
  fabricante); art. 30.3 (introducción en el mercado, que abre las dos semanas del
  importador); art. 31.4 (modificación de la información registrada); art. 75.1
  (modificación sustancial de una investigación clínica); art. 77.1 (paralización
  temporal o finalización anticipada); art. 80.2 (acontecimiento adverso grave en
  una investigación clínica); art. 87.1.a (incidente grave); art. 87.1.b (acción
  correctiva de seguridad sobre el terreno); art. 88.1 (aumento estadísticamente
  significativo de la frecuencia o gravedad de incidentes no graves).

### mica (Reglamento (UE) 2023/1114) - contado en la tercera pasada

522 apartados candidatos, y la proporción de plazos de autoridad más alta del
censo: MiCA es sobre todo un reglamento de procedimiento de autorización. Aun así,
de 19 obligaciones estimadas se pasa a 53 contadas, y MiCA sube de la posición
cuatro a la dos de la tabla.

Este marco trae un regalo que ningún otro tiene y que conviene decir porque cambia
el coste de escribirlo: **los anexos V y VI son un catálogo de infracciones que
enumera, una por una y por artículo, las obligaciones del emisor**. Sirven de
control cruzado del censo sin leer el articulado dos veces, y confirmaron seis de
las periodicidades nuevas.

- **Plazo (21 apartados, 23 relojes)**: art. 9.1 (libro blanco publicado con
  antelación razonable y a más tardar en la fecha de inicio de la oferta o de la
  admisión a negociación, hito sin cifra); art. 10.1 (resultado de la oferta
  pública, 20 días hábiles desde el final del plazo de suscripción); art. 12.9
  (versiones anteriores del libro blanco a disposición del público al menos diez
  años); art. 13.1 (14 días naturales de desistimiento del titular minorista,
  ventana que el oferente vigila); art. 13.2 (reembolso a más tardar 14 días desde
  que se le informa del desistimiento); art. 14.3 (reembolso por cancelación de la
  oferta, a más tardar 25 días naturales); art. 28 (libro blanco aprobado
  publicado a más tardar en la fecha de inicio de la oferta, confirmado por el
  anexo V punto 10); art. 36.10 (resultado de la auditoría de la reserva, a más
  tardar seis semanas desde la fecha de referencia de la valoración); art. 37.3
  (activos de reserva en custodia a más tardar cinco días hábiles desde la
  emisión); art. 46.2 (plan de recuperación, seis meses desde la autorización);
  art. 47.3 (plan de reembolso, seis meses); art. 55 (las dos variantes anteriores
  para emisores de fichas de dinero electrónico, seis meses desde la oferta
  pública); art. 65.4 (inicio de actividad en otro Estado miembro, no antes del
  decimoquinto día natural desde la presentación de la información); art. 67.4.a
  (seguro con duración inicial no inferior a un año); art. 67.4.b (preaviso de
  cancelación del seguro, al menos 90 días); art. 68.9 (registros conservados
  cinco años, ampliables a siete si la autoridad lo pide antes de que venzan, que
  son dos relojes); art. 76.11 (información de mercado gratuita quince minutos
  después de su publicación); art. 76.12 (liquidación final en las 24 horas
  posteriores a la ejecución, o a más tardar al cierre del día si se liquida fuera
  del registro distribuido, que son dos relojes); art. 76.15 (datos de todas las
  órdenes a disposición de la autoridad al menos cinco años); art. 85.2
  (notificación al alcanzar el umbral de usuarios activos, dos meses); art.
  143.2.b (libro blanco de los criptoactivos ya admitidos a negociación, a más
  tardar el 31-12-2027).

  Quedan fuera de la cuenta, y se dice porque el cliente los va a preguntar, los
  dos "plazo razonable" del texto: art. 31.4 (comunicar al titular el resultado de
  la investigación de su reclamación) y art. 71.4 (lo mismo para el cliente de un
  proveedor de servicios). Son la cuarta categoría de la sección 1, no plazos.
- **Periodicidad (20, once con número)**. Con número: art. 10.2 (publicación al
  menos mensual del número de unidades en circulación por los oferentes sin plazo
  de oferta); art. 22.1 (comunicación trimestral a la autoridad para emisores de
  fichas referenciadas a activos con valor de emisión superior a 100 millones de
  euros); art. 30.1 (actualización al menos mensual de la cantidad en circulación y
  de la reserva de activos); art. 35.1, párrafo segundo (revisión anual del importe
  de fondos propios); art. 36.9 (**auditoría independiente de la reserva de
  activos cada seis meses** desde la autorización o desde la oferta pública, la
  cadencia semestral más importante del corpus, y la segunda pasada no la vio);
  art. 58 (la misma auditoría semestral para emisores de fichas significativas de
  dinero electrónico, contada desde la decisión de clasificación); art. 67.1.b
  (gastos fijos generales del año anterior, revisados anualmente); art. 72.4
  (revisión al menos anual de la política de conflictos de intereses); art. 75.5
  (estado de posición al cliente de custodia, al menos una vez cada tres meses);
  art. 81.12 (revisión de la evaluación de idoneidad de cada cliente al menos cada
  dos años); art. 81.14 (estado de cuentas de gestión de cartera, cada tres
  meses). Sin cuantificar: art. 34.3 (el órgano de dirección del emisor evalúa y
  revisa periódicamente la eficacia de las medidas, confirmado por el anexo V punto
  29); art. 34.12 (auditoría periódica por auditores independientes); art. 35.5
  (pruebas de resistencia periódicas, confirmado por el anexo V punto 43); art.
  37.5 (revisión periódica de la designación de los custodios de la reserva); art.
  46.2 (revisión o actualización periódica del plan de recuperación, anexo V punto
  80); art. 47.3 (lo mismo del plan de reembolso, anexo V punto 86); art. 68.6 (el
  órgano de dirección del proveedor de servicios evalúa y revisa periódicamente);
  art. 68.8 (control y evaluación periódica de la adecuación de los mecanismos de
  gestión de riesgos); art. 78.6 (comprobación periódica de si los centros de
  ejecución siguen dando el mejor resultado posible al cliente).
- **Evento (12)**: art. 12.1 (hecho nuevo significativo, error material o
  inexactitud material, que obliga a modificar el libro blanco); art. 12.2 (libro
  blanco modificado, que obliga a notificarlo a la autoridad); art. 13.1
  (desistimiento del titular minorista); art. 14.3 (cancelación de la oferta
  pública); art. 23.1 (superación del umbral de un millón de operaciones diarias y
  200 millones de euros como medio de cambio, que obliga a cesar la emisión y a
  presentar un plan); art. 25.1 (cambio previsto en el modelo de negocio); art.
  31.4 (reclamación de un titular de fichas); art. 43 con 56 (clasificación de la
  ficha como significativa, que enciende entre otras la auditoría semestral del
  art. 58); art. 65.1 (intención de prestar servicios en otro Estado miembro);
  art. 71.4 (reclamación de un cliente de servicios de criptoactivos); art. 85.2
  (alcanzar los quince millones de usuarios activos de media anual); art. 88.1
  (información privilegiada, que obliga a hacerla pública lo antes posible).

### psd2 (Directiva (UE) 2015/2366) - contado en la tercera pasada

200 apartados candidatos, revisados uno a uno. De 11 obligaciones estimadas se
pasa a 27 contadas. Lo que faltaba: la retención del título II (rejilla 3), los
plazos de preaviso del contrato marco y las cadencias de información al usuario.

- **Plazo (11 apartados, 13 relojes)**: art. 18.4.b (el crédito concedido en
  relación con el pago se reembolsa en un plazo corto que en ningún caso supera
  los doce meses); art. 21 (la entidad de pago conserva los documentos del título
  II durante cinco años como mínimo); art. 54.1 (modificación de las condiciones
  del contrato marco, con antelación no inferior a dos meses); art. 55.3
  (rescisión del contrato marco por el proveedor, preaviso mínimo de dos meses);
  art. 71.1 (el usuario dispone de trece meses para notificar, ventana que el
  proveedor tiene que vigilar); art. 73.1 (devolución del importe de la operación
  no autorizada a más tardar al final del día hábil siguiente); art. 77.2
  (devolución solicitada, diez días hábiles desde la recepción de la solicitud);
  art. 82.2 (plazo pactado más largo para operaciones no cubiertas por el apartado
  1, tope de cuatro días hábiles); art. 83.1 (abono en la cuenta del proveedor del
  beneficiario a más tardar al final del día hábil siguiente, prorrogable un día
  hábil para las órdenes iniciadas en papel, que son dos relojes); art. 87.1
  (fecha de valor del abono no posterior al día hábil en que se abonó en la cuenta
  del proveedor del beneficiario); art. 101.2 (respuesta a reclamaciones a más
  tardar quince días hábiles, y como máximo treinta y cinco en situaciones
  excepcionales, que son dos relojes).

  No se cuenta el art. 84, que remite al plazo del art. 83 para el beneficiario
  sin cuenta: es otra obligación con el mismo reloj.
- **Periodicidad (6, cinco con número)**: art. 5.1.l (controles sobre agentes y
  sucursales que el solicitante se compromete a realizar como mínimo una vez al
  año); art. 17.2 (auditoría de las cuentas anuales y consolidadas de la entidad
  de pago); art. 57.2 (información sobre cada operación al ordenante, de manera
  periódica y al menos una vez al mes, cuando el ordenante lo exija); art. 95.2
  (evaluación actualizada y completa de los riesgos operativos y de seguridad,
  anualmente o a intervalos más breves si la autoridad lo determina); art. 96.6
  (datos estadísticos sobre fraude, por lo menos una vez al año). Sin cuantificar:
  art. 5.1.h (procedimiento para poner a prueba y revisar periódicamente la
  adecuación y eficiencia de los planes de contingencia).

  Los dos del art. 5.1 se declaran en la solicitud de autorización, no en un
  artículo de obligaciones, y se cuentan porque vinculan a la entidad autorizada
  desde ese momento. Se dice de dónde salen para que el autor pueda discrepar. Y
  queda fuera el art. 58.2, que es la misma cadencia mensual del 57.2 hacia el
  beneficiario pero en cláusula potestativa ("podrán contener"), no obligatoria.
- **Evento (10)**: art. 19.1 (intención de recurrir a un agente); art. 28.1
  (intención de ejercer el derecho de establecimiento o la libre prestación en
  otro Estado miembro); art. 68.3 (bloqueo del instrumento de pago, que obliga a
  informar al ordenante); art. 68.4 (cesan los motivos del bloqueo, que obliga a
  desbloquear o sustituir); art. 68.5 (denegación de acceso a la cuenta a un
  proveedor tercero, que obliga a informar al ordenante); art. 71 con 73
  (operación no autorizada o mal ejecutada, notificada por el usuario); art. 76
  con 77 (solicitud de devolución); art. 96.1, párrafo primero (incidente
  operativo o de seguridad grave, notificación a la autoridad); art. 96.1, párrafo
  segundo (incidente que afecta o puede afectar a los intereses financieros de los
  usuarios, información a estos); art. 101.2 (reclamación del usuario).
- **Aviso, ahora resuelto**: en España lo que vincula es el Real Decreto-ley
  19/2018, no la directiva. Ya está censado, en la ficha siguiente, y **no
  coincide**.

### rdl19-2018, la capa española de psd2 - contado (no es un paquete, es lo que vincula)

Censado sobre el consolidado del BOE (BOE-A-2018-16036). No tiene fila en la
tabla porque no hay directorio en `paquetes/`; está aquí porque **quien escriba el
paquete `psd2` con las cifras de la directiva escribirá cuatro relojes
equivocados para España**.

Las cuatro divergencias, que son el hallazgo:

| Materia | Directiva 2015/2366 | RDL 19/2018 |
|---|---|---|
| Conservación de documentos | cinco años (art. 21) | **seis años** (art. 24) |
| Respuesta definitiva a una reclamación | treinta y cinco días hábiles (art. 101.2) | **un mes** (art. 69.2) |
| Resolución del contrato marco por el usuario | preaviso pactable de hasta un mes (art. 55.1) | **sin preaviso, y el proveedor cumple la orden antes de 24 horas** (art. 32.1) |
| Superar el umbral de exención | no existe | **30 días naturales para solicitar la autorización** (art. 14.2.d) |

El art. 32.1 es el que más pesa: es un reloj de veinticuatro horas, de la familia
más cara de fallar, que **no existe en la directiva**. Un pipeline de autoría que
lea el nivel europeo y dé el nacional por equivalente no lo escribe y no se
entera, que es exactamente el hallazgo 1 de la sección 6 en otra dirección.

El resto del RDL sí traslada la directiva con las mismas cifras: art. 20.b (doce
meses del crédito), art. 33.1 (dos meses de preaviso de modificación), art. 43.1
(trece meses), art. 45.1 (final del día hábil siguiente), art. 49.2 (diez días
hábiles), art. 54.2 (cuatro días hábiles), art. 55.1 (final del día hábil
siguiente, más uno para papel), art. 58.1 (fecha de valor), art. 69.1 (quince días
hábiles). Y las dos cadencias del art. 66.2 y del art. 67.4 son las del art. 95.2
y 96.6 de la directiva, con una diferencia de redacción que importa: las dos dicen
"con la periodicidad y forma que el Banco de España determine, **al menos** una
vez al año", así que el número por defecto de plazum tiene un suelo legal y un
techo administrativo.

### demo-empresa - contado por construcción, y es el trigésimo primer paquete

Esta es la fila que faltaba. `demo-empresa` no es una norma, es el paquete de
datos sintéticos de la demostración, pero está en `paquetes/`, carga con el
linter, tiene siete obligaciones y tres de ellas llevan reloj. Dejarla fuera del
censo era exactamente el error que un censo no puede cometer: contar lo que se
tenía en la cabeza en vez de contar lo que hay en el disco.

No hay artículo que citar porque no hay norma. La cita es el propio
`paquetes/demo-empresa/paquete.json`, y por eso la marca dice "contado por
construcción": aquí el recuento no se deriva de un texto legal, se lee del bloque
`temporalidad` de cada obligación, que es la fuente.

- **Plazo (1)**: `demo.notificacion_de_incidente`, PT72H con cómputo naturales,
  cierre exacto y sin traslado, disparada por `deteccion_del_incidente`, con dos
  escalados internos a PT24H y PT48H. Es la copia sintética de la familia A.
- **Periodicidad (2, las dos con número)**:
  `demo.revision_trimestral_de_accesos`, cadencia P3M anclada en
  `ultima_revision_de_accesos`; `demo.auditoria_bienal`, cadencia P24M anclada en
  `ultima_auditoria`. Son las copias sintéticas de las familias B y C.
- **Evento (1)**: la detección del incidente, que es lo que enciende el plazo
  anterior. Los dos anclajes de las cadencias no se cuentan como evento: son el
  hito de la última ocurrencia, no un hecho que hace nacer la obligación.
- **Para qué sirve en este documento**: las cuatro familias de reloj que el motor
  ya ejecuta de verdad están representadas aquí en miniatura, así que este paquete
  es el banco de pruebas de cualquier primitiva nueva antes de gastarla en un
  marco real. Las otras cuatro obligaciones, las que no llevan reloj
  (`demo.politica_de_seguridad`, `demo.inventario_de_activos`,
  `demo.clausulas_del_contrato_publico` y `demo.plan_de_continuidad`), están ahí
  para lo contrario, para enseñar que una obligación sin reloj también se muestra.

### Los ocho referenciales - no verificado, con lo que sí se sabe

- `iso27001`: **0 cadencias numéricas en la norma**, hecho ya publicado por el
  proyecto en `paquetes/iso27001/RITUALES.md`. Los 6 relojes del paquete (cinco
  cadencias de 12 o 24 meses y un plazo de 10 días hábiles) son rituales de plazum
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
  a esa decisión el número que le faltaba: de las 360 obligaciones con reloj
  contadas aquí, **ninguna se podría representar en OSCAL sin perder el reloj**.

## 6. Los nueve hallazgos que cambian el plan de autoría

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

3. **Los marcos transversales imponen cadencia, pero sin número.** RGPD art.
   32.1.d exige un proceso de verificación "regulares"; AI Act art. 9.2 exige
   "revisiones y actualizaciones sistemáticas periódicas". Ninguno dice cada
   cuánto. Y dos marcos sí quedan a cero de verdad, comprobado dos veces: Ley
   2/2023 y LOPDGDD no imponen ninguna cadencia al obligado. Argumento de venta,
   corregido y ahora más fuerte: la "revisión anual del RGPD" que el cliente
   tiene en su plan no es que sobre, es que **el número es suyo y tiene que poder
   defenderlo**. Plazum es donde ese número se declara, se justifica por escrito y
   se prueba con un caso dorado.

4. **La densidad de reloj no correlaciona con el tamaño del texto.** El
   Reglamento de Ejecución 2024/2690 tiene 16 artículos y 61 obligaciones con
   reloj. MiCA tiene 149 artículos y 53, con un texto seis veces más largo. El
   argumento sobrevive a la tercera pasada pero **más débil de lo que se escribió**:
   con los números de la segunda (61 contra 19) parecía que MiCA no valía la pena,
   y con los de la tercera (61 contra 53) lo que se ve es que 2024/2690 cuesta seis
   veces menos por reloj, que es otra cosa. La conclusión operativa no cambia,
   2024/2690 va primero; el motivo sí, y era importante corregirlo porque el motivo
   es lo que se reutiliza para el marco siguiente.

5. **El corpus entero tiene 48 cadencias con número y 74 sin cuantificar**, casi
   dos tercios sin número. Eso confirma que el patrón de `iso27001` (plazum pone un
   valor de partida razonable, lo justifica por escrito en `RITUALES.md` o
   `COMPUTO.md`, y el cliente lo cambia en su copia) no es una excepción del
   estrato referencial: es el modo por defecto de casi dos tercios del corpus,
   incluido el transcrito. Y es también, por lo mismo, la categoría donde este
   censo se equivoca cuando se equivoca (sección 6 bis). La proporción se ha
   mantenido casi exacta al pasar de 99 a 122 periodicidades, lo que dice que no
   es un artefacto del muestreo.

6. **La transposición nacional no es el mismo reloj con otra bandera.** El RDL
   19/2018 traslada PSD2 y cambia cuatro cifras, una de ellas inventando un plazo
   de veinticuatro horas que la directiva no tiene (art. 32.1, cumplir la orden de
   resolución del contrato marco). Consecuencia, y es más fuerte que la del
   hallazgo 2: no basta con resolver **qué instrumento** vincula, hay que censar el
   instrumento nacional entero, porque el europeo no predice sus números.

7. **El reloj que vincula hoy en España para incidentes de red no está en el
   corpus.** NIS2 sigue sin transponer, comprobado el 26-08-2026 contra el BOE. Lo
   que obliga es NIS1, el RDL 12/2018 con el RD 43/2021, y su anexo trae una
   notificación escalonada completa por nivel de peligrosidad (inmediata, 24 o 48
   horas, 20 días para CRÍTICO; inmediata, 72 horas, 40 días para MUY ALTO).
   Consecuencia: hay un paquete que falta, y es de la familia que se escribe
   primero.

8. **Un marco marcado "contado" puede no estarlo si tiene anexos.** El CRA llevaba
   dos pasadas marcado contado y le faltaban ocho plazos de retención, siete de
   ellos en el anexo VIII. Consecuencia: la marca "contado" solo vale con la fecha
   de la pasada que la puso, y por eso la tabla ahora dice de qué pasada es cada
   número. La rejilla que lo caza está en la sección 2 ter.

9. **"Estimado" resultó ser un eufemismo de "sin contar".** Los tres marcos
   estimados subestimaban por un factor de entre 2,4 y 2,8: mica de 19 a 53, mdr de
   16 a 39, psd2 de 11 a 27. Ninguno se movió a la baja. Consecuencia para el
   método, no para el plan: **la marca "estimado" no se vuelve a usar**. O se
   cuenta, o se dice "no verificado" con el motivo. Una cifra estimada ocupa el
   sitio de una contada y nadie vuelve a mirarla, que es justo lo que pasó aquí
   durante dos pasadas.

## 6 bis. El fallo de la primera pasada, y en qué dirección

Este censo se va a usar para decidir el orden de autoría, así que quien lo lea
tiene derecho a saber qué clase de error contiene y hacia dónde se equivoca.

**Qué pasó.** La primera pasada declaró, en negrita, que el RGPD no impone
ninguna cadencia al responsable ni al encargado. Es falso. El art. 32.1.d exige
"un proceso de verificación, evaluación y valoración **regulares** de la eficacia
de las medidas", y el art. 32.1 se abre nombrando al responsable y al encargado.
El mismo error estaba en el AI Act: art. 9.2, "revisiones y actualizaciones
sistemáticas periódicas", dirigido al proveedor.

**Por qué se escapó.** El juego de marcadores de periodicidad tenía el adverbio
`periodicamente` y no tenía ni las formas adjetivas (`periódicos`, `periódicas`)
ni `regular(es)`. Los dos artículos usan justo esas formas. No fue un error de
criterio, fue un agujero de vocabulario, y produjo el peor tipo de resultado:
un cero, que es la afirmación más fuerte que un censo puede hacer, y por eso la
que más caro sale si es falsa.

**En qué dirección se equivoca este censo.** Siempre por defecto, nunca por
exceso, y siempre en la misma categoría: **la cadencia declarada sin número**.
Un plazo con cifra ("72 horas", "cada dos años") es difícil de no ver. Una
cadencia sin cifra es una palabra suelta en mitad de una frase larga. Y esa
categoría es 74 de las 122 periodicidades del censo, casi dos tercios. Si alguien
encuentra un error nuevo aquí, la apuesta razonable es que sea otro de estos, y
que el número correcto sea mayor que el escrito.

La tercera pasada confirmó la dirección y **amplió la predicción**: la cadencia
sin cifra ya no es la única familia que se escapa. La retención documental se
escapa igual, y por el mismo motivo, porque tampoco lleva verbo de urgencia.

**Qué cambió al corregirlo.** Once periodicidades nuevas en seis marcos, y una de
ellas con número:

| Marco | Antes | Después | Lo que faltaba |
|---|---|---|---|
| dora | 18 (9 con núm.) | 21 (9) | art. 10.1, 13.3 y 16.1.g |
| mica | 5 (5) | 8 (6) | art. 34.12, 37.5 y 81.12 (esta con número) |
| ai-act | 0 | 2 (0) | art. 9.2 y 72.2 |
| ens | 7 (6) | 8 (6) | art. 27 |
| mdr | 5 (4) | 6 (4) | anexo I, capítulo I, punto 3 |
| rgpd | 0 | 1 (0) | art. 32.1.d |

Total: de 260 relojes a 271. Ni los plazos ni los eventos se movieron.

**Qué resistió.** Los ceros de `ley2-2023`, `lopdgdd` y `data-act` se han vuelto a
comprobar con el vocabulario completo de la sección 2 bis y siguen en pie, cada
uno con el motivo escrito en su ficha. El de `eni` no resistió, y por qué está en
la sección siguiente. Y los siete referenciales siguen en "no verificado" por la
frontera de licencia, que no es un problema de vocabulario y no se toca.

## 6 ter. El fallo de la segunda pasada, y en qué dirección

Mismo formato que la sección anterior, porque el compromiso es el mismo: quien use
este censo para decidir tiene derecho a saber qué clase de error contiene.

**Qué pasó.** La segunda pasada dejó tres marcos marcados "estimado", uno marcado
"contado" que no lo estaba, un marco a cero por no leer el instrumento que lo
completa, y un paquete sin fila.

**Por qué se escapó.** Cuatro motivos, y ninguno es de vocabulario:

1. **Los anexos no se leyeron.** En el CRA los plazos de retención viven en el
   anexo VIII, repetidos por módulo de conformidad. En el MDR, en los anexos VI,
   XIII y XV.
2. **Solo se recorrió el actor principal.** El MDR obliga también al importador,
   al agente económico y al promotor de investigaciones clínicas, con plazos
   propios. Se contó al fabricante.
3. **La retención documental no tenía marcador.** No lleva verbo de urgencia, así
   que ninguna rejilla de plazo la producía como candidata.
4. **"Estimado" congeló tres filas.** Una marca que dice "esto es aproximado" se
   lee como "esto ya está" y nadie vuelve. Por eso el hallazgo 9 la retira del
   método.

**En qué dirección se equivoca.** Otra vez por defecto y en las dos familias que
menos ruido hacen al leer: retención documental y obligaciones de actores
secundarios. Ninguna cifra bajó. Si aparece un error nuevo, la apuesta sigue
siendo que el número correcto es mayor.

**Qué cambió al corregirlo.**

| Paquete | Antes | Después | Lo que faltaba |
|---|---|---|---|
| mica | 7 / 8 (6) / 4 = 19 | 21 / 20 (11) / 12 = 53 | retención, seguro, liquidación, anexos V y VI como control |
| mdr | 5 / 6 (4) / 5 = 16 | 17 / 8 (4) / 14 = 39 | promotor de investigaciones clínicas, importador, anexos VI, XIII y XV |
| psd2 | 6 / 2 (2) / 3 = 11 | 11 / 6 (5) / 10 = 27 | art. 21 de retención, preavisos del contrato marco, cadencias del art. 5.1 |
| cra | 7 / 1 (0) / 8 = 16 | 15 / 1 (0) / 8 = 24 | ocho plazos de retención, siete de ellos en el anexo VIII |
| eni | 0 / 0 / 0 = 0 | 0 / 3 (0) / 2 = 5 | las doce Normas Técnicas de Interoperabilidad |
| ley2-2023 | 7 / 0 / 3 = 10 | 6 / 0 / 3 = 9 | nada: la columna contaba relojes donde pide obligaciones |
| demo-empresa | sin fila | 1 / 2 (2) / 1 = 4 | la fila |

Total: de 271 a 360 obligaciones con reloj. **Es la única corrección de las tres
pasadas que mueve las tres columnas**, y la única que mueve la posición de un
marco en la tabla: mica sube de la cuarta a la segunda y mdr de la sexta a la
tercera.

**Qué resistió, y hasta dónde exactamente.**

*Rejilla de retención*, pasada sobre once textos (`rgpd`, `nis2-ue`,
`nis2-tecnica`, `dora`, `ai-act`, `data-act`, `dga`, `cra`, `ens`, `lopdgdd` y
`ley2-2023`): solo movió `cra`. El AI Act era el único que ya la tenía completa,
ocho apartados con sus diez años, y eso es lo que hace creíble que su 26 sea un
26. `dora`, `dga`, `nis2-ue`, `nis2-tecnica` y `ens` no tienen ninguna retención
documental; `data-act` tiene una que no es del obligado (art. 21.4, un permiso, no
un deber); en `ley2-2023` y `lopdgdd` las que hay ya estaban contadas.

*Rejilla de actores secundarios*, pasada sobre los mismos once textos buscando
`importador`, `distribuidor`, `representante autorizado`, `promotor` y `agente
económico`: solo el AI Act tiene cadena de suministro, y **ya estaba contada**
(arts. 22.3, 23.5, 24.4, 54.3 y 54.5). Las dos apariciones sueltas son falsos
positivos y se dicen para que nadie las vuelva a perseguir: el anexo I de NIS2
nombra a los distribuidores de agua como sector, y el art. 26.3 de DGA usa la
expresión hablando del personal de las autoridades competentes.

Los dos resultados son negativos y por eso valen: la corrección de la tercera
pasada es grande, pero **está localizada en los cuatro marcos que la sección 6 ter
nombra**, no repartida por toda la tabla.

## 7. Orden de autoría propuesto

Relojes primero, cruzando marcos. La unidad de trabajo no es el marco, es la
familia de reloj: escribir la misma primitiva temporal en siete marcos a la vez
cuesta menos que escribir siete marcos enteros, y activa más obligaciones por
hora de trabajo.

**Criterio, en una línea**: relojes verificados dividido por número de fuentes
primarias que hay que leer, ponderado por cuántos de los clientes objetivo
(organización europea de 20 a 5.000 empleados) alcanza el marco.

### Familia A: notificación escalonada de incidente. Primera, y con diferencia

Once fuentes, treinta y tres relojes, una sola primitiva (plazo con disparador
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
| psd2 | art. 96.1 | sin número en la directiva; el reloj español es el art. 67.1 del RDL 19/2018, también sin número |
| **nis1-es** | **RD 43/2021, anexo, tabla 3** | **cinco relojes, en vigor hoy en España, y no hay paquete; ver abajo** |
| mdr | art. 87.3, 87.4 y 87.5 | quince, dos y diez días sobre el mismo disparador, con el límite decidido por la gravedad |
| dora | art. 19.1 (nivel 1) con el Delegado | el disparador es la *clasificación* del incidente, no el conocimiento: es el caso raro |

**El cambio de orden dentro de la familia A que introduce la tercera pasada**:
`nis1-es` entra, y entra arriba. Es el RD 43/2021, anexo, tabla 3, con notificación
inicial inmediata, intermedia a 24 o 48 horas y final a 20 días para nivel
CRÍTICO, y 72 horas y 40 días para MUY ALTO. Razones, en orden: **vincula hoy** en
España mientras NIS2 no se transponga, es exactamente la misma primitiva de tres
hitos que el resto de la familia, y **no caduca** cuando salga la transposición
porque entonces solo cambiarán los límites sobre una forma ya construida. Frente a
`nis2-ue`, que es lo que hoy encabeza la venta española y que no obliga a nadie
por sí mismo, es la misma venta con la ventaja de ser cierta.

Trae además una primitiva que ningún otro miembro de la familia tiene y que
conviene construir aquí, donde hay cinco relojes que la usan: **el límite depende
de una categoría que asigna el propio obligado** (el nivel de peligrosidad o
impacto), y el cómputo de la intermedia y la final arranca en la remisión de la
inicial, no en el incidente. Dos hitos encadenados, que es lo que también hacen el
art. 14.2.c del CRA y el art. 5.1.c del Delegado de DORA.

### Familia B: revisión anual del marco y de la apreciación de riesgos. Segunda

Ocho marcos, una primitiva periódica de 12 meses con disparador "última
ocurrencia". Es la cadencia que más aparece en todo el corpus y la que el cliente
ya tiene en su calendario, así que es la que convierte el producto en algo que se
mira todas las semanas.

1. **nis2-tecnica** (Reg. Ejec. 2024/2690), puntos 1.1.2, 2.1.4 y 10.1.3, más los
   38 puntos de cadencia sin cuantificar. Es el marco con mejor relación entre
   relojes y trabajo de todo el corpus: 61 relojes en un anexo con numeración
   estable.
2. **dora**, los nueve artículos con cadencia numérica (6.5, 8.1, 8.2, 8.7, 11.6,
   13.5, 24.6, 26.1, 28.3).
3. **rgpd** art. 32.1.d y **ai-act** art. 9.2 y 72.2, que la segunda pasada
   incorporó a esta familia. Son las tres cadencias sin número de mayor alcance
   del corpus, y las tres se escriben con el patrón de `iso27001`: plazum propone
   el intervalo, lo justifica por escrito y el cliente lo cambia.
4. **iso27001**, ya hecho, sirve de referencia de cómputo.
5. **ens**, ya hecho.
6. **psd2** art. 95.2 con el art. 66.2 del RDL 19/2018, que ya está resuelto: los
   dos dicen "al menos una vez al año", así que el número por defecto tiene suelo
   legal y el cliente solo puede apretarlo, nunca aflojarlo. Es el mejor caso de
   demostración del patrón `iso27001` que hay en todo el corpus transcrito.
7. **mica** art. 34.3, 35.5, 46.2, 47.3, 68.6, 68.8 y 78.6, siete cadencias sin
   número del mismo tipo (el órgano de dirección revisa la eficacia de lo que
   tiene puesto), que la tercera pasada añadió. Van aquí y no en el bloque
   "después" de mica: son siete relojes de una primitiva que ya se está
   escribiendo, así que su coste marginal es casi nulo aunque el marco entero no
   sea prioritario.

### Familia C: auditoría y certificación de ciclo largo. Tercera

Seis marcos, primitiva periódica de 6, 24 o 36 meses, y es la que produce el aviso
que un CISO agradece con seis meses de antelación, así que es la que vende el
escalado.

`ens` art. 31.1 e ITS de Conformidad III.2 y III.3 (ya hechos), `eidas2` art. 20.1
(24 meses), `dora` art. 26.1 (36 meses), `mdr` art. 31.5 (24 meses) y art. 86.1
(PSUR anual y bienal según la clase), `mica` art. 36.9 y art. 58 (**auditoría
semestral de la reserva de activos**, que la tercera pasada añadió y que es la
única cadencia de seis meses de todo el corpus, así que es la que obliga a que la
primitiva periódica no esté cableada a doce o veinticuatro), `mica` art. 81.12 (24
meses) y `eni` (NTI de Política de gestión de documentos, apartado X.2, auditoría
periódica sin número).

### Familia D: disparador por cambio sustancial. Cuarta

Seis marcos con el mismo disparador y ninguno con plazo: `nis2-tecnica` (19
puntos), `ens` art. 31.1 párrafo segundo, `dora` art. 8.3 y 11.6, `cra` art. 22,
`ai-act` art. 43.4, `iso27001` requisito 8.2. Se deja para la cuarta posición
porque **necesita el motor de eventos**, no solo el de ventana, y porque
`iso27001/RITUALES.md` ya avisa de que ese disparador se declara aparte.

### Familia E: retención y conservación. Sube a la quinta con mucho más peso

Era la familia menor del plan, cinco marcos y siete relojes. Después de abrir la
rejilla 3 de la sección 2 ter **son siete marcos y treinta y un relojes**, la
segunda familia más numerosa del corpus. Eso no la sube de posición por sí solo,
porque su valor operativo sigue siendo bajo (nadie se olvida de un plazo de diez
años), pero sí cambia lo que cuesta y lo que rinde escribirla: es la que más
obligaciones activa por primitiva construida.

- `cra`: art. 13.8 (soporte de al menos 5 años), art. 13.9, art. 13.13, art. 19.6
  y los cinco puntos del anexo VIII, todos de diez años **o el período de soporte
  si es más largo**.
- `mdr`: art. 10.8 (10 y 15 años), anexo XIII sección 4 (10 y 15), anexo XV
  capítulo III sección 3 (10), y las remisiones de los arts. 11.3.c y 13.9.
- `ai-act`: arts. 18.1, 19.1, 22.3, 23.5, 26.6, 47.1 y 54.3 (10 años y 6 meses).
- `mica`: art. 12.9 (10 años), art. 68.9 (5 años, ampliables a 7 a requerimiento
  de la autoridad) y art. 76.15 (5 años).
- `psd2`: art. 21 (5 años), y **art. 24 del RDL 19/2018 (6 años) para España**.
- `ley2-2023`: art. 26.2 (10 años). `lopdgdd`: art. 22.3 (1 mes).

**La primitiva que hay que construir aquí, y que el motor no tiene**: el máximo de
dos duraciones, una fija en la norma y otra declarada por el propio obligado. Es
la forma del CRA (diez años o el período de soporte, el que sea mayor) y la de
`mica` art. 68.9 (cinco años, siete si la autoridad lo pide antes de que venzan).
Sin ella, el paquete `cra` no se puede escribir bien, y eso mueve esta familia por
delante de la D en la práctica aunque su valor operativo sea menor.

### Familia F: derechos y reclamaciones con plazo. Sexta

`rgpd` art. 12.3 y 12.4, `ley2-2023` art. 9.2.c y 9.2.d, `psd2` art. 101.2 y
77.2, `data-act` art. 18.2, `lopdgdd` art. 37.2 y 65.4, `mica` art. 13.2 (14 días
para el reembolso del desistimiento) y art. 14.3 (25 días naturales). Plazos
cortos disparados por una solicitud externa. Es la familia que más se parece a un
ticket y por eso la que más compite con herramientas que el cliente ya tiene.

Y trae el contraste que mejor explica el producto: en esta misma familia, el art.
101.2 de PSD2 da treinta y cinco días hábiles de tope y el art. 69.2 del RDL
19/2018 da un mes, para la misma obligación y el mismo obligado. Un catálogo que
solo tenga el número europeo le da al cliente español una fecha equivocada, en la
familia donde el cliente mira la fecha todos los días.

### Familia G: preaviso contractual. Nueva, séptima

No existía en el plan porque no había ningún reloj de esta forma contado. Ahora
hay siete, todos de la tercera pasada, y todos con la misma primitiva: **un plazo
que corre hacia atrás desde una fecha que el obligado elige**, no hacia adelante
desde un hecho que le ocurre.

`psd2` art. 54.1 y art. 55.3 (dos meses de antelación para modificar o rescindir
el contrato marco), `mica` art. 67.4.b (90 días de preaviso de cancelación del
seguro), `mdr` art. 75.3 (38 días de espera antes de aplicar la modificación de
una investigación clínica), `mica` art. 65.4 (no antes del decimoquinto día
natural), `data-act` art. 25.2.d (preaviso de hasta dos meses) y `psd2` art. 55.1
(el preaviso del usuario, tope de un mes, que el proveedor tiene que honrar).

Va la última porque es la que menos alarma produce, pero se anota como familia
propia porque el motor la calcula al revés que todas las demás: la fecha límite es
un dato de entrada y lo que se calcula es **cuándo hay que empezar**.

### Lo que se deja para después, y por qué en una línea cada uno

- **mica como marco entero**: 53 obligaciones con reloj, que es mucho, pero la
  población es la más estrecha del corpus y la mayoría de sus plazos son de
  autorización. Sus relojes entran por las familias B, C, E y F cuando esas
  familias se escriben, y el marco completo espera a que haya un cliente.
- **mdr como marco entero**: 39 obligaciones con reloj y densidad alta, población
  estrecha, y el reloj bueno (PSUR) ya está en el calendario de calidad de
  cualquier fabricante. Sus plazos de notificación entran por la familia A y los
  de retención por la E; el resto espera.
- **csrd**: la aplicación la aplazó dos años la Directiva (UE) 2025/794 y España
  no la ha transpuesto, comprobado el 26-08-2026 en el BOE, así que hoy se
  escribirían fechas que van a cambiar.
- **dga**: cinco plazos y una cadencia, pero la población son proveedores de
  intermediación de datos, casi nadie.
- **psd2**: ya no espera al censo del RDL 19/2018, que está hecho. Espera a una
  decisión que ahora es concreta: **el paquete `psd2` lleva dos juegos de cifras**,
  el de la directiva y el del RDL, y hay que decidir si son dos paquetes, o uno
  con dos vigencias territoriales, o una capa. Es la misma decisión de identidad
  que la de `eidas2` y `csrd`, y ahora hay un caso donde la diferencia es un reloj
  de veinticuatro horas.
- **eni**: cinco relojes, no cero, y todos en las NTI. Sigue sin ser prioritario
  (tres cadencias sin número y dos eventos, solo para el sector público), pero ya
  no es un "no se escribe nunca". Se escribe con la familia B, y la `fuente` de
  cada obligación es la resolución de la NTI, no el RD 4/2010.
- **demo-empresa**: no se escribe, ya está escrito. Se usa: cada primitiva nueva
  de este plan (el máximo de dos duraciones de la familia E, el preaviso invertido
  de la G, el límite por categoría de `nis1-es`) debería tener aquí su obligación
  sintética antes de gastarse en un marco real.
- **iso27002, iso22301, iso42001, iso27701**: cero cadencias numéricas en la
  norma, así que el paquete es una lista de rituales de plazum. Se escriben cuando
  haya un cliente que los pida, replicando el patrón de `iso27001`, no antes.
- **soc2**: la periodicidad la fija el contrato de atestación, no el marco.
- **pci-dss** y **tisax**: no verificables sin la copia del cliente. Se escriben,
  si se escriben, dentro de la instancia del cliente y con su copia.
- **cis** y **stig**: sin reloj propio, no hay nada que escribir.
- **magerit**: sin reloj propio.
- **nist-800-53** y **nist-csf**: sin autoría prevista, decisión D-1.

## 8. Lo que este censo no ha verificado, en una lista

Los puntos 2, 3 y 6 de la lista anterior están cerrados por la tercera pasada y se
han quitado de aquí: las doce NTI del ENI están censadas (ficha `eni`), el RDL
19/2018 está censado (ficha `rdl19-2018`), y los apartados de `mdr`, `mica` y
`psd2` fuera del núcleo se han revisado uno a uno. Los puntos 1 y 4 se han
comprobado hoy contra fuente primaria y siguen abiertos, pero por otra razón: no
porque no se hayan mirado, sino porque lo que se busca no existe todavía.

1. **La transposición española de NIS2**: comprobada el 26-08-2026 en el índice de
   legislación consolidada del BOE, no está. Vuelve a hacer falta la consulta el
   día de escribir el paquete, con las tres búsquedas de la ficha `nis2-ue`.
   Límite conocido del método: mira consolidada, no publicación del día.
2. **La Ley de Sostenibilidad Empresarial** (transposición de CSRD): comprobada el
   mismo día y con el mismo método, tampoco está. Mismo límite.
3. **Los recuentos de los siete referenciales**. Haría falta: la copia licenciada
   del cliente, leída dentro de su instancia, sin que el recuento salga de su
   paquete. No es un pendiente de trabajo, es la frontera legal.
4. **El resto del RDL 12/2018 y del RD 43/2021**. Se ha censado el reloj de
   notificación, que es el que importaba para el plan, y los plazos de designación
   del responsable de seguridad. El régimen sancionador y las obligaciones de los
   proveedores de servicios digitales no se han revisado apartado a apartado.
5. **Los niveles de peligrosidad e impacto del anexo del RD 43/2021**. Se sabe que
   son cinco y cuáles obligan a notificar, pero no se ha censado el detalle de los
   criterios que asignan cada nivel, que es lo que un paquete necesitaría para que
   el cliente pueda justificar su clasificación.
6. **La edición vigente de ISO/IEC 27701** frente a la que declara el paquete.
7. **Los actos delegados y de ejecución de nivel 2** de `mica`, `dora` (más allá
   del Delegado 2025/301, que sí está), `ai-act` y `data-act`. El hallazgo 1 dice
   que ahí vive parte del reloj y solo se ha seguido esa pista en DORA. Haría
   falta: por cada artículo que delega un plazo, localizar el acto que lo fija.
   Esta es hoy la mayor fuente de error por defecto del censo, por delante de la
   cadencia sin número.
8. **Los anexos de `dora`, `dga`, `eidas2`, `lopdgdd` y `csrd`**. Las rejillas de
   retención y de actores secundarios se han pasado sobre los once textos que
   nombra la sección 6 ter y no han movido nada fuera de `cra`, pero la rejilla de
   anexos solo se ha abierto entera donde había motivo (`cra`, `mdr`, `mica`,
   `nis2-tecnica`, `ai-act`, `ens`). En los cinco de arriba se ha mirado el índice
   de anexos y no el contenido de cada uno.
9. **Las ITS del ENS posteriores a las tres censadas**, si las hay. El censo de
   `ens` se hizo contra el `paquete.json`, que lleva tres, y no se ha comprobado en
   el BOE si el CCN ha publicado alguna más desde entonces. Es el mismo tipo de
   pendiente que las NTI del ENI, que sí resultó tener reloj.
