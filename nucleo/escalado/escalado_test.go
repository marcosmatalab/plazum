package escalado

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/marcosmatalab/plazum/nucleo/corpus"
	"github.com/marcosmatalab/plazum/nucleo/ventana"
)

func ins(t *testing.T, s string) time.Time {
	t.Helper()
	x, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatalf("instante %q: %v", s, err)
	}
	return x
}

func reg(comp ventana.Computo) ventana.Regimen {
	return ventana.Regimen{
		Cal:    ventana.NuevoCalendario("utc-v1", "escalado", "prueba", time.UTC),
		Comp:   comp,
		Cierre: ventana.CierreExacto,
	}
}

// obligacionConEscalones no cablea ninguna norma: compone la FORMA que un
// paquete puede declarar.
func obligacionConEscalones(esc ...corpus.Escalon) corpus.Obligacion {
	return corpus.Obligacion{
		ID: "prueba.obligacion", Titulo: "Revisar lo que haya que revisar",
		ClaseE2E: "procedimental", Escalado: esc,
	}
}

func enlaceLocal(o, h string) string { return "http://localhost:8080/obligacion/" + o + "#" + h }

// ---------------------------------------------------------------------------
// (1) El mensaje lleva lo MINIMO, y lo garantiza la FORMA del tipo
// ---------------------------------------------------------------------------

// La garantia estructural, mecanizada. Si alguien anade a `Aviso` un campo
// donde quepa el contenido de un incidente, este test se pone rojo ANTES de que
// exista el codigo que lo rellene.
//
// Es distinto de comprobar el texto de salida: el texto se puede arreglar
// despues de una fuga; un campo que no existe no se puede rellenar nunca.
func TestUnAvisoNoTIENEDondeMeterElContenidoDeUnIncidente(t *testing.T) {
	quiero := []string{"Obligacion", "Titulo", "Hito", "Vence", "Figura", "Nivel", "Enlace"}
	tp := reflect.TypeOf(Aviso{})
	var hay []string
	for i := 0; i < tp.NumField(); i++ {
		hay = append(hay, tp.Field(i).Name)
	}
	if !reflect.DeepEqual(hay, quiero) {
		t.Fatalf("los campos de Aviso son %v y la lista autorizada es %v.\n\n"+
			"  Un aviso sale de la organizacion por un canal de terceros (SMTP ajeno, Teams). "+
			"Cada campo nuevo es una via por la que puede salir dato de cliente, y la promesa "+
			"del producto es que eso no pasa. Si el campo hace falta de verdad, se anade AQUI "+
			"a proposito y se explica en la cabecera del tipo.", hay, quiero)
	}
}

// Y la garantia de comportamiento, con el patron del export a SIEM: un
// centinela plantado en todo lo que NO esta en la lista blanca no sale, y el
// control negativo demuestra que la comprobacion sabe encontrarlo cuando si.
func TestNiElContenidoNiLaClasificacionDeUnIncidenteSalenEnElAviso(t *testing.T) {
	const centinela = "CENTINELA-DATO-DE-CLIENTE-QUE-NO-DEBE-SALIR"

	a := Aviso{
		Obligacion: "prueba.obligacion", Titulo: "Notificar lo que toque", Hito: "inicial",
		Vence: ins(t, "2026-03-05T09:00:00Z"), Figura: "prueba.responsable", Nivel: 2,
		Enlace: enlaceLocal("prueba.obligacion", "inicial"),
	}
	salida := a.Asunto() + "\n" + a.Cuerpo()

	// Lo que un incidente sabe y un aviso NO puede contar.
	for _, prohibido := range []string{
		centinela,
		"incidente_fallecimiento",  // la clasificacion: cuenta algo que nadie autorizo
		"datos de 12.000 clientes", // el alcance de la brecha
		"soc@ejemplo",              // quien lo registro
	} {
		if strings.Contains(salida, prohibido) {
			t.Errorf("el aviso lleva %q:\n%s", prohibido, salida)
		}
	}

	// CONTROL NEGATIVO: el mismo centinela en un campo AUTORIZADO si sale. Sin
	// esto, un Cuerpo() que devolviera la cadena vacia pasaria lo de arriba.
	b := a
	b.Titulo = centinela
	if !strings.Contains(b.Asunto()+b.Cuerpo(), centinela) {
		t.Fatal("el centinela tampoco sale desde un campo autorizado, asi que la comprobacion " +
			"de arriba no demuestra que el aviso filtre: demuestra que no dice nada")
	}
}

