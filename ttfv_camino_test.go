package plazum

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/marcosmatalab/plazum/superficies/camino"
	"github.com/marcosmatalab/plazum/superficies/serve"
)

// EL TTFV SOBRE EL CAMINO COMPLETO (puertas D11-a y D11-e).
//
// # Que se mide aqui y en que se distingue del TTFV que ya habia
//
// `.github/workflows/etapa2-ttfv.yml` mide EL ARRANQUE: conseguir el binario y
// llegar a la primera pantalla con valor (`plazum demo`). Es un numero util y no
// es este. Aqui se mide EL CAMINO ENTERO: los seis pasos de camino.Canonico(),
// de principio a fin, sobre el binario de verdad arrancado con `plazum serve`.
//
// # El producto se levanta DE VERDAD, y esa es la decision cara del fichero
//
// Se compila el binario, se arranca como proceso y se le hacen peticiones HTTP.
// No se construyen las superficies a mano: una reconstruccion parecida al
// cableado mide otra cosa, y lo que este numero tiene que contestar es «que le
// pasa a quien se descarga esto un martes por la manana», no «que le pasa a
// quien escribe un test».
//
// EL PRECIO ES QUE ESTE TEST TARDA, porque compila. Se asume: la alternativa es
// medir un producto que no existe.
//
// # El modelo de coste, declarado para que se pueda recontar
//
//	TTFV = T_maquina + T_humano
//
//	T_maquina = compilar el binario + arrancar hasta que /salud contesta +
//	            la suma de las latencias de los pasos. Reloj monotono.
//	T_humano  = por cada paso alcanzado, una LECTURA; por cada pregunta de la
//	            entrevista, una RESPUESTA; por cada orden de terminal que la
//	            pantalla exige, un TECLEO.
//
// LOS CONTEOS NO SE ESCRIBEN, SE SACAN DEL PRODUCTO: los pasos, de
// camino.Canonico(); las preguntas, contando las que pinta /alcance; las
// ordenes, de lo que cada pantalla PIDE, y ademas cada orden declarada se
// comprueba contra el <main> de su pantalla, asi que no se puede inflar ni
// desinflar respecto de lo que el producto dice de verdad.
//
// LOS TRES COSTES HUMANOS SI SON ESTIMACIONES, y se dicen como lo que son. No
// salen de una medida con personas: no hay ninguna todavia. Se eligen
// CONSERVADORES (por arriba) a proposito, porque el error que importa aqui es
// el que hace pasar la puerta, no el que la hace fallar. Cualquiera puede
// recontar el numero cambiando estas tres constantes, y por eso estan sueltas y
// nombradas en vez de sumadas en una sola cifra.
//
// # EL RESULTADO, Y LO QUE CAMBIO EL 03-09-2026
//
// Hasta esa fecha el camino completo NO SE PODIA RECORRER, y no por lentitud:
// TRES de los seis pasos (acta, revision de accesos y escalado) contestaban 401
// en un binario recien descargado y NO HABIA FORMA DE ENTRAR, porque
// `cmd/plazum/serve.go` construia serve.Config sin Autenticar, sin HayAdmin y
// sin CrearAdmin. Este fichero medio durante semanas los tres sextos que si se
// podian andar, y conto los otros tres con su cardinal.
//
// # LAS DOS MITADES SE ARREGLARON EL MISMO DIA, EN FRENTES DISTINTOS
//
// Este comentario es la resolucion de un conflicto real entre dos frentes que
// tocaron este fichero a la vez, y las dos mitades son ciertas.
//
// LA PRIMERA: la entrevista dejo de preguntarlo todo. La medida paso de 19m16s
// a 11m36s SOBRE EL TRAMO RECORRIBLE, y NO se toco ni una de las tres
// constantes de coste humano: bajar el coste por pregunta a doce segundos
// habria hecho pasar la puerta el mismo dia y habria sido la forma mas limpia
// de mentirse. Lo que bajo fueron las preguntas, de 41 a 19, porque
// /alcance dejo de preguntar las que no deciden nada. El detalle esta en
// superficies/pantallas/revelacion.go y el hueco de corpus que lo hace posible
// (23 preguntas que ninguna obligacion requiere) en
// docs/hallazgos-entrevista.md. El TRINQUETE del cuello de botella no vive
// aqui: vive en PreguntasVivasAlEmpezar, que compara por igualdad exacta en los
// dos sentidos. Aqui se mide el total.
//
// LA SEGUNDA: AHORA SE ENTRA, y este fichero mide EL CAMINO ENTERO. Se cableo
// `adaptadores/usuarios`, y con el la instalacion pasa a ser un paso REAL del
// recorrido, con su coste humano (`CosteDeInstalar`). Aqui se hace lo mismo que
// hace una persona: leer el recuadro del token que imprime el arranque, abrirlo
// en /primer-admin, pegarlo y elegir credenciales.
//
// # Y POR ESO EL NUMERO NO SE PUEDE COMPARAR CON EL DE AYER
//
// Una mitad lo baja y la otra lo sube, y la que lo sube no es un empeoramiento:
// es que el recorrido medido ya no es el mismo. Antes se median tres sextos.
// Un techo que se quedara donde estaba obligaria a esconder la mitad del camino
// para pasar, que es exactamente lo contrario de lo que este fichero mide.
//
// La trampa que si sigue prohibida es la otra: subir el presupuesto o bajar el
// coste por pregunta porque la entrevista ha crecido.

// LOS TRES COSTES HUMANOS, en segundos. Estimaciones declaradas, no medidas.
const (
	// CosteDeLeerUnaPantalla: abrir una pantalla que no habias visto, leer de
	// que va y encontrar por donde se sigue. Conservador por arriba.
	CosteDeLeerUnaPantalla = 45 * time.Second
	// CosteDeResponderUnaPregunta: leer una pregunta de la entrevista sobre tu
	// organizacion y contestar si o no. Es la interaccion mas repetida del
	// camino, asi que es la constante que mas manda en el total.
	CosteDeResponderUnaPregunta = 20 * time.Second
	// CosteDeTeclearUnaOrden: salir de la pantalla, ir al terminal, copiar una
	// orden con sus banderas, ejecutarla y volver. Es la mas cara de las tres,
	// y a proposito: cada una de estas es una salida del producto.
	CosteDeTeclearUnaOrden = 90 * time.Second
	// PreguntasDelPrimerAdmin son los datos que se contestan en el formulario
	// de instalacion Y QUE NO ESTABAN ANTES.
	//
	// Hoy es una: el nombre de la organizacion, que se pregunta ahi para que
	// el acta no haya que configurarla por bandera. ESE CAMBIO SE COBRA, y
	// cobrarlo es la mitad que importa: mover una orden de terminal a un campo
	// de formulario ahorra 1m30s y cuesta 20s, y publicar solo el ahorro seria
	// medir con la intencion de aprobar. Un campo nuevo en este formulario no
	// es gratis por estar en una pantalla que ya se estaba mirando.
	//
	// Se cobra al precio de una pregunta de la entrevista y no al de una orden,
	// porque es lo que es: leer una etiqueta y teclear una respuesta corta sin
	// salir de donde estabas.
	PreguntasDelPrimerAdmin = 1

	// PreguntasDeLaSubida son las respuestas que pide subir el censo: elegir el
	// fichero y decir de que sistema son esas cuentas.
	//
	// SON DOS Y NO UNA, y no se redondea a la baja: el fichero hay que
	// encontrarlo (lo acabas de exportar de tu IdP) y el sistema hay que
	// escribirlo. Lo que NO se cobra aqui es exportar del IdP, que ocurre fuera
	// de plazum y le pasaria igual a cualquier herramienta.
	PreguntasDeLaSubida = 2

	// PreguntasDeLaPublicacion es el acto de publicar el calendario de la
	// instalacion: un boton mas en una pantalla que ya se estaba mirando.
	//
	// SE COBRA, igual que el campo del nombre en el formulario de instalacion,
	// y por lo mismo: lo que sustituye a una orden de terminal no es gratis, y
	// publicar solo el ahorro seria medir con la intencion de aprobar. Al
	// precio de una pregunta y no de una orden, porque es lo que es: leer un
	// rotulo y pulsar, sin salir de donde estabas.
	PreguntasDeLaPublicacion = 1

	// CosteDeInstalar: leer el recuadro que imprime el arranque, encontrar el
	// token entre lo demas, copiarlo al navegador, y elegir usuario y
	// contrasena. Es UNA VEZ EN LA VIDA de la instalacion, y por eso va aparte
	// de los tres de arriba, que se repiten.
	//
	// Se cuenta desde el 03-09-2026, cuando dejo de ser imposible. Antes no
	// estaba en el modelo porque no habia nada que contar: no se podia entrar.
	CosteDeInstalar = 120 * time.Second
)

// PresupuestoTTFV es el numero de la casilla D11-e de ETAPAS.md.
const PresupuestoTTFV = 15 * time.Minute

