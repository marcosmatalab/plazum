package corpus

import (
	"errors"
	"strings"
	"testing"
)

// Las puertas del identificador de fuente.
//
// LA PROPIEDAD QUE SE SOSTIENE AQUI: en un paquete NO se guarda una direccion,
// se guarda una identidad, y la direccion se deriva. Todo lo de abajo son
// intentos de meter una direccion por alguna puerta lateral, mas la
// comprobacion de que la derivacion no se puede desviar a otro anfitrion.
//
// Los identificadores son SINTETICOS donde se puede (urn:demo:). Donde no se
// puede es en la forma de un ELI o de una designacion ISO, que es justo lo que
// se esta validando: ahi el dato es la forma, no la norma.

// conIdent devuelve el paquete base con otro identificador.
func conIdent(id Identificador) *Paquete {
	p := base()
	p.Identificador = id
	return p
}

// buscar dice si alguno de los errores es el centinela. Por identidad y no por
// subcadena del mensaje: ese patron ya dio aqui siete verdes con el fallo
// delante.
func buscar(errs []error, centinela error) bool {
	for _, e := range errs {
		if errors.Is(e, centinela) {
			return true
		}
	}
	return false
}

func exigir(t *testing.T, p *Paquete, centinela error, que string) {
	t.Helper()
	errs := p.Validar()
	if !buscar(errs, centinela) {
		t.Fatalf("%s: el linter no dio %v. Dio: %v", que, centinela, errs)
	}
}

// EL LINTER EXIGE EL IDENTIFICADOR. Sin el no hay de donde derivar la cita, y
// citar la fuente es condicion de reutilizacion del BOE y de la Decision
// 2011/833/UE.
func TestElIdentificadorDeFuenteEsObligatorio(t *testing.T) {
	exigir(t, conIdent(Identificador{}), ErrSinIdentificador, "paquete sin identificador")
	exigir(t, conIdent(Identificador{Tipo: ELIBOE}), ErrSinIdentificador, "tipo sin valor")
}

// El vocabulario es CERRADO. Un esquema nuevo entra con su constante y su rama
// en Enlace, que es lo que hace que el enlace se pueda derivar; si entrara
// escribiendo otra cadena en un JSON, la derivacion no podria existir.
func TestUnEsquemaFueraDelVocabularioNoCarga(t *testing.T) {
	exigir(t, conIdent(Identificador{Tipo: "eli-inventado", Valor: "x/y"}),
		ErrEsquemaDesconocido, "esquema inventado")
}

// EL ATAQUE PRINCIPAL: colar la URL entera por la puerta nueva. Es lo que sale
// solo al migrar (copiar el valor viejo al campo nuevo) y es exactamente el
// formato que este cambio retira.
func TestUnaDireccionDentroDelIdentificadorNoCarga(t *testing.T) {
	// Las direcciones se COMPONEN con los prefijos de la propia derivacion, no
	// se escriben aqui. Dos razones: es exactamente lo que hace quien migra (le
	// pega al campo nuevo el valor viejo, que era la direccion entera), y asi
	// el anfitrion de cada editor sigue estando escrito en un solo sitio del
	// repositorio, que es lo que vigila TestSoloUnaFuncionComponeDireccionesDeEditor.
	casos := []Identificador{
		{Tipo: ELIUE, Valor: prefijoELIUE + "reg/2016/679/oj"},
		{Tipo: ELIBOE, Valor: prefijoELIBOE + "es/rd/2022/05/03/311/con"},
		{Tipo: PublicacionCSRC, Valor: prefijoCSRC + "sp/800/53/r5/upd1" + sufijoCSRC},
		{Tipo: NormaISO, Valor: prefijoISO + "27001" + sufijoISO, Registro: "27001"},
		{Tipo: VersionPCIDSS, Valor: bibliotecaPCI},
		// Y las formas de al lado: barra inicial, barra final, mayusculas y
		// espacios. Ninguna es la ruta de un ELI.
		{Tipo: ELIUE, Valor: "/reg/2016/679/oj"},
		{Tipo: ELIUE, Valor: "reg/2016/679/oj/"},
		{Tipo: ELIUE, Valor: "REG/2016/679/oj"},
		{Tipo: ELIUE, Valor: "reg/2016 /679/oj"},
		{Tipo: ELIUE, Valor: "reg"},
		{Tipo: NormaISO, Valor: "27001", Registro: "27001"},
		{Tipo: VersionPCIDSS, Valor: "v4.0"},
	}
	for _, id := range casos {
		exigir(t, conIdent(id), ErrIdentificadorMalFormado, "valor "+id.Valor)
	}
}

