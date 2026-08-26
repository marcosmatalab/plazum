package export

import (
	"strings"
	"testing"
	"time"

	"plazum/nucleo/estado"
	"plazum/nucleo/expediente"
	"plazum/nucleo/ledger"
)

// Las puertas hostiles del export. Lo que se ataca aqui no es el codigo, son
// tres propiedades que el fichero promete al que lo recibe:
//
//  1. lo que el borrado legal borro no reaparece;
//  2. lo que nadie ha revisado no sale (ni tokens, ni texto libre de terceros);
//  3. el receptor puede deduplicar sin contar el mismo hecho dos veces.

// centinelas son valores que SOLO existen dentro del contenido cifrado de la
// entrada que se va a suprimir. Ninguno aparece en el expediente por otro
// camino, asi que encontrarlos en el fichero significa exactamente una cosa.
var centinelas = []string{
	"CENTINELA-RECURSO-DEL-INTERESADO",
	"CENTINELA-RECOLECTOR",
	"CENTINELA-HASH-DE-CARGA",
	"CENTINELA-VERSION",
}

func cargaConCentinelas(t *testing.T) []byte {
	t.Helper()
	return carga(t, estado.Observacion{
		Prueba:      "control.uno",
		Recurso:     centinelas[0],
		Satisfecho:  false,
		Recolectada: comoEstaba().Add(-3 * time.Hour),
		Recolector:  centinelas[1],
		HashCarga:   centinelas[2],
		Version:     centinelas[3],
	})
}

