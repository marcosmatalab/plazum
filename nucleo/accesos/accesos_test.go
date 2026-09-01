package accesos

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/marcosmatalab/plazum/nucleo/censo"
)

var (
	t0 = time.Date(2026, 9, 1, 9, 0, 0, 0, time.UTC)
	t1 = t0.Add(time.Hour)
	t2 = t0.Add(2 * time.Hour)
)

func instantanea(t *testing.T, csv string) censo.Instantanea {
	t.Helper()
	ins, err := censo.Tomar([]byte(csv), censo.Opciones{
		Sistema:   "erp",
		Fuente:    "export de usuarios",
		Quien:     "u-042",
		Tomada:    t0,
		Retencion: "12 meses tras el cierre",
		Columnas:  censo.ColumnasHabituales(),
	})
	if err != nil {
		t.Fatalf("la instantanea no carga: %v", err)
	}
	return ins
}

const censoBase = "usuario;nombre;permiso\n" +
	"u1;Ana Perez;admin\n" +
	"u1;Ana Perez;lector\n" +
	"u2;Luis Gil;lector\n"

func campana(t *testing.T, csv string, revisores map[string]string) *Campana {
	t.Helper()
	c, err := Abrir("uar-2026-h2", instantanea(t, csv), t0, revisores)
	if err != nil {
		t.Fatalf("no abre: %v", err)
	}
	return c
}

func todosRevisadosPor(t *testing.T, c *Campana, quien string) {
	t.Helper()
	for _, f := range c.Instantanea().Filas {
		if err := c.Registrar(Decision{Fila: f.Clave(), Veredicto: Aprobar, Quien: quien, Cuando: t1}); err != nil {
			t.Fatalf("registrando %s: %v", f.Clave(), err)
		}
	}
}

// LA GUARDA DEL INVARIANTE 7. Una decision solo casa con una fila de ESTA
// instantanea, y casa por la clave, que va dentro de lo sellado.
//
// Sin esto, el recuento de "decididos" puede subir sin que ningun acceso real
// cambie de cubo, que es la forma exacta en la que una campana se certifica a si
// misma.
func TestUnaDecisionSobreUnaFilaQueNoEstaSeRechaza(t *testing.T) {
	c := campana(t, censoBase, nil)
	err := c.Registrar(Decision{Fila: "erp|u9|admin", Veredicto: Aprobar, Quien: "jefa", Cuando: t1})
	if err == nil {
		t.Fatal("se ha admitido una decision sobre un acceso que no esta en la instantanea")
	}
	if !errors.Is(err, ErrFilaDesconocida) {
		t.Fatalf("centinela: %v", err)
	}
	if c.Informar().Decididos != 0 {
		t.Error("la decision rechazada ha contado igual")
	}
}

// Y EL EMPAREJAMIENTO NO ES POR POSICION: reordenar el CSV no mueve ni una
// decision. Es la otra mitad del invariante 7, y es la que se comprueba poco.
func TestReordenarElFicheroNoMueveNingunaDecision(t *testing.T) {
	otroOrden := "usuario;nombre;permiso\n" +
		"u2;Luis Gil;lector\n" +
		"u1;Ana Perez;lector\n" +
		"u1;Ana Perez;admin\n"

	a := campana(t, censoBase, nil)
	b := campana(t, otroOrden, nil)

	// El fichero es OTRO (otros bytes), asi que el hash y el sello cambian: eso
	// es correcto y es lo que hace reproducible la revision.
	if a.Instantanea().Hash == b.Instantanea().Hash {
		t.Fatal("dos ficheros distintos comparten hash")
	}
	// Pero las CLAVES son las mismas, que es lo que sostiene las decisiones.
	claves := func(c *Campana) map[string]bool {
		m := map[string]bool{}
		for _, f := range c.Instantanea().Filas {
			m[f.Clave()] = true
		}
		return m
	}
	ca, cb := claves(a), claves(b)
	if len(ca) != len(cb) {
		t.Fatalf("distinto numero de claves: %d / %d", len(ca), len(cb))
	}
	for k := range ca {
		if !cb[k] {
			t.Fatalf("la clave %q no esta en el fichero reordenado: el emparejamiento depende "+
				"del orden, que es justo lo que el invariante 7 prohibe", k)
		}
	}
	// Y las mismas decisiones dan los mismos cubos en las dos.
	d := Decision{Fila: "erp|u1|admin", Veredicto: Revocar, Quien: "jefa", Cuando: t1,
		Motivo: "cambio de puesto en junio"}
	for _, c := range []*Campana{a, b} {
		if err := c.Registrar(d); err != nil {
			t.Fatal(err)
		}
		if c.EstadoDe("erp|u1|admin") != Revocada {
			t.Fatalf("la decision no ha llegado a su fila: %v", c.Cuenta())
		}
		if c.EstadoDe("erp|u2|lector") != SinRevisar {
			t.Fatal("la decision ha tocado una fila que no era")
		}
	}
}

