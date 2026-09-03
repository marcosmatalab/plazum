# Hallazgos de la persistencia del alcance

Frente A del tramo 1 (04-09-2026). Aquí vive lo que salió de convertir las
respuestas de la entrevista en estado de la cuenta: lo que se cerró, lo que se
midió y lo que se queda abierto con su cardinal.

Todos los números de este documento salen de una ejecución de este día, contra
el binario de verdad y el corpus del repositorio. Ninguno está escrito de
memoria.

---

## 1. Lo que había, y por qué estaba bien mientras duró

Hasta hoy toda ruta de `superficies/pantallas` era GET, y el encabezado del
paquete lo explicaba: las respuestas viajaban en la dirección, la página se
compartía, y sobre todo *no se fingía una persistencia que no existía*.
`docs/instantanea.md` lo dejó escrito: **un botón que no guarda sería la peor
mentira de esa pantalla**.

Era cierto porque no había dónde guardar. Con `adaptadores/usuarios` construido,
el alcance pasa a ser estado de la cuenta, y por eso ahora sí hay un POST.

---

## 2. La cadena entera, medida

Desde un directorio vacío, con `curl`, sobre `plazum serve` de verdad:

| paso | petición | código |
|---|---|---|
| sin instalar | `GET /alcance` | 303 a `/primer-admin` |
| instalación | `GET /primer-admin` | 200 |
| instalación | `POST /primer-admin` | 303 a `/` |
| entrevista | `GET /alcance` | 200, con 38 formularios de guardado |
| ataque | `POST /alcance` **sin token** | **403** |
| contestar | `POST /alcance` con token | 303 a `/alcance#p-ens.q.categoria` |
| disco | `respuestas.json` | escrito, versión 1, con la respuesta dentro |
| cerrar sesión y volver a entrar | `GET`+`POST /entrar` | 200 y 303 |
| volver | `GET /alcance` **a secas** | 200, la pregunta sale respondida |

Los 38 formularios son 2 por pregunta viva más los globales: 19 preguntas.

---

## 3. HALLAZGO 1 (P1, PRE-EXISTENTE, ARREGLADO): el exportador se caía entero
   con la PRIMERA pregunta de la entrevista

`plazum alcance` mandaba al puente toda respuesta de un paquete que declarase el
puente **en algún atributo**, sin mirar el atributo concreto. El puente exige
valor para la forma `con_valor`, la entrevista web solo sabe contestar sí/no, y
el resultado era que la orden **salía con código 1 y no escribía nada**, con
este mensaje:

```
traduciendo las respuestas de urn:es:rd:2022:311: urn:es:rd:2022:311:
la respuesta 0 sobre sistema.ambito lleva valor por su forma "con_valor" y llega vacia.
  No se exporta a medias: un alcance al que le faltan hechos deriva menos
  obligaciones y no lo dice
```

La pregunta que lo dispara es **`ens.q.categoria`, que es la PRIMERA que la
entrevista sugiere**: la pantalla dice literalmente «empieza por esta». O sea que
el camino más corto del producto (responder la sugerida, exportar) terminaba en
un error escrito para quien cablea.

No lo vio nadie porque los tests del exportador elegían dos preguntas booleanas
a mano (`ens.q.datos_personales`, `ens.q.externalizacion`). Lo encontró la ida y
vuelta desde la cuenta, que exporta lo que la persona haya contestado y por tanto
puede tocar cualquier pregunta que la entrevista pinte.

**Arreglo**: el exportador mira la forma del puente **por atributo** y no por
paquete, y una respuesta `con_valor` cae en su propio cubo, contada y nombrada,
igual que `SinPuente`. Un exportador que se cae no exporta a medias: no exporta
nada.

---

## 4. HALLAZGO 2 (P1, ABIERTO, NO ES DE ESTA COLUMNA): de las 19 preguntas que
   ve un CISO, solo 3 llegan al motor

Medido con el binario de hoy, contestando que sí a todas:

| lista | preguntas | traducidas a hechos | sin puente | con valor |
|---|---|---|---|---|
| corta (la que se ve) | **19** | **3** | 14 | 2 |
| larga (`?ver=todas`) | **42** | **3** | 25 | 14 |

Los 25 sin puente, por paquete: `urn:iso-iec:27001:2022` 8,
`urn:plazum:demo:empresa` 6, `urn:eu:reg:2024:1689` 3, `urn:iso-iec:42001:2023`
3, y uno cada uno `urn:es:rd:2021:43`, `urn:eu:dir:2022:2555`,
`urn:eu:reg:2016:679`, `urn:eu:reg:2017:745`, `urn:eu:reg:2022:2554`.

Los 14 con valor son todos del ENS: `ens.q.ambito`, `ens.q.categoria` y las doce
de dimensión (`ens.q.informacion.*`, `ens.q.servicio.*`).

**Qué significa para el comprador**: contesta diecinueve preguntas, tarda siete
minutos, y su `alcance.json` sale con tres hechos. La orden lo dice con todas
las letras en cada ejecución, así que no es silencio; pero es la brecha entre lo
que la entrevista PREGUNTA y lo que el motor USA, y hoy es de 16 sobre 19.

**No se arregla desde esta columna.** Cerrarlo son dos trabajos distintos:
declarar el puente en los paquetes que no lo tienen (corpus), y que la entrevista
sepa preguntar un valor y no solo sí/no (`nucleo/pantalla` + `superficies/pantallas`,
y la parte del núcleo no es mía).

---

## 4 bis. HALLAZGO 3 (P0, INTRODUCIDO POR ESTE TRABAJO Y ARREGLADO): una
   sesión anónima rompía `/alcance`

Salió de la pasada contra el atacante, no de leer el diff.

`serve.SujetoDe` devuelve **dos cosas distintas** que `quienOpera` estaba
leyendo como una:

- «no hay sesión» → cadena vacía;
- «hay sesión y **no ha entrado nadie**» → la efímera con sujeto
  `serve.SujetoAnonimo` (`anonimo:sin-autenticar`).

Y esa segunda **la reparte el propio producto**: basta abrir `/entrar`, porque
el formulario necesita una sesión para poder emitir su token CSRF. El godoc de
`SujetoDe` lo dice con esas palabras («quien monte pantallas tiene que comprobar
además EsAnonimo») y `quienOpera` no lo comprobaba.

Medido sobre el binario, con el guardado ya cableado y **antes** del arreglo:

```
GET  /alcance sin cookie ............ 200
GET  /entrar (da cookie anonima) .... 200
GET  /alcance con esa cookie ........ 500     <-- pagina rota
POST /alcance con esa cookie ........ 500
```

O sea: **cualquiera que intentara entrar y después mirara la entrevista veía un
500.** El sujeto con dos puntos llegaba al almacén y
`usuarios.NormalizarUsuario` lo rechazaba, que es justo para lo que esa
prohibición existe; la guarda salvaba de escribir en un cajón común y no salvaba
de la página rota.

Después del arreglo, mismo guion: **200** y **403**.

**Y alcanzaba también a `superficies/uar`**, que usa el mismo `quienOpera`: sin
esto habría anotado una decisión en el ledger a nombre de
`anonimo:sin-autenticar`, que es peor que una página rota porque no se deshace.
El arreglo va en `quienOpera` (el único fichero que conoce `superficies/serve`) y
por eso cubre las dos superficies a la vez.

---

## 5. HALLAZGO 4 (P2, ARREGLADO): dos ficheros que se llamaban casi igual

El almacén nuevo iba a llamarse `alcances.json` y vivir en el mismo directorio
que el `alcance.json` de `plazum serve --alcance`, que es otra cosa (los hechos
ya derivados de la organización, los lee el calendario). Dos ficheros a una
letra de distancia, con contenidos distintos, en el mismo sitio: quien restaure
el equivocado desde su copia no se entera.

