# Hallazgos del frente B (03-09-2026)

Rellenar los huecos de siete paquetes ya escritos: `ai-act`, `iso42001`, `ens`,
`iso27001`, `rgpd`, `nis2-ue`, `nis1-es`. Este fichero recoge lo que el trabajo
encontró y lo que dejó sin hacer. **`docs/censo-relojes.md` no se toca en esta
campaña**: hay otro frente escribiendo hallazgos a la vez y el integrador funde.

## Lo escrito, medido antes y después

| paquete | relojes antes | relojes ahora | censados |
|---|---|---|---|
| ai-act | 5 | **12** | 25 |
| ens | 8 | **12** | 13 |
| iso42001 | 7 | **10** | sin censo verificado |
| iso27001 | 6 | **9** | 0 (contado y defendido) |
| rgpd | 6 | **8** | 9 |
| nis2-ue | 2 | **4** | 9 |
| nis1-es | 1 | **2** | sin censo verificado |

Total del árbol: 179 hitos y 521 dorados antes, **201 hitos y 574 dorados**
ahora. Los recuentos de la columna «antes» se verificaron contando el bloque
`temporalidad` de cada `paquete.json` sobre `b4e550d`, no se copiaron del
encargo, y coinciden con él.

## Los once hallazgos

### H1. El art. 9.1 del RD 43/2021 son DOS deberes, no uno

El párrafo 1 obliga a notificar los **incidentes** por su nivel de **impacto**
(apartado 4 de la Instrucción del anexo). El párrafo 2 obliga, «asimismo», a
notificar los **sucesos o incidencias** por su nivel de **peligrosidad**
(apartado 3 de la misma Instrucción) *«aun cuando no hayan tenido todavía un
efecto adverso real»*. Distinto objeto, distinto umbral y distinta escala. El
paquete tenía el art. 9.1 como una sola obligación sin reloj, así que la vía que
existe para avisar **antes de que pase nada** no producía ninguna fila. Escrito
como `nis1es.art9_1p2.notificacion_de_sucesos_por_peligrosidad`.

Es la familia de «los arts. 14.1/14.2 del CRA son un punto y el art. 5.2 de DORA
eran cuatro», por el lado de partir en vez de por el de juntar.

### H2. El art. 47.1 del AI Act tiene TRES verbos y los diez años cuelgan del segundo

*«El proveedor **redactará** una declaración UE de conformidad [...] y la
**mantendrá a disposición** de las autoridades nacionales competentes durante un
período de diez años [...]. **Se entregará** una copia [...] a las autoridades
[...] que lo soliciten.»* Los diez años son de «mantener a disposición». Colgarlos
de «redactar» habría dado una obligación que se cumple sola el día que se firma
el documento; colgarlos de «entregar» habría dado una que no vence nunca porque
depende de una solicitud ajena.

### H3. La primitiva `preaviso` tiene un octavo reloj esperando, y está DENTRO de la v1

`nucleo/corpus/primitivas_encendidas.go` declara `preaviso` apagada con **7
relojes esperando** y con este motivo escrito: *«los siete están FUERA de los 12
marcos de la v1 (psd2, mica, mdr, data-act), así que la deuda no bloquea la
v1»*. Eso ya no es cierto.

El art. 60.4, letra f), del AI Act dice que las pruebas en condiciones reales no
duran más de seis meses, *«que podrán prorrogarse por un período adicional de
seis meses, con sujeción al envío de una **notificación previa** por parte del
proveedor [...] a la autoridad de vigilancia del mercado»*. Es un plazo que
corre hacia atrás desde una fecha que **elige el obligado** (el inicio de la
prórroga), que es exactamente `preaviso`, con `antelacion: "indeterminado"`
porque el apartado no dice cuánto antes.

**No se ha escrito**, y el motivo es de frontera, no de duda: encenderlo obliga a
cambiar `PrimitivasDelCorpus` de `PrimitivaApagada` a `PrimitivaEnUso` en
`nucleo/corpus/primitivas_encendidas.go`, que está fuera de la columna de este
frente. Con el paquete escrito y ese fichero sin tocar,
`TestTodaPrimitivaDelMotorOSeUsaOSeExplica` se pone rojo por su sentido 3.

**Petición al integrador**: son dos relojes del art. 60.4.f (el tope de seis
meses, que es un `plazo`, y la prórroga, que es el `preaviso`), y el cardinal de
`RelojesEsperando` pasa de 7 a 8 con el motivo reescrito, porque el «fuera de la
v1» deja de valer.

### H4. El paquete `rgpd` daba los seis relojes del RESPONSABLE a cualquiera que trate datos personales

