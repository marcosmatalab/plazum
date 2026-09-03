package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/marcosmatalab/plazum/internal/modulo"
	"github.com/marcosmatalab/plazum/superficies/calendario"
	"github.com/marcosmatalab/plazum/superficies/camino"
	escaladoWeb "github.com/marcosmatalab/plazum/superficies/escalado"
	"github.com/marcosmatalab/plazum/superficies/pantallas"
)

// EL TRINQUETE DE ALCANZABILIDAD, MITAD DE LAS SUPERFICIES.
//
// La declaracion vive en alcanzabilidad.go, al lado del cableado, y ahi esta el
// porque. Aqui esta lo que la comprueba, y lo que importa de este fichero es que
// NO SE CREE LA DECLARACION:
//
//	el conjunto de superficies se ENUMERA RECORRIENDO superficies/, no de una
//	lista escrita aqui, porque una lista escrita al lado del test se
//	desincroniza el dia que alguien anade un paquete, que es el dia en que
//	hace falta;
//	cual de ellas SIRVE HTTP se decide leyendo su AST, no por el nombre ni por
//	una anotacion, porque el paquete que se olvide de anotarse es justo el que
//	hay que cazar;
//	y «montada» no se da por buena por estar escrito: se levanta el servidor
//	de verdad y se le pide la ruta.
//
// La diferencia entre esta puerta y la que ya habia (TestElCaminoGuiadoNoTiene
// Callejones) es la direccion en la que miran. Aquella recorre lo DECLARADO en
// el camino y comprueba que contesta; si una superficie no esta declarada en
// ningun sitio, no la ve. Esta parte del ARBOL, que es el unico conjunto que
// nadie puede olvidarse de actualizar. La que faltaba era esta.

// superficiesDelArbol enumera los paquetes bajo superficies/ y dice cuales
// sirven HTTP.
//
// SIRVE HTTP = declara un metodo ServeHTTP(http.ResponseWriter, *http.Request)
// en un fichero que no es de test. Es el interfaz http.Handler, o sea la unica
// definicion que no depende de que alguien se acuerde de declararse.
//
// Los ficheros _test.go se excluyen porque un doble de prueba que implemente
// http.Handler no es una superficie del producto; los subdirectorios internal/
// tambien, porque no son paquetes montables por si mismos.
func superficiesDelArbol(t *testing.T) (todas []string, sirvenHTTP map[string]string) {
	t.Helper()
	raiz := filepath.Join("..", "..", "superficies")
	entradas, err := os.ReadDir(raiz)
	if err != nil {
		t.Fatalf("no puedo leer %s: %v", raiz, err)
	}
	sirvenHTTP = map[string]string{}
	for _, e := range entradas {
		if !e.IsDir() {
			continue // superficies/doc.go
		}
		nombre := e.Name()
		todas = append(todas, nombre)
		if fichero := declaraServeHTTP(t, filepath.Join(raiz, nombre)); fichero != "" {
			sirvenHTTP[nombre] = fichero
		}
	}
	sort.Strings(todas)
	return todas, sirvenHTTP
}

// declaraServeHTTP devuelve el fichero donde el paquete declara ServeHTTP, o
// cadena vacia. No entra en subdirectorios: un internal/ no es montable.
func declaraServeHTTP(t *testing.T, dir string) string {
	t.Helper()
	entradas, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("no puedo leer %s: %v", dir, err)
	}
	fset := token.NewFileSet()
	for _, e := range entradas {
		nombre := e.Name()
		if e.IsDir() || !strings.HasSuffix(nombre, ".go") || strings.HasSuffix(nombre, "_test.go") {
			continue
		}
		ruta := filepath.Join(dir, nombre)
		f, err := parser.ParseFile(fset, ruta, nil, 0)
		if err != nil {
			t.Fatalf("%s: %v", ruta, err)
		}
		for _, decl := range f.Decls {
			fd, ok := decl.(*ast.FuncDecl)
			if !ok || fd.Name.Name != "ServeHTTP" || fd.Recv == nil {
				continue
			}
			if esFirmaDeHandler(fd) {
				return ruta
			}
		}
	}
	return ""
}

