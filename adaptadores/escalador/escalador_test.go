package escalador

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/marcosmatalab/plazum/nucleo/corpus"
	"github.com/marcosmatalab/plazum/nucleo/escalado"
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

// canalFalso cuenta lo que se le manda y puede fallar a voluntad.
type canalFalso struct {
	enviados []string
	err      error
	// alEnviar se llama antes de devolver, para simular que el proceso muere
	// justo despues del envio y antes de anotar el curso.
	alEnviar func()
}

func (c *canalFalso) Enviar(canal, destinatario, asunto, cuerpo string) error {
	c.enviados = append(c.enviados, canal+" -> "+destinatario+": "+asunto)
	if c.alEnviar != nil {
		c.alEnviar()
	}
	return c.err
}

func (c *canalFalso) UltimoExito(string) (string, error) { return "", nil }

func trabajo(escalones ...corpus.Escalon) Trabajo {
	return Trabajo{
		Obligacion: corpus.Obligacion{ID: "prueba.obligacion", Titulo: "Revisar lo que toque",
			ClaseE2E: "procedimental", Escalado: escalones},
		Hito:  "ocurrencia#1",
		Vence: time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC),
		Regimen: ventana.Regimen{
			Cal:  ventana.NuevoCalendario("utc-v1", "escalador", "prueba", time.UTC),
			Comp: ventana.Naturales, Cierre: ventana.CierreExacto,
		},
	}
}

func lazo(t *testing.T, c *canalFalso, ruta string) *Lazo {
	t.Helper()
	d, err := AbrirDiario(ruta)
	if err != nil {
		t.Fatal(err)
	}
	return &Lazo{
		Diario: d, Canal: c,
		Figuras: escalado.Asignacion{"f.seguridad": "ana", "f.direccion": "beatriz"},
		Enlace:  func(o, h string) string { return "http://localhost:8080/o/" + o + "#" + h },
		Destino: func(p string) (string, string) { return "email", p + "@ejemplo.com" },
	}
}

// ---------------------------------------------------------------------------
// La ley de conservacion de la vuelta
// ---------------------------------------------------------------------------

// TODO AVISO PLANIFICADO DE CADA VUELTA TERMINA EN EXACTAMENTE UN CUBO. Sin
// esto, un aviso que el lazo se saltara por cualquier motivo se leeria igual que
// uno que no existia.
func TestTodoAvisoDeLaVueltaCaeEnExactamenteUnCubo(t *testing.T) {
	c := &canalFalso{}
	l := lazo(t, c, filepath.Join(t.TempDir(), "diario.jsonl"))
	r, err := l.Vuelta(ins(t, "2026-05-01T09:00:00Z"), []Trabajo{trabajo(
		corpus.Escalon{Tras: "P60D_antes", A: "f.seguridad"}, // sale
		corpus.Escalon{Tras: "P30D_antes", A: "f.direccion"}, // sale
		corpus.Escalon{Tras: "P10D_antes", A: "f.nadie"},     // sin destinatario
	)})
	if err != nil {
		t.Fatal(err)
	}
	if r.Planificados != 3 {
		t.Fatalf("se planificaron %d avisos y habia 3 escalones", r.Planificados)
	}
	if r.Suma() != r.Planificados {
		t.Fatalf("los cubos suman %d y se planificaron %d: hay avisos que no estan en "+
			"ningun sitio", r.Suma(), r.Planificados)
	}
	if len(c.enviados) != 2 {
		t.Fatalf("se mandaron %d avisos y tenian que salir 2: %v", len(c.enviados), c.enviados)
	}
}

// ---------------------------------------------------------------------------
// La clave de deduplicacion: un reinicio no re-manda la tormenta
// ---------------------------------------------------------------------------

