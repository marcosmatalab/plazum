// Package corpus define el formato de paquete normativo y lo que se deriva de el.
//
// La decision de diseno que sostiene este paquete: un paquete no declara solo
// obligaciones. Declara ademas los tipos de entidad con su esquema de atributos,
// las preguntas de alcance, las plantillas de entregable y los tipos de recurso
// que necesita. De ahi se derivan, sin escribir una linea por norma:
//
//	EsquemaUI    los formularios de la interfaz
//	Entrevista   el cuestionario de alcance, ordenado por obligaciones desbloqueadas
//	Entregables  los documentos, con trazabilidad obligacion -> plantilla -> campo
//	Conectores   que recolectores hacen falta y cuales no
//
// La propiedad que esto compra, y que es verificable en CI: anadir la norma 31
// no toca ni un fichero fuera de su directorio de paquete. Un GRC cuyo corpus es
// un arbol plano de requisitos cargado desde Excel no puede tener esta propiedad,
// porque no tiene donde declarar nada de lo anterior.
package corpus

// Obligacion es el atomo. Aqui va solo lo que el resto del sistema necesita
// del paquete; la temporalidad completa vive en ventana y la aplicabilidad en
// aplicabilidad.
type Obligacion struct {
	ID       string `json:"id"`
	Articulo string `json:"articulo"`
	// Titulo es la etiqueta legible de la obligacion, la que sale en una lista
	// de controles. OPCIONAL a proposito: hacerla obligatoria obligaria a
	// reescribir hoy los 30 paquetes del corpus, y un formato que se rompe al
	// crecer no lo adopta nadie. Cuando falta, TituloLegible da el respaldo.
	//
	// Lleva el limite de la frontera legal como cualquier otra prosa: un titulo
	// es justo donde alguien pega el enunciado de un control de un catalogo de
	// pago, y lo pega de buena fe, porque "es solo el titulo".
	Titulo string `json:"titulo,omitempty"`
	// Ancla es el fragmento de ESTA obligacion dentro del documento de la
	// fuente, en el vocabulario del esquema del paquete (`art_23`, `a31`).
	// OPCIONAL: ver fragmento.go, que explica por que.
	Fragmento  string        `json:"fragmento,omitempty"`
	TextoLegal string        `json:"texto_legal,omitempty"` // vacio en referencial y delegado
	Cita       string        `json:"cita"`
	Vigencia   Vigencia      `json:"vigencia"`
	Entregable string        `json:"entregable,omitempty"` // ref a Plantilla.ID
	Recursos   []TipoRecurso `json:"recursos,omitempty"`
	// Delegado dice que herramienta externa comprueba esto. Obligatorio y solo
	// permitido en paquetes de clase Delegado.
	Delegado  string   `json:"delegado,omitempty"`
	Preguntas []string `json:"preguntas,omitempty"` // IDs de Pregunta que la desbloquean

	// La extension e2e (Anexo B): clase primaria obligatoria, facetas
	// opcionales, reloj declarado y cadena de escalado.
	ClaseE2E     string        `json:"clase_e2e"`
	Facetas      []string      `json:"facetas,omitempty"`
	Temporalidad *Temporalidad `json:"temporalidad,omitempty"`
	Escalado     []Escalon     `json:"escalado,omitempty"`

	// VersionesLinguisticas es el mismo articulo en las otras lenguas en las que
	// la norma se publico, indexado por su etiqueta ("en"). Ver D-25.
	//
	// # EL IDIOMA NO PUEDE TOCAR EL RELOJ, y por eso vive AQUI DENTRO
	//
	// Es la restriccion que eligio este modelo entre las tres que habia. Un
	// paquete hermano por idioma (`mdr-en`) daria dos obligaciones distintas
	// para el motor, cada una con SU `Temporalidad`, y nada impediria que
	// divergieran: dos clientes leyendo la misma norma en dos idiomas verian dos
	// fechas. Aqui no hay un segundo sitio donde poner un reloj porque no hay
	// una segunda obligacion: hay UNA, con UNA `Temporalidad`, y el idioma
	// alcanza al texto y a nada mas.
	//
	// LO VIGILA: TestNingunaVersionLinguisticaPuedeLlevarUnReloj, y hace falta
	// AUNQUE el tipo no tenga el campo: `nucleo/corpus` no usa
	// DisallowUnknownFields, asi que un "temporalidad" escrito dentro de una
	// version linguistica se ignoraria EN SILENCIO. El paquete cargaria, el
	// linter callaria, y quien lo escribio creeria que hizo algo.
	//
	// EL VALOR CERO ES LA AUSENCIA y significa lo honesto: esta obligacion no
	// tiene version en otra lengua, y la interfaz en ingles lo dice con
	// `aviso.idioma_del_corpus` en vez de disimularlo.
	VersionesLinguisticas map[string]VersionLinguistica `json:"versiones_linguisticas,omitempty"`
}

