package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/marcosmatalab/plazum/nucleo/accesos"
	"github.com/marcosmatalab/plazum/nucleo/censo"
	"github.com/marcosmatalab/plazum/nucleo/ledger"
)

// LA CAMPANA QUE VIVE EN EL DIRECTORIO DE DATOS, que es la que permite subir el
// censo por el navegador.
//
// # Que cambia respecto de campanaEnFichero, y por que hacen falta las dos
//
// campanaEnFichero es para quien YA TIENE los ficheros y le dice a `serve`
// cuales son: los gestiona otro sistema, o los dejo ahi `plazum accesos ver`. Su
// contrato es "lee estos dos ficheros exactos", y esa pantalla no escribe en
// ellos, porque escribir en ficheros que gestiona otro es como se pierde un
// registro que no es tuyo.
//
// Esta otra es para quien acaba de descargarse plazum y no tiene nada: el
// directorio es de la instalacion, plazum lo crea, y por eso puede escribir
// dentro. Es la que apaga las dos ordenes de terminal que la pantalla pedia.
//
// # DONDE ESTA CADA COSA, y por que el CSV se llama como se llama
//
//	<datos>/accesos/ledger.json   los hechos: la apertura y luego las decisiones
//	<datos>/accesos/<sello>.csv   el fichero exacto sobre el que se abrio
//
// El nombre del CSV es el SELLO de la instantanea, que es un sha256 en
// hexadecimal, y eso resuelve tres cosas de una vez:
//
//  1. NO HAY TRAVESIA DE RUTAS POSIBLE. El nombre no lleva ni un byte que venga
//     del usuario. Si el nombre saliera del identificador de la campana, y ese
//     del sistema que teclea quien sube, un sistema llamado `../../etc/passwd`
//     escribiria donde no debe. Aun asi se comprueba que son 64 caracteres
//     hexadecimales antes de componer la ruta: el sello sale del ledger, y un
//     ledger es un fichero que alguien puede haber editado.
//  2. DOS CAMPANAS NO SE PISAN. Subir un censo nuevo no deja ilegible el
//     anterior, porque son dos ficheros. Con un nombre fijo, abrir la campana de
//     octubre habria dejado la de septiembre sin su fichero, y las decisiones ya
//     tomadas sin nada sobre lo que reconstruirse.
//  3. LA RUTA SE DERIVA DEL LEDGER, no de una segunda anotacion. El ledger ya
//     guarda el sello dentro de la apertura, o sea DENTRO DE LO ENCADENADO, asi
//     que el fichero y la campana casan por un campo que nadie puede mover sin
//     romper la cadena (invariante 7).
//
// # CUAL DE LAS CAMPANAS SE ENSENA, dicho porque es una decision
//
// LA ULTIMA ABIERTA, y "ultima" es por POSICION EN EL LEDGER, no por la fecha
// que traiga la entrada. La posicion es lo que esta encadenado: cada entrada
// lleva el hash de la anterior y su Seq, asi que reordenar el fichero rompe la
// verificacion. Un instante, en cambio, es un campo mas y lo escribe quien
// anota. Emparejar por el campo firmado y no por el declarado es la regla de
// siempre.
//
// El limite, dicho: esta pantalla ensena UNA campana, la viva. Las anteriores
// siguen enteras en el ledger y con su fichero al lado, y quien las quiera ver
// las reconstruye con las ordenes de `plazum accesos`. La pantalla que ensene el
// historico es otra pantalla y todavia no existe.
type campanaEnDirectorio struct {
	dir   string
	ahora func() time.Time
}

// NombreDelLedger es como se llama el registro dentro del directorio de datos.
const nombreDelLedgerDeAccesos = "ledger.json"

// fuenteDeLaSubida y retencionPorDefecto son los dos datos que la orden de
// terminal pide por bandera y que aqui no se pueden preguntar.
//
// NO SE INVENTAN Y SE ESCRIBEN EN EL LEDGER TAL CUAL, asi que quien lea la
// apertura ve de donde salio la lista y cuanto se dijo que se iba a guardar. La
// retencion es la misma cadena que trae por defecto `plazum accesos ver`: dos
// valores por defecto distintos para el mismo campo serian dos promesas
// distintas al titular de los datos segun por donde entrara el fichero.
const (
	fuenteDeLaSubida    = "importacion manual desde el navegador"
	retencionPorDefecto = "12 meses desde el cierre de la campana"
)

