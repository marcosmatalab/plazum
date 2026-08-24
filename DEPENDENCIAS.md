# Dependencias: lista cerrada

Regla: `nucleo/` cero dependencias, para siempre (lo vigila el test de AST). Fuera del núcleo, solo lo listado aquí. Añadir una dependencia exige una fila con su porqué y su licencia, y pasar revisión.

| Módulo | Dónde | Por qué | Licencia |
|---|---|---|---|
| modernc.org/sqlite | adaptadores/sqlite | SQLite sin cgo: binario único portable | BSD-3 |
| github.com/google/cel-go | adaptadores (predicados) | CEL para predicados de verificación; no Turing completo | Apache-2.0 |
| github.com/extism/go-sdk | adaptadores/wasm | host de conectores WASM sandboxed | BSD-3 |
| golang.org/x/crypto | adaptadores | primitivas fuera de stdlib si hacen falta | BSD-3 |
| golang.org/x/oauth2 | adaptadores/oidc | OIDC | BSD-3 |
| (pendiente de elegir) verificación RFC 3161 | adaptadores/tsa | el ASN.1/CMS a mano son semanas; candidato: github.com/digitorus/timestamp | revisar |

Decisiones que EVITAN dependencias: los paquetes de corpus se firman con Ed25519 propio (stdlib), no cosign; la distribución es descarga HTTP firmada, no OCI; la búsqueda base es FTS5 de SQLite, no un motor vectorial; htmx va vendorizado como fichero estático, sin npm.