// EL TECHO DECLARADO, y su numero se pone DESPUES de medir, nunca antes.
//
// # Los dos frentes midieron cosas distintas y ninguno pudo medir esta
//
// El que abrio la entrada midio 23m11s sobre el camino entero, pero con la
// entrevista vieja de 41 preguntas. El que encogio la entrevista midio 11m36s
// con 19 preguntas, pero sobre medio camino. Ninguno de los dos numeros es el
// de este arbol, que tiene las dos cosas, y estimarlo sumando y restando seria
// exactamente lo que la casa prohibe: escribir de memoria lo que hay que ir a
// mirar. El numero de aqui abajo sale de una ejecucion.
//
// ESTO NO ES LA TRAMPA QUE ESTE TECHO PERSIGUE. Esa trampa es subirlo porque la
// entrevista ha crecido, y sigue prohibida. Lo que ha pasado es que el recorrido
// medido ya no es el mismo: antes se median tres sextos y ahora seis. Un techo
// que se quedara donde estaba obligaria a esconder la mitad del camino para
// pasar, que es lo contrario de lo que mide.
//
// NO SE HA TOCADO EL MODELO PARA QUE SALGA EL NUMERO. Bajar el coste por
// pregunta a 12 s haria pasar la puerta hoy mismo y seria la forma mas limpia
// de mentirse: lo que hay que bajar son las preguntas, no la estimacion.
//
// EL TECHO TIENE DIENTES EN LAS DOS DIRECCIONES, igual que PUERTAS_ESPERADAS:
// por arriba, porque un TTFV que crece se pone rojo antes de que nadie lo
// note; por abajo, porque el dia que el total baje del presupuesto la puerta
// TAMBIEN se pone roja y obliga a borrar este techo en el mismo commit. Un
// hueco que se cierra y deja su cardinal puesto miente hacia arriba para
// siempre.
//
// LA MEDIDA DEL 04-09-2026, con las dos mitades dentro: 15m51s sobre los SEIS
// pasos, todos contestando 200. La casilla D11-e NO se cumple, y se queda a
// 51 segundos, que son dos preguntas y media de entrevista.
//
// Es la primera vez que este numero mide lo que dice medir. Los dos que se
// publicaron antes (23m11s con la entrevista vieja, 11m36s sobre medio camino)
// eran ciertos y median otra cosa cada uno.
//
// El margen sobre la medida de hoy es de 1m9s, o sea unas tres preguntas: es a
// proposito, para que anadir entrevista sin mirar el TTFV se note.
//
// LA MEDIDA DEL 04-09-2026, con el guardado de la entrevista ya cableado:
// **15m51s, exactamente el mismo numero**, con los mismos 19 preguntas, las
// mismas 2 ordenes y los mismos 6 pasos en 200.
//
// Y ESO ES LA RESPUESTA HONESTA, no un empate por casualidad: este modelo mide
// EL PRIMER DIA, y guardar no le ahorra ni un segundo a quien contesta por
// primera vez. Lo que el guardado quita es el SEGUNDO dia, que este numero no
// mira: hasta hoy volver costaba la entrevista entera otra vez (19 x 20 s) o
// encontrar el enlace largo en el historial. Poner eso dentro del TTFV seria
// cambiar lo que la cifra mide para que salga mejor, que es exactamente la
// trampa que este techo persigue.
//
// LA MEDIDA DEL 04-09-2026, TARDE, Y ES LA PRIMERA QUE NO SE ENGANA: 20m20s.
//
// No es que el producto haya empeorado en un dia. Es que hasta hoy la medida
// contaba las ordenes de terminal EN UNA SOLA DIRECCION: recorria la lista
// declarada y la buscaba en la pantalla, asi que cazaba una orden declarada de
// mas y NO cazaba una orden que la pantalla pide y nadie declara. El acta, la
// revision de accesos y el plan de avisos pedian DOS cada uno en sus estados
// vacios y ninguno tenia ni una declarada.
//
// SEIS ORDENES INVISIBLES, y el error iba siempre a favor: la cifra publicada
// (15m51s) llevaba minutos de menos desde que existe. Es la regla de la casa
// sobre las cifras cuyo fallo probable es favorecerte, cobrada sobre la unica
// cifra que este producto ensena al comprador.
//
// # POR QUE ESTE TECHO SUBE Y ESO NO ES LA TRAMPA QUE PERSIGUE
//
// La trampa prohibida es subir el presupuesto o bajar el coste por pregunta
// PORQUE LA ENTREVISTA HA CRECIDO. La entrevista no ha crecido: siguen siendo
// las mismas 19 preguntas y los mismos 6 pasos. Lo que ha cambiado es que la
// medida dejo de ser ciega, que es la misma clase de motivo por el que este
// techo ya se movio una vez («el recorrido medido ya no es el mismo»).
//
// Y EL PRESUPUESTO NO SE TOCA. PresupuestoTTFV sigue en 15 minutos y la casilla
// D11-e sigue SIN CUMPLIRSE, ahora por 5m20s en vez de por 51s. Bajarlo seria
// exactamente lo que este fichero existe para no hacer.
//
// # EL TECHO TIENE DIENTES DESDE HOY, y hasta hoy no los tenia
//
// Este godoc decia «EL TECHO TIENE DIENTES EN LAS DOS DIRECCIONES» y la
// constante no la leia NADIE: `grep TechoDeclaradoTTFV` daba una sola linea, su
// propia declaracion. Un techo que ningun test comprueba no es un techo, es un
// comentario, y ademas afirmaba lo contrario de lo que pasaba. Ahora se
// comprueba, por arriba y por abajo, en la misma puerta.
//
// # EL TECHO BAJA A 18m30s EL 05-09-2026, y bajarlo es obligatorio
//
// La revision de accesos dejo de pedir sus dos ordenes de terminal, asi que el
// numero paso de 20m21s a 17m21s. La regla de este fichero dice que el dia que
// el TTFV baje de verdad hay que BORRAR el techo viejo en el MISMO commit: un
// hueco que se cierra y deja su cardinal puesto miente hacia arriba para
// siempre, y un techo con 4m de margen deja de avisar de que algo ha crecido.
// El test lo exige por abajo, y aqui se cumple porque lo exige, no por buena
// voluntad.
//
// # DE QUE SE COMPONE EL 17m20s, para que se pueda recontar
//
//	lectura de 6 pantallas    6 x 45s   = 4m30s
//	19 preguntas              19 x 20s  = 6m20s
//	3 ordenes cobradas        3 x 1m30s = 4m30s
//	instalacion                           2m0s
//
// Cinco invocaciones del binario en tres pantallas, cobradas tres veces: el plan
// de avisos repite la pareja del calendario y no se cobra dos veces, porque una
// persona no la teclea dos veces.
//
// # EL CUELLO CAMBIA DE MANOS, y esa es la noticia
//
// Ahora es la entrevista: 6m20s de 17m20s (37 %), contra 4m30s de las ordenes
// (26 %). No se escribe a mano en ningun sitio, se deriva en elCuello() del
// mismo reparto que se imprime, que es la unica forma de que una frase asi no
// describa un mundo que ya no existe.
//
// LAS TRES ORDENES QUE QUEDAN, con quien las apaga al lado:
//
//	calendario  plazum alcance + plazum serve --alcance    el alcance de la
//	                                                       INSTALACION
//	acta        plazum serve --acta-organizacion           la identidad de la
//	                                                       instalacion
//	escalado    las mismas del calendario, ya cobradas     -
//
// Con las tres fuera, el mismo modelo da 12m51s, o sea por debajo del
// presupuesto. Y EL PRESUPUESTO NO SE TOCA: sigue en 15 minutos y la casilla
// D11-e sigue sin cumplirse, ahora por 2m21s en vez de por 5m21s.
//
// # Y BAJA OTRA VEZ A 17m0s, con el acta
//
// La pantalla del acta dejo de pedir su orden y el numero paso de 17m21s a
// 16m11s. Son 1m30s de orden menos 20s del campo nuevo del formulario de
// instalacion, que se cobra: mover una orden de terminal a un campo de un
// formulario que ya se estaba rellenando ahorra mucho y no ahorra todo, y
// publicar solo el ahorro seria medir con la intencion de aprobar.
//
// LO QUE QUEDA SON LAS DOS DEL CALENDARIO (`plazum alcance` y
// `plazum serve --alcance`, 3m0s cobrados, que el plan de avisos repite sin
// volver a cobrar). Con las dos fuera, el mismo modelo da 13m11s, o sea por
// debajo del presupuesto. El presupuesto sigue en 15m0s y no se toca.
//
// # 05-09-2026: 14m11s, Y LA CASILLA SE CUMPLE POR PRIMERA VEZ
//
// Bajo de 16m31s por dos cables y subio por uno, y los tres se dicen porque el
// neto solo no se puede recontar:
//
//	la cifra que no se abria     -1m30s  ahora tiene su propia pagina (D11-c)
//	el bloque de «como mandar»   -1m30s  ahora tiene la suya
//	subir el censo de accesos    +40s    la medida deja de mirar esa pantalla
//	                                     vacia y la ejerce, que cuesta
//
// EL TECHO SE PONE POR DEBAJO DEL PRESUPUESTO, que es la diferencia entre
// perseguir un objetivo y defender uno cumplido. Mientras el numero estuvo por
// encima de 15m0s, el techo servia para que no CRECIERA; desde que esta por
// debajo, lo que hay que impedir es que vuelva a cruzarlo, y un techo en 17m no
// impediria nada.
const TechoDeclaradoTTFV = 14*time.Minute + 40*time.Second

// AlcanceDelPaso dice si un paso del camino se puede recorrer en un binario
// recien descargado. Vocabulario cerrado.
type AlcanceDelPaso uint8

const (
	// PasoSinDeclarar es el VALOR CERO Y ES INVALIDO. Un paso nuevo del camino
	// que nadie declare llega aqui con el cero, y entonces el TTFV se mediria
	// sobre un camino distinto del que el producto tiene.
	PasoSinDeclarar AlcanceDelPaso = iota
	// PasoAlcanzable: contesta 200 SIN haber entrado.
	PasoAlcanzable
	// PasoQueExigeSesion: solo se sirve a quien ha entrado. Exige motivo.
	//
	// HASTA EL 03-09-2026 ESTO SIGNIFICABA «no se puede recorrer», porque no
	// habia forma de abrir sesion. Ya la hay, asi que ahora significa lo que
	// siempre tuvo que significar: que la pantalla lleva nombres de personas
	// dentro y no se ensena a quien no ha entrado. Los pasos asi SI entran en
	// el TTFV, porque instalar y entrar es parte del recorrido y esta medido.
	PasoQueExigeSesion
)

func (a AlcanceDelPaso) String() string {
	switch a {
	case PasoAlcanzable:
		return "se sirve sin haber entrado"
	case PasoQueExigeSesion:
		return "solo se sirve tras entrar"
	default:
		return "SIN DECLARAR (valor cero)"
	}
}

// PasosQueExigenSesion es cuantos de los seis pasos del camino solo se sirven a
// quien ha entrado.
//
// EL CARDINAL SE COMPARA CON IGUALDAD EXACTA Y EN LOS DOS SENTIDOS, y sigue
// valiendo 3 despues de cablear la entrada, porque lo que cuenta no ha cambiado:
// esas tres pantallas llevan nombres de personas y no se sirven sin sesion. Lo
// que cambio es que ahora se puede abrir una.
//
// Si SUBE, hay una pantalla mas que pide sesion y hay que decir por que. Si
// BAJA, alguien le ha quitado la proteccion a una de las tres, y eso es peor que
// un TTFV largo: son las unicas pantallas del producto cuyo contenido son
// personas con nombre.
const PasosQueExigenSesion = 3

