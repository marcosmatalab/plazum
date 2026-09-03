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
	"errors"
	"sort"
	"strconv"
	"time"

	"github.com/marcosmatalab/plazum/nucleo/corpus"
	"github.com/marcosmatalab/plazum/nucleo/metrica"
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
	// NombresDeHitos son esos mismos hitos POR NOMBRE, y es lo que permite que
	// quien pinta saque UNA FILA POR HITO.
	//
	// POR QUE NO BASTABA CON EL NUMERO, y estuvo a punto de costar un descuadre
	// de los que D11-c existe para impedir: la cifra «N que dejan de obligar
	// dentro de la ventana» cuenta HITOS, y la seccion que la abre pintaba una
	// fila por OBLIGACION. Con el corpus de hoy los dos numeros coinciden
	// porque ninguna obligacion que cesa tiene mas de un hito, o sea que el
	// fallo estaba escondido detras de un dato afortunado. Con el nombre, la
	// seccion pinta lo que la cifra cuenta y el contraste es contar filas.
	NombresDeHitos []string
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
	// NombresDeHitos, por lo mismo que en Estreno: la cifra `HitosQueCesan`
	// cuenta hitos y su seccion tiene que pintar hitos.
	NombresDeHitos []string
}

// Mes agrupa las fechas de un mes natural.
// Vencida es una obligacion cuyo plazo YA PASO y sigue sin cumplirse.
//
// EL AGUJERO QUE CIERRA, y es hermano del que cerro Estreno. La derivacion
// hacia `if v.Vence.Before(ahora) || !v.Vence.Before(hasta) { FueraDeLaVentana++
// ; continue }`: un vencimiento pasado y uno posterior a la ventana caian en el
// MISMO cubo, y ese cubo se imprimia con la etiqueta «fechas mas alla de los
// doce meses». O sea que un incumplimiento de hace cuatro anos salia contado
// como si fuera algo que pasara en el futuro lejano, y sin una sola fila.
//
// Medido el 29-08-2026: una obligacion anual cuya ultima ejecucion consta en
// 2022-01-15, mirada el 2026-08-27, daba «0 fechas en los proximos doce meses»
// y «4 fechas mas alla de los doce meses». Las cuatro estaban DETRAS, no
// delante, y eran el unico dato que ese calendario tenia que dar.
//
// Para un producto de continuidad de cumplimiento, «llevas cuatro ciclos sin
// hacer esto» es la fila mas importante que puede imprimir. Iba sin etiqueta y
// con la etiqueta cambiada.
//
// SE ENSENA UNA FILA POR OBLIGACION, no una por ocurrencia. Cuatro anos de
// incumplimiento anual son cuatro vencimientos y UNA noticia; imprimir los
// cuatro convierte la seccion en un muro. Se da el MAS ANTIGUO, que es el que
// dice desde cuando se incumple, y el numero de ciclos.
type Vencida struct {
	// Desde es el vencimiento MAS ANTIGUO que sigue sin cumplirse. Es el que
	// contesta "¿desde cuando?", que es la pregunta de un inspector.
	Desde time.Time
	// Ciclos son cuantos vencimientos de esta obligacion han pasado ya.
	Ciclos     int
	Marco      string
	Obligacion string
	Titulo     string
	Articulo   string
	Cita       string
	Hito       string
	Regla      string
	Supuesta   bool
}

