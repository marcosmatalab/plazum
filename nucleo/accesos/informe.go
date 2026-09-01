package accesos

import (
	"fmt"
	"sort"
	"strings"
)

// LaFraseDeLoNoRevisado es el patron de la casa, aplicado aqui.
//
// Un acceso sin revisar NO ES UN ACCESO INDEBIDO. Es una ausencia de dato, y
// plazum no sabe distinguirlos: puede que ese permiso sea perfectamente
// correcto y que lo unico que falte sea que alguien lo mire. Presentarlo como
// hallazgo convierte una campana a medias en una lista de acusaciones falsas, y
// quien la lea dejara de creerse el resto de la pantalla, con razon.
//
// Va PEGADA AL DATO, no en una nota al pie, y hay un test que la exige.
const LaFraseDeLoNoRevisado = "Esto NO dice que estos accesos sean indebidos: dice que en esta " +
	"campana no consta que nadie los haya revisado."

// Informe es lo que se ensena y lo que se firma.
type Informe struct {
	Campana string
	Sello   string
	Hash    string

	Accesos   int
	Decididos int
	Cubos     map[Estado]int

	LineasDeDatos       int
	FilasDuplicadas     int
	IlegiblesSinExcusar []int
	Excusas             []Excusa

	SinRevisor []Falta
	Empates    []string

	Cerrada bool
	Cierre  *Cierre
}

// Informar arma el informe de la campana tal como esta ahora.
func (c *Campana) Informar() Informe {
	cuenta := c.Cuenta()
	inf := Informe{
		Campana:             c.id,
		Sello:               c.ins.Sello(),
		Hash:                c.ins.Hash,
		Accesos:             len(c.ins.Filas),
		Decididos:           cuenta[Aprobada] + cuenta[Revocada],
		Cubos:               cuenta,
		LineasDeDatos:       c.ins.LineasDeDatos,
		FilasDuplicadas:     len(c.ins.Duplicadas),
		IlegiblesSinExcusar: c.lineasIlegiblesSinExcusar(),
		Excusas:             append([]Excusa(nil), c.excusas...),
		SinRevisor:          c.SinRevisor(),
		Cerrada:             c.Cerrada(),
		Cierre:              c.cierre,
	}
	for _, f := range c.ins.Filas {
		if _, _, err := c.Vigente(f.Clave()); err != nil {
			inf.Empates = append(inf.Empates, err.Error())
		}
	}
	return inf
}

// Cuadra dice si los cubos suman los accesos. Si esto es falso, el informe NO
// vale: no es un detalle de presentacion, es que la campana no sabe de que esta
// hablando.
func (i Informe) Cuadra() bool {
	n := 0
	for _, v := range i.Cubos {
		n += v
	}
	return n == i.Accesos
}

