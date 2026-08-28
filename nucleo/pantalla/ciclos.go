package pantalla

// Los ciclos y las sentadas: cuantas veces hay que sentarse, no cuantas
// casillas hay.
//
// EL PROBLEMA DE PRODUCTO, que es D-15 y es EL problema de producto. Un CISO de
// doscientos empleados abre el calendario y ve 47 obligaciones solo del anexo
// del Reglamento de Ejecucion (UE) 2024/2690. El numero es correcto y es de la
// norma, no nuestro: sumarlo es justamente lo que nadie mas hace. Pero un
// numero grande sin nada al lado no ordena, espanta, y un calendario que
// espanta se cierra el segundo mes.
//
// LA RESPUESTA NO ES ESCONDER EL NUMERO, es ensenar el otro al lado:
// **las obligaciones no son ceremonias**. Veintiocho obligaciones anuales de
// nueve capitulos distintos no son veintiocho reuniones: son las veces que hay
// que sentarse, y eso se puede CALCULAR.
//
// LAS DOS AGRUPACIONES, y son distintas:
//
//	ciclo    por CADENCIA. Todo lo que se repite cada doce meses es la misma
//	         clase de trabajo, venza cuando venza. Es la respuesta a "¿cuantos
//	         ritmos distintos tengo?", que en el corpus de hoy son cinco.
//	sentada  por MES, dentro de un ciclo. Es la respuesta a "¿cuantas veces
//	         tengo que sentarme?", y sale de las fechas de verdad: dos
//	         obligaciones anuales cuyos relojes arrancaron en marzo y en octubre
//	         son dos sentadas, no una.
//
// LO QUE HACE QUE ESTO NO SEA UN CONSEJO IRRESPONSABLE. Juntar dos fechas
// significa ADELANTAR una, y adelantar no siempre se puede. Por eso cada fecha
// trae `origen_del_intervalo`, que se escribio ayer por otra razon y hoy hace
// el trabajo:
//
//	suelo_legal  la norma pone un MINIMO de frecuencia ("al menos una vez al
//	             ano"): apretar siempre cumple, asi que se puede adelantar
//	propuesto    el numero lo pone plazum: se puede mover, y ademas el campo
//	             `cuando_cambiarlo` dice bajo que supuestos
//	fijado       la norma da el numero EXACTO: no se toca
//
// Sin esa distincion, "junta estas doce en una sola sesion" seria proponerle al
// cliente que incumpla una de ellas. Con ella, el consejo es exacto y dice
// cuantas se pueden mover y cuantas no.
//
// LO QUE SE QUEDA FUERA, Y SE CUENTA. Un `plazo` o una `puntual` NO entran en
// ningun ciclo: un vencimiento unico no es una ceremonia que se repite, y
// meterlo en una sentada le diria al operador que puede adelantarlo para
// juntarlo con otra cosa, que es falso y ademas peligroso (la mayoria de los
// plazos unicos de este corpus son notificaciones de incidente). Se cuentan en
// `FechasSueltas`, porque en una derivacion que el usuario ve, lo que
// desaparece se cuenta.

import (
	"sort"
	"time"
)

// Los tres origenes, repetidos aqui como constantes de pantalla.
//
// NO SE IMPORTAN DE nucleo/corpus a proposito, aunque alli existen: este
// paquete ya importa corpus para derivar, asi que importarlas seria gratis. El
// motivo es otro y es de diseno: lo que este fichero necesita saber no es "que
// vocabulario tiene el corpus" sino "que puedo mover y que no", que es una
// pregunta de pantalla. Si algun dia el corpus anade un cuarto origen, la
// respuesta segura aqui es la restrictiva (no se puede mover), y eso se
// consigue comparando contra estas dos y no contra una lista abierta.
const (
	origenSueloLegal = "suelo_legal"
	origenPropuesto  = "propuesto"
)

// Sentada es todo lo que vence en un mismo mes dentro de un mismo ciclo: una
// sola vez que hay que sentarse.
type Sentada struct {
	Ano int
	Mes time.Month
	// Clave es la clave de catalogo del nombre del mes, igual que en Mes.
	Clave  string
	Fechas []Fecha
	// Marcos son los URN distintos que cubre esta sentada, ordenados. Es el
	// numero que hace que esto sea composicion entre marcos y no un resumen:
	// "una sentada, tres marcos" es lo que un catalogo de controles no sabe
	// decir.
	Marcos []string
}

