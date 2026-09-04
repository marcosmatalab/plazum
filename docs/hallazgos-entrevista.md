# Hallazgos de la entrevista de alcance (frente C, 03-09-2026)

Este documento es del frente que construyó la **revelación progresiva** de la
entrevista. Recoge lo medido, lo construido, lo que quedó fuera y, sobre todo,
los dos huecos que la medición destapó y que **no son de la pantalla**.

Todas las cifras de aquí salen de una puerta. Las que no, lo dicen.

---

## 1. El número, antes y después

| | antes | después |
|---|---|---|
| TTFV del tramo recorrible del camino | **18m56s** | **11m36s** |
| Presupuesto de la casilla D11-e | 15m0s | 15m0s |
| Preguntas que pinta `/alcance` | 41 contadas (42 reales) | **19** |
| Coste humano de la entrevista | 14m25s (76 % del total) | 7m5s (61 %) |
| Pasos del camino que se pueden recorrer | 3 de 6 | 3 de 6 |

**La casilla D11-e pasa a cumplirse en el tramo que se puede recorrer.** El
techo declarado de 20 minutos se borró: la propia puerta lo exigió al bajar el
total por debajo del presupuesto, que es para lo que tenía dientes por abajo.

**No se tocó ninguna de las tres constantes de coste humano.** Siguen en 45 s
por lectura de pantalla, 20 s por respuesta y 90 s por orden de terminal. Bajar
el coste por pregunta a 12 s habría hecho pasar la puerta el mismo día sin haber
arreglado nada, y es la tentación que este número existe para no aceptar.

**Y sigue siendo el número de la mitad del camino.** Tres de los seis pasos
(acta, revisión de accesos, escalado) contestan 401 en un binario recién
descargado porque no hay forma de abrir sesión. Eso lo arregla otro frente; el
cardinal sigue puesto en `PasosQueExigenSesion = 3` y la puerta lo compara por
igualdad exacta en los dos sentidos.

### El desglose por paso, tal como sale de la puerta

```
  MODELO: TTFV = T_maquina + T_humano; lectura 45s, respuesta 20s, orden 1m30s
  T_maquina 1.087s  = binario 891ms + arranque 189ms + peticiones 7ms
  T_humano  11m35s
  TOTAL     11m36s  (presupuesto 15m0s)
  pasos alcanzados 3 de 6; exigen sesion 3
  paso alcance      /alcance       codigo 200  preguntas 19  ordenes 0  coste 7m5s
  paso calendario   /calendario/   codigo 200  preguntas  0  ordenes 2  coste 3m45s
  paso derivacion   /controles     codigo 200  preguntas  0  ordenes 0  coste 45s
  paso acta         /acta/         codigo 401  ...  coste 0s
  paso uar          /uar/          codigo 401  ...  coste 0s
  paso escalado     /escalado/     codigo 401  ...  coste 0s
```

**El nuevo cuello de botella tiene nombre y no es la entrevista**: son las **dos
órdenes de terminal** que el calendario exige en una instalación recién hecha
(`plazum alcance` y `plazum serve --alcance`), 3m45s, el 32 % del total. Cada
una es una salida del producto y un rearranque del servidor. Quien quiera bajar
de once minutos tiene que atacar eso, no la entrevista.

---

## 2. El hueco que destapó la medición: 23 preguntas que no deciden nada

**23 de las 42 preguntas del corpus instalado no las requiere ninguna
obligación.** Declaran `desbloquea` apuntando a obligaciones que existen, y
ninguna de esas obligaciones las nombra de vuelta en su `preguntas`.

Es el invariante 7 en su forma de siempre: **el enlace se declara en las dos
puntas y el linter sólo recorre una.** Comprueba que lo que una pregunta dice
desbloquear existe; no comprueba que la obligación lo confirme. Y la que decide
es la que no se recorre: toda la derivación de la pantalla evalúa contra
`Obligacion.Preguntas`, y `Desbloquea` sólo ordena la entrevista.

Consecuencia medida antes de tocar nada: la entrevista pintaba en segunda
posición una pregunta anunciada como «decide 14 obligaciones» cuya respuesta no
movía **ninguna**. A 20 s por pregunta, las 23 costaban **7m40s** del TTFV, más
de la mitad de la entrevista entera.

