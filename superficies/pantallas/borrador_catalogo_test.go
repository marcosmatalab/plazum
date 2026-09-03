package pantallas

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Un BORRADOR de catalogo en espanol, y para que sirve.
//
// Esto NO es el catalogo del producto: el catalogo lo escribe el frente de
// i18n, detras del puerto puertos.Catalogo, y este fichero es un _test.go que
// no entra en el binario. Existe por dos razones concretas:
//
//	prueba que ClavesDeCatalogo() se puede rellenar entera y que el resultado
//	se lee como una interfaz de verdad. Una lista de claves que nadie ha
//	intentado traducir suele tener claves imposibles de redactar ("el titulo
//	de la cosa"), y es mucho mejor descubrirlo aqui.
//	deja escrito el borrador para quien haga el catalogo de verdad, con el
//	tono y el nivel de detalle que estas pantallas necesitan.
//
// Y ademas se puede volcar a fichero para leer las pantallas como las lee un
// comprador que abre esto a las nueve de la manana:
//
//	PLAZUM_VOLCAR=/algun/directorio go test ./superficies/pantallas -run TestVolcar

type catEs struct{ textos map[string]string }

func (c catEs) Traducir(_, clave string, args ...any) string {
	t, ok := c.textos[clave]
	if !ok {
		return "FALTA(" + clave + ")"
	}
	if len(args) > 0 && strings.Contains(t, "%") {
		return fmt.Sprintf(t, args...)
	}
	return t
}
func (c catEs) Idiomas() []string         { return []string{"es"} }
func (c catEs) Faltantes(string) []string { return nil }