// esFirmaDeHandler comprueba los dos parametros de http.Handler. Mirar solo el
// nombre del metodo cazaria un `ServeHTTP()` cualquiera; el interfaz es la
// firma, no el nombre.
func esFirmaDeHandler(fd *ast.FuncDecl) bool {
	if fd.Type.Params == nil || len(fd.Type.Params.List) != 2 {
		return false
	}
	sel, ok := fd.Type.Params.List[0].Type.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "ResponseWriter" {
		return false
	}
	estrella, ok := fd.Type.Params.List[1].Type.(*ast.StarExpr)
	if !ok {
		return false
	}
	sel2, ok := estrella.X.(*ast.SelectorExpr)
	return ok && sel2.Sel.Name == "Request"
}

// TestTodaSuperficieHTTPOSeMontaOSeExplica es la puerta.
//
// EL ESTRENO SE HIZO CONTRA EL ARBOL REAL antes que contra ninguna mutacion, y
// nacio roja: `superficies/scim` no la monta nadie y no habia una sola linea en
// el arbol que dijera que eso era a proposito. Es el mismo estado en el que
// estuvo el acta, y esta vez no habia que fiarse de que alguien se acordara.
func TestTodaSuperficieHTTPOSeMontaOSeExplica(t *testing.T) {
	todas, sirvenHTTP := superficiesDelArbol(t)
	// Los dos minimos contados protegen del fallo silencioso de esta misma
	// puerta: si el recorrido o el detector de AST dejan de casar, los mapas
	// salen vacios y todo lo de abajo se pone verde recorriendo la nada.
	if len(todas) < 8 {
		t.Fatalf("bajo superficies/ salen %d paquetes (%v) y hoy hay al menos 8. O han "+
			"desaparecido, o este recorrido esta midiendo el vacio", len(todas), todas)
	}
	if len(sirvenHTTP) < 6 {
		t.Fatalf("solo %d de los %d paquetes de superficies/ declaran ServeHTTP (%v) y hoy "+
			"son al menos 6. El detector de AST ha dejado de casar y esta puerta estaria "+
			"dando verde sobre un conjunto vacio",
			len(sirvenHTTP), len(todas), clavesDeSuperficies(sirvenHTTP))
	}

	// SENTIDO 1: todo paquete que sirva HTTP tiene que estar declarado, y no
	// con el valor cero.
	for _, nombre := range clavesDeSuperficies(sirvenHTTP) {
		d, declarada := SuperficiesHTTP[nombre]
		if !declarada {
			t.Errorf("superficies/%s sirve HTTP (%s) y NO sale en SuperficiesHTTP.\n"+
				"  El silencio es el estado que dejo la pantalla del acta construida, con "+
				"sus tests en verde y sin una sola direccion del producto que llevara a "+
				"ella. Arreglo: declarala en alcanzabilidad.go, montada o no montada a "+
				"proposito con su motivo.", nombre, sirvenHTTP[nombre])
			continue
		}
		if d.Estado == SinDeclarar {
			t.Errorf("superficies/%s esta en el censo con el VALOR CERO (%s). El cero no es "+
				"un estado, es el olvido: escribe cual de los tres es", nombre, d.Estado)
		}
	}

	// SENTIDO 2: toda entrada del censo tiene que existir y seguir sirviendo
	// HTTP. Sin esta mitad el censo envejece hasta ser una lista de lo que hubo.
	for _, nombre := range clavesDelCensoDeSuperficies() {
		if _, hay := sirvenHTTP[nombre]; hay {
			continue
		}
		if existeDirectorio(t, filepath.Join("..", "..", "superficies", nombre)) {
			t.Errorf("el censo declara superficies/%s y ese paquete ya NO sirve HTTP. O dejo "+
				"de ser una superficie web, y entonces sale del censo, o alguien se llevo "+
				"su ServeHTTP por delante", nombre)
			continue
		}
		t.Errorf("el censo declara superficies/%s y ese paquete no existe en el arbol",
			nombre)
	}

	// SENTIDO 3: cada estado exige lo suyo, y lo que exige no es decorativo.
	for _, nombre := range clavesDelCensoDeSuperficies() {
		d := SuperficiesHTTP[nombre]
		switch d.Estado {
		case Montada:
			// Una superficie montada promete que SE LLEGA a ella desde el
			// camino. Sin paso declarado, «montada» solo significa que
			// contesta si tecleas la direccion, que es lo que decia el acta.
			if strings.TrimSpace(d.PasoDelCamino) == "" {
				t.Errorf("superficies/%s se declara montada y no dice por que paso del "+
					"camino se llega. Montada sin paso es alcanzable solo tecleando la "+
					"direccion, o sea alcanzable solo para quien ya sabia que existia",
					nombre)
				continue
			}
			ruta, esPantalla := camino.RutaDe(d.PasoDelCamino)
			if !esPantalla {
				t.Errorf("superficies/%s dice llegar por el paso %q y el camino canonico no "+
					"declara ese paso como pantalla. El enlace lleva a la nada",
					nombre, d.PasoDelCamino)
			} else if ruta == "" {
				t.Errorf("superficies/%s dice llegar por el paso %q y ese paso no tiene ruta",
					nombre, d.PasoDelCamino)
			}
			if strings.TrimSpace(d.ComoSeLevanta) != "" {
				t.Errorf("superficies/%s esta montada y declara ComoSeLevanta (%q). Ese "+
					"campo es para las que NO monta el servidor: aqui afirma que hace "+
					"falta un paso manual que no hace falta", nombre, d.ComoSeLevanta)
			}
		case MontadaFueraDelCamino, NoMontadaAProposito:
			if strings.TrimSpace(d.Motivo) == "" {
				t.Errorf("superficies/%s se declara %q y no dice por que. Un hueco sin "+
					"motivo se lee como deuda y como decision a la vez, y la lectura "+
					"barata («ya se montara») es la que dejo el acta suelta",
					nombre, d.Estado)
			}
			if d.PasoDelCamino != "" {
				t.Errorf("superficies/%s se declara %q y ademas nombra el paso %q. O es un "+
					"paso del camino o no lo es", nombre, d.Estado, d.PasoDelCamino)
			}
			// Y LA QUE NO MONTA EL SERVIDOR TIENE QUE PODER LEVANTARSE.
			// D11-b aplicada a una superficie entera: sin verbo, «apagada a
			// proposito» es indistinguible de «muerta».
			if d.Estado == NoMontadaAProposito && strings.TrimSpace(d.ComoSeLevanta) == "" {
				t.Errorf("superficies/%s no la monta el servidor y no dice como se levanta. "+
					"Una superficie que no se puede poner en pie de ninguna forma no esta "+
					"deliberadamente apagada: esta muerta", nombre)
			}
		}
	}
}

