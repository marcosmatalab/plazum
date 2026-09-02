package incidente

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

// EL REGISTRO DE INCIDENTES EN DISCO.
//
// # Que faltaba
//
// `nucleo/incidente` tenia el objeto y su validacion desde hacia semanas, y NO
// HABIA FORMA DE LEER UNO DE DISCO. La consecuencia estaba a dos capas de
// distancia y no la veia ninguna puerta: el acta de revision por la direccion se
// compone de tres fuentes, y esta era una de las dos que no se podian leer, asi
// que la mejor pantalla del producto salia con dos tercios de su contenido
// diciendo «esta fuente no esta conectada».
//
// # Se reconstruye REPLICANDO, nunca rellenando campos
//
// `Reconstruir` no escribe en los campos privados: llama a `Abrir` y a
// `Registrar` con los mismos sucesos, en orden. Es deliberado y es la unica
// forma honesta de leer un objeto cuyo valor esta en sus reglas:
//
//	un incidente leido de disco pasa POR LAS MISMAS validaciones que uno
//	creado a mano (clase obligatoria en Clasificacion y prohibida en el resto,
//	hito obligatorio en Notificacion, ningun instante en cero, apertura unica
//	y primera);
//	y un fichero corrupto o manipulado NO PRODUCE un objeto a medias: produce
//	un error. Rellenar campos privados desde JSON convertiria el fichero en la
//	autoridad y dejaria las reglas de fuera, que es exactamente como se cuela
//	un incidente sin apertura en un expediente.
//
// Es el mismo patron de `accesos.Reconstruir`, y por la misma razon.
//
// # Sin reloj y sin red, como todo el nucleo
//
// Este fichero no llama a time.Now() ni abre nada: recibe BYTES. Quien los lee
// del disco es el adaptador, igual que con `censo.Tomar`.

var (
	// ErrRegistroIlegible: el documento no es lo que dice ser.
	ErrRegistroIlegible = errors.New("registro de incidentes ilegible")
	// ErrRegistroSinVersion: falta la version del formato.
	ErrRegistroSinVersion = errors.New("registro de incidentes sin version")
	// ErrRegistroVersionDesconocida: una version que este binario no sabe leer.
	ErrRegistroVersionDesconocida = errors.New("version de registro desconocida")
	// ErrIncidenteRepetido: dos incidentes con el mismo id en el mismo fichero.
	ErrIncidenteRepetido = errors.New("dos incidentes con el mismo id")
	// ErrSucesoIlegible: un suceso con un tipo o un instante que no se entiende.
	ErrSucesoIlegible = errors.New("suceso ilegible")
)

// VersionDelRegistro es la unica que este binario lee y escribe.
//
// SE EXIGE Y NO SE SUPONE. Un fichero sin version se leeria hoy con las reglas
// de hoy y dentro de un ano con las de entonces, en silencio y sobre el mismo
// contenido. Es la tercera forma del invariante 8 aplicada a un formato: el
// campo ausente no es «la version actual», es un dato que falta.
const VersionDelRegistro = 1

// sucesoEnDisco es la forma serializada de un Suceso.
//
// LOS TIPOS VIAJAN POR SU NOMBRE Y NO POR SU NUMERO. `Tipo` es un uint8 con
// iota, asi que serializarlo como numero ata el fichero al ORDEN de las
// constantes: insertar un tipo nuevo en medio reinterpretaria en silencio todos
// los ficheros ya escritos, y un `cierre` pasaria a leerse como `notificacion`.
// Nadie firma el orden de un iota.
type sucesoEnDisco struct {
	Tipo             string `json:"tipo"`
	Clase            string `json:"clase,omitempty"`
	Hito             string `json:"hito,omitempty"`
	InstanteHecho    string `json:"instante_hecho"`
	InstanteRegistro string `json:"instante_registro"`
	Fuente           string `json:"fuente,omitempty"`
}

type incidenteEnDisco struct {
	ID      string          `json:"id"`
	Sucesos []sucesoEnDisco `json:"sucesos"`
}

