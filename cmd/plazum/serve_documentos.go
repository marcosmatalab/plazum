package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"

	"github.com/marcosmatalab/plazum/adaptadores/busqueda"
	"github.com/marcosmatalab/plazum/adaptadores/evidencia"
	"github.com/marcosmatalab/plazum/adaptadores/ia"
	"github.com/marcosmatalab/plazum/adaptadores/ingesta"
	"github.com/marcosmatalab/plazum/adaptadores/usuarios"
	"github.com/marcosmatalab/plazum/nucleo/corpus"
	"github.com/marcosmatalab/plazum/puertos"
	"github.com/marcosmatalab/plazum/superficies/camino"
	"github.com/marcosmatalab/plazum/superficies/documentos"
)

// EL CABLE DE LA PIEZA 3, de un fichero subido a un hallazgo con su cita.
//
// # Por que vive aqui
//
// Porque es el unico sitio que conoce a los cinco: `ingesta` lee el fichero,
// `ia` lo convierte en fuentes citables con hash, `busqueda` lo indexa,
// `evidencia` empareja y `corpus` dice contra que. `superficies/documentos` no
// puede importar ninguno de ellos: su contrato es un puerto y nada mas. Es la
// misma forma que `consecuenciasDeLaEntrevista` y que `alcanceDeLaInstalacion`,
// y la doctrina esta escrita en serve_evidencia.go.
//
// # LA CADENA, ENTERA Y EN ORDEN
//
//	ingesta.Leer(nombre, datos)        -> Documento con sus fragmentos y paginas
//	ia.Huella(datos)                   -> sha256 del fichero, que identifica el
//	                                      documento: subirlo dos veces no duplica
//	ia.FuentesAportadas(huella, doc)   -> una Fuente por fragmento, con su hash
//	ia.Documentos(fuentes)             -> los documentos del indice
//	busqueda.Nuevo(docs)               -> el indice invertido, BM25
//	ia.Nuevo(... Admite: Aportado)     -> el verificador de citas
//	evidencia.Mapear(idx, ver, cs)     -> los hallazgos, ya verificados por hash
//
// # NO HAY MODELO EN NINGUN ESLABON, y eso es la decision y no una limitacion
//
// La pieza 3 funciona con PLAZUM_SIN_IA=1 porque no llama a ninguno: la
// busqueda es BM25 con parametros fijos y la verificacion es un hash. El
// encabezado de `adaptadores/evidencia` lo razona entero. Lo que compra es que
// el resultado sea RECONTABLE, que en una pantalla que le dice a alguien donde
// mirar en su propia politica es la unica propiedad que cuenta.
//
// # EL VERIFICADOR ES POR CUENTA, Y NO SE CONSTRUYE CON ia.Estricto
//
// `ia.Estricto` admite SOLO el corpus firmado, a proposito: es la configuracion
// segura y es la corta de escribir. Aqui hace falta la otra, `ia.Nuevo` con
// `Admite: [Aportado]`, y son mas letras tambien a proposito. Un verificador que
// admite documentos del cliente construido por descuido dejaria salir una cita
// de una politica por una pantalla que dice citar la ley.
//
// # INVARIANTE 12: EL INDICE ES DE LA CUENTA Y NO DE LA INSTALACION
//
// La pregunta se contesta aqui y no en la cabeza: ¿de quien es este dato? De la
// CUENTA. Un indice por proceso serviria la cita literal de la politica de una
// persona a cualquier otra que entrara, y el verificador no lo impediria porque
// comprueba procedencia y literalidad, no propiedad. Por eso la cuenta va en la
// FIRMA del puerto y no en el constructor: sin ella en la firma, la
// implementacion tendria que sacarla de una variable compartida, que es
// exactamente la fuga.
//
// # QUE PASA AL REINICIAR: SE PIERDE, Y LA PANTALLA LO DICE
//
// El indice vive en memoria del proceso y no se persiste. La decision es de esta
// casilla y no de la siguiente: `ETAPAS.md` pide el mapeo, no el almacen, y
// persistir el documento de un cliente es una frontera de custodia que se abre
// una vez y con su propia casilla (cifrado en reposo, borrado, retencion). Lo
// que NO se hace es callarlo: al reiniciar, la pantalla sale en su estado vacio,
// y el estado vacio dice que hay que subir el documento, no que no lo hayas
// subido nunca.
//
// # LOS TRES TOPES, y el tercero es el que no salta a la vista
//
//	por documento   MaximoDelDocumento, en la superficie, con su aviso
//	por cuenta      MaxBytesIndexadosPorCuenta, aqui abajo
//	de frecuencia   el limitador general de superficies/serve. NO hay uno
//	                propio, y se dice: una cuenta puede pedir muchas
//	                reconstrucciones por minuto y cada una reindexa lo suyo
//	                entero. Con el tope por cuenta puesto, el pico esta acotado;
//	                lo que no esta es el numero de veces.

