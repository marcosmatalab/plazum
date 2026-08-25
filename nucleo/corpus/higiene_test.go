package corpus

import (
	"errors"
	"strings"
	"testing"
)

// La higiene legal del corpus, con sus dos puertas nuevas.
//
// QUE SE VIGILA AQUI Y POR QUE NO BASTABA CON LO QUE HABIA.
//
//  1. Todo paquete declara de que REGIMEN sale su contenido (licencia_fuente) y
//     que AVISO hay que ensenar a quien lo usa (atribucion). Antes el estrato se
//     declaraba con `clase` y la licencia iba suelta en un texto libre. Eso no
//     basta: la Decision 2011/833/UE autoriza reutilizar el DOUE con atribucion,
//     y una atribucion que vive en la cabeza de quien escribio el paquete no es
//     una atribucion. Si no es un dato, no se puede ensenar.
//  2. Las reglas de aplicabilidad ENTRAN en la frontera legal. Estaban fuera,
//     con la excusa de que tienen su propio linter; ese linter comprueba que la
//     regla se parsea, no cuanto texto lleva dentro.
//
// Cada caso lleva su control negativo pegado: se demuestra que el mismo paquete,
// arreglado, valida limpio. Sin eso, un linter que rechazara todo pasaria igual.

// TestUnPaqueteSinLicenciaFuenteNoCarga: el campo es obligatorio en las cinco
// clases, tambien en las que no le deben atribucion a nadie.
func TestUnPaqueteSinLicenciaFuenteNoCarga(t *testing.T) {
	p := base()
	p.LicenciaFuente = ""
	errs := p.Validar()
	if !contiene(errs, ErrSinLicenciaFuente) {
		t.Fatalf("un paquete sin licencia_fuente tiene que rechazarse: %v", errs)
	}
	// Y el error dice QUE hay que escribir, no solo que falta algo.
	if !strings.Contains(errs[0].Error(), string(BOETRLPI13)) {
		t.Errorf("el error no enumera los regimenes que admite la clase, asi que quien lo "+
			"lea no sabe con que rellenarlo: %v", errs[0])
	}

	// Control negativo: con el campo puesto, el mismo paquete valida limpio.
	p.LicenciaFuente = BOETRLPI13
	if errs := p.Validar(); len(errs) != 0 {
		t.Fatalf("el paquete con licencia_fuente tiene que validar: %v", errs)
	}
}

// TestUnPaqueteSinAtribucionNoCarga: el aviso viaja con el paquete o no existe.
func TestUnPaqueteSinAtribucionNoCarga(t *testing.T) {
	p := base()
	p.Atribucion = ""
	if errs := p.Validar(); !contiene(errs, ErrSinAtribucion) {
		t.Fatalf("un paquete sin atribucion tiene que rechazarse: %v", errs)
	}

	// Control negativo.
	p.Atribucion = "Texto reutilizado citando la fuente oficial."
	if errs := p.Validar(); len(errs) != 0 {
		t.Fatalf("el paquete con atribucion tiene que validar: %v", errs)
	}
}

// TestLaLicenciaFuenteTieneQueSerCoherenteConLaClase.
//
// Es la comprobacion que impide que los dos campos se separen. Un paquete que se
// declara referencial (no puede redistribuir texto) y a la vez dice que su
// contenido viene amparado por el art. 13 TRLPI (puede redistribuirlo entero)
// miente en uno de los dos sitios, y el linter no puede adivinar en cual: lo
// para y lo dice.
func TestLaLicenciaFuenteTieneQueSerCoherenteConLaClase(t *testing.T) {
	casos := []struct {
		que   string
		clase Clase
		lf    LicenciaFuente
		vale  bool
	}{
		{"referencial que dice traer texto de una ley", Referencial, BOETRLPI13, false},
		{"referencial coherente", Referencial, SinLicenciaDeTexto, true},
		{"transcrito que dice no traer texto", Transcrito, SinLicenciaDeTexto, false},
		{"transcrito de fuente espanola", Transcrito, BOETRLPI13, true},
		{"transcrito de fuente de la Union", Transcrito, DOUEDecision2011833, true},
		{"delegado que se atribuye el texto", Delegado, DOUEDecision2011833, false},
		{"delegado coherente", Delegado, LaTieneLaHerramienta, true},
		{"importado coherente", Importado, DominioPublicoEEUU, true},
		{"importado con regimen de otro estrato", Importado, BOETRLPI13, false},
		{"propio coherente", Propio, DelProyecto, true},
		{"propio reutilizando el sector publico", Propio, RISPConAtribucion, true},
	}
	for _, c := range casos {
		t.Run(c.que, func(t *testing.T) {
			p := base()
			p.Clase, p.LicenciaFuente = c.clase, c.lf
			// Un delegado no puede llevar texto legal ni quedarse sin
			// herramienta: se ajusta para que el UNICO fallo posible sea el
			// de coherencia y el caso mida lo que dice medir.
			if c.clase == Delegado {
				p.Obligaciones[0].TextoLegal = ""
				p.Obligaciones[0].Delegado = "openscap:xccdf_org.demo_benchmark"
			}
			if c.clase == Referencial {
				p.Obligaciones[0].TextoLegal = "C.5.1 Titulo corto"
			}
			errs := p.Validar()
			incoherente := contiene(errs, ErrLicenciaFuenteIncoherente)
			if c.vale && incoherente {
				t.Fatalf("clase %s con %q es coherente y se rechaza: %v", c.clase, c.lf, errs)
			}
			if !c.vale && !incoherente {
				t.Fatalf("HALLAZGO: clase %s declara %q y el linter no dice nada. Uno de los "+
					"dos campos miente y el paquete se publica igual: %v", c.clase, c.lf, errs)
			}
		})
	}
}

