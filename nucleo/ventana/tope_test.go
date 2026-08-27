package ventana

import (
	"testing"
	"time"
)

// EL TOPE: dos plazos que vinculan A LA VEZ desde dos hechos distintos.
//
// Lo pidio la familia A al medirla antes de escribir corpus, y lo pidio un
// articulo concreto: el art. 5.1.a del Reglamento Delegado (UE) 2025/301 de
// DORA da cuatro horas desde la CLASIFICACION del incidente como grave "y a mas
// tardar veinticuatro horas despues" del CONOCIMIENTO. No son dos lecturas
// discrepantes (para eso estan las Alternativas) ni dos plazos que se excluyen
// (para eso esta Clase): son dos plazos que hay que cumplir los dos, asi que la
// fecha util es una sola, la primera de las dos.
//
// Ensenar las dos y dejar que el operador elija seria dejarle una cuenta que
// puede hacer mal el dia que mas prisa tiene.

// Nombres neutros: aqui se mide el motor, no se cablea ninguna norma.
const (
	baseClasificacion = "incidente.clasificacion"
	baseConocimiento  = "incidente.conocimiento"
)

func topeDe(t *testing.T, limite string, caduca bool) *Tope {
	t.Helper()
	d, err := ParseDuracion(limite)
	if err != nil {
		t.Fatal(err)
	}
	return &Tope{Desde: baseConocimiento, Limite: d, Reg: regNat(t), Caduca: caduca, Cita: "cita"}
}

func plazoConTope(t *testing.T, limite string, tope *Tope) Plazo {
	t.Helper()
	d, err := ParseDuracion(limite)
	if err != nil {
		t.Fatal(err)
	}
	return Plazo{Disparador: baseClasificacion, Hitos: []Hito{
		{ID: "notificacion_inicial", Limite: d, Reg: regNat(t), Tope: tope},
	}}
}

func unico(t *testing.T, vs []Vencimiento) Vencimiento {
	t.Helper()
	if len(vs) != 1 {
		t.Fatalf("se esperaba un vencimiento y hay %d: %+v", len(vs), vs)
	}
	return vs[0]
}

func instante(t *testing.T, s string) time.Time {
	t.Helper()
	x, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatal(err)
	}
	return x
}

func TestElTopeMandaCuandoCaeAntesQueElLimitePrincipal(t *testing.T) {
	p := plazoConTope(t, "PT4H", topeDe(t, "PT24H", false))
	// Se conoce a las 08:00 y se clasifica veintidos horas despues: cuatro
	// horas desde la clasificacion caerian a las 10:00 del dia siguiente, dos
	// horas DESPUES del tope de veinticuatro desde el conocimiento.
	v := unico(t, p.Vencimientos(Hechos{
		baseConocimiento:  instante(t, "2026-09-01T08:00:00Z"),
		baseClasificacion: instante(t, "2026-09-02T06:00:00Z"),
	}, time.Time{}))
	quiero := instante(t, "2026-09-02T08:00:00Z")
	if v.Estado != Determinado || !v.Vence.Equal(quiero) {
		t.Errorf("vence %s (%v) y tenia que vencer %s: cuando los dos plazos vinculan, manda "+
			"el que cae antes", v.Vence.Format(time.RFC3339), v.Estado, quiero.Format(time.RFC3339))
	}
	if !contiene(v.Regla, "MANDA EL TOPE") {
		t.Errorf("la regla no dice cual de los dos plazos ha ganado: %q", v.Regla)
	}
}

func TestElLimitePrincipalMandaCuandoCaeAntesQueElTope(t *testing.T) {
	p := plazoConTope(t, "PT4H", topeDe(t, "PT24H", false))
	// Se clasifica una hora despues de conocerlo: cuatro horas desde ahi caen
	// mucho antes que veinticuatro desde el conocimiento.
	v := unico(t, p.Vencimientos(Hechos{
		baseConocimiento:  instante(t, "2026-09-01T08:00:00Z"),
		baseClasificacion: instante(t, "2026-09-01T09:00:00Z"),
	}, time.Time{}))
	quiero := instante(t, "2026-09-01T13:00:00Z")
	if v.Estado != Determinado || !v.Vence.Equal(quiero) {
		t.Errorf("vence %s (%v) y tenia que vencer %s", v.Vence.Format(time.RFC3339),
			v.Estado, quiero.Format(time.RFC3339))
	}
	if contiene(v.Regla, "MANDA EL TOPE") {
		t.Errorf("dice que manda el tope y manda el principal: %q", v.Regla)
	}
}