// RelojDescartado es una obligacion ENTERA que se cae de la ventana, con sus
// hitos al lado.
//
// POR QUE EXISTE, Y ES LA PUERTA D11-c APLICADA AL CALENDARIO. Diez de las
// catorce cifras del pie no se podian abrir, y el motivo declarado no era
// pereza: esta derivacion guardaba esos descartes como CONTADORES DE HITOS y lo
// unico que retenia por elemento era `Destinos`, que va por obligacion. Abrir
// una cifra en hitos contra una lista en obligaciones da una lista que NO CUADRA
// con el numero que la abre, y un numero que no cuadra con su lista hace que se
// deje de leer la pantalla entera, con razon.
//
// Asi que se retiene la lista EN LA MISMA UNIDAD QUE EL CONTADOR: cada fila trae
// sus `Hitos`, y la suma de los hitos de la lista es exactamente el contador. Lo
// comprueba `Calendario.Cuadra`.
type RelojDescartado struct {
	Marco      string
	Obligacion string
	Titulo     string
	Articulo   string
	// Hitos son los NOMBRES de los hitos que declara la temporalidad, no su
	// numero. Es lo que hace que la lista cuadre con un contador que va en
	// hitos: quien pinte esto saca una fila por hito y cuenta exactamente lo
	// que cuenta la cifra que la abre.
	//
	// Con un `int` la pantalla habria tenido que ensenar una fila por
	// OBLIGACION bajo una cabecera que cuenta HITOS, y una obligacion con tres
	// hitos escalonados habria dejado la lista tres cortas sin que nada se
	// pusiera rojo. Es el mismo descuadre que la cifra huerfana, con una capa
	// de pintura encima.
	Hitos []string
	// Regla dice CON QUE DATO se decidio, no solo que se decidio: la fecha de
	// vigencia que lo deja fuera, o el error de lectura. No es clave de
	// catalogo, es derivacion, y viaja tal cual como la de una fecha.
	Regla string
}

