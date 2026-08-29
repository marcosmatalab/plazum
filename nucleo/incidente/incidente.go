// Package incidente es el objeto con identidad de la etapa 4.
//
// POR QUE EXISTE, Y QUE ESTABA ESPERANDOLO. Tres piezas del plan lo consumian y
// ninguna etapa lo construia: el cronometro del art. 33 del RGPD, la clase
// notificatoria (19 de las 21 obligaciones notificatorias del corpus son
// notificaciones de incidente) y el acta 9.3. Hasta hoy los treinta y tres
// relojes de la familia A solo disparaban dentro de un caso dorado, porque el
// hecho de arranque no tenia de donde salir.
//
// LO QUE ES, EN UNA FRASE: un objeto con identidad y un FLUJO DE HECHOS
// INMUTABLES. No es un registro que se edita. Reclasificar no cambia la
// clasificacion anterior, anade una nueva con su instante, y los plazos se
// recalculan solos porque el motor ya elige por la clasificacion mas reciente
// (ver ventana.claseVigente). Es la doctrina de Hito.Clase aplicada al objeto
// entero: nada se corrige encima, todo se corrige AL LADO.
//
// LOS DOS INSTANTES DEL NACIMIENTO, y son dos porque la norma los distingue.
// Cuando ocurrio y cuando se supo no son lo mismo, y el plazo de 72 horas del
// art. 33 del RGPD cuenta desde el SEGUNDO ("desde que haya tenido constancia
// de ella"). Un incidente ocurrido en marzo y descubierto en julio no lleva
// cuatro meses de incumplimiento: lleva las horas que van desde que se supo.
// Confundirlos acusa en falso, que es el unico error que un producto de
// cumplimiento no puede cometer ni una vez.
//
// El contrato bitemporal es el de nucleo/historia, que ya lo nombra en su
// cabecera: InstanteHecho es el eje del mundo, InstanteRegistro el del sistema,
// y una correccion es un evento mas y no una reescritura. Lo que NO se hace es
// meter el ciclo de vida del incidente dentro de estado.Estado: ese vocabulario
// es cerrado, describe el estado de una PRUEBA y lo consumen el expediente y el
// MTTR. Un incidente abierto, clasificado, reclasificado, notificado y cerrado
// no tiene imagen ahi, y las que casi valen ("falla dentro del SLA" para un
// incidente recien conocido) dejan de valer en el segundo suceso. Doblar el
// modelo interno para que quepa en una forma ajena es el error de D-1. Cuando
// el incidente SI cambia el estado de una prueba, eso va a historia, que es de
// quien es.
//
// DE DONDE SALE LA IDENTIDAD (D-18, medido en nucleo/corpus/por_objeto_test.go
// antes de escribir esto). Un reloj por objeto se parte en dos mitades que no
// se pueden confundir: el PAQUETE escribe el nombre del hecho ("constancia",
// "incidente.nivel.alto"), porque es lo unico que la norma dice; el INCIDENTE
// aporta la instancia, porque es lo unico que el paquete no puede saber. Este
// paquete es la segunda mitad. Sin el, dos incidentes bajo el mismo nombre de
// hecho se pisaban en el mapa y el motor daba UNA fecha: medido, el que
// desaparecia vencia nueve dias antes.
//
// LO QUE ESTE PAQUETE NO DECIDE NUNCA: si un incidente es grave, significativo
// o notificable. Eso son los arts. 3 a 14 del Reglamento de Ejecucion (UE)
// 2024/2690 y equivalentes, y viven en su paquete de datos como cualquier otra
// norma (invariante 2). Aqui solo consta QUIEN clasifico, COMO y CUANDO.
package incidente

import (
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/marcosmatalab/plazum/nucleo/ventana"
)

// Tipo es el vocabulario CERRADO de sucesos de un incidente.
//
// Cerrado a proposito: un tipo libre convertiria este flujo en un cajon de
// notas, y lo que sostiene el calculo del plazo es que apertura, clasificacion
// y notificacion signifiquen exactamente una cosa.
type Tipo uint8

