package corpus

import (
	"testing"
	"time"

	"github.com/marcosmatalab/plazum/nucleo/ventana"
)

// EL CABLEADO DEL PREAVISO. La primitiva ya estaba probada; esto mira la
// frontera y que el paquete llegue ENTERO hasta ella.

func preavisoDe(efecto, antelacion string) *Paquete {
	p := enciendeElReloj(base())
	p.Obligaciones[0].Vigencia = Vigencia{Desde: "2022-05-05", Origen: VigenciaHeredada}
	p.Obligaciones[0].Temporalidad = &Temporalidad{
		Primitiva:  "preaviso",
		Hito:       "aviso_de_modificacion",
		Efecto:     efecto,
		Antelacion: antelacion,
		Regimen:    RegimenSpec{Computo: "naturales", Cierre: "fin_de_dia"},
	}
	return p
}

func TestUnPreavisoSinEfectoNoCarga(t *testing.T) {
	p := preavisoDe("", "P2M")
	if !hay(p.Validar(), ErrPreavisoSinEfecto) {
		t.Fatalf("sin la fecha que elige el obligado no hay cuenta atras: %v", p.Validar())
	}
	q := preavisoDe("fecha_de_efecto", "P2M")
	if hay(q.Validar(), ErrPreavisoSinEfecto) {
		t.Errorf("con efecto no puede quejarse: %v", q.Validar())
	}
}

func TestUnPreavisoSinAntelacionNoCarga(t *testing.T) {
	p := preavisoDe("fecha_de_efecto", "")
	if !hay(p.Validar(), ErrPreavisoSinAntelacion) {
		t.Fatalf("sin antelacion no hay preaviso, hay un aviso: %v", p.Validar())
	}
	// Control negativo DOBLE: con numero y con "indeterminado", porque «la norma
	// exige avisar antes y no dice cuanto» es una respuesta (D-17), no un hueco.
	for _, a := range []string{"P2M", "indeterminado"} {
		q := preavisoDe("fecha_de_efecto", a)
		if hay(q.Validar(), ErrPreavisoSinAntelacion) {
			t.Errorf("antelacion %q es una respuesta y se rechaza: %v", a, q.Validar())
		}
	}
}

// EL DISPARADOR SOBRA, y esta es la guarda que mas piensa por el lector: un
// preaviso con disparador hace creer que cuenta desde un hecho que le ocurre al
// obligado, que es lo contrario de lo que hace. El motor ni lo mira, asi que
// seria una afirmacion falsa que ademas no cambia el resultado.
func TestUnPreavisoConDisparadorNoCarga(t *testing.T) {
	p := preavisoDe("fecha_de_efecto", "P2M")
	p.Obligaciones[0].Temporalidad.Disparador = map[string]string{"hecho": "solicitud_del_cliente"}
	if !hay(p.Validar(), ErrPreavisoConDisparador) {
		t.Fatalf("un preaviso no cuenta desde un hecho que ocurre: %v", p.Validar())
	}
	if hay(preavisoDe("fecha_de_efecto", "P2M").Validar(), ErrPreavisoConDisparador) {
		t.Error("sin disparador no puede quejarse")
	}
}

// LOS CAMPOS DE UNA PRIMITIVA EN OTRA, en las dos direcciones que existen hoy.
func TestLosCamposDeUnaPrimitivaEnOtraNoCargan(t *testing.T) {
	// preaviso con campos de maximo
	p := preavisoDe("fecha_de_efecto", "P2M")
	p.Obligaciones[0].Temporalidad.Suelo = "P120M"
	if !hay(p.Validar(), ErrCampoDePrimitivaFueraDeSitio) {
		t.Errorf("un suelo sobre un preaviso no lo lee nadie: %v", p.Validar())
	}
	// y al reves: maximo con campos de preaviso
	q := preavisoDe("fecha_de_efecto", "P2M")
	q.Obligaciones[0].Temporalidad = &Temporalidad{
		Primitiva: "maximo", Hito: "fin", Suelo: "P120M",
		Regimen:    RegimenSpec{Computo: "naturales", Cierre: "fin_de_dia"},
		Disparador: map[string]string{"hecho": "x"},
		Antelacion: "P2M",
	}
	if !hay(q.Validar(), ErrCampoDePrimitivaFueraDeSitio) {
		t.Errorf("una antelacion sobre un maximo no la lee nadie: %v", q.Validar())
	}
	// Control negativo: cada una con lo suyo, limpia.
	if hay(preavisoDe("fecha_de_efecto", "P2M").Validar(), ErrCampoDePrimitivaFueraDeSitio) {
		t.Error("un preaviso con solo sus campos no puede acusarse")
	}
}

