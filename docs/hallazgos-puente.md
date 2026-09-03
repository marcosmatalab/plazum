# Hallazgos del frente del puente entrevista a motor

Tramo 1 de la campana de dos semanas. Este fichero es el cuaderno del frente: lo
que se midio, con que comando, y las dos paradas que impidieron escribir el
puente en los catorce paquetes que faltan.

Todas las cifras de aqui salen de ejecutar el motor y el linter DEL PRODUCTO
sobre el corpus instalado, con `aplicabilidad.ParsearRegla` y `corpus.Cargar`.
Ninguna se conto a mano y ninguna se estimo.

## 0. Resumen en cinco lineas

1. **8 de 72.** Un CISO de una SaaS espanola de 200 personas contesta todo lo que
   la entrevista de hoy sabe recibir y su calendario sale con **8 obligaciones**;
   las que de verdad le tocan son **72**. Y las 64 que faltan no salen con un
   aviso: salen como si no le tocaran. Seccion 3.
2. **El puente no se pudo escribir en ningun paquete mas.** El SEGUNDO paquete
   que lo declare pone rojo `puente_piloto_test.go`, que esta en la raiz del
   repositorio y fuera de la columna de este frente. Demostrado, con el rojo
   pegado en la seccion 1.
3. **Falta una forma del vocabulario**, y afecta a 19 hechos de 8 de los 15
   marcos de la v1: un si o un no que afirma `predicado(instancia, CONSTANTE)`.
   Hoy solo se puede expresar con un rodeo que ni el linter ni la pantalla
   pueden vigilar. Seccion 2.
4. **La pantalla de entrevista solo sabe mandar `si` y `no`.** Con eso, el techo
   de TODO el corpus son **16 obligaciones**, y el puente de `ens`, que es el
   unico escrito, solo puede encender **8** de las 30 que deriva su escenario.
   Seccion 3.
5. **El linter del puente tiene dos agujeros medidos**: rechaza un puente que
   cruza de paquete aunque el corpus comparte predicados a proposito, y acepta
   un puente cuyos VALORES no los mira ninguna regla. Seccion 4.

## 1. LA PARADA: el segundo puente no cabe sin tocar la raiz

`nucleo/corpus/puente.go` deja el bloque `hecho` opcional durante el piloto, y
eso es correcto. Lo que para el trabajo esta en la raiz:

`puente_piloto_test.go`, funcion `paqueteConPuente`, rama `default`:

    // NO ES UN FALLO, ES UNA DECISION QUE HAY QUE TOMAR. En cuanto haya un
    // segundo paquete con puente, esta medida deja de ser «el piloto» [...]
    // Se para aqui para que nadie lo amplie sin darse cuenta.

O sea que el diseno anticipo exactamente este encargo y puso ahi una parada
deliberada. La parada funciona.

### La sonda, con su rojo

Se declaro el puente en UN atributo mas, el mas trivial de todo el corpus:
`rgpd/organizacion_rgpd.papel_rgpd`, tipo enumerado, forma `con_valor`,
predicado `papel_rgpd`, que las reglas del propio paquete `rgpd` usan con
aridad 2. Es un puente correcto por construccion.

Compilacion comprobada aparte:

    $ go build ./...
    BUILD OK

Y el resultado:

    --- FAIL: TestElPuenteDeclaradoDerivaObligacionesDeVerdad (0.07s)
        puente_piloto_test.go:121: hay 2 paquetes con puente declarado
        ([urn:es:rd:2022:311 urn:eu:reg:2016:679]) y este test mide UN piloto con
        UN escenario. El piloto ha terminado: toca decidir si el puente pasa a
        obligatorio y darle escenario a cada paquete
    --- FAIL: TestUnNoDeLaEntrevistaNoAfirmaNadaEnElMotor (0.08s)
        puente_piloto_test.go:236: hay 2 paquetes con puente declarado
        ([urn:es:rd:2022:311 urn:eu:reg:2016:679]) y este test mide UN piloto con
        UN escenario. [...]
    FAIL
    FAIL	github.com/marcosmatalab/plazum	18.672s

La sonda se restauro con `cp` desde una copia, no con `git checkout`.

### Los tres ficheros de raiz que hay que mover, y que nadie tiene en su columna

| fichero | que hay que hacerle | por que |
|---|---|---|
| `puente_piloto_test.go` | dejar de medir «el piloto» y pasar a medir «los paquetes que declaran puente», con un escenario declarado por paquete | `paqueteConPuente` hace `t.Fatalf` con mas de uno, a proposito |
| `entrevista_alcanza_al_motor_test.go` | recalcular `TotalDePreguntasDelCorpus` (hoy 42) y `PreguntasQueNoLleganAlMotor` (hoy 37) | toda pregunta nueva mueve los dos, y son igualdad exacta |
| `conservacion_calendario_test.go` | actualizar el censo topado, si el trabajo anade obligaciones con reloj | igualdad exacta declarada |

Ninguno de los tres esta en ninguna columna de `.github/frontera.sh`, ni en la
matriz vieja ni en la del tramo 1 (commit `1d954e8`), que es la que rige y que da
a este frente `paquetes/`, `nucleo/corpus/`, `docs/censo-relojes.md` y este
fichero. Con la matriz buena delante, la parada no es de reparto: **estos tres
ficheros no son de nadie a proposito**, porque congelan medidas de todo el
producto. Los mueve quien integra.

## 2. LA FORMA QUE FALTA: el si o el no que afirma una constante

