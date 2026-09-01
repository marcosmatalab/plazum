package acta

import (
	"fmt"
	"strings"

	"github.com/marcosmatalab/plazum/nucleo/accesos"
	"github.com/marcosmatalab/plazum/nucleo/auditoria"
	"github.com/marcosmatalab/plazum/nucleo/incidente"
)

// Las frases que son de ESTE paquete, porque describen un emparejamiento que
// hace este paquete y no un dato de los otros tres. Las demas se toman del
// paquete que produce el dato y no se copian aqui.
const (
	// LaFraseDeLaAsignacionQueNoCasa acompana a un responsable asignado a algo
	// que este programa no audita. Es la direccion que se olvida del
	// emparejamiento (invariante 7): no se traga, se cuenta y se dice.
	LaFraseDeLaAsignacionQueNoCasa = "Esto NO dice que la asignacion este mal: dice que este " +
		"programa no tiene esa unidad en su alcance, asi que aqui no se ha mirado su independencia."

	// LaFraseDeLoFueraDelPeriodo acompana a lo que se conocio fuera del periodo.
	LaFraseDeLoFueraDelPeriodo = "Esto NO dice que estos incidentes no cuenten: dice que se " +
		"conocieron fuera del periodo que cubre este acta, asi que los cuenta el acta de su periodo."

	// LaFraseDeLaNotificacionQueNoCasa es la otra direccion del mismo
	// emparejamiento: una notificacion que el corpus espera de un incidente que
	// este acta no cuenta.
	LaFraseDeLaNotificacionQueNoCasa = "Esto NO dice que falte una notificacion: dice que el " +
		"incidente al que se refiere no es de los que cuenta este acta."

	// LaFraseDelEmpateDeClasificacion: dos clasificaciones distintas en el mismo
	// instante. No se resuelve, se dice, igual que en accesos y en ventana.
	LaFraseDelEmpateDeClasificacion = "Esto NO dice que la clasificacion sea otra: dice que " +
		"constan dos clasificaciones distintas en el mismo instante y plazum no elige una. Quien " +
		"firme tiene que saber que aqui hubo dos manos."

	// LaFraseDeLaRemisionFueraDelPeriodo: consta remitida, pero no dentro del
	// periodo. Contarla como hecha falsearia el periodo y contarla como no
	// constatada acusaria en falso, asi que es su propio cubo.
	LaFraseDeLaRemisionFueraDelPeriodo = "Esto NO dice que no se notificara: dice que consta " +
		"remitida fuera del periodo que cubre este acta."

	// LaFraseDeLoQuePlazumNoEscribe es la respuesta entera a la pregunta de por
	// que este documento no trae conclusiones redactadas.
	LaFraseDeLoQuePlazumNoEscribe = "Esta parte la escribe la direccion. Plazum no la redacta y " +
		"no la puede redactar, y no es una limitacion tecnica: una conclusion sobre si el sistema " +
		"funciona es justo lo que el organo de gobierno tiene que decir con sus palabras, y un " +
		"parrafo verosimil escrito por una maquina en este documento seria una firma falsa. En " +
		"este acta no consta todavia ninguna decision de la direccion."
)

// Notificacion es una notificacion que EL CORPUS espera de un incidente.
//
// Llega como dato y no se deduce aqui: que un incidente obligue a notificar lo
// dicen los arts. 3 a 14 del Reglamento de Ejecucion que corresponda y sus
// hermanos, y eso vive en su paquete de datos (invariante 2). Este paquete solo
// contrasta lo que le den contra lo que el registro del incidente dice.
type Notificacion struct {
	// Incidente es el id, y es el campo por el que casa.
	Incidente string
	// Hito es el id del hito notificatorio, tal y como lo nombra la obligacion.
	Hito string
	// Que es el titulo de la obligacion, para que una persona lo reconozca.
	Que string
}

// Clave es la identidad del par. Nunca la posicion en la lista.
func (n Notificacion) Clave() string { return n.Incidente + "|" + n.Hito }

