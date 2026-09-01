package accesos

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/marcosmatalab/plazum/nucleo/censo"
	"github.com/marcosmatalab/plazum/nucleo/ledger"
)

// anotar mete en un ledger las entradas de una campana, encadenadas.
func anotar(t *testing.T, l *ledger.Ledger, es ...ledger.Entrada) {
	t.Helper()
	for _, e := range es {
		if _, err := l.Anadir(e); err != nil {
			t.Fatal(err)
		}
	}
}

func aperturaDe(t *testing.T, ins censo.Instantanea, id string) ledger.Entrada {
	t.Helper()
	e, err := AperturaComoEntrada(ins, id)
	if err != nil {
		t.Fatal(err)
	}
	return e
}

// LA VUELTA COMPLETA: se decide, se guarda, se vuelve mananana con el fichero y
// el ledger, y la campana esta donde se dejo.
//
// Sin esto el motor es una biblioteca. Una campana de accesos vive semanas y la
// revisan varias personas: si no sobrevive a cerrar el portatil, no existe.
func TestUnaCampanaSeReconstruyeDesdeElFicheroYElLedger(t *testing.T) {
	ins := instantanea(t, censoBase)
	var l ledger.Ledger
	anotar(t, &l, aperturaDe(t, ins, "uar-h2"))

	// Se decide sobre dos accesos y se excusa nada.
	d1 := Decision{Fila: "erp|u1|admin", Veredicto: Revocar, Quien: "ciso", Cuando: t1,
		Motivo: "cambio de puesto en junio"}
	d2 := Decision{Fila: "erp|u2|lector", Veredicto: Aprobar, Quien: "jefa", Cuando: t1}
	for _, d := range []Decision{d1, d2} {
		e, err := DecisionComoEntrada(d, ins.Sello(), "uar-h2")
		if err != nil {
			t.Fatal(err)
		}
		anotar(t, &l, e)
	}

	c, err := Reconstruir("uar-h2", ins, l, nil)
	if err != nil {
		t.Fatalf("no se reconstruye: %v", err)
	}
	if e := c.EstadoDe("erp|u1|admin"); e != Revocada {
		t.Errorf("la revocacion no ha vuelto: %q", e)
	}
	if e := c.EstadoDe("erp|u2|lector"); e != Aprobada {
		t.Errorf("la aprobacion no ha vuelto: %q", e)
	}
	if e := c.EstadoDe("erp|u1|lector"); e != SinRevisar {
		t.Errorf("un acceso sin decidir ha vuelto decidido: %q", e)
	}
	// Y VUELVEN LOS HECHOS, no solo el resultado: quien decidio, cuando y por
	// que. Es lo que el acta 9.3 consume.
	hs := c.Decisiones()
	if len(hs) != 2 {
		t.Fatalf("hechos reconstruidos: %d", len(hs))
	}
	var rev Decision
	for _, h := range hs {
		if h.Fila == "erp|u1|admin" {
			rev = h
		}
	}
	if rev.Quien != "ciso" || !rev.Cuando.Equal(t1) || rev.Motivo != "cambio de puesto en junio" {
		t.Errorf("el hecho ha vuelto sin su autor, su instante o su motivo: %+v", rev)
	}
}

// EL LEDGER NO LLEVA LA CLAVE DE LA FILA, LLEVA SU HUELLA.
//
// Es la unica diferencia entre un ledger que se puede ensenar a un auditor y uno
// que hay que custodiar como una base de datos de personal.
func TestLoQueSeAnotaDeUnaDecisionNoLlevaElIdentificadorDeNadie(t *testing.T) {
	ins := instantanea(t, censoBase)
	e, err := DecisionComoEntrada(Decision{Fila: "erp|u1|admin", Veredicto: Revocar,
		Quien: "ciso", Cuando: t1, Motivo: "se fue"}, ins.Sello(), "uar-h2")
	if err != nil {
		t.Fatal(err)
	}
	crudo := string(e.Carga)
	for _, prohibido := range []string{"u1", "admin", "Ana"} {
		if strings.Contains(crudo, prohibido) {
			t.Errorf("%q viaja en la carga de la decision:\n%s", prohibido, crudo)
		}
	}
	// Y la huella SI casa con la fila, o la decision no serviria de nada.
	var cd CargaDeDecision
	if err := json.Unmarshal(e.Carga, &cd); err != nil {
		t.Fatal(err)
	}
	if cd.Huella != HuellaDeFila(ins.Sello(), "erp|u1|admin") {
		t.Error("la huella no casa con la fila")
	}
}