El vocabulario cerrado de `puente.go` tiene tres formas:

| forma | tipo del atributo | hecho que produce |
|---|---|---|
| `afirma_si` | booleano | `predicado(instancia)`, aridad 1 |
| `con_valor` | enumerado, texto, entero, fecha | `predicado(instancia, respuesta)`, aridad 2 |
| `no_llega_al_motor` | cualquiera | ninguno, y se dice por que |

Lo que el corpus necesita y no esta: **un atributo BOOLEANO cuyo si afirma
`predicado(instancia, CONSTANTE)`**, donde la constante la declara el paquete
junto al predicado. Un no sigue sin afirmar nada, como en `afirma_si`.

### Los 19 hechos que la piden, con su paquete

Sale de recorrer las reglas de los quince marcos con el parser del producto y
quedarse con los atomos de cuerpo cuyo predicado no deriva ninguna regla (los
hechos de base) y cuyo segundo argumento es una constante:

| # | hecho de base | paquete | pregunta natural |
|---|---|---|---|
| 1 | `adopta(E,"iso27001")` | iso27001 | tienes un SGSI segun esa norma? |
| 2 | `adopta(E,"iso42001")` | iso42001 | tienes sistema de gestion de IA? |
| 3 | `adopta(E,"soc2")` | soc2 | te examinas contra SOC 2? |
| 4 | `adopta(E,"pci-dss")` | pci-dss | tratas datos de tarjeta? |
| 5 | `adopta(E,"tisax")` | tisax | tienes etiqueta TISAX? |
| 6 | `designado(E,"delegado_de_proteccion_de_datos_designado")` | lopdgdd | tienes delegado? |
| 7 | `designado(E,"adherido_a_resolucion_extrajudicial_de_conflictos")` | lopdgdd | estas adherido? |
| 8 | `designado(E,"mantiene_sistema_de_informacion_crediticia")` | lopdgdd | mantienes uno? |
| 9 | `designado(E,"trata_imagenes_de_videovigilancia")` | lopdgdd | tienes camaras? |
| 10 | `designado(E,"entidad_esencial_o_importante")` | nis2-ue | te ha designado la autoridad? |
| 11 | `papel_nis2_registro(E,"entidad_del_art_27_1")` | nis2-ue | eres del art. 27.1? |
| 12 | `papel_nis2_dominios(E,"registro_o_servicio_de_registro")` | nis2-ue | eres registro de dominios? |
| 13 | `papel_nis2_tecnica(E,"entidad_pertinente")` | nis2-tecnica | eres una de las once clases del art. 1? |
| 14 | `designado(E,"operador_servicios_esenciales")` | nis1-es | te han designado operador? |
| 15 | `designado(E,"proveedor_servicios_digitales")` | nis1-es | encajas en el art. 10.5? |
| 16 | `designado(E,"entidad_financiera")` | dora | eres del art. 2 del reglamento? |
| 17 | `designado(E,"microempresa_dora")` | dora | eres microempresa? |
| 18 | `designado(E,"marco_simplificado_dora")` | dora | estas en el marco simplificado? |
| 19 | `designado(E,"entidad_financiera_marco_simplificado")` | dora | las dos cosas? |

**19 hechos, 8 paquetes de los 15.** Los cuatro de `lopdgdd` y los cuatro de
`dora` cuelgan todos del MISMO predicado `designado` y son INDEPENDIENTES entre
si: una entidad puede tener delegado y camaras a la vez. No hay ninguna forma
hoy de que un solo atributo produzca dos hechos, ni de que cuatro atributos
booleanos produzcan cuatro hechos sobre el mismo predicado con constante.

### Por que no se ha usado el rodeo, aunque existe y esta comprobado

El rodeo es un atributo `enumerado` con UN SOLO valor y forma `con_valor`. El
linter lo acepta, comprobado:

    SONDA 3 (enumerado de un solo valor) ACEPTADO: el rodeo existe.

No se ha usado por tres razones, y la tercera es la que decide:

1. Genera un desplegable de una sola opcion, que es una pregunta que no se puede
   contestar que no.
2. Los cuatro `designado` de `dora` y los cuatro de `lopdgdd` saldrian como
   cuatro desplegables de una opcion cada uno sobre el mismo predicado. Un
   formulario asi no lo entiende nadie.
3. **La pregunta obligatoria de la pasada 2 de este frente sale que SI.** Un
   desplegable de una sola opcion que la superficie envie por defecto afirma
   `designado(E,"entidad_financiera")` en una organizacion que no es entidad
   financiera, y eso enciende `dora.art19.notificacion_de_incidente_grave_tic`,
   que es `notificatoria` y cuyo entregable sale ante el supervisor. **Encender
   de mas ahi provoca una actuacion indebida y eso no se deshace.** Medido:
   `designado(s,entidad_financiera)` por si solo enciende **28** obligaciones de
   DORA. No se mete en el corpus una codificacion cuya seguridad depende de que
   la superficie acierte, sobre todo cuando la superficie de hoy ni siquiera
   sabe mandar valores (seccion 3).

Se declara aqui y no se inventa, que es lo que pedia el encargo: cambiar el
esquema afecta a los quince paquetes y lo decide quien integra.

### Lo que si cabe con las tres formas de hoy

De los hechos de base de los quince marcos, con las formas actuales se pueden
declarar:

