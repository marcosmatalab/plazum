package main

// EL CABLE DE LA PIEZA 7: de un PDF subido a una ficha propuesta y confirmada.
//
// # La cadena, entera y en orden
//
//	metadatos.Proponer(doc, fuenteDe)   -> los candidatos, cada uno con el
//	                                       trozo LITERAL del que sale
//	ia.Verificador.Verificar(...)       -> el hash de ese trozo contra su
//	                                       fuente. La que no resuelve SE CAE
//	Almacen.Aceptar(ctx, quien, p)      -> lo confirmado, con quien y cuando
//
// # LA PUERTA ANTIALUCINACION, Y AQUI NO ES UNA FORMALIDAD
//
// La extraccion es determinista, asi que «la propuesta se ha inventado el texto»
// no deberia poder ocurrir nunca. Se verifica igual, y por dos razones que no
// son la misma:
//
//   - porque lo que la verificacion descarta de verdad es la DESINCRONIZACION.
//     Si el indice de una cuenta se rehace y la propuesta viene de un documento
//     que ya no esta, su cita deja de resolver. Ensenarla entonces seria citar
//     un documento que el cliente no tiene, que es exactamente lo que la puerta
//     existe para impedir, con modelo o sin el.
//   - porque la puerta se escribe UNA VEZ y vale para el dia que la extraccion
//     deje de ser determinista. Ponerla despues, con una pieza que ya funciona,
//     es la clase de trabajo que nunca gana la prioridad.
//
// La propuesta que no verifica NO se ensena con un aviso: se descarta. Un aviso
// junto a un dato dudoso se lee por encima y se acepta igual.
//
// # SIN MODELO EN NINGUN ESLABON
//
// `metadatos` son patrones fijos y la verificacion es un sha256, asi que esto
// pasa con `PLAZUM_SIN_IA=1` igual que la pieza 3.
//
// # INVARIANTE 12: LA FICHA ES DE LA CUENTA
//
// Las propuestas salen de los documentos de una cuenta y lo aceptado lleva el
// nombre de quien lo acepto. Las dos cosas van con la cuenta EN LA FIRMA del
// puerto, por lo mismo que el indice: sin ella, la implementacion tendria que
// sacarla de una variable compartida y serviria la ficha de una persona a otra.

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/marcosmatalab/plazum/adaptadores/ia"
	"github.com/marcosmatalab/plazum/adaptadores/ingesta"
	"github.com/marcosmatalab/plazum/adaptadores/metadatos"
	"github.com/marcosmatalab/plazum/puertos"
	"github.com/marcosmatalab/plazum/superficies/documentos"
)

// MinimoCitaDeFicha es cuantas runas tiene que tener como poco la cita de un
// campo de la ficha.
//
// # POR QUE NO VALE EL MINIMO DE `ia`, Y ESTO NO ES AFLOJAR UNA PUERTA
//
// `ia.MinimoCitaPorDefecto` son 24 runas y su numero esta medido para CITAS DE
// UNA NORMA: el termino mas largo del corpus transcrito tiene 19, asi que con 24
// ninguna cita de una sola palabra pasa, que es el atajo de un modelo que
// parafrasea y necesita un trozo literal para colar el resto.
//
// Una cita de ficha no se parece a eso en nada. Es una ETIQUETA MAS SU VALOR
// —«Fecha: 2026-01-15», 17 runas— y su distintividad no viene del largo, viene
// de la etiqueta: no hay forma de parafrasear una fecha. Exigirle 24 no protege
// de nada y descarta la mitad de las fichas legitimas, que fue exactamente lo
// que paso: con el minimo de `ia`, de los cuatro campos de una cabecera normal
// solo pasaban dos, y la pantalla salia a medias sin decir por que. Lo dijo el
// test de extremo a extremo en su primera ejecucion.
//
// # DE DONDE SALE EL 8
//
// De la cita mas corta que los patrones de `adaptadores/metadatos` pueden
// producir, contada y no estimada: la etiqueta mas corta es «Date» (4) mas «: »
// (2) mas la fecha mas corta que `unaFecha` acepta, «1/1/2026» (8), o sea 14
// runas. Para el firmante y el alcance el valor exige 2 runas como minimo, y la
// etiqueta mas corta es «Firma» (5), o sea 9. Con 8 no se descarta ninguna cita
// que estos patrones puedan generar, y sigue siendo imposible que pase una cita
// vacia o de un caracter, que es contra lo que un minimo protege de verdad.
//
// Si algun dia esto lo escribe un modelo, este numero se vuelve a mirar Y se
// dice en el commit: un minimo pensado para un extractor determinista no es el
// mismo que hace falta contra algo que parafrasea.
const MinimoCitaDeFicha = 8

