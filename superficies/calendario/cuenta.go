package calendario

// LA PUERTA D11-c EN EL CALENDARIO: ninguna cifra huerfana de enlace, y las
// cifras se ENUMERAN DEL TIPO, no de una lista escrita en la plantilla.
//
// # El agujero que cierra, con su cardinal
//
// La puerta D11-c existia para el panel de inicio (`superficies/pantallas`,
// TestNingunaCifraDelPanelSeQuedaSinSuDerivacion) y NO existia para esta
// pantalla, que pinta CATORCE numeros al pie y no llevaba ni un enlace. Catorce
// cifras que hay que creerse, en la pantalla que un CISO abre para saber que le
// vence, en un producto que se vende exactamente al reves.
//
// # Por que la lista se deriva del tipo y no se escribe
//
// La plantilla tenia los catorce `<li>` escritos a mano, uno por campo de
// CuentaVista. Eso es una segunda copia de la estructura, y se separa el dia que
// alguien anada el campo quince: el campo existiria, la contabilidad lo sumaria
// y la pantalla no lo pintaria, sin que nada se pusiera rojo. Aqui la lista sale
// de recorrer CuentaVista por reflexion en el test y de UNA sola declaracion
// aqui, cruzadas en los dos sentidos.
//
// # Lo que se puede abrir hoy y lo que no, con su numero
//
// TRECE de las catorce se abren. UNA no, y esa una es el ancla de la pagina.
//
// Y se abren de DOS maneras distintas, porque hay dos formas de que un numero
// deje de haber que creerselo:
//
//	CON SECCION (11)   un enlace a la lista que lo compone, contable a mano:
//	                   en la ventana, mas alla, pasados, antes de vigor, sin
//	                   fecha, alcanzados, estrenan, cesan, ya cesados, empiezan
//	                   tarde, ilegibles
//	CON PARTICION (2)  la SUMA de otras cifras de esta misma lista, escrita al
//	                   lado: instalados y en vigor
//	SIN ABRIR (1)      no alcanzados
//
// # De diez huerfanas a una, en dos tramos, y que fallaba en cada uno
//
// El primer arreglo (03-09-2026) bajo de diez a cinco por la raiz:
// `nucleo/pantalla.Calendario` guardaba los descartes como CONTADORES y lo unico
// que retenia por elemento era `Destinos`, que va por obligacion. Abrir una
// cifra en hitos contra una lista en obligaciones da una lista que NO CUADRA con
// el numero que la abre, y eso es peor que no tener enlace.
//
// El segundo (04-09-2026) bajo de cinco a una, y las tres cosas que lo
// permitieron son tres correcciones a lo que se dijo entonces:
//
//	«estrenan no se abre porque su seccion trae menos»   cierto, y la salida no
//	  era cerrar la cifra: era darle SU lista, la que cuenta lo mismo que ella.
//	  Medido en es-fabricante-software: la cifra dice 9 y la seccion de estrenos
//	  tiene 4 filas
//	«alcanzados es el corpus mirado de otra forma»       NO lo es: es lo que SI
//	  te alcanza, entre 17 y 73 hitos en los tres perfiles publicados. D-13
//	  prohibe enumerar lo que NO te alcanza, y esto es lo contrario
//	«instalados y en vigor no se pueden enumerar»        cierto, y una cifra
//	  tambien se comprueba SUMANDO. Las dos particiones estaban escritas y
//	  comprobadas desde el 28-08-2026 en contabilidad_test.go, y la pantalla se
//	  las guardaba
//
// La leccion, que es de la casa: un motivo que describe por que no vale UNA
// salida y calla que hay OTRA no es un motivo, es una descripcion, y se queda
// puesto para siempre porque nadie lo puede contradecir.
//
// # La que queda, con su cardinal y su puerta
//
// `no alcanzados`. Enumerarla serian entre 145 y 201 filas ajenas en los tres
// perfiles publicados (D-13), y abrirla por particion seria circular:
// `en vigor = alcanzados + no alcanzados` es UNA ecuacion con DOS incognitas si
// las dos se apoyan solo en ella. `alcanzados` se sostiene solo, asi que la
// ecuacion abre `en vigor` y `no alcanzados` se queda como el UNICO numero de la
// pagina que hay que creerse. Su motivo lo dice, con la orden de terminal que si
// los ensena.
//
// # Y el rotulo de cada seccion es la clave de su propia cifra
//
// No es ahorro: una seccion que es una cifra desplegada no puede decir una cosa
// distinta de la cifra que la abre si las dos salen de la misma cadena, y ademas
// no hay una frase nueva que traducir ni una segunda copia que se quede vieja.
// Efecto lateral buscado: los dos tramos han costado CERO claves de catalogo
// nuevas, que este frente no podia tocar.