- `con_valor` sobre enumerado con varios valores: `papel_rgpd` (rgpd),
  `papel_ia` y `riesgo_ia` (ai-act), `papel_cra` (cra), `ambito` y `categoria` y
  los cinco `nivel_*` (ens, ya escritos).
- `afirma_si` sobre booleano: `trata_datos_personales`, `preexistente_al_ens`,
  `servicios_externalizados`, `usa_servicios_en_la_nube` (ens, ya escritos) y
  `canal_de_denuncias_obligatorio` (ley2-2023).
- `no_llega_al_motor`: todos los atributos de fecha y de alcance de `iso27001` e
  `iso42001`, y las clasificaciones de incidente de `dora`, `ai-act`, `nis1-es`
  y `nis2-ue`.

Con eso, **6 de los 14 paquetes pendientes se podrian declarar enteros hoy**
(rgpd, ai-act, cra, ley2-2023, y los callejones de iso42001 y nis1-es parciales),
y **8 no**. Los 6 siguen bloqueados por la parada de la seccion 1.

## 3. EL TECHO DEL SI O NO, que es la cifra que decide si esto sirve

`superficies/pantallas` solo lee dos parametros, `ParamSi` y `ParamNo`. No hay
ningun camino por el que una respuesta con VALOR llegue al motor.

Los unicos hechos que un si puede producir son los de aridad 1. En todo el
corpus instalado hay **siete**:

    canal_de_denuncias_obligatorio
    demo.trata_datos_personales
    expide_certificados_cualificados
    preexistente_al_ens
    servicios_externalizados
    trata_datos_personales
    usa_servicios_en_la_nube

Afirmandolos TODOS a la vez sobre el mismo sujeto, que es un escenario imposible
y por tanto una cota superior generosa:

    TECHO del si/no: TODOS los unarios del corpus afirmados a la vez: 16 obligaciones

**Dieciseis.** Y el puente de `ens`, que es el unico escrito y el que el piloto
mide en 30 obligaciones, con lo que la pantalla sabe mandar hoy da:

    el puente de ens, SOLO lo que la pantalla sabe mandar (sus 4 afirma_si): 8 obligaciones

O sea que de las 30 del piloto, **22 dependen de valores que la entrevista no
sabe preguntar**, empezando por `ambito` y `categoria`, que son las dos que
mandan en el ENS entero.

Esto no lo arregla el puente: lo arregla la pantalla. Va a `superficies/pantallas`,
que no es de este frente.

### La pasada contra el comprador, con el numero

Un CISO de una SaaS espanola de 200 personas abre esto y contesta. De las **42**
preguntas del corpus, las que la pantalla de hoy puede convertir en un hecho son
**cinco**, y una de las cinco es del paquete de demostracion:

    ens.q.externalizacion        Hay servicios externalizados dentro del alcance del sistema?
    ens.q.datos_personales       El sistema trata datos personales?
    iso27001.q.datos_personales  Se tratan datos personales dentro del alcance?
    demo.q.datos_personales      Trata la organizacion datos personales?
    ens.q.nube                   Se usan servicios en la nube dentro del alcance del sistema?

Contesta que si a las tres cosas distintas que preguntan (datos personales, nube
y externalizacion) y su calendario sale con **8 obligaciones**, siete del RGPD y
una de la ley organica.

Lo que de verdad le toca a esa organizacion, si la entrevista supiera preguntar
las cuatro cosas que le faltan (que es responsable del tratamiento, que tiene
cincuenta o mas trabajadores, que tiene SGSI y que es una de las once clases del
art. 1 del Reglamento de Ejecucion 2024/2690), son **72 obligaciones**.

**8 de 72.** Y lo peor no es el 8: es que las 64 que faltan no salen con un aviso
de «esto no te lo hemos preguntado», salen **como si no le tocaran**. Un CISO que
lea ese calendario concluye que la Ley 2/2023 no le alcanza, y le alcanza desde
los cincuenta empleados. Es el mismo patron que la regla de no acusar en falso,
por el otro lado: aqui no se acusa de mas, se ABSUELVE de mas, y con la misma
cara de dato.

Las cuatro que faltan son, exactamente, las que necesitan la forma que no existe
(las tres de constante) o un valor que la pantalla no sabe mandar (el papel del
RGPD). No es casualidad: es la misma parada contada desde la silla del que paga.

## 4. LOS DOS AGUJEROS DEL LINTER DEL PUENTE, medidos sobre copia del corpus

Las dos sondas corren sobre una COPIA de `paquetes/` en un directorio temporal.
El arbol de trabajo no se toco.

### 4.1 Rechaza el puente que cruza de paquete, y el corpus cruza a proposito

`validarPuente` compara el predicado contra `p.aridadesDeSusReglas()`, o sea
solo contra las reglas del PROPIO paquete. Pero el corpus comparte predicados
entre paquetes deliberadamente: el propio comentario de `puente_piloto_test.go`
lo dice, «los predicados se comparten entre paquetes».

Sonda: `iso27001/sgsi.trata_datos_personales` con forma `afirma_si` y predicado
`trata_datos_personales`, que usan con aridad 1 las reglas de `rgpd`, `lopdgdd`,
`ley2-2023` y `ens`.

    SONDA 1 (puente que cruza de paquete) RECHAZADO:
     iso27001: 1 fallos de linter, el primero: predicado que ninguna regla usa:
     urn:iso-iec:27001:2022/sgsi.trata_datos_personales afirma
     "trata_datos_personales" y ninguna regla de este paquete usa ese predicado.
     [...] Arreglo: o se escribe la regla que lo lee, o el atributo declara
     "no_llega_al_motor" con su motivo. Predicados que si usa: [adopta aplica en_ambito]

