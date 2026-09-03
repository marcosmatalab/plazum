package plazum

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/marcosmatalab/plazum/superficies/camino"
	"github.com/marcosmatalab/plazum/superficies/serve"
)

// EL TTFV SOBRE EL CAMINO COMPLETO (puertas D11-a y D11-e).
//
// # Que se mide aqui y en que se distingue del TTFV que ya habia
//
// `.github/workflows/etapa2-ttfv.yml` mide EL ARRANQUE: conseguir el binario y
// llegar a la primera pantalla con valor (`plazum demo`). Es un numero util y no
// es este. Aqui se mide EL CAMINO ENTERO: los seis pasos de camino.Canonico(),
// de principio a fin, sobre el binario de verdad arrancado con `plazum serve`.
//
// # El producto se levanta DE VERDAD, y esa es la decision cara del fichero
//
// Se compila el binario, se arranca como proceso y se le hacen peticiones HTTP.
// No se construyen las superficies a mano: una reconstruccion parecida al
// cableado mide otra cosa, y lo que este numero tiene que contestar es «que le
// pasa a quien se descarga esto un martes por la manana», no «que le pasa a
// quien escribe un test».
//
// EL PRECIO ES QUE ESTE TEST TARDA, porque compila. Se asume: la alternativa es
// medir un producto que no existe.
//
// # El modelo de coste, declarado para que se pueda recontar
//
//	TTFV = T_maquina + T_humano
//
//	T_maquina = compilar el binario + arrancar hasta que /salud contesta +
//	            la suma de las latencias de los pasos. Reloj monotono.
//	T_humano  = por cada paso alcanzado, una LECTURA; por cada pregunta de la
//	            entrevista, una RESPUESTA; por cada orden de terminal que la
//	            pantalla exige, un TECLEO.
//
// LOS CONTEOS NO SE ESCRIBEN, SE SACAN DEL PRODUCTO: los pasos, de
// camino.Canonico(); las preguntas, contando las que pinta /alcance; las
// ordenes, de lo que cada pantalla PIDE, y ademas cada orden declarada se
// comprueba contra el <main> de su pantalla, asi que no se puede inflar ni
// desinflar respecto de lo que el producto dice de verdad.
//
// LOS TRES COSTES HUMANOS SI SON ESTIMACIONES, y se dicen como lo que son. No
// salen de una medida con personas: no hay ninguna todavia. Se eligen
// CONSERVADORES (por arriba) a proposito, porque el error que importa aqui es
// el que hace pasar la puerta, no el que la hace fallar. Cualquiera puede
// recontar el numero cambiando estas tres constantes, y por eso estan sueltas y
// nombradas en vez de sumadas en una sola cifra.
//
// # EL RESULTADO, Y LO QUE CAMBIO EL 03-09-2026
//
// Hasta esa fecha el camino completo NO SE PODIA RECORRER, y no por lentitud:
// TRES de los seis pasos (acta, revision de accesos y escalado) contestaban 401
// en un binario recien descargado y NO HABIA FORMA DE ENTRAR, porque
// `cmd/plazum/serve.go` construia serve.Config sin Autenticar, sin HayAdmin y
// sin CrearAdmin. Este fichero medio durante semanas los tres sextos que si se
// podian andar, y conto los otros tres con su cardinal.
//
// AHORA SE ENTRA, y este fichero mide EL CAMINO ENTERO. El cambio no es de
// medida sino de producto: se cableo `adaptadores/usuarios`, y con el la
// instalacion pasa a ser un paso REAL del recorrido, con su coste humano
// (`CosteDeInstalar`). Aqui se hace lo mismo que hace una persona: leer el
// recuadro del token que imprime el arranque, abrirlo en /primer-admin, pegarlo
// y elegir credenciales.
//
// EL NUMERO EMPEORA Y SE DICE. Un camino de seis pasos cuesta mas que un tramo
// de tres, y encima ahora hay una instalacion que antes no existia. Subir el
// techo por esto NO es la trampa que este fichero persigue (esa es subirlo
// porque la entrevista ha crecido): es que el recorrido que se mide ya no es el
// mismo, porque antes se media medio camino. El cuello de botella sigue siendo
// la entrevista de /alcance, y sigue anotado en docs/hallazgos-pantallas.md.

