package corpus

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/marcosmatalab/plazum/nucleo/ventana"
)

func periodicaConReapertura(reabre ...string) Obligacion {
	return Obligacion{
		ID: "o", Articulo: "ritual plazum sobre el punto 1", Cita: "c",
		ClaseE2E: "documental", TextoLegal: "texto",
		Temporalidad: &Temporalidad{
			Primitiva: "periodica", Hito: "revision", Cadencia: "P12M",
			Regimen:    RegimenSpec{Computo: "naturales", Cierre: "fin_de_dia", Traslado: "ninguno"},
			Disparador: map[string]string{"hecho": "ultima_revision"},
			ReabrePor:  reabre,
		},
	}
}

// Un hecho posterior a la ultima ejecucion REABRE el ciclo, y el ciclo deja de
// mandar.
//
// EL PORQUE DEL MODELO. Casi todo punto de revision del anexo de 2024/2690 dice
// «a intervalos planificados O CUANDO SE PRODUZCAN INCIDENTES SIGNIFICATIVOS».
// Eso no crea un segundo deber: crea un segundo disparador del mismo deber.
// Escribirlos como dos obligaciones duplicaria el recuento (22 de 47 en ese
// anexo) y le diria al cliente que tiene el doble de ceremonias.
func TestUnHechoPosteriorReabreElCicloYQuitaLaFecha(t *testing.T) {
	o := periodicaConReapertura("ultimo_incidente_significativo")
	hechos := ventana.Hechos{
		"ultima_revision":                time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
		"ultimo_incidente_significativo": time.Date(2026, 5, 3, 0, 0, 0, 0, time.UTC),
	}
	vs, err := VencimientosDe(o, hechos, time.Date(2027, 8, 27, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if len(vs) != 1 {
		t.Fatalf("%d vencimientos, esperaba 1", len(vs))
	}
	// SIN PLAZO LEGAL, y no una fecha inventada: la norma dice CUANDO hay que
	// revisar (al ocurrir el hecho) y no da plazo para hacerlo. Poner aqui una
	// fecha limite seria inventarse un numero que el texto no da, que es lo que
	// este repositorio lleva un corpus entero evitando.
	if vs[0].Estado != ventana.SinPlazoLegal {
		t.Errorf("estado %v, esperaba «sin plazo legal»: la norma no da plazo para la "+
			"revision reabierta y no se le inventa uno", vs[0].Estado)
	}
	for _, quiero := range []string{"ultimo_incidente_significativo", "2026-05-03", "REABRE"} {
		if !strings.Contains(vs[0].Regla, quiero) {
			t.Errorf("la regla no dice %q: %s", quiero, vs[0].Regla)
		}
	}
}

// Un hecho ANTERIOR a la ultima ejecucion no reabre nada.
//
// Es el caso normal: hubo un incidente, se revisó despues, y el ciclo sigue su
// curso. Sin esta mitad, cualquier incidente historico dejaria la obligacion
// abierta para siempre.
func TestUnHechoAnteriorALaUltimaEjecucionNoReabre(t *testing.T) {
	o := periodicaConReapertura("ultimo_incidente_significativo")
	hechos := ventana.Hechos{
		"ultimo_incidente_significativo": time.Date(2025, 5, 3, 0, 0, 0, 0, time.UTC),
		"ultima_revision":                time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
	}
	vs, err := VencimientosDe(o, hechos, time.Date(2027, 8, 27, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if len(vs) == 0 || vs[0].Estado != ventana.Determinado {
		t.Fatalf("el ciclo tenia que seguir mandando: %+v", vs)
	}
	quiero := time.Date(2027, 1, 15, 23, 59, 59, 0, time.UTC)
	if !vs[0].Vence.Equal(quiero) {
		t.Errorf("vence el %s y esperaba el %s", vs[0].Vence, quiero)
	}
}

// EL EMPATE NO REABRE, y esa decision evita una obligacion que no se puede
// cerrar nunca.
//
// Si la revision consta el mismo dia del incidente, se hizo DESPUES de el (por
// eso consta). Tratar el empate como reapertura pediria repetirla para siempre:
// cada vez que se registrara la nueva revision, el incidente volveria a empatar
// con ella. No daria error; daria un bucle.
func TestUnHechoDelMismoInstanteNoReabre(t *testing.T) {
	mismo := time.Date(2026, 5, 3, 0, 0, 0, 0, time.UTC)
	o := periodicaConReapertura("ultimo_incidente_significativo")
	vs, err := VencimientosDe(o, ventana.Hechos{
		"ultima_revision":                mismo,
		"ultimo_incidente_significativo": mismo,
	}, time.Date(2027, 8, 27, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if len(vs) == 0 || vs[0].Estado != ventana.SinPlazoLegal {
		return // si reabre, cae aqui
	}
	t.Error("un hecho del mismo instante que la ultima ejecucion ha reabierto el ciclo: " +
		"eso hace una obligacion que no se puede cerrar nunca")
}

// De varias reaperturas manda la MAS RECIENTE, que es desde cuando se mide.
func TestDeVariasReaperturasMandaLaMasReciente(t *testing.T) {
	o := periodicaConReapertura("ultimo_incidente_significativo", "ultimo_cambio_significativo")
	vs, err := VencimientosDe(o, ventana.Hechos{
		"ultima_revision":                time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
		"ultimo_incidente_significativo": time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		"ultimo_cambio_significativo":    time.Date(2026, 6, 20, 0, 0, 0, 0, time.UTC),
	}, time.Date(2027, 8, 27, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if len(vs) != 1 || !strings.Contains(vs[0].Regla, "ultimo_cambio_significativo") {
		t.Errorf("tenia que mandar la mas reciente: %+v", vs)
	}
}

// El linter: una reapertura que no puede dispararse jamas es una proteccion
// afirmada y no cumplida.
func TestElLinterRechazaReaperturasQueNoPuedenDispararse(t *testing.T) {
	casos := []struct {
		nombre    string
		o         Obligacion
		centinela error
	}{
		{"vacia", periodicaConReapertura(""), ErrReaperturaSinNombre},
		{"es el propio disparador", periodicaConReapertura("ultima_revision"),
			ErrReaperturaEsElDisparador},
		{"repetida", periodicaConReapertura("ultimo_incidente", "ultimo_incidente"),
			ErrReaperturaRepetida},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			p := &Paquete{URN: "x@1", Obligaciones: []Obligacion{c.o}}
			var errs []error
			p.validarOrigenDelIntervalo(func(e error) { errs = append(errs, e) })
			for _, e := range errs {
				if errors.Is(e, c.centinela) {
					return
				}
			}
			t.Fatalf("el linter da por buena una reapertura %s. Lo que vio: %v", c.nombre, errs)
		})
	}
}

// Y `reabre_por` no cabe en algo que no tiene ciclo.
func TestElLinterRechazaReabrirLoQueNoTieneCiclo(t *testing.T) {
	o := Obligacion{
		ID: "o", Articulo: "art. 1", Cita: "c", ClaseE2E: "documental", TextoLegal: "t",
		Temporalidad: &Temporalidad{
			Primitiva: "plazo", Hito: "limite", Limite: "P10D",
			Regimen:    RegimenSpec{Computo: "naturales", Cierre: "fin_de_dia", Traslado: "ninguno"},
			Disparador: map[string]string{"hecho": "x"},
			ReabrePor:  []string{"ultimo_incidente_significativo"},
		},
	}
	p := &Paquete{URN: "x@1", Obligaciones: []Obligacion{o}}
	var errs []error
	p.validarOrigenDelIntervalo(func(e error) { errs = append(errs, e) })
	for _, e := range errs {
		if errors.Is(e, ErrReaperturaFueraDeSitio) {
			return
		}
	}
	t.Fatalf("un plazo no tiene ciclo que reabrir y el linter lo acepta: %v", errs)
}
