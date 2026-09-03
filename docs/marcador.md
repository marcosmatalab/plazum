# El marcador: el subíndice de plataforma, al lado del global y nunca en su lugar

> **Fecha de esta medición: 04-09-2026.** Las notas salen de `docs/instantanea.md`, los pesos de `docs/diseno.md` §14 (tabla vigente tras D-20), y las tres cifras publicadas las computa `subindice_test.go` del dato y las contrasta con lo que dice este fichero, con igualdad exacta y en las dos direcciones. Si una cifra sube, rompe; si baja, rompe igual.

## Las tres cifras

<!-- marcador:inicio -->
- **Subíndice de plataforma open source, publicable: 8,32**, sobre **12 dimensiones** y **78 puntos de peso**, con numerador **649,2**.
- **Global de las 17 dimensiones: 6,41**, sobre **109 puntos de peso**, con numerador **699,2**.
- **Las 5 dimensiones excluidas, medidas aparte: 1,61**, sobre **31 puntos de peso**, con numerador **50,0**.
<!-- marcador:fin -->

**El numerador se publica y no es adorno: es la mitad de la puerta que tiene dientes.** Un ponderado con dos decimales se traga un movimiento pequeño: bajar D9 de 9,7 a 9,6 cambia el subíndice de 8,3231 a 8,3192, que redondea al mismo **8,32**, así que una nota podría bajar sin que nada se pusiera rojo. El numerador no: baja de **649,2** a **648,9**, y eso es un dígito distinto en un número publicado. La puerta compara los cuatro valores de cada línea, así que **cualquier movimiento de cualquier nota, en cualquier dirección, rompe**.

Las tres líneas se publican juntas siempre. **La de en medio es el número del producto**; la primera dice cuánto de la plataforma que hoy se puede descargar, arrancar y publicar está hecha; la tercera dice cuánto vale lo que la primera se deja fuera, para que nadie tenga que ir a buscarlo.

## Qué mide el subíndice, y sobre todo qué NO mide

Mide **la plataforma open source publicable**: lo que un tercero puede descargar, arrancar, recalcular y auditar hoy, sin credenciales nuestras, sin conectores, sin IA y sin pasar por una caja. Es la mitad del producto que ya existe y se puede enseñar.

**Deja fuera cinco dimensiones enteras, y las cinco por el mismo motivo: no están construidas y no les toca todavía.** No es una lista de cosas poco importantes: pesan **31 de 109**, o sea el **28,4 %** del tablero, y una de ellas (D12, IA verificable) es la que más peso ganó con D-20 precisamente porque es la que más promete.

| Excluida | Peso | Por qué está fuera | Etapa en que le toca |
|---|---|---|---|
| D5 Conectores WASM con conformidad | 7 | no hay ni ABI ni host WASM; no hay nada que publicar | E6 |
| D8 Riesgos con MAGERIT | 5 | nada construido | E7 |
| D12 IA verificable | 8 | nada construido; el interruptor `PLAZUM_SIN_IA` existe y el adaptador no | v1 (bloque IA) y E5 |
| D14 Open core self-serve | 6 | no hay licencia, ni checkout, ni carpeta de compras | E3 y E8 |
| D16 Cross-framework computado | 5 | no hay equivalencias escritas en ningún formato | E3 |

**Y aquí va la advertencia que hace este documento contestable en vez de creíble**, porque es la regla del propio estratega: *si un cambio de definición sube un número sin que suba nada real, se dice en voz alta.* Este subíndice **es un cambio de definición**, y sube el número. Cuánto exactamente, medido y no intuido, está tres secciones más abajo.

## La tabla, con los pesos y las notas a la vista

Los pesos son los de `docs/diseno.md` §14 tras D-20 y las notas las de `docs/instantanea.md`. Están copiados aquí para que se pueda recalcular sin abrir otro fichero, y **no pueden separarse de su origen**: la puerta compara celda a celda.

