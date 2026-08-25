package main

// La vigilancia se prueba en LAS DOS DIRECCIONES, y sin las dos solo hay media
// prueba:
//
//	caza un cambio real          -> TestLaVigilanciaCazaUnCambioReal
//	no grita cuando no hay nada  -> TestLaVigilanciaNoGritaSiNoHaCambiadoNada
//
// Un detector que dijera "modificado" siempre pasaria la primera y suspenderia
// la segunda; uno que no dijera nunca nada, al reves. Los dos son inutiles y los
// dos se ven aqui.
//
// El cambio que se usa es REAL y no inventado: la disposicion adicional segunda
// del Real Decreto 311/2022 se modifico el 6 de noviembre de 2024 (Real Decreto
// 1125/2024). El fixture "antes" es ese mismo bloque del BOE sin su segunda
// version, que es exactamente como el BOE lo servia hasta esa fecha.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func extraccionDePrueba(t *testing.T, fixture string) *Extraccion {
	t.Helper()
	as := articulosDePrueba(t, fixture, opcionesBOE{Referencia: hoyDePrueba})
	o := origenDeMetadatos(metadatosDePrueba(t), piezasELI{
		Rango: "rd", Ano: "2022", Mes: "05", Dia: "03", Numero: "311",
		Base: "https://www.boe.es/eli/es/rd/2022/05/03/311",
	}, "https://www.boe.es/eli/es/rd/2022/05/03/311/con", "")
	return armar(o, LicenciaBOE, AtribucionBOE, as, time.Date(2026, 8, 26, 9, 0, 0, 0, time.UTC))
}

func TestLaVigilanciaCazaUnCambioReal(t *testing.T) {
	antes := extraccionDePrueba(t, "boe-texto-antes.xml")
	ahora := extraccionDePrueba(t, "boe-texto.xml")
	if antes.Huella == ahora.Huella {
		t.Fatal("los dos fixtures son distintos y la huella de la norma sale igual: la huella " +
			"no esta mirando el texto")
	}
	c := Comparar(antes, ahora)
	if !c.Hay() {
		t.Fatal("HALLAZGO: la disposicion adicional segunda cambio de verdad el 06-11-2024 y " +
			"la vigilancia dice que no ha pasado nada. Sin esto la herramienta no vigila")
	}
	if len(c.Modificados) != 1 || c.Modificados[0] != "Disposición adicional segunda" {
		t.Fatalf("modificados %v: tenia que ser exactamente la disposicion adicional segunda",
			c.Modificados)
	}
	if len(c.Nuevos) != 0 || len(c.Derogados) != 0 {
		t.Errorf("nada nacio ni murio, y salio nuevos=%v derogados=%v", c.Nuevos, c.Derogados)
	}
	if c.SinCambio != 2 {
		t.Errorf("los otros dos articulos del fixture estan igual y el contador dice %d", c.SinCambio)
	}
	if c.HuellaAntes == "" || c.HuellaAhora == "" || c.HuellaAntes == c.HuellaAhora {
		t.Errorf("las huellas de la comparacion no cuadran: %q y %q", c.HuellaAntes, c.HuellaAhora)
	}
}

func TestLaVigilanciaNoGritaSiNoHaCambiadoNada(t *testing.T) {
	// La MISMA norma leida dos veces. Una herramienta de vigilancia que avisa
	// cuando no ha pasado nada se acaba ignorando, y entonces no avisa nunca.
	a := extraccionDePrueba(t, "boe-texto.xml")
	b := extraccionDePrueba(t, "boe-texto.xml")
	c := Comparar(a, b)
	if c.Hay() {
		t.Fatalf("HALLAZGO: nada ha cambiado y la vigilancia grita: nuevos=%v modificados=%v "+
			"derogados=%v", c.Nuevos, c.Modificados, c.Derogados)
	}
	if c.SinCambio != len(b.Articulos) {
		t.Errorf("los %d articulos estan igual y solo se contaron %d", len(b.Articulos), c.SinCambio)
	}
	if c.PrimeraVez {
		t.Error("habia instantanea anterior: esto no es una primera vez")
	}
}

// La primera vez no es un cambio. Decir "3 articulos nuevos" la primera vez es
// ruido, y el ruido entrena a ignorar la herramienta.
func TestLaPrimeraVezNoEsUnCambio(t *testing.T) {
	c := Comparar(nil, extraccionDePrueba(t, "boe-texto.xml"))
	if !c.PrimeraVez {
		t.Fatal("sin instantanea anterior es primera vez")
	}
	if c.Hay() {
		t.Fatal("la primera vez no cuenta como cambio")
	}
	if len(c.Nuevos) != 0 {
		t.Errorf("y no se enumeran como nuevos los articulos: %v", c.Nuevos)
	}
}

