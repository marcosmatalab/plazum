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
// CUATRO de las catorce cifras se derivan enteras en esta misma pagina, porque
// la seccion que las compone ya esta pintada arriba:
//
//	en la ventana  -> las fechas de los meses
//	pasados        -> los vencimientos pasados
//	sin fecha      -> los relojes que obligan y no dan fecha
//	cesan          -> lo que deja de obligar dentro de la ventana
//
// LAS OTRAS DIEZ NO SE PUEDEN ABRIR TODAVIA, y el motivo es de otra columna, no
// de esta pantalla: `nucleo/pantalla.Calendario` guarda esos descartes como
// CONTADORES y no como listas. Lo unico que retiene por elemento es `Destinos`,
// que es un mapa de obligacion a etiqueta, y las cifras de la cuenta estan en
// HITOS (una obligacion con tres hitos suma tres y ocupa una entrada), asi que
// abrir una cifra contra ese mapa daria una lista que NO CUADRA con el numero
// que la abre, que es peor que no tener el enlace: un numero que no cuadra con
// su lista hace que se deje de leer la pantalla entera, y con razon.
//
// El hueco esta TOPADO: SinDerivacionEsperadas se compara con igualdad exacta en
// los dos sentidos, asi que no puede crecer en silencio y no puede encogerse sin
// que alguien baje el numero a mano.

import "sort"

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
	// CifraSinDerivacion: todavia no se puede abrir. Exige motivo, y su
	// numero esta topado.
	CifraSinDerivacion
)

func (d DerivacionDeCifra) String() string {
	switch d {
	case CifraConSeccion:
		return "se abre en una seccion de esta pagina"
	case CifraSinDerivacion:
		return "no se puede abrir todavia"
	default:
		return "SIN DECLARAR (valor cero)"
	}
}

// SinDerivacionEsperadas es cuantas de las cifras de la cuenta NO se pueden
// abrir hoy.
//
// EL CARDINAL SE ESCRIBE PARA QUE MOLESTE. Un hueco sin numero se olvida; con
// numero, y con igualdad exacta en los dos sentidos, se entera todo el mundo
// cuando crece Y cuando encoge. Bajarlo exige haber abierto una cifra de verdad.
const SinDerivacionEsperadas = 10

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
)

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
	// Siempre dice si se pinta aunque valga cero. Los cuatro primeros son la
	// particion y un cero ahi informa; un descarte en cero es una linea que no
	// dice nada y empuja fuera de la vista a la que si dice algo.
	Siempre bool
}

// SePinta dice si esta cifra sale en la pagina.
func (c CifraDeLaCuenta) SePinta() bool { return c.Siempre || c.N != 0 }

// motivoDeContador es el motivo compartido de las diez que no se abren. Se
// escribe UNA vez: diez copias de la misma frase se separan a la tercera
// edicion, y entonces el hueco parece diez huecos distintos.
const motivoDeContador = "nucleo/pantalla.Calendario lo guarda como contador de HITOS y no " +
	"como lista. Lo unico que retiene por elemento es Destinos, que va por obligacion: " +
	"abrir la cifra contra ese mapa daria una lista que no cuadra con el numero que la " +
	"abre. Abrirla exige que nucleo/pantalla devuelva el cubo en la misma unidad, y eso " +
	"es de otra columna"

