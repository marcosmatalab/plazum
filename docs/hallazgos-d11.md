# D11 en el calendario, el acta y el escalado: hallazgos

**Fecha:** 04-09-2026. Frente C del tramo 1 de la campaña de dos semanas.
**Columna:** `superficies/calendario/`, `superficies/acta/`, `superficies/escalado/`,
`nucleo/pantalla/`, este fichero.

La tercera pasada, hecha como se pide: un CISO de 200 empleados abre estas tres
pantallas a las nueve de la mañana, sin documentación y sin soporte. Cada
hallazgo con su prioridad y su cardinal. Los que están **arreglados** llevan el
commit que los cierra; los que no, dicen por qué no y qué hace falta.

## De dónde sale cada número de este documento, porque no todos valen igual

La regla de la casa dice que **ningún resultado cuenta en un informe si no salió
de la puerta**. Aquí hay números de dos procedencias y se marcan, porque
confundirlas es exactamente lo que la regla prohíbe:

| número | de dónde sale |
|---|---|
| `alcanzados` = **204** filas, `estrenan` = **4**, `cesan` = **0** sobre el corpus publicado | del log de la puerta `TestCadaListaDeDescartesCuadraConSuContador`, que corre dentro de `./comprobar.sh` |
| `SinDerivacionEsperadas` = **1**, las 7 secciones, los cuadres | de las puertas de `superficies/calendario` |
| **17 / 28 / 73** alcanzados y **201 / 190 / 145** no alcanzados **por perfil**, y el **9 contra 4** de los estrenos | de una **sonda desechable** que escribí y borré (`go test -run`), NO de una puerta |

Los de la tercera fila son los que sostienen los motivos de `alcanzados` y de `no
alcanzados`, o sea los que deciden si el hueco está bien tapado. **Están medidos
y no están vigilados**: el día que el corpus cambie, esas cifras del texto
mienten y nadie se entera. Es la familia de la *afirmación acompañada*, con la
prosa como parte que caduca. Lo que haría falta para cerrarlo es que el censo de
`conservacion_calendario_test.go` imprima el reparto **por perfil**, y ese
fichero no es de ninguna columna este tramo. **P2**, anotado aquí para que
moleste.

---

## D11-c: de cinco cifras huérfanas a UNA

`SinDerivacionEsperadas`: **5 → 1**.

Las tres que se abrieron no estaban cerradas por falta de dato: estaban cerradas
por un **motivo mal escrito**. El motivo compartido de los tres totales decía
*«no son descartes, son el corpus entero mirado de tres formas, y enumerarlos
sería pintar centenares de obligaciones que en su mayoría no son tuyas»*. Era
cierto y no era un motivo: **describía por qué no vale UNA salida (enumerar) y
callaba que había OTRA (sumar)**. Un motivo así no se puede contradecir, así que
se queda puesto para siempre.

| cifra | qué se hizo | cardinal medido el 04-09-2026 |
|---|---|---|
| `estrenan` | lista propia (`RelojesQueEstrenan`). El diagnóstico viejo era bueno y la conclusión no: la sección de estrenos trae menos porque solo trae lo tuyo **y** una fila por obligación | la cifra dice **9** y esa sección tiene **4** filas en `es-fabricante-software` |
| `alcanzados` | lista propia (`RelojesAlcanzados`). **No** es «el corpus mirado de otra forma»: es lo que SÍ te alcanza, y D-13 prohíbe enumerar lo que **no** te alcanza | **17 / 28 / 73** hitos en los tres perfiles publicados |
| `instalados` y `en vigor` | **partición**: la suma de otras cifras de la misma lista, escrita al lado. Las dos particiones existen y las comprueba `contabilidad_test.go` desde el 28-08-2026: la pantalla tenía la demostración escrita y se la guardaba | 249 = 218 + 9 + 1 + 21 + 0, y 218 = 17 + 201 (perfil de fabricante) |

### La que queda, con su motivo mejor que el de ayer

**`no alcanzados`**, y son dos razones independientes, no una:

1. **D-13, decidido y no pendiente.** Enumerarla serían entre **145 y 201 filas
   ajenas** en los tres perfiles publicados. Eso no informa, entierra.
2. **La partición tampoco vale, y sería circular.**
   `en vigor = alcanzados + no alcanzados` es **una** ecuación con **dos**
   incógnitas si ninguna de las dos se sostiene sola. `alcanzados` se sostiene
   (tiene su lista), así que la ecuación abre `en vigor`, y `no alcanzados` se
   queda como **el único número de la página que hay que creerse**.

Tener exactamente **uno**, dicho en voz alta y con su orden de terminal al lado
(`plazum calendario --todos-los-relojes`), es lo que se podía conseguir sin
romper D-13. El motivo viejo cubría tres cifras con una frase y no decía ninguna
de estas dos cosas.

### La forma de mentir que trae la partición, y su puerta

Abrir por suma tiene un fallo que un enlace no tiene: **se puede cerrar sobre sí
misma**.

```
en vigor      = alcanzados + no alcanzados
no alcanzados = en vigor - alcanzados
```

Las dos ecuaciones son ciertas, las dos sumas cuadran, las dos cifras quedarían
marcadas como abiertas y **ninguna de las dos se podría comprobar**: son la misma
ecuación escrita del derecho y del revés. `CifrasQueSeDerivanEnCirculo` lo caza,
vive en el código (no en el test) para que el control negativo pase por el mismo
detector, y trae las dos direcciones: caza el círculo **y** no acusa a dos cifras
apoyadas en la misma tercera.

---

## P0 arreglado: una cifra se abría a una sección más corta que ella

`cesan` cuenta **hitos** y su sección pintaba **una fila por obligación**. Las
dos puertas que había lo dejaban pasar por sitios distintos: una compara `N` con
`len(Filas)` **en el modelo** y solo para las secciones de descarte; la otra
comprueba que el enlace y su destino **existan**, no que el destino cuente lo
mismo.

Nació **verde** y de suerte: el único cese del dato sintético tenía un hito, así
que 1 = 1. Y el corpus publicado tampoco lo habría puesto rojo: sus tres perfiles
dan `cesan` = **0**, o sea que **esa rama no la recorre el dato real**. Con un
segundo cese escalonado (lo que es toda notificación por fases) salió:

```
cuadre_test.go:115: la cabecera de #ceses cuenta 3 y la seccion pinta 2 filas.
  Quien pulse esa cifra va a contar las filas y le van a salir 2. Un numero que
  no cuadra con su lista es PEOR que un numero sin enlace: el enlace prometia que
  se podia comprobar.
  La cifra es CuentaVista.Cesan.
```

