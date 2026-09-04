package escalado

// LA PUERTA D11-c DEL ESCALADO: ninguna cifra de esta pantalla se queda sin
// abrir, y la lista que se abre CUENTA LO QUE DICE SU CABECERA.
//
// # Por que las filas se cuentan en la RESPUESTA y no en el modelo
//
// Porque lo que el lector cuenta es la pagina. El 04-09-2026 el calendario se
// cobro un P0 exactamente aqui: tenia dos puertas, una comparaba `N` con
// `len(Filas)` EN EL MODELO y la otra comprobaba que el enlace y su destino
// EXISTIERAN, y entre las dos dejaron pasar una cifra que se abria a una seccion
// mas corta que ella. Un numero que no cuadra con su lista es PEOR que un numero
// sin enlace: el enlace prometia que se podia comprobar.
//
// # Y por que campo casa cada cosa (invariante 7)
//
//	el cubo con su lista     por `nucleo/escalado.Estado`, letra por letra. Es
//	                         la constante del vocabulario cerrado del motor, la
//	                         misma que indexa `Plan.Cuenta`, la misma que lleva
//	                         dentro cada `Paso` y la misma que viaja escapada en
//	                         la direccion. NUNCA por la posicion del cubo en la
//	                         lista ni por el orden de EstadosPosibles().
//	el sumando con su cubo   por el mismo campo, y por eso `ParteDelPlan` lleva
//	                         `Estado` dentro ademas del numero.

import (
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"testing"

	nescalado "github.com/marcosmatalab/plazum/nucleo/escalado"
	"github.com/marcosmatalab/plazum/superficies/camino"
)

// reFilaDeCubo cuenta las filas de la lista de una cifra abierta.
//
// CUENTA `<li>` DENTRO DE `<ul class="escalones">` Y NO EN LA PAGINA ENTERA: la
// pagina trae ademas la tira del camino, que es una lista, y contarla haria que
// la cifra pareciera cuadrar con filas que no son suyas.
var reFilaDeCubo = regexp.MustCompile(`(?s)<ul class="escalones">(.*?)</ul>`)

// filasDeLaDerivacion cuenta lo que el lector puede contar.
func filasDeLaDerivacion(t *testing.T, cuerpo string) int {
	t.Helper()
	m := reFilaDeCubo.FindStringSubmatch(cuerpo)
	if m == nil {
		t.Fatalf("la derivacion no trae su lista de escalones:\n%s", recorta(cuerpo, 700))
	}
	return strings.Count(m[1], "<li>")
}

// reCuboEnlazado saca los enlaces de la seccion de la cuenta, con su numero.
//
// SALE DE LA SECCION RECORTADA, por lo mismo que el control negativo de los
// cubos: buscar enlaces en la pagina entera traeria los de la barra lateral.
var reCuboEnlazado = regexp.MustCompile(
	`(?s)<a href="([^"]+)"[^>]*>\s*<span class="cubo-n">(\d+)</span>`)

