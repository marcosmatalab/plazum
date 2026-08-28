package main

import (
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"os"

	"github.com/marcosmatalab/plazum/adaptadores/tsa"
	"github.com/marcosmatalab/plazum/nucleo/estricto"
	"github.com/marcosmatalab/plazum/nucleo/expediente"
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
	// DECODIFICACION ESTRICTA, y aqui es donde mas se nota. El contexto lo
	// escribe EL RECEPTOR con sus anclas y sus claves, y cada campo que se
	// descarte en silencio cae a un comportamiento por defecto mas permisivo:
	// `raices_tsa` mal escrito deja la verificacion usando las raices que trae
	// el binario, `anclas` mal escrito la deja sin ancla ninguna. Un auditor
	// que escribe mal una clave y recibe "verificado" ha sido enganado por su
	// propia herramienta. El porque de la clase, en nucleo/estricto.
	var f ficheroContexto
	if err := estricto.Decodificar(b, &f, "el contexto "+ruta); err != nil {
		return ctx, err
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

	// Las raices de TSA: las del receptor si las pone, y si no las que trae el
	// binario. Un verificador que no trae raices no es usable offline, y
	// offline es toda su razon de ser: nadie deberia tener que descargarse
	// certificados para poder comprobar un sello en su portatil sin red.
	pool, err := raicesDelContexto(f)
	if err != nil {
		return ctx, err
	}
	cadena := &tsa.Cadena{Anclas: pool}
	ctx.VerificarSello = cadena.VerificarOffline
	return ctx, nil
}

func raicesDelContexto(f ficheroContexto) (*x509.CertPool, error) {
	if f.RaicesTSA == "" {
		pool, err := tsa.RaicesPorDefecto()
		if err != nil {
			return nil, fmt.Errorf("no puedo cargar las raices de TSA que trae el binario: %w", err)
		}
		return pool, nil
	}
	// Si el receptor declara las suyas, valen SOLO esas: sustituyen a las de
	// por defecto, no se suman. Quien acota su confianza espera que se acote.
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
			return nil, fmt.Errorf("raices_tsa: certificado ilegible: %w", err)
		}
		pool.AddCert(cert)
		n++
	}
	if n == 0 {
		return nil, fmt.Errorf("raices_tsa no contiene ningun certificado PEM; " +
			"quitalo del fichero si quieres usar las raices que trae el binario")
	}
	return pool, nil
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
		av = append(av, "no hay con que comprobar los sellos de tiempo, asi que la cadena solo "+
			"probaria coherencia interna y no fecha")
	}
	return av
}