// Entradas es todo lo que hace falta para componer. Nada se va a buscar: de
// donde salga cada cosa (un fichero, el ledger, otra instalacion) es del
// adaptador que llama.
type Entradas struct {
	ID           string
	Organizacion string
	Periodo      Periodo
	// Cubre son las obligaciones de las que este acta es evidencia, con la cita
	// que escribio su paquete.
	Cubre []Obligacion

	// Programa puede ser nil: la seccion sale igual, diciendo que falta.
	Programa *auditoria.Programa
	// Responsables es unidad -> quien responde de ella (SCIM, asignacion
	// manual). Puede faltar entera: entonces la independencia no se ha podido
	// mirar, y eso tiene su cubo en vez de desaparecer.
	Responsables map[string]string

	// Campana puede ser nil, igual que Programa.
	Campana *accesos.Campana
	// ConNombresDelCenso hace que las filas de accesos lleven el ROTULO con el
	// que el IdP nombra a la cuenta ("Ana Perez") ademas de su identidad.
	//
	// EL VALOR CERO ES EL RESTRICTIVO, y aqui eso no es celo: es la unica
	// estructura de opciones que este paquete tiene y esta en una frontera de
	// confianza de verdad, porque el acta ES LA PIEZA QUE VIAJA. Se imprime, se
	// manda por correo y se adjunta a un expediente; olvidarse de apagar algo en
	// el documento que mas circula del producto es exactamente como se publica
	// sin querer un directorio de empleados.
	//
	// Y LA LINEA CAE DONDE CAE POR UNA RAZON, no por prudencia general. El acta
	// lleva dentro dos clases de persona que no se pueden tratar igual:
	//
	//	los ACTORES     quien audito, quien difirio y por que, quien decidio un
	//	                acceso, quien excuso una linea, quien asistio. Son
	//	                imprescindibles: un acta que no dice quien hizo que no es
	//	                evidencia de nada, y quitarlos seria vaciar el documento.
	//	los SUJETOS     el rotulo con el que el IdP nombra la cuenta revisada. NO
	//	                es imprescindible: la identidad de una fila es
	//	                sistema|cuenta|permiso, y el propio censo dice de su
	//	                rotulo que "NO identifica". Un consejo no necesita saber
	//	                que Ana Perez tiene admin en el ERP; necesita cuantos
	//	                accesos quedaron sin revisar y poder abrirlos.
	//
	// Asi que el interruptor cubre EXACTAMENTE los sujetos, y los actores viajan
	// siempre. Quien quiera ver a las personas del censo tiene la pantalla de la
	// UAR, que es de quien es ese dato.
	ConNombresDelCenso bool

	// HayRegistroDeIncidentes distingue las DOS FORMAS DE LA NADA
	// (invariante 8), y es el unico sitio del acta donde hace falta un campo
	// para eso: con Programa y Campana el nil ya las separa, pero un slice de
	// incidentes vacio no dice si el periodo fue tranquilo o si nadie ha
	// conectado el registro. Y las dos cosas se leen de forma opuesta en un
	// acta: "cero incidentes" es una noticia y "no lo hemos mirado" es un hueco.
	HayRegistroDeIncidentes bool
	Incidentes              []*incidente.Incidente
	// Esperadas son las notificaciones que el corpus espera de esos incidentes.
	Esperadas []Notificacion

	// QuienAsistio a la revision. Lo aporta una persona.
	QuienAsistio []string
	// Decisiones es lo que decide la direccion. TIENEN que venir con
	// procedencia DeUnaPersona: una decision del organo redactada por plazum es
	// exactamente la falsificacion que este documento no puede contener.
	Decisiones []Parrafo
}

// Componer arma el acta. Es la unica puerta: valida las tres reglas antes de
// devolver nada, asi que un Acta que existe es un Acta que las cumple.
func Componer(e Entradas) (Acta, error) {
	var faltan []string
	if strings.TrimSpace(e.ID) == "" {
		faltan = append(faltan, "id del acta")
	}
	if strings.TrimSpace(e.Organizacion) == "" {
		faltan = append(faltan, "de quien es el acta: un acta sin organizacion no es evidencia "+
			"de nadie")
	}
	if !e.Periodo.valido() {
		faltan = append(faltan, "periodo con desde y hasta (y hasta despues de desde): sin "+
			"periodo no se puede decir que incidente entra y cual no")
	}
	if len(faltan) > 0 {
		return Acta{}, fmt.Errorf("%w: %s", ErrActa, strings.Join(faltan, "; "))
	}
	if err := comprobarIncidentes(e); err != nil {
		return Acta{}, err
	}
	if err := comprobarEsperadas(e); err != nil {
		return Acta{}, err
	}
	for _, d := range e.Decisiones {
		if d.De != DeUnaPersona {
			return Acta{}, fmt.Errorf("%w: la decision %q se presenta como %q.\n"+
				"  Las decisiones de la direccion las escribe una persona y consta quien. Que "+
				"plazum redacte una conclusion del organo de gobierno y la firme el organo es la "+
				"falsificacion exacta que este documento no puede contener",
				ErrProsaSinProcedencia, corto(d.Texto), d.De)
		}
	}

	a := Acta{
		ID:           e.ID,
		Organizacion: e.Organizacion,
		Periodo:      e.Periodo,
		Cubre:        append([]Obligacion(nil), e.Cubre...),
		Cabecera:     cabecera(e),
	}
	// EN EL ORDEN DEL VOCABULARIO, y las cuatro siempre. Una seccion que
	// desaparece cuando no tiene datos deja un acta que parece completa.
	for _, f := range FuentesPosibles() {
		switch f {
		case DelProgramaDeAuditoria:
			a.Secciones = append(a.Secciones, seccionAuditoria(e))
		case DeLaCampanaDeAccesos:
			a.Secciones = append(a.Secciones, seccionAccesos(e))
		case DeLosIncidentes:
			a.Secciones = append(a.Secciones, seccionIncidentes(e))
		case DeLaDireccion:
			a.Secciones = append(a.Secciones, seccionDireccion(e))
		}
	}
	if err := a.validar(); err != nil {
		return Acta{}, err
	}
	return a, nil
}

