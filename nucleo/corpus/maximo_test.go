package corpus

import (
	"testing"
	"time"

	"github.com/marcosmatalab/plazum/nucleo/ventana"
)

// EL CABLEADO DEL MAXIMO, QUE ES LO QUE FALTABA Y NO LA PRIMITIVA.
//
// ventana.Maximo estaba construido y probado desde antes de esto, con sus tres
// ramas y la que importa entre ellas: una declaracion del obligado MAS CORTA
// que el suelo no acorta el suelo. Lo que no existia era poder escribirlo en un
// paquete.json, asi que la primitiva estaba encendida en el motor y apagada
// para el corpus: nadie podia usarla sin tocar codigo Go, que es exactamente lo
// que el invariante 2 no quiere.
//
// Estos tests miran la FRONTERA, no el calculo. El calculo ya tiene los suyos.

func maximoDe(suelo, ampliacion string, exigible *bool) *Paquete {
	p := enciendeElReloj(base())
	p.Obligaciones[0].Temporalidad = &Temporalidad{
		Primitiva:          "maximo",
		Hito:               "fin_de_la_retencion",
		Suelo:              suelo,
		Ampliacion:         ampliacion,
		AmpliacionExigible: exigible,
		Regimen:            RegimenSpec{Computo: "naturales", Cierre: "fin_de_dia"},
		Disparador:         map[string]string{"hecho": "publicacion_de_la_actualizacion"},
	}
	return p
}

func siNo(b bool) *bool { return &b }

// ---------------------------------------------------------------------------
// 1. El invariante 8 en esta frontera: el valor cero esta PROHIBIDO.
// ---------------------------------------------------------------------------

// UN MAXIMO CON AMPLIACION Y SIN DECIR SI LA NORMA LA EXIGE NO CARGA.
//
// Las dos respuestas son opuestas y las dos son plausibles, asi que no hay
// defecto que elegir. Y lo que decide que se prohiba en vez de elegir uno es
// CUAL SALE POR DESCUIDO: el valor cero de un bool es `false`, o sea el lado
// permisivo, que colapsa al suelo en silencio y ensena una fecha cerrada donde
// no la hay. Quien lea esa fecha tira la evidencia creyendo que ya podia.
func TestUnaAmpliacionSinDecirSiLaNormaLaExigeNoCarga(t *testing.T) {
	p := maximoDe("P120M", "fin_del_periodo_de_soporte", nil)
	if !hay(p.Validar(), ErrAmpliacionSinExigible) {
		t.Fatalf("el nil de ampliacion_exigible tiene que ser error, no un false comodo: %v",
			p.Validar())
	}
	// CONTROL NEGATIVO, y en las DOS direcciones, porque el fallo probable de
	// esta guarda es acusar al que si contesta.
	for _, v := range []bool{true, false} {
		q := maximoDe("P120M", "fin_del_periodo_de_soporte", siNo(v))
		if hay(q.Validar(), ErrAmpliacionSinExigible) {
			t.Errorf("ampliacion_exigible=%v es una respuesta y la guarda la rechaza: %v",
				v, q.Validar())
		}
	}
}

// Y LA BANDERA SOBRE UNA RAMA QUE NO EXISTE, que es el mismo fallo del otro
// lado: afirma que alguien penso en una segunda rama que el paquete no declara.
func TestUnaBanderaDeAmpliacionSinAmpliacionNoCarga(t *testing.T) {
	p := maximoDe("P120M", "", siNo(true))
	if !hay(p.Validar(), ErrExigibleSinAmpliacion) {
		t.Fatalf("una bandera sobre una rama inexistente tiene que caerse: %v", p.Validar())
	}
	q := maximoDe("P120M", "", nil) // rama unica: legitimo
	if hay(q.Validar(), ErrExigibleSinAmpliacion) {
		t.Errorf("un maximo de rama unica es legitimo y la guarda lo acusa: %v", q.Validar())
	}
}

// ---------------------------------------------------------------------------
// 2. Lo que hace que un maximo sea un maximo.
// ---------------------------------------------------------------------------

