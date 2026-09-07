// Package manual es el PRIMER recolector de plazum, y no necesita ni una
// credencial (A3 de D-22).
//
// # Por que este va primero y no un conector de la nube
//
// El godoc de `puertos.Ingesta` lo dice desde que se escribio: un aporte humano
// es «un conector de pleno derecho, no un parche». Esto lo cobra.
//
// La cadena que hay que cerrar tiene siete eslabones —observacion, prueba,
// `estado.Calcular`, pantalla, expediente, export y `verify` offline— y lo unico
// que anade OAuth contra un proveedor es una FUENTE distinta en el primero. Todo
// lo que puede romperse esta en los otros seis. Construir el conector de la nube
// primero es como se descubre eso en la semana seis en vez de en la primera, y
// con credenciales de un cliente de por medio.
//
// # LO QUE ENTREGA SON HECHOS, JAMAS VEREDICTOS (invariante 13)
//
// Una fila de este fichero dice «el recurso X satisfizo el predicado de la
// prueba Y el dia D». No dice «cumples». El juicio lo compone `estado.Calcular`
// con la prueba del paquete delante, y lo confirma una persona.
//
// El campo `satisfecho` que se lee aqui es el resultado de un predicado mecanico
// DECLARADO EN LA PRUEBA del paquete, que el linter del corpus exige desde A1.
// Sin ese predicado seria un veredicto disfrazado de dato; con el, quien lea el
// expediente puede ir a mirar QUE se evaluo.
//
// # «FIRMADO» AQUI NO ES UNA FIRMA CRIPTOGRAFICA, Y SE DICE
//
// `ETAPAS.md` dice «desde fichero firmado», y hay que ser exacto: lo que hace
// este paquete es lo mismo que `nucleo/censo`, que tampoco firma con Ed25519.
// Sella: calcula el sha256 del contenido y un sello sobre los campos que
// deciden COMO se leyo, cada uno con su longitud delante para que dos parejas
// distintas no den el mismo sello. La unica firma criptografica del proyecto
// esta en el checkpoint del ledger.
//
// Decirlo importa: si el godoc o la pantalla dijeran «firmado» a secas, seria
// una afirmacion acompanada, y de las caras, porque quien la lea creera que hay
// una clave detras.
//
// # LA LEY DE CONSERVACION, COPIADA DE `nucleo/censo` PORQUE ES LO QUE VALE
//
// Un recolector que lee un fichero y no cuenta lo que descarta hereda el fallo
// exacto que aquel paquete documenta: el parser se traga filas en silencio y
// devuelve un resultado plausible. Aqui, cada linea del fichero cae en
// EXACTAMENTE UN cubo (leida, en blanco, comentario, ilegible), la suma tiene
// que dar el total, y si no cuadra el recolector PARA en vez de entregar.
//
// Y el contador del total es CIEGO A LA ESTRUCTURA: cuenta saltos de linea
// antes de que nadie parsee nada. Un contador que respetara el formato se
// tragaria las mismas lineas que el parser, y entonces cuadraria siempre.
//
// # LAS TRES FORMAS DE LA NADA (invariante 8)
//
//	fichero AUSENTE           nadie ha recolectado todavia. Cero observaciones,
//	                          sin error. Es el caso de una instalacion recien
//	                          descargada y es un DATO.
//	fichero PRESENTE Y VACIO  cero observaciones, sin error. Tambien es un dato:
//	                          alguien puso un fichero y no dijo nada.
//	PRESENTE Y NO
//	INTERPRETABLE             error, SIEMPRE. Una version desconocida, una linea
//	                          que no casa, un descuadre del contador. Leerlo
//	                          como «todavia nadie ha recolectado» le diria a
//	                          alguien que su instalacion esta al dia cuando lo
//	                          que pasa es que no hemos podido leer su fichero.
//
// # EL CURSOR, DECIDIDO Y ESCRITO
//
// La firma del puerto trae `cursor` y `siguiente`, que son para una fuente
// paginada. Este recolector lee un fichero entero de una vez, asi que:
//
//	cursor a la ENTRADA     tiene que venir vacio. Un cursor que no se entiende
//	                        es la tercera forma de la nada y sale error, jamas
//	                        «empieza por el principio»: quien lo mando creia
//	                        estar continuando algo.
//	siguiente a la SALIDA   siempre vacio, que significa «no hay mas». Devolver
//	                        cualquier otra cosa haria que un llamante correcto
//	                        entrara en un bucle infinito.
package manual

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/marcosmatalab/plazum/nucleo/estado"
	"github.com/marcosmatalab/plazum/puertos"
)

