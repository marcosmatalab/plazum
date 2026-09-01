package acta

import (
	"fmt"
	"sort"

	"github.com/marcosmatalab/plazum/nucleo/accesos"
	"github.com/marcosmatalab/plazum/nucleo/auditoria"
	"github.com/marcosmatalab/plazum/nucleo/incidente"
)

// TODO LO QUE ESCRIBE PLAZUM EN EL ACTA, EN UN SITIO Y CON SU CLAVE.
//
// POR QUE ESTA LISTA EXISTE Y NO ESTA REPARTIDA POR LOS CONSTRUCTORES. El
// catalogo tiene que traer en espanol EXACTAMENTE estos textos, letra por letra,
// y eso se comprueba recorriendo esta lista, no escribiendola otra vez al lado
// del test. Una lista escrita dos veces es una lista que se queda vieja en una
// de las dos.
//
// Y LOS DESCARGOS NO SE ESCRIBEN AQUI: se toman de la constante del paquete que
// produce el dato (auditoria, accesos, incidente). Aqui solo se les pone la
// clave. Si manana accesos cambia su redaccion, esta lista cambia sola y el test
// del catalogo se pone rojo hasta que alguien mire el espanol; el INGLES no se
// entera, y por eso el arreglo de un descargo lleva en su commit la linea de si
// el ingles se reviso y por que.

// Los descargos. Trece frases distintas repartidas en diecisiete cubos: la de lo
// no auditado sirve a "diferida" y a "sin auditar", la de lo no revisado a
// "delegada" y a "sin revisar", y la de lo ilegible a las excusadas y a las que
// bloquean.
var (
	descargoNoAuditado     = Frase{"acta.descargo.no_auditado", auditoria.LaFraseDeLoNoAuditado}
	descargoIndependencia  = Frase{"acta.descargo.independencia", auditoria.LaFraseDeLaIndependencia}
	descargoSinResponsable = Frase{"acta.descargo.sin_responsable", auditoria.LaFraseDeLoSinResponsable}
	descargoSalidaAlcance  = Frase{"acta.descargo.salida_del_alcance", auditoria.LaFraseDeLaSalidaDelAlcance}
	descargoNoRevisado     = Frase{"acta.descargo.no_revisado", accesos.LaFraseDeLoNoRevisado}
	descargoIlegible       = Frase{"acta.descargo.ilegible", accesos.LaFraseDeLoIlegible}
	descargoNoNotificado   = Frase{"acta.descargo.no_notificado", incidente.LaFraseDeLoNoNotificado}
	descargoNoClasificado  = Frase{"acta.descargo.no_clasificado", incidente.LaFraseDeLoNoClasificado}
	descargoAsignacion     = Frase{"acta.descargo.asignacion_que_no_casa", LaFraseDeLaAsignacionQueNoCasa}
	descargoFueraPeriodo   = Frase{"acta.descargo.fuera_del_periodo", LaFraseDeLoFueraDelPeriodo}
	descargoNotificacion   = Frase{"acta.descargo.notificacion_que_no_casa", LaFraseDeLaNotificacionQueNoCasa}
	descargoEmpate         = Frase{"acta.descargo.empate_de_clasificacion", LaFraseDelEmpateDeClasificacion}
	descargoRemisionFuera  = Frase{"acta.descargo.remision_fuera_del_periodo", LaFraseDeLaRemisionFueraDelPeriodo}
)

// DescargosDelActa son las trece frases que van pegadas a un numero para que no
// se lea como culpa. Es lo que recorre el test del catalogo.
func DescargosDelActa() []Frase {
	return []Frase{
		descargoNoAuditado, descargoIndependencia, descargoSinResponsable, descargoSalidaAlcance,
		descargoNoRevisado, descargoIlegible, descargoNoNotificado, descargoNoClasificado,
		descargoAsignacion, descargoFueraPeriodo, descargoNotificacion, descargoEmpate,
		descargoRemisionFuera,
	}
}

