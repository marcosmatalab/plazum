package acta

import (
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/marcosmatalab/plazum/nucleo/accesos"
	"github.com/marcosmatalab/plazum/nucleo/auditoria"
	"github.com/marcosmatalab/plazum/nucleo/censo"
	"github.com/marcosmatalab/plazum/nucleo/incidente"
)

// El caso completo, y esta ES la mitad del trabajo de este fichero.
//
// M47 dice que toda rama de acusacion o de descargo exige un control POSITIVO
// que la recorra: un descargo que ninguna entrada alcanza es un descargo que no
// existe, y la mutacion lo deja verde porque no hay nada que romper. Asi que el
// caso de abajo esta construido para que NINGUN CUBO DEL ACTA quede vacio por
// falta de datos, incluidos los tres de "no consta" y los cuatro de "consta y no
// es culpa". Hay un test que lo exige y que se pone rojo si alguien anade un cubo
// nuevo sin darle datos.

var (
	desde     = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	hasta     = time.Date(2026, 12, 31, 23, 59, 59, 0, time.UTC)
	elPeriodo = Periodo{Desde: desde, Hasta: hasta}
)

func dia(a, m, d int) time.Time { return time.Date(a, time.Month(m), d, 12, 0, 0, 0, time.UTC) }

// programa arma el programa de auditoria del caso completo.
func programa(t *testing.T) *auditoria.Programa {
	t.Helper()
	alcance := []auditoria.Unidad{
		{Paquete: "p1", Version: "0.1", Obligacion: "o1", Titulo: "Copias de seguridad"},
		{Paquete: "p1", Version: "0.1", Obligacion: "o2", Titulo: "Gestion de accesos"},
		{Paquete: "p1", Version: "0.1", Obligacion: "o3", Titulo: "Continuidad"},
		{Paquete: "p1", Version: "0.1", Obligacion: "o4", Titulo: "Formacion"},
		{Paquete: "p2", Version: "0.2", Obligacion: "o1", Titulo: "Registro de actividad"},
	}
	arr := auditoria.Arrastre{
		DeCiclo:    "2023-2025",
		SinAuditar: map[string]int{"p1|o2": 2, "p9|viejo": 1},
		Abiertos:   map[string]int{"H-VIEJO": 1},
	}
	p, err := auditoria.Abrir("prog-1", auditoria.Ciclo{
		Nombre: "2026-2028",
		Desde:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Hasta:  time.Date(2028, 12, 31, 0, 0, 0, 0, time.UTC),
	}, alcance, arr)
	if err != nil {
		t.Fatalf("no abre el programa: %v", err)
	}
	if err := p.Auditar(auditoria.Sesion{
		ID: "S1", Auditor: "aud-1", Cuando: dia(2026, 3, 1),
		Unidades: []string{"p1|o1", "p2|o1", "p1|o4"},
		Alcance:  "revision documental y muestreo",
	}); err != nil {
		t.Fatalf("no audita: %v", err)
	}
	if err := p.Diferir(auditoria.Diferimiento{
		Unidad: "p1|o3", Quien: "jefa", Cuando: dia(2026, 4, 1),
		Motivo: "el sistema se migra en el ultimo trimestre y auditar el viejo no dice nada",
	}); err != nil {
		t.Fatalf("no difiere: %v", err)
	}
	if err := p.Anotar(auditoria.Hallazgo{
		ID: "H1", Sesion: "S1", Unidad: "p1|o1", Clase: auditoria.NoConformidadMenor,
		Texto: "la copia mensual de marzo no tiene registro de restauracion",
		Quien: "aud-1", Cuando: dia(2026, 3, 2),
	}); err != nil {
		t.Fatalf("no anota H1: %v", err)
	}
	if err := p.Anotar(auditoria.Hallazgo{
		ID: "H2", Sesion: "S1", Unidad: "p2|o1", Clase: auditoria.Observacion,
		Texto: "el registro rota cada 30 dias y la politica dice 90",
		Quien: "aud-1", Cuando: dia(2026, 3, 2),
	}); err != nil {
		t.Fatalf("no anota H2: %v", err)
	}
	if err := p.Cerrar(auditoria.CierreDeHallazgo{
		Hallazgo: "H2", Quien: "sistemas", Cuando: dia(2026, 5, 10),
		Como: "se subio la retencion a 90 dias y se adjunta la captura de la configuracion",
	}); err != nil {
		t.Fatalf("no cierra H2: %v", err)
	}
	return p
}

// responsables trae las tres direcciones a proposito: una unidad de la que
// responde el propio auditor (conflicto), una de la que responde otra persona, y
// una asignacion a algo que este programa no tiene en el alcance.
func responsables() map[string]string {
	return map[string]string{
		"p1|o1": "aud-1",
		"p2|o1": "operaciones",
		"p7|x":  "alguien",
	}
}

const censoDelCaso = "usuario;nombre;permiso\n" +
	"u1;Ana Perez;admin\n" + // linea 2
	"u1;Ana Perez;lector\n" + // linea 3
	"u2;Luis Gil;lector\n" + // linea 4
	"u3;Eva Diaz;admin\n" + // linea 5, se queda sin revisar
	"u1;Ana Perez;admin\n" + // linea 6, duplicada
	";Sin identificador;lector\n" + // linea 7, ilegible, se excusa
	";Otro sin identificador;admin\n" // linea 8, ilegible y sin excusar

func campana(t *testing.T) *accesos.Campana {
	t.Helper()
	ins, err := censo.Tomar([]byte(censoDelCaso), censo.Opciones{
		Sistema:   "erp",
		Fuente:    "export de usuarios de Entra ID",
		Quien:     "u-042",
		Tomada:    dia(2026, 6, 1),
		Retencion: "12 meses tras el cierre de la campana",
		Columnas:  censo.ColumnasHabituales(),
	})
	if err != nil {
		t.Fatalf("la instantanea no carga: %v", err)
	}
	if len(ins.Filas) != 4 || len(ins.Duplicadas) != 1 || len(ins.Ilegibles) != 2 {
		t.Fatalf("el caso no trae lo que dice traer: %d filas, %d duplicadas, %d ilegibles",
			len(ins.Filas), len(ins.Duplicadas), len(ins.Ilegibles))
	}
	c, err := accesos.Abrir("uar-2026", ins, dia(2026, 6, 2), map[string]string{
		"erp|u1|admin":  "jefa",
		"erp|u1|lector": "jefa",
		"erp|u2|lector": "jefa",
		"erp|u3|admin":  "jefa",
	})
	if err != nil {
		t.Fatalf("no abre la campana: %v", err)
	}
	decidir := func(fila string, v accesos.Veredicto, a string) {
		d := accesos.Decision{Fila: fila, Veredicto: v, Quien: "jefa", Cuando: dia(2026, 6, 10),
			Motivo: "revision del semestre", A: a}
		if err := c.Registrar(d); err != nil {
			t.Fatalf("no registra %s: %v", fila, err)
		}
	}
	decidir("erp|u1|admin", accesos.Aprobar, "")
	decidir("erp|u1|lector", accesos.Revocar, "")
	decidir("erp|u2|lector", accesos.Delegar, "operaciones")
	// erp|u3|admin se queda sin tocar: es el control positivo del descargo.
	if err := c.Excusar(accesos.Excusa{
		Desde: 7, Hasta: 7, Quien: "u-042", Cuando: dia(2026, 6, 3),
		Motivo: "el export corto una fila a mitad y el sistema de origen ya no la tiene",
	}); err != nil {
		t.Fatalf("no excusa: %v", err)
	}
	return c
}

