package latido

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"plazum/nucleo/pantalla"
)

var ahora = time.Date(2026, 8, 26, 9, 0, 0, 0, time.UTC)

// secretosDePrueba es una fuente determinista. Vive aqui, en un _test.go, y no
// hay ninguna variable de paquete que la deje puesta en produccion: es la razon
// por la que puertos.Secretos existe.
type secretosDePrueba struct{ n int }

func (s *secretosDePrueba) Token(nb int) (string, error) {
	s.n++
	return fmt.Sprintf("%0*x", nb*2, s.n), nil
}

func (s *secretosDePrueba) Bytes(b []byte) error {
	for i := range b {
		b[i] = byte(s.n)
	}
	return nil
}

// ---------------------------------------------------------------------------
// El opt-in
// ---------------------------------------------------------------------------

// LA PUERTA DEL OPT-IN, y es un test porque un producto autoalojado cuya tesis
// es que el receptor no se fia del emisor no puede mandar telemetria sin
// permiso. Se comprueba donde de verdad importa: en la funcion que manda
// paquetes, no en la interfaz que la llama.
func TestElLatidoEstaApagadoPorDefectoYNoSaleNada(t *testing.T) {
	dir := t.TempDir()

	// Un directorio de datos recien hecho: ni fichero, ni error, ni aviso.
	e, err := Cargar(dir)
	if err != nil {
		t.Fatalf("cargar un directorio sin fichero de latido da error: %v. Una instalacion "+
			"recien hecha no tiene telemetria, y eso no es un problema que resolver", err)
	}
	if e.Activado {
		t.Fatal("EL LATIDO SALE ENCENDIDO POR DEFECTO. Es la puerta entera de esta pieza: " +
			"el valor por defecto de una estructura Estado en cero tiene que ser apagado")
	}
	if e.Instancia != "" {
		t.Errorf("hay identificador de instalacion (%q) sin que nadie haya activado nada",
			e.Instancia)
	}
	if e.Consentimiento != nil {
		t.Error("hay consentimiento registrado sin que nadie lo haya dado")
	}

	// Y con el latido apagado, pulsar NO TOCA EL CANAL.
	c := &CanalMemoria{}
	if _, err := Pulsar(context.Background(), e, c, ahora); !errors.Is(err, ErrApagado) {
		t.Errorf("pulsar con el latido apagado devuelve %v y esperaba ErrApagado", err)
	}
	if c.N() != 0 {
		t.Fatalf("con el latido APAGADO han salido %d entregas por el canal. Esto es lo "+
			"unico que esta pieza no se puede permitir", c.N())
	}

	// Y un ciclo del planificador con el latido apagado tampoco manda nada,
	// pero SI apunta el ciclo: el aviso de las 24 horas funciona sin
	// telemetria, que es la mitad que de verdad se vende.
	e, err = Ciclo(context.Background(), dir, c, ahora)
	if err != nil {
		t.Fatalf("un ciclo con el latido apagado da error: %v", err)
	}
	if c.N() != 0 {
		t.Fatalf("un ciclo con el latido apagado ha mandado %d pulsos", c.N())
	}
	if !e.UltimoCiclo.Equal(ahora) {
		t.Errorf("el ciclo no ha quedado apuntado (%v). Sin esa marca, el aviso de las 24 "+
			"horas no existe para quien no activa la telemetria, que son casi todos",
			e.UltimoCiclo)
	}
}

