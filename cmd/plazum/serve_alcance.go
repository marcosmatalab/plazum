package main

// EL CABLE ENTRE LA ENTREVISTA Y EL DISCO.
//
// `superficies/pantallas` declara el interfaz que necesita (pantallas.Alcances)
// y `adaptadores/usuarios/alcances` guarda el fichero. Ninguno de los dos conoce
// al otro, y es a proposito: un adaptador que importara una superficie ataria el
// almacenamiento a una interfaz concreta, y una superficie que importara el
// adaptador no se podria montar con otro.
//
// Aqui esta la junta, que es tres funciones y una traduccion de vocabulario. Es
// el mismo sitio y el mismo patron que `campanaEnFichero` para la revision de
// accesos y que `tokensDeLaSesion` para el token CSRF.
//
// # LA TRADUCCION NO ES UNA CONVERSION DE ENTEROS
//
// `pantallas.Respuesta` tiene CUATRO valores (sin responder, si, no,
// contradictoria) y `alcances.Respuesta` tiene DOS y su cero es invalido. Un
// `alcances.Respuesta(r)` compilaria y guardaria basura: el 3 de una
// contradictoria seria un valor que el almacen no sabe escribir, y el 0 de «sin
// responder» se colaria como una fila vacia. Se traduce a mano, con las dos que
// no se pueden guardar convertidas en ERROR y no en la nada.
//
// # Y DESDE EL 04-09-2026 TAMPOCO ES UNA TRADUCCION DE UN SOLO CAMPO
//
// Las dos partes tienen ahora una `Contestacion`, que es «un si/no O un valor,
// exactamente una de las dos». Son dos tipos distintos con la misma forma y a
// proposito: el adaptador no importa la superficie y la superficie no importa el
// adaptador, asi que la unica junta esta aqui.
//
// El cardinal que lo trajo: **35 de las 68 preguntas del corpus real se
// contestan con un valor**, y hasta ese dia ninguna de las 35 cabia en la
// cuenta. No se perdian con un aviso: se perdian como AUSENTES, que es una
// respuesta legitima, o sea sin que nada lo dijera.

import (
	"context"
	"errors"
	"fmt"

	"github.com/marcosmatalab/plazum/adaptadores/usuarios/alcances"
	"github.com/marcosmatalab/plazum/superficies/pantallas"
	"github.com/marcosmatalab/plazum/superficies/serve"
)

// alcancesDeLaInstalacion adapta el almacen de disco al interfaz de la
// superficie.
type alcancesDeLaInstalacion struct{ almacen *alcances.Almacen }

// compilacion: que la junta siga siendo lo que la superficie espera.
var _ pantallas.Alcances = alcancesDeLaInstalacion{}

func (a alcancesDeLaInstalacion) De(ctx context.Context, usuario string) (
	pantallas.AlcanceGuardado, error) {

	al, err := a.almacen.De(ctx, usuario)
	if err != nil {
		return pantallas.AlcanceGuardado{}, err
	}
	out := pantallas.AlcanceGuardado{
		Respuestas:  make(map[string]pantallas.Contestacion, len(al.Respuestas)),
		Actualizado: al.Actualizado,
	}
	for id, r := range al.Respuestas {
		v, err := deAlmacen(r)
		if err != nil {
			// NO SE DEGRADA A «esa no la tengo». Una respuesta guardada que no
			// se sabe traducir es un dato que HAY y no se entiende, y ensenar la
			// entrevista sin ella le diria a quien la respondio que no la
			// respondio. El almacen ya se niega a leer un valor asi, asi que
			// esto solo puede saltar si los dos vocabularios se separan: por eso
			// el error nombra al culpable.
			return pantallas.AlcanceGuardado{}, fmt.Errorf("la respuesta guardada a %s no se "+
				"puede traducir: %w.\n"+
				"  Los dos vocabularios de respuesta (el del almacen y el de la pantalla) se "+
				"han separado, y eso es un fallo del producto", id, err)
		}
		out.Respuestas[id] = v
	}
	return out, nil
}

func (a alcancesDeLaInstalacion) Responder(ctx context.Context, usuario, pregunta string,
	c pantallas.Contestacion) error {

	v, err := aAlmacen(c)
	if err != nil {
		return err
	}
	return a.almacen.Responder(ctx, usuario, pregunta, v)
}