import (
	"sort"
	"strings"
)

// DerivacionDeCifra dice si una cifra de la cuenta se puede abrir. Vocabulario
// cerrado.
type DerivacionDeCifra uint8

const (
	// CifraSinDeclarar es el VALOR CERO Y ES INVALIDO. Un campo nuevo de
	// CuentaVista que nadie declare llega aqui con el cero, y el silencio es
	// justo lo que dejaria una cifra huerfana sin que nada se pusiera rojo.
	CifraSinDeclarar DerivacionDeCifra = iota
	// CifraConSeccion: esta cifra se abre en una seccion de esta misma pagina,
	// que es la que la compone entera.
	CifraConSeccion
	// CifraConParticion: esta cifra NO se abre a una lista y se abre igual, a
	// la ARITMETICA que la compone: es la suma exacta de otras cifras de esta
	// misma pagina, y la pagina la escribe al lado.
	//
	// POR QUE ES UNA DERIVACION DE VERDAD Y NO UNA EXCUSA. «Instalados» y «en
	// vigor» cuentan el corpus entero, y enumerarlos seria pintar centenares de
	// obligaciones que en su mayoria no son tuyas, que es lo que D-13 decidio
	// que no se hace. Pero una cifra tambien se comprueba sumando: quien lea
	// «249 = 218 + 9 + 1 + 21 + 0» tiene los cinco sumandos delante, en la misma
	// lista, y cada uno de ellos se abre por su cuenta. No hay que creerse el
	// numero, hay que sumar cinco numeros que ya estan ahi.
	//
	// Y ES EXACTAMENTE LA LEY DE CONSERVACION QUE EL CODIGO YA COMPRUEBA: la
	// particion por tiempo y la particion por alcance de contabilidad_test.go.
	// La pantalla deja de esconder la unica prueba que ya tenia.
	CifraConParticion
	// CifraSinDerivacion: todavia no se puede abrir. Exige motivo, y su
	// numero esta topado.
	CifraSinDerivacion
)

func (d DerivacionDeCifra) String() string {
	switch d {
	case CifraConSeccion:
		return "se abre en una seccion de esta pagina"
	case CifraConParticion:
		return "se compone de otras cifras de esta pagina, y la suma se escribe"
	case CifraSinDerivacion:
		return "no se puede abrir todavia"
	default:
		return "SIN DECLARAR (valor cero)"
	}
}

// FormaDeCuadre dice COMO se contrasta una cifra con su seccion. Vocabulario
// cerrado y con el valor cero prohibido: una cifra que se abre y no dice como se
// comprueba es una cifra que nadie comprueba.
type FormaDeCuadre uint8

const (
	// CuadreSinDeclarar es el VALOR CERO Y ES INVALIDO.
	CuadreSinDeclarar FormaDeCuadre = iota
	// CuadreFilas: el numero es cuantas filas pinta la seccion. Es el caso
	// normal y el unico que el lector puede comprobar contando.
	CuadreFilas
	// CuadreCiclos: el numero cuenta OCURRENCIAS y la seccion trae una fila por
	// obligacion con sus ciclos al lado, asi que se comprueba sumando los
	// ciclos de las filas. Es el unico caso donde contar filas NO es el
	// contraste, y esta escrito para que no se pueda usar como excusa en otro.
	CuadreCiclos
)

