package main

// La puerta del demo publicado.
//
// Esta herramienta se ejecuta a mano, pero su comparacion corre en CI: si
// alguien edita expediente-demo.json o contexto-demo.json a mano, o toca el
// escenario y se olvida de regenerar, aqui se pone rojo y dice en que linea.
// Sin esto el generador seria una sugerencia.
//
// Este fichero SI puede nombrar el ENS, el RGPD y el CRA: herramientas/ es
// donde el codigo se encuentra con el corpus, y TestNingunaNormaCableada solo
// vigila los _test.go de nucleo/ y adaptadores/. main.go no puede, y por eso el
// escenario vive en escenario.json.

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func raizDePrueba(t *testing.T) string {
	t.Helper()
	raiz, err := raizDelRepo()
	if err != nil {
		t.Fatal(err)
	}
	return raiz
}

func escenarioDePrueba(t *testing.T) (*Escenario, []byte) {
	t.Helper()
	raiz := raizDePrueba(t)
	esc, err := leerEscenario(filepath.Join(raiz, rutaEscenario))
	if err != nil {
		t.Fatal(err)
	}
	sello, err := leerSello(filepath.Join(raiz, rutaSello))
	if err != nil {
		t.Fatal(err)
	}
	return esc, sello
}

func generadoDePrueba(t *testing.T) *Generado {
	t.Helper()
	g, err := generar(raizDePrueba(t))
	if err != nil {
		t.Fatal(err)
	}
	return g
}

// La propiedad principal: el demo publicado es EXACTAMENTE lo que sale del
// escenario. Byte a byte, no "equivalente".
func TestElDemoPublicadoSaleDeEsteGenerador(t *testing.T) {
	raiz := raizDePrueba(t)
	g := generadoDePrueba(t)

	for _, c := range []struct {
		ruta   string
		quiero []byte
	}{
		{rutaDemo, g.Demo},
		{rutaContexto, g.Contexto},
	} {
		publicado, err := os.ReadFile(filepath.Join(raiz, c.ruta))
		if err != nil {
			t.Fatal(err)
		}
		if difs := diferencias(publicado, c.quiero); len(difs) > 0 {
			t.Errorf("%s no es lo que sale del escenario. Si el cambio es a proposito:\n"+
				"  go run ./herramientas/generardemo -escribir\n%s",
				c.ruta, strings.Join(difs, "\n"))
		}
	}
}

// Control negativo del anterior. Un test de igualdad que no se pone rojo cuando
// el contenido cambia no es una puerta. Se toca UN campo del escenario en
// memoria y la comparacion tiene que cazarlo en los dos ficheros: en el
// expediente por el nombre, y en el contexto porque el ancla del paquete sale
// del contenido.
func TestLaComparacionSePoneRojaSiElEscenarioCambia(t *testing.T) {
	raiz := raizDePrueba(t)
	esc, sello := escenarioDePrueba(t)

	esc.Organizacion = "Otra organizacion"
	esc.Obligaciones[0].Afirmacion = "otra afirmacion cualquiera"
	g, err := Montar(esc, sello)
	if err != nil {
		t.Fatal(err)
	}

	for _, c := range []struct {
		ruta   string
		quiero []byte
	}{
		{rutaDemo, g.Demo},
		{rutaContexto, g.Contexto},
	} {
		publicado, err := os.ReadFile(filepath.Join(raiz, c.ruta))
		if err != nil {
			t.Fatal(err)
		}
		if len(diferencias(publicado, c.quiero)) == 0 {
			t.Errorf("cambiando el escenario, %s tenia que salir distinto y sale igual: "+
				"la comparacion no esta mirando nada", c.ruta)
		}
	}
}

// El sello RFC 3161 se LEE y no se regenera. Sale a la red y lo repone
// herramientas/sellardemo; esta herramienta solo lo empotra. Si algun dia
// alguien lo genera aqui, el CI dejaria de ser hermetico.
func TestElSelloSeLeeYNoSeRegenera(t *testing.T) {
	raiz := raizDePrueba(t)
	enDisco, err := os.ReadFile(filepath.Join(raiz, rutaSello))
	if err != nil {
		t.Fatal(err)
	}
	g := generadoDePrueba(t)
	if len(g.Expediente.Cadena.Checkpoints) == 0 {
		t.Fatal("el demo tiene que traer al menos un checkpoint")
	}
	if !bytes.Equal(g.Expediente.Cadena.Checkpoints[0].Token, enDisco) {
		t.Fatal("el token del checkpoint no es el fichero de testdata: alguien lo esta " +
			"fabricando aqui en vez de leerlo")
	}

	// Control negativo: con otro sello, el token del checkpoint cambia. Sin
	// esto, la igualdad de arriba podria estar comparando dos ceros.
	otro := append([]byte{}, enDisco...)
	otro[0] ^= 0xff
	g2, err := montarConSello(t, otro)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(g2.Expediente.Cadena.Checkpoints[0].Token, enDisco) {
		t.Fatal("cambiando el sello, el token del checkpoint tenia que cambiar")
	}
}