### El reparto, por si sirve para repartirlo

```
12  del marco español de seguridad nacional (las seis dimensiones, por duplicado:
    una tanda para la información y otra para el servicio)
 3  del marco de inteligencia artificial de la UE
 3  del marco de gestión de IA
 5  una cada uno, de otros cinco marcos
```

Que el bloque grande sean **doce dimensiones que nadie requiere** es un dato en
sí: no son doce erratas sueltas, es una plantilla entera del paquete escrita en
una dirección y no en la otra. Se cierra de una vez o no se cierra.

(los identificadores no se escriben aquí; el invariante 2 los prohíbe en código
y esta lista se saca en un segundo con `corpus.PreguntasQueNadieRequiere`).

### No es el mismo hueco que las «20 preguntas huérfanas» ya contadas

`docs/pendientes.md` cuenta **20 preguntas cuyo atributo no usa ninguna regla**
de aplicabilidad. Es un hueco distinto y en otra capa: aquél mide si la
respuesta llega al **motor Datalog**, éste mide si llega a la **pantalla**. Se
solapan pero no coinciden, y una pregunta puede tener uno sin el otro.

**Los dos juntos son el mismo diagnóstico de producto**: hay preguntas en el
corpus que se hacen por costumbre y no condicionan nada. Cada una es una de dos
cosas y las dos exigen leer la norma: **le falta la regla** (o le falta el
enlace desde la obligación) o **la pregunta sobra**.

### Lo que se hizo y lo que NO

- **Se hizo**: `corpus.PreguntasQueNadieRequiere` (`nucleo/corpus/enlace_pregunta.go`),
  y un trinquete de igualdad exacta en los dos sentidos sobre el corpus real.
- **NO se metió en el linter.** Un linter que rechaza 23 preguntas reales deja el
  corpus publicado sin cargar, y arreglarlas es trabajo de `paquetes/`, que no es
  columna de este frente. Se mide y se cuenta; cerrarlo es de quien escriba corpus.

---

## 3. La revelación progresiva: qué es y por qué no puede esconder de más

La regla se calcula, no se cablea, y sale de dos cosas que ya estaban en el
modelo: qué preguntas nombra cada obligación y qué se ha respondido ya.

Una pregunta se **enseña** mientras pueda cambiar algo. Se deja fuera cuando se
puede demostrar que no, y sólo hay dos formas:

- **nadie la pide**: ninguna obligación la nombra. Su respuesta no entra en el
  veredicto de nada.
- **ya decidida**: todas las obligaciones que la nombran están ya decididas por
  respuestas anteriores.

Y una pregunta **respondida no se esconde nunca**, pase lo que pase con la
clasificación: esconderla borraría de la pantalla lo que alguien acaba de
contestar y dejaría su respuesta sin forma de deshacerse que no sea editar la
dirección a mano.

### La respuesta a «¿puede esconderse una pregunta que hacía falta?»

**No, y se demuestra, no se afirma.** `evaluarControl` lee la respuesta de una
pregunta exclusivamente a través de `Fila.Requiere`. Con eso:

- si **nadie la pide**, no está en ningún `Requiere` y ninguna evaluación la
  consulta;
- si está **ya decidida**, las obligaciones que la nombran están en *aplica* (y
  entonces todas sus preguntas están respondidas que sí, incluida ésta, que por
  tanto no se esconde) o en *no aplica* (y entonces la rama de negativas gana
  sobre todo lo demás en `evaluarControl`: ningún «sí» posterior las devuelve).

La demostración no se queda en el comentario. `TestEsconderUnaPreguntaNoPuedeCambiarNingunVeredicto`
la ejecuta sobre el **corpus real**: por cada uno de seis estados de respuestas,
toma **cada** pregunta escondida, la contesta que sí y que no, y exige que el
veredicto de las 528 obligaciones **y** el de los entregables salgan idénticos.
234 parejas (pregunta escondida, respuesta) probadas; ninguna mueve nada.

**Nació verde sobre el corpus real, y se dice**: la propiedad se demostró antes
de escribir el código, así que la puerta llega detrás. Lo que la sostiene es la
mutación, anotada en el commit.

