package main

// LA VUELTA: de un alcance.json a las respuestas guardadas de una cuenta.
//
// # Por que la ida sola no bastaba
//
// El bloque `hechos` de un alcance.json es con perdidas A PROPOSITO: un «no» no
// afirma nada (en este motor la ausencia de un hecho no es su negacion) y una
// respuesta de un paquete que todavia no declara el puente tampoco emite nada.
// Eso esta bien para el motor, y hacia la vuelta IMPOSIBLE: de un fichero con
// solo hechos no se recuperan las respuestas, porque la mitad de ellas no dejo
// rastro. Quien exportara y volviera a importar veria desaparecer todos sus «no»
// sin un aviso en ningun sitio, que es la forma cara de fallar aqui.
//
// Por eso el exportador escribe TAMBIEN el bloque `respuestas`, que es el
// registro de lo contestado, y esta orden lo lee.
//
// # POR QUE CAMPO CASA (invariante 7)
//
// Por `pregunta`, el identificador de la pregunta que declara el paquete y que
// es el mismo que viaja en la direccion de la pantalla y el mismo que guarda el
// almacen. NO por posicion en la lista (nadie firma el orden), NO por `campo`
// (dos preguntas pueden pedir el mismo campo sobre instancias distintas) y NO
// por subcadena de nada.
//
// Ese campo NO ESTA FIRMADO, y se dice: `alcance.json` es un fichero de trabajo
// del operador, no una entrada de ledger ni una pieza del expediente. Lo que
// esta orden promete es que lo que entra vuelve a salir igual y que lo que no
// entra SE CUENTA; no promete nada sobre quien escribio el fichero.
//
// # LOS CINCO DESTINOS DE UNA FILA, y no hay un sexto
//
//	importada        tiene `pregunta` y su `valor` es «si» o «no»
//	repetida         repite, con el MISMO valor, una pregunta que ya salio. En
//	                 la cuenta entra una sola, asi que necesita cubo propio: sin
//	                 el, un fichero con una fila duplicada haria que la ley de
//	                 conservacion no cuadrase y la orden acusaria al producto de
//	                 un fallo que esta en el fichero. Salio escribiendo el cubo
//	                 de la conservacion, no leyendo el codigo.
//	sin pregunta     `pregunta` va vacia. No identifica ninguna pregunta de la
//	                 entrevista, asi que no se puede guardar en la cuenta. Pasa
//	                 con los alcances escritos a mano y con el del demo.
//	con valor        tiene `pregunta` y su `valor` no es un si/no: es un
//	                 atributo CON VALOR (una categoria, un nivel). El formato los
//	                 admite y la entrevista todavia no sabe preguntarlos, asi que
//	                 no caben en el almacen. Se cuenta y se dice, igual que
//	                 `SinPuente` en la ida.
//	desconocida      su `pregunta` no la declara el corpus instalado
//
// Y hay un caso que NO es ninguno de los cuatro y es un ERROR: una fila con
// `pregunta` puesta y `valor` VACIO. Es una respuesta que dice a que pregunta
// contesta y no dice que contesta, o sea un dato roto, no un dato de otra clase.
// Tomarlo por «sin responder» seria inventarse la nada donde hay una fila.

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"sort"
	"strings"

	"github.com/marcosmatalab/plazum/adaptadores/usuarios/alcances"
	"github.com/marcosmatalab/plazum/nucleo/corpus"
	"github.com/marcosmatalab/plazum/nucleo/metrica"
	"github.com/marcosmatalab/plazum/nucleo/pantalla"
	"github.com/marcosmatalab/plazum/superficies/pantallas"
)