// SinDerivacionEsperadas es cuantas de las cifras de la cuenta NO se pueden
// abrir hoy.
//
// EL CARDINAL SE ESCRIBE PARA QUE MOLESTE. Un hueco sin numero se olvida; con
// numero, y con igualdad exacta en los dos sentidos, se entera todo el mundo
// cuando crece Y cuando encoge. Bajarlo exige haber abierto una cifra de verdad.
const SinDerivacionEsperadas = 1

// Las anclas de las secciones de esta pagina. Se declaran aqui, en Go, y la
// plantilla las pinta desde aqui: escritas a mano en el HTML serian una segunda
// copia que se separa del enlace el dia que una cambie, y el sintoma seria un
// enlace que no lleva a ningun sitio, o sea la cifra huerfana con una capa de
// pintura encima.
const (
	AnclaFechas   = "fechas"
	AnclaVencidas = "vencidas"
	AnclaSinFecha = "sin-fecha"
	AnclaCeses    = "ceses"
	AnclaEstrenos = "estrenos"
	// LAS CINCO SECCIONES DE DESCARTE. No existian, y por eso cinco de las diez
	// cifras huerfanas lo eran: no hay enlace posible hacia una seccion que no
	// se pinta. Su rotulo es la clave de la propia cifra, asi que no hay una
	// segunda frase que se pueda separar de la primera.
	AnclaYaCesados     = "ya-cesados"
	AnclaEmpiezanTarde = "empiezan-tarde"
	AnclaIlegibles     = "ilegibles"
	AnclaMasAlla       = "mas-alla"
	AnclaAntesDeVigor  = "antes-de-vigor"
	// LA SECCION DE LO QUE SI ES TUYO. No es un descarte y no va con ellos: es
	// la lista de los hitos que la aplicabilidad deriva de tus respuestas, y es
	// la unica de estas listas que habla de ti. D-13 no la alcanza: D-13 dice
	// que no se enumere lo que NO te alcanza, porque serian centenares ajenos.
	AnclaAlcanzados = "alcanzados"
	// Y LA DE LO QUE ESTRENA, te alcance o no. Es distinta de la seccion de
	// estrenos de arriba, que solo trae lo tuyo: la cifra cuenta el corpus
	// entero, asi que su lista tiene que contarlo tambien.
	AnclaEstrenan = "estrenan"
)

// ParteDeCifra es un sumando de una cifra que se abre por particion.
//
// Lleva el CAMPO ademas del numero porque el campo es la identidad por la que la
// puerta cruza el sumando con la cifra que lo produce (invariante 7): por
// posicion, reordenar CifrasDeLaCuenta cambiaria de que se dice compuesta cada
// cifra sin que nada se rompiera.
type ParteDeCifra struct {
	Campo string
	N     int
}

// CifraDeLaCuenta es un numero del pie del calendario, listo para pintar.
type CifraDeLaCuenta struct {
	// Campo es el nombre del campo de CuentaVista del que sale. ES EL CAMPO
	// POR EL QUE CASA el cruce con el tipo, y por eso no es decorativo: sin el,
	// el test tendria que emparejar por el orden de la lista, que no lo firma
	// nadie y que reordenar cambia entero (invariante 7).
	Campo string
	// Clave es la clave de catalogo del rotulo, que lleva el numero dentro.
	Clave string
	// N es el numero.
	N int
	// Derivacion dice si se puede abrir.
	Derivacion DerivacionDeCifra
	// Ancla es la seccion de esta pagina que la compone, en CifraConSeccion.
	Ancla string
	// Motivo es por que no se puede abrir, en CifraSinDerivacion. No se pinta:
	// es para quien lea el codigo y para la puerta.
	Motivo string
	// Cuadre dice COMO se contrasta esta cifra con las filas de su seccion, en
	// CifraConSeccion. El valor cero esta prohibido: un enlace sin forma de
	// cuadre promete una comprobacion que nadie hace. Ver cuadre_test.go.
	Cuadre FormaDeCuadre
	// Partes son los sumandos, en CifraConParticion. Su suma TIENE que dar N y
	// lo comprueba la puerta: una particion que no cuadra es la misma promesa
	// rota que un enlace a una lista mas corta, escrita con numeros.
	Partes []ParteDeCifra
	// Siempre dice si se pinta aunque valga cero. Los cuatro primeros son la
	// particion y un cero ahi informa; un descarte en cero es una linea que no
	// dice nada y empuja fuera de la vista a la que si dice algo.
	Siempre bool
}

