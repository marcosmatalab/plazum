package corpus

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

var (
	// ErrPreavisoSinEfecto: sin la fecha que elige el obligado no hay cuenta
	// atras que hacer.
	ErrPreavisoSinEfecto = errors.New("preaviso sin fecha de efecto")
	// ErrPreavisoSinAntelacion: sin antelacion no hay preaviso, hay un aviso.
	ErrPreavisoSinAntelacion = errors.New("preaviso sin antelacion")
	// ErrPreavisoConDisparador: el preaviso no cuenta desde un hecho que ocurre.
	ErrPreavisoConDisparador = errors.New("preaviso con disparador")
	// ErrCampoDePrimitivaFueraDeSitio: un campo que no lee nadie y que ademas
	// afirma que alguien decidio algo. Familia de la directiva inerte.
	ErrCampoDePrimitivaFueraDeSitio = errors.New("campo de una primitiva declarado en otra")
)

// camposPropios dice que campos de Temporalidad lee CADA primitiva. Es la tabla
// que hace posible la comprobacion de abajo, y hay que anadirle una linea cada
// vez que se cablea una primitiva nueva.
var camposPropios = map[string][]string{
	"maximo":   {"suelo", "ampliacion", "ampliacion_exigible"},
	"preaviso": {"efecto", "antelacion"},
}

// LOS CAMPOS DE UNA PRIMITIVA NO VIVEN EN OTRA.
//
// Un `suelo` sobre una `periodica` no hace nada, y ese es el problema menor. El
// mayor es que AFIRMA que alguien penso en el minimo legal de esa obligacion:
// quien lo lea dara por decidido lo que nadie decidio. Es la familia de la
// directiva inerte (`//nolint:` para una herramienta que no corre) escrita en un
// campo de datos, y se cierra igual: o el campo hace algo, o sale.
//
// La comprobacion es de ida y vuelta a proposito. Que un `preaviso` no lleve
// `efecto` lo caza el validador de abajo; que una `periodica` SI lo lleve lo
// caza este, y son dos descuidos distintos: el primero es un olvido, el segundo
// es casi siempre un copiar-pegar de otra obligacion.
func (p *Paquete) validarCamposDePrimitiva(anotar func(error)) {
	for _, o := range p.Obligaciones {
		t := o.Temporalidad
		if t == nil {
			continue
		}
		presentes := map[string]string{
			"suelo":               strings.TrimSpace(t.Suelo),
			"ampliacion":          strings.TrimSpace(t.Ampliacion),
			"efecto":              strings.TrimSpace(t.Efecto),
			"antelacion":          strings.TrimSpace(t.Antelacion),
			"ampliacion_exigible": "",
		}
		if t.AmpliacionExigible != nil {
			presentes["ampliacion_exigible"] = "declarado"
		}
		suyos := map[string]bool{}
		for _, c := range camposPropios[t.Primitiva] {
			suyos[c] = true
		}
		var intrusos []string
		for campo, valor := range presentes {
			if valor != "" && !suyos[campo] {
				intrusos = append(intrusos, campo)
			}
		}
		if len(intrusos) == 0 {
			continue
		}
		sort.Strings(intrusos) // el mensaje tiene que ser estable entre ejecuciones
		anotar(fmt.Errorf("%w: obligacion %s es una %q y declara %v. Esos campos los lee otra "+
			"primitiva: aqui no hacen nada y hacen creer que alguien decidio algo. Arreglo: "+
			"quitalos, o cambia la primitiva si de verdad es la otra",
			ErrCampoDePrimitivaFueraDeSitio, o.ID, t.Primitiva, intrusos))
	}
}

// EL PLAZO QUE CORRE HACIA ATRAS, VISTO DESDE EL PAQUETE.
//
// Es la familia G del censo, siete relojes, y la primitiva llevaba construida y
// probada desde antes con sus dorados. Estaba APAGADA para el corpus, como
// `maximo`, y por el mismo motivo: nadie habia aplicado el invariante 2 a las
// capacidades del motor. La pregunta que lo destapa es fija desde el 02-09-2026
// en la pasada 1: por cada primitiva, ¿puede un paquete usarla sin tocar codigo?
func (p *Paquete) validarPreaviso(anotar func(error)) {
	for _, o := range p.Obligaciones {
		t := o.Temporalidad
		if t == nil || t.Primitiva != "preaviso" {
			continue
		}
		if strings.TrimSpace(t.Efecto) == "" {
			anotar(fmt.Errorf("%w: obligacion %s es un `preaviso` y no declara `efecto`. El "+
				"efecto es el nombre del hecho que trae la fecha que ELIGE el obligado, y sin "+
				"ella no hay cuenta atras que hacer: un preaviso no cuenta desde algo que pasa, "+
				"cuenta hacia atras desde algo que se decide", ErrPreavisoSinEfecto, o.ID))
		}
		if strings.TrimSpace(t.Antelacion) == "" {
			anotar(fmt.Errorf("%w: obligacion %s es un `preaviso` y no declara `antelacion`. Sin "+
				"antelacion no hay preaviso, hay un aviso. Si la norma exige avisar antes y NO "+
				"dice cuanto, se escribe `\"antelacion\": \"indeterminado\"`, que es una "+
				"respuesta y sale como sin plazo legal (D-17)",
				ErrPreavisoSinAntelacion, o.ID))
		}
		// EL DISPARADOR SOBRA, Y NO ES UN DETALLE DE ESTILO. Si un preaviso
		// declara disparador, quien lo lea entendera que cuenta desde un hecho
		// que le ocurre al obligado, que es lo contrario de lo que hace. El
		// motor ni lo mira, asi que el campo seria una afirmacion falsa que no
		// cambia el calculo: la peor combinacion.
		if strings.TrimSpace(t.Disparador["hecho"]) != "" {
			anotar(fmt.Errorf("%w: obligacion %s es un `preaviso` y declara disparador (%q). Un "+
				"preaviso NO cuenta desde un hecho que le ocurre al obligado: cuenta hacia atras "+
				"desde la fecha que el obligado elige, que va en `efecto`. El motor no mira el "+
				"disparador aqui, asi que dejarlo es escribir algo que no es verdad y que ademas "+
				"no cambia el resultado", ErrPreavisoConDisparador, o.ID, t.Disparador["hecho"]))
		}
	}
}
