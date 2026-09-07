package plazum

import (
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/marcosmatalab/plazum/internal/modulo"
	actaWeb "github.com/marcosmatalab/plazum/superficies/acta"
	calendarioWeb "github.com/marcosmatalab/plazum/superficies/calendario"
	"github.com/marcosmatalab/plazum/superficies/camino"
	escaladoWeb "github.com/marcosmatalab/plazum/superficies/escalado"
	"github.com/marcosmatalab/plazum/superficies/pantallas"
	uarWeb "github.com/marcosmatalab/plazum/superficies/uar"
)

// LA PUERTA: NINGUN TEXTO QUE PLAZUM ESCRIBA LLEGA A UNA PLANTILLA SIN PASAR
// POR EL CATALOGO.
//
// # El defecto que la trae, y por que no lo vio ninguna puerta existente
//
// `estado.Calcular` escribia el porque de cada estado en castellano, y esa
// cadena viajaba cruda hasta la celda de la tabla de controles: la pagina
// inglesa imprimia espanol. Este repositorio tiene varias puertas de i18n y
// NINGUNA podia verlo, porque **todas vigilan el catalogo** —que no le falten
// claves, que no le sobren, que el ingles exista— y aquello no era una clave.
//
// La leccion es la que hay que retener: **la forma de publicar sin traducir en
// este repositorio es no pasar por el catalogo.** Vigilar el catalogo mejor no
// habria servido de nada; lo que faltaba era vigilar la OTRA punta, que es el
// modelo de vista.
//
// # POR QUE VIVE EN LA RAIZ Y NO EN superficies/pantallas
//
// Nacio dentro de `superficies/pantallas` el 07-09-2026 y ahi **cubria una
// superficie de siete**. No era un limite teorico: las dos entradas que contaba
// como deuda (`Fecha.Regla` y `Vencida.Regla`) las pinta tambien
// `superficies/calendario`, en SEIS sitios entre `calendario.html` y
// `cifra.html`, y con CUATRO tipos de vista propios que las envuelven. O sea que
// el techo de igualdad exacta protegia a **uno de los dos consumidores**, y en
// el otro un campo nuevo entraba en silencio.
//
// Al subir a la raiz, la deuda medida pasa de **2 entradas a 6**. El numero no
// creció: creció lo que se estaba mirando.
//
// # Que se afirma aqui, en una frase
//
// Por cada campo de texto que puede llegar a una plantilla, hay escrita una
// PROCEDENCIA. Tres viajan en crudo con motivo (corpus, persona, dato) y una
// dice que lo escribe plazum, y entonces es clave de catalogo. No hay bucket
// para «lo escribe plazum y no se traduce»: eso es el defecto.
//
// La quinta, `DePlazumSinCatalogo`, es la excepcion que confirma lo anterior y
// no estaba prevista: la puerta NACIO ROJA sobre el arbol real y encontro
// campos que ya eran el defecto. No se podia dejar roja ni clasificarlos de
// mentira, asi que se cuentan con techo. Su cardinal, en
// TestLaDeudaDeTextoSinCatalogoTieneTecho.
//
// # Como se mide de verdad, y no de palabra
//
// Un censo en el que cada entrada dijera «esto es una clave, palabra de honor»
// seria la afirmacion acompañada otra vez. Asi que la clasificacion se CONTRASTA
// contra las plantillas, que son quienes de verdad deciden:
//
//	DeCatalogo   el nombre del campo tiene que aparecer como argumento de `t` o
//	             de `targs` en alguna plantilla. Si nadie lo traduce, no es una
//	             clave por mucho que el censo lo diga.
//	las otras    el nombre NO puede aparecer como argumento de `t`. Un dato o
//	             una cita del BOE pasados por el catalogo saldrian como
//	             «FALTA(...)» o, peor, traducidos.
//
// # Lo que esta puerta NO mira, con sus DOS cardinales
//
// **Uno de dentro**: el contraste con las plantillas se hace **por nombre de
// campo**, porque una referencia de plantilla (`{{.Titulo}}`) es relativa al
// contexto del `with` o del `range` que la rodea y resolverla hasta el tipo
// exigiria un comprobador de tipos de plantillas que este repositorio no tiene.
// Cuando dos campos de tipos distintos comparten nombre y NO comparten
// procedencia, el nombre es ambiguo y queda fuera de la mitad negativa. Lo
// cuenta `TestElHuecoDelContrasteDeProcedenciaSeCuenta`.
//
// **Y uno de fuera, que es el que faltaba**: el censo cubre hoy los campos de
// `superficies/pantallas` y las cuatro `Regla` de `superficies/calendario`, y NO
// los de las otras cinco superficies. Extenderlo entero costaba declarar 148
// campos mas de una sentada, que es como se escriben 148 justificaciones que
// nadie lee. Se cuenta la deuda con igualdad exacta y por superficie
// (`TestLaDeudaDeCamposSinDeclararTieneTechoPorSuperficie`), para que **no pueda
// crecer callada** y se pueda pagar superficie a superficie.
type DeDonde uint8

const (
	// SinDeclarar es el valor cero, y es el olvido: un campo nuevo cae aqui
	// solo y la puerta lo dice. No significa «no aplica».
	SinDeclarar DeDonde = iota
	// DeCatalogo: lo escribe plazum, asi que es una clave y se traduce.
	DeCatalogo
	// DelCorpus: palabras de un paquete normativo. Viajan tal cual, en el
	// idioma del paquete. Traducirlas crearia obra derivada de un texto legal.
	DelCorpus
	// DeLaPersona: lo escribio quien usa plazum. Traducirlo seria ponerle
	// palabras a alguien.
	DeLaPersona
	// NoEsProsa: una ruta, un identificador, un token, una fecha, un codigo o
	// una clase de CSS. No hay nada que traducir porque no hay palabras.
	NoEsProsa
	// DePlazumSinCatalogo es DEUDA CONOCIDA, y la unica razon de que exista es
	// que esta puerta nacio ROJA y no se podia dejar roja.
	//
	// Significa: lo escribe plazum, viaja en castellano y NO pasa por el
	// catalogo, o sea que hoy sale en espanol en la pagina inglesa. Es
	// exactamente el defecto que esta puerta existe para impedir, y las
	// entradas que lo llevan son las que YA ESTABAN cuando la puerta se
	// escribio.
	//
	// NO ES UN BUCKET PARA USAR. Su cardinal esta vigilado con igualdad exacta
	// por TestLaDeudaDeTextoSinCatalogoTieneTecho, asi que anadir una entrada
	// obliga a subir un numero a mano y a explicarlo en el commit; el numero
	// solo puede bajar. Un rojo permanente no protege —enseña a ignorarlo— y un
	// bucket sin techo tampoco: lo que protege es que crecer cueste.
	DePlazumSinCatalogo
)

func (p DeDonde) String() string {
	switch p {
	case DeCatalogo:
		return "DeCatalogo"
	case DelCorpus:
		return "DelCorpus"
	case DeLaPersona:
		return "DeLaPersona"
	case NoEsProsa:
		return "NoEsProsa"
	case DePlazumSinCatalogo:
		return "DePlazumSinCatalogo"
	}
	return "SinDeclarar"
}

type campoDeVista struct{ Tipo, Campo string }

// RaicesDeVista son los modelos de vista que cada superficie le pasa al motor de
// plantillas.
//
// SE ESCRIBEN Y NO SE DESCUBREN, y hay que decir por que: descubrirlas
// recorriendo el arbol haria que una superficie NUEVA entrara en esta puerta sin
// que nadie lo decidiera, y con ella su censo. Que cueste una linea es lo que
// hace que alguien mire. Lo que SI se descubre del arbol es la lista de
// superficies que tienen plantillas, y `TestLasSuperficiesConPlantillaEstanTodas`
// las cruza contra este mapa en las dos direcciones: si aqui falta una, se ve.
var RaicesDeVista = map[string][]any{
	"pantallas": {
		pantallas.VistaAlcance{}, pantallas.VistaTabla{}, pantallas.VistaVacia{},
		pantallas.VistaHoy{}, pantallas.VistaError{},
	},
	"calendario": {calendarioWeb.Vista{}, calendarioWeb.VistaDeUnaCifra{}},
	"acta":       {actaWeb.Vista{}},
	"uar":        {uarWeb.Vista{}},
	"escalado":   {escaladoWeb.Vista{}},
	"camino":     {camino.Vista{}},
}