Las siete reglas de aplicabilidad de `rgpd` colgaban todas de
`trata_datos_personales(E)`. Los arts. 12.3, 12.4, 33.1 y 34.1 obligan al
**responsable** del tratamiento, no al **encargado**, y las dos figuras las
define el art. 4, puntos 7 y 8. Un encargado puro (un SaaS que solo trata datos
por cuenta de sus clientes) veía las 72 horas del art. 33.1 hacia la autoridad de
control, que no son suyas: las suyas son las del art. 33.2, hacia el responsable
y sin número.

Este frente **no ha corregido las seis reglas existentes**, a propósito: cambiar
`trata_datos_personales(E)` por `papel_rgpd(E, "responsable")` en los seis relojes
que ya estaban los deja invisibles desde los tres perfiles de arranque, que no
afirman el papel, y eso es el hallazgo del que arranca esta campaña. Lo que sí se
ha hecho es añadir el atributo `papel_rgpd` y su pregunta, y colgar de él el
único reloj que es indiscutiblemente del encargado (art. 33.2).

**P1 para el integrador**: la corrección de las seis reglas y el hecho
`papel_rgpd` en los perfiles van juntas o no van.

### H5. El «28 obligan sin número» de `paquetes/CORPUS.md` no era reconstruible

El párrafo declara su criterio: *«hitos que NO son de cadencia y cuyo `limite`
está vacío o vale `indeterminado`»*. Aplicado ese criterio al árbol de `b4e550d`
salen **19**, no 28. El número se ha recomputado con el criterio escrito (19
sobre aquel árbol, **27** sobre este) y se ha dejado dicho en el propio fichero
que la cuenta anterior tampoco era reproducible, que es lo que ese mismo párrafo
ya había hecho una vez con la cuenta de trece.

**Ningún test ata ese número.** `cuentas_publicadas_test.go` y
`marcos_v1_test.go` atan los hitos y los dorados; este se quedó viejo dos veces
seguidas porque nadie lo mira.

### H6. El porcentaje de cobertura de la v1 mezcla rituales de plazum con relojes legales

`TestElPorcentajeDeLaV1LoComputaUnTestYNoUnaPersona` divide **relojes escritos**
entre **relojes censados**. El numerador cuenta el bloque `temporalidad` de cada
obligación, sea un plazo legal o un **ritual de plazum**. `iso27001` tiene
`censados: 0` (contado y defendido: la norma no fija ni un período numérico) y
ahora aporta **9 rituales** al numerador contra un denominador de **0**.

Con esta tanda el porcentaje pasa de 80,5 % a 91,1 %, y **9 de los 154 del
numerador son números que pone plazum, no la ley**. El número no está mal
calculado: está mal nombrado. Dice «cobertura de los relojes censados» y mide
«filas con `temporalidad`».

Arreglo barato y fuera de la columna de este frente: contar en el numerador solo
las obligaciones cuyo `origen_del_intervalo` no sea `propuesto` (y las que no son
periódicas), o publicar los dos números.

### H7. Las lecturas divergentes nuevas del AI Act no las vigila ningún item

Las seis obligaciones de alto riesgo nuevas llevan las dos lecturas divergentes
del ómnibus (`capitulo-iii-anexo-iii` y `capitulo-iii-anexo-i`), y **no declaran
`espera`**, así que no aparecen en el `cuelga_de` de
`vigilancia/items/*.json`. Es legal (`vigilancia_test.go` salta las lecturas sin
`espera`) y está razonado en la propia cita: lo que las resolvería no es un
evento futuro, es un dato estructural que ya existe. Pero el efecto práctico es
que **12 lecturas divergentes nuevas envejecen sin vigilancia**.

`vigilancia/items/` está fuera de la columna de este frente. Si el integrador
prefiere atarlas, son 6 obligaciones × 2 lecturas = 12 entradas de `cuelga_de`,
seis en cada uno de los dos items que ya existen, y hay que añadirles `espera` en
el paquete.

### H8. La instantánea de Cellar no trae capítulos ni secciones, y de eso depende una vigencia

El art. 113, párrafo tercero, letra c), del AI Act (sustituido por el ómnibus)
difiere *«el capítulo III, **secciones 1, 2 y 3**»* a 02-12-2027 y 02-08-2028. La
instantánea `corpus-vigilancia/ue-32024r1689` trae 113 artículos y 13 anexos y
**ni un encabezamiento de capítulo o de sección**: se buscaron `CAPÍTULO [IVX]+`
y `SECCIÓN [0-9]+` y hay cero apariciones. Así que de la fuente ingerida **no se
puede deducir** en qué sección vive el art. 18, el 19, el 22, el 23, el 26 ni el
47, y de eso depende cuál de las tres fechas del art. 113 les toca.

Consecuencia asumida y escrita en cada ficha: la vigencia declarada es la
**general** del párrafo segundo del art. 113 (02-08-2026), que sí está en el
texto ingerido, y las dos del ómnibus van como lecturas divergentes con su cita.
Para una retención, la declarada es además la conservadora: adelanta el arranque,
nunca lo acorta.

