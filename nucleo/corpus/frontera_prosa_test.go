package corpus

import (
	"errors"
	"strings"
	"testing"
)

// LA PUERTA DE ATRAS DE LA PROSA.
//
// EL CASO REAL, y no salio de una mutacion: al justificar el intervalo del punto
// 6.7.3 del anexo de 2024/2690, el argumento propuesto fue "el sector de medios
// de pago lleva anos exigiendo la revision del conjunto de reglas de cortafuegos
// cada seis meses". Eso es criterio de PCI DSS, y el linter no lo veia: el
// limite de la frontera legal mide LONGITUD, no PROCEDENCIA.
//
// Esta puerta cierra la via ancha (nombrar el marco) y NO cierra la parafrasis
// anonima, que es justo el caso de arriba. Se dice aqui en voz alta para que
// nadie lea esta puerta como si cubriera la familia entera: la otra capa es
// humana y esta declarada en docs/pendientes.md.

func paquetesDePrueba(t *testing.T) ([]*Paquete, []string) {
	t.Helper()
	cerrado := base()
	cerrado.URN = "urn:iso-iec:27001:2022"
	cerrado.Clase = Referencial
	cerrado.LicenciaFuente = SinLicenciaDeTexto
	cerrado.Obligaciones[0].TextoLegal = "Auditoria interna"
	mio := base()
	mio.URN = "urn:demo:transcrita"
	return []*Paquete{cerrado, mio}, []string{"iso27001", "demo"}
}

func TestUnPaqueteNoNombraUnMarcoDeEstratoCerradoAjeno(t *testing.T) {
	// Las formas con las que se nombra a la 27001 en la practica, todas.
	for _, forma := range []string{
		"ISO 27001", "ISO/IEC 27001", "iso-iec 27001", "iso27001", "la 27001",
	} {
		t.Run(forma, func(t *testing.T) {
			ps, dirs := paquetesDePrueba(t)
			ps[1].Obligaciones[0].Cita = "Se revisa cada doce meses porque es lo que espera " +
				forma + " de un programa de auditoria."
			errs := ValidarProsaEntrePaquetes(ps, dirs)
			if len(errs) == 0 {
				t.Fatalf("nombrar %q en la prosa tiene que caerse", forma)
			}
			if !errors.Is(errs[0], ErrProsaNombraMarcoCerrado) {
				t.Fatalf("centinela equivocado: %v", errs[0])
			}
		})
	}
}

// CONTROL NEGATIVO 1, y es la mitad del diseno: NOMBRARSE A SI MISMO SE PUEDE.
// El paquete de la 27001 tiene que poder decir que trata de la 27001. El estrato
// referencial prohibe redistribuir su TEXTO, no mencionar que la norma existe.
func TestUnPaqueteSiPuedeNombrarseASiMismo(t *testing.T) {
	ps, dirs := paquetesDePrueba(t)
	ps[0].Obligaciones[0].Cita = "plazum, ritual sobre ISO/IEC 27001:2022 9.2.2"
	if errs := ValidarProsaEntrePaquetes(ps, dirs); len(errs) != 0 {
		t.Fatalf("un paquete tiene que poder nombrar la norma de la que trata: %v", errs)
	}
}

// CONTROL NEGATIVO 2, Y ES LA FRONTERA QUE IMPORTA: el TEXTO LEGAL no se mira
// nunca. Ahi va transcrito lo que dice el boletin, y el boletin remite a normas
// privadas continuamente (el ENS lo hace). Aplicar la regla al texto legal seria
// censurar la ley para cumplir una regla nuestra, que es al reves de para lo que
// existe.
func TestElTextoLegalPuedeNombrarLoQueLaLeyNombre(t *testing.T) {
	ps, dirs := paquetesDePrueba(t)
	ps[1].Obligaciones[0].TextoLegal = "Los sistemas se certificaran conforme a la norma " +
		"UNE-EN ISO/IEC 27001, sin perjuicio de lo dispuesto en el articulo anterior."
	if errs := ValidarProsaEntrePaquetes(ps, dirs); len(errs) != 0 {
		t.Fatalf("el texto legal transcrito no se censura: %v", errs)
	}
}

