package actualizador

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"dutiq/puertos"
	"dutiq/puertos/contrato"
)

// Los identificadores que salen aqui son SINTETICOS a proposito: este arbol
// esta vigilado por TestNingunaNormaCableada y un adaptador no puede nombrar una
// norma ni en sus pruebas.

const (
	versionVieja = "v0.1.0"
	versionNueva = "v0.2.0"
)

func digest(b []byte) string {
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}

// canalDePrueba monta un canal de directorio con dos versiones y devuelve su
// ruta. El contenido de cada fichero se deriva de la version, para que un test
// pueda comprobar QUE se instalo y no solo que se instalo algo.
func canalDePrueba(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	var vs []Version
	for _, v := range []string{versionNueva, versionVieja} {
		fich := ficherosDe(v)
		digests := map[string]string{}
		for rel, b := range fich {
			digests[rel] = digest(b)
			ruta := filepath.Join(dir, v, filepath.FromSlash(rel))
			if err := os.MkdirAll(filepath.Dir(ruta), 0o750); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(ruta, b, 0o600); err != nil {
				t.Fatal(err)
			}
		}
		vs = append(vs, Version{Version: v, Notas: "notas de " + v, Ficheros: digests})
	}
	if err := EscribirCatalogo(dir, vs); err != nil {
		t.Fatal(err)
	}
	return dir
}

func ficherosDe(v string) map[string][]byte {
	return map[string][]byte{
		"dutiq":                              []byte("binario de " + v),
		"paquetes/demo/paquete.json":         []byte(`{"urn":"urn:demo:x","version":"` + v + `"}`),
		"paquetes/demo/pruebas/caso.json":    []byte(`[{"caso":"de ` + v + `"}]`),
		"plantillas/hoy.html":                []byte("<p>" + v + "</p>"),
		"documentacion/CAMBIOS-" + v + ".md": []byte("cambios de " + v),
	}
}

func nuevoActualizador(t *testing.T) (*Actualizador, string) {
	t.Helper()
	raiz := t.TempDir()
	canal := canalDePrueba(t)
	a := Nuevo(Opciones{
		Raiz:  raiz,
		Canal: CanalDirectorio{Dir: canal},
		Ahora: time.Date(2026, 8, 25, 10, 0, 0, 0, time.UTC),
	})
	return a, raiz
}

// La suite de contrato. Es la puerta de entrada: si esto no pasa, el adaptador
// no se integra, por muy bien que se porte en sus propios tests.
func TestElActualizadorCumpleElContrato(t *testing.T) {
	contrato.Actualizador(t, func() puertos.Actualizador {
		a, _ := nuevoActualizador(t)
		return a
	}, versionNueva)
}

