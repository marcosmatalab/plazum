package pantalla

import (
	"sort"

	"github.com/marcosmatalab/plazum/nucleo/corpus"
)

// EL AVISO DE DIRECTIVA, DEL CORPUS A LA PANTALLA.
//
// # Que se estaba diciendo, y por que era un P0
//
// Una directiva de la Union NO OBLIGA A NADIE por si misma: obliga la norma
// nacional que la transpone. Mientras eso no se dice, la tabla de controles le
// dice «Te aplica» a un espanol sobre NIS2, que a fecha de hoy no esta
// transpuesta. No es un matiz academico: es una afirmacion sobre el cumplimiento
// de alguien, y es falsa.
//
// Hasta el commit 7e4850d el aviso existia como PROSA REPETIDA dentro del
// paquete: diez copias en cuatro redacciones dentro de los campos `cita`, seis
// nombrando el RD 43/2021 y cuatro no, o sea que ya habian divergido. Aquel
// commit hizo lo correcto —convertirlo en el campo `transposicion`, con linter
// en las dos direcciones— y dejo la otra mitad sin hacer: el campo no lo pintaba
// NADIE. `git ls-files | xargs grep -c Transposicion` fuera de nucleo/corpus y
// herramientas/ daba cero. O sea que el aviso paso de estar mal dicho diez veces
// a no estar dicho ninguna, que es peor.
//
// # Por que clave y datos, y no una frase
//
// Porque una frase que escriba el nucleo se imprime igual en la pagina inglesa,
// y eso ya paso una vez: estado.Calcular escribia el porque de cada estado en
// castellano y la pagina en ingles lo imprimia tal cual. La regla de la casa es
// que todo lo que plazum ESCRIBE sale del catalogo, y que los datos (un pais,
// una fecha, el nombre de una norma) viajan APARTE y no se traducen nunca.
//
// # Por que campo casa (invariante 7)
//
// Por Marco, que es el URN del paquete, y NUNCA por posicion. La fila de la
// tabla ya trae `Fila.Paquete = pq.URN` y la ficha del calendario ya trae
// `Fecha.Marco = p.URN`: el aviso se busca por esa misma cadena. No es un
// emparejamiento entre dos listas construidas por separado, es una busqueda por
// identidad, asi que insertar o reordenar paquetes no puede mover el aviso de
// sitio.
//
// LO QUE NO ESTA FIRMADO, dicho al lado del cable: los paquetes de corpus NO se
// firman hoy (nucleo/corpus/cargar.go no calcula ni verifica ninguna huella y
// corpus.Paquete no tiene campo de firma). El URN es la identidad del paquete en
// todo el producto y el primer campo de su fichero; emparejar por el hereda la
// garantia del arbol de git y ninguna otra. El dia que el corpus se firme, esto
// se convierte en la garantia que hoy no es.

// AvisoDeMarco es lo que hay que decir sobre un marco entero, ademas de lo que
// diga cada una de sus obligaciones.
//
// Hoy solo lo produce la transposicion de una directiva. Se llama por lo que ES
// y no por lo que trae para que la segunda clase de aviso no obligue a un tipo
// nuevo ni a un segundo raíl hasta las plantillas.
type AvisoDeMarco struct {
	// Marco es el URN del paquete. Es el campo por el que casa (invariante 7).
	Marco string `json:"marco"`
	// Clave es la clave de catalogo del aviso. Lo que se dice lo escribe
	// plazum, asi que se traduce.
	Clave string `json:"clave"`
	// Datos son los huecos de esa clave, en orden. Son DATOS (un codigo de
	// pais, una fecha) y no se traducen nunca.
	Datos []string `json:"datos,omitempty"`
}

// Las claves del aviso de directiva. Son dos y no una porque son dos
// afirmaciones distintas, y confundirlas cuesta en las dos direcciones: decir
// «no consta transpuesta» de una que si lo esta manda al operador a buscar una
// norma que ya existe, y decir «ya esta transpuesta» de una que no lo esta es
// exactamente el «Te aplica» falso que este fichero viene a arreglar.
const (
	// ClaveDirectivaConsta lleva un dato: el pais.
	ClaveDirectivaConsta = "aviso.directiva.consta"
	// ClaveDirectivaNoConsta lleva dos: el pais y la fecha de comprobacion.
	//
	// LA FECHA VA DENTRO Y NO ES ADORNO. «No consta transpuesta» es una
	// afirmacion sobre el mundo que caduca sola: la escribe una persona
	// mirando el BOE un dia concreto y nada del arbol vuelve a mirar. Sin la
	// fecha al lado, quien la lea no puede saber si mirar otra vez, que es la
	// unica accion que le queda.
	ClaveDirectivaNoConsta = "aviso.directiva.no_consta"
)

// avisosDeTransposicion saca un aviso por (paquete, pais declarado).
//
// Un paquete sin bloque `transposicion` no produce ninguno, y eso NO es un
// olvido que haya que rellenar aqui: el linter de corpus ya exige las dos
// direcciones (una directiva sin bloque no carga, y un bloque en lo que no es
// directiva tampoco), asi que la ausencia significa «esto no es una directiva».
// Poner un aviso por defecto seria avisar de una transposicion a un reglamento.
func avisosDeTransposicion(ps []*corpus.Paquete) []AvisoDeMarco {
	var out []AvisoDeMarco
	for _, p := range ps {
		if p.Transposicion == nil {
			continue
		}
		for _, e := range p.Transposicion.Estado {
			a := AvisoDeMarco{Marco: p.URN, Clave: ClaveDirectivaNoConsta,
				Datos: []string{e.Pais, e.Comprobado}}
			if e.Consta {
				a.Clave, a.Datos = ClaveDirectivaConsta, []string{e.Pais}
			}
			out = append(out, a)
		}
	}
	// Orden total por (marco, primer dato): el mismo corpus tiene que dar el
	// mismo modelo aunque el cargador recorra el directorio en otro orden, que
	// es lo que hace comparables los casos dorados.
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Marco != out[j].Marco {
			return out[i].Marco < out[j].Marco
		}
		return primerDato(out[i]) < primerDato(out[j])
	})
	return out
}

func primerDato(a AvisoDeMarco) string {
	if len(a.Datos) == 0 {
		return ""
	}
	return a.Datos[0]
}
