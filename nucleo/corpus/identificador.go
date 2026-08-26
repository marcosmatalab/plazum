package corpus

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// ---------------------------------------------------------------------------
// El identificador de la fuente, que es DATO, y el enlace, que es DERIVADO.
//
// EL PROBLEMA QUE ESTO CIERRA, y es estructural, no de higiene. Hasta el
// 26-08-2026 cada paquete guardaba su procedencia como una URL completa en el
// campo `fuente`. Una URL es la direccion de una PAGINA, y una pagina se mueve:
// dos de los treinta y un paquetes publicados ya apuntaban a direcciones que
// redirigian (una de ellas con cambio de anfitrion). Con la URL como dato, una
// pagina que se mueve rompe un paquete, y arreglar el esquema de un editor
// obliga a tocar tantos ficheros de datos como paquetes tenga ese editor.
//
// LA REGLA. Lo que se guarda es el IDENTIFICADOR ESTABLE que publica la propia
// autoridad (el ELI de la Union o del BOE, la designacion de una norma ISO, la
// version de PCI DSS, el identificador de publicacion de NIST). El enlace se
// DERIVA de el, y se deriva al pintar. Si manana EUR-Lex cambia su esquema de
// URLs, se cambia Identificador.Enlace y no treinta y un ficheros de datos.
//
// LO QUE ESTO RETIRA. Con la URL como dato hacia falta vigilar que los enlaces
// siguieran vivos, y esa vigilancia no se podia montar: EUR-Lex e ISO responden
// 403 a una peticion automatica, asi que un comprobador no distingue "la pagina
// se ha muerto" de "me han tomado por un robot". Con el identificador como dato
// la pregunta cambia: lo que hay que vigilar es que el IDENTIFICADOR siga
// existiendo, y un identificador no se mueve, se deroga. Ver docs/pendientes.md
// P2 numero 37.
//
// LA VALVULA DE ESCAPE, con su motivo escrito. No todo editor publica un
// identificador estable: AICPA, DISA, CIS, ENX y el PAe no lo hacen, y datos
// propios del proyecto no son una norma. Para esos existe SinIdentificador, que
// EXIGE `motivo`: la direccion se queda, pero queda dicho POR QUE se queda, y
// asi el hueco es una decision consultable en vez de una omision.
// ---------------------------------------------------------------------------

// TipoIdentificador es el esquema de identificacion de la fuente. Vocabulario
// CERRADO, por la misma razon que LicenciaFuente: un esquema nuevo entra con su
// constante aqui y su rama en Enlace, no escribiendo otra cadena en un JSON.
// Si el vocabulario fuera abierto, la derivacion no podria existir.
type TipoIdentificador string

const (
	// ELIUE: European Legislation Identifier de la Union. El valor es la ruta
	// del ELI sin anfitrion ("reg/2016/679/oj", "dir/2022/2555/oj").
	ELIUE TipoIdentificador = "eli-ue"
	// ELIBOE: el mismo ELI, en su version espanola. El valor es la ruta sin
	// anfitrion ("es/rd/2022/05/03/311/con"); el sufijo /con es la version
	// consolidada y forma parte del identificador.
	ELIBOE TipoIdentificador = "eli-boe"
	// NormaISO: la designacion de la norma ("ISO/IEC 27001:2022"), que es lo
	// que la identifica y lo que hay que pedir para comprarla.
	//
	// Lleva ademas Registro, y ese es el unico caso del vocabulario en el que
	// hace falta un segundo dato. Motivo: el catalogo de ISO NO esta indexado
	// por la designacion sino por un numero de registro propio que no se
	// deriva de ella (ISO/IEC 27002:2022 es el registro 75652). El registro es
	// una CLAVE del catalogo del editor, no una direccion: el dia que ISO
	// cambie la forma de sus paginas, cambia Enlace y no cinco paquetes.
	NormaISO TipoIdentificador = "iso"
	// VersionPCIDSS: la version del estandar ("4.0"). PCI SSC sirve todas las
	// versiones desde una unica biblioteca de documentos y no publica una
	// direccion por version, asi que la version es lo que le dice al lector que
	// documento coger de esa biblioteca.
	VersionPCIDSS TipoIdentificador = "pci-dss"
	// PublicacionCSRC: el identificador de publicacion del catalogo CSRC del
	// NIST, tal como lo escribe el propio NIST ("sp/800/53/r5/upd1").
	PublicacionCSRC TipoIdentificador = "nist-csrc"
	// SinIdentificador: la valvula de escape. El editor no publica un
	// identificador estable, o el contenido no es una norma. Se guarda la
	// direccion tal cual y se EXIGE el motivo, porque un hueco sin motivo se
	// lee como un descuido y nadie vuelve a mirarlo.
	SinIdentificador TipoIdentificador = "sin-identificador"
)