// El formato viejo se RECHAZA, no se ignora. Un campo que se lee y se tira deja
// a quien lo escribio creyendo que su paquete cita la fuente.
func TestElCampoFuenteDelFormatoViejoSeRechaza(t *testing.T) {
	p := base()
	p.FuenteHeredada = prefijoELIBOE + "es/rd/2022/05/03/311/con"
	exigir(t, p, ErrFuenteHeredada, "paquete con fuente heredada")
	// Y el mensaje tiene que decir que hacer, no solo que esta mal.
	var mensaje string
	for _, e := range p.Validar() {
		if errors.Is(e, ErrFuenteHeredada) {
			mensaje = e.Error()
		}
	}
	for _, quiero := range []string{"identificador", "borra fuente"} {
		if !strings.Contains(mensaje, quiero) {
			t.Errorf("el error de la fuente heredada no dice %q y no es accionable: %s",
				quiero, mensaje)
		}
	}
}

// La valvula de escape EXIGE motivo, y solo la valvula lo admite. Las dos
// direcciones: un hueco sin motivo se lee como un descuido y nadie vuelve a
// mirarlo, y un motivo al lado de un identificador estable no lo lee nadie.
func TestLaValvulaDeEscapeExigeMotivoYSoloElla(t *testing.T) {
	exigir(t, conIdent(Identificador{Tipo: SinIdentificador, Valor: "https://ejemplo.invalid/x"}),
		ErrSinMotivo, "sin-identificador sin motivo")
	exigir(t, conIdent(Identificador{Tipo: SinIdentificador, Valor: "https://ejemplo.invalid/x",
		Motivo: "   "}), ErrSinMotivo, "motivo en blanco")
	exigir(t, conIdent(Identificador{Tipo: ELIUE, Valor: "reg/2016/679/oj",
		Motivo: "porque si"}), ErrMotivoSobrante, "motivo en un esquema estable")
}

// Y la valvula no es barra libre de esquemas: javascript:, data: y http:// no
// entran. Hoy la fuente se pinta como texto, asi que no hay inyeccion; el dia
// que esa superficie decida hacer enlaces de verdad, lo que ya este guardado en
// el corpus es lo que se va a pintar dentro de un href.
func TestLaValvulaDeEscapeNoAdmiteCualquierEsquema(t *testing.T) {
	for _, malo := range []string{
		"javascript:alert(1)",
		"data:text/html;base64,PHNjcmlwdD4=",
		"http://ejemplo.invalid/x",
		"//ejemplo.invalid/x",
		"https://ejemplo.invalid/ con espacio",
		"file:///etc/passwd",
	} {
		exigir(t, conIdent(Identificador{Tipo: SinIdentificador, Valor: malo,
			Motivo: "prueba"}), ErrIdentificadorMalFormado, "valvula con "+malo)
	}
	// Y las dos formas que SI valen: una direccion https y una ruta del propio
	// repositorio, que es como se declara un paquete de datos del proyecto.
	for _, bueno := range []string{
		"https://ejemplo.invalid/x",
		"paquetes/demo-empresa/LEEME.md",
	} {
		p := conIdent(Identificador{Tipo: SinIdentificador, Valor: bueno, Motivo: "prueba"})
		if errs := p.Validar(); len(errs) != 0 {
			t.Errorf("%q tenia que valer y el linter dio: %v", bueno, errs)
		}
	}
}

