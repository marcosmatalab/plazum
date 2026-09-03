# La ley de conservación del calendario: hallazgos

**Fecha:** 03-09-2026. Frente B de la campaña de cuatro frentes.

## El caso que lo trae

Se movió **quince meses** la vigencia del art. 14.6 del CRA (`cra.art14_6.informe_provisional_a_instancia_del_csirt`), que era la fecha más cercana de todo el corpus: aplicable el 11-09-2026, ocho días después del día en que se midió.

**No se puso roja ni una puerta.** La suite entera, 2256 casos, verde. El único efecto medible fue que el reloj **desaparece** del calendario del perfil de fabricante de software, de la sección de estrenos a un contador de descarte.

## Por qué no lo vio ninguna de las tres leyes que ya había

El proyecto ya tenía tres leyes de conservación sobre esta derivación, y las tres siguen cuadrando **después** de la mutación:

| ley | qué dice | por qué no lo ve |
|---|---|---|
| partición por **tiempo** | en vigor + estrenan + ya cesados + empiezan después + ilegibles = instalados | la obligación se mueve de un cubo a otro; la suma no cambia |
| partición por **alcance** | alcanzados + no alcanzados = en vigor | la obligación sale de «en vigor» por arriba; los dos lados bajan a la vez |
| partición por **destino** | todo reloj acaba en exactamente un destino, y el destino que promete fila la tiene | «empieza a obligar después de la ventana» **es un destino válido**, con su nombre y su motivo |

La lección, y es lo que hace de esto una familia y no un caso:

> **Una ley que sólo comprueba que la suma cuadre no puede ver un cubo que se vacía.** Para verlo hay que saber **cuántos** había en cada cubo, y eso no es una suma: es un **cardinal**.

Es la misma forma que ya tienen `SinDerivacionEsperadas` y `PUERTAS_ESPERADAS`, aplicada al sitio donde más duele. Y el fallo que persigue tiene una asimetría que conviene decir en voz alta: **una obligación de más se ve y se discute; una de menos no se ve, y por eso nadie la discute.**

## Lo que se construyó

`conservacion_calendario_test.go` (raíz), `TestNingunRelojDelCorpusDesapareceEnSilencio`:

- El universo se enumera **del árbol** (`corpus.Cargar("paquetes")`), no de una lista escrita al lado.
- Las vistas también: los **tres perfiles empotrados**, montados con plantilla 200 para que se enciendan todas las bandas.
- Cada reloj recibe **un veredicto y sólo uno**, de un vocabulario cerrado con el **valor cero prohibido**.
- **Los que se ven van por NOMBRE**, comparados como conjunto en los dos sentidos; las ausencias van por **cardinal**, con igualdad exacta en los dos sentidos.
- Los cubos cuadran, y la suma pasa por `metrica.Cuadra`.

### Los cardinales, al 03-09-2026 y con el instante cableado

**230 obligaciones con reloj instaladas. 92 se ven en el calendario de algún perfil.** Las 138 restantes están en una ausencia declarada:

| ausencia | cardinal | motivo |
|---|---|---|
| en vigor y ningún perfil lo alcanza | **116** | D-13: no se enumera, se cuenta. La puerta para verlos es `plazum calendario --todos-los-relojes` |
| empieza a obligar más allá de la ventana | **21** | el cubo al que cayó el art. 14.6 con la mutación |
| dejó de obligar antes de la ventana | **1** | no es una transición de estos doce meses |
| te alcanza y todas sus fechas caen más allá | **0** | vacío hoy; su rama la recorre el caso sintético |
| vigencia ilegible | **0** | vacío hoy; misma razón |

## Nació verde, y se dice

**La ley nació verde sobre el corpus entero.** No encontró ni un reloj perdido el día que se escribió. Siguiendo la regla de la casa, eso se dice en voz alta: *o vigila poco o llegó tarde*, y aquí es **llegó tarde** — el fallo que persigue ya se conocía y se le pidió la puerta después. Lo que demuestra que vigila no es su verde, es su mutación, y es la única cosa de la suite entera que la ve.

Su hermana de `nucleo/pantalla` (`TestCadaListaDeDescartesCuadraConSuContador`) **también nació verde** sobre el corpus publicado, por la misma razón: las listas se escribieron a la vez que la ley.

## El agujero que salió de intentar tumbar la propia ley

Con **sólo el cardinal**, dos movimientos opuestos en la misma pasada **se cancelan**: si un reloj deja de verse y otro empieza a verse, `relojSeVe` sigue valiendo 92 y los cubos de origen y destino también cuadran. La ley daría verde con un reloj perdido dentro.

No es una hipótesis cómoda: es exactamente lo que pasa cuando alguien **mueve una vigencia y añade una obligación en el mismo commit**, que es un commit de corpus normal.

Por eso los que se ven van **por nombre** y no sólo por número. Con la lista, además, el rojo dice **cuál** se cayó, que es lo que hace falta para decidir en un minuto si es un fallo o una novedad de la norma.

## El precio, dicho

El cardinal está topado con igualdad exacta, así que **toda adición al corpus pone esta puerta roja**. Es deliberado y es la disciplina de la casa (*un hueco con número molesta hasta que se cierra*), pero tiene un coste real para quien escribe corpus: un paquete nuevo obliga a actualizar el censo.

Se compensa así:

- el fallo imprime el censo **listo para pegar**;
- y la lista de los que se ven **nombra** lo que entró y lo que salió, así que actualizarla es leer dos líneas, no recontar nada.

Lo que **no** se puede hacer al actualizarla es bajar un número sin decir por qué. El mensaje del fallo lo dice con esas palabras.

## D11-c en el calendario: de diez cifras huérfanas a cinco