func TestUnMaximoSinSueloNoCarga(t *testing.T) {
	p := maximoDe("", "fin_del_periodo_de_soporte", siNo(true))
	if !hay(p.Validar(), ErrMaximoSinSuelo) {
		t.Fatalf("un maximo sin la rama de la norma es un plazo con otro nombre: %v", p.Validar())
	}
	// Control negativo doble: con suelo carga, y con "indeterminado" tambien,
	// porque «obliga y no dice cuanto» es una respuesta (D-17) y no un hueco.
	for _, s := range []string{"P120M", "indeterminado"} {
		q := maximoDe(s, "fin_del_periodo_de_soporte", siNo(true))
		if hay(q.Validar(), ErrMaximoSinSuelo) {
			t.Errorf("suelo %q es una respuesta y la guarda la rechaza: %v", s, q.Validar())
		}
	}
}

func TestUnMaximoSinDisparadorNoCarga(t *testing.T) {
	p := maximoDe("P120M", "fin_del_periodo_de_soporte", siNo(true))
	p.Obligaciones[0].Temporalidad.Disparador = nil
	if !hay(p.Validar(), ErrMaximoSinDisparador) {
		t.Fatalf("las dos ramas cuentan desde el mismo hecho: sin el no arranca ninguna: %v",
			p.Validar())
	}
	q := maximoDe("P120M", "fin_del_periodo_de_soporte", siNo(true))
	if hay(q.Validar(), ErrMaximoSinDisparador) {
		t.Errorf("con disparador no puede quejarse: %v", q.Validar())
	}
}

// LOS CAMPOS DE UNA PRIMITIVA NO VIVEN EN OTRA. Un `suelo` sobre una periodica
// no hace nada y ademas AFIRMA que alguien penso en el minimo legal de esa
// obligacion: es la familia de la directiva inerte, escrita en un campo.
func TestLosCamposDelMaximoEnOtraPrimitivaNoCargan(t *testing.T) {
	base := func() *Paquete {
		p := enciendeElReloj(base())
		p.Obligaciones[0].Temporalidad = &Temporalidad{
			Primitiva: "periodica", Cadencia: "P12M",
			Regimen:                   RegimenSpec{Computo: "naturales", Cierre: "fin_de_dia"},
			OrigenDelIntervalo:        IntervaloPropuesto,
			JustificacionDelIntervalo: justifBuena,
			CuandoCambiarlo:           cuandoBueno,
			Disparador:                map[string]string{"hecho": "ultima_revision"},
		}
		return p
	}
	con := base()
	con.Obligaciones[0].Temporalidad.Suelo = "P120M"
	if !hay(con.Validar(), ErrCampoDeMaximoFueraDeSitio) {
		t.Errorf("un suelo sobre una periodica no hace nada y afirma que alguien lo penso: %v",
			con.Validar())
	}
	// Control negativo: la misma periodica sin esos campos carga.
	if hay(base().Validar(), ErrCampoDeMaximoFueraDeSitio) {
		t.Errorf("una periodica limpia no puede acusarse: %v", base().Validar())
	}
}

// ---------------------------------------------------------------------------
// 3. EXIGIBLE RECORRIDO POR LAS DOS RAMAS, con dato sintetico.
// ---------------------------------------------------------------------------