// Version es el formato del fichero. Una version que no se conoce es error, no
// «la de por defecto»: el fichero lo escribe una persona y una version futura
// puede significar campos que este codigo no sabe leer.
const Version = 1

// MaximoFichero es lo mas grande que se acepta leer. Un recolector manual lee
// lo que exporta una herramienta, no un volcado entero.
const MaximoFichero = 32 << 20 // 32 MiB

// Nombre es como se identifica este recolector en cada observacion. Va en
// `estado.Observacion.Recolector` y es lo que la pantalla ensena en la columna
// de procedencia.
const Nombre = "manual"

// Los errores. Centinela cada uno, para que quien llame distinga «no hay
// fichero» de «hay fichero y no se entiende» sin leer una cadena.
var (
	ErrSinRuta         = errors.New("recoleccion/manual: sin ruta de la que leer")
	ErrDemasiadoGrande = errors.New("recoleccion/manual: el fichero pasa del maximo")
	ErrVersion         = errors.New("recoleccion/manual: version del fichero desconocida")
	ErrLineaIlegible   = errors.New("recoleccion/manual: linea que no se entiende")
	ErrDescuadre       = errors.New("recoleccion/manual: las lineas leidas no suman el total")
	ErrCursor          = errors.New("recoleccion/manual: cursor que este recolector no entiende")
	ErrFuente          = errors.New("recoleccion/manual: fuente que este recolector no sirve")
	ErrSinPrueba       = errors.New("recoleccion/manual: observacion sin prueba")
	ErrSinRecurso      = errors.New("recoleccion/manual: observacion sin recurso")
	ErrSinInstante     = errors.New("recoleccion/manual: observacion sin instante de recoleccion")
	ErrVeredicto       = errors.New("recoleccion/manual: el fichero trae un campo de veredicto")
)

// Fuente es el nombre que hay que pasarle a Recolectar. Se exige que coincida
// para que un llamante que se equivoque de recolector se entere, en vez de
// recibir cero observaciones y creer que no hay nada.
const Fuente = "manual"

// filaEnDisco es una observacion tal como la escribe una persona.
//
// LOS NOMBRES DE LOS CAMPOS SON LOS DEL DOMINIO Y NO LOS DEL MOTOR: quien
// escribe este fichero es una persona con un export delante, no quien conoce
// `estado.Observacion`.
type filaEnDisco struct {
	Prueba     string `json:"prueba"`
	Recurso    string `json:"recurso"`
	Satisfecho *bool  `json:"satisfecho"`
	Cuando     string `json:"cuando"`
	Caduca     string `json:"caduca,omitempty"`
	Error      string `json:"error,omitempty"`
}

// cabecera es la primera linea del fichero.
type cabecera struct {
	Version int    `json:"version"`
	Sistema string `json:"sistema,omitempty"`
	Quien   string `json:"quien,omitempty"`
}

// Recuento es la ley de conservacion: cada linea cae en exactamente un cubo.
type Recuento struct {
	Total      int
	Cabecera   int
	Leidas     int
	EnBlanco   int
	Comentario int
}

// ComprobarRecuento es la ley de conservacion, y HOY NINGUN FICHERO PUEDE
// LLEGAR AQUI CON UN DESCUADRE. Se dice, porque una guarda que ninguna entrada
// alcanza es una guarda que no existe, y este proyecto lleva catorce hallazgos
// aprendiendo a no confundir las dos cosas.
//
// LO DESTAPO UNA MUTACION QUE SOBREVIVIO (M37, 07-09-2026): apagar esta
// comprobacion dejaba la suite entera en verde. El motivo es que el contador
// del total y el parser recorren exactamente las mismas lineas, asi que no hay
// fichero que los separe.
//
// Y AUN ASI SE QUEDA, con su prueba sintetica. Lo que vigila no es un fichero:
// es el PARSER DEL FUTURO. El dia que alguien meta un `continue` de mas en el
// bucle de `leer` —que es como se traga filas un parser, y es exactamente lo que
// documenta `nucleo/censo`— los dos contadores dejaran de coincidir y esto
// parara. Sin ella, ese dia se entregarian observaciones incompletas presentadas
// como completas, que es peor que no entregar ninguna.
//
// LO VIGILA: TestLaLeyDeConservacionParaSiLosCubosNoSumanElTotal, con dato
// sintetico y en las dos direcciones.
func ComprobarRecuento(rec Recuento) error {
	if rec.Cuadra() {
		return nil
	}
	return fmt.Errorf("%w: %s. Arreglo: no se entrega nada, porque un recuento que no "+
		"cuadra significa que el parser se ha tragado lineas en silencio, y unas "+
		"observaciones incompletas presentadas como completas son peores que ninguna",
		ErrDescuadre, rec)
}