// Activar es la unica forma de encenderlo, y deja el consentimiento ESCRITO,
// con el texto que se acepto y su fecha. Un booleano a true no es un
// consentimiento: dentro de un ano nadie sabria a que se dijo que si.
func TestActivarRegistraElConsentimientoConSuTextoYSuFecha(t *testing.T) {
	dir := t.TempDir()
	e, err := Activar(dir, "", ahora, &secretosDePrueba{})
	if err != nil {
		t.Fatal(err)
	}
	if !e.Activado {
		t.Fatal("activar no ha activado")
	}
	if e.Consentimiento == nil {
		t.Fatal("activar no ha registrado consentimiento")
	}
	if e.Consentimiento.Texto != QueSeManda {
		t.Errorf("el consentimiento guarda %q y lo que se acepto es %q",
			e.Consentimiento.Texto, QueSeManda)
	}
	if !e.Consentimiento.Otorgado.Equal(ahora) {
		t.Errorf("el consentimiento se fecha en %v y se dio en %v",
			e.Consentimiento.Otorgado, ahora)
	}
	if len(e.Instancia) != BytesDeInstancia*2 {
		t.Errorf("el identificador de instalacion mide %d y esperaba %d caracteres hex",
			len(e.Instancia), BytesDeInstancia*2)
	}

	// Y esta en el disco, no solo en memoria.
	leido, err := Cargar(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !leido.Activado || leido.Instancia != e.Instancia {
		t.Errorf("lo guardado no cuadra con lo devuelto: %+v", leido)
	}
	b, err := os.ReadFile(filepath.Join(dir, NombreDelFichero))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "identificador aleatorio") {
		t.Error("el fichero de estado no lleva dentro el texto que se acepto, asi que el " +
			"consentimiento no se puede releer un ano despues")
	}
}

// Activar dos veces no cambia el identificador: seria perder el hilo del pulso
// por teclear el mismo comando dos veces.
func TestActivarDosVecesNoRegeneraElIdentificador(t *testing.T) {
	dir := t.TempDir()
	s := &secretosDePrueba{}
	uno, err := Activar(dir, "", ahora, s)
	if err != nil {
		t.Fatal(err)
	}
	dos, err := Activar(dir, "", ahora.Add(time.Hour), s)
	if err != nil {
		t.Fatal(err)
	}
	if uno.Instancia != dos.Instancia {
		t.Errorf("el identificador ha cambiado al reactivar: %q -> %q", uno.Instancia, dos.Instancia)
	}
}

// Desactivar BORRA el identificador y el consentimiento, y conserva las marcas
// del planificador.
//
// Lo primero no es limpieza: si se conservara, quien recibe los pulsos podria
// enlazar el antes y el despues de una baja. Lo segundo es que el aviso de las
// 24 horas tiene que seguir funcionando con la telemetria apagada.
func TestDesactivarBorraElIdentificadorYConservaLasMarcasDelPlanificador(t *testing.T) {
	dir := t.TempDir()
	if _, err := Activar(dir, "", ahora, &secretosDePrueba{}); err != nil {
		t.Fatal(err)
	}
	c := &CanalMemoria{}
	if _, err := Ciclo(context.Background(), dir, c, ahora); err != nil {
		t.Fatal(err)
	}

	e, err := Desactivar(dir)
	if err != nil {
		t.Fatal(err)
	}
	if e.Activado {
		t.Error("desactivar no ha desactivado")
	}
	if e.Instancia != "" {
		t.Errorf("el identificador sobrevive a la baja (%q): quien recibe los pulsos podria "+
			"enlazar el antes y el despues", e.Instancia)
	}
	if e.Consentimiento != nil {
		t.Error("el consentimiento sobrevive a la baja")
	}
	if !e.UltimoPulso.IsZero() {
		t.Error("la marca del ultimo pulso sobrevive a la baja")
	}
	if !e.UltimoCiclo.Equal(ahora) {
		t.Errorf("desactivar la telemetria se ha llevado por delante la marca del "+
			"planificador (%v). El aviso de las 24 horas es de el, no nuestro", e.UltimoCiclo)
	}

	// Y reactivar da un identificador NUEVO.
	otra, err := Activar(dir, "", ahora.Add(24*time.Hour), &secretosDePrueba{n: 40})
	if err != nil {
		t.Fatal(err)
	}
	if otra.Instancia == "" {
		t.Fatal("reactivar no ha generado identificador")
	}
	if otra.Instancia == "00000000000000000000000000000001" {
		t.Error("reactivar ha reusado el identificador de la primera vez")
	}
}