La puerta nueva cuenta las filas **en la respuesta**, que es lo que cuenta el
lector, para toda cifra que se abre, y cada cifra declara **cómo** se contrasta
(`FormaDeCuadre`, con el valor cero prohibido).

---

## P1 CERRADO (04-09-2026): la colocación afirmaba, y ahora además lo dice

**Cuatro** secciones se calculan **antes** de la aplicabilidad y cuentan el
corpus entero, te alcance o no: `estrenan`, `ya cesados`, `empiezan tarde`,
`ilegibles`. Ni el rótulo ni las filas dicen que sean tuyas, y aun así acusan,
porque están en **tu** calendario: la página no lo dice, **el sitio sí**.

**Hecho:** las que hablan de ti (las tres calculadas **después** de la
aplicabilidad: `alcanzados`, `más allá`, `antes de vigor`) suben con lo tuyo, y
las cuatro del corpus bajan detrás de todo. Lo vigila una puerta que mide el
**orden de la página**, no el reparto: comprobar el reparto contra el mapa que lo
hace sería preguntarle a la respuesta por la respuesta.

**Hecho también, y era la mitad que importaba:** la **nota** al frente del
bloque, con `calendario.pantalla.descarte.no_es_tuyo`.

### Por qué la colocación sola no bastaba, y era peor de lo que decía este texto

Lo que estaba escrito arriba es que la separación «ayuda y no cierra el
hallazgo». Es corto. Bajar las cuatro secciones quita la insinuación de que te
obligan y **no dice lo que pasa de verdad**, así que deja al lector con la
lectura contraria y exactamente igual de falsa: un bloque de listas cortas al
final de **tu** calendario se lee como que plazum ya lo miró y decidió que eso no
era tuyo.

Eso es **absolver de más en silencio**, que es el error simétrico de acusar en
falso y es peor, porque una acusación la corrige quien la lee y una absolución la
descubre quien te inspecciona. La nota es lo único que dice la verdad entera:
plazum todavía no ha mirado si alguna de esas te alcanza.

**Con sus dos controles**, en `acusacion_test.go`. El positivo recorre la rama
(la página con bloque pinta la nota, delante de las **cuatro** secciones y no
sólo de la primera, porque una nota entre la segunda y la tercera llega tarde
para las que ya se leyeron). El negativo la cierra: sin bloque no hay nota, y sin
él una nota que saliera siempre pasaría el positivo diga lo que diga la página.
Es M47 literal, y esta pantalla ya lo pagó una vez.

---

## P1: una región con scroll del acta no se alcanzaba con el teclado

`.marco-tabla` lleva `overflow: auto` y `max-height: 78vh` en la hoja de la casa,
así que **es una región con scroll**. `superficies/acta/plantillas/acta.html`
tenía su tabla en un `<div class="marco-tabla">` pelado: con ratón se arrastra,
con teclado no se llega. Es `scrollable-region-focusable`, **la misma violación
que se cobró esa pantalla el 04-09-2026**, en otro elemento y en la tabla en la
que se abre cada cifra del acta, o sea la razón de ser de la página.

**Sobrevivió porque vivía en la junta entre dos puertas**, que es la familia de
este tramo:

| puerta | qué mira | por qué no lo vio |
|---|---|---|
| `superficies/pantallas/armazon_test.go` | `.marco-tabla` | solo en las cuatro rutas de esa superficie |
| `regiones_con_scroll_test.go` (raíz) | todas las plantillas | solo el elemento preformateado, que era lo del día anterior |

**Arreglado** en el acta, con su puerta nueva (`superficies/acta/regiones_test.go`),
que recorre las tres superficies de esta columna. Nació roja sobre el árbol de
verdad, sin mutación.

### Lo mismo, sin arreglar, fuera de mi columna

`superficies/uar/plantillas/uar.html:193` tiene **el mismo** `<div
class="marco-tabla">` pelado, y esa tabla es la campaña de accesos entera.
**P1 para quien tenga esa columna**: `tabindex="0" role="region" aria-label=…`,
igual que el acta. Cardinal: **1 región**.

---

## P2: la puerta de la raíz acusa a la prosa que la explica

`regiones_con_scroll_test.go` busca la etiqueta del bloque preformateado con una
expresión regular **sobre el fichero entero, comentarios incluidos**. Escribir esa
etiqueta dentro de un comentario de plantilla la pone roja. Me pasó al documentar
el arreglo de arriba, y lo resolví nombrando el elemento con palabras.

Es exactamente la familia del falso positivo de «aviso.md» del 03-09-2026, que ya
está anotada en el script de axe: **una puerta que acusa en falso se acaba
borrando, y entonces no vigila nada.** El arreglo es del dueño de la raíz
(recortar los comentarios `{{- /* … */ -}}` antes de buscar). Cardinal: **1
aparición**, la mía, resuelta esquivándola.

---

## D11-a en las tres pantallas: dónde hay que leer código o documentación

### 1. El calendario pide dos órdenes de terminal, y siguen ahí. **P1**

`calendario.pantalla.sin_alcance.paso` = *«Responde la entrevista, expórtala con
`plazum alcance` y arranca el servidor con `plazum serve --alcance
alcance.json`»*.

**No las he quitado, y digo por qué:** al rebasar sobre `main` (que está en
`1d954e8`, la matriz del tramo, y en `origin/main` sigue `334ae3b`) el trabajo
del frente A que convierte el alcance en estado de la cuenta **todavía no está
dentro**. Y aunque estuviera, la frase vive en `adaptadores/catalogo/cadenas/`,
que es su columna: quitarla es una edición suya, no mía.

Cardinal del hueco: **4 órdenes de terminal en 2 claves**, las dos de la columna
A — `calendario.pantalla.sin_alcance.paso` (2 órdenes) y
`escalado.pantalla.sin_alcance.paso` (2 órdenes, la misma pareja).

### 2. El acta no se puede crear desde el navegador. **P1**

Su estado vacío pide **dos órdenes con seis banderas** y las dos empiezan por
`plazum serve`, o sea que **hay que parar el servidor y volver a arrancarlo**:

```
plazum serve --acta-organizacion "Tu S.L." --acta-desde 2026-01-01 --acta-hasta 2026-12-31
plazum serve --acta-programa programa.json --acta-incidentes incidentes.json --accesos-fichero usuarios.csv --accesos-ledger uar.json
```

