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

// ErrClaveHuerfana: el keystore restaurado trae la clave de una entrada que la
// base restaurada NO tiene. Las dos replicas se contradicen solas.
//
// Por que hace falta un centinela propio y no vale con contar: la base puede
// volver CORTA (tres entradas de cuatro) y el keystore traer tres claves, asi
// que los numeros cuadran y lo que no cuadra es DE QUE ENTRADA es cada una.
// Contar no basta, hay que emparejar por identidad.
var ErrClaveHuerfana = errors.New("el keystore restaurado tiene la clave de una entrada que la base no trae")

// ErrEvidenciaAusente: el keystore restaurado trae la clave de una evidencia que
// la base no contiene. La copia perdio la evidencia y nadie la echaba de menos,
// porque la verificacion solo recorria las evidencias que HAY.
var ErrEvidenciaAusente = errors.New("el keystore restaurado tiene la clave de una evidencia que la base no trae")

// Confianza es lo que aporta EL RECEPTOR. Mismo formato que el contexto de
// `plazum verify` para que sea el mismo fichero.
type Confianza struct {
	ClavesConfiables []string `json:"claves_confiables,omitempty"`
	ClaveOperador    string   `json:"clave_operador"`
	RaicesTSA        string   `json:"raices_tsa,omitempty"`

	// Acta es lo que el receptor RECUERDA de antes del desastre, y es opcional
	// porque no siempre se tiene. Cuando esta, cierra dos agujeros que ninguna
	// verificacion de la replica sola puede cerrar. El porque, en el godoc de
	// Acta.
	Acta *Acta `json:"acta,omitempty"`
}

// Acta es el recuerdo del receptor: que habia y que se borro, anotado ANTES y
// guardado FUERA de la replica.
//
// POR QUE EXISTE, y es la parte que mas conviene entender de este fichero.
//
// Un revisor hostil tumbo dos veces la propiedad que da nombre a la casilla,
// "un borrado legal sigue borrado despues de restaurar", y las dos veces por la
// misma razon de fondo: **el emparejamiento entre una evidencia y su entrada, y
// entre una entrada y su lapida, lo declara la REPLICA**, que es justo lo que
// controla quien restaura.
//
//   - Quitar UNA lapida de dos y reponer su clave: la lista de lapidas no la
//     cubre ningun hash de la cadena ni ningun checkpoint, asi que la entrada
//     vuelve a ser una entrada viva normal y todo verifica.
//   - Recolgar la evidencia suprimida de una entrada VIVA: el campo que ata una
//     evidencia a su entrada esta en la base y no lo firma nadie.
//
// Arreglarlo en el formato seria meter las lapidas y esa atadura en lo firmado.
// NO SE HACE, y no por pereza: la capa probatoria esta cerrada por decision D-2
// de docs/decisiones.md, y este hallazgo es el ataque 14. Queda escrito en
// docs/modelo-de-amenaza.md con lo que se puede y no se puede hacer.
//
// Lo que SI se puede hacer, y es lo que hay aqui, es exactamente el patron que
// el modelo de amenaza ya recomienda para el truncado de cola: **que el receptor
// sea el testigo**. Quien exige la restauracion sabe que habia antes, porque
// estaba delante. Anotarlo cuesta un fichero pequeno, no exige red, no exige un
// log de transparencia y no filtra nada a nadie.
//
// El acta viaja en el MISMO fichero que la clave del operador y con la misma
// regla: fuera del directorio de datos y fuera de la replica. Si estuviera
// dentro, quien puede escribir la copia se escribiria tambien el recuerdo con el
// que se contrasta, que es el agujero que tuvo el expediente en la etapa 1.
//
// Es OPCIONAL a proposito. Sin acta el ensayo sigue comprobando todo lo demas y
// **lo dice**: no se puede exigir un acta a quien restaura una instalacion
// ajena, y un ensayo que se niega a correr sin ella no se ejecutaria nunca.
type Acta struct {
	// Entradas son los indices que la instalacion tenia. Que no falte ninguna.
	Entradas []uint64 `json:"entradas,omitempty"`
	// Suprimidas son las entradas que se borraron con base legal. Cada una tiene
	// que seguir teniendo su lapida en la replica restaurada.
	Suprimidas []uint64 `json:"suprimidas,omitempty"`
	// EvidenciasSuprimidas son las direcciones de blob que colgaban de una
	// entrada suprimida. Ninguna clave del keystore restaurado puede abrirlas,
	// cuelguen de donde cuelguen AHORA.
	EvidenciasSuprimidas []string `json:"evidencias_suprimidas,omitempty"`
}