// comprobarIncidentes aplica la TERCERA hermana del invariante 8: un incidente
// presente y no interpretable no se convierte en un cubo por defecto, da error.
//
// Un *Incidente nil o sin apertura no tiene primer conocimiento, asi que no se
// puede situar en el periodo. Meterlo en un cubo "no se sabe" seria inventarse
// que alguien lo aporto asi a proposito; lo que hay es un fallo de quien llama, y
// se dice donde se puede arreglar.
func comprobarIncidentes(e Entradas) error {
	vistos := map[string]bool{}
	for i, in := range e.Incidentes {
		if in == nil || !in.Abierto() {
			return fmt.Errorf("%w: el incidente en la posicion %d no tiene apertura, asi que no "+
				"tiene primer conocimiento y no se puede situar en el periodo.\n"+
				"  No se cuenta en un cubo de \"no se sabe\": eso seria dar por aportado a "+
				"proposito lo que es un fallo de quien llama. Arreglo: construirlo con "+
				"incidente.Abrir", ErrActa, i)
		}
		if strings.Contains(in.ID(), "|") {
			return fmt.Errorf("%w: el incidente %q lleva la barra dentro, y la barra es el "+
				"separador con el que se compone la clave (incidente|hito).\n"+
				"  Con ella dentro, dos notificaciones esperadas DISTINTAS dan la misma clave, "+
				"y el acta las cuenta como una o rechaza la buena por repetida.\n"+
				"  Arreglo: un identificador de incidente sin barras", ErrActa, in.ID())
		}
		if vistos[in.ID()] {
			return fmt.Errorf("%w: el incidente %q viene dos veces.\n"+
				"  Cada numero de este acta es la lista de lo que lo compone, y un incidente "+
				"repetido lo cuenta dos veces sin que se note", ErrActa, in.ID())
		}
		vistos[in.ID()] = true
	}
	if !e.HayRegistroDeIncidentes && len(e.Incidentes) > 0 {
		return fmt.Errorf("%w: llegan %d incidentes y a la vez se dice que no hay registro de "+
			"incidentes.\n"+
			"  Las dos formas de la nada no son la misma y esta no es ninguna: o hay registro y "+
			"los incidentes son los que son, o no lo hay y no puede haber ninguno",
			ErrActa, len(e.Incidentes))
	}
	return nil
}

func comprobarEsperadas(e Entradas) error {
	vistas := map[string]bool{}
	for _, n := range e.Esperadas {
		if strings.TrimSpace(n.Incidente) == "" || strings.TrimSpace(n.Hito) == "" {
			return fmt.Errorf("%w: una notificacion esperada sin incidente o sin hito no casa "+
				"con nada y no se puede contar en ninguna direccion", ErrActa)
		}
		// LA MISMA GUARDA QUE EN auditoria.Unidad, y por el mismo motivo: sin
		// ella, ("INC|A", "n1") y ("INC", "A|n1") son dos notificaciones
		// esperadas distintas con la misma clave.
		if strings.Contains(n.Incidente, "|") || strings.Contains(n.Hito, "|") {
			return fmt.Errorf("%w: la notificacion esperada (%q, %q) lleva la barra dentro, y la "+
				"barra es el separador de la clave.\n"+
				"  Dos notificaciones distintas darian la misma, y el acta contaria una donde "+
				"hay dos.\n"+
				"  Arreglo: identificadores de incidente y de hito sin barras",
				ErrActa, n.Incidente, n.Hito)
		}
		if vistas[n.Clave()] {
			return fmt.Errorf("%w: la notificacion esperada %q viene dos veces", ErrActa, n.Clave())
		}
		vistas[n.Clave()] = true
	}
	return nil
}

// cabecera es lo que se lee antes de los numeros.
//
// TODO ES DePlazum menos las obligaciones que el acta evidencia, y eso significa
// que las PALABRAS estan en este repositorio, palabra por palabra, no que los
// datos que nombran salgan de aqui: los identificadores, las fechas y los
// recuentos vienen de los registros. La procedencia es de la prosa.
func cabecera(e Entradas) []Parrafo {
	ps := []Parrafo{
		dePlazum(pfCompone),
		dePlazum(pfNoDiceCumplido),
	}
	if len(e.Cubre) == 0 {
		ps = append(ps, dePlazum(pfSinCubre))
	} else {
		ps = append(ps, dePlazum(pfEvidenciaDe), dePlazum(pfIdiomaDelCorpus))
	}
	for _, o := range e.Cubre {
		// SIN CLAVE Y A PROPOSITO: estas palabras son del paquete de la norma,
		// no de plazum, y por eso no se traducen. El parrafo de arriba explica
		// por que salen en otro idioma cuando la interfaz esta en ingles.
		ps = append(ps, Parrafo{
			Frase: Frase{Texto: fmt.Sprintf("%q (%s %s, %s)", o.Titulo, o.Paquete, o.Version, o.ID)},
			De:    DeLaNorma,
			Cita:  o.Cita,
		})
	}
	if len(e.QuienAsistio) > 0 {
		ps = append(ps, dePlazum(pfAsistieron, strings.Join(e.QuienAsistio, ", ")))
	} else {
		ps = append(ps, dePlazum(pfSinQuienAsistio))
	}
	return ps
}