// CifrasDeLaCuenta es la lista, en el orden en que se pinta.
//
// EL ORDEN NO ES DECORATIVO: los cuatro primeros son la particion (instalados,
// en vigor, alcanzados, en la ventana) y cada uno solo miente si se lee sin los
// otros tres. Detras van los descartes.
func CifrasDeLaCuenta(c CuentaVista) []CifraDeLaCuenta {
	return []CifraDeLaCuenta{
		{Campo: "Instalados", Clave: "calendario.pantalla.cuenta.instalados", N: c.Instalados,
			Siempre: true, Derivacion: CifraSinDerivacion, Motivo: motivoDeContador},
		{Campo: "EnVigor", Clave: "calendario.pantalla.cuenta.en_vigor", N: c.EnVigor,
			Siempre: true, Derivacion: CifraSinDerivacion, Motivo: motivoDeContador},
		{Campo: "Alcanzados", Clave: "calendario.pantalla.cuenta.alcanzados", N: c.Alcanzados,
			Siempre: true, Derivacion: CifraSinDerivacion, Motivo: motivoDeContador},
		// LA PRIMERA QUE SE ABRE: las fechas de los meses son exactamente lo
		// que este numero cuenta (Calendario.Total()).
		{Campo: "EnLaVentana", Clave: "calendario.pantalla.cuenta.en_la_ventana",
			N: c.EnLaVentana, Siempre: true,
			Derivacion: CifraConSeccion, Ancla: AnclaFechas},
		{Campo: "MasAlla", Clave: "calendario.pantalla.cuenta.mas_alla", N: c.MasAlla,
			Derivacion: CifraSinDerivacion, Motivo: motivoDeContador},
		// LOS VENCIDOS. El numero cuenta OCURRENCIAS y la lista trae una fila
		// por obligacion con sus ciclos al lado, asi que la seccion lo deriva
		// entero: se lee sumando los ciclos de cada fila.
		{Campo: "Pasados", Clave: "calendario.pantalla.cuenta.pasados", N: c.Pasados,
			Derivacion: CifraConSeccion, Ancla: AnclaVencidas},
		{Campo: "AntesDeVigor", Clave: "calendario.pantalla.cuenta.antes_de_vigor",
			N: c.AntesDeVigor, Derivacion: CifraSinDerivacion, Motivo: motivoDeContador},
		{Campo: "SinFecha", Clave: "calendario.pantalla.cuenta.sin_fecha", N: c.SinFecha,
			Derivacion: CifraConSeccion, Ancla: AnclaSinFecha},
		// LOS ESTRENOS NO SE ABREN, y merece decirse por que aunque su seccion
		// exista: este numero cuenta TODO lo que estrena dentro de la ventana,
		// alcance aparte, y la lista de arriba solo trae lo que ademas te
		// alcanza (HitosQueEstrenanYTeAlcanzan, que esta pantalla ni siquiera
		// recibe). Enlazar ahi seria mandar a una lista mas corta que el numero
		// que la abre, que es el error que D11-c existe para impedir.
		{Campo: "Estrenan", Clave: "calendario.pantalla.cuenta.estrenan", N: c.Estrenan,
			Derivacion: CifraSinDerivacion,
			Motivo: "cuenta todo lo que estrena en la ventana, te alcance o no, y la seccion " +
				"de estrenos solo trae lo que te alcanza. Abrirla ahi mandaria a una lista " +
				"mas corta que su numero"},
		// LOS CESES SI: el contador y la lista se rellenan en la misma rama,
		// despues de la aplicabilidad, asi que el numero es la suma de los
		// hitos de las filas que se ven.
		{Campo: "Cesan", Clave: "calendario.pantalla.cuenta.cesan", N: c.Cesan,
			Derivacion: CifraConSeccion, Ancla: AnclaCeses},
		{Campo: "NoAlcanzados", Clave: "calendario.pantalla.cuenta.no_alcanzados",
			N: c.NoAlcanzados, Derivacion: CifraSinDerivacion, Motivo: motivoDeContador},
		{Campo: "YaCesados", Clave: "calendario.pantalla.cuenta.ya_cesados", N: c.YaCesados,
			Derivacion: CifraSinDerivacion, Motivo: motivoDeContador},
		{Campo: "EmpiezanTarde", Clave: "calendario.pantalla.cuenta.empiezan_tarde",
			N: c.EmpiezanTarde, Derivacion: CifraSinDerivacion, Motivo: motivoDeContador},
		{Campo: "Ilegibles", Clave: "calendario.pantalla.cuenta.ilegibles", N: c.Ilegibles,
			Derivacion: CifraSinDerivacion, Motivo: motivoDeContador},
	}
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
