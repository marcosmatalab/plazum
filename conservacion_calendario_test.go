package plazum

// LA LEY DE CONSERVACION DEL CALENDARIO: ningun reloj del corpus desaparece en
// silencio.
//
// # El fallo que la trae, y su forma
//
// El 03-09-2026 se movio QUINCE MESES la vigencia del art. 14.6 del CRA, que es
// la fecha mas cercana de todo el corpus (aplicable el 11-09-2026), y NO SE PUSO
// ROJA NI UNA PUERTA. El unico efecto medible fue que el reloj DESAPARECE del
// calendario: pasa de la seccion de estrenos a un contador de descarte.
//
// Y no es que faltaran leyes de conservacion. Habia tres y las tres daban verde:
//
//	particion por TIEMPO    en vigor + estrenan + ya cesados + empiezan despues
//	                        + ilegibles = instalados      (contabilidad_test.go)
//	particion por ALCANCE   alcanzados + no alcanzados = en vigor
//	particion por DESTINO   todo reloj instalado acaba en exactamente un destino
//	                        (nucleo/pantalla/conservacion_extremo_a_extremo_test.go)
//
// Las tres siguen cuadrando DESPUES de la mutacion, y por una razon que hay que
// leer despacio: **la mutacion no rompe ninguna particion, MUEVE una obligacion
// de un cubo visible a un cubo de ausencia legitima**. «Empieza a obligar mas
// alla de la ventana» es una respuesta perfectamente valida, la derivacion la
// cuenta, y la suma sigue dando. Una ley que solo comprueba que la suma cuadre
// no puede ver eso: para verlo hay que saber CUANTOS habia en cada cubo, y eso
// no es una suma, es un CARDINAL.
//
// # Que dice esta ley
//
//	Toda obligacion con reloj del corpus, o SE VE en el calendario de algun
//	perfil, o esta en una categoria de ausencia DECLARADA, con su motivo y con
//	su cardinal escrito. Y los cubos cuadran: instaladas = las que se ven + la
//	suma de las ausencias.
//
// Es la misma forma que la ley de conservacion de la ingesta de CSV
// (`nucleo/censo.Instantanea.Cuadra`: las lineas que cuentan los cubos tienen
// que cuadrar con las que conto un camino independiente) y que la de los cubos
// de la UAR (`nucleo/accesos.Campana.Cuenta`: todo acceso cae en exactamente un
// estado y los cubos vacios tambien salen). Lo que se anade aqui es el CARDINAL
// COMPARADO CON IGUALDAD EXACTA EN LOS DOS SENTIDOS, que es lo unico que
// convierte «no se ha perdido nada» en «no se ha MOVIDO nada».
//
// # Por que la unidad es la OBLIGACION y no el hito
//
// Porque es la unidad que la derivacion RETIENE por elemento:
// `pantalla.Calendario.Destinos` es un mapa de obligacion a etiqueta. Los
// contadores de la contabilidad van en HITOS, que es otra unidad, y mezclarlas
// daria una suma que cuadra por casualidad o que no cuadra nunca. Una ley en dos
// unidades no es una ley.
//
// # Por que campo casa el emparejamiento, y si ese campo es fiable
//
// (Invariante 7.) Casa por `corpus.Obligacion.ID`, que es el mismo campo con el
// que `Derivar12Meses` etiqueta `Destinos`. Es un campo del paquete de datos,
// dentro de lo que el cargador verifica por huella, y NO es un indice ni una
// posicion: reordenar el corpus no mueve ni un emparejamiento.
//
// Lo que ese campo NO trae de fabrica es unicidad GLOBAL: el mapa `Destinos` es
// plano sobre el corpus entero, asi que dos paquetes con el mismo ID de
// obligacion se pisarian y esta ley contaria uno donde hay dos. Por eso la
// unicidad se comprueba aqui, antes de contar nada: es la condicion de la que
// depende el emparejamiento y no se supone.
//
// # Por que el censo se enumera DEL ARBOL
//
// El universo sale de `corpus.Cargar("paquetes")` y las vistas salen de leer el
// directorio `perfiles/`. Ni una lista escrita al lado. Es el patron de
// `cmd/plazum/alcanzabilidad.go` y de `primitivas_alcanzables_test.go`: un censo
// escrito a mano acaba siendo una lista de lo que hubo.

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/marcosmatalab/plazum/nucleo/aplicabilidad"
	"github.com/marcosmatalab/plazum/nucleo/corpus"
	"github.com/marcosmatalab/plazum/nucleo/metrica"
	"github.com/marcosmatalab/plazum/nucleo/pantalla"
	"github.com/marcosmatalab/plazum/nucleo/ventana"
	"github.com/marcosmatalab/plazum/perfiles"
)

// instanteDeLaConservacion es el instante desde el que se juzga, y entra como
// dato igual que en el nucleo.
//
// VA CABLEADO A PROPOSITO. Un censo con cardinales exactos calculado con el
// reloj del sistema cambiaria de numero cada dia sin que nadie tocara nada, y un
// numero que se mueve solo no puede toparse. Con el instante fijo, lo unico que
// mueve estos cardinales es el CORPUS, que es exactamente lo que esta ley
// vigila. Y no compara con nada que ocurra en tiempo real, asi que no es una
// bomba de mecha encendida: no puede caducar sola.
var instanteDeLaConservacion = time.Date(2026, 9, 3, 9, 0, 0, 0, time.UTC)

// plantillaDeLaConservacion son los empleados con los que se monta cada perfil.
//
// Doscientos, que es el CISO de la tercera pasada, y ademas es el numero que
// enciende TODAS las bandas de plantilla de los perfiles publicados (la mas alta
// es 50). Con una plantilla menor, las bandas altas no se afirmarian y esta ley
// estaria midiendo un perfil mas pequeno que el que se vende.
const plantillaDeLaConservacion = 200

// veredictoDelReloj es donde acaba una obligacion con reloj mirando TODAS las
// vistas a la vez. Vocabulario cerrado.
//
// No es lo mismo que `pantalla.Destino`, y la diferencia es el fondo de esta
// ley: un destino es por vista (una obligacion tiene uno por cada perfil), y un
// veredicto es la respuesta unica a «¿alguien llega a ver esto?».
type veredictoDelReloj uint8

