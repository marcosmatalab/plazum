# Traducir plazum

Esta página es para quien va a escribir o revisar las cadenas de la interfaz. Se
lee entera en diez minutos y la primera sección es la que no se puede saltar.

## 1. Qué entra en el catálogo y qué no entra jamás

En el catálogo van las cadenas de la **interfaz**, es decir, lo que dice la
herramienta con su propia voz, títulos de pantalla, botones, ayudas, mensajes de
error.

En el catálogo **no va el texto de una norma**. Nunca. Ni traducido, ni
resumido, ni parafraseado.

No es una regla de estilo, es una regla legal, y conviene entender el porqué
antes de escribir la primera cadena. El texto del BOE y del DOUE se puede
reproducir citando la fuente (art. 13 del TRLPI y Decisión 2011/833/UE), y por
eso los paquetes de corpus lo transcriben. Pero una traducción nuestra de ese
texto ya no es el BOE, es obra nuestra presentada como si fuera la norma. Eso se
sale de la estratificación de licencias que sostiene todo el corpus, y le da al
usuario una versión de la ley que no ha aprobado nadie.

El idioma del corpus va **por paquete, dentro del paquete**. Un paquete en
alemán es un paquete distinto, con su propia fuente oficial, no una traducción
en tiempo de render.

Esto no depende de que nadie se acuerde. El cargador rechaza una cadena con
pinta de norma y no arranca, y hay un test que compara cada cadena del catálogo
contra el texto legal de todos los paquetes instalados, trozo a trozo. Los
detalles están en la sección 5.

## 2. El caso que confunde a todo el mundo

Un usuario pone la interfaz en inglés y ve los botones en inglés y los
artículos de la norma en español. Parece un fallo y no lo es.

Por eso existe la clave `aviso.idioma_del_corpus`, y por eso hay que enseñarla
al lado del texto que viene del corpus, no escondida en una página de ayuda. Lo
que dice, en inglés, es que el texto legal se muestra en el idioma oficial de su
fuente, que la interfaz sí está traducida y que la ley no, porque una traducción
nuestra dejaría de ser la ley.

Si alguien renderiza texto de corpus sin ese aviso al lado, el usuario lo lee
como un producto a medio traducir.

Y hay una consecuencia técnica que se olvida siempre. Si la interfaz está en
inglés y el artículo del ENS sigue en castellano, ese bloque tiene que llevar su
propio `lang="es"` en el HTML. Es WCAG 3.1.2, Language of Parts, nivel AA, y
axe no lo detecta porque axe no sabe en qué idioma está escrito un párrafo. Sin
ese atributo, un lector de pantalla en inglés pronuncia el castellano como si
fuera inglés y no se entiende nada.

## 3. Quién decide qué claves existen

No las decide el catálogo. Las decide la interfaz, y las declara en un solo
sitio, `superficies/pantallas.ClavesDeCatalogo()`. Esa función es el contrato
entre la superficie y el catálogo, y hay un test que compara las dos listas en
los dos sentidos, así que no puede quedarse corta ni sobrarle nada.

Para añadir una clave:

1. Decláratela primero en la superficie que la va a pintar.
2. Elija el espacio de nombres. La lista es cerrada y está en
   `adaptadores/catalogo/frontera.go`, hoy `ui`, `pantalla`, `menu`, `origen`,
   `vacia`, `alcance`, `derivacion`, `estado`, `filtro`, `tabla`, `columna`,
   `error` y `aviso`. Un espacio nuevo es una decisión, no un descuido, y jamás
   puede ser el nombre de una norma.
3. Escriba la clave en minúsculas, con puntos y sin tildes,
   `alcance.pregunta.si`.
4. Añádala a `adaptadores/catalogo/cadenas/es.json` **y** a `en.json`. Las dos.
   Si solo la pone en una, la puerta de CI se pone roja, y hace bien.

El castellano es el idioma de referencia. Fija el inventario de claves y es
contra el que se mide si a otro idioma le falta algo.

