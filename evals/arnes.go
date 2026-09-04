// Package evals es el arnes de los conjuntos dorados de la IA.
//
// # Que es un eval aqui, y en que se diferencia de un test
//
// Un test comprueba que el codigo hace lo que su autor quiso. Un eval comprueba
// que el ARNES aguanta lo que le manda un modelo, y sus casos se escriben desde
// el ataque, no desde la implementacion. Por eso los casos son DATOS en un JSON
// y no funciones en Go: se anaden sin tocar codigo, igual que una obligacion se
// anade sin tocar el motor. Es el invariante 2 aplicado a los evals.
//
// # El primer conjunto NO NECESITA MODELO, y eso es la mitad de su valor
//
// El verificador de citas es determinista: un sha256 y una comparacion de
// cadenas. Su conjunto dorado corre en CADA PR, sin red, sin GPU y sin dinero,
// y sigue en verde con PLAZUM_SIN_IA=1. Los conjuntos que SI necesitan un
// modelo (extraccion de obligaciones, contradicciones) llegan despues y corren
// en nightly con el modelo fijado, que es lo que `docs/ia.md` promete publicar
// en cada release.
//
// Que el primer eval del proyecto sea el de la puerta antialucinacion no es
// casualidad: es el unico que mide algo que el comprador puede comprobar el
// mismo.
package evals

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"

	"github.com/marcosmatalab/plazum/adaptadores/ia"
	"github.com/marcosmatalab/plazum/nucleo/corpus"
	"github.com/marcosmatalab/plazum/puertos"
)

// Conjunto es un fichero de casos dorados con su corpus de mentira dentro.
//
// Va TODO en el mismo fichero a proposito: un conjunto dorado que depende del
// corpus real cambia de significado cada vez que alguien edita un paquete, y
// entonces un rojo no distingue "el arnes se ha roto" de "el corpus ha
// cambiado". El barrido sobre el corpus real existe y es otra cosa, y esta en
// arnes_test.go.
type Conjunto struct {
	Nombre  string         `json:"nombre"`
	Porque  string         `json:"porque"`
	Fuentes []FuenteDorada `json:"fuentes"`
	Casos   []Caso         `json:"casos"`
}

// FuenteDorada es un texto del conjunto.
//
// NO TIENE CAMPO "citable" A PROPOSITO, y es la decision que mas importa de
// este formato. Si el conjunto pudiera declarar por su cuenta que un paquete
// referencial es citable, el eval estaria midiendo su propia opinion en vez de
// la frontera legal del producto: un caso escrito con `clase: referencial,
// citable: true` pasaria en verde mientras el verificador de verdad lo rechaza,
// o al reves. La citabilidad se DERIVA de la clase con la misma funcion que usa
// el producto (ia.ClaseCitable), asi que el conjunto no puede discrepar.
type FuenteDorada struct {
	ID          string `json:"id"`
	Marco       string `json:"marco"`
	Articulo    string `json:"articulo"`
	Clase       string `json:"clase"`
	Procedencia string `json:"procedencia"`
	Texto       string `json:"texto"`
}

// Caso es una propuesta con el veredicto que el arnes tiene que dar.
type Caso struct {
	ID     string `json:"id"`
	Porque string `json:"porque"`
	Diff   string `json:"diff,omitempty"`
	Cita   string `json:"cita"`
	// HashDe nombra una fuente de este conjunto y el arnes DERIVA su hash.
	//
	// SE NOMBRA LA FUENTE Y NO SE ESCRIBE EL HASH, y es la regla de este
	// repositorio sobre los identificadores: un sha256 tecleado a mano tiene la
	// FORMA de lo verificable, y esa forma es justo lo que hace que nadie vaya
	// a verificarlo. Ademas, escrito a mano, el dia que se cambie una coma del
	// texto de la fuente el caso empezaria a probar "el hash no existe" en vez
	// de lo que dice probar, y seguiria en verde.
	HashDe string `json:"hash_de,omitempty"`
	// HashLiteral es para los casos cuyo ataque ES el hash: vacio, corto, con
	// mayusculas, con caracteres que no son hexadecimales.
	HashLiteral *string `json:"hash_literal,omitempty"`

	Veredicto string `json:"veredicto"`
	Motivo    string `json:"motivo,omitempty"`
}

