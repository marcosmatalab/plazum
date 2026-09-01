package auditoria

import (
	"errors"
	"strings"
	"testing"
	"time"
)

var (
	inicio = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	medio  = time.Date(2027, 6, 1, 0, 0, 0, 0, time.UTC)
	fin    = time.Date(2028, 12, 31, 0, 0, 0, 0, time.UTC)
)

func ciclo(nombre string) Ciclo { return Ciclo{Nombre: nombre, Desde: inicio, Hasta: fin} }

// alcanceDePrueba NO nombra ninguna norma (invariante 2): los identificadores
// son sinteticos, como lo serian los de cualquier paquete que el cliente
// instale. Este paquete no puede saber el nombre de una norma y sus tests
// tampoco deben aprenderlo.
func alcanceDePrueba() []Unidad {
	return []Unidad{
		{Paquete: "marco-a", Version: "1.0.0", Obligacion: "c1", Titulo: "la primera"},
		{Paquete: "marco-a", Version: "1.0.0", Obligacion: "c2", Titulo: "la segunda"},
		{Paquete: "marco-b", Version: "2.1.0", Obligacion: "x9", Titulo: "la de otro marco"},
	}
}

func programa(t *testing.T, arr Arrastre) *Programa {
	t.Helper()
	p, err := Abrir("prog-1", ciclo("2026-2028"), alcanceDePrueba(), arr)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func sesion(unidades ...string) Sesion {
	return Sesion{ID: "s1", Auditor: "auditor-interno", Cuando: medio, Unidades: unidades,
		Alcance: "revision documental y entrevista"}
}

// LA LEY DE CONSERVACION: toda unidad del alcance en exactamente un cubo, y los
// vacios tambien salen.
func TestTodaUnidadDelAlcanceCaeEnExactamenteUnCubo(t *testing.T) {
	p := programa(t, Arrastre{})
	if err := p.Auditar(sesion("marco-a|c1")); err != nil {
		t.Fatal(err)
	}
	if err := p.Diferir(Diferimiento{Unidad: "marco-b|x9", Quien: "ciso",
		Motivo: "el proveedor no da acceso hasta el trimestre que viene", Cuando: medio}); err != nil {
		t.Fatal(err)
	}
	c := p.Cuenta()
	for _, cob := range CoberturasPosibles() {
		if _, hay := c[cob]; !hay {
			t.Errorf("el cubo %q no aparece: uno que solo sale cuando tiene algo dentro es un "+
				"cubo que nadie echa de menos", cob)
		}
	}
	if c[Auditada] != 1 || c[Diferida] != 1 || c[SinAuditar] != 1 {
		t.Fatalf("cubos: %v", c)
	}
	if !p.Cuadra() {
		t.Fatalf("los cubos no suman el alcance (%d): %v", p.Alcance(), c)
	}
}

// EL ARRASTRE SE CALCULA, NO SE APUNTA, y la edad se ACUMULA.
//
// Es la pregunta del auditor externo y la que hoy no contesta nadie: no "que se
// audito" sino "que lleva tres ciclos sin auditarse".
func TestLaEdadDeLoNoAuditadoSeAcumulaEntreCiclos(t *testing.T) {
	// Ciclo 1: solo se audita una.
	p1 := programa(t, Arrastre{})
	if err := p1.Auditar(sesion("marco-a|c1")); err != nil {
		t.Fatal(err)
	}
	arr := p1.ParaElCicloSiguiente()
	if arr.SinAuditar["marco-a|c2"] != 1 || arr.SinAuditar["marco-b|x9"] != 1 {
		t.Fatalf("primer arrastre: %v", arr.SinAuditar)
	}
	if _, sobra := arr.SinAuditar["marco-a|c1"]; sobra {
		t.Error("una unidad auditada se ha arrastrado")
	}

	// Ciclo 2: se audita otra, y la que sigue sin tocarse suma.
	p2, err := Abrir("prog-2", ciclo("2029-2031"), alcanceDePrueba(), arr)
	if err != nil {
		t.Fatal(err)
	}
	if err := p2.Auditar(sesion("marco-a|c2")); err != nil {
		t.Fatal(err)
	}
	arr2 := p2.ParaElCicloSiguiente()
	if arr2.SinAuditar["marco-b|x9"] != 2 {
		t.Fatalf("la edad no se acumula: %v", arr2.SinAuditar)
	}
	// Y la que se audito en el ciclo 2 sale del arrastre, aunque viniera con
	// edad: auditarla es lo que la saca.
	if _, sigue := arr2.SinAuditar["marco-a|c2"]; sigue {
		t.Error("una unidad que se acaba de auditar sigue arrastrandose")
	}
	// La que nunca se ha auditado sale la primera de la lista que se ensena.
	pend := p2.SinAuditarDesdeHace()
	if len(pend) == 0 || pend[0].Unidad.Clave() != "marco-b|x9" || pend[0].Ciclos != 2 {
		t.Fatalf("lo mas viejo no sale primero: %+v", pend)
	}
	if !strings.Contains(pend[0].Frase(), "2 ciclos seguidos") {
		t.Errorf("la frase no dice cuanto lleva: %q", pend[0].Frase())
	}
}

// LO DIFERIDO ARRASTRA IGUAL QUE LO NO AUDITADO.
//
// ES LA DECISION QUE MAS FACIL SERIA COLAR AL REVES, y por eso tiene su caso
// aislado: un diferimiento con su motivo se SIENTE resuelto y no lo esta. Si lo
// diferido no arrastrara, tres ciclos de diferimientos razonables darian una
// unidad que no se audita nunca y un programa que dice que va bien.
func TestUnDiferimientoRazonableSigueArrastrandoElCicloSiguiente(t *testing.T) {
	p := programa(t, Arrastre{})
	// Todo auditado MENOS una, que se difiere con un motivo perfectamente
	// legitimo. El caso aisla la rama: sin esto, el arrastre podria fallar por
	// las otras dos y esta afirmacion no probaria nada.
	if err := p.Auditar(sesion("marco-a|c1", "marco-a|c2")); err != nil {
		t.Fatal(err)
	}
	if err := p.Diferir(Diferimiento{Unidad: "marco-b|x9", Quien: "ciso",
		Motivo: "el sistema se retira en marzo y auditarlo no aporta nada", Cuando: medio}); err != nil {
		t.Fatal(err)
	}
	if c := p.Cuenta(); c[SinAuditar] != 0 || c[Diferida] != 1 {
		t.Fatalf("el caso no aisla el diferimiento: %v", c)
	}
	arr := p.ParaElCicloSiguiente()
	if arr.SinAuditar["marco-b|x9"] != 1 {
		t.Fatalf("lo diferido no arrastra: %v.\n"+
			"  Diferir explica por que falta; no hace que deje de faltar. Sin esto, tres ciclos "+
			"de diferimientos razonables dan una unidad que no se audita nunca", arr.SinAuditar)
	}
}

// UN HALLAZGO ABIERTO NO REJUVENECE AL CAMBIAR DE CICLO.
//
// Es la otra mitad del arrastre y la que se olvida: el hallazgo se anoto en el
// programa anterior, asi que no esta en los hallazgos de este. Si su edad no se
// sumara desde el arrastre, cambiar de ciclo pondria a cero la cuenta de uno de
// tres anos, que es exactamente como se pierde de vista.
func TestUnHallazgoAbiertoNoRejuveneceAlCambiarDeCiclo(t *testing.T) {
	p1 := programa(t, Arrastre{})
	if err := p1.Auditar(sesion("marco-a|c1", "marco-a|c2", "marco-b|x9")); err != nil {
		t.Fatal(err)
	}
	if err := p1.Anotar(Hallazgo{ID: "h1", Sesion: "s1", Unidad: "marco-a|c1",
		Clase: NoConformidadMenor, Texto: "el registro no se conserva", Quien: "auditor-interno",
		Cuando: medio}); err != nil {
		t.Fatal(err)
	}
	arr := p1.ParaElCicloSiguiente()
	if arr.Abiertos["h1"] != 1 {
		t.Fatalf("el hallazgo no arrastra: %v", arr.Abiertos)
	}

	// Ciclo 2: el hallazgo NO se cierra, y no esta en los hallazgos de este
	// programa porque se anoto en el anterior.
	p2, err := Abrir("prog-2", ciclo("2029-2031"), alcanceDePrueba(), arr)
	if err != nil {
		t.Fatal(err)
	}
	if len(p2.Abiertos()) != 0 {
		t.Fatal("el caso no aisla nada: el hallazgo del ciclo anterior no puede estar en este")
	}
	arr2 := p2.ParaElCicloSiguiente()
	if arr2.Abiertos["h1"] != 2 {
		t.Fatalf("el hallazgo ha rejuvenecido al cambiar de ciclo: %v.\n"+
			"  Es como se pierde de vista uno de tres anos", arr2.Abiertos)
	}
}

// UNA UNIDAD QUE SALE DEL ALCANCE NO SE ARRASTRA, PERO SE DICE.
//
// Las dos mitades importan: arrastrarla daria un programa que echa de menos para
// siempre algo que la organizacion dejo de tener; callarlo dejaria que una
// obligacion desapareciera del programa sin que conste que desaparecio.
func TestUnaUnidadQueSaleDelAlcanceNoSeArrastraPeroSeDice(t *testing.T) {
	viejo := Arrastre{SinAuditar: map[string]int{"marco-a|c2": 2, "marco-z|ya-no": 3},
		DeCiclo: "2023-2025"}
	p := programa(t, viejo)
	arr := p.DelCicloAnterior()
	if _, sigue := arr.SinAuditar["marco-z|ya-no"]; sigue {
		t.Error("una unidad que ya no esta en el alcance se sigue arrastrando")
	}
	if arr.SinAuditar["marco-a|c2"] != 2 {
		t.Errorf("una unidad que sigue en el alcance ha perdido su edad: %v", arr.SinAuditar)
	}
	if len(arr.Salidas) != 1 || arr.Salidas[0] != "marco-z|ya-no" {
		t.Fatalf("la salida no se dice: %+v", arr.Salidas)
	}
}

// LOS HECHOS SE CONTRASTAN CONTRA EL ALCANCE (invariante 7), y esto se escribe
// con la leccion de la excusa de la UAR ya aprendida: el mismo agujero costo un
// P1 el mismo dia, y un arreglo que solo tapa el caso encontrado deja el
// siguiente hallazgo servido.
func TestNingunHechoApuntaFueraDelAlcance(t *testing.T) {
	casos := []struct {
		nombre string
		hacer  func(*Programa) error
		dice   string
	}{
		{"auditar algo que no esta", func(p *Programa) error {
			return p.Auditar(sesion("marco-a|c1", "marco-z|inventada"))
		}, "sube el recuento de cobertura sin cubrir"},
		{"diferir algo que no esta", func(p *Programa) error {
			return p.Diferir(Diferimiento{Unidad: "marco-z|inventada", Quien: "ciso",
				Motivo: "x", Cuando: medio})
		}, "no difiere nada"},
		{"anotar un hallazgo sobre algo que no esta", func(p *Programa) error {
			return p.Anotar(Hallazgo{ID: "h9", Unidad: "marco-z|inventada", Clase: Observacion,
				Texto: "x", Quien: "a", Cuando: medio})
		}, "no se puede arrastrar"},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			p := programa(t, Arrastre{})
			err := c.hacer(p)
			if err == nil {
				t.Fatal("admitido")
			}
			if !errors.Is(err, ErrFueraDelAlcance) {
				t.Fatalf("centinela: %v", err)
			}
			if !strings.Contains(err.Error(), c.dice) {
				t.Errorf("el error no dice %q:\n%v", c.dice, err)
			}
			if p.Cuenta()[Auditada] != 0 || len(p.Diferimientos()) != 0 || len(p.Hallazgos()) != 0 {
				t.Fatal("el hecho rechazado se ha quedado dentro")
			}
		})
	}
}