// montarConSello monta el escenario publicado con OTRO sello. Se salta la
// verificacion porque un sello falseado no verifica, que es justo lo que se
// quiere: aqui solo se mira de donde sale el token.
func montarConSello(t *testing.T, sello []byte) (*Generado, error) {
	t.Helper()
	esc, _ := escenarioDePrueba(t)
	e, err := construir(esc, sello)
	if err != nil {
		return nil, err
	}
	return &Generado{Expediente: e}, nil
}

// Sin sello no se publica. La alternativa, un token de relleno, es la peor
// primera impresion posible: lo primero que hace cualquiera es verificar el
// demo, y con un sello inventado eso falla.
func TestSinSelloNoSePublica(t *testing.T) {
	for _, caso := range []struct {
		nombre string
		monta  func(dir string) string
	}{
		{"ausente", func(dir string) string { return filepath.Join(dir, "no-existe.bin") }},
		{"vacio", func(dir string) string {
			r := filepath.Join(dir, "vacio.bin")
			if err := os.WriteFile(r, nil, 0o600); err != nil {
				t.Fatal(err)
			}
			return r
		}},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			_, err := leerSello(caso.monta(t.TempDir()))
			if !errors.Is(err, ErrSinSello) {
				t.Fatalf("esperaba ErrSinSello y obtuve %v", err)
			}
			if !strings.Contains(err.Error(), "sellardemo") {
				t.Errorf("el error tiene que decir como reponerlo, y dijo: %v", err)
			}
		})
	}
}

// El escenario es un fichero de datos que se edita a mano, o sea que es entrada
// hostil por definicion, aunque el hostil sea uno mismo a las dos de la manana.
// Un emisor que declara aplicable una obligacion que sus reglas no derivan no
// puede publicar: la verificacion se hace ANTES de escribir.
func TestUnEscenarioQueDeclaraDeMasNoSePublica(t *testing.T) {
	esc, sello := escenarioDePrueba(t)
	esc.Declarado.Aplicables = append(esc.Declarado.Aplicables, "iso27001.a.5.1")
	_, err := Montar(esc, sello)
	if !errors.Is(err, ErrNoVerifica) {
		t.Fatalf("declarar aplicable algo que ninguna regla deriva tiene que impedir la "+
			"publicacion; obtuve %v", err)
	}
	if !strings.Contains(err.Error(), "sellardemo") {
		t.Errorf("el error tiene que decir que hacer, y dijo: %v", err)
	}
}

// Y el mismo ataque por el otro lado: falsear un vencimiento que el reloj no
// calcula. Es lo que separa un expediente verificable de un PDF.
func TestUnVencimientoFalseadoEnElEscenarioNoSePublica(t *testing.T) {
	esc, sello := escenarioDePrueba(t)
	var tocado bool
	for i := range esc.Declarado.Reclamaciones {
		if esc.Declarado.Reclamaciones[i].Hito == "notificacion_72h" {
			esc.Declarado.Reclamaciones[i].Vence =
				esc.Declarado.Reclamaciones[i].Vence.AddDate(0, 0, 3)
			tocado = true
		}
	}
	if !tocado {
		t.Fatal("el escenario ya no tiene el hito notificacion_72h: este test no prueba nada")
	}
	if _, err := Montar(esc, sello); !errors.Is(err, ErrNoVerifica) {
		t.Fatalf("un vencimiento falseado tiene que impedir la publicacion; obtuve %v", err)
	}
}

// Una errata en el nombre de una seccion del escenario haria desaparecer esa
// seccion del expediente EN SILENCIO. Con DisallowUnknownFields sale por
// pantalla. Es el mismo patron "tapado" del que este proyecto se defiende.
func TestUnaErrataEnElEscenarioNoSeTragaEnSilencio(t *testing.T) {
	bueno := []byte(`{"organizacion":"x","observaciones":[]}`)
	if _, err := LeerEscenario(bueno); err != nil {
		t.Fatalf("control positivo: este escenario minimo tenia que leerse, y dio %v", err)
	}
	malo := []byte(`{"organizacion":"x","obsevaciones":[]}`)
	if _, err := LeerEscenario(malo); !errors.Is(err, ErrEscenario) {
		t.Fatalf("una seccion mal escrita tiene que cazarse, y obtuve %v", err)
	} else if !strings.Contains(err.Error(), "obsevaciones") {
		t.Errorf("el error tiene que decir QUE campo, y dijo: %v", err)
	}
}

