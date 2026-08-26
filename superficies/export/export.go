// Package export convierte el expediente en un log de auditoria para un SIEM,
// en JSON lineas: una linea, un evento, sin envoltura.
//
// # Por que JSON lineas y no otra cosa
//
// Es lo unico que traga cualquier SIEM sin transformador: Splunk (con
// INDEXED_EXTRACTIONS=json), Elastic (filebeat con ndjson), Sentinel (regla de
// recoleccion con json/stream), Loki (promtail con etapa json). Un array
// envuelto obliga a leer el fichero entero antes de la primera linea, y un
// fichero que crece mientras se lee no se puede envolver.
//
// # Que es un evento de auditoria AQUI, y que no
//
// Esto NO es un volcado del ledger. El ledger es la capa probatoria: sus
// entradas van cifradas con clave por entrada y su valor esta en que un tercero
// las recalcule, no en que las lea un panel. El SIEM es lo contrario: un
// receptor de terceros, con retencion propia, al que se le manda texto en claro
// para que dispare alertas.
//
// De ahi salen las dos reglas de esta superficie:
//
//  1. Sale METADATO de la cadena (indice, hash, encadenamiento, checkpoints,
//     lapidas) mas una PROYECCION de campos conocidos del contenido. Nunca el
//     contenido en crudo, porque nadie ha revisado lo que hay dentro.
//  2. Lo que el borrado legal borro NO sale, y la supresion manda sobre la
//     clave. Ver LA GUARDA, en contenidoDe.
//
// # La lapida manda sobre la clave
//
// Borrar es destruir la clave de la entrada y firmar una lapida con su base
// legal. Los dos actos viven en sitios distintos: la lapida viaja dentro de la
// cadena firmada, y la clave vive en el keystore, que se replica aparte y con
// retencion propia. O sea que pueden discrepar, y discrepan justo cuando mas
// duele: una copia de seguridad restaurada dentro de la ventana de retencion
// devuelve una clave que ya se habia destruido.
//
// Un export que decidiera "si tengo la clave, lo exporto" convertiria esa
// discrepancia operativa en una reaparicion del dato borrado, en texto plano,
// dentro del SIEM de un tercero que lo retiene un ano y al que ya no llega
// ninguna orden de supresion nuestra. Por eso la guarda pregunta por la LAPIDA
// y no por la clave, y lo pregunta ANTES de mirar si hay clave.
//
// Y por lo mismo vale cualquiera de las dos senales de supresion, no solo la
// lapida: suprimir son dos escrituras (la lapida dentro de la cadena y la
// atribucion en el expediente) y entre ellas cabe un fallo. Retener de mas es un
// evento con menos campos; filtrar de mas no tiene vuelta.
//
// # Determinismo
//
// Dos ejecuciones sobre el mismo expediente dan el mismo fichero byte a byte, o
// el receptor no puede detectar huecos. No hay ningun mapa dentro del evento (el
// orden de un mapa en Go es aleatorio por ejecucion), los instantes se
// normalizan a UTC con ancho fijo y la ordenacion final es total: instante,
// accion e identificador.
package export

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"plazum/nucleo/estado"
	"plazum/nucleo/expediente"
	"plazum/nucleo/ledger"
)

// Esquema es la version del formato de evento. Sube cuando cambia el significado
// de un campo, no cuando se anade uno: el consumidor de un SIEM tiene reglas
// escritas contra estos nombres y necesita saber contra que version las escribio.
const Esquema = "1"

// Lo que identifica el origen dentro del SIEM.
const (
	Dataset  = "plazum.auditoria"
	Modulo   = "plazum"
	Producto = "plazum"
)

// Las acciones, en lista cerrada. Un SIEM escribe sus alertas contra estos
// valores, asi que anadir uno es barato y cambiar uno rompe al cliente.
const (
	AccionEntrada     = "ledger.entrada"
	AccionCheckpoint  = "ledger.punto_de_control"
	AccionSupresion   = "ledger.supresion_legal"
	AccionEstado      = "control.estado"
	AccionVencimiento = "obligacion.vencimiento"
)

// Los tres desenlaces de ECS. No hay mas: un campo con cuatro valores posibles
// deja de servir para agrupar, que es para lo unico que existe.
const (
	Exito       = "success"
	Fallo       = "failure"
	Desconocido = "unknown"
)

