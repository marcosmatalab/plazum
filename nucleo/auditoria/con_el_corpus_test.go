package auditoria_test

import (
	"strings"
	"testing"
	"time"

	"github.com/marcosmatalab/plazum/nucleo/auditoria"
	"github.com/marcosmatalab/plazum/nucleo/corpus"
)

// EL ALCANCE SALE DEL CORPUS INSTALADO, y este test lo demuestra a escala real.
//
// LAS UNIDADES SE ELIGEN POR PROPIEDAD, NUNCA POR NOMBRE. El invariante 2
// prohibe nombrar una norma en el codigo, y ademas una lista de identificadores
// escrita al lado del test solo prueba lo que penso su autor: la propiedad se
// aplica sola a la obligacion que alguien escriba manana.
//
// La propiedad aqui es la que de verdad decide que es auditable: una obligacion
// de clase PROCEDIMENTAL es un procedimiento que alguien tiene que ejecutar, y
// eso es lo que una auditoria interna mira. Una documental se comprueba leyendo
// el documento y una tecnica midiendo el sistema, que son otras dos cosas.
func unidadesDelCorpus(t *testing.T) []auditoria.Unidad {
	t.Helper()
	ps, err := corpus.Cargar("../../paquetes")
	if err != nil {
		t.Fatalf("el corpus no carga: %v", err)
	}
	var out []auditoria.Unidad
	for _, p := range ps {
		for _, o := range p.Obligaciones {
			if o.ClaseE2E != "procedimental" {
				continue
			}
			out = append(out, auditoria.Unidad{
				Paquete: p.URN, Version: p.Version, Obligacion: o.ID, Titulo: o.Titulo,
			})
		}
	}
	return out
}

// A ESCALA REAL: se abre un programa sobre el corpus entero y la ley de
// conservacion cuadra.
//
// El numero no se cablea: se mide y se exige un suelo, porque un corpus que se
// vacie dejaria este test pasando sobre cero unidades, que es el verde falso de
// siempre.
func TestUnProgramaSeAbreSobreElCorpusInstaladoYCuadra(t *testing.T) {
	us := unidadesDelCorpus(t)
	if len(us) < 20 {
		t.Fatalf("solo %d unidades procedimentales en el corpus entero: o el corpus se ha "+
			"vaciado, o esta seleccion ha dejado de reconocerlas y este test estaria midiendo "+
			"el vacio", len(us))
	}
	t.Logf("alcance derivado del corpus: %d unidades procedimentales", len(us))

	desde := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	hasta := time.Date(2028, 12, 31, 0, 0, 0, 0, time.UTC)
	p, err := auditoria.Abrir("prog-real",
		auditoria.Ciclo{Nombre: "2026-2028", Desde: desde, Hasta: hasta}, us, auditoria.Arrastre{})
	if err != nil {
		t.Fatalf("no abre sobre el corpus real: %v", err)
	}
	if p.Alcance() != len(us) {
		t.Fatalf("alcance %d y unidades %d: hay claves repetidas en el corpus", p.Alcance(), len(us))
	}

	// Se audita un tercio, se difiere una y el resto se queda sin auditar.
	var primeras []string
	for i, u := range p.Unidades() {
		if i >= len(us)/3 {
			break
		}
		primeras = append(primeras, u.Clave())
	}
	if err := p.Auditar(auditoria.Sesion{ID: "s1", Auditor: "auditor-interno",
		Cuando: time.Date(2027, 6, 1, 0, 0, 0, 0, time.UTC), Unidades: primeras}); err != nil {
		t.Fatal(err)
	}
	ultima := p.Unidades()[len(us)-1].Clave()
	if err := p.Diferir(auditoria.Diferimiento{Unidad: ultima, Quien: "ciso",
		Motivo: "el sistema se retira", Cuando: time.Date(2027, 6, 1, 0, 0, 0, 0, time.UTC)}); err != nil {
		t.Fatal(err)
	}

	c := p.Cuenta()
	suma := 0
	for _, v := range c {
		suma += v
	}
	if suma != p.Alcance() || !p.Cuadra() {
		t.Fatalf("los cubos suman %d y el alcance es %d: %v", suma, p.Alcance(), c)
	}
	if c[auditoria.Auditada] != len(primeras) || c[auditoria.Diferida] != 1 {
		t.Fatalf("cubos: %v", c)
	}
	t.Logf("cubos sobre el corpus real: %v", c)

	// Y EL ARRASTRE SALE CON TODO LO QUE NO SE AUDITO, diferido incluido.
	arr := p.ParaElCicloSiguiente()
	if len(arr.SinAuditar) != c[auditoria.SinAuditar]+c[auditoria.Diferida] {
		t.Fatalf("el arrastre lleva %d y sin auditar mas diferidas son %d",
			len(arr.SinAuditar), c[auditoria.SinAuditar]+c[auditoria.Diferida])
	}
}

// LA FRASE DEL DESCARGO EXISTE Y NO ACUSA.
//
// Una unidad sin auditar NO es una no conformidad: puede estar perfectamente
// implantada y lo unico que falte es que alguien la mire. Es el mismo patron que
// el calendario y la UAR, y se comprueba igual: con la frase, y con que NO
// aparezca la palabra que acusaria.
func TestLoNoAuditadoNoSePresentaComoIncumplimiento(t *testing.T) {
	f := auditoria.LaFraseDeLoNoAuditado
	if !strings.Contains(f, "NO dice") || !strings.Contains(f, "no consta") {
		t.Fatalf("la frase no tiene la forma del patron de la casa: %q", f)
	}
	for _, acusa := range []string{"incumplimiento", "no conformidad", "infraccion"} {
		if strings.Contains(strings.ToLower(f), acusa) &&
			!strings.Contains(strings.ToLower(f), "no dice que estas obligaciones se incumplan") {
			t.Errorf("la frase acusa con %q: %q", acusa, f)
		}
	}
	// Y el vocabulario de cobertura no llama incumplimiento a lo que no se miro.
	for _, c := range auditoria.CoberturasPosibles() {
		s := strings.ToLower(string(c))
		if strings.Contains(s, "incumpl") || strings.Contains(s, "conformidad") {
			t.Errorf("el cubo %q presenta como incumplimiento algo que solo esta sin mirar", c)
		}
	}
}