// UNA SESION DE FUERA DEL CICLO NO CUBRE ESTE CICLO.
//
// Todo el arrastre se apoya en que "auditada" signifique "auditada EN ESTE
// CICLO". Admitir una de hace cuatro anos presenta cobertura de trabajo que no
// se hizo en el periodo que se esta certificando.
func TestUnaSesionDeFueraDelCicloNoCuentaComoCobertura(t *testing.T) {
	p := programa(t, Arrastre{})
	vieja := sesion("marco-a|c1")
	vieja.Cuando = inicio.AddDate(-2, 0, 0)
	err := p.Auditar(vieja)
	if !errors.Is(err, ErrFueraDelAlcance) {
		t.Fatalf("centinela: %v", err)
	}
	if !strings.Contains(err.Error(), "no se hizo en el periodo") {
		t.Errorf("el error no dice por que importa:\n%v", err)
	}
	if p.Cuenta()[Auditada] != 0 {
		t.Fatal("ha contado como cobertura")
	}
	// Los dos extremos del ciclo SI entran: una auditoria del ultimo dia es del
	// ciclo. Sin este control positivo, la comprobacion se cumpliria
	// rechazandolo todo.
	for _, cuando := range []time.Time{inicio, fin} {
		p := programa(t, Arrastre{})
		s := sesion("marco-a|c1")
		s.Cuando = cuando
		if err := p.Auditar(s); err != nil {
			t.Errorf("una sesion del %s se rechaza: %v", cuando.Format("2006-01-02"), err)
		}
	}
}

