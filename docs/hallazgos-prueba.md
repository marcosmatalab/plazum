# Hallazgos del eslabón de la prueba (D-22, A1)

> **Fecha: 07-09-2026.** Todo número de aquí sale de la salida de un comando de
> esa sesión, no de memoria. El punto de partida es `258d06f`.

---

## La verificación del paso cero, y lo que no cuadraba

Las siete cifras del encargo se reprodujeron una a una y **todas cuadran**: 549
obligaciones, 34 `observable` por faceta, el reparto de recursos entero,
`Recolectar` sin implementaciones, `estado.Calcular` llamado sólo desde
`nucleo/expediente/expediente.go:717`, ninguna clave `pruebas` en el esquema, y
12 paquetes vacíos.

**Lo que no cuadraba no era una cifra: era el techo que se derivaba de una.**

| dónde vive `observable` | obligaciones |
|---|---|
| clase primaria (`clase_e2e`) | **99** |
| faceta (`facetas[]`) | **34** |
| en las dos a la vez | **0** |
| **techo real** | **133 de 549** |

El linter valida los dos campos contra el mismo vocabulario
(`nucleo/corpus/paquete.go:1833` y `:1839`) y los conjuntos son disjuntos. La
orden del encargo contaba sólo las facetas, así que su 34 es cierto y mide
facetas; lo que no se sostenía era *«tienen 34 obligaciones donde
engancharse»*, que habría publicado un techo **cuatro veces más bajo que el
real**, y además en la dirección que hace el argumento más dramático.

**El argumento sobrevive entero** —133 de 549 sigue siendo menos de una cuarta
parte— y la lección es de forma y no de aritmética: **un cardinal sobre un
vocabulario que vive en dos sitios se cuenta en los dos o no se cuenta.**

Y hay una segunda corrección, de numeración: la entrada se pidió como D-21 y
**D-21 existe** desde el 02-09-2026. Va como D-22.

---

## Lo que A1 cambia, medido

- El corpus real declara **1 prueba** (`paquetes/ens`, art. 20, mínimo
  privilegio, recurso `Privilegio`, TTL `P30D`, SLA `P7D`). Una, y se dice que
  es una: la campaña de los 12 marcos es la casilla de la etapa 3.
- **133 obligaciones observables, 132 sin prueba declarada.** El cardinal lo
  deriva `TestElTechoDeLaRecoleccionSeDerivaYNoSeEscribe`, que **no afirma un
  techo ni un suelo** —eso sería una promesa sobre el contenido del corpus, y el
  corpus crece— sino que el contador cuenta.

## Las tres puertas existentes que pararon esto, las tres con razón

1. **La frontera legal** exigió clasificar los cinco campos de texto nuevos. El
   `predicado` quedó como **prosa**, que es lo correcto y lo importante: es
   exactamente el sitio donde alguien pega el enunciado de un control de un
   catálogo de pago creyendo que ayuda.
2. **El vocabulario cerrado de `licencia_fuente`** rechazó `del_proyecto` (es
   `del-proyecto`).
3. **El de `identificador`** rechazó `sin_identificador` y, después, un
   `registro` que sólo usa el esquema de ISO.

Un paquete sintético de test tuvo que pasar por las mismas tres puertas que un
paquete real, que es la señal de que las puertas no tienen puerta de servicio.

---

## Las mutaciones (M25 a M30)

Todas sobre árbol limpio, con `.github/mutar.sh`, restauradas desde la copia.

| # | Qué se rompió | Puerta | Resultado |
|---|---|---|---|
| M25 | un campo `Cumple bool` añadido a `estado.Observacion` | `TestUnRecolectorNoPuedeDevolverUnVeredicto` | **cazada** |
| M26 | `Satisfecho` renombrado a `Conforme` en todo el árbol | la misma | **cazada** (a la segunda: ver abajo) |
| M27 | la guarda de `observable` apagada (`if false`) | `TestLasFormasDeRomperUnaPrueba` | **cazada** |
| M28 | el contraste `sla > ttl` aflojado ×1000 | la misma | **cazada** (a la segunda: la primera no compilaba) |
| M29 | el TTL vacío pasa a significar «no caduca» | la misma | **cazada** |
| M30 | el recurso deja de contrastarse contra la obligación | la misma | **cazada** |

### M26 sobrevivió la primera vez, y el motivo es doctrina

El `sed` que renombraba `Satisfecho` alcanzó también a `veredicto_test.go`, o
sea que **la justificación siguió al campo**: se renombró la excusa a la vez que
lo excusado. La regla de la casa lo dice con estas palabras: *si el test prueba
contra una lista escrita a su lado, muta **fuera** de esa lista.* Repetida
excluyendo el fichero de la puerta, se cazó en las dos direcciones (el campo sin
justificar y la justificación huérfana).

### M28 no compiló, y `mutar.sh` se negó a leer su rojo

`if okTTL && okSLA && false` deja `ttl` declarada y sin usar. La guarda 3 del
script hizo su trabajo: **un fallo de compilación no produce líneas `--- FAIL`**,
así que su rojo no demuestra que la puerta cace nada. Se rehízo como `sla >
ttl*1000`, que compila y afloja de verdad.

