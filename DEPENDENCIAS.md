# Dependencias: lista cerrada

Regla: `nucleo/` cero dependencias, para siempre (lo vigila el test de AST). Fuera del núcleo, solo lo listado aquí. Añadir una dependencia exige una fila con su porqué y su licencia, y pasar revisión.

| Módulo | Dónde | Por qué | Licencia |
|---|---|---|---|
| modernc.org/sqlite | adaptadores/sqlite | SQLite sin cgo: binario único portable | BSD-3 |
| github.com/google/cel-go | adaptadores (predicados) | CEL para predicados de verificación; no Turing completo | Apache-2.0 |
| github.com/extism/go-sdk | adaptadores/wasm | host de conectores WASM sandboxed | BSD-3 |
| golang.org/x/crypto | adaptadores | primitivas fuera de stdlib si hacen falta | BSD-3 |
| golang.org/x/oauth2 | adaptadores/oidc | OIDC | BSD-3 |
| github.com/digitorus/timestamp | adaptadores/tsa | construir la consulta RFC 3161 y parsear el token (TSTInfo); el ASN.1/CMS a mano son semanas | BSD-2 |
| github.com/digitorus/pkcs7 | adaptadores/tsa | verificar la firma CMS del token y encadenarla a las anclas de confianza; entra como transitiva de la anterior y se usa directa a propósito (ver abajo) | MIT |

Decisiones que EVITAN dependencias: los paquetes de corpus se firman con Ed25519 propio (stdlib), no cosign; la distribución es descarga HTTP firmada, no OCI; la búsqueda base es FTS5 de SQLite, no un motor vectorial; htmx va vendorizado como fichero estático, sin npm.

## Sobre las dos de RFC 3161, que hay que mirar de cerca

Se anotan aquí porque son las únicas dependencias del proyecto que parsean bytes de origen no fiable (el token viaja dentro del expediente, que lo aporta alguien de quien explícitamente no nos fiamos), y porque ninguna de las dos está en buena forma.

- **Ninguna publica versiones semver.** Van fijadas por pseudo-versión, no por tag. `timestamp` es del 2025-05-24 y `pkcs7` del 2023-07-13, o sea unos tres años sin un commit en la que hace la criptografía.
- **`timestamp.Parse` solo comprueba la firma si el token trae certificados.** Un token sin certificado le pasa entero sin que se verifique nada. Por eso `VerificarOffline` no se apoya en él para la decisión de confianza y llama a `pkcs7.VerifyWithOpts` aparte, que exige firmante y encadena por emisor y número de serie en vez de por posición. Hay un test que lo demuestra contra la librería (`TestLaLibreriaTragaUnTokenSinCertificadoYPorEsoNoLeCreemos`): si algún día deja de ser cierto, se pone rojo.
- **`pkcs7` revienta con entrada malformada.** El fuzzing del adaptador encontró que `0x30 0x84` (una SEQUENCE que declara cuatro bytes de longitud y no los trae) sale por `index out of range` en `pkcs7.readObject`. Es alcanzable con solo mandar un expediente con el token roto, así que `VerificarOffline` e `Instante` aíslan el parseo con `recover` y lo convierten en un rechazo normal. La semilla está en `adaptadores/tsa/testdata` y corre en cada `go test`.

Qué hacer con esto, por orden: reportar el panic aguas arriba, mantener el fuzzing como puerta, y al llegar el QTSP cualificado de la etapa 8 reevaluar si compensa parsear el CMS con `encoding/asn1` propio y quitarse las dos de encima. El `recover` no es una excusa para no mirarlo, es lo que impide que un token roto tumbe al verificador mientras se mira.
