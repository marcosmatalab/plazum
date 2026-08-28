// Package actualizador implementa puertos.Actualizador: es `plazum update`.
//
// Por que la vuelta atras esta en la interfaz y no en la documentacion. En un
// producto que vigila plazos legales, una actualizacion que no se puede
// deshacer convierte un fallo de actualizacion en un incumplimiento: la
// instancia se queda sin arrancar, nadie ve los relojes, y el plazo corre
// igual. Por eso aqui el punto de retorno no es una copia de cortesia, es parte
// del contrato, y por eso hay tres cosas que este paquete se niega a hacer:
//
//  1. Aplicar una version cuyo contenido no case con el digest que declara el
//     canal. Un binario que no es el que dice ser no se instala aunque venga
//     firmado el catalogo: el catalogo dice QUE se espera y los bytes dicen QUE
//     hay, y si no coinciden se para.
//  2. Dejar la instalacion a medias. Si la instalacion falla por lo que sea, se
//     vuelve atras sola y se dice que se ha vuelto.
//  3. Fingir que ha deshecho. Deshacer un punto que no existe, o uno cuya copia
//     no cuadra con su manifiesto, devuelve ERROR. Un operador que cree que ha
//     vuelto atras y no ha vuelto esta peor que uno que sabe que no puede.
//
// Lo que NO hace, dicho para que no se confunda con lo que hace: no migra la
// base de datos (llega con el adaptador de almacen), no reinicia el servicio
// (eso lo hace systemd o quien arranque), y no habla con la red mientras el
// canal sea de directorio.
package actualizador

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/marcosmatalab/plazum/puertos"
)

// Errores del actualizador, como centinelas. Un test que compruebe que dos
// actualizaciones a la vez no se pisan tiene que poder hacerlo con errors.Is y
// no buscando una subcadena en un mensaje que manana cambia.
var (
	// ErrOcupado: otra actualizacion tiene el cerrojo ahora mismo.
	ErrOcupado = errors.New("hay otra actualizacion en curso")
	// ErrAMedias: una actualizacion anterior no termino y no se ha reparado.
	ErrAMedias = errors.New("hay una actualizacion a medias sin reparar")
	// ErrPuntoDesconocido: se pide deshacer un punto de retorno que no existe.
	ErrPuntoDesconocido = errors.New("el punto de retorno no existe")
	// ErrPuntoCorrupto: el punto existe y su copia no cuadra con su manifiesto.
	ErrPuntoCorrupto = errors.New("el punto de retorno esta corrupto")
	// ErrDigest: lo que trae el canal no es lo que el catalogo dice que es.
	ErrDigest = errors.New("el contenido descargado no coincide con su digest")
	// ErrNombreInvalido: nombre de version o ruta que no se acepta.
	ErrNombreInvalido = errors.New("nombre invalido")
)

// Nombres de los ficheros de control. Van como constantes para que el que
// escribe y el que lee no puedan discrepar.
const (
	// DirInterno es donde vive todo lo que gestiona el actualizador. Empieza
	// por punto para no estorbar en un listado, y esta EXCLUIDO de lo que una
	// version puede tocar: un canal hostil no puede escribir sus propios
	// puntos de retorno.
	DirInterno = ".plazum"
	// FicheroVersion guarda la version instalada.
	FicheroVersion = "VERSION"
	nombreCerrojo  = "cerrojo"
	nombrePuntos   = "puntos"
	nombreMarcador = "actualizacion.json"
	nombreManif    = "manifiesto.json"
	nombreCopia    = "copia"
)

// Opciones configura el actualizador.
type Opciones struct {
	// Raiz es el directorio de la instalacion: donde estan el binario, los
	// paquetes y el fichero VERSION.
	Raiz string
	// Canal es de donde salen las versiones.
	Canal Canal
	// Ahora es el instante que se estampa en los puntos de retorno. Cero
	// significa el reloj del sistema. Entra como dato para que una prueba
	// pueda fijar la fecha de un punto de retorno.
	Ahora time.Time
}

// Actualizador implementa puertos.Actualizador.
type Actualizador struct {
	raiz  string
	canal Canal
	ahora time.Time
}

var _ puertos.Actualizador = (*Actualizador)(nil)