// MaxBytesIndexadosPorCuenta es cuanto texto puede tener una cuenta indexado a
// la vez.
//
// EL NUMERO SALE DE UNA CUENTA Y NO DEL GUSTO. El presupuesto de RAM de
// `plazum serve` son 256 MiB y hoy se usan 6. Un indice invertido de este
// paquete pesa del orden de tres veces el texto que indexa (postings mas los
// documentos retenidos), asi que 8 MiB de texto por cuenta son del orden de 24
// MiB de indice, y diez cuentas activas a la vez caben. Una politica de
// seguridad corriente son 100 KB: esto son ochenta.
//
// SE MIDE EL TEXTO INDEXADO Y NO EL FICHERO, porque es lo que ocupa: un PDF de
// 4 MiB puede rendir 8 MiB de texto al descomprimir, y contar bytes de fichero
// dejaria el tope diciendo lo que no es.
const MaxBytesIndexadosPorCuenta = 8 << 20

// ErrCuentaLlena: la cuenta ha llegado a su tope de texto indexado.
var ErrCuentaLlena = errors.New("documentos: la cuenta ha llegado a su tope de texto indexado")

// indicesPorCuenta guarda, por cuenta, lo que esa cuenta ha subido.
//
// EL CANDADO PROTEGE EL MAPA, NO EL INDICE. `busqueda.Indice` es inmutable por
// diseno y se puede leer desde muchas peticiones sin candado; lo que si hay que
// sincronizar es SUSTITUIR el puntero, que es lo que hace una subida. Sin esto,
// dos subidas concurrentes de la misma cuenta se pisan y una de las dos
// desaparece sin que nadie lo note.
type indicesPorCuenta struct {
	mu  sync.RWMutex
	por map[string]*loDeUnaCuenta
	// consultas son las del corpus, compuestas UNA VEZ al construir: los
	// paquetes no cambian mientras el proceso vive, y recomponerlas en cada
	// peticion seria recorrer 563 obligaciones para obtener siempre lo mismo.
	consultas []evidencia.Consulta
	// titulos y marcos dicen como se llama cada obligacion, para que la
	// pantalla pueda nombrarla. Salen del corpus y viajan tal cual.
	titulos map[string]string
	marcos  map[string]string
}

// loDeUnaCuenta es todo lo que hay que retener por cuenta.
//
// SE GUARDAN LAS FUENTES Y NO SOLO EL INDICE, y no es redundancia: el indice no
// devuelve sus documentos (`Documentos()` da un entero), asi que sin las fuentes
// no se puede reconstruir al subir el segundo documento. Y el verificador se
// construye de las mismas fuentes, o sea que las dos mitades de la cadena
// dependen de ellas.
type loDeUnaCuenta struct {
	fuentes []ia.Fuente
	idx     *busqueda.Indice
	ver     *ia.Verificador
	docs    []documentos.ResumenDeDocumento
	bytes   int
	// sitios dice, POR IDENTIFICADOR DE FUENTE, en que pagina y en que orden
	// esta ese fragmento dentro de su documento.
	//
	// SE GUARDA AQUI Y NO SE LEE DE LA FUENTE porque `adaptadores/ia` compone su
	// referencia en castellano («pagina 4»), y esta pantalla emite clave y datos:
	// la frase la pone el catalogo. Se indexa por el ID de la fuente, que es
	// identidad y no posicion (invariante 7).
	sitios map[string]sitioDelFragmento
}

// sitioDelFragmento es DONDE esta un fragmento dentro de su documento.
//
// Pagina cero significa «no se sabe» y no «la pagina cero»: en un formato sin
// paginas no hay ninguna, y decir una seria inventarsela.
type sitioDelFragmento struct {
	Pagina int
	Orden  int
}

// nuevosIndicesPorCuenta compone las consultas del corpus una vez.
func nuevosIndicesPorCuenta(ps []*corpus.Paquete) *indicesPorCuenta {
	a := &indicesPorCuenta{
		por:     map[string]*loDeUnaCuenta{},
		titulos: map[string]string{},
		marcos:  map[string]string{},
	}
	for _, p := range ps {
		for _, o := range p.Obligaciones {
			// SOLO LAS QUE TIENEN TEXTO LEGAL. Un paquete referencial no lo
			// trae (invariante 3: identificador y titulo corto como maximo), y
			// buscar contra un titulo de cinco palabras da hallazgos por
			// casualidad. Se salta en vez de rebajar la consulta.
			if o.TextoLegal == "" {
				continue
			}
			a.consultas = append(a.consultas, evidencia.Consulta{ID: o.ID, Texto: o.TextoLegal})
			a.titulos[o.ID] = o.TituloLegible()
			a.marcos[o.ID] = p.URN
		}
	}
	return a
}

