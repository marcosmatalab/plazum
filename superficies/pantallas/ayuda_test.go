package pantallas

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"sync"
	"testing"

	"plazum/nucleo/corpus"
)

// Utillaje de los tests de esta superficie.
//
// Los paquetes de corpus de aqui son SINTETICOS (urn:demo:...). No pueden ser
// normas reales: el invariante 2 del proyecto prohibe identificadores de norma
// en el codigo, ficheros de test incluidos.

// catalogo es un doble de puertos.Catalogo que ademas APUNTA que claves le han
// pedido. Lo de apuntar no es un adorno: es lo que permite comprobar en las dos
// direcciones que ClavesDeCatalogo() no se queda corta (una clave que la
// interfaz pide y no esta en la lista sale como clave cruda en pantalla) ni le
// sobra nada (una clave de la lista que nadie pide obliga a traducir texto que
// no se ve).
type catalogo struct {
	mu      sync.Mutex
	pedidas map[string]int
	// textos, si esta puesto, gana sobre el texto generado. Sirve para meter
	// texto hostil de catalogo y comprobar que se escapa.
	textos map[string]string
	// idiomas declarados; el primero es el de por defecto.
	idiomas []string
}

func nuevoCatalogo() *catalogo {
	return &catalogo{pedidas: map[string]int{}, idiomas: []string{"es", "en"}}
}

func (c *catalogo) Traducir(idioma, clave string, args ...any) string {
	c.mu.Lock()
	c.pedidas[clave]++
	texto, hay := c.textos[clave]
	c.mu.Unlock()
	if hay {
		if len(args) > 0 && strings.Contains(texto, "%") {
			return fmt.Sprintf(texto, args...)
		}
		return texto
	}
	// Texto generado, y con el idioma dentro para poder comprobar que se
	// eligio el catalogo correcto. Los argumentos se pegan detras para que un
	// contador que no llega se note.
	s := "[" + idioma + ":" + clave + "]"
	for _, a := range args {
		s += fmt.Sprintf(" %v", a)
	}
	return s
}