// Ciclo agrupa por cadencia: un ritmo de trabajo, con las veces que hay que
// sentarse a ese ritmo.
type Ciclo struct {
	// Cadencia es el intervalo ISO-8601 declarado (P12M, P6M...).
	Cadencia string
	// Sentadas van ordenadas por fecha, la mas cercana primero.
	Sentadas []Sentada
	// Obligaciones distintas de este ciclo. NO es la suma de las fechas: una
	// obligacion trimestral produce cuatro fechas y sigue siendo una
	// obligacion, y contar fechas aqui inflaria justo el numero que este
	// fichero existe para desinflar.
	Obligaciones int
	// ConFecha son las que han producido al menos una fecha en la ventana.
	ConFecha int
	// EsperandoDato son las que no han producido fecha porque falta el hecho
	// del que arranca su ciclo ("ultima_auditoria_interna" y compania).
	//
	// SE CUENTAN AQUI Y NO SOLO EN SinFecha, y es la decision que hace util
	// esta seccion el dia uno: un cliente que no ha registrado nada tiene TODAS
	// sus obligaciones asi, y ese es exactamente el dia en que mas falta hace
	// saber cuantas veces al ano habra que sentarse. Un ciclo existe aunque su
	// primera fecha no se pueda calcular todavia.
	EsperandoDato int
	// Marcos distintos, ordenados.
	Marcos []string
	// Alineables son las obligaciones de este ciclo cuya fecha SE PUEDE
	// adelantar para juntarla con otra sentada, o sea las de suelo legal y las
	// que lleven un numero de plazum.
	Alineables int
	// Fijas son las que no se pueden mover porque la norma da el numero
	// exacto. Se cuentan aparte y NUNCA se meten en un consejo de agrupacion.
	Fijas int
}

// PuedeAdelantarse dice si una fecha se puede mover hacia atras para juntarla
// con otras.
//
// EL VALOR CERO ES EL RESTRICTIVO, y es deliberado (invariante 8): una fecha
// sin `origen_del_intervalo` declarado NO se puede adelantar. Hoy el linter
// exige el campo en toda periodica, asi que ese caso no deberia existir; pero
// si algun dia existiera, la respuesta segura es que no se toca. Proponer
// adelantar algo cuyo regimen no se conoce es proponer un incumplimiento.
func (f Fecha) PuedeAdelantarse() bool {
	return puedeAdelantarse(f.OrigenDelIntervalo)
}

// PuedeAdelantarse, lo mismo para un reloj que todavia no ha dado fecha.
func (s SinFecha) PuedeAdelantarse() bool {
	return puedeAdelantarse(s.OrigenDelIntervalo)
}

func puedeAdelantarse(origen string) bool {
	return origen == origenSueloLegal || origen == origenPropuesto
}

