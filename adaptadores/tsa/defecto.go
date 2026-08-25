package tsa

import (
	"crypto/x509"
	"embed"
	"encoding/pem"
	"fmt"
	"sync"
)

// Las dos TSAs gratuitas de arranque. Van EN ORDEN: primero FreeTSA, y Certum
// como reserva. Se eligen dos de operadores distintos a proposito, porque una
// reserva que comparte operador con la principal no es una reserva.
//
// Que son y que no son: son sellos gratuitos, buenos para el dia a dia y para
// que el expediente tenga anclaje desde el primer arranque. NO son un QTSP
// cualificado eIDAS. La promesa probatoria fuerte (la que se ensena en un
// juicio o ante un auditor exigente) pide un QTSP de la lista de confianza, y
// eso se configura sustituyendo estas dos, no anadiendolas.
const (
	URLFreeTSA = "https://freetsa.org/tsr"
	URLCertum  = "http://time.certum.pl"
)

// Las raices de esas dos TSAs viajan DENTRO del binario.
//
// Por que embebidas y no en un fichero que ponga el operador: un verificador
// que no trae raices no es usable offline, y offline es toda su razon de ser.
// El auditor abre el expediente en su maquina, sin red, meses despues, y tiene
// que poder comprobar el sello sin ir a buscar nada. Un "descargate primero
// estos certificados" convierte la promesa en un tramite.
//
// Que sean por defecto NO las hace obligatorias: el receptor pone las suyas en
// su fichero de contexto y entonces solo valen esas. Ver raices/LEEME.md para
// las fechas de caducidad y como regenerarlas.
//
//go:embed raices/*.pem
var raicesEmbebidas embed.FS

var (
	poolUnaVez sync.Once
	poolPorDef *x509.CertPool
	poolErr    error
)

// RaicesPorDefecto devuelve las raices de las TSAs que trae el binario.
func RaicesPorDefecto() (*x509.CertPool, error) {
	poolUnaVez.Do(func() {
		entradas, err := raicesEmbebidas.ReadDir("raices")
		if err != nil {
			poolErr = fmt.Errorf("no puedo leer las raices embebidas: %w", err)
			return
		}
		pool := x509.NewCertPool()
		n := 0
		for _, e := range entradas {
			if e.IsDir() {
				continue
			}
			b, err := raicesEmbebidas.ReadFile("raices/" + e.Name())
			if err != nil {
				poolErr = fmt.Errorf("no puedo leer raices/%s: %w", e.Name(), err)
				return
			}
			resto := b
			for {
				var blq *pem.Block
				blq, resto = pem.Decode(resto)
				if blq == nil {
					break
				}
				cert, err := x509.ParseCertificate(blq.Bytes)
				if err != nil {
					poolErr = fmt.Errorf("raices/%s: certificado ilegible: %w", e.Name(), err)
					return
				}
				pool.AddCert(cert)
				n++
			}
		}
		if n == 0 {
			poolErr = fmt.Errorf("no hay ninguna raiz embebida; el binario no puede verificar sellos")
			return
		}
		poolPorDef = pool
	})
	return poolPorDef, poolErr
}

// PorDefecto devuelve la cadena de arranque: las dos TSAs puestas y sus raices
// cargadas, de modo que sellar y verificar funcionan sin configurar nada. La
// cola local si la pone el operador, porque necesita una ruta en disco.
func PorDefecto() (*Cadena, error) {
	pool, err := RaicesPorDefecto()
	if err != nil {
		return nil, err
	}
	return &Cadena{
		Autoridades: []Autoridad{
			{Nombre: "FreeTSA", URL: URLFreeTSA},
			{Nombre: "Certum", URL: URLCertum},
		},
		Anclas: pool,
	}, nil
}