// El registro del catalogo del editor: solo ISO, y ISO siempre. Las dos
// direcciones, porque un registro suelto en otro esquema es un dato que nadie
// lee (y el dia que alguien lo mire, mentira) y un ISO sin registro no deriva
// ninguna direccion.
func TestElRegistroEsDeISOYSoloDeISO(t *testing.T) {
	exigir(t, conIdent(Identificador{Tipo: NormaISO, Valor: "ISO/IEC 27001:2022"}),
		ErrRegistroDelEditor, "ISO sin registro")
	exigir(t, conIdent(Identificador{Tipo: NormaISO, Valor: "ISO/IEC 27001:2022",
		Registro: "27001.html"}), ErrRegistroDelEditor, "registro que es media direccion")
	exigir(t, conIdent(Identificador{Tipo: ELIUE, Valor: "reg/2016/679/oj",
		Registro: "75652"}), ErrRegistroDelEditor, "registro fuera de ISO")
}

// Un paquete bien declarado con cada esquema del vocabulario carga. Sin esto,
// todo lo de arriba se demuestra rechazando y nadie comprueba que se pueda
// aceptar: un linter que lo rechaza todo tambien pasa los tests de rechazo.
func TestCadaEsquemaDelVocabularioCargaBienEscrito(t *testing.T) {
	buenos := map[TipoIdentificador]Identificador{
		ELIUE:           {Tipo: ELIUE, Valor: "reg/2016/679/oj"},
		ELIBOE:          {Tipo: ELIBOE, Valor: "es/rd/2022/05/03/311/con"},
		NormaISO:        {Tipo: NormaISO, Valor: "ISO/IEC 27001:2022", Registro: "27001"},
		VersionPCIDSS:   {Tipo: VersionPCIDSS, Valor: "4.0"},
		PublicacionCSRC: {Tipo: PublicacionCSRC, Valor: "sp/800/53/r5/upd1"},
		SinIdentificador: {Tipo: SinIdentificador, Valor: "https://ejemplo.invalid/x",
			Motivo: "el editor no publica identificador citable"},
	}
	// Exhaustividad: si manana entra un esquema al vocabulario y nadie escribe
	// aqui su ejemplo bueno, esto lo dice. Es la mitad que se olvida.
	for _, e := range esquemas {
		if _, hay := buenos[e]; !hay {
			t.Fatalf("el esquema %q esta en el vocabulario y no tiene ejemplo bueno aqui: "+
				"un esquema sin caso de aceptacion es un esquema que nadie ha probado", e)
		}
	}
	if len(buenos) != len(esquemas) {
		t.Fatalf("hay %d ejemplos buenos y %d esquemas", len(buenos), len(esquemas))
	}
	for e, id := range buenos {
		if errs := conIdent(id).Validar(); len(errs) != 0 {
			t.Errorf("el esquema %q bien escrito no carga: %v", e, errs)
		}
	}
}

// TODO esquema del vocabulario deriva una direccion. Es el fallo que deja un
// esquema nuevo sin su rama en Enlace: el paquete carga, y en la pantalla sale
// una fuente sin fuente.
func TestTodoEsquemaDelVocabularioDerivaUnaDireccion(t *testing.T) {
	valores := map[TipoIdentificador]Identificador{
		ELIUE:            {Tipo: ELIUE, Valor: "reg/2016/679/oj"},
		ELIBOE:           {Tipo: ELIBOE, Valor: "es/rd/2022/05/03/311/con"},
		NormaISO:         {Tipo: NormaISO, Valor: "ISO/IEC 27001:2022", Registro: "27001"},
		VersionPCIDSS:    {Tipo: VersionPCIDSS, Valor: "4.0"},
		PublicacionCSRC:  {Tipo: PublicacionCSRC, Valor: "sp/800/53/r5/upd1"},
		SinIdentificador: {Tipo: SinIdentificador, Valor: "https://ejemplo.invalid/x"},
	}
	for _, e := range valores {
		if enlace := e.Enlace(); !strings.HasPrefix(enlace, "https://") {
			t.Errorf("el esquema %q no deriva direccion (%q). Un esquema del vocabulario "+
				"sin rama en Enlace deja la fuente en blanco en la pantalla", e.Tipo, enlace)
		}
	}
	for _, e := range esquemas {
		if _, hay := valores[e]; !hay {
			t.Fatalf("el esquema %q no se prueba aqui: falta su fila", e)
		}
	}
}

