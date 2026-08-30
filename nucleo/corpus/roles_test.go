package corpus

import (
	"errors"
	"strings"
	"testing"
)

// conEscalado devuelve el paquete base con un escalon y su figura declarada, o
// sea el caso BUENO. Todo lo demas de este fichero rompe una cosa de aqui.
func conEscalado() *Paquete {
	p := base()
	p.Obligaciones[0].Escalado = []Escalon{{Tras: "P3D", A: "demo.responsable_de_la_seguridad"}}
	p.Roles = []Rol{{
		ID: "demo.responsable_de_la_seguridad", Titulo: "Responsable de la seguridad",
		Origen: FiguraDeLaNorma,
		Cita:   "Real Decreto 311/2022, art. 13.2, letra c). Verificado el 30-08-2026.",
	}}
	return p
}

// EL CONTROL POSITIVO, y va primero a proposito: sin el, una comprobacion que
// rechazara cualquier figura dejaria en verde todos los tests de abajo.
func TestUnEscalonConSuFiguraDeclaradaPasa(t *testing.T) {
	if errs := conEscalado().Validar(); len(errs) != 0 {
		t.Fatalf("el caso bueno se rechaza: %v", errs)
	}
}

// LA PUERTA. Un aviso a un nombre que la organizacion no ha asignado no llega a
// nadie y no da error: es un escalon que no escala.
func TestUnEscalonHaciaUnaFiguraQueNadieDeclaraNoCarga(t *testing.T) {
	casos := map[string]string{
		"nombre que no existe":    "demo.responsable_de_otra_cosa",
		"el nombre sin prefijo":   "responsable_de_la_seguridad",
		"la figura de OTRA norma": "otronorma.responsable_de_la_seguridad",
	}
	for nombre, destino := range casos {
		t.Run(nombre, func(t *testing.T) {
			p := conEscalado()
			p.Obligaciones[0].Escalado[0].A = destino
			errs := p.Validar()
			if len(errs) == 0 {
				t.Fatalf("un escalon hacia %q pasa el linter", destino)
			}
			if !tieneError(errs, ErrEscalonHaciaNadie) {
				t.Fatalf("el error no es el del escalon hacia nadie: %v", errs)
			}
		})
	}
}

// LA SEGUNDA DIRECCION (invariante 7), que es la que se olvida: una figura
// declarada a la que nadie escala no rompe nada hoy y manana es una pregunta
// mas al cliente cuyo dato no lee nadie. Es el campo huerfano con personas.
func TestUnaFiguraQueNadieUsaNoCarga(t *testing.T) {
	p := conEscalado()
	p.Roles = append(p.Roles, Rol{
		ID: "demo.figura_de_adorno", Titulo: "Figura de adorno", Origen: FiguraPropuesto,
		Justificacion: "Nadie escala a ella, asi que este caso existe para que el linter lo diga.",
	})
	errs := p.Validar()
	if !tieneError(errs, ErrFiguraQueNadieUsa) {
		t.Fatalf("una figura que nadie usa pasa el linter: %v", errs)
	}
}

// Una figura que dice que la define la norma sin articulo que lo diga es una
// afirmacion sin fuente, y de esas viven las equivalencias inventadas entre
// marcos. Las tres formas de cita vaga, no una.
func TestUnaFiguraDeLaNormaSinCitaQueIDENTIFIQUENoCarga(t *testing.T) {
	casos := map[string]string{
		"vacia":           "",
		"demasiado corta": "el ENS",
		"sin numero":      "el Esquema Nacional de Seguridad, articulo del responsable",
	}
	for nombre, cita := range casos {
		t.Run(nombre, func(t *testing.T) {
			p := conEscalado()
			p.Roles[0].Cita = cita
			if !tieneError(p.Validar(), ErrFiguraSinCita) {
				t.Fatalf("la cita %q pasa como figura definida por la norma", cita)
			}
		})
	}
}

// Y la simetrica: una propuesta sin justificacion. El cliente la va a cambiar,
// y para cambiarla con criterio necesita leer de donde salio.
func TestLaFiguraQuePlazumProponeSinJustificacionNoCarga(t *testing.T) {
	for _, just := range []string{"", "porque si", "la propone plazum"} {
		p := conEscalado()
		p.Roles[0].Origen, p.Roles[0].Cita, p.Roles[0].Justificacion = FiguraPropuesto, "", just
		if !tieneError(p.Validar(), ErrFiguraSinJustificar) {
			t.Errorf("la justificacion %q pasa", just)
		}
	}
}

// El origen es vocabulario cerrado de dos, y su valor cero (la cadena vacia) es
// el restrictivo: sin origen no se carga (invariante 8). Sin esto, olvidarse
// del campo colaria una figura sin cita y sin justificacion.
func TestElOrigenDeUnaFiguraEsVocabularioCerradoYSuVacioNoPasa(t *testing.T) {
	for _, origen := range []string{"", "definida_por_la_norma", "sugerida", "PROPUESTO"} {
		p := conEscalado()
		p.Roles[0].Origen = origen
		if !tieneError(p.Validar(), ErrFiguraSinOrigen) {
			t.Errorf("el origen %q pasa", origen)
		}
	}
}