// DeclaracionDePaso es lo que se afirma de un paso del camino a efectos de
// medir el tiempo hasta el valor.
type DeclaracionDePaso struct {
	Alcance AlcanceDelPaso
	// Motivo es obligatorio en PasoQueExigeSesion.
	Motivo string
	// Ordenes son las ordenes de terminal que ESTA pantalla exige teclear a
	// quien acaba de instalar. Cada una se comprueba contra el <main> de la
	// respuesta: una orden declarada que la pantalla no pide inflaria el coste,
	// y una que la pantalla pide y no se declara lo desinflaria.
	Ordenes []string
	// CuentaPreguntas dice si en esta pantalla se cuentan las preguntas de la
	// entrevista. Solo la de alcance.
	CuentaPreguntas bool
	// ExigeGuardado dice que esta pantalla, en el producto de verdad y con la
	// sesion abierta, tiene que ofrecer GUARDAR lo que se conteste.
	//
	// # Por que vive en esta medida y no solo en su suite
	//
	// El TTFV mide «que le pasa a quien se descarga esto un martes». Desde el
	// 04-09-2026 esa persona tambien vuelve al dia siguiente, y si el guardado
	// se desconectara del cableado (que es exactamente lo que le paso a la
	// ENTRADA hasta el 03-09-2026: las dos mitades en verde y la junta sin
	// poner) el numero de aqui no se movería ni un segundo, porque contestar
	// cuesta lo mismo se guarde o no. Un TTFV que no lo mira mediria un
	// recorrido que el dia dos no existe.
	//
	// Se comprueba por el FORMULARIO, no por un texto: lo que distingue una
	// pantalla que guarda de una que no es que la respuesta sea un POST.
	ExigeGuardado bool
	// Marca es un trozo de HTML que el <main> de este paso TIENE que traer
	// para que se acepte que ha entregado su valor.
	//
	// # Por que hace falta, y es la guarda de este bloque
	//
	// El 05-09-2026 dos pantallas dejaron de pedir su orden de terminal
	// MOVIENDO el bloque que la pedia a una pagina aparte, a la que se entra
	// a proposito. Eso es legitimo (es lo que D-13 bendice) y es tambien,
	// exactamente, la forma de aprobar esta puerta sin arreglar nada: basta
	// con sacar de <main> cualquier cosa que moleste.
	//
	// Lo que impide la trampa no es prohibir el movimiento, es EXIGIR QUE EL
	// PASO SIGA ENTREGANDO. Una pantalla que se vacia para pasar la puerta se
	// queda sin su marca y se pone roja; una que mueve una instruccion
	// opcional y sigue pintando su plan, su calendario o su acta, no.
	//
	// SE ELIGE UNA MARCA ESTRUCTURAL (la clase de la seccion que solo existe
	// con datos) y NO un texto del catalogo: un rotulo se cambia por gusto y
	// entonces la puerta se pone roja sin que nada este mal, y la reaccion
	// barata es aflojarla.
	Marca string
	// SubeCenso dice que en esta pantalla se SUBE el fichero de cuentas, en
	// vez de mirarla vacia.
	//
	// Lo destapo la Marca de arriba el mismo dia que se escribio: el paso de
	// la revision de accesos se estaba midiendo EN SU ESTADO VACIO, porque el
	// recorrido nunca subia un censo. Es exactamente el fallo que tenia la
	// entrevista hasta esta manana, una capa mas abajo: una medida que no
	// ejerce el sistema no lo mide, lo simula.
	//
	// Y LA SUBIDA SE COBRA (PreguntasDeLaSubida): elegir el fichero y decir de
	// que sistema es son dos respuestas, no cero. Ejercer el producto para
	// medirlo bien SUBE el numero, y eso es lo que tiene que pasar.
	SubeCenso bool
	// Publica dice que en esta pantalla se CONTESTA la entrevista y se
	// publica el alcance de la instalacion, en vez de solo mirarla.
	//
	// # Por que hacia falta, y es un fallo de la medida y no del producto
	//
	// Hasta el 05-09-2026 este recorrido hacia SEIS GET y nada mas. Con eso,
	// el calendario, el plan de avisos y el acta salian SIEMPRE en su estado
	// vacio, hiciera lo que hiciera el producto: nadie habia contestado
	// nada. O sea que las ordenes de terminal de esas pantallas eran
	// inevitables POR CONSTRUCCION de la medida, no del camino.
	//
	// El error iba a favor en la unica direccion que importa: hacia creer
	// que el cuello estaba en sitios donde no estaba, y habria dejado en
	// verde a un producto que publicara el alcance sin que se notara.
	//
	// Y LAS PREGUNTAS SE CUENTAN CONTESTANDO, no mirando la primera pagina:
	// la entrevista tiene revelacion progresiva, asi que contestar una
	// puede sacar otras. Contar solo las 19 de la primera pantalla es medir
	// una entrevista que nadie termina.
	Publica bool
}

// PasosDelCamino es el censo, por el ID del paso de camino.Canonico().
//
// SE CRUZA CON EL CAMINO EN LOS DOS SENTIDOS: un paso del camino que no salga
// aqui rompe la puerta, y una entrada de aqui que el camino ya no declare,
// tambien.
var PasosDelCamino = map[string]DeclaracionDePaso{
	camino.IDDelAlcance: {
		Alcance:         PasoAlcanzable,
		CuentaPreguntas: true,
		// La lista de preguntas: sin ella no hay entrevista que contestar.
		Marca:         `<ol class="preguntas">`,
		ExigeGuardado: true,
		Publica:       true,
	},
	camino.IDDelCalendario: {
		Alcance: PasoAlcanzable,
		// CERO ORDENES DESDE EL 05-09-2026, y aqui habia dos.
		//
		// Eran `plazum alcance` (componer el fichero) y
		// `plazum serve --alcance` (rearrancar el servidor con el), 3m0s
		// cobrados y el ultimo tramo de terminal del camino. Han desaparecido
		// porque la entrevista PUBLICA el alcance de la instalacion desde el
		// navegador, y el calendario lo lee de ahi.
		//
		// Y ESO OBLIGA A QUE ESTA MEDIDA CONTESTE LA ENTREVISTA DE VERDAD. Ver
		// DeclaracionDePaso.Publica: hasta hoy el recorrido solo MIRABA la
		// pantalla del alcance, asi que las de despues salian siempre vacias y
		// sus ordenes parecian inevitables.
		//
		// LO QUE QUEDA ES OTRA ORDEN, Y NO ES LA MISMA. Con el calendario ya
		// lleno aparece la que hasta ahora tapaba el estado vacio: la cifra que
		// no se puede abrir manda al terminal a ver la lista entera
		// (`plazum calendario --todos-los-relojes`). O sea que la ultima cifra
		// huerfana de D11-c y los ultimos 3m0s de D11-e son EL MISMO TRABAJO, y
		// eso no se sabia hasta que esta medida contesto la entrevista.
		// CERO DESDE EL 05-09-2026: la cifra que solo se abria con
		// `plazum calendario --todos-los-relojes` se abre ahora en su propia
		// pagina, a la que se entra a proposito (D-13). Es la casilla D11-c
		// cerrada, y resulto ser el mismo trabajo que este 1m30s.
		Ordenes: nil,
		// La cuenta con sus catorce cifras: solo se pinta con calendario.
		Marca: `<section class="cuenta">`,
	},
	camino.IDDeLaDerivacion: {
		Alcance: PasoAlcanzable,
		// La tabla de obligaciones, con su region enfocable alrededor.
		Marca: `<div class="marco-tabla"`,
	},
	camino.IDDelActa: {
		Alcance: PasoQueExigeSesion,
		Motivo: "el acta lleva quien audito que dentro de la organizacion, asi que no se " +
			"sirve a quien no ha entrado. Desde el 03-09-2026 entrar se puede: la " +
			"instalacion crea el primer administrador con el token que imprime el " +
			"arranque, y el coste de hacerlo esta dentro de esta medida (CosteDeInstalar).",
		// LA ORDEN DE SU ESTADO VACIO, que llevaba sin declarar desde que existe
		// esta medida. Eran DOS hasta el 04-09-2026, y las dos empezaban por
		// `plazum serve`: quien pegaba la segunda perdia las banderas de la
		// primera. Ahora es una sola, entera y pegable.
		// CERO ORDENES DESDE EL 05-09-2026, y aqui habia una.
		//
		// Era `plazum serve --acta-organizacion`, 1m30s cobrados. Ha desaparecido
		// porque de quien es el acta ya no se teclea: se contesta UNA VEZ en el
		// formulario del primer administrador, se guarda en la identidad de la
		// instalacion y el acta la lee en cada peticion. El periodo, que era la
		// otra mitad de esa orden, sale por defecto del ultimo trimestre natural
		// cerrado, que esta entero en el pasado.
		//
		// LO QUE SE HA MOVIDO NO ES GRATIS Y SE COBRA: el campo nuevo del
		// formulario de instalacion esta en PreguntasDelPrimerAdmin y suma 20s.
		Ordenes: nil,
		// Una seccion del acta: solo se pinta con acta compuesta.
		Marca: `<section class="seccion">`,
	},
	camino.IDDeLaUAR: {
		Alcance: PasoQueExigeSesion,
		Motivo: "la revision de accesos decide sobre accesos de personas con nombre, y sin " +
			"sesion no hay autor. Se recorre tras entrar, igual que el acta.",
		// CERO ORDENES DESDE EL 05-09-2026, y aqui habia dos.
		//
		// Eran `plazum accesos ver` (sellar la campana) y
		// `plazum serve --accesos-fichero` (servirla), 3m0s cobrados. Han
		// desaparecido porque el censo se sube por el navegador: el estado
		// vacio de esta pantalla pinta un formulario que envia a /uar/abrir, y
		// el adaptador que escribe en <datos>/accesos hace lo mismo que hacian
		// las dos ordenes.
		//
		SubeCenso: true,
		// Los cubos de la campana: solo se pintan con campana abierta.
		Marca: `<section class="cubos">`,
		// NO SE HA QUITADO NADA DE LA COLUMNA PARA QUE BAJE EL NUMERO: la
		// direccion contraria de esta misma puerta (lo que la pantalla pinta y
		// el censo calla) recorre el <main> y contaria cualquier invocacion que
		// quedara. Se comprobo con las dos: al quitar las ordenes de la
		// plantilla y no de aqui, la puerta se puso roja por los dos lados a la
		// vez.
		Ordenes: nil,
	},
	camino.IDDelEscalado: {
		Alcance: PasoQueExigeSesion,
		Motivo: "el plan de avisos dice a quien escribiria plazum, o sea nombres de personas. " +
			"Se recorre tras entrar, igual que el acta.",
		// AQUI HABIA LAS MISMAS DOS DEL CALENDARIO, y se han ido con el cable
		// del alcance publicado. Lo que queda es su propia puerta al terminal,
		// la hermana de la del calendario: con el plan ya lleno, la cifra que
		// no se abre manda a `plazum escalado` a ver la lista entera.
		//
		// SE DECLARA AUNQUE NO SE COBRE HOY, porque quien recorre el camino en
		// orden ya ha tecleado la del calendario cuando llega aqui... y NO es la
		// misma cadena, asi que el deduplicado no la ve y se cobra otra vez. Es
		// un hallazgo de producto y no un ajuste de la medida: son dos salidas
		// al terminal de verdad, y cobrar de mas se dice, no se corrige a mano.
		// CERO DESDE EL 05-09-2026: el bloque de «como se mandan de verdad» se
		// fue a su propia pagina. NO es la misma clase de arreglo que el del
		// calendario y conviene no confundirlos: alli habia una lista que el
		// producto ya tenia calculada y no ensenaba; aqui hay una orden que
		// sigue siendo la unica forma de mandar avisos, y lo que se dice es
		// que no hace falta para llegar al valor de esta pantalla, que es ver
		// a quien avisaria plazum.
		//
		// Lo que impide que esto sea una trampa es Marca: el plan tiene que
		// seguir pintandose. Vaciar la pantalla para quitarse la orden pone
		// esta puerta roja.
		Ordenes: nil,
		// La linea de «de quien es este plan»: solo sale con plan.
		Marca: `<p class="identidad">`,
	},
}

