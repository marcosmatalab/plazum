package corpus

import (
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// La gramatica se lee, y siempre igual
// ---------------------------------------------------------------------------

func TestLaGramaticaDelOrigenSeLee(t *testing.T) {
	casos := []struct {
		origen    string
		clase     ClaseOrigen
		patron    string
		sufijo    string
		derivable bool
	}{
		{"entidad:organizacion.sector", DeEntidad, "organizacion", "sector", true},
		{"obligacion:demo.auditoria", DeObligacion, "demo.auditoria", "", false},
		{"obligacion:demo.auditoria.ultimo_hito", DeObligacion, "demo.auditoria", "ultimo_hito", true},
		{"obligacion:demo.auditoria.primer_hito", DeObligacion, "demo.auditoria", "primer_hito", true},
		{"obligacion:demo.a.*.aplicable", DeObligacion, "demo.a.*", "aplicable", true},
		{"obligacion:demo.a.*.exclusion", DeObligacion, "demo.a.*", "exclusion", true},
		{"obligacion:demo.informe.hallazgos", DeObligacion, "demo.informe", "hallazgos", false},
		{"obligacion:demo.10.2.estado", DeObligacion, "demo.10.2", "estado", false},
		{"obligacion:demo.politica.campo.a", DeObligacion, "demo.politica", "campo.a", false},
		{"incidente:primer_conocimiento", DeIncidente, "primer_conocimiento", "", true},
		{"incidente:id", DeIncidente, "id", "", true},
	}
	for _, c := range casos {
		o, err := ParseOrigen(c.origen)
		if err != nil {
			t.Errorf("%s: %v", c.origen, err)
			continue
		}
		if o.Clase != c.clase || o.Patron != c.patron || o.Sufijo != c.sufijo {
			t.Errorf("%s: sale (%v, %q, %q) y tenia que salir (%v, %q, %q)",
				c.origen, o.Clase, o.Patron, o.Sufijo, c.clase, c.patron, c.sufijo)
		}
		// LA MITAD QUE HACE UTIL AL TIPO: si lo deriva plazum o lo aporta una
		// persona. Sin esto, un hueco que nadie va a rellenar tiene el mismo
		// aspecto que un dato que sale solo.
		if o.Derivable != c.derivable {
			t.Errorf("%s: derivable=%v y tenia que ser %v", c.origen, o.Derivable, c.derivable)
		}
		if o.Que == "" {
			t.Errorf("%s: sin descripcion, asi que la pantalla que pida el dato que falta "+
				"no tiene nada que decir", c.origen)
		}
	}
}

// Lo que la gramatica RECHAZA, que es la mitad que sirve de algo.
func TestLoQueNoEsUnOrigenSeRechazaConSuArreglo(t *testing.T) {
	casos := map[string]string{
		"organizacion.sector":      "le falta el prefijo",      // el caso real de demo-empresa
		"":                         "le falta el prefijo",      //
		"activo:x.y":               "no es una clase",          // prefijo inventado
		"obligacion:":              "no dice a QUE",            //
		"entidad:organizacion":     "la entidad y el atributo", // sin atributo
		"incidente:nombre_del_ceo": "solo sabe contestar",      // fuera del vocabulario cerrado
		"incidente:":               "no dice a QUE",            //
	}
	for origen, trozo := range casos {
		o, err := ParseOrigen(origen)
		if err == nil {
			t.Errorf("%q se acepta como origen y sale %+v", origen, o)
			continue
		}
		if !strings.Contains(err.Error(), trozo) {
			t.Errorf("%q: el error no dice %q, dice %q", origen, trozo, err)
		}
	}
}

// Recorrer un mapa de Go da un orden distinto en cada ejecucion. Una gramatica
// que dependa de eso deja de ser una gramatica, asi que se comprueba que la
// respuesta no cambia: es la misma razon por la que el empate de
// clasificaciones se declara en vez de resolverse a cara o cruz.
func TestLaLecturaDeUnOrigenEsDeterminista(t *testing.T) {
	const origen = "obligacion:demo.plan.evidencia"
	primero, err := ParseOrigen(origen)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 500; i++ {
		o, err := ParseOrigen(origen)
		if err != nil {
			t.Fatal(err)
		}
		if o != primero {
			t.Fatalf("la vuelta %d da %+v y la primera dio %+v", i, o, primero)
		}
	}
}

// ---------------------------------------------------------------------------
// La puerta: un origen que apunta a nada no pasa
// ---------------------------------------------------------------------------

// Es la subfamilia "alcanzabilidad, no existencia" cerrada para plantillas. El
// linter comprobaba que el origen no estuviera vacio, y `iso27001.anexo_a.*`
// tampoco lo estaba: no casaba con ninguna de las 93 obligaciones del anexo,
// que se llaman `iso27001.a.5.1`. El entregable habria salido con dos huecos y
// sin decir que los tenia.
func TestUnOrigenQueApuntaANadaNoPasaElLinter(t *testing.T) {
	casos := []struct {
		nombre string
		origen string
		dice   string
	}{
		{"obligacion que no existe", "obligacion:demo.que_no_existe",
			"ninguna obligacion que case"},
		{"glob que no casa con nada", "obligacion:demo.anexo_x.*.aplicable",
			"ninguna obligacion que case"},
		{"entidad no declarada", "entidad:organizacion.sector",
			"que el paquete no declara"},
		{"atributo no declarado", "entidad:sistema.color_favorito",
			"esa entidad no lo declara"},
		{"sin prefijo", "sistema.categoria", "le falta el prefijo"},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			p := base()
			p.Plantillas[0].Campos = append(p.Plantillas[0].Campos,
				CampoPlantilla{Nombre: "campo_nuevo", Origen: c.origen})
			errs := p.Validar()
			if len(errs) == 0 {
				t.Fatalf("el origen %q pasa el linter", c.origen)
			}
			junto := ""
			for _, e := range errs {
				junto += e.Error() + "\n"
			}
			if !strings.Contains(junto, c.dice) {
				t.Fatalf("el error no dice %q:\n%s", c.dice, junto)
			}
		})
	}
}

// El control negativo de la puerta: el paquete base, con sus origenes buenos,
// TIENE que pasar. Sin esto, una comprobacion que rechazara todo se leeria
// igual de verde en el test de arriba.
func TestLosOrigenesDelPaqueteBasePasan(t *testing.T) {
	p := base()
	// Uno de cada clase que el paquete puede resolver.
	p.Plantillas[0].Campos = append(p.Plantillas[0].Campos,
		CampoPlantilla{Nombre: "cumplimiento", Origen: "obligacion:demo.auditoria_bienal"},
		CampoPlantilla{Nombre: "aplicable", Origen: "obligacion:demo.*.aplicable"},
	)
	if errs := p.Validar(); len(errs) != 0 {
		t.Fatalf("origenes buenos rechazados: %v", errs)
	}
}
