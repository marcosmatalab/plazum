package main

// La verificacion de una instalacion RESTAURADA. Es la mitad que convierte esto
// en un ensayo de restauracion y no en una comprobacion de que el fichero
// existe.
//
// Una copia que devuelve bytes y deja un ledger que no verifica es PEOR que no
// tener copia, porque da confianza sin darla: el dia que hace falta, el
// expediente restaurado no prueba nada y ya no queda de donde sacarlo.
//
// Lo que se comprueba, y por que cada cosa:
//
//	1. La cadena, con el verificador del nucleo. No una relectura: la misma
//	   funcion que recorre un tercero, ledger.CadenaV2.Verificar. Encadenado,
//	   hashes, lapidas (base legal presente, atadas a SU entrada, firmadas) y
//	   checkpoints con su sello.
//	2. Las entradas VIVAS se abren con las claves restauradas. Si el keystore
//	   restaurado no corresponde a esta base, esto lo dice.
//	3. Las entradas SUPRIMIDAS siguen suprimidas. Es la comprobacion cara: no
//	   basta con que falte la clave de su indice, se prueban TODAS las claves
//	   del keystore restaurado contra la entrada. Reponer la clave borrada bajo
//	   otro indice no cuela.
//	4. La generacion del keystore no es anterior a ningun borrado de la base.
//	   Una pareja descuadrada es la forma NORMAL de resucitar una clave: nadie
//	   la repone a mano, se restaura la generacion de anteayer.
//	5. Las evidencias vivas abren y hashean a su direccion; la evidencia de la
//	   entrada suprimida no abre con ninguna clave del keystore.
//	6. El ancla de confianza no vive dentro de lo restaurado.
//
// LO QUE ESTA VERIFICACION NO PUEDE PROBAR, y no se disimula: que no exista una
// copia de la clave destruida en otro sitio. Aqui se mira la instalacion
// restaurada. Que la clave no siga viva en una generacion anterior de la
// replica, en un disco externo o en el portatil de alguien es una propiedad de
// RETENCION, no de criptografia, y se sostiene con el plazo declarado de 35
// dias (docs/copias.md) y con la politica de privacidad, no con este programa.

import (
	"bytes"
	"crypto/ed25519"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"plazum/adaptadores/tsa"
	"plazum/nucleo/blobs"
	"plazum/nucleo/ledger"
)

// Centinelas de la verificacion. Cada uno nombra UN fallo distinto: el ensayo
// tiene seis formas de ponerse rojo y un control negativo que afirmara con una
// subcadena daria verde por el motivo equivocado.
var (
	// ErrCadenaNoVerifica: la cadena restaurada no pasa el verificador del nucleo.
	ErrCadenaNoVerifica = errors.New("la cadena restaurada no verifica")
	// ErrEntradaIlegible: una entrada sin lapida no se abre con la clave restaurada.
	ErrEntradaIlegible = errors.New("una entrada viva no se abre con la clave restaurada")
	// ErrClaveResucitada: el keystore restaurado trae la clave de una entrada
	// suprimida con base legal. Es un incidente de proteccion de datos, no un
	// fallo de copia.
	ErrClaveResucitada = errors.New("la restauracion devuelve una clave destruida por borrado legal")
	// ErrSupresionLegible: alguna clave del keystore restaurado abre una entrada
	// suprimida, aunque no estuviera apuntada a su indice.
	ErrSupresionLegible = errors.New("una entrada suprimida se vuelve a abrir tras restaurar")
	// ErrKeystoreAnteriorAlBorrado: la generacion del keystore es anterior a un
	// borrado que la base ya tiene.
	ErrKeystoreAnteriorAlBorrado = errors.New("el keystore restaurado es anterior a un borrado legal de la base")
	// ErrEvidenciaNoAbre: una evidencia viva no abre, o no hashea a su direccion.
	ErrEvidenciaNoAbre = errors.New("una evidencia no abre con la clave restaurada")
	// ErrAnclaDentroDeLaCopia: el fichero de confianza vive dentro de lo restaurado.
	ErrAnclaDentroDeLaCopia = errors.New("el ancla de confianza vive dentro de lo que se restaura")
	// ErrSinLapidas: la instalacion no trae ninguna supresion, asi que el ensayo
	// no estaria mirando lo que dice mirar.
	ErrSinLapidas = errors.New("la instalacion restaurada no tiene ninguna supresion que comprobar")
	// ErrContextoIlegible: el fichero que aporta el receptor no se puede leer o
	// no sirve.
	//
	// HALLAZGO DE LA PASADA DEL COMPRADOR. Sin este centinela, teclear mal la
	// ruta del fichero de confianza salia por el mismo camino que una copia
	// rota, con "LO RESTAURADO NO PRUEBA NADA" y codigo 3. A las tres de la
	// manana eso hace creer que el respaldo esta roto cuando lo unico roto es
	// la linea de ordenes, y quien lo lea se pondra a restaurar otra generacion
	// en vez de mirar la ruta. Es la misma distincion que ya hace
	// cmd/plazum/contexto.go: un contexto a medias no invalida lo restaurado,
	// invalida la verificacion, y son dos cosas distintas.
	ErrContextoIlegible = errors.New("el contexto que aportas no se puede usar")
)