// planQueCuadraConSusEscalones llena TRES cubos y pone en Trabajos exactamente
// los escalones que esos tres cubos cuentan.
//
// EXISTE PORQUE `planConLosOchoCubos` NO SIRVE PARA ESTO: aquel pone recuentos
// 1..8 sobre dos escalones, o sea que seis de sus ocho cubos descuadran a
// proposito. Es el dato bueno para el control del descuadre y el dato malo para
// el control de que la lista cuadra, y son dos ramas distintas.
func planQueCuadraConSusEscalones() Plan {
	p := Plan{
		Organizacion: "Acme SL",
		ComoMandar:   "plazum escalado --alcance alcance.json --mandar",
		Cuenta: map[nescalado.Estado]int{
			nescalado.Pendiente:       2,
			nescalado.SinDestinatario: 1,
			nescalado.EnSilencio:      1,
		},
		Planificados: 4,
	}
	nuevo := func(id, titulo string, pasos ...nescalado.Paso) Trabajo {
		return Trabajo{
			Obligacion: id, Titulo: titulo, Hito: "inicial", Vence: dia(2026, 10, 1),
			Pasos: pasos,
		}
	}
	p.Trabajos = []Trabajo{
		nuevo("m1.o1", "Notificacion de incidente grave",
			nescalado.Paso{
				Nivel: 1, Cuando: dia(2026, 9, 24), Figura: "m1.responsable",
				Persona: "Bea Nunez", Estado: nescalado.Pendiente,
				Aviso: &nescalado.Aviso{
					Obligacion: "m1.o1", Titulo: "Notificacion de incidente grave",
					Hito: "inicial", Vence: dia(2026, 10, 1),
					Figura: "m1.responsable", Nivel: 1, Enlace: "http://localhost:8443/x",
				},
			},
			nescalado.Paso{
				Nivel: 2, Cuando: dia(2026, 9, 30), Figura: "m1.direccion",
				Estado: nescalado.SinDestinatario,
				Motivo: "la figura m1.direccion no tiene persona en esta organizacion",
			}),
		nuevo("m1.o2", "Revision del plan de continuidad",
			nescalado.Paso{
				Nivel: 1, Cuando: dia(2026, 11, 2), Figura: "m1.responsable",
				Persona: "Bea Nunez", Estado: nescalado.Pendiente,
				Aviso: &nescalado.Aviso{
					Obligacion: "m1.o2", Titulo: "Revision del plan de continuidad",
					Hito: "inicial", Vence: dia(2026, 10, 1),
					Figura: "m1.responsable", Nivel: 1, Enlace: "http://localhost:8443/y",
				},
			},
			nescalado.Paso{
				Nivel: 2, Cuando: dia(2026, 11, 8), Figura: "m1.direccion",
				Estado: nescalado.EnSilencio,
				Motivo: "cae dentro de una ventana de silencio declarada",
			}),
	}
	return p
}

// planQueDescuadraEnUnCubo dice 5 pendientes y solo pone 2 escalones detras.
//
// ES DATO SINTETICO A PROPOSITO: el adaptador que monta hoy deriva `Cuenta` y
// `Trabajos` de los mismos pasos, asi que la rama del descuadre no la recorre el
// dato real y sin este plan seria una rama que no existe (M47).
func planQueDescuadraEnUnCubo() Plan {
	p := planQueCuadraConSusEscalones()
	p.Cuenta[nescalado.Pendiente] = 5
	p.Planificados = 8
	return p
}

// marcaConArgumentos es el PREFIJO del marcador del espia cuando la clave lleva
// huecos.
//
// EL ESPIA PEGA LOS ARGUMENTOS DENTRO DEL MARCADOR (`[[clave[5 2]]]`), asi que
// `marca(clave)` no casa nunca con una cadena que reciba datos y un test escrito
// con `marca` daria un rojo que no es del producto. Se pregunta por el prefijo,
// que es lo que identifica a la clave.
func marcaConArgumentos(clave string) string { return "[[" + clave }

