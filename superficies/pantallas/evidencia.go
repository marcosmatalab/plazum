package pantallas

import (
	"context"
	"time"
)

// EL ESTADO DE LA EVIDENCIA EN LA PANTALLA DE CONTROLES (A2 de D-22).
//
// Es el eslabon 4 de los siete: hasta hoy `estado.Calcular` lo llamaba un solo
// sitio, el verificador del expediente, asi que su resultado no lo veia nadie
// hasta que el documento salia por la puerta.
//
// # LA PREGUNTA DEL INVARIANTE 12, CONTESTADA AQUI Y NO EN LA CABEZA
//
// **¿Esta superficie se sirve con sesion o sin ella?** `/controles` se sirve
// SIN ELLA: `superficies/camino` la declara `PasoAlcanzable`, o sea que contesta
// 200 a quien no ha entrado, y `superficies/pantallas` no tiene ni una rama de
// 401, a diferencia de acta, uar y escalado.
//
// **¿Y de quien es una observacion?** DE LA INSTALACION, no de la cuenta.
// `estado.Observacion` no tiene ni un campo de dueno, y lo que dice —si un
// recurso real de la organizacion satisface un predicado— no cambia segun quien
// abra la pagina. Colgarla de la cuenta daria dos veredictos de cumplimiento
// distintos para la misma empresa y ninguna forma de saber cual es el bueno,
// que es exactamente el fallo barato del invariante 12. Por eso este interfaz
// NO recibe la peticion, igual que `calendario.Fuente`.
//
// # Y DE AHI SALE EL PROBLEMA, QUE ES EL CARO
//
// Si la observacion es de la instalacion y la pagina se sirve sin sesion,
// **pintarla es publicarla a quien no ha entrado**. Y lo que se publicaria no es
// inocuo: que un control esta fallando, desde cuando, y con que recolector. Eso
// es reconocimiento gratis sobre la postura de seguridad de una organizacion, y
// es la misma clase de contenido por la que el acta contesta 401.
//
// **LA DECISION, con sus dos alternativas dichas y por que se descartan.**
//
//	(a) `/controles` pasa a exigir sesion.  Honesto y caro: la lista de
//	    controles que te alcanzan NO es sensible (sale del corpus publico y del
//	    alcance publicado), es un paso del camino guiado, y cerrarla entera para
//	    tapar tres columnas es apagar la pantalla para arreglar una fila.
//	(b) las tres columnas solo con sesion, Y LA PAGINA LO DICE.  Es esta.
//
// La objecion a (b) es real y es la que la salva si se contesta: *la misma
// pagina significa dos cosas segun quien mire*. Se contesta como se contesto en
// el plan de avisos hace dos bloques, **con una frase en la pantalla que se
// queda**: sin sesion, la pagina dice que el estado de la evidencia no se
// ensena sin entrar.
//
// Y LA FRASE NO DEPENDE DE SI HAY EVIDENCIA O NO. Es lo que la hace no filtrar:
// «hay evidencia, entra para verla» ya dice que la hay. Sin sesion se dice
// siempre lo mismo, haya cero observaciones o mil.
//
// LO VIGILA: TestSinSesionNoSaleNiUnDatoDeEvidenciaYLaPaginaLoDice
//
// # LAS TRES FORMAS DE LA NADA, QUE AQUI SON TRES DISTINTAS
//
//	sin adaptador        esta instalacion no sabe leer evidencia. La pantalla
//	                     no promete nada: ni columnas ni frase de «entra».
//	sin prueba declarada el paquete no dice que se mira para esta obligacion.
//	                     Es un hueco del CORPUS y se dice como tal.
//	con prueba y sin observacion  nadie ha recolectado todavia. Es un hueco de
//	                     la INSTALACION y se dice distinto.
//	ilegible             hay fichero de observaciones y no se puede leer.
//	                     SIEMPRE error, jamas «todavia nadie ha recolectado».
//
// Las dos del medio se confunden con una sola mirada y tienen arreglos
// distintos: una la arregla quien escribe el paquete y la otra quien monta el
// recolector. La cuarta es la tercera forma del invariante 8 y es la unica que
// no puede degradarse a las otras.
type Evidencias interface {
	// De devuelve el estado de la evidencia por identificador de obligacion.
	//
	// Devuelve un mapa y no una lista por lo mismo que el resto de esta
	// superficie empareja por identidad: la pantalla pinta filas en el orden
	// del corpus, y casar por posicion movería el estado de un control al de
	// otro en cuanto alguien reordenara un paquete (invariante 7).
	//
	// Una obligacion SIN entrada en el mapa es «no hay prueba declarada». Una
	// obligacion CON entrada y `SinObservaciones` es «hay prueba y nadie ha
	// recolectado». No son lo mismo y no se pueden colapsar.
	De(ctx context.Context) (map[string]Evidencia, error)
}