// Confianza es lo que aporta EL RECEPTOR. Mismo formato que el contexto de
// `plazum verify` para que sea el mismo fichero.
type Confianza struct {
	ClavesConfiables []string `json:"claves_confiables,omitempty"`
	ClaveOperador    string   `json:"clave_operador"`
	RaicesTSA        string   `json:"raices_tsa,omitempty"`
}

// CargarConfianza lee el fichero del receptor.
func CargarConfianza(ruta string) (ed25519.PublicKey, []string, *x509.CertPool, error) {
	var f Confianza
	b, err := os.ReadFile(ruta) // #nosec G304 -- ruta que teclea el operador en su maquina
	if err != nil {
		return nil, nil, nil, fmt.Errorf("%w: no puedo leer %s: %v.\n"+
			"  Esto NO dice nada de la copia: el fichero de confianza lo aportas TU, y es el\n"+
			"  que trae la clave publica del operador. Sin el, las lapidas se comprobarian\n"+
			"  contra la clave que viene en la propia copia, que no prueba nada.\n"+
			"  Arreglo: mira la ruta. Sirve el mismo contexto que le pasas a `plazum verify`",
			ErrContextoIlegible, ruta, err)
	}
	if err := json.Unmarshal(b, &f); err != nil {
		return nil, nil, nil, fmt.Errorf("%w: %s no es JSON valido: %v", ErrContextoIlegible, ruta, err)
	}
	pub, err := hex.DecodeString(f.ClaveOperador)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("%w: clave_operador de %s no es hexadecimal: %v",
			ErrContextoIlegible, ruta, err)
	}
	if len(pub) != ed25519.PublicKeySize {
		return nil, nil, nil, fmt.Errorf("%w: clave_operador de %s mide %d bytes y tiene que medir %d; "+
			"con una clave del tamano equivocado no se pueden verificar las lapidas",
			ErrContextoIlegible, ruta, len(pub), ed25519.PublicKeySize)
	}
	pool, err := tsa.RaicesPorDefecto()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("no puedo cargar las raices de TSA: %w", err)
	}
	return pub, f.ClavesConfiables, pool, nil
}

// Resultado es lo que el ensayo imprime cuando sale bien.
type Resultado struct {
	Entradas        int
	Vivas           int
	Supresiones     []string
	EvidenciasVivas int
	MaestraAusente  bool
}

