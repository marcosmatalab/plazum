# Hallazgos del frente A: los relojes del AI Act

Campaña del 03-09-2026. Columna del frente: `paquetes/ai-act/` y este fichero.
Todo lo que aquí se pide a otra columna se pide, no se toca.

El paquete pasa de **12 relojes a 20** y de 39 a 58 casos dorados. El que da
nombre a la tanda es el artículo 60.4, letra f), que estrena la primitiva
`preaviso`: el primer plazo del corpus que corre **hacia atrás** desde una fecha
que elige el obligado.

## H1. `preaviso` NO se puede usar sin tocar Go, y el censo de primitivas afirma lo contrario

`nucleo/corpus/primitivas_encendidas.go` declara la primitiva `apagada` con este
motivo escrito: *«NO es un hueco de código: un paquete puede usarla hoy sin tocar
Go, lo que falta es escribir los relojes»*. **Es falso**, y se descubre al ser el
primero en escribir uno.

`EjecutarDorado` exige que todo dorado declare el disparador de su obligación:

```go
// nucleo/corpus/dorados.go
arrancaDeUnHecho := tmp.Primitiva != "puntual" && tmp.Primitiva != "continua"
if _, ok := hechos[tmp.Disparador["hecho"]]; !ok && arrancaDeUnHecho {
```

Un `preaviso` **no tiene disparador y no puede tenerlo**: `validarPreaviso` lo
rechaza expresamente, porque su fecha no le ocurre al obligado, la elige él, y va
en `efecto`. Así que `tmp.Disparador["hecho"]` vale `""`, la comprobación no
encuentra el hecho vacío y **todos** los dorados de un preaviso salen rojos con
este mensaje, que además nombra la nada:

```
dorado "sin prorroga decidida no hay nada que preavisar": falta el hecho "",
que es el disparador de la obligacion
```

Es la pregunta fija de la pasada 1 («¿puede un paquete usar esta primitiva sin
tocar código?») contestada **no** por segunda vez en dos días, después de
`maximo`, y con la agravante de que aquí la respuesta estaba escrita al revés en
el propio censo. La causa es la misma de siempre: cada mitad pasaba su puerta.
La primitiva tenía sus dorados verdes en `nucleo/ventana`, el linter la validaba,
la rama de `VencimientosDe` existía, y nadie había recorrido la junta con un
paquete de verdad.

**Lo que pide el frente A al integrador, y son dos cosas, no una.**

1. En `nucleo/corpus/dorados.go`, excluir `preaviso` de `arrancaDeUnHecho`:

   ```go
   arrancaDeUnHecho := tmp.Primitiva != "puntual" && tmp.Primitiva != "continua" &&
       tmp.Primitiva != "preaviso"
   ```

   Con el comentario que falta: una `puntual` y una `continua` no cuelgan de
   ningún hecho, y un `preaviso` cuelga de uno que **no es el disparador**; su
   ausencia es además un caso legítimo que hay que poder escribir, porque
   «todavía no hay nada que preavisar» es una respuesta.

2. En `nucleo/corpus/primitivas_encendidas.go`, la entrada entera pasa a:

   ```go
   "preaviso":  {Estado: PrimitivaEnUso},
   ```

   Sin `Motivo`: `TestTodaPrimitivaDelMotorOSeUsaOSeExplica` se pone rojo si una
   primitiva en uso arrastra el motivo de cuando no lo estaba.

**Medido, no supuesto.** Con esos dos cambios aplicados en el árbol de trabajo,
`go test ./...` deja de acusar a este paquete: los 58 dorados de `ai-act` pasan
contra el motor, incluidos los tres del preaviso, y
`TestTodaPrimitivaDelMotorOSeUsaOSeExplica` se pone verde. Los cambios se
retiraron después: **este frente no los entrega**, están fuera de su columna.

## H2. Encender `preaviso` deja el censo de primitivas sin ninguna apagada, y hay una puerta que lo dice

Con las dos líneas de H1 aplicadas sale este rojo, que **no es un fallo del
cambio sino la puerta avisando de que cambia de forma**:

```
--- FAIL: TestNingunMotivoDeUnaPrimitivaEscribeSuCardinal
    primitivas_alcanzables_test.go:425: ninguna primitiva esta apagada o sin
    cablear, asi que este recorrido no ha mirado ni un motivo: es un verde
    vacio. O se han encendido todas, y entonces esta puerta cambia de forma, o
    el censo no se esta leyendo
```