Un CISO que abre `/acta/` a las nueve de la mañana no puede tener un acta sin
matar el proceso que está mirando. Las órdenes están escritas **tal cual** en la
plantilla (bien: una orden de terminal no se traduce) y el problema no es el
texto, es que **no hay otro camino**. El arreglo es de producto y de otra columna
(el formulario vive en `superficies/serve`), así que va como hallazgo.

Cardinal: **2 órdenes, 6 banderas, 1 reinicio del servidor**.

### 3. CERRADO. El escalado enseñaba vocabulario del núcleo sin traducir. **P2**

`superficies/escalado/plantillas/escalado.html` pinta la cuenta así:

```
<li>{{.Estado}}: {{.N}}</li>
```

`.Estado` es `nucleo/escalado.Estado` en crudo: **ocho valores** en prosa
española (`pendiente`, `sin destinatario`, `colapsado en un escalon anterior`,
`suprimido por una ventana de silencio`, `enviado al canal`, `entregado`,
`fallido en la entrega`, `atendido`). En la página en inglés salen en español, y
en las dos salen sin explicación.

**El acta ya resolvió este mismo problema** con la familia `acta.cubo.*`, así que
hay patrón y no hubo que inventar nada. **8 claves, puestas el 04-09-2026.**

#### Había un argumento escrito EN CONTRA, y hubo que refutarlo, no ignorarlo

El godoc de `ClavesDeCatalogo` decía que traducir los estados «crearía dos
nombres para el mismo cubo en dos medios del mismo producto, que es como se
pierde a alguien que compara una captura de pantalla con un log». Es un buen
argumento y la conclusión era falsa, porque **compara los dos medios en el mismo
idioma** y el problema estaba en el otro: el lector inglés no tenía un nombre
distinto para ese cubo, no tenía **ninguno**. La coherencia que aquel párrafo
defendía se conserva donde de verdad se compara, que es en castellano, porque el
rótulo español de cada cubo es **letra por letra** la constante del núcleo.

#### El mapeo vive en la superficie, y su cardinal se deriva

`nucleo/escalado/` no se toca (es de otra rebanada, y además una clave de
catálogo es vocabulario de interfaz: un `func (e Estado) Clave()` ataría el motor
a cómo se rotula una pantalla). El mapa está en `superficies/escalado/vista.go` y
`ClavesDeLosCubos()` **recorre `EstadosPosibles()`, no el mapa**: esa dirección es
la que importa, porque recorrer el mapa daría las claves que hay, que es justo lo
que no se quiere saber. Un noveno estado sin emparejar se cae de la lista, el
inventario del catálogo lo echa de menos y la puerta de los cubos lo nombra.

### 4. La cuenta del escalado no abre ni una cifra. **P2**

D11-c está aplicada al panel de inicio, al acta y ahora al calendario, y **no** al
escalado: sus cubos (`estado: N`) no llevan enlace a los avisos que los componen,
aunque los `Trabajos` con sus `Pasos` están pintados justo encima. Cardinal:
**8 cifras sin derivación** (una por estado), más `planificados`.

Es de mi columna y **no lo he hecho**: cabía en el tramo o cabía la partición del
calendario, y la partición cierra un P1 del tramo anterior. Va con su número para
que moleste.

### 5. La sección de `alcanzados` repite lo que ya sale arriba. **P2**

En el perfil de servicios digitales, `alcanzados` = 73 y `sin fecha` = 73: son
los mismos relojes, y la página los pinta **dos veces**, una en cada sección. No
es una coincidencia del dato ni un fallo: hoy casi todo lo que te alcanza espera
un hecho del operador, así que los dos conjuntos casi coinciden. Cuando haya
hechos declarados dejarán de coincidir solos.

Se acepta a sabiendas: la alternativa era dejar `alcanzados` sin abrir, y una
cifra que hay que creerse es peor que una lista repetida. Pero **es ruido y hay
que decirlo**, y la salida buena, cuando exista, es que la sección de
`alcanzados` marque en cada fila **dónde más sale ese reloj** en vez de repetirlo
entero.

### 6. CERRADO. La partición se leía en cifras y sin palabras. **P2**

La página escribía `= 218 + 9 + 1 + 21 + 0` al lado de la cifra. Es
**comprobable** y es **independiente del idioma** (los signos `+` y `=` no se
traducen), que es lo que permitió cerrar D11-c con **cero claves nuevas**. Pero
se leía como una fórmula, no como una frase.

**Hecho** con `calendario.pantalla.cuenta.se_compone_de`. Los signos **se
quedan**: la frase se añade delante y no sustituye a nada, porque lo comprobable
es la aritmética y no la frase. Hay puerta para las dos mitades, y hacen falta las
dos: se puede escribir la frase y quitar la suma (y entonces la cifra vuelve a ser
un número que hay que creerse, con una frase encima), y se puede dejar la suma sin
frase, que es de donde se venía.

---

## Las doce claves de catálogo, PUESTAS (04-09-2026, rebanada 1 del tramo 2)

La versión anterior de esta sección decía «que este frente NO ha tocado», y
encabezaba con **Once claves** una tabla de **doce filas**. Las dos cosas se
arreglan aquí, y la segunda merece decirse en voz alta porque es la familia de la
*afirmación acompañada*: el cardinal de la prosa estaba escrito a mano, no
derivado, así que discrepaba de su propia tabla el día que se escribió. Nadie lo
vigilaba. Ahora el cardinal tiene puerta, y no en una lista escrita al lado sino
en la que además las pinta: `ClavesDeLosCubos()` sale de recorrer
`nescalado.EstadosPosibles()`, y las cuatro del calendario están en
`ClavesDeCatalogo()`, que los dos inventarios cruzan en los dos sentidos.

**Por qué entran con su código y no solas:** es la lección de `a1f65fa`. La
puerta del catálogo cruza en las dos direcciones y una clave que nadie pide es
tan roja como una que falta, así que las claves y el código que las pinta van en
la misma rama o no van. Se demostró otra vez al mutar: quitar una sola línea del
mapa de cubos pone rojo *«el catálogo traduce `escalado.cubo.en_silencio` y no la
pide nadie»*, que es literalmente el mensaje del commit que aplazó este trabajo.

