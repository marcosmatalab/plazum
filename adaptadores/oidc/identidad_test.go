package oidc

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/marcosmatalab/plazum/adaptadores/scim"
)

// Este fichero cierra la flecha que va del aprovisionamiento a la entrada.
//
// Los dos adaptadores NO se importan entre si en produccion: la union es el
// campo Admision del autenticador, que quien cablea rellena. Que la union sea un
// campo y no una dependencia es lo que permite sustituir el directorio sin
// tocar el flujo de identidad, y sustituir el flujo sin tocar el directorio.
// Aqui se prueba que la union funciona de verdad, porque un gancho que nadie
// engancha es un gancho que no protege.

func directorioConAna(t *testing.T) (*scim.Directorio, scim.Usuario) {
	t.Helper()
	d, err := scim.NuevoDirectorio(&secretosDeterministas{})
	if err != nil {
		t.Fatal(err)
	}
	ana, err := d.Crear(scim.Usuario{
		UserName:   "ana@ejemplo.es",
		ExternalID: "usuario-0001", // el mismo valor que el `sub` del ID token
		Mostrar:    "Ana Ejemplo",
		Activo:     true,
	}, ahoraFijo)
	if err != nil {
		t.Fatal(err)
	}
	return d, ana
}

// entrar corre el flujo completo con el IdP falso y devuelve el error si lo hay.
func entrar(t *testing.T, a *Autenticador, i *idpFalso) error {
	t.Helper()
	q := iniciar(t, a, "/")
	cuerpo := i.cuerpoBueno(clientePrueba)
	cuerpo["nonce"] = q.Get("nonce")
	i.mu.Lock()
	i.idTokenCanje = i.acunar(t, map[string]any{"alg": "RS256", "kid": i.kid, "typ": "JWT"}, cuerpo)
	i.mu.Unlock()
	_, _, _, err := a.Retorno(context.Background(), ahoraFijo, url.Values{
		"state": {q.Get("state")}, "code": {"c"},
	})
	return err
}

