# El marcador: el subíndice de plataforma, al lado del global y nunca en su lugar

> **Fecha de esta medición: 04-09-2026, por la TARDE.** Las notas salen de `docs/instantanea.md`, los pesos de `docs/diseno.md` §14 (tabla vigente tras D-20), y las tres cifras publicadas las computa `subindice_test.go` del dato y las contrasta con lo que dice este fichero, con igualdad exacta y en las dos direcciones. Si una cifra sube, rompe; si baja, rompe igual.
>
> **Por qué pone «por la tarde», que es lo único importante de esta cabecera.** La medición anterior se hizo el **mismo día**, entre las 00:47 y las 02:22. Las cuatro rebanadas del tramo 3 aterrizaron entre las **14:47 y las 15:35**. O sea que el marcador que se publicó y se citó durante un tramo entero era una foto tomada **antes** del trabajo que decía medir, y sus números eran ciertos por la mañana y falsos por la tarde. La sección «Por qué este marcador no se movía» lo cuenta entero, porque es el hallazgo y no una nota al pie.

## Las tres cifras

<!-- marcador:inicio -->
- **Subíndice de plataforma open source, publicable: 8,41**, sobre **12 dimensiones** y **78 puntos de peso**, con numerador **656,3**.
- **Global de las 17 dimensiones: 6,66**, sobre **109 puntos de peso**, con numerador **726,3**.
- **Las 5 dimensiones excluidas, medidas aparte: 2,26**, sobre **31 puntos de peso**, con numerador **70,0**.
<!-- marcador:fin -->

**El numerador se publica y no es adorno: es la mitad de la puerta que tiene dientes.** Un ponderado con dos decimales se traga un movimiento pequeño: bajar D9 de 9,7 a 9,6 cambia el subíndice de 8,4141 a 8,4103, que redondea al mismo **8,41**, así que una nota podría bajar sin que nada se pusiera rojo. El numerador no: baja de **656,3** a **656,0**, y eso es un dígito distinto en un número publicado. La puerta compara los cuatro valores de cada línea, así que **cualquier movimiento de cualquier nota, en cualquier dirección, rompe**.

Las tres líneas se publican juntas siempre. **La de en medio es el número del producto**; la primera dice cuánto de la plataforma que hoy se puede descargar, arrancar y publicar está hecha; la tercera dice cuánto vale lo que la primera se deja fuera, para que nadie tenga que ir a buscarlo.

## Por qué este marcador no se movía, que es el hallazgo del día

El tramo 3 metió 22 relojes nuevos, la capa visual entera y los cimientos de la IA, y el marcador siguió diciendo **8,32 y 6,41**, los mismos números del tramo anterior. Un instrumento que no se mueve cuando el producto se mueve mucho está roto. Éste lo estaba, y por dos motivos distintos que conviene no mezclar.

**Motivo 1, y es el barato: nadie lo recomputó.** Los tres commits del marcador llevan hora (`00:47`, `00:53`, `02:22`) y los cuatro de las rebanadas también (`14:47`, `14:47`, `14:47`, `15:35`). No hay misterio: la foto se tomó doce horas antes del trabajo. Ese número muerto se citó durante un tramo entero.

**Motivo 2, y es el que hay que arreglar con una puerta: su puerta no podía cazarlo.** `subindice_test.go` compara `docs/marcador.md` con `docs/instantanea.md` y con `docs/diseno.md`, celda a celda y en las dos direcciones. Es una buena puerta y no tenía nada que decir, porque **los tres documentos eran consistentes entre sí**. Lo que ninguno hacía era mirar el **árbol**.

> **Coherencia y frescura no son lo mismo, y sólo una de las dos tenía puerta.** Tres documentos de acuerdo entre ellos y en desacuerdo con el repositorio dan verde para siempre.