func TestUnaFiguraSinIdOSinTituloNoCarga(t *testing.T) {
	for _, quitar := range []string{"id", "titulo"} {
		p := conEscalado()
		if quitar == "id" {
			p.Roles[0].ID = ""
		} else {
			p.Roles[0].Titulo = ""
		}
		if !tieneError(p.Validar(), ErrFiguraSinIdOTitulo) {
			t.Errorf("una figura sin %s pasa", quitar)
		}
	}
}

func TestDosFigurasConElMismoIdNoCargan(t *testing.T) {
	p := conEscalado()
	p.Roles = append(p.Roles, p.Roles[0])
	if !tieneError(p.Validar(), ErrFiguraDuplicada) {
		t.Fatal("una figura declarada dos veces pasa")
	}
}

func tieneError(errs []error, quiero error) bool {
	for _, e := range errs {
		if errors.Is(e, quiero) {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// El corpus, en las dos direcciones
// ---------------------------------------------------------------------------

// NO SE UNIFICAN FIGURAS ENTRE NORMAS. Que dos paquetes usaran el mismo id
// diria, sin decirlo y sin fuente, que son la misma figura. Es la misma
// afirmacion que una cadencia gemela y necesita lo mismo: una cita.
//
// Los ids van prefijados por paquete igual que los de obligacion, asi que la
// comprobacion es barata; lo que no era barato es acordarse, y por eso hay
// test. El caso que lo trae: nis2-ue escalaba a `responsable_seguridad`, un
// nombre tomado del ENS, para una Directiva que NO define esa figura.
func TestNingunaFiguraSeCompartEntreNormas(t *testing.T) {
	paquetes, err := Cargar("../../paquetes")
	if err != nil {
		t.Fatalf("cargando el corpus: %v", err)
	}
	de := map[string]string{}
	n := 0
	for _, p := range paquetes {
		for _, r := range p.Roles {
			n++
			if otro, ya := de[r.ID]; ya {
				t.Errorf("la figura %s la declaran %s y %s. Que dos normas compartan figura es "+
					"una equivalencia, y una equivalencia necesita fuente", r.ID, otro, p.URN)
			}
			de[r.ID] = p.URN
		}
	}
	if n == 0 {
		t.Fatal("ninguna figura en el corpus: el test no ha comprobado nada")
	}
	if !t.Failed() {
		t.Logf("%d figuras en el corpus, ninguna compartida entre normas", n)
	}
}

// Y el reparto, contado, porque es el dato que hace honesta la pantalla que le
// pregunte al cliente quien ocupa cada figura: unas las nombra la norma y otras
// las propone plazum, y no se presentan igual.
func TestElCorpusDiceDeQuienEsCadaFigura(t *testing.T) {
	paquetes, err := Cargar("../../paquetes")
	if err != nil {
		t.Fatalf("cargando el corpus: %v", err)
	}
	deLaNorma, propuestas := 0, 0
	for _, p := range paquetes {
		for _, r := range p.Roles {
			switch r.Origen {
			case FiguraDeLaNorma:
				deLaNorma++
				// Una figura de la norma se titula con las palabras del
				// articulo, asi que su cita tiene que nombrar uno.
				if !strings.Contains(strings.ToLower(r.Cita), "art") {
					t.Errorf("%s dice que la define la norma y su cita no nombra ningun "+
						"articulo: %q", r.ID, r.Cita)
				}
			case FiguraPropuesto:
				propuestas++
			default:
				t.Errorf("%s tiene un origen que el linter tenia que haber rechazado: %q",
					r.ID, r.Origen)
			}
		}
	}
	if deLaNorma == 0 || propuestas == 0 {
		t.Fatalf("el reparto es %d de la norma y %d propuestas: con una de las dos a cero "+
			"este test no distingue nada", deLaNorma, propuestas)
	}
	if !t.Failed() {
		t.Logf("%d figuras definidas por la norma, %d propuestas por plazum",
			deLaNorma, propuestas)
	}
}

// El `tras` de un escalon se parsea AL CARGAR. Hasta el 30-08-2026 el campo
// llevaba declarado desde el primer dia y no lo leia nadie: el linter miraba
// que no estuviera vacio. Un `P60D_ants` habria salido el dia del incidente.
func TestUnTrasIlegibleNoCarga(t *testing.T) {
	casos := map[string]string{
		"con la palabra mal escrita":               "P60D_ants",
		"en castellano":                            "sesenta dias antes",
		"sin numero":                               "P",
		"indeterminado, que no se puede programar": "indeterminado",
	}
	for nombre, tras := range casos {
		t.Run(nombre, func(t *testing.T) {
			p := conEscalado()
			p.Obligaciones[0].Escalado[0].Tras = tras
			if errs := p.Validar(); len(errs) == 0 {
				t.Fatalf("el `tras` %q pasa el linter", tras)
			}
		})
	}
	// CONTROL POSITIVO: los que el corpus usa de verdad tienen que pasar. Sin
	// esta mitad, un parser que rechazara todo daria el mismo verde arriba.
	for _, tras := range []string{"P60D_antes", "P30D_antes", "PT4H", "P5D", "P1M", "PT12H"} {
		p := conEscalado()
		p.Obligaciones[0].Escalado[0].Tras = tras
		if errs := p.Validar(); len(errs) != 0 {
			t.Errorf("el `tras` %q se rechaza: %v", tras, errs)
		}
	}
}
