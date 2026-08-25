// Package expediente exporta y verifica el expediente completo.
//
// La propiedad que persigue: un tercero con el fichero y el binario, SIN RED
// y sin confiar en quien lo emitio, recalcula desde cero la aplicabilidad, los
// plazos y los estados de control, y obtiene exactamente lo mismo, o le dice
// donde no coincide.
//
// Ningun GRC del mercado, libre o comercial, hace esto hoy. Es lo contrario de
// la ola de "AI compliance" de 2026: en vez de pedir confianza en el emisor,
// se entrega algo que el receptor puede recomputar.
package expediente

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"dutiq/nucleo/aplicabilidad"
	"dutiq/nucleo/estado"
	"dutiq/nucleo/ledger"
	"dutiq/nucleo/ventana"
)

// Version sube a v2 con el contrato de verificacion: la cadena pasa a ledger
// v2, las anclas y las claves confiables salen del fichero, y el checkpoint
// lleva el token del sello. Un expediente v1 no se puede verificar con las
// reglas nuevas, asi que no se acepta en silencio.
const Version = "dutiq-expediente-v2"

type Paquete struct {
	URN      string    `json:"urn"`
	Version  string    `json:"version"`
	Digest   string    `json:"digest"`
	Clase    string    `json:"clase"` // normativo | referencial
	Vigencia Intervalo `json:"vigencia"`
}

type Intervalo struct {
	Desde time.Time `json:"desde"`
	Hasta time.Time `json:"hasta,omitempty"`
}

type Obligacion struct {
	ID         string `json:"id"`
	Paquete    string `json:"paquete"`
	Articulo   string `json:"articulo"`
	Afirmacion string `json:"afirmacion"`
	Control    string `json:"control"`
	Primitiva  string `json:"primitiva"`
}

// Reclamacion es un vencimiento que el emisor AFIRMA haber calculado.
// La verificacion lo recalcula y compara.
type Reclamacion struct {
	Obligacion string    `json:"obligacion"`
	Hito       string    `json:"hito"`
	Estado     string    `json:"estado"`
	Vence      time.Time `json:"vence,omitempty"`
	Regla      string    `json:"regla"`
}

type EstadoControl struct {
	Prueba string `json:"prueba"`
	Estado string `json:"estado"`
	Motivo string `json:"motivo"`
}

// SupresionDeEvidencia ata una supresion del ledger a la prueba que se quedo
// sin esa evidencia. La declara el emisor; la verificacion la contrasta contra
// la lapida firmada de la cadena, que es lo unico de aqui que el emisor no
// puede fabricarse (la firma la comprueba el receptor con SU clave de operador).
//
// POR QUE EXISTE, y la decision de diseno que hay detras (P1 10). Tras un
// borrado legal el expediente se quedaba con un EstadoControl huerfano: retirada
// la observacion suprimida, el recalculo daba otra cosa y eso salia por la
// puerta de "discrepancia". Confundir "aqui hubo un borrado con base legal" con
// "aqui hay algo que no cuadra" es confundir justo las dos cosas que este
// producto existe para distinguir. Y en el ciclo e2e la confusion era peor que
// estetica: la observacion borrada era la que FALLABA, asi que al retirarla el
// recalculo MEJORABA, de fail_en_plazo a pass. Un borrado legal que blanquea un
// incumplimiento.
//
// LA REGLA. Una prueba cuya evidencia se suprimio con base legal vale
// "obsoleto", digan lo que digan las observaciones que sobrevivieron. Tres
// razones: obsoleto ya significa exactamente eso en nucleo/estado ("no se puede
// afirmar el estado actual"); no es un fallo, asi que no acusa a nadie de
// incumplir por haber ejercido un derecho; y EscalaAlAuditor lo devuelve true,
// asi que el riesgo sigue a la vista, que es lo que no puede perderse. De
// propina, el borrado legal deja de poder mejorar la postura de nadie: de pass
// y de fail se sale igual a obsoleto, que en los denominadores cuenta como
// caducado_o_contradicho y no como maquina.
//
// LO QUE SE DESCARTO, y por que:
//
//   - Que el emisor declare el estado que tenia ANTES del borrado y marcarlo
//     como salvedad que no invalida. Indefendible: la evidencia ya no esta, o
//     sea que el receptor no puede recalcular ese estado ni ahora ni nunca, y la
//     puerta que deja pasar "seguia siendo fail_en_plazo" deja pasar igual
//     "era pass". Es pedir confianza en el emisor, que es lo unico que este
//     formato existe para no pedir.
//   - Sacar la prueba del expediente. Lo prohibe el hallazgo mismo: una
//     obligacion sin evidencia sigue siendo un riesgo aunque la evidencia se
//     borrara con base legal, y el receptor tiene que poder verlo.
//   - Un noveno estado en nucleo/estado, "suprimido_por_ley". Duplicaria a
//     Obsoleto sin cambiar NI UNA consecuencia para el auditor (los dos dicen
//     "no puedo afirmarlo" y los dos escalan), abriria un sexto cajon en unos
//     denominadores que son cinco a proposito, y lo que al receptor le falta no
//     es otro nombre de estado sino el MOTIVO con su base legal. El motivo va
//     donde se lee, en las comprobaciones del informe, y sale de la lapida.
//
// LO QUE ESTA DECLARACION NO PUEDE PROBAR, dicho aqui y no escondido: que la
// prueba nombrada sea la que de verdad se quedo sin evidencia. El contenido
// suprimido es irrecuperable por construccion (la clave se destruye) y la
// entrada de la cadena no se compromete con la prueba que ancla, asi que ningun
// verificador puede atribuirlo. Un emisor puede atribuir el borrado a un control
// que no le duele, dejar en pass el que si, y esto no lo caza: solo se caza
// cuando atribuye honestamente, que es lo que prueba
// TestHostilElBorradoLegalNoBlanqueaUnIncumplimiento.
//
// El arreglo de verdad NO esta en este paquete: seria que EntradaV2 o Lapida
// (nucleo/ledger) llevaran, fijado en el momento de escribir y firmado con el
// resto, a que prueba pertenece la entrada. Entonces destruir el contenido no
// destruiria la atribucion y esto se podria contrastar en vez de creer. Queda
// como hallazgo abierto contra nucleo/ledger.
//
// Lo que si se consigue aqui: que no haya borrados mudos. Toda lapida obliga a
// una declaracion, la declaracion obliga a un obsoleto, y las dos cosas quedan
// impresas con su base legal, o sea que la mentira hay que escribirla y firmarla
// para que un humano que sepa que se borro pueda cantarla.
type SupresionDeEvidencia struct {
	// Entrada es el indice de la entrada de la cadena que se suprimio. Tiene
	// que llevar lapida: sin acto de borrado firmado no hay supresion.
	Entrada uint64 `json:"entrada"`
	// Prueba es la que se queda sin esa evidencia. Vacia significa que esa
	// entrada no sostenia el estado de ningun control, y es una afirmacion del
	// emisor como cualquier otra: si miente, miente por escrito.
	Prueba string `json:"prueba"`
}

