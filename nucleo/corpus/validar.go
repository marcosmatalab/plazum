// EL LINTER DEL FORMATO: lo que un paquete tiene que cumplir para cargar.
//
// Es la frontera del producto con el corpus, y su razon de cambio es propia: una
// comprobacion nueva entra aqui sin tocar ni el esquema ni la carga.

package corpus

import (
	"fmt"
	"sort"
	"strings"

	"github.com/marcosmatalab/plazum/nucleo/ventana"
)

// validarFronteraLegal aplica el limite de la clase a todos los campos de texto.
//
// Solo corre en las clases que NO pueden redistribuir texto de un tercero:
//
//	Referencial  ISO, PCI DSS, SOC 2, TISAX. El cliente aporta su copia
//	             licenciada; el paquete solo puede traer identificadores.
//	Delegado     CIS, STIG. No se distribuye nada, y por eso texto_legal tiene
//	             que estar VACIO (eso se comprueba aparte, en Validar). El resto
//	             de campos llevan el mismo limite que un referencial: "nada de
//	             texto" no puede significar "ni siquiera una etiqueta", porque
//	             entonces la obligacion no se puede ni listar.
//
// Importado, Transcrito y Propio no llevan limite: en los tres hay derecho a
// redistribuir el texto entero (dominio publico, art. 13 TRLPI y Decision
// 2011/833/UE, o datos del propio proyecto).
func (p *Paquete) validarFronteraLegal(anotar func(error)) {
	switch {
	case p.Clase == Referencial || p.Clase == Delegado:
	case !p.Clase.Valida():
		// Una clase que no existe no acredita ningun derecho de redistribucion,
		// asi que se le aplica la frontera mas estricta. El paquete ademas no
		// carga, porque Validar rechaza la clase fuera de rango; esto esta aqui
		// para que la frontera no dependa de que ese otro chequeo siga vivo.
	default:
		return
	}
	for _, c := range camposDeTexto(p) {
		lim := c.Tipo.limite()
		if len(c.Valor) <= lim {
			continue
		}
		arreglo := "ISO, PCI DSS, SOC 2, TISAX y CIS no autorizan la redistribucion de su " +
			"texto: identificador y titulo corto, nada mas"
		if c.Tipo != prosa {
			arreglo = "un campo de referencia apunta al texto, no lo lleva dentro: " +
				"deja el localizador y quita el parrafo"
		}
		anotar(fmt.Errorf("%w: %s, campo %s con %d caracteres (limite %d en un paquete "+
			"de clase %s). %s",
			c.Tipo.centinela(), c.Donde, c.Campo, len(c.Valor), lim, p.Clase, arreglo))
	}
}

// validarLicenciaFuente exige los dos campos de higiene legal y los cruza con
// la clase.
//
// Se comprueba en TODAS las clases, tambien en las que no tienen a nadie a
// quien atribuir. Un referencial no le debe atribucion a ISO, pero quien abre
// la pantalla sigue teniendo que saber que ese paquete no trae el texto y que
// la copia la pone el, y ese es exactamente el mismo campo. Hacerlo opcional
// para media tabla es dejar la mitad de las pantallas en blanco.
func (p *Paquete) validarLicenciaFuente(anotar func(error)) {
	switch lf := p.LicenciaFuente; {
	case lf == "":
		anotar(fmt.Errorf("%w: %s. Sin ella no se sabe si el texto se puede redistribuir "+
			"ni a quien hay que atribuirlo. Arreglo: declara licencia_fuente con uno de "+
			"los regimenes de docs/LICENCIAS.md (%s)",
			ErrSinLicenciaFuente, p.URN, listaDeLicencias(p.Clase)))
	default:
		if motivo, prohibida := licenciasProhibidas[lf]; prohibida {
			anotar(fmt.Errorf("%w: %s declara %q. %s. No es un pendiente, es un no: "+
				"docs/LICENCIAS.md lo explica y ahi consta por que",
				ErrLicenciaProhibida, p.URN, lf, motivo))
			break
		}
		if !licenciaConocida(lf) {
			anotar(fmt.Errorf("%w: %s declara %q, que no existe. El vocabulario es cerrado "+
				"a proposito: una fuente nueva entra con su constante en nucleo/corpus y su "+
				"fila en docs/LICENCIAS.md, no escribiendo otra cadena",
				ErrLicenciaFuenteDesconocida, p.URN, lf))
			break
		}
		if p.Clase.Valida() && !admite(p.Clase, lf) {
			anotar(fmt.Errorf("%w: %s es de clase %s y declara %q. La clase admite %s. "+
				"Uno de los dos campos miente y el linter no puede saber cual: arregla el "+
				"que este mal antes de publicar",
				ErrLicenciaFuenteIncoherente, p.URN, p.Clase, lf, listaDeLicencias(p.Clase)))
		}
	}
	if p.Atribucion == "" {
		anotar(fmt.Errorf("%w: %s. La Decision 2011/833/UE autoriza reutilizar el DOUE con "+
			"atribucion, y una atribucion que no viaja con el paquete no se puede ensenar a "+
			"quien usa el producto. Arreglo: escribe en atribucion el aviso literal que "+
			"tiene que salir en pantalla", ErrSinAtribucion, p.URN))
	}
}

