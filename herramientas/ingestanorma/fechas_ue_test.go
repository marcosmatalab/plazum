package main

// LAS TRES FECHAS DE LA UE, SIN RED. Los fixtures son fichas `notice=branch`
// REALES de Cellar recortadas a los elementos que el parser usa:
//
//	eurlex-fechas.xml               Reglamento (UE) 2024/2847 (CRA). Diario nuevo
//	                                (la publicacion va en la obra), entrada en
//	                                vigor con regla «DATPUB +20» y TRES hitos de
//	                                aplicacion, dos de ellos parciales
//	eurlex-fechas-diario-viejo.xml  Reglamento (UE) 2016/679 (RGPD). Diario
//	                                anterior a octubre de 2023: la publicacion no
//	                                esta en la obra, esta en el numero del Diario
//	                                al que enlaza. Vigor 2016 y aplicacion 2018
//	eurlex-fechas-doble-papel.xml   Reglamento (UE) 2023/1114 (MiCA). Una misma
//	                                fecha (29-06-2023) hace DOS papeles y Cellar
//	                                lo escribe como dos ANNOTATION en el mismo
//	                                elemento: aplicacion parcial (art. 149.4) y
//	                                entrada en vigor (art. 149.1)
//	eurlex-fechas-centinela.xml     Reglamento (UE) 2017/745 (MDR). Trae un hito
//	                                con `1001-01-01`, que es como Cellar escribe
//	                                «no consta», y otro con dos anotaciones
//
// Los cuatro casos son del corpus de HOY, no inventados: el que no se pudiera
// leer dejaria una norma del corpus con la vigencia sin contrastar por nadie.
//
// Y ESTE FICHERO YA SE COBRO SU PIEZA. La primera version leia solo la PRIMERA
// anotacion de cada hito, y el fixture de MiCA la puso roja: salia una entrada
// en vigor donde el autor creia que la fuente no decia ninguna. El error no
// estaba en el codigo, estaba en lo que se creia que decia la fuente, y por eso
// ninguna mutacion lo habria encontrado. Es la regla de la casa: una puerta se
// estrena contra el corpus REAL antes que contra una mutacion.

import (
	"errors"
	"strings"
	"testing"
)

func fechasDe(t *testing.T, fixture, celex string) FechasUE {
	t.Helper()
	f, err := parsearFechasCellar(leerFixture(t, fixture), celex)
	if err != nil {
		t.Fatalf("%s: %v", fixture, err)
	}
	return f
}

// mutar cambia un trozo del fixture en memoria. Sirve para las entradas
// adversarias: el fichero de testdata no se toca, asi que ninguna mutacion se
// queda puesta por descuido.
func mutar(t *testing.T, fixture, viejo, nuevo string) []byte {
	t.Helper()
	s := string(leerFixture(t, fixture))
	if !strings.Contains(s, viejo) {
		t.Fatalf("la mutacion no casa con nada en %s: %q. Una mutacion que no se aplica da verde "+
			"y parece un hallazgo", fixture, viejo)
	}
	return []byte(strings.Replace(s, viejo, nuevo, 1))
}

