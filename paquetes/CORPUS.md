# El corpus: los 30 marcos

Estado real, sin maquillar. Cada marco es un directorio con su `paquete.json`
pasando el linter legal (la frontera por estrato se comprueba en CI, no de
palabra). "Esqueleto" = metadatos correctos (URN, estrato, regimen de licencia
de la fuente, atribucion, fuente oficial) y cero obligaciones todavia: la
transcripcion es el plan de autoria de ETAPAS (E3-E7 y ano 2), a ritmo medido,
con revision juridica y 3 casos dorados por reloj. Rellenar esto deprisa seria
fabricar derecho, y este proyecto existe para lo contrario.

**Todo paquete declara `licencia_fuente` y `atribucion`, y sin los dos no
carga.** El primero es el regimen de derechos de la fuente, de un vocabulario
cerrado que el linter cruza con la clase; el segundo es el aviso literal que el
producto ENSENA a quien lo usa, en el pie de todas las pantallas. La tabla de
regimenes, con lo que se puede transcribir, lo que solo se puede referenciar y
la lista negra con su porque, esta en `docs/LICENCIAS.md`.

**La procedencia se guarda como IDENTIFICADOR, no como direccion.** Cada paquete
declara un bloque `identificador` con su `tipo` (el ELI de la Union o del BOE, la
designacion de una norma ISO, la version de PCI DSS, el identificador de
publicacion del NIST) y su `valor`; el enlace a la fuente oficial se DERIVA de
ahi al pintarlo, en una sola funcion (`corpus.Identificador.Enlace`). El motivo
es que una pagina se mueve y un identificador no: con la URL guardada como dato,
el dia que un editor reorganiza su sitio hay que tocar tantos ficheros de datos
como paquetes tenga ese editor. Lo que no tiene identificador estable (AICPA,
DISA, CIS, ENX, el PAe y los datos propios del proyecto) usa el tipo
`sin-identificador`, que EXIGE escribir el motivo: el hueco es una decision
consultable, no una omision.

Las vigencias de los esqueletos son la entrada en vigor a confirmar al
transcribir; la vigencia que vincula es siempre la de cada obligacion.

