# Esqueletos: paquetes que existen y todavia no entregan nada

Un paquete de aqui tiene **metadatos correctos y cero obligaciones**: su URN, su
estrato legal, el regimen de licencia de su fuente, su atribucion y su fuente
oficial. Lo que no tiene es ni una obligacion escrita, asi que no hay nada que
un cliente pueda cumplir con el.

## Por que estan fuera de `paquetes/`

Hasta el 08-09-2026 estaban dentro, y por tanto contaban como marco en el
README, en `paquetes/CORPUS.md` y en la pantalla de controles. **Un escaparate
que dice 33 marcos y entrega 21 no es optimismo, es un dato falso**, y el precio
lo paga quien instala plazum porque su marco esta en la lista, abre la pantalla
y no encuentra ni una obligacion suya. Esa primera impresion no se repite.

## Por que no se borran

Porque lo que traen no es basura: es el andamiaje **ya verificado** de un
paquete futuro. Averiguar el URN correcto de una norma, su estrato legal y bajo
que regimen se puede citar su texto cuesta una sesion de trabajo por marco, y
borrarlo obligaria a rehacerla.

## Que se exige de un esqueleto

Sigue pasando el mismo linter que el corpus publicado (`corpus.Cargar`), y hay
una puerta que lo comprueba. **Un esqueleto que se pudre en silencio no vale mas
que uno borrado**: el dia que alguien vaya a escribirlo se encontraria el URN
mal formado o la atribucion caducada, justo cuando ya no lo espera.

## Como vuelve uno al escaparate

Se le escriben sus obligaciones y se mueve a `paquetes/`. En el **mismo commit**
suben `MinimoDeMarcos` y bajan `EsqueletosEsperados`, los dos con igualdad
exacta: se mueven juntos o uno de los dos miente. La puerta que lo dice es
`TestNingunPaquetePublicadoLlegaVacio`, que mira en las dos direcciones: aqui no
puede quedarse un paquete que ya este escrito, porque eso es trabajo hecho y sin
publicar, que es la forma cara de perderlo.