// El enlace es a la instancia LOCAL y lo construye quien llama, no este
// paquete: el nucleo no sabe donde vive la instancia del cliente y no se lo
// puede inventar.
func TestElNucleoNoSeInventaNingunDestinoParaElEnlace(t *testing.T) {
	o := obligacionConEscalones(corpus.Escalon{Tras: "P1D", A: "prueba.responsable"})
	pasos, err := Planificar(o, "inicial", ins(t, "2026-03-05T09:00:00Z"), reg(ventana.Naturales),
		Asignacion{"prueba.responsable": "ana"}, nil,
		func(string, string) string { return "/local/expediente" })
	if err != nil {
		t.Fatal(err)
	}
	if pasos[0].Aviso.Enlace != "/local/expediente" {
		t.Fatalf("el enlace sale como %q y tenia que ser el que da quien llama",
			pasos[0].Aviso.Enlace)
	}
}

// ---------------------------------------------------------------------------
// (2) El orden lo decide el instante, no el orden del fichero
// ---------------------------------------------------------------------------

func TestElOrdenLoDecideElInstanteYNoComoEsteEscritoElPaquete(t *testing.T) {
	vence := ins(t, "2026-06-01T12:00:00Z")
	a := Asignacion{"f.uno": "ana", "f.dos": "beatriz", "f.tres": "carlos"}
	escalones := []corpus.Escalon{
		{Tras: "P7D", A: "f.tres"},       // despues del vencimiento: el ultimo
		{Tras: "P60D_antes", A: "f.uno"}, // el primero
		{Tras: "P30D_antes", A: "f.dos"}, // el segundo
	}
	quiero := []string{"f.uno", "f.dos", "f.tres"}

	// El mismo conjunto en dos ordenes distintos tiene que dar el mismo plan.
	for _, orden := range [][]corpus.Escalon{
		escalones,
		{escalones[1], escalones[2], escalones[0]},
	} {
		pasos, err := Planificar(obligacionConEscalones(orden...), "h", vence,
			reg(ventana.Naturales), a, nil, enlaceLocal)
		if err != nil {
			t.Fatal(err)
		}
		var hay []string
		for _, p := range pasos {
			hay = append(hay, p.Figura)
		}
		if !reflect.DeepEqual(hay, quiero) {
			t.Errorf("el plan sale %v y el orden por instante es %v", hay, quiero)
		}
		for i, p := range pasos {
			if p.Nivel != i+1 {
				t.Errorf("el nivel de %s es %d y tenia que ser %d", p.Figura, p.Nivel, i+1)
			}
		}
	}
}

// EL AVISO DE CORTESIA VA HACIA ATRAS, y con dias HABILES es donde muerde: si
// se calculara sumando una duracion negada, `ventana.Sumar` recorre
// `for restantes > 0`, que con un negativo no entra ni una vez y devuelve la
// fecha del vencimiento SIN ERROR. O sea, un aviso de "60 dias habiles antes"
// que llega el mismo dia. Por eso `_antes` usa Restar.
func TestElAvisoDeCortesiaVaHaciaAtrasTambienEnDiasHabiles(t *testing.T) {
	vence := ins(t, "2026-06-01T12:00:00Z")
	for _, comp := range []ventana.Computo{ventana.Naturales, ventana.Habiles} {
		m, err := corpus.ParseTras("P10D_antes")
		if err != nil {
			t.Fatal(err)
		}
		cuando, _ := m.Instante(vence, reg(comp))
		if !cuando.Before(vence) {
			t.Errorf("con computo %v, el aviso de 10 dias antes cae el %s y el vencimiento "+
				"es el %s: no ha ido hacia atras", comp, cuando.Format(time.RFC3339),
				vence.Format(time.RFC3339))
		}
	}
}