// EL DIFERIMIENTO, CON LAS TRES GUARDAS QUE LA EXCUSA NO TENIA.
func TestUnDiferimientoNoSeRepiteNiContradiceLoAuditado(t *testing.T) {
	p := programa(t, Arrastre{})
	d := Diferimiento{Unidad: "marco-b|x9", Quien: "ciso", Motivo: "el proveedor no da acceso",
		Cuando: medio}
	if err := p.Diferir(d); err != nil {
		t.Fatal(err)
	}
	if err := p.Diferir(d); !errors.Is(err, ErrSinEfecto) {
		t.Fatalf("se difiere dos veces la misma unidad: %v", err)
	}
	// Y no se difiere lo ya auditado: el informe diria a la vez que se miro y
	// que se dejo de mirar.
	if err := p.Auditar(sesion("marco-a|c1")); err != nil {
		t.Fatal(err)
	}
	err := p.Diferir(Diferimiento{Unidad: "marco-a|c1", Quien: "ciso", Motivo: "x", Cuando: medio})
	if !errors.Is(err, ErrSinEfecto) {
		t.Fatalf("se ha diferido algo ya auditado: %v", err)
	}
	// Sin quien, sin motivo o sin cuando, tampoco.
	for _, malo := range []Diferimiento{
		{Unidad: "marco-a|c2", Motivo: "x", Cuando: medio},
		{Unidad: "marco-a|c2", Quien: "ciso", Cuando: medio},
		{Unidad: "marco-a|c2", Quien: "ciso", Motivo: "x"},
	} {
		if err := p.Diferir(malo); !errors.Is(err, ErrHechoIncompleto) {
			t.Errorf("admitido %+v: %v", malo, err)
		}
	}
	// Control positivo: el bueno sigue pasando.
	if err := p.Diferir(Diferimiento{Unidad: "marco-a|c2", Quien: "ciso", Motivo: "y",
		Cuando: medio}); err != nil {
		t.Fatalf("el diferimiento legitimo se rechaza: %v", err)
	}
}