// Los rotulos de reparto: QUE se reparte en cada particion.
var (
	repartoDeCobertura     = Frase{"acta.reparto.cobertura", "las unidades del alcance del programa"}
	repartoDeHallazgos     = Frase{"acta.reparto.hallazgos", "los hallazgos vivos en este ciclo"}
	repartoDeArrastre      = Frase{"acta.reparto.arrastre", "lo que el ciclo anterior dejo sin auditar"}
	repartoDeAsignaciones  = Frase{"acta.reparto.asignaciones", "las asignaciones de responsable aportadas"}
	repartoDeIndependencia = Frase{"acta.reparto.independencia", "las auditorias de una unidad (pares sesion-unidad distintos)"}
	repartoDeAccesos       = Frase{"acta.reparto.accesos", "los accesos de la instantanea"}
	repartoDeLineas        = Frase{"acta.reparto.lineas", "las lineas de datos del fichero"}
	repartoDeIncidentes    = Frase{"acta.reparto.incidentes", "los incidentes aportados"}
	repartoDeClasificacion = Frase{"acta.reparto.clasificacion", "los incidentes del periodo, por su clasificacion al cierre del periodo"}
	repartoDeEsperadas     = Frase{"acta.reparto.esperadas", "las notificaciones que el corpus espera"}
	repartoDeRemision      = Frase{"acta.reparto.remision", "las notificaciones esperadas de incidentes del periodo"}
)

// Los cubos. Los tres de cobertura y los cuatro de estado de acceso NO se
// escriben aqui: salen del vocabulario de su paquete, con una funcion que no
// tiene rama por defecto. Un valor de vocabulario que nadie haya traido aqui da
// una frase sin clave, y una frase sin clave no compone (invariante 8: el valor
// que sale por olvido es un error, no un defecto silencioso).
var (
	cuboHallazgoAbierto     = Frase{"acta.cubo.hallazgo_abierto", "abierto, anotado en este ciclo"}
	cuboHallazgoArrastrado  = Frase{"acta.cubo.hallazgo_arrastrado", "abierto, arrastrado de ciclos anteriores"}
	cuboHallazgoCerrado     = Frase{"acta.cubo.hallazgo_cerrado", "cerrado en este ciclo"}
	cuboSigueEnAlcance      = Frase{"acta.cubo.sigue_en_alcance", "sigue en el alcance de este ciclo"}
	cuboSalioDelAlcance     = Frase{"acta.cubo.salio_del_alcance", "ya no esta en el alcance de este ciclo"}
	cuboAsignacionCasa      = Frase{"acta.cubo.asignacion_casa", "casa con una unidad del alcance"}
	cuboAsignacionNoCasa    = Frase{"acta.cubo.asignacion_no_casa", "no casa con ninguna unidad del alcance"}
	cuboAuditorDistinto     = Frase{"acta.cubo.auditor_distinto", "el auditor no responde de la unidad"}
	cuboAuditorResponsable  = Frase{"acta.cubo.auditor_responsable", "el auditor es quien responde de la unidad"}
	cuboSinResponsable      = Frase{"acta.cubo.sin_responsable", "no consta quien responde de la unidad"}
	cuboLineaLegible        = Frase{"acta.cubo.linea_legible", "legible, un acceso"}
	cuboLineaDuplicada      = Frase{"acta.cubo.linea_duplicada", "repetia un acceso ya listado"}
	cuboLineaExcusada       = Frase{"acta.cubo.linea_excusada", "ilegible, excusada por escrito"}
	cuboLineaBloquea        = Frase{"acta.cubo.linea_bloquea", "ilegible y sin excusar, bloquea el cierre"}
	cuboIncidenteDentro     = Frase{"acta.cubo.incidente_dentro", "conocido dentro del periodo"}
	cuboIncidenteFuera      = Frase{"acta.cubo.incidente_fuera", "conocido fuera del periodo, no lo cuenta este acta"}
	cuboConClasificacion    = Frase{"acta.cubo.con_clasificacion", "con clasificacion vigente"}
	cuboClasificacionEmpate = Frase{"acta.cubo.clasificacion_empatada", "con dos clasificaciones en el mismo instante"}
	cuboSinClasificacion    = Frase{"acta.cubo.sin_clasificacion", "sin clasificacion que conste"}
	cuboEsperadaCasa        = Frase{"acta.cubo.esperada_casa", "casa con un incidente del periodo"}
	cuboEsperadaNoCasa      = Frase{"acta.cubo.esperada_no_casa", "no casa con ningun incidente del periodo"}
	cuboRemitidaDentro      = Frase{"acta.cubo.remitida_dentro", "consta remitida dentro del periodo"}
	cuboRemitidaFuera       = Frase{"acta.cubo.remitida_fuera", "consta remitida fuera del periodo"}
	cuboNoConstaRemitida    = Frase{"acta.cubo.no_consta_remitida", "no consta remitida"}
)