const (
	// Apertura es el nacimiento, y es unico. Trae los dos instantes.
	Apertura Tipo = iota
	// Clasificacion es lo que el OBLIGADO declara del incidente. Puede haber
	// varias: reclasificar es un suceso mas, nunca una edicion.
	Clasificacion
	// Notificacion es la remision efectiva de un hito notificatorio: la que
	// hace que el siguiente hito de la cadena arranque de verdad (una
	// notificacion intermedia cuenta desde la REMISION de la inicial, no desde
	// cuando deberia haberse remitido).
	Notificacion
	// Cierre es la constancia de que el incidente termino. No apaga los
	// plazos vencidos ni borra nada: sigue siendo un suceso mas.
	Cierre
)

var nombresDeTipo = [...]string{"apertura", "clasificacion", "notificacion", "cierre"}

func (t Tipo) String() string {
	if !t.Valido() {
		// Sin esto, un Tipo fuera de rango entra en panico al formatearse, y
		// el sitio donde se formatea suele ser justo el mensaje de error que
		// explicaba el problema.
		return fmt.Sprintf("tipo desconocido(%d)", uint8(t))
	}
	return nombresDeTipo[t]
}

func (t Tipo) Valido() bool { return int(t) < len(nombresDeTipo) }

// Suceso es un hecho del incidente, y es INMUTABLE por construccion: se
// registra y ya no se toca. No hay borrado ni edicion, igual que en historia.
type Suceso struct {
	Tipo Tipo

	// Clase es OBLIGATORIA en Clasificacion y esta PROHIBIDA en el resto.
	//
	// Es el nombre del hecho que espera el paquete, no una palabra nuestra: el
	// paquete escribe "incidente.nivel.alto" en Hito.Clase y aqui se dice
	// cuando consta esa clasificacion. Escribir aqui un vocabulario propio y
	// traducirlo despues seria cablear normas en el codigo (invariante 2) y
	// ademas necesitaria una tabla de traduccion, que es un sitio nuevo donde
	// perder un caso.
	Clase string

	// Hito es OBLIGATORIO en Notificacion y esta PROHIBIDO en el resto: el id
	// del hito que se remitio, tal y como lo nombra la obligacion.
	Hito string

	// Los dos ejes, con el mismo significado que en historia.CambioEstado.
	// Ninguno puede ser el cero: un instante cero es una fecha de 1 de enero
	// del ano 1 con cara de dato, y de ahi salen plazos vencidos hace dos mil
	// anos.
	InstanteHecho    time.Time
	InstanteRegistro time.Time

	// Fuente es quien lo registra. Va en el objeto y no en un log porque la
	// linea de tiempo que ve el auditor tiene que decir de quien es cada dato.
	Fuente string
}

var (
	ErrSinIdentidad          = errors.New("un incidente sin identidad no puede llevar relojes")
	ErrTipoDesconocido       = errors.New("tipo de suceso fuera del vocabulario")
	ErrInstanteCero          = errors.New("instante cero")
	ErrRegistroAntesDelHecho = errors.New("el registro es anterior al hecho")
	ErrSegundaApertura       = errors.New("un incidente nace una sola vez")
	ErrSinApertura           = errors.New("incidente sin apertura")
	ErrCampoQueFalta         = errors.New("falta un campo obligatorio para este tipo")
	ErrCampoAjeno            = errors.New("campo que no corresponde a este tipo")
)

// Incidente es el objeto. Sus sucesos son privados y no hay setter: la unica
// forma de cambiarlo es anadir.
type Incidente struct {
	id      string
	sucesos []Suceso
}

