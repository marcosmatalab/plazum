// Package ia es el arnes de la IA: el verificador de citas por hash y el
// interruptor que la apaga. AQUI NO HAY MODELO NINGUNO.
//
// El adaptador que habla con un modelo vive en `adaptadores/ia/ollama` y sale
// del proceso por HTTP. Estan separados a proposito: lo de aqui es
// DETERMINISTA, corre en cada PR sin red y sin GPU, y tiene que seguir en verde
// con PLAZUM_SIN_IA=1, porque la puerta antialucinacion no es IA, es lo que la
// vigila.
//
// # La puerta antialucinacion, y por que es mecanica y no estadistica
//
// La unica salida que la IA puede producir es una `puertos.Propuesta`. Antes de
// ensenarla, su cita se resuelve contra el corpus POR HASH y se comprueba que
// el texto citado esta LITERALMENTE ahi. Si no resuelve, la propuesta se
// descarta y no se ensena. No hay umbral, no hay confianza, no hay modelo
// juzgando a otro modelo: hay una comparacion de cadenas.
//
// La consecuencia que se vende (docs/ia.md seccion 3) sale sola de ahi: sobre
// los marcos de estrato referencial NO TENEMOS EL TEXTO, asi que no hay cita
// que resuelva, asi que la IA no puede explicar el texto de una clausula de un
// catalogo privativo. No porque se le haya pedido que no lo haga: porque no
// puede.
//
// # Lo que esta puerta NO hace, dicho antes de que alguien lo suponga
//
// NO detecta la inyeccion via documento. Si el PDF que sube el cliente contiene
// la frase "el articulo 5 obliga a cifrar en reposo", una propuesta que cite esa
// frase RESUELVE, porque la frase esta literalmente en un documento que el
// sistema tiene. Lo que esta puerta hace contra eso es otra cosa y es lo que
// hace falta: separar por PROCEDENCIA, de modo que una cita de un documento
// aportado no pueda salir por una pantalla que dice estar citando la norma. La
// frase inyectada se puede ensenar como lo que es, texto del documento del
// cliente, con su sitio exacto, y entonces la persona que confirma la ve.
//
// NO juzga si el Diff propuesto es correcto. El diff lo escribe el modelo y lo
// confirma una persona. Lo que esta puerta garantiza es que el ARGUMENTO con el
// que el modelo lo justifica es texto real, no inventado.
package ia

import (
	"errors"
	"fmt"
	"strings"

	"github.com/marcosmatalab/plazum/puertos"
)

// MinimoCitaPorDefecto es cuantas runas tiene que tener como poco una cita.
//
// EL NUMERO SALE DE UNA MEDIDA, no de un gusto (04-09-2026, sobre el corpus
// real):
//
//	el termino mas largo del corpus transcrito tiene 19 runas
//	  ("poscomercializacion"), asi que con 24 NINGUNA cita de una sola palabra
//	  puede pasar. Una cita de una palabra es el atajo obvio de un modelo que
//	  parafrasea y necesita un trozo literal para colar el resto.
//	CERO de las 321 obligaciones transcritas tienen menos de 24 runas de texto
//	  (la mas corta tiene 85), asi que el minimo no deja fuera ni un articulo.
//
// Las dos direcciones medidas, que es lo que pide una cifra cuyo fallo probable
// seria favorecerte: por arriba no rechaza nada legitimo, por abajo no deja
// pasar una palabra suelta.
const MinimoCitaPorDefecto = 24