// MaxCamposAceptadosPorCuenta acota cuantos campos confirmados se guardan.
//
// EL NUMERO SALE DE LA CUENTA Y NO DEL GUSTO: son cuatro campos por documento y
// el tope de documentos de una cuenta lo pone ya `MaxBytesIndexadosPorCuenta`.
// Con 80 caben veinte documentos con su ficha entera, que es mucho mas de lo que
// cabe en el indice. Existe para que una cuenta no pueda crecer sin techo
// pulsando un boton, no para acotar el uso normal.
const MaxCamposAceptadosPorCuenta = 80

// ErrFichaLlena: la cuenta ha llegado a su tope de campos confirmados.
var ErrFichaLlena = fmt.Errorf("documentos: la cuenta ha llegado a su tope de campos aceptados")

// propuestaGuardada es una propuesta con lo que hace falta para volver a
// verificarla y para saber de que documento sale.
type propuestaGuardada struct {
	p metadatos.Propuesta
	// hash es el de la FUENTE de la que sale, guardado al proponer.
	//
	// SE GUARDA Y NO SE RECOMPUTA, y la primera version lo recomputaba mal:
	// llamaba a `ia.HashDe(id, "")`, o sea el hash del identificador con el
	// texto vacio, que no es el hash de ninguna fuente. El resultado era que
	// TODAS las propuestas se descartaban en la verificacion y la pantalla salia
	// sin ficha, con la puerta antialucinacion funcionando de mas. Lo dijo el
	// test de extremo a extremo en su primera ejecucion.
	hash    string
	huella  string
	fichero string
}

// proponerLaFicha saca los candidatos de un documento recien leido.
//
// SE LLAMA DENTRO DE Subir Y NO AL PINTAR, y es una decision de coste: leer el
// texto es lo caro y ya se ha hecho. Recomputar la ficha en cada GET de la
// pantalla obligaria a guardar el texto entero de cada documento solo para eso.
func proponerLaFicha(doc ingesta.Documento, fuentes []ia.Fuente,
	huella, fichero string) []propuestaGuardada {

	// EL IDENTIFICADOR DE LA FUENTE SE SACA DE LA LISTA QUE ACABA DE PRODUCIR
	// `ia.FuentesAportadas` sobre ESTE MISMO documento, en la misma llamada. Es
	// la misma proyeccion que ya hace `sitios` en Subir y por el mismo motivo:
	// no son dos listas construidas por separado (que es lo que el invariante 7
	// prohibe), son la misma lista vista dos veces.
	porOrden := map[int]string{}
	hashPorID := map[string]string{}
	for i, f := range fuentes {
		if i < len(doc.Fragmentos) {
			porOrden[doc.Fragmentos[i].Orden] = f.ID
		}
		hashPorID[f.ID] = f.Hash
	}
	var out []propuestaGuardada
	for _, p := range metadatos.Proponer(doc, func(orden int) string { return porOrden[orden] }) {
		out = append(out, propuestaGuardada{
			p: p, hash: hashPorID[p.Fuente], huella: huella, fichero: fichero,
		})
	}
	return out
}

