// EL ESQUEMA DE DATOS de un paquete: los tipos que un autor de corpus escribe
// en su JSON. Lo mueve el formato, o sea la guia, y cambiarlo obliga a mirar
// todos los paquetes publicados.

package corpus

// TipoAtributo es el tipo de un atributo de entidad. La interfaz se genera de aqui.
type TipoAtributo uint8

const (
	Texto TipoAtributo = iota
	Entero
	Booleano
	Fecha
	Enumerado // usa Escala para el orden, si lo tiene
)

func (t TipoAtributo) String() string {
	return [...]string{"texto", "entero", "booleano", "fecha", "enumerado"}[t]
}

// Atributo describe un dato de una entidad. Genera un campo de formulario.
type Atributo struct {
	Nombre   string       `json:"nombre"`
	Tipo     TipoAtributo `json:"tipo"`
	Valores  []string     `json:"valores,omitempty"` // solo enumerado
	Escala   string       `json:"escala,omitempty"`  // ref a una Escala declarada
	Obligado bool         `json:"obligado"`
	Ayuda    string       `json:"ayuda,omitempty"`
	Cita     string       `json:"cita"` // de donde sale que este dato importa
	// Hecho es EL PUENTE: que afirma este atributo en el motor de
	// aplicabilidad. Ver puente.go, que explica por que lo declara el paquete
	// y no el codigo (invariante 2: es conocimiento normativo).
	//
	// PUNTERO Y OPCIONAL MIENTRAS DURA EL PILOTO. El nil de hoy significa «este
	// paquete todavia no ha declarado su puente», que no es lo mismo que «este
	// atributo no llega al motor»: eso ultimo se dice con forma
	// `no_llega_al_motor` y su motivo. Las dos formas de la nada otra vez, y
	// aqui la distincion es temporal a proposito: cuando el piloto demuestre
	// que el diseno mueve el numero, esto pasa a obligatorio y el nil rompe.
	Hecho *HechoDeAtributo `json:"hecho,omitempty"`
}

// TipoEntidad es un tipo de sujeto que el paquete introduce.
type TipoEntidad struct {
	Nombre      string     `json:"nombre"`
	Descripcion string     `json:"descripcion"`
	Atributos   []Atributo `json:"atributos"`
}

// Pregunta de alcance. La entrevista es la union de las preguntas de los
// paquetes instalados, ordenada por cuantas obligaciones desbloquea cada una.
type Pregunta struct {
	ID         string   `json:"id"`
	Texto      string   `json:"texto"`
	Cita       string   `json:"cita"`
	Entidad    string   `json:"entidad"`    // que TipoEntidad rellena
	Atributo   string   `json:"atributo"`   // que Atributo fija
	Desbloquea []string `json:"desbloquea"` // IDs de obligacion, para ordenar
	Ayuda      string   `json:"ayuda,omitempty"`
}

// CampoPlantilla es un hueco de un entregable documental.
type CampoPlantilla struct {
	Nombre string `json:"nombre"`
	// Origen dice de donde sale el valor. Debe ser derivable: un atributo de
	// entidad, un estado de control o una obligacion. Nunca texto libre de un LLM.
	Origen string `json:"origen"`
}

// Plantilla es un entregable documental versionado.
type Plantilla struct {
	ID     string           `json:"id"`
	Titulo string           `json:"titulo"`
	Cita   string           `json:"cita"`
	Campos []CampoPlantilla `json:"campos"`
}

// TipoRecurso es un recurso canonico que el paquete necesita observar.
// De aqui sale que conectores hacen falta, y cuales no aportan nada.
type TipoRecurso string

// clasesE2E son las cinco maneras de implantar una obligacion de extremo a
// extremo (guia, Anexo B). La clase primaria es obligatoria: sin ella no se
// puede medir la profundidad ni decidir la cadena de implantacion.
var clasesE2E = map[string]bool{
	"observable": true, "documental": true, "procedimental": true,
	"notificatoria": true, "remediacion": true,
}

