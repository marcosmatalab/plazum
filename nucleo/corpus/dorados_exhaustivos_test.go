package corpus

// LAS PUERTAS DEL ESPERADO EXHAUSTIVO.
//
// El agujero que cierran, medido antes de escribir una linea: hasta el
// 27-08-2026 un dorado declaraba UN vencimiento y el ejecutor filtraba por su
// hito, asi que decia lo que TIENE que salir y no decia nada de lo que NO tiene
// que salir. Quitandole la clase al hito del plazo general del art. 73 del AI
// Act, ese incidente pasaba a ensenar DOS fechas para la misma obligacion y los
// doce dorados del paquete seguian verdes.
//
// Cada test de aqui trae su control negativo pegado: se demuestra que el mismo
// caso pasa cuando la propiedad se cumple y falla cuando no. Un test que solo
// ensena el rojo no distingue "vigila" de "siempre dice que no".

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/marcosmatalab/plazum/nucleo/ventana"
)

// obligacionDeDosHitos es un incidente con dos plazos EXCLUYENTES por clase (el
// patron del art. 73 del AI Act y del art. 87 del MDR) mas un hito encadenado
// que no puede tener fecha hasta que se cumpla el primero. Con esto se pueden
// escribir las tres formas de discrepancia: sobra, falta y estado distinto.
func obligacionDeDosHitos() Obligacion {
	return Obligacion{
		ID: "demo.incidente", Cita: "c", ClaseE2E: "notificatoria",
		Temporalidad: &Temporalidad{
			Primitiva:  "plazo",
			Regimen:    RegimenSpec{Computo: "naturales", Cierre: "exacto"},
			Disparador: map[string]string{"hecho": "conocimiento"},
			Hitos: []HitoSpec{
				{ID: "notificacion_general", Limite: "PT72H", Clase: "grave_general"},
				{ID: "notificacion_agravada", Limite: "PT24H", Clase: "grave_agravado"},
				{ID: "informe_final", Limite: "P1M", DesdeHito: "notificacion_general"},
			},
		},
	}
}

// El caso agravado: el hito general NO puede salir, porque la clase lo desplaza.
func doradoAgravado() Dorado {
	return Dorado{
		Caso: "agravado: veinticuatro horas", Obligacion: "demo.incidente",
		Hechos: map[string]string{
			"conocimiento":   "2026-09-01T10:00:00Z",
			"grave_agravado": "2026-09-01T11:00:00Z",
		},
		Esperado: []EsperadoDorado{
			{Hito: "notificacion_agravada", Vence: "2026-09-02T10:00:00Z"},
			{Hito: "informe_final", Estado: "pendiente de hecho"},
		},
		CitaDelEsperado: "el apartado agravado desplaza al general",
	}
}

func TestElConjuntoEsperadoBienEscritoPasa(t *testing.T) {
	if err := EjecutarDorado(obligacionDeDosHitos(), doradoAgravado()); err != nil {
		t.Fatalf("el conjunto exhaustivo correcto tiene que pasar: %v", err)
	}
}

// DIRECCION 1: falta una fila que el motor si devuelve... al reves, falta en el
// motor lo que el dorado declara.
func TestUnDoradoSeCaeSiFaltaUnVencimientoQueElTextoExige(t *testing.T) {
	d := doradoAgravado()
	d.Esperado = append(d.Esperado, EsperadoDorado{
		Hito: "notificacion_de_otra_norma", Vence: "2026-09-02T10:00:00Z"})
	err := EjecutarDorado(obligacionDeDosHitos(), d)
	if err == nil {
		t.Fatal("HALLAZGO: el ejecutor no mira si el motor deja fuera un hito que el texto exige")
	}
	if !strings.Contains(err.Error(), "FALTA el hito \"notificacion_de_otra_norma\"") {
		t.Fatalf("el error tiene que decir QUE falta: %v", err)
	}
}