Las seis primitivas del motor quedan **todas en uso** y ese recorrido se queda
sin nada que mirar. La salida no es bajar el `t.Fatal`: es darle un control
positivo con dato sintético, una `DeclaracionDePrimitiva` construida en el propio
test con un numeral en la prosa, para que la puerta siga demostrando que caza el
recuento a mano cuando el corpus ya no le da ningún caso. Es la misma doctrina de
M47: **una rama que ninguna entrada recorre es una rama que no existe.**

Decidirlo es del integrador; queda dicho porque su commit lo va a encontrar.

## H3. El guardián del objeto dentro del hecho no mira el campo del preaviso

`TestNingunPaqueteNombraUnObjetoDentroDeUnHecho` (D-18) recorre
`Temporalidad.Disparador` y `Temporalidad.ReabrePor` buscando instancias metidas
dentro del nombre de un hecho. **No recorre `Temporalidad.Efecto`**, que es
exactamente un nombre de hecho y el único que tiene un `preaviso`.

Hoy no hay ningún fallo que cazar (el único `efecto` del corpus es
`inicio_de_la_prorroga_de_las_pruebas_en_condiciones_reales`, sin barra ni
almohadilla), y por eso esto es **P2**: es un agujero que se abre justo ahora,
cuando la primitiva entra en uso, y la tentación que D-18 describe
(`prorroga/2027-014`) cabe ahí igual que en un disparador. Tres líneas en
`nucleo/corpus/por_objeto_test.go`, dentro del mismo bucle:

```go
if sospechoso(o.Temporalidad.Efecto) { ... }
```

## H4. Dos ficheros fuera de la columna quedan rojos, con sus números exactos

Escribir corpus mueve cuentas que se publican fuera de `paquetes/ai-act/`. Los
cuatro números, medidos por las propias puertas después de esta tanda:

| dónde | qué | dice | es |
|---|---|---|---|
| `README.md` | hitos | 218 | **226** |
| `README.md` | casos dorados | 626 | **645** |
| `README.md` | bloque `cobertura-v1` | 48,8 % | **53,5 %** (92 relojes de la norma sobre 172 censados) |
| `paquetes/CORPUS.md` | hitos y dorados | 218 y 626 | **226 y 645** |

**No se tocan a propósito.** Son ficheros de una sola línea compartida por los
cuatro frentes de esta campaña, y cuatro ramas editando la misma frase es un
conflicto garantizado sobre un número que además va a volver a moverse cuando se
fusione la siguiente. Los números de arriba son los de **esta rama sola**: si el
integrador fusiona antes otro frente, hay que volver a medir, y la forma de
medirlo es correr la puerta, no sumar.

## H5. La regla del artículo 22.3, letra b), es más ancha que su artículo

`art22_3b_es_del_representante_autorizado` cuelga solo de
`papel_ia(S, "representante_autorizado")`. El artículo 22 es el de los
representantes de proveedores de sistemas de **alto riesgo** (su apartado 1 lo
dice), y el representante de un proveedor de **modelo de uso general** tiene su
propia retención en el artículo 54.3, letra b), sobre otra documentación (el
anexo XI) y hacia otro destinatario (la Oficina de IA). Con la regla como está,
un representante de modelo de uso general ve las dos.

**Este frente no la ha cambiado**, y el motivo es el mismo que el del frente B
con las seis reglas del RGPD: estrechar una regla existente deja invisible una
obligación para todo alcance que no afirme el hecho nuevo, y eso se decide de una
vez, no en cuatro ramas a la vez. Lo que sí se ha hecho es **no repetir el
error**: la regla del artículo 22.4, que es nueva, exige además el alto riesgo, y
la asimetría dentro del mismo artículo está explicada en su cita. Se eligió así
porque las dos obligaciones no cuestan lo mismo si se equivocan: la del apartado
3 es una retención de más, la del apartado 4 es **comparecer ante la autoridad
equivocada**, que es de los pocos errores de cumplimiento que no se deshacen.

**P1 para el integrador**: corregir `art22_3b` a las dos cláusulas (`alto_anexo_iii`
y `alto_anexo_i`), en el mismo lote que las seis del RGPD.

## H6. La corrección del 02-09-2026 arregló el dato y dejó la prosa mintiendo