Eso es lo que cierra `instantanea_test.go` (nuevo el 04-09-2026): ata los cardinales que la foto publica a quien ya los computa del árbol. **Relojes escritos** contra `relojesDelCorpus()`, el mismo contador que vigila `ETAPAS.md`; **cobertura de la v1** contra el bloque `cobertura-v1` del `README.md`, que ya está atado al árbol por su propia puerta; y **puertas de CI** contra las invocaciones `puerta "` de los workflows. La cobertura se compara contra el README y no se recomputa aquí a propósito: una tercera implementación del mismo número es como se consigue que dos estén de acuerdo y la que mande sea la otra.

**Y esa puerta se estrenó contra el dato real, no contra una mutación**, que es la regla de la casa. Puesta sobre la instantánea publicada esta mañana (`80627ed`), sale roja con los tres contrastes disparando a la vez:

```
la instantanea publica 230 relojes escritos y el corpus tiene 252.
la instantanea dice 51,4 % de cobertura de la v1 y el README dice 56,7 %.
la instantanea publica 24 puertas de CI y en los workflows hay 25.
```

**Nace verde sobre el árbol de hoy y eso se dice en voz alta: llegó tarde.** Vigila algo real —lo acaba de demostrar sobre el commit anterior— pero no evitó el fallo que la trajo, lo documenta.

Lo que esta puerta **no** puede hacer, dicho aquí y no descubierto luego: no puede exigir que una **nota** se mueva, porque una nota es un juicio. Lo que hace es quitarle la excusa al juicio. Con los cardinales atados, una justificación que siga diciendo «230 relojes» rompe, y quien la escriba tiene que mirar el número de hoy antes de decidir si la nota se mueve.

## Qué mide el subíndice, y sobre todo qué NO mide

Mide **la plataforma open source publicable**: lo que un tercero puede descargar, arrancar, recalcular y auditar hoy, sin credenciales nuestras, sin conectores, sin IA y sin pasar por una caja. Es la mitad del producto que ya existe y se puede enseñar.

**Deja fuera cinco dimensiones enteras.** Pesan **31 de 109**, o sea el **28,4 %** del tablero.

| Excluida | Peso | Por qué está fuera | Etapa en que le toca |
|---|---|---|---|
| D5 Conectores WASM con conformidad | 7 | no hay ni ABI ni host WASM; no hay nada que publicar | E6 |
| D8 Riesgos con MAGERIT | 5 | nada construido | E7 |
| D12 IA verificable | 8 | **ya no es «nada construido»: están los cimientos** (verificador de citas por hash, interruptor, Ollama fuera de proceso, búsqueda, arnés de evals). Sigue fuera porque **ninguna de las cinco piezas de adopción existe**: hay motor y no hay producto que un tercero pueda descargar y usar | v1 (bloque IA) y E5 |
| D14 Open core self-serve | 6 | no hay licencia, ni checkout, ni carpeta de compras | E3 y E8 |
| D16 Cross-framework computado | 5 | no hay equivalencias escritas en ningún formato | E3 |

**La fila de D12 es la que hay que leer dos veces, y es la tentación de este tramo.** Su nota sube de 1,5 a 4,0, que es el movimiento más grande de la tarde. Meterla **dentro** subiría el subíndice publicado sin que nadie pueda descargar y usar nada de eso. Se queda fuera, y el precio de esa honestidad se paga en la sección siguiente, en la que el subíndice sube nueve céntimos mientras el global sube veinticinco.

## Lo que se movió, con su peso y su aritmética

Cuatro notas. Cada movimiento dice **qué razón escrita en la nota vieja dejó de ser cierta**; si la razón sigue en pie, la nota no se mueve aunque «se haya avanzado».

