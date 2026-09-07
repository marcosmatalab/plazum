package manual_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/marcosmatalab/plazum/adaptadores/recoleccion/manual"
	"github.com/marcosmatalab/plazum/nucleo/corpus"
	"github.com/marcosmatalab/plazum/nucleo/estado"
)

func conFichero(t *testing.T, contenido string) *manual.Recolector {
	t.Helper()
	dir := t.TempDir()
	ruta := filepath.Join(dir, "observaciones.jsonl")
	if err := os.WriteFile(ruta, []byte(contenido), 0o600); err != nil {
		t.Fatal(err)
	}
	r, err := manual.Abrir(manual.Opciones{Ruta: ruta})
	if err != nil {
		t.Fatal(err)
	}
	return r
}

const cabecera = `{"version":1,"sistema":"directorio corporativo","quien":"ana@ejemplo"}`

const bueno = cabecera + `
# el 1 de marzo se exporto el directorio y se comprobaron los privilegios
{"prueba":"p.privilegios","recurso":"srv-01","satisfecho":true,"cuando":"2026-03-01T09:00:00Z"}

{"prueba":"p.privilegios","recurso":"srv-02","satisfecho":false,"cuando":"2026-03-01T09:00:00Z"}
{"prueba":"p.privilegios","recurso":"srv-03","error":"la API contesto 429","cuando":"2026-03-01T09:00:00Z"}
`

// ---------------------------------------------------------------------------
// LO QUE SI SE LEE
// ---------------------------------------------------------------------------