// LOS TRES COSTES HUMANOS, en segundos. Estimaciones declaradas, no medidas.
const (
	// CosteDeLeerUnaPantalla: abrir una pantalla que no habias visto, leer de
	// que va y encontrar por donde se sigue. Conservador por arriba.
	CosteDeLeerUnaPantalla = 45 * time.Second
	// CosteDeResponderUnaPregunta: leer una pregunta de la entrevista sobre tu
	// organizacion y contestar si o no. Es la interaccion mas repetida del
	// camino, asi que es la constante que mas manda en el total.
	CosteDeResponderUnaPregunta = 20 * time.Second
	// CosteDeTeclearUnaOrden: salir de la pantalla, ir al terminal, copiar una
	// orden con sus banderas, ejecutarla y volver. Es la mas cara de las tres,
	// y a proposito: cada una de estas es una salida del producto.
	CosteDeTeclearUnaOrden = 90 * time.Second
	// CosteDeInstalar: leer el recuadro que imprime el arranque, encontrar el
	// token entre lo demas, copiarlo al navegador, y elegir usuario y
	// contrasena. Es UNA VEZ EN LA VIDA de la instalacion, y por eso va aparte
	// de los tres de arriba, que se repiten.
	//
	// Se cuenta desde el 03-09-2026, cuando dejo de ser imposible. Antes no
	// estaba en el modelo porque no habia nada que contar: no se podia entrar.
	CosteDeInstalar = 120 * time.Second
)

// PresupuestoTTFV es el numero de la casilla D11-e de ETAPAS.md.
const PresupuestoTTFV = 15 * time.Minute

// TechoDeclaradoTTFV es LO QUE CUESTA HOY, escrito para que moleste.
//
// LA CASILLA D11-e NO SE CUMPLE, y este numero es la forma de decirlo sin
// mentir en ninguna de las dos direcciones.
//
// # El techo subio el 03-09-2026, de 20 a 25 minutos, y hay que decir por que
//
// La medida anterior era 18m56s Y ERA SOBRE MEDIO CAMINO: tres de los seis
// pasos contestaban 401 y no habia forma de entrar, asi que ni se recorrian ni
// costaban tiempo humano. Con la entrada cableada, la medida del mismo dia sobre
// el camino ENTERO da 23m11s, y la diferencia se explica entera sin ningun
// misterio: tres lecturas de pantalla que antes no se hacian (2m15s) y la
// instalacion, que antes era imposible (2m0s).
//
// ESTO NO ES LA TRAMPA QUE ESTE TECHO PERSIGUE. Esa trampa es subirlo porque la
// entrevista ha crecido, y sigue prohibida. Lo que ha pasado aqui es que el
// recorrido medido ya no es el mismo: antes se median tres sextos y ahora seis.
// Un techo que se quedara en 20 minutos obligaria a esconder la mitad del camino
// para pasar, que es exactamente lo contrario de lo que mide.
//
// El cuello de botella no ha cambiado y sigue teniendo nombre: la entrevista de
// /alcance pinta 41 preguntas, que a 20 s cada una son 13m40s, o sea casi TRES
// QUINTAS PARTES del total.
//
// NO SE HA TOCADO EL MODELO PARA QUE SALGA EL NUMERO. Bajar el coste por
// pregunta a 12 s haria pasar la puerta hoy mismo y seria la forma mas limpia
// de mentirse: lo que hay que bajar son las preguntas, no la estimacion.
//
// EL TECHO TIENE DIENTES EN LAS DOS DIRECCIONES, igual que PUERTAS_ESPERADAS:
// por arriba, porque un TTFV que crece se pone rojo antes de que nadie lo
// note; por abajo, porque el dia que el total baje del presupuesto la puerta
// TAMBIEN se pone roja y obliga a borrar este techo en el mismo commit. Un
// hueco que se cierra y deja su cardinal puesto miente hacia arriba para
// siempre.
//
// El margen sobre la medida de hoy es de 1m49s, o sea unas cinco preguntas: es
// a proposito, para que anadir entrevista sin mirar el TTFV se note.
const TechoDeclaradoTTFV = 25 * time.Minute

// AlcanceDelPaso dice si un paso del camino se puede recorrer en un binario
// recien descargado. Vocabulario cerrado.
type AlcanceDelPaso uint8