// LA TABLA DE MONTAJE DE VERDAD, Y POR QUE CASA POR EL PAQUETE.
//
// AQUI CAYO LA PRIMERA VERSION DE ESTA PUERTA, y merece contarse porque es el
// invariante 7 en una pieza recien escrita. La comprobacion era: «para cada
// superficie declarada Montada, mira que la ruta de su paso conteste». Eso
// empareja por `PasoDelCamino`, que es UNA CADENA QUE ESCRIBE A MANO quien
// rellena el censo, y no la comprueba nadie contra el montaje.
//
// La mutacion que la tumbo es la mentira mas comoda que existe aqui: declarar
// `scim` como Montada en el paso "uar". La puerta se puso VERDE, porque /uar/
// contesta de verdad, solo que lo contesta la superficie de la UAR. Una
// superficie puede heredar el cable de otra con solo escribir su nombre, que es
// exactamente «piezas terminadas sin el cable» con una capa de pintura encima.
//
// EL ARREGLO ES EMPAREJAR POR ALGO QUE NO SE PUEDA ESCRIBIR A MANO: la ruta de
// import del paquete del handler que `plazum serve` monta DE VERDAD, sacada por
// reflexion del objeto montado. Ese dato no sale de una cadena del censo, sale
// del cableado, asi que una entrada del censo no puede fabricarlo.
func tablaDeMontaje(t *testing.T) map[string]string {
	t.Helper()
	cat := catDePrueba(t)
	act := actaDePrueba(t)
	rev, err := construirUAR(opcionesUAR{Catalogo: cat, Quien: func(*http.Request) string { return "ciso" }})
	if err != nil {
		t.Fatal(err)
	}
	cal, err := construirCalendario(cat, nil)
	if err != nil {
		t.Fatal(err)
	}
	esc, err := construirEscalado(cat, func(*http.Request) string { return "ciso" }, nil)
	if err != nil {
		t.Fatal(err)
	}
	// LAS MISMAS LLAMADAS QUE HACE cmdServe, no una reconstruccion parecida: si
	// esto se separa del cableado real, la puerta mide otra cosa.
	montajes := montajesDelCamino(caminoDePrueba(t), act, rev, cal, esc)

	porPaquete := map[string]string{}
	// pantallas se monta en la raiz y no pasa por montajesDelCamino, asi que
	// se construye igual que lo hace cmdServe.
	app, err := pantallas.Nuevo(pantallas.Opciones{
		Catalogo:    cat,
		CaminoRuta:  camino.BasePorDefecto + "/",
		CaminoClave: camino.ClaveTitulo,
	})
	if err != nil {
		t.Fatal(err)
	}
	porPaquete[paqueteDe(app)] = "/"
	for _, m := range montajes {
		if m.h == nil || m.prefijo == "" {
			continue
		}
		porPaquete[paqueteDe(m.h)] = m.prefijo
	}
	return porPaquete
}