// De donde sale el instante de cada evento, dicho en el propio evento.
//
// Hace falta porque la cadena NO fecha sus entradas: una entrada v2 lleva
// indice, encadenamiento y cifrado, y ninguna marca de tiempo. Lo unico fechado
// y anclado es el checkpoint, que acota por arriba el instante de todo lo que
// cubre. Un export que escribiera esa cota como si fuera el momento del hecho
// estaria inventando precision, que es justo lo que este producto no hace.
const (
	InstanteObservacion = "observacion_recolectada"
	InstanteCheckpoint  = "checkpoint"
	InstanteCota        = "cota_superior_del_checkpoint"
	InstanteLapida      = "lapida"
	InstanteEmision     = "emision_del_expediente"
	InstanteVencimiento = "vencimiento_declarado"
)

// MaxValor acota cada valor de texto que sale. Un recolector que devuelve medio
// megabyte en un campo no puede reventar la ingesta del receptor, y un SIEM que
// trunca por su cuenta lo hace por bytes y parte el UTF-8 por la mitad.
const MaxValor = 512

// formatoInstante es UTC con milisegundos y ancho FIJO, a proposito: con ancho
// fijo el orden lexicografico coincide con el cronologico, y de ahi sale que la
// ordenacion del fichero sea total sin comparar time.Time.
const formatoInstante = "2006-01-02T15:04:05.000"

// Evento es una linea del fichero.
//
// Los nombres de campo son ECS (Elastic Common Schema) para lo que todo SIEM ya
// sabe mapear solo, y con prefijo propio para lo que es de este dominio. El
// orden de los campos aqui es el orden en el que salen, y eso es parte del
// contrato de determinismo: sin struct no habria orden, porque un mapa de Go no
// lo tiene.
type Evento struct {
	Instante  string `json:"@timestamp"`
	Clase     string `json:"event.kind"`
	DataSet   string `json:"event.dataset"`
	Modulo    string `json:"event.module"`
	Accion    string `json:"event.action"`
	Secuencia uint64 `json:"event.sequence"`
	ID        string `json:"event.id"`
	Resultado string `json:"event.outcome"`
	Mensaje   string `json:"message"`

	Organizacion string `json:"organization.name"`
	Observador   string `json:"observer.product"`
	Usuario      string `json:"user.name,omitempty"`

	Version    string `json:"plazum.esquema"`
	InstanteEs string `json:"plazum.instante_es"`

	Entrada    *uint64 `json:"plazum.entrada,omitempty"`
	Hash       string  `json:"plazum.hash,omitempty"`
	HashPrevio string  `json:"plazum.hash_previo,omitempty"`

	CheckpointHasta *uint64 `json:"plazum.checkpoint_hasta,omitempty"`
	RaizMerkle      string  `json:"plazum.raiz_merkle,omitempty"`
	Anclaje         string  `json:"plazum.anclaje_declarado,omitempty"`
	Sellado         *bool   `json:"plazum.sellado,omitempty"`
	ClavePublica    string  `json:"plazum.clave_publica,omitempty"`

	BaseLegal string `json:"plazum.base_legal,omitempty"`

	Prueba     string `json:"plazum.prueba,omitempty"`
	Recurso    string `json:"plazum.recurso,omitempty"`
	EstadoDecl string `json:"plazum.estado,omitempty"`
	Motivo     string `json:"plazum.motivo,omitempty"`
	Escala     *bool  `json:"plazum.escala_al_auditor,omitempty"`

	Obligacion string `json:"plazum.obligacion,omitempty"`
	Hito       string `json:"plazum.hito,omitempty"`
	Vence      string `json:"plazum.vence,omitempty"`
	Vencido    *bool  `json:"plazum.vencido,omitempty"`
	Regla      string `json:"plazum.regla,omitempty"`

	Sujeto           string `json:"plazum.sujeto,omitempty"`
	TipoDeEntrada    string `json:"plazum.tipo_de_entrada,omitempty"`
	Evidencia        string `json:"plazum.evidencia,omitempty"`
	Recolector       string `json:"plazum.recolector,omitempty"`
	VersionRecol     string `json:"plazum.version_recolector,omitempty"`
	HashCarga        string `json:"plazum.hash_carga,omitempty"`
	Caduca           string `json:"plazum.caduca,omitempty"`
	ErrorRecoleccion *bool  `json:"plazum.error_de_recoleccion,omitempty"`

	ContenidoDisponible *bool  `json:"plazum.contenido_disponible,omitempty"`
	MotivoSinContenido  string `json:"plazum.motivo_sin_contenido,omitempty"`
	CamposOmitidos      *int   `json:"plazum.campos_omitidos,omitempty"`
}

