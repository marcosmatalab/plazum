package pantalla

// EL CALENDARIO: los proximos doce meses del corpus instalado.
//
// Es una DERIVACION, no un motor. Todo lo que hay debajo existe desde hace
// semanas (el reloj legal en nucleo/ventana, la aplicabilidad en
// nucleo/aplicabilidad, el corpus en nucleo/corpus): esto es lo que faltaba para
// que un CISO vea sus fechas sin leer JSON.
//
// POR QUE VIVE EN nucleo/pantalla Y NO EN cmd/plazum. Porque `plazum calendario`
// y la pantalla Hoy de `plazum serve` tienen que decir LO MISMO, y la unica
// forma de garantizarlo es que salga del mismo sitio. Una derivacion metida en
// el comando queda fuera del alcance de los dorados byte a byte de este paquete
// y fuera del alcance de la superficie web, y entonces las dos versiones
// empiezan a separarse el primer dia.
//
// LAS TRES PROPIEDADES QUE SOSTIENE, y las tres tienen test:
//
//	determinista  dos ejecuciones con el mismo instante dan el mismo orden. El
//	              orden es total, con desempate por (obligacion, hito), porque
//	              recorrer un mapa de Go no tiene orden y quince fechas del mismo
//	              dia saldrian barajadas.
//	sin reloj     el instante entra como dato (invariante 1). Este fichero no
//	              llama a time.Now() ni una vez.
//	honesto       lo que no produce fecha NO desaparece: sale en SinFecha con el
//	              motivo. Una agenda que solo ensena lo que pudo calcular le hace
//	              creer al que la lee que lo demas no existe.

import (
	"sort"
	"strconv"
	"time"

	"github.com/marcosmatalab/plazum/nucleo/corpus"
	"github.com/marcosmatalab/plazum/nucleo/ventana"
)

// Fecha es un vencimiento listo para pintar, con de donde sale.
type Fecha struct {
	Vence time.Time
	// Marco es el URN del paquete. Es lo unico que hay: el formato de corpus
	// todavia no tiene nombre corto de marco, asi que en pantalla sale
	// "urn:es:rd:2022:311" y no "ENS". Esta anotado en docs/pendientes.md.
	Marco      string
	Obligacion string
	Titulo     string
	Articulo   string
	Cita       string
	Hito       string
	Estado     ventana.EstadoVenc
	// Regla es la derivacion completa del motor: de que hecho arranca, que
	// computo se aplico y que traslado. Es lo que convierte una fecha en una
	// fecha defendible.
	Regla string
	Aviso string
	// Divergencias son las otras lecturas del MISMO plazo, con su cita. El
	// motor no elige en silencio y el calendario tampoco las esconde.
	Divergencias []ventana.Divergencia
	// NoAntesDe es el suelo cuando la fecha final todavia no se sabe.
	NoAntesDe time.Time
	// Supuesta dice que esta obligacion se deriva de un hecho SUPUESTO (por
	// ejemplo el que pone un perfil de arranque) y no de uno declarado por la
	// organizacion. Un calendario que no distingue las dos cosas convierte una
	// conjetura en una obligacion, y eso se paga en la primera reunion.
	Supuesta bool
	// Cadencia es el intervalo declarado si la obligacion es `periodica`, y
	// vacio si no. Es lo que permite agrupar por CICLO: dos vencimientos con la
	// misma cadencia son la misma clase de trabajo repetido.
	Cadencia string
	// OrigenDelIntervalo dice DE QUIEN es ese numero (`suelo_legal`,
	// `propuesto` o `fijado`), y es lo que decide si una fecha SE PUEDE MOVER
	// para juntarla con otras. Con suelo legal solo se puede apretar, o sea
	// adelantar; con un numero de plazum, tambien; con un numero exacto de la
	// norma, no. Sin este campo, agrupar seria proponerle al cliente que
	// incumpla.
	OrigenDelIntervalo string
}

