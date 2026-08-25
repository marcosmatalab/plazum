package aplicabilidad

import (
	"fmt"
	"testing"
)

// Programas de prueba con la FORMA de normas reales, pero con identificadores
// sinteticos del espacio urn:demo: y demo.: el nucleo es autonomo y no conoce
// el directorio del corpus, asi que ningun test de nucleo/ puede depender de
// un URN real de paquetes/. Cada regla conserva su cita, como exige el formato
// de paquete: una regla de aplicabilidad sin articulo no se acepta, y la cita
// es ademas la que dice de donde sale la forma que se esta probando.

// Forma: la categoria de un sistema es el MAXIMO del nivel de cada dimension
// (confidencialidad, integridad, disponibilidad...) sobre cada informacion y
// servicio que maneja. Es la del calculo de categoria de un esquema nacional
// de seguridad, RD 311/2022 art. 40 y Anexo I.
func programaAgregacionPorMaximo() Programa {
	// categoria se EXPORTA porque es a la vez un dato que el sujeto puede
	// declarar y uno que esta norma deriva: tiene que ser el MISMO predicado en
	// los dos caminos. nivel_max no, que es un paso intermedio suyo.
	return Programa{Paquete: "urn:demo:agregada", Exporta: []string{"categoria"}, Reglas: []Regla{
		// Esto es agregacion: un selector plano no puede calcularlo.
		// Valores REALES de la escala del RD 311/2022, no "1", "2", "3". Con
		// orden lexicografico esto devolveria MEDIO, que es el bug que la
		// revision destapo. La escala la declara el paquete.
		{ID: "categoria_por_maximo", Cita: "RD 311/2022 art. 40 y Anexo I",
			Cabeza:   A("nivel_max", V("S"), V("_AGG")),
			Cuerpo:   []Atomo{A("maneja", V("S"), V("I")), A("nivel_dimension", V("I"), V("D"), V("N"))},
			Agregado: Maximo, SobreVar: "N",
			Escala: Escala{Nombre: "demo.niveles", Orden: []string{"BAJO", "MEDIO", "ALTO"}}},
		{ID: "categoria_basica", Cita: "RD 311/2022 Anexo I",
			Cabeza: A("categoria", V("S"), C("BASICA")),
			Cuerpo: []Atomo{A("nivel_max", V("S"), C("BAJO"))}},
		{ID: "categoria_media", Cita: "RD 311/2022 Anexo I",
			Cabeza: A("categoria", V("S"), C("MEDIA")),
			Cuerpo: []Atomo{A("nivel_max", V("S"), C("MEDIO"))}},
		{ID: "categoria_alta", Cita: "RD 311/2022 Anexo I",
			Cabeza: A("categoria", V("S"), C("ALTA")),
			Cuerpo: []Atomo{A("nivel_max", V("S"), C("ALTO"))}},
		// La auditoria bienal del art. 31 solo alcanza a MEDIA y ALTA.
		{ID: "auditoria_bienal_media", Cita: "RD 311/2022 art. 31",
			Cabeza: A("aplica", C("demo.auditoria_bienal"), V("S")),
			Cuerpo: []Atomo{A("categoria", V("S"), C("MEDIA"))}},
		{ID: "auditoria_bienal_alta", Cita: "RD 311/2022 art. 31",
			Cabeza: A("aplica", C("demo.auditoria_bienal"), V("S")),
			Cuerpo: []Atomo{A("categoria", V("S"), C("ALTA"))}},
		// En BASICA basta autoevaluacion, y se dice explicitamente.
		{ID: "autoevaluacion_basica", Cita: "RD 311/2022 art. 38",
			Cabeza: A("aplica", C("demo.autoevaluacion"), V("S")),
			Cuerpo: []Atomo{A("categoria", V("S"), C("BASICA"))}},
	}}
}