var textoEs = map[string]string{
	"ui.marca":                         "plazum",
	"ui.saltar":                        "Ir al contenido",
	"ui.navegacion":                    "Pantallas",
	"ui.pie.no_asesoramiento":          "plazum no presta asesoramiento juridico. Lo que ves aqui es lo que dicen los paquetes normativos que tienes instalados, con su cita, para que puedas comprobarlo tu.",
	"pantalla.alcance.titulo":          "Alcance",
	"pantalla.hoy.titulo":              "Hoy",
	"pantalla.controles.titulo":        "Controles",
	"pantalla.certificados.titulo":     "Certificados",
	"pantalla.personas.titulo":         "Personas",
	"pantalla.estado.titulo":           "Estado",
	"pantalla.error.titulo":            "No hemos podido abrir esa pagina",
	"pantalla.alcance.sin_corpus":      "No hay ningun paquete normativo instalado, asi que no hay nada que preguntarte todavia.",
	"pantalla.controles.sin_corpus":    "No hay ningun paquete normativo instalado, asi que no hay obligaciones que listar.",
	"pantalla.certificados.sin_corpus": "No hay ningun paquete normativo instalado, asi que no hay entregables que preparar.",
	"pantalla.hoy.vacia":               "Aqui apareceran los plazos que vencen y lo que toca hacer esta semana.",
	"pantalla.personas.vacia":          "Aqui apareceran las personas responsables de cada obligacion.",
	"pantalla.estado.vacia":            "Aqui apareceran el estado del expediente, las actualizaciones y el pulso del sistema.",
	"menu.aplican":                     "%d aplican",
	"menu.vacia":                       "sin contenido",
	// La tira del camino guiado en la barra lateral. Los rotulos de los pasos
	// son los que declara superficies/camino: aqui se redactan igual que en su
	// pantalla, porque son el mismo paso visto desde otro sitio.
	"camino.titulo":          "Camino guiado",
	"camino.paso.alcance":    "Alcance",
	"camino.paso.calendario": "Calendario",
	"camino.paso.derivacion": "Derivacion",
	"camino.paso.acta":       "Revision por la direccion",
	"camino.paso.uar":        "Revision de accesos",
	"camino.paso.escalado":   "Escalado",
	"ui.aqui":                "estas aqui",
	"ui.paso_por_terminal":   "por terminal",
	// El panel de inicio.
	"pantalla.hoy.cifra.vence_semana":      "vence en los proximos siete dias",
	"pantalla.hoy.cifra.sin_constancia":    "vencimientos pasados de los que no consta nada",
	"pantalla.hoy.cifra.te_alcanzan":       "obligaciones que te alcanzan",
	"pantalla.hoy.cifra.marcos":            "marcos instalados",
	"pantalla.hoy.cifra.sin_dato":          "sin dato",
	"pantalla.hoy.cifra.sin_corpus":        "No hay ningun paquete normativo instalado, asi que no hay nada que contar.",
	"pantalla.hoy.cifra.sin_responder":     "Todavia no has respondido nada, asi que aqui solo estan las que alcanzan a todo el mundo.",
	"pantalla.hoy.cifra.derivacion":        "Ver de donde sale",
	"pantalla.hoy.vence_semana.esperando":  "Y hay %d relojes esperando un dato tuyo, asi que todavia no tienen fecha.",
	"pantalla.hoy.vence_semana.vacio":      "En los proximos siete dias no vence nada de lo que te alcanza.",
	"pantalla.hoy.sin_constancia.descargo": "Esto NO dice que se haya incumplido: dice que en tus respuestas no consta que se hiciera.",
	"pantalla.hoy.sin_constancia.vacio":    "No hay ningun vencimiento pasado del que falte constancia.",
	"pantalla.hoy.vencida.desde":           "desde el %s, %d ciclos",
	"pantalla.hoy.marcos.caption":          "Los marcos instalados y cuantas obligaciones aporta cada uno",
	"pantalla.hoy.ir_alcance":              "Responder mas preguntas",
	"origen.corpus":                        "Esta pantalla se construye con los paquetes normativos instalados.",
	"origen.estado":                        "Esta pantalla se construye con tu expediente y con el reloj legal.",
	"vacia.que_hacer":                      "Empieza por Alcance, responde la entrevista y vuelve aqui.",
	"vacia.sin_explicacion":                "Esta pantalla no tiene contenido y no sabemos decirte por que. Es un fallo nuestro, cuentanoslo.",
	"vacia.volver_alcance":                 "Ir a Alcance",
	"alcance.intro":                        "Responde estas preguntas y veras al momento que obligaciones te alcanzan y por que articulo. Empieza por la primera, es la que mas desbloquea.",
	"alcance.progreso":                     "Has respondido %d de %d preguntas",
	"alcance.siguiente":                    "Empieza por esta",
	"alcance.sin_preguntas":                "Los paquetes que tienes instalados no hacen ninguna pregunta de alcance, asi que sus obligaciones te alcanzan sin condiciones.",
	"alcance.pregunta.si":                  "Si",
	"alcance.pregunta.no":                  "No",
	"alcance.pregunta.limpiar":             "Deshacer",
	"alcance.pregunta.desbloquea":          "decide %d obligaciones",
	"alcance.pregunta.la_pide":             "lo pregunta %s",
	"alcance.pregunta.contradictoria":      "Esta pregunta llega respondida que si y que no a la vez, asi que la damos por sin responder.",
	"alcance.pregunta.respondida_si":       "Has respondido que si.",
	"alcance.pregunta.respondida_no":       "Has respondido que no.",
	// LA REVELACION PROGRESIVA. El tono importa aqui mas que en ninguna otra
	// familia: lo que se dice de una pregunta dormida no puede sonar a que el
	// operador ha hecho algo mal, porque no ha hecho nada. Es un dato que
	// falta en el paquete normativo, y se dice asi.
	"alcance.dormidas.titulo":                 "%d pregunta no decide nada todavia|%d preguntas no deciden nada todavia",
	"alcance.dormidas.ver":                    "Verla igualmente|Verlas igualmente",
	"alcance.dormidas.volver":                 "Volver a las que deciden",
	"alcance.dormidas.porque":                 "Estas son todas las preguntas de tus paquetes. Las marcadas no cambian hoy ninguna obligacion, asi que la lista corta las deja fuera para que llegues antes al calendario. Puedes responderlas igual: no se pierde nada.",
	"alcance.dormidas.nadie_la_pide":          "Ninguna obligacion de tus paquetes dice depender de esta pregunta, asi que responderla no mueve nada. Es un hueco del paquete normativo, no tuyo.",
	"alcance.dormidas.ya_decidida":            "Las obligaciones que dependian de esta ya han quedado decididas por lo que respondiste antes.",
	"alcance.derivacion.titulo":               "Lo que te aplica ahora mismo",
	"alcance.derivacion.sin_respuestas":       "Todavia no has respondido nada. Solo se listan las obligaciones que alcanzan a todo el mundo.",
	"alcance.derivacion.no_es_dictamen":       "Esto es un avance de alcance a partir de lo que has respondido, no un dictamen juridico.",
	"alcance.derivacion.no_guardado":          "Tus respuestas viajan en la direccion de esta pagina y todavia no se guardan en ningun sitio. Guarda el enlace o compartelo si quieres volver.",
	"alcance.derivacion.aplican":              "Te aplican %d",
	"alcance.derivacion.y_mas":                "y %d mas, en Controles",
	"alcance.derivacion.proximas":             "Si respondes la de arriba, decides estas %d",
	"alcance.derivacion.ver_controles":        "Ver todas en Controles",
	"alcance.derivacion.limpiar":              "Empezar de cero",
	"alcance.campos.titulo":                   "Los datos que vas a necesitar",
	"alcance.campos.intro":                    "Esto es lo que piden los paquetes instalados. Un dato que piden tres normas se pregunta una vez.",
	"alcance.campos.obligatorio":              "obligatorio",
	"alcance.campos.lo_piden":                 "lo piden",
	"derivacion.sin_condiciones":              "Te aplica sin condiciones",
	"derivacion.respondiste_si":               "Respondiste que si a",
	"derivacion.respondiste_no":               "Respondiste que no a",
	"derivacion.sin_responder":                "Falta responder",
	"derivacion.respuesta_contradictoria":     "Respuesta contradictoria en",
	"derivacion.pregunta_desconocida":         "El paquete la condiciona a una pregunta que el paquete no trae",
	"derivacion.entregable_huerfano":          "Ninguna obligacion pide este documento",
	"derivacion.lo_pide_y_aplica":             "Lo pide, y te aplica,",
	"derivacion.lo_pide_y_no_aplica":          "Lo pide, y no te aplica,",
	"derivacion.lo_pide":                      "Lo pide",
	"estado.aplica":                           "Te aplica",
	"estado.no_aplica":                        "No te aplica",
	"estado.pendiente":                        "Sin decidir",
	"filtro.etiqueta":                         "Filtrar por estado",
	"filtro.todos":                            "Todas",
	"tabla.intro.controles":                   "Todo lo que piden los paquetes instalados, con el estado que sale de lo que has respondido en Alcance.",
	"tabla.intro.certificados":                "Los documentos que hay que entregar, y de que obligacion cuelga cada uno.",
	"tabla.mostrando":                         "De la %d a la %d, de %d",
	"tabla.sin_resultados":                    "No hay ninguna fila con ese filtro.",
	"tabla.anterior":                          "Anterior",
	"tabla.siguiente":                         "Siguiente",
	"tabla.volver_alcance":                    "Volver a Alcance",
	"columna.id":                              "Identificador",
	"columna.paquete":                         "Norma",
	"columna.estado":                          "Estado",
	"columna.porque":                          "Por que",
	"columna.articulo":                        "Articulo",
	"columna.titulo":                          "Titulo",
	"columna.cita":                            "Cita",
	"columna.clase_e2e":                       "Como se implanta",
	"columna.primitiva":                       "Reloj",
	"columna.cadencia":                        "Cada",
	"columna.limite":                          "Plazo",
	"columna.entregable":                      "Entregable",
	"pantalla.hoy.planificador":               "El planificador",
	"pantalla.hoy.canal":                      "El pulso de vida",
	"pantalla.hoy.plazos":                     "Lo que vence",
	"aviso.planificador.late":                 "El planificador esta vivo. Su ultimo ciclo termino hace %d hora.|El planificador esta vivo. Su ultimo ciclo termino hace %d horas.",
	"aviso.planificador.callado":              "Tu planificador lleva %d hora sin correr un ciclo. Mientras siga parado, los plazos vencen sin que nadie los mire.|Tu planificador lleva %d horas sin correr un ciclo. Mientras siga parado, los plazos vencen sin que nadie los mire.",
	"aviso.planificador.nunca":                "El planificador no ha corrido ningun ciclo todavia. Si acabas de instalar plazum es lo normal, si lleva dias instalado no lo es.",
	"aviso.planificador.futuro":               "La ultima marca del planificador esta en el futuro, asi que no podemos decirte si esta vivo. Suele ser el reloj de la maquina, no plazum.",
	"aviso.planificador.sin_instante":         "No sabemos desde que instante juzgar, asi que no podemos decirte si el planificador esta vivo. Es un fallo nuestro, no tuyo.",
	"aviso.planificador.arregla_callado":      "Corre un ciclo con plazum latido ciclo y revisa el temporizador que lo lanza. Avisamos cuando pasa de %d horas sin correr.",
	"aviso.planificador.arregla_nunca":        "Programa plazum latido ciclo cada hora en tu cron o en un temporizador de systemd. Tienes el ejemplo en docs/latido.md.",
	"aviso.planificador.arregla_futuro":       "Pon en hora el reloj de la maquina y espera al siguiente ciclo.",
	"aviso.planificador.arregla_sin_instante": "Cuentanoslo, esto no tendria que pasar.",
	"aviso.latido.apagado":                    "El pulso esta apagado, que es como viene de fabrica. No sale nada de esta maquina hacia nosotros.",
	"aviso.latido.late":                       "El ultimo pulso salio bien hace %d hora.|El ultimo pulso salio bien hace %d horas.",
	"aviso.latido.callado":                    "El pulso no llega desde hace %d hora.|El pulso no llega desde hace %d horas.",
	"aviso.latido.nunca":                      "El pulso esta encendido y todavia no ha salido ninguno.",
	"aviso.latido.fallo":                      "El ultimo intento de pulso no llego.",
	"aviso.latido.arregla_callado":            "Prueba el canal con plazum latido probar y revisa la salida a internet de esta maquina.",
	"aviso.latido.arregla_nunca":              "Prueba el canal con plazum latido probar.",
	"aviso.latido.arregla_fallo":              "Prueba el canal con plazum latido probar y mira que contesta el destino.",
	"aviso.latido.no_es_tu_planificador":      "Esto es el canal de esta maquina hacia nosotros. Que calle no significa que tus plazos dejen de vigilarse: el aviso de arriba se calcula con tu reloj y sin salir a la red.",
	"error.no_encontrado":                     "Esa direccion no existe en plazum.",
	"error.consulta_larga":                    "La direccion trae demasiados datos. Vuelve a Alcance y responde otra vez.",
}