// Y lo que va DETRAS del objeto. Un decodificador de JSON se para en la primera
// llave que cierra: dos escenarios pegados usarian el primero y tirarian el
// segundo sin decir nada, que es el peor resultado posible en un fichero que
// decide lo que se publica.
func TestUnSegundoObjetoPegadoAlEscenarioNoSeIgnora(t *testing.T) {
	uno := []byte(`{"organizacion":"x"}`)
	if _, err := LeerEscenario(uno); err != nil {
		t.Fatalf("control positivo: un objeto solo tenia que leerse, y dio %v", err)
	}
	dos := []byte(`{"organizacion":"x"}{"organizacion":"y"}`)
	if _, err := LeerEscenario(dos); !errors.Is(err, ErrEscenario) {
		t.Fatalf("lo que va detras del escenario tiene que cazarse, y obtuve %v", err)
	}
}

// Las anclas que se publican tienen que cuadrar con el expediente que se
// publica. Es la mitad del P1 que se esta cerrando: antes eran dos ficheros
// editados a mano que podian separarse sin que nadie lo notara.
func TestLasAnclasPublicadasCuadranConElExpediente(t *testing.T) {
	g := generadoDePrueba(t)
	var ctx contextoPublicado
	if err := json.Unmarshal(g.Contexto, &ctx); err != nil {
		t.Fatal(err)
	}
	if len(ctx.Anclas) != len(g.Expediente.Paquetes) {
		t.Fatalf("el contexto publica %d anclas y el expediente trae %d paquetes",
			len(ctx.Anclas), len(g.Expediente.Paquetes))
	}
	for _, p := range g.Expediente.Paquetes {
		if ctx.Anclas[p.URN] != p.Digest {
			t.Errorf("el ancla publicada de %s (%s) no es el digest del paquete (%s)",
				p.URN, ctx.Anclas[p.URN], p.Digest)
		}
	}
	if len(ctx.ClavesConfiables) == 0 || ctx.ClaveOperador != ctx.ClavesConfiables[0] {
		t.Errorf("el contexto tiene que publicar la clave del operador del demo: %+v", ctx)
	}
	if _, err := hex.DecodeString(ctx.ClaveOperador); err != nil {
		t.Errorf("la clave del operador tiene que ser hexadecimal: %v", err)
	}

	// Control negativo: si el ancla se despega del digest, esto se cae.
	for u := range ctx.Anclas {
		ctx.Anclas[u] = "sha256:00"
		break
	}
	cuadran := true
	for _, p := range g.Expediente.Paquetes {
		if ctx.Anclas[p.URN] != p.Digest {
			cuadran = false
		}
	}
	if cuadran {
		t.Error("la comprobacion de anclas no mira nada: con un ancla cambiada sigue cuadrando")
	}
}

// El calendario se declara una vez en el escenario y se le pega a cada reloj.
// Un reloj sin calendario computaria los plazos habiles como naturales, que es
// una respuesta incorrecta y no un fallo visible.
func TestCadaRelojSaleConSuCalendario(t *testing.T) {
	g := generadoDePrueba(t)
	if len(g.Expediente.Relojes) == 0 {
		t.Fatal("el demo tiene que traer relojes")
	}
	for _, r := range g.Expediente.Relojes {
		if r.Calendario.ID == "" || len(r.Calendario.Festivos) == 0 {
			t.Errorf("el reloj de %s sale sin calendario: %+v", r.Obligacion, r.Calendario)
		}
	}
}

// Dos generaciones seguidas dan el mismo fichero. Si no, "el demo cambio" no
// significaria nada y esta puerta no se podria mantener.
func TestLaGeneracionEsDeterminista(t *testing.T) {
	a := generadoDePrueba(t)
	b := generadoDePrueba(t)
	if !bytes.Equal(a.Demo, b.Demo) || !bytes.Equal(a.Contexto, b.Contexto) {
		t.Fatal("dos generaciones del mismo escenario tienen que dar los mismos bytes")
	}
}

// Y la comprobacion de que el demo ensena normas REALES, que es la razon de que
// esta herramienta exista. Un demo con identificadores sinteticos verificaria
// igual y no valdria para nada.
func TestElDemoEnsenaNormasReales(t *testing.T) {
	g := generadoDePrueba(t)
	quiero := []string{"ens@2022.311", "rgpd@2016.679", "cra@2024.2847"}
	tengo := map[string]bool{}
	for _, p := range g.Expediente.Paquetes {
		tengo[p.URN] = true
	}
	for _, u := range quiero {
		if !tengo[u] {
			t.Errorf("el demo tiene que traer %s: es lo que le demuestra algo a quien lo abre", u)
		}
	}
	for _, o := range g.Expediente.Obligaciones {
		if o.Articulo == "" {
			t.Errorf("la obligacion %s sale sin articulo, y el articulo es lo que se ensena", o.ID)
		}
	}
}