// Abrir es el UNICO constructor, y por eso el valor cero de Incidente es
// inservible a proposito (invariante 8): un Incidente{} no tiene apertura, y
// todos los metodos que dan un dato con consecuencia legal lo dicen en vez de
// devolver el cero. Un instante cero devuelto en silencio es un plazo vencido
// en el ano 1.
//
// ocurrio es cuando paso en el mundo; seSupo es cuando la organizacion tuvo
// constancia, que es de donde cuenta el art. 33.
func Abrir(id string, ocurrio, seSupo time.Time, fuente string) (*Incidente, error) {
	if id == "" {
		return nil, fmt.Errorf("%w: la identidad es lo que separa un incidente de otro y lo "+
			"que hace que sus plazos no se pisen. Arreglo: dale un id estable (por ejemplo "+
			"el numero de expediente interno)", ErrSinIdentidad)
	}
	i := &Incidente{id: id}
	if err := i.Registrar(Suceso{
		Tipo: Apertura, InstanteHecho: ocurrio, InstanteRegistro: seSupo, Fuente: fuente,
	}); err != nil {
		return nil, err
	}
	return i, nil
}

func (i *Incidente) ID() string { return i.id }

// Abierto dice si el objeto es utilizable. Un Incidente que no ha pasado por
// Abrir no lo es.
func (i *Incidente) Abierto() bool {
	return i != nil && len(i.sucesos) > 0 && i.sucesos[0].Tipo == Apertura
}

// Registrar anade un suceso. No hay Modificar ni Borrar, y esa ausencia es el
// diseno: una correccion es un suceso mas con su propio instante de registro.
func (i *Incidente) Registrar(s Suceso) error {
	if !s.Tipo.Valido() {
		return fmt.Errorf("%w: %s. Los que hay son %v", ErrTipoDesconocido, s.Tipo, nombresDeTipo)
	}
	if s.InstanteHecho.IsZero() || s.InstanteRegistro.IsZero() {
		return fmt.Errorf("%w en el suceso %s: los dos ejes son obligatorios. Cuando ocurrio "+
			"y cuando se supo no son lo mismo, y hay plazos que cuentan del segundo (art. 33 "+
			"del RGPD). Si solo se sabe uno, se registra el que se sabe en los dos y se dice "+
			"en la fuente", ErrInstanteCero, s.Tipo)
	}
	// EL RELOJ QUE MIENTE, en su forma barata: no se puede tener constancia de
	// algo antes de que pase. Sin esta guarda, un registro anterior al hecho da
	// un plazo que vence antes de arrancar.
	if s.InstanteRegistro.Before(s.InstanteHecho) {
		return fmt.Errorf("%w en el suceso %s: consta registrado el %s y ocurrido el %s. "+
			"Nadie sabe algo antes de que ocurra. Arreglo: si el dato viene de un sistema con "+
			"la hora mal, corregir la hora, no el orden", ErrRegistroAntesDelHecho, s.Tipo,
			s.InstanteRegistro.Format(time.RFC3339), s.InstanteHecho.Format(time.RFC3339))
	}
	if err := camposDe(s); err != nil {
		return err
	}
	if s.Tipo == Apertura && len(i.sucesos) > 0 {
		return fmt.Errorf("%w: el incidente %s ya tiene apertura. Si lo que se quiere es "+
			"corregir la fecha en que se supo, eso es un suceso de correccion, y hoy no "+
			"existe: se abre un incidente nuevo o se anota en la fuente", ErrSegundaApertura, i.id)
	}
	if s.Tipo != Apertura && !i.Abierto() {
		return fmt.Errorf("%w: no se puede registrar un %s en un incidente que no se ha "+
			"abierto. Arreglo: usar Abrir", ErrSinApertura, s.Tipo)
	}
	i.sucesos = append(i.sucesos, s)
	return nil
}

// camposDe comprueba que cada tipo trae LO SUYO y no lo ajeno.
//
// Las dos direcciones, no una: un campo que falta deja el suceso sin poder
// usarse, y un campo ajeno (una clase en una notificacion) es un dato escrito
// con cuidado que nadie lee nunca, que es la clase del campo huerfano.
func camposDe(s Suceso) error {
	quiere := func(nombre, valor string, hace bool) error {
		switch {
		case hace && valor == "":
			return fmt.Errorf("%w: un suceso de %s necesita %s", ErrCampoQueFalta, s.Tipo, nombre)
		case !hace && valor != "":
			return fmt.Errorf("%w: un suceso de %s no lleva %s (vale %q), y escribirlo ahi "+
				"deja un dato que nadie lee", ErrCampoAjeno, s.Tipo, nombre, valor)
		}
		return nil
	}
	if err := quiere("clase", s.Clase, s.Tipo == Clasificacion); err != nil {
		return err
	}
	return quiere("hito", s.Hito, s.Tipo == Notificacion)
}

