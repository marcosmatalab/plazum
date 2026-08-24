// Package expediente exporta y verifica el expediente completo.
//
// La propiedad que persigue: un tercero con el fichero y el binario, SIN RED
// y sin confiar en quien lo emitio, recalcula desde cero la aplicabilidad, los
// plazos y los estados de control, y obtiene exactamente lo mismo, o le dice
// donde no coincide.
//
// Ningun GRC del mercado, libre o comercial, hace esto hoy. Es lo contrario de
// la ola de "AI compliance" de 2026: en vez de pedir confianza en el emisor,
// se entrega algo que el receptor puede recomputar.
package expediente

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"dutiq/nucleo/aplicabilidad"
	"dutiq/nucleo/estado"
	"dutiq/nucleo/ledger"
	"dutiq/nucleo/ventana"
)

const Version = "dutiq-expediente-v1"

type Paquete struct {
	URN      string    `json:"urn"`
	Version  string    `json:"version"`
	Digest   string    `json:"digest"`
	Clase    string    `json:"clase"` // normativo | referencial
	Vigencia Intervalo `json:"vigencia"`
}

type Intervalo struct {
	Desde time.Time `json:"desde"`
	Hasta time.Time `json:"hasta,omitempty"`
}

type Obligacion struct {
	ID         string `json:"id"`
	Paquete    string `json:"paquete"`
	Articulo   string `json:"articulo"`
	Afirmacion string `json:"afirmacion"`
	Control    string `json:"control"`
	Primitiva  string `json:"primitiva"`
}

// Reclamacion es un vencimiento que el emisor AFIRMA haber calculado.
// La verificacion lo recalcula y compara.
type Reclamacion struct {
	Obligacion string    `json:"obligacion"`
	Hito       string    `json:"hito"`
	Estado     string    `json:"estado"`
	Vence      time.Time `json:"vence,omitempty"`
	Regla      string    `json:"regla"`
}

type EstadoControl struct {
	Prueba string `json:"prueba"`
	Estado string `json:"estado"`
	Motivo string `json:"motivo"`
}

type Expediente struct {
	Version      string    `json:"version"`
	Emitido      time.Time `json:"emitido"`
	ComoEstaba   time.Time `json:"como_estaba"`
	Organizacion string    `json:"organizacion"`
	Alcance      string    `json:"alcance"`

	// AnclasDeConfianza son los digests de paquete que el RECEPTOR acepta,
	// obtenidos del registro firmado, no del expediente.
	//
	// HALLAZGO DE REVISION. Sin esto la verificacion era circular: el emisor
	// aportaba las reglas Y el digest, y nadie los contrastaba con nada. Se
	// podia inventar una obligacion junto con la regla que la deriva y todo
	// verificaba limpio. La propiedad solo es real cuando el corpus se
	// comprueba contra una fuente que el emisor no controla.
	AnclasDeConfianza map[string]string `json:"anclas_de_confianza,omitempty"`

	Paquetes     []Paquete                `json:"paquetes"`
	Programas    []aplicabilidad.Programa `json:"programas"`
	Hechos       []aplicabilidad.Hecho    `json:"hechos"`
	Obligaciones []Obligacion             `json:"obligaciones"`

	Pruebas       []estado.Prueba      `json:"pruebas"`
	Observaciones []estado.Observacion `json:"observaciones"`
	Excepciones   []estado.Excepcion   `json:"excepciones"`
	Exclusiones   []estado.Exclusion   `json:"exclusiones"`

	Relojes []RelojDeclarado `json:"relojes"`
	Ledger  ledger.Ledger    `json:"ledger"`

	// Lo que el emisor afirma. La verificacion lo recalcula.
	Aplicables    []string             `json:"aplicables"`
	Reclamaciones []Reclamacion        `json:"reclamaciones"`
	Estados       []EstadoControl      `json:"estados"`
	Denominadores estado.Denominadores `json:"denominadores"`
}

// RelojDeclarado guarda los datos de un plazo de forma serializable.
type RelojDeclarado struct {
	Obligacion string              `json:"obligacion"`
	Disparador string              `json:"disparador"`
	Hechos     map[string]string   `json:"hechos"` // clave -> RFC3339
	Hitos      []HitoDeclarado     `json:"hitos"`
	Calendario CalendarioDeclarado `json:"calendario"`
}

type HitoDeclarado struct {
	ID        string `json:"id"`
	Limite    string `json:"limite"`
	DesdeHito string `json:"desde_hito,omitempty"`
	Computo   string `json:"computo"`  // naturales | habiles
	Cierre    string `json:"cierre"`   // auto | exacto | fin_dia
	Traslado  string `json:"traslado"` // ninguno | siguiente_habil
	Fuente    string `json:"fuente"`
}

