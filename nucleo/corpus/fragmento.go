package corpus

// LA NORMA ES IDENTIFICABLE Y EL PUNTO CONCRETO NO. EL DESCENSO QUE FALTABA.
//
// # Lo que ya estaba, y por que se queda corto
//
// `Identificador` resolvio la identidad al nivel del DOCUMENTO: vocabulario
// cerrado de cinco esquemas, enlace derivado al pintar en un solo sitio, y una
// valvula (`sin-identificador`) que exige motivo. Bien.
//
// Un nivel mas abajo no hay nada. `Obligacion.Articulo` es texto libre («anexo,
// punto 12.2.3», «ritual plazum sobre el art. 78.6, segunda frase») y la
// direccion del punto concreto viaja PEGADA dentro de `Cita`, como URL completa
// y a mano. Medido el 08-09-2026 sobre el corpus publicado: **190 de 266
// obligaciones con reloj llevan una URL dentro de su cita**.
//
// Tres cosas que eso impide, y ninguna es cosmetica:
//
//	1. Enlazar una obligacion a SU fragmento. Hoy el enlace o esta pegado a mano
//	   o no esta, y quien pinta no puede derivarlo.
//	2. Cotejarla contra la instantanea POR IDENTIFICADOR. Las instantaneas de
//	   `corpus-vigilancia/` trocean por articulo y guardan su huella; casar
//	   obligacion con articulo hoy exige adivinar del texto de `Articulo`.
//	3. Cambiar de sitio la fuente sin editar el corpus. Es exactamente lo que el
//	   `Identificador` resolvio arriba, y aqui sigue abierto.
//
// # Lo que se hace: UN CAMPO, no una arquitectura
//
// `Obligacion.Ancla` es el fragmento dentro del documento de la fuente, en el
// vocabulario del esquema que el paquete ya declara: `art_23` en ELI de la Union,
// `a31` en ELI del BOE. El enlace profundo se DERIVA al pintar, con la misma
// regla que el del documento: una sola funcion, y ninguna otra parte del producto
// compone direcciones.
//
// # Es OPCIONAL, y eso esta pensado
//
// Hacerlo obligatorio hoy pondria en rojo doscientas sesenta y seis obligaciones
// de una vez, y una puerta que nace pidiendo una campana entera no se cumple: se
// afloja. Lo que si es obligatorio es la COHERENCIA de lo que se escriba, y eso
// es lo que el linter mira.

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var (
	// ErrFragmentoSinIdentificador: hay fragmento y el paquete no sabe hacer enlaces.
	ErrFragmentoSinIdentificador = errors.New("fragmento de articulo en un paquete sin identificador")
	// ErrFragmentoConEsquemaQueNoLosDirecciona: el esquema no direcciona fragmentos.
	ErrFragmentoConEsquemaQueNoLosDirecciona = errors.New(
		"fragmento de articulo en un esquema que no direcciona fragmentos")
	// ErrFragmentoMalFormado: la forma no es la del esquema que el paquete declara.
	ErrFragmentoMalFormado = errors.New("fragmento de articulo con la forma de otro esquema")
)

// Las dos formas que hoy tienen fragmento direccionable. Se validan por forma y
// no solo por prefijo porque el fragmento acaba en una URL: lo que entre aqui sale
// detras de una almohadilla en el navegador de un cliente.
// LOS ANEXOS VAN EN NUMERACION ROMANA Y ESO LO DIJO EL CORPUS, NO YO. La primera
// version de `formaFragmentoELIUE` pedia digitos detras del prefijo, y al subir los
// fragmentos que ya estaban pegados en las citas se quedaron fuera SEIS del CRA:
// `anx_VIII` cinco veces y `anx_I` una. El ELI de la Union numera los articulos
// en arabigo y los anexos en romano, que es como los numera el propio Diario.
// Cabian dos salidas y solo una es honesta: ensanchar la forma, o dejar seis
// obligaciones sin fragmento y no decirlo.
// Y LA FORMA DEL BOE ESTABA MAL, que es la causa raiz y no una consecuencia.
//
// Decia `^(a|da|dt|df|dd)[0-9]+$`, o sea «a» mas digitos. Verificado contra las
// paginas ELI que este producto enlaza (`boe.es/eli/es/lo/2018/12/05/3/con` y
// `boe.es/eli/es/l/2023/02/20/2/con`, consultadas el 11-09-2026), **el BOE no
// numera asi**:
//
//	articulos 1 a 9     a1 ... a9
//	articulo 10 en      a1-2, a1-3, ... o sea `a<primer digito>-<orden>`
//	adelante            (art. 22 = a2-4, art. 65 = a6-7)
//	disposiciones       da, da-2, dt, dt-2, df, df-2, dd
//
// Asi que `a22` NO EXISTE en el documento y `a65` tampoco: la forma declarada
// era **incapaz de expresar el ancla correcta de cualquier articulo del 10 en
// adelante**. Por eso los diez fragmentos de articulo de dos digitos del corpus
// estaban los diez mal y los cinco de un digito los cinco bien: no era una
// errata repetida, era que lo correcto no se podia escribir.
//
// Y las disposiciones iban igual de mal en la otra direccion: la forma pedia
// `da1` y el BOE escribe `da` y `da-2`.
var (
	formaFragmentoELIUE = regexp.MustCompile(`^(art|rct)_[0-9]+[a-z]{0,3}$|^anx_[IVXLC]+$`)
	// El primer digito, opcionalmente seguido del orden dentro de su decena,
	// para los articulos; y las cuatro clases de disposicion, con o sin orden.
	formaFragmentoELIBOE = regexp.MustCompile(`^a[0-9](-[0-9]+)?$|^(da|dt|df|dd)(-[0-9]+)?$`)
)