// TituloLegible es la etiqueta que se ensena cuando hay que ensenar una sola
// linea de la obligacion. Devuelve siempre algo derivado de lo que hay, en este
// orden y por esta razon:
//
//	Titulo    lo que escribio quien autoro el paquete, si lo escribio.
//	Articulo  el localizador. En la practica ya trae etiqueta dentro ("Anexo II
//	          4.2.5 Mecanismo de autenticacion (usuarios externos) [op.acc.5]"),
//	          asi que es un respaldo legible, aunque no sea un titulo.
//	ID        el identificador. Feo, pero unico y citable: mejor eso que un
//	          hueco en blanco en una tabla de controles.
//
// Devuelve cadena vacia solo si la obligacion no tiene ninguna de las tres, y
// eso el linter ya no lo deja cargar (ErrObligacionSinID). Quien pinte esto no
// tiene que inventarse un texto de relleno: si llega vacio, es un fallo de
// carga, no una obligacion sin nombre.
func (o Obligacion) TituloLegible() string {
	switch {
	case o.Titulo != "":
		return o.Titulo
	case o.Articulo != "":
		return o.Articulo
	default:
		return o.ID
	}
}

// Paquete es la unidad de distribucion del corpus.
type Paquete struct {
	URN      string `json:"urn"`
	Version  string `json:"version"`
	Clase    Clase  `json:"clase"`
	Licencia string `json:"licencia"`
	// LicenciaFuente es el regimen de derechos de la fuente, del vocabulario
	// cerrado de arriba. OBLIGATORIO: el linter no carga un paquete sin el.
	LicenciaFuente LicenciaFuente `json:"licencia_fuente"`
	// Atribucion es el aviso literal que hay que ENSENAR a quien usa el
	// producto. OBLIGATORIO, y en todos los estratos: donde hay obligacion de
	// atribuir dice a quien, y donde no la hay dice que puede hacer el lector
	// con ese contenido, que es la misma pregunta desde el otro lado.
	//
	// Es texto y no clave de catalogo: no se traduce. Ver nucleo/pantalla.
	Atribucion string `json:"atribucion"`
	// Identificador es de donde sale el contenido, guardado como IDENTIDAD y
	// no como direccion. El enlace que exigen las condiciones de reutilizacion
	// se DERIVA de el al pintar, con Identificador.Enlace. Ver identificador.go.
	Identificador Identificador `json:"identificador"`
	// FuenteHeredada es el campo `fuente` del formato viejo, que llevaba la URL
	// completa. Sigue leyendose SOLO para rechazarlo con un error que diga que
	// hacer: si se quitara del tipo, encoding/json lo ignoraria en silencio y
	// quien lo escribio se quedaria creyendo que su paquete cita la fuente.
	// Lleva `omitempty` a proposito: solo se lee, nunca se escribe, y si algun
	// dia se serializa un Paquete el campo retirado no puede reaparecer.
	FuenteHeredada string        `json:"fuente,omitempty"`
	Consolidado    bool          `json:"consolidado"` // obliga al aviso de texto informativo
	Vigencia       Vigencia      `json:"vigencia"`
	Entidades      []TipoEntidad `json:"entidades,omitempty"`
	Preguntas      []Pregunta    `json:"preguntas,omitempty"`
	Obligaciones   []Obligacion  `json:"obligaciones"`
	// Pruebas es QUE SE MIRA para saber si una obligacion consta. Va en el
	// nivel del paquete y no dentro de la obligacion a proposito: una
	// obligacion puede tener varias, y asi la prueba NOMBRA a la suya en vez de
	// heredarla de su posicion (invariante 7). Ver prueba.go.
	Pruebas    []Prueba    `json:"pruebas,omitempty"`
	Plantillas []Plantilla `json:"plantillas,omitempty"`
	// Roles son las FIGURAS a las que escalan las obligaciones de este
	// paquete, cada una con su origen: la nombra la norma, o la propone
	// plazum. Van en el paquete y no en codigo por el invariante 2: un rol
	// es una figura de la norma, no un vocabulario del producto. Ver roles.go.
	Roles []Rol `json:"roles,omitempty"`
	// Transposicion solo lo declaran las DIRECTIVAS, y lo declaran todas. Una
	// directiva no vincula por si misma: lo que obliga es la norma nacional que
	// la transpone. Ver transposicion.go.
	Transposicion *Transposicion `json:"transposicion,omitempty"`
	Escalas       []string       `json:"escalas,omitempty"`
	// Aplicabilidad son las reglas que deciden a quien alcanza cada
	// obligacion, en el dialecto Datalog estratificado. Van aqui, en el
	// fichero de datos, y no en codigo Go: es lo que hace cierto el
	// invariante 2 y lo que permite que el corpus se actualice con un
	// fichero firmado en vez de con una release del binario.
	Aplicabilidad Aplicabilidad `json:"aplicabilidad,omitempty"`
	// Dorados se carga desde pruebas/*.json del directorio del paquete; no se
	// declara en paquete.json.
	Dorados []Dorado `json:"-"`
}

