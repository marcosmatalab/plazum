// Comando cribamarca: criba de anterioridades de marca contra TMview.
//
// Por que existe, y por que mira en tres direcciones.
//
// El fallo que nos costo la marca no fue no buscar. Fue buscar mal: se
// comprueba si el candidato colisiona con algo, y nadie comprueba si el
// candidato CONTIENE una marca ajena. DUTIQ contiene UTIQ entero, y UTIQ es
// una marca denominativa registrada por la empresa conjunta de cuatro
// operadoras, en las mismas clases 9 y 42 que necesitamos. Una busqueda por
// igualdad no lo habria encontrado nunca.
//
// De ahi las tres lentes, que son tres preguntas distintas:
//
//  1. COLISION: existe una marca que se llame igual que el candidato.
//     Es la que todo el mundo hace.
//  2. CONTENEDORAS: existe una marca registrada que contenga al candidato.
//     "dutiq" dentro de "midutiqpro". Menos comun, pero se mira.
//  3. CONTENIDAS: el candidato contiene una marca registrada.
//     "utiq" dentro de "dutiq". ESTA es la que nadie mira y la que nos mordio.
//
// La tercera es la cara: hay que probar cada subcadena del candidato contra la
// base, no una sola consulta. Por eso hay cache en disco y un rate limit
// conservador: la API es publica y gratuita y no se le martillea.
//
// LO QUE ESTA HERRAMIENTA NO MIRA, y costo descartar un finalista el 26-08-2026:
// TMview es un registro de MARCAS. No sabe nada de empresas en activo que
// operan sin registrar, y el uso anterior no registrado crea derechos en
// varias jurisdicciones. "Deontia" salio limpia en EUIPO y en OEPM, y existe
// Deontic (deontic.ai, Lovaina, 2022), plataforma de IA para cumplimiento
// regulatorio: mismo sector y una letra de diferencia. El registro estaba
// limpio y el mercado no.
//
// Asi que la criba tiene un paso MANUAL obligatorio despues, y la salida lo
// dice: buscar el finalista como nombre de empresa y como dominio. No se
// automatiza porque no hay fuente gratuita y fiable de razones sociales de la
// Union, y una automatizacion a medias aqui daria justo el falso verde que
// esta herramienta existe para no dar.
//
// Esto NO es un dictamen. Una criba automatica reduce la sorpresa, no sustituye
// a un agente de la propiedad industrial. Lo dice tambien la salida.
//
// Uso:
//
//	go run ./herramientas/cribamarca -candidatos dutiq,otronombre
//	go run ./herramientas/cribamarca -candidatos dutiq -clases 9,42 -json
package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	urlTMview = "https://www.tmdn.org/tmview/api/search/results"

	// TMview rechaza por cabecera, no por red. Sin un user-agent de navegador
	// la conexion ni siquiera se establece, y eso me llevo a concluir por error
	// que la base era inalcanzable. Queda escrito para que no se repita.
	agente = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 " +
		"(KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

	// Rate limit deliberadamente conservador: es una API publica y gratuita
	// que no nos debe nada. Una consulta cada segundo y medio.
	esperaEntreConsultas = 1500 * time.Millisecond

	// Subcadenas mas cortas que esto generan ruido inservible: "ut", "iq".
	minSubcadena = 3
)

// Umbrales del semaforo, y por que hubo que ponerlos.
//
// La primera version pintaba ROJO en cuanto aparecia una marca contenida, de
// la longitud que fuera. Con minSubcadena=3 y millones de marcas de la Union,
// TODO candidato lleva dentro algun acronimo de tres letras registrado en la
// clase 9: VEN, ENC, NCI, CIA, REC, ECE, CEP, EPT. El semaforo decia ROJO
// siempre, y un semaforo que siempre dice lo mismo no dice nada. Es la misma
// familia que "una puerta que nunca se ha visto fallar no es una puerta",
// vista del reves: una que salta siempre tampoco guarda.
//
// Lo que separo a DUTIQ de una casualidad no fue que UTIQ existiera, fue
// CUANTO de DUTIQ era UTIQ: cuatro letras de cinco, el 80%. Un acronimo de
// tres letras dentro de un nombre de nueve es el 33% y no se parece a nada.
// Asi que lo que pesa es la COBERTURA, no la presencia.
const (
	// Por debajo de cuatro letras es acronimo. Se cuenta y se dice, no se
	// pinta. Con -todo se listan igualmente: aqui no se tira nada en silencio.
	minRelevante = 4

	// Lente 3, la marca va DENTRO del candidato: cobertura sobre el candidato.
	// UTIQ en DUTIQ = 0,80. PRECEPT en PRECEPTUM = 0,78.
	cobRojaDentro  = 0.60
	cobAmbarDentro = 0.50

	// Lente 2, el candidato va DENTRO de la marca: cobertura sobre la marca.
	// VENCIA en AVENCIA = 0,86, que a efectos practicos es identidad.
	cobRojaFuera  = 0.70
	cobAmbarFuera = 0.50
)

