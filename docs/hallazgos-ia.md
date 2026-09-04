# Hallazgos del arranque de la IA (tramo 3, rebanada 3)

> **Qué es este fichero.** El informe de los cuatro cimientos de la IA: búsqueda, verificador de citas, arnés de evals e interruptor. Lo que se construyó, lo que se decidió apartándose de la casilla, lo que salió mal y lo que queda abierto con su cardinal.
>
> **Fecha: 04-09-2026.** Todo número de aquí sale de la salida de un comando de esa sesión, no de memoria.

---

## 1. Lo que se construyó, y dónde

| pieza | paquete | qué garantiza |
|---|---|---|
| Búsqueda BM25 | `adaptadores/busqueda` | encuentra el texto sobre el que citar, con orden determinista |
| Verificador de citas | `adaptadores/ia` | la puerta antialucinación, con salida de tipo opaco |
| Interruptor | `adaptadores/ia`, `PLAZUM_SIN_IA` | con él puesto no sale ni una petición de la máquina |
| Adaptador de modelo | `adaptadores/ia/ollama` | fuera de proceso, sin escritura de estado, sin URL en errores |
| Arnés de evals | `evals/` | 28 casos de ataque, deterministas, en cada PR |

**Lo que NO se construyó, a propósito:** las cinco piezas de adopción (`docs/ia.md` §4.1 y §4.2) necesitan pantalla, y el encargo de esta rebanada dice que las pantallas no se tocan en este tramo. Van al tramo 4, encima de esto.

---

## 2. El apartamiento de la casilla, dicho antes que nada

La casilla de `ETAPAS.md` dice **«Búsqueda FTS5 (BM25)»**. Lo entregado es **el BM25 sin el FTS5**: un índice invertido en memoria con la misma función de ranking y los mismos parámetros por defecto que `bm25()` de SQLite (k1 = 1,2 y b = 0,75).

**Por qué, y no es preferencia técnica:**

1. **FTS5 entra con `modernc.org/sqlite`, y eso es una dependencia.** Hoy el binario se compila con **cero** (`go.mod` sin una sola línea `require`), lo vigila `TestElBinarioNoLlevaNingunaDependenciaExterna`, y una puerta de CI corre la suite entera con `GOPROXY=off`. Añadirla toca `go.mod`, `go.sum`, `DEPENDENCIAS.md` y `dependencias_test.go`, y **ninguno de los cuatro está en la columna de esta rebanada**. Un fichero fuera de columna es un merge rechazado, no una excepción.
2. **El tamaño medido dice que hoy no hace falta.** El corpus transcrito son **28.675 tokens en 321 obligaciones, con 3.099 términos distintos**. Eso cabe en memoria de sobra. FTS5 empieza a valer la pena con los documentos que sube el cliente, que son las piezas 1 y 7 y llegan después.

El contrato del paquete está escrito para que el cambio a FTS5 sea **un adaptador nuevo y no una reescritura**: entra `[]Documento`, sale `[]Resultado` ordenado.

### La dependencia que se pide, con su fila lista para `DEPENDENCIAS.md`

| Módulo | Dónde | Por qué | Licencia |
|---|---|---|---|
| modernc.org/sqlite | adaptadores/busqueda (segundo adaptador, sustituye al índice en memoria cuando entren los documentos del cliente) | FTS5 con BM25 sobre volúmenes que no caben en memoria: un cliente sube su política, su inventario y su Excel de controles, y eso ya no son 28.675 tokens. SQLite sin cgo, así que el binario sigue siendo único y portable | BSD-3 |

**No se pide para ya.** Se pide **cuando entre la pieza 1** (entrevista asistida desde los documentos del cliente), y con dos condiciones que no son negociables porque las impone el propio repositorio: la fila de arriba en `DEPENDENCIAS.md`, y **tocar `dependencias_test.go` a propósito, en el mismo commit**, porque hoy ese test afirma que el binario no lleva ninguna dependencia externa y esa afirmación dejaría de ser cierta. Que ese día haya que tocar un test es justo lo que se busca.

---

## 3. El hallazgo que nació rojo sobre el corpus REAL

**El más importante de esta rebanada, y no salió de una mutación: salió de medir.**

El diseño inicial hacía `HashFuente = sha256(texto)`. Medido contra `paquetes/` el 04-09-2026:

```
fuentes=528 citables=328 no-citables=200 hashes-distintos=495 choques=29
    17a72454bb00... -> [iso27001.7.5.3 iso42001.7.5.3]
    1a1da2c7342d... -> [iso27001.9.1 iso42001.9.1]
    3172334f2efa... -> [nis2.art29_4.notificacion_de_la_incorporacion_en_mecanismos_de_intercambio
                        nis2.art29_4.notificacion_de_la_retirada_en_mecanismos_de_intercambio]
    b2043c689e52... -> [iso42001.ritual.apreciacion_riesgos_ia iso42001.ritual.auditoria_interna
                        iso42001.ritual.evaluacion_impacto_ia ...]
```

