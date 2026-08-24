package perimetro

import "testing"

func arbolDePrueba(t *testing.T) *Arbol {
	t.Helper()
	a, err := NuevoArbol([]Perimetro{
		{ID: "grupo", Nombre: "Grupo"},
		{ID: "es", Nombre: "Filial ES", Padre: "grupo"},
		{ID: "pt", Nombre: "Filial PT", Padre: "grupo"},
		{ID: "es-salud", Nombre: "BU Salud", Padre: "es"},
	})
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func TestHerenciaBajaPorLaCadena(t *testing.T) {
	a := arbolDePrueba(t)
	// un paquete instalado en el grupo aplica a la BU nieta
	if ok, _ := a.Hereda("es-salud", "grupo"); !ok {
		t.Fatal("lo del grupo aplica a la nieta")
	}
	// lo instalado en PT no aplica a ES
	if ok, _ := a.Hereda("es", "pt"); ok {
		t.Fatal("las filiales no se heredan entre si")
	}
}

func TestRollUpSumaHaciaArriba(t *testing.T) {
	a := arbolDePrueba(t)
	out, err := a.RollUp(map[string]int{"es-salud": 3, "es": 2, "pt": 5})
	if err != nil {
		t.Fatal(err)
	}
	if out["grupo"] != 10 || out["es"] != 5 || out["pt"] != 5 || out["es-salud"] != 3 {
		t.Fatalf("roll-up mal: %v", out)
	}
}

func TestCicloSeRechazaAlCargar(t *testing.T) {
	_, err := NuevoArbol([]Perimetro{
		{ID: "a", Padre: "b"}, {ID: "b", Padre: "a"},
	})
	if err == nil {
		t.Fatal("un ciclo de perimetros es error de carga")
	}
}

func TestPadreInexistenteSeRechaza(t *testing.T) {
	if _, err := NuevoArbol([]Perimetro{{ID: "x", Padre: "nadie"}}); err == nil {
		t.Fatal("padre inexistente es error de carga")
	}
}