// cuboDeCobertura traduce el vocabulario de auditoria a una frase con clave.
//
// SIN RAMA POR DEFECTO. Una cobertura nueva que nadie haya traido aqui devuelve
// la frase vacia, y una frase vacia no compone: el acta se para y dice cual es.
// La alternativa, caer a un rotulo generico, daria un cubo que se pinta con un
// nombre que no es el suyo y nadie lo notaria.
func cuboDeCobertura(c auditoria.Cobertura) Frase {
	switch c {
	case auditoria.Auditada:
		return Frase{"acta.cubo.auditada", string(c)}
	case auditoria.Diferida:
		return Frase{"acta.cubo.diferida", string(c)}
	case auditoria.SinAuditar:
		return Frase{"acta.cubo.sin_auditar", string(c)}
	}
	return Frase{}
}

// cuboDeEstado hace lo mismo con el vocabulario de accesos.
func cuboDeEstado(e accesos.Estado) Frase {
	switch e {
	case accesos.Aprobada:
		return Frase{"acta.cubo.aprobada", string(e)}
	case accesos.Revocada:
		return Frase{"acta.cubo.revocada", string(e)}
	case accesos.Delegada:
		return Frase{"acta.cubo.delegada", string(e)}
	case accesos.SinRevisar:
		return Frase{"acta.cubo.sin_revisar", string(e)}
	}
	return Frase{}
}

// CubosDelActa son todos los rotulos de cubo que el acta puede emitir, incluidos
// los que salen de un vocabulario de otro paquete. Es lo que recorre el test del
// catalogo, y lo recorre SOBRE EL VOCABULARIO, no sobre una lista escrita a mano:
// el dia que auditoria anada una cobertura, esto la trae sola y el catalogo se
// pone rojo por faltarle la clave, que es justo lo que tiene que pasar.
func CubosDelActa() []Frase {
	out := []Frase{
		cuboHallazgoAbierto, cuboHallazgoArrastrado, cuboHallazgoCerrado,
		cuboSigueEnAlcance, cuboSalioDelAlcance, cuboAsignacionCasa, cuboAsignacionNoCasa,
		cuboAuditorDistinto, cuboAuditorResponsable, cuboSinResponsable,
		cuboLineaLegible, cuboLineaDuplicada, cuboLineaExcusada, cuboLineaBloquea,
		cuboIncidenteDentro, cuboIncidenteFuera, cuboConClasificacion, cuboClasificacionEmpate,
		cuboSinClasificacion, cuboEsperadaCasa, cuboEsperadaNoCasa,
		cuboRemitidaDentro, cuboRemitidaFuera, cuboNoConstaRemitida,
	}
	for _, c := range auditoria.CoberturasPosibles() {
		out = append(out, cuboDeCobertura(c))
	}
	for _, e := range accesos.EstadosPosibles() {
		out = append(out, cuboDeEstado(e))
	}
	return out
}

// RepartosDelActa son los once rotulos de particion.
func RepartosDelActa() []Frase {
	return []Frase{
		repartoDeCobertura, repartoDeHallazgos, repartoDeArrastre, repartoDeAsignaciones,
		repartoDeIndependencia, repartoDeAccesos, repartoDeLineas, repartoDeIncidentes,
		repartoDeClasificacion, repartoDeEsperadas, repartoDeRemision,
	}
}