Lo que sí se pudo encadenar desde el texto: **el art. 54 vive en el capítulo V**
(su apartado 3, letra c, habla de *«las obligaciones establecidas en el presente
capítulo»* y su letra a nombra los arts. 53 y 55; el art. 41 llama a esas mismas
obligaciones *«capítulo V, secciones 2 y 3»*). Por eso el art. 52.1 se ha fechado
el 02-08-2025 sin lecturas divergentes.

**Petición**: que el ingestor de Cellar guarde el encabezamiento estructural de
cada artículo. Sin él, cualquier norma de la UE con aplicación escalonada por
capítulos tiene este mismo agujero, y hoy son al menos dos (AI Act y DORA).

### H9. Mi propia pregunta ensancha el hueco de la entrevista en 1

`rgpd.q.papel` es necesaria para que el art. 33.2 sea contestable, y la entrevista
asistida no sabe preguntarla. `PreguntasQueNoLleganAlMotor` sube de 36 a 37 y
`TotalDePreguntasDelCorpus` de 41 a 42. Está en la lista de «necesitan un valor
que la entrevista no pregunta», que ya tenía 16 y ahora tiene 17.

### H10. El art. 22.4 del AI Act sí es un reloj, y esta ficha lo dijo mal primero

La primera redacción de `aiact.art22_3b` escribió que el apartado 4 *«cambia
QUIÉN está obligado, no CUÁNDO»*. Es falso: son **dos verbos** y el segundo es
*«informará **de inmediato** de la terminación del mandato [...] a la autoridad
de vigilancia del mercado»*, que es una notificatoria sin número. Corregido en la
propia cita, en voz alta y no en silencio. **No está escrito como obligación**, y
lo mismo vale para su gemelo, el art. 54.5.

Es la pregunta fija «¿de qué verbo cuelga el número?» cazando el error de quien
la estaba aplicando: se miró el primer verbo del apartado y se cerró la pregunta.

### H11. El art. 33.2 del RGPD no puede heredar las 72 horas del 33.1, y la razón es aritmética

El encargado avisa al responsable *«sin dilación indebida»* y sin cifra. Copiarle
las 72 horas del apartado 1 no solo inventaría un plazo: le daría **el mismo**
que al responsable, cuando el reloj del responsable **no arranca hasta que ese
aviso llega**. Un encargado que agotara 72 horas dejaría al responsable con cero.
El dorado del art. 33.2 ejerce ese control: si alguien le pone `PT72H`, se pone
rojo.

## Lo que NO se hizo, con su cardinal

**AI Act, 6 relojes identificados y no escritos** (de los 25 censados, quedan 13
sin mapear en total):

1. **art. 5.3** (24 horas, autorización del uso urgente de identificación
   biométrica remota en tiempo real) y **art. 26.10** (48 horas, identificación
   en diferido). Los dos obligan a una **autoridad garante del cumplimiento del
   Derecho**, no al comprador objetivo del producto. Se escriben cuando haya un
   perfil de fuerzas y cuerpos de seguridad, no antes.
2. **art. 60.4, letra f)**, 2 relojes: el tope de seis meses de las pruebas en
   condiciones reales y la prórroga con notificación previa. Bloqueado por H3.
3. **art. 111.2**, 1 reloj (`puntual`, 02-08-2030). Necesita un hecho que hoy no
   existe en ninguna parte: *«sistema de IA de alto riesgo destinado a ser
   utilizado por autoridades públicas»*. Colgarlo de `ambito(sector_publico)`
   sería una regla correcta pero incompleta (deja fuera al proveedor privado que
   suministra a una administración) y una regla incompleta no da error en ningún
   sitio.
4. **art. 22.4 y art. 54.5**, 2 relojes notificatorios sin número (H10).
5. **art. 54.3, letra b)**, 1 reloj de retención de diez años del representante
   autorizado de un proveedor de modelo de uso general. Es escribible hoy (su
   vigencia se encadena desde el texto, ver H8) y se ha quedado fuera por
   presupuesto, no por duda.
6. **art. 47.4** (*«mantendrá actualizada la declaración UE de conformidad según
   proceda»*), deber permanente sin número, candidato a `continua`.

**ENS**: el art. 33.2 lo cuenta el censo como un evento aparte y el paquete ya lo
tenía cubierto por `ens.its_incidentes.notificacion_al_ccn`, que es donde vive el
número (la ITS, apartado IV.3). Escribirlo otra vez habría puesto dos filas para
un solo acto. **0 relojes nuevos por ahí, y el censo cuenta uno.**

**RGPD**: quedan **2 sin mapear** de los 9. El art. 14.3.a espera a una primitiva
que sepa decir «el más temprano de N límites condicionales» (ya está en el censo
con su motivo) y el art. 35.1 encadenado con el 36.1.