// LAS TRES FECHAS SALEN POR SEPARADO Y CON SU NOMBRE.
func TestLaFichaDeCellarDaLasTresFechasDeUnActoPorSeparado(t *testing.T) {
	f := fechasDe(t, "eurlex-fechas.xml", "32024R2847")
	// Las tres son distintas y ninguna es la otra. Entre el acto y la
	// publicacion hay veintiocho dias, y entre la publicacion y el vigor veinte.
	if f.Acto != "2024-10-23" {
		t.Errorf("fecha del acto %q", f.Acto)
	}
	if f.Publicacion != "2024-11-20" {
		t.Errorf("fecha de publicacion %q", f.Publicacion)
	}
	if f.Vigor != "2024-12-10" {
		t.Errorf("entrada en vigor %q", f.Vigor)
	}
	if f.MotivoSinVigor != "" {
		t.Errorf("hay entrada en vigor y sobra el motivo: %q", f.MotivoSinVigor)
	}
	// Y LA APLICACION ESCALONADA, que es lo que de verdad obliga a un cliente: el
	// CRA entra en vigor en 2024 y su art. 14 no se aplica hasta septiembre de
	// 2026. Un producto que solo mirara la entrada en vigor pondria en el
	// calendario, desde 2024, filas que nadie tiene que cumplir todavia.
	if len(f.Aplicacion) != 3 {
		t.Fatalf("la fuente declara tres hitos de aplicacion y salieron %d: %+v",
			len(f.Aplicacion), f.Aplicacion)
	}
	porFecha := map[string]AplicacionUE{}
	for _, a := range f.Aplicacion {
		porFecha[a.Desde] = a
	}
	total, hay := porFecha["2027-12-11"]
	if !hay || total.Alcance != "todo el acto" {
		t.Errorf("el hito de 2027-12-11 tenia que alcanzar a todo el acto: %+v", porFecha)
	}
	parcial, hay := porFecha["2026-09-11"]
	if !hay || parcial.Alcance != "parte del acto" {
		t.Errorf("el hito de 2026-09-11 es PARCIAL en la fuente (MA/PART): %+v", porFecha)
	}
	// El apoyo es el articulo que la fuente cita, para poder ir a leerlo.
	if parcial.Apoyo != "71.2" {
		t.Errorf("apoyo %q: la fuente lo cuelga del art. 71.2", parcial.Apoyo)
	}
}

// EL DIARIO ANTERIOR A OCTUBRE DE 2023 PONE LA PUBLICACION EN OTRO SITIO.
//
// No es un detalle de formato: si solo se mirara la obra, SEIS de las diez
// normas de la UE del corpus se quedarian sin fecha de publicacion, y con ella
// se pierde el unico contraste que caza la conflacion del invariante 10.
func TestLaFechaDePublicacionSaleTambienDelDiarioViejo(t *testing.T) {
	f := fechasDe(t, "eurlex-fechas-diario-viejo.xml", "32016R0679")
	if f.Publicacion != "2016-05-04" {
		t.Fatalf("publicacion %q: en el Diario viejo va dentro del enlace al numero del Diario, "+
			"no en la obra", f.Publicacion)
	}
	if f.Vigor != "2016-05-24" {
		t.Errorf("entrada en vigor %q", f.Vigor)
	}
	// Y LA DISTANCIA QUE IMPORTA: el RGPD entro en vigor en 2016 y no obligo a
	// nadie hasta 2018. Escribir la de vigor donde va la de aplicacion pone en el
	// calendario de un cliente dos anos de filas que no le obligaban.
	if len(f.Aplicacion) != 1 || f.Aplicacion[0].Desde != "2018-05-25" {
		t.Fatalf("aplicacion %+v", f.Aplicacion)
	}
	if f.Aplicacion[0].Desde == f.Vigor {
		t.Error("vigor y aplicacion han salido iguales, que es justo lo que este parser separa")
	}
}