func licenciaConocida(lf LicenciaFuente) bool {
	for _, lista := range licenciasPorClase {
		for _, x := range lista {
			if x == lf {
				return true
			}
		}
	}
	return false
}

func admite(c Clase, lf LicenciaFuente) bool {
	for _, x := range licenciasPorClase[c] {
		if x == lf {
			return true
		}
	}
	return false
}

// listaDeLicencias enumera los regimenes de una clase, en orden estable, para
// que el error diga que hay que escribir y no solo que lo escrito esta mal.
func listaDeLicencias(c Clase) string {
	lista := licenciasPorClase[c]
	if len(lista) == 0 {
		var todas []string
		for _, l := range licenciasPorClase {
			for _, x := range l {
				todas = append(todas, string(x))
			}
		}
		sort.Strings(todas)
		return strings.Join(todas, ", ")
	}
	out := make([]string, 0, len(lista))
	for _, x := range lista {
		out = append(out, string(x))
	}
	return strings.Join(out, ", ")
}

// validarVigencias comprueba que las fechas de vigencia se pueden leer y que no
// van al reves. Se comprueba en el linter y no al usarlas porque una vigencia
// ilegible no es un caso raro de tiempo de ejecucion: es un fichero de datos de
// un tercero mal escrito, y el sitio de pararlo es la carga.
func (p *Paquete) validarVigencias(anotar func(error)) {
	if p.Vigencia.Desde == "" {
		anotar(fmt.Errorf("%w: paquete %s. Sin vigencia.desde no se sabe desde cuando se "+
			"exige nada de este paquete", ErrVigenciaSinDesde, p.URN))
	}
	if _, err := p.Vigencia.interpretar(); err != nil {
		anotar(fmt.Errorf("paquete %s: %w", p.URN, err))
	}
	validarLecturasDeVigencia("paquete "+p.URN, p.Vigencia, anotar)
	for _, o := range p.Obligaciones {
		if _, err := o.Vigencia.interpretar(); err != nil {
			anotar(fmt.Errorf("obligacion %s: %w", o.ID, err))
		}
		validarLecturasDeVigencia("obligacion "+o.ID, o.Vigencia, anotar)
	}
}