const (
	// relojSinDeclarar es el VALOR CERO Y ES INVALIDO. Una obligacion que
	// llegue aqui con el cero es una que ningun camino etiqueto, que es
	// justamente el silencio que esta ley persigue (invariante 8).
	relojSinDeclarar veredictoDelReloj = iota
	// relojSeVe: sale en el calendario de al menos un perfil, en una seccion
	// que el usuario ve.
	relojSeVe
	// relojNingunPerfilLoAlcanza: esta en vigor y ningun perfil publicado lo
	// alcanza. Es la ausencia mas grande y la mas legitima, y tiene su puerta
	// declarada: `plazum calendario --todos-los-relojes` (D-13).
	relojNingunPerfilLoAlcanza
	// relojTodosSusVencimientosMasAlla: algun perfil lo alcanza y todas sus
	// fechas caen despues de los doce meses.
	relojTodosSusVencimientosMasAlla
	// relojEmpiezaDespuesDeLaVentana: todavia no obliga y empezara despues de
	// la ventana. ES EL CUBO AL QUE CAYO EL ART. 14.6 CON LA MUTACION.
	relojEmpiezaDespuesDeLaVentana
	// relojDejoDeObligarAntes: dejo de obligar antes de la ventana.
	relojDejoDeObligarAntes
	// relojVigenciaIlegible: su vigencia no se puede leer.
	relojVigenciaIlegible
)

func (v veredictoDelReloj) String() string {
	switch v {
	case relojSeVe:
		return "se ve en el calendario de algun perfil"
	case relojNingunPerfilLoAlcanza:
		return "en vigor y ningun perfil lo alcanza (puerta: --todos-los-relojes)"
	case relojTodosSusVencimientosMasAlla:
		return "te alcanza y todas sus fechas caen mas alla de la ventana"
	case relojEmpiezaDespuesDeLaVentana:
		return "empieza a obligar mas alla de la ventana"
	case relojDejoDeObligarAntes:
		return "dejo de obligar antes de la ventana"
	case relojVigenciaIlegible:
		return "vigencia ilegible"
	default:
		// El cero se nombra por lo que es. Llamarlo «desconocido» suavizaria
		// justo el caso que esto existe para cazar.
		return "SIN DECLARAR (valor cero)"
	}
}

// veredictosConocidos es el vocabulario cerrado, sin el valor cero.
//
// Cerrado a proposito: un veredicto nuevo que nadie anada aqui rompe la ley, que
// es lo que se quiere el dia que alguien meta una rama nueva en la derivacion.
var veredictosConocidos = []veredictoDelReloj{
	relojSeVe, relojNingunPerfilLoAlcanza, relojTodosSusVencimientosMasAlla,
	relojEmpiezaDespuesDeLaVentana, relojDejoDeObligarAntes, relojVigenciaIlegible,
}

// CensoEsperado es el CARDINAL de cada cubo, y se compara con IGUALDAD EXACTA en
// los dos sentidos.
//
// EL CARDINAL SE ESCRIBE PARA QUE MOLESTE, igual que `SinDerivacionEsperadas` y
// que `PUERTAS_ESPERADAS`. Con un `>=` el cubo de las ausencias podria crecer sin
// que nadie se enterara, que es literalmente el fallo del art. 14.6; con un `<=`,
// abrir un reloj nuevo no obligaria a subir el numero de los que se ven.
//
// COMO SE ACTUALIZA CUANDO SALTA. La puerta imprime el censo entero listo para
// pegar. Pero antes de pegarlo hay UNA pregunta que contestar en voz alta, y es
// la razon de ser de todo esto: **¿ha bajado el cubo de los que SE VEN?** Si ha
// bajado, un reloj que un cliente veia ha dejado de verse, y eso no se arregla
// bajando el numero: se arregla sabiendo por que, y escribiendolo.
var CensoEsperado = map[veredictoDelReloj]int{
	relojSeVe:                  94,
	relojNingunPerfilLoAlcanza: 136,
	// VACIO HOY, y se declara igual. Un cubo que solo aparece cuando tiene algo
	// dentro es un cubo que nadie echa de menos: con el cero escrito, el dia que
	// deje de estar vacio esta puerta lo dice. Su control positivo no lo da el
	// corpus, lo da el caso sintetico de TestCadaVeredictoTieneSuInquilino.
	relojTodosSusVencimientosMasAlla: 0,
	relojEmpiezaDespuesDeLaVentana:   21,
	relojDejoDeObligarAntes:          1,
	// Vacio hoy, con el mismo trato y por la misma razon.
	relojVigenciaIlegible: 0,
}

