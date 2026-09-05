package calendario

import (
	"embed"
	"io/fs"
	"sort"

	"github.com/marcosmatalab/plazum/nucleo/pantalla"
	"github.com/marcosmatalab/plazum/superficies/camino"
)

//go:embed plantillas
var plantillasFS embed.FS

// Plantillas expone el sistema de ficheros embebido, por si alguien quiere
// montar otro motor de render.
func Plantillas() fs.FS { return plantillasFS }

// LA REGLA DEL FICHERO. Los ROTULOS son claves de catalogo y los traduce quien
// pinta; el CONTENIDO del corpus (el titulo de una obligacion, su articulo, su
// cita, la derivacion del motor) viaja TAL CUAL, en el idioma de su paquete.
// Traducir texto transcrito del BOE crea obra derivada y se sale de la
// estratificacion de licencias del corpus.
//
// Las fechas se formatean en Go y llegan como cadena. No es pereza: una fecha
// formateada en la plantilla se formatea distinto en cada plantilla que la
// pinte, y este calendario ya sale por dos medios (pantalla y terminal).

// Vista es lo unico que ve la plantilla.
type Vista struct {
	Idioma   string
	Base     string
	Estatico string
	// Titulo es la CLAVE de catalogo.
	Titulo string
	// Aviso es texto ya resuelto: viene de un error, no del catalogo.
	Aviso string
	// Camino es la vuelta al camino guiado. Vacio, no se pinta.
	Camino EnlaceCamino
	// Inicio es la raiz del SITIO, para el enlace de la marca del armazon. No
	// es el prefijo de esta pantalla: usar aquel mandaria la marca a
	// "/calendario/" en vez de a la portada, que es lo contrario de lo que un
	// logo tiene que hacer.
	Inicio string
	// Tira es la barra lateral con el camino entero, marcando este paso. Vacia
	// no se pinta, que es el valor cero restrictivo.
	Tira []camino.PasoTira
	// ICS es la direccion del mismo calendario en iCalendar.
	ICS string

	// SinAlcance es el estado vacio: existe la pantalla y no hay de que
	// derivar. Se pinta con su siguiente paso (puerta D11-b).
	SinAlcance bool

	Organizacion string
	Desde        string
	Hasta        string
	// Supuesto dice que el alcance entero sale de un perfil de arranque.
	Supuesto bool

	Vencidas []VencidaVista
	Meses    []MesVista
	Estrenos []TransicionVista
	Ceses    []TransicionVista
	SinFecha []SinFechaVista
	Cuenta   CuentaVista
	// Cifras es la cuenta LISTA PARA PINTAR, cada una con su enlace a la
	// seccion que la deriva o sin el y con su motivo (puerta D11-c). Sale de
	// CifrasDeLaCuenta, no de una lista escrita en la plantilla: catorce <li>
	// a mano son una segunda copia de CuentaVista y se separan el dia que
	// alguien anada el campo quince.
	Cifras []CifraDeLaCuenta
	// Tuyas son las secciones abiertas que hablan de TI: hoy solo la de los
	// hitos que la aplicabilidad alcanza. Van con el resto de lo tuyo, arriba.
	//
	// POR QUE VAN SEPARADAS DE `Descartes` Y NO EN LA MISMA LISTA. Porque la
	// COLOCACION afirma. Las de abajo se calculan ANTES de la aplicabilidad y
	// cuentan el corpus entero, te alcance o no; puestas entre lo tuyo, el sitio
	// insinua que te obligan aunque ni el rotulo ni las filas lo digan, y en una
	// pantalla que ensena pasado eso es acusar en falso. La separacion era la
	// mitad que se podia hacer sin una clave de catalogo nueva; la otra mitad,
	// la NOTA al frente del bloque, entro el 04-09-2026 con
	// `calendario.pantalla.descarte.no_es_tuyo`.
	Tuyas []DescarteVista
	// Descartes son las secciones que ABREN las cifras del pie que se calculan
	// ANTES de la aplicabilidad: cuentan el corpus entero. Cada una lleva la
	// MISMA clave de catalogo y el MISMO numero que la cifra que la abre, porque
	// las dos salen de CifrasDeLaCuenta: escribir el rotulo de la seccion aparte
	// seria una segunda copia, y la copia sin puerta es la que se queda vieja.
	Descartes []DescarteVista
	// Descuadres son las cifras de esta pagina que no cuadran con su propia
	// derivacion. Vacia es lo normal y es el valor cero: no se pinta nada.
	//
	// SE PINTAN EN LA PAGINA Y NO SOLO EN UN LOG. Un descuadre aqui significa
	// que hay hitos que no salen en ningun sitio, y lo que no sale nadie lo echa
	// de menos. El escalado lo dice desde su primer dia y el calendario no lo
	// decia: `nucleo/pantalla.Calendario.Cuadra()` existia, las puertas lo
	// ejecutaban con el corpus publicado y con el dato sintetico, y el
	// calendario del cliente no pasaba por ninguna de las dos.
	Descuadres []Descuadre
	// SinNingunaFecha dice que no hay ni un vencimiento en la ventana. Es un
	// estado distinto de SinAlcance: aqui SI hay respuestas, y lo que dicen es
	// que no vence nada en doce meses.
	SinNingunaFecha bool
}