| # | Peso | Mañana | Tarde | Δ numerador | Dónde | Qué razón dejó de ser cierta |
|---|---|---|---|---|---|---|
| D2 | 8 | 9,3 | **9,5** | +1,6 | dentro | *«el núcleo es determinista»* era un eslogan. La **puerta 25** corre la suite entera con `PLAZUM_SIN_IA=1`, así que se comprueba en dos minutos |
| D3 | 6 | 6,5 | **7,0** | +3,0 | dentro | decía «230 relojes y **51,4 %**». Hoy **252** y **56,7 %**, computado por puerta en las dos direcciones. Medio punto y no más: los **7 de 15** marcos fuera del denominador y los **46 relojes sin escribir** siguen igual |
| D10 | 5 | 8,5 | **9,0** | +2,5 | dentro | decía «falta el tramo alto y **publicar la imagen**». La imagen está, y sobre todo la release **lleva el corpus dentro**: una máquina limpia pasa de **3 relojes a 222**. La otra media razón (Postgres) sigue en pie |
| D12 | 8 | 1,5 | **4,0** | **+20,0** | **fuera** | decía «**nada construido**». Falso desde las 14:47: `adaptadores/ia` con el verificador de citas por hash (521 líneas, 750 de test adversarial), interruptor, Ollama, `adaptadores/busqueda`, arnés de evals con su dorado. No sube de 4,0 porque **ninguna de las cinco piezas de `docs/ia.md` existe** |

```
dentro   +7,1 de numerador     fuera  +20,0     global  +27,1
```

**El 74 % del numerador que este tramo añadió cayó fuera del subíndice de plataforma.** No es un defecto de la medida: es lo que la medida está para decir. El titular del tramo fue la IA, y la IA está excluida por construcción.

De ahí salen los tres movimientos, y su desigualdad es la información:

| cifra | mañana | tarde | movimiento |
|---|---|---|---|
| subíndice de plataforma | 8,32 | **8,41** | **+0,09** |
| global de las 17 | 6,41 | **6,66** | **+0,25** |
| las 5 excluidas | 1,61 | **2,26** | **+0,65** |

**El global se mueve casi tres veces más que el subíndice.** Es exactamente lo contrario de lo que pasa cuando alguien redefine el tablero, y por eso las tres líneas se publican juntas: un subíndice que salta mientras el global no se mueve es la firma de una redefinición; un global que sube más que el subíndice es trabajo hecho donde el subíndice no mira.

**Y la frase que había que corregir de la versión anterior**, porque era cierta y ha dejado de serlo: aquella decía que las cinco excluidas *«se quedan clavadas en 1,61 antes y 1,61 después, y en nueve días no se movió ni una décima»*. Hoy se han movido **64 céntimos**, todos de D12.

**Las que NO se movieron teniendo excusa**, que es lo que demuestra que la regla se aplica:

- **D11 (intuitividad), sigue en 8,0**, con la capa visual entera detrás (CSS de 454 a 1.185 líneas, Inter autoalojada, marca, `@media print`, tres plantillas nuevas) y con las cifras huérfanas bajando de 5 a **1**. No sube porque la razón escrita —*«3 de sus 5 puertas propias siguen abiertas»*— sigue siendo cierta, y **una de las tres se ha alejado**: el TTFV pasó de 15m51s a **20m27s** contra un presupuesto de 15m. Y esto es lo importante: **no es que el producto haya empeorado, es que la medida dejó de ser ciega**; antes no cobraba las órdenes de terminal de los estados vacíos, que son 7m30s, el 37 %. El marcador anterior justificaba D11 con *«51 s de más sobre el TTFV»*, y hoy el hueco real es de **5m27s**. Una nota sostenida por un número que la favorecía seis veces por debajo del real.
- **D1 (modelo y temporalidad), sigue en 9,0.** Hay 22 relojes más, pero el mecanismo es el mismo y los dos huecos contados (39 relojes sin vigencia contrastable, 17 vigencias que no casan) siguen ahí. Más corpus sobre el mismo motor es D3, no D1.
- **D4 y D17**, quietas por lo mismo: la razón escrita en cada una no la ha tocado nadie.

## La tabla, con los pesos y las notas a la vista

Los pesos son los de `docs/diseno.md` §14 tras D-20 y las notas las de `docs/instantanea.md`. Están copiados aquí para que se pueda recalcular sin abrir otro fichero, y **no pueden separarse de su origen**: la puerta compara celda a celda.