// SinFecha es un reloj que existe y no ha producido fecha, con el motivo. No es
// un hueco: es informacion.
type SinFecha struct {
	Marco      string
	Obligacion string
	Titulo     string
	Articulo   string
	Hito       string
	// Motivo es una clave de catalogo, nunca texto: quien pinta esto decide el
	// idioma. Ver ClavesDelCalendario.
	Motivo string
	Regla  string
	// Cadencia y OrigenDelIntervalo viajan igual que en Fecha, y aqui son mas
	// importantes que alli: una obligacion periodica que espera un dato del
	// operador NO TIENE FECHA todavia y sigue teniendo CICLO. El dia uno de un
	// cliente, todas sus obligaciones estan asi, y es justo el dia en el que
	// mas falta hace saber cuantas veces al ano habra que sentarse.
	Cadencia           string
	OrigenDelIntervalo string
}

// Estreno es una obligacion que TODAVIA no obliga y que empezara a obligar
// DENTRO de la ventana que se esta mirando.
//
// EL AGUJERO QUE CIERRA. La derivacion hacia `if !vigente { continue }`, un
// continue mudo, y con el se iba entera del calendario cualquier obligacion
// cuya vigencia empieza manana. Esa fila no es ruido: para un calendario de
// cumplimiento es LA noticia. Medido con el corpus de hoy: el perfil de
// fabricante de software no ensenaba NI UNA de las dos notificaciones del art.
// 14 del CRA, que empiezan a aplicarse el 11-09-2026, quince dias despues del
// dia en que se midio. El producto entero existe para decir esa fecha y era
// justo la que se callaba.
//
// NO ES UNA `Fecha` y va en su propia lista a proposito. Una Fecha es un
// VENCIMIENTO: algo que tienes que haber hecho antes de esa hora. Un estreno es
// lo contrario, el dia en que empieza la cuenta. Meterlos en la misma lista le
// diria al operador "entrega esto el 11-09-2026", que es falso y ademas
// alarmante.
type Estreno struct {
	// Desde es el instante en que la obligacion empieza a obligar, ya cruzado
	// con la vigencia de su norma.
	Desde      time.Time
	Marco      string
	Obligacion string
	Titulo     string
	Articulo   string
	Cita       string
	// Supuesta viaja igual que en Fecha: un estreno derivado de un hecho de
	// perfil es una conjetura sobre una fecha real, y las dos mitades importan.
	Supuesta bool
	// Hitos son los que trae la obligacion. Va en el dato y no lo recalcula
	// quien pinta: la cuenta de abajo habla en hitos, asi que una cabecera que
	// contara filas diria un numero distinto del de la cuenta para lo mismo,
	// que es exactamente el fallo que esta unificacion viene a cerrar.
	Hitos int
}

// Cese es una obligacion que HOY te obliga y que dejara de obligarte dentro de
// la ventana. Es el espejo exacto de Estreno, y existe por la misma razon.
//
// POR QUE ES NOTICIA, Y BUENA. Un producto de cumplimiento que solo sabe sumar
// obligaciones es un producto en el que el trabajo solo crece, y el operador
// aprende deprisa que la herramienta nunca le quita nada de encima. Decir "esto
// deja de obligarte el 15 de marzo y puedes parar de hacerlo" es la mitad del
// trabajo que nadie hace, y es la que se gana la confianza: quien te avisa de
// lo que ya no debes es quien te esta leyendo la norma de verdad y no
// acumulando controles.
//
// NO ES UNA `Fecha`, por lo mismo que un Estreno no lo es. La fecha de un cese
// no es un vencimiento: no hay nada que entregar ese dia. Mezclarlos pondria en
// la agenda del operador un plazo que no existe.
//
// LO QUE SE CUENTA ES EL CESE DENTRO DE LA VENTANA, no la obligacion ya
// derogada hace tres anos. Una que dejo de obligar ANTES de hoy no es una
// transicion de esta ventana y no se pinta; se cuenta, que es distinto, y esa
// cuenta esta en HitosYaCesados.
type Cese struct {
	// Hasta es el ultimo instante en que la obligacion obliga.
	Hasta      time.Time
	Marco      string
	Obligacion string
	Titulo     string
	Articulo   string
	Cita       string
	Supuesta   bool
	Hitos      int
}