// Forma: clasificacion por sector y tamano, y arrastre contractual por la
// cadena de suministro. Es la de la Directiva 2022/2555 (art. 2, 3 y 21.2.d).
func programaCadenaDeProveedores() Programa {
	// clase se EXPORTA: es una clasificacion que otras normas encadenan (una
	// entidad esencial arrastra obligaciones de otros marcos). en_ambito y
	// proveedor_de no: son el razonamiento interno de esta norma.
	return Programa{Paquete: "urn:demo:cadena-proveedores", Exporta: []string{"clase"}, Reglas: []Regla{
		// Cierre transitivo de la cadena de suministro: el art. 21.2.d alcanza
		// a los proveedores directos de una entidad en ambito. Un selector
		// sobre una entidad plana no puede recorrer un grafo.
		{ID: "en_ambito_anexo1", Cita: "Directiva 2022/2555 art. 2 y Anexo I",
			Cabeza: A("en_ambito", V("E")),
			Cuerpo: []Atomo{A("sector_anexo1", V("E"), Anon()), A("supera_umbral_tamano", V("E"))}},
		{ID: "esencial", Cita: "Directiva 2022/2555 art. 3.1",
			Cabeza: A("clase", V("E"), C("esencial")),
			Cuerpo: []Atomo{A("sector_anexo1", V("E"), Anon()), A("supera_techo_mediana", V("E"))}},
		{ID: "importante", Cita: "Directiva 2022/2555 art. 3.2",
			Cabeza:  A("clase", V("E"), C("importante")),
			Cuerpo:  []Atomo{A("en_ambito", V("E"))},
			Negados: []Atomo{A("supera_techo_mediana", V("E"))}},
		// Proveedor directo de una entidad en ambito: obligacion contractual.
		{ID: "proveedor_alcanzado", Cita: "Directiva 2022/2555 art. 21.2.d",
			Cabeza: A("aplica", C("demo.cadena_proveedores"), V("P")),
			Cuerpo: []Atomo{A("proveedor_de", V("P"), V("E")), A("en_ambito", V("E"))}},
		// El cierre transitivo se hace sobre un predicado PROPIO derivado del
		// hecho, no sobre el hecho.
		//
		// Antes la regla era provee_a(A,C) :- provee_a(A,B), provee_a(B,C), o
		// sea que el paquete REDEFINIA el predicado con el que el sujeto declara
		// sus contratos. Con el espacio de nombres eso deja de funcionar en
		// silencio: provee_a pasaria a ser propio del paquete y los hechos del
		// sujeto no lo alimentarian. Y el fallo de fondo no es de nombres, es de
		// modelado: los hechos son del sujeto y el paquete razona sobre ellos,
		// no los reescribe. Lo caza comprobarHechosContraLocales.
		{ID: "proveedor_directo", Cita: "Directiva 2022/2555 art. 21.2.d",
			Cabeza: A("proveedor_de", V("A"), V("B")),
			Cuerpo: []Atomo{A("provee_a", V("A"), V("B"))}},
		{ID: "cadena_transitiva", Cita: "Directiva 2022/2555 art. 21.2.d y considerando 85",
			Cabeza: A("proveedor_de", V("A"), V("C")),
			Cuerpo: []Atomo{A("provee_a", V("A"), V("B")), A("proveedor_de", V("B"), V("C"))}},
	}}
}

// Forma: una obligacion general con una exencion de requisitos ACUMULATIVOS,
// que solo se puede modelar con negacion estratificada. Es la del RGPD
// (art. 30 y 30.5 para el registro, art. 37.1 para la figura designada).
func programaExencionAcumulativa() Programa {
	return Programa{Paquete: "urn:demo:exencion-acumulativa", Reglas: []Regla{
		// Encadenamiento entre normas: la cabeza de una regla de este paquete
		// es el cuerpo de una regla del paquete de abajo, que en el corpus real
		// es otra norma (la ley nacional que desarrolla el reglamento europeo).
		// Sin punto fijo esto no se puede.
		{ID: "dpd_por_observacion", Cita: "RGPD art. 37.1.b",
			Cabeza: A("aplica", C("demo.designacion_responsable"), V("E")),
			Cuerpo: []Atomo{A("observacion_sistematica_gran_escala", V("E"))}},
		{ID: "dpd_por_categorias_especiales", Cita: "RGPD art. 37.1.c",
			Cabeza: A("aplica", C("demo.designacion_responsable"), V("E")),
			Cuerpo: []Atomo{A("trata_art9_gran_escala", V("E"))}},
		// Exencion del registro de actividades: los cuatro requisitos del
		// art. 30.5 son ACUMULATIVOS, y esto se modela con negacion.
		{ID: "rat_obligatorio", Cita: "RGPD art. 30",
			Cabeza:  A("aplica", C("demo.registro_actividades"), V("E")),
			Cuerpo:  []Atomo{A("responsable", V("E"))},
			Negados: []Atomo{A("exento_art305", V("E"))}},
		{ID: "exencion_art305", Cita: "RGPD art. 30.5",
			Cabeza:  A("exento_art305", V("E")),
			Cuerpo:  []Atomo{A("menos_de_250_empleados", V("E")), A("tratamiento_ocasional", V("E"))},
			Negados: []Atomo{A("trata_art9_gran_escala", V("E")), A("riesgo_para_derechos", V("E"))}},
	}}
}