// SuperficiesSinVista son las que sirven HTTP y no pintan HTML, con su motivo.
//
// Existe para que la pregunta «¿y export?» tenga respuesta escrita en vez de
// silencio: una superficie que no sale ni en un sitio ni en el otro es
// indistinguible de un olvido, que es la forma exacta en que `eni` se coló entre
// dos recuentos del corpus.
var SuperficiesSinVista = map[string]string{
	"export": "escribe JSON y NDJSON para un SIEM, no HTML. No tiene plantillas ni " +
		"modelo de vista, asi que no hay campo que pueda llegar a una plantilla. Lo " +
		"comprueba TestLasSuperficiesConPlantillaEstanTodas mirando que no tenga plantillas/.",
	"scim": "habla SCIM 2.0 con un IdP: su salida es JSON con un esquema que fija la " +
		"RFC 7644, y no la lee una persona.",
	"serve": "es el aparato de sesion, CSRF y entrada que envuelve a las demas. Sus " +
		"paginas propias (entrar, primer-admin) las pinta con las plantillas de camino.",
}

var ProcedenciaDelTexto = map[campoDeVista]DeDonde{
	// EL MARCO, que va en las cinco pantallas.
	{"Marco", "Titulo"}:   DeCatalogo,
	{"Marco", "Base"}:     NoEsProsa,
	{"Marco", "Inicio"}:   NoEsProsa,
	{"Marco", "Estatico"}: NoEsProsa,
	{"Marco", "Idioma"}:   NoEsProsa,
	// Cuerpo es el NOMBRE de la sub-plantilla que pinta el contenido, no un
	// rotulo: nunca se imprime, se ejecuta.
	{"Marco", "Cuerpo"}: NoEsProsa,

	// EL MENU y la tira del camino guiado.
	{"Entrada", "Titulo"}:   DeCatalogo,
	{"Entrada", "Marcador"}: DeCatalogo,
	{"Entrada", "ID"}:       NoEsProsa,
	{"Entrada", "URL"}:      NoEsProsa,
	{"Enlace", "Titulo"}:    DeCatalogo,
	{"Enlace", "ID"}:        NoEsProsa,
	{"Enlace", "URL"}:       NoEsProsa,

	// LAS CINCO PANTALLAS: su titulo, su por que y sus enlaces.
	{"VistaAlcance", "PorQue"}:       DeCatalogo,
	{"VistaAlcance", "Origen"}:       DeCatalogo,
	{"VistaAlcance", "Siguiente"}:    NoEsProsa,
	{"VistaAlcance", "CSRF"}:         NoEsProsa,
	{"VistaAlcance", "CampoCSRF"}:    NoEsProsa,
	{"VistaAlcance", "IDsSi"}:        NoEsProsa,
	{"VistaAlcance", "IDsNo"}:        NoEsProsa,
	{"VistaAlcance", "URLGuardar"}:   NoEsProsa,
	{"VistaAlcance", "URLLimpiar"}:   NoEsProsa,
	{"VistaAlcance", "URLControles"}: NoEsProsa,
	{"VistaAlcance", "URLVerTodas"}:  NoEsProsa,
	{"VistaAlcance", "URLVerVivas"}:  NoEsProsa,
	// Guardado es CUANDO se guardo, y viaja como argumento de una clave.
	{"VistaAlcance", "Guardado"}: NoEsProsa,

	{"VistaTabla", "PorQue"}:               DeCatalogo,
	{"VistaTabla", "Origen"}:               DeCatalogo,
	{"VistaTabla", "URLAlcance"}:           NoEsProsa,
	{"VistaTabla", "URLAnterior"}:          NoEsProsa,
	{"VistaTabla", "URLSiguiente"}:         NoEsProsa,
	{"VistaTabla", "Columnas"}:             NoEsProsa,
	{"VistaTabla", "ColumnasDesconocidas"}: NoEsProsa,

	{"VistaVacia", "PorQue"}:     DeCatalogo,
	{"VistaVacia", "Origen"}:     DeCatalogo,
	{"VistaVacia", "URLAlcance"}: NoEsProsa,
	{"VistaHoy", "PorQue"}:       DeCatalogo,
	{"VistaHoy", "Origen"}:       DeCatalogo,
	{"VistaHoy", "URLAlcance"}:   NoEsProsa,
	{"VistaError", "Clave"}:      DeCatalogo,
	{"VistaError", "URLAlcance"}: NoEsProsa,
	{"VistaFiltro", "Clave"}:     DeCatalogo,
	{"VistaFiltro", "URL"}:       NoEsProsa,

	// LAS CIFRAS DEL PANEL DE INICIO. Rotulo, matiz, descargo y «sin dato» son
	// las cuatro cosas que plazum dice de un numero, y las cuatro se traducen.
	// El descargo es la que mas: un cubo que en un idioma dice «esto NO dice que
	// se haya incumplido» y en otro no lo dice es el error que este producto no
	// puede cometer.
	{"VistaCifra", "Rotulo"}:   DeCatalogo,
	{"VistaCifra", "Matiz"}:    DeCatalogo,
	{"VistaCifra", "Descargo"}: DeCatalogo,
	{"VistaCifra", "SinDato"}:  DeCatalogo,
	{"VistaCifra", "Tono"}:     NoEsProsa,
	{"VistaCifra", "URL"}:      NoEsProsa,
	{"VistaMarco", "URL"}:      NoEsProsa,
	{"VistaMarco", "URN"}:      NoEsProsa,

	// EL VEREDICTO DEL PLANIFICADOR. Lo emite el nucleo y lo emite EN CLAVE,
	// que es exactamente la forma que este bloque le ha dado a los motivos.
	{"Planificador", "Clave"}:   DeCatalogo,
	{"Planificador", "Arreglo"}: DeCatalogo,
	{"Canal", "Clave"}:          DeCatalogo,
	{"Canal", "Arreglo"}:        DeCatalogo,
	{"Canal", "Descargo"}:       DeCatalogo,

	// LA ENTREVISTA. El texto, la ayuda y la cita de una pregunta son palabras
	// del PAQUETE y viajan en su idioma; lo que plazum pone alrededor es clave.
	{"Pregunta", "Texto"}:              DelCorpus,
	{"Pregunta", "Ayuda"}:              DelCorpus,
	{"Pregunta", "Cita"}:               DelCorpus,
	{"Pregunta", "ID"}:                 NoEsProsa,
	{"Pregunta", "Entidad"}:            NoEsProsa,
	{"Pregunta", "Atributo"}:           NoEsProsa,
	{"Pregunta", "Paquete"}:            NoEsProsa,
	{"Pregunta", "Desbloquea"}:         NoEsProsa,
	{"VistaPregunta", "AvisoValor"}:    DeCatalogo,
	{"VistaPregunta", "Formato"}:       DeCatalogo,
	{"VistaPregunta", "PorQueDormida"}: DeCatalogo,
	{"VistaPregunta", "CampoValor"}:    NoEsProsa,
	{"VistaPregunta", "URLSi"}:         NoEsProsa,
	{"VistaPregunta", "URLNo"}:         NoEsProsa,
	{"VistaPregunta", "URLLimpiar"}:    NoEsProsa,
	{"VistaPregunta", "URLAccion"}:     NoEsProsa,
	{"VistaPregunta", "URLSinValor"}:   NoEsProsa,
	// ValorPuesto es LO QUE CONTESTO UNA PERSONA. Es la unica entrada de esta
	// procedencia hoy, y por eso el bucket existe: traducirlo seria reescribir
	// la respuesta de alguien a una pregunta de cumplimiento.
	{"VistaPregunta", "ValorPuesto"}: DeLaPersona,
	{"VistaOpcion", "Valor"}:         DelCorpus,
	{"VistaOpcion", "URL"}:           NoEsProsa,
	{"VistaOculto", "Nombre"}:        NoEsProsa,
	{"VistaOculto", "Valor"}:         NoEsProsa,
	{"Consecuencia", "Titulos"}:      DelCorpus,
	{"Consecuencia", "Marcos"}:       NoEsProsa,

	// LOS CAMPOS QUE PIDE EL CORPUS, y quien los pide.
	{"Campo", "Etiqueta"}:   DelCorpus,
	{"Campo", "Ayuda"}:      DelCorpus,
	{"Campo", "Cita"}:       DelCorpus,
	{"Campo", "Valores"}:    DelCorpus,
	{"Campo", "Entidad"}:    NoEsProsa,
	{"Campo", "Atributo"}:   NoEsProsa,
	{"Campo", "Tipo"}:       NoEsProsa,
	{"Campo", "Paquetes"}:   NoEsProsa,
	{"Peticion", "Cita"}:    DelCorpus,
	{"Peticion", "Ayuda"}:   DelCorpus,
	{"Peticion", "Paquete"}: NoEsProsa,

	// LA TABLA. Las celdas de Columnas son CONTENIDO del corpus y se pintan
	// tal cual; el rotulo de la cabecera es clave y se compone aparte.
	{"Fila", "ID"}:       NoEsProsa,
	{"Fila", "Paquete"}:  NoEsProsa,
	{"Fila", "Requiere"}: NoEsProsa,
	{"Fila", "Columnas"}: DelCorpus,

	// EL POR QUE DE LA APLICABILIDAD. Clave es de plazum (que CLASE de motivo
	// es); el texto, la cita y el valor son del paquete.
	{"Motivo", "Clave"}:      DeCatalogo,
	{"Motivo", "Texto"}:      DelCorpus,
	{"Motivo", "Cita"}:       DelCorpus,
	{"Motivo", "Valor"}:      DelCorpus,
	{"Motivo", "PreguntaID"}: NoEsProsa,

	// LA EVIDENCIA, que es donde salio el defecto.
	//
	// MotivoClave es la entrada que este bloque cambio: era `Motivo string` con
	// la frase del motor dentro.
	//
	// Y Estado NO ES UNA CLAVE aunque componga una: la plantilla hace
	// `t (printf "evidencia.%s" .Estado)`, o sea que el campo es un FRAGMENTO de
	// clave. Se clasifica por lo que es, un identificador, y por eso la mitad
	// negativa acierta al exigir que no se le pase a `t` directamente.
	{"Evidencia", "MotivoClave"}: DeCatalogo,
	{"Evidencia", "MotivoArgs"}:  NoEsProsa,
	{"Evidencia", "Estado"}:      NoEsProsa,
	{"Evidencia", "Recolector"}:  NoEsProsa,
	{"Evidencia", "Predicado"}:   DelCorpus,

	// EL PIE: los paquetes instalados con su aviso de derechos. La atribucion
	// la escribe el paquete y va tal cual, que es justo lo que la Decision
	// 2011/833/UE exige conservar.
	{"Fuente", "Atribucion"}:     DelCorpus,
	{"Fuente", "URN"}:            NoEsProsa,
	{"Fuente", "Version"}:        NoEsProsa,
	{"Fuente", "LicenciaFuente"}: NoEsProsa,
	{"Fuente", "Identificador"}:  NoEsProsa,
	{"Fuente", "Enlace"}:         NoEsProsa,

	// EL CALENDARIO Y LOS VENCIMIENTOS DEL PANEL. Titulo, articulo y cita son
	// del paquete; lo demas identifica.
	{"Fecha", "Titulo"}:     DelCorpus,
	{"Fecha", "Articulo"}:   DelCorpus,
	{"Fecha", "Cita"}:       DelCorpus,
	{"Fecha", "Marco"}:      NoEsProsa,
	{"Fecha", "Obligacion"}: NoEsProsa,
	{"Fecha", "Hito"}:       NoEsProsa,
	{"Fecha", "Cadencia"}:   NoEsProsa,
	// Aviso es la NOTA DEL HITO, que la escribe el paquete: `nucleo/ventana` la
	// copia tal cual desde `hi.Nota`. Este censo la daba por clave de catalogo
	// y la puerta lo desmintio en su primera ejecucion, que es justo para lo
	// que sirve contrastar contra las plantillas en vez de creerse el censo.
	{"Fecha", "Aviso"}:              DelCorpus,
	{"Fecha", "OrigenDelIntervalo"}: NoEsProsa,
	{"Vencida", "Titulo"}:           DelCorpus,
	{"Vencida", "Articulo"}:         DelCorpus,
	{"Vencida", "Cita"}:             DelCorpus,
	{"Vencida", "Marco"}:            NoEsProsa,
	{"Vencida", "Obligacion"}:       NoEsProsa,
	{"Vencida", "Hito"}:             NoEsProsa,
	{"Divergencia", "Cita"}:         DelCorpus,
	{"Divergencia", "Lectura"}:      NoEsProsa,

	// LOS METODOS QUE UNA PLANTILLA PUEDE IMPRIMIR. Entraron en el censo por la
	// mutacion M3 de la pasada 2: `{{.Coletilla}}` llama igual a un campo y a un
	// metodo sin argumentos, asi que mover una frase de campo a metodo la sacaba
	// de esta puerta entera.
	//
	// Los cinco de hoy son cuatro identificadores y una clave. `Estado` tiene
	// los dos a la vez y es el ejemplo de por que se censan por separado:
	// `Clave()` devuelve «estado.aplica», que se traduce, y `String()` devuelve
	// «aplica», que es la clase de CSS.
	{"Estado", "Clave()"}:  DeCatalogo,
	{"Estado", "String()"}: NoEsProsa,
	{"Nivel", "String()"}:  NoEsProsa,
	// EstadoVenc.String() devuelve el nombre del estado, que es un codigo. Su
	// rama por defecto SI es castellano («estado desconocido (7)»), y se queda
	// asi: solo la alcanza un valor fuera del dominio del propio enum, o sea un
	// estado que el motor no puede producir. Queda dicho porque el censo es el
	// sitio donde se dice.
	{"EstadoVenc", "String()"}:          NoEsProsa,
	{"VistaAlcance", "ParamVerTodas()"}: NoEsProsa,

	// LA DEUDA, QUE LA ENCONTRO ESTA PUERTA EL DIA QUE NACIO.
	//
	// `Regla` es la derivacion del motor de plazos y la escribe `nucleo/ventana`
	// EN CASTELLANO, concatenando: «instante fijado por la norma», «vigencia
	// continua sin fecha de fin declarada», «el reloj no ha arrancado: falta
	// <origen>». `hoy.html` la imprime cruda en dos sitios (lineas 85 y 117), o
	// sea que la pagina inglesa saca castellano.
	//
	// ES EL MISMO DEFECTO QUE ESTA PUERTA VIENE A CERRAR, en un segundo sitio y
	// en otro paquete del nucleo. No se arregla en este bloque y hay que decir
	// por que: los motivos de `estado.Calcular` son once y estan enumerados en
	// `return`s, y las reglas de `ventana` se COMPONEN por concatenacion en
	// decenas de sitios y ademas las consumen la superficie de calendario y la
	// linea de ordenes. Convertirlas a clave y argumentos es un bloque propio,
	// no un arreglo de paso.
	//
	// Queda contado, con techo y a la vista, en vez de escondido detras de una
	// clasificacion complaciente: ponerlas NoEsProsa habria dejado esta puerta
	// verde el dia de su estreno y la deuda sin nombre.
	{"Fecha", "Regla"}:   DePlazumSinCatalogo,
	{"Vencida", "Regla"}: DePlazumSinCatalogo,

	// Y LAS CUATRO DEL SEGUNDO CONSUMIDOR, que aparecieron al subir esta puerta
	// a la raiz el 08-09-2026.
	//
	// `superficies/calendario` envuelve la misma `Regla` de `nucleo/ventana` en
	// cuatro tipos de vista propios, y la pinta en SEIS sitios: cinco en
	// `calendario.html` (lineas 90, 122, 181, 208 y 266) y uno en `cifra.html`
	// (linea 79). Mientras la puerta vivia dentro de `superficies/pantallas`, el
	// techo de igualdad exacta protegia a UNO de los dos consumidores y en el
	// otro un campo nuevo entraba en silencio.
	//
	// Es la misma deuda y el mismo arreglo: cuando `nucleo/ventana` emita clave
	// y argumentos, las seis se van juntas.
	{"FechaVista", "Regla"}:        DePlazumSinCatalogo,
	{"VencidaVista", "Regla"}:      DePlazumSinCatalogo,
	{"DescarteFilaVista", "Regla"}: DePlazumSinCatalogo,
	{"SinFechaVista", "Regla"}:     DePlazumSinCatalogo,
}

