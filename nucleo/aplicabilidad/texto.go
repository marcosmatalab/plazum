package aplicabilidad

import (
	"errors"
	"fmt"
	"strings"
)

// La sintaxis de superficie del dialecto: como se escribe una regla en un
// paquete de corpus.
//
// Por que existe. El motor de Datalog ya estaba, pero solo se podia programar
// desde Go, construyendo Atomo a Atomo. Eso hacia FALSO el invariante 2 del
// proyecto: las reglas de aplicabilidad del ENS vivian en un fichero Go de test
// llamado progENS, no en paquetes/ens. Y con las reglas en codigo, actualizar el
// corpus es una release del binario en vez de un fichero de datos firmado, que
// es justo lo que la suscripcion del corpus y el canal consultor necesitan que
// NO sea. Con dos paquetes autorizados esto es una tarde; con doce es una
// migracion.
//
// Como se escribe:
//
//	aplica(x.art31.auditoria, S) :- categoria(S, "MEDIA")
//	aplica(x.art34, E) :- responsable(E), not exento(E)
//	nivel_max(S, _AGG) :- maneja(S, I), nivel_dimension(I, _, N)
//
// Las reglas del lexico, que son tres y no admiten excepcion:
//
//	Variable   empieza por mayuscula o por guion bajo: S, E, Nivel, _AGG.
//	Anonima    el guion bajo solo: _. Significa "no me importa el valor".
//	Constante  todo lo demas: en minuscula sin comillas (x.art31.auditoria),
//	           o entre comillas cuando lleva mayusculas o espacios ("MEDIA").
//
// Y la trampa que esa convencion trae puesta, por la que hay un guardia abajo:
// quien escriba categoria(S, MEDIA) sin comillas NO esta escribiendo la
// constante MEDIA, esta declarando una variable nueva que unifica con
// cualquier categoria. La regla no falla: deriva de mas, en silencio. Por eso
// una variable que aparece una sola vez en la regla es un ERROR y hay que
// escribir _ si de verdad no importa. Es el mismo fallo que ya nos costo una
// revision con la variable anonima, escrito una vez y cazado para siempre.
//
// Que NO va en la sintaxis: el agregado, la escala y la cita. Van como campos
// del fichero de datos, al lado de la regla. Un agregado con su escala es una
// lista ordenada, y una lista ordenada se escribe mejor en JSON que en una
// gramatica que habria que fuzzear entera para nada.

// Limites del parser. Un paquete de corpus lo aporta un tercero, asi que la
// entrada es hostil por definicion. Estos topes no son de rendimiento: son para
// que un fichero de datos no pueda hacer trabajar al parser sin limite.
const (
	maxLongitudRegla = 4096
	maxAtomos        = 32
	maxArgumentos    = 16
)

// Errores del parser, como centinelas: un test que quiera comprobar que se
// rechaza una regla sin cuerpo tiene que poder comprobarlo con errors.Is y no
// buscando una subcadena en un mensaje.
var (
	ErrReglaVacia      = errors.New("regla vacia")
	ErrSinCuerpo       = errors.New("regla sin cuerpo")
	ErrSintaxis        = errors.New("error de sintaxis")
	ErrDemasiadoLarga  = errors.New("regla demasiado larga")
	ErrVariableUnica   = errors.New("variable que aparece una sola vez")
	ErrPredicadoInvado = errors.New("nombre de predicado invalido")
)