func abrir(t *testing.T, id string, ocurrio, seSupo time.Time) *incidente.Incidente {
	t.Helper()
	in, err := incidente.Abrir(id, ocurrio, seSupo, "guardia")
	if err != nil {
		t.Fatalf("no abre el incidente %s: %v", id, err)
	}
	return in
}

func registrar(t *testing.T, in *incidente.Incidente, s incidente.Suceso) {
	t.Helper()
	if err := in.Registrar(s); err != nil {
		t.Fatalf("no registra en %s: %v", in.ID(), err)
	}
}

func incidentes(t *testing.T) ([]*incidente.Incidente, []Notificacion) {
	t.Helper()
	// INC-1: dentro, clasificado, notificado dentro del periodo.
	i1 := abrir(t, "INC-1", dia(2026, 2, 1), dia(2026, 2, 2))
	registrar(t, i1, incidente.Suceso{Tipo: incidente.Clasificacion, Clase: "incidente.nivel.alto",
		InstanteHecho: dia(2026, 2, 3), InstanteRegistro: dia(2026, 2, 3), Fuente: "ciso"})
	registrar(t, i1, incidente.Suceso{Tipo: incidente.Notificacion, Hito: "n-inicial",
		InstanteHecho: dia(2026, 2, 4), InstanteRegistro: dia(2026, 2, 4), Fuente: "ciso"})

	// INC-2: dentro y SIN clasificacion. Control positivo del descargo.
	i2 := abrir(t, "INC-2", dia(2026, 5, 1), dia(2026, 5, 1))

	// INC-3: dentro, con dos clasificaciones distintas en el mismo instante.
	i3 := abrir(t, "INC-3", dia(2026, 7, 1), dia(2026, 7, 1))
	registrar(t, i3, incidente.Suceso{Tipo: incidente.Clasificacion, Clase: "incidente.nivel.alto",
		InstanteHecho: dia(2026, 7, 2), InstanteRegistro: dia(2026, 7, 2), Fuente: "ciso"})
	registrar(t, i3, incidente.Suceso{Tipo: incidente.Clasificacion, Clase: "incidente.nivel.bajo",
		InstanteHecho: dia(2026, 7, 2), InstanteRegistro: dia(2026, 7, 3), Fuente: "legal"})

	// INC-4: se supo ANTES del periodo. No lo cuenta este acta.
	i4 := abrir(t, "INC-4", dia(2025, 5, 1), dia(2025, 6, 1))

	// INC-5: dentro, y su notificacion consta remitida FUERA del periodo.
	i5 := abrir(t, "INC-5", dia(2026, 12, 20), dia(2026, 12, 21))
	registrar(t, i5, incidente.Suceso{Tipo: incidente.Clasificacion, Clase: "incidente.nivel.alto",
		InstanteHecho: dia(2026, 12, 22), InstanteRegistro: dia(2026, 12, 22), Fuente: "ciso"})
	registrar(t, i5, incidente.Suceso{Tipo: incidente.Notificacion, Hito: "n-inicial",
		InstanteHecho: dia(2027, 1, 5), InstanteRegistro: dia(2027, 1, 5), Fuente: "ciso"})

	esperadas := []Notificacion{
		{Incidente: "INC-1", Hito: "n-inicial", Que: "notificacion inicial a la autoridad"},
		{Incidente: "INC-2", Hito: "n-inicial", Que: "notificacion inicial a la autoridad"},
		{Incidente: "INC-5", Hito: "n-inicial", Que: "notificacion inicial a la autoridad"},
		{Incidente: "INC-4", Hito: "n-inicial", Que: "notificacion inicial a la autoridad"},
	}
	return []*incidente.Incidente{i1, i2, i3, i4, i5}, esperadas
}

func entradasCompletas(t *testing.T) Entradas {
	t.Helper()
	ins, esp := incidentes(t)
	return Entradas{
		ID:           "acta-2026",
		Organizacion: "Molduras del Norte SL",
		Periodo:      elPeriodo,
		Cubre: []Obligacion{{
			Paquete: "p1", Version: "0.1", ID: "p1.ritual.revision",
			Titulo: "Revision del sistema por la direccion",
			Cita:   "p1, apartado de revision (identificador de requisito)",
		}},
		Programa:                programa(t),
		Responsables:            responsables(),
		Campana:                 campana(t),
		HayRegistroDeIncidentes: true,
		Incidentes:              ins,
		Esperadas:               esp,
		QuienAsistio:            []string{"Ana Perez (consejera delegada)", "Luis Gil (CISO)"},
		Decisiones: []Parrafo{{
			Frase: Frase{Texto: "El consejo da por adecuado el sistema y aprueba dos horas de " +
				"dedicacion semanal para cerrar el hallazgo H1 antes de julio."},
			De: DeUnaPersona, Quien: "Ana Perez",
		}},
	}
}

func componer(t *testing.T, e Entradas) Acta {
	t.Helper()
	a, err := Componer(e)
	if err != nil {
		t.Fatalf("no compone: %v", err)
	}
	return a
}

func cifra(t *testing.T, a Acta, f Fuente, reparto, cubo string) Cifra {
	t.Helper()
	for _, c := range a.Cifras() {
		if c.Fuente == f && c.Reparto.Texto == reparto && c.Cifra.Cubo.Texto == cubo {
			return c.Cifra
		}
	}
	t.Fatalf("no existe el cubo %q del reparto %q en %q", cubo, reparto, f)
	return Cifra{}
}

// ---------------------------------------------------------------------------
// D11-c: ningun numero se imprime sin poder abrirse
// ---------------------------------------------------------------------------