// esquemas enumera el vocabulario en orden estable. Es la unica lista: quien
// anada un tipo y se olvide de Enlace se lo encuentra rojo, porque hay un test
// que exige que TODO tipo del vocabulario derive un enlace.
var esquemas = []TipoIdentificador{
	ELIUE, ELIBOE, NormaISO, VersionPCIDSS, PublicacionCSRC, SinIdentificador,
}

// Identificador es la procedencia del paquete, guardada como identidad y no
// como direccion.
type Identificador struct {
	Tipo  TipoIdentificador `json:"tipo"`
	Valor string            `json:"valor"`
	// Registro es la clave con la que el editor tiene indexado su catalogo,
	// cuando esa clave NO se deriva del identificador. Hoy solo la usa ISO.
	Registro string `json:"registro,omitempty"`
	// Motivo es obligatorio en SinIdentificador y esta PROHIBIDO en los demas:
	// un motivo escrito al lado de un identificador estable es un motivo que
	// nadie va a leer, y ademas invita a rellenarlo para aparentar diligencia.
	Motivo string `json:"motivo,omitempty"`
}

// Los anfitriones y prefijos de cada editor. Estan AQUI, en una sola funcion, y
// ese es todo el punto de este fichero: son lo unico que se mueve cuando un
// editor reorganiza su sitio.
const (
	prefijoELIUE    = "https://eur-lex.europa.eu/eli/"
	prefijoELIBOE   = "https://www.boe.es/eli/"
	prefijoISO      = "https://www.iso.org/standard/"
	sufijoISO       = ".html"
	prefijoCSRC     = "https://csrc.nist.gov/pubs/"
	sufijoCSRC      = "/final"
	bibliotecaPCI   = "https://www.pcisecuritystandards.org/document_library/"
	esquemaObligado = "https://"
)

// Enlace deriva la direccion publica desde el identificador.
//
// ES LA UNICA FUNCION QUE CONVIERTE IDENTIDAD EN DIRECCION. Todo lo que pinte
// la procedencia de un paquete pasa por aqui; ningun otro sitio del producto
// puede componer una URL de fuente, y por eso el dia que un editor mueva sus
// paginas hay exactamente un sitio que tocar.
//
// Se llama al PINTAR y no al cargar: el enlace no es dato del paquete, es una
// vista de su identificador. Guardarlo en el modelo al cargar lo volveria a
// convertir en dato, que es el fallo del que se viene.
//
// El anfitrion queda FIJADO por el prefijo y el valor entra siempre detras de
// una barra, asi que un valor adversario no puede cambiar de anfitrion: lo peor
// que consigue es una ruta rara dentro del sitio del editor. Aun asi el valor
// se valida por forma en el linter, para que ni siquiera eso entre al corpus.
//
// Un tipo fuera del vocabulario devuelve cadena vacia y NO adivina: el linter
// ya no deja cargar un paquete asi, y si algun dia lo dejara, una fuente sin
// enlace se ve en pantalla, mientras que un enlace inventado no.
func (i Identificador) Enlace() string {
	switch i.Tipo {
	case ELIUE:
		return prefijoELIUE + i.Valor
	case ELIBOE:
		return prefijoELIBOE + i.Valor
	case NormaISO:
		return prefijoISO + i.Registro + sufijoISO
	case VersionPCIDSS:
		// La version no entra en la direccion porque PCI SSC no publica una
		// por version. Entra en la pantalla como identificador: es lo que dice
		// que documento coger de la biblioteca. El dia que PCI publique una
		// direccion por version, se cambia esta linea.
		return bibliotecaPCI
	case PublicacionCSRC:
		return prefijoCSRC + i.Valor + sufijoCSRC
	case SinIdentificador:
		return i.Valor
	}
	return ""
}

// Enlace deriva la direccion publica de la fuente del paquete. Atajo de
// p.Identificador.Enlace(), para que quien pinte no tenga que saber que hay un
// campo intermedio.
func (p *Paquete) Enlace() string { return p.Identificador.Enlace() }