// DIRECCION 2, LA QUE NO EXISTIA. Un vencimiento que el motor ensena y el texto
// no da. Es la que muerde: dos fechas para la misma obligacion dejan al
// operador sin saber cual es la suya.
func TestUnDoradoSeCaeSiSobraUnVencimientoQueElTextoNoDa(t *testing.T) {
	o := obligacionDeDosHitos()
	// La mutacion del corpus, en pequeno: al hito general se le olvida la clase,
	// asi que rige SIEMPRE y sale a la vez que el agravado.
	o.Temporalidad.Hitos[0].Clase = ""
	err := EjecutarDorado(o, doradoAgravado())
	if err == nil {
		t.Fatal("HALLAZGO: un hito sin clase convive con el que tenia que desplazarlo y el " +
			"dorado sigue verde. Es exactamente el agujero del art. 73 del AI Act")
	}
	if !strings.Contains(err.Error(), "SOBRA el hito \"notificacion_general\"") {
		t.Fatalf("el error tiene que decir QUE sobra: %v", err)
	}
	// Control negativo: con la clase puesta, el mismo dorado pasa. Sin esto, el
	// rojo de arriba podria ser un rojo permanente.
	if err := EjecutarDorado(obligacionDeDosHitos(), doradoAgravado()); err != nil {
		t.Fatalf("con la clase puesta el mismo caso tiene que pasar: %v", err)
	}
}

// EL ESTADO ES UN RESULTADO, NO UN HUECO. Un hito pendiente de hecho declarado
// como determinado (o al reves) es una discrepancia, no un detalle de forma.
func TestUnDoradoComparaElEstadoYNoSoloLaFecha(t *testing.T) {
	d := doradoAgravado()
	for i := range d.Esperado {
		if d.Esperado[i].Hito == "informe_final" {
			d.Esperado[i] = EsperadoDorado{Hito: "informe_final", Vence: "2026-10-02T10:00:00Z"}
		}
	}
	err := EjecutarDorado(obligacionDeDosHitos(), d)
	if err == nil {
		t.Fatal("HALLAZGO: un hito que el motor deja pendiente de hecho se puede declarar con " +
			"fecha y el dorado pasa. Entonces el dorado no dice nada del estado, que es la " +
			"mitad de lo que ve el operador")
	}
	if !strings.Contains(err.Error(), "pendiente de hecho") {
		t.Fatalf("el error tiene que nombrar los dos estados: %v", err)
	}
}

// EL EMPAREJAMIENTO ES POR HITO, NO POR ORDEN (invariante 7). Se demuestra
// tumbando la propiedad contraria: si casara por posicion, reordenar las filas
// cambiaria el veredicto, e intercambiar las fechas entre dos hitos pasaria.
func TestElEmparejamientoEsPorHitoYNoPorPosicion(t *testing.T) {
	o := Obligacion{
		ID: "demo.dos_fechas", Cita: "c", ClaseE2E: "notificatoria",
		Temporalidad: &Temporalidad{
			Primitiva: "plazo", Regimen: RegimenSpec{Computo: "naturales", Cierre: "exacto"},
			Disparador: map[string]string{"hecho": "conocimiento"},
			Hitos: []HitoSpec{
				{ID: "pronto", Limite: "PT24H"},
				{ID: "tarde", Limite: "PT72H"},
			},
		},
	}
	base := Dorado{Caso: "dos plazos", Obligacion: "demo.dos_fechas",
		Hechos:          map[string]string{"conocimiento": "2026-09-01T10:00:00Z"},
		CitaDelEsperado: "los dos plazos del mismo articulo"}
	enOrden := base
	enOrden.Esperado = []EsperadoDorado{
		{Hito: "pronto", Vence: "2026-09-02T10:00:00Z"},
		{Hito: "tarde", Vence: "2026-09-04T10:00:00Z"},
	}
	alReves := base
	alReves.Esperado = []EsperadoDorado{
		{Hito: "tarde", Vence: "2026-09-04T10:00:00Z"},
		{Hito: "pronto", Vence: "2026-09-02T10:00:00Z"},
	}
	if err := EjecutarDorado(o, enOrden); err != nil {
		t.Fatalf("en orden tiene que pasar: %v", err)
	}
	if err := EjecutarDorado(o, alReves); err != nil {
		t.Fatalf("HALLAZGO: reordenar las filas cambia el veredicto, o sea que el "+
			"emparejamiento va por posicion y nadie firma un orden: %v", err)
	}
	// Y la direccion que lo prueba de verdad: las MISMAS dos fechas cruzadas
	// entre los dos hitos. Por posicion pasaria; por hito tiene que caer.
	cruzado := base
	cruzado.Esperado = []EsperadoDorado{
		{Hito: "pronto", Vence: "2026-09-04T10:00:00Z"},
		{Hito: "tarde", Vence: "2026-09-02T10:00:00Z"},
	}
	if err := EjecutarDorado(o, cruzado); err == nil {
		t.Fatal("HALLAZGO: intercambiar las fechas de dos hitos pasa, o sea que se compara " +
			"el multiconjunto de fechas y no quien vence cuando")
	}
}