// MedidaDeUnPaso es lo que se saca de recorrer un paso.
type MedidaDeUnPaso struct {
	ID        string
	Ruta      string
	Codigo    int
	Latencia  time.Duration
	Preguntas int
	// PreguntasHumanas son las que se COBRAN al precio de una pregunta, que no
	// son exactamente las de la pantalla: incluye el clic de publicar el
	// alcance de la instalacion.
	//
	// SON DOS CAMPOS Y NO UNO PORQUE SON DOS COSAS. Preguntas es lo que la
	// pantalla pinta y se contrasta contra ella; esto es lo que cuesta. Con un
	// solo campo, o el contraste con la pantalla se rompe o el desglose deja
	// de sumar el total, y esto ultimo es lo que estaba pasando: 20s vivian en
	// el total y no en el reparto, o sea que el desglose publicado no cuadraba
	// con la cifra publicada al lado.
	PreguntasHumanas int
	// Ordenes son las que esta pantalla pide.
	Ordenes int
	// OrdenesCobradas son las que esta pantalla pide Y NADIE HA TECLEADO YA.
	//
	// # Por que se deduplican, y por que eso NO es rebajar el coste
	//
	// El calendario y el plan de avisos piden LA MISMA pareja de ordenes, y una
	// persona que recorre el camino las teclea UNA VEZ: cuando llega al plan de
	// avisos ya arranco el servidor con --alcance. Cobrarlas otra vez mide la
	// PANTALLA y no a la persona, y lo que este numero decide es cuanto tarda
	// una persona.
	//
	// Lo que esta prohibido es bajar el coste POR orden, o sea negociar la
	// constante hasta que salga el numero. Esto es lo contrario: se cobra
	// entero, una vez, y las dos cifras se imprimen para que nadie tenga que
	// creerse el deduplicado.
	OrdenesCobradas int
	Alcanzado       bool
	CosteHuman      time.Duration
}