// Nuevo construye el actualizador.
func Nuevo(o Opciones) *Actualizador {
	if o.Ahora.IsZero() {
		o.Ahora = time.Now().UTC()
	}
	if o.Raiz == "" {
		o.Raiz = "."
	}
	return &Actualizador{raiz: o.Raiz, canal: o.Canal, ahora: o.Ahora.UTC()}
}

// Manifiesto es lo que guarda un punto de retorno: que habia antes, con su
// digest, para poder comprobar al volver que la copia sigue siendo la que se
// hizo. Sin digests, "restaurar" seria copiar bytes de vuelta sin saber si son
// los buenos.
type Manifiesto struct {
	Punto string `json:"punto"`
	Fecha string `json:"fecha"`
	// Desde es la version que estaba instalada al crear el punto. Vacio si la
	// instalacion no tenia fichero VERSION.
	Desde string `json:"desde"`
	// Hacia es la version que se iba a instalar.
	Hacia string `json:"hacia"`
	// Ficheros: ruta relativa -> digest de la copia guardada.
	Ficheros map[string]string `json:"ficheros"`
	// Nuevos son las rutas que NO existian antes de la actualizacion. Al
	// deshacer hay que BORRARLAS, no restaurarlas: si se dejaran puestas, la
	// vuelta atras dejaria ficheros de la version nueva en una instalacion que
	// dice ser la vieja, y eso es un estado que nadie ha probado nunca.
	Nuevos []string `json:"nuevos"`
}

// marcador es el rastro de una actualizacion empezada y no terminada.
type marcador struct {
	Punto   string `json:"punto"`
	Version string `json:"version"`
	Inicio  string `json:"inicio"`
	PID     int    `json:"pid"`
}

// ---------------------------------------------------------------------------
// El puerto
// ---------------------------------------------------------------------------

// Disponible consulta si hay version nueva. Devuelve version vacia si no.
func (a *Actualizador) Disponible(ctx context.Context) (string, string, error) {
	if a.canal == nil {
		return "", "", errors.New("no hay canal de actualizacion configurado; " +
			"apunta --canal al directorio o al espejo desde el que se distribuyen las versiones")
	}
	vs, err := a.canal.Catalogo(ctx)
	if err != nil {
		return "", "", err
	}
	v, ok := masNueva(vs)
	if !ok {
		return "", "", nil
	}
	actual, err := a.VersionInstalada()
	if err != nil {
		return "", "", err
	}
	if v.Version == actual {
		return "", "", nil
	}
	return v.Version, v.Notas, nil
}

