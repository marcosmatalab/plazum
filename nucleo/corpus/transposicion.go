package corpus

// UNA DIRECTIVA NO VINCULA POR SI MISMA, Y ESO ES UN DATO, NO UN PARRAFO.
//
// # El caso que lo trae, medido
//
// El paquete `nis2-ue` llevaba el aviso escrito A MANO dentro del campo `cita` de
// **nueve** de sus doce obligaciones, mas una regla de aplicabilidad: diez copias
// del mismo parrafo. Esta bien argumentado y es correcto; el problema es otro.
//
// **El dia que Espana transponga NIS2 hay que editar diez citas en vez de cambiar
// un campo**, y editar diez sitios a mano es como se consigue que ocho digan una
// cosa y dos la contraria. Es exactamente el problema que el `identificador` de
// paquete ya resolvio por el otro extremo: alli la norma se citaba con una URL en
// cada sitio y paso a ser un identificador con el enlace derivado al pintar.
//
// # Y engancha con la fecha de transposicion, que ahora es un dato
//
// `herramientas/ingestanorma` lee `DIRECTIVE_DATE_TRANSPOSITION` de la ficha de
// Cellar desde el 08-09-2026, y trae DOS fechas que la fuente separa: el limite
// para ADOPTAR las medidas y la fecha desde la que se APLICAN. En NIS2 van con un
// dia de diferencia (17 y 18 de octubre de 2024, art. 41.1). Este bloque es donde
// aterrizan.
//
// # Las dos direcciones del linter, y por que la segunda importa tanto
//
//	1. Un paquete cuyo `urn` dice que es una DIRECTIVA declara este bloque.
//	   Sin el, el aviso vuelve a ser prosa repetida o desaparece.
//	2. Un paquete que NO es una directiva no lo declara. Un reglamento con un
//	   bloque de transposicion le estaria diciendo al cliente que espere a una
//	   norma nacional que no va a existir, y eso RETRASA un cumplimiento que ya
//	   vincula. Es el error caro de los dos.
//
// # El valor cero de `consta` es el peligroso (invariante 8)
//
// `Consta: false` es lo que sale de no escribir nada, y significa «no hay norma
// nacional». Es la respuesta con mas consecuencias de las dos, porque es la que
// le dice al cliente que todavia no le obliga. Asi que **no se admite como valor
// por defecto**: toda entrada de pais tiene que traer CUANDO se comprobo y COMO,
// y si no consta, que le vincula mientras tanto o por que no le vincula nada.

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	// ErrDirectivaSinTransposicion: el urn dice directiva y falta el bloque.
	ErrDirectivaSinTransposicion = errors.New("directiva sin bloque de transposicion")
	// ErrTransposicionEnLoQueNoEsDirectiva: la direccion cara.
	ErrTransposicionEnLoQueNoEsDirectiva = errors.New(
		"bloque de transposicion en un marco que no es una directiva")
	// ErrTransposicionSinCita, ErrTransposicionSinPais y las demas: el bloque
	// existe y no dice lo que tiene que decir.
	ErrTransposicionSinCita     = errors.New("bloque de transposicion sin cita del articulo")
	ErrTransposicionSinPaises   = errors.New("bloque de transposicion sin ningun pais")
	ErrPaisSinComprobacion      = errors.New("estado de transposicion sin fecha de comprobacion")
	ErrPaisSinMetodo            = errors.New("estado de transposicion sin decir como se comprobo")
	ErrPaisQueConstaSinNorma    = errors.New("transposicion que consta y no dice cual es la norma")
	ErrPaisQueNoConstaSinApoyo  = errors.New("transposicion que no consta y no dice que vincula mientras")
	ErrFechaDeTransposicionMala = errors.New("fecha del bloque de transposicion ilegible")
	ErrPaisRepetido             = errors.New("el mismo pais dos veces en el bloque de transposicion")
)

// minimoDelMetodoDeComprobacion: un metodo, no una etiqueta.
//
// Mismo suelo y mismo motivo que `SubconjuntoPorque`: un campo libre sin suelo se
// rellena con «ok» o «mirado», y entonces la comprobacion vuelve a ser el booleano
// que no queremos, escrito con letras. La frase mas corta que sirve nombra DONDE
// se miro ("indice de legislacion consolidada del BOE" son 44).
const minimoDelMetodoDeComprobacion = 40