// SePinta dice si esta cifra sale en la pagina.
func (c CifraDeLaCuenta) SePinta() bool { return c.Siempre || c.N != 0 }

// SinAbrir dice si esta cifra hay que creersela.
//
// EXISTE PARA QUE LA PAGINA LO DIGA, que es la mitad que faltaba. `Motivo` lleva
// desde el 04-09-2026 la explicacion entera y estaba escrito «no se pinta: es
// para quien lea el codigo y para la puerta», o sea que el UNICO numero de la
// pagina que hay que creerse se pintaba exactamente igual que los trece que no.
// Quien mira la pantalla no puede distinguir un numero comprobable de uno que no
// lo es si los dos salen iguales, y entonces la comprobabilidad de los otros
// trece no le sirve de nada: se los cree todos o no se cree ninguno.
//
// El motivo largo sigue sin pintarse (es prosa de codigo y cita una decision de
// diseno), y lo que sale es la clave `calendario.pantalla.cuenta.sin_abrir`, que
// dice las dos cosas que el lector necesita: que esta no se abre, y cual es la
// orden que si las ensena.
func (c CifraDeLaCuenta) SinAbrir() bool { return c.Derivacion == CifraSinDerivacion }

// Descuadre es una cifra de la cuenta que no cuadra con su propia derivacion.
//
// Suma es lo que la derivacion suma DE VERDAD en esta pagina (los sumandos de la
// particion, o las filas de la seccion) y N es lo que la cabecera dice. Los dos
// viajan porque el aviso los pinta los dos: un aviso que dijera solo «esto no
// cuadra» obliga a creerse tambien el aviso.
type Descuadre struct {
	Campo string
	Clave string
	N     int
	Suma  int
}