// Texto es el informe para una persona.
//
// Se escribe aqui y no en la superficie porque la frase de lo no revisado tiene
// que viajar CON EL DATO. Si la pusiera cada pantalla, la primera que se olvide
// acusa en falso, y ese es el unico error que un producto de cumplimiento no
// puede cometer ni una vez.
func (i Informe) Texto() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Campana de revision de accesos: %s\n", i.Campana)
	fmt.Fprintf(&b, "  sello de la instantanea: %s\n", i.Sello)
	fmt.Fprintf(&b, "  sha256 del fichero:      %s\n", i.Hash)
	fmt.Fprintf(&b, "  (con ese fichero delante, cualquiera repite la lectura y comprueba que se\n")
	fmt.Fprintf(&b, "   reviso esto y no otra cosa)\n\n")

	fmt.Fprintf(&b, "%d accesos, %d decididos.\n", i.Accesos, i.Decididos)
	for _, e := range EstadosPosibles() {
		fmt.Fprintf(&b, "  %-28s %d\n", string(e)+":", i.Cubos[e])
	}
	if !i.Cuadra() {
		fmt.Fprintf(&b, "\nAVISO: los cubos no suman %d. Este informe NO vale hasta que cuadren.\n",
			i.Accesos)
	}

	// LO NO REVISADO, CON SU FRASE AL LADO.
	if n := i.Cubos[SinRevisar] + i.Cubos[Delegada]; n > 0 {
		if i.Cubos[Delegada] > 0 {
			fmt.Fprintf(&b, "\nQuedan %d accesos sin revisar: %d sin tocar y %d delegados y aun sin\n"+
				"decidir (delegar traslada la revision, no la termina).\n",
				n, i.Cubos[SinRevisar], i.Cubos[Delegada])
		} else {
			fmt.Fprintf(&b, "\nQuedan %s sin revisar.\n", plural(n, "1 acceso", "%d accesos"))
		}
		fmt.Fprintf(&b, "%s\n", LaFraseDeLoNoRevisado)
	}

	if len(i.SinRevisor) > 0 {
		fmt.Fprintf(&b, "\n%d accesos no tienen revisor asignado. Se dice ahora y no al cerrar:\n"+
			"un aviso que llega cuando ya no se puede hacer nada no es un aviso.\n", len(i.SinRevisor))
		for _, f := range primeros(i.SinRevisor, 5) {
			fmt.Fprintf(&b, "  %s%s\n", f.Fila, rotuloEntre(f.Rotulo))
		}
		if len(i.SinRevisor) > 5 {
			fmt.Fprintf(&b, "  ...y %d mas\n", len(i.SinRevisor)-5)
		}
	}

	if len(i.IlegiblesSinExcusar) > 0 {
		fmt.Fprintf(&b, "\n%s del fichero no se pudo leer como un acceso y nadie la ha excusado: %s.\n"+
			"Bloquea el cierre. No se sabe que habia ahi, asi que no se puede certificar que se\n"+
			"reviso todo.\n",
			plural(len(i.IlegiblesSinExcusar), "1 linea", "%d lineas"),
			listaCorta(i.IlegiblesSinExcusar))
	}
	for _, e := range i.Excusas {
		fmt.Fprintf(&b, "\nLineas %d-%d excusadas por %s el %s: %s\n",
			e.Desde, e.Hasta, e.Quien, e.Cuando.Format("2006-01-02"), e.Motivo)
	}
	if i.FilasDuplicadas > 0 {
		fmt.Fprintf(&b, "\n%s del fichero repetia un acceso ya listado. No se revisan dos veces:\n"+
			"dos decisiones contrarias sobre el mismo acceso es como se consigue un expediente\n"+
			"que se contradice.\n", plural(i.FilasDuplicadas, "1 fila", "%d filas"))
	}
	for _, e := range i.Empates {
		fmt.Fprintf(&b, "\nEMPATE: %s\n", e)
	}

	if i.Cerrada && i.Cierre != nil {
		fmt.Fprintf(&b, "\nCerrada por %s el %s.\n", i.Cierre.Quien,
			i.Cierre.Cuando.Format("2006-01-02 15:04 MST"))
	} else {
		fmt.Fprintf(&b, "\nSin cerrar.\n")
	}
	return b.String()
}

// plural: estos informes los lee una persona que va a firmarlos, y "1 lineas"
// hace dudar del resto de la pantalla.
func plural(n int, uno, varios string) string {
	if n == 1 {
		return uno
	}
	return fmt.Sprintf(varios, n)
}

func primeros(fs []Falta, n int) []Falta {
	if len(fs) < n {
		return fs
	}
	return fs[:n]
}

func rotuloEntre(r string) string {
	if strings.TrimSpace(r) == "" {
		return ""
	}
	return " (" + r + ")"
}

// ---------------------------------------------------------------------------
// Cotejo entre el import manual y otra fuente (SCIM, manana un conector)
// ---------------------------------------------------------------------------

// Cuenta es lo minimo que hace falta de la otra fuente. Se pasa como dato plano
// a proposito: nucleo no conoce el adaptador de SCIM, y el dia que haya un
// conector distinto no cambia nada aqui.
type Cuenta struct {
	Sistema string
	Cuenta  string
	Rotulo  string
}