// Una segunda vuelta sobre el mismo ciclo no vuelve a mandar nada. Es LO QUE
// IMPIDE que un reinicio saque los 53 avisos del corpus otra vez, que seria el
// fin de la confianza del operador en el remitente.
func TestUnaSegundaVueltaSobreElMismoCicloNoRepiteNingunAviso(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), "diario.jsonl")
	tr := []Trabajo{trabajo(
		corpus.Escalon{Tras: "P60D_antes", A: "f.seguridad"},
		corpus.Escalon{Tras: "P30D_antes", A: "f.direccion"},
	)}

	c1 := &canalFalso{}
	if _, err := lazo(t, c1, ruta).Vuelta(ins(t, "2026-05-01T09:00:00Z"), tr); err != nil {
		t.Fatal(err)
	}
	if len(c1.enviados) != 2 {
		t.Fatalf("la primera vuelta mando %d", len(c1.enviados))
	}

	// Segunda vuelta, con el diario que dejo la primera: proceso nuevo.
	c2 := &canalFalso{}
	r, err := lazo(t, c2, ruta).Vuelta(ins(t, "2026-05-02T09:00:00Z"), tr)
	if err != nil {
		t.Fatal(err)
	}
	if len(c2.enviados) != 0 {
		t.Fatalf("la segunda vuelta re-mando %d avisos: %v", len(c2.enviados), c2.enviados)
	}
	if r.YaCerrados != 2 {
		t.Fatalf("la segunda vuelta dice que ya estaban cerrados %d de 2", r.YaCerrados)
	}
	if r.Suma() != r.Planificados {
		t.Fatal("la ley de conservacion no cuadra en la segunda vuelta")
	}
}

// EL CICLO SIGUIENTE SI SE AVISA. Si la clave no llevara el vencimiento dentro,
// la revision del ano que viene se daria por avisada con el correo del ano
// pasado, que es el fallo contrario y peor.
func TestElCicloSIGUIENTEDelMismoRelojSiSeAvisa(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), "diario.jsonl")
	esc := corpus.Escalon{Tras: "P30D_antes", A: "f.seguridad"}

	c1 := &canalFalso{}
	if _, err := lazo(t, c1, ruta).Vuelta(ins(t, "2026-05-01T09:00:00Z"),
		[]Trabajo{trabajo(esc)}); err != nil {
		t.Fatal(err)
	}
	// El mismo reloj y EL MISMO HITO, con el vencimiento movido: lo unico que
	// cambia es el ciclo.
	//
	// La primera version de este caso movia tambien el hito ("ocurrencia#1" a
	// "ocurrencia#2") y por eso NO PROBABA LO QUE DICE: quitarle el vencimiento
	// a la clave lo dejaba verde, porque el hito ya la hacia distinta. Lo cazo
	// la mutacion M70, y es la misma familia que la rama que nunca se ejecuta:
	// un caso que varia dos cosas a la vez no aisla ninguna.
	//
	// Y el caso es real, no de laboratorio: un plazo que arranca de un hecho
	// tiene SIEMPRE el mismo hito y un vencimiento distinto por cada incidente.
	siguiente := trabajo(esc)
	siguiente.Vence = siguiente.Vence.AddDate(1, 0, 0)

	c2 := &canalFalso{}
	if _, err := lazo(t, c2, ruta).Vuelta(ins(t, "2027-05-01T09:00:00Z"),
		[]Trabajo{siguiente}); err != nil {
		t.Fatal(err)
	}
	if len(c2.enviados) != 1 {
		t.Fatalf("el ciclo siguiente mando %d avisos y tenia que mandar 1", len(c2.enviados))
	}
}

// ---------------------------------------------------------------------------
// El registro antes que el envio, y la muerte en medio
// ---------------------------------------------------------------------------