// Los tres niveles del semaforo, por marca encontrada.
const (
	nivelRojo  = "rojo"
	nivelAmbar = "ambar"
	nivelRuido = "ruido"
)

// marca es lo que interesa de un registro de TMview.
type marca struct {
	Numero      string `json:"numero"`
	Nombre      string `json:"nombre"`
	Tipo        string `json:"tipo"`
	Estado      string `json:"estado"`
	Clases      []int  `json:"clases"`
	Titular     string `json:"titular"`
	Registrada  string `json:"registrada,omitempty"`
	Solicitada  string `json:"solicitada,omitempty"`
	Expira      string `json:"expira,omitempty"`
	Oficina     string `json:"oficina"`
	URLDetalle  string `json:"url_detalle,omitempty"`
	Coincidente string `json:"coincidente,omitempty"` // que subcadena la trajo
	// Cobertura: que fraccion del signo corto ocupa la coincidencia. Es lo que
	// separa "UTIQ dentro de DUTIQ" de "CIA dentro de VENCIA".
	Cobertura float64 `json:"cobertura"`
	Nivel     string  `json:"nivel"` // rojo, ambar o ruido
}

// vigente dice si la marca puede oponerse a algo. Una caducada no.
func (m marca) vigente() bool {
	switch strings.ToLower(m.Estado) {
	case "registered", "filed", "opposition pending", "under examination":
		return true
	}
	return false
}

func (m marca) tocaClases(clases []int) bool {
	if len(clases) == 0 {
		return true
	}
	for _, c := range m.Clases {
		for _, q := range clases {
			if c == q {
				return true
			}
		}
	}
	return false
}

// Hallazgo agrupa lo que se encontro para un candidato, por lente.
type Hallazgo struct {
	Candidato    string  `json:"candidato"`
	Colisiones   []marca `json:"colisiones"`
	Contenedoras []marca `json:"contenedoras"`
	Contenidas   []marca `json:"contenidas"`
	Consultas    int     `json:"consultas_hechas"`
	DesdeCache   int     `json:"consultas_desde_cache"`
}

// nivelPorCobertura traduce cobertura a semaforo. Ruido cuando la coincidencia
// es demasiado corta o demasiado parcial para que nadie confunda los signos.
func nivelPorCobertura(letras int, cob, roja, ambar float64) string {
	switch {
	case letras < minRelevante:
		return nivelRuido
	case cob >= roja:
		return nivelRojo
	case cob >= ambar:
		return nivelAmbar
	}
	return nivelRuido
}

// clasificar rellena cobertura y nivel de cada marca encontrada.
//
// Va aparte de la busqueda a proposito: el juicio es una funcion pura y se
// prueba entera sin salir a la red, que es donde estaba el fallo del semaforo.
func clasificar(h *Hallazgo) {
	cand := float64(len(h.Candidato))
	if cand == 0 {
		return
	}
	for i := range h.Colisiones {
		h.Colisiones[i].Cobertura = 1
		h.Colisiones[i].Nivel = nivelRojo
	}
	for i := range h.Contenedoras {
		m := &h.Contenedoras[i]
		largo := len(m.Nombre)
		if largo == 0 {
			largo = len(h.Candidato)
		}
		m.Cobertura = cand / float64(largo)
		m.Nivel = nivelPorCobertura(len(h.Candidato), m.Cobertura, cobRojaFuera, cobAmbarFuera)
	}
	for i := range h.Contenidas {
		m := &h.Contenidas[i]
		m.Cobertura = float64(len(m.Coincidente)) / cand
		m.Nivel = nivelPorCobertura(len(m.Coincidente), m.Cobertura, cobRojaDentro, cobAmbarDentro)
	}
}

