package plazum

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// CADA PRIMITIVA DEL RELOJ, CON CUANTOS LA USAN Y SI SU SALIDA LA VE ALGUIEN.
//
// # La pregunta que contesta, y de donde sale
//
// «Un motor con un solo consumidor no esta probado, esta de acuerdo consigo
// mismo.» Es la leccion del bloque A2 (07-09-2026): `estado.Calcular` llevaba
// desde la etapa 1 con sus dorados en verde y **un solo llamante**, el
// verificador del expediente, que es un camino que nadie mira a diario. El dia
// que una segunda superficie lo llamo aparecieron **dos agujeros de etapa 1** que
// ninguna revision habia visto: `Valida()` existia y no se invocaba nunca, y
// `FailVencido` —el unico estado que escala— era inalcanzable con la
// configuracion normal.
//
// La pregunta natural es si el motor de RELOJES tiene el mismo problema, y hay
// que contestarla derivando, no estimando.
//
// # Que se mide aqui, y por que estas dos cosas
//
//	existe        la primitiva esta implementada en `nucleo/ventana`
//	la declara N  cuantas obligaciones del corpus la usan, y en cuantos paquetes
//
// **La segunda es la que importa y no es obvia.** Todas las primitivas comparten
// los mismos consumidores de salida (el calendario, el panel, el expediente,
// el `.ics`), porque todos consumen `ventana.Vencimiento` y ninguno sabe de que
// primitiva salio. Asi que «cuantos consumidores tiene» no distingue una de otra.
// Lo que SI las distingue es **si alguna obligacion real la enciende**: una
// primitiva que ningun paquete declara solo la ejercitan sus propios dorados, o
// sea que esta de acuerdo consigo misma, y su salida **no la ve nadie**.
//
// Es exactamente el caso `maximo` que ya se pago una vez: construido, probado y
// **apagado para el corpus** durante semanas, con ocho retenciones del CRA
// esperandolo sin que nadie lo hubiera notado.
//
// # El resultado, medido el 08-09-2026
//
// De las ocho primitivas del plan, **seis existen y las seis estan encendidas**;
// las otras dos (`secuencia` y `observacion`) **no existen ni como tipo**, asi
// que no son un motor de acuerdo consigo mismo: son trabajo sin empezar, que es
// otra cosa y mas honesta.
//
// **Ninguna sale «uno y ninguno»**, que era la respuesta que se temia. La mas
// fina es `puntual`, con 3 obligaciones en 1 paquete: no esta apagada, pero es
// la que menos dato real ha visto y la primera candidata a esconder un
// `FailVencido` propio.
func TestCadaPrimitivaDelRelojDiceSiAlguienLaEnciende(t *testing.T) {
	existen := primitivasImplementadas(t)
	declaradas, paquetes := primitivasDeclaradasEnElCorpus(t)

	// SENTIDO 1: toda primitiva implementada la enciende alguna obligacion.
	//
	// Es la puerta que el caso `maximo` habria puesto roja: construido, con sus
	// dorados en verde, y sin una sola obligacion del corpus que lo usara.
	for _, p := range existen {
		if declaradas[p] == 0 {
			t.Errorf("la primitiva %q esta implementada en nucleo/ventana y NINGUNA "+
				"obligacion del corpus la declara.\n"+
				"  Solo la ejercitan sus propios dorados, o sea que esta de acuerdo consigo\n"+
				"  misma: su salida no la ve una persona en ninguna pantalla.\n"+
				"  Ya paso con `maximo`, construido y apagado durante semanas con ocho\n"+
				"  retenciones del CRA esperandolo.\n"+
				"  Arreglo: encenderla en el paquete que la necesita, o decir aqui por que\n"+
				"  sigue apagada y desde cuando.", p)
		}
	}

	// SENTIDO 2: toda primitiva que el corpus declara existe. Sin esta mitad,
	// un paquete podria declarar una primitiva inventada y el fallo saldria
	// como un reloj que no vence en vez de como un error.
	hay := map[string]bool{}
	for _, p := range existen {
		hay[p] = true
	}
	for p, n := range declaradas {
		if !hay[p] {
			t.Errorf("el corpus declara la primitiva %q en %d obligacion(es) y nucleo/ventana "+
				"no la implementa", p, n)
		}
	}

	// EL CENSO, con igualdad exacta en las dos direcciones.
	//
	// Los numeros salen del arbol; este mapa es lo que se espera encontrar. Si
	// SUBE, alguien ha encendido una primitiva en mas sitios, que es una buena
	// noticia y tiene que constar. Si BAJA, un paquete ha dejado de usarla y
	// puede estar quedandose sola.
	//
	// 10-09-2026: `periodica` baja de {134,16} a {132,15} y `plazo` de {98,19} a
	// {97,18}. NO ES QUE UNA PRIMITIVA HAYA PERDIDO TERRENO NORMATIVO: es que
	// `demo-empresa` salio de paquetes/ y se fue a demo/, y con el se van sus dos
	// `periodica` y su `plazo`, que eran sinteticos. El censo recorre solo
	// paquetes/ a proposito y se queda asi: lo que esta cifra tiene que medir es
	// cuanto DERECHO enciende cada primitiva, y un reloj inventado por nosotros
	// para una demo contaba de mas justo en la direccion que nos favorece.
	esperado := map[string][2]int{
		// primitiva: {obligaciones, paquetes}
		"periodica": {132, 15},
		"plazo":     {100, 18},
		"maximo":    {17, 2},
		"preaviso":  {8, 4},
		"continua":  {6, 4},
		"puntual":   {3, 1},
	}
	for p, q := range esperado {
		got := [2]int{declaradas[p], paquetes[p]}
		if got != q {
			t.Errorf("la primitiva %q la declaran %d obligaciones en %d paquetes, y se "+
				"esperaban %d en %d.\n"+
				"  Arreglo: actualiza el censo en el mismo commit que mueve el numero, y di\n"+
				"  en el cuerpo si la primitiva ha ganado o perdido terreno.",
				p, got[0], got[1], q[0], q[1])
		}
	}
	for p := range declaradas {
		if _, hay := esperado[p]; !hay {
			t.Errorf("el corpus declara la primitiva %q y el censo de este test no la cuenta: "+
				"una primitiva que no sale en el censo es una que nadie vigila", p)
		}
	}

	// LAS QUE NO EXISTEN, contadas y con su motivo.
	//
	// `secuencia` y `observacion` estan en el plan de las ocho y no estan
	// implementadas. NO son el caso de «uno y ninguno»: son trabajo sin
	// empezar, y confundir las dos cosas haria que un hueco de producto se
	// leyera como un motor sin probar.
	sinEmpezar := []string{"observacion", "secuencia"}
	for _, p := range sinEmpezar {
		if hay[p] {
			t.Errorf("la primitiva %q ya esta implementada y este test la sigue contando como "+
				"sin empezar. Sacala de la lista y metela en el censo", p)
		}
		if declaradas[p] != 0 {
			t.Errorf("el corpus declara %q y no esta implementada: eso no carga", p)
		}
	}
	if len(existen)+len(sinEmpezar) != 8 {
		t.Errorf("entre implementadas (%d) y sin empezar (%d) salen %d primitivas y el plan "+
			"declara ocho.\n"+
			"  Si ha entrado una nueva, entra tambien en este recuento. Si el plan ya no son\n"+
			"  ocho, este numero es el que hay que corregir, y en el commit se dice por que.",
			len(existen), len(sinEmpezar), len(existen)+len(sinEmpezar))
	}

	t.Logf("primitivas implementadas: %v", existen)
	for _, p := range existen {
		t.Logf("  %-10s %3d obligaciones en %2d paquetes", p, declaradas[p], paquetes[p])
	}
	t.Logf("sin empezar (no existen ni como tipo): %v", sinEmpezar)
}