func TestUnTrasIlegibleNoSePlanifica(t *testing.T) {
	for _, malo := range []string{"", "P60D_ants", "sesenta dias", "indeterminado", "P"} {
		if _, err := corpus.ParseTras(malo); err == nil {
			t.Errorf("%q se acepta como `tras`", malo)
		}
	}
	// Y el bueno, para que el rechazo de arriba signifique algo.
	for _, bueno := range []string{"P60D_antes", "PT4H", "P1M", "P5D"} {
		if _, err := corpus.ParseTras(bueno); err != nil {
			t.Errorf("%q se rechaza: %v", bueno, err)
		}
	}
}

// ---------------------------------------------------------------------------
// (3) El colapso de niveles
// ---------------------------------------------------------------------------

func TestLaMismaPersonaEnDosEscalonesNoRecibeDosVeces(t *testing.T) {
	pasos, err := Planificar(obligacionConEscalones(
		corpus.Escalon{Tras: "P60D_antes", A: "f.seguridad"},
		corpus.Escalon{Tras: "P30D_antes", A: "f.direccion"},
	), "h", ins(t, "2026-06-01T12:00:00Z"), reg(ventana.Naturales),
		// La misma persona ocupa las dos figuras, que es lo normal en una
		// organizacion pequena.
		Asignacion{"f.seguridad": "ana", "f.direccion": "ana"}, nil, enlaceLocal)
	if err != nil {
		t.Fatal(err)
	}
	if pasos[0].Estado != Pendiente || pasos[0].Aviso == nil {
		t.Fatalf("el primer escalon no sale: %+v", pasos[0])
	}
	if pasos[1].Estado != Colapsado {
		t.Fatalf("el segundo escalon sale como %q y tenia que colapsar", pasos[1].Estado)
	}
	if pasos[1].Aviso != nil {
		t.Error("un escalon colapsado trae aviso, asi que alguien podria mandarlo")
	}
	if !strings.Contains(pasos[1].Motivo, "escalon 1") {
		t.Errorf("el motivo del colapso no dice que escalon lo absorbio: %q", pasos[1].Motivo)
	}
	// Y no colapsa lo que no debe: dos personas distintas reciben las dos.
	pasos, err = Planificar(obligacionConEscalones(
		corpus.Escalon{Tras: "P60D_antes", A: "f.seguridad"},
		corpus.Escalon{Tras: "P30D_antes", A: "f.direccion"},
	), "h", ins(t, "2026-06-01T12:00:00Z"), reg(ventana.Naturales),
		Asignacion{"f.seguridad": "ana", "f.direccion": "beatriz"}, nil, enlaceLocal)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range pasos {
		if p.Estado != Pendiente {
			t.Errorf("con dos personas distintas, el escalon %d sale como %q", p.Nivel, p.Estado)
		}
	}
}

// ---------------------------------------------------------------------------
// (4) La ley de conservacion del escalado
// ---------------------------------------------------------------------------

