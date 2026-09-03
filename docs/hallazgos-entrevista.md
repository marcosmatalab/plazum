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
  `superficies/pantallas`, pero `adaptadores/catalogo/cadenas/` no es columna de
  este frente. **Mientras no entren, 6 puertas están en rojo por esta única
  causa.** La lista, con su español y su inglés, va en el informe.
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