// EstadoDeTransposicion es lo que consta en UN pais, con su fecha y su metodo.
type EstadoDeTransposicion struct {
	// Pais en ISO 3166-1 alfa-2 ("ES").
	Pais string `json:"pais"`
	// Consta dice si la norma nacional de transposicion existe. El valor cero es
	// `false`, que es el permisivo, asi que el linter exige el resto.
	Consta bool `json:"consta"`
	// Norma es el urn del paquete o la referencia de la norma que transpone.
	// Obligatorio cuando Consta.
	Norma string `json:"norma,omitempty"`
	// VinculaMientras es lo que SI obliga hoy en ese pais mientras no haya
	// transposicion. Obligatorio cuando no consta, y admite el descargo
	// explicito de que no vincula nada.
	VinculaMientras string `json:"vincula_mientras,omitempty"`
	// Comprobado es el dia en que alguien fue a mirarlo. AAAA-MM-DD.
	Comprobado string `json:"comprobado"`
	// Como es DONDE se miro. Sin esto, «no consta» es indistinguible de «no se
	// pudo comprobar», que es la distincion que este proyecto lleva persiguiendo
	// desde el invariante 8.
	Como string `json:"como"`
}

// Transposicion es el bloque de una directiva.
type Transposicion struct {
	// Cita es el articulo de la propia directiva que fija el plazo ("art. 41.1").
	Cita string `json:"cita"`
	// LimiteAdopcion y LimiteAplicacion salen de la ficha de Cellar
	// (DIRECTIVE_DATE_TRANSPOSITION) y NO son la misma fecha.
	LimiteAdopcion   string `json:"limite_adopcion,omitempty"`
	LimiteAplicacion string `json:"limite_aplicacion,omitempty"`
	// Estado, por pais. Hoy solo Espana, que es el mercado del producto.
	Estado []EstadoDeTransposicion `json:"estado"`
}

// El vocabulario del ESQUEMA de urn, que no es una norma: la jurisdiccion y el
// tipo de acto. Se nombran como constantes por dos motivos, y el segundo no es
// estetico. El primero, que asi el detector y quien lo prueba dicen lo mismo. El
// segundo, que el invariante 2 prohibe escribir `urn:eu:` en el codigo, y con
// razon: un test que quisiera un urn de directiva tendria que cablear el
// identificador de una norma. Componiendolo de estas piezas, el test habla del
// ESQUEMA y no de NIS2.
const (
	jurisdiccionUE = "eu"
	actoDirectiva  = "dir"
)

// esDirectiva mira el urn, que es lo unico que el paquete declara sobre su propia
// naturaleza. La forma es `urn:<jurisdiccion>:<tipo>:<ano>:<numero>`.
func esDirectiva(urn string) bool {
	partes := strings.Split(strings.ToLower(strings.TrimSpace(urn)), ":")
	return len(partes) >= 3 && partes[1] == jurisdiccionUE && partes[2] == actoDirectiva
}