// EL ANFITRION LO DECIDE LA FUNCION, NUNCA EL PAQUETE.
//
// El valor entra siempre DETRAS de una barra del prefijo, asi que la autoridad
// de la URL ya esta cerrada cuando el dato se concatena. Esto lo fija contra
// los valores con los que se intenta romper una concatenacion de URLs. Se
// prueba aunque el linter ya rechace esas formas: la derivacion tiene que ser
// segura POR SI MISMA, porque un dia alguien la llamara con un paquete que no
// paso por el linter (un corpus de otro origen, una prueba, un adaptador).
//
// SinIdentificador queda fuera a proposito: ahi la direccion ES el dato, y por
// eso el linter le cierra el esquema en vez de fiarse de la forma.
func TestElAnfitrionDelEnlaceNoLoDecideElPaquete(t *testing.T) {
	// El anfitrion esperado se toma del PREFIJO de la propia derivacion, y eso
	// no es circular: lo que se comprueba aqui no es cual es el anfitrion (eso
	// lo fija TestSoloUnaFuncionComponeDireccionesDeEditor, con los anfitriones
	// escritos a mano y fuera de este paquete), es que el VALOR del paquete no
	// puede sacar la direccion de el, sea cual sea.
	anfitrion := map[TipoIdentificador]string{
		ELIUE:           prefijoELIUE,
		ELIBOE:          prefijoELIBOE,
		NormaISO:        prefijoISO,
		VersionPCIDSS:   bibliotecaPCI,
		PublicacionCSRC: prefijoCSRC,
	}
	hostiles := []string{
		"evil.invalid",
		"/evil.invalid/x",
		"//evil.invalid/x",
		"@evil.invalid/x",
		"..//evil.invalid",
		"x?a=https://evil.invalid",
		"x#https://evil.invalid",
		"\nHost: evil.invalid",
		"https://evil.invalid",
	}
	for tipo, quiero := range anfitrion {
		for _, v := range hostiles {
			id := Identificador{Tipo: tipo, Valor: v, Registro: v}
			if e := id.Enlace(); !strings.HasPrefix(e, quiero) {
				t.Errorf("con %q el esquema %q deriva %q, que se sale del anfitrion %q",
					v, tipo, e, quiero)
			}
		}
	}
}

// El enlace se DERIVA: cambiar el identificador cambia la direccion. Control
// negativo de la derivacion entera, porque una funcion que devolviera siempre
// el portal del editor pasaria todo lo de arriba sin despeinarse.
func TestCambiarElIdentificadorCambiaLaDireccion(t *testing.T) {
	uno := Identificador{Tipo: ELIUE, Valor: "reg/2016/679/oj"}
	otro := Identificador{Tipo: ELIUE, Valor: "dir/2022/2555/oj"}
	if uno.Enlace() == otro.Enlace() {
		t.Fatalf("dos identificadores distintos derivan la misma direccion (%q): la "+
			"derivacion no esta mirando el dato", uno.Enlace())
	}
	if !strings.HasSuffix(uno.Enlace(), uno.Valor) {
		t.Errorf("la direccion derivada (%q) no lleva dentro el identificador (%q)",
			uno.Enlace(), uno.Valor)
	}
	// Y un tipo que no existe no adivina una direccion: mejor una fuente vacia,
	// que se ve, que una direccion inventada, que no.
	if e := (Identificador{Tipo: "inventado", Valor: "x"}).Enlace(); e != "" {
		t.Errorf("un esquema desconocido deriva %q en vez de nada", e)
	}
	// El atajo del paquete es el mismo enlace y no otra copia de la logica.
	// Es lo que llaman las pantallas, asi que si se separara, la superficie
	// pintaria una direccion distinta de la que deriva el nucleo.
	p := base()
	if p.Enlace() != p.Identificador.Enlace() {
		t.Errorf("Paquete.Enlace (%q) no es Identificador.Enlace (%q): son dos "+
			"derivaciones distintas de lo mismo", p.Enlace(), p.Identificador.Enlace())
	}
	if p.Enlace() == "" {
		t.Error("el paquete base no deriva ninguna direccion")
	}
}
