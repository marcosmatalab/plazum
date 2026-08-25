// Package diagnostico implementa puertos.Diagnostico: es `dutiq doctor`.
//
// Que responde y a quien. A un operador que ha arrancado el binario un martes
// por la manana, no tiene a quien preguntar y ve algo que no funciona. La
// pregunta que se hace es "por que no funciona", y la respuesta no puede ser
// leer el codigo fuente. Por eso cada comprobacion de aqui dice tres cosas: que
// se esperaba, que se encontro y COMO SE ARREGLA. La tercera es la que lo
// convierte en una herramienta en vez de en un informe.
//
// Regla que sostiene el paquete: aqui NO se comprueba nada que no pueda fallar
// de verdad en una maquina real. Una comprobacion que siempre sale verde no da
// informacion, da falsa tranquilidad, y ademas engorda la pantalla justo cuando
// el operador esta buscando la linea que importa.
//
// El reloj entra como DATO, igual que en el nucleo. No porque este paquete lo
// prohiba (esta fuera de nucleo/ y podria llamar a time.Now), sino porque la
// mitad de lo que se comprueba aqui depende del instante y un diagnostico que
// no se puede fijar en el tiempo no se puede probar: el test de la raiz
// caducada exige poder situarse en 2030.
package diagnostico

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"dutiq/adaptadores/tsa"
	"dutiq/nucleo/corpus"
	"dutiq/puertos"
)

// FechaDeReferencia es el suelo del reloj: el instante en que se construyo esta
// version del diagnostico.
//
// Sirve para lo unico que se puede detectar sin red: un reloj que va MUY
// atrasado. Si la maquina cree estar antes de que este binario existiera, el
// reloj miente, y con el reloj mintiendo todos los plazos legales salen mal y
// ningun sello de tiempo verifica. Es una comprobacion barata que ataja una
// familia entera de incidencias raras.
//
// No detecta un reloj adelantado unos minutos, y no pretende: para eso hace
// falta NTP, o sea red, y `doctor` tiene que funcionar sin ella.
var FechaDeReferencia = time.Date(2026, 8, 25, 0, 0, 0, 0, time.UTC)

// MargenDeAviso es cuanto antes de caducar se avisa de una raiz de TSA.
// Medio ano es el tiempo minimo razonable para conseguir, revisar y publicar un
// certificado nuevo sin prisas.
const MargenDeAviso = 180 * 24 * time.Hour

// AnosDeVidaUtil es cuanto se considera que un binario puede seguir siendo el
// ultimo antes de que "el reloj esta adelantado" sea mas probable que "el
// binario es viejo".
const AnosDeVidaUtil = 10

// RaizConocida es una raiz de TSA que trae el binario, con su caducidad.
//
// POR QUE ESTA DECLARADA Y NO LEIDA DEL POOL, que es lo primero que uno
// pregunta: x509.CertPool no expone los certificados que contiene. Subjects()
// esta obsoleto y ademas solo devuelve el sujeto en DER, sin fechas, asi que no
// hay forma de sacarle un NotAfter. Las alternativas eran tocar
// adaptadores/tsa para que exportara los certificados parseados, que es de otro
// frente, o declarar aqui lo que ya esta declarado en
// adaptadores/tsa/raices/LEEME.md. Se ha hecho lo segundo, y queda como P1 en
// docs/pendientes.md: la tabla puede envejecer sin que nadie se entere.
//
// Lo que SI se comprueba de verdad, sin tabla, son las raices que aporta el
// operador: esas llegan en PEM y se les lee el NotAfter.
type RaizConocida struct {
	Nombre string
	Caduca time.Time
}

// raicesDelBinario son las dos raices embebidas. Espejo de
// adaptadores/tsa/raices/LEEME.md.
var raicesDelBinario = []RaizConocida{
	{Nombre: "Certum Trusted Network CA 2", Caduca: time.Date(2029, 9, 17, 0, 0, 0, 0, time.UTC)},
	{Nombre: "www.freetsa.org", Caduca: time.Date(2041, 3, 7, 0, 0, 0, 0, time.UTC)},
}