// TestNingunaCifraDelEscaladoSeQuedaSinSuDerivacion es la puerta.
//
// Recorre las DOS direcciones, y las dos hacen falta:
//
//	de la pagina al enlace   todo numero pintado en la cuenta lleva enlace, y
//	                         ese enlace contesta 200.
//	del enlace a la lista    lo que hay al otro lado CUENTA lo que decia la
//	                         cabecera. Sin esto, un enlace a una pagina vacia
//	                         pasaria la primera direccion entera.
func TestNingunaCifraDelEscaladoSeQuedaSinSuDerivacion(t *testing.T) {
	p := planQueCuadraConSusEscalones()
	s, _ := pantallaDePrueba(t, fuenteDoble{p: p, hay: true}, conSesion)
	_, cuerpo := pedir(t, s, http.MethodGet, BasePorDefecto+"/")
	cuenta := seccionDeLaCuenta(t, cuerpo)

	enlaces := reCuboEnlazado.FindAllStringSubmatch(cuenta, -1)
	esperados := EstadosQueSePintan(p)
	if len(esperados) == 0 {
		t.Fatal("el plan de prueba no pinta ni un cubo: esta puerta estaria recorriendo el vacio")
	}
	if len(enlaces) != len(esperados) {
		t.Fatalf("la cuenta pinta %d cubos y solo %d llevan enlace.\n"+
			"  Una cifra sin enlace es una cifra que hay que creerse, y este producto se "+
			"vende exactamente al reves.\n%s", len(esperados), len(enlaces), recorta(cuenta, 900))
	}

	for _, e := range esperados {
		ruta := EnlaceDelCubo(BasePorDefecto, e)
		codigo, pagina := pedir(t, s, http.MethodGet, ruta)
		if codigo != http.StatusOK {
			t.Errorf("el cubo %q se pinta en la cuenta y su derivacion contesta %d en %s",
				e, codigo, ruta)
			continue
		}
		n := p.Cuenta[nescalado.Estado(e)]
		filas := filasDeLaDerivacion(t, pagina)
		if filas != n {
			t.Errorf("la cabecera del cubo %q cuenta %d y su derivacion pinta %d filas.\n"+
				"  Quien pulse esa cifra va a contar las filas y le van a salir %d. Un numero "+
				"que no cuadra con su lista es PEOR que un numero sin enlace: el enlace "+
				"prometia que se podia comprobar.", e, n, filas, filas)
		}
	}

	// Y EL CARDINAL, con igualdad exacta en los dos sentidos.
	if SinDerivacionEsperadas != 0 {
		t.Errorf("SinDerivacionEsperadas dice %d y esta pantalla abre todas sus cifras.\n"+
			"  Un hueco que se cierra y deja su cardinal puesto miente hacia arriba para "+
			"siempre.", SinDerivacionEsperadas)
	}
}

// TestLaParticionDeLosAvisosPlanificadosSeEscribeYCuadra.
//
// La segunda forma de abrir una cifra: el total no tiene lista propia (seria la
// union de todos los cubos, o sea la pagina entera repetida) y se abre SUMANDO
// unos numeros que ya estan delante. No es circular: cada sumando se sostiene
// solo, con su enlace y su lista.
//
// HAY PUERTA PARA LAS DOS MITADES y hacen falta las dos: se puede escribir la
// frase y quitar la suma (y entonces el total vuelve a ser un numero que hay que
// creerse, con una frase encima), y se puede dejar la suma sin frase, que se lee
// como una formula y no como una afirmacion.
func TestLaParticionDeLosAvisosPlanificadosSeEscribeYCuadra(t *testing.T) {
	p := planQueCuadraConSusEscalones()
	var v Vista
	v.rellenarCon(p, BasePorDefecto)

	if len(v.Partes) == 0 {
		t.Fatal("el total no declara sumandos: no habria particion que comprobar")
	}
	suma := 0
	porEstado := map[string]int{}
	for _, parte := range v.Partes {
		suma += parte.N
		porEstado[parte.Estado] = parte.N
	}
	if suma != v.Planificados {
		t.Errorf("los sumandos del total suman %d y el total dice %d.\n"+
			"  Una particion que no cuadra es la misma promesa rota que un enlace a una lista "+
			"mas corta, escrita con numeros", suma, v.Planificados)
	}
	// EL SUMANDO CASA CON SU CUBO POR EL ESTADO, no por la posicion.
	for _, c := range v.Cuenta {
		n, hay := porEstado[c.Estado]
		if !hay {
			t.Errorf("el cubo %q se pinta y no es sumando del total", c.Estado)
			continue
		}
		if n != c.N {
			t.Errorf("el cubo %q vale %d y como sumando del total vale %d", c.Estado, c.N, n)
		}
	}
	if len(v.Partes) != len(v.Cuenta) {
		t.Errorf("hay %d cubos pintados y %d sumandos", len(v.Cuenta), len(v.Partes))
	}

	// LAS DOS MITADES, EN LA PAGINA. La frase y la aritmetica.
	s, esp := pantallaDePrueba(t, fuenteDoble{p: p, hay: true}, conSesion)
	_, cuerpo := pedir(t, s, http.MethodGet, BasePorDefecto+"/")
	cuenta := seccionDeLaCuenta(t, cuerpo)
	if !esp.pidio("escalado.pantalla.cuenta.se_compone_de") {
		t.Error("la particion se pinta sin decir con palabras que lo es: se lee como una " +
			"formula y no como una afirmacion")
	}
	if !strings.Contains(cuenta, `<span class="particion">`) {
		t.Errorf("la cuenta no trae la particion escrita:\n%s", recorta(cuenta, 700))
	}
	// LA ARITMETICA, escrita con los signos que no se traducen.
	if !strings.Contains(cuenta, " = ") || !strings.Contains(cuenta, " + ") {
		t.Errorf("la particion no escribe la suma: sin ella la frase promete una "+
			"comprobacion que la pagina no da.\n%s", recorta(cuenta, 700))
	}
}