// VistaDeUnaCifra es la pagina que abre UNA cifra de la cuenta, entera.
//
// # Por que es otra vista y no un campo mas de Vista
//
// Porque no pinta lo mismo. La pantalla del calendario tiene fechas, vencidas,
// transiciones, la cuenta y sus secciones; esta pagina tiene una lista y un
// enlace de vuelta. Meterla en Vista habria dado una estructura con dos mitades
// que nunca se rellenan a la vez y una plantilla con un `if` gigante alrededor,
// que es como una pantalla acaba pintando media de otra el dia que alguien
// olvida una rama.
//
// COMPARTE EL ARMAZON y por tanto sus campos de marco (idioma, base, estatico,
// camino, tira): eso no es duplicacion, es el contrato de la plantilla
// compartida, y si faltara uno la barra lateral saldria muda.
type VistaDeUnaCifra struct {
	Idioma   string
	Base     string
	Estatico string
	// Titulo es la CLAVE del titulo de la pagina.
	Titulo string
	// Clave es la clave de catalogo de la cifra, con su numero dentro. Es la
	// MISMA que pinta la cuenta en la pantalla principal, por lo mismo que en
	// las secciones: asi la pagina no puede decir una cosa distinta de la cifra
	// que la abre.
	Clave string
	N     int
	Filas []DescarteFilaVista
	// Volver es la ruta de la pantalla del calendario.
	Volver string
	// SinAlcance dice que no hay nada que abrir todavia.
	SinAlcance bool
	Aviso      string

	Camino EnlaceCamino
	Inicio string
	Tira   []camino.PasoTira
}

// DescarteVista es una cifra de descarte ABIERTA: el numero con las filas que lo
// componen.
//
// POR QUE EL ROTULO ES LA CLAVE DE LA PROPIA CIFRA. Porque la seccion no es otra
// cosa: es esa cifra desplegada. «21 que empiezan a obligar mas alla de esta
// ventana» es el titulo exacto de la lista de esas 21, no hace falta inventarle
// otro, y ademas asi la seccion NO PUEDE decir una cosa distinta de la cifra que
// la abre. Un titulo propio habria sido una cadena nueva que traducir y una
// segunda oportunidad de que las dos frases se separen.
type DescarteVista struct {
	// Ancla es el id de la seccion, el mismo al que enlaza la cifra.
	Ancla string
	// Clave es la clave de catalogo del rotulo, con el numero dentro.
	Clave string
	// N es el numero de la cifra, y TIENE que ser len(Filas). Lo comprueba la
	// puerta: un numero que no cuadra con su lista es peor que no tener enlace.
	N     int
	Filas []DescarteFilaVista
	// Tuyo dice si esta seccion habla del sujeto o del corpus entero. Es lo que
	// decide DONDE se pinta, y el valor cero es el restrictivo: una seccion que
	// nadie declare tuya baja con las del corpus, que es el sitio que no afirma
	// nada sobre ti.
	Tuyo bool
}