const (
	// PasoSinDeclarar es el VALOR CERO Y ES INVALIDO. Un paso nuevo del camino
	// que nadie declare llega aqui con el cero, y entonces el TTFV se mediria
	// sobre un camino distinto del que el producto tiene.
	PasoSinDeclarar AlcanceDelPaso = iota
	// PasoAlcanzable: contesta 200 SIN haber entrado.
	PasoAlcanzable
	// PasoQueExigeSesion: solo se sirve a quien ha entrado. Exige motivo.
	//
	// HASTA EL 03-09-2026 ESTO SIGNIFICABA «no se puede recorrer», porque no
	// habia forma de abrir sesion. Ya la hay, asi que ahora significa lo que
	// siempre tuvo que significar: que la pantalla lleva nombres de personas
	// dentro y no se ensena a quien no ha entrado. Los pasos asi SI entran en
	// el TTFV, porque instalar y entrar es parte del recorrido y esta medido.
	PasoQueExigeSesion
)

func (a AlcanceDelPaso) String() string {
	switch a {
	case PasoAlcanzable:
		return "se sirve sin haber entrado"
	case PasoQueExigeSesion:
		return "solo se sirve tras entrar"
	default:
		return "SIN DECLARAR (valor cero)"
	}
}

// PasosQueExigenSesion es cuantos de los seis pasos del camino solo se sirven a
// quien ha entrado.
//
// EL CARDINAL SE COMPARA CON IGUALDAD EXACTA Y EN LOS DOS SENTIDOS, y sigue
// valiendo 3 despues de cablear la entrada, porque lo que cuenta no ha cambiado:
// esas tres pantallas llevan nombres de personas y no se sirven sin sesion. Lo
// que cambio es que ahora se puede abrir una.
//
// Si SUBE, hay una pantalla mas que pide sesion y hay que decir por que. Si
// BAJA, alguien le ha quitado la proteccion a una de las tres, y eso es peor que
// un TTFV largo: son las unicas pantallas del producto cuyo contenido son
// personas con nombre.
const PasosQueExigenSesion = 3

// DeclaracionDePaso es lo que se afirma de un paso del camino a efectos de
// medir el tiempo hasta el valor.
type DeclaracionDePaso struct {
	Alcance AlcanceDelPaso
	// Motivo es obligatorio en PasoQueExigeSesion.
	Motivo string
	// Ordenes son las ordenes de terminal que ESTA pantalla exige teclear a
	// quien acaba de instalar. Cada una se comprueba contra el <main> de la
	// respuesta: una orden declarada que la pantalla no pide inflaria el coste,
	// y una que la pantalla pide y no se declara lo desinflaria.
	Ordenes []string
	// CuentaPreguntas dice si en esta pantalla se cuentan las preguntas de la
	// entrevista. Solo la de alcance.
	CuentaPreguntas bool
}

// PasosDelCamino es el censo, por el ID del paso de camino.Canonico().
//
// SE CRUZA CON EL CAMINO EN LOS DOS SENTIDOS: un paso del camino que no salga
// aqui rompe la puerta, y una entrada de aqui que el camino ya no declare,
// tambien.
var PasosDelCamino = map[string]DeclaracionDePaso{
	camino.IDDelAlcance: {
		Alcance:         PasoAlcanzable,
		CuentaPreguntas: true,
	},
	camino.IDDelCalendario: {
		Alcance: PasoAlcanzable,
		// LAS DOS ORDENES DEL ESTADO VACIO, y son el cuello de botella humano
		// del tramo que si se recorre: en una instalacion recien hecha esta
		// pantalla no ensena fechas, ensena como conseguirlas, y para eso hay
		// que salir al terminal y REARRANCAR el servidor.
		Ordenes: []string{"plazum alcance", "plazum serve --alcance"},
	},
	camino.IDDeLaDerivacion: {
		Alcance: PasoAlcanzable,
	},
	camino.IDDelActa: {
		Alcance: PasoQueExigeSesion,
		Motivo: "el acta lleva quien audito que dentro de la organizacion, asi que no se " +
			"sirve a quien no ha entrado. Desde el 03-09-2026 entrar se puede: la " +
			"instalacion crea el primer administrador con el token que imprime el " +
			"arranque, y el coste de hacerlo esta dentro de esta medida (CosteDeInstalar).",
	},
	camino.IDDeLaUAR: {
		Alcance: PasoQueExigeSesion,
		Motivo: "la revision de accesos decide sobre accesos de personas con nombre, y sin " +
			"sesion no hay autor. Se recorre tras entrar, igual que el acta.",
	},
	camino.IDDelEscalado: {
		Alcance: PasoQueExigeSesion,
		Motivo: "el plan de avisos dice a quien escribiria plazum, o sea nombres de personas. " +
			"Se recorre tras entrar, igual que el acta.",
	},
}