// TestUnaCifraQueNoEstaEnLaPaginaNoSeAbreConUnaListaVacia.
//
// Invariante 8 en su TERCERA forma, la de la frontera de ENTRADA: ausente,
// presente-y-conocido, y presente-y-no-interpretable son TRES cosas y solo las
// dos primeras son la nada. Un estado que esta pagina no pinta es un dato que
// HAY y no se entiende, y contestarlo con una lista vacia seria inventarse un
// valor: se leeria como «ese cubo vale cero», que es una afirmacion sobre el
// plan cuando lo unico que pasa es que la direccion no nombra ninguna cifra.
func TestUnaCifraQueNoEstaEnLaPaginaNoSeAbreConUnaListaVacia(t *testing.T) {
	p := planQueCuadraConSusEscalones()
	s, _ := pantallaDePrueba(t, fuenteDoble{p: p, hay: true}, conSesion)

	casos := []struct{ nombre, estado string }{
		// Un estado que no existe en ningun sitio.
		{"inventado", "un estado que nadie ha escrito"},
		// UN ESTADO DE LA PARTICION CUYO CUBO VALE CERO, que es el caso
		// peligroso: existe en el vocabulario del nucleo y NO es una cifra de
		// esta pagina. Una implementacion que mirara solo el vocabulario lo
		// serviria con cero filas y diria que ese cubo vale cero.
		{"cubo en cero", string(nescalado.Atendido)},
	}
	for _, c := range casos {
		ruta := EnlaceDelCubo(BasePorDefecto, c.estado)
		codigo, pagina := pedir(t, s, http.MethodGet, ruta)
		if codigo != http.StatusNotFound {
			t.Errorf("%s: pedir %q contesta %d y tenia que contestar 404", c.nombre, ruta, codigo)
		}
		if strings.Contains(pagina, `<ul class="escalones">`) {
			t.Errorf("%s: la pagina trae una lista de escalones, o sea que contesta con una "+
				"derivacion vacia en vez de decir que esa cifra no esta", c.nombre)
		}
		if !strings.Contains(pagina, marca("escalado.pantalla.cubo.no_existe.titulo")) {
			t.Errorf("%s: la pagina no dice que esa cifra no esta:\n%s",
				c.nombre, recorta(pagina, 500))
		}
	}
}

