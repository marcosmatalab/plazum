# Hallazgos del frente del preaviso

Lo que salio al encender la primitiva `preaviso` para el corpus, el 03-09-2026.
Columna del frente: `paquetes/psd2/`, `paquetes/psd2-es/`, `paquetes/mica/`,
`paquetes/mdr/`, `paquetes/data-act/` y este fichero. `docs/censo-relojes.md` no
lo toca nadie durante la campana, asi que las correcciones al censo van aqui y
las fusiona quien integra.

## 1. P0: la primitiva estaba cableada a medias, y el trinquete no lo veia

`nucleo/corpus/primitivas_encendidas.go` declaraba `preaviso` como
`PrimitivaApagada` con este motivo:

> cableada el 02-09-2026 (rama en VencimientosDe, validarPreaviso en el linter)
> y todavia sin un solo paquete que la declare. NO es un hueco de codigo: **un
> paquete puede usarla hoy sin tocar Go**, lo que falta es escribir los relojes.

**La frase en negrita es falsa, y se descubrio al escribir el primer paquete.**
Un `preaviso` no puede publicarse hoy:

1. El linter exige tres casos dorados a toda obligacion con reloj computable, y
   `computables()` devuelve `true` para todo `preaviso`.
2. `EjecutarDorado` (`nucleo/corpus/dorados.go`) exige que los hechos del caso
   traigan el disparador de la obligacion, y exime solo a `puntual` y a
   `continua`:

   ```go
   arrancaDeUnHecho := tmp.Primitiva != "puntual" && tmp.Primitiva != "continua"
   if _, ok := hechos[tmp.Disparador["hecho"]]; !ok && arrancaDeUnHecho {
   ```

3. `validarPreaviso` **le PROHIBE a un `preaviso` declarar disparador**
   (`ErrPreavisoConDisparador`), asi que `tmp.Disparador["hecho"]` es siempre la
   cadena vacia y `hechos[""]` nunca existe.

Resultado: **todo caso dorado de un `preaviso` muere con `falta el hecho ""`**,
sin importar lo que diga. Una guarda le pide al autor exactamente lo que la otra
le veta. Los 27 dorados de este frente lo demuestran sobre dato real.

Por que ninguna puerta lo cazo: la sonda del trinquete (`elCorpusSabeConstruir`,
en `primitivas_alcanzables_test.go`) mira **solo** el `switch` de
`VencimientosDe`. `PrimitivaApagada` significa hoy «VencimientosDe la sabe
construir», que NO es lo mismo que «un paquete la puede publicar»: entre las dos
cosas estan el linter y el ejecutor de dorados, y la sonda no los mira. Es el
mismo agujero que el trinquete existe para cerrar, un piso mas abajo, y es la
cuarta aparicion de la familia de la **afirmacion acompañada**: el cardinal (8)
tenia puerta y la frase que lo explicaba no.

### El arreglo, que este frente NO aplica porque el fichero no es de su columna

`nucleo/corpus/dorados.go` no esta en ninguna columna de `.github/frontera.sh`.
La matriz manda, asi que aqui va el parche y no el commit:

```go
	// `preaviso` NO cuelga de un disparador: su fecha la ELIGE el obligado y va
	// en `efecto`, y su AUSENCIA es un caso legitimo (pendiente de hecho) que un
	// dorado tiene que poder declarar. Ademas el linter le PROHIBE traer
	// disparador (ErrPreavisoConDisparador), asi que exigirselo aqui es pedirle
	// lo que otra guarda le veta.
	arrancaDeUnHecho := tmp.Primitiva != "puntual" && tmp.Primitiva != "continua" &&
		tmp.Primitiva != "preaviso"
```

Comprobado en esta sesion: con esa unica linea los 27 dorados pasan y
`TestLosDoradosPublicadosPasanContraElMotor` sale en verde. Sin ella, la rama
queda roja en ese test y en
`TestElDemoConElCorpusRealNoVuelcaElCatalogoEntero`.

**Y falta la puerta**, que es lo que impide la quinta aparicion: la sonda del
trinquete deberia preguntar si un paquete puede PUBLICAR la primitiva (linter mas
ejecutor de dorados), no solo si `VencimientosDe` la construye. Mientras eso no
exista, `PrimitivaApagada` seguira queriendo decir dos cosas distintas.

## 2. La medida: la familia G del censo se cae a dos de siete