Hay una sola excepción y está escrita con su motivo en el propio test, en
`clavesPropias`: `aviso.idioma_del_corpus`, que existe porque explica la
decisión legal de la sección 1 y todavía no hay pantalla que la pinte.

## 3 bis. El plural

Una cadena con contador lleva sus formas separadas por barra vertical:

```json
"menu.aplican": "%d aplica|%d aplican"
```

Quien pinta la pantalla pasa el número y nada más. La forma la elige el
catálogo, porque la pantalla no sabe en qué idioma está escribiendo, y las
formas del plural dependen del idioma (el ruso tiene tres, el árabe seis). La
regla de hoy es la del castellano y el inglés, uno es singular y el resto
plural, y el sitio donde se amplía es `elegirForma` en
`adaptadores/catalogo/catalogo.go`.

Dos reglas al escribirlas:

- Todas las formas de una clave llevan los mismos verbos de formateo. Si el
  singular lleva `%d` y el plural no, la etiqueta sale sin el número justo en el
  caso que menos se prueba. La puerta de CI lo caza.
- Si no hay contador entre los argumentos se devuelve la última forma, el
  plural, que es la que suena bien con una cantidad desconocida.

## 4. Traducir

- Traduzca el sentido, no las palabras. Es la interfaz de una herramienta, no un
  documento legal.
- Los verbos de formateo (`%s`) tienen que aparecer los mismos y en el mismo
  orden que en castellano. Si en su idioma el orden natural es otro, use el
  índice explícito, `%[2]s ... %[1]s`. Si se le cuela uno de más o de menos,
  la puerta de CI lo caza y se lo dice por su nombre.
- Deje el valor vacío antes que inventar una traducción. Un hueco lo caza CI y
  se arregla; una traducción inventada de un mensaje legal no la caza nadie.
- Nada de HTML dentro de una cadena. El marcado lo pone la plantilla.
- Una cadena es de una línea y corta. El límite son 240 caracteres, y está ahí
  porque un párrafo largo en el catálogo casi siempre es texto de una norma que
  se ha colado.

## 5. Lo que el cargador rechaza, y por qué

Si una entrada rompe una de estas reglas, el catálogo no carga y el mensaje dice
qué pasa y cómo se arregla. Un hueco degrada, una ilegalidad no.

| Se rechaza | Motivo |
| --- | --- |
| Clave fuera de la lista de espacios de nombres | Un espacio nuevo es una familia de cadenas nueva, y una clave con nombre de norma sería además una norma cableada |
| Clave con mayúsculas, tildes, espacios o sin punto | Una clave no es texto |
| Valor de más de 240 caracteres | Un rótulo no es un párrafo |
| Valor con salto de línea o tabulador | Un rótulo es de una línea |
| Valor con etiquetas HTML, entidades numéricas o `javascript:` | El marcado lo pone la plantilla, y una cadena con etiquetas dentro es una invitación a un XSS almacenado |
| Valor que contiene `urn:` | El catálogo no nombra normas |
| Valor que parece una cita normativa | "artículo 31", "Real Decreto", "Reglamento (UE)", "Annex I" y compañía. El catálogo no transporta texto legal |
| Fichero con JSON inválido, clave repetida, valor no textual o codificación que no es UTF-8 | Una clave repetida tapa a la anterior en silencio y el que la escribió nunca sabe por qué no sale |

Y por encima de todo eso, el test `TestElCatalogoNoTransportaTextoDelCorpus`
compara cada cadena contra el texto legal de todos los paquetes instalados por
trozos de seis palabras. Si una cadena repite seis palabras seguidas de una
norma, el test dice de qué paquete salen.

Lo que ese test no ve, dicho para que conste: una paráfrasis, y un trozo literal
de cinco palabras o menos. Contra eso quedan las reglas de la tabla y la
revisión humana.

## 6. Añadir un idioma

Hoy se cargan dos, castellano e inglés. Y el mecanismo admite los que hagan
falta.

