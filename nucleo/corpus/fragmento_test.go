package corpus

import (
	"errors"
	"strings"
	"testing"
)

// EL FRAGMENTO DE ARTICULO, CON SUS TRES FALLOS Y SU CONTROL POSITIVO.

func conFragmento(tipo TipoIdentificador, valor, fragmento string) *Paquete {
	p := base()
	p.Identificador = Identificador{Tipo: tipo, Valor: valor}
	p.Obligaciones[0].Fragmento = fragmento
	return p
}

func fragmentosDe(p *Paquete) []error {
	var out []error
	p.validarFragmentos(func(err error) { out = append(out, err) })
	return out
}

// CONTROL POSITIVO EN LOS DOS ESQUEMAS QUE DIRECCIONAN FRAGMENTOS, y el enlace
// derivado, que es para lo que existe el campo.
func TestUnAnclaBienEscritaDerivaSuEnlaceProfundo(t *testing.T) {
	casos := []struct {
		nombre    string
		tipo      TipoIdentificador
		valor     string
		fragmento string
		acaba     string
	}{
		{"ELI de la Union, un articulo", ELIUE, "reg/2024/2847/oj", "art_14", "#art_14"},
		{"ELI de la Union, un anexo en romano", ELIUE, "reg/2024/2847/oj", "anx_VIII", "#anx_VIII"},
		// LAS CUATRO FORMAS DEL BOE, VERIFICADAS CONTRA EL DOCUMENTO el
		// 11-09-2026 y no supuestas. El BOE numera los articulos del 1 al 9 con
		// el digito y del 10 en adelante con la decena mas el orden dentro de
		// ella: el art. 22 es `a2-4` y el art. 65 es `a6-7`. Las disposiciones
		// van con su clase, sola o con orden.
		//
		// ESTE BLOQUE DECIA `da3` Y ERA UNA FORMA INVENTADA. El BOE escribe `da`
		// y `da-2`; `da3` no existe en ningun documento. Un control positivo que
		// da por bueno algo que la fuente no escribe es la forma mas cara de este
		// fallo: bendice el error en vez de cazarlo.
		{"ELI del BOE, un articulo de un digito", ELIBOE, "es/l/2023/02/20/2/con", "a9", "#a9"},
		{"ELI del BOE, un articulo por decenas", ELIBOE, "es/l/2023/02/20/2/con", "a2-4", "#a2-4"},
		{"ELI del BOE, una disposicion adicional", ELIBOE, "es/l/2023/02/20/2/con", "da", "#da"},
		{"ELI del BOE, la segunda disposicion final", ELIBOE, "es/l/2023/02/20/2/con", "df-2", "#df-2"},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			p := conFragmento(c.tipo, c.valor, c.fragmento)
			if errs := fragmentosDe(p); len(errs) != 0 {
				t.Fatalf("un fragmento bien escrita tiene que pasar: %v", errs)
			}
			enlace := p.EnlaceDeLaObligacion(p.Obligaciones[0])
			if !strings.HasSuffix(enlace, c.acaba) {
				t.Errorf("el enlace derivado es %q y tenia que acabar en %q", enlace, c.acaba)
			}
			// Y EL ANFITRION NO SE MUEVE: es la misma garantia que el enlace del
			// documento, un piso mas abajo.
			if !strings.HasPrefix(enlace, p.Enlace()) {
				t.Errorf("el enlace profundo %q no cuelga del enlace del documento %q",
					enlace, p.Enlace())
			}
		})
	}
}

// SIN FRAGMENTO, EL ENLACE DEL DOCUMENTO Y NO VACIO.
//
// Es una decision de producto y no un detalle: llevar al lector a la norma sin el
// salto es peor que el salto y MUCHO mejor que no llevarlo. Devolver vacio dejaria
// sin enlace a 490 de las 563 obligaciones del corpus de hoy.
func TestSinAnclaSeDevuelveElEnlaceDelDocumento(t *testing.T) {
	p := conFragmento(ELIUE, "reg/2024/2847/oj", "")
	if got := p.EnlaceDeLaObligacion(p.Obligaciones[0]); got != p.Enlace() {
		t.Errorf("sin fragmento el enlace es el del documento; salio %q y esperaba %q",
			got, p.Enlace())
	}
	if p.Enlace() == "" {
		t.Fatal("el fixture no tiene enlace de documento: este test estaria comparando dos vacios")
	}
}