func (a alcancesDeLaInstalacion) Olvidar(ctx context.Context, usuario, pregunta string) error {
	return a.almacen.Olvidar(ctx, usuario, pregunta)
}

func (a alcancesDeLaInstalacion) Reemplazar(ctx context.Context, usuario string,
	rs map[string]pantallas.Contestacion) error {

	out := make(map[string]alcances.Contestacion, len(rs))
	for id, r := range rs {
		v, err := aAlmacen(r)
		if err != nil {
			return fmt.Errorf("la respuesta a %s: %w", id, err)
		}
		out[id] = v
	}
	return a.almacen.Reemplazar(ctx, usuario, out)
}

// ErrRespuestaQueNoSeGuarda: se ha pedido guardar algo que no es «si» ni «no».
var ErrRespuestaQueNoSeGuarda = errors.New("esa respuesta no se puede guardar")

// aAlmacen traduce del vocabulario de la pantalla al del disco.
//
// LAS DOS QUE NO PASAN SON ERROR Y NO LA NADA:
//
//	SinResponder    no responder se escribe NO TENIENDO FILA, y para eso esta
//	                Olvidar. Guardarlo aqui pondria en disco una fila vacia.
//	Contradictoria  es una entrada que se contradice, y viene de la direccion de
//	                la pagina, no de una cuenta. Elegir una de las dos en
//	                silencio seria afirmar un alcance que nadie afirmo.
func aAlmacen(c pantallas.Contestacion) (alcances.Contestacion, error) {
	// LA FORMA CON VALOR VA PRIMERO Y ES EXCLUYENTE, igual que al otro lado. Una
	// contestacion que trajera las dos se rechaza abajo, en Valida(), que es la
	// unica comprobacion: repetirla aqui seria una segunda implementacion de la
	// misma regla.
	if c.EsValor() {
		v := alcances.ConValor(c.Valor)
		if err := v.Valida(); err != nil {
			return alcances.Contestacion{}, fmt.Errorf("%w: %w", ErrRespuestaQueNoSeGuarda, err)
		}
		return v, nil
	}
	switch c.Booleana {
	case pantallas.Si:
		return alcances.Booleana(alcances.Si), nil
	case pantallas.No:
		return alcances.Booleana(alcances.No), nil
	case pantallas.SinResponder:
		return alcances.Contestacion{}, fmt.Errorf("%w: «sin responder» no es una respuesta, "+
			"es la ausencia de una. Se quita con Olvidar", ErrRespuestaQueNoSeGuarda)
	case pantallas.Contradictoria:
		return alcances.Contestacion{}, fmt.Errorf("%w: una pregunta respondida que si Y que "+
			"no a la vez no se resuelve eligiendo una", ErrRespuestaQueNoSeGuarda)
	}
	return alcances.Contestacion{}, fmt.Errorf("%w: valor %d desconocido",
		ErrRespuestaQueNoSeGuarda, c.Booleana)
}

// deAlmacen traduce del vocabulario del disco al de la pantalla.
func deAlmacen(c alcances.Contestacion) (pantallas.Contestacion, error) {
	if c.EsValor() {
		return pantallas.ConValor(c.Valor), nil
	}
	switch c.Booleana {
	case alcances.Si:
		return pantallas.Booleana(pantallas.Si), nil
	case alcances.No:
		return pantallas.Booleana(pantallas.No), nil
	}
	return pantallas.Contestacion{}, fmt.Errorf("%w: valor %d desconocido",
		ErrRespuestaQueNoSeGuarda, c.Booleana)
}

// ElMismoCampoCSRF dice si las dos superficies escriben el mismo nombre de
// campo para el token.
//
// `superficies/pantallas` declara su CampoCSRF SIN importar `superficies/serve`,
// porque es un http.Handler autonomo y no depende de quien lo monte. El precio
// de esa independencia es que las dos cadenas tienen que coincidir: si se
// separan, el formulario manda `csrf` y el middleware busca otra cosa, o sea que
// TODOS los botones de la entrevista contestan 403 y nada se pone rojo, porque
// cada paquete pasa su propia suite con su propia constante.
//
// Este fichero es el unico del producto que importa las dos, o sea el unico que
// puede compararlas. Lo hace `serve_alcance_test.go`.
const ElMismoCampoCSRF = pantallas.CampoCSRF == serve.CampoCSRF
