package main

import (
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"os"

	"dutiq/adaptadores/tsa"
	"dutiq/nucleo/expediente"
)

// ficheroContexto es lo que aporta EL RECEPTOR, y no sale nunca del expediente.
//
// Existe por el bloqueante de la revision hostil de la etapa 1: antes las
// anclas y las claves confiables viajaban dentro del fichero del emisor, asi
// que la verificacion lo comparaba consigo mismo. Esto es el otro lado del
// contrato: el auditor trae su propio fichero, obtenido del registro firmado y
// no de quien le entrega el expediente.
type ficheroContexto struct {
	// Anclas: URN de paquete -> digest que el receptor tiene por bueno.
	Anclas map[string]string `json:"anclas"`
	// ClavesConfiables: claves publicas de checkpoint, en hex.
	ClavesConfiables []string `json:"claves_confiables"`
	// ClaveOperador: la que firma las lapidas de supresion, en hex.
	ClaveOperador string `json:"clave_operador"`
	// RaicesTSA: PEM con las raices de las TSAs que el receptor acepta. Sin
	// ellas no se puede comprobar el sello de tiempo de los checkpoints.
	RaicesTSA string `json:"raices_tsa"`
}

// cargarContexto lee el fichero del receptor y lo convierte en el contexto.
// Es explicito en los errores porque un contexto a medias no invalida el
// expediente, invalida la verificacion, y son dos cosas distintas.
func cargarContexto(ruta string) (expediente.ContextoReceptor, error) {
	var ctx expediente.ContextoReceptor
	b, err := os.ReadFile(ruta) // #nosec G304,G703 -- CLI: la ruta la teclea el operador en su propia maquina
	if err != nil {
		return ctx, fmt.Errorf("no puedo leer el contexto del receptor %s: %w; "+
			"es el fichero con tus anclas y tus claves, sin el la verificacion no decide nada", ruta, err)
	}
	var f ficheroContexto
	if err := json.Unmarshal(b, &f); err != nil {
		return ctx, fmt.Errorf("el contexto %s no es JSON valido: %w", ruta, err)
	}

	ctx.Anclas = f.Anclas
	ctx.ClavesConfiables = f.ClavesConfiables

	if f.ClaveOperador != "" {
		k, err := hex.DecodeString(f.ClaveOperador)
		if err != nil {
			return ctx, fmt.Errorf("clave_operador no es hexadecimal: %w", err)
		}
		ctx.ClaveOperador = k
	}

	if f.RaicesTSA != "" {
		pool := x509.NewCertPool()
		resto := []byte(f.RaicesTSA)
		n := 0
		for {
			var bloque *pem.Block
			bloque, resto = pem.Decode(resto)
			if bloque == nil {
				break
			}
			cert, err := x509.ParseCertificate(bloque.Bytes)
			if err != nil {
				return ctx, fmt.Errorf("raices_tsa: certificado ilegible: %w", err)
			}
			pool.AddCert(cert)
			n++
		}
		if n == 0 {
			return ctx, fmt.Errorf("raices_tsa no contiene ningun certificado PEM")
		}
		cadena := &tsa.Cadena{Anclas: pool}
		ctx.VerificarSello = cadena.VerificarOffline
	}
	return ctx, nil
}

// avisosDelContexto dice que se pierde con cada hueco, en vez de fallar en
// silencio o fingir que la verificacion vale lo mismo.
func avisosDelContexto(ctx expediente.ContextoReceptor) []string {
	var av []string
	if len(ctx.Anclas) == 0 {
		av = append(av, "sin anclas: el corpus del expediente no se contrasta con nada y "+
			"la verificacion es circular en esa parte")
	}
	if len(ctx.ClavesConfiables) == 0 {
		av = append(av, "sin claves confiables: la firma de los checkpoints se comprobaria "+
			"contra la clave que trae el propio fichero")
	}
	if len(ctx.ClaveOperador) == 0 {
		av = append(av, "sin clave del operador: no se pueden verificar las lapidas de supresion")
	}
	if ctx.VerificarSello == nil {
		av = append(av, "sin raices de TSA: no se puede comprobar ningun sello de tiempo, "+
			"asi que la cadena solo prueba coherencia interna, no fecha")
	}
	return av
}