// RelojesQueSeVenEsperados son, POR NOMBRE, los relojes que hoy sale en el
// calendario de algun perfil publicado.
//
// POR QUE UNA LISTA Y NO SOLO EL CARDINAL, y salio de intentar tumbar el propio
// censo. Con solo el numero, DOS MOVIMIENTOS OPUESTOS SE CANCELAN: si un reloj
// deja de verse y otro empieza a verse en la misma pasada, `relojSeVe` sigue
// valiendo 92 y el cubo del que salio y al que entro el otro tambien cuadran.
// El censo daria verde con un reloj perdido dentro. No es una hipotesis comoda:
// es exactamente lo que pasa cuando alguien mueve una vigencia y anade una
// obligacion en el mismo commit, que es un commit de corpus normal.
//
// Con la lista, el rojo ademas NOMBRA al que se cayo, que es lo que hace falta
// para decidir en un minuto si es un fallo o una novedad de la norma. Un
// cardinal solo dice que algo se movio.
//
// SE COMPARA COMO CONJUNTO Y EN LOS DOS SENTIDOS: lo que esta y ha dejado de
// verse, y lo que se ve y no estaba declarado. El segundo sentido no es
// simetria por gusto: sin el, esta lista se convierte poco a poco en una lista
// de lo que hubo.
var RelojesQueSeVenEsperados = []string{
	"aiact.art111_4.marcado_de_lo_ya_comercializado",
	"aiact.art50.informacion_antes_de_la_primera_interaccion",
	"cra.art14.notificacion_de_incidente_grave",
	"cra.art14.notificacion_de_vulnerabilidad_explotada",
	"cra.art14_6.informe_provisional_a_instancia_del_csirt",
	"cra.art14_8.informacion_a_los_usuarios_afectados",
	"eni.art9_1.mantenimiento_de_los_inventarios_de_informacion_administrativa",
	"ens.anexoI.reevaluacion_de_la_categoria",
	"ens.art10.3.reevaluacion_periodica_de_las_medidas",
	"ens.art27.mejora_continua",
	"ens.art31.auditoria_extraordinaria",
	"ens.art31.auditoria_ordinaria",
	"ens.ines.informe_anual",
	"ens.its_conformidad.certificacion_media_alta",
	"ens.its_incidentes.estadisticas_anuales",
	"ens.its_incidentes.notificacion_al_ccn",
	"iso27001.ritual.apreciacion_riesgos",
	"iso27001.ritual.auditoria_interna",
	"iso27001.ritual.formacion_y_concienciacion",
	"iso27001.ritual.plan_accion_no_conformidad",
	"iso27001.ritual.prueba_de_continuidad_tic",
	"iso27001.ritual.revision_de_derechos_de_acceso",
	"iso27001.ritual.revision_declaracion_aplicabilidad",
	"iso27001.ritual.revision_direccion",
	"iso27001.ritual.revision_independiente",
	"ley2-2023.art26_2.conservacion_maxima_del_libro_registro",
	"ley2-2023.art32_4.supresion_de_la_comunicacion_sin_actuaciones",
	"ley2-2023.art7_2.reunion_presencial_con_el_informante",
	"ley2-2023.art8_3.notificacion_del_nombramiento_o_cese_del_responsable",
	"ley2-2023.art9_2_c.acuse_de_recibo_al_informante",
	"ley2-2023.art9_2_d.respuesta_a_las_actuaciones_de_investigacion",
	"ley2-2023.art9_2_j.remision_al_ministerio_fiscal",
	"lopdgdd.art34_3.comunicacion_del_delegado_a_la_autoridad",
	"lopdgdd.art36_4.comunicacion_de_la_vulneracion_relevante",
	"lopdgdd.art37_1.comunicacion_de_la_decision_al_afectado",
	"lopdgdd.art37_2.respuesta_del_delegado_a_la_reclamacion_remitida",
	"lopdgdd.art65_4.respuesta_del_responsable_a_la_reclamacion_remitida",
	"nis2tec.anexo.10_1_3.revision_de_la_asignacion_de_personal",
	"nis2tec.anexo.10_2_3.revision_de_la_politica_de_comprobacion_de_antecedentes",
	"nis2tec.anexo.10_4_2.revision_de_los_procedimientos_disciplinarios",
	"nis2tec.anexo.11_1_3.revision_de_las_politicas_de_control_de_accesos",
	"nis2tec.anexo.11_2_3.revision_de_los_derechos_de_acceso",
	"nis2tec.anexo.11_3_3.revision_de_los_accesos_privilegiados",
	"nis2tec.anexo.11_5_4.revision_de_las_identidades",
	"nis2tec.anexo.11_6_4.revision_de_los_procedimientos_de_autenticacion",
	"nis2tec.anexo.12_1_3.revision_de_los_niveles_de_clasificacion",
	"nis2tec.anexo.12_2_3.revision_de_la_politica_de_gestion_de_activos",
	"nis2tec.anexo.12_3_3.revision_de_la_politica_de_soportes_extraibles",
	"nis2tec.anexo.12_4_3.revision_del_inventario_de_activos",
	"nis2tec.anexo.13_1_3.revision_de_las_medidas_sobre_servicios_publicos",
	"nis2tec.anexo.13_2_3.revision_de_las_medidas_frente_a_amenazas_fisicas",
	"nis2tec.anexo.13_3_3.revision_de_las_medidas_de_control_de_acceso_fisico",
	"nis2tec.anexo.1_1_2.revision_de_la_politica",
	"nis2tec.anexo.1_2_6.revision_de_roles_y_responsabilidades",
	"nis2tec.anexo.2_1_4.revision_de_la_evaluacion_de_riesgos",
	"nis2tec.anexo.2_2_1.informe_de_cumplimiento_al_organo_de_direccion",
	"nis2tec.anexo.2_2_3.control_del_cumplimiento",
	"nis2tec.anexo.2_3_4.revision_independiente",
	"nis2tec.anexo.3_1_3.prueba_de_la_politica_de_gestion_de_incidentes",
	"nis2tec.anexo.3_2_4.revision_de_tendencias_en_los_registros",
	"nis2tec.anexo.3_3_2.formacion_en_el_mecanismo_de_notificacion_de_sucesos",
	"nis2tec.anexo.3_4_2_b.evaluacion_de_incidentes_recurrentes",
	"nis2tec.anexo.3_5_5.prueba_de_los_procedimientos_de_respuesta",
	"nis2tec.anexo.3_6_3.comprobacion_de_las_revisiones_posincidente",
	"nis2tec.anexo.4_1_4.prueba_del_plan_de_continuidad_y_recuperacion",
	"nis2tec.anexo.4_2_3.verificacion_de_la_integridad_de_las_copias",
	"nis2tec.anexo.4_2_6.prueba_de_recuperacion_de_copias_y_redundancias",
	"nis2tec.anexo.4_3_4.revision_de_los_planes_de_gestion_de_crisis",
	"nis2tec.anexo.5_1_6.revision_de_la_politica_de_cadena_de_suministro",
	"nis2tec.anexo.5_1_7.supervision_de_los_informes_de_nivel_de_servicio",
	"nis2tec.anexo.6_10_2.exploracion_de_vulnerabilidades",
	"nis2tec.anexo.6_10_4.revision_de_los_canales_de_vulnerabilidades",
	"nis2tec.anexo.6_1_3.revision_de_los_procedimientos_de_adquisicion",
	"nis2tec.anexo.6_2_4.revision_de_las_normas_de_desarrollo_seguro",
	"nis2tec.anexo.6_3_3.revision_de_las_configuraciones",
	"nis2tec.anexo.6_4_4.revision_de_los_procedimientos_de_gestion_de_cambios",
	"nis2tec.anexo.6_5_3.revision_de_las_politicas_de_pruebas_de_seguridad",
	"nis2tec.anexo.6_7_3.revision_de_las_medidas_de_seguridad_de_la_red",
	"nis2tec.anexo.6_8_3.revision_de_la_segmentacion_de_la_red",
	"nis2tec.anexo.6_9_2.comprobacion_de_la_cobertura_antimalware",
	"nis2tec.anexo.7_3.revision_de_las_orientaciones_de_evaluacion_de_eficacia",
	"nis2tec.anexo.8_1_3.actualizacion_y_oferta_del_programa_de_sensibilizacion",
	"nis2tec.anexo.8_2_1.formacion_de_los_roles_especializados",
	"nis2tec.anexo.8_2_5.actualizacion_del_programa_de_formacion",
	"nis2tec.anexo.9_3.revision_de_las_orientaciones_de_criptografia",
	"rgpd.art12_3.aviso_de_la_prorroga_al_interesado",
	"rgpd.art12_3.respuesta_a_la_solicitud_del_interesado",
	"rgpd.art12_4.informacion_de_la_no_actuacion",
	"rgpd.art19.comunicacion_a_los_destinatarios",
	"rgpd.art32_1_d.verificacion_de_la_eficacia_de_las_medidas",
	"rgpd.art33.notificacion_brecha",
	"rgpd.art34_1.comunicacion_de_la_violacion_al_interesado",
	"rgpd.art35_1.evaluacion_de_impacto_antes_del_tratamiento",
	"rgpd.art36_1.consulta_previa_a_la_autoridad_de_control",
}

