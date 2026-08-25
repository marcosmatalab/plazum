# El corpus: los 30 marcos

Estado real, sin maquillar. Cada marco es un directorio con su `paquete.json`
pasando el linter legal (la frontera por estrato se comprueba en CI, no de
palabra). "Esqueleto" = metadatos correctos (URN, estrato, licencia, fuente
oficial) y cero obligaciones todavia: la transcripcion es el plan de autoria de
ETAPAS (E3-E7 y ano 2), a ritmo medido, con revision juridica y 3 casos dorados
por reloj. Rellenar esto deprisa seria fabricar derecho, y este proyecto existe
para lo contrario.

Las vigencias de los esqueletos son la entrada en vigor a confirmar al
transcribir; la vigencia que vincula es siempre la de cada obligacion.

| Marco | Estrato | Estado |
|---|---|---|
| ens | transcrito | **completo**: 132 obligaciones (articulado, anexo I, las 73 medidas del anexo II y las tres ITS), 8 relojes, 24 dorados en verde. Faltan los refuerzos del anexo II y la tabla de aplicacion por nivel, que esperan a las reglas de aplicabilidad: detalle en `ens/COBERTURA.md` |
| iso27001 | referencial | **completo**: 129 obligaciones (30 clausulas + 93 controles del anexo A + 6 rituales de dutiq), 6 relojes, 18 dorados. Cero texto normativo: identificador y titulo corto, el mas largo de 86 caracteres |
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