`aiact.art111_4` tenía la fecha de **publicación** del ómnibus (24-07-2026) donde
va la de **entrada en vigor** (27-07-2026), y se corrigió el 02-09-2026 en el
JSON. `LEEME.md` y `COMPUTO.md`, en el mismo directorio, **siguieron diciendo
2026-07-24 durante un día**, y el LEEME además lo explicaba con un argumento
entero («puede ser hasta tres días anterior a la entrada en vigor real, que es el
lado inofensivo») que describía un paquete que ya no existía.

Es la quinta aparición de **la prosa es la mitad que caduca** y la primera dentro
de un directorio de paquete. Corregido en los dos, diciendo que estaba mal en vez
de sustituirlo en silencio. Y no había puerta: **ninguna de las cuentas de
`LEEME.md` ni de `COMPUTO.md` la vigila nadie**, a diferencia de las del
`README.md` de la raíz. Por eso el LEEME ahora dice, con esas palabras, que sus
números se cuentan a mano y que si no cuadran gana el directorio.

Lo mismo pasaba con la sección «Qué NO hace este paquete», que seguía afirmando
que el paquete no cubría la retención documental ni los modelos de uso general
después de que otra tanda escribiera las siete retenciones y el artículo 52.1.
Un documento que enumera lo que falta y no se actualiza **acaba escondiendo lo
que hay**, que es el daño opuesto y menos visible.

## H7. El censo no tenía contado el artículo 111.3, y es una fecha cierta de 2027

La ficha de `docs/censo-relojes.md` cuenta el artículo 111.2 como evento y el
111.4 (que añadió el ómnibus) como obligación nueva. **El apartado 3 no aparece
en ninguna de las dos listas**, y es un reloj con fecha escrita en el texto:

> Los proveedores de modelos de IA de uso general que se hayan introducido en el
> mercado antes del 2 de agosto de 2025 adoptarán las medidas necesarias para
> cumplir las obligaciones establecidas en el presente Reglamento **a más tardar
> el 2 de agosto de 2027**.

Es texto original del reglamento, no del ómnibus (comprobado: el punto 39
sustituye el apartado 2 y añade el 4, y no toca el 3), así que llevaba en el
censo desde el primer día sin que nadie lo contara. Está escrito, con la misma
forma `puntual` del 111.4 y sus tres dorados.

**El denominador de la fila de `ai-act` sube en 1 como mínimo.** No se toca
`docs/censo-relojes.md`, que está fuera de esta columna, y la fila ya llevaba
escrito que estaba pendiente de un recuento entero.

## Cobertura: qué quedó sin mapear, nominalmente

Medido contra la lista de apartados de la propia ficha del censo, uno a uno. La
ficha cuenta **14 de plazo + 2 de periodicidad + 10 de evento = 26**, y declara
**25** netos por un solapamiento («51 con 52» en evento y el 52.1 en plazo).

**17 de los 25 están mapeados. Quedan 8 sin mapear**, con su motivo:

| apartado | qué es | por qué no está |
|---|---|---|
| art. 5.3 | 24 h, autorización del uso urgente de identificación biométrica remota en tiempo real | obliga a una **autoridad garante del cumplimiento del Derecho**, no al comprador objetivo. Se escribe cuando haya un perfil de fuerzas y cuerpos de seguridad |
| art. 26.10 | 48 h, identificación biométrica en diferido | lo mismo |
| art. 20.1 | no conformidad detectada por el proveedor | presupuesto: es escribible hoy |
| art. 20.2 | riesgo del art. 79.1 conocido, investigación inmediata | presupuesto |
| art. 24.4 | obligación del distribuidor | presupuesto |
| art. 26.5 | quien despliega detecta un riesgo | presupuesto |
| art. 43.4 | modificación sustancial, nueva evaluación de la conformidad | presupuesto |
| art. 111.2 | tope del 02-08-2030 para el alto riesgo destinado a **autoridades públicas** | necesita un hecho que hoy no existe en ninguna parte. Colgarlo de `ambito(sector_publico)` sería una regla correcta e **incompleta**, porque deja fuera al proveedor privado que suministra a una administración, y una regla incompleta no da error en ningún sitio |

Y dos cosas que la cuenta de arriba no recoge:

- **El art. 60.7 no es una fila nueva.** Manda informar de un incidente grave
  detectado durante las pruebas «de conformidad con el artículo 73», que ya está
  escrito. Escribirlo aparte pondría dos filas de calendario para una sola
  notificación, que es lo que pasó con el art. 33.2 del ENS.