// DescarteFilaVista es una fila de una seccion de descarte.
//
// UNA FILA POR HITO en las listas que abren un contador de hitos, y una fila por
// OCURRENCIA en las que abren un contador de vencimientos. Es lo que hace que
// contar las filas de la seccion de el numero de su cabecera.
type DescarteFilaVista struct {
	// Fecha va vacia en las secciones que no son de ocurrencias, y entonces no
	// se pinta. El valor cero es el restrictivo: no se inventa una fecha para
	// una fila que no la tiene.
	Fecha    string
	Marco    string
	Titulo   string
	Articulo string
	Hito     string
	Regla    string
}

// EnlaceCamino es la vuelta al camino guiado: direccion y CLAVE del rotulo.
type EnlaceCamino struct {
	URL   string
	Clave string
}

// Hay dice si se pinta. Las dos mitades o ninguna; medio enlace se rechaza al
// construir la superficie.
func (e EnlaceCamino) Hay() bool { return e.URL != "" && e.Clave != "" }

// VencidaVista es una obligacion cuyo plazo ya paso y de cuyo cumplimiento no
// consta nada. NO ES UNA ACUSACION: ver la frase que la acompana en la plantilla.
type VencidaVista struct {
	Desde      string
	Ciclos     int
	Marco      string
	Titulo     string
	Articulo   string
	Hito       string
	Regla      string
	Supuesta   bool
	VariosCicl bool
}

// FechaVista es un vencimiento pintable.
type FechaVista struct {
	Vence        string
	Marco        string
	Titulo       string
	Articulo     string
	Hito         string
	Regla        string
	Aviso        string
	Supuesta     bool
	Divergencias []DivergenciaVista
}

// DivergenciaVista es OTRA lectura del mismo plazo. El motor no elige en
// silencio y esta pantalla no las esconde.
type DivergenciaVista struct {
	Lectura string
	Vence   string
}

// MesVista agrupa las fechas de un mes. Clave es la del catalogo (ui.mes.1).
type MesVista struct {
	Clave  string
	Ano    int
	Fechas []FechaVista
}

// TransicionVista sirve para los estrenos y para los ceses: las dos son el
// mismo dato (una fecha en la que cambia si algo te obliga) y se pintan igual.
//
// UNA FILA POR HITO, y no por obligacion. La cifra «N dejan de obligar dentro de
// la ventana» cuenta HITOS, asi que una fila por obligacion deja la seccion mas
// corta que su cabecera en cuanto una transicion es escalonada (alerta,
// notificacion, informe final), y nadie lo ve: los tres hitos son tres numeros
// de la cifra y una sola linea en la pantalla. Se midio: con dos ceses, uno de
// ellos de dos hitos, la cabecera decia 3 y la seccion pintaba 2.
type TransicionVista struct {
	Dia      string
	Marco    string
	Titulo   string
	Articulo string
	// Hito es el nombre del hito de ESTA fila. Puede venir vacio (el campo es
	// opcional en el formato de corpus) y entonces no se pinta el rotulo: se
	// omite lo que el paquete no dijo en vez de inventarle un nombre.
	Hito     string
	Supuesta bool
}

// SinFechaVista es un reloj que obliga y no ha producido fecha, con su motivo.
// Motivo es una CLAVE de catalogo que declara nucleo/pantalla.
type SinFechaVista struct {
	Marco    string
	Titulo   string
	Articulo string
	Hito     string
	Motivo   string
	Regla    string
}

