# ingestanorma

La tuberia de ingesta legal. Convierte una norma publicada en articulado
estructurado con su cita y su enlace, y al volver a ejecutarla dice que ha
cambiado desde la ultima vez.

Son dos cosas y las dos importan. La primera te ahorra copiar articulos a mano.
La segunda es la vigilancia normativa del producto, y es la que no se puede
fingir, porque es historico.

```
go run ./herramientas/ingestanorma -eli https://www.boe.es/eli/es/rd/2022/05/03/311
go run ./herramientas/ingestanorma -id BOE-A-2022-7191 -articulos 31,33
go run ./herramientas/ingestanorma -celex 32016R0679 -json > rgpd.json
go run ./herramientas/ingestanorma -celex 32016R0679 -borrador > borrador.json
go run ./herramientas/ingestanorma -historial
```

## De un ELI a un borrador de paquete, en tres pasos

1. **Busca el identificador.** En la ficha de la norma en el BOE, el campo ELI.
   Tiene la forma `https://www.boe.es/eli/es/RANGO/AAAA/MM/DD/NUMERO`. En
   EUR-Lex, el numero CELEX de la ficha, tipo `32016R0679`. No hace falta nada
   mas: el identificador interno lo resuelve la herramienta.

2. **Mira lo que hay.** Una ejecucion sin mas argumentos lista el articulado con
   su titulo, desde cuando esta en vigor cada articulo y cual esta derogado.

   ```
   go run ./herramientas/ingestanorma -eli https://www.boe.es/eli/es/rd/2022/05/03/311
   ```

3. **Saca el borrador.** `-borrador` escribe un `paquete.json` en la salida
   estandar con el URN propuesto, la licencia, la atribucion, la fuente y una
   obligacion por articulo, cada una con su texto legal, su cita y su enlace con
   ancla al articulo concreto.

   ```
   go run ./herramientas/ingestanorma -eli ... -borrador > paquetes/mi-norma/paquete.json
   ```

   **El borrador NO carga tal cual, y es a proposito.** Cada obligacion sale sin
   `id` y sin `clase_e2e`, que son las dos decisiones que no puede tomar una
   maquina: como se llama esa obligacion en el corpus, y si es observable,
   documental, procedimental, notificatoria o de remediacion. El linter rechaza
   el paquete hasta que una persona las escriba. Que no se pueda commitear por
   descuido es la propiedad, no la molestia.

   Lo que si viene hecho es lo que se copia mal a mano: la cita exacta, el
   enlace, la vigencia de cada articulo, y la transcripcion literal.

4. Lo que sigue (los relojes, los casos dorados, las reglas de aplicabilidad) es
   el trabajo de autoria y esta en `docs/guia.md`, anexos B y C, y en el comando
   `/autoria`. Esta herramienta llega hasta el borrador y ni un paso mas.

## La vigilancia

Cada ejecucion completa guarda una instantanea y anota una linea en el
historial. La siguiente ejecucion compara y dice que ha cambiado:

```
   VIGILANCIA
      1 nuevos, 1 modificados, 0 derogados, 51 sin cambio
      MODIFICADO  Disposición adicional segunda  (por BOE-A-2024-22935, en vigor 2024-11-07)
```

Los articulos se comparan por su ROTULO ("Artículo 31", "ANEXO II"), no por el
identificador interno del bloque, porque ese se desplaza cuando la fuente
inserta un articulo y entonces el diff diria que cambio media norma.

El almacen esta en `corpus-vigilancia/` (se cambia con `-almacen`):

```
corpus-vigilancia/es-boe-a-2022-7191/instantanea.json   contra lo que se compara
corpus-vigilancia/es-boe-a-2022-7191/historial.jsonl    el track record, solo se anade
```

`-historial` lo lista, y `-historial -json` lo sirve para que otra cosa pinte la
pagina publica de vigilancia. Una observacion sin cambios tambien se anota: el
track record tiene que poder demostrar que se miro y no habia nada.

### El cambio que este barrido NO puede senalar: la supresion de un punto de anexo

Medido el 02-09-2026 sobre el Reglamento (UE) 2026/1744, que modifica el AI Act
en 43 puntos. **Cuarenta y uno de esos puntos son visibles aqui**: sustituyen,
anaden o insertan, y el diff por rotulo los ensena como MODIFICADO. Dos no:

```
41) El anexo I se modifica como sigue: en la seccion A, se suprime el punto 1;
42) En el anexo VIII, seccion B, se suprimen los puntos 7 y 9.
```

**Una supresion CORRE LA NUMERACION de todos los puntos que quedan detras.** El
que era el punto 8 pasa a ser el 7, y ni el texto del punto ni su rotulo cambian:
cambia su NUMERO, que es justamente por lo que lo citamos.

