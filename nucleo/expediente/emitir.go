package expediente

// EL PRODUCTOR QUE NO EXISTIA.
//
// # El agujero, medido el 11-09-2026
//
// `plazum verify`, `plazum explain` y `plazum export` LEEN un expediente. Ninguno
// escribe. En todo el repositorio habia **un solo** sitio que construyera un
// `Expediente`, y era `herramientas/generardemo`, que lo monta desde un fichero
// de escenario escrito a mano: cuatro hitos, cinco aplicables y dos estados,
// todos tecleados.
//
// O sea que el diferenciador del producto —lo que `docs/modelo-de-amenaza.md`
// promete, «un tercero recalcula desde cero y obtiene exactamente lo mismo, o le
// dice donde no coincide»— estaba construido, probado contra si mismo, y un
// comprador no podia generar el suyo.
//
// Y el precio de tener un solo consumidor ya se estaba cobrando: el verificador
// leia el cierre del regimen con `case "fin_dia"` y el corpus escribe
// `fin_de_dia` 196 veces. La cadena `fin_dia` no existia en ningun otro sitio del
// arbol. Nadie lo vio porque el unico fixture que alimentaba ese codigo escribia
// `auto` y `exacto`, que son las dos unicas cadenas que aquel `switch` acertaba
// —una por tener `case` y la otra porque el valor cero coincidia—. **Un motor con
// un solo consumidor no esta probado, esta de acuerdo consigo mismo.**
//
// # Lo que este fichero hace, y lo que deliberadamente no
//
// Deriva del CORPUS todo lo que el verificador recalcula, y lo deriva **llamando
// a las mismas funciones**. No hay una segunda implementacion del motor de
// plazos ni del de aplicabilidad: `Emitir` construye los `RelojDeclarado` y luego
// pregunta por sus vencimientos con `construirPlazo`, que es literalmente la que
// usa `Verificar`.
//
// Eso hace que la puerta de ida y vuelta pruebe menos de lo que parece, y se dice
// aqui en vez de dejar que lo parezca: **no prueba que el motor calcule bien**
// (para eso estan los 808 dorados), prueba que el expediente que sale del corpus
// esta COMPLETO y que el vocabulario CRUZA. Que es justo lo que fallaba.
//
// Lo que no emite todavia, contado y no callado: solo se emiten los relojes de
// primitiva `plazo`, que son los que `RelojDeclarado` sabe representar. Las demas
// primitivas (`periodica`, `continua`, `puntual`, `preaviso`, `maximo`) no caben
// en ese tipo, y meterlas a la fuerza seria inventarse una representacion. El
// emisor las CUENTA y devuelve el numero, para que el hueco tenga cardinal.

import (
	"fmt"
	"sort"
	"time"

	"github.com/marcosmatalab/plazum/nucleo/aplicabilidad"
	"github.com/marcosmatalab/plazum/nucleo/corpus"
	"github.com/marcosmatalab/plazum/nucleo/ventana"
)

// Emision son las entradas del emisor: el corpus, lo que la organizacion ha
// contestado, y los dos instantes.
type Emision struct {
	// Paquetes es el corpus cargado. De aqui salen las obligaciones, las reglas
	// de aplicabilidad, los relojes y las vigencias.
	Paquetes []*corpus.Paquete

	// Organizacion y Alcance son de la CUENTA que emite, no de la instalacion
	// (invariante 12). Quien llame tiene que haberlos sacado de una superficie
	// con sesion o de un fichero que el operador senala; este paquete no los
	// adivina.
	Organizacion string
	Alcance      string

	// Hechos son los de la organizacion, para el motor de aplicabilidad.
	Hechos []aplicabilidad.Hecho

	// HechosDelReloj son los instantes que disparan cada obligacion, por
	// obligacion y por nombre de hecho. Una obligacion sin sus hechos NO se
	// inventa un disparador: se queda fuera y se cuenta.
	HechosDelReloj map[string]map[string]time.Time

	// Calendario es el que se declara en el expediente. Sin zona no se puede
	// calcular nada, asi que el valor cero es un error y no UTC por defecto:
	// elegir la zona por el emisor es elegirle el vencimiento al cliente.
	Calendario CalendarioDeclarado

	// Emitido es cuando se escribe; ComoEstaba, a que instante se refiere el
	// estado. Son dos y no uno a proposito, igual que en el resto del producto.
	Emitido    time.Time
	ComoEstaba time.Time
}

// Resumen dice que se emitio y que se quedo fuera, con su cardinal.
type Resumen struct {
	Paquetes             int
	Obligaciones         int
	Relojes              int
	Reclamaciones        int
	Aplicables           int
	SinHechos            int      // obligaciones con reloj de plazo y sin instante que lo dispare
	PrimitivasNoEmitidas int      // relojes que RelojDeclarado no sabe representar
	PrimitivasFuera      []string // cuales, ordenadas
}

