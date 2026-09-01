// Package censo es la ingesta manual firmada: un fichero de cuentas y permisos
// que sube una persona, convertido en una INSTANTANEA con identidad propia.
//
// POR QUE EXISTE Y QUE ESTABA ESPERANDOLO. La UAR (revision de accesos) y el
// seguimiento de formacion necesitan una lista de personas desde el dia uno, y
// los conectores no llegan hasta la etapa 6. La guia lo dice sin vergüenza
// (§6.1): la campana del ano 1 funciona sobre datos importados a mano mas SCIM.
// Este paquete es el "a mano", y es la fuente de la que cuelga todo lo demas.
//
// EL FICHERO SUBIDO ES DATO DE EMISOR, y la doctrina de la casa aplica hacia
// dentro igual que hacia fuera. Nadie de aqui escribio ese CSV: lo exporto un
// IdP ajeno, lo abrio alguien con Excel y lo volvio a guardar. Asi que se trata
// como se trata un token RFC 3161: se parsea sin fiarse, se dice exactamente que
// se entendio, y **se sella con el hash de los BYTES**, no del resultado.
//
// El hash es de los bytes y no de las filas por un motivo concreto: una
// conclusion de la UAR tiene que poder reproducirse. Con el hash del fichero,
// cualquiera con el mismo fichero vuelve a sacar las mismas filas y comprueba
// que la revision se hizo sobre eso. Con el hash de las filas, la unica forma de
// comprobarlo seria fiarse de este parser, que es justo lo que no se puede.
//
// LO QUE SE MIDIO ANTES DE ESCRIBIR UNA LINEA, contra `encoding/csv` de la
// biblioteca estandar y con ficheros hostiles de verdad. Los cuatro resultados
// mandan sobre el diseno entero:
//
//  1. **El BOM se cuela en el nombre de la primera columna.** Un fichero de
//     Excel empieza por EF BB BF y la cabecera sale como "\ufeffusuario". La
//     busqueda de la columna "usuario" falla, y el error que le llega a quien lo
//     sube es "falta la columna usuario" **sobre un fichero que la tiene**.
//
//  2. **cp1252 no da NINGUN error.** "Martinez" con la i acentuada en cp1252 es
//     el byte 0xED, y csv lo deja pasar tal cual: sale una cadena que no es
//     UTF-8 valido y viaja hasta la pantalla. Si el acento esta en el
//     IDENTIFICADOR y no en el rotulo, lo que se ha corrompido en silencio es la
//     identidad por la que se empareja todo (invariante 7). Es la familia del
//     no-op silencioso de M55: no falla, DEVUELVE algo.
//
//  3. **Una fila con una columna de mas mata el fichero entero.** Con el
//     FieldsPerRecord por defecto, ReadAll devuelve CERO filas y un error. Si
//     quien llama registra el error y sigue, la campana revisa un censo vacio, y
//     un censo vacio se lee igual que un censo limpio.
//
//  4. **Y el peor, el que decide el diseno: una comilla sin cerrar SE COME LAS
//     FILAS QUE SIGUEN, en silencio.** Medido: fichero de cinco lineas, comilla
//     abierta en la 3, el lector devuelve la 1 y la 2, un error, y "fin".
//     **Tres personas desaparecen de la revision y no hay ni un error por fila
//     perdida.** Leer fila a fila no lo arregla: el lector se traga el resto
//     buscando la comilla de cierre.
//
// DE AHI SALE LA UNICA GUARDA QUE SIRVE, y es la ley de conservacion aplicada al
// fichero: **las LINEAS de datos que cuenta un camino independiente tienen que
// cuadrar con las que cubren los cubos** (una por fila legible, el rango entero
// por cada ilegible, una por duplicada). Si no cuadra, la instantanea NO CARGA.
// No se avisa y se sigue: se para. Un descuadre significa que alguien
// desaparecio del censo, y una campana de accesos que certifica completitud
// sobre un censo al que le faltan filas es la peor afirmacion falsa que puede
// hacer este producto, porque **la firma una persona**.
//
// Y EL CONTADOR INDEPENDIENTE NO SABE DE COMILLAS, A PROPOSITO. Es la parte del
// diseno que mas facil seria estropear "mejorandola": si contara respetando las
// comillas como el parser, se tragaria exactamente las mismas lineas que el
// parser se traga y cuadraria siempre. Seria una guarda que confirma al
// vigilado. La ceguera es lo que la hace independiente, igual que los tres
// metodos ciegos entre si del barrido de URL.
//
// El precio esta dicho y se acepta: un campo con un salto de linea DENTRO, que
// es CSV legitimo, hace que no cuadre y que el fichero no cargue. Cuesta mirar
// el fichero. El fallo contrario cuesta certificar de mas.
//
// LA MINIMIZACION NO ES UN ADORNO. Una lista de cuentas y permisos es dato
// personal, y este paquete se queda con lo minimo para poder revisar: el sistema
// del que sale, el identificador estable de la cuenta, el permiso, y un rotulo
// legible para que quien revisa sepa a quien esta aprobando. **Toda columna que
// no este declarada se descarta, y se dice cuantas y cuales**, porque un
// descarte mudo de columnas es el mismo fallo que un descarte mudo de filas
// visto desde el otro eje. De las filas ilegibles se guarda el NUMERO DE LINEA y
// el motivo, nunca el contenido: para arreglarla hace falta saber cual es, no
// que decia.
//
// LA IDENTIDAD ES ESTABLE O NO ES. La fila se identifica por
// sistema|cuenta|permiso, tres campos que estan DENTRO de los bytes firmados
// (invariante 7). El ROTULO no identifica nunca: dos personas se llaman igual, y
// una persona cambia de apellido.
package censo

