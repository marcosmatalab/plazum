// LA CLASIFICACION DE LOS CAMPOS DE TEXTO: que campo de un paquete es prosa,
// cual es una referencia y cual una derivacion. Es lo que el linter recorre para
// aplicar la frontera legal, y es la pieza que mas crece, porque crece con cada
// campo nuevo del esquema.

package corpus

import (
	"fmt"
	"sort"
)

// tipoCampo clasifica un campo de texto libre por lo que puede llevar dentro.
type tipoCampo uint8

const (
	prosa tipoCampo = iota
	referencia
	derivacion
)

func (t tipoCampo) limite() int {
	switch t {
	case prosa:
		return LimiteTextoReferencial
	case derivacion:
		return LimiteDerivacionReferencial
	default:
		return LimiteCitaReferencial
	}
}

func (t tipoCampo) centinela() error {
	if t == prosa {
		return ErrTextoRedistribuido
	}
	return ErrCitaDesbordada
}

// campoTexto es un campo de texto libre ya localizado dentro del paquete.
type campoTexto struct {
	// Campo es la ruta canonica en el formato (Paquete.Obligaciones[].Titulo).
	// Es la que casa el test de exhaustividad con la estructura de datos.
	Campo string
	// Donde es el sitio concreto (obligacion demo.auditoria_bienal), para que
	// el error diga que fila hay que arreglar y no solo que campo.
	Donde string
	Valor string
	Tipo  tipoCampo
}

// lecturasDeVigencia clasifica los campos de las lecturas divergentes de una
// vigencia. Se escribe aparte porque Vigencia aparece en dos sitios del formato
// (la cabecera del paquete y cada obligacion) y una copia de esto en cada sitio
// seria una copia que se queda vieja.
//
// ID, Desde y Hasta son DERIVACION: un identificador nuestro y dos fechas, o sea
// la forma del dato, no el enunciado de nadie. Cita es REFERENCIA, por el mismo
// motivo exacto que HitoSpec.Alternativas[].Cita: su trabajo es senalar de donde
// sale la lectura discrepante, y sin techo ahi una "cita" se convierte en la
// transcripcion de un referencial por la puerta de atras.
func lecturasDeVigencia(prefijo, donde string, v Vigencia, uno func(string, string, string, tipoCampo)) {
	for _, l := range v.Alternativas {
		uno(prefijo+".Alternativas[].ID", donde, l.ID, derivacion)
		uno(prefijo+".Alternativas[].Desde", donde, l.Desde, derivacion)
		uno(prefijo+".Alternativas[].Hasta", donde, l.Hasta, derivacion)
		uno(prefijo+".Alternativas[].Cita", donde, l.Cita, referencia)
		// Espera es DERIVACION: es el identificador de un item nuestro, no
		// texto de nadie.
		uno(prefijo+".Alternativas[].Espera", donde, l.Espera, derivacion)
	}
}

// camposDeTexto enumera TODOS los campos de texto libre del paquete con su
// clasificacion. Es la unica lista, y el veredicto de cada campo esta escrito
// aqui al lado del campo, no en un documento aparte que nadie abre.
//
// Emite tambien los campos vacios: el linter no se entera de la diferencia
// (una cadena vacia nunca pasa de ningun limite) y el test de exhaustividad
// necesita verlos para comprobar que no falta ninguno.