// LA PUERTA DE LA REGLA 1, y es la que hace de este documento algo distinto de
// un informe: cada cifra tiene su referencia, la referencia resuelve, y lo que
// resuelve tiene exactamente tantos elementos como dice el numero.
//
// Lo ultimo es cierto POR CONSTRUCCION (Valor() es len(Elementos)), y por eso el
// test mira lo que la construccion no garantiza: que la referencia impresa lleve
// de vuelta a ESA cifra y no a otra, y que dentro de un cubo no haya dos filas
// con la misma identidad, que es como un numero se infla sin que se note.
func TestTodoNumeroDelActaSeAbreYLoQueSaleEsEseNumero(t *testing.T) {
	a := componer(t, entradasCompletas(t))
	texto := a.Texto()
	if len(a.Cifras()) == 0 {
		t.Fatal("el acta no tiene ninguna cifra")
	}
	for _, c := range a.Cifras() {
		vuelta, ok := a.Derivar(c.Ref)
		if !ok {
			t.Fatalf("la referencia %s no resuelve", c.Ref)
		}
		if vuelta.Cifra.Cubo != c.Cifra.Cubo || vuelta.Reparto != c.Reparto {
			t.Errorf("la referencia %s lleva a %q/%q y tenia que llevar a %q/%q",
				c.Ref, vuelta.Reparto.Texto, vuelta.Cifra.Cubo.Texto, c.Reparto.Texto,
				c.Cifra.Cubo.Texto)
		}
		if vuelta.Cifra.Valor() != c.Cifra.Valor() {
			t.Errorf("la referencia %s abre %d elementos y el numero decia %d",
				c.Ref, vuelta.Cifra.Valor(), c.Cifra.Valor())
		}

		visto := map[string]bool{}
		for _, el := range c.Cifra.Elementos {
			if strings.TrimSpace(el.Clave) == "" {
				t.Errorf("%s: un elemento sin clave no se puede seguir hasta su fuente", c.Ref)
			}
			if visto[el.Clave] {
				t.Errorf("%s: la clave %q sale dos veces en el mismo cubo, asi que el numero "+
					"cuenta dos veces lo mismo", c.Ref, el.Clave)
			}
			visto[el.Clave] = true
		}
	}

	// Y LA REFERENCIA QUE VA PEGADA AL NUMERO ES LA SUYA, leida DEL DOCUMENTO.
	//
	// ESTE BLOQUE EXISTE POR UNA MUTACION QUE SOBREVIVIO. La primera version
	// comprobaba `strings.Contains(texto, "["+ref+"]")`, y con las referencias del
	// cuerpo cambiadas por otras el test seguia verde: las encontraba en el
	// APENDICE, que las imprime bien. O sea, comprobaba que el modelo es
	// coherente consigo mismo y no que el documento diga la verdad, que es lo
	// unico que le importa a quien lo lee. Ahora se leen las lineas impresas y se
	// enfrentan al modelo, cubo, valor y referencia.
	impresas := numerosImpresos(t, texto)
	if len(impresas) != len(a.Cifras()) {
		t.Fatalf("el documento imprime %d numeros y el acta tiene %d cifras",
			len(impresas), len(a.Cifras()))
	}
	for i, c := range a.Cifras() {
		im := impresas[i]
		if im.ref != c.Ref || im.cubo != c.Cifra.Cubo.Texto || im.valor != c.Cifra.Valor() {
			t.Errorf("la linea impresa %d dice (%q, %d, [%s]) y la cifra es (%q, %d, [%s])",
				i, im.cubo, im.valor, im.ref, c.Cifra.Cubo.Texto, c.Cifra.Valor(), c.Ref)
		}
	}
}

// impresa es un numero tal y como sale en el cuerpo del documento.
type impresa struct {
	cubo  string
	valor int
	ref   string
}

// numerosImpresos lee los numeros DEL TEXTO, no del modelo. Una linea que
// parezca una cifra y no se pueda leer entera no se salta en silencio: se dice.
// Saltarla haria que el recuento cuadrara por abajo y el test se leeria verde
// justo cuando el documento esta roto.
func numerosImpresos(t *testing.T, texto string) []impresa {
	t.Helper()
	var out []impresa
	for _, l := range strings.Split(texto, "\n") {
		if !strings.HasPrefix(l, "    ") || !strings.HasSuffix(l, "]") {
			continue
		}
		i := strings.LastIndex(l, "   [")
		if i < 0 {
			continue
		}
		ref := strings.TrimSuffix(l[i+4:], "]")
		izq := strings.TrimSpace(l[:i])
		j := strings.LastIndex(izq, " ")
		if j < 0 {
			t.Fatalf("linea con referencia y sin numero: %q", l)
		}
		v, err := strconv.Atoi(strings.TrimSpace(izq[j:]))
		if err != nil {
			t.Fatalf("linea con referencia y con un numero que no se lee: %q", l)
		}
		out = append(out, impresa{cubo: strings.TrimSpace(izq[:j]), valor: v, ref: ref})
	}
	return out
}

// LA LEY DE CONSERVACION, y crece sola: el dia que alguien anada una rama a una
// derivacion y se olvide de contarla, la suma se rompe sin que nadie tenga que
// acordarse de escribir el caso.
func TestTodoRepartoDelActaCuadra(t *testing.T) {
	a := componer(t, entradasCompletas(t))
	if !a.Cuadra() {
		t.Fatalf("el acta no cuadra: %v", a.Descuadres())
	}
	for _, s := range a.Secciones {
		for _, r := range s.Repartos {
			if r.Suma() != r.Universo {
				t.Errorf("%s / %s: %d != %d", s.Fuente, r.Rotulo, r.Suma(), r.Universo)
			}
		}
	}
	// Y CON EL DESCUADRE PUESTO A MANO, el acta lo dice en la primera pantalla y
	// no lo esconde. Sin esto, "cuadra" seria una propiedad que nadie ha visto
	// fallar.
	roto := a
	roto.Secciones = append([]Seccion(nil), a.Secciones...)
	roto.Secciones[0].Repartos = append([]Reparto(nil), a.Secciones[0].Repartos...)
	roto.Secciones[0].Repartos[0].Universo += 7
	if roto.Cuadra() {
		t.Fatal("un reparto con el universo cambiado sigue cuadrando")
	}
	if !strings.Contains(roto.Texto(), "NO cuadra") {
		t.Error("el acta descuadrada no lo dice donde se lee primero")
	}
}

// EL SEGUNDO CAMINO AL MISMO NUMERO. El acta enumera y el objeto cuenta, y los
// dos tienen que dar lo mismo.
//
// Lo que esto caza y lo que no, dicho para no venderlo de mas: los dos caminos
// comparten CoberturaDe y EstadoDe, asi que un fallo DENTRO de esa funcion los
// enganaria a los dos. Lo que si caza es lo que de verdad pasa, que es que el
// recorrido del acta se salte un elemento o lo meta en el cubo de al lado.
func TestLosCubosDelActaDicenLoMismoQueSuObjeto(t *testing.T) {
	e := entradasCompletas(t)
	a := componer(t, e)

	cuenta := e.Programa.Cuenta()
	for cob, n := range cuenta {
		c := cifra(t, a, DelProgramaDeAuditoria, "las unidades del alcance del programa", string(cob))
		if c.Valor() != n {
			t.Errorf("cobertura %q: el acta enumera %d y el programa cuenta %d", cob, c.Valor(), n)
		}
	}
	for est, n := range e.Campana.Cuenta() {
		c := cifra(t, a, DeLaCampanaDeAccesos, "los accesos de la instantanea", string(est))
		if c.Valor() != n {
			t.Errorf("estado %q: el acta enumera %d y la campana cuenta %d", est, c.Valor(), n)
		}
	}
	// La independencia: el acta la reparte por su cuenta y auditoria la calcula
	// por la suya. Se deduplica el par (sesion, unidad) en los dos lados porque
	// Auditar no dedupe las unidades de una sesion.
	pares := map[string]bool{}
	for _, cf := range e.Programa.Independencia(e.Responsables) {
		pares[cf.Sesion+"|"+cf.Unidad] = true
	}
	c := cifra(t, a, DelProgramaDeAuditoria,
		"las auditorias de una unidad (pares sesion-unidad distintos)",
		"el auditor es quien responde de la unidad")
	if c.Valor() != len(pares) {
		t.Errorf("independencia: el acta enumera %d conflictos y auditoria calcula %d",
			c.Valor(), len(pares))
	}
}