// CuentaVista es la contabilidad honesta, entera.
//
// SE PINTAN LOS CEROS DE LOS CUATRO PRIMEROS y se ocultan los de los descartes.
// No es capricho: los cuatro primeros son la particion (instalados, en vigor,
// alcanzados, en la ventana) y un cero ahi es informacion; un descarte en cero
// es una linea que no dice nada y empuja la que si dice algo fuera de la vista.
type CuentaVista struct {
	Instalados    int
	EnVigor       int
	Alcanzados    int
	EnLaVentana   int
	MasAlla       int
	Pasados       int
	AntesDeVigor  int
	SinFecha      int
	Estrenan      int
	Cesan         int
	NoAlcanzados  int
	YaCesados     int
	EmpiezanTarde int
	Ilegibles     int
}

// rellenarCon vuelca el calendario derivado en la vista.
//
// Aqui no se decide nada del calendario: se ordena para pintar. El orden de las
// secciones si es decision de producto y es la misma que la salida de terminal:
// lo VENCIDO va delante de todo, porque de todo lo que este calendario puede
// decir es lo unico que ya no admite planificacion.
func (v *Vista) rellenarCon(d Derivado) {
	cal := d.Calendario
	v.Organizacion = d.Organizacion
	v.Supuesto = d.Supuesto
	v.Desde = cal.Desde.Format(formatoDeDia)
	v.Hasta = cal.Hasta.Format(formatoDeDia)
	v.SinNingunaFecha = cal.Total() == 0

	for _, x := range cal.Vencidas {
		v.Vencidas = append(v.Vencidas, VencidaVista{
			Desde: x.Desde.Format(formatoDeDia), Ciclos: x.Ciclos, Marco: x.Marco,
			Titulo: x.Titulo, Articulo: x.Articulo, Hito: x.Hito, Regla: x.Regla,
			Supuesta: x.Supuesta, VariosCicl: x.Ciclos > 1,
		})
	}
	for _, m := range cal.Meses {
		mv := MesVista{Clave: m.Clave, Ano: m.Ano}
		for _, f := range m.Fechas {
			fv := FechaVista{
				Vence: f.Vence.Format(formatoDeInstante), Marco: f.Marco, Titulo: f.Titulo,
				Articulo: f.Articulo, Hito: f.Hito, Regla: f.Regla, Aviso: f.Aviso,
				Supuesta: f.Supuesta,
			}
			for _, dv := range f.Divergencias {
				fv.Divergencias = append(fv.Divergencias, DivergenciaVista{
					Lectura: dv.Lectura, Vence: dv.Vence.Format(formatoDeInstante),
				})
			}
			mv.Fechas = append(mv.Fechas, fv)
		}
		v.Meses = append(v.Meses, mv)
	}
	for _, e := range cal.Estrenos {
		for _, h := range e.NombresDeHitos {
			v.Estrenos = append(v.Estrenos, TransicionVista{
				Dia: e.Desde.Format(formatoDeDia), Marco: e.Marco, Titulo: e.Titulo,
				Articulo: e.Articulo, Hito: h, Supuesta: e.Supuesta,
			})
		}
	}
	for _, c := range cal.Ceses {
		for _, h := range c.NombresDeHitos {
			v.Ceses = append(v.Ceses, TransicionVista{
				Dia: c.Hasta.Format(formatoDeDia), Marco: c.Marco, Titulo: c.Titulo,
				Articulo: c.Articulo, Hito: h, Supuesta: c.Supuesta,
			})
		}
	}
	for _, s := range cal.SinFecha {
		v.SinFecha = append(v.SinFecha, SinFechaVista{
			Marco: s.Marco, Titulo: s.Titulo, Articulo: s.Articulo, Hito: s.Hito,
			Motivo: s.Motivo, Regla: s.Regla,
		})
	}
	v.Cuenta = CuentaVista{
		Instalados: cal.HitosDelCorpus, EnVigor: cal.HitosEnVigor,
		Alcanzados: cal.HitosAplicables, EnLaVentana: cal.Total(),
		MasAlla: cal.MasAllaDeLaVentana, Pasados: cal.VencimientosPasados,
		AntesDeVigor: cal.VencimientosAntesDeLaVigencia, SinFecha: len(cal.SinFecha),
		Estrenan: cal.HitosQueEstrenan, Cesan: cal.HitosQueCesan,
		NoAlcanzados: cal.HitosNoAlcanzados, YaCesados: cal.HitosYaCesados,
		EmpiezanTarde: cal.HitosQueEmpiezanDespues, Ilegibles: cal.HitosConVigenciaIlegible,
	}
	// LAS CIFRAS SE COMPONEN AQUI, del mismo dato y en el mismo sitio. Si se
	// compusieran en la plantilla habria dos listas de catorce y la que no
	// tiene puerta se quedaria vieja.
	v.Cifras = CifrasDeLaCuenta(v.Cuenta)
	// Y LAS SECCIONES QUE LAS ABREN SALEN DE ESA MISMA LISTA, no de una segunda
	// escrita al lado: rotulo y numero son los de la cifra.
	v.Tuyas, v.Descartes = seccionesDe(cal, v.Cifras)
	// Y LA PAGINA SE CONTRASTA A SI MISMA ANTES DE PINTARSE.
	v.Descuadres = DescuadresDeLaCuenta(v.Cifras, v.contadas(cal))
}