### Lo que se esconde se ve, con su cardinal

La lista corta dice cuántas ha dejado fuera y enlaza a la lista entera
(`/alcance?ver=todas`), donde cada pregunta dormida lleva escrito **por qué** lo
está. Todo con navegación del servidor: un parámetro en la dirección, ningún
JavaScript, ningún `display:none`, ningún `<details>`. La página larga se
comparte y se marca igual que la corta, y responder desde ella no te devuelve a
la corta.

**El progreso sigue contando sobre el total del corpus** («has respondido 0 de
42»), no sobre las visibles. Contar sólo las visibles convertiría «19 de 19» en
una entrevista aparentemente terminada con 23 preguntas sin mirar, que es la
trampa más fácil de todas y tiene puerta propia.

### El parámetro `ver` y la tercera forma de la nada

`ver` es opcional y tiene **tres** casos, no dos (invariante 8):

| lo que llega | qué se hace |
|---|---|
| ausente | la lista corta. Es el valor por defecto y es legítimo: no se pidió nada |
| `ver=todas` | la lista larga |
| `ver=`, `ver=basura`, `ver` repetido | **404**. Es un dato que hay y no se entiende |

Tomar lo tercero por la nada sería inventarse un valor. Se contesta 404 y no 400
porque la dirección pedida no es ninguna página de esta superficie, y así se
reutiliza el error que ya existe en vez de inventar una clave nueva.

---

## 4. Un fallo de la propia medida, encontrado por el camino

`ttfv_camino_test.go` contaba las preguntas con `<li class="pregunta"`, **con la
comilla de cierre**, así que no casaba con la pregunta sugerida, que se pinta
`class="pregunta sugerida"`. **La medida venía saliendo con una pregunta de menos
desde que existe**, y nadie lo notó porque el error iba en la dirección cómoda:
veinte segundos menos en el total.

Se descubrió al bajar la entrevista a 19 y no cuadrar el 19 con el 18 que decía
la página. Corregido, la línea base honesta de antes de este frente es
**19m16s**, no 18m56s.

La lección es la de siempre: un patrón que casa con el atributo entero se rompe
en silencio el día que alguien añade una clase, y romperse en silencio hacia
abajo es peor que romperse.

---

## 5. Lo que este frente NO hizo, con su cardinal

- **6 claves de catálogo sin redactar en el catálogo de verdad.** Están
  publicadas por `ClavesDeCatalogo()` y redactadas en el borrador de
  `superficies/pantallas`, pero `adaptadores/catalogo/cadenas/` es columna del
  **frente A** según `.github/frontera.sh`. **Mientras no entren, 6 puertas
  están en rojo por esta única causa, y en una instalación real la pantalla
  enseña las claves crudas** (comprobado a mano contra `plazum serve`: donde
  debería decir «23 preguntas no deciden nada todavía» pone
  `alcance.dormidas.titulo`). La lista, con su español y su inglés, va en el
  informe. **Es lo primero que hay que cerrar al integrar.**

  Que la interfaz y sus cadenas caigan en columnas distintas no es un frente
  desobediente: es la partición, y la propia `frontera.sh` dice qué hacer con
  eso («se resuelve decidiendo de quién es el fichero, no fusionando y viendo
  qué pasa»). Se dice aquí para que se decida y no se repita.
- **1 línea del workflow de accesibilidad.** El auditor de axe recorre las rutas
  de las pantallas y no `/alcance?ver=todas`, así que la lista larga (con el
  motivo por pregunta) **no está auditada**. La ruta corta sí, y con las 23
  dormidas del corpus real, o sea con el bloque nuevo pintado. `.github/` no es
  columna de este frente.
- **23 preguntas del corpus sin decidir.** Ver el punto 2: cada una exige leer la
  norma y `paquetes/` no es columna de este frente.
- **El linter no gana la comprobación de la dirección que falta.** Ver el punto 2.
- **Nada sobre los 3 pasos que contestan 401.** Otro frente.
- **El segundo cuello de botella, las 2 órdenes de terminal del calendario
  (3m45s, 32 % del TTFV).** `superficies/calendario` y `cmd/plazum` no son
  columna de este frente.