// LAS DOS FORMAS DE LA NADA (invariante 8), por separado y con mensajes
// distintos, porque son fallos distintos.
func TestElEsperadoNilYElEsperadoVacioSeRechazanPorSeparado(t *testing.T) {
	p := base()
	p.Obligaciones[0].Temporalidad = &Temporalidad{Primitiva: "periodica", Cadencia: "P24M",
		Regimen: RegimenSpec{Computo: "naturales", Cierre: "fin_de_dia"}}
	con := func(esp []EsperadoDorado) []error {
		q := *p
		q.Dorados = []Dorado{{Caso: "sin nada", Obligacion: p.Obligaciones[0].ID,
			Esperado: esp, CitaDelEsperado: "x"}}
		return q.Validar()
	}
	buscar := func(errs []error, trozo string) bool {
		for _, err := range errs {
			if strings.Contains(err.Error(), trozo) {
				return true
			}
		}
		return false
	}
	// nil: la forma peligrosa, la que sale sola por olvido.
	if !buscar(con(nil), "sin esperado") {
		t.Errorf("HALLAZGO: un dorado con el esperado AUSENTE carga, no afirma nada y aun asi "+
			"cuenta para el minimo de tres por reloj: %v", con(nil))
	}
	// vacio presente: la que parece legitima ("el motor no devuelve nada") y no
	// lo es, porque se cumple sola.
	if !buscar(con([]EsperadoDorado{}), "esperado vacio") {
		t.Errorf("HALLAZGO: un esperado [] carga. Con una periodica se cumple SOLO: sin "+
			"fechas declaradas el horizonte es cero y el motor no devuelve ocurrencias: %v",
			con([]EsperadoDorado{}))
	}
	// Y el ejecutor tampoco se los traga, porque tambien se le llama con
	// dorados que no han pasado por el linter.
	for nombre, esp := range map[string][]EsperadoDorado{"nil": nil, "vacio": {}} {
		d := Dorado{Caso: "sin nada", Obligacion: "demo.auditoria_bienal", Esperado: esp,
			Hechos: map[string]string{"ultima": "2025-03-10"}, CitaDelEsperado: "x"}
		o := Obligacion{ID: "demo.auditoria_bienal", Cita: "c", ClaseE2E: "procedimental",
			Temporalidad: &Temporalidad{Primitiva: "periodica", Cadencia: "P24M",
				Regimen:    RegimenSpec{Computo: "naturales", Cierre: "fin_de_dia"},
				Disparador: map[string]string{"hecho": "ultima"}}}
		if err := EjecutarDorado(o, d); err == nil {
			t.Errorf("HALLAZGO: el ejecutor da verde con el esperado %s, comparando el vacio "+
				"contra el vacio", nombre)
		}
	}
}

// El hito repetido dentro de un esperado: una de las dos filas no se compararia
// contra nada y quedaria verde diga lo que diga.
func TestUnHitoRepetidoEnElEsperadoSeRechaza(t *testing.T) {
	d := doradoAgravado()
	d.Esperado = append(d.Esperado, EsperadoDorado{
		Hito: "notificacion_agravada", Vence: "2030-01-01T00:00:00Z"})
	if err := EjecutarDorado(obligacionDeDosHitos(), d); err == nil {
		t.Fatal("HALLAZGO: dos filas con el mismo hito, y una de ellas dice 2030")
	}
	p := base()
	p.Dorados = []Dorado{{Caso: "repetido", Obligacion: p.Obligaciones[0].ID,
		CitaDelEsperado: "x", Esperado: []EsperadoDorado{
			{Hito: "h", Vence: "2027-01-01T00:00:00Z"},
			{Hito: "h", Vence: "2028-01-01T00:00:00Z"},
		}}}
	if errs := p.Validar(); len(errs) == 0 {
		t.Fatal("el linter tambien tiene que rechazar el hito repetido")
	}
}

// ---------------------------------------------------------------------------
// El opt-out: una CADENA con el motivo, nunca un booleano.
// ---------------------------------------------------------------------------