// -----------------------------------------------------------------------------
// Las vistas: un calendario por perfil publicado.
// -----------------------------------------------------------------------------

// vistaDeConservacion es un calendario derivado para un perfil concreto.
type vistaDeConservacion struct {
	Perfil string
	Cal    pantalla.Calendario
}

// perfilDeConservacion es el trozo del fichero de perfil que esta ley necesita.
//
// Se decodifica con `json` normal y NO con `nucleo/estricto`, y se dice por que:
// aqui se leen tres campos de un fichero que ya valida su propio cargador en
// cmd/plazum. Lo que si se comprueba es que el fichero tenga hechos, porque un
// perfil vacio produciria un calendario vacio y esta ley estaria midiendo la
// nada con cara de estar midiendo un producto.
type perfilDeConservacion struct {
	ID     string `json:"id"`
	Hechos []struct {
		Pred string   `json:"pred"`
		Args []string `json:"args"`
	} `json:"hechos"`
	Fechas map[string]string `json:"fechas"`
	Bandas []struct {
		Desde  int `json:"desde"`
		Hechos []struct {
			Pred string   `json:"pred"`
			Args []string `json:"args"`
		} `json:"hechos"`
	} `json:"bandas"`
}

const sujetoDeConservacion = "perfil"

func consvCargarPerfiles(t *testing.T) []perfilDeConservacion {
	t.Helper()
	entradas, err := fs.ReadDir(perfiles.Ficheros, ".")
	if err != nil {
		t.Fatalf("no puedo listar los perfiles empotrados: %v", err)
	}
	var out []perfilDeConservacion
	for _, e := range entradas {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		b, err := perfiles.Ficheros.ReadFile(e.Name())
		if err != nil {
			t.Fatalf("no puedo leer el perfil %s: %v", e.Name(), err)
		}
		var p perfilDeConservacion
		if err := json.Unmarshal(b, &p); err != nil {
			t.Fatalf("el perfil %s no decodifica: %v", e.Name(), err)
		}
		if p.ID == "" || len(p.Hechos) == 0 {
			t.Fatalf("el perfil %s no trae id o no trae hechos: su calendario saldria vacio "+
				"y esta ley estaria midiendo el vacio", e.Name())
		}
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	if len(out) == 0 {
		t.Fatal("no hay ni un perfil empotrado, asi que no hay ni una vista que mirar")
	}
	return out
}

// consvFecha resuelve una fecha sembrada por un perfil.
//
// LAS TRES FORMAS DE LA NADA, y la tercera es la que importa (invariante 8): un
// valor PRESENTE Y NO INTERPRETABLE no se convierte en el cero, se convierte en
// un fallo. Un hecho de arranque en el ano 1 haria que una cadencia anual
// produjera dos mil vencimientos pasados y la lista de vencidos se llenaria de
// acusaciones nacidas de un error de lectura.
func consvFecha(v string, ahora time.Time) (time.Time, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return time.Time{}, fmt.Errorf("esta vacia")
	}
	if v[0] == '-' || v[0] == '+' {
		signo := 1
		if v[0] == '-' {
			signo = -1
		}
		cuerpo := v[1:]
		if len(cuerpo) < 2 || cuerpo[len(cuerpo)-1] != 'd' {
			return time.Time{}, fmt.Errorf("%q no es un desplazamiento en dias como -45d", v)
		}
		n, err := strconv.Atoi(cuerpo[:len(cuerpo)-1])
		if err != nil {
			return time.Time{}, fmt.Errorf("%q no lleva un numero de dias dentro", v)
		}
		return ahora.AddDate(0, 0, signo*n), nil
	}
	t, err := time.Parse("2006-01-02", v)
	if err != nil {
		return time.Time{}, fmt.Errorf("%q ni es AAAA-MM-DD ni un desplazamiento como -45d", v)
	}
	return t.UTC(), nil
}

