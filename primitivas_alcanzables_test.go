package plazum

import (
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
	"unicode"

	"github.com/marcosmatalab/plazum/nucleo/corpus"
	"github.com/marcosmatalab/plazum/nucleo/ventana"
)

// EL TRINQUETE DE ALCANZABILIDAD, MITAD DE LAS PRIMITIVAS.
//
// LA FAMILIA QUE LO TRAE, tres casos en cuatro dias y ninguna puerta roja:
// `maximo` encendida en el motor y apagada para el corpus, con ocho retenciones
// del CRA esperandola; la pantalla del acta entera y sin montar; y el camino
// guiado con el enlace puesto y vacio. Ninguno era un despiste: los tres son
// piezas terminadas sin el cable, y ninguna puerta los vio porque cada mitad
// pasaba la suya. Habia puertas que verifican piezas y ninguna que verificara
// juntas.
//
// LO QUE ESTA PUERTA AFIRMA, en una frase: por cada primitiva del motor, o hay
// un paquete que la usa, o hay una linea escrita que dice por que esta apagada
// y cuantos relojes la esperan.
//
// LOS TRES CONJUNTOS SE ENUMERAN DEL ARBOL, no de una lista escrita al lado:
//
//	el motor        del AST de nucleo/ventana: todo tipo con `Nombre() string`
//	                que devuelve un literal implementa ventana.Primitiva;
//	el corpus       PREGUNTANDOSELO a VencimientosDe con el centinela
//	                ErrPrimitivaSinEjecutor, o sea al codigo que de verdad
//	                traduce, no a una copia de su switch escrita aqui;
//	los paquetes    leyendo los paquete.json de paquetes/.
//
// Una lista escrita aqui seria la segunda copia, y se desincronizaria justo el
// dia que alguien anadiera la primitiva octava, que es el dia en que hace falta.

// instanteDeLaSonda es el `hasta` que se le pasa al traductor. Da igual cual
// sea: la sonda solo mira si vuelve el centinela, no que se calcule nada.
var instanteDeLaSonda = time.Date(2026, 9, 2, 0, 0, 0, 0, time.UTC)

// primitivasDelMotor lee del AST de nucleo/ventana los nombres que devuelven los
// implementadores de ventana.Primitiva, con el tipo que los declara.
//
// Se lee el AST y no se usa reflexion porque no hay ningun sitio donde esten
// listadas: la unica forma de enumerarlas sin escribirlas otra vez es mirar el
// codigo. El patron es estrecho a proposito (`func (T) Nombre() string { return
// "x" }`), asi que si alguien escribe una con el cuerpo de otra forma esta
// funcion devolvera de menos: por eso el test exige ademas un minimo contado.
func primitivasDelMotor(t *testing.T) map[string]string {
	t.Helper()
	out := map[string]string{}
	fset := token.NewFileSet()
	raiz := filepath.Join("nucleo", "ventana")
	err := filepath.WalkDir(raiz, func(ruta string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(ruta, ".go") || strings.HasSuffix(ruta, "_test.go") {
			return nil
		}
		f, err := parser.ParseFile(fset, ruta, nil, 0)
		if err != nil {
			return fmt.Errorf("%s: %w", ruta, err)
		}
		for _, decl := range f.Decls {
			fd, ok := decl.(*ast.FuncDecl)
			if !ok || fd.Name.Name != "Nombre" || fd.Recv == nil || len(fd.Recv.List) != 1 {
				continue
			}
			if fd.Type.Results == nil || len(fd.Type.Results.List) != 1 {
				continue
			}
			if id, ok := fd.Type.Results.List[0].Type.(*ast.Ident); !ok || id.Name != "string" {
				continue
			}
			nombre, tipo := literalDevuelto(fd), tipoDelReceptor(fd)
			if nombre == "" || tipo == "" {
				continue
			}
			out[nombre] = tipo
		}
		return nil
	})
	if err != nil {
		t.Fatalf("recorriendo %s: %v", raiz, err)
	}
	return out
}

func literalDevuelto(fd *ast.FuncDecl) string {
	if fd.Body == nil || len(fd.Body.List) != 1 {
		return ""
	}
	ret, ok := fd.Body.List[0].(*ast.ReturnStmt)
	if !ok || len(ret.Results) != 1 {
		return ""
	}
	lit, ok := ret.Results[0].(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return ""
	}
	return strings.Trim(lit.Value, string('"'))
}