type Expediente struct {
	Version      string    `json:"version"`
	Emitido      time.Time `json:"emitido"`
	ComoEstaba   time.Time `json:"como_estaba"`
	Organizacion string    `json:"organizacion"`
	Alcance      string    `json:"alcance"`

	// AnclasDeclaradas es lo que el emisor DICE haber usado como anclas. No
	// decide nada: la verificacion usa las del ContextoReceptor.
	//
	// HALLAZGO DE REVISION HOSTIL, el bloqueante de la etapa 1. Este campo se
	// llamaba AnclasDeConfianza y su comentario decia que eran "los digests que
	// el RECEPTOR acepta, obtenidos del registro firmado, no del expediente".
	// Pero llevaba etiqueta json y viajaba dentro del fichero, y Verificar(e) no
	// recibia nada mas, asi que el emisor se escribia sus propias anclas y la
	// comprobacion anti-circular lo comparaba consigo mismo. Se podia inventar
	// una obligacion dentro de un paquete anclado, aportar la regla que la
	// deriva, recalcular el digest y escribirse el ancla que cuadra.
	//
	// Se conserva porque una discrepancia entre lo declarado y lo que trae el
	// receptor es informacion util para el auditor, no un fallo que esconder.
	// Contrastarlo INVALIDA el expediente (capa 2.a de Verificar), pero es una
	// capa declarativa: solo ve las URN que el emisor decidio escribir aqui. Lo
	// que para el ataque 10 es el recalculo del contenido, capa 2.b.
	AnclasDeclaradas map[string]string `json:"anclas_declaradas,omitempty"`

	Paquetes     []Paquete                `json:"paquetes"`
	Programas    []aplicabilidad.Programa `json:"programas"`
	Hechos       []aplicabilidad.Hecho    `json:"hechos"`
	Obligaciones []Obligacion             `json:"obligaciones"`

	Pruebas       []estado.Prueba      `json:"pruebas"`
	Observaciones []estado.Observacion `json:"observaciones"`
	Excepciones   []estado.Excepcion   `json:"excepciones"`
	Exclusiones   []estado.Exclusion   `json:"exclusiones"`

	Relojes []RelojDeclarado `json:"relojes"`

	// Cadena es el ledger v2: entradas cifradas con compromiso de clave,
	// lapidas firmadas y checkpoints anclados.
	//
	// HALLAZGO DE REVISION HOSTIL: aqui vivia un ledger.Ledger v1, con las
	// cargas en claro y sin compromiso, asi que todo lo que la etapa 1
	// construyo en v2 se quedaba fuera del camino que recorre un tercero.
	Cadena ledger.CadenaV2 `json:"cadena"`

	// ClavesEntradas son las claves por entrada que el emisor DIVULGA para que
	// el receptor pueda abrir la cadena y contrastarla, en hex por indice. Una
	// entrada sin clave tiene que tener lapida que explique la supresion: si no
	// la tiene, el emisor esta ocultando contenido sin decir por que, y eso es
	// una discrepancia.
	ClavesEntradas map[uint64]string `json:"claves_entradas,omitempty"`

	// SupresionesDeEvidencia dice, por cada borrado legal de la cadena, que
	// prueba se quedo sin esa evidencia. Toda lapida tiene que tener la suya:
	// un borrado mudo dejaria al receptor sin saber que control se quedo sin
	// apoyo. Ver el comentario de SupresionDeEvidencia.
	SupresionesDeEvidencia []SupresionDeEvidencia `json:"supresiones_de_evidencia,omitempty"`

	// Lo que el emisor afirma. La verificacion lo recalcula.
	Aplicables    []string             `json:"aplicables"`
	Reclamaciones []Reclamacion        `json:"reclamaciones"`
	Estados       []EstadoControl      `json:"estados"`
	Denominadores estado.Denominadores `json:"denominadores"`
}

