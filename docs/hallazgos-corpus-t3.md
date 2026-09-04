# Hallazgos de la rebanada del corpus, tramo 3 (04-09-2026)

Este fichero es el informe de la rebanada 2 del tramo 3. Su columna son
`paquetes/`, `docs/censo-relojes.md` y este fichero.

## 1. Las tres cifras, antes y después

Las tres van siempre juntas, y salen del log de la puerta
(`TestElPorcentajeDeLaV1LoComputaUnTestYNoUnaPersona`), no de una cuenta a mano.

| | antes | después |
|---|---|---|
| cobertura estricta de la v1 | 73 / 142 = **51,4 %** | 89 / 157 = **56,7 %** |
| rituales de plazum sobre esos marcos | **+69** | **+69** |
| marcos sin denominador | **7** (37 relojes y 28 rituales escritos) | **7** (42 relojes y 29 rituales escritos) |

**El numerador sube 16 y el denominador 15**, así que el porcentaje se mueve
cinco puntos y no veinte. Eso es a propósito: el aviso del encargo decía que el
fallo probable de esta métrica es favorecerte, y que si el número sube mucho hay
que mirar primero si el denominador se quedó quieto. No se quedó: la fila de
`ai-act` se recontó entera y pasó de 26 a 40, y la de `rgpd` de 9 a 10.

## 2. El hallazgo principal: los 69 que faltaban no eran 69

El encargo pedía «los 69 relojes de norma que faltan en los marcos con censo».
Descompuesto contra el árbol, ese 69 es esto:

| cuántos | dónde | qué son |
|---|---|---|
| **44** | `nis2-tecnica` | puntos del anexo del Reglamento de Ejecución (UE) 2024/2690 que imponen cadencia **sin número**. Están escritos, transcritos y con dorados, y son rituales por D-12. No pueden pasar al numerador mientras el legislador no ponga una cifra |
| **11** | `dora` | diez puntos sin número ya cubiertos por rituales, y el art. 11.4, que **el propio censo refutó como reloj** el 02-09-2026 (es el mismo acto que el art. 11.6, letra a), con el número una línea más abajo) |
| **3** | `dora` | puntos absorbidos por la obligación escalonada del art. 19: cuatro puntos censados, una obligación escrita, porque es un deber con tres hitos |
| **2** | `ens` | el art. 10.3 es ritual (la norma dice «periódicamente» y no da número), y el art. 33.2 **es el mismo deber** que la ITS de Notificación de Incidentes, apartado IV.3, que ya lleva el reloj |
| **2** | `rgpd` | deuda real: arts. 35.1 y 36.1. Escritos |
| **7** | `ai-act` | deuda real sobre el denominador viejo. Escritos, y el recuento entero encontró más |

**Cinco de los sesenta y nueve eran escribibles con el denominador de partida.**
El resto es o imposible de cerrar (el hueco es de la norma, no del corpus) o una
discrepancia de unidad de cuenta.

Lo que esto dice del número: **la cobertura estricta no puede llegar a 100 %
mientras el denominador cuente puntos que la norma deja sin cifra.** El techo
real de `nis2-tecnica` es 4 de 48, y no es deuda: es la descripción correcta de
un anexo que manda revisar y no dice cada cuánto.

### El caso de `ens`, que es el que más se paga por no escribir

El art. 33.2 del RD 311/2022 manda notificar al CCN «de acuerdo con la
correspondiente Instrucción Técnica de Seguridad», y la ITS IV.3 es la que da el
momento («en el momento en que se produzcan», modelado como `PT0H`). La
obligación `ens.art33.2.notificacion_al_ccn` existe **sin reloj** y la del ITS lo
tiene, y su cita nombra el art. 33.2. Ponerle un segundo reloj al art. 33.2
habría subido el numerador en uno y habría puesto **dos filas de calendario para
una sola notificación al CCN**. Es el patrón del art. 11.4 de DORA y del art.
14.1 con el 14.2 del CRA, en su tercera aparición, y en una clase
`notificatoria`, que es donde más caro sale.