// Y EL CABLE LLEGA ENTERO: la cuenta atras se calcula desde el paquete.
func TestElPreavisoLlegaEnteroDesdeElPaqueteHastaLaPrimitiva(t *testing.T) {
	p := preavisoDe("fecha_de_efecto", "P2M")
	efecto := time.Date(2027, 6, 15, 0, 0, 0, 0, time.UTC)
	vs, err := VencimientosDe(p.Obligaciones[0],
		map[string]time.Time{"fecha_de_efecto": efecto}, time.Time{})
	if err != nil {
		t.Fatalf("el ejecutor no sabe correr un preaviso desde un paquete: %v", err)
	}
	if len(vs) != 1 || vs[0].Estado != ventana.Determinado {
		t.Fatalf("esperaba un vencimiento determinado: %+v", vs)
	}
	// EL INSTANTE, SIN CERRAR EL DIA, y conviene decir por que porque la primera
	// version de este test esperaba 23:59:59 y el equivocado era el test.
	//
	// Un plazo hacia adelante cierra el dia (Rgto. 1182/71 art. 3.2.b) y eso
	// ALARGA el tiempo del obligado, que es el lado seguro. Hacia atras la
	// simetria se invierte: cerrar el dia le REGALA un dia entero de silencio,
	// y este producto no puede decirle a nadie que todavia le queda tiempo
	// cuando puede no quedarle. El motor devuelve el instante, que es el lado
	// restrictivo, y sus propios dorados lo fijan asi desde antes.
	//
	// LA PREGUNTA QUE ESTO DEJA ABIERTA, dicha en vez de zanjada: si «con una
	// antelacion minima de dos meses» admite avisar DURANTE el 15 de abril, la
	// respuesta del motor es un dia mas estricta que la norma. Es la direccion
	// buena para equivocarse, pero sigue siendo una lectura. El dia que un
	// paquete de la familia G traiga un considerando que lo aclare, se resuelve
	// con una `alternativa` de lectura, que es el mecanismo que este corpus ya
	// tiene para las discrepancias de computo.
	quiero := time.Date(2027, 4, 15, 0, 0, 0, 0, time.UTC)
	if !vs[0].Vence.Equal(quiero) {
		t.Fatalf("dos meses antes del 2027-06-15 es %s y el motor dice %s.\n  regla: %s",
			quiero.Format(time.RFC3339), vs[0].Vence.Format(time.RFC3339), vs[0].Regla)
	}
	// SIN LA FECHA QUE ELIGE EL OBLIGADO no hay nada que preavisar, y no se
	// calla: sale pendiente del hecho.
	sin, err := VencimientosDe(p.Obligaciones[0], map[string]time.Time{}, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if sin[0].Estado != ventana.PendienteDeHecho {
		t.Errorf("sin fecha de efecto tiene que salir pendiente de hecho y sale %v", sin[0].Estado)
	}
}

// LAS TRES RESPUESTAS DE hechoQuePideUnDorado, Y LAS TRES DE
// afirmaQueFaltaElDato, con dato sintetico.
//
// # Por que sinteticas y no contra el corpus
//
// El corpus recorre hoy dos de las seis ramas. Las otras cuatro son descargos:
// una rama de descargo que ninguna entrada alcanza es una rama que NO EXISTE, y
// una mutacion la deja verde porque no hay nada que romper (M47). Con dato
// sintetico siempre hay algo que romper.
//
// # Lo que estas dos funciones costaron
//
// `preaviso` estuvo semanas en el censo declarada como «disponible para el
// corpus: lo que falta es escribir los relojes, no tocar codigo», y era falso.
// El ejecutor de dorados pedia el disparador a toda primitiva que no fuera
// `puntual` ni `continua`, y `validarPreaviso` PROHIBE que un preaviso declare
// disparador. Una guarda pedia exactamente lo que la otra vetaba, asi que
// `hechos[""]` no existia nunca y todo dorado de un preaviso moria igual.
// Salio el 03-09-2026, al escribir el primer paquete que la declara.
func TestDeQueHechoCuelgaCadaRelojYCuandoLaFaltaEsElCaso(t *testing.T) {
	t.Run("de que hecho cuelga", func(t *testing.T) {
		casos := []struct {
			nombre  string
			tmp     Temporalidad
			quiere  string
			exigido bool
		}{
			{"una puntual no cuelga de ningun hecho: la fecha la da la norma",
				Temporalidad{Primitiva: "puntual"}, "", false},
			{"una continua tampoco: es permanente",
				Temporalidad{Primitiva: "continua"}, "", false},
			{"un preaviso cuelga de su EFECTO, que es la fecha que elige el obligado",
				Temporalidad{Primitiva: "preaviso", Efecto: "fecha_de_efecto"},
				"fecha_de_efecto", true},
			{"y no de su disparador, que ademas tiene prohibido declarar",
				Temporalidad{Primitiva: "preaviso", Efecto: "fecha_de_efecto",
					Disparador: map[string]string{"hecho": "no_deberia_estar_aqui"}},
				"fecha_de_efecto", true},
			{"las demas cuelgan del disparador",
				Temporalidad{Primitiva: "plazo",
					Disparador: map[string]string{"hecho": "conocimiento_del_incidente"}},
				"conocimiento_del_incidente", true},
		}
		for _, c := range casos {
			t.Run(c.nombre, func(t *testing.T) {
				nombre, campo, hace := hechoQuePideUnDorado(c.tmp)
				if nombre != c.quiere || hace != c.exigido {
					t.Errorf("ha dicho (%q, %t) y esperaba (%q, %t)",
						nombre, hace, c.quiere, c.exigido)
				}
				if hace && campo == "" {
					t.Error("exige un hecho y no dice como se llama el campo, asi que el " +
						"error mandaria al autor a mirar a ninguna parte")
				}
			})
		}
	})

	t.Run("cuando la falta del dato ES el caso", func(t *testing.T) {
		pendiente := EsperadoDorado{Hito: "aviso", Estado: "pendiente de hecho"}
		determinado := EsperadoDorado{Hito: "aviso", Vence: "2027-04-15T00:00:00Z"}
		casos := []struct {
			nombre string
			d      Dorado
			quiere bool
		}{
			{"todas las filas afirman que el dato no consta: la falta es el caso",
				Dorado{Esperado: []EsperadoDorado{pendiente}}, true},
			{"dos filas y las dos lo afirman",
				Dorado{Esperado: []EsperadoDorado{pendiente, pendiente}}, true},
			// LA MEZCLA NO EXIME, y es la rama que decide: una fila
			// determinada solo puede salir del hecho, asi que el hecho hace
			// falta aunque su vecina diga que no consta.
			{"una determinada y otra pendiente: el hecho sigue haciendo falta",
				Dorado{Esperado: []EsperadoDorado{determinado, pendiente}}, false},
			{"una determinada sola",
				Dorado{Esperado: []EsperadoDorado{determinado}}, false},
			// EL VACIO NUNCA ABRE LA PUERTA. Es el invariante 8 sobre esta
			// frontera: un `esperado` vacio no afirma nada, y la forma
			// degenerada tiene que caer del lado restrictivo.
			{"un esperado vacio no afirma nada, asi que no exime",
				Dorado{Esperado: nil}, false},
			{"ni un estado que no se entiende, que es la tercera forma de la nada",
				Dorado{Esperado: []EsperadoDorado{{Hito: "aviso", Estado: "mas o menos"}}},
				false},
		}
		for _, c := range casos {
			t.Run(c.nombre, func(t *testing.T) {
				if got := afirmaQueFaltaElDato(c.d); got != c.quiere {
					t.Errorf("ha dicho %t y esperaba %t", got, c.quiere)
				}
			})
		}
	})
}