// AUDITADA GANA A DIFERIDA. Si se difirio en enero y se acabo auditando en
// octubre, esta auditada: lo contrario deja una unidad mirada contada como
// pendiente.
func TestSiSeDifirioYLuegoSeAuditoCuentaComoAuditada(t *testing.T) {
	p := programa(t, Arrastre{})
	if err := p.Diferir(Diferimiento{Unidad: "marco-b|x9", Quien: "ciso", Motivo: "en enero no se podia",
		Cuando: inicio}); err != nil {
		t.Fatal(err)
	}
	s := sesion("marco-b|x9")
	s.ID = "s2"
	if err := p.Auditar(s); err != nil {
		t.Fatal(err)
	}
	if got := p.CoberturaDe("marco-b|x9"); got != Auditada {
		t.Fatalf("cobertura: %q", got)
	}
	if p.ParaElCicloSiguiente().SinAuditar["marco-b|x9"] != 0 {
		t.Error("una unidad auditada arrastra")
	}
}

// UN CIERRE SIN EL "COMO" NO ES UN CIERRE.
//
// "Cerrado" a secas no es evidencia de nada, y es lo primero que pide un auditor
// externo.
func TestUnCierreSinDecirComoNoSeAdmite(t *testing.T) {
	p := programa(t, Arrastre{})
	if err := p.Auditar(sesion("marco-a|c1")); err != nil {
		t.Fatal(err)
	}
	h := Hallazgo{ID: "h1", Sesion: "s1", Unidad: "marco-a|c1", Clase: NoConformidadMayor,
		Texto: "no hay registro", Quien: "auditor-interno", Cuando: medio}
	if err := p.Anotar(h); err != nil {
		t.Fatal(err)
	}
	if err := p.Cerrar(CierreDeHallazgo{Hallazgo: "h1", Quien: "ciso", Cuando: fin}); !errors.Is(
		err, ErrHechoIncompleto) {
		t.Fatalf("cierre sin como: %v", err)
	}
	if len(p.Abiertos()) != 1 {
		t.Fatal("el cierre rechazado ha cerrado igual")
	}
	// Y uno que no consta no se cierra.
	if err := p.Cerrar(CierreDeHallazgo{Hallazgo: "h404", Quien: "ciso", Como: "x",
		Cuando: fin}); !errors.Is(err, ErrHallazgoDesconocido) {
		t.Fatalf("centinela: %v", err)
	}
	// Control positivo.
	if err := p.Cerrar(CierreDeHallazgo{Hallazgo: "h1", Quien: "ciso",
		Como: "se implanto el registro y se conserva desde marzo", Cuando: fin}); err != nil {
		t.Fatalf("el cierre legitimo se rechaza: %v", err)
	}
	if len(p.Abiertos()) != 0 {
		t.Fatal("no se ha cerrado")
	}
	if err := p.Cerrar(CierreDeHallazgo{Hallazgo: "h1", Quien: "otra", Como: "y",
		Cuando: fin}); !errors.Is(err, ErrSinEfecto) {
		t.Fatalf("se cierra dos veces: %v", err)
	}
}