// DescuadresDeLaCuenta cruza CADA cifra con lo que su derivacion suma de verdad,
// y devuelve las que no coinciden.
//
// # Cada cifra, incluidas las que no se pintan, y ese es el caso que importa
//
// Una cifra en cero no se pinta (`SePinta`), asi que no tiene seccion y no tiene
// filas. Si su lista retenida trae tres relojes, esos tres hitos no salen en
// ninguna parte de la pagina y nadie los echa de menos, porque nadie vio nunca
// que existieran. Filtrar por `SePinta` aqui habria dejado fuera exactamente
// ese caso, que es el unico en el que un descuadre es del todo invisible.
//
// # Por que la pagina tiene que saber decir esto, y no bastaba el test
//
// `cuadre_test.go` ya cuenta las filas de cada seccion contra su cabecera, y
// `nucleo/pantalla.Calendario.Cuadra()` ya contrasta cada contador con su lista.
// Las dos corren contra el corpus publicado y contra un dato sintetico. Ninguna
// de las dos corre contra el calendario del cliente, que es el unico que ese
// cliente va a mirar: si SU corpus produce un descuadre, la pantalla se lo pinta
// tan tranquila y el numero que sobra o falta simplemente NO SALE. Es el mismo
// argumento que ya tenia escrito el escalado, que si pinta su aviso desde el
// primer dia, y el calendario no.
//
// # El emparejamiento, dicho en voz alta (invariante 7)
//
// `contadas` casa por CAMPO, que es el mismo identificador por el que la puerta
// D11-c cruza la declaracion con `CuentaVista` y por el que `seccionesDe` reparte
// las secciones. No casa por orden ni por posicion: reordenar `CifrasDeLaCuenta`
// no puede cambiar contra que se contrasta una cifra. Aqui no hay nada firmado
// (esto es una pantalla, no un expediente), pero la regla vale igual porque el
// fallo es el mismo: un emparejamiento posicional se rompe en silencio al
// insertar una fila.
//
// # El valor cero, dicho en voz alta (invariante 8)
//
// Una cifra que se declara abrible por seccion y de la que NADIE ha contado la
// derivacion sale como descuadre, no en silencio. Las dos formas de la nada
// hacen lo mismo aqui a proposito: `contadas` a nil y `contadas` vacio dan los
// mismos descuadres, porque «no me han pasado nada» y «me han pasado que no hay
// nada» son los dos «esta cifra no la ha contrastado nadie». La alternativa
// permisiva (saltarse la cifra que no aparece en el mapa) es exactamente como se
// cuela una cifra sin vigilancia: se anade el campo quince, nadie lo cuenta y la
// pagina no dice nada.
func DescuadresDeLaCuenta(cifras []CifraDeLaCuenta, contadas map[string]int) []Descuadre {
	var out []Descuadre
	for _, c := range cifras {
		switch c.Derivacion {
		case CifraConParticion:
			suma := 0
			for _, p := range c.Partes {
				suma += p.N
			}
			if suma != c.N {
				out = append(out, Descuadre{Campo: c.Campo, Clave: c.Clave, N: c.N, Suma: suma})
			}
		case CifraConSeccion:
			// Sin entrada en el mapa, `suma` vale 0 y el descuadre sale. Es lo
			// que se quiere: una cifra abrible que nadie contrasta.
			suma := contadas[c.Campo]
			if suma != c.N {
				out = append(out, Descuadre{Campo: c.Campo, Clave: c.Clave, N: c.N, Suma: suma})
			}
		}
	}
	return out
}

// CamposQueSeContrastan son los campos de las cifras que se abren en una
// seccion, o sea los que `contadas` tiene que traer.
//
// Existe para que la puerta pueda cruzar los dos sentidos: que no falte ninguno
// (una cifra que nadie contrasta, y que ademas saldria como descuadre falso en
// cuanto valiera algo distinto de cero) y que no sobre ninguno (un contraste
// contra una cifra que ya no se abre en ninguna seccion).
func CamposQueSeContrastan(cifras []CifraDeLaCuenta) []string {
	var out []string
	for _, c := range cifras {
		if c.Derivacion == CifraConSeccion {
			out = append(out, c.Campo)
		}
	}
	sort.Strings(out)
	return out
}