// Opciones es todo lo que el diagnostico necesita saber de la instalacion.
// Va como estructura y no como argumentos sueltos porque casi todo tiene un
// valor por defecto razonable y quien llama solo quiere cambiar uno.
type Opciones struct {
	// Ahora es el instante desde el que se juzga lo que depende del tiempo.
	// Cero significa "el reloj del sistema", y lo resuelve Nuevo, no cada
	// comprobacion: asi todas las comprobaciones de una misma pasada ven el
	// mismo instante y el informe es coherente consigo mismo.
	Ahora time.Time
	// Datos es el directorio donde la instalacion escribe. Vacio: el actual.
	Datos string
	// Corpus es el directorio de paquetes instalados. Vacio: Datos/paquetes.
	Corpus string
	// Direccion es donde va a escuchar el servidor. Vacio: 127.0.0.1:8080.
	Direccion string
	// Keystore es la ruta del fichero de claves. Vacio: Datos/keystore.json.
	Keystore string
	// RaicesTSA en PEM, si el operador declara las suyas. Vacio: se juzgan
	// las que trae el binario contra la tabla declarada.
	RaicesTSA []byte
}

// DireccionPorDefecto es donde escucha `dutiq serve` mientras no se configure
// otra cosa. Se declara aqui para que doctor compruebe el mismo puerto que se
// va a usar y no uno parecido.
const DireccionPorDefecto = "127.0.0.1:8080"

// Doctor implementa puertos.Diagnostico.
type Doctor struct {
	o Opciones
}

// Nuevo construye el diagnostico, rellenando los valores por defecto UNA VEZ.
func Nuevo(o Opciones) *Doctor {
	if o.Ahora.IsZero() {
		o.Ahora = time.Now().UTC()
	}
	if o.Datos == "" {
		o.Datos = "."
	}
	if o.Corpus == "" {
		o.Corpus = filepath.Join(o.Datos, "paquetes")
	}
	if o.Direccion == "" {
		o.Direccion = DireccionPorDefecto
	}
	if o.Keystore == "" {
		o.Keystore = filepath.Join(o.Datos, "keystore.json")
	}
	return &Doctor{o: o}
}

var _ puertos.Diagnostico = (*Doctor)(nil)

// Comprobar ejecuta todas las comprobaciones. No falla aunque haya problemas:
// los problemas son el resultado.
//
// El orden NO es alfabetico, es de causa a efecto: el reloj primero porque un
// reloj que miente hace fallar a todo lo demas por razones que no son la suya,
// y quien lea el informe de arriba abajo tiene que encontrar la causa antes que
// los sintomas.
func (d *Doctor) Comprobar(ctx context.Context) []puertos.Comprobacion {
	comprobaciones := []func(context.Context) puertos.Comprobacion{
		d.reloj, d.escritura, d.corpus, d.keystore, d.raicesTSA, d.anclaje, d.puerto,
	}
	out := make([]puertos.Comprobacion, 0, len(comprobaciones))
	for _, f := range comprobaciones {
		if err := ctx.Err(); err != nil {
			out = append(out, puertos.Comprobacion{
				Nombre: "diagnostico", Estado: puertos.Aviso,
				Detalle: "el diagnostico se corto antes de terminar: " + err.Error(),
				Arreglo: "vuelve a lanzar `dutiq doctor` sin interrumpirlo; si tarda demasiado, " +
					"es que alguna comprobacion esta esperando a disco o a red",
			})
			break
		}
		out = append(out, f(ctx))
	}
	return out
}

// ---------------------------------------------------------------------------
// Las comprobaciones
// ---------------------------------------------------------------------------