// Los errores del identificador. Con centinela, como el resto de la frontera
// legal: se comprueban con errors.Is y no buscando una subcadena del mensaje.
var (
	// ErrSinIdentificador: el paquete no dice de donde sale. Las condiciones de
	// reutilizacion del BOE y la Decision 2011/833/UE exigen citar la fuente, y
	// sin identificador no hay de donde derivar la cita.
	ErrSinIdentificador = errors.New("paquete sin identificador de fuente")
	// ErrEsquemaDesconocido: un tipo que no esta en el vocabulario cerrado.
	ErrEsquemaDesconocido = errors.New("tipo de identificador fuera del vocabulario")
	// ErrIdentificadorMalFormado: el valor no tiene la forma de su esquema. La
	// pega mas importante que caza: una URL escrita dentro de un identificador,
	// que es justo el formato viejo colandose por la puerta nueva.
	ErrIdentificadorMalFormado = errors.New("identificador con forma que no es la de su esquema")
	// ErrSinMotivo: se usa la valvula de escape sin decir por que. Un hueco sin
	// motivo se lee como un descuido y nadie vuelve a mirarlo.
	ErrSinMotivo = errors.New("sin-identificador sin motivo escrito")
	// ErrMotivoSobrante: hay motivo en un esquema que SI tiene identificador
	// estable. Sobra, y ademas invita a rellenarlo para aparentar diligencia.
	ErrMotivoSobrante = errors.New("motivo en un esquema con identificador estable")
	// ErrRegistroDelEditor: falta o sobra la clave de catalogo del editor.
	ErrRegistroDelEditor = errors.New("registro del catalogo del editor mal declarado")
	// ErrFuenteHeredada: el paquete todavia declara el campo `fuente` del
	// formato viejo. Se RECHAZA en vez de ignorarse: un campo ignorado en
	// silencio deja a quien lo escribio creyendo que dice algo.
	ErrFuenteHeredada = errors.New("paquete con el campo fuente del formato viejo")
)

