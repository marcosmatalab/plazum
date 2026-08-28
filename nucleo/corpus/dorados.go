package corpus

// El ejecutor de dorados: convierte la Temporalidad declarada del paquete en
// la primitiva del motor de ventana, calcula, y compara con el esperado que se
// derivo del texto legal. Si discrepan, GANA EL DORADO y se arregla el motor.
//
// Calendario: v1 usa UTC sin festivos (los calendarios por paquete llegan con
// su propia seccion del corpus). Los dorados que dependan de festivos deben
// esperar a esa pieza, y decirlo en su caso.

import (
	"fmt"
	"strings"
	"time"

	"github.com/marcosmatalab/plazum/nucleo/ventana"
)

func regimenDe(r RegimenSpec) (ventana.Regimen, error) {
	reg := ventana.Regimen{Cal: ventana.NuevoCalendario("utc-v1", "dorados", "corpus", time.UTC)}
	switch r.Computo {
	case "naturales", "":
		reg.Comp = ventana.Naturales
	case "habiles":
		reg.Comp = ventana.Habiles
	default:
		return reg, fmt.Errorf("computo %q no reconocido", r.Computo)
	}
	switch r.Cierre {
	case "":
		reg.Cierre = ventana.CierreAuto
	case "exacto":
		reg.Cierre = ventana.CierreExacto
	case "fin_de_dia":
		reg.Cierre = ventana.CierreFinDia
	default:
		return reg, fmt.Errorf("cierre %q no reconocido", r.Cierre)
	}
	switch r.Traslado {
	case "", "ninguno":
		reg.Trasl = ventana.TrasladoNinguno
	case "siguiente_habil":
		reg.Trasl = ventana.TrasladoSiguienteHabil
	default:
		return reg, fmt.Errorf("traslado %q no reconocido", r.Traslado)
	}
	return reg, nil
}

func parseFecha(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	return time.Parse("2006-01-02", s)
}

// filaEsperada es una fila del esperado ya leida: el hito, el estado y la fecha
// si la hay.
type filaEsperada struct {
	Hito   string
	Estado ventana.EstadoVenc
	Vence  time.Time
	TieneF bool
}

// leerEsperado traduce el conjunto declarado y devuelve ademas el HORIZONTE.
//
// EL HORIZONTE ES LA ULTIMA FECHA DECLARADA MAS UN DIA, y el margen no es
// decoracion. `hasta` solo acota la primitiva `periodica` (un plazo lo ignora),
// y ahi decide cuantas ocurrencias devuelve el motor. Con el horizonte clavado
// en la ultima fecha declarada, una ocurrencia que el motor calculara UNAS HORAS
// TARDE se caeria por el borde y el fallo se leeria como "falta el hito
// auditoria#2", que suena a caso mal escrito. Con un dia de margen, esa misma
// ocurrencia entra en la lista y el fallo se lee como lo que es: "el motor dice
// esta fecha y el texto dice esta otra".
//
// El margen tiene su precio y es el bueno: si el motor colara una ocurrencia de
// mas dentro de ese dia, sale por la direccion de "sobra", que es la ruidosa.
// Ninguna cadencia del corpus baja de un mes, asi que hoy no ocurre.
//
// Sin ninguna fecha declarada el horizonte es el cero, y eso solo lo nota una
// `periodica`, que entonces no devuelve nada y deja el dorado en rojo por
// "falta". Es la direccion correcta: un caso de reloj periodico sin ninguna
// fecha no es un caso.
func leerEsperado(d Dorado) ([]filaEsperada, time.Time, error) {
	if len(d.Esperado) == 0 {
		// Defensa en profundidad: el linter ya lo rechaza al cargar, pero
		// EjecutarDorado tambien se llama con dorados construidos a mano, y un
		// esperado vacio saldria verde comparando el vacio con el vacio.
		return nil, time.Time{}, fmt.Errorf("dorado %q: esperado sin ninguna fila. "+
			"Un caso que no declara ningun vencimiento no afirma nada", d.Caso)
	}
	var out []filaEsperada
	var horizonte time.Time
	vistos := map[string]bool{}
	for i, esp := range d.Esperado {
		if esp.Hito == "" {
			return nil, time.Time{}, fmt.Errorf("dorado %q: la fila %d del esperado no dice "+
				"hito, y el emparejamiento con el motor se hace por hito", d.Caso, i+1)
		}
		if vistos[esp.Hito] {
			return nil, time.Time{}, fmt.Errorf("dorado %q: el hito %q sale dos veces en el "+
				"esperado; una de las dos filas no se compararia contra nada", d.Caso, esp.Hito)
		}
		vistos[esp.Hito] = true
		f := filaEsperada{Hito: esp.Hito, Estado: ventana.Determinado}
		if esp.Estado != "" {
			est, err := ventana.ParseEstadoVenc(esp.Estado)
			if err != nil {
				return nil, time.Time{}, fmt.Errorf("dorado %q, hito %q: %w", d.Caso, esp.Hito, err)
			}
			f.Estado = est
		}
		if esp.Vence != "" {
			t, err := parseFecha(esp.Vence)
			if err != nil {
				return nil, time.Time{}, fmt.Errorf("dorado %q, hito %q: vence ilegible: %w",
					d.Caso, esp.Hito, err)
			}
			f.Vence, f.TieneF = t, true
			if t.After(horizonte) {
				horizonte = t
			}
		}
		if f.Estado == ventana.Determinado && !f.TieneF {
			return nil, time.Time{}, fmt.Errorf("dorado %q, hito %q: determinado sin fecha", d.Caso, esp.Hito)
		}
		if f.Estado != ventana.Determinado && f.TieneF {
			return nil, time.Time{}, fmt.Errorf("dorado %q, hito %q: estado %s con fecha; "+
				"un vencimiento que no esta determinado no tiene fecha", d.Caso, esp.Hito, f.Estado)
		}
		out = append(out, f)
	}
	if !horizonte.IsZero() {
		horizonte = horizonte.Add(24 * time.Hour)
	}
	return out, horizonte, nil
}