// Todo escalon planificado cae en EXACTAMENTE un estado, y el que promete envio
// trae aviso y el que no, no. La segunda mitad es la que cierra la junta: una
// etiqueta sin aviso detras dejaria la ley en verde y el escalon sin mandarse.
func TestTodoEscalonCaeEnExactamenteUnEstadoYSoloEnviaElQuePuede(t *testing.T) {
	vence := ins(t, "2026-06-01T12:00:00Z")
	pasos, err := Planificar(obligacionConEscalones(
		corpus.Escalon{Tras: "P90D_antes", A: "f.nadie"},     // sin destinatario
		corpus.Escalon{Tras: "P60D_antes", A: "f.seguridad"}, // sale
		corpus.Escalon{Tras: "P30D_antes", A: "f.direccion"}, // colapsa en ana
		corpus.Escalon{Tras: "P5D", A: "f.auditor"},          // cae en silencio
	), "h", vence, reg(ventana.Naturales),
		Asignacion{"f.seguridad": "ana", "f.direccion": "ana", "f.auditor": "beatriz"},
		[]Silencio{{
			Desde: ins(t, "2026-06-05T00:00:00Z"), Hasta: ins(t, "2026-06-10T00:00:00Z"),
			Motivo: "parada de mantenimiento declarada", Quien: "ana",
		}}, enlaceLocal)
	if err != nil {
		t.Fatal(err)
	}

	suma := 0
	for _, n := range Cuenta(pasos) {
		suma += n
	}
	if suma != len(pasos) {
		t.Fatalf("los estados suman %d y hay %d escalones", suma, len(pasos))
	}
	// La junta: prometer fila y tenerla.
	for _, p := range pasos {
		envia := p.Estado == Pendiente
		if envia != (p.Aviso != nil) {
			t.Errorf("el escalon %d esta en %q y su aviso es %v: la etiqueta y el aviso no "+
				"dicen lo mismo", p.Nivel, p.Estado, p.Aviso != nil)
		}
		// Y ningun cubo que cierra el escalon se queda sin explicacion.
		if p.Estado.EsFinal() && strings.TrimSpace(p.Motivo) == "" {
			t.Errorf("el escalon %d acaba en %q sin motivo: un cubo sin explicacion es una "+
				"etiqueta", p.Nivel, p.Estado)
		}
	}
	quiero := map[Estado]int{SinDestinatario: 1, Pendiente: 1, Colapsado: 1, EnSilencio: 1}
	if got := Cuenta(pasos); !reflect.DeepEqual(got, quiero) {
		t.Fatalf("el reparto es %v y tenia que ser %v", got, quiero)
	}
}

// Una ventana de silencio SUPRIME y no BORRA: el escalon sigue contado, con su
// motivo. Callar un aviso sin dejar rastro es el fallo que esta clase persigue.
func TestUnSilencioSuprimeYDejaRastro(t *testing.T) {
	vence := ins(t, "2026-06-01T12:00:00Z")
	pasos, err := Planificar(obligacionConEscalones(
		corpus.Escalon{Tras: "P1D", A: "f.seguridad"},
	), "h", vence, reg(ventana.Naturales), Asignacion{"f.seguridad": "ana"},
		[]Silencio{{Desde: ins(t, "2026-06-01T00:00:00Z"),
			Hasta: ins(t, "2026-06-10T00:00:00Z"), Motivo: "corte planificado",
			Quien: "ana"}}, enlaceLocal)
	if err != nil {
		t.Fatal(err)
	}
	if len(pasos) != 1 {
		t.Fatalf("el escalon suprimido ha desaparecido de la lista: %+v", pasos)
	}
	if pasos[0].Estado != EnSilencio || !strings.Contains(pasos[0].Motivo, "corte planificado") {
		t.Fatalf("el silencio no deja rastro: %+v", pasos[0])
	}
	if pasos[0].Aviso != nil {
		t.Error("un escalon en silencio trae aviso, asi que alguien podria mandarlo igual")
	}
}

// ---------------------------------------------------------------------------
// (5) La comprobacion del dia uno
// ---------------------------------------------------------------------------