// Un estado activado a mano, sin consentimiento escrito, NO carga.
//
// Es la unica forma de que "el producto lo mando solo" no pueda ser cierto:
// encender el opt-in editando un booleano en un fichero deja de funcionar.
func TestUnLatidoEncendidoAManoSinConsentimientoNoCarga(t *testing.T) {
	dir := t.TempDir()
	crudo := `{"activado": true, "instancia": "abc"}`
	if err := os.WriteFile(filepath.Join(dir, NombreDelFichero), []byte(crudo), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Cargar(dir); !errors.Is(err, ErrSinConsentimiento) {
		t.Fatalf("un estado activado a mano y sin consentimiento carga sin protestar: %v", err)
	}

	// Control: con el consentimiento escrito, carga.
	e := Estado{Activado: true, Instancia: "abc",
		Consentimiento: &Consentimiento{Otorgado: ahora, Texto: QueSeManda}}
	if err := Guardar(dir, e); err != nil {
		t.Fatal(err)
	}
	if _, err := Cargar(dir); err != nil {
		t.Fatalf("un estado con consentimiento escrito no carga: %v", err)
	}
}

// ---------------------------------------------------------------------------
// La direccion del aviso
// ---------------------------------------------------------------------------

// La propiedad de Ciclo: la marca del planificador se escribe TAMBIEN cuando el
// pulso falla.
//
// Si un fallo de red hacia nosotros dejara sin escribir la marca, nuestra caida
// se convertiria en el aviso "tu planificador lleva 24 horas muerto". Eso es
// exactamente la mentira que esta pieza existe para no contar.
func TestUnCanalCaidoNoBorraLaMarcaDelPlanificador(t *testing.T) {
	dir := t.TempDir()
	if _, err := Activar(dir, "", ahora, &secretosDePrueba{}); err != nil {
		t.Fatal(err)
	}
	roto := &CanalMemoria{Fallo: fmt.Errorf("%w: la red no existe", ErrCanal)}

	e, err := Ciclo(context.Background(), dir, roto, ahora)
	if !errors.Is(err, ErrCanal) {
		t.Errorf("el fallo del canal no se informa: %v", err)
	}
	if !e.UltimoCiclo.Equal(ahora) {
		t.Fatalf("el pulso ha fallado y la marca del ciclo no se ha escrito (%v). Nuestra "+
			"caida se leeria como su planificador muerto", e.UltimoCiclo)
	}
	if !e.FalloElUltimoIntento {
		t.Error("el fallo del canal no ha quedado apuntado, asi que el smoke test no se ve")
	}

	// Y en el disco, que es lo que leera el vigilante manana.
	leido, err := Cargar(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !leido.UltimoCiclo.Equal(ahora) {
		t.Errorf("la marca del ciclo no ha llegado al disco: %+v", leido)
	}

	// Y lo que el vigilante concluye: planificador CORRECTO, canal en aviso.
	v := pantalla.Vigilar(leido.Marcas(), ahora.Add(time.Minute))
	if v.Nivel != pantalla.NivelCorrecto {
		t.Errorf("con el canal caido y el ciclo recien corrido, el veredicto del "+
			"planificador es %q", v.Nivel)
	}
	if v.Canal.Nivel != pantalla.NivelAviso {
		t.Errorf("con el canal caido, el canal sale %q", v.Canal.Nivel)
	}
}

// El smoke test del canal: prueba el canal de verdad y deja el resultado
// escrito, sin tocar la marca del ciclo.
//
// No tocar la marca del ciclo importa: probar el canal no es haber corrido un
// ciclo, y confundirlos dejaria al operador tranquilo con el planificador
// parado, que es el fallo que esta pieza persigue.
func TestElSmokeTestDelCanalNoSeHacePasarPorUnCicloDelPlanificador(t *testing.T) {
	dir := t.TempDir()
	if _, err := Activar(dir, "", ahora, &secretosDePrueba{}); err != nil {
		t.Fatal(err)
	}
	c := &CanalMemoria{}

	e, err := Probar(context.Background(), dir, c, ahora)
	if err != nil {
		t.Fatalf("el smoke test con un canal que funciona da error: %v", err)
	}
	if c.N() != 1 {
		t.Fatalf("el smoke test ha hecho %d entregas y tenia que hacer una", c.N())
	}
	if !e.UltimoPulso.Equal(ahora) {
		t.Errorf("el smoke test no ha apuntado el pulso: %v", e.UltimoPulso)
	}
	if !e.UltimoCiclo.IsZero() {
		t.Errorf("el smoke test ha apuntado un ciclo del planificador (%v) que nadie ha "+
			"corrido: el operador se quedaria tranquilo con el planificador parado",
			e.UltimoCiclo)
	}

	// Con el canal roto, el fallo queda escrito y se ve sin esperar 24 horas.
	c.Fallo = fmt.Errorf("%w: el receptor contesta 500", ErrCanal)
	e, err = Probar(context.Background(), dir, c, ahora.Add(time.Minute))
	if !errors.Is(err, ErrCanal) {
		t.Errorf("el smoke test contra un canal roto devuelve %v", err)
	}
	if !e.FalloElUltimoIntento {
		t.Error("el smoke test no ha apuntado el fallo")
	}
	v := pantalla.Vigilar(e.Marcas(), ahora.Add(2*time.Minute))
	if v.Canal.Clave != pantalla.ClaveLatidoFallo {
		t.Errorf("tras un smoke test fallido la pantalla diria %q", v.Canal.Clave)
	}
}

// Probar con el latido apagado no escribe nada. El estado en disco no puede
// cambiar porque alguien haya preguntado.
func TestProbarConElLatidoApagadoNoEscribeNada(t *testing.T) {
	dir := t.TempDir()
	c := &CanalMemoria{}
	if _, err := Probar(context.Background(), dir, c, ahora); !errors.Is(err, ErrApagado) {
		t.Errorf("probar con el latido apagado devuelve %v", err)
	}
	if c.N() != 0 {
		t.Errorf("probar con el latido apagado ha mandado %d pulsos", c.N())
	}
	if _, err := os.Stat(filepath.Join(dir, NombreDelFichero)); !errors.Is(err, os.ErrNotExist) {
		t.Error("probar con el latido apagado ha creado el fichero de estado")
	}
}

// ---------------------------------------------------------------------------
// El destino
// ---------------------------------------------------------------------------

// El destino se comprueba, y cada rechazo tiene su razon. La de la parte de
// consulta es la que importa de verdad: es donde se cuela un identificador sin
// que nadie lo note.
func TestElDestinoRechazaLoQueFiltrariaAlgo(t *testing.T) {
	casos := []struct {
		nombre  string
		destino string
		vale    bool
	}{
		{"el de por defecto", DestinoPorDefecto, true},
		{"un receptor propio por https", "https://monitor.interno.example/plazum", true},
		{"localhost por http, para probar", "http://localhost:8080/latido", true},
		{"http a un tercero", "http://plazum.dev/latido", false},
		{"con parte de consulta", "https://plazum.dev/latido?org=acme", false},
		{"con usuario y contrasena", "https://u:p@plazum.dev/latido", false},
		{"con fragmento", "https://plazum.dev/latido#acme", false},
		{"sin maquina", "https:///latido", false},
		{"otro esquema", "ftp://plazum.dev/latido", false},
		{"vacio", "", false},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			err := ComprobarDestino(c.destino)
			if c.vale && err != nil {
				t.Errorf("%q tenia que valer y da %v", c.destino, err)
			}
			if !c.vale {
				if err == nil {
					t.Fatalf("%q se acepta como destino de pulso", c.destino)
				}
				if !errors.Is(err, ErrDestinoInseguro) {
					t.Errorf("%q se rechaza sin el centinela: %v", c.destino, err)
				}
				if !strings.Contains(err.Error(), "Arreglo:") {
					t.Errorf("el rechazo de %q no dice como se arregla: %v", c.destino, err)
				}
			}
		})
	}
}