// TestTTFVDelCaminoCompleto es la puerta D11-e, y la mitad medible de D11-a.
func TestTTFVDelCaminoCompleto(t *testing.T) {
	pasos := camino.Canonico()
	if len(pasos) < 6 {
		t.Fatalf("el camino declara %d pasos y hoy son 6: esta puerta estaria midiendo un "+
			"camino que no es el del producto", len(pasos))
	}
	cruzarElCensoDePasos(t, pasos)

	dir := t.TempDir()
	binario := filepath.Join(dir, "plazum"+extensionDeBinario())

	// 1. CONSEGUIR EL BINARIO. En una release esto es una descarga; compilar es
	//    la cota superior de las dos, asi que medir esto no favorece al numero.
	tBinario := cronometrar(func() {
		cmd := exec.Command("go", "build", "-o", binario, "./cmd/plazum")
		if salida, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("no se puede compilar el binario, asi que este TTFV no mide el "+
				"producto sino la maquina: %v\n%s", err, salida)
		}
	})

	// 2. ARRANCAR, hasta que el servidor contesta.
	srv := arrancarServidor(t, binario)
	defer srv.parar()

	// 3. INSTALAR: leer el token del terminal y crear el primer administrador.
	//    Es un paso REAL del recorrido desde que existe el almacen de usuarios,
	//    asi que cuesta maquina y cuesta humano, y las dos cosas se suman.
	tInstalacion := srv.instalar(t)

	// 4. RECORRER LOS SEIS PASOS, en su orden y con la sesion abierta.
	var tPasos, costeHumano time.Duration
	var costeDeLectura, costeDeEntrevista, costeDeOrdenes time.Duration
	medidas := make([]MedidaDeUnPaso, 0, len(pasos))
	alcanzados, exigenSesion := 0, 0
	// LO QUE YA SE HA TECLEADO. Ver MedidaDeUnPaso.OrdenesCobradas: una persona
	// no teclea dos veces la misma orden porque dos pantallas se la pidan.
	yaTecleado := map[string]bool{}
	// Y LOS SUBCOMANDOS DEL BINARIO, para poder mirar la direccion contraria.
	subcomandos := subcomandosDelBinario(t, binario)
	for _, p := range pasos {
		if !p.EsPantalla() {
			// Un paso que todavia es de terminal cuesta un tecleo y no una
			// peticion. Hoy no hay ninguno, y el cardinal de arriba lo dice.
			continue
		}
		m := recorrerUnPaso(t, srv, p, yaTecleado, subcomandos)
		tPasos += m.Latencia
		costeHumano += m.CosteHuman
		// EL REPARTO, para que el cuello se derive y no se escriba.
		costeDeLectura += CosteDeLeerUnaPantalla
		// EL REPARTO CUENTA LO MISMO QUE EL TOTAL, incluido el clic de
		// publicar. m.Preguntas son las preguntas de la pantalla; PreguntasHumanas
		// son esas mas lo que se cobra al mismo precio sin ser una pregunta.
		costeDeEntrevista += time.Duration(m.PreguntasHumanas) * CosteDeResponderUnaPregunta
		costeDeOrdenes += time.Duration(m.OrdenesCobradas) * CosteDeTeclearUnaOrden
		medidas = append(medidas, m)
		if m.Alcanzado {
			alcanzados++
		}
		if PasosDelCamino[p.ID].Alcance == PasoQueExigeSesion {
			exigenSesion++
		}
	}

	tMaquina := tBinario + srv.tArranque + tInstalacion + tPasos
	// LA INSTALACION, con sus preguntas dentro. El desglose de abajo la
	// imprime como una sola partida a proposito: quien lee el numero quiere
	// saber cuanto cuesta poner esto en marcha, no como se reparte entre leer
	// un recuadro y teclear un nombre.
	costeDeInstalacion := CosteDeInstalar +
		time.Duration(PreguntasDelPrimerAdmin)*CosteDeResponderUnaPregunta
	costeHumano += costeDeInstalacion

	// EL DESGLOSE TIENE QUE SUMAR EL TOTAL, y esta puerta faltaba.
	//
	// Este fichero imprime dos cosas al lado: una CIFRA (T_humano) y un
	// REPARTO (lectura, entrevista, ordenes, instalacion). Las dos se calculan
	// por caminos distintos a proposito -- el total suma el coste de cada paso,
	// el reparto suma por partidas -- y hasta hoy nadie comprobaba que dieran
	// lo mismo.
	//
	// NO ES HIPOTETICO: el 05-09-2026 discrepaban en 20s. El clic de publicar
	// el alcance de la instalacion entraba en el coste del paso y no en
	// ninguna partida, asi que el numero publicado era 16m30s y el desglose
	// que lo explicaba sumaba 16m10s. Es la familia de la afirmacion
	// acompanada en su forma mas incomoda: las dos mitades son cifras, las dos
	// se publican juntas, y la que se cree quien lee es la que cuadra.
	//
	// Y ademas es la puerta que impide la trampa comoda: meter una partida
	// nueva en el total sin darle sitio en el reparto esconde coste en un sitio
	// donde nadie lo busca, porque el reparto es lo que se lee para saber que
	// arreglar.
	if suma := costeDeLectura + costeDeEntrevista + costeDeOrdenes + costeDeInstalacion; suma != costeHumano {
		t.Errorf("el coste humano es %s y su desglose suma %s: se han perdido %s por el "+
			"camino.\n"+
			"  reparto: lectura %s, entrevista %s, ordenes %s, instalacion %s\n"+
			"  Las dos cifras se publican juntas, asi que una de las dos miente y quien lea "+
			"se creera la que cuadre. Arreglo: toda partida que entre en el total tiene que "+
			"tener sitio en el reparto.",
			costeHumano, suma, (costeHumano - suma).Abs(),
			costeDeLectura, costeDeEntrevista, costeDeOrdenes, costeDeInstalacion)
	}
	total := tMaquina + costeHumano

	// EL DESGLOSE SE IMPRIME SIEMPRE, en verde y en rojo: un numero sin
	// desglose no se puede recontar, y un TTFV que nadie puede recontar no vale.
	t.Logf("TTFV del camino guiado, desglose\n"+
		"  MODELO: TTFV = T_maquina + T_humano; lectura %s, respuesta %s, orden %s, "+
		"instalacion %s\n"+
		"  T_maquina %s  = binario %s + arranque %s + instalacion %s + peticiones %s\n"+
		"  T_humano  %s\n"+
		"  TOTAL     %s  (presupuesto %s)\n"+
		"  pasos alcanzados %d de %d; exigen sesion %d",
		CosteDeLeerUnaPantalla, CosteDeResponderUnaPregunta, CosteDeTeclearUnaOrden,
		costeDeInstalacion,
		tMaquina.Round(time.Millisecond), tBinario.Round(time.Millisecond),
		srv.tArranque.Round(time.Millisecond), tInstalacion.Round(time.Millisecond),
		tPasos.Round(time.Millisecond),
		costeHumano, total.Round(time.Second), PresupuestoTTFV,
		alcanzados, len(medidas), exigenSesion)
	for _, m := range medidas {
		t.Logf("  paso %-12s %-14s codigo %d  latencia %s  preguntas %2d  ordenes %d "+
			"(cobradas %d)  coste humano %s",
			m.ID, m.Ruta, m.Codigo, m.Latencia.Round(time.Millisecond), m.Preguntas,
			m.Ordenes, m.OrdenesCobradas, m.CosteHuman)
	}

	// EL CARDINAL DE LAS PANTALLAS QUE EXIGEN SESION, topado en los dos sentidos.
	if exigenSesion != PasosQueExigenSesion {
		t.Errorf("hay %d pasos del camino que solo se sirven a quien ha entrado, y "+
			"PasosQueExigenSesion dice %d.\n"+
			"  Si han SUBIDO, hay una pantalla mas cerrada y hay que decir por que.\n"+
			"  Si han BAJADO, alguien le ha quitado la sesion a una de las tres pantallas "+
			"cuyo contenido son personas con nombre.", exigenSesion, PasosQueExigenSesion)
	}
	// Y EL CAMINO ENTERO SE RECORRE. Es el criterio de exito del producto, y sin
	// esta linea el TTFV podria seguir midiendo medio camino y saliendo bonito.
	if alcanzados != len(medidas) {
		t.Errorf("se han recorrido %d de los %d pasos del camino. Un camino guiado con un "+
			"tramo que nadie puede andar no es un camino: es una lista de pantallas",
			alcanzados, len(medidas))
	}
	// EL PRESUPUESTO, Y AHORA SI ES UNA PUERTA.
	//
	// Mientras la casilla no se cumplia, esto no podia ser un error sin dejar
	// la suite en rojo permanente, y por eso habia un techo aparte. Desde que
	// se cumple, lo que hay que vigilar es que no se pierda: cualquier cosa que
	// devuelva el camino por encima de los quince minutos se pone roja aqui.
	// EL TECHO, EN LAS DOS DIRECCIONES. Por arriba, porque un TTFV que crece se
	// pone rojo antes de que nadie lo note. Por abajo, porque el dia que baje de
	// verdad hay que BORRAR el techo en el mismo commit: un hueco que se cierra
	// y deja su cardinal puesto miente hacia arriba para siempre.
	if total > TechoDeclaradoTTFV {
		t.Errorf("el TTFV es %s y el techo declarado es %s.\n"+
			"  Algo ha devuelto el camino al terminal, o ha crecido la entrevista. El arreglo "+
			"NO es subir esta constante: es mirar el desglose de arriba y ver que linea "+
			"crecio.", total.Round(time.Second), TechoDeclaradoTTFV)
	}
	// El margen es de unas tres preguntas, a proposito. Menos que eso y la
	// medida se pondria roja por el ruido de la maquina; mas, y anadir media
	// entrevista dejaria de notarse.
	const margenDelTecho = 90 * time.Second
	if total < TechoDeclaradoTTFV-margenDelTecho {
		t.Errorf("el TTFV es %s y el techo declarado sigue en %s, con mas de %s de margen.\n"+
			"  El numero ha bajado y el techo no se ha bajado con el: un techo que sobra "+
			"miente hacia arriba y deja de avisar. Se baja en el mismo commit que lo mejora.",
			total.Round(time.Second), TechoDeclaradoTTFV, margenDelTecho)
	}
	if total > PresupuestoTTFV {
		// DESDE EL 05-09-2026 ESTO ES UN ERROR Y NO UN AVISO.
		//
		// Mientras la casilla no se cumplia, un error aqui habria dejado la
		// suite en rojo permanente, que es como se ensena a ignorar una
		// puerta; lo que se hacia era decirlo en voz alta en cada ejecucion.
		// Ahora se cumple, y lo que hay que vigilar es que no se pierda: el
		// presupuesto es una promesa al usuario, y volver a cruzarlo es una
		// regresion del producto, no una nota informativa.
		//
		// Y EL CUELLO SE DERIVA, NO SE ESCRIBE. Esta linea decia «el cuello de
		// botella es la entrevista de /alcance» con el numero al lado, y el
		// numero se corregia solo mientras la frase se quedaba puesta: desde
		// que la medida cuenta las ordenes que faltaban, el cuello son LAS
		// ORDENES y no la entrevista. Es la familia de la afirmacion
		// acompanada, con la prosa como parte que caduca porque nadie la
		// vigila. Ahora la frase se compone del mismo reparto que se imprime
		// arriba, asi que no puede describir un mundo que ya no existe.
		t.Errorf("LA CASILLA D11-e SE HA PERDIDO: %s sobre un presupuesto de %s, sobre los %d "+
			"pasos del camino entero.\n"+
			"  EL CUELLO, derivado del reparto y no escrito: %s.\n"+
			"  reparto del coste humano: lectura %s, entrevista %s, ordenes %s, instalacion %s\n"+
			"  Ver docs/hallazgos-d11.md",
			total.Round(time.Second), PresupuestoTTFV, alcanzados,
			elCuello(costeDeLectura, costeDeEntrevista, costeDeOrdenes, costeDeInstalacion),
			costeDeLectura, costeDeEntrevista, costeDeOrdenes, costeDeInstalacion)
	}
}

// elCuello dice cual de las cuatro partidas del coste humano es la mayor.
//
// SE DERIVA DEL REPARTO MEDIDO. Un cuello escrito a mano al lado de un numero
// que se corrige solo acaba describiendo un mundo que ya no existe, y este
// fichero ya lo hizo: decia «la entrevista» mientras las ordenes pasaban a ser
// la partida mas cara.
func elCuello(lectura, entrevista, ordenes, instalacion time.Duration) string {
	partidas := []struct {
		nombre string
		coste  time.Duration
	}{
		{"las ordenes de terminal de los estados vacios", ordenes},
		{"la entrevista de /alcance", entrevista},
		{"la lectura de las pantallas", lectura},
		{"la instalacion", instalacion},
	}
	mayor := partidas[0]
	total := time.Duration(0)
	for _, x := range partidas {
		total += x.coste
		if x.coste > mayor.coste {
			mayor = x
		}
	}
	if total == 0 {
		return "no hay coste humano que repartir, asi que esta medida no mide nada"
	}
	return fmt.Sprintf("%s, %s de %s (%.0f %%)",
		mayor.nombre, mayor.coste, total, 100*float64(mayor.coste)/float64(total))
}

// cruzarElCensoDePasos comprueba que el censo y el camino dicen lo mismo, en
// los dos sentidos, y que ninguna entrada llega con el valor cero.
func cruzarElCensoDePasos(t *testing.T, pasos []camino.Paso) {
	t.Helper()
	enElCamino := map[string]bool{}
	for _, p := range pasos {
		enElCamino[p.ID] = true
		d, hay := PasosDelCamino[p.ID]
		if !hay {
			t.Errorf("el camino declara el paso %q y el censo del TTFV no lo tiene.\n"+
				"  Sin declararlo, el tiempo hasta el valor se estaria midiendo sobre un "+
				"camino mas corto que el que el producto ensena.", p.ID)
			continue
		}
		if d.Alcance == PasoSinDeclarar {
			t.Errorf("el paso %q esta en el censo con el VALOR CERO (%s). El cero no es un "+
				"estado, es el olvido", p.ID, d.Alcance)
		}
		if d.Alcance == PasoQueExigeSesion && strings.TrimSpace(d.Motivo) == "" {
			t.Errorf("el paso %q se declara %q y no dice por que. Un tramo del camino que no "+
				"se puede recorrer y no explica su bloqueo se lee como una decision",
				p.ID, d.Alcance)
		}
	}
	for id := range PasosDelCamino {
		if !enElCamino[id] {
			t.Errorf("el censo del TTFV declara el paso %q y camino.Canonico() ya no lo "+
				"tiene: la medida se ha quedado vieja", id)
		}
	}
}