// Mes agrupa las fechas de un mes natural.
type Mes struct {
	Ano int
	Mes time.Month
	// Clave es la clave de catalogo del nombre del mes (ui.mes.1 a ui.mes.12).
	// El nombre no se cablea aqui porque este paquete no sabe en que idioma se
	// va a pintar, y "September" en una agenda en espanol es un fallo que se ve
	// desde la otra punta de la sala.
	Clave  string
	Fechas []Fecha
}

// Calendario es lo que se pinta, con su propia contabilidad.
type Calendario struct {
	Desde time.Time
	Hasta time.Time
	Meses []Mes
	// SinFecha son los relojes que no dieron fecha, ordenados igual.
	SinFecha []SinFecha
	// Ciclos son las mismas fechas agrupadas por CADENCIA y, dentro de cada
	// cadencia, por mes: cuantos ritmos distintos hay y cuantas veces hay que
	// sentarse a cada uno. Es la respuesta a D-15 y vive en ciclos.go.
	Ciclos []Ciclo
	// FechasSueltas son las que no entran en ningun ciclo por no ser
	// periodicas (un plazo, una puntual). No se agrupan a proposito y se
	// cuentan a proposito: en una derivacion que el usuario ve, lo que
	// desaparece se cuenta.
	FechasSueltas int
	// Estrenos son las obligaciones que empiezan a obligar dentro de la
	// ventana. Ordenadas por fecha de estreno, la mas cercana primero.
	Estrenos []Estreno
	// Ceses son las que dejan de obligar dentro de la ventana. Ordenadas igual.
	Ceses []Cese

	// La contabilidad honesta. Los cuatro numeros se ensenan juntos porque cada
	// uno solo miente si se lee sin los otros tres.
	HitosDelCorpus   int // cuantos hitos de reloj hay instalados
	HitosEnVigor     int // cuantos estaban en vigor en el instante de calculo
	HitosAplicables  int // cuantos derivo la aplicabilidad
	FueraDeLaVentana int // FECHAS calculadas que caen fuera de los doce meses
	// HitosQueEstrenan son los que todavia no obligan y empezaran dentro de
	// la ventana. Se cuenta aparte porque NO esta dentro de HitosEnVigor: en
	// el instante del calculo no estaban en vigor, y sumarlos ahi haria que la
	// contabilidad dejara de cuadrar con lo que dice su propio nombre.
	HitosQueEstrenan int
	// HitosQueEstrenanYTeAlcanzan son los que ademas de estrenar dentro de la
	// ventana te alcanzan, o sea los que salen en la lista Estrenos. Se cuenta
	// aparte de HitosQueEstrenan para que la particion por TIEMPO siga siendo
	// exhaustiva (esa es de hechos del calendario) sin mentir sobre cuantos se
	// estan ensenando (esa es una respuesta sobre ti).
	HitosQueEstrenanYTeAlcanzan int
	// HitosQueCesan hoy obligan y dejaran de obligar dentro de la ventana.
	// Estos SI estan dentro de HitosEnVigor, porque hoy lo estan: es la
	// diferencia con los estrenos, y por eso se dice aqui.
	HitosQueCesan int

	// LOS TRES CUBOS DE LO QUE SE DESCARTA, para que no quede ni un descarte
	// mudo en esta derivacion.
	//
	// El barrido que los trajo (28-08-2026) salio de un fallo: `if !vigente {
	// continue }` se llevaba del calendario, sin decir nada, cualquier
	// obligacion que empieza a obligar manana. La regla que queda escrita es
	// que en una derivacion que el usuario ve, un elemento solo desaparece si
	// desaparecer es la respuesta, y entonces SE CUENTA. Estos tres son lo que
	// quedaba por contar.

	// HitosNoAlcanzados estan en vigor y la aplicabilidad dice que no te
	// alcanzan. No se enumeran a proposito: con el corpus instalado serian casi
	// todos, y una lista de trescientas obligaciones que no son tuyas no
	// informa, entierra. Pero el numero se dice, porque callarlo deja al
	// operador sin saber si el producto miro el corpus entero o solo un trozo.
	// La puerta para verlos existe y es --todos-los-relojes.
	HitosNoAlcanzados int
	// HitosYaCesados dejaron de obligar ANTES de la ventana. No son noticia de
	// estos doce meses (una obligacion derogada en 2023 no es una transicion de
	// 2026) y por eso no se pintan, pero se cuentan: es la diferencia entre "no
	// te aplica" y "el producto no lo ha mirado".
	HitosYaCesados int
	// HitosQueEmpiezanDespues empiezan a obligar MAS ALLA de la ventana. Mismo
	// trato y misma razon, por el otro extremo del tiempo.
	HitosQueEmpiezanDespues int
	// HitosConVigenciaIlegible no se pudieron situar en el tiempo porque su
	// vigencia no se lee. Ya salen en SinFecha con su motivo; se cuentan ademas
	// para que la particion por tiempo sea exhaustiva y la conservacion se
	// pueda comprobar sumando.
	HitosConVigenciaIlegible int
}