// Aplicar instala una version y devuelve el identificador del punto de retorno
// que deja preparado.
//
// El orden de los pasos NO es negociable y cada uno esta antes que el siguiente
// por una razon concreta:
//
//	cerrojo      dos actualizaciones a la vez se pisarian los ficheros a medio
//	             escribir, y el punto de retorno de la segunda guardaria la
//	             instalacion medio actualizada por la primera
//	sin marcador si hay una actualizacion a medias, aplicar otra encima entierra
//	             el punto de retorno bueno
//	descarga     ANTES de tocar nada: una descarga que falla no puede dejar la
//	             instalacion a medias si todavia no se ha tocado
//	digest       antes de copiar nada: no se hace un punto de retorno para algo
//	             que no se va a instalar
//	punto        y se RELEE para comprobarlo: un punto de retorno que no se ha
//	             verificado no es un punto de retorno, es una carpeta
//	instalar     lo ultimo, y si falla se vuelve atras aqui mismo
func (a *Actualizador) Aplicar(ctx context.Context, version string) (string, error) {
	if a.canal == nil {
		return "", errors.New("no hay canal de actualizacion configurado; " +
			"apunta --canal al directorio o al espejo desde el que se distribuyen las versiones")
	}
	if err := ComprobarNombreDeVersion(version); err != nil {
		return "", err
	}
	soltar, err := a.tomarCerrojo()
	if err != nil {
		return "", err
	}
	defer soltar()

	if m, hay, err := a.leerMarcador(); err != nil {
		return "", err
	} else if hay {
		return "", fmt.Errorf("%w: la actualizacion a %s empezada el %s no termino. "+
			"Ejecuta `plazum update --reparar` para volver al punto %s antes de intentar otra; "+
			"aplicar encima enterraria el unico punto de retorno bueno que queda",
			ErrAMedias, m.Version, m.Inicio, m.Punto)
	}

	vs, err := a.canal.Catalogo(ctx)
	if err != nil {
		return "", err
	}
	v, err := buscarVersion(vs, version)
	if err != nil {
		return "", err
	}
	if len(v.Ficheros) == 0 {
		return "", fmt.Errorf("la version %s no declara ningun fichero: no hay nada que "+
			"instalar. Revisa el %s del canal", version, NombreDelCatalogo)
	}

	// Descarga entera y verificada ANTES de tocar la instalacion.
	contenido := make(map[string][]byte, len(v.Ficheros))
	for _, rel := range rutasOrdenadas(v.Ficheros) {
		if err := comprobarRutaRelativa(rel); err != nil {
			return "", fmt.Errorf("%w: version %s: %w", ErrRutaInsegura, version, err)
		}
		b, err := a.canal.Traer(ctx, version, rel)
		if err != nil {
			return "", err
		}
		suma := hex.EncodeToString(sumaDe(b))
		esperado := strings.ToLower(strings.TrimSpace(v.Ficheros[rel]))
		if esperado == "" {
			return "", fmt.Errorf("la version %s no declara digest para %s. Un fichero sin "+
				"digest es un fichero que nadie ha comprobado, y no se instala", version, rel)
		}
		if suma != esperado {
			return "", fmt.Errorf("%w: %s de la version %s trae el digest %s y el catalogo "+
				"dice %s. No se instala nada. O el canal esta corrupto, o el fichero se ha "+
				"sustituido por el camino: en los dos casos la respuesta es la misma, y es no",
				ErrDigest, rel, version, suma, esperado)
		}
		contenido[rel] = b
	}

	desde, err := a.VersionInstalada()
	if err != nil {
		return "", err
	}
	punto, manif, err := a.crearPunto(v, desde)
	if err != nil {
		return "", err
	}

	if err := a.escribirMarcador(marcador{
		Punto: punto, Version: version,
		Inicio: a.ahora.Format(time.RFC3339), PID: os.Getpid(),
	}); err != nil {
		return "", err
	}

	if err := a.instalar(contenido, version); err != nil {
		// Vuelta atras inmediata, y se dice. Un error de instalacion que deja
		// la instalacion a medias y no lo cuenta es la peor de las salidas.
		if errVuelta := a.deshacer(punto); errVuelta != nil {
			return "", fmt.Errorf("la instalacion de %s fallo (%w) y ADEMAS la vuelta atras "+
				"al punto %s fallo (%v). La instalacion esta a medias: restaura desde tu copia "+
				"de seguridad antes de arrancar nada", version, err, punto, errVuelta)
		}
		_ = a.borrarMarcador()
		return "", fmt.Errorf("la instalacion de %s fallo y se ha vuelto atras al punto %s: "+
			"la instalacion sigue en la version %s. Causa: %w", version, punto, manif.Desde, err)
	}
	if err := a.borrarMarcador(); err != nil {
		return "", err
	}
	return punto, nil
}

// Deshacer vuelve a un punto de retorno.
func (a *Actualizador) Deshacer(ctx context.Context, puntoRetorno string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	soltar, err := a.tomarCerrojo()
	if err != nil {
		return err
	}
	defer soltar()
	return a.deshacer(puntoRetorno)
}

// ---------------------------------------------------------------------------
// Lo que anade el CLI por encima del puerto
// ---------------------------------------------------------------------------

// VersionInstalada lee el fichero VERSION. Vacio si no existe: una instalacion
// sin VERSION es una instalacion recien hecha a mano, no un error.
func (a *Actualizador) VersionInstalada() (string, error) {
	b, err := os.ReadFile(filepath.Join(a.raiz, FicheroVersion)) // #nosec G304 -- la raiz la elige el operador
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("no puedo leer %s: %w; sin el no se sabe que version esta "+
			"instalada y `plazum update` no puede decidir si hace falta actualizar",
			filepath.Join(a.raiz, FicheroVersion), err)
	}
	return strings.TrimSpace(string(b)), nil
}

