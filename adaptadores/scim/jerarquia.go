package scim

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// MaxProfundidadJerarquia acota la cadena de mando que se acepta ESCRIBIR.
//
// Una empresa de 200 personas tiene cuatro o cinco niveles; una multinacional,
// doce. Sesenta y cuatro es absurdo por arriba a proposito: no esta para
// modelar organigramas, esta para que un `manager` mal mapeado en el IdP se
// pare al escribirse en vez de convertirse en un recorrido inutil.
//
// Deliberadamente NO es un criterio de carga. La deteccion de ciclos al cargar
// se hace con el conjunto de visitados, que es completa: en un grafo finito sin
// ciclos el recorrido termina siempre. Si la carga rechazara ademas por
// profundidad, habria datos escribibles que no cargan, y esa discrepancia ya
// nos costo un hallazgo (ver validarManagerBloqueado).
const MaxProfundidadJerarquia = 64

// Relacion es una arista del organigrama: quien reporta a quien, de donde salio
// y desde cuando.
type Relacion struct {
	Empleado string
	Manager  string
	Origen   Origen
	Desde    time.Time
	// Autor solo lo llevan las manuales: quien la declaro.
	Autor string
}

// Jerarquia es el organigrama ya resuelto y validado.
//
// Se construye entera y se valida entera, como el arbol de perimetros del
// nucleo, y por el mismo motivo: un ciclo no es un caso borde raro de
// produccion, es un error de carga, y descubrirlo al recorrer significa
// descubrirlo mientras corre un plazo.
type Jerarquia struct {
	rel      map[string]Relacion
	directos map[string][]string
	vivos    map[string]Usuario
	// rotas son las relaciones que apuntan a alguien que ya no existe o esta
	// borrado. No invalidan el arbol: se excluyen y se ensenan.
	rotas []Relacion
	// conflictos son los empleados que tienen manager del IdP Y manual, y no
	// coinciden. No se resuelven en silencio: se ensenan.
	conflictos []Conflicto
}

// Conflicto es un empleado con dos jefes declarados por dos vias distintas.
type Conflicto struct {
	Empleado    string
	SegunIdP    string
	SegunManual string
}

// Jerarquia construye y valida el organigrama a partir de lo que hay guardado.
//
// La resolucion, escrita para que se pueda discutir:
//
//  1. Si el IdP publica `manager` para alguien, MANDA el IdP. Es la fuente
//     autoritativa mientras hable.
//  2. Si no lo publica, vale el mapeo manual del operador.
//  3. Si hay los dos y difieren, manda el IdP y el caso sale en Conflictos. No
//     se decide en silencio a favor de nadie: el operador tiene que poder ver
//     que su mapeo manual quedo desplazado, o se pasara meses creyendo que el
//     escalado sube por donde no sube.
func (d *Directorio) Jerarquia() (*Jerarquia, error) {
	d.mu.RLock()
	rel := map[string]Relacion{}
	vivos := map[string]Usuario{}
	var rotas []Relacion
	var conflictos []Conflicto

	for id, u := range d.usuarios {
		if u.Vivo() {
			vivos[id] = *u
		}
	}
	for id, u := range d.usuarios {
		if !u.Vivo() {
			continue
		}
		manual, hayManual := d.manualPorEmpleado[id]
		switch {
		case u.ManagerIdP != "":
			r := Relacion{Empleado: id, Manager: u.ManagerIdP, Origen: OrigenIdP, Desde: u.Modificado}
			if _, ok := vivos[u.ManagerIdP]; !ok {
				rotas = append(rotas, r)
			} else {
				rel[id] = r
			}
			if hayManual && manual.manager != u.ManagerIdP {
				conflictos = append(conflictos, Conflicto{
					Empleado: id, SegunIdP: u.ManagerIdP, SegunManual: manual.manager,
				})
			}
		case hayManual:
			r := Relacion{
				Empleado: id, Manager: manual.manager, Origen: OrigenManual,
				Desde: manual.desde, Autor: manual.autor,
			}
			if _, ok := vivos[manual.manager]; !ok {
				rotas = append(rotas, r)
			} else {
				rel[id] = r
			}
		}
	}
	d.mu.RUnlock()

	j := &Jerarquia{
		rel:      rel,
		directos: map[string][]string{},
		vivos:    vivos,
		rotas:    rotas,
	}
	sort.Slice(j.rotas, func(a, b int) bool { return j.rotas[a].Empleado < j.rotas[b].Empleado })
	sort.Slice(conflictos, func(a, b int) bool { return conflictos[a].Empleado < conflictos[b].Empleado })
	j.conflictos = conflictos

	// Los ciclos se rechazan AL CARGAR, igual que en el arbol de perimetros
	// del nucleo. Un ciclo en la jerarquia cuelga el escalado: la obligacion
	// vencida sube a su jefe, que sube al primero, y nadie avisa nunca.
	for id := range rel {
		visto := map[string]bool{id: true}
		cur := rel[id].Manager
		for n := 0; cur != ""; n++ {
			if visto[cur] {
				return nil, fmt.Errorf("ciclo en la jerarquia de managers: %s. "+
					"Un ciclo cuelga el escalado: la obligacion vencida sube a su jefe, que "+
					"sube al primero, y no avisa nunca. Arreglo: corrige el atributo `manager` "+
					"en el IdP para uno de los implicados, o declara su jefe a mano",
					describirCiclo(rel, cur))
			}
			// Cinturon estructural: en un grafo de N aristas sin ciclos no
			// se puede subir mas de N veces, asi que esto no se alcanza. Se
			// deja escrito para que un cambio futuro en el conjunto de
			// visitados no pueda convertir este bucle en uno infinito, que
			// aqui significa que una obligacion vencida no avisa a nadie.
			if n > len(rel) {
				return nil, fmt.Errorf("el recorrido de la jerarquia desde %s no termina "+
					"pese a no encontrar ciclo: la estructura esta corrupta", id)
			}
			visto[cur] = true
			siguiente, ok := rel[cur]
			if !ok {
				break
			}
			cur = siguiente.Manager
		}
	}
	for id, r := range rel {
		j.directos[r.Manager] = append(j.directos[r.Manager], id)
	}
	for k := range j.directos {
		sort.Strings(j.directos[k])
	}
	return j, nil
}

