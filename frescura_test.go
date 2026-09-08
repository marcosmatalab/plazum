package plazum

import (
	"regexp"
	"testing"
	"time"
)

// EL MARCADOR TIENE DOS MITADES Y SOLO UNA TENIA RELOJ.
//
// # El hueco, medido
//
// `docs/marcador.md` publica tres cifras que son un ponderado de PESOS y NOTAS.
// Los pesos salen de `docs/diseno.md` §14 y las notas de `docs/instantanea.md`,
// que es una foto del **04-09-2026** y esta congelada a proposito.
// `instantanea_test.go` ata los CARDINALES de esa foto al arbol, y las notas se
// quedan fuera: una nota es un juicio y atarla seria inventarse un criterio
// mecanico para algo que no lo tiene. Asi esta escrito alli y esta bien escrito.
//
// El efecto es que **la mitad con puerta esta fresca y la mitad sin puerta puede
// quedarse parada sin que nada lo diga**. Entre el 04-09 y el 08-09 entraron
// D-22, el eslabon de la prueba entero, A1 a A6, el TTFV honesto, D-23 y catorce
// relojes, y las tres cifras que miden el proyecto no se movieron ni un digito.
//
// # Lo que esta puerta hace, y sobre todo lo que NO hace
//
// **No exige que una nota se mueva.** No puede y no debe: seguiria siendo un
// juicio y forzarlo produciria notas movidas para pasar una puerta, que es peor
// que una nota vieja. Lo que exige es que el marcador **diga de cuando son sus
// notas**, y que por encima del umbral lo diga en su cabecera.
//
// Es la doctrina del descargo aplicada a nosotros mismos. La misma frase que el
// calendario le dice al cliente sobre un vencimiento sin registro —*esto NO dice
// que se haya incumplido, dice que en tus respuestas no consta*— aqui es *esto no
// dice que el proyecto no haya avanzado, dice que estas notas son del 4*.
//
// # Por que no puede convertirse en un rojo permanente
//
// Porque lo que se publica es la FECHA de la nota mas vieja, que es un hecho del
// pasado y no caduca, y no su antiguedad en dias, que cambiaria sola cada
// madrugada y obligaria a editar el fichero a diario. La antiguedad la computa
// este test contra el reloj de hoy y la imprime. El unico dia que hay que tocar
// el fichero es el que se cruza el umbral, y entonces hay dos arreglos y los dos
// son correctos: volver a medir las notas, o escribir el descargo.
//
// # El umbral, con su argumento y con su limite dicho
//
// **14 dias.** El proyecto cierra unas 2,9 casillas al dia y una campana dura
// entre tres y cinco, asi que a los catorce dias las notas son mas viejas que la
// campana que las produjo y es casi seguro que al menos una de las diecisiete ya
// no es cierta.
//
// **Y el umbral es un mal proxy en una direccion concreta, que se dice aqui y no
// se descubre luego**: mide tiempo de calendario y lo que invalida una nota es
// que el arbol se mueva. El 04-09-2026 esta foto se tomo DOS VECES el mismo dia y
// la de la manana ya era falsa por la tarde, con 22 relojes, la capa visual
// entera y los cimientos de la IA aterrizando entre medias. Catorce dias no
// habrian cazado eso. Lo que este umbral caza es el olvido largo, que es el otro
// modo de fallo y el que nadie mira.

const (
	// UmbralDeFrescuraDeLasNotas son los dias que una nota puede tener antes de
	// que el marcador tenga que decir que sus notas estan caducadas.
	UmbralDeFrescuraDeLasNotas = 14

	// NotasDeLaInstantanea es cuantas dimensiones se autoevaluan. Con techo por
	// igualdad exacta: si el conjunto ENCOGE, esta puerta estaria mirando menos
	// notas y creyendose fresca por las que ya no mira.
	NotasDeLaInstantanea = 17
)

var (
	// La columna «Medida» de la tabla de autoevaluacion, que es la quinta.
	reFechaDeLaNota = regexp.MustCompile(
		`(?m)^\|\s*(D\d{1,2})\s*\|[^|]*\|[^|]*\|\s*\*\*\d{1,2},\d\*\*\s*\|\s*` +
			`(\d{2}-\d{2}-\d{4})\s*\|`)
	// Lo que el marcador publica de su propia frescura.
	reNotaMasVieja = regexp.MustCompile(
		`(?s)<!-- frescura:inicio -->.*?La nota más vieja de las (\d+) es del ` +
			`\*\*(\d{2}-\d{2}-\d{4})\*\*.*?<!-- frescura:fin -->`)
	reUmbralPublicado = regexp.MustCompile(
		`(?s)<!-- frescura:inicio -->.*?El umbral son (\d+) días.*?<!-- frescura:fin -->`)
	// El descargo, que solo puede estar cuando toca.
	reNotasCaducadas = regexp.MustCompile(`(?m)^<!-- notas:caducadas -->$`)
)