| Marco | Estrato | Estado |
|---|---|---|
| ens | transcrito | **completo**: 132 obligaciones (articulado, anexo I, las 73 medidas del anexo II y las tres ITS), 8 relojes, 24 dorados en verde. Faltan los refuerzos del anexo II y la tabla de aplicacion por nivel, que esperan a las reglas de aplicabilidad: detalle en `ens/COBERTURA.md` |
| iso27001 | referencial | **completo**: 132 obligaciones (30 clausulas + 93 controles del anexo A + 9 rituales de plazum), 9 relojes, 27 dorados. Cero texto normativo: identificador y titulo corto, el mas largo de 86 caracteres |
| iso42001 | referencial | **completo con un hueco declarado**: 48 obligaciones (32 clausulas + las 9 categorias del anexo A + 7 rituales de plazum), 7 relojes, 21 dorados. Los 38 titulos de control individuales del anexo A NO estan, y el porque esta escrito en `iso42001/LEEME.md`: sin copia licenciada delante, escribirlos de memoria seria fabricar el catalogo |
| ai-act | transcrito | **el marco mas cubierto del corpus tras el recuento del 04-09-2026**: 38 obligaciones, 34 relojes, 97 dorados, 32 reglas de aplicabilidad. Ademas del art. 50, del art. 73 y de las retenciones, cubre las tres cadenas de no conformidad y riesgo (arts. 20.1, 20.2, 24.4 y 26.5 en sus dos deberes), la evaluacion de impacto en los derechos fundamentales (arts. 27.1 y 27.3), la nueva evaluacion de la conformidad por modificacion sustancial (art. 43.4), la notificacion del incidente grave del modelo de uso general a la Oficina de IA (art. 55.1.c), las pruebas en condiciones reales (arts. 60.7 y 60.8) y los tres topes del art. 111. **La unica cadencia CON NUMERO que el reglamento pone a un obligado** es el informe anual del art. 26.10, parrafo sexto, y el censo afirmaba que no habia ninguna. Quedan **6 deberes identificados y sin escribir**: los tres del art. 5 (uso urgente de identificacion biometrica en tiempo real), el art. 5.4, el art. 27.2 y el preaviso del art. 73.6, parrafo segundo, que espera a la primitiva `preaviso` |
| rgpd | transcrito | **derechos del interesado, brecha y evaluacion de impacto**: 11 obligaciones, 10 relojes, 24 dorados. El art. 33.1 (72 h a la autoridad) y el ritual del art. 32.1.d, los cuatro del 02-09-2026 (arts. 12.3 en sus dos deberes, 12.4 y 34.1), los arts. 19 y 33.2, y desde el 04-09-2026 los arts. 35.1 (evaluacion de impacto antes del tratamiento) y 36.1 (consulta previa a la autoridad de control), que el censo contaba como UN punto y son dos. Los tres del art. 12 estrenan el traslado al habil siguiente del art. 3.4 del Rgto. 1182/71, que un plazo en horas como el 33.1 no puede tener. Sin escribir queda **1**: el art. 14.3.a, que necesita una primitiva capaz de decir «el mas temprano de N limites condicionales» |
| cra | transcrito | **familia A completa**: las dos cadenas del art. 14 (vulnerabilidad e incidente), 7 hitos, 10 dorados. El informe final de la vulnerabilidad cuenta desde que HAY MEDIDA CORRECTORA, no desde el conocimiento |
| nis1-es | transcrito | **familia A mas el regimen del responsable de la seguridad**: 10 obligaciones, 8 relojes, 24 dorados. La tabla 3 del anexo (5 hitos) es lo unico que vincula HOY en Espana en notificacion de incidentes de red, y desde el 04-09-2026 estan tambien los arts. 7.1 (designar al responsable de la seguridad en tres meses), 7.2 en sus dos deberes (comunicar la designacion, y los nombramientos y ceses en un mes) y 6.4 en los suyos (remitir la Declaracion de Aplicabilidad en seis meses, y revisarla al menos cada tres anos, que es SUELO legal). Trae ademas el unico ritual de plazum del paquete, los controles periodicos de seguridad del art. 7.3, letra b), cuyo intervalo se justifica con dos numeros del propio real decreto. **El art. 6.4 no estaba en la ficha del censo**, que contaba ocho plazos y no lo nombraba |
| dora | transcrito | **familia A**: art. 19 con el Delegado (UE) 2025/301 art. 5, 3 hitos, 5 dorados. Estrena el TOPE: cuatro horas desde la clasificacion y a mas tardar veinticuatro desde el conocimiento, y manda el que caiga antes |
| nis2-ue | transcrito | **familia A mas el registro**: 2 obligaciones, 6 hitos, 8 dorados. El art. 23.4 (5 hitos) y, desde el 02-09-2026, el art. 27.3 (tres meses desde el cambio en la informacion de registro). El 27.3 NO alcanza a toda entidad esencial o importante: el art. 27.1 es lista cerrada de infraestructura digital, y eso se prueba en las dos direcciones. Es una DIRECTIVA sin transponer en Espana: sus plazos no se le pueden ensenar aqui como exigibles |
| eidas2 | transcrito | **familia A**: los tres plazos de 24 h (arts. 19 bis.1.b, 24.2.f ter y 24.3), 9 dorados. Los dos primeros cuentan desde hechos DISTINTOS y ese contraste es el mejor ejemplo del corpus |
| mdr | transcrito | **familia A**: art. 87, 3 hitos por calificacion (15, 2 y 10 dias), 4 dorados. Misma forma que el art. 73 del AI Act |
| psd2-es | transcrito | **familia A y B**: RDL 19/2018 arts. 67.1 (obliga y NO tiene numero) y 66.2 (al menos anual), 3 dorados. Es lo que vincula en Espana, no la directiva |
| nis2-tecnica | transcrito | **las tres cadencias CON NUMERO del anexo**: puntos 1.1.2 (revision anual de la politica, y la hace el ORGANO DE DIRECCION), 2.1.4 (revision de la evaluacion de riesgos y del plan de tratamiento) y 10.1.3 (revision de la asignacion de personal a roles). 3 hitos, 9 dorados, 4 reglas de aplicabilidad. El ambito NO es NIS2 entero: el art. 1 da una lista cerrada de once tipos que llama *entidades pertinentes*. Quedan las 38 cadencias sin numero y los 20 disparadores por evento, censados en `docs/censo-relojes.md` |
| lopdgdd, data-act, dga, ley2-2023, mica, psd2, eni, csrd | transcrito | esqueleto |
| iso27002, iso22301, iso27701 | referencial | esqueleto (solo identificadores y titulos; el cliente aporta su copia) |
| soc2, pci-dss, tisax | referencial | **rituales de plazum**: 17 relojes con intervalo `propuesto`, su justificacion y su `cuando_cambiarlo`, y 52 dorados. Cero texto del marco: el estrato cerrado lo prohibe, asi que aqui no hay transcripcion, hay autoria nuestra. **Ninguno de los 17 dice a que requisito del catalogo sirve**, y eso esta escrito en sus tres LEEME: verificarlo exige la copia licenciada, y un numero de requisito escrito de memoria tiene la FORMA de lo verificable, que es lo que hace que nadie vaya a comprobarlo |
| cis, stig | delegado | esqueleto (el texto lo tiene la herramienta: OpenSCAP, Trivy, Prowler) |
| nist-800-53, nist-csf | importado | esqueleto, **sin autoria prevista**. No hay importador OSCAL: mil controles federales estadounidenses no le sirven a un CISO europeo, y el modelo de OSCAL no tiene donde poner un plazo (`docs/decisiones.md` D-1) |
| magerit | propio | esqueleto (catalogo de riesgo, reutilizacion RISP con atribucion) |