// Los parrafos que escribe plazum: su clave y su PLANTILLA, que es lo que va al
// catalogo. Los %s los rellena el dato, y el dato no se traduce: un
// identificador, una fecha y un recuento se leen igual en cualquier idioma; lo
// que se traduce es la frase que los rodea.
//
// Estan aqui y no en el sitio donde se usan por lo mismo que los descargos: el
// catalogo se comprueba recorriendo esta lista, no escribiendola otra vez.
var (
	pfCompone = Frase{"acta.parrafo.compone", "Este documento lo compone plazum a partir de " +
		"los registros que la propia organizacion tiene dentro. Cada numero lleva la lista de " +
		"lo que lo compone: en este acta no hay ninguna cifra que haya que creerse."}
	pfNoDiceCumplido = Frase{"acta.parrafo.no_dice_cumplido", "Que este acta exista no dice que " +
		"las obligaciones que cubre esten cumplidas. Dice que hay un acta, con estos datos y " +
		"este periodo. Quien decide si el sistema funciona es la direccion, y lo hace en la " +
		"ultima seccion."}
	pfEvidenciaDe = Frase{"acta.parrafo.evidencia_de",
		"Este acta es evidencia de las obligaciones que siguen."}
	pfIdiomaDelCorpus = Frase{"acta.parrafo.idioma_del_corpus", "Sus titulos y sus citas van en " +
		"el idioma de su paquete y no se traducen: el texto de una norma es de su fuente, y " +
		"traducirlo crearia obra derivada."}
	pfSinCubre = Frase{"acta.parrafo.sin_cubre", "No consta de que obligacion es evidencia este " +
		"acta. Se compone igual, pero sin eso nadie puede decir para que sirve delante de un " +
		"auditor."}
	pfAsistieron      = Frase{"acta.parrafo.asistieron", "Asistieron: %s."}
	pfSinQuienAsistio = Frase{"acta.parrafo.sin_quien_asistio", "No consta quien asistio a la " +
		"revision. Lo aporta quien la convoca: plazum no lo sabe y no se lo inventa."}
	pfPrograma   = Frase{"acta.parrafo.programa", "Programa %s, ciclo %s del %s al %s."}
	pfArrastraDe = Frase{"acta.parrafo.arrastra_de", "Arrastra del ciclo %s: lo que aquel no " +
		"llego a auditar entra en este con su edad."}
	pfPrimerCiclo = Frase{"acta.parrafo.primer_ciclo", "Es el primer ciclo del programa, asi que " +
		"no arrastra nada. Eso es distinto de no haberlo mirado."}
	pfSinPrograma = Frase{"acta.parrafo.sin_programa", "No consta ningun programa de auditoria " +
		"interna. Sin el, este acta no puede decir que se ha quedado sin auditar ni desde " +
		"cuando, que es la pregunta que hace siempre un auditor externo. Hace falta abrir un " +
		"programa con su ciclo y su alcance."}
	pfCampana = Frase{"acta.parrafo.campana", "Campana %s sobre la instantanea sellada %s " +
		"(sha256 del fichero %s). Con ese fichero delante, cualquiera repite la lectura y " +
		"comprueba que se reviso esto y no otra cosa."}
	pfSinCampana = Frase{"acta.parrafo.sin_campana", "No consta ninguna campana de revision de " +
		"accesos. Sin ella este acta no puede decir cuantos permisos se miraron ni sobre que " +
		"censo. Hace falta subir una instantanea de accesos y abrir la campana sobre ella."}
	pfVerboDelPeriodo = Frase{"acta.parrafo.verbo_del_periodo", "Un incidente es de este periodo " +
		"si su PRIMER CONOCIMIENTO cae dentro, no si ocurrio dentro: lo que una revision por la " +
		"direccion puede juzgar es lo que la organizacion pudo hacer desde que lo supo."}
	pfSinIncidentes = Frase{"acta.parrafo.sin_incidentes", "No consta ningun registro de " +
		"incidentes. Esto NO dice que no haya habido ninguno: dice que nadie ha conectado el " +
		"registro, que es otra cosa. Sin el, este acta no puede decir si se notifico lo que " +
		"habia que notificar."}
	pfPlazumNoEscribe = Frase{"acta.parrafo.plazum_no_escribe", LaFraseDeLoQuePlazumNoEscribe}
)

// ParrafosDelActa son las plantillas de todo lo que escribe plazum en prosa.
func ParrafosDelActa() []Frase {
	return []Frase{
		pfCompone, pfNoDiceCumplido, pfEvidenciaDe, pfIdiomaDelCorpus, pfSinCubre,
		pfAsistieron, pfSinQuienAsistio, pfPrograma, pfArrastraDe, pfPrimerCiclo,
		pfSinPrograma, pfCampana, pfSinCampana, pfVerboDelPeriodo, pfSinIncidentes,
		pfPlazumNoEscribe,
	}
}

// CadenasDelActa es TODO lo que plazum escribe en un acta y que por tanto tiene
// que estar en el catalogo: rotulos de reparto, rotulos de cubo, descargos y
// plantillas de parrafo. Es lo que recorre el test del catalogo, en las dos
// direcciones.
func CadenasDelActa() []Frase {
	out := append([]Frase(nil), RepartosDelActa()...)
	out = append(out, CubosDelActa()...)
	out = append(out, DescargosDelActa()...)
	out = append(out, ParrafosDelActa()...)
	sort.Slice(out, func(i, j int) bool { return out[i].Clave < out[j].Clave })
	return out
}

// dePlazum arma un parrafo de plazum: su frase con los huecos ya rellenos y los
// datos que los rellenaron.
//
// El texto se resuelve AQUI con la misma plantilla que va al catalogo en
// espanol, asi que el board pack impreso y la pantalla en espanol no pueden
// separarse: son la misma cadena rellenada dos veces.
func dePlazum(f Frase, args ...string) Parrafo {
	vals := make([]any, len(args))
	for i, a := range args {
		vals[i] = a
	}
	return Parrafo{
		Frase: Frase{Clave: f.Clave, Texto: fmt.Sprintf(f.Texto, vals...)},
		Args:  args,
		De:    DePlazum,
	}
}