// ---------------------------------------------------------------------------
// El descargo, con su control positivo por seccion
// ---------------------------------------------------------------------------

// LA PUERTA DE LA REGLA 3, con el control POSITIVO que M47 pide: cada seccion de
// datos tiene su cubo de "no consta" CON ALGO DENTRO, y su frase pegada.
//
// Y la frase no se compara con una copia escrita al lado: se compara con la
// constante del paquete que produce el dato. Asi, si manana accesos cambia su
// frase y el acta se hubiera quedado con un duplicado, esto se pone rojo.
func TestCadaSeccionTraeSuNoConstaConLaFraseDeSuPaquete(t *testing.T) {
	a := componer(t, entradasCompletas(t))
	casos := []struct {
		fuente  Fuente
		reparto string
		cubo    string
		frase   string
	}{
		{DelProgramaDeAuditoria, "las unidades del alcance del programa", "sin auditar",
			auditoria.LaFraseDeLoNoAuditado},
		{DelProgramaDeAuditoria, "las auditorias de una unidad (pares sesion-unidad distintos)",
			"no consta quien responde de la unidad", auditoria.LaFraseDeLoSinResponsable},
		{DeLaCampanaDeAccesos, "los accesos de la instantanea", "sin revisar",
			accesos.LaFraseDeLoNoRevisado},
		{DeLaCampanaDeAccesos, "las lineas de datos del fichero",
			"ilegible y sin excusar, bloquea el cierre", accesos.LaFraseDeLoIlegible},
		{DeLosIncidentes, "los incidentes del periodo, por su clasificacion al cierre del periodo",
			"sin clasificacion que conste", incidente.LaFraseDeLoNoClasificado},
		{DeLosIncidentes, "las notificaciones esperadas de incidentes del periodo",
			"no consta remitida", incidente.LaFraseDeLoNoNotificado},
	}
	texto := a.Texto()
	for _, c := range casos {
		got := cifra(t, a, c.fuente, c.reparto, c.cubo)
		if got.Valor() == 0 {
			t.Errorf("%s / %s: el cubo esta vacio, asi que su descargo no lo recorre nadie y "+
				"una mutacion lo dejaria verde", c.fuente, c.cubo)
		}
		if !got.EsAusencia() {
			t.Errorf("%s / %s: no esta marcado como ausencia", c.fuente, c.cubo)
		}
		if got.Descargo.Texto != c.frase {
			t.Errorf("%s / %s: el descargo no es el de su paquete.\n  tiene: %q\n  esperaba: %q",
				c.fuente, c.cubo, got.Descargo.Texto, c.frase)
		}
		if !strings.Contains(sinEspacios(texto), sinEspacios(c.frase)) {
			t.Errorf("%s / %s: la frase no llega al documento", c.fuente, c.cubo)
		}
	}
	// Y LA FRASE VA PEGADA AL DATO, no en una nota al pie. Se mide igual que en
	// la UAR: entre el numero y su frase no puede haber otro numero.
	for _, s := range a.Secciones {
		for _, r := range s.Repartos {
			for _, c := range r.Cifras {
				if c.Valor() == 0 || c.Descargo.Vacia() {
					continue
				}
				linea := "    " + c.Cubo.Texto
				i := strings.Index(texto, linea)
				if i < 0 {
					t.Fatalf("el cubo %q no sale en el documento", c.Cubo.Texto)
				}
				j := strings.Index(sinEspacios(texto[i:]), sinEspacios(c.Descargo.Texto))
				if j < 0 || j > 400 {
					t.Errorf("%s / %s: la frase no va pegada al numero (distancia %d)",
						s.Fuente, c.Cubo.Texto, j)
				}
			}
		}
	}
}

// EL CONTROL POSITIVO DE LO QUE CONSTA Y NO ES CULPA. Es el hermano que se
// olvida: aqui el dato SI esta, y presentarlo solo acusa igual.
func TestLoQueConstaYNoEsCulpaTambienLlevaSuFrase(t *testing.T) {
	a := componer(t, entradasCompletas(t))
	casos := []struct {
		fuente  Fuente
		reparto string
		cubo    string
		frase   string
	}{
		{DelProgramaDeAuditoria, "las auditorias de una unidad (pares sesion-unidad distintos)",
			"el auditor es quien responde de la unidad", auditoria.LaFraseDeLaIndependencia},
		{DelProgramaDeAuditoria, "lo que el ciclo anterior dejo sin auditar",
			"ya no esta en el alcance de este ciclo", auditoria.LaFraseDeLaSalidaDelAlcance},
		{DelProgramaDeAuditoria, "las asignaciones de responsable aportadas",
			"no casa con ninguna unidad del alcance", LaFraseDeLaAsignacionQueNoCasa},
		{DeLosIncidentes, "los incidentes aportados",
			"conocido fuera del periodo, no lo cuenta este acta", LaFraseDeLoFueraDelPeriodo},
		{DeLosIncidentes, "los incidentes del periodo, por su clasificacion al cierre del periodo",
			"con dos clasificaciones en el mismo instante", LaFraseDelEmpateDeClasificacion},
		{DeLosIncidentes, "las notificaciones esperadas de incidentes del periodo",
			"consta remitida fuera del periodo", LaFraseDeLaRemisionFueraDelPeriodo},
		{DeLosIncidentes, "las notificaciones que el corpus espera",
			"no casa con ningun incidente del periodo", LaFraseDeLaNotificacionQueNoCasa},
		{DeLaCampanaDeAccesos, "las lineas de datos del fichero", "ilegible, excusada por escrito",
			accesos.LaFraseDeLoIlegible},
	}
	for _, c := range casos {
		got := cifra(t, a, c.fuente, c.reparto, c.cubo)
		if got.Valor() == 0 {
			t.Errorf("%s / %s: vacio, asi que la frase no la recorre ninguna entrada", c.fuente, c.cubo)
		}
		if got.Descargo.Texto != c.frase {
			t.Errorf("%s / %s: frase %q", c.fuente, c.cubo, got.Descargo.Texto)
		}
	}
}

// NINGUN CUBO DEL ACTA SE QUEDA SIN RECORRER. Es la puerta que crece sola: si
// alguien anade un cubo nuevo y no le da datos en el caso completo, esto avisa,
// que es exactamente lo que fallo en M47.
func TestElCasoCompletoRecorreTodosLosCubos(t *testing.T) {
	a := componer(t, entradasCompletas(t))
	var vacios []string
	for _, c := range a.Cifras() {
		if c.Cifra.Valor() == 0 {
			vacios = append(vacios, c.Ref+" "+string(c.Fuente)+" / "+c.Cifra.Cubo.Texto)
		}
	}
	if len(vacios) > 0 {
		t.Errorf("el caso completo deja %d cubos sin recorrer, asi que sus ramas no las prueba "+
			"nadie:\n  %s", len(vacios), strings.Join(vacios, "\n  "))
	}
}