// reSistema acota lo que se admite como nombre de sistema.
//
// Va aqui, en la frontera de entrada, y no dentro del nucleo: el nucleo recibe
// un dato ya juzgado. Se acota porque este valor acaba DENTRO del sujeto de una
// entrada de un registro append-only (`accesos/accesos-<sistema>-<fecha>`), y de
// ahi ya no sale nunca: un salto de linea o una barra metidos ahi se quedan para
// siempre en un fichero que no se reescribe.
var reSistema = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9 ._-]{0,63}$`)

var reSello = regexp.MustCompile(`^[0-9a-f]{64}$`)

func (c campanaEnDirectorio) rutaDelLedger() string {
	return filepath.Join(c.dir, nombreDelLedgerDeAccesos)
}

// rutaDelCenso compone la ruta del CSV a partir del sello.
//
// SE VALIDA AUNQUE VENGA DE CASA. El sello sale del ledger, y un ledger es un
// fichero de disco que alguien pudo editar: un sello con `../` dentro haria que
// esta funcion leyera lo que ese alguien quiera. Que hoy solo lo escribamos
// nosotros no es una propiedad del codigo, es una costumbre.
func (c campanaEnDirectorio) rutaDelCenso(sello string) (string, error) {
	if !reSello.MatchString(sello) {
		return "", fmt.Errorf("el ledger de %s dice que la campana se abrio sobre el sello %q, "+
			"y un sello es un sha256 en hexadecimal.\n"+
			"  No se compone una ruta con el: un nombre de fichero que sale de un dato que no "+
			"se entiende es por donde se lee lo que no se debe.\n"+
			"  Arreglo: ese registro esta corrupto o es de otra version. Compruebalo con "+
			"`plazum verify` antes de tocarlo", c.rutaDelLedger(), sello)
	}
	return filepath.Join(c.dir, sello+".csv"), nil
}

// Abierta reconstruye la campana viva. Devuelve (nil, nil) cuando todavia no hay
// ninguna, que NO es un error: es el estado vacio, y la pantalla lo sabe pintar
// con su formulario de subida dentro.
func (c campanaEnDirectorio) Abierta() (*accesos.Campana, error) {
	l, err := leerLedger(c.rutaDelLedger())
	if err != nil {
		return nil, err
	}
	id, ap, subioLo, cuando, ok := ultimaApertura(l)
	if !ok {
		// NI UNA CAMPANA TODAVIA. Aqui es donde se decide que una instalacion
		// recien descargada ve el formulario y no un error: leer esto como
		// fallo pintaria un aviso rojo la primera vez que alguien abre la
		// pantalla, y un producto que empieza acusando no lo abre nadie dos
		// veces.
		return nil, nil
	}
	ruta, err := c.rutaDelCenso(ap.Sello)
	if err != nil {
		return nil, err
	}
	datos, err := os.ReadFile(ruta) // #nosec G304 -- el nombre es el sello, 64 hexadecimales comprobados arriba
	if err != nil {
		return nil, fmt.Errorf("la campana %q consta abierta en %s y su fichero no esta en %s: "+
			"%v.\n"+
			"  Las filas no se guardan en el registro a proposito (son dato personal), asi que "+
			"sin el fichero no hay a quien revisar.\n"+
			"  Arreglo: devolver ese fichero a su sitio, o subir el censo otra vez para abrir "+
			"una campana nueva", id, c.rutaDelLedger(), ruta, err)
	}
	ins, err := censo.Tomar(datos, censo.Opciones{
		Sistema: ap.Sistema, Fuente: ap.Fuente, Quien: subioLo, Tomada: cuando,
		Retencion: ap.Retencion, Columnas: censo.ColumnasHabituales(),
	})
	if err != nil {
		return nil, err
	}
	return accesos.Reconstruir(id, ins, l, nil)
}

func (c campanaEnDirectorio) Anotar(e ledger.Entrada) error {
	l, err := leerLedger(c.rutaDelLedger())
	if err != nil {
		return err
	}
	return anotarEnLedger(c.rutaDelLedger(), l, e)
}

// Abrir sella el fichero que ha subido el navegador y registra su apertura.
//
// EL ORDEN DE LAS DOS ESCRITURAS NO ES INDIFERENTE, y por eso esta dicho:
// primero el CSV y despues el ledger. Si falla la segunda queda un CSV huerfano,
// que no le hace dano a nadie y se sobrescribe solo la proxima vez. Al reves
// quedaria una campana anotada en un registro APPEND-ONLY sin fichero sobre el
// que reconstruirse, o sea una fila que no se puede borrar apuntando a nada.
func (c campanaEnDirectorio) Abrir(datos []byte, sistema, quien string) error {
	sistema = strings.TrimSpace(sistema)
	if !reSistema.MatchString(sistema) {
		return fmt.Errorf("el sistema %q no vale como nombre.\n"+
			"  Se admiten letras, numeros, espacios, puntos, guiones y guiones bajos, hasta 64.\n"+
			"  No es capricho: este nombre entra en el sujeto de una entrada de un registro "+
			"que no se reescribe nunca.\n"+
			"  Arreglo: algo como erp, directorio o nomina-2026", sistema)
	}
	instante := c.ahora().UTC()
	// SE SELLA ANTES DE ESCRIBIR NADA. Si el fichero no se entiende como censo,
	// el error de censo.Tomar sale entero (dice la codificacion, el separador y
	// que columna falta) y en el disco no ha quedado nada a medias.
	ins, err := censo.Tomar(datos, censo.Opciones{
		Sistema:   sistema,
		Fuente:    fuenteDeLaSubida,
		Quien:     quien,
		Tomada:    instante,
		Retencion: retencionPorDefecto,
		Columnas:  censo.ColumnasHabituales(),
	})
	if err != nil {
		return err
	}
	if err := os.MkdirAll(c.dir, 0o700); err != nil {
		return fmt.Errorf("no se puede crear %s, que es donde vive la revision de accesos: %w",
			c.dir, err)
	}
	ruta, err := c.rutaDelCenso(ins.Sello())
	if err != nil {
		return err
	}
	// 0600 y no 0644: esto es un censo de personas con sus permisos. El fsync
	// es el mismo que usa la orden de terminal, y por lo mismo: un fichero que
	// se pierde en el cache de pagina cuando se va la luz deja una campana
	// anotada sin nada debajo.
	if err := escribirConFsync(ruta, datos); err != nil {
		return fmt.Errorf("no se puede guardar el censo en %s: %w", ruta, err)
	}
	// EL IDENTIFICADOR LLEVA EL SELLO DENTRO, y esa es la diferencia con el
	// que compone la orden de terminal.
	//
	// Alli el id es `accesos-<sistema>-<fecha>` y lo teclea una persona que
	// sabe lo que esta haciendo. Aqui lo compone el producto, y subir dos
	// veces el mismo dia el mismo sistema NO es un error del usuario: es lo
	// que pasa cuando alguien arregla la exportacion de su IdP y la vuelve a
	// mandar, que es la manana de un martes cualquiera.
	//
	// Con el id repetido, la segunda subida seria INVISIBLE: accesos.Reconstruir
	// se queda con la PRIMERA apertura que encuentra para ese sujeto, asi que la
	// pantalla seguiria ensenando el fichero viejo sin decir nada. Un fallo que
	// no da error y ademas ensena datos caducados es exactamente el que nadie
	// encuentra. Los ocho primeros del sello lo hacen imposible sin depender de
	// ningun contador ni de ningun reloj.
	id := fmt.Sprintf("accesos-%s-%s-%s", identificadorDeSistema(sistema),
		instante.Format("2006-01-02"), ins.Sello()[:8])
	return anotarApertura(c.rutaDelLedger(), ins, id)
}

// identificadorDeSistema convierte el nombre en algo que se puede leer dentro de
// un identificador de campana. No es seguridad (de eso se encarga reSistema
// antes), es legibilidad: `accesos/accesos-mi erp-2026-09-05` con un espacio
// dentro se lee mal en cualquier salida de terminal.
func identificadorDeSistema(v string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(v) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	return strings.Trim(b.String(), "-")
}

// ultimaApertura da la campana viva del ledger: la ULTIMA anotada.
//
// Recorre de atras hacia delante y devuelve la primera apertura que encuentra.
// La posicion es lo que esta encadenado por hash, asi que "la ultima" es una
// propiedad de lo firmado y no del instante que traiga la entrada, que lo
// escribe quien anota (invariante 7).
//
// Una carga de apertura que no se entiende NO se salta: se para. Saltarla
// ensenaria la campana anterior como si fuera la de ahora, que es la peor de
// las tres respuestas posibles, porque parece que funciona.
func ultimaApertura(l ledger.Ledger) (string, accesos.CargaDeApertura, string, time.Time, bool) {
	for i := len(l.Entradas) - 1; i >= 0; i-- {
		e := l.Entradas[i]
		if e.Tipo != accesos.TipoApertura || !strings.HasPrefix(e.Sujeto, "accesos/") {
			continue
		}
		id := strings.TrimPrefix(e.Sujeto, "accesos/")
		ap, subioLo, cuando, ok := datosDeApertura(l, id)
		if !ok {
			return "", accesos.CargaDeApertura{}, "", time.Time{}, false
		}
		return id, ap, subioLo, cuando, true
	}
	return "", accesos.CargaDeApertura{}, "", time.Time{}, false
}