// reloj: el sistema tiene que creer estar despues de que este binario existiera.
func (d *Doctor) reloj(context.Context) puertos.Comprobacion {
	c := puertos.Comprobacion{Nombre: "reloj"}
	ahora := d.o.Ahora
	arreglo := "sincroniza el reloj y vuelve a probar. En Linux con systemd, " +
		"`sudo timedatectl set-ntp true`; en Windows, `w32tm /resync`; en macOS, " +
		"`sudo sntp -sS time.apple.com`"

	if ahora.Before(FechaDeReferencia) {
		c.Estado = puertos.Roto
		c.Detalle = fmt.Sprintf("el sistema dice que son las %s, anterior a la fecha en que se "+
			"construyo este binario (%s): el reloj esta atrasado",
			ahora.Format(time.RFC3339), FechaDeReferencia.Format("2006-01-02"))
		c.Arreglo = arreglo + ". Con el reloj atrasado los plazos salen mal y ningun sello de " +
			"tiempo verifica, asi que esto invalida cualquier otra cosa que veas"
		return c
	}
	// Segunda senal, y esta si es local de verdad: si un fichero nuestro se
	// modifico DESPUES de lo que el sistema cree que es ahora, el reloj se ha
	// movido hacia atras desde la ultima ejecucion.
	if fi, err := os.Stat(d.o.Datos); err == nil && fi.ModTime().After(ahora.Add(time.Minute)) {
		c.Estado = puertos.Roto
		c.Detalle = fmt.Sprintf("el directorio de datos %s se modifico el %s, DESPUES de lo que "+
			"el sistema cree que es ahora (%s): el reloj se ha movido hacia atras",
			d.o.Datos, fi.ModTime().UTC().Format(time.RFC3339), ahora.Format(time.RFC3339))
		c.Arreglo = arreglo
		return c
	}
	if ahora.After(FechaDeReferencia.AddDate(AnosDeVidaUtil, 0, 0)) {
		c.Estado = puertos.Aviso
		c.Detalle = fmt.Sprintf("el sistema dice que son las %s, mas de %d anos despues de la "+
			"construccion de este binario (%s)", ahora.Format(time.RFC3339), AnosDeVidaUtil,
			FechaDeReferencia.Format("2006-01-02"))
		c.Arreglo = "solo puede ser una de dos, y desde aqui no se distinguen sin red: o el reloj " +
			"esta muy adelantado, y entonces " + arreglo + "; o este binario tiene mas de " +
			fmt.Sprintf("%d", AnosDeVidaUtil) + " anos, y entonces actualizalo con `dutiq update`"
		return c
	}
	c.Estado = puertos.Correcto
	c.Detalle = fmt.Sprintf("el sistema dice %s, coherente con la fecha de este binario (%s)",
		ahora.Format(time.RFC3339), FechaDeReferencia.Format("2006-01-02"))
	return c
}

// tamanoDeLaPrueba es lo que se escribe para comprobar el disco. 64 KiB es
// suficiente para que un disco lleno o una cuota agotada fallen, y poco como
// para no molestar.
const tamanoDeLaPrueba = 64 * 1024

// escritura: se escribe de verdad, se relee y se compara. Comprobar solo que el
// directorio existe deja pasar el caso que mas duele, que es el sistema de
// ficheros montado de solo lectura o la cuota agotada.
func (d *Doctor) escritura(context.Context) puertos.Comprobacion {
	c := puertos.Comprobacion{Nombre: "escritura"}
	arreglo := fmt.Sprintf("comprueba que %s existe, que el usuario que ejecuta dutiq puede "+
		"escribir en el y que queda espacio en disco. Si el directorio es de otro usuario, "+
		"`chown` o arranca dutiq con --datos apuntando a uno propio", d.o.Datos)

	if err := os.MkdirAll(d.o.Datos, 0o750); err != nil {
		c.Estado = puertos.Roto
		c.Detalle = fmt.Sprintf("no puedo crear el directorio de datos %s: %v", d.o.Datos, err)
		c.Arreglo = arreglo
		return c
	}
	f, err := os.CreateTemp(d.o.Datos, ".dutiq-doctor-*")
	if err != nil {
		c.Estado = puertos.Roto
		c.Detalle = fmt.Sprintf("no puedo crear un fichero en %s: %v", d.o.Datos, err)
		c.Arreglo = arreglo
		return c
	}
	ruta := f.Name()
	defer func() { _ = os.Remove(ruta) }()

	datos := make([]byte, tamanoDeLaPrueba)
	for i := range datos {
		datos[i] = byte(i)
	}
	if _, err := f.Write(datos); err != nil {
		_ = f.Close()
		c.Estado = puertos.Roto
		c.Detalle = fmt.Sprintf("no puedo escribir %d bytes en %s: %v", len(datos), d.o.Datos, err)
		c.Arreglo = arreglo
		return c
	}
	// Sync a proposito: sin el, un disco lleno no se entera hasta el cierre y
	// el error se pierde. El ledger escribe con la misma disciplina.
	if err := f.Sync(); err != nil {
		_ = f.Close()
		c.Estado = puertos.Roto
		c.Detalle = fmt.Sprintf("los datos no llegan al disco en %s: %v", d.o.Datos, err)
		c.Arreglo = arreglo
		return c
	}
	if err := f.Close(); err != nil {
		c.Estado = puertos.Roto
		c.Detalle = fmt.Sprintf("no puedo cerrar el fichero de prueba en %s: %v", d.o.Datos, err)
		c.Arreglo = arreglo
		return c
	}
	leido, err := os.ReadFile(ruta) // #nosec G304 -- la ruta la acaba de crear CreateTemp aqui mismo
	if err != nil {
		c.Estado = puertos.Roto
		c.Detalle = fmt.Sprintf("escribo en %s pero no puedo releer lo escrito: %v", d.o.Datos, err)
		c.Arreglo = arreglo
		return c
	}
	if len(leido) != len(datos) {
		c.Estado = puertos.Roto
		c.Detalle = fmt.Sprintf("escribi %d bytes en %s y al releer hay %d: el disco esta lleno "+
			"o la cuota agotada", len(datos), d.o.Datos, len(leido))
		c.Arreglo = arreglo
		return c
	}
	c.Estado = puertos.Correcto
	c.Detalle = fmt.Sprintf("%s admite escritura y relectura de %d bytes", d.o.Datos, len(datos))
	return c
}