// TestUnUsuarioDesactivadoEnElIdPNoEntra.
//
// Desactivar a alguien en Entra ID y que siga entrando aqui es la mitad del
// offboarding, y la peor: la empresa cree que la baja esta hecha.
func TestUnUsuarioDesactivadoEnElIdPNoEntra(t *testing.T) {
	i := nuevoIdP(t)
	d, ana := directorioConAna(t)
	a, ses := autenticadorDePrueba(t, i)
	a.Admision = func(id Identidad) error { return d.PuedeEntrar(id.Sujeto, id.Correo) }

	// CONTROL NEGATIVO primero: activa, entra.
	if err := entrar(t, a, i); err != nil {
		t.Fatalf("CONTROL NEGATIVO EN ROJO: una usuaria activa y aprovisionada no entra: %v", err)
	}
	if len(ses.vivas) != 1 {
		t.Fatalf("no se abrio la sesion: %d", len(ses.vivas))
	}

	// El IdP la desactiva por SCIM.
	if _, err := d.Parchear(ana.ID, parcheoDesactivar(), ahoraFijo.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	err := entrar(t, a, i)
	if err == nil {
		t.Fatal("una usuaria desactivada en el IdP entro. Eso es la mitad del offboarding, y " +
			"la peor: la empresa cree que la baja esta hecha")
	}
	if !errors.Is(err, ErrAdmision) {
		t.Errorf("el rechazo tiene que distinguirse de un token invalido, porque el token era "+
			"bueno y el administrador buscaria el problema donde no esta: %v", err)
	}
	if !strings.Contains(err.Error(), "desactivada en el IdP") {
		t.Errorf("el motivo no lo dice: %v", err)
	}
	if len(ses.vivas) != 1 {
		t.Fatalf("se abrio una segunda sesion pese al rechazo: %d", len(ses.vivas))
	}
}

// TestUnUsuarioBorradoDelIdPNoEntra.
func TestUnUsuarioBorradoDelIdPNoEntra(t *testing.T) {
	i := nuevoIdP(t)
	d, ana := directorioConAna(t)
	a, _ := autenticadorDePrueba(t, i)
	a.Admision = func(id Identidad) error { return d.PuedeEntrar(id.Sujeto, id.Correo) }

	if err := entrar(t, a, i); err != nil {
		t.Fatalf("CONTROL NEGATIVO EN ROJO: %v", err)
	}
	if err := d.Borrar(ana.ID, ahoraFijo.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := entrar(t, a, i); err == nil {
		t.Fatal("una usuaria borrada del IdP entro")
	}
}

// TestAlguienSinAprovisionarNoEntra.
//
// Es la otra cara del gancho: que el IdP autentique a alguien no significa que
// esa persona tenga cuenta aqui. Sin esta comprobacion, cualquier usuario del
// tenant entra en el GRC de la empresa.
func TestAlguienSinAprovisionarNoEntra(t *testing.T) {
	i := nuevoIdP(t)
	d, err := scim.NuevoDirectorio(&secretosDeterministas{})
	if err != nil {
		t.Fatal(err)
	}
	a, ses := autenticadorDePrueba(t, i)
	a.Admision = func(id Identidad) error { return d.PuedeEntrar(id.Sujeto, id.Correo) }

	err = entrar(t, a, i)
	if err == nil {
		t.Fatal("alguien que el IdP autentica pero que no esta aprovisionado entro. Con eso, " +
			"cualquier usuario del tenant entra en el GRC de la empresa")
	}
	if !strings.Contains(err.Error(), "aprovisionada") {
		t.Errorf("el motivo no lo explica: %v", err)
	}
	if len(ses.vivas) != 0 {
		t.Fatal("se abrio sesion")
	}

	// Control negativo: en cuanto se aprovisiona, entra.
	if _, err := d.Crear(scim.Usuario{
		UserName: "ana@ejemplo.es", ExternalID: "usuario-0001", Activo: true,
	}, ahoraFijo); err != nil {
		t.Fatal(err)
	}
	if err := entrar(t, a, i); err != nil {
		t.Fatalf("tras aprovisionarla tiene que entrar: %v", err)
	}
}

// TestElEscaladoDeUnaObligacionHuerfanaSaleDelManagerDelIdP.
//
// Es el motivo por el que el atributo `manager` esta en el encargo: cuando una
// obligacion vence y su responsable no responde, hay que subirla. Este test
// recorre entero el camino desde el aprovisionamiento hasta la persona a la que
// se avisa.
func TestElEscaladoDeUnaObligacionHuerfanaSaleDelManagerDelIdP(t *testing.T) {
	d, ana := directorioConAna(t)
	jefa, err := d.Crear(scim.Usuario{UserName: "jefa@ejemplo.es", Mostrar: "La Jefa", Activo: true}, ahoraFijo)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.Parchear(ana.ID, parcheoManager(jefa.ID), ahoraFijo); err != nil {
		t.Fatal(err)
	}
	// La empresa la da de baja.
	if _, err := d.Parchear(ana.ID, parcheoDesactivar(), ahoraFijo.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	e := d.EstadoDe(ana.ID)
	if !e.Huerfana {
		t.Fatal("la obligacion de una persona dada de baja no sale como huerfana")
	}
	if e.Nombre != "Ana Ejemplo" {
		t.Errorf("no se puede ensenar de quien era: %q", e.Nombre)
	}
	if len(e.Escalado) != 1 || e.Escalado[0] != jefa.ID {
		t.Fatalf("el escalado es %v y tenia que ser la jefa. Sin esto, el manager de la "+
			"extension enterprise no sirve para nada", e.Escalado)
	}
}

func parcheoDesactivar() scim.Parcheo {
	return scim.Parcheo{Operaciones: []scim.Operacion{
		{Op: "replace", Ruta: "active", Valor: []byte(`false`)},
	}}
}

func parcheoManager(id string) scim.Parcheo {
	return scim.Parcheo{Operaciones: []scim.Operacion{
		{Op: "replace", Ruta: "manager", Valor: []byte(`{"value":"` + id + `"}`)},
	}}
}