// Forma: una norma nacional cuya obligacion se dispara con lo que deriva otra
// norma. Es la de la LO 3/2018 art. 34.3, que engancha con el art. 37 del RGPD.
func programaEncadenada() Programa {
	return Programa{Paquete: "urn:demo:encadenada", Reglas: []Regla{
		// Depende del predicado que deriva el paquete de arriba.
		{ID: "comunicacion_dpd", Cita: "LO 3/2018 art. 34.3",
			Cabeza: A("aplica", C("demo.comunicacion_a_la_autoridad"), V("E")),
			Cuerpo: []Atomo{A("aplica", C("demo.designacion_responsable"), V("E"))}},
	}}
}

func cargar(t *testing.T, m *Motor, p Programa) {
	t.Helper()
	if err := m.Cargar(p); err != nil {
		t.Fatalf("el paquete %s no pasa el linter: %v", p.Paquete, err)
	}
}

func TestCategoriaPorAgregacionDeMaximos(t *testing.T) {
	m := NuevoMotor()
	cargar(t, m, programaAgregacionPorMaximo())
	// Un sistema con dos informaciones: una de nivel bajo y otra de nivel alto.
	m.Afirmar(H("maneja", "sede-electronica", "datos-padron"))
	m.Afirmar(H("maneja", "sede-electronica", "expedientes-sancionadores"))
	m.Afirmar(H("nivel_dimension", "datos-padron", "confidencialidad", "BAJO"))
	m.Afirmar(H("nivel_dimension", "datos-padron", "integridad", "MEDIO"))
	m.Afirmar(H("nivel_dimension", "expedientes-sancionadores", "confidencialidad", "ALTO"))

	if _, err := m.Evaluar(); err != nil {
		t.Fatal(err)
	}
	cats := m.Consultar(A("categoria", C("sede-electronica"), V("C")))
	if len(cats) != 1 || cats[0].Args[1] != "ALTA" {
		t.Fatalf("la categoria es el maximo de las dimensiones, esperaba ALTA, obtuve %v", cats)
	}
	ap := m.Consultar(A("aplica", V("O"), C("sede-electronica")))
	var ids []string
	for _, h := range ap {
		ids = append(ids, h.Args[0])
	}
	if len(ids) != 1 || ids[0] != "demo.auditoria_bienal" {
		t.Fatalf("en ALTA aplica auditoria y NO autoevaluacion, obtuve %v", ids)
	}
	if got := m.Explicar(H("categoria", "sede-electronica", "ALTA")); got == "" {
		t.Fatal("toda derivacion tiene que poder explicarse")
	}
}

func TestCadenaDeSuministroTransitiva(t *testing.T) {
	m := NuevoMotor()
	cargar(t, m, programaCadenaDeProveedores())
	m.Afirmar(H("sector_anexo1", "hospital-x", "salud"))
	m.Afirmar(H("supera_umbral_tamano", "hospital-x"))
	m.Afirmar(H("supera_techo_mediana", "hospital-x"))
	m.Afirmar(H("provee_a", "software-his", "hospital-x"))
	m.Afirmar(H("provee_a", "hosting-y", "software-his"))
	m.Afirmar(H("provee_a", "backup-z", "hosting-y"))

	if _, err := m.Evaluar(); err != nil {
		t.Fatal(err)
	}
	alc := m.Consultar(A("aplica", C("demo.cadena_proveedores"), V("P")))
	got := map[string]bool{}
	for _, h := range alc {
		got[h.Args[1]] = true
	}
	for _, esperado := range []string{"software-his", "hosting-y", "backup-z"} {
		if !got[esperado] {
			t.Fatalf("la cadena es transitiva: falta %s en %v", esperado, got)
		}
	}
	cl := m.Consultar(A("clase", C("hospital-x"), V("C")))
	if len(cl) != 1 || cl[0].Args[1] != "esencial" {
		t.Fatalf("esperaba esencial, obtuve %v", cl)
	}
}