// LA HUELLA VA SALADA CON EL SELLO, asi que la misma cuenta en dos campanas da
// dos huellas distintas: quien tenga dos ledgers no puede cruzarlos para seguir
// a una persona entre revisiones.
func TestLaHuellaDeUnaFilaNoSeRepiteEntreCampanas(t *testing.T) {
	a := instantanea(t, censoBase)
	b := instantanea(t, censoBase+"u3;Eva Roca;admin\n")
	if a.Sello() == b.Sello() {
		t.Fatal("el caso no vale: los dos censos tienen el mismo sello")
	}
	clave := "erp|u1|admin"
	if HuellaDeFila(a.Sello(), clave) == HuellaDeFila(b.Sello(), clave) {
		t.Fatal("la misma cuenta da la misma huella en dos campanas: dos ledgers se pueden " +
			"cruzar para seguir a una persona entre revisiones")
	}
	// Y dentro de la misma campana es estable, o no se podria reconstruir nada.
	// Los dos lados son la misma expresion A PROPOSITO: lo que se comprueba es
	// que HuellaDeFila sea pura. Si algun dia metiera una sal aleatoria o leyera
	// el reloj, ninguna campana se volveria a reabrir y este es el sitio donde se
	// ve con una sola entrada.
	//lint:ignore SA4000 la igualdad consigo misma ES el caso: ver arriba
	if HuellaDeFila(a.Sello(), clave) != HuellaDeFila(a.Sello(), clave) {
		t.Fatal("la huella no es estable")
	}
}

// SI EL FICHERO CAMBIO, LA CAMPANA NO SE REABRE.
//
// El caso caro, dicho con nombres: se revoco la fila 12 de aquel fichero. Si se
// reabriera sobre un export nuevo, esa revocacion acabaria firmada sobre quien
// este hoy en la 12. Se para antes de reproducir un solo hecho.
func TestSiElFicheroCambioLaCampanaNoSeReabre(t *testing.T) {
	viejo := instantanea(t, censoBase)
	var l ledger.Ledger
	anotar(t, &l, aperturaDe(t, viejo, "uar-h2"))
	e, err := DecisionComoEntrada(Decision{Fila: "erp|u1|admin", Veredicto: Revocar,
		Quien: "ciso", Cuando: t1, Motivo: "se fue"}, viejo.Sello(), "uar-h2")
	if err != nil {
		t.Fatal(err)
	}
	anotar(t, &l, e)

	nuevo := instantanea(t, censoBase+"u9;Nueva Persona;admin\n")
	_, err = Reconstruir("uar-h2", nuevo, l, nil)
	if err == nil {
		t.Fatal("se ha reabierto la campana sobre otro fichero")
	}
	if !errors.Is(err, ErrSelloDistinto) {
		t.Fatalf("centinela: %v", err)
	}
	for _, quiero := range []string{"acabaria firmada sobre quien este hoy", "campana nueva"} {
		if !strings.Contains(err.Error(), quiero) {
			t.Errorf("el error no dice %q:\n%v", quiero, err)
		}
	}
}

// UNA CAMPANA QUE NO CONSTA NO SE INVENTA, y el error dice cuales si constan.
func TestUnaCampanaQueNoEstaEnElLedgerNoSeAbreEnBlanco(t *testing.T) {
	ins := instantanea(t, censoBase)
	var l ledger.Ledger
	anotar(t, &l, aperturaDe(t, ins, "uar-h1"))
	_, err := Reconstruir("uar-h2", ins, l, nil)
	if !errors.Is(err, ErrSinIngesta) {
		t.Fatalf("centinela: %v", err)
	}
	if !strings.Contains(err.Error(), "uar-h1") {
		t.Errorf("el error no dice que campanas si constan:\n%v", err)
	}
}

