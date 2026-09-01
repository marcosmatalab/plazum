// Package acta compone el acta de la revision por la direccion, que es EL
// documento que sale de plazum hacia gente que no ha visto plazum nunca.
//
// QUE ES Y QUE NO ES. No es un informe mas: es el unico entregable de este
// producto que lee un organo de gobierno. Lo que se juega aqui no es que el dato
// sea correcto (eso lo deciden los tres objetos que lo alimentan), es que
// CUALQUIER NUMERO SE PUEDA SEGUIR HASTA LO QUE LO COMPONE sin fiarse de nadie.
// Un acta con cifras que hay que creerse es exactamente el documento que hoy
// firma todo el mundo sin leerlo.
//
// # Las tres reglas de este paquete, y las tres son estructurales
//
//  1. NINGUN NUMERO EXISTE SIN SU DERIVACION. Una Cifra no lleva un entero: es
//     la lista de lo que cuenta, y su valor SE DERIVA de la lista. No hay forma
//     de escribir aqui un numero y olvidarse de decir de que sale, porque no hay
//     campo donde escribirlo. Encima, cada reparto se contrasta contra el
//     recuento que hace su propio objeto (Programa.Cuenta, Campana.Cuenta), que
//     es un segundo camino ciego al mismo numero: si el acta enumera un conjunto
//     y el objeto cuenta otro, no cuadra y se dice.
//
//  2. TODA PROSA DICE DE DONDE SALEN SUS PALABRAS. Un Parrafo lleva su
//     Procedencia, y el vocabulario tiene exactamente tres valores: lo escribe
//     plazum (y entonces esta en este repositorio, palabra por palabra), lo
//     escribe una persona (y consta quien), o es una cita de la norma (y consta
//     cual). NO HAY UN CUARTO VALOR, y esa ausencia es el diseno: este es el
//     documento que mas tentacion va a tener de prosa generada, y la respuesta
//     no es una advertencia en una guia de estilo, es que no exista el sitio
//     donde meterla. El acta se compone entera dentro de nucleo/, que no importa
//     el puerto de IA ni lo nombra (invariante 9), asi que el board pack sale
//     igual con la IA apagada: no es que se degrade, es que nunca la uso.
//
//  3. LO NO CONSTATADO NO ACUSA, Y LO DICE PEGADO AL DATO. Cada seccion trae su
//     cubo de "no consta" con su frase al lado, y la frase NO se reescribe aqui:
//     se toma del paquete que produce el dato (accesos.LaFraseDeLoNoRevisado,
//     auditoria.LaFraseDeLoNoAuditado, incidente.LaFraseDeLoNoNotificado). Una
//     frase que vive en dos sitios se corrige en uno.
//
// # El vocabulario de cubos esta completo aunque este vacio
//
// Un cubo que solo aparece cuando tiene algo dentro es un cubo que nadie echa de
// menos. Vale igual para una SECCION entera: si no hay campana de accesos, la
// seccion sale igual, diciendo que no la hay y que hace falta para tenerla. Una
// seccion que desaparece deja un acta que parece completa y no lo esta.
//
// # Los emparejamientos, y las dos direcciones de cada uno (invariante 7)
//
// Este paquete cruza conjuntos que vienen de sitios distintos, y ninguno casa
// por posicion:
//
//	responsables -> unidades   por Unidad.Clave() (paquete|obligacion), que sale
//	                           del corpus. La direccion que se olvida es la otra:
//	                           un responsable asignado a una unidad que este
//	                           programa no tiene. No se traga: se cuenta y se
//	                           dice, en su propio reparto.
//	esperadas    -> incidentes por Incidente.ID, que es la misma identidad que
//	                           lleva cada suceso del incidente. Igual: una
//	                           notificacion esperada de un incidente que no esta
//	                           en el acta se cuenta y se dice.
//	sesiones     -> unidades   ya lo guarda auditoria.Auditar, que rechaza una
//	                           sesion sobre algo fuera del alcance. Aqui no se
//	                           vuelve a comprobar: se apoya en esa guarda y se
//	                           dice que se apoya.
//
// # De que verbo cuelga cada instante
//
// Se dice porque escoger mal no da error en ningun sitio: da un acta que cuadra
// y miente.
//
//	un incidente es DE ESTE PERIODO si su PRIMER CONOCIMIENTO cae dentro. No si
//	ocurrio dentro: un incidente de marzo que se supo en febrero del ano
//	siguiente es de la revision del ano siguiente, que es cuando la organizacion
//	pudo hacer algo. Es la distincion del art. 33 que ya hace nucleo/incidente,
//	aplicada al periodo en vez de al plazo.
//
//	la clasificacion vigente se lee CON LOS OJOS DEL FINAL DEL PERIODO, no con
//	los de hoy. Una reclasificacion posterior no reescribe el acta de un periodo
//	cerrado: es la bitemporalidad de nucleo/historia aplicada aqui.
package acta

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