// DELEGAR NO ES DECIDIR. Es el veredicto que mas facil seria colar como cierre:
// deja el acceso sin revisar y mueve al revisor.
func TestDelegarTrasladaLaRevisionPeroNoLaTermina(t *testing.T) {
	c := campana(t, censoBase, map[string]string{"erp|u1|admin": "jefa"})
	err := c.Registrar(Decision{Fila: "erp|u1|admin", Veredicto: Delegar, Quien: "jefa",
		Cuando: t1, A: "responsable-erp"})
	if err != nil {
		t.Fatal(err)
	}
	estado := c.EstadoDe("erp|u1|admin")
	if estado != Delegada {
		t.Fatalf("estado tras delegar: %q", estado)
	}
	if estado.Termina() {
		t.Fatal("Delegada no puede ser un estado terminal")
	}
	// El revisor se MUEVE: si no, el acceso se queda esperando a quien ya dijo
	// que no era suyo.
	if r, _ := c.RevisorDe("erp|u1|admin"); r != "responsable-erp" {
		t.Errorf("el revisor sigue siendo %q despues de delegar", r)
	}
	// Y una delegacion sin destinatario no se admite.
	if err := c.Registrar(Decision{Fila: "erp|u2|lector", Veredicto: Delegar, Quien: "jefa",
		Cuando: t1}); err == nil {
		t.Error("se admite una delegacion sin decir a quien")
	}
}

// EL CIERRE NO SE FIRMA CON NADA PENDIENTE, y el error dice exactamente que.
func TestElCierreNoSeFirmaMientrasQuedeAlgoSinRevisar(t *testing.T) {
	c := campana(t, censoBase, nil)
	if err := c.Registrar(Decision{Fila: "erp|u1|admin", Veredicto: Aprobar, Quien: "jefa",
		Cuando: t1}); err != nil {
		t.Fatal(err)
	}
	if err := c.Registrar(Decision{Fila: "erp|u1|lector", Veredicto: Delegar, Quien: "jefa",
		Cuando: t1, A: "otra"}); err != nil {
		t.Fatal(err)
	}
	_, err := c.Cerrar("ciso", t2)
	if err == nil {
		t.Fatal("se ha firmado una campana con un acceso sin tocar y otro delegado")
	}
	if !errors.Is(err, ErrSinCerrar) {
		t.Fatalf("centinela: %v", err)
	}
	for _, quiero := range []string{"1 sin revisar", "1 delegados", "delegar traslada la revision"} {
		if !strings.Contains(err.Error(), quiero) {
			t.Errorf("el error no dice %q:\n%v", quiero, err)
		}
	}
	if c.Cerrada() {
		t.Fatal("la campana se ha marcado cerrada aunque el cierre fallo")
	}
}