// MedidaDeUnPaso es lo que se saca de recorrer un paso.
type MedidaDeUnPaso struct {
	ID         string
	Ruta       string
	Codigo     int
	Latencia   time.Duration
	Preguntas  int
	Ordenes    int
	Alcanzado  bool
	CosteHuman time.Duration
}

// TestTTFVDelCaminoCompleto es la puerta D11-e, y la mitad medible de D11-a.
func TestTTFVDelCaminoCompleto(t *testing.T) {
	pasos := camino.Canonico()
	if len(pasos) < 6 {
		t.Fatalf("el camino declara %d pasos y hoy son 6: esta puerta estaria midiendo un "+
			"camino que no es el del producto", len(pasos))
	}
	cruzarElCensoDePasos(t, pasos)

	dir := t.TempDir()
	binario := filepath.Join(dir, "plazum"+extensionDeBinario())

	// 1. CONSEGUIR EL BINARIO. En una release esto es una descarga; compilar es
	//    la cota superior de las dos, asi que medir esto no favorece al numero.
	tBinario := cronometrar(func() {
		cmd := exec.Command("go", "build", "-o", binario, "./cmd/plazum")
		if salida, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("no se puede compilar el binario, asi que este TTFV no mide el "+
				"producto sino la maquina: %v\n%s", err, salida)
		}
	})

	// 2. ARRANCAR, hasta que el servidor contesta.
	srv := arrancarServidor(t, binario)
	defer srv.parar()

	// 3. INSTALAR: leer el token del terminal y crear el primer administrador.
	//    Es un paso REAL del recorrido desde que existe el almacen de usuarios,
	//    asi que cuesta maquina y cuesta humano, y las dos cosas se suman.
	tInstalacion := srv.instalar(t)

	// 4. RECORRER LOS SEIS PASOS, en su orden y con la sesion abierta.
	var tPasos, costeHumano time.Duration
	medidas := make([]MedidaDeUnPaso, 0, len(pasos))
	alcanzados, exigenSesion := 0, 0
	for _, p := range pasos {
		if !p.EsPantalla() {
			// Un paso que todavia es de terminal cuesta un tecleo y no una
			// peticion. Hoy no hay ninguno, y el cardinal de arriba lo dice.
			continue
		}
		m := recorrerUnPaso(t, srv, p)
		tPasos += m.Latencia
		costeHumano += m.CosteHuman
		medidas = append(medidas, m)
		if m.Alcanzado {
			alcanzados++
		}
		if PasosDelCamino[p.ID].Alcance == PasoQueExigeSesion {
			exigenSesion++
		}
	}

	tMaquina := tBinario + srv.tArranque + tInstalacion + tPasos
	costeHumano += CosteDeInstalar
	total := tMaquina + costeHumano

	// EL DESGLOSE SE IMPRIME SIEMPRE, en verde y en rojo: un numero sin
	// desglose no se puede recontar, y un TTFV que nadie puede recontar no vale.
	t.Logf("TTFV del camino guiado, desglose\n"+
		"  MODELO: TTFV = T_maquina + T_humano; lectura %s, respuesta %s, orden %s, "+
		"instalacion %s\n"+
		"  T_maquina %s  = binario %s + arranque %s + instalacion %s + peticiones %s\n"+
		"  T_humano  %s\n"+
		"  TOTAL     %s  (presupuesto %s)\n"+
		"  pasos alcanzados %d de %d; exigen sesion %d",
		CosteDeLeerUnaPantalla, CosteDeResponderUnaPregunta, CosteDeTeclearUnaOrden,
		CosteDeInstalar,
		tMaquina.Round(time.Millisecond), tBinario.Round(time.Millisecond),
		srv.tArranque.Round(time.Millisecond), tInstalacion.Round(time.Millisecond),
		tPasos.Round(time.Millisecond),
		costeHumano, total.Round(time.Second), PresupuestoTTFV,
		alcanzados, len(medidas), exigenSesion)
	for _, m := range medidas {
		t.Logf("  paso %-12s %-14s codigo %d  latencia %s  preguntas %2d  ordenes %d  "+
			"coste humano %s",
			m.ID, m.Ruta, m.Codigo, m.Latencia.Round(time.Millisecond), m.Preguntas,
			m.Ordenes, m.CosteHuman)
	}

	// EL CARDINAL DE LAS PANTALLAS QUE EXIGEN SESION, topado en los dos sentidos.
	if exigenSesion != PasosQueExigenSesion {
		t.Errorf("hay %d pasos del camino que solo se sirven a quien ha entrado, y "+
			"PasosQueExigenSesion dice %d.\n"+
			"  Si han SUBIDO, hay una pantalla mas cerrada y hay que decir por que.\n"+
			"  Si han BAJADO, alguien le ha quitado la sesion a una de las tres pantallas "+
			"cuyo contenido son personas con nombre.", exigenSesion, PasosQueExigenSesion)
	}
	// Y EL CAMINO ENTERO SE RECORRE. Es el criterio de exito del producto, y sin
	// esta linea el TTFV podria seguir midiendo medio camino y saliendo bonito.
	if alcanzados != len(medidas) {
		t.Errorf("se han recorrido %d de los %d pasos del camino. Un camino guiado con un "+
			"tramo que nadie puede andar no es un camino: es una lista de pantallas",
			alcanzados, len(medidas))
	}
	// EL PRESUPUESTO Y EL TECHO, y los dos tienen dientes.
	//
	// La casilla pide 15 minutos y hoy cuesta casi diecinueve, asi que lo que
	// se vigila no es que se cumpla (no se cumple, y se dice en voz alta) sino
	// que no empeore, Y que el dia que se cumpla alguien borre el techo.
	if total > TechoDeclaradoTTFV {
		t.Errorf("el TTFV del tramo recorrible del camino es %s y el techo declarado es %s.\n"+
			"  Ha EMPEORADO. El desglose de arriba dice por donde: casi siempre son "+
			"preguntas nuevas en la entrevista, que es el cuello de botella con diferencia.\n"+
			"  El arreglo NO es subir el techo ni bajar el coste por pregunta: es bajar las "+
			"preguntas o agruparlas.", total.Round(time.Second), TechoDeclaradoTTFV)
	}
	if total <= PresupuestoTTFV && TechoDeclaradoTTFV > PresupuestoTTFV {
		t.Errorf("el TTFV es %s y ya cumple el presupuesto de D11-e (%s), pero sigue habiendo "+
			"un TechoDeclaradoTTFV de %s.\n"+
			"  Esto NO es un fallo del producto: es la deuda cerrada y su cardinal sin "+
			"borrar. Un hueco que se cierra y se deja el numero puesto miente hacia arriba "+
			"para siempre.\n"+
			"  Arreglo: quitar TechoDeclaradoTTFV y dejar solo el presupuesto.",
			total.Round(time.Second), PresupuestoTTFV, TechoDeclaradoTTFV)
	}
	if total > PresupuestoTTFV {
		// SE DICE, NO SE ESCONDE. La casilla D11-e no esta cumplida, y quien
		// lea esta salida tiene que enterarse aunque el test este en verde.
		t.Logf("LA CASILLA D11-e NO SE CUMPLE: %s sobre un presupuesto de %s, ahora sobre "+
			"los %d pasos del camino entero. El cuello de botella es la entrevista de "+
			"/alcance. Ver docs/hallazgos-pantallas.md",
			total.Round(time.Second), PresupuestoTTFV, alcanzados)
	}
}