// validarLecturasDeVigencia comprueba las lecturas divergentes de una vigencia.
//
// Las cuatro cosas que se exigen, y las cuatro por el mismo motivo: una lectura
// divergente se le ENSENA al cliente al lado de la fecha que vincula, asi que
// tiene que poder defenderse sola.
//
//	id      para poder nombrarla en la pantalla y en el expediente.
//	cita    de donde sale. Sin cita es una opinion.
//	fecha   al menos una de las dos, o no dice nada.
//	que diverja de verdad. Una lectura identica a la declarada se lee como un
//	        desacuerdo que no existe, y eso hace dudar de la fecha buena.
func validarLecturasDeVigencia(donde string, v Vigencia, anotar func(error)) {
	vistos := map[string]bool{}
	for i, l := range v.Alternativas {
		nombre := l.ID
		if nombre == "" {
			nombre = fmt.Sprintf("#%d", i)
			anotar(fmt.Errorf("%s: la lectura divergente de vigencia %s no tiene id, "+
				"asi que no se puede nombrar donde se ensene", donde, nombre))
		}
		if vistos[l.ID] && l.ID != "" {
			anotar(fmt.Errorf("%s: la lectura divergente de vigencia %q esta declarada "+
				"dos veces", donde, l.ID))
		}
		vistos[l.ID] = true
		if strings.TrimSpace(l.Cita) == "" {
			anotar(fmt.Errorf("%w: %s, lectura %s. Escribe de donde sale esa otra fecha "+
				"(instrumento, articulo, y si no esta publicada, dilo)",
				ErrLecturaVigenciaSinCita, donde, nombre))
		}
		if l.Desde == "" && l.Hasta == "" {
			anotar(fmt.Errorf("%w: %s, lectura %s. Una lectura sin desde ni hasta no "+
				"discrepa de nada", ErrLecturaVigenciaVacia, donde, nombre))
			continue
		}
		// Se interpreta con las MISMAS reglas que la declarada: una lectura
		// divergente que no se puede leer es peor que ninguna, porque se ensena.
		if _, err := (Vigencia{Desde: l.Desde, Hasta: l.Hasta}).interpretar(); err != nil {
			anotar(fmt.Errorf("%s, lectura %s: %w", donde, nombre, err))
			continue
		}
		if l.Desde == v.Desde && l.Hasta == v.Hasta {
			anotar(fmt.Errorf("%w: %s, lectura %s dice lo mismo que la vigencia declarada "+
				"(desde=%q hasta=%q). Borrala, o corrige la que este mal",
				ErrLecturaVigenciaQueNoDiverge, donde, nombre, l.Desde, l.Hasta))
		}
	}
}

// ---------------------------------------------------------------------------
// El linter. Rechaza lo que no es seguro en vez de ejecutarlo a ver que pasa.
// ---------------------------------------------------------------------------