import (
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
)

// Los centinelas. Todos los caminos de "esto no carga" son distinguibles, porque
// quien los recibe tiene que poder hacer cosas distintas.
var (
	// ErrDescuadre es el importante: el fichero tenia mas registros de los que
	// han salido por algun lado. Alguien ha desaparecido.
	ErrDescuadre = errors.New("censo: el recuento de registros no cuadra")
	// ErrSinColumnaDeCuenta: no se ha podido identificar la columna del
	// identificador, que es lo unico sin lo que no hay nada que revisar.
	ErrSinColumnaDeCuenta = errors.New("censo: no se encuentra la columna del identificador de cuenta")
	// ErrColumnaAmbigua: dos columnas del fichero valen para lo mismo. No se
	// adivina: se pregunta.
	ErrColumnaAmbigua = errors.New("censo: dos columnas compiten por el mismo papel")
	// ErrOpciones: faltan datos de quien sube el fichero.
	ErrOpciones = errors.New("censo: faltan datos obligatorios para sellar la instantanea")
	// ErrVacio: el fichero no trae ni una fila de datos.
	ErrVacio = errors.New("censo: el fichero no tiene ni una fila de datos")
)

// Fila es una unidad de revision, y la unidad NO es la persona: es el acceso.
//
// Quien revisa no aprueba a Ana, aprueba "Ana tiene admin en el ERP". Una
// persona con cinco permisos son cinco decisiones, y eso es lo correcto: la
// mitad de las revocaciones de una UAR real son de un permiso concreto de
// alguien que sigue en la empresa.
type Fila struct {
	// Sistema NO sale del fichero: lo declara quien sube, porque un export de
	// un IdP no sabe como llama la organizacion al sistema que audita.
	Sistema string
	// Cuenta es el IDENTIFICADOR ESTABLE. Nunca el nombre.
	Cuenta string
	// Permiso es el acceso concreto.
	Permiso string
	// Rotulo es para que una persona reconozca la cuenta. NO IDENTIFICA NADA:
	// dos personas se llaman igual y una persona cambia de apellido.
	Rotulo string
	// Linea es donde estaba en el fichero, para poder volver a mirarla.
	Linea int
}