`demo-empresa` (propio) es la empresa sintetica de la demo y no cuenta entre los
30. `nis1-es` y `psd2-es` tampoco: son los instrumentos ESPANOLES que transponen
lo que las directivas `nis2-ue` y `psd2` no pueden exigir por si mismas, y tienen
paquete propio porque un identificador que mezcle dos instrumentos no se puede
citar en un expediente. Con ellos, `paquetes/` tiene 33 directorios.

Comprobaciones en CI sobre TODO lo anterior: linter legal por estrato,
`identificador` de fuente obligatorio, clase e2e por obligacion, minimo 3
dorados por reloj, y
los dorados ejecutados contra el motor real (si discrepan, gana el dorado).
Hoy son **271 hitos de reloj y 766 dorados** en verde, repartidos en veintiun paquetes
(veinte marcos mas `demo-empresa`). De esos 271 hitos, **83 obligan sin numero**,
medido el 04-09-2026 con este criterio, y ahora el criterio dice tambien que
primitivas excluye, que es lo que le faltaba: hitos de una obligacion cuya
primitiva NO es `periodica` y cuyo `limite` esta vacio o vale `indeterminado`.

ESTA ES LA TERCERA VEZ QUE ESTE NUMERO NO ES RECONSTRUIBLE, y las dos anteriores
las cuenta este mismo parrafo: dijo trece cuando su enumeracion nombraba ocho, y
dijo 28 cuando su criterio daba 19. **La tercera es la del 03-09-2026, que decia
27**: aplicado el criterio literal que ella misma escribia (hitos que no son de
cadencia, con `limite` vacio o `indeterminado`) al arbol de aquel dia salen
**69**, y no hay ninguna combinacion de exclusiones de primitiva que de 27
(excluyendo tambien `maximo` salen 52, y excluyendo ademas `continua`, 47).
Contar obligaciones en vez de hitos da los mismos numeros. Asi que el 27 contaba
algo que el parrafo no decia, igual que el 28 y que el trece.

Lo que se hace es lo que el parrafo lleva pidiendo desde el principio y no habia
hecho: **decir el criterio con precision suficiente para reejecutarlo**. Con el
de arriba salen 69 sobre el arbol del 03-09-2026 y 83 sobre este. Ningun test ata
este numero, y esa es exactamente la razon de que se haya quedado viejo tres
veces: el que atan `cuentas_publicadas_test.go` y `marcos_v1_test.go` son los
hitos y los dorados, no este.
Los tres de siempre (la notificacion inicial de la tabla 3 del RD 43/2021, el
art. 67.1 del RDL 19/2018 y la disponibilidad de la medida correctora del art.
14.2.c del CRA), los **cinco relojes de evento del CRA** anadidos el 02-09-2026
(arts. 13.21, 14.8, 19.5, 20.4 y 57.2) y, desde el mismo dia, la **comunicacion
al interesado del art. 34.1 del RGPD**, que dice «sin dilacion indebida» y no da
cifra: las 72 horas son del art. 33.1 y son para la autoridad de control. Salen
como *sin plazo legal* y el motor mide el tiempo transcurrido, en vez de
inventarse una fecha que la norma no da. Los dos ultimos entraron el 02-09-2026 con los dos
marcos espanoles: la **remision al Ministerio Fiscal del art. 9.2.j de la Ley
2/2023**, que dice «con caracter inmediato», y la **comunicacion del art. 36.4 de
la Ley Organica 3/2018**, que dice «inmediatamente». En los dos, copiarle el numero
al apartado de al lado (los siete dias del acuse de recibo, los diez del art. 34.3)
habria sido inventar un plazo, y ademas uno mas largo que «inmediato».