// Puntos lista los puntos de retorno guardados, del mas nuevo al mas viejo.
func (a *Actualizador) Puntos() ([]Manifiesto, error) {
	dir := filepath.Join(a.raiz, DirInterno, nombrePuntos)
	ents, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []Manifiesto
	for _, e := range ents {
		if !e.IsDir() {
			continue
		}
		m, err := a.leerManifiesto(e.Name())
		if err != nil {
			continue // un punto ilegible se ensena como ausente; Deshacer si lo dira
		}
		out = append(out, m)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Fecha > out[j].Fecha })
	return out, nil
}

// Interrumpida devuelve la actualizacion a medias, si la hay.
func (a *Actualizador) Interrumpida() (Manifiesto, bool, error) {
	m, hay, err := a.leerMarcador()
	if err != nil || !hay {
		return Manifiesto{}, false, err
	}
	manif, err := a.leerManifiesto(m.Punto)
	if err != nil {
		return Manifiesto{}, true, fmt.Errorf("hay una actualizacion a medias a %s y su punto "+
			"de retorno %s no se puede leer: %w. Restaura desde tu copia de seguridad",
			m.Version, m.Punto, err)
	}
	return manif, true, nil
}

// Reparar deshace la actualizacion interrumpida, si la hay.
func (a *Actualizador) Reparar(ctx context.Context) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	soltar, err := a.tomarCerrojo()
	if err != nil {
		return "", err
	}
	defer soltar()

	m, hay, err := a.leerMarcador()
	if err != nil {
		return "", err
	}
	if !hay {
		return "", nil
	}
	if err := a.deshacer(m.Punto); err != nil {
		return "", err
	}
	return m.Punto, a.borrarMarcador()
}

// ---------------------------------------------------------------------------
// Cerrojo
// ---------------------------------------------------------------------------

// tomarCerrojo abre el cerrojo en exclusiva. O_EXCL es lo que hace que dos
// procesos no puedan tenerlo a la vez, y lo hace en el sistema de ficheros, que
// es donde tiene que estar: un mutex en memoria no protege de dos `plazum
// update` lanzados desde dos terminales.
func (a *Actualizador) tomarCerrojo() (func(), error) {
	dir := filepath.Join(a.raiz, DirInterno)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, fmt.Errorf("no puedo crear %s: %w; sin el no hay ni cerrojo ni puntos de "+
			"retorno, y sin punto de retorno esto no actualiza", dir, err)
	}
	ruta := filepath.Join(dir, nombreCerrojo)
	// #nosec G304 -- ruta construida aqui mismo: raiz del operador + DirInterno +
	// nombreCerrojo, que es una constante. No entra nada de una peticion.
	f, err := os.OpenFile(ruta, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			otro, _ := os.ReadFile(ruta) // #nosec G304 -- ruta construida aqui mismo
			return nil, fmt.Errorf("%w (%s). Si estas seguro de que no hay ninguna corriendo, "+
				"el proceso anterior murio sin soltarlo: borra %s y vuelve a intentarlo",
				ErrOcupado, strings.TrimSpace(string(otro)), ruta)
		}
		return nil, err
	}
	_, _ = fmt.Fprintf(f, "pid %d desde %s", os.Getpid(), a.ahora.Format(time.RFC3339))
	_ = f.Close()
	return func() { _ = os.Remove(ruta) }, nil
}

// ---------------------------------------------------------------------------
// Punto de retorno
// ---------------------------------------------------------------------------