// Los centinelas. Hay uno por cada motivo de descarte porque quien llama tiene
// que poder distinguirlos: "no tenemos el texto de este marco" y "el modelo se
// lo ha inventado" son dos cosas que se le dicen distinto a una persona, y
// confundirlas es acusar en falso.
var (
	// De construccion: el valor cero de las opciones.
	ErrSinFuentes             = errors.New("ia: verificador sin fuentes")
	ErrSinProcedencias        = errors.New("ia: verificador sin procedencias admitidas")
	ErrSinMinimoCita          = errors.New("ia: verificador sin minimo de cita")
	ErrClaseDesconocida       = errors.New("ia: fuente con una clase de paquete que no existe")
	ErrCitabilidadIncoherente = errors.New("ia: fuente cuya citabilidad no cuadra con su clase")

	// De verificacion.
	ErrHashAusente           = errors.New("ia: propuesta sin hash de fuente")
	ErrHashIlegible          = errors.New("ia: hash de fuente que no se entiende")
	ErrFuenteNoResuelve      = errors.New("ia: el hash de fuente no existe en el corpus")
	ErrProcedenciaNoAdmitida = errors.New("ia: fuente de una procedencia que este verificador no admite")
	ErrSinTextoCitable       = errors.New("ia: de este marco no tenemos el texto")
	ErrCitaAusente           = errors.New("ia: propuesta sin cita")
	ErrCitaCorta             = errors.New("ia: cita demasiado corta para sostener nada")
	ErrCitaNoAparece         = errors.New("ia: la cita no aparece en la fuente que dice citar")
	// ErrRecorteIncoherente es un fallo NUESTRO, no del modelo: la cita casa
	// pero el trozo de la fuente que se ensenaria no es el que casa.
	ErrRecorteIncoherente = errors.New("ia: el trozo que se ensenaria no es el que se ha verificado")
)

// Opciones configura el verificador.
//
// EL VALOR CERO DE ESTA ESTRUCTURA ESTA PROHIBIDO CAMPO A CAMPO, y aqui esta el
// porque de cada uno, que es lo que el invariante 8 obliga a decir en voz alta:
//
//	Fuentes     nil no es "todas las fuentes", pero tampoco es inocuo. Un
//	            verificador sin fuentes descarta el 100% de lo que le llega, y
//	            eso desde fuera se ve IGUAL que "el modelo esta alucinando
//	            siempre". Confundir "no se pudo comprobar" con "encontro algo"
//	            es el mismo fallo que tuvo el lazo local con gosec. Prohibido.
//	Admite      nil ES el peligroso de manual: en Go un slice a nil se lee como
//	            "sin restriccion", y aqui "sin restriccion" significa que una
//	            cita de un PDF que subio el cliente sale por una pantalla que
//	            dice estar citando la ley. Prohibido.
//	MinimoCita  0 significa "sin minimo", o sea que la cita "de" verifica
//	            contra cualquier articulo. El cero permisivo de manual.
//	            Prohibido.
//
// Y las DOS formas de la nada no son la misma: `Admite: nil` es un error de
// construccion, y `Admite: []Procedencia{}` construye bien y no admite NADA,
// que es lo contrario. Igual que `x509.NewCertPool()` frente a `Roots: nil`.
type Opciones struct {
	Fuentes    []Fuente
	Admite     []Procedencia
	MinimoCita int
}

// Verificador resuelve citas contra fuentes. Es inmutable una vez construido.
type Verificador struct {
	porHash map[string]Fuente
	admite  map[Procedencia]bool
	minimo  int
}

// Estricto es el constructor que hay que usar por defecto: admite SOLO el
// corpus firmado y pone el minimo de cita medido.
//
// Existe para que la configuracion segura sea la corta de escribir. Un
// verificador que admite documentos del cliente se construye con Nuevo y
// diciendolo, que son mas letras a proposito.
func Estricto(fuentes []Fuente) (*Verificador, error) {
	return Nuevo(Opciones{
		Fuentes:    fuentes,
		Admite:     []Procedencia{Corpus},
		MinimoCita: MinimoCitaPorDefecto,
	})
}