// motivoDeLoQueNoTeAlcanza es el motivo de LA UNICA cifra que sigue sin abrirse.
//
// # Por que este motivo es mejor que el que sustituye
//
// El anterior cubria TRES cifras con una frase («no son descartes, son el corpus
// entero mirado de tres formas») y esa frase, siendo cierta, no era un motivo:
// era una descripcion. Describia por que no se pueden ENUMERAR y callaba que una
// cifra tambien se comprueba SUMANDO, que es lo que dos de esas tres hacen
// ahora. Un motivo que tapa una salida que existe es un motivo que se vuelve
// permanente.
//
// Este dice tres cosas y las tres son comprobables:
//
//	que la decision es D-13 y esta tomada, no pendiente;
//	cuanto mide el hueco (145 a 201 hitos en los tres perfiles publicados,
//	  medido el 04-09-2026), que es lo que hace que enumerar entierre;
//	y CUAL es la puerta que si los ensena, con su orden exacta.
//
// # Y por que no se abre por particion, como sus dos hermanas
//
// Porque seria circular. `en vigor = alcanzados + no alcanzados` es UNA ecuacion
// con DOS incognitas si ninguna de las dos se sostiene por si sola, y usarla para
// abrir las dos es demostrar cada una con la otra. `alcanzados` se sostiene solo
// (tiene su lista), asi que la ecuacion abre `en vigor`, y `no alcanzados` se
// queda como el UNICO numero de esta pagina que hay que creerse. Tener
// exactamente uno, dicho en voz alta y con su orden de terminal al lado, es lo
// que se podia conseguir sin romper D-13.
const motivoDeLoQueNoTeAlcanza = "D-13, decidido y no pendiente: enumerar lo que NO te " +
	"alcanza serian entre 145 y 201 filas en los tres perfiles publicados (medido el " +
	"04-09-2026), y eso no informa, entierra. Tampoco se abre por particion sin " +
	"circularidad: `en vigor = alcanzados + no alcanzados` es una ecuacion con dos " +
	"incognitas, y la que se sostiene sola es `alcanzados`, que tiene su lista. Queda " +
	"como el UNICO numero de esta pagina que hay que creerse, y su puerta existe: " +
	"`plazum calendario --todos-los-relojes`"