| # | Dimensión | Peso | Nota hoy | Subíndice |
|---|---|---|---|---|
| D1 | Modelo de obligación y temporalidad | 12 | 9,0 | dentro |
| D2 | Determinismo y reproducibilidad | 8 | 9,3 | dentro |
| D3 | Cobertura por estratos y calendarios país | 6 | 6,5 | dentro |
| D4 | Implantación e2e, 5 clases con facetas | 8 | 7,5 | dentro |
| D5 | Conectores WASM con conformidad | 7 | 2,0 | fuera |
| D6 | Continuidad: certificado, escalado, silencio | 8 | 7,5 | dentro |
| D7 | Evidencia y valor probatorio | 6 | 9,5 | dentro |
| D8 | Riesgos con MAGERIT | 5 | 1,5 | fuera |
| D9 | Ligereza y huella | 3 | 9,7 | dentro |
| D10 | Instalación local y datacenter | 5 | 8,5 | dentro |
| D11 | Intuitividad y guiado | 7 | 8,0 | dentro |
| D12 | IA verificable | 8 | 1,5 | fuera |
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
D2    8 × 9,3 =  74,4        D10   5 × 8,5 =  42,5
D3    6 × 6,5 =  39,0        D11   7 × 8,0 =  56,0
D4    8 × 7,5 =  60,0        D13   4 × 9,8 =  39,2
D6    8 × 7,5 =  60,0        D15   6 × 9,0 =  54,0
D7    6 × 9,5 =  57,0        D17   5 × 6,0 =  30,0
                                   -------------------
                                   suma = 649,2
                                   pesos = 78
                                   649,2 / 78 = 8,3231  ->  8,32
```

**Las cinco de fuera (31 puntos de peso):**

```
D5    7 × 2,0 = 14,0
D8    5 × 1,5 =  7,5
D12   8 × 1,5 = 12,0
D14   6 × 1,5 =  9,0
D16   5 × 1,5 =  7,5
                -------------------
                suma = 50,0
                pesos = 31
                50,0 / 31 = 1,6129  ->  1,61
```

**El global, que es la suma de los dos numeradores sobre la suma de los dos denominadores:**

```
(649,2 + 50,0) / (78 + 31) = 699,2 / 109 = 6,4147  ->  6,41
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

**Camino 1: mover la membresía.** Sacar D17 (6,0) de las doce sube el subíndice de 8,32 a **8,48** —`619,2 / 73`— sin que se escriba una línea de código. Éste **sí** se cierra mecánicamente, con tres cosas a la vez:

1. La puerta cruza `dentro ∪ fuera` con las 17 dimensiones de `docs/diseno.md` **en las dos direcciones**: una dimensión que no esté clasificada rompe, y una clasificada que no exista rompe.
2. Los dos cardinales (12 dentro, 5 fuera) y los dos pesos (78 y 31) se publican y se comprueban, así que mover una dimensión cambia **cuatro números publicados** a la vez.
3. Y **el global se publica al lado**. Ésta es la parte que hace de detector, y por eso «al lado, nunca en su lugar» no es cortesía: **mover la membresía cambia el subíndice y NO cambia el global**. Un subíndice que salta mientras el global no se mueve es la firma de una redefinición, y se ve sin saber nada del proyecto.

**Camino 2: subir una nota.** Escribir 9,0 donde hoy pone 6,5 sube el subíndice y **ninguna puerta lo puede impedir**, porque una nota es un juicio y un test no juzga. Lo que sí se hace, y es todo lo que se puede hacer:

- La nota vive en `docs/instantanea.md` **con la frase que la sostiene al lado**, y esa frase tiene que nombrar un número medido o un comando.
- La puerta exige que la nota de aquí y la de allí sean **la misma**, así que no hay una segunda copia que se mueva sola.
- Y otra vez el global: **subir una nota sube los DOS números**. Un movimiento honesto mueve los dos; un movimiento de definición mueve uno. Ese es el discriminador, y es mecánico.