// ParsearRegla lee una regla escrita en la sintaxis de superficie.
//
// Devuelve la Regla SIN ID, SIN cita y SIN agregado: esos tres los pone quien
// llama, desde los campos del fichero de datos. Aqui solo se traduce el texto.
func ParsearRegla(s string) (Regla, error) {
	if len(s) > maxLongitudRegla {
		return Regla{}, fmt.Errorf("%w: %d caracteres, el tope son %d. Una regla tan larga "+
			"casi siempre son varias reglas encadenadas; partela y dale un id a cada una",
			ErrDemasiadoLarga, len(s), maxLongitudRegla)
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return Regla{}, fmt.Errorf("%w: escribe cabeza(...) :- cuerpo", ErrReglaVacia)
	}

	cabezaTxt, cuerpoTxt, hayFlecha := cortarFlecha(s)
	if !hayFlecha {
		return Regla{}, fmt.Errorf("%w: %q no tiene :-. Un paquete declara REGLAS, no hechos: "+
			"los hechos los aporta el sujeto obligado al describir su alcance, y un paquete "+
			"que afirma hechos sobre el sujeto esta afirmando algo que no puede saber",
			ErrSinCuerpo, s)
	}

	cabeza, err := parsearAtomo(cabezaTxt)
	if err != nil {
		return Regla{}, fmt.Errorf("en la cabeza: %w", err)
	}

	var r Regla
	r.Cabeza = cabeza
	for _, lit := range partirLiterales(cuerpoTxt) {
		lit = strings.TrimSpace(lit)
		if lit == "" {
			return Regla{}, fmt.Errorf("%w: hay una coma de mas en el cuerpo de %q",
				ErrSintaxis, s)
		}
		negado := false
		if resto, ok := quitarPrefijoPalabra(lit, "not"); ok {
			negado, lit = true, resto
		}
		a, err := parsearAtomo(lit)
		if err != nil {
			return Regla{}, fmt.Errorf("en el cuerpo: %w", err)
		}
		if negado {
			r.Negados = append(r.Negados, a)
		} else {
			r.Cuerpo = append(r.Cuerpo, a)
		}
	}
	if len(r.Cuerpo) == 0 {
		return Regla{}, fmt.Errorf("%w: %q solo tiene literales negados. Una regla que solo "+
			"niega no deriva nada y ademas no es segura: sus variables no quedan ligadas",
			ErrSinCuerpo, s)
	}
	if n := len(r.Cuerpo) + len(r.Negados); n > maxAtomos {
		return Regla{}, fmt.Errorf("%w: %d atomos en el cuerpo, el tope son %d",
			ErrDemasiadoLarga, n, maxAtomos)
	}
	if err := comprobarVariablesUnicas(r, s); err != nil {
		return Regla{}, err
	}
	// El tope se comprueba tambien sobre la forma CANONICA, no solo sobre lo
	// que llego. HALLAZGO DEL FUZZING (el cuarto): una regla de 4090 caracteres
	// escrita apretada se reescribe con espacios detras de las comas y pasa de
	// 4096, o sea que el parser aceptaba reglas que su propio escritor no podia
	// reemitir. Eso rompe `dutiq explain` justo en las reglas mas grandes, que
	// son las que mas falta hace explicar. Comprobandolo aqui, lo que entra se
	// puede volver a escribir y a leer siempre.
	if n := len(r.Escribir()); n > maxLongitudRegla {
		return Regla{}, fmt.Errorf("%w: escrita en forma canonica ocupa %d caracteres y el "+
			"tope son %d. Partela en varias reglas encadenadas, cada una con su id",
			ErrDemasiadoLarga, n, maxLongitudRegla)
	}
	return r, nil
}

// cortarFlecha parte por el PRIMER :- que no este dentro de comillas.
func cortarFlecha(s string) (cabeza, cuerpo string, ok bool) {
	dentro := false
	for i := 0; i+1 < len(s); i++ {
		if s[i] == '"' {
			dentro = !dentro
			continue
		}
		if !dentro && s[i] == ':' && s[i+1] == '-' {
			return s[:i], s[i+2:], true
		}
	}
	return "", "", false
}

// partirLiterales parte por comas de nivel cero: ni dentro de parentesis ni
// dentro de comillas. Partir por "," a secas rompe cualquier atomo con dos
// argumentos, que son casi todos.
func partirLiterales(s string) []string {
	var out []string
	prof, dentro, ini := 0, false, 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '"':
			dentro = !dentro
		case '(':
			if !dentro {
				prof++
			}
		case ')':
			if !dentro {
				prof--
			}
		case ',':
			if !dentro && prof == 0 {
				out = append(out, s[ini:i])
				ini = i + 1
			}
		}
	}
	return append(out, s[ini:])
}

// quitarPrefijoPalabra quita "not " exigiendo que sea palabra entera, para que
// un predicado llamado notificar no se lea como la negacion de "ificar".
func quitarPrefijoPalabra(s, pal string) (string, bool) {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, pal) || len(s) == len(pal) {
		return s, false
	}
	// Cualquier espacio en blanco, no solo espacio y tabulador. Si aqui solo
	// se aceptaran esos dos, not seguido de un retorno de carro no seria una
	// negacion, pero parsearAtomo si leeria un predicado llamado not, y el
	// mismo texto tendria dos lecturas distintas. Lo encontro el fuzzing.
	c := s[len(pal)]
	if !strings.ContainsRune(" \t\n\v\f\r", rune(c)) && c != '(' {
		return s, false
	}
	return strings.TrimSpace(s[len(pal):]), true
}