// crearPunto guarda el estado actual de todo lo que la version va a tocar, mas
// el fichero VERSION, y LO VERIFICA releyendolo.
//
// La relectura no es paranoia: es la unica forma de cazar un disco lleno o una
// cuota agotada ANTES de tocar la instalacion. Un punto de retorno truncado
// pasa desapercibido hasta el dia en que hace falta, que es el peor dia.
func (a *Actualizador) crearPunto(v Version, desde string) (string, Manifiesto, error) {
	id, err := nuevoID(a.ahora)
	if err != nil {
		return "", Manifiesto{}, err
	}
	dir := filepath.Join(a.raiz, DirInterno, nombrePuntos, id)
	if err := os.MkdirAll(filepath.Join(dir, nombreCopia), 0o750); err != nil {
		return "", Manifiesto{}, fmt.Errorf("no puedo crear el punto de retorno %s: %w", dir, err)
	}
	manif := Manifiesto{
		Punto: id, Fecha: a.ahora.Format(time.RFC3339),
		Desde: desde, Hacia: v.Version,
		Ficheros: map[string]string{},
	}

	rutas := rutasOrdenadas(v.Ficheros)
	// El fichero VERSION se respalda siempre, lo declare o no la version: es
	// lo que dice que version esta instalada, y una vuelta atras que no lo
	// restaura deja la instalacion mintiendo sobre si misma.
	if !contiene(rutas, FicheroVersion) {
		rutas = append(rutas, FicheroVersion)
	}

	for _, rel := range rutas {
		origen := filepath.Join(a.raiz, filepath.FromSlash(rel))
		b, err := os.ReadFile(origen) // #nosec G304 -- rel pasa por comprobarRutaRelativa
		if errors.Is(err, os.ErrNotExist) {
			manif.Nuevos = append(manif.Nuevos, rel)
			continue
		}
		if err != nil {
			a.limpiarPunto(id)
			return "", Manifiesto{}, fmt.Errorf("no puedo respaldar %s: %w. Sin poder respaldarlo "+
				"no hay vuelta atras, asi que no se actualiza", origen, err)
		}
		destino := filepath.Join(dir, nombreCopia, filepath.FromSlash(rel))
		if err := escribirEnElPunto(destino, b); err != nil {
			a.limpiarPunto(id)
			return "", Manifiesto{}, fmt.Errorf("no puedo guardar la copia de %s en el punto de "+
				"retorno: %w. Comprueba el espacio libre en disco", rel, err)
		}
		manif.Ficheros[rel] = hex.EncodeToString(sumaDe(b))
	}

	// La verificacion del punto, releyendo de disco.
	if err := a.verificarPunto(id, manif); err != nil {
		a.limpiarPunto(id)
		return "", Manifiesto{}, fmt.Errorf("el punto de retorno recien creado no se verifica: %w. "+
			"No se actualiza: sin vuelta atras comprobada, una actualizacion fallida seria "+
			"irreversible", err)
	}
	b, err := json.MarshalIndent(manif, "", "  ")
	if err != nil {
		a.limpiarPunto(id)
		return "", Manifiesto{}, err
	}
	if err := escribirAtomico(filepath.Join(dir, nombreManif), b); err != nil {
		a.limpiarPunto(id)
		return "", Manifiesto{}, err
	}
	return id, manif, nil
}

// verificarPunto relee las copias y compara sus digests con el manifiesto.
func (a *Actualizador) verificarPunto(id string, m Manifiesto) error {
	dir := filepath.Join(a.raiz, DirInterno, nombrePuntos, id, nombreCopia)
	for _, rel := range rutasOrdenadas(m.Ficheros) {
		ruta := filepath.Join(dir, filepath.FromSlash(rel))
		b, err := os.ReadFile(ruta) // #nosec G304 -- rel pasa por comprobarRutaRelativa
		if err != nil {
			return fmt.Errorf("falta la copia de %s: %w", rel, err)
		}
		if got := hex.EncodeToString(sumaDe(b)); got != m.Ficheros[rel] {
			return fmt.Errorf("la copia de %s tiene digest %s y el manifiesto dice %s",
				rel, got, m.Ficheros[rel])
		}
	}
	return nil
}