Lo que queda sin cerrar, dicho con su nombre: **nada impide inflar una nota**. Contra eso sólo hay la disciplina de justificarla, y este documento la ejerce en la sección siguiente, moviendo cuatro notas y dejando quietas trece.

## Cuánto de la distancia es trabajo y cuánto es el denominador

El 26-08-2026 el global honesto era **6,13** con los pesos de hoy. El subíndice publicado es **8,32**. Entre los dos hay **2,20 puntos**, y esa distancia se parte en tres, midiendo cada mitad por separado:

| qué se cambia | cifra | movimiento |
|---|---|---|
| nada (punto de partida: global, notas del 26-08) | 6,13 | — |
| **sólo el denominador** (subíndice con las notas del 26-08) | 7,92 | **+1,79** |
| **sólo las notas** (global con las notas de hoy) | 6,41 | **+0,29** |
| las dos cosas (el subíndice publicado) | 8,32 | +2,20 |

**El 82 % de la distancia es el cambio de definición y el 13 % es trabajo hecho** (el 5 % restante es la interacción entre los dos). Dicho sin adornos: **este subíndice sube 1,79 puntos por dejar cinco casillas vacías fuera de la foto**, y sólo 0,29 por lo que se construyó en nueve días. Quien lea 8,32 y no lea esta tabla se está llevando una idea equivocada del proyecto, y por eso las tres cifras van juntas y esta tabla va aquí y no en un anexo.

**Y el reverso, que también es verdad:** las cinco excluidas se quedan clavadas en **1,61 antes y 1,61 después**, exactamente el mismo número. En nueve días de campaña no se movió ni una décima de las cinco. Eso no es un defecto de la medida: es lo que la medida está para decir.

## Qué pasaría con el global

Nada: **el global se calcula sobre las 17 y este subíndice no lo toca**. Se publica arriba, en el mismo bloque, y hoy vale **6,41**.

Lo único que este subíndice le hace al global es hacerlo **más fácil de leer mal**, porque pone al lado un número mayor. Contra eso está la tabla de la sección anterior, y esta frase: **el número del producto es 6,41**. El 8,32 responde a otra pregunta, que es *«¿está publicable la parte que hoy existe?»*, y a esa pregunta la respuesta honesta es que sí y que se nota.

## Las cuatro notas que se movieron, y las trece que no

Cada movimiento tiene que decir **qué razón escrita en la nota vieja dejó de ser cierta**, con la evidencia de hoy. Si la razón sigue en pie, la nota no se mueve aunque «se haya avanzado».

| # | 26-08 | 04-09 | qué razón de la nota vieja dejó de ser cierta |
|---|---|---|---|
| D1 | 8,0 | **9,0** | decía «el censo identificó **dos primitivas que faltan**». Las dos están y **encendidas**: `primitivas_alcanzables_test.go` informa *«hoy no hay ninguna primitiva apagada ni sin cablear»*. Y los relojes escritos pasan de 8 a **230**. No sube más: **39 relojes** cuya vigencia nadie puede contrastar y **17 vigencias** que no casan con la fecha que declara su fuente |
| D3 | 4,5 | **6,5** | decía «sólo **4 de 31** paquetes tienen contenido». Hoy son **21 de 33**, con **230 relojes** y un **51,4 %** de cobertura estricta sobre la v1 computado por puerta. No sube más: **7 de los 15** marcos de la v1 siguen fuera del denominador y hay **46 relojes identificados y sin escribir** |
| D4 | 7,0 | **7,5** | decía «baja porque **sólo mide sobre los paquetes que existen**». El mecanismo no ha cambiado; lo que ha cambiado es su base, de 4 paquetes con contenido a 21. Por eso el movimiento es medio punto y no más |
| D11 | 7,5 | **8,0** | decía «baja porque **todavía no se puede guardar nada**: todas las rutas son GET». Refutado hoy: `superficies/uar` escribe decisiones (`PostFormValue`), `/entrar` y `/primer-admin` son POST, y **los seis pasos del camino contestan 200** medidos contra el binario desde un directorio vacío. No sube más: **3 de sus 5 puertas propias siguen abiertas**, con cardinal (2 órdenes de terminal, 5 cifras huérfanas de 14, 51 s de más sobre el TTFV) |