// CuentaDeLaImportacion es el cardinal de la vuelta, entero.
//
// LA LEY DE CONSERVACION, otra vez y en la otra direccion: toda fila leida cae
// en exactamente uno de los cubos. Se comprueba con metrica.Cuadra, que es
// aritmetica pura y no deja que un cubo que falte pase por un total que cuadra.
type CuentaDeLaImportacion struct {
	// Leidas son las filas del bloque `respuestas` del fichero.
	Leidas int
	// Importadas son las que han entrado en la cuenta.
	Importadas int
	// SinPregunta son las filas sin identificador de pregunta.
	SinPregunta int
	// ConValor son las de un atributo con valor (una categoria, un nivel).
	//
	// HASTA EL 04-09-2026 ESTE CUBO ERA UNA PERDIDA y hoy es un reparto: el
	// almacen no sabia guardar valores, asi que estas filas se contaban y se
	// tiraban. Ahora ENTRAN en la cuenta, y el cubo sigue existiendo porque
	// decir cuantas de las importadas son de cada forma es informacion, no
	// contabilidad. Suman dentro de Importadas.
	ConValor []string
	// Desconocidas son las que nombran una pregunta que el corpus instalado no
	// declara. NO se descartan en silencio.
	Desconocidas []string
	// Repetidas son filas que repiten, con el MISMO valor, una pregunta que ya
	// habia salido. Entran una sola vez en la cuenta, asi que necesitan cubo
	// propio: sin el, un fichero con una fila duplicada haria que la ley de
	// conservacion no cuadrara y la orden acusaria al producto de un fallo que
	// esta en el fichero.
	Repetidas int
}

// Cubos devuelve el reparto para metrica.Cuadra.
//
// SE DERIVA DE LOS CAMPOS y no se escribe a mano: un motivo que repita un
// cardinal a mano es sospechoso entero, y un cubo escrito dos veces es el que se
// queda viejo.
func (c CuentaDeLaImportacion) Cubos() map[string]int {
	return map[string]int{
		// ConValor NO es un cubo de la particion: sus filas ya estan dentro de
		// Importadas desde que el almacen sabe guardar valores. Contarlo aqui
		// haria que la suma pasara del total y `metrica.Cuadra` acusaria al
		// producto de un fallo que no existe.
		"importadas":         c.Importadas,
		"sin id de pregunta": c.SinPregunta,
		"de una pregunta que este corpus no declara": len(c.Desconocidas),
		"repetidas con el mismo valor":               c.Repetidas,
	}
}

// cmdImportarAlcance es `plazum alcance --importar F --cuenta U`.
func cmdImportarAlcance(fichero, cuenta, datos, dirCorpus string,
	salida, errores io.Writer) int {

	fichero, cuenta = strings.TrimSpace(fichero), strings.TrimSpace(cuenta)
	if cuenta == "" {
		fmt.Fprintln(errores, "falta --cuenta.")
		fmt.Fprintln(errores, "  Las respuestas se guardan POR CUENTA, asi que hay que decir en")
		fmt.Fprintln(errores, "  cual entran. No se inventa una: un alcance sin dueno acabaria")
		fmt.Fprintln(errores, "  en un cajon comun que leeria el siguiente que entrase.")
		return 2
	}

	al, err := cargarAlcance(fichero)
	if err != nil {
		fmt.Fprintln(errores, err)
		return 1
	}

	ps, err := corpus.Cargar(strings.TrimSpace(dirCorpus))
	if err != nil {
		fmt.Fprintf(errores, "el corpus de %s no carga: %v\n", dirCorpus, err)
		return 1
	}
	if len(ps) == 0 {
		fmt.Fprintf(errores, "no hay ni un paquete en %s. Sin corpus no se sabe que preguntas "+
			"existen, asi que no se puede importar nada sin inventarselas.\n", dirCorpus)
		return 1
	}
	// LAS PREGUNTAS QUE EXISTEN SALEN DEL CORPUS INSTALADO, no del fichero: es
	// lo que impide que un alcance.json de otro sitio meta en la cuenta
	// identificadores que ninguna obligacion nombra.
	conocidas := map[string]bool{}
	for _, p := range pantalla.Derivar(ps) {
		for _, q := range p.Preguntas {
			conocidas[q.ID] = true
		}
	}

	nuevas, cuentaImp, err := respuestasDelAlcance(al, conocidas)
	if err != nil {
		fmt.Fprintln(errores, err)
		return 1
	}

	almacen, err := alcances.Abrir(alcances.Opciones{Ruta: alcances.RutaPorDefecto(datos)})
	if err != nil {
		fmt.Fprintln(errores, "el almacen de respuestas no se puede abrir:", err)
		return 1
	}
	// REEMPLAZA Y NO MEZCLA, y se dice en la salida. Mezclar dejaria en la
	// cuenta respuestas de dos alcances distintos sin que nadie pudiera saber
	// cual vino de donde, que es peor que perder unas cuantas: un alcance mitad
	// de un fichero y mitad de otro no es el alcance de nadie.
	if err := almacen.Reemplazar(context.Background(), cuenta, nuevas); err != nil {
		fmt.Fprintln(errores, "las respuestas NO se han guardado:", err)
		return 1
	}

	imprimirCuentaDeLaImportacion(salida, cuentaImp, cuenta, almacen.Ruta())
	return 0
}