`docs/censo-relojes.md`, seccion «Familia G: preaviso contractual», cuenta siete
relojes. Comprobados uno a uno contra el texto, con el criterio del propio censo
(**un plazo que corre hacia atras desde una fecha que ELIGE el obligado**):

| # | Censo | Veredicto | Por que |
|---|---|---|---|
| 1 | `psd2` art. 54.1 | **preaviso** | «deberá proponer [...] con una antelación no inferior a dos meses respecto de la fecha de aplicación propuesta». Verbo, numero y fecha elegida por el obligado. |
| 2 | `psd2` art. 55.3 | **preaviso** | «podrá rescindir [...] si avisa con una antelación mínima de dos meses». El aviso es la condicion de licitud de la resolucion. |
| 3 | `psd2` art. 55.1 | **descarte** | «El plazo de preaviso no podrá exceder de un mes» es un TECHO A UNA CLAUSULA del contrato, y el preaviso que limita lo da EL USUARIO, no el obligado. No hay reloj del proveedor. |
| 4 | `mica` art. 67.4.b | **descarte, y la cita esta mal** | El plazo de preaviso de 90 dias para cancelar el seguro esta en el **art. 67.5, letra b)**, no en el 67.4.b (que es la poliza como forma de salvaguardia). Y es una CARACTERISTICA EXIGIDA A LA POLIZA, no un plazo del obligado: nadie elige ahi una fecha. El reloj util que hay cerca es otro y es hacia adelante (sustituir la cobertura antes de la cancelacion avisada). |
| 5 | `mica` art. 65.4 | **descarte** | «podrán empezar a prestar tales servicios [...] a más tardar, a partir del decimoquinto día natural después de haber presentado la información». Es una FACULTAD y el plazo corre HACIA ADELANTE desde la presentacion, que es un hecho que ya ocurrio. El verbo exigible del art. 65 esta en el apartado 1 («presentarán») y no lleva numero. |
| 6 | `mdr` art. 75.3 | **descarte** | «El promotor podrá aplicar las modificaciones [...] como mínimo 38 días DESPUÉS de la notificación». Facultad, y cuenta hacia adelante desde la notificacion. Ademas el art. 75.4 permite prorrogarlo siete dias, que es la forma de `maximo`, no la de `preaviso`. |
| 7 | `data-act` art. 25.2.d | **descarte** | «un plazo máximo de preaviso para el inicio del proceso de cambio, que no excederá de dos meses» es un TECHO A UNA CLAUSULA, igual que el 3, y el preaviso lo da EL CLIENTE (art. 25.3). El obligado no tiene fecha que elegir. |

**Dos de siete son preavisos. Cinco no lo son**, y tres de los cinco (el 3, el 4
y el 7) son el mismo error de lectura: confundir *«la norma limita lo que puede
pactarse sobre un preaviso»* con *«la norma le impone un preaviso al obligado»*.
Es el reverso exacto de la pregunta fija de la pasada 2: **de que verbo cuelga
este numero**. En los tres, el verbo es «incluir en el contrato», que es un deber
permanente sin fecha, no un reloj.

Los dos que si lo son estan escritos, pero **en el instrumento que vincula, no en
la directiva**: `psd2-es` arts. 33.1 y 32.4, que son la transposicion espanola de
los arts. 54.1 y 55.3 con las mismas cifras. No se escriben en `paquetes/psd2`
por dos razones, y la segunda es la que decide:

- el `LEEME.md` de ese paquete dice, desde antes de esta campana, que se queda
  como esqueleto y que lo exigible en Espana esta en `psd2-es`;
- y sobre todo, la unica regla de ambito disponible seria `designado(E,
  "proveedor_de_servicios_de_pago")`, **la misma que ya usa `psd2-es`**, asi que
  un proveedor espanol veria el mismo reloj DOS VECES, con dos citas distintas.
  Dos filas para la misma obligacion es peor que una: el operador no sabe cual es
  la suya. Es la decision de identidad que el censo deja abierta en su seccion 7
  (dos paquetes, uno con dos vigencias territoriales, o una capa) y no se resuelve
  desde este frente.

## 3. Lo que el censo NO conto: siete preavisos nuevos

El barrido de la columna (`antelación`, `preaviso`, `antes de la fecha`, `días
antes`, `previamente a`, `con carácter previo`) sobre las cuatro normas dio siete
relojes de esta forma que la familia G no tiene. Los siete estan escritos, con
sus 27 casos dorados:

| Paquete | Articulo | Antelacion | Fecha que elige el obligado |
|---|---|---|---|
| `mica` | 8.5 | 20 dias habiles | fecha de publicacion del libro blanco |
| `mica` | 25.1, parrafo final | 30 dias habiles | fecha de efecto de los cambios del modelo de negocio |
| `mica` | 48.6 | 40 dias habiles | fecha prevista de oferta publica o admision a negociacion |
| `mica` | 69 | **indeterminada** (D-17) | fecha en que los nuevos miembros empiezan a desempenar sus actividades |
| `mdr` | 16.4 | 28 dias naturales | fecha prevista de comercializacion del producto reetiquetado |
| `psd2-es` | 33.1 | 2 meses | fecha de entrada en vigor de la modificacion propuesta |
| `psd2-es` | 32.4 | 2 meses | fecha de efecto de la resolucion del contrato marco |

Tres de ellos (`mica` 8.5, 25.1 y 48.6) cuentan en **dias habiles hacia atras**,
que es la rama de `ventana.Restar` que ningun dorado del corpus recorria.

## 4. Descartes propios, para que no haya que volver a mirarlos

Salieron del barrido y NO se escriben. Con su articulo, que es lo que hace que el
descarte sea recontable:

- **`mica` art. 9.1**: «con una antelación razonable y, a más tardar, en la fecha
  de inicio de la oferta pública». La antelacion es indeterminada pero el tope
  duro es la propia fecha de efecto, o sea antelacion cero, y eso no es ni
  `indeterminado` ni una duracion que hoy se pueda escribir sin forzar un `P0D`.
- **`psd2-es` art. 40.3** (bloqueo del instrumento de pago, «con carácter previo
  al bloqueo y, de no resultar posible, inmediatamente después»): la fecha de
  efecto no se elige por adelantado en el caso tipico, y por eso la propia norma
  trae la valvula ex post. Falla el criterio en la direccion que importa.
- **`psd2-es` art. 22.1** («deberá comunicarlo previamente al Banco de España»
  antes de prestar servicios en otro Estado miembro): tiene forma de preaviso
  indeterminado, pero el censo ya lo cuenta como EVENTO (art. 28.1 de la
  directiva). Reclasificarlo desde este frente seria mover un reloj de familia
  para engordar un recuento. Queda anotado para que lo decida el censo.
- **`psd2` art. 51.1** («con suficiente antelación a la fecha en que el usuario
  quede vinculado»): preaviso indeterminado real, pero vive solo en la directiva
  y le aplica el problema de identidad del punto 2. El RDL 19/2018 no lo
  transpone con numero: su art. 29.3 lo difiere a orden ministerial.
- **`psd2` art. 76.3.b** y **`psd2-es` art. 48.b** (informacion de la operacion
  futura con cuatro semanas de antelacion): es una CONDICION para poder pactar la
  exclusion del derecho de devolucion, no una obligacion. Nadie esta obligado a
  darla.
- **`mdr` art. 74.1** (30 dias antes del comienzo de una investigacion de
  seguimiento clinico poscomercializacion) y **`mdr` art. 87.8** (informar de la
  accion correctiva de seguridad antes de llevarla a cabo): **son preavisos de
  verdad y aun asi no se escriben**, porque su fecha de aplicacion no se puede
  escribir. El art. 123.3, letra d), del Reglamento (UE) 2017/745 difiere los
  «artículos 70 a 77» y los «artículos 87 y 88» a seis meses despues del anuncio
  de la Comision sobre Eudamed (art. 34.3), y ese anuncio no se ha verificado en
  esta sesion. Poner `2021-05-26` seria inventar una fecha.
- **`data-act`**: cero preavisos. El barrido completo del Reglamento (UE)
  2023/2854 solo devuelve topes a clausulas contractuales (arts. 13, 23 y 25).
  El paquete se queda como estaba.

## 5. Lo que se encontro de paso, y no es del preaviso

Dos vigencias del corpus que el barrido de las tres fechas destapo. Las dos estan
en esta columna y las dos se corrigen aparte:

- **`mica` arts. 22.1, 30.1 y 35.1**: llevan `2024-12-30` como vigencia heredada.
  Los tres articulos estan en el **titulo III** del Reglamento (UE) 2023/1114
  (comprobado sobre el XHTML de Cellar el 03-09-2026, porque la extraccion por
  articulos no trae los rotulos de titulo), y el art. 149.3 adelanta los titulos
  III y IV al **30-06-2024**. Seis meses. Direccion del error: el corpus dice que
  la obligacion empezo MAS TARDE de lo que empezo.