// UNA DELEGACION SOLA BLOQUEA EL CIERRE.
//
// ESTE TEST EXISTE POR UN VERDE. La mutacion que hacia contar las delegaciones
// como decididas (M95) no puso nada rojo, porque el caso de arriba tiene ADEMAS
// un acceso sin tocar y el cierre fallaba igual por el otro motivo. La
// afirmacion sobre las delegaciones viajaba de gorra en un fallo ajeno.
//
// Es la familia de M47 otra vez: la rama existia y ninguna entrada la activaba
// SOLA. Aqui el dato es el minimo que la aisla: todo decidido menos una
// delegacion.
func TestUnaDelegacionPendienteBloqueaElCierreEllaSola(t *testing.T) {
	c := campana(t, censoBase, nil)
	filas := c.Instantanea().Filas
	for _, f := range filas[1:] {
		if err := c.Registrar(Decision{Fila: f.Clave(), Veredicto: Aprobar, Quien: "jefa",
			Cuando: t1}); err != nil {
			t.Fatal(err)
		}
	}
	if err := c.Registrar(Decision{Fila: filas[0].Clave(), Veredicto: Delegar, Quien: "jefa",
		Cuando: t1, A: "responsable-erp"}); err != nil {
		t.Fatal(err)
	}
	cuenta := c.Cuenta()
	if cuenta[SinRevisar] != 0 {
		t.Fatalf("el caso no aisla la delegacion: quedan %d sin tocar, asi que el cierre fallaria "+
			"por eso y esta afirmacion no probaria nada", cuenta[SinRevisar])
	}
	if cuenta[Delegada] != 1 {
		t.Fatalf("el caso no tiene la delegacion que necesita: %v", cuenta)
	}
	_, err := c.Cerrar("ciso", t2)
	if err == nil {
		t.Fatal("se ha firmado una campana cuyo unico pendiente es una delegacion. Delegar " +
			"traslada la revision, no la termina: esto certifica que alguien miro lo que nadie miro")
	}
	if !errors.Is(err, ErrSinCerrar) {
		t.Fatalf("centinela: %v", err)
	}
	// Y en cuanto la persona a la que se delego decide, cierra.
	if err := c.Registrar(Decision{Fila: filas[0].Clave(), Veredicto: Aprobar,
		Quien: "responsable-erp", Cuando: t2}); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Cerrar("ciso", t2); err != nil {
		t.Fatalf("con la delegacion ya decidida tenia que cerrar: %v", err)
	}
}

// LO ILEGIBLE BLOQUEA, Y LA EXCUSA DESBLOQUEA DEJANDO RASTRO.
//
// Control positivo de LAS DOS RAMAS con dato sintetico, que es lo que M47 enseno:
// una rama que ninguna entrada recorre es una rama que no existe, y la mutacion
// la deja verde porque no hay nada que romper.
func TestUnaLineaIlegibleBloqueaElCierreYLaExcusaLoAbreDiciendolo(t *testing.T) {
	// La linea 3 no tiene identificador: es ilegible, y es un cubo.
	con := "usuario;nombre;permiso\n" +
		"u1;Ana Perez;admin\n" +
		";Sin Nombre;lector\n" +
		"u2;Luis Gil;lector\n"
	c := campana(t, con, nil)
	if len(c.Instantanea().Ilegibles) != 1 {
		t.Fatalf("el caso no trae la fila ilegible que necesita: %+v", c.Instantanea().Ilegibles)
	}
	todosRevisadosPor(t, c, "jefa")

	// RAMA 1: bloquea.
	_, err := c.Cerrar("ciso", t2)
	if err == nil {
		t.Fatal("se ha certificado completitud con una linea que nadie pudo leer")
	}
	if !strings.Contains(err.Error(), "no se pudieron leer") {
		t.Fatalf("el error no habla de las lineas ilegibles: %v", err)
	}

	// RAMA 2: la excusa la abre, y con nombre y motivo.
	if err := c.Excusar(Excusa{Desde: 3, Hasta: 3, Quien: "ciso", Motivo: "fila de prueba del IdP",
		Cuando: t2}); err != nil {
		t.Fatal(err)
	}
	cierre, err := c.Cerrar("ciso", t2)
	if err != nil {
		t.Fatalf("con la linea excusada tenia que cerrar: %v", err)
	}
	if cierre.LineasExcusadas != 1 {
		t.Errorf("el cierre no cuenta la linea excusada: %+v", cierre)
	}
	// Y SALE EN EL INFORME. Una excusa que no se ve es exactamente lo que esto
	// existe para no permitir.
	texto := c.Informar().Texto()
	if !strings.Contains(texto, "excusadas por ciso") || !strings.Contains(texto, "fila de prueba del IdP") {
		t.Errorf("la excusa no aparece en el informe:\n%s", texto)
	}
	// Y una excusa sin quien o sin motivo no se admite.
	c2 := campana(t, con, nil)
	if err := c2.Excusar(Excusa{Desde: 3, Hasta: 3, Cuando: t2}); err == nil {
		t.Error("se admite una excusa sin quien y sin por que, que es una linea que desaparece " +
			"del recuento sin que nadie responda de ella")
	}
}

