package uar

import (
	"embed"
	"fmt"
	"io/fs"
	"strings"

	"github.com/marcosmatalab/plazum/nucleo/accesos"
	"github.com/marcosmatalab/plazum/puertos"
)

//go:embed plantillas
var plantillasFS embed.FS

// Plantillas expone el sistema de ficheros embebido, por si alguien quiere
// montar otro motor de render.
func Plantillas() fs.FS { return plantillasFS }

// Vista es lo unico que ve la plantilla.
//
// La regla del fichero es la de superficies/pantallas: los ROTULOS son claves de
// catalogo y el CONTENIDO viaja tal cual. Aqui el contenido no es corpus, es el
// censo de una organizacion (rotulos de cuenta, permisos), y por eso no pasa por
// el catalogo: traducir el nombre de un permiso de un IdP ajeno seria inventar.
type Vista struct {
	Idioma   string
	Base     string
	Estatico string
	// Titulo es la CLAVE de catalogo.
	Titulo string
	// Aviso es texto ya resuelto (viene de un error del dominio o del catalogo).
	Aviso string

	CSRF      string
	CampoCSRF string
	// PuedeMutar dice si se pintan los formularios. Falso cuando no hay token.
	PuedeMutar bool

	// SinCampana es el ESTADO VACIO, y lleva su siguiente paso dentro.
	SinCampana bool
	// SinSesion es quien llega sin haber entrado. Esta pantalla ensena nombres
	// de personas y sus permisos, asi que no se pinta el censo: es la unica
	// superficie del producto cuyo contenido es dato personal.
	SinSesion bool

	Campana    string
	Sello      string
	Hash       string
	Cerrada    bool
	CerradaPor string
	CerradaEl  string

	Accesos   int
	Decididos int
	Cubos     []CuboVista
	Filas     []FilaVista

	// SinRevisar es cuantos accesos no tienen a nadie que los mire.
	SinRevisor int
	// Ilegibles son las lineas del fichero que nadie pudo leer y nadie ha
	// excusado. Bloquean el cierre.
	Ilegibles  []int
	Excusas    []ExcusaVista
	Duplicadas int

	// Empates son decisiones contrarias en el mismo instante. Se ensenan, no se
	// resuelven.
	Empates []string

	// Cubo es el filtro activo, si lo hay.
	Cubo string
	// LoNoRevisado es cuantos accesos siguen sin decidir. Cuando es cero, la
	// frase de descargo NO se pinta: una frase que sale siempre deja de leerse.
	LoNoRevisado int
}

// CuboVista es un recuento CON SU ENLACE. La puerta D11-c es esta: ninguna cifra
// queda huerfana de enlace, porque una cifra sin derivacion obliga a fiarse.
type CuboVista struct {
	// Clave del catalogo que rotula el cubo.
	Clave string
	// Estado es el valor crudo, que va en la URL del filtro.
	Estado string
	N      int
	URL    string
	Activo bool
}

// FilaVista es un acceso, tal como se revisa.
type FilaVista struct {
	// Clave es la identidad (sistema|cuenta|permiso). Viaja en el formulario.
	Clave string
	// Cuenta, Permiso y Rotulo vienen del fichero del cliente y NO se traducen.
	Cuenta  string
	Permiso string
	Rotulo  string
	Linea   int
	// ID es la Clave saneada para caber en un atributo id de HTML sin barras
	// verticales. La identidad sigue siendo Clave, que es la que viaja en el
	// formulario: esto es solo para que `label for` case.
	ID string
	// EstadoClave es la clave de catalogo del cubo en el que esta.
	EstadoClave string
	Terminado   bool
	// Revisor es quien tiene que mirarlo. Vacio es una falta, y se ensena.
	Revisor string
	// DecididoPor y Motivo son del hecho vigente, si lo hay.
	DecididoPor string
	Motivo      string
}

// ExcusaVista es una linea dejada fuera del cierre, con quien y por que.
type ExcusaVista struct {
	Desde  int
	Hasta  int
	Quien  string
	Motivo string
}