// EL CASO QUE JUSTIFICA EL ORDEN. El proceso muere DESPUES de mandar y ANTES de
// anotar el curso. La vuelta siguiente ve una intencion sin curso — que es un
// DATO, no un hueco — y reenvia: entrega al-menos-una-vez, con el precio contado
// en Reintentos.
//
// Al reves (enviar y luego anotar) no habria intencion en el diario, y ese aviso
// seria indistinguible de uno que nunca se planifico.
func TestSiElLazoMuereEntreElEnvioYElRegistroLaVueltaSiguienteReenvia(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), "diario.jsonl")
	tr := []Trabajo{trabajo(corpus.Escalon{Tras: "P30D_antes", A: "f.seguridad"})}

	// Primera vuelta: el canal manda y el proceso "muere" ahi mismo.
	murio := errors.New("el proceso se muere justo aqui")
	c1 := &canalFalso{}
	l1 := lazo(t, c1, ruta)
	c1.alEnviar = func() { l1.Diario.ruta = string(rune(0)) } // el diario deja de poder escribir
	_, err := l1.Vuelta(ins(t, "2026-05-01T09:00:00Z"), tr)
	if err == nil {
		t.Fatalf("la vuelta no fallo al no poder anotar el curso (%v)", murio)
	}
	if len(c1.enviados) != 1 {
		t.Fatalf("el aviso no llego a salir, asi que este caso no prueba lo que dice")
	}

	// El diario que quedo en disco: la intencion esta, el curso no.
	b, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"tipo":"intencion"`) {
		t.Fatalf("no quedo la intencion en el diario:\n%s", b)
	}
	if strings.Contains(string(b), `"tipo":"curso"`) {
		t.Fatalf("quedo un curso que no se llego a escribir:\n%s", b)
	}

	// Segunda vuelta, proceso nuevo: reenvia y lo cuenta.
	c2 := &canalFalso{}
	r, err := lazo(t, c2, ruta).Vuelta(ins(t, "2026-05-02T09:00:00Z"), tr)
	if err != nil {
		t.Fatal(err)
	}
	if len(c2.enviados) != 1 {
		t.Fatalf("tras el corte no se reenvio (%d envios): con al-menos-una-vez, un aviso "+
			"que pudo perderse se manda otra vez", len(c2.enviados))
	}
	if r.Reintentos != 1 {
		t.Fatalf("el reenvio no se cuenta como reintento (%d): el precio de la entrega "+
			"al-menos-una-vez que no se mide no se puede juzgar", r.Reintentos)
	}
}

// Un diario truncado por un corte NO SE SALTA. Saltar la linea rota convertiria
// un corte en "esa intencion nunca existio", y el aviso en vuelo se perderia o
// se daria por hecho.
func TestUnDiarioTruncadoNoSeLeeAMedias(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), "diario.jsonl")
	contenido := `{"clave":"a","tipo":"curso","instante":"2026-05-01T09:00:00Z","estado":"enviado al canal"}
{"clave":"b","tipo":"intenc`
	if err := os.WriteFile(ruta, []byte(contenido), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := AbrirDiario(ruta); !errors.Is(err, ErrDiarioRoto) {
		t.Fatalf("un diario truncado se lee como bueno: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Lo que no sale por decision deja constancia de la decision
// ---------------------------------------------------------------------------

func TestUnAvisoSuprimidoDejaSuMotivoEnElDiario(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), "diario.jsonl")
	c := &canalFalso{}
	l := lazo(t, c, ruta)
	l.Silencios = []escalado.Silencio{{
		Desde: ins(t, "2026-04-01T00:00:00Z"), Hasta: ins(t, "2026-07-01T00:00:00Z"),
		Motivo: "parada de mantenimiento", Quien: "ana",
	}}
	if _, err := l.Vuelta(ins(t, "2026-05-01T09:00:00Z"), []Trabajo{
		trabajo(corpus.Escalon{Tras: "P30D_antes", A: "f.seguridad"})}); err != nil {
		t.Fatal(err)
	}
	if len(c.enviados) != 0 {
		t.Fatal("se mando un aviso que caia dentro de una ventana de silencio")
	}
	b, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatal(err)
	}
	// "No se aviso" y "se decidio no avisar" tienen que leerse distinto.
	for _, quiero := range []string{`"tipo":"suprimido"`, "ventana de silencio",
		"declaro ana", "Caduca sola"} {
		if !strings.Contains(string(b), quiero) {
			t.Errorf("el diario no dice %q:\n%s", quiero, b)
		}
	}
}

// ---------------------------------------------------------------------------
// Un fallo de entrega se ENSENA: no cierra el ciclo y se puede listar
// ---------------------------------------------------------------------------

func TestUnFalloDeEntregaNiCierraElCicloNiSeQuedaCallado(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), "diario.jsonl")
	tr := []Trabajo{trabajo(corpus.Escalon{Tras: "P30D_antes", A: "f.seguridad"})}

	roto := &canalFalso{err: errors.New("el servidor contesto 503")}
	l1 := lazo(t, roto, ruta)
	r, err := l1.Vuelta(ins(t, "2026-05-01T09:00:00Z"), tr)
	if err != nil {
		t.Fatal(err)
	}
	if r.Cuenta[escalado.Fallido] != 1 {
		t.Fatalf("el fallo de entrega no cae en su cubo: %v", r.Cuenta)
	}
	// Se puede LISTAR, que es lo que permite ensenarlo donde el operador mira.
	// Un fallo que solo se anota es un escalado que muere en silencio.
	if fs := l1.Diario.Fallidos(); len(fs) != 1 {
		t.Fatalf("el diario no sabe decir que quedo fallido: %v", fs)
	}

	// Y NO cierra el ciclo: la vuelta siguiente lo reintenta, que es media
	// razon de que exista un lazo.
	bueno := &canalFalso{}
	if _, err := lazo(t, bueno, ruta).Vuelta(ins(t, "2026-05-02T09:00:00Z"), tr); err != nil {
		t.Fatal(err)
	}
	if len(bueno.enviados) != 1 {
		t.Fatalf("un aviso fallido no se reintenta en la vuelta siguiente (%d envios)",
			len(bueno.enviados))
	}
}

