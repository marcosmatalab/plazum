package corpus

import "testing"

// LA HERENCIA SILENCIOSA, CON SUS CUATRO FALLOS Y SUS CONTROLES.
//
// Lo que se vigila no es que la fecha sea la correcta (eso no lo puede saber un
// linter: lo contrasta vigencias_test.go contra la instantanea) sino que el
// paquete DIGA de quien es. La copia y la comprobacion producen el mismo JSON,
// y este campo es lo unico que las separa.

func conVigencia(desdePaquete, desdeObl, origen string) *Paquete {
	p := enciendeElReloj(base())
	p.Vigencia = Vigencia{Desde: desdePaquete}
	p.Obligaciones[0].Vigencia = Vigencia{Desde: desdeObl, Origen: origen}
	// El reloj hace falta: sin el, esta puerta no pide nada, y un fixture sin
	// temporalidad dejaria el test verde sin mirar nada.
	p.Obligaciones[0].Temporalidad = &Temporalidad{Primitiva: "periodica", Cadencia: "P24M",
		Regimen:            RegimenSpec{Computo: "naturales", Cierre: "fin_de_dia"},
		OrigenDelIntervalo: IntervaloSueloLegal,
		CitaDelIntervalo:   "RD 311/2022, art. 31.1: AL MENOS CADA DOS ANOS (fixture)"}
	return p
}

// EL VALOR CERO ES ERROR, y las tres formas de la nada se miran por separado:
// la que sale por olvido es la ausencia, y es la peligrosa porque es la que
// heredaba en silencio.
func TestUnaObligacionConRelojSinOrigenDeVigenciaNoCarga(t *testing.T) {
	for _, c := range []struct{ nombre, origen string }{
		{"campo ausente", ""},
		{"solo espacios", "   "},
		{"tabulador", "\t"},
	} {
		t.Run(c.nombre, func(t *testing.T) {
			p := conVigencia("2022-05-05", "2022-05-05", c.origen)
			if !hay(p.Validar(), ErrSinOrigenDeVigencia) {
				t.Fatalf("heredar en silencio tiene que dejar de ser posible: %v", p.Validar())
			}
		})
	}
	// Control negativo: con el campo puesto, carga.
	q := conVigencia("2022-05-05", "2022-05-05", VigenciaHeredada)
	if hay(q.Validar(), ErrSinOrigenDeVigencia) {
		t.Errorf("una obligacion que SI lo declara no puede acusarse: %v", q.Validar())
	}
}

// Y NO SE PIDE DONDE NO HAY RELOJ: una obligacion sin temporalidad tiene fecha
// pero no mueve ningun vencimiento, asi que exigirsela seria ruido.
func TestElOrigenDeVigenciaNoSePideSinReloj(t *testing.T) {
	p := conVigencia("2022-05-05", "2022-05-05", "")
	p.Obligaciones[0].Temporalidad = nil
	if hay(p.Validar(), ErrSinOrigenDeVigencia) {
		t.Errorf("sin reloj la fecha no mueve nada y no se pide: %v", p.Validar())
	}
}

func TestElVocabularioDelOrigenDeVigenciaEsCerrado(t *testing.T) {
	p := conVigencia("2022-05-05", "2022-05-05", "supuesta")
	if !hay(p.Validar(), ErrOrigenDeVigenciaDesconocido) {
		t.Fatalf("solo hay dos valores: %v", p.Validar())
	}
	for _, o := range []string{VigenciaHeredada, VigenciaPropia} {
		d := "2022-05-05"
		if o == VigenciaPropia {
			d = "2018-04-20"
		}
		q := conVigencia("2022-05-05", d, o)
		if hay(q.Validar(), ErrOrigenDeVigenciaDesconocido) {
			t.Errorf("%q es del vocabulario y se rechaza: %v", o, q.Validar())
		}
	}
}

// LO DECLARADO TIENE QUE CUADRAR CON LO ESCRITO, que es lo unico mecanico que
// se puede exigir aqui. Las dos direcciones, porque son dos descuidos distintos.
func TestLoDeclaradoCuadraConLaFechaEscrita(t *testing.T) {
	mal := conVigencia("2022-05-05", "2018-04-20", VigenciaHeredada)
	if !hay(mal.Validar(), ErrHeredadaQueNoCoincide) {
		t.Errorf("dice heredar y trae otra fecha: %v", mal.Validar())
	}
	// La otra direccion es mas sutil y por eso se rechaza igual: «propia» con la
	// misma fecha del paquete puede ser cierto por casualidad, y entonces no se
	// distingue de «copiada», que es justamente lo que este campo separa.
	tambien := conVigencia("2022-05-05", "2022-05-05", VigenciaPropia)
	if !hay(tambien.Validar(), ErrPropiaQueCoincide) {
		t.Errorf("«propia» identica a la del paquete no se distingue de copiada: %v",
			tambien.Validar())
	}
	// Control negativo de las dos: lo coherente carga.
	for _, c := range []struct{ desde, origen string }{
		{"2022-05-05", VigenciaHeredada},
		{"2018-04-20", VigenciaPropia},
	} {
		q := conVigencia("2022-05-05", c.desde, c.origen)
		for _, e := range []error{ErrHeredadaQueNoCoincide, ErrPropiaQueCoincide} {
			if hay(q.Validar(), e) {
				t.Errorf("%s/%s es coherente y se acusa: %v", c.desde, c.origen, q.Validar())
			}
		}
	}
}