- **Nada que impida bajar las tres constantes de coste humano.** No hay puerta
  que vigile `CosteDeResponderUnaPregunta`: quien quiera aprobar el TTFV
  bajándola de 20 s a 12 s puede, y nada se pone rojo. Se deja dicho porque es
  la trampa que este frente tenía delante y no tomó; ponerle puerta exige una
  segunda copia del número, que es su propio problema.

---

## 6. La pasada del comprador, cronometrada contra el producto de verdad

Se levantó `plazum serve` con el corpus publicado y se recorrió la entrevista
siguiendo los enlaces de la propia página, que es lo que hace un operador.

- **La entrevista corta se termina en 19 respuestas.** En cada paso la página
  marca exactamente una pregunta como «empieza por esta» y responderla lleva a
  la siguiente. Nunca se queda sin sugerencia a medias.
- **La derivación arranca en «te aplican 394» con cero respuestas.** Es la
  semántica conocida (una obligación sin preguntas alcanza a todo el mundo) y ya
  estaba anotada en `docs/pendientes.md` con 312; con el corpus de hoy son 394.
  **Es el primer número que ve un comprador y sigue siendo el más confuso.**
- **La primera respuesta vale 79 obligaciones** (394 → 473). O sea que el primer
  calendario que depende de lo que tú contestas está a **una** pregunta, no a
  cuarenta y dos. Ésa es la respuesta a «cuántas preguntas hacen falta para el
  primer calendario no vacío»: **cero** para lo que alcanza a todo el mundo,
  **una** para lo primero que sale de tus respuestas, **19** para agotar todo lo
  que hoy se puede decidir.
- **Y lo que ve hoy de verdad, con el catálogo sin las seis cadenas**, es
  `alcance.dormidas.titulo` y `alcance.dormidas.ver` en crudo al pie de la
  entrevista. Es feo y es exactamente el defecto que la puerta del inventario
  existe para gritar.

---

# La entrevista aprende a preguntar valores (rebanada 3 del tramo 2, 04-09-2026)

Cuaderno de la rebanada 3 del tramo 2. Rama `tramo2/valores`, nacida sobre
`origin/tramo2/puente`.

Lo que se cierra aquí, en una línea: la pantalla de Alcance sólo sabía mandar
`si` y `no`, así que una pregunta que pide un valor se contestaba que sí y ese
sí no llegaba al motor. El resultado no era una pantalla incompleta, era un
alcance corto, o sea obligaciones que no aparecían presentadas como si no
tocaran.

## Las cifras, medidas y no recordadas

Todas salen de ejecutar el binario o una puerta sobre el corpus instalado, y
todas están congeladas en un test para que no se muevan en silencio.

### El corpus, contado por mí

Casando por (paquete, entidad, atributo), con cero preguntas sin casar:

| dato | cardinal |
|---|---|
| preguntas de la entrevista | 68 |
| booleanas | 33 |
| piden un valor | 35 (28 enumerado, 4 texto, 3 fecha) |

Cruzado con la forma del puente que declara cada atributo:

| forma | cardinal | quién la manda |
|---|---|---|
| `afirma_si` | 8 | un sí, y ya se mandaba |
| `afirma_si_valor` | 19 | un sí, y ya se mandaba |
| `con_valor` | 25 (23 enumerado, 2 texto) | **lo que esta rebanada añade** |
| `no_llega_al_motor` | 16 | nadie, y está escrito con su motivo |

**El encargo decía «31 de 42» y el corpus con el que trabajo dice otra cosa**,
porque la rebanada 2 lo movió por debajo: de 42 preguntas a 68 y de 1 paquete
con puente a los 21 con reglas. Las que se perdían por falta de valor no son 31
ni 14: son **25**.

### Cuánto llega al motor

Puerta: `cmd/plazum/entrevista_completa_test.go`, que contesta la entrevista
entera con lo que declara cada atributo (no con una lista escrita al lado) y la
pasa por el exportador de verdad.

```
entrevista entera: 68 preguntas contestadas, 52 hechos, 44 de ellos con valor
solo si/no:        27 hechos. 25 preguntas no tenian por donde llegar
```