// Cuadra dice si los cubos suman el total.
func (r Recuento) Cuadra() bool {
	return r.Cabecera+r.Leidas+r.EnBlanco+r.Comentario == r.Total
}

func (r Recuento) String() string {
	return fmt.Sprintf("total=%d cabecera=%d leidas=%d en_blanco=%d comentario=%d",
		r.Total, r.Cabecera, r.Leidas, r.EnBlanco, r.Comentario)
}

// Opciones construye el recolector. El valor cero esta prohibido y el error
// dice que falta.
type Opciones struct {
	// Ruta es el fichero del que se lee. Obligatoria.
	Ruta string
}

// Recolector lee observaciones de un fichero.
type Recolector struct {
	ruta string
	// sello es el del ultimo fichero leido, para que quien quiera anotarlo en
	// el ledger no tenga que recalcularlo.
	sello string
}

// Comprobacion en compilacion de que esto implementa el puerto. Es el gesto que
// hace que el compilador vigile la implementacion, y hasta hoy no lo hacia
// ningun tipo para `puertos.Recoleccion`.
var _ puertos.Recoleccion = (*Recolector)(nil)

// Abrir construye el recolector. NO lee el fichero: leerlo es Recolectar, y
// separarlos es lo que permite arrancar el servidor antes de que exista.
func Abrir(o Opciones) (*Recolector, error) {
	if strings.TrimSpace(o.Ruta) == "" {
		return nil, fmt.Errorf("%w. Arreglo: pasa Opciones.Ruta con el fichero de "+
			"observaciones. Una ruta vacia no es «el sitio por defecto»: es que no se "+
			"ha dicho de donde leer", ErrSinRuta)
	}
	return &Recolector{ruta: o.Ruta}, nil
}

// RutaPorDefecto es donde vive el fichero dentro del directorio de datos.
func RutaPorDefecto(datos string) string {
	return filepath.Join(datos, "observaciones.jsonl")
}

// Sello es el del ultimo fichero leido, o vacio si no se ha leido ninguno.
func (r *Recolector) Sello() string { return r.sello }

// Recolectar lee el fichero entero y devuelve sus observaciones.
func (r *Recolector) Recolectar(fuente, cursor string) ([]estado.Observacion, string, error) {
	if fuente != "" && fuente != Fuente {
		return nil, "", fmt.Errorf("%w: se ha pedido %q y este recolector sirve %q. "+
			"Arreglo: llama al recolector que sirva esa fuente. Devolver cero "+
			"observaciones haria creer que esa fuente no tiene nada",
			ErrFuente, fuente, Fuente)
	}
	if cursor != "" {
		return nil, "", fmt.Errorf("%w: %q. Este recolector lee el fichero entero de una "+
			"vez y siempre devuelve el cursor vacio, asi que un cursor de entrada "+
			"significa que quien llama cree estar continuando algo que nunca empezo. "+
			"Arreglo: llama sin cursor", ErrCursor, cursor)
	}

	b, err := os.ReadFile(r.ruta) // #nosec G304 -- ruta de configuracion del operador, no de la peticion
	if errors.Is(err, os.ErrNotExist) {
		// FICHERO AUSENTE: nadie ha recolectado todavia. Es un DATO y no un
		// error, y es el caso de una instalacion recien descargada.
		r.sello = ""
		return nil, "", nil
	}
	if err != nil {
		return nil, "", fmt.Errorf("recoleccion/manual: leyendo %s: %w",
			filepath.Base(r.ruta), err)
	}
	if len(b) > MaximoFichero {
		return nil, "", fmt.Errorf("%w: %d bytes y el maximo son %d. Arreglo: parte el "+
			"fichero, o exporta menos recursos por vez", ErrDemasiadoGrande, len(b), MaximoFichero)
	}

	obs, rec, sello, err := leer(b)
	if err != nil {
		return nil, "", err
	}
	if err := ComprobarRecuento(rec); err != nil {
		return nil, "", err
	}
	r.sello = sello
	return obs, "", nil
}

