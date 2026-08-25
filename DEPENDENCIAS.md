# Dependencias: lista cerrada

Regla: `nucleo/` cero dependencias, para siempre (lo vigila el test de AST). Fuera del núcleo, solo lo listado aquí. Añadir una dependencia exige una fila con su porqué y su licencia, y pasar revisión.

## Regla de los módulos sin semver

Aprendida a base de un susto, no de teoría (el caso completo está más abajo, en la sección de RFC 3161).

**Un módulo Go sin tags de versión no lo vigila nadie.** Dependabot compara versiones semánticas; ante una pseudo-versión no tiene con qué comparar y se queda callado. El resultado es que una dependencia fijada por commit puede envejecer años sin que salte ni una alerta, mientras aguas arriba se arreglan fallos que tú sigues importando. Es peor que una dependencia abandonada, porque una abandonada al menos se nota.

Por tanto, para toda dependencia sin semver:

1. **Se anota en su fila que no publica semver**, para que el que la lea sepa que su actualización es manual.
2. **Se revisa a mano al cerrar cada etapa**, junto al resto de puertas. Comparar la fecha de la pseudo-versión fijada con la del último commit aguas arriba es un minuto.
3. **Se prefiere una dependencia con tags** cuando existe alternativa razonable, aunque sea algo peor por lo demás.
4. Si además parsea entrada no fiable, **fuzzing propio sobre ella**, y no se da por buena la ausencia de fallos conocidos: los fallos conocidos son los que alguien buscó.

| Módulo | Dónde | Por qué | Licencia |
|---|---|---|---|
| modernc.org/sqlite | adaptadores/sqlite | SQLite sin cgo: binario único portable | BSD-3 |
| github.com/google/cel-go | adaptadores (predicados) | CEL para predicados de verificación; no Turing completo | Apache-2.0 |
| github.com/extism/go-sdk | adaptadores/wasm | host de conectores WASM sandboxed | BSD-3 |
| golang.org/x/crypto | adaptadores | primitivas fuera de stdlib si hacen falta | BSD-3 |
| github.com/digitorus/timestamp | adaptadores/tsa | construir la consulta RFC 3161 y parsear el token (TSTInfo); el ASN.1/CMS a mano son semanas | BSD-2 |
| github.com/digitorus/pkcs7 | adaptadores/tsa | verificar la firma CMS del token y encadenarla a las anclas de confianza; entra como transitiva de la anterior y se usa directa a propósito (ver abajo) | MIT |

Decisiones que EVITAN dependencias: los paquetes de corpus se firman con Ed25519 propio (stdlib), no cosign; la distribución es descarga HTTP firmada, no OCI; la búsqueda base es FTS5 de SQLite, no un motor vectorial; htmx va vendorizado como fichero estático, sin npm.

**`golang.org/x/oauth2` estaba planeada para `adaptadores/oidc` y se ha retirado de la tabla.** El adaptador se construyó entero con la biblioteca estándar: descubrimiento OIDC, JWKS y verificación del ID token con `crypto/rsa`, `crypto/ecdsa`, `crypto/sha256`, `crypto/subtle`, `encoding/base64` y `net/http`. Es más código que importar una biblioteca, y es a propósito, porque está en la frontera de confianza: 25 formas de falsificar un ID token comprobadas una a una, cada una con su control negativo. La fila se quita para que nadie la añada creyendo que hacía falta.

## Codigo ajeno vendorizado

No son dependencias del modulo Go: son ficheros que estan en el repo y entran en el binario con `go:embed`. Se anotan aqui igualmente, porque distribuir codigo ajeno obliga a decir cual es y con que licencia, y porque su actualizacion es manual y sin dependabot que avise.

| Fichero | Version | Donde | Por que | Licencia |
|---|---|---|---|---|
| `superficies/pantallas/estatico/htmx-2.0.10.min.js` | 2.0.10 | superficies/pantallas | interactividad de las seis pantallas sin paso de construccion ni npm; por CDN convertiria a un tercero en autor de la pagina donde el operador decide si cumple la ley, y obligaria a tener salida a internet en redes que pueden no tenerla | 0BSD |

La licencia viaja al lado del fichero (`htmx-LICENSE.txt`) y se sirve como un estatico mas. El nombre del fichero lleva la version dentro a proposito: asi se puede cachear para siempre y al subir de version cambia la direccion.

Al actualizar htmx: cambiar el fichero, su licencia, el nombre en `plantillas/base.html` y la fila de esta tabla. El test `TestHtmxVaVendorizadoYNoPorCDN` se pone rojo si la pagina referencia algo que no esta embebido.

## Sobre las dos de RFC 3161, que hay que mirar de cerca

Se anotan aquí porque son las únicas dependencias del proyecto que parsean bytes de origen no fiable: el token viaja dentro del expediente, que lo aporta alguien de quien explícitamente no nos fiamos.

- **Ninguna publica versiones semver.** Van fijadas por pseudo-versión, no por tag, así que "actualizar" es elegir un commit a mano y no hay notas de versión que leer.
- **`timestamp` lleva parada desde 2025-05-24**, que es a la vez su último commit y lo que tenemos fijado. No hay nada más nuevo que coger.
- **`pkcs7` sí está viva** (commits de agosto de 2026). El aviso aquí es el contrario: quedarse atrás sale caro, y ya pasó una vez (ver abajo). Su rama actual exige **Go 1.27**, muy por encima del 1.24 que declara nuestro go.mod, así que la versión fijada es la del **2025-07-29**, la última que trae los arreglos y todavía compila con 1.24. Al subir el mínimo de Go, revisar si compensa ir a la de cabeza.
- **`timestamp.Parse` solo comprueba la firma si el token trae certificados.** Un token sin certificado le pasa entero sin que se verifique nada. Por eso `VerificarOffline` no se apoya en él para la decisión de confianza y llama a `pkcs7.VerifyWithOpts` aparte, que exige firmante y encadena por emisor y número de serie en vez de por posición. Hay un test que lo demuestra contra la librería (`TestLaLibreriaTragaUnTokenSinCertificadoYPorEsoNoLeCreemos`): si algún día deja de ser cierto, se pone rojo.

### El panic de `pkcs7`, y la lección

El fuzzing del adaptador encontró que `0x30 0x84` (una SEQUENCE que declara cuatro bytes de longitud y no los trae) salía por `index out of range` en `pkcs7.readObject`. Dos bytes tumbando al verificador, alcanzable con solo mandar un expediente con el token roto.

Lo que enseñó, que importa más que el bug: **estaba arreglado aguas arriba desde el 2025-07-29 y nosotros lo importamos igual**, porque la pseudo-versión que se eligió al añadir la dependencia era de 2023 y nadie volvió a mirar. El fuzzing no encontró un fallo de la librería, encontró un fallo nuestro de mantenimiento. La dependencia está subida y el caso concreto ya no depende de nuestro código.

El `recover` de `VerificarOffline` e `Instante` se queda igualmente, y no como parche: un parser de ASN.1 ajeno colocado justo en la frontera de confianza no puede tener la capacidad de tumbar al verificador, lo arreglen rápido o no. Las semillas están en `adaptadores/tsa/testdata` y corren en cada `go test`.

Qué queda por hacer, por orden: mantener el fuzzing como puerta, no dejar que la pseudo-versión vuelva a envejecer tres años (dependabot no avisa de módulos sin tags, así que esto es revisión manual), y al llegar el QTSP cualificado de la etapa 8 reevaluar si compensa parsear el CMS con `encoding/asn1` propio y quitarse las dos de encima.