// corpus: el corpus instalado tiene que CARGAR, no solo estar.
func (d *Doctor) corpus(context.Context) puertos.Comprobacion {
	c := puertos.Comprobacion{Nombre: "corpus"}
	if _, err := os.Stat(d.o.Corpus); err != nil {
		c.Estado = puertos.Aviso
		c.Detalle = fmt.Sprintf("no hay corpus instalado en %s: sin paquetes no hay obligaciones "+
			"que vigilar y las pantallas salen vacias", d.o.Corpus)
		c.Arreglo = "ejecuta `dutiq demo` para ver el producto lleno con una empresa de ejemplo, " +
			"o copia los paquetes que quieras vigilar a " + d.o.Corpus
		return c
	}
	ps, err := corpus.Cargar(d.o.Corpus)
	if err != nil {
		c.Estado = puertos.Roto
		c.Detalle = fmt.Sprintf("el corpus de %s no carga: %v", d.o.Corpus, err)
		c.Arreglo = "el mensaje de arriba dice que paquete y que regla concreta lo rechaza. " +
			"Un paquete que no pasa el linter no se ejecuta a ver que pasa, se rechaza: " +
			"quita ese paquete de " + d.o.Corpus + " o corrigelo y vuelve a probar"
		return c
	}
	if len(ps) == 0 {
		c.Estado = puertos.Aviso
		c.Detalle = fmt.Sprintf("%s existe pero no tiene ningun paquete dentro", d.o.Corpus)
		c.Arreglo = "un paquete es un directorio con paquete.json dentro. Ejecuta `dutiq demo` " +
			"para instalar uno de ejemplo y ver la forma que tiene"
		return c
	}
	obligaciones, dorados := 0, 0
	for _, p := range ps {
		obligaciones += len(p.Obligaciones)
		dorados += len(p.Dorados)
	}
	c.Estado = puertos.Correcto
	c.Detalle = fmt.Sprintf("%d paquete(s) cargados desde %s, %d obligaciones y %d casos dorados",
		len(ps), d.o.Corpus, obligaciones, dorados)
	return c
}