Se llama `respuestas.json` y la bandera es `--respuestas`. El nombre de dentro
del código puede ser el del concepto; el de fuera tiene que ser el que no se
confunde.

---

## 6. Por qué campo casa cada emparejamiento nuevo

Los cuatro sitios donde este trabajo empareja dos conjuntos, con su campo y si
está firmado (invariante 7):

| emparejamiento | campo | ¿firmado? |
|---|---|---|
| respuesta guardada ↔ pregunta del corpus | **id de pregunta**, clave del mapa en disco y clave del índice derivado de `pantalla.Derivar` | **No.** `respuestas.json` es estado local de la cuenta, en 0600; no es ledger ni expediente y no se presenta como prueba de nada. |
| envío del formulario ↔ pregunta del corpus | **id de pregunta**, campo `pregunta` del POST, contrastado contra el índice antes de escribir | No. Lo que lo protege no es una firma: es que el conjunto de ids válidos lo pone el corpus instalado y no la petición. |
| fila de `alcance.json` ↔ pregunta del corpus | **`pregunta`** del bloque `respuestas` | No. `alcance.json` es un fichero de trabajo del operador. |
| respuesta exportada ↔ respuesta reimportada | el par **(id de pregunta, respuesta)**, comparado en los DOS sentidos | No, y la puerta no afirma nada sobre autoría: afirma conservación. |

En ninguno se empareja por posición, por orden de lista ni por subcadena. El
campo `campo` del bloque de respuestas **no** se usa para emparejar y se dice por
qué: dos preguntas pueden pedir el mismo campo sobre instancias distintas.

La dirección contraria se recorre en los cuatro:

- del almacén al corpus: las respuestas guardadas cuya pregunta el corpus ya no
  declara se **cuentan y se enseñan** (`alcance.guardado.huerfanas`), no se
  descartan en silencio;
- del fichero a la cuenta: las cinco clases de fila se cuentan y la suma se
  comprueba con `nucleo/metrica.Cuadra`;
- de la cuenta al fichero: los cinco destinos de una respuesta, igual.

---

## 7. La ida y vuelta se comprueba por CONSERVACIÓN, no por recuento

«Salen tantas como entraron» deja pasar dos cambios que se cancelan: una
respuesta que se pierde y otra que aparece dan el mismo cardinal y son un alcance
distinto. Y no es un caso de laboratorio: la exportación recorre dos listas (`si`
y `no`) y la importación las vuelve a juntar, así que un «sí» convertido en «no»
por el camino conserva el cardinal, conserva el conjunto de preguntas y cambia el
veredicto de todas las obligaciones que dependan de ella.

`TestLaIdaYLaVueltaDelAlcanceConservanCadaRespuesta` compara pareja a pareja en
los dos sentidos, con doce preguntas reales sacadas del corpus (mitad sí y mitad
no, a propósito: con solo síes, un fallo que convirtiera todos los noes en síes
pasaría sin ponerse rojo), y además pasa los cubos por `metrica.Cuadra`.

Y lleva su control negativo: `TestLaComparacionPorIdentidadVeLoQueElCardinalNoVe`
fabrica el caso que el recuento deja pasar y exige que la comparación por
identidad lo vea.

**El bloque `respuestas` del `alcance.json` tuvo que existir para que la vuelta
fuera posible.** El bloque `hechos` es con pérdidas a propósito (un «no» no
afirma nada, y una respuesta sin puente tampoco), así que de un fichero con solo
hechos no se recuperan las respuestas: quien exportara e importara vería
desaparecer todos sus «no» sin una línea en ningún sitio. El formato ya declaraba
ese bloque (`campo`, `valor`, `pregunta`) desde el principio y el exportador no
lo escribía.

---

## 8. Lo que NO se hizo, con su cardinal

1. **La entrevista sigue sin saber preguntar un valor.** 14 de las 42 preguntas
   del corpus (2 de las 19 vivas) piden una categoría o un nivel y solo se
   pueden contestar sí/no. Ver hallazgo 2. Toca `nucleo/pantalla`, que no es de
   esta columna.