// UNA FECHA PUEDE HACER DOS PAPELES, Y LOS DOS SE LEEN.
//
// # La medida falsa que este test corrigio
//
// Cellar escribe los papeles de una fecha como ANNOTATION, y pone VARIAS en el
// mismo elemento cuando la fecha hace varios. El 29-06-2023 de MiCA es la
// entrada en vigor (art. 149.1) y a la vez el escalon de aplicacion de unos
// articulos (art. 149.4). Un lector que se quede con la primera anotacion ve
// solo el segundo papel y concluye que la norma NO tiene entrada en vigor, que
// es lo que se llego a escribir en la exploracion de esta sesion sobre DOS de
// las diez normas de la UE del corpus. Las diez la tienen.
//
// Es la forma mas cara del silencio: no da error, da un hueco creible.
func TestUnaFechaQueHaceDosPapelesSeLeeEnLosDos(t *testing.T) {
	f := fechasDe(t, "eurlex-fechas-doble-papel.xml", "32023R1114")
	if f.Vigor != "2023-06-29" {
		t.Fatalf("entrada en vigor %q: la fuente la anota en la SEGUNDA anotacion del hito del "+
			"29-06-2023, y leer solo la primera la esconde", f.Vigor)
	}
	if f.MotivoSinVigor != "" {
		t.Errorf("hay entrada en vigor y sobra el motivo: %q", f.MotivoSinVigor)
	}
	// Y la MISMA fecha sigue estando entre los escalones de aplicacion, con su
	// alcance parcial: los dos papeles, no uno.
	parcial := false
	for _, a := range f.Aplicacion {
		if a.Desde == "2023-06-29" && a.Alcance == "parte del acto" && a.Apoyo == "149.4" {
			parcial = true
		}
	}
	if !parcial {
		t.Errorf("el papel de aplicacion parcial del 29-06-2023 se ha perdido al leer el de "+
			"entrada en vigor: %+v", f.Aplicacion)
	}
	if len(f.Aplicacion) != 3 {
		t.Errorf("tres escalones de aplicacion y salieron %d: %+v", len(f.Aplicacion), f.Aplicacion)
	}
}

// NO HABER ENTRADA EN VIGOR NO SE RELLENA CON LA FECHA DE AL LADO.
//
// Es el reverso del invariante 8: ausente y no-interpretable son dos nadas
// distintas. Cuando la fuente sencillamente no lo dice, lo unico honesto es
// dejar el hueco CON SU MOTIVO. La tentacion es coger el hito de aplicacion mas
// antiguo, que ademas cuadraria con el articulo final del acto; y seria
// exactamente la conflacion que el invariante 10 prohibe, cometida por un
// programa en vez de por una persona.
//
// CON DATO SINTETICO, Y SE DICE: las diez normas de la UE del corpus de hoy
// traen entrada en vigor, asi que esta rama no la recorre ningun dato real. Se
// construye quitandole a MiCA la anotacion que la declara.
func TestSinEntradaEnVigorNoSeInventaUnaConLaDeAplicacion(t *testing.T) {
	crudo := mutar(t, "eurlex-fechas-doble-papel.xml",
		`{EV|http://publications.europa.eu/resource/authority/fd_335/EV}`,
		`{MA|http://publications.europa.eu/resource/authority/fd_335/MA}`)
	f, err := parsearFechasCellar(crudo, "32023R1114")
	if err != nil {
		t.Fatal(err)
	}
	if f.Vigor != "" {
		t.Fatalf("la fuente no declara entrada en vigor y ha salido %q", f.Vigor)
	}
	if f.MotivoSinVigor == "" {
		t.Fatal("un hueco sin motivo se lee como un cero: hay que decir POR QUE no hay fecha")
	}
	for _, quiero := range []string{"32023R1114", codigoEnVigor} {
		if !strings.Contains(f.MotivoSinVigor, quiero) {
			t.Errorf("el motivo no dice %q: %s", quiero, f.MotivoSinVigor)
		}
	}
	// Los hitos de aplicacion SI estan, y ninguno se ha colado como vigor.
	if len(f.Aplicacion) != 4 {
		t.Fatalf("cuatro papeles de aplicacion y salieron %d: %+v", len(f.Aplicacion), f.Aplicacion)
	}
}

