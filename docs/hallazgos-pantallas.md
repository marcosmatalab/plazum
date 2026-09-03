# Hallazgos de las pantallas (puerta D11-a, cero formaciones)

> **La puerta D11-a dice**: si una pantalla necesita explicación externa para
> llegar al valor, es hallazgo con prioridad. El producto se explica solo, paso
> a paso.
>
> Este fichero es la mitad **cualitativa** de la medición. La cuantitativa vive
> en `ttfv_camino_test.go`, que levanta el binario de verdad, recorre
> `camino.Canonico()` y saca el número. Los dos se leen juntos: el número dice
> cuánto cuesta, este fichero dice por qué.

**Cómo se hizo la medición.** Se compiló el binario, se arrancó
`plazum serve --direccion 127.0.0.1:PUERTO --corpus paquetes` en un directorio
vacío y se recorrieron los seis pasos del camino guiado por HTTP, sin sesión y
sin haber leído documentación, que es lo que hace quien se descarga el producto.
Fecha de la medición: **03-09-2026**. Todo lo de abajo se puede reproducir
ejecutando la puerta.

---

## El número, con su desglose

    MODELO: TTFV = T_maquina + T_humano
            lectura de pantalla 45 s, respuesta de pregunta 20 s, orden de terminal 90 s

    T_maquina   1,1 s   = binario 0,9 s + arranque 0,2 s + peticiones 0,01 s
    T_humano   18m55s
    TOTAL      18m56s   sobre un presupuesto de 15m0s

    paso alcance      /alcance       200   41 preguntas   0 ordenes   14m25s
    paso calendario   /calendario/   200    0 preguntas   2 ordenes    3m45s
    paso derivacion   /controles     200    0 preguntas   0 ordenes      45s
    paso acta         /acta/         401                                  --
    paso uar          /uar/          401                                  --
    paso escalado     /escalado/     401                                  --

**Dos cosas hay que leer a la vez, y la segunda es peor que la primera:**

1. El TTFV del tramo recorrible es **18m56s**, o sea **casi cuatro minutos por
   encima** del presupuesto de la casilla D11-e.
2. Y ese número sale de **tres de los seis pasos**. Los otros tres no se
   recorren: no hay TTFV del camino completo porque **el camino completo no se
   puede recorrer**.

El modelo de coste no se ha tocado para que salga ningún número. Las tres
constantes son estimaciones declaradas, elegidas conservadoras por arriba, y
están sueltas y nombradas en el fichero para que cualquiera recuente cambiando
el valor. Los conteos **no son estimaciones**: los pasos salen de
`camino.Canonico()`, las preguntas se cuentan de la página que devuelve
`/alcance`, y cada orden de terminal declarada se comprueba contra el `<main>`
de su pantalla, así que no se puede inflar ni desinflar respecto de lo que el
producto dice.

---

## P0-1. Tres de los seis pasos del camino guiado no existen para quien descarga el binario

**Qué pasa.** `/acta/`, `/uar/` y `/escalado/` contestan **401** en una
instalación recién hecha, y **no hay forma de entrar**. Al arrancar, el propio
`plazum serve` lo dice:

    AVISO: este servidor se construyo sin almacen de usuarios (Config.HayAdmin o
    Config.CrearAdmin sin cablear), asi que no se puede crear el primer
    administrador ni entrar. Sirve para montar pantallas y para nada mas.

Y las dos salidas que el producto ofrece están cerradas las dos:

| ruta | qué contesta | qué lee la persona |
|---|---|---|
| `/primer-admin` | **503** | *«este servidor se construyo sin Config.CrearAdmin [...] Arreglo para quien lo cablea: pasa la funcion al construir serve.Config»* |
| `/entrar` | 200 | un formulario de usuario y contraseña, para credenciales que no puede tener nadie |

**Por qué es P0 y no P1.** Es la puerta D11-a en su forma más pura: el mensaje
de `/primer-admin` no está escrito para quien lo va a leer, está escrito para
**quien programa el producto**. Quien lo lea no tiene ninguna acción posible, y
además el mensaje le dice que el problema es de otro. Y es la puerta D11-e en su
forma más pura también: el tiempo hasta el valor de la mitad del camino no es
largo, es **infinito**.

Que el acta, la revisión de accesos y el escalado exijan sesión **está bien**:
los tres llevan nombres de personas y quién hizo qué dentro de la organización.
Lo que está mal es que no exista la sesión.

**Dónde está el arreglo, y por qué no lo hace este frente.** En
`cmd/plazum/serve.go`, que construye `serve.Config` sin `HayAdmin` ni
`CrearAdmin`. `superficies/serve` **ya tiene el mecanismo entero y probado**
(token de un solo uso impreso por la salida estándar, caducable, con su CSRF y
su rotación de sesión al subir de privilegio): lo que falta es el almacén de
usuarios que las dos funciones necesitan, y un almacén de usuarios es un
adaptador nuevo bajo `adaptadores/`, que no está en la columna de este frente.
Se para y se dice, que es lo que toca.

**Cardinal, y está topado.** `PasosQueExigenSesion = 3`, comparado con igualdad
exacta en los dos sentidos por `TestTTFVDelCaminoCompleto`. Si sube, el camino
se ha roto más; si baja, alguien ha cableado la entrada y tiene que bajar el
número en el mismo commit.

---

## P1-2. La entrevista son 41 preguntas seguidas, y es tres cuartas partes del TTFV

**Qué pasa.** `/alcance` pinta **41 preguntas** con el corpus que se publica.
A 20 s cada una son **13m40s**, o sea el 72 % del TTFV entero. Todo lo demás del
camino recorrible (tres lecturas de pantalla y dos órdenes de terminal) suma
5m16s.