// CONTROL NEGATIVO 3: un marco TRANSCRITO se puede nombrar, porque es citable.
// Es la diferencia que sostiene toda la frontera legal.
func TestUnMarcoTranscritoSiSePuedeNombrar(t *testing.T) {
	ps, dirs := paquetesDePrueba(t)
	ps[0].Clase = Transcrito
	ps[0].LicenciaFuente = BOETRLPI13
	ps[1].Obligaciones[0].Cita = "Se alinea con el RD 311/2022 y con ISO 27001"
	if errs := ValidarProsaEntrePaquetes(ps, dirs); len(errs) != 0 {
		t.Fatalf("si el otro marco es transcrito, nombrarlo es legitimo: %v", errs)
	}
}

// LA FRONTERA DE PALABRA, que es lo que hace la regla usable en castellano. Sin
// ella, "cis" (el directorio de un paquete delegado real) casaria dentro de
// "decision", "precision" y "ejercicio", y la puerta daria un falso positivo por
// linea de prosa.
func TestElNombreDeUnMarcoNoCasaDentroDeOtraPalabra(t *testing.T) {
	ps, dirs := paquetesDePrueba(t)
	ps[0].URN = "urn:cis:benchmarks"
	ps[0].Clase = Delegado
	ps[0].LicenciaFuente = SinLicenciaDeTexto
	dirs[0] = "cis"
	ps[1].Obligaciones[0].Cita = "Esta decision se toma con precision en cada ejercicio, " +
		"y el analisis es exhaustivo."
	if errs := ValidarProsaEntrePaquetes(ps, dirs); len(errs) != 0 {
		t.Fatalf("decision, precision y ejercicio no nombran a CIS: %v", errs)
	}
	// Y la misma palabra suelta SI se caza.
	ps[1].Obligaciones[0].Cita = "El intervalo es el que recomienda CIS para este control."
	if errs := ValidarProsaEntrePaquetes(ps, dirs); len(errs) == 0 {
		t.Fatal("CIS como palabra suelta si es nombrar el marco")
	}
}

// Las tildes no son una via de escape. Sin normalizar, bastaria escribir el
// nombre con un acento de mas para saltarse la puerta.
func TestLasTildesNoSaltanLaPuerta(t *testing.T) {
	ps, dirs := paquetesDePrueba(t)
	ps[1].Obligaciones[0].Cita = "Segun la norma ISO 27001, la revisión periódica es anual."
	if errs := ValidarProsaEntrePaquetes(ps, dirs); len(errs) == 0 {
		t.Fatal("el texto con tildes alrededor no deja de nombrar el marco")
	}
}

// El corpus publicado, entero: ni un paquete nombra un marco cerrado ajeno.
// Vive aqui y no en la raiz porque ValidarProsaEntrePaquetes es de este paquete,
// pero lo que se comprueba de verdad es el corpus real, que lo carga Cargar.
func TestElInformeDeLaPuertaNombraLaRutaExacta(t *testing.T) {
	ps, dirs := paquetesDePrueba(t)
	ps[1].Obligaciones[0].Temporalidad = &Temporalidad{
		Primitiva: "periodica", Cadencia: "P12M",
		Regimen:            RegimenSpec{Computo: "naturales", Cierre: "fin_de_dia"},
		OrigenDelIntervalo: IntervaloPropuesto,
		JustificacionDelIntervalo: "Doce meses porque es lo que exige PCI DSS al conjunto de " +
			"reglas del cortafuegos, y el resto de la industria lo ha adoptado.",
	}
	ps[0].URN = "urn:pcissc:dss:4"
	ps[0].Clase = Referencial
	ps[0].LicenciaFuente = SinLicenciaDeTexto
	dirs[0] = "pci-dss"
	errs := ValidarProsaEntrePaquetes(ps, dirs)
	if len(errs) == 0 {
		t.Fatal("la justificacion apoyada en PCI DSS tiene que caerse")
	}
	// El error dice DONDE, no solo que. Un informe que dice "hay un problema en
	// el paquete" obliga a leer 2000 lineas de JSON para encontrarlo.
	if !strings.Contains(errs[0].Error(), "justificacion_del_intervalo") {
		t.Errorf("el error no dice en que campo esta: %v", errs[0])
	}
}