// ---------------------------------------------------------------------------
// El valor cero del lazo no manda nada
// ---------------------------------------------------------------------------

func TestUnLazoSinDiarioOSinCanalNoManda(t *testing.T) {
	c := &canalFalso{}
	d, err := AbrirDiario(filepath.Join(t.TempDir(), "d.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	for nombre, l := range map[string]*Lazo{
		"sin diario": {Canal: c},
		"sin canal":  {Diario: d},
		"vacio":      {},
	} {
		t.Run(nombre, func(t *testing.T) {
			if _, err := l.Vuelta(time.Now(), []Trabajo{trabajo()}); err == nil {
				t.Fatal("una vuelta sin lo que hace falta no da error")
			}
			if len(c.enviados) != 0 {
				t.Fatal("ha mandado algo")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// La vida del lazo alimenta el latido
// ---------------------------------------------------------------------------

// La marca del ciclo se escribe AUNQUE ALGUN ENVIO HAYA FALLADO. Es la misma
// decision que toma el latido con su pulso, y por el mismo motivo: si un canal
// roto borrara la marca, el operador leeria "el planificador esta muerto" cuando
// lo que pasa es que el correo no sale. Son dos problemas con dos arreglos.
func TestLaVueltaApuntaElCicloAunqueElCanalEsteRoto(t *testing.T) {
	for nombre, canal := range map[string]*canalFalso{
		"con el canal bien": {},
		"con el canal roto": {err: errors.New("503")},
	} {
		t.Run(nombre, func(t *testing.T) {
			l := lazo(t, canal, filepath.Join(t.TempDir(), "d.jsonl"))
			var marcado time.Time
			l.MarcarCiclo = func(ahora time.Time) error { marcado = ahora; return nil }
			ahora := ins(t, "2026-05-01T09:00:00Z")
			if _, err := l.Vuelta(ahora, []Trabajo{
				trabajo(corpus.Escalon{Tras: "P30D_antes", A: "f.seguridad"})}); err != nil {
				t.Fatal(err)
			}
			if !marcado.Equal(ahora) {
				t.Fatalf("la vuelta no apunto el ciclo (%v)", marcado)
			}
		})
	}
}

// Y si no se puede apuntar, se dice: un planificador vivo que el latido cree
// muerto acaba en un aviso que el operador aprende a ignorar.
func TestSiNoSePuedeApuntarElCicloLaVueltaLoDice(t *testing.T) {
	l := lazo(t, &canalFalso{}, filepath.Join(t.TempDir(), "d.jsonl"))
	l.MarcarCiclo = func(time.Time) error { return errors.New("disco lleno") }
	_, err := l.Vuelta(ins(t, "2026-05-01T09:00:00Z"), []Trabajo{
		trabajo(corpus.Escalon{Tras: "P30D_antes", A: "f.seguridad"})})
	if err == nil || !strings.Contains(err.Error(), "el planificador esta parado sin estarlo") {
		t.Fatalf("no se dice lo que significa no poder apuntar el ciclo: %v", err)
	}
}
