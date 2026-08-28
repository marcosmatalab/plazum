package calendario

import (
	"strings"
	"testing"
	"time"

	"github.com/marcosmatalab/plazum/nucleo/pantalla"
)

// LAS TRANSICIONES EN EL .ics: lo que empieza y lo que deja de obligarte.
//
// EL AGUJERO QUE CIERRA. `Escribir` solo recorria `cal.Meses`, o sea solo los
// VENCIMIENTOS. La pantalla decia "empieza a obligarte el 11-09-2026" y el
// fichero que el usuario se lleva a Outlook no lo decia. En el perfil del
// fabricante de software, la fila con mas actualidad de todo el producto salia
// por pantalla y se quedaba en la pantalla.

func conTransiciones() pantalla.Calendario {
	dia := func(a int, m time.Month, d int) time.Time {
		return time.Date(a, m, d, 0, 0, 0, 0, time.UTC)
	}
	return pantalla.Calendario{
		Desde: instante, Hasta: instante.AddDate(1, 0, 0),
		Estrenos: []pantalla.Estreno{{
			Desde: dia(2026, time.September, 11), Marco: "urn:eu:reg:2024:2847",
			Obligacion: "cra.art14.incidente", Titulo: "Notificar un incidente grave",
			Articulo: "14.3 y 14.4", Cita: "cita del estreno", Hitos: 4,
		}},
		Ceses: []pantalla.Cese{{
			Hasta: dia(2027, time.March, 15), Marco: "urn:es:rd:2022:311",
			Obligacion: "ens.dt.adecuacion", Titulo: "Adecuacion de sistemas preexistentes",
			Articulo: "disposicion transitoria unica", Cita: "cita del cese", Hitos: 1,
		}},
	}
}

func TestElEstrenoYElCeseViajanEnElFichero(t *testing.T) {
	ls := desplegarSegunElRFC(t, escribir(t, conTransiciones()))
	unido := strings.Join(ls, "\n")

	if n := strings.Count(unido, "BEGIN:VEVENT"); n != 2 {
		t.Fatalf("%d eventos, esperaba 2: un estreno y un cese", n)
	}
	for _, quiero := range []string{
		"SUMMARY:Empieza a obligarte: Notificar un incidente grave",
		"SUMMARY:Deja de obligarte: Adecuacion de sistemas preexistentes",
	} {
		if !strings.Contains(unido, quiero) {
			t.Errorf("falta la linea %q.\n%s", quiero, unido)
		}
	}
	// El SUMMARY dice QUE es, no solo el titulo. Sin el prefijo, un estreno en
	// la agenda de alguien se lee como un vencimiento y le hace creer que ese
	// dia tiene que entregar algo.
	if strings.Contains(unido, "SUMMARY:Notificar un incidente grave") {
		t.Error("un estreno sin prefijo se lee como un vencimiento")
	}
}

// UNA TRANSICION ES UN DIA, UN VENCIMIENTO ES UN INSTANTE, y en iCalendar eso
// son dos tipos distintos. Es la regla contraria a la del vencimiento, que es
// justo lo que la hace facil de equivocar.
func TestUnaTransicionEsUnEventoDeDiaCompleto(t *testing.T) {
	ls := desplegarSegunElRFC(t, escribir(t, conTransiciones()))

	var vistos []string
	for _, l := range ls {
		if strings.HasPrefix(l, "DTSTART") || strings.HasPrefix(l, "DTEND") {
			vistos = append(vistos, l)
		}
	}
	quiero := []string{
		"DTSTART;VALUE=DATE:20260911",
		// DTEND de un evento de dia completo es EXCLUSIVO (RFC 5545, 3.8.2.2):
		// para ocupar el dia 11 hay que cerrar en el 12. Con el mismo dia, unos
		// clientes lo pintan de duracion cero y otros lo esconden.
		"DTEND;VALUE=DATE:20260912",
		"DTSTART;VALUE=DATE:20270315",
		"DTEND;VALUE=DATE:20270316",
	}
	for _, q := range quiero {
		encontrado := false
		for _, v := range vistos {
			if v == q {
				encontrado = true
			}
		}
		if !encontrado {
			t.Errorf("falta %q. Lo que hay: %v", q, vistos)
		}
	}
	// Y NINGUNA transicion sale como DATE-TIME: eso la pondria a las 00:00Z, y
	// con zona horaria por medio la mueve al dia ANTERIOR en media Europa.
	for _, v := range vistos {
		if strings.HasPrefix(v, "DTSTART:") || strings.HasPrefix(v, "DTEND:") {
			t.Errorf("una transicion ha salido como instante y no como dia: %q", v)
		}
	}
}

// EL UID, por las dos propiedades que tiene que cumplir a la vez.
func TestElUIDDeUnaTransicionEsEstableYNoChocaConNadie(t *testing.T) {
	uids := func(s string) []string {
		var out []string
		for _, l := range desplegarSegunElRFC(t, s) {
			if strings.HasPrefix(l, "UID:") {
				out = append(out, l)
			}
		}
		return out
	}
	a := uids(escribir(t, conTransiciones()))
	b := uids(escribir(t, conTransiciones()))
	if len(a) != 2 {
		t.Fatalf("%d UID, esperaba 2", len(a))
	}
	// ESTABLE: reimportar no puede duplicar eventos en el calendario de nadie.
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("el UID cambia entre ejecuciones: %s vs %s", a[i], b[i])
		}
	}
	if a[0] == a[1] {
		t.Fatal("el estreno y el cese comparten UID")
	}

	// Y NO CHOCA CON UN VENCIMIENTO de la misma obligacion. La clase entra en
	// el hash justo por esto: sin ella, una obligacion que estrena el mismo dia
	// que vence algo suyo produciria dos eventos con el mismo UID y el cliente
	// se quedaria con uno.
	mismoDia := conTransiciones()
	mismoDia.Estrenos[0].Obligacion = "demo.una"
	mismoDia.Estrenos[0].Marco = "urn:demo"
	todos := map[string]bool{}
	for _, u := range uids(escribir(t, mismoDia)) {
		if todos[u] {
			t.Errorf("UID repetido: %s", u)
		}
		todos[u] = true
	}
}

// La marca de supuesto llega hasta el .ics, igual que en las fechas: un perfil
// que exporta a Outlook sin marcar convierte una conjetura en una cita.
func TestLaMarcaDeSupuestoLlegaAlEstrenoDelFichero(t *testing.T) {
	cal := conTransiciones()
	cal.Estrenos[0].Supuesta = true
	cal.Ceses[0].Supuesta = true
	s := strings.Join(desplegarSegunElRFC(t, escribir(t, cal)), "\n")
	if !strings.Contains(s, "SUMMARY:[supuesto] Empieza a obligarte:") {
		t.Error("el estreno supuesto no lleva la marca en el SUMMARY")
	}
	if !strings.Contains(s, "SUMMARY:[supuesto] Deja de obligarte:") {
		t.Error("el cese supuesto no lleva la marca en el SUMMARY")
	}
	if !strings.Contains(s, "SUPUESTO: sale de un perfil de arranque") {
		t.Error("la descripcion no dice de donde sale")
	}
}
