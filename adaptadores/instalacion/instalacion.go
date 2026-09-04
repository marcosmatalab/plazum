// Package instalacion guarda QUIEN ES la instalacion: como se llama la
// organizacion y con que nombre hablan de ella las reglas de aplicabilidad.
//
// # Que hueco cierra, y por que es el que bloquea una fecha
//
// El TTFV del camino guiado esta en 20m27s contra un presupuesto de 15m0s, y su
// cuello no es la entrevista: son **5 ordenes de terminal, 7m30s, el 37 %**, que
// las pantallas piden en sus estados vacios. Dos de ellas (las del calendario y
// el plan de avisos) existen porque `serve` solo sabe componer un alcance si el
// operador le pasa `--alcance fichero.json`, y ese fichero necesita dos datos
// que la entrevista NO pregunta: el nombre de la organizacion y el sujeto.
//
// Aqui viven esos dos datos, para que se pregunten UNA VEZ en el navegador en
// vez de teclearse en cada arranque.
//
// # POR QUE ES DE LA INSTALACION Y NO DE LA CUENTA
//
// Porque las obligaciones de una organizacion no dependen de quien las mire. Si
// el nombre y el sujeto vivieran en la cuenta, dos personas de la misma empresa
// verian dos calendarios distintos y ninguna de las dos sabria cual es el bueno.
// Es la misma razon por la que el acta se niega a componerse sin organizacion:
// un documento de cumplimiento que no dice de quien es no es evidencia de nadie.
//
// # EL SUJETO SE ACUNA UNA VEZ Y NO SE MUEVE, que es la decision que importa
//
// El sujeto es el identificador con el que las reglas hablan de la organizacion:
// todas las respuestas de la entrevista cuelgan de el como instancia. Derivarlo
// del nombre en cada lectura seria comodo y estaria mal, porque **cambiar el
// nombre moveria el sujeto**, y con el sujeto se mueven las derivaciones, las
// citas y lo que ya este escrito en un expediente.
//
// Asi que se acuna la primera vez, se guarda, y a partir de ahi el nombre se
// puede cambiar y el sujeto NO. Renombrar la empresa es un hecho normal;
// re-derivar su cumplimiento entero por haberla renombrado, no.
//
// # LAS TRES FORMAS DE LA NADA (invariante 8)
//
//	fichero AUSENTE           nadie ha configurado la instalacion todavia. Es el
//	                          unico caso en que la nada es de verdad, y el que
//	                          tiene una instalacion recien descargada.
//	fichero PRESENTE Y VACIO  error. Cero bytes donde deberia haber identidad es
//	                          una escritura cortada, y leerlo como «sin
//	                          configurar» reabriria la pregunta en un sistema ya
//	                          configurado, que es como se acuna un sujeto
//	                          distinto encima de uno que ya tenia datos colgando.
//	PRESENTE Y NO
//	INTERPRETABLE             error, siempre. Una version desconocida, un nombre
//	                          en blanco, un sujeto que no tiene forma de
//	                          identificador. Son datos que HAY y no se entienden.
//
// # Lo que este paquete NO promete
//
// No es evidencia y no va firmado, igual que el almacen de alcances. Es
// configuracion de la instalacion, no un hecho oponible a un auditor.
package instalacion

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode"
)

// VersionDelAlmacen es la version del formato en disco. Un fichero sin version,
// o con otra, NO se lee: ver el encabezado.
const VersionDelAlmacen = 1

// Los topes. Un nombre de organizacion es un nombre, no un documento.
const (
	// MaxLongitudDelNombre acota el nombre visible.
	MaxLongitudDelNombre = 200
	// MaxLongitudDelSujeto acota el identificador acunado.
	MaxLongitudDelSujeto = 64
)

// Los centinelas. Cada uno se comprueba con errors.Is desde quien monta.
var (
	// ErrAlmacenIlegible: el fichero existe y no es lo que dice ser.
	ErrAlmacenIlegible = errors.New("la identidad de la instalacion no se puede leer")
	// ErrAlmacenVacio: el fichero existe y no tiene ni un byte.
	ErrAlmacenVacio = errors.New("el fichero de identidad existe y esta vacio")
	// ErrVersionDesconocida: una version que este binario no sabe leer.
	ErrVersionDesconocida = errors.New("version del fichero de identidad desconocida")
	// ErrSinRuta: se pidio abrir sin decir donde vive.
	ErrSinRuta = errors.New("identidad de la instalacion sin ruta")
	// ErrNombreNoValido: el nombre de la organizacion no sirve.
	ErrNombreNoValido = errors.New("nombre de organizacion no valido")
	// ErrSujetoNoValido: el sujeto acunado no tiene forma de identificador.
	ErrSujetoNoValido = errors.New("sujeto no valido")
	// ErrSujetoYaAcunado: se ha intentado cambiar el sujeto de una instalacion
	// que ya lo tiene.
	ErrSujetoYaAcunado = errors.New("el sujeto ya esta acunado y no se mueve")
)

