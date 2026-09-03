package plazum

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/marcosmatalab/plazum/superficies/camino"
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
// # EL RESULTADO, DICHO ANTES DE MEDIR NADA
//
// El camino completo NO SE PUEDE RECORRER hoy, y no por lentitud: TRES de los
// seis pasos (acta, revision de accesos y escalado) contestan 401 en un binario
// recien descargado, y NO HAY FORMA DE ENTRAR. `cmd/plazum/serve.go` construye
// serve.Config sin HayAdmin ni CrearAdmin, asi que /primer-admin contesta 503
// con un mensaje dirigido a «quien lo cablea» y /entrar pinta un formulario que
// pide credenciales que no puede tener nadie. El propio `plazum serve` lo dice
// al arrancar: «sirve para montar pantallas y para nada mas».
//
// Asi que este fichero mide lo que SI se puede recorrer y CUENTA lo que no, con
// su cardinal topado en los dos sentidos. No se ajusta el modelo para que salga
// un numero bonito: el numero que importa es que tres sextos del camino guiado
// no existen para quien se descarga el producto. Va entero en
// docs/hallazgos-pantallas.md, con su prioridad.

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
)

// PresupuestoTTFV es el numero de la casilla D11-e de ETAPAS.md.
const PresupuestoTTFV = 15 * time.Minute

// TechoDeclaradoTTFV es LO QUE CUESTA HOY, escrito para que moleste.
//
// LA CASILLA D11-e NO SE CUMPLE, y este numero es la forma de decirlo sin
// mentir en ninguna de las dos direcciones. La medida del 03-09-2026, sobre los
// TRES pasos que se pueden recorrer (los otros tres contestan 401), da 18m56s,
// y el cuello de botella tiene nombre: la entrevista de /alcance pinta 41
// preguntas, que a 20 s cada una son 13m40s, o sea TRES CUARTAS PARTES del
// total. El resto son tres lecturas de pantalla y las dos ordenes de terminal
// que exige el estado vacio del calendario.
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
// El margen sobre la medida de hoy es de poco mas de un minuto, o sea unas tres
// preguntas: es a proposito, para que anadir entrevista sin mirar el TTFV se
// note.
const TechoDeclaradoTTFV = 20 * time.Minute

// AlcanceDelPaso dice si un paso del camino se puede recorrer en un binario
// recien descargado. Vocabulario cerrado.
type AlcanceDelPaso uint8

const (
	// PasoSinDeclarar es el VALOR CERO Y ES INVALIDO. Un paso nuevo del camino
	// que nadie declare llega aqui con el cero, y entonces el TTFV se mediria
	// sobre un camino distinto del que el producto tiene.
	PasoSinDeclarar AlcanceDelPaso = iota
	// PasoAlcanzable: contesta 200 sin haber entrado.
	PasoAlcanzable
	// PasoQueExigeSesion: no se puede recorrer sin haber entrado, y hoy no hay
	// forma de entrar. Exige motivo.
	PasoQueExigeSesion
)

func (a AlcanceDelPaso) String() string {
	switch a {
	case PasoAlcanzable:
		return "alcanzable en un binario recien descargado"
	case PasoQueExigeSesion:
		return "exige sesion, y hoy no hay forma de abrirla"
	default:
		return "SIN DECLARAR (valor cero)"
	}
}