- **Las prohibiciones nuevas del art. 5** (letras b bis y b ter y apartados 1 bis
  y 1 ter, aplicables desde el 02-12-2026) siguen sin escribirse. Una prohibición
  es un deber `continua` con fecha de arranque, no un plazo, y tiene fecha
  encima: **tres meses**.

## Lo que se descartó, que también es un resultado

- **art. 60.4, letra b)**: *«si la autoridad de vigilancia del mercado no responde
  en un plazo de treinta días, se entenderá que [...] han sido aprobados»*. Los
  treinta días **no cuelgan de ningún verbo del obligado**: son el silencio de la
  autoridad, y del silencio nace una aprobación tácita, no un acto exigible.
  Escribirlo como reloj metería en el calendario una fila que el obligado no
  puede cumplir ni incumplir.
- **art. 47.4, primera frase**: *«al elaborar la declaración UE de conformidad, el
  proveedor asumirá la responsabilidad del cumplimiento»*. Atribuye
  responsabilidad, no manda hacer nada en ningún momento. Es exactamente el caso
  del art. 22 del CRA: cambia **quién** responde, no **cuándo**. El reloj es la
  segunda frase, «mantendrá actualizada [...] según proceda», que sale como
  `continua`.
- **art. 60 bis** (insertado por el ómnibus): existe solo si el Estado miembro
  adopta un marco de pruebas, y traslada las condiciones del art. 60.4, letras d)
  a j), leyendo «autoridad de vigilancia del mercado» como la autoridad nacional
  competente. Depende de una opción de un Estado miembro que plazum no conoce.
  Va citado en la obligación del art. 60.4.f, no escrito aparte.

## Las mutaciones, con su rojo

Cinco, todas sobre estado commiteado, aplicadas y restauradas con copia en dos
comandos separados.

| id | qué se rompió | qué se puso rojo |
|---|---|---|
| M-A | la regla del art. 22.4 pierde la cláusula del alto riesgo | el representante de un modelo de uso general deriva el aviso hacia la **autoridad de vigilancia del mercado**, que no es el suyo |
| M-B | `traslado: siguiente_habil` en el tope de seis meses | dos dorados: el motor da 01-03-2027 donde el texto da 28-02-2027, y 22-03-2027 donde da 20-03-2027 |
| M-C | `antelacion: "P30D"` en el preaviso (con el parche de H1 puesto) | dos dorados: el hito sale «determinado» donde el texto dice «sin plazo legal» |
| M-D | el art. 54.3.b toma el disparador del art. 22.3.b | sus tres dorados: falta el hecho, o sea que declarar la comercialización de un **sistema** ya no movería la retención del **modelo** |
| M-E | la fecha del art. 111.3 se mueve un día | sus tres dorados, con la fecha del motor al lado de la del texto |

Una sexta no llegó a serlo: un `sed` sobre el JSON no casaba por las comillas
escapadas y **daba verde con la mutación sin aplicar**. Se cazó comparando
`git diff --stat` y contando ocurrencias antes de leer el resultado, que es
justamente el aviso que `CLAUDE.md` da sobre las mutaciones que no se aplican.

## Los errores propios de este frente

1. **Escribí tres dorados de estado «pendiente de hecho» sin comprobar que el
   corredor los admitía.** No los admite: exige el disparador. Los cuatro casos
   («si no haces pruebas, esto no te vence») son legítimos como lectura y no se
   pueden escribir hoy, así que se cambiaron por variantes que sí ejercen algo
   (la hora del hecho, el fin de semana, el instante movido). La lección es la de
   siempre: la forma del caso se comprueba contra la puerta antes de escribir
   diecinueve.
2. **Di por hecho que el ómnibus no tocaba el artículo 60** y estuve a punto de
   escribir la vigencia sin mirarlo. Sí lo toca: el punto 24 sustituye sus
   apartados 1 y 2 y ensancha la población al anexo I, sección A. No cambia la
   letra f), pero cambia a quién alcanza, y eso está ahora en la cita y en la
   regla.
3. **La primera versión de la vigencia del art. 60 iba a heredarse de otra
   obligación** («2026-08-02, como las demás»). Heredarla habría sido copiar
   también sus dos lecturas divergentes del capítulo III, que al art. 60 **no le
   tocan**. La cadena de exclusión que lo demuestra desde el propio texto
   (art. 50.6 y art. 88.1 con art. 60.6) está en `COMPUTO.md`, sección 6, y es
   trabajo que el encargo avisaba de no saltarse: *no lo heredes de otra
   obligación sin mirarlo*.
