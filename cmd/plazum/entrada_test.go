package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/marcosmatalab/plazum/superficies/camino"
	"github.com/marcosmatalab/plazum/superficies/serve"
)

// EL TRINQUETE DE LA ENTRADA: nadie puede volver a dejar el producto sin puerta.
//
// # Que agujero cierra, con su medida
//
// El 03-09-2026, sobre el binario de verdad: TRES de los seis pasos del camino
// guiado (/acta/, /uar/ y /escalado/) contestaban 401 a quien acababa de
// instalar plazum, `/primer-admin` contestaba 503 con un mensaje escrito para un
// programador («este servidor se construyo sin Config.CrearAdmin [...] Arreglo
// para quien lo cablea») y `/entrar` servia un formulario que pedia credenciales
// que no podia tener nadie.
//
// La causa no era un fallo de ninguna de las dos mitades. `superficies/serve`
// tenia el mecanismo entero construido y probado, con sus tests en verde;
// `cmd/plazum/serve.go` montaba el servidor y su suite tambien estaba en verde.
// Lo que no existia era LA JUNTA: el literal `serve.Config{...}` no pasaba
// `Autenticar`, ni `HayAdmin`, ni `CrearAdmin`, y el valor cero de esos tres
// campos es «denegar y decirlo». Cada mitad pasaba su puerta.
//
// # Las dos puertas de aqui, y por que hacen falta las dos
//
//	la ESTRUCTURAL   enumera las decisiones de identidad LEYENDO serve.Config y
//	                 exige que el literal de cmd/plazum/serve.go las pase todas.
//	                 Es barata, no levanta nada y caza el olvido el mismo dia que
//	                 se escribe. Y no lleva lista: el conjunto sale del AST de
//	                 `superficies/serve`, asi que un cuarto gancho de identidad
//	                 que se anada manana entra solo.
//	la CONDUCTUAL    levanta el producto sobre un directorio VACIO, lee el token
//	                 de instalacion de la salida de arranque, crea el
//	                 administrador por HTTP como haria el navegador y recorre los
//	                 seis pasos. Es cara y es la unica que demuestra que se llega.
//
// Ninguna sola habria cazado el fallo entero: la estructural no sabe si las
// funciones que se pasan sirven de algo, y la conductual no puede decir cual de
// los tres ganchos falta cuando falla.

// --- la puerta estructural ---

// decisionesDeIdentidadDeServeConfig enumera los campos de serve.Config que son
// decisiones de identidad, LEYENDO el AST de superficies/serve.
//
// EL CRITERIO ES ESTRUCTURAL Y NO UNA LISTA: son los campos cuyo tipo es una
// funcion cuyo PRIMER parametro es un context.Context. En serve.Config eso son
// exactamente los tres ganchos que hablan con el almacen de usuarios
// (Autenticar, HayAdmin, CrearAdmin) y ninguno de los otros campos-funcion, que
// son de infraestructura y su nil es benigno: `Reloj` es func() time.Time y
// `Salida` es un io.Writer.
//
// Se eligio esta forma y no «leer el comentario que dice Nil deniega» porque un
// comentario se reescribe sin querer y una firma no. Y no una lista escrita
// aqui, porque una lista al lado del test es la segunda copia que se queda vieja
// el dia que alguien anade el cuarto gancho, que es justo el dia que importa.
func decisionesDeIdentidadDeServeConfig(t *testing.T) []string {
	t.Helper()
	ruta := filepath.Join("..", "..", "superficies", "serve", "servidor.go")
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, ruta, nil, 0)
	if err != nil {
		t.Fatalf("no puedo parsear %s: %v", ruta, err)
	}
	var campos []string
	ast.Inspect(f, func(n ast.Node) bool {
		ts, ok := n.(*ast.TypeSpec)
		if !ok || ts.Name.Name != "Config" {
			return true
		}
		st, ok := ts.Type.(*ast.StructType)
		if !ok || st.Fields == nil {
			return false
		}
		for _, campo := range st.Fields.List {
			ft, ok := campo.Type.(*ast.FuncType)
			if !ok || ft.Params == nil || len(ft.Params.List) == 0 {
				continue
			}
			if !esContexto(ft.Params.List[0].Type) {
				continue
			}
			for _, nombre := range campo.Names {
				if nombre.IsExported() {
					campos = append(campos, nombre.Name)
				}
			}
		}
		return false
	})
	sort.Strings(campos)
	return campos
}

// esContexto dice si un tipo es context.Context.
func esContexto(e ast.Expr) bool {
	sel, ok := e.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Context" {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	return ok && pkg.Name == "context"
}