---

## Lo que A1 NO cierra, dicho

- **La pantalla de controles no enseña estado de evidencia.** Lo que pinta hoy
  es aplicabilidad derivada de la entrevista (`aplica` / `no aplica` / `falta
  contestar`), en `superficies/pantallas/derivacion.go:470`. Así que la pregunta
  de la pasada 3 —*un CISO ve un control en verde: ¿sabe por qué, desde cuándo y
  de dónde salió el dato?*— **hoy ni siquiera se puede formular**, y A1 no la
  arregla: la hace contestable, porque el dato ya existe (el `predicado` dice por
  qué, el `ttl` con `Recolectada` dice desde cuándo, y `Recolector` con
  `HashCarga` dicen de dónde). Ponerlo en pantalla es **A2**.
- **Y un riesgo para A2, apuntado ahora que se ve**: el verde de hoy significa
  **aplica** y el verde de mañana significará **consta**, y los dos van a estar
  en la misma pantalla. Dos verdes que significan cosas distintas a un palmo uno
  del otro es como se construye una pantalla que engaña sin mentir en ningún
  campo.
- **La obligatoriedad del predicado ya es exigible; su CONTENIDO no se
  comprueba.** El linter exige que haya predicado, no que el predicado diga algo
  evaluable. Eso último no lo puede hacer un linter sin un lenguaje de
  predicados (CEL, que está planeado en `DEPENDENCIAS.md` y no ha entrado).

---

# A2, A3 y A4, y las dos correcciones de A1 (07-09-2026)

## C1: la frase del invariante 13 describía un mundo que el propio bloque cambió

Decía que el predicado «lo exigirá el linter de A1» y que hasta entonces
`Satisfecho` estaba «documentado y no exigido». **A1 entró dos commits después,
en el mismo bloque**, y lo dejó exigido (`prueba.go:247`). La frase sobrevivió a
`main` y a CI.

Es la afirmación acompañada en su forma de **prosa a futuro**, que es la más
barata de cometer —quien la escribe tiene razón en el momento de escribirla— y
la más fácil de que caduque, porque describe un mundo que el propio bloque está
a punto de cambiar.

**Y el detalle que la hace doctrina**: su gemela vivía en el godoc de un
`_test.go` y llevaba un `NADIE LO VIGILA` puesto a mano. Esa marca **no la
validó nadie**, porque `godoc_vigilado_test.go` se salta los ficheros de test
(línea 128). *Un descargo escrito donde la puerta no mira es exactamente el
hueco que el descargo decía estar tapando.*

La orden del cierre de bloque entró en `CLAUDE.md`, es para **leer** y no para
filtrar, y su primer uso ya dio un falso positivo (la propia frase que documenta
esta corrección). Eso está bien: afinar el patrón convertiría una ayuda de
lectura en una puerta mala. **La escalada queda decidida de antemano**: si esta
clase falla dos veces más, `NADIE LO VIGILA` pasa a llevar fecha y registro con
cardinal de igualdad exacta.

## C2: `cierra`, porque una faceta en verde no cierra una obligación

De las 133 observables, **99 lo son por clase primaria y 34 por faceta**. En
esas 34, lo que la norma exige es otra cosa —un documento, un procedimiento— y
además hay un aspecto observable. Una prueba en verde ahí **aporta**; no cierra.

Sin el campo, A2 habría pintado «consta» sobre una obligación documental porque
su aspecto observable dio verde: **absolver de más con cara de dato**, que es el
error simétrico de acusar en falso y el que nadie mira.

Es `*bool` y no `bool`, y esa es la decisión: `false` es una respuesta legítima y
frecuente, así que no se distingue de «no lo he dicho» mirando el valor.
**Nació roja** sobre el corpus real, porque la única prueba que había no lo
declaraba. Y `ObservablesSinPrueba` publica **las dos mitades y no la unión**:
por clase 99 (faltan 98), por faceta 34 (faltan 34).

## Las tres preguntas de A2, contestadas al lado del cable

**Invariante 12.** `/controles` se sirve **sin sesión** y una observación es
dato de la **instalación**. De las dos cosas juntas sale el problema: pintarla
sería publicar a quien no ha entrado que un control falla, desde cuándo y con qué
recolector. Se descartan las dos salidas obvias por escrito y se toma la tercera:
**las columnas sólo con sesión, y la página lo dice**. Y **la frase no depende de
si hay evidencia**, porque «hay evidencia, entra para verla» ya diría que la hay.
Sin sesión **ni se llama** al adaptador.

**Invariante 7.** Casa por `Prueba.ID` con `Observacion.Prueba`, y por
`Prueba.Obligacion` con `estado.Prueba.Control`. Los dos son identidades,
ninguno es una posición, y los dos van dentro de lo firmado. El puente vive en
`nucleo/corpus` y no en `nucleo/estado` porque la dirección no es simétrica.