**Y el arreglo que propone el error es una mentira.** Declarar
`no_llega_al_motor` ahi afirmaria que esa respuesta no alimenta ninguna regla,
y alimenta ocho: `trata_datos_personales(s)` enciende 7 obligaciones de `rgpd` y
1 de `lopdgdd`, medido. El propio `puente.go` dice que «un puente declarado y
falso es peor que ninguno». El linter esta empujando a escribir uno.

**Cardinal hoy: 1 atributo** (`iso27001/sgsi.trata_datos_personales`). Crece en
cuanto se declare el puente de los paquetes que no tienen entidades propias.

### 4.2 Aceptaba un puente cuyos valores no mira ninguna regla. CERRADO

`validarPuente` comprueba el NOMBRE del predicado y la ARIDAD. No comprueba que
los `valores` del enumerado esten entre las constantes que las reglas prueban.

Sonda: `iso27001/sgsi.estado_certificacion` (valores `no_certificado`,
`en_certificacion`, `certificado`) con forma `con_valor` y predicado `adopta`.
Las reglas de `iso27001` solo miran `adopta(E,"iso27001")`.

    SONDA 2 (valores que ninguna regla mira) ACEPTADO POR EL LINTER.
      El paquete declara que esa respuesta llega al motor y no llega:
      produce adopta(sgsi,"certificado") y las reglas miran adopta(E,"iso27001").

Es exactamente el fallo que el bloque `hecho` existe para cerrar, un escalon mas
abajo: la respuesta se recoge, se pinta, produce un hecho, el hecho no casa con
nada, y nadie se entera. La direccion que falta del emparejamiento (invariante 7)
no es la del predicado, que si se recorre en las dos: es la del VALOR, que no se
recorre en ninguna.

**ARREGLADO en este tramo**, una vez que la matriz del tramo 1 (`1d954e8`) dio
`nucleo/corpus/` a este frente. En la forma `con_valor` sobre un atributo
`Enumerado`, se exige que **al menos uno** de sus `valores` aparezca como
constante en esa posicion en alguna regla del paquete, y el error nombra lo que
las reglas si miran. No se exigen todos: un enumerado puede tener valores que
apagan en vez de encender.

Y **la variable manda sobre la constante**: si alguna regla usa ese hueco con
una variable, acepta cualquier valor y no hay nada que exigir. Sin esa rama la
puerta rechazaria corpus correcto, que es como acaba apagada.

Se vio fallar sobre la sonda, que es dato real y no mutacion:

    antes: SONDA 2 (valores que ninguna regla mira) ACEPTADO POR EL LINTER.
    ahora: SONDA 2 RECHAZADO: valores que ninguna regla mira:
      urn:iso-iec:27001:2022/sgsi.estado_certificacion afirma "adopta" con sus
      valores [no_certificado en_certificacion certificado], y ninguna regla de
      este paquete prueba ninguno de ellos en ese hueco: las reglas solo miran
      [iso27001].

Y la sonda 3, que es el rodeo legitimo, sigue aceptada: la puerta no se lo llevo
por delante.

**Esta puerta nacio VERDE sobre el corpus entero y se dice**, que es la regla de
la casa. `corpus.Cargar("paquetes")` no rechaza nada, porque hoy solo un paquete
declara el puente. No es que vigile poco: es que llega antes de que se escriban
los catorce puentes que faltan. Queda escrito en la cabecera del test para que
nadie lea ese verde como una medida.

## 5. LA TABLA POR PAQUETE: cuantas obligaciones enciende cada respuesta

Medido con el motor del producto. «Todas sus respuestas positivas» significa
afirmar a la vez todos los hechos de base de ese paquete que no aparecen solo en
posicion negada, sobre un unico sujeto. Es una cota SUPERIOR, porque incluye
combinaciones que en la realidad se excluyen (`ambito` no puede ser a la vez
sector publico y sector privado contratista).

| paquete | obligaciones | preguntas | encendidas | % | huerfanas |
|---|---|---|---|---|---|
| iso27001 | 132 | 8 | 9 | 7 | 0 |
| rgpd | 9 | 1 | 8 | 89 | 1 |
| lopdgdd | 8 | 0 | 7 | 88 | 0 |
| ens | 133 | 17 | 21 | 16 | 12 |
| nis2-ue | 12 | 1 | 12 | 100 | 1 |
| nis2-tecnica | 48 | 0 | 48 | 100 | 0 |
| nis1-es | 4 | 1 | 2 | 50 | 1 |
| dora | 30 | 1 | 26 | 87 | 1 |
| ai-act | 25 | 3 | 25 | 100 | 3 |
| iso42001 | 51 | 3 | 10 | 20 | 3 |
| cra | 24 | 0 | 24 | 100 | 0 |
| ley2-2023 | 7 | 0 | 7 | 100 | 0 |
| soc2 | 5 | 0 | 5 | 100 | 0 |
| pci-dss | 7 | 0 | 7 | 100 | 0 |
| tisax | 5 | 0 | 5 | 100 | 0 |

Y respuesta a respuesta, lo que enciende cada una por si sola. «Ajenas» son
obligaciones de OTRO paquete, que es el efecto que el piloto vio y que un
recuento por paquete no habria encontrado:

| respuesta | propias | ajenas |
|---|---|---|
| `adopta(s,"iso27001")` | 9 | 0 |
| `adopta(s,"iso42001")` | 10 | 0 |
| `adopta(s,"soc2")` | 5 | 0 |
| `adopta(s,"pci-dss")` | 7 | 0 |
| `adopta(s,"tisax")` | 5 | 0 |
| `trata_datos_personales(s)` | 7 en rgpd | 1 en lopdgdd |
| `papel_rgpd(s,"encargado")` | 1 | 0 |
| `ambito(s,"sector_publico")` | 9 en ens | 1 en eni |
| `ambito(s,"sector_privado_contratista")` | 10 en ens | 0 |
| `designado(s,"entidad_esencial_o_importante")` | 9 | 0 |
| `papel_nis2_dominios(s,"registro_o_servicio_de_registro")` | 2 | 0 |
| `papel_nis2_registro(s,"entidad_del_art_27_1")` | 1 | 0 |
| `papel_nis2_tecnica(s,"entidad_pertinente")` | **48** | 0 |
| `designado(s,"operador_servicios_esenciales")` | 2 | 0 |
| `designado(s,"proveedor_servicios_digitales")` | 2 | 0 |
| `designado(s,"entidad_financiera")` | **28** | 0 |
| `designado(s,"entidad_financiera_marco_simplificado")` | 2 | 0 |
| `papel_ia(s,"proveedor")` | 4 | 0 |
| `papel_ia(s,"responsable_del_despliegue")` | 3 | 0 |
| `papel_ia(s,"importador")` | 1 | 0 |
| `papel_ia(s,"representante_autorizado")` | 1 | 0 |
| `papel_cra(s,"fabricante")` | 16 | 0 |
| `papel_cra(s,"importador")` | 4 | 0 |
| `papel_cra(s,"distribuidor")` | 3 | 0 |
| `papel_cra(s,"representante_autorizado")` | 1 | 0 |
| `canal_de_denuncias_obligatorio(s)` | 6 | 0 |

Tres lecturas que hay que decir en voz alta:

1. **Dos respuestas cargan con 76 de las obligaciones de la v1.**
   `papel_nis2_tecnica` con 48 y `designado(entidad_financiera)` con 28. Las dos
   estan en el grupo de las que **no se pueden declarar con las formas de hoy**.
2. **`riesgo_ia(s,X)` enciende 0 por si solo, en los tres valores.** No es un
   fallo: las reglas del AI Act piden papel Y riesgo juntos, que es lo que dice
   el reglamento. Sale contado para que nadie lea el cero como una regla rota.
3. **`designado(s,"microempresa_dora")` y `designado(s,"marco_simplificado_dora")`
   encienden 0 y APAGAN.** Afirmando solo `designado(entidad_financiera)` salen
   28 obligaciones; afirmando los cuatro hechos de DORA a la vez salen **26**.
   Son las dos unicas respuestas del corpus de la v1 cuyo efecto es restar, y
   restan bien: los arts. 6.5, 8.7, 24.6 y 26.1 excluyen a la microempresa.

## 6. LAS 23 PREGUNTAS QUE NADIE REQUIERE, repartidas

`corpus.PreguntasQueNadieRequiere` da **23** sobre el corpus entero. Repartidas
por paquete y por motivo, que es lo que el numero agregado escondia:

### Motivo A, «solo falta el campo `preguntas`»: 15

Su atributo SI lo usa alguna regla, o sea que la respuesta llegaria al motor.
Lo que falta es que las obligaciones que dependen de ella la nombren en su campo
`preguntas`, que es lo unico que la pantalla de alcance evalua.

| paquete | preguntas | cuales |
|---|---|---|
| ens | 12 | las seis de `informacion.*` y las seis de `servicio.*` |
| ai-act | 2 | `aiact.q.papel`, `aiact.q.riesgo` |
| rgpd | 1 | `rgpd.q.papel` |

15 en el motivo A y 8 en el motivo B suman los 23 que da
`corpus.PreguntasQueNadieRequiere`. Las dos tablas enumeran las 23 una a una: no
hay ninguna que no aparezca.

Estas 15 son **deuda barata**: no hay que leer ninguna norma, hay que anadir el
id de la pregunta al campo `preguntas` de las obligaciones que ya dependen de
ese predicado en sus reglas. Se puede derivar mecanicamente de las reglas.

### Motivo B, «falta la regla, o sobra la pregunta»: 8

Ninguna regla usa su atributo Y ninguna obligacion la nombra. Cada una exige
leer la norma para decidir cual de las dos es.

| paquete | pregunta | atributo | veredicto |
|---|---|---|---|
| ai-act | `aiact.q.clasificacion` | `incidente_grave.clasificacion` | **ninguna de las dos: es un hecho de INSTANCIA que alimenta el reloj**, no la aplicabilidad. Ver descarte 1 |
| dora | `dora.q.clasificacion` | `incidente_tic.clasificacion` | **igual, y ademas la cita estaba cruzada**. Ver descarte 2 y la seccion 8 |
| nis1-es | `nis1es.q.operador` | `incidente.nivel` | **la pregunta y el atributo hablaban de cosas distintas. CORREGIDO**, seccion 8 |
| nis2-ue | `nis2.q.tipo` | `entidad_nis2.tipo_de_entidad` | **sin resolver**: las reglas usan `designado(E,"entidad_esencial_o_importante")` y la pregunta fija `tipo_de_entidad`, que ninguna regla mira y que ademas mezcla dos ejes (esencial/importante y prestador de servicios de confianza) en un enumerado de tres valores |
| iso42001 | `iso42001.q.alcance` | `aims.alcance_declarado` | **sobra como condicion**: el alcance documenta, no decide aplicabilidad. Ver descarte 3 |
| iso42001 | `iso42001.q.desarrolla` | `aims.desarrolla_sistemas_ia` | **sin resolver** |
| iso42001 | `iso42001.q.impacto` | `aims.fecha_ultima_evaluacion_impacto_ia` | **sobra como condicion**: hecho fechado que alimenta el reloj. Mismo caso que las seis fechas de `ens`, ya declaradas `no_llega_al_motor`. Ver descarte 4 |
| mdr (fuera de la v1) | `mdr.q.gravedad` | `incidente_producto_sanitario.gravedad` | igual que las de incidente, y el paquete esta fuera del recorte de D-19 |