// ConsumeElHorizonte dice si una primitiva mira el parametro `hasta`.
//
// Hoy solo `periodica`: `Plazo.Vencimientos` y `Puntual.Vencimientos` lo
// ignoran en su firma. Vive en una funcion y no repartido en condicionales para
// que el dia que otra primitiva lo consuma cambie un sitio y no tres, que es
// como este proyecto acabo con dos traductores del mismo reloj.
func ConsumeElHorizonte(primitiva string) bool { return primitiva == "periodica" }

// EjecutarDorado calcula el reloj de la obligacion con los hechos del caso y
// compara el conjunto ENTERO de vencimientos con el esperado. Error =
// discrepancia motor/texto, y gana el dorado.
//
// SE COMPARAN LAS DOS DIRECCIONES. Ni un vencimiento de menos (la norma da un
// plazo que el motor no ensena) ni uno de mas (el motor ensena un plazo que la
// norma no da). La segunda es la que un dorado no sabia decir hasta hoy, y es
// la que muerde: dos fechas para la misma obligacion es peor que ninguna,
// porque el operador no tiene forma de saber cual es la suya.
//
// EL EMPAREJAMIENTO ES POR EL HITO (invariante 7). Es una identidad DENTRO del
// dato: el `id` que el paquete le da al hito en su temporalidad, que es el mismo
// que el motor copia en Vencimiento.Hito. No es el indice ni el orden, que aqui
// ademas cambia solo (Plazo.Vencimientos ordena por fecha, asi que corregir un
// plazo reordena la lista entera). Nadie firma un orden.
func EjecutarDorado(o Obligacion, d Dorado) error {
	if o.Temporalidad == nil {
		return fmt.Errorf("dorado %q: la obligacion %s no declara temporalidad", d.Caso, o.ID)
	}
	tmp := *o.Temporalidad
	filas, horizonte, err := leerEsperado(d)
	if err != nil {
		return err
	}
	// TODOS los hechos del dorado, no solo el disparador.
	//
	// Hasta el 26-08-2026 aqui solo se leia el disparador, asi que un dorado no
	// podia expresar ni el cumplimiento de un hito previo (que es de donde
	// cuenta la notificacion intermedia) ni la clasificacion del incidente (que
	// es lo que decide el limite). Los dos son hechos con su instante, asi que
	// caben en el mismo mapa y no hacia falta un tipo nuevo: lo que faltaba era
	// pasarlos.
	todosLosHechos := func() (ventana.Hechos, error) {
		h := ventana.Hechos{}
		for k, v := range d.Hechos {
			t, err := parseFecha(v)
			if err != nil {
				return nil, fmt.Errorf("el hecho %q no es una fecha legible: %w", k, err)
			}
			h[k] = t
		}
		return h, nil
	}

	// UNA SOLA TRADUCCION, la misma que usa el producto (VencimientosDe). Aqui
	// habia una copia con su propia tabla de regimenes, y por eso un dorado
	// podia pasar contra una deriva que el producto no hacia.
	hechos, err := todosLosHechos()
	if err != nil {
		return fmt.Errorf("dorado %q: %w", d.Caso, err)
	}
	if _, ok := hechos[tmp.Disparador["hecho"]]; !ok && tmp.Primitiva != "puntual" {
		return fmt.Errorf("dorado %q: falta el hecho %q, que es el disparador de la obligacion",
			d.Caso, tmp.Disparador["hecho"])
	}
	// LA VENTANA DECLARADA MANDA SOBRE LA DERIVADA, y esa inversion es el
	// arreglo entero. Ver Dorado.Hasta: un horizonte que sale de las fechas
	// declaradas se mueve cuando alguien trunca la lista, y entonces la
	// direccion de "sobra" deja de existir para la primitiva periodica.
	if d.Hasta != "" {
		h, err := parseFecha(d.Hasta)
		if err != nil {
			return fmt.Errorf("dorado %q: `hasta` ilegible (%q): %w", d.Caso, d.Hasta, err)
		}
		horizonte = h
	}
	vs, err := VencimientosDe(o, hechos, horizonte)
	if err != nil {
		return fmt.Errorf("dorado %q: %w", d.Caso, err)
	}

	// El indice del motor, POR HITO. Si el motor devolviera dos vencimientos
	// con el mismo hito, uno taparia al otro en el mapa y la comprobacion se
	// haria sobre el que ganara la carrera: se dice en voz alta.
	delMotor := map[string]ventana.Vencimiento{}
	for _, v := range vs {
		if _, ya := delMotor[v.Hito]; ya {
			return fmt.Errorf("dorado %q: el motor devuelve el hito %q dos veces, asi que "+
				"emparejar por hito dejaria uno de los dos sin comprobar. Arreglo: los ids "+
				"de hito de una obligacion son unicos, revisar la temporalidad de %s",
				d.Caso, v.Hito, o.ID)
		}
		delMotor[v.Hito] = v
	}

	var fallos []string
	// DIRECCION 1: lo que la norma da y el motor no ensena.
	declarados := map[string]bool{}
	for _, f := range filas {
		declarados[f.Hito] = true
		v, ok := delMotor[f.Hito]
		if !ok {
			fallos = append(fallos, fmt.Sprintf("FALTA el hito %q, que el texto exige. "+
				"El motor solo devuelve %v", f.Hito, hitosDelMotor(vs)))
			continue
		}
		if v.Estado != f.Estado {
			fallos = append(fallos, fmt.Sprintf("el hito %q sale como %q y el texto dice %q "+
				"(regla del motor: %s)", f.Hito, v.Estado, f.Estado, v.Regla))
			continue
		}
		if f.Estado != ventana.Determinado {
			continue // sin fecha que comparar: el estado ERA la afirmacion
		}
		if !v.Vence.Equal(f.Vence) {
			fallos = append(fallos, fmt.Sprintf("el hito %q: el motor dice %s y el texto dice %s",
				f.Hito, v.Vence.Format(time.RFC3339), f.Vence.Format(time.RFC3339)))
		}
	}
	// DIRECCION 2, la que un dorado no sabia decir: lo que el motor ensena y la
	// norma no da. Es la que muerde, porque dos fechas para la misma obligacion
	// dejan al operador sin saber cual es la suya.
	if d.SubconjuntoPorque == "" {
		for _, v := range vs {
			if declarados[v.Hito] {
				continue
			}
			fallos = append(fallos, fmt.Sprintf("SOBRA el hito %q (%s%s), que el texto no da. "+
				"Regla del motor: %s", v.Hito, v.Estado, fechaSiLaHay(v), v.Regla))
		}
	}
	if len(fallos) == 0 {
		return nil
	}
	nota := ""
	if d.SubconjuntoPorque != "" {
		nota = fmt.Sprintf("\n  (este caso declara un SUBCONJUNTO: %s. Solo se comprueba que no "+
			"falte ninguna de las filas declaradas)", d.SubconjuntoPorque)
	}
	return fmt.Errorf("dorado %q (obligacion %s): %d discrepancias entre el motor y el texto.\n"+
		"  %s\n  Cita del esperado: %s%s\n"+
		"  GANA EL DORADO: si el texto dice esto, se arregla el motor o el paquete, no el caso",
		d.Caso, o.ID, len(fallos), strings.Join(fallos, "\n  "), d.CitaDelEsperado, nota)
}