// TestLaDerivacionDeUnCuboDiceCuandoNoCuadraConSuCabecera.
//
// EL CONTROL POSITIVO DE UNA RAMA QUE EL DATO REAL NO RECORRE (M47). El
// adaptador que monta hoy deriva `Cuenta` y `Trabajos` de los mismos pasos, asi
// que cuadran por construccion: sin dato sintetico esta rama no la ejecuta
// nadie, y la mutacion la dejaria verde porque no habria nada que romper.
//
// `Fuente` es un puerto y los dos lados llegan por el: quien lo implemente puede
// darlos separados, y entonces la pantalla tiene que decirlo en vez de pintar
// una lista corta debajo de un numero grande.
func TestLaDerivacionDeUnCuboDiceCuandoNoCuadraConSuCabecera(t *testing.T) {
	// La cabecera dice 5 y en los trabajos solo hay 2 escalones pendientes.
	p := planQueDescuadraEnUnCubo()
	s, _ := pantallaDePrueba(t, fuenteDoble{p: p, hay: true}, conSesion)

	ruta := EnlaceDelCubo(BasePorDefecto, string(nescalado.Pendiente))
	codigo, pagina := pedir(t, s, http.MethodGet, ruta)
	if codigo != http.StatusOK {
		t.Fatalf("la derivacion contesta %d", codigo)
	}
	if !strings.Contains(pagina, marcaConArgumentos("escalado.pantalla.cubo.descuadre")) {
		t.Errorf("la cabecera dice 5, la lista trae %d y la pagina no lo dice.\n"+
			"  Una lista mas corta que su cabecera, callada, es la promesa rota que el enlace "+
			"venia a evitar.\n%s", filasDeLaDerivacion(t, pagina), recorta(pagina, 700))
	}

	// Y EL CONTROL NEGATIVO: sobre un plan que cuadra, la pagina no se inventa
	// el aviso. Sin esto, un aviso que saliera siempre pasaria el positivo.
	s2, _ := pantallaDePrueba(t, fuenteDoble{p: planQueCuadraConSusEscalones(), hay: true},
		conSesion)
	_, buena := pedir(t, s2, http.MethodGet, ruta)
	if strings.Contains(buena, marcaConArgumentos("escalado.pantalla.cubo.descuadre")) {
		t.Errorf("la derivacion de un cubo que cuadra avisa de descuadre:\n%s",
			recorta(buena, 700))
	}
}

// TestLaDerivacionDeUnCuboExigeSesionIgualQueElPlan.
//
// LA RUTA NUEVA LLEVA NOMBRES DE PERSONAS DENTRO, o sea exactamente el
// organigrama de responsabilidades de cumplimiento que el encabezado del paquete
// dice que no se sirve a quien no ha entrado. Una ruta nueva que se olvidara de
// la sesion publicaria por la puerta de atras lo que la principal protege por la
// de delante, y eso no lo ve ninguna puerta de la pantalla principal.
func TestLaDerivacionDeUnCuboExigeSesionIgualQueElPlan(t *testing.T) {
	p := planQueCuadraConSusEscalones()
	s, _ := pantallaDePrueba(t, fuenteDoble{p: p, hay: true}, nil)
	ruta := EnlaceDelCubo(BasePorDefecto, string(nescalado.Pendiente))
	codigo, pagina := pedir(t, s, http.MethodGet, ruta)
	if codigo != http.StatusUnauthorized {
		t.Errorf("la derivacion de un cubo contesta %d sin sesion", codigo)
	}
	// Y NO SOLO EL CODIGO: el nombre de la persona no puede estar en el cuerpo.
	if strings.Contains(pagina, "Bea Nunez") {
		t.Errorf("la derivacion sin sesion trae el nombre de una persona dentro:\n%s",
			recorta(pagina, 500))
	}

	// CONTROL POSITIVO: con sesion, ese mismo nombre SI sale. Sin esta mitad, un
	// 401 permanente pasaria la comprobacion de arriba diga lo que diga la
	// pagina, y la puerta no distinguiria «protegido» de «roto».
	s2, _ := pantallaDePrueba(t, fuenteDoble{p: p, hay: true}, conSesion)
	if _, con := pedir(t, s2, http.MethodGet, ruta); !strings.Contains(con, "Bea Nunez") {
		t.Errorf("con sesion la derivacion no trae los nombres, asi que el 401 de arriba no " +
			"demuestra nada")
	}
}