// Temporalidad es el reloj declarado de la obligacion, como datos: la
// primitiva del motor de ventana, su cadencia o limite, y el regimen.
type Temporalidad struct {
	Primitiva string `json:"primitiva"`          // puntual|periodica|continua|plazo|observacion|secuencia
	Hito      string `json:"hito,omitempty"`     // nombre del hito (por defecto "ocurrencia" / "limite")
	Cadencia  string `json:"cadencia,omitempty"` // periodica: ISO-8601 (P24M)
	Limite    string `json:"limite,omitempty"`   // plazo: ISO-8601 (P10D, PT72H)
	// En es el instante que fija LA NORMA, para la primitiva puntual. No es un
	// hecho del obligado y no se cuenta desde nada: la fecha esta escrita en el
	// texto legal.
	//
	// El caso que lo trajo: el art. 111.4 del AI Act, anadido por el Reglamento
	// (UE) 2026/1744, obliga a los proveedores de sistemas que ya estaban en el
	// mercado a cumplir el art. 50.2 "a mas tardar el 2 de diciembre de 2026".
	// No hay disparador que valga: la fecha es esa para todos.
	//
	// Se escribe con la hora dentro (2026-12-02T23:59:59Z) porque una primitiva
	// puntual no tiene regimen y por tanto no sabe cerrar el dia. Poner solo la
	// fecha significaria vencer a las 00:00, que es un dia entero de menos.
	En string `json:"en,omitempty"`

	// Suelo es la duracion MINIMA que fija la norma en un `maximo`: la retencion
	// no puede terminar antes de aqui, pase lo que pase.
	//
	// EL CASO QUE LO TRAE. El art. 13.9 del CRA dice que cada actualizacion de
	// seguridad sigue estando disponible "un periodo minimo de diez anos tras su
	// publicacion, o durante el resto del periodo de soporte si este plazo fuera
	// mas largo". Son dos duraciones que vinculan a la vez, una fija que pone la
	// norma y otra que declara el propio obligado, y gana la mayor. Escribirlo
	// como un plazo de diez anos da una fecha MAS CORTA que la legal en el
	// sentido peligroso: el obligado tira la evidencia creyendo que ya podia.
	Suelo string `json:"suelo,omitempty"`
	// Ampliacion es el NOMBRE DEL HECHO que trae la fecha de la segunda rama, no
	// una duracion: el fin del periodo de soporte que declara el fabricante, o
	// el nuevo limite que impone una autoridad.
	//
	// Que sea un hecho y no un campo editable es deliberado y es lo mismo que ya
	// hace HitoSpec.Clase: declarar un periodo de soporte ocurre en un instante,
	// igual que ocurre un incidente, asi que va fechado a la historia y el acta
	// puede decir QUIEN lo declaro y CUANDO. Cambiarlo mas tarde es un hecho
	// nuevo que gana por ser posterior, no una correccion que borra al anterior.
	Ampliacion string `json:"ampliacion,omitempty"`
	// AmpliacionExigible dice si la norma OBLIGA al obligado a declarar la
	// ampliacion. Es un PUNTERO, y el nil es error, no un `false` comodo.
	//
	// POR QUE NO PUEDE TENER VALOR POR DEFECTO (invariante 8). Las dos
	// respuestas son opuestas y las dos son plausibles:
	//
	//	true   la ausencia del hecho NO es "no hay ampliacion", es "falta un dato
	//	       que la norma exige", y el hito sale PendienteDeHecho con el suelo
	//	       como NoAntesDe. No se presenta una fecha cerrada que puede ser
	//	       mas corta que la real.
	//	false  la ausencia si significa que no hay segunda rama, y rige el suelo.
	//
	// Elegir por omision seria acertar por casualidad, y el lado del que se
	// acierta por descuido (`false`, el valor cero de un bool) es justo el
	// PERMISIVO: colapsa al suelo en silencio y ensena una fecha cerrada donde
	// no la hay. Es la misma salida que OrigenDelIntervalo: cuando el valor cero
	// no puede ser el restrictivo, se prohibe explicitamente.
	AmpliacionExigible *bool `json:"ampliacion_exigible,omitempty"`

	// Efecto es el nombre del hecho que trae la fecha en la que la decision del
	// obligado va a SURTIR EFECTO, para la primitiva `preaviso`.
	//
	// POR QUE NO ES UN `disparador` (que es donde la tentacion lleva a meterlo).
	// Un disparador es un hecho que LE OCURRE al obligado y desde el que se
	// cuenta hacia adelante: el incidente, el conocimiento, la solicitud. Aqui
	// es al reves: la fecha la ELIGE el obligado (cuando quiere que su
	// modificacion del contrato marco surta efecto) y lo que se calcula es
	// hasta cuando puede seguir callado. Meterlo en `disparador` haria que el
	// campo significara dos cosas opuestas segun la primitiva, y entonces
	// ninguna pantalla podria explicarlo sin mirar la primitiva primero.
	//
	// La consecuencia practica: este vencimiento SE MUEVE cuando se mueve el
	// hecho. Adelantar la fecha de efecto adelanta la fecha limite de aviso y
	// puede dejarla en el pasado, que es justo lo que hay que ensenar.
	Efecto string `json:"efecto,omitempty"`
	// Antelacion es cuanto antes hay que avisar, en un `preaviso`. Es una
	// duracion, como Limite, pero corre HACIA ATRAS desde Efecto.
	Antelacion string            `json:"antelacion,omitempty"`
	Regimen    RegimenSpec       `json:"regimen"`
	Disparador map[string]string `json:"disparador,omitempty"` // p.ej. {"hecho": "ultima_auditoria"}

	// Hitos son los hitos ENCADENADOS de un plazo, para las normas que
	// escalonan la misma obligacion en varias notificaciones.
	//
	// POR QUE NO BASTA CON Hito Y Limite. Hasta el 26-08-2026 una obligacion
	// solo podia declarar UN hito con UN limite contado desde el disparador.
	// La familia A del censo (notificacion escalonada de incidente, once
	// fuentes y treinta y tres relojes) no cabe ahi por dos motivos que el
	// texto legal dice literalmente:
	//
	//	- la notificacion intermedia y la final cuentan desde la REMISION DE LA
	//	  INICIAL, no desde el incidente;
	//	- y sus limites los decide el NIVEL que asigna el propio obligado.
	//
	// Cuando Hitos viene relleno, manda; Hito y Limite se quedan para el caso
	// simple, que es la mayoria del corpus.
	Hitos []HitoSpec `json:"hitos,omitempty"`

	// OrigenDelIntervalo dice DE QUIEN es el numero de una cadencia, y es
	// obligatorio en toda `periodica`. Vocabulario cerrado, en el bloque de
	// constantes de abajo.
	//
	// POR QUE ES UN CAMPO Y NO SE DEDUCE LEYENDO LA CITA. Porque "revisar al
	// menos una vez al ano" y "revisar a intervalos planificados" son la misma
	// frase para un lector distraido y obligaciones OPUESTAS para un inspector:
	// la primera pone un techo legal al intervalo y la segunda no pone nada. La
	// diferencia decide lo unico que el cliente necesita saber, que es si puede
	// mover el numero y hacia donde. Hasta hoy se distinguia leyendo el campo
	// `articulo` (`anexo, punto N` contra `ritual plazum sobre N`), que funciona
	// entre personas que conocen el acuerdo y no es un dato: nada impedia
	// escribir un intervalo propuesto con cara de intervalo legal.
	//
	// El valor cero (cadena vacia) NO se interpreta: es error. No hay defecto
	// seguro que elegir, porque el permisivo (`propuesto`, el cliente lo mueve
	// libremente) y el restrictivo (`suelo_legal`, solo puede apretar) son las
	// dos respuestas posibles y acertar por omision seria casualidad. Es el
	// invariante 8 por su otra salida: cuando el valor cero no puede ser el
	// restrictivo, se prohibe explicitamente.
	//
	// La decision entera, con la tabla de quien puede mover que, en
	// docs/decisiones.md D-12.
	OrigenDelIntervalo string `json:"origen_del_intervalo,omitempty"`
	// CitaDelIntervalo es el articulo que DA el numero, con las palabras que lo
	// dan. Obligatoria en `suelo_legal` y en `fijado`, y prohibida en
	// `propuesto`: si el numero es nuestro, no hay articulo que citar y fingir
	// que lo hay es lo peor que puede pasar aqui.
	CitaDelIntervalo string `json:"cita_del_intervalo,omitempty"`
	// JustificacionDelIntervalo es POR QUE ESE numero y no otro, cuando lo pone
	// plazum. Obligatoria en `propuesto` y prohibida en los otros dos.
	//
	// Tiene suelo de caracteres por la misma razon que `subconjunto_porque`: un
	// numero sin argumento es un numero inventado, y una etiqueta de tres
	// palabras es un numero inventado con adorno.
	JustificacionDelIntervalo string `json:"justificacion_del_intervalo,omitempty"`
	// CadenciaDistintaPorque es la valvula de escape de la regla "mismo texto,
	// misma cadencia": dos obligaciones con el MISMO texto legal tienen que
	// llevar el mismo intervalo y el mismo origen, y si no lo llevan, TODAS las
	// del grupo tienen que decir aqui por que.
	//
	// Se pide a todas y no solo a la que se desvia porque con dos obligaciones
	// no hay una canonica: decidir cual es "la normal" seria elegir por el
	// autor. Que las dos tengan que argumentarlo es lo que convierte la
	// excepcion en una decision escrita en vez de en un descuido.
	CadenciaDistintaPorque string `json:"cadencia_distinta_porque,omitempty"`
	// CuandoCambiarlo dice bajo que supuestos el cliente deberia mover este
	// intervalo, en las DOS direcciones: una condicion para acortarlo y otra
	// para alargarlo, cada una con el supuesto que la hace cierta.
	//
	// POR QUE ES OBLIGATORIO EN `propuesto` Y NO EN LOS OTROS DOS. Porque en
	// `propuesto` el numero es NUESTRO, y un numero nuestro sin instrucciones
	// de uso es una imposicion disfrazada de dato: el cliente no sabe si puede
	// tocarlo, y ante la duda no lo toca. Es el campo que convierte un defecto
	// en un defecto ADAPTABLE, que es la diferencia entre un calendario que
	// ordena y uno que se abandona al segundo mes (D-15).
	//
	// En `suelo_legal` y en `fijado` no aplica: ahi lo que el cliente puede
	// hacer lo dice la norma, no nosotros, y ya esta en la cita.
	CuandoCambiarlo string `json:"cuando_cambiarlo,omitempty"`
	// FuentesDelIntervalo son las fuentes CITABLES en las que se apoya la
	// justificacion: una por entrada, con lo que hace falta para ir a
	// buscarla.
	//
	// POR QUE EXISTE, y no es documentacion. La frontera legal prohibe copiar
	// el texto de un marco de estrato cerrado, y el linter de prosa prohibe
	// NOMBRARLO. Ninguna de las dos caza la tercera forma, que es la que
	// aparecio de verdad: un argumento que se apoya en el criterio de un
	// catalogo de pago SIN nombrarlo ("el sector de medios de pago lleva anos
	// exigiendo revisar el conjunto de reglas cada seis meses"). Eso
	// redistribuye el CRITERIO sin copiar una palabra, y ningun linter lo
	// distingue de un razonamiento propio.
	//
	// Lo que si se puede hacer es cambiar la pregunta. Con este campo, la
	// pasada de coherencia deja de tener que leer cada frase buscando un eco y
	// pasa a preguntar una sola cosa, que se contesta mirando: **¿por que este
	// argumento no tiene fuente?** Un numero que se apoya en algo real puede
	// citarlo (NIST, ENISA, BOE, DOUE, el propio texto de la norma); uno que
	// se apoya en un apoyo fantasma, no. Las dos que salieron en la pasada de
	// cierre de las 34 (una curva de decaimiento de tasa de clic, una guia de
	// fabricante sin decir cual) se cazan asi en un vistazo.
	//
	// ES OPCIONAL A PROPOSITO. Hay intervalos que se sostienen solos sobre la
	// estructura del propio texto legal ("esto tiene que estar hecho antes que
	// aquello, y aquello es anual"), y esos no tienen fuente externa ni la
	// necesitan. Exigirlo siempre convertiria el campo en un tramite que se
	// rellena con lo primero que suene bien, que es peor que no tenerlo.
	FuentesDelIntervalo []string `json:"fuentes_del_intervalo,omitempty"`
	// ReabrePor son los hechos que REABREN el ciclo antes de que venza su
	// intervalo.
	//
	// POR QUE NO SON OBLIGACIONES APARTE, que era la otra opcion y es la que
	// duplica el trabajo. Casi todo punto de revision del anexo de 2024/2690
	// dice lo mismo: «revisaran y, cuando proceda, actualizaran X a intervalos
	// planificados O CUANDO SE PRODUZCAN INCIDENTES SIGNIFICATIVOS O CAMBIOS
	// SIGNIFICATIVOS en las operaciones o los riesgos». Eso NO crea un segundo
	// deber: crea un segundo disparador del mismo deber. Escribirlos como dos
	// obligaciones duplicaria el recuento (22 de 47 en ese anexo) y le diria al
	// cliente que tiene el doble de ceremonias de las que tiene, que es
	// exactamente el problema que D-15 existe para no empeorar.
	//
	// QUE PASA CUANDO SE REABRE, y aqui se decide lo que NO se hace. La norma
	// dice CUANDO hay que revisar (al ocurrir el hecho) y no da plazo para
	// hacerlo. Asi que el reloj reabierto sale como `sin plazo legal` y el
	// motor mide el tiempo transcurrido desde el hecho, en vez de inventarse
	// una fecha limite que el texto no fija. Es el mismo trato que ya reciben
	// las tres obligaciones sin numero del corpus, y por la misma razon.
	//
	// UN HECHO REABRE MUCHAS. `ultimo_incidente_significativo` es uno solo por
	// organizacion y reabre las 22 revisiones del anexo a la vez, que es lo que
	// de verdad pasa cuando una entidad tiene un incidente serio.
	ReabrePor []string `json:"reabre_por,omitempty"`
}