// CargarConfianza lee el fichero del receptor.
func CargarConfianza(ruta string) (ed25519.PublicKey, []string, *x509.CertPool, *Acta, error) {
	var f Confianza
	b, err := os.ReadFile(ruta) // #nosec G304 -- ruta que teclea el operador en su maquina
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("%w: no puedo leer %s: %v.\n"+
			"  Esto NO dice nada de la copia: el fichero de confianza lo aportas TU, y es el\n"+
			"  que trae la clave publica del operador. Sin el, las lapidas se comprobarian\n"+
			"  contra la clave que viene en la propia copia, que no prueba nada.\n"+
			"  Arreglo: mira la ruta. Sirve el mismo contexto que le pasas a `plazum verify`",
			ErrContextoIlegible, ruta, err)
	}
	if err := json.Unmarshal(b, &f); err != nil {
		return nil, nil, nil, nil, fmt.Errorf("%w: %s no es JSON valido: %v", ErrContextoIlegible, ruta, err)
	}
	pub, err := hex.DecodeString(f.ClaveOperador)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("%w: clave_operador de %s no es hexadecimal: %v",
			ErrContextoIlegible, ruta, err)
	}
	if len(pub) != ed25519.PublicKeySize {
		return nil, nil, nil, nil, fmt.Errorf("%w: clave_operador de %s mide %d bytes y tiene que medir %d; "+
			"con una clave del tamano equivocado no se pueden verificar las lapidas",
			ErrContextoIlegible, ruta, len(pub), ed25519.PublicKeySize)
	}
	pool, err := tsa.RaicesPorDefecto()
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("no puedo cargar las raices de TSA: %w", err)
	}
	return pub, f.ClavesConfiables, pool, f.Acta, nil
}