func TestLasFigurasSinPersonaSeSabenElDiaUnoYNoElDelIncidente(t *testing.T) {
	paquetes, err := corpus.Cargar("../../paquetes")
	if err != nil {
		t.Fatalf("cargando el corpus: %v", err)
	}

	// Sin asignar nada: TODAS las figuras del corpus faltan.
	faltan := FigurasSinPersona(paquetes, nil)
	total := 0
	for _, p := range paquetes {
		total += len(p.Roles)
	}
	if total == 0 {
		t.Fatal("el corpus no declara figuras: el test no comprueba nada")
	}
	if len(faltan) != total {
		t.Fatalf("con el alcance vacio faltan %d figuras de %d", len(faltan), total)
	}
	// La mas cara arriba: la que deja mas avisos sin destinatario.
	for i := 1; i < len(faltan); i++ {
		if faltan[i-1].Escalones < faltan[i].Escalones {
			t.Fatalf("la lista no esta ordenada por coste: %d antes que %d",
				faltan[i-1].Escalones, faltan[i].Escalones)
		}
	}
	// Y la frase distingue de quien es la figura, porque no se piden igual.
	vistoDeLaNorma, vistoSugerida := false, false
	for _, f := range faltan {
		frase := f.Frase()
		if !strings.Contains(frase, f.Figura) || !strings.Contains(frase, "sin destinatario") {
			t.Errorf("la frase de %s no dice lo que hace falta: %q", f.Figura, frase)
		}
		switch f.Origen {
		case corpus.FiguraDeLaNorma:
			vistoDeLaNorma = true
			if !strings.Contains(frase, "la nombra la norma") {
				t.Errorf("%s la nombra la norma y su frase no lo dice: %q", f.Figura, frase)
			}
		case corpus.FiguraPropuesto:
			vistoSugerida = true
			if !strings.Contains(frase, "puedes cambiarla") {
				t.Errorf("%s la propone plazum y su frase no lo dice: %q", f.Figura, frase)
			}
		}
	}
	if !vistoDeLaNorma || !vistoSugerida {
		t.Fatal("no se han recorrido las dos clases de figura, asi que la mitad de la " +
			"comprobacion de la frase no se ha ejecutado")
	}

	// CONTROL POSITIVO: con todas ocupadas, no falta ninguna. Sin esto, una
	// funcion que devolviera siempre todo daria el mismo verde arriba.
	todas := Asignacion{}
	for _, p := range paquetes {
		for _, r := range p.Roles {
			todas[r.ID] = "una persona"
		}
	}
	if n := len(FigurasSinPersona(paquetes, todas)); n != 0 {
		t.Fatalf("con todas las figuras ocupadas siguen faltando %d", n)
	}
	// Y el blanco no cuenta como ocupada: "   " es la forma de la nada que sale
	// de un CSV mal recortado.
	todas["prueba"] = "  "
	if n := len(FigurasSinPersona(paquetes, Asignacion{})); n != total {
		t.Fatalf("el mapa vacio no da todas: %d", n)
	}
}

func TestUnaPersonaEnBlancoNoOcupaLaFigura(t *testing.T) {
	pasos, err := Planificar(obligacionConEscalones(
		corpus.Escalon{Tras: "P1D", A: "f.seguridad"},
	), "h", ins(t, "2026-06-01T12:00:00Z"), reg(ventana.Naturales),
		Asignacion{"f.seguridad": "   "}, nil, enlaceLocal)
	if err != nil {
		t.Fatal(err)
	}
	if pasos[0].Estado != SinDestinatario {
		t.Fatalf("una persona en blanco ocupa la figura: %+v", pasos[0])
	}
	if !strings.Contains(pasos[0].Motivo, "NO dice que el aviso haya fallado") {
		t.Errorf("el motivo acusa en vez de decir que falta el dato: %q", pasos[0].Motivo)
	}
}

// ---------------------------------------------------------------------------
// (6) El curso: se notifico y se atendio son dos hechos
// ---------------------------------------------------------------------------

func TestEntregadoYAtendidoNoSonElMismoHecho(t *testing.T) {
	var c Curso
	envio := ins(t, "2026-06-01T12:00:00Z")
	if err := c.Registrar(SucesoDeEnvio{Tipo: SeEnvio, Canal: "email",
		InstanteHecho: envio, InstanteRegistro: envio}); err != nil {
		t.Fatal(err)
	}
	if e, _ := c.Estado(); e != Enviado {
		t.Fatalf("tras el envio el estado es %q", e)
	}
	entrega := envio.Add(time.Minute)
	if err := c.Registrar(SucesoDeEnvio{Tipo: SeEntrego, Canal: "email",
		InstanteHecho: entrega, InstanteRegistro: entrega}); err != nil {
		t.Fatal(err)
	}
	if e, _ := c.Estado(); e != Entregado {
		t.Fatalf("tras la entrega el estado es %q", e)
	}
	// Entregado NO es atendido: el correo llego a una bandeja que nadie abre.
	if _, _, consta := c.Atendido(); consta {
		t.Fatal("un aviso entregado consta como atendido, y el acta del 9.3 se creeria que " +
			"alguien se hizo cargo")
	}
	att := entrega.Add(2 * time.Hour)
	if err := c.Registrar(SucesoDeEnvio{Tipo: SeAtendio, Quien: "ana",
		InstanteHecho: att, InstanteRegistro: att}); err != nil {
		t.Fatal(err)
	}
	if e, _ := c.Estado(); e != Atendido {
		t.Fatalf("tras atenderlo el estado es %q", e)
	}
	quien, cuando, consta := c.Atendido()
	if !consta || quien != "ana" || !cuando.Equal(att) {
		t.Fatalf("el acuse no guarda quien y cuando: %q %v %v", quien, cuando, consta)
	}
}