// Verificar comprueba una instalacion restaurada. El fichero de confianza NO
// puede estar dentro de dir.
func Verificar(dir, rutaConfianza string) (Resultado, error) {
	var r Resultado

	if DentroDe(dir, rutaConfianza) {
		return r, fmt.Errorf("%w: %s esta dentro de %s.\n"+
			"  Quien pueda escribir en la copia se escribiria tambien la clave con la que\n"+
			"  se comprueba su propia firma, y la verificacion se compararia consigo misma.\n"+
			"  Es el mismo agujero que tuvo el expediente en la etapa 1, aplicado a las copias.\n"+
			"  Arreglo: guarda el fichero de confianza fuera del directorio de datos y\n"+
			"  fuera de la replica",
			ErrAnclaDentroDeLaCopia, rutaConfianza, dir)
	}

	pub, confiables, pool, err := CargarConfianza(rutaConfianza)
	if err != nil {
		return r, err
	}
	base, err := CargarBase(dir)
	if err != nil {
		return r, err
	}
	ks, err := CargarKeystore(dir)
	if err != nil {
		return r, err
	}

	// 1. La cadena, con el verificador del nucleo.
	cadena := base.Cadena
	sellos := &tsa.Cadena{Anclas: pool}
	inf, err := cadena.Verificar(ledger.Confianza{
		ClavesConfiables: confiables,
		ClaveOperador:    pub,
		VerificarSello:   sellos.VerificarOffline,
	})
	if err != nil {
		return r, fmt.Errorf("%w: %w.\n"+
			"  La copia ha devuelto bytes y lo que han formado no prueba nada. Arreglo:\n"+
			"  restaura otra generacion de la base y vuelve a lanzar el ensayo; si todas\n"+
			"  fallan igual, el problema no es la copia sino lo que se estaba copiando",
			ErrCadenaNoVerifica, err)
	}
	r.Entradas = inf.Entradas
	r.Supresiones = inf.Suprimidas

	if len(cadena.Lapidas) == 0 {
		return r, fmt.Errorf("%w. Un ensayo sobre una cadena sin lapidas comprueba la mitad "+
			"que no importa: la que importa es que un borrado legal siga borrado despues de "+
			"restaurar. Arreglo: siembra el ensayo con un borrado, o apunta a una "+
			"instalacion que tenga alguno", ErrSinLapidas)
	}

	suprimidas := map[uint64]ledger.Lapida{}
	for _, l := range cadena.Lapidas {
		suprimidas[l.EntradaBorrada] = l
	}

	// 4. La generacion del keystore contra el instante de cada borrado.
	genKS, errGen := time.Parse(time.RFC3339, ks.Generacion)
	for _, l := range ordenarLapidas(cadena.Lapidas) {
		if errGen != nil {
			return r, fmt.Errorf("el keystore restaurado no declara generacion legible (%q): %v.\n"+
				"  Sin ella no se puede saber si es anterior al borrado de la entrada %d, y esa\n"+
				"  es la forma normal de resucitar una clave: no se repone a mano, se restaura\n"+
				"  la generacion de anteayer.\n"+
				"  Arreglo: la replica del keystore tiene que sellar cada generacion con su "+
				"instante", ks.Generacion, errGen, l.EntradaBorrada)
		}
		borrado, err := time.Parse(time.RFC3339, l.Instante)
		if err != nil {
			return r, fmt.Errorf("la lapida de la entrada %d declara el instante %q, que no es "+
				"RFC3339: %w", l.EntradaBorrada, l.Instante, err)
		}
		if genKS.Before(borrado) {
			return r, fmt.Errorf("%w: el keystore es del %s y la entrada %d se borro el %s "+
				"con base legal %q.\n"+
				"  Un keystore anterior al borrado trae la clave que el borrado destruyo. Una\n"+
				"  restauracion que resucita una clave borrada por el derecho de supresion es un\n"+
				"  incidente de proteccion de datos, no un fallo de copia.\n"+
				"  Arreglo: restaura la generacion del keystore POSTERIOR al borrado. Si no\n"+
				"  queda ninguna, hay que volver a destruir la clave sobre la instalacion\n"+
				"  restaurada Y registrar el incidente",
				ErrKeystoreAnteriorAlBorrado, ks.Generacion, l.EntradaBorrada, l.Instante, l.BaseLegal)
		}
	}

	// 2 y 3. Entradas vivas y entradas suprimidas.
	todas := ks.Todas()
	for _, e := range cadena.Entradas {
		l, esta := suprimidas[e.Indice]
		if !esta {
			clave, hay := ks.ClaveEntrada(e.Indice)
			if !hay {
				return r, fmt.Errorf("%w: la entrada %d no tiene lapida y su clave no esta en "+
					"el keystore restaurado.\n"+
					"  O el keystore es de otra base, o la copia se hizo entre el alta de la\n"+
					"  entrada y el alta de su clave.\n"+
					"  Arreglo: restaura la generacion del keystore que corresponda a esta base",
					ErrEntradaIlegible, e.Indice)
			}
			if _, err := ledger.AbrirComprometido(clave, e.Nonce, e.Cifrado, e.Compromiso); err != nil {
				return r, fmt.Errorf("%w: entrada %d: %w.\n"+
					"  Arreglo: la pareja base/keystore no cuadra, restaura las dos de la misma "+
					"generacion", ErrEntradaIlegible, e.Indice, err)
			}
			r.Vivas++
			continue
		}
		// Suprimida. Ninguna clave del keystore restaurado puede abrirla.
		if _, hay := ks.ClaveEntrada(e.Indice); hay {
			return r, fmt.Errorf("%w: la entrada %d tiene lapida con base legal %q del %s y su "+
				"clave esta en el keystore restaurado.\n"+
				"  La supresion que consta firmada NO ha sobrevivido a la restauracion.\n"+
				"  Arreglo: destruye la clave sobre la instalacion restaurada, comprueba que\n"+
				"  ninguna generacion de la replica dentro del plazo la conserva, y registra el\n"+
				"  incidente: el interesado tiene derecho a que el borrado sea efectivo",
				ErrClaveResucitada, e.Indice, l.BaseLegal, l.Instante)
		}
		for _, clave := range todas {
			if _, err := ledger.AbrirComprometido(clave, e.Nonce, e.Cifrado, e.Compromiso); err == nil {
				return r, fmt.Errorf("%w: la entrada %d tiene lapida con base legal %q y hay una "+
					"clave en el keystore restaurado que la abre.\n"+
					"  No estaba apuntada a su indice, asi que la comprobacion barata (mirar si\n"+
					"  falta la clave de esa entrada) habria dado verde.\n"+
					"  Arreglo: el keystore restaurado no es el que corresponde a esta base, o\n"+
					"  alguien repuso la clave a mano",
					ErrSupresionLegible, e.Indice, l.BaseLegal)
			}
		}
	}

	// 5. Las evidencias.
	for _, ev := range base.Evidencias {
		l, suprimida := suprimidas[ev.Entrada]
		clave, hay := ks.ClaveEvidencia(ev.Hash)
		if suprimida {
			if hay {
				return r, fmt.Errorf("%w: la evidencia %s cuelga de la entrada %d, que se "+
					"suprimio con base legal %q, y su clave esta en el keystore restaurado.\n"+
					"  Borrar una entrada y dejar viva su evidencia es no haber borrado: el PDF\n"+
					"  sigue abriendose.\n"+
					"  Arreglo: destruye tambien la clave de la evidencia y registra el incidente",
					ErrClaveResucitada, ev.Hash[:12], ev.Entrada, l.BaseLegal)
			}
			for _, k := range todas {
				if _, err := blobs.Abrir(k, ev.Blob()); err == nil {
					return r, fmt.Errorf("%w: la evidencia %s de la entrada suprimida %d se abre "+
						"con una clave del keystore restaurado",
						ErrSupresionLegible, ev.Hash[:12], ev.Entrada)
				}
			}
			continue
		}
		if !hay {
			return r, fmt.Errorf("%w: la evidencia %s no tiene clave en el keystore restaurado y "+
				"su entrada %d no esta suprimida.\n"+
				"  Arreglo: restaura la generacion del keystore que corresponda a esta base",
				ErrEvidenciaNoAbre, ev.Hash[:12], ev.Entrada)
		}
		if _, err := blobs.Abrir(clave, ev.Blob()); err != nil {
			return r, fmt.Errorf("%w: evidencia %s: %w.\n"+
				"  blobs.Abrir comprueba ademas que el claro hashea a la direccion, asi que\n"+
				"  esto caza tambien una evidencia sustituida por otra dentro de la copia.\n"+
				"  Arreglo: restaura otra generacion de la base",
				ErrEvidenciaNoAbre, ev.Hash[:12], err)
		}
		r.EvidenciasVivas++
	}

	// 6. La maestra no viaja en la copia, y es correcto que no viaje.
	if _, err := os.Stat(filepath.Join(dir, NombreMaestra)); os.IsNotExist(err) {
		r.MaestraAusente = true
	}
	return r, nil
}

// ordenarLapidas da un orden estable para que el primer fallo que se reporta
// sea siempre el mismo. Un ensayo que senala una entrada distinta en cada
// ejecucion no se puede reproducir.
func ordenarLapidas(ls []ledger.Lapida) []ledger.Lapida {
	out := append([]ledger.Lapida(nil), ls...)
	sort.Slice(out, func(i, j int) bool { return out[i].EntradaBorrada < out[j].EntradaBorrada })
	return out
}

// mismoContenido esta aqui para el control negativo del ensayo: comprobar que
// destruir el original lo destruye de verdad antes de restaurar. Sin ella, un
// "destruir" que no destruye dejaria el ensayo verificando el original y no la
// copia, que es el verde mas falso que puede dar esta herramienta.
func mismoContenido(a, b string) bool {
	ba, err := os.ReadFile(a) // #nosec G304 -- rutas del propio ensayo
	if err != nil {
		return false
	}
	bb, err := os.ReadFile(b) // #nosec G304 -- rutas del propio ensayo
	if err != nil {
		return false
	}
	return bytes.Equal(ba, bb)
}