// Riesgo resume en una palabra, para la tabla. No es un dictamen.
func (h Hallazgo) Riesgo() string {
	ambar := false
	for _, ms := range [][]marca{h.Colisiones, h.Contenedoras, h.Contenidas} {
		for _, m := range ms {
			switch m.Nivel {
			case nivelRojo:
				return "ROJO"
			case nivelAmbar:
				ambar = true
			}
		}
	}
	if ambar {
		return "AMBAR"
	}
	return "sin hallazgos"
}

// verTodo lo pone la bandera -todo: lista tambien el ruido, para poder
// auditar el umbral en vez de creerselo.
var verTodo bool

func main() {
	var (
		candidatos = flag.String("candidatos", "", "lista separada por comas de nombres a cribar")
		clasesTxt  = flag.String("clases", "9,42", "clases Niza que importan, separadas por comas; vacio = todas")
		oficina    = flag.String("oficina", "EM", "oficina TMview: EM = EUIPO, ES = OEPM")
		dirCache   = flag.String("cache", ".cache/cribamarca", "directorio de cache en disco")
		salidaJSON = flag.Bool("json", false, "salida en JSON en vez de tabla")
		sinCache   = flag.Bool("sin-cache", false, "ignorar la cache y volver a consultar")
		todo       = flag.Bool("todo", false, "listar tambien lo que queda por debajo del umbral")
	)
	flag.Parse()
	verTodo = *todo

	if strings.TrimSpace(*candidatos) == "" {
		fmt.Fprintln(os.Stderr, "falta -candidatos: los nombres que quieres cribar, separados por comas")
		fmt.Fprintln(os.Stderr, "ejemplo: go run ./herramientas/cribamarca -candidatos dutiq,otronombre")
		os.Exit(2)
	}
	clases, err := parsearClases(*clasesTxt)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(2)
	}

	c := &criba{
		oficina:  *oficina,
		clases:   clases,
		cache:    *dirCache,
		sinCache: *sinCache,
		http:     &http.Client{Timeout: 45 * time.Second},
	}

	var todos []Hallazgo
	for _, cand := range strings.Split(*candidatos, ",") {
		cand = strings.ToLower(strings.TrimSpace(cand))
		if cand == "" {
			continue
		}
		h, err := c.cribar(cand)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error cribando %q: %v\n", cand, err)
			os.Exit(1)
		}
		todos = append(todos, h)
	}

	if *salidaJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(todos); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		return
	}
	imprimirTabla(todos, clases)
}

func parsearClases(s string) ([]int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	var out []int
	for _, p := range strings.Split(s, ",") {
		n, err := strconv.Atoi(strings.TrimSpace(p))
		if err != nil {
			return nil, fmt.Errorf("clase %q no es un numero", p)
		}
		if n < 1 || n > 45 {
			return nil, fmt.Errorf("la clase %d no existe: las de Niza van de 1 a 45", n)
		}
		out = append(out, n)
	}
	return out, nil
}

type criba struct {
	oficina   string
	clases    []int
	cache     string
	sinCache  bool
	http      *http.Client
	ultima    time.Time
	consultas int
	deCache   int
}

// cribar pasa un candidato por las tres lentes.
func (c *criba) cribar(cand string) (Hallazgo, error) {
	h := Hallazgo{Candidato: cand}
	c.consultas, c.deCache = 0, 0

	// Lentes 1 y 2 con una sola consulta: se busca el candidato y se separa
	// lo que se llama igual de lo que lo contiene.
	ms, err := c.buscar(cand)
	if err != nil {
		return h, err
	}
	for _, m := range ms {
		if !m.vigente() || !m.tocaClases(c.clases) {
			continue
		}
		n := strings.ToLower(m.Nombre)
		switch {
		case n == cand:
			m.Coincidente = cand
			h.Colisiones = append(h.Colisiones, m)
		case strings.Contains(n, cand):
			m.Coincidente = cand
			h.Contenedoras = append(h.Contenedoras, m)
		}
	}

	// Lente 3, la que nadie mira: cada subcadena del candidato, de la mas
	// larga a la mas corta, buscada como marca propia. Si alguna existe y esta
	// viva en nuestras clases, el candidato la lleva dentro.
	vistas := map[string]bool{}
	for _, sub := range subcadenas(cand) {
		ms, err := c.buscar(sub)
		if err != nil {
			return h, err
		}
		for _, m := range ms {
			if !m.vigente() || !m.tocaClases(c.clases) {
				continue
			}
			if strings.ToLower(m.Nombre) != sub {
				continue // solo coincidencia exacta con la subcadena
			}
			if vistas[m.Numero] {
				continue
			}
			vistas[m.Numero] = true
			m.Coincidente = sub
			h.Contenidas = append(h.Contenidas, m)
		}
	}
	// De la subcadena mas larga a la mas corta: una marca mas larga dentro del
	// candidato preocupa mas que una de tres letras.
	sort.SliceStable(h.Contenidas, func(i, j int) bool {
		return len(h.Contenidas[i].Coincidente) > len(h.Contenidas[j].Coincidente)
	})

	h.Consultas, h.DesdeCache = c.consultas, c.deCache
	clasificar(&h)
	return h, nil
}