// EL CENTINELA. Una ausencia sin frase no se compone, y no porque lo diga un
// test: porque Componer no la deja pasar.
func TestUnaAusenciaSinDescargoNoSeCompone(t *testing.T) {
	a := componer(t, entradasCompletas(t))
	a.Secciones[0].Repartos[0].Cifras[0] = Cifra{
		Cubo:      cuboDeCobertura(auditoria.SinAuditar),
		Elementos: []Elemento{{Clave: "p1|o2"}}, exigeDescargo: true,
	}
	err := a.validar()
	if !errors.Is(err, ErrAusenciaSinDescargo) {
		t.Fatalf("un cubo de lo que no consta sin su frase ha pasado la puerta: %v", err)
	}
	// Y la otra direccion: con la frase puesta, pasa. Sin esto, el test estaria
	// contento con una puerta que rechaza siempre.
	a.Secciones[0].Repartos[0].Cifras[0].Descargo = descargoNoAuditado
	if err := a.validar(); err != nil {
		t.Fatalf("con la frase puesta tenia que pasar: %v", err)
	}
}

// ---------------------------------------------------------------------------
// La prosa, y por que no puede haberla generada
// ---------------------------------------------------------------------------

func TestTodaLaProsaDelActaDiceDeDondeSalenSusPalabras(t *testing.T) {
	a := componer(t, entradasCompletas(t))
	if len(a.Parrafos()) == 0 {
		t.Fatal("un acta sin prosa")
	}
	for _, p := range a.Parrafos() {
		if err := p.validar(); err != nil {
			t.Errorf("parrafo sin procedencia util: %v", err)
		}
	}
	// LA LEY DE CONSERVACION DE LA PROSA: todo parrafo cae en exactamente una
	// procedencia y la suma da el total. Si manana alguien mete un cuarto sitio
	// de donde puedan salir palabras, esto no cuadra.
	prosa := a.Prosa()
	n := 0
	for _, p := range ProcedenciasPosibles() {
		n += len(prosa[p])
	}
	if n != len(a.Parrafos()) {
		t.Fatalf("hay %d parrafos y las procedencias conocidas explican %d", len(a.Parrafos()), n)
	}
	if len(ProcedenciasPosibles()) != 3 {
		t.Fatalf("el vocabulario de procedencias tiene %d valores y son tres: lo escribe plazum, "+
			"lo escribe una persona, es una cita. Un cuarto valor es donde entraria la prosa "+
			"generada", len(ProcedenciasPosibles()))
	}
	// Las tres estan recorridas de verdad, no solo declaradas.
	for _, p := range ProcedenciasPosibles() {
		if len(prosa[p]) == 0 {
			t.Errorf("ningun parrafo del caso completo es %q, asi que esa rama no la prueba nadie", p)
		}
	}
	// Y las dos que son de alguien lo dicen en el documento.
	texto := a.Texto()
	for _, p := range prosa[DeUnaPersona] {
		if !strings.Contains(texto, "-- "+p.Quien) {
			t.Errorf("el parrafo de %s sale sin decir de quien es", p.Quien)
		}
	}
	for _, p := range prosa[DeLaNorma] {
		if !strings.Contains(texto, p.Cita) {
			t.Errorf("una cita sale sin su fuente: %q", p.Texto)
		}
	}
	// Control negativo: una procedencia fuera del vocabulario no pasa.
	mala := Parrafo{Frase: Frase{Texto: "esto lo ha redactado algo"}, De: Procedencia(9)}
	if err := mala.validar(); !errors.Is(err, ErrProsaSinProcedencia) {
		t.Fatalf("una procedencia desconocida ha pasado: %v", err)
	}
}

// LA PUERTA DE LA FALSIFICACION. Una decision del organo de gobierno que no
// venga de una persona no se compone.
//
// Es la unica seccion del acta cuyo contenido es un juicio, y es exactamente
// donde una prosa generada haria mas dano: firmada por el consejo, diciendo que
// el sistema funciona, escrita por una maquina.
func TestUnaDecisionDeLaDireccionQueNoEsDeUnaPersonaNoSeCompone(t *testing.T) {
	e := entradasCompletas(t)
	e.Decisiones = []Parrafo{{
		Frase: Frase{Texto: "El consejo considera que el sistema de gestion es eficaz y adecuado."},
		De:    DePlazum,
	}}
	_, err := Componer(e)
	if !errors.Is(err, ErrProsaSinProcedencia) {
		t.Fatalf("plazum ha podido redactar una conclusion del consejo: %v", err)
	}
	// Y sin decisiones, el acta lo dice con todas las letras en vez de rellenar
	// el hueco con un parrafo verosimil.
	e.Decisiones = nil
	a := componer(t, e)
	s, ok := a.Busca(DeLaDireccion)
	if !ok {
		t.Fatal("falta la seccion de la direccion")
	}
	if s.Aportada {
		t.Error("una seccion sin decisiones se presenta como aportada")
	}
	if !strings.Contains(sinEspacios(a.Texto()), sinEspacios(LaFraseDeLoQuePlazumNoEscribe)) {
		t.Error("el acta no dice por que no trae conclusiones redactadas")
	}
}

// EL BOARD PACK SALE CON LA IA APAGADA, y aqui NO hay test que lo compruebe, a
// proposito.
//
// Lo comprueba TestElNucleoNoConoceLaIA en la raiz, que recorre el AST de nucleo/
// entero y ademas corre con PLAZUM_SIN_IA=1 en CI. Hubo aqui una version local, y
// la puerta de la raiz la tumbo el primer dia: para buscar el nombre del puerto
// habia que escribirlo, asi que el detector era la infraccion. Escribirlo por
// trozos para esquivar la puerta habria sido justo la evasion que esa puerta
// documenta que no admite.
//
// Y de paso saco un falso positivo de verdad: el campo de este paquete que
// guardaba quien asistio a la revision se llamaba igual que el puerto de IA en
// plural, y la puerta busca esa palabra tal cual. Se renombro a QuienAsistio, que
// es lo que el mensaje de la propia puerta manda hacer con una palabra que
// coincide. Y esta nota no la escribe entera por lo mismo: la puerta lee tambien
// los comentarios, y un detector que hay que explicar escribiendo lo que busca es
// un detector que se acaba desactivando.

// ---------------------------------------------------------------------------
// Las dos formas de la nada, y la tercera
// ---------------------------------------------------------------------------