// TestElCensoDePrimitivasSabePonerseRojo es el control negativo.
//
// La puerta de arriba NACE VERDE: las seis primitivas implementadas estan hoy
// encendidas. Sin este control no se distingue «ninguna esta apagada» de «no
// estoy mirando si lo estan», que es la diferencia entre vigilar y acompanar.
func TestElCensoDePrimitivasSabePonerseRojo(t *testing.T) {
	existen := []string{"puntual", "fantasma"}
	declaradas := map[string]int{"puntual": 3}
	var apagadas []string
	for _, p := range existen {
		if declaradas[p] == 0 {
			apagadas = append(apagadas, p)
		}
	}
	if len(apagadas) != 1 || apagadas[0] != "fantasma" {
		t.Errorf("el detector no ve una primitiva implementada y apagada: apagadas=%v.\n"+
			"  Con un detector asi, la puerta de arriba habria dejado pasar el caso `maximo`",
			apagadas)
	}
	// Y la direccion contraria: no acusa a una que si esta encendida.
	if declaradas["puntual"] == 0 {
		t.Error("el detector da por apagada una primitiva que el corpus declara")
	}
}

// primitivasImplementadas lee del AST de nucleo/ventana los nombres que devuelve
// cada `Nombre() string`.
//
// SE LEE DEL ARBOL Y NO SE ESCRIBE: una lista puesta aqui se quedaria vieja el
// dia que entre `secuencia`, y el sintoma seria justo el contrario del que hace
// falta: la primitiva nueva nace sin que nadie compruebe si alguien la enciende.
func primitivasImplementadas(t *testing.T) []string {
	t.Helper()
	fs := token.NewFileSet()
	paq, err := parser.ParseDir(fs, filepath.Join("nucleo", "ventana"), nil, 0)
	if err != nil {
		t.Fatalf("parseando nucleo/ventana: %v", err)
	}
	var out []string
	for _, p := range paq {
		for nombre, f := range p.Files {
			if len(nombre) > 8 && nombre[len(nombre)-8:] == "_test.go" {
				continue
			}
			for _, d := range f.Decls {
				fn, ok := d.(*ast.FuncDecl)
				if !ok || fn.Name.Name != "Nombre" || fn.Recv == nil || fn.Body == nil {
					continue
				}
				for _, s := range fn.Body.List {
					ret, ok := s.(*ast.ReturnStmt)
					if !ok || len(ret.Results) != 1 {
						continue
					}
					lit, ok := ret.Results[0].(*ast.BasicLit)
					if !ok || lit.Kind != token.STRING {
						continue
					}
					out = append(out, lit.Value[1:len(lit.Value)-1])
				}
			}
		}
	}
	sort.Strings(out)
	if len(out) < 5 {
		t.Fatalf("solo se han encontrado %d primitivas en nucleo/ventana (%v) y hoy son al "+
			"menos cinco: el AST no esta viendo los metodos Nombre()", len(out), out)
	}
	return out
}