// hitosDelMotor lista los hitos que devolvio el motor, para que el error de
// "falta" diga contra que se comparo. Se llama asi y no hitosDe porque hitosDe
// traduce los hitos DECLARADOS por el paquete: son dos cosas distintas y
// compartir nombre seria pedir una confusion.
func hitosDelMotor(vs []ventana.Vencimiento) []string {
	out := make([]string, 0, len(vs))
	for _, v := range vs {
		out = append(out, v.Hito)
	}
	return out
}

func fechaSiLaHay(v ventana.Vencimiento) string {
	if v.Vence.IsZero() {
		return ""
	}
	return " " + v.Vence.Format(time.RFC3339)
}

// EjecutarDorados corre todos los dorados de un paquete.
func EjecutarDorados(p *Paquete) []error {
	idx := map[string]Obligacion{}
	for _, o := range p.Obligaciones {
		idx[o.ID] = o
	}
	var errs []error
	for _, d := range p.Dorados {
		if err := EjecutarDorado(idx[d.Obligacion], d); err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

// hitosDe traduce los hitos declarados por el paquete a los del motor.
//
// El limite vacio o "indeterminado" produce una Duracion indeterminada, que es
// como el motor dice "la norma exige la accion y no fija plazo". Es el caso de
// la notificacion inicial de la tabla 3 del anexo del RD 43/2021, que dice
// "Inmediata": hay obligacion y no hay numero, y las dos cosas son ciertas a la
// vez. Devolver ahi cero horas seria inventarse un plazo que la norma no da.
func hitosDe(specs []HitoSpec, reg ventana.Regimen) ([]ventana.Hito, error) {
	out := make([]ventana.Hito, 0, len(specs))
	vistos := map[string]bool{}
	for i, h := range specs {
		if h.ID == "" {
			return nil, fmt.Errorf("el hito %d no tiene id, asi que no se puede referenciar "+
				"desde desde_hito ni desde el esperado de un dorado", i)
		}
		if vistos[h.ID] {
			return nil, fmt.Errorf("el hito %q esta declarado dos veces: desde_hito no sabria "+
				"a cual de los dos apunta", h.ID)
		}
		vistos[h.ID] = true
		lim, err := limiteDe(h.Limite)
		if err != nil {
			return nil, fmt.Errorf("hito %q: %w", h.ID, err)
		}
		// El regimen del hito, si lo declara; si no, el de la obligacion. Una
		// notificacion escalonada mezcla horas (instante exacto, sin traslado)
		// con meses (fin de dia, con traslado): ver HitoSpec.Regimen.
		reg := reg
		if h.Regimen != nil {
			propio, err := regimenDe(*h.Regimen)
			if err != nil {
				return nil, fmt.Errorf("hito %q: regimen propio: %w", h.ID, err)
			}
			reg = propio
		}
		hito := ventana.Hito{ID: h.ID, Limite: lim, Reg: reg,
			DesdeHito: h.DesdeHito, Clase: h.Clase, Nota: h.Nota}
		if h.Tope != nil {
			lt, err := ventana.ParseDuracion(h.Tope.Limite)
			if err != nil {
				return nil, fmt.Errorf("hito %q, tope desde %q: %w", h.ID, h.Tope.Desde, err)
			}
			hito.Tope = &ventana.Tope{Desde: h.Tope.Desde, Limite: lt, Reg: reg,
				Caduca: h.Tope.Caduca, Cita: h.Tope.Cita}
		}
		for _, a := range h.Alternativas {
			al, err := limiteDe(a.Limite)
			if err != nil {
				return nil, fmt.Errorf("hito %q, lectura %q: %w", h.ID, a.ID, err)
			}
			hito.Alternativas = append(hito.Alternativas,
				ventana.Lectura{ID: a.ID, Limite: al, Reg: reg, Cita: a.Cita})
		}
		out = append(out, hito)
	}
	// desde_hito tiene que apuntar a un hito QUE EXISTE. Un encadenamiento a un
	// nombre mal escrito deja ese hito pendiente para siempre, y eso se lee
	// como "el reloj no ha arrancado" en vez de como el error que es.
	for _, h := range specs {
		if h.DesdeHito != "" && !vistos[h.DesdeHito] {
			return nil, fmt.Errorf("el hito %q cuenta desde %q, que no existe en esta "+
				"obligacion. Un encadenamiento a un nombre que no esta deja el hito pendiente "+
				"para siempre y se lee como si el reloj no hubiera arrancado", h.ID, h.DesdeHito)
		}
	}
	return out, nil
}

// limiteDe admite el vacio y la palabra "indeterminado" como "la norma no fija
// plazo".
func limiteDe(s string) (ventana.Duracion, error) {
	if s == "" || s == "indeterminado" {
		return ventana.Duracion{Indeterminado: true}, nil
	}
	return ventana.ParseDuracion(s)
}

// VencimientosDe traduce el reloj declarado por un paquete a vencimientos del
// motor. ES LA UNICA TRADUCCION QUE HAY, y esa es toda la gracia.
//
// POR QUE ESTA AQUI Y NO EN cmd/plazum. Hasta el 26-08-2026 habia DOS: una en
// el ejecutor de dorados y otra en `cmd/plazum/demo.go`, con la misma tabla de
// regimenes copiada en las dos. El comentario de aquella decia:
//
//	El dia que las dos derivas se separen, el test se pone rojo. Anotado como
//	P1 en docs/pendientes.md: el sitio correcto para esto es una funcion
//	exportada de nucleo/corpus.
//
// Ese dia fue el que se escribio el primer plazo ESCALONADO (nis1-es, tabla 3
// del anexo del RD 43/2021): se enseno a una de las dos a leer los hitos
// encadenados y la otra se quedo atras, y el test se puso rojo exactamente como
// estaba escrito que pasaria.
//
// Es la misma familia que los dos parsers de ASN.1 del adaptador de sellado:
// dos lecturas independientes del mismo dato que no ata ninguna identidad. Aqui
// no era explotable, solo caro, pero el arreglo es el mismo: quedarse con una.
//
// `hasta` acota las primitivas que generan ocurrencias (periodica); las demas
// lo ignoran.
func VencimientosDe(o Obligacion, hechos ventana.Hechos, hasta time.Time) ([]ventana.Vencimiento, error) {
	if o.Temporalidad == nil {
		return nil, fmt.Errorf("la obligacion %s no declara reloj", o.ID)
	}
	t := *o.Temporalidad
	reg, err := regimenDe(t.Regimen)
	if err != nil {
		return nil, fmt.Errorf("obligacion %s: %w", o.ID, err)
	}
	disparador := t.Disparador["hecho"]

	switch t.Primitiva {
	case "puntual":
		// La fecha la fija la NORMA, no un hecho. Por eso esta rama no mira
		// `hechos` ni una sola vez, y por eso el ejecutor de dorados exime a
		// `puntual` de traer el disparador.
		if t.En == "" {
			return nil, fmt.Errorf("obligacion %s: primitiva puntual sin `en`. Una fecha que "+
				"fija la norma se escribe entera (2026-12-02T23:59:59Z), con su hora, porque "+
				"una puntual no tiene regimen y no sabe cerrar el dia", o.ID)
		}
		en, err := parseFecha(t.En)
		if err != nil {
			return nil, fmt.Errorf("obligacion %s: `en` ilegible (%q): %w", o.ID, t.En, err)
		}
		hito := t.Hito
		if hito == "" {
			hito = "limite"
		}
		return ventana.Puntual{Hito: hito, En: en}.Vencimientos(nil, hasta), nil

	case "periodica":
		cada, err := ventana.ParseDuracion(t.Cadencia)
		if err != nil {
			return nil, fmt.Errorf("obligacion %s: cadencia %q: %w", o.ID, t.Cadencia, err)
		}
		hito := t.Hito
		if hito == "" {
			hito = "ocurrencia"
		}
		base, ok := hechos[disparador]
		if !ok {
			// EL DESCARTE MAS CARO QUE HA TENIDO ESTE MOTOR, y estuvo aqui
			// hasta el 29-08-2026: `return nil, nil`. Una cadencia cuyo hecho
			// de arranque no consta devolvia una lista VACIA, o sea nada: ni
			// fecha, ni fila, ni numero. La obligacion desaparecia del
			// calendario, del expediente y de explain sin dejar rastro.
			//
			// Lo saco la seccion de sentadas al enchufar el reglamento tecnico
			// de NIS2 en un perfil de arranque: 55 relojes alcanzados por la
			// aplicabilidad producian 6 fechas y 3 filas sin fecha, y los otros
			// 46 no estaban en ningun sitio. Y es exactamente contra lo que ya
			// avisaba, escrito, la rama equivalente de `Plazo`: «una lista
			// vacia se leeria como "nada que hacer", que es el peor error
			// posible aqui».
			//
			// Un reloj que espera un dato del operador NO es un reloj que no
			// existe: es el estado normal de TODA obligacion periodica el dia
			// uno de un cliente.
			return []ventana.Vencimiento{{
				Hito: hito, Estado: ventana.PendienteDeHecho,
				Regla: "la cadencia de " + t.Cadencia + " arranca del hecho " + disparador +
					", que todavia no consta. No es que no haya obligacion: es que falta " +
					"la fecha de la ultima vez que se hizo. Arreglo: registrar " + disparador +
					" en el alcance",
			}}, nil
		}
		// LA REAPERTURA POR EVENTO, antes que el ciclo. Si algun hecho de
		// `reabre_por` es POSTERIOR a la ultima ejecucion registrada, el punto
		// vuelve a pedir la revision y ya no manda el calendario: manda el
		// hecho. Ver Temporalidad.ReabrePor.
		if reabre, cual, ok := reaperturaDe(t, hechos, base); ok {
			return []ventana.Vencimiento{{
				Hito: hito, Estado: ventana.SinPlazoLegal,
				Regla: fmt.Sprintf("el hecho %q consta el %s, posterior a la ultima ejecucion "+
					"registrada (%s), asi que el punto REABRE la revision y el ciclo de %s deja "+
					"de mandar. La norma dice CUANDO hay que revisar (al ocurrir el hecho) y NO "+
					"da plazo para hacerlo, asi que aqui no hay fecha limite: lo que se mide es "+
					"el tiempo transcurrido desde el hecho. Se cierra registrando %s",
					cual, reabre.Format("2006-01-02"), base.Format("2006-01-02"),
					t.Cadencia, disparador),
			}}, nil
		}
		p := ventana.Periodica{Hito: hito, Desde: base, Cada: cada, Reg: reg}
		return p.Vencimientos(nil, hasta), nil

	case "plazo":
		hitos, err := hitosDelPlazo(o.ID, t, reg)
		if err != nil {
			return nil, err
		}
		p := ventana.Plazo{Disparador: disparador, Hitos: hitos}
		return p.Vencimientos(hechos, hasta), nil

	default:
		return nil, fmt.Errorf("obligacion %s: la primitiva %q todavia no tiene ejecutor "+
			"(llega con su etapa)", o.ID, t.Primitiva)
	}
}

// hitosDelPlazo devuelve los hitos de un plazo, sean escalonados o el simple.
//
// EL ORDEN IMPORTA Y COSTO UN ROJO: cuando el paquete declara `hitos`, el
// `limite` suelto de arriba NI SIQUIERA SE MIRA. La primera version lo parseaba
// antes, asi que un paquete escalonado (que no tiene limite suelto, porque cada
// hito lleva el suyo) moria con "duracion no reconocida" sobre la cadena vacia.
func hitosDelPlazo(id string, t Temporalidad, reg ventana.Regimen) ([]ventana.Hito, error) {
	if len(t.Hitos) > 0 {
		hs, err := hitosDe(t.Hitos, reg)
		if err != nil {
			return nil, fmt.Errorf("obligacion %s: %w", id, err)
		}
		return hs, nil
	}
	lim, err := ventana.ParseDuracion(t.Limite)
	if err != nil {
		return nil, fmt.Errorf("obligacion %s: limite %q: %w", id, t.Limite, err)
	}
	hito := t.Hito
	if hito == "" {
		hito = "limite"
	}
	return []ventana.Hito{{ID: hito, Limite: lim, Reg: reg}}, nil
}

// reaperturaDe devuelve el hecho de `reabre_por` mas reciente que sea POSTERIOR
// a la base del ciclo, si lo hay.
//
// «Posterior» es estricto: un hecho del mismo instante que la ultima ejecucion
// NO reabre nada. Si la revision se hizo el mismo dia del incidente, se hizo
// despues de el (por eso consta), y tratarlo como reapertura pediria repetirla
// para siempre: cada vez que se registrara la nueva revision, el incidente
// seguiria empatando con ella. Un bucle de ese tipo no da error, da una
// obligacion que no se puede cerrar nunca.
func reaperturaDe(t Temporalidad, hechos ventana.Hechos, base time.Time) (time.Time, string, bool) {
	var mejor time.Time
	var cual string
	for _, h := range t.ReabrePor {
		e, ok := hechos[h]
		if !ok || !e.After(base) {
			continue
		}
		if cual == "" || e.After(mejor) {
			mejor, cual = e, h
		}
	}
	return mejor, cual, cual != ""
}