func TestElMotivoDeSubconjuntoTieneSueloYNoEsUnBooleano(t *testing.T) {
	p := base()
	con := func(motivo string) []error {
		q := *p
		q.Dorados = []Dorado{{Caso: "subconjunto", Obligacion: p.Obligaciones[0].ID,
			CitaDelEsperado: "x", SubconjuntoPorque: motivo,
			Esperado: []EsperadoDorado{{Hito: "h", Vence: "2027-01-01T00:00:00Z"}}}}
		return q.Validar()
	}
	// El relleno que un booleano habria dejado pasar.
	for _, malo := range []string{"si", "n/a", "TODO", "porque si", strings.Repeat("z", MinimoDelMotivoDeSubconjunto-1)} {
		if errs := con(malo); len(errs) == 0 {
			t.Errorf("HALLAZGO: %q vale como motivo para renunciar a la exhaustividad", malo)
		}
	}
	// Y los espacios no cuentan: rellenar con blancos es la forma barata de
	// pasar un minimo de longitud.
	if errs := con(strings.Repeat(" ", MinimoDelMotivoDeSubconjunto+10) + "x"); len(errs) == 0 {
		t.Error("HALLAZGO: un motivo de blancos pasa el suelo")
	}
	// Control negativo: un motivo de verdad, justo en el limite, vale.
	if errs := con(strings.Repeat("z", MinimoDelMotivoDeSubconjunto)); len(errs) != 0 {
		t.Errorf("un motivo de %d caracteres tiene que valer: %v",
			MinimoDelMotivoDeSubconjunto, errs)
	}
}

// El opt-out relaja UNA sola direccion. Si relajara las dos, seria un
// "no comprobar nada" con un parrafo de coartada.
func TestElSubconjuntoSoloRelajaLaDireccionQueSobra(t *testing.T) {
	o := obligacionDeDosHitos()
	o.Temporalidad.Hitos[0].Clase = "" // el general convive: sobra una fila
	motivo := "este caso solo afirma el plazo agravado del apartado 4; el general lo cubre su " +
		"propio dorado"
	d := doradoAgravado()
	d.SubconjuntoPorque = motivo
	if err := EjecutarDorado(o, d); err != nil {
		t.Fatalf("con el motivo escrito, lo que SOBRA deja de fallar: %v", err)
	}
	// Pero lo que FALTA sigue fallando: el subconjunto no es una amnistia.
	d.Esperado = append(d.Esperado, EsperadoDorado{
		Hito: "hito_que_no_existe", Vence: "2026-09-02T10:00:00Z"})
	if err := EjecutarDorado(o, d); err == nil {
		t.Fatal("HALLAZGO: el subconjunto tambien tapa lo que falta, o sea que no comprueba nada")
	}
	// Y las fechas de las filas declaradas se siguen exigiendo exactas.
	d.Esperado = []EsperadoDorado{{Hito: "notificacion_agravada", Vence: "2026-09-02T11:00:00Z"}}
	if err := EjecutarDorado(o, d); err == nil {
		t.Fatal("HALLAZGO: el subconjunto tambien perdona una fecha equivocada")
	}
}

// ---------------------------------------------------------------------------
// La forma vieja no carga, y el error dice como se migra.
// ---------------------------------------------------------------------------