**528 obligaciones daban 495 hashes distintos: 29 hashes con más de una obligación detrás, o sea 33 obligaciones tapadas por otra**, y cuál ganaba lo decidía el orden de un mapa.

Y no son erratas del corpus. `iso27001.4.2` e `iso42001.4.2` son dos normas con la misma estructura de cláusulas y el mismo título corto, que es exactamente lo que un paquete referencial guarda. **El choque es normal y va a crecer con cada marco nuevo.**

**Qué habría pasado en producción:** una cita legítima resuelve a un artículo y la pantalla dice *«el artículo X dice esto»* nombrando **otro** artículo. No es una mentira sobre el contenido, es **una mentira sobre la atribución**, y en un producto de cumplimiento cuesta lo mismo.

**El arreglo.** La identidad de una unidad citable es la pareja **(identificador, texto)**, no el texto solo:

```
HashDe(id, texto) = sha256(id + "\x00" + texto)
```

Las dos partes van **dentro de lo firmado** (el paquete se firma entero con Ed25519), así que sigue cumpliendo el invariante 7. El separador es un byte nulo y no un salto de línea porque con `"\n"` la pareja (id `a`, texto `b\nc`) y la pareja (id `a\nb`, texto `c`) hashean igual, y entonces el separador no separa nada; además se comprueba que el identificador no lo contiene.

Medido después del arreglo: **528 fuentes, 528 hashes distintos, 0 choques.**

**Por qué esto vale más que una mutación:** una mutación demuestra que la puerta caza un fallo que tú le metiste. Esto era un fallo **que nadie le metió** y que ninguna mutación habría encontrado, porque nadie sabía que estaba ahí. Lo vigila `TestCadaUnidadCitableDelCorpusRealTieneUnHashSuyo`.

---

## 4. La demostración de la puerta antialucinación

**La cita que no resuelve se descarta:**

```
--- FAIL: TestLaCitaQueNoResuelveSeDescartaYLaQueResuelveSeEnsena/la_cita_inventada_se_descarta
    una cita que NO esta en la fuente ha pasado la puerta.
```

Ese es el mensaje que sale **cuando la puerta se rompe**. Con la puerta puesta, la propuesta

```
Cita: "el responsable debera comunicar el incidente en un plazo de tres dias"
HashFuente: <el del artículo de notificación>
```

se descarta con `ErrCitaNoAparece` y **no llega a pantalla**. La frase es plausible, del mismo registro, sobre el mismo tema y no dice nada falso: es exactamente lo que produce un modelo que parafrasea.

**El control positivo, al lado y en el mismo test:**

```
Cita: "a mas tardar 72 horas despues de que haya tenido constancia de ella"
```

pasa, y lo que la pantalla enseñaría es **el trozo de la fuente con sus saltos de línea**, no lo que tecleó el modelo, con su desplazamiento exacto (`Desde`, `Hasta`) para poder resaltarlo dentro del artículo entero.

**Sin ese control positivo, el descarte de arriba lo pasaría igual un verificador que rechaza absolutamente todo**, y desde el código las dos cosas se leen igual.

### El caso que más me importa del conjunto dorado

`numero-cambiado-dentro-de-una-cita-literal`: todo es literal menos un número, **72 horas pasan a 24**. Un CISO que lea la cita en pantalla la da por buena, y el plazo que se le queda en la cabeza es el equivocado. Contra eso no hay revisión visual que valga; hay comparación de cadenas.

### Y el barrido sobre el corpus real, con sus cardinales

```
corpus real: 528 unidades citables, 528 hashes distintos; 29 textos repetidos
             que tapaban a 33 obligaciones cuando el hash era solo del texto
corpus real: 328 citas literales aceptadas, 187 rechazadas por estrato,
             13 saltadas por texto demasiado corto
309 parejas cruzadas del corpus real, ninguna pasa; 19 saltadas
183590 runas citables del corpus real, 0 marcas combinantes
328 documentos indexados del corpus real
12 citas del corpus real recorren el puente entero, de la consulta al hash verificado
```

**Nació verde sobre el corpus entero salvo el punto 3, y se dice en voz alta.** Las ramas que el dato real no toca (hash malformado, cita ausente, inyección vía documento) las cubren los 28 casos dorados, que sí son sintéticos.

---

## 5. La frontera legal, convertida en mecánica

`docs/ia.md` §3 promete que sobre estrato referencial la IA **no explica el texto y lo dice**. Eso ya no es una promesa sobre el comportamiento de un modelo:

- **La citabilidad se decide por la CLASE del paquete**, no por una lista de marcos en el código. `importado`, `transcrito` y `propio` son citables; `referencial` y `delegado` no. Un paquete referencial nuevo **nace no citable sin tocar una línea de Go**, que es la pregunta de la pasada 1 (*«¿puede un paquete usarlo sin tocar código?»*) contestada que sí.
- **Una clase fuera de rango tampoco es citable.** El valor llega de un JSON de un tercero, y lo desconocido cae del lado restrictivo.
- **El texto no citable ni siquiera entra al índice de búsqueda.** Lo que no entra al contexto del modelo no puede salir parafraseado por ningún lado.