// camposDelLiteralDeConfig lee de cmd/plazum los literales serve.Config{...} y
// devuelve, por cada uno, los campos que pone y si los pone a nil.
//
// SE RECORRE EL DIRECTORIO ENTERO y no solo serve.go: si manana el cableado se
// parte en dos ficheros, un segundo literal que se olvidara de los ganchos
// seria exactamente el mismo fallo con otra ruta.
func camposDelLiteralDeConfig(t *testing.T) (literales int, puestos map[string]bool, aNil []string) {
	t.Helper()
	entradas, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("no puedo leer el directorio de la orden: %v", err)
	}
	puestos = map[string]bool{}
	fset := token.NewFileSet()
	for _, e := range entradas {
		nombre := e.Name()
		if e.IsDir() || !strings.HasSuffix(nombre, ".go") || strings.HasSuffix(nombre, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, nombre, nil, 0)
		if err != nil {
			t.Fatalf("no puedo parsear %s: %v", nombre, err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			cl, ok := n.(*ast.CompositeLit)
			if !ok {
				return true
			}
			sel, ok := cl.Type.(*ast.SelectorExpr)
			if !ok || sel.Sel.Name != "Config" {
				return true
			}
			pkg, ok := sel.X.(*ast.Ident)
			if !ok || pkg.Name != "serve" {
				return true
			}
			literales++
			for _, elt := range cl.Elts {
				kv, ok := elt.(*ast.KeyValueExpr)
				if !ok {
					continue
				}
				clave, ok := kv.Key.(*ast.Ident)
				if !ok {
					continue
				}
				puestos[clave.Name] = true
				// UN CAMPO PUESTO A nil NO ESTA PUESTO. Es la forma mas comoda
				// de cumplir la letra de esta puerta rompiendo su fondo, y es
				// gratis cazarla aqui.
				if id, ok := kv.Value.(*ast.Ident); ok && id.Name == "nil" {
					aNil = append(aNil, clave.Name)
				}
			}
			return true
		})
	}
	return literales, puestos, aNil
}

// TestPlazumServeCableaTodaDecisionDeIdentidadDeServeConfig es la puerta barata.
//
// EL ESTRENO FUE ROJO SOBRE EL ARBOL REAL, y esa es la unica forma de saber que
// vigila: escrita contra el `cmd/plazum/serve.go` de antes de este commit, decia
// que faltaban los tres campos.
func TestPlazumServeCableaTodaDecisionDeIdentidadDeServeConfig(t *testing.T) {
	identidad := decisionesDeIdentidadDeServeConfig(t)
	// EL SUELO PROTEGE DEL VERDE VACIO. Si el detector de AST deja de casar
	// (serve.Config se mueve de fichero, o los ganchos cambian de forma), el
	// conjunto sale vacio y todo lo de abajo se pondria verde recorriendo la
	// nada, que es la misma familia que `go test -run` sin casar.
	if len(identidad) < 3 {
		t.Fatalf("de serve.Config salen %d decisiones de identidad (%v) y hoy son al menos "+
			"3 (Autenticar, HayAdmin, CrearAdmin). O el servidor ha cambiado de forma, o "+
			"este detector ha dejado de casar y esta puerta mide el vacio",
			len(identidad), identidad)
	}

	literales, puestos, aNil := camposDelLiteralDeConfig(t)
	if literales == 0 {
		t.Fatalf("no hay ni un literal serve.Config{...} en cmd/plazum. O el cableado se " +
			"escribe ahora de otra forma, o esta puerta ha dejado de encontrarlo")
	}
	for _, campo := range identidad {
		if !puestos[campo] {
			t.Errorf("serve.Config.%s es una decision de identidad y `plazum serve` NO la "+
				"pasa.\n"+
				"  Su valor cero es denegar: sin ella, quien acaba de instalar plazum se "+
				"encuentra pantallas que contestan 401 y una instalacion que no puede "+
				"terminar.\n"+
				"  Es el estado exacto en el que estuvo el producto hasta el 03-09-2026, con "+
				"las dos mitades en verde.\n"+
				"  Arreglo: pasarla en el literal serve.Config de cmd/plazum/serve.go, desde "+
				"adaptadores/usuarios.", campo)
		}
	}
	for _, campo := range aNil {
		for _, id := range identidad {
			if campo == id {
				t.Errorf("serve.Config.%s se pasa explicitamente a nil. Escribir el campo no "+
					"es cablearlo: nil es exactamente el valor que deniega", campo)
			}
		}
	}
}

// --- la puerta conductual ---

// reCSRF saca el token del formulario. El NOMBRE del campo no se escribe aqui:
// sale de serve.CampoCSRF, que es quien lo declara.
var reCSRF = regexp.MustCompile(`name="` + regexp.QuoteMeta(serve.CampoCSRF) +
	`" value="([0-9a-f]+)"`)

