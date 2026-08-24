// Package estado calcula el estado de un control a partir de sus
// observaciones, su politica de frescura y sus excepciones.
//
// Ocho estados, no dos. Un GRC que solo tiene "cumple" y "no cumple" produce
// ruido, y el ruido es lo que hace que la gente apague las alertas. Las
// distinciones que faltan casi siempre:
//
//	fail_en_plazo vs fail_vencido  Vanta lo resuelve con un SLA por categoria
//	                               y solo escala y ensena al auditor lo vencido.
//	obsoleto vs fallo              "no lo he mirado hoy" no es "esta mal".
//	error vs fallo                 no poder recolectar no es incumplir.
//	no_aplica                      viene de la declaracion de aplicabilidad,
//	                               nunca de un boton de la interfaz.
//	exceptuado                     con dueno, motivo, aprobador y FECHA FIN.
//	                               Sin fecha fin es una excepcion eterna, que
//	                               es como se pierde un certificado.
package estado

import (
	"fmt"
	"time"
)

type Estado uint8

const (
	Pass        Estado = iota // observacion fresca y predicado satisfecho
	FailEnPlazo               // falla, pero dentro del SLA de remediacion
	FailVencido               // falla y el SLA se agoto. Esto si escala
	Obsoleto                  // no hay observacion fresca. No es un fallo
	Error                     // el recolector no pudo obtener el dato
	NoAplica                  // excluido por la declaracion de aplicabilidad
	Manual                    // evidencia aportada por una persona, con caducidad
	Exceptuado                // excepcion vigente, aprobada y con fecha fin
)

var nombres = [...]string{"pass", "fail_en_plazo", "fail_vencido", "obsoleto",
	"error", "no_aplica", "manual", "exceptuado"}

func (e Estado) String() string { return nombres[e] }

// EscalaAlAuditor devuelve si este estado debe aparecer en el expediente que
// ve el auditor. Un fallo dentro de SLA es trabajo normal, no un hallazgo.
// EscalaAlAuditor: un fallo dentro de SLA es trabajo normal, no un hallazgo.
// El error de recoleccion SI escala, porque un 429 que dura un ano significa
// que ese control lleva un ano sin comprobarse. Antes no escalaba y quedaba
// invisible, que era un agujero.
func (e Estado) EscalaAlAuditor() bool {
	return e == FailVencido || e == Obsoleto || e == Error
}

// Observacion es lo que devuelve un recolector. Los campos siguen el modelo
// OSCAL assessment-results: `collected` obligatorio, `expires` opcional.
type Observacion struct {
	Prueba      string
	Recurso     string
	Satisfecho  bool
	ErrorRecol  string
	Recolectada time.Time
	Caduca      time.Time // cero = usar el TTL de la prueba
	Recolector  string
	Version     string
	HashCarga   string
}

// Prueba es la comprobacion automatizada asociada a un control.
type Prueba struct {
	ID          string
	Control     string
	TTL         time.Duration // frescura maxima admitida
	SLA         time.Duration // plazo de remediacion antes de escalar
	Activa      time.Time     // rollout: antes de esta fecha no genera historico
	PassPorDef  bool          // el proveedor lo garantiza y no es configurable
	Descripcion string
}

// Excepcion es una dispensa temporal. Sin aprobador y sin fecha fin no se acepta.
type Excepcion struct {
	Control   string
	Motivo    string
	Aprobador string
	Desde     time.Time
	Hasta     time.Time
}

func (e Excepcion) Valida() error {
	if e.Aprobador == "" {
		return fmt.Errorf("excepcion sin aprobador")
	}
	if e.Hasta.IsZero() {
		return fmt.Errorf("excepcion sin fecha fin: las excepciones eternas son como se pierde un certificado")
	}
	if !e.Hasta.After(e.Desde) {
		return fmt.Errorf("excepcion con fecha fin anterior al inicio")
	}
	return nil
}

// Exclusion saca UN recurso de la evaluacion, no el control entero.
type Exclusion struct {
	Prueba  string
	Recurso string
	Motivo  string // se ensena al auditor: nombre y motivo, nada mas
}

type Entrada struct {
	Estado    Estado
	Desde     time.Time
	Vence     time.Time // cuando expira el SLA, si aplica
	Motivo    string
	Recursos  int
	Fallando  []string
	Excluidos []string
}

type Contexto struct {
	Ahora       time.Time
	Aplicable   bool // lo decide el motor de aplicabilidad, no la interfaz
	Excepciones []Excepcion
	Exclusiones []Exclusion
}

