package corpus

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// Hallazgo del frente de corpus, verificado aqui antes de arreglarlo.
//
// La clase de un paquete es un uint8 que viene de un fichero JSON, o sea de
// fuera. Validar la mira con un switch que tiene default, y String() indexa un
// array de cinco. Las dos cosas juntas dan un agujero en la unica frontera que
// CLAUDE.md declara no negociable: el linter legal.

const textoLargoSimulado = "texto normativo simulado de una norma de pago que no se puede " +
	"redistribuir, repetido hasta pasar holgadamente del limite de ciento veinte caracteres " +
	"que impone el linter para los paquetes referenciales"

// Una clase fuera de rango esquiva el limite de texto: cae en el default del
// switch, que no comprueba nada. Un paquete que en la practica es referencial
// pero declara "clase": 9 redistribuye texto de un catalogo de pago sin que
// nadie lo pare.
//
// El paquete de este test es sintetico, del espacio urn:demo:. Da igual de que
// norma se trate: el agujero esta en el numero de la clase, no en el catalogo.
func TestHostilUnaClaseFueraDeRangoEsquivaLaFronteraLegal(t *testing.T) {
	p := &Paquete{
		URN: "urn:demo:clase-fuera-de-rango", Version: "2022", Clase: Clase(9),
		Vigencia: Vigencia{Desde: "2022-01-01"},
		Obligaciones: []Obligacion{{
			ID: "demo.control.5.1", Articulo: "A.5.1", Cita: "catalogo de pago, control A.5.1", ClaseE2E: "observable",
			TextoLegal: textoLargoSimulado,
		}},
	}
	if len(textoLargoSimulado) <= LimiteTextoReferencial {
		t.Fatalf("el texto de prueba tiene que pasar del limite para que el test pruebe algo: %d",
			len(textoLargoSimulado))
	}
	errs := p.Validar()
	if len(errs) == 0 {
		t.Fatalf("HALLAZGO: un paquete con clase 9 y %d caracteres de texto normativo valida "+
			"limpio. El switch de Validar tiene default, asi que una clase fuera de rango no "+
			"es referencial y no tiene limite. Es la frontera legal esquivada con un numero",
			len(textoLargoSimulado))
	}
	var mencionaClase bool
	for _, e := range errs {
		if strings.Contains(e.Error(), "fuera de rango") {
			mencionaClase = true
		}
	}
	if !mencionaClase {
		t.Fatalf("tiene que quejarse de la CLASE fuera de rango. Ojo: buscar solo la palabra "+
			"clase da un falso negativo, porque clase_e2e la contiene. Errores: %v", errs)
	}
	// Y ademas se le aplica la frontera legal, sin depender de que el chequeo de
	// la clase siga vivo. Una clase que no existe no acredita ningun derecho de
	// redistribucion, asi que se trata como la mas estricta. Dos cierres para el
	// mismo agujero, porque este agujero ya se abrio una vez.
	var frontera bool
	for _, e := range errs {
		if errors.Is(e, ErrTextoRedistribuido) {
			frontera = true
		}
	}
	if !frontera {
		t.Fatalf("una clase fuera de rango tiene que llevar ademas el limite de texto mas "+
			"estricto, y no lo lleva. Errores: %v", errs)
	}
}

// Y la misma clase fuera de rango revienta al imprimirla: String() indexa un
// array de cinco elementos con un uint8 que viene de un fichero.
func TestHostilUnaClaseFueraDeRangoNoRevientaAlImprimirse(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("HALLAZGO: Clase(9).String() hace panic (%v). El valor viene de un JSON "+
				"que aporta un tercero, asi que un paquete malformado tumba a quien lo liste", r)
		}
	}()
	for _, c := range []Clase{Importado, Transcrito, Referencial, Delegado, Propio, Clase(9), Clase(255)} {
		if s := c.String(); s == "" {
			t.Fatalf("la clase %d no puede imprimirse como cadena vacia", uint8(c))
		}
	}
}

// Control negativo de los dos: las clases validas siguen funcionando y el
// limite sigue aplicandose a los referenciales de verdad.
func TestHostilElLimiteSigueVigenteEnLosReferencialesDeVerdad(t *testing.T) {
	p := &Paquete{
		URN: "urn:demo:referencial", Version: "2022", Clase: Referencial,
		Vigencia: Vigencia{Desde: "2022-01-01"},
		Obligaciones: []Obligacion{{
			ID: "demo.control.5.1", Articulo: "A.5.1", Cita: "catalogo de pago, control A.5.1", ClaseE2E: "observable",
			TextoLegal: textoLargoSimulado,
		}},
	}
	errs := p.Validar()
	if len(errs) == 0 {
		t.Fatal("un referencial con texto largo tiene que rechazarse: es la frontera legal")
	}
	// Por identidad del error, no por "hay algun error". Este test daba verde
	// con un clase_e2e mal escrito dentro del propio caso: cualquier fallo del
	// linter lo satisfacia, incluido uno que no tiene nada que ver con la
	// frontera legal. Es el patron tapado del que este proyecto se defiende.
	if !errors.Is(errs[0], ErrTextoRedistribuido) {
		t.Fatalf("el rechazo tiene que ser el de la frontera legal: %v", errs)
	}
}