func (c *catalogo) Idiomas() []string         { return c.idiomas }
func (c *catalogo) Faltantes(string) []string { return nil }
func (c *catalogo) vistas() map[string]int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return copiarCuentas(c.pedidas)
}
func (c *catalogo) olvidar() { c.mu.Lock(); c.pedidas = map[string]int{}; c.mu.Unlock() }
func copiarCuentas(m map[string]int) map[string]int {
	out := make(map[string]int, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// rotulo devuelve el texto que el doble de catalogo genera para una clave, para
// poder buscarlo en el HTML sin repetir el formato en cada test.
func rotulo(idioma, clave string) string { return "[" + idioma + ":" + clave + "]" }

// paqueteAlfa es el paquete de corpus base: dos preguntas con distinto poder de
// desbloqueo, tres obligaciones (una condicionada a dos preguntas, otra a una,
// otra sin condiciones) y una plantilla que piden dos obligaciones.
func paqueteAlfa() *corpus.Paquete {
	return &corpus.Paquete{
		URN: "urn:demo:alfa", Version: "2026.1", Clase: corpus.Propio,
		Licencia: "Apache-2.0",
		// Un identificador DERIVABLE a proposito, y no la valvula de escape:
		// con la valvula, el identificador y el enlace son la misma cadena, asi
		// que la puerta que comprueba que el pie ensena el enlace DERIVADO no
		// sabria distinguir "lo deriva" de "lo copia". Con este, si.
		Identificador:  corpus.Identificador{Tipo: corpus.ELIUE, Valor: "reg/9999/1/oj"},
		LicenciaFuente: corpus.DelProyecto,
		Atribucion:     "Paquete sintetico de demostracion. Sin tercero con derechos.",
		Vigencia:       corpus.Vigencia{Desde: "2026-01-01"},
		Entidades: []corpus.TipoEntidad{{
			Nombre: "sistema", Descripcion: "un sistema del sujeto obligado",
			Atributos: []corpus.Atributo{
				{Nombre: "categoria", Tipo: corpus.Enumerado,
					Valores: []string{"BAJA", "MEDIA", "ALTA"}, Obligado: true,
					Ayuda: "sale del maximo de cada dimension",
					Cita:  "demo alfa art. 3"},
				{Nombre: "nombre", Tipo: corpus.Texto, Obligado: true,
					Cita: "demo alfa art. 2"},
			},
		}},
		Preguntas: []corpus.Pregunta{
			{ID: "alfa.q.categoria", Texto: "Que categoria tiene el sistema",
				Cita: "demo alfa art. 3", Entidad: "sistema", Atributo: "categoria",
				Ayuda:      "si no lo sabes, empieza por el nivel de cada dimension",
				Desbloquea: []string{"alfa.o.auditoria", "alfa.o.copias"}},
			{ID: "alfa.q.nombre", Texto: "Como se llama el sistema",
				Cita: "demo alfa art. 2", Entidad: "sistema", Atributo: "nombre",
				Desbloquea: []string{"alfa.o.inventario"}},
		},
		Obligaciones: []corpus.Obligacion{
			{ID: "alfa.o.auditoria", Articulo: "31", Cita: "demo alfa art. 31",
				Vigencia: corpus.Vigencia{Desde: "2026-01-01"}, Entregable: "alfa.pl.informe",
				ClaseE2E: "procedimental", Preguntas: []string{"alfa.q.categoria"},
				Temporalidad: &corpus.Temporalidad{Primitiva: "periodica", Cadencia: "P24M",
					Regimen: corpus.RegimenSpec{Computo: "naturales"}}},
			{ID: "alfa.o.copias", Articulo: "mp.info.6", Cita: "demo alfa anexo II mp.info.6",
				Vigencia: corpus.Vigencia{Desde: "2026-01-01"}, ClaseE2E: "observable",
				Preguntas: []string{"alfa.q.categoria", "alfa.q.nombre"}},
			{ID: "alfa.o.inventario", Articulo: "2", Cita: "demo alfa art. 2",
				Vigencia: corpus.Vigencia{Desde: "2026-01-01"}, Entregable: "alfa.pl.informe",
				ClaseE2E: "documental", Preguntas: []string{"alfa.q.nombre"}},
		},
		Plantillas: []corpus.Plantilla{{
			ID: "alfa.pl.informe", Titulo: "Informe de conformidad",
			Cita:   "demo alfa art. 31",
			Campos: []corpus.CampoPlantilla{{Nombre: "categoria", Origen: "sistema.categoria"}},
		}},
	}
}

// paqueteBeta anade una obligacion SIN preguntas (aplica siempre, con reloj de
// plazo) y una plantilla huerfana que ninguna obligacion pide.
func paqueteBeta() *corpus.Paquete {
	return &corpus.Paquete{
		URN: "urn:demo:beta", Version: "2026.1", Clase: corpus.Propio,
		Licencia: "Apache-2.0",
		Identificador: corpus.Identificador{Tipo: corpus.SinIdentificador,
			Valor: "https://ejemplo.invalid/demo/beta", Motivo: "paquete sintetico"},
		LicenciaFuente: corpus.DelProyecto,
		Atribucion:     "Segundo paquete sintetico. Reutilizacion libre citando la fuente.",
		Vigencia:       corpus.Vigencia{Desde: "2026-03-01"},
		Entidades: []corpus.TipoEntidad{{
			Nombre: "tratamiento", Descripcion: "un tratamiento de datos personales",
			Atributos: []corpus.Atributo{{Nombre: "riesgo_alto", Tipo: corpus.Booleano,
				Obligado: true, Cita: "demo beta art. 35"}},
		}},
		Preguntas: []corpus.Pregunta{{
			ID: "beta.q.riesgo", Texto: "Hay algun tratamiento de riesgo alto",
			Cita: "demo beta art. 35", Entidad: "tratamiento", Atributo: "riesgo_alto",
			Desbloquea: []string{"beta.o.evaluacion"}}},
		Obligaciones: []corpus.Obligacion{
			{ID: "beta.o.notificacion", Articulo: "33", Cita: "demo beta art. 33",
				Vigencia: corpus.Vigencia{Desde: "2026-03-01"}, ClaseE2E: "notificatoria",
				Temporalidad: &corpus.Temporalidad{Primitiva: "plazo", Limite: "PT72H",
					Regimen: corpus.RegimenSpec{Computo: "naturales"}}},
			{ID: "beta.o.evaluacion", Articulo: "35", Cita: "demo beta art. 35",
				Vigencia: corpus.Vigencia{Desde: "2026-03-01"}, ClaseE2E: "documental",
				Preguntas: []string{"beta.q.riesgo"}},
		},
		Plantillas: []corpus.Plantilla{{
			ID: "beta.pl.huerfana", Titulo: "Registro que nadie pide",
			Cita: "demo beta art. 30"}},
	}
}

func corpusDemo() []*corpus.Paquete { return []*corpus.Paquete{paqueteAlfa(), paqueteBeta()} }

// superficie construye una superficie de pruebas con su catalogo.
func superficie(t *testing.T, ps []*corpus.Paquete, opts ...func(*Opciones)) (*Superficie, *catalogo) {
	t.Helper()
	cat := nuevoCatalogo()
	o := Opciones{Paquetes: ps, Catalogo: cat}
	for _, f := range opts {
		f(&o)
	}
	if o.Catalogo == nil {
		o.Catalogo = cat
	}
	s, err := Nuevo(o)
	if err != nil {
		t.Fatalf("construir la superficie: %v", err)
	}
	return s, cat
}

// pedir hace una peticion GET y devuelve la respuesta y el cuerpo.
func pedir(t *testing.T, s *Superficie, destino string) (*httptest.ResponseRecorder, string) {
	t.Helper()
	r := httptest.NewRequest(http.MethodGet, destino, nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	return w, w.Body.String()
}

// exige comprueba que el cuerpo contiene todos los trozos.
func exige(t *testing.T, cuerpo string, trozos ...string) {
	t.Helper()
	for _, x := range trozos {
		if !strings.Contains(cuerpo, x) {
			t.Errorf("la pagina no contiene %q", x)
		}
	}
}

// prohibe comprueba que el cuerpo no contiene ninguno de los trozos.
func prohibe(t *testing.T, cuerpo string, trozos ...string) {
	t.Helper()
	for _, x := range trozos {
		if strings.Contains(cuerpo, x) {
			t.Errorf("la pagina contiene %q y no debia", x)
		}
	}
}

// claves devuelve las claves apuntadas por el catalogo, ordenadas.
func claves(m map[string]int) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