// Identidad es quien es esta instalacion.
type Identidad struct {
	// Organizacion es el nombre que se ensena. Se puede cambiar.
	Organizacion string
	// Sujeto es el identificador con el que las reglas hablan de ella. SE ACUNA
	// UNA VEZ Y NO SE MUEVE.
	Sujeto string
	// Acunado es cuando se configuro por primera vez. Cero significa que no hay
	// identidad, y ese cero SI es la nada: viene de no haber fichero.
	Acunado time.Time
}

// Hay dice si esta instalacion tiene identidad configurada.
//
// SE PREGUNTA POR EL SUJETO Y NO POR EL NOMBRE, porque el sujeto es el que no se
// puede quedar a medias: un nombre en blanco con sujeto acunado es un estado que
// el almacen rechaza al escribir, pero preguntar por el nombre haria que una
// futura edicion que lo vaciara pareciera «sin configurar» y reabriera la
// acunacion.
func (i Identidad) Hay() bool { return strings.TrimSpace(i.Sujeto) != "" }

// --- la forma en disco ---

type identidadEnDisco struct {
	Version      int    `json:"version"`
	Organizacion string `json:"organizacion"`
	Sujeto       string `json:"sujeto"`
	Acunado      string `json:"acunado"`
}

// Almacen es la identidad de una instalacion, respaldada por un fichero. Seguro
// para uso concurrente: lo atienden peticiones HTTP.
type Almacen struct {
	mu    sync.Mutex
	ruta  string
	ahora func() time.Time
	quien Identidad
}

// Opciones para Abrir.
type Opciones struct {
	// Ruta es obligatoria. No hay ruta por defecto aqui a proposito: quien monta
	// decide donde vive, y RutaPorDefecto es una ayuda, no un valor implicito.
	Ruta string
	// Ahora se inyecta en los tests. Sin ella, time.Now en UTC.
	Ahora func() time.Time
}

// Abrir lee la identidad de disco.
//
// EL FICHERO AUSENTE NO ES UN ERROR: es una instalacion recien descargada. Las
// otras dos formas de la nada si lo son.
func Abrir(o Opciones) (*Almacen, error) {
	ruta := strings.TrimSpace(o.Ruta)
	if ruta == "" {
		return nil, fmt.Errorf("%w: hay que decir en que fichero vive", ErrSinRuta)
	}
	ahora := o.Ahora
	if ahora == nil {
		ahora = func() time.Time { return time.Now().UTC() }
	}
	a := &Almacen{ruta: ruta, ahora: ahora}

	b, err := os.ReadFile(ruta) // #nosec G304 -- la ruta la decide quien monta
	if errors.Is(err, os.ErrNotExist) {
		return a, nil // instalacion sin configurar: la unica nada de verdad
	}
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %w", ErrAlmacenIlegible, ruta, err)
	}
	if len(b) == 0 {
		return nil, fmt.Errorf("%w: %s. Un fichero de cero bytes donde deberia estar la "+
			"identidad es una escritura cortada, y leerlo como «sin configurar» reabriria la "+
			"pregunta en un sistema que ya la contesto", ErrAlmacenVacio, ruta)
	}
	var doc identidadEnDisco
	if err := json.Unmarshal(b, &doc); err != nil {
		return nil, fmt.Errorf("%w: %s no es JSON: %w", ErrAlmacenIlegible, ruta, err)
	}
	if doc.Version == 0 {
		return nil, fmt.Errorf("%w: %s no dice con que version del formato se escribio",
			ErrAlmacenIlegible, ruta)
	}
	if doc.Version != VersionDelAlmacen {
		return nil, fmt.Errorf("%w: %s dice version %d y este binario lee la %d",
			ErrVersionDesconocida, ruta, doc.Version, VersionDelAlmacen)
	}
	nombre, err := NormalizarNombre(doc.Organizacion)
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %w", ErrAlmacenIlegible, ruta, err)
	}
	sujeto, err := NormalizarSujeto(doc.Sujeto)
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %w", ErrAlmacenIlegible, ruta, err)
	}
	// EL INSTANTE QUE NO SE ENTIENDE ES UN ERROR, NUNCA EL CERO. El cero de
	// time.Time es el ano 1, y de ahi salen marcas con cara de dato.
	acunado, err := time.Parse(time.RFC3339, strings.TrimSpace(doc.Acunado))
	if err != nil {
		return nil, fmt.Errorf("%w: %s no trae un instante de acunacion RFC3339",
			ErrAlmacenIlegible, ruta)
	}
	a.quien = Identidad{Organizacion: nombre, Sujeto: sujeto, Acunado: acunado.UTC()}
	return a, nil
}