var (
	// ErrActa: falta algo para componer.
	ErrActa = errors.New("acta: faltan datos para componer el acta")
	// ErrProsaSinProcedencia: un parrafo que no dice de donde salen sus
	// palabras. Es el centinela de la regla 2.
	ErrProsaSinProcedencia = errors.New("acta: prosa sin procedencia")
	// ErrAusenciaSinDescargo: un cubo de "no consta" sin la frase al lado. Es el
	// centinela de la regla 3.
	ErrAusenciaSinDescargo = errors.New("acta: un dato que falta, presentado sin su descargo")
	// ErrSeccionSinDescargo: una seccion entera sin ningun cubo de ausencia.
	ErrSeccionSinDescargo = errors.New("acta: una seccion sin ningun cubo de lo que no consta")
	// ErrSeccionVaciaSinMotivo: una fuente que no se aporto y no se dice que
	// hace falta. Un estado vacio sin verbo es un callejon (D11-b).
	ErrSeccionVaciaSinMotivo = errors.New("acta: una fuente que falta y no dice que hace falta")
)

// Periodo es lo que cubre el acta. Los dos extremos entran, con el mismo
// criterio que auditoria.Ciclo.Cubre: un incidente conocido el ultimo dia del
// periodo es del periodo.
type Periodo struct {
	Desde time.Time
	Hasta time.Time
}

func (p Periodo) valido() bool {
	return !p.Desde.IsZero() && !p.Hasta.IsZero() && p.Hasta.After(p.Desde)
}

// Cubre dice si un instante cae dentro.
func (p Periodo) Cubre(t time.Time) bool {
	return !t.Before(p.Desde) && !t.After(p.Hasta)
}

func (p Periodo) String() string {
	return p.Desde.Format("2006-01-02") + " a " + p.Hasta.Format("2006-01-02")
}

// Fuente es de que objeto sale una seccion. Vocabulario CERRADO.
//
// Cerrado por lo mismo que los demas de la casa: una seccion cuya fuente sea
// texto libre es una seccion que manana puede salir de cualquier sitio, y de
// donde sale cada numero de este documento es justo lo que no puede quedar al
// gusto de quien lo compone.
type Fuente string

const (
	// DelProgramaDeAuditoria: nucleo/auditoria.
	DelProgramaDeAuditoria Fuente = "programa de auditoria interna"
	// DeLaCampanaDeAccesos: nucleo/accesos.
	DeLaCampanaDeAccesos Fuente = "campana de revision de accesos"
	// DeLosIncidentes: nucleo/incidente.
	DeLosIncidentes Fuente = "registro de incidentes"
	// DeLaDireccion: no sale de ningun objeto. Lo escribe quien firma, y por eso
	// esta en el vocabulario: la parte que ninguna maquina produce tiene que
	// tener nombre, o parece que falta por descuido.
	DeLaDireccion Fuente = "lo que decide la direccion"
)

// FuentesPosibles es el vocabulario entero y EN ORDEN DE LECTURA. El acta lleva
// SIEMPRE las cuatro secciones, tenga datos o no.
func FuentesPosibles() []Fuente {
	return []Fuente{DelProgramaDeAuditoria, DeLaCampanaDeAccesos, DeLosIncidentes, DeLaDireccion}
}

// Procedencia dice de donde salen las palabras de un parrafo.
//
// TRES VALORES, Y NO HAY UN CUARTO. La ausencia es el diseno: ver la regla 2 de
// la cabecera del paquete.
type Procedencia uint8

const (
	// DePlazum: constante de este repositorio. Se puede leer en el codigo,
	// palabra por palabra, y no cambia entre dos ejecuciones.
	DePlazum Procedencia = iota
	// DeUnaPersona: lo escribio alguien y consta quien.
	DeUnaPersona
	// DeLaNorma: cita del corpus, con el paquete y la obligacion de la que sale.
	DeLaNorma
)

var nombresDeProcedencia = [...]string{"lo escribe plazum", "lo escribe una persona",
	"es una cita de la norma"}

func (p Procedencia) String() string {
	if int(p) < len(nombresDeProcedencia) {
		return nombresDeProcedencia[p]
	}
	return fmt.Sprintf("procedencia desconocida (%d)", uint8(p))
}