// HitoSpec es un hito de un plazo escalonado.
type HitoSpec struct {
	ID     string `json:"id"`
	Limite string `json:"limite"` // ISO-8601; vacio o "indeterminado" = la norma no fija limite
	// DesdeHito encadena: el reloj de este hito arranca cuando se CUMPLE aquel,
	// no cuando ocurre el disparador. Vacio = desde el disparador.
	DesdeHito string `json:"desde_hito,omitempty"`
	// Clase es el hecho que tiene que constar para que este hito rija. Vacio =
	// rige siempre. Es como se expresa "los plazos dependen del nivel que
	// asigne el obligado": la clasificacion es un hecho con su instante, asi
	// que una reclasificacion posterior es otro hecho y manda la mas reciente.
	Clase string `json:"clase,omitempty"`
	// Alternativas son lecturas discrepantes del mismo plazo, con su cita. El
	// motor calcula todas, usa Limite y ensena la divergencia: no elige en
	// silencio.
	Alternativas []LecturaSpec `json:"alternativas,omitempty"`
	// Tope es un SEGUNDO limite del mismo hito que corre desde otro hecho y que
	// acorta al principal cuando vence antes. Los dos vinculan a la vez.
	Tope *TopeSpec `json:"tope,omitempty"`
	// Regimen propio de ESTE hito. Nil = el de la obligacion.
	//
	// POR QUE HACE FALTA. Una notificacion escalonada mezcla plazos de dos
	// naturalezas en la MISMA obligacion: el art. 14 del CRA da 24 y 72 HORAS
	// para las dos primeras y un MES para el informe final. Y el regimen no es
	// el mismo: el art. 3.4 del Reglamento 1182/71 traslada al habil siguiente
	// el vencimiento que cae en inhabil "expresado de cualquier modo, salvo en
	// horas", y el 3.2.b hace terminar el plazo en dias o meses al expirar la
	// ultima hora del ultimo dia. O sea que las horas vencen en un instante
	// exacto sin traslado y los meses a fin de dia con traslado.
	//
	// Con un solo regimen por obligacion habia que elegir: partir la obligacion
	// en dos (y entonces el informe final no puede encadenarse al hito del que
	// cuelga, porque desde_hito no cruza obligaciones) o aplicar el regimen de
	// las horas a los meses. Lo segundo da una fecha MAS TEMPRANA que la legal,
	// que es el lado inofensivo, pero sigue siendo una fecha equivocada, y este
	// producto se vende por dar la fecha buena.
	//
	// El motor ya lo soportaba: ventana.Hito lleva su propio Regimen desde el
	// principio. Lo que faltaba era poder decirlo desde un paquete.
	Regimen *RegimenSpec `json:"regimen,omitempty"`
	Nota    string       `json:"nota,omitempty"`
}