// UN CENTINELA DE LA FUENTE NO ES UNA FECHA, Y TAMPOCO SE TIRA EN SILENCIO.
//
// Cellar escribe `1001-01-01` donde no consta la fecha, y lo hace HOY en el
// Reglamento 2017/745. Tragarselo pondria en el calendario un vencimiento del
// ano 1001; descartarlo callando perderia un escalon de aplicacion sin que nadie
// lo echara de menos. Se guarda el hito con la fecha vacia y la nota que lo dice.
func TestUnaFechaCentinelaDeLaFuenteNoEsUnaFechaNiSeTiraEnSilencio(t *testing.T) {
	f := fechasDe(t, "eurlex-fechas-centinela.xml", "32017R0745")
	if f.Vigor != "2017-05-25" {
		t.Errorf("entrada en vigor %q", f.Vigor)
	}
	sinFecha := 0
	for _, a := range f.Aplicacion {
		if strings.HasPrefix(a.Desde, "1001") || strings.HasPrefix(a.Desde, "9999") {
			t.Errorf("un centinela de la fuente ha pasado por fecha: %+v", a)
		}
		if a.Desde == "" {
			sinFecha++
			if a.Nota == "" {
				t.Errorf("hito sin fecha y sin nota: %+v. Un hueco callado se lee como un cero", a)
			}
			if a.Apoyo == "" {
				t.Errorf("hito sin fecha y sin el articulo que lo apoya: %+v. Sin el no hay por "+
					"donde empezar a mirarlo a mano", a)
			}
		}
	}
	if sinFecha != 1 {
		t.Errorf("el fixture trae un hito con centinela y se han contado %d", sinFecha)
	}
}

// LA MUTACION QUE IMPORTA: CAMBIAR LA FECHA Y QUE SE PONGA ROJO.
//
// La entrada en vigor no se cree a ciegas: la fuente declara la REGLA que la
// produjo (`DATPUB +20`) y se resuelve contra la fecha de publicacion. Es el
// unico sitio del recorrido donde dos datos independientes de la ficha se miran
// el uno al otro.
func TestUnaFechaDeVigorQueNoCuadraConLaReglaDeLaFuenteEsUnError(t *testing.T) {
	// La fecha correcta del CRA es 2024-12-10 (publicacion 2024-11-20 + 20).
	crudo := mutar(t, "eurlex-fechas.xml", "<VALUE>2024-12-10</VALUE>", "<VALUE>2025-03-11</VALUE>")
	_, err := parsearFechasCellar(crudo, "32024R2847")
	if !errors.Is(err, ErrRespuestaIlegible) {
		t.Fatalf("una fecha de vigor que contradice la regla de la propia fuente tiene que ser un "+
			"error y dio %v", err)
	}
	for _, quiero := range []string{"2024-11-20", "2024-12-10", "2025-03-11"} {
		if !strings.Contains(err.Error(), quiero) {
			t.Errorf("el error no dice %q, asi que no se puede ir a mirarlo: %v", quiero, err)
		}
	}
}

// Y LA CONTRARIA, QUE ES LA QUE DE VERDAD IMPORTA: UNA REGLA QUE NO SE SABE
// RESOLVER DA ERROR, NUNCA UNA FECHA PLAUSIBLE.
//
// Las dos formas, porque son dos y la que se olvida es la segunda: la regla a
// medias (dice desde donde y no cuantos dias) y la regla con un desplazamiento
// que no se entiende. Las dos son «presente y no interpretable», que es la
// tercera hermana del invariante 8 y la unica de las tres que es SIEMPRE error:
// tomarla por la nada es inventarse un valor.
func TestUnaReglaDeEntradaEnVigorQueNoSeSabeResolverEsUnError(t *testing.T) {
	const regla = `{DATPUB|http://publications.europa.eu/resource/authority/fd_335/DATPUB} +20`
	casos := []struct {
		nombre string
		nueva  string
	}{
		{"a medias: dice desde donde y no cuantos dias",
			`{DATPUB|http://publications.europa.eu/resource/authority/fd_335/DATPUB}`},
		{"desplazamiento que no se entiende",
			`{DATPUB|http://publications.europa.eu/resource/authority/fd_335/DATPUB} +veinte`},
		{"desplazamiento sin signo, que no es lo mismo que +20",
			`{DATPUB|http://publications.europa.eu/resource/authority/fd_335/DATPUB} 20`},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			crudo := mutar(t, "eurlex-fechas.xml", regla, c.nueva)
			f, err := parsearFechasCellar(crudo, "32024R2847")
			if !errors.Is(err, ErrRespuestaIlegible) {
				t.Fatalf("tenia que ser error y dio %v con la fecha %q. Una regla que no se sabe "+
					"resolver NO se completa con la fecha que venia al lado: la fecha que venia "+
					"al lado es justo lo que la regla existe para comprobar", err, f.Vigor)
			}
		})
	}
}