// Emitir construye el expediente a partir del corpus y de lo que ha contestado
// una organizacion.
//
// LO VIGILA: TestUnExpedienteEmitidoDelCorpusRealSeVerificaSinDiscrepancias.
func Emitir(em Emision) (*Expediente, Resumen, error) {
	var res Resumen
	if em.Calendario.Zona == "" {
		return nil, res, fmt.Errorf("emitir: falta la zona del calendario. " +
			"No se pone UTC por defecto: elegir la zona es elegirle el vencimiento al cliente")
	}
	if em.ComoEstaba.IsZero() {
		return nil, res, fmt.Errorf("emitir: falta `como estaba`, que es el instante al que " +
			"se refiere el estado. Sin el, el expediente no dice de cuando habla")
	}
	if em.Emitido.IsZero() {
		em.Emitido = em.ComoEstaba
	}

	e := &Expediente{
		Version: Version, Emitido: em.Emitido, ComoEstaba: em.ComoEstaba,
		Organizacion: em.Organizacion, Alcance: em.Alcance,
		Hechos: em.Hechos,
	}

	fueraDePrimitiva := map[string]int{}
	for _, p := range em.Paquetes {
		if p == nil {
			continue
		}
		e.Paquetes = append(e.Paquetes, paqueteDelCorpus(p))

		// Las reglas de aplicabilidad, con el mismo derivador que usa el linter.
		// Un paquete sin reglas aporta programa vacio y eso es legitimo: un
		// referencial no deriva nada.
		if prog, errs := p.Programa(); len(errs) == 0 && len(prog.Reglas) > 0 {
			e.Programas = append(e.Programas, prog)
		} else if len(errs) > 0 {
			return nil, res, fmt.Errorf("emitir: las reglas de %s no se pueden cargar: %v",
				p.URN, errs[0])
		}

		for _, o := range p.Obligaciones {
			e.Obligaciones = append(e.Obligaciones, obligacionDelCorpus(p, o))

			if o.Temporalidad == nil {
				continue
			}
			if o.Temporalidad.Primitiva != "plazo" {
				fueraDePrimitiva[o.Temporalidad.Primitiva]++
				res.PrimitivasNoEmitidas++
				continue
			}
			hechos := em.HechosDelReloj[o.ID]
			if len(hechos) == 0 {
				// SIN HECHO NO HAY RELOJ, y eso NO se rellena con el instante de
				// emision. Un disparador inventado produce una fecha que el
				// cliente se va a creer.
				res.SinHechos++
				continue
			}
			rd, err := relojDelCorpus(o, hechos, em.Calendario)
			if err != nil {
				return nil, res, fmt.Errorf("emitir: el reloj de %s: %w", o.ID, err)
			}
			e.Relojes = append(e.Relojes, rd)
		}
	}
	for k, n := range fueraDePrimitiva {
		res.PrimitivasFuera = append(res.PrimitivasFuera, fmt.Sprintf("%s x%d", k, n))
	}
	sort.Strings(res.PrimitivasFuera)

	// LO QUE EL EMISOR AFIRMA SE DERIVA CON LAS MISMAS FUNCIONES QUE LO
	// RECALCULAN. No se teclea, que es como estaba en el fixture.
	aplicables, err := derivarAplicables(e.Programas, e.Hechos)
	if err != nil {
		return nil, res, fmt.Errorf("emitir: %w", err)
	}
	e.Aplicables = aplicables

	for _, rd := range e.Relojes {
		p, hechos, err := construirPlazo(rd)
		if err != nil {
			return nil, res, fmt.Errorf("emitir: el reloj de %s no se puede construir: %w",
				rd.Obligacion, err)
		}
		for _, v := range p.Vencimientos(hechos, time.Time{}) {
			r := Reclamacion{Obligacion: rd.Obligacion, Hito: v.Hito, Estado: v.Estado.String()}
			if v.Estado == ventana.Determinado {
				r.Vence = v.Vence
			}
			e.Reclamaciones = append(e.Reclamaciones, r)
		}
	}

	// Los digests se calculan sobre el contenido que viaja, que es lo que la
	// capa 2.b de Verificar recalcula.
	e.AnclasDeclaradas = map[string]string{}
	for i := range e.Paquetes {
		d := DigestPaquete(e.Paquetes[i].URN, e.Programas, e.Obligaciones)
		e.Paquetes[i].Digest = d
		e.AnclasDeclaradas[e.Paquetes[i].URN] = d
	}

	res.Paquetes, res.Obligaciones = len(e.Paquetes), len(e.Obligaciones)
	res.Relojes, res.Reclamaciones = len(e.Relojes), len(e.Reclamaciones)
	res.Aplicables = len(e.Aplicables)
	return e, res, nil
}

