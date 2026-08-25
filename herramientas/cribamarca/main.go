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

// Riesgo resume en una palabra, para la tabla. No es un dictamen.
func (h Hallazgo) Riesgo() string {
	switch {
	case len(h.Contenidas) > 0:
		return "ROJO"
	case len(h.Colisiones) > 0:
		return "ROJO"
	case len(h.Contenedoras) > 0:
		return "AMBAR"
	}
	return "sin hallazgos"
}

func main() {
	var (
		candidatos = flag.String("candidatos", "", "lista separada por comas de nombres a cribar")
		clasesTxt  = flag.String("clases", "9,42", "clases Niza que importan, separadas por comas; vacio = todas")
		oficina    = flag.String("oficina", "EM", "oficina TMview: EM = EUIPO, ES = OEPM")
		dirCache   = flag.String("cache", ".cache/cribamarca", "directorio de cache en disco")
		salidaJSON = flag.Bool("json", false, "salida en JSON en vez de tabla")
		sinCache   = flag.Bool("sin-cache", false, "ignorar la cache y volver a consultar")
	)
	flag.Parse()

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
	fmt.Println("Una criba automatica reduce la sorpresa, no sustituye a un agente de la")
	fmt.Println("propiedad industrial. Nada de esto es un dictamen juridico.")
}

func seccion(titulo string, ms []marca, nota string) {
	fmt.Printf("   %s: %d\n", titulo, len(ms))
	if len(ms) == 0 {
		return
	}
	if nota != "" {
		fmt.Println(nota)
	}
	for _, m := range ms {
		fmt.Printf("      %-12s %-22s %-12s %-11s clases %-18s %s\n",
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