// LOS TRES FALLOS.
func TestUnAnclaQueNoLlevaANingunSitioNoPasa(t *testing.T) {
	casos := []struct {
		nombre string
		monta  func() *Paquete
		quiero error
	}{
		{"sin identificador de fuente: el fragmento no cuelga de nada",
			func() *Paquete {
				p := conFragmento(SinIdentificador, "https://ejemplo.invalid/norma", "art_14")
				p.Identificador.Motivo = "el editor no publica identificador estable"
				return p
			}, ErrFragmentoSinIdentificador},
		{"en un esquema que no direcciona fragmentos",
			func() *Paquete {
				p := conFragmento(NormaISO, "27001", "art_14")
				p.Identificador.Registro = "27001"
				return p
			}, ErrFragmentoConEsquemaQueNoLosDirecciona},
		{"con la forma del OTRO esquema, que es el fallo que no se ve",
			func() *Paquete { return conFragmento(ELIUE, "reg/2024/2847/oj", "a14") },
			ErrFragmentoMalFormado},
		{"con la forma del otro, al reves",
			func() *Paquete {
				return conFragmento(ELIBOE, "es/l/2023/02/20/2/con", "art_9")
			}, ErrFragmentoMalFormado},
		{"con algo que no es un fragmento",
			func() *Paquete {
				return conFragmento(ELIUE, "reg/2024/2847/oj", "el articulo catorce")
			}, ErrFragmentoMalFormado},
		// EL CASO QUE LA FORMA ANTERIOR DABA POR BUENO, y es el que dejo once
		// enlaces rotos en el corpus. `a22` tiene una pinta impecable y NO EXISTE
		// en ningun documento del BOE: el articulo 22 es `a2-4`. La forma vieja
		// (`^(a|da|dt|df|dd)[0-9]+$`) lo aceptaba, y ademas era incapaz de
		// expresar el ancla correcta de cualquier articulo del 10 en adelante.
		{"un ancla del BOE con el numero entero del articulo, que el BOE no escribe asi",
			func() *Paquete {
				return conFragmento(ELIBOE, "es/l/2023/02/20/2/con", "a22")
			}, ErrFragmentoMalFormado},
		{"una disposicion con numero pegado, que tampoco es la forma del BOE",
			func() *Paquete {
				return conFragmento(ELIBOE, "es/l/2023/02/20/2/con", "da3")
			}, ErrFragmentoMalFormado},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			errs := fragmentosDe(c.monta())
			hay := false
			for _, e := range errs {
				if errors.Is(e, c.quiero) {
					hay = true
				}
			}
			if !hay {
				t.Errorf("esperaba %v y salio: %v", c.quiero, errs)
			}
		})
	}
}

// Y EL CASO QUE EL CORPUS REAL CORRIGIO, con su nombre.
//
// La primera version de la forma del ELI de la Union pedia digitos detras del
// prefijo. Al subir los fragmentos que ya estaban pegados en las citas se
// quedaron fuera SEIS del CRA, todos anexos en numeracion romana. No lo encontro
// una mutacion: lo encontro el dato.
func TestElAnexoEnRomanoEsUnAnclaValidaDelEliDeLaUnion(t *testing.T) {
	for _, a := range []string{"anx_I", "anx_VIII", "anx_XV"} {
		p := conFragmento(ELIUE, "reg/2024/2847/oj", a)
		if errs := fragmentosDe(p); len(errs) != 0 {
			t.Errorf("%q es un anexo del ELI de la Union y tiene que pasar: %v", a, errs)
		}
	}
	// Y el control por el otro lado: un anexo con digitos NO es la forma que usa
	// el Diario, y admitirlo seria ensanchar hasta que la forma no valide nada.
	p := conFragmento(ELIUE, "reg/2024/2847/oj", "anx_8")
	if errs := fragmentosDe(p); len(errs) == 0 {
		t.Error("«anx_8» no es la forma del ELI de la Union: si esto pasa, la validacion " +
			"por forma ha dejado de distinguir y solo queda el prefijo")
	}
}
