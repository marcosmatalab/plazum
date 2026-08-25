package scim

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// ponerManager pone el manager por la via del IdP, que es lo que hace un PATCH.
func ponerManager(t *testing.T, d *Directorio, empleado, manager string) error {
	t.Helper()
	ops := []Operacion{{Op: "replace", Ruta: "manager",
		Valor: json.RawMessage(`{"value":"` + manager + `"}`)}}
	_, err := d.Parchear(empleado, Parcheo{Operaciones: ops}, ahoraFijo)
	return err
}

// TestLaJerarquiaSeConstruyeYSirveParaEscalar es el control negativo de todo
// este fichero: sin un organigrama que funcione, rechazar ciclos no demuestra
// nada.
func TestLaJerarquiaSeConstruyeYSirveParaEscalar(t *testing.T) {
	d := nuevoDirectorioDePrueba(t)
	dir := crear(t, d, "direccion@ejemplo.es")
	jefa := crear(t, d, "jefa@ejemplo.es")
	ana := crear(t, d, "ana@ejemplo.es")

	if err := ponerManager(t, d, jefa.ID, dir.ID); err != nil {
		t.Fatal(err)
	}
	if err := ponerManager(t, d, ana.ID, jefa.ID); err != nil {
		t.Fatal(err)
	}
	j, err := d.Jerarquia()
	if err != nil {
		t.Fatalf("una jerarquia sana tiene que construirse: %v", err)
	}
	cadena := j.Cadena(ana.ID)
	if len(cadena) != 2 || cadena[0] != jefa.ID || cadena[1] != dir.ID {
		t.Fatalf("la cadena de escalado de Ana es %v; se esperaba [jefa, direccion]", cadena)
	}
	if d := j.Directos(jefa.ID); len(d) != 1 || d[0] != ana.ID {
		t.Errorf("los directos de la jefa son %v", d)
	}
	if sin := j.SinManager(); len(sin) != 1 || sin[0] != dir.ID {
		t.Errorf("sin manager: %v; solo la direccion tenia que estar", sin)
	}
	// Y el origen se conserva.
	r, ok := j.Manager(ana.ID)
	if !ok || r.Origen != OrigenIdP {
		t.Errorf("el origen de la relacion de Ana es %v, se esperaba idp", r.Origen)
	}
}

// TestUnCicloEnLaJerarquiaSeRechaza. Un ciclo cuelga el escalado: la obligacion
// vencida sube a su jefe, que sube al primero, y no avisa nunca.
func TestUnCicloEnLaJerarquiaSeRechaza(t *testing.T) {
	t.Run("de dos", func(t *testing.T) {
		d := nuevoDirectorioDePrueba(t)
		a := crear(t, d, "a@ejemplo.es")
		b := crear(t, d, "b@ejemplo.es")
		if err := ponerManager(t, d, a.ID, b.ID); err != nil {
			t.Fatal(err)
		}
		err := ponerManager(t, d, b.ID, a.ID)
		if err == nil {
			t.Fatal("se cerro un ciclo de dos y el PATCH devolvio exito. El IdP creeria que " +
				"su dato entro y el escalado se quedaria colgado sin que nadie se entere")
		}
		if !strings.Contains(err.Error(), "cierra un ciclo") {
			t.Errorf("se rechazo por otro motivo: %v", err)
		}
	})

	t.Run("de tres", func(t *testing.T) {
		d := nuevoDirectorioDePrueba(t)
		a := crear(t, d, "a@ejemplo.es")
		b := crear(t, d, "b@ejemplo.es")
		c := crear(t, d, "c@ejemplo.es")
		if err := ponerManager(t, d, a.ID, b.ID); err != nil {
			t.Fatal(err)
		}
		if err := ponerManager(t, d, b.ID, c.ID); err != nil {
			t.Fatal(err)
		}
		if err := ponerManager(t, d, c.ID, a.ID); err == nil {
			t.Fatal("se cerro un ciclo de tres")
		}
	})

	t.Run("uno que se pone a si mismo de manager", func(t *testing.T) {
		d := nuevoDirectorioDePrueba(t)
		a := crear(t, d, "a@ejemplo.es")
		err := ponerManager(t, d, a.ID, a.ID)
		if err == nil {
			t.Fatal("alguien se puso a si mismo de manager. El escalado subiria de el a el " +
				"mismo y la obligacion vencida no avisaria a nadie")
		}
		if !strings.Contains(err.Error(), "su propio manager") {
			t.Errorf("se rechazo por otro motivo: %v", err)
		}
	})

	t.Run("en el alta, no solo en el PATCH", func(t *testing.T) {
		d := nuevoDirectorioDePrueba(t)
		a := crear(t, d, "a@ejemplo.es")
		// Crear a alguien apuntando a un manager que existe: bien.
		if _, err := d.Crear(Usuario{UserName: "b@ejemplo.es", Activo: true, ManagerIdP: a.ID},
			ahoraFijo); err != nil {
			t.Fatalf("CONTROL NEGATIVO EN ROJO: crear con un manager valido tiene que "+
				"funcionar: %v", err)
		}
		// Y apuntando a uno que no existe: mal, con mensaje que explica el
		// orden de aprovisionamiento.
		_, err := d.Crear(Usuario{UserName: "c@ejemplo.es", Activo: true, ManagerIdP: "no-existe"},
			ahoraFijo)
		if err == nil {
			t.Fatal("se creo un usuario cuyo manager no existe")
		}
		if !strings.Contains(err.Error(), "siguiente ciclo") {
			t.Errorf("el mensaje no explica que el IdP lo reintenta solo: %v", err)
		}
	})
}

