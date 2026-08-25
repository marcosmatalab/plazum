package main

import (
	"errors"
	"strings"
	"testing"
)

// La raiz que se sella es lo unico que importa aqui: un sello sobre una raiz
// equivocada verifica perfectamente y no cubre NADA de lo que dice cubrir.
//
// De donde sale la bandera -raiz. Cuando cambia el contenido de la cadena y no
// solo el sello, hay huevo y gallina: generardemo se niega a escribir un
// expediente que no verifica, y no verifica sin un sello que cubra su raiz
// nueva, que solo existe despues de construirlo. El error de generardemo dice
// cual es la raiz que esperaba, y se le pasa a esta herramienta.
//
// Ese camino tiene un peligro que este test cierra: la raiz llega COPIADA A
// MANO de un mensaje de error, y copiar a mano de un mensaje de error se hace
// mal. Media raiz, un espacio delante, un caracter de mas.
func TestLaRaizQueSeSellaSeComprueba(t *testing.T) {
	buena := strings.Repeat("ab", 32) // 32 bytes en hexadecimal

	raiz, hexa, err := raizASellar(buena)
	if err != nil {
		t.Fatalf("una raiz de 32 bytes en hexadecimal tiene que valer: %v", err)
	}
	if len(raiz) != LongitudRaiz || hexa != buena {
		t.Fatalf("la raiz buena no ha llegado entera: %d bytes, %q", len(raiz), hexa)
	}

	// Con espacios alrededor, que es como sale de copiar de una terminal.
	if _, _, err := raizASellar("  " + buena + "\n"); err != nil {
		t.Errorf("una raiz con espacios alrededor es la misma raiz: %v", err)
	}

	malas := []struct {
		nombre string
		valor  string
	}{
		{"media raiz, que es lo que pasa al copiar de un mensaje cortado", buena[:32]},
		{"un caracter de mas", buena + "a"},
		{"un byte de mas", buena + "ff"},
		{"no es hexadecimal", strings.Repeat("zz", 32)},
	}
	for _, m := range malas {
		_, _, err := raizASellar(m.valor)
		if err == nil {
			t.Errorf("%s: se ha aceptado %q. Sellar eso da un sello que verifica y "+
				"no cubre la cadena", m.nombre, m.valor)
			continue
		}
		if !errors.Is(err, ErrRaizIlegible) {
			t.Errorf("%s: el error tiene que ser ErrRaizIlegible y es %v", m.nombre, err)
		}
	}
}

// CONTROL NEGATIVO: sin bandera, la raiz sale del expediente publicado, no de
// la nada. Si esto se rompiera, el test de arriba pasaria igual mientras la
// herramienta sellara cualquier cosa.
func TestSinBanderaLaRaizSaleDelExpedientePublicado(t *testing.T) {
	_, _, err := raizASellar("")
	// Este test corre desde herramientas/sellardemo, asi que rutaDemo no existe
	// aqui: lo que se comprueba es que LO INTENTA y dice donde mirar, no que lo
	// consiga.
	if err == nil {
		t.Fatal("sin bandera y sin expediente delante, esto no puede salir bien: " +
			"significaria que la raiz se la esta inventando")
	}
	if !strings.Contains(err.Error(), rutaDemo) {
		t.Errorf("el error no nombra %s, asi que no se ha buscado ahi: %v", rutaDemo, err)
	}
	if errors.Is(err, ErrRaizIlegible) {
		t.Errorf("el fichero no existe, que no es lo mismo que una raiz ilegible: %v", err)
	}
}