// TestLasLicenciasProhibidasSonListaNegraYNoPendiente.
//
// El vocabulario cerrado ya rechaza cualquier cadena desconocida, asi que estas
// cuatro entradas no anaden ni una linea de seguridad: anaden el PORQUE. Alguien
// va a volver a proponer los CIS Controls o el SCF gratuito dentro de seis
// meses, y lo que se tiene que encontrar es el motivo escrito, no un "valor
// invalido" que invita a insistir.
func TestLasLicenciasProhibidasSonListaNegraYNoPendiente(t *testing.T) {
	if len(licenciasProhibidas) == 0 {
		t.Fatal("la lista negra esta vacia, asi que este test recorreria la nada")
	}
	for lf, motivo := range licenciasProhibidas {
		t.Run(string(lf), func(t *testing.T) {
			if motivo == "" {
				t.Fatalf("%q esta prohibida y no dice por que. Una prohibicion sin motivo se "+
					"lee como un pendiente y alguien la reabre", lf)
			}
			p := base()
			p.LicenciaFuente = lf
			errs := p.Validar()
			if !contiene(errs, ErrLicenciaProhibida) {
				t.Fatalf("HALLAZGO: %q entra en el corpus: %v", lf, errs)
			}
			if !strings.Contains(errs[0].Error(), motivo) {
				t.Errorf("el rechazo de %q no lleva el motivo delante: %v", lf, errs[0])
			}
			// Y no se cuela por la puerta de al lado: una licencia prohibida
			// no puede ser ademas una licencia conocida de ninguna clase.
			if licenciaConocida(lf) {
				t.Errorf("%q esta a la vez en la lista negra y en el vocabulario admitido, "+
					"asi que el orden de las comprobaciones decide si entra o no", lf)
			}
		})
	}

	// Control negativo: un regimen permitido no se rechaza.
	p := base()
	p.LicenciaFuente = BOETRLPI13
	if errs := p.Validar(); contiene(errs, ErrLicenciaProhibida) {
		t.Fatalf("el detector de licencias prohibidas muerde a una permitida: %v", errs)
	}
}