// primitivasDeclaradasEnElCorpus cuenta, por primitiva, cuantas obligaciones la
// usan y en cuantos paquetes.
//
// Recorre `paquetes/` Y `esqueletos/`: un esqueleto no declara obligaciones hoy,
// pero mirar solo el publicado haria que encender una primitiva desde un paquete
// recien devuelto al escaparate no se notara en este censo.
func primitivasDeclaradasEnElCorpus(t *testing.T) (map[string]int, map[string]int) {
	t.Helper()
	obligaciones := map[string]int{}
	vistos := map[string]map[string]bool{}
	for _, raiz := range []string{"paquetes", DirDeEsqueletos} {
		ents, err := os.ReadDir(raiz)
		if err != nil {
			t.Fatalf("no puedo leer %s/: %v", raiz, err)
		}
		for _, e := range ents {
			if !e.IsDir() {
				continue
			}
			ruta := filepath.Join(raiz, e.Name(), "paquete.json")
			b, err := os.ReadFile(ruta) // #nosec G304 -- recorre el arbol del repositorio
			if err != nil {
				continue
			}
			var p struct {
				Obligaciones []struct {
					Temporalidad *struct {
						Primitiva string `json:"primitiva"`
					} `json:"temporalidad"`
				} `json:"obligaciones"`
			}
			if err := json.Unmarshal(b, &p); err != nil {
				t.Fatalf("%s no parsea: %v", ruta, err)
			}
			for _, o := range p.Obligaciones {
				if o.Temporalidad == nil || o.Temporalidad.Primitiva == "" {
					continue
				}
				n := o.Temporalidad.Primitiva
				obligaciones[n]++
				if vistos[n] == nil {
					vistos[n] = map[string]bool{}
				}
				vistos[n][e.Name()] = true
			}
		}
	}
	paquetes := map[string]int{}
	for n, s := range vistos {
		paquetes[n] = len(s)
	}
	if len(obligaciones) == 0 {
		t.Fatal("ninguna obligacion del corpus declara primitiva: este recorrido esta " +
			"midiendo el vacio")
	}
	return obligaciones, paquetes
}