type CalendarioDeclarado struct {
	ID       string   `json:"id"`
	Zona     string   `json:"zona"`
	Ambito   string   `json:"ambito"`
	Fuente   string   `json:"fuente"`
	Festivos []string `json:"festivos"`
}

type Discrepancia struct {
	Que      string `json:"que"`
	Esperado string `json:"esperado"`
	Obtenido string `json:"obtenido"`
}

type Informe struct {
	Valido         bool           `json:"valido"`
	Comprobaciones []string       `json:"comprobaciones"`
	Discrepancias  []Discrepancia `json:"discrepancias"`
}

// DigestPaquete calcula el digest canonico de un paquete a partir de su
// contenido: reglas y obligaciones. Es lo que se compara contra el registro.
func DigestPaquete(urn string, progs []aplicabilidad.Programa, obls []Obligacion) string {
	var partes []string
	for _, p := range progs {
		if p.Paquete != urn {
			continue
		}
		for _, r := range p.Reglas {
			b, _ := json.Marshal(r)
			partes = append(partes, string(b))
		}
	}
	for _, o := range obls {
		if o.Paquete != urn {
			continue
		}
		b, _ := json.Marshal(o)
		partes = append(partes, string(b))
	}
	sort.Strings(partes)
	h := sha256.Sum256([]byte(strings.Join(partes, "\x1f")))
	return "sha256:" + hex.EncodeToString(h[:])
}