// Nuevo construye un verificador. Rechaza las tres formas del valor cero.
func Nuevo(o Opciones) (*Verificador, error) {
	if o.Fuentes == nil {
		return nil, fmt.Errorf("%w. Arreglo: pasa las fuentes contra las que resolver "+
			"(ia.FuentesDelCorpus del corpus cargado). Un verificador sin fuentes descarta "+
			"todo, y eso se lee igual que un modelo que alucina siempre", ErrSinFuentes)
	}
	if o.Admite == nil {
		return nil, fmt.Errorf("%w. Arreglo: di que procedencias admites. Un slice a nil "+
			"en Go se lee como 'sin restriccion', y aqui sin restriccion significa que una "+
			"cita de un documento del cliente sale por una pantalla que dice citar la ley. "+
			"Si de verdad no quieres admitir ninguna, pasa []ia.Procedencia{}, que es otra "+
			"cosa y esa si vale", ErrSinProcedencias)
	}
	if o.MinimoCita <= 0 {
		return nil, fmt.Errorf("%w: %d. Arreglo: usa ia.MinimoCitaPorDefecto (%d runas, "+
			"medido contra el corpus). Sin minimo, la cita \"de\" verifica contra "+
			"cualquier articulo", ErrSinMinimoCita, o.MinimoCita, MinimoCitaPorDefecto)
	}

	v := &Verificador{
		porHash: make(map[string]Fuente, len(o.Fuentes)),
		admite:  map[Procedencia]bool{},
		minimo:  o.MinimoCita,
	}
	for _, p := range o.Admite {
		if !p.Valida() {
			return nil, fmt.Errorf("%w: se ha pedido admitir %q, que no es una procedencia. "+
				"Arreglo: ia.Corpus o ia.Aportado", ErrFuenteSinProcedencia, p)
		}
		v.admite[p] = true
	}
	for _, f := range o.Fuentes {
		if f.ID == "" {
			return nil, fmt.Errorf("%w: una de las fuentes llega sin ID", ErrFuenteSinID)
		}
		if !f.Procedencia.Valida() {
			return nil, fmt.Errorf("%w: %s", ErrFuenteSinProcedencia, f.ID)
		}
		// EL HASH SE RECALCULA AQUI, aunque NuevaFuente ya lo calcule.
		//
		// No es redundante: Fuente es una estructura exportada con campos
		// exportados, asi que se puede construir a mano con un hash que no es
		// el que le corresponde, y entonces el verificador resolveria una cita
		// contra un texto que no es el que el hash dice. Es baratisimo y cierra
		// la unica via por la que el emparejamiento del invariante 7 se podia
		// falsear desde dentro del proceso.
		if HashDe(f.ID, f.Texto) != f.Hash {
			return nil, fmt.Errorf("%w: %s. Arreglo: construye las fuentes con "+
				"ia.NuevaFuente, que lo calcula. Un hash escrito a mano que no cuadra "+
				"convierte el emparejamiento por hash en una mentira permanente",
				ErrFuenteConHashFalso, f.ID)
		}
		// LA CITABILIDAD TIENE QUE CUADRAR CON LA CLASE.
		//
		// ESTO SALIO DE REFUTAR UNA PROPIEDAD, no de leer el diff. El hash
		// recalculado de arriba cierra la mentira sobre el TEXTO, y no cierra
		// la del campo de al lado: una Fuente construida a mano con
		// `Clase: "referencial", Citable: true` pasaba entera, y entonces el
		// texto de un catalogo privativo sale por pantalla como cita. Es la
		// misma forma que el agujero del linter legal, que solo miraba
		// texto_legal mientras el enunciado entraba por cualquiera de los otros
		// veinte campos.
		//
		// Solo se comprueba en las fuentes del CORPUS: un documento que sube el
		// cliente no tiene estrato legal, y lo que lo separa de la norma no es
		// la citabilidad sino la procedencia.
		if f.Procedencia == Corpus {
			c, ok := ClaseDeNombre(f.Clase)
			if !ok {
				return nil, fmt.Errorf("%w: %s dice ser de clase %q, que no existe. "+
					"Arreglo: usa el nombre que da corpus.Clase.String(). Un nombre que no "+
					"se reconoce no puede caer en el valor cero, que es `importado` y SI es "+
					"citable", ErrClaseDesconocida, f.ID, f.Clase)
			}
			if debido := ClaseCitable(c) && f.Texto != ""; f.Citable != debido {
				return nil, fmt.Errorf("%w: %s es de clase %q (citable=%v) y llega con "+
					"Citable=%v. Arreglo: construyela con ia.NuevaFuente y ia.ClaseCitable. "+
					"La frontera legal la decide la clase, no un campo que alguien rellena "+
					"al lado", ErrCitabilidadIncoherente, f.ID, f.Clase, debido, f.Citable)
			}
		}
		// DOS FUENTES BAJO EL MISMO HASH: la segunda tapa a la primera y cual
		// gana lo decide el ORDEN de la lista, que no lo firma nadie
		// (invariante 7). Con el hash de (id, texto) esto solo puede pasar si
		// dos fuentes traen el mismo ID, y entonces el problema esta un piso
		// mas abajo y hay que verlo, no absorberlo.
		if ya, hay := v.porHash[f.Hash]; hay {
			return nil, fmt.Errorf("%w: %s y %s. Arreglo: dos unidades citables no pueden "+
				"compartir identificador, porque entonces cual resuelve una cita lo decide "+
				"el orden de una lista", ErrFuenteRepetida, ya.ID, f.ID)
		}
		if f.normal == "" && f.Texto != "" {
			// Una fuente construida a mano sin pasar por NuevaFuente no trae
			// la forma normalizada ni el mapa, y entonces casar la cita
			// buscaria en una cadena vacia y NO ENCONTRARIA NUNCA NADA. Un
			// verificador que rechaza todo por un fallo de construccion se lee
			// exactamente igual que uno que funciona sobre un modelo malo.
			f.normal, f.mapa = normalizar(f.Texto)
			f.origen = []rune(f.Texto)
		}
		v.porHash[f.Hash] = f
	}
	return v, nil
}