// RelojDeclarado guarda los datos de un plazo de forma serializable.
type RelojDeclarado struct {
	Obligacion string              `json:"obligacion"`
	Disparador string              `json:"disparador"`
	Hechos     map[string]string   `json:"hechos"` // clave -> RFC3339
	Hitos      []HitoDeclarado     `json:"hitos"`
	Calendario CalendarioDeclarado `json:"calendario"`
}

type HitoDeclarado struct {
	ID        string `json:"id"`
	Limite    string `json:"limite"`
	DesdeHito string `json:"desde_hito,omitempty"`
	Computo   string `json:"computo"`  // naturales | habiles
	Cierre    string `json:"cierre"`   // auto | exacto | fin_dia
	Traslado  string `json:"traslado"` // ninguno | siguiente_habil
	Fuente    string `json:"fuente"`
}

type CalendarioDeclarado struct {
	ID       string   `json:"id"`
	Zona     string   `json:"zona"`
	Ambito   string   `json:"ambito"`
	Fuente   string   `json:"fuente"`
	Festivos []string `json:"festivos"`
}

type Discrepancia struct {
	Que      string `json:"que"`
	Esperado string `json:"esperado"`
	Obtenido string `json:"obtenido"`
}

type Informe struct {
	Valido         bool           `json:"valido"`
	Comprobaciones []string       `json:"comprobaciones"`
	Discrepancias  []Discrepancia `json:"discrepancias"`
}

// DigestPaquete calcula el digest canonico de un paquete a partir de su
// contenido: reglas y obligaciones. Es lo que se compara contra el registro.
func DigestPaquete(urn string, progs []aplicabilidad.Programa, obls []Obligacion) string {
	var partes []string
	for _, p := range progs {
		if p.Paquete != urn {
			continue
		}
		for _, r := range p.Reglas {
			b, _ := json.Marshal(r)
			partes = append(partes, string(b))
		}
	}
	for _, o := range obls {
		if o.Paquete != urn {
			continue
		}
		b, _ := json.Marshal(o)
		partes = append(partes, string(b))
	}
	sort.Strings(partes)
	h := sha256.Sum256([]byte(strings.Join(partes, "\x1f")))
	return "sha256:" + hex.EncodeToString(h[:])
}

// ContextoReceptor es TODO lo que la verificacion da por bueno, y lo aporta
// quien recibe el expediente, nunca el fichero.
//
// Existe por el bloqueante de la revision hostil de la etapa 1: la confianza
// vivia como campos con etiqueta json dentro del expediente (AnclasDeConfianza,
// ClavesConfiables), asi que el emisor se escribia sus propias anclas y sus
// propias claves y la verificacion lo comparaba consigo mismo. Un expediente
// "verificable sin confiar en el emisor" no puede sacar del emisor lo que
// decide si confia.
//
// Regla para quien lo mantenga: si algo de aqui vuelve a aparecer como campo
// serializado de Expediente o de la cadena, es el mismo bug otra vez. Hay un
// test de AST que lo vigila (TestLaConfianzaNoViajaEnElFichero).
type ContextoReceptor struct {
	// Anclas son los digests de paquete que el receptor obtuvo del registro
	// firmado. Sin ellas la verificacion del corpus seria circular.
	Anclas map[string]string
	// ClavesConfiables son las claves publicas de checkpoint que el receptor ya
	// conocia, en hex.
	ClavesConfiables []string
	// ClaveOperador verifica las lapidas de supresion.
	ClaveOperador ed25519.PublicKey
	// VerificarSello comprueba el token RFC 3161 del checkpoint contra la raiz
	// Merkle. Se inyecta porque nucleo/ no importa nada externo: la
	// implementacion vive en adaptadores/tsa.
	VerificarSello func(hash, token []byte) error
}

// confianzaLedger traduce el contexto a lo que espera el paquete ledger.
func (c ContextoReceptor) confianzaLedger() ledger.Confianza {
	return ledger.Confianza{
		ClavesConfiables: c.ClavesConfiables,
		VerificarSello:   c.VerificarSello,
		ClaveOperador:    c.ClaveOperador,
	}
}