// TestElCicloSeRechazaTambienAlCargar.
//
// Comprobarlo al escribir no basta como unica defensa: los datos pueden llegar
// por una restauracion o por una importacion que no pase por el PATCH. Se
// fabrica el ciclo saltandose la validacion de escritura, a proposito, para
// comprobar que la carga tambien lo caza.
func TestElCicloSeRechazaTambienAlCargar(t *testing.T) {
	d := nuevoDirectorioDePrueba(t)
	a := crear(t, d, "a@ejemplo.es")
	b := crear(t, d, "b@ejemplo.es")

	// Control negativo primero: sin ciclo, la carga funciona.
	if err := ponerManager(t, d, a.ID, b.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := d.Jerarquia(); err != nil {
		t.Fatalf("CONTROL NEGATIVO EN ROJO: una jerarquia sana no carga: %v", err)
	}

	// Y ahora el ciclo, metido por debajo como lo meteria una restauracion.
	d.mu.Lock()
	d.usuarios[b.ID].ManagerIdP = a.ID
	d.mu.Unlock()

	_, err := d.Jerarquia()
	if err == nil {
		t.Fatal("un ciclo metido por debajo (restauracion, importacion) no se caza al cargar")
	}
	if !strings.Contains(err.Error(), "ciclo en la jerarquia") {
		t.Errorf("se rechazo por otro motivo: %v", err)
	}
	// El mensaje tiene que decir CUAL es el ciclo. Uno que solo dice "hay un
	// ciclo" obliga a buscarlo a mano en un organigrama de 200 personas.
	if !strings.Contains(err.Error(), "->") {
		t.Errorf("el mensaje no dibuja el ciclo: %v", err)
	}
}

// ---------------------------------------------------------------------------
// El mapeo manual
// ---------------------------------------------------------------------------

// TestElMapeoManualRellenaLoQueElIdPNoPublica. La mitad de los clientes no
// publica `manager`.
func TestElMapeoManualRellenaLoQueElIdPNoPublica(t *testing.T) {
	d := nuevoDirectorioDePrueba(t)
	jefa := crear(t, d, "jefa@ejemplo.es")
	ana := crear(t, d, "ana@ejemplo.es")

	// Antes de declarar nada: Ana no tiene a quien escalar, y eso se VE.
	j, err := d.Jerarquia()
	if err != nil {
		t.Fatal(err)
	}
	if len(j.SinManager()) != 2 {
		t.Fatalf("con un IdP que no publica manager, los dos tenian que salir sin jefe: %v",
			j.SinManager())
	}

	if err := d.FijarManagerManual(ana.ID, jefa.ID, "operador@ejemplo.es", ahoraFijo); err != nil {
		t.Fatalf("declarar la jerarquia a mano: %v", err)
	}
	j, err = d.Jerarquia()
	if err != nil {
		t.Fatal(err)
	}
	cadena := j.Cadena(ana.ID)
	if len(cadena) != 1 || cadena[0] != jefa.ID {
		t.Fatalf("la cadena de escalado con mapeo manual es %v", cadena)
	}
	// Misma estructura, otro origen, y se ve de donde vino.
	r, ok := j.Manager(ana.ID)
	if !ok {
		t.Fatal("la relacion manual no aparece en el organigrama")
	}
	if r.Origen != OrigenManual {
		t.Errorf("origen %v, se esperaba manual. Si no se distingue, el operador no puede "+
			"saber que parte del organigrama escribio el y que parte dice su IdP", r.Origen)
	}
	if r.Autor != "operador@ejemplo.es" {
		t.Errorf("la relacion manual no dice quien la declaro: %q", r.Autor)
	}
	if !r.Desde.Equal(ahoraFijo) {
		t.Errorf("la relacion manual no dice desde cuando")
	}
}

// TestElMapeoManualNoEsUnSegundoSistema: pasa por las MISMAS comprobaciones.
func TestElMapeoManualNoEsUnSegundoSistema(t *testing.T) {
	d := nuevoDirectorioDePrueba(t)
	a := crear(t, d, "a@ejemplo.es")
	b := crear(t, d, "b@ejemplo.es")

	if err := d.FijarManagerManual(a.ID, a.ID, "op", ahoraFijo); err == nil {
		t.Fatal("por la via manual, alguien se pudo poner a si mismo de manager. Si las " +
			"comprobaciones no son las mismas, el mapeo manual ES un segundo sistema")
	}
	if err := d.FijarManagerManual(a.ID, "no-existe", "op", ahoraFijo); err == nil {
		t.Fatal("por la via manual se pudo apuntar a un manager que no existe")
	}
	if err := d.FijarManagerManual(a.ID, b.ID, "", ahoraFijo); err == nil {
		t.Fatal("se declaro una relacion a mano sin decir quien la declara")
	}
	// El ciclo, mezclando las dos vias: el IdP dice que b reporta a a, y el
	// operador intenta declarar a mano que a reporta a b.
	if err := ponerManager(t, d, b.ID, a.ID); err != nil {
		t.Fatal(err)
	}
	if err := d.FijarManagerManual(a.ID, b.ID, "op", ahoraFijo); err == nil {
		t.Fatal("se cerro un ciclo MEZCLANDO el manager del IdP con el declarado a mano. " +
			"Los ciclos hay que buscarlos en el grafo resuelto, no en cada mitad")
	}
	// Control negativo: una relacion manual legitima si entra.
	c := crear(t, d, "c@ejemplo.es")
	if err := d.FijarManagerManual(c.ID, a.ID, "op", ahoraFijo); err != nil {
		t.Fatalf("CONTROL NEGATIVO EN ROJO: una relacion manual legitima se rechaza: %v", err)
	}
}

// TestElIdPManda: cuando hay las dos, gana el IdP y el conflicto se VE.
func TestElIdPManda(t *testing.T) {
	d := nuevoDirectorioDePrueba(t)
	jefaIdP := crear(t, d, "jefa-idp@ejemplo.es")
	jefaManual := crear(t, d, "jefa-manual@ejemplo.es")
	ana := crear(t, d, "ana@ejemplo.es")

	if err := d.FijarManagerManual(ana.ID, jefaManual.ID, "op", ahoraFijo); err != nil {
		t.Fatal(err)
	}
	if err := ponerManager(t, d, ana.ID, jefaIdP.ID); err != nil {
		t.Fatal(err)
	}
	j, err := d.Jerarquia()
	if err != nil {
		t.Fatal(err)
	}
	r, _ := j.Manager(ana.ID)
	if r.Manager != jefaIdP.ID || r.Origen != OrigenIdP {
		t.Fatalf("manda el IdP mientras hable, y aqui gano %s (%v)", r.Manager, r.Origen)
	}
	cs := j.Conflictos()
	if len(cs) != 1 || cs[0].Empleado != ana.ID {
		t.Fatalf("el conflicto no se ensena: %v. Sin verlo, el operador se pasa meses "+
			"creyendo que el escalado sube por donde no sube", cs)
	}
	if cs[0].SegunIdP != jefaIdP.ID || cs[0].SegunManual != jefaManual.ID {
		t.Errorf("el conflicto no dice las dos versiones: %+v", cs[0])
	}
}

// TestUnaRelacionQueApuntaAAlguienBorradoNoTumbaElOrganigrama.
//
// Escribir es estricto y leer es resistente: un jefe se puede borrar DESPUES,
// y entonces el arbol tiene que seguir construyendose y ensenar la rotura, no
// dejar de construirse mientras corre un plazo.
func TestUnaRelacionQueApuntaAAlguienBorradoNoTumbaElOrganigrama(t *testing.T) {
	d := nuevoDirectorioDePrueba(t)
	jefa := crear(t, d, "jefa@ejemplo.es")
	ana := crear(t, d, "ana@ejemplo.es")
	if err := ponerManager(t, d, ana.ID, jefa.ID); err != nil {
		t.Fatal(err)
	}
	if err := d.Borrar(jefa.ID, ahoraFijo.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	j, err := d.Jerarquia()
	if err != nil {
		t.Fatalf("borrar al jefe tumbo el organigrama entero: %v. Escribir es estricto, "+
			"leer es resistente", err)
	}
	if len(j.Rotas()) != 1 || j.Rotas()[0].Empleado != ana.ID {
		t.Fatalf("la relacion rota no se ensena: %v", j.Rotas())
	}
	if len(j.Cadena(ana.ID)) != 0 {
		t.Error("la cadena de escalado apunta a alguien que ya no esta")
	}
}

// ---------------------------------------------------------------------------
// Las obligaciones huerfanas
// ---------------------------------------------------------------------------

// TestUnaObligacionSinResponsableQuedaVisible.
//
// La regla del producto: una obligacion cuyo responsable desaparece NO
// desaparece. Una obligacion sin dueno es un riesgo, y hacerla invisible es
// convertir un riesgo en un problema aparentemente resuelto.
func TestUnaObligacionSinResponsableQuedaVisible(t *testing.T) {
	d := nuevoDirectorioDePrueba(t)
	jefa := crear(t, d, "jefa@ejemplo.es")
	ana := crear(t, d, "ana@ejemplo.es")
	if err := ponerManager(t, d, ana.ID, jefa.ID); err != nil {
		t.Fatal(err)
	}

	// Control negativo: mientras esta activa, no es huerfana.
	if e := d.EstadoDe(ana.ID); e.Huerfana {
		t.Fatalf("una usuaria activa sale como huerfana: %+v", e)
	}

	// Desactivada en el IdP.
	if _, err := d.Parchear(ana.ID, parcheo(`[{"op":"replace","path":"active","value":false}]`),
		ahoraFijo); err != nil {
		t.Fatal(err)
	}
	e := d.EstadoDe(ana.ID)
	if !e.Huerfana {
		t.Fatal("una obligacion cuyo responsable esta desactivado no sale como huerfana")
	}
	if e.Nombre == "" {
		t.Error("no se puede ensenar el nombre de quien la tenia")
	}
	if !strings.Contains(e.Motivo, "desactivado") {
		t.Errorf("el motivo no lo explica: %q", e.Motivo)
	}
	if len(e.Escalado) != 1 || e.Escalado[0] != jefa.ID {
		t.Fatalf("no se sabe a quien subirla: %v. Ese es todo el sentido del manager", e.Escalado)
	}

	// Borrada del IdP: sigue visible, con la fecha.
	if err := d.Borrar(ana.ID, ahoraFijo.Add(24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	e = d.EstadoDe(ana.ID)
	if !e.Huerfana {
		t.Fatal("tras el borrado deja de verse como huerfana: la obligacion desaparecio con " +
			"su responsable, que es exactamente lo que no puede pasar")
	}
	if !strings.Contains(e.Motivo, "2026-08-26") {
		t.Errorf("el motivo no dice cuando se borro: %q", e.Motivo)
	}
}

// TestUnaHuerfanaSinJefeLoDice. El peor caso: nadie a quien reclamar y nadie a
// quien subir. Tiene que decirse, y decir como se arregla.
func TestUnaHuerfanaSinJefeLoDice(t *testing.T) {
	d := nuevoDirectorioDePrueba(t)
	ana := crear(t, d, "ana@ejemplo.es")
	if _, err := d.Parchear(ana.ID, parcheo(`[{"op":"replace","path":"active","value":false}]`),
		ahoraFijo); err != nil {
		t.Fatal(err)
	}
	e := d.EstadoDe(ana.ID)
	if !e.Huerfana || len(e.Escalado) != 0 {
		t.Fatalf("%+v", e)
	}
	if !strings.Contains(e.Motivo, "no hay a quien escalar") {
		t.Errorf("no se dice que no hay a quien escalar: %q", e.Motivo)
	}
	if !strings.Contains(e.Motivo, "Declaralo a mano") {
		t.Errorf("no se dice como se arregla: %q", e.Motivo)
	}
}

// TestElEscaladoNoSaltaSobreUnJefeDesactivado. Escalar a alguien que ya no
// trabaja aqui es no escalar.
func TestElEscaladoNoSaltaSobreUnJefeDesactivado(t *testing.T) {
	d := nuevoDirectorioDePrueba(t)
	dirg := crear(t, d, "direccion@ejemplo.es")
	jefa := crear(t, d, "jefa@ejemplo.es")
	ana := crear(t, d, "ana@ejemplo.es")
	if err := ponerManager(t, d, jefa.ID, dirg.ID); err != nil {
		t.Fatal(err)
	}
	if err := ponerManager(t, d, ana.ID, jefa.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := d.Parchear(jefa.ID, parcheo(`[{"op":"replace","path":"active","value":false}]`),
		ahoraFijo); err != nil {
		t.Fatal(err)
	}
	e := d.EstadoDe(ana.ID)
	for _, m := range e.Escalado {
		if m == jefa.ID {
			t.Fatal("el escalado incluye a una jefa desactivada. Escalar a alguien que ya no " +
				"trabaja aqui es no escalar")
		}
	}
	if len(e.Escalado) != 1 || e.Escalado[0] != dirg.ID {
		t.Fatalf("el escalado tenia que saltar a la direccion y es %v", e.Escalado)
	}
}

// TestNoSePuedeEscribirUnaJerarquiaQueLuegoNoCarga.
//
// HALLAZGO de la pasada del atacante. La comprobacion de escritura abandonaba
// el recorrido al llegar al limite de profundidad y devolvia exito, mientras
// que la carga rechaza el arbol entero cuando lo pasa. O sea que un IdP podia
// escribir una cadena que despues no cargaba, y a partir de ahi el organigrama
// entero dejaba de construirse: sin escalado y sin diagnostico, en silencio.
//
// La regla tiene que ser la misma en los dos sitios, y este test es lo que lo
// mantiene: construye una cadena hasta pasarse y exige que el rechazo llegue al
// ESCRIBIR, no al leer.
func TestNoSePuedeEscribirUnaJerarquiaQueLuegoNoCarga(t *testing.T) {
	d := nuevoDirectorioDePrueba(t)
	// La cadena crece HACIA ABAJO: cada nuevo usuario reporta al anterior. Es
	// la direccion en la que la comprobacion de escritura tiene trabajo que
	// hacer, porque para validar la arista tiene que subir toda la cadena.
	raiz := crear(t, d, "u0@ejemplo.es")
	previo := raiz.ID
	var rechazado bool
	for n := 1; n < MaxProfundidadJerarquia+10; n++ {
		u := crear(t, d, "u"+itoa(n)+"@ejemplo.es")
		if err := ponerManager(t, d, u.ID, previo); err != nil {
			if !strings.Contains(err.Error(), "niveles") {
				t.Fatalf("se rechazo por otro motivo en el nivel %d: %v", n, err)
			}
			rechazado = true
			break
		}
		previo = u.ID
		// El invariante, comprobado en CADA paso: lo que se acaba de escribir
		// tiene que poder cargar. Es el hallazgo convertido en regresion.
		if _, err := d.Jerarquia(); err != nil {
			t.Fatalf("en el nivel %d se escribio algo que luego no carga: %v. Escribir y "+
				"cargar tienen que aplicar la misma regla", n, err)
		}
	}
	if !rechazado {
		t.Fatalf("se pudo encadenar mas de %d niveles sin que nadie dijera nada",
			MaxProfundidadJerarquia)
	}
}

// TestRecorrerLaCadenaSiempreTermina. Un bucle infinito en el escalado
// significa que una obligacion vencida no avisa a nadie.
//
// Se fabrica el ciclo por debajo, saltandose la validacion de escritura, porque
// esa es la unica forma en que puede aparecer: por una restauracion o una
// importacion. Cadena tiene que terminar igualmente.
func TestRecorrerLaCadenaSiempreTermina(t *testing.T) {
	d := nuevoDirectorioDePrueba(t)
	a := crear(t, d, "a@ejemplo.es")
	b := crear(t, d, "b@ejemplo.es")
	if err := ponerManager(t, d, a.ID, b.ID); err != nil {
		t.Fatal(err)
	}
	j, err := d.Jerarquia()
	if err != nil {
		t.Fatal(err)
	}
	// Se mete el ciclo en la estructura YA construida, que es lo que veria
	// Cadena si un dato corrupto se colara pese a las dos comprobaciones.
	j.rel[b.ID] = Relacion{Empleado: b.ID, Manager: a.ID, Origen: OrigenIdP}

	hecho := make(chan int, 1)
	go func() { hecho <- len(j.Cadena(a.ID)) }()
	select {
	case n := <-hecho:
		if n > MaxProfundidadJerarquia {
			t.Fatalf("la cadena devolvio %d niveles", n)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("recorrer una cadena con ciclo no termino en 5 segundos: el escalado se " +
			"colgaria y la obligacion vencida no avisaria a nadie")
	}
}