// VencimientoDescartado es UNA ocurrencia que se cae, con su fecha.
//
// Va por OCURRENCIA y no por obligacion, y esa es toda la diferencia con
// RelojDescartado: los dos contadores que abre (`MasAllaDeLaVentana` y
// `VencimientosAntesDeLaVigencia`) cuentan vencimientos, no hitos, porque una
// periodica multiplica. Una lista por obligacion no cuadraria con ellos.
type VencimientoDescartado struct {
	Vence      time.Time
	Marco      string
	Obligacion string
	Titulo     string
	Articulo   string
	Hito       string
	Regla      string
}

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
	// Destinos dice donde acaba CADA obligacion con reloj: una etiqueta y solo
	// una. Es la ley de conservacion de extremo a extremo, la que cubre la
	// junta que las dos particiones numericas dejaban sin vigilar. Ver
	// destinos.go.
	Destinos map[string]Destino
	// Vencidas son las obligaciones cuyo plazo ya paso. Ordenadas por
	// antiguedad, la que lleva mas tiempo incumplida primero: es el orden en
	// que le pesan a quien tiene que responder por ellas.
	Vencidas []Vencida
	// Estrenos son las obligaciones que empiezan a obligar dentro de la
	// ventana. Ordenadas por fecha de estreno, la mas cercana primero.
	Estrenos []Estreno
	// Ceses son las que dejan de obligar dentro de la ventana. Ordenadas igual.
	Ceses []Cese

	// La contabilidad honesta. Los cuatro numeros se ensenan juntos porque cada
	// uno solo miente si se lee sin los otros tres.
	HitosDelCorpus  int // cuantos hitos de reloj hay instalados
	HitosEnVigor    int // cuantos estaban en vigor en el instante de calculo
	HitosAplicables int // cuantos derivo la aplicabilidad
	// MasAllaDeLaVentana son FECHAS calculadas posteriores a la ventana. Antes
	// se llamaba FueraDeLaVentana y metia en el mismo saco las pasadas, que no
	// estan fuera por delante sino por detras y son noticia. Ver Vencida.
	MasAllaDeLaVentana int
	// VencimientosPasados son las ocurrencias ya vencidas, contando TODAS: la
	// lista Vencidas trae una fila por obligacion y este numero trae las
	// ocurrencias, que con una anual de cuatro anos son cuatro.
	VencimientosPasados int
	// VencimientosAntesDeLaVigencia son ocurrencias que el ciclo calcula ANTES
	// de que la obligacion entrara en vigor.
	//
	// NO SON INCUMPLIMIENTOS Y NO PUEDEN SALIR EN LA LISTA. El ancla de una
	// cadencia es un hecho del operador ("la ultima vez que lo hice"), y ese
	// hecho puede ser muy anterior a la norma: quien reviso su politica en
	// 2022 no incumplia el Reglamento de Ejecucion (UE) 2024/2690 en 2023,
	// porque en 2023 no le obligaba. Aparecio en la primera ejecucion de la
	// seccion de vencidos, con un «vencio el 2023-01-15» de una norma en vigor
	// desde el 2024-11-07.
	//
	// Se cuentan en vez de descartarse en silencio, por la regla de siempre.
	VencimientosAntesDeLaVigencia int
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

	// LAS LISTAS DE LOS DESCARTES, en la MISMA unidad que su contador.
	//
	// Antes solo estaban los numeros de arriba, y por eso diez de las catorce
	// cifras del pie del calendario no se podian abrir: enlazar un numero en
	// hitos a una lista en obligaciones manda a una lista que no cuadra con el
	// numero que la abre, que es peor que no tener enlace.
	//
	// LO QUE NO SE ENUMERA, Y NO ES UN OLVIDO: `HitosNoAlcanzados` sigue siendo
	// solo un numero. Es D-13 y esta decidido: con el corpus instalado serian
	// casi todos, y una lista de trescientas obligaciones que no son tuyas no
	// informa, entierra. Su puerta es `--todos-los-relojes`. Lo mismo vale para
	// los tres totales de la particion (instalados, en vigor, alcanzados), que
	// son el corpus entero mirado de tres formas.

	// RelojesAlcanzados estan en vigor y la aplicabilidad dice que te alcanzan.
	// La suma de sus Hitos es HitosAplicables.
	//
	// NO ES UN DESCARTE Y POR ESO IMPORTA MAS QUE LOS OTROS: es la unica de
	// estas listas que enumera lo que SI es tuyo. Se retiene por la misma razon
	// que las demas (una cifra que no se puede abrir hay que creersela) y
	// ademas porque D-13 no la alcanza: D-13 dice que no se enumere lo que NO
	// te alcanza, porque serian centenares que no son tuyos. Estos son los
	// tuyos, y son entre 17 y 73 en los tres perfiles publicados.
	RelojesAlcanzados []RelojDescartado
	// RelojesQueEstrenan empiezan a obligar DENTRO de la ventana, te alcancen o
	// no. La suma de sus Hitos es HitosQueEstrenan.
	//
	// POR QUE NO SIRVE LA LISTA `Estrenos` PARA ABRIR ESA CIFRA, y es el
	// descuadre que esta lista viene a cerrar: `Estrenos` solo trae lo que
	// ademas te alcanza (su cardinal es HitosQueEstrenanYTeAlcanzan) y ademas
	// trae una fila por OBLIGACION. Medido el 04-09-2026 sobre el perfil de
	// fabricante de software: la cifra dice 9 y esa lista tiene 4 filas.
	// Enlazar la cifra ahi mandaria a una lista mas corta que su numero, que es
	// el error que D11-c existe para impedir.
	RelojesQueEstrenan []RelojDescartado
	// RelojesYaCesados dejaron de obligar antes de la ventana. La suma de sus
	// Hitos es HitosYaCesados.
	RelojesYaCesados []RelojDescartado
	// RelojesQueEmpiezanDespues empiezan a obligar mas alla de la ventana. La
	// suma de sus Hitos es HitosQueEmpiezanDespues.
	RelojesQueEmpiezanDespues []RelojDescartado
	// RelojesConVigenciaIlegible no se pueden situar en el tiempo. La suma de
	// sus Hitos es HitosConVigenciaIlegible.
	RelojesConVigenciaIlegible []RelojDescartado
	// VencimientosMasAlla son las ocurrencias posteriores a la ventana, una por
	// fila. Su cardinal es MasAllaDeLaVentana.
	VencimientosMasAlla []VencimientoDescartado
	// VencimientosAnterioresALaVigencia son las ocurrencias que el ciclo calcula
	// ANTES de que la obligacion entrara en vigor. Su cardinal es
	// VencimientosAntesDeLaVigencia.
	//
	// NO SON INCUMPLIMIENTOS Y LA LISTA TIENE QUE DECIRLO DONDE SE PINTA. El
	// ancla de una cadencia es un hecho del operador, y ese hecho puede ser muy
	// anterior a la norma: quien reviso su politica en 2022 no incumplia en 2023
	// un reglamento en vigor desde 2024. Es la unica de estas cinco listas que
	// ensena fechas PASADAS al lado de una obligacion, asi que es la unica que
	// puede leerse como una acusacion si se pinta sin su frase.
	VencimientosAnterioresALaVigencia []VencimientoDescartado
}