// paqueteDe da la ruta de import del paquete que declara el tipo del handler.
// Es el campo por el que se empareja, y no se puede escribir en el censo.
func paqueteDe(h http.Handler) string {
	tipo := reflect.TypeOf(h)
	for tipo != nil && tipo.Kind() == reflect.Ptr {
		tipo = tipo.Elem()
	}
	if tipo == nil {
		return ""
	}
	return tipo.PkgPath()
}

// TestElCensoDeSuperficiesCasaConElMontajeReal es la mitad que convierte la
// declaracion en un hecho.
//
// POR QUE HACE FALTA APARTE de la puerta de arriba: aquella lee texto y
// comprueba que lo escrito es coherente consigo mismo y con el arbol. Nada de
// eso impide escribir `Montada` en una superficie que el servidor no cuelga,
// que es el error que se quiere cazar y el mas comodo de cometer. Aqui se
// construye la tabla de montaje REAL y se cruza por la ruta del paquete.
func TestElCensoDeSuperficiesCasaConElMontajeReal(t *testing.T) {
	montadas := tablaDeMontaje(t)
	// ESTE SUELO SOLO PROTEGE DEL COLAPSO, no del hueco. Si se pone alto (habia
	// un 4), se lleva por delante el diagnostico bueno: al desmontar el acta
	// saltaba ESTE Fatalf y no el mensaje que nombra al acta, o sea que la
	// puerta decia «mides el vacio» donde tenia que decir «falta este cable».
	// El hueco de una superficie concreta lo caza el cruce de abajo, que lo
	// nombra; aqui solo se comprueba que la tabla no venga vacia.
	if len(montadas) < 2 {
		t.Fatalf("la tabla de montaje real trae %d superficies (%v). Con eso no hay nada que "+
			"cruzar: `plazum serve` ha dejado de montar, o esta puerta mide el vacio",
			len(montadas), montadas)
	}
	// LA RUTA DEL MODULO SE LEE DE go.mod, no se escribe aqui. Escrita, seria
	// una segunda copia que no se rompe cuando la primera cambia: se queda
	// vieja y sigue dando verde. Lo caza TestNadieCableaLaRutaDelModulo, y de
	// hecho lo cazo mientras se escribia esta puerta.
	mod, err := modulo.Ruta()
	if err != nil {
		t.Fatalf("leyendo la ruta del modulo de go.mod: %v", err)
	}
	prefijo := mod + "/superficies/"

	h := servidorDelCamino(t, nil, func(*http.Request) string { return "ciso" }).Handler()
	pasoTomado := map[string]string{}
	comprobadas := 0
	for _, nombre := range clavesDelCensoDeSuperficies() {
		d := SuperficiesHTTP[nombre]
		_, estaMontada := montadas[prefijo+nombre]
		declaraMontaje := d.Estado == Montada || d.Estado == MontadaFueraDelCamino

		// EL CRUCE, en los dos sentidos. Ninguno de los dos sobra: el primero
		// caza la superficie que se declara montada y no lo esta (el acta), y
		// el segundo la que se declara apagada y el servidor esta colgando
		// igual, que es peor porque es una superficie viva que nadie vigila.
		//
		// `serve` es la excepcion y no una trampa: es QUIEN monta, asi que no
		// aparece en su propia tabla de montaje. Se exime nombrandolo, y se
		// exige que siga siendo el unico eximido.
		if nombre == "serve" {
			if estaMontada {
				t.Errorf("superficies/serve sale en la tabla de montaje. Es quien monta: si " +
					"ademas esta montado, alguien lo ha colgado de si mismo")
			}
			continue
		}
		comprobadas++
		if declaraMontaje && !estaMontada {
			t.Errorf("superficies/%s se declara %q y NO sale en la tabla de montaje que "+
				"construye `plazum serve`. Es el estado exacto de la pantalla del acta: "+
				"la declaracion afirma un cable que no existe", nombre, d.Estado)
		}
		if !declaraMontaje && estaMontada {
			t.Errorf("superficies/%s se declara %q y el servidor la cuelga en %q. Una "+
				"superficie viva declarada apagada es la que nadie vigila",
				nombre, d.Estado, montadas[prefijo+nombre])
		}

		if d.Estado != Montada {
			continue
		}
		// UN PASO LO RECLAMA UNA SOLA SUPERFICIE. Sin esto, dos entradas pueden
		// nombrar el mismo paso y la segunda hereda el cable de la primera, que
		// es como se colo `scim` en la version anterior de esta puerta.
		if otra, repetido := pasoTomado[d.PasoDelCamino]; repetido {
			t.Errorf("superficies/%s y superficies/%s reclaman el mismo paso %q. Solo una lo "+
				"sirve: la otra esta heredando un cable ajeno",
				otra, nombre, d.PasoDelCamino)
		}
		pasoTomado[d.PasoDelCamino] = nombre

		// Y EL PREFIJO EN EL QUE SE MONTA TIENE QUE SER EL DEL PASO QUE DICE.
		// Aqui es donde una superficie montada en otro sitio deja de poder
		// decir que es un paso del camino.
		ruta, esPantalla := camino.RutaDe(d.PasoDelCamino)
		if !esPantalla {
			continue // ya lo dijo la puerta de arriba
		}
		if prefijoMontado := montadas[prefijo+nombre]; !rutaCaeBajo(ruta, prefijoMontado) {
			t.Errorf("superficies/%s dice ser el paso %q (%s) y el servidor la monta en %q. "+
				"El enlace del camino no lleva a esta superficie",
				nombre, d.PasoDelCamino, ruta, prefijoMontado)
		}
		// Y POR ULTIMO CONTESTA DE VERDAD, por la cadena entera de middleware.
		if codigo, _ := pedirCamino(t, h, ruta); codigo == http.StatusNotFound {
			t.Errorf("superficies/%s se declara montada en %s y el servidor responde 404",
				nombre, ruta)
		}
	}
	if comprobadas < 4 {
		t.Fatalf("solo se han cruzado %d superficies y el censo declara varias mas", comprobadas)
	}
}