// contadas dice, por CAMPO, cuanto suma de verdad la derivacion de cada cifra
// que se abre en una seccion.
//
// # De donde sale cada numero, y por que ninguno sale del contador
//
// La regla es que el contraste no puede leer el mismo dato que la cifra, o seria
// preguntarle a la respuesta por la respuesta. Los siete de las secciones de
// descarte se cuentan de las LISTAS que retiene nucleo/pantalla, que es de donde
// salen las filas que se pintan; los cuatro que se pintan con su propia rama de
// plantilla se cuentan de la VISTA ya rellenada, que es literalmente lo que el
// lector va a poder contar.
//
// SE CUENTA DE LAS LISTAS Y NO DE `v.Tuyas`/`v.Descartes`, y la diferencia es el
// caso que importa: aquellas dos solo traen las cifras que SE PINTAN, asi que una
// cifra en cero con tres relojes retenidos detras no tendria seccion, no tendria
// filas y no tendria descuadre. Tres hitos que no salen en ningun sitio y nadie
// los echa de menos, que es el fallo entero.
//
// Los dos casos tautologicos se dicen en vez de disimularse: `SinFecha` y
// `EnLaVentana` contrastan contra lo mismo de lo que ya salia su contador, asi
// que hoy no pueden ponerse rojos. Se dejan escritos igual, porque el dia que
// cualquiera de los dos cambie de origen el contraste empieza a ser de verdad sin
// que nadie tenga que acordarse de anadirlo.
func (v *Vista) contadas(cal pantalla.Calendario) map[string]int {
	hitos := func(rs []pantalla.RelojDescartado) int {
		n := 0
		for _, r := range rs {
			n += len(r.Hitos)
		}
		return n
	}
	fechas := 0
	for _, m := range v.Meses {
		fechas += len(m.Fechas)
	}
	ciclos := 0
	for _, x := range v.Vencidas {
		ciclos += x.Ciclos
	}
	return map[string]int{
		// De las listas retenidas por el nucleo, una fila por hito.
		"Alcanzados": hitos(cal.RelojesAlcanzados),
		// LA QUE SE ABRE EN OTRA PAGINA SE CONTRASTA AQUI IGUAL, y eso es lo
		// que impide que CifraConPagina sea una excusa mas comoda que
		// CifraConSeccion. La lista se pinta en otro sitio (son entre 145 y 201
		// relojes y aqui enterrarian lo que si es tuyo, D-13), pero el numero
		// que la abre cuadra contra la MISMA lista de la que salen sus filas.
		"NoAlcanzados":  hitos(cal.RelojesNoAlcanzados),
		"Estrenan":      hitos(cal.RelojesQueEstrenan),
		"YaCesados":     hitos(cal.RelojesYaCesados),
		"EmpiezanTarde": hitos(cal.RelojesQueEmpiezanDespues),
		"Ilegibles":     hitos(cal.RelojesConVigenciaIlegible),
		// De las listas de ocurrencias, una fila por ocurrencia.
		"MasAlla":      len(cal.VencimientosMasAlla),
		"AntesDeVigor": len(cal.VencimientosAnterioresALaVigencia),
		// De la vista ya rellenada, que es lo que se pinta. `Pasados` cuenta
		// OCURRENCIAS y su seccion trae una fila por obligacion con sus ciclos
		// al lado, asi que lo que se contrasta es la suma de los ciclos: es la
		// unica cifra cuyo cuadre no es contar filas (CuadreCiclos).
		"EnLaVentana": fechas,
		"Pasados":     ciclos,
		"Cesan":       len(v.Ceses),
		"SinFecha":    len(v.SinFecha),
	}
}