// "Cero incidentes" y "no hay registro" no son lo mismo, y el acta no los
// confunde. Es el invariante 8 en la frontera de entrada de este paquete.
func TestCeroIncidentesYSinRegistroNoSonLoMismo(t *testing.T) {
	base := entradasCompletas(t)

	sin := base
	sin.HayRegistroDeIncidentes = false
	sin.Incidentes = nil
	sin.Esperadas = nil
	a := componer(t, sin)
	s, _ := a.Busca(DeLosIncidentes)
	if s.Aportada {
		t.Error("sin registro, la seccion se presenta como aportada")
	}
	if !strings.Contains(s.PorQueFalta, "NO dice que no haya habido ninguno") {
		t.Errorf("el hueco se presenta como una noticia: %q", s.PorQueFalta)
	}

	tranquilo := base
	tranquilo.HayRegistroDeIncidentes = true
	tranquilo.Incidentes = nil
	tranquilo.Esperadas = nil
	b := componer(t, tranquilo)
	s2, _ := b.Busca(DeLosIncidentes)
	if !s2.Aportada {
		t.Error("con registro y cero incidentes, la seccion tenia que estar aportada")
	}
	if len(s2.Repartos) == 0 {
		t.Fatal("un periodo tranquilo se queda sin cubos, asi que nadie ve que se miro")
	}
	// Y LOS CUBOS SALEN IGUAL, a cero. Uno que solo aparece cuando tiene algo
	// dentro es un cubo que nadie echa de menos.
	if !strings.Contains(b.Texto(), "sin clasificacion que conste") {
		t.Error("el cubo vacio no se pinta")
	}

	// Y la combinacion imposible se rechaza en vez de elegir una de las dos.
	raro := base
	raro.HayRegistroDeIncidentes = false
	if _, err := Componer(raro); !errors.Is(err, ErrActa) {
		t.Fatalf("incidentes sin registro: %v", err)
	}
}

// LA TERCERA HERMANA: presente y no interpretable. Un incidente sin apertura no
// se cuenta en un cubo por defecto, da error.
func TestUnIncidenteSinAperturaNoSeCuentaEnNingunCubo(t *testing.T) {
	e := entradasCompletas(t)
	e.Incidentes = append(e.Incidentes, &incidente.Incidente{})
	_, err := Componer(e)
	if !errors.Is(err, ErrActa) {
		t.Fatalf("un incidente sin apertura ha entrado en el acta: %v", err)
	}
	if !strings.Contains(err.Error(), "incidente.Abrir") {
		t.Errorf("el error no dice como se arregla: %v", err)
	}
	// Un nil tampoco, y por la misma puerta.
	e.Incidentes[len(e.Incidentes)-1] = nil
	if _, err := Componer(e); !errors.Is(err, ErrActa) {
		t.Fatalf("un incidente nil ha entrado: %v", err)
	}
}

// Y LAS SECCIONES SIN FUENTE SALEN IGUAL, con lo que hace falta para tenerlas.
func TestElActaSiempreLlevaLasCuatroSecciones(t *testing.T) {
	a := componer(t, Entradas{
		ID: "acta-vacia", Organizacion: "Molduras del Norte SL", Periodo: elPeriodo,
	})
	if len(a.Secciones) != len(FuentesPosibles()) {
		t.Fatalf("un acta sin datos trae %d secciones y el vocabulario tiene %d",
			len(a.Secciones), len(FuentesPosibles()))
	}
	for i, f := range FuentesPosibles() {
		if a.Secciones[i].Fuente != f {
			t.Fatalf("la seccion %d es %q y tenia que ser %q", i, a.Secciones[i].Fuente, f)
		}
		if a.Secciones[i].Aportada {
			t.Errorf("%q se presenta como aportada sin fuente", f)
		}
		if strings.TrimSpace(a.Secciones[i].PorQueFalta) == "" {
			t.Errorf("%q esta vacia y no dice que hace falta: un estado vacio sin verbo es un "+
				"callejon", f)
		}
	}
	// El documento sale entero y sin fingir que tiene datos.
	texto := a.Texto()
	for _, f := range FuentesPosibles() {
		if !strings.Contains(texto, strings.ToUpper(string(f))) {
			t.Errorf("la seccion %q no llega al documento", f)
		}
	}
	if !a.Cuadra() {
		t.Errorf("un acta vacia tampoco cuadra: %v", a.Descuadres())
	}
}

// ---------------------------------------------------------------------------
// De que verbo cuelga cada instante
// ---------------------------------------------------------------------------

// UN INCIDENTE ES DEL PERIODO POR SU PRIMER CONOCIMIENTO, no por cuando ocurrio.
// Escoger mal no da error: da un acta que cuadra y miente.
func TestUnIncidenteEsDelPeriodoPorCuandoSeSupo(t *testing.T) {
	e := entradasCompletas(t)
	// Ocurrio ANTES del periodo y se supo DENTRO: es de este acta.
	viejo := abrir(t, "INC-VIEJO", dia(2024, 3, 1), dia(2026, 8, 1))
	// Ocurrio DENTRO y se supo DESPUES: no es de este acta.
	tardio := abrir(t, "INC-TARDIO", dia(2026, 12, 30), dia(2027, 3, 1))
	e.Incidentes = append(e.Incidentes, viejo, tardio)
	a := componer(t, e)

	dentro := cifra(t, a, DeLosIncidentes, "los incidentes aportados", "conocido dentro del periodo")
	fuera := cifra(t, a, DeLosIncidentes, "los incidentes aportados",
		"conocido fuera del periodo, no lo cuenta este acta")
	if !tiene(dentro, "INC-VIEJO") {
		t.Error("un incidente que ocurrio antes y se supo dentro no cuenta en este acta, y la " +
			"revision solo puede juzgar lo que se pudo hacer desde que se supo")
	}
	if !tiene(fuera, "INC-TARDIO") {
		t.Error("un incidente que se supo despues del periodo cuenta en este acta")
	}
}

// LA CLASIFICACION SE LEE AL CIERRE DEL PERIODO. Una reclasificacion posterior
// no reescribe un acta ya cerrada.
func TestUnaReclasificacionPosteriorNoReescribeElActa(t *testing.T) {
	e := entradasCompletas(t)
	tarde := abrir(t, "INC-TARDE", dia(2026, 11, 1), dia(2026, 11, 1))
	e.Incidentes = append(e.Incidentes, tarde)
	antes := componer(t, e)
	if !tiene(cifra(t, antes, DeLosIncidentes,
		"los incidentes del periodo, por su clasificacion al cierre del periodo",
		"sin clasificacion que conste"), "INC-TARDE") {
		t.Fatal("el caso no arranca donde dice arrancar")
	}
	registrar(t, tarde, incidente.Suceso{Tipo: incidente.Clasificacion,
		Clase: "incidente.nivel.alto", InstanteHecho: dia(2027, 4, 1),
		InstanteRegistro: dia(2027, 4, 1), Fuente: "legal"})
	despues := componer(t, e)
	if !tiene(cifra(t, despues, DeLosIncidentes,
		"los incidentes del periodo, por su clasificacion al cierre del periodo",
		"sin clasificacion que conste"), "INC-TARDE") {
		t.Error("una clasificacion escrita en 2027 ha cambiado el acta de 2026")
	}
}