// LA DIRECCION CONTRARIA, Y ES LA QUE FALTABA.
//
// La puerta de arriba recorre las superficies declaradas Montada y comprueba
// que su paso existe y contesta. Nadie recorria el sentido opuesto: una
// superficie declarada FUERA del camino cuyo paso el camino SI declara. Ese
// estado no da error en ningun sitio y se lee como una decision, cuando es una
// declaracion que se quedo vieja: la pantalla estaria en el camino y el censo
// diciendo que no, o sea que nadie la vigilaria como paso.
//
// ES EL ESTADO QUE ESTE FRENTE DEJA A PROPOSITO. El calendario y el escalado se
// montan hoy en /calendario/ y /escalado/ y el camino todavia no declara sus
// rutas, porque superficies/camino es de otro frente. Esta puerta es lo que
// obliga a cerrar el circulo: en cuanto el camino declare una de esas dos
// rutas, esta se pone roja y el censo tiene que pasar a Montada en el mismo
// commit.
//
// NACE VERDE Y SE DICE EN VOZ ALTA. Hoy no hay ninguna superficie en ese
// estado (la unica MontadaFueraDelCamino con prefijo propio es el camino
// mismo, y ningun paso cuelga de /camino), asi que esta puerta no ha encontrado
// nada al nacer: es un trinquete para un cambio que se sabe que va a llegar, no
// un hallazgo. Su fallo se demuestra con mutacion, que es lo unico que puede
// demostrarlo mientras el corpus de superficies no lo alcance.
func TestUnaSuperficieFueraDelCaminoNoPuedeSerYaUnPasoDelCamino(t *testing.T) {
	pasos := camino.Canonico()
	if len(pasos) < 6 {
		t.Fatalf("el camino declara %d pasos: con menos, esta puerta recorreria casi nada",
			len(pasos))
	}
	montadas := tablaDeMontaje(t)
	mod, err := modulo.Ruta()
	if err != nil {
		t.Fatalf("leyendo la ruta del modulo de go.mod: %v", err)
	}
	prefijo := mod + "/superficies/"

	comprobadas := 0
	for _, nombre := range clavesDelCensoDeSuperficies() {
		d := SuperficiesHTTP[nombre]
		if d.Estado != MontadaFueraDelCamino {
			continue
		}
		prefijoMontado, esta := montadas[prefijo+nombre]
		if !esta || prefijoMontado == "" || prefijoMontado == "/" {
			// La raiz sirve todo lo que no reclame otro, asi que compararla
			// contra las rutas de los pasos diria que si a todo.
			continue
		}
		comprobadas++
		for _, p := range pasos {
			if !p.EsPantalla() {
				continue
			}
			if !rutaCaeBajo(p.Ruta, prefijoMontado) {
				continue
			}
			t.Errorf("superficies/%s se declara %q y el camino guiado YA declara el paso %q "+
				"en %q, que cae bajo su montaje %q.\n"+
				"  O sea que es un paso del camino y el censo dice que no lo es: nadie la "+
				"vigila como paso, y las comprobaciones que exigen que un paso conteste no "+
				"la miran.\n"+
				"  Arreglo: pasarla a Montada con PasoDelCamino: %q y quitarle el Motivo.",
				nombre, d.Estado, p.ID, p.Ruta, prefijoMontado, p.ID)
		}
	}
	// El suelo protege del verde por vacio: si el censo se queda sin ninguna
	// superficie montada fuera del camino con prefijo propio, este test dejaria
	// de mirar nada y seguiria verde.
	if comprobadas == 0 {
		t.Fatal("no hay ni una superficie declarada fuera del camino con prefijo propio, " +
			"asi que esta puerta esta recorriendo el vacio. Hoy tendrian que estar al menos " +
			"el calendario y el escalado")
	}
}