// Cuadra comprueba que cada lista retenida cubre EXACTAMENTE su contador.
//
// POR QUE ES UN METODO Y NO UN TEST. Porque la promesa que sostiene es de
// producto y no de repositorio: la pantalla enlaza una cifra a una lista, y si
// la lista no cuadra con la cifra, quien lo lea deja de creerse el pie entero.
// Un test lo comprueba con el corpus publicado; esto lo puede comprobar quien
// pinte, con el calendario que va a pintar.
//
// LA COMPROBACION VA POR metrica.Cuadra y no a mano. Es aritmetica de una cifra
// publicada, que es exactamente lo que ese paquete existe para no dejar suelto,
// y da el descuadre CON SIGNO: faltar y sobrar son fallos distintos y los dos
// son silencio, porque lo que sobra o falta simplemente no sale.
func (c Calendario) Cuadra() error {
	hitos := func(rs []RelojDescartado) int {
		n := 0
		for _, r := range rs {
			n += len(r.Hitos)
		}
		return n
	}
	var fallos []error
	for _, p := range []struct {
		contador int
		queEs    string
		lista    string
		n        int
	}{
		{c.HitosAplicables, "hitos que te alcanzan", "los hitos de RelojesAlcanzados",
			hitos(c.RelojesAlcanzados)},
		{c.HitosQueEstrenan, "hitos que estrenan dentro de la ventana",
			"los hitos de RelojesQueEstrenan", hitos(c.RelojesQueEstrenan)},
		{c.HitosQueCesan, "hitos que cesan dentro de la ventana",
			"los hitos nombrados de Ceses", hitosDeCeses(c.Ceses)},
		{c.HitosYaCesados, "hitos que ya cesaron", "los hitos de RelojesYaCesados",
			hitos(c.RelojesYaCesados)},
		{c.HitosQueEmpiezanDespues, "hitos que empiezan despues de la ventana",
			"los hitos de RelojesQueEmpiezanDespues", hitos(c.RelojesQueEmpiezanDespues)},
		{c.HitosConVigenciaIlegible, "hitos con la vigencia ilegible",
			"los hitos de RelojesConVigenciaIlegible", hitos(c.RelojesConVigenciaIlegible)},
		{c.MasAllaDeLaVentana, "vencimientos mas alla de la ventana",
			"las filas de VencimientosMasAlla", len(c.VencimientosMasAlla)},
		{c.VencimientosAntesDeLaVigencia, "vencimientos anteriores a la vigencia",
			"las filas de VencimientosAnterioresALaVigencia",
			len(c.VencimientosAnterioresALaVigencia)},
	} {
		if err := metrica.Cuadra(p.contador, p.queEs, map[string]int{p.lista: p.n}); err != nil {
			fallos = append(fallos, err)
		}
	}
	return errors.Join(fallos...)
}