// Un fallo manda sobre un envio: un escalon que se intento, fallo y no se
// reintento esta FALLIDO, no enviado. Si se leyera como enviado, el operador
// creeria que esta cubierto.
func TestUnFalloDeEntregaNoSeQuedaComoEnviado(t *testing.T) {
	var c Curso
	x := ins(t, "2026-06-01T12:00:00Z")
	for _, s := range []SucesoDeEnvio{
		{Tipo: SeEnvio, Canal: "email", InstanteHecho: x, InstanteRegistro: x},
		{Tipo: Fallo, Canal: "email", Motivo: "buzon lleno", InstanteHecho: x, InstanteRegistro: x},
	} {
		if err := c.Registrar(s); err != nil {
			t.Fatal(err)
		}
	}
	if e, _ := c.Estado(); e != Fallido {
		t.Fatalf("tras un fallo el estado es %q", e)
	}
	// Y atender despues manda sobre el fallo: lo que importa al expediente es
	// si alguien se hizo cargo, no cuantas veces lo intento el canal.
	att := x.Add(time.Hour)
	if err := c.Registrar(SucesoDeEnvio{Tipo: SeAtendio, Quien: "ana",
		InstanteHecho: att, InstanteRegistro: att}); err != nil {
		t.Fatal(err)
	}
	if e, _ := c.Estado(); e != Atendido {
		t.Fatalf("tras atenderlo el estado sigue siendo %q", e)
	}
}

func TestUnCursoRechazaLosHechosImposibles(t *testing.T) {
	x := ins(t, "2026-06-01T12:00:00Z")
	casos := []struct {
		nombre string
		s      SucesoDeEnvio
		quiero error
	}{
		{"entrega sin envio previo",
			SucesoDeEnvio{Tipo: SeEntrego, Canal: "email", InstanteHecho: x, InstanteRegistro: x},
			ErrCursoSinEnvioAntes},
		{"tipo fuera del vocabulario",
			SucesoDeEnvio{Tipo: TipoSuceso(9), Canal: "email", InstanteHecho: x, InstanteRegistro: x},
			ErrSucesoDesconocido},
		{"sin uno de los dos ejes",
			SucesoDeEnvio{Tipo: SeEnvio, Canal: "email", InstanteHecho: x},
			ErrSucesoSinInstante},
		{"registrado antes de ocurrir",
			SucesoDeEnvio{Tipo: SeEnvio, Canal: "email", InstanteHecho: x.Add(time.Hour),
				InstanteRegistro: x}, ErrSucesoAlReves},
		{"atendido sin quien",
			SucesoDeEnvio{Tipo: SeAtendio, InstanteHecho: x, InstanteRegistro: x},
			ErrSucesoCampoFalta},
		{"fallo sin motivo",
			SucesoDeEnvio{Tipo: Fallo, Canal: "email", InstanteHecho: x, InstanteRegistro: x},
			ErrSucesoCampoFalta},
		{"envio con quien, que no le toca",
			SucesoDeEnvio{Tipo: SeEnvio, Canal: "email", Quien: "ana", InstanteHecho: x,
				InstanteRegistro: x}, ErrSucesoCampoAjeno},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			var cu Curso
			err := cu.Registrar(c.s)
			if !errors.Is(err, c.quiero) {
				t.Fatalf("devuelve %v y tenia que ser %v", err, c.quiero)
			}
		})
	}
}

func TestSucesosNoDaAccesoAlFlujoInterno(t *testing.T) {
	var c Curso
	x := ins(t, "2026-06-01T12:00:00Z")
	if err := c.Registrar(SucesoDeEnvio{Tipo: SeEnvio, Canal: "email",
		InstanteHecho: x, InstanteRegistro: x}); err != nil {
		t.Fatal(err)
	}
	s := c.Sucesos()
	s[0].Tipo = SeAtendio
	if e, _ := c.Estado(); e != Enviado {
		t.Fatalf("tocando el resultado de Sucesos el estado paso a %q", e)
	}
}