La medida que lo sostiene: los 200 referenciales del corpus tienen **9.223 runas en total, media de 46 y la más corta de 12**. Ahí no hay texto normativo: no cabe.

---

## 6. El invariante 8, campo a campo

| campo | `nil` | vacío-presente | presente y no interpretable |
|---|---|---|---|
| `Opciones.Fuentes` | `ErrSinFuentes` | construye, no resuelve nada | — |
| `Opciones.Admite` | `ErrSinProcedencias` | construye, no admite nada | procedencia inválida → error |
| `Opciones.MinimoCita` | `ErrSinMinimoCita` (cero y negativo) | — | — |
| `Propuesta.HashFuente` | `ErrHashAusente` | `ErrHashAusente` | `ErrHashIlegible` |
| `Propuesta.Cita` | `ErrCitaAusente` | `ErrCitaAusente` (sólo espacios) | `ErrCitaCorta` |
| `Fuente.Procedencia` | el cero es `Ninguna` y está prohibido | — | `Procedencia(99)` → error |
| `Busqueda.tope` | `<= 0` → `ErrSinTope`, **no «todos»** | — | — |
| `Busqueda.consulta` | `ErrConsultaVacia` | `ErrConsultaVacia` | `ErrConsultaIlegible` |
| `PLAZUM_SIN_IA` | sin poner → IA encendida | vacía → IA encendida | `ErrInterruptorIlegible` |
| `Caso.Veredicto` (evals) | `ErrCasoSinVeredicto` | idem | `ErrVeredictoIlegible` |
| `FuenteDorada.Clase` | `ErrClaseIlegible` | idem | idem |

**`Opciones.Fuentes` a `nil` se prohíbe aunque su nada sea restrictiva**, y el motivo es el del lazo local con gosec: un verificador sin fuentes descarta el 100 % de lo que le llega, y **eso desde fuera se ve exactamente igual que «el modelo alucina siempre»**. Confundir *«no se pudo comprobar»* con *«encontró algo»* es el fallo que este repositorio ya se hizo a sí mismo.

**En `adaptadores/busqueda`, la nada de `Nuevo(nil)` NO se prohíbe y se documenta por qué**: en un buscador no existe el «sin restricción» que en un almacén de certificados significa «acepto cualquier CA». No hay comodín que se pueda dejar suelto. Las dos formas se recorren igual, porque *«aquí la nada es inocua»* sólo vale si alguien la comprueba.

### El interruptor, que es el caso más caro

Lo obvio es `os.Getenv("PLAZUM_SIN_IA") != ""`, y está mal en las dos direcciones a la vez:

- `PLAZUM_SIN_IA=0` **apagaría** la IA, que es lo contrario de lo que escribió quien lo puso.
- `PLAZUM_SIN_IA=quiza` la apagaría por casualidad; con un `strconv.ParseBool` cuyo error se tira, la **encendería**.

La peligrosa es la que enciende: un operador que escribió algo raro creyendo que apagaba la IA, y la IA hablando con la red. Así que **vocabulario cerrado y valor no interpretable a error, nunca a defecto**.

---

## 7. Las mutaciones (pasada 2)

**24 mutaciones, 22 cazadas, 1 que no compilaba y 1 que sobrevivió.** El cardinal son las filas de la tabla de abajo, contadas ahí y no de memoria: la primera vez que se escribió decía 20 y 19, y estaba mal en las dos. Todas sobre árbol limpio, con huella antes y después, comprobación de compilación por código de salida de `go vet ./...` y restauración desde la copia.

