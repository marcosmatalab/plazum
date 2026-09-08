package corpus

import (
	"errors"
	"strings"
	"testing"
)

// LAS DOS DIRECCIONES DEL BLOQUE DE TRANSPOSICION, Y LAS DOS CON CONTROL.
//
// LA PUERTA NACIO ROJA SOBRE EL ARBOL REAL, Y TRES VECES SEGUIDAS, que es mas de
// lo que yo esperaba y merece contarse: la escribi creyendo que su universo era
// UNO. Se puso roja sobre `paquetes/nis2-ue`, que es el caso que la motivo; al
// arreglarlo se puso roja sobre `esqueletos/csrd`, y al arreglar ese, sobre
// `esqueletos/psd2`. TRES directivas, no una, y las dos ultimas no las habia
// mirado nadie.
//
// Y la tercera cambio el modelo: la ficha de la CSRD trae UNA sola fecha de
// transposicion sin distinguir adopcion de aplicacion, asi que la clase `limite`
// existe porque el dato real la pidio. La primera version del parser la habria
// rechazado. Ver `TestUnaFichaQueNoDistingueAdopcionDeAplicacionLoDiceEnVezDeAdivinar`
// en herramientas/ingestanorma.
//
// Y `psd2` trajo el primer `consta: true` DE VERDAD del arbol: su transposicion
// espanola es el RDL 19/2018, que ya es un paquete escrito. Hasta entonces esa
// rama solo la recorria el dato sintetico de aqui abajo.
//
// Lo que sigue sin recorrer ninguna entrada real es la DIRECCION 2, un reglamento
// que declara el bloque, y por eso va con dato sintetico y se dice.

// urnDe compone un urn del ESQUEMA, sin escribir nunca el prefijo entero.
//
// No es un rodeo para esquivar al invariante 2: es lo que el invariante 2 pide.
// Un test que cablease `urn:eu:dir:2022:2555` estaria metiendo el identificador
// de NIS2 en el codigo, y lo que este test prueba no es NIS2, es la forma del
// urn. Componiendolo de las mismas constantes que usa `esDirectiva`, el test y el
// detector hablan del mismo vocabulario y ninguno nombra una norma.
func urnDe(jurisdiccion, acto string) string {
	return strings.Join([]string{"urn", jurisdiccion, acto, "9999", "1"}, ":")
}

func paqueteDirectiva() *Paquete {
	return &Paquete{
		URN: urnDe(jurisdiccionUE, actoDirectiva),
		Transposicion: &Transposicion{
			Cita:             "acto ficticio 9999/1, art. 41.1",
			LimiteAdopcion:   "2024-10-17",
			LimiteAplicacion: "2024-10-18",
			Estado: []EstadoDeTransposicion{{
				Pais:            "ES",
				Consta:          false,
				VinculaMientras: "la norma nacional que obliga hoy, con su referencia",
				Comprobado:      "2026-08-26",
				Como:            "indice de legislacion consolidada del BOE, tres busquedas",
			}},
		},
	}
}

func fallosDe(p *Paquete) []error {
	var out []error
	p.validarTransposicion(func(err error) { out = append(out, err) })
	return out
}

func TestUnBloqueDeTransposicionBienEscritoPasa(t *testing.T) {
	if fs := fallosDe(paqueteDirectiva()); len(fs) != 0 {
		t.Fatalf("el bloque correcto tiene que pasar y ha dado %d fallos: %v", len(fs), fs)
	}
	// Y un REGLAMENTO sin bloque tambien: es el caso de veinte de los veintiun
	// paquetes, y una puerta que los acusara seria un rojo permanente.
	if fs := fallosDe(&Paquete{URN: urnDe(jurisdiccionUE, "reg")}); len(fs) != 0 {
		t.Fatalf("un reglamento sin bloque no tiene nada que declarar: %v", fs)
	}
}

// DIRECCION 1: la directiva que se calla.
func TestUnaDirectivaSinBloqueDeTransposicionNoPasa(t *testing.T) {
	fs := fallosDe(&Paquete{URN: urnDe(jurisdiccionUE, actoDirectiva)})
	if len(fs) != 1 || !errors.Is(fs[0], ErrDirectivaSinTransposicion) {
		t.Fatalf("esperaba ErrDirectivaSinTransposicion y salio: %v", fs)
	}
}

// DIRECCION 2, LA CARA, con dato sintetico porque el corpus no la alcanza.
//
// Un reglamento con bloque de transposicion le dice al cliente que espere a una
// norma nacional que no va a existir. La direccion 1 hace que falte un aviso; esta
// hace que sobre, y la que sobra RETRASA un cumplimiento que ya vincula.
func TestUnReglamentoConBloqueDeTransposicionNoPasa(t *testing.T) {
	p := paqueteDirectiva()
	p.URN = urnDe(jurisdiccionUE, "reg")
	fs := fallosDe(p)
	if len(fs) != 1 || !errors.Is(fs[0], ErrTransposicionEnLoQueNoEsDirectiva) {
		t.Fatalf("esperaba ErrTransposicionEnLoQueNoEsDirectiva y salio: %v", fs)
	}
	if !strings.Contains(fs[0].Error(), "RETRASA") {
		t.Error("el mensaje no dice por que esta direccion es la cara, y sin eso quien la " +
			"lea la arregla quitando el urn en vez de quitando el bloque")
	}
}

