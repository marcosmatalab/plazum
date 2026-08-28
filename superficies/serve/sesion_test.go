package serve_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/marcosmatalab/plazum/adaptadores/secretos"
	"github.com/marcosmatalab/plazum/puertos"
	"github.com/marcosmatalab/plazum/puertos/contrato"
	"github.com/marcosmatalab/plazum/superficies/serve"
)

func nuevaSesion(t *testing.T, o serve.OpcionesSesion) *serve.Sesion {
	t.Helper()
	if o.Secretos == nil {
		o.Secretos = secretos.Nuevo()
	}
	s, err := serve.NuevaSesion(o)
	if err != nil {
		t.Fatalf("construir el almacen de sesiones: %v", err)
	}
	return s
}

// La puerta: el contrato del puerto, tal cual, sin adaptar nada.
func TestLaSesionCumpleElContrato(t *testing.T) {
	contrato.Sesion(t, func() puertos.Sesion {
		return nuevaSesion(t, serve.OpcionesSesion{})
	})
}

// --- lo que el contrato no cubre y el atacante si intenta ---

// El reloj que miente: una sesion caducada no puede seguir leyendose porque el
// proceso no haya mirado el reloj todavia.
func TestUnaSesionCaducadaDejaDeValerAunqueNadieLaHayaTocado(t *testing.T) {
	reloj := &relojFalso{t: time.Date(2026, 8, 25, 9, 0, 0, 0, time.UTC)}
	s := nuevaSesion(t, serve.OpcionesSesion{Reloj: reloj.Ahora})
	ctx := context.Background()

	id, err := s.Abrir(ctx, "ciso", 30*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	tok, err := s.TokenCSRF(ctx, id)
	if err != nil {
		t.Fatal(err)
	}

	reloj.Avanzar(29 * time.Minute)
	if _, err := s.Leer(ctx, id); err != nil {
		t.Fatalf("a los 29 minutos de una sesion de 30 todavia tiene que valer: %v", err)
	}

	reloj.Avanzar(2 * time.Minute) // 31 minutos: pasada
	if _, err := s.Leer(ctx, id); err == nil {
		t.Fatal("una sesion de 30 minutos sigue leyendose a los 31")
	}
	if err := s.ComprobarCSRF(ctx, id, tok); err == nil {
		t.Fatal("el token CSRF de una sesion caducada sigue validando. Una pestana " +
			"olvidada abierta toda la noche seguiria pudiendo mutar estado")
	}
	if _, err := s.TokenCSRF(ctx, id); err == nil {
		t.Fatal("una sesion caducada sigue emitiendo tokens nuevos: eso la resucita")
	}
}

// El reloj que retrocede (NTP mal puesto, maquina virtual restaurada). Una
// sesion caducada NO puede revivir porque el reloj vuelva atras.
func TestUnaSesionCaducadaNoRevivePorqueElRelojRetroceda(t *testing.T) {
	reloj := &relojFalso{t: time.Date(2026, 8, 25, 9, 0, 0, 0, time.UTC)}
	s := nuevaSesion(t, serve.OpcionesSesion{Reloj: reloj.Ahora})
	ctx := context.Background()

	id, err := s.Abrir(ctx, "ciso", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	reloj.Avanzar(2 * time.Minute)
	if _, err := s.Leer(ctx, id); err == nil {
		t.Fatal("la sesion tenia que haber caducado")
	}
	reloj.Avanzar(-10 * time.Minute) // el reloj vuelve atras
	if _, err := s.Leer(ctx, id); err == nil {
		t.Fatal("la sesion ha revivido al retroceder el reloj. Una sesion que caduco " +
			"tiene que quedarse caducada pase lo que pase con el reloj")
	}
}

// El identificador de sesion no se acepta del cliente en ningun sitio: no hay
// forma de plantar uno. Es la fijacion de sesion, cerrada por construccion.
func TestNoSePuedeImponerUnIdentificadorDeSesion(t *testing.T) {
	s := nuevaSesion(t, serve.OpcionesSesion{})
	ctx := context.Background()
	plantado := "el-que-yo-elijo"
	if _, err := s.Leer(ctx, plantado); err == nil {
		t.Fatal("un identificador inventado abre sesion")
	}
	id, err := s.Abrir(ctx, "ciso", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if id == plantado {
		t.Fatal("el almacen ha devuelto el identificador que el atacante queria")
	}
	if len(id) != 64 {
		t.Fatalf("el identificador tiene %d caracteres; se esperan 64 (32 bytes en hex)", len(id))
	}
}

// Cerrar una sesion no puede tocar a otra, ni siquiera si comparten prefijo.
func TestCerrarUnaSesionNoTocaLasDemas(t *testing.T) {
	s := nuevaSesion(t, serve.OpcionesSesion{})
	ctx := context.Background()
	a, err := s.Abrir(ctx, "uno", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	b, err := s.Abrir(ctx, "dos", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Cerrar(ctx, a); err != nil {
		t.Fatal(err)
	}
	if suj, err := s.Leer(ctx, b); err != nil || suj != "dos" {
		t.Fatalf("cerrar la sesion de uno ha afectado a la de dos: %q, %v", suj, err)
	}
}

// El tope de tokens por sesion existe para que pedir formularios en bucle no
// haga crecer el proceso sin limite.
func TestLosTokensPorSesionTienenTope(t *testing.T) {
	s := nuevaSesion(t, serve.OpcionesSesion{MaxTokensPorSesion: 4})
	ctx := context.Background()
	id, err := s.Abrir(ctx, "ciso", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	var emitidos []string
	for i := 0; i < 10; i++ {
		tok, err := s.TokenCSRF(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		emitidos = append(emitidos, tok)
	}
	// Los cuatro ultimos valen.
	for _, tok := range emitidos[len(emitidos)-4:] {
		if err := s.ComprobarCSRF(ctx, id, tok); err != nil {
			t.Fatalf("uno de los 4 tokens mas recientes no vale: %v", err)
		}
	}
	// El primero, no: se recorto.
	if err := s.ComprobarCSRF(ctx, id, emitidos[0]); err == nil {
		t.Fatal("con tope de 4 tokens, el primero de 10 sigue valiendo: no hay tope")
	}
}

// El tope de sesiones vivas: sin el, quien pueda autenticarse (o el propio
// flujo anonimo del formulario de entrada) hace crecer el proceso hasta que el
// planificador de obligaciones se queda sin memoria.
func TestElNumeroDeSesionesVivasTieneTopeYElErrorDiceComoSubirlo(t *testing.T) {
	s := nuevaSesion(t, serve.OpcionesSesion{MaxSesiones: 3})
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		if _, err := s.Abrir(ctx, "ciso", time.Hour); err != nil {
			t.Fatalf("la sesion %d tenia que caber: %v", i, err)
		}
	}
	_, err := s.Abrir(ctx, "ciso", time.Hour)
	if err == nil {
		t.Fatal("con tope de 3 se ha abierto la cuarta")
	}
	if !strings.Contains(err.Error(), "MaxSesiones") {
		t.Errorf("el error no dice como se sube el tope: %v", err)
	}
}

// Las sesiones caducadas se recogen solas: si no, el tope de arriba convierte
// una instalacion vieja en una que no deja entrar a nadie.
func TestLasSesionesCaducadasSeRecogenYLiberanSitio(t *testing.T) {
	reloj := &relojFalso{t: time.Date(2026, 8, 25, 9, 0, 0, 0, time.UTC)}
	s := nuevaSesion(t, serve.OpcionesSesion{MaxSesiones: 2, Reloj: reloj.Ahora})
	ctx := context.Background()
	for i := 0; i < 2; i++ {
		if _, err := s.Abrir(ctx, "ciso", time.Minute); err != nil {
			t.Fatal(err)
		}
	}
	reloj.Avanzar(2 * time.Minute)
	if _, err := s.Abrir(ctx, "ciso", time.Minute); err != nil {
		t.Fatalf("las dos sesiones anteriores estan caducadas y aun asi no cabe una nueva: %v", err)
	}
	if v := s.Vivas(); v != 1 {
		t.Fatalf("quedan %d sesiones vivas y solo tenia que quedar la recien abierta", v)
	}
}

// Una duracion absurda se rechaza: ni cero ni negativa ni un ano.
func TestUnaDuracionAbsurdaSeRechazaConMotivo(t *testing.T) {
	s := nuevaSesion(t, serve.OpcionesSesion{})
	ctx := context.Background()
	for _, d := range []time.Duration{0, -time.Hour, 400 * 24 * time.Hour} {
		if _, err := s.Abrir(ctx, "ciso", d); err == nil {
			t.Errorf("Abrir con duracion %v no ha fallado", d)
		}
	}
}

// Si no hay aleatoriedad, no se abre sesion: mejor no arrancar que emitir
// identificadores adivinables.
func TestSinAleatoriedadNoSeAbreSesion(t *testing.T) {
	s := nuevaSesion(t, serve.OpcionesSesion{Secretos: secretosRotos{}})
	ctx := context.Background()
	if _, err := s.Abrir(ctx, "ciso", time.Hour); err == nil {
		t.Fatal("se ha abierto sesion sin fuente de aleatoriedad")
	}
}

// NuevaSesion sin fuente de secretos no construye: el fallback silencioso a un
// generador cualquiera es exactamente el fallo que el puerto Secretos evita.
func TestNuevaSesionSinSecretosNoConstruye(t *testing.T) {
	if _, err := serve.NuevaSesion(serve.OpcionesSesion{}); err == nil {
		t.Fatal("NuevaSesion ha construido un almacen sin fuente de aleatoriedad")
	}
}

// El almacen lo tocan todas las peticiones a la vez desde el primer dia.
func TestElAlmacenDeSesionesAguantaConcurrencia(t *testing.T) {
	s := nuevaSesion(t, serve.OpcionesSesion{})
	ctx := context.Background()
	var wg sync.WaitGroup
	for i := 0; i < 24; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			id, err := s.Abrir(ctx, "ciso", time.Hour)
			if err != nil {
				t.Error(err)
				return
			}
			for j := 0; j < 8; j++ {
				tok, err := s.TokenCSRF(ctx, id)
				if err != nil {
					t.Error(err)
					return
				}
				if err := s.ComprobarCSRF(ctx, id, tok); err != nil {
					t.Error(err)
					return
				}
				if _, err := s.Leer(ctx, id); err != nil {
					t.Error(err)
					return
				}
			}
			if err := s.Cerrar(ctx, id); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
}

// Un contexto ya cancelado no ejecuta la operacion: si el cliente colgo, no se
// abre una sesion que nadie va a cerrar.
func TestUnContextoCanceladoNoAbreSesion(t *testing.T) {
	s := nuevaSesion(t, serve.OpcionesSesion{})
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	if _, err := s.Abrir(ctx, "ciso", time.Hour); !errors.Is(err, context.Canceled) {
		t.Fatalf("se esperaba context.Canceled y se obtuvo: %v", err)
	}
}

// --- dobles ---

type relojFalso struct {
	mu sync.Mutex
	t  time.Time
}

func (r *relojFalso) Ahora() time.Time {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.t
}

func (r *relojFalso) Avanzar(d time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.t = r.t.Add(d)
}

// secretosRotos simula la maquina sin /dev/urandom.
type secretosRotos struct{}

func (secretosRotos) Token(int) (string, error) { return "", errors.New("sin aleatoriedad") }
func (secretosRotos) Bytes([]byte) error        { return errors.New("sin aleatoriedad") }
