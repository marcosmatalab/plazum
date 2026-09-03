package metrica

import (
	"errors"
	"strings"
	"testing"
)

// LOS TRES VALORES IMPOSIBLES, CON SU CASO REAL CADA UNO.
//
// No son hipotesis: los tres han pasado en este repositorio en dos dias, y los
// tres se leyeron como cifras normales.
func TestUnValorImposibleNoSePublica(t *testing.T) {
	casos := []struct {
		nombre    string
		f         Fraccion
		centinela error
		yDice     string
	}{
		// EL CASO REAL: `cra` tenia 23 relojes con cita escritos y su censo
		// contaba 22. Un paquete al 104,5 % sube el agregado sin que nada lo
		// nombre.
		{"la parte mayor que el total, que es la unidad equivocada",
			Fraccion{Parte: 23, Total: 22,
				QueEsParte: "relojes con cita escritos", QueEsTotal: "puntos censados"},
			ErrNumeradorMayor, "no cuentan la misma unidad"},
		// EL OTRO CASO REAL: un paquete referencial no se puede censar, asi que
		// su denominador no existe. Ponerle un 0 y tratarlo como uno daria un
		// porcentaje inventado; el frente A se nego a proponer ese 0.
		{"el total cero, que NO se trata como uno",
			Fraccion{Parte: 5, Total: 0,
				QueEsParte: "rituales escritos", QueEsTotal: "puntos censados"},
			ErrDenominadorCero, "sin denominador, 5 rituales escritos"},
		{"un cardinal negativo se para aqui",
			Fraccion{Parte: -1, Total: 10,
				QueEsParte: "relojes", QueEsTotal: "puntos"},
			ErrNegativo, "error aguas arriba"},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			pct, err := c.f.Porcentaje()
			if err == nil {
				t.Fatalf("se ha publicado un %.1f %% imposible", pct)
			}
			if !errors.Is(err, c.centinela) {
				t.Errorf("el error no es el centinela %v: %v", c.centinela, err)
			}
			if pct != 0 {
				t.Errorf("con error devuelve %.1f y tiene que devolver 0: si no, quien "+
					"llame podria usar la cifra sin mirar el error", pct)
			}
			if !strings.Contains(err.Error(), c.yDice) {
				t.Errorf("el error no dice %q, asi que no es accionable: %v", c.yDice, err)
			}
			// Y NOMBRA LOS DOS LADOS. Es lo que convierte «23 sobre 22» en
			// «relojes escritos sobre puntos censados», que es donde esta la
			// respuesta.
			for _, n := range []string{c.f.QueEsParte, c.f.QueEsTotal} {
				if n != "" && !strings.Contains(err.Error(), n) {
					t.Errorf("el error no nombra %q, asi que no dice donde mirar: %v", n, err)
				}
			}
		})
	}
}

// CONTROL POSITIVO: las fracciones legitimas SI se publican, y con el valor
// correcto. Sin esta mitad, un validador que dijera que no a todo pasaria los
// casos de arriba por el motivo equivocado.
func TestUnaFraccionLegitimaSePublicaConSuValor(t *testing.T) {
	casos := []struct {
		nombre string
		f      Fraccion
		quiere float64
	}{
		{"la cobertura de la v1 del 03-09-2026",
			Fraccion{Parte: 73, Total: 142,
				QueEsParte: "relojes con intervalo de la norma", QueEsTotal: "puntos censados"},
			51.408450704225352},
		// EL CERO ARRIBA ES LEGITIMO Y NO ES UN VALOR IMPOSIBLE. `iso27001`
		// declara `censados: 0`... pero ese es el otro lado. Aqui: un paquete
		// con puntos censados y ni un reloj escrito todavia es un 0 % honesto,
		// y confundirlo con un error dejaria de publicar la peor noticia, que
		// es la que mas hay que publicar.
		{"cero arriba es 0 % y no es un imposible",
			Fraccion{Parte: 0, Total: 9, QueEsParte: "relojes", QueEsTotal: "puntos"}, 0},
		{"la parte igual al total es 100 % y tampoco lo es",
			Fraccion{Parte: 48, Total: 48, QueEsParte: "relojes", QueEsTotal: "puntos"}, 100},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			pct, err := c.f.Porcentaje()
			if err != nil {
				t.Fatalf("una fraccion legitima ha sido rechazada: %v", err)
			}
			if diferencia(pct, c.quiere) > 0.0001 {
				t.Errorf("ha dado %.6f y esperaba %.6f", pct, c.quiere)
			}
			if pct > 100 {
				t.Errorf("ha salido un %.1f %%, que es imposible y tenia que haberse "+
					"rechazado antes de dividir", pct)
			}
		})
	}
}

