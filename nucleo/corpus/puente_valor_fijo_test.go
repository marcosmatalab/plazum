package corpus

import "testing"

// LA CUARTA FORMA DEL PUENTE: UN BOOLEANO CUYO «SI» AFIRMA UNA CONSTANTE.
//
// # De donde sale, y por que no se resolvio con el rodeo que ya existia
//
// El frente que iba a declarar el puente en los catorce paquetes que faltan
// PARO el 04-09-2026 y midio: 19 hechos de 8 de los 15 marcos necesitan afirmar
// `predicado(instancia, CONSTANTE)` desde una respuesta de si o no. Con las tres
// formas de entonces, la unica salida era declararlos como enumerado de un solo
// valor, y el linter lo aceptaba: lo comprobo.
//
// LO QUE DESCARTA EL RODEO ES UN DANO CONCRETO Y MEDIDO. Un desplegable de una
// sola opcion que la superficie mande por defecto afirma que quien contesta esta
// designado como entidad financiera, y ese hecho solo enciende 28 obligaciones,
// entre ellas una NOTIFICATORIA ante el supervisor. Un umbral escrito de menos
// en una notificatoria no cuesta horas: provoca una actuacion indebida ante el
// supervisor, y eso no se deshace.
//
// # Lo que hace segura a esta forma, y es una sola cosa
//
// Que hereda de `afirma_si` la propiedad de que UN «NO» NO AFIRMA NADA. El
// valor por defecto de un booleano en un formulario es «no»; el de un
// desplegable es su unica opcion. Ahi esta toda la diferencia, y por eso esta
// forma exige tipo booleano: es su razon de ser, no una restriccion heredada.
//
// # Estos casos existen porque el corpus no los recorre
//
// Hoy ningun paquete declara la cuarta forma, asi que sin estos casos sinteticos
// TODAS sus ramas serian ramas que nadie recorre, y una mutacion las dejaria
// verdes porque no habria nada que romper (M47). Cada rama de rechazo lleva
// aqui su control POSITIVO al lado.

func TestLaCuartaFormaAfirmaLaConstanteDelPaqueteYSoloConUnSi(t *testing.T) {
	// El caso bueno: booleano, predicado que la regla usa con aridad 2, y la
	// constante que la regla prueba en ese hueco.
	bueno := func() *Paquete {
		return conCuartaForma("categoria", "ALTA",
			`aplica("demo.auditoria_bienal", S) :- categoria(S, "ALTA")`)
	}

	t.Run("control positivo: el caso bueno carga", func(t *testing.T) {
		if errs := bueno().Validar(); len(errs) != 0 {
			t.Fatalf("la cuarta forma bien escrita no carga: %v", errs)
		}
	})

	// Y LA MITAD QUE DE VERDAD IMPORTA: que el «si» afirme la constante y el
	// «no» no afirme nada. Es la propiedad por la que existe esta forma, asi
	// que si algun dia deja de cumplirse, el rodeo del desplegable vuelve a ser
	// equivalente y esta forma deja de tener sentido.
	t.Run("un si afirma la constante del paquete", func(t *testing.T) {
		p := bueno()
		hs, err := HechosDeLaEntrevista(p, []RespuestaDeEntrevista{{
			Entidad:   p.Entidades[0].Nombre,
			Instancia: "s1",
			Atributo:  p.Entidades[0].Atributos[0].Nombre,
			Si:        true,
		}})
		if err != nil {
			t.Fatalf("traducir: %v", err)
		}
		if len(hs) != 1 {
			t.Fatalf("un si tenia que afirmar un hecho y salieron %d: %v", len(hs), hs)
		}
		if got := hs[0]; got.Pred != "categoria" || len(got.Args) != 2 ||
			got.Args[0] != "s1" || got.Args[1] != "ALTA" {
			t.Errorf("el hecho no es categoria(s1, ALTA): %+v", got)
		}
	})

	t.Run("un no NO afirma nada, que es toda la razon de ser de esta forma", func(t *testing.T) {
		p := bueno()
		hs, err := HechosDeLaEntrevista(p, []RespuestaDeEntrevista{{
			Entidad:   p.Entidades[0].Nombre,
			Instancia: "s1",
			Atributo:  p.Entidades[0].Atributos[0].Nombre,
			Si:        false,
		}})
		if err != nil {
			t.Fatalf("traducir: %v", err)
		}
		if len(hs) != 0 {
			t.Errorf("un «no» ha afirmado %d hechos: %v.\n"+
				"  Si un «no» afirma, esta forma deja de ser mas segura que el enumerado de "+
				"un solo valor, y entonces no tiene razon de existir: lo unico que aporta es "+
				"que el defecto del formulario no afirme nada.", len(hs), hs)
		}
	})
}