// Clave es la identidad de la fila, y sus tres partes estan DENTRO de los bytes
// que cubre el hash (invariante 7). Nunca se empareja por posicion ni por orden:
// reordenar el CSV no puede mover ninguna decision.
func (f Fila) Clave() string {
	return f.Sistema + "|" + f.Cuenta + "|" + f.Permiso
}

// Ilegible es una fila que no se pudo leer. ES UN CUBO, no un descarte.
//
// No lleva el contenido de la linea a proposito: es dato personal y para
// arreglarla hace falta saber CUAL es, no que decia.
type Ilegible struct {
	Desde  int // primera linea del registro
	Hasta  int // ultima; distinta de Desde cuando una comilla se comio varias
	Motivo string
}

// Duplicada es una fila cuya clave ya habia salido. Se cuenta y se dice, y no se
// vuelve a revisar: pedirle a una persona dos veces la misma decision es como se
// consiguen dos decisiones contrarias sobre el mismo acceso.
type Duplicada struct {
	Clave   string
	Linea   int
	Primera int // donde salio la primera vez
}

// Notas es todo lo que hubo que DECIDIR al leer el fichero. Va dentro de la
// instantanea porque una decision de lectura que no se cuenta es una suposicion
// que nadie puede revisar.
type Notas struct {
	Codificacion      string   // "utf-8" o "cp1252 (el fichero no era UTF-8 valido)"
	QuitadoElBOM      bool     //
	Separador         string   // el que se dedujo, dicho con todas las letras
	Cabeceras         []string // las que traia el fichero, tal cual
	ColumnasIgnoradas []string // las que no se guardan, por minimizacion
	Avisos            []string // lo que se vio y no se corrigio, porque adivinar es peor
}

// Opciones son los datos que NO estan en el fichero y sin los cuales la
// instantanea no significa nada.
//
// SU VALOR CERO ESTA PROHIBIDO (invariante 8). Una instantanea sin quien la
// subio, sin cuando y sin de que sistema es una lista de nombres: no se puede
// atar a nada, no se puede auditar y no certifica nada. En una frontera de
// confianza el cero tiene que ser el restrictivo, asi que Tomar lo rechaza en
// vez de rellenar con valores por defecto amables.
type Opciones struct {
	Sistema   string
	Fuente    string    // "export de usuarios de Entra ID", lo escribe quien sube
	Quien     string    // identificador estable de quien sube, no su nombre
	Tomada    time.Time // el instante entra como DATO: nucleo no lee el reloj
	Retencion string    // cuanto se guarda y por que. Se declara o no se sube
	Columnas  Columnas
}

// Columnas declara que nombres de columna valen para cada papel.
//
// CADA IdP LES PONE UN NOMBRE DISTINTO y no hay estandar: userName, login,
// correo, email, samAccountName, id. Por eso es una lista de sinonimos y no un
// nombre fijo. Lo que NO se hace es adivinar cuando hay dos: si el fichero trae
// "usuario" y "email" y los dos estan declarados como identificador, eso es
// ErrColumnaAmbigua, no una eleccion. Adivinar la columna de identidad es
// adivinar a quien se esta revisando.
type Columnas struct {
	Cuenta  []string
	Permiso []string
	Rotulo  []string
}

// ColumnasHabituales son los nombres que se ven en la practica. Se comparan sin
// mayusculas, sin acentos y sin espacios.
func ColumnasHabituales() Columnas {
	return Columnas{
		Cuenta:  []string{"usuario", "cuenta", "login", "username", "user", "samaccountname", "upn", "id", "identificador"},
		Permiso: []string{"permiso", "rol", "role", "grupo", "group", "perfil", "privilegio", "acceso"},
		Rotulo:  []string{"nombre", "name", "displayname", "nombrecompleto", "descripcion"},
	}
}

