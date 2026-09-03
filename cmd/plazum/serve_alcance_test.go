package main

import (
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/marcosmatalab/plazum/adaptadores/usuarios/alcances"
	"github.com/marcosmatalab/plazum/superficies/pantallas"
	"github.com/marcosmatalab/plazum/superficies/serve"
)

// LA JUNTA DEL GUARDADO, PROBADA SOBRE EL PRODUCTO LEVANTADO.
//
// Es la misma leccion que costo la entrada el 03-09-2026: `superficies/pantallas`
// pasa su suite con un almacen falso, `adaptadores/usuarios/alcances` pasa la
// suya con un fichero temporal, y las dos pueden estar en verde con el cable
// entre ellas sin poner. Aqui se levanta `plazum serve`, se instala, se contesta
// una pregunta por HTTP y se mira EL FICHERO.

// rePreguntaEnLaPagina saca el id de la primera pregunta que la entrevista
// pinta. NO se escribe un id aqui: depende del corpus instalado, y escribirlo
// dejaria este test viejo el dia que entre un marco nuevo.
var rePreguntaEnLaPagina = regexp.MustCompile(`id="p-([^"]+)"`)

func primeraPregunta(t *testing.T, pagina string) string {
	t.Helper()
	m := rePreguntaEnLaPagina.FindStringSubmatch(pagina)
	if m == nil {
		t.Fatalf("la entrevista no pinta ninguna pregunta, asi que no hay nada que "+
			"guardar.\n%s", recortarPagina(pagina))
	}
	return m[1]
}

// TestContestarCerrarYVolverConservaLasRespuestas es el criterio del tramo,
// sobre el producto de verdad y mirando el disco.
func TestContestarCerrarYVolverConservaLasRespuestas(t *testing.T) {
	s := arrancarServeInstalado(t)

	codigo, _, pagina := s.pedir(t, "/alcance")
	if codigo != http.StatusOK {
		t.Fatalf("GET /alcance con sesion contesta %d", codigo)
	}
	pregunta := primeraPregunta(t, pagina)
	// Con sesion y con almacen, la entrevista tiene que ofrecer FORMULARIOS y
	// decir que guarda. Si sigue ofreciendo enlaces, no se ha cableado nada.
	if !strings.Contains(pagina, `name="accion"`) {
		t.Fatalf("la entrevista no pinta el formulario de guardado, asi que responder sigue "+
			"sin guardar nada.\n%s", recortarPagina(pagina))
	}

	// 1. CONTESTAR. Es lo que hace el navegador al pulsar «Si».
	valores := url.Values{
		serve.CampoCSRF: {tokenCSRFDe(t, pagina)},
		"accion":        {"si"},
		"pregunta":      {pregunta},
	}
	resp, err := s.cli.PostForm(s.base+"/alcance", valores)
	if err != nil {
		t.Fatalf("enviando la respuesta: %v", err)
	}
	cuerpo := leerHasta(t, resp.Body, 1<<20)
	if err := resp.Body.Close(); err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("POST /alcance contesta %d y tenia que redirigir con 303.\n%s",
			resp.StatusCode, recortarPagina(cuerpo))
	}
	destino := resp.Header.Get("Location")
	if strings.Contains(destino, "si=") || strings.Contains(destino, "no=") {
		t.Errorf("la redireccion lleva las respuestas en la direccion (%q): si viajan ahi, no "+
			"se ha guardado nada", destino)
	}

	// 2. MIRAR EL DISCO. «Se ha guardado» solo es cierto si hay un fichero.
	ruta := filepath.Join(s.dirEstado, alcances.NombreDelFichero)
	b, err := os.ReadFile(ruta) // #nosec G304 -- ruta de un temporal de este test
	if err != nil {
		t.Fatalf("no hay fichero de respuestas en %s: la respuesta no se ha guardado en "+
			"ningun sitio. %v", ruta, err)
	}
	var doc struct {
		Version  int `json:"version"`
		Alcances []struct {
			Usuario    string `json:"usuario"`
			Respuestas []struct {
				Pregunta  string `json:"pregunta"`
				Respuesta string `json:"respuesta"`
			} `json:"respuestas"`
		} `json:"alcances"`
	}
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatalf("el fichero de respuestas no es JSON: %v\n%s", err, b)
	}
	if doc.Version != alcances.VersionDelAlmacen {
		t.Errorf("el fichero dice version %d y el almacen escribe la %d", doc.Version,
			alcances.VersionDelAlmacen)
	}
	// EL EMPAREJAMIENTO CASA POR (usuario, pregunta), que es lo que identifica
	// una respuesta dentro del fichero. Ni por posicion ni por orden.
	hallada := false
	for _, al := range doc.Alcances {
		if al.Usuario != UsuarioDePrueba {
			continue
		}
		for _, r := range al.Respuestas {
			if r.Pregunta == pregunta && r.Respuesta == "si" {
				hallada = true
			}
		}
	}
	if !hallada {
		t.Fatalf("la respuesta a %q de la cuenta %q no esta en %s:\n%s",
			pregunta, UsuarioDePrueba, ruta, b)
	}

	// 3. VOLVER AL DIA SIGUIENTE. Se pide /alcance A SECAS, sin nada en la
	//    direccion, que es lo que escribe alguien que abre su marcador.
	codigo, _, vuelta := s.pedir(t, "/alcance")
	if codigo != http.StatusOK {
		t.Fatalf("volver a /alcance contesta %d", codigo)
	}
	bloque := bloqueDePregunta(t, vuelta, pregunta)
	if !strings.Contains(bloque, "respondida") {
		t.Errorf("al volver, la pregunta %q no aparece respondida. La persona ha vuelto al "+
			"dia siguiente y su trabajo no esta.\n%s", pregunta, bloque)
	}
}