// TestNingunTextoDeLaVistaLlegaSinPasarPorElCatalogo es la puerta.
func TestNingunTextoDeLaVistaLlegaSinPasarPorElCatalogo(t *testing.T) {
	porSuperficie := camposDeTextoPorSuperficie(t)
	vivos := unionDeCampos(porSuperficie)
	if len(vivos) < 250 {
		t.Fatalf("el recorrido encuentra %d campos de texto en las siete superficies y son "+
			"muchos menos de los que tiene: o el arbol se ha vaciado, o este test esta "+
			"mirando otra cosa", len(vivos))
	}
	traducidos := camposPasadosATraducir(t)
	if len(traducidos) == 0 {
		t.Fatal("ninguna plantilla pasa un campo a `t`, lo cual es imposible: el contraste " +
			"esta leyendo mal las plantillas y la mitad positiva aprobaria cualquier cosa")
	}

	// SENTIDO 1: todo campo DECLARADO tiene una procedencia de verdad.
	//
	// LO QUE ESTE SENTIDO NO HACE, Y ES EL CAMBIO AL SUBIR A LA RAIZ: ya no
	// exige que TODO campo vivo este declarado, porque hay 148 sin declarar en
	// cinco superficies y exigirlo dejaria esta puerta en rojo permanente, que
	// no protege, entrena a saltarsela. Lo que se hace con esos 148 es CONTARLOS
	// con igualdad exacta, abajo. Aqui se exige lo que si se puede exigir hoy:
	// que ninguna entrada del censo lleve el valor cero.
	for c, p := range ProcedenciaDelTexto {
		if p == SinDeclarar {
			t.Errorf("%s.%s esta en el censo con el VALOR CERO. El cero no es una procedencia, "+
				"es el olvido: escribe cual de las cinco es", c.Tipo, c.Campo)
		}
	}

	// SENTIDO 2: toda entrada del censo sigue existiendo. Sin esta mitad el
	// censo envejece hasta ser una lista de los campos que hubo, y una entrada
	// huerfana hace parecer revisado lo que ya no esta.
	vivo := map[campoDeVista]bool{}
	for _, c := range vivos {
		vivo[c] = true
	}
	for c := range ProcedenciaDelTexto {
		if !vivo[c] {
			t.Errorf("el censo declara la procedencia de %s.%s y ese campo ya no llega a "+
				"ninguna plantilla. O se renombro, o se fue: en los dos casos la linea sobra",
				c.Tipo, c.Campo)
		}
	}

	// SENTIDO 3, QUE ES EL QUE MIDE: la clasificacion se contrasta contra las
	// plantillas, no se cree.
	for _, c := range ordenadoPorProcedencia(ProcedenciaDelTexto) {
		p := ProcedenciaDelTexto[c]
		if !vivo[c] {
			continue // ya se ha dicho arriba
		}
		// EL NOMBRE CON EL QUE LO LLAMA LA PLANTILLA no lleva parentesis:
		// `{{t .Estado.Clave}}` invoca el metodo con la misma sintaxis que un
		// campo. El censo si los lleva, para que se vea de un vistazo cual de
		// los dos es.
		nombre := strings.TrimSuffix(c.Campo, "()")
		if p == DeCatalogo && !traducidos[nombre] {
			t.Errorf("%s.%s se declara DeCatalogo y NINGUNA plantilla se lo pasa a `t` ni a "+
				"`targs`.\n"+
				"  Una clave que nadie traduce no es una clave: o se pinta en crudo, o se\n"+
				"  pinta el castellano que traiga dentro.\n"+
				"  Arreglo: pasarla por `t` en la plantilla, o corregir su procedencia",
				c.Tipo, c.Campo)
		}
		if p != DeCatalogo && traducidos[nombre] && !ambiguo(c.Campo) {
			t.Errorf("%s.%s se declara %s y alguna plantilla se lo pasa a `t`.\n"+
				"  Pasar por el catalogo un dato, una cita de una norma o las palabras de una\n"+
				"  persona sale como FALTA(...) en la pagina, o peor: sale traducido.\n"+
				"  Arreglo: imprimirlo tal cual, o corregir su procedencia",
				c.Tipo, c.Campo, p)
		}
	}
}