## 3. Lo escrito: 21 relojes de la norma, 1 ritual, 63 dorados

Todos verificados contra instantánea ingerida con huella (invariante 10). Ni un
dato de un informe, de una sesión anterior ni de memoria.

### `ai-act` (13 obligaciones nuevas y 1 reloj puesto a una que no lo tenía)

Fuente: `corpus-vigilancia/ue-32024r1689`, CELEX `32024R1689`, ELI
`reg/2024/1689/oj`, huella del documento
`sha256:7c790c4bb6489d865981c05598209dff6f12e2fcbce3607ba1a3d4a4bcc8ef40`,
obtenida de Cellar el 03-09-2026. Y `corpus-vigilancia/ue-32026r1744`, CELEX
`32026R1744`, huella del art. 1
`sha256:32deb09cea2baad0cfb197125208775849617a7ed1424051250b598c6abb3a90`.
Artículos leídos enteros el **04-09-2026**: 5, 17, 18, 19, 20, 22, 23, 24, 26,
27, 43, 47, 50, 52, 54, 55, 60, 72, 73, 111 y 113, más el anexo III.

| artículo | qué obliga | primitiva | clase |
|---|---|---|---|
| 20.1 | medidas correctoras inmediatas y aviso a la cadena de suministro | plazo sin cifra | remediación |
| 20.2 | investigar el riesgo del art. 79.1 e informar a la autoridad de vigilancia | plazo sin cifra | notificatoria |
| 24.4 | distribuidor: medidas correctoras e información inmediata | plazo sin cifra | notificatoria |
| 26.5 (2ª frase) | quien despliega informa del riesgo y suspende el uso | plazo sin cifra | notificatoria |
| 26.5 (3ª frase) | quien despliega informa del incidente grave | plazo sin cifra | notificatoria |
| 26.10 (párr. 1) | solicitar la autorización de la biometría en diferido | plazo **PT48H** | notificatoria |
| 26.10 (párr. 6) | informe **anual** del uso de la biometría en diferido | periódica **P12M**, `fijado` | notificatoria |
| 27.1 | evaluación de impacto en los derechos fundamentales antes de desplegar | plazo sin cifra | procedimental |
| 27.3 | notificar sus resultados a la autoridad de vigilancia | plazo sin cifra | notificatoria |
| 43.4 | nueva evaluación de la conformidad por modificación sustancial | plazo sin cifra | procedimental |
| 55.1.c | comunicar el incidente grave del modelo de uso general **a la Oficina de IA** | plazo sin cifra | notificatoria |
| 60.7 | medidas de reducción inmediatas o suspensión de las pruebas | plazo sin cifra | remediación |
| 111.2 (2ª frase) | conformidad de los sistemas para autoridades públicas | **puntual** 02-08-2030 | procedimental |
| 73.6 (párr. 1) | investigación «sin demora» tras notificar el incidente grave | plazo sin cifra | remediación |

39 + 3 dorados, 20 reglas de aplicabilidad nuevas.

### `rgpd` (2 obligaciones nuevas)

Fuente: `corpus-vigilancia/ue-32016r0679`, huella del documento
`sha256:97645ca81d2603954eb1efd2db7c2bb605361f6d86436c60219b70a4aad6a7d6`. Arts.
35 y 36 leídos enteros el 04-09-2026.

- **art. 35.1**: realizar la evaluación de impacto **antes del tratamiento**.
  Plazo sin cifra: la norma dice antes, no dice cuánto antes.
- **art. 36.1**: consultar a la autoridad de control cuando la evaluación muestre
  alto riesgo no mitigado. Plazo sin cifra, `notificatoria`.

6 dorados. **Las ocho semanas del art. 36.2 no son de esta obligación**: son de
la autoridad de control para asesorar por escrito, y copiárselas al responsable
sería darle un plazo que la norma da a otro.

