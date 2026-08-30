package escalado

import (
	"errors"
	"fmt"
	"sort"
	"time"
)

// EL CURSO DE UN ESCALON: lo que le pasa a un aviso despues de salir.
//
// POR QUE ES UN FLUJO Y NO TRES BOOLEANOS. Un escalon con campos `enviado`,
// `entregado` y `atendido` que se pisan pierde justo lo que un auditor
// pregunta: cuando se mando, cuanto tardo en entregarse y cuanto en atenderse.
// Y pierde tambien el reintento: un envio que falla y se repite son dos hechos,
// no uno corregido. Es la misma forma que el incidente, por el mismo motivo.
//
// "SE NOTIFICO" Y "ALGUIEN LO ATENDIO" SON DOS HECHOS DISTINTOS, y esta es la
// razon de que Entregado y Atendido no sean el mismo estado. Un correo que
// llego a una bandeja que nadie abre esta entregado y no esta atendido, y el
// acta del 9.3 necesita el segundo: "se aviso" no es "se ocuparon".
//
// UN FALLO DE ENTREGA NO SE LOGUEA: SE ENSENA. Fallido es un estado del cubo,
// no una linea de log, precisamente para que la ley de conservacion lo cuente y
// la pantalla lo saque. Un escalado que muere en silencio es peor que no
// tenerlo, porque el operador cree que esta cubierto y no lo esta.

// TipoSuceso es el vocabulario CERRADO de lo que le pasa a un aviso.
type TipoSuceso uint8

const (
	// SeEnvio: el aviso se entrego AL CANAL. No dice que llegara.
	SeEnvio TipoSuceso = iota
	// SeEntrego: el canal confirmo la entrega.
	SeEntrego
	// Fallo: la entrega fallo. Trae el motivo, sin secretos dentro.
	Fallo
	// SeAtendio: una persona se hizo cargo. Trae quien.
	SeAtendio
)

var nombresDeSuceso = [...]string{"se envio", "se entrego", "fallo", "se atendio"}

func (t TipoSuceso) String() string {
	if int(t) >= len(nombresDeSuceso) {
		return fmt.Sprintf("suceso desconocido(%d)", uint8(t))
	}
	return nombresDeSuceso[t]
}

func (t TipoSuceso) Valido() bool { return int(t) < len(nombresDeSuceso) }

// SucesoDeEnvio es un hecho del curso de un escalon. Inmutable: se anade.
type SucesoDeEnvio struct {
	Tipo TipoSuceso
	// Canal por el que fue (o iba a ir). Obligatorio salvo en SeAtendio, que
	// puede ocurrir dentro de plazum sin canal ninguno.
	Canal string
	// Quien atendio. Obligatorio en SeAtendio y PROHIBIDO en el resto: un
	// escalon atendido sin nombre no sirve de nada en un expediente.
	Quien string
	// Motivo del fallo. Obligatorio en Fallo y prohibido en el resto.
	Motivo string
	// Los dos ejes, con el mismo significado que en historia y en incidente.
	InstanteHecho    time.Time
	InstanteRegistro time.Time
}

var (
	ErrSucesoDesconocido  = errors.New("tipo de suceso de envio fuera del vocabulario")
	ErrSucesoSinInstante  = errors.New("suceso de envio sin uno de sus dos ejes")
	ErrSucesoAlReves      = errors.New("el registro de un suceso de envio es anterior al hecho")
	ErrSucesoCampoFalta   = errors.New("falta un campo obligatorio del suceso de envio")
	ErrSucesoCampoAjeno   = errors.New("campo que no corresponde a este tipo de suceso")
	ErrCursoSinEnvioAntes = errors.New("suceso de entrega sin envio previo")
)

// Curso es el flujo de hechos de UN escalon. Append-only.
type Curso struct {
	sucesos []SucesoDeEnvio
}

// Registrar anade un hecho. No hay Modificar ni Borrar.
func (c *Curso) Registrar(s SucesoDeEnvio) error {
	if !s.Tipo.Valido() {
		return fmt.Errorf("%w: %s", ErrSucesoDesconocido, s.Tipo)
	}
	if s.InstanteHecho.IsZero() || s.InstanteRegistro.IsZero() {
		return fmt.Errorf("%w (%s): cuando paso y cuando se supo son dos datos y los dos "+
			"hacen falta", ErrSucesoSinInstante, s.Tipo)
	}
	if s.InstanteRegistro.Before(s.InstanteHecho) {
		return fmt.Errorf("%w (%s): consta registrado el %s y ocurrido el %s", ErrSucesoAlReves,
			s.Tipo, s.InstanteRegistro.Format(time.RFC3339), s.InstanteHecho.Format(time.RFC3339))
	}
	if err := camposDelSuceso(s); err != nil {
		return err
	}
	// UNA ENTREGA O UN FALLO SIN ENVIO ES UN DATO IMPOSIBLE. Aceptarlo dejaria
	// un curso que dice que algo llego sin haber salido, y esa fila viajaria al
	// expediente con cara de hecho.
	if s.Tipo == SeEntrego || s.Tipo == Fallo {
		visto := false
		for _, x := range c.sucesos {
			if x.Tipo == SeEnvio {
				visto = true
			}
		}
		if !visto {
			return fmt.Errorf("%w: llega un %q y este escalon no consta enviado",
				ErrCursoSinEnvioAntes, s.Tipo)
		}
	}
	c.sucesos = append(c.sucesos, s)
	return nil
}

