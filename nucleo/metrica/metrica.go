// Package metrica es la aritmetica de las cifras que plazum PUBLICA, con los
// valores imposibles prohibidos antes de calcular.
//
// # Por que existe
//
// La cobertura de la v1 se equivoco tres veces en dos dias y LAS TRES A FAVOR:
//
//	el numerador sumaba rituales de plazum contra un denominador que no los
//	contaba, asi que escribir relojes nuestros subia la cobertura;
//	el denominador arrastraba dos censos corregidos y no propagados;
//	y dos paquetes aportaban mas arriba que abajo (23 sobre 22, 12 sobre 9),
//	lo que en un agregado sube el total sin que nada lo nombre.
//
// Que un error se equivoque siempre a favor no es casualidad estadistica: es
// sesgo del diseno de la metrica. Y las tres veces la causa fue la misma.
//
// # LA REGLA, y es lo unico que hay que llevarse de este paquete
//
// Un valor imposible NO ES UN DATO MALO: es la prueba de que la unidad de los
// dos lados no es la misma. Un numerador mayor que su denominador no significa
// «hemos escrito de mas», significa que arriba se cuentan obligaciones y abajo
// puntos del censo, o relojes arriba y normas abajo. Por eso no se redondea, no
// se recorta a 100 y no se avisa: se rechaza antes de calcular, y quien lo
// reciba tiene que ir a mirar cual de los dos lados cuenta otra cosa.
//
// # Y el denominador cero JAMAS se trata como uno
//
// Es el mismo invariante 8 en aritmetica: el valor degenerado tiene que caer
// del lado restrictivo. Un denominador cero convertido en uno da un porcentaje
// con forma de dato, y esa forma es justamente lo que hace que nadie vaya a
// comprobarlo. Cero abajo significa «no hay nada medido», que es una noticia, y
// se devuelve como error para que quien publique tenga que decirla con
// palabras.
//
// Vive en nucleo/ y no en internal/ a proposito: el nucleo no puede importar
// internal/ (lo vigila arquitectura_test.go), y las cifras del calendario, que
// son las que ve el cliente, se componen en el nucleo.
package metrica

import (
	"errors"
	"fmt"
)

var (
	// ErrNumeradorMayor: la parte es mayor que el total. NO es un dato malo,
	// es la prueba de que los dos lados no cuentan lo mismo.
	ErrNumeradorMayor = errors.New("la parte es mayor que el total")
	// ErrDenominadorCero: no hay nada medido. Nunca se trata como uno.
	ErrDenominadorCero = errors.New("total cero")
	// ErrNegativo: un cardinal negativo es un error de conteo aguas arriba, y
	// se para aqui en vez de propagarse a un porcentaje.
	ErrNegativo = errors.New("cardinal negativo")
)

// Fraccion es una cifra publicada: una parte sobre un total, con el nombre de
// lo que cuenta cada lado.
//
// LOS NOMBRES NO SON ADORNO. Son lo que hace legible el error cuando las
// unidades no casan, que es el fallo que este paquete existe para cazar:
// «relojes escritos (23) sobre puntos censados (22)» dice donde mirar, y
// «23 sobre 22» no dice nada.
type Fraccion struct {
	Parte      int
	Total      int
	QueEsParte string
	QueEsTotal string
}

// Porcentaje devuelve la cifra, o el motivo por el que no hay cifra.
//
// Se comprueba ANTES de dividir. Calcular primero y validar despues deja pasar
// el caso que importa: un 104,5 % ya calculado invita a redondearlo a 100, que
// es exactamente como se tapa una unidad equivocada.
func (f Fraccion) Porcentaje() (float64, error) {
	if err := f.Validar(); err != nil {
		return 0, err
	}
	return 100 * float64(f.Parte) / float64(f.Total), nil
}