// cruzarElCensoDePasos comprueba que el censo y el camino dicen lo mismo, en
// los dos sentidos, y que ninguna entrada llega con el valor cero.
func cruzarElCensoDePasos(t *testing.T, pasos []camino.Paso) {
	t.Helper()
	enElCamino := map[string]bool{}
	for _, p := range pasos {
		enElCamino[p.ID] = true
		d, hay := PasosDelCamino[p.ID]
		if !hay {
			t.Errorf("el camino declara el paso %q y el censo del TTFV no lo tiene.\n"+
				"  Sin declararlo, el tiempo hasta el valor se estaria midiendo sobre un "+
				"camino mas corto que el que el producto ensena.", p.ID)
			continue
		}
		if d.Alcance == PasoSinDeclarar {
			t.Errorf("el paso %q esta en el censo con el VALOR CERO (%s). El cero no es un "+
				"estado, es el olvido", p.ID, d.Alcance)
		}
		if d.Alcance == PasoQueExigeSesion && strings.TrimSpace(d.Motivo) == "" {
			t.Errorf("el paso %q se declara %q y no dice por que. Un tramo del camino que no "+
				"se puede recorrer y no explica su bloqueo se lee como una decision",
				p.ID, d.Alcance)
		}
	}
	for id := range PasosDelCamino {
		if !enElCamino[id] {
			t.Errorf("el censo del TTFV declara el paso %q y camino.Canonico() ya no lo "+
				"tiene: la medida se ha quedado vieja", id)
		}
	}
}