// subcadenas devuelve las subcadenas propias del candidato, de larga a corta.
// Propias: se excluye el candidato entero, que ya cubre la lente 1.
func subcadenas(s string) []string {
	var out []string
	for l := len(s) - 1; l >= minSubcadena; l-- {
		for i := 0; i+l <= len(s); i++ {
			out = append(out, s[i:i+l])
		}
	}
	return out
}

// buscar consulta TMview, con cache en disco y rate limit.
func (c *criba) buscar(termino string) ([]marca, error) {
	if !c.sinCache {
		if ms, ok := c.leerCache(termino); ok {
			c.deCache++
			return ms, nil
		}
	}
	// Rate limit: se espera lo que falte desde la ultima consulta real.
	if !c.ultima.IsZero() {
		if falta := esperaEntreConsultas - time.Since(c.ultima); falta > 0 {
			time.Sleep(falta)
		}
	}
	c.ultima = time.Now()
	c.consultas++

	cuerpo, err := json.Marshal(map[string]any{
		"page":        "1",
		"pageSize":    "50",
		"criteria":    "E", // E = empieza por / exacta segun termino
		"basicSearch": termino,
		"fOffices":    []string{c.oficina},
	})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, urlTMview, bytes.NewReader(cuerpo))
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", agente)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("no responde TMview: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("TMview devolvio HTTP %d para %q; "+
			"si es 403 o 405, revisa el user-agent y el metodo", resp.StatusCode, termino)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	ms, err := parsear(b)
	if err != nil {
		return nil, fmt.Errorf("respuesta de TMview ilegible para %q: %w", termino, err)
	}
	c.escribirCache(termino, ms)
	return ms, nil
}

// respuestaTMview es solo lo que se usa. TMview devuelve mucho mas.
type respuestaTMview struct {
	TradeMarks []struct {
		ApplicationNumber string   `json:"applicationNumber"`
		TmName            string   `json:"tmName"`
		TmOffice          string   `json:"tmOffice"`
		TmOfficeURL       string   `json:"tmOfficeURL"`
		TradeMarkStatus   string   `json:"tradeMarkStatus"`
		TradeMarkType     string   `json:"tradeMarkType"`
		NiceClass         []int    `json:"niceClass"`
		ApplicantName     []string `json:"applicantName"`
		ApplicationDate   string   `json:"applicationDate"`
		RegistrationDate  string   `json:"registrationDate"`
		ExpirationDate    string   `json:"expirationDate"`
	} `json:"tradeMarks"`
}

func parsear(b []byte) ([]marca, error) {
	var r respuestaTMview
	if err := json.Unmarshal(b, &r); err != nil {
		return nil, err
	}
	out := make([]marca, 0, len(r.TradeMarks))
	for _, t := range r.TradeMarks {
		out = append(out, marca{
			Numero: t.ApplicationNumber, Nombre: t.TmName, Tipo: t.TradeMarkType,
			Estado: t.TradeMarkStatus, Clases: t.NiceClass,
			Titular:    strings.Join(t.ApplicantName, "; "),
			Registrada: soloFecha(t.RegistrationDate), Solicitada: soloFecha(t.ApplicationDate),
			Expira:  soloFecha(t.ExpirationDate),
			Oficina: t.TmOffice, URLDetalle: t.TmOfficeURL,
		})
	}
	return out, nil
}

func soloFecha(s string) string {
	if len(s) >= 10 {
		return s[:10]
	}
	return s
}