// clavePorEstado traduce el vocabulario cerrado del nucleo a claves de catalogo.
//
// Es una tabla y no una conversion automatica a proposito: el dia que el nucleo
// gane un estado, esto no compila hasta que alguien decida como se llama en la
// pantalla y lo escriba en los dos idiomas. Una conversion automatica lo pintaria
// en espanol dentro de la interfaz inglesa sin que nada se pusiera rojo.
var clavePorEstado = map[accesos.Estado]string{
	accesos.Aprobada:   "uar.cubo.aprobada",
	accesos.Revocada:   "uar.cubo.revocada",
	accesos.Delegada:   "uar.cubo.delegada",
	accesos.SinRevisar: "uar.cubo.sin_revisar",
}

// rellenarCon vuelca la campana en la vista. No decide nada del dominio.
func (v *Vista) rellenarCon(c *accesos.Campana, idioma string, cat puertos.Catalogo) {
	inf := c.Informar()
	v.Campana = inf.Campana
	v.Sello = inf.Sello
	v.Hash = inf.Hash
	v.Accesos = inf.Accesos
	v.Decididos = inf.Decididos
	v.Duplicadas = inf.FilasDuplicadas
	v.Ilegibles = inf.IlegiblesSinExcusar
	v.Empates = inf.Empates
	v.SinRevisor = len(inf.SinRevisor)
	v.Cerrada = inf.Cerrada
	if inf.Cierre != nil {
		v.CerradaPor = inf.Cierre.Quien
		v.CerradaEl = inf.Cierre.Cuando.Format("2006-01-02 15:04 MST")
	}
	for _, e := range inf.Excusas {
		v.Excusas = append(v.Excusas, ExcusaVista{Desde: e.Desde, Hasta: e.Hasta,
			Quien: e.Quien, Motivo: e.Motivo})
	}

	for _, e := range accesos.EstadosPosibles() {
		n := inf.Cubos[e]
		// LOS CUBOS VACIOS TAMBIEN SE PINTAN. Uno que solo aparece cuando tiene
		// algo dentro es un cubo que nadie echa de menos.
		v.Cubos = append(v.Cubos, CuboVista{
			Clave: clavePorEstado[e], Estado: string(e), N: n,
			URL:    fmt.Sprintf("%s/?cubo=%s", v.Base, enlaceDe(e)),
			Activo: v.Cubo == enlaceDe(e),
		})
		if !e.Termina() {
			v.LoNoRevisado += n
		}
	}

	for _, f := range c.Instantanea().Filas {
		est := c.EstadoDe(f.Clave())
		if v.Cubo != "" && enlaceDe(est) != v.Cubo {
			continue
		}
		fv := FilaVista{
			Clave: f.Clave(), ID: idDeHTML(f.Clave()),
			Cuenta: f.Cuenta, Permiso: f.Permiso, Rotulo: f.Rotulo,
			Linea: f.Linea, EstadoClave: clavePorEstado[est], Terminado: est.Termina(),
		}
		if p, ok := c.RevisorDe(f.Clave()); ok {
			fv.Revisor = p
		}
		if d, hay, _ := c.Vigente(f.Clave()); hay {
			fv.DecididoPor = d.Quien
			fv.Motivo = d.Motivo
		}
		v.Filas = append(v.Filas, fv)
	}
	ordenarFilas(v.Filas)
}

// enlaceDe convierte el estado en algo que cabe en una URL sin escapar.
//
// El vocabulario del nucleo es texto para leer ("delegada, aun sin revisar") y
// no un identificador; meterlo en una query string tal cual haria enlaces
// distintos segun el escapado de cada navegador.
func enlaceDe(e accesos.Estado) string {
	s := strings.ToLower(string(e))
	if i := strings.Index(s, ","); i >= 0 {
		s = s[:i]
	}
	return strings.ReplaceAll(strings.TrimSpace(s), " ", "-")
}

// idDeHTML deja solo lo que cabe sin discutir en un atributo id.
func idDeHTML(clave string) string {
	var b strings.Builder
	for _, r := range clave {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}
	return b.String()
}