// Las formas de cada esquema. Se comprueban porque el valor viene de un fichero
// de datos de un tercero: un identificador con forma equivocada no es una
// rareza teorica, es el sitio por el que se vuelve a colar una URL.
var (
	// Una ruta de identificador: segmentos en minuscula separados por barra,
	// sin esquema, sin anfitrion, sin barra al principio ni al final. Es lo que
	// impide que "https://..." entre por aqui: los dos puntos no casan.
	reRutaIdent = regexp.MustCompile(`^[a-z0-9][a-z0-9_.-]*(/[a-z0-9][a-z0-9_.-]*)+$`)
	// La designacion de una norma ISO, con o sin IEC, con o sin parte.
	reNormaISO = regexp.MustCompile(`^ISO(/IEC)? [0-9]{3,6}(-[0-9]+)?:[0-9]{4}$`)
	// El registro del catalogo de ISO es un numero y nada mas.
	reRegistroISO = regexp.MustCompile(`^[0-9]{3,6}$`)
	// Una version de estandar: numeros separados por puntos.
	reVersion = regexp.MustCompile(`^[0-9]+(\.[0-9]+)*$`)
	// Una ruta dentro de este repositorio, para la valvula de escape cuando la
	// fuente no es una pagina web. Sin dos puntos: eso mata javascript:, data:
	// y cualquier otro esquema que alguien quiera dejar puesto para el dia que
	// esta superficie decida hacer enlaces de verdad.
	reRutaRepo = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._/-]*$`)
)

// esquemaConocido dice si el tipo esta en el vocabulario.
func esquemaConocido(t TipoIdentificador) bool {
	for _, x := range esquemas {
		if x == t {
			return true
		}
	}
	return false
}

// listaDeEsquemas enumera el vocabulario en orden estable, para que el error
// diga que hay que escribir y no solo que lo escrito esta mal.
func listaDeEsquemas() string {
	out := make([]string, 0, len(esquemas))
	for _, x := range esquemas {
		out = append(out, string(x))
	}
	sort.Strings(out)
	return strings.Join(out, ", ")
}

// validarIdentificador EXIGE el identificador y comprueba su forma.
//
// Se comprueba en TODAS las clases. Un paquete referencial no le debe
// atribucion a nadie, pero es justo donde el identificador es lo UNICO que
// aporta: sin el texto de la norma, "ISO/IEC 27001:2022" es todo lo que el
// lector se lleva para ir a buscarla.
func (p *Paquete) validarIdentificador(anotar func(error)) {
	// El formato viejo, primero. Si alguien conserva `fuente` y ademas escribe
	// el identificador, lo que hay que decirle no es que le falta algo: es que
	// lo que dejo puesto ya no lo lee nadie.
	if p.FuenteHeredada != "" {
		anotar(fmt.Errorf("%w: %s declara fuente=%q, que ya no se lee. Una URL como dato "+
			"rompe el paquete el dia que la pagina se mueve. Arreglo: sustituyela por "+
			"identificador con su tipo (%s) y borra fuente",
			ErrFuenteHeredada, p.URN, p.FuenteHeredada, listaDeEsquemas()))
	}

	id := p.Identificador
	if id.Tipo == "" && id.Valor == "" {
		anotar(fmt.Errorf("%w: %s. Las condiciones de reutilizacion del BOE y la Decision "+
			"2011/833/UE exigen citar la fuente, y el enlace se DERIVA del identificador: "+
			"sin el no hay cita. Arreglo: declara identificador con tipo (%s) y valor",
			ErrSinIdentificador, p.URN, listaDeEsquemas()))
		return
	}
	if !esquemaConocido(id.Tipo) {
		anotar(fmt.Errorf("%w: %s declara tipo %q. El vocabulario es cerrado a proposito: "+
			"un esquema nuevo entra con su constante en nucleo/corpus y su rama en "+
			"Identificador.Enlace, que es lo que hace que el enlace se pueda derivar. "+
			"Los que hay: %s", ErrEsquemaDesconocido, p.URN, id.Tipo, listaDeEsquemas()))
		return
	}
	if id.Valor == "" {
		anotar(fmt.Errorf("%w: %s declara tipo %q sin valor. El tipo dice como se deriva el "+
			"enlace; el valor es lo que identifica la norma",
			ErrSinIdentificador, p.URN, id.Tipo))
		return
	}

	malFormado := func(quiero string) {
		anotar(fmt.Errorf("%w: %s declara tipo %q con valor %q, y un %s se escribe %s. "+
			"Ojo con pegar una URL entera aqui: la direccion se deriva, el dato es la "+
			"identidad", ErrIdentificadorMalFormado, p.URN, id.Tipo, id.Valor, id.Tipo, quiero))
	}
	switch id.Tipo {
	case ELIUE, ELIBOE:
		if !reRutaIdent.MatchString(id.Valor) {
			malFormado("como la ruta del ELI sin anfitrion, en minuscula y sin barra " +
				"inicial (reg/2016/679/oj, es/rd/2022/05/03/311/con)")
		}
	case PublicacionCSRC:
		if !reRutaIdent.MatchString(id.Valor) {
			malFormado("como el identificador de publicacion del CSRC (sp/800/53/r5/upd1)")
		}
	case NormaISO:
		if !reNormaISO.MatchString(id.Valor) {
			malFormado("como la designacion de la norma (ISO/IEC 27001:2022, ISO 22301:2019)")
		}
	case VersionPCIDSS:
		if !reVersion.MatchString(id.Valor) {
			malFormado("como la version del estandar, solo numeros y puntos (4.0)")
		}
	case SinIdentificador:
		esWeb := strings.HasPrefix(id.Valor, esquemaObligado) && !strings.ContainsAny(id.Valor, " \t\n")
		esRuta := !strings.Contains(id.Valor, ":") && reRutaRepo.MatchString(id.Valor)
		if !esWeb && !esRuta {
			malFormado("o como una direccion https:// o como una ruta de este " +
				"repositorio; ningun otro esquema de URL entra")
		}
	}

	// Registro: solo ISO, y ISO siempre. Las dos direcciones, porque un
	// registro suelto en otro esquema seria un dato que nadie usa (y el dia que
	// alguien lo mirara, mentiria) y un ISO sin registro no deriva enlace.
	switch {
	case id.Tipo == NormaISO && !reRegistroISO.MatchString(id.Registro):
		anotar(fmt.Errorf("%w: %s es una norma ISO y declara registro %q. El catalogo de ISO "+
			"no esta indexado por la designacion sino por un numero de registro propio "+
			"(ISO/IEC 27002:2022 es el 75652), y sin el no se puede derivar el enlace",
			ErrRegistroDelEditor, p.URN, id.Registro))
	case id.Tipo != NormaISO && id.Registro != "":
		anotar(fmt.Errorf("%w: %s declara tipo %q y registro %q. El registro es la clave del "+
			"catalogo de ISO y no la usa ningun otro esquema: un dato que nadie lee acaba "+
			"mintiendo", ErrRegistroDelEditor, p.URN, id.Tipo, id.Registro))
	}

	// Motivo: solo la valvula de escape, y siempre.
	switch {
	case id.Tipo == SinIdentificador && strings.TrimSpace(id.Motivo) == "":
		anotar(fmt.Errorf("%w: %s se queda con la direccion en crudo y no dice por que. "+
			"Arreglo: escribe en motivo que le falta a ese editor (no publica identificador "+
			"citable, el contenido no es una norma, ...), para que el hueco sea una decision "+
			"consultable y no un descuido", ErrSinMotivo, p.URN))
	case id.Tipo != SinIdentificador && id.Motivo != "":
		anotar(fmt.Errorf("%w: %s declara tipo %q y ademas un motivo. El motivo solo existe "+
			"para %s: al lado de un identificador estable no lo lee nadie",
			ErrMotivoSobrante, p.URN, id.Tipo, SinIdentificador))
	}
}