| # | Dimensión | Peso | Nota hoy | Subíndice |
|---|---|---|---|---|
| D1 | Modelo de obligación y temporalidad | 12 | 9,0 | dentro |
| D2 | Determinismo y reproducibilidad | 8 | 9,5 | dentro |
| D3 | Cobertura por estratos y calendarios país | 6 | 7,0 | dentro |
| D4 | Implantación e2e, 5 clases con facetas | 8 | 7,5 | dentro |
| D5 | Conectores WASM con conformidad | 7 | 2,0 | fuera |
| D6 | Continuidad: certificado, escalado, silencio | 8 | 7,5 | dentro |
| D7 | Evidencia y valor probatorio | 6 | 9,5 | dentro |
| D8 | Riesgos con MAGERIT | 5 | 1,5 | fuera |
| D9 | Ligereza y huella | 3 | 9,7 | dentro |
| D10 | Instalación local y datacenter | 5 | 9,0 | dentro |
| D11 | Intuitividad y guiado | 7 | 8,0 | dentro |
| D12 | IA verificable | 8 | 4,0 | fuera |
| D13 | Extensibilidad | 4 | 9,8 | dentro |
| D14 | Open core self-serve | 6 | 1,5 | fuera |
| D15 | Legalidad del corpus | 6 | 9,0 | dentro |
| D16 | Cross-framework computado | 5 | 1,5 | fuera |
| D17 | Autoservicio radical | 5 | 6,0 | dentro |

## La aritmética, entera y a mano

Ponderado = suma de (peso × nota) / suma de pesos. Nada más.

**Las doce de dentro (78 puntos de peso):**

```
D1   12 × 9,0 = 108,0        D9    3 × 9,7 =  29,1
D2    8 × 9,5 =  76,0        D10   5 × 9,0 =  45,0
D3    6 × 7,0 =  42,0        D11   7 × 8,0 =  56,0
D4    8 × 7,5 =  60,0        D13   4 × 9,8 =  39,2
D6    8 × 7,5 =  60,0        D15   6 × 9,0 =  54,0
D7    6 × 9,5 =  57,0        D17   5 × 6,0 =  30,0
                                   -------------------
                                   suma = 656,3
                                   pesos = 78
                                   656,3 / 78 = 8,4141  ->  8,41
```

**Las cinco de fuera (31 puntos de peso):**

```
D5    7 × 2,0 = 14,0
D8    5 × 1,5 =  7,5
D12   8 × 4,0 = 32,0
D14   6 × 1,5 =  9,0
D16   5 × 1,5 =  7,5
                -------------------
                suma = 70,0
                pesos = 31
                70,0 / 31 = 2,2581  ->  2,26
```

**El global, que es la suma de los dos numeradores sobre la suma de los dos denominadores:**

```
(656,3 + 70,0) / (78 + 31) = 726,3 / 109 = 6,6633  ->  6,66
```

Esa última línea es la comprobación barata de que aquí no se ha partido nada mal: **el global no es el promedio de los otros dos números, es la fracción de sus sumas**, y si alguna vez deja de cuadrar, uno de los tres está mal.

## Los pesos de lo excluido: 31, no 32

El encargo que pidió este subíndice decía que las cinco excluidas «pesan **32** de 109». La tabla de `docs/diseno.md` §14, que es la vigente tras D-20, da **31**:

```
D5 7 + D8 5 + D12 8 + D14 6 + D16 5 = 31
```

**El bueno es 31, y el 32 tiene una explicación concreta**, que vale la pena escribir porque es un error de los que se repiten: la tabla de movimiento de D-20 tiene dos columnas, «antes» y «ahora», y D8 baja de **6** a **5**. Con el «antes» de D8 y el «ahora» de D5 y D12 sale exactamente 32. **Es una fila leída de la columna equivocada**, no una cuenta mal hecha; y por eso no se ve al recontar, sólo al mirar de qué columna sale cada sumando.

La puerta suma los pesos del árbol, así que este número no se puede volver a escribir a ojo: si alguien pone 32, rompe.

## ¿Se puede subir este subíndice sin construir nada?

**Sí, por dos caminos. Uno se cierra con una puerta y el otro no, y el que no se puede cerrar es el importante.**