- **`psd2-es` arts. 66.2 y 67.1**: llevan `2018-11-25`, que es la entrada en vigor
  del real decreto-ley. Los dos estan en el **titulo III** (comprobado sobre el
  consolidado del BOE el 03-09-2026), y la disposicion final decimotercera,
  apartado 2, letra a), difiere los titulos II y III «a los tres meses de su
  publicación en el Boletín Oficial del Estado». Publicado el 2018-11-24, o sea
  **2019-02-24**. Direccion del error: el corpus dice que la obligacion empezo
  TRES MESES ANTES de lo que empezo, que es la direccion que puede acusar en
  falso.

## 6. Huecos declarados, con su cardinal

- **2 papeles de `mica` sin modelar**: el art. 8.5 obliga tambien a «las personas
  que soliciten la admisión a negociación» y a «los operadores de plataformas de
  negociación», y el paquete solo tiene `papel_mica(E, "oferente")`. La regla se
  queda corta, que en una `notificatoria` es la direccion que no provoca una
  actuacion indebida ante la autoridad.
- **0 de 7 preavisos declaran `traslado: siguiente_habil`**, asi que la rama
  hacia atras de `ventana.Restar` (la que ADELANTA al habil anterior en vez de
  retrasar, porque retrasar acortaria la antelacion por debajo del minimo) sigue
  sin recorrerse desde el corpus. No se declara en ninguno porque ninguna de las
  cuatro normas dice que el aviso deba caer en dia habil, e inventarlo seria
  escribir un computo que la norma no da.
- **1 lectura alternativa que no se puede escribir**: «al menos 20 dias habiles
  antes» admite contar el vigesimo dia habil anterior (lo que hace el motor) o el
  vigesimo primero. Los `hitos` de un `plazo` tienen `alternativas` para eso; un
  `preaviso` no las tiene, porque no pasa por `hitos`. La lectura elegida se dice
  en el `cita_del_esperado` de cada dorado.
- **1 campo que no llega**: `ventana.Preaviso` tiene `Nota` y la rama de
  `VencimientosDe` no la rellena, porque `Temporalidad` no tiene `nota` fuera de
  `hitos`. Un preaviso con antelacion indeterminada (art. 69 de MiCA) no puede
  explicar su hueco donde el resto del corpus lo explica; hoy lo hace en el
  `titulo` y en la `cita`, que se leen en otra pantalla.
- **1 rama del linter que no distingue**: `computables()` devuelve `true` para
  todo `preaviso`, tambien para el de antelacion `indeterminado`, que no produce
  ninguna fecha. Le exige los tres dorados igual, y la valvula que el resto del
  corpus usa para eso (declarar `hitos` con su `nota`) no existe en esta
  primitiva. Aqui se ha podido cumplir escribiendo tres casos que afirman
  ESTADOS y no fechas, pero es una coincidencia afortunada, no un diseno.
- **1 dia de incertidumbre en la vigencia de `psd2-es`**: «serán de aplicación a
  los tres meses de su publicación» se ha leido como el dia que completa los tres
  meses de fecha a fecha (art. 5.1 del Codigo Civil), o sea el 2019-02-24 desde
  una publicacion del 2018-11-24. La lectura alternativa (que los tres meses se
  agoten al final de ese dia, y la aplicacion empiece el 2019-02-25) no se puede
  descartar con el texto delante, y el BOE no publica esa fecha como dato: su
  `fecha_vigencia` es la general, 2018-11-25. `Vigencia` tiene `alternativas`
  para escribir esa discrepancia; no se usa aqui para no meter una lectura
  divergente de un dia en el mismo commit que corrige tres meses, pero queda
  anotado con su cardinal: 2 obligaciones afectadas.
- **1 designacion que no se pudo escribir**: el art. 32.4 del RDL 19/2018 solo
  obliga si el contrato marco es indefinido Y la facultad esta pactada. Escribir
  eso como `designado(E, "contrato_marco_indefinido_con_resolucion_unilateral_pactada")`
  pone rojo `TestTodoPerfilContestaATodaDesignacionDelCorpus`, porque los tres
  perfiles de `perfiles/` tendrian que contestarla y ese directorio esta fuera de
  esta columna. La regla se dejo en `designado(E,
  "proveedor_de_servicios_de_pago")` y la discriminacion ocurre en el hecho: sin
  fecha de efecto no hay fila con fecha.