func camposDelSuceso(s SucesoDeEnvio) error {
	quiere := func(nombre, valor string, hace bool) error {
		switch {
		case hace && valor == "":
			return fmt.Errorf("%w: un suceso %q necesita %s", ErrSucesoCampoFalta, s.Tipo, nombre)
		case !hace && valor != "":
			return fmt.Errorf("%w: un suceso %q no lleva %s (vale %q)", ErrSucesoCampoAjeno,
				s.Tipo, nombre, valor)
		}
		return nil
	}
	if err := quiere("quien", s.Quien, s.Tipo == SeAtendio); err != nil {
		return err
	}
	if err := quiere("motivo", s.Motivo, s.Tipo == Fallo); err != nil {
		return err
	}
	return quiere("canal", s.Canal, s.Tipo != SeAtendio)
}

// Sucesos devuelve una COPIA ordenada por el eje del mundo.
func (c *Curso) Sucesos() []SucesoDeEnvio {
	out := append([]SucesoDeEnvio(nil), c.sucesos...)
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].InstanteHecho.Before(out[j].InstanteHecho)
	})
	return out
}

// Estado es donde ha acabado el escalon, mirando su flujo.
//
// EL ORDEN DE PRECEDENCIA NO ES EL CRONOLOGICO, y conviene decirlo porque es lo
// que un lector espera y no es lo que hace: si alguien lo ATENDIO, eso manda
// aunque despues hubiera un reintento fallido. Lo que le importa al expediente
// es si alguien se hizo cargo, no cuantas veces lo intento el canal.
//
// El segundo dato es que un FALLO manda sobre un envio: un escalon que se
// intento, fallo y no se reintento esta fallido, no enviado.
func (c *Curso) Estado() (Estado, bool) {
	if len(c.sucesos) == 0 {
		return "", false
	}
	visto := map[TipoSuceso]bool{}
	for _, s := range c.sucesos {
		visto[s.Tipo] = true
	}
	switch {
	case visto[SeAtendio]:
		return Atendido, true
	case visto[SeEntrego]:
		return Entregado, true
	case visto[Fallo]:
		return Fallido, true
	default:
		return Enviado, true
	}
}

// Atendido devuelve quien se hizo cargo y cuando, si consta.
//
// El nombre del metodo dice "consta" en su segundo valor y no "se hizo": que no
// conste que alguien lo atendio NO dice que nadie lo hiciera, dice que en el
// sistema no aparece. La pantalla que lo pinte tiene que decirlo asi.
func (c *Curso) Atendido() (quien string, cuando time.Time, consta bool) {
	for _, s := range c.Sucesos() {
		if s.Tipo == SeAtendio {
			return s.Quien, s.InstanteHecho, true
		}
	}
	return "", time.Time{}, false
}

// AplicarCursos pliega los cursos sobre un plan y devuelve los pasos con su
// estado final.
//
// Los cursos van en un mapa POR NIVEL, que es la identidad del paso dentro de
// su obligacion e hito. No por indice del slice: reordenar el plan movera el
// emparejamiento entero y nadie lo notaria (invariante 7). El nivel se calcula
// del instante, que es un dato del propio paso.
//
// UN CURSO SOBRE UN PASO QUE NO IBA A ENVIAR ES UNA CONTRADICCION y se dice:
// significa que algo mando un aviso que el plan habia descartado (colapsado,
// en silencio o sin destinatario), y eso es exactamente lo que no puede pasar
// sin que alguien se entere.
func AplicarCursos(pasos []Paso, cursos map[int]*Curso) ([]Paso, error) {
	out := append([]Paso(nil), pasos...)
	for i := range out {
		c, hay := cursos[out[i].Nivel]
		if !hay || c == nil {
			continue
		}
		e, tiene := c.Estado()
		if !tiene {
			continue
		}
		if out[i].Aviso == nil {
			return nil, fmt.Errorf("el escalon %d de %s tiene curso de envio (%s) y el plan "+
				"lo habia dejado en %q: se ha mandado un aviso que nadie planifico",
				out[i].Nivel, out[i].Figura, e, out[i].Estado)
		}
		out[i].Estado = e
		if e == Fallido {
			for _, s := range c.Sucesos() {
				if s.Tipo == Fallo {
					out[i].Motivo = "la entrega fallo por " + s.Canal + ": " + s.Motivo
				}
			}
		}
	}
	return out, nil
}