func parsearAtomo(s string) (Atomo, error) {
	s = strings.TrimSpace(s)
	i := strings.IndexByte(s, '(')
	if i < 0 || !strings.HasSuffix(s, ")") {
		return Atomo{}, fmt.Errorf("%w: %q no tiene la forma predicado(argumentos)", ErrSintaxis, s)
	}
	pred := strings.TrimSpace(s[:i])
	if !predicadoValido(pred) {
		return Atomo{}, fmt.Errorf("%w: %q. Un predicado va en minuscula y admite . y _ "+
			"(categoria, nivel_max, x.en_ambito)", ErrPredicadoInvado, pred)
	}
	dentro := strings.TrimSpace(s[i+1 : len(s)-1])
	if dentro == "" {
		return Atomo{}, fmt.Errorf("%w: %s() no tiene argumentos. Un predicado de cero "+
			"argumentos es una bandera global, y la aplicabilidad siempre es de ALGUIEN",
			ErrSintaxis, pred)
	}
	partes := partirLiterales(dentro)
	if len(partes) > maxArgumentos {
		return Atomo{}, fmt.Errorf("%w: %s tiene %d argumentos, el tope son %d",
			ErrDemasiadoLarga, pred, len(partes), maxArgumentos)
	}
	a := Atomo{Pred: pred}
	for _, p := range partes {
		t, err := parsearTermino(strings.TrimSpace(p))
		if err != nil {
			return Atomo{}, fmt.Errorf("en %s: %w", pred, err)
		}
		a.Args = append(a.Args, t)
	}
	return a, nil
}

func parsearTermino(s string) (Termino, error) {
	if s == "" {
		return Termino{}, fmt.Errorf("%w: argumento vacio", ErrSintaxis)
	}
	if s[0] == '"' {
		if len(s) < 2 || s[len(s)-1] != '"' {
			return Termino{}, fmt.Errorf("%w: %q abre comillas y no las cierra", ErrSintaxis, s)
		}
		v := s[1 : len(s)-1]
		if strings.ContainsAny(v, "\"\n\r") {
			return Termino{}, fmt.Errorf("%w: %q lleva comillas o saltos de linea dentro. "+
				"El dialecto no tiene escapes a proposito: si un valor los necesita, "+
				"no es un identificador y no va en una regla", ErrSintaxis, s)
		}
		return C(v), nil
	}
	if s[0] == '_' || (s[0] >= 'A' && s[0] <= 'Z') {
		if !identificadorValido(s) {
			return Termino{}, fmt.Errorf("%w: la variable %q lleva caracteres que no son "+
				"letras, digitos ni guion bajo", ErrSintaxis, s)
		}
		return Termino{Var: true, Val: s}, nil
	}
	if !constanteValida(s) {
		return Termino{}, fmt.Errorf("%w: %q no es una constante valida. Si lleva mayusculas "+
			"o espacios, entrecomillala: \"%s\"", ErrSintaxis, s, s)
	}
	return C(s), nil
}

// comprobarVariablesUnicas es el guardia contra la trampa de la convencion.
//
// categoria(S, MEDIA) sin comillas no dice lo que parece: MEDIA es una variable
// nueva que unifica con cualquier valor, y la regla deriva de mas en silencio.
// Una variable que aparece una sola vez en toda la regla casi siempre es eso, y
// cuando de verdad no importa el valor se escribe _, que es explicito.
func comprobarVariablesUnicas(r Regla, origen string) error {
	cuenta := map[string]int{}
	ver := func(as []Atomo) {
		for _, a := range as {
			for _, t := range a.Args {
				if t.Var && !t.esAnonima() {
					cuenta[t.Val]++
				}
			}
		}
	}
	ver([]Atomo{r.Cabeza})
	ver(r.Cuerpo)
	ver(r.Negados)
	for _, a := range append(append([]Atomo{r.Cabeza}, r.Cuerpo...), r.Negados...) {
		for _, t := range a.Args {
			if !t.Var || t.esAnonima() || cuenta[t.Val] != 1 || t.Val == "_AGG" {
				continue
			}
			return fmt.Errorf("%w: %s aparece una sola vez en %q. Si no te importa el valor, "+
				"escribe _. Y si querias la constante %s, entrecomillala (\"%s\"): sin "+
				"comillas es una variable y la regla deriva de mas sin avisar",
				ErrVariableUnica, t.Val, strings.TrimSpace(origen), t.Val, t.Val)
		}
	}
	return nil
}