// recorrerUnPaso pide la ruta del paso y saca de la RESPUESTA todo lo que se
// puede sacar de ella.
func recorrerUnPaso(t *testing.T, s *servidorDePruebaTTFV, p camino.Paso) MedidaDeUnPaso {
	t.Helper()
	d := PasosDelCamino[p.ID]
	m := MedidaDeUnPaso{ID: p.ID, Ruta: p.Ruta}

	inicio := time.Now()
	// CON LA SESION DEL ADMINISTRADOR, que es lo que tiene una persona que acaba
	// de instalar. Antes se pedia con http.Get a pelo, y por eso tres pasos
	// salian 401 y quedaban fuera de la medida.
	resp, err := s.cli.Get(s.base + p.Ruta)
	m.Latencia = time.Since(inicio)
	if err != nil {
		t.Fatalf("pidiendo el paso %q en %s: %v", p.ID, p.Ruta, err)
	}
	cuerpo, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	m.Codigo = resp.StatusCode
	pagina := string(cuerpo)

	// TODO PASO DEL CAMINO SE RECORRE, y lo declarado dice ademas si hace falta
	// sesion para ello.
	m.Alcanzado = m.Codigo == http.StatusOK
	if !m.Alcanzado {
		t.Errorf("el paso %q (%s) contesta %d en una instalacion recien hecha, con la sesion "+
			"del administrador abierta. El camino guiado tiene un tramo que nadie puede "+
			"andar", p.ID, p.Ruta, m.Codigo)
		return m
	}
	// EL CONTROL POSITIVO DEL DESCARGO. Un paso declarado PasoQueExigeSesion
	// tiene que seguir sin servirse SIN cookie; si no, el arreglo mas comodo
	// para que esta medida salga entera seria abrir las tres pantallas que
	// llevan nombres de personas, y nada se pondria rojo.
	if d.Alcance == PasoQueExigeSesion {
		sinSesion, err := s.crudo.Get(s.base + p.Ruta)
		if err != nil {
			t.Fatalf("pidiendo el paso %q sin sesion: %v", p.ID, err)
		}
		codigo := sinSesion.StatusCode
		_ = sinSesion.Body.Close()
		if codigo == http.StatusOK {
			t.Errorf("el paso %q (%s) se sirve con 200 SIN sesion, y esa pantalla lleva "+
				"nombres de personas dentro", p.ID, p.Ruta)
		}
	}

	principal := entreMain(t, p.ID, pagina)
	if d.CuentaPreguntas {
		// LAS PREGUNTAS SE CUENTAN DE LA PAGINA, no de una lista escrita aqui:
		// dependen del corpus instalado, y escribir el numero lo dejaria viejo
		// el dia que entre el marco trece.
		m.Preguntas = strings.Count(principal, `<li class="pregunta"`)
		if m.Preguntas == 0 {
			t.Errorf("el paso %q dice contar preguntas y la pantalla no pinta ninguna: el "+
				"coste humano del camino saldria de una entrevista vacia", p.ID)
		}
	}
	// LAS ORDENES DECLARADAS SE COMPRUEBAN CONTRA LA PANTALLA. Una orden que la
	// pantalla no pide infla el coste; una que pide y no se declara lo desinfla.
	for _, orden := range d.Ordenes {
		if !strings.Contains(principal, orden) {
			t.Errorf("el paso %q declara que hay que teclear %q y su pantalla no lo pide.\n"+
				"  O el coste humano esta inflado, o la pantalla ha dejado de decir como se "+
				"sale de su estado vacio (puerta D11-b).", p.ID, orden)
			continue
		}
		m.Ordenes++
	}
	m.CosteHuman = CosteDeLeerUnaPantalla +
		time.Duration(m.Preguntas)*CosteDeResponderUnaPregunta +
		time.Duration(m.Ordenes)*CosteDeTeclearUnaOrden
	return m
}

