package pantallas

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/marcosmatalab/plazum/internal/modulo"
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
// # Que se afirma aqui, en una frase
//
// Por cada campo de texto que puede llegar a una plantilla, hay escrita una
// PROCEDENCIA. Tres viajan en crudo con motivo (corpus, persona, dato) y una
// dice que lo escribe plazum, y entonces es clave de catalogo. No hay bucket
// para «lo escribe plazum y no se traduce»: eso es el defecto.
//
// La quinta, `DePlazumSinCatalogo`, es la excepcion que confirma lo anterior y
// no estaba prevista: la puerta NACIO ROJA sobre el arbol real y encontro dos
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
// # Lo que esta puerta NO mira, con su cardinal
//
// El contraste con las plantillas se hace **por nombre de campo**, porque una
// referencia de plantilla (`{{.Titulo}}`) es relativa al contexto del `with` o
// del `range` que la rodea y resolverla hasta el tipo exigiria un comprobador de
// tipos de plantillas que este repositorio no tiene. Cuando dos campos de tipos
// distintos comparten nombre y NO comparten procedencia, el nombre es ambiguo y
// queda fuera de la mitad negativa. El cardinal lo imprime y lo vigila
// `TestElHuecoDelContrasteDeProcedenciaSeCuenta`, con igualdad exacta y con los
// nombres escritos, para que no pueda crecer en silencio.
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

// ProcedenciaDelTexto es el censo: de donde salen las palabras de cada campo de
// texto del modelo de vista.
//
// NO HAY ENTRADA «DePlazum», y esa ausencia es la puerta entera. Lo que escribe
// plazum va por catalogo o no va.
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
}