// Sucesos devuelve una COPIA ordenada por el eje del mundo, con el mismo
// criterio estable que historia.porHecho: dos sucesos del mismo instante
// mantienen el orden en que se registraron.
//
// Copia, y no el slice interno, porque devolver el interno es dar una via de
// edicion a un objeto cuya propiedad entera es que no se edita.
func (i *Incidente) Sucesos() []Suceso {
	out := append([]Suceso(nil), i.sucesos...)
	sort.SliceStable(out, func(a, b int) bool {
		return out[a].InstanteHecho.Before(out[b].InstanteHecho)
	})
	return out
}

// Ocurrio es el instante del mundo en que empezo el incidente.
func (i *Incidente) Ocurrio() (time.Time, bool) {
	if !i.Abierto() {
		return time.Time{}, false
	}
	return i.sucesos[0].InstanteHecho, true
}

// PrimerConocimiento es el instante de REGISTRO de la apertura: el "tener
// constancia" del art. 33 del RGPD, de donde arrancan las 72 horas.
//
// Aqui no hay nada que buscar y por eso no se reimplementa el barrido de
// historia.PrimerConocimiento: aquella funcion existe para encontrar el primer
// evento que llevo una prueba a un estado, entre muchos. La apertura de un
// incidente es unica y esta identificada por su tipo, asi que la respuesta es
// el dato, no una busqueda.
func (i *Incidente) PrimerConocimiento() (time.Time, bool) {
	if !i.Abierto() {
		return time.Time{}, false
	}
	return i.sucesos[0].InstanteRegistro, true
}

// ClaseEn devuelve la clasificacion vigente en un instante del MUNDO, y si hay
// EMPATE (dos clasificaciones distintas con el mismo instante).
//
// El empate no se resuelve: se dice. Es una contradiccion del dato, y elegir
// una de las dos daria un plazo distinto en cada ejecucion segun el orden del
// recorrido. Es la misma decision que ventana.claseVigente toma para el mismo
// caso, y por el mismo motivo.
func (i *Incidente) ClaseEn(t time.Time) (clase string, empate bool, ok bool) {
	var cuando time.Time
	for _, s := range i.sucesos {
		if s.Tipo != Clasificacion || s.InstanteHecho.After(t) {
			continue
		}
		switch {
		case !ok:
			clase, cuando, empate, ok = s.Clase, s.InstanteHecho, false, true
		case s.InstanteHecho.After(cuando):
			clase, cuando, empate = s.Clase, s.InstanteHecho, false
		case s.InstanteHecho.Equal(cuando) && s.Clase != clase:
			empate = true
		}
	}
	return clase, empate, ok
}

// Notificado dice si CONSTA la remision de un hito, y cuando.
//
// EL NOMBRE Y LA FRASE IMPORTAN. Devuelve "consta" y no "se hizo": que no
// conste una notificacion no dice que no se hiciera, dice que en las respuestas
// del cliente no aparece. Toda pantalla que lo pinte tiene que decirlo asi
// (doctrina del falso positivo, CLAUDE.md).
func (i *Incidente) Notificado(hito string) (time.Time, bool) {
	var primero time.Time
	ok := false
	for _, s := range i.sucesos {
		if s.Tipo != Notificacion || s.Hito != hito {
			continue
		}
		if !ok || s.InstanteHecho.Before(primero) {
			primero, ok = s.InstanteHecho, true
		}
	}
	return primero, ok
}

