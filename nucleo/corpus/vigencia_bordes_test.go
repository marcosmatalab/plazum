package corpus

import "testing"

// FinDeVigencia: el booleano, comprobado directamente.
//
// POR QUE ESTE TEST EXISTE Y NO BASTA CON EL DEL CALENDARIO. Se escribio un test
// en nucleo/pantalla que decia guardar esto ("una vigencia abierta no cesa
// nunca") y se le paso la mutacion por encima: haciendo que FinDeVigencia
// devolviera `true` con el cero de time.Time, el test SEGUIA VERDE. El motivo es
// que quien llama comprueba ademas `fin.After(ahora)`, y el ano 1 no esta
// despues de hoy, asi que el cero se caia igual por la otra condicion.
//
// O sea: el booleano NO era load-bearing para el unico caller que hay hoy, y el
// test que decia protegerlo protegia en realidad otra cosa. Eso es una guarda
// que no guardaba, de la familia de siempre, y con el agravante de que su
// comentario afirmaba lo contrario.
//
// Se arregla comprobando el contrato DONDE SE DECLARA. Lo que el booleano
// protege es al proximo caller: uno que pregunte solo "acaba antes del final de
// la ventana?" sin comprobar tambien "acaba despues de hoy?" tratara TODA
// vigencia abierta como si cesara, que es la mayoria del corpus. El invariante 8
// dice que la forma que sale por olvidarse tiene que ser la que no compila, y
// por eso la firma devuelve (time.Time, bool, error) y no un cero centinela.
func TestFinDeVigenciaDistingueLaAbiertaDeLaQueAcaba(t *testing.T) {
	p := base() // vigencia del paquete: desde 2022-05-05, abierta

	t.Run("abierta por arriba: no hay fin", func(t *testing.T) {
		o := p.Obligaciones[0]
		o.Vigencia = Vigencia{Desde: "2022-05-05"} // sin `hasta`
		fin, hayFin, err := p.FinDeVigencia(o)
		if err != nil {
			t.Fatalf("no deberia fallar: %v", err)
		}
		if hayFin {
			t.Fatalf("una vigencia sin `hasta` NO acaba, y aqui dice que acaba el %s. "+
				"Devolver el cero de time.Time con hayFin=true es el valor cero permisivo "+
				"del invariante 8: quien pregunte solo `acaba antes del final de la ventana?` "+
				"tratara toda obligacion abierta como si cesara", fin.Format("2006-01-02"))
		}
	})

	t.Run("con hasta: hay fin y es esa fecha", func(t *testing.T) {
		o := p.Obligaciones[0]
		o.Vigencia = Vigencia{Desde: "2022-05-05", Hasta: "2027-03-15"}
		fin, hayFin, err := p.FinDeVigencia(o)
		if err != nil {
			t.Fatalf("no deberia fallar: %v", err)
		}
		if !hayFin {
			t.Fatal("una vigencia con `hasta` acaba")
		}
		if got := fin.Format("2006-01-02"); got != "2027-03-15" {
			t.Errorf("acaba el %s y se declaro 2027-03-15", got)
		}
	})

	// Y el cruce con la vigencia del PAQUETE, que es lo que hace que esto no sea
	// una lectura de un campo: una obligacion no puede seguir obligando despues
	// de que su norma deje de estar en vigor, aunque su propio `hasta` diga otra
	// cosa. Manda la interseccion.
	t.Run("la norma acaba antes que la obligacion: manda la norma", func(t *testing.T) {
		q := base()
		q.Vigencia = Vigencia{Desde: "2022-05-05", Hasta: "2026-01-01"}
		o := q.Obligaciones[0]
		o.Vigencia = Vigencia{Desde: "2022-05-05", Hasta: "2030-01-01"}
		fin, hayFin, err := q.FinDeVigencia(o)
		if err != nil {
			t.Fatalf("no deberia fallar: %v", err)
		}
		if !hayFin {
			t.Fatal("si la norma acaba, la obligacion acaba")
		}
		if got := fin.Format("2006-01-02"); got != "2026-01-01" {
			t.Errorf("acaba el %s y la NORMA acaba el 2026-01-01: manda la interseccion, "+
				"no lo que declare la obligacion", got)
		}
	})
}
