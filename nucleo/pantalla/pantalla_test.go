package pantalla

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"dutiq/nucleo/corpus"
)

// El dorado se compara byte a byte, asi que hay que poder regenerarlo sin
// copiarlo a mano de la salida de un test fallido:
//
//	go test ./nucleo/pantalla -actualizar
//
// Y despues se MIRA EL DIFF. Un dorado que se regenera sin leer el diff es un
// fichero que va detras del codigo en vez de vigilarlo.
var actualizar = flag.Bool("actualizar", false,
	"reescribe testdata/dorado.json con el modelo derivado y no compara")

// Los casos dorados de la derivacion.
//
// Que es un dorado aqui: un fichero de entrada (paquetes) y un fichero de
// salida (el modelo), comparados byte a byte. Se comparan asi y no campo a
// campo a proposito. Un test que comprueba "hay 3 filas" pasa aunque las filas
// sean las equivocadas; un fichero completo no deja sitio donde esconderse, y
// ademas el diff de git enensa exactamente que cambio en la interfaz cuando se
// toca un paquete. Es la unica forma de que "anadir una norma cambia la UI
// sola" sea revisable en una revision de codigo.
//
// Los paquetes de entrada son SINTETICOS (urn:demo:...). No pueden ser normas
// reales: el invariante 2 del proyecto prohibe identificadores de norma en el
// codigo, ficheros de test incluidos.

func cargarEntrada(t *testing.T, nombre string) []*corpus.Paquete {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", nombre))
	if err != nil {
		t.Fatalf("no puedo leer la entrada del dorado: %v", err)
	}
	var ps []*corpus.Paquete
	if err := json.Unmarshal(b, &ps); err != nil {
		t.Fatalf("la entrada del dorado no es un JSON de paquetes: %v", err)
	}
	if len(ps) == 0 {
		t.Fatal("la entrada del dorado esta vacia: el dorado no probaria nada")
	}
	return ps
}

func serializar(t *testing.T, ps []Pantalla) []byte {
	t.Helper()
	b, err := json.MarshalIndent(ps, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	return append(b, '\n')
}

func TestElDoradoDeLaDerivacionSeCumple(t *testing.T) {
	got := serializar(t, Derivar(cargarEntrada(t, "entrada.json")))
	ruta := filepath.Join("testdata", "dorado.json")
	if *actualizar {
		if err := os.WriteFile(ruta, got, 0o644); err != nil {
			t.Fatalf("no puedo reescribir el dorado: %v", err)
		}
		t.Log("dorado reescrito. Mira el diff: tiene que ensenar exactamente lo que " +
			"esperabas cambiar en la interfaz, y nada mas")
		return
	}
	esperado, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatalf("no puedo leer el dorado: %v", err)
	}
	if string(got) != strings.ReplaceAll(string(esperado), "\r\n", "\n") {
		t.Errorf("el modelo derivado ya no coincide con el dorado.\n"+
			"Si el cambio es intencionado, regeneralo con go test ./nucleo/pantalla "+
			"-actualizar, MIRA EL DIFF, y explica en el commit que cambio en la "+
			"interfaz y por que.\n--- derivado ---\n%s", got)
	}
}

// Control negativo del dorado. Un dorado que pasa pase lo que pase no prueba
// nada, y la forma de que eso ocurra es trivial: que Derivar devuelva algo que
// no dependa de la entrada. Aqui se anade un paquete a la entrada y se exige
// que el modelo CAMBIE.
func TestElDoradoSaltaCuandoElCorpusCambia(t *testing.T) {
	base := cargarEntrada(t, "entrada.json")
	antes := serializar(t, Derivar(base))

	extra := &corpus.Paquete{
		URN: "urn:demo:zeta", Version: "1", Clase: corpus.Propio,
		Entidades: []corpus.TipoEntidad{{Nombre: "sistema", Atributos: []corpus.Atributo{
			{Nombre: "zeta_nueva", Tipo: corpus.Texto, Cita: "demo zeta art. 1"},
		}}},
		Obligaciones: []corpus.Obligacion{{
			ID: "zeta.1", Articulo: "1", Cita: "demo zeta art. 1", ClaseE2E: "documental",
		}},
	}
	despues := serializar(t, Derivar(append(base, extra)))

	if string(antes) == string(despues) {
		t.Fatal("anadir un paquete no ha cambiado el modelo. O la derivacion ignora la " +
			"entrada, o el dorado no prueba nada. Las dos cosas son el mismo fallo")
	}
}

// Determinismo, que es la otra mitad de que un dorado valga: si el modelo
// depende del orden en que el cargador recorrio el directorio, el dorado se
// pone rojo un dia sin que nadie haya tocado nada, y a partir de ahi se ignora.
func TestElModeloNoDependeDelOrdenDeLosPaquetes(t *testing.T) {
	ps := cargarEntrada(t, "entrada.json")
	if len(ps) < 2 {
		t.Fatal("hacen falta al menos dos paquetes para que este test pruebe algo")
	}
	directo := serializar(t, Derivar(ps))

	alreves := make([]*corpus.Paquete, len(ps))
	for i, p := range ps {
		alreves[len(ps)-1-i] = p
	}
	if string(serializar(t, Derivar(alreves))) != string(directo) {
		t.Error("el modelo cambia si los paquetes llegan en otro orden. El cargador recorre " +
			"un directorio, asi que ese orden no esta garantizado y el modelo seria inestable")
	}
}