// LAS DECISIONES SON HECHOS: cambiar de opinion anade, no edita, y vale la mas
// reciente. Es lo que el acta 9.3 va a consumir.
func TestCambiarDeOpinionAnadeUnHechoYNoBorraElAnterior(t *testing.T) {
	c := campana(t, censoBase, nil)
	primera := Decision{Fila: "erp|u1|admin", Veredicto: Aprobar, Quien: "jefa", Cuando: t1}
	segunda := Decision{Fila: "erp|u1|admin", Veredicto: Revocar, Quien: "ciso", Cuando: t2,
		Motivo: "se fue del equipo"}
	if err := c.Registrar(primera); err != nil {
		t.Fatal(err)
	}
	if err := c.Registrar(segunda); err != nil {
		t.Fatal(err)
	}
	if n := len(c.Decisiones()); n != 2 {
		t.Fatalf("se esperaban DOS hechos y hay %d: la primera decision se ha perdido, y con ella "+
			"quien la tomo y cuando", n)
	}
	if c.Decisiones()[0] != primera {
		t.Error("el primer hecho se ha modificado")
	}
	if e := c.EstadoDe("erp|u1|admin"); e != Revocada {
		t.Fatalf("no manda la mas reciente: %q", e)
	}
}

// Y UN EMPATE SE DECLARA, NO SE RESUELVE. Dos decisiones distintas en el mismo
// instante son dos manos, y quien firma tiene derecho a saberlo.
func TestDosDecisionesEnElMismoInstanteSeDeclaranComoEmpate(t *testing.T) {
	c := campana(t, censoBase, nil)
	if err := c.Registrar(Decision{Fila: "erp|u1|admin", Veredicto: Aprobar, Quien: "jefa",
		Cuando: t1}); err != nil {
		t.Fatal(err)
	}
	if err := c.Registrar(Decision{Fila: "erp|u1|admin", Veredicto: Revocar, Quien: "ciso",
		Cuando: t1, Motivo: "se fue"}); err != nil {
		t.Fatal(err)
	}
	_, hay, err := c.Vigente("erp|u1|admin")
	if !hay {
		t.Fatal("no hay decision vigente")
	}
	if err == nil {
		t.Fatal("el empate no se declara: se ha elegido una de las dos en silencio")
	}
	if !strings.Contains(c.Informar().Texto(), "EMPATE") {
		t.Error("el empate no llega al informe")
	}
}

// SIN REVISOR ES LA COMPROBACION DEL DIA UNO, igual que las figuras de escalado
// sin persona. Un aviso que llega cuando ya no se puede hacer nada no es un
// aviso.
func TestSinRevisorSeVeAlAbrirLaCampanaYNoAlCerrarla(t *testing.T) {
	c := campana(t, censoBase, map[string]string{"erp|u1|admin": "jefa"})
	faltan := c.SinRevisor()
	if len(faltan) != 2 {
		t.Fatalf("faltan revisores en 2 accesos y se han visto %d: %+v", len(faltan), faltan)
	}
	// Con rotulo, para que quien lo lea sepa de quien habla.
	if faltan[0].Rotulo == "" {
		t.Error("la falta no dice de quien es la cuenta")
	}
	texto := c.Informar().Texto()
	if !strings.Contains(texto, "no tienen revisor asignado") {
		t.Errorf("el informe no lo dice el dia uno:\n%s", texto)
	}
	// Y una campana con todos asignados no lo dice: si lo dijera siempre, la
	// frase dejaria de significar algo.
	todos := map[string]string{}
	for _, f := range c.Instantanea().Filas {
		todos[f.Clave()] = "jefa"
	}
	c2 := campana(t, censoBase, todos)
	if len(c2.SinRevisor()) != 0 {
		t.Fatalf("con todos asignados no puede faltar ninguno: %+v", c2.SinRevisor())
	}
	if strings.Contains(c2.Informar().Texto(), "no tienen revisor asignado") {
		t.Error("el informe avisa de revisores que no faltan")
	}
}