// LA IDENTIDAD, ANTES QUE EL DATO (invariante 7).
//
// Una ficha de otra norma trae tres fechas perfectamente formadas, y son de otro
// acto. El emparejamiento va por el CELEX que declara la propia respuesta, que
// es un campo que viaja DENTRO de lo que se descargo, no por el orden en que se
// pidio.
func TestLaFichaDeOtraNormaNoPasaPorLaDeEsta(t *testing.T) {
	_, err := parsearFechasCellar(leerFixture(t, "eurlex-fechas.xml"), "32016R0679")
	if !errors.Is(err, ErrRespuestaIlegible) {
		t.Fatalf("se pidieron las fechas del RGPD y respondio la ficha del CRA: %v", err)
	}
	if !strings.Contains(err.Error(), "32024R2847") {
		t.Errorf("el error tiene que decir de quien es la ficha que llego: %v", err)
	}
	// Y una ficha sin CELEX tampoco pasa: sin identidad no hay emparejamiento, y
	// «no dice de quien es» no es lo mismo que «es de quien pedi».
	crudo := mutar(t, "eurlex-fechas.xml", "<ID_CELEX type=\"data\">", "<ID_OTRA_COSA>")
	if _, err := parsearFechasCellar(crudo, "32024R2847"); !errors.Is(err, ErrRespuestaIlegible) {
		t.Fatalf("una ficha sin CELEX no puede emparejarse con nada: %v", err)
	}
}

// DOS ENTRADAS EN VIGOR NO SE RESUELVEN COGIENDO LA PRIMERA.
//
// Un acto entra en vigor UNA vez. Dos es una contradiccion de la fuente, y
// elegir por orden es elegir por lo que nadie garantiza.
func TestDosEntradasEnVigorNoSeResuelvenCogiendoLaPrimera(t *testing.T) {
	crudo := mutar(t, "eurlex-fechas.xml",
		`{MA|http://publications.europa.eu/resource/authority/fd_335/MA}`,
		`{EV|http://publications.europa.eu/resource/authority/fd_335/EV}`)
	_, err := parsearFechasCellar(crudo, "32024R2847")
	if !errors.Is(err, ErrRespuestaIlegible) {
		t.Fatalf("dos entradas en vigor tienen que parar la ingesta y dio %v", err)
	}
}