// Instantanea es el fichero convertido en dato, con su identidad.
type Instantanea struct {
	// Hash es el sha256 en hexadecimal de los BYTES del fichero, tal como se
	// subio, con BOM incluido si lo traia. Es la identidad, y toda conclusion de
	// una campana cuelga de el.
	Hash string

	Sistema   string
	Fuente    string
	Quien     string
	Tomada    time.Time
	Retencion string

	Filas      []Fila
	Ilegibles  []Ilegible
	Duplicadas []Duplicada
	Notas      Notas

	// LineasDeDatos es cuantas habia en el fichero segun el contador
	// independiente, sin contar la cabecera. Se guarda porque es la mitad que
	// hace comprobable la ley de conservacion mas tarde, sin volver a tener el
	// fichero delante.
	LineasDeDatos int
}

// LineasCubiertas son las lineas de datos que los cubos dan por explicadas.
//
// Una fila legible explica su linea. Una ilegible explica TODO SU RANGO, que es
// lo que hace que una comilla sin cerrar no se lleve dos personas por delante
// sin que nadie las eche de menos: el rango dice cuantas y cuales.
func (i Instantanea) LineasCubiertas() int {
	n := len(i.Filas) + len(i.Duplicadas)
	for _, il := range i.Ilegibles {
		n += il.Hasta - il.Desde + 1
	}
	return n
}

// Cuenta es la LEY DE CONSERVACION del fichero, impresa.
//
// Toda linea de datos del fichero termina en exactamente un cubo. Que la suma
// cuadre no es una comprobacion interna: es lo que permite a quien firma la
// campana decir "he revisado N de N" y que sea verdad.
func (i Instantanea) Cuenta() map[string]int {
	return map[string]int{
		"legibles":         len(i.Filas),
		"ilegibles":        len(i.Ilegibles),
		"duplicadas":       len(i.Duplicadas),
		"lineas cubiertas": i.LineasCubiertas(),
		"lineas de datos":  i.LineasDeDatos,
	}
}

// Cuadra dice si los cubos cubren todas las lineas de datos. Si alguna vez
// devuelve false sobre una instantanea que salio de Tomar es un fallo de este
// paquete: Tomar no devuelve instantaneas descuadradas, se para antes.
func (i Instantanea) Cuadra() bool { return i.LineasCubiertas() == i.LineasDeDatos }

// Sello es la identidad de LA LECTURA, y es de lo que tiene que colgar una
// conclusion de campana. Hash es la identidad del FICHERO, que es menos.
//
// POR QUE HACEN FALTA LOS DOS, y salio de preguntar por que campo casa el
// emparejamiento y si ese campo esta firmado (invariante 7). La clave de una
// fila es sistema|cuenta|permiso: `cuenta` y `permiso` salen de los bytes que
// cubre el hash, pero **`sistema` lo declara quien sube**. El mismo fichero
// subido dos veces diciendo que es de dos sistemas distintos da el MISMO hash y
// claves DISTINTAS, asi que "la revision cuelga del hash exacto" no bastaba:
// dos campanas podian compartir hash y estar revisando cosas que no se pueden
// comparar.
//
// El sello cubre el fichero MAS todo lo que decide como se leyo. El hash sigue
// publicandose aparte porque es lo que un tercero puede recalcular con el
// fichero en la mano; el sello es lo que ata la conclusion.
func (i Instantanea) Sello() string {
	h := sha256.New()
	// Cada campo con su longitud delante: sin eso, ("ab","c") y ("a","bc")
	// darian el mismo sello, que es el mismo fallo de concatenar sin separar
	// que ya cuesta caro en cualquier canonicalizacion.
	for _, campo := range []string{i.Hash, i.Sistema, i.Fuente, i.Quien,
		i.Tomada.UTC().Format(time.RFC3339Nano), i.Retencion} {
		fmt.Fprintf(h, "%d:%s|", len(campo), campo)
	}
	return hex.EncodeToString(h.Sum(nil))
}