// deshacer es la vuelta atras SIN cerrojo, para que Aplicar pueda llamarla
// mientras lo tiene tomado. Fuera del paquete solo se llega por Deshacer.
func (a *Actualizador) deshacer(id string) error {
	if err := comprobarComponente(id); err != nil {
		return fmt.Errorf("%w: %q. Los puntos de retorno los nombra plazum, no se teclean a mano: "+
			"mira los que hay con `plazum update --puntos`", ErrPuntoDesconocido, id)
	}
	m, err := a.leerManifiesto(id)
	if err != nil {
		return err
	}
	// El punto se verifica ANTES de restaurar nada. Restaurar desde una copia
	// que no cuadra con su manifiesto seria escribir bytes desconocidos encima
	// de una instalacion que al menos se sabia rota de una forma concreta.
	if err := a.verificarPunto(id, m); err != nil {
		return fmt.Errorf("%w (%s): %w. NO se ha restaurado nada. Restaura desde tu copia de "+
			"seguridad: volver desde un punto que no cuadra dejaria la instalacion en un estado "+
			"que nadie ha probado, y ademas creerias que has vuelto atras", ErrPuntoCorrupto, id, err)
	}
	dir := filepath.Join(a.raiz, DirInterno, nombrePuntos, id, nombreCopia)
	for _, rel := range rutasOrdenadas(m.Ficheros) {
		b, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(rel))) // #nosec G304 -- verificado arriba
		if err != nil {
			return fmt.Errorf("%w (%s): %w", ErrPuntoCorrupto, id, err)
		}
		if err := escribirAtomico(filepath.Join(a.raiz, filepath.FromSlash(rel)), b); err != nil {
			return fmt.Errorf("volviendo al punto %s no puedo restaurar %s: %w. La instalacion "+
				"esta a medio restaurar: no la arranques y repite `plazum update --deshacer %s` "+
				"cuando el disco lo permita", id, rel, err, id)
		}
	}
	for _, rel := range m.Nuevos {
		if err := comprobarRutaRelativa(rel); err != nil {
			return fmt.Errorf("%w (%s): el manifiesto declara borrar %q, que no es una ruta "+
				"segura dentro de la instalacion", ErrPuntoCorrupto, id, rel)
		}
		if err := os.Remove(filepath.Join(a.raiz, filepath.FromSlash(rel))); err != nil &&
			!errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("volviendo al punto %s no puedo retirar %s, que no existia antes "+
				"de la actualizacion: %w", id, rel, err)
		}
	}
	return nil
}

func (a *Actualizador) leerManifiesto(id string) (Manifiesto, error) {
	if err := comprobarComponente(id); err != nil {
		return Manifiesto{}, fmt.Errorf("%w: %q", ErrPuntoDesconocido, id)
	}
	ruta := filepath.Join(a.raiz, DirInterno, nombrePuntos, id, nombreManif)
	b, err := os.ReadFile(ruta) // #nosec G304 -- id pasa por comprobarComponente
	if errors.Is(err, os.ErrNotExist) {
		return Manifiesto{}, fmt.Errorf("%w: %q. Mira los que hay con `plazum update --puntos`; "+
			"si el que buscas no esta, es que nunca se creo o que alguien borro %s",
			ErrPuntoDesconocido, id, filepath.Join(a.raiz, DirInterno, nombrePuntos))
	}
	if err != nil {
		return Manifiesto{}, err
	}
	var m Manifiesto
	if err := json.Unmarshal(b, &m); err != nil {
		return Manifiesto{}, fmt.Errorf("%w (%s): su manifiesto no es JSON valido: %w",
			ErrPuntoCorrupto, id, err)
	}
	if m.Punto != id {
		return Manifiesto{}, fmt.Errorf("%w (%s): su manifiesto dice llamarse %q. Alguien ha "+
			"movido o renombrado el punto de retorno, y una copia que no sabe de donde viene "+
			"no se restaura", ErrPuntoCorrupto, id, m.Punto)
	}
	for rel := range m.Ficheros {
		if err := comprobarRutaRelativa(rel); err != nil {
			return Manifiesto{}, fmt.Errorf("%w (%s): declara la ruta %q, que no esta dentro de "+
				"la instalacion", ErrPuntoCorrupto, id, rel)
		}
	}
	return m, nil
}

func (a *Actualizador) limpiarPunto(id string) {
	if comprobarComponente(id) != nil {
		return
	}
	_ = os.RemoveAll(filepath.Join(a.raiz, DirInterno, nombrePuntos, id))
}

// ---------------------------------------------------------------------------
// Marcador de actualizacion en curso
// ---------------------------------------------------------------------------

func (a *Actualizador) rutaMarcador() string {
	return filepath.Join(a.raiz, DirInterno, nombreMarcador)
}

func (a *Actualizador) escribirMarcador(m marcador) error {
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return escribirAtomico(a.rutaMarcador(), b)
}