// PasosQueExigenSesion es cuantos de los seis pasos del camino NO se pueden
// recorrer hoy.
//
// EL CARDINAL SE COMPARA CON IGUALDAD EXACTA Y EN LOS DOS SENTIDOS. Si sube,
// el camino guiado se ha roto mas; si baja, alguien ha cableado la entrada y
// tiene que bajarlo aqui. Un hueco sin numero se olvida.
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
			"sirve a quien no ha entrado, y eso es correcto. Lo que no lo es: en el binario " +
			"que se descarga un evaluador NO HAY FORMA DE ENTRAR. cmd/plazum/serve.go " +
			"construye serve.Config sin HayAdmin ni CrearAdmin, asi que /primer-admin " +
			"contesta 503 y /entrar pide credenciales que no existen.",
	},
	camino.IDDeLaUAR: {
		Alcance: PasoQueExigeSesion,
		Motivo: "la revision de accesos decide sobre accesos de personas con nombre, y sin " +
			"sesion no hay autor. Mismo bloqueo que el acta: no hay forma de abrir sesion.",
	},
	camino.IDDelEscalado: {
		Alcance: PasoQueExigeSesion,
		Motivo: "el plan de avisos dice a quien escribiria plazum, o sea nombres de personas. " +
			"Mismo bloqueo que el acta: no hay forma de abrir sesion.",
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

	// 3. RECORRER LOS SEIS PASOS, en su orden.
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
		} else {
			exigenSesion++
		}
	}

	tMaquina := tBinario + srv.tArranque + tPasos
	total := tMaquina + costeHumano

	// EL DESGLOSE SE IMPRIME SIEMPRE, en verde y en rojo: un numero sin
	// desglose no se puede recontar, y un TTFV que nadie puede recontar no vale.
	t.Logf("TTFV del camino guiado, desglose\n"+
		"  MODELO: TTFV = T_maquina + T_humano; lectura %s, respuesta %s, orden %s\n"+
		"  T_maquina %s  = binario %s + arranque %s + peticiones %s\n"+
		"  T_humano  %s\n"+
		"  TOTAL     %s  (presupuesto %s)\n"+
		"  pasos alcanzados %d de %d; exigen sesion %d",
		CosteDeLeerUnaPantalla, CosteDeResponderUnaPregunta, CosteDeTeclearUnaOrden,
		tMaquina.Round(time.Millisecond), tBinario.Round(time.Millisecond),
		srv.tArranque.Round(time.Millisecond), tPasos.Round(time.Millisecond),
		costeHumano, total.Round(time.Second), PresupuestoTTFV,
		alcanzados, len(medidas), exigenSesion)
	for _, m := range medidas {
		t.Logf("  paso %-12s %-14s codigo %d  latencia %s  preguntas %2d  ordenes %d  "+
			"coste humano %s",
			m.ID, m.Ruta, m.Codigo, m.Latencia.Round(time.Millisecond), m.Preguntas,
			m.Ordenes, m.CosteHuman)
	}

	// EL CARDINAL DE LO QUE NO SE PUEDE RECORRER, topado en los dos sentidos.
	if exigenSesion != PasosQueExigenSesion {
		t.Errorf("hay %d pasos del camino que no se pueden recorrer en un binario recien "+
			"descargado, y PasosQueExigenSesion dice %d.\n"+
			"  Si han subido, el camino guiado se ha roto mas.\n"+
			"  Si han bajado, alguien ha cableado la entrada y tiene que bajar el numero "+
			"aqui: el cardinal es lo unico que hace que este hueco moleste hasta que se "+
			"cierre.", exigenSesion, PasosQueExigenSesion)
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
		t.Logf("LA CASILLA D11-e NO SE CUMPLE: %s sobre un presupuesto de %s, y ademas sobre "+
			"%d de los %d pasos, porque los otros %d contestan 401 y no hay forma de entrar. "+
			"El cuello de botella es la entrevista de /alcance. Ver docs/hallazgos-pantallas.md",
			total.Round(time.Second), PresupuestoTTFV, alcanzados, len(medidas), exigenSesion)
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
	resp, err := http.Get(s.base + p.Ruta)
	m.Latencia = time.Since(inicio)
	if err != nil {
		t.Fatalf("pidiendo el paso %q en %s: %v", p.ID, p.Ruta, err)
	}
	cuerpo, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	m.Codigo = resp.StatusCode
	pagina := string(cuerpo)

	// LO QUE SE DECLARA Y LO QUE CONTESTA TIENEN QUE COINCIDIR. Sin esto, el
	// censo podria decir «alcanzable» de un paso que contesta 401 y el TTFV
	// saldria de un camino imaginario.
	m.Alcanzado = m.Codigo == http.StatusOK
	switch d.Alcance {
	case PasoAlcanzable:
		if !m.Alcanzado {
			t.Errorf("el paso %q se declara alcanzable y %s contesta %d en un binario recien "+
				"arrancado", p.ID, p.Ruta, m.Codigo)
		}
	case PasoQueExigeSesion:
		if m.Alcanzado {
			t.Errorf("el paso %q se declara bloqueado por la sesion y %s contesta 200. O se "+
				"ha cableado la entrada, o la pantalla ha dejado de proteger lo que protegia",
				p.ID, p.Ruta)
		}
	}
	if !m.Alcanzado {
		// Un paso al que no se llega no cuesta tiempo humano: cuesta el camino
		// entero, y eso lo dice el cardinal, no el reloj.
		return m
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
	s := &servidorDePruebaTTFV{base: "http://" + direccion}
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
	go func() {
		b, _ := io.ReadAll(tuberia)
		s.mu.Lock()
		s.salida.Write(b)
		s.mu.Unlock()
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