// CamposSinDeclararPorSuperficie es la DEUDA DE ALCANCE, con su cardinal.
//
// # Por que existe este numero y no una puerta que lo exija todo
//
// Al subir a la raiz, esta puerta pasa a ver 283 campos de texto en seis
// superficies y el censo cubre los de `pantallas` mas las cuatro `Regla` de
// `calendario`. Declarar los otros 148 de una sentada habria producido 148
// justificaciones escritas del tirón, que es exactamente la afirmacion
// acompañada en su forma industrial: nadie las lee y todas parecen revisadas.
//
// Y dejar la puerta en rojo mientras tanto tampoco vale: un rojo permanente
// enseña a saltarsela, que es el mismo daño con mejor apariencia.
//
// # Lo que si hace un cardinal con igualdad exacta
//
// Impide que la deuda crezca callada. Un campo de texto nuevo en `acta` sube el
// numero de `acta` y pone esto rojo, con el nombre del campo delante: entonces o
// se declara su procedencia (y el numero no sube) o se dice en el commit por que
// entra sin declarar. Y por abajo, cada superficie que se pague obliga a venir a
// bajar SU numero, que es como se sabe que se pagó.
//
// SE CUENTA POR SUPERFICIE Y NO EN TOTAL a proposito: un total deja compensar
// —declarar dos de `uar` y meter dos en `acta` sin que nada se mueva—, y lo que
// se quiere saber es cual de las seis sigue sin mirar.
var CamposSinDeclararPorSuperficie = map[string]int{
	// pantallas esta al dia: es donde nacio el censo.
	"pantallas": 0,
	// calendario tiene declaradas sus cuatro `Regla` (la deuda que trajo esta
	// puerta a la raiz) y le faltan las demas.
	"calendario": 60,
	"acta":       38,
	"escalado":   38,
	"uar":        32,
	"camino":     9,
}

// LOS SEIS NUMEROS SALEN DEL ARBOL Y NO DE UNA ESTIMACION, y la primera vez que
// se escribieron estaban los seis mal: se pusieron los campos que cada
// superficie alcanza (67, 41, 41, 35, 12) sin descontar los que YA declara el
// censo por compartir tipo con `pantallas` —`Fecha`, `Vencida`, `Fuente`,
// `Divergencia`— y la puerta los corrigio a la baja en su primera ejecucion.
//
// Es la regla de la casa cobrandose una pieza al escribirla: **cuando se estima
// el coste de algo que se quiere terminar, se estima a favor**, y aqui el sesgo
// fue al reves porque lo que se contaba era deuda. La cuenta buena la imprime
// esta misma puerta con `t.Logf`, asi que recontarla es leer el log.