// TestNingunTextoDeLaVistaLlegaSinPasarPorElCatalogo es la puerta.
func TestNingunTextoDeLaVistaLlegaSinPasarPorElCatalogo(t *testing.T) {
	vivos := camposDeTextoDeLaVista(t)
	if len(vivos) < 100 {
		t.Fatalf("el recorrido encuentra %d campos de texto en el modelo de vista y son "+
			"muchos menos de los que tiene: o el arbol se ha vaciado, o este test esta "+
			"mirando otra cosa", len(vivos))
	}
	traducidos := camposPasadosATraducir(t)
	if len(traducidos) == 0 {
		t.Fatal("ninguna plantilla pasa un campo a `t`, lo cual es imposible: el contraste " +
			"esta leyendo mal las plantillas y la mitad positiva aprobaria cualquier cosa")
	}

	// SENTIDO 1: todo campo vivo tiene procedencia escrita.
	for _, c := range vivos {
		p, hay := ProcedenciaDelTexto[c]
		if !hay {
			t.Errorf("%s.%s puede llegar a una plantilla y NADIE HA DICHO DE DONDE SALEN SUS "+
				"PALABRAS.\n"+
				"  Si lo escribe plazum, tiene que ser una clave de catalogo (DeCatalogo): una\n"+
				"  cadena que el codigo redacta y la plantilla imprime sale en castellano en la\n"+
				"  pagina inglesa, y ninguna puerta del catalogo puede verlo.\n"+
				"  Si no lo escribe plazum, di cual de las tres es: DelCorpus, DeLaPersona o\n"+
				"  NoEsProsa.\n"+
				"  Arreglo: una linea en ProcedenciaDelTexto, en procedencia_del_texto_test.go",
				c.Tipo, c.Campo)
			continue
		}
		if p == SinDeclarar {
			t.Errorf("%s.%s esta en el censo con el VALOR CERO. El cero no es una procedencia, "+
				"es el olvido: escribe cual de las cuatro es", c.Tipo, c.Campo)
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
	for _, c := range ordenado(ProcedenciaDelTexto) {
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
// arriba NO mira, y le pone techo.
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
	// De los tres, HOY solo `Titulo` se le pasa de verdad a `t`, o sea que solo
	// ahi el hueco esta abierto de par en par. Los otros dos se cuentan igual
	// porque el hueco es el mismo: si manana alguien pasa `Valor` por el
	// catalogo, esta puerta se callaria.
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
// # La igualdad es EXACTA, y en los dos sentidos
//
// Hacia arriba, porque una deuda con techo blando no es una deuda, es una
// costumbre. Hacia abajo, porque cuando alguien arregle una de estas dos, este
// test se pone rojo y le obliga a venir a bajar el numero: es la unica forma de
// que el marcador no siga contando una deuda que ya no existe.
func TestLaDeudaDeTextoSinCatalogoTieneTecho(t *testing.T) {
	var deuda []campoDeVista
	for _, c := range ordenado(ProcedenciaDelTexto) {
		if ProcedenciaDelTexto[c] == DePlazumSinCatalogo {
			deuda = append(deuda, c)
		}
	}
	esperada := []campoDeVista{{"Fecha", "Regla"}, {"Vencida", "Regla"}}
	if !reflect.DeepEqual(deuda, esperada) {
		t.Errorf("la deuda de texto que plazum escribe y no traduce es %v y se esperaba %v.\n"+
			"  Si ha CRECIDO: acabas de meter en una pantalla una frase que el codigo redacta\n"+
			"  en castellano, o sea el defecto que esta puerta existe para impedir. La salida\n"+
			"  no es subir este numero, es emitir clave y argumentos como hace\n"+
			"  nucleo/estado.CadenasDelEstado().\n"+
			"  Si ha MENGUADO: alguien la ha arreglado. Baja el numero aqui, que para eso la\n"+
			"  igualdad es exacta.", deuda, esperada)
	}
	t.Logf("deuda conocida: %d campo(s) que plazum escribe y no pasan por el catalogo: %v",
		len(deuda), deuda)
}

// TestElCensoDeProcedenciaSabePonerseRojo es el control negativo, EN LAS DOS
// DIRECCIONES.
//
// Sin el no se sabe si la puerta vigila o acompaña: sus tres sentidos podrian
// estar comparando conjuntos que siempre casan. Se le ponen delante las tres
// formas de romperla, sobre datos sinteticos, para no tener que mutar el arbol.
func TestElCensoDeProcedenciaSabePonerseRojo(t *testing.T) {
	traducidos := map[string]bool{"Titulo": true, "PorQue": true}
	vivo := map[campoDeVista]bool{{"X", "Rotulo"}: true, {"X", "Cita"}: true}

	// (1) un campo vivo sin entrada en el censo.
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

// camposDeTextoDeLaVista recorre el modelo de vista desde sus RAICES y devuelve
// todos los campos de texto que puede alcanzar.
//
// Las raices NO se escriben: se sacan del AST, buscando los tipos que
// implementan `fijarIdioma`, que es el metodo por el que `responder` reconoce a
// una pagina. Asi, una sexta pantalla entra en esta puerta sola. Una lista
// escrita al lado se quedaria corta justo el dia que hay una pantalla nueva, que
// es el dia que mas falta hace.
func camposDeTextoDeLaVista(t *testing.T) []campoDeVista {
	t.Helper()
	raices := raicesDeVista(t)
	if len(raices) < 5 {
		t.Fatalf("solo se han encontrado %d raices de vista (%v) y son al menos cinco: el "+
			"AST no esta viendo los metodos fijarIdioma", len(raices), raices)
	}
	porNombre := map[string]reflect.Type{}
	for _, v := range []any{VistaAlcance{}, VistaTabla{}, VistaVacia{}, VistaHoy{}, VistaError{}} {
		porNombre[reflect.TypeOf(v).Name()] = reflect.TypeOf(v)
	}
	mod := rutaDelModulo(t)
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
		// Lo encontro la mutacion M3 de la pasada 2. Una plantilla de Go llama a
		// un metodo sin argumentos con la MISMA sintaxis que a un campo
		// (`{{.Coletilla}}`), asi que mover la frase de un campo a un metodo la
		// sacaba del censo entero. Con la puerta mirando solo campos, la suite
		// completa se quedaba VERDE mientras una frase que redacta plazum en
		// castellano llegaba a la celda.
		//
		// La forma que importa es la unica que una plantilla puede imprimir sin
		// argumentos: cero entradas y una sola salida de tipo cadena.
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
				// UN MAPA DE CADENAS TAMBIEN ES TEXTO QUE LLEGA A LA PLANTILLA.
				// Las celdas de la tabla viven en `Fila.Columnas`, que es un
				// map[string]string, y sin esta rama el campo con MAS texto de
				// toda la pantalla de controles se quedaba fuera del censo.
				if ft.Elem().Kind() == reflect.String {
					uniq[campoDeVista{tp.Name(), f.Name}] = true
				}
				rec(ft.Elem())
			default:
				// Se recorre TODO tipo nuestro, no solo los structs: `Estado` es
				// un entero con nombre y su metodo `Clave()` es el que rotula la
				// columna de aplicabilidad.
				rec(ft)
			}
		}
	}
	for _, r := range raices {
		tp, hay := porNombre[r]
		if !hay {
			t.Fatalf("el AST declara la raiz de vista %q y el recorrido por reflexion no la "+
				"conoce: las dos mitades tienen que ver las mismas pantallas", r)
		}
		rec(tp)
	}
	out := make([]campoDeVista, 0, len(uniq))
	for c := range uniq {
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Tipo != out[j].Tipo {
			return out[i].Tipo < out[j].Tipo
		}
		return out[i].Campo < out[j].Campo
	})
	return out
}

// rutaDelModulo es el prefijo de los tipos QUE SON NUESTROS.
//
// SE LEE DE go.mod y no se escribe: la ruta del modulo ya se cableo a mano en
// cinco puertas una vez, y dos de ellas se quedaron verdes vigilando el vacio el
// dia que el modulo se renombro. Lo impide TestNadieCableaLaRutaDelModulo, que
// puso roja la primera version de este fichero.
//
// Se usa para no censar los metodos de la biblioteca estandar: `time.Time`
// llega al modelo de vista (la fecha de recoleccion) y trae `String()`,
// `Format()` y una docena mas. Meterlos aqui llenaria el censo de entradas que
// nadie de este repositorio puede cambiar.
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
// Se miran los del tipo Y los del puntero al tipo, porque un metodo con
// receptor de puntero tambien lo llama una plantilla cuando el valor es
// direccionable, y el modelo de vista se pasa por puntero desde `responder`.
func metodosDeTexto(tp reflect.Type, mod string, uniq map[campoDeVista]bool) {
	if tp.Name() == "" || !strings.HasPrefix(tp.PkgPath(), mod) {
		return
	}
	anota := func(t reflect.Type) {
		for i := 0; i < t.NumMethod(); i++ {
			m := t.Method(i)
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

// raicesDeVista lee del AST los tipos con metodo fijarIdioma.
func raicesDeVista(t *testing.T) []string {
	t.Helper()
	fs := token.NewFileSet()
	paq, err := parser.ParseDir(fs, ".", nil, 0)
	if err != nil {
		t.Fatalf("parseando el paquete: %v", err)
	}
	var out []string
	for _, p := range paq {
		for _, f := range p.Files {
			for _, d := range f.Decls {
				fn, ok := d.(*ast.FuncDecl)
				if !ok || fn.Name.Name != "fijarIdioma" || fn.Recv == nil {
					continue
				}
				if len(fn.Recv.List) == 0 {
					continue
				}
				tp := fn.Recv.List[0].Type
				if e, ok := tp.(*ast.StarExpr); ok {
					tp = e.X
				}
				if id, ok := tp.(*ast.Ident); ok {
					out = append(out, id.Name)
				}
			}
		}
	}
	sort.Strings(out)
	return out
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
// Se leen del disco las plantillas propias de esta superficie Y la del armazon
// compartido, que es la que pinta el marco: sin ella, `Marco.Titulo` no
// aparaceria traducido por ninguna parte y la mitad positiva acusaria en falso.
func camposPasadosATraducir(t *testing.T) map[string]bool {
	t.Helper()
	ficheros, err := filepath.Glob(filepath.Join("plantillas", "*.html"))
	if err != nil {
		t.Fatalf("buscando plantillas: %v", err)
	}
	armazon, err := filepath.Glob(filepath.Join("..", "camino", "armazon", "*.html"))
	if err != nil {
		t.Fatalf("buscando el armazon: %v", err)
	}
	ficheros = append(ficheros, armazon...)
	if len(ficheros) < 5 {
		t.Fatalf("se han encontrado %d plantillas y son al menos cinco (cuatro propias y el "+
			"armazon): el contraste estaria mirando media superficie", len(ficheros))
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

func ordenado(m map[campoDeVista]DeDonde) []campoDeVista {
	out := make([]campoDeVista, 0, len(m))
	for c := range m {
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Tipo != out[j].Tipo {
			return out[i].Tipo < out[j].Tipo
		}
		return out[i].Campo < out[j].Campo
	})
	return out
}