// palabrasReservadas del dialecto. Solo hay una, y esta aqui por un hallazgo
// del fuzzing: not se aceptaba como nombre de predicado, asi que el mismo
// texto se leia de dos maneras segun por donde entrara. Un predicado que se
// llama como la palabra de la negacion no es un caso borde teorico: es una
// regla que significa una cosa al escribirla y otra al releerla.
var palabrasReservadas = map[string]bool{"not": true}

func predicadoValido(s string) bool {
	if s == "" || !(s[0] >= 'a' && s[0] <= 'z') {
		return false
	}
	if palabrasReservadas[s] {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_' || c == '.' {
			continue
		}
		return false
	}
	return true
}

func identificadorValido(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' {
			continue
		}
		return false
	}
	return true
}

// constanteValida dice si una constante se puede escribir SIN comillas.
//
// HALLAZGO DEL FUZZING (el tercero de este parser): ':' y '-' estan permitidos
// porque los URN los llevan, pero juntos forman ':-', que es el operador de la
// regla. La constante ":-" se escribia sin comillas y al releerla partia la
// regla por la mitad. Se rechaza la secuencia, no los caracteres: urn:es:... y
// mi-norma siguen valiendo.
func constanteValida(s string) bool {
	if strings.Contains(s, ":-") {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') ||
			c == '_' || c == '.' || c == ':' || c == '-' || c == '/' || c == '@' {
			continue
		}
		return false
	}
	return true
}

// Escribir devuelve la regla en la sintaxis de superficie. Sirve para que un
// mensaje de error o un `dutiq explain` puedan ensenar la regla como la escribio
// quien la escribio, y para comprobar en un test que parsear y escribir dan la
// vuelta entera.
func (r Regla) Escribir() string {
	var b strings.Builder
	b.WriteString(escribirAtomo(r.Cabeza))
	b.WriteString(" :- ")
	partes := make([]string, 0, len(r.Cuerpo)+len(r.Negados))
	for _, a := range r.Cuerpo {
		partes = append(partes, escribirAtomo(a))
	}
	for _, a := range r.Negados {
		partes = append(partes, "not "+escribirAtomo(a))
	}
	b.WriteString(strings.Join(partes, ", "))
	return b.String()
}

func escribirAtomo(a Atomo) string {
	args := make([]string, len(a.Args))
	for i, t := range a.Args {
		args[i] = escribirTermino(t)
	}
	return a.Pred + "(" + strings.Join(args, ", ") + ")"
}

// escribirTermino decide si un termino necesita comillas PREGUNTANDOSELO AL
// PARSER, no repitiendo sus reglas.
//
// HALLAZGO DEL FUZZING, a los 21 segundos: escribiendo sin comillas todo lo que
// constanteValida aceptaba, la constante "_0" salia como _0, y _0 empieza por
// guion bajo, o sea que al releerla volvia como VARIABLE. Una constante que se
// convierte en variable al dar la vuelta es el fallo de derivar de mas otra vez,
// esta vez entrando por el escritor: bastaba con que una herramienta reescribiera
// un paquete para que una regla exacta pasara a aplicar a todo el mundo.
//
// Repetir las reglas del lexico en dos sitios es como se llego ahi. Asi que aqui
// no se repiten: se escribe sin comillas solo si volver a leerlo devuelve
// exactamente el mismo termino constante.
func escribirTermino(t Termino) string {
	if t.Var {
		return t.Val
	}
	if v, err := parsearTermino(t.Val); err == nil && !v.Var && v.Val == t.Val {
		return t.Val
	}
	// Con comillas. Si el valor lleva comillas o saltos de linea dentro, esto
	// tampoco se podra releer: es un valor que solo se puede construir desde
	// Go con C(), y el dialecto no lo sabe expresar a proposito.
	return `"` + t.Val + `"`
}