| # | qué se rompe | resultado |
|---|---|---|
| M1 | la cita ya no se busca en la fuente (`i := 0`) | CAZADA: 3 tests de `ia` + 3 de `evals`, incluido el barrido del corpus real |
| M2 | sin mínimo de cita | CAZADA |
| M3 | la procedencia deja de mirarse | CAZADA: la inyección vía documento sale como ley |
| M4 | el estrato referencial se puede citar | CAZADA, también en el corpus real |
| M5 | el hash vuelve a ser sólo del texto | CAZADA: 4 tests, entre ellos el que nació rojo |
| M6 | el interruptor con el `getenv` obvio | CAZADA: 8 subcasos, los cuatro negativos y los cuatro ilegibles |
| M7 (1.ª forma) | renombrar `prop` a `Prop` en `Verificada` | **NO COMPILABA.** La guarda 3 lo paró: un fallo de build no produce líneas `--- FAIL`, así que su rojo no demuestra nada |
| M7 (2.ª forma) | **añadir** un campo exportado `Cruda` | CAZADA |
| M8 | envolver el error de `http.Client` con `%w` | CAZADA: el centinela sale en el error |
| M9 | el índice deja de filtrar por citable | CAZADA |
| M10 | sin desempate por identificador en `Buscar` | CAZADA |
| M11 | `Admite: nil` deja de estar prohibido | CAZADA |
| M12 | el arnés toma «aceptada» cuando falta el veredicto | CAZADA |
| M13 | un hash presente que no se entiende deja de ser error | CAZADA: 6 subcasos |
| M14 | dos fuentes con el mismo hash entran | **SOBREVIVIÓ.** Ver abajo |
| M15 | un término repetido en la consulta puntúa N veces | CAZADA |
| M16 | el hash de la fuente ya no se recalcula | CAZADA |
| M17a | renombrar `ia.Variable`, con el test leyéndola | CAZADA |
| M17b | renombrar `ia.Variable` **y** escribir el literal a mano en el test | CAZADA, y no era lo esperado. Ver abajo |
| M18 | `ci.yml` exporta `PLAZUM_SIN_IA: "0"` | CAZADA |
| M18b | `ci.yml` exporta `PLAZUM_SIN_IA: "quiza"` | CAZADA |
| M19 | la citabilidad deja de cuadrarse contra la clase | CAZADA (la guarda que trajo la refutación de propiedad, ver abajo) |
| M20 | el LEEME de `evals/` dice un cardinal que no es | CAZADA. Se muta **fuera** de la lista que el propio test lee: se cambia el número del LEEME, no el test |
| M21 | el recorte de la cita se desplaza una runa (se cae el `+1`) | CAZADA. Ver «la comprobación de forma que dejaba pasar el contenido», abajo |

### La propiedad que se intentó tumbar, y cayó

*«El emparejamiento va por el hash, que está dentro de lo firmado, así que una fuente no puede mentir.»*

**Cierta a medias.** El verificador recalcula el hash de cada fuente, o sea que cierra la mentira sobre el **texto**. La pregunta que faltaba era si cierra la del **campo de al lado**, y no la cerraba: una `Fuente` construida a mano con

```go
Fuente{Clase: "referencial", Citable: true, Texto: "<enunciado de un control de pago>"}
```

entraba entera, y entonces el texto de un catálogo privativo sale por pantalla como cita. **Es la misma forma que el agujero del linter legal**, que sólo miraba `texto_legal` mientras el enunciado de un control entraba por cualquiera de los otros veinte campos de texto del formato.

Se cerró comprobando en `Nuevo` que `Citable` cuadra con la clase, con su test escrito **a la vez y no después** (que es la lección de M14) y en las dos direcciones: un referencial que se dice citable **y** un transcrito que se dice no citable, porque un descargo falso también es una mentira. Y una clase mal escrita no puede caer en el valor cero de `corpus.Clase`, que es `importado` y **sí** es citable.

Leer el diff encuentra lo que el autor hizo mal; refutar una propiedad encuentra lo que el autor no pensó.

### La comprobación de forma que dejaba pasar el contenido

**Llegó como aviso del frente de corpus y aterrizó en un agujero real de aquí.** Allí una mutación sobrevivió porque un campo (`cita_del_intervalo`) sólo se comprobaba **por longitud**, así que podía citar un plazo que la norma no dice con todos los dorados en verde. La forma general: *una comprobación que mira la FORMA deja pasar lo que una que mira el CONTENIDO no dejaría.*

Aplicada aquí, la pregunta no era si el verificador resuelve el hash y compara la cita contra el texto real (lo hace, y ningún paso pasa sólo por forma: `pareceHash` va seguido de una resolución en el mapa, y el mínimo de cita va seguido de un `strings.Index` sobre la fuente). Era otra, un piso más abajo:

> El verificador comprueba **la cita**. Lo que acaba en pantalla no es la cita: es **el trozo de la fuente** recortado por dos índices que salen del mapa de normalización.

Un mapa desplazado una runa daría un recorte que empieza media palabra antes, **la cita habría casado igual**, y el único test que miraba esto comparaba `origen[Desde():Hasta()]` con `Cita()` — **los dos salen de los mismos índices, así que es circular y estaba verde por construcción**.

Cerrado por dos sitios:

1. **Dentro de `Verificar`**: el trozo que se va a enseñar, normalizado, tiene que ser exactamente la cita que se acaba de verificar. Si no, se **descarta** con `ErrRecorteIncoherente`, y el mensaje dice que es fallo de plazum y no del modelo. Enseñar un texto que no se ha podido confirmar es lo único que esta puerta existe para impedir, y da igual de quién sea la culpa.
2. **Un barrido sobre el corpus real** que contrasta contra la cita **enviada**, que es el único dato que no viene de los índices: tres recortes por fuente (principio, medio y final, que es donde caen los saltos de línea y la sangría). **984 recortes, todos enseñan exactamente lo verificado.**

M21 lo demuestra: quitando el `+1` del final del recorte, se ponen rojos 9 tests, y el barrido nuevo es el que lo dice por su nombre.