// Contar devuelve la ley de conservacion del ultimo fichero, para que quien
// monte una pantalla pueda decir cuantas lineas se leyeron y cuantas no.
func (r *Recolector) Contar() (Recuento, error) {
	b, err := os.ReadFile(r.ruta) // #nosec G304 -- ruta de configuracion del operador, no de la peticion
	if errors.Is(err, os.ErrNotExist) {
		return Recuento{}, nil
	}
	if err != nil {
		return Recuento{}, err
	}
	_, rec, _, err := leer(b)
	return rec, err
}

// leer parsea el contenido con su ley de conservacion.
func leer(b []byte) ([]estado.Observacion, Recuento, string, error) {
	var rec Recuento
	// EL TOTAL SE CUENTA ANTES DE PARSEAR NADA Y SIN MIRAR LA ESTRUCTURA. Un
	// contador que entendiera el formato se tragaria exactamente las mismas
	// lineas que el parser, y entonces cuadraria siempre y no vigilaria nada.
	rec.Total = contarLineas(b)

	var obs []estado.Observacion
	sc := bufio.NewScanner(bytes.NewReader(b))
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	n := 0
	vistaCabecera := false
	for sc.Scan() {
		n++
		linea := strings.TrimSpace(sc.Text())
		switch {
		case linea == "":
			rec.EnBlanco++
			continue
		case strings.HasPrefix(linea, "#"):
			rec.Comentario++
			continue
		}
		if !vistaCabecera {
			var c cabecera
			if err := json.Unmarshal([]byte(linea), &c); err != nil {
				return nil, rec, "", fmt.Errorf("%w: linea %d, la cabecera: %v. Arreglo: "+
					"la primera linea con contenido es la cabecera, y es un objeto JSON "+
					"con al menos {\"version\": %d}", ErrLineaIlegible, n, err, Version)
			}
			if c.Version != Version {
				return nil, rec, "", fmt.Errorf("%w: el fichero dice version %d y este "+
					"plazum lee la %d. Arreglo: actualiza plazum, o exporta el fichero "+
					"en la version que este lee. Una version desconocida NO se lee «lo "+
					"mejor que se pueda»: puede traer campos que cambien el significado "+
					"de los que si se entienden", ErrVersion, c.Version, Version)
			}
			vistaCabecera = true
			rec.Cabecera++
			continue
		}
		o, err := aObservacion(linea, n)
		if err != nil {
			return nil, rec, "", err
		}
		obs = append(obs, o)
		rec.Leidas++
	}
	if err := sc.Err(); err != nil {
		return nil, rec, "", fmt.Errorf("%w: %v", ErrLineaIlegible, err)
	}
	return obs, rec, sello(b, rec), nil
}

// contarLineas cuenta saltos de linea, y nada mas.
func contarLineas(b []byte) int {
	if len(b) == 0 {
		return 0
	}
	n := bytes.Count(b, []byte("\n"))
	if b[len(b)-1] != '\n' {
		n++ // la ultima linea sin salto tambien es una linea
	}
	return n
}