// LA FRASE DE LO NO REVISADO, CON SUS DOS CONTROLES.
//
// El positivo: con accesos sin revisar, la frase esta. El negativo, que importa
// igual: con todo revisado NO esta, porque una frase que sale siempre deja de
// leerse y entonces no protege de nada.
func TestElInformeNoAcusaDeLoQueSoloEsUnDatoQueFalta(t *testing.T) {
	c := campana(t, censoBase, nil)
	texto := c.Informar().Texto()
	if !strings.Contains(texto, LaFraseDeLoNoRevisado) {
		t.Fatalf("el informe presenta accesos sin revisar SIN decir que no son una acusacion.\n"+
			"  Un acceso sin revisar puede ser perfectamente correcto: lo que falta es que "+
			"alguien lo mire.\n%s", texto)
	}
	// La frase va PEGADA al dato, no al final del documento.
	i := strings.Index(texto, "accesos sin revisar")
	j := strings.Index(texto, LaFraseDeLoNoRevisado)
	if i < 0 || j < i || j-i > 200 {
		t.Errorf("la frase no va junto al dato (dato en %d, frase en %d):\n%s", i, j, texto)
	}

	todosRevisadosPor(t, c, "jefa")
	limpio := c.Informar().Texto()
	if strings.Contains(limpio, LaFraseDeLoNoRevisado) {
		t.Errorf("la frase sale con todo revisado. Una frase que sale siempre deja de leerse:\n%s", limpio)
	}
}

// LA LEY DE CONSERVACION DE LA CAMPANA: todo acceso en exactamente un cubo, y
// los cubos vacios tambien se pintan.
func TestTodoAccesoCaeEnExactamenteUnCubo(t *testing.T) {
	c := campana(t, censoBase, nil)
	if err := c.Registrar(Decision{Fila: "erp|u1|admin", Veredicto: Aprobar, Quien: "j", Cuando: t1}); err != nil {
		t.Fatal(err)
	}
	if err := c.Registrar(Decision{Fila: "erp|u1|lector", Veredicto: Revocar, Quien: "j", Cuando: t1,
		Motivo: "ya no lo usa"}); err != nil {
		t.Fatal(err)
	}
	cuenta := c.Cuenta()
	suma := 0
	for _, e := range EstadosPosibles() {
		if _, hay := cuenta[e]; !hay {
			t.Errorf("el cubo %q no aparece; un cubo que solo sale cuando tiene algo dentro es un "+
				"cubo que nadie echa de menos", e)
		}
		suma += cuenta[e]
	}
	if suma != len(c.Instantanea().Filas) {
		t.Fatalf("los cubos suman %d y hay %d accesos: %v", suma, len(c.Instantanea().Filas), cuenta)
	}
	if !c.Informar().Cuadra() {
		t.Fatal("el informe dice que no cuadra")
	}
}

// UNA CAMPANA CERRADA NO ADMITE HECHOS NUEVOS. Un hecho posterior al cierre
// convierte el informe firmado en la foto de algo que ya no es.
func TestUnaCampanaCerradaNoAdmiteHechosNuevos(t *testing.T) {
	c := campana(t, censoBase, nil)
	todosRevisadosPor(t, c, "jefa")
	if _, err := c.Cerrar("ciso", t2); err != nil {
		t.Fatal(err)
	}
	err := c.Registrar(Decision{Fila: "erp|u1|admin", Veredicto: Revocar, Quien: "otra",
		Cuando: t2.Add(time.Hour), Motivo: "tarde"})
	if !errors.Is(err, ErrCampanaCerrada) {
		t.Fatalf("centinela: %v", err)
	}
	if err := c.Excusar(Excusa{Desde: 2, Hasta: 2, Quien: "otra", Motivo: "x", Cuando: t2}); !errors.Is(err, ErrCampanaCerrada) {
		t.Fatalf("se excusa despues de firmar: %v", err)
	}
}

// EL CIERRE LLEVA DENTRO CON QUE CONTRASTARLO: sello, hash y los recuentos. Sin
// eso, "he revisado 3 de 3" es una afirmacion sin nada detras.
func TestElCierreLlevaElSelloYLosNumerosQueLoHacenContrastable(t *testing.T) {
	c := campana(t, censoBase, nil)
	todosRevisadosPor(t, c, "jefa")
	cierre, err := c.Cerrar("ciso", t2)
	if err != nil {
		t.Fatal(err)
	}
	if cierre.Sello != c.Sello() || cierre.Sello == "" {
		t.Error("el cierre no lleva el sello de la instantanea")
	}
	if cierre.HashDelFichero != c.Instantanea().Hash {
		t.Error("el cierre no lleva el hash del fichero, que es lo unico que un tercero puede " +
			"recalcular por su cuenta")
	}
	if cierre.Accesos != 3 || cierre.Decididos != 3 {
		t.Errorf("recuentos: %+v", cierre)
	}
	if len(cierre.VeredictoPorFila) != 3 {
		t.Errorf("el cierre no guarda el veredicto por acceso: %+v", cierre.VeredictoPorFila)
	}
}