// LA PUERTA DE LA CASILLA.
//
// Una entrada con lapida no puede sacar su contenido a un fichero de texto plano
// que se manda a un SIEM de terceros. Y el caso que importa no es el comodo (la
// clave ya no esta, asi que no habria nada que sacar): es el de DERIVA, donde la
// lapida esta y la clave sigue por ahi. Pasa de verdad, porque los dos actos
// viven en almacenes distintos con retenciones distintas: una copia de seguridad
// del keystore restaurada dentro de su ventana devuelve una clave destruida.
//
// LA MUTACION que vigila esto: quitar la guarda de contenidoDe (las tres lineas
// que preguntan por la lapida ANTES de mirar la clave). Con la guarda fuera, el
// contenido de la entrada suprimida sale entero, y este test se pone rojo.
//
// EL CONTROL NEGATIVO va dentro, en el segundo subtest: sin lapida, esos mismos
// valores SI salen. Sin el, el verde de arriba seria compatible con un export
// que no sacara nunca nada, que pasaria por vacuidad.
func TestElExportNoFiltraLoQueElBorradoLegalBorro(t *testing.T) {
	t.Run("con lapida, el contenido no sale aunque la clave siga divulgada", func(t *testing.T) {
		e := nuevoEscenario(t, cargaConCentinelas(t),
			carga(t, obsDePrueba("control.dos", "recurso-que-si-sale", true)))
		e.suprime(t, 0, "urn:demo:ley art. 17", "2026-09-18T07:30:00Z", "control.uno")

		// La deriva, dicha en voz alta: la clave de la entrada suprimida SIGUE
		// divulgada en el expediente. Si el test la quitara, estaria probando
		// el caso comodo y la guarda podria no existir.
		if _, hay := e.exp.ClavesEntradas[0]; !hay {
			t.Fatal("este test necesita que la clave de la entrada suprimida siga estando: " +
				"sin ella no hay nada que la guarda pueda dejar escapar y el verde no " +
				"demostraria nada")
		}

		salida, res := exportar(t, e.exp)
		for _, c := range centinelas {
			if strings.Contains(salida, c) {
				t.Errorf("el contenido de una entrada BORRADA CON BASE LEGAL ha salido al "+
					"fichero que se manda al SIEM: %q.\n"+
					"  El SIEM es un tercero con retencion propia: lo que entre ahi ya no lo "+
					"alcanza ninguna orden de supresion nuestra.\n"+
					"  Causa tipica: la guarda de contenidoDe pregunta por la clave en vez de "+
					"por la lapida, o lo pregunta despues.\n--- fichero ---\n%s", c, salida)
			}
		}

		// Y lo que SI tiene que salir: el hecho de la supresion, con su base
		// legal. Un borrado mudo deja al receptor sin saber que ese control se
		// quedo sin apoyo.
		if res.Suprimidas != 1 {
			t.Fatalf("el resumen cuenta %d supresiones y tenia que contar 1", res.Suprimidas)
		}
		var vistaSupresion, vistaEntrada bool
		for _, m := range lineas(t, salida) {
			switch m["event.action"] {
			case AccionSupresion:
				vistaSupresion = true
				if m["plazum.base_legal"] != "urn:demo:ley art. 17" {
					t.Errorf("la supresion sale sin su base legal: %v", m)
				}
				if m["plazum.prueba"] != "control.uno" {
					t.Errorf("la supresion no dice que control se queda sin evidencia: %v", m)
				}
			case AccionEntrada:
				if m["plazum.entrada"] != float64(0) {
					continue
				}
				vistaEntrada = true
				if m["plazum.contenido_disponible"] != false {
					t.Errorf("la entrada suprimida no se marca como sin contenido: %v", m)
				}
				if s, _ := m["plazum.motivo_sin_contenido"].(string); !strings.Contains(s, "base legal") {
					t.Errorf("la entrada suprimida no dice por que no trae contenido: %v", m)
				}
			}
		}
		if !vistaSupresion || !vistaEntrada {
			t.Fatalf("faltan eventos: supresion=%t entrada=%t. Si el export dejara de emitir "+
				"la entrada suprimida, la comprobacion de arriba pasaria sola",
				vistaSupresion, vistaEntrada)
		}
	})

	t.Run("CONTROL NEGATIVO: sin lapida, ese mismo contenido SI sale", func(t *testing.T) {
		e := nuevoEscenario(t, cargaConCentinelas(t),
			carga(t, obsDePrueba("control.dos", "recurso-que-si-sale", true)))
		salida, res := exportar(t, e.exp)
		if res.Suprimidas != 0 {
			t.Fatalf("el control negativo no puede tener supresiones y tiene %d", res.Suprimidas)
		}
		for _, c := range centinelas {
			if !strings.Contains(salida, c) {
				t.Fatalf("sin nada suprimido, el valor %q NO sale al fichero.\n"+
					"  Entonces el subtest de arriba pasa por vacuidad: no demuestra que la "+
					"guarda filtre, solo que el export no saca ese campo nunca.\n"+
					"--- fichero ---\n%s", c, salida)
			}
		}
	})
}

// El borrado a medias, que es la forma en que esto se rompe de verdad.
//
// Suprimir son DOS escrituras en sitios distintos: la lapida firmada dentro de
// la cadena y la atribucion en el expediente (que control se queda sin
// evidencia). Entre las dos cabe un fallo. Un expediente que declara la
// supresion y no trae lapida no verifica, pero eso lo dice `plazum verify`
// DESPUES, y exportar ese contenido a un tercero no se deshace.
//
// La regla: ante la duda, no sale. Retener de mas es un evento con menos campos;
// filtrar de mas no tiene vuelta.
func TestUnBorradoADemiasTampocoDejaSalirElContenido(t *testing.T) {
	e := nuevoEscenario(t, cargaConCentinelas(t))
	// La atribucion SIN la lapida: la primera de las dos escrituras fallo.
	e.exp.SupresionesDeEvidencia = []expediente.SupresionDeEvidencia{
		{Entrada: 0, Prueba: "control.uno"},
	}
	if len(e.exp.Cadena.Lapidas) != 0 {
		t.Fatal("este test necesita que NO haya lapida: con ella estaria probando el " +
			"camino de al lado")
	}
	salida, _ := exportar(t, e.exp)
	for _, c := range centinelas {
		if strings.Contains(salida, c) {
			t.Errorf("con la supresion declarada y la lapida sin escribir, el contenido "+
				"sale igual: %q", c)
		}
	}
	if !strings.Contains(salida, "no hay lapida que lo respalde") {
		t.Errorf("no se dice que la supresion declarada no tiene lapida detras. El "+
			"receptor tiene que poder distinguir esto de un borrado completo:\n%s", salida)
	}
}