// TopeSpec es el segundo limite de un hito, contado desde otro hecho. Ver
// ventana.Tope para el porque; aqui solo esta la forma que escribe un paquete.
type TopeSpec struct {
	Desde  string `json:"desde"`
	Limite string `json:"limite"`
	// Caduca: el valor cero (false) es el RESTRICTIVO, el tope vincula siempre.
	// Caducar hay que pedirlo, y con cita.
	Caduca bool   `json:"caduca,omitempty"`
	Cita   string `json:"cita"`
}

// LecturaSpec es una interpretacion discrepante de un plazo.
type LecturaSpec struct {
	ID     string `json:"id"`
	Limite string `json:"limite"`
	Cita   string `json:"cita"`
}

// RegimenSpec es el regimen de computo declarado por el paquete.
type RegimenSpec struct {
	Computo  string `json:"computo"`            // naturales | habiles
	Cierre   string `json:"cierre,omitempty"`   // exacto | fin_de_dia | (vacio = auto)
	Traslado string `json:"traslado,omitempty"` // ninguno | siguiente_habil
}

// Escalon es un paso de la cadena de escalado de la obligacion.
type Escalon struct {
	Tras string `json:"tras"` // ISO-8601, admite sufijo _antes (P60D_antes)
	A    string `json:"a"`    // rol destinatario
}