// motivos de SinFecha, como claves de catalogo.
const (
	MotivoPendienteDeHecho = "ui.calendario.pendiente"
	MotivoSinPlazoLegal    = "ui.calendario.sin_plazo_legal"
	MotivoSinEjecutor      = "ui.calendario.sin_ejecutor"
)

// ClavesDelCalendario son TODAS las claves de catalogo que el calendario puede
// emitir. El inventario del catalogo las lee de aqui: una clave que se emite y
// no esta traducida sale como el identificador en bruto en la pantalla de un
// cliente.
// Las doce van escritas UNA A UNA y no en un bucle, y no es por gusto: el
// inventario del catalogo (adaptadores/catalogo) busca claves LITERALES en el
// fuente, asi que una clave construida con strconv.Itoa no la ve nadie y se
// queda sin traducir hasta que un cliente la vea en crudo en su pantalla. Ya
// paso con "ui.mes.3" y "ui.mes.5", que es lo que uno espera de un bucle: falla
// en las que nadie escribio a mano en un test.
func ClavesDelCalendario() []string {
	return []string{
		MotivoPendienteDeHecho, MotivoSinPlazoLegal, MotivoSinEjecutor,
		"ui.mes.1", "ui.mes.2", "ui.mes.3", "ui.mes.4", "ui.mes.5", "ui.mes.6",
		"ui.mes.7", "ui.mes.8", "ui.mes.9", "ui.mes.10", "ui.mes.11", "ui.mes.12",
	}
}

// claveDelMes devuelve la clave de catalogo de un mes. Los valores que devuelve
// estan todos en ClavesDelCalendario, y el test lo comprueba en las dos
// direcciones.
func claveDelMes(m time.Month) string { return "ui.mes." + strconv.Itoa(int(m)) }

// Aplicable dice si una obligacion le alcanza al sujeto, y si eso se deriva de
// un hecho supuesto. La firma es una funcion y no el motor de aplicabilidad
// entero por dos motivos: este paquete no tiene por que saber de Datalog, y asi
// el calendario se puede probar sin montar un programa.
//
// EL VALOR CERO ES EL RESTRICTIVO, y es el invariante 8 en una frontera nueva:
// una funcion nil significa que NO SE HA FILTRADO NADA, y eso hay que decirlo en
// voz alta en vez de dejar que se lea como "todo aplica". Por eso Derivar12Meses
// exige la funcion y quien no quiera filtrar pasa TodoAplica, que se llama asi
// para que se lea en el sitio de la llamada.
type Aplicable func(idObligacion string) (aplica bool, supuesta bool)

// TodoAplica no filtra nada. Se escribe en el sitio de la llamada a proposito:
// un calendario sin filtrar le ensena a un banco las obligaciones de un
// fabricante de productos sanitarios, y eso hay que pedirlo, no heredarlo.
func TodoAplica(string) (bool, bool) { return true, false }

