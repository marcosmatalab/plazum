package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"plazum/nucleo/expediente"
)

// Las puertas de `plazum export`, la orden que alimenta el SIEM del cliente.
//
// Lo que se comprueba aqui no es el contenido de los eventos (eso vive en
// superficies/export, con su mutacion y su control negativo) sino el contrato de
// la ORDEN, que es lo que se rompe al montarla: por donde sale cada cosa, con
// que permisos se escribe el fichero y que el expediente publicado del
// repositorio pasa por ella entero.

func expedienteDelRepo(t *testing.T) *expediente.Expediente {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "expediente-demo.json"))
	if err != nil {
		t.Fatalf("no se puede leer el expediente publicado del repositorio: %v", err)
	}
	e, err := expediente.Cargar(b)
	if err != nil {
		t.Fatalf("el expediente publicado no carga: %v", err)
	}
	return e
}

// Por la salida estandar salen EVENTOS y nada mas. Una sola linea de cortesia
// mezclada rompe la ingesta del receptor, y la rompe en silencio: el SIEM
// descarta la linea que no parsea y no avisa a nadie.
func TestPorLaSalidaEstandarSalenEventosYNadaMas(t *testing.T) {
	e := expedienteDelRepo(t)
	var salida, errores bytes.Buffer
	if rc := cmdExport(e, nil, &salida, &errores); rc != 0 {
		t.Fatalf("export ha salido con %d. Errores:\n%s", rc, errores.String())
	}
	texto := salida.String()
	if texto == "" {
		t.Fatal("el expediente publicado no ha producido ni un evento: si esto se queda " +
			"vacio, el resto de comprobaciones de este test recorren la nada")
	}
	lineas := strings.Split(strings.TrimSuffix(texto, "\n"), "\n")
	if len(lineas) < 5 {
		t.Fatalf("solo han salido %d eventos del expediente publicado, que trae 3 entradas, "+
			"1 punto de control, 2 controles y 4 plazos", len(lineas))
	}
	for i, l := range lineas {
		var m map[string]any
		if err := json.Unmarshal([]byte(l), &m); err != nil {
			t.Fatalf("la linea %d de la salida estandar no es un evento JSON (%v): %q\n"+
				"  Cualquier texto de cortesia en esta tuberia rompe la ingesta del SIEM.",
				i+1, err, l)
		}
		if m["event.action"] == nil {
			t.Errorf("la linea %d no trae accion: %v", i+1, m)
		}
	}
	// Y el resumen SI tiene que estar, por el otro canal: un export que sale con
	// 0 y no mando nada es indistinguible de uno que mando todo.
	if !strings.Contains(errores.String(), "eventos") {
		t.Errorf("el canal de errores no dice cuantos eventos se mandaron:\n%s", errores.String())
	}
}

// CONTROL NEGATIVO del test de arriba: si el resumen se colara por la salida
// estandar, la comprobacion tiene que verlo. Sin esto, un parser indulgente
// dejaria pasar cualquier cosa.
func TestUnaLineaDeCortesiaEnLaTuberiaSeCaza(t *testing.T) {
	sucia := "{\"event.action\":\"x\"}\n8 eventos: 3 entradas de la cadena\n"
	fallos := 0
	for _, l := range strings.Split(strings.TrimSuffix(sucia, "\n"), "\n") {
		var m map[string]any
		if err := json.Unmarshal([]byte(l), &m); err != nil {
			fallos++
		}
	}
	if fallos != 1 {
		t.Fatalf("la comprobacion de la tuberia no distingue una linea de resumen de un "+
			"evento (%d fallos). Mientras no la distinga, el verde de arriba no dice nada",
			fallos)
	}
}

func TestConSalidaAFicheroElLogNoQuedaLegibleParaTodaLaMaquina(t *testing.T) {
	e := expedienteDelRepo(t)
	destino := filepath.Join(t.TempDir(), "auditoria.jsonl")
	var salida, errores bytes.Buffer
	if rc := cmdExport(e, []string{"-salida", destino}, &salida, &errores); rc != 0 {
		t.Fatalf("export a fichero ha salido con %d: %s", rc, errores.String())
	}
	if salida.Len() != 0 {
		t.Errorf("con --salida a fichero no puede salir nada por la salida estandar: %q",
			salida.String())
	}
	inf, err := os.Stat(destino)
	if err != nil {
		t.Fatalf("no se ha escrito el fichero: %v", err)
	}
	if inf.Size() == 0 {
		t.Fatal("el fichero esta vacio")
	}
	if runtime.GOOS == "windows" {
		// Windows no aplica el modo POSIX; la restriccion la pone la ACL del
		// directorio. Se comprueba donde significa algo, y se dice aqui para
		// que nadie lea el salto como un descuido.
		t.Log("permisos no comprobados en windows: el modo POSIX no se aplica aqui")
		return
	}
	if m := inf.Mode().Perm(); m != 0o600 {
		t.Errorf("el log de auditoria se ha escrito con permisos %o y tenia que ser 600. "+
			"Lleva el rastro de auditoria de la organizacion en texto plano", m)
	}
}

func TestElExportSobreUnDestinoImposibleLoDiceYNoSaleConCero(t *testing.T) {
	e := expedienteDelRepo(t)
	// Un directorio que no existe: el fallo es del entorno, no del expediente.
	destino := filepath.Join(t.TempDir(), "no", "existe", "auditoria.jsonl")
	var salida, errores bytes.Buffer
	if rc := cmdExport(e, []string{"-salida", destino}, &salida, &errores); rc == 0 {
		t.Fatal("escribir en un directorio que no existe ha salido con 0: el operador " +
			"creeria que mando el log")
	}
	if !strings.Contains(errores.String(), "Arreglo") {
		t.Errorf("el error no dice como se arregla:\n%s", errores.String())
	}
}

func TestDosEjecucionesDeLaOrdenDanElMismoFichero(t *testing.T) {
	e := expedienteDelRepo(t)
	var uno, dos, errores bytes.Buffer
	if rc := cmdExport(e, nil, &uno, &errores); rc != 0 {
		t.Fatalf("primera ejecucion: %s", errores.String())
	}
	if rc := cmdExport(e, nil, &dos, &errores); rc != 0 {
		t.Fatalf("segunda ejecucion: %s", errores.String())
	}
	if uno.String() != dos.String() {
		t.Error("dos ejecuciones de la orden sobre el mismo expediente dan ficheros " +
			"distintos: el receptor no puede deduplicar")
	}
}

// La ayuda de la orden tiene que decir lo que un operador necesita saber ANTES
// de mandar el fichero a un tercero, que es que lo borrado no sale.
func TestLaAyudaDiceQueElBorradoLegalNoSaleEnElFichero(t *testing.T) {
	e := expedienteDelRepo(t)
	var salida, errores bytes.Buffer
	if rc := cmdExport(e, []string{"-h"}, &salida, &errores); rc != 0 {
		t.Fatalf("la ayuda ha salido con %d", rc)
	}
	if !strings.Contains(errores.String(), "borrado legal") {
		t.Errorf("la ayuda no dice que lo borrado con base legal no sale:\n%s", errores.String())
	}
	if salida.Len() != 0 {
		t.Errorf("la ayuda ensucia la salida estandar: %q", salida.String())
	}
}