// clave normaliza la cuenta con el mismo normalizador que el almacen de
// alcances.
//
// UN SOLO NORMALIZADOR Y NO DOS: si esta superficie normalizara distinto que el
// resto, «Ana@Ejemplo» y «ana@ejemplo» serian la misma cuenta para el alcance y
// dos cuentas para los documentos, y quien subiera con una no veria lo suyo con
// la otra.
// Consultas es cuantas obligaciones del corpus instalado se pueden buscar dentro
// de un documento.
//
// # Por que existe este metodo y no un recuento en la pantalla
//
// Porque la pantalla del alcance publica ese numero al lado del enlace a la
// pieza 3, y recomputarlo alli seria una SEGUNDA implementacion de la misma
// cifra: asi es como se consigue que dos numeros esten de acuerdo entre si y el
// que mande sea un tercero. Lo dice quien las compone, que es este.
//
// ES UN DATO DE LA INSTALACION Y NO DE LA CUENTA (invariante 12): sale de los
// paquetes cargados, es el mismo para todo el mundo y no depende de quien mire,
// asi que puede pintarse en una pantalla que se sirve sin sesion. Cuantos
// documentos tiene subidos alguien SI es de la cuenta, y por eso ese numero no
// sale de aqui: sale de Documentos(ctx, quien), con la cuenta en la firma.
func (a *indicesPorCuenta) Consultas() int { return len(a.consultas) }

func (a *indicesPorCuenta) clave(quien string) (string, error) {
	return usuarios.NormalizarUsuario(quien)
}