// Derivar12Meses es la derivacion entera. `ahora` entra como dato: este paquete
// no llama a time.Now() (invariante 1, vigilado por arquitectura_test.go).
func Derivar12Meses(ps []*corpus.Paquete, aplica Aplicable, hechos ventana.Hechos,
	ahora time.Time) Calendario {

	if aplica == nil {
		// No se adivina. Una funcion nil es un error de quien llama, y
		// tratarla como "todo aplica" seria el valor cero permisivo que el
		// invariante 8 prohibe.
		aplica = func(string) (bool, bool) { return false, false }
	}
	hasta := ahora.AddDate(1, 0, 0)
	cal := Calendario{Desde: ahora, Hasta: hasta}

	var fechas []Fecha
	var sin []SinFecha
	var estrenos []Estreno
	var ceses []Cese

	for _, p := range ps {
		for _, o := range p.Obligaciones {
			if o.Temporalidad == nil {
				continue
			}
			cal.HitosDelCorpus += hitosDeclarados(o)

			// La vigencia primero: un reloj de una obligacion que todavia no
			// obliga (o que ya no) no es una fecha del calendario de nadie. El
			// error de vigencia se trata como "no en vigor", que es lo
			// restrictivo, y no se traga: sale como SinFecha.
			vigente, err := p.EnVigor(o, ahora)
			if err != nil {
				cal.HitosConVigenciaIlegible += hitosDeclarados(o)
				sin = append(sin, SinFecha{Marco: p.URN, Obligacion: o.ID,
					Titulo: o.TituloLegible(), Articulo: o.Articulo,
					Motivo: MotivoSinEjecutor, Regla: "vigencia ilegible: " + err.Error()})
				continue
			}
			if !vigente {
				// EL CONTINUE QUE ERA MUDO. Antes de irse, se mira si la
				// obligacion empieza a obligar DENTRO de la ventana: si es
				// asi, es una fila del calendario, no un silencio.
				//
				// Se pide la aplicabilidad igual que a las demas: un estreno de
				// algo que no te alcanza no es noticia tuya. Y si la vigencia
				// no se puede leer, no se inventa un estreno: el caso ilegible
				// ya salio por SinFecha unas lineas mas arriba.
				desde, err := p.InicioDeVigencia(o)
				if err != nil {
					continue // el caso ilegible ya salio por SinFecha arriba
				}
				if !desde.After(ahora) {
					// No esta en vigor y su vigencia empezo en el pasado: dejo
					// de obligar antes de esta ventana. No es una transicion de
					// estos doce meses, asi que no se pinta; se CUENTA, que es
					// lo que la distingue de un descarte mudo.
					cal.HitosYaCesados += hitosDeclarados(o)
					continue
				}
				if !desde.Before(hasta) {
					cal.HitosQueEmpiezanDespues += hitosDeclarados(o)
					continue
				}
				// HitosQueEstrenan cuenta TODO lo que empieza dentro de la
				// ventana, alcance aparte, porque es un hecho del calendario y
				// no una respuesta sobre ti: asi la particion por tiempo de la
				// contabilidad es exhaustiva y se puede comprobar sumando (ver
				// el test de la conservacion). La LISTA, en cambio, solo trae lo
				// que te alcanza, porque el estreno de algo que no es tuyo no
				// es noticia tuya. Que los dos numeros puedan diferir se dice
				// en la salida en vez de esconderlo.
				cal.HitosQueEstrenan += hitosDeclarados(o)
				if ok, supuesta := aplica(o.ID); ok {
					cal.HitosQueEstrenanYTeAlcanzan += hitosDeclarados(o)
					estrenos = append(estrenos, Estreno{
						Desde: desde, Marco: p.URN, Obligacion: o.ID,
						Titulo: o.TituloLegible(), Articulo: o.Articulo,
						Cita: o.Cita, Supuesta: supuesta, Hitos: hitosDeclarados(o),
					})
				}
				continue
			}
			cal.HitosEnVigor += hitosDeclarados(o)

			ok, supuesta := aplica(o.ID)
			if !ok {
				// EL DESCARTE MAS GRANDE DE LA DERIVACION, y el que estuvo mudo
				// hasta hoy. No se enumera (serian casi todas) y no se calla
				// (callarlo deja al operador sin saber si el producto ha mirado
				// el corpus entero): se cuenta, y la puerta para verlos es
				// --todos-los-relojes.
				cal.HitosNoAlcanzados += hitosDeclarados(o)
				continue
			}
			cal.HitosAplicables += hitosDeclarados(o)

			// EL CESE, espejo del estreno: hoy te obliga y dejara de hacerlo
			// dentro de la ventana. Va DESPUES de la aplicabilidad por la misma
			// razon que el estreno: el cese de algo que no te alcanza no es
			// noticia tuya. Y va antes de calcular vencimientos porque no
			// depende de ellos: una obligacion puede cesar sin llegar a vencer
			// ni una vez en la ventana, y eso sigue siendo noticia.
			if fin, hayFin, err := p.FinDeVigencia(o); err == nil && hayFin &&
				fin.After(ahora) && fin.Before(hasta) {
				cal.HitosQueCesan += hitosDeclarados(o)
				ceses = append(ceses, Cese{
					Hasta: fin, Marco: p.URN, Obligacion: o.ID,
					Titulo: o.TituloLegible(), Articulo: o.Articulo,
					Cita: o.Cita, Supuesta: supuesta, Hitos: hitosDeclarados(o),
				})
			}

			vs, err := corpus.VencimientosDe(o, hechos, hasta)
			if err != nil {
				// Una primitiva sin ejecutor es una fila, no un silencio. Hoy
				// solo `puntual`, `periodica` y `plazo` tienen ejecutor.
				sin = append(sin, SinFecha{Marco: p.URN, Obligacion: o.ID,
					Titulo: o.TituloLegible(), Articulo: o.Articulo,
					Motivo: MotivoSinEjecutor, Regla: err.Error()})
				continue
			}
			for _, v := range vs {
				switch v.Estado {
				case ventana.Determinado:
					if v.Vence.Before(ahora) || !v.Vence.Before(hasta) {
						cal.FueraDeLaVentana++
						continue
					}
					f := Fecha{
						Vence: v.Vence, Marco: p.URN, Obligacion: o.ID,
						Titulo: o.TituloLegible(), Articulo: o.Articulo, Cita: o.Cita,
						Hito: v.Hito, Estado: v.Estado, Regla: v.Regla, Aviso: v.Aviso,
						Divergencias: v.Divergencias, NoAntesDe: v.NoAntesDe,
						Supuesta: supuesta,
					}
					if o.Temporalidad.Primitiva == "periodica" {
						f.Cadencia = o.Temporalidad.Cadencia
						f.OrigenDelIntervalo = o.Temporalidad.OrigenDelIntervalo
					}
					fechas = append(fechas, f)
				case ventana.PendienteDeHecho:
					sf := SinFecha{Marco: p.URN, Obligacion: o.ID,
						Titulo: o.TituloLegible(), Articulo: o.Articulo, Hito: v.Hito,
						Motivo: MotivoPendienteDeHecho, Regla: v.Regla}
					if o.Temporalidad.Primitiva == "periodica" {
						sf.Cadencia = o.Temporalidad.Cadencia
						sf.OrigenDelIntervalo = o.Temporalidad.OrigenDelIntervalo
					}
					sin = append(sin, sf)
				case ventana.SinPlazoLegal:
					sin = append(sin, SinFecha{Marco: p.URN, Obligacion: o.ID,
						Titulo: o.TituloLegible(), Articulo: o.Articulo, Hito: v.Hito,
						Motivo: MotivoSinPlazoLegal, Regla: v.Regla})
				}
			}
		}
	}

	ordenarFechas(fechas)
	ordenarSinFecha(sin)
	// Por fecha de estreno, la mas cercana primero; a igual fecha, por marco y
	// obligacion, que son identidades del dato y dan un orden estable. Sin el
	// desempate, dos estrenos del mismo dia saldrian en el orden en que el
	// corpus se cargue, que no es un orden.
	sort.Slice(estrenos, func(i, j int) bool {
		a, b := estrenos[i], estrenos[j]
		if !a.Desde.Equal(b.Desde) {
			return a.Desde.Before(b.Desde)
		}
		if a.Marco != b.Marco {
			return a.Marco < b.Marco
		}
		return a.Obligacion < b.Obligacion
	})
	cal.Estrenos = estrenos
	// Mismo desempate que los estrenos, y por lo mismo: sin el, dos ceses del
	// mismo dia salen en el orden en que se recorra el corpus, que no es orden.
	sort.Slice(ceses, func(i, j int) bool {
		a, b := ceses[i], ceses[j]
		if !a.Hasta.Equal(b.Hasta) {
			return a.Hasta.Before(b.Hasta)
		}
		if a.Marco != b.Marco {
			return a.Marco < b.Marco
		}
		return a.Obligacion < b.Obligacion
	})
	cal.Ceses = ceses
	cal.SinFecha = sin
	cal.Meses = agrupar(fechas)
	cal.Ciclos, cal.FechasSueltas = agruparEnCiclos(fechas, sin)
	return cal
}