// fragmentoDelEsquema dice si un esquema direcciona fragmentos y con que forma.
//
// LOS TRES QUE NO ESTAN, con su motivo, porque un mapa incompleto y callado es
// como se cuela un enlace inventado:
//
//	iso        el enlace es la pagina de compra de la norma; no hay fragmento
//	           que direccionar y ademas el texto no es redistribuible.
//	pci-dss    el enlace es la biblioteca del consejo, que no publica una
//	           direccion por version, menos aun por requisito.
//	nist-csrc  publica el PDF entero; el fragmento seria una pagina, no un punto.
func fragmentoDelEsquema(t TipoIdentificador) (*regexp.Regexp, bool) {
	switch t {
	case ELIUE:
		return formaFragmentoELIUE, true
	case ELIBOE:
		return formaFragmentoELIBOE, true
	}
	return nil, false
}

// EnlaceDeLaObligacion deriva la direccion del PUNTO CONCRETO.
//
// Misma doctrina que `Identificador.Enlace`, un piso mas abajo: se llama al
// pintar, el anfitrion queda fijado por el prefijo del esquema, y sin fragmento
// devuelve el enlace del documento en vez de inventarse un fragmento. Devolver el
// documento y no vacio es deliberado: llevar al lector a la norma sin el salto es
// peor que el salto y mucho mejor que no llevarlo.
func (p *Paquete) EnlaceDeLaObligacion(o Obligacion) string {
	base := p.Enlace()
	a := strings.TrimSpace(o.Fragmento)
	if a == "" || base == "" {
		return base
	}
	if _, hay := fragmentoDelEsquema(p.Identificador.Tipo); !hay {
		return base
	}
	return base + "#" + a
}

func (p *Paquete) validarFragmentos(anotar func(error)) {
	forma, direccionable := fragmentoDelEsquema(p.Identificador.Tipo)
	for _, o := range p.Obligaciones {
		a := strings.TrimSpace(o.Fragmento)
		if a == "" {
			continue
		}
		if p.Identificador.Tipo == SinIdentificador || p.Identificador.Tipo == "" {
			anotar(fmt.Errorf("%w: %s/%s declara el fragmento %q y el paquete no tiene "+
				"identificador de fuente, asi que no hay documento al que colgarla.\n"+
				"  Un fragmento sin documento no lleva a ningun sitio, y ademas hace creer que el "+
				"punto concreto es citable cuando no lo es",
				ErrFragmentoSinIdentificador, p.URN, o.ID, recortar(a, 30)))
			continue
		}
		if !direccionable {
			anotar(fmt.Errorf("%w: %s/%s declara el fragmento %q y el esquema %q no direcciona "+
				"fragmentos.\n"+
				"  Hoy solo lo hacen el ELI de la Union (art_23) y el del BOE (a31). Los otros "+
				"tres enlazan a una pagina de compra, a una biblioteca o a un PDF entero, y un "+
				"fragmento ahi seria un salto inventado",
				ErrFragmentoConEsquemaQueNoLosDirecciona, p.URN, o.ID, recortar(a, 30),
				p.Identificador.Tipo))
			continue
		}
		if !forma.MatchString(a) {
			anotar(fmt.Errorf("%w: %s/%s declara el fragmento %q y el esquema %q la escribe de "+
				"otra manera (%s).\n"+
				"  Se valida por FORMA y no solo por prefijo porque esto acaba detras de una "+
				"almohadilla en el navegador de un cliente: un fragmento con la forma del otro "+
				"esquema no falla, lleva a una pagina que se abre y no salta, y eso no se "+
				"distingue de un salto correcto sin ir a mirar",
				ErrFragmentoMalFormado, p.URN, o.ID, recortar(a, 30), p.Identificador.Tipo,
				forma))
		}
	}
}