// UN BOOLEANO SIN LAS DOS RAMAS RECORRIDAS ES LA RAMA QUE NUNCA SE EJECUTA.
//
// `exigible` decide si el reloj OBLIGA o solo informa, y las dos respuestas
// producen resultados distintos con los MISMOS datos. Este test las recorre las
// dos con la ampliacion AUSENTE, que es donde se separan, y por eso es control
// positivo de las dos y no de una.
//
// El corpus no alcanza hoy la rama `false` (las ocho retenciones del CRA son
// todas exigibles), asi que el dato es sintetico y se dice: una rama que solo
// se probaria cuando exista un paquete que la use es una rama sin probar.
func TestExigibleSeRecorrePorLasDosRamasConLaAmpliacionAusente(t *testing.T) {
	publicacion := time.Date(2028, 3, 15, 0, 0, 0, 0, time.UTC)
	hechos := map[string]time.Time{"publicacion_de_la_actualizacion": publicacion}
	hasta := time.Date(2045, 1, 1, 0, 0, 0, 0, time.UTC)

	casos := []struct {
		nombre   string
		exigible bool
		estado   ventana.EstadoVenc
		porque   string
	}{
		{"exigible: falta un dato que la norma pide", true, ventana.PendienteDeHecho,
			"la fecha final NO se conoce y el suelo si: dar el suelo como fecha cerrada seria " +
				"un verde mas debil que se lee igual que uno fuerte"},
		{"no exigible: la ausencia significa que no hay segunda rama", false, ventana.Determinado,
			"aqui la ausencia si es una respuesta, asi que rige el suelo"},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			p := maximoDe("P120M", "fin_del_periodo_de_soporte", siNo(c.exigible))
			vs, err := VencimientosDe(p.Obligaciones[0], hechos, hasta)
			if err != nil {
				t.Fatalf("el ejecutor no sabe correr un maximo desde un paquete: %v", err)
			}
			if len(vs) != 1 {
				t.Fatalf("un maximo da un vencimiento y ha dado %d", len(vs))
			}
			if vs[0].Estado != c.estado {
				t.Fatalf("exigible=%v tiene que dar %v y ha dado %v.\n  %s\n  regla: %s",
					c.exigible, c.estado, vs[0].Estado, c.porque, vs[0].Regla)
			}
			// Y EL SUELO NO SE PIERDE EN NINGUNA DE LAS DOS. En la exigible sale
			// como cota inferior (NoAntesDe) y en la otra como vencimiento: lo
			// que no puede pasar es que el suelo desaparezca, porque es lo unico
			// que la norma dice con certeza.
			suelo := time.Date(2038, 3, 15, 23, 59, 59, 0, time.UTC)
			got := vs[0].Vence
			if c.estado == ventana.PendienteDeHecho {
				got = vs[0].NoAntesDe
			}
			if !got.Equal(suelo) {
				t.Errorf("el suelo legal se ha perdido: esperaba %s y hay %s",
					suelo.Format(time.RFC3339), got.Format(time.RFC3339))
			}
		})
	}
}

// Y LAS TRES RAMAS DEL CALCULO, ATRAVESANDO EL CABLEADO. No duplican los
// dorados del motor: lo que afirman es que el paquete llega ENTERO hasta la
// primitiva, que es donde un cableado se rompe (un campo que se lee mal deja el
// calculo bueno y la respuesta mala).
func TestElMaximoLlegaEnteroDesdeElPaqueteHastaLaPrimitiva(t *testing.T) {
	publicacion := time.Date(2028, 3, 15, 0, 0, 0, 0, time.UTC)
	suelo := time.Date(2038, 3, 15, 23, 59, 59, 0, time.UTC)
	hasta := time.Date(2060, 1, 1, 0, 0, 0, 0, time.UTC)

	casos := []struct {
		nombre    string
		soporte   time.Time
		esperado  time.Time
		queGana   string
		hayAmplia bool
	}{
		{"gana la ampliacion declarada", time.Date(2042, 6, 1, 0, 0, 0, 0, time.UTC),
			time.Date(2042, 6, 1, 0, 0, 0, 0, time.UTC), "la ampliacion", true},
		{"gana el suelo: la declaracion es mas corta y no reduce el minimo legal",
			time.Date(2031, 1, 1, 0, 0, 0, 0, time.UTC), suelo, "el suelo", true},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			p := maximoDe("P120M", "fin_del_periodo_de_soporte", siNo(true))
			hechos := map[string]time.Time{
				"publicacion_de_la_actualizacion": publicacion,
				"fin_del_periodo_de_soporte":      c.soporte,
			}
			vs, err := VencimientosDe(p.Obligaciones[0], hechos, hasta)
			if err != nil {
				t.Fatalf("%v", err)
			}
			if len(vs) != 1 || vs[0].Estado != ventana.Determinado {
				t.Fatalf("esperaba un vencimiento determinado: %+v", vs)
			}
			if !vs[0].Vence.Equal(c.esperado) {
				t.Fatalf("tenia que ganar %s (%s) y el motor dice %s.\n  regla: %s",
					c.queGana, c.esperado.Format(time.RFC3339),
					vs[0].Vence.Format(time.RFC3339), vs[0].Regla)
			}
		})
	}
}