### `nis1-es` (5 de la norma y 1 ritual)

Fuente: `corpus-vigilancia/es-boe-a-2021-1192`, BOE-A-2021-1192, ELI
`es/rd/2021/01/26/43`, huella del documento
`sha256:bfed8abbbcde9f43cab1e09746eb68aa10b021e50fea624592e0e7d7c24803f1`. Arts.
6 y 7 leídos enteros el 04-09-2026. Las tres fechas del real decreto, por
separado: disposición 26-01-2021, publicación 28-01-2021, entrada en vigor
29-01-2021, que es la que rige.

| artículo | qué obliga | primitiva |
|---|---|---|
| 7.1 | designar al responsable de la seguridad de la información | plazo **P3M** |
| 7.2 (1er deber) | comunicar la designación a la autoridad competente | plazo **P3M** |
| 7.2 (2º deber) | comunicar nombramientos y ceses | plazo **P1M** |
| 6.4 (1er deber) | remitir la Declaración de Aplicabilidad | plazo **P6M** |
| 6.4 (2º deber) | revisar la Declaración de Aplicabilidad | periódica **P36M**, `suelo_legal` |
| 7.3.b | **ritual**: controles periódicos de seguridad | periódica **P12M**, `propuesto` |

18 dorados. Es el único marco de la familia NIS que vincula **hoy** en España, y
tenía dos relojes escritos de los ocho que su ficha contaba.

## 4. El ritual, y por qué su intervalo es defendible ante un auditor

Sólo se ha escrito **uno**, y no por pereza: en los otros dos marcos sin
denominador que se podían transcribir no hay sitio para un ritual, y eso también
se midió (sección 6).

**`nis1es.art7_3b.controles_periodicos_de_seguridad`, P12M, `propuesto`.** El
art. 7.3, letra b), del RD 43/2021 tiene tres verbos y sólo el tercero lleva el
adverbio: «llevar a cabo **controles periódicos** de seguridad». No da cifra, así
que el número es de plazum y el campo `articulo` lo declara ritual.

El argumento sale de **dos números del propio real decreto**, no de una costumbre
del sector, y eso es lo que lo hace contestable:

1. El **art. 6.4** obliga a revisar la Declaración de Aplicabilidad al menos cada
   **tres años**, y esa revisión decide si las medidas siguen siendo adecuadas.
   Si los controles corrieran a la misma cadencia, la revisión trienal decidiría
   sobre **una sola medición**, de hasta tres años: revisaría el documento y no
   las medidas. Doce meses es el intervalo más largo que garantiza al menos tres
   mediciones antes de cada revisión, que es el mínimo para ver una tendencia y
   no un punto.
2. El **art. 6.5** manda tomar como referencia las medidas del anexo II del
   Esquema Nacional de Seguridad, y el ENS reevalúa **anualmente** la categoría
   del sistema, que es lo que determina qué medidas son exigibles. Controlar con
   menos frecuencia que la reevaluación de la que cuelgan las medidas deja un año
   en el que la categoría ya cambió y los controles todavía no lo saben.

`cuando_cambiarlo` dice una condición para acortar (sistema en evolución
continua, y siempre tras un ciberincidente de nivel ALTO o superior de la tabla 3
o tras una revisión que cambie las medidas) y una para alargar (sólo con
certificación ENS en vigor sobre el mismo perímetro, y nunca más allá de P24M,
porque a partir de ahí la revisión trienal se queda con una sola medición).
`fuentes_del_intervalo` nombra los tres documentos.

## 5. Lo que este trabajo REFUTA del censo, con la fuente de cada refutación

### 5.1. «El AI Act no fija ni una cadencia numérica» es falso

La ficha de `ai-act` afirmaba, en negrita:

> el AI Act no fija ni una cadencia numérica al proveedor ni al responsable del
> despliegue