// hitosDeCeses suma los hitos NOMBRADOS de la lista de ceses.
//
// Cuenta los nombres y no el campo `Hitos int` a proposito: lo que la seccion
// del calendario pinta son los nombres, asi que es esa longitud la que tiene
// que cuadrar con el contador. Sumar el `int` compararia el contador consigo
// mismo y daria verde con la lista corta, que es exactamente el descuadre.
func hitosDeCeses(cs []Cese) int {
	n := 0
	for _, c := range cs {
		n += len(c.NombresDeHitos)
	}
	return n
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
	// Las vencidas se acumulan POR OBLIGACION: cuatro anos de incumplimiento
	// anual son cuatro vencimientos y una sola noticia. El orden de aparicion
	// se guarda aparte porque recorrer el mapa no es un orden.
	vencidas := map[string]*Vencida{}
	var ordenVencidas []string
	// masAlla recuerda que obligaciones han producido ALGUN vencimiento
	// posterior a la ventana. Se resuelve al final: si ademas no han producido
	// ninguna fila, su destino es ese y no la nada.
	masAlla := map[string]bool{}

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
				cal.anotarDestino(o.ID, DestinoVigenciaIlegible)
				cal.RelojesConVigenciaIlegible = append(cal.RelojesConVigenciaIlegible,
					relojDescartado(p, o, "vigencia ilegible: "+err.Error()))
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
					cal.anotarDestino(o.ID, DestinoYaCeso)
					cal.RelojesYaCesados = append(cal.RelojesYaCesados,
						relojDescartado(p, o, "dejo de obligar antes de esta ventana"+finLegible(p, o)))
					continue
				}
				if !desde.Before(hasta) {
					cal.HitosQueEmpiezanDespues += hitosDeclarados(o)
					cal.anotarDestino(o.ID, DestinoEmpiezaDespues)
					cal.RelojesQueEmpiezanDespues = append(cal.RelojesQueEmpiezanDespues,
						relojDescartado(p, o, "empieza a obligar el "+desde.Format("2006-01-02")+
							", despues del "+hasta.Format("2006-01-02")))
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
				// LA LISTA VA EN LA UNIDAD DEL CONTADOR y se rellena AQUI,
				// antes de preguntar por el alcance, porque el contador
				// tampoco pregunta: los dos hablan del corpus entero. Poner
				// esta linea dentro del `if ok` la dejaria contando otra cosa
				// que el numero que abre, que es el descuadre.
				cal.RelojesQueEstrenan = append(cal.RelojesQueEstrenan,
					relojDescartado(p, o, "empieza a obligar el "+desde.Format("2006-01-02")+
						", dentro de esta ventana"))
				// Estrena y no te alcanza: la etiqueta es la del alcance, que
				// es la respuesta que le importa a quien lee. Estrena y SI te
				// alcanza: sale en la lista de estrenos.
				cal.anotarDestino(o.ID, DestinoNoTeAlcanza)
				if ok, supuesta := aplica(o.ID); ok {
					delete(cal.Destinos, o.ID)
					cal.anotarDestino(o.ID, DestinoEstrena)
					cal.HitosQueEstrenanYTeAlcanzan += hitosDeclarados(o)
					estrenos = append(estrenos, Estreno{
						Desde: desde, Marco: p.URN, Obligacion: o.ID,
						Titulo: o.TituloLegible(), Articulo: o.Articulo,
						Cita: o.Cita, Supuesta: supuesta, Hitos: hitosDeclarados(o),
						NombresDeHitos: nombresDeHitos(o),
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
				cal.anotarDestino(o.ID, DestinoNoTeAlcanza)
				continue
			}
			cal.HitosAplicables += hitosDeclarados(o)
			// LA UNICA DE ESTAS LISTAS QUE ENUMERA LO QUE SI ES TUYO. Va en la
			// unidad del contador (un elemento por obligacion con sus hitos
			// dentro) y su Regla dice CON QUE se decidio, no solo que se
			// decidio: la vigencia que lo pone en juego y si el hecho del que
			// cuelga el alcance es SUPUESTO. Sin esa segunda mitad, una
			// conjetura de un perfil de arranque se leeria como una respuesta
			// de la organizacion.
			cal.RelojesAlcanzados = append(cal.RelojesAlcanzados,
				relojDescartado(p, o, "en vigor"+inicioLegible(p, o)+
					", y la aplicabilidad lo deriva"+deQueHecho(supuesta)))

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
					NombresDeHitos: nombresDeHitos(o),
				})
			}

			vs, err := corpus.VencimientosDe(o, hechos, hasta)
			if err != nil {
				// Una primitiva sin ejecutor es una fila, no un silencio. Hoy
				// solo `puntual`, `periodica` y `plazo` tienen ejecutor.
				sin = append(sin, SinFecha{Marco: p.URN, Obligacion: o.ID,
					Titulo: o.TituloLegible(), Articulo: o.Articulo,
					Motivo: MotivoSinEjecutor, Regla: err.Error()})
				cal.anotarDestino(o.ID, DestinoSinFecha)
				continue
			}
			for _, v := range vs {
				switch v.Estado {
				case ventana.Determinado:
					// LAS DOS FORMAS DE ESTAR FUERA DE LA VENTANA NO SON LA
					// MISMA, y hasta el 29-08-2026 caian en el mismo cubo con
					// la etiqueta del futuro. Una fecha PASADA es un
					// incumplimiento en curso, que es la fila mas importante
					// que este producto puede imprimir; una posterior a la
					// ventana es algo que ya se vera.
					if v.Vence.Before(ahora) {
						// Una ocurrencia anterior a la entrada en vigor NO es
						// un incumplimiento: en esa fecha la norma no obligaba.
						if desde, err := p.InicioDeVigencia(o); err == nil && v.Vence.Before(desde) {
							cal.VencimientosAntesDeLaVigencia++
							cal.VencimientosAnterioresALaVigencia = append(
								cal.VencimientosAnterioresALaVigencia,
								vencimientoDescartado(p, o, v,
									"la ocurrencia cae antes del "+desde.Format("2006-01-02")+
										", que es cuando esta obligacion empezo a obligar: "+
										"ese dia la norma no obligaba"))
							continue
						}
						cal.VencimientosPasados++
						vd := vencidas[o.ID]
						if vd == nil {
							vd = &Vencida{
								Desde: v.Vence, Marco: p.URN, Obligacion: o.ID,
								Titulo: o.TituloLegible(), Articulo: o.Articulo,
								Cita: o.Cita, Hito: v.Hito, Regla: v.Regla,
								Supuesta: supuesta,
							}
							vencidas[o.ID] = vd
							ordenVencidas = append(ordenVencidas, o.ID)
						}
						vd.Ciclos++
						cal.anotarDestino(o.ID, DestinoVencida)
						// El MAS ANTIGUO es el que contesta "¿desde cuando?",
						// que es la pregunta de un inspector.
						if v.Vence.Before(vd.Desde) {
							vd.Desde, vd.Hito, vd.Regla = v.Vence, v.Hito, v.Regla
						}
						continue
					}
					if !v.Vence.Before(hasta) {
						cal.MasAllaDeLaVentana++
						masAlla[o.ID] = true
						cal.VencimientosMasAlla = append(cal.VencimientosMasAlla,
							vencimientoDescartado(p, o, v, v.Regla))
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
					cal.anotarDestino(o.ID, DestinoConFecha)
				case ventana.PendienteDeHecho:
					sf := SinFecha{Marco: p.URN, Obligacion: o.ID,
						Titulo: o.TituloLegible(), Articulo: o.Articulo, Hito: v.Hito,
						Motivo: MotivoPendienteDeHecho, Regla: v.Regla}
					if o.Temporalidad.Primitiva == "periodica" {
						sf.Cadencia = o.Temporalidad.Cadencia
						sf.OrigenDelIntervalo = o.Temporalidad.OrigenDelIntervalo
					}
					sin = append(sin, sf)
					cal.anotarDestino(o.ID, DestinoSinFecha)
				case ventana.SinPlazoLegal:
					sin = append(sin, SinFecha{Marco: p.URN, Obligacion: o.ID,
						Titulo: o.TituloLegible(), Articulo: o.Articulo, Hito: v.Hito,
						Motivo: MotivoSinPlazoLegal, Regla: v.Regla})
					cal.anotarDestino(o.ID, DestinoSinFecha)
				}
			}
		}
	}

	// EL ULTIMO DESTINO, que se resuelve al final porque solo se sabe cuando ya
	// se han visto TODOS los vencimientos de la obligacion: la que solo produjo
	// vencimientos posteriores a la ventana no tiene fila, y su ausencia tiene
	// nombre.
	for id := range masAlla {
		cal.anotarDestino(id, DestinoMasAllaDeLaVentana)
	}

	ordenarFechas(fechas)
	ordenarSinFecha(sin)
	// LAS LISTAS DE DESCARTE SE ORDENAN IGUAL QUE TODO LO DEMAS, y por lo
	// mismo: se recorren los paquetes en el orden en que lleguen, que no es un
	// orden. Sin esto, dos ejecuciones con el mismo corpus dan dos paginas
	// distintas y ninguna comparacion byte a byte prueba nada.
	for _, l := range [][]RelojDescartado{cal.RelojesAlcanzados, cal.RelojesQueEstrenan,
		cal.RelojesYaCesados, cal.RelojesQueEmpiezanDespues, cal.RelojesConVigenciaIlegible} {
		ordenarRelojesDescartados(l)
	}
	for _, l := range [][]VencimientoDescartado{cal.VencimientosMasAlla,
		cal.VencimientosAnterioresALaVigencia} {
		ordenarVencimientosDescartados(l)
	}
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
	// Las vencidas, de la mas antigua a la mas reciente: es el orden en que le
	// pesan a quien tiene que responder por ellas. Desempate por marco y
	// obligacion, que son identidades del dato, para que el orden sea total.
	for _, id := range ordenVencidas {
		cal.Vencidas = append(cal.Vencidas, *vencidas[id])
	}
	sort.Slice(cal.Vencidas, func(i, j int) bool {
		a, b := cal.Vencidas[i], cal.Vencidas[j]
		if !a.Desde.Equal(b.Desde) {
			return a.Desde.Before(b.Desde)
		}
		if a.Marco != b.Marco {
			return a.Marco < b.Marco
		}
		return a.Obligacion < b.Obligacion
	})
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