// Los dos veredictos. No hay un tercero, y "no se sabe" no es una opcion: un
// caso dorado sin veredicto no es un caso, es una nota.
const (
	Aceptada   = "aceptada"
	Descartada = "descartada"
)

// motivos es el vocabulario CERRADO de razones de descarte, y cada una empareja
// con su centinela del arnes.
//
// Es cerrado a proposito: con texto libre, un caso podria decir que espera
// "algun error" y pasar por el motivo equivocado, que es la forma mas silenciosa
// de que un eval mida otra cosa. Aqui, un motivo que no este en esta tabla es
// un error de carga del conjunto, no un caso que se salta.
var motivos = map[string]error{
	"hash_ausente":            ia.ErrHashAusente,
	"hash_ilegible":           ia.ErrHashIlegible,
	"fuente_no_resuelve":      ia.ErrFuenteNoResuelve,
	"procedencia_no_admitida": ia.ErrProcedenciaNoAdmitida,
	"sin_texto_citable":       ia.ErrSinTextoCitable,
	"cita_ausente":            ia.ErrCitaAusente,
	"cita_corta":              ia.ErrCitaCorta,
	"cita_no_aparece":         ia.ErrCitaNoAparece,
}

// MotivosConocidos son las claves de arriba, ordenadas. Sirve para que un test
// pueda exigir que el conjunto dorado las recorra TODAS: un motivo que ningun
// caso ejercita es una rama del verificador que nadie prueba.
func MotivosConocidos() []string {
	out := make([]string, 0, len(motivos))
	for k := range motivos {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

var (
	ErrConjuntoSinCasos    = errors.New("evals: conjunto sin casos")
	ErrConjuntoSinFuentes  = errors.New("evals: conjunto sin fuentes")
	ErrCasoSinVeredicto    = errors.New("evals: caso sin veredicto")
	ErrVeredictoIlegible   = errors.New("evals: veredicto que no se entiende")
	ErrMotivoDesconocido   = errors.New("evals: motivo de descarte que no existe")
	ErrCasoSinMotivo       = errors.New("evals: caso descartado sin decir por que")
	ErrHashAmbiguo         = errors.New("evals: caso que dice el hash de dos formas")
	ErrHashSinDeclarar     = errors.New("evals: caso que no dice de que fuente sale")
	ErrFuenteDesconocida   = errors.New("evals: el caso nombra una fuente que no esta")
	ErrProcedenciaIlegible = errors.New("evals: procedencia que no se entiende")
	ErrClaseIlegible       = errors.New("evals: clase de paquete que no se entiende")
	ErrIDRepetido          = errors.New("evals: dos casos con el mismo identificador")
)

// Cargar lee un conjunto y lo VALIDA ENTERO antes de devolverlo.
//
// Valida aqui y no al ejecutar por un motivo del invariante 8: un campo que
// falta o que no se entiende tiene que parar la carga, nunca tomar un valor por
// defecto. Un caso cuyo veredicto se leyera como "aceptada" porque el campo
// estaba vacio seria un caso que pasa siempre, dentro de un conjunto que dice
// medir la puerta antialucinacion.
func Cargar(ruta string) (*Conjunto, error) {
	b, err := os.ReadFile(ruta) // #nosec G304 -- la ruta la pone el arnes, no un usuario
	if err != nil {
		return nil, fmt.Errorf("leyendo el conjunto dorado: %w", err)
	}
	var c Conjunto
	dec := json.NewDecoder(bytes.NewReader(b))
	// Un campo que el formato no conoce PARA la carga. Sin esto, un caso con
	// "veredito" mal escrito se cargaria con el veredicto vacio y el error
	// llegaria disfrazado de otra cosa.
	dec.DisallowUnknownFields()
	if err := dec.Decode(&c); err != nil {
		return nil, fmt.Errorf("el conjunto dorado %s no se entiende: %w", ruta, err)
	}
	if len(c.Fuentes) == 0 {
		return nil, fmt.Errorf("%w: %s", ErrConjuntoSinFuentes, ruta)
	}
	if len(c.Casos) == 0 {
		return nil, fmt.Errorf("%w: %s. Un conjunto vacio pasa siempre, y su verde se lee "+
			"igual que el de un conjunto entero", ErrConjuntoSinCasos, ruta)
	}
	vistos := map[string]bool{}
	fuentes := map[string]bool{}
	for _, f := range c.Fuentes {
		fuentes[f.ID] = true
	}
	for _, k := range c.Casos {
		if vistos[k.ID] {
			return nil, fmt.Errorf("%w: %s", ErrIDRepetido, k.ID)
		}
		vistos[k.ID] = true
		if err := k.validar(fuentes); err != nil {
			return nil, fmt.Errorf("caso %s: %w", k.ID, err)
		}
	}
	return &c, nil
}

func (k Caso) validar(fuentes map[string]bool) error {
	switch k.Veredicto {
	case "":
		return ErrCasoSinVeredicto
	case Aceptada:
		if k.Motivo != "" {
			return fmt.Errorf("%w: un caso aceptado no lleva motivo de descarte",
				ErrMotivoDesconocido)
		}
	case Descartada:
		if k.Motivo == "" {
			return fmt.Errorf("%w. Un caso que solo exige 'que falle' pasa tambien cuando "+
				"falla por el motivo equivocado, y entonces el eval mide otra cosa",
				ErrCasoSinMotivo)
		}
		if _, ok := motivos[k.Motivo]; !ok {
			return fmt.Errorf("%w: %q. El vocabulario es cerrado: %v",
				ErrMotivoDesconocido, k.Motivo, MotivosConocidos())
		}
	default:
		// PRESENTE Y NO INTERPRETABLE. No se toma por "aceptada" ni por
		// "descartada": las dos serian inventarse el veredicto.
		return fmt.Errorf("%w: %q. Solo hay dos: %q y %q",
			ErrVeredictoIlegible, k.Veredicto, Aceptada, Descartada)
	}

	tieneDe := k.HashDe != ""
	tieneLiteral := k.HashLiteral != nil
	switch {
	case tieneDe && tieneLiteral:
		return ErrHashAmbiguo
	case !tieneDe && !tieneLiteral:
		return fmt.Errorf("%w. Usa hash_de con el id de una fuente, o hash_literal si el "+
			"ataque ES el hash", ErrHashSinDeclarar)
	case tieneDe && !fuentes[k.HashDe]:
		return fmt.Errorf("%w: %q", ErrFuenteDesconocida, k.HashDe)
	}
	return nil
}

// Resultado es lo que sale de ejecutar un caso.
type Resultado struct {
	Caso      Caso
	Veredicto string
	Motivo    string
	// Cita es lo que el verificador dejaria ensenar, y sale de la FUENTE. Se
	// devuelve para que un caso aceptado pueda comprobar que lo que se ensena
	// es texto real y no lo que escribio el modelo.
	Cita string
	// Detalle es el error tal cual, para el mensaje del fallo.
	Detalle error
}

// Paso dice si el resultado es el que el caso esperaba.
func (r Resultado) Paso() bool {
	if r.Veredicto != r.Caso.Veredicto {
		return false
	}
	return r.Veredicto == Aceptada || r.Motivo == r.Caso.Motivo
}

// Ejecutar corre el conjunto entero contra el verificador ESTRICTO, que es el
// que usa el producto: solo admite el corpus firmado.
//
// Los casos que hablan de documentos aportados van con su fuente marcada como
// tal y esperan `procedencia_no_admitida`, que es justo lo que tiene que pasar
// en el camino de una pantalla que dice citar la ley.
func Ejecutar(c *Conjunto) ([]Resultado, error) {
	var fuentes []ia.Fuente
	porID := map[string]ia.Fuente{}
	for _, f := range c.Fuentes {
		p, err := procedenciaDe(f.Procedencia)
		if err != nil {
			return nil, fmt.Errorf("fuente %s: %w", f.ID, err)
		}
		// LA CITABILIDAD SALE DE LA CLASE, Y LA CLASE SOLO EXISTE PARA EL
		// CORPUS. Un documento que sube el cliente no tiene estrato legal: no
		// hay un tercero con derechos sobre el, es suyo, y por eso su texto SI
		// se le puede ensenar. Lo que lo separa de la norma no es la
		// citabilidad, es la PROCEDENCIA, y esa es la que mira el verificador
		// estricto. Confundir las dos seria dejar la puerta abierta por el
		// campo de al lado.
		etiqueta, citable := "aportado", true
		if p == ia.Corpus {
			clase, err := ClaseDe(f.Clase)
			if err != nil {
				return nil, fmt.Errorf("fuente %s: %w", f.ID, err)
			}
			etiqueta, citable = f.Clase, ia.ClaseCitable(clase)
		} else if f.Clase != "" {
			return nil, fmt.Errorf("fuente %s: %w: un documento aportado no tiene clase del "+
				"corpus, y ponersela seria decidir su frontera legal a mano", f.ID, ErrClaseIlegible)
		}
		nueva, err := ia.NuevaFuente(f.ID, f.Marco, f.Articulo, etiqueta, p, citable, f.Texto)
		if err != nil {
			return nil, fmt.Errorf("fuente %s: %w", f.ID, err)
		}
		fuentes = append(fuentes, nueva)
		porID[f.ID] = nueva
	}
	v, err := ia.Estricto(fuentes)
	if err != nil {
		return nil, err
	}

	out := make([]Resultado, 0, len(c.Casos))
	for _, k := range c.Casos {
		hash := ""
		if k.HashLiteral != nil {
			hash = *k.HashLiteral
		} else {
			hash = porID[k.HashDe].Hash
		}
		ok, err := v.Verificar(puertos.Propuesta{
			Diff:       k.Diff,
			Cita:       k.Cita,
			HashFuente: hash,
			Modelo:     "sin modelo: este conjunto no llama a ninguno",
		})
		r := Resultado{Caso: k, Detalle: err}
		if err == nil {
			r.Veredicto = Aceptada
			r.Cita = ok.Cita()
		} else {
			r.Veredicto = Descartada
			r.Motivo = nombreDelMotivo(err)
		}
		out = append(out, r)
	}
	return out, nil
}

// ClaseDe traduce el nombre de una clase a la clase del corpus.
//
// LA TABLA NO ESTA ESCRITA AQUI: se compone recorriendo las clases que declara
// el nucleo y pidiendoles su nombre. Escrita a mano, seria una segunda lista, y
// una segunda lista es una lista que se queda vieja: el dia que entrara una
// clase nueva, el conjunto dorado no podria nombrarla y nadie sabria por que.
//
// Un nombre que no se reconoce es ERROR y no una clase por defecto: tomar
// "referncial" mal escrito por `importado` (el valor cero de corpus.Clase)
// convertiria un caso que prueba la frontera legal en un caso que la ignora, y
// seguiria en verde. Es la tercera forma de la nada, en la frontera de entrada.
func ClaseDe(s string) (corpus.Clase, error) {
	for c := corpus.Clase(0); c <= corpus.Propio; c++ {
		if c.String() == s {
			return c, nil
		}
	}
	var nombres []string
	for c := corpus.Clase(0); c <= corpus.Propio; c++ {
		nombres = append(nombres, c.String())
	}
	return 0, fmt.Errorf("%w: %q. Las clases son %v", ErrClaseIlegible, s, nombres)
}

func procedenciaDe(s string) (ia.Procedencia, error) {
	switch s {
	case "corpus":
		return ia.Corpus, nil
	case "aportado":
		return ia.Aportado, nil
	}
	// Incluido el vacio: una fuente que no dice de donde viene no entra.
	return ia.Ninguna, fmt.Errorf("%w: %q. Solo hay dos: \"corpus\" y \"aportado\"",
		ErrProcedenciaIlegible, s)
}

// nombreDelMotivo traduce un error del arnes a la clave del vocabulario.
//
// Devuelve "" cuando no reconoce el error, y eso hace fallar al caso en vez de
// pasarlo: un motivo nuevo del verificador que este arnes no conozca tiene que
// salir a la luz, no colarse como "descartada y ya".
func nombreDelMotivo(err error) string {
	for nombre, centinela := range motivos {
		if errors.Is(err, centinela) {
			return nombre
		}
	}
	return ""
}