**Por qué es un hallazgo de D11-a y no solo de D11-e.** No es que sean muchas:
es que llegan **todas de golpe y sin consecuencia visible**. Quien las contesta
no sabe, mientras las contesta, qué le va a pasar por contestar que sí, así que
las 41 se leen como un formulario de alta y no como el paso que produce el
valor. Es exactamente el punto donde `ETAPAS.md` sitúa el abandono, y la pieza
que lo cubre ya está escrita en el plan: *«la pregunta con su consecuencia al
lado: si contestas que sí, se te activan estas nueve obligaciones»* (pieza 2 del
bloque IA de adopción, D-20).

**Lo que NO es el arreglo.** Bajar el coste por pregunta de 20 s a 12 s hace
pasar la puerta hoy mismo sin tocar el producto. Es la forma más limpia de
mentirse, y por eso el techo del TTFV está escrito aparte y con dientes en las
dos direcciones.

**Techo.** `TechoDeclaradoTTFV = 20m`, con poco más de un minuto de margen sobre
la medida de hoy, o sea unas tres preguntas. Añadir entrevista sin mirar el TTFV
se pone rojo.

---

## P1-3. El calendario, en una instalación recién hecha, manda al terminal y a rearrancar

**Qué pasa.** El paso 2 del camino no enseña fechas: enseña su estado vacío, y su
verbo es

> Responde la entrevista, expórtala con `plazum alcance` y arranca el servidor
> con `plazum serve --alcance alcance.json`

Es **correcto** (la puerta D11-b lo exige y este frente lo comprueba contra la
respuesta HTTP) y a la vez es **el segundo coste humano del camino**: 3m45s, dos
salidas al terminal y un **rearranque del servidor** en mitad del recorrido.

**Por qué es hallazgo.** El camino guiado dice que se recorre «sin salir de
`plazum serve`», y en el paso 2 hay que salir, teclear dos órdenes y volver a
arrancar. La entrevista se contesta en el navegador y sus respuestas viajan en
la dirección, así que la persona ya tiene el dato en la mano y aun así tiene que
exportarlo a un fichero y reiniciar el proceso para que la pantalla siguiente lo
vea.

**Dónde estaría el arreglo.** En que la pantalla del calendario leyera el alcance
de la consulta igual que lo hacen `/alcance` y `/controles`, en vez de sólo del
fichero que se pasa al arrancar. Es un cambio de `cmd/plazum/serve_calendario.go`
y de la fuente que lo alimenta, y toca decidir si el alcance de la consulta puede
alimentar una pantalla que además escribe un `.ics`. Se anota y no se hace aquí:
cambia el contrato de una fuente.

---

## P2-4. El panel de inicio enseña «363» sin que nadie haya contestado nada

**Qué pasa.** `/hoy` en una instalación recién arrancada, sin ninguna respuesta
de la entrevista, pinta cuatro cifras y dos de ellas con número: **363** y
**21**. Las dos primeras salen a cero, que es correcto y lleva su descargo.

**Por qué se anota.** 363 es el corpus instalado, no *lo tuyo*. La regla de la
casa dice que un cero que no se ha podido contar se pinta como SIN DATO y no
como cero, y esa mitad está bien resuelta; la mitad simétrica (**un número
grande que no es una respuesta sobre ti**) no tiene la misma protección. No es
una acusación, así que no es P0 ni P1: es que la primera pantalla del producto
enseña su cifra más grande antes de saber nada de quien la mira.

**No se toca desde aquí sin medir**: el panel es de `superficies/pantallas` y
tiene su propia puerta de descargos; cambiar qué se pinta sin datos es una
decisión de producto, no una corrección.

---

## P2-5. El estado vacío del acta explicaba la condición y no el paso

**ARREGLADO en este mismo frente**, y se deja escrito porque es el caso que hizo
nacer roja la puerta D11-b enumerada.

La pantalla del acta sin ninguna acta decía qué es un acta y que *«se compone en
cuanto haya al menos una de esas tres fuentes»*. Eso es una **condición**, no un
paso: quien lo leía se quedaba sabiendo lo que le faltaba y sin saber qué
teclear. Ahora lleva las dos órdenes exactas.

Lo que hay que recordar de aquí no es el arreglo, es **cómo se encontró**: no
por revisar la pantalla, que llevaba su test de estado vacío en verde, sino por
enumerar las superficies desde el árbol y exigir que el verbo saliera en la
respuesta HTTP. La pieza pasaba su puerta; lo que faltaba era la puerta que las
mirara juntas.

---

## Lo que este frente NO ha podido mirar, con su cardinal

- **3 de 6 pasos del camino** no se han podido recorrer (P0-1), así que de sus
  pantallas llenas no hay medida: el acta, la revisión de accesos y el escalado
  sólo se han visto en su estado vacío y en sus tests.
- **10 de las 14 cifras** del calendario siguen sin poder abrirse (ver
  `superficies/calendario/cuenta.go`): abrirlas exige que `nucleo/pantalla`
  devuelva los cubos descartados en la misma unidad en que los cuenta, y
  `nucleo/` es otra columna.
- **La accesibilidad de lo nuevo** (los enlaces de la cuenta del calendario y el
  bloque de órdenes del acta) no se ha pasado por axe en local: el paso de axe
  vive en `.github/workflows/etapa2-accesibilidad.yml` y se ejecuta en CI. El
  marcado añadido reutiliza el de pantallas que ya pasan (`<a>` a un ancla de la
  misma página, `<pre><code>` como el de la revisión de accesos), pero eso es un
  argumento, no una medida.
