# Hallazgos del frente B: `cra` y `nis2-ue` (03-09-2026)

Este documento es del frente B de la campaña de cuatro frentes. `docs/censo-relojes.md`
no lo toca nadie durante la campaña, así que las correcciones al censo se escriben
aquí y las aplica quien integra.

## 0. Lo que quien integra tiene que mover, con las cifras exactas

Tres tests de raíz están **en rojo en esta rama a propósito**, y los tres apuntan a
ficheros que la matriz de fronteras (`.github/frontera.sh`) no le da a ningún frente:

| fichero | dice | tiene que decir |
|---|---|---|
| `README.md` | 218 hitos | **234** |
| `README.md` | 626 casos dorados | **654** |
| `README.md`, bloque cobertura-v1 | 48,8 % | **58,1 %** (100 relojes de la norma sobre 172 censados) |
| `paquetes/CORPUS.md` | 218 hitos | **234** |
| `paquetes/CORPUS.md` | 626 dorados | **654** |

Los tests son `TestLasCuentasPublicadasSalenDelCorpusYNoDeLaMemoria`,
`TestLosNumerosDelCorpusEnElREADMESalenDelArbol` y
`TestElPorcentajeDeLaV1LoComputaUnTestYNoUnaPersona`. **Las cifras de arriba son las
de esta rama sola**: con los frentes A y C escribiendo corpus a la vez, cualquier
número que escribiera este frente sería falso antes del merge. Por eso se dejan aquí
y no en el README, que es exactamente lo que la matriz manda.

**Y un apunte sobre la propia matriz**: su comentario nombra `ETAPAS.md` y `README.md`
como del integrador, y se deja `paquetes/CORPUS.md`, que lleva las mismas dos cifras y
la misma puerta. Quien integre tiene que mover **los dos ficheros**, no uno.

## 1. Lo escrito

`cra`: de **16 a 24** obligaciones, de **21 a 29** hitos, de **50 a 64** dorados.

| artículo | qué es | número |
|---|---|---|
| 13.18 | retención de la información e instrucciones del anexo II | diez años **o** el período de soporte |
| 18.3.a | la misma retención, del **representante autorizado** | diez años **o** el período de soporte |
| 13.19 párr. 2 | avisar al usuario de que el producto llegó al fin del soporte | sin cifra, disparador calculable |
| 14.6 | informe provisional a instancia del CSIRT coordinador | sin cifra, **vigente desde el 11-09-2026** |
| 19.5 párr. 2, 1.ª frase | el importador avisa al fabricante de la vulnerabilidad | sin cifra |
| 19.5 párr. 2, 2.ª frase | el importador avisa a las autoridades del riesgo significativo | sin cifra |
| 20.4 párr. 2, 1.ª frase | el distribuidor avisa al fabricante de la vulnerabilidad | sin cifra |
| 20.4 párr. 2, 2.ª frase | el distribuidor avisa a las autoridades del riesgo significativo | sin cifra |

`nis2-ue`: de **4 a 12** obligaciones, de **8 a 16** hitos, de **10 a 24** dorados.

| artículo | qué es | número |
|---|---|---|
| 3.4 párr. 2 | cambios en la información de la lista nacional del art. 3.3 | **dos semanas** |
| 23.4.e | informe final del incidente que seguía en curso | **un mes** desde la gestión |
| 28.5 | responder a la solicitud lícita de acceso a datos de dominios | **72 horas** |
| 23.4.c | informe intermedio a instancias del CSIRT | sin cifra |
| 23.1 párr. 1, 2.ª frase | avisar del incidente a los destinatarios del servicio | sin cifra |
| 28.4 | publicar los datos no personales tras el alta de un dominio | sin cifra |
| 29.4 | notificar la incorporación a un mecanismo de intercambio | sin cifra |
| 29.4 | notificar la retirada, cuando la retirada surta efecto | sin cifra |

**Papeles nuevos, y existen para que el reloj NO se encienda a quien no le toca**:
`papel_cra(E, "representante_autorizado")` y
`papel_nis2_dominios(E, "registro_o_servicio_de_registro")`.

