package tsa

// Las dos TSAs gratuitas de arranque. Van EN ORDEN: primero FreeTSA, y Certum
// como reserva. Se eligen dos de operadores distintos a proposito, porque una
// reserva que comparte operador con la principal no es una reserva.
//
// Que son y que no son: son sellos gratuitos, buenos para el dia a dia y para
// que el expediente tenga anclaje desde el primer arranque. NO son un QTSP
// cualificado eIDAS. La promesa probatoria fuerte (la que se ensena en un
// juicio o ante un auditor exigente) pide un QTSP de la lista de confianza, y
// eso se configura sustituyendo estas dos, no anadiendolas.
//
// El operador tiene que cargar ademas las raices de estas TSAs en Anclas: sin
// ellas se pueden pedir sellos pero no verificarlos, que es la mitad inutil.
// Se dejan fuera del binario a proposito, porque un certificado embebido
// caduca y convierte una actualizacion de producto en un requisito legal.
const (
	URLFreeTSA = "https://freetsa.org/tsr"
	URLCertum  = "http://time.certum.pl"
)

// PorDefecto devuelve la cadena de arranque, con las dos TSAs puestas y sin
// anclas ni cola: eso lo pone el operador, y Revisar() se lo recuerda.
func PorDefecto() *Cadena {
	return &Cadena{
		Autoridades: []Autoridad{
			{Nombre: "FreeTSA", URL: URLFreeTSA},
			{Nombre: "Certum", URL: URLCertum},
		},
	}
}
