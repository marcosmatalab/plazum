package main

// Los tests del ensayo de restauracion.
//
// Lo que aqui se prueba NO es que el codigo haga lo que dice. Es que el ensayo
// SE VE FALLAR: cada forma de romper una copia tiene su centinela y su test, y
// cada test afirma con errors.Is y no con una subcadena, porque los mensajes de
// esta herramienta son largos y varios comparten palabras ("clave", "keystore",
// "restaurado" salen en cinco de los ocho). Un control negativo que afirmara
// con una subcadena daria verde por el motivo equivocado, que es peor que no
// tenerlo.

import (
	"bytes"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// montar deja una instalacion sembrada, su copia y el fichero de confianza,
// todo dentro de un temporal del test.
func montar(t *testing.T) (trabajo, vivo, replica, confianza string) {
	t.Helper()
	trabajo = t.TempDir()
	vivo = filepath.Join(trabajo, "vivo")
	replica = filepath.Join(trabajo, "replica")
	confianza = filepath.Join(trabajo, "confianza.json")

	esc, err := cargarEscenario()
	if err != nil {
		t.Fatalf("el escenario empotrado no carga: %v", err)
	}
	s, err := Sembrar(vivo, "prueba", esc)
	if err != nil {
		t.Fatalf("no se puede sembrar: %v", err)
	}
	if err := EscribirConfianza(confianza, s.ClaveOperador); err != nil {
		t.Fatal(err)
	}
	if _, err := Copiar(vivo, replica, "2026-08-21T03:10:00Z"); err != nil {
		t.Fatalf("no se puede copiar: %v", err)
	}
	return trabajo, vivo, replica, confianza
}

// restaurarY devuelve el resultado de verificar la copia tal y como este, tras
// destruir el original. Destruirlo NO es ceremonia: sin eso, un fallo de la
// restauracion podria quedar tapado porque lo que se lee sigue siendo el
// original.
func restaurarY(t *testing.T, trabajo, vivo, replica, confianza string) (Resultado, error) {
	t.Helper()
	if err := os.RemoveAll(vivo); err != nil {
		t.Fatal(err)
	}
	restaurado := filepath.Join(trabajo, "restaurado")
	if err := Restaurar(replica, restaurado); err != nil {
		return Resultado{}, err
	}
	return Verificar(restaurado, confianza)
}

func TestElEnsayoCompletoSaleEnVerde(t *testing.T) {
	trabajo, vivo, replica, confianza := montar(t)
	r, err := restaurarY(t, trabajo, vivo, replica, confianza)
	if err != nil {
		t.Fatalf("una copia sana tiene que restaurar y verificar, y ha fallado: %v", err)
	}
	if r.Entradas != 4 {
		t.Errorf("la cadena restaurada tiene %d entradas y el escenario siembra 4", r.Entradas)
	}
	if r.Vivas != 3 {
		t.Errorf("se han abierto %d entradas vivas y tenian que ser 3 (4 menos la suprimida)", r.Vivas)
	}
	if len(r.Supresiones) != 1 {
		t.Fatalf("se esperaba 1 supresion y hay %d: %v", len(r.Supresiones), r.Supresiones)
	}
	// La lapida tiene que LISTAR la base legal, que es la tercera cosa que
	// docs/guia.md pide del restore drill. Sin ella, el auditor sabe que algo
	// se borro y no bajo que amparo.
	if !strings.Contains(r.Supresiones[0], "RGPD art. 17") {
		t.Errorf("la supresion informada no cita la base legal: %q", r.Supresiones[0])
	}
	if r.EvidenciasVivas != 1 {
		t.Errorf("se ha abierto %d evidencia viva y tenia que ser 1: la otra cuelga de la "+
			"entrada suprimida y su clave se destruyo con ella", r.EvidenciasVivas)
	}
}

// La propiedad del titulo de la casilla, en positivo y aislada: la clave de la
// entrada suprimida NO vuelve con la copia.
func TestLaSupresionSobreviveALaRestauracion(t *testing.T) {
	trabajo, vivo, replica, confianza := montar(t)
	if _, err := restaurarY(t, trabajo, vivo, replica, confianza); err != nil {
		t.Fatal(err)
	}
	ks, err := CargarKeystore(filepath.Join(trabajo, "restaurado"))
	if err != nil {
		t.Fatal(err)
	}
	base, err := CargarBase(filepath.Join(trabajo, "restaurado"))
	if err != nil {
		t.Fatal(err)
	}
	if len(base.Cadena.Lapidas) == 0 {
		t.Fatal("la base restaurada no trae lapidas, asi que este test no comprueba nada")
	}
	for _, l := range base.Cadena.Lapidas {
		if _, hay := ks.ClaveEntrada(l.EntradaBorrada); hay {
			t.Errorf("la clave de la entrada %d ha vuelto con la copia. Una restauracion que "+
				"resucita una clave borrada por el derecho de supresion es un incidente de "+
				"proteccion de datos", l.EntradaBorrada)
		}
		// Y la evidencia que colgaba de ella tampoco.
		for _, ev := range base.Evidencias {
			if ev.Entrada != l.EntradaBorrada {
				continue
			}
			if _, hay := ks.ClaveEvidencia(ev.Hash); hay {
				t.Errorf("la clave de la evidencia %s, que colgaba de la entrada suprimida %d, "+
					"ha vuelto con la copia", ev.Hash[:12], l.EntradaBorrada)
			}
		}
	}
}

// EL CONTROL NEGATIVO ENTERO: cada modo de rotura, con SU centinela.
//
// La lista de modos vive en romper.go y no aqui a proposito. Este test recorre
// ModosRotos, asi que anadir un modo sin decir que lo caza obliga a tocar este
// mapa, y un modo que nadie ejercita no puede pasar desapercibido.
func TestCadaCopiaRotaSaleEnRojoConSuCentinela(t *testing.T) {
	esperado := map[string]error{
		"sin-keystore":              ErrFaltaKeystore,
		"entrada-manipulada":        ErrCadenaNoVerifica,
		"lapida-sin-base-legal":     ErrCadenaNoVerifica,
		"clave-resucitada":          ErrClaveResucitada,
		"keystore-viejo":            ErrKeystoreAnteriorAlBorrado,
		"evidencia-sustituida":      ErrEvidenciaNoAbre,
		"ancla-dentro":              ErrAnclaDentroDeLaCopia,
		"manifiesto-fuera-de-sitio": ErrNombreDeArtefactoInvalido,
	}
	if len(esperado) != len(ModosRotos) {
		t.Fatalf("hay %d modos de rotura y %d centinelas esperados. Un modo sin centinela "+
			"declarado es un modo que nadie comprueba", len(ModosRotos), len(esperado))
	}
	for _, m := range ModosRotos {
		t.Run(m.Nombre, func(t *testing.T) {
			quiero, ok := esperado[m.Nombre]
			if !ok {
				t.Fatalf("el modo %q no declara que centinela tiene que salir", m.Nombre)
			}
			trabajo, vivo, replica, confianza := montar(t)
			aplicado, err := RomperReplica(replica, m.Nombre, "prueba")
			if err != nil {
				t.Fatalf("no se pudo aplicar la rotura: %v", err)
			}
			if err := os.RemoveAll(vivo); err != nil {
				t.Fatal(err)
			}
			restaurado := filepath.Join(trabajo, "restaurado")
			errRestaurar := Restaurar(replica, restaurado)
			if errRestaurar == nil {
				usada, err := RomperTrasRestaurar(restaurado, m.Nombre, confianza)
				if err != nil {
					t.Fatalf("no se pudo aplicar la rotura tras restaurar: %v", err)
				}
				if !aplicado && usada == confianza {
					t.Fatalf("el modo %q no se ha aplicado ni antes ni despues de restaurar, "+
						"asi que este subtest esta comprobando una copia sana", m.Nombre)
				}
				_, errRestaurar = Verificar(restaurado, usada)
			}
			if errRestaurar == nil {
				t.Fatalf("la copia rota con %q ha salido en VERDE. Esa comprobacion no vigila "+
					"nada: %s", m.Nombre, m.Caza)
			}
			if !errors.Is(errRestaurar, quiero) {
				t.Fatalf("la copia rota con %q tenia que salir por %v y ha salido por otra cosa.\n"+
					"  Un rojo por el motivo equivocado es un verde disfrazado: la comprobacion\n"+
					"  que se queria ejercitar (%s) no llego a ejecutarse.\n"+
					"  lo que dijo: %v", m.Nombre, quiero, m.Caza, errRestaurar)
			}
		})
	}
}

// El binario, no las funciones: que el codigo de salida distinga "no prueba
// nada" de "no pude ejecutarme". Sin esta separacion, un fallo de compilacion o
// un disco lleno pasarian por control negativo cazado.
func TestElCodigoDeSalidaSeparaElRojoDeLaCopiaDelRojoDelEnsayo(t *testing.T) {
	var salida, errores bytes.Buffer
	if c := ejecutar([]string{"ensayo", "-dir", t.TempDir()}, &salida, &errores); c != salidaOK {
		t.Fatalf("un ensayo sano tiene que salir con 0 y ha salido con %d: %s", c, errores.String())
	}
	salida.Reset()
	errores.Reset()
	if c := ejecutar([]string{"ensayo", "-dir", t.TempDir(), "-romper", "clave-resucitada"},
		&salida, &errores); c != salidaNoPrueba {
		t.Errorf("una copia rota tiene que salir con %d (lo restaurado no prueba nada) y ha "+
			"salido con %d: %s", salidaNoPrueba, c, errores.String())
	}
	salida.Reset()
	errores.Reset()
	if c := ejecutar([]string{"ensayo", "-romper", "modo-que-no-existe"}, &salida, &errores); c != salidaUso {
		t.Errorf("un modo inventado es un error de USO (%d) y ha salido con %d", salidaUso, c)
	}
	salida.Reset()
	errores.Reset()
	if c := ejecutar(nil, &salida, &errores); c != salidaUso {
		t.Errorf("sin argumentos tiene que salir con %d y ha salido con %d", salidaUso, c)
	}
}

// La clave maestra no viaja en la copia, y el ensayo lo DICE en vez de callarlo.
func TestLaCopiaNoLlevaLaClaveMaestra(t *testing.T) {
	trabajo, vivo, replica, confianza := montar(t)
	if _, err := os.Stat(filepath.Join(vivo, NombreMaestra)); err != nil {
		t.Fatalf("la instalacion sembrada tenia que tener %s: %v", NombreMaestra, err)
	}
	if _, err := os.Stat(filepath.Join(replica, NombreMaestra)); !os.IsNotExist(err) {
		t.Fatalf("%s ha viajado en la copia. Una privada que va en cada backup esta en tantos "+
			"sitios como copias haya; su copia es de custodia, no de replica", NombreMaestra)
	}
	r, err := restaurarY(t, trabajo, vivo, replica, confianza)
	if err != nil {
		t.Fatal(err)
	}
	if !r.MaestraAusente {
		t.Error("el ensayo no ha notado que la maestra no esta. Un operador que restaura y no " +
			"lee ese aviso descubre que no puede firmar nada el dia que le toca borrar")
	}
}

// Restaurar falla CERRADO cuando no hay manifiesto: sin el no se puede saber si
// los bytes son los que se copiaron.
func TestRestaurarSeNiegaSinManifiesto(t *testing.T) {
	trabajo, _, replica, _ := montar(t)
	if err := os.Remove(filepath.Join(replica, NombreManifiesto)); err != nil {
		t.Fatal(err)
	}
	err := Restaurar(replica, filepath.Join(trabajo, "restaurado"))
	if !errors.Is(err, ErrCopiaSinManifiesto) {
		t.Fatalf("una replica sin manifiesto tenia que salir por %v y salio por %v",
			ErrCopiaSinManifiesto, err)
	}
}

// Y caza el artefacto cambiado cuando el manifiesto NO se rehace. Es el otro
// lado de la moneda de romper.go: alli se rehace a proposito para que la
// comprobacion de mas abajo llegue a ejecutarse; aqui se comprueba que, sin
// rehacerlo, el manifiesto muerde.
func TestElManifiestoCazaUnArtefactoCambiadoSinRehacerlo(t *testing.T) {
	trabajo, _, replica, _ := montar(t)
	base, err := CargarBase(replica)
	if err != nil {
		t.Fatal(err)
	}
	base.Generacion = "2000-01-01T00:00:00Z"
	if err := escribirJSON(filepath.Join(replica, NombreBase), base); err != nil {
		t.Fatal(err)
	}
	err = Restaurar(replica, filepath.Join(trabajo, "restaurado"))
	if !errors.Is(err, ErrArtefactoNoCuadra) {
		t.Fatalf("un artefacto cambiado tenia que salir por %v y salio por %v",
			ErrArtefactoNoCuadra, err)
	}
}

// Un ensayo sobre una cadena SIN lapidas se niega. Es el guardia contra el peor
// verde de esta herramienta: si el escenario perdiera su borrado, el ensayo
// seguiria restaurando y verificando, y estaria comprobando la mitad que no
// importa mientras el informe dice "restore drill en verde".
func TestUnEnsayoSobreUnaCadenaSinLapidasSeNiega(t *testing.T) {
	trabajo, vivo, replica, confianza := montar(t)
	base, err := CargarBase(replica)
	if err != nil {
		t.Fatal(err)
	}
	base.Cadena.Lapidas = nil
	if err := escribirJSON(filepath.Join(replica, NombreBase), base); err != nil {
		t.Fatal(err)
	}
	if err := rehacerManifiesto(replica); err != nil {
		t.Fatal(err)
	}
	_, err = restaurarY(t, trabajo, vivo, replica, confianza)
	if !errors.Is(err, ErrSinLapidas) {
		t.Fatalf("una cadena sin supresiones tenia que salir por %v y salio por %v",
			ErrSinLapidas, err)
	}
}

// La clave resucitada bajo OTRO indice. Es la comprobacion cara de verificar.go
// y merece su propio test, porque la barata (mirar si falta la clave de esa
// entrada) da verde aqui.
func TestUnaClaveResucitadaBajoOtroIndiceTampocoCuela(t *testing.T) {
	trabajo, vivo, replica, confianza := montar(t)
	base, err := CargarBase(replica)
	if err != nil {
		t.Fatal(err)
	}
	ks, err := CargarKeystore(replica)
	if err != nil {
		t.Fatal(err)
	}
	borrada := base.Cadena.Lapidas[0].EntradaBorrada
	// La clave de la entrada suprimida, apuntada a una direccion de evidencia
	// que no existe: no esta en Entradas[borrada], asi que la comprobacion
	// barata no la ve.
	ks.Evidencias["0000000000000000000000000000000000000000000000000000000000000000"] =
		hex.EncodeToString(derivar("prueba", "entrada", borrada, 32))
	if err := escribirJSON(filepath.Join(replica, NombreKeystore), ks); err != nil {
		t.Fatal(err)
	}
	if err := rehacerManifiesto(replica); err != nil {
		t.Fatal(err)
	}
	_, err = restaurarY(t, trabajo, vivo, replica, confianza)
	if !errors.Is(err, ErrSupresionLegible) {
		t.Fatalf("una clave de entrada suprimida escondida en otro hueco del keystore tenia "+
			"que salir por %v y salio por %v", ErrSupresionLegible, err)
	}
}

// HALLAZGO DE LA PASADA ADVERSARIA, en escritura arbitraria. Restaurar recorre
// lo que el manifiesto DECLARA y hace filepath.Join con el destino, asi que un
// manifiesto con "../../algo" escribia fuera del destino con los permisos del
// que restaura. Y una replica vive en un bucket o en un NAS, que es justo donde
// no se puede dar por hecho que nadie escribe.
func TestRestaurarSeNiegaAUnNombreDeArtefactoQueSaleDelDirectorio(t *testing.T) {
	for _, nombre := range []string{
		"../fuera.json",
		"..",
		"sub/dentro.json",
		`..\fuera.json`,
	} {
		t.Run(nombre, func(t *testing.T) {
			trabajo, _, replica, _ := montar(t)
			var m Manifiesto
			if err := leerJSON(filepath.Join(replica, NombreManifiesto), &m); err != nil {
				t.Fatal(err)
			}
			// Se le da el hash del keystore, que existe, para que la unica
			// razon posible de rechazo sea el NOMBRE y no que el fichero falte.
			m.Artefactos[nombre] = m.Artefactos[NombreKeystore]
			if err := escribirJSON(filepath.Join(replica, NombreManifiesto), m); err != nil {
				t.Fatal(err)
			}
			err := Restaurar(replica, filepath.Join(trabajo, "restaurado"))
			if !errors.Is(err, ErrNombreDeArtefactoInvalido) {
				t.Fatalf("un manifiesto que declara %q tenia que salir por %v y salio por %v",
					nombre, ErrNombreDeArtefactoInvalido, err)
			}
		})
	}
}

// El otro hallazgo de la misma pasada, y es de proteccion de datos. Antes se
// miraba solo el disco, y el bucle de copia recorre el manifiesto: una replica
// con keystore.json en el disco pero SIN su entrada en el manifiesto pasaba y no
// copiaba el keystore. Restaurando sobre la instalacion que se quiere reparar,
// que es el caso normal, el keystore VIEJO se quedaba donde estaba y la clave
// destruida volvia sin que nadie tocara una clave.
func TestRestaurarExigeElKeystoreEnElManifiestoYNoSoloEnElDisco(t *testing.T) {
	trabajo, _, replica, _ := montar(t)
	var m Manifiesto
	if err := leerJSON(filepath.Join(replica, NombreManifiesto), &m); err != nil {
		t.Fatal(err)
	}
	delete(m.Artefactos, NombreKeystore) // el fichero SIGUE en el disco
	if err := escribirJSON(filepath.Join(replica, NombreManifiesto), m); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(replica, NombreKeystore)); err != nil {
		t.Fatalf("este test necesita que el keystore siga en el disco de la replica: %v", err)
	}
	err := Restaurar(replica, filepath.Join(trabajo, "restaurado"))
	if !errors.Is(err, ErrFaltaKeystore) {
		t.Fatalf("un manifiesto que no lista el keystore tenia que salir por %v y salio por %v",
			ErrFaltaKeystore, err)
	}
}