// ---------------------------------------------------------------------------
// La seccion de auditoria interna
// ---------------------------------------------------------------------------

func seccionAuditoria(e Entradas) Seccion {
	s := Seccion{Fuente: DelProgramaDeAuditoria}
	p := e.Programa
	if p == nil {
		falta := dePlazum(pfSinPrograma)
		s.PorQueFalta = falta.Texto
		s.Parrafos = []Parrafo{falta}
		return s
	}
	s.Aportada = true
	// SIN RECUENTOS EN LA PROSA. Los numeros de este acta viven en los repartos,
	// que es donde llevan su referencia y se pueden abrir; repetir uno aqui daria
	// una cifra sin derivacion, y ademas dos numeros parecidos a tres lineas de
	// distancia se leen como una contradiccion.
	s.Parrafos = []Parrafo{dePlazum(pfPrograma, p.ID(), p.Ciclo().Nombre,
		p.Ciclo().Desde.Format("2006-01-02"), p.Ciclo().Hasta.Format("2006-01-02"))}
	if arr := p.DelCicloAnterior(); arr.DeCiclo != "" {
		s.Parrafos = append(s.Parrafos, dePlazum(pfArrastraDe, arr.DeCiclo))
	} else {
		s.Parrafos = append(s.Parrafos, dePlazum(pfPrimerCiclo))
	}
	s.Repartos = []Reparto{
		repartoCobertura(p),
		repartoHallazgos(p),
		repartoArrastreAnterior(p),
		repartoAsignaciones(p, e.Responsables),
		repartoIndependencia(p, e.Responsables),
	}
	return s
}

func repartoCobertura(p *auditoria.Programa) Reparto {
	pendiente := map[string]auditoria.UnidadPendiente{}
	for _, u := range p.SinAuditarDesdeHace() {
		pendiente[u.Unidad.Clave()] = u
	}
	// La PRIMERA sesion que cubre cada unidad, para poder decir cual fue. Casa
	// por la clave de unidad, que es la misma que Auditar ya exigio que
	// estuviera en el alcance.
	sesionDe := map[string]auditoria.Sesion{}
	for _, ses := range p.Sesiones() {
		for _, u := range ses.Unidades {
			if _, ya := sesionDe[u]; !ya {
				sesionDe[u] = ses
			}
		}
	}
	diferida := map[string]auditoria.Diferimiento{}
	for _, d := range p.Diferimientos() {
		diferida[d.Unidad] = d
	}

	cubos := map[auditoria.Cobertura][]Elemento{}
	for _, u := range p.Unidades() {
		k := u.Clave()
		cob := p.CoberturaDe(k)
		el := Elemento{Clave: k, Que: u.Titulo}
		switch cob {
		case auditoria.Auditada:
			ses := sesionDe[k]
			el.Nota = fmt.Sprintf("sesion %s, %s, auditor %s", ses.ID,
				ses.Cuando.Format("2006-01-02"), ses.Auditor)
		case auditoria.Diferida:
			d := diferida[k]
			el.Nota = fmt.Sprintf("diferida por %s el %s: %s (%s)", d.Quien,
				d.Cuando.Format("2006-01-02"), d.Motivo, pendiente[k].Antiguedad())
		default:
			el.Nota = pendiente[k].Antiguedad()
		}
		cubos[cob] = append(cubos[cob], el)
	}

	r := Reparto{
		Rotulo:   repartoDeCobertura,
		Universo: p.Alcance(),
		DeDonde:  "auditoria.Programa.Alcance(), que sale del corpus instalado y del alcance declarado",
	}
	// EL VOCABULARIO ENTERO, en su orden y con los vacios.
	for _, cob := range auditoria.CoberturasPosibles() {
		switch cob {
		case auditoria.SinAuditar:
			r.Cifras = append(r.Cifras,
				ausencia(cuboDeCobertura(cob), cubos[cob], descargoNoAuditado))
		case auditoria.Diferida:
			// Diferir explica por que falta; no hace que deje de faltar, asi que
			// lleva la misma frase. Y no es una ausencia: hay un motivo escrito
			// y consta quien lo escribio.
			r.Cifras = append(r.Cifras,
				noEsCulpa(cuboDeCobertura(cob), cubos[cob], descargoNoAuditado))
		default:
			r.Cifras = append(r.Cifras, recuento(cuboDeCobertura(cob), cubos[cob]))
		}
	}
	return r
}