// Verificar recomputa el expediente entero sin red y sin confiar en el emisor.
//
// ctx es lo que el receptor aporta. Si llega vacio, la verificacion no se
// calla: dice que no puede decidir nada y por que.
func Verificar(e *Expediente, ctx ContextoReceptor) Informe {
	inf := Informe{Valido: true}
	add := func(s string) { inf.Comprobaciones = append(inf.Comprobaciones, s) }
	fallo := func(que, esp, obt string) {
		inf.Valido = false
		inf.Discrepancias = append(inf.Discrepancias, Discrepancia{que, esp, obt})
	}

	if e.Version != Version {
		fallo("version", Version, e.Version)
	}

	// 1. Cadena de custodia. La confianza entra por parametro, no del fichero.
	//
	// cadenaVerificada manda sobre lo que se puede hacer con las lapidas mas
	// abajo: una lapida solo cuenta si su firma verifico contra la clave de
	// operador que aporta el RECEPTOR. Si no, cualquiera se escribiria una
	// lapida de mentira para rebajar un control a obsoleto.
	cadenaVerificada := false
	if inf2, err := e.Cadena.Verificar(ctx.confianzaLedger()); err != nil {
		fallo("cadena", "entradas encadenadas, lapidas validas y checkpoints anclados", err.Error())
	} else {
		cadenaVerificada = true
		add(fmt.Sprintf("cadena: %d entradas encadenadas, %d checkpoint(s) con sello verificado, "+
			"%d supresion(es) con base legal",
			len(e.Cadena.Entradas), len(e.Cadena.Checkpoints), len(inf2.Suprimidas)))
		for _, s := range inf2.Suprimidas {
			add("supresion: " + s)
		}
	}

	// 2. Corpus: el digest declarado tiene que salir del contenido, y el
	//    contenido tiene que coincidir con el ancla QUE TRAE EL RECEPTOR.
	if len(ctx.Anclas) == 0 {
		fallo("anclas de confianza", "el receptor aporta los digests de su registro firmado",
			"ninguna: sin ellas la verificacion del corpus seria circular")
	}
	// 2.a CAPA DECLARATIVA. Lo que el emisor dice haber usado se contrasta con
	//     el registro del receptor, no se obedece.
	//
	//     HALLAZGO (P1 12). Aqui ponia "una diferencia no invalida por si sola,
	//     pero el auditor tiene que verla". Las dos mitades eran falsas: fallo()
	//     pone Valido en false, asi que SI invalida; y lo que el auditor tenia
	//     que ver era, encima, lo unico que estaba cazando el ataque 10, porque
	//     su test solo miraba inf.Valido. Se conserva el comportamiento (declarar
	//     un corpus distinto del que trae el registro es sustantivo: dice que el
	//     emisor calculo contra otra cosa), y se corrige lo que promete.
	//
	//     Lo que esta capa NO hace: sostener el ataque 10. Solo mira las URN que
	//     el emisor se molesto en declarar; un emisor que declare exactamente lo
	//     que el receptor espera la deja muda y sigue entregando otro contenido.
	//     Esa es la capa 2.b. Cada una tiene su test y su mutacion:
	//       esta      -> TestHostilElAnclaDeclaradaQueNoCuadraSeInforma
	//       la 2.b    -> TestHostilElEmisorYaNoSeFabricaSusPropiasAnclas
	//                    TestHostilElAnclaDeclaradaImpecableNoTapaElContenidoFalseado
	//                    TestControlContenidoQueNoCuadraConElRegistro
	for urn, declarada := range e.AnclasDeclaradas {
		if real, ok := ctx.Anclas[urn]; ok && real != declarada {
			fallo("ancla declarada de "+urn, real+" (registro del receptor)",
				declarada+" (lo que declara el emisor)")
		}
	}
	// 2.b CAPA SUSTANTIVA, la que sostiene el ataque 10: el digest se RECALCULA
	//     sobre el contenido que viaja en el fichero y se contrasta con el ancla
	//     del receptor. No mira nada que el emisor haya declarado, asi que no hay
	//     declaracion que la deje muda.
	for _, p := range e.Paquetes {
		calc := DigestPaquete(p.URN, e.Programas, e.Obligaciones)
		if calc != p.Digest {
			fallo("digest de "+p.URN, p.Digest, calc)
			continue
		}
		esperado, ok := ctx.Anclas[p.URN]
		if !ok {
			fallo("ancla de "+p.URN, "digest conocido por el receptor", "paquete no reconocido")
			continue
		}
		if esperado != calc {
			fallo("contenido de "+p.URN, esperado+" (registro)", calc+" (expediente)")
		}
	}
	// Y ningun programa puede venir de un paquete que no este declarado y
	// anclado: si no, basta con adjuntar la regla que deriva lo que quieras.
	declarados := map[string]bool{}
	for _, p := range e.Paquetes {
		declarados[p.URN] = true
	}
	for _, pr := range e.Programas {
		if !declarados[pr.Paquete] {
			fallo("programa de "+pr.Paquete, "paquete declarado y anclado",
				"reglas aportadas sin paquete que las respalde")
			continue
		}
		// Defensa en profundidad, y la unica comprobacion del verificador que
		// NO tiene control negativo aislado, a proposito y no por olvido: para
		// llegar aqui el paquete tiene que estar declarado (lo garantiza el
		// continue de arriba), y todo paquete declarado pasa antes por el bucle
		// de Paquetes, donde la ausencia de ancla ya da "ancla de X". O sea que
		// es estructuralmente inalcanzable en solitario. Se queda porque si
		// alguien reordena los bucles deja de serlo, y entonces es la ultima
		// linea de defensa. Comprobado por mutacion: quitarla no pone nada rojo.
		if _, ok := ctx.Anclas[pr.Paquete]; !ok {
			fallo("programa de "+pr.Paquete, "ancla del receptor", "paquete no reconocido")
		}
	}
	add(fmt.Sprintf("corpus: %d paquetes con digest recalculado sobre su contenido y contrastado con el registro, "+
		"y %d programas atados a su paquete", len(e.Paquetes), len(e.Programas)))

	// 3. Vigencia normativa: ninguna obligacion puede evaluarse fuera de la
	//    vigencia de su paquete a la fecha del expediente.
	vig := map[string]Paquete{}
	for _, p := range e.Paquetes {
		vig[p.URN] = p
	}
	for _, o := range e.Obligaciones {
		p, ok := vig[o.Paquete]
		if !ok {
			fallo("paquete de "+o.ID, "declarado en el expediente", "ausente")
			continue
		}
		if e.ComoEstaba.Before(p.Vigencia.Desde) {
			fallo("vigencia de "+o.ID,
				"no evaluada antes de "+p.Vigencia.Desde.Format("2006-01-02"),
				"evaluada el "+e.ComoEstaba.Format("2006-01-02"))
		}
	}
	add(fmt.Sprintf("vigencia: %d obligaciones de %d paquetes comprobadas contra la fecha del expediente",
		len(e.Obligaciones), len(e.Paquetes)))

	// 3. Aplicabilidad: se reejecuta el Datalog de los paquetes.
	m := aplicabilidad.NuevoMotor()
	for _, p := range e.Programas {
		if err := m.Cargar(p); err != nil {
			fallo("programa de "+p.Paquete, "programa valido y cargable", err.Error())
		}
	}
	for _, h := range e.Hechos {
		m.Afirmar(h)
	}
	if _, err := m.Evaluar(); err != nil {
		fallo("aplicabilidad", "programa estratificable y convergente", err.Error())
	} else {
		recalc := map[string]bool{}
		for _, h := range m.Consultar(aplicabilidad.A("aplica",
			aplicabilidad.V("O"), aplicabilidad.V("S"))) {
			recalc[h.Args[0]] = true
		}
		decl := map[string]bool{}
		for _, a := range e.Aplicables {
			decl[a] = true
		}
		for a := range decl {
			if !recalc[a] {
				fallo("aplicabilidad de "+a, "derivable de las reglas y los hechos", "no derivable")
			}
		}
		for a := range recalc {
			if !decl[a] {
				fallo("aplicabilidad de "+a, "declarada en el expediente", "derivada pero no declarada")
			}
		}
		add(fmt.Sprintf("aplicabilidad: %d obligaciones aplicables rederivadas de %d reglas y %d hechos",
			len(recalc), contarReglas(e.Programas), len(e.Hechos)))
	}

	// 4. Relojes: se recalcula cada vencimiento con el motor temporal.
	recl := map[string]Reclamacion{}
	for _, r := range e.Reclamaciones {
		recl[r.Obligacion+"/"+r.Hito] = r
	}
	nrel := 0
	for _, rd := range e.Relojes {
		p, hechos, err := construirPlazo(rd)
		if err != nil {
			fallo("reloj de "+rd.Obligacion, "declaracion valida", err.Error())
			continue
		}
		for _, v := range p.Vencimientos(hechos, time.Time{}) {
			nrel++
			k := rd.Obligacion + "/" + v.Hito
			d, ok := recl[k]
			if !ok {
				fallo(k, "declarado en el expediente", "calculado pero no declarado")
				continue
			}
			if d.Estado != v.Estado.String() {
				fallo(k+" (estado)", d.Estado, v.Estado.String())
			}
			if v.Estado == ventana.Determinado && !d.Vence.Equal(v.Vence) {
				fallo(k+" (vencimiento)", d.Vence.Format(time.RFC3339), v.Vence.Format(time.RFC3339))
			}
		}
	}
	// Y al reves: una reclamacion declarada que ningun reloj calcula es una
	// fecha inventada. Antes solo se comprobaba el sentido calculado -> declarado.
	calculadas := map[string]bool{}
	for _, rd := range e.Relojes {
		p, hechos, err := construirPlazo(rd)
		if err != nil {
			continue
		}
		for _, v := range p.Vencimientos(hechos, time.Time{}) {
			calculadas[rd.Obligacion+"/"+v.Hito] = true
		}
	}
	for k := range recl {
		if !calculadas[k] {
			fallo(k, "calculable a partir de un reloj declarado", "declarado sin reloj que lo produzca")
		}
	}
	add(fmt.Sprintf("relojes: %d vencimientos recalculados con el motor temporal", nrel))

	// 5b. Toda observacion tiene que estar anclada en el ledger.
	//     Sin esto se podia cambiar `Satisfecho` en la lista de observaciones
	//     con el ledger intacto, y el expediente verificaba.
	//     Con la cadena v2 las entradas van cifradas, asi que el emisor tiene
	//     que DIVULGAR la clave de cada entrada que quiera que cuente. Aqui es
	//     donde el compromiso de clave deja de ser teoria: la clave divulgada
	//     abre exactamente un contenido y no dos, asi que no se puede ensenar
	//     una cosa al auditor y otra al juzgado con la misma cadena.
	enLedger := map[string]bool{}
	// obsEnCadena: lo que la cadena abre de verdad, para poder comprobar la
	// direccion contraria (ver mas abajo).
	obsEnCadena := map[string]observacionAnclada{}
	lapidas := map[uint64]ledger.Lapida{}
	for _, l := range e.Cadena.Lapidas {
		lapidas[l.EntradaBorrada] = l
	}
	for _, ent := range e.Cadena.Entradas {
		_, suprimida := lapidas[ent.Indice]
		hexClave, hay := e.ClavesEntradas[ent.Indice]
		if !hay {
			// Sin clave solo se puede estar si hay lapida que lo explique.
			if !suprimida {
				fallo(fmt.Sprintf("entrada %d de la cadena", ent.Indice),
					"clave divulgada, o lapida con base legal que explique la supresion",
					"ni una cosa ni la otra: hay contenido oculto sin decir por que")
			}
			continue
		}
		// Y al reves, que no se comprobaba: suprimir con base legal ES destruir
		// la clave. Un emisor que jura haber borrado una entrada y publica su
		// clave en el mismo fichero no ha borrado nada, y ademas se estaria
		// cobrando el obsoleto del apartado 5c sin pagar el borrado.
		if suprimida {
			fallo(fmt.Sprintf("supresion de la entrada %d", ent.Indice),
				"lapida con base legal y clave destruida, no divulgada",
				"la lapida dice que se suprimio y la clave viaja en el expediente: "+
					"la supresion que se declara no ha ocurrido")
		}
		clave, err := hex.DecodeString(hexClave)
		if err != nil {
			fallo(fmt.Sprintf("clave de la entrada %d", ent.Indice), "hexadecimal", err.Error())
			continue
		}
		claro, err := ledger.AbrirComprometido(clave, ent.Nonce, ent.Cifrado, ent.Compromiso)
		if err != nil {
			fallo(fmt.Sprintf("entrada %d de la cadena", ent.Indice),
				"la clave divulgada abre la entrada", err.Error())
			continue
		}
		var o estado.Observacion
		if err := json.Unmarshal(claro, &o); err == nil {
			enLedger[huellaObs(o)] = true
			obsEnCadena[huellaObs(o)] = observacionAnclada{indice: ent.Indice, obs: o}
		}
	}
	sinAnclar := 0
	for _, o := range e.Observaciones {
		if !enLedger[huellaObs(o)] {
			sinAnclar++
			fallo("observacion "+o.Prueba+"/"+o.Recurso, "anclada en el ledger",
				"no aparece en la cadena, o aparece con otro contenido")
		}
	}

	// Y LA DIRECCION CONTRARIA, que faltaba y era el agujero mas grande que ha
	// tenido este verificador.
	//
	// EL ATAQUE. Hasta aqui solo se comprobaba observaciones -> cadena: que cada
	// observacion declarada estuviera anclada. Nadie comprobaba cadena ->
	// observaciones. Asi que un emisor con un control en fail_en_plazo no tenia
	// que borrar nada: le bastaba con QUITAR esa observacion de la lista,
	// dejando su entrada y su clave publicadas e intactas en la cadena. El
	// verificador recalculaba el estado con las observaciones que quedaban,
	// salia pass, y devolvia Valido=true con cero discrepancias.
	//
	// Eso convertia en decorado toda la maquinaria de borrado legal: lapidas,
	// destruccion de clave, declaracion de supresion y forzado a obsoleto. Nada
	// de eso hacia falta para blanquear un incumplimiento.
	//
	// LA REGLA. Divulgar la clave de una entrada es decir "esto cuenta". Si
	// cuenta, tiene que estar en Observaciones. Un emisor que no quiera que
	// cuente tiene el camino escrito y caro: lapida con base legal, clave
	// destruida y supresion declarada, y entonces el control se va a obsoleto.
	// No hay tercera via, y esa es justo la propiedad.
	//
	// Que NO cubre, dicho para que nadie lo de por cubierto: esto vale mientras
	// la cadena de un expediente contenga las observaciones DE ESE expediente.
	// El dia que la cadena acumule historia (observaciones viejas superadas por
	// otras nuevas), hara falta un tercer estado declarado para "entrada de la
	// historia, no de este expediente", y ese estado tendra que ser tan caro de
	// declarar como la lapida, o el agujero vuelve por ahi.
	ocultas := 0
	declaradas := map[string]bool{}
	for _, o := range e.Observaciones {
		declaradas[huellaObs(o)] = true
	}
	for h, a := range obsEnCadena {
		if declaradas[h] {
			continue
		}
		ocultas++
		fallo(fmt.Sprintf("entrada %d de la cadena (%s/%s)", a.indice, a.obs.Prueba, a.obs.Recurso),
			"declarada en Observaciones, o suprimida con lapida y clave destruida",
			"la clave esta divulgada, o sea que la entrada cuenta, y la observacion "+
				"no aparece en el expediente: es evidencia retirada sin pagar el borrado legal")
	}

	if sinAnclar == 0 && ocultas == 0 {
		add(fmt.Sprintf("anclaje: las %d observaciones estan en la cadena con el mismo "+
			"contenido, y la cadena no abre a ninguna que el expediente no declare",
			len(e.Observaciones)))
	}

	// 5c. Supresiones de evidencia: que control se quedo sin apoyo por cada
	//     borrado legal. La decision de diseno y las alternativas descartadas
	//     estan en el comentario de SupresionDeEvidencia; aqui solo se contrasta.
	//
	//     Dos direcciones, y las dos hacen falta: toda declaracion tiene que
	//     tener lapida detras (si no, cualquiera rebaja un control a obsoleto sin
	//     borrar nada) y toda lapida tiene que tener declaracion delante (si no,
	//     el borrado es mudo y el receptor no sabe que control se quedo cojo).
	sinEvidencia := map[string][]string{} // prueba -> motivos, para el informe
	pruebaDeclarada := map[string]bool{}
	for _, pr := range e.Pruebas {
		pruebaDeclarada[pr.ID] = true
	}
	atribuidas := map[uint64]bool{}
	for _, s := range e.SupresionesDeEvidencia {
		// Las tres comprobaciones de este bucle comparten Que a proposito: hablan
		// del mismo sujeto, la declaracion de esa entrada. Lo que las separa es
		// Esperado, que es un literal fijo, y asi es como las aislan sus tests.
		que := fmt.Sprintf("supresion de evidencia declarada de la entrada %d", s.Entrada)
		l, hayLapida := lapidas[s.Entrada]
		if !hayLapida || !cadenaVerificada {
			fallo(que, "lapida con base legal, firmada por el operador, en una cadena que verifica",
				"no la hay: sin acto de borrado firmado no hay supresion que declarar")
			continue
		}
		if atribuidas[s.Entrada] {
			fallo(que, "una sola declaracion por entrada suprimida",
				"declarada mas de una vez: dos declaraciones de un borrado inflan el recuento")
			continue
		}
		atribuidas[s.Entrada] = true
		motivo := fmt.Sprintf("entrada %d, %s, el %s", s.Entrada, l.BaseLegal, l.Instante)
		if s.Prueba == "" {
			add("supresion sin efecto en controles: " + motivo +
				"; el emisor afirma que esa evidencia no sostenia el estado de ninguna prueba")
			continue
		}
		if !pruebaDeclarada[s.Prueba] {
			fallo(que, "una prueba declarada en el expediente",
				"la prueba "+s.Prueba+" no esta en el expediente: una supresion no puede "+
					"dejar sin evidencia a un control que no existe")
			continue
		}
		sinEvidencia[s.Prueba] = append(sinEvidencia[s.Prueba], motivo)
	}
	for _, l := range e.Cadena.Lapidas {
		if !atribuidas[l.EntradaBorrada] {
			fallo(fmt.Sprintf("supresion de evidencia de la entrada %d", l.EntradaBorrada),
				"declarada en supresiones_de_evidencia, con la prueba que se queda sin apoyo",
				"la cadena la borro con base legal "+l.BaseLegal+" y el expediente no dice "+
					"que control se quedo sin evidencia")
		}
	}

	// 6. Estados de control: se recalculan con la misma funcion pura.
	obsPorPrueba := map[string][]estado.Observacion{}
	for _, o := range e.Observaciones {
		obsPorPrueba[o.Prueba] = append(obsPorPrueba[o.Prueba], o)
	}
	declEst := map[string]EstadoControl{}
	for _, s := range e.Estados {
		declEst[s.Prueba] = s
	}
	var den estado.Denominadores
	for _, pr := range e.Pruebas {
		// La aplicabilidad la decide el motor, no una constante.
		aplicable := true
		if pr.Control != "" {
			aplicable = false
			for _, a := range e.Aplicables {
				if a == pr.Control {
					aplicable = true
				}
			}
		}
		ent := estado.Calcular(pr, obsPorPrueba[pr.ID], estado.Contexto{
			Ahora: e.ComoEstaba, Aplicable: aplicable,
			Excepciones: e.Excepciones, Exclusiones: e.Exclusiones,
		})
		// El borrado legal manda sobre lo que digan las observaciones que
		// sobrevivieron. De un control al que le han quitado evidencia no se
		// puede afirmar el estado actual, y en particular no se puede afirmar
		// que pasa. Se cuenta tambien asi en los denominadores: la regla que se
		// exige y el recuento independiente tienen que ser la misma regla.
		efectivo := ent.Estado
		if motivos, sinApoyo := sinEvidencia[pr.ID]; sinApoyo {
			efectivo = estado.Obsoleto
			add("estado de " + pr.ID + ": sin evidencia por supresion legal (" +
				strings.Join(motivos, "; ") + "). Vale obsoleto, que no es un fallo pero " +
				"escala al auditor, y no lo que digan las observaciones que quedan")
		}
		d, ok := declEst[pr.ID]
		if !ok {
			fallo("estado de "+pr.ID, "declarado", "ausente")
			continue
		}
		if d.Estado != efectivo.String() {
			fallo("estado de "+pr.ID, d.Estado, efectivo.String())
		}
		switch efectivo {
		case estado.Pass, estado.FailEnPlazo, estado.FailVencido:
			den.Maquina++
		case estado.Manual:
			den.Humano++
		case estado.Obsoleto:
			den.CaducadoOContradicho++
		case estado.Error, estado.NoAplica, estado.Exceptuado:
			den.Desconocido++
		}
	}
	add(fmt.Sprintf("estados: %d pruebas recalculadas", len(e.Pruebas)))
	// Los CINCO, no dos. Antes Humano, Externo y Desconocido admitian cualquier
	// valor y la verificacion seguia diciendo "coinciden".
	// NOTA (trampa conocida, encontrada en revision): ningun estado de los 8
	// mapea aun a Externo, asi que el recuento independiente siempre da 0 y un
	// expediente que declare Externo>0 NO verificara. Es deliberado hasta que
	// exista la atestacion externa (etapa 4/6): entonces su estado mapeara aqui.
	if den != e.Denominadores {
		fallo("denominadores", e.Denominadores.String(), den.String())
	} else {
		add("denominadores: los cinco coinciden con el recuento independiente")
	}

	sort.Slice(inf.Discrepancias, func(i, j int) bool { return inf.Discrepancias[i].Que < inf.Discrepancias[j].Que })
	return inf
}