**Cardinal de lo que queda sin resolver dentro de la v1: 2** (`nis2.q.tipo` y
`iso42001.q.desarrolla`). Las dos exigen escribir regla nueva, y escribir regla
mueve el recuento del piloto, que es la parada de la seccion 1.

## 7. LOS DESCARTES, con su articulo

Descartar es un resultado y se cuenta. **Cuatro puntos que parecian condicion de
aplicabilidad y no lo son.** Los cuatro tienen la misma forma: son datos de una
INSTANCIA concreta (un incidente, una fecha) y no propiedades del sujeto
obligado, asi que alimentan el reloj (`ventana.Hechos`) y no la regla. Meterlos
en la aplicabilidad haria que la obligacion dependiera de que ya hubiera pasado
algo, que es justo lo que el paquete `ens` ya rechaza por escrito en el `porque`
de sus seis fechas.

1. **Reglamento (UE) 2024/1689, art. 73**, la clasificacion del incidente grave.
   Verificado contra la instantanea local `corpus-vigilancia/ue-32024r1689` (ELI
   `reg/2024/1689/oj`, obtenida el 2026-09-03T20:07:22Z). El **73.1** dice a quien
   alcanza: *«Los proveedores de sistemas de IA de alto riesgo introducidos en el
   mercado de la Union notificaran cualquier incidente grave...»*. Los apartados
   **73.2, 73.3 y 73.4** no cambian el obligado: graduan el PLAZO segun el tipo de
   incidente, quince dias en el caso general, **dos** en la infraccion
   generalizada o el incidente del art. 3, punto 49, letra b), y **diez** en caso
   de fallecimiento. O sea que la clasificacion es un dato del incidente que
   escoge el limite, no una condicion del sujeto.
2. **Reglamento (UE) 2022/2554, art. 18**, «Clasificacion de los incidentes
   relacionados con las TIC y las ciberamenazas». Verificado contra
   `corpus-vigilancia/ue-32022r2554` (ELI `reg/2022/2554/oj`, instantanea obtenida
   el 2026-09-03T20:07:05Z). El art. 18.1 obliga a la entidad financiera a
   clasificar; el art. 19.1 obliga a notificar los graves. **El deber de
   notificar alcanza a toda entidad financiera**, y lo que la clasificacion
   decide es si un incidente concreto abre el reloj. La regla actual
   (`designado(E,"entidad_financiera")`) **no es mas ancha que su articulo**: es
   exactamente lo que dice el 19.1.
3. **ISO/IEC 42001, el alcance declarado del sistema de gestion.** Es un campo
   documental del propio sistema de gestion, no un umbral. Sin identificador de
   apartado aqui, porque el paquete es de estrato referencial y este documento no
   puede transcribirlo (invariante 3).
4. **Las fechas de ultima ejecucion**, en `iso42001` como en `ens`. Un hecho
   fechado del obligado dice CUANDO se hizo algo. Si entrara en la aplicabilidad,
   la obligacion de auditar dependeria de haber auditado ya, que es un ciclo.

**Y un quinto descarte, de otra familia:** el `estado_certificacion` de
`iso27001` y de `iso42001` no se ha enchufado a `adopta`, aunque el tipo y la
aridad casan y el linter lo aceptaria (sonda 2). Adoptar una norma y estar
certificado en ella son dos cosas distintas: se puede tener SGSI sin certificar,
y son los rituales del sistema de gestion los que producen las fechas, no el
certificado.

## 8. DOS FALLOS DEL CORPUS ENCONTRADOS AL ESCRIBIR EL PUENTE, y arreglados

Los dos salieron de la misma pregunta del encargo: por que campo casa cada
emparejamiento. Ninguno daba error en ningun sitio.

### 8.1 `nis1es.q.operador` preguntaba una cosa y escribia otra

El paquete `nis1-es` tenia:

    "texto":    "Esta la entidad designada como operador de servicios esenciales?"
    "entidad":  "incidente"
    "atributo": "nivel"

La pregunta es sobre la ORGANIZACION y el atributo que fijaba era el nivel de un
INCIDENTE, en otra entidad. Y el propio `ayuda` del atributo lo decia con todas
las letras: *«no es un dato de la organizacion sino de cada incidente»*. Los dos
textos se contradecian dentro del mismo fichero.

No lo caza ningun linter porque la unica comprobacion es que la entidad y el
atributo EXISTAN, y existian los dos. Es el emparejamiento por un campo que no
significa lo que se cree.