func tipoDelReceptor(fd *ast.FuncDecl) string {
	switch tipo := fd.Recv.List[0].Type.(type) {
	case *ast.Ident:
		return tipo.Name
	case *ast.StarExpr:
		if id, ok := tipo.X.(*ast.Ident); ok {
			return id.Name
		}
	}
	return ""
}

// elCorpusSabeConstruir pregunta al TRADUCTOR DE VERDAD si un paquete podria
// declarar esta primitiva, en vez de mirar una copia de su switch.
//
// Se le da una obligacion minima y se mira SOLO si vuelve el centinela. Que
// falle por cualquier otra razon (le falta el `en`, le falta la cadencia, le
// falta el disparador) significa que la rama existe y esta pidiendo sus campos,
// que es exactamente lo que se queria saber: la primitiva esta cableada.
func elCorpusSabeConstruir(primitiva string) bool {
	_, err := corpus.VencimientosDe(corpus.Obligacion{
		ID:           "sonda." + primitiva,
		Temporalidad: &corpus.Temporalidad{Primitiva: primitiva},
	}, ventana.Hechos{}, instanteDeLaSonda)
	return !errors.Is(err, corpus.ErrPrimitivaSinEjecutor)
}

// primitivasEnLosPaquetes cuenta que declara el corpus publicado, leyendo los
// paquete.json. Es el dato real: no lo que se puede escribir, lo que se escribio.
func primitivasEnLosPaquetes(t *testing.T) map[string]int {
	t.Helper()
	out := map[string]int{}
	err := filepath.WalkDir("paquetes", func(ruta string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || d.Name() != "paquete.json" {
			return err
		}
		b, err := os.ReadFile(ruta) // #nosec G304 -- test: recorre el arbol del propio repositorio
		if err != nil {
			return err
		}
		var p struct {
			Obligaciones []struct {
				Temporalidad *struct {
					Primitiva string `json:"primitiva"`
				} `json:"temporalidad"`
			} `json:"obligaciones"`
		}
		if err := json.Unmarshal(b, &p); err != nil {
			return fmt.Errorf("%s: %w", ruta, err)
		}
		for _, o := range p.Obligaciones {
			if o.Temporalidad != nil && o.Temporalidad.Primitiva != "" {
				out[o.Temporalidad.Primitiva]++
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("recorriendo paquetes/: %v", err)
	}
	return out
}

// TestTodaPrimitivaDelMotorOSeUsaOSeExplica es la puerta.
//
// NACIO ROJA SOBRE EL ARBOL REAL, que es lo que se queria: `observacion` no la
// puede declarar ningun paquete y ningun reloj contado la pide, o sea que
// cumplia entera la condicion por la que se borro `Secuencia` cuatro dias antes,
// sin que nadie lo hubiera notado. Una mutacion no habria encontrado eso, porque
// nadie lo habia metido.
func TestTodaPrimitivaDelMotorOSeUsaOSeExplica(t *testing.T) {
	motor := primitivasDelMotor(t)
	// El minimo contado protege del fallo silencioso de esta misma puerta: si
	// el patron del AST deja de casar, el mapa sale vacio y todo lo de abajo se
	// pone verde recorriendo la nada.
	// EL SUELO BAJO DE 7 A 6 el 02-09-2026, al borrar `Observacion`, Y ESE ES EL
	// COMPORTAMIENTO QUE SE QUERIA: un borrado legitimo tiene que poner roja
	// esta puerta y obligar a bajar el numero en el MISMO commit. Es la
	// disciplina de PUERTAS_ESPERADAS: la unica direccion que nadie mira es la
	// de que un conjunto mengue, y aqui menguar es justo lo que hay que notar.
	if len(motor) < 6 {
		t.Fatalf("del AST de nucleo/ventana salen %d primitivas (%v) y hoy hay al menos 6. "+
			"O han desaparecido, o el patron de esta puerta ha dejado de casar y estaria "+
			"midiendo el vacio", len(motor), clavesOrdenadas(motor))
	}
	enPaquetes := primitivasEnLosPaquetes(t)

	// SENTIDO 1: toda primitiva del motor tiene que estar declarada.
	for nombre, tipo := range motor {
		d, declarada := corpus.PrimitivasDelCorpus[nombre]
		if !declarada {
			t.Errorf("ventana.%s se llama %q y NO sale en corpus.PrimitivasDelCorpus.\n"+
				"  El silencio es justo el estado que produjo `maximo` (construida, probada "+
				"y apagada para el corpus, con ocho relojes esperando). Arreglo: declarala "+
				"ahi como en uso, apagada o sin cablear, con su motivo y su cardinal.",
				tipo, nombre)
			continue
		}
		if d.Estado == corpus.PrimitivaSinDeclarar {
			t.Errorf("la primitiva %q esta en el censo con el VALOR CERO (%s). El cero no es "+
				"un estado, es el olvido: escribe cual de los tres es", nombre, d.Estado)
		}
	}

	// SENTIDO 2: toda entrada del censo tiene que existir en el motor. Sin esta
	// mitad, el censo se convierte en una lista de primitivas que hubo.
	for nombre := range corpus.PrimitivasDelCorpus {
		if _, hay := motor[nombre]; !hay {
			t.Errorf("el censo declara la primitiva %q y el motor no la tiene. O se borro y "+
				"nadie limpio el censo, o el nombre no coincide con el que devuelve "+
				"Nombre()", nombre)
		}
	}

	// SENTIDO 3, EL QUE IMPORTA: el estado declarado tiene que ser el que el
	// arbol ensena. Aqui es donde una pieza sin cable se pone roja.
	for _, nombre := range clavesDelCenso() {
		d := corpus.PrimitivasDelCorpus[nombre]
		if _, hay := motor[nombre]; !hay {
			continue // ya se dijo en el sentido 2
		}
		usos := enPaquetes[nombre]
		cableada := elCorpusSabeConstruir(nombre)
		real := corpus.PrimitivaSinCablear
		switch {
		case usos > 0:
			real = corpus.PrimitivaEnUso
		case cableada:
			real = corpus.PrimitivaApagada
		}
		if real != d.Estado {
			t.Errorf("la primitiva %q se declara %q y el arbol dice %q "+
				"(paquetes que la usan: %d; VencimientosDe la construye: %t).\n"+
				"  Arreglo: si el arbol tiene razon, corrige el censo; si el censo tiene "+
				"razon, falta el cable.", nombre, d.Estado, real, usos, cableada)
		}
		// UNA PRIMITIVA QUE NO ESTA EN USO TIENE QUE DECIR POR QUE, Y CUANTO
		// CUESTA. Sin motivo, un hueco es indistinguible de un olvido a los
		// tres meses; sin cardinal contrastable, no se le puede poner techo ni
		// enterarse de si crece.
		if d.Estado != corpus.PrimitivaEnUso {
			if strings.TrimSpace(d.Motivo) == "" {
				t.Errorf("la primitiva %q esta %q y no dice por que. Un hueco sin motivo se "+
					"lee como deuda y como decision a la vez", nombre, d.Estado)
			}
			if strings.TrimSpace(d.DondeSeCuentan) == "" {
				t.Errorf("la primitiva %q esta %q y no dice de donde sale su cardinal (%d). "+
					"Un numero que nadie puede recontar envejece sin que se note",
					nombre, d.Estado, d.RelojesEsperando)
			}
		}
		// Y UNA QUE SI ESTA EN USO NO NECESITA MOTIVO: si lo trae, es que el
		// censo se quedo viejo y afirma una deuda que ya se pago.
		if d.Estado == corpus.PrimitivaEnUso && strings.TrimSpace(d.Motivo) != "" {
			t.Errorf("la primitiva %q esta en uso (%d paquetes) y arrastra un motivo de "+
				"cuando no lo estaba: %q", nombre, usos, d.Motivo)
		}
	}
}

// CONTROL NEGATIVO DE LA SONDA, en las dos direcciones.
//
// Todo lo de arriba cuelga de elCorpusSabeConstruir, y esa funcion tiene dos
// formas de mentir en silencio: decir que si a todo (y entonces ninguna
// primitiva saldria nunca «sin cablear», que es justo el hallazgo que esta
// puerta existe para dar) o decir que no a todo (y entonces saltarian a la vez
// todas las declaradas en uso, que se lee como un fallo del censo y no de la
// sonda). Se recorren las dos ramas con dato real.
func TestLaSondaDePrimitivasDistingueLasDosRespuestas(t *testing.T) {
	// Un nombre que el motor no tiene NUNCA puede estar cableado.
	if elCorpusSabeConstruir("primitiva-que-no-existe-jamas") {
		t.Error("la sonda dice que el corpus sabe construir una primitiva inventada: " +
			"entonces no esta mirando el centinela, y ninguna primitiva saldra sin cablear")
	}
	// Y una que si esta cableada tiene que dar que si. Se eligen del censo, no
	// se escriben aqui: el dia que `periodica` se renombre, esto sigue midiendo.
	probadas := 0
	for _, nombre := range clavesDelCenso() {
		if corpus.PrimitivasDelCorpus[nombre].Estado != corpus.PrimitivaEnUso {
			continue
		}
		probadas++
		if !elCorpusSabeConstruir(nombre) {
			t.Errorf("la sonda dice que el corpus NO sabe construir %q, y hay paquetes "+
				"publicados que la declaran. Entonces la rota es la sonda, no el censo",
				nombre)
		}
	}
	if probadas < 3 {
		t.Fatalf("solo se han probado %d primitivas en uso y este control necesita varias "+
			"para significar algo", probadas)
	}
}

func clavesOrdenadas(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// clavesDelCenso da los nombres en orden estable. Recorrer un mapa da un orden
// distinto en cada ejecucion, y una puerta cuyo informe cambia de orden es una
// puerta que nadie compara entre dos ejecuciones.
func clavesDelCenso() []string {
	out := make([]string, 0, len(corpus.PrimitivasDelCorpus))
	for k := range corpus.PrimitivasDelCorpus {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// LA PROSA ES LA MITAD QUE CADUCA: ningun motivo escribe su propio cardinal.
//
// # El fallo que la trae, y es de esta misma casa
//
// El 03-09-2026 el cardinal de `preaviso` subio de 7 a 8 y su `Motivo` siguio
// diciendo que «los siete» estaban FUERA de los 12 marcos de la v1. El octavo
// esta DENTRO (art. 60.4.f del AI Act), o sea que la deuda habia pasado a
// bloquear la v1 y el fichero afirmaba lo contrario. Nada se puso rojo: el
// cardinal tenia puerta (`RelojesEsperando`, contrastado contra el arbol) y la
// explicacion no tenia ninguna.
//
// Es la cuarta aparicion de la familia de la AFIRMACION ACOMPANADA y la primera
// en la que quien miente es la explicacion y no el dato, que es peor: un dato
// falso se contrasta en un minuto, una explicacion falsa se cree.
//
// # Por que se prohiben las PALABRAS y no los digitos
//
// Un motivo tiene que poder citar una fecha (`02-09-2026`) y un articulo
// (`60.4.f`), que son digitos y no son cardinales de nada. Lo que envejece es la
// prosa que RECUENTA: «los siete», «las dos primeras», «el octavo». Asi que la
// lista prohibida son los numerales escritos con letra, cardinales y ordinales,
// y el numero se compone con Explicacion(), que lo deriva del campo que la
// puerta vigila.
//
// Un motivo que de verdad necesite decir «siete» tiene la salida abierta y es la
// correcta: mover ese numero a un campo, que es donde una puerta puede verlo.

// numeralesEnLetra son los numerales que un motivo NO puede escribir. Cardinales
// y ordinales bajos, que es lo que aparece en un recuento en prosa. La lista es
// corta a proposito: cazar «diecisiete» importa menos que no dar falsos
// positivos sobre palabras que no son numeros.
//
// Y FALTAN TRES A PROPOSITO: `uno`, `una` y `primera`. En castellano son mucho
// mas a menudo articulo o pronombre que recuento («una fecha que elige el
// obligado», «la primera vez»), asi que incluirlos hace que la puerta acuse a
// prosa correcta. La puerta nacio ROJA por esto: `una fecha` la disparo contra
// el motivo real de `preaviso`, que estaba bien escrito. Una puerta que acusa en
// falso se acaba borrando, y entonces no vigila nada; se prefiere dejar pasar un
// «un octavo» a cambio de que el resto siga vigilando.
var numeralesEnLetra = []string{
	"cero", "dos", "tres", "cuatro", "cinco", "seis", "siete",
	"ocho", "nueve", "diez", "once", "doce", "trece", "catorce", "quince",
	"dieciseis", "veinte", "treinta", "cuarenta", "cincuenta", "cien",
	"segundo", "segunda", "tercero", "tercera", "cuarto",
	"quinto", "sexto", "septimo", "octavo", "noveno", "decimo",
}

// escribeUnNumeralEnLetra devuelve el numeral que encuentra, o "" si no hay
// ninguno. Se compara por PALABRA ENTERA: sin eso, "uno" casaria dentro de
// "alguno" y "una" dentro de "ninguna", y la puerta se pasaria el dia acusando
// a prosa correcta, que es como se acaba desactivando una puerta.
func escribeUnNumeralEnLetra(texto string) string {
	campos := strings.FieldsFunc(strings.ToLower(texto), func(r rune) bool {
		return !unicode.IsLetter(r)
	})
	for _, c := range campos {
		for _, n := range numeralesEnLetra {
			if c == n {
				return c
			}
		}
	}
	return ""
}

func TestNingunMotivoDeUnaPrimitivaEscribeSuCardinal(t *testing.T) {
	declaradas := 0
	for nombre, d := range corpus.PrimitivasDelCorpus {
		if d.Estado == corpus.PrimitivaEnUso {
			continue
		}
		declaradas++
		if n := escribeUnNumeralEnLetra(d.Motivo); n != "" {
			t.Errorf("el motivo de la primitiva %q escribe el numeral %q a mano.\n"+
				"  Un motivo que recuenta en prosa es la mitad que CADUCA: el cardinal "+
				"(%d) tiene puerta y la frase no, asi que el dia que el numero cambie la "+
				"explicacion se queda afirmando lo viejo con cara de decision tomada. "+
				"Paso el 03-09-2026 con esta misma primitiva.\n"+
				"  Arreglo: quitar el numeral de la frase y dejar que Explicacion() lo "+
				"componga de RelojesEsperando, o mover ese recuento a un campo propio, "+
				"que es donde una puerta puede verlo.", nombre, n, d.RelojesEsperando)
		}
		// Y EL CARDINAL SI SALE, por Explicacion(). Sin esta rama, quitar el
		// numero de la prosa dejaria la explicacion MUDA, que es peor que vieja:
		// una deuda sin cardinal se olvida.
		if !strings.Contains(d.Explicacion(), strconv.Itoa(d.RelojesEsperando)) {
			t.Errorf("Explicacion() de %q no dice cuantos relojes la esperan (%d). "+
				"Prohibir el numeral en la prosa sin componerlo aqui deja la deuda sin "+
				"cardinal, y un hueco sin numero se olvida", nombre, d.RelojesEsperando)
		}
	}
	if declaradas == 0 {
		t.Fatal("ninguna primitiva esta apagada o sin cablear, asi que este recorrido no ha " +
			"mirado ni un motivo: es un verde vacio. O se han encendido todas, y entonces " +
			"esta puerta cambia de forma, o el censo no se esta leyendo")
	}
	t.Logf("%d primitivas con motivo, ninguna con su cardinal escrito a mano", declaradas)
}

// CONTROL NEGATIVO EN LAS DOS DIRECCIONES. La puerta tiene que cazar el recuento
// en prosa Y tiene que dejar pasar fechas, articulos y palabras que contienen un
// numeral dentro. La segunda mitad es la que decide si la puerta sobrevive: una
// que acusa a prosa correcta se acaba borrando.
func TestElDetectorDeNumeralesDistingueUnRecuentoDeUnaFecha(t *testing.T) {
	casos := []struct {
		nombre string
		texto  string
		caza   string
	}{
		{"el recuento en prosa, que es el fallo real",
			"los siete relojes que la esperan caen fuera de los 12 marcos", "siete"},
		{"el ordinal, que es la otra forma del mismo fallo",
			"hay un OCTAVO y esta dentro de la v1", "octavo"},
		{"una fecha no es un recuento",
			"cableada el 02-09-2026, rama en VencimientosDe", ""},
		{"un articulo tampoco",
			"el art. 60.4.f del AI Act, la prorroga de las pruebas", ""},
		{"y una palabra que lleva un numeral dentro tampoco",
			"ninguna obligacion la declara todavia, y alguno la necesitara", ""},
		// LOS TRES EXCLUIDOS, con su caso, para que la exclusion sea una
		// decision escrita y no un olvido: si manana alguien los mete en la
		// lista, este caso se pone rojo y le cuenta por que estaban fuera.
		{"un articulo indefinido no es un recuento, y es lo que puso roja la puerta",
			"un plazo que corre hacia atras desde una fecha que ELIGE el obligado", ""},
		{"ni un ordinal que ordena en vez de contar",
			"la primera vez que un paquete la declare, esto cambia de estado", ""},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			got := escribeUnNumeralEnLetra(c.texto)
			if got != c.caza {
				t.Errorf("ha cazado %q y esperaba %q en %q", got, c.caza, c.texto)
			}
		})
	}
}