// Validar dice si la fraccion se puede publicar.
func (f Fraccion) Validar() error {
	if f.Parte < 0 || f.Total < 0 {
		return fmt.Errorf("%w: %s = %d, %s = %d. Un conteo negativo viene de un error "+
			"aguas arriba, y se para aqui en vez de convertirse en un porcentaje",
			ErrNegativo, f.nombreParte(), f.Parte, f.nombreTotal(), f.Total)
	}
	if f.Total == 0 {
		return fmt.Errorf("%w: no hay ningun %s que medir, asi que no hay porcentaje. "+
			"NO se trata como uno: un denominador cero convertido en uno da una cifra con "+
			"forma de dato, y esa forma es lo que hace que nadie vaya a comprobarla. "+
			"Se publica «sin denominador, %d %s», que es la verdad",
			ErrDenominadorCero, f.nombreTotal(), f.Parte, f.nombreParte())
	}
	if f.Parte > f.Total {
		return fmt.Errorf("%w: %d %s sobre %d %s.\n"+
			"  Esto NO es «hemos hecho de mas»: es la prueba de que los dos lados no "+
			"cuentan la misma unidad. Las tres veces que esta metrica se equivoco, se "+
			"equivoco a favor y por esto.\n"+
			"  No se recorta a 100 ni se redondea: se va a mirar cual de los dos lados "+
			"cuenta otra cosa, y se arregla ahi",
			ErrNumeradorMayor, f.Parte, f.nombreParte(), f.Total, f.nombreTotal())
	}
	return nil
}

// nombreParte y nombreTotal dan una etiqueta util aunque quien construya la
// fraccion no la haya puesto. Un error sin nombres sigue siendo mejor que
// ninguno, pero se nota que falta.
func (f Fraccion) nombreParte() string {
	if f.QueEsParte == "" {
		return "la parte (SIN NOMBRAR: ponle nombre y el error dira donde mirar)"
	}
	return f.QueEsParte
}

func (f Fraccion) nombreTotal() string {
	if f.QueEsTotal == "" {
		return "el total (SIN NOMBRAR: ponle nombre y el error dira donde mirar)"
	}
	return f.QueEsTotal
}

// Cuadra comprueba una LEY DE CONSERVACION: un total que tiene que ser
// exactamente la suma de sus partes.
//
// Es la otra mitad de lo mismo. Un porcentaje miente cuando los dos lados
// cuentan unidades distintas; un reparto miente cuando algo se pierde entre el
// total y los cubos, y ahi el fallo NO SALE COMO ERROR, SALE COMO SILENCIO:
// nadie echa de menos lo que nunca vio.
//
// Las partes van con nombre por la misma razon que arriba: saber que faltan
// tres no sirve; saber que faltan tres de los que no alcanzan a nadie, si.
func Cuadra(total int, queEsTotal string, partes map[string]int) error {
	if total < 0 {
		return fmt.Errorf("%w: %s = %d", ErrNegativo, queEsTotal, total)
	}
	suma := 0
	for nombre, n := range partes {
		if n < 0 {
			// EL ERROR NOMBRA TAMBIEN EL TODO, y no solo el cubo. Nacio sin
			// nombrarlo y su propio test lo cazo: «el cubo "raro" vale -2» no
			// dice de que reparto se cayo, y con tres repartos en la misma
			// pantalla eso obliga a ir a buscarlo.
			return fmt.Errorf("%w: repartiendo %s, el cubo %q vale %d",
				ErrNegativo, queEsTotal, nombre, n)
		}
		suma += n
	}
	if suma == total {
		return nil
	}
	return fmt.Errorf("la cuenta no cuadra: %s son %d y los cubos suman %d (%+d).\n"+
		"  Un descuadre aqui no se ve en la pantalla: lo que sobra o falta simplemente "+
		"NO SALE, y nadie echa de menos lo que nunca vio.\n"+
		"  Cubos: %v", queEsTotal, total, suma, suma-total, partes)
}