El **art. 26.10, párrafo sexto** obliga al responsable del despliegue de un
sistema de identificación biométrica remota en diferido a presentar **informes
anuales** a la autoridad de vigilancia del mercado y a la autoridad nacional de
protección de datos. Cadencia con número, de un obligado y no de una autoridad, y
por D-12 es `fijado` y no `suelo_legal`, porque dice «anuales» sin «al menos».

**Cómo se escapó**, dicho para que la próxima ejecución no dependa de que alguien
se acuerde: el marcador estaba en el **párrafo sexto de un apartado décimo**, y
el barrido troceaba por apartado y leía el principio. Es la forma del fallo de la
primera pasada del censo, un nivel más abajo.

### 5.2. La primera frase del art. 111.2 no es un reloj

Dice a quién se aplica el reglamento («se aplicará a los operadores [...]
**únicamente si** [...] dichos sistemas se ven sometidos a cambios significativos
en sus diseños»). Cambia **quién** está obligado, no **cuándo**. Es exactamente la
forma del art. 22 del CRA, que este censo ya descartó por no colgar de ningún
verbo temporal, y la ficha la tenía contada como evento. Escribirla con reloj
habría metido en el calendario una fila que no vence nunca.

Lo que sí es reloj es su **segunda** frase, y está escrita.

### 5.3. Los arts. 73.2, 73.3 y 73.4 no son tres puntos

Son los tres límites del deber del art. 73.1, igual que el art. 14.2 del CRA son
los plazos del 14.1. El paquete ya los tenía escritos como una sola obligación
con tres hitos, y la ficha contaba cuatro puntos.

### 5.4. Once apartados del AI Act no estaban en ninguna columna

Arts. 5.2, 5.3 (segundo deber), 5.4, 27.1, 27.2, 27.3, 47.4, 50.5, 55.1.c, 60.8
y 73.6. **Dos de ellos ya estaban escritos en el paquete** (50.5 y 60.8): eran
relojes que aportaban al numerador y a ningún denominador, que es una de las tres
formas conocidas de inflar esta métrica.

### 5.5. `rgpd`: «35.1 con 36.1» son dos puntos

Dos apartados de dos artículos, dos actos, dos hechos de arranque y dos
destinatarios. Se incumplen por separado y la unidad de la sección 1 del censo es
el par (artículo, apartado). La fila pasa de 9 a 10.

### 5.6. `nis1-es`: el art. 6.4 no estaba en la ficha

La ficha cuenta ocho plazos y no nombra el art. 6.4, que trae dos deberes: la
remisión de la Declaración de Aplicabilidad en seis meses y su revisión al menos
cada tres años.

Y **una discrepancia de numeración dentro de la propia norma**, anotada para que
nadie la arregle a ojo: el art. 7.3, letra c), manda elaborar la Declaración de
Aplicabilidad «considerado en el artículo 6.3 párrafo segundo», y en el texto
consolidado ingerido la Declaración vive en el **apartado 4**.

### 5.7. `paquetes/CORPUS.md`: el «27 obligan sin número» no era reconstruible

Y es la **tercera** vez que le pasa a la misma frase, que ya cuenta las dos
anteriores (dijo trece cuando enumeraba ocho, y dijo 28 cuando su criterio daba
19). Medido sobre el árbol del 03-09-2026 con el criterio literal que ella misma
escribía (hitos que no son de cadencia, con `limite` vacío o `indeterminado`):
salen **69**. Y no hay combinación de exclusiones de primitiva que dé 27
(excluyendo también `maximo`, 52; excluyendo además `continua`, 47). Contar
obligaciones en vez de hitos da los mismos números.

Corregido con el número que el criterio produce, y **el criterio pasa a decir qué
primitiva excluye**, que es lo que le faltaba las tres veces.

## 6. Lo que se DESCARTA, con su motivo

Descartar es la mitad del trabajo y es lo que impide que el calendario se llene
de filas que nadie puede cerrar.