// UN HITO SIN TIPO NO SE REPARTE AL MONTON QUE MENOS MOLESTE.
//
// Sin TYPE_OF_DATE no se sabe si es la entrada en vigor o una fecha de
// aplicacion, y entre las dos hay dos anos en el RGPD. El valor por defecto
// comodo seria «aplicacion», que no rompe nada y esconde la fecha.
func TestUnHitoSinTipoDeFechaNoSeColocaPorDefecto(t *testing.T) {
	crudo := mutar(t, "eurlex-fechas.xml", "<TYPE_OF_DATE>", "<TIPO_QUE_NO_ES>")
	_, err := parsearFechasCellar(crudo, "32024R2847")
	if !errors.Is(err, ErrRespuestaIlegible) {
		t.Fatalf("un hito sin tipo tiene que parar la ingesta y dio %v", err)
	}
	// Y un hito SIN NINGUNA anotacion, que es la otra forma de la nada: el
	// elemento esta, la fecha esta, y no dice que papel hace.
	crudo = mutar(t, "eurlex-fechas.xml", "<ANNOTATION>", "<ANOTACION_QUE_NO_ES>")
	if _, err := parsearFechasCellar(crudo, "32024R2847"); !errors.Is(err, ErrRespuestaIlegible) {
		t.Fatalf("un hito sin anotaciones tiene que parar la ingesta y dio %v", err)
	}
	// Y un codigo que esta pero no se conoce, que es la otra mitad: descartarlo
	// en silencio perderia una fecha y nadie la echaria de menos.
	crudo = mutar(t, "eurlex-fechas.xml",
		`{EV|http://publications.europa.eu/resource/authority/fd_335/EV}`,
		`{ZZ|http://publications.europa.eu/resource/authority/fd_335/ZZ}`)
	if _, err := parsearFechasCellar(crudo, "32024R2847"); !errors.Is(err, ErrRespuestaIlegible) {
		t.Fatalf("un codigo de fecha desconocido tiene que parar la ingesta y dio %v", err)
	}
}

// LAS DOS FORMAS DE LA NADA, LAS DOS RECORRIDAS (invariante 8).
//
// `nil` y vacio-presente no son lo mismo, y la peligrosa es siempre la primera
// porque es la que sale por olvidarse. Aqui las dos tienen que dar error: una
// ficha sin fechas NO es una norma sin fechas, es una respuesta que no sirve.
func TestUnaFichaDeFechasVaciaNoDevuelveCerosPlausibles(t *testing.T) {
	casos := map[string][]byte{
		"nil":                    nil,
		"vacio presente":         []byte(``),
		"NOTICE sin WORK":        []byte(`<NOTICE type="branch"></NOTICE>`),
		"WORK sin ningun dato":   []byte(`<NOTICE type="branch"><WORK></WORK></NOTICE>`),
		"con CELEX y sin fechas": []byte(`<NOTICE type="branch"><WORK><ID_CELEX><VALUE>32024R2847</VALUE></ID_CELEX></WORK></NOTICE>`),
	}
	for nombre, b := range casos {
		t.Run(nombre, func(t *testing.T) {
			f, err := parsearFechasCellar(b, "32024R2847")
			if err == nil {
				t.Fatalf("tenia que ser error y devolvio %+v", f)
			}
			if !errors.Is(err, ErrRespuestaIlegible) {
				t.Fatalf("centinela equivocado: %v", err)
			}
		})
	}
}

// El troceador del comentario de Cellar, que es de donde salen el alcance y el
// articulo de apoyo. Se prueba aparte porque un fallo suyo no rompe nada: deja
// el alcance en «todo el acto», que es el valor comodo.
func TestElComentarioDeCellarSeTroceaEnCodigos(t *testing.T) {
	const c = `{MA/PART|http://publications.europa.eu/resource/authority/fd_335/MA%2FPART} ` +
		`{V|http://publications.europa.eu/resource/authority/fd_335/V} ` +
		`{ART|http://publications.europa.eu/resource/authority/fd_335/ART} 123.3(f)`
	got := palabrasDelComentario(c)
	quiero := []string{codigoParcial, "V", codigoArticulo, "123.3(f)"}
	if len(got) != len(quiero) {
		t.Fatalf("salieron %v", got)
	}
	for i := range quiero {
		if got[i] != quiero[i] {
			t.Fatalf("salieron %v y se esperaba %v", got, quiero)
		}
	}
	// Lo que no tiene la forma `{codigo|url}` viaja tal cual: es como llegan el
	// desplazamiento de la regla y el numero de articulo.
	if codigoAutoridad("+20") != "+20" {
		t.Errorf("el desplazamiento de la regla no puede tocarse: %q", codigoAutoridad("+20"))
	}
}