// COTEJO CON OTRA FUENTE: por identificador, en las dos direcciones, y sin
// fundir nunca por el nombre.
func TestCotejarMiraLasDosDireccionesYNoFundePorElNombre(t *testing.T) {
	c := campana(t, censoBase, nil)
	otras := []Cuenta{
		{Sistema: "erp", Cuenta: "u2", Rotulo: "Luis Gil"},
		{Sistema: "erp", Cuenta: "u7", Rotulo: "Ana Perez"}, // mismo nombre, OTRO id
	}
	cot := c.Cotejar(otras)

	// LA DIRECCION QUE SE OLVIDA: una cuenta que esta en la otra fuente y NO en
	// el censo es una cuenta que la revision no esta mirando.
	if len(cot.SoloEnLaOtraFuente) != 1 || cot.SoloEnLaOtraFuente[0] != "erp|u7" {
		t.Fatalf("no se ve la cuenta que la revision se esta dejando fuera: %+v", cot)
	}
	if len(cot.SoloEnElCenso) != 1 || cot.SoloEnElCenso[0] != "erp|u1" {
		t.Fatalf("no se ve la cuenta que ya no conoce la otra fuente: %+v", cot)
	}
	// Y NO SE FUNDEN: "Ana Perez" sale como sospecha con los DOS identificadores.
	if len(cot.Sospechas) != 1 {
		t.Fatalf("sospechas: %+v", cot.Sospechas)
	}
	s := cot.Sospechas[0]
	if len(s.Claves) != 2 {
		t.Fatalf("la sospecha tiene que ensenar los dos identificadores: %+v", s)
	}
	if !strings.Contains(s.Frase(), "no lo sabe y no las funde") {
		t.Errorf("la frase de la sospecha decide por su cuenta: %q", s.Frase())
	}
	// Y la cuenta que coincide en las dos no es sospechosa de nada.
	for _, sp := range cot.Sospechas {
		if strings.Contains(strings.ToLower(sp.Rotulo), "luis") {
			t.Error("una cuenta que casa por identificador en las dos fuentes se ha marcado " +
				"como sospecha")
		}
	}
}

// ABRIR SOBRE UNA INSTANTANEA QUE NO CUADRA NO SE PUEDE. Es la unica forma de
// que la garantia del censo llegue hasta aqui: si la campana admitiera un censo
// incompleto, la ley de conservacion del fichero no serviria de nada.
func TestNoSeAbreUnaCampanaSobreUnCensoQueNoCuadra(t *testing.T) {
	ins := instantanea(t, censoBase)
	// Se estropea a mano lo que censo.Tomar no deja salir.
	ins.LineasDeDatos = 99
	_, err := Abrir("uar", ins, t0, nil)
	if !errors.Is(err, ErrCampana) {
		t.Fatalf("centinela: %v", err)
	}
	if !strings.Contains(err.Error(), "certificar de mas") {
		t.Errorf("el error no dice por que importa:\n%v", err)
	}
}

// Y EL VALOR CERO DE LA CAMPANA ESTA PROHIBIDO, en sus dos formas de la nada.
func TestAbrirRechazaLosDatosQueFaltan(t *testing.T) {
	ins := instantanea(t, censoBase)
	casos := []struct {
		nombre string
		id     string
		cuando time.Time
		ins    censo.Instantanea
		falta  string
	}{
		{"sin id", "  ", t0, ins, "id de la campana"},
		{"sin instante", "uar", time.Time{}, ins, "instante de apertura"},
		{"instantanea sin sellar (el cero)", "uar", t0, censo.Instantanea{}, "sellada"},
		{"instantanea vacia pero presente", "uar", t0, censo.Instantanea{Hash: "x", Sistema: "erp"},
			"al menos un acceso"},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			_, err := Abrir(c.id, c.ins, c.cuando, nil)
			if !errors.Is(err, ErrCampana) {
				t.Fatalf("centinela: %v", err)
			}
			if !strings.Contains(err.Error(), c.falta) {
				t.Errorf("el error no dice que falta %q:\n%v", c.falta, err)
			}
		})
	}
}