### M14, la que sobrevivió

La guarda de fuentes repetidas estaba escrita en `Nuevo` y **ningún caso la recorría**: apagándola entera, las dos suites seguían en verde. Era **una guarda que no guardaba**, y estaba dentro del código que se escribió para no repetir ese error.

Se escribió `TestDosFuentesBajoElMismoHashNoEntran`, con su control positivo (dos obligaciones **distintas** con el mismo texto sí entran, que es lo que pasa 29 veces en el corpus real y rechazarlo dejaría 33 obligaciones sin poder citarse), y M14 se volvió a correr: **cazada**.

### M17b, la que se esperaba que sobreviviera y no

Escribir el literal `"PLAZUM_SIN_IA"` a mano en el test **debería** dejar la puerta verde tras renombrar la constante, que es el fallo del 28-08-2026 con el renombrado del módulo. No lo hace, y el motivo es la **segunda mitad** de la puerta: el valor que `ci.yml` exporta se pasa por `ia.Apagada()`, que lee la constante de verdad, así que el desajuste sale por ahí aunque el nombre esté escrito a mano. Se anota porque el resultado no fue el previsto y eso hay que decirlo.

---

## 8. Lo que estas puertas NO hacen, dicho antes de que alguien lo suponga

1. **El verificador NO detecta la inyección vía documento.** Si el PDF que sube el cliente contiene *«el artículo 5 obliga a cifrar en reposo»*, esa cita **resuelve**, porque la frase está literalmente en un documento que el sistema tiene. Lo que se hace contra eso es otra cosa: **separar por procedencia**, de modo que la cita de un documento aportado no salga por una pantalla que dice citar la norma. La frase inyectada se puede enseñar como lo que es, texto del documento del cliente, y entonces la persona que confirma la ve.
2. **El verificador NO juzga si el `Diff` propuesto es correcto.** El diff lo escribe el modelo y lo confirma una persona. Lo que se garantiza es que **el argumento con el que se justifica es texto real**.
3. **El verificador NO pliega composición Unicode.** Una cita en NFD contra una fuente en NFC no casa, y eso es correcto (son caracteres distintos) pero sólo es inocuo mientras el corpus lleve los acentos precompuestos. Medido: **0 marcas combinantes en 183.590 runas citables**. Hay una puerta (`TestElCorpusRealNoTraeMarcasCombinantes`) que se pone roja el día que entre corpus descompuesto, para **decidir** en vez de descubrirlo en producción.
4. **El mínimo de cita es un techo, no una prueba semántica.** 24 runas impide la palabra suelta; no impide una cita literal irrelevante de 30 runas.
5. **El adaptador de modelo no tiene presupuesto ni transcript al ledger.** Son los puntos 4 y 5 del arnés de `docs/ia.md` y no están: hacen falta el puerto con `context.Context` y la escritura al ledger, que es de otra columna.

---

## 9. La propuesta de cambio de puertos, que este worktree NO ha aplicado

`docs/puertos-propuestas.md` no es de esta columna, así que la propuesta va aquí y el código se construyó **contra el interfaz de hoy**, con un `TODO(puertos)` en `adaptadores/ia/ollama/ollama.go`.

Firma actual:

```go
type Asistente interface {
    Proponer(tarea string, contexto []byte) (Propuesta, error)
}
```

**Tres cambios pedidos, por orden de lo que duele:**

1. **`context.Context` como primer parámetro.** Sin él no hay cancelación ni plazo, así que una petición a un modelo que se queda pensando bloquea el handler que la lanzó hasta que el cliente se aburre. Hoy se tapa con un `http.Client{Timeout: 120s}`, que es un tope y no una cancelación: si el operador cierra la pestaña, la petición sigue viva. Es el único de los tres que es un **problema hoy**.
2. **Devolver `[]Propuesta` y no una.** La entrevista asistida (pieza 1) propone **una respuesta por pregunta** desde el mismo documento; con una propuesta por llamada, son 19 llamadas al modelo para una entrevista de 19 preguntas.
3. **`Propuesta` necesita dos campos más**: `Procedencia` (de dónde sale la cita, para que la pantalla no presente igual un artículo del BOE que una frase del PDF del cliente) y un contador de coste en tokens, que es el punto 4 del arnés (*presupuesto por tarea, visible para el operador*) y hoy no tiene dónde viajar.

**Lo que NO se pide:** quitar `HashFuente` ni `Cita`. Son la puerta entera.

---

## 10. Los huecos, con su cardinal