func TestUnArticuloQueApareceEsNuevoYUnoQueDesapareceEsDerogado(t *testing.T) {
	base := extraccionDePrueba(t, "boe-texto.xml")

	conMenos := *base
	conMenos.Articulos = base.Articulos[1:]
	conMenos.Huella = HuellaDeExtraccion(conMenos.Articulos)

	// Aparece: antes faltaba uno, ahora esta.
	c := Comparar(&conMenos, base)
	if len(c.Nuevos) != 1 || c.Nuevos[0] != base.Articulos[0].Referencia {
		t.Errorf("nuevos %v, se esperaba %q", c.Nuevos, base.Articulos[0].Referencia)
	}
	// Desaparece: el BOE deja de servir el bloque.
	c = Comparar(base, &conMenos)
	if len(c.Derogados) != 1 || c.Derogados[0] != base.Articulos[0].Referencia {
		t.Errorf("derogados %v, se esperaba %q", c.Derogados, base.Articulos[0].Referencia)
	}
	if len(c.Modificados) != 0 {
		t.Errorf("un articulo que desaparece no es una modificacion: %v", c.Modificados)
	}
}

// Derogar en el sitio se cuenta como derogacion y no como modificacion, aunque
// el texto tambien cambie: lo primero que tiene que leer quien mantiene el
// paquete es que ese precepto ya no obliga.
func TestDerogarEnElSitioSeCuentaComoDerogacion(t *testing.T) {
	antes := extraccionDePrueba(t, "boe-texto.xml")
	despues := *antes
	despues.Articulos = append([]Articulo(nil), antes.Articulos...)
	i := 0
	despues.Articulos[i].Parrafos = []string{"(Derogado)"}
	despues.Articulos[i].Texto = "(Derogado)"
	despues.Articulos[i].Derogado = true
	despues.Articulos[i].Huella = despues.Articulos[i].HuellaDe()
	despues.Huella = HuellaDeExtraccion(despues.Articulos)

	c := Comparar(antes, &despues)
	if len(c.Derogados) != 1 || c.Derogados[0] != antes.Articulos[i].Referencia {
		t.Fatalf("derogados %v", c.Derogados)
	}
	if len(c.Modificados) != 0 {
		t.Errorf("no se cuenta ademas como modificado: %v", c.Modificados)
	}
}

// La huella mira el TITULO ademas del cuerpo: cambiarle el rotulo a un articulo
// es un cambio normativo y saldria "sin cambio" si solo se mirara el texto.
func TestCambiarSoloElTituloYaEsUnCambio(t *testing.T) {
	antes := extraccionDePrueba(t, "boe-texto.xml")
	despues := *antes
	despues.Articulos = append([]Articulo(nil), antes.Articulos...)
	despues.Articulos[0].Titulo = antes.Articulos[0].Titulo + " (revisada)"
	despues.Articulos[0].Huella = despues.Articulos[0].HuellaDe()
	despues.Huella = HuellaDeExtraccion(despues.Articulos)

	if c := Comparar(antes, &despues); len(c.Modificados) != 1 {
		t.Fatalf("un cambio de titulo tiene que verse: %+v", c)
	}
}

// La clave de comparacion es el ROTULO y no el id interno. Los id del BOE se
// derivan de la posicion, asi que insertar un articulo los desplaza; con el id
// como clave, insertar un articulo diria que cambio media norma.
func TestLaComparacionNoSeRompeSiLaFuenteRenumeraSusIdInternos(t *testing.T) {
	antes := extraccionDePrueba(t, "boe-texto.xml")
	despues := *antes
	despues.Articulos = append([]Articulo(nil), antes.Articulos...)
	for i := range despues.Articulos {
		despues.Articulos[i].ID += "-9" // la fuente reordena sus bloques
		despues.Articulos[i].Huella = despues.Articulos[i].HuellaDe()
	}
	despues.Huella = HuellaDeExtraccion(despues.Articulos)
	if c := Comparar(antes, &despues); c.Hay() {
		t.Fatalf("renumerar los id internos no cambia ni una coma del derecho, y la vigilancia "+
			"grita: %+v", c)
	}
}

// --- el almacen ---