// Valida dice si el valor esta dentro del vocabulario.
func (p Procedencia) Valida() bool { return int(p) < len(nombresDeProcedencia) }

// ProcedenciasPosibles es el vocabulario entero.
func ProcedenciasPosibles() []Procedencia { return []Procedencia{DePlazum, DeUnaPersona, DeLaNorma} }

// Parrafo es prosa del acta CON SU PROCEDENCIA.
type Parrafo struct {
	Texto string
	De    Procedencia
	// Quien es obligatorio cuando De == DeUnaPersona. Sin el, "lo escribio
	// alguien" no es mejor que no decir nada.
	Quien string
	// Cita es obligatoria cuando De == DeLaNorma: de donde sale, tal y como lo
	// escribio el paquete de la norma.
	Cita string
}

func (p Parrafo) validar() error {
	if strings.TrimSpace(p.Texto) == "" {
		return fmt.Errorf("%w: un parrafo sin texto", ErrProsaSinProcedencia)
	}
	if !p.De.Valida() {
		return fmt.Errorf("%w: %q no dice de donde salen sus palabras. Las procedencias son: %s",
			ErrProsaSinProcedencia, corto(p.Texto), strings.Join(nombresDeProcedencia[:], ", "))
	}
	if p.De == DeUnaPersona && strings.TrimSpace(p.Quien) == "" {
		return fmt.Errorf("%w: %q dice que lo escribio una persona y no dice cual.\n"+
			"  En el documento que lee un consejo, un parrafo sin autor es un parrafo del que "+
			"nadie responde", ErrProsaSinProcedencia, corto(p.Texto))
	}
	if p.De == DeLaNorma && strings.TrimSpace(p.Cita) == "" {
		return fmt.Errorf("%w: %q se presenta como cita y no dice de donde sale.\n"+
			"  Una cita sin origen, en un acta de cumplimiento, es peor que no citar",
			ErrProsaSinProcedencia, corto(p.Texto))
	}
	return nil
}

// Elemento es una de las cosas que componen un numero. Es LO QUE SE VE AL CLICAR.
type Elemento struct {
	// Clave es la identidad DENTRO de su fuente: paquete|obligacion,
	// sistema|cuenta|permiso, el id del incidente, el numero de linea. Nunca un
	// indice de una lista (invariante 7): las listas se reordenan.
	Clave string
	// Que es para que una persona lo reconozca. NO identifica y no empareja.
	Que string
	// Nota es el matiz que va pegado a ESTA fila y no al cubo entero: cuantos
	// ciclos lleva, quien lo decidio, cuando se remitio.
	//
	// AQUI TAMBIEN VIAJA PROSA DE UNA PERSONA (el motivo de un diferimiento, el
	// texto de un hallazgo, el de una excusa), y es la via por la que la regla 2
	// se podria colar: una Nota no es un Parrafo y no lleva Procedencia. La regla
	// que la cierra es que toda Nota que embeba palabras de alguien las pone
	// DETRAS DE SU NOMBRE, en la misma cadena ("diferida por X el D: <motivo>"),
	// asi que la atribucion viaja pegada al texto y no se puede separar de el al
	// pintarla. Lo vigila un test que recorre las notas de un acta con las tres
	// clases de prosa ajena dentro.
	Nota string
}

// Cifra es un numero del acta CON SU DERIVACION.
//
// NO TIENE CAMPO PARA EL VALOR, y eso es la regla 1 hecha estructura: el valor
// se deriva de la lista, asi que no hay manera de imprimir aqui un numero que no
// se pueda abrir. La forma de romper D11-c en este paquete seria anadirle un
// entero.
type Cifra struct {
	// Cubo es como se llama dentro de su reparto.
	Cubo      string
	Elementos []Elemento
	// Descargo es la frase que va PEGADA al dato para que no se lea como culpa.
	//
	// Cubre dos casos, y los dos se leen como acusacion si van solos: un cubo de
	// LO QUE NO CONSTA ("no se reviso" no es "esta mal") y un cubo de LO QUE
	// CONSTA Y NO ES CULPA (el auditor que audita su propia unidad, que en una
	// empresa de veinte personas puede ser la unica respuesta posible).
	Descargo string
	// esAusencia marca que este cubo cuenta cosas de las que plazum NO tiene
	// registro. exigeDescargo marca que el cubo se lee como culpa si va solo, que
	// es lo que obliga a la frase, y lo llevan tanto las ausencias como los cubos
	// de lo que consta y no es un incumplimiento.
	//
	// Ninguno de los dos se exporta como campo: las cifras las construye este
	// paquete con recuento(), ausencia() o noEsCulpa(), y las dos ultimas son las
	// unicas puertas, las dos con la frase por parametro obligatorio.
	esAusencia    bool
	exigeDescargo bool
}