**El descargo y los dos verdes.** Se separan por **tres** cosas y ninguna es el
color: espacio de catálogo propio (`evidencia.` contra `estado.`), clase CSS
propia (`ev-` contra `e-`) y rótulo de columna propio. Hacía falta: los dos
vocabularios tienen un `no_aplica` que significa cosas distintas. Y la pastilla
de evidencia **no es redonda**: se distingue por forma, que es lo que vale
cuando quien mira no separa los dos verdes o imprime en blanco y negro.

## La pasada 3, que en A1 no se podía formular

Medida sobre el binario, con dos observaciones reales y el corpus del árbol.
**Dos hallazgos, los dos arreglados:** la fecha salía como el volcado de un
`time.Time` de Go, y el «por qué» no se pintaba (el predicado no llegaba a la
página y el motivo vivía en un `title`, que no lee un lector de pantalla, no se
imprime y en un móvil no existe).

La fila que ve el CISO ahora: *te aplica · no consta, en plazo · cada privilegio
concedido consta asignado a una función declarada · 1 recurso falla, plazo hasta
el 13 · dato de 2026-09-06 · lo trajo manual.*

**El residuo, declarado**: el motivo del motor lleva dentro un instante en
RFC 3339 (`2026-09-13T09:00:00Z`), que es formato de máquina en una pantalla de
persona. No se cambia aquí y se dice por qué: esa misma cadena viaja al
expediente que verifica un tercero, y allí un instante legible por máquina es lo
correcto. Cambiarlo es una casilla de su propio tamaño.

## Las mutaciones (M31 a M39)

| # | Qué se rompió | Puerta | Resultado |
|---|---|---|---|
| M31 | `cierra: true` sobre una faceta deja de rechazarse | `TestLasFormasDeRomperUnaPrueba` | **cazada** |
| M32 | `cierra` ausente se lee como `false` | la misma | **cazada** |
| M33 | la faceta se funde con la clase primaria | `TestLasDosMitadesDeObservableNoSeSolapan` | **cazada** |
| M34 | la evidencia se pinta sin sesión | `TestSinSesionNoSaleNiUnDatoDeEvidenciaYLaPaginaLoDice` | **cazada** |
| M35 | la evidencia ilegible se degrada a «nadie ha recolectado» | `TestLaEvidenciaIlegibleNoSeLeeComoQueNadieHaRecolectado` | **cazada** |
| M36 | «aporta» se pinta como «cierra» | `TestUnaPruebaQueAportaNoSePintaComoQueCierra` | **cazada** |
| M37 | la ley de conservación apagada | — | **SOBREVIVIÓ** (ver abajo) |
| M38 | el veredicto del fichero se ignora en vez de parar | `TestUnFicheroConUnVeredictoDentroNoSeLeeIgnorandoloSePara` | **cazada** |
| M39 | una versión desconocida se lee «lo mejor que se pueda» | `TestElRecolectorRotoNoDaUnExpedienteVacio` | **cazada** |

### M37 sobrevivió, y su motivo es el hallazgo

Apagar la comprobación del recuento dejaba la suite entera en verde. El motivo
es exacto: **el contador del total y el parser recorren las mismas líneas**, así
que hoy **ningún fichero puede producir un descuadre**. Una guarda que ninguna
entrada alcanza es una guarda que no existe.

Se queda, con prueba sintética en las dos direcciones, porque **lo que vigila no
es un fichero: es el parser del futuro**. El día que alguien meta un `continue`
de más en el bucle —que es como se traga filas un parser, y es exactamente lo que
documenta `nucleo/censo`— los dos contadores dejarán de coincidir.

## Lo que sigue sin cerrar, con su nombre

- **La excepción autodeclarada.** `Excepciones` y `Exclusiones` entran directas a
  `estado.Calcular` desde el expediente y **no están ancladas en la cadena**: una
  excepción con cualquier aprobador y un `Hasta` futuro pasa `Valida()`,
  `Calcular` corta y devuelve `Exceptuado` sin mirar ni una observación, y el
  verificador reproduce lo mismo porque lee las excepciones del propio emisor.
  Grep sobre el árbol: aparecen en **tres líneas fuera de `nucleo/estado`, las
  tres en `expediente.go`, y CERO en cualquier `_test.go`**. Es la familia «el
  emisor mete la mano en el expediente», que la capa probatoria tiene **cerrada**
  desde el ataque 14: se documenta y no se arregla.
- **El tamaño del binario.** Subió de 11,7 a **11,8 MB**, y el número es una
  **derivación y no una medida**: el delta se midió en el mismo banco (cross
  linux/amd64: 12.849.314 a 12.906.658, o sea 57.344 bytes) y se sumó al nativo,
  que en `e006610` era 11,7 exacto porque CI estaba verde y allí la comparación
  es por igualdad. 11,755 cae cerca del corte del redondeo: **si CI dice otra
  cosa, manda CI**.
- **La banda estaba casi agotada antes.** En `e006610` el desvío contra el README
  era 4,74 % sobre una banda del 5 %, y 57 KB la cruzaron. El número publicado
  llevaba tiempo a la deriva y la banda lo absorbía.