| qué | por qué no se escribe |
|---|---|
| art. 111.2, 1ª frase (AI Act) | regla de atribución, no reloj (sección 5.2) |
| art. 60.7, 1ª frase (AI Act) | remite «de conformidad con el artículo 73», que ya está escrito con sus quince, dos y diez días. Escribirlo aparte daría **dos filas para una sola notificación a la misma autoridad** |
| art. 73.6, párr. 2 (AI Act) | es un **preaviso** («sin haber informado antes»), y la primitiva `preaviso` sigue apagada para el corpus. Disfrazarlo de plazo diría al obligado que puede modificar el sistema y avisar después |
| art. 33.2 (ENS) | el mismo deber que la ITS de Notificación de Incidentes IV.3, que ya lleva el reloj |
| art. 11.4 (DORA) | ya refutado por el censo el 02-09-2026 |
| art. 20.2 (NIS2) | ya refutado por el censo el 02-09-2026: el adverbio cuelga de «alentarán», que no impone resultado a la entidad. La decisión se respeta y no se revisa |
| art. 7 del RDL 12/2018 | **el RDL 12/2018 no está en `corpus-vigilancia/`**. Invariante 10: al corpus sólo entra lo verificado contra fuente primaria en el momento de escribirlo |
| disposición adicional única (RD 43/2021) | fecha que venció el 29-04-2021. Enseñarla hoy pondría en el calendario una fila vencida de un deber que ya no nace |
| art. 14.3.a (RGPD) | necesita una primitiva que sepa decir «el más temprano de N límites condicionales» (sección 9) |

**`cra` no admite rituales, y se midió.** Barrido el articulado y los anexos de
`corpus-vigilancia/ue-32024r2847` con las formas de cadencia de la sección 2 bis
del censo: la **única** dirigida al obligado es el anexo I, parte II, punto 3, y
**ya está escrita**. Todo lo demás es de la Comisión (arts. 9.2 y 70.1), de los
organismos notificados (anexo VIII) o de las autoridades de vigilancia del
mercado (art. 52). El CRA es un reglamento de plazos y de eventos, no de
cadencias, y ésa es la respuesta y no un hueco.

## 7. Las guardas que nacieron rojas sobre dato real

Dos, y ninguna la puso nadie a propósito. Valen más que las seis mutaciones de
abajo.

### 7.1. La guarda de la fracción, sobre `ai-act`

```
la parte es mayor que el total: 32 relojes con cita escritos en ai-act
sobre 26 puntos censados de ai-act.
```

Dijo exactamente lo que tenía que decir: **el denominador cuenta APARTADOS y el
numerador cuenta OBLIGACIONES**, y en cuanto un apartado lleva dos deberes
exigibles por separado (arts. 26.10, 26.5 y 60.4.f del AI Act, art. 7.2 del RD
43/2021) la fracción se pasa de uno. Por eso la fila de `ai-act` se ha recontado
**en deberes**, que es la unidad del numerador.

**Y por eso las otras diecisiete filas con número siguen expuestas al mismo
salto**: siguen contando apartados. Cada una que se recuente en deberes va a
subir, así que el 310 del barrido de disyunción es un suelo más blando de lo que
parece. Con las cuatro filas que ya tienen número nuevo, el suelo va a **329**, y
sigue sin escribirse como total por la misma razón que el censo ya declaraba.

### 7.2. La guarda de la vigilancia normativa, cazando un error mío

Quité el `espera` de las dos lecturas divergentes del art. 73.6 creyendo que
apuntaba a un hecho ya ocurrido (el ómnibus se publicó el 24-07-2026, y la propia
cita dice PUBLICADO Y VINCULANTE). `TestTodoItemDeVigilanciaCasaConSuCorpusEnLasDosDirecciones`
se puso rojo con tres líneas y tenía razón: **`espera` en un item ya disparado no
es un residuo caducado, es la lista de pendientes** de ese item, cuyo
`al_dispararse` dice que cada lectura afectada se resuelve a mano. Revertido.

## 8. Las mutaciones

Seis, con el árbol limpio y sobre estado commiteado.