// El identificador del evento no puede depender del contenido.
//
// Si dependiera, la misma entrada tendria un id antes del borrado y otro
// despues, y el SIEM contaria el mismo hecho dos veces: primero como entrada con
// contenido, luego como entrada nueva sin el. Un panel de auditoria que duplica
// hechos deja de servir para contar nada.
func TestElIdentificadorDeUnaEntradaNoCambiaAlSuprimirla(t *testing.T) {
	antes := nuevoEscenario(t, cargaConCentinelas(t))
	salidaAntes, _ := exportar(t, antes.exp)

	despues := nuevoEscenario(t, cargaConCentinelas(t))
	despues.suprime(t, 0, "urn:demo:ley art. 17", "2026-09-18T07:30:00Z", "control.uno")
	salidaDespues, _ := exportar(t, despues.exp)

	id := func(s string) string {
		for _, m := range lineas(t, s) {
			if m["event.action"] == AccionEntrada && m["plazum.entrada"] == float64(0) {
				v, _ := m["event.id"].(string)
				return v
			}
		}
		return ""
	}
	a, d := id(salidaAntes), id(salidaDespues)
	if a == "" || d == "" {
		t.Fatalf("no se ha encontrado el evento de la entrada 0 en uno de los dos ficheros "+
			"(antes=%q despues=%q)", a, d)
	}
	if a != d {
		t.Errorf("el identificador de la entrada 0 cambia al suprimirla (%s -> %s). "+
			"El receptor contaria el mismo hecho dos veces", a, d)
	}
}

// La lista blanca: lo que nadie ha revisado no sale.
//
// Un token en un log es un token comprometido, tambien cuando es el equivocado.
// Con lista negra habria que acordarse de prohibir cada nombre nuevo; con lista
// blanca, el campo que nadie ha mirado se queda fuera solo.
func TestNiTokensNiSecretosLleganAlSIEM(t *testing.T) {
	const secreto = "CENTINELA-VALOR-QUE-NO-DEBE-SALIR"
	sospechosos := []string{"token", "Authorization", "password", "secreto", "api_key",
		"cookie", "clave_privada"}

	t.Run("una clave desconocida no sale, y se cuenta", func(t *testing.T) {
		m := map[string]any{"Prueba": "control.uno"}
		for _, k := range sospechosos {
			m[k] = secreto
		}
		e := nuevoEscenario(t, carga(t, m))
		salida, _ := exportar(t, e.exp)
		if strings.Contains(salida, secreto) {
			t.Errorf("un valor de una clave que no esta en la lista blanca ha salido al "+
				"fichero:\n%s", salida)
		}
		hay := false
		for _, ev := range lineas(t, salida) {
			n, ok := ev["plazum.campos_omitidos"].(float64)
			if !ok {
				continue
			}
			hay = true
			if int(n) != len(sospechosos) {
				t.Errorf("se han omitido %v campos y habia %d desconocidos. Si el recuento "+
					"no cuadra, alguno se estaria proyectando por otro camino",
					n, len(sospechosos))
			}
		}
		if !hay {
			t.Error("ningun evento dice cuantos campos se omitieron. Sin ese recuento, un " +
				"export que dejara de leer la carga entera daria el mismo verde")
		}
	})

	t.Run("CONTROL NEGATIVO: ese mismo valor bajo una clave conocida SI sale", func(t *testing.T) {
		e := nuevoEscenario(t, carga(t, map[string]any{"sujeto": secreto}))
		salida, _ := exportar(t, e.exp)
		if !strings.Contains(salida, secreto) {
			t.Fatalf("bajo una clave de la lista blanca el valor tampoco sale, asi que la "+
				"comprobacion de arriba no demuestra que la lista blanca filtre: demuestra "+
				"que este export no saca nada.\n%s", salida)
		}
	})
}