// LAS CUATRO FORMAS DE ESCRIBIRLA MAL, cada una con su centinela.
func TestLaCuartaFormaMalEscritaNoCarga(t *testing.T) {
	casos := []struct {
		nombre    string
		p         *Paquete
		centinela error
	}{
		{"dice afirmar una constante y no dice cual",
			booleanoConPuente(
				&HechoDeAtributo{Forma: PuenteAfirmaSiConValor, Predicado: "categoria"},
				`aplica("demo.auditoria_bienal", S) :- categoria(S, "ALTA")`),
			ErrPuenteSinValorFijo},
		// LA CONSTANTE TIENE QUE SER LA QUE LA REGLA PRUEBA, y aqui la
		// comprobacion es mas dura que en `con_valor`: alli basta con que ALGUNO
		// de los valores del enumerado case, porque un enumerado puede tener un
		// valor que legitimamente apaga; aqui la constante es UNA, asi que si no
		// casa, el «si» del operador no enciende nada.
		{"afirma una constante que ninguna regla prueba",
			conCuartaForma("categoria", "ALTISIMA",
				`aplica("demo.auditoria_bienal", S) :- categoria(S, "ALTA")`),
			ErrPuenteValorHuerfano},
		{"la usa un atributo que no es booleano",
			paqueteConAtributoEnumeradoYCuartaForma(),
			ErrPuenteFormaNoCasaConTipo},
		{"otra forma trae el valor fijo escrito",
			booleanoConPuente(&HechoDeAtributo{
				Forma: PuenteAfirmaSi, Predicado: "auditable", Valor: "ALTA"},
				`aplica("demo.auditoria_bienal", S) :- auditable(S)`),
			ErrPuenteValorFijoSobrante},
		{"el callejon trae el valor fijo escrito",
			booleanoConPuente(&HechoDeAtributo{
				Forma: PuenteNoLlegaAlMotor, Porque: "documenta el alcance y no deriva nada",
				Valor: "ALTA"}),
			ErrPuenteValorFijoSobrante},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			errs := c.p.Validar()
			if !tieneError(errs, c.centinela) {
				t.Errorf("tenia que caerse con %v y dio: %v", c.centinela, errs)
			}
		})
	}
}

// booleanoConPuente es `conPuente` con el atributo convertido en BOOLEANO.
//
// El fixture compartido trae un enumerado, que es lo que necesitan las otras
// tres formas; esta cuarta EXIGE booleano, y esa exigencia es su razon de ser y
// no un detalle del fixture. Se hace aqui una vez y no en cada caso para que la
// tabla siga leyendose.
func booleanoConPuente(h *HechoDeAtributo, reglas ...string) *Paquete {
	p := conPuente(h, reglas...)
	p.Entidades[0].Atributos[0].Tipo = Booleano
	p.Entidades[0].Atributos[0].Valores = nil
	return p
}

// conCuartaForma es el caso corriente de esta forma: booleano, predicado y la
// constante que su «si» afirma.
func conCuartaForma(pred, valor string, reglas ...string) *Paquete {
	return booleanoConPuente(&HechoDeAtributo{
		Forma: PuenteAfirmaSiConValor, Predicado: pred, Valor: valor}, reglas...)
}

// paqueteConAtributoEnumeradoYCuartaForma construye el caso del tipo
// equivocado: el fixture tal cual viene, que es enumerado.
func paqueteConAtributoEnumeradoYCuartaForma() *Paquete {
	return conPuente(&HechoDeAtributo{
		Forma: PuenteAfirmaSiConValor, Predicado: "categoria", Valor: "ALTA"},
		`aplica("demo.auditoria_bienal", S) :- categoria(S, "ALTA")`)
}
