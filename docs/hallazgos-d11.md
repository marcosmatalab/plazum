# D11 en el calendario, el acta y el escalado: hallazgos

**Fecha:** 04-09-2026. Frente C del tramo 1 de la campaña de dos semanas.
**Columna:** `superficies/calendario/`, `superficies/acta/`, `superficies/escalado/`,
`nucleo/pantalla/`, este fichero.

La tercera pasada, hecha como se pide: un CISO de 200 empleados abre estas tres
pantallas a las nueve de la mañana, sin documentación y sin soporte. Cada
hallazgo con su prioridad y su cardinal. Los que están **arreglados** llevan el
commit que los cierra; los que no, dicen por qué no y qué hace falta.

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

## P1 arreglado a medias: la colocación afirma

**Cuatro** secciones se calculan **antes** de la aplicabilidad y cuentan el
corpus entero, te alcance o no: `estrenan`, `ya cesados`, `empiezan tarde`,
`ilegibles`. Ni el rótulo ni las filas dicen que sean tuyas, y aun así acusan,
porque están en **tu** calendario: la página no lo dice, **el sitio sí**.

**Hecho:** las que hablan de ti (las tres calculadas **después** de la
aplicabilidad: `alcanzados`, `más allá`, `antes de vigor`) suben con lo tuyo, y
las cuatro del corpus bajan detrás de todo. Lo vigila una puerta que mide el
**orden de la página**, no el reparto: comprobar el reparto contra el mapa que lo
hace sería preguntarle a la respuesta por la respuesta.

**No hecho, y es la mitad que importa:** la **nota** al frente del bloque.
Necesita una clave de catálogo y `adaptadores/catalogo/cadenas/` es de otra
columna este tramo. Va en la lista de claves de abajo. **Prioridad P1**: mientras
no esté, la separación ayuda y no cierra el hallazgo.

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

### 3. El escalado enseña vocabulario del núcleo sin traducir. **P2**

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
hay patrón y no hay que inventar nada. Necesita **8 claves** (columna A).

### 4. La cuenta del escalado no abre ni una cifra. **P2**

D11-c está aplicada al panel de inicio, al acta y ahora al calendario, y **no** al
escalado: sus cubos (`estado: N`) no llevan enlace a los avisos que los componen,
aunque los `Trabajos` con sus `Pasos` están pintados justo encima. Cardinal:
**8 cifras sin derivación** (una por estado), más `planificados`.

Es de mi columna y **no lo he hecho**: cabía en el tramo o cabía la partición del
calendario, y la partición cierra un P1 del tramo anterior. Va con su número para
que moleste.

### 5. La partición se lee en cifras y sin palabras. **P2**

La página escribe `= 218 + 9 + 1 + 21 + 0` al lado de la cifra. Es
**comprobable** y es **independiente del idioma** (los signos `+` y `=` no se
traducen), que es lo que permitió cerrarla con **cero claves nuevas**. Pero se
lee como una fórmula, no como una frase.

Con una clave (`calendario.pantalla.cuenta.se_compone_de`, *«se compone de»*) se
leería como lo que es. **P2**, no P1: hoy el número se puede comprobar, que es lo
que D11-c exige.

---

## Las claves de catálogo que hacen falta, y que este frente NO ha tocado

`adaptadores/catalogo/cadenas/{es,en}.json` es de la columna A. **Once claves**,
por orden de prioridad.

| clave | prioridad | para qué | texto propuesto (ES) |
|---|---|---|---|
| `calendario.pantalla.descarte.no_es_tuyo` | **P1** | la nota al frente del bloque de secciones anteriores a la aplicabilidad | «Esta lista sale del corpus entero, no de tus respuestas: plazum todavía no ha mirado si alguna de estas te alcanza.» |
| `calendario.pantalla.cuenta.sin_abrir` | P2 | por qué la única cifra sin enlace no lo tiene | «Esta cifra no se puede abrir. Para verlas todas, `plazum calendario --todos-los-relojes`.» |
| `calendario.pantalla.cuenta.descuadre` | P2 | el aviso de cubos que no cuadran, como el que ya pinta el escalado | «AVISO: los cubos suman %d y se contaron %d. Es un fallo del producto, no tuyo.» |
| `calendario.pantalla.cuenta.se_compone_de` | P2 | leer la partición como una frase | «se compone de» |
| `escalado.cubo.pendiente` | P2 | los ocho estados del escalado, que hoy salen en crudo | «pendiente» |
| `escalado.cubo.sin_destinatario` | P2 | ídem | «sin destinatario» |
| `escalado.cubo.colapsado` | P2 | ídem | «colapsado en un escalón anterior» |
| `escalado.cubo.en_silencio` | P2 | ídem | «suprimido por una ventana de silencio» |
| `escalado.cubo.enviado` | P2 | ídem | «enviado al canal» |
| `escalado.cubo.entregado` | P2 | ídem | «entregado» |
| `escalado.cubo.fallido` | P2 | ídem | «fallido en la entrega» |
| `escalado.cubo.atendido` | P2 | ídem | «atendido» |

Son doce filas y once claves distintas más la de la partición: las ocho del
escalado son una familia y entran o no entran juntas, igual que `acta.cubo.*`.

---

## Lo que sigue abierto en el calendario, con su cardinal

1. **La pantalla todavía no sabe decir que sus cubos no cuadran.**
   `pantalla.Calendario.Cuadra()` existe, ha crecido con tres comprobaciones más
   y las puertas lo ejecutan, pero la página no pinta el aviso que sí pinta el
   escalado. Necesita clave. Heredado del tramo anterior, sin cambio.
2. **`corpus.Obligacion.ID` no tiene unicidad global garantizada por el
   cargador.** Sin cambio; la comprobación sigue viviendo en un test de la raíz y
   no en el linter de paquetes.
3. **La rama de `cesan` no la recorre el corpus publicado.** Los tres perfiles dan
   **0**, así que su único dato es el sintético. Está dicho en el log de la puerta
   de `nucleo/pantalla` y aquí; no es deuda de código, es un hueco de cobertura
   del corpus que conviene saber.