**De 27 a 52 de 68.** Las 16 que faltan son los callejones declarados: se
recogen, se pintan y no afirman nada a propósito, con su motivo escrito en el
paquete.

### El CISO de la SaaS española de 200 personas

Escenario: privada, proveedora del sector público, trata datos personales, usa
nube, desarrolla software, tiene ISO 27001, no es financiera ni de
criptoactivos ni de productos sanitarios. Contesta todo lo que le toca y deja
sin contestar lo que no.

| cómo contesta | respuestas | hechos | obligaciones alcanzadas |
|---|---|---|---|
| sólo las 11 booleanas que tenían sentido como sí/no | 11 | 5 | **8** |
| las 60 en la pantalla de antes (los valores, con un «sí») | 60 | 12 | **90** |
| las 60 en la pantalla de ahora, con sus valores | 60 | 34 | **108** |

El **8** de la primera fila es el número del encargo, y sale de contestar sólo
lo que en la pantalla de antes era contestable sin decir tonterías. La
comparación honesta es la de las dos últimas filas, que son las **mismas 60
respuestas**: 22 de ellas se tiraban por el camino, y eso son **18 obligaciones
que no aparecían**.

Se mide así, y se puede repetir:

```
plazum alcance --respuestas "<la entrevista>" --sujeto sis \
               --organizacion "Acme SaaS SL" --salida ciso.json
plazum calendario --alcance ciso.json          # «N alcanzados por la aplicabilidad»
```

## Las decisiones, con su porqué

### Ningún desplegable, y no es estética

El encargo pedía un desplegable con «sin contestar» por defecto. Se ha hecho
otra cosa y se dice en voz alta: **un enumerado se pinta como un enlace por
valor**, igual que los botones de sí y no que esta superficie lleva pintando
desde el principio.

El requisito («el valor por defecto tiene que ser sin contestar, y sin contestar
no afirma nada») se cumple, y se cumple más fuerte. Un desplegable con una
primera opción vacía resuelve el problema **por convención**: hay que acordarse
de ponerla, de que vaya primera y de que su valor sea vacío, en cada sitio que
pinte uno. Con enlaces se resuelve **por construcción**: «sin contestar» no es
una opción que haya que recordar poner, es no haber pulsado ninguna, y la
ausencia del parámetro en la dirección es literalmente la nada. No hay estado
por defecto que afirme porque no hay estado por defecto.

Texto, entero y fecha sí necesitan campo libre, y ahí el valor cero del campo es
la cadena vacía, que tampoco afirma.

### Una pregunta con valor no tiene «no», y eso es una corrección

Con la pantalla de antes, un «no» a «¿qué categoría alcanza el sistema según el
anexo I?» escondía de golpe las 84 filas que dependen de la categoría. Era una
vía para absolver por una respuesta que nadie puede dar en serio. Ahora no
existe, y las obligaciones se quedan visibles hasta que el operador diga cuál.

Absolver de más es el error caro: el que acusa lo corrige quien lee, el que
absuelve lo descubre el inspector.

### Las tres formas de la nada, y la tercera no es la nada

| forma | qué es | qué hace |
|---|---|---|
| ausente | el parámetro no viene | no afirma nada. Es el valor cero |
| presente y vacío | el parámetro viene vacío | «deshacer». Tampoco afirma nada |
| presente y **no interpretable** | hay un dato y no se entiende | **error**: se dice, no se usa, y nunca se toma por el valor por defecto |

La tercera cubre una fecha que pone `ayer`, un entero que pone `muchos`, un
enumerado con un valor que su paquete no declara, un valor sobre una pregunta
booleana, uno más largo que el máximo, y la misma pregunta contestada dos veces.
En la pantalla se cuenta y se enseña; en `plazum alcance` **para la exportación
entera** y no escribe fichero, porque los demás cubos del exportador son
capacidades que faltan y éste es un dato que el operador sí contestó.

El campo opcional y el obligatorio se leen con **dos funciones distintas**
(`leerValorOpcional` y `leerValorObligatorio`), y en la obligatoria las tres
formas colapsan en la misma respuesta a propósito: la distinción ya se hizo
donde el dato entra.