// Fuentes es cuantas fuentes tiene dentro. Existe para poner suelos: un
// verificador vacio descarta todo, igual que uno lleno ante un modelo que
// miente, y desde fuera se ven iguales.
func (v *Verificador) Fuentes() int { return len(v.porHash) }

// Descarte es una propuesta que NO se ensena, con el motivo.
//
// NO LLEVA DENTRO EL TEXTO QUE ESCRIBIO EL MODELO, y es deliberado: estos
// errores acaban en logs y en el bloque copiable de `plazum doctor --issue`. Un
// texto de un modelo alimentado con un documento del cliente puede llevar
// dentro lo que sea, incluida una instruccion dirigida a quien lea el log.
// Aqui salen medidas y motivos, no contenido.
type Descarte struct {
	// Motivo es el centinela, para errors.Is.
	Motivo error
	// Fuente es el ID de la fuente cuando llego a resolverse, y vacio cuando
	// no. No es el hash entero: un hash de 64 caracteres en un mensaje no
	// ayuda a nadie.
	Fuente string
	// Detalle es prosa nuestra, escrita a mano, nunca del modelo.
	Detalle string
}

func (d *Descarte) Error() string {
	if d.Fuente != "" {
		return fmt.Sprintf("%v (fuente %s): %s", d.Motivo, d.Fuente, d.Detalle)
	}
	return fmt.Sprintf("%v: %s", d.Motivo, d.Detalle)
}

func (d *Descarte) Unwrap() error { return d.Motivo }