// TestLaDeudaDeCamposSinDeclararTieneTechoPorSuperficie vigila el cardinal de
// arriba, en las dos direcciones y superficie a superficie.
func TestLaDeudaDeCamposSinDeclararTieneTechoPorSuperficie(t *testing.T) {
	porSuperficie := camposDeTextoPorSuperficie(t)
	nombres := make([]string, 0, len(porSuperficie))
	for s := range porSuperficie {
		nombres = append(nombres, s)
	}
	sort.Strings(nombres)

	for _, s := range nombres {
		var sinDeclarar []string
		for _, c := range porSuperficie[s] {
			if _, hay := ProcedenciaDelTexto[c]; !hay {
				sinDeclarar = append(sinDeclarar, c.Tipo+"."+c.Campo)
			}
		}
		sort.Strings(sinDeclarar)
		esperados, declarada := CamposSinDeclararPorSuperficie[s]
		if !declarada {
			t.Errorf("la superficie %s tiene %d campos de texto sin declarar y no sale en "+
				"CamposSinDeclararPorSuperficie. Una superficie que no sale es una deuda "+
				"que nadie cuenta", s, len(sinDeclarar))
			continue
		}
		if len(sinDeclarar) != esperados {
			t.Errorf("superficies/%s tiene %d campos de texto sin procedencia declarada y el "+
				"techo dice %d.\n"+
				"  Si ha SUBIDO, hay texto nuevo que puede llegar a una plantilla sin que\n"+
				"  nadie haya dicho de donde salen sus palabras. Declaralo en\n"+
				"  ProcedenciaDelTexto, o sube este numero Y di en el commit por que.\n"+
				"  Si ha BAJADO, alguien ha pagado deuda: baja el numero aqui, que para eso\n"+
				"  la igualdad es exacta.\n"+
				"  Los que hay: %v", s, len(sinDeclarar), esperados, sinDeclarar)
		}
		t.Logf("superficies/%s: %d campos sin declarar", s, len(sinDeclarar))
	}

	// LA OTRA DIRECCION: no se cuenta una superficie que ya no existe.
	for s := range CamposSinDeclararPorSuperficie {
		if _, hay := porSuperficie[s]; !hay {
			t.Errorf("CamposSinDeclararPorSuperficie cuenta la superficie %s y el recorrido "+
				"no la conoce: o se renombro, o dejo de tener vista", s)
		}
	}
}

// TestLasSuperficiesConPlantillaEstanTodas cruza el arbol contra RaicesDeVista.
//
// Es la mitad que impide que esta puerta se quede corta sola: una superficie
// nueva con plantillas y sin raiz declarada no la miraria nadie, y su ausencia
// no se veria porque lo que no se recorre no sale en ningun numero.
func TestLasSuperficiesConPlantillaEstanTodas(t *testing.T) {
	ents, err := os.ReadDir("superficies")
	if err != nil {
		t.Fatalf("no puedo leer superficies/: %v", err)
	}
	conPlantillas := map[string]bool{}
	for _, e := range ents {
		if !e.IsDir() {
			continue
		}
		hs, _ := filepath.Glob(filepath.Join("superficies", e.Name(), "plantillas", "*.html"))
		if len(hs) > 0 {
			conPlantillas[e.Name()] = true
		}
	}
	if len(conPlantillas) < 6 {
		t.Fatalf("solo %d superficies tienen plantillas y hoy son al menos seis: este "+
			"recorrido esta midiendo el vacio", len(conPlantillas))
	}
	for s := range conPlantillas {
		if _, hay := RaicesDeVista[s]; hay {
			continue
		}
		if porque, dicho := SuperficiesSinVista[s]; dicho {
			t.Errorf("superficies/%s se declara SIN vista (%q) y TIENE plantillas. Una de las "+
				"dos cosas es vieja", s, porque)
			continue
		}
		t.Errorf("superficies/%s tiene plantillas y no declara raiz de vista, asi que esta "+
			"puerta no mira ni uno de sus campos.\n"+
			"  Arreglo: anadirla a RaicesDeVista con su modelo de vista, y su cardinal de\n"+
			"  campos sin declarar a CamposSinDeclararPorSuperficie", s)
	}
	for s := range RaicesDeVista {
		if !conPlantillas[s] {
			t.Errorf("RaicesDeVista declara superficies/%s y ese paquete ya no tiene "+
				"plantillas", s)
		}
	}
	// Y LAS QUE DICEN NO TENER VISTA, comprobado y no creido.
	for s, porque := range SuperficiesSinVista {
		if strings.TrimSpace(porque) == "" {
			t.Errorf("superficies/%s se declara sin vista y no dice por que", s)
		}
		if conPlantillas[s] {
			t.Errorf("superficies/%s se declara sin vista y tiene plantillas", s)
		}
	}
}

// ambiguo dice si un nombre de campo lo comparten dos procedencias distintas.
//
// Se DERIVA del censo y no se escribe: una lista de nombres ambiguos puesta a
// mano se queda vieja el dia que uno deje de serlo, y entonces el hueco sigue
// abierto sin que nadie lo sepa.
func ambiguo(campo string) bool {
	visto := SinDeclarar
	for c, p := range ProcedenciaDelTexto {
		if c.Campo != campo {
			continue
		}
		if visto != SinDeclarar && visto != p {
			return true
		}
		visto = p
	}
	return false
}

// TestElHuecoDelContrasteDeProcedenciaSeCuenta imprime lo que la puerta de
// arriba NO mira por dentro, y le pone techo.
//
// La mitad negativa (un campo que no es clave y se pasa a `t`) se salta los
// nombres ambiguos, porque el contraste con las plantillas es por nombre y no
// por tipo. Un hueco sin numero se olvida; con numero, molesta hasta que se
// cierra y ademas se ve si crece.
//
// LA IGUALDAD ES EXACTA a proposito: asi rompe tambien cuando el conjunto
// ENCOGE, que es cuando hay que venir a quitar la excepcion.
func TestElHuecoDelContrasteDeProcedenciaSeCuenta(t *testing.T) {
	nombres := map[string]bool{}
	for c := range ProcedenciaDelTexto {
		if ambiguo(c.Campo) {
			nombres[c.Campo] = true
		}
	}
	var lista []string
	for n := range nombres {
		lista = append(lista, n)
	}
	sort.Strings(lista)
	// Hoy son tres, y los tres por la misma razon: dos tipos distintos usan el
	// mismo nombre para cosas con procedencia distinta.
	//
	//	Columnas  clave de columna en la cabecera (VistaTabla), contenido del
	//	          corpus en la celda (Fila.Columnas, que es el mapa)
	//	Titulo    clave en el marco, en el menu y en la tira del camino; texto
	//	          del corpus en un vencimiento y en una fecha
	//	Valor     texto del corpus en un motivo y en una opcion; nombre de un
	//	          campo oculto de formulario en VistaOculto
	//
	// `Regla` NO esta aqui, y merece decirse porque parecia que si: sus seis
	// entradas comparten procedencia (`DePlazumSinCatalogo`), asi que el nombre
	// no es ambiguo y la mitad negativa SI lo mira. Lo dijo esta puerta al
	// primer intento de meterlo.
	esperados := []string{"Columnas", "Titulo", "Valor"}
	if !reflect.DeepEqual(lista, esperados) {
		t.Errorf("los nombres que la mitad negativa no mira son %v y se esperaban %v.\n"+
			"  Si ha entrado uno, hay un campo mas que puede pasar por el catalogo sin que\n"+
			"  esta puerta lo diga. Si ha salido uno, quita la excepcion.", lista, esperados)
	}
	t.Logf("el contraste por nombre no cubre %d nombre(s): %v", len(lista), lista)
}