func repartoHallazgos(p *auditoria.Programa) Reparto {
	abierto := map[string]bool{}
	for _, h := range p.Abiertos() {
		abierto[h.ID] = true
	}
	var esteAbierto, esteCerrado, arrastrados []Elemento
	deEsteCiclo := map[string]bool{}
	for _, h := range p.Hallazgos() {
		deEsteCiclo[h.ID] = true
		// El texto lo escribio el auditor y va DETRAS DE SU NOMBRE, en la misma
		// cadena: es la regla que cierra la via de la prosa ajena dentro de una
		// Nota. Y va entero, porque un board pack sin el texto de los hallazgos
		// obliga a pedir otro documento, que es como se deja de leer este.
		el := Elemento{
			Clave: h.ID,
			Que:   h.Unidad,
			Nota: fmt.Sprintf("%s, anotado por %s el %s: %s", h.Clase, h.Quien,
				h.Cuando.Format("2006-01-02"), h.Texto),
		}
		if abierto[h.ID] {
			esteAbierto = append(esteAbierto, el)
		} else {
			esteCerrado = append(esteCerrado, el)
		}
	}
	// Los que vienen del ciclo anterior y no se anotaron aqui. La edad se cuenta
	// CONTANDO ESTE ciclo, igual que SinAuditarDesdeHace: el valor que llega es
	// lo que llevaba al cerrar el anterior.
	for id, edad := range p.DelCicloAnterior().Abiertos {
		if deEsteCiclo[id] {
			continue
		}
		arrastrados = append(arrastrados, Elemento{
			Clave: id,
			Que:   "anotado en un ciclo anterior",
			Nota: fmt.Sprintf("lleva %d ciclos abierto contando este; se cierra en el programa "+
				"donde se anoto", edad+1),
		})
	}
	return Reparto{
		Rotulo:   repartoDeHallazgos,
		Universo: len(p.Hallazgos()) + len(arrastrados),
		DeDonde: "los hallazgos anotados en este programa mas los que el ciclo anterior dejo " +
			"abiertos",
		Cifras: []Cifra{
			// Ninguno lleva descargo, y se dice por que: un hallazgo lo escribio
			// un auditor sobre algo que miro. No es plazum acusando a nadie de
			// un dato que no tiene, que es lo que la frase existe para evitar.
			recuento(cuboHallazgoAbierto, esteAbierto),
			recuento(cuboHallazgoArrastrado, arrastrados),
			recuento(cuboHallazgoCerrado, esteCerrado),
		},
	}
}

func repartoArrastreAnterior(p *auditoria.Programa) Reparto {
	arr := p.DelCicloAnterior()
	var sigue, salidas []Elemento
	for k, edad := range arr.SinAuditar {
		sigue = append(sigue, Elemento{
			Clave: k,
			Que:   "sigue en el alcance",
			Nota:  fmt.Sprintf("llevaba %d ciclos sin auditarse al cerrar el ciclo anterior", edad),
		})
	}
	for _, k := range arr.Salidas {
		salidas = append(salidas, Elemento{Clave: k, Que: "ya no esta en el alcance"})
	}
	return Reparto{
		Rotulo:   repartoDeArrastre,
		Universo: len(arr.SinAuditar) + len(arr.Salidas),
		DeDonde:  "auditoria.Programa.DelCicloAnterior(), contrastado contra el alcance de ahora",
		Cifras: []Cifra{
			recuento(cuboSigueEnAlcance, sigue),
			noEsCulpa(cuboSalioDelAlcance, salidas, descargoSalidaAlcance),
		},
	}
}

func repartoAsignaciones(p *auditoria.Programa, responsables map[string]string) Reparto {
	enAlcance := map[string]bool{}
	for _, u := range p.Unidades() {
		enAlcance[u.Clave()] = true
	}
	var casa, noCasa []Elemento
	for unidad, quien := range responsables {
		el := Elemento{Clave: unidad, Que: "responde " + quien}
		if enAlcance[unidad] {
			casa = append(casa, el)
		} else {
			noCasa = append(noCasa, el)
		}
	}
	return Reparto{
		Rotulo:   repartoDeAsignaciones,
		Universo: len(responsables),
		DeDonde: "el mapa unidad -> responsable que aporta quien llama (SCIM, asignacion " +
			"manual). Casa por paquete|obligacion",
		Cifras: []Cifra{
			recuento(cuboAsignacionCasa, casa),
			noEsCulpa(cuboAsignacionNoCasa, noCasa, descargoAsignacion),
		},
	}
}