// UNA NOTIFICACION REMITIDA FUERA DEL PERIODO NO ES NI HECHA NI NO CONSTATADA.
// Meterla en cualquiera de los otros dos cubos miente en una direccion o en la
// otra, y la cara mala empuja a notificar otra vez a un supervisor.
func TestUnaRemisionFueraDelPeriodoTieneSuPropioCubo(t *testing.T) {
	a := componer(t, entradasCompletas(t))
	rep := "las notificaciones esperadas de incidentes del periodo"
	fuera := cifra(t, a, DeLosIncidentes, rep, "consta remitida fuera del periodo")
	dentro := cifra(t, a, DeLosIncidentes, rep, "consta remitida dentro del periodo")
	noConsta := cifra(t, a, DeLosIncidentes, rep, "no consta remitida")
	if !tiene(fuera, "INC-5|n-inicial") {
		t.Error("una notificacion remitida despues del periodo no esta en su cubo")
	}
	if tiene(dentro, "INC-5|n-inicial") || tiene(noConsta, "INC-5|n-inicial") {
		t.Error("una notificacion remitida fuera del periodo se cuenta como hecha o como no " +
			"constatada, y no es ninguna de las dos")
	}
	if !tiene(noConsta, "INC-2|n-inicial") {
		t.Error("una notificacion que no consta no esta en su cubo")
	}
}

// LAS DOS DIRECCIONES DE CADA EMPAREJAMIENTO. Lo que no casa no se traga.
func TestLoQueNoCasaConNadaSeCuentaYSeDice(t *testing.T) {
	a := componer(t, entradasCompletas(t))
	noCasa := cifra(t, a, DelProgramaDeAuditoria, "las asignaciones de responsable aportadas",
		"no casa con ninguna unidad del alcance")
	if !tiene(noCasa, "p7|x") {
		t.Error("un responsable asignado a algo que este programa no audita ha desaparecido")
	}
	suelta := cifra(t, a, DeLosIncidentes, "las notificaciones que el corpus espera",
		"no casa con ningun incidente del periodo")
	if !tiene(suelta, "INC-4|n-inicial") {
		t.Error("una notificacion esperada de un incidente que no cuenta este acta ha desaparecido")
	}
	// Y si desaparecieran, el reparto no cuadraria: el universo se cuenta aparte.
	// Esa es la propiedad que hace que esto no dependa de que alguien se acuerde.
	e := entradasCompletas(t)
	e.Responsables["p8|y"] = "otro mas"
	b := componer(t, e)
	if !b.Cuadra() {
		t.Fatalf("anadir una asignacion suelta descuadra el acta: %v", b.Descuadres())
	}
	if cifra(t, b, DelProgramaDeAuditoria, "las asignaciones de responsable aportadas",
		"no casa con ninguna unidad del alcance").Valor() != noCasa.Valor()+1 {
		t.Error("la asignacion nueva no ha llegado a ningun cubo")
	}
}

// LA PROSA DE UNA PERSONA QUE VIAJA DENTRO DE UNA NOTA VA DETRAS DE SU NOMBRE.
// Es la via por la que la regla 2 se podria colar: una Nota no es un Parrafo y no
// lleva procedencia, asi que la atribucion tiene que ir en la misma cadena.
func TestLaProsaAjenaDentroDeUnaNotaViajaConSuAutor(t *testing.T) {
	a := componer(t, entradasCompletas(t))
	casos := []struct{ fragmento, autor string }{
		{"la copia mensual de marzo no tiene registro de restauracion", "aud-1"},
		{"el sistema se migra en el ultimo trimestre", "jefa"},
		{"el export corto una fila a mitad", "u-042"},
	}
	for _, c := range casos {
		nota := ""
		for _, cf := range a.Cifras() {
			for _, el := range cf.Cifra.Elementos {
				if strings.Contains(el.Nota, c.fragmento) {
					nota = el.Nota
				}
			}
		}
		if nota == "" {
			t.Errorf("el texto %q no llega al acta, asi que este caso no prueba nada", c.fragmento)
			continue
		}
		if !strings.Contains(nota, c.autor) {
			t.Errorf("la nota lleva palabras de alguien y no dice de quien: %q", nota)
		}
		if strings.Index(nota, c.autor) > strings.Index(nota, c.fragmento) {
			t.Errorf("el autor va detras de sus palabras y tiene que ir delante: %q", nota)
		}
	}
}

// ---------------------------------------------------------------------------
// Utilidades del test
// ---------------------------------------------------------------------------

func tiene(c Cifra, clave string) bool {
	for _, el := range c.Elementos {
		if el.Clave == clave {
			return true
		}
	}
	return false
}

// sinEspacios normaliza para poder comparar una frase con su version envuelta en
// lineas: el documento parte los parrafos y la constante no.
func sinEspacios(s string) string { return strings.Join(strings.Fields(s), " ") }

// LOS ERRORES DICEN QUE FALTA Y POR QUE IMPORTA, que es la diferencia entre un
// mensaje que se arregla y uno que se busca en un buscador.
func TestComponerDiceQueFaltaYPorQue(t *testing.T) {
	casos := []struct {
		nombre string
		toca   func(*Entradas)
		dice   string
	}{
		{"sin id", func(e *Entradas) { e.ID = "" }, "id del acta"},
		{"sin organizacion", func(e *Entradas) { e.Organizacion = " " }, "no es evidencia de nadie"},
		{"sin periodo", func(e *Entradas) { e.Periodo = Periodo{} }, "que incidente entra"},
		{"periodo del reves", func(e *Entradas) {
			e.Periodo = Periodo{Desde: hasta, Hasta: desde}
		}, "hasta despues de desde"},
		{"esperada sin hito", func(e *Entradas) {
			e.Esperadas = append(e.Esperadas, Notificacion{Incidente: "INC-1"})
		}, "no casa con nada"},
		{"esperada repetida", func(e *Entradas) {
			e.Esperadas = append(e.Esperadas, e.Esperadas[0])
		}, "viene dos veces"},
		{"incidente repetido", func(e *Entradas) {
			e.Incidentes = append(e.Incidentes, e.Incidentes[0])
		}, "lo cuenta dos veces"},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			e := entradasCompletas(t)
			c.toca(&e)
			_, err := Componer(e)
			if err == nil {
				t.Fatal("se ha compuesto un acta que no se puede componer")
			}
			if !errors.Is(err, ErrActa) {
				t.Fatalf("centinela: %v", err)
			}
			if !strings.Contains(err.Error(), c.dice) {
				t.Errorf("el error no dice por que importa: %v", err)
			}
		})
	}
	// Y una referencia que no existe no devuelve una cifra vacia con cara de
	// dato: dice que no.
	a := componer(t, entradasCompletas(t))
	if _, ok := a.Derivar("9.9.9"); ok {
		t.Error("una referencia inventada ha resuelto")
	}
}

