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

Las vigencias de los esqueletos son la entrada en vigor a confirmar al
transcribir; la vigencia que vincula es siempre la de cada obligacion.

| Marco | Estrato | Estado |
|---|---|---|
| ens | transcrito | **completo**: 132 obligaciones (articulado, anexo I, las 73 medidas del anexo II y las tres ITS), 8 relojes, 24 dorados en verde. Faltan los refuerzos del anexo II y la tabla de aplicacion por nivel, que esperan a las reglas de aplicabilidad: detalle en `ens/COBERTURA.md` |
| iso27001 | referencial | **completo**: 129 obligaciones (30 clausulas + 93 controles del anexo A + 6 rituales de plazum), 6 relojes, 18 dorados. Cero texto normativo: identificador y titulo corto, el mas largo de 86 caracteres |
| rgpd | transcrito | **semilla con reloj**: art. 33 (72 h), 3 dorados en verde |
| cra | transcrito | **semilla con reloj**: art. 14.1 alerta temprana 24 h (vigente 11-09-2026), 3 dorados |
| lopdgdd, nis2-ue, nis2-tecnica, dora, ai-act, data-act, dga, eidas2, ley2-2023, mica, psd2, mdr, eni, csrd | transcrito | esqueleto |
| iso27002, iso22301, iso42001, iso27701, soc2, pci-dss, tisax | referencial | esqueleto (solo identificadores y titulos; el cliente aporta su copia) |
| cis, stig | delegado | esqueleto (el texto lo tiene la herramienta: OpenSCAP, Trivy, Prowler) |
| nist-800-53, nist-csf | importado | esqueleto, **sin autoria prevista**. No hay importador OSCAL: mil controles federales estadounidenses no le sirven a un CISO europeo, y el modelo de OSCAL no tiene donde poner un plazo (`docs/decisiones.md` D-1) |
| magerit | propio | esqueleto (catalogo de riesgo, reutilizacion RISP con atribucion) |

`demo-empresa` (propio) es la empresa sintetica de la demo y no cuenta entre los 30.

Comprobaciones en CI sobre TODO lo anterior: linter legal por estrato,
`fuente` obligatoria, clase e2e por obligacion, minimo 3 dorados por reloj, y
los dorados ejecutados contra el motor real (si discrepan, gana el dorado).
Hoy son 16 relojes y 48 dorados en verde.

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

- `eidas2` y `csrd` apuntaban al acto **modificativo**. Su `fuente` ya apunta al
  instrumento donde viven las obligaciones (Reglamento 910/2014 y Directiva
  contable 2013/34/UE). **El `urn` sigue nombrando al modificativo**: cambiarlo
  cambia la identidad del paquete en el expediente y en las equivalencias, asi
  que lo decide la autoria. Cada uno lo dice en su `LEEME.md`.
- `nis2-ue` y `psd2` son **directivas**, que en Espana no vinculan por si mismas.
  Su `LEEME.md` dice que vincula de verdad: en PSD2, el RDL 19/2018, que no esta
  censado; en NIS2, nada todavia, porque la transposicion no se ha publicado.