func TestDentroDeReconoceLoQueEstaDentroYLoQueNo(t *testing.T) {
	base := t.TempDir()
	casos := []struct {
		nombre string
		ruta   string
		quiero bool
	}{
		{"el propio directorio", base, true},
		{"un fichero dentro", filepath.Join(base, "confianza.json"), true},
		{"un fichero en un subdirectorio", filepath.Join(base, "sub", "c.json"), true},
		{"el padre", filepath.Dir(base), false},
		{"un hermano", filepath.Join(filepath.Dir(base), "otro-sitio", "c.json"), false},
		// El caso que se escapa si se compara con strings.HasPrefix: un
		// directorio hermano cuyo nombre EMPIEZA por el del nuestro.
		{"un hermano con el nombre por prefijo", base + "-replica", false},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			if got := DentroDe(base, c.ruta); got != c.quiero {
				t.Errorf("DentroDe(%q, %q) = %v y tenia que ser %v", base, c.ruta, got, c.quiero)
			}
		})
	}
}

// El escenario tiene lo que el ensayo dice comprobar. Si alguien le quita el
// borrado o la base legal, esto lo dice antes que un ensayo a medias.
func TestElEscenarioTraeUnBorradoConBaseLegalYUnaEvidenciaQueCuelgaDeEl(t *testing.T) {
	esc, err := cargarEscenario()
	if err != nil {
		t.Fatal(err)
	}
	if esc.Borrado.BaseLegal == "" {
		t.Fatal("el escenario borra sin base legal, y un borrado sin base legal no es legal")
	}
	if esc.Borrado.Instante == "" {
		t.Fatal("el borrado no declara instante, asi que no se puede contrastar con la " +
			"generacion del keystore, que es la forma normal de resucitar una clave")
	}
	cuelga := false
	for _, ev := range esc.Evidencias {
		if ev.Entrada == esc.Borrado.Entrada {
			cuelga = true
		}
	}
	if !cuelga {
		t.Error("ninguna evidencia cuelga de la entrada que se borra, asi que el ensayo no " +
			"comprueba que borrar una entrada borra tambien su PDF")
	}
}