## 2. P0 para quien integra: `preaviso` NO es usable desde un paquete

`nucleo/corpus/primitivas_encendidas.go` declara `preaviso` como `PrimitivaApagada`
con este motivo: *«cableada el 02-09-2026 [...] NO es un hueco de código: un paquete
puede usarla hoy sin tocar Go, lo que falta es escribir los relojes»*.

**Eso es falso, medido hoy escribiendo el reloj.** El art. 13.23 del CRA es un
preaviso de manual (*«informará del próximo cese de las actividades, ANTES DE QUE
DICHO CESE SURTA EFECTO»*: la fecha de efecto la elige el obligado y lo que se
calcula es hasta cuándo puede callar). Se escribió, y hay una contradicción entre dos
puertas que lo impide:

- `nucleo/corpus/preaviso.go`, `validarPreaviso`, **prohíbe** que un `preaviso`
  declare `disparador`, con `ErrPreavisoConDisparador` y un razonamiento correcto:
  un preaviso no cuenta desde un hecho que le ocurre al obligado.
- `nucleo/corpus/dorados.go` **exige** que todo dorado traiga el hecho del
  `disparador`, salvo para `puntual` y `continua`:

  ```go
  arrancaDeUnHecho := tmp.Primitiva != "puntual" && tmp.Primitiva != "continua"
  if _, ok := hechos[tmp.Disparador["hecho"]]; !ok && arrancaDeUnHecho {
  ```

Como un `preaviso` no tiene disparador, la comprobación busca el hecho `""`, no lo
encuentra, y **los tres dorados fallan siempre**. El error que sale es
`falta el hecho "", que es el disparador de la obligacion`, o sea un mensaje que ni
siquiera nombra la primitiva. Y `computables()` tampoco conoce `preaviso`, así que le
exige los tres dorados que no pueden existir. Resultado: la obligación no se puede
publicar de ninguna manera.

**El arreglo es una línea**, y es del integrador porque toca `nucleo/corpus/`:

```go
arrancaDeUnHecho := tmp.Primitiva != "puntual" && tmp.Primitiva != "continua" &&
    tmp.Primitiva != "preaviso"
```

Detrás de esa línea, la ficha del art. 13.23 y sus tres dorados están escritos y se
pegan en diez minutos: los estados que da el motor son `pendiente de hecho` sin la
fecha de efecto y `sin plazo legal` con ella (antelación `indeterminado`, porque el
apartado exige avisar antes y no dice cuánto antes).

**Y el cardinal de `preaviso` sube de 8 a 9**, y el noveno está DENTRO de los 12
marcos de la v1: el art. 13.23 del CRA.

## 3. P1: la vigencia de una obligación no la vigila ninguna puerta

Mutación M5 de la pasada 2, aplicada sobre `df098f4` con el árbol limpio: se movió
`cra.art14_6.informe_provisional_a_instancia_del_csirt` de `2026-09-11` a
`2027-12-11`, quince meses, sobre la fecha más cercana de todo el corpus.

`go test ./...` devuelve **exactamente los mismos tres fallos que el árbol sin mutar**
(los tres del README y `paquetes/CORPUS.md`). Ninguna puerta nueva se pone roja.

Y el efecto sí existe: con la mutación puesta,
`plazum calendario --pais=ES --sector=fabricante-software --ahora=2026-09-15` deja de
enseñar la fila del art. 14.6 (`grep -c "art. 14.6"` pasa de 1 a 0). No sale un error:
sale **silencio**, que es lo que un CISO lee como «no me toca».

Por qué no lo caza nada: un caso dorado ejecuta el reloj **directamente**, sin pasar
por la vigencia, y `vigencias_test.go` solo comprueba que la fecha no sea la de
publicación de la norma. Falta la tercera afirmación: que la vigencia declarada esté
respaldada por un apartado citado. Es de la familia de la *afirmación acompañada*.

