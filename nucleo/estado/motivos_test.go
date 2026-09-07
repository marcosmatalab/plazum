package estado

import (
	"sort"
	"strings"
	"testing"
	"time"
)

// TODA RAMA DE Calcular EMITE CLAVE, Y SUS ARGUMENTOS CASAN CON SUS HUECOS.
//
// # El emparejamiento que esto vigila, dicho como pide el invariante 7
//
// Un motivo son DOS COSAS que viajan juntas: una clave y una lista de
// argumentos. **Casan POR POSICION**, porque los `%s` de una plantilla de fmt se
// rellenan en orden, y **la posicion no la firma nadie**. Es el unico
// emparejamiento posicional de esta pieza y por eso lleva puerta propia.
//
// Lo que puede salir mal es barato de cometer y caro de ver:
//
//	un argumento de menos  fmt deja «%!s(MISSING)» dentro del texto que se
//	                       guarda en el expediente, y el catalogo, al no poder
//	                       formatear, devuelve la PLANTILLA SIN RELLENAR, o sea
//	                       que la pantalla ensena «%s» literal a un CISO
//	un argumento de mas    el dato sobrante se pierde sin que nada lo diga
//	una rama sin clave     el motivo vuelve a ser una frase, que es el defecto
//	                       entero que motivos.go existe para cerrar
//
// # Por que se recorren las once ramas y no una muestra
//
// Porque el fallo es POR RAMA: una sola mal emparejada basta para que una celda
// de la pantalla ensene un `%s`. Y porque el conjunto se contrasta al final
// contra `CadenasDelEstado()` en las dos direcciones: una rama nueva sin caso
// aqui deja la puerta corta, y una cadena declarada que ninguna rama emite es
// una explicacion que el producto no da.
func TestTodaRamaDeCalcularEmiteClaveYSusArgumentosCasan(t *testing.T) {
	ahora := time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)
	p := Prueba{ID: "mfa.todos", Control: "op.acc.5", TTL: 24 * time.Hour, SLA: 72 * time.Hour}

	casos := []struct {
		nombre string
		prueba Prueba
		obs    []Observacion
		ctx    Contexto
		quiero Frase
	}{
		{"no aplica", p, nil, Contexto{Ahora: ahora}, mtNoAplica},
		{"exceptuado", p, nil, Contexto{Ahora: ahora, Aplicable: true,
			Excepciones: []Excepcion{{Control: "op.acc.5", Motivo: "migracion de IdP",
				Aprobador: "CISO", Desde: ahora.Add(-time.Hour), Hasta: ahora.Add(240 * time.Hour)}}},
			mtExceptuado},
		{"en despliegue", func() Prueba { q := p; q.Activa = ahora.Add(48 * time.Hour); return q }(),
			nil, Contexto{Ahora: ahora, Aplicable: true}, mtDespliegue},
		{"pass por defecto", func() Prueba { q := p; q.PassPorDef = true; return q }(),
			nil, Contexto{Ahora: ahora, Aplicable: true}, mtPassPorDefecto},
		{"sin observaciones", p, nil, Contexto{Ahora: ahora, Aplicable: true}, mtSinObservaciones},
		{"fallo con el plazo agotado", p,
			[]Observacion{{Prueba: "mfa.todos", Recurso: "u2", Recolectada: ahora.Add(-100 * time.Hour)}},
			Contexto{Ahora: ahora, Aplicable: true}, mtFalloVencido},
		// Fallo DENTRO de plazo y con otra observacion ya caducada: es la rama
		// que dice que no se puede afirmar el estado actual. Hace falta un SLA
		// largo para que el fallo no venza y un TTL corto para que la otra si.
		{"fallo en plazo con una observacion caducada",
			func() Prueba { q := p; q.SLA = 2000 * time.Hour; return q }(),
			[]Observacion{
				{Prueba: "mfa.todos", Recurso: "u2", Recolectada: ahora.Add(-time.Hour)},
				{Prueba: "mfa.todos", Recurso: "u9", Satisfecho: true, Recolectada: ahora.Add(-100 * time.Hour)},
			},
			Contexto{Ahora: ahora, Aplicable: true}, mtFalloYCaducada},
		{"fallo en plazo", p,
			[]Observacion{{Prueba: "mfa.todos", Recurso: "u2", Recolectada: ahora.Add(-time.Hour)}},
			Contexto{Ahora: ahora, Aplicable: true}, mtFalloEnPlazo},
		{"caducada", p,
			[]Observacion{{Prueba: "mfa.todos", Recurso: "u1", Satisfecho: true,
				Recolectada: ahora.Add(-48 * time.Hour)}},
			Contexto{Ahora: ahora, Aplicable: true}, mtCaducada},
		{"error de recoleccion", p,
			[]Observacion{{Prueba: "mfa.todos", Recurso: "u1", ErrorRecol: "429 rate limit",
				Recolectada: ahora}},
			Contexto{Ahora: ahora, Aplicable: true}, mtErrorDeRecoleccion},
		{"todas satisfacen", p,
			[]Observacion{{Prueba: "mfa.todos", Recurso: "u1", Satisfecho: true,
				Recolectada: ahora.Add(-time.Hour)}},
			Contexto{Ahora: ahora, Aplicable: true}, mtTodasSatisfacen},
	}

	vistas := map[string]bool{}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			got := Calcular(c.prueba, c.obs, c.ctx)
			vistas[got.Motivo.Clave] = true
			if got.Motivo.Clave != c.quiero.Clave {
				t.Fatalf("esta rama emite la clave %q y se esperaba %q: el caso no recorre "+
					"la rama que dice recorrer", got.Motivo.Clave, c.quiero.Clave)
			}
			if got.Motivo.Clave == "" {
				t.Fatal("esta rama devuelve un motivo SIN CLAVE, o sea una frase: es el " +
					"defecto que motivos.go existe para cerrar")
			}
			// EL EMPAREJAMIENTO, que es lo caro. fmt deja su propia huella
			// cuando los argumentos no casan con los huecos, y esa huella
			// acaba dentro del expediente.
			if strings.Contains(got.Motivo.Texto, "%!") {
				t.Errorf("los argumentos no casan con los huecos de %q:\n  texto: %q\n"+
					"  args:  %q\n  Esto viaja al expediente tal cual, y en la pantalla el "+
					"catalogo devuelve la plantilla SIN rellenar, o sea un %%s literal "+
					"delante de una persona", c.quiero.Clave, got.Motivo.Texto, got.Motivo.Args)
			}
			// Y LA OTRA DIRECCION DEL MISMO EMPAREJAMIENTO: fmt no se queja de
			// un hueco que nadie escribio, asi que un `%s` que sobreviva al
			// formateo es un argumento que falta.
			if strings.Contains(got.Motivo.Texto, "%s") {
				t.Errorf("el texto resuelto de %q conserva un hueco sin rellenar: %q",
					c.quiero.Clave, got.Motivo.Texto)
			}
			if n := strings.Count(c.quiero.Texto, "%s"); n != len(got.Motivo.Args) {
				t.Errorf("%q tiene %d hueco(s) y la rama pasa %d argumento(s): %q",
					c.quiero.Clave, n, len(got.Motivo.Args), got.Motivo.Args)
			}
		})
	}

	// LAS DOS DIRECCIONES CONTRA CadenasDelEstado(). Sin esto, una rama nueva
	// entraria sin caso y sin que nada lo dijera.
	declaradas := map[string]bool{}
	for _, f := range CadenasDelEstado() {
		declaradas[f.Clave] = true
		if !vistas[f.Clave] {
			t.Errorf("la cadena %q se declara y NINGUNA rama de Calcular la emite en este "+
				"recorrido. O sobra en CadenasDelEstado(), o falta el caso que la alcanza",
				f.Clave)
		}
	}
	var sinDeclarar []string
	for c := range vistas {
		if !declaradas[c] {
			sinDeclarar = append(sinDeclarar, c)
		}
	}
	sort.Strings(sinDeclarar)
	if len(sinDeclarar) > 0 {
		t.Errorf("estas ramas emiten claves que CadenasDelEstado() no declara: %v.\n"+
			"  El catalogo no las va a tener, asi que la pantalla las pintaria en crudo",
			sinDeclarar)
	}
}