// El texto de un error de recoleccion NO sale nunca, y si el hecho de que lo
// hubo. Es el caso concreto que justifica la lista blanca: ese campo lo escribe
// un tercero y ahi es donde acaba una URL firmada o una cabecera con credencial
// cuando un recolector falla.
func TestElTextoDeUnErrorDeRecoleccionNoSaleYSiElHechoDeQueLoHubo(t *testing.T) {
	const dentro = "CENTINELA-401-Bearer-abcdef0123456789"
	o := obsDePrueba("control.uno", "recurso-a", false)
	o.ErrorRecol = dentro
	e := nuevoEscenario(t, carga(t, o))
	salida, _ := exportar(t, e.exp)

	if strings.Contains(salida, dentro) {
		t.Errorf("el texto del error del recolector ha salido al fichero:\n%s", salida)
	}
	visto := false
	for _, m := range lineas(t, salida) {
		if m["plazum.error_de_recoleccion"] == true {
			visto = true
		}
	}
	if !visto {
		t.Error("tampoco sale el hecho de que hubo error. El SIEM necesita poder alertar " +
			"de un recolector que lleva un mes fallando; lo que no necesita es su texto")
	}

	// CONTROL NEGATIVO: sin error, la marca NO aparece. Una marca que estuviera
	// siempre no distinguiria nada.
	sin := nuevoEscenario(t, carga(t, obsDePrueba("control.uno", "recurso-a", true)))
	salidaSin, _ := exportar(t, sin.exp)
	if strings.Contains(salidaSin, "plazum.error_de_recoleccion") {
		t.Errorf("sin error de recoleccion la marca sale igual, asi que no significa "+
			"nada:\n%s", salidaSin)
	}
}

// Las claves por entrada que el emisor divulga en el expediente son material
// criptografico. Que viajen dentro del expediente es su funcion (el receptor
// abre la cadena con ellas); que salgan en un log de texto plano hacia un
// tercero no lo es.
func TestLasClavesDivulgadasNoSalenEnNingunEvento(t *testing.T) {
	e := nuevoEscenario(t, carga(t, obsDePrueba("control.uno", "recurso-a", true)))
	salida, _ := exportar(t, e.exp)
	for i, k := range e.exp.ClavesEntradas {
		if strings.Contains(salida, k) {
			t.Errorf("la clave de la entrada %d aparece en el log de auditoria:\n%s", i, salida)
		}
	}

	// CONTROL NEGATIVO de la busqueda: la misma cadena, puesta en un campo de la
	// lista blanca, SI se encuentra. Sin esto, un fallo del buscador pasaria por
	// una ausencia de claves.
	k := e.exp.ClavesEntradas[0]
	otro := nuevoEscenario(t, carga(t, map[string]any{"sujeto": k}))
	salidaOtro, _ := exportar(t, otro.exp)
	if !strings.Contains(salidaOtro, k) {
		t.Fatal("la busqueda no encuentra la cadena ni cuando esta puesta a proposito: " +
			"la comprobacion de arriba no vigila nada")
	}
}

// Una lapida puede senalar a una entrada que no existe: la cadena rechaza esa
// combinacion al verificar, pero el export corre sobre ficheros que todavia no
// ha verificado nadie, y un panico aqui deja al operador sin log y sin motivo.
func TestUnaLapidaQueSenalaFueraDeRangoNoTumbaElExport(t *testing.T) {
	e := nuevoEscenario(t, carga(t, obsDePrueba("control.uno", "recurso-a", true)))
	e.exp.Cadena.Lapidas = append(e.exp.Cadena.Lapidas, ledger.Lapida{
		EntradaBorrada: 99,
		BaseLegal:      "urn:demo:ley art. 17",
		Instante:       "2026-09-18T07:30:00Z",
	})
	salida, res := exportar(t, e.exp)
	if res.Suprimidas != 1 {
		t.Errorf("una lapida fuera de rango tiene que salir igual como supresion "+
			"declarada: el receptor tiene que verla. Contadas: %d", res.Suprimidas)
	}
	if !strings.Contains(salida, AccionSupresion) {
		t.Error("no se ha emitido el evento de supresion")
	}
}