// keystore: la clave que cifra el ledger. Que no exista todavia no es un fallo;
// que este y no se pueda leer, si.
func (d *Doctor) keystore(context.Context) puertos.Comprobacion {
	c := puertos.Comprobacion{Nombre: "keystore"}
	fi, err := os.Stat(d.o.Keystore)
	if errors.Is(err, os.ErrNotExist) {
		c.Estado = puertos.Aviso
		c.Detalle = fmt.Sprintf("todavia no hay keystore en %s", d.o.Keystore)
		c.Arreglo = "es normal en una instalacion recien hecha: el keystore se crea al primer " +
			"arranque de `dutiq serve`. Si esperabas que estuviera, comprueba que --datos apunta " +
			"al directorio de la instalacion que crees y no a uno nuevo"
		return c
	}
	if err != nil {
		c.Estado = puertos.Roto
		c.Detalle = fmt.Sprintf("no puedo mirar el keystore %s: %v", d.o.Keystore, err)
		c.Arreglo = "comprueba permisos del directorio y del fichero. Sin keystore legible no se " +
			"puede descifrar ni una entrada del ledger, o sea que el expediente no se puede emitir"
		return c
	}
	if fi.IsDir() {
		c.Estado = puertos.Roto
		c.Detalle = fmt.Sprintf("%s es un directorio y tenia que ser el fichero de claves", d.o.Keystore)
		c.Arreglo = "apunta --keystore al fichero de claves, o retira ese directorio si lo creo " +
			"algo por error"
		return c
	}
	f, err := os.Open(d.o.Keystore) // #nosec G304 -- la ruta la fija el operador en su propia maquina
	if err != nil {
		c.Estado = puertos.Roto
		c.Detalle = fmt.Sprintf("el keystore %s existe y no se puede abrir: %v", d.o.Keystore, err)
		c.Arreglo = fmt.Sprintf("dale permiso de lectura al usuario que ejecuta dutiq: "+
			"`chown $(id -u):$(id -g) %s && chmod 600 %s`", d.o.Keystore, d.o.Keystore)
		return c
	}
	_ = f.Close()

	// Los permisos solo se juzgan donde significan algo. En Windows el modo que
	// devuelve os.Stat es sintetico y decir "esta abierto a todo el mundo"
	// sobre una ACL que no se ha mirado seria mentir.
	if runtime.GOOS != "windows" && fi.Mode().Perm()&0o077 != 0 {
		c.Estado = puertos.Aviso
		c.Detalle = fmt.Sprintf("el keystore %s tiene permisos %04o: lo puede leer alguien mas "+
			"que su dueno", d.o.Keystore, fi.Mode().Perm())
		c.Arreglo = fmt.Sprintf("`chmod 600 %s`. La clave del keystore descifra el ledger entero: "+
			"si la lee otro usuario de la maquina, el cifrado no esta protegiendo de nada",
			d.o.Keystore)
		return c
	}
	c.Estado = puertos.Correcto
	c.Detalle = fmt.Sprintf("%s legible, %d bytes", d.o.Keystore, fi.Size())
	return c
}

// raicesTSA: sin raices no se verifica ningun sello, y una raiz caducada deja de
// firmar sellos nuevos sin que nada mas se entere.
func (d *Doctor) raicesTSA(context.Context) puertos.Comprobacion {
	c := puertos.Comprobacion{Nombre: "raices-tsa"}
	if len(d.o.RaicesTSA) > 0 {
		return d.raicesDelOperador()
	}
	if _, err := tsa.RaicesPorDefecto(); err != nil {
		c.Estado = puertos.Roto
		c.Detalle = "el binario no puede cargar sus raices de TSA embebidas: " + err.Error()
		c.Arreglo = "es un fallo de la propia compilacion del binario, no de tu instalacion: " +
			"vuelve a descargarlo de la release oficial y comprueba el SHA256SUMS. Mientras " +
			"tanto, `dutiq verify` avisara de que no puede comprobar los sellos en vez de darlos " +
			"por buenos"
		return c
	}
	var caducadas, porCaducar []string
	for _, r := range raicesDelBinario {
		switch {
		case !d.o.Ahora.Before(r.Caduca):
			caducadas = append(caducadas, fmt.Sprintf("%s (caduco el %s)", r.Nombre,
				r.Caduca.Format("2006-01-02")))
		case d.o.Ahora.Add(MargenDeAviso).After(r.Caduca):
			porCaducar = append(porCaducar, fmt.Sprintf("%s (caduca el %s)", r.Nombre,
				r.Caduca.Format("2006-01-02")))
		}
	}
	switch {
	case len(caducadas) == len(raicesDelBinario):
		c.Estado = puertos.Roto
		c.Detalle = fmt.Sprintf("todas las raices de TSA que trae el binario han caducado: %v",
			caducadas)
		c.Arreglo = "actualiza dutiq con `dutiq update`, que trae raices nuevas. Si no puedes " +
			"actualizar, declara tus propias raices en raices_tsa dentro del fichero de contexto " +
			"del receptor. Los sellos YA emitidos siguen verificando, porque se comprueban contra " +
			"el instante del sello y no contra hoy"
	case len(caducadas) > 0:
		c.Estado = puertos.Aviso
		c.Detalle = fmt.Sprintf("hay raices de TSA caducadas: %v; quedan otras validas", caducadas)
		c.Arreglo = "actualiza dutiq con `dutiq update` para recuperar la cadena de reserva. " +
			"Con una sola raiz viva, el dia que esa TSA falle todo anclaje se queda en la cola"
	case len(porCaducar) > 0:
		c.Estado = puertos.Aviso
		c.Detalle = fmt.Sprintf("hay raices de TSA a punto de caducar: %v", porCaducar)
		c.Arreglo = "actualiza dutiq antes de esa fecha con `dutiq update`. No corre prisa hoy, " +
			"pero el dia que caduque los sellos NUEVOS dejaran de verificar mientras la TSA " +
			"sigue respondiendo 200, que es la forma mas silenciosa de romperse"
	default:
		c.Estado = puertos.Correcto
		c.Detalle = fmt.Sprintf("%d raices de TSA embebidas y ninguna caduca en los proximos %d dias",
			len(raicesDelBinario), int(MargenDeAviso.Hours()/24))
	}
	return c
}