// El ataque de verdad: el texto NO entra por texto_legal, que es el unico campo
// que se miraba. Entra por la puerta de al lado, y ademas se hace desde DISCO,
// que es como llega un paquete de un tercero: fichero JSON y a cargar.
//
// Cada caso es un campo distinto del formato. Si alguno vuelve a quedarse sin
// vigilar, aqui se ve, y se ve con el texto dentro.
func TestHostilElTextoDeUnCatalogoDePagoEntraPorElCampoDeAlLado(t *testing.T) {
	casos := []struct {
		nombre  string
		paquete string
	}{
		{"por la ayuda de un atributo", `{
          "urn":"urn:demo:por-la-ayuda","version":"1","clase":2,
          "fuente":"https://ejemplo.invalid/catalogo","vigencia":{"desde":"2022-01-01"},
          "entidades":[{"nombre":"sistema","descripcion":"el sistema",
            "atributos":[{"nombre":"cifrado","tipo":2,"cita":"catalogo A.8.24",
              "ayuda":"` + textoLargoSimulado + `"}]}],
          "obligaciones":[{"id":"demo.control.8.24","articulo":"A.8.24",
            "cita":"catalogo de pago, control A.8.24","clase_e2e":"observable"}]}`},
		{"por el titulo de la obligacion", `{
          "urn":"urn:demo:por-el-titulo","version":"1","clase":2,
          "fuente":"https://ejemplo.invalid/catalogo","vigencia":{"desde":"2022-01-01"},
          "obligaciones":[{"id":"demo.control.8.24","articulo":"A.8.24",
            "titulo":"` + textoLargoSimulado + `",
            "cita":"catalogo de pago, control A.8.24","clase_e2e":"observable"}]}`},
		{"por la descripcion de la entidad", `{
          "urn":"urn:demo:por-la-descripcion","version":"1","clase":2,
          "fuente":"https://ejemplo.invalid/catalogo","vigencia":{"desde":"2022-01-01"},
          "entidades":[{"nombre":"sistema","descripcion":"` + textoLargoSimulado + `",
            "atributos":[{"nombre":"cifrado","tipo":2,"cita":"catalogo A.8.24"}]}],
          "obligaciones":[{"id":"demo.control.8.24","articulo":"A.8.24",
            "cita":"catalogo de pago, control A.8.24","clase_e2e":"observable"}]}`},
		{"por el titulo de la plantilla", `{
          "urn":"urn:demo:por-la-plantilla","version":"1","clase":2,
          "fuente":"https://ejemplo.invalid/catalogo","vigencia":{"desde":"2022-01-01"},
          "plantillas":[{"id":"demo.pl","titulo":"` + textoLargoSimulado + `",
            "cita":"catalogo A.5.1","campos":[{"nombre":"alcance","origen":"entidad:sistema.cifrado"}]}],
          "obligaciones":[{"id":"demo.control.8.24","articulo":"A.8.24",
            "cita":"catalogo de pago, control A.8.24","clase_e2e":"observable"}]}`},
		{"por el articulo, que parece un localizador", `{
          "urn":"urn:demo:por-el-articulo","version":"1","clase":2,
          "fuente":"https://ejemplo.invalid/catalogo","vigencia":{"desde":"2022-01-01"},
          "obligaciones":[{"id":"demo.control.8.24","articulo":"` + textoLargoSimulado + `",
            "cita":"catalogo de pago, control A.8.24","clase_e2e":"observable"}]}`},
	}
	for _, c := range casos {
		dir := t.TempDir()
		escribirPaquete(t, dir, "catalogo-de-pago", c.paquete)
		_, err := Cargar(dir)
		if err == nil {
			t.Errorf("HALLAZGO: %s se cuela %d caracteres de texto de un catalogo de pago "+
				"en un paquete referencial y el corpus carga limpio", c.nombre, len(textoLargoSimulado))
			continue
		}
		if !errors.Is(err, ErrTextoRedistribuido) {
			t.Errorf("%s: el paquete se rechaza, pero no por la frontera legal: %v", c.nombre, err)
		}
	}
}

// Control negativo del ataque anterior: el mismo paquete, con el identificador y
// el titulo corto que SI se pueden distribuir, carga sin una queja. Sin esto, un
// linter que rechazara todos los referenciales tambien pasaria el test.
func TestHostilElReferencialLegitimoSigueCargando(t *testing.T) {
	dir := t.TempDir()
	escribirPaquete(t, dir, "catalogo-de-pago", `{
      "urn":"urn:demo:legitimo","version":"1","clase":2,
      "licencia_fuente":"sin-licencia-de-texto",
      "atribucion":"Sin texto normativo: identificador y titulo corto. La copia la aporta el cliente.",
      "fuente":"https://ejemplo.invalid/catalogo","vigencia":{"desde":"2022-01-01"},
      "entidades":[{"nombre":"sistema","descripcion":"el sistema dentro del alcance",
        "atributos":[{"nombre":"cifrado","tipo":2,"cita":"CAT/DEMO 9999:2026 A.8.24",
          "ayuda":"si no lo sabes, mira el inventario de sistemas"}]}],
      "obligaciones":[{"id":"demo.control.8.24","articulo":"A.8.24",
        "titulo":"Cifrado de la informacion en transito",
        "cita":"CAT/DEMO 9999:2026 A.8.24. El texto del control lo aporta el cliente con su copia licenciada",
        "clase_e2e":"observable"}]}`)
	if _, err := Cargar(dir); err != nil {
		t.Fatalf("un referencial con identificador y titulo corto tiene que cargar: %v", err)
	}
}