func tokenCSRFDe(t *testing.T, pagina string) string {
	t.Helper()
	m := reCSRF.FindStringSubmatch(pagina)
	if m == nil {
		t.Fatalf("la pagina no trae el campo CSRF %q, asi que no se puede enviar el "+
			"formulario. Pagina:\n%s", serve.CampoCSRF, recortarPagina(pagina))
	}
	return m[1]
}

func recortarPagina(s string) string {
	if len(s) > 1500 {
		return s[:1500] + "\n[...]"
	}
	return s
}

// TestUnaInstalacionNuevaRecorreLosSeisPasosDelCamino es LA puerta de este
// frente, y la que da nombre al criterio de exito del tramo.
//
// Levanta `plazum serve` sobre un directorio VACIO, hace exactamente lo que hace
// una persona (leer el token del terminal, abrir /primer-admin, pegarlo, elegir
// credenciales) y recorre los seis pasos de camino.Canonico().
//
// LOS PASOS SE ENUMERAN DEL PRODUCTO, no de una lista de aqui: un septimo paso
// entra solo en esta puerta el dia que el camino lo declare, en vez de quedarse
// fuera en silencio.
func TestUnaInstalacionNuevaRecorreLosSeisPasosDelCamino(t *testing.T) {
	pasos := camino.Canonico()
	if len(pasos) < 6 {
		t.Fatalf("el camino declara %d pasos y la v1 nombra seis. Con menos, esta puerta "+
			"estaria midiendo otro camino", len(pasos))
	}

	s := arrancarServe(t)

	// 1. ANTES DE INSTALAR. Ningun paso se sirve, y eso es correcto: una
	//    instalacion sin administrador no ensena nada. Lo que se exige es que
	//    lleve a la instalacion y no a un callejon, o sea 303 a /primer-admin y
	//    NUNCA un 401 ni un 5xx.
	pantallas := 0
	for _, p := range pasos {
		if !p.EsPantalla() {
			continue
		}
		pantallas++
		codigo, cabeceras, _ := s.pedirCrudo(t, p.Ruta)
		if codigo == http.StatusUnauthorized {
			t.Errorf("sin instalar, el paso %q (%s) contesta 401. Un 401 a quien acaba de "+
				"descargar el producto es un callejon: no dice que hacer y no hay "+
				"credenciales que dar", p.ID, p.Ruta)
			continue
		}
		if codigo != http.StatusSeeOther {
			t.Errorf("sin instalar, el paso %q (%s) contesta %d y tenia que redirigir a la "+
				"instalacion", p.ID, p.Ruta, codigo)
			continue
		}
		if destino := cabeceras.Get("Location"); destino != "/primer-admin" {
			t.Errorf("sin instalar, el paso %q redirige a %q en vez de a /primer-admin",
				p.ID, destino)
		}
	}
	if pantallas < 6 {
		t.Fatalf("solo %d de los pasos del camino son pantalla y hoy son 6: esta puerta "+
			"estaria midiendo medio camino", pantallas)
	}

	// 2. LA INSTALACION, por HTTP y con el token que salio por el terminal.
	s.instalar(t)

	// 3. LOS SEIS PASOS, con la sesion del administrador. Esto es el criterio de
	//    exito del tramo entero.
	for _, p := range pasos {
		if !p.EsPantalla() {
			continue
		}
		codigo, _, cuerpo := s.pedir(t, p.Ruta)
		if codigo == http.StatusUnauthorized {
			t.Errorf("recien instalado y con sesion de administrador, el paso %q (%s) "+
				"contesta 401.\n"+
				"  Es el fallo que este fichero existe para cazar: el camino guiado tiene "+
				"un tramo que nadie puede recorrer.", p.ID, p.Ruta)
			continue
		}
		if codigo != http.StatusOK {
			t.Errorf("recien instalado, el paso %q (%s) contesta %d", p.ID, p.Ruta, codigo)
			continue
		}
		if !strings.Contains(cuerpo, "<main") {
			t.Errorf("el paso %q contesta 200 y no trae <main>: no es la pantalla, es otra "+
				"cosa con su codigo", p.ID)
		}
	}
}