// Tomar lee el fichero y sella la instantanea.
//
// datos son los bytes tal como llegaron. No se normalizan antes de hashear:
// el hash es de lo que subio la persona, no de lo que este paquete entendio.
func Tomar(datos []byte, o Opciones) (Instantanea, error) {
	if err := o.validar(); err != nil {
		return Instantanea{}, err
	}

	if esUTF16(datos) {
		return Instantanea{}, fmt.Errorf("%w: el fichero empieza por un BOM de UTF-16, que es lo que "+
			"guarda la opcion \"Texto Unicode\" de Excel.\n"+
			"  No se convierte aqui a proposito: esa opcion ademas separa por tabuladores, y "+
			"decodificar una cosa mientras se adivina la otra es como salen censos plausibles y "+
			"equivocados.\n"+
			"  Arreglo: en Excel, Guardar como -> \"CSV UTF-8 (delimitado por comas)\"", ErrCodificacion)
	}

	suma := sha256.Sum256(datos)
	ins := Instantanea{
		Hash:      hex.EncodeToString(suma[:]),
		Sistema:   o.Sistema,
		Fuente:    o.Fuente,
		Quien:     o.Quien,
		Tomada:    o.Tomada,
		Retencion: o.Retencion,
	}

	texto, notas := aTexto(datos)
	ins.Notas = notas

	// El contador independiente, ANTES de parsear y con otro codigo que no sabe
	// de comillas. Ver contarLineas: la ceguera es lo que lo hace servir.
	lineasTotales := contarLineas(texto)
	if lineasTotales < 2 {
		return Instantanea{}, fmt.Errorf("%w: solo se ha encontrado %d linea (hacen falta al menos "+
			"la cabecera y una fila).\n  Arreglo: comprobar que el fichero es el que se queria subir "+
			"y que no se ha guardado vacio", ErrVacio, lineasTotales)
	}
	// La cabecera no es una linea de datos.
	ins.LineasDeDatos = lineasTotales - 1

	sep, empatados := deducirSeparador(texto)
	ins.Notas.Separador = nombreDelSeparador(sep)
	for _, e := range empatados {
		ins.Notas.Avisos = append(ins.Notas.Avisos, fmt.Sprintf(
			"la cabecera tiene tantos %s como %s: se ha leido con %s. Si no era ese, las columnas "+
				"saldran pegadas y el error dira que falta la de identificador",
			nombreDelSeparador(sep), nombreDelSeparador(e), nombreDelSeparador(sep)))
	}

	lector := csv.NewReader(strings.NewReader(texto))
	lector.Comma = sep
	// -1 a proposito: con el valor por defecto, UNA fila con una columna de mas
	// devuelve cero filas y un error, y el fichero entero se pierde. Aqui una
	// fila desigual es una fila ilegible, que es un cubo.
	lector.FieldsPerRecord = -1
	lector.ReuseRecord = false

	cabecera, err := lector.Read()
	if err != nil {
		return Instantanea{}, fmt.Errorf("%w: la primera linea no se puede leer como cabecera: %v.\n"+
			"  Arreglo: abrir el fichero y comprobar que la primera linea son los nombres de las "+
			"columnas, separados por %s", ErrSinColumnaDeCuenta, err, ins.Notas.Separador)
	}
	ins.Notas.Cabeceras = append([]string{}, cabecera...)

	papeles, ignoradas, err := mapearColumnas(cabecera, o.Columnas)
	if err != nil {
		return Instantanea{}, err
	}
	ins.Notas.ColumnasIgnoradas = ignoradas

	vistas := map[string]int{}
	// EL NUMERO DE LINEA LO DA EL LECTOR, no un contador propio.
	//
	// Un `linea++` por vuelta parece equivalente y no lo es: el lector SE SALTA
	// las lineas en blanco sin decir nada (medido), asi que a partir de la
	// primera el contador propio va corrido y todas las lineas que se guardan
	// apuntan al sitio equivocado. Y apuntan mal justo donde importa: en la fila
	// ilegible que alguien tiene que ir a arreglar y en la duplicada que dice
	// "esta ya salio en la linea N".
	ultima := 1
	for {
		registro, err := lector.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			desde, hasta := span(err, ultima+1)
			ins.Ilegibles = append(ins.Ilegibles, Ilegible{
				Desde: desde, Hasta: hasta, Motivo: motivoLegible(err),
			})
			ultima = hasta
			continue
		}
		linea := ultima + 1
		if len(registro) > 0 {
			linea, _ = lector.FieldPos(0)
		}
		ultima = linea
		f, motivo := aFila(registro, papeles, o.Sistema, linea)
		if motivo != "" {
			ins.Ilegibles = append(ins.Ilegibles, Ilegible{Desde: linea, Hasta: linea, Motivo: motivo})
			continue
		}
		if primera, ya := vistas[f.Clave()]; ya {
			ins.Duplicadas = append(ins.Duplicadas, Duplicada{
				Clave: f.Clave(), Linea: linea, Primera: primera,
			})
			continue
		}
		vistas[f.Clave()] = linea
		ins.Filas = append(ins.Filas, f)
		if avisa := avisoDeListaEnUnCampo(f.Permiso); avisa != "" {
			ins.Notas.Avisos = append(ins.Notas.Avisos, fmt.Sprintf("linea %d: %s", linea, avisa))
		}
	}

	// LA GUARDA. Si esto no cuadra, alguien ha desaparecido del censo.
	if !ins.Cuadra() {
		return Instantanea{}, fmt.Errorf("%w: el fichero tiene %d lineas de datos y los cubos solo "+
			"explican %d (%d legibles + %d ilegibles que cubren su rango + %d duplicadas).\n"+
			"  %s sin cubo, y eso significa que hay cuentas del fichero que NO estarian en la "+
			"revision sin que nada lo dijera.\n"+
			"  Dos causas, y las dos se arreglan en el fichero:\n"+
			"    - una comilla sin cerrar, que hace que el lector se trague las lineas siguientes\n"+
			"    - un campo con un salto de linea DENTRO, que es CSV legitimo pero aqui no se\n"+
			"      admite, porque distinguirlo del caso anterior exigiria fiarse del mismo lector\n"+
			"      del que esta comprobacion desconfia\n"+
			"  La instantanea NO se ha creado: revisar sobre un censo incompleto seria certificar "+
			"de mas, y eso lo firma una persona",
			ErrDescuadre, ins.LineasDeDatos, ins.LineasCubiertas(),
			len(ins.Filas), len(ins.Ilegibles), len(ins.Duplicadas),
			plural(ins.LineasDeDatos-ins.LineasCubiertas(), "Queda 1 linea", "Quedan %d lineas"))
	}
	return ins, nil
}

func (o Opciones) validar() error {
	var faltan []string
	if strings.TrimSpace(o.Sistema) == "" {
		faltan = append(faltan, "Sistema (de que sistema son estas cuentas; el fichero no lo sabe)")
	}
	if strings.TrimSpace(o.Quien) == "" {
		faltan = append(faltan, "Quien (identificador de quien sube el fichero)")
	}
	if o.Tomada.IsZero() {
		faltan = append(faltan, "Tomada (el instante; en nucleo no se lee el reloj, entra como dato)")
	}
	if strings.TrimSpace(o.Retencion) == "" {
		faltan = append(faltan, "Retencion (cuanto se guarda y por que: es dato personal)")
	}
	if len(o.Columnas.Cuenta) == 0 {
		faltan = append(faltan, "Columnas.Cuenta (los nombres que valen como identificador; "+
			"ColumnasHabituales() trae los que se ven en la practica)")
	}
	if len(faltan) == 0 {
		return nil
	}
	return fmt.Errorf("%w: %s.\n"+
		"  El valor cero de Opciones esta prohibido a proposito: una lista de cuentas sin quien la "+
		"subio, sin cuando y sin de que sistema no se puede atar a nada y no certifica nada",
		ErrOpciones, strings.Join(faltan, "; "))
}