// seccionesDe compone las secciones que abren las cifras del pie, repartidas
// en las que hablan de TI y las que cuentan el corpus entero.
//
// EL EMPAREJAMIENTO CASA POR EL NOMBRE DEL CAMPO de CuentaVista, que es el mismo
// campo por el que la puerta D11-c cruza la declaracion con el tipo, y no por el
// ORDEN de la lista (invariante 7): reordenar CifrasDeLaCuenta no puede mover una
// seccion a otra cabecera. Y las filas salen de las listas que retiene
// nucleo/pantalla, no de recontar nada aqui.
func seccionesDe(cal pantalla.Calendario, cifras []CifraDeLaCuenta) (tuyas, delCorpus []DescarteVista) {
	porRelojes := func(rs []pantalla.RelojDescartado) []DescarteFilaVista {
		var out []DescarteFilaVista
		for _, r := range rs {
			// UNA FILA POR HITO. La cabecera cuenta hitos, asi que una fila por
			// obligacion dejaria la lista corta cada vez que una obligacion
			// escalonada (alerta, notificacion, informe final) entrara en el
			// cubo, y nadie lo veria: los tres hitos son tres numeros de la
			// cifra y una sola linea en la pantalla.
			for _, h := range r.Hitos {
				out = append(out, DescarteFilaVista{
					Marco: r.Marco, Titulo: r.Titulo, Articulo: r.Articulo,
					Hito: h, Regla: r.Regla,
				})
			}
		}
		return out
	}
	porVencimientos := func(vs []pantalla.VencimientoDescartado) []DescarteFilaVista {
		var out []DescarteFilaVista
		for _, v := range vs {
			out = append(out, DescarteFilaVista{
				Fecha: v.Vence.Format(formatoDeInstante), Marco: v.Marco,
				Titulo: v.Titulo, Articulo: v.Articulo, Hito: v.Hito, Regla: v.Regla,
			})
		}
		return out
	}
	filas := map[string][]DescarteFilaVista{
		"Alcanzados":    porRelojes(cal.RelojesAlcanzados),
		"Estrenan":      porRelojes(cal.RelojesQueEstrenan),
		"YaCesados":     porRelojes(cal.RelojesYaCesados),
		"EmpiezanTarde": porRelojes(cal.RelojesQueEmpiezanDespues),
		"Ilegibles":     porRelojes(cal.RelojesConVigenciaIlegible),
		"MasAlla":       porVencimientos(cal.VencimientosMasAlla),
		"AntesDeVigor":  porVencimientos(cal.VencimientosAnterioresALaVigencia),
	}
	// DE QUE LADO CAE CADA UNA, y el criterio no es de estilo: es SI LA CIFRA SE
	// CALCULA ANTES O DESPUES DE LA APLICABILIDAD.
	//
	// Despues (alcanzados, mas alla, antes de vigor) la cifra ya ha pasado por
	// tus respuestas y hablar de ti es correcto. Antes (estrenan, ya cesados,
	// empiezan tarde, ilegibles) cuenta el corpus entero, y la unica colocacion
	// que no afirma nada sobre ti es abajo, detras de todo lo tuyo.
	//
	// El mapa lleva SOLO las de despues y el valor por defecto es «del corpus»,
	// que es el restrictivo: una seccion nueva que nadie clasifique baja al lado
	// que no acusa. Con el mapa al reves, el olvido subiria (invariante 8).
	tuyaPorCampo := map[string]bool{
		"Alcanzados": true, "MasAlla": true, "AntesDeVigor": true,
	}
	for _, c := range cifras {
		f, hay := filas[c.Campo]
		if !hay || !c.SePinta() {
			continue
		}
		d := DescarteVista{Ancla: c.Ancla, Clave: c.Clave, N: c.N, Filas: f,
			Tuyo: tuyaPorCampo[c.Campo]}
		if d.Tuyo {
			tuyas = append(tuyas, d)
			continue
		}
		delCorpus = append(delCorpus, d)
	}
	return tuyas, delCorpus
}

