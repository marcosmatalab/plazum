package estado

import (
	"fmt"
	"sort"
)

// LOS MOTIVOS DEL MOTOR, EN CLAVE Y NO EN FRASE.
//
// # El defecto que cierra
//
// `Calcular` escribia el porque de cada estado en castellano y esa cadena viajaba
// CRUDA hasta la plantilla de la tabla de controles, asi que la pagina inglesa
// imprimia espanol. Y ninguna puerta de i18n podia verlo: todas vigilan el
// CATALOGO, y esto no era una clave. **La forma de publicar sin traducir en este
// repositorio es no pasar por el catalogo**, y la pantalla de controles fue la
// primera que lo hizo.
//
// # Por que la frase NO se reescribe en la superficie
//
// Porque seria una segunda copia de la derivacion, y la segunda copia es la que
// se queda vieja: el dia que el motor cambie un motivo, la pantalla seguiria
// explicando el anterior con toda la cara de estar en lo cierto. Es la misma
// razon por la que el acta no deja que la pantalla redacte sus descargos.
//
// # La forma, que es la que ya resolvio el acta el 01-09
//
// El nucleo emite CLAVE mas ARGUMENTOS, no frase. La clave la traduce el
// catalogo; los argumentos son datos (un recuento, una fecha, el nombre de un
// recurso, las palabras de una persona) y esos no se traducen nunca. Y se emite
// ADEMAS el espanol ya resuelto, porque el expediente sale del proceso sin
// navegador delante y ahi no hay catalogo que lo resuelva, igual que el board
// pack impreso del acta.
//
// LAS TRES COSAS SALEN DE UNA SOLA CONSTANTE, y eso es lo que impide que se
// separen: `porque()` rellena la misma plantilla que viaja al catalogo, asi que
// el texto del expediente y el de la pantalla en espanol no pueden decir cosas
// distintas. No hay manera de escribir aqui una frase sin su clave.
//
// LO VIGILA: TestElCatalogoDiceDelEstadoLoMismoQueElNucleo, que ata las dos
// redacciones a estas constantes letra por letra y en las dos direcciones, y
// TestNingunTextoDeLaVistaLlegaSinPasarPorElCatalogo, que es el que impide que
// esto vuelva a pasar en la pantalla siguiente.

// Frase es una cadena que escribe plazum: su clave de catalogo y la PLANTILLA en
// espanol, con sus huecos todavia sin rellenar.
//
// Es el mismo par que `acta.Frase` y no el mismo tipo a proposito: `nucleo/acta`
// y `nucleo/estado` no se importan, y hacer que uno dependiera del otro para
// compartir dos campos ataria el motor de estados al compositor de actas.
type Frase struct {
	Clave string
	Texto string
}

// Motivo es el porque de un estado: su clave, su espanol ya resuelto y los datos
// que rellenaron los huecos.
//
// LOS ARGUMENTOS VAN APARTE DEL TEXTO porque no se traducen. Un recuento, una
// fecha en ISO, el nombre de un recurso y las palabras de la persona que aprobo
// una excepcion se dicen igual en los dos idiomas; lo que se traduce es la frase
// que los rodea.
type Motivo struct {
	Frase
	Args []string
}

// String devuelve el espanol resuelto, para que un `%s` sobre un Motivo siga
// imprimiendo lo que imprimia cuando esto era una cadena.
func (m Motivo) String() string { return m.Texto }

// Vacio dice si no hay motivo. El valor cero de Motivo es «no se ha escrito
// ninguno», que no es lo mismo que uno con texto vacio y clave puesta.
func (m Motivo) Vacio() bool { return m.Clave == "" && m.Texto == "" }

// Las once plantillas, una por rama de `Calcular` que devuelve.
//
// SON ONCE Y NO DIEZ NI DOCE: cada `return` de `Calcular` trae la suya, y el
// test de arriba recorre las dos direcciones, asi que una rama nueva sin clave
// no compila contra el catalogo y una clave que se quede sin rama se ve.
var (
	mtNoAplica = Frase{"evidencia.motivo.no_aplica",
		"fuera de la declaracion de aplicabilidad"}
	mtExceptuado = Frase{"evidencia.motivo.exceptuado",
		"excepcion aprobada por %s: %s"}
	mtDespliegue = Frase{"evidencia.motivo.despliegue",
		"prueba en periodo de despliegue hasta el %s"}
	mtPassPorDefecto = Frase{"evidencia.motivo.pass_por_defecto",
		"garantizado por el proveedor y no configurable por el cliente"}
	mtSinObservaciones = Frase{"evidencia.motivo.sin_observaciones",
		"sin observaciones para esta prueba"}
	mtFalloVencido = Frase{"evidencia.motivo.fallo_vencido",
		"%s recurso(s) no pasan la comprobacion y el plazo de remediacion vencio el %s"}
	mtFalloYCaducada = Frase{"evidencia.motivo.fallo_y_caducada",
		"hay recursos que no pasan, dentro de plazo, pero la observacion de %s caduco " +
			"el %s: no se puede afirmar el estado actual"}
	mtFalloEnPlazo = Frase{"evidencia.motivo.fallo_en_plazo",
		"%s recurso(s) no pasan la comprobacion, plazo de remediacion hasta el %s"}
	mtCaducada = Frase{"evidencia.motivo.caducada",
		"la observacion de %s caduco el %s: obsoleto no es fallo"}
	mtErrorDeRecoleccion = Frase{"evidencia.motivo.error_de_recoleccion",
		"el recolector no pudo obtener el dato: %s"}
	mtTodasSatisfacen = Frase{"evidencia.motivo.todas_satisfacen",
		"todas las observaciones satisfacen el predicado"}
)

// CadenasDelEstado es TODO lo que este motor escribe en prosa y que por tanto
// tiene que estar en el catalogo. Es lo que recorre el test del catalogo, en las
// dos direcciones.
//
// Se devuelve ordenado por clave para que la salida de un fallo sea estable.
func CadenasDelEstado() []Frase {
	out := []Frase{mtNoAplica, mtExceptuado, mtDespliegue, mtPassPorDefecto,
		mtSinObservaciones, mtFalloVencido, mtFalloYCaducada, mtFalloEnPlazo,
		mtCaducada, mtErrorDeRecoleccion, mtTodasSatisfacen}
	sort.Slice(out, func(i, j int) bool { return out[i].Clave < out[j].Clave })
	return out
}

// porque arma el motivo: la misma plantilla rellenada y los datos que la
// rellenaron.
//
// Es el unico sitio donde se construye un Motivo, y por eso no puede haber uno
// con texto y sin clave. Es la forma de `acta.dePlazum` y esta aqui por lo
// mismo.
func porque(f Frase, args ...string) Motivo {
	if len(args) == 0 {
		return Motivo{Frase: f}
	}
	vals := make([]any, len(args))
	for i, a := range args {
		vals[i] = a
	}
	return Motivo{
		Frase: Frase{Clave: f.Clave, Texto: fmt.Sprintf(f.Texto, vals...)},
		Args:  args,
	}
}

// dia es como se escribe una fecha que va a leer una persona.
//
// NO ES RFC 3339, y el cambio es de la pasada del comprador. Los motivos los
// pinta la tabla de controles al lado de una fecha que YA se formatea asi
// («2026-09-06»), asi que la celda decia la misma clase de dato de dos maneras y
// una de las dos era un volcado de Go. Es D11-a en su forma mas barata: nada
// roto, todo 200, y el producto pareciendo un terminal.
const dia = "2006-01-02"