2. **25 de 42 preguntas son de paquetes que no declaran el puente.** Es trabajo
   de corpus, no de esta columna.
3. **Las respuestas guardadas no llegan al calendario web.** `plazum serve
   --alcance` sigue leyendo un `alcance.json` de disco, así que hoy hay que
   exportar y reiniciar. Cerrarlo es de `superficies/calendario`, que es de otro
   frente: ver el punto 9.
4. **No hay borrado de cuenta.** Se puede vaciar el alcance («empezar de cero»),
   no borrar la fila de la cuenta del fichero. Ningún flujo del producto lo pide
   todavía.
5. **Una sola instalación, un solo fichero.** El almacén se lee entero en cada
   arranque y se reescribe entero en cada cambio. Con las cuentas que hoy admite
   el producto (una instalación, un puñado de personas) no compensa nada más; el
   día que compense, lo que hay que cambiar es el almacén y no la superficie,
   porque el interfaz `pantallas.Alcances` ya está entre medias.
6. **Las puertas con `-race` no se han podido ejecutar en esta máquina** (3 de
   las 24: `-race` exige cgo y aquí `CGO_ENABLED=0`). El código nuevo lleva
   mutex y escritura concurrente, así que **eso es un hueco real de esta
   validación**, no una formalidad: quien integre tiene que mirar que esas tres
   puertas salen verdes en CI antes de dar por bueno el guardado concurrente. En
   local se comprueba lo que se puede sin detector: 24 gorutinas contestando
   preguntas distintas a la vez y las 24 respuestas en disco al reabrir.

---

## 9. Lo que se le pide al frente C (`superficies/calendario/`)

Su pantalla de calendario pide hoy **dos órdenes de terminal** (`plazum alcance`
y `plazum serve --alcance`) y esas dos órdenes son **3m45s de los 15m51s del
TTFV, el 24 %**. Con las respuestas ya guardadas en la cuenta, la primera de las
dos deja de hacer falta para la pantalla: el calendario puede leer el alcance de
la cuenta que está mirando en vez de un fichero pasado por bandera.

No se quitan desde aquí porque esa pantalla es de su columna. Lo que esta
columna deja puesto para que pueda:

- `adaptadores/usuarios/alcances.Almacen`, con `De(ctx, usuario)`;
- `cmd/plazum/serve.go` ya lo abre y ya sabe el sujeto de la sesión
  (`quienOpera`);
- `cmd/plazum` ya sabe traducir de respuestas a hechos por el puente
  (`exportarAlcance`), sin pasar por disco.

Y un aviso: `exportarAlcance` devuelve **3 hechos** sobre las 19 preguntas
vivas. Un calendario que lea el alcance de la cuenta va a enseñar lo mismo que
enseña hoy el fichero, ni más ni menos; el hueco es el del hallazgo 2 y no se
cierra por ahí.

---

## 10. El TTFV, medido hoy

**15m51s**, exactamente el mismo número que antes de este trabajo, con las
mismas 19 preguntas, las mismas 2 órdenes y los seis pasos en 200.

Y esa es la respuesta honesta, no un empate por casualidad: el modelo mide **el
primer día**, y guardar no le ahorra ni un segundo a quien contesta por primera
vez. Lo que el guardado quita es el segundo día, que ese número no mira. Meterlo
dentro sería cambiar lo que la cifra mide para que salga mejor, que es justo la
trampa que su techo persigue.

Lo que sí se añadió a la medida es una guarda: el paso de alcance declara
`ExigeGuardado` y la puerta comprueba, **sobre el binario de verdad**, que la
entrevista ofrece formularios de guardado. Sin ella, desconectar el cableado del
almacén no movería el TTFV ni un segundo y la medida seguiría diciendo que el
recorrido está entero, que es exactamente lo que le pasó a la entrada hasta el
03-09-2026.