// Validar comprueba las invariantes del paquete. Devuelve todos los fallos, no
// solo el primero, porque quien escribe un paquete quiere arreglarlos de una vez.
func (p *Paquete) Validar() []error {
	var errs []error
	e := func(f string, a ...any) { errs = append(errs, fmt.Errorf(f, a...)) }
	// anotar es para los errores con centinela, que se comprueban con errors.Is
	// y no buscando una subcadena del mensaje. Ese patron ya dio aqui siete
	// tests en verde con el fallo delante: uno buscaba "clase" y lo encontraba
	// dentro de "clase_e2e".
	anotar := func(err error) { errs = append(errs, err) }

	// La clase, ANTES que nada, porque de ella depende que limites se aplican.
	//
	// HALLAZGO DEL FRENTE DE CORPUS, y es de la frontera legal, no de estilo:
	// el switch por clase de mas abajo tiene un default, asi que una clase
	// fuera de rango no era referencial y por tanto no tenia limite de texto.
	// Un paquete con "clase": 9 y 200 caracteres de texto de ISO validaba
	// limpio. La unica frontera que este proyecto declara no negociable se
	// esquivaba escribiendo un numero distinto en un fichero JSON.
	//
	// Se comprueba aqui arriba y no dentro del switch para que sea imposible
	// llegar a las comprobaciones por clase con una clase que no existe.
	if !p.Clase.Valida() {
		e("paquete %s: clase %d fuera de rango (0 importado, 1 transcrito, 2 referencial, "+
			"3 delegado, 4 propio). Una clase desconocida no puede cargar: es la que decide "+
			"si se puede redistribuir el texto normativo", p.URN, uint8(p.Clase))
	}

	// La frontera legal y las vigencias, antes que la forma. Un paquete puede
	// tener veinte fallos de forma; el que hay que ver primero en la salida es
	// el que redistribuye texto que no se puede redistribuir.
	p.validarFronteraLegal(anotar)
	p.validarLicenciaFuente(anotar)
	p.validarIdentificador(anotar)
	p.validarVigencias(anotar)

	p.validarAplicabilidad(e)
	p.validarRelojesEncendibles(anotar)
	p.validarOrigenDelIntervalo(anotar)
	p.validarMaximo(anotar)
	p.validarOrigenDeVigencia(anotar)
	p.validarCamposDePrimitiva(anotar)
	p.validarTransposicion(anotar)
	p.validarFragmentos(anotar)
	p.validarPreaviso(anotar)
	// EL PUENTE ENTRE LA ENTREVISTA Y EL MOTOR. Opcional mientras dura el
	// piloto; si esta, tiene que ser cierto (ver puente.go).
	p.validarPuente(anotar)
	p.validarOrigenesDePlantilla(e)
	p.validarRoles(e)
	p.validarCadenciasGemelas(anotar)
	p.validarPruebas(anotar)

	if p.URN == "" {
		e("paquete sin urn")
	}
	if p.Version == "" {
		e("paquete %s sin version", p.URN)
	}
	plantillas := map[string]bool{}
	for _, t := range p.Plantillas {
		plantillas[t.ID] = true
		if t.Cita == "" {
			e("plantilla %s sin cita normativa", t.ID)
		}
		for _, c := range t.Campos {
			if c.Origen == "" {
				e("plantilla %s campo %s sin origen: un entregable no puede tener "+
					"huecos que rellene un humano sin trazabilidad", t.ID, c.Nombre)
			}
		}
	}

	preguntas := map[string]bool{}
	entidades := map[string]map[string]bool{}
	for _, te := range p.Entidades {
		at := map[string]bool{}
		for _, a := range te.Atributos {
			at[a.Nombre] = true
			if a.Cita == "" {
				e("entidad %s atributo %s sin cita: si no se sabe de que articulo "+
					"sale el dato, no se le pregunta al usuario", te.Nombre, a.Nombre)
			}
			if a.Tipo == Enumerado && len(a.Valores) == 0 {
				e("entidad %s atributo %s es enumerado y no declara valores", te.Nombre, a.Nombre)
			}
		}
		entidades[te.Nombre] = at
	}
	for _, q := range p.Preguntas {
		preguntas[q.ID] = true
		if q.Cita == "" {
			e("pregunta %s sin cita", q.ID)
		}
		at, ok := entidades[q.Entidad]
		if !ok {
			e("pregunta %s apunta a la entidad %s, que el paquete no declara", q.ID, q.Entidad)
		} else if !at[q.Atributo] {
			e("pregunta %s apunta al atributo %s.%s, que no existe", q.ID, q.Entidad, q.Atributo)
		}
		if len(q.Desbloquea) == 0 {
			e("pregunta %s no desbloquea ninguna obligacion: es una pregunta que no "+
				"sirve para nada y no se le hace al usuario", q.ID)
		}
	}

	obl := map[string]bool{}
	for _, o := range p.Obligaciones {
		if o.ID == "" {
			anotar(fmt.Errorf("%w: la obligacion %q del paquete %s no se puede citar, ni "+
				"referenciar desde una pregunta, ni seguir en el expediente",
				ErrObligacionSinID, o.TituloLegible(), p.URN))
		}
		if obl[o.ID] {
			e("obligacion %s duplicada", o.ID)
		}
		obl[o.ID] = true
		if o.Cita == "" {
			e("obligacion %s sin cita normativa", o.ID)
		}
		if !clasesE2E[o.ClaseE2E] {
			e("obligacion %s: clase_e2e %q invalida u omitida (observable, documental, "+
				"procedimental, notificatoria, remediacion). Sin clase no hay medida "+
				"de profundidad e2e", o.ID, o.ClaseE2E)
		}
		for _, f := range o.Facetas {
			if !clasesE2E[f] {
				e("obligacion %s: faceta %q invalida", o.ID, f)
			}
		}
		for _, esc := range o.Escalado {
			if esc.Tras == "" || esc.A == "" {
				e("obligacion %s: escalon sin plazo o sin destinatario", o.ID)
				continue
			}
			// EL `tras` SE PARSEA AL CARGAR, y hasta hoy no lo parseaba nadie:
			// el campo llevaba desde el primer dia declarado en el formato y el
			// linter solo miraba que no estuviera vacio. Los 53 escalones del
			// corpus estan todos bien escritos — medido antes de escribir esto,
			// no supuesto — pero eso era suerte: el primer `P60D_ants` habria
			// salido el dia del incidente, que es el unico momento en que nadie
			// quiere descubrir un error de formato.
			if _, err := ParseTras(esc.Tras); err != nil {
				e("obligacion %s: %v", o.ID, err)
			}
		}
		if o.Entregable != "" && !plantillas[o.Entregable] {
			e("obligacion %s declara el entregable %s, que el paquete no incluye",
				o.ID, o.Entregable)
		}
		for _, q := range o.Preguntas {
			if !preguntas[q] {
				e("obligacion %s referencia la pregunta %s, que no existe", o.ID, q)
			}
		}
		// La frontera legal, comprobada por el linter y no por buena voluntad.
		// El limite de texto de la clase NO se comprueba aqui: lo hace
		// validarFronteraLegal sobre TODOS los campos de texto del paquete, no
		// solo sobre texto_legal. Mirar un campo de veinte era el agujero.
		switch p.Clase {
		case Referencial:
			if o.Delegado != "" {
				e("obligacion %s: solo un paquete delegado declara herramienta externa", o.ID)
			}
		case Delegado:
			if o.TextoLegal != "" {
				e("obligacion %s: un paquete delegado no distribuye texto. La licencia "+
					"del contenido la tiene la herramienta que lo comprueba", o.ID)
			}
			if o.Delegado == "" {
				e("obligacion %s: paquete delegado sin herramienta declarada. CIS "+
					"Benchmarks es CC BY-NC-SA: incompatible con AGPL y con vender, "+
					"asi que se lee la salida de quien si tiene la licencia", o.ID)
			}
		default:
			if o.Delegado != "" {
				e("obligacion %s: solo un paquete delegado declara herramienta externa", o.ID)
			}
		}
	}
	// Las preguntas apuntan a obligaciones que existen.
	for _, q := range p.Preguntas {
		for _, id := range q.Desbloquea {
			if !obl[id] {
				e("pregunta %s dice desbloquear %s, que no es una obligacion del paquete",
					q.ID, id)
			}
		}
	}
	// Todo reloj exige sus dorados: minimo 3 por obligacion con temporalidad,
	// derivados del texto. Y ningun dorado puede apuntar a una obligacion que
	// no existe.
	// Indice por ID para poder mirar la primitiva de la obligacion de cada
	// dorado. `obl` solo dice si existe.
	porID := map[string]Obligacion{}
	for _, o := range p.Obligaciones {
		porID[o.ID] = o
	}
	porObl := map[string]int{}
	clasesEjercitadas := map[string]map[string]bool{}
	for _, d := range p.Dorados {
		if !obl[d.Obligacion] {
			e("dorado %q apunta a la obligacion %s, que no existe", d.Caso, d.Obligacion)
		}
		// LA VENTANA ES OBLIGATORIA DONDE EL HORIZONTE MANDA. Sin ella, la
		// primitiva periodica devuelve exactamente hasta donde el autor dejo de
		// escribir, y truncar la lista por la cola sale VERDE: la afirmacion se
		// hace verdadera a si misma. Medido el 27-08-2026 sobre este mismo
		// corpus, y es la razon de que el campo exista.
		if o, hay := porID[d.Obligacion]; hay && o.Temporalidad != nil &&
			ConsumeElHorizonte(o.Temporalidad.Primitiva) && strings.TrimSpace(d.Hasta) == "" {
			e("dorado %q: la obligacion %s es de primitiva %q, que se acota con el horizonte, "+
				"y el caso no declara `hasta`. Sin ventana declarada, el motor devuelve "+
				"exactamente hasta la ultima fecha que tu escribes, asi que borrar la ultima "+
				"fila deja el caso VERDE y la exhaustividad es mentira. Escribe la ventana "+
				"dentro de la cual afirmas que esas son TODAS las ocurrencias",
				d.Caso, d.Obligacion, o.Temporalidad.Primitiva)
		}
		// Y se apunta que clases ejercita, para la comprobacion de cobertura.
		if clasesEjercitadas[d.Obligacion] == nil {
			clasesEjercitadas[d.Obligacion] = map[string]bool{}
		}
		for k := range d.Hechos {
			clasesEjercitadas[d.Obligacion][k] = true
		}
		if d.CitaDelEsperado == "" {
			e("dorado %q sin cita_del_esperado: el esperado se deriva del texto, no de la implementacion", d.Caso)
		}
		validarEsperadoDeDorado(d, e)
		porObl[d.Obligacion]++
	}
	// TODA CLASE DECLARADA TIENE QUE ESTAR EJERCITADA POR ALGUN DORADO.
	//
	// Es la direccion que ni el emparejamiento por hito ni la exhaustividad
	// alcanzan, y es el DUAL EXACTO de la mutacion que estreno este formato.
	// Quitar una clase hace que su hito rija siempre y el conjunto se llena:
	// rojo. Anadir un hito con una clase que ningun dorado ejercita no produce
	// fila en NINGUNA direccion: el motor lo excluye rio arriba (Plazo hace
	// `continue` cuando la clase no es la vigente), asi que no falta porque
	// nadie lo declara y no sobra porque nadie lo emite. Un plazo inventado
	// entra en el corpus sin que se entere ninguna puerta.
	for _, o := range p.Obligaciones {
		if o.Temporalidad == nil {
			continue
		}
		for _, h := range o.Temporalidad.Hitos {
			if h.Clase == "" {
				continue
			}
			if !clasesEjercitadas[o.ID][h.Clase] {
				e("obligacion %s: el hito %q solo rige cuando consta el hecho %q, y NINGUN "+
					"dorado lo declara. Un hito asi no sale en ninguna de las dos direcciones "+
					"del esperado (no falta, porque nadie lo pide; no sobra, porque el motor lo "+
					"excluye antes de formar el conjunto), asi que se le puede meter al corpus "+
					"un plazo que la norma no da y ninguna puerta lo ve. Escribe un dorado con "+
					"ese hecho", o.ID, h.ID, h.Clase)
			}
		}
	}

	for _, o := range p.Obligaciones {
		if o.Temporalidad == nil {
			continue
		}
		// EL MINIMO DE TRES DORADOS SOLO ALCANZA A LO QUE SE PUEDE CALCULAR.
		//
		// Lo destapo el art. 67.1 del RDL 19/2018 (notificacion de incidentes de
		// pago), que obliga a notificar "de forma inmediata" y NO DA NINGUN
		// NUMERO. El motor sabe decir eso: limite indeterminado, estado "sin
		// plazo legal", y se mide el tiempo transcurrido. Pero un dorado fija
		// una FECHA esperada, y de un plazo sin numero no sale ninguna: los
		// tres dorados que el linter exigia no se pueden escribir.
		//
		// Lo que hacia la regla anterior era empujar al autor a QUITARLE EL
		// RELOJ a esa obligacion para que el paquete cargara, y entonces el
		// producto deja de ensenar el cronometro y la obligacion se lee como
		// una mas sin urgencia. La regla castigaba justo la transcripcion
		// honesta.
		//
		// La regla nueva no abre un agujero: la exencion solo vale cuando NINGUN
		// limite es computable, y ademas exige que cada hito lleve NOTA. Asi el
		// hueco es una decision escrita y consultable, no una omision. Sin la
		// nota, el camino barato (dejar el limite vacio para librarse de los
		// dorados) vuelve a estar cerrado.
		if computables(*o.Temporalidad) {
			if porObl[o.ID] < 3 {
				e("obligacion %s declara reloj computable y tiene %d dorados (minimo 3: normal, "+
					"borde de calendario, y ocurrencia u variante)", o.ID, porObl[o.ID])
			}
			continue
		}
		// LA `continua` NO PASA POR LA VIA DE LOS `hitos`, y tiene su propia
		// guarda.
		//
		// Un deber permanente no tiene hitos que enumerar: tiene UN estado, que
		// es "no vence". Exigirle la forma de un plazo escalonado seria pedirle
		// que se disfrace de otra cosa.
		//
		// PERO ES UNA SALIDA BARATA SI NO SE VIGILA: marcar `continua` una
		// obligacion que si tiene cadencia libraria de escribir los tres
		// dorados. Contra eso, dos guardas, y la segunda es la que muerde:
		//
		//	1. tiene que traer AL MENOS UN dorado, porque el ejecutor de dorados
		//	   compara el conjunto entero y ahi es donde se afirma el estado.
		//	2. su propio TEXTO LEGAL no puede decir que es periodica. Si el
		//	   boletin dice «periodicamente» y el paquete dice `continua`, uno de
		//	   los dos miente, y no es el boletin.
		if o.Temporalidad.Primitiva == "continua" {
			if porObl[o.ID] < 1 {
				e("obligacion %s es una `continua` y no trae ni un dorado. Un deber permanente "+
					"no admite los tres (no produce fechas), pero si el que afirma que NO VENCE: "+
					"sin el, nadie ha comprobado nunca que el motor diga eso", o.ID)
			}
			if m := periodicidadEnElTexto(o.TextoLegal); m != "" {
				e("obligacion %s se declara `continua` y su propio texto legal dice %q. Si la "+
					"norma le pone ritmo, no es un deber permanente: es una `periodica`, y como "+
					"tal debe traer cadencia, origen del intervalo y sus tres dorados. Declarar "+
					"`continua` para librarse de ellos es la salida barata que esta guarda cierra",
					o.ID, m)
			}
			continue
		}
		if len(o.Temporalidad.Hitos) == 0 {
			e("obligacion %s declara un reloj sin numero con la forma simple (hito y limite). "+
				"Un plazo que la norma no cuantifica se escribe con `hitos`, para que cada uno "+
				"lleve su `nota` diciendo que dice la norma en vez del numero que no da", o.ID)
			continue
		}
		for _, h := range o.Temporalidad.Hitos {
			if strings.TrimSpace(h.Nota) == "" {
				e("obligacion %s, hito %s: no fija limite y no dice por que. Un plazo sin numero "+
					"se queda sin los tres dorados que el linter exige a los demas, asi que la "+
					"nota es lo unico que queda para que el hueco sea una decision y no un "+
					"descuido: escribe que dice la norma (de forma inmediata, sin demora "+
					"indebida) y su cita", o.ID, h.ID)
			}
		}
	}
	return errs
}