// entreMain recorta lo que escribe la pantalla. Se para si no casa: contar
// preguntas u ordenes en la pagina entera contaria tambien lo que pinta el
// armazon, que sale en todas.
func entreMain(t *testing.T, id, pagina string) string {
	t.Helper()
	i := strings.Index(pagina, "<main ")
	j := strings.Index(pagina, "</main>")
	if i < 0 || j < i {
		t.Fatalf("la respuesta del paso %q no trae <main>...</main>: esta medida estaria "+
			"contando el armazon", id)
	}
	return pagina[i:j]
}

// --- el servidor de verdad, como proceso ---

type servidorDePruebaTTFV struct {
	cmd       *exec.Cmd
	base      string
	tArranque time.Duration
	mu        sync.Mutex
	salida    strings.Builder
	// cli lleva la sesion del administrador; crudo no lleva ninguna. Ninguno de
	// los dos sigue redirecciones: con un cliente que las siguiera, el 303 a la
	// instalacion se leeria como el 200 de la pagina de destino y esta medida
	// contaria pantallas que nadie estaba sirviendo.
	cli   *http.Client
	crudo *http.Client
}

// reTokenDeInstalacionTTFV casa el token que imprime el arranque: 64 caracteres
// hexadecimales solos en su linea.
var reTokenDeInstalacionTTFV = regexp.MustCompile(`(?m)^\s*([0-9a-f]{64})\s*$`)

// reCampoCSRFTTFV saca el campo oculto del formulario. El nombre del campo NO se
// escribe aqui: sale de serve.CampoCSRF, que es quien lo declara.
var reCampoCSRFTTFV = regexp.MustCompile(`name="` + regexp.QuoteMeta(serve.CampoCSRF) +
	`" value="([0-9a-f]+)"`)

// instalar hace lo que hace una persona la primera vez: leer el recuadro del
// token en el terminal, abrir /primer-admin, pegarlo y elegir credenciales.
// Devuelve lo que costo de maquina; lo que cuesta de humano es CosteDeInstalar.
//
// SI ESTO FALLA, EL PRODUCTO NO SE PUEDE INSTALAR, y eso es mas grave que un
// TTFV largo: por eso se para en vez de seguir midiendo medio camino.
func (s *servidorDePruebaTTFV) instalar(t *testing.T) time.Duration {
	t.Helper()
	inicio := time.Now()

	s.mu.Lock()
	arranque := s.salida.String()
	s.mu.Unlock()
	tok := reTokenDeInstalacionTTFV.FindStringSubmatch(arranque)
	if tok == nil {
		t.Fatalf("`plazum serve` no ha impreso ningun token de primer administrador al "+
			"arrancar sobre un directorio vacio, asi que no hay forma de instalar el "+
			"producto ni de entrar en el.\n--- salida del arranque ---\n%s", arranque)
	}

	resp, err := s.cli.Get(s.base + "/primer-admin")
	if err != nil {
		t.Fatalf("abriendo /primer-admin: %v", err)
	}
	cuerpo, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /primer-admin contesta %d en una instalacion nueva. Sin esa pantalla "+
			"no hay forma de entrar en plazum.\n%s", resp.StatusCode, cuerpo)
	}
	csrf := reCampoCSRFTTFV.FindStringSubmatch(string(cuerpo))
	if csrf == nil {
		t.Fatalf("el formulario de primer administrador no trae el campo %q, asi que no se "+
			"puede enviar.\n%s", serve.CampoCSRF, cuerpo)
	}

	alta, err := s.cli.PostForm(s.base+"/primer-admin", url.Values{
		serve.CampoCSRF: {csrf[1]},
		"token":         {tok[1]},
		"usuario":       {"ciso"},
		"secreto":       {"contrasena-de-prueba-1"},
	})
	if err != nil {
		t.Fatalf("enviando el formulario de primer administrador: %v", err)
	}
	respuesta, _ := io.ReadAll(alta.Body)
	_ = alta.Body.Close()
	if alta.StatusCode != http.StatusSeeOther {
		t.Fatalf("POST /primer-admin contesta %d y tenia que crear el administrador y "+
			"redirigir.\n%s", alta.StatusCode, respuesta)
	}
	return time.Since(inicio)
}