// repartoIndependencia reparte los pares (sesion, unidad) DISTINTOS.
//
// Distintos porque Auditar no dedupe las unidades de una sesion: una sesion que
// liste dos veces la misma unidad no la audito dos veces, y contarla dos veces
// inflaria el denominador de la unica cifra de esta seccion que habla de
// personas.
func repartoIndependencia(p *auditoria.Programa, responsables map[string]string) Reparto {
	var distinta, misma, sinResponsable []Elemento
	visto := map[string]bool{}
	n := 0
	for _, ses := range p.Sesiones() {
		for _, u := range ses.Unidades {
			k := ses.ID + "|" + u
			if visto[k] {
				continue
			}
			visto[k] = true
			n++
			r := strings.TrimSpace(responsables[u])
			switch {
			case r == "":
				sinResponsable = append(sinResponsable, Elemento{Clave: k, Que: u,
					Nota: "audito " + ses.Auditor + "; no consta quien responde de la unidad"})
			case r == ses.Auditor:
				misma = append(misma, Elemento{Clave: k, Que: u,
					Nota: ses.Auditor + " audito la unidad de la que responde"})
			default:
				distinta = append(distinta, Elemento{Clave: k, Que: u,
					Nota: "audito " + ses.Auditor + "; responde " + r})
			}
		}
	}
	return Reparto{
		Rotulo:   repartoDeIndependencia,
		Universo: n,
		DeDonde:  "las sesiones del programa cruzadas con el mapa de responsables",
		Cifras: []Cifra{
			recuento(cuboAuditorDistinto, distinta),
			noEsCulpa(cuboAuditorResponsable, misma, descargoIndependencia),
			ausencia(cuboSinResponsable, sinResponsable, descargoSinResponsable),
		},
	}
}

// ---------------------------------------------------------------------------
// La seccion de revision de accesos
// ---------------------------------------------------------------------------

func seccionAccesos(e Entradas) Seccion {
	s := Seccion{Fuente: DeLaCampanaDeAccesos}
	c := e.Campana
	if c == nil {
		falta := dePlazum(pfSinCampana)
		s.PorQueFalta = falta.Texto
		s.Parrafos = []Parrafo{falta}
		return s
	}
	s.Aportada = true
	s.Parrafos = []Parrafo{dePlazum(pfCampana, c.ID(), c.Sello(), c.Instantanea().Hash)}
	s.Repartos = []Reparto{repartoAccesos(c, e.ConNombresDelCenso), repartoLineas(c)}
	return s
}

func repartoAccesos(c *accesos.Campana, conNombres bool) Reparto {
	cubos := map[accesos.Estado][]Elemento{}
	for _, f := range c.Instantanea().Filas {
		k := f.Clave()
		el := Elemento{Clave: k}
		if conNombres {
			el.Que = f.Rotulo
		}
		if r, tiene := c.RevisorDe(k); tiene {
			el.Nota = "revisor asignado: " + r
		} else {
			el.Nota = "sin revisor asignado"
		}
		if d, hay, err := c.Vigente(k); hay {
			// El cubo ya dice el veredicto, asi que la nota dice quien y cuando.
			// Y el motivo va detras de su autor, como toda prosa ajena.
			el.Nota = fmt.Sprintf("decidido por %s el %s", d.Quien,
				d.Cuando.Format("2006-01-02"))
			if d.A != "" {
				el.Nota += ", delegado a " + d.A
			}
			if strings.TrimSpace(d.Motivo) != "" {
				el.Nota += ": " + d.Motivo
			}
			if err != nil {
				el.Nota += " (EMPATE: " + err.Error() + ")"
			}
		}
		est := c.EstadoDe(k)
		cubos[est] = append(cubos[est], el)
	}
	r := Reparto{
		Rotulo:   repartoDeAccesos,
		Universo: len(c.Instantanea().Filas),
		DeDonde:  "las filas legibles de la instantanea sellada",
	}
	for _, est := range accesos.EstadosPosibles() {
		if est.Termina() {
			r.Cifras = append(r.Cifras, recuento(cuboDeEstado(est), cubos[est]))
			continue
		}
		// Delegada y sin revisar son las dos formas de "aqui no consta que
		// nadie lo mirara": delegar traslada la revision, no la termina.
		r.Cifras = append(r.Cifras, ausencia(cuboDeEstado(est), cubos[est], descargoNoRevisado))
	}
	return r
}