**Camino 1: mover la membresía.** Sacar D17 (6,0) de las doce sube el subíndice de 8,41 a **8,58** —`626,3 / 73`— sin que se escriba una línea de código. Y este tramo trae la tentación en su forma más golosa, que es la contraria: **meter D12 dentro** ahora que su nota ha subido a 4,0. Éste **sí** se cierra mecánicamente, con tres cosas a la vez:

1. La puerta cruza `dentro ∪ fuera` con las 17 dimensiones de `docs/diseno.md` **en las dos direcciones**: una dimensión que no esté clasificada rompe, y una clasificada que no exista rompe.
2. Los dos cardinales (12 dentro, 5 fuera) y los dos pesos (78 y 31) se publican y se comprueban, así que mover una dimensión cambia **cuatro números publicados** a la vez.
3. Y **el global se publica al lado**. Ésta es la parte que hace de detector: **mover la membresía cambia el subíndice y NO cambia el global**. Un subíndice que salta mientras el global no se mueve es la firma de una redefinición, y se ve sin saber nada del proyecto.

**Camino 2: subir una nota.** Escribir 9,0 donde hoy pone 7,0 sube el subíndice y **ninguna puerta lo puede impedir**, porque una nota es un juicio y un test no juzga. Lo que sí se hace, y ahora es una cosa más que antes:

- La nota vive en `docs/instantanea.md` **con la frase que la sostiene al lado**, y esa frase tiene que nombrar un número medido o un comando.
- La puerta exige que la nota de aquí y la de allí sean **la misma**, así que no hay una segunda copia que se mueva sola.
- **Y desde hoy, los cardinales que esa frase cita están atados al árbol** (`instantanea_test.go`). Eso no impide inflar una nota, pero impide sostenerla con un número que ya no es verdad, que es como se infló D11 con sus «51 segundos».
- Y otra vez el global: **subir una nota sube los DOS números**. Un movimiento honesto mueve los dos; un movimiento de definición mueve uno.

Lo que queda sin cerrar, dicho con su nombre: **nada impide inflar una nota**. Contra eso sólo hay la disciplina de justificarla.

## Las siete notas que se han movido desde el 26-08, que es la base de la sección siguiente

La tabla de arriba mide **esta tarde contra esta mañana**. Ésta mide **hoy contra el 26-08-2026**, que es la línea de partida de la descomposición, y por eso las dos tienen que estar: la de arriba dice qué hizo este tramo, y ésta dice de dónde viene el número que se publica. La puerta reconstruye las notas del 26-08 **de esta tabla**, así que una fila que falte aquí se lleva por delante la descomposición entera.

| # | 26-08 | hoy | qué razón de la nota del 26-08 dejó de ser cierta |
|---|---|---|---|
| D1 | 8,0 | **9,0** | decía «el censo identificó **dos primitivas que faltan**». Las dos están y **encendidas**, y los relojes escritos pasan de 8 a **252** |
| D2 | 9,3 | **9,5** | *«el núcleo es determinista»* era un eslogan; la puerta 25 lo comprueba en dos minutos con `PLAZUM_SIN_IA=1` |
| D3 | 4,5 | **7,0** | decía «sólo **4 de 31** paquetes tienen contenido». Hoy **21 de 33**, con 252 relojes y **56,7 %** de cobertura estricta computada por puerta |
| D4 | 7,0 | **7,5** | decía «sólo mide sobre los paquetes que existen». El mecanismo no ha cambiado; su base sí, de 4 paquetes a 21 |
| D10 | 8,5 | **9,0** | decía «falta el tramo alto y **publicar la imagen**». La imagen está y la release lleva el corpus dentro |
| D11 | 7,5 | **8,0** | decía «todavía no se puede guardar: todas las rutas son GET». Refutado: la UAR escribe decisiones y los seis pasos contestan 200 |
| D12 | 1,5 | **4,0** | decía «**nada construido**». Están los cimientos de la IA, con la puerta antialucinación demostrada |

**Diez de las diecisiete no se han movido en nueve días**, y cinco de esas diez son cuatro de las excluidas más D6. Eso también es lo que la medida está para decir.

## Cuánto de la distancia es trabajo y cuánto es el denominador