// EL P1 DE LA EXCUSA: el rango se contrasta contra el censo sellado.
//
// SALIO DE UNA REVISION ADVERSARIA SOBRE UNA CASILLA YA MARCADA, y los tres
// casos se midieron aceptados en verde antes de arreglarlos. La primera version
// validaba quien, motivo y cuando, y del rango no miraba nada: la unica guarda
// real era un `min="1" required` de la plantilla, o sea del navegador, que un
// `curl` ignora. Una validacion que solo vive en el cliente no es una
// validacion, es una sugerencia.
func TestUnaExcusaSeContrastaContraLoIlegibleDelCensoSellado(t *testing.T) {
	// Linea 3 ilegible (sin identificador); 2 y 4 legibles.
	con := "usuario;nombre;permiso\nu1;Ana Perez;admin\n;Sin Nombre;lector\nu2;Luis Gil;lector\n"

	casos := []struct {
		nombre    string
		e         Excusa
		centinela error
		dice      string
	}{
		{
			// LA DEGENERADA. Es lo que produce un `strconv.Atoi` con el error
			// descartado: un campo vacio, ausente o con letras se vuelve cero.
			nombre:    "la linea cero, que es el error tragado",
			e:         Excusa{Desde: 0, Hasta: 0},
			centinela: ErrExcusaVacia,
			dice:      "no se pudo leer y se convirtio en cero",
		},
		{
			// LA DE BARRIDO. Cumplia la letra de "quien y por que" y derrotaba
			// lo que esa letra protege: la responsabilidad POR LINEA.
			nombre:    "el rango que se lo lleva todo con un motivo",
			e:         Excusa{Desde: 1, Hasta: 999999},
			centinela: ErrExcusaFueraDelCenso,
			dice:      "se pasa del final",
		},
		{
			nombre:    "una linea que si se pudo leer",
			e:         Excusa{Desde: 2, Hasta: 2},
			centinela: ErrExcusaFueraDelCenso,
			dice:      "esconde un acceso que habria que revisar",
		},
		{
			nombre:    "un rango que mezcla la ilegible con una legible",
			e:         Excusa{Desde: 3, Hasta: 4},
			centinela: ErrExcusaFueraDelCenso,
			dice:      "Solo se excusa lo que no se pudo leer",
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			c := campana(t, con, nil)
			e := caso.e
			e.Quien, e.Motivo, e.Cuando = "ciso", "un motivo cualquiera", t2
			err := c.Excusar(e)
			if err == nil {
				t.Fatalf("aceptada: %+v", e)
			}
			if !errors.Is(err, caso.centinela) {
				t.Fatalf("centinela: %v", err)
			}
			if !strings.Contains(err.Error(), caso.dice) {
				t.Errorf("el error no dice %q:\n%v", caso.dice, err)
			}
			// Y NO ENTRA EN EL FLUJO: si entrara, el hecho viajaria al ledger
			// append-only para siempre.
			if len(c.Informar().Excusas) != 0 {
				t.Fatalf("la excusa rechazada se ha quedado dentro: %+v", c.Informar().Excusas)
			}
		})
	}

	// Y LA BUENA SIGUE PASANDO, que es el control positivo: sin el, todo esto
	// se cumpliria rechazandolo todo.
	c := campana(t, con, nil)
	buena := Excusa{Desde: 3, Hasta: 3, Quien: "ciso", Motivo: "fila de prueba del IdP", Cuando: t2}
	if err := c.Excusar(buena); err != nil {
		t.Fatalf("la excusa legitima se ha rechazado: %v", err)
	}
	if len(c.Informar().Excusas) != 1 {
		t.Fatal("la excusa buena no ha entrado")
	}
	// Repetirla no anade nada: un hecho que no cambia nada en un registro
	// append-only es ruido que alguien tendra que interpretar dentro de un ano.
	if err := c.Excusar(buena); !errors.Is(err, ErrExcusaVacia) {
		t.Fatalf("la excusa repetida no se rechaza: %v", err)
	}
}