// Los dos formatos de esta pantalla. Un vencimiento es un INSTANTE y no un dia
// (un plazo que vence a las 23:59 no es «ese dia»), asi que las fechas llevan
// hora y las transiciones no: el dia en que una norma empieza a obligar es un
// dia entero, y ponerle una hora seria inventarse una precision.
const (
	formatoDeDia      = "2006-01-02"
	formatoDeInstante = "2006-01-02 15:04"
)

// ClavesDeCatalogo son las claves que pide ESTA pantalla.
//
// POR QUE EXISTE, igual que en las demas superficies: Faltantes(idioma) responde
// «¿al ingles le falta lo que el espanol tiene?», y la pregunta que rompe la
// pantalla es otra, «¿cubre el catalogo lo que la interfaz pide?». Un catalogo
// completo en los dos idiomas y sin "calendario.pantalla.vencido.frase" pinta el
// descargo mas importante del producto como una clave, y Faltantes da verde.
//
// LOS NOMBRES DE LOS MESES Y LOS MOTIVOS DE «sin fecha» NO ESTAN AQUI: los
// declara nucleo/pantalla, que es quien los emite, y el inventario del catalogo
// los pide desde alli. Repetirlos seria una segunda copia.
func ClavesDeCatalogo() []string {
	out := append([]string(nil), claves...)
	// Las del armazon compartido las declara quien lo escribe, no esta
	// pantalla: son las palabras del marco y no las suyas.
	out = append(out, camino.ClavesDelArmazon()...)
	// Los motivos de los relojes sin fecha se piden por su declarador, que es
	// quien sabe cuantos hay: escribirlos aqui se quedaria corto el dia que
	// aparezca un cuarto motivo.
	out = append(out, pantalla.ClavesDelCalendario()...)
	// LOS ROTULOS DE LA CUENTA SALEN DE LA MISMA LISTA QUE LOS PINTA. Es lo
	// que hace que anadir una cifra sin su rotulo rompa el inventario del
	// catalogo en vez de sacar una clave cruda en la pantalla.
	for _, c := range CifrasDeLaCuenta(CuentaVista{}) {
		out = append(out, c.Clave)
	}
	sort.Strings(out)
	return out
}