// UN HECHO HUERFANO NO SE DESCARTA EN SILENCIO.
//
// Con el sello cuadrando, toda huella tendria que casar. Si una no casa, el
// ledger se ha mezclado, y descartarla dejaria un acceso contado como sin
// revisar mientras alguien firma que lo reviso.
func TestUnaDecisionQueNoCasaConNingunaFilaSeDiceEnVezDeDescartarse(t *testing.T) {
	ins := instantanea(t, censoBase)
	var l ledger.Ledger
	anotar(t, &l, aperturaDe(t, ins, "uar-h2"))
	carga, err := json.Marshal(CargaDeDecision{
		Campana: "uar-h2", Sello: ins.Sello(),
		Huella:    HuellaDeFila(ins.Sello(), "erp|u404|admin"),
		Veredicto: "aprobar",
	})
	if err != nil {
		t.Fatal(err)
	}
	anotar(t, &l, ledger.Entrada{Instante: t1, Tipo: TipoDecision, Sujeto: Sujeto("uar-h2"),
		Actor: "ciso", Carga: carga})

	_, err = Reconstruir("uar-h2", ins, l, nil)
	if !errors.Is(err, ErrHechoHuerfano) {
		t.Fatalf("centinela: %v", err)
	}
	if !strings.Contains(err.Error(), "sin revisar") {
		t.Errorf("el error no dice cual es la consecuencia:\n%v", err)
	}
}

// EL VEREDICTO DESCONOCIDO NO CAE AL VALOR POR DEFECTO.
//
// El cero de Veredicto es Aprobar, que es el permisivo: aprobar un acceso por no
// entender una palabra es exactamente lo que no puede pasar. Invariante 8 en una
// frontera de lectura.
func TestUnVeredictoQueNoSeReconoceNoSeConvierteEnAprobar(t *testing.T) {
	v, err := VeredictoDe("aprobar")
	if err != nil || v != Aprobar {
		t.Fatalf("el camino bueno no funciona: %v %v", v, err)
	}
	for _, malo := range []string{"", "APROBAR", "ok", "aprobado", "revocado"} {
		v, err := VeredictoDe(malo)
		if err == nil {
			t.Errorf("%q se ha aceptado como %q", malo, v)
			continue
		}
		if v == Aprobar && !errors.Is(err, ErrDecision) {
			t.Errorf("%q ha caido al cero sin centinela", malo)
		}
	}
	// Y por el camino de la reconstruccion, que es donde llega de verdad.
	ins := instantanea(t, censoBase)
	var l ledger.Ledger
	anotar(t, &l, aperturaDe(t, ins, "uar-h2"))
	carga, err := json.Marshal(CargaDeDecision{
		Campana: "uar-h2", Sello: ins.Sello(),
		Huella: HuellaDeFila(ins.Sello(), "erp|u1|admin"), Veredicto: "vale",
	})
	if err != nil {
		t.Fatal(err)
	}
	anotar(t, &l, ledger.Entrada{Instante: t1, Tipo: TipoDecision, Sujeto: Sujeto("uar-h2"),
		Actor: "ciso", Carga: carga})
	if _, err := Reconstruir("uar-h2", ins, l, nil); err == nil {
		t.Fatal("un veredicto que no se reconoce ha entrado en la campana")
	}
}