| hueco | cardinal | por qué importa |
|---|---|---|
| Piezas de adopción sin construir | **5 de 5** (1, 2, 3, 4 y 7 de `docs/ia.md`) | necesitan pantalla, y esta rebanada no toca pantallas a propósito. Tramo 4 |
| Puntos del arnés de `docs/ia.md` sin implementar | **4 de 8** (presupuesto, transcript al ledger, consentimiento de nube, evals publicados en release) | los tres primeros necesitan puerto y ledger; el cuarto necesita el paso de CI de nightly |
| Conjuntos dorados escritos | **1 de 3** (citas, 28 casos). Faltan extracción de obligaciones (50) y contradicciones (20) | los dos que faltan **necesitan modelo**, así que van a nightly y no a cada PR |
| Documentos del cliente sin ingesta | **0 adaptadores** | `Procedencia.Aportado` existe, está probada y no hay quien construya una `Fuente` desde un PDF. Es la pieza 1 |
| Búsqueda con embeddings | **0** | la casilla dice «embeddings opcionales vía Ollama». No entra: el BM25 sobre 3.099 términos ya encuentra, y los embeddings sin medir que hagan falta son una dependencia y un modelo más |
| `adaptadores/busqueda` sin fuzzing | **0 objetivos** | el tokenizador y `Buscar` reciben texto de fuera. `paquetes/`, el ledger y el verificador tienen fuzzing; esto no. **P2** |
| Cobertura de los tres paquetes nuevos | **sin medir** | no hay puerta de cobertura para ellos. Lo pide el paso de CI del §11 |

---

## 11. Los pasos de CI que se piden al integrador

`ci.yml` no es de esta columna. **Estos son los pasos exactos, para meter en un commit propio cuando esta rama esté dentro.**

### 11.1. El paso que YA EXISTE y hay que RETOCAR

El paso `la suite entera con la IA desactivada (invariante 9)` existe desde antes y **su suelo se queda corto**: esta rama añade casos y el mínimo de 700 deja de medir. Hay que subirlo **en el mismo commit que hace crecer la suite**, que es la disciplina declarada.

```yaml
      - name: la suite entera con la IA desactivada (invariante 9)
        shell: bash
        env:
          PLAZUM_SIN_IA: "1"
        run: |
          set +e
          set -uo pipefail
          source .github/puerta.sh
          puerta "suite completa sin IA" 800 ./...
          cerrar_puertas
```

El número exacto lo tiene que sacar el integrador de la ejecución en `main` después del merge, **no de aquí**: el suelo se pone en el número real menos un margen, y ese número depende de qué otras rebanadas hayan entrado. El mismo retoque hace falta en los otros dos pasos que corren `./...` con suelo 700.

### 11.2. El paso NUEVO que se pide

Por el mismo motivo que lo tienen SCIM y el anclaje: **el suelo de la suite completa dice que el repo tiene N casos, no que ESTOS sigan estando.** Si alguien borra el fichero de los adversariales del verificador, o el conjunto dorado, el suelo global lo absorbe sin pestañear y la puerta antialucinación se queda sin vigilancia.

```yaml
      # LA PUERTA ANTIALUCINACION, CON PUERTA PROPIA.
      #
      # Va aparte por el mismo motivo que el anclaje y SCIM: el suelo de la
      # suite completa dice que el repo tiene N casos, no que ESTOS sigan
      # estando. Si alguien borra el conjunto dorado de citas o los
      # adversariales del verificador, el suelo global se lo traga y la unica
      # pieza que decide si una salida de modelo llega a una pantalla se queda
      # sin vigilar.
      #
      # Y pesa mas que en ningun otro sitio porque es la puerta que separa
      # "propuesta con cita verificada" de "parrafo inventado con la cara de una
      # cita". Es lo unico que el comprador puede comprobar el mismo en dos
      # minutos.
      #
      # Corre SIN RED y SIN MODELO a proposito: el verificador es un sha256 y
      # una comparacion de cadenas, asi que su conjunto dorado cabe en cada PR.
      # Los evals que necesitan modelo llegan despues y van a nightly.
      #
      # El suelo es el numero real de hoy menos un margen.
      - name: puerta antialucinacion (verificador de citas, busqueda y conjunto dorado)
        shell: bash
        env:
          GOPROXY: "off"
        run: |
          set +e
          set -uo pipefail
          source .github/puerta.sh
          puerta "verificador de citas, busqueda y evals" 140 \
            ./adaptadores/ia/... ./adaptadores/busqueda/... ./evals/...
          cerrar_puertas
```

El suelo de **140** sale del número real medido el 04-09-2026 con `GOPROXY=off`, **155 casos ejecutados**, menos un margen. El integrador debería recontarlo en `main` después del merge.

**Y con el paso entra su cuenta:** `comprobar.sh` declara `PUERTAS_ESPERADAS`, y `comprobar_test.go` exige que cuadre con las puertas que hay en CI **en los dos sentidos**. Añadir este paso **obliga a subir `PUERTAS_ESPERADAS` de 24 a 25 en el mismo commit**, o el lazo local se pone rojo. `comprobar.sh` tampoco es de esta columna.

El suelo de 100 sale de la ejecución local del 04-09-2026; el integrador debe recontarlo en `main` y ponerlo en el número real menos un margen.

---

## 11.bis. El lazo local, con su código real