El motivo de que diez de las catorce cifras del pie no se pudieran abrir estaba escrito y era real: `nucleo/pantalla.Calendario` guardaba los descartes como **contadores de hitos** y lo único que retenía por elemento era `Destinos`, que va por obligación. Abrir una cifra en hitos contra una lista en obligaciones da una lista que **no cuadra** con el número que la abre.

Se arregló por la raíz. La derivación retiene cada descarte **en la unidad de su contador**, y `Calendario.Cuadra()` lo comprueba con `metrica.Cuadra`.

**Cinco cifras abiertas** (`mas alla`, `antes de vigor`, `ya cesados`, `empiezan tarde`, `ilegibles`). `SinDerivacionEsperadas`: **10 → 5**.

### Las cinco que siguen topadas, con su motivo

| cifra | por qué no se abre |
|---|---|
| instalados, en vigor, alcanzados | **no son descartes**: son el corpus entero mirado de tres formas. Enumerarlos sería pintar centenares de obligaciones que en su mayoría no son tuyas. D-13 |
| no alcanzados | D-13, **decidido y no pendiente**. La puerta es `--todos-los-relojes` |
| estrenan | cuenta **todo** lo que estrena, te alcance o no, y la sección de estrenos sólo trae lo que te alcanza. Enlazar ahí mandaría a una lista más corta que su número: es el error que D11-c existe para impedir |

### El detalle que era fácil hacer mal

Las tres listas que abren un contador **de hitos** llevan los **nombres** de sus hitos, no su número, y la pantalla emite **una fila por hito**. Con un `int`, la sección habría pintado una fila por *obligación* bajo una cabecera que cuenta *hitos*, y una obligación escalonada (alerta, notificación, informe final) habría dejado la lista dos corta **sin que nada se pusiera rojo**. El dato sintético de la puerta trae ese caso a propósito, y la mutación que lo demuestra vive dentro del propio test.

### Y el rótulo de cada sección es la clave de su propia cifra

Una sección que es *una cifra desplegada* no puede decir una cosa distinta de la cifra que la abre si las dos salen de la misma cadena. Efecto lateral buscado: **cero claves de catálogo nuevas**.

## Tercera pasada: el CISO de 200 a las nueve de la mañana

Tres cosas que se ven al abrirlo sin haber leído nada, y las tres son de D11 y D17.

1. **Tres de las cinco secciones nuevas no son «tuyas», y la página no lo dice.** `ya cesados`, `empiezan tarde` e `ilegibles` se calculan **antes** de la aplicabilidad, así que su número y su lista cuentan lo mismo (cuadran, que era el requisito) pero cuentan **todo el corpus**, te alcance o no. Puestas en *tu* calendario, la colocación insinúa que te obligarán. Las secciones que sí distinguen (`estrenos`) llevan su nota; estas no pueden llevarla sin una clave de catálogo nueva, que es de otra columna. **Es el hallazgo más serio de esta pasada y va con su clave pedida abajo.** Lo que sí se hizo mientras tanto: ni el rótulo ni las filas afirman en ningún sitio que la obligación sea del lector, y cada fila lleva su derivación con la fecha que la decidió.
2. **Las cinco cifras que no se abren no dicen en pantalla por qué.** Su `Motivo` existe, está escrito y lo comprueba la puerta, pero es «para quien lea el código». Quien vea nueve números con enlace y cinco sin él va a suponer que los cinco están a medias. Necesita clave.
3. **`N instalados que NO te alcanzan según tus respuestas` no dice cuál es la puerta para verlos.** `--todos-los-relojes` existe desde D-13 y no aparece en ninguna parte de la pantalla. Es la línea que convierte «el producto no lo ha mirado» en «míralos si quieres», y hoy sólo la sabe quien lee `docs/decisiones.md`.

### Las claves de catálogo que hacen falta, y que este frente NO ha tocado

`adaptadores/catalogo/cadenas/{es,en}.json` es de otra columna. Las cuatro que pide lo de arriba:

| clave | para qué | texto propuesto (ES) |
|---|---|---|
| `calendario.pantalla.descarte.no_es_tuyo` | nota de las tres secciones anteriores a la aplicabilidad | «Esta lista sale del corpus entero, no de tus respuestas: plazum todavía no ha mirado si alguna de estas te alcanza.» |
| `calendario.pantalla.cuenta.sin_abrir` | por qué una cifra no lleva enlace | «Esta cifra no se puede abrir todavía.» |
| `calendario.pantalla.cuenta.puerta_todos_los_relojes` | la puerta de D-13 | «Para verlos todos, `plazum calendario --todos-los-relojes`.» |
| `calendario.pantalla.cuenta.descuadre` | el aviso de cubos que no cuadran, como el del escalado | «AVISO: los cubos suman %d y se contaron %d. Es un fallo del producto, no tuyo.» |

## Lo que queda abierto

1. **La pantalla todavía no sabe decir que sus cubos no cuadran.** `pantalla.Calendario.Cuadra()` existe y las puertas lo ejecutan, pero la página no pinta el aviso que sí pinta el escalado (*«los cubos suman N y se planificaron M. Es un fallo del producto, no tuyo»*). Necesita una clave de catálogo nueva, que es de otra columna.
2. **`corpus.Obligacion.ID` no tiene unicidad global garantizada por el cargador.** El mapa `Destinos` es plano sobre el corpus entero, así que dos paquetes con el mismo id se pisarían. Hoy no pasa (lo comprueba esta puerta antes de contar), pero la comprobación vive en un test de la raíz y no en el linter de paquetes, que es donde le tocaría.
3. **Las cuatro cifras que ya se abrían siguen sin contraste de número contra lista**, y no es deuda: el número de vencidos cuenta ocurrencias y su lista trae una fila por obligación con sus ciclos al lado. Ahí el contraste no es contar filas.