// Verificada es una propuesta que HA PASADO la puerta.
//
// NO TIENE NI UN CAMPO EXPORTADO, y esa es la mitad del diseno: no hay forma de
// construir una desde fuera de este paquete, asi que "la cita se verifica ANTES
// de ensenarla" deja de ser una convencion que alguien recuerda y pasa a ser
// una propiedad del sistema de tipos. Quien quiera ensenar una propuesta tiene
// que haber pasado por Verificar.
//
// Y lo que devuelve Cita() SALE DE LA FUENTE, no de lo que escribio el modelo.
// Son iguales salvo espacios, porque si no lo fueran esto no existiria; pero
// devolver el trozo de la fuente significa que en la pantalla acaba el texto
// del BOE con su tipografia, no una copia que un modelo tecleo.
type Verificada struct {
	prop   puertos.Propuesta
	fuente Fuente
	desde  int
	hasta  int
}

// Diff es el cambio que propone el modelo. ES DEL MODELO, y por eso lo confirma
// una persona: lo que esta puerta garantiza no es que el diff sea correcto, es
// que el argumento con el que se justifica es texto real.
func (v Verificada) Diff() string { return v.prop.Diff }

// Cita es el trozo de la FUENTE al que la propuesta se agarra, con la
// tipografia y los espacios del original.
func (v Verificada) Cita() string { return string(v.fuente.origen[v.desde:v.hasta]) }

// Desde y Hasta son el sitio exacto de la cita dentro del texto de la fuente,
// en runas. Con ellos la pantalla puede resaltar el trozo dentro del articulo
// entero, que es la diferencia entre ensenar una cita y ensenar donde pone.
func (v Verificada) Desde() int { return v.desde }
func (v Verificada) Hasta() int { return v.hasta }

// Fuente, Marco, Articulo y Clase identifican de donde sale la cita.
func (v Verificada) Fuente() string      { return v.fuente.ID }
func (v Verificada) Marco() string       { return v.fuente.Marco }
func (v Verificada) Articulo() string    { return v.fuente.Articulo }
func (v Verificada) Clase() string       { return v.fuente.Clase }
func (v Verificada) TextoFuente() string { return v.fuente.Texto }

// Procedencia dice si la cita sale del corpus firmado o de un documento del
// cliente. La pantalla TIENE que ensenarlo: una frase de un PDF que subio el
// cliente no se presenta con la misma cara que un articulo del BOE.
func (v Verificada) Procedencia() Procedencia { return v.fuente.Procedencia }

// Modelo y DigestPrompt son la trazabilidad de quien produjo la propuesta.
func (v Verificada) Modelo() string       { return v.prop.Modelo }
func (v Verificada) DigestPrompt() string { return v.prop.DigestPrompt }

