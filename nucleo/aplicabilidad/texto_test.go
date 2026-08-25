package aplicabilidad

import (
	"errors"
	"strings"
	"testing"
)

func TestParsearUnaReglaSencilla(t *testing.T) {
	r, err := ParsearRegla(`aplica(x.auditoria, S) :- categoria(S, "MEDIA")`)
	if err != nil {
		t.Fatalf("no deberia fallar: %v", err)
	}
	if r.Cabeza.Pred != "aplica" || len(r.Cabeza.Args) != 2 {
		t.Fatalf("cabeza mal parseada: %v", r.Cabeza)
	}
	if r.Cabeza.Args[0].Var {
		t.Error("x.auditoria va en minuscula, o sea que es constante, no variable")
	}
	if !r.Cabeza.Args[1].Var {
		t.Error("S empieza por mayuscula, o sea que es variable")
	}
	if len(r.Cuerpo) != 1 || r.Cuerpo[0].Args[1].Var {
		t.Errorf(`"MEDIA" entre comillas es constante, y salio %v`, r.Cuerpo[0].Args[1])
	}
}

// LA TRAMPA DE LA CONVENCION, que es la razon de que este parser tenga un
// guardia y no solo una gramatica.
//
// Quien escribe categoria(S, MEDIA) cree estar comparando con la constante
// MEDIA. No lo esta: MEDIA empieza por mayuscula, o sea que es una variable
// nueva que unifica con CUALQUIER categoria. La regla no falla, no avisa y
// deriva de mas. Es el mismo fallo que la variable anonima de la revision
// anterior, escrito una vez y cazado para siempre.
func TestLaConstanteSinComillasSeRechazaPorqueSeriaUnaVariable(t *testing.T) {
	_, err := ParsearRegla(`aplica(x.auditoria, S) :- categoria(S, MEDIA)`)
	if err == nil {
		t.Fatal("HALLAZGO: categoria(S, MEDIA) sin comillas se acepta. MEDIA es una variable " +
			"que unifica con cualquier cosa, asi que la regla aplica la obligacion a todo " +
			"el que tenga alguna categoria, y nadie se entera")
	}
	if !errors.Is(err, ErrVariableUnica) {
		t.Fatalf("tiene que ser ErrVariableUnica y fue: %v", err)
	}
	if !strings.Contains(err.Error(), `"MEDIA"`) {
		t.Errorf("el error tiene que decir como se arregla (entrecomillar), y dijo: %v", err)
	}
	// Control positivo: con comillas entra sin rechistar.
	if _, err := ParsearRegla(`aplica(x.auditoria, S) :- categoria(S, "MEDIA")`); err != nil {
		t.Fatalf("con comillas tiene que valer, y dio: %v", err)
	}
}

// La otra mitad del guardia: cuando de verdad no importa el valor, se escribe _
// y entonces si vale. Sin esto, el guardia seria inutilizable y alguien lo
// quitaria.
func TestLaVariableAnonimaSalvaElCasoLegitimo(t *testing.T) {
	if _, err := ParsearRegla(`nivel_max(S, Cual) :- maneja(S, I), nivel_dimension(I, D, Cual)`); err == nil {
		t.Fatal("D aparece una sola vez, asi que tiene que rechazarse: o es un descuido o " +
			"era una constante sin comillas")
	}
	r, err := ParsearRegla(`nivel_max(S, N) :- maneja(S, I), nivel_dimension(I, _, N)`)
	if err != nil {
		t.Fatalf("con _ en la dimension y N usada dos veces tiene que valer: %v", err)
	}
	if len(r.Cuerpo) != 2 {
		t.Fatalf("dos atomos en el cuerpo, y hay %d", len(r.Cuerpo))
	}
	if !r.Cuerpo[1].Args[1].esAnonima() {
		t.Error("el segundo argumento de nivel_dimension tiene que ser la variable anonima")
	}
}

// _AGG es la variable con la que el MOTOR recibe el resultado de un agregado.
// Es un detalle interno y no se escribe en el dialecto: quien declara la regla
// pone en la cabeza la variable que agrega y la nombra en el campo "sobre" del
// fichero de datos. Si _AGG se colara aqui habria dos formas de decir lo mismo
// y una de ellas no diria nada al leerla.
func TestLaVariableInternaDelMotorNoSeEscribeEnElDialecto(t *testing.T) {
	_, err := ParsearRegla(`nivel_max(S, ` + VarAgregada + `) :- maneja(S, I), nivel_dimension(I, _, N)`)
	if err == nil {
		t.Fatalf("%s es interna del motor y no puede escribirse en una regla", VarAgregada)
	}
	if !errors.Is(err, ErrSintaxis) {
		t.Fatalf("tiene que ser ErrSintaxis y fue: %v", err)
	}
	if !strings.Contains(err.Error(), "sobre") {
		t.Errorf("el error tiene que decir como se declara de verdad (campo sobre), y dijo: %v", err)
	}
}

func TestUnPaqueteNoPuedeDeclararHechos(t *testing.T) {
	_, err := ParsearRegla(`responsable(acme)`)
	if !errors.Is(err, ErrSinCuerpo) {
		t.Fatalf("un hecho sin cuerpo tiene que rechazarse con ErrSinCuerpo, y dio: %v", err)
	}
	if !strings.Contains(err.Error(), "sujeto") {
		t.Errorf("el error tiene que explicar POR QUE (los hechos los aporta el sujeto), y dijo: %v", err)
	}
}