// repartoLineas reparte las LINEAS DE DATOS del fichero, no las filas.
//
// Una ilegible explica TODO SU RANGO, que es lo que hace que una comilla sin
// cerrar no se lleve dos personas por delante sin que nadie las eche de menos.
// Por eso aqui cada linea del rango es un elemento propio: un cubo que contara
// rangos diria "1 ilegible" donde faltan cuatro accesos.
func repartoLineas(c *accesos.Campana) Reparto {
	ins := c.Instantanea()
	inf := c.Informar()
	sinExcusar := map[int]bool{}
	for _, l := range inf.IlegiblesSinExcusar {
		sinExcusar[l] = true
	}
	// Que excusa cubre cada linea, para poder decir quien la escribio. Si dos
	// excusas se solapan manda la primera que se escribio, que es la que
	// respondio de la linea cuando nadie mas lo habia hecho.
	excusaDe := map[int]accesos.Excusa{}
	for _, ex := range inf.Excusas {
		for l := ex.Desde; l <= ex.Hasta; l++ {
			if _, ya := excusaDe[l]; !ya {
				excusaDe[l] = ex
			}
		}
	}
	var legibles, duplicadas, excusadas, bloquean []Elemento
	for _, f := range ins.Filas {
		legibles = append(legibles, Elemento{Clave: linea(f.Linea), Que: f.Clave()})
	}
	for _, d := range ins.Duplicadas {
		duplicadas = append(duplicadas, Elemento{Clave: linea(d.Linea), Que: d.Clave,
			Nota: fmt.Sprintf("ya aparecia en la linea %d", d.Primera)})
	}
	for _, il := range ins.Ilegibles {
		for l := il.Desde; l <= il.Hasta; l++ {
			el := Elemento{Clave: linea(l), Que: "no se pudo leer como un acceso", Nota: il.Motivo}
			if sinExcusar[l] {
				bloquean = append(bloquean, el)
				continue
			}
			if ex, hay := excusaDe[l]; hay {
				el.Nota += fmt.Sprintf("; excusada por %s el %s: %s", ex.Quien,
					ex.Cuando.Format("2006-01-02"), ex.Motivo)
			}
			excusadas = append(excusadas, el)
		}
	}
	return Reparto{
		Rotulo:   repartoDeLineas,
		Universo: ins.LineasDeDatos,
		DeDonde:  "censo.Instantanea.LineasDeDatos, contadas al leer el fichero",
		Cifras: []Cifra{
			recuento(cuboLineaLegible, legibles),
			recuento(cuboLineaDuplicada, duplicadas),
			noEsCulpa(cuboLineaExcusada, excusadas, descargoIlegible),
			ausencia(cuboLineaBloquea, bloquean, descargoIlegible),
		},
	}
}

// linea da la clave de una linea con relleno para que ordene como numero. Sin el
// relleno, la linea 10 iria antes que la 9 y un acta con las filas desordenadas
// se lee como un acta con las filas mal.
func linea(n int) string { return fmt.Sprintf("linea-%08d", n) }

// ---------------------------------------------------------------------------
// La seccion de incidentes
// ---------------------------------------------------------------------------

func seccionIncidentes(e Entradas) Seccion {
	s := Seccion{Fuente: DeLosIncidentes}
	if !e.HayRegistroDeIncidentes {
		falta := dePlazum(pfSinIncidentes)
		s.PorQueFalta = falta.Texto
		s.Parrafos = []Parrafo{falta}
		return s
	}
	s.Aportada = true
	dentro, fuera := repartirPorPeriodo(e)
	s.Parrafos = []Parrafo{dePlazum(pfVerboDelPeriodo)}
	s.Repartos = []Reparto{
		repartoPeriodo(e, dentro, fuera),
		repartoClasificacion(e, dentro),
	}
	casan, sueltas := repartirEsperadas(e, dentro)
	s.Repartos = append(s.Repartos,
		repartoEsperadas(e, casan, sueltas),
		repartoRemision(e, dentro, casan))
	return s
}

// repartirPorPeriodo separa por el eje del REGISTRO (primer conocimiento).
func repartirPorPeriodo(e Entradas) (dentro, fuera []*incidente.Incidente) {
	for _, in := range e.Incidentes {
		// ok es siempre true: Componer ya rechazo los incidentes sin apertura,
		// asi que aqui no hay una tercera rama que tratar en silencio.
		sc, _ := in.PrimerConocimiento()
		if e.Periodo.Cubre(sc) {
			dentro = append(dentro, in)
		} else {
			fuera = append(fuera, in)
		}
	}
	return dentro, fuera
}

func repartoPeriodo(e Entradas, dentro, fuera []*incidente.Incidente) Reparto {
	var ed, ef []Elemento
	for _, in := range dentro {
		sc, _ := in.PrimerConocimiento()
		oc, _ := in.Ocurrio()
		ed = append(ed, Elemento{Clave: in.ID(), Que: "conocido en el periodo",
			Nota: fmt.Sprintf("ocurrio el %s, se supo el %s", oc.Format("2006-01-02"),
				sc.Format("2006-01-02"))})
	}
	for _, in := range fuera {
		sc, _ := in.PrimerConocimiento()
		ef = append(ef, Elemento{Clave: in.ID(), Que: "conocido fuera del periodo",
			Nota: "se supo el " + sc.Format("2006-01-02")})
	}
	return Reparto{
		Rotulo:   repartoDeIncidentes,
		Universo: len(e.Incidentes),
		DeDonde:  "los incidentes que aporta quien llama, situados por su primer conocimiento",
		Cifras: []Cifra{
			recuento(cuboIncidenteDentro, ed),
			noEsCulpa(cuboIncidenteFuera, ef, descargoFueraPeriodo),
		},
	}
}