// describirCiclo escribe el ciclo entero para que el mensaje sea accionable.
// Un error que dice "hay un ciclo" y no dice cual obliga a buscarlo a mano.
func describirCiclo(rel map[string]Relacion, desde string) string {
	var camino []string
	visto := map[string]bool{}
	cur := desde
	for cur != "" && !visto[cur] {
		visto[cur] = true
		camino = append(camino, cur)
		r, ok := rel[cur]
		if !ok {
			break
		}
		cur = r.Manager
	}
	camino = append(camino, desde)
	return strings.Join(camino, " -> ")
}

// Manager devuelve la relacion de alguien, con su origen.
func (j *Jerarquia) Manager(id string) (Relacion, bool) {
	r, ok := j.rel[id]
	return r, ok
}

// Directos devuelve quienes reportan a alguien.
func (j *Jerarquia) Directos(id string) []string { return j.directos[id] }

// Cadena devuelve la cadena de escalado hacia arriba, sin incluir al propio
// empleado: primero su jefe, luego el jefe de su jefe, hasta la raiz.
//
// Es lo que consume el escalado de la etapa 4.
func (j *Jerarquia) Cadena(id string) []string {
	var out []string
	visto := map[string]bool{id: true}
	cur := id
	// El tope es estructural, no una regla de negocio: con N relaciones no se
	// puede subir mas de N veces sin repetir, y repetir ya lo corta el
	// conjunto de visitados. Asi una cadena legitima larga no se trunca en
	// silencio, que seria escalar a menos gente de la que toca.
	for n := 0; n <= len(j.rel); n++ {
		r, ok := j.rel[cur]
		if !ok || visto[r.Manager] {
			break
		}
		visto[r.Manager] = true
		out = append(out, r.Manager)
		cur = r.Manager
	}
	return out
}

// SinManager lista los usuarios vivos y activos a los que no se les conoce
// jefe, ordenados.
//
// Es la respuesta a "que ve el comprador cuando el IdP no publica manager": una
// lista concreta de personas cuyo escalado no tiene a donde subir, no un
// silencio.
func (j *Jerarquia) SinManager() []string {
	var out []string
	for id, u := range j.vivos {
		if !u.Activo {
			continue
		}
		if _, ok := j.rel[id]; !ok {
			out = append(out, id)
		}
	}
	sort.Strings(out)
	return out
}