// TestElEnlaceDeUnCuboSobreviveAlEscapadoDeLaDireccion.
//
// CINCO DE LOS OCHO ESTADOS LLEVAN ESPACIOS («sin destinatario», «colapsado en
// un escalon anterior»). Un enlace compuesto sin escapar produce una direccion
// que el navegador normaliza y que no casa con ninguna constante, y el sintoma
// seria un 404 en la mitad de los cubos: la cifra huerfana otra vez, con una
// capa de pintura encima.
func TestElEnlaceDeUnCuboSobreviveAlEscapadoDeLaDireccion(t *testing.T) {
	conEspacios := 0
	for _, e := range nescalado.EstadosPosibles() {
		if strings.Contains(string(e), " ") {
			conEspacios++
		}
	}
	if conEspacios < 2 {
		t.Fatalf("solo %d estados llevan espacio: este test estaria recorriendo el caso facil",
			conEspacios)
	}
	p := planConLosOchoCubos()
	s, _ := pantallaDePrueba(t, fuenteDoble{p: p, hay: true}, conSesion)
	for _, e := range nescalado.EstadosPosibles() {
		ruta := EnlaceDelCubo(BasePorDefecto, string(e))
		if strings.Contains(string(e), " ") && !strings.Contains(ruta, "%20") {
			t.Errorf("el enlace del cubo %q no escapa el espacio: %s", e, ruta)
		}
		codigo, _ := pedir(t, s, http.MethodGet, ruta)
		if codigo != http.StatusOK {
			t.Errorf("el cubo %q se pinta y su enlace %s contesta %d", e, ruta, codigo)
		}
	}
	// Y LA VUELTA: la direccion escapada se lee como la constante del nucleo.
	for _, e := range nescalado.EstadosPosibles() {
		crudo, err := url.PathUnescape(strings.TrimPrefix(
			EnlaceDelCubo("", string(e)), SegmentoDelCubo))
		if err != nil {
			t.Errorf("el enlace del cubo %q no se puede desescapar: %v", e, err)
			continue
		}
		if crudo != string(e) {
			t.Errorf("el enlace del cubo %q se desescapa a %q", e, crudo)
		}
	}
}

// TestLaRutaDelCuboSigueSiendoSoloGET.
//
// La propiedad que sostiene esta pantalla entera es que NO MANDA NADA. Una ruta
// nueva es una via nueva, asi que se comprueba en la ruta nueva y no solo en la
// vieja: se puede registrar solo GET en una y abrir un POST en la otra.
func TestLaRutaDelCuboSigueSiendoSoloGET(t *testing.T) {
	s, _ := pantallaDePrueba(t, fuenteDoble{p: planQueCuadraConSusEscalones(), hay: true},
		conSesion)
	ruta := EnlaceDelCubo(BasePorDefecto, string(nescalado.Pendiente))
	for _, m := range []string{http.MethodPost, http.MethodPut, http.MethodDelete,
		http.MethodPatch} {
		if codigo, _ := pedir(t, s, m, ruta); codigo == http.StatusOK {
			t.Errorf("%s contra %s contesta 200", m, ruta)
		}
	}
	// Y la ruta declarada tiene que empezar por GET, como todas.
	hay := false
	for _, patron := range s.Patrones() {
		if strings.Contains(patron, SegmentoDelCubo) {
			hay = true
			if !strings.HasPrefix(patron, "GET ") {
				t.Errorf("la ruta del cubo se registra como %q", patron)
			}
		}
	}
	if !hay {
		t.Errorf("no hay ninguna ruta registrada con %q: los enlaces de la cuenta llevarian "+
			"a un 404", SegmentoDelCubo)
	}
	// Y NI UN FORMULARIO EN LA DERIVACION.
	_, pagina := pedir(t, s, http.MethodGet, ruta)
	for _, prohibido := range []string{"<form", "<button", "method="} {
		if strings.Contains(pagina, prohibido) {
			t.Errorf("la derivacion de un cubo trae %q dentro", prohibido)
		}
	}
}