// ordenarFechas fija un orden TOTAL. El desempate por (obligacion, hito) no es
// celo: quince relojes anuales arrancados el mismo dia vencen en el mismo
// instante, y sin desempate salen en un orden distinto en cada ejecucion porque
// el recorrido de los paquetes viene de un mapa. Un dorado byte a byte lo caza
// una vez de cada tres, que es la peor forma de rojo.
func ordenarFechas(f []Fecha) {
	sort.SliceStable(f, func(i, j int) bool {
		if !f[i].Vence.Equal(f[j].Vence) {
			return f[i].Vence.Before(f[j].Vence)
		}
		if f[i].Obligacion != f[j].Obligacion {
			return f[i].Obligacion < f[j].Obligacion
		}
		return f[i].Hito < f[j].Hito
	})
}

func ordenarSinFecha(s []SinFecha) {
	sort.SliceStable(s, func(i, j int) bool {
		if s[i].Obligacion != s[j].Obligacion {
			return s[i].Obligacion < s[j].Obligacion
		}
		return s[i].Hito < s[j].Hito
	})
}

// agrupar parte las fechas en meses naturales. Se agrupa en la zona en la que
// viene el instante, que es la del regimen que lo calculo: agrupar en UTC un
// vencimiento de las 23:59:59 en Madrid lo manda al mes siguiente uno de cada
// dos meses.
func agrupar(f []Fecha) []Mes {
	var out []Mes
	for _, x := range f {
		a, m, _ := x.Vence.Date()
		if n := len(out); n > 0 && out[n-1].Ano == a && out[n-1].Mes == m {
			out[n-1].Fechas = append(out[n-1].Fechas, x)
			continue
		}
		out = append(out, Mes{Ano: a, Mes: m, Clave: claveDelMes(m),
			Fechas: []Fecha{x}})
	}
	return out
}