// Rotas son las relaciones que apuntan a alguien que ya no existe.
func (j *Jerarquia) Rotas() []Relacion { return j.rotas }

// Conflictos son los empleados con jefe declarado por las dos vias y distinto.
func (j *Jerarquia) Conflictos() []Conflicto { return j.conflictos }

// Relaciones devuelve todo el organigrama resuelto, ordenado por empleado.
// Cada relacion dice de donde vino, que es el requisito del mapeo manual.
func (j *Jerarquia) Relaciones() []Relacion {
	out := make([]Relacion, 0, len(j.rel))
	for _, r := range j.rel {
		out = append(out, r)
	}
	sort.Slice(out, func(a, b int) bool { return out[a].Empleado < out[b].Empleado })
	return out
}

// ---------------------------------------------------------------------------
// El mapeo manual
// ---------------------------------------------------------------------------

// FijarManagerManual declara a mano el jefe de alguien.
//
// Mismo modelo, mismas comprobaciones y mismo almacen que el del IdP: no es un
// segundo sistema paralelo. Lo unico que cambia es el origen, y el origen se
// conserva para que en pantalla se distinga lo que dice el IdP de lo que
// escribio el operador.
func (d *Directorio) FijarManagerManual(empleado, manager, autor string, ahora time.Time) error {
	if strings.TrimSpace(autor) == "" {
		return errValor("una relacion de jerarquia declarada a mano tiene que decir quien la " +
			"declaro: en un producto de cumplimiento las decisiones llevan nombre")
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if u, ok := d.usuarios[empleado]; !ok || !u.Vivo() {
		return errValor("el empleado %q no existe", empleado)
	}
	if u, ok := d.usuarios[manager]; !ok || !u.Vivo() {
		return errValor("el manager %q no existe. Si todavia no se ha aprovisionado, espera "+
			"al siguiente ciclo del IdP y vuelve a declararlo", manager)
	}
	if err := d.validarManagerBloqueado(empleado, manager); err != nil {
		return err
	}
	d.manualPorEmpleado[empleado] = relacionManual{manager: manager, desde: ahora, autor: autor}
	return nil
}

// QuitarManagerManual retira una relacion declarada a mano.
func (d *Directorio) QuitarManagerManual(empleado string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, ok := d.manualPorEmpleado[empleado]; !ok {
		return errNoEncontrado("mapeo manual para el empleado", empleado)
	}
	delete(d.manualPorEmpleado, empleado)
	return nil
}

// validarManagerBloqueado rechaza en el momento de ESCRIBIR lo que la carga
// rechazaria despues: nadie es su propio jefe, y ninguna arista cierra un ciclo.
//
// Comprobarlo al escribir y no solo al cargar no es redundancia. Si solo se
// comprobara al cargar, el IdP recibiria un 200, creeria que su dato entro, y
// el organigrama entero dejaria de construirse hasta que alguien fuera a
// arreglarlo a mano. Al escribir, el IdP recibe un 400 que dice exactamente
// que atributo corregir y en que usuario.
//
// Se llama con el candado de escritura tomado.
func (d *Directorio) validarManagerBloqueado(empleado, manager string) error {
	if empleado == manager {
		return errValor("un usuario no puede ser su propio manager (%s). El escalado subiria "+
			"de el a el mismo y la obligacion vencida no avisaria a nadie", empleado)
	}
	if _, ok := d.usuarios[manager]; !ok {
		return errValor("el manager %q no existe todavia. Los IdP no garantizan el orden de "+
			"aprovisionamiento: si el jefe aun no se ha creado, este PATCH se reintenta en el "+
			"siguiente ciclo y entra solo", manager)
	}
	// Se sube desde el manager propuesto: si por el camino se encuentra al
	// empleado, esta arista cierra un ciclo.
	//
	// HALLAZGO de la pasada del atacante, y era de la clase mala: aqui se
	// abandonaba el recorrido al llegar a MaxProfundidadJerarquia y se
	// devolvia nil, mientras que Jerarquia() rechaza el arbol entero cuando lo
	// pasa. O sea que se podia ESCRIBIR una cadena que luego NO CARGA: el IdP
	// recibia un 200, y a partir de ese momento el organigrama entero dejaba
	// de construirse, y con el el escalado y el diagnostico. La regla tiene
	// que ser la misma en los dos sitios, y ahora lo es.
	cur := manager
	for n := 1; cur != ""; n++ {
		if cur == empleado {
			return errValor("poner a %s de manager de %s cierra un ciclo en la jerarquia. "+
				"Un ciclo cuelga el escalado: la obligacion vencida sube a su jefe, que sube "+
				"al primero, y no avisa nunca", manager, empleado)
		}
		if n > MaxProfundidadJerarquia {
			return errValor("poner a %s de manager de %s deja una cadena de mando de mas de "+
				"%d niveles. Ninguna organizacion real los tiene: casi siempre es un ciclo "+
				"que se cerro por otro camino o un `manager` mal mapeado en el IdP",
				manager, empleado, MaxProfundidadJerarquia)
		}
		cur = d.managerResueltoBloqueado(cur)
	}
	return nil
}

// managerResueltoBloqueado aplica la misma precedencia que Jerarquia: manda el
// IdP, y el manual rellena el hueco. Que sea la MISMA regla en los dos sitios
// es lo que impide que se pueda escribir algo que luego no carga.
func (d *Directorio) managerResueltoBloqueado(id string) string {
	u, ok := d.usuarios[id]
	if !ok || !u.Vivo() {
		return ""
	}
	if u.ManagerIdP != "" {
		return u.ManagerIdP
	}
	if m, ok := d.manualPorEmpleado[id]; ok {
		return m.manager
	}
	return ""
}

// ---------------------------------------------------------------------------
// Las obligaciones huerfanas
// ---------------------------------------------------------------------------

// EstadoResponsable dice si el responsable de una obligacion sigue siendo
// alguien a quien reclamar, y a quien se sube si no.
//
// Existe porque la regla del producto es que una obligacion cuyo responsable
// desaparece NO desaparece: queda visible como huerfana. Una obligacion sin
// dueno es un riesgo, y hacerla invisible es convertir un riesgo en un problema
// aparentemente resuelto.
type EstadoResponsable struct {
	// Sujeto es el identificador por el que se pregunto.
	Sujeto string
	// Nombre para mostrar, aunque el usuario este borrado.
	Nombre string
	// Huerfana es cierto cuando no hay a quien reclamar.
	Huerfana bool
	// Motivo dice por que, en una linea para pantalla.
	Motivo string
	// Escalado es la cadena de jefes vivos y activos a los que subir. Vacia
	// significa que no hay a quien subir, y eso tambien hay que ensenarlo.
	Escalado []string
}

// EstadoDe responde, para un identificador de usuario, si sigue habiendo a
// quien reclamar y a quien escalar.
func (d *Directorio) EstadoDe(sujeto string) EstadoResponsable {
	e := EstadoResponsable{Sujeto: sujeto}
	u, existe := d.Historico(sujeto)
	if !existe {
		e.Huerfana = true
		e.Motivo = "el responsable no esta en el directorio. Si nunca llego a aprovisionarse, " +
			"asigna la obligacion a alguien que si exista"
		return e
	}
	e.Nombre = u.Mostrar
	if e.Nombre == "" {
		e.Nombre = u.UserName
	}
	switch {
	case !u.Vivo():
		e.Huerfana = true
		e.Motivo = "el responsable se borro del IdP el " + u.Borrado.Format("2006-01-02") +
			". La obligacion sigue viva y sin dueno"
	case !u.Activo:
		e.Huerfana = true
		e.Motivo = "el responsable esta desactivado en el IdP. La obligacion sigue viva y sin dueno"
	}
	j, err := d.Jerarquia()
	if err != nil {
		// Un ciclo en la jerarquia no puede impedir contestar quien es el
		// responsable: se contesta lo que se sabe y se dice que el escalado
		// esta roto.
		e.Motivo = strings.TrimSpace(e.Motivo + " Ademas, el escalado no se puede calcular: " + err.Error())
		return e
	}
	for _, m := range j.Cadena(sujeto) {
		if jefe, ok := j.vivos[m]; ok && jefe.Activo {
			e.Escalado = append(e.Escalado, m)
		}
	}
	if e.Huerfana && len(e.Escalado) == 0 {
		e.Motivo += ". Y no hay a quien escalar: no se le conoce jefe. Declaralo a mano en " +
			"Personas o publica el atributo `manager` en el aprovisionamiento del IdP"
	}
	return e
}