// consvAplicables monta el motor con los programas del corpus, afirma los hechos
// del perfil y devuelve las obligaciones que ese perfil alcanza.
func consvAplicables(t *testing.T, ps []*corpus.Paquete, p perfilDeConservacion) map[string]bool {
	t.Helper()
	m := aplicabilidad.NuevoMotor()
	conReglas := 0
	for _, pq := range ps {
		if len(pq.Aplicabilidad.Reglas) == 0 {
			continue
		}
		prog, errs := pq.Programa()
		if len(errs) != 0 {
			t.Fatalf("%s: el paquete no produce un programa valido: %v", pq.URN, errs)
		}
		if err := m.Cargar(prog); err != nil {
			t.Fatalf("%s: el motor rechaza el programa: %v", pq.URN, err)
		}
		conReglas++
	}
	if conReglas == 0 {
		t.Fatal("ningun paquete declara reglas de aplicabilidad: sin reglas, todas las vistas " +
			"serian identicas y esta ley no distinguiria nada")
	}

	afirmar := func(pred string, args []string) {
		a := make([]string, len(args))
		for i, x := range args {
			if x == "$sujeto" {
				x = sujetoDeConservacion
			}
			a[i] = x
		}
		h := aplicabilidad.H(pred, a...)
		h.Procedencia = "perfil de arranque " + p.ID
		m.Afirmar(h)
	}
	for _, h := range p.Hechos {
		afirmar(h.Pred, h.Args)
	}
	for _, b := range p.Bandas {
		if plantillaDeLaConservacion >= b.Desde {
			for _, h := range b.Hechos {
				afirmar(h.Pred, h.Args)
			}
		}
	}
	if _, err := m.Evaluar(); err != nil {
		t.Fatalf("el motor no evalua los hechos del perfil %s: %v", p.ID, err)
	}
	out := map[string]bool{}
	for _, h := range m.Consultar(aplicabilidad.A("aplica",
		aplicabilidad.V("O"), aplicabilidad.C(sujetoDeConservacion))) {
		out[h.Args[0]] = true
	}
	return out
}

// consvVistas deriva un calendario por perfil publicado.
func consvVistas(t *testing.T, ps []*corpus.Paquete) []vistaDeConservacion {
	t.Helper()
	var out []vistaDeConservacion
	for _, p := range consvCargarPerfiles(t) {
		hechos := ventana.Hechos{}
		for clave, valor := range p.Fechas {
			f, err := consvFecha(valor, instanteDeLaConservacion)
			if err != nil {
				t.Fatalf("el perfil %s siembra la fecha %q con el valor %q y no se entiende: %v.\n"+
					"  Un valor presente y no interpretable NO es la nada: tomarlo por el cero "+
					"inventaria un hecho de arranque en el ano 1", p.ID, clave, valor, err)
			}
			hechos[clave] = f
		}
		alcanzadas := consvAplicables(t, ps, p)
		aplica := func(id string) (bool, bool) { return alcanzadas[id], true }
		out = append(out, vistaDeConservacion{
			Perfil: p.ID,
			Cal:    pantalla.Derivar12Meses(ps, aplica, hechos, instanteDeLaConservacion),
		})
	}
	return out
}

// -----------------------------------------------------------------------------
// El censo.
// -----------------------------------------------------------------------------

// consvRelojesDelCorpus enumera el universo DEL ARBOL: toda obligacion con
// temporalidad de todo paquete instalado, con el marco al que pertenece.
//
// Comprueba de paso la unicidad global del ID, que es la condicion de la que
// depende el emparejamiento con `Calendario.Destinos` (invariante 7).
func consvRelojesDelCorpus(t *testing.T, ps []*corpus.Paquete) map[string]string {
	t.Helper()
	out := map[string]string{}
	for _, p := range ps {
		for _, o := range p.Obligaciones {
			if o.Temporalidad == nil {
				continue
			}
			if otro, ya := out[o.ID]; ya {
				t.Fatalf(`la obligacion %q sale en %s y en %s con el MISMO id.

  Calendario.Destinos es un mapa plano por id sobre el corpus entero, asi que dos
  obligaciones con el mismo id se pisan: una recibe destino y la otra desaparece
  sin que ninguna suma se rompa. Es el emparejamiento de esta ley y deja de valer.`,
					o.ID, otro, p.URN)
			}
			out[o.ID] = p.URN
		}
	}
	return out
}

// consvVeredicto decide el veredicto de una obligacion mirando TODAS las vistas.
//
// LA PRECEDENCIA ESTA ESCRITA Y TIENE MOTIVO. Una obligacion puede caer en cubos
// distintos segun el perfil: la que un perfil no alcanza puede ser, para otro,
// una cuyas fechas caen mas alla de la ventana. Gana la respuesta MAS ESPECIFICA
// SOBRE TI, que es la que dice algo del sujeto y no solo del corpus:
//
//	se ve                     cualquier vista que lo ensene manda sobre todas
//	mas alla de la ventana    alguien lo alcanza; solo sus fechas quedan lejos
//	ningun perfil lo alcanza  nadie lo alcanza
//	empieza despues / ya ceso / ilegible   son hechos del calendario, iguales en
//	                          todas las vistas, y por eso van al final: si
//	                          aparecen es porque no habia nada mas que decir
func consvVeredicto(destinos []pantalla.Destino) veredictoDelReloj {
	visto := map[pantalla.Destino]bool{}
	for _, d := range destinos {
		if d.EsVisible() {
			return relojSeVe
		}
		visto[d] = true
	}
	for _, par := range []struct {
		d pantalla.Destino
		v veredictoDelReloj
	}{
		{pantalla.DestinoMasAllaDeLaVentana, relojTodosSusVencimientosMasAlla},
		{pantalla.DestinoNoTeAlcanza, relojNingunPerfilLoAlcanza},
		{pantalla.DestinoEmpiezaDespues, relojEmpiezaDespuesDeLaVentana},
		{pantalla.DestinoYaCeso, relojDejoDeObligarAntes},
		{pantalla.DestinoVigenciaIlegible, relojVigenciaIlegible},
	} {
		if visto[par.d] {
			return par.v
		}
	}
	return relojSinDeclarar
}

// censoDelCalendario es el reparto entero, y es un DATO, no una afirmacion.
type censoDelCalendario struct {
	// Veredictos: el cubo de cada reloj, por id de obligacion.
	Veredictos map[string]veredictoDelReloj
	// QuienLoVe dice QUE perfiles ensenan cada reloj. No es adorno: es lo que
	// convierte «se ve» en algo comprobable por una persona con el producto
	// delante, y lo que hace util el mensaje de la puerta cuando salta.
	QuienLoVe map[string][]string
	// Huerfanos son los relojes que una vista no etiqueto: ni fila, ni motivo,
	// ni cubo con nombre.
	Huerfanos []string
}