**`.github/mutar.sh` no funciona en un worktree** y hay que decirlo: hace
`mkdir -p .git/mutaciones`, y en un worktree `.git` es un **fichero** que apunta
al gitdir real, así que sale con
`mkdir: cannot create directory '.git': Not a directory` y después
`PARADA: no hay ninguna mutacion preparada`. Ese fichero es de la columna de la
rebanada 0, así que aquí sólo se reporta. Las seis mutaciones se hicieron con un
equivalente que conserva las cuatro guardas y lleva el depósito fuera del árbol.

| # | qué se muta | qué tiene que ponerse rojo | resultado |
|---|---|---|---|
| M1 | `"limite": "PT48H"` → `PT72H` (art. 26.10) | ejecutor de dorados | **CAZADA** |
| M2 | `origen_del_intervalo: fijado` → `propuesto` (informe anual del art. 26.10) | linter de paquetes | **CAZADA** |
| M3 | el disparador del art. 36.1 del RGPD pasa a ser el hecho del art. 35.1 | ejecutor de dorados | **CAZADA** |
| M4 | `"cadencia": "P36M"` → `P48M` (revisión de la Declaración de Aplicabilidad) | ejecutor de dorados | **CAZADA** |
| M5 | `origen_del_intervalo: suelo_legal` → `propuesto` (mismo reloj) | linter de paquetes | **CAZADA** |
| M6 | `cita_del_intervalo` pasa a decir «cada CINCO años» donde la norma dice tres | linter de paquetes | **SOBREVIVE** |

M3 es la que más importa: el tercer dorado del art. 36.1 existe para afirmar que
**haber hecho la evaluación de impacto no obliga a consultar a la autoridad**, y
sin él nada impediría encadenar los dos relojes y llevar ante el supervisor a
quien mitigó el riesgo. Es el control positivo de un descargo, que sin un caso
que lo recorra no existe.

**M6 es un agujero y se declara como tal (P2).** `cita_del_intervalo` sólo se
comprueba por longitud (40 caracteres útiles). Un reloj con `suelo_legal` puede
citar un plazo que la norma no dice, con la cadencia correcta y los dorados en
verde, y nada se pone rojo. Es la familia de «¿de dónde salen las palabras de
este campo?»: el linter cierra la vía del nombre y la de la longitud, no la del
contenido. **Hay un arreglo mecánico disponible y no se ha hecho aquí porque
vive en `nucleo/corpus/`, que no es de nadie en este tramo**: el `texto_legal`
está en el mismo objeto, así que el linter puede exigir que el fragmento
entrecomillado de `cita_del_intervalo` aparezca en él. Necesita normalizar
(minúsculas y sin tildes, porque el corpus escribe las citas sin tildes) y
admitir los corchetes de elisión.

## 9. Lo que para, y por qué (decisiones que no son mías)

### 9.1. Una primitiva que no existe: «el más temprano de N límites condicionales»

El art. 14.3 del RGPD da tres momentos y el que rige es **el más temprano de los
que apliquen**: un mes desde la obtención de los datos (letra a), el momento de
la primera comunicación al interesado (letra b) y el de la primera cesión (letra
c). El `tope` del motor admite **uno solo**. Es el único punto de `rgpd` que
queda sin escribir, y no se escribe la letra a) sola porque daría una fecha **más
tarde** que la legal siempre que aplique la b) o la c), que es la dirección en la
que un producto de cumplimiento no puede equivocarse.

Ya estaba documentado en el censo; se confirma desde el corpus y se cuenta aquí
con su cardinal: **1 reloj esperando**.

### 9.2. `preaviso` sigue apagada, y ahora tiene un consumidor más

El art. 73.6, párrafo segundo, del AI Act prohíbe modificar el sistema afectado
sin haber informado antes a las autoridades competentes. Es un preaviso. Con
éste, la deuda de `preaviso` sube en uno respecto de lo contado en `CLAUDE.md`.