func TestLaFormaViejaDelEsperadoNoCargaYDiceComoSeMigra(t *testing.T) {
	raiz := t.TempDir()
	dir := filepath.Join(raiz, "demo")
	if err := os.MkdirAll(filepath.Join(dir, "pruebas"), 0o755); err != nil {
		t.Fatal(err)
	}
	manifiesto := `{"urn":"urn:demo:vieja","version":"1","clase":4,"licencia":"Apache-2.0",
	  "licencia_fuente":"del-proyecto","atribucion":"Datos propios del proyecto plazum.",
	  "identificador":{"tipo":"sin-identificador","valor":"demo-forma-vieja",
	    "motivo":"paquete sintetico de prueba, no lo publica ningun editor"},
	  "vigencia":{"desde":"2026-01-01"},
	  "obligaciones":[{"id":"o","articulo":"1","cita":"c","clase_e2e":"procedimental",
	    "vigencia":{"desde":"2026-01-01"}}]}`
	if err := os.WriteFile(filepath.Join(dir, "paquete.json"), []byte(manifiesto), 0o600); err != nil {
		t.Fatal(err)
	}
	// La forma VIEJA: esperado como objeto.
	viejo := `[{"caso":"el de siempre","obligacion":"o","hechos":{"x":"2026-01-01"},
	  "esperado":{"vence":"2026-01-11T23:59:59Z","hito":"h"},"cita_del_esperado":"art. 1"}]`
	ruta := filepath.Join(dir, "pruebas", "caso.json")
	if err := os.WriteFile(ruta, []byte(viejo), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := Cargar(raiz)
	if err == nil {
		t.Fatal("HALLAZGO: la forma vieja del esperado sigue cargando. Un formato con dos " +
			"formas vivas es un formato en el que gana la floja, y la floja aqui es la que " +
			"no dice nada de lo que NO tiene que salir")
	}
	if !errors.Is(err, ErrEsperadoDeLaFormaVieja) {
		t.Fatalf("el error tiene que ser el centinela de la migracion, no un fallo de tipos "+
			"de encoding/json: %v", err)
	}
	for _, trozo := range []string{"el de siempre", "\"esperado\": [", "hito"} {
		if !strings.Contains(err.Error(), trozo) {
			t.Errorf("el error de migracion no contiene %q: %v", trozo, err)
		}
	}
	// Control negativo: la forma nueva del MISMO caso si carga.
	nuevo := `[{"caso":"el de siempre","obligacion":"o","hechos":{"x":"2026-01-01"},
	  "esperado":[{"hito":"h","vence":"2026-01-11T23:59:59Z"}],"cita_del_esperado":"art. 1"}]`
	if err := os.WriteFile(ruta, []byte(nuevo), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Cargar(raiz); err != nil {
		t.Fatalf("la forma nueva tiene que cargar: %v", err)
	}
}

// Un fichero de pruebas que no es ni un dorado ni una lista se dice como lo que
// es, en vez de con el error del primero de dos intentos encadenados.
func TestUnFicheroDePruebasQueNoEsNiDoradoNiListaLoDice(t *testing.T) {
	if _, err := leerFicheroDeDorados([]byte(`"soy una cadena"`)); err == nil {
		t.Fatal("una cadena suelta no es un fichero de pruebas")
	}
	// Un dorado suelto (sin corchetes) sigue valiendo: es la forma que usan los
	// paquetes de un solo caso.
	ds, err := leerFicheroDeDorados([]byte(`{"caso":"uno","esperado":[{"hito":"h","vence":"2026-01-01"}]}`))
	if err != nil || len(ds) != 1 {
		t.Fatalf("un dorado suelto tiene que cargar: %v %v", ds, err)
	}
}

// ---------------------------------------------------------------------------
// El estado que se declara sale del vocabulario del motor, y solo de ahi.
// ---------------------------------------------------------------------------

func TestUnEstadoInventadoEnElEsperadoSeRechaza(t *testing.T) {
	d := doradoAgravado()
	d.Esperado[1].Estado = "pendiente" // le falta "de hecho"
	err := EjecutarDorado(obligacionDeDosHitos(), d)
	if err == nil {
		t.Fatal("HALLAZGO: un estado que el motor no conoce se acepta, y entonces la " +
			"comparacion de estados no compara nada")
	}
	if !errors.Is(err, ventana.ErrEstadoDesconocido) {
		t.Fatalf("el error tiene que ser el centinela del vocabulario: %v", err)
	}
	// Los tres nombres del motor SI valen, y son los que String() imprime.
	for _, e := range []ventana.EstadoVenc{ventana.Determinado, ventana.PendienteDeHecho, ventana.SinPlazoLegal} {
		if _, err := ventana.ParseEstadoVenc(e.String()); err != nil {
			t.Errorf("%q lo imprime el motor y no se puede leer: %v", e.String(), err)
		}
	}
}

// Declarar determinado sin fecha, o un estado sin fecha CON fecha, es afirmar
// algo que el motor no dice. Las dos direcciones.
func TestLaFechaYElEstadoTienenQueSerCoherentes(t *testing.T) {
	sinFecha := doradoAgravado()
	sinFecha.Esperado[0].Vence = ""
	if err := EjecutarDorado(obligacionDeDosHitos(), sinFecha); err == nil {
		t.Error("HALLAZGO: determinado sin fecha pasa, y determinado quiere decir que hay fecha")
	}
	conFecha := doradoAgravado()
	conFecha.Esperado[1].Vence = "2026-10-01T00:00:00Z"
	if err := EjecutarDorado(obligacionDeDosHitos(), conFecha); err == nil {
		t.Error("HALLAZGO: pendiente de hecho CON fecha pasa, y un vencimiento que no esta " +
			"determinado no tiene fecha")
	}
}