// paqueteDelCorpus pasa un paquete cargado a su forma en el expediente.
func paqueteDelCorpus(p *corpus.Paquete) Paquete {
	q := Paquete{URN: p.URN, Version: p.Version, Clase: p.Clase.String()}
	if t, err := time.Parse("2006-01-02", p.Vigencia.Desde); err == nil {
		q.Vigencia.Desde = t
	}
	if t, err := time.Parse("2006-01-02", p.Vigencia.Hasta); err == nil {
		q.Vigencia.Hasta = t
	}
	return q
}

// obligacionDelCorpus pasa una obligacion cargada a su forma en el expediente.
func obligacionDelCorpus(p *corpus.Paquete, o corpus.Obligacion) Obligacion {
	q := Obligacion{ID: o.ID, Paquete: p.URN, Articulo: o.Articulo, Afirmacion: o.Titulo}
	if o.Temporalidad != nil {
		q.Primitiva = o.Temporalidad.Primitiva
	}
	return q
}

// relojDelCorpus construye el reloj declarado de una obligacion de plazo.
//
// AQUI ES DONDE CRUZA EL VOCABULARIO, y es el punto exacto que estaba roto: el
// regimen que escribe el paquete se LEE con `ventana.RegimenDesde` y se vuelve a
// escribir con los `String()` de los tipos, o sea contra la misma tabla. Asi una
// palabra que el corpus escriba y el verificador no entienda falla AQUI, al
// emitir, en vez de producir un expediente que el verificador leera mal y dara
// por bueno.
func relojDelCorpus(o corpus.Obligacion, hechos map[string]time.Time,
	cal CalendarioDeclarado) (RelojDeclarado, error) {
	t := o.Temporalidad
	reg, err := ventana.RegimenDesde(t.Regimen.Computo, t.Regimen.Cierre, t.Regimen.Traslado)
	if err != nil {
		return RelojDeclarado{}, err
	}

	rd := RelojDeclarado{
		Obligacion: o.ID,
		Disparador: t.Disparador["hecho"],
		Hechos:     map[string]string{},
		Calendario: cal,
	}
	for k, v := range hechos {
		rd.Hechos[k] = v.Format(time.RFC3339)
	}

	escribir := func(id, limite, desdeHito string) HitoDeclarado {
		return HitoDeclarado{
			ID: id, Limite: limite, DesdeHito: desdeHito,
			Computo:  reg.Comp.String(),
			Cierre:   reg.Cierre.String(),
			Traslado: reg.Trasl.String(),
			Fuente:   o.Cita,
		}
	}
	if len(t.Hitos) > 0 {
		for _, h := range t.Hitos {
			// UN HITO PUEDE TRAER SU PROPIO REGIMEN, y entonces manda el suyo.
			hreg := reg
			if h.Regimen != nil {
				hreg, err = ventana.RegimenDesde(h.Regimen.Computo, h.Regimen.Cierre, h.Regimen.Traslado)
				if err != nil {
					return RelojDeclarado{}, fmt.Errorf("hito %s: %w", h.ID, err)
				}
			}
			hd := escribir(h.ID, h.Limite, h.DesdeHito)
			hd.Computo, hd.Cierre, hd.Traslado = hreg.Comp.String(), hreg.Cierre.String(), hreg.Trasl.String()
			rd.Hitos = append(rd.Hitos, hd)
		}
		return rd, nil
	}
	id := t.Hito
	if id == "" {
		id = "limite"
	}
	rd.Hitos = append(rd.Hitos, escribir(id, t.Limite, ""))
	return rd, nil
}

// derivarAplicables ejecuta el mismo motor que reejecuta la verificacion.
func derivarAplicables(progs []aplicabilidad.Programa, hechos []aplicabilidad.Hecho) ([]string, error) {
	m := aplicabilidad.NuevoMotor()
	for _, p := range progs {
		if err := m.Cargar(p); err != nil {
			return nil, fmt.Errorf("el programa de %s no carga: %w", p.Paquete, err)
		}
	}
	for _, h := range hechos {
		m.Afirmar(h)
	}
	if _, err := m.Evaluar(); err != nil {
		return nil, fmt.Errorf("la aplicabilidad no converge: %w", err)
	}
	visto := map[string]bool{}
	var out []string
	for _, h := range m.Consultar(aplicabilidad.A("aplica",
		aplicabilidad.V("O"), aplicabilidad.V("S"))) {
		if visto[h.Args[0]] {
			continue
		}
		visto[h.Args[0]] = true
		out = append(out, h.Args[0])
	}
	sort.Strings(out)
	return out, nil
}