// LA CLASE QUE NO SE RECONOCE NO CAE A NINGUN DEFECTO.
//
// Invariante 8 en una frontera de lectura, y aqui el error va en las DOS
// direcciones: caer al cero acusa de mas (es "no conformidad mayor") y caer a la
// ultima esconde un incumplimiento (es "oportunidad de mejora").
func TestUnaClaseQueNoSeReconoceNoCaeANingunDefecto(t *testing.T) {
	for nombre, quiero := range map[string]Clase{
		"no conformidad mayor":  NoConformidadMayor,
		"no conformidad menor":  NoConformidadMenor,
		"observacion":           Observacion,
		"oportunidad de mejora": Oportunidad,
	} {
		got, err := ClaseDe(nombre)
		if err != nil || got != quiero {
			t.Errorf("%q -> %v, %v", nombre, got, err)
		}
	}
	for _, malo := range []string{"", "mayor", "NO CONFORMIDAD MAYOR", "critica", "nc"} {
		if _, err := ClaseDe(malo); err == nil {
			t.Errorf("%q se ha aceptado", malo)
		}
	}
	// Y solo las no conformidades exigen plan de accion: confundirlas con las
	// observaciones es como un programa acaba con cuarenta abiertos que nadie
	// mira.
	if !NoConformidadMayor.Exige() || !NoConformidadMenor.Exige() {
		t.Error("una no conformidad no exige plan de accion")
	}
	if Observacion.Exige() || Oportunidad.Exige() {
		t.Error("una observacion exige plan de accion")
	}
}

// LA INDEPENDENCIA SE DICE, NO SE DECIDE.
//
// La norma exige objetividad e imparcialidad, que es un juicio que plazum no
// puede hacer. Lo mecanico si: si quien audito una unidad es la persona
// responsable de ella, eso es exactamente lo que la exigencia mira, y hoy no lo
// comprueba nadie hasta que llega el auditor externo.
func TestElAuditorQueSeAuditaASiMismoSeDiceYNoSeRechaza(t *testing.T) {
	p := programa(t, Arrastre{})
	s := Sesion{ID: "s1", Auditor: "ana", Cuando: medio,
		Unidades: []string{"marco-a|c1", "marco-a|c2"}}
	if err := p.Auditar(s); err != nil {
		t.Fatal("la sesion se ha rechazado, y esto se dice, no se rechaza: " + err.Error())
	}
	responsables := map[string]string{"marco-a|c1": "ana", "marco-a|c2": "luis"}
	cs := p.Independencia(responsables)
	if len(cs) != 1 || cs[0].Unidad != "marco-a|c1" || cs[0].Persona != "ana" {
		t.Fatalf("conflictos: %+v", cs)
	}
	if !strings.Contains(cs[0].Frase(), "No lo decide plazum") {
		t.Errorf("la frase decide por su cuenta: %q", cs[0].Frase())
	}
	// CONTROL NEGATIVO: sin responsable asignado no se inventa un conflicto, y
	// con otro responsable tampoco. Sin esto, la comprobacion se cumpliria
	// senalando siempre.
	if n := len(p.Independencia(map[string]string{})); n != 0 {
		t.Errorf("sin responsables asignados se han inventado %d conflictos", n)
	}
	if n := len(p.Independencia(map[string]string{"marco-a|c1": "otra", "marco-a|c2": "luis"})); n != 0 {
		t.Errorf("con responsables distintos del auditor se han visto %d conflictos", n)
	}
}

