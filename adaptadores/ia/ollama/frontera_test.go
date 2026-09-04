package ollama

import (
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"

	"github.com/marcosmatalab/plazum/internal/modulo"
)

// EL INVARIANTE 4 ESCRITO COMO CODIGO: "los adaptadores de LLM viven fuera de
// proceso y JAMAS importan escritura de estado o ledger".
//
// La lista es BLANCA y no negra. Una lista negra deja pasar lo que a nadie se
// le ocurrio prohibir, que es donde entran las cosas nuevas; una lista blanca
// obliga a venir aqui y escribir el import con su motivo.
//
// LO QUE NO PUEDE ENTRAR NUNCA, y por que cada uno:
//
//	nucleo/ledger      escribir en el registro append-only lo que dijo un
//	                   modelo, sin que una persona lo haya confirmado, rompe
//	                   la promesa entera del expediente.
//	nucleo/estado      mutar estado desde el adaptador se salta la aceptacion
//	                   humana registrada (docs/ia.md, arnes punto 1).
//	nucleo/expediente  lo mismo, un piso mas arriba.
//	os                 un adaptador de modelo que lee ficheros del disco es un
//	                   adaptador que puede mandar el disco a un modelo. Lo que
//	                   tenga que ver entra por el parametro `contexto`.
//
// Y `adaptadores/ia` SI entra: es de donde sale el interruptor. Notese la
// direccion: el adaptador de modelo depende del arnes, y no al reves. El arnes
// no sabe que existe Ollama, y por eso se puede verificar una cita sin que haya
// ningun modelo instalado.
var deLaBiblioteca = map[string]bool{
	"bytes":         true,
	"crypto/sha256": true,
	"encoding/hex":  true,
	"encoding/json": true,
	"errors":        true,
	"fmt":           true,
	"io":            true,
	"net/http":      true,
	"net/url":       true,
	"strings":       true,
	"time":          true,
}

// deCasa va SIN el prefijo del modulo: la ruta del modulo vive en go.mod y en
// ningun otro sitio (TestNadieCableaLaRutaDelModulo en la raiz). Escrita aqui
// seria una segunda copia que se queda vieja dando verde, que es justo lo que
// paso con el renombrado del 28-08-2026.
var deCasa = map[string]bool{
	"adaptadores/ia":     true,
	"internal/redactado": true,
	"puertos":            true,
}

func permitido(imp, mod string) bool {
	if !modulo.EsDeCasa(imp, mod) {
		return deLaBiblioteca[imp]
	}
	return deCasa[modulo.Interno(imp, mod)]
}

func TestElAdaptadorDeModeloNoImportaNadaQueEscribaEstado(t *testing.T) {
	mod, err := modulo.Ruta()
	if err != nil {
		t.Fatal(err)
	}
	entradas, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	ficheros := 0
	for _, e := range entradas {
		n := e.Name()
		if e.IsDir() || !strings.HasSuffix(n, ".go") || strings.HasSuffix(n, "_test.go") {
			continue
		}
		ficheros++
		a, err := parser.ParseFile(fset, n, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("no puedo parsear %s: %v", n, err)
		}
		for _, imp := range a.Imports {
			v := strings.Trim(imp.Path.Value, `"`)
			if permitido(v, mod) {
				continue
			}
			t.Errorf(`%s importa %q, que no esta en la lista blanca.

  Invariante 4: los adaptadores de LLM viven fuera de proceso y JAMAS importan
  escritura de estado ni ledger. Este adaptador habla HTTP con un modelo y
  devuelve una propuesta; todo lo demas lo hace otro.

  Si de verdad hace falta, se anade aqui con su motivo escrito, que es la
  conversacion que hay que tener antes de tenerla en produccion.`, n, v)
		}
	}
	if ficheros < 1 {
		t.Fatal("no se ha encontrado ni un fuente en este paquete: el recorrido esta roto " +
			"y esta puerta daria verde sin mirar nada")
	}
	// CONTROL DEL DETECTOR: los cuatro prohibidos de la cabecera no estan en la
	// lista. Si alguien los anadiera "temporalmente", esto lo dice.
	for _, prohibido := range []string{
		mod + "/nucleo/ledger",
		mod + "/nucleo/estado",
		mod + "/nucleo/expediente",
		"os",
	} {
		if permitido(prohibido, mod) {
			t.Errorf("la lista blanca admite %q", prohibido)
		}
	}
	// Y EL CONTROL POSITIVO, que es el que impide que esto sea un `permitido`
	// que dice que no a todo: los que si tienen que estar, estan.
	for _, bueno := range []string{mod + "/adaptadores/ia", mod + "/puertos", "net/http"} {
		if !permitido(bueno, mod) {
			t.Errorf("la lista blanca no admite %q, que si tiene que estar", bueno)
		}
	}
}