// Ruta dice donde vive el fichero.
func (a *Almacen) Ruta() string { return a.ruta }

// Quien devuelve la identidad de esta instalacion. La copia es por valor, asi
// que quien la recibe no puede tocar la de dentro.
func (a *Almacen) Quien() Identidad {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.quien
}

// Fijar configura la instalacion por primera vez, o cambia el NOMBRE de una que
// ya lo estaba.
//
// EL SUJETO SE ACUNA EN LA PRIMERA LLAMADA Y NO SE MUEVE DESPUES. Las llamadas
// siguientes solo cambian el nombre visible.
func (a *Almacen) Fijar(organizacion string) error {
	nombre, err := NormalizarNombre(organizacion)
	if err != nil {
		return err
	}
	a.mu.Lock()
	defer a.mu.Unlock()

	nueva := a.quien
	nueva.Organizacion = nombre
	if !a.quien.Hay() {
		s, err := AcunarSujeto(nombre)
		if err != nil {
			return err
		}
		nueva.Sujeto = s
		nueva.Acunado = a.ahora().UTC()
	}
	if err := a.guardar(nueva); err != nil {
		return err
	}
	// EN MEMORIA SOLO DESPUES DE DISCO, igual que en el almacen de cuentas y en
	// el de alcances: al reves, un fallo de disco dejaria una identidad que
	// existe en este proceso y desaparece al reiniciar.
	a.quien = nueva
	return nil
}

// FijarSujeto pone el sujeto a mano, y SOLO si no hay ninguno acunado.
//
// Existe para la importacion de una instalacion que ya tenia un alcance.json
// escrito a mano con su propio sujeto: sin esto, adoptar esa instalacion
// acunaria uno nuevo y todo lo que colgara del viejo dejaria de casar. No hay
// forma de cambiarlo despues, y es a proposito.
func (a *Almacen) FijarSujeto(sujeto string) error {
	s, err := NormalizarSujeto(sujeto)
	if err != nil {
		return err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.quien.Hay() {
		return fmt.Errorf("%w: esta instalacion ya habla de si misma como %q.\n"+
			"  Cambiarlo moveria la instancia de la que cuelgan todas las respuestas, y con "+
			"ella las derivaciones y lo que ya este escrito en un expediente", ErrSujetoYaAcunado,
			a.quien.Sujeto)
	}
	nueva := a.quien
	nueva.Sujeto = s
	nueva.Acunado = a.ahora().UTC()
	if strings.TrimSpace(nueva.Organizacion) == "" {
		nueva.Organizacion = s
	}
	if err := a.guardar(nueva); err != nil {
		return err
	}
	a.quien = nueva
	return nil
}

// guardar escribe de forma atomica. Se llama con el candado tomado.
func (a *Almacen) guardar(i Identidad) error {
	doc := identidadEnDisco{
		Version: VersionDelAlmacen, Organizacion: i.Organizacion, Sujeto: i.Sujeto,
		Acunado: i.Acunado.UTC().Format(time.RFC3339),
	}
	b, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("no puedo serializar la identidad: %w", err)
	}
	b = append(b, '\n')

	dir := filepath.Dir(a.ruta)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("no puedo crear el directorio %s: %w", dir, err)
	}
	tmp, err := os.CreateTemp(dir, "instalacion-*.json.tmp")
	if err != nil {
		return fmt.Errorf("no puedo escribir en %s: %w", dir, err)
	}
	nombreTmp := tmp.Name()
	limpiar := func() { _ = tmp.Close(); _ = os.Remove(nombreTmp) }
	if err := os.Chmod(nombreTmp, 0o600); err != nil {
		limpiar()
		return fmt.Errorf("no puedo dejar la identidad en 0600: %w", err)
	}
	if _, err := tmp.Write(b); err != nil {
		limpiar()
		return fmt.Errorf("no puedo escribir la identidad: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		limpiar()
		return fmt.Errorf("no puedo sincronizar la identidad a disco: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(nombreTmp)
		return fmt.Errorf("no puedo cerrar el fichero de identidad: %w", err)
	}
	if err := os.Rename(nombreTmp, a.ruta); err != nil {
		_ = os.Remove(nombreTmp)
		return fmt.Errorf("no puedo dejar la identidad en %s: %w", a.ruta, err)
	}
	return nil
}

// NormalizarNombre valida el nombre visible de la organizacion.
//
// LOS INVISIBLES SE RECHAZAN, no se limpian. Un nombre con un separador de
// derecha a izquierda dentro se ensena igual y no es el mismo, y limpiarlo en
// silencio guardaria algo que la persona no escribio.
func NormalizarNombre(v string) (string, error) {
	n := strings.TrimSpace(v)
	if n == "" {
		return "", fmt.Errorf("%w: esta en blanco. El nombre de la organizacion es lo que "+
			"aparece en el acta y en el calendario, y un documento de cumplimiento que no dice "+
			"de quien es no es evidencia de nadie", ErrNombreNoValido)
	}
	if len(n) > MaxLongitudDelNombre {
		return "", fmt.Errorf("%w: mide %d caracteres y el tope son %d",
			ErrNombreNoValido, len(n), MaxLongitudDelNombre)
	}
	for _, r := range n {
		if r == '\n' || r == '\r' || r == '\t' {
			return "", fmt.Errorf("%w: lleva un salto de linea o un tabulador dentro",
				ErrNombreNoValido)
		}
		if unicode.Is(unicode.Cf, r) || (unicode.IsControl(r) && r != ' ') {
			return "", fmt.Errorf("%w: lleva un caracter invisible (U+%04X) dentro. No se "+
				"limpia en silencio: se ensena igual y no es el mismo nombre",
				ErrNombreNoValido, r)
		}
	}
	return n, nil
}

// reSujeto: minusculas, digitos, guion. Es la forma de un identificador con el
// que hablan las reglas, no la de un nombre.
var reSujeto = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)

