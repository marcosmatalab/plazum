package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Un incidente significativo REABRE las revisiones, y la pantalla dice por que.
//
// Es el camino entero del disparador por evento: un solo hecho registrado
// (`ultimo_incidente_significativo`) reabre las 22 revisiones del anexo de
// 2024/2690 que lo declaran, y cada una deja de tener fecha porque la norma
// dice CUANDO hay que revisar y no da plazo para hacerlo.
func TestUnIncidenteReabreLasRevisionesYLaPantallaDicePorQue(t *testing.T) {
	dir := t.TempDir()
	ruta := filepath.Join(dir, "alcance.json")
	if err := os.WriteFile(ruta, []byte(`{
  "organizacion": "Prueba de reapertura",
  "sujeto": "acme",
  "hechos": [{"pred": "papel_nis2_tecnica", "args": ["acme", "entidad_pertinente"]}],
  "fechas": {
    "ultima_revision_de_roles_y_responsabilidades": "2026-06-01",
    "ultimo_incidente_significativo": "2026-07-15"
  }
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	out, _, codigo := correrCalendario(t, "--corpus=../../paquetes", "--alcance="+ruta)
	if codigo != 0 {
		t.Fatalf("codigo %d\n%s", codigo, out)
	}

	// La revision del punto 1.2.6 tenia fecha (junio + 12 meses) y la pierde,
	// porque manda el hecho.
	if !strings.Contains(out, "REABRE la revision") {
		t.Errorf(`la pantalla no explica la reapertura.

  Un «obliga y la norma no da numero» a secas no distingue dos cosas muy
  distintas para quien lo lee: que la norma nunca de plazo, o que un hecho suyo
  haya reabierto una revision que ya tenia su fecha. La unica que las separa es
  la derivacion del motor, y por eso se imprime.
%s`, out)
	}
	// Y dice de que hecho y de que fecha, que es lo que la hace defendible.
	for _, quiero := range []string{"ultimo_incidente_significativo", "2026-07-15", "2026-06-01"} {
		if !strings.Contains(out, quiero) {
			t.Errorf("la explicacion no dice %q", quiero)
		}
	}
	// Y dice como se cierra, que es lo unico que el operador puede hacer.
	if !strings.Contains(out, "Se cierra registrando") {
		t.Error("la explicacion no dice como se cierra la reapertura")
	}
}

// CONTROL: sin el hecho, la revision conserva su fecha.
//
// Sin este tramo, una derivacion que dejara sin fecha a todas las periodicas
// pasaria el test de arriba con nota.
func TestSinElHechoLaRevisionConservaSuFecha(t *testing.T) {
	dir := t.TempDir()
	ruta := filepath.Join(dir, "alcance.json")
	if err := os.WriteFile(ruta, []byte(`{
  "organizacion": "Prueba sin incidente",
  "sujeto": "acme",
  "hechos": [{"pred": "papel_nis2_tecnica", "args": ["acme", "entidad_pertinente"]}],
  "fechas": {"ultima_revision_de_roles_y_responsabilidades": "2026-06-01"}
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	out, _, codigo := correrCalendario(t, "--corpus=../../paquetes", "--alcance="+ruta)
	if codigo != 0 {
		t.Fatalf("codigo %d\n%s", codigo, out)
	}
	if strings.Contains(out, "REABRE la revision") {
		t.Error("sin el hecho registrado no hay nada que reabrir, y la pantalla dice que si")
	}
	// La fecha del ciclo, en junio de 2027.
	if !strings.Contains(out, "junio de 2027") {
		t.Errorf("sin reapertura, el ciclo de doce meses tenia que dar una fecha en junio "+
			"de 2027:\n%s", out)
	}
}