func TestNegacionEstratificadaConExencionAcumulativa(t *testing.T) {
	// Caso 1: pyme que cumple los cuatro requisitos del art. 30.5 -> exenta.
	m := NuevoMotor()
	cargar(t, m, programaExencionAcumulativa())
	m.Afirmar(H("responsable", "pyme"))
	m.Afirmar(H("menos_de_250_empleados", "pyme"))
	m.Afirmar(H("tratamiento_ocasional", "pyme"))
	if _, err := m.Evaluar(); err != nil {
		t.Fatal(err)
	}
	if len(m.Consultar(A("aplica", C("demo.registro_actividades"), C("pyme")))) != 0 {
		t.Fatal("con los cuatro requisitos del art. 30.5 la exencion opera")
	}

	// Caso 2: la misma pyme trata datos del art. 9 -> deja de estar exenta.
	m2 := NuevoMotor()
	cargar(t, m2, programaExencionAcumulativa())
	cargar(t, m2, programaEncadenada())
	m2.Afirmar(H("responsable", "pyme"))
	m2.Afirmar(H("menos_de_250_empleados", "pyme"))
	m2.Afirmar(H("tratamiento_ocasional", "pyme"))
	m2.Afirmar(H("trata_art9_gran_escala", "pyme"))
	if _, err := m2.Evaluar(); err != nil {
		t.Fatal(err)
	}
	if len(m2.Consultar(A("aplica", C("demo.registro_actividades"), C("pyme")))) != 1 {
		t.Fatal("los cuatro requisitos son acumulativos: tratar art. 9 rompe la exencion")
	}
	// Y el encadenamiento entre paquetes, que en el corpus real es el del
	// art. 37 del reglamento europeo con el art. 34.3 de la ley nacional.
	if len(m2.Consultar(A("aplica", C("demo.comunicacion_a_la_autoridad"), C("pyme")))) != 1 {
		t.Fatal("si aplica la designacion del art. 37, aplica la comunicacion del art. 34.3")
	}
}

// Un paquete de corpus es codigo de un tercero. El motor tiene que rechazar
// lo que no termina, no colgarse.
func TestRechazaProgramaNoEstratificable(t *testing.T) {
	m := NuevoMotor()
	m.CargarSinValidar(Programa{Paquete: "malicioso", Reglas: []Regla{
		{ID: "ciclo_con_negacion",
			Cabeza:  A("p", V("X")),
			Cuerpo:  []Atomo{A("base", V("X"))},
			Negados: []Atomo{A("q", V("X"))}},
		{ID: "ciclo_inverso",
			Cabeza:  A("q", V("X")),
			Cuerpo:  []Atomo{A("base", V("X"))},
			Negados: []Atomo{A("p", V("X"))}},
	}})
	m.Afirmar(H("base", "a"))
	if _, err := m.Evaluar(); err == nil {
		t.Fatal("un programa con negacion en un ciclo tiene que rechazarse, no evaluarse")
	}
}