// Verificar es LA PUERTA. Devuelve la propuesta verificada, o un *Descarte.
//
// El orden de las comprobaciones no es casual: primero lo que se puede juzgar
// sin resolver nada (el hash tiene forma de hash, la cita existe y no es un
// trozo suelto), y solo despues se resuelve. Asi un hash basura no llega a
// tocar el mapa de fuentes.
func (v *Verificador) Verificar(p puertos.Propuesta) (Verificada, error) {
	// 1. El hash: ausente, presente-y-no-interpretable, o bueno.
	//
	// LAS TRES SON DISTINTAS y esta es la tercera forma de la nada del
	// invariante 8, la de la frontera de ENTRADA. Un hash presente que no se
	// entiende NO es un hash ausente: es un dato que hay y no se entiende, y
	// tomarlo por la nada seria inventarse un valor. Sale error, y uno
	// distinto, siempre.
	if p.HashFuente == "" {
		return Verificada{}, &Descarte{
			Motivo: ErrHashAusente,
			Detalle: "la propuesta no dice de que texto sale. Sin eso no hay nada que " +
				"comprobar, asi que no se ensena.",
		}
	}
	if !pareceHash(p.HashFuente) {
		return Verificada{}, &Descarte{
			Motivo: ErrHashIlegible,
			Detalle: fmt.Sprintf("el hash de fuente tiene %d caracteres y no son 64 "+
				"hexadecimales en minuscula. Un hash presente que no se entiende no es un "+
				"hash que falta, es un dato que hay y no se entiende, y por eso sale error "+
				"y no un descarte silencioso.", len(p.HashFuente)),
		}
	}

	// 2. La cita: ausente, corta, o buena. La longitud se mide en RUNAS, no en
	// bytes: en castellano una cita de 24 caracteres puede ocupar 30 bytes, y
	// contar bytes seria un minimo distinto segun el idioma.
	cita, _ := normalizar(p.Cita)
	if cita == "" {
		return Verificada{}, &Descarte{
			Motivo: ErrCitaAusente,
			Detalle: "la propuesta no trae cita, o su cita son solo espacios. Una propuesta " +
				"sin cita es una afirmacion sin respaldo, que es justo lo que esta puerta " +
				"existe para no ensenar.",
		}
	}
	if n := len([]rune(cita)); n < v.minimo {
		return Verificada{}, &Descarte{
			Motivo: ErrCitaCorta,
			Detalle: fmt.Sprintf("la cita tiene %d runas y el minimo es %d. Una cita mas "+
				"corta que la palabra mas larga del corpus no sostiene nada, y es el atajo "+
				"de un modelo que parafrasea y mete un trozo literal para colar el resto.",
				n, v.minimo),
		}
	}

	// 3. Resolver el hash contra las fuentes.
	f, ok := v.porHash[p.HashFuente]
	if !ok {
		return Verificada{}, &Descarte{
			Motivo: ErrFuenteNoResuelve,
			Detalle: "el hash de fuente no corresponde a ningun texto del corpus. La " +
				"propuesta cita algo que no tenemos, asi que se descarta y no se ensena.",
		}
	}

	// 4. La procedencia. Va DESPUES de resolver porque hasta que no se resuelve
	// no se sabe de donde sale, y va ANTES de mirar el texto porque una cita de
	// una procedencia que este verificador no admite no se ensena aunque case
	// perfectamente.
	if !v.admite[f.Procedencia] {
		return Verificada{}, &Descarte{
			Motivo: ErrProcedenciaNoAdmitida,
			Fuente: f.ID,
			Detalle: fmt.Sprintf("la cita sale de %s y este verificador no admite esa "+
				"procedencia. Una frase de un documento que subio el cliente no puede salir "+
				"por una pantalla que dice estar citando la norma, aunque la frase este "+
				"literalmente en ese documento.", f.Procedencia),
		}
	}

	// 5. La frontera legal. ESTA ES LA CONSECUENCIA MECANICA del apartado 3 de
	// docs/ia.md, y su mensaje es la respuesta que se le da a la persona.
	if !f.Citable {
		return Verificada{}, &Descarte{
			Motivo: ErrSinTextoCitable,
			Fuente: f.ID,
			Detalle: fmt.Sprintf("el marco %s es de estrato %s, o sea que plazum no "+
				"distribuye su texto y por tanto no lo tiene. No se puede citar lo que no "+
				"esta, y no se inventa. Sobre este marco solo se puede trabajar con el "+
				"ritual y la cadencia declarados.", f.Marco, f.Clase),
		}
	}

	// 6. La cita, literal, dentro del texto de la fuente.
	i := strings.Index(f.normal, cita)
	if i < 0 {
		return Verificada{}, &Descarte{
			Motivo:  ErrCitaNoAparece,
			Fuente:  f.ID,
			Detalle: motivoDeNoAparecer(f, cita),
		}
	}
	desdeNormal := len([]rune(f.normal[:i]))
	hastaNormal := desdeNormal + len([]rune(cita))
	desde := f.mapa[desdeNormal]
	hasta := f.mapa[hastaNormal-1] + 1

	// LO QUE SE VA A ENSENAR SE CONTRASTA CONTRA LO QUE SE ACABA DE VERIFICAR.
	//
	// POR QUE ESTA COMPROBACION NO ES REDUNDANTE, y es la unica de este fichero
	// que vigila al propio fichero. Todo lo de arriba comprueba la CITA; lo que
	// acaba en pantalla no es la cita, es el TROZO DE LA FUENTE que sale de
	// recortar por `desde` y `hasta`, y esos dos salen del mapa de
	// normalizacion. Un mapa desplazado una runa produciria un recorte que
	// empieza media palabra antes, y nada de lo anterior lo diria: la cita
	// habria casado igual.
	//
	// Es la forma general del aviso que llego del frente de corpus: una
	// comprobacion que mira la FORMA (aqui, que los indices existen) deja pasar
	// lo que una que mira el CONTENIDO no dejaria. Asi que se mira el
	// contenido: el trozo que se va a ensenar, normalizado, tiene que ser
	// exactamente la cita que se acaba de verificar.
	//
	// Y si no lo es, esto NO es un descarte por culpa del modelo: es un fallo
	// nuestro. Se dice asi, y aun asi no se ensena, porque ensenar un texto que
	// no se ha podido confirmar es lo unico que esta puerta existe para
	// impedir.
	if trozo, _ := normalizar(string(f.origen[desde:hasta])); trozo != cita {
		return Verificada{}, &Descarte{
			Motivo: ErrRecorteIncoherente,
			Fuente: f.ID,
			Detalle: "la cita casa con la fuente pero el trozo que se ensenaria no es ese. " +
				"Es un fallo de plazum, no del modelo, y aun asi no se ensena: un texto " +
				"que no se ha podido confirmar no sale por pantalla.",
		}
	}

	return Verificada{prop: p, fuente: f, desde: desde, hasta: hasta}, nil
}