// validarEsperadoDeDorado comprueba la FORMA del conjunto esperado. Lo que dice
// (que las fechas sean las que da la norma) lo comprueba EjecutarDorado contra
// el motor; aqui solo se rechaza un esperado que no pueda afirmar nada.
//
// LAS DOS FORMAS DE LA NADA SE MIRAN POR SEPARADO (invariante 8), y las dos se
// rechazan, con mensajes distintos porque son fallos distintos:
//
//	nil (campo ausente o null)  se le olvido al autor. Es la forma peligrosa,
//	                            porque sale sola: un dorado sin esperado cuenta
//	                            para el minimo de tres y no afirma nada.
//	lista vacia presente        el autor quiso decir "el motor no devuelve
//	                            nada". Hoy eso no puede ser cierto para ningun
//	                            reloj del formato, y ademas se cumple SOLO: el
//	                            horizonte del ejecutor sale de la ultima fecha
//	                            declarada, asi que una lista vacia da horizonte
//	                            cero, y con horizonte cero una `periodica` no
//	                            devuelve ocurrencias. La afirmacion se hace
//	                            verdadera a si misma. El dia que una primitiva
//	                            devuelva legitimamente el vacio, se declarara
//	                            con una palabra que lo diga, no con la ausencia
//	                            de filas.
func validarEsperadoDeDorado(d Dorado, e func(string, ...any)) {
	switch {
	case d.Esperado == nil:
		e("dorado %q sin esperado: un caso que no declara ningun vencimiento no afirma "+
			"nada y aun asi cuenta para el minimo de tres por reloj. Arreglo: "+
			"\"esperado\": [{\"hito\": \"...\", \"vence\": \"...\"}], con TODOS los "+
			"vencimientos que el motor da con esos hechos", d.Caso)
	case len(d.Esperado) == 0:
		e("dorado %q con esperado vacio: una lista sin filas dice \"el motor no devuelve "+
			"nada\", y eso hoy no es cierto para ningun reloj (un plazo devuelve una fila "+
			"por hito, aunque sea pendiente de hecho o sin plazo legal). Ademas se cumple "+
			"sola: sin fechas declaradas el horizonte es cero y una periodica tampoco "+
			"devuelve nada. Arreglo: escribe las filas que salen", d.Caso)
	}
	vistos := map[string]bool{}
	for i, esp := range d.Esperado {
		if esp.Hito == "" {
			e("dorado %q, fila %d del esperado sin hito: el emparejamiento con el motor se "+
				"hace POR HITO y no por posicion, asi que una fila sin hito no casa con "+
				"nada. Arreglo: pon el id del hito de la obligacion (en una periodica, "+
				"\"nombre#n\")", d.Caso, i+1)
			continue
		}
		if vistos[esp.Hito] {
			e("dorado %q declara el hito %q dos veces en el esperado: el motor devuelve un "+
				"vencimiento por hito, asi que una de las dos filas no se comprobaria "+
				"contra nada y quedaria verde diga lo que diga", d.Caso, esp.Hito)
		}
		vistos[esp.Hito] = true

		estado := ventana.Determinado
		if esp.Estado != "" {
			var err error
			if estado, err = ventana.ParseEstadoVenc(esp.Estado); err != nil {
				e("dorado %q, hito %q: %v. Vacio significa \"determinado\"", d.Caso, esp.Hito, err)
				continue
			}
		}
		if estado == ventana.Determinado && esp.Vence == "" {
			e("dorado %q, hito %q: estado determinado sin vence. Determinado quiere decir "+
				"que hay fecha y hora exactas, asi que sin fecha la fila no afirma nada. "+
				"Arreglo: pon la fecha, o declara el estado que de verdad da el motor "+
				"(%q o %q)", d.Caso, esp.Hito,
				ventana.PendienteDeHecho.String(), ventana.SinPlazoLegal.String())
		}
		if estado != ventana.Determinado && esp.Vence != "" {
			e("dorado %q, hito %q: estado %q Y vence %q a la vez. Un vencimiento que no esta "+
				"determinado NO tiene fecha, asi que declararla es afirmar algo que el "+
				"motor no dice. Arreglo: quita una de las dos", d.Caso, esp.Estado, esp.Vence)
		}
	}
	// El opt-out: si esta, tiene que ser un argumento. Ver
	// MinimoDelMotivoDeSubconjunto.
	motivo := strings.TrimSpace(d.SubconjuntoPorque)
	if d.SubconjuntoPorque != "" && len(motivo) < MinimoDelMotivoDeSubconjunto {
		e("dorado %q: subconjunto_porque con %d caracteres utiles (minimo %d). Renunciar a "+
			"la exhaustividad tiene que costar un argumento: di QUE hitos quedan fuera y "+
			"POR QUE este caso no los afirma. Un motivo que no cabe en esa frase es una "+
			"etiqueta, y entonces el campo seria el booleano que este formato no quiere",
			d.Caso, len(motivo), MinimoDelMotivoDeSubconjunto)
	}
}