// recorrerUnPaso pide la ruta del paso y saca de la RESPUESTA todo lo que se
// puede sacar de ella.
func recorrerUnPaso(t *testing.T, s *servidorDePruebaTTFV, p camino.Paso,
	yaTecleado map[string]bool, subcomandos []string) MedidaDeUnPaso {
	t.Helper()
	d := PasosDelCamino[p.ID]
	m := MedidaDeUnPaso{ID: p.ID, Ruta: p.Ruta}

	inicio := time.Now()
	// CON LA SESION DEL ADMINISTRADOR, que es lo que tiene una persona que acaba
	// de instalar. Antes se pedia con http.Get a pelo, y por eso tres pasos
	// salian 401 y quedaban fuera de la medida.
	resp, err := s.cli.Get(s.base + p.Ruta)
	m.Latencia = time.Since(inicio)
	if err != nil {
		t.Fatalf("pidiendo el paso %q en %s: %v", p.ID, p.Ruta, err)
	}
	cuerpo, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	m.Codigo = resp.StatusCode
	pagina := string(cuerpo)

	// TODO PASO DEL CAMINO SE RECORRE, y lo declarado dice ademas si hace falta
	// sesion para ello.
	m.Alcanzado = m.Codigo == http.StatusOK
	if !m.Alcanzado {
		t.Errorf("el paso %q (%s) contesta %d en una instalacion recien hecha, con la sesion "+
			"del administrador abierta. El camino guiado tiene un tramo que nadie puede "+
			"andar", p.ID, p.Ruta, m.Codigo)
		return m
	}
	// EL CONTROL POSITIVO DEL DESCARGO. Un paso declarado PasoQueExigeSesion
	// tiene que seguir sin servirse SIN cookie; si no, el arreglo mas comodo
	// para que esta medida salga entera seria abrir las tres pantallas que
	// llevan nombres de personas, y nada se pondria rojo.
	if d.Alcance == PasoQueExigeSesion {
		sinSesion, err := s.crudo.Get(s.base + p.Ruta)
		if err != nil {
			t.Fatalf("pidiendo el paso %q sin sesion: %v", p.ID, err)
		}
		codigo := sinSesion.StatusCode
		_ = sinSesion.Body.Close()
		if codigo == http.StatusOK {
			t.Errorf("el paso %q (%s) se sirve con 200 SIN sesion, y esa pantalla lleva "+
				"nombres de personas dentro", p.ID, p.Ruta)
		}
	}

	// SE CONTESTA LA ENTREVISTA Y SE PUBLICA, que es lo que hace una persona
	// y lo que esta medida no hacia. Ver DeclaracionDePaso.Publica. Se hace
	// ANTES de contar, porque contestar revela preguntas nuevas.
	if d.Publica {
		pagina, m.Preguntas = s.contestarYPublicar(t, p.Ruta, pagina)
	}
	// EL CENSO SE SUBE, que es lo que hace una persona y lo que esta medida no
	// hacia: sin esto, la revision de accesos y el acta que se compone de ella
	// se miden en su estado vacio.
	if d.SubeCenso {
		pagina = s.subirElCenso(t, p.Ruta, pagina)
	}
	principal := entreMain(t, p.ID, pagina)
	if d.CuentaPreguntas {
		// LAS PREGUNTAS SE CUENTAN DE LA PAGINA, no de una lista escrita aqui:
		// dependen del corpus instalado, y escribir el numero lo dejaria viejo
		// el dia que entre el marco trece.
		//
		// EL PATRON NO LLEVA LA COMILLA DE CIERRE, Y ESO ERA UN FALLO REAL.
		// Contaba `<li class="pregunta"`, con comilla, asi que NO casaba con la
		// pregunta sugerida, que se pinta `class="pregunta sugerida"`. La
		// medida venia saliendo con UNA PREGUNTA DE MENOS desde que existe, y
		// nadie lo noto porque el error iba en la direccion comoda: veinte
		// segundos menos en el total. Se descubrio al bajar la entrevista de 42
		// a 19 y no cuadrar el 19 con el 18 que decia la pagina.
		//
		// La leccion es la de siempre: un patron que casa con el ATRIBUTO
		// entero se rompe en silencio el dia que alguien anade una clase, y
		// romperse en silencio hacia abajo es peor que romperse.
		// SI SE HA CONTESTADO, EL NUMERO YA ESTA Y ES MAYOR: son todas las que
		// se han visto a lo largo del recorrido, no las de la primera pantalla.
		// Volver a contar aqui las machacaria con las de la ultima pagina, que
		// es justo la trampa: contestadas todas, la pagina ensena menos.
		if !d.Publica {
			m.Preguntas = strings.Count(principal, `<li class="pregunta`)
		}
		if m.Preguntas == 0 {
			t.Errorf("el paso %q dice contar preguntas y la pantalla no pinta ninguna: el "+
				"coste humano del camino saldria de una entrevista vacia", p.ID)
		}
	}
	// EL GUARDADO, sobre el binario de verdad y con la sesion abierta. Ver
	// DeclaracionDePaso.ExigeGuardado: sin esto, desconectar el cableado del
	// almacen no movería este numero ni un segundo y la medida seguiria
	// diciendo que el recorrido esta entero.
	if d.ExigeGuardado {
		if !strings.Contains(principal, `name="accion"`) {
			t.Errorf("el paso %q no ofrece guardar lo que se conteste: la entrevista sigue "+
				"siendo enlaces y las respuestas se pierden al cerrar el navegador.\n"+
				"  Este TTFV mide el primer dia; sin guardado, el segundo dia empieza otra vez "+
				"desde cero.", p.ID)
		}
	}
	// LA MARCA: QUE ESTE PASO HAYA ENTREGADO SU VALOR.
	//
	// Va ANTES de contar ordenes a proposito, porque es lo que da sentido a
	// contar cero: una pantalla vacia tambien tiene cero ordenes. Sin esto, la
	// forma mas barata de aprobar esta puerta es dejar de pintar cosas.
	if d.Marca != "" && !strings.Contains(principal, d.Marca) {
		t.Errorf("el paso %q no trae %s en su <main>, asi que no ha entregado su valor.\n"+
			"  Un paso vacio tiene cero ordenes de terminal y cero preguntas, o sea que "+
			"sale barato en esta medida. Esta linea existe para que salir barato por "+
			"estar vacio no cuente como salir barato.\n--- <main> ---\n%s",
			p.ID, d.Marca, recortar(principal))
	}
	// LAS ORDENES, EN LAS DOS DIRECCIONES, Y LA SEGUNDA FALTABA.
	//
	// La que habia recorria la lista DECLARADA y la buscaba en la pantalla, o
	// sea que cazaba una orden declarada de mas (coste inflado) y NO cazaba una
	// orden que la pantalla pide y nadie declara (coste desinflado). Es el
	// patron que este repositorio persigue en el corpus y en el expediente
	// («cuando una comprobacion recorre una lista para contrastarla con otra,
	// preguntar SIEMPRE si la direccion contraria tambien se recorre»), aplicado
	// a la unica cifra que mide el producto de cara al comprador.
	//
	// Y el error iba SIEMPRE A FAVOR, que es la senal: el acta y el plan de
	// avisos pedian dos ordenes cada uno en sus estados vacios, ninguna estaba
	// declarada, y la medida publicada llevaba SEIS MINUTOS de menos desde que
	// existe. Una cifra cuyo fallo probable es favorecerte necesita puerta en
	// las dos direcciones.
	for _, orden := range d.Ordenes {
		if !strings.Contains(principal, orden) {
			t.Errorf("el paso %q declara que hay que teclear %q y su pantalla no lo pide.\n"+
				"  O el coste humano esta inflado, o la pantalla ha dejado de decir como se "+
				"sale de su estado vacio (puerta D11-b).", p.ID, orden)
			continue
		}
		m.Ordenes++
		// EL DEDUPLICADO: se cobra la primera vez que aparece en el recorrido.
		if !yaTecleado[orden] {
			yaTecleado[orden] = true
			m.OrdenesCobradas++
		}
	}
	// LA DIRECCION CONTRARIA: lo que la pantalla pide y el censo no dice.
	if vistas := lasOrdenesDeLaPantalla(principal, subcomandos); len(vistas) != len(d.Ordenes) {
		t.Errorf("el paso %q pinta %v en su <main> y el censo declara "+
			"%d.\n"+
			"  Cada una es una SALIDA DEL PRODUCTO y cuesta %s de persona. Una que la pantalla "+
			"pide y el censo calla desinfla el TTFV, y ese error va siempre a favor.\n"+
			"  Arreglo: declararla en PasosDelCamino, o quitarla de la pantalla dandole un "+
			"camino que no pase por el terminal.", p.ID, vistas, len(d.Ordenes),
			CosteDeTeclearUnaOrden)
	}
	m.PreguntasHumanas = m.Preguntas
	if d.Publica {
		// EL BOTON DE PUBLICAR SE COBRA. Ver PreguntasDeLaPublicacion.
		m.PreguntasHumanas += PreguntasDeLaPublicacion
	}
	if d.SubeCenso {
		// Y LA SUBIDA TAMBIEN. Ver PreguntasDeLaSubida.
		m.PreguntasHumanas += PreguntasDeLaSubida
	}
	m.CosteHuman = CosteDeLeerUnaPantalla +
		time.Duration(m.PreguntasHumanas)*CosteDeResponderUnaPregunta +
		time.Duration(m.OrdenesCobradas)*CosteDeTeclearUnaOrden
	return m
}

// lasOrdenesDeLaPantalla cuenta las invocaciones del binario que hay en el <main>.
//
// # Como se distingue una orden de una frase que nombra el producto
//
// Por el SUBCOMANDO, y la lista de subcomandos NO se escribe aqui: se saca de la
// ayuda del propio binario. Escrita aqui seria una segunda copia que se queda
// vieja el dia que el producto estrene una orden, y el sintoma seria una orden
// que la pantalla pide y esta puerta no ve, o sea justo lo que viene a cazar.
//
// Hace falta porque el catalogo tiene frases que nombran a plazum sin pedir nada
// («plazum todavia no ha mirado si alguna de estas te alcanza»), y acusarlas
// seria el falso positivo que acaba con una puerta borrada.
//
// # Y DICE CUALES SON, no solo cuantas
//
// El cardinal solo no arregla nada: cuando esta puerta dice «pinta 1 y el
// censo declara 0», lo primero que hace falta saber es CUAL, y sin esto hay
// que ir a buscarla a mano por las plantillas y por el catalogo, que es donde
// suele estar. Un mensaje que obliga a reproducir el fallo para entenderlo es
// medio mensaje.
func lasOrdenesDeLaPantalla(principal string, subcomandos []string) []string {
	var out []string
	for _, sc := range subcomandos {
		for i := 0; i < strings.Count(principal, "plazum "+sc); i++ {
			out = append(out, "plazum "+sc)
		}
	}
	sort.Strings(out)
	return out
}

// reSubcomando casa una linea de la ayuda del binario: `     plazum <verbo>`.
var reSubcomando = regexp.MustCompile(`(?m)^\s+plazum ([a-z][a-z-]+)\b`)