## 4. P2: un dorado no puede afirmar la rama «pendiente de hecho» de un reloj por evento

Consecuencia del mismo `if` de `dorados.go`. Todo reloj por evento tiene dos ramas
—con el hecho y sin él— y **la segunda no tiene caso dorado en ningún sitio del
corpus**, porque un dorado sin el hecho disparador se rechaza antes de ejecutarse. Se
intentó escribir seis de esos casos en este frente y se retiraron los seis.

No es inofensivo: la rama sin hecho es la que produce la frase *«falta un dato que
pones tú»* del calendario, que es la que separa «no me consta» de «no me toca». Es un
descargo sin control positivo, que es el patrón de M47.

## 5. Descartes: puntos que se miraron y NO son relojes

Se cuentan porque descartar es un resultado.

**`cra`, 9 descartes**:

1. **art. 22.1 y 22.2** (confirmado, ya estaba en el censo): regla de atribución, no
   de plazo. Dice a quién se considera fabricante, no cuándo hay que hacer algo.
2. **art. 21**: la misma forma. Dice cuándo un importador o un distribuidor pasa a
   estar sujeto a las obligaciones del fabricante.
3. **art. 13.10**: *«podrá garantizar el cumplimiento [...] únicamente para la versión
   más reciente»*. Es una facultad, no un deber.
4. **art. 13.11**: *«podrán mantener archivos públicos de programas informáticos»*.
   Facultad.
5. **art. 13.24 y 13.25**: de la Comisión y del ADCO.
6. **art. 14.5**: define cuándo un incidente es grave. Es un umbral, no un verbo.
7. **art. 14.7**: regla de competencia, dice a QUÉ CSIRT se notifica. No mueve ninguna
   fecha.
8. **art. 14.9 y 14.10**: de la Comisión.
9. **art. 18.1 y 18.2**: facultad de designar representante y delimitación del
   mandato.

Un décimo candidato **no se descarta y se dice por qué no**: los arts. 19.2 y 20.2
(comprobaciones previas a introducir o a comercializar) parecen condición de puesta en
el mercado y no plazo, pero son deberes exigibles al obligado, así que van al recuento
de candidatos a `continua` del §6 con esa duda escrita, no a la papelera. Descartar es
un resultado; descartar con duda es un olvido.

**`nis2-ue`, 8 descartes**:

1. **art. 20.2** (confirmado): el adverbio *periódicamente* cuelga de *alentarán* a
   las entidades sobre la formación de los **empleados**, no del deber del órgano de
   dirección. Ponerle cadencia al órgano sería colgar el número del verbo equivocado.
2. **art. 23.3**: define cuándo un incidente es significativo.
3. **art. 23.5 a 23.11**: del CSIRT, de la autoridad competente, del punto de contacto
   único, de ENISA y de la Comisión.
4. **art. 3.3, 3.5 y 3.6**: de los Estados miembros y de las autoridades.
5. **art. 27.1, 27.4 y 27.5**: de ENISA y del punto de contacto único.
6. **art. 29.1, 29.2, 29.3 y 29.5**: intercambio **voluntario** de información y
   deberes de los Estados miembros. Solo el 29.4 obliga a la entidad.
7. **art. 30**: notificación **voluntaria**. *«Podrán notificar»* no es un reloj.
8. **art. 28.6**: prohíbe duplicar la recopilación y manda cooperar. Sin momento.

## 6. Identificados y NO escritos, con su cardinal

**`cra`, 36 puntos.** Doce son relojes por evento sin cifra y veinticuatro son deberes
permanentes candidatos a `continua` (D-17).

*Relojes por evento sin cifra (12)*: art. 13.6 (avisar al fabricante o mantenedor del
componente vulnerable), 13.22 (facilitar información a requerimiento motivado de la
autoridad), **13.23 (el preaviso, bloqueado por el motor, ver §2)**, 18.3 párr. 1
(copia del mandato a petición), 18.3.b (información a requerimiento), 19.3 párr. 1
segunda frase (riesgo significativo antes de introducir), 19.3 párr. 2 (riesgo por
factores **no técnicos**, que es un disparador que no aparece en ningún otro sitio del
CRA), 19.7, 19.8 (cese de actividades del fabricante, visto por el importador), 20.3,
20.5 y 20.6 (el mismo cese, visto por el distribuidor).