func TestAplicarInstalaExactamenteLoQueDeclaraElCatalogo(t *testing.T) {
	a, raiz := nuevoActualizador(t)
	if _, err := a.Aplicar(context.Background(), versionNueva); err != nil {
		t.Fatalf("aplicar: %v", err)
	}
	for rel, quiero := range ficherosDe(versionNueva) {
		b, err := os.ReadFile(filepath.Join(raiz, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("%s no se instalo: %v", rel, err)
		}
		if string(b) != string(quiero) {
			t.Errorf("%s contiene %q y el catalogo declaraba %q", rel, b, quiero)
		}
	}
	v, err := a.VersionInstalada()
	if err != nil {
		t.Fatal(err)
	}
	if v != versionNueva {
		t.Errorf("VERSION dice %q y se instalo %q: la instalacion miente sobre si misma", v, versionNueva)
	}
}

// La propiedad que da nombre a la casilla: deshacer DESHACE. No basta con que
// devuelva nil, tiene que dejar los bytes de antes.
func TestDeshacerDevuelveLosBytesDeAntes(t *testing.T) {
	a, raiz := nuevoActualizador(t)
	ctx := context.Background()

	if _, err := a.Aplicar(ctx, versionVieja); err != nil {
		t.Fatalf("instalacion inicial: %v", err)
	}
	punto, err := a.Aplicar(ctx, versionNueva)
	if err != nil {
		t.Fatalf("aplicar la nueva: %v", err)
	}
	if err := a.Deshacer(ctx, punto); err != nil {
		t.Fatalf("deshacer: %v", err)
	}

	for rel, quiero := range ficherosDe(versionVieja) {
		b, err := os.ReadFile(filepath.Join(raiz, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("tras deshacer falta %s: %v", rel, err)
		}
		if string(b) != string(quiero) {
			t.Errorf("tras deshacer, %s contiene %q y tenia que contener %q", rel, b, quiero)
		}
	}
	if v, _ := a.VersionInstalada(); v != versionVieja {
		t.Errorf("tras deshacer VERSION dice %q y tenia que decir %q", v, versionVieja)
	}
	// Y el fichero que SOLO traia la version nueva tiene que haber desaparecido:
	// dejarlo puesto seria una instalacion que dice ser la vieja con trozos de
	// la nueva dentro, o sea un estado que nadie ha probado.
	soloEnLaNueva := "documentacion/CAMBIOS-" + versionNueva + ".md"
	if _, err := os.Stat(filepath.Join(raiz, filepath.FromSlash(soloEnLaNueva))); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("%s sigue ahi tras deshacer: la vuelta atras no retira lo que la version nueva "+
			"anadio, y la instalacion queda mezclada", soloEnLaNueva)
	}
}

// La exigencia del enunciado: deshacer un punto INVENTADO falla en vez de
// fingir. Se comprueba ademas que no toca nada, que es la mitad que importa.
func TestDeshacerUnPuntoInventadoFallaYNoTocaNada(t *testing.T) {
	a, raiz := nuevoActualizador(t)
	ctx := context.Background()
	if _, err := a.Aplicar(ctx, versionNueva); err != nil {
		t.Fatal(err)
	}
	antes := instantanea(t, raiz)

	for _, inventado := range []string{
		"punto-que-no-existe",
		"20260825T100000Z-0000000000000000",
		"", ".", "..",
		"../../etc",
		strings.Repeat("a", LongitudMaximaDeNombre+1),
	} {
		err := a.Deshacer(ctx, inventado)
		if err == nil {
			t.Errorf("deshacer %q devolvio exito. El operador creeria que ha vuelto atras "+
				"y no ha vuelto, que es peor que saber que no puede", inventado)
			continue
		}
		if !errors.Is(err, ErrPuntoDesconocido) {
			t.Errorf("deshacer %q tenia que fallar con ErrPuntoDesconocido y fallo con %v",
				inventado, err)
		}
	}
	if despues := instantanea(t, raiz); despues != antes {
		t.Errorf("deshacer un punto inventado ha modificado la instalacion:\nantes:   %s\ndespues: %s",
			antes, despues)
	}
}

// Un punto de retorno corrupto NO se restaura. Restaurar desde una copia que no
// cuadra con su manifiesto seria escribir bytes desconocidos encima de una
// instalacion que al menos se sabia rota de una forma concreta.
func TestDeshacerUnPuntoCorruptoFallaEnVezDeRestaurarBasura(t *testing.T) {
	casos := map[string]func(t *testing.T, raiz, punto string){
		"la copia de un fichero cambiada": func(t *testing.T, raiz, punto string) {
			ruta := filepath.Join(raiz, DirInterno, nombrePuntos, punto, nombreCopia, "dutiq")
			if err := os.WriteFile(ruta, []byte("binario que no es el que se guardo"), 0o600); err != nil {
				t.Fatal(err)
			}
		},
		"la copia de un fichero truncada": func(t *testing.T, raiz, punto string) {
			ruta := filepath.Join(raiz, DirInterno, nombrePuntos, punto, nombreCopia, "dutiq")
			if err := os.WriteFile(ruta, nil, 0o600); err != nil {
				t.Fatal(err)
			}
		},
		"la copia de un fichero borrada": func(t *testing.T, raiz, punto string) {
			ruta := filepath.Join(raiz, DirInterno, nombrePuntos, punto, nombreCopia, "dutiq")
			if err := os.Remove(ruta); err != nil {
				t.Fatal(err)
			}
		},
		"el manifiesto renombrado a otro punto": func(t *testing.T, raiz, punto string) {
			ruta := filepath.Join(raiz, DirInterno, nombrePuntos, punto, nombreManif)
			b, err := os.ReadFile(ruta)
			if err != nil {
				t.Fatal(err)
			}
			s := strings.Replace(string(b), punto, "20200101T000000Z-1111111111111111", 1)
			if err := os.WriteFile(ruta, []byte(s), 0o600); err != nil {
				t.Fatal(err)
			}
		},
	}
	for nombre, romper := range casos {
		t.Run(nombre, func(t *testing.T) {
			a, raiz := nuevoActualizador(t)
			ctx := context.Background()
			if _, err := a.Aplicar(ctx, versionVieja); err != nil {
				t.Fatal(err)
			}
			punto, err := a.Aplicar(ctx, versionNueva)
			if err != nil {
				t.Fatal(err)
			}
			romper(t, raiz, punto)
			antes := instantanea(t, raiz)

			err = a.Deshacer(ctx, punto)
			if err == nil {
				t.Fatal("deshacer desde un punto corrupto devolvio exito")
			}
			if !errors.Is(err, ErrPuntoCorrupto) {
				t.Fatalf("tenia que fallar con ErrPuntoCorrupto y fallo con %v", err)
			}
			if despues := instantanea(t, raiz); despues != antes {
				t.Errorf("no se ha restaurado nada, se dijo, y la instalacion ha cambiado:\n"+
					"antes:   %s\ndespues: %s", antes, despues)
			}
		})
	}
}

// El canal es un tercero: lo que trae puede no ser lo que dice que es.
func TestUnaVersionQueNoEsLaQueDiceSerNoSeInstala(t *testing.T) {
	a, raiz := nuevoActualizador(t)
	canal := a.canal.(CanalDirectorio).Dir

	// Se sustituye el binario de la version nueva por otra cosa, dejando el
	// catalogo (y por tanto el digest declarado) intacto. Es exactamente lo que
	// pasa si alguien mete mano por el camino o si el espejo esta corrupto.
	ruta := filepath.Join(canal, versionNueva, "dutiq")
	if err := os.WriteFile(ruta, []byte("esto no es el binario que dice el catalogo"), 0o600); err != nil {
		t.Fatal(err)
	}
	antes := instantanea(t, raiz)

	_, err := a.Aplicar(context.Background(), versionNueva)
	if err == nil {
		t.Fatal("se instalo una version cuyo contenido no casa con su digest")
	}
	if !errors.Is(err, ErrDigest) {
		t.Fatalf("tenia que fallar con ErrDigest y fallo con %v", err)
	}
	if despues := instantanea(t, raiz); despues != antes {
		t.Errorf("la instalacion se ha tocado pese a rechazar la version:\nantes:   %s\ndespues: %s",
			antes, despues)
	}
	// Y no se deja basura: ni punto de retorno, ni marcador, ni cerrojo.
	puntos, err := a.Puntos()
	if err != nil {
		t.Fatal(err)
	}
	if len(puntos) != 0 {
		t.Errorf("se creo un punto de retorno para una version que no se llego a instalar: %v", puntos)
	}
}

// Un canal hostil no puede escribir fuera de la instalacion ni dentro de donde
// viven los puntos de retorno.
func TestUnaVersionQueApuntaFueraDeLaInstalacionSeRechaza(t *testing.T) {
	for _, ruta := range []string{
		"../fuera.txt",
		"../../.ssh/authorized_keys",
		"/etc/passwd",
		DirInterno + "/puntos/falso/manifiesto.json",
		"./raro.txt",
		"a//b.txt",
		"sub\\windows.txt",
		"C:/x.txt",
	} {
		t.Run(ruta, func(t *testing.T) {
			raiz := t.TempDir()
			canal := t.TempDir()
			contenido := []byte("carga")
			if err := EscribirCatalogo(canal, []Version{{
				Version: versionNueva, Ficheros: map[string]string{ruta: digest(contenido)},
			}}); err != nil {
				t.Fatal(err)
			}
			a := Nuevo(Opciones{Raiz: raiz, Canal: CanalDirectorio{Dir: canal}})
			if _, err := a.Aplicar(context.Background(), versionNueva); err == nil {
				t.Fatalf("se acepto la ruta %q, que sale de la instalacion o entra en %s",
					ruta, DirInterno)
			}
		})
	}
}

// Si la instalacion falla a mitad, se vuelve atras SOLA y se dice.
func TestSiLaInstalacionFallaAMitadSeVuelveAtrasSola(t *testing.T) {
	a, raiz := nuevoActualizador(t)
	ctx := context.Background()
	if _, err := a.Aplicar(ctx, versionVieja); err != nil {
		t.Fatal(err)
	}
	esperado := instantanea(t, raiz)

	// Averia la escritura de la INSTALACION a partir del segundo fichero. La
	// del punto de retorno y la de la restauracion no pasan por aqui, que es lo
	// que permite comprobar que la vuelta atras sigue funcionando con la
	// instalacion rota.
	original := escribirEnLaInstalacion
	escritos := 0
	escribirEnLaInstalacion = func(destino string, datos []byte) error {
		escritos++
		if escritos > 1 {
			return fmt.Errorf("no queda espacio en el dispositivo (simulado)")
		}
		return original(destino, datos)
	}
	defer func() { escribirEnLaInstalacion = original }()

	_, err := a.Aplicar(ctx, versionNueva)
	if err == nil {
		t.Fatal("la instalacion fallo a mitad y Aplicar devolvio exito")
	}
	if !strings.Contains(err.Error(), "vuelto atras") {
		t.Errorf("el error no dice que se ha vuelto atras, asi que el operador no sabe en que "+
			"estado esta su instalacion: %v", err)
	}
	if got := instantanea(t, raiz); got != esperado {
		t.Errorf("la instalacion no volvio al estado de antes:\nesperado: %s\nobtenido: %s",
			esperado, got)
	}
	if _, hay, err := a.Interrumpida(); err != nil || hay {
		t.Errorf("tras volver atras sola no puede quedar una actualizacion a medias (hay=%v, err=%v)",
			hay, err)
	}
}

// Un punto de retorno que sale truncado (disco lleno, cuota agotada) se caza
// ANTES de tocar la instalacion. Es la mitad silenciosa del problema: sin esta
// comprobacion el punto pasaria por bueno hasta el dia en que hiciera falta.
func TestUnPuntoDeRetornoTruncadoAbortaLaActualizacion(t *testing.T) {
	a, raiz := nuevoActualizador(t)
	ctx := context.Background()
	if _, err := a.Aplicar(ctx, versionVieja); err != nil {
		t.Fatal(err)
	}
	esperado := instantanea(t, raiz)

	original := escribirEnElPunto
	escribirEnElPunto = func(destino string, datos []byte) error {
		// La copia se queda a medias, como con el disco lleno, y ademas SIN
		// error: es el caso que de verdad enganna, porque un error si se
		// notaria.
		if len(datos) > 4 {
			datos = datos[:4]
		}
		return original(destino, datos)
	}
	defer func() { escribirEnElPunto = original }()

	_, err := a.Aplicar(ctx, versionNueva)
	if err == nil {
		t.Fatal("se actualizo con un punto de retorno truncado: la vuelta atras seria imposible " +
			"y nadie se enteraria hasta que hiciera falta")
	}
	if !strings.Contains(err.Error(), "punto de retorno") {
		t.Errorf("el error no dice que el problema es el punto de retorno: %v", err)
	}
	if got := instantanea(t, raiz); got != esperado {
		t.Errorf("la instalacion se toco pese a abortar:\nesperado: %s\nobtenido: %s", esperado, got)
	}
	puntos, err := a.Puntos()
	if err != nil {
		t.Fatal(err)
	}
	if len(puntos) != 1 {
		t.Errorf("tenia que quedar solo el punto de la primera actualizacion y quedan %d: "+
			"el punto truncado no se ha limpiado", len(puntos))
	}
}

// Dos `dutiq update` a la vez no se pisan. El cerrojo esta en el sistema de
// ficheros porque los dos procesos pueden ser dos terminales distintas.
func TestDosActualizacionesALaVezNoSePisan(t *testing.T) {
	a, raiz := nuevoActualizador(t)
	const n = 8

	var mu sync.Mutex
	var exitos int
	var otros []error

	var arranque sync.WaitGroup
	arranque.Add(1)
	var fin sync.WaitGroup
	for i := 0; i < n; i++ {
		fin.Add(1)
		go func() {
			defer fin.Done()
			arranque.Wait()
			_, err := a.Aplicar(context.Background(), versionNueva)
			mu.Lock()
			defer mu.Unlock()
			switch {
			case err == nil:
				exitos++
			case errors.Is(err, ErrOcupado):
			default:
				otros = append(otros, err)
			}
		}()
	}
	arranque.Done()
	fin.Wait()

	if exitos == 0 {
		t.Fatalf("ninguna de las %d actualizaciones simultaneas salio adelante: el cerrojo "+
			"bloquea en vez de serializar. Errores: %v", n, otros)
	}
	for _, err := range otros {
		t.Errorf("una actualizacion simultanea fallo por algo que no es el cerrojo: %v", err)
	}
	// Y lo que importa: la instalacion queda coherente, no entrelazada.
	for rel, quiero := range ficherosDe(versionNueva) {
		b, err := os.ReadFile(filepath.Join(raiz, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("%s no quedo instalado: %v", rel, err)
		}
		if string(b) != string(quiero) {
			t.Errorf("%s quedo con %q: las escrituras se entrelazaron", rel, b)
		}
	}
	if v, _ := a.VersionInstalada(); v != versionNueva {
		t.Errorf("VERSION quedo en %q", v)
	}
}

func TestConElCerrojoTomadoAplicarNoEntra(t *testing.T) {
	a, raiz := nuevoActualizador(t)
	if err := os.MkdirAll(filepath.Join(raiz, DirInterno), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(raiz, DirInterno, nombreCerrojo), []byte("pid 1"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := a.Aplicar(context.Background(), versionNueva)
	if !errors.Is(err, ErrOcupado) {
		t.Fatalf("con el cerrojo puesto, aplicar tenia que dar ErrOcupado y dio %v", err)
	}
	if !strings.Contains(err.Error(), nombreCerrojo) {
		t.Errorf("el error no dice donde esta el cerrojo, asi que un cerrojo huerfano deja al "+
			"operador atascado sin saber que borrar: %v", err)
	}
}

// Una actualizacion interrumpida (el proceso muere entre el marcador y el final)
// se detecta y se repara, y hasta que se repare no se deja aplicar otra encima.
func TestUnaActualizacionInterrumpidaSeDetectaYSeRepara(t *testing.T) {
	a, raiz := nuevoActualizador(t)
	ctx := context.Background()
	if _, err := a.Aplicar(ctx, versionVieja); err != nil {
		t.Fatal(err)
	}
	esperado := instantanea(t, raiz)

	// Se simula la muerte a mitad: se aplica de verdad y despues se vuelve a
	// poner el marcador que el proceso muerto no llego a borrar.
	punto, err := a.Aplicar(ctx, versionNueva)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.escribirMarcador(marcador{Punto: punto, Version: versionNueva,
		Inicio: "2026-08-25T10:00:00Z", PID: 4321}); err != nil {
		t.Fatal(err)
	}

	m, hay, err := a.Interrumpida()
	if err != nil || !hay {
		t.Fatalf("no se detecta la actualizacion a medias (hay=%v, err=%v)", hay, err)
	}
	if m.Hacia != versionNueva || m.Desde != versionVieja {
		t.Errorf("la actualizacion a medias se describe mal: de %q a %q", m.Desde, m.Hacia)
	}

	// Aplicar otra encima enterraria el unico punto bueno: se niega.
	if _, err := a.Aplicar(ctx, versionVieja); !errors.Is(err, ErrAMedias) {
		t.Fatalf("con una actualizacion a medias, aplicar otra tenia que dar ErrAMedias y dio %v", err)
	}

	reparado, err := a.Reparar(ctx)
	if err != nil {
		t.Fatalf("reparar: %v", err)
	}
	if reparado != punto {
		t.Errorf("reparar volvio al punto %q y la actualizacion a medias era la del punto %q",
			reparado, punto)
	}
	if got := instantanea(t, raiz); got != esperado {
		t.Errorf("reparar no dejo la instalacion como antes:\nesperado: %s\nobtenido: %s", esperado, got)
	}
	if _, hay, _ := a.Interrumpida(); hay {
		t.Error("tras reparar sigue habiendo una actualizacion a medias")
	}
	// Y ahora si deja actualizar.
	if _, err := a.Aplicar(ctx, versionNueva); err != nil {
		t.Errorf("tras reparar, aplicar tenia que funcionar y dio %v", err)
	}
}

func TestDisponibleCallaCuandoYaEstaLaUltima(t *testing.T) {
	a, _ := nuevoActualizador(t)
	ctx := context.Background()

	v, notas, err := a.Disponible(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if v != versionNueva {
		t.Fatalf("con la instalacion vacia, disponible tenia que ofrecer %q y ofrecio %q",
			versionNueva, v)
	}
	if notas == "" {
		t.Error("una version disponible sin notas obliga a actualizar a ciegas")
	}
	if _, err := a.Aplicar(ctx, versionNueva); err != nil {
		t.Fatal(err)
	}
	v, _, err = a.Disponible(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if v != "" {
		t.Errorf("con la ultima ya instalada, disponible tenia que devolver vacio y devolvio %q", v)
	}
}

// Un error accionable dice QUE hay, no solo que lo que pedias no esta.
func TestUnaVersionInexistenteDiceCualesHay(t *testing.T) {
	a, _ := nuevoActualizador(t)
	_, err := a.Aplicar(context.Background(), "v99.99.99")
	if err == nil {
		t.Fatal("aplicar una version inexistente tiene que fallar")
	}
	if !errors.Is(err, ErrVersionDesconocida) {
		t.Fatalf("tenia que ser ErrVersionDesconocida y fue %v", err)
	}
	for _, v := range []string{versionVieja, versionNueva} {
		if !strings.Contains(err.Error(), v) {
			t.Errorf("el error no dice que %s esta disponible, asi que el operador tiene que "+
				"adivinar: %v", v, err)
		}
	}
}

func TestSinCanalConfiguradoSeDiceQueFalta(t *testing.T) {
	a := Nuevo(Opciones{Raiz: t.TempDir()})
	if _, _, err := a.Disponible(context.Background()); err == nil {
		t.Error("sin canal, consultar disponibles tiene que fallar diciendolo")
	}
	if _, err := a.Aplicar(context.Background(), versionNueva); err == nil {
		t.Error("sin canal, aplicar tiene que fallar diciendolo")
	}
}

// instantanea describe el arbol de la instalacion IGNORANDO el directorio
// interno, que cambia legitimamente en cada operacion. Sirve para comprobar por
// identidad que una operacion no ha tocado nada, en vez de comprobar fichero a
// fichero y dejarse el que importa.
func instantanea(t *testing.T, raiz string) string {
	t.Helper()
	var b strings.Builder
	err := filepath.WalkDir(raiz, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(raiz, p)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel == DirInterno {
			return filepath.SkipDir
		}
		if d.IsDir() || rel == "." {
			return nil
		}
		datos, err := os.ReadFile(p) // #nosec G304 -- arbol temporal del test
		if err != nil {
			return err
		}
		fmt.Fprintf(&b, "%s=%s\n", rel, digest(datos))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return b.String()
}