// subcomandosDelBinario saca de la ayuda los verbos que el producto acepta.
//
// SE PARA SI NO ENCUENTRA NINGUNO, y ese suelo no es adorno: sin subcomandos, el
// contraste de arriba contaria cero ordenes en todas las pantallas y daria verde
// sobre un camino lleno de salidas al terminal, que es el verde vacio de siempre.
func subcomandosDelBinario(t *testing.T, binario string) []string {
	t.Helper()
	// #nosec G204 -- el binario lo acaba de compilar este mismo test en su
	// directorio temporal, y el argumento es una constante de aqui.
	salida, _ := exec.Command(binario, "--help").CombinedOutput()
	visto := map[string]bool{}
	var out []string
	for _, m := range reSubcomando.FindAllStringSubmatch(string(salida), -1) {
		if !visto[m[1]] {
			visto[m[1]] = true
			out = append(out, m[1])
		}
	}
	if len(out) < 8 {
		t.Fatalf("la ayuda del binario declara %d subcomandos y son muchos menos de los que "+
			"tiene: sin ellos, el contraste de ordenes contaria cero en todas las pantallas "+
			"y daria verde sobre un camino lleno de salidas al terminal.\n"+
			"--- ayuda ---\n%s", len(out), salida)
	}
	return out
}

// entreMain recorta lo que escribe la pantalla. Se para si no casa: contar
// preguntas u ordenes en la pagina entera contaria tambien lo que pinta el
// armazon, que sale en todas.
func entreMain(t *testing.T, id, pagina string) string {
	t.Helper()
	i := strings.Index(pagina, "<main ")
	j := strings.Index(pagina, "</main>")
	if i < 0 || j < i {
		t.Fatalf("la respuesta del paso %q no trae <main>...</main>: esta medida estaria "+
			"contando el armazon", id)
	}
	return pagina[i:j]
}

// --- el servidor de verdad, como proceso ---

type servidorDePruebaTTFV struct {
	cmd       *exec.Cmd
	base      string
	tArranque time.Duration
	mu        sync.Mutex
	salida    strings.Builder
	// cli lleva la sesion del administrador; crudo no lleva ninguna. Ninguno de
	// los dos sigue redirecciones: con un cliente que las siguiera, el 303 a la
	// instalacion se leeria como el 200 de la pagina de destino y esta medida
	// contaria pantallas que nadie estaba sirviendo.
	cli   *http.Client
	crudo *http.Client
}

// reTokenDeInstalacionTTFV casa el token que imprime el arranque: 64 caracteres
// hexadecimales solos en su linea.
var reTokenDeInstalacionTTFV = regexp.MustCompile(`(?m)^\s*([0-9a-f]{64})\s*$`)

// reCampoCSRFTTFV saca el campo oculto del formulario. El nombre del campo NO se
// escribe aqui: sale de serve.CampoCSRF, que es quien lo declara.
var reCampoCSRFTTFV = regexp.MustCompile(`name="` + regexp.QuoteMeta(serve.CampoCSRF) +
	`" value="([0-9a-f]+)"`)

// instalar hace lo que hace una persona la primera vez: leer el recuadro del
// token en el terminal, abrir /primer-admin, pegarlo y elegir credenciales.
// Devuelve lo que costo de maquina; lo que cuesta de humano es CosteDeInstalar.
//
// SI ESTO FALLA, EL PRODUCTO NO SE PUEDE INSTALAR, y eso es mas grave que un
// TTFV largo: por eso se para en vez de seguir midiendo medio camino.
func (s *servidorDePruebaTTFV) instalar(t *testing.T) time.Duration {
	t.Helper()
	inicio := time.Now()

	s.mu.Lock()
	arranque := s.salida.String()
	s.mu.Unlock()
	tok := reTokenDeInstalacionTTFV.FindStringSubmatch(arranque)
	if tok == nil {
		t.Fatalf("`plazum serve` no ha impreso ningun token de primer administrador al "+
			"arrancar sobre un directorio vacio, asi que no hay forma de instalar el "+
			"producto ni de entrar en el.\n--- salida del arranque ---\n%s", arranque)
	}

	resp, err := s.cli.Get(s.base + "/primer-admin")
	if err != nil {
		t.Fatalf("abriendo /primer-admin: %v", err)
	}
	cuerpo, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /primer-admin contesta %d en una instalacion nueva. Sin esa pantalla "+
			"no hay forma de entrar en plazum.\n%s", resp.StatusCode, cuerpo)
	}
	csrf := reCampoCSRFTTFV.FindStringSubmatch(string(cuerpo))
	if csrf == nil {
		t.Fatalf("el formulario de primer administrador no trae el campo %q, asi que no se "+
			"puede enviar.\n%s", serve.CampoCSRF, cuerpo)
	}

	alta, err := s.cli.PostForm(s.base+"/primer-admin", url.Values{
		serve.CampoCSRF: {csrf[1]},
		"token":         {tok[1]},
		"usuario":       {"ciso"},
		"secreto":       {"contrasena-de-prueba-1"},
		// EL NOMBRE DE LA ORGANIZACION. Es un campo mas del mismo
		// formulario, y SE COBRA: ver PreguntasDelPrimerAdmin.
		"organizacion": {"Ejemplo SL"},
	})
	if err != nil {
		t.Fatalf("enviando el formulario de primer administrador: %v", err)
	}
	respuesta, _ := io.ReadAll(alta.Body)
	_ = alta.Body.Close()
	if alta.StatusCode != http.StatusSeeOther {
		t.Fatalf("POST /primer-admin contesta %d y tenia que crear el administrador y "+
			"redirigir.\n%s", alta.StatusCode, respuesta)
	}
	return time.Since(inicio)
}

func (s *servidorDePruebaTTFV) parar() {
	if s.cmd != nil && s.cmd.Process != nil {
		_ = s.cmd.Process.Kill()
		_ = s.cmd.Wait()
	}
}

func arrancarServidor(t *testing.T, binario string) *servidorDePruebaTTFV {
	t.Helper()
	puerto := puertoLibre(t)
	direccion := fmt.Sprintf("127.0.0.1:%d", puerto)
	// EL CORPUS ES EL DEL REPOSITORIO, que es el que se publica. Con un corpus
	// vacio la entrevista no tendria preguntas y el coste humano saldria cero,
	// que es la forma mas comoda de aprobar esta puerta sin merecerlo.
	corpus, err := filepath.Abs("paquetes")
	if err != nil {
		t.Fatalf("resolviendo el directorio del corpus: %v", err)
	}
	tarro, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("construyendo el tarro de galletas: %v", err)
	}
	sinSeguir := func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	s := &servidorDePruebaTTFV{
		base:  "http://" + direccion,
		cli:   &http.Client{Timeout: 30 * time.Second, Jar: tarro, CheckRedirect: sinSeguir},
		crudo: &http.Client{Timeout: 30 * time.Second, CheckRedirect: sinSeguir},
	}
	// #nosec G204 -- el binario lo acaba de compilar este mismo test en su
	// directorio temporal y los argumentos son constantes de aqui: no hay
	// entrada externa en esta linea.
	s.cmd = exec.Command(binario, "serve", "--direccion", direccion, "--corpus", corpus)
	s.cmd.Dir = t.TempDir()
	tuberia, err := s.cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("abriendo la salida del servidor: %v", err)
	}
	s.cmd.Stderr = os.Stderr
	inicio := time.Now()
	if err := s.cmd.Start(); err != nil {
		t.Fatalf("arrancando `plazum serve`: %v", err)
	}
	// SE LEE A TROZOS Y NO CON io.ReadAll, y esto costo un rojo: ReadAll no
	// vuelve hasta que el otro extremo cierra, o sea hasta que el servidor
	// TERMINA. Con el, la salida del arranque estaba vacia mientras el servidor
	// servia, que es justo el rato en el que hace falta: ahi es donde vive el
	// token de instalacion.
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := tuberia.Read(buf)
			if n > 0 {
				s.mu.Lock()
				s.salida.Write(buf[:n])
				s.mu.Unlock()
			}
			if err != nil {
				return
			}
		}
	}()
	// Hasta que /salud contesta. Es lo que un operador ve como «ya esta
	// arriba», y es la unica espera que se puede medir sin inventarse un reloj.
	plazo := time.Now().Add(30 * time.Second)
	for {
		resp, err := http.Get(s.base + "/salud")
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				break
			}
		}
		if time.Now().After(plazo) {
			s.parar()
			s.mu.Lock()
			texto := s.salida.String()
			s.mu.Unlock()
			t.Fatalf("`plazum serve` no contesta en %s despues de 30 s. No es que el TTFV "+
				"sea largo: es que el producto no arranca.\n--- salida ---\n%s",
				s.base, texto)
		}
		time.Sleep(20 * time.Millisecond)
	}
	s.tArranque = time.Since(inicio)
	t.Cleanup(s.parar)
	return s
}

func puertoLibre(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("no hay puerto libre para levantar el producto: %v", err)
	}
	defer func() { _ = l.Close() }()
	return l.Addr().(*net.TCPAddr).Port
}

func cronometrar(f func()) time.Duration {
	inicio := time.Now()
	f()
	return time.Since(inicio)
}

func extensionDeBinario() string {
	if os.PathSeparator == '\\' {
		return ".exe"
	}
	return ""
}

// LAS DOS FORMAS DE PREGUNTA, Y HACEN FALTA LAS DOS.
//
// rePreguntaEnLista saca el id de TODA pregunta, del <li> que la envuelve, que
// es el unico sitio donde estan todas. reBooleana saca el de las de si/no, que
// son las que SE PUEDEN contestar por POST.
//
// CONTAR SOLO LAS PRIMERAS SERIA UNDERCOUNT, Y SE VIO: la primera version de
// este recorrido conto 10 donde la pantalla pinta 19, o sea que se dejo NUEVE
// preguntas fuera del coste humano y bajo el TTFV en tres minutos por la cara.
// El error iba, otra vez, en la direccion que favorece. Se cruza contra el
// recuento de <li class="pregunta" de la pagina, que es la afirmacion
// independiente: si las dos expresiones dejan de casar lo que la pantalla
// pinta, esto se para en vez de medir de menos.
//
// LO QUE NO SE CONTESTA SE CUENTA IGUAL: una pregunta de valor cuesta lo mismo
// de leer y de contestar, y que su respuesta no llegue al alcance publicado es
// un limite del producto que la propia pantalla declara
// (alcance.pregunta.valor.no_se_guarda), no un descuento del coste humano.
var reBooleana = regexp.MustCompile(`name="pregunta" value="([^"]+)"`)
var rePreguntaEnLista = regexp.MustCompile(`id="p-([^"]+)"`)

