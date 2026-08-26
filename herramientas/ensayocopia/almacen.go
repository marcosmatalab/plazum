package main

// La instalacion en disco, la copia y la restauracion.
//
// QUE ES CADA ARTEFACTO, y por que estan separados. La separacion no es de
// comodidad: es la que hace que un borrado legal siga borrado despues de
// restaurar.
//
//	base.json       la base. La cadena v2 con sus lapidas y sus checkpoints, y
//	                las evidencias DENTRO (docs/guia.md 3.1: un solo fichero que
//	                respaldar). Es lo que replica Litestream.
//	keystore.json   las claves, una por entrada y una por evidencia. Fichero
//	                APARTE, replica APARTE y retencion corta (35 dias), porque
//	                destruir una clave tiene que destruirla tambien en las
//	                copias dentro del plazo declarado.
//	maestra.key     la privada del operador, la que firma lapidas y cierra
//	                checkpoints. NO SE COPIA. Su copia es de custodia (frase o
//	                QR impresos), no de replica automatica.
//	expediente.json un expediente emitido, si lo hay. Va en la copia porque es
//	                lo que el receptor recibio y tiene que seguir verificando.
//
// POR QUE HOY LA BASE ES UN JSON Y NO UN SQLite. El adaptador de almacen no
// esta construido (ETAPAS.md lo dice en la casilla de blobs: "la tabla SQLite y
// el chunking >32 MB van con el adaptador de almacen"). Este ensayo no lo
// adelanta ni lo finge: monta la instalacion con los tipos del nucleo, que son
// los definitivos, y la escribe en el formato que hay hoy. Lo que el ensayo
// mide (que la copia devuelve una cadena que verifica y unas supresiones que
// siguen suprimidas) no cambia cuando la base pase a ser un fichero de SQLite,
// porque no depende del formato sino de los bytes. Lo que SI queda sin
// ejercitar hasta entonces es Litestream en si: esta documentado en
// docs/copias.md y anotado como hueco en docs/pendientes.md.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"plazum/nucleo/blobs"
	"plazum/nucleo/ledger"
)

// Nombres de los artefactos. Van como constantes para que el que escribe y el
// que lee no puedan discrepar, igual que en adaptadores/actualizador.
const (
	NombreBase       = "base.json"
	NombreKeystore   = "keystore.json"
	NombreMaestra    = "maestra.key"
	NombreExpediente = "expediente.json"
	NombreManifiesto = "manifiesto.json"
)

// Centinelas. Con errors.Is y no comparando texto: los mensajes cambian, y un
// control negativo que afirma con una subcadena acaba dando verde por la
// subcadena equivocada.
var (
	// ErrFaltaBase: la copia no trae la base.
	ErrFaltaBase = errors.New("la copia no trae la base")
	// ErrFaltaKeystore: la copia no trae el keystore. Es el caso que mas
	// enganna, porque la base restaura sin protestar y la cadena sigue
	// verificando: lo que se ha perdido es todo el CONTENIDO.
	ErrFaltaKeystore = errors.New("la copia no trae el keystore")
	// ErrArtefactoNoCuadra: un artefacto no hashea a lo que dice el manifiesto.
	ErrArtefactoNoCuadra = errors.New("un artefacto de la copia no cuadra con su manifiesto")
	// ErrCopiaSinManifiesto: no hay manifiesto, asi que no se puede saber si lo
	// que hay es lo que se copio.
	ErrCopiaSinManifiesto = errors.New("la copia no trae manifiesto")
	// ErrNombreDeArtefactoInvalido: el manifiesto nombra algo que no es un
	// fichero suelto de la replica. Ver nombreDeArtefactoValido.
	ErrNombreDeArtefactoInvalido = errors.New("el manifiesto nombra un artefacto que sale del directorio de la copia")
)