// Calcular es una funcion pura: mismos datos, mismo estado, siempre.
func Calcular(p Prueba, obs []Observacion, ctx Contexto) Entrada {
	if !ctx.Aplicable {
		return Entrada{Estado: NoAplica, Desde: ctx.Ahora,
			Motivo: "fuera de la declaracion de aplicabilidad"}
	}
	for _, e := range ctx.Excepciones {
		// HALLAZGO DE REVISION: Valida() existia y no se llamaba nunca, asi que
		// una excepcion sin aprobador y sin fecha fin se aplicaba igual.
		if e.Valida() != nil {
			continue
		}
		if e.Control == p.Control && !ctx.Ahora.Before(e.Desde) && ctx.Ahora.Before(e.Hasta) {
			return Entrada{Estado: Exceptuado, Desde: e.Desde, Vence: e.Hasta,
				Motivo: fmt.Sprintf("excepcion aprobada por %s: %s", e.Aprobador, e.Motivo)}
		}
	}
	if !p.Activa.IsZero() && ctx.Ahora.Before(p.Activa) {
		return Entrada{Estado: Manual, Desde: ctx.Ahora,
			Motivo: "prueba en periodo de despliegue hasta " + p.Activa.Format(time.RFC3339)}
	}
	// PassPorDef solo vale si NO hay observaciones que lo contradigan: si el
	// recolector ve un fallo, el fallo manda sobre la declaracion del proveedor.
	if p.PassPorDef {
		contradicho := false
		for _, o := range obs {
			if o.ErrorRecol == "" && !o.Satisfecho {
				contradicho = true
			}
		}
		if !contradicho {
			return Entrada{Estado: Pass, Desde: ctx.Ahora,
				Motivo: "garantizado por el proveedor y no configurable por el cliente"}
		}
	}

	excl := map[string]string{}
	for _, x := range ctx.Exclusiones {
		if x.Prueba == p.ID {
			excl[x.Recurso] = x.Motivo
		}
	}
	var considerados []Observacion
	var excluidos []string
	for _, o := range obs {
		if m, ok := excl[o.Recurso]; ok {
			excluidos = append(excluidos, o.Recurso+": "+m)
			continue
		}
		considerados = append(considerados, o)
	}
	if len(considerados) == 0 {
		return Entrada{Estado: Obsoleto, Desde: ctx.Ahora, Excluidos: excluidos,
			Motivo: "sin observaciones para esta prueba"}
	}

	// HALLAZGO DE REVISION. La primera version comprobaba la frescura DENTRO
	// del bucle y devolvia obsoleto antes de llegar al calculo del SLA. Con la
	// configuracion normal (SLA mayor que TTL) el estado fail_vencido, que es
	// justo el unico que escala al auditor, era inalcanzable: 1 h fail_en_plazo,
	// 25 h obsoleto, 800 h obsoleto. El unico test que lo alcanzaba fabricaba
	// una caducidad que ningun recolector produce.
	//
	// La correccion: recorrer todo primero, y despues decidir por severidad.
	// Un fallo con el plazo de remediacion agotado manda sobre la obsolescencia,
	// porque lo que sabemos es peor que lo que no sabemos.
	var masAntigua, primerCaducada time.Time
	hayError, primerFallo := "", time.Time{}
	var fallando []string
	var recursoCaducado string
	for i, o := range considerados {
		if i == 0 || o.Recolectada.Before(masAntigua) {
			masAntigua = o.Recolectada
		}
		if o.ErrorRecol != "" {
			hayError = o.ErrorRecol
			continue
		}
		if !o.Satisfecho {
			fallando = append(fallando, o.Recurso)
			if primerFallo.IsZero() || o.Recolectada.Before(primerFallo) {
				primerFallo = o.Recolectada
			}
		}
		caduca := o.Caduca
		if caduca.IsZero() && p.TTL > 0 {
			caduca = o.Recolectada.Add(p.TTL)
		}
		if !caduca.IsZero() && ctx.Ahora.After(caduca) &&
			(primerCaducada.IsZero() || caduca.Before(primerCaducada)) {
			primerCaducada, recursoCaducado = caduca, o.Recurso
		}
	}
	if len(fallando) > 0 {
		vence := primerFallo.Add(p.SLA)
		if ctx.Ahora.After(vence) {
			return Entrada{Estado: FailVencido, Desde: primerFallo, Vence: vence, Recursos: len(considerados),
				Fallando: fallando, Excluidos: excluidos,
				Motivo: fmt.Sprintf("%d recurso(s) fallan y el plazo de remediacion vencio el %s",
					len(fallando), vence.Format(time.RFC3339))}
		}
		if !primerCaducada.IsZero() {
			return Entrada{Estado: Obsoleto, Desde: primerCaducada, Recursos: len(considerados), Excluidos: excluidos,
				Motivo: fmt.Sprintf("hay fallos dentro de plazo, pero la observacion de %s caduco el %s: "+
					"no se puede afirmar el estado actual", recursoCaducado, primerCaducada.Format(time.RFC3339))}
		}
		return Entrada{Estado: FailEnPlazo, Desde: primerFallo, Vence: vence, Recursos: len(considerados),
			Fallando: fallando, Excluidos: excluidos,
			Motivo: fmt.Sprintf("%d recurso(s) fallan, plazo de remediacion hasta el %s",
				len(fallando), vence.Format(time.RFC3339))}
	}
	if !primerCaducada.IsZero() {
		return Entrada{Estado: Obsoleto, Desde: primerCaducada, Recursos: len(considerados), Excluidos: excluidos,
			Motivo: fmt.Sprintf("la observacion de %s caduco el %s: obsoleto no es fallo",
				recursoCaducado, primerCaducada.Format(time.RFC3339))}
	}
	if hayError != "" && len(fallando) == 0 {
		return Entrada{Estado: Error, Desde: ctx.Ahora, Recursos: len(considerados), Excluidos: excluidos,
			Motivo: "el recolector no pudo obtener el dato: " + hayError}
	}
	return Entrada{Estado: Pass, Desde: masAntigua, Recursos: len(considerados), Excluidos: excluidos,
		Motivo: "todas las observaciones satisfacen el predicado"}
}

// Denominadores es el contador honesto. Cinco numeros, nunca un porcentaje.
type Denominadores struct {
	Maquina, Humano, Externo, Desconocido, CaducadoOContradicho int
}

func (d Denominadores) String() string {
	return fmt.Sprintf("maquina=%d humano=%d externo=%d desconocido=%d caducado_o_contradicho=%d",
		d.Maquina, d.Humano, d.Externo, d.Desconocido, d.CaducadoOContradicho)
}
