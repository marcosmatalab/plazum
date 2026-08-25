// Comando servidorprueba: el binario que arranca la puerta de CI de seguridad
// web (.github/workflows/etapa2-seguridad-web.yml).
//
// Existe porque un test que comprueba que una funcion devuelve la cabecera
// correcta no prueba que la cabecera llegue al navegador. Entre esa funcion y
// el navegador hay una cadena de middleware, un http.Server, un handler que
// escribe antes de tiempo y un proxy. La unica forma honesta de comprobarlo es
// arrancar el binario y hacer peticiones de verdad, y para eso hace falta un
// binario que arrancar.
//
// NO es el producto. Vive bajo internal/ para que nada fuera de superficies/
// pueda importarlo, y su almacen de usuarios es un mapa en memoria: el almacen
// de verdad llega con su adaptador. Lo que si es de verdad, y es todo el
// asunto, es el servidor: se construye con serve.Nuevo exactamente igual que
// lo hara el binario plazum.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"plazum/adaptadores/secretos"
	"plazum/superficies/serve"
)

func main() {
	direccion := flag.String("direccion", "127.0.0.1:8099", "donde escuchar")
	limiteAuth := flag.Int("limite-auth", 5, "intentos de autenticacion por ventana")
	ventanaAuth := flag.Duration("ventana-auth", time.Minute, "ventana del rate limit de auth")
	hosts := flag.String("hosts", "127.0.0.1,localhost", "hosts permitidos, separados por comas")
	flag.Parse()

	usuarios := &almacen{claves: map[string]string{}}

	app := serve.NuevoEnrutador()
	app.Manejar(http.MethodGet, "/app/hoy", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sujeto, hay := serve.SujetoDe(r)
		if !hay || serve.EsAnonimo(sujeto) {
			http.Redirect(w, r, "/entrar", http.StatusSeeOther)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprintln(w, "hoy")
	}))
	app.Manejar(http.MethodPost, "/app/guardar", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprintln(w, "guardado")
	}))

	// El almacen de sesiones se construye aqui, en vez de dejarselo al
	// servidor, porque una pantalla que enseña un formulario necesita emitir el
	// token CSRF de la sesion que lo va a enviar. Es exactamente lo que hara el
	// frente de pantallas.
	ses, err := serve.NuevaSesion(serve.OpcionesSesion{Secretos: secretos.Nuevo()})
	if err != nil {
		fmt.Fprintln(os.Stderr, "no se puede construir el almacen de sesiones:", err)
		os.Exit(1)
	}
	app.Manejar(http.MethodGet, "/app/formulario", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(serve.CookieSesion)
		if err != nil {
			http.Redirect(w, r, "/entrar", http.StatusSeeOther)
			return
		}
		tok, err := ses.TokenCSRF(r.Context(), c.Value)
		if err != nil {
			http.Redirect(w, r, "/entrar", http.StatusSeeOther)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		// Con html/template y no con Fprintf, aunque el token lo emitamos
		// nosotros y sea hexadecimal: en una pagina que existe para probar el
		// CSRF, escribir el ejemplo malo seria pedir que alguien lo copie.
		if err := plantillaFormulario.Execute(w, datos{Campo: serve.CampoCSRF, Token: tok}); err != nil {
			return
		}
	}))

	s, err := serve.Nuevo(serve.Config{
		App:             app,
		Sesion:          ses,
		Autenticar:      usuarios.autenticar,
		HayAdmin:        usuarios.hay,
		CrearAdmin:      usuarios.crear,
		HostsPermitidos: separar(*hosts),
		LimiteAuth:      serve.Limite{Maximo: *limiteAuth, Ventana: *ventanaAuth},
		LimiteGeneral:   serve.Limite{Maximo: 10000, Ventana: time.Minute},
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "no se puede construir el servidor:", err)
		os.Exit(1)
	}

	ctx, parar := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer parar()

	if err := s.Arrancar(ctx, *direccion); err != nil && !errors.Is(err, http.ErrServerClosed) {
		fmt.Fprintln(os.Stderr, "el servidor ha terminado con error:", err)
		os.Exit(1)
	}
}

type datos struct{ Campo, Token string }

var plantillaFormulario = template.Must(template.New("f").Parse(
	`<form method="post" action="/app/guardar">` +
		`<input type="hidden" name="{{.Campo}}" value="{{.Token}}"></form>`))

func separar(s string) []string {
	var out []string
	for _, x := range strings.Split(s, ",") {
		if x = strings.TrimSpace(x); x != "" {
			out = append(out, x)
		}
	}
	return out
}

// almacen es el doble del almacen de usuarios, que llega con su adaptador.
type almacen struct {
	mu     sync.Mutex
	claves map[string]string
}

func (a *almacen) hay(context.Context) (bool, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return len(a.claves) > 0, nil
}

func (a *almacen) crear(_ context.Context, usuario, secreto string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if usuario == "" {
		return errors.New("el usuario no puede ir vacio")
	}
	a.claves[usuario] = secreto
	return nil
}

func (a *almacen) autenticar(_ context.Context, usuario, secreto string) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if s, ok := a.claves[usuario]; ok && s == secreto && secreto != "" {
		return usuario, nil
	}
	return "", errors.New("credenciales incorrectas")
}