### La compatibilidad hacia atrás no se rompe

Los enlaces compartidos y las cuentas guardadas están llenos de
`si=<pregunta con valor>`. Rechazarlos convertiría cada uno en una página de
error el día del despliegue. Se conservan: siguen moviendo la derivación
provisional de la pantalla, como hasta hoy, y siguen sin producir ningún hecho,
también como hasta hoy. Si llegan las dos formas sobre la misma pregunta, es
contradictoria y no afirma nada.

## Lo que esta rebanada NO puede, con su cardinal

### 1. Las respuestas con valor no se guardan en la cuenta

`Alcances.Responder` toma una `Respuesta` (Sí o No) y un valor no cabe en esa
frontera. Ensancharla toca `adaptadores/usuarios/alcances`, que **no está en la
columna de esta rebanada**, así que no se ha tocado.

Consecuencias, y las dos se dicen en el producto en vez de taparse:

- la pantalla cuenta cuántas respuestas con valor viajan en la dirección y no se
  guardan, y lo dice al lado del botón de guardar;
- `plazum alcance --cuenta` avisa de cuántas preguntas del corpus instalado se
  contestan con valor y que esa puerta no las trae, con la salida que sí
  (`--url`).

**Ese aviso es lo único que separa a `--cuenta` de absolver en silencio**: las
respuestas con valor no llegan ni siquiera como un cubo de la cuenta, porque
nunca entran en la consulta, y «ausente» es una respuesta legítima. Es la única
puerta capaz de dar un alcance corto sin que ningún cardinal lo diga.

Cierre: una `Respuesta` que admita valor, o un segundo método en `Alcances`.
Decisión de puertos, no de esta rebanada.

### 2. Una sola instancia

Ya estaba contado y sigue: todas las respuestas caen sobre el sujeto. El ENS
pregunta por CADA información y CADA servicio, así que con dos informaciones de
niveles distintos la segunda pisa a la primera. Sale impreso en cada ejecución
del exportador.

## Las tres pasadas: qué se intentó romper

### Pasada 1, contra la especificación

- **¿Es lo que pedía el encargo?** No del todo, y está dicho arriba: el
  desplegable se ha sustituido por enlaces. El requisito de seguridad se cumple
  con un mecanismo más fuerte; la forma no es la pedida.
- **¿Puede un paquete usar esto sin tocar código?** Sí, y es la comprobación que
  faltaba en este eje. El tipo del campo y sus valores salen del atributo que
  declara el paquete: un marco nuevo que traiga un enumerado de siete valores se
  pinta solo. Lo único cableado es la lista de tipos, que es la de
  `corpus.TipoAtributo`.
- **¿De dónde salen las palabras?** Los valores que se pintan son
  identificadores del corpus y viajan tal cual, sin pasar por el catálogo, igual
  que el texto de la pregunta y su cita. Las claves nuevas del catálogo son de
  interfaz y ninguna nombra una norma.
- **Las cifras del encargo eran de otro corpus** y se han vuelto a medir. Se
  dice arriba con las dos medidas.

### Pasada 2, contra el atacante

Ocho mutaciones, todas sobre árbol commiteado y restauradas con `cp` y nunca con
`git checkout`, comprobando con `go build ./...` que compilaban y con
`git diff --stat` que se aplicaban.

| # | qué se rompió | resultado |
|---|---|---|
| 1 | el valor de un enumerado deja de comprobarse contra la lista del paquete | rojo (5 tests) |
| 2 | lo no interpretable pasa a «sin contestar» | rojo (7 tests) |
| 3 | `Consulta()` copia el valor también cuando no afirma | **verde: sobrevivió** |
| 4 | el exportador deja de parar ante un valor que no se entiende | rojo |
| 5 | sin contestar, se toma la primera opción declarada (el peligro del desplegable) | rojo (13 tests) |
| 6 | valor y sí sobre la misma pregunta dejan de ser contradictorios | rojo |
| 7 | lo no interpretable deja de dejar la pregunta sin contestar | rojo |
| 8 | el aviso de `--cuenta` desaparece | rojo |