// LAS DOS FORMAS DE LA CADUCIDAD, que es el invariante 8 sobre una frontera
// nueva. El valor cero de Caduca es false, o sea el tope vincula SIEMPRE,
// aunque sea imposible de cumplir. Caducar es la lectura blanda y hay que
// PEDIRLA con su cita.
//
// Si fuera al reves, un paquete que se olvidara del campo aflojaria un plazo
// sin que nadie lo notara.
func TestElTopeCaducaSoloCuandoElPaqueteLoPideYLoCita(t *testing.T) {
	// La clasificacion llega DESPUES de que el tope ya haya vencido: se conoce
	// el dia 1 a las 08:00 y no se clasifica hasta el dia 3.
	hechos := Hechos{
		baseConocimiento:  instante(t, "2026-09-01T08:00:00Z"),
		baseClasificacion: instante(t, "2026-09-03T10:00:00Z"),
	}

	t.Run("valor cero: el tope vincula igual, y la fecha ya esta pasada", func(t *testing.T) {
		p := plazoConTope(t, "PT4H", topeDe(t, "PT24H", false))
		v := unico(t, p.Vencimientos(hechos, time.Time{}))
		quiero := instante(t, "2026-09-02T08:00:00Z")
		if !v.Vence.Equal(quiero) {
			t.Errorf("vence %s y tenia que vencer %s. El valor cero de Caduca es el "+
				"RESTRICTIVO: sin pedirlo, el tope no se relaja",
				v.Vence.Format(time.RFC3339), quiero.Format(time.RFC3339))
		}
	})

	t.Run("caduca pedido: manda el limite principal", func(t *testing.T) {
		p := plazoConTope(t, "PT4H", topeDe(t, "PT24H", true))
		v := unico(t, p.Vencimientos(hechos, time.Time{}))
		quiero := instante(t, "2026-09-03T14:00:00Z")
		if !v.Vence.Equal(quiero) {
			t.Errorf("vence %s y tenia que vencer %s: el tope habia vencido antes de que "+
				"ocurriera el disparador, asi que ya no manda",
				v.Vence.Format(time.RFC3339), quiero.Format(time.RFC3339))
		}
		if !contiene(v.Regla, "caducado") {
			t.Errorf("la regla no dice que el tope ha caducado: %q", v.Regla)
		}
	})
}

// SIN LA BASE DEL TOPE NO SE DA FECHA, y la direccion importa: el tope solo
// puede ACORTAR, asi que ignorarlo daria una fecha mas TARDE que la real. En un
// producto de cumplimiento, equivocarse hacia tarde es el error caro.
//
// Se recorren LAS DOS FORMAS DE LA NADA: el hecho ausente y el mapa entero
// vacio.
func TestSinLaBaseDelTopeNoSeDaFecha(t *testing.T) {
	p := plazoConTope(t, "PT4H", topeDe(t, "PT24H", false))
	casos := map[string]Hechos{
		"falta el hecho del tope": {baseClasificacion: instante(t, "2026-09-01T09:00:00Z")},
		"mapa de hechos vacio":    {},
	}
	for nombre, h := range casos {
		t.Run(nombre, func(t *testing.T) {
			v := unico(t, p.Vencimientos(h, time.Time{}))
			if v.Estado != PendienteDeHecho {
				t.Errorf("estado %v con fecha %s. Sin la base del tope, la fecha que sale del "+
					"limite principal es mas TARDE que la real, y darla por buena es el error "+
					"que cuesta dinero", v.Estado, v.Vence.Format(time.RFC3339))
			}
			if !v.Vence.IsZero() {
				t.Errorf("se ha devuelto una fecha (%s) en un vencimiento pendiente",
					v.Vence.Format(time.RFC3339))
			}
		})
	}
}

// CONTROL NEGATIVO: sin tope, el hito se comporta exactamente como antes. Si
// esto fallara, el campo nuevo habria cambiado el calculo de los veintitantos
// relojes que ya estaban escritos.
func TestSinTopeElHitoSeComportaComoSiempre(t *testing.T) {
	p := plazoConTope(t, "PT4H", nil)
	v := unico(t, p.Vencimientos(Hechos{
		baseClasificacion: instante(t, "2026-09-01T09:00:00Z"),
	}, time.Time{}))
	quiero := instante(t, "2026-09-01T13:00:00Z")
	if v.Estado != Determinado || !v.Vence.Equal(quiero) {
		t.Errorf("vence %s (%v) y tenia que vencer %s", v.Vence.Format(time.RFC3339),
			v.Estado, quiero.Format(time.RFC3339))
	}
}