// Sospecha son dos identificadores DISTINTOS que comparten rotulo.
//
// SE ENSENAN LOS DOS Y NO SE FUNDEN, y esta es la regla entera. Cuando conviven
// el import manual y SCIM, "el mismo senor dos veces" es el problema de los
// alias con personas: puede ser la misma persona con dos cuentas (que es normal
// y a menudo correcto: una nominal y una de administracion) o dos personas que
// se llaman igual. **Plazum no sabe cual de las dos cosas es, y fundirlas por
// parecido es inventarse una identidad.** Se deduplica por identificador o se
// ensenan los dos con aviso. Nunca por nombre.
type Sospecha struct {
	Rotulo string
	Claves []string
}

// Cotejo es la comparacion entre dos fuentes.
type Cotejo struct {
	SoloEnElCenso      []string
	SoloEnLaOtraFuente []string
	EnLasDos           []string
	Sospechas          []Sospecha
}

// Cotejar compara la instantanea con otra fuente POR IDENTIFICADOR ESTABLE.
//
// Empareja por sistema|cuenta, que es lo que las dos fuentes pueden decir de la
// misma cosa. El rotulo NO empareja nunca: solo sirve para levantar una
// sospecha que decide una persona.
func (c *Campana) Cotejar(otras []Cuenta) Cotejo {
	enCenso := map[string]string{} // sistema|cuenta -> rotulo
	for _, f := range c.ins.Filas {
		enCenso[f.Sistema+"|"+f.Cuenta] = f.Rotulo
	}
	enOtra := map[string]string{}
	for _, o := range otras {
		enOtra[o.Sistema+"|"+o.Cuenta] = o.Rotulo
	}

	var cot Cotejo
	// LAS DOS DIRECCIONES, y no es simetria por gusto: la que falta es la que el
	// emisor usa. "Esta en el censo y no en SCIM" es una cuenta que el IdP ya no
	// conoce; "esta en SCIM y no en el censo" es una cuenta que la revision NO
	// ESTA MIRANDO, que es la peligrosa y la que se olvida.
	for k := range enCenso {
		if _, hay := enOtra[k]; hay {
			cot.EnLasDos = append(cot.EnLasDos, k)
		} else {
			cot.SoloEnElCenso = append(cot.SoloEnElCenso, k)
		}
	}
	for k := range enOtra {
		if _, hay := enCenso[k]; !hay {
			cot.SoloEnLaOtraFuente = append(cot.SoloEnLaOtraFuente, k)
		}
	}
	sort.Strings(cot.SoloEnElCenso)
	sort.Strings(cot.SoloEnLaOtraFuente)
	sort.Strings(cot.EnLasDos)

	// Las sospechas: mismo rotulo, identificadores distintos. Se listan, no se
	// funden.
	porRotulo := map[string]map[string]bool{}
	anotar := func(rotulo, clave string) {
		r := strings.ToLower(strings.TrimSpace(rotulo))
		if r == "" {
			return
		}
		if porRotulo[r] == nil {
			porRotulo[r] = map[string]bool{}
		}
		porRotulo[r][clave] = true
	}
	for k, r := range enCenso {
		anotar(r, k)
	}
	for k, r := range enOtra {
		anotar(r, k)
	}
	for r, claves := range porRotulo {
		if len(claves) < 2 {
			continue
		}
		s := Sospecha{Rotulo: r}
		for k := range claves {
			s.Claves = append(s.Claves, k)
		}
		sort.Strings(s.Claves)
		cot.Sospechas = append(cot.Sospechas, s)
	}
	sort.Slice(cot.Sospechas, func(i, j int) bool { return cot.Sospechas[i].Rotulo < cot.Sospechas[j].Rotulo })
	return cot
}

// Frase de una sospecha, para que la pantalla no se invente el matiz.
func (s Sospecha) Frase() string {
	return fmt.Sprintf("%s aparece con %d identificadores distintos (%s). Puede ser la misma "+
		"persona con dos cuentas o dos personas que se llaman igual: plazum no lo sabe y no las "+
		"funde. Lo decide quien conozca a la plantilla.",
		s.Rotulo, len(s.Claves), strings.Join(s.Claves, ", "))
}