// Resumen es lo que el operador ve por el canal de errores mientras el fichero
// se va por la tuberia. Sin esto, `plazum export ... | nc siem 514` no dice
// absolutamente nada de si mando algo o mando el vacio.
type Resumen struct {
	Eventos    int
	Entradas   int
	Suprimidas int
	Estados    int
	Vencidos   int
}

func (r Resumen) String() string {
	return fmt.Sprintf("%d eventos: %d entradas de la cadena (%d suprimidas con base legal), "+
		"%d controles, %d plazos ya vencidos a fecha del expediente",
		r.Eventos, r.Entradas, r.Suprimidas, r.Estados, r.Vencidos)
}

// resultadoPorEstado traduce los estados del dominio al vocabulario de ECS.
//
// Se indexa por la CONSTANTE, no por su nombre: asi renombrar un estado en
// nucleo/estado rompe la compilacion aqui, en vez de dejar un mapa con una clave
// que ya no existe y un desenlace que se calla.
var resultadoPorEstado = map[estado.Estado]string{
	estado.Pass:        Exito,
	estado.Manual:      Exito,
	estado.Exceptuado:  Exito,
	estado.FailEnPlazo: Fallo,
	estado.FailVencido: Fallo,
	estado.Error:       Fallo,
	estado.Obsoleto:    Desconocido,
	estado.NoAplica:    Desconocido,
}

// estadoPorNombre es el camino de vuelta: el expediente declara el estado como
// cadena (lo escribio el emisor) y hay que resolverlo al del dominio para poder
// preguntarle si escala al auditor.
var estadoPorNombre = func() map[string]estado.Estado {
	m := make(map[string]estado.Estado, len(resultadoPorEstado))
	for e := range resultadoPorEstado {
		m[e.String()] = e
	}
	return m
}()

// Exportar escribe el log de auditoria del expediente en w, en JSON lineas.
func Exportar(w io.Writer, exp *expediente.Expediente) (Resumen, error) {
	evs, res, err := Eventos(exp)
	if err != nil {
		return res, err
	}
	return res, Escribir(w, evs)
}

// Eventos construye la lista completa, ya ordenada y numerada.
//
// Es una funcion pura: mismo expediente, mismos eventos, siempre. El instante
// entra como dato, lo trae el expediente; aqui no hay reloj.
func Eventos(exp *expediente.Expediente) ([]Evento, Resumen, error) {
	if exp == nil {
		return nil, Resumen{}, fmt.Errorf("no hay expediente que exportar. " +
			"Cargalo con expediente.Cargar y pasa el puntero que devuelve")
	}
	c := &constructor{exp: exp, lapidas: map[uint64]ledger.Lapida{},
		porHash: map[string]ledger.Lapida{}, prueba: map[uint64]string{}}
	for _, l := range exp.Cadena.Lapidas {
		if _, ya := c.lapidas[l.EntradaBorrada]; !ya {
			c.lapidas[l.EntradaBorrada] = l
		}
		// El hash vacio no indexa: si lo hiciera, una lapida sin HashEntrada
		// casaria con toda entrada que tampoco lo traiga y taparia la cadena
		// entera.
		if h := hex.EncodeToString(l.HashEntrada); h != "" {
			if _, ya := c.porHash[h]; !ya {
				c.porHash[h] = l
			}
		}
	}
	for _, s := range exp.SupresionesDeEvidencia {
		if _, ya := c.prueba[s.Entrada]; !ya {
			c.prueba[s.Entrada] = s.Prueba
		}
	}

	c.entradas()
	c.checkpoints()
	c.supresiones()
	c.estados()
	c.vencimientos()

	ordenar(c.evs)
	for i := range c.evs {
		c.evs[i].Secuencia = uint64(i) + 1
	}
	c.res.Eventos = len(c.evs)
	return c.evs, c.res, nil
}

// Escribir vuelca los eventos como JSON lineas: un objeto por linea, terminada
// en salto, sin envoltura y sin coma. No se escapa HTML porque el destino no es
// un navegador y el escapado convierte los signos de menor y mayor en < y
// >, que ensucia las busquedas del analista sin protegerlo de nada.
func Escribir(w io.Writer, evs []Evento) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	for i, ev := range evs {
		if err := enc.Encode(ev); err != nil {
			return fmt.Errorf("evento %d (%s): %w", i+1, ev.Accion, err)
		}
	}
	return nil
}