func (s *servidorDePruebaTTFV) parar() {
	if s.cmd != nil && s.cmd.Process != nil {
		_ = s.cmd.Process.Kill()
		_ = s.cmd.Wait()
	}
}

func arrancarServidor(t *testing.T, binario string) *servidorDePruebaTTFV {
	t.Helper()
	puerto := puertoLibre(t)
	direccion := fmt.Sprintf("127.0.0.1:%d", puerto)
	// EL CORPUS ES EL DEL REPOSITORIO, que es el que se publica. Con un corpus
	// vacio la entrevista no tendria preguntas y el coste humano saldria cero,
	// que es la forma mas comoda de aprobar esta puerta sin merecerlo.
	corpus, err := filepath.Abs("paquetes")
	if err != nil {
		t.Fatalf("resolviendo el directorio del corpus: %v", err)
	}
	tarro, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("construyendo el tarro de galletas: %v", err)
	}
	sinSeguir := func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	s := &servidorDePruebaTTFV{
		base:  "http://" + direccion,
		cli:   &http.Client{Timeout: 30 * time.Second, Jar: tarro, CheckRedirect: sinSeguir},
		crudo: &http.Client{Timeout: 30 * time.Second, CheckRedirect: sinSeguir},
	}
	// #nosec G204 -- el binario lo acaba de compilar este mismo test en su
	// directorio temporal y los argumentos son constantes de aqui: no hay
	// entrada externa en esta linea.
	s.cmd = exec.Command(binario, "serve", "--direccion", direccion, "--corpus", corpus)
	s.cmd.Dir = t.TempDir()
	tuberia, err := s.cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("abriendo la salida del servidor: %v", err)
	}
	s.cmd.Stderr = os.Stderr
	inicio := time.Now()
	if err := s.cmd.Start(); err != nil {
		t.Fatalf("arrancando `plazum serve`: %v", err)
	}
	// SE LEE A TROZOS Y NO CON io.ReadAll, y esto costo un rojo: ReadAll no
	// vuelve hasta que el otro extremo cierra, o sea hasta que el servidor
	// TERMINA. Con el, la salida del arranque estaba vacia mientras el servidor
	// servia, que es justo el rato en el que hace falta: ahi es donde vive el
	// token de instalacion.
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := tuberia.Read(buf)
			if n > 0 {
				s.mu.Lock()
				s.salida.Write(buf[:n])
				s.mu.Unlock()
			}
			if err != nil {
				return
			}
		}
	}()
	// Hasta que /salud contesta. Es lo que un operador ve como «ya esta
	// arriba», y es la unica espera que se puede medir sin inventarse un reloj.
	plazo := time.Now().Add(30 * time.Second)
	for {
		resp, err := http.Get(s.base + "/salud")
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				break
			}
		}
		if time.Now().After(plazo) {
			s.parar()
			s.mu.Lock()
			texto := s.salida.String()
			s.mu.Unlock()
			t.Fatalf("`plazum serve` no contesta en %s despues de 30 s. No es que el TTFV "+
				"sea largo: es que el producto no arranca.\n--- salida ---\n%s",
				s.base, texto)
		}
		time.Sleep(20 * time.Millisecond)
	}
	s.tArranque = time.Since(inicio)
	t.Cleanup(s.parar)
	return s
}

func puertoLibre(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("no hay puerto libre para levantar el producto: %v", err)
	}
	defer func() { _ = l.Close() }()
	return l.Addr().(*net.TCPAddr).Port
}

func cronometrar(f func()) time.Duration {
	inicio := time.Now()
	f()
	return time.Since(inicio)
}

func extensionDeBinario() string {
	if os.PathSeparator == '\\' {
		return ".exe"
	}
	return ""
}