// Resultado es lo que el ensayo imprime cuando sale bien.
type Resultado struct {
	Entradas        int
	Vivas           int
	Supresiones     []string
	EvidenciasVivas int
	MaestraAusente  bool
	// ConActa dice si el receptor aporto su recuerdo. Sin el, dos comprobaciones
	// no se hacen y el ensayo TIENE que decirlo: un verde mas debil que se lee
	// igual que uno fuerte es lo que este proyecto no hace.
	ConActa bool
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

	pub, confiables, pool, acta, err := CargarConfianza(rutaConfianza)
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

	enBase := map[uint64]bool{}
	for _, e := range cadena.Entradas {
		enBase[e.Indice] = true
	}
	todas := ks.Todas()

	// 3b. EL ACTA DEL RECEPTOR, si la trae.
	//
	// Va ANTES que las comprobaciones de evidencias a proposito. Quitar la
	// lapida de una entrada deja a su evidencia pareciendo una evidencia viva
	// sin clave, asi que sin esto el ensayo salia rojo por ErrEvidenciaNoAbre:
	// un rojo correcto con el diagnostico equivocado. El operador leia "una
	// evidencia no abre" cuando lo que tenia delante era un borrado del art. 17
	// deshecho. El rojo mas preciso tiene que ganar al mas generico.
	//
	// Cierra los dos agujeros que la replica sola no puede cerrar, porque en los
	// dos el emparejamiento lo declara la propia replica. Ver el godoc de Acta.
	if acta != nil {
		r.ConActa = true

		for _, idx := range acta.Entradas {
			if enBase[idx] {
				continue
			}
			return r, fmt.Errorf("%w: el acta dice que existia la entrada %d y la base restaurada\n"+
				"  trae %d entrada(s) sin ella.\n"+
				"  Arreglo: restaura una generacion de la base que la tenga. Si ninguna la tiene,\n"+
				"  la copia perdio datos",
				ErrClaveHuerfana, idx, len(cadena.Entradas))
		}

		// Cada entrada que el acta declara suprimida SIGUE teniendo su lapida.
		//
		// Esta es la comprobacion que caza quitar una lapida de dos. Sin acta es
		// indetectable: la lista de lapidas no la cubre ningun hash, asi que
		// quitar una deja una cadena internamente coherente que verifica, y la
		// entrada vuelve a ser una entrada viva cualquiera.
		//
		// POR QUE CAMPO CASA: por el indice de la entrada, que va dentro del hash
		// encadenado. No por la posicion en la lista de lapidas, que es
		// precisamente lo que el atacante reordena.
		for _, idx := range acta.Suprimidas {
			if _, sigue := suprimidas[idx]; sigue {
				continue
			}
			return r, fmt.Errorf("%w: el acta dice que la entrada %d se suprimio con base legal y\n"+
				"  la replica restaurada NO trae su lapida.\n"+
				"  La supresion ha desaparecido en la restauracion, asi que la entrada vuelve a\n"+
				"  ser una entrada viva y su contenido se abre si la clave volvio con ella. Eso es\n"+
				"  un borrado del art. 17 deshecho, o sea un incidente de proteccion de datos.\n"+
				"  Nadie mas lo puede cazar: la lista de lapidas no la cubre ningun hash de la\n"+
				"  cadena ni ningun checkpoint, asi que quitar una deja todo lo demas coherente.\n"+
				"  Arreglo: vuelve a poner la lapida sobre la instalacion restaurada, destruye la\n"+
				"  clave otra vez, y registra el incidente",
				ErrClaveResucitada, idx)
		}

		// Ninguna evidencia que el acta declara suprimida se abre, CUELGUE DE DONDE
		// CUELGUE ahora.
		//
		// Esta caza recolgar la evidencia suprimida de una entrada viva. La
		// comprobacion de arriba no puede: pregunta por la entrada de la que la
		// evidencia DICE colgar, y ese campo esta en la base y no lo firma nadie.
		//
		// POR QUE CAMPO CASA: por la DIRECCION del blob, que es el hash de su
		// claro. Mover una evidencia de entrada no le cambia la direccion, y
		// cambiarle la direccion exige cambiar el contenido, que es lo que se
		// queria proteger. Es la unica identidad de aqui que no se puede falsear.
		suprimidaEnActa := map[string]bool{}
		for _, h := range acta.EvidenciasSuprimidas {
			suprimidaEnActa[h] = true
		}
		for _, ev := range base.Evidencias {
			if !suprimidaEnActa[ev.Hash] {
				continue
			}
			corto := ev.Hash
			if len(corto) > 12 {
				corto = corto[:12]
			}
			for _, k := range todas {
				if _, err := blobs.Abrir(k, ev.Blob()); err == nil {
					return r, fmt.Errorf("%w: el acta dice que la evidencia %s colgaba de una entrada\n"+
						"  suprimida, y una clave del keystore restaurado la abre.\n"+
						"  Ahora dice colgar de la entrada %d. Da igual de cual diga colgar: ese campo\n"+
						"  vive en la base y no lo firma nadie, asi que recolgarla de una entrada viva\n"+
						"  la saca de la comprobacion de supresiones sin tocar ni una firma.\n"+
						"  Lo que no se puede mover es la direccion del blob, que es el hash de su\n"+
						"  claro, y por ahi es por donde se la reconoce.\n"+
						"  Arreglo: destruye la clave de esa evidencia sobre la instalacion restaurada\n"+
						"  y registra el incidente",
						ErrSupresionLegible, corto, ev.Entrada)
				}
			}
		}
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

	// 5b. LA DIRECCION CONTRARIA, que es la que faltaba.
	//
	// Todo lo de arriba recorre la BASE y le pide claves al keystore. Nadie
	// recorria el keystore para preguntarle de que entrada es cada clave que
	// sobra, y ahi vivian dos ataques que un revisor hostil tumbo:
	//
	//   - la base vuelve CORTA (tres entradas de cuatro) y el ensayo salia verde
	//     diciendo "cadena verificada: 3 entradas". El keystore, que es una
	//     replica APARTE, traia clave para la entrada 3. Las dos replicas se
	//     contradecian solas y nadie las contrastaba. Ojo al recuento: 3 claves y
	//     3 entradas, o sea que contar habria dado verde igual.
	//   - la copia pierde TODAS las evidencias y el ensayo salia verde con
	//     "evidencias abiertas: 0", porque el bucle recorre las evidencias que hay
	//     y no habia ninguna. Cero se lee igual que "no habia".
	//
	// Es el invariante 7 de CLAUDE.md y es el ataque 13 del expediente otra vez:
	// cuando una comprobacion recorre una lista para contrastarla con otra, la
	// direccion que falta es la que el atacante usa.
	//
	// POR QUE CAMPO CASA, dicho en voz alta como exige la pasada 2: por el INDICE
	// de la entrada y por la DIRECCION del blob. El indice entra en el hash de la
	// entrada, que es lo que encadena y lo que cubre el checkpoint firmado; la
	// direccion de un blob es el hash de su claro, asi que no se puede mover sin
	// cambiar el contenido. Ninguno de los dos es una posicion en una lista.
	//
	// Y una clave que sobra NO se ignora por parecer inofensiva: en las tres
	// formas normales de que sobre (base corta, base de otra generacion, keystore
	// de otra instalacion) lo que hay es una pareja que no cuadra, y seguir
	// adelante es dar por restaurado algo que no lo esta.
	for _, idx := range ks.IndicesDeEntrada() {
		if enBase[idx] {
			continue
		}
		return r, fmt.Errorf("%w: la entrada %d. La base restaurada trae %d entrada(s) y\n"+
			"  ninguna es esa.\n"+
			"  Las dos replicas se contradicen: el keystore sabe de una entrada que la base\n"+
			"  no tiene. La forma normal de que pase es que la base volviera CORTA, y eso es\n"+
			"  invisible mirando solo la base, porque una cadena truncada por el final es\n"+
			"  internamente coherente y verifica.\n"+
			"  Arreglo: restaura una generacion posterior de la base, o la del keystore que\n"+
			"  corresponda a esta base. Si ninguna generacion la tiene, la copia perdio datos\n"+
			"  y hay que decirlo, no restaurar y seguir",
			ErrClaveHuerfana, idx, len(cadena.Entradas))
	}

	evEnBase := map[string]bool{}
	for _, ev := range base.Evidencias {
		evEnBase[ev.Hash] = true
	}
	for _, h := range ks.DireccionesDeEvidencia() {
		if evEnBase[h] {
			continue
		}
		corto := h
		if len(corto) > 12 {
			corto = corto[:12]
		}
		return r, fmt.Errorf("%w: la evidencia %s. La base restaurada trae %d evidencia(s) y\n"+
			"  ninguna es esa.\n"+
			"  Un ensayo que da por buena una copia sin evidencias no mide lo que dice medir:\n"+
			"  el expediente restaurado no tiene con que probar nada, y el recuento de\n"+
			"  evidencias abiertas seria cero, que se lee igual que no habia ninguna.\n"+
			"  Arreglo: restaura la generacion de la base que corresponda a este keystore",
			ErrEvidenciaAusente, corto, len(base.Evidencias))
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
