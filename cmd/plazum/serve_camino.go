package main

// EL CAMINO GUIADO, CABLEADO.
//
// QUE PASABA ANTES DE ESTO. Las piezas estaban construidas y sueltas. `plazum
// serve` levantaba las seis pantallas y la revision de accesos, y ademas:
//
//	la pantalla del acta (superficies/acta) estaba entera, con sus tests en
//	verde, y NO LA MONTABA NADIE. No habia una sola direccion del producto que
//	llevara a ella;
//	la revision de accesos si estaba montada, en /uar/, y NADIE ENLAZABA ALLI.
//	Para llegar habia que teclear la direccion, que es lo mismo que decir que
//	solo llegaba quien ya sabia que existia;
//	y no habia ningun sitio donde constara EN QUE ORDEN se recorre esto.
//
// Es el mismo fallo que tuvo `plazum serve` con su propia orden: la pieza
// estaba hecha y no se podia descubrir, y una pieza que no se descubre no esta
// hecha, por muchos tests que tenga.
//
// COMO SE ARREGLA. superficies/camino declara el orden una sola vez y lo sirve
// como pantalla; aqui se monta cada paso DONDE ESA DECLARACION DICE (leyendo
// camino.RutaDe, no escribiendo la ruta otra vez) y se le da a cada superficie
// la vuelta al camino. La otra mitad la comprueba la puerta de extremo a
// extremo de serve_camino_test.go: cada ruta declarada tiene que contestar de
// verdad en el servidor montado.

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/marcosmatalab/plazum/superficies/acta"
	"github.com/marcosmatalab/plazum/superficies/camino"
	"github.com/marcosmatalab/plazum/superficies/uar"
)

// catalogoDeInterfaz es lo que las superficies necesitan del catalogo. Se
// declara aqui, en quien monta, para no atar estas funciones al adaptador
// concreto y poder construirlas en un test con otro.
type catalogoDeInterfaz interface {
	Traducir(idioma, clave string, args ...any) string
	Idiomas() []string
	Faltantes(idioma string) []string
}

// construirCamino arma la pantalla del camino guiado.
//
// El camino que se le pasa es SIEMPRE el canonico, y se pasa explicitamente en
// vez de dejar que la superficie lo ponga por su cuenta: un camino que se
// rellena solo cuando llega vacio convierte un olvido en una pantalla
// plausible, y esta es la pantalla que dice de que va el producto.
func construirCamino(cat catalogoDeInterfaz) (*camino.Superficie, error) {
	if cat == nil {
		return nil, fmt.Errorf("camino: falta el catalogo")
	}
	return camino.Nuevo(camino.Opciones{
		Pasos:    camino.Canonico(),
		Catalogo: cat,
		Base:     camino.BasePorDefecto,
		// Los pasos cuelgan de la raiz del servidor, no del prefijo de esta
		// pantalla: "/alcance" es "/alcance", no "/camino/alcance".
		Raiz:     "",
		Estatico: "/estatico",
	})
}

// construirActa arma la pantalla del acta de revision por la direccion.
//
// SIN FUENTE, Y ESO ES LO QUE HAY HOY. El compositor de nucleo/acta existe y la
// pantalla existe, pero no hay todavia ningun adaptador que junte el programa de
// auditoria, la campana de accesos y el registro de incidentes para esta
// instalacion, asi que la pantalla se sirve en su estado vacio.
//
// Y SE MONTA IGUAL, que es la decision. Es la misma que se tomo con la revision
// de accesos sin ficheros (puerta D11-b): una pantalla que solo aparece cuando
// hay datos deja al operador sin saber que existia, y el estado vacio de esta
// dice de que se compone un acta, que es exactamente lo que quien llega
// necesita saber. Lo que NO se hace es fingir que hay un acta.
func construirActa(cat catalogoDeInterfaz, quien func(*http.Request) string) (*acta.Superficie, error) {
	if cat == nil {
		return nil, fmt.Errorf("acta: falta el catalogo")
	}
	base, _ := camino.RutaDe("acta")
	return acta.Nuevo(acta.Opciones{
		Fuente:   nil,
		Catalogo: cat,
		Base:     strings.TrimSuffix(base, "/"),
		Estatico: "/estatico",
		Quien:    quien,
		// La vuelta al camino guiado. Sin esto, esta pantalla es un callejon:
		// no tiene menu y desde ella no se enlaza a ningun otro sitio.
		CaminoRuta:  camino.BasePorDefecto + "/",
		CaminoClave: camino.ClaveTitulo,
	})
}

// montajesDelCamino son los prefijos bajo los que cuelga cada superficie que no
// es la de las seis pantallas.
//
// EL PREFIJO SALE DE LA DECLARACION DEL CAMINO y no de una cadena escrita aqui.
// Es la regla de la casa aplicada a una direccion: dos copias del mismo dato se
// separan el dia que una cambie, y el sintoma de esa separacion seria un 404 en
// un paso del camino, que es la forma mas cara de romper esto.
// Los tipos son CONCRETOS y no http.Handler a proposito: un puntero nil metido
// en un interfaz deja de ser nil, asi que un `h == nil` sobre http.Handler no
// caza el caso que importa y el mux registraria un handler que entra en panico
// al primer visitante.
func montajesDelCamino(cam *camino.Superficie, act *acta.Superficie,
	revision *uar.Superficie) []montaje {

	var out []montaje
	if cam != nil {
		out = append(out, montaje{prefijo: camino.BasePorDefecto + "/", h: cam})
	}
	if act != nil {
		out = append(out, montaje{prefijo: prefijoDelPaso("acta"), h: act})
	}
	if revision != nil {
		out = append(out, montaje{prefijo: prefijoDelPaso("uar"), h: revision})
	}
	// Un montaje sin prefijo no se registra: se lo come montarSuperficies, y
	// entonces la puerta de extremo a extremo lo ve como un 404.
	filtrados := out[:0]
	for _, m := range out {
		if m.prefijo != "" {
			filtrados = append(filtrados, m)
		}
	}
	return filtrados
}

// prefijoDelPaso da el prefijo de montaje de un paso, o cadena vacia si ese
// paso no es pantalla.
func prefijoDelPaso(id string) string {
	ruta, esPantalla := camino.RutaDe(id)
	if !esPantalla {
		return ""
	}
	if !strings.HasSuffix(ruta, "/") {
		ruta += "/"
	}
	return ruta
}