// El borrador cubre TODAS las claves declaradas, y ninguna de mas. Es la
// segunda comprobacion sobre ClavesDeCatalogo(), independiente de la que hace
// el barrido de las pantallas: si alguien anade una clave, esto se pone rojo y
// obliga a redactarla antes de que salga cruda en la pantalla de un cliente.
func TestElBorradorDeCatalogoCubreTodasLasClaves(t *testing.T) {
	declaradas := map[string]bool{}
	for _, c := range ClavesDeCatalogo() {
		declaradas[c] = true
		if _, ok := textoEs[c]; !ok {
			t.Errorf("la clave %q no tiene texto en el borrador. Redactala aqui: si no se "+
				"puede redactar en una linea, casi siempre es que la clave esta mal pensada", c)
		}
	}
	for c := range textoEs {
		if !declaradas[c] {
			t.Errorf("el borrador redacta %q y la interfaz no la pide nunca", c)
		}
	}
}

// TestVolcar escribe las pantallas a fichero para poder leerlas. Solo a mano.
func TestVolcar(t *testing.T) {
	if os.Getenv("PLAZUM_VOLCAR") == "" {
		t.Skip("solo a mano: PLAZUM_VOLCAR=/un/directorio go test -run TestVolcar")
	}
	s, err := Nuevo(Opciones{Paquetes: corpusDemo(), Catalogo: catEs{textoEs}})
	if err != nil {
		t.Fatal(err)
	}
	dir := os.Getenv("PLAZUM_VOLCAR")
	quitaEtiquetas := regexp.MustCompile(`(?s)<script.*?</script>|<[^>]+>`)
	espacios := regexp.MustCompile(`[ \t]*\n[ \t\n]*`)
	for _, ruta := range []string{"/alcance", "/alcance?si=alfa.q.categoria",
		"/controles?si=alfa.q.categoria", "/certificados?si=alfa.q.categoria", "/hoy",
		"/no-existe"} {
		r := httptest.NewRequest(http.MethodGet, ruta, nil)
		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)
		nombre := strings.NewReplacer("/", "_", "?", "-", "=", "-", ".", "_").Replace(ruta)
		texto := espacios.ReplaceAllString(quitaEtiquetas.ReplaceAllString(w.Body.String(), "\n"), "\n")
		if err := os.WriteFile(filepath.Join(dir, nombre+".txt"), []byte(texto), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, nombre+".html"), w.Body.Bytes(), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}