// ---------------------------------------------------------------------------
// (7) Plegar el curso sobre el plan
// ---------------------------------------------------------------------------

func TestUnAvisoQueNadiePlanificoSeDiceEnVozAlta(t *testing.T) {
	pasos, err := Planificar(obligacionConEscalones(
		corpus.Escalon{Tras: "P1D", A: "f.nadie"},
	), "h", ins(t, "2026-06-01T12:00:00Z"), reg(ventana.Naturales), nil, nil, enlaceLocal)
	if err != nil {
		t.Fatal(err)
	}
	if pasos[0].Estado != SinDestinatario {
		t.Fatalf("el paso no esta sin destinatario: %+v", pasos[0])
	}
	var c Curso
	x := ins(t, "2026-06-01T12:00:00Z")
	if err := c.Registrar(SucesoDeEnvio{Tipo: SeEnvio, Canal: "email",
		InstanteHecho: x, InstanteRegistro: x}); err != nil {
		t.Fatal(err)
	}
	if _, err := AplicarCursos(pasos, map[int]*Curso{1: &c}); err == nil {
		t.Fatal("un envio sobre un escalon que el plan habia descartado pasa en silencio")
	}
}

func TestElEstadoFinalDeUnPasoSaleDeSuCurso(t *testing.T) {
	pasos, err := Planificar(obligacionConEscalones(
		corpus.Escalon{Tras: "P1D", A: "f.seguridad"},
	), "h", ins(t, "2026-06-01T12:00:00Z"), reg(ventana.Naturales),
		Asignacion{"f.seguridad": "ana"}, nil, enlaceLocal)
	if err != nil {
		t.Fatal(err)
	}
	var c Curso
	x := ins(t, "2026-06-02T12:00:00Z")
	for _, s := range []SucesoDeEnvio{
		{Tipo: SeEnvio, Canal: "email", InstanteHecho: x, InstanteRegistro: x},
		{Tipo: Fallo, Canal: "email", Motivo: "dominio inexistente", InstanteHecho: x,
			InstanteRegistro: x},
	} {
		if err := c.Registrar(s); err != nil {
			t.Fatal(err)
		}
	}
	fin, err := AplicarCursos(pasos, map[int]*Curso{1: &c})
	if err != nil {
		t.Fatal(err)
	}
	if fin[0].Estado != Fallido {
		t.Fatalf("el paso queda en %q y su curso fallo", fin[0].Estado)
	}
	// El fallo se ENSENA con su motivo: un fallo que solo se loguea es un
	// escalado que muere en silencio.
	if !strings.Contains(fin[0].Motivo, "dominio inexistente") {
		t.Errorf("el motivo del fallo no llega al paso: %q", fin[0].Motivo)
	}
	// Y el plan original no se toca: AplicarCursos devuelve una copia.
	if pasos[0].Estado != Pendiente {
		t.Error("AplicarCursos ha modificado el plan que recibio")
	}
}

// El vocabulario de estados es cerrado y la particion esta completa: si alguien
// anade un estado y se olvida de EstadosPosibles, la ley de conservacion podria
// dejar de contarlo.
func TestLaParticionDeEstadosEstaCompleta(t *testing.T) {
	vistos := map[Estado]bool{}
	for _, e := range EstadosPosibles() {
		if vistos[e] {
			t.Errorf("el estado %q sale dos veces en la particion", e)
		}
		vistos[e] = true
	}
	// Los que produce el planificador y los que produce un curso, todos dentro.
	for _, e := range []Estado{Pendiente, SinDestinatario, Colapsado, EnSilencio,
		Enviado, Entregado, Fallido, Atendido} {
		if !vistos[e] {
			t.Errorf("el estado %q no esta en EstadosPosibles", e)
		}
	}
}

// ---------------------------------------------------------------------------
// (8) Las ventanas de silencio: opt-in, con fin, y auditables
// ---------------------------------------------------------------------------