func (p *Paquete) validarTransposicion(anotar func(error)) {
	dir := esDirectiva(p.URN)
	if p.Transposicion == nil {
		if dir {
			anotar(fmt.Errorf("%w: %s es una directiva (su urn dice `dir`) y no declara el "+
				"bloque `transposicion`.\n"+
				"  Una directiva no vincula por si misma: lo que obliga es la norma nacional "+
				"que la transpone. Sin este bloque ese aviso vuelve a ser un parrafo repetido "+
				"dentro de cada cita, y el dia que el pais transponga hay que editarlas todas "+
				"en vez de cambiar un campo.\n"+
				"  Arreglo: `\"transposicion\": {\"cita\": ..., \"limite_adopcion\": ..., "+
				"\"limite_aplicacion\": ..., \"estado\": [{\"pais\": \"ES\", ...}]}`",
				ErrDirectivaSinTransposicion, p.URN))
		}
		return
	}
	if !dir {
		anotar(fmt.Errorf("%w: %s declara `transposicion` y su urn no dice `dir`.\n"+
			"  Es la direccion cara de las dos: un reglamento con bloque de transposicion le "+
			"dice al cliente que espere a una norma nacional que no va a existir, y eso "+
			"RETRASA un cumplimiento que ya le vincula.\n"+
			"  Arreglo: quitar el bloque, o corregir el urn si de verdad es una directiva",
			ErrTransposicionEnLoQueNoEsDirectiva, p.URN))
		return
	}

	t := p.Transposicion
	if porque := citaVagaPorque(t.Cita); porque != "" {
		anotar(fmt.Errorf("%w: %s, la cita del bloque de transposicion %s. Tiene que nombrar "+
			"el articulo de la directiva que fija el plazo, para poder ir a leerlo",
			ErrTransposicionSinCita, p.URN, porque))
	}
	for _, par := range []struct{ que, valor string }{
		{"limite_adopcion", t.LimiteAdopcion},
		{"limite_aplicacion", t.LimiteAplicacion},
	} {
		if par.valor == "" {
			continue // opcional: hay directivas viejas cuya ficha no lo trae
		}
		if _, err := time.Parse("2006-01-02", par.valor); err != nil {
			anotar(fmt.Errorf("%w: %s, %s = %q no es una fecha AAAA-MM-DD",
				ErrFechaDeTransposicionMala, p.URN, par.que, recortar(par.valor, 20)))
		}
	}
	if len(t.Estado) == 0 {
		anotar(fmt.Errorf("%w: %s declara `transposicion` y no dice de ningun pais si consta "+
			"la norma nacional. Un bloque sin paises no responde la unica pregunta que el "+
			"cliente hace: «¿esto me obliga ya?»", ErrTransposicionSinPaises, p.URN))
	}

	vistos := map[string]bool{}
	for _, e := range t.Estado {
		pais := strings.ToUpper(strings.TrimSpace(e.Pais))
		if vistos[pais] {
			anotar(fmt.Errorf("%w: %s dice dos veces el estado de %q, y las dos pueden "+
				"contradecirse sin que nada lo note", ErrPaisRepetido, p.URN, pais))
		}
		vistos[pais] = true

		if _, err := time.Parse("2006-01-02", strings.TrimSpace(e.Comprobado)); err != nil {
			anotar(fmt.Errorf("%w: %s/%s, `comprobado` = %q.\n"+
				"  «No consta» es una afirmacion sobre el boletin de un pais en un DIA "+
				"concreto, y sin ese dia no se sabe si esta vieja. Es el mismo trato que la "+
				"pista de la transposicion espanola lleva en el censo desde el 02-09-2026",
				ErrPaisSinComprobacion, p.URN, pais, recortar(e.Comprobado, 20)))
		}
		if len(strings.TrimSpace(e.Como)) < minimoDelMetodoDeComprobacion {
			anotar(fmt.Errorf("%w: %s/%s, `como` tiene %d caracteres y el minimo son %d.\n"+
				"  Tiene que decir DONDE se miro. Sin eso, «no consta» es indistinguible de "+
				"«no se pudo comprobar», y confundirlas convierte una maquina sin acceso en "+
				"una que ha verificado",
				ErrPaisSinMetodo, p.URN, pais, len(strings.TrimSpace(e.Como)),
				minimoDelMetodoDeComprobacion))
		}
		if e.Consta && strings.TrimSpace(e.Norma) == "" {
			anotar(fmt.Errorf("%w: %s/%s dice que la transposicion consta y no dice cual es. "+
				"Un cliente que lee «ya te obliga» necesita saber QUE le obliga",
				ErrPaisQueConstaSinNorma, p.URN, pais))
		}
		if !e.Consta && strings.TrimSpace(e.VinculaMientras) == "" {
			anotar(fmt.Errorf("%w: %s/%s dice que la transposicion no consta y no dice que "+
				"vincula mientras tanto.\n"+
				"  `consta: false` es el VALOR CERO de este campo, o sea lo que sale de no "+
				"escribir nada, y es ademas el permisivo: le dice al cliente que todavia no "+
				"le obliga. Por eso no se admite solo (invariante 8).\n"+
				"  Arreglo: `vincula_mientras` con la norma que si le obliga hoy, o con el "+
				"descargo explicito de que no le obliga ninguna",
				ErrPaisQueNoConstaSinApoyo, p.URN, pais))
		}
	}
}