// El comprador pregunta "por que me piden este dato" y hasta ahora se le
// respondia con el articulo de UNA de las normas que lo piden, la de URN menor.
// El dorado ya lo compara byte a byte, pero esto lo dice con nombres: si manana
// alguien colapsa las peticiones a una, aqui se lee por que estaba mal.
func TestCadaCampoDiceQueArticuloPoneCadaNormaQueLoPide(t *testing.T) {
	ps := Derivar(cargarEntrada(t, "entrada.json"))
	var compartido *Campo
	for i, c := range ps[0].Campos {
		if len(c.Paquetes) > 1 {
			compartido = &ps[0].Campos[i]
			break
		}
	}
	if compartido == nil {
		t.Fatal("la entrada del dorado tiene que traer un dato pedido por dos normas, o " +
			"este test no prueba nada")
	}
	if len(compartido.Peticiones) != len(compartido.Paquetes) {
		t.Fatalf("%s.%s lo piden %d normas y solo se sabe por que lo piden %d: %+v",
			compartido.Entidad, compartido.Atributo, len(compartido.Paquetes),
			len(compartido.Peticiones), compartido.Peticiones)
	}
	// Por pertenencia y no por indice: Paquetes cuenta una vez por declaracion
	// y Peticiones una vez por paquete, asi que emparejarlos por posicion es
	// una suposicion que se rompe con un paquete que se repita.
	quien := map[string]bool{}
	for _, u := range compartido.Paquetes {
		quien[u] = true
	}
	citas := map[string]bool{}
	for _, x := range compartido.Peticiones {
		if !quien[x.Paquete] {
			t.Errorf("la peticion viene de %q, que no esta entre los que piden el dato (%v)",
				x.Paquete, compartido.Paquetes)
		}
		if x.Cita == "" {
			t.Errorf("la peticion de %q no trae cita: sin articulo es una opinion", x.Paquete)
		}
		citas[x.Cita] = true
	}
	if len(citas) < 2 {
		t.Errorf("las dos normas dan la misma cita (%v), asi que el caso no distingue "+
			"si se pierde una", citas)
	}
}

// La fila de Controles ensena un titulo legible SIEMPRE, tambien cuando el
// paquete todavia no declara el campo titulo, que es el caso de los 30 paquetes
// del corpus de hoy.
func TestLaFilaDeControlesTraeTituloAunqueElPaqueteNoLoDeclare(t *testing.T) {
	ps := cargarEntrada(t, "entrada.json")
	for _, o := range ps[0].Obligaciones {
		if o.Titulo != "" {
			t.Fatal("la entrada del dorado no puede declarar titulos, o este test no " +
				"prueba el respaldo")
		}
	}
	controles := Derivar(ps)[2]
	if len(controles.Filas) == 0 {
		t.Fatal("sin filas no hay nada que comprobar")
	}
	for _, f := range controles.Filas {
		if f.Columnas["titulo"] == "" {
			t.Errorf("la fila %s no trae titulo: la tabla de controles ensenaria una "+
				"celda en blanco", f.ID)
		}
	}
}

// Las seis pantallas existen siempre, tambien sin corpus, y cada una dice por
// que esta vacia. Una pantalla que desaparece del menu porque no hay datos deja
// al operador sin saber que existia.
func TestLasSeisPantallasExistenSinCorpusYExplicanPorQueEstanVacias(t *testing.T) {
	ps := Derivar(nil)
	quiero := []ID{Alcance, Hoy, Controles, Certificados, Personas, Estado}
	if len(ps) != len(quiero) {
		t.Fatalf("se esperaban %d pantallas y hay %d", len(quiero), len(ps))
	}
	for i, id := range quiero {
		if ps[i].ID != id {
			t.Errorf("la pantalla %d es %q y se esperaba %q", i, ps[i].ID, id)
		}
		if !ps[i].Vacia {
			t.Errorf("sin corpus ni estado, la pantalla %q no puede venir con contenido", id)
		}
		if ps[i].PorQue == "" {
			t.Errorf("la pantalla %q esta vacia y no dice por que. Una pantalla en blanco sin "+
				"explicacion es la forma mas rapida de perder a quien acaba de instalar esto", id)
		}
	}
}

// El rotulo de la interfaz es una CLAVE de catalogo, no texto. Si alguien mete
// texto aqui, el producto deja de ser traducible y ademas se cuela texto de
// interfaz en un sitio donde no lo puede ver el linter de catalogos.
func TestLosTitulosSonClavesDeCatalogoNoTexto(t *testing.T) {
	for _, p := range Derivar(nil) {
		if !strings.HasPrefix(p.Titulo, "pantalla.") {
			t.Errorf("el titulo de %q es %q, que no parece una clave de catalogo. "+
				"Los rotulos de la interfaz los pone el puerto Catalogo, aqui van claves", p.ID, p.Titulo)
		}
		if strings.ContainsAny(p.Titulo, " áéíóúñ") {
			t.Errorf("el titulo de %q (%q) es texto, no una clave", p.ID, p.Titulo)
		}
	}
}