// observacionAnclada es una observacion que la cadena abre de verdad, con la
// entrada de la que salio. Se guarda la entrada para poder senalarla en el
// informe: "entrada 2 de la cadena" es accionable, "una observacion" no.
type observacionAnclada struct {
	indice uint64
	obs    estado.Observacion
}

func huellaObs(o estado.Observacion) string {
	b, _ := json.Marshal(o)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func contarReglas(ps []aplicabilidad.Programa) int {
	n := 0
	for _, p := range ps {
		n += len(p.Reglas)
	}
	return n
}

func construirPlazo(rd RelojDeclarado) (ventana.Plazo, ventana.Hechos, error) {
	loc, err := time.LoadLocation(rd.Calendario.Zona)
	if err != nil {
		return ventana.Plazo{}, nil, fmt.Errorf("zona horaria %q: %w", rd.Calendario.Zona, err)
	}
	cal := ventana.NuevoCalendario(rd.Calendario.ID, rd.Calendario.Ambito, rd.Calendario.Fuente, loc, rd.Calendario.Festivos...)
	p := ventana.Plazo{Disparador: rd.Disparador}
	for _, h := range rd.Hitos {
		d, err := ventana.ParseDuracion(h.Limite)
		if err != nil {
			return ventana.Plazo{}, nil, fmt.Errorf("hito %s: %w", h.ID, err)
		}
		reg := ventana.Regimen{Cal: cal, Fuente: h.Fuente}
		if h.Computo == "habiles" {
			reg.Comp = ventana.Habiles
		}
		switch h.Cierre {
		case "exacto":
			reg.Cierre = ventana.CierreExacto
		case "fin_dia":
			reg.Cierre = ventana.CierreFinDia
		}
		if h.Traslado == "siguiente_habil" {
			reg.Trasl = ventana.TrasladoSiguienteHabil
		}
		p.Hitos = append(p.Hitos, ventana.Hito{ID: h.ID, Limite: d, Reg: reg, DesdeHito: h.DesdeHito})
	}
	hechos := ventana.Hechos{}
	for k, v := range rd.Hechos {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return ventana.Plazo{}, nil, fmt.Errorf("hecho %s: %w", k, err)
		}
		hechos[k] = t
	}
	return p, hechos, nil
}

func Cargar(b []byte) (*Expediente, error) {
	var e Expediente
	if err := json.Unmarshal(b, &e); err != nil {
		return nil, err
	}
	return &e, nil
}

func (e *Expediente) Guardar() ([]byte, error) { return json.MarshalIndent(e, "", "  ") }