// consvCenso reparte todo reloj del corpus en su veredicto.
//
// NO RECIBE EL *testing.T Y NO AFIRMA NADA: devuelve el reparto. Es lo mismo que
// hace `pantalla.RelojesSinDestino`, y por el mismo motivo: asi el control
// negativo pasa por ESTE codigo y no por una copia escrita en el test. Un
// detector que solo se ha ejecutado contra el caso bueno no ha demostrado que
// sepa decir que no.
func consvCenso(relojes map[string]string, vistas []vistaDeConservacion) censoDelCalendario {
	c := censoDelCalendario{
		Veredictos: map[string]veredictoDelReloj{},
		QuienLoVe:  map[string][]string{},
	}
	for id := range relojes {
		var ds []pantalla.Destino
		for _, v := range vistas {
			d, ok := v.Cal.Destinos[id]
			if !ok {
				c.Huerfanos = append(c.Huerfanos,
					id+" ("+relojes[id]+") no recibe destino en la vista "+v.Perfil)
				continue
			}
			ds = append(ds, d)
			if d.EsVisible() {
				c.QuienLoVe[id] = append(c.QuienLoVe[id], v.Perfil)
			}
		}
		c.Veredictos[id] = consvVeredicto(ds)
	}
	sort.Strings(c.Huerfanos)
	return c
}

// consvCuenta cuenta los cubos.
func consvCuenta(veredictos map[string]veredictoDelReloj) map[veredictoDelReloj]int {
	out := map[veredictoDelReloj]int{}
	// LOS CUBOS VACIOS TAMBIEN SALEN, igual que en `accesos.EstadosPosibles`: un
	// cubo que solo aparece cuando tiene algo dentro es un cubo que nadie echa
	// de menos el dia que se vacia.
	for _, v := range veredictosConocidos {
		out[v] = 0
	}
	for _, v := range veredictos {
		out[v]++
	}
	return out
}

// consvCensoParaPegar imprime el censo listo para copiar a CensoEsperado.
func consvCensoParaPegar(cuenta map[veredictoDelReloj]int) string {
	var b strings.Builder
	b.WriteString("var CensoEsperado = map[veredictoDelReloj]int{\n")
	for _, v := range veredictosConocidos {
		fmt.Fprintf(&b, "\t%s: %d,\n", consvNombreEnGo(v), cuenta[v])
	}
	b.WriteString("}\n")
	return b.String()
}

func consvNombreEnGo(v veredictoDelReloj) string {
	switch v {
	case relojSeVe:
		return "relojSeVe"
	case relojNingunPerfilLoAlcanza:
		return "relojNingunPerfilLoAlcanza"
	case relojTodosSusVencimientosMasAlla:
		return "relojTodosSusVencimientosMasAlla"
	case relojEmpiezaDespuesDeLaVentana:
		return "relojEmpiezaDespuesDeLaVentana"
	case relojDejoDeObligarAntes:
		return "relojDejoDeObligarAntes"
	case relojVigenciaIlegible:
		return "relojVigenciaIlegible"
	default:
		return "relojSinDeclarar"
	}
}

// -----------------------------------------------------------------------------
// La puerta.
// -----------------------------------------------------------------------------