// Evidencia es lo que se sabe del estado de UNA obligacion.
//
// NO LLEVA LA LISTA DE RECURSOS QUE FALLAN: `estado.Entrada.Fallando` trae
// identidades de maquinas y de cuentas enteras, y eso vive en el expediente, que
// va detras de otra puerta. Lo que se pinta es el estado, la edad y quien lo
// trajo.
//
// PERO SI PUEDE LLEVAR UNA IDENTIDAD SUELTA, Y HAY QUE DECIRLO PORQUE ESTE
// GODOC AFIRMABA LO CONTRARIO. Dos de los once motivos del motor nombran el
// recurso cuya observacion caduco, y otro lleva dentro el error con el que
// contesto el recolector: los tres viajan en `MotivoArgs`. La frase «no lleva ni
// un nombre de recurso» era falsa desde el primer dia y no la vigilaba nada,
// porque `Motivo` era una cadena opaca y ninguna puerta mira dentro de una
// cadena.
//
// Lo que hace que eso no sea una fuga es la frontera de sesion de arriba, no
// esta estructura: las tres columnas SOLO se pintan con sesion, asi que la
// identidad se le ensena a quien ya ha entrado en su propia instalacion, que es
// justo la persona que necesita saber CUAL recurso mirar. Sin sesion no sale
// ninguna de las tres, y la pagina lo dice.
//
// LO VIGILA: TestSinSesionNoSaleNiUnDatoDeEvidenciaYLaPaginaLoDice
type Evidencia struct {
	// Estado es el valor de `estado.Estado`, en su forma de texto. Se guarda
	// como cadena y no como el tipo del nucleo para que esta superficie no
	// tenga que importar el motor: lo unico que hace con el es elegir una
	// clave de catalogo.
	Estado string
	// MotivoClave es la CLAVE DE CATALOGO del porque que dio el motor, y
	// MotivoArgs los datos que rellenan sus huecos.
	//
	// POR QUE UNA CLAVE Y NO LA FRASE. Aqui viajaba la cadena del motor tal
	// cual, con el argumento de que reescribirla en la superficie seria una
	// segunda copia de la derivacion. El argumento sigue siendo bueno y la
	// conclusion era mala: el motor escribe en castellano, asi que la pagina
	// inglesa imprimia espanol, y ninguna puerta de i18n podia verlo porque
	// todas vigilan el catalogo y esto no era una clave. La salida no es
	// reescribir la frase, es que el motor emita CLAVE y no frase; la
	// derivacion sigue teniendo un solo dueno y ademas se traduce.
	//
	// Vacia cuando el motor no dio motivo, que no ocurre hoy en ninguna de sus
	// once ramas y por eso la plantilla no pinta nada en vez de pintar una
	// clave en crudo.
	MotivoClave string
	// MotivoArgs son DATOS y por eso no se traducen: un recuento, una fecha en
	// ISO, el nombre de un recurso, o las palabras de quien aprobo una
	// excepcion. Lo que se traduce es la frase que los rodea.
	MotivoArgs []string
	// Recolectada es CUANDO SE TOMO EL DATO MAS VIEJO considerado. Cero
	// significa que no se considero ninguna observacion, que ocurre de verdad
	// (excepciones, pass por defecto, no aplica) y NO es «hace mucho».
	Recolectada time.Time
	// SinObservaciones distingue «hay prueba y nadie ha recolectado» de «hay
	// prueba y el motor decidio sin mirar observaciones». Las dos dan
	// Recolectada cero y solo una es un hueco de la instalacion.
	SinObservaciones bool
	// Recolector es QUIEN trajo el dato. Vacio cuando no hubo observaciones.
	Recolector string
	// Cierra dice si esta prueba en verde CIERRA la obligacion o solo APORTA a
	// un aspecto suyo (C2 de D-22). Sobre una obligacion observable solo por
	// faceta, `false`: lo que la norma exige ahi es otra cosa, y darlo por
	// cerrado porque el aspecto observable dio verde es absolver de mas.
	Cierra bool
	// Predicado es QUE se evaluo, en las palabras del paquete. Es lo que
	// sostiene al estado: sin el, un verde es una afirmacion sin nada detras.
	Predicado string
}