// Dos lapidas para la misma entrada: la cadena lo rechaza al verificar, pero el
// contenido tiene que seguir sin salir. Es la comprobacion de que la guarda mira
// el conjunto de indices suprimidos y no la primera lapida que encuentra.
func TestDosLapidasParaLaMismaEntradaSiguenTapandoElContenido(t *testing.T) {
	e := nuevoEscenario(t, cargaConCentinelas(t))
	e.suprime(t, 0, "urn:demo:ley art. 17", "2026-09-18T07:30:00Z", "control.uno")
	dup := e.exp.Cadena.Lapidas[0]
	dup.BaseLegal = "urn:demo:ley art. 18"
	e.exp.Cadena.Lapidas = append(e.exp.Cadena.Lapidas, dup)

	salida, _ := exportar(t, e.exp)
	for _, c := range centinelas {
		if strings.Contains(salida, c) {
			t.Errorf("con dos lapidas sobre la misma entrada el contenido vuelve a salir: %q", c)
		}
	}
}

// Una entrada sin clave y sin lapida no puede salir en blanco y callada: eso es
// contenido oculto sin decir por que, que es justo lo que el verificador del
// expediente marca como discrepancia.
func TestUnaEntradaSinClaveNiLapidaLoDice(t *testing.T) {
	e := nuevoEscenario(t, carga(t, obsDePrueba("control.uno", "recurso-a", true)))
	delete(e.exp.ClavesEntradas, 0)
	salida, _ := exportar(t, e.exp)
	visto := false
	for _, m := range lineas(t, salida) {
		if m["event.action"] != AccionEntrada {
			continue
		}
		visto = true
		if m["plazum.contenido_disponible"] != false {
			t.Errorf("la entrada sin clave se marca como si trajera contenido: %v", m)
		}
		s, _ := m["plazum.motivo_sin_contenido"].(string)
		if !strings.Contains(s, "no divulga la clave") {
			t.Errorf("no dice que el emisor no divulga la clave: %v", m)
		}
	}
	if !visto {
		t.Fatal("no ha salido el evento de la entrada")
	}
}

// Una clave que no abre la entrada (sustituida, o de otra cadena) no puede
// producir contenido inventado ni tumbar el export.
func TestUnaClaveQueNoAbreLaEntradaNoProduceContenido(t *testing.T) {
	e := nuevoEscenario(t, cargaConCentinelas(t))
	e.exp.ClavesEntradas[0] = strings.Repeat("ab", 32)
	salida, _ := exportar(t, e.exp)
	for _, c := range centinelas {
		if strings.Contains(salida, c) {
			t.Errorf("con una clave sustituida ha salido contenido: %q", c)
		}
	}
	if !strings.Contains(salida, "no abre esta entrada") {
		t.Errorf("no se dice que la clave divulgada no abre la entrada:\n%s", salida)
	}
}

// El expediente que se exporta puede venir de fuera. Un campo declarado con
// texto arbitrario del emisor no puede partir una linea en dos, porque un
// evento partido es un evento que el receptor no ve.
func TestUnCampoDeclaradoConSaltosDeLineaNoParteElFichero(t *testing.T) {
	e := nuevoEscenario(t, carga(t, obsDePrueba("control.uno", "recurso-a", true)))
	e.exp.Estados = []expediente.EstadoControl{{
		Prueba: "control\n{\"event.action\":\"inyectado\"}",
		Estado: estado.Pass.String(),
	}}
	salida, _ := exportar(t, e.exp)
	for _, m := range lineas(t, salida) {
		if m["event.action"] == "inyectado" {
			t.Fatalf("un salto de linea en un campo declarado ha metido un evento falso "+
				"en el fichero:\n%s", salida)
		}
	}
}