*Deberes permanentes candidatos a `continua` (24)*: arts. 13.1, 13.2, 13.3, 13.4,
13.5, 13.7, 13.8 párr. 1, 13.8 párr. 3 (el suelo de cinco años del período de
soporte), 13.8 párr. 5, 13.8 párr. 6, 13.12, 13.14, 13.15, 13.16, 13.17, 13.18 (la
primera parte, acompañar el producto de la información del anexo II), 13.19 párr. 1
(la fecha de fin de soporte en el momento de la compra), 13.20, 18.3.c, 19.1, 19.4,
20.1, más 19.2 y 20.2 si la pasada decide que son deber permanente y no condición de
puesta en el mercado. **Esa frontera hay que decidirla, y son 4 de los 24** (19.1,
19.2, 20.1 y 20.2).

Alcance de este recuento, dicho para que se pueda rehacer: se leyeron **enteros** los
artículos 13, 14, 18, 19, 20 y 71 del CRA. Los arts. 1 a 12, 15 a 17, 23 a 70 y los
anexos I a VIII **no se han barrido con esta lupa**, salvo los puntos del anexo VIII y
del anexo I parte II que el paquete ya traía.

**`nis2-ue`, 10 puntos.**

1. **art. 27.2** (fecha fija 17-01-2025, ya pasada). **No se escribe, y el motivo
   sigue en pie**: en España no hay transposición, así que enseñarla vencida le diría a
   un proveedor de nube español que incumplió un registro que nadie le ha exigido. Entra
   con la norma de transposición y con la fecha de esa norma.
2. **arts. 21.1, 21.2 y 21.3**: las medidas de gestión de riesgos. Son deberes
   permanentes sin cifra (3 puntos) y el paquete solo tiene su remedio, el art. 21.4.
3. **art. 20.1**: la aprobación y supervisión por el órgano de dirección. `continua`.
4. **art. 28.1, 28.2 y 28.3**: recopilar y mantener la base de datos de dominios y
   hacer públicas las políticas de verificación. `continua`, tres puntos.
5. **arts. 32.4 y 33.4**: plazos que **fija la autoridad** en sus instrucciones
   vinculantes (*«plazos para la ejecución de esas medidas»*, *«en un plazo
   concreto»*, *«en un plazo razonable»*). Son la forma del art. 57.2 del CRA, que el
   corpus ya sabe escribir: número cerrado cuyo valor plazum no conoce. **Se escriben
   el día que se decida si el papel es «entidad esencial» o «entidad importante», que
   son dos regímenes distintos (arts. 32 y 33) y el paquete hoy no los distingue.**

**Y un hueco de alcance declarado, cardinal 1**: el art. 3.4 alcanza a *«las entidades
esenciales e importantes ASÍ COMO las entidades que prestan servicios de registro de
nombres de dominio»*, y las segundas pueden no ser ni esenciales ni importantes. La
regla escrita solo enciende el reloj para las primeras. Se escribió estrecho a
propósito: en una `notificatoria` pasarse de ancho provoca una actuación ante el
supervisor que no se deshace.

## 7. Correcciones al censo, para quien lo mantenga

**Ninguna se aplica aquí**: `docs/censo-relojes.md` y `paquetes/marcos-v1.json` no son
de este frente.

- **fila `cra`**: dice **22** puntos únicos y el paquete tiene ya **24 obligaciones
  con reloj**, todas con su cita. Los ocho que el censo no tenía son los de la tabla
  del §1. El más llamativo es el **art. 13.18**: es la misma retención de diez años o
  período de soporte que los arts. 13.9 y 13.13, vive **en el mismo artículo que
  ellos**, y la rejilla de retención de la sección 2 ter no lo recogió. Con los 36 del
  §6 la fila real de este marco está por encima de **60**.