var claves = []string{
	// Marco, compartidas con las demas superficies.
	"ui.marca",
	"ui.saltar",
	"ui.pie.no_asesoramiento",

	"calendario.pantalla.titulo",
	"calendario.pantalla.error_render",
	"calendario.pantalla.error_fuente",

	// El estado vacio, con su siguiente paso (puerta D11-b).
	"calendario.pantalla.sin_alcance.titulo",
	"calendario.pantalla.sin_alcance.que_es",
	"calendario.pantalla.sin_alcance.paso",
	"calendario.pantalla.ics_sin_alcance",

	// La cabecera.
	"calendario.pantalla.de_quien",
	"calendario.pantalla.periodo",
	"calendario.pantalla.supuesto",
	"calendario.pantalla.aviso_perfil",
	"calendario.pantalla.ics",
	"calendario.pantalla.sin_ninguna_fecha",

	// LO VENCIDO, con su descargo pegado al dato y no en un pie.
	"calendario.pantalla.vencido.titulo",
	"calendario.pantalla.vencido.frase",
	"calendario.pantalla.vencido.desde",
	"calendario.pantalla.vencido.ciclos",

	// Las fechas.
	"calendario.pantalla.hito",
	"calendario.pantalla.derivacion",
	"calendario.pantalla.otra_lectura",

	// Las dos transiciones de la ventana.
	"calendario.pantalla.estrenos.titulo",
	"calendario.pantalla.estrenos.nota",
	"calendario.pantalla.ceses.titulo",
	"calendario.pantalla.ceses.nota",

	// Lo que obliga y no tiene fecha.
	"calendario.pantalla.sin_fecha.titulo",

	// LA NOTA AL FRENTE DEL BLOQUE QUE NO SALE DE TUS RESPUESTAS.
	//
	// Es la mitad que le faltaba al arreglo de la colocacion (P1 de
	// docs/hallazgos-d11.md). Las cuatro secciones que van detras de todo lo
	// tuyo se calculan ANTES de la aplicabilidad y cuentan el corpus entero;
	// bajarlas quita la insinuacion y no dice lo que pasa, y lo que hay que
	// decir es que plazum todavia no ha mirado si alguna de esas te alcanza.
	// Absolver de mas en silencio es el error simetrico de acusar en falso, y
	// es peor: una acusacion la corrige quien la lee.
	"calendario.pantalla.descarte.no_es_tuyo",

	// La cuenta, entera.
	"calendario.pantalla.cuenta.titulo",
	"calendario.pantalla.cuenta.ver",
	// El rotulo del enlace a una pagina aparte NO es el mismo que el de una
	// seccion de esta pagina, y no es un capricho: «Ver de que sale» promete
	// que la derivacion esta aqui abajo, y este enlace SACA de la pantalla. Un
	// rotulo que miente sobre a donde lleva es como se pierde a quien lo pulsa.
	"calendario.pantalla.cuenta.ver_lista",
	// La pagina de la cifra que no cabe en la vista normal.
	"calendario.cifra.no_alcanzados.titulo",
	// EL DESCARGO VA PARTIDO EN DOS, y no por gusto: el catalogo de la casa
	// rechaza un valor de mas de 240 caracteres, y con razon. Un rotulo de
	// interfaz no es un parrafo, y un parrafo ahi casi siempre es texto de una
	// norma colandose por la puerta de atras. La primera mitad es lo que NO se
	// esta diciendo; la segunda, por que se ensena igual.
	"calendario.cifra.no_es_tuyo",
	"calendario.cifra.no_es_tuyo.por_que",
	"calendario.cifra.volver",
	"calendario.cifra.vacia",
	"calendario.cifra.sin_alcance",
	// LA CIFRA QUE HAY QUE CREERSE LO DICE. Sin esto, el UNICO numero de la
	// pagina sin derivacion sale igual que los trece que si la tienen, y
	// entonces la derivacion de los trece no le sirve de nada al lector.
	"calendario.pantalla.cuenta.sin_abrir",
	// EL AVISO DE QUE LOS CUBOS NO CUADRAN, el mismo que el escalado pinta
	// desde su primer dia. Solo sale cuando hay descuadre.
	"calendario.pantalla.cuenta.descuadre",
	// Y LA PARTICION, LEIDA COMO UNA FRASE. El «= 218 + 9 + 1» se queda (los
	// signos no se traducen y son lo comprobable); lo que faltaba era decir de
	// que es esa suma.
	"calendario.pantalla.cuenta.se_compone_de",
	// LOS CATORCE ROTULOS DE LA CUENTA NO ESTAN AQUI: los pide
	// ClavesDeCatalogo desde CifrasDeLaCuenta, que es la lista que ademas los
	// pinta. Escritos aqui serian la TERCERA copia de CuentaVista (el tipo, la
	// plantilla y esta lista), y la copia sin puerta es la que se queda vieja.
}