**NIS2-UE**: quedan **5 sin mapear** de los 9, y de ellos 2 están deliberadamente
fuera con su motivo ya escrito en el censo (art. 27.2, fecha vencida sin
transposición española, y art. 20.2, cuyo adverbio cuelga del verbo equivocado).

**ISO**: no se han escrito más rituales de los tres y tres que hay. Un ritual es
un número que **inventa plazum**, y fabricarlos para mover un contador es
exactamente el fallo que D-12 existe para impedir. Los seis escritos tienen su
argumento, su `cuando_cambiarlo` en las dos direcciones y sus fuentes con número.

## Los hechos que ningún perfil afirma (petición al frente D)

Medido leyendo `perfiles/*.json` en modo lectura. Los perfiles afirman hoy:
`ambito`, `papel_cra`, `papel_ia`, `trata_datos_personales`,
`canal_de_denuncias_obligatorio`, `categoria`, `designado(delegado...)`,
`adopta(iso27001)` y `papel_nis2_tecnica`.

**9 hechos que ninguna de las tres declaraciones afirma, y los relojes que dejan
invisibles:**

| hecho | relojes que quedan sin ver | dónde encajaría |
|---|---|---|
| `riesgo_ia(alto_anexo_iii)` / `riesgo_ia(alto_anexo_i)` | 8 del AI Act (arts. 9.2, 18.1, 19.1, 26.6, 47.1, 72.2, 73, 73.6) | ya está en el `no_supone` de `es-fabricante-software`, con su motivo. Haría falta un perfil nuevo, no cambiar este |
| `riesgo_ia(uso_general)` | 1 (art. 52.1) | igual: perfil de proveedor de modelo fundacional |
| `papel_ia(representante_autorizado)` | 1 (art. 22.3.b) | perfil de representante autorizado en la UE |
| `papel_ia(importador)` | 1 (art. 23.5) | perfil de importador |
| `papel_rgpd(encargado)` | 1 (art. 33.2), y ver H4 | **el más barato y el más útil**: casi toda organización es encargada de algo. Cabría en los tres perfiles con `responsable_y_encargado` |
| `ambito(sector_privado_contratista)` | 4 del ENS, incluida la notificación al INCIBE-CERT del art. 33.7 recién escrita | los perfiles privados afirman `ambito(sector_privado)`, que **no** es el valor que las reglas del ENS esperan. Es un hecho a un carácter de distancia de funcionar |
| `designado(entidad_esencial_o_importante)` | 4 de `nis2-ue` | ya está en el `no_supone` de dos perfiles |
| `designado(operador_servicios_esenciales)` | 2 de `nis1-es` | ya está en el `no_supone` |
| `adopta(iso42001)` | 10 rituales | ya está en el `no_supone` de `es-servicios-digitales` |

**Y 4 fechas de ejemplo que ningún perfil siembra**, así que los relojes nuevos
salen *pendiente de hecho* en vez de con fecha: `ultima_reevaluacion_de_las_medidas`
(ENS art. 10.3), `ultima_revision_de_accesos`, `ultima_formacion_en_seguridad` y
`ultima_prueba_de_continuidad` (rituales de ISO/IEC 27001). Los tres últimos
alcanzan a `es-servicios-digitales`, que ya siembra cinco fechas de ejemplo para
los rituales que había; el primero, a `es-sector-publico`, que siembra otras
cinco.

**Salir *pendiente de hecho* no es un fallo**: es el estado correcto y el
calendario lo dice con su frase. Pero un perfil de arranque existe para enseñar
la forma de un calendario en diez segundos, y cuatro filas sin fecha de más lo
empeoran sin necesidad.

## Ficheros tocados fuera de la columna del frente, y por qué

Cuatro, y los cuatro son **contadores que un test deriva del árbol**, no
contenido. Cualquier frente que escriba corpus los mueve, así que la partición
no los podía dar a nadie:

| fichero | qué | antes | ahora |
|---|---|---|---|
| `README.md` | hitos / dorados / cobertura v1 | 179 / 521 / 80,5 % | 201 / 574 / 91,1 % |
| `paquetes/CORPUS.md` | hitos / dorados / sin número | 179 / 521 / 28 | 201 / 574 / 27 (ver H5) |
| `puente_piloto_test.go` | `ObligacionesQueDerivaElPiloto` | 26 | 30 |
| `entrevista_alcanza_al_motor_test.go` | `TotalDePreguntasDelCorpus` y `PreguntasQueNoLleganAlMotor` | 41 y 36 | 42 y 37 |

Los cuatro números los dice el propio test cuando falla, así que tras fundir los
frentes se recalculan corriendo `./comprobar.sh` y leyendo el rojo. No hay que
resolver el conflicto a mano: hay que volver a medir.