// respuestasDelAlcance clasifica el bloque `respuestas` de un alcance.
func respuestasDelAlcance(al alcance, conocidas map[string]bool) (
	map[string]alcances.Contestacion, CuentaDeLaImportacion, error) {

	out := map[string]alcances.Contestacion{}
	var c CuentaDeLaImportacion
	for i, r := range al.Respuestas {
		c.Leidas++
		id := strings.TrimSpace(r.Pregunta)
		valor := strings.TrimSpace(r.Valor)
		if id == "" {
			// No identifica ninguna pregunta de la entrevista. Es legitimo: el
			// formato lo admite y el alcance del demo lo usa.
			c.SinPregunta++
			continue
		}
		if valor == "" {
			// UN DATO ROTO, Y NO UNA CLASE DE DATO. Dice a que pregunta
			// contesta y no dice que contesta. Tomarlo por «sin responder»
			// seria inventarse la nada donde hay una fila.
			return nil, c, fmt.Errorf("la respuesta %d del alcance dice contestar a %q y no "+
				"trae ningun valor.\n"+
				"  Una fila que dice a que pregunta contesta y no dice que contesta esta rota, "+
				"y no es lo mismo que no tener fila.\n"+
				"  Arreglo: quita esa fila del fichero, o ponle su valor (si o no)", i+1, id)
		}
		if !conocidas[id] {
			c.Desconocidas = append(c.Desconocidas, id)
			continue
		}
		// LAS DOS FORMAS, Y LA DE VALOR YA NO SE PIERDE. Si no es «si» ni «no»,
		// es una respuesta con valor: el almacen sabe guardarla desde el
		// 04-09-2026. Se cuenta aparte para poder decir cuantas son, pero entra
		// en la cuenta como cualquier otra.
		var v alcances.Contestacion
		if b, err := alcances.LeerRespuesta(valor); err == nil {
			v = alcances.Booleana(b)
		} else {
			v = alcances.ConValor(valor)
			if err := v.Valida(); err != nil {
				return nil, c, fmt.Errorf("la respuesta %d del alcance contesta a %q con un "+
					"valor que no se puede guardar: %w", i+1, id, err)
			}
			c.ConValor = append(c.ConValor, id)
		}
		anterior, repetida := out[id]
		if repetida && anterior != v {
			// DOS RESPUESTAS DISTINTAS A LA MISMA PREGUNTA NO SE RESUELVEN
			// ELIGIENDO UNA: cual ganara dependeria del orden del fichero, y el
			// orden no lo firma nadie (invariante 7).
			return nil, c, fmt.Errorf("el alcance trae la pregunta %q contestada %q y %q.\n"+
				"  Cual manda dependeria del orden del fichero, y el orden no lo firma nadie.\n"+
				"  Arreglo: deja una sola de las dos filas", id, anterior, v)
		}
		if repetida {
			// La repetida IDENTICA no se cuenta como importada dos veces: en la
			// cuenta entra una sola. Va a su cubo para que la suma cuadre.
			c.Repetidas++
		} else {
			c.Importadas++
		}
		out[id] = v
	}
	sort.Strings(c.ConValor)
	sort.Strings(c.Desconocidas)
	return out, c, nil
}