func TestLaNegacionSeParseaYNoSeConfundeConUnPredicado(t *testing.T) {
	r, err := ParsearRegla(`aplica(x.a, E) :- responsable(E), not exento(E)`)
	if err != nil {
		t.Fatalf("%v", err)
	}
	if len(r.Cuerpo) != 1 || len(r.Negados) != 1 {
		t.Fatalf("un positivo y un negado, y salieron %d y %d", len(r.Cuerpo), len(r.Negados))
	}
	// notificar empieza por "not" y NO es una negacion. Si el parser corta por
	// prefijo a secas, aqui deriva una negacion de "ificar", que no existe.
	r2, err := ParsearRegla(`aplica(x.b, E) :- responsable(E), notificar(E, X), obliga(X)`)
	if err != nil {
		t.Fatalf("notificar no es una negacion: %v", err)
	}
	if len(r2.Negados) != 0 {
		t.Fatalf("notificar se ha leido como negacion: %v", r2.Negados)
	}
}

func TestLaComaDentroDeComillasNoParteElAtomo(t *testing.T) {
	r, err := ParsearRegla(`aplica(x.a, S) :- etiqueta(S, "uno, dos"), usa(S)`)
	if err != nil {
		t.Fatalf("%v", err)
	}
	if len(r.Cuerpo) != 2 {
		t.Fatalf("dos atomos en el cuerpo, y salieron %d: %v", len(r.Cuerpo), r.Cuerpo)
	}
	if got := r.Cuerpo[0].Args[1].Val; got != "uno, dos" {
		t.Fatalf("la constante entrecomillada se partio por la coma: %q", got)
	}
}

func TestSeRechazaLoQueTieneQueRechazarse(t *testing.T) {
	casos := []struct {
		regla    string
		quiero   error
		porQue   string
		esperado string
	}{
		{``, ErrReglaVacia, "una regla vacia", ""},
		{`   `, ErrReglaVacia, "solo espacios", ""},
		{`aplica(x.a, S) :- `, ErrSintaxis, "cuerpo vacio", ""},
		{`aplica(x.a, S) :- categoria(S, "M"),`, ErrSintaxis, "coma de mas", ""},
		{`aplica(x.a, S) :- not exento(S)`, ErrSinCuerpo, "solo negacion", ""},
		{`Aplica(x.a, S) :- usa(S, x)`, ErrPredicadoInvado, "predicado en mayuscula", ""},
		{`aplica(x.a, S) :- usa(S`, ErrSintaxis, "parentesis sin cerrar", ""},
		{`aplica() :- usa(S, x)`, ErrSintaxis, "predicado sin argumentos", ""},
		{`aplica(x.a, S) :- usa(S, "abre)`, ErrSintaxis, "comilla sin cerrar", ""},
		{`aplica(x.a, S) :- usa(S, tiene espacio)`, ErrSintaxis, "constante con espacio sin comillas", ""},
		{strings.Repeat("a", maxLongitudRegla+1), ErrDemasiadoLarga, "pasa del tope de longitud", ""},
	}
	for _, c := range casos {
		_, err := ParsearRegla(c.regla)
		if err == nil {
			t.Errorf("%s: %q se acepto y no debia", c.porQue, c.regla)
			continue
		}
		if !errors.Is(err, c.quiero) {
			t.Errorf("%s: %q dio %v, se esperaba %v", c.porQue, c.regla, err, c.quiero)
		}
	}
}

func TestParsearYEscribirDanLaVueltaEntera(t *testing.T) {
	reglas := []string{
		`aplica(x.auditoria, S) :- categoria(S, "MEDIA")`,
		`aplica(x.a, E) :- responsable(E), not exento(E)`,
		`nivel_max(S, N) :- maneja(S, I), nivel_dimension(I, _, N)`,
		`proveedor_de(A, B) :- contrata(A, C), proveedor_de(C, B)`,
	}
	for _, txt := range reglas {
		r, err := ParsearRegla(txt)
		if err != nil {
			t.Fatalf("%q: %v", txt, err)
		}
		vuelta := r.Escribir()
		if vuelta != txt {
			t.Errorf("no da la vuelta:\n  original: %s\n  escrito:  %s", txt, vuelta)
		}
		if _, err := ParsearRegla(vuelta); err != nil {
			t.Errorf("lo que escribe el propio parser no se puede volver a parsear: %v", err)
		}
	}
}

// Fuzzing del parser, que es puerta del proyecto: el corpus lo escribe un
// tercero, asi que este es codigo que lee entrada hostil.
//
// Dos propiedades: no revienta nunca, y lo que acepta lo puede volver a leer.
// La segunda importa mas de lo que parece: si el parser acepta algo que su
// propio escritor no sabe reproducir, hay un valor que entra al motor y no se
// puede ensenar en un `plazum explain`.
func FuzzParsearRegla(f *testing.F) {
	f.Add(`aplica(x.a, S) :- categoria(S, "MEDIA")`)
	f.Add(`aplica(x.a, E) :- responsable(E), not exento(E)`)
	f.Add(`nivel_max(S, N) :- maneja(S, I), nivel_dimension(I, _, N)`)
	f.Add(`:-`)
	f.Add(`a(`)
	f.Add(`a("`)
	f.Add(`a(b) :- c(d) :- e(f)`)
	f.Add(`a(_, _) :- b(_, _)`)
	f.Add("a(b) :- c(\x00)")

	f.Fuzz(func(t *testing.T, s string) {
		r, err := ParsearRegla(s)
		if err != nil {
			return
		}
		vuelta := r.Escribir()
		r2, err := ParsearRegla(vuelta)
		if err != nil {
			t.Fatalf("el parser acepto %q, lo escribio como %q y ya no lo puede leer: %v",
				s, vuelta, err)
		}
		if r2.Escribir() != vuelta {
			t.Fatalf("la segunda vuelta no es estable:\n  entrada: %q\n  1: %q\n  2: %q",
				s, vuelta, r2.Escribir())
		}
	})
}