// relojDescartado compone una fila de descarte por obligacion.
//
// LOS HITOS SALEN DE hitosDeclarados, la MISMA funcion que alimenta el contador
// que esta lista abre. Es lo unico que garantiza que la lista y el numero digan
// lo mismo: dos formas de contar hitos, aunque hoy den igual, se separan a la
// tercera edicion y entonces el enlace manda a una lista que no cuadra.
func relojDescartado(p *corpus.Paquete, o corpus.Obligacion, regla string) RelojDescartado {
	return RelojDescartado{
		Marco: p.URN, Obligacion: o.ID, Titulo: o.TituloLegible(),
		Articulo: o.Articulo, Hitos: nombresDeHitos(o), Regla: regla,
	}
}

// vencimientoDescartado compone una fila de descarte por ocurrencia.
func vencimientoDescartado(p *corpus.Paquete, o corpus.Obligacion,
	v ventana.Vencimiento, regla string) VencimientoDescartado {

	return VencimientoDescartado{
		Vence: v.Vence, Marco: p.URN, Obligacion: o.ID, Titulo: o.TituloLegible(),
		Articulo: o.Articulo, Hito: v.Hito, Regla: regla,
	}
}

// finLegible dice hasta cuando obligaba, si se puede leer. Vacio si no.
//
// EL VALOR ILEGIBLE NO SE INVENTA. Si la fecha de fin no se lee, esta fila sale
// con su motivo generico y sin fecha, en vez de con una fecha en el ano 1: un
// dato que hay y no se entiende no es la nada, y tomarlo por el cero es
// inventarse un valor (invariante 8).
// inicioLegible dice desde cuando obliga, si se puede leer. Vacio si no.
//
// MISMA REGLA QUE finLegible Y POR LO MISMO: si la fecha no se lee, la fila sale
// sin ella en vez de con una fecha en el ano 1. Un dato que hay y no se entiende
// no es la nada, y tomarlo por el cero es inventarse un valor (invariante 8).
func inicioLegible(p *corpus.Paquete, o corpus.Obligacion) string {
	desde, err := p.InicioDeVigencia(o)
	if err != nil {
		return ""
	}
	return " desde el " + desde.Format("2006-01-02")
}