### 9.3. Dos guardas del calendario se contradicen, y el dato real que las enfrenta lo trae esta rebanada

`aiact.art111_2` es **el primer reloj del corpus cuyo único vencimiento cae más
allá de la ventana de doce meses** (02-08-2030). El cubo que existe para ese caso
estaba declarado a cero desde que se escribió, con estas palabras:

> VACIO HOY, y se declara igual. Un cubo que solo aparece cuando tiene algo
> dentro es un cubo que nadie echa de menos: con el cero escrito, el dia que deje
> de estar vacio esta puerta lo dice.

Dejó de estar vacío, y quien habló fue **otra** puerta, acusando al producto de
perder un reloj que no ha perdido:

```
--- FAIL: TestTodoRelojAlcanzadoSaleEnAlgunSitio
    1 de 226 obligaciones con reloj en vigor NO salen ni con fecha ni sin ella.
    Las primeras: [aiact.art111_2.puesta_en_conformidad_de_los_sistemas_para_autoridades_publicas]
```

**Es la guarda la que está corta, no el producto.** Comprobado ejecutando
`Derivar12Meses` sobre el corpus con `TodoAplica` y sin ningún hecho: el reloj
sale en `VencimientosMasAlla`, con su fecha, su título, su artículo y su hito, y
`MasAllaDeLaVentana` vale 1. O sea que **no desaparece**: está contado y está
listado, que es exactamente lo que D-13 exige. Lo que pasa es que
`TestTodoRelojAlcanzadoSaleEnAlgunSitio` sólo mira `Fechas` y `SinFecha`, y el
calendario tiene **tres** destinos legítimos para un reloj alcanzado, no dos. La
cabecera de esa ley es anterior al cubo.

**El arreglo es de una línea y no se hace aquí**, porque vive en
`nucleo/pantalla/conservacion_relojes_test.go` y `nucleo/` no es de nadie en este
tramo: añadir las filas de `cal.VencimientosMasAlla` al mapa `salen`, con el
motivo escrito (un vencimiento posterior a la ventana es una respuesta, no una
desaparición). **Deja roja la puerta `cobertura del nucleo` de CI hasta que se
haga.**

Y la lección del caso, que es lo que vale: **un cubo declarado a cero es una
predicción, y el día que se llena no basta con que hable su propia puerta**. Hay
que mirar también qué otras guardas dan por imposible ese estado, porque son las
que van a acusar en falso.

### 9.4. La matriz del tramo deja seis ficheros de raíz sin dueño, y son los que
esta rebanada mueve

`.github/frontera.sh corpus main tramo3/corpus` sale **verde**: 28 ficheros,
todos en la columna. Pero seis ficheros que **congelan números del corpus** no
están asignados a ninguna rebanada, y por eso quedan **rojos**:

| fichero | qué congela | qué hay que poner |
|---|---|---|
| `README.md` | cobertura de la v1 | **56,7 %**; «+69 rituales» no cambia; «7 de los 15 marcos» no cambia; «sin denominador, 28 rituales y 37 relojes» → **29 rituales y 42 relojes** |
| `README.md` | hitos y dorados | **271 hitos**, **766 casos dorados** |
| `ETAPAS.md` | relojes escritos | **252** (decía 230) |
| `ciso_de_doscientos_test.go` | `ObligacionesQueVeElCiso` | **74** (decía 72) |
| `puente_piloto_test.go` | `ObligacionesQueDerivaElPuente` | **218** (decía 207) |
| `conservacion_calendario_test.go` | `CensoEsperado` y la lista de relojes declarados | `relojSeVe: 94`, `relojNingunPerfilLoAlcanza: 136`, y añadir a la lista `rgpd.art35_1.evaluacion_de_impacto_antes_del_tratamiento` y `rgpd.art36_1.consulta_previa_a_la_autoridad_de_control` |