// UN SILENCIO SIN FIN ES EL ROJO PERMANENTE CON OTRO NOMBRE: se pone "hasta que
// arreglemos esto", nadie lo quita, y seis meses despues el operador cree que no
// vence nada. Y uno sin nombre hace que un aviso callado sea indistinguible de
// un aviso perdido.
func TestUnSilencioQueNoSePuedeAuditarNiCaducarNoSePlanifica(t *testing.T) {
	desde := ins(t, "2026-06-01T00:00:00Z")
	casos := map[string]struct {
		s      Silencio
		quiero error
	}{
		"sin fin":       {Silencio{Desde: desde, Motivo: "m", Quien: "ana"}, ErrSilencioSinFin},
		"sin principio": {Silencio{Hasta: desde, Motivo: "m", Quien: "ana"}, ErrSilencioSinFin},
		"al reves": {Silencio{Desde: desde, Hasta: desde.Add(-time.Hour), Motivo: "m",
			Quien: "ana"}, ErrSilencioAlReves},
		"del mismo instante": {Silencio{Desde: desde, Hasta: desde, Motivo: "m", Quien: "ana"},
			ErrSilencioAlReves},
		"sin quien": {Silencio{Desde: desde, Hasta: desde.Add(time.Hour), Motivo: "m"},
			ErrSilencioSinMotivo},
		"sin motivo": {Silencio{Desde: desde, Hasta: desde.Add(time.Hour), Quien: "ana"},
			ErrSilencioSinMotivo},
		"con quien en blanco": {Silencio{Desde: desde, Hasta: desde.Add(time.Hour), Motivo: "m",
			Quien: "   "}, ErrSilencioSinMotivo},
	}
	for nombre, c := range casos {
		t.Run(nombre, func(t *testing.T) {
			if err := c.s.Validar(); !errors.Is(err, c.quiero) {
				t.Fatalf("Validar devuelve %v y tenia que ser %v", err, c.quiero)
			}
			// Y no se ignora: el plan entero se rechaza. Ignorarlo dejaria la
			// configuracion del operador sin hacer lo que dice.
			_, err := Planificar(obligacionConEscalones(
				corpus.Escalon{Tras: "P1D", A: "f.seguridad"}), "h", desde,
				reg(ventana.Naturales), Asignacion{"f.seguridad": "ana"},
				[]Silencio{c.s}, enlaceLocal)
			if !errors.Is(err, c.quiero) {
				t.Fatalf("planificar con esa ventana devuelve %v", err)
			}
		})
	}
	// CONTROL POSITIVO: la ventana bien puesta pasa. Sin esto, un Validar que
	// rechazara todo daria el mismo verde arriba.
	buena := Silencio{Desde: desde, Hasta: desde.Add(24 * time.Hour),
		Motivo: "parada de mantenimiento", Quien: "ana"}
	if err := buena.Validar(); err != nil {
		t.Fatalf("la ventana bien puesta se rechaza: %v", err)
	}
}

// LA CADUCIDAD DESPIERTA SOLA: pasado el fin, el aviso vuelve a salir sin que
// nadie tenga que levantar la ventana. Es lo que distingue un silencio de un
// apagado, y es la razon de que el fin sea obligatorio.
func TestUnSilencioCaducadoDejaSalirElAvisoSinQueNadieLoLevante(t *testing.T) {
	vence := ins(t, "2026-06-20T12:00:00Z")
	// La ventana cubrio del 1 al 10; el escalon cae el 21, o sea despues.
	s := Silencio{Desde: ins(t, "2026-06-01T00:00:00Z"), Hasta: ins(t, "2026-06-10T00:00:00Z"),
		Motivo: "parada de mantenimiento", Quien: "ana"}
	pasos, err := Planificar(obligacionConEscalones(
		corpus.Escalon{Tras: "P1D", A: "f.seguridad"}), "h", vence, reg(ventana.Naturales),
		Asignacion{"f.seguridad": "ana"}, []Silencio{s}, enlaceLocal)
	if err != nil {
		t.Fatal(err)
	}
	if pasos[0].Estado != Pendiente || pasos[0].Aviso == nil {
		t.Fatalf("con la ventana ya caducada el aviso sigue callado: %+v", pasos[0])
	}
}