// computables dice si el reloj declarado produce alguna FECHA. Un plazo cuyos
// limites son todos indeterminados obliga y no se puede calcular, que son dos
// cosas ciertas a la vez.
func computables(t Temporalidad) bool {
	indeterminado := func(s string) bool { return s == "" || s == "indeterminado" }
	// UNA `continua` SIN FIN DECLARADO NO DA NINGUNA FECHA, y por tanto no
	// admite los tres dorados.
	//
	// El minimo de tres (normal, borde de calendario, y ocurrencia u variante)
	// esta escrito para relojes que producen FECHAS: los tres casos son tres
	// fechas. Un deber permanente no produce ninguna, y su unico caso posible
	// es el que dice que no vence. Exigirle un "borde de calendario" es exigir
	// una prueba sobre un calendario que no existe, y el camino barato para
	// librarse de ella seria quitarle el reloj, que es exactamente lo que esta
	// regla lleva evitando desde el art. 67.1 del RDL 19/2018.
	//
	// LA EXENCION ES ESTRECHA A PROPOSITO: si la `continua` declara `en` (una
	// fecha de fin), si produce fecha y vuelven a exigirse los tres.
	if t.Primitiva == "continua" {
		return !indeterminado(t.En)
	}
	// UN `maximo` SIN SUELO no da ninguna fecha: la norma obliga a conservar y
	// no dice cuanto, asi que el motor mide el tiempo transcurrido igual que
	// hace con las tres obligaciones sin numero que el corpus ya trae.
	if t.Primitiva == "maximo" {
		return !indeterminado(t.Suelo)
	}
	if t.Primitiva != "plazo" {
		return true // periodica, puntual y las demas siempre dan fecha
	}
	if len(t.Hitos) == 0 {
		return !indeterminado(t.Limite)
	}
	for _, h := range t.Hitos {
		if !indeterminado(h.Limite) {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// Lo derivado. Nada de esto se escribe por norma: sale del paquete.
// ---------------------------------------------------------------------------