func TestUnFicheroBuenoDaSusObservacionesConSuProcedencia(t *testing.T) {
	r := conFichero(t, bueno)
	obs, siguiente, err := r.Recolectar(manual.Fuente, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(obs) != 3 {
		t.Fatalf("%d observaciones, esperaba 3: %+v", len(obs), obs)
	}
	// EL CURSOR DE SALIDA ES SIEMPRE VACIO. Devolver cualquier otra cosa haria
	// que un llamante correcto entrara en un bucle infinito.
	if siguiente != "" {
		t.Errorf("cursor de salida %q, y este recolector lee el fichero entero", siguiente)
	}
	if obs[0].Recurso != "srv-01" || !obs[0].Satisfecho {
		t.Errorf("primera observacion: %+v", obs[0])
	}
	if obs[1].Satisfecho {
		t.Error("la segunda dice satisfecho:false y ha salido true")
	}
	// EL ERROR DE RECOLECCION VIAJA, y no se convierte ni en un fallo ni en un
	// pase: es la tercera forma de la nada de la recoleccion (A5).
	if obs[2].ErrorRecol == "" {
		t.Error("la tercera trae un error de recoleccion y no ha viajado")
	}
	// LA PROCEDENCIA, que es una de las tres cosas que la pantalla ensena.
	for i, o := range obs {
		if o.Recolector != manual.Nombre {
			t.Errorf("observacion %d sin recolector: %q", i, o.Recolector)
		}
		if o.Version == "" {
			t.Errorf("observacion %d sin version del recolector", i)
		}
		if o.Recolectada.IsZero() {
			t.Errorf("observacion %d sin instante", i)
		}
	}
	if r.Sello() == "" {
		t.Error("no se ha sellado el fichero leido")
	}
}

func TestLaLeyDeConservacionCuadraYSePuedeContar(t *testing.T) {
	r := conFichero(t, bueno)
	rec, err := r.Contar()
	if err != nil {
		t.Fatal(err)
	}
	if !rec.Cuadra() {
		t.Fatalf("no cuadra: %s", rec)
	}
	if rec.Leidas != 3 || rec.Cabecera != 1 || rec.Comentario != 1 || rec.EnBlanco != 1 {
		t.Errorf("reparto: %s", rec)
	}
	if rec.Total != rec.Leidas+rec.Cabecera+rec.Comentario+rec.EnBlanco {
		t.Errorf("el total no es la suma: %s", rec)
	}
}

// TestElSelloCambiaSiCambiaElFichero: sin esto, el sello seria un adorno.
func TestElSelloCambiaSiCambiaElFichero(t *testing.T) {
	a := conFichero(t, bueno)
	if _, _, err := a.Recolectar(manual.Fuente, ""); err != nil {
		t.Fatal(err)
	}
	b := conFichero(t, bueno+
		`{"prueba":"p.privilegios","recurso":"srv-04","satisfecho":true,"cuando":"2026-03-01T09:00:00Z"}`+"\n")
	if _, _, err := b.Recolectar(manual.Fuente, ""); err != nil {
		t.Fatal(err)
	}
	if a.Sello() == b.Sello() {
		t.Error("dos ficheros distintos dan el mismo sello")
	}
	// Y EL MISMO FICHERO DA EL MISMO SELLO, que es la otra mitad: un sello que
	// cambiara sin que cambie nada no serviria para anotar en el ledger que
	// fichero se leyo.
	c := conFichero(t, bueno)
	if _, _, err := c.Recolectar(manual.Fuente, ""); err != nil {
		t.Fatal(err)
	}
	if a.Sello() != c.Sello() {
		t.Error("el mismo fichero da dos sellos distintos")
	}
}

// ---------------------------------------------------------------------------
// LAS TRES FORMAS DE LA NADA (invariante 8)
// ---------------------------------------------------------------------------

func TestSinFicheroYConFicheroVacioNoSonUnError(t *testing.T) {
	// AUSENTE: nadie ha recolectado todavia. Es el caso de una instalacion
	// recien descargada y es un DATO.
	dir := t.TempDir()
	r, err := manual.Abrir(manual.Opciones{Ruta: manual.RutaPorDefecto(dir)})
	if err != nil {
		t.Fatal(err)
	}
	obs, _, err := r.Recolectar(manual.Fuente, "")
	if err != nil {
		t.Fatalf("un fichero ausente no es un error, y dio: %v", err)
	}
	if len(obs) != 0 {
		t.Errorf("%d observaciones de un fichero que no existe", len(obs))
	}

	// PRESENTE Y VACIO: alguien puso un fichero y no dijo nada. Tambien es un
	// dato.
	for _, c := range []string{"", "\n\n", "# solo un comentario\n"} {
		v := conFichero(t, c)
		obs, _, err := v.Recolectar(manual.Fuente, "")
		if err != nil {
			t.Errorf("un fichero vacio (%q) no es un error, y dio: %v", c, err)
		}
		if len(obs) != 0 {
			t.Errorf("%d observaciones de %q", len(obs), c)
		}
	}
}

// TestLoQueNoSeEntiendeEsSIEMPREUnErrorYNuncaCeroObservaciones es el corazon del
// paquete. Leer cualquiera de estos como «todavia nadie ha recolectado» le
// diria a alguien que su instalacion esta al dia cuando lo que pasa es que no
// hemos podido leer su fichero.
func TestLoQueNoSeEntiendeEsSIEMPREUnErrorYNuncaCeroObservaciones(t *testing.T) {
	casos := []struct {
		nombre    string
		contenido string
		centinela error
		porque    string
	}{
		{
			nombre:    "una version que este plazum no lee",
			contenido: `{"version":99}` + "\n",
			centinela: manual.ErrVersion,
			porque: "una version futura puede traer campos que cambien el significado " +
				"de los que si se entienden",
		},
		{
			nombre:    "una cabecera que no es JSON",
			contenido: "esto no es json\n",
			centinela: manual.ErrLineaIlegible,
			porque:    "sin cabecera no se sabe en que formato esta el resto",
		},
		{
			nombre:    "una linea que no es JSON",
			contenido: cabecera + "\nno soy json\n",
			centinela: manual.ErrLineaIlegible,
			porque:    "una linea que no se entiende no es una linea que no existe",
		},
		{
			nombre:    "un campo que nadie lee",
			contenido: cabecera + "\n" + `{"prueba":"p","recurso":"r","satisfecho":true,"cuando":"2026-03-01T09:00:00Z","corroborado":true}` + "\n",
			centinela: manual.ErrLineaIlegible,
			porque: "un dato que nadie lee acaba mintiendo: quien lo escribio cree que " +
				"ha entrado",
		},
		{
			nombre:    "sin prueba",
			contenido: cabecera + "\n" + `{"recurso":"r","satisfecho":true,"cuando":"2026-03-01T09:00:00Z"}` + "\n",
			centinela: manual.ErrSinPrueba,
			porque:    "no se puede colgar de ninguna obligacion",
		},
		{
			nombre:    "sin recurso",
			contenido: cabecera + "\n" + `{"prueba":"p","satisfecho":true,"cuando":"2026-03-01T09:00:00Z"}` + "\n",
			centinela: manual.ErrSinRecurso,
			porque:    "dos observaciones del mismo control no se podrian distinguir",
		},
		{
			nombre:    "sin instante",
			contenido: cabecera + "\n" + `{"prueba":"p","recurso":"r","satisfecho":true}` + "\n",
			centinela: manual.ErrSinInstante,
			porque: "sin frescura, una observacion de hace tres anos vale igual que la " +
				"de esta manana",
		},
		{
			nombre:    "un instante que no se entiende",
			contenido: cabecera + "\n" + `{"prueba":"p","recurso":"r","satisfecho":true,"cuando":"ayer"}` + "\n",
			centinela: manual.ErrSinInstante,
			porque:    "presente y no interpretable, que es la tercera forma",
		},
		{
			nombre:    "una caducidad que no se entiende",
			contenido: cabecera + "\n" + `{"prueba":"p","recurso":"r","satisfecho":true,"cuando":"2026-03-01T09:00:00Z","caduca":"pronto"}` + "\n",
			centinela: manual.ErrLineaIlegible,
			porque:    "una caducidad ilegible NO es «sin caducidad»",
		},
		{
			nombre:    "sin decir si el predicado se cumplio",
			contenido: cabecera + "\n" + `{"prueba":"p","recurso":"r","cuando":"2026-03-01T09:00:00Z"}` + "\n",
			centinela: manual.ErrLineaIlegible,
			porque:    "el campo ausente no se lee como false: eso es elegir por el autor",
		},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			r := conFichero(t, c.contenido)
			obs, _, err := r.Recolectar(manual.Fuente, "")
			if err == nil {
				t.Fatalf("ha devuelto %d observaciones sin error. Y no puede: %s",
					len(obs), c.porque)
			}
			if !errors.Is(err, c.centinela) {
				t.Fatalf("ha fallado por otra cosa:\n  esperaba: %v\n  y dio:    %v",
					c.centinela, err)
			}
			if obs != nil {
				t.Errorf("con error devuelve %d observaciones: no se entrega nada a "+
					"medias", len(obs))
			}
			if !strings.Contains(err.Error(), "Arreglo:") {
				t.Errorf("el error no dice como salir de ahi: %v", err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// INVARIANTE 13: hechos, jamas veredictos
// ---------------------------------------------------------------------------

// TestUnFicheroConUnVeredictoDentroNoSeLeeIgnorandoloSePara.
//
// Es el invariante 13 en la frontera de ENTRADA. La puerta de `nucleo/estado`
// vigila que el TIPO no gane un campo de juicio; esta vigila que uno no entre
// por el fichero pidiendo que se ignore. Ignorarlo dejaria a quien lo escribio
// creyendo que su veredicto ha entrado en el expediente.
func TestUnFicheroConUnVeredictoDentroNoSeLeeIgnorandoloSePara(t *testing.T) {
	for _, campo := range []string{"cumple", "conforme", "veredicto", "severidad",
		"nivel_de_riesgo", "puntuacion"} {
		t.Run(campo, func(t *testing.T) {
			linea := `{"prueba":"p","recurso":"r","satisfecho":true,` +
				`"cuando":"2026-03-01T09:00:00Z","` + campo + `":"alto"}`
			r := conFichero(t, cabecera+"\n"+linea+"\n")
			_, _, err := r.Recolectar(manual.Fuente, "")
			if !errors.Is(err, manual.ErrVeredicto) {
				t.Fatalf("un campo de veredicto tiene que parar la lectura, y dio: %v", err)
			}
			if !strings.Contains(err.Error(), "Invariante 13") {
				t.Errorf("el error no dice de donde sale la regla: %v", err)
			}
		})
	}

	// CONTROL POSITIVO: un fichero sin campos de juicio SI se lee. Sin esto, la
	// puerta de arriba la aprobaria un recolector que rechaza todo.
	r := conFichero(t, bueno)
	if _, _, err := r.Recolectar(manual.Fuente, ""); err != nil {
		t.Fatalf("un fichero legitimo tiene que leerse: %v", err)
	}
}

// ---------------------------------------------------------------------------
// EL CONTRATO DEL PUERTO
// ---------------------------------------------------------------------------

func TestElCursorYLaFuenteNoSeIgnoranEnSilencio(t *testing.T) {
	r := conFichero(t, bueno)

	// UN CURSOR DE ENTRADA significa que quien llama cree estar continuando
	// algo que nunca empezo.
	if _, _, err := r.Recolectar(manual.Fuente, "pagina-2"); !errors.Is(err, manual.ErrCursor) {
		t.Errorf("un cursor de entrada tiene que ser error, y dio: %v", err)
	}

	// UNA FUENTE AJENA no puede devolver cero observaciones: eso haria creer
	// que esa fuente no tiene nada.
	if _, _, err := r.Recolectar("entra-id", ""); !errors.Is(err, manual.ErrFuente) {
		t.Errorf("una fuente ajena tiene que ser error, y dio: %v", err)
	}

	// Y LA FUENTE VACIA SI VALE, que es el caso de quien tiene un solo
	// recolector montado y no se molesta en nombrarlo.
	if _, _, err := r.Recolectar("", ""); err != nil {
		t.Errorf("la fuente vacia tiene que valer: %v", err)
	}
}

func TestElValorCeroDeOpcionesEstaProhibido(t *testing.T) {
	if _, err := manual.Abrir(manual.Opciones{}); !errors.Is(err, manual.ErrSinRuta) {
		t.Errorf("Opciones{} tiene que fallar: %v", err)
	}
	if _, err := manual.Abrir(manual.Opciones{Ruta: "   "}); !errors.Is(err, manual.ErrSinRuta) {
		t.Errorf("una ruta en blanco tiene que fallar: %v", err)
	}
	// CONTROL POSITIVO.
	if _, err := manual.Abrir(manual.Opciones{Ruta: "x.jsonl"}); err != nil {
		t.Errorf("una ruta legitima tiene que valer: %v", err)
	}
}

// TestLoRecolectadoLlegaAEstadoCalcularYDaUnEstado es el eslabon 1 enganchado
// al 3, que es lo que hace que este paquete sirva para algo.
func TestLoRecolectadoLlegaAEstadoCalcularYDaUnEstado(t *testing.T) {
	r := conFichero(t, cabecera+"\n"+
		`{"prueba":"p.privilegios","recurso":"srv-01","satisfecho":true,"cuando":"2026-03-01T09:00:00Z"}`+"\n")
	obs, _, err := r.Recolectar(manual.Fuente, "")
	if err != nil {
		t.Fatal(err)
	}
	ent := calcular(obs, time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC))
	if ent.Estado.String() != "pass" {
		t.Fatalf("estado %q, esperaba pass: %s", ent.Estado, ent.Motivo)
	}
	// Y LA ANTIGUEDAD SALE DE AQUI, que es la segunda de las tres cosas que la
	// pantalla ensena.
	if ent.Recolectada.IsZero() {
		t.Error("la entrada no trae de cuando es el dato")
	}
	// Y un dia despues del TTL, el mismo dato ya no vale. Sin este segundo
	// caso, el de arriba lo aprobaria un motor que devuelve pass siempre.
	viejo := calcular(obs, time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC))
	if viejo.Estado.String() != "obsoleto" {
		t.Errorf("dos meses despues el estado es %q y tenia que ser obsoleto: %s",
			viejo.Estado, viejo.Motivo)
	}
}

// calcular monta la prueba del paquete, la cruza al expediente y llama al
// motor. Es el arnes de los eslabones 2 y 3, y va aqui porque el recolector no
// puede depender del corpus: quien los une es quien cabla el producto.
func calcular(obs []estado.Observacion, ahora time.Time) estado.Entrada {
	pr, err := (corpus.Prueba{
		ID: "p.privilegios", Obligacion: "demo.accesos", Recurso: "Privilegio",
		TTL: "P30D", SLA: "P7D", Predicado: "cada privilegio tiene funcion declarada",
	}).AlExpediente()
	if err != nil {
		panic(err)
	}
	return estado.Calcular(pr, obs, estado.Contexto{Ahora: ahora, Aplicable: true})
}

// TestLaLeyDeConservacionParaSiLosCubosNoSumanElTotal es la guarda que la
// mutacion M37 destapo, y su historia es la mitad de su valor.
//
// Apagarla dejaba la suite entera en verde, porque HOY NINGUN FICHERO PUEDE
// PRODUCIR UN DESCUADRE: el contador del total y el parser recorren las mismas
// lineas. Una guarda que ninguna entrada alcanza es una guarda que no existe.
//
// Se queda, con dato sintetico, porque lo que vigila no es un fichero: es el
// parser del futuro. El dia que alguien meta un `continue` de mas en el bucle,
// los dos contadores dejaran de coincidir y esto parara.
func TestLaLeyDeConservacionParaSiLosCubosNoSumanElTotal(t *testing.T) {
	// LA DIRECCION QUE ACUSA: faltan lineas por explicar.
	roto := manual.Recuento{Total: 10, Cabecera: 1, Leidas: 3, EnBlanco: 1, Comentario: 1}
	err := manual.ComprobarRecuento(roto)
	if !errors.Is(err, manual.ErrDescuadre) {
		t.Fatalf("un recuento que no suma tiene que parar, y dio: %v", err)
	}
	if !strings.Contains(err.Error(), "Arreglo:") {
		t.Errorf("el error no dice que hacer: %v", err)
	}
	// El error ENSENA los cubos: sin ellos, quien lo lea no sabe cuantas
	// lineas se han perdido ni por donde buscarlas.
	if !strings.Contains(err.Error(), "total=10") {
		t.Errorf("el error no ensena el reparto: %v", err)
	}

	// LA DIRECCION CONTRARIA, que es la que impide que esto sea un rojo
	// permanente: un recuento que suma pasa.
	if err := manual.ComprobarRecuento(
		manual.Recuento{Total: 6, Cabecera: 1, Leidas: 3, EnBlanco: 1, Comentario: 1}); err != nil {
		t.Errorf("un recuento que suma tiene que pasar: %v", err)
	}
	// Y el vacio tambien: cero lineas suman cero.
	if err := manual.ComprobarRecuento(manual.Recuento{}); err != nil {
		t.Errorf("el recuento vacio tiene que pasar: %v", err)
	}
}