// Verificar recomputa el expediente entero sin red y sin confiar en el emisor.
func Verificar(e *Expediente) Informe {
	inf := Informe{Valido: true}
	add := func(s string) { inf.Comprobaciones = append(inf.Comprobaciones, s) }
	fallo := func(que, esp, obt string) {
		inf.Valido = false
		inf.Discrepancias = append(inf.Discrepancias, Discrepancia{que, esp, obt})
	}

	if e.Version != Version {
		fallo("version", Version, e.Version)
	}

	// 1. Cadena de custodia.
	if err := e.Ledger.Verificar(); err != nil {
		fallo("ledger", "cadena integra y checkpoints firmados y anclados", err.Error())
	} else {
		add(fmt.Sprintf("ledger: %d entradas encadenadas y %d checkpoint(s) firmados y anclados",
			len(e.Ledger.Entradas), len(e.Ledger.Checkpoints)))
	}

	// 2. Corpus: el digest declarado tiene que salir del contenido, y el
	//    contenido tiene que coincidir con el ancla que trae el receptor.
	if len(e.AnclasDeConfianza) == 0 {
		fallo("anclas de confianza", "el receptor aporta los digests del registro firmado",
			"ninguna: la verificacion seria circular")
	}
	for _, p := range e.Paquetes {
		calc := DigestPaquete(p.URN, e.Programas, e.Obligaciones)
		if calc != p.Digest {
			fallo("digest de "+p.URN, p.Digest, calc)
			continue
		}
		esperado, ok := e.AnclasDeConfianza[p.URN]
		if !ok {
			fallo("ancla de "+p.URN, "digest conocido por el receptor", "paquete no reconocido")
			continue
		}
		if esperado != calc {
			fallo("contenido de "+p.URN, esperado+" (registro)", calc+" (expediente)")
		}
	}
	// Y ningun programa puede venir de un paquete que no este declarado y
	// anclado: si no, basta con adjuntar la regla que deriva lo que quieras.
	declarados := map[string]bool{}
	for _, p := range e.Paquetes {
		declarados[p.URN] = true
	}
	for _, pr := range e.Programas {
		if !declarados[pr.Paquete] {
			fallo("programa de "+pr.Paquete, "paquete declarado y anclado",
				"reglas aportadas sin paquete que las respalde")
			continue
		}
		if _, ok := e.AnclasDeConfianza[pr.Paquete]; !ok {
			fallo("programa de "+pr.Paquete, "ancla del receptor", "paquete no reconocido")
		}
	}
	add(fmt.Sprintf("corpus: %d paquetes con digest recalculado sobre su contenido y contrastado con el registro, "+
		"y %d programas atados a su paquete", len(e.Paquetes), len(e.Programas)))

	// 3. Vigencia normativa: ninguna obligacion puede evaluarse fuera de la
	//    vigencia de su paquete a la fecha del expediente.
	vig := map[string]Paquete{}
	for _, p := range e.Paquetes {
		vig[p.URN] = p
	}
	for _, o := range e.Obligaciones {
		p, ok := vig[o.Paquete]
		if !ok {
			fallo("paquete de "+o.ID, "declarado en el expediente", "ausente")
			continue
		}
		if e.ComoEstaba.Before(p.Vigencia.Desde) {
			fallo("vigencia de "+o.ID,
				"no evaluada antes de "+p.Vigencia.Desde.Format("2006-01-02"),
				"evaluada el "+e.ComoEstaba.Format("2006-01-02"))
		}
	}
	add(fmt.Sprintf("vigencia: %d obligaciones de %d paquetes comprobadas contra la fecha del expediente",
		len(e.Obligaciones), len(e.Paquetes)))

	// 3. Aplicabilidad: se reejecuta el Datalog de los paquetes.
	m := aplicabilidad.NuevoMotor()
	for _, p := range e.Programas {
		m.Cargar(p)
	}
	for _, h := range e.Hechos {
		m.Afirmar(h)
	}
	if _, err := m.Evaluar(); err != nil {
		fallo("aplicabilidad", "programa estratificable y convergente", err.Error())
	} else {
		recalc := map[string]bool{}
		for _, h := range m.Consultar(aplicabilidad.A("aplica",
			aplicabilidad.V("O"), aplicabilidad.V("S"))) {
			recalc[h.Args[0]] = true
		}
		decl := map[string]bool{}
		for _, a := range e.Aplicables {
			decl[a] = true
		}
		for a := range decl {
			if !recalc[a] {
				fallo("aplicabilidad de "+a, "derivable de las reglas y los hechos", "no derivable")
			}
		}
		for a := range recalc {
			if !decl[a] {
				fallo("aplicabilidad de "+a, "declarada en el expediente", "derivada pero no declarada")
			}
		}
		add(fmt.Sprintf("aplicabilidad: %d obligaciones aplicables rederivadas de %d reglas y %d hechos",
			len(recalc), contarReglas(e.Programas), len(e.Hechos)))
	}

	// 4. Relojes: se recalcula cada vencimiento con el motor temporal.
	recl := map[string]Reclamacion{}
	for _, r := range e.Reclamaciones {
		recl[r.Obligacion+"/"+r.Hito] = r
	}
	nrel := 0
	for _, rd := range e.Relojes {
		p, hechos, err := construirPlazo(rd)
		if err != nil {
			fallo("reloj de "+rd.Obligacion, "declaracion valida", err.Error())
			continue
		}
		for _, v := range p.Vencimientos(hechos, time.Time{}) {
			nrel++
			k := rd.Obligacion + "/" + v.Hito
			d, ok := recl[k]
			if !ok {
				fallo(k, "declarado en el expediente", "calculado pero no declarado")
				continue
			}
			if d.Estado != v.Estado.String() {
				fallo(k+" (estado)", d.Estado, v.Estado.String())
			}
			if v.Estado == ventana.Determinado && !d.Vence.Equal(v.Vence) {
				fallo(k+" (vencimiento)", d.Vence.Format(time.RFC3339), v.Vence.Format(time.RFC3339))
			}
		}
	}
	// Y al reves: una reclamacion declarada que ningun reloj calcula es una
	// fecha inventada. Antes solo se comprobaba el sentido calculado -> declarado.
	calculadas := map[string]bool{}
	for _, rd := range e.Relojes {
		p, hechos, err := construirPlazo(rd)
		if err != nil {
			continue
		}
		for _, v := range p.Vencimientos(hechos, time.Time{}) {
			calculadas[rd.Obligacion+"/"+v.Hito] = true
		}
	}
	for k := range recl {
		if !calculadas[k] {
			fallo(k, "calculable a partir de un reloj declarado", "declarado sin reloj que lo produzca")
		}
	}
	add(fmt.Sprintf("relojes: %d vencimientos recalculados con el motor temporal", nrel))

	// 5b. Toda observacion tiene que estar anclada en el ledger.
	//     Sin esto se podia cambiar `Satisfecho` en la lista de observaciones
	//     con el ledger intacto, y el expediente verificaba.
	enLedger := map[string]bool{}
	for _, ent := range e.Ledger.Entradas {
		if ent.Tipo != "observacion" {
			continue
		}
		var o estado.Observacion
		if err := json.Unmarshal(ent.Carga, &o); err == nil {
			enLedger[huellaObs(o)] = true
		}
	}
	sinAnclar := 0
	for _, o := range e.Observaciones {
		if !enLedger[huellaObs(o)] {
			sinAnclar++
			fallo("observacion "+o.Prueba+"/"+o.Recurso, "anclada en el ledger",
				"no aparece en la cadena, o aparece con otro contenido")
		}
	}
	if sinAnclar == 0 {
		add(fmt.Sprintf("anclaje: las %d observaciones estan en la cadena con el mismo contenido",
			len(e.Observaciones)))
	}

	// 6. Estados de control: se recalculan con la misma funcion pura.
	obsPorPrueba := map[string][]estado.Observacion{}
	for _, o := range e.Observaciones {
		obsPorPrueba[o.Prueba] = append(obsPorPrueba[o.Prueba], o)
	}
	declEst := map[string]EstadoControl{}
	for _, s := range e.Estados {
		declEst[s.Prueba] = s
	}
	var den estado.Denominadores
	for _, pr := range e.Pruebas {
		// La aplicabilidad la decide el motor, no una constante.
		aplicable := true
		if pr.Control != "" {
			aplicable = false
			for _, a := range e.Aplicables {
				if a == pr.Control {
					aplicable = true
				}
			}
		}
		ent := estado.Calcular(pr, obsPorPrueba[pr.ID], estado.Contexto{
			Ahora: e.ComoEstaba, Aplicable: aplicable,
			Excepciones: e.Excepciones, Exclusiones: e.Exclusiones,
		})
		d, ok := declEst[pr.ID]
		if !ok {
			fallo("estado de "+pr.ID, "declarado", "ausente")
			continue
		}
		if d.Estado != ent.Estado.String() {
			fallo("estado de "+pr.ID, d.Estado, ent.Estado.String())
		}
		switch ent.Estado {
		case estado.Pass, estado.FailEnPlazo, estado.FailVencido:
			den.Maquina++
		case estado.Manual:
			den.Humano++
		case estado.Obsoleto:
			den.CaducadoOContradicho++
		case estado.Error, estado.NoAplica, estado.Exceptuado:
			den.Desconocido++
		}
	}
	add(fmt.Sprintf("estados: %d pruebas recalculadas", len(e.Pruebas)))
	// Los CINCO, no dos. Antes Humano, Externo y Desconocido admitian cualquier
	// valor y la verificacion seguia diciendo "coinciden".
	// NOTA (trampa conocida, encontrada en revision): ningun estado de los 8
	// mapea aun a Externo, asi que el recuento independiente siempre da 0 y un
	// expediente que declare Externo>0 NO verificara. Es deliberado hasta que
	// exista la atestacion externa (etapa 4/6): entonces su estado mapeara aqui.
	if den != e.Denominadores {
		fallo("denominadores", e.Denominadores.String(), den.String())
	} else {
		add("denominadores: los cinco coinciden con el recuento independiente")
	}

	sort.Slice(inf.Discrepancias, func(i, j int) bool { return inf.Discrepancias[i].Que < inf.Discrepancias[j].Que })
	return inf
}