// ---------------------------------------------------------------------------
// La frontera legal, campo a campo.
//
// EL AGUJERO QUE ESTO CIERRA. El limite de texto de un paquete referencial solo
// miraba texto_legal. Los otros veinte y pico campos de texto libre del formato
// (la ayuda de un atributo, la descripcion de una entidad, el texto de una
// pregunta, el titulo de una plantilla) no los miraba nadie, asi que el
// enunciado de un control de ISO, PCI DSS, SOC 2 o TISAX entraba por cualquiera
// de ellos y el linter no decia nada. Es el mismo agujero que la clase fuera de
// rango, por otra puerta: la unica frontera que este proyecto declara no
// negociable se esquivaba escribiendo el texto en el campo de al lado.
//
// EL CRITERIO, que es lo que hay que poder discutir. Cada campo de texto libre
// se clasifica en uno de dos tipos, y el tipo decide el limite:
//
//	prosa       texto escrito para que lo lea una persona. Es donde cabe el
//	            enunciado de un control, y es donde se cuela sin mala intencion,
//	            porque "es solo la ayuda" o "es solo el titulo". Limite corto.
//	referencia  identificador, localizador, clave de formulario, fecha, enlace o
//	            declaracion de licencia. No sustituye al texto normativo, y por
//	            eso "CAT/DEMO 9999:2026 A.5.1" tiene que seguir valiendo. Pero
//	            sigue siendo texto libre, asi que lleva techo, no barra libre.
//	derivacion  dos campos, los dos de un dorado y los dos razonamiento del
//	            autor sobre su propio caso, no texto de un tercero: la
//	            cita_del_esperado (por que esa fecha, con la cuenta hecha) y el
//	            subconjunto_porque (por que ese caso renuncia a la
//	            exhaustividad). Legitimamente largos los dos.
//
// NADIE QUEDA FUERA, y eso lo vigila un test: camposDeTexto tiene que enumerar
// TODOS los campos de cadena del formato. Si manana alguien anade un campo y se
// olvida de clasificarlo, el test de exhaustividad lo dice, porque un campo
// nuevo sin clasificar es exactamente por donde volveria a entrar el texto.
//
// LO QUE NO CIERRA, dicho para que conste. El limite es POR CAMPO: quien quiera
// copiar un catalogo entero puede repartirlo entre la ayuda, la descripcion y el
// titulo de cien obligaciones. Contra eso no hay linter que valga, hay revision
// del paquete; lo que el limite corta es el caso real, que es pegar el control
// de un tiron en el campo que tenia a mano.
// ---------------------------------------------------------------------------