// deQueHecho dice si el alcance cuelga de un hecho SUPUESTO o de uno declarado.
//
// No es adorno: es la diferencia entre «esto te obliga» y «esto te obligaria si
// el perfil acierta», y una lista que las junta convierte una conjetura en una
// obligacion.
func deQueHecho(supuesta bool) string {
	if supuesta {
		return " de un hecho SUPUESTO, no declarado por ti"
	}
	return " de tus respuestas"
}

func finLegible(p *corpus.Paquete, o corpus.Obligacion) string {
	fin, hay, err := p.FinDeVigencia(o)
	if err != nil || !hay {
		return ""
	}
	return ", el " + fin.Format("2006-01-02")
}

func ordenarRelojesDescartados(r []RelojDescartado) {
	sort.SliceStable(r, func(i, j int) bool {
		if r[i].Marco != r[j].Marco {
			return r[i].Marco < r[j].Marco
		}
		return r[i].Obligacion < r[j].Obligacion
	})
}

// ordenarVencimientosDescartados ordena por fecha y desempata por identidad.
//
// El desempate no es celo: quince ocurrencias del mismo dia salen si no en el
// orden del recorrido de un mapa, y un dorado byte a byte las caza una vez de
// cada tres, que es la peor forma de rojo que hay.
func ordenarVencimientosDescartados(v []VencimientoDescartado) {
	sort.SliceStable(v, func(i, j int) bool {
		if !v[i].Vence.Equal(v[j].Vence) {
			return v[i].Vence.Before(v[j].Vence)
		}
		if v[i].Obligacion != v[j].Obligacion {
			return v[i].Obligacion < v[j].Obligacion
		}
		return v[i].Hito < v[j].Hito
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
func hitosDeclarados(o corpus.Obligacion) int { return len(nombresDeHitos(o)) }

// nombresDeHitos son los hitos de una obligacion, por nombre.
//
// ES LA UNICA FUENTE DE LA CUENTA, y por eso hitosDeclarados es su longitud y no
// una segunda forma de contar. Dos formas de contar hitos que hoy dan lo mismo
// se separan a la tercera edicion, y el sintoma seria una cifra del pie que no
// cuadra con la lista que abre: el fallo exacto que estas listas vienen a cerrar.
//
// Sin `hitos`, la temporalidad declara UNO en el campo `hito`, asi que el suelo
// es uno y nunca cero: una obligacion con temporalidad y cero hitos no existe.
// El nombre puede salir vacio (el campo es opcional en el formato) y NO se
// rellena con uno inventado: quien pinte omite el rotulo, que es la verdad, en
// vez de ensenar un nombre que el paquete no dijo.
func nombresDeHitos(o corpus.Obligacion) []string {
	if o.Temporalidad == nil {
		return nil
	}
	if len(o.Temporalidad.Hitos) > 0 {
		out := make([]string, 0, len(o.Temporalidad.Hitos))
		for _, h := range o.Temporalidad.Hitos {
			out = append(out, h.ID)
		}
		return out
	}
	return []string{o.Temporalidad.Hito}
}
