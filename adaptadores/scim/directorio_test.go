package scim

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"
)

// El instante fijo. Igual que en el resto del proyecto: el instante entra como
// dato, y asi un usuario desactivado el martes se prueba sin esperar al martes.
var ahoraFijo = time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC)

// secretosDeterministas es el puertos.Secretos de los tests. Escrito aqui, no
// importado del test de otro frente.
type secretosDeterministas struct {
	mu sync.Mutex
	n  uint64
}

func (s *secretosDeterministas) Bytes(b []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := 0; i < len(b); {
		s.n++
		var sem [8]byte
		binary.BigEndian.PutUint64(sem[:], s.n)
		h := sha256.Sum256(sem[:])
		i += copy(b[i:], h[:])
	}
	return nil
}

func (s *secretosDeterministas) Token(n int) (string, error) {
	b := make([]byte, n)
	if err := s.Bytes(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func nuevoDirectorioDePrueba(t *testing.T) *Directorio {
	t.Helper()
	d, err := NuevoDirectorio(&secretosDeterministas{})
	if err != nil {
		t.Fatal(err)
	}
	return d
}

// crear da de alta a alguien con lo minimo, y falla el test si no se puede.
func crear(t *testing.T, d *Directorio, userName string) Usuario {
	t.Helper()
	u, err := d.Crear(Usuario{UserName: userName, Mostrar: userName, Activo: true}, ahoraFijo)
	if err != nil {
		t.Fatalf("crear %s: %v", userName, err)
	}
	return u
}

// ---------------------------------------------------------------------------
// Ciclo de vida
// ---------------------------------------------------------------------------

// TestElCicloDeVidaCompleto: crear, actualizar, desactivar y borrar.
func TestElCicloDeVidaCompleto(t *testing.T) {
	d := nuevoDirectorioDePrueba(t)

	u, err := d.Crear(Usuario{
		UserName:   "ana@ejemplo.es",
		ExternalID: "ext-ana",
		Mostrar:    "Ana Ejemplo",
		Correos:    []Correo{{Valor: "ana@ejemplo.es", Tipo: "work", Principal: true}},
		Activo:     true,
	}, ahoraFijo)
	if err != nil {
		t.Fatalf("crear: %v", err)
	}
	if u.ID == "" {
		t.Fatal("el usuario creado no tiene id")
	}
	if !u.Creado.Equal(ahoraFijo) {
		t.Errorf("creado %s, se esperaba %s", u.Creado, ahoraFijo)
	}

	// PATCH: cambia el titulo.
	despues := ahoraFijo.Add(time.Hour)
	u2, err := d.Parchear(u.ID, parcheo(`[{"op":"replace","path":"title","value":"Responsable de seguridad"}]`), despues)
	if err != nil {
		t.Fatalf("parchear: %v", err)
	}
	if u2.Titulo != "Responsable de seguridad" {
		t.Errorf("titulo %q", u2.Titulo)
	}
	if !u2.Modificado.Equal(despues) {
		t.Errorf("no se actualizo la fecha de modificacion")
	}

	// Desactivar.
	u3, err := d.Parchear(u.ID, parcheo(`[{"op":"replace","path":"active","value":false}]`), despues)
	if err != nil {
		t.Fatalf("desactivar: %v", err)
	}
	if u3.Activo {
		t.Fatal("el usuario sigue activo despues de `active: false`")
	}
	if !u3.Vivo() {
		t.Fatal("desactivar no es borrar: el recurso sigue existiendo")
	}
	// Y desactivado no entra.
	if err := d.PuedeEntrar("ext-ana", ""); err == nil {
		t.Fatal("un usuario desactivado en el IdP pudo entrar. Eso es la mitad del " +
			"offboarding, y la peor")
	}

	// Borrar.
	if err := d.Borrar(u.ID, despues); err != nil {
		t.Fatalf("borrar: %v", err)
	}
	if _, err := d.Leer(u.ID); err == nil {
		t.Fatal("un usuario borrado sigue devolviendose por GET, y RFC 7644 dice que no")
	}
	us, total, err := d.Listar(Filtro{})
	if err != nil {
		t.Fatal(err)
	}
	if total != 0 || len(us) != 0 {
		t.Fatalf("un usuario borrado sigue en la lista (%d)", total)
	}
	// Pero el rastro se conserva, que es lo que hace visible la obligacion
	// huerfana.
	h, ok := d.Historico(u.ID)
	if !ok {
		t.Fatal("se perdio el rastro del usuario borrado: una obligacion suya se quedaria " +
			"apuntando a un id que no resuelve, y en pantalla eso se lee como un fallo del " +
			"sistema en vez de como una obligacion sin dueno")
	}
	if h.Borrado.IsZero() {
		t.Error("el borrado no quedo fechado")
	}
}

// TestUnUsuarioActivoSiEntra es el control negativo del bloqueo: si el rechazo
// no se contrasta con una aceptacion, "no entra nadie" pasa el test igual.
func TestUnUsuarioActivoSiEntra(t *testing.T) {
	d := nuevoDirectorioDePrueba(t)
	if _, err := d.Crear(Usuario{
		UserName: "ana@ejemplo.es", ExternalID: "ext-ana", Activo: true,
	}, ahoraFijo); err != nil {
		t.Fatal(err)
	}
	if err := d.PuedeEntrar("ext-ana", ""); err != nil {
		t.Fatalf("un usuario activo tiene que poder entrar: %v", err)
	}
	err := d.PuedeEntrar("no-existe", "")
	if err == nil {
		t.Fatal("alguien sin cuenta aprovisionada entro. Con eso, cualquier usuario del " +
			"tenant del IdP entra en el GRC de la empresa")
	}
	// El MOTIVO importa tanto como el rechazo. HALLAZGO del barrido de
	// mutacion: al borrar la comprobacion de existencia, el rechazo seguia
	// ocurriendo (el usuario cero no esta activo) pero el mensaje pasaba a
	// decir "la cuenta esta desactivada en el IdP", y el administrador se iba
	// a buscar a esa persona al IdP en vez de aprovisionarla.
	if !strings.Contains(err.Error(), "aprovisionada") {
		t.Errorf("se rechaza por el motivo equivocado, y eso manda al administrador a "+
			"buscar donde no es: %v", err)
	}
}

// TestUnUsuarioBorradoNoEntra.
func TestUnUsuarioBorradoNoEntra(t *testing.T) {
	d := nuevoDirectorioDePrueba(t)
	u, err := d.Crear(Usuario{UserName: "ana@ejemplo.es", ExternalID: "ext-ana", Activo: true}, ahoraFijo)
	if err != nil {
		t.Fatal(err)
	}
	if err := d.PuedeEntrar("ext-ana", ""); err != nil {
		t.Fatalf("CONTROL NEGATIVO EN ROJO: %v", err)
	}
	if err := d.Borrar(u.ID, ahoraFijo); err != nil {
		t.Fatal(err)
	}
	err = d.PuedeEntrar("ext-ana", "")
	if err == nil {
		t.Fatal("un usuario borrado del IdP pudo entrar")
	}
	if !strings.Contains(err.Error(), "2026-08-25") {
		t.Errorf("el motivo no dice cuando se borro: %v", err)
	}
}

// TestElUserNameEsUnico. Sin esto, dos altas del mismo usuario crean dos
// cuentas y desactivar una deja la otra viva.
func TestElUserNameEsUnico(t *testing.T) {
	d := nuevoDirectorioDePrueba(t)
	crear(t, d, "ana@ejemplo.es")
	_, err := d.Crear(Usuario{UserName: "ANA@Ejemplo.ES", Activo: true}, ahoraFijo)
	if err == nil {
		t.Fatal("se creo un segundo usuario con el mismo `userName` cambiando mayusculas. " +
			"SCIM declara `userName` como caseExact false, y dos cuentas para la misma " +
			"persona significa que desactivar una deja la otra viva")
	}
	e, ok := err.(*Error)
	if !ok || e.Estado != 409 || e.Tipo != "uniqueness" {
		t.Errorf("tenia que ser un 409 de uniqueness y fue: %v", err)
	}
}

// TestElAltaConcurrenteDelMismoUsuarioDaUnaSola. Entra ID manda peticiones en
// paralelo.
func TestElAltaConcurrenteDelMismoUsuarioDaUnaSola(t *testing.T) {
	d := nuevoDirectorioDePrueba(t)
	var wg sync.WaitGroup
	var mu sync.Mutex
	exitos := 0
	for n := 0; n < 32; n++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := d.Crear(Usuario{UserName: "ana@ejemplo.es", Activo: true}, ahoraFijo); err == nil {
				mu.Lock()
				exitos++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if exitos != 1 {
		t.Fatalf("32 altas simultaneas del mismo usuario dieron %d exitos; tenia que ser 1", exitos)
	}
}

// TestElFiltroNoAdmitidoNoDevuelveLaListaEntera.
//
// Es el fallo que rompe el aprovisionamiento en silencio: el IdP pregunta
// "¿existe ya alguien con este userName?", si se le devuelve la lista entera
// concluye que si, y deja de crear a nadie.
func TestElFiltroNoAdmitidoNoDevuelveLaListaEntera(t *testing.T) {
	d := nuevoDirectorioDePrueba(t)
	crear(t, d, "ana@ejemplo.es")
	crear(t, d, "luis@ejemplo.es")

	if _, _, err := d.Listar(Filtro{Atributo: "nickname", Valor: "x"}); err == nil {
		t.Fatal("un filtro sobre un atributo que no se sabe filtrar devolvio resultados en " +
			"vez de decir que no lo entiende")
	}
	// Control negativo: el filtro que SI se admite filtra de verdad.
	us, total, err := d.Listar(Filtro{Atributo: "username", Valor: "ana@ejemplo.es"})
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(us) != 1 || us[0].UserName != "ana@ejemplo.es" {
		t.Fatalf("el filtro por userName devolvio %d resultados: %+v", total, us)
	}
}

// ---------------------------------------------------------------------------
// PATCH
// ---------------------------------------------------------------------------

func parcheo(operaciones string) Parcheo {
	var ops []Operacion
	if err := json.Unmarshal([]byte(operaciones), &ops); err != nil {
		panic(err)
	}
	return Parcheo{Esquemas: []string{EsquemaParcheo}, Operaciones: ops}
}

// TestUnPatchNoEscalaPrivilegios es la tabla de la pasada del atacante sobre
// PATCH.
//
// Cada fila es una operacion que, aceptada, daria mas de lo que el
// aprovisionamiento tiene que poder dar. Cada una lleva su control negativo: el
// PATCH honrado equivalente si se aplica.
func TestUnPatchNoEscalaPrivilegios(t *testing.T) {
	casos := []struct {
		nombre     string
		operacion  string
		espera     string
		tipo       string
		esperaEsta string
	}{
		{
			nombre:    "roles, que es como el IdP intentaria dar privilegios",
			operacion: `[{"op":"add","path":"roles","value":[{"value":"admin"}]}]`,
			espera:    "el rol dentro de plazum se asigna dentro de plazum",
			tipo:      "mutability",
		},
		{
			nombre:    "roles sin path, dentro de un objeto de atributos",
			operacion: `[{"op":"replace","value":{"roles":[{"value":"admin"}]}}]`,
			espera:    "el rol dentro de plazum se asigna dentro de plazum",
			tipo:      "mutability",
		},
		{
			nombre:    "entitlements",
			operacion: `[{"op":"add","path":"entitlements","value":[{"value":"todo"}]}]`,
			espera:    "los privilegios no entran por el aprovisionamiento",
			tipo:      "mutability",
		},
		{
			nombre:    "password, que crearia una segunda via de entrada",
			operacion: `[{"op":"replace","path":"password","value":"hunter2"}]`,
			espera:    "segunda via de entrada que nadie vigila",
			tipo:      "mutability",
		},
		{
			nombre:    "id, que es inmutable",
			operacion: `[{"op":"replace","path":"id","value":"el-id-que-yo-quiera"}]`,
			espera:    "lo asigna plazum y es inmutable",
			tipo:      "mutability",
		},
		{
			nombre:    "groups, que es de solo lectura en el esquema User",
			operacion: `[{"op":"add","path":"groups","value":[{"value":"grupo-de-admins"}]}]`,
			espera:    "de solo lectura en el esquema User",
			tipo:      "mutability",
		},
		{
			nombre:    "meta",
			operacion: `[{"op":"replace","path":"meta","value":{"created":"1970-01-01T00:00:00Z"}}]`,
			espera:    "de solo lectura",
			tipo:      "mutability",
		},
		{
			// El texto que se exige es el que ENUMERA los atributos, no el
			// generico "no se sabe escribir". HALLAZGO del barrido de
			// mutacion: borrar la lista blanca dejaba el rechazo en pie (lo
			// hacia el ultimo return de la funcion) con un mensaje distinto y
			// mucho peor, y el test no notaba nada. Un mensaje que no dice
			// cuales son los atributos validos obliga al administrador a
			// adivinar.
			nombre:    "un atributo que no existe, ignorado en silencio seria peor",
			operacion: `[{"op":"replace","path":"esteAtributoNoExiste","value":"x"}]`,
			espera:    "Los atributos que plazum guarda son",
			tipo:      "invalidPath",
		},
		{
			nombre:    "remove sin path, que borraria el recurso entero",
			operacion: `[{"op":"remove"}]`,
			espera:    "RFC 7644 lo prohibe",
			tipo:      "invalidPath",
		},
		{
			nombre:    "una op que no existe",
			operacion: `[{"op":"destroy","path":"active","value":false}]`,
			espera:    "solo se admiten add, replace y remove",
			tipo:      "invalidSyntax",
		},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			d := nuevoDirectorioDePrueba(t)
			u := crear(t, d, "ana@ejemplo.es")

			_, err := d.Parchear(u.ID, parcheo(c.operacion), ahoraFijo)
			if err == nil {
				t.Fatalf("el PATCH (%s) se aplico", c.nombre)
			}
			if !strings.Contains(err.Error(), c.espera) {
				t.Errorf("se rechazo por otro motivo: %v\nse esperaba %q", err, c.espera)
			}
			if e, ok := err.(*Error); ok && e.Tipo != c.tipo {
				t.Errorf("scimType %q, se esperaba %q. El IdP decide si reintentar mirando "+
					"esto: con el tipo mal, un choque recuperable se vuelve un fallo permanente",
					e.Tipo, c.tipo)
			}

			// Control negativo: el PATCH honrado del mismo usuario SI entra.
			if _, err := d.Parchear(u.ID, parcheo(`[{"op":"replace","path":"title","value":"Analista"}]`),
				ahoraFijo); err != nil {
				t.Fatalf("CONTROL NEGATIVO EN ROJO: un PATCH legitimo tambien se rechaza (%v). "+
					"El caso de arriba no prueba nada", err)
			}
		})
	}
}

// TestElPatchEsAtomico. RFC 7644 seccion 3.5.2: o entran todas las operaciones
// o no entra ninguna. Aplicar la mitad deja al IdP creyendo que fallo entero y
// al directorio con la mitad puesta.
func TestElPatchEsAtomico(t *testing.T) {
	d := nuevoDirectorioDePrueba(t)
	u := crear(t, d, "ana@ejemplo.es")

	_, err := d.Parchear(u.ID, parcheo(`[
		{"op":"replace","path":"title","value":"Este cambio no puede quedarse"},
		{"op":"add","path":"roles","value":[{"value":"admin"}]}
	]`), ahoraFijo)
	if err == nil {
		t.Fatal("el PATCH con una operacion prohibida se aplico entero")
	}
	tras, err := d.Leer(u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if tras.Titulo != "" {
		t.Fatalf("el titulo quedo en %q: la primera operacion se aplico aunque la segunda "+
			"fallo. Un PATCH a medias es lo peor de los dos mundos", tras.Titulo)
	}
}

// TestElPatchDeEntraIDConActiveEnCadena. Entra ID manda `"False"` como cadena.
// Rechazarlo romperia el offboarding por una comilla.
func TestElPatchDeEntraIDConActiveEnCadena(t *testing.T) {
	d := nuevoDirectorioDePrueba(t)
	u := crear(t, d, "ana@ejemplo.es")
	tras, err := d.Parchear(u.ID, parcheo(`[{"op":"replace","value":{"active":"False"}}]`), ahoraFijo)
	if err != nil {
		t.Fatalf("Entra ID manda `active` como cadena y hay que aceptarlo: %v", err)
	}
	if tras.Activo {
		t.Fatal("`active: \"False\"` no desactivo")
	}
	// Y una cadena que no es ni true ni false sigue siendo un error.
	if _, err := d.Parchear(u.ID, parcheo(`[{"op":"replace","path":"active","value":"quizas"}]`),
		ahoraFijo); err == nil {
		t.Fatal("`active: \"quizas\"` se acepto")
	}
}

// TestLasTresFormasDeLaRutaDelManagerLleganAlMismoSitio.
//
// Es interoperabilidad, no purismo: si solo se entiende una, el atributo que
// sostiene el escalado funciona o no segun el IdP que toque.
func TestLasTresFormasDeLaRutaDelManagerLleganAlMismoSitio(t *testing.T) {
	rutas := []string{
		"manager",
		EsquemaEmpresa + ":manager",
		EsquemaEmpresa + ".manager",
		strings.ToUpper(EsquemaEmpresa) + ":Manager",
	}
	for _, ruta := range rutas {
		t.Run(ruta, func(t *testing.T) {
			d := nuevoDirectorioDePrueba(t)
			jefa := crear(t, d, "jefa@ejemplo.es")
			ana := crear(t, d, "ana@ejemplo.es")
			ops := []Operacion{{Op: "replace", Ruta: ruta,
				Valor: json.RawMessage(`{"value":"` + jefa.ID + `"}`)}}
			tras, err := d.Parchear(ana.ID, Parcheo{Operaciones: ops}, ahoraFijo)
			if err != nil {
				t.Fatalf("la ruta %q no llego: %v", ruta, err)
			}
			if tras.ManagerIdP != jefa.ID {
				t.Fatalf("el manager quedo en %q", tras.ManagerIdP)
			}
		})
	}
}

// TestElManagerLlegaComoCadenaOComoReferencia.
func TestElManagerLlegaComoCadenaOComoReferencia(t *testing.T) {
	for _, forma := range []string{`"%s"`, `{"value":"%s"}`, `{"value":"%s","displayName":"La Jefa"}`} {
		d := nuevoDirectorioDePrueba(t)
		jefa := crear(t, d, "jefa@ejemplo.es")
		ana := crear(t, d, "ana@ejemplo.es")
		valor := strings.Replace(forma, "%s", jefa.ID, 1)
		valor = strings.Replace(valor, "%s", jefa.ID, 1)
		ops := []Operacion{{Op: "replace", Ruta: "manager", Valor: json.RawMessage(valor)}}
		tras, err := d.Parchear(ana.ID, Parcheo{Operaciones: ops}, ahoraFijo)
		if err != nil {
			t.Fatalf("la forma %s no se acepto: %v", forma, err)
		}
		if tras.ManagerIdP != jefa.ID {
			t.Fatalf("con la forma %s el manager quedo en %q", forma, tras.ManagerIdP)
		}
	}
}