| clave | ES | EN (británico) |
|---|---|---|
| `calendario.pantalla.descarte.no_es_tuyo` | Esta lista sale del corpus entero, no de tus respuestas: plazum todavia no ha mirado si alguna de estas te alcanza. | This list comes from the whole corpus, not from your answers: plazum has not looked yet at whether any of these reach you. |
| `calendario.pantalla.cuenta.sin_abrir` | Esta cifra no se puede abrir. Para verlas todas, plazum calendario --todos-los-relojes | This figure cannot be opened. To see them all, plazum calendario --todos-los-relojes |
| `calendario.pantalla.cuenta.descuadre` | AVISO: los cubos suman %d y se contaron %d. Es un fallo del producto, no tuyo. | WARNING: the buckets add up to %d and %d were counted. This is a fault in the product, not yours. |
| `calendario.pantalla.cuenta.se_compone_de` | se compone de | is made up of |
| `escalado.cubo.pendiente` | pendiente | pending |
| `escalado.cubo.sin_destinatario` | sin destinatario | no recipient |
| `escalado.cubo.colapsado` | colapsado en un escalon anterior | collapsed into an earlier step |
| `escalado.cubo.en_silencio` | suprimido por una ventana de silencio | suppressed by a silence window |
| `escalado.cubo.enviado` | enviado al canal | sent to the channel |
| `escalado.cubo.entregado` | entregado | delivered |
| `escalado.cubo.fallido` | fallido en la entrega | delivery failed |
| `escalado.cubo.atendido` | atendido | handled by a person |

**Doce filas, doce claves distintas.** Las ocho del escalado son una familia y
entraron juntas, igual que `acta.cubo.*`.

### Tres cosas del texto que no eran obvias

1. **El castellano de los ocho cubos es LETRA POR LETRA la constante del núcleo.**
   No es pereza: es lo que conserva la promesa que el godoc anterior defendía
   (que la pantalla y la terminal no den dos nombres al mismo cubo) allí donde de
   verdad se comparan, que es en castellano. Lo que aquel argumento no miraba es
   que el lector inglés no tenía un nombre distinto: no tenía **ninguno**.
2. **Sin acentos y sin comillas invertidas, contra la propuesta.** El texto
   propuesto traía `plazum calendario --todos-los-relojes` entre comillas
   invertidas y «todavía» con tilde. Los vecinos del espacio `calendario.*` y
   `escalado.*` no llevan ni una cosa ni la otra (`sin_alcance.paso` escribe sus
   dos órdenes a pelo), y una pantalla mitad acentuada y mitad no se lee como que
   nadie mira. Se sigue al vecino.
3. **La partición conserva sus signos.** `se compone de = 30 + 2 + 1 + 3 + 1`. El
   `=` y los `+` no se traducen y son lo comprobable, así que la frase se añade
   **delante** y no sustituye a nada. Hay puerta para las dos mitades, porque
   cada una se puede cumplir rompiendo la otra.

---

## Lo que sigue abierto en el calendario, con su cardinal

1. ~~**La pantalla todavía no sabe decir que sus cubos no cuadran.**~~ **CERRADO
   el 04-09-2026**, y con más de lo que pedía: ver la sección siguiente.
2. **`corpus.Obligacion.ID` no tiene unicidad global garantizada por el
   cargador.** Sin cambio; la comprobación sigue viviendo en un test de la raíz y
   no en el linter de paquetes.
3. **La rama de `cesan` no la recorre el corpus publicado.** Los tres perfiles dan
   **0**, así que su único dato es el sintético. Está dicho en el log de la puerta
   de `nucleo/pantalla` y aquí; no es deuda de código, es un hueco de cobertura
   del corpus que conviene saber.

---

## Rebanada 1 del tramo 2 (04-09-2026): lo que salió de escribir las puertas

Lo de arriba es el frente C del tramo 1. Esto es lo que encontró la rebanada de
la nota al poner las doce claves con su código. Tres cosas no eran el encargo y
salieron de escribir las puertas, que es donde salen.

## P1: `Cuadra()` no bastaba, y la pantalla ahora se contrasta a sí misma

El encargo decía *«la página no pinta el aviso que sí pinta el escalado»*. Al
mirar por qué, el hueco resultó ser mayor que el aviso:
`nucleo/pantalla.Calendario.Cuadra()` contrasta ocho contadores contra sus ocho
listas y **sus dos puertas corren contra el corpus publicado y contra el dato
sintético**. El calendario **del cliente** no pasa por ninguna de las dos. Si su
corpus produce un descuadre, la página se lo pinta tan tranquila y el hito que
sobra o falta simplemente **no sale**.

Así que el contraste se hace **al pintar**, sobre lo que el lector puede contar
(`DescuadresDeLaCuenta`), y cubre **más** que el método del núcleo: las **2**
particiones y las **4** secciones que se pintan con su propia rama de plantilla
(`fechas`, `vencidas`, `ceses`, `sin fecha`), que el núcleo no mira. Con su puerta
de subsunción en las dos direcciones: si el núcleo dice que no cuadra, la página
lo dice; y sobre un calendario que cuadra, la página no se inventa un aviso.

Dos casos tautológicos se dicen en vez de disimularse: `SinFecha` y `EnLaVentana`
se contrastan hoy contra lo mismo de lo que salía su contador, así que **no pueden
ponerse rojos**. Se dejan escritos porque el día que cualquiera de los dos cambie
de origen el contraste empieza a ser de verdad sin que nadie se acuerde.

## P1: el descuadre INVISIBLE, que ninguna otra puerta de esta pantalla ve

Una cifra en **cero** no se pinta; sin cifra no hay sección; sin sección no hay
filas que contar. Un contador a cero con **dos relojes retenidos detrás** deja
esos hitos fuera de la página entera, y nadie los echa de menos porque nadie
llegó a saber que existían.

El primer borrador de `DescuadresDeLaCuenta` filtraba por `SePinta()`, que es lo
natural, y dejaba fuera **justo ese caso**. Se quitó el filtro y tiene su dato
sintético propio (`calendarioConUnaCifraEnCeroYFilasDetras`) y su puerta, que
empieza comprobando que la premisa es la que dice: si la sección se pintara, el
test estaría mirando el caso fácil.

## P1: el escalado PERDÍA recuentos, y el arreglo hizo viva una rama muerta

`rellenarCon` recorría `EstadosPosibles()` y consultaba el mapa del plan, así que
un estado **fuera** de la partición cerrada no salía en ningún cubo **y no se
sumaba**. Lo único que quedaba de él era el aviso de descuadre, que dice que los
números no cuadran y **no dice cuál falta**.

Salió al escribir el control del respaldo, y el rojo no fue el que se buscaba: el
cubo salía **vacío**, no sin traducir. Ahora esos estados se pintan detrás,
ordenados alfabéticamente (un mapa se recorre distinto en cada petición y
bailarían entre dos visitas), con la palabra del núcleo por rótulo. Efecto
lateral buscado: el respaldo de `CuboVista.Clave` deja de ser una rama defensiva
que nadie ejecuta, que es M47 en su forma más pura.

