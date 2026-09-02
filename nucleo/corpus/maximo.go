package corpus

import (
	"errors"
	"fmt"
	"strings"
)

// Los centinelas del `maximo`, uno por clase de fallo. Van sueltos y no en un
// error generico porque un centinela por clase es lo que permite a un test
// afirmar QUE fallo y no solo que fallo algo.
var (
	// ErrMaximoSinSuelo: un maximo sin la rama que pone la norma es un plazo
	// con otro nombre.
	ErrMaximoSinSuelo = errors.New("maximo sin suelo legal")
	// ErrMaximoSinDisparador: las dos ramas cuentan desde el mismo hecho.
	ErrMaximoSinDisparador = errors.New("maximo sin disparador")
	// ErrAmpliacionSinExigible es el invariante 8 en esta frontera: el valor
	// cero de un bool es `false`, que aqui es el lado PERMISIVO, asi que no hay
	// defecto y se prohibe explicitamente.
	ErrAmpliacionSinExigible = errors.New("ampliacion sin decir si la norma la exige")
	// ErrExigibleSinAmpliacion: la bandera sobre una rama que no existe.
	ErrExigibleSinAmpliacion = errors.New("ampliacion_exigible sin ampliacion")
)

// EL MAXIMO DE DOS DURACIONES, VISTO DESDE EL PAQUETE.
//
// La primitiva `maximo` compone lo que fija la norma con lo que declara el
// propio obligado, y gana la mayor. El caso que la trae es el art. 13.9 del
// CRA: cada actualizacion de seguridad sigue disponible "un periodo minimo de
// diez anos tras su publicacion, o durante el resto del periodo de soporte si
// este plazo fuera mas largo".
//
// EL MOTOR YA LA SABIA CALCULAR (ventana.Maximo, con sus tres ramas probadas,
// incluida la que importa: una declaracion del obligado MAS CORTA que el suelo
// no acorta el suelo). Lo que no existia era poder decirla desde un
// paquete.json, asi que la primitiva estaba construida y apagada: nadie podia
// encenderla sin tocar codigo, que es justo lo que el invariante 2 no quiere.
//
// Este fichero es la frontera de entrada de esa primitiva, y por eso lo unico
// que hace es negarse a adivinar.
func (p *Paquete) validarMaximo(anotar func(error)) {
	for _, o := range p.Obligaciones {
		t := o.Temporalidad
		if t == nil {
			continue
		}
		suelo := strings.TrimSpace(t.Suelo)
		amp := strings.TrimSpace(t.Ampliacion)

		// Los campos de una primitiva en otra los caza validarCamposDePrimitiva,
		// que mira los cinco de las dos primitivas cableadas a la vez.
		if t.Primitiva != "maximo" {
			continue
		}

		// UN MAXIMO SIN SUELO NO ES UN MAXIMO. El suelo es la rama que pone la
		// norma; sin ella no hay nada que componer con la declaracion del
		// obligado, y lo que quedaria es un plazo escrito con otro nombre.
		//
		// "indeterminado" SI vale, y es la respuesta honesta cuando la norma
		// obliga a conservar y no dice cuanto: el motor mide el tiempo
		// transcurrido en vez de inventarse una fecha (D-17).
		if suelo == "" {
			anotar(fmt.Errorf("%w: obligacion %s es un `maximo` y no declara `suelo`. El suelo es "+
				"la duracion que pone la NORMA, y sin ella no hay maximo que calcular: lo que "+
				"queda es un plazo con otro nombre. Si la norma obliga a conservar y no dice "+
				"cuanto, escribe `\"suelo\": \"indeterminado\"`, que es una respuesta y sale como "+
				"sin plazo legal", ErrMaximoSinSuelo, o.ID))
		}

		// EL DISPARADOR, porque las dos ramas corren desde el MISMO hecho.
		if strings.TrimSpace(t.Disparador["hecho"]) == "" {
			anotar(fmt.Errorf("%w: obligacion %s es un `maximo` y no declara disparador. Las dos "+
				"ramas (el suelo legal y la ampliacion declarada) cuentan desde el mismo hecho, "+
				"asi que sin el no arranca ninguna", ErrMaximoSinDisparador, o.ID))
		}

		// EL VALOR CERO PROHIBIDO EXPLICITAMENTE (invariante 8).
		//
		// `ampliacion_exigible` es un puntero y su nil NO se interpreta,
		// porque las dos respuestas son opuestas y las dos son plausibles:
		//
		//	true   la ausencia del hecho es "falta un dato que la norma exige",
		//	       y el hito sale PendienteDeHecho con el suelo como NoAntesDe.
		//	false  la ausencia significa que no hay segunda rama y rige el suelo.
		//
		// Y el lado del que se acierta por descuido es el PERMISIVO: el valor
		// cero de un bool es `false`, que colapsa al suelo en silencio y ensena
		// una fecha cerrada donde no la hay. Un obligado que lee esa fecha tira
		// la evidencia creyendo que ya podia. Asi que no hay defecto: o se
		// escribe, o no carga.
		if amp != "" && t.AmpliacionExigible == nil {
			anotar(fmt.Errorf("%w: obligacion %s declara `ampliacion` (%s) y no declara "+
				"`ampliacion_exigible`. No hay valor por defecto y no lo va a haber: `true` "+
				"significa que la norma OBLIGA a declararla, y entonces su ausencia deja el "+
				"hito PENDIENTE DE HECHO con el suelo como cota inferior; `false` significa "+
				"que no obliga, y entonces su ausencia hace regir el suelo como fecha cerrada. "+
				"Son respuestas opuestas y la que sale por descuido (el cero de un bool es "+
				"`false`) es la permisiva: ensena una fecha cerrada que puede ser MAS CORTA "+
				"que la real, y quien la lea tirara la evidencia creyendo que ya podia. "+
				"Escribe cual de las dos dice la norma, con su cita en el campo `cita`",
				ErrAmpliacionSinExigible, o.ID, amp))
		}

		// LA BANDERA SOBRE UNA RAMA QUE NO EXISTE. Es el mismo fallo por el
		// otro lado: `ampliacion_exigible` sin `ampliacion` afirma que se penso
		// en una segunda rama que el paquete no declara.
		if amp == "" && t.AmpliacionExigible != nil {
			anotar(fmt.Errorf("%w: obligacion %s declara `ampliacion_exigible` y no declara "+
				"`ampliacion`. La bandera dice si la norma obliga a declarar la segunda rama, "+
				"y aqui no hay segunda rama: o falta el nombre del hecho que la trae, o sobra "+
				"la bandera", ErrExigibleSinAmpliacion, o.ID))
		}
	}
}