// fechasDeLasNotas lee la columna «Medida» de las 17 filas de autoevaluacion.
func fechasDeLasNotas(t *testing.T) map[string]time.Time {
	t.Helper()
	inst := leerDoc(t, rutaDeLaInstantanea)
	out := map[string]time.Time{}
	for _, m := range reFechaDeLaNota.FindAllStringSubmatch(inst, -1) {
		if _, repe := out[m[1]]; repe {
			t.Fatalf("%s da la fecha de %s dos veces", rutaDeLaInstantanea, m[1])
		}
		out[m[1]] = aFecha(t, m[2])
	}
	if len(out) != NotasDeLaInstantanea {
		t.Fatalf("%s trae %d notas con fecha y son %d.\n"+
			"  Cada fila de la autoevaluacion lleva su columna «Medida» con la fecha en que "+
			"se emitio ese juicio. Sin ella no se puede saber si el marcador esta "+
			"publicando un numero de hoy o de hace tres semanas.\n"+
			"  Y el numero va con igualdad exacta: si el conjunto ENCOGE, esta puerta "+
			"mira menos notas y se cree fresca por las que ya no mira.",
			rutaDeLaInstantanea, len(out), NotasDeLaInstantanea)
	}
	return out
}

func TestElMarcadorDiceDeCuandoSonSusNotas(t *testing.T) {
	fechas := fechasDeLasNotas(t)

	var masVieja time.Time
	var quien string
	for id, f := range fechas {
		if masVieja.IsZero() || f.Before(masVieja) {
			masVieja, quien = f, id
		}
	}

	marcador := leerDoc(t, rutaDelMarcador)
	m := reNotaMasVieja.FindStringSubmatch(marcador)
	if m == nil {
		t.Fatalf("%s no publica de cuando son sus notas, entre <!-- frescura:inicio --> y "+
			"<!-- frescura:fin --> con «La nota más vieja de las N es del DD-MM-AAAA».\n"+
			"  Es la mitad de este marcador que no tiene puerta y por tanto la que se puede "+
			"quedar parada en silencio. Hoy seria: **La nota más vieja de las %d es del "+
			"%s** (la de %s).",
			rutaDelMarcador, len(fechas), masVieja.Format("02-01-2006"), quien)
	}
	if got := aEntero(t, m[1]); got != len(fechas) {
		t.Errorf("%s dice que las notas son %d y la instantanea trae %d",
			rutaDelMarcador, got, len(fechas))
	}
	if got := m[2]; got != masVieja.Format("02-01-2006") {
		t.Errorf("%s dice que su nota más vieja es del %s y la instantanea da %s (la de %s).\n"+
			"  Se publica la FECHA y no la antiguedad a proposito: una fecha es un hecho del "+
			"pasado y no caduca; los dias los computa este test contra el reloj de hoy, que "+
			"es lo que impide que este fichero haya que editarlo cada madrugada.",
			rutaDelMarcador, got, masVieja.Format("02-01-2006"), quien)
	}

	u := reUmbralPublicado.FindStringSubmatch(marcador)
	if u == nil {
		t.Fatalf("%s no publica su umbral de frescura con «El umbral son N días»", rutaDelMarcador)
	}
	if got := aEntero(t, u[1]); got != UmbralDeFrescuraDeLasNotas {
		t.Errorf("%s publica un umbral de %d días y la puerta usa %d: el documento y quien "+
			"lo vigila tienen que decir el mismo número, o el lector no sabe cuál manda",
			rutaDelMarcador, got, UmbralDeFrescuraDeLasNotas)
	}

	// LA ANTIGUEDAD, CONTRA EL RELOJ DE HOY. Y LAS DOS DIRECCIONES.
	dias := int(time.Since(masVieja).Hours() / 24)
	caducadas := reNotasCaducadas.MatchString(marcador)
	switch {
	case dias > UmbralDeFrescuraDeLasNotas && !caducadas:
		t.Errorf("la nota más vieja de %s es del %s, o sea de hace %d días, y el umbral es "+
			"%d: %s tiene que decirlo en su cabecera y no lo dice.\n"+
			"  ESTA PUERTA NO EXIGE QUE LA NOTA SE MUEVA. Un juicio no se mueve por una "+
			"puerta, y forzarlo daría notas movidas para pasar un test, que es peor que una "+
			"nota vieja. Lo que exige es que el número no parezca de hoy.\n"+
			"  Dos arreglos y los dos valen: volver a medir las diecisiete notas, o escribir "+
			"el descargo con la marca <!-- notas:caducadas --> en su propia línea, diciendo "+
			"que esto NO dice que el proyecto no haya avanzado, dice que estas notas son "+
			"del %s.",
			rutaDeLaInstantanea, masVieja.Format("02-01-2006"), dias,
			UmbralDeFrescuraDeLasNotas, rutaDelMarcador, masVieja.Format("02-01-2006"))
	case dias <= UmbralDeFrescuraDeLasNotas && caducadas:
		t.Errorf("%s lleva la marca <!-- notas:caducadas --> y sus notas son del %s, de hace "+
			"%d días, por debajo del umbral de %d.\n"+
			"  La dirección que se olvida: un descargo que se queda puesto después de "+
			"volver a medir es tan falso como no ponerlo, y además enseña a ignorarlo.\n"+
			"  Arreglo: quitar el descargo, que ya no es cierto.",
			rutaDelMarcador, masVieja.Format("02-01-2006"), dias, UmbralDeFrescuraDeLasNotas)
	}
	t.Logf("notas del marcador: la más vieja es la de %s, del %s, de hace %d días "+
		"(umbral %d, descargo %v)", quien, masVieja.Format("02-01-2006"), dias,
		UmbralDeFrescuraDeLasNotas, caducadas)
}