El del art. 57.2 no es igual que los otros siete y por eso se dice aparte: ahi
el numero EXISTE y lo fija la autoridad de vigilancia del mercado en su
requerimiento. No es un plazo abierto, es un plazo cerrado cuyo valor plazum no
puede saber, y el hueco se rellena copiando la fecha del requerimiento.

El salto de 61 a 105 es de una sola pieza: las **44 cadencias sin numero del
anexo del Reglamento de Ejecucion (UE) 2024/2690**, escritas el 28-08-2026 en
`nis2-tecnica`. Las 44 llevan `origen_del_intervalo: propuesto`, o sea que **el
numero lo pone plazum y no la norma**, y cada una viaja con su justificacion,
sus fuentes del propio anexo y sus instrucciones de uso (`cuando_cambiarlo`, una
condicion para acortarlo y otra para alargarlo). El censo de ese anexo se
corrigio al escribirlas: eran 44 y no 34, porque el recuento anterior solo
miraba los puntos de tres niveles y las secciones 7 y 9 se numeran a dos. El
detalle, en `docs/pendientes.md`.

Todo paquete con obligaciones transcritas o referenciales trae ademas, en su
directorio, los documentos que un CISO lee antes que el JSON:

- `LEEME.md`: que obliga, desde cuando, y que NO hace el paquete.
- `COBERTURA.md` o la seccion equivalente: lo que quedo sin mapear, nominalmente.
- `COMPUTO.md` (transcritos) o `RITUALES.md` (referenciales): de donde sale cada
  fecha. Son la fuente que citan los casos dorados, no la implementacion.

Los dos limites que este parrafo declaraba ya no existen, y conviene decir en
que se han convertido porque los dos cambian como se escribe un paquete:

- **El paquete SI declara sus reglas de aplicabilidad**, en el dialecto Datalog
  estratificado del Anexo C de `docs/guia.md`. Una obligacion que solo alcanza a
  una categoria ya no se le ensena a todo el mundo. El ENS lleva 29 reglas como
  datos, con cita articulo por articulo, ejecutadas contra el motor y no solo
  parseadas por el linter.
- **El limite del estrato referencial ya no mira solo `texto_legal`.** Hay tres
  techos, y la diferencia entre ellos es la diferencia entre prosa y
  localizador: `LimiteTextoReferencial` (120 bytes) para los campos que son
  PROSA, `LimiteCitaReferencial` (300) para los que son REFERENCIA (cita, URN,
  clave de formulario, fuente, licencia), y `LimiteDerivacionReferencial` (600)
  para la `cita_del_esperado` de un dorado, que justifica una fecha paso a paso
  y no se puede resumir sin dejar de ser auditable. Se mide en BYTES y no en
  runas, a la baja: un texto acentuado gasta mas bytes que caracteres, asi que
  el limite aprieta mas justo donde el texto es prosa de verdad.
- **Y el bloque de aplicabilidad ya esta DENTRO de la frontera.** Se quedo fuera
  con la excusa de que tiene su propio linter; ese linter comprueba que la regla
  se parsea, no cuanto texto lleva dentro, y una regla es una cadena libre con
  literales (`aplica("...", S) :- ...`). Sus campos van con el techo de
  referencia, que la regla mas larga del corpus no roza: gasta 116 bytes.

## Cuatro paquetes apuntaban al instrumento equivocado

Lo encontro el censo de relojes y esta a medio arreglar, dicho aqui para que no
se pierda:

- `eidas2` y `csrd` apuntaban al acto **modificativo**. Su `identificador` ya apunta al
  instrumento donde viven las obligaciones (Reglamento 910/2014 y Directiva
  contable 2013/34/UE). **El `urn` sigue nombrando al modificativo**: cambiarlo
  cambia la identidad del paquete en el expediente y en las equivalencias, asi
  que lo decide la autoria. Cada uno lo dice en su `LEEME.md`.
- `nis2-ue` y `psd2` son **directivas**, que en Espana no vinculan por si mismas.
  Su `LEEME.md` dice que vincula de verdad: en PSD2, el RDL 19/2018, que ya tiene
  paquete propio (`psd2-es`, 27-08-2026); en NIS2, nada todavia, porque la
  transposicion no se ha publicado, y por eso `nis1-es` sigue siendo lo unico
  exigible en Espana.