El alemán **no está**, y no se promete en ningún sitio, ni en la web, ni en el
README, ni en un desplegable con una bandera en gris. Llega cuando exista un
partner DACH que lo revise y responda de lo que dice la interfaz en alemán. Una
traducción sin nadie que responda de ella, en una herramienta que le dice a un
CISO qué fecha tiene que cumplir, no es media función, es un riesgo.

Para añadir uno hacen falta tres cosas, en este orden:

1. **Alguien que revise ese idioma y responda de él.** Sin esto, lo demás no se
   hace.
2. `adaptadores/catalogo/cadenas/<idioma>.json` con todas las claves que tiene
   `es.json`.
3. Dos líneas en `adaptadores/catalogo/catalogo.go`, la directiva `go:embed` que
   nombra los ficheros uno a uno y la lista `idiomasEmbebidos`. Y el test
   `TestSoloSeCarganEsYEnMientrasNoHayaPartnerQueRevise`, que está puesto justo
   para que este paso no se dé por descuido.

Un fichero de idioma que aparezca en el directorio sin tocar el `go:embed` no
hace nada, a propósito.

## 7. Las puertas que va a encontrar en CI

El workflow `etapa2-accesibilidad.yml` falla, no avisa, si:

- a un idioma le falta una clave que el castellano tiene,
- un idioma tiene una clave que el castellano no tiene, que es una clave muerta
  o un renombrado a medias,
- los verbos de formateo no casan entre idiomas, o no casan entre las formas del
  plural de una misma clave,
- la interfaz pide una clave que el catálogo no tiene, o el catálogo traduce una
  que ya no pide nadie,
- una pantalla servida enseña una clave sin traducir, en cualquiera de los dos
  idiomas.

Ese último es el que de verdad importa. Se comprueba sobre la página renderizada
de verdad, con el navegador, en castellano y en inglés.

## 8. Estilo

### El inglés es británico

`programme`, `organisation`, `authorisation`, `analyse`, `centre`. Nunca
`program`, `organization`, `authorization`, `analyze`, `center`.

No es una preferencia estética. El comprador de plazum es europeo y el inglés
que ya está leyendo, el de las normas de la UE publicadas en EUR-Lex, está
escrito así. Un catálogo mitad británico y mitad americano no se lee como una
elección, se lee como que nadie lo mira, y esta herramienta le dice a un CISO
qué fecha tiene que cumplir.

La decisión se tomó de hecho al escribir las cadenas del acta, que salieron
todas en británico sin que nadie lo hubiera escrito en ningún sitio. Una
elección que no está escrita no es una elección: es lo que salió, y lo siguiente
que salga puede salir distinto. Por eso está aquí y por eso tiene puerta.

Lo vigila `TestElInglesDelCatalogoEsBritanico`, que recorre los valores de
`en.json`, nunca las claves (una clave se llama `acta.pantalla.organizacion`
porque los identificadores de este repositorio van en castellano, y su valor
`Organisation` es correcto). La lista de grafías americanas es cerrada y corta a
propósito, porque la regla general del sufijo no se puede escribir como
subcadena: `size` y `seize` acaban en `ize` y no son verbos. Si se le cuela una
que no está en la lista, se añade una línea a la lista, no se quita la puerta.

Y un caso que va a aparecer: `program` es inglés británico legítimo cuando
significa un programa de ordenador. En plazum no significa eso ni una vez, el
programa es el de auditoría interna, que en británico es `programme`. Si algún
día hace falta hablar de software, se reescribe la frase.

### Lo demás

- Al usuario se le tutea. Lo decidió el frente que diseñó las pantallas y el
  catálogo lo respeta, en los dos idiomas.
- Sin guiones largos. Una coma o un punto hacen el mismo trabajo.
- Un mensaje de error dice qué ha pasado y qué hacer. "Error inesperado" no es
  un mensaje.
- Las tildes en las cadenas del catálogo sí van, y en castellano no son
  opcionales. "Si" y "Sí" son dos palabras distintas, y una de ellas es el botón
  con el que un CISO declara que le aplica una norma. La regla de escribir sin
  tildes es para los identificadores del código, no para lo que lee una persona.