// NINGUN DESCARGO DEL ACTA ACUSA, y este test es el que sobrevive a la mutacion
// que los otros dos no.
//
// EL PUNTO CIEGO QUE VIENE A TAPAR: los tests de arriba comparan el descargo del
// acta con la CONSTANTE de su paquete, asi que si alguien reescribe la constante
// y pone "lo has incumplido", los dos lados cambian a la vez y siguen verdes. Es
// exactamente la trampa de "si el test prueba contra una lista escrita a su lado,
// muta FUERA de esa lista": aquellos vigilan que el acta no se separe de su
// fuente, y este vigila la fuente.
//
// Se comprueba la FORMA, que es lo unico que se puede comprobar sin escribir las
// frases otra vez: el patron de la casa es "Esto NO dice <lo que se podria leer>:
// dice <lo que de verdad consta>". Las dos mitades. Una frase que solo tenga la
// segunda informa; una que solo tenga la primera niega sin decir que hay.
func TestNingunDescargoDelActaAcusa(t *testing.T) {
	a := componer(t, entradasCompletas(t))
	vistos := 0
	for _, c := range a.Cifras() {
		if c.Cifra.Descargo.Vacia() {
			continue
		}
		vistos++
		d := c.Cifra.Descargo.Texto
		if !strings.HasPrefix(d, "Esto NO dice") {
			t.Errorf("[%s] %s: el descargo no empieza negando lo que se podria leer mal, asi "+
				"que se lee como la acusacion: %q", c.Ref, c.Cifra.Cubo.Texto, d)
		}
		if !strings.Contains(d, ": dice que") {
			t.Errorf("[%s] %s: el descargo niega y no dice que es lo que si consta, que es la "+
				"mitad util: %q", c.Ref, c.Cifra.Cubo.Texto, d)
		}
	}
	if vistos < 10 {
		t.Fatalf("solo %d cubos del acta llevan descargo: o falta alguno o este test esta "+
			"mirando un acta a medias", vistos)
	}
}

// LA BARRA DENTRO DE UNA IDENTIDAD, la otra mitad del hallazgo que salio en
// auditoria: aqui la clave es incidente|hito.
func TestUnaIdentidadConLaBarraDentroNoEntraEnElActa(t *testing.T) {
	e := entradasCompletas(t)
	e.Incidentes = append(e.Incidentes, abrir(t, "INC|A", dia(2026, 4, 1), dia(2026, 4, 1)))
	if _, err := Componer(e); !errors.Is(err, ErrActa) {
		t.Fatalf("un id de incidente con la barra dentro ha entrado: %v", err)
	}

	e = entradasCompletas(t)
	// Las dos son notificaciones DISTINTAS y dan la misma clave: "INC-1|n|x".
	e.Esperadas = append(e.Esperadas,
		Notificacion{Incidente: "INC-1", Hito: "n|x", Que: "una"})
	_, err := Componer(e)
	if !errors.Is(err, ErrActa) {
		t.Fatalf("un hito con la barra dentro ha entrado: %v", err)
	}
	if !strings.Contains(err.Error(), "contaria una donde hay dos") {
		t.Errorf("el error no dice que es lo que se pierde: %v", err)
	}
}

// LO PRIMERO QUE NECESITA QUIEN NO HA VISTO PLAZUM NUNCA: de cuantas de sus
// cuatro fuentes hay registro. Sale de la pasada contra el comprador.
//
// Sin este bloque, un acta con la mitad de las fuentes sin conectar se lee
// exactamente igual de completa que una entera, porque las cuatro secciones
// salen siempre. Salen siempre a proposito, y por eso hace falta decirlo arriba.
func TestElActaDiceArribaDeQueFuentesHayRegistro(t *testing.T) {
	completa := componer(t, entradasCompletas(t)).Texto()
	cabecera := completa[:strings.Index(completa, "PROGRAMA DE AUDITORIA")]
	for _, f := range FuentesPosibles() {
		if !strings.Contains(cabecera, "SI  "+string(f)) {
			t.Errorf("con todas las fuentes, %q no sale como disponible en la cabecera", f)
		}
	}
	e := entradasCompletas(t)
	e.Campana = nil
	e.HayRegistroDeIncidentes = false
	e.Incidentes, e.Esperadas = nil, nil
	media := componer(t, e).Texto()
	cabecera = media[:strings.Index(media, "PROGRAMA DE AUDITORIA")]
	if !strings.Contains(cabecera, "NO  "+string(DeLaCampanaDeAccesos)) {
		t.Error("una fuente que falta no se dice en la cabecera, asi que el acta se lee entera")
	}
	if !strings.Contains(cabecera, "NO  "+string(DeLosIncidentes)) {
		t.Error("el registro de incidentes falta y la cabecera no lo dice")
	}
	if !strings.Contains(cabecera, "SI  "+string(DelProgramaDeAuditoria)) {
		t.Error("la fuente que si esta no se distingue de las que no")
	}
	// Y con el motivo al lado: un estado vacio sin verbo es un callejon.
	if !strings.Contains(sinEspacios(cabecera), "abrir la campana sobre ella") {
		t.Error("la cabecera dice que falta y no dice que hace falta para tenerlo")
	}
}

// EL VALOR CERO DE LAS OPCIONES DEL ACTA NO PUBLICA NOMBRES DEL CENSO.
//
// Invariante 8 en la frontera que de verdad importa aqui: el acta ES LA PIEZA
// QUE VIAJA. Si el valor cero llevara los rotulos, olvidarse de apagarlos en el
// documento que mas circula del producto seria publicar un directorio de
// empleados sin querer, y el olvido es lo que sale por defecto.
//
// Las DOS formas de la nada, las dos recorridas: unas Entradas sin tocar el
// campo, y unas con el campo puesto a false expresamente.
func TestElValorCeroDelActaNoSacaNombresDelCenso(t *testing.T) {
	rotulos := []string{"Bea Nunez", "Carlos Ortiz", "Eva Diaz"}

	sinTocar := entradasCompletas(t) // el campo ni se nombra
	expreso := entradasCompletas(t)
	expreso.ConNombresDelCenso = false
	for nombre, e := range map[string]Entradas{"sin tocar el campo": sinTocar, "a false": expreso} {
		a := componer(t, e)
		texto := a.Texto()
		for _, r := range rotulos {
			if strings.Contains(texto, r) {
				t.Errorf("%s: el acta saca %q, que es el nombre con el que el IdP llama a una "+
					"cuenta revisada, y este documento se imprime y se manda por correo",
					nombre, r)
			}
		}
		// Y la identidad SI viaja: sin ella el numero no se puede abrir, que es
		// lo que separa minimizar de esconder.
		if !strings.Contains(texto, "erp|u3|admin") {
			t.Errorf("%s: sin la identidad de la fila, el cubo no se puede abrir", nombre)
		}
		// Y LOS ACTORES TAMBIEN, que es la otra mitad de la linea: un acta que no
		// dice quien hizo que no es evidencia de nada.
		for _, actor := range []string{"jefa", "aud-1", "u-042", "Ana Perez (consejera delegada)"} {
			if !strings.Contains(texto, actor) {
				t.Errorf("%s: falta %q, que es quien hizo algo y no un sujeto revisado",
					nombre, actor)
			}
		}
	}

	// Y con el interruptor puesto a mano, salen. Sin este control positivo, el
	// test de arriba estaria contento con un campo que no hace nada.
	con := entradasCompletas(t)
	con.ConNombresDelCenso = true
	if !strings.Contains(componer(t, con).Texto(), "Eva Diaz") {
		t.Error("con el interruptor puesto, el rotulo del censo tenia que salir")
	}
}