// EL CIERRE SE REPRODUCE Y VUELVE A PASAR SUS COMPROBACIONES.
//
// Si alguien edita el ledger para quitar una decision, el cierre ya no cuadra:
// un cierre que no se sostiene con los hechos que hay delante no es un cierre.
func TestUnCierreQueYaNoSeSostieneNoSeDaPorBueno(t *testing.T) {
	ins := instantanea(t, censoBase)
	var l ledger.Ledger
	anotar(t, &l, aperturaDe(t, ins, "uar-h2"))
	c, err := Abrir("uar-h2", ins, t0, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range ins.Filas {
		d := Decision{Fila: f.Clave(), Veredicto: Aprobar, Quien: "jefa", Cuando: t1}
		if err := c.Registrar(d); err != nil {
			t.Fatal(err)
		}
		e, err := DecisionComoEntrada(d, ins.Sello(), "uar-h2")
		if err != nil {
			t.Fatal(err)
		}
		anotar(t, &l, e)
	}
	cierre, err := c.Cerrar("ciso", t2)
	if err != nil {
		t.Fatal(err)
	}
	ec, err := CierreComoEntrada(cierre, "uar-h2")
	if err != nil {
		t.Fatal(err)
	}
	anotar(t, &l, ec)

	// Camino bueno: se reconstruye y sale cerrada.
	rec, err := Reconstruir("uar-h2", ins, l, nil)
	if err != nil {
		t.Fatalf("no se reconstruye una campana cerrada: %v", err)
	}
	if !rec.Cerrada() {
		t.Fatal("la campana no ha vuelto cerrada")
	}

	// Y ahora se le quita una decision al ledger, como haria quien lo editara.
	var sin ledger.Ledger
	for _, e := range l.Entradas {
		if e.Tipo == TipoDecision && len(sin.Entradas) == 1 {
			continue // se salta la primera decision
		}
		sin.Entradas = append(sin.Entradas, e)
	}
	if _, err := Reconstruir("uar-h2", ins, sin, nil); err == nil {
		t.Fatal("se ha dado por cerrada una campana a la que le falta una decision: eso es " +
			"exactamente lo que un ledger editado produciria")
	}
}

// LAS EXCUSAS TAMBIEN VUELVEN, con quien y por que.
func TestLasExcusasVuelvenConSuAutorYSuMotivo(t *testing.T) {
	con := "usuario;nombre;permiso\nu1;Ana Perez;admin\n;Sin Nombre;lector\nu2;Luis Gil;lector\n"
	ins := instantanea(t, con)
	var l ledger.Ledger
	anotar(t, &l, aperturaDe(t, ins, "uar-h2"))
	e, err := ExcusaComoEntrada(Excusa{Desde: 3, Hasta: 3, Quien: "ciso",
		Motivo: "fila de prueba del IdP", Cuando: t2}, ins.Sello(), "uar-h2")
	if err != nil {
		t.Fatal(err)
	}
	anotar(t, &l, e)

	c, err := Reconstruir("uar-h2", ins, l, nil)
	if err != nil {
		t.Fatal(err)
	}
	texto := c.Informar().Texto()
	if !strings.Contains(texto, "excusadas por ciso") || !strings.Contains(texto, "fila de prueba del IdP") {
		t.Fatalf("la excusa no ha vuelto:\n%s", texto)
	}
}

// Y EL SUJETO SE COMPONE EN UN SOLO SITIO. Dos formas de escribirlo es como se
// pierden entradas al leer, en silencio y sin que nada se ponga rojo.
func TestElSujetoDeUnaCampanaSeEscribeDeUnaSolaForma(t *testing.T) {
	ins := instantanea(t, censoBase)
	a := aperturaDe(t, ins, "uar-h2")
	d, err := DecisionComoEntrada(Decision{Fila: "erp|u1|admin", Veredicto: Aprobar,
		Quien: "j", Cuando: t1}, ins.Sello(), "uar-h2")
	if err != nil {
		t.Fatal(err)
	}
	x, err := ExcusaComoEntrada(Excusa{Desde: 1, Hasta: 1, Quien: "j", Motivo: "m", Cuando: t1},
		ins.Sello(), "uar-h2")
	if err != nil {
		t.Fatal(err)
	}
	cc, err := CierreComoEntrada(Cierre{Quien: "j", Cuando: t2, Sello: ins.Sello()}, "uar-h2")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range []ledger.Entrada{a, d, x, cc} {
		if e.Sujeto != Sujeto("uar-h2") {
			t.Errorf("entrada de tipo %q con sujeto %q", e.Tipo, e.Sujeto)
		}
	}
	// Y los cuatro tipos son distintos, porque leer el ledger por tipo es como
	// se separan los hechos de las afirmaciones.
	tipos := map[string]bool{}
	for _, e := range []ledger.Entrada{a, d, x, cc} {
		if tipos[e.Tipo] {
			t.Errorf("dos entradas comparten tipo %q", e.Tipo)
		}
		tipos[e.Tipo] = true
	}
}

// El instante de la entrada ES el de la decision, no el de escribirla.
//
// Importa para el acta: lo que se pregunta es cuando se DECIDIO, y si la entrada
// llevara el instante de la escritura, una decision del martes anotada el jueves
// diria jueves.
func TestElInstanteDeLaEntradaEsElDeLaDecision(t *testing.T) {
	ins := instantanea(t, censoBase)
	cuando := time.Date(2026, 9, 3, 17, 30, 0, 0, time.UTC)
	e, err := DecisionComoEntrada(Decision{Fila: "erp|u1|admin", Veredicto: Aprobar,
		Quien: "j", Cuando: cuando}, ins.Sello(), "uar-h2")
	if err != nil {
		t.Fatal(err)
	}
	if !e.Instante.Equal(cuando) {
		t.Fatalf("instante %v, esperado %v", e.Instante, cuando)
	}
}