`README.md` y `ETAPAS.md` están declarados «de nadie» en la matriz, así que su
rojo estaba previsto. **Los otros cuatro no lo están**, y la matriz enuncia la
regla que los asignaría: *«cada fichero de raíz a la rebanada que MUEVE EL NÚMERO
que ese fichero congela»*. La regla existe y la lista no la aplica: sólo enumera
`ttfv_camino_test.go` y `distribucion_test.go`. No se han tocado, porque un
fichero fuera de la columna es un merge rechazado y no una excepción.

**Y los seis números van hacia arriba**, que es lo que importa mirar antes de
pegarlos: ninguno baja, así que ningún reloj que un cliente veía ha dejado de
verse.

## 10. El lazo local, con su código real

`./comprobar.sh`, en segundo plano y a fichero (nunca por `tail` ni por un pipe,
que devolverían el código de `tail`):

```
6 de 21 puertas rotas.
3 puertas saltadas en esta maquina (dicho arriba, con el motivo).
CODIGO_REAL=1
```

Las **quince en verde** incluyen las tres que vigilan lo que esta rebanada toca:
ingesta legal y vigilancia normativa (102 casos, mínimo 70), frontera legal del
catálogo (24 casos, mínimo 20) y el linter de paquetes con el ejecutor de dorados
dentro de la suite.

**Las seis rotas son todas la misma causa, contada dos veces**: cuatro son
variantes de «suite completa» (completa, sin IA, en local, antes de empaquetar) y
las otras dos son `cobertura del nucleo` y `derivación del calendario`. Ninguna
falla por el corpus: fallan por los seis ficheros de la sección 9.4 (números
publicados que esta rebanada mueve y que no son de su columna) y por la guarda
corta de la sección 9.3.

Las tres **saltadas** son las de `-race`, que exigen cgo y esta máquina es
Windows sin compilador de C. Se dicen aquí con su motivo, porque «no se pudo
ejecutar» y «no encontró nada» no son lo mismo.

Las puertas de la columna, ejecutadas aparte para poder afirmarlas:

```
=== RUN   TestTodosLosPaquetesPublicadosPasanElLinter
--- PASS: TestTodosLosPaquetesPublicadosPasanElLinter (0.08s)
=== RUN   TestLosDoradosPublicadosPasanContraElMotor
--- PASS: TestLosDoradosPublicadosPasanContraElMotor (0.07s)
ok  	github.com/marcosmatalab/plazum	0.309s
```

y la frontera:

```
frontera del frente corpus respetada: 28 ficheros, todos en su columna.
```

## 11. Mis errores

1. **Leí `espera` como «esperando un hecho futuro» y también significa «pendiente
   de resolver de un item ya disparado».** Quité dos y la puerta de vigilancia me
   paró. Revertido.
2. **Mi propio script de mutación dio dos CAZADA falsas seguidas**, por dos
   causas distintas y las dos catalogadas en este repositorio: la primera porque
   el baseline (`go test .`) **ya estaba rojo** por los seis ficheros de la
   sección 9.3, así que el rojo no era de la mutación; la segunda porque
   `subprocess.run(shell=True)` en Windows no interpreta las comillas simples de
   un `-run 'A|B'` y `go test` salió con 255 **sin ejecutar un solo caso**. Se
   arregló con una guarda que ahora rechaza un código distinto de cero **sin ni
   una línea `--- FAIL`**, que es la forma exacta de esa trampa.
3. **Escribí tres veces mal la ruta del scratchpad** y creé dos directorios
   basura, borrados después. Sin consecuencia sobre el repositorio.
4. **El recuento del AI Act creció tres veces mientras lo hacía** (26 → 29 → 32 →
   35 → 40), y las tres veces por encontrar deberes que el barrido de marcadores
   no veía. La primera versión de este informe iba a publicar 32 y habría
   inflado el porcentaje casi tres puntos. La lección no es «medir mejor»: es que
   **un recuento que sólo encuentra lo que ya has escrito es sospechoso**, y por
   eso el perímetro leído entero está escrito en el censo, artículo por artículo.