`./comprobar.sh` sobre esta rama, ejecutado en segundo plano a fichero (nunca por `tail` ni por un pipe, que devuelven el código de `tail` y siempre es 0):

```
21 puertas, todas en verde.
govulncheck ok.
gosec ok.
staticcheck ok.
3 puertas saltadas en esta maquina (dicho arriba, con el motivo).

COMPROBACION EN VERDE: 24 puertas leidas de los workflows,
21 ejecutadas aqui, mas formato, vet y build,
mas 3 herramientas de seguridad leidas de ci.yml,
3 ejecutadas aqui.
```

**Código de salida: 0**, leído de `$?` del propio `comprobar.sh`.

**Las 3 puertas saltadas y su motivo, distinguiendo «no se pudo ejecutar» de «encontró algo»:** son las tres de `-race`, que exigen cgo, y esta máquina es Windows sin compilador de C. **No se ejecutaron; no es que salieran limpias.** Son `suite completa con detector de carreras`, `superficies y secretos con detector de carreras` y `actualizador con detector de carreras`, y las tres corren en CI.

La suite completa ejecuta **2.672 casos** (suelo declarado 700), de los cuales **155 son de los tres paquetes nuevos**.

## 11.ter. Qué casillas de `ETAPAS.md` toca esto, y cuáles NO se pueden marcar

`ETAPAS.md` no es de esta columna: las casillas las mueve quien integra, cuando el trabajo ya está dentro. Se dice aquí qué se puede marcar y qué no, para que no haya que decidirlo a ojo.

**«Búsqueda FTS5 (BM25) sobre el corpus transcrito y sobre los documentos que sube el cliente; embeddings opcionales vía Ollama»** — **NO se marca.** Y no es prudencia: la casilla nombra tres cosas y hay una hecha.

- BM25 sobre el corpus transcrito: **hecho**, 328 documentos indexados.
- FTS5: **no**, y el porqué está en el §2. Es una decisión de dependencia, no de código.
- Sobre los documentos que sube el cliente: **no**, no hay ingesta de documentos.
- Embeddings vía Ollama: **no**, y se argumenta en el §10 que hoy no hacen falta.

Una casilla escrita como puerta no se cierra a medias. Marcarla haría planificar sobre una capacidad que no existe, que es exactamente lo que el barrido del 04-09-2026 encontró al revés.

**«Verificador de citas por hash, determinista, corriendo en cada PR con sus adversariales»** — **se marca cuando entre el paso de CI del §11.2, no antes.** El verificador existe, es determinista y tiene sus adversariales (28 casos dorados más el barrido sobre el corpus real). Lo que falta de la casilla es literalmente *«corriendo en cada PR»*, y eso es un paso de `ci.yml` que esta rebanada no puede escribir. **Hoy corre dentro del suelo global de la suite completa, y eso no es lo mismo**: si alguien borra el conjunto dorado, el suelo global se lo traga.

**«PUERTA: el camino completo en verde con `PLAZUM_SIN_IA=1`»** — sigue sin cerrarse, y ahora por un motivo mejor que antes. Antes era casi vacía porque no había adaptador; ahora hay adaptador y el interruptor lo apaga de verdad, medido en bytes. Lo que falta para cerrarla es que el **camino completo** (las seis pantallas) esté dentro, y eso llega con las piezas de adopción del tramo 4.

## 12. Mis errores en esta rebanada

1. **La primera forma de M7 no compilaba.** Renombrar `prop` a `Prop` rompe cuatro usos, y un fallo de build no produce líneas `--- FAIL`. La guarda de compilación lo paró; sin ella, ese rojo habría viajado al informe como «cazada». Es la trampa número 3 de la mutación, cometida otra vez.
2. **M14 sobrevivió**, y la guarda que no guardaba estaba en mi propio código. Enumeré las guardas al escribirlas y no comprobé que cada una tuviera un caso que la recorriera.
3. **Escribí el hash sobre el texto solo** y lo di por bueno hasta medirlo. La corrección la trajo el corpus, no el diseño.
4. **`adaptadores/ia/frontera_test.go` cableaba la ruta del módulo** en tres literales, y lo cazó una puerta que ya existía (`TestNadieCableaLaRutaDelModulo`). El arreglo obliga a componerla con `internal/modulo.Ruta()`.
5. **`parser.ParseDir` está obsoleto desde Go 1.22** y staticcheck es puerta bloqueante. Lo escribí igual y hubo que rehacerlo con `os.ReadDir` + `ParseFile`.
6. **Escribí dos cadenas Unicode «iguales»** (una NFC y otra NFD) confiando en que se distinguirían al leerlas. No se distinguen. Ahora van con escapes `ó` y `́`, y hay un caso que comprueba que las dos son distintas byte a byte, porque si no el test estaría comparando una cosa consigo misma.

---

## 12.bis. Pasada 3, contra el comprador