// Las 30 normas se anaden como datos: cargar un paquete nuevo no toca el nucleo.
func TestAnadirUnaNormaEsAnadirUnPrograma(t *testing.T) {
	m := NuevoMotor()
	cargar(t, m, programaAgregacionPorMaximo())
	cargar(t, m, programaCadenaDeProveedores())
	cargar(t, m, programaExencionAcumulativa())
	cargar(t, m, programaEncadenada())
	// Y una quinta norma, la de obligaciones del fabricante de producto
	// digital (Rgto. 2024/2847), escrita aqui mismo en 6 lineas.
	cargar(t, m, Programa{Paquete: "urn:demo:quinta", Reglas: []Regla{
		{ID: "fabricante_en_ambito", Cita: "Rgto. 2024/2847 art. 2 y 13",
			Cabeza: A("aplica", C("demo.obligaciones_fabricante"), V("E")),
			Cuerpo: []Atomo{A("comercializa_producto_digital", V("E")), A("actividad_comercial", V("E"))}},
		{ID: "reporte_desde_vigencia", Cita: "Rgto. 2024/2847 art. 14 y 71",
			Cabeza: A("aplica", C("demo.notificacion"), V("E")),
			Cuerpo: []Atomo{A("aplica", C("demo.obligaciones_fabricante"), V("E"))}},
	}})
	m.Afirmar(H("responsable", "acme"))
	m.Afirmar(H("comercializa_producto_digital", "acme"))
	m.Afirmar(H("actividad_comercial", "acme"))
	m.Afirmar(H("observacion_sistematica_gran_escala", "acme"))
	n, err := m.Evaluar()
	if err != nil {
		t.Fatal(err)
	}
	ap := m.Consultar(A("aplica", V("O"), C("acme")))
	if len(ap) < 4 {
		t.Fatalf("esperaba obligaciones de al menos 4 paquetes, obtuve %d (%d hechos derivados)", len(ap), n)
	}
	t.Logf("%d obligaciones aplicables a 'acme' derivadas de 5 paquetes, %d hechos totales", len(ap), m.Total())
	for _, h := range ap {
		t.Logf("  %s  <- %s", h.Args[0], m.Explicar(h))
	}
}

// --- Tests anadidos tras la revision adversarial ---

// El bug que la revision destapo: con orden lexicografico, BAJO/MEDIO/ALTO da
// MEDIO como maximo y degrada un sistema ALTA a MEDIA.
func TestSinEscalaDeclaradaSeRechazaEnVezDeAdivinar(t *testing.T) {
	m := NuevoMotor()
	cargar(t, m, Programa{Paquete: "malo", Reglas: []Regla{
		{ID: "max_sin_escala", Cita: "articulo inventado",
			Cabeza:   A("nivel_max", V("S"), V("_AGG")),
			Cuerpo:   []Atomo{A("nivel_dimension", V("S"), V("D"), V("N"))},
			Agregado: Maximo, SobreVar: "N"},
	}})
	m.Afirmar(H("nivel_dimension", "s1", "c", "BAJO"))
	m.Afirmar(H("nivel_dimension", "s1", "i", "ALTO"))
	if _, err := m.Evaluar(); err == nil {
		t.Fatal("agregar sobre valores sin orden declarado tiene que fallar, no adivinar")
	} else {
		t.Logf("rechazado: %v", err)
	}
}

func TestElLinterRechazaReglasInseguras(t *testing.T) {
	casos := []struct {
		nombre string
		p      Programa
	}{
		{"variable de cabeza no ligada", Programa{Paquete: "x", Reglas: []Regla{
			{ID: "r", Cita: "art. 1", Cabeza: A("aplica", C("o"), V("E")),
				Cuerpo: []Atomo{A("base", V("Z"))}}}}},
		{"regla sin cita normativa", Programa{Paquete: "x", Reglas: []Regla{
			{ID: "r", Cabeza: A("aplica", C("o"), V("E")), Cuerpo: []Atomo{A("base", V("E"))}}}}},
		{"negacion insegura", Programa{Paquete: "x", Reglas: []Regla{
			{ID: "r", Cita: "art. 1", Cabeza: A("aplica", C("o"), V("E")),
				Cuerpo: []Atomo{A("base", V("E"))}, Negados: []Atomo{A("otro", V("Q"))}}}}},
		{"agregado sin variable", Programa{Paquete: "x", Reglas: []Regla{
			{ID: "r", Cita: "art. 1", Cabeza: A("n", V("E"), V("_AGG")),
				Cuerpo: []Atomo{A("base", V("E"))}, Agregado: Cuenta}}}},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			if err := c.p.Validar(); err == nil {
				t.Fatal("el linter tiene que rechazar esto antes de cargarlo")
			}
		})
	}
}