**Y la que NO se movió teniendo excusa, que es la que demuestra que la regla se aplica:** D6 (continuidad) sigue en **7,5**. Se construyeron `superficies/escalado`, `adaptadores/escalador` y `adaptadores/canal`, y aun así la razón escrita en la nota vieja —*«falta el planificador propio: hoy quien apunta que ha corrido es un temporizador del operador»*— **sigue siendo cierta palabra por palabra**: medido hoy, `plazum latido` responde *«Programa `plazum latido ciclo` cada hora, en tu cron o en un temporizador de systemd»*. Lo mismo con D17, que sigue en 6,0 porque su razón (falta la carpeta de compras y el autoservicio de licencia) tampoco se ha tocado.

## Cómo se ha visto fallar esta puerta

Una puerta que nunca se ha visto fallar no es una puerta. Estas son las cinco formas de romperla que se probaron el 04-09-2026, sobre el árbol ya commiteado, aplicando y restaurando en comandos separados.

| # | Qué se rompió | Qué se puso rojo |
|---|---|---|
| M1 | bajar D9 de 9,7 a 9,6 **en los dos ficheros** | **sólo el numerador**: «publica un numerador de 649.2 y el dato da 648.9», y lo mismo con el global. Los dos ponderados siguieron dando 8,32 y 6,41, o sea que **sin el numerador esta bajada habría pasado en verde**. Es la demostración de por qué se publica |
| M2 | mover D17 de `dentro` a `fuera` | ocho errores a la vez: 12 dimensiones contra 11, 78 de peso contra 73, numerador 649,2 contra 619,2, valor 8,32 contra **8,48**; y en las excluidas, 5 contra 6, 31 contra 36, 50,0 contra 80,0, 1,61 contra 2,22. **Y el global no se movió: siguió en 6,41**, que es exactamente el detector que describe la sección anterior |
| M3 | borrar la fila de D5 de la tabla | «el marcador no dice si [D5] están dentro o fuera del subíndice» |
| M4 | aflojar el ancla del identificador en el lector de pesos (`(D\d{1,2})\s*\|` a `(D\d{1,2})[^|]*\|`) | tres tests: el control negativo («ha casado 1 filas y esperaba 2») y las dos puertas que leen la rúbrica («docs/diseno.md da 4 dimensiones con peso y la rúbrica tiene 17») |
| M5 | cambiar el peso copiado de D1 de 12 a 14 | «docs/marcador.md dice que D1 pesa 14 y docs/diseno.md dice 12» |

Las cinco compilan (`go build ./...` y `go vet ./...` limpios con la mutación puesta), que es la trampa de la que este repositorio ya se ha caído: una mutación que no compila no produce líneas `--- FAIL` y se lee igual que una mutación no cazada.

## Cómo recalcular esto sin fiarse de nadie

```bash
go test . -run TestElSubindiceDePlataformaLoComputaUnTestYNoUnaPersona -v
```

La puerta lee los pesos de `docs/diseno.md`, las notas de `docs/instantanea.md` y la membresía y las cifras de este fichero, computa las tres y las contrasta. **En las dos direcciones**: que una cifra baje rompe igual que si sube, porque un número que sólo está vigilado hacia arriba es medio número vigilado.

Y a mano, que es más corto de lo que parece: multiplica cada peso por su nota en la tabla de arriba, suma las doce que dicen `dentro`, divide entre 78, y compara.