func camposDeTexto(p *Paquete) []campoTexto {
	var cs []campoTexto
	uno := func(campo, donde, valor string, tipo tipoCampo) {
		cs = append(cs, campoTexto{Campo: campo, Donde: donde, Valor: valor, Tipo: tipo})
	}
	varios := func(campo, donde string, valores []string, tipo tipoCampo) {
		for _, v := range valores {
			uno(campo, donde, v, tipo)
		}
	}
	// Un mapa se recorre ordenado por clave: el linter tiene que dar los mismos
	// errores en el mismo orden en dos ejecuciones, o deja de ser comparable.
	mapa := func(campo, donde string, m map[string]string, tipo tipoCampo) {
		claves := make([]string, 0, len(m))
		for k := range m {
			claves = append(claves, k)
		}
		sort.Strings(claves)
		for _, k := range claves {
			uno(campo, donde+", clave "+k, m[k], tipo)
		}
	}

	// Cabecera del paquete. Todo referencia: el URN y la version son claves, la
	// fuente es un enlace, y la licencia es la declaracion de derechos, que es
	// justo donde el paquete TIENE que poder explicarse (el corpus de hoy gasta
	// 228 caracteres en explicar que un referencial no trae texto).
	donde := "paquete " + p.URN
	uno("Paquete.URN", donde, p.URN, referencia)
	uno("Paquete.Version", donde, p.Version, referencia)
	uno("Paquete.Licencia", donde, p.Licencia, referencia)
	// LicenciaFuente es vocabulario cerrado, o sea que su longitud la decide
	// este fichero y no el paquete. Se emite igual para que el control de
	// exhaustividad lo vea: un campo del formato que no aparece aqui es un
	// campo que la frontera legal no mira.
	uno("Paquete.LicenciaFuente", donde, string(p.LicenciaFuente), referencia)
	// Atribucion es REFERENCIA por la misma razon que Licencia: es la
	// declaracion de derechos, que es justo donde el paquete tiene que poder
	// explicarse. Lleva techo, no barra libre.
	uno("Paquete.Atribucion", donde, p.Atribucion, referencia)
	// El identificador de la fuente. Todo REFERENCIA: un tipo de vocabulario
	// cerrado, un localizador, una clave de catalogo y el motivo por el que un
	// editor no tiene identificador. Ninguno es sitio para el enunciado de un
	// control, y los cuatro llevan techo igual.
	uno("Paquete.Identificador.Tipo", donde, string(p.Identificador.Tipo), referencia)
	uno("Paquete.Identificador.Valor", donde, p.Identificador.Valor, referencia)
	uno("Paquete.Identificador.Registro", donde, p.Identificador.Registro, referencia)
	uno("Paquete.Identificador.Motivo", donde, p.Identificador.Motivo, referencia)
	// El campo del formato viejo se mira igual mientras siga en el tipo: un
	// campo que se lee y no se clasifica es un campo que la frontera legal no
	// vigila, aunque su unico destino sea el error del linter.
	uno("Paquete.FuenteHeredada", donde, p.FuenteHeredada, referencia)
	uno("Paquete.Vigencia.Desde", donde, p.Vigencia.Desde, referencia)
	// EL BLOQUE DE TRANSPOSICION, con sus tres tipos bien repartidos.
	//
	// La `cita` y la `norma` son REFERENCIA: senalan un articulo y una norma
	// nacional, y sin techo ahi una «cita» se convierte en la transcripcion de
	// media directiva. Las dos fechas y el pais son DERIVACION, o sea la forma
	// del dato. Y los dos campos largos son PROSA porque son nuestros: `como` es
	// el metodo con el que alguien fue a mirar el boletin de un pais, y
	// `vincula_mientras` es la decision de que se le dice al cliente entre tanto.
	if t := p.Transposicion; t != nil {
		uno("Paquete.Transposicion.Cita", donde, t.Cita, referencia)
		uno("Paquete.Transposicion.LimiteAdopcion", donde, t.LimiteAdopcion, derivacion)
		uno("Paquete.Transposicion.LimiteAplicacion", donde, t.LimiteAplicacion, derivacion)
		for _, e := range t.Estado {
			d := donde + ", transposicion en " + e.Pais
			uno("Paquete.Transposicion.Estado[].Pais", d, e.Pais, derivacion)
			uno("Paquete.Transposicion.Estado[].Norma", d, e.Norma, referencia)
			uno("Paquete.Transposicion.Estado[].VinculaMientras", d, e.VinculaMientras, prosa)
			uno("Paquete.Transposicion.Estado[].Comprobado", d, e.Comprobado, derivacion)
			uno("Paquete.Transposicion.Estado[].Como", d, e.Como, prosa)
		}
	}
	// Origen es vocabulario cerrado de dos valores: al limite mas estrecho,
	// por lo mismo que OrigenDelIntervalo.
	uno("Paquete.Vigencia.Origen", donde, p.Vigencia.Origen, prosa)
	uno("Paquete.Vigencia.Hasta", donde, p.Vigencia.Hasta, referencia)
	lecturasDeVigencia("Paquete.Vigencia", donde, p.Vigencia, uno)
	varios("Paquete.Escalas[]", donde, p.Escalas, referencia)

	for _, te := range p.Entidades {
		d := "entidad " + te.Nombre
		uno("Paquete.Entidades[].Nombre", d, te.Nombre, referencia)
		// Descripcion es PROSA: se ensena en el formulario y cabe entera la
		// definicion de alcance de un catalogo de pago.
		uno("Paquete.Entidades[].Descripcion", d, te.Descripcion, prosa)
		for _, a := range te.Atributos {
			da := d + ", atributo " + a.Nombre
			uno("Paquete.Entidades[].Atributos[].Nombre", da, a.Nombre, referencia)
			varios("Paquete.Entidades[].Atributos[].Valores[]", da, a.Valores, referencia)
			uno("Paquete.Entidades[].Atributos[].Escala", da, a.Escala, referencia)
			// Ayuda es PROSA, y es el campo mas tentador de todos: explicar un
			// control copiando el control es lo que sale solo al autorar.
			uno("Paquete.Entidades[].Atributos[].Ayuda", da, a.Ayuda, prosa)
			uno("Paquete.Entidades[].Atributos[].Cita", da, a.Cita, referencia)
			if a.Hecho != nil {
				// EL PUENTE. Forma y Predicado son IDENTIFICADORES (un
				// vocabulario cerrado y el nombre de un predicado del propio
				// paquete), asi que van como referencia: no pueden traer texto
				// normativo de nadie. Porque SI es prosa nuestra, y por eso
				// pasa por el limite de la frontera legal como cualquier otra.
				uno("Paquete.Entidades[].Atributos[].Hecho.Forma", da, a.Hecho.Forma, referencia)
				uno("Paquete.Entidades[].Atributos[].Hecho.Predicado", da, a.Hecho.Predicado, referencia)
				uno("Paquete.Entidades[].Atributos[].Hecho.Porque", da, a.Hecho.Porque, prosa)
				uno("Paquete.Entidades[].Atributos[].Hecho.Valor", da, a.Hecho.Valor, referencia)
			}
		}
	}

	for _, q := range p.Preguntas {
		d := "pregunta " + q.ID
		uno("Paquete.Preguntas[].ID", d, q.ID, referencia)
		// Texto y Ayuda son PROSA: una pregunta de alcance se escribe con
		// palabras propias, no transcribiendo el requisito que la motiva.
		uno("Paquete.Preguntas[].Texto", d, q.Texto, prosa)
		uno("Paquete.Preguntas[].Ayuda", d, q.Ayuda, prosa)
		uno("Paquete.Preguntas[].Cita", d, q.Cita, referencia)
		uno("Paquete.Preguntas[].Entidad", d, q.Entidad, referencia)
		uno("Paquete.Preguntas[].Atributo", d, q.Atributo, referencia)
		varios("Paquete.Preguntas[].Desbloquea[]", d, q.Desbloquea, referencia)
	}

	for _, o := range p.Obligaciones {
		d := "obligacion " + o.ID
		uno("Paquete.Obligaciones[].ID", d, o.ID, referencia)
		// Articulo es PROSA aunque parezca un localizador: en el corpus real
		// lleva dentro la etiqueta del control ("Anexo II 4.2.5 Mecanismo de
		// autenticacion (usuarios externos) [op.acc.5]"), o sea que un catalogo
		// de pago cabria ahi tal cual.
		uno("Paquete.Obligaciones[].Articulo", d, o.Articulo, prosa)
		// El fragmento es DERIVACION: un fragmento del esquema, o sea la forma del
		// localizador, no el enunciado de nadie. Al limite mas estrecho, que es
		// lo correcto para algo que acaba detras de una almohadilla.
		uno("Paquete.Obligaciones[].Fragmento", d, o.Fragmento, derivacion)
		uno("Paquete.Obligaciones[].Titulo", d, o.Titulo, prosa)
		uno("Paquete.Obligaciones[].TextoLegal", d, o.TextoLegal, prosa)
		uno("Paquete.Obligaciones[].Cita", d, o.Cita, referencia)
		// LAS VERSIONES LINGUISTICAS PASAN LA MISMA FRONTERA LEGAL QUE EL TEXTO
		// CASTELLANO, y eso es la mitad que hace seguro a D-25: si el `Texto` no
		// entrara aqui como `prosa`, la version inglesa seria la puerta de atras
		// por la que vuelve a entrar el texto de un catalogo de pago. Un
		// referencial no puede tener version inglesa larga por la misma razon por
		// la que no puede tener texto castellano largo.
		//
		// El recorrido va ORDENADO POR LENGUA, como el resto de los mapas de esta
		// funcion: el linter tiene que dar los mismos errores en el mismo orden en
		// dos ejecuciones o deja de ser comparable.
		for _, lengua := range lenguasOrdenadas(o.VersionesLinguisticas) {
			ver := o.VersionesLinguisticas[lengua]
			dv := d + ", version " + lengua
			uno("Paquete.Obligaciones[].VersionesLinguisticas[].Texto", dv, ver.Texto, prosa)
			uno("Paquete.Obligaciones[].VersionesLinguisticas[].Enlace", dv, ver.Enlace, referencia)
			uno("Paquete.Obligaciones[].VersionesLinguisticas[].Celex", dv, ver.Celex, referencia)
			uno("Paquete.Obligaciones[].VersionesLinguisticas[].Consultado", dv, ver.Consultado, referencia)
		}
		uno("Paquete.Obligaciones[].Vigencia.Desde", d, o.Vigencia.Desde, referencia)
		uno("Paquete.Obligaciones[].Vigencia.Origen", d, o.Vigencia.Origen, prosa)
		uno("Paquete.Obligaciones[].Vigencia.Hasta", d, o.Vigencia.Hasta, referencia)
		lecturasDeVigencia("Paquete.Obligaciones[].Vigencia", d, o.Vigencia, uno)
		uno("Paquete.Obligaciones[].Entregable", d, o.Entregable, referencia)
		uno("Paquete.Obligaciones[].Delegado", d, o.Delegado, referencia)
		uno("Paquete.Obligaciones[].ClaseE2E", d, o.ClaseE2E, referencia)
		varios("Paquete.Obligaciones[].Facetas[]", d, o.Facetas, referencia)
		varios("Paquete.Obligaciones[].Preguntas[]", d, o.Preguntas, referencia)
		for _, r := range o.Recursos {
			uno("Paquete.Obligaciones[].Recursos[]", d, string(r), referencia)
		}
		if t := o.Temporalidad; t != nil {
			uno("Paquete.Obligaciones[].Temporalidad.Primitiva", d, t.Primitiva, referencia)
			uno("Paquete.Obligaciones[].Temporalidad.Hito", d, t.Hito, referencia)
			uno("Paquete.Obligaciones[].Temporalidad.Cadencia", d, t.Cadencia, referencia)
			uno("Paquete.Obligaciones[].Temporalidad.Limite", d, t.Limite, referencia)
			// Suelo es una duracion ISO-8601 y Ampliacion el NOMBRE de un hecho:
			// los dos son referencia, como Limite y Cadencia. No hay lectura en la
			// que el enunciado de un control quepa en un identificador de hecho, y
			// el limite estrecho deja escrito que no se espera que quepa.
			uno("Paquete.Obligaciones[].Temporalidad.Suelo", d, t.Suelo, referencia)
			uno("Paquete.Obligaciones[].Temporalidad.Ampliacion", d, t.Ampliacion, referencia)
			// Efecto es el NOMBRE de un hecho y Antelacion una duracion ISO-8601:
			// referencia los dos, como sus hermanos del maximo.
			uno("Paquete.Obligaciones[].Temporalidad.Efecto", d, t.Efecto, referencia)
			uno("Paquete.Obligaciones[].Temporalidad.Antelacion", d, t.Antelacion, referencia)
			// OrigenDelIntervalo va al limite MAS ESTRECHO de los tres, y no
			// porque le haga falta: sus tres valores posibles son de once
			// caracteres. Un vocabulario cerrado no puede llevar dentro el
			// enunciado de nadie, asi que el limite estrecho no cuesta nada y
			// deja el campo clasificado por lo que es en vez de por lo que
			// cabe.
			uno("Paquete.Obligaciones[].Temporalidad.OrigenDelIntervalo", d, t.OrigenDelIntervalo, prosa)
			// CitaDelIntervalo es una CITA: el articulo que da el numero y las
			// palabras que lo dan. Mismo trato que las demas citas.
			uno("Paquete.Obligaciones[].Temporalidad.CitaDelIntervalo", d, t.CitaDelIntervalo, referencia)
			// JustificacionDelIntervalo es DERIVACION, y es la clasificacion
			// que hay que argumentar de las tres. No es texto de la fuente: es
			// el razonamiento de plazum sobre por que ESE numero, y por eso
			// vive justamente en los paquetes referenciales, donde no hay texto
			// que citar. Es la misma familia que la cita_del_esperado de un
			// dorado (la cuenta dia a dia, que tambien es nuestra) y lleva su
			// mismo techo. Con el limite de prosa no cabria un argumento, y un
			// argumento que no cabe se convierte en una etiqueta, que es
			// exactamente lo que el suelo de caracteres existe para impedir.
			uno("Paquete.Obligaciones[].Temporalidad.JustificacionDelIntervalo", d, t.JustificacionDelIntervalo, derivacion)
			// Misma familia que la justificacion: razonamiento de plazum, no
			// texto de la fuente.
			uno("Paquete.Obligaciones[].Temporalidad.CadenciaDistintaPorque", d, t.CadenciaDistintaPorque, derivacion)
			// Misma familia: razonamiento de plazum sobre su propio numero.
			uno("Paquete.Obligaciones[].Temporalidad.CuandoCambiarlo", d, t.CuandoCambiarlo, derivacion)
			// Cada fuente es una CITA: identifica un documento y dice donde
			// mirar. No cabe ahi el texto de lo citado.
			varios("Paquete.Obligaciones[].Temporalidad.FuentesDelIntervalo[]", d,
				t.FuentesDelIntervalo, referencia)
			// Un nombre de hecho es una REFERENCIA: identifica un dato del
			// alcance, no lleva enunciado de nadie.
			varios("Paquete.Obligaciones[].Temporalidad.ReabrePor[]", d, t.ReabrePor, referencia)
			// En es DERIVACION: una fecha, o sea la forma del dato. No cabe ahi
			// el enunciado de nadie.
			uno("Paquete.Obligaciones[].Temporalidad.En", d, t.En, derivacion)
			uno("Paquete.Obligaciones[].Temporalidad.Regimen.Computo", d, t.Regimen.Computo, referencia)
			uno("Paquete.Obligaciones[].Temporalidad.Regimen.Cierre", d, t.Regimen.Cierre, referencia)
			uno("Paquete.Obligaciones[].Temporalidad.Regimen.Traslado", d, t.Regimen.Traslado, referencia)
			// Los hitos de un plazo escalonado. Casi todo DERIVACION: son
			// identificadores nuestros y duraciones ISO-8601, o sea la forma
			// del reloj y no el enunciado de nadie. Nada de esto es sitio donde
			// pueda colarse el texto de un catalogo de pago.
			//
			// Las dos excepciones estan pensadas y son las que importan:
			//
			//	- Nota es PROSA, porque es donde el autor del paquete explica
			//	  una decision de lectura ("la norma da dos cifras y no dice
			//	  cual"). Es nuestra prosa sobre la norma, no la norma.
			//	- Alternativas[].Cita es REFERENCIA, porque su trabajo es
			//	  senalar donde dice la norma la lectura discrepante. Sin techo
			//	  ahi, una "cita" se convierte en la transcripcion de un
			//	  referencial por la puerta de atras.
			for _, h := range t.Hitos {
				uno("Paquete.Obligaciones[].Temporalidad.Hitos[].ID", d, h.ID, derivacion)
				uno("Paquete.Obligaciones[].Temporalidad.Hitos[].Limite", d, h.Limite, derivacion)
				uno("Paquete.Obligaciones[].Temporalidad.Hitos[].DesdeHito", d, h.DesdeHito, derivacion)
				uno("Paquete.Obligaciones[].Temporalidad.Hitos[].Clase", d, h.Clase, derivacion)
				uno("Paquete.Obligaciones[].Temporalidad.Hitos[].Nota", d, h.Nota, prosa)
				for _, a := range h.Alternativas {
					uno("Paquete.Obligaciones[].Temporalidad.Hitos[].Alternativas[].ID", d, a.ID, derivacion)
					uno("Paquete.Obligaciones[].Temporalidad.Hitos[].Alternativas[].Limite", d, a.Limite, derivacion)
					uno("Paquete.Obligaciones[].Temporalidad.Hitos[].Alternativas[].Cita", d, a.Cita, referencia)
				}
				if hr := h.Regimen; hr != nil {
					uno("Paquete.Obligaciones[].Temporalidad.Hitos[].Regimen.Computo", d, hr.Computo, referencia)
					uno("Paquete.Obligaciones[].Temporalidad.Hitos[].Regimen.Cierre", d, hr.Cierre, referencia)
					uno("Paquete.Obligaciones[].Temporalidad.Hitos[].Regimen.Traslado", d, hr.Traslado, referencia)
				}
				if tp := h.Tope; tp != nil {
					uno("Paquete.Obligaciones[].Temporalidad.Hitos[].Tope.Desde", d, tp.Desde, derivacion)
					uno("Paquete.Obligaciones[].Temporalidad.Hitos[].Tope.Limite", d, tp.Limite, derivacion)
					// La cita del tope es REFERENCIA por el mismo motivo que la
					// de una lectura alternativa: senala donde dice la norma el
					// segundo plazo, y sin techo se convierte en transcripcion.
					uno("Paquete.Obligaciones[].Temporalidad.Hitos[].Tope.Cita", d, tp.Cita, referencia)
				}
			}
			mapa("Paquete.Obligaciones[].Temporalidad.Disparador[]", d, t.Disparador, referencia)
		}
		for _, esc := range o.Escalado {
			uno("Paquete.Obligaciones[].Escalado[].Tras", d, esc.Tras, referencia)
			uno("Paquete.Obligaciones[].Escalado[].A", d, esc.A, referencia)
		}
	}

	// LAS PRUEBAS. Clasificacion campo a campo, con su motivo al lado, que es lo
	// que este linter existe para forzar:
	//
	//	ID, Obligacion   REFERENCIA: identificadores nuestros que apuntan a algo.
	//	Recurso          REFERENCIA: vocabulario del producto, no de nadie.
	//	TTL, SLA, Activa DERIVACION: son la forma del dato (una duracion, una
	//	                 fecha), no el enunciado de nadie.
	//	Predicado        PROSA, y es el que de verdad importa de esta lista. Es
	//	                 exactamente el sitio donde alguien pega el enunciado de
	//	                 un control de un catalogo de pago creyendo que ayuda:
	//	                 «A.5.15 Control de acceso: la organizacion debera...».
	//	                 Con el limite de prosa, ese pegado no carga.
	for _, pr := range p.Pruebas {
		d := "prueba " + pr.ID
		uno("Paquete.Pruebas[].ID", d, pr.ID, referencia)
		uno("Paquete.Pruebas[].Obligacion", d, pr.Obligacion, referencia)
		uno("Paquete.Pruebas[].Recurso", d, string(pr.Recurso), referencia)
		uno("Paquete.Pruebas[].TTL", d, pr.TTL, derivacion)
		uno("Paquete.Pruebas[].SLA", d, pr.SLA, derivacion)
		uno("Paquete.Pruebas[].Activa", d, pr.Activa, derivacion)
		uno("Paquete.Pruebas[].Predicado", d, pr.Predicado, prosa)
	}

	for _, r := range p.Roles {
		d := "figura " + r.ID
		uno("Paquete.Roles[].ID", d, r.ID, referencia)
		// Titulo y Descripcion son PROSA: describen una figura con palabras, y
		// ahi cabe el enunciado de un rol de un catalogo de estrato cerrado
		// igual que en el titulo de una obligacion.
		uno("Paquete.Roles[].Titulo", d, r.Titulo, prosa)
		uno("Paquete.Roles[].Descripcion", d, r.Descripcion, prosa)
		uno("Paquete.Roles[].Justificacion", d, r.Justificacion, prosa)
		uno("Paquete.Roles[].Cita", d, r.Cita, referencia)
		uno("Paquete.Roles[].Origen", d, r.Origen, referencia)
	}

	for _, pl := range p.Plantillas {
		d := "plantilla " + pl.ID
		uno("Paquete.Plantillas[].ID", d, pl.ID, referencia)
		// Titulo de plantilla es PROSA: es el nombre del entregable que ve el
		// auditor, y ahi cabe el enunciado del requisito que lo pide.
		uno("Paquete.Plantillas[].Titulo", d, pl.Titulo, prosa)
		uno("Paquete.Plantillas[].Cita", d, pl.Cita, referencia)
		for _, c := range pl.Campos {
			uno("Paquete.Plantillas[].Campos[].Nombre", d, c.Nombre, referencia)
			uno("Paquete.Plantillas[].Campos[].Origen", d, c.Origen, referencia)
		}
	}

	for _, dor := range p.Dorados {
		d := "dorado " + dor.Caso
		// Caso es PROSA: describe el supuesto con palabras propias. Los dorados
		// viajan dentro del paquete, asi que cuentan para la frontera.
		uno("Paquete.Dorados[].Caso", d, dor.Caso, prosa)
		uno("Paquete.Dorados[].Obligacion", d, dor.Obligacion, referencia)
		mapa("Paquete.Dorados[].Hechos[]", d, dor.Hechos, referencia)
		// Hasta es DERIVACION: una fecha, o sea la forma del dato.
		uno("Paquete.Dorados[].Hasta", d, dor.Hasta, derivacion)
		// Las tres columnas del conjunto esperado son REFERENCIA y ninguna es
		// prosa: un identificador de hito, una fecha y una palabra de un
		// vocabulario cerrado de tres. Ninguna es sitio para el enunciado de un
		// control, y las tres llevan techo igual, porque "identificador de
		// hito" es una cadena libre como cualquier otra.
		for _, e := range dor.Esperado {
			uno("Paquete.Dorados[].Esperado[].Hito", d, e.Hito, referencia)
			uno("Paquete.Dorados[].Esperado[].Vence", d, e.Vence, referencia)
			uno("Paquete.Dorados[].Esperado[].Estado", d, e.Estado, referencia)
		}
		// SubconjuntoPorque es DERIVACION, el SEGUNDO campo de esa clase: es el
		// razonamiento del autor sobre su propio caso, igual que la
		// cita_del_esperado, no texto transcrito de nadie. Con el techo de
		// referencia (300) la renuncia habria que escribirla en telegrama justo
		// donde el formato esta pidiendo un argumento. Y lleva ademas suelo,
		// que es lo que ningun otro campo tiene: MinimoDelMotivoDeSubconjunto.
		uno("Paquete.Dorados[].SubconjuntoPorque", d, dor.SubconjuntoPorque, derivacion)
		// CitaDelEsperado es DERIVACION: el porque de la fecha esperada, con la
		// cuenta hecha. Techo propio y alto, ver LimiteDerivacionReferencial.
		uno("Paquete.Dorados[].CitaDelEsperado", d, dor.CitaDelEsperado, derivacion)
	}

	// La aplicabilidad, que ESTABA FUERA DE LA FRONTERA. Es el P1 numero 1 del
	// corpus por su otra mitad: el limite se amplio a los veinte y pico campos
	// del formato, y el bloque de reglas se quedo fuera del barrido porque vive
	// en otro fichero y tiene su propio linter. Su linter comprueba que la
	// regla se PARSEA; no comprueba cuanto texto lleva dentro, y una regla es
	// una cadena libre con literales dentro:
	//
	//	aplica("<aqui cabe el enunciado entero de un control de pago>", S) :- ...
	//
	// Todas son REFERENCIA y ninguna es prosa: un identificador de predicado,
	// una cita de articulo, un nombre de escala y una regla en un dialecto
	// formal son localizadores, no texto escrito para leerse. La regla mas
	// larga del corpus de hoy gasta 116 bytes, o sea que el techo de 300 no
	// aprieta a nadie legitimo y si corta el parrafo copiado.
	donde = "aplicabilidad de " + p.URN
	varios("Paquete.Aplicabilidad.Exporta[]", donde, p.Aplicabilidad.Exporta, referencia)
	for i, rs := range p.Aplicabilidad.Reglas {
		d := "regla " + rs.ID
		if rs.ID == "" {
			d = fmt.Sprintf("regla %d (sin id)", i)
		}
		uno("Paquete.Aplicabilidad.Reglas[].ID", d, rs.ID, referencia)
		uno("Paquete.Aplicabilidad.Reglas[].Cita", d, rs.Cita, referencia)
		uno("Paquete.Aplicabilidad.Reglas[].Regla", d, rs.Regla, referencia)
		uno("Paquete.Aplicabilidad.Reglas[].Agregado", d, rs.Agregado, referencia)
		uno("Paquete.Aplicabilidad.Reglas[].Sobre", d, rs.Sobre, referencia)
		if e := rs.Escala; e != nil {
			uno("Paquete.Aplicabilidad.Reglas[].Escala.Nombre", d, e.Nombre, referencia)
			varios("Paquete.Aplicabilidad.Reglas[].Escala.Orden[]", d, e.Orden, referencia)
		}
	}
	return cs
}
