package main

// LAS TRES FECHAS DE UNA NORMA DE LA UNION, SACADAS COMO DATO.
//
// # El agujero que cierra, medido
//
// El 03-09-2026 el corpus tenia 230 relojes y 196 con la vigencia sin contrastar
// por nadie. La causa estaba escrita en `vigencias_test.go`: las instantaneas del
// BOE traen las tres fechas como campo (`fecha_disposicion`, `fecha_publicacion`,
// `fecha_vigencia`) y las de Cellar no, asi que el contraste mecanico solo
// alcanzaba a las normas espanolas. Y lo que vive en ese lado ciego no falla
// ruidosamente: DESAPARECE del calendario, que es la unica forma de fallo que un
// producto de cumplimiento no puede permitirse, porque nadie echa de menos una
// fila que nunca vio.
//
// # Por que esto NO es un parser de prosa, que es lo que uno espera escribir
//
// La entrada en vigor de un reglamento vive en su ultimo articulo y en forma de
// REGLA («entrara en vigor a los veinte dias de su publicacion»), no de fecha. La
// tentacion es una expresion regular sobre ese parrafo. Se miro antes la otra
// puerta y el dato estaba COMO DATO: la ficha `notice=branch` de Cellar (el
// servicio de la Oficina de Publicaciones) trae, al nivel de la OBRA:
//
//	WORK_DATE_DOCUMENT                     la fecha DEL ACTO
//	DATE_PUBLICATION                       la de publicacion en el DOUE, para el
//	                                       Diario nuevo (a partir de 2023-10)
//	RESOURCE_LEGAL_PUBLISHED_IN_...        y para el Diario viejo, dentro del
//	  EMBEDDED_NOTICE/WORK/DATE_PUBLICATION enlace al numero del Diario
//	RESOURCE_LEGAL_DATE_ENTRY-INTO-FORCE   una por cada hito, con su ANNOTATION
//
// Cada hito lleva TYPE_OF_DATE con el codigo de la autoridad fd_335: `EV` es la
// entrada en vigor y `MA` es la fecha de aplicacion, que NO son la misma cosa y
// entre ellas hay dos anos enteros en el RGPD (vigor 2016-05-24, aplicacion
// 2018-05-25). Y lleva COMMENT_ON_DATE con la REGLA que la produjo, en la forma
// `DATPUB +20`, que es lo que este fichero usa para no creerse la fecha a ciegas.
//
// # UN HITO PUEDE LLEVAR VARIAS ANOTACIONES, Y ESO NO ES UN DETALLE
//
// Una misma fecha puede hacer DOS PAPELES a la vez, y Cellar lo escribe como dos
// ANNOTATION dentro del mismo elemento. El Reglamento 2023/1114 (MiCA) tiene el
// 29-06-2023 anotado como aplicacion parcial (art. 149.4) Y como entrada en
// vigor (art. 149.1); el 2014/910 hace lo mismo con el 17-09-2014, y el 2017/745
// tiene el 26-05-2021 con dos anotaciones de aplicacion.
//
// COSTO UNA MEDIDA FALSA EN ESTA MISMA SESION. La primera exploracion leyo solo
// la PRIMERA anotacion de cada hito y concluyo que dos de las diez normas de la
// UE del corpus no tenian entrada en vigor. Las diez la tienen. Lo cazo el test
// contra el fixture REAL de MiCA, que se puso rojo diciendo que salia una fecha
// donde se esperaba un hueco; ninguna mutacion lo habria encontrado, porque el
// error estaba en lo que se creia que decia la fuente.
//
// # Lo que se comprueba, y por que cada comprobacion
//
//  1. EL CELEX DE LA FICHA ES EL QUE SE PIDIO (invariante 7). Se empareja por el
//     identificador que la propia respuesta declara, no por el orden de la
//     peticion. Una ficha de otra norma daria tres fechas perfectamente formadas
//     de un acto que no es el nuestro, y eso no se ve nunca.
//  2. LA FECHA CUADRA CON LA REGLA QUE LA FUENTE DECLARA. Si el comentario dice
//     `DATPUB +20`, publicacion + 20 dias tiene que dar la fecha de vigor. No es
//     redundante: es el unico sitio del recorrido donde dos datos independientes
//     de la fuente se contrastan entre si.
//  3. UNA REGLA QUE EMPIEZA POR `DATPUB` Y NO SE SABE RESOLVER ES UN ERROR, NUNCA
//     UN VALOR POR DEFECTO (invariante 8, la tercera hermana: presente y no
//     interpretable). Ausente y no-interpretable no son la misma nada, y la que
//     sale por descuido es la segunda.
//  4. UNA FECHA CENTINELA DE LA FUENTE NO ES UNA FECHA. Cellar escribe
//     `9999-12-31` para «sin fin de vigencia» y `1001-01-01` para «no consta»
//     (esta ultima esta HOY en el Reglamento 2017/745, art. 123.3(d)). Se
//     descarta el valor y se deja dicho en su sitio, que es lo contrario de
//     tragarselo.
//  5. NO HABER ENTRADA EN VIGOR NO ES UN ERROR DE LA RESPUESTA: es una ausencia
//     de la fuente. Se deja la fecha vacia CON SU MOTIVO ESCRITO y no se rellena
//     con la fecha de aplicacion, que es justo la conflacion que el invariante
//     10 prohibe. Hoy no le pasa a ninguna de las diez normas de la UE del
//     corpus, asi que esa rama se recorre con dato sintetico y se dice.