// aObservacion convierte una linea en un hecho.
func aObservacion(linea string, n int) (estado.Observacion, error) {
	// EL VEREDICTO SE RECHAZA ANTES DE PARSEAR EL RESTO (invariante 13). Un
	// fichero que traiga un campo de juicio no se lee ignorandolo: se para, y
	// se dice por que. Ignorarlo dejaria a quien lo escribio creyendo que su
	// veredicto ha entrado en el expediente.
	if r := veredictoEnLaLinea(linea); r != "" {
		return estado.Observacion{}, fmt.Errorf("%w: linea %d trae %q. Arreglo: quitalo. "+
			"Un recolector entrega HECHOS («este recurso satisfizo el predicado el dia "+
			"D»), y el juicio lo compone plazum con la prueba del paquete delante y lo "+
			"confirma una persona. Invariante 13", ErrVeredicto, n, r)
	}
	var f filaEnDisco
	dec := json.NewDecoder(strings.NewReader(linea))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&f); err != nil {
		return estado.Observacion{}, fmt.Errorf("%w: linea %d: %v. Arreglo: cada linea es "+
			"un objeto JSON con prueba, recurso, satisfecho y cuando. Un campo que no se "+
			"conoce se rechaza en vez de descartarse en silencio: un dato que nadie lee "+
			"acaba mintiendo", ErrLineaIlegible, n, err)
	}
	if f.Prueba == "" {
		return estado.Observacion{}, fmt.Errorf("%w: linea %d. Arreglo: di el "+
			"identificador de la prueba del paquete que esta linea comprueba. Sin el, la "+
			"observacion no se puede colgar de ninguna obligacion", ErrSinPrueba, n)
	}
	if f.Recurso == "" {
		return estado.Observacion{}, fmt.Errorf("%w: linea %d. Arreglo: di sobre que "+
			"recurso se observo. Sin el, dos observaciones del mismo control no se "+
			"pueden distinguir y una tapa a la otra", ErrSinRecurso, n)
	}
	cuando, err := time.Parse(time.RFC3339, f.Cuando)
	if err != nil {
		return estado.Observacion{}, fmt.Errorf("%w: linea %d, cuando=%q: %v. Arreglo: "+
			"una fecha y hora en RFC 3339 (2026-03-01T09:00:00Z). Sin instante no hay "+
			"frescura, y sin frescura una observacion de hace tres anos vale igual que "+
			"la de esta manana", ErrSinInstante, n, f.Cuando, err)
	}
	var caduca time.Time
	if f.Caduca != "" {
		caduca, err = time.Parse(time.RFC3339, f.Caduca)
		if err != nil {
			return estado.Observacion{}, fmt.Errorf("%w: linea %d, caduca=%q: %v. "+
				"Arreglo: una fecha RFC 3339, o nada para que mande el TTL de la prueba. "+
				"Una caducidad que no se entiende NO es «sin caducidad»",
				ErrLineaIlegible, n, f.Caduca, err)
		}
	}
	// SATISFECHO AUSENTE Y SATISFECHO A FALSE SON DOS COSAS DISTINTAS, salvo
	// cuando la linea declara un error de recoleccion: ahi no hubo predicado que
	// evaluar y el bool no significa nada.
	if f.Satisfecho == nil && f.Error == "" {
		return estado.Observacion{}, fmt.Errorf("%w: linea %d sin `satisfecho` y sin "+
			"`error`. Arreglo: di si el predicado se cumplio (true/false), o di que no "+
			"se pudo recolectar con `error`. El campo ausente no se lee como false: eso "+
			"seria elegir por quien escribio el fichero", ErrLineaIlegible, n)
	}
	sat := false
	if f.Satisfecho != nil {
		sat = *f.Satisfecho
	}
	return estado.Observacion{
		Prueba:      f.Prueba,
		Recurso:     f.Recurso,
		Satisfecho:  sat,
		ErrorRecol:  f.Error,
		Recolectada: cuando,
		Caduca:      caduca,
		Recolector:  Nombre,
		Version:     fmt.Sprintf("v%d", Version),
	}, nil
}

// veredictoDeLaLinea son los campos que delatan un juicio en el fichero. Es el
// invariante 13 aplicado a la ENTRADA: la puerta de `nucleo/estado` vigila que
// el TIPO no gane un campo de veredicto, y esta vigila que uno no entre por el
// fichero pidiendo que se ignore.
var camposDeVeredicto = []string{
	"cumple", "conforme", "veredicto", "aprobado", "apto",
	"riesgo", "gravedad", "criticidad", "severidad", "puntuacion", "nota",
}

func veredictoEnLaLinea(linea string) string {
	var crudo map[string]json.RawMessage
	if json.Unmarshal([]byte(linea), &crudo) != nil {
		return ""
	}
	for k := range crudo {
		l := strings.ToLower(k)
		for _, c := range camposDeVeredicto {
			if strings.Contains(l, c) {
				return k
			}
		}
	}
	return ""
}

// sello ata el contenido a como se leyo, igual que `nucleo/censo`.
//
// NO ES UNA FIRMA CRIPTOGRAFICA y el nombre lo dice: es un sha256 sobre el
// contenido mas los cubos del recuento, cada campo con su longitud delante para
// que dos repartos distintos no den el mismo sello. Sirve para anotar en el
// ledger QUE fichero se leyo y COMO; no sirve para demostrar quien lo escribio.
func sello(b []byte, rec Recuento) string {
	suma := sha256.Sum256(b)
	h := sha256.New()
	for _, campo := range []string{
		hex.EncodeToString(suma[:]),
		Nombre,
		fmt.Sprintf("v%d", Version),
		rec.String(),
	} {
		fmt.Fprintf(h, "%d:%s|", len(campo), campo)
	}
	return hex.EncodeToString(h.Sum(nil))
}