func TestNingunRelojDelCorpusDesapareceEnSilencio(t *testing.T) {
	ps, err := corpus.Cargar("paquetes")
	if err != nil {
		t.Fatalf("no puedo cargar el corpus publicado: %v", err)
	}
	vistas := consvVistas(t, ps)
	censo := consvCenso(consvRelojesDelCorpus(t, ps), vistas)
	veredictos, quienLoVe := censo.Veredictos, censo.QuienLoVe
	cuenta := consvCuenta(veredictos)

	if len(censo.Huerfanos) > 0 {
		t.Errorf(`%d relojes no reciben destino en alguna vista: %v

  Ni fila, ni motivo, ni cubo con nombre: desaparecen. Para quien lee el
  calendario es indistinguible de que no existieran.`,
			len(censo.Huerfanos), consvMuestra(censo.Huerfanos))
	}

	// SUELO: sin el, un corpus que dejara de cargar relojes daria verde sin
	// haber comprobado ni uno, y un perfil roto daria verde sin una vista.
	if len(veredictos) < 100 {
		t.Fatalf("solo %d obligaciones con reloj en el corpus: esta ley estaria comprobando "+
			"el vacio", len(veredictos))
	}
	if len(vistas) < 3 {
		t.Fatalf("solo %d vistas: los perfiles publicados son tres y esta ley mira "+
			"«¿lo ve alguien?»", len(vistas))
	}

	// EL VALOR CERO ESTA PROHIBIDO. Una obligacion sin veredicto es una que
	// ningun camino etiqueto.
	var sinDeclarar []string
	for id, v := range veredictos {
		if v == relojSinDeclarar {
			sinDeclarar = append(sinDeclarar, id)
		}
	}
	sort.Strings(sinDeclarar)
	if len(sinDeclarar) > 0 {
		t.Errorf(`%d relojes se quedan SIN DECLARAR (valor cero): %v

  El cero no es un estado, es el olvido. O se ve en alguna vista, o esta en una
  categoria de ausencia con su motivo.`, len(sinDeclarar), consvMuestra(sinDeclarar))
	}

	// LOS CUBOS CUADRAN: instaladas = las que se ven + la suma de las ausencias.
	//
	// Se comprueba con `metrica.Cuadra` y no a mano. La razon es la de ese
	// paquete: una guarda de conservacion escrita a mano en cada sitio deja
	// fuera todos los demas, y ademas la de aqui nacio sin distinguir el
	// descuadre CON SIGNO, que es lo que dice si algo no sale en ninguna vista
	// (suman de menos) o si algo sale dos veces (suman de mas).
	partes := map[string]int{}
	for _, v := range veredictosConocidos {
		partes[v.String()] = cuenta[v]
	}
	if err := metrica.Cuadra(len(veredictos), "obligaciones con reloj instaladas", partes); err != nil {
		t.Errorf("%v\n\n  Es un fallo del producto, no del corpus: hay un reloj que cae "+
			"fuera del vocabulario cerrado.", err)
	}

	// LOS QUE SE VEN, POR NOMBRE Y EN LOS DOS SENTIDOS.
	//
	// Es la mitad que el cardinal solo no puede dar: dos movimientos opuestos en
	// la misma pasada (uno se cae, otro entra) dejan todos los numeros iguales.
	// Aqui no se cancelan, y ademas el rojo dice CUAL.
	declarados := map[string]bool{}
	for _, id := range RelojesQueSeVenEsperados {
		declarados[id] = true
	}
	var dejaronDeVerse, empezaronAVerse []string
	for _, id := range RelojesQueSeVenEsperados {
		if veredictos[id] != relojSeVe {
			dejaronDeVerse = append(dejaronDeVerse,
				id+" (ahora: "+veredictos[id].String()+")")
		}
	}
	for id, v := range veredictos {
		if v == relojSeVe && !declarados[id] {
			empezaronAVerse = append(empezaronAVerse, id)
		}
	}
	sort.Strings(dejaronDeVerse)
	sort.Strings(empezaronAVerse)
	if len(dejaronDeVerse) > 0 {
		t.Errorf(`%d relojes DEJAN DE VERSE en el calendario de todos los perfiles:

  %s

  Esto es el fallo del art. 14.6 otra vez: la obligacion sigue en el corpus, la
  contabilidad sigue cuadrando y el CISO ya no la ve. Si el cambio es correcto
  (la norma movio su vigencia de verdad), se baja de la lista Y SE DICE en el
  commit cual y por que. Bajarla sin decirlo es dejar el silencio puesto.`,
			len(dejaronDeVerse), strings.Join(dejaronDeVerse, "\n  "))
	}
	if len(empezaronAVerse) > 0 {
		t.Errorf(`%d relojes se ven y no estaban declarados: %v

  Puede ser una buena noticia (corpus nuevo que alcanza a un perfil) y hay que
  anadirlos. El sentido contrario no sobra: sin el, esta lista se convierte poco
  a poco en una lista de lo que hubo.`, len(empezaronAVerse), consvMuestra(empezaronAVerse))
	}

	// EL CARDINAL, CON IGUALDAD EXACTA Y EN LOS DOS SENTIDOS.
	descuadres := 0
	for _, v := range veredictosConocidos {
		esperado, declarado := CensoEsperado[v]
		if !declarado {
			t.Errorf("el veredicto %q no tiene cardinal declarado en CensoEsperado: un cubo "+
				"sin numero es un cubo que se vacia sin que nadie se entere", v)
			descuadres++
			continue
		}
		if cuenta[v] != esperado {
			descuadres++
			t.Errorf("el cubo %q trae %d y CensoEsperado dice %d", v, cuenta[v], esperado)
		}
	}
	// SENTIDO 2: nada declarado que ya no exista en el vocabulario.
	enVocabulario := map[veredictoDelReloj]bool{}
	for _, v := range veredictosConocidos {
		enVocabulario[v] = true
	}
	for v := range CensoEsperado {
		if !enVocabulario[v] {
			t.Errorf("CensoEsperado declara el veredicto %q, que ya no esta en el vocabulario "+
				"cerrado: la declaracion se ha quedado vieja", v)
			descuadres++
		}
	}

	if descuadres > 0 {
		t.Errorf(`EL CENSO DEL CALENDARIO HA CAMBIADO.

  Antes de pegar los numeros nuevos, la pregunta que da sentido a esta puerta:
  ¿ha BAJADO el cubo de los que se ven? Si ha bajado, un reloj que un cliente
  veia ha dejado de verse, y eso no se arregla bajando el numero. El 03-09-2026,
  mover quince meses la vigencia del art. 14.6 del CRA hizo exactamente eso y no
  puso roja ni una puerta.

  El censo de hoy, listo para pegar:

%s`, consvCensoParaPegar(cuenta))
	}

	if !t.Failed() {
		// SE DICE EN VOZ ALTA lo que se ha recorrido. Un verde sin cifra al lado
		// no distingue «ha mirado el corpus entero» de «no ha mirado nada».
		t.Logf("%d relojes instalados, %d vistas, %d se ven en alguna",
			len(veredictos), len(vistas), cuenta[relojSeVe])
		var conVista []string
		for id, ps := range quienLoVe {
			conVista = append(conVista, id+" -> "+strings.Join(ps, ", "))
		}
		sort.Strings(conVista)
		t.Logf("los que se ven:\n  %s", strings.Join(conVista, "\n  "))
		t.Logf("censo:\n%s", consvCensoParaPegar(cuenta))
	}
}

func consvMuestra(s []string) []string {
	if len(s) > 8 {
		return s[:8]
	}
	return s
}