// Valor es el numero que se imprime, y es len(Elementos) por construccion.
func (c Cifra) Valor() int { return len(c.Elementos) }

// EsAusencia dice si el cubo cuenta cosas que no constan.
func (c Cifra) EsAusencia() bool { return c.esAusencia }

// recuento arma un cubo que no acusa de nada.
func recuento(cubo string, es []Elemento) Cifra {
	return Cifra{Cubo: cubo, Elementos: ordenarElementos(es)}
}

// ausencia arma un cubo de lo que no consta, y EXIGE la frase.
//
// Es la puerta de la regla 3, y esta en el constructor y no en un test: una
// comprobacion que solo hace el test no protege al que llama.
func ausencia(cubo string, es []Elemento, descargo string) Cifra {
	return Cifra{Cubo: cubo, Elementos: ordenarElementos(es), Descargo: descargo,
		esAusencia: true, exigeDescargo: true}
}

// noEsCulpa arma un cubo de algo que CONSTA y que no es un incumplimiento.
//
// Se separa de recuento() a proposito: la diferencia entre "aqui hay un dato" y
// "aqui hay un dato que se va a leer mal" es la que decide si hace falta la
// frase, y dejarla al criterio de quien escribe el cubo es como se pierde.
func noEsCulpa(cubo string, es []Elemento, descargo string) Cifra {
	return Cifra{Cubo: cubo, Elementos: ordenarElementos(es), Descargo: descargo,
		exigeDescargo: true}
}

// Reparto es una particion: un universo y los cubos en los que cae, TODOS.
type Reparto struct {
	// Rotulo dice QUE se reparte. Sin esto, dos repartos de la misma seccion se
	// leen como si contaran lo mismo.
	Rotulo string
	// Universo es cuantas cosas hay que repartir, CONTADAS APARTE. Es el segundo
	// camino ciego: si el acta enumera un conjunto y el objeto cuenta otro,
	// Cuadra() lo dice.
	Universo int
	// DeDonde explica de donde sale el universo, para que quien lea un descuadre
	// sepa donde mirar.
	DeDonde string
	Cifras  []Cifra
}

// Suma es lo que suman los cubos.
func (r Reparto) Suma() int {
	n := 0
	for _, c := range r.Cifras {
		n += c.Valor()
	}
	return n
}

// Cuadra dice si los cubos suman el universo. Si es falso, el acta NO vale: no
// es un detalle de presentacion, es que no sabe de que esta hablando.
func (r Reparto) Cuadra() bool { return r.Suma() == r.Universo }

// Seccion es un bloque del acta. SIEMPRE hay una por fuente.
type Seccion struct {
	Fuente Fuente
	// Aportada es falso cuando nadie ha dado esta fuente. La seccion sale igual.
	Aportada bool
	// PorQueFalta es obligatorio cuando !Aportada: que es lo que no hay y que
	// haria falta para tenerlo.
	PorQueFalta string
	Repartos    []Reparto
	Parrafos    []Parrafo
}

// Descargos son las frases de esta seccion que van pegadas a un dato, en orden
// estable y sin repetir.
func (s Seccion) Descargos() []string {
	visto := map[string]bool{}
	var out []string
	for _, r := range s.Repartos {
		for _, c := range r.Cifras {
			if c.Descargo == "" || visto[c.Descargo] {
				continue
			}
			visto[c.Descargo] = true
			out = append(out, c.Descargo)
		}
	}
	return out
}

// Obligacion es una obligacion del corpus de la que este acta es evidencia.
//
// Llega COMO DATO, entera, porque este paquete no puede nombrar ninguna norma
// (invariante 2) y porque la cita la escribe el paquete de la norma, no nosotros.
type Obligacion struct {
	Paquete string
	Version string
	ID      string
	Titulo  string
	Cita    string
}

// Acta es el documento compuesto.
type Acta struct {
	ID           string
	Organizacion string
	Periodo      Periodo
	// Cubre son las obligaciones de las que este acta es evidencia.
	Cubre []Obligacion
	// Cabecera es lo que se lee antes de los numeros.
	Cabecera  []Parrafo
	Secciones []Seccion
}