type registroEnDisco struct {
	Version    int                `json:"version"`
	Incidentes []incidenteEnDisco `json:"incidentes"`
}

// tipoPorNombre resuelve el nombre del tipo. Se construye de nombresDeTipo, que
// es la misma lista que usa String(): dos listas se separan el dia que alguien
// anada un tipo y toque solo una.
func tipoPorNombre(n string) (Tipo, bool) {
	for i, nombre := range nombresDeTipo {
		if nombre == n {
			return Tipo(i), true
		}
	}
	return 0, false
}

// instanteDeDisco lee un instante RFC3339.
//
// UN INSTANTE QUE NO SE ENTIENDE ES UN ERROR, NUNCA EL CERO. Es la tercera
// hermana del invariante 8: campo ausente, campo vacio y campo presente que no
// se entiende son tres cosas, y aqui las tres tienen que fallar, porque el cero
// de time.Time es el 1 de enero del ano 1 y de ahi salen plazos vencidos hace
// dos mil anos con cara de dato.
func instanteDeDisco(campo, v string) (time.Time, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return time.Time{}, fmt.Errorf("%w: %s esta vacio. Los dos instantes de un suceso son "+
			"obligatorios: uno dice cuando paso en el mundo y otro cuando se supo, y de ese "+
			"segundo cuentan los plazos de notificacion", ErrSucesoIlegible, campo)
	}
	t, err := time.Parse(time.RFC3339, v)
	if err != nil {
		return time.Time{}, fmt.Errorf("%w: %s = %q no es un instante RFC3339 "+
			"(2026-09-02T09:00:00Z)", ErrSucesoIlegible, campo, v)
	}
	return t.UTC(), nil
}