type constructor struct {
	exp     *expediente.Expediente
	lapidas map[uint64]ledger.Lapida
	// porHash indexa las mismas lapidas por el hash de la entrada que borraron,
	// que es lo que la lapida lleva DENTRO de su firma.
	//
	// HALLAZGO DE REVISION HOSTIL: la guarda casaba lapida con entrada solo por
	// el indice, y el indice de una entrada es un entero del fichero que no
	// protege ninguna firma. Renumerar la entrada suprimida (mas mover su clave
	// divulgada de casilla) dejaba la lapida intacta, con su firma buena, y el
	// contenido borrado con base legal salia en claro al SIEM de un tercero.
	// nucleo/ledger ya habia aprendido esto: la lapida se firma sobre
	// HashEntrada precisamente porque el indice solo no ata nada, y de ahi sale
	// su ErrLapidaDeOtraEntrada. Aqui se habia vuelto a casar por indice.
	porHash map[string]ledger.Lapida
	prueba  map[uint64]string // entrada suprimida -> prueba que se quedo sin evidencia
	evs     []Evento
	res     Resumen
}

// base rellena lo que lleva TODO evento. Que este en un solo sitio es lo que
// hace que anadir una accion no pueda olvidarse de la organizacion ni del
// esquema, que son los dos campos por los que el receptor separa origenes.
func (c *constructor) base(accion string, cuando time.Time, deDonde string) Evento {
	return Evento{
		Instante:     instante(cuando),
		Clase:        "event",
		DataSet:      Dataset,
		Modulo:       Modulo,
		Accion:       accion,
		Resultado:    Desconocido,
		Organizacion: recortar(c.exp.Organizacion),
		Observador:   Producto,
		Version:      Esquema,
		InstanteEs:   deDonde,
	}
}

func (c *constructor) entradas() {
	for _, ent := range c.exp.Cadena.Entradas {
		idx := ent.Indice
		cuando, deDonde := c.cotaDe(idx)
		ev := c.base(AccionEntrada, cuando, deDonde)
		ev.Entrada = &idx
		ev.Hash = hex.EncodeToString(ent.Hash)
		ev.HashPrevio = hex.EncodeToString(ent.Previo)

		campos, motivo := c.contenidoDe(ent)
		if motivo != "" {
			no := false
			ev.ContenidoDisponible = &no
			ev.MotivoSinContenido = motivo
		} else {
			si := true
			ev.ContenidoDisponible = &si
			c.proyectar(&ev, campos)
		}
		ev.Mensaje = mensajeEntrada(ev)
		ev.ID = identidad(ev.Accion, c.exp.Organizacion, ev.Hash)
		c.evs = append(c.evs, ev)
		c.res.Entradas++
	}
}