// TestLaDeudaDeTextoSinCatalogoTieneTecho pone numero y tope a lo que esta
// puerta encontro y NO arregla.
//
// # Por que existe este test y no un rojo
//
// La puerta nacio roja sobre el arbol real, que es como tiene que nacer una
// puerta. Lo que encontro no cabia en el bloque que la escribio, y las dos
// salidas faciles eran malas: dejarla roja enseña a ignorarla, y clasificar la
// deuda como «NoEsProsa» la habria dejado verde y sin nombre, que es peor
// porque ademas afirma que se miro y estaba bien.
//
// # De 2 a 6 el 08-09-2026, y el numero no crecio
//
// Crecio lo que se estaba mirando. Mientras la puerta vivia dentro de
// `superficies/pantallas`, la deuda medida eran las dos `Regla` que pinta
// `hoy.html`. Al subir a la raiz aparecen las CUATRO de `superficies/calendario`
// (`FechaVista`, `VencidaVista`, `DescarteFilaVista` y `SinFechaVista`), que la
// pintan en seis sitios entre `calendario.html` y `cifra.html`. Es la misma
// prosa de `nucleo/ventana` y el segundo consumidor, que es justo lo que el
// techo viejo no protegia.
//
// # La igualdad es EXACTA, y en los dos sentidos
//
// Hacia arriba, porque una deuda con techo blando no es una deuda, es una
// costumbre. Hacia abajo, porque el dia que `nucleo/ventana` emita clave y
// argumentos —que es el arreglo de verdad— este test se pone rojo y obliga a
// venir a bajar el numero: es la unica forma de que el marcador no siga contando
// una deuda que ya no existe.
func TestLaDeudaDeTextoSinCatalogoTieneTecho(t *testing.T) {
	var deuda []campoDeVista
	for _, c := range ordenadoPorProcedencia(ProcedenciaDelTexto) {
		if ProcedenciaDelTexto[c] == DePlazumSinCatalogo {
			deuda = append(deuda, c)
		}
	}
	esperada := []campoDeVista{
		{"DescarteFilaVista", "Regla"},
		{"Fecha", "Regla"},
		{"FechaVista", "Regla"},
		{"SinFechaVista", "Regla"},
		{"Vencida", "Regla"},
		{"VencidaVista", "Regla"},
	}
	if !reflect.DeepEqual(deuda, esperada) {
		t.Errorf("la deuda de texto que plazum escribe y no traduce es %v y se esperaba %v.\n"+
			"  Si ha CRECIDO: acabas de meter en una pantalla una frase que el codigo redacta\n"+
			"  en castellano, o sea el defecto que esta puerta existe para impedir. La salida\n"+
			"  no es subir este numero, es emitir clave y argumentos como hace\n"+
			"  nucleo/estado.CadenasDelEstado().\n"+
			"  Si ha MENGUADO: alguien la ha arreglado. Baja el numero aqui, que para eso la\n"+
			"  igualdad es exacta.", deuda, esperada)
	}
	t.Logf("deuda conocida: %d campo(s) que plazum escribe y no pasan por el catalogo, en "+
		"DOS consumidores (hoy.html y calendario.html/cifra.html): %v", len(deuda), deuda)
}

// TestElCensoDeProcedenciaSabePonerseRojo es el control negativo, EN LAS DOS
// DIRECCIONES.
//
// Sin el no se sabe si la puerta vigila o acompaña: sus sentidos podrian estar
// comparando conjuntos que siempre casan. Se le ponen delante las formas de
// romperla, sobre datos sinteticos, para no tener que mutar el arbol.
func TestElCensoDeProcedenciaSabePonerseRojo(t *testing.T) {
	traducidos := map[string]bool{"Titulo": true, "PorQue": true}
	vivo := map[campoDeVista]bool{{"X", "Rotulo"}: true, {"X", "Cita"}: true}

	// (1) un campo vivo sin entrada en el censo: lo cuenta el techo por
	//     superficie, y el detector tiene que verlo.
	censo := map[campoDeVista]DeDonde{{"X", "Rotulo"}: DeCatalogo}
	if _, hay := censo[campoDeVista{"X", "Cita"}]; hay {
		t.Fatal("el detector da por censado un campo que no lo esta")
	}

	// (2) una entrada del censo sin campo vivo.
	if vivo[campoDeVista{"X", "Fantasma"}] {
		t.Fatal("el detector da por vivo un campo que ya no existe")
	}

	// (3) una clave que ninguna plantilla traduce, que es la mitad que de
	// verdad mide. `Rotulo` esta declarado DeCatalogo y no esta traducido.
	if traducidos["Rotulo"] {
		t.Fatal("el arnes se ha montado mal: Rotulo no deberia estar traducido")
	}
	roto := false
	for c, p := range censo {
		if p == DeCatalogo && !traducidos[c.Campo] {
			roto = true
		}
	}
	if !roto {
		t.Fatal("la mitad que contrasta contra las plantillas NO se pone roja con una clave " +
			"que nadie traduce, asi que aprobaria el defecto que esta puerta existe para " +
			"impedir")
	}

	// (4) y la contraria: un dato que SI pasa por el catalogo.
	censo2 := map[campoDeVista]DeDonde{{"X", "PorQue"}: NoEsProsa}
	roto2 := false
	for c, p := range censo2 {
		if p != DeCatalogo && traducidos[c.Campo] {
			roto2 = true
		}
	}
	if !roto2 {
		t.Fatal("la mitad negativa no se pone roja con un dato pasado por el catalogo")
	}

	// (5) y el detector de ambiguedad sabe decir que no: un nombre con una sola
	// procedencia no es ambiguo, o la mitad negativa se apagaria entera.
	if ambiguo("MotivoClave") {
		t.Error("MotivoClave tiene una sola procedencia y el detector lo llama ambiguo: con " +
			"un detector asi, la mitad negativa no mira nada")
	}
	if !ambiguo("Titulo") {
		t.Error("Titulo es clave en el marco y texto del corpus en un vencimiento, y el " +
			"detector no lo ve: entonces no esta leyendo el censo")
	}
}

// ---------------------------------------------------------------------------
// El recorrido del arbol de vistas y el de las plantillas.
// ---------------------------------------------------------------------------