Un CISO de 200 empleados abre esto a las 9 de la mañana. **Esta rebanada no tiene pantalla a propósito**, así que la pregunta no es «¿llega al valor?» sino **«¿qué puede comprobar él mismo?»**. Tres hallazgos, con prioridad.

**P1. Los números del eval no salen a ningún sitio que un comprador lea.** `docs/ia.md` §2 punto 8 promete *«evals publicados en cada release, con modelo y versión fijados»*, y el hito es *«el primer GRC que publica la precisión de su IA»*. Hoy los 28 casos y su resultado viven **sólo en la salida de `go test -v`**. Un comprador tendría que clonar el repositorio y saber qué comando correr. Lo que falta es que el paso de CI del §11.2 escriba su recuento a un artefacto y que la release lo recoja; el segundo trozo es de la columna de release.

**P1. No hay ninguna orden de `plazum` que enseñe esto funcionando.** El argumento de venta es *«la IA de los competidores se inventa el texto de una cláusula de ISO; la nuestra no puede»*, y para verlo hoy hay que leer un test. Bastaría `plazum citar --hash <h> --cita "<texto>"` diciendo verificada o descartada con su motivo, y una `plazum buscar "<consulta>"`. **`cmd/plazum` no es de esta columna** y por eso no se ha hecho; se deja escrito porque es barato y es lo que convierte el argumento en algo comprobable en un minuto.

**P2. El descarte por estrato referencial se le dice bien a un programador y no a un CISO.** El mensaje dice *«el marco X es de estrato referencial, o sea que plazum no distribuye su texto y por tanto no lo tiene»*. Es exacto y usa una palabra del proyecto, no del comprador. Cuando llegue la pantalla habrá que reescribirlo con las palabras del catálogo (`adaptadores/catalogo/cadenas`, que tampoco es de esta columna), y **la reescritura no puede aflojar lo que dice**: la frase tiene que seguir diciendo que el texto no está y que no se inventa, que es el argumento entero.

**Lo que sí puede comprobar hoy** cualquiera que clone el repositorio, sin instalar nada y sin red:

```bash
GOPROXY=off go test ./adaptadores/ia/... ./adaptadores/busqueda/... ./evals/... -v
PLAZUM_SIN_IA=1 go test ./... -count=1
```

El segundo es el que convierte *«el núcleo es determinista»* en un hecho en dos minutos, y es el que hay que poner en el README cuando llegue la venta.

## 13. Lo que ESTA rebanada encontró y NO es suyo — YA ARREGLADO

> **Cerrado el 04-09-2026 por el integrador, en `main`, y comprobado desde aquí.** Lo encontraron dos frentes a la vez, cada uno por su lado, y el arreglo fue el que se pedía abajo. Se deja escrito entero porque el porqué vale más que el arreglo.
>
> Comprobado en este worktree después de rebasar: `.github/mutar.sh preparar` crea su depósito en `.../worktrees/agent-.../mutaciones`, `comprobar` caza M21 y `restaurar` deja el árbol limpio. **El ciclo entero, en el sitio donde se usa.**

**`.github/mutar.sh` no funcionaba dentro de un worktree**, y el tramo 3 entero se construye en worktrees.

```
$ .github/mutar.sh preparar adaptadores/ia/verificador.go
mkdir: cannot create directory '.git': Not a directory
```

El script fija su depósito en `.git/mutaciones`, y en un worktree de git **`.git` no es un directorio: es un fichero** con la línea `gitdir: ...`. El `mkdir -p` falla y el script no arranca, así que **las cuatro guardas de la mutación no las tiene ninguna de las cuatro rebanadas de este tramo**, que es justo donde más falta hacen.

**El arreglo es de una línea**, y `.github/mutar.sh` es de la columna de la rebanada 0:

```bash
deposito="$(git rev-parse --git-dir)/mutaciones"
```

`git rev-parse --git-dir` devuelve el directorio real en los dos casos, con checkout normal y con worktree.

**Las mutaciones M1 a M20 de esta rebanada se hicieron con un equivalente fuera del repositorio**, con las mismas cuatro guardas: árbol limpio, huella antes y después, `go vet ./...` por código de salida, y restauración desde la copia. **No se metió una copia del script en el árbol a propósito**: una segunda copia es la segunda lista que este repositorio lleva catorce hallazgos prohibiendo. **M21 ya se corrió con el `mutar.sh` de `main`, después de rebasar.**

**La lección, que es la que vale y no es del script:** la herramienta escrita para que las mutaciones no se hagan a ojo estaba rota **justo donde se usa**, y funcionaba sólo en el checkout donde casi no se usa. Es la regla que este repositorio ya tiene escrita — *«una puerta se demuestra en el shell en el que CORRE, no en el que la escribes»* — con cara nueva: **una herramienta se demuestra en el árbol en el que se usa, no en el que se escribe.** Y el modo de fallo fue el amable: `mkdir` gritó. El caro habría sido que el depósito acabara en un sitio que existe pero no es el suyo, y que `restaurar` copiara encima de otra cosa.