// LA GUARDA DE LA CAIDA, y sin ella el montaje de estas dos pantallas es una
// bomba con la mecha puesta.
//
// prefijoDeLaPantalla lee primero camino.RutaDe y CAE a la base que declara la
// superficie cuando el camino no dice nada. Esa caida es lo que permite montar
// hoy sin tocar el fichero de otro frente. Su peligro es evidente en cuanto se
// escribe: el dia que el camino declare "/agenda" para el paso del calendario,
// la pantalla seguiria montada en /calendario/ y el enlace del camino daria 404
// sin que nada se pusiera rojo, porque la caida se habria dejado de usar y nadie
// habria comparado las dos cadenas.
//
// Aqui se comparan. Mientras el camino no declare la ruta no hay nada que
// comprobar y se dice; en cuanto la declare, tiene que ser EXACTAMENTE la base
// de su superficie.
func TestSiElCaminoDeclaraLaRutaTieneQueSerLaDeLaSuperficie(t *testing.T) {
	casos := []struct{ paso, base string }{
		{camino.IDDelCalendario, calendario.BasePorDefecto},
		{camino.IDDelEscalado, escaladoWeb.BasePorDefecto},
	}
	declaradas := 0
	for _, c := range casos {
		ruta, esPantalla := camino.RutaDe(c.paso)
		if !esPantalla {
			continue
		}
		declaradas++
		if strings.TrimSuffix(ruta, "/") != strings.TrimSuffix(c.base, "/") {
			t.Errorf("el camino declara el paso %q en %q y su superficie se monta en %q.\n"+
				"  prefijoDeLaPantalla ya no cae a la base, asi que el montaje se ha ido a "+
				"la ruta del camino y la constante de la superficie se ha quedado mintiendo, "+
				"o al reves. Las dos tienen que decir lo mismo.", c.paso, ruta, c.base)
		}
	}
	if declaradas == 0 {
		t.Logf("el camino todavia no declara la ruta de ninguno de los %d pasos vigilados "+
			"(calendario, escalado), asi que esta comprobacion no ha comparado nada. Es el "+
			"estado esperado hasta que se aplique el cambio a superficies/camino", len(casos))
	}
}