// Descuadres devuelve los repartos que no cuadran, con su seccion.
func (a Acta) Descuadres() []string {
	var out []string
	for _, s := range a.Secciones {
		for _, r := range s.Repartos {
			if r.Cuadra() {
				continue
			}
			out = append(out, fmt.Sprintf("%s / %s: los cubos suman %d y el universo es %d (%s)",
				s.Fuente, r.Rotulo, r.Suma(), r.Universo, r.DeDonde))
		}
	}
	return out
}

// Cuadra es la ley de conservacion del acta entera.
func (a Acta) Cuadra() bool { return len(a.Descuadres()) == 0 }

// Busca una seccion por su fuente. Siempre existe: el acta lleva las cuatro.
func (a Acta) Busca(f Fuente) (Seccion, bool) {
	for _, s := range a.Secciones {
		if s.Fuente == f {
			return s, true
		}
	}
	return Seccion{}, false
}

// CifraSituada es una cifra con el sitio del que sale, para poder recorrerlas
// todas sin perder de que seccion y de que reparto es cada una.
type CifraSituada struct {
	// Ref es la referencia estable con la que se abre: seccion.reparto.cubo.
	Ref     string
	Fuente  Fuente
	Reparto string
	Cifra   Cifra
}

// Cifras devuelve todas las cifras del acta, en orden de lectura. Es lo que
// recorre la puerta de D11-c: ningun numero de este documento se imprime sin
// poder abrirse.
func (a Acta) Cifras() []CifraSituada {
	var out []CifraSituada
	for i, s := range a.Secciones {
		for j, r := range s.Repartos {
			for k, c := range r.Cifras {
				out = append(out, CifraSituada{
					Ref: ref(i, j, k), Fuente: s.Fuente, Reparto: r.Rotulo, Cifra: c,
				})
			}
		}
	}
	return out
}

// Parrafos devuelve toda la prosa del acta, cabecera incluida. Es lo que recorre
// la puerta de la regla 2.
func (a Acta) Parrafos() []Parrafo {
	out := append([]Parrafo(nil), a.Cabecera...)
	for _, s := range a.Secciones {
		out = append(out, s.Parrafos...)
	}
	return out
}

// validar es la puerta de las tres reglas, y corre SIEMPRE al componer.
func (a Acta) validar() error {
	for _, p := range a.Parrafos() {
		if err := p.validar(); err != nil {
			return err
		}
	}
	for _, s := range a.Secciones {
		if !s.Aportada && strings.TrimSpace(s.PorQueFalta) == "" {
			return fmt.Errorf("%w: la seccion %q no tiene fuente y no dice que hace falta para "+
				"tenerla", ErrSeccionVaciaSinMotivo, s.Fuente)
		}
		for _, r := range s.Repartos {
			for _, c := range r.Cifras {
				if c.exigeDescargo && strings.TrimSpace(c.Descargo) == "" {
					return fmt.Errorf("%w: %q / %q / cubo %q.\n"+
						"  Un cubo que cuenta lo que no consta va con su frase al lado, no en "+
						"una nota al pie: la primera pantalla que se la olvide acusa en falso",
						ErrAusenciaSinDescargo, s.Fuente, r.Rotulo, c.Cubo)
				}
			}
		}
		// Las dos que no reparten nada y por tanto no tienen de que descargar: la
		// de la direccion, que es prosa y no cubos, y cualquiera cuya fuente no
		// se aporto, que ya esta obligada a decir que hace falta. Exigirles la
		// frase seria pedirla donde no hay dato al que pegarla.
		if s.Fuente == DeLaDireccion || !s.Aportada {
			continue
		}
		if len(s.Descargos()) == 0 {
			return fmt.Errorf("%w: %q.\n"+
				"  Toda seccion que ensena pasado tiene un cubo de lo que no consta, y ese cubo "+
				"lleva su frase. Sin ella, un dato que falta se lee como un incumplimiento, y "+
				"acusar en falso es el unico error que este producto no puede cometer ni una vez",
				ErrSeccionSinDescargo, s.Fuente)
		}
	}
	return nil
}

func corto(s string) string {
	s = strings.TrimSpace(s)
	if len(s) <= 60 {
		return s
	}
	return s[:57] + "..."
}

// ordenarElementos deja las filas en orden estable por su identidad. Nunca por
// el orden en que se recorrio el mapa de origen: dos ejecuciones darian dos
// actas distintas del mismo dato, y un documento que no se reproduce no es
// evidencia de nada.
func ordenarElementos(es []Elemento) []Elemento {
	sort.SliceStable(es, func(i, j int) bool { return es[i].Clave < es[j].Clave })
	return es
}