import (
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Codigos de la autoridad fd_335 de la Oficina de Publicaciones. Se nombran aqui
// porque son vocabulario de la FUENTE, no normas: no es cablear un marco.
const (
	codigoEnVigor    = "EV"      // entrada en vigor del acto
	codigoAplicacion = "MA"      // fecha de aplicacion
	codigoParcial    = "MA/PART" // ... de una parte del acto, no de todo
	codigoDesdePub   = "DATPUB"  // la regla cuenta desde la publicacion
	codigoArticulo   = "ART"     // lo que sigue es el articulo que lo dice
)

// AplicacionUE es un hito de aplicacion tal cual lo declara la fuente.
//
// NO dice QUE capitulo se aplica en esa fecha: Cellar nombra el articulo de la
// disposicion final que fija el escalon (el 71.2 del CRA), no el ambito que
// alcanza. Eso se queda fuera a proposito y se cuenta en vez de inventarse.
type AplicacionUE struct {
	Desde string `json:"desde,omitempty"`
	// Alcance: "todo el acto" o "parte del acto", segun la fuente marque MA/PART.
	Alcance string `json:"alcance"`
	// Apoyo es el articulo de la propia norma que la fuente cita como origen del
	// hito ("71.2"). Sirve para ir a leerlo, que es lo unico que zanja un plazo.
	Apoyo string `json:"apoyo,omitempty"`
	Nota  string `json:"nota,omitempty"`
}

// FechasUE son las tres fechas de un acto de la Union, cada una por separado,
// mas los hitos de aplicacion escalonada.
type FechasUE struct {
	Acto        string
	Publicacion string
	Vigor       string
	// MotivoSinVigor se escribe cuando la fuente no declara entrada en vigor. Un
	// hueco con motivo es un hueco; un hueco sin motivo se lee como un cero.
	MotivoSinVigor string
	Aplicacion     []AplicacionUE
}

// --- la ficha branch de Cellar ---

// anotacionCellar es UN PAPEL de una fecha. Una fecha puede hacer varios (ser la
// entrada en vigor y a la vez el escalon de aplicacion de unos articulos), y
// entonces trae una anotacion por papel. Leer solo la primera se lleva por
// delante los demas sin decir nada.
type anotacionCellar struct {
	Tipo       string `xml:"TYPE_OF_DATE"`
	Comentario string `xml:"COMMENT_ON_DATE"`
}

type fechaCellar struct {
	Valor       string            `xml:"VALUE"`
	Anotaciones []anotacionCellar `xml:"ANNOTATION"`
}

type obraCellar struct {
	CELEX []struct {
		Valor string `xml:"VALUE"`
	} `xml:"ID_CELEX"`
	Acto        []fechaCellar `xml:"WORK_DATE_DOCUMENT"`
	Publicacion []fechaCellar `xml:"DATE_PUBLICATION"`
	EnVigor     []fechaCellar `xml:"RESOURCE_LEGAL_DATE_ENTRY-INTO-FORCE"`
	// El Diario anterior a octubre de 2023 no pone la fecha de publicacion en la
	// obra: la pone en el numero del Diario al que enlaza.
	Diario []struct {
		Publicacion []fechaCellar `xml:"EMBEDDED_NOTICE>WORK>DATE_PUBLICATION"`
	} `xml:"RESOURCE_LEGAL_PUBLISHED_IN_OFFICIAL-JOURNAL"`
}

type ramaCellar struct {
	Obra obraCellar `xml:"WORK"`
}

// parsearFechasCellar saca las tres fechas y los hitos de aplicacion de la ficha
// `notice=branch`. celexPedido es con lo que se comprueba la identidad.
func parsearFechasCellar(b []byte, celexPedido string) (FechasUE, error) {
	var r ramaCellar
	if err := xml.Unmarshal(b, &r); err != nil {
		return FechasUE{}, fmt.Errorf("%w: la ficha de fechas de Cellar no es el XML de siempre "+
			"(%v). Arreglo: repite con -sin-cache y mira la respuesta cruda en la cache",
			ErrRespuestaIlegible, err)
	}
	o := r.Obra

	// 1. La identidad, antes que nada (invariante 7).
	if len(o.CELEX) == 0 {
		return FechasUE{}, fmt.Errorf("%w: la ficha de fechas no declara ningun CELEX, asi que no "+
			"hay forma de saber de que norma son las fechas que trae. Arreglo: comprueba que la "+
			"peticion lleva la cabecera Accept: application/xml;notice=branch", ErrRespuestaIlegible)
	}
	quiero := strings.ToUpper(strings.TrimSpace(celexPedido))
	casa := false
	for _, c := range o.CELEX {
		if strings.EqualFold(strings.TrimSpace(c.Valor), quiero) {
			casa = true
		}
	}
	if !casa {
		return FechasUE{}, fmt.Errorf("%w: se pidieron las fechas del CELEX %s y la ficha que "+
			"responde es de %s. Arreglo: repite con -sin-cache",
			ErrRespuestaIlegible, quiero, strings.Join(celexDe(o), ", "))
	}

	var f FechasUE
	var err error
	if f.Acto, err = unaFecha(o.Acto, "la fecha del acto (WORK_DATE_DOCUMENT)"); err != nil {
		return FechasUE{}, err
	}
	pub := o.Publicacion
	if len(pub) == 0 {
		for _, d := range o.Diario {
			pub = append(pub, d.Publicacion...)
		}
	}
	if f.Publicacion, err = unaFecha(pub, "la fecha de publicacion en el DOUE"); err != nil {
		return FechasUE{}, err
	}

	// 2. Los hitos, separados por lo que la FUENTE dice que son. Se recorre POR
	// ANOTACION y no por hito: una misma fecha puede ser la entrada en vigor y a
	// la vez el escalon de aplicacion de unos articulos, y entonces trae dos.
	var vigores []anotacionCellar
	var fechasDeVigor []string
	for _, h := range o.EnVigor {
		if len(h.Anotaciones) == 0 {
			return FechasUE{}, fmt.Errorf("%w: la ficha trae un hito de entrada en vigor (%s) sin "+
				"ninguna anotacion, asi que no se sabe si es la entrada en vigor o una fecha de "+
				"aplicacion. Son cosas distintas y confundirlas mueve el reloj legal anos "+
				"enteros, asi que no se elige por defecto",
				ErrRespuestaIlegible, recortar(h.Valor, 20))
		}
		for _, a := range h.Anotaciones {
			switch codigoAutoridad(a.Tipo) {
			case codigoEnVigor:
				vigores = append(vigores, a)
				fechasDeVigor = append(fechasDeVigor, h.Valor)
			case codigoAplicacion:
				f.Aplicacion = append(f.Aplicacion, aplicacionDe(h.Valor, a))
			case "":
				return FechasUE{}, fmt.Errorf("%w: la ficha trae un hito de entrada en vigor (%s) "+
					"con una anotacion sin TYPE_OF_DATE, asi que no se sabe que papel hace esa "+
					"fecha. No se elige por defecto",
					ErrRespuestaIlegible, recortar(h.Valor, 20))
			default:
				// Un codigo nuevo de la autoridad fd_335. No se descarta en
				// silencio: callarlo dejaria una fecha sin contar y nadie la
				// echaria de menos.
				return FechasUE{}, fmt.Errorf("%w: la ficha trae un hito de fecha con el codigo "+
					"%q, que esta herramienta no conoce. Arreglo: mira la autoridad fd_335 de la "+
					"Oficina de Publicaciones y decide si es entrada en vigor (%s) o aplicacion "+
					"(%s); no se adivina, porque adivinar mal mueve el calendario de un cliente",
					ErrRespuestaIlegible, codigoAutoridad(a.Tipo), codigoEnVigor, codigoAplicacion)
			}
		}
	}
	switch len(vigores) {
	case 0:
		f.MotivoSinVigor = fmt.Sprintf("la ficha de Cellar del CELEX %s no declara ningun hito "+
			"con el codigo %s (entrada en vigor): solo trae %d de aplicacion. La fecha se queda "+
			"VACIA a proposito: rellenarla con la de aplicacion seria la conflacion que prohibe "+
			"el invariante 10. Arreglo: leer el articulo final del acto y escribir la fecha a "+
			"mano en el paquete, diciendo de donde sale",
			quiero, codigoEnVigor, len(f.Aplicacion))
	case 1:
		if f.Vigor, err = fechaUtil(fechasDeVigor[0]); err != nil {
			return FechasUE{}, fmt.Errorf("%w: la entrada en vigor del CELEX %s no es una fecha "+
				"utilizable (%v)", ErrRespuestaIlegible, quiero, err)
		}
		if err := comprobarLaRegla(f.Publicacion, f.Vigor, vigores[0].Comentario); err != nil {
			return FechasUE{}, err
		}
	default:
		return FechasUE{}, fmt.Errorf("%w: la ficha del CELEX %s declara %d entradas en vigor y "+
			"un acto entra en vigor una vez. Arreglo: mirar la ficha en EUR-Lex; si de verdad "+
			"son varias, hay que decidir cual manda y escribirlo, no coger la primera",
			ErrRespuestaIlegible, quiero, len(vigores))
	}
	return f, nil
}

func celexDe(o obraCellar) []string {
	out := make([]string, 0, len(o.CELEX))
	for _, c := range o.CELEX {
		out = append(out, strings.TrimSpace(c.Valor))
	}
	return out
}

// unaFecha exige exactamente una. Cero es un hueco y dos es una ambiguedad, y en
// los dos casos coger la primera seria inventarse cual.
func unaFecha(fs []fechaCellar, que string) (string, error) {
	if len(fs) == 0 {
		return "", fmt.Errorf("%w: la ficha de Cellar no trae %s. Arreglo: comprueba el CELEX en "+
			"EUR-Lex; si la ficha existe y no la trae, esta herramienta no puede fechar la norma "+
			"y hay que hacerlo a mano diciendo de donde sale el dato", ErrRespuestaIlegible, que)
	}
	if len(fs) > 1 {
		return "", fmt.Errorf("%w: la ficha de Cellar trae %d valores para %s. Arreglo: mirar la "+
			"ficha; coger el primero seria elegir por orden, y el orden no lo garantiza nadie",
			ErrRespuestaIlegible, len(fs), que)
	}
	d, err := fechaUtil(fs[0].Valor)
	if err != nil {
		return "", fmt.Errorf("%w: %s de la ficha de Cellar no se puede usar (%v)",
			ErrRespuestaIlegible, que, err)
	}
	return d, nil
}

// fechaUtil convierte el VALUE de Cellar en una fecha, y rechaza los centinelas.
//
// Las TRES formas de la nada, y solo la tercera es un error (invariante 8):
// ausente lo trata quien llama; presente y centinela es la fuente diciendo «no
// consta», y se devuelve vacio con su motivo; presente y no interpretable es un
// dato que hay y no se entiende, y eso es error siempre.
func fechaUtil(v string) (string, error) {
	s := strings.TrimSpace(v)
	if s == "" {
		return "", fmt.Errorf("%w: la fecha viene vacia", ErrRespuestaIlegible)
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return "", fmt.Errorf("%w: %q no es una fecha AAAA-MM-DD", ErrRespuestaIlegible, recortar(s, 20))
	}
	// 1952 es el ano del primer acto publicado en el Diario Oficial (Tratado
	// CECA). Fuera de la ventana no hay actos: son los centinelas con los que
	// Cellar escribe «no consta» (1001-01-01) y «sin fin de vigencia»
	// (9999-12-31).
	if t.Year() < 1952 || t.Year() > 2200 {
		return "", errCentinela{s}
	}
	return t.Format("2006-01-02"), nil
}

// errCentinela distingue «la fuente escribio su marca de no-consta» de «esto no
// se entiende». Las dos dan cadena vacia y solo una es un fallo.
type errCentinela struct{ valor string }

func (e errCentinela) Error() string {
	return fmt.Sprintf("la fuente escribio %q, que es su marca de «no consta» y no una fecha", e.valor)
}

// aplicacionDe arma un hito de aplicacion. Un centinela NO se descarta en
// silencio: se guarda el hito con la fecha vacia y la nota que dice por que, que
// es lo que permite contarlo despues.
func aplicacionDe(valor string, an anotacionCellar) AplicacionUE {
	a := AplicacionUE{Alcance: "todo el acto"}
	palabras := palabrasDelComentario(an.Comentario)
	for i, p := range palabras {
		if p == codigoParcial {
			a.Alcance = "parte del acto"
		}
		if p == codigoArticulo && i+1 < len(palabras) {
			a.Apoyo = palabras[i+1]
		}
	}
	d, err := fechaUtil(valor)
	if err != nil {
		a.Nota = "la fuente no da fecha para este hito: " + err.Error()
		return a
	}
	a.Desde = d
	return a
}

// comprobarLaRegla contrasta la fecha con la REGLA que la propia fuente declara.
//
// Es el unico sitio del recorrido donde dos datos independientes de la fuente se
// miran el uno al otro, y por eso vale la pena aunque parezca redundante: si
// Cellar se equivoca en la fecha o nosotros leemos la publicacion de otro sitio,
// aqui se ve. Una regla que empieza por DATPUB y no se sabe resolver es ERROR y
// no un pase: es la tercera hermana del invariante 8.
func comprobarLaRegla(publicacion, vigor, comentario string) error {
	palabras := palabrasDelComentario(comentario)
	for i, p := range palabras {
		if p != codigoDesdePub {
			continue
		}
		if i+1 >= len(palabras) {
			return fmt.Errorf("%w: la fuente dice que la entrada en vigor se cuenta desde la "+
				"publicacion (%s) y no dice cuantos dias. Arreglo: leer el articulo final del "+
				"acto; una regla a medias no se completa con un valor por defecto",
				ErrRespuestaIlegible, codigoDesdePub)
		}
		dias, err := strconv.Atoi(strings.TrimPrefix(palabras[i+1], "+"))
		if err != nil || !strings.HasPrefix(palabras[i+1], "+") {
			return fmt.Errorf("%w: la fuente dice que la entrada en vigor es «%s %s» y ese "+
				"desplazamiento no se sabe leer. Arreglo: mirar la ficha en EUR-Lex y ampliar "+
				"esta comprobacion; NO se deja pasar la fecha, porque una regla presente y no "+
				"interpretable es un dato que hay y no se entiende, no una ausencia",
				ErrRespuestaIlegible, codigoDesdePub, recortar(palabras[i+1], 20))
		}
		desde, err := time.Parse("2006-01-02", publicacion)
		if err != nil {
			return fmt.Errorf("%w: no se puede comprobar la regla de entrada en vigor porque la "+
				"fecha de publicacion (%q) no es una fecha", ErrRespuestaIlegible, recortar(publicacion, 20))
		}
		esperada := desde.AddDate(0, 0, dias).Format("2006-01-02")
		if esperada != vigor {
			return fmt.Errorf("%w: la fuente dice que el acto entra en vigor «%s +%d» y publica "+
				"el %s, o sea el %s, pero declara el %s. Son datos de la MISMA ficha "+
				"contradiciendose. Arreglo: mirar la ficha en EUR-Lex antes de usar ninguna de "+
				"las dos; escoger una seria decidir a cara o cruz sobre el reloj legal",
				ErrRespuestaIlegible, codigoDesdePub, dias, publicacion, esperada, vigor)
		}
		return nil
	}
	return nil // sin regla declarada no hay nada que contrastar
}

// palabrasDelComentario pasa el COMMENT_ON_DATE de Cellar a palabras sueltas.
// Los codigos vienen como `{MA/PART|http://...}` y lo que importa es el codigo,
// no la URL; el resto (el `+20`, el numero de articulo) viaja suelto.
func palabrasDelComentario(c string) []string {
	var out []string
	for _, campo := range strings.Fields(strings.TrimSpace(c)) {
		out = append(out, codigoAutoridad(campo))
	}
	return out
}

// codigoAutoridad saca el codigo de un valor `{CODIGO|url}`. Lo que no tenga esa
// forma se devuelve tal cual, que es como llegan el `+20` y el `71.2`.
func codigoAutoridad(v string) string {
	s := strings.TrimSpace(v)
	if !strings.HasPrefix(s, "{") {
		return s
	}
	s = strings.TrimPrefix(s, "{")
	if i := strings.Index(s, "|"); i >= 0 {
		return s[:i]
	}
	return strings.TrimSuffix(s, "}")
}