// rutaCaeBajo dice si la ruta del paso la sirve ese prefijo de montaje. La raiz
// sirve todo lo que no reclame otro, asi que es el caso comodin.
func rutaCaeBajo(ruta, prefijoMontado string) bool {
	if prefijoMontado == "" {
		return false
	}
	if prefijoMontado == "/" {
		return true
	}
	return strings.HasPrefix(ruta, strings.TrimSuffix(prefijoMontado, "/"))
}

// CONTROL NEGATIVO DEL DETECTOR DE HTTP.
//
// Todo cuelga de declaraServeHTTP, y esa funcion tiene dos formas de mentir en
// silencio: decir que si a todo (y entonces el sentido 1 exigiria declarar
// paquetes que no sirven nada) o decir que no a todo (y entonces ninguna
// superficie nueva se pediria jamas, que es el fallo que esta puerta existe
// para impedir y ademas es el que se lee como verde). Se recorren las dos ramas
// con paquetes reales del arbol.
func TestElDetectorDeSuperficiesHTTPDistingueLasDosRespuestas(t *testing.T) {
	todas, sirvenHTTP := superficiesDelArbol(t)

	// RAMA POSITIVA: los que sabemos que sirven, salen.
	for _, nombre := range []string{"pantallas", "acta", "uar"} {
		if _, hay := sirvenHTTP[nombre]; !hay {
			t.Errorf("el detector dice que superficies/%s no sirve HTTP, y es una pantalla "+
				"del producto. El detector esta roto, y roto en la direccion que deja "+
				"pasar superficies nuevas sin declarar", nombre)
		}
	}
	// RAMA NEGATIVA: los que no sirven, no salen. Se buscan en el arbol en vez
	// de escribirlos aqui, para que esto siga significando algo si cambian.
	sinHTTP := 0
	for _, nombre := range todas {
		if _, hay := sirvenHTTP[nombre]; !hay {
			sinHTTP++
		}
	}
	if sinHTTP == 0 {
		// EL EJEMPLO DE ESTE MENSAJE HAY QUE MANTENERLO VIVO. Decia «calendario
		// escribe un .ics, export escribe ficheros» y el calendario gano
		// pantalla el 03-09-2026, asi que la mitad del ejemplo era falsa y el
		// mensaje seguia saliendo igual: un mensaje de fallo que nombra un
		// estado que ya no existe manda a quien lo lea a mirar donde no es.
		t.Error("el detector dice que TODOS los paquetes de superficies/ sirven HTTP. " +
			"Hoy hay al menos uno que no (export escribe ficheros para un SIEM), asi que " +
			"esta diciendo que si a todo")
	}
}

func existeDirectorio(t *testing.T, ruta string) bool {
	t.Helper()
	info, err := os.Stat(ruta)
	if err != nil {
		if !os.IsNotExist(err) {
			t.Fatalf("mirando %s: %v", ruta, err)
		}
		return false
	}
	return info.IsDir()
}

// Los dos ordenadores de claves existen para que el informe de un fallo salga
// siempre igual: un mapa se recorre en orden distinto en cada ejecucion, y una
// puerta cuyo informe baila no se puede comparar entre dos ejecuciones.
func clavesDeSuperficies(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func clavesDelCensoDeSuperficies() []string {
	out := make([]string, 0, len(SuperficiesHTTP))
	for k := range SuperficiesHTTP {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