// Total es cuantas fechas hay en la ventana. Se calcula y no se guarda para que
// no pueda quedarse vieja.
func (c Calendario) Total() int {
	n := 0
	for _, m := range c.Meses {
		n += len(m.Fechas)
	}
	return n
}

// hitosDeclarados es cuantos hitos declara la temporalidad de una obligacion.
//
// LA UNIDAD QUE VE EL USUARIO ES EL HITO, y esta funcion es donde se decide.
// Antes la contabilidad contaba OBLIGACIONES y las llamaba "relojes", mientras
// el corpus se describia a si mismo en hitos: dos numeros distintos para lo
// mismo en dos pantallas, que es una pregunta de soporte autoinfligida. Gana el
// hito porque es lo que produce fechas: una obligacion con tres hitos
// escalonados (alerta, notificacion, informe final) da tres fechas y contarla
// como "un reloj" esconde dos tercios del trabajo que le espera al operador.
//
// Sin `hitos`, la temporalidad declara UNO en el campo `hito`, asi que el suelo
// es uno y nunca cero: una obligacion con temporalidad y cero hitos no existe.
func hitosDeclarados(o corpus.Obligacion) int {
	if o.Temporalidad == nil {
		return 0
	}
	if n := len(o.Temporalidad.Hitos); n > 0 {
		return n
	}
	return 1
}