// rePublicar dice si la pantalla ofrece publicar el alcance de la instalacion.
var rePublicar = regexp.MustCompile(`name="accion" value="publicar"`)

// contestarYPublicar hace lo que hace una persona: contestar la entrevista
// entera y pulsar el boton que publica el calendario de la instalacion.
//
// # POR QUE ESTO VIVE EN LA MEDIDA Y NO EN UNA SUITE
//
// Porque sin ello la medida MIENTE HACIA ABAJO en un sitio y HACIA ARRIBA en
// otro, y las dos mentiras se tapaban. Hacia abajo, porque contaba solo las
// preguntas de la primera pantalla y la entrevista tiene revelacion progresiva:
// contestar saca mas. Hacia arriba, porque sin contestar nada, TODAS las
// pantallas de despues salen en su estado vacio y sus ordenes de terminal
// parecen inevitables cuando lo que pasa es que nadie ha contestado.
//
// # EL BUCLE PARA POR DOS SITIOS, Y LOS DOS HACEN FALTA
//
// Para cuando una vuelta no descubre ninguna pregunta nueva, que es lo normal, y
// para en un tope duro de vueltas, que es la red. Sin el tope, un producto que
// revelara una pregunta nueva por cada respuesta dejaria este test girando hasta
// el timeout del paquete, y un test colgado se lee como una maquina lenta.
//
// Devuelve la ultima pagina y CUANTAS PREGUNTAS DISTINTAS se han contestado, que
// es el numero que cuesta 20s cada una.
func (s *servidorDePruebaTTFV) contestarYPublicar(t *testing.T, ruta, pagina string) (string, int) {
	t.Helper()
	const maxVueltas = 12
	contestadas := map[string]bool{}
	vistas := map[string]bool{}
	for vuelta := 0; vuelta < maxVueltas; vuelta++ {
		principal := entreMain(t, "alcance", pagina)
		// TODAS LAS PREGUNTAS SE CUENTAN, se puedan contestar por POST o no.
		// El identificador sale del <li> de la lista, que lo lleva SIEMPRE:
		// contar por el formulario dejaba fuera las que piden un valor, que son
		// un tercio, y su respuesta viaja en la direccion. Que el guardado no se
		// las lleve es un limite del producto que la pantalla declara, no un
		// descuento del coste de contestarlas.
		for _, m := range rePreguntaEnLista.FindAllStringSubmatch(principal, -1) {
			vistas[m[1]] = true
		}
		nuevas := 0
		for _, m := range reBooleana.FindAllStringSubmatch(principal, -1) {
			vistas[m[1]] = true
			if contestadas[m[1]] {
				continue
			}
			contestadas[m[1]] = true
			nuevas++
			// SE CONTESTA QUE SI A TODAS, y se dice por que: es el caso que mas
			// obligaciones deriva, o sea el calendario mas caro de pintar y el
			// recorrido mas largo. Contestar que no a la mitad daria un numero
			// mas bonito midiendo un producto que hace menos.
			pagina = s.enviarAlAlcance(t, ruta, pagina, url.Values{
				"accion": {"si"}, "pregunta": {m[1]},
			})
		}
		// EL CONTRASTE, en la vuelta en la que la pantalla todavia las pinta
		// todas: lo que las dos expresiones reconocen tiene que ser lo que la
		// pagina dice tener. Sin esto, una plantilla que cambie el atributo deja
		// a este recorrido contando de menos y en verde, que es como se perdio
		// una pregunta entera la ultima vez.
		if vuelta == 0 {
			if pintadas := strings.Count(principal, `<li class="pregunta`); pintadas != len(vistas) {
				t.Fatalf("la pantalla del alcance pinta %d preguntas y este recorrido reconoce "+
					"%d.\n"+
					"  Las que no se reconocen no se contestan NI SE COBRAN, asi que el TTFV "+
					"saldria mas bajo de lo que es, que es la direccion en la que estos "+
					"errores salen siempre.", pintadas, len(vistas))
			}
		}
		if nuevas == 0 {
			break
		}
	}
	// Y SE PUBLICA. Sin esto, el calendario y el plan de avisos siguen vacios
	// aunque la entrevista este contestada: publicar es un acto propio, con su
	// boton, porque lo que se publica lo ve cualquiera que abra la instalacion.
	if !rePublicar.MatchString(pagina) {
		t.Fatalf("la pantalla del alcance no ofrece publicar el calendario de la instalacion "+
			"despues de contestar %d preguntas.\n"+
			"  Sin ese boton, el calendario y el plan de avisos solo se pueden llenar desde el "+
			"terminal, que es lo que este numero mide.", len(contestadas))
	}
	valores := url.Values{"accion": {"publicar"}}
	for id := range contestadas {
		valores.Add("si", id)
	}
	pagina = s.enviarAlAlcance(t, ruta, pagina, valores)
	// EL NUMERO QUE SE COBRA ES EL DE PREGUNTAS VISTAS, no el de contestadas:
	// las que piden un valor se leen y se contestan igual aunque su respuesta no
	// llegue al alcance publicado.
	return pagina, len(vistas)
}

// enviarAlAlcance envia un formulario de la pantalla del alcance con el token
// que trae la propia pagina, y devuelve la pagina resultante.
//
// EL TOKEN SALE DE LA PAGINA Y NO DE UNA CONSTANTE: si un dia el formulario
// dejara de llevarlo, esto se pondria rojo en vez de seguir midiendo con un
// token inventado que el servidor rechazaria en silencio.
func (s *servidorDePruebaTTFV) enviarAlAlcance(t *testing.T, ruta, pagina string,
	valores url.Values) string {

	t.Helper()
	csrf := reCampoCSRFTTFV.FindStringSubmatch(pagina)
	if csrf == nil {
		t.Fatalf("la pantalla del alcance no trae el campo %q, asi que no se puede contestar "+
			"la entrevista", serve.CampoCSRF)
	}
	valores.Set(serve.CampoCSRF, csrf[1])
	resp, err := s.cli.PostForm(s.base+ruta, valores)
	if err != nil {
		t.Fatalf("enviando %v a %s: %v", valores["accion"], ruta, err)
	}
	cuerpo, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	// SE EXIGE LA REDIRECCION, no una pagina. Este cliente NO sigue
	// redirecciones a proposito (asi se puede afirmar que /primer-admin
	// redirige de verdad), y ademas el 303 es la afirmacion que importa: una
	// escritura que contesta 200 con la pagina dentro es la que se repite al
	// recargar.
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("%s ha contestado %d a la accion %v y tenia que redirigir con 303\n%s",
			ruta, resp.StatusCode, valores["accion"], recortar(string(cuerpo)))
	}
	siguiente, err := s.cli.Get(s.base + ruta)
	if err != nil {
		t.Fatalf("volviendo a %s despues de %v: %v", ruta, valores["accion"], err)
	}
	despues, _ := io.ReadAll(siguiente.Body)
	_ = siguiente.Body.Close()
	if siguiente.StatusCode != http.StatusOK {
		t.Fatalf("%s contesta %d despues de %v", ruta, siguiente.StatusCode, valores["accion"])
	}
	return string(despues)
}

// censoDelRecorrido es el fichero que sube esta medida.
//
// Cuatro accesos en tres cuentas, con las columnas que censo.ColumnasHabituales
// reconoce. Es pequeno A PROPOSITO: lo que esta medida cronometra es el trabajo
// de la PERSONA, y revisar cuatro accesos o cuatrocientos cuesta lo mismo de
// subir. El tamano del censo mueve el trabajo de revisarlos, que es otra medida
// y no esta.
const censoDelRecorrido = "usuario,permiso,nombre\n" +
	"ana,admin,Ana Ruiz\n" +
	"ana,lectura,Ana Ruiz\n" +
	"luis,lectura,Luis Paz\n" +
	"eva,admin,Eva Sanz\n"

// subirElCenso hace lo que hace una persona en la pantalla de revision de
// accesos: elegir el CSV que acaba de exportar de su IdP y decir de que sistema
// es.
//
// SIN ESTO LA PANTALLA SE MEDIA VACIA, y con ella el acta, que se compone de la
// misma campana. Lo destapo la Marca del censo de pasos el dia que se escribio:
// una pantalla en su estado vacio tiene cero ordenes y cero preguntas, o sea que
// sale barata en esta medida, y esa baratura es la del arnes y no la del
// producto.
func (s *servidorDePruebaTTFV) subirElCenso(t *testing.T, ruta, pagina string) string {
	t.Helper()
	csrf := reCampoCSRFTTFV.FindStringSubmatch(pagina)
	if csrf == nil {
		t.Fatalf("la pantalla de revision de accesos no trae el campo %q, asi que no se puede "+
			"subir el censo desde el navegador", serve.CampoCSRF)
	}
	var cuerpo bytes.Buffer
	w := multipart.NewWriter(&cuerpo)
	if err := w.WriteField(serve.CampoCSRF, csrf[1]); err != nil {
		t.Fatal(err)
	}
	if err := w.WriteField("sistema", "erp"); err != nil {
		t.Fatal(err)
	}
	f, err := w.CreateFormFile("fichero", "usuarios.csv")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write([]byte(censoDelRecorrido)); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest(http.MethodPost, s.base+ruta+"abrir", &cuerpo)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	resp, err := s.cli.Do(req)
	if err != nil {
		t.Fatalf("subiendo el censo a %sabrir: %v", ruta, err)
	}
	respuesta, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("la subida del censo ha contestado %d y tenia que redirigir con 303\n%s",
			resp.StatusCode, recortar(string(respuesta)))
	}
	siguiente, err := s.cli.Get(s.base + ruta)
	if err != nil {
		t.Fatalf("volviendo a %s despues de subir el censo: %v", ruta, err)
	}
	despues, _ := io.ReadAll(siguiente.Body)
	_ = siguiente.Body.Close()
	if siguiente.StatusCode != http.StatusOK {
		t.Fatalf("%s contesta %d despues de subir el censo", ruta, siguiente.StatusCode)
	}
	return string(despues)
}