**Demostrado con dos mutaciones, y la segunda es la que importa.** Con el
atributo apuntando a un nombre que NO existe, la puerta salta:

    --- FAIL: TestTodosLosPaquetesPublicadosPasanElLinter
        paquetes_test.go:191: el corpus publicado no pasa el linter: nis1-es: 1
        fallos de linter, el primero: pregunta nis1es.q.operador apunta al
        atributo entidad_nis1.designado_que_no_existe, que no existe

Con el atributo apuntando al nombre EQUIVOCADO PERO EXISTENTE, que es el fallo
que estaba en el arbol, `go test ./...` sale entero en verde. **La guarda existe
para la existencia y no existe para el significado**, y la que hace falta es la
segunda: nadie escribe por error el nombre de un atributo que no existe, y en
cambio apuntar al atributo de otra entidad se hace solo.

Que pasaba en la practica: contestar «si soy operador de servicios esenciales»
habria escrito un nivel de incidente. Y las reglas del paquete miran
`designado(E,"operador_servicios_esenciales")`, que nadie podia afirmar.

**Arreglo**: entidad nueva `entidad_nis1` con atributo `designado` (enumerado,
valores `operador_servicios_esenciales` y `proveedor_servicios_digitales`), y la
pregunta repuntada a ella. `incidente.nivel` se queda donde estaba, que es donde
le toca.

Verificacion (invariante 10), contra la instantanea local
`corpus-vigilancia/es-boe-a-2021-1192` (BOE-A-2021-1192, ELI
`boe.es/eli/es/rd/2021/01/26/43`, obtenida el 2026-08-26T21:33:33Z):

- art. 9.1: *«Los operadores de servicios esenciales notificaran a la autoridad
  competente respectiva, a traves del CSIRT de referencia, los incidentes que
  puedan tener efectos perturbadores significativos...»*
- art. 10.5: *«Lo establecido en los apartados anteriores sera de aplicacion a
  los proveedores de servicios digitales...»*

Las dos figuras del real decreto, y las dos son designaciones.

### 8.2 La cita de `dora.q.clasificacion` decia un articulo y enlazaba otro

    "cita": "Reglamento (UE) 2022/2554, art. 18. https://...2554/oj#art_19"

El texto decia 18 y el ancla decia 19. Verificado contra
`corpus-vigilancia/ue-32022r2554`: el art. 18 se titula «Clasificacion de los
incidentes relacionados con las TIC y las ciberamenazas» y el art. 19
«Notificacion de los incidentes graves...». La pregunta es sobre la
clasificacion, asi que el numero era el bueno y el ancla la mala. **Arreglada el
ancla, no el numero.**

Es la clase de fallo que sobrevive a cualquier revision porque las dos mitades
son verosimiles por separado y solo se ve poniendolas juntas.

## 9. LAS NOTIFICATORIAS: a quien alcanzan y de donde sale que no alcanzan a los demas

Es la unica clase cuyo entregable sale de la organizacion, asi que la pregunta
se contesta una a una. En los 15 marcos de la v1 hay **55** obligaciones
`notificatoria`. La pregunta que importaba en este frente es la del encargo:
**puede una traduccion del puente encender una notificatoria a quien no le
toca?**

**Respuesta medida: hoy no, y hay margen en la direccion segura.** Las 52 que
tienen regla la tienen anclada al sujeto que nombra su articulo, y ninguna de
las anclas es mas ancha que su apartado. Las tres que faltan no encienden de
mas: **no encienden nunca**, que es el fallo contrario.

### Las tres notificatorias que ninguna regla puede derivar

| obligacion | articulo | por que nadie la alcanza |
|---|---|---|
| `ens.art33.2.notificacion_al_ccn` | ENS art. 33.2 | ninguna regla del corpus la nombra. Su hermana `ens.its_incidentes.notificacion_al_ccn` si tiene regla (`ambito(S,"sector_publico")`) |
| `iso27001.a.6.8` | A.6.8 | ninguna regla la nombra, y el paquete es referencial: se adopta voluntariamente |
| `nis1es.art9.notificacion_de_incidentes` | RD 43/2021 art. 9.1 | ninguna regla la nombra. Su hermana del parrafo segundo, `nis1es.art9_1p2`, si la tiene (`en_ambito(E)`) |

Comprobado que la direccion es la segura y no la peligrosa: `aplicablesDe`
(`cmd/plazum/motor.go`) consulta `aplica(O, sujeto)` y **no tiene ninguna rama de
por defecto**. Una obligacion sin regla no deriva ningun `aplica`, asi que no
sale en el calendario ni en el alcance de nadie. Es la subfamilia
«alcanzabilidad, no existencia» que `nucleo/corpus/roles.go` ya nombra: la
obligacion existe, se ve en el catalogo, y nadie puede tenerla derivada.

**No se arreglan en este tramo** y se dice por que: escribir la regla que falta
cambia lo que deriva el motor, y eso mueve `ObligacionesQueDerivaElPiloto`, que
es la parada de la seccion 1. Las tres quedan aqui contadas y con nombre.

### El cardinal grande que salio por el camino

Contando por el ID de la obligacion contra las cabezas de las reglas de su
propio paquete, y despues de comprobar que **ningun paquete del corpus tiene una
regla `aplica` generica** (todas nombran un ID literal en la cabeza):

    obligaciones de los 15 marcos sin regla de aplicabilidad: 274
      de esas, con reloj declarado:                             0
      de esas, de clase notificatoria:                           3