func TestElAlmacenGuardaCompararYAnotarEnElHistorial(t *testing.T) {
	dir := t.TempDir()
	alm := Almacen{Dir: dir}
	e1 := extraccionDePrueba(t, "boe-texto-antes.xml")
	k := clave(e1.Fuente)

	anterior, err := alm.Anterior(k)
	if err != nil {
		t.Fatal(err)
	}
	if anterior != nil {
		t.Fatal("el almacen esta vacio y devuelve algo")
	}
	t0 := time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC)
	if err := alm.Registrar(k, e1, Comparar(nil, e1), t0); err != nil {
		t.Fatal(err)
	}

	e2 := extraccionDePrueba(t, "boe-texto.xml")
	anterior, err = alm.Anterior(k)
	if err != nil {
		t.Fatal(err)
	}
	if anterior == nil {
		t.Fatal("la instantanea guardada no vuelve")
	}
	c := Comparar(anterior, e2)
	if len(c.Modificados) != 1 {
		t.Fatalf("el cambio no sobrevive al viaje por disco: %+v", c)
	}
	t1 := t0.Add(48 * time.Hour)
	if err := alm.Registrar(k, e2, c, t1); err != nil {
		t.Fatal(err)
	}

	h, err := alm.Historial()
	if err != nil {
		t.Fatal(err)
	}
	if len(h) != 2 {
		t.Fatalf("dos observaciones y el historial tiene %d: el historial es append-only", len(h))
	}
	// De lo mas reciente a lo mas antiguo.
	if h[0].Observado != t1.Format(time.RFC3339) {
		t.Errorf("el historial no viene ordenado: %v", h[0].Observado)
	}
	if !h[1].Cambios.PrimeraVez {
		t.Error("la primera anotacion tiene que decir que era la primera vez")
	}
	if h[0].Identificador == "" || h[0].Titulo == "" || h[0].URLDocumento == "" {
		t.Errorf("a la entrada del historial le falta con que pintar la tabla publica: %+v", h[0])
	}
	if h[0].Cambios.FuenteAhora == "" {
		t.Error("sin la fecha que declara la fuente no hay tabla de fecha-fuente a fecha-paquete")
	}

	// Y la tercera ejecucion, sin cambios, tambien se anota: el track record
	// tiene que poder demostrar que se miro y no habia nada.
	if err := alm.Registrar(k, e2, Comparar(e2, e2), t1.Add(24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	h, _ = alm.Historial()
	if len(h) != 3 {
		t.Fatalf("una observacion sin cambios tambien se anota: %d", len(h))
	}
}

// Una instantanea corrupta NO puede leerse como "primera vez": eso convertiria
// un fichero roto en un parte de sin novedad, que es la peor mentira que puede
// contar una herramienta de vigilancia.
func TestUnaInstantaneaCorruptaNoSeLeeComoPrimeraVez(t *testing.T) {
	dir := t.TempDir()
	alm := Almacen{Dir: dir}
	k := "es-boe-a-2022-7191"
	if err := os.MkdirAll(filepath.Join(dir, k), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, k, "instantanea.json"), []byte("{roto"), 0o600); err != nil {
		t.Fatal(err)
	}
	e, err := alm.Anterior(k)
	if err == nil {
		t.Fatal("HALLAZGO: una instantanea corrupta se lee como si no existiera, y la siguiente " +
			"ejecucion diria primera vez en vez de decir que algo va mal")
	}
	if e != nil {
		t.Error("no devuelve extraccion")
	}
	if !strings.Contains(err.Error(), "Arreglo") {
		t.Errorf("el error no dice como arreglarlo: %v", err)
	}
}

// El identificador viene de una fuente externa y acaba en una ruta de fichero.
func TestLaClaveNoDejaSalirDelAlmacen(t *testing.T) {
	casos := map[string]string{
		"../../../etc/passwd": "es-_________etc_passwd",
		"BOE-A-2022-7191":     "es-boe-a-2022-7191",
		"":                    "es-sin-identificador",
	}
	for id, quiero := range casos {
		got := clave(Origen{Jurisdiccion: "es", Identificador: id})
		if got != quiero {
			t.Errorf("clave(%q) = %q, se esperaba %q", id, got, quiero)
		}
		if strings.ContainsAny(got, `/\.`) {
			t.Errorf("clave(%q) = %q lleva separadores de ruta", id, got)
		}
	}
	// Y no la limpia hasta dejarla vacia, que seria colisionar todas las normas
	// en el mismo directorio.
	if clave(Origen{Jurisdiccion: "ue", Identificador: "32016R0679"}) ==
		clave(Origen{Jurisdiccion: "ue", Identificador: "32016R0680"}) {
		t.Error("dos normas distintas comparten clave")
	}
}