// Y no se puede activar contra un destino que filtraria algo: el rechazo llega
// al activar, no al primer pulso de madrugada.
func TestNoSePuedeActivarContraUnDestinoQueFiltra(t *testing.T) {
	dir := t.TempDir()
	if _, err := Activar(dir, "https://plazum.dev/latido?org=acme", ahora, &secretosDePrueba{}); !errors.Is(err, ErrDestinoInseguro) {
		t.Fatalf("activar contra un destino con parte de consulta devuelve %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, NombreDelFichero)); !errors.Is(err, os.ErrNotExist) {
		t.Error("activar ha fallado y aun asi ha dejado estado escrito")
	}
}

// El estado ilegible se dice con su arreglo, y no se toma por "apagado".
//
// Tomarlo por apagado seria comodo y seria mentira: si el fichero dice algo que
// no entendemos, puede estar diciendo "activado", y seguir mandando pulsos con
// un estado que no se sabe leer es lo contrario del opt-in.
func TestUnEstadoIlegibleSeDiceConSuArreglo(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, NombreDelFichero), []byte("{roto"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := Cargar(dir)
	if !errors.Is(err, ErrEstadoIlegible) {
		t.Fatalf("un estado ilegible devuelve %v", err)
	}
	if !strings.Contains(err.Error(), "Arreglo:") {
		t.Errorf("el error no dice como se arregla: %v", err)
	}
}

// Las marcas que se le pasan al nucleo son las del estado, sin decidir nada por
// el camino. La regla de las 24 horas vive en un solo sitio.
func TestLasMarcasViajanAlNucleoSinInterpretar(t *testing.T) {
	e := Estado{
		Activado: true, UltimoCiclo: ahora.Add(-time.Hour),
		UltimoPulso: ahora.Add(-2 * time.Hour), FalloElUltimoIntento: true,
	}
	m := e.Marcas()
	if !m.UltimoCiclo.Equal(e.UltimoCiclo) || !m.UltimoPulso.Equal(e.UltimoPulso) ||
		!m.LatidoActivado || !m.FalloElUltimoIntento {
		t.Errorf("las marcas no cuadran con el estado: %+v contra %+v", m, e)
	}
}

// El fichero de estado se escribe con permisos restrictivos. No lleva secretos
// dentro, pero tampoco tiene por que leerlo el resto de la maquina.
func TestElFicheroDeEstadoNoEsLegiblePorTodoElMundo(t *testing.T) {
	if os.Getenv("OS") == "Windows_NT" {
		t.Skip("los permisos POSIX no significan lo mismo en Windows")
	}
	dir := t.TempDir()
	if _, err := Activar(dir, "", ahora, &secretosDePrueba{}); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(filepath.Join(dir, NombreDelFichero))
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm()&0o077 != 0 {
		t.Errorf("el fichero de estado tiene permisos %v", fi.Mode().Perm())
	}
}

// El estado se serializa a JSON estable: lo que se guarda se vuelve a leer
// igual. Un fichero que no da la vuelta es un fichero que un dia pierde el
// consentimiento y deja el latido encendido sin permiso escrito.
func TestElEstadoDaLaVuelta(t *testing.T) {
	e := Estado{Activado: true, Instancia: "abc", Destino: "https://ejemplo.invalid/l",
		Consentimiento: &Consentimiento{Otorgado: ahora, Texto: QueSeManda},
		UltimoCiclo:    ahora, UltimoPulso: ahora}
	b, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	var v Estado
	if err := json.Unmarshal(b, &v); err != nil {
		t.Fatal(err)
	}
	if v.Instancia != e.Instancia || v.Destino != e.Destino ||
		v.Consentimiento == nil || v.Consentimiento.Texto != QueSeManda ||
		!v.UltimoCiclo.Equal(e.UltimoCiclo) {
		t.Errorf("el estado no da la vuelta: %+v", v)
	}
}

// EL PULSO ES DIARIO, aunque el ciclo del planificador corra cada hora.
//
// Son dos cosas distintas con dos ritmos distintos, y confundirlas es mandar
// veinticuatro veces mas datos de los que la declaracion dice. El ciclo es del
// operador y cuanto mas a menudo mejor; el pulso sale de su maquina y cuanto
// menos, mejor.
func TestElPulsoEsDiarioAunqueElCicloCorraCadaHora(t *testing.T) {
	dir := t.TempDir()
	if _, err := Activar(dir, "", ahora, &secretosDePrueba{}); err != nil {
		t.Fatal(err)
	}
	c := &CanalMemoria{}

	// Veinticuatro ciclos, uno por hora. El primero pulsa.
	for i := 0; i < 24; i++ {
		if _, err := Ciclo(context.Background(), dir, c, ahora.Add(time.Duration(i)*time.Hour)); err != nil {
			t.Fatal(err)
		}
	}
	if c.N() != 1 {
		t.Fatalf("veinticuatro ciclos en veintitres horas han mandado %d pulsos y tenia "+
			"que salir uno. La casilla dice pulso DIARIO", c.N())
	}

	// A las 24 horas del ultimo pulso aceptado, sale el segundo.
	if _, err := Ciclo(context.Background(), dir, c, ahora.Add(24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if c.N() != 2 {
		t.Errorf("pasado un dia entero, el pulso no ha vuelto a salir (%d entregas): "+
			"entonces no es diario, es uno y ya", c.N())
	}

	// Y la marca del ciclo se escribe en TODOS, pulsen o no: es la que
	// sostiene el aviso de las 24 horas.
	e, err := Cargar(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !e.UltimoCiclo.Equal(ahora.Add(24 * time.Hour)) {
		t.Errorf("la marca del ultimo ciclo es %v y el ultimo ciclo fue a las 24 h",
			e.UltimoCiclo)
	}
}

// Un pulso que no llega se reintenta al ciclo siguiente, sin esperar un dia.
//
// Sale de comparar contra el ultimo pulso ACEPTADO y no contra el ultimo
// intento: mientras el canal este roto se reintenta cada ciclo, y en cuanto
// funciona se vuelve al ritmo diario.
func TestUnPulsoQueFallaSeReintentaAlSiguienteCiclo(t *testing.T) {
	dir := t.TempDir()
	if _, err := Activar(dir, "", ahora, &secretosDePrueba{}); err != nil {
		t.Fatal(err)
	}
	roto := &CanalMemoria{Fallo: fmt.Errorf("%w: sin red", ErrCanal)}
	for i := 0; i < 3; i++ {
		if _, err := Ciclo(context.Background(), dir, roto, ahora.Add(time.Duration(i)*time.Hour)); !errors.Is(err, ErrCanal) {
			t.Fatalf("el ciclo %d no informa del fallo del canal: %v", i, err)
		}
	}
	// El canal vuelve. El siguiente ciclo tiene que pulsar, no esperar un dia.
	roto.Fallo = nil
	if _, err := Ciclo(context.Background(), dir, roto, ahora.Add(3*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if roto.N() != 1 {
		t.Fatalf("tras recuperarse el canal se han entregado %d pulsos y esperaba 1", roto.N())
	}
	e, err := Cargar(dir)
	if err != nil {
		t.Fatal(err)
	}
	if e.FalloElUltimoIntento {
		t.Error("el canal ya entrega y el estado sigue diciendo que el ultimo intento fallo")
	}
}

// Una marca de pulso en el futuro no deja el pulso muerto para siempre.
//
// Es el mismo reloj que miente del veredicto, por el otro lado: si se comparara
// a secas, la resta saldria negativa, nunca alcanzaria el intervalo y la
// instalacion dejaria de pulsar EN SILENCIO hasta que alguien tocara el fichero.
func TestUnaMarcaDePulsoEnElFuturoNoDejaLaInstalacionSinPulsar(t *testing.T) {
	dir := t.TempDir()
	if _, err := Activar(dir, "", ahora, &secretosDePrueba{}); err != nil {
		t.Fatal(err)
	}
	e, err := Cargar(dir)
	if err != nil {
		t.Fatal(err)
	}
	e.UltimoPulso = ahora.Add(30 * 24 * time.Hour)
	if err := Guardar(dir, e); err != nil {
		t.Fatal(err)
	}
	c := &CanalMemoria{}
	if _, err := Ciclo(context.Background(), dir, c, ahora.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if c.N() != 1 {
		t.Errorf("con la marca del ultimo pulso un mes en el futuro se han mandado %d "+
			"pulsos: la instalacion habria dejado de pulsar sin decirlo", c.N())
	}
}

// ---------------------------------------------------------------------------
// Un problema del LATIDO no puede borrar la constancia de que el planificador
// corrio. Hallazgo de la revision hostil.
// ---------------------------------------------------------------------------

// EL ATAQUE: el arreglo que imprime el propio producto borraba la alarma.
//
// Un estado con `activado: true` y sin consentimiento (fichero tocado a mano,
// copia restaurada a medias, plantilla de configuracion repartida a varias
// maquinas) no carga, y el error dice, con estas palabras, "Arreglo: `plazum
// latido desactivar`". Ese arreglo pasa por Cargar, que devolvia Estado{} junto
// al centinela, y Desactivar guardaba ese cero encima del bueno: se llevaba por
// delante UltimoCiclo.
//
// El resultado era el peor posible. Un planificador muerto hace seis dias
// (NivelRoto, `plazum latido` con codigo 1) pasaba a "no ha corrido ningun
// ciclo todavia, si acabas de instalar plazum es lo normal" (NivelAviso, codigo
// 0), o sea que el monitor del operador se ponia VERDE, para siempre, en el
// unico caso que esta pieza existe para cazar. Y el comando lo remataba
// imprimiendo "El aviso de que tu planificador lleva 24 horas callado sigue
// funcionando: no dependia de esto".
//
// Se prueba en las dos ordenes que toleran el centinela, porque las dos
// guardan: desactivar y activar.
func TestUnLatidoRotoNoPuedeBorrarLaMarcaDelPlanificador(t *testing.T) {
	// Seis dias de silencio: el planificador esta muerto y hay que verlo.
	muerto := ahora.Add(-6 * 24 * time.Hour)
	crudo := `{"activado":true,"instancia":"abc",` +
		`"destino":"https://monitor.interno.example/plazum",` +
		`"ultimo_ciclo":"` + muerto.Format(time.RFC3339) + `"}`

	casos := []struct {
		nombre string
		correr func(dir string) error
	}{
		{"desactivar", func(dir string) error { _, err := Desactivar(dir); return err }},
		{"activar", func(dir string) error {
			_, err := Activar(dir, "", ahora, &secretosDePrueba{})
			return err
		}},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, NombreDelFichero),
				[]byte(crudo), 0o600); err != nil {
				t.Fatal(err)
			}
			// Antes: el veredicto es ROTO. Si no lo fuera, el resto del
			// test no estaria midiendo nada.
			antes := Estado{UltimoCiclo: muerto}
			if v := pantalla.Vigilar(antes.Marcas(), ahora); v.Nivel != pantalla.NivelRoto {
				t.Fatalf("el montaje esta mal: con seis dias de silencio el veredicto "+
					"de partida es %q y tenia que ser roto", v.Nivel)
			}

			if err := c.correr(dir); err != nil {
				t.Fatalf("`%s` sobre un estado sin consentimiento falla: %v", c.nombre, err)
			}

			e, err := Cargar(dir)
			if err != nil {
				t.Fatalf("despues de `%s` el estado no carga: %v", c.nombre, err)
			}
			if e.UltimoCiclo.IsZero() {
				t.Fatalf("`%s` ha BORRADO la marca del ultimo ciclo.\n"+
					"  Un planificador muerto hace seis dias vuelve a salir como \"no ha\n"+
					"  corrido ningun ciclo todavia\", que es AVISO y no ROTO, y `plazum\n"+
					"  latido` pasa de codigo 1 a codigo 0: el monitor del operador se\n"+
					"  pone verde justo en el caso que esta pieza existe para cazar.\n"+
					"  Un problema del latido no puede llevarse por delante la constancia\n"+
					"  de que el planificador corrio", c.nombre)
			}
			if !e.UltimoCiclo.Equal(muerto) {
				t.Errorf("`%s` ha movido la marca del ultimo ciclo de %v a %v",
					c.nombre, muerto, e.UltimoCiclo)
			}
			if v := pantalla.Vigilar(e.Marcas(), ahora); v.Nivel != pantalla.NivelRoto {
				t.Errorf("despues de `%s` el veredicto es %q (%s) y el planificador "+
					"sigue muerto hace seis dias", c.nombre, v.Nivel, v.Clave)
			}
		})
	}
}

// El destino que el operador escribio tampoco se pierde por el mismo camino.
//
// Es la otra mitad del mismo agujero, y no es cosmetica: quien apunto el pulso
// a su propio monitor de "dead man's switch" (que es LA forma de que el pulso
// le avise a el, segun docs/latido.md) se encontraba con que reactivar lo
// devolvia en silencio al destino por defecto, o sea al nuestro. Su monitor
// deja de recibir pulsos y los pulsos empiezan a salir hacia nosotros.
func TestReactivarNoPuedeDevolverElDestinoDelOperadorAlNuestro(t *testing.T) {
	dir := t.TempDir()
	suyo := "https://monitor.interno.example/plazum"
	crudo := `{"activado":true,"instancia":"abc","destino":"` + suyo + `"}`
	if err := os.WriteFile(filepath.Join(dir, NombreDelFichero), []byte(crudo), 0o600); err != nil {
		t.Fatal(err)
	}
	e, err := Activar(dir, "", ahora, &secretosDePrueba{})
	if err != nil {
		t.Fatal(err)
	}
	if e.DestinoEfectivo() != suyo {
		t.Errorf("reactivar ha movido el destino de %q a %q.\n"+
			"  El pulso deja de llegar al monitor del operador y empieza a salir hacia\n"+
			"  nosotros, sin que nadie lo haya pedido", suyo, e.DestinoEfectivo())
	}
}

// CONTROL NEGATIVO de los dos de arriba. Sin esto no se sabe si el test vigila
// o si acompana: se comprueba que la comprobacion se pone roja cuando el estado
// vuelve a guardarse en cero, que es lo que hacia el camino viejo.
//
// Se reproduce el camino viejo aqui en vez de mutar el de produccion, porque
// una mutacion que hay que acordarse de deshacer no es una puerta.
func TestElControlDeLaMarcaCazaElGuardarQueBorrabaLaMarca(t *testing.T) {
	dir := t.TempDir()
	muerto := ahora.Add(-6 * 24 * time.Hour)
	crudo := `{"activado":true,"instancia":"abc","ultimo_ciclo":"` +
		muerto.Format(time.RFC3339) + `"}`
	if err := os.WriteFile(filepath.Join(dir, NombreDelFichero), []byte(crudo), 0o600); err != nil {
		t.Fatal(err)
	}

	// El Desactivar de antes del arreglo, en dos lineas: Cargar devolvia
	// Estado{} con el centinela y esto guardaba el cero encima.
	viejo := Estado{}
	viejo.Activado = false
	if err := Guardar(dir, viejo); err != nil {
		t.Fatal(err)
	}

	e, err := Cargar(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !e.UltimoCiclo.IsZero() {
		t.Fatal("el control negativo no reproduce el fallo: entonces el test de arriba " +
			"pasaria aunque nadie conservara la marca")
	}
	if v := pantalla.Vigilar(e.Marcas(), ahora); v.Nivel == pantalla.NivelRoto {
		t.Fatal("el control negativo no reproduce el fallo: con la marca borrada el " +
			"veredicto sigue saliendo roto")
	}
}