// bloqueDePregunta recorta el <li> de una pregunta.
func bloqueDePregunta(t *testing.T, pagina, id string) string {
	t.Helper()
	i := strings.Index(pagina, `id="p-`+id+`"`)
	if i < 0 {
		t.Fatalf("la pagina no trae la pregunta %q", id)
	}
	j := strings.Index(pagina[i:], "</li>")
	if j < 0 {
		t.Fatalf("el bloque de la pregunta %q no cierra", id)
	}
	return pagina[i : i+j]
}

// TestGuardarElAlcanceExigeTokenCSRF: la ruta que escribe entra en la puerta de
// CSRF de `superficies/serve`, que la exige POR METODO.
//
// Se prueba sobre el servidor montado y no sobre la superficie suelta, porque la
// superficie no comprueba el token (no puede: no conoce la sesion). El unico
// sitio donde esta propiedad es cierta o falsa es el conjunto.
func TestGuardarElAlcanceExigeTokenCSRF(t *testing.T) {
	s := arrancarServeInstalado(t)
	_, _, pagina := s.pedir(t, "/alcance")
	pregunta := primeraPregunta(t, pagina)

	casos := []struct {
		nombre  string
		valores url.Values
	}{
		{"sin token", url.Values{"accion": {"si"}, "pregunta": {pregunta}}},
		{"con un token inventado", url.Values{serve.CampoCSRF: {strings.Repeat("a", 64)},
			"accion": {"si"}, "pregunta": {pregunta}}},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			resp, err := s.cli.PostForm(s.base+"/alcance", c.valores)
			if err != nil {
				t.Fatal(err)
			}
			cuerpo := leerHasta(t, resp.Body, 1<<20)
			if err := resp.Body.Close(); err != nil {
				t.Fatal(err)
			}
			if resp.StatusCode != http.StatusForbidden {
				t.Fatalf("POST /alcance %s contesta %d y tenia que ser 403. Una peticion "+
					"mutante atendida sin CSRF es la que el atacante va a buscar.\n%s",
					c.nombre, resp.StatusCode, recortarPagina(cuerpo))
			}
		})
	}
}

// Sin sesion no se guarda nada, tambien sobre el producto: sin haber entrado, la
// peticion ni siquiera llega a la superficie (el servidor redirige a la
// instalacion o a la entrada), y lo que NO puede pasar es que se atienda.
func TestGuardarSinSesionNoSeAtiende(t *testing.T) {
	s := arrancarServeInstalado(t)
	resp, err := s.crudo.PostForm(s.base+"/alcance", url.Values{
		"accion": {"limpiar"}})
	if err != nil {
		t.Fatal(err)
	}
	if err := resp.Body.Close(); err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		t.Fatalf("POST /alcance sin cookie contesta %d: se estaria escribiendo el alcance de "+
			"una organizacion en nombre de nadie", resp.StatusCode)
	}
}

// LAS DOS CADENAS QUE TIENEN QUE COINCIDIR.
//
// `superficies/pantallas` declara su CampoCSRF sin importar `superficies/serve`,
// porque es un http.Handler autonomo. El precio es que las dos tienen que
// escribir el mismo nombre de campo, y si se separan NADA se pone rojo en sus
// suites: cada paquete pasa la suya con su propia constante, y lo que falla es
// el producto montado, donde todos los botones de la entrevista contestan 403.
//
// Este es el unico paquete que importa los dos.
func TestLasDosSuperficiesEscribenElMismoCampoCSRF(t *testing.T) {
	if !ElMismoCampoCSRF {
		t.Fatalf("pantallas.CampoCSRF es %q y serve.CampoCSRF es %q.\n"+
			"  El formulario mandaria un campo que el middleware no busca, asi que TODOS los "+
			"botones de la entrevista contestarian 403 y ninguna suite de paquete se pondria "+
			"roja.", pantallas.CampoCSRF, serve.CampoCSRF)
	}
}