// CONTROL NEGATIVO DE LOS TRES PATRONES.
//
// El fallo probable es el de siempre en este trio de ficheros: casar una fila de
// otra tabla o una fecha de otra parte del documento. `docs/instantanea.md` esta
// lleno de fechas y de tablas con identificadores de dimension.
func TestLosPatronesDeLaFrescuraNoCazanDeMas(t *testing.T) {
	casos := []struct {
		nombre string
		fuente string
		quiero string
	}{
		{"la fila de autoevaluacion, con su columna Medida",
			"| D1 | Modelo | 9,7 | **9,0** | 04-09-2026 | lo que sostiene |\n", "04-09-2026"},
		{"la fila del marcador, que NO lleva nota en negrita ni fecha",
			"| D1 | Modelo | 12 | 9,0 | dentro |\n", ""},
		{"la fila de diseno, que lleva peso y no nota de hoy",
			"| D1 | Modelo | 12 | **9,7** | hecho | lo que sostiene |\n", ""},
		{"una fecha en la prosa no es la fecha de una nota",
			"la foto se tomo el 04-09-2026 por la tarde\n", ""},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			m := reFechaDeLaNota.FindStringSubmatch(c.fuente)
			got := ""
			if m != nil {
				got = m[2]
			}
			if got != c.quiero {
				t.Errorf("ha sacado %q y esperaba %q: la puerta estaria leyendo la frescura "+
					"de una tabla que no es la de las notas", got, c.quiero)
			}
		})
	}

	// El bloque de frescura, y que no case fuera de sus marcas.
	dentro := "<!-- frescura:inicio -->\nLa nota más vieja de las 17 es del **04-09-2026**.\n" +
		"El umbral son 14 días.\n<!-- frescura:fin -->\n"
	if m := reNotaMasVieja.FindStringSubmatch(dentro); m == nil || m[2] != "04-09-2026" {
		t.Errorf("el patron no reconoce el bloque que el marcador escribe: %v", m)
	}
	fuera := "por ahi abajo dice La nota más vieja de las 17 es del **01-01-2020** y no " +
		"esta en el bloque\n"
	if reNotaMasVieja.MatchString(fuera) {
		t.Error("el patron casa fuera del bloque de frescura: entonces cualquier frase " +
			"parecida del documento vale como declaracion, que es como este numero " +
			"volveria a no tener puerta")
	}

	// Y el descargo tiene que estar ANCLADO en su linea, no dentro de una frase
	// que hable de el: este mismo fichero lo menciona en su godoc.
	if !reNotasCaducadas.MatchString("<!-- notas:caducadas -->\n") {
		t.Error("el patron del descargo no reconoce su propia marca")
	}
	if reNotasCaducadas.MatchString("se escribe <!-- notas:caducadas --> en su linea\n") {
		t.Error("el patron del descargo casa dentro de una frase: entonces explicar el " +
			"descargo cuenta como ponerlo, y la puerta se cumple hablando de ella")
	}
}

// EL CONTROL POSITIVO DE LAS DOS RAMAS, con dato sintetico.
//
// Sin esto, la rama del descargo es un descargo que ninguna entrada alcanza, que
// es lo que M47 dejo escrito: la mutacion lo dejaria verde porque no hay nada que
// romper. Las dos ramas se recorren aqui con fechas fabricadas.
func TestLasDosRamasDelUmbralDeFrescuraSeRecorren(t *testing.T) {
	hoy := time.Now()
	casos := []struct {
		nombre     string
		dias       int
		caducadas  bool
		quieroRoja bool
	}{
		{"fresca y sin descargo: verde", 3, false, false},
		{"fresca y CON descargo: roja, y es la direccion que se olvida", 3, true, true},
		{"vieja y sin descargo: roja", UmbralDeFrescuraDeLasNotas + 1, false, true},
		{"vieja y con descargo: verde", UmbralDeFrescuraDeLasNotas + 1, true, false},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			vieja := hoy.AddDate(0, 0, -c.dias)
			dias := int(time.Since(vieja).Hours() / 24)
			roja := (dias > UmbralDeFrescuraDeLasNotas && !c.caducadas) ||
				(dias <= UmbralDeFrescuraDeLasNotas && c.caducadas)
			if roja != c.quieroRoja {
				t.Errorf("con %d días y descargo=%v la puerta da roja=%v y esperaba %v",
					dias, c.caducadas, roja, c.quieroRoja)
			}
		})
	}
}