// Cuenta contaba combinaciones del cuerpo, no valores distintos. Con eso, los
// umbrales de NIS2 (250 empleados) y CSRD (1.000) salian inflados.
func TestCuentaValoresDistintosNoCombinaciones(t *testing.T) {
	m := NuevoMotor()
	prog := Programa{Paquete: "urn:demo:umbral", Reglas: []Regla{
		{ID: "n_empleados", Cita: "Rec. 2003/361/CE",
			Cabeza:   A("n_empleados", V("E"), V("_AGG")),
			Cuerpo:   []Atomo{A("empleado", V("E"), V("P")), A("contrato", V("E"), Anon())},
			Agregado: Cuenta, SobreVar: "P"},
	}}
	cargar(t, m, prog)
	m.Afirmar(H("empleado", "acme", "ana"))
	m.Afirmar(H("empleado", "acme", "luis"))
	m.Afirmar(H("contrato", "acme", "c1"))
	m.Afirmar(H("contrato", "acme", "c2"))
	m.Afirmar(H("contrato", "acme", "c3"))
	if _, err := m.Evaluar(); err != nil {
		t.Fatal(err)
	}
	// n_empleados es un paso intermedio del paquete y no se exporta, asi que
	// desde fuera se consulta por su nombre local. Es la otra cara del
	// aislamiento: quien quiera ver el interior de una norma tiene que nombrar
	// la norma.
	got := m.Consultar(A(prog.Local("n_empleados"), C("acme"), V("N")))
	if len(got) != 1 || got[0].Args[1] != "2" {
		t.Fatalf("son 2 empleados, no 2x3=6. Obtenido: %v", got)
	}
}

// La variable anonima tiene que casar con cualquier cosa y no ligar nada.
func TestVariableAnonima(t *testing.T) {
	m := NuevoMotor()
	cargar(t, m, Programa{Paquete: "x", Reglas: []Regla{
		{ID: "r", Cita: "art. 1", Cabeza: A("aplica", C("o"), V("E")),
			Cuerpo: []Atomo{A("sector", V("E"), Anon()), A("tam", V("E"), Anon())}},
	}})
	m.Afirmar(H("sector", "acme", "salud"))
	m.Afirmar(H("tam", "acme", "grande"))
	if _, err := m.Evaluar(); err != nil {
		t.Fatal(err)
	}
	if len(m.Consultar(A("aplica", C("o"), C("acme")))) != 1 {
		t.Fatal("dos anonimas distintas no pueden unificarse entre si")
	}
}

// Rendimiento del cierre transitivo tras anadir el indice por predicado.
func TestCierreTransitivoEscala(t *testing.T) {
	m := NuevoMotor()
	// El cierre se hace sobre un predicado propio derivado del hecho, no sobre
	// el hecho. Ver la nota de programaCadenaDeProveedores: un paquete no
	// reescribe lo que declara el sujeto.
	cargar(t, m, Programa{Paquete: "urn:demo:cadena-proveedores", Reglas: []Regla{
		{ID: "directo", Cita: "Directiva 2022/2555 art. 21.2.d",
			Cabeza: A("proveedor_de", V("A"), V("B")),
			Cuerpo: []Atomo{A("provee_a", V("A"), V("B"))}},
		{ID: "transitiva", Cita: "Directiva 2022/2555 art. 21.2.d",
			Cabeza: A("proveedor_de", V("A"), V("C")),
			Cuerpo: []Atomo{A("provee_a", V("A"), V("B")), A("proveedor_de", V("B"), V("C"))}},
	}})
	const n = 120
	nodo := func(i int) string { return fmt.Sprintf("p%03d", i) }
	for i := 0; i < n; i++ {
		m.Afirmar(H("provee_a", nodo(i), nodo(i+1)))
	}
	if _, err := m.Evaluar(); err != nil {
		t.Fatal(err)
	}
	t.Logf("cadena de %d nodos: %d hechos derivados (cierre transitivo completo)", n, m.Total())
	// Nota honesta: el cierre transitivo SIN cota de profundidad es cuadratico
	// en la salida por definicion (n(n-1)/2 pares). Con 250 nodos son 31.375
	// hechos y 8,5 s. En el corpus real hay que acotar la profundidad, porque
	// el art. 21.2.d de NIS2 habla de proveedores DIRECTOS y la cascada
	// contractual tiene dos o tres saltos, no doscientos.
}