// Ficha son los campos propuestos de esa cuenta, YA VERIFICADOS POR HASH.
//
// LA VERIFICACION VA AQUI Y NO AL PROPONER, y el sitio importa: el verificador
// de una cuenta se rehace con cada subida, asi que una propuesta guardada ayer
// tiene que volver a resolver contra el de HOY. Verificarla una sola vez al
// nacer dejaria pasar exactamente el caso que esta puerta existe para cazar.
func (a *indicesPorCuenta) Ficha(_ context.Context, quien string) (
	[]documentos.PropuestaDeFicha, error) {

	cuenta, err := a.clave(quien)
	if err != nil {
		return nil, err
	}
	a.mu.RLock()
	c := a.por[cuenta]
	a.mu.RUnlock()
	if c == nil || c.verFicha == nil {
		return nil, nil
	}
	out := make([]documentos.PropuestaDeFicha, 0, len(c.ficha))
	for _, g := range c.ficha {
		ver, err := c.verFicha.Verificar(puertos.Propuesta{
			Diff:       "proponer " + string(g.p.Campo) + " de la ficha",
			Cita:       g.p.Cita,
			HashFuente: g.hash,
			Modelo:     "ninguno: extraccion determinista",
		})
		if err != nil {
			// LA QUE NO VERIFICA NO SE ENSENA. No lleva aviso y no se cuenta
			// aparte: si su cita no resuelve contra el documento que esta
			// cuenta tiene ahora, lo que se ensenaria es una ficha de un
			// documento que no esta.
			continue
		}
		out = append(out, documentos.PropuestaDeFicha{
			Campo: claveDelCampo(g.p.Campo),
			Valor: g.p.Valor,
			// EL PARRAFO SALE DEL VERIFICADOR Y NO DE LA PROPUESTA. Es la
			// diferencia entera: lo que se ensena es el trozo que el hash
			// acaba de resolver contra la fuente, no el que traia dentro
			// quien propuso.
			Parrafo:   ver.Cita(),
			Documento: g.fichero,
			Pagina:    g.p.Pagina,
			Fragmento: g.p.Orden,
			Huella:    g.huella,
		})
	}
	return out, nil
}

// Aceptar guarda un campo confirmado, con quien y cuando.
func (a *indicesPorCuenta) Aceptar(_ context.Context, quien string,
	p documentos.PropuestaDeFicha) error {

	cuenta, err := a.clave(quien)
	if err != nil {
		return err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	c := a.por[cuenta]
	if c == nil {
		return fmt.Errorf("documentos: esta cuenta no tiene ningun documento subido")
	}
	if len(c.aceptados) >= MaxCamposAceptadosPorCuenta {
		return fmt.Errorf("%w: %d campos, sobre un tope de %d", ErrFichaLlena,
			len(c.aceptados), MaxCamposAceptadosPorCuenta)
	}
	ac := documentos.CampoAceptado{
		Campo: p.Campo, Valor: p.Valor,
		// QUIEN Y CUANDO. `quien` es el sujeto de la sesion tal cual llega, y
		// el instante sale del reloj de este proceso. Los dos son datos y no
		// pasan por el catalogo: traducir el nombre de una persona es
		// reescribir quien es.
		Quien:     quien,
		Cuando:    a.ahora().UTC().Format(time.RFC3339),
		Documento: p.Documento,
		Parrafo:   p.Parrafo,
	}
	// ACEPTAR DOS VECES EL MISMO CAMPO SUSTITUYE, NO DUPLICA. Y sustituye
	// entero, con su nuevo firmante y su nueva hora: cambiar de opinion sobre la
	// fecha de un documento es un acto nuevo y lo firma quien lo hace ahora.
	for i, y := range c.aceptados {
		if y.Campo == ac.Campo && y.Documento == ac.Documento {
			c.aceptados[i] = ac
			return nil
		}
	}
	c.aceptados = append(c.aceptados, ac)
	return nil
}

// Aceptados son los campos que esa cuenta ya confirmo.
func (a *indicesPorCuenta) Aceptados(_ context.Context, quien string) (
	[]documentos.CampoAceptado, error) {

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
	out := append([]documentos.CampoAceptado(nil), c.aceptados...)
	// ORDEN ESTABLE: por documento y despues por campo. Recorrer un slice en el
	// orden en que se pulsaron los botones tambien seria estable, pero cambia
	// entre dos personas que hagan lo mismo en otro orden, y una lista que se
	// reordena sola es una lista que nadie compara.
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Documento != out[j].Documento {
			return out[i].Documento < out[j].Documento
		}
		return out[i].Campo < out[j].Campo
	})
	return out, nil
}

// claveDelCampo traduce el campo del extractor a la clave de catalogo con la que
// la pantalla lo nombra.
//
// EL NOMBRE DEL CAMPO ES VOCABULARIO DE PLAZUM Y SE TRADUCE, a diferencia del
// valor, que son palabras del documento del cliente. Por eso viaja como clave y
// no como texto: es el mismo reparto que el resto de esta superficie.
func claveDelCampo(c metadatos.Campo) string {
	return "documentos.ficha.campo." + strings.ToLower(string(c))
}