// raicesDelOperador juzga las raices que aporta el operador. Estas SI se leen de
// verdad: llegan en PEM y tienen su NotAfter dentro.
func (d *Doctor) raicesDelOperador() puertos.Comprobacion {
	c := puertos.Comprobacion{Nombre: "raices-tsa"}
	resto := d.o.RaicesTSA
	var vivas, caducadas, porCaducar []string
	for {
		var bloque *pem.Block
		bloque, resto = pem.Decode(resto)
		if bloque == nil {
			break
		}
		cert, err := x509.ParseCertificate(bloque.Bytes)
		if err != nil {
			c.Estado = puertos.Roto
			c.Detalle = "las raices de TSA que declaras traen un certificado ilegible: " + err.Error()
			c.Arreglo = "revisa el bloque raices_tsa de tu fichero de contexto: tiene que ser uno o " +
				"mas certificados en PEM, empezando por -----BEGIN CERTIFICATE-----"
			return c
		}
		nombre := cert.Subject.CommonName
		if nombre == "" {
			nombre = cert.Subject.String()
		}
		switch {
		case !d.o.Ahora.Before(cert.NotAfter):
			caducadas = append(caducadas, fmt.Sprintf("%s (caduco el %s)", nombre,
				cert.NotAfter.Format("2006-01-02")))
		case d.o.Ahora.Add(MargenDeAviso).After(cert.NotAfter):
			porCaducar = append(porCaducar, fmt.Sprintf("%s (caduca el %s)", nombre,
				cert.NotAfter.Format("2006-01-02")))
			vivas = append(vivas, nombre)
		default:
			vivas = append(vivas, nombre)
		}
	}
	switch {
	case len(vivas) == 0 && len(caducadas) == 0:
		c.Estado = puertos.Roto
		c.Detalle = "declaras raices_tsa y no contiene ningun certificado PEM"
		c.Arreglo = "quita raices_tsa del fichero de contexto si quieres usar las que trae el " +
			"binario, o pega ahi los certificados de las TSAs que aceptas"
	case len(vivas) == 0:
		c.Estado = puertos.Roto
		c.Detalle = fmt.Sprintf("todas las raices de TSA que declaras han caducado: %v", caducadas)
		c.Arreglo = "sustituyelas por las vigentes de tu TSA en raices_tsa. Cuando declaras raices " +
			"propias valen SOLO esas, las del binario no se suman"
	case len(caducadas) > 0:
		c.Estado = puertos.Aviso
		c.Detalle = fmt.Sprintf("de las raices de TSA que declaras, estas han caducado: %v", caducadas)
		c.Arreglo = "sustituyelas en raices_tsa. Los sellos ya emitidos con ellas siguen " +
			"verificando, porque se comprueban contra el instante del sello"
	case len(porCaducar) > 0:
		c.Estado = puertos.Aviso
		c.Detalle = fmt.Sprintf("de las raices de TSA que declaras, estas caducan pronto: %v", porCaducar)
		c.Arreglo = "pide el certificado nuevo a tu TSA y sustituyelo en raices_tsa antes de esa fecha"
	default:
		c.Estado = puertos.Correcto
		c.Detalle = fmt.Sprintf("%d raiz(ces) de TSA declaradas por ti y vigentes: %v",
			len(vivas), vivas)
	}
	return c
}