// Filtrar pasa una tanda por la puerta y devuelve las dos listas por separado.
//
// SON DOS LISTAS Y NO UNA CON UN CAMPO "ok", porque una lista mezclada obliga a
// quien pinta a acordarse de mirar el campo, y ese "acordarse" es exactamente
// lo que esta puerta no puede depender de que ocurra.
func (v *Verificador) Filtrar(ps []puertos.Propuesta) ([]Verificada, []*Descarte) {
	var buenas []Verificada
	var descartes []*Descarte
	for _, p := range ps {
		ok, err := v.Verificar(p)
		if err != nil {
			var d *Descarte
			if errors.As(err, &d) {
				descartes = append(descartes, d)
				continue
			}
			descartes = append(descartes, &Descarte{Motivo: err, Detalle: err.Error()})
			continue
		}
		buenas = append(buenas, ok)
	}
	return buenas, descartes
}

// motivoDeNoAparecer da el motivo concreto cuando la cita no casa. Un "no
// aparece" a secas no dice que hacer; estos si.
func motivoDeNoAparecer(f Fuente, cita string) string {
	if tieneMarcasCombinantes(cita) && !tieneMarcasCombinantes(f.normal) {
		return "la cita no aparece en la fuente, y trae acentos escritos como caracter " +
			"aparte (Unicode NFD) mientras la fuente los lleva precompuestos. No es el " +
			"mismo texto caracter a caracter, asi que no se ensena: aqui no se pliega nada, " +
			"porque plegar seria aceptar como cita algo que el modelo escribio distinto."
	}
	if strings.Contains(strings.ToLower(f.normal), strings.ToLower(cita)) {
		return "la cita aparece en la fuente pero con otras mayusculas. La comparacion es " +
			"literal a proposito, asi que se descarta."
	}
	return "la cita no aparece en el texto de la fuente que dice citar. O el modelo la ha " +
		"parafraseado, o la ha sacado de otro sitio. En los dos casos se descarta y no se " +
		"ensena."
}

// pareceHash dice si una cadena tiene la forma de un sha256 en hexadecimal en
// minuscula. Es la comprobacion de FORMA, no de existencia: separa "no se
// entiende" de "no esta", que son dos descartes distintos.
func pareceHash(s string) bool {
	if len(s) != 64 {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}