// CifrasDeLaCuenta es la lista, en el orden en que se pinta.
//
// EL ORDEN NO ES DECORATIVO: los cuatro primeros son la particion (instalados,
// en vigor, alcanzados, en la ventana) y cada uno solo miente si se lee sin los
// otros tres. Detras van los descartes.
func CifrasDeLaCuenta(c CuentaVista) []CifraDeLaCuenta {
	return []CifraDeLaCuenta{
		// LA PARTICION POR TIEMPO, escrita donde el lector la ve. Es la ley de
		// conservacion que contabilidad_test.go comprueba desde el 28-08-2026 y
		// que la pantalla se guardaba para si.
		{Campo: "Instalados", Clave: "calendario.pantalla.cuenta.instalados", N: c.Instalados,
			Siempre: true, Derivacion: CifraConParticion, Partes: []ParteDeCifra{
				{Campo: "EnVigor", N: c.EnVigor},
				{Campo: "Estrenan", N: c.Estrenan},
				{Campo: "YaCesados", N: c.YaCesados},
				{Campo: "EmpiezanTarde", N: c.EmpiezanTarde},
				{Campo: "Ilegibles", N: c.Ilegibles},
			}},
		// LA PARTICION POR ALCANCE. Se sostiene porque `Alcanzados` tiene lista
		// propia: si las dos mitades fueran ciegas, esta ecuacion no demostraria
		// nada (ver motivoDeLoQueNoTeAlcanza).
		{Campo: "EnVigor", Clave: "calendario.pantalla.cuenta.en_vigor", N: c.EnVigor,
			Siempre: true, Derivacion: CifraConParticion, Partes: []ParteDeCifra{
				{Campo: "Alcanzados", N: c.Alcanzados},
				{Campo: "NoAlcanzados", N: c.NoAlcanzados},
			}},
		// LA UNICA DE LOS TRES TOTALES QUE SI SE ENUMERA, y es la que habla de ti:
		// 17, 28 y 73 hitos en los tres perfiles publicados. D-13 prohibe enumerar
		// lo que NO te alcanza; estos son justo los que si.
		{Campo: "Alcanzados", Clave: "calendario.pantalla.cuenta.alcanzados", N: c.Alcanzados,
			Siempre: true, Derivacion: CifraConSeccion, Ancla: AnclaAlcanzados,
			Cuadre: CuadreFilas},
		// LA PRIMERA QUE SE ABRE: las fechas de los meses son exactamente lo
		// que este numero cuenta (Calendario.Total()).
		{Campo: "EnLaVentana", Clave: "calendario.pantalla.cuenta.en_la_ventana",
			N: c.EnLaVentana, Siempre: true,
			Derivacion: CifraConSeccion, Ancla: AnclaFechas, Cuadre: CuadreFilas},
		// LAS QUE SE ABRIERON. nucleo/pantalla retiene ahora la lista de cada
		// descarte EN LA MISMA UNIDAD que su contador: una fila por hito donde
		// el numero cuenta hitos, y una por ocurrencia donde cuenta
		// vencimientos. Que cuadren no se supone: lo comprueba la puerta, fila
		// a fila, contra el numero que las abre.
		{Campo: "MasAlla", Clave: "calendario.pantalla.cuenta.mas_alla", N: c.MasAlla,
			Derivacion: CifraConSeccion, Ancla: AnclaMasAlla, Cuadre: CuadreFilas},
		// LOS VENCIDOS. El numero cuenta OCURRENCIAS y la lista trae una fila
		// por obligacion con sus ciclos al lado, asi que la seccion lo deriva
		// entero: se lee sumando los ciclos de cada fila.
		{Campo: "Pasados", Clave: "calendario.pantalla.cuenta.pasados", N: c.Pasados,
			Derivacion: CifraConSeccion, Ancla: AnclaVencidas, Cuadre: CuadreCiclos},
		// EL DESCARGO VA EN LA CABECERA DE SU PROPIA SECCION, que es el rotulo
		// de esta cifra: «que NO son incumplimientos». Es la unica de las cinco
		// que ensena fechas PASADAS al lado de una obligacion, o sea la unica
		// que se puede leer como una acusacion, y por eso la frase va pegada al
		// dato y no en un pie.
		{Campo: "AntesDeVigor", Clave: "calendario.pantalla.cuenta.antes_de_vigor",
			N: c.AntesDeVigor, Derivacion: CifraConSeccion, Ancla: AnclaAntesDeVigor,
			Cuadre: CuadreFilas},
		{Campo: "SinFecha", Clave: "calendario.pantalla.cuenta.sin_fecha", N: c.SinFecha,
			Derivacion: CifraConSeccion, Ancla: AnclaSinFecha, Cuadre: CuadreFilas},
		// LOS ESTRENOS SE ABREN, Y NO EN LA SECCION DE ARRIBA. El motivo anterior
		// acertaba el diagnostico y erraba la conclusion: esta cifra cuenta TODO lo
		// que estrena, te alcance o no, y la seccion de estrenos solo trae lo tuyo
		// Y una fila por obligacion (medido el 04-09-2026 en es-fabricante-software:
		// la cifra dice 9 y esa lista tiene 4 filas). La salida no era cerrar la
		// cifra, era darle SU lista: nucleo/pantalla retiene `RelojesQueEstrenan`,
		// que cuenta lo mismo que el contador y en la misma unidad.
		{Campo: "Estrenan", Clave: "calendario.pantalla.cuenta.estrenan", N: c.Estrenan,
			Derivacion: CifraConSeccion, Ancla: AnclaEstrenan, Cuadre: CuadreFilas},
		// LOS CESES SI: el contador y la lista se rellenan en la misma rama,
		// despues de la aplicabilidad, asi que el numero es la suma de los
		// hitos de las filas que se ven.
		{Campo: "Cesan", Clave: "calendario.pantalla.cuenta.cesan", N: c.Cesan,
			Derivacion: CifraConSeccion, Ancla: AnclaCeses, Cuadre: CuadreFilas},
		// LA UNICA QUE SIGUE SIN ABRIRSE. Su motivo esta arriba, entero, y dice
		// las tres cosas que el anterior no decia: que decision es, cuanto mide el
		// hueco, y por que la particion tampoco vale aqui.
		{Campo: "NoAlcanzados", Clave: "calendario.pantalla.cuenta.no_alcanzados",
			N: c.NoAlcanzados, Derivacion: CifraSinDerivacion,
			Motivo: motivoDeLoQueNoTeAlcanza},
		{Campo: "YaCesados", Clave: "calendario.pantalla.cuenta.ya_cesados", N: c.YaCesados,
			Derivacion: CifraConSeccion, Ancla: AnclaYaCesados, Cuadre: CuadreFilas},
		{Campo: "EmpiezanTarde", Clave: "calendario.pantalla.cuenta.empiezan_tarde",
			N: c.EmpiezanTarde, Derivacion: CifraConSeccion, Ancla: AnclaEmpiezanTarde,
			Cuadre: CuadreFilas},
		{Campo: "Ilegibles", Clave: "calendario.pantalla.cuenta.ilegibles", N: c.Ilegibles,
			Derivacion: CifraConSeccion, Ancla: AnclaIlegibles, Cuadre: CuadreFilas},
	}
}