## Los dos errores míos, con lo que enseñan

1. **Mi control negativo acusaba a la página correcta.** Buscaba la palabra del
   núcleo (`pendiente`) en la sección de la cuenta, y la clave
   `escalado.cubo.pendiente` **contiene** esa palabra: con el catálogo espía, que
   pinta `[[clave]]`, la encontraba siempre. Es exactamente la trampa que
   documenta `ingles_test.go` (*«se miran los VALORES y no las claves»*): los
   identificadores de este repositorio están escritos en castellano, así que se
   parecen a la cadena que nombran. Se quitan los marcadores del espía antes de
   preguntar. **El fallo probable de un control negativo es acusar a lo que está
   bien**, y aquí se cumplió a la primera.
2. **Escribí las doce claves antes que sus entradas.** Los dos inventarios
   nacieron rojos con *«publica X y la pantalla no la pide en ninguno de sus
   estados»*: seis de los ocho cubos (el plan de prueba llena dos y los ceros no
   se pintan) y el aviso de descuadre (el calendario de prueba cuadra a
   propósito). El rojo es el correcto y la respuesta no es quitar la clave: es
   traer la entrada que recorre su rama. Hicieron falta dos datos sintéticos
   nuevos, `planConLosOchoCubos()` y `calendarioQueNoCuadra()`.

## Lo que NO se ha cerrado, con su cardinal, para que no se dé por hecho