// Dorado es un caso de prueba derivado DEL TEXTO legal, no de la
// implementacion: si el motor y el dorado discrepan, gana el dorado.
type Dorado struct {
	Caso       string            `json:"caso"`
	Obligacion string            `json:"obligacion"`
	Hechos     map[string]string `json:"hechos"` // clave -> fecha RFC3339 o 2006-01-02

	// Esperado es el conjunto COMPLETO de vencimientos que el motor tiene que
	// devolver con esos hechos. Ni uno de menos ni uno de mas.
	//
	// POR QUE ES UNA LISTA, Y POR QUE ES EXHAUSTIVA. Hasta el 27-08-2026 el
	// esperado era UN vencimiento y el ejecutor filtraba por su hito: un dorado
	// decia lo que TIENE que salir y no decia NADA de lo que NO tiene que
	// salir. Eso deja fuera media familia de fallos, la misma de siempre
	// (invariante 7): cuando una comprobacion recorre una lista para
	// contrastarla con otra, la direccion que falta es la que muerde.
	//
	// Muerde asi, y esta medido: quitandole la clase al hito del plazo general
	// del art. 73 del AI Act, ese hito rige SIEMPRE, y un incidente con
	// fallecimiento le ensena al operador DOS fechas para la misma obligacion
	// (la del 73.4 y la del 73.2) sin ninguna forma de saber cual es la suya.
	// Los doce dorados del paquete seguian en verde, porque cada uno miraba su
	// hito. Ahora esa mutacion pone rojos varios dorados.
	//
	// EL EMPAREJAMIENTO ES POR HITO, que es una identidad DENTRO del dato, no
	// por indice ni por orden (invariante 7): reordenar la lista no puede
	// cambiar lo que se compara con que. Por eso `hito` es obligatorio en cada
	// fila y no puede repetirse dentro de un dorado.
	Esperado []EsperadoDorado `json:"esperado"`

	// Hasta es LA VENTANA dentro de la cual el esperado es exhaustivo.
	//
	// POR QUE NO SE DERIVA DE LAS FECHAS DECLARADAS, que es como nacio y estuvo
	// mal durante un dia. `hasta` acota a la primitiva `periodica`: le dice
	// cuantas ocurrencias devolver. Si el horizonte sale de la ultima fecha que
	// el propio dorado escribe, TRUNCAR LA LISTA POR LA COLA MUEVE EL HORIZONTE
	// CON ELLA, el motor deja de emitir la ocurrencia borrada, y la direccion
	// de "sobra" no tiene nada que decir. La afirmacion se hace verdadera a si
	// misma, que es justo la familia de guardas que este repositorio lleva
	// catorce entradas cazando.
	//
	// Medido, no supuesto: quitando la ultima fila de un dorado periodico, la
	// suite entera salia VERDE; quitando una fila INTERIOR del mismo fichero,
	// roja con "SOBRA". Esa asimetria era el agujero entero.
	//
	// Una ventana declarada no la mueve la respuesta, asi que truncar la cola
	// se cae por "sobra" igual que cualquier otra fila. Es obligatoria en todo
	// dorado cuya primitiva consuma el horizonte, y ahi el linter la exige: el
	// valor cero (vacio) NO puede significar "el horizonte que salga", porque
	// ese es el permisivo (invariante 8).
	Hasta string `json:"hasta,omitempty"`

	// SubconjuntoPorque renuncia a la exhaustividad, y es una CADENA CON EL
	// MOTIVO en vez de un booleano a proposito.
	//
	// El invariante 8 dice que en una frontera el valor cero tiene que ser el
	// RESTRICTIVO. El valor cero de un bool es `false`, y un `exhaustivo: bool`
	// tendria el problema al reves (olvidarse del campo aflojaria la
	// comprobacion). Con una cadena, el valor cero (vacia) significa
	// EXHAUSTIVO, que es lo duro, y relajarlo cuesta escribir por que y para
	// que hito. Ademas deja el motivo consultable en el propio dato en vez de
	// en la cabeza de quien escribio el caso.
	//
	// Solo relaja UNA de las dos direcciones: la de "sobra". Las filas
	// declaradas se siguen exigiendo todas, y con su fecha exacta.
	SubconjuntoPorque string `json:"subconjunto_porque,omitempty"`

	CitaDelEsperado string `json:"cita_del_esperado"`
}