func huellaObs(o estado.Observacion) string {
	b, _ := json.Marshal(o)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func contarReglas(ps []aplicabilidad.Programa) int {
	n := 0
	for _, p := range ps {
		n += len(p.Reglas)
	}
	return n
}

func construirPlazo(rd RelojDeclarado) (ventana.Plazo, ventana.Hechos, error) {
	loc, err := time.LoadLocation(rd.Calendario.Zona)
	if err != nil {
		return ventana.Plazo{}, nil, fmt.Errorf("zona horaria %q: %w", rd.Calendario.Zona, err)
	}
	cal := ventana.NuevoCalendario(rd.Calendario.ID, rd.Calendario.Ambito, rd.Calendario.Fuente, loc, rd.Calendario.Festivos...)
	p := ventana.Plazo{Disparador: rd.Disparador}
	for _, h := range rd.Hitos {
		d, err := ventana.ParseDuracion(h.Limite)
		if err != nil {
			return ventana.Plazo{}, nil, fmt.Errorf("hito %s: %w", h.ID, err)
		}
		reg := ventana.Regimen{Cal: cal, Fuente: h.Fuente}
		if h.Computo == "habiles" {
			reg.Comp = ventana.Habiles
		}
		switch h.Cierre {
		case "exacto":
			reg.Cierre = ventana.CierreExacto
		case "fin_dia":
			reg.Cierre = ventana.CierreFinDia
		}
		if h.Traslado == "siguiente_habil" {
			reg.Trasl = ventana.TrasladoSiguienteHabil
		}
		p.Hitos = append(p.Hitos, ventana.Hito{ID: h.ID, Limite: d, Reg: reg, DesdeHito: h.DesdeHito})
	}
	hechos := ventana.Hechos{}
	for k, v := range rd.Hechos {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return ventana.Plazo{}, nil, fmt.Errorf("hecho %s: %w", k, err)
		}
		hechos[k] = t
	}
	return p, hechos, nil
}

func Cargar(b []byte) (*Expediente, error) {
	var e Expediente
	if err := json.Unmarshal(b, &e); err != nil {
		return nil, err
	}
	return &e, nil
}

func (e *Expediente) Guardar() ([]byte, error) { return json.MarshalIndent(e, "", "  ") }