// La puerta del workflow: todo modo de rotura se ejercita en CI.
//
// POR QUE HACE FALTA. Los siete modos existen en el codigo; lo que hace que
// sean una puerta es que CI los corra. Un modo que se anade y no se cablea al
// workflow es codigo muerto que da la impresion de estar vigilando.
func TestElWorkflowEjercitaTodosLosModosRotos(t *testing.T) {
	ruta := filepath.Join("..", "..", ".github", "workflows", "etapa2-copias.yml")
	b, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatalf("no esta %s, que es donde corre el ensayo de restauracion: %v", ruta, err)
	}
	cuerpo := string(b)
	for _, m := range ModosRotos {
		if !strings.Contains(cuerpo, m.Nombre) {
			t.Errorf("%s no ejercita el modo %q. Existe en el codigo y no corre en CI, "+
				"asi que no vigila nada", ruta, m.Nombre)
		}
	}
}

// Y la documentacion, que en esta casilla es la mitad del entregable: un
// operador tiene que poder leer QUE se copia, QUE no y COMO se restaura sin
// abrir el codigo.
func TestLaDocumentacionDeCopiasDiceLoQueTieneQueDecir(t *testing.T) {
	ruta := filepath.Join("..", "..", "docs", "copias.md")
	b, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatalf("no esta %s: %v", ruta, err)
	}
	cuerpo := strings.ToLower(string(b))
	for _, quiero := range []struct{ que, porque string }{
		{"litestream", "es la herramienta que replica la base y el documento va de eso"},
		{"keystore", "es el artefacto cuya replica separada sostiene el borrado legal"},
		// "35 d" y no "35 dias": el documento es para el usuario final y lleva
		// tildes, este fichero no. El prefijo casa con las dos formas.
		{"35 d", "es el plazo declarado de retencion, y es lo que hace efectivo un borrado " +
			"frente al mundo y no solo frente a la instancia viva"},
		{"maestra", "la clave del operador no se replica y el operador tiene que saberlo antes " +
			"de necesitarla"},
		{"ensayocopia", "sin el nombre de la herramienta, el procedimiento no se puede seguir"},
	} {
		if !strings.Contains(cuerpo, quiero.que) {
			t.Errorf("docs/copias.md no menciona %q, y %s", quiero.que, quiero.porque)
		}
	}
}