// EsperadoDorado es UNA fila del conjunto esperado: un hito y lo que la norma
// dice de el.
//
// LOS ESTADOS CUENTAN. Un vencimiento "pendiente de hecho" o "sin plazo legal"
// es un RESULTADO del motor, no un hueco: la norma exige la accion y el reloj
// no puede dar fecha todavia (o no la da nunca). Un conjunto exhaustivo los
// incluye, porque si no, el dorado volveria a callar sobre la mitad de lo que
// ve el operador en pantalla.
type EsperadoDorado struct {
	// Hito es la identidad de la fila: por aqui casa con el vencimiento del
	// motor. Obligatorio siempre, tambien cuando la obligacion tiene un solo
	// hito, porque emparejar "el unico que hay" es emparejar por posicion.
	Hito string `json:"hito"`

	// Vence es la fecha exacta, en RFC3339 o 2006-01-02. Obligatoria cuando el
	// estado es "determinado" (o sea, por defecto) y PROHIBIDA en los otros
	// dos: un vencimiento sin fecha no la tiene, y declararla seria afirmar
	// algo que el motor no dice.
	Vence string `json:"vence,omitempty"`

	// Estado es el vocabulario cerrado de ventana.EstadoVenc: "determinado",
	// "pendiente de hecho" o "sin plazo legal". Vacio significa "determinado",
	// y eso NO es un valor cero permisivo: determinado con fecha obligatoria es
	// la afirmacion MAS fuerte que una fila puede hacer. Declarar cualquiera de
	// los otros dos afirma otra cosa, no menos cosa.
	Estado string `json:"estado,omitempty"`
}