// TODA RAMA LLEVA SU CONTROL POSITIVO, con dato sintetico donde el corpus no
// llega.
//
// POR QUE HACE FALTA AQUI. Dos cubos del censo estan HOY a cero sobre el corpus
// real (`mas alla de la ventana` y `vigencia ilegible`). Una rama que ninguna
// entrada recorre es una rama que no existe, y una mutacion la deja verde porque
// no hay nada que romper: es el fallo M47, que cambio «en tus respuestas no
// aparece» por «lo has incumplido» sin poner nada rojo.
//
// Se ejercita `consvVeredicto`, que es donde vive la decision, con los conjuntos
// de destinos que produce cada caso. Que la DERIVACION sepa producir cada uno de
// esos destinos ya lo prueba `nucleo/pantalla.TestCadaDestinoTieneSuInquilino`,
// con un corpus sintetico con un inquilino por destino; repetirlo aqui seria una
// segunda copia de aquel y se separaria de el.
func TestCadaVeredictoDelCalendarioTieneSuInquilino(t *testing.T) {
	casos := []struct {
		nombre   string
		destinos []pantalla.Destino
		quiero   veredictoDelReloj
	}{
		{"lo ve un perfil de tres",
			[]pantalla.Destino{pantalla.DestinoNoTeAlcanza, pantalla.DestinoConFecha,
				pantalla.DestinoNoTeAlcanza}, relojSeVe},
		{"solo esta vencida en una vista, y una vencida se ve",
			[]pantalla.Destino{pantalla.DestinoVencida}, relojSeVe},
		{"sin fecha con motivo tambien se ve",
			[]pantalla.Destino{pantalla.DestinoSinFecha}, relojSeVe},
		{"un estreno se ve",
			[]pantalla.Destino{pantalla.DestinoEstrena}, relojSeVe},
		{"ningun perfil lo alcanza",
			[]pantalla.Destino{pantalla.DestinoNoTeAlcanza, pantalla.DestinoNoTeAlcanza},
			relojNingunPerfilLoAlcanza},
		{"uno lo alcanza y sus fechas caen lejos: gana la respuesta sobre ti",
			[]pantalla.Destino{pantalla.DestinoNoTeAlcanza, pantalla.DestinoMasAllaDeLaVentana},
			relojTodosSusVencimientosMasAlla},
		{"empieza despues de la ventana",
			[]pantalla.Destino{pantalla.DestinoEmpiezaDespues, pantalla.DestinoEmpiezaDespues},
			relojEmpiezaDespuesDeLaVentana},
		{"dejo de obligar antes",
			[]pantalla.Destino{pantalla.DestinoYaCeso}, relojDejoDeObligarAntes},
		{"vigencia ilegible",
			[]pantalla.Destino{pantalla.DestinoVigenciaIlegible}, relojVigenciaIlegible},
		// EL VALOR CERO, que es el que sale por olvido y el que hay que ver
		// fallar (invariante 8). Las DOS formas de la nada: la lista nil y la
		// lista presente y vacia. No son la misma cosa y la que se olvida es la
		// primera, porque es la que sale sin escribir nada.
		{"sin ni un destino (nil)", nil, relojSinDeclarar},
		{"sin ni un destino (vacia y presente)", []pantalla.Destino{}, relojSinDeclarar},
	}
	recorridos := map[veredictoDelReloj]bool{}
	for _, c := range casos {
		if got := consvVeredicto(c.destinos); got != c.quiero {
			t.Errorf("%s: el veredicto es %q y tenia que ser %q", c.nombre, got, c.quiero)
			continue
		}
		recorridos[c.quiero] = true
	}
	for _, v := range veredictosConocidos {
		if !recorridos[v] {
			t.Errorf(`el veredicto %q no lo recorre ningun caso.

  Una rama que ninguna entrada recorre es una rama que no existe, y la mutacion
  la deja verde porque no hay nada que romper. Si este veredicto ya no puede
  darse, se quita del vocabulario; si puede, aqui va su caso.`, v)
		}
	}
	if !recorridos[relojSinDeclarar] {
		t.Error("el valor cero no lo recorre ningun caso: es el unico que sale por olvido")
	}
}

// CONTROL NEGATIVO: el emparejamiento sabe decir que NO.
//
// El verde de la puerta de arriba podria venir de un recorrido que no mira nada.
// Se le da un calendario al que le falta a proposito la etiqueta de un reloj y
// se exige que el mismo recorrido la eche en falta, y que el reloj que si esta
// no salte.
func TestLaLeyDelCalendarioCazaUnRelojQueDesaparece(t *testing.T) {
	conReloj := func(id string) corpus.Obligacion {
		return corpus.Obligacion{ID: id, Temporalidad: &corpus.Temporalidad{Primitiva: "plazo"}}
	}
	ps := []*corpus.Paquete{{
		URN: "urn:demo:conservacion", Version: "1", Clase: corpus.Transcrito,
		Obligaciones: []corpus.Obligacion{
			conReloj("k.se_ve"), conReloj("k.desaparece"),
			// Sin temporalidad: no es un reloj y no entra en el universo. Es la
			// rama que separa «no tiene reloj» de «tiene reloj y se ha perdido».
			{ID: "k.sin_reloj"},
		},
	}}

	completa := vistaDeConservacion{Perfil: "sintetico", Cal: pantalla.Calendario{
		Destinos: map[string]pantalla.Destino{
			"k.se_ve":       pantalla.DestinoConFecha,
			"k.desaparece":  pantalla.DestinoNoTeAlcanza,
			"k.sin_reloj":   pantalla.DestinoConFecha,
			"k.no_existe":   pantalla.DestinoConFecha,
			"k.tampoco_hay": pantalla.DestinoVencida,
		}}}

	relojes := consvRelojesDelCorpus(t, ps)
	// Y una obligacion SIN reloj no entra en el universo: contarla haria que el
	// censo creciera con puntos que no producen ninguna fecha.
	if _, hay := relojes["k.sin_reloj"]; hay {
		t.Error("k.sin_reloj no tiene temporalidad y ha entrado en el universo: lo que esta " +
			"ley cuenta son los RELOJES, no las obligaciones")
	}

	// RAMA POSITIVA: con todas las etiquetas puestas, no echa nada en falta.
	// Sin esta mitad, el detector podria estar diciendo que si a todo.
	bueno := consvCenso(relojes, []vistaDeConservacion{completa})
	if len(bueno.Huerfanos) > 0 {
		t.Fatalf("con el calendario completo no tenia que echar nada en falta y dijo %v",
			bueno.Huerfanos)
	}
	if bueno.Veredictos["k.desaparece"] != relojNingunPerfilLoAlcanza {
		t.Errorf("k.desaparece esta etiquetada como no alcanzada y salio %q",
			bueno.Veredictos["k.desaparece"])
	}

	// RAMA NEGATIVA: se quita una etiqueta y tiene que saltar.
	rota := vistaDeConservacion{Perfil: "sintetico", Cal: pantalla.Calendario{
		Destinos: map[string]pantalla.Destino{"k.se_ve": pantalla.DestinoConFecha}}}
	malo := consvCenso(relojes, []vistaDeConservacion{rota})
	if len(malo.Huerfanos) != 1 || !strings.Contains(malo.Huerfanos[0], "k.desaparece") {
		t.Fatalf("quitada la etiqueta de k.desaparece, el recorrido tenia que echarla en "+
			"falta y devolvio %v: sin esto, el verde de la puerta no demuestra que mire",
			malo.Huerfanos)
	}
	// Y el que desaparece cae en el valor cero, que es lo que la puerta rechaza.
	if malo.Veredictos["k.desaparece"] != relojSinDeclarar {
		t.Errorf("k.desaparece tenia que quedarse SIN DECLARAR y salio %q",
			malo.Veredictos["k.desaparece"])
	}
	if malo.Veredictos["k.se_ve"] != relojSeVe {
		t.Errorf("k.se_ve tenia que seguir viendose y salio %q", malo.Veredictos["k.se_ve"])
	}
}