// agruparEnCiclos reparte en ciclos y sentadas, y devuelve ademas cuantas
// fechas se quedaron fuera por no ser periodicas.
//
// Entran las DOS listas: las fechas ya calculadas y los relojes que esperan un
// dato del operador. Un ciclo es una propiedad de la obligacion, no de su
// proxima fecha.
func agruparEnCiclos(fechas []Fecha, sin []SinFecha) ([]Ciclo, int) {
	sueltas := 0
	porCadencia := map[string][]Fecha{}
	for _, f := range fechas {
		if f.Cadencia == "" {
			sueltas++ // un plazo o una puntual: no es una ceremonia que se repite
			continue
		}
		porCadencia[f.Cadencia] = append(porCadencia[f.Cadencia], f)
	}
	// Los que esperan un dato, indexados por cadencia. Solo los que ESPERAN un
	// hecho: "sin plazo legal" y "sin ejecutor" son otra cosa y no tienen ciclo
	// que ofrecer.
	esperando := map[string][]SinFecha{}
	for _, s := range sin {
		if s.Cadencia == "" || s.Motivo != MotivoPendienteDeHecho {
			continue
		}
		esperando[s.Cadencia] = append(esperando[s.Cadencia], s)
	}

	cadencias := map[string]bool{}
	for c := range porCadencia {
		cadencias[c] = true
	}
	for c := range esperando {
		cadencias[c] = true
	}

	var out []Ciclo
	for cad := range cadencias {
		fs := porCadencia[cad]
		c := Ciclo{Cadencia: cad}

		// Las sentadas, por (ano, mes). Solo de lo que tiene fecha: una sentada
		// es una cita en el calendario y un reloj sin fecha no la tiene.
		type mesClave struct {
			a int
			m time.Month
		}
		porMes := map[mesClave][]Fecha{}
		for _, f := range fs {
			k := mesClave{f.Vence.Year(), f.Vence.Month()}
			porMes[k] = append(porMes[k], f)
		}
		for k, v := range porMes {
			ordenarFechas(v)
			c.Sentadas = append(c.Sentadas, Sentada{
				Ano: k.a, Mes: k.m, Clave: claveDelMes(k.m),
				Fechas: v, Marcos: marcosDe(v),
			})
		}
		sort.Slice(c.Sentadas, func(i, j int) bool {
			a, b := c.Sentadas[i], c.Sentadas[j]
			if a.Ano != b.Ano {
				return a.Ano < b.Ano
			}
			return a.Mes < b.Mes
		})

		// Las obligaciones DISTINTAS y su regimen, de las dos listas. Una
		// obligacion que aparece en las dos (un hito con fecha y otro
		// esperando) cuenta como CON FECHA: ya tiene cita.
		vistas := map[string]bool{}
		marcos := map[string]bool{}
		anota := func(id, marco, origen string, conFecha bool) {
			if vistas[id] {
				return
			}
			vistas[id] = true
			marcos[marco] = true
			c.Obligaciones++
			if conFecha {
				c.ConFecha++
			} else {
				c.EsperandoDato++
			}
			if puedeAdelantarse(origen) {
				c.Alineables++
			} else {
				c.Fijas++
			}
		}
		for _, f := range fs {
			anota(f.Obligacion, f.Marco, f.OrigenDelIntervalo, true)
		}
		for _, s := range esperando[cad] {
			anota(s.Obligacion, s.Marco, s.OrigenDelIntervalo, false)
		}
		c.Marcos = make([]string, 0, len(marcos))
		for m := range marcos {
			c.Marcos = append(c.Marcos, m)
		}
		sort.Strings(c.Marcos)
		out = append(out, c)
	}

	// Los ciclos, por cuantas SENTADAS producen en la ventana, de mas a menos:
	// es el orden en que le pesan a quien tiene que sentarse. A igual numero,
	// por cuantas obligaciones agrupan, y luego por cadencia, para que el orden
	// sea total y estable.
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if len(a.Sentadas) != len(b.Sentadas) {
			return len(a.Sentadas) > len(b.Sentadas)
		}
		if a.Obligaciones != b.Obligaciones {
			return a.Obligaciones > b.Obligaciones
		}
		return a.Cadencia < b.Cadencia
	})
	return out, sueltas
}

func marcosDe(fs []Fecha) []string {
	set := map[string]bool{}
	for _, f := range fs {
		set[f.Marco] = true
	}
	out := make([]string, 0, len(set))
	for m := range set {
		out = append(out, m)
	}
	sort.Strings(out)
	return out
}

// Sentadas es cuantas veces hay que sentarse en total dentro de la ventana.
func (c Calendario) Sentadas() int {
	n := 0
	for _, ci := range c.Ciclos {
		n += len(ci.Sentadas)
	}
	return n
}

// ObligacionesEnCiclo son las obligaciones distintas que entran en algun ciclo.
func (c Calendario) ObligacionesEnCiclo() int {
	n := 0
	for _, ci := range c.Ciclos {
		n += ci.Obligaciones
	}
	return n
}

// MarcosEnCiclo son los marcos distintos que aparecen en algun ciclo.
func (c Calendario) MarcosEnCiclo() int {
	set := map[string]bool{}
	for _, ci := range c.Ciclos {
		for _, m := range ci.Marcos {
			set[m] = true
		}
	}
	return len(set)
}