func (a *Actualizador) leerMarcador() (marcador, bool, error) {
	b, err := os.ReadFile(a.rutaMarcador()) // #nosec G304 -- ruta construida aqui
	if errors.Is(err, os.ErrNotExist) {
		return marcador{}, false, nil
	}
	if err != nil {
		return marcador{}, false, err
	}
	var m marcador
	if err := json.Unmarshal(b, &m); err != nil {
		return marcador{}, true, fmt.Errorf("%w: el marcador %s no es JSON valido: %w. "+
			"Comprueba a mano el estado de la instalacion antes de tocar nada",
			ErrAMedias, a.rutaMarcador(), err)
	}
	return m, true, nil
}

func (a *Actualizador) borrarMarcador() error {
	if err := os.Remove(a.rutaMarcador()); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// ---------------------------------------------------------------------------
// Instalacion
// ---------------------------------------------------------------------------

// escribirEnLaInstalacion es la escritura que usa la INSTALACION, y solo ella.
//
// Va como variable por una razon concreta y no por gusto: la vuelta atras
// automatica cuando la instalacion falla a mitad es la propiedad mas importante
// de este paquete y a la vez la mas dificil de provocar de forma portable, y un
// fallo que no se puede provocar es una rama que nadie ha ejecutado nunca. El
// test la sustituye para hacer fallar el segundo fichero. El punto de retorno y
// la restauracion NO pasan por aqui a proposito: con la escritura de la
// instalacion averiada, la vuelta atras tiene que seguir funcionando.
var escribirEnLaInstalacion = escribirAtomico

// escribirEnElPunto es la escritura de la COPIA del punto de retorno.
//
// Existe como variable por el mismo motivo que la anterior: un disco que se
// llena justo mientras se hace la copia es la forma real de acabar con un punto
// de retorno truncado, y un punto de retorno truncado pasa desapercibido hasta
// el dia en que hace falta, que es el peor dia. Provocarlo de forma portable no
// se puede; sustituir esta funcion, si.
var escribirEnElPunto = escribirAtomico

func (a *Actualizador) instalar(contenido map[string][]byte, version string) error {
	for _, rel := range rutasOrdenadas(contenido) {
		if err := escribirEnLaInstalacion(filepath.Join(a.raiz, filepath.FromSlash(rel)), contenido[rel]); err != nil {
			return err
		}
	}
	if _, ya := contenido[FicheroVersion]; ya {
		return nil
	}
	return escribirEnLaInstalacion(filepath.Join(a.raiz, FicheroVersion), []byte(version+"\n"))
}

// escribirAtomico escribe a un temporal, lo sincroniza y lo renombra encima.
//
// El sync antes del rename es lo que separa "escrito" de "escrito de verdad":
// sin el, un corte de corriente entre las dos operaciones deja el nombre bueno
// apuntando a un fichero vacio, que es la forma mas rapida de convertir una
// actualizacion interrumpida en una instalacion irrecuperable.
func escribirAtomico(destino string, datos []byte) error {
	if err := os.MkdirAll(filepath.Dir(destino), 0o750); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(destino), ".plazum-tmp-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	if _, err := f.Write(datos); err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	// Los permisos: CreateTemp deja 0600, y el binario tiene que poder
	// ejecutarse. 0700 para el dueno y nada para nadie mas.
	if err := os.Chmod(tmp, 0o700); err != nil { // #nosec G302 -- el binario del producto tiene que ser ejecutable por su dueno
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, destino); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

// ---------------------------------------------------------------------------
// Nombres, rutas y digests
// ---------------------------------------------------------------------------

func sumaDe(b []byte) []byte {
	s := sha256.Sum256(b)
	return s[:]
}

// LongitudMaximaDeNombre acota los nombres que llegan del canal. Un canal es un
// tercero: sus nombres son entrada no fiable.
const LongitudMaximaDeNombre = 64

// ComprobarNombreDeVersion acepta lo que puede ser un componente de ruta seguro
// y a la vez un nombre de version legible.
func ComprobarNombreDeVersion(v string) error {
	if strings.TrimSpace(v) == "" {
		return fmt.Errorf("%w: la version esta vacia. Di cual quieres instalar, o mira las que "+
			"hay con `plazum update --disponible`", ErrNombreInvalido)
	}
	if err := comprobarComponente(v); err != nil {
		return fmt.Errorf("%w: %q no vale como nombre de version. %v", ErrNombreInvalido, v, err)
	}
	return nil
}

// comprobarComponente acepta un unico componente de ruta, sin separadores ni
// referencias al directorio padre. Es lo que impide que una version llamada
// "../../etc" salga de la instalacion.
func comprobarComponente(s string) error {
	if s == "" || s == "." || s == ".." {
		return errors.New("no es un nombre")
	}
	if len(s) > LongitudMaximaDeNombre {
		return fmt.Errorf("son %d caracteres y el tope son %d", len(s), LongitudMaximaDeNombre)
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
		case c == '.' || c == '-' || c == '_' || c == '+':
		default:
			return fmt.Errorf("lleva el caracter %q, y solo se admiten letras, digitos, "+
				"punto, guion, guion bajo y mas", string(c))
		}
	}
	return nil
}

// comprobarRutaRelativa acepta una ruta con barras, relativa y dentro de la
// instalacion.
//
// Es la frontera con el canal, que es un tercero: una version que declarara
// "../../.ssh/authorized_keys" escribiria fuera de la instalacion, y una que
// declarara ".plazum/puntos/..." podria falsificar sus propios puntos de retorno.
// Las dos se rechazan aqui y no mas adelante, porque mas adelante ya se habria
// leido el contenido.
func comprobarRutaRelativa(rel string) error {
	if strings.TrimSpace(rel) == "" {
		return fmt.Errorf("%w: ruta vacia", ErrRutaInsegura)
	}
	if strings.ContainsRune(rel, '\\') {
		return fmt.Errorf("%w: %q lleva barra invertida; las rutas del catalogo van con barra "+
			"normal para que signifiquen lo mismo en todos los sistemas", ErrRutaInsegura, rel)
	}
	if path.IsAbs(rel) || filepath.IsAbs(rel) || strings.HasPrefix(rel, "/") {
		return fmt.Errorf("%w: %q es absoluta; una version solo puede tocar ficheros dentro de "+
			"la instalacion", ErrRutaInsegura, rel)
	}
	if len(rel) > 1 && rel[1] == ':' {
		return fmt.Errorf("%w: %q lleva unidad de disco", ErrRutaInsegura, rel)
	}
	limpia := path.Clean(rel)
	if limpia != rel {
		return fmt.Errorf("%w: %q no esta en forma canonica (seria %q). Una ruta que cambia al "+
			"normalizarla puede significar dos cosas distintas segun quien la mire",
			ErrRutaInsegura, rel, limpia)
	}
	for _, seg := range strings.Split(limpia, "/") {
		if seg == "" || seg == "." || seg == ".." {
			return fmt.Errorf("%w: %q sale de la instalacion o tiene un segmento vacio",
				ErrRutaInsegura, rel)
		}
		if err := comprobarComponente(seg); err != nil {
			return fmt.Errorf("%w: %q: el segmento %q %v", ErrRutaInsegura, rel, seg, err)
		}
	}
	if primero := strings.Split(limpia, "/")[0]; primero == DirInterno {
		return fmt.Errorf("%w: %q entra en %s, que es donde viven el cerrojo y los puntos de "+
			"retorno. Una version que pudiera escribir ahi podria falsificar su propia vuelta "+
			"atras", ErrRutaInsegura, rel, DirInterno)
	}
	return nil
}

// rutasOrdenadas devuelve las claves ordenadas. El orden fijo importa: hace
// reproducible el orden de escritura, y con el, el punto en que se queda una
// instalacion interrumpida.
func rutasOrdenadas[T any](m map[string]T) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func contiene(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}

// nuevoID nombra un punto de retorno. Lleva la fecha delante para que un
// listado salga ordenado solo, y ocho bytes al azar detras para que dos
// actualizaciones en el mismo segundo no compartan punto.
func nuevoID(ahora time.Time) (string, error) {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("no puedo nombrar el punto de retorno: %w", err)
	}
	return ahora.UTC().Format("20060102T150405Z") + "-" + hex.EncodeToString(b[:]), nil
}