// Reconstruir lee un registro de incidentes y devuelve los objetos, ordenados
// por id para que dos lecturas del mismo fichero den el mismo orden.
//
// Devuelve TAMBIEN el booleano que el acta necesita: si el documento se ha
// podido leer. No es lo mismo «cero incidentes en el periodo» que «nadie ha
// conectado el registro», y el acta las pinta distinto porque son cosas
// opuestas: la primera es una noticia y la segunda un hueco.
func Reconstruir(datos []byte) ([]*Incidente, error) {
	var doc registroEnDisco
	if err := json.Unmarshal(datos, &doc); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRegistroIlegible, err)
	}
	if doc.Version == 0 {
		return nil, fmt.Errorf("%w: el documento no dice con que version del formato se "+
			"escribio. Sin ella se leeria hoy con las reglas de hoy y manana con las de "+
			"manana, en silencio y sobre el mismo contenido. Arreglo: anade "+
			"\"version\": %d", ErrRegistroSinVersion, VersionDelRegistro)
	}
	if doc.Version != VersionDelRegistro {
		return nil, fmt.Errorf("%w: el documento dice version %d y este binario lee la %d",
			ErrRegistroVersionDesconocida, doc.Version, VersionDelRegistro)
	}

	vistos := map[string]bool{}
	out := make([]*Incidente, 0, len(doc.Incidentes))
	for n, ind := range doc.Incidentes {
		if vistos[ind.ID] {
			return nil, fmt.Errorf("%w: %q sale dos veces. Dos incidentes con el mismo id "+
				"hacen que sus plazos se pisen y que el acta cuente uno por dos",
				ErrIncidenteRepetido, ind.ID)
		}
		vistos[ind.ID] = true

		if len(ind.Sucesos) == 0 {
			return nil, fmt.Errorf("%w: el incidente %d (%q) no trae ni un suceso, asi que no "+
				"tiene apertura y no se puede situar en el tiempo", ErrRegistroIlegible, n, ind.ID)
		}
		// EL PRIMERO TIENE QUE SER LA APERTURA, y se pasa por Abrir para que
		// valide igual que si lo hubiera creado una persona.
		primero := ind.Sucesos[0]
		if primero.Tipo != nombresDeTipo[Apertura] {
			return nil, fmt.Errorf("%w: el primer suceso del incidente %q es %q y tiene que "+
				"ser %q. El orden no es cosmetico: sin apertura primera no hay primer "+
				"conocimiento del que contar los plazos",
				ErrRegistroIlegible, ind.ID, primero.Tipo, nombresDeTipo[Apertura])
		}
		ocurrio, err := instanteDeDisco("instante_hecho de la apertura de "+ind.ID, primero.InstanteHecho)
		if err != nil {
			return nil, err
		}
		seSupo, err := instanteDeDisco("instante_registro de la apertura de "+ind.ID, primero.InstanteRegistro)
		if err != nil {
			return nil, err
		}
		i, err := Abrir(ind.ID, ocurrio, seSupo, primero.Fuente)
		if err != nil {
			return nil, fmt.Errorf("abriendo el incidente %q del registro: %w", ind.ID, err)
		}

		for k, s := range ind.Sucesos[1:] {
			tipo, ok := tipoPorNombre(s.Tipo)
			if !ok {
				return nil, fmt.Errorf("%w: el suceso %d del incidente %q dice ser de tipo %q, "+
					"y los que hay son %v", ErrSucesoIlegible, k+1, ind.ID, s.Tipo, nombresDeTipo)
			}
			hecho, err := instanteDeDisco(fmt.Sprintf("instante_hecho del suceso %d de %s", k+1, ind.ID), s.InstanteHecho)
			if err != nil {
				return nil, err
			}
			registro, err := instanteDeDisco(fmt.Sprintf("instante_registro del suceso %d de %s", k+1, ind.ID), s.InstanteRegistro)
			if err != nil {
				return nil, err
			}
			// POR Registrar, QUE ES LA PUERTA. Aqui se aplican las reglas de
			// campo ajeno (clase solo en clasificacion, hito solo en
			// notificacion) sin que este fichero tenga que repetirlas: una
			// segunda copia de esas reglas se separaria de la primera.
			if err := i.Registrar(Suceso{
				Tipo: tipo, Clase: s.Clase, Hito: s.Hito,
				InstanteHecho: hecho, InstanteRegistro: registro, Fuente: s.Fuente,
			}); err != nil {
				return nil, fmt.Errorf("registrando el suceso %d del incidente %q: %w",
					k+1, ind.ID, err)
			}
		}
		out = append(out, i)
	}
	sort.Slice(out, func(a, b int) bool { return out[a].ID() < out[b].ID() })
	return out, nil
}

// Escribir serializa los incidentes al formato que lee Reconstruir.
//
// EXISTE PARA QUE LA IDA Y VUELTA SEA COMPROBABLE. Un lector sin escritor se
// prueba contra ficheros escritos a mano en un test, que es una segunda
// implementacion del formato: el dia que el formato cambie, esos ficheros
// siguen verdes midiendo algo que el producto ya no escribe. Con los dos, el
// test que importa es que lo escrito se relee igual.
func Escribir(is []*Incidente) ([]byte, error) {
	doc := registroEnDisco{Version: VersionDelRegistro}
	for _, i := range is {
		if !i.Abierto() {
			return nil, fmt.Errorf("%w: un incidente sin apertura no se puede escribir: no "+
				"tiene primer conocimiento", ErrRegistroIlegible)
		}
		ind := incidenteEnDisco{ID: i.ID()}
		for _, s := range i.Sucesos() {
			ind.Sucesos = append(ind.Sucesos, sucesoEnDisco{
				Tipo: s.Tipo.String(), Clase: s.Clase, Hito: s.Hito,
				InstanteHecho:    s.InstanteHecho.UTC().Format(time.RFC3339),
				InstanteRegistro: s.InstanteRegistro.UTC().Format(time.RFC3339),
				Fuente:           s.Fuente,
			})
		}
		doc.Incidentes = append(doc.Incidentes, ind)
	}
	return json.MarshalIndent(doc, "", "  ")
}