// TestLosEstadosSueltosTambienSeAbren.
//
// El respaldo de `CuboVista.Clave` existe para un estado que el plan traiga y la
// particion del nucleo no nombre. Hasta hoy ese estado se pintaba CON rotulo y
// SIN enlace, o sea que seguia siendo una cifra huerfana con nombre. La
// derivacion casa por la palabra del nucleo, que un estado suelto tiene igual
// que los ocho de la particion.
func TestLosEstadosSueltosTambienSeAbren(t *testing.T) {
	inventado := nescalado.Estado("devuelto por el servidor de correo")
	p := planQueCuadraConSusEscalones()
	p.Cuenta[inventado] = 1
	p.Planificados++
	// Y CON SU ESCALON DETRAS, para que la lista cuadre: sin el, este test
	// estaria comprobando el enlace y no la derivacion.
	p.Trabajos[0].Pasos = append(p.Trabajos[0].Pasos, nescalado.Paso{
		Nivel: 3, Cuando: dia(2026, 10, 5), Figura: "m1.direccion",
		Estado: inventado, Motivo: "el servidor de correo lo rechazo",
	})

	s, _ := pantallaDePrueba(t, fuenteDoble{p: p, hay: true}, conSesion)
	ruta := EnlaceDelCubo(BasePorDefecto, string(inventado))
	codigo, pagina := pedir(t, s, http.MethodGet, ruta)
	if codigo != http.StatusOK {
		t.Fatalf("el cubo suelto %q se pinta y su derivacion contesta %d", inventado, codigo)
	}
	if filas := filasDeLaDerivacion(t, pagina); filas != 1 {
		t.Errorf("el cubo suelto dice 1 y su derivacion pinta %d filas", filas)
	}
	// Y SU ROTULO ES LA PALABRA DEL NUCLEO, que es cierta aunque este en otro
	// idioma: no hay clave que pedirle al catalogo.
	if !strings.Contains(pagina, string(inventado)) {
		t.Errorf("la derivacion del cubo suelto no dice de que cubo es:\n%s",
			recorta(pagina, 500))
	}
}

// TestLaBarraLateralSaleTambienEnLaDerivacionDeUnCubo.
//
// Una pagina del producto sin armazon es una pagina de la que no se sale: el
// camino guiado es la unica respuesta a «y ahora que», y una ruta nueva que se
// lo saltara dejaria al lector en un callejon con datos dentro.
func TestLaBarraLateralSaleTambienEnLaDerivacionDeUnCubo(t *testing.T) {
	s, _ := pantallaDePrueba(t, fuenteDoble{p: planQueCuadraConSusEscalones(), hay: true},
		conSesion)
	ruta := EnlaceDelCubo(BasePorDefecto, string(nescalado.Pendiente))
	_, pagina := pedir(t, s, http.MethodGet, ruta)
	if !strings.Contains(pagina, `class="tira-camino"`) {
		t.Errorf("la derivacion de un cubo no pinta la barra lateral:\n%s", recorta(pagina, 600))
	}
	if len(camino.Canonico()) == 0 {
		t.Fatal("el camino esta vacio: esta comprobacion no estaria mirando nada")
	}
	// Y LA VUELTA AL PLAN, que es de donde se ha venido.
	if !strings.Contains(pagina, marca("escalado.pantalla.cubo.volver")) {
		t.Errorf("la derivacion no ofrece volver al plan:\n%s", recorta(pagina, 600))
	}
}