// Hechos traduce el incidente a los hechos que espera un reloj del corpus.
//
// ES LA JUNTA DE D-18, y por eso lleva el nombre del hecho como parametro en
// vez de inventarselo: el disparador lo escribe el PAQUETE (es lo unico que la
// norma dice) y el instante lo pone el INCIDENTE (es lo unico que el paquete no
// puede saber). Si esta funcion se inventara el nombre, el paquete tendria que
// conocer los objetos del cliente, que es justo lo que no puede hacer.
//
// Emite tres cosas, y las tres las consume ya el motor tal cual:
//
//	disparador          -> el primer conocimiento (de aqui cuentan las 72 h)
//	<clase>             -> el instante de cada clasificacion, para Hito.Clase
//	<hito>.cumplido     -> la remision efectiva, para los hitos encadenados
//
// La tercera es la que hace que una notificacion intermedia cuente desde la
// REMISION de la inicial y no desde cuando deberia haberse remitido, que es lo
// que la familia A pidio cuando se midio.
func (i *Incidente) Hechos(disparador string) (ventana.Hechos, error) {
	if !i.Abierto() {
		return nil, fmt.Errorf("%w: el incidente %q no se ha abierto, asi que no tiene "+
			"primer conocimiento y ningun reloj puede arrancar", ErrSinApertura, i.id)
	}
	if disparador == "" {
		// El valor cero permisivo seria escribir el instante bajo la clave
		// vacia: el reloj no arrancaria y nadie sabria por que (invariante 8).
		return nil, fmt.Errorf("%w: falta el nombre del hecho de arranque, que lo pone el "+
			"paquete (Temporalidad.Disparador). Sin el, el instante se escribiria bajo una "+
			"clave que ningun reloj busca", ErrCampoQueFalta)
	}
	h := ventana.Hechos{disparador: i.sucesos[0].InstanteRegistro}
	for _, s := range i.sucesos {
		switch s.Tipo {
		case Clasificacion:
			// La MAS RECIENTE manda, y el motor ya lo resuelve solo mirando el
			// instante: reclasificar es escribir otra clave con otra fecha, no
			// pisar la anterior. Si la misma clase consta dos veces, vale la
			// primera vez que consto.
			if ya, esta := h[s.Clase]; !esta || s.InstanteHecho.Before(ya) {
				h[s.Clase] = s.InstanteHecho
			}
		case Notificacion:
			k := s.Hito + ".cumplido"
			if ya, esta := h[k]; !esta || s.InstanteHecho.Before(ya) {
				h[k] = s.InstanteHecho
			}
		}
	}
	return h, nil
}

// Campo contesta al vocabulario cerrado de los origenes `incidente:` de una
// plantilla. El nombre llega SIN el prefijo: quien lo lee es corpus.ParseOrigen,
// y este paquete no conoce la gramatica de las plantillas.
//
// Devuelve tambien si el dato CONSTA, y la palabra importa: que no conste una
// clasificacion no dice que el incidente no sea grave, dice que en las
// respuestas del cliente no hay ninguna. Un entregable que rellenara ese hueco
// con el valor cero estaria afirmando algo que nadie ha dicho.
func (i *Incidente) Campo(nombre string) (string, bool) {
	if !i.Abierto() {
		return "", false
	}
	switch nombre {
	case "id":
		return i.id, true
	case "ocurrio":
		return i.sucesos[0].InstanteHecho.Format(time.RFC3339), true
	case "primer_conocimiento":
		return i.sucesos[0].InstanteRegistro.Format(time.RFC3339), true
	case "clasificacion_vigente":
		// La vigente es la mas reciente que consta, mirada desde el ultimo
		// suceso registrado y no desde "ahora": nucleo/ no llama a time.Now().
		var ultimo time.Time
		for _, s := range i.sucesos {
			if s.InstanteHecho.After(ultimo) {
				ultimo = s.InstanteHecho
			}
		}
		clase, empate, ok := i.ClaseEn(ultimo)
		if !ok || empate {
			// Un empate NO se resuelve para rellenar un hueco: se deja sin
			// constar, que es lo que es.
			return "", false
		}
		return clase, true
	}
	return "", false
}