// TestSinSesionLasPantallasConNombresDePersonasSiguenSinServirse es el CONTROL
// POSITIVO del descargo.
//
// La puerta de arriba comprueba que con sesion se llega. Sin esta, el arreglo
// mas comodo para ponerla verde seria quitarle la proteccion a las tres
// pantallas que llevan nombres de personas, y nada se pondria rojo. Aqui se
// recorre esa rama con dato real: instalada la aplicacion, y por tanto sin la
// redireccion a /primer-admin de por medio, una peticion SIN cookie a /acta/,
// /uar/ y /escalado/ no puede salir con 200.
func TestSinSesionLasPantallasConNombresDePersonasSiguenSinServirse(t *testing.T) {
	s := arrancarServe(t)
	s.instalar(t)

	// Las tres se nombran por su ID del camino, no por su ruta: la ruta la
	// declara el camino y cambiarla no puede dejar esta comprobacion mirando a
	// una direccion que ya no existe.
	for _, id := range []string{camino.IDDelActa, camino.IDDeLaUAR, camino.IDDelEscalado} {
		ruta, esPantalla := camino.RutaDe(id)
		if !esPantalla {
			t.Fatalf("el paso %q ha dejado de ser pantalla: este control positivo ya no "+
				"recorre lo que dice recorrer", id)
		}
		codigo, _, _ := s.pedirCrudo(t, ruta)
		if codigo == http.StatusOK {
			t.Errorf("con la aplicacion instalada, %s (%s) se sirve con 200 SIN sesion, y "+
				"esa pantalla lleva nombres de personas dentro", id, ruta)
		}
	}
}

// pedirCrudo pide una ruta SIN cookies y sin seguir redirecciones, que es la
// unica forma de ver el codigo que de verdad contesta el servidor.
func (s *servidorServe) pedirCrudo(t *testing.T, ruta string) (int, http.Header, string) {
	t.Helper()
	return s.pedirCon(t, s.crudo, ruta)
}

// pedir usa el cliente con la sesion del administrador.
func (s *servidorServe) pedir(t *testing.T, ruta string) (int, http.Header, string) {
	t.Helper()
	return s.pedirCon(t, s.cli, ruta)
}

func (s *servidorServe) pedirCon(t *testing.T, cli *http.Client, ruta string) (int, http.Header, string) {
	t.Helper()
	resp, err := cli.Get(s.base + ruta)
	if err != nil {
		t.Fatalf("pidiendo %s: %v", ruta, err)
	}
	cuerpo := leerHasta(t, resp.Body, 1<<20)
	if err := resp.Body.Close(); err != nil {
		t.Fatal(err)
	}
	return resp.StatusCode, resp.Header, cuerpo
}

// instalar hace lo que hace una persona: lee el token del terminal, abre
// /primer-admin, lo pega y elige credenciales. Deja la sesion en s.cli.
func (s *servidorServe) instalar(t *testing.T) {
	t.Helper()
	if s.instalado {
		return
	}
	tok := s.tokenDeInstalacion(t)

	codigo, _, formulario := s.pedirCon(t, s.cli, "/primer-admin")
	if codigo != http.StatusOK {
		t.Fatalf("GET /primer-admin contesta %d en una instalacion nueva. Sin esa pantalla "+
			"no hay forma de entrar en plazum.\n%s", codigo, recortarPagina(formulario))
	}
	valores := url.Values{
		serve.CampoCSRF: {tokenCSRFDe(t, formulario)},
		"token":         {tok},
		"usuario":       {UsuarioDePrueba},
		"secreto":       {SecretoDePrueba},
		// EL NOMBRE DE LA ORGANIZACION, que este formulario pregunta desde
		// que la instalacion sabe quien es. Es de la INSTALACION y no de la
		// cuenta, y por eso se pregunta aqui una vez y no en cada entrevista.
		"organizacion": {OrganizacionDePrueba},
	}
	resp, err := s.cli.PostForm(s.base+"/primer-admin", valores)
	if err != nil {
		t.Fatalf("enviando el formulario de primer administrador: %v", err)
	}
	cuerpo := leerHasta(t, resp.Body, 1<<20)
	if err := resp.Body.Close(); err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("POST /primer-admin contesta %d y tenia que crear el administrador y "+
			"redirigir.\n%s", resp.StatusCode, recortarPagina(cuerpo))
	}
	s.instalado = true
}

// reTokenDeInstalacion casa el token que imprime el arranque: 64 caracteres
// hexadecimales solos en su linea (secretos.Token(32)).
var reTokenDeInstalacion = regexp.MustCompile(`(?m)^\s*([0-9a-f]{64})\s*$`)

func (s *servidorServe) tokenDeInstalacion(t *testing.T) string {
	t.Helper()
	texto := s.salida()
	m := reTokenDeInstalacion.FindStringSubmatch(texto)
	if m == nil {
		t.Fatalf("`plazum serve` no ha impreso ningun token de primer administrador al "+
			"arrancar sobre un directorio vacio. Sin el, la instalacion no se puede "+
			"terminar y no hay forma de entrar.\n--- salida del arranque ---\n%s",
			recortarPagina(texto))
	}
	return m[1]
}