Lo que ve la vigilancia es `MODIFICADO ANEXO VIII`, igual que cualquier otra
modificacion, porque compara por rotulo y el rotulo del anexo no ha cambiado. Y
lo que ve nuestro corpus es **nada**: una obligacion que cite «anexo VIII, punto
8» sigue diciendo exactamente lo mismo, con su cita intacta, apuntando a otro
contenido. **Ningun test se pone rojo, porque no hay nada que se haya roto: hay
un numero que ahora significa otra cosa.**

> **Es una clase de cambio invisible al barrido textual, y hay que leerla a
> mano.** Cuando el diff diga que un ANEXO esta modificado, la pregunta no es
> «que dice ahora», es **«se ha suprimido algun punto, y citamos alguno posterior
> a el»**. La primera parte la contesta el texto del acto modificativo; la
> segunda, un `grep` del numero de punto en `paquetes/`.

En el caso medido no habia dano: `paquetes/ai-act` no cita ningun punto del anexo
I seccion A ni del anexo VIII seccion B. La proxima vez puede haberlo.

**Tres casos no se registran, y la salida dice cual:** `-articulos` (una
extraccion parcial haria que la siguiente completa viera el resto de la norma
como articulos nuevos), `-fecha` (haria retroceder la vigilancia) y
`-sin-registrar`.

## De donde se baja, exactamente

**BOE.** La API de datos abiertos de legislacion consolidada,
`/datosabiertos/api/legislacion-consolidada`. El ELI se resuelve buscando por
fecha de disposicion y casando el campo `url_eli` que devuelve la propia API, no
rascando HTML. El texto llega con TODAS las versiones de cada bloque, y de ahi
sale `-fecha`: el texto tal como estaba en vigor ese dia.

**EUR-Lex.** Cellar, el servicio de la Oficina de Publicaciones, por negociacion
de contenido sobre `publications.europa.eu/resource/celex/{CELEX}`. La entrada
que uno espera, `eur-lex.europa.eu/legal-content/ES/TXT/XML/?uri=CELEX:...`, ya
no sirve: devuelve HTTP 200 con la portada del Diario Oficial. Comprobado contra
dos reglamentos. El Formex solo existe para los actos de hasta 2023; el XHTML
esta para todos, incluidas las versiones consolidadas, asi que se usa ese.

Para una version consolidada del DOUE se pide el CELEX consolidado, que lleva la
fecha detras: `-celex 02016R0679-20160504`. `-fecha` no vale ahi, y la
herramienta lo dice en vez de darte la ultima version callandoselo.

Hay cache en disco (`.cache/ingestanorma`) y un limite de una peticion cada
segundo y medio. Son APIs publicas y gratuitas que no nos deben nada.

## La frontera legal

- **BOE se transcribe** por el art. 13 del TRLPI: las disposiciones legales no
  son objeto de propiedad intelectual. Las condiciones de reutilizacion exigen
  citar la fuente y decir que el texto consolidado es de caracter meramente
  informativo.
- **DOUE se transcribe** por la Decision 2011/833/UE, **con atribucion
  obligatoria** (su art. 6.2.a).
- Cada extraccion emite `licencia_fuente` y `atribucion`, siempre, sin
  `omitempty`. Una atribucion que hay que acordarse de poner mas tarde es una
  atribucion que se pierde.
- **Solo fuente primaria.** La herramienta se niega a descargar de cualquier
  anfitrion que no sea BOE o EUR-Lex, y tambien si la fuente redirige fuera. Un
  espejo de GitHub con licencia MIT no vale: la licencia de un repositorio no
  alcanza al texto normativo que quien lo subio no poseia.
- ISO, PCI DSS, SOC 2, TISAX y CIS **no se tocan**: no tienen ELI y su texto no
  se puede redistribuir. Esos paquetes son referenciales o delegados y se
  escriben a mano, con identificador y titulo corto.

Nada de esto es asesoramiento juridico, y la extraccion no es texto oficial. Lo
dice tambien la salida.

## Lo que sabe y lo que no

Sabe separar el rotulo del titulo, quedarse con la version vigente de cada
articulo, decir que norma lo modifico y desde cuando, reconocer un articulo
derogado, y meter en el texto las tablas que la fuente usa para pintar listas
(las celdas van separadas por ` | `).

No sabe decidir que obligacion hay en un articulo, ni cual es su reloj, ni a
quien se aplica. Eso es autoria.

## Los tests no salen a la red

Todo se prueba contra respuestas reales recortadas en `testdata/`. Los dos
parsers llevan fuzzing, porque tragan XML ajeno y eso es superficie de ataque.