// NormalizarSujeto valida el identificador acunado.
func NormalizarSujeto(v string) (string, error) {
	s := strings.TrimSpace(v)
	if s == "" {
		return "", fmt.Errorf("%w: esta en blanco. Sin sujeto, el motor deriva las "+
			"obligaciones de nadie y el calendario sale vacio sin decir por que",
			ErrSujetoNoValido)
	}
	if len(s) > MaxLongitudDelSujeto {
		return "", fmt.Errorf("%w: mide %d caracteres y el tope son %d",
			ErrSujetoNoValido, len(s), MaxLongitudDelSujeto)
	}
	if !reSujeto.MatchString(s) {
		return "", fmt.Errorf("%w: %q no tiene forma de identificador (minusculas, digitos y "+
			"guiones, sin empezar ni acabar en guion)", ErrSujetoNoValido, s)
	}
	return s, nil
}

// AcunarSujeto deriva un identificador del nombre de la organizacion.
//
// SE LLAMA UNA SOLA VEZ EN LA VIDA DE UNA INSTALACION, y por eso no es una
// funcion pura de conveniencia sino una decision: lo que salga de aqui es la
// instancia de la que van a colgar todas las respuestas para siempre.
//
// Si del nombre no sale nada utilizable (un nombre entero en un alfabeto que
// esta transliteracion no cubre) NO se inventa uno: es error, y quien lo vea
// puede poner el sujeto a mano con FijarSujeto. Inventarse un `organizacion-1`
// aqui seria acunar en silencio un identificador que nadie eligio.
func AcunarSujeto(nombre string) (string, error) {
	var b strings.Builder
	guionPendiente := false
	for _, r := range strings.ToLower(nombre) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			if guionPendiente && b.Len() > 0 {
				b.WriteByte('-')
			}
			guionPendiente = false
			b.WriteRune(r)
		case r >= 'à' && r <= 'ÿ':
			// Las vocales acentuadas del latin-1, que son las que aparecen en un
			// nombre de empresa espanol. Se transliteran a proposito y con una
			// tabla corta: una transliteracion Unicode completa seria una
			// dependencia, y lo que no cubra sale por el error de abajo.
			if sin := transliterar(r); sin != 0 {
				if guionPendiente && b.Len() > 0 {
					b.WriteByte('-')
				}
				guionPendiente = false
				b.WriteRune(sin)
			} else {
				guionPendiente = true
			}
		default:
			guionPendiente = true
		}
		if b.Len() >= MaxLongitudDelSujeto {
			break
		}
	}
	s := b.String()
	if s == "" {
		return "", fmt.Errorf("%w: del nombre %q no sale ningun identificador.\n"+
			"  No se inventa uno: lo que salga de aqui es la instancia de la que cuelgan todas "+
			"tus respuestas para siempre.\n"+
			"  Arreglo: dilo tu, con el sujeto que quieras usar", ErrSujetoNoValido, nombre)
	}
	return NormalizarSujeto(s)
}

// transliterar devuelve la letra sin acento, o 0 si no la conoce.
func transliterar(r rune) rune {
	const acentuadas = "àáâãäåçèéêëìíîïñòóôõöùúûüýÿ"
	const llanas = "aaaaaaceeeeiiiinooooouuuuyy"
	if i := strings.IndexRune(acentuadas, r); i >= 0 {
		return rune(llanas[len([]rune(acentuadas[:i]))])
	}
	return 0
}

// RutaPorDefecto compone donde vive el fichero dentro del directorio de datos.
func RutaPorDefecto(datos string) string {
	return filepath.Join(datos, "instalacion.json")
}