// anclaje: la configuracion de la cadena de sellado. No comprueba la red, y por
// eso vale para un doctor que tiene que funcionar sin ella: comprueba lo que se
// pierde con cada hueco de configuracion.
func (d *Doctor) anclaje(context.Context) puertos.Comprobacion {
	c := puertos.Comprobacion{Nombre: "anclaje"}
	cadena, err := tsa.PorDefecto()
	if err != nil {
		c.Estado = puertos.Roto
		c.Detalle = "no puedo construir la cadena de sellado por defecto: " + err.Error()
		c.Arreglo = "vuelve a descargar el binario de la release oficial: sin cadena de sellado los " +
			"checkpoints salen sin anclaje, y un checkpoint sin anclaje no verifica"
		return c
	}
	cadena.Cola = tsa.NuevaCola(filepath.Join(d.o.Datos, "cola-tsa"))
	avisos := cadena.Revisar()
	if len(avisos) > 0 {
		c.Estado = puertos.Aviso
		c.Detalle = fmt.Sprintf("la cadena de sellado tiene %d hueco(s): %v", len(avisos), avisos)
		c.Arreglo = "cada aviso dice que se pierde. La cola local se crea sola al primer sellado; " +
			"si falta una TSA de reserva, anadela en la configuracion de anclaje antes de emitir " +
			"expedientes que alguien vaya a auditar"
		return c
	}
	pendientes, err := cadena.Cola.Pendientes()
	if err != nil {
		c.Estado = puertos.Aviso
		c.Detalle = "no puedo leer la cola de anclajes pendientes: " + err.Error()
		c.Arreglo = fmt.Sprintf("comprueba permisos de %s. Con la cola ilegible, un sello que "+
			"falle se pierde en vez de reintentarse", filepath.Join(d.o.Datos, "cola-tsa"))
		return c
	}
	if len(pendientes) > 0 {
		c.Estado = puertos.Aviso
		c.Detalle = fmt.Sprintf("hay %d anclaje(s) en la cola esperando a que responda una TSA",
			len(pendientes))
		c.Arreglo = "comprueba la salida a internet hacia las TSAs configuradas. Mientras esten en " +
			"la cola, esos checkpoints no tienen fecha demostrable ante un tercero"
		return c
	}
	c.Estado = puertos.Correcto
	c.Detalle = fmt.Sprintf("%d TSAs configuradas con cola local y sin anclajes pendientes",
		len(cadena.Autoridades))
	return c
}

// puerto: se intenta escuchar de verdad. Mirar una lista de procesos no vale:
// el caso que interesa es "el puerto esta ocupado O el sistema no me deja
// abrirlo", y las dos cosas se contestan igual, intentandolo.
func (d *Doctor) puerto(context.Context) puertos.Comprobacion {
	c := puertos.Comprobacion{Nombre: "puerto"}
	l, err := net.Listen("tcp", d.o.Direccion)
	if err != nil {
		c.Estado = puertos.Roto
		c.Detalle = fmt.Sprintf("no puedo escuchar en %s: %v", d.o.Direccion, err)
		c.Arreglo = fmt.Sprintf("o el puerto lo tiene otro proceso, y entonces `ss -ltnp | grep %s` "+
			"(o `netstat -ano | findstr %s` en Windows) dice cual y lo paras o cambias --direccion; "+
			"o el puerto es menor de 1024 y el sistema no te deja abrirlo sin privilegios, y "+
			"entonces usa uno alto y termina TLS en el proxy", d.o.Direccion, d.o.Direccion)
		return c
	}
	_ = l.Close()
	c.Estado = puertos.Correcto
	c.Detalle = fmt.Sprintf("%s esta libre y se puede escuchar en el", d.o.Direccion)
	return c
}
