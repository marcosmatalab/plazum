package plazum

import (
	"net/http"
	"os/exec"
	"path/filepath"
	"testing"
)

// PantallasQueExigenSesion son las rutas que el producto publica como «las seis
// pantallas», mas la raiz.
//
// La lista va escrita aqui y NO derivada del enrutador, a proposito y por el
// motivo de siempre: una lista derivada del codigo que vigila no puede acusarlo.
// El dia que alguien saque una de estas rutas del middleware de sesion, una
// lista derivada la sacaria tambien y esta puerta seguiria verde.
var PantallasQueExigenSesion = []string{
	"/", "/hoy", "/calendario", "/acta", "/camino", "/alcance", "/controles",
}

// NINGUNA DE LAS SEIS PANTALLAS CONTESTA A QUIEN NO HA ENTRADO.
//
// # El hallazgo que la trae, y viene de la pasada del comprador
//
// El 22-09-2026 se cableo `plazum demo --serve`, y el texto que se escribio para
// anunciarlo decia que levantaba «las seis pantallas sobre este mismo estado, y
// no hay nada que configurar». Al EJERCERLO —arrancar el binario de verdad y
// pedir las siete rutas— contestaron todas **303**: van detras de sesion, y
// antes hay que dar de alta al primer administrador con un token de un solo uso.
//
// O sea que la frase era falsa el dia que se escribio. Es la familia de la
// afirmacion de interfaz sin atar al comportamiento, y la cazo ejercer el
// producto y no leer el diff: un arnes que solo hubiera comprobado que el
// servidor arranca habria dado verde.
//
// # Por que vive aqui y no en `superficies/serve`
//
// Porque ahi se escribio primero y estaba MIDIENDO EL ARNES. El servidor de
// pruebas de aquel paquete no monta las pantallas —las cablea `cmd/plazum`— asi
// que las siete rutas contestaban 404, y un 404 habria pasado por «no es 200» en
// una puerta escrita con menos cuidado. Es la misma familia que la medida del
// TTFV que no ejercia el sistema: si lo que se mide no es el producto, el numero
// que sale habla del banco de pruebas.
//
// # Por que se exige la redireccion y no «distinto de 200»
//
// Porque «distinto de 200» lo cumple un 500, y un servidor que revienta tambien
// deja de servir la pagina. Se exige la clase de respuesta que significa «no has
// entrado», no la ausencia de la buena.
func TestNingunaPantallaContestaSinSesion(t *testing.T) {
	if testing.Short() {
		t.Skip("SALTADO, no comprobado: arranca el binario de verdad. NO es verde")
	}
	dir := t.TempDir()
	binario := filepath.Join(dir, "plazum"+extensionDeBinario())
	if salida, err := exec.Command("go", "build", "-o", binario, "./cmd/plazum").CombinedOutput(); err != nil {
		t.Fatalf("no se puede compilar el binario, asi que esto no comprueba el producto "+
			"sino la maquina: %v\n%s", err, salida)
	}
	srv := arrancarServidor(t, binario)
	defer srv.parar()

	// NO se llama a srv.instalar: el punto entero es pedir las paginas SIN
	// haber creado el administrador, que es el estado en el que se queda quien
	// teclea `plazum demo --serve`.
	for _, ruta := range PantallasQueExigenSesion {
		t.Run(ruta, func(t *testing.T) {
			resp, err := srv.crudo.Get(srv.base + ruta)
			if err != nil {
				t.Fatalf("pidiendo %s sin sesion: %v", ruta, err)
			}
			defer func() { _ = resp.Body.Close() }()

			if resp.StatusCode == http.StatusOK {
				t.Errorf(`%s ha contestado 200 a una peticion sin sesion.

  Estas pantallas ensenan el alcance y el calendario de UNA organizacion. Servir
  cualquiera de ellas a quien no ha entrado es publicar las respuestas de una
  persona a un anonimo, que es el invariante 12 en su forma mas cara.`, ruta)
				return
			}
			if resp.StatusCode != http.StatusSeeOther && resp.StatusCode != http.StatusFound {
				t.Errorf("%s ha contestado %d sin sesion, y se esperaba una redireccion.\n"+
					"  «Distinto de 200» no basta: un 500 tambien lo cumple, y un servidor "+
					"que revienta no es lo mismo que uno que pide entrar.",
					ruta, resp.StatusCode)
			}
		})
	}

	// EL CONTROL POSITIVO, y sin el la puerta de arriba no demuestra nada: si
	// este servidor contestara lo mismo a TODO, «ninguna pantalla contesta»
	// seria cierto y vacio. `/salud` existe justo para contestar sin sesion.
	resp, err := srv.crudo.Get(srv.base + "/salud")
	if err != nil {
		t.Fatalf("pidiendo /salud sin sesion: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("/salud ha contestado %d y tendria que contestar 200 sin sesion.\n"+
			"  Sin una ruta que SI conteste, lo de arriba no demuestra que las seis esten "+
			"protegidas: demuestra que este servidor no sirve nada.", resp.StatusCode)
	}
}