// imprimirCuentaDeLaImportacion es la mitad que hace honesta a la otra.
func imprimirCuentaDeLaImportacion(w io.Writer, c CuentaDeLaImportacion, cuenta, ruta string) {
	fmt.Fprintf(w, "Respuestas guardadas en la cuenta %q, en %s.\n\n", cuenta, ruta)
	fmt.Fprintln(w, "LA CUENTA, ENTERA")
	fmt.Fprintf(w, "    %3d filas leidas del bloque de respuestas del alcance\n", c.Leidas)
	fmt.Fprintf(w, "    %3d guardadas en la cuenta\n", c.Importadas)
	if c.SinPregunta > 0 {
		fmt.Fprintf(w, "    %3d sin identificador de pregunta, asi que no dicen a que pregunta\n",
			c.SinPregunta)
		fmt.Fprintln(w, "        de la entrevista contestan. Pasa con los alcances escritos a")
		fmt.Fprintln(w, "        mano y con el del demo.")
	}
	if c.Repetidas > 0 {
		fmt.Fprintf(w, "    %3d repetian una pregunta que ya habia salido, con el mismo "+
			"valor,\n", c.Repetidas)
		fmt.Fprintln(w, "        asi que en tu cuenta entra una sola.")
	}
	if len(c.ConValor) > 0 {
		fmt.Fprintf(w, "    de esas, %d son de un atributo CON VALOR (una categoria, un "+
			"nivel).\n", len(c.ConValor))
		fmt.Fprintln(w, "        Entran en tu cuenta igual que las de si/no: el almacen sabe")
		fmt.Fprintln(w, "        guardar valores desde el 04-09-2026. Son:")
		for _, id := range c.ConValor {
			fmt.Fprintf(w, "          %s\n", id)
		}
	}
	if len(c.Desconocidas) > 0 {
		fmt.Fprintf(w, "    %3d con un id de pregunta que este corpus no declara (el fichero es\n",
			len(c.Desconocidas))
		fmt.Fprintln(w, "        de otra instalacion, o le falta un paquete). No se descartan en")
		fmt.Fprintln(w, "        silencio; son:")
		for _, id := range c.Desconocidas {
			fmt.Fprintf(w, "          %s\n", id)
		}
	}
	// LA LEY DE CONSERVACION, IMPRESA Y COMPROBADA CON ARITMETICA. Si no cuadra
	// hay filas que no estan en ningun cubo, o sea respuestas que han
	// desaparecido sin motivo visible.
	if err := metrica.Cuadra(c.Leidas, "las filas del bloque de respuestas", c.Cubos()); err != nil {
		fmt.Fprintf(w, "\n    AVISO: %v\n", err)
		fmt.Fprintln(w, "    Eso es un fallo del producto, no tuyo.")
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "ESTO HA REEMPLAZADO lo que hubiera guardado en esa cuenta, no lo ha")
	fmt.Fprintln(w, "mezclado: un alcance mitad de un fichero y mitad de otro no es el de nadie.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Ahora, en el navegador:")
	fmt.Fprintln(w, "    plazum serve            y entra con esa cuenta; la entrevista sale")
	fmt.Fprintln(w, "                            con estas respuestas puestas.")
}

// consultaDeLaCuenta saca las respuestas GUARDADAS de una cuenta y las devuelve
// en la misma forma que trae la direccion de la pantalla.
//
// Devolverlas como url.Values y no como otra cosa no es pereza: es lo que hace
// que las tres fuentes (--cuenta, --url, --respuestas) pasen por EL MISMO
// exportador. Con un camino propio habria dos clasificadores de respuestas y un
// dia dirian cosas distintas.
func consultaDeLaCuenta(cuenta, datos string, errores io.Writer) (url.Values, int) {
	ruta := alcances.RutaPorDefecto(datos)
	almacen, err := alcances.Abrir(alcances.Opciones{Ruta: ruta})
	if err != nil {
		fmt.Fprintln(errores, "el almacen de respuestas no se puede abrir:", err)
		return nil, 1
	}
	al, err := almacen.De(context.Background(), cuenta)
	if err != nil {
		fmt.Fprintln(errores, "no se pueden leer las respuestas de esa cuenta:", err)
		return nil, 2
	}
	if len(al.Respuestas) == 0 {
		// NO SE ESCRIBE UN ALCANCE VACIO. Un fichero sin hechos deriva menos
		// obligaciones y no lo dice, asi que es peor que no tener fichero.
		fmt.Fprintf(errores, "la cuenta %q no tiene ninguna respuesta guardada en %s.\n",
			cuenta, ruta)
		fmt.Fprintln(errores, "  Entra en plazum, responde la entrevista de /alcance y vuelve a")
		fmt.Fprintln(errores, "  ejecutar esto. Si respondiste con otra cuenta, di cual con --cuenta.")
		return nil, 2
	}
	v := url.Values{}
	for id, r := range al.Respuestas {
		// LA FORMA CON VALOR VIAJA POR SU PROPIA CLAVE, la misma que usa la
		// direccion de la pagina, asi que pasa por el mismo puente que las de
		// `--url` en vez de por un camino paralelo.
		if r.EsValor() {
			v.Set(pantallas.ClaveValor(id), r.Valor)
			continue
		}
		switch r.Booleana {
		case alcances.Si:
			v.Add(pantallas.ParamSi, id)
		case alcances.No:
			v.Add(pantallas.ParamNo, id)
		}
	}
	return v, 0
}