**El cero de la fila del medio es la buena noticia y hay que decirla:** toda
obligacion con reloj de los quince marcos tiene regla, asi que la ley de
conservacion del calendario no esta afectada por nada de esto. Las 274 son
catalogo de controles sin reloj, y **107 de ellas son el anexo II del ENS**, que
hoy no lo deriva la categoria. Contado: la maquinaria de `categoria(S,...)`
existe entera (`nivel_dimension`, `nivel_requerido`, `nivel_max` y las tres
reglas de categoria) y de ella cuelgan **5 obligaciones**, todas de conformidad
(la auditoria ordinaria, la autoevaluacion basica y su publicidad, y la
certificacion de media y alta y su publicidad). Del anexo II tienen regla
**5 controles**, y ninguno por categoria: `mp.info.1` por datos personales,
`op.ext.1`, `op.ext.2` y `op.ext.3` por externalizacion y `op.nub.1` por nube.
Es un hueco de catalogo, no de calendario, y no se ha tocado.

## 10. Lo que NO se hizo, con su cardinal

Todo lo de esta tabla comparte la misma causa cuando dice «mueve el piloto»: el
recuento de `ObligacionesQueDerivaElPiloto` es igualdad exacta y vive en la raiz,
fuera de la columna de este frente. Es la parada de la seccion 1.

| cosa | cardinal | por que |
|---|---|---|
| declarar el puente en los paquetes que faltan | **14 de 15** | el segundo puente pone rojo un test de la raiz. Demostrado |
| escribir la forma que falta | **1 forma, 19 hechos, 8 paquetes** | el encargo pide decirlo antes de inventarlo, y cambiar el esquema afecta a los quince |
| cerrar el motivo A de las huerfanas | **15 preguntas** | anadir `preguntas` a una obligacion cambia lo que la pantalla de alcance deriva, y no se puede medir sin mover el piloto |
| resolver el motivo B | **2 dentro de la v1** | exigen escribir regla nueva |
| escribir la regla de las tres notificatorias inalcanzables | **3** | escribir la regla mueve el piloto |
| cerrar el agujero 4.1 del linter, el del puente que cruza de paquete | **1** | no es un arreglo mecanico: obliga a decidir si el emparejamiento del puente se comprueba contra el paquete o contra el corpus entero, y eso cambia la promesa escrita en `puente.go` («las dos puntas viven DENTRO del mismo paquete firmado»). Se decide, no se parchea. El 4.2 si esta cerrado |
| derivar el anexo II del ENS por categoria | **107 controles sin regla** | ninguno tiene reloj, asi que no afecta al calendario. Es hueco de catalogo y es una decision de producto |
| recontar la fila del censo de `ai-act` por el Reglamento 2026/1744 | **1 fila, de 26 a 29 como minimo** | exige fuente primaria y no era el mandato de este tramo |
| recontar `cra` y `nis2-ue` contra fuente primaria | **2 filas** | son las dos que estan en `null` en `marcos-v1.json` y las que impiden dar un total exacto del censo |

## 11. Errores propios de este tramo

1. **Escribi 16 donde la medida decia 15** en el reparto de las preguntas
   huerfanas por motivo, y encima anadi una nota para justificar el descuadre en
   vez de volver a mirar el numero. Corregido antes de commitear. El numero
   estaba en la salida del programa desde el principio.
2. **Escribi «verificado contra la instantanea local» del art. 73 del AI Act
   antes de haberla abierto.** Es la afirmacion acompanada en su forma pura: la
   frase tenia la forma de lo verificable y no lo era. Se abrio la instantanea,
   el descarte resulto correcto, y la frase ahora lleva los tres plazos (quince,
   dos y diez dias) que solo se pueden escribir habiendola leido.
3. **Di por hecho que una obligacion sin regla «alcanza a todo el mundo»** y lo
   escribi asi en la salida de un script. Es exactamente al reves: `aplicablesDe`
   no tiene rama de por defecto y una obligacion sin regla no alcanza a nadie.
   Las dos cosas son fallos, pero una acusa en falso y la otra calla, y
   confundirlas en un informe sobre notificatorias habria sido el error mas caro
   de este frente.
4. **Conte 274 obligaciones sin regla con una expresion regular antes de
   comprobar si habia reglas `aplica` genericas.** Si las hubiera habido, el 274
   habria sido basura con forma de dato. Se comprobo despues y no las hay, asi
   que el numero vale, pero el orden estaba mal: la comprobacion iba primero.
5. **Escribi un control negativo que no recorria la rama que decia recorrer, y
   ademas le puse un comentario afirmando que sin esa rama la puerta habria
   rechazado once atributos del ENS.** Las dos cosas eran falsas y las cazo la
   mutacion: al borrar `!valoresDelPuente[pred]` la suite se quedo EN VERDE,
   porque el fixture usaba el hueco solo con variable y entonces el conjunto de
   constantes queda vacio y la guarda anterior ya no deja entrar. Es M47 otra
   vez, y en el mismo commit en el que lo estaba citando: un descargo que
   ninguna entrada alcanza es un descargo que no existe. El fixture ahora usa el
   hueco de las dos maneras y la mutacion lo pone rojo.
6. **Y una del proceso, no del contenido:** la primera version de la mutacion de
   esa rama no compilaba (`declared and not used`), y si no llego a comprobar el
   build aparte lo habria leido como «la mutacion no la caza nadie». Es la
   trampa que CLAUDE.md nombra, y me la comi igual; solo que la comprobacion
   separada estaba puesta.