// TestElContrasteDeHuecosDeLosMotivosSabePonerseRojo es el control negativo del
// emparejamiento de arriba.
//
// La afirmacion cara —«los argumentos casan con los huecos»— nacio VERDE sobre
// las once ramas reales, y una comprobacion que nunca se ha visto fallar no
// distingue vigilar de acompañar. Aqui se le pone delante lo que fmt hace
// cuando NO casan, que es lo unico en lo que la puerta se apoya.
func TestElContrasteDeHuecosDeLosMotivosSabePonerseRojo(t *testing.T) {
	// Un argumento de menos: fmt escribe su huella.
	corto := porque(mtFalloEnPlazo, "1")
	if !strings.Contains(corto.Texto, "%!") {
		t.Errorf("con un argumento de menos, fmt no deja huella en %q: entonces la puerta "+
			"de arriba no se apoya en nada", corto.Texto)
	}
	// Un argumento de mas: tambien.
	largo := porque(mtCaducada, "u1", "2026-09-13", "sobra")
	if !strings.Contains(largo.Texto, "%!") {
		t.Errorf("con un argumento de mas, fmt no deja huella en %q", largo.Texto)
	}
	// Y el positivo, para que el detector no sea uno que siempre dice que si.
	bien := porque(mtCaducada, "u1", "2026-09-13")
	if strings.Contains(bien.Texto, "%!") || strings.Contains(bien.Texto, "%s") {
		t.Errorf("el caso correcto se declara roto: %q", bien.Texto)
	}
	// Y sin argumentos no se formatea, asi que un motivo sin huecos no puede
	// romperse por aqui.
	if a := porque(mtTodasSatisfacen).Args; a != nil {
		t.Errorf("un motivo sin huecos trae argumentos: %q", a)
	}
}