1. **Los ocho cubos del escalado siguen sin abrir ni una cifra** (D11-a #4).
   Traducir el rótulo **no** es darle su derivación: `estado: N` ahora se lee, y
   sigue sin enlace a los avisos que lo componen. Cardinal sin cambio: **8 cifras
   sin derivación**, más `planificados`. Quien lea las claves nuevas no debe leer
   ese hallazgo como cerrado.
2. **Las 4 órdenes de terminal en 2 claves**
   (`calendario.pantalla.sin_alcance.paso` y `escalado.pantalla.sin_alcance.paso`)
   siguen ahí, y ahora esas dos claves **sí** están en esta columna. No se tocan
   porque el hueco no es el texto: es que no hay otro camino, y el camino lo abre
   la rebanada que tiene `superficies/serve/` y `cmd/plazum/`. Cambiar la frase
   antes que el camino sería escribir una instrucción que no funciona.
3. **El castellano de los ocho cubos es letra por letra la constante del núcleo y
   NO hay puerta que lo exija.** El acta sí la tiene
   (`nucleo/acta.CadenasDelActa()`, contrastada en `adaptadores/catalogo`). Aquí
   no se puede escribir sin salirse de la columna: viviría en
   `adaptadores/catalogo/*_test.go`, que no es de esta rebanada. Cardinal: **8
   cadenas atadas por convención y no por puerta**. **P2 para quien tenga esa
   columna.**

## Tercera pasada: dos cosas que sólo se ven leyendo la página, no el diff

### El aviso más fuerte de la página se pinta como prosa normal. **P2, fuera de mi columna**

El descuadre sale como `<p class="error">`, que es lo que ya hacía el escalado. En
`superficies/pantallas/estatico/plazum.css` **no hay ninguna regla** para `.error`
salvo `.principal.error`, que es otra cosa (el ancho de la página de error). O sea
que el aviso que dice *«hay avisos que no están en ningún sitio»* se ve
exactamente igual que el resto del texto. Medido: **3 clases usadas por estas dos
plantillas y sin regla propia** en la hoja, `.error` (en `<p>`, dos pantallas),
`.particion` (ya estaba) y `.sin-abrir` (nueva).

Lo mitiga que el texto empiece por **AVISO** y **WARNING** en mayúsculas, que es
énfasis que no depende de la hoja de estilos, y esa es la razón de que esto sea P2
y no P1. El arreglo es del dueño de `superficies/pantallas/`, que no es esta
rebanada.

### Los catorce rótulos de la cuenta no concuerdan en número. **P2**

`"%d hitos de reloj instalados"` con `N` = 1 da *«1 hitos de reloj instalados»*.
El catálogo tiene mecanismo de plural desde siempre (**17** cadenas lo usan) y
**ninguno de los 14 rótulos de la cuenta** lo usa.

**Lo que sí se ha arreglado, y por qué sólo eso:** la frase nueva de la partición.
Su concordancia es con el sujeto que va delante, así que *«37 hitos de reloj
instalados se compone de»* estaría mal en **casi todos** los renders y no sólo en
el borde de uno. Lleva sus dos formas (`se compone de|se componen de`,
`is made up of|are made up of`) y recibe `.N` para que el catálogo elija. Los otros
catorce fallan **sólo** cuando su cifra vale 1, así que se cuentan y se dejan:
tocarlos son **28 ediciones** (14 × 2 idiomas) de texto que nadie pidió cambiar,
con una forma plural que hay que acertar también en inglés británico. Cardinal:
**14 rótulos sin forma plural**.

---

# Rebanada 1 del tramo 3 (04-09-2026): la capa visual y las tres puertas de D11

Rama `tramo3/visual`. Columna: `superficies/pantallas/`, `superficies/camino/`,
`superficies/acta/`, `superficies/uar/plantillas/`, `superficies/calendario/`,
`superficies/escalado/`, `adaptadores/catalogo/cadenas/`, `ttfv_camino_test.go`
y este fichero.

Lo de arriba es el frente C del tramo 1 y la rebanada 1 del tramo 2. Esto es lo
que salió de cerrar sus deudas, y **cuatro de los seis hallazgos nuevos salieron
de escribir las puertas, no de leer el diff**.

## El resumen contable, para no tener que leerlo entero

| cosa | antes | ahora |
|---|---|---|
| cifras huérfanas en el escalado | 8 + `planificados` | **0** |
| cifras huérfanas en el calendario | 1 | **1**, y ahora con la demostración de que 1 es el suelo |
| TTFV medido | 15m51s | **20m21s** (la medida dejó de ser ciega, no el producto de empeorar) |
| cuello del TTFV | «la entrevista», escrito a mano | **las órdenes de terminal, 7m30s de 20m20s (37 %)**, derivado |
| clases sin regla en la hoja | 3, contadas a mano | **0**, con puerta; el barrido encontró 33 |
| pantallas con captura | 8 × 2 temas | **11 × 2 temas** |
| rótulos de la cuenta sin plural | 14 | **0** (13 con dos formas, 1 invariable con su motivo) |
| las 8 cadenas de los cubos | convención | **puerta** |
| regiones con scroll sin teclado | 1 (la UAR) | **0** |

---

## D11-a #4 CERRADO: las ocho cifras del escalado se abren

El informe anterior lo dejó dicho con todas las letras: *«traducir el rótulo NO
es darle su derivación»*. Se cierra con la ruta `GET /escalado/cubo/{estado}`,
con el mismo molde que el acta usa para `/acta/derivacion/{ref}`.

**Por qué campo casa un escalón con su cubo, dicho en voz alta** (invariante 7):
por `nucleo/escalado.Estado`, **letra por letra**. Es la constante del
vocabulario cerrado del motor: la misma que indexa `Plan.Cuenta`, la misma que
lleva dentro cada `Paso` y la misma que viaja escapada en la dirección. Nunca por
la posición del cubo en la lista ni por el orden de `EstadosPosibles()`: insertar
un noveno estado movería el emparejamiento entero sin romper nada. Aquí no hay
nada firmado que proteger, pero la familia del fallo es la misma.

**Y el total se abre por partición**, que es la otra forma: `planificados = 1 + 1
+ 1`, escrita al lado con su frase delante. No es circular, que es la trampa que
documenta `superficies/calendario/cuenta.go`: cada sumando se sostiene solo, con
su enlace y su lista.

### Lo que hizo falta y no estaba en el encargo

1. **El recuento se contrasta contra la lista, en la RESPUESTA.** `Plan.Cuenta` y
   `Plan.Trabajos` llegan por el mismo puerto y **no tienen por qué cuadrar**:
   `Fuente` es un interfaz y quien lo implemente puede darlos separados. Una
   cabecera que dice 5 sobre una lista de 2 es peor que una cifra sin enlace,
   porque el enlace prometía que se podía comprobar. Es el P0 que se cobró el
   calendario, en otra pantalla y con la misma forma.
2. **Los estados sueltos también se abren.** El respaldo de `CuboVista.Clave`
   pintaba un estado fuera de la partición **con** rótulo y **sin** enlace, o sea
   una cifra huérfana con nombre.
3. **La sesión, en la ruta nueva.** Esa lista lleva nombres de personas dentro.
   Una ruta nueva que se olvidara de la sesión publicaría por la puerta de atrás
   lo que la principal protege por la de delante, y eso no lo ve ninguna puerta
   de la pantalla principal. Con control positivo: **con** sesión el nombre sí
   sale, para que un 401 permanente no pase por «protegido».

### Un rojo que me encontré y no era del producto

`TestLosOchoCubosSalenTraducidosEnLaPagina` se puso rojo acusando a la página
correcta, **por tercera vez y por la misma razón**: la palabra del núcleo ahora
viaja dentro del `href`, que es identidad y no rótulo. Se añadió el tercer
recorte (`sinDirecciones`) **con su control negativo**, porque el fallo probable
de un recorte es llevarse por delante justo lo que vigila y dejar el control
pasando sobre el vacío.

---

## D11-c: el escalado a CERO, y por qué el calendario se queda en UNO

**Escalado: 9 → 0.** `SinDerivacionEsperadas = 0`, con igualdad exacta en los dos
sentidos.

**Calendario: sigue en 1** (`no alcanzados`), y esta vez con la demostración de
que **1 es el suelo y no una decisión perezosa**:

- La cuenta pinta **14** cifras.
- **11** tienen lista propia y se abren con ella.
- **3** no tienen lista en ninguna parte: `instalados`, `en vigor` y
  `no alcanzados`. `nucleo/pantalla.Calendario` guarda `HitosNoAlcanzados` como
  **contador y nada más** (no hay `RelojesNoAlcanzados`, y `Destinos` va por
  obligación y no por hito, así que no cuadra con esa cifra).
- Entre esas 3 hay **2** ecuaciones de partición independientes:
  `instalados = en vigor + estrenan + ya cesados + empiezan tarde + ilegibles` y
  `en vigor = alcanzados + no alcanzados`.
- **3 incógnitas y 2 ecuaciones: exactamente una se queda sin abrir.** Cuál de
  las tres es la que queda es una elección; que quede una no lo es.

Lo único que bajaría ese 1 a 0 es una lista en `nucleo/pantalla`, y `nucleo/` no
lo toca **nadie** en el tramo 3, dicho en la matriz. **P1 para el tramo 4**, con
el diff exacto: `Calendario.RelojesNoAlcanzados`, poblado en `Derivar12Meses`
donde hoy sólo se hace `cal.HitosNoAlcanzados += hitosDeclarados(o)`. Con eso, la
cifra se abre a su lista y D-13 se respeta igual, porque D-13 dice *«un contador,
y una puerta para verlos si quiere»*, y una página aparte a la que se llega
pulsando **es** esa puerta.

---

## D11-e: el TTFV contaba las órdenes en una sola dirección

**El número publicado era 15m51s. El de verdad es 20m21s.** El producto no
empeoró en un día: la medida dejó de ser ciega.

El contraste de órdenes recorría la lista **declarada** en el censo y la buscaba
en la pantalla. Con eso cazaba una orden declarada de más (coste inflado) y **no**
cazaba una orden que la pantalla pide y nadie declara (coste desinflado). Es el
patrón que la casa persigue en el corpus y en el expediente, aplicado a la única
cifra que este producto le enseña al comprador. **Y el error iba siempre a
favor.**

Lo que escondía: **seis invocaciones del binario, en tres pantallas, ninguna
declarada** (el acta 2, la revisión de accesos 2, el plan de avisos 2), todas en
sus estados vacíos, todas desde que existe esta medida.

### El reparto de hoy, para que se pueda recontar

```
lectura de 6 pantallas    6 x 45s   = 4m30s
19 preguntas              19 x 20s  = 6m20s
5 ordenes cobradas        5 x 1m30s = 7m30s
instalacion                           2m0s
                                   -------
T_humano                             20m20s   (+ 1.3 s de maquina)
```

**Ocho invocaciones en cuatro pantallas, cobradas cinco veces.** El plan de
avisos repite la pareja del calendario y no se cobra dos veces: una persona no
teclea dos veces la misma orden porque dos pantallas se la pidan. El desglose
imprime las dos cifras (pedidas y cobradas) para que nadie tenga que creerse el
deduplicado.

**Deduplicar no es rebajar el coste.** Lo prohibido es bajar el coste **por**
orden hasta que salga el número; esto es lo contrario: se cobra entero, una vez.
`CosteDeResponderUnaPregunta` sigue en 20 s y `CosteDeTeclearUnaOrden` en 90 s,
sin tocar.

### Cómo se distingue una orden de una frase que nombra el producto

Por el **subcomando**, y la lista de subcomandos **no se escribe en el test**: se
saca de la ayuda del propio binario, que el test ya compila y ya arranca. Hacía
falta porque el catálogo tiene frases que nombran a plazum sin pedir nada
(*«plazum todavía no ha mirado si alguna de estas te alcanza»*), y acusarlas sería
el falso positivo que acaba con una puerta borrada.

### El techo tenía dientes sólo en su godoc

`TechoDeclaradoTTFV` decía *«EL TECHO TIENE DIENTES EN LAS DOS DIRECCIONES»* y no
lo leía **nadie**: `grep` daba una sola línea, su propia declaración. Un techo que
ningún test comprueba no es un techo, es un comentario que además afirma lo
contrario de lo que pasa. Ahora se comprueba por arriba y por abajo.

Sube de 17m a 21m30s **y eso no es la trampa que persigue**: la trampa prohibida
es subirlo porque la entrevista ha crecido, y la entrevista no ha crecido (mismas
19 preguntas, mismos 6 pasos). **`PresupuestoTTFV` sigue en 15 minutos y la
casilla D11-e sigue sin cumplirse**, ahora por 5m20s en vez de por 51s.

### Y el cuello se deriva, porque la prosa es la que caduca

La línea final decía *«el cuello de botella es la entrevista de /alcance»* con el
número al lado: el número se corregía solo y la frase se quedaba puesta. **Ya no
es cierta.** Las órdenes son 7m30s (37 %) y la entrevista 6m20s (31 %). Ahora la
frase se compone del mismo reparto que se imprime encima.

### Qué haría falta para cumplir los 15 minutos, con su número

Con las **cinco órdenes cobradas fuera**, el mismo modelo da **12m50s**, por
debajo del presupuesto. Y ninguna se puede quitar desde esta columna, porque **el
hueco no es el texto: es que no hay otro camino**. El camino que falta es que el
alcance guardado en la cuenta (que existe desde `f49af01`) alimente al
calendario, al plan de avisos y al acta sin pasar por un fichero, y eso vive en
`cmd/plazum/serve_calendario.go` (`alcanceEnFichero`) y en `superficies/serve`.
**P0 de producto para el tramo 4**, y es la casilla D11-e entera.

### Un fallo de producto que salió de mirar esto

El estado vacío del acta tenía **dos** bloques y los dos empezaban por `plazum
serve`. Quien siga las instrucciones al pie de la letra pega la segunda,
**arranca otra vez el servidor y pierde `--acta-organizacion`, `--acta-desde` y
`--acta-hasta`**: se queda sin acta y sin saber por qué. Al lado había un
argumento escrito (*«son DOS porque son dos decisiones distintas»*) que es bueno y
cuya conclusión es falsa: **dos decisiones no son dos órdenes cuando las dos son
el mismo proceso**. Ahora es una, entera y pegable.

---

## La hoja de estilo: tres clases contadas a mano eran treinta y tres

El cardinal anterior decía **3** (`.error` en `<p>`, `.particion`, `.sin-abrir`) y
**no estaba mal contado**: eran las tres que alguien había *mirado*, no las que
había. La puerta nueva (`superficies/pantallas/hoja_test.go`) **nació roja sobre
el árbol, sin mutación**, y encontró 35 entradas: **33 clases sin ninguna regla**
más las dos familias `.e-` y `.n-`, que son clases a medias y ya estaban
declaradas.

**Por qué nadie se enteraba:** una clase sin regla no rompe nada de lo que este
repositorio vigila. El HTML sigue siendo válido, el contraste sigue pasando
porque no hay color que medir, y axe no dice nada porque no es un problema de
accesibilidad. El único síntoma es que esa parte de la página se lee como un
volcado de terminal.

Lo que dolía era el P2 heredado: `<p class="error">` es lo que usan el calendario
y el escalado para decir que sus cuentas **no cuadran**, y se pintaba como prosa
normal.

### La mutación encontró un agujero en mi propia puerta

Se le quitó a la hoja la regla `p.error` y **la puerta salió verde**, porque
`plazum.css` sigue teniendo `.principal.error`, que menciona la clase y no
alcanza a un `<p class="error">`. **Es literalmente la frase que este mismo
fichero escribió sobre esta misma clase**: la puerta reproducía el error que venía
a cazar.

Y el arreglo enseñó lo suyo: rechazar todo selector compuesto salió rojo con
**doce** clases más, y ninguna era un fallo. `hoy`, `tabla`, `vacia`, `dormida`,
`elegido`, `activa`, `activo`, `apagada`, `apagado`, `sugerida`, `e-decidido` y
`e-sin-decidir` son **modificadores**: existen para combinarse, y `.principal.hoy`
es su regla correcta. Hay **dos poblaciones** y la diferencia no se decide a ojo,
se lee de la plantilla: una clase que en algún sitio es la **única** de su
elemento necesita una regla que la alcance sola.

---

## Las capturas: las dos pantallas con más cifras no se habían fotografiado nunca

8 pantallas → **11**, en los dos temas: **22 capturas**. Faltaban el calendario y
el plan de avisos, que son justo los dos con más números y más listas, o sea
donde un bloque sin regla se lee como un volcado. Entra también la cifra abierta
del plan, que es la ruta estrenada hoy.

Los datos de las dos nuevas son del producto y no de una maqueta: el calendario
se deriva con `pantalla.Derivar12Meses`, la misma función que usa `plazum
calendario`.

---

## Las deudas heredadas, cerradas

1. **Las 8 cadenas atadas por convención → puerta.** El rótulo castellano de cada
   cubo es letra por letra la constante de `nucleo/escalado`, y ahora hay quien lo
   exija. **Esta puerta nace verde y se dice en voz alta**: llegó tarde, no vigila
   poco. Vive en `superficies/escalado/catalogo_test.go` y no en
   `adaptadores/catalogo/`, por dos motivos: el mapa `cubos` vive en la superficie,
   y `adaptadores/catalogo/` no es de esta columna (sólo `cadenas/` lo es).
2. **Los 14 rótulos sin plural → 13 con dos formas.** *«1 hitos de reloj
   instalados»* era lo que leía quien tuviera un solo hito. El catorce no es un
   olvido: `en_vigor` («%d en vigor» / «%d in force») **no concuerda en número**, y
   darle dos formas idénticas sería ruido con cara de arreglo.
3. **La región con scroll de la UAR.** El informe anterior la dejó como P1 «para
   quien tenga esa columna», con su cardinal (1 región). En el tramo 3 esa
   columna es esta. Arreglada, y **la puerta de `superficies/acta/regiones_test.go`
   se extiende a las plantillas de la UAR**, que es lo que impide que vuelva.

---

## Lo que queda abierto, con su cardinal y su dueño

1. **1 cifra huérfana en el calendario** (`no alcanzados`). Suelo demostrado
   arriba; sólo baja con una lista en `nucleo/pantalla`. **Tramo 4.**
2. **5 órdenes de terminal cobradas en el TTFV**, 7m30s, el 37 % del total, en
   cuatro pantallas. Ninguna se quita desde esta columna. **`cmd/plazum` +
   `superficies/serve`, tramo 4.** Es la casilla D11-e entera.
3. **La segunda orden de la UAR solapa con la del acta.** `plazum serve
   --accesos-fichero usuarios.csv --accesos-ledger uar.json` son banderas que la
   orden única del acta ya lleva dentro, y el acta va **antes** en el camino: quien
   lo recorra en orden ya la ha tecleado. El deduplicado del TTFV no lo ve porque
   casa por texto y son dos cadenas distintas, así que **esa orden se cobra dos
   veces**. Cobrar de más se dice, no se corrige a mano. **P2 para
   `superficies/uar/`.**
4. **`.github/mutar.sh` no funciona dentro de un worktree.** Guarda su depósito en
   `.git/mutaciones` y en un worktree `.git` es un **fichero**, no un directorio:
   `preparar` muere con `mkdir: cannot create directory '.git': Not a directory`.
   **El tramo 3 entero se construye en worktrees**, así que hoy ninguna rebanada
   puede usar el script que la casa exige para la pasada 2. Las ocho mutaciones de
   esta rebanada se hicieron con una copia del script con las mismas cuatro
   guardas, fuera del árbol. Arreglo de una línea: `deposito="$(git rev-parse
   --git-dir)/mutaciones"`. **P1 para la rebanada 0**, que tiene ese fichero.
5. **Las 4 órdenes en 2 claves siguen ahí**, y ahora se sabe que eran **8
   apariciones en 4 pantallas**. El texto no se toca porque el hueco no es el
   texto: cambiar la frase antes que el camino sería escribir una instrucción que
   no funciona.

## Las ocho mutaciones, y las dos que enseñaron algo

| # | qué se rompió | qué se puso rojo |
|---|---|---|
| M1 | `d.Descuadre = false` en `derivar` | la puerta del descuadre **y** el inventario de claves |
| M2 | el cubo pintado sin `URL` | «la cuenta pinta 3 cubos y solo 0 llevan enlace» |
| M3 | cualquier estado se abre | los dos casos del 404 **y** el inventario |
| M4 | la ruta del cubo sin sesión | «contesta 200 sin sesión» y el nombre de la persona dentro |
| M5 | `p.error` fuera de la hoja | (la primera vez, **verde**: ver abajo) |
| M6 | un rótulo de cubo cambiado en `es.json` | «dice una cosa en la pantalla y otra en la terminal» |
| M7 | las órdenes de la UAR fuera del censo | «pinta 2 invocaciones y el censo declara 0» |
| M8 | el techo a 30 minutos | «un techo que sobra miente hacia arriba» |

**M5 salió verde y ese fue el hallazgo**, no un fallo de la mutación: la puerta
contaba `.principal.error` como regla de `.error`. Está contado arriba.

**Y una mutación no compiló**, que es la trampa 3 del guion: quitar el `if
!sePinta` dejó `declared and not used`. La guarda lo dijo por código de salida en
vez de dejarlo pasar por «no lo caza», que es exactamente para lo que existe.

## Mis errores

1. **Escribí el mensaje del commit en el scratchpad compartido y otra sesión me lo
   sobrescribió.** El directorio de temporales es del *proyecto*, no de la sesión:
   `commit1.txt` apareció con el texto de la rebanada de la release. No se perdió
   nada porque ya estaba commiteado, pero el siguiente fichero con nombre genérico
   se pierde. Desde ahí, todo con prefijo propio.
2. **Mi primera versión del buscador de reglas CSS contaba un selector compuesto
   como regla de la clase suelta**, o sea reproducía el error que la puerta venía
   a cazar. Lo encontró la mutación, no la revisión, y sólo porque la mutación
   elegida fue la de la clase de la que este fichero ya había escrito la
   distinción.
3. **Di por hecho que `mutar.sh` funcionaría** y perdí un intento antes de mirar
   por qué fallaba. La lección es la del propio guion: un comando que puede dar un
   falso verde se convierte en script antes de usarse dos veces, y un script que
   no arranca hay que mirarlo, no rodearlo en silencio.

## P2 leído del workflow, no medido: axe no audita ninguna ruta de derivación

**Cardinal: 2 rutas, y una es la que estrena esta rebanada.**

`.github/workflows/etapa2-accesibilidad.yml` compone su lista así: la raíz,
`/entrar`, las seis pantallas **descubiertas del menú** (`<nav
id="navegacion">`), `/alcance?ver=todas` y, desde el 04-09-2026, `/acta/`,
`/uar/` y `/escalado/`. Fuera quedan:

| ruta | qué es |
|---|---|
| `/acta/derivacion/{ref}` | una cifra del acta, abierta |
| `/escalado/cubo/{estado}` | una cifra del plan de avisos, abierta (nueva hoy) |

Y **`/calendario/` no sale en ninguna de las dos vías**: no es una de las seis
del menú de `superficies/pantallas` y no está en el bucle de las tres que se
añaden a mano. Habría que confirmarlo sobre una ejecución del job.

**Es exactamente la familia que ese mismo fichero documenta** sobre
`/alcance?ver=todas`: *«el menú descubre rutas, y esa no es una ruta: es la misma
con otro parámetro»*. Una derivación es lo mismo con otra forma, y en las dos que
faltan el contenido es una **tabla o una lista larga**, o sea justo donde viven
las violaciones de región y de encabezado.

**Se dice como leído y no como medido**: sale de leer el workflow, no de correr
axe, que es un paso de CI con node y no se puede ejecutar en el lazo local. Las
dos capturas de `/escalado/cubo/` (clara y oscura) están tomadas y miradas, y eso
**no sustituye** a axe: una captura dice cómo se ve, no si se navega.

**No se arregla desde aquí a propósito.** `.github/workflows/` es puerta
compartida y cambiarla a mitad de campaña caduca lo que las otras tres rebanadas
ya validaron. Va con su diff: añadir al bucle `/acta/derivacion/1.1.1`,
`/escalado/cubo/pendiente` y `/calendario/`, con el mismo servidor y la misma
cookie que ya usa.