// Subir lee el documento, lo indexa para esa cuenta y devuelve su resumen.
//
// LA RECONSTRUCCION VA DENTRO DEL CANDADO Y SOBRE LO QUE YA HABIA. El indice es
// inmutable, asi que anadir un documento es rehacerlo con las fuentes viejas mas
// las nuevas. Hacerlo fuera del candado dejaria que dos subidas leyeran el mismo
// estado de partida y la ultima en escribir se llevara por delante a la otra.
func (a *indicesPorCuenta) Subir(_ context.Context, quien, nombre string, datos []byte) (
	documentos.ResumenDeDocumento, error) {

	cuenta, err := a.clave(quien)
	if err != nil {
		return documentos.ResumenDeDocumento{}, err
	}
	// LA LECTURA VA FUERA DEL CANDADO: es la parte cara (descomprimir un PDF) y
	// no toca estado compartido. Dentro del candado va solo lo que lo toca.
	doc, err := ingesta.Leer(nombre, datos)
	if err != nil {
		return documentos.ResumenDeDocumento{}, err
	}
	huella := ia.Huella(datos)
	nuevas, err := ia.FuentesAportadas(huella, doc)
	if err != nil {
		return documentos.ResumenDeDocumento{}, err
	}
	res := documentos.ResumenDeDocumento{
		Fichero: nombre, Huella: huella, Fragmentos: len(doc.Fragmentos),
		Paginas: doc.Paginas, Truncado: doc.Truncado,
		NombreEnganoso: doc.NombreEnganoso,
	}
	if len(nuevas) == 0 {
		// El documento estaba vacio de verdad. No entra en el indice y no se
		// apunta: un documento sin ni un fragmento no aporta nada y ocuparia
		// una fila en la lista diciendo que hay algo donde no lo hay.
		return res, nil
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	c := a.por[cuenta]
	if c == nil {
		c = &loDeUnaCuenta{}
	}
	// MISMO FICHERO SUBIDO DOS VECES: la huella es del contenido, asi que se
	// reconoce y no se duplica. Se devuelve el resumen igual, porque quien lo
	// subio tiene que ver que ha llegado.
	for _, d := range c.docs {
		if d.Huella == huella {
			return d, nil
		}
	}
	pesa := 0
	for _, f := range nuevas {
		pesa += len(f.Texto)
	}
	if c.bytes+pesa > MaxBytesIndexadosPorCuenta {
		return documentos.ResumenDeDocumento{}, fmt.Errorf("%w: esta cuenta lleva %d bytes de texto "+
			"indexado y este documento anade %d, sobre un tope de %d. Arreglo: el tope es "+
			"por cuenta y por proceso, asi que reiniciar plazum lo vacia; si hace falta mas, "+
			"es una decision de quien monta", ErrCuentaLlena, c.bytes, pesa,
			MaxBytesIndexadosPorCuenta)
	}

	todas := append(append([]ia.Fuente(nil), c.fuentes...), nuevas...)
	idx, err := busqueda.Nuevo(ia.Documentos(todas))
	if err != nil {
		return documentos.ResumenDeDocumento{}, err
	}
	ver, err := ia.Nuevo(ia.Opciones{
		Fuentes: todas,
		// APORTADO Y SOLO APORTADO. Este verificador resuelve citas de
		// documentos del cliente; admitir tambien el corpus aqui dejaria que un
		// hallazgo de esta pantalla citara la ley diciendo que es tu politica.
		Admite:     []ia.Procedencia{ia.Aportado},
		MinimoCita: ia.MinimoCitaPorDefecto,
	})
	if err != nil {
		return documentos.ResumenDeDocumento{}, err
	}
	c.fuentes, c.idx, c.ver = todas, idx, ver
	c.bytes += pesa
	c.docs = append(c.docs, res)
	if c.sitios == nil {
		c.sitios = map[string]sitioDelFragmento{}
	}
	// EL SITIO DE CADA FRAGMENTO, EMPAREJADO POR POSICION Y AQUI SI VALE.
	//
	// `ia.FuentesAportadas` recorre `doc.Fragmentos` en orden y devuelve una
	// fuente por fragmento, asi que los dos slices son la MISMA lista vista dos
	// veces y no dos conjuntos distintos. No es el emparejamiento que el
	// invariante 7 prohibe (dos listas construidas por separado): es una
	// proyeccion, hecha en la misma llamada y sin nada en medio. Lo que si se
	// guarda por IDENTIDAD es el resultado, indexado por el ID de la fuente.
	for i, f := range nuevas {
		if i < len(doc.Fragmentos) {
			c.sitios[f.ID] = sitioDelFragmento{
				Pagina: doc.Fragmentos[i].Pagina, Orden: doc.Fragmentos[i].Orden}
		}
	}
	a.por[cuenta] = c
	return res, nil
}

// Documentos son los que esa cuenta tiene subidos.
func (a *indicesPorCuenta) Documentos(_ context.Context, quien string) (
	[]documentos.ResumenDeDocumento, error) {

	cuenta, err := a.clave(quien)
	if err != nil {
		return nil, err
	}
	a.mu.RLock()
	defer a.mu.RUnlock()
	c := a.por[cuenta]
	if c == nil {
		return nil, nil
	}
	return append([]documentos.ResumenDeDocumento(nil), c.docs...), nil
}

// Hallazgos mapea lo que esa cuenta ha subido contra lo que pide el corpus.
//
// # POR QUE CAMPO CASA CADA EMPAREJAMIENTO (invariante 7)
//
// Hay dos y los dos van por identidad:
//
//	hallazgo -> cita       por HASH, que es el sha256 del propio texto citado.
//	                       Esta DENTRO de lo que se verifica: cambiar el texto
//	                       cambia el hash, asi que no se puede mover una cita a
//	                       otro fragmento sin que la verificacion lo note.
//	hallazgo -> obligacion por ConsultaID, que es el ID de la obligacion del
//	                       corpus. Nunca por el orden de la lista de consultas:
//	                       Mapear se salta las que no dan hallazgo, asi que las
//	                       posiciones de entrada y de salida NO se corresponden,
//	                       y casarlas por indice atribuiria el parrafo a la
//	                       obligacion equivocada.
func (a *indicesPorCuenta) Hallazgos(_ context.Context, quien string) (
	[]documentos.Hallazgo, error) {

	cuenta, err := a.clave(quien)
	if err != nil {
		return nil, err
	}
	a.mu.RLock()
	c := a.por[cuenta]
	consultas := a.consultas
	a.mu.RUnlock()
	if c == nil || c.idx == nil {
		return nil, nil
	}
	if len(consultas) == 0 {
		// SIN CORPUS NO HAY CONTRA QUE MAPEAR, y eso no es un hallazgo vacio:
		// es que no hay preguntas. Se devuelve vacio y la pantalla lo dice con
		// la frase de «no se ha encontrado», que es cierta.
		return nil, nil
	}
	hs, err := evidencia.Mapear(c.idx, c.ver, consultas)
	if err != nil {
		return nil, err
	}
	// EL NOMBRE DEL DOCUMENTO SE BUSCA POR HUELLA, que es el prefijo que
	// `ia.FuentesAportadas` mete en el identificador de cada fuente.
	out := make([]documentos.Hallazgo, 0, len(hs))
	for _, h := range hs {
		sitio := c.sitios[h.Fuente]
		out = append(out, documentos.Hallazgo{
			Obligacion: h.ConsultaID,
			Titulo:     a.titulos[h.ConsultaID],
			Marco:      a.marcos[h.ConsultaID],
			Parrafo:    h.Cita,
			Pagina:     sitio.Pagina,
			Fragmento:  sitio.Orden,
			Documento:  nombreDelDocumento(c.docs, h.Fuente),
		})
	}
	return out, nil
}

// nombreDelDocumento traduce el identificador de una fuente al nombre del
// fichero del que sale.
//
// El identificador de una fuente aportada es «aportado:<16 de la huella>:<orden>»
// (ver ia.FuentesAportadas), asi que se empareja por el PREFIJO DE LA HUELLA, que
// es contenido del fichero y no una posicion.
//
// SI NO SE ENCUENTRA, NO SE INVENTA: se devuelve cadena vacia y la pantalla
// pinta la referencia sin nombre. Atribuir un parrafo al documento equivocado es
// peor que no decir de cual sale.
func nombreDelDocumento(docs []documentos.ResumenDeDocumento, fuenteID string) string {
	for _, d := range docs {
		if len(d.Huella) >= 16 && len(fuenteID) > 9 &&
			fuenteID[:9] == "aportado:" && fuenteID[9:9+16] == d.Huella[:16] {
			return d.Fichero
		}
	}
	return ""
}

var _ documentos.Almacen = (*indicesPorCuenta)(nil)

// ---------------------------------------------------------------------------
// El montaje
// ---------------------------------------------------------------------------

// construirDocumentos arma la superficie.
//
// El tipo del almacen es CONCRETO y no el interfaz, por lo mismo que en
// montajesDelCamino: un puntero nil metido en un interfaz deja de ser nil, asi
// que un `almacen == nil` sobre el interfaz no cazaria el caso que importa y la
// pantalla pintaria un formulario que entra en panico al primer envio.
func construirDocumentos(cat puertos.Catalogo, quien func(*http.Request) string,
	tokens func(*http.Request) (string, error), almacen *indicesPorCuenta,
	alFallar func(error)) (*documentos.Superficie, error) {

	o := documentos.Opciones{
		Catalogo: cat,
		Base:     documentos.BasePorDefecto,
		Estatico: "/estatico",
		Quien:    quien,
		Tokens:   tokens,
		AlFallar: alFallar,
		// La vuelta al camino guiado y la barra lateral. Sin esto, esta
		// pantalla es un callejon: no es un paso, asi que no se llega a ella
		// desde la tira, y sin el enlace de vuelta no se sale.
		CaminoRuta:  camino.BasePorDefecto + "/",
		CaminoClave: camino.ClaveTitulo,
		Pasos:       camino.Canonico(),
	}
	if almacen != nil {
		o.Almacen = almacen
	}
	return documentos.Nuevo(o)
}

// montajesFueraDelCamino son las superficies que se montan y NO son un paso del
// camino guiado.
//
// # POR QUE ES OTRA FUNCION Y NO UNA LINEA MAS EN montajesDelCamino
//
// Porque aquella saca el prefijo de la DECLARACION DEL CAMINO, a proposito: dos
// copias de una direccion se separan y el sintoma es un 404 en un paso. Aqui el
// prefijo NO puede salir de ahi, porque esta superficie no esta declarada en el
// camino y no va a estarlo. Meterla en la misma funcion obligaria a que su
// godoc dijera «el prefijo sale del camino, salvo esta», que es como una regla
// deja de ser una regla.
//
// # Y POR QUE NO ESTA EN EL CAMINO, con el numero delante
//
// El modelo del TTFV cobra 45 s de lectura por cada paso. La medida del
// 10-09-2026 es 14m38s sobre un presupuesto de 15m0s: un paso mas, aunque su
// pantalla saliera EN BLANCO, deja el camino en 15m23s y reabre D11-e, que es la
// casilla que decide la fecha de la v1. Se entra a proposito, desde un enlace en
// `/alcance`, que es la misma forma con la que se entra a la seccion de campos.
func montajesFueraDelCamino(doc *documentos.Superficie) []montaje {
	var out []montaje
	if doc != nil {
		out = append(out, montaje{prefijo: documentos.BasePorDefecto + "/", h: doc})
	}
	return out
}