// El paquete impostor: un directorio de mas que declara el URN de una norma que
// ya esta instalada. El corpus es un arbol de ficheros que se copia y se
// sincroniza, asi que colar un directorio no es una hipotesis remota, y el URN
// es lo que identifica a la norma en el expediente y en las equivalencias.
func TestHostilDosPaquetesNoPuedenCompartirURN(t *testing.T) {
	real := `{
      "urn":"urn:demo:norma","version":"1","clase":4,
      "licencia_fuente":"del-proyecto",
      "atribucion":"Datos sinteticos del proyecto, sin tercero con derechos.",
      "fuente":"https://ejemplo.invalid/norma","vigencia":{"desde":"2022-01-01"},
      "obligaciones":[{"id":"demo.o.1","articulo":"1","cita":"demo art. 1",
        "clase_e2e":"documental"}]}`
	impostor := strings.Replace(real, `"version":"1"`, `"version":"1.0.1"`, 1)

	dir := t.TempDir()
	escribirPaquete(t, dir, "norma", real)
	escribirPaquete(t, dir, "norma-actualizada", impostor)
	_, err := Cargar(dir)
	if err == nil {
		t.Fatal("HALLAZGO: dos directorios declaran el mismo urn y el corpus carga los dos. " +
			"Quien resuelva la norma por su urn se lleva el que salga")
	}
	if !errors.Is(err, ErrURNDuplicado) {
		t.Fatalf("el rechazo tiene que ser por el urn repetido: %v", err)
	}

	// Control negativo: con urn propio, los dos cargan.
	limpio := t.TempDir()
	escribirPaquete(t, limpio, "norma", real)
	escribirPaquete(t, limpio, "norma-otra", strings.Replace(real, "urn:demo:norma", "urn:demo:otra", 1))
	ps, err := Cargar(limpio)
	if err != nil || len(ps) != 2 {
		t.Fatalf("dos paquetes con urn distinto tienen que cargar: %d %v", len(ps), err)
	}
}

// El paquete que miente con las fechas. Ninguna de las tres formas puede acabar
// en "vigente": o se rechaza en la carga, o se responde que no.
func TestHostilUnaVigenciaQueMienteNoAlargaLaObligacion(t *testing.T) {
	ahora, err := time.Parse(time.RFC3339, "2026-01-01T00:00:00Z")
	if err != nil {
		t.Fatal(err)
	}

	// 1. La norma esta derogada y la obligacion se declara viva hasta el 2999.
	p := &Paquete{
		URN: "urn:demo:derogada", Version: "1", Clase: Propio,
		Fuente: "https://ejemplo.invalid/demo", Vigencia: Vigencia{Desde: "2010-01-01", Hasta: "2024-05-05"},
		Obligaciones: []Obligacion{{ID: "demo.o", Articulo: "1", Cita: "demo art. 1",
			ClaseE2E: "documental", Vigencia: Vigencia{Desde: "2010-01-01", Hasta: "2999-12-31"}}},
	}
	if ok, err := p.EnVigor(p.Obligaciones[0], ahora); err != nil || ok {
		t.Errorf("HALLAZGO: la obligacion sobrevive a su norma porque se declara a si "+
			"misma vigente hasta el 2999: %v %v", ok, err)
	}

	// 2. Fechas hostiles: ninguna puede leerse como vigente ni tumbar el linter.
	for _, mala := range []string{
		"2026-02-30",                 // dia que no existe
		"2026-13-01",                 // mes que no existe
		"2026-01-01T00:00:00+99:00",  // huso horario imposible
		"999999999999999-01-01",      // ano que no cabe
		"2026-01-01 00:00:00",        // casi RFC3339, con espacio
		strings.Repeat("2026", 1000), // basura larga
		"-2026-01-01",                // ano negativo
	} {
		v := Vigencia{Desde: mala}
		ok, err := v.VigenteEn(ahora)
		if ok {
			t.Errorf("HALLAZGO: la fecha %q se lee como vigente", mala)
		}
		if !errors.Is(err, ErrVigenciaIlegible) {
			t.Errorf("la fecha %q tiene que rechazarse con ErrVigenciaIlegible y dio %v", mala, err)
		}
	}

	// 3. Control negativo: una fecha buena si se lee, o sea que lo de arriba no
	// esta rechazandolo todo.
	if ok, err := (Vigencia{Desde: "2025-01-01"}).VigenteEn(ahora); err != nil || !ok {
		t.Errorf("una vigencia bien escrita tiene que leerse: %v %v", ok, err)
	}
}