func repartoClasificacion(e Entradas, dentro []*incidente.Incidente) Reparto {
	var con, empatados, sin []Elemento
	for _, in := range dentro {
		// CON LOS OJOS DEL FINAL DEL PERIODO. Una reclasificacion posterior no
		// reescribe el acta de un periodo cerrado.
		//
		// No se usa Campo("clasificacion_vigente") porque aquella funcion junta
		// el empate y la ausencia en un mismo "no consta", y aqui son dos cubos
		// distintos: uno es que no hay dato y el otro es que hay dos y se
		// contradicen.
		clase, empate, ok := in.ClaseEn(e.Periodo.Hasta)
		switch {
		case ok && !empate:
			con = append(con, Elemento{Clave: in.ID(), Que: clase})
		case ok && empate:
			empatados = append(empatados, Elemento{Clave: in.ID(),
				Que: "dos clasificaciones distintas en el mismo instante"})
		default:
			sin = append(sin, Elemento{Clave: in.ID(), Que: "sin clasificacion que conste"})
		}
	}
	return Reparto{
		Rotulo:   repartoDeClasificacion,
		Universo: len(dentro),
		DeDonde:  "el cubo \"conocido dentro del periodo\" del reparto anterior",
		Cifras: []Cifra{
			recuento(cuboConClasificacion, con),
			noEsCulpa(cuboClasificacionEmpate, empatados, descargoEmpate),
			ausencia(cuboSinClasificacion, sin, descargoNoClasificado),
		},
	}
}

// repartirEsperadas casa las notificaciones que espera el corpus con los
// incidentes DEL PERIODO, por Incidente.ID.
//
// La direccion que se olvida es la segunda: una esperada que no casa con nada.
// Si se tragara, el denominador de "cuantas notificaciones tocaban" saldria mas
// pequeno de lo que es y el acta diria que se cubrio mas de lo que se cubrio.
func repartirEsperadas(e Entradas, dentro []*incidente.Incidente) (casan, sueltas []Notificacion) {
	delPeriodo := map[string]bool{}
	for _, in := range dentro {
		delPeriodo[in.ID()] = true
	}
	for _, n := range e.Esperadas {
		if delPeriodo[n.Incidente] {
			casan = append(casan, n)
		} else {
			sueltas = append(sueltas, n)
		}
	}
	return casan, sueltas
}

func repartoEsperadas(e Entradas, casan, sueltas []Notificacion) Reparto {
	var ec, es []Elemento
	for _, n := range casan {
		ec = append(ec, Elemento{Clave: n.Clave(), Que: n.Que})
	}
	for _, n := range sueltas {
		es = append(es, Elemento{Clave: n.Clave(), Que: n.Que,
			Nota: "el incidente " + n.Incidente + " no es de los que cuenta este acta"})
	}
	return Reparto{
		Rotulo:   repartoDeEsperadas,
		Universo: len(e.Esperadas),
		DeDonde:  "las obligaciones notificatorias que aporta quien llama, casadas por id de incidente",
		Cifras: []Cifra{
			recuento(cuboEsperadaCasa, ec),
			noEsCulpa(cuboEsperadaNoCasa, es, descargoNotificacion),
		},
	}
}

func repartoRemision(e Entradas, dentro []*incidente.Incidente, casan []Notificacion) Reparto {
	porID := map[string]*incidente.Incidente{}
	for _, in := range dentro {
		porID[in.ID()] = in
	}
	var enPlazo, fuera, noConsta []Elemento
	for _, n := range casan {
		el := Elemento{Clave: n.Clave(), Que: n.Que}
		cuando, consta := porID[n.Incidente].Notificado(n.Hito)
		switch {
		case !consta:
			noConsta = append(noConsta, el)
		case e.Periodo.Cubre(cuando):
			el.Nota = "consta remitida el " + cuando.Format("2006-01-02")
			enPlazo = append(enPlazo, el)
		default:
			// NI HECHA NI NO CONSTATADA: consta, y fuera. Meterla en cualquiera
			// de los otros dos cubos miente en una direccion o en la otra, y la
			// cara mala es la que empuja a notificar otra vez a un supervisor.
			el.Nota = "consta remitida el " + cuando.Format("2006-01-02")
			fuera = append(fuera, el)
		}
	}
	return Reparto{
		Rotulo:   repartoDeRemision,
		Universo: len(casan),
		DeDonde:  "el cubo \"casa con un incidente del periodo\" del reparto anterior",
		Cifras: []Cifra{
			recuento(cuboRemitidaDentro, enPlazo),
			noEsCulpa(cuboRemitidaFuera, fuera, descargoRemisionFuera),
			ausencia(cuboNoConstaRemitida, noConsta, descargoNoNotificado),
		},
	}
}

// ---------------------------------------------------------------------------
// La seccion que plazum no escribe
// ---------------------------------------------------------------------------

func seccionDireccion(e Entradas) Seccion {
	s := Seccion{Fuente: DeLaDireccion}
	if len(e.Decisiones) == 0 {
		falta := dePlazum(pfPlazumNoEscribe)
		s.PorQueFalta = falta.Texto
		s.Parrafos = []Parrafo{falta}
		return s
	}
	s.Aportada = true
	s.Parrafos = append([]Parrafo(nil), e.Decisiones...)
	return s
}