// CifrasQueSeDerivanEnCirculo devuelve las cifras cuya particion se apoya, por
// el camino que sea, en ella misma. Con la declaracion sana, la lista esta vacia.
//
// # POR QUE ES LA COMPROBACION QUE HABIA QUE ESCRIBIR
//
// Abrir una cifra por particion es demostrar un numero con otros numeros, y esa
// forma de demostracion tiene un fallo que un enlace no tiene: se puede cerrar
// sobre si misma. Nada impide declarar
//
//	en vigor      = alcanzados + no alcanzados
//	no alcanzados = en vigor - alcanzados
//
// Las dos ecuaciones son CIERTAS, las dos sumas cuadran, y las dos cifras
// quedarian marcadas como abiertas sin que ninguna de las dos se pueda
// comprobar: son la misma ecuacion escrita dos veces. Seria una cifra huerfana
// con una demostracion encima, que es peor que una cifra huerfana, y es
// exactamente lo que D11-c persigue.
//
// Vive en el codigo y no dentro del test para que el control negativo pase por
// ESTE mismo detector: uno que solo se ha ejecutado contra el caso bueno no ha
// demostrado que sepa decir que no. Es el patron de `pantalla.RelojesSinDestino`.
func CifrasQueSeDerivanEnCirculo(cs []CifraDeLaCuenta) []string {
	partes := map[string][]string{}
	for _, c := range cs {
		if c.Derivacion != CifraConParticion {
			continue
		}
		for _, p := range c.Partes {
			partes[c.Campo] = append(partes[c.Campo], p.Campo)
		}
	}
	var malas []string
	// Recorrido en profundidad por cada cifra, con el CAMINO actual marcado.
	// Con `visitados` a secas no valdria: dos cifras pueden apoyarse en la
	// misma tercera sin que haya ningun circulo, y eso es legitimo.
	var camino []string
	enElCamino := map[string]bool{}
	var baja func(string) bool
	baja = func(campo string) bool {
		if enElCamino[campo] {
			return true
		}
		enElCamino[campo] = true
		camino = append(camino, campo)
		for _, sig := range partes[campo] {
			if baja(sig) {
				return true
			}
		}
		camino = camino[:len(camino)-1]
		enElCamino[campo] = false
		return false
	}
	for _, c := range cs {
		if c.Derivacion != CifraConParticion {
			continue
		}
		camino = camino[:0]
		enElCamino = map[string]bool{}
		if baja(c.Campo) {
			malas = append(malas, c.Campo+" se apoya en si misma: "+
				strings.Join(camino, " -> "))
		}
	}
	sort.Strings(malas)
	return malas
}

// CamposDeclaradosDeLaCuenta son los nombres de campo que declara la lista de
// arriba, ordenados. Se exporta para que la puerta pueda cruzarlos con los
// campos reales de CuentaVista en los dos sentidos.
func CamposDeclaradosDeLaCuenta() []string {
	cs := CifrasDeLaCuenta(CuentaVista{})
	out := make([]string, 0, len(cs))
	for _, c := range cs {
		out = append(out, c.Campo)
	}
	sort.Strings(out)
	return out
}