// UNA FRACCION SIN NOMBRES SE PUEDE CALCULAR, PERO EL ERROR LO DICE.
//
// No se prohibe: prohibirlo obligaria a nombrar en sitios donde no aporta, y
// una regla que estorba se esquiva. Lo que se hace es que el error sea peor de
// leer, que es el incentivo correcto.
func TestUnaFraccionSinNombresAvisaEnSuError(t *testing.T) {
	_, err := Fraccion{Parte: 5, Total: 3}.Porcentaje()
	if err == nil {
		t.Fatal("5 sobre 3 tenia que rechazarse")
	}
	if !strings.Contains(err.Error(), "SIN NOMBRAR") {
		t.Errorf("el error de una fraccion sin nombres no lo dice, asi que quien la lea no "+
			"sabra que le falta la mitad de la informacion: %v", err)
	}
}

// LA LEY DE CONSERVACION, con el descuadre en las dos direcciones.
//
// Que sobre y que falte son fallos distintos y los dos son silencio: si los
// cubos suman de menos, algo no sale en ninguna vista; si suman de mas, algo
// sale dos veces y la cuenta del pie miente.
func TestUnRepartoQueNoCuadraSeDice(t *testing.T) {
	casos := []struct {
		nombre string
		total  int
		partes map[string]int
		cuadra bool
		yDice  string
	}{
		{"cuadra", 10, map[string]int{"mostrados": 6, "no_alcanzados": 3, "sin_fecha": 1}, true, ""},
		{"faltan tres: no salen en ninguna vista", 10,
			map[string]int{"mostrados": 6, "sin_fecha": 1}, false, "-3"},
		{"sobran dos: algo se cuenta dos veces", 10,
			map[string]int{"mostrados": 8, "no_alcanzados": 4}, false, "+2"},
		// EL CASO DEGENERADO: cero total y cero cubos cuadra, y tiene que
		// cuadrar. Lo contrario obligaria a inventar un cubo para un corpus
		// vacio.
		{"cero y cero cuadra", 0, map[string]int{}, true, ""},
		// Y EL CUBO NEGATIVO, que es un error de conteo aguas arriba y no un
		// descuadre: se para antes de sumarlo.
		{"un cubo negativo se para antes de sumar", 10,
			map[string]int{"mostrados": 12, "raro": -2}, false, "negativo"},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			err := Cuadra(c.total, "obligaciones con reloj instaladas", c.partes)
			if c.cuadra {
				if err != nil {
					t.Fatalf("un reparto que cuadra ha sido rechazado: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("un reparto que NO cuadra ha pasado, que es el silencio que esto " +
					"existe para impedir")
			}
			if !strings.Contains(err.Error(), c.yDice) {
				t.Errorf("el error no dice %q: %v", c.yDice, err)
			}
			// Y NOMBRA LOS CUBOS. Saber que faltan tres no sirve; saber de que
			// cubo, si.
			if !strings.Contains(err.Error(), "obligaciones con reloj instaladas") {
				t.Errorf("el error no nombra lo que se estaba repartiendo: %v", err)
			}
		})
	}
}

func diferencia(a, b float64) float64 {
	if a > b {
		return a - b
	}
	return b - a
}
