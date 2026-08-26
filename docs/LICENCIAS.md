# Licencias del corpus: qué se puede transcribir y qué no

> **Para qué sirve este documento.** El corpus de plazum distribuye contenido
> normativo. Parte de ese contenido se puede reproducir entero, parte solo se
> puede referenciar, y parte no se puede tocar. Aquí está la línea, con el
> fundamento de cada tramo, y aquí está la lista negra con su porqué escrito.
>
> **Esto no es un documento de intenciones.** El linter lo hace cumplir. Cada
> régimen de esta página es una constante de `nucleo/corpus` y el campo
> `licencia_fuente` de un paquete solo puede tomar uno de esos valores. Un
> paquete que declare cualquier otra cosa no carga.
>
> Esto no es asesoramiento jurídico.

---

## Los dos campos que declara cada paquete

Todo `paquete.json` trae dos campos obligatorios de higiene legal. El linter
rechaza el paquete si falta cualquiera de los dos.

| Campo | Qué es | Por qué es obligatorio |
|---|---|---|
| `licencia_fuente` | El régimen de derechos de la fuente, de un vocabulario **cerrado** | La `clase` dice qué se distribuye del paquete, y el texto libre de `licencia` es la explicación del autor. Ninguno de los dos es comprobable. Este sí |
| `atribucion` | El aviso literal que hay que **enseñar** a quien usa el producto | La Decisión 2011/833/UE autoriza reutilizar el DOUE con atribución. Una atribución que vive en la cabeza de quien escribió el paquete no es una atribución, y un fichero como este tampoco: se lo cuenta a quien lee el repositorio, no a quien usa el producto |

**La atribución sale en pantalla.** Va en el pie de todas las pantallas del
producto, junto al descargo de asesoramiento jurídico, y hay una puerta que se
pone roja si deja de salir. Ver `superficies/pantallas/atribucion_test.go`.

**El vocabulario es cerrado a propósito.** Una fuente nueva no entra escribiendo
otra cadena en un JSON: entra con su constante en `nucleo/corpus/paquete.go` y su
fila en esta tabla, igual que una dependencia entra por `DEPENDENCIAS.md`. Y el
régimen tiene que ser coherente con la clase: un paquete que se declara
referencial (no puede redistribuir texto) y a la vez dice que su contenido viene
amparado por el artículo 13 del TRLPI (puede redistribuirlo entero) miente en uno
de los dos sitios, y el linter lo para sin poder adivinar en cuál.

---

## 1. Permitido transcribir

El texto va entero dentro del paquete, con su `identificador` declarado (ELI o CELEX). El enlace **no se guarda**: se deriva de ese identificador al pintarlo, en una sola funcion (`corpus.Identificador.Enlace`). Una pagina que se mueve rompe un enlace, nunca un paquete.

| `licencia_fuente` | Fuente | Fundamento | Condición |
|---|---|---|---|
| `boe-trlpi-13` | BOE, disposiciones legales españolas | **Artículo 13 del texto refundido de la Ley de Propiedad Intelectual**: las disposiciones legales y sus correspondientes proyectos no son objeto de propiedad intelectual | Citar la fuente, que exigen las condiciones de reutilización del BOE. El `identificador` del paquete (ELI del BOE) es de donde sale el enlace que se muestra |
| `doue-decision-2011-833` | DOUE y EUR-Lex | **Decisión 2011/833/UE** de la Comisión, sobre reutilización de documentos de la Comisión | **Atribución obligatoria.** No es cortesía, es la condición de la autorización. Y solo se considera auténtico el texto publicado en la edición impresa del Diario Oficial |
| `dominio-publico-eeuu` | NIST y otras obras de la administración federal de los Estados Unidos | Obra de la administración federal estadounidense, sin derechos de autor federales | Ninguna, más allá de citar la fuente por higiene |

**Sobre NIST, que está en la tabla y no tiene autoría prevista.** `nist-800-53` y
`nist-csf` se pueden transcribir enteros y aun así están fuera del plan de
autoría, por decisión de producto y no por derechos: mil controles federales
estadounidenses no le sirven a un CISO europeo, y el modelo de OSCAL no tiene
dónde poner un plazo, que es el diferenciador de este producto. El razonamiento
completo, con el dato de adopción, en `docs/decisiones.md` D-1. La fila está aquí
para que nadie tenga que volver a preguntarse si el problema era la licencia.

---

## 2. Solo referencial

Identificador y título corto. **Jamás el texto.** La copia licenciada la aporta
el cliente.

| `licencia_fuente` | Marcos | Por qué |
|---|---|---|
| `sin-licencia-de-texto` | ISO e ISO/IEC, PCI DSS, SOC 2 (Trust Services Criteria del AICPA), TISAX y el catálogo VDA ISA | Son normas de pago con derechos reservados. No hay ninguna autorización de reutilización, ni gratuita ni con atribución |

Lo que sí puede llevar un paquete referencial:

- el identificador de la cláusula o del control (`A.8.24`, `4.3`),
- un título corto propio, y
- el mapeo, las preguntas de alcance y los relojes, que son obra nuestra.

Lo que no puede llevar, en **ningún** campo: el enunciado del control. El linter
mide **todos** los campos de texto libre del formato, no solo `texto_legal`, con
tres techos según lo que el campo sea:

| Techo | Bytes | Para qué campos |
|---|---|---|
| `LimiteTextoReferencial` | 120 | Prosa: título, ayuda, descripción, texto de una pregunta, artículo |
| `LimiteCitaReferencial` | 300 | Referencia: cita, URN, clave de formulario, enlace, licencia, atribución, **y las reglas de aplicabilidad con sus citas** |
| `LimiteDerivacionReferencial` | 600 | Solo la `cita_del_esperado` de un caso dorado, que justifica una fecha paso a paso |

Se mide en **bytes** y no en runas, a la baja: un texto acentuado gasta más bytes
que caracteres, así que el límite aprieta más justo donde el texto es prosa de
verdad.

**Lo que esto no cierra, dicho para que conste.** El límite es por campo. Quien
quiera copiar un catálogo entero puede repartirlo entre la ayuda, la descripción
y el título de cien obligaciones. Contra eso no hay linter, hay revisión del
paquete. Lo que el límite corta es el caso real, que es pegar el control de un
tirón en el campo que se tenía a mano.

---

## 3. Delegado y propio

| `licencia_fuente` | Marcos | Qué significa |
|---|---|---|
| `la-tiene-la-herramienta` | CIS Benchmarks, STIG | **No se distribuye nada.** La comprobación la ejecuta una herramienta externa que ya tiene la licencia del contenido: OpenSCAP, Trivy, Prowler. El paquete solo dice qué herramienta comprueba qué |
| `risp-con-atribucion` | MAGERIT | Reutilización de información del sector público, permitida con atribución y sin desnaturalizar el contenido |
| `del-proyecto` | `demo-empresa` y los datos propios | Creado por este proyecto. No hay tercero con derechos |

---

## 4. Prohibidos

**Esta lista es lista negra, no lista de pendientes.** No son "todavía no", son
"no". Están escritas aquí, con su motivo, porque alguien va a volver a
proponerlas dentro de seis meses y lo que se tiene que encontrar es el porqué.

El linter las rechaza por nombre y devuelve el motivo en el error, para que quien
lo intente lea la razón y no un "valor inválido" que invita a insistir.

| Valor rechazado | Qué es | Por qué no, y por qué no va a cambiar |
|---|---|---|
| `cc-by-nc-nd` | La licencia de los **CIS Controls** | El **NC** prohíbe el uso comercial y este producto se vende. El **ND** prohíbe cualquier adaptación, y un paquete de corpus **es** una adaptación: parte el texto en obligaciones, le pone relojes y lo reordena. Los dos términos matan el caso por separado, así que ni siquiera hace falta discutir cuál pesa más |
| `cc-by-nc-sa` | La licencia de los **CIS Benchmarks** | El **NC** otra vez, y el **SA** obligaría a relicenciar lo derivado en los mismos términos, que es incompatible con la AGPL de este proyecto. La vía para los benchmarks es la clase **delegado**: se lee la salida de una herramienta que sí tiene la licencia |
| `cc-by-nd` | La licencia del marco **gratuito del SCF** | El **ND** mata cualquier adaptación, igual que arriba. Que sea gratuito no lo hace reutilizable |
| `repositorio-de-terceros` | Un volcado de una norma en GitHub que dice MIT o Apache | **La licencia de un repositorio no alcanza al contenido que quien lo subió no poseía.** Un MIT sobre un fichero con el texto de ISO 27002 no da ningún derecho sobre ISO 27002: da derechos sobre lo que el subidor podía licenciar, que era nada. Es la invariante 3 de `CLAUDE.md` |

### La regla de la fuente primaria, que es la que más se salta

**No se copia contenido normativo de repositorios de terceros.** Ni de GitHub, ni
de recopilaciones comerciales, ni de un PDF que circula. Da igual la licencia que
declare el repositorio.

Dos razones, y la segunda importa tanto como la primera:

1. **Derechos.** Quien subió el fichero no era el titular, así que no podía
   licenciarlo. La licencia del repositorio cubre lo suyo, no lo ajeno.
2. **Exactitud.** Un volcado de tercero no tiene versión consolidada, ni fecha de
   vigencia, ni forma de saber qué modificación incorpora. Un motor de plazos
   legales alimentado con una copia de procedencia desconocida da fechas que
   nadie puede defender delante de un auditor.

Fuentes primarias, que son las únicas: **BOE** por su ELI consolidado, **EUR-Lex
y la Oficina de Publicaciones** por CELEX, **NIST** por su publicación oficial.
El método de extracción está en `docs/censo-relojes.md`, apartado 2.

---

## 5. Qué hacer si aparece una fuente nueva

1. Comprobar que hay una autorización de reutilización **expresa**, y leerla
   entera. El silencio no es autorización.
2. Anotar la condición que impone, que casi siempre es atribuir y a veces es no
   modificar. Si es no modificar, no sirve: un paquete de corpus modifica.
3. Añadir la constante en `nucleo/corpus/paquete.go`, decir con qué clases es
   coherente, y añadir la fila a este documento con el fundamento.
4. Escribir la `atribucion` que va a ver el usuario, con las palabras que exija
   la licencia y no con las nuestras.
5. Si no se cumple alguno de los cuatro pasos, el marco entra como
   **referencial** o no entra.