**La 3 es el hallazgo.** Sobrevivió porque un valor que no se entiende no se
conserva, así que lo que se copiaba era la cadena vacía y el contenido de la
página no cambiaba. Lo que sí cambiaba, y no lo miraba nadie, es **la dirección**,
que en esta superficie es el artefacto que se comparte y se guarda en
marcadores: «deshacer» dejaba un `v.<id>=` pegado a cada enlace de la página
para siempre, y ese parámetro vacío se vuelve a leer después como «el operador
eligió no contestar», que es una afirmación que nadie hizo. Test nuevo:
`TestNingunEnlaceLlevaUnValorVacio`, con su control positivo.

**Y una propiedad que se intentó tumbar y aguantó**: que una petición fabricada
pudiera meter en el estado de la pantalla una clave `v.<lo que sea>`. No puede,
porque `De` recorre las preguntas que declara el corpus y no las claves de la
consulta, que es la misma guarda que ya tenían `si` y `no`.

**Un fallo que encontró un test y no la revisión del diff**: la pantalla elegía
la pregunta sugerida con `Dice(id) == SinResponder`, que sólo mira la mitad de
sí/no. Contestar una pregunta con valor la dejaba marcada como «empieza por
esta» para siempre y la entrevista no avanzaba nunca.

**Y otro que encontró un test escrito para otra cosa**: `SinContestar` daba por
contestada una pregunta que llegaba con un valor contradictorio y un `sí`
antiguo, o sea que la pantalla anunciaba «esta respuesta no se ha usado» y a la
vez dejaba de sugerirla.

### Pasada 3, contra el comprador

- **P1, arreglado en esta rama**: `--cuenta` perdía en silencio exactamente las
  respuestas que esta rebanada añade. Ahora avisa. El arreglo de fondo es la
  frontera del almacén, fuera de columna.
- **P2**: la etiqueta del campo libre es la misma para los tres tipos («escribe
  la respuesta»). El texto de la pregunta va justo encima, así que se entiende;
  una etiqueta por tipo sería mejor.
- **P2**: con 68 preguntas y la entrevista entera contestada, la dirección que
  hay que pegar en `--url` es larga. `--cuenta` la evita, pero no trae los
  valores hasta que se cierre el hueco 1.

## Un error de proceso, para que no se repita

La primera ejecución de `./comprobar.sh` salió en rojo **por mi culpa**: la lancé
en segundo plano y seguí aplicando mutaciones encima del árbol mientras corría,
así que la puerta de la suite completa pilló `cmd/plazum` a medio mutar y
reportó `[build failed]`. No era un fallo del trabajo: era el lazo midiendo un
árbol que yo estaba moviendo.

La regla que faltaba, y que es hermana de «la pasada 2 empieza con `git status`
limpio»: **mientras una puerta corre, el árbol no se toca**. Una mutación y una
comprobación son dos usos incompatibles del mismo checkout.

## El lazo local, con su código real

```
COMPROBACION EN VERDE: 24 puertas leidas de los workflows,
21 ejecutadas aqui, mas formato, vet y build,
mas 3 herramientas de seguridad leidas de ci.yml,
3 ejecutadas aqui.
COMPROBAR_EXIT=0
```

Las 3 puertas saltadas son las tres de `-race`, y el motivo lo imprime el propio
lazo: `-race` exige cgo y aquí `CGO_ENABLED=0`. En CI sí corren. Se dice
distinguiendo «no se pudo ejecutar» de «encontró algo», que es el invariante 8
aplicado al propio lazo.

La frontera, contra la rama de integración real de esta rebanada (`tramo2/puente`,
que todavía no está en `main`):

```
$ PLAZUM_INTEGRACION=origin/tramo2/puente .github/frontera.sh valores main tramo2/valores
frontera del frente valores respetada: 19 ficheros, todos en su columna.
```

Contra `main` a secas sale roja con 26 ficheros ajenos, y **es el falso positivo
que el propio script documenta**: calcula el `merge-base` con la rama de
integración, y como `tramo2/puente` no está fusionada todavía, el diff arrastra
sus 26 ficheros. Es el mismo caso que el script explica al revés (un frente
rebasado sobre lo ya integrado), y por eso `PLAZUM_INTEGRACION` existe.