// EL VALOR CERO DEL PROGRAMA ESTA PROHIBIDO, en sus dos formas de la nada.
func TestAbrirRechazaLoQueFalta(t *testing.T) {
	casos := []struct {
		nombre  string
		id      string
		c       Ciclo
		alcance []Unidad
		falta   string
	}{
		{"sin id", "  ", ciclo("x"), alcanceDePrueba(), "id del programa"},
		{"ciclo cero", "p", Ciclo{}, alcanceDePrueba(), "ciclo con nombre"},
		{"ciclo al reves", "p", Ciclo{Nombre: "x", Desde: fin, Hasta: inicio}, alcanceDePrueba(),
			"hasta despues de desde"},
		{"alcance nil", "p", ciclo("x"), nil, "al menos una unidad"},
		{"alcance vacio pero presente", "p", ciclo("x"), []Unidad{}, "al menos una unidad"},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			_, err := Abrir(c.id, c.c, c.alcance, Arrastre{})
			if !errors.Is(err, ErrPrograma) {
				t.Fatalf("centinela: %v", err)
			}
			if !strings.Contains(err.Error(), c.falta) {
				t.Errorf("el error no dice %q:\n%v", c.falta, err)
			}
		})
	}
	// Y el arrastre cero es LEGITIMO: un primer ciclo no arrastra nada, y eso
	// es distinto de no haberlo mirado.
	p, err := Abrir("p", ciclo("x"), alcanceDePrueba(), Arrastre{})
	if err != nil {
		t.Fatalf("un primer ciclo sin arrastre no se puede abrir: %v", err)
	}
	if len(p.DelCicloAnterior().SinAuditar) != 0 {
		t.Error("el arrastre cero ha inventado unidades")
	}
}

// UNA UNIDAD REPETIDA EN EL ALCANCE NO CUENTA DOS VECES.
//
// El alcance es un conjunto: si el mismo par (paquete, obligacion) entra dos
// veces, la cobertura saldria sobre un denominador inflado y "hemos auditado 40
// de 45" seria falso por el lado que mas conviene.
func TestElAlcanceEsUnConjuntoYNoInflaElDenominador(t *testing.T) {
	con := append(alcanceDePrueba(), Unidad{Paquete: "marco-a", Version: "1.0.0", Obligacion: "c1",
		Titulo: "la primera, otra vez"})
	p, err := Abrir("p", ciclo("x"), con, Arrastre{})
	if err != nil {
		t.Fatal(err)
	}
	if p.Alcance() != 3 {
		t.Fatalf("alcance %d, y las unidades distintas son 3", p.Alcance())
	}
}

// LA BARRA DENTRO DE UNA IDENTIDAD, que salio atacando el acta y no leyendo esto.
//
// La clave de una unidad es paquete + "|" + obligacion. Sin guarda, ("p1",
// "o1|x") y ("p1|o1", "x") son DOS UNIDADES DISTINTAS con la MISMA clave, y como
// el alcance es un mapa por clave, la segunda se cae por la rama que existe para
// no auditar dos veces la misma. El sintoma es el peor de esta familia: una
// obligacion desaparece del programa en silencio y TODO SIGUE CUADRANDO, porque
// cuadra sobre lo que quedo.
//
// Medido antes de poner la guarda: Alcance() devolvia 1 con dos unidades dentro.
// Hoy no llega a abrirse.
func TestUnaUnidadConLaBarraDentroNoEntraEnElAlcance(t *testing.T) {
	c := ciclo("2026-2028")
	_, err := Abrir("prog-x", c, []Unidad{
		{Paquete: "p1", Obligacion: "o1|x", Titulo: "una"},
		{Paquete: "p1|o1", Obligacion: "x", Titulo: "otra"},
	}, Arrastre{})
	if err == nil {
		t.Fatal("dos unidades distintas con la misma clave han entrado al alcance, y una de las " +
			"dos ha desaparecido sin que nada se pusiera rojo")
	}
	if !errors.Is(err, ErrPrograma) {
		t.Fatalf("centinela: %v", err)
	}
	if !strings.Contains(err.Error(), "desaparece") {
		t.Errorf("el error no dice que es lo que se pierde: %v", err)
	}
	// La otra direccion: sin barras, las dos entran y son dos.
	p, err := Abrir("prog-y", c, []Unidad{
		{Paquete: "p1", Obligacion: "o1x", Titulo: "una"},
		{Paquete: "p1o1", Obligacion: "x", Titulo: "otra"},
	}, Arrastre{})
	if err != nil {
		t.Fatalf("sin barras tenia que abrir: %v", err)
	}
	if p.Alcance() != 2 {
		t.Errorf("el alcance tenia que tener 2 unidades y tiene %d", p.Alcance())
	}
}