// EL VALOR CERO DE `consta` (invariante 8), con las dos ramas recorridas.
func TestLasDosRamasDeConstaExigenSuApoyo(t *testing.T) {
	casos := []struct {
		nombre string
		toca   func(*EstadoDeTransposicion)
		quiero error
	}{
		{"no consta y no dice que vincula mientras: el valor cero suelto",
			func(e *EstadoDeTransposicion) { e.VinculaMientras = "" },
			ErrPaisQueNoConstaSinApoyo},
		{"consta y no dice cual es la norma",
			func(e *EstadoDeTransposicion) { e.Consta = true; e.Norma = "" },
			ErrPaisQueConstaSinNorma},
		{"sin fecha de comprobacion",
			func(e *EstadoDeTransposicion) { e.Comprobado = "" },
			ErrPaisSinComprobacion},
		{"sin decir como se comprobo",
			func(e *EstadoDeTransposicion) { e.Como = "en el BOE" },
			ErrPaisSinMetodo},
		{"con una fecha de comprobacion que no es una fecha",
			func(e *EstadoDeTransposicion) { e.Comprobado = "hace poco" },
			ErrPaisSinComprobacion},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			p := paqueteDirectiva()
			c.toca(&p.Transposicion.Estado[0])
			fs := fallosDe(p)
			hay := false
			for _, f := range fs {
				if errors.Is(f, c.quiero) {
					hay = true
				}
			}
			if !hay {
				t.Errorf("esperaba %v y salio: %v", c.quiero, fs)
			}
		})
	}
	// CONTROL POSITIVO DE LA RAMA QUE CONSTA, que hoy no recorre ningun paquete:
	// sin el, «consta: true» es una rama que nadie ejerce y su exigencia no
	// existiria aunque estuviera escrita.
	p := paqueteDirectiva()
	p.Transposicion.Estado[0].Consta = true
	p.Transposicion.Estado[0].Norma = "la ley nacional que la transpone, con su referencia"
	p.Transposicion.Estado[0].VinculaMientras = ""
	if fs := fallosDe(p); len(fs) != 0 {
		t.Errorf("una transposicion que consta y dice su norma tiene que pasar: %v", fs)
	}
}

// Y LAS PIEZAS DEL PROPIO BLOQUE.
func TestElBloqueDeTransposicionExigeCitaPaisesYFechasLegibles(t *testing.T) {
	p := paqueteDirectiva()
	p.Transposicion.Cita = "la directiva"
	p.Transposicion.LimiteAdopcion = "octubre de 2024"
	p.Transposicion.Estado = nil
	fs := fallosDe(p)
	for _, quiero := range []error{
		ErrTransposicionSinCita, ErrFechaDeTransposicionMala, ErrTransposicionSinPaises,
	} {
		hay := false
		for _, f := range fs {
			if errors.Is(f, quiero) {
				hay = true
			}
		}
		if !hay {
			t.Errorf("esperaba %v entre los fallos y salio: %v", quiero, fs)
		}
	}

	// El mismo pais dos veces, que es como dos respuestas contrarias conviven.
	q := paqueteDirectiva()
	q.Transposicion.Estado = append(q.Transposicion.Estado, q.Transposicion.Estado[0])
	hay := false
	for _, f := range fallosDe(q) {
		if errors.Is(f, ErrPaisRepetido) {
			hay = true
		}
	}
	if !hay {
		t.Error("dos entradas del mismo pais tienen que romper: si no, una dice que consta y " +
			"otra que no, y manda la que se lea primero")
	}
}

// CONTROL NEGATIVO DEL DETECTOR DE DIRECTIVAS.
//
// Su fallo probable es de sobra: casar cualquier urn que lleve las letras «dir»,
// y en este corpus hay `urn:eu:reg-ejec:...` y `urn:es:rdl:...`, que se le parecen
// lo suficiente para que la duda valga la pena.
func TestElDetectorDeDirectivasNoCazaDeMas(t *testing.T) {
	casos := map[string]bool{
		urnDe(jurisdiccionUE, actoDirectiva):                  true,
		strings.ToUpper(urnDe(jurisdiccionUE, actoDirectiva)): true,
		urnDe(jurisdiccionUE, "reg"):                          false,
		urnDe(jurisdiccionUE, "reg-ejec"):                     false,
		urnDe("es", "rdl"):                                    false,
		urnDe("es", "rd"):                                     false,
		urnDe("plazum", "demo"):                               false,
		urnDe("iso-iec", "27001"):                             false,
		"":                                                    false,
	}
	for urn, quiero := range casos {
		if got := esDirectiva(urn); got != quiero {
			t.Errorf("esDirectiva(%q) = %v y esperaba %v", urn, got, quiero)
		}
	}
}