// camposDeTextoPorSuperficie recorre el modelo de vista de CADA superficie y
// devuelve los campos de texto que puede alcanzar, con la superficie al lado.
//
// SE GUARDA LA SUPERFICIE porque el techo de deuda se lleva por superficie: un
// total dejaria compensar —declarar dos en una y meter dos en otra sin que nada
// se mueva—, y lo que hace falta saber es cual de las seis sigue sin mirar.
//
// UN TIPO COMPARTIDO SALE EN LAS DOS. `pantalla.Fecha` la alcanzan el panel y el
// calendario, y eso es correcto: lo que se cuenta es «campos que esta superficie
// puede pintar», no «tipos que solo son suyos».
func camposDeTextoPorSuperficie(t *testing.T) map[string][]campoDeVista {
	t.Helper()
	mod := rutaDelModulo(t)
	out := map[string][]campoDeVista{}
	for sup, raices := range RaicesDeVista {
		uniq := map[campoDeVista]bool{}
		visto := map[reflect.Type]bool{}
		var rec func(reflect.Type)
		rec = func(tp reflect.Type) {
			for tp.Kind() == reflect.Ptr || tp.Kind() == reflect.Slice || tp.Kind() == reflect.Array {
				tp = tp.Elem()
			}
			if visto[tp] {
				return
			}
			visto[tp] = true
			// LOS METODOS TAMBIEN, Y NO ES UN EXTRA: era el agujero.
			//
			// Lo encontro la mutacion M3 de la pasada 2 del 07-09-2026. Una
			// plantilla de Go llama a un metodo sin argumentos con la MISMA
			// sintaxis que a un campo (`{{.Coletilla}}`), asi que mover la frase
			// de un campo a un metodo la sacaba del censo entero. Con la puerta
			// mirando solo campos, la suite completa se quedaba VERDE mientras
			// una frase que redacta plazum en castellano llegaba a la celda.
			metodosDeTexto(tp, mod, uniq)
			if tp.Kind() != reflect.Struct {
				return
			}
			for i := 0; i < tp.NumField(); i++ {
				f := tp.Field(i)
				if !f.IsExported() {
					continue
				}
				ft := f.Type
				for ft.Kind() == reflect.Ptr || ft.Kind() == reflect.Slice || ft.Kind() == reflect.Array {
					ft = ft.Elem()
				}
				switch {
				case ft.Kind() == reflect.String:
					uniq[campoDeVista{tp.Name(), f.Name}] = true
				case ft.Kind() == reflect.Map:
					// UN MAPA DE CADENAS TAMBIEN ES TEXTO QUE LLEGA A LA
					// PLANTILLA. Las celdas de la tabla viven en
					// `Fila.Columnas`, que es un map[string]string, y sin esta
					// rama el campo con MAS texto de toda la pantalla de
					// controles se quedaba fuera del censo.
					if ft.Elem().Kind() == reflect.String {
						uniq[campoDeVista{tp.Name(), f.Name}] = true
					}
					rec(ft.Elem())
				default:
					// Se recorre TODO tipo nuestro, no solo los structs:
					// `Estado` es un entero con nombre y su metodo `Clave()` es
					// el que rotula la columna de aplicabilidad.
					rec(ft)
				}
			}
		}
		for _, r := range raices {
			rec(reflect.TypeOf(r))
		}
		lista := make([]campoDeVista, 0, len(uniq))
		for c := range uniq {
			lista = append(lista, c)
		}
		ordenarCampos(lista)
		if len(lista) == 0 {
			t.Fatalf("la superficie %s no aporta ni un campo de texto: su raiz de vista no "+
				"es la que pinta la pagina", sup)
		}
		out[sup] = lista
	}
	return out
}

func unionDeCampos(porSuperficie map[string][]campoDeVista) []campoDeVista {
	uniq := map[campoDeVista]bool{}
	for _, cs := range porSuperficie {
		for _, c := range cs {
			uniq[c] = true
		}
	}
	out := make([]campoDeVista, 0, len(uniq))
	for c := range uniq {
		out = append(out, c)
	}
	ordenarCampos(out)
	return out
}

func ordenarCampos(cs []campoDeVista) {
	sort.Slice(cs, func(i, j int) bool {
		if cs[i].Tipo != cs[j].Tipo {
			return cs[i].Tipo < cs[j].Tipo
		}
		return cs[i].Campo < cs[j].Campo
	})
}

// rutaDelModulo es el prefijo de los tipos QUE SON NUESTROS.
//
// SE LEE DE go.mod y no se escribe: la ruta del modulo ya se cableo a mano en
// cinco puertas una vez, y dos de ellas se quedaron verdes vigilando el vacio el
// dia que el modulo se renombro. Lo impide TestNadieCableaLaRutaDelModulo, que
// puso roja la primera version de este fichero.
//
// Se usa para no censar los metodos de la biblioteca estandar: `time.Time` llega
// al modelo de vista (la fecha de recoleccion) y trae `String()`, `Format()` y
// una docena mas. Meterlos aqui llenaria el censo de entradas que nadie de este
// repositorio puede cambiar.
//
// LO QUE ESO DEJA FUERA, DICHO: `time.Time.String()` produce
// «2026-09-06 09:00:00 +0000 UTC», que es exactamente el volcado de terminal que
// la pasada del comprador ya encontro una vez en esta misma celda. No es prosa
// traducible y no cabe en este censo, pero tampoco lo vigila esta puerta.
func rutaDelModulo(t *testing.T) string {
	t.Helper()
	m, err := modulo.Ruta()
	if err != nil {
		t.Fatalf("leyendo la ruta del modulo de go.mod: %v", err)
	}
	return m + "/"
}

// metodosDeTexto anota los metodos que una plantilla puede imprimir: sin
// argumentos y con una sola salida de tipo cadena.
//
// Se miran los del tipo Y los del puntero al tipo, porque un metodo con receptor
// de puntero tambien lo llama una plantilla cuando el valor es direccionable, y
// el modelo de vista se pasa por puntero desde `responder`.
func metodosDeTexto(tp reflect.Type, mod string, uniq map[campoDeVista]bool) {
	if tp.Name() == "" || !strings.HasPrefix(tp.PkgPath(), mod) {
		return
	}
	anota := func(x reflect.Type) {
		for i := 0; i < x.NumMethod(); i++ {
			m := x.Method(i)
			f := m.Type
			// NumIn() incluye el receptor cuando se pregunta por el tipo, asi
			// que la firma que buscamos es 1 entrada y 1 salida de cadena.
			if f.NumIn() != 1 || f.NumOut() != 1 || f.Out(0).Kind() != reflect.String {
				continue
			}
			uniq[campoDeVista{tp.Name(), m.Name + "()"}] = true
		}
	}
	anota(tp)
	anota(reflect.PointerTo(tp))
}

var (
	// comentarioDePlantilla: {{/* ... */}} y {{- /* ... */ -}}.
	//
	// SE QUITAN ANTES DE MIRAR NADA, y hace falta de verdad: el armazon
	// compartido explica en un comentario que «cada rotulo sale de `t "clave"` o
	// de `t .Campo`», y sin esta linea el contraste se creia que existe un campo
	// llamado `Campo` que se traduce. Una puerta que lee la prosa que habla de
	// ella misma se aprueba sola.
	comentarioDePlantilla = regexp.MustCompile(`(?s)\{\{-?\s*/\*.*?\*/\s*-?\}\}`)
	// pasadoATraducir: el primer argumento de t o de targs, cuando es un campo.
	pasadoATraducir = regexp.MustCompile(`(?:^|[^\w.])(?:t|targs)\s+\$?\.([A-Za-z0-9_.]+)`)
)

// camposPasadosATraducir devuelve los NOMBRES de campo que alguna plantilla le
// pasa a `t` o a `targs`.
//
// Se leen del disco TODAS las plantillas de superficies/, no solo las de una:
// el armazon compartido es el que pinta el marco, y sin el `Marco.Titulo` no
// apareceria traducido por ninguna parte y la mitad positiva acusaria en falso.
func camposPasadosATraducir(t *testing.T) map[string]bool {
	t.Helper()
	var ficheros []string
	for _, patron := range []string{
		filepath.Join("superficies", "*", "plantillas", "*.html"),
		filepath.Join("superficies", "*", "armazon", "*.html"),
	} {
		fs, err := filepath.Glob(patron)
		if err != nil {
			t.Fatalf("buscando plantillas con %s: %v", patron, err)
		}
		ficheros = append(ficheros, fs...)
	}
	if len(ficheros) < 10 {
		t.Fatalf("se han encontrado %d plantillas en superficies/ y son al menos diez: el "+
			"contraste estaria mirando media interfaz", len(ficheros))
	}
	out := map[string]bool{}
	for _, f := range ficheros {
		b, err := os.ReadFile(f) // #nosec G304 -- ruta compuesta por este test, no por una entrada
		if err != nil {
			t.Fatalf("leyendo %s: %v", f, err)
		}
		limpio := comentarioDePlantilla.ReplaceAllString(string(b), " ")
		for _, m := range pasadoATraducir.FindAllStringSubmatch(limpio, -1) {
			// El ultimo segmento es el nombre del campo: `.Planificador.Clave`
			// se traduce por su campo `Clave`.
			partes := strings.Split(m[1], ".")
			out[partes[len(partes)-1]] = true
		}
	}
	return out
}

func ordenadoPorProcedencia(m map[campoDeVista]DeDonde) []campoDeVista {
	out := make([]campoDeVista, 0, len(m))
	for c := range m {
		out = append(out, c)
	}
	ordenarCampos(out)
	return out
}

