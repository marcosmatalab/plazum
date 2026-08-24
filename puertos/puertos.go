// Package puertos define las interfaces de la arquitectura hexagonal.
//
// Regla de oro: el nucleo no conoce estos tipos. Los puertos dependen del
// nucleo, los adaptadores dependen de los puertos, y nada del nucleo mira
// hacia afuera (lo vigila el test de AST de la raiz).
//
// Cada interfaz esta deliberadamente pequena: un adaptador debe poder
// escribirse en una tarde y sustituirse sin tocar nada mas.
package puertos

import (
	"obligo/nucleo/corpus"
	"obligo/nucleo/estado"
)

// Ingesta recibe evidencia aportada por humanos (ficheros, actas, exports
// firmados). Es un conector de pleno derecho, no un parche.
type Ingesta interface {
	// Aportar registra una evidencia manual con su procedencia. El contenido
	// entra al almacen de blobs content-addressed; aqui solo viaja la referencia.
	Aportar(prueba string, hashContenido string, quien string, instante string) error
}

// Recoleccion trae observaciones del mundo: conectores WASM, delegados
// (Prowler, OpenSCAP, Trivy, ScubaGear) y clientes MCP. Toda observacion
// entra con procedencia y NO corroborada por defecto.
type Recoleccion interface {
	Recolectar(fuente string, cursor string) (obs []estado.Observacion, siguiente string, err error)
}

// Almacen persiste y recupera. La implementacion de referencia es SQLite
// (con los blobs dentro, cifrados); Postgres es un segundo adaptador.
type Almacen interface {
	Guardar(clave string, valor []byte) error
	Leer(clave string) ([]byte, error)
	// GuardarBlob almacena contenido content-addressed cifrado por entrada.
	GuardarBlob(hash string, cifrado []byte) error
	LeerBlob(hash string) ([]byte, error)
}

// Notificacion entrega donde vive la gente: email y Teams primero,
// Slack y Jira despues. El canal informa de su ultimo exito (smoke test).
type Notificacion interface {
	Enviar(canal string, destinatario string, asunto string, cuerpo string) error
	UltimoExito(canal string) (instante string, err error)
}

// Escalado decide a quien toca cuando nadie atiende. La jerarquia viene de
// Identidad (SCIM manager) o del mapeo manual; los niveles que resuelven a
// la misma persona se colapsan.
type Escalado interface {
	Siguiente(obligacion string, nivel int) (destinatario string, err error)
}

// Documento rellena plantillas trazadas del corpus con datos del expediente
// y devuelve el hash del resultado para sellarlo.
type Documento interface {
	Rellenar(plantilla corpus.Plantilla, datos map[string]string) (contenido []byte, hash string, err error)
}

// Anclaje sella hacia fuera: RFC 3161 con cadena de reserva, QTSP eIDAS
// opcional, Rekor opcional. La verificacion del token es offline.
type Anclaje interface {
	Sellar(hash []byte) (token []byte, err error)
	VerificarOffline(hash []byte, token []byte) error
}

// Asistente es la UNICA puerta de la IA. Vive fuera de proceso; su unica
// salida es una Propuesta cuya cita se verifica por hash contra el corpus
// antes de mostrarse. Nada muta estado sin aceptacion humana registrada.
type Asistente interface {
	Proponer(tarea string, contexto []byte) (Propuesta, error)
}

// Propuesta es el unico tipo que la IA puede devolver.
type Propuesta struct {
	Diff         string
	Cita         string // el span exacto citado
	HashFuente   string // debe existir en el corpus o la propuesta nace bloqueada
	Modelo       string
	DigestPrompt string
}

// Identidad resuelve personas, grupos y jerarquia (OIDC para entrar,
// SCIM con la extension enterprise para el atributo manager).
type Identidad interface {
	Persona(id string) (nombre string, manager string, grupos []string, err error)
}