- **fila `nis2-ue`**: dice **9** puntos únicos y el paquete tiene **12**. Faltaban el
  art. 3.4 párr. 2, el 23.4.e, el 28.4, el 28.5 y el 29.4 (que son dos). Con los 10
  del §6 la fila real ronda los **22**.
- **el método, que es lo que se repite**: los ocho del CRA y los seis de NIS2 no eran
  filas pendientes, eran puntos que el censo **nunca contó**. El censo lee citas y el
  paquete se escribe leyendo el artículo, y la nota que ya está en la cabecera del
  censo (*«cada marco que se termina puede corregir su propia fila»*) se ha cobrado
  dos filas más el mismo día.

## 8. Las mutaciones de la pasada 2, con su rojo

Todas sobre `df098f4` con `git status` limpio, aplicadas y restauradas con **dos
comandos separados** y con copia (`cp`), nunca con `git checkout`. Compilación
comprobada con `go build ./...` en las cinco.

| # | qué se mutó | qué se puso rojo |
|---|---|---|
| M1 | `P14D` → `P3M` en el art. 3.4 (el número del art. 27.3, que es la confusión que la ficha avisa) | los **tres** dorados del art. 3.4: *el motor dice 2026-09-03T23:59:59Z y el texto dice 2026-06-17T23:59:59Z* |
| M2 | `traslado: siguiente_habil` → `ninguno` en el art. 3.4 | **solo** el dorado del sábado: *el motor dice 2026-06-20T23:59:59Z y el texto dice 2026-06-22T23:59:59Z*. Los otros dos siguen verdes, que es lo que demuestra que aíslan el traslado |
| M3 | `ampliacion_exigible: true` → `false` en el art. 13.18 | el dorado del período de soporte sin declarar: *el hito «fin_de_la_retencion» sale como «determinado» y el texto dice «pendiente de hecho»* |
| M4 | hito intruso con `limite: PT24H` en el art. 23.4.c | el **linter**, por el mínimo de tres dorados. Cazado, pero por el motivo equivocado, así que se repitió |
| M4bis | el mismo hito intruso con `limite: indeterminado`, para que el linter no lo vea | la dirección de exhaustividad: *SOBRA el hito «hito_intruso» (sin plazo legal), que el texto no da* |
| M5 | vigencia del art. 14.6 de `2026-09-11` a `2027-12-11` | **NADA.** Es el §3 |

## 9. Errores propios de este frente

1. **La lectura de más del art. 19.5 y del 20.4.** La segunda frase se escribió con
   disparador **doble** (no conformidad **y** riesgo significativo). Su condición
   literal es **una**: el riesgo significativo. Lo de la no conformidad describe el
   **contenido** del aviso. Escrito doble, el reloj se quedaba apagado justo en el
   momento en que la autoridad quiere enterarse. Corregido antes de commitear, y la
   corrección viaja dentro de la cita para que no vuelva.
2. **Se dieron por buenos los cardinales del encargo** («faltan 6 y 5») y no lo eran
   en ninguno de los dos sentidos: comparaban **obligaciones escritas** contra
   **puntos censados**, que son unidades distintas, y además el censo se quedaba corto.
   Salieron 8 y 8.
3. **La primera mutación del hito intruso (M4) no midió lo que quería.** Lo cazó el
   linter por el conteo de dorados, no la comprobación de exhaustividad, así que no
   demostraba nada de lo que se quería demostrar. Se repitió con el límite
   `indeterminado` para esquivar al linter y dejar sola a la puerta que importaba.
4. **Se escribieron seis casos dorados imposibles** (la rama «sin el hecho») antes de
   descubrir el `if` de `dorados.go`. Se retiraron los seis y el hallazgo quedó en el
   §4, que vale más que los dorados.
5. **El `preaviso` del art. 13.23 se escribió entero y hubo que sacarlo.** No fue
   trabajo perdido: la ficha está redactada y el §2 dice la línea exacta que la
   habilita.
