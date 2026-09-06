package ia

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/marcosmatalab/plazum/adaptadores/ingesta"
)

// LOS DOCUMENTOS QUE SUBE EL CLIENTE, CONVERTIDOS EN FUENTES CITABLES.
//
// Es la otra mitad de `FuentesDelCorpus`, y la unica diferencia que importa es
// la que no se ve en la firma: estas fuentes nacen con `Aportado` y no con
// `Corpus`, y por tanto un verificador `Estricto` las descarta antes de mirar
// su texto. Admitirlas exige construir el verificador diciendolo, que son mas
// letras a proposito.
//
// # POR QUE EL MARCO ES UNA ETIQUETA FIJA Y NO EL NOMBRE DEL FICHERO
//
// Porque el nombre del fichero lo elige quien lo sube. Un cliente (o alguien
// que le ha llegado al buzon) puede llamar a su PDF
// `Reglamento (UE) 2016-679.pdf`, y si ese nombre viajara al campo `Marco` la
// pantalla lo pintaria en la misma columna donde pone `rgpd@2016-679`. No haria
// falta ni mala fe: bastaria con que el nombre se parezca. Asi que el marco de
// todo lo aportado es la MISMA cadena, siempre, y la identidad del documento
// viaja por su huella, que no la elige nadie.
//
// El nombre del fichero no entra en la fuente. Quien lo necesite para pintarlo
// lo guarda al lado, indexado por huella, que es el sitio donde puede
// escaparse una sola vez.
//
// # LA HUELLA ES DEL CONTENIDO, ASI QUE SUBIRLO DOS VECES NO DUPLICA NADA
//
// Y ademas hace el identificador estable: la misma politica subida en enero y
// en marzo da las mismas fuentes, con lo que una cita guardada en enero sigue
// resolviendo. Con un identificador por subida, no.

// MarcoAportado es el marco de TODA fuente que venga de un documento del
// cliente. Fijo a proposito: ver el comentario de arriba.
const MarcoAportado = "documento aportado"

// Huella es el identificador de un documento aportado: el sha256 de sus bytes.
func Huella(datos []byte) string {
	suma := sha256.Sum256(datos)
	return hex.EncodeToString(suma[:])
}

// FuentesAportadas convierte los fragmentos de un documento leido en fuentes.
//
// huella es la de `Huella(datos)`, o sea del fichero entero y no de cada
// fragmento: es lo que ata los fragmentos entre si y lo que permite decir «esto
// sale del mismo documento».
func FuentesAportadas(huella string, doc ingesta.Documento) ([]Fuente, error) {
	if len(huella) != 64 {
		return nil, fmt.Errorf("ia: la huella de un documento aportado es un sha256 "+
			"en hexadecimal (64 caracteres) y ha llegado de %d. Arreglo: usa ia.Huella "+
			"sobre los bytes del fichero", len(huella))
	}
	out := make([]Fuente, 0, len(doc.Fragmentos))
	for _, fr := range doc.Fragmentos {
		// El identificador lleva la huella Y el orden. El orden solo no vale
		// (dos documentos chocarian) y la huella sola tampoco (todos los
		// fragmentos del mismo documento chocarian entre si).
		id := fmt.Sprintf("aportado:%s:%d", huella[:16], fr.Orden)
		f, err := NuevaFuente(id, MarcoAportado, referenciaDe(fr), "aportado",
			Aportado, true, fr.Texto)
		if err != nil {
			return nil, fmt.Errorf("construyendo la fuente de %s: %w", id, err)
		}
		out = append(out, f)
	}
	return out, nil
}

// referenciaDe dice DONDE esta un fragmento dentro de su documento, con la
// tercera forma de la nada respetada: si no se sabe la pagina, se dice que no se
// sabe en vez de poner una.
//
// Una pagina equivocada manda a una persona a mirar donde no esta y le hace
// dudar del resto de la pantalla, que es mas caro que no decirle la pagina.
func referenciaDe(fr ingesta.Fragmento) string {
	if fr.Pagina > 0 {
		return fmt.Sprintf("pagina %d", fr.Pagina)
	}
	return fmt.Sprintf("fragmento %d, pagina desconocida", fr.Orden)
}
