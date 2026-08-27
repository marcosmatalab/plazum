package corpus

import (
	"strings"
	"testing"
)

// EL RELOJ QUE OBLIGA Y NO SE PUEDE CALCULAR.
//
// El art. 67.1 del RDL 19/2018 manda notificar los incidentes de pago graves
// "de forma inmediata" y no da ningun numero. El motor sabe decir exactamente
// eso: limite indeterminado, estado "sin plazo legal", y se mide el tiempo
// transcurrido en vez de inventarse una fecha.
//
// PERO el linter exigia tres casos dorados a toda obligacion con reloj, y un
// dorado fija una FECHA esperada: de un plazo sin numero no sale ninguna. O sea
// que la regla empujaba al autor a QUITARLE EL RELOJ a esa obligacion para que
// el paquete cargara, y entonces el producto deja de ensenar el cronometro y la
// obligacion se lee como una mas, sin urgencia. La regla castigaba la
// transcripcion honesta y premiaba la que se calla.
//
// La regla nueva: los tres dorados se exigen cuando hay ALGUN limite
// computable. Cuando no lo hay, la exencion se paga con una NOTA por hito, para
// que el hueco sea una decision escrita y no el camino barato para librarse de
// los dorados.

func conReloj(t *testing.T, tmp *Temporalidad) *Paquete {
	t.Helper()
	p := base()
	p.Obligaciones[0].Temporalidad = tmp
	return p
}

func errores(errs []error) string {
	var b strings.Builder
	for _, e := range errs {
		b.WriteString(e.Error())
		b.WriteString("\n")
	}
	return b.String()
}

var regimenSimple = RegimenSpec{Computo: "naturales", Cierre: "exacto", Traslado: "ninguno"}

func TestUnRelojSinNumeroCargaSinDoradosPeroPagaLaNota(t *testing.T) {
	sinNumero := func(nota string) *Temporalidad {
		return &Temporalidad{
			Primitiva: "plazo", Regimen: regimenSimple,
			Disparador: map[string]string{"hecho": "conocimiento"},
			Hitos: []HitoSpec{{
				ID: "notificacion", Limite: "indeterminado", Nota: nota,
			}},
		}
	}

	t.Run("con nota carga, y sin un solo dorado", func(t *testing.T) {
		p := conReloj(t, sinNumero("El art. 67.1 dice de forma inmediata: hay obligacion y no "+
			"hay numero. Inventarlo seria peor que decirlo."))
		if errs := p.Validar(); len(errs) != 0 {
			t.Fatalf("no carga un reloj sin numero bien documentado:\n%s", errores(errs))
		}
	})

	t.Run("sin nota no carga: el hueco tiene que ser una decision escrita", func(t *testing.T) {
		p := conReloj(t, sinNumero(""))
		errs := p.Validar()
		if len(errs) == 0 {
			t.Fatal("ha cargado un reloj sin numero y sin explicacion, que es el camino barato " +
				"para librarse de los tres dorados")
		}
		if !strings.Contains(errores(errs), "no dice por que") {
			t.Errorf("el error no explica que falta la nota:\n%s", errores(errs))
		}
	})

	t.Run("la forma simple no vale: no tiene donde escribir la nota", func(t *testing.T) {
		p := conReloj(t, &Temporalidad{
			Primitiva: "plazo", Hito: "notificacion", Limite: "indeterminado",
			Regimen: regimenSimple, Disparador: map[string]string{"hecho": "conocimiento"},
		})
		errs := p.Validar()
		if len(errs) == 0 {
			t.Fatal("ha cargado un plazo sin numero escrito con hito y limite sueltos, que no " +
				"tiene sitio para la nota")
		}
		if !strings.Contains(errores(errs), "se escribe con") {
			t.Errorf("el error no dice como se arregla:\n%s", errores(errs))
		}
	})

	// Y el vacio, que es la otra forma de la nada: un limite ausente significa
	// lo mismo que la palabra "indeterminado" para el motor, asi que tiene que
	// significar lo mismo para el linter.
	t.Run("limite vacio se trata igual que indeterminado", func(t *testing.T) {
		p := conReloj(t, &Temporalidad{
			Primitiva: "plazo", Regimen: regimenSimple,
			Disparador: map[string]string{"hecho": "conocimiento"},
			Hitos:      []HitoSpec{{ID: "notificacion", Limite: ""}},
		})
		errs := p.Validar()
		if len(errs) == 0 {
			t.Fatal("un limite VACIO se ha colado sin nota y sin dorados: las dos formas de la " +
				"nada tienen que tratarse igual, o la que se olvida es la que se usa")
		}
		if !strings.Contains(errores(errs), "no dice por que") {
			t.Errorf("el error no es el de la nota que falta:\n%s", errores(errs))
		}
	})
}

// CONTROL NEGATIVO, y es el que sostiene todo lo de arriba: en cuanto UN limite
// es computable, los tres dorados vuelven a ser obligatorios. Sin esto, la
// relajacion podria haber apagado la regla entera y estos tests seguirian
// verdes.
func TestUnRelojConAlgunLimiteComputableSigueExigiendoTresDorados(t *testing.T) {
	casos := map[string]*Temporalidad{
		"un solo hito con numero": {
			Primitiva: "plazo", Regimen: regimenSimple,
			Disparador: map[string]string{"hecho": "conocimiento"},
			Hitos:      []HitoSpec{{ID: "notificacion", Limite: "PT24H"}},
		},
		"mezcla: uno sin numero y otro con numero": {
			Primitiva: "plazo", Regimen: regimenSimple,
			Disparador: map[string]string{"hecho": "conocimiento"},
			Hitos: []HitoSpec{
				{ID: "inicial", Limite: "indeterminado", Nota: "de forma inmediata"},
				{ID: "final", Limite: "P20D"},
			},
		},
		"la forma simple con numero": {
			Primitiva: "plazo", Hito: "notificacion", Limite: "PT72H",
			Regimen: regimenSimple, Disparador: map[string]string{"hecho": "conocimiento"},
		},
		"periodica": {
			Primitiva: "periodica", Hito: "revision", Cadencia: "P12M",
			Regimen: regimenSimple, Disparador: map[string]string{"hecho": "ultima_revision"},
		},
	}
	for nombre, tmp := range casos {
		t.Run(nombre, func(t *testing.T) {
			p := conReloj(t, tmp)
			errs := p.Validar()
			if len(errs) == 0 {
				t.Fatalf("ha cargado un reloj computable SIN dorados. La relajacion de la regla " +
					"se ha llevado por delante la regla entera")
			}
			if !strings.Contains(errores(errs), "minimo 3") {
				t.Errorf("el error no es el de los tres dorados:\n%s", errores(errs))
			}
		})
	}
}