// ClaveDelIdiomaDelCorpus es la frase que explica por que el contenido de los
// paquetes sale sin traducir.
const ClaveDelIdiomaDelCorpus = "ui.idioma_del_corpus"

// TestLaFraseDelIdiomaDelCorpusSoloSaleDondeEsCierta.
//
// # El descargo que nadie contestaria porque es cierto
//
// La pasada 3 del 07-09-2026 midio la pagina inglesa de `/controles`: de 936
// trozos de prosa, 729 salen en castellano, el 94,6 % de los caracteres. No es
// un fallo —es la frontera legal del invariante 3— pero la pagina no lo decia, y
// quien la abre no puede distinguirlo de un producto a medio traducir. La frase
// lo dice.
//
// **Y aqui esta la trampa que esta puerta existe para cerrar.** La frase afirma
// que lo que sale sin traducir es CONTENIDO DE LOS PAQUETES. Eso es cierto en
// `/controles`, donde lo que va en castellano son titulos, citas, predicados y
// «como se implementa», todo del corpus. En el calendario y en el panel es
// FALSO: alli lo que sale sin traducir incluye la `Regla` que escribe
// `nucleo/ventana` concatenando castellano, y esa no es contenido de ningun
// paquete, es el defecto que este censo cuenta como `DePlazumSinCatalogo`.
//
// Reusar la clave alli dejaria **una frase verdadera tapando un defecto real**,
// que es la peor clase de descargo que hay: nadie lo va a contestar, porque es
// cierto. Y el defecto quedaria explicado y por tanto invisible.
//
// # Como se mide, y por que se puede
//
// La regla es mecanica y sale del censo que ya existe: **una PLANTILLA puede
// pedir esta clave solo si no pinta ningun campo `DePlazumSinCatalogo`.** Si lo
// pinta, la frase mentiria por omision en esa pagina.
//
// LA GRANULARIDAD ES LA PLANTILLA Y NO LA SUPERFICIE, y lo enseño esta puerta al
// primer intento: la primera version comparaba superficies y se puso roja sobre
// `pantallas`, que pide la frase en `tabla.html` (`/controles`, donde es cierta)
// y pinta `Regla` en `hoy.html` (`/hoy`, donde no lo seria). Las dos son la
// misma superficie y son dos paginas distintas, y lo que lee una persona es una
// pagina.
//
// SE CASA POR NOMBRE DE CAMPO, con la misma aproximacion que el resto de este
// fichero y por el mismo motivo. Aqui el error posible es conservador: una
// plantilla que nombre `.Regla` por cualquier motivo cuenta como que pinta
// deuda, o sea que la puerta prohibe de mas y nunca de menos.
func TestLaFraseDelIdiomaDelCorpusSoloSaleDondeEsCierta(t *testing.T) {
	// LOS NOMBRES DE LA DEUDA, sacados del censo y no escritos: el dia que se
	// pague, esta lista se vacia sola y la puerta deja de prohibir donde ya
	// seria cierta.
	deuda := map[string]bool{}
	for c, p := range ProcedenciaDelTexto {
		if p == DePlazumSinCatalogo {
			deuda[strings.TrimSuffix(c.Campo, "()")] = true
		}
	}
	if len(deuda) == 0 {
		t.Fatal("el censo no declara ni un campo DePlazumSinCatalogo, asi que esta puerta " +
			"no puede prohibir nada. O se pago la deuda entera —y entonces esto sobra y hay " +
			"que borrarlo— o el censo no la esta viendo")
	}

	var fs []string
	for _, patron := range []string{
		filepath.Join("superficies", "*", "plantillas", "*.html"),
		filepath.Join("superficies", "*", "armazon", "*.html"),
	} {
		g, err := filepath.Glob(patron)
		if err != nil {
			t.Fatalf("buscando plantillas: %v", err)
		}
		fs = append(fs, g...)
	}
	var pidenLaFrase, pintanDeuda []string
	for _, f := range fs {
		b, err := os.ReadFile(f) // #nosec G304 -- ruta compuesta por este test
		if err != nil {
			t.Fatalf("leyendo %s: %v", f, err)
		}
		limpio := comentarioDePlantilla.ReplaceAllString(string(b), " ")
		pide := strings.Contains(limpio, ClaveDelIdiomaDelCorpus)
		pinta := false
		for n := range deuda {
			if regexp.MustCompile(`\.` + regexp.QuoteMeta(n) + `\b`).MatchString(limpio) {
				pinta = true
			}
		}
		if pide {
			pidenLaFrase = append(pidenLaFrase, f)
		}
		if pinta {
			pintanDeuda = append(pintanDeuda, f)
		}
		if pide && pinta {
			t.Errorf("%s pide %q Y pinta un campo que plazum escribe sin traducir.\n"+
				"  Esa frase dice que lo que sale sin traducir es contenido de los paquetes,\n"+
				"  y en esta pagina parte de lo que sale sin traducir lo escribe plazum.\n"+
				"  Seria una frase VERDADERA tapando un defecto real, que es el descargo que\n"+
				"  nadie contesta porque es cierto.\n"+
				"  Arreglo: quitarla de aqui, o pagar la deuda de esta pagina primero.",
				f, ClaveDelIdiomaDelCorpus)
		}
	}
	sort.Strings(pidenLaFrase)
	sort.Strings(pintanDeuda)
	if len(pidenLaFrase) == 0 {
		t.Fatalf("ninguna plantilla pide %q. La frase que explica el 94,6 %% de castellano de "+
			"la pagina inglesa ha desaparecido, y con ella lo unico que distingue una "+
			"decision legal de un producto a medio traducir", ClaveDelIdiomaDelCorpus)
	}
	if len(pintanDeuda) == 0 {
		t.Fatalf("ninguna plantilla pinta los campos %v, asi que el contraste no esta "+
			"comparando nada: la mitad que prohibe aprobaria cualquier cosa", claves(deuda))
	}
	t.Logf("la frase la piden %v; pintan prosa de plazum sin traducir %v",
		pidenLaFrase, pintanDeuda)
}

// TestElDetectorDeLaFraseDelIdiomaSabePonerseRojo es el control negativo.
//
// Sin el no se sabe si la regla vigila o acompaña: hoy `/controles` no tiene
// deuda y el calendario si, asi que los dos conjuntos son disjuntos y la puerta
// pasa sin comparar nada interesante. Aqui se le pone delante el caso que tiene
// que prohibir.
func TestElDetectorDeLaFraseDelIdiomaSabePonerseRojo(t *testing.T) {
	deuda := map[string]bool{"Regla": true}
	choca := func(html string) bool {
		pide := strings.Contains(html, ClaveDelIdiomaDelCorpus)
		for n := range deuda {
			if pide && regexp.MustCompile(`\.`+regexp.QuoteMeta(n)+`\b`).MatchString(html) {
				return true
			}
		}
		return false
	}
	// La pagina que pide la frase Y pinta prosa sin traducir: prohibida.
	if !choca(`{{t "` + ClaveDelIdiomaDelCorpus + `"}} <p>{{.Regla}}</p>`) {
		t.Error("la regla NO se pone roja con la frase puesta en una pagina que pinta prosa " +
			"de plazum sin traducir, que es exactamente lo que existe para impedir")
	}
	// La que solo pide la frase: permitida, o seria un rojo permanente.
	if choca(`{{t "` + ClaveDelIdiomaDelCorpus + `"}} <p>{{.Titulo}}</p>`) {
		t.Error("la regla prohibe la frase en una pagina sin deuda: con eso seria un rojo " +
			"permanente y se acabaria aflojando")
	}
	// Y la que solo pinta la deuda: no se le exige la frase, se le exige que no
	// la ponga. Sin este caso, la regla podria estar acusando a media interfaz.
	if choca(`<p>{{.Regla}}</p>`) {
		t.Error("la regla acusa a una pagina que pinta la deuda y NO pone la frase, que es " +
			"el estado normal de hoy en el calendario y en el panel")
	}
}

func claves(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