El 26-08-2026 el global honesto era **6,13** con los pesos de hoy. El subíndice publicado es **8,41**. Entre los dos hay **2,29 puntos**, y esa distancia se parte midiendo cada mitad por separado:

| qué se cambia | cifra | movimiento |
|---|---|---|
| nada (punto de partida: global, notas del 26-08) | 6,13 | — |
| **sólo el denominador** (subíndice con las notas del 26-08) | 7,92 | **+1,79** |
| **sólo las notas** (global con las notas de hoy) | 6,66 | **+0,54** |
| las dos cosas (el subíndice publicado) | 8,41 | +2,29 |

**Por qué restar las dos cifras de al lado puede dar un céntimo menos.** Los movimientos se redondean de la resta sin redondear y las cuatro cifras se redondean cada una por su cuenta. La **diferencia de dos redondeos no es el redondeo de la diferencia**, y quien comprueba merece leerlo aquí antes de pensar que sobra un decimal.

**El 78 % de la distancia sigue siendo el cambio de definición y el 24 % es trabajo hecho.** La proporción mejora respecto a la medición anterior, donde el trabajo explicaba el 13 %: en nueve días el trabajo aportaba 0,29 y hoy aporta 0,54. Sigue siendo verdad lo que había que decir entonces y hay que repetir ahora: **este subíndice sube 1,79 puntos por dejar cinco casillas fuera de la foto**, y quien lea 8,41 sin leer esta tabla se lleva una idea equivocada del proyecto.

## Cómo se ha visto fallar esta puerta

Una puerta que nunca se ha visto fallar no es una puerta. Las cinco formas de romper `subindice_test.go` se probaron el 04-09-2026 sobre el árbol commiteado, aplicando y restaurando en comandos separados. Siguen valiendo, con los números de hoy.

| # | Qué se rompió | Qué se puso rojo |
|---|---|---|
| M1 | bajar D9 de 9,7 a 9,6 **en los dos ficheros** | **sólo el numerador**. Los dos ponderados siguieron dando lo mismo redondeados, o sea que **sin el numerador esta bajada habría pasado en verde** |
| M2 | mover D17 de `dentro` a `fuera` | ocho errores a la vez, y **el global no se movió**, que es exactamente el detector que describe la sección anterior |
| M3 | borrar la fila de D5 de la tabla | «el marcador no dice si [D5] están dentro o fuera del subíndice» |
| M4 | aflojar el ancla del identificador en el lector de pesos | tres tests: el control negativo y las dos puertas que leen la rúbrica |
| M5 | cambiar el peso copiado de D1 de 12 a 14 | «docs/marcador.md dice que D1 pesa 14 y docs/diseno.md dice 12» |

Y la sexta, que es de la puerta nueva y **no es una mutación sino dato real**, que es como se estrena mejor: `instantanea_test.go` puesto sobre la instantánea publicada esta mañana (`80627ed`) sale rojo con tres contrastes a la vez (230 relojes contra 252, 51,4 % contra 56,7 %, 24 puertas contra 25). Ninguna de las cinco de arriba la habría encontrado, porque las cinco mutan documentos y el fallo estaba en que los documentos estaban de acuerdo.

## Cómo recalcular esto sin fiarse de nadie

```bash
go test . -run TestElSubindiceDePlataformaLoComputaUnTestYNoUnaPersona -v
go test . -run TestLaInstantaneaNoPublicaCardinalesQueElArbolYaDesmiente -v
```

El primero lee los pesos de `docs/diseno.md`, las notas de `docs/instantanea.md` y la membresía y las cifras de este fichero, computa las tres y las contrasta. El segundo comprueba que las notas no se apoyan en cardinales que el árbol ya desmiente. **Hacen falta los dos**: el primero solo garantiza que tres documentos digan lo mismo, y eso es exactamente lo que pasaba el 04-09 por la mañana mientras los tres estaban equivocados.

Y a mano, que es más corto de lo que parece: multiplica cada peso por su nota en la tabla de arriba, suma las doce que dicen `dentro`, divide entre 78, y compara.