// contenidoDe decide si el contenido de una entrada puede salir, y lo abre.
//
// LA GUARDA. La primera pregunta es por la supresion, y se hace ANTES de mirar
// si hay clave. Si la entrada esta suprimida no se toca el material cifrado, no
// se busca clave y no se descifra nada: la unica salida posible es el motivo.
//
// Quitar esas lineas hace que una entrada borrada con base legal salga con su
// contenido en claro en cuanto la clave siga por ahi: una replica del keystore
// que aun no ha caducado, una copia de seguridad restaurada, o un emisor que
// puso la lapida y no limpio las claves divulgadas. Es la mutacion que vigila
// TestElExportNoFiltraLoQueElBorradoLegalBorro, que lleva su control negativo
// dentro.
//
// Y vale CUALQUIERA de las dos senales de supresion, no solo la lapida. La
// segunda salio de preguntarse que pasa con un borrado a medias: la supresion
// son dos escrituras en sitios distintos (la lapida firmada dentro de la cadena
// y la atribucion en el expediente), y entre las dos cabe un fallo. Un emisor
// que declara que la entrada 3 se suprimio y no consigue escribir la lapida
// tiene un expediente que no verifica, pero eso lo dice `plazum verify` DESPUES;
// mientras tanto, exportar ese contenido a un tercero es irreversible. Ante la
// duda, no sale: retener de mas es un evento con menos campos, y filtrar de mas
// no se deshace.
func (c *constructor) contenidoDe(ent ledger.EntradaV2) (map[string]json.RawMessage, string) {
	if l, suprimida := c.lapidas[ent.Indice]; suprimida {
		return nil, "suprimida con base legal " + recortar(l.BaseLegal)
	}
	// Y por el hash, que es por donde la lapida esta firmada. El indice de una
	// entrada es un entero del fichero que no protege ninguna firma: sin esta
	// linea, renumerar la entrada suprimida saca su contenido en claro.
	if l, suprimida := c.porHash[hex.EncodeToString(ent.Hash)]; suprimida {
		return nil, "suprimida con base legal " + recortar(l.BaseLegal) +
			"; la lapida esta firmada sobre el hash de esta entrada aunque su indice ya no cuadre"
	}
	if _, declarada := c.prueba[ent.Indice]; declarada {
		return nil, "el emisor declara suprimida esta entrada y no hay lapida que lo respalde"
	}
	hexClave, hay := c.exp.ClavesEntradas[ent.Indice]
	if !hay {
		return nil, "el emisor no divulga la clave de esta entrada"
	}
	clave, err := hex.DecodeString(hexClave)
	if err != nil {
		return nil, "la clave divulgada no es hexadecimal"
	}
	claro, err := ledger.AbrirComprometido(clave, ent.Nonce, ent.Cifrado, ent.Compromiso)
	if err != nil {
		return nil, "la clave divulgada no abre esta entrada"
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(claro, &m); err != nil {
		return nil, "el contenido no es un objeto JSON con campos conocidos"
	}
	return m, ""
}

// proyectar copia al evento SOLO las claves de la lista cerrada de abajo.
//
// Es lista blanca y no lista negra, y esa es la regla de los secretos: una clave
// que nadie ha revisado no sale. Un recolector que algun dia escriba
// Authorization o un identificador de sesion dentro de la carga no filtra nada
// por este camino, porque no hace falta acordarse de prohibirlo.
//
// El campo de error del recolector esta EXCLUIDO a proposito, y es el caso que
// da sentido a lo anterior: es texto libre que devuelve un tercero, y ahi es
// donde acaba una URL firmada o una cabecera con credencial cuando un recolector
// falla. Lo que el SIEM necesita de ese campo, que hubo error, sale como
// booleano.
func (c *constructor) proyectar(ev *Evento, campos map[string]json.RawMessage) {
	omitidos := 0
	claves := make([]string, 0, len(campos))
	for k := range campos {
		claves = append(claves, k)
	}
	sort.Strings(claves) // ni el recuento ni el resultado dependen del orden del mapa
	for _, k := range claves {
		crudo := campos[k]
		switch k {
		case "Prueba":
			ev.Prueba = cadena(crudo)
		case "Recurso":
			ev.Recurso = cadena(crudo)
		case "Recolector":
			ev.Recolector = cadena(crudo)
		case "Version":
			ev.VersionRecol = cadena(crudo)
		case "HashCarga":
			ev.HashCarga = cadena(crudo)
		case "Satisfecho":
			if b, ok := booleano(crudo); ok {
				ev.Resultado = Fallo
				if b {
					ev.Resultado = Exito
				}
			}
		case "Recolectada":
			if t, ok := fecha(crudo); ok {
				ev.Instante = instante(t)
				ev.InstanteEs = InstanteObservacion
			}
		case "Caduca":
			if t, ok := fecha(crudo); ok {
				ev.Caduca = instante(t)
			}
		case "ErrorRecol":
			// El TEXTO no sale nunca. Solo el hecho de que lo hubo.
			if s := cadena(crudo); s != "" {
				si := true
				ev.ErrorRecoleccion = &si
			}
		case "tipo":
			ev.TipoDeEntrada = cadena(crudo)
		case "actor":
			ev.Usuario = cadena(crudo)
		case "sujeto":
			ev.Sujeto = cadena(crudo)
		case "evidencia":
			ev.Evidencia = cadena(crudo)
		default:
			omitidos++
		}
	}
	if omitidos > 0 {
		ev.CamposOmitidos = &omitidos
	}
}

func (c *constructor) checkpoints() {
	for _, cp := range c.exp.Cadena.Checkpoints {
		hasta := cp.Hasta
		ev := c.base(AccionCheckpoint, cp.Instante, InstanteCheckpoint)
		ev.CheckpointHasta = &hasta
		ev.RaizMerkle = cp.RaizMerkle
		ev.Anclaje = recortar(cp.AnclajeDeclarado)
		ev.ClavePublica = cp.ClavePub
		sellado := len(cp.Token) > 0
		ev.Sellado = &sellado
		ev.Resultado = Exito
		if !sellado {
			// Un checkpoint sin sello no esta anclado: queda pendiente de
			// anclar, y eso es exactamente lo que el receptor tiene que poder
			// alertar sin leerse el expediente entero.
			ev.Resultado = Fallo
		}
		ev.Mensaje = fmt.Sprintf("punto de control sobre %d entrada(s), raiz %s, sellado=%t",
			cp.Hasta, corto(cp.RaizMerkle), sellado)
		ev.ID = identidad(ev.Accion, c.exp.Organizacion, fmt.Sprintf("%d|%s", cp.Hasta, cp.RaizMerkle))
		c.evs = append(c.evs, ev)
	}
}

func (c *constructor) supresiones() {
	for _, l := range c.exp.Cadena.Lapidas {
		idx := l.EntradaBorrada
		cuando, deDonde := c.instanteLapida(l, idx)
		ev := c.base(AccionSupresion, cuando, deDonde)
		ev.Entrada = &idx
		ev.Hash = hex.EncodeToString(l.HashEntrada)
		ev.BaseLegal = recortar(l.BaseLegal)
		ev.Prueba = recortar(c.prueba[idx])
		ev.Resultado = Exito
		ev.Mensaje = fmt.Sprintf("entrada %d suprimida con base legal %s", idx, ev.BaseLegal)
		if ev.Prueba != "" {
			ev.Mensaje += "; se queda sin evidencia " + ev.Prueba
		}
		ev.ID = identidad(ev.Accion, c.exp.Organizacion,
			fmt.Sprintf("%d|%s|%s", idx, ev.Hash, l.Instante))
		c.evs = append(c.evs, ev)
		c.res.Suprimidas++
	}
}

func (c *constructor) estados() {
	for _, s := range c.exp.Estados {
		ev := c.base(AccionEstado, c.exp.ComoEstaba, InstanteEmision)
		ev.Prueba = recortar(s.Prueba)
		ev.EstadoDecl = recortar(s.Estado)
		ev.Motivo = recortar(s.Motivo)
		if e, conocido := estadoPorNombre[s.Estado]; conocido {
			ev.Resultado = resultadoPorEstado[e]
			escala := e.EscalaAlAuditor()
			ev.Escala = &escala
		}
		ev.Mensaje = fmt.Sprintf("control %s declarado %s", ev.Prueba, ev.EstadoDecl)
		ev.ID = identidad(ev.Accion, c.exp.Organizacion,
			fmt.Sprintf("%s|%s", s.Prueba, instante(c.exp.ComoEstaba)))
		c.evs = append(c.evs, ev)
		c.res.Estados++
	}
}

func (c *constructor) vencimientos() {
	for _, r := range c.exp.Reclamaciones {
		cuando, deDonde := r.Vence, InstanteVencimiento
		if r.Vence.IsZero() {
			// Un hito pendiente de hecho existe y todavia no tiene fecha.
			// Fecharlo en el ano 1 llenaria el panel del receptor de eventos
			// del siglo I, que es como se pierde la confianza en un origen.
			cuando, deDonde = c.exp.ComoEstaba, InstanteEmision
		}
		ev := c.base(AccionVencimiento, cuando, deDonde)
		ev.Obligacion = recortar(r.Obligacion)
		ev.Hito = recortar(r.Hito)
		ev.EstadoDecl = recortar(r.Estado)
		ev.Regla = recortar(r.Regla)
		if !r.Vence.IsZero() {
			ev.Vence = instante(r.Vence)
			// Comparacion entre dos datos que el expediente ya declara, no un
			// recalculo del motor: el plazo y la fecha a la que el expediente
			// dice retratar la organizacion.
			vencido := r.Vence.Before(c.exp.ComoEstaba)
			ev.Vencido = &vencido
			if vencido {
				ev.Resultado = Fallo
				c.res.Vencidos++
			}
		}
		ev.Mensaje = fmt.Sprintf("obligacion %s, hito %s (%s)", ev.Obligacion, ev.Hito, ev.EstadoDecl)
		ev.ID = identidad(ev.Accion, c.exp.Organizacion,
			fmt.Sprintf("%s|%s|%s", r.Obligacion, r.Hito, instante(r.Vence)))
		c.evs = append(c.evs, ev)
	}
}

// cotaDe devuelve el instante del checkpoint mas temprano que cubre la entrada.
// Es una COTA SUPERIOR, no el momento del hecho, y el evento lo dice en su campo
// de procedencia del instante.
func (c *constructor) cotaDe(indice uint64) (time.Time, string) {
	var mejor time.Time
	hay := false
	for _, cp := range c.exp.Cadena.Checkpoints {
		if cp.Hasta <= indice {
			continue
		}
		if !hay || cp.Instante.Before(mejor) {
			mejor, hay = cp.Instante, true
		}
	}
	if hay {
		return mejor, InstanteCota
	}
	return c.exp.ComoEstaba, InstanteEmision
}

// instanteLapida usa el instante que firma la lapida. Si no se puede leer, se
// cae a la cota del checkpoint, nunca al reloj de la maquina.
func (c *constructor) instanteLapida(l ledger.Lapida, indice uint64) (time.Time, string) {
	if t, err := time.Parse(time.RFC3339, l.Instante); err == nil && !t.IsZero() {
		return t, InstanteLapida
	}
	return c.cotaDe(indice)
}

func mensajeEntrada(ev Evento) string {
	var b strings.Builder
	fmt.Fprintf(&b, "entrada %d de la cadena", *ev.Entrada)
	if ev.Prueba != "" {
		b.WriteString(", prueba " + ev.Prueba)
	}
	if ev.Recurso != "" {
		b.WriteString(", recurso " + ev.Recurso)
	}
	if ev.MotivoSinContenido != "" {
		b.WriteString(": " + ev.MotivoSinContenido)
	}
	return b.String()
}

// identidad da un id estable por evento para que el SIEM pueda deduplicar dos
// exportaciones que se solapan. Se calcula sobre la accion y la clave natural, y
// NUNCA sobre el contenido: si dependiera del contenido, una entrada suprimida
// cambiaria de id al desaparecer su carga y el receptor contaria el mismo hecho
// dos veces, una antes del borrado y otra despues.
func identidad(accion, organizacion, clave string) string {
	s := sha256.Sum256([]byte(accion + "|" + organizacion + "|" + clave))
	return hex.EncodeToString(s[:16])
}

func instante(t time.Time) string {
	return t.UTC().Format(formatoInstante) + "Z"
}

func corto(s string) string {
	if len(s) <= 12 {
		return s
	}
	return s[:12]
}

// recortar deja el valor en algo que un SIEM puede ingerir y un analista puede
// leer: sin caracteres de control (una secuencia de escape dentro de un campo
// pinta lo que quiera en la terminal del que lee los logs, y una linea nueva
// parte un evento en dos) y acotado por MaxValor, cortando por frontera de runa
// para no partir el UTF-8.
func recortar(s string) string {
	s = strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' || r == '\t' {
			return ' '
		}
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, s)
	if len(s) <= MaxValor {
		return s
	}
	corte := MaxValor
	for corte > 0 && !utf8.RuneStart(s[corte]) {
		corte--
	}
	return s[:corte] + " [recortado]"
}

func cadena(crudo json.RawMessage) string {
	var s string
	if err := json.Unmarshal(crudo, &s); err != nil {
		return ""
	}
	return recortar(s)
}

func booleano(crudo json.RawMessage) (bool, bool) {
	var b bool
	if err := json.Unmarshal(crudo, &b); err != nil {
		return false, false
	}
	return b, true
}

func fecha(crudo json.RawMessage) (time.Time, bool) {
	s := ""
	if err := json.Unmarshal(crudo, &s); err != nil {
		return time.Time{}, false
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil || t.IsZero() {
		return time.Time{}, false
	}
	return t, true
}

// ordenar impone un orden TOTAL: instante, accion e identificador. Sin el tercer
// criterio, dos eventos de la misma accion en el mismo milisegundo quedarian
// empatados, y sort.Slice no es estable, asi que el fichero podria cambiar entre
// dos ejecuciones sobre el mismo expediente.
func ordenar(evs []Evento) {
	sort.Slice(evs, func(i, j int) bool {
		a, b := evs[i], evs[j]
		if a.Instante != b.Instante {
			return a.Instante < b.Instante
		}
		if a.Accion != b.Accion {
			return a.Accion < b.Accion
		}
		return a.ID < b.ID
	})
}