// TestUnaLicenciaFuenteInventadaNoCarga: el vocabulario es cerrado, y esa es
// toda la gracia. Una fuente nueva entra con su constante y su fila en
// docs/LICENCIAS.md, no escribiendo otra cadena en un JSON.
func TestUnaLicenciaFuenteInventadaNoCarga(t *testing.T) {
	p := base()
	p.LicenciaFuente = "la-que-me-parezca"
	if errs := p.Validar(); !contiene(errs, ErrLicenciaFuenteDesconocida) {
		t.Fatalf("una licencia_fuente inventada tiene que rechazarse: %v", errs)
	}

	// Control negativo: las del vocabulario cargan, cada una con su clase.
	for clase, lista := range licenciasPorClase {
		for _, lf := range lista {
			q := base()
			q.Clase, q.LicenciaFuente = clase, lf
			if errs := q.Validar(); contiene(errs, ErrLicenciaFuenteDesconocida) {
				t.Errorf("%q esta en el vocabulario y el linter dice que no existe: %v", lf, errs)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// La aplicabilidad, dentro de la frontera legal.
// ---------------------------------------------------------------------------

// referencialConReglas es un referencial legitimo que ademas declara una regla
// de aplicabilidad. Es el caso real: un marco referencial tambien decide a quien
// alcanza cada control.
func referencialConReglas() *Paquete {
	p := referencialConTexto()
	p.Aplicabilidad = Aplicabilidad{
		Reglas: []ReglaSpec{{
			ID:    "demo.r.ambito",
			Cita:  "CAT/DEMO 9999:2026 clausula 4.3",
			Regla: `aplica("demo.auditoria_bienal", S) :- en_ambito(S)`,
		}},
	}
	return p
}

// TestLaFronteraLegalMiraLasReglasDeAplicabilidad.
//
// EL AGUJERO QUE CIERRA. El limite de texto se amplio a los veinte y pico campos
// del formato, y el bloque `aplicabilidad` se quedo fuera: sus cadenas (la regla
// y su cita) no pasaban por ningun techo. Una regla con cita es un sitio por el
// que se cuela el enunciado de un control, y encima con pinta de tecnicismo que
// nadie relee.
func TestLaFronteraLegalMiraLasReglasDeAplicabilidad(t *testing.T) {
	// poner deja el campo con EXACTAMENTE n bytes. En la regla hay que
	// descontar el envoltorio del dialecto, o el caso al limite se pasa de
	// largo por la sintaxis y no por el texto, que es medir otra cosa.
	relleno := func(n int) string { return strings.Repeat("y", n) }

	casos := []struct {
		campo string
		poner func(p *Paquete, n int)
	}{
		{"Reglas[].Cita", func(p *Paquete, n int) {
			p.Aplicabilidad.Reglas[0].Cita = relleno(n)
		}},
		{"Reglas[].Regla", func(p *Paquete, n int) {
			envoltorio := `aplica("", S) :- en_ambito(S)`
			p.Aplicabilidad.Reglas[0].Regla = `aplica("` + relleno(n-len(envoltorio)) +
				`", S) :- en_ambito(S)`
		}},
		{"Reglas[].ID", func(p *Paquete, n int) {
			p.Aplicabilidad.Reglas[0].ID = relleno(n)
		}},
		{"Exporta[]", func(p *Paquete, n int) {
			p.Aplicabilidad.Exporta = []string{relleno(n)}
		}},
	}

	for _, c := range casos {
		t.Run(c.campo, func(t *testing.T) {
			p := referencialConReglas()
			c.poner(p, LimiteCitaReferencial+1)
			if !contiene(p.Validar(), ErrCitaDesbordada) {
				t.Fatalf("HALLAZGO: %s se lleva %d caracteres dentro de un paquete "+
					"referencial y la frontera legal no dice nada",
					c.campo, LimiteCitaReferencial+1)
			}
			// Control negativo, en el mismo caso: con un caracter menos, la
			// frontera calla. Sin esto no se distingue un limite de un linter
			// que rechaza cualquier cosa que se le ponga en ese campo.
			q := referencialConReglas()
			c.poner(q, LimiteCitaReferencial)
			if contiene(q.Validar(), ErrCitaDesbordada) {
				t.Fatalf("%s con exactamente %d caracteres tiene que valer: %v",
					c.campo, LimiteCitaReferencial, q.Validar())
			}
		})
	}
}

// TestUnTranscritoSiPuedeLlevarReglasLargas: la frontera es de la CLASE, no del
// campo. Un paquete transcrito puede redistribuir el texto entero, asi que sus
// reglas no llevan techo. Si este test se pusiera rojo, el arreglo del agujero
// habria pasado a limitar a quien tiene derecho a no estar limitado.
func TestUnTranscritoSiPuedeLlevarReglasLargas(t *testing.T) {
	p := referencialConReglas()
	p.Clase, p.LicenciaFuente = Transcrito, BOETRLPI13
	p.Obligaciones[0].TextoLegal = strings.Repeat("y", LimiteCitaReferencial+1)
	p.Aplicabilidad.Reglas[0].Cita = strings.Repeat("y", LimiteCitaReferencial+1)
	if errs := p.Validar(); contiene(errs, ErrCitaDesbordada) {
		t.Fatalf("un transcrito no lleva techo de texto y se le esta aplicando: %v", errs)
	}
}

// contiene dice si alguno de los errores es el centinela dado. Por identidad y
// no por subcadena: en este linter "clase" aparece dentro de "clase_e2e" y eso
// ya dio siete tests en verde con el fallo delante.
func contiene(errs []error, centinela error) bool {
	for _, e := range errs {
		if errors.Is(e, centinela) {
			return true
		}
	}
	return false
}