// nombreDeArtefactoValido exige que un nombre del manifiesto sea un fichero
// suelto del directorio de la replica.
//
// HALLAZGO DE LA PASADA ADVERSARIA, y es de escritura arbitraria. Restaurar
// recorre lo que el manifiesto DECLARA y hace filepath.Join con el destino. Un
// manifiesto con "../../algo" o con una ruta absoluta hace que restaurar lea
// fuera de la replica y ESCRIBA fuera del destino, con los permisos de quien
// restaura. Y el sitio donde vive una replica (un bucket, un NAS, un disco que
// se lleva alguien) es justo donde no se puede dar por hecho que nadie escribe.
//
// La comprobacion es la mas estrecha posible a proposito: los artefactos de una
// copia son los tres de `copiables`, todos nombres pelados.
func nombreDeArtefactoValido(n string) bool {
	if n == "" || n == "." || n == ".." {
		return false
	}
	if filepath.IsAbs(n) {
		return false
	}
	// Las dos barras, no solo la del sistema: un manifiesto escrito en Linux se
	// restaura en Windows y al reves, y en Windows la barra normal tambien
	// separa.
	if strings.ContainsAny(n, `/\`) {
		return false
	}
	return n == filepath.Base(n)
}

// EntradaClara es el contenido de una entrada de la cadena antes de cifrarse.
// Se guarda como JSON dentro de la entrada.
type EntradaClara struct {
	Prueba  string `json:"prueba"`
	Recurso string `json:"recurso"`
	Detalle string `json:"detalle"`
}

// Evidencia es un blob guardado dentro de la base, con la entrada de la cadena
// que lo ancla.
//
// Por que lleva Entrada. Sin ella, borrar legalmente una entrada dejaria su
// evidencia huerfana y viva: la lapida diria que se suprimio y el PDF seguiria
// abriendose. El borrado legal destruye la clave de la entrada Y la de su
// evidencia, y este campo es lo que permite comprobarlo al restaurar.
type Evidencia struct {
	Hash       string `json:"hash"`
	Nonce      []byte `json:"nonce"`
	Cifrado    []byte `json:"cifrado"`
	Compromiso []byte `json:"compromiso"`
	Entrada    uint64 `json:"entrada"`
}

// Blob devuelve la evidencia en la forma que entiende nucleo/blobs.
func (e Evidencia) Blob() blobs.Blob {
	return blobs.Blob{Hash: e.Hash, Nonce: e.Nonce, Cifrado: e.Cifrado, Compromiso: e.Compromiso}
}

// Base es lo que Litestream replica: la cadena y las evidencias, juntas.
type Base struct {
	// Generacion es cuando se escribio esta base. Sirve para contrastarla con
	// la generacion del keystore: una pareja descuadrada es la forma normal de
	// resucitar una clave borrada.
	Generacion string          `json:"generacion"`
	Cadena     ledger.CadenaV2 `json:"cadena"`
	Evidencias []Evidencia     `json:"evidencias"`
}

// Keystore es el fichero de claves. Separado de la base a proposito.
type Keystore struct {
	Generacion string `json:"generacion"`
	// Entradas: indice de la entrada -> clave en hex.
	Entradas map[string]string `json:"entradas"`
	// Evidencias: direccion del blob -> clave en hex.
	Evidencias map[string]string `json:"evidencias"`
}

// ClaveEntrada devuelve la clave de una entrada, si el keystore la tiene.
func (k Keystore) ClaveEntrada(i uint64) ([]byte, bool) {
	return claveHex(k.Entradas, fmt.Sprint(i))
}

// ClaveEvidencia devuelve la clave de una evidencia, si el keystore la tiene.
func (k Keystore) ClaveEvidencia(hash string) ([]byte, bool) {
	return claveHex(k.Evidencias, hash)
}

// Todas devuelve TODAS las claves que el keystore contiene, sin mirar a que
// pertenecen. Existe para poder comprobar la propiedad fuerte de la supresion:
// que ninguna clave del keystore restaurado abre una entrada con lapida, no
// solo la que estaba apuntada a su indice. Sin esto, reponer la clave borrada
// bajo otro indice pasaria la comprobacion.
func (k Keystore) Todas() [][]byte {
	var out [][]byte
	for _, m := range []map[string]string{k.Entradas, k.Evidencias} {
		claves := make([]string, 0, len(m))
		for c := range m {
			claves = append(claves, c)
		}
		sort.Strings(claves) // determinismo: el informe no puede cambiar de orden
		for _, c := range claves {
			if b, ok := claveHex(m, c); ok {
				out = append(out, b)
			}
		}
	}
	return out
}

func claveHex(m map[string]string, clave string) ([]byte, bool) {
	h, ok := m[clave]
	if !ok || h == "" {
		return nil, false
	}
	b, err := hex.DecodeString(h)
	if err != nil {
		return nil, false
	}
	return b, true
}

// Manifiesto dice que artefactos tiene la copia y con que digest.
//
// LO QUE EL MANIFIESTO NO ES, dicho aqui para que nadie se apoye en el mas de
// lo que aguanta: no es integridad contra un adversario. Quien pueda escribir
// en la replica puede reescribir tambien el manifiesto. Sirve contra la copia a
// medias, el disco que miente y el rsync interrumpido, que es lo que de verdad
// pasa. La integridad frente a alguien que quiere enganar la da la cadena: los
// hashes encadenados, las firmas de las lapidas y el checkpoint, y todos ellos
// se comprueban contra claves que aporta el receptor y NO viajan en la copia.
type Manifiesto struct {
	Instante string `json:"instante"`
	// Artefactos: nombre de fichero -> sha256 en hex.
	Artefactos map[string]string `json:"artefactos"`
}

// ---------------------------------------------------------------------------
// Escritura y lectura de la instalacion
// ---------------------------------------------------------------------------

func escribirJSON(ruta string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("no puedo serializar %s: %w", filepath.Base(ruta), err)
	}
	b = append(b, '\n')
	if err := os.MkdirAll(filepath.Dir(ruta), 0o750); err != nil {
		return fmt.Errorf("no puedo crear el directorio de %s: %w", ruta, err)
	}
	if err := os.WriteFile(ruta, b, 0o600); err != nil {
		return fmt.Errorf("no puedo escribir %s: %w", ruta, err)
	}
	return nil
}

func leerJSON(ruta string, v any) error {
	b, err := os.ReadFile(ruta) // #nosec G304 -- ruta de la instalacion, que teclea el operador
	if err != nil {
		return err
	}
	if err := json.Unmarshal(b, v); err != nil {
		return fmt.Errorf("%s no es JSON valido: %w", ruta, err)
	}
	return nil
}

// CargarBase lee la base de una instalacion.
func CargarBase(dir string) (Base, error) {
	var b Base
	err := leerJSON(filepath.Join(dir, NombreBase), &b)
	if os.IsNotExist(err) {
		return b, fmt.Errorf("%w: no esta %s. Sin la base no hay cadena que verificar; "+
			"restaura la replica de la base antes de seguir. %w",
			ErrFaltaBase, filepath.Join(dir, NombreBase), err)
	}
	return b, err
}

// CargarKeystore lee el keystore de una instalacion.
//
// El error de ausencia es el mensaje mas importante de esta herramienta, porque
// es el fallo que MAS se parece a un exito: la base restaura, la cadena
// verifica, las lapidas firman, y no se puede leer ni una sola entrada.
func CargarKeystore(dir string) (Keystore, error) {
	var k Keystore
	ruta := filepath.Join(dir, NombreKeystore)
	err := leerJSON(ruta, &k)
	if os.IsNotExist(err) {
		return k, fmt.Errorf("%w: no esta %s.\n"+
			"  Que se ha perdido: TODO el contenido de la cadena. Las entradas siguen\n"+
			"  encadenadas y las lapidas siguen firmando, asi que la cadena verifica y\n"+
			"  parece que la restauracion ha ido bien, pero sin las claves no se abre ni\n"+
			"  una entrada ni una evidencia.\n"+
			"  Arreglo: el keystore tiene replica PROPIA, con retencion de 35 dias\n"+
			"  (docs/copias.md). Restaura la generacion del keystore que corresponda a\n"+
			"  esta base y vuelve a lanzar el ensayo. Si no queda ninguna generacion\n"+
			"  dentro del plazo, el contenido es irrecuperable y eso hay que decirlo, no\n"+
			"  descubrirlo el dia de la auditoria",
			ErrFaltaKeystore, ruta)
	}
	if err != nil {
		return k, err
	}
	if k.Entradas == nil {
		k.Entradas = map[string]string{}
	}
	if k.Evidencias == nil {
		k.Evidencias = map[string]string{}
	}
	return k, nil
}

// ---------------------------------------------------------------------------
// Copia y restauracion
// ---------------------------------------------------------------------------

// copiables son los artefactos que entran en la copia, EN ORDEN.
//
// El orden importa y es el contrario al que sale intuitivo. Primero la base,
// despues el keystore. Asi, si la copia se corta por la mitad, lo que queda es
// una base sin claves (contenido ilegible, recuperable con la generacion
// anterior del keystore) y nunca unas claves sin base, que es la pareja que
// permite abrir contenido que la base ya habia suprimido.
var copiables = []string{NombreBase, NombreKeystore, NombreExpediente}

// noSeCopia son los ficheros de la instalacion que la replica NO lleva, con el
// motivo. Se declara aqui, y no en un comentario, porque es lo que imprime el
// ensayo: un operador tiene que poder leer que NO tiene en la copia sin abrir
// el codigo.
var noSeCopia = []struct{ Fichero, Motivo string }{
	{NombreMaestra, "la clave maestra del operador. Su copia es de custodia (frase o QR " +
		"impresos), no de replica: una privada que viaja en cada backup esta en tantos " +
		"sitios como copias haya. El historico verifica sin ella; lo que no se puede es " +
		"firmar lapidas ni cerrar checkpoints nuevos hasta reponerla"},
	{"paquetes/", "el corpus instalado. Se reinstala desde el canal firmado y se comprueba " +
		"contra su digest, que es mas fuerte que restaurar los bytes de una copia"},
}

func sha256Fichero(ruta string) (string, error) {
	b, err := os.ReadFile(ruta) // #nosec G304 -- ruta de la instalacion, que teclea el operador
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:]), nil
}

// Copiar hace la copia de una instalacion. Devuelve el manifiesto escrito.
func Copiar(datos, replica, instante string) (Manifiesto, error) {
	m := Manifiesto{Instante: instante, Artefactos: map[string]string{}}
	if err := os.MkdirAll(replica, 0o750); err != nil {
		return m, fmt.Errorf("no puedo crear el destino de la copia %s: %w", replica, err)
	}
	for _, nombre := range copiables {
		origen := filepath.Join(datos, nombre)
		b, err := os.ReadFile(origen) // #nosec G304 -- ruta de la instalacion
		if os.IsNotExist(err) {
			if nombre == NombreExpediente {
				continue // no toda instalacion ha emitido un expediente
			}
			return m, fmt.Errorf("la instalacion %s no tiene %s, asi que no hay nada que "+
				"copiar de ella: %w", datos, nombre, err)
		}
		if err != nil {
			return m, fmt.Errorf("no puedo leer %s: %w", origen, err)
		}
		// #nosec G703 -- nombre viene de `copiables`, que son constantes de este
		// fichero, y replica es la ruta que teclea el operador. Aqui no entra
		// nada escrito por un tercero: lo que si entra es en Restaurar, y alli
		// se valida con nombreDeArtefactoValido.
		if err := os.WriteFile(filepath.Join(replica, nombre), b, 0o600); err != nil {
			return m, fmt.Errorf("no puedo escribir la copia de %s: %w", nombre, err)
		}
		h := sha256.Sum256(b)
		m.Artefactos[nombre] = hex.EncodeToString(h[:])
	}
	if err := escribirJSON(filepath.Join(replica, NombreManifiesto), m); err != nil {
		return m, err
	}
	return m, nil
}

// Restaurar deja en destino lo que hay en la replica, comprobando cada
// artefacto contra el manifiesto.
//
// Falla CERRADO en los dos casos que importan: sin manifiesto no restaura (no
// se puede saber si lo que hay es lo que se copio) y sin keystore no restaura
// (una base sola parece una restauracion buena y no lo es).
func Restaurar(replica, destino string) error {
	var m Manifiesto
	err := leerJSON(filepath.Join(replica, NombreManifiesto), &m)
	if os.IsNotExist(err) {
		return fmt.Errorf("%w: no esta %s. Sin el no se puede saber si la copia esta "+
			"entera ni si los bytes son los que se copiaron. Arreglo: restaura desde una "+
			"generacion que lo tenga",
			ErrCopiaSinManifiesto, filepath.Join(replica, NombreManifiesto))
	}
	if err != nil {
		return err
	}
	if err := os.MkdirAll(destino, 0o750); err != nil {
		return fmt.Errorf("no puedo crear el destino %s: %w", destino, err)
	}

	// Los dos obligatorios primero y por su nombre, y se exigen EN EL
	// MANIFIESTO ademas de en el disco.
	//
	// HALLAZGO DE LA PASADA ADVERSARIA. Antes solo se miraba el disco, y el
	// bucle de mas abajo copia lo que el manifiesto DECLARA. Una replica con
	// keystore.json en el disco pero sin la entrada en el manifiesto pasaba la
	// comprobacion y no copiaba el keystore: si el destino no estaba vacio (una
	// restauracion sobre la instalacion que se quiere reparar, que es el caso
	// normal), el keystore VIEJO que ya habia alli se quedaba. O sea, la clave
	// destruida volvia sin que nadie tocara una clave.
	for _, obligatorio := range []struct {
		nombre string
		err    error
	}{{NombreBase, ErrFaltaBase}, {NombreKeystore, ErrFaltaKeystore}} {
		_, enManifiesto := m.Artefactos[obligatorio.nombre]
		_, errDisco := os.Stat(filepath.Join(replica, obligatorio.nombre))
		if !enManifiesto || os.IsNotExist(errDisco) {
			if obligatorio.nombre == NombreKeystore {
				// El mensaje largo vive en CargarKeystore; aqui se cita el
				// mismo centinela para que el que restaura y el que verifica
				// hablen del mismo problema.
				return fmt.Errorf("%w: no esta %s en la replica.\n"+
					"  Restaurar solo la base deja una instalacion que VERIFICA y no LEE:\n"+
					"  cadena entera, lapidas firmadas, y ni una entrada abrible.\n"+
					"  Arreglo: el keystore tiene su propia replica (docs/copias.md).\n"+
					"  Restaura la generacion que corresponda a esta base",
					ErrFaltaKeystore, filepath.Join(replica, obligatorio.nombre))
			}
			return fmt.Errorf("%w: no esta %s en la replica", obligatorio.err,
				filepath.Join(replica, obligatorio.nombre))
		}
	}

	nombres := make([]string, 0, len(m.Artefactos))
	for n := range m.Artefactos {
		nombres = append(nombres, n)
	}
	sort.Strings(nombres)
	for _, nombre := range nombres {
		if !nombreDeArtefactoValido(nombre) {
			return fmt.Errorf("%w: el manifiesto declara %q.\n"+
				"  Un nombre con separadores o con .. sale del directorio de la replica al\n"+
				"  leerlo y sale del destino al escribirlo, o sea que restaurar una copia\n"+
				"  manipulada escribiria donde el manifiesto diga, con los permisos del que\n"+
				"  restaura.\n"+
				"  Arreglo: los artefactos de una copia son ficheros sueltos del directorio de\n"+
				"  la replica. Si esta copia trae otra cosa, no es una copia de este producto",
				ErrNombreDeArtefactoInvalido, nombre)
		}
		origen := filepath.Join(replica, nombre)
		visto, err := sha256Fichero(origen)
		if err != nil {
			return fmt.Errorf("el manifiesto declara %s y no se puede leer: %w", nombre, err)
		}
		if visto != m.Artefactos[nombre] {
			return fmt.Errorf("%w: %s hashea a %s y el manifiesto dice %s. "+
				"Arreglo: la copia esta corrupta o incompleta, restaura otra generacion",
				ErrArtefactoNoCuadra, nombre, visto[:12], m.Artefactos[nombre][:12])
		}
		b, err := os.ReadFile(origen) // #nosec G304 -- ruta de la replica
		if err != nil {
			return err
		}
		// #nosec G703 -- ESTA es la ruta que si lleva entrada de un tercero (el
		// nombre sale del manifiesto de la replica), y por eso arriba, en este
		// mismo bucle, pasa por nombreDeArtefactoValido antes de llegar aqui:
		// ni separadores, ni "..", ni rutas absolutas. El analisis de taint de
		// gosec no ve esa guarda; la ve el test
		// TestRestaurarSeNiegaAUnNombreDeArtefactoQueSaleDelDirectorio, que
		// prueba las cuatro formas y se pone rojo si la guarda desaparece.
		if err := os.WriteFile(filepath.Join(destino, nombre), b, 0o600); err != nil {
			return fmt.Errorf("no puedo escribir %s en %s: %w", nombre, destino, err)
		}
	}
	return nil
}

// DentroDe dice si ruta cae dentro de dir. Se usa para negarse a copiar el
// ancla de confianza, y no es una comprobacion de higiene: si el fichero con la
// clave publica del operador viaja DENTRO de lo que se restaura, entonces quien
// pueda escribir en la copia se escribe tambien la clave con la que se comprueba
// su propia firma, y la verificacion se compara consigo misma. Es el mismo
// hallazgo que ya tuvo el expediente en la etapa 1, aplicado a las copias.
func DentroDe(dir, ruta string) bool {
	a, err := filepath.Abs(dir)
	if err != nil {
		return false
	}
	b, err := filepath.Abs(ruta)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(a, b)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