// --- Cache en disco ---
//
// Por termino, no por candidato: las subcadenas se repiten mucho entre
// candidatos parecidos y asi una segunda ejecucion no vuelve a salir a la red.

func (c *criba) rutaCache(termino string) string {
	// hex del termino: evita cualquier sorpresa con nombres de fichero.
	return filepath.Join(c.cache, c.oficina+"-"+hex.EncodeToString([]byte(termino))+".json")
}

func (c *criba) leerCache(termino string) ([]marca, bool) {
	b, err := os.ReadFile(c.rutaCache(termino)) // #nosec G304 -- ruta derivada del termino, dentro del dir de cache
	if err != nil {
		return nil, false
	}
	var ms []marca
	if err := json.Unmarshal(b, &ms); err != nil {
		return nil, false
	}
	return ms, true
}

func (c *criba) escribirCache(termino string, ms []marca) {
	if err := os.MkdirAll(c.cache, 0o750); err != nil {
		return // la cache es una optimizacion; si falla, se sigue sin ella
	}
	b, err := json.Marshal(ms)
	if err != nil {
		return
	}
	_ = os.WriteFile(c.rutaCache(termino), b, 0o600)
}

// --- Salida ---

func imprimirTabla(hs []Hallazgo, clases []int) {
	fmt.Printf("criba de marca, clases %s, oficina EUIPO\n\n", listaClases(clases))
	for _, h := range hs {
		fmt.Printf("== %s  [%s]  (%d consultas, %d de cache)\n",
			strings.ToUpper(h.Candidato), h.Riesgo(), h.Consultas, h.DesdeCache)

		seccion("COLISIONES (se llaman igual)", h.Colisiones, "")
		seccion("CONTENEDORAS (una marca contiene el candidato)", h.Contenedoras, "")
		seccion("CONTENIDAS (el candidato contiene una marca)", h.Contenidas,
			"        ^ esta es la lente que casi nadie mira, y la que encontro UTIQ dentro de DUTIQ")
		fmt.Println()
	}
	fmt.Println("FALTA UN PASO, Y ES MANUAL: esto es un registro de MARCAS, no sabe de")
	fmt.Println("empresas en activo sin registrar. Busca cada finalista como nombre de")
	fmt.Println("empresa y como dominio antes de decidir. Un finalista limpio aqui se cayo")
	fmt.Println("asi el 26-08-2026: competidor directo con una letra de diferencia.")
	fmt.Println()
	fmt.Println("Una criba automatica reduce la sorpresa, no sustituye a un agente de la")
	fmt.Println("propiedad industrial. Nada de esto es un dictamen juridico.")
}

func seccion(titulo string, ms []marca, nota string) {
	pintar := make([]marca, 0, len(ms))
	ruido := 0
	for _, m := range ms {
		if m.Nivel == nivelRuido && !verTodo {
			ruido++
			continue
		}
		pintar = append(pintar, m)
	}
	ms = pintar

	fmt.Printf("   %s: %d\n", titulo, len(ms))
	if ruido > 0 {
		// El ruido se cuenta y se dice. Un umbral que descarta en silencio hace
		// que "sin hallazgos" se lea como "se ha mirado todo".
		fmt.Printf("      (+%d por debajo del umbral: acronimos de menos de %d letras o "+
			"coincidencias parciales. Con -todo se listan)\n", ruido, minRelevante)
	}
	if len(ms) == 0 {
		return
	}
	if nota != "" {
		fmt.Println(nota)
	}
	for _, m := range ms {
		fmt.Printf("      [%-5s %3.0f%%] %-12s %-22s %-12s %-11s clases %-18s %s\n",
			m.Nivel, m.Cobertura*100,
			m.Numero, recortar(m.Nombre, 22), m.Tipo, m.Estado, listaClases(m.Clases), recortar(m.Titular, 30))
		if m.Coincidente != "" && !strings.EqualFold(m.Coincidente, m.Nombre) {
			fmt.Printf("                   (encontrada buscando %q)\n", m.Coincidente)
		}
	}
}

func listaClases(cs []int) string {
	if len(cs) == 0 {
		return "todas"
	}
	p := make([]string, 0, len(cs))
	for _, c := range cs {
		p = append(p, strconv.Itoa(c))
	}
	return strings.Join(p, ",")
}

func recortar(s string, n int) string {
	if len(s) <= n {
		return s
	}
	if n <= 1 {
		return s[:n]
	}
	return s[:n-1] + "."
}
