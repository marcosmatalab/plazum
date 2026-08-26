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
| github.com/digitorus/timestamp | adaptadores/tsa | **sólo construye la consulta RFC 3161.** Desde el 26-08-2026 no parsea nada: ver "Objetivo declarado" abajo | BSD-2 |
| github.com/digitorus/pkcs7 | ninguno, **solo transitiva** | ningún fichero de producción nuestro la importa: está vendorizada (ver abajo). Sigue en `go.mod` porque `timestamp` la importa | MIT |

Decisiones que EVITAN dependencias: los paquetes de corpus se firman con Ed25519 propio (stdlib), no cosign; la distribución es descarga HTTP firmada, no OCI; la búsqueda base es FTS5 de SQLite, no un motor vectorial; htmx va vendorizado como fichero estático, sin npm.

**`golang.org/x/oauth2` estaba planeada para `adaptadores/oidc` y se ha retirado de la tabla.** El adaptador se construyó entero con la biblioteca estándar: descubrimiento OIDC, JWKS y verificación del ID token con `crypto/rsa`, `crypto/ecdsa`, `crypto/sha256`, `crypto/subtle`, `encoding/base64` y `net/http`. Es más código que importar una biblioteca, y es a propósito, porque está en la frontera de confianza: 25 formas de falsificar un ID token comprobadas una a una, cada una con su control negativo. La fila se quita para que nadie la añada creyendo que hacía falta.

## Codigo ajeno vendorizado

No son dependencias del modulo Go: son ficheros que estan en el repo y entran en el binario con `go:embed`. Se anotan aqui igualmente, porque distribuir codigo ajeno obliga a decir cual es y con que licencia, y porque su actualizacion es manual y sin dependabot que avise.

| Fichero | Version | Donde | Por que | Licencia |
|---|---|---|---|---|
| `superficies/pantallas/estatico/htmx-2.0.10.min.js` | 2.0.10 | superficies/pantallas | interactividad de las seis pantallas sin paso de construccion ni npm; por CDN convertiria a un tercero en autor de la pagina donde el operador decide si cumple la ley, y obligaria a tener salida a internet en redes que pueden no tenerla | 0BSD |
| `adaptadores/tsa/internal/pkcs7/` (4 ficheros) | commit `57bd227bfa2f32afb86ec739a0330be8d5584378`, 2025-07-29 | adaptadores/tsa | verificar la firma CMS del token RFC 3161 **y extraer su contenido, que desde el 26-08-2026 es de donde sale tambien el TSTInfo**: y encadenarla a las anclas de confianza. Es la unica criptografia del proyecto que trabaja sobre bytes de un tercero, y su version fijada envejecio tres años sin que nadie mirara porque no hay semver que comparar. Vendorizada, la vigila nuestro fuzzing en cada CI en vez de una alerta que no llega | MIT |

La licencia viaja al lado del fichero (`htmx-LICENSE.txt`) y se sirve como un estatico mas. El nombre del fichero lleva la version dentro a proposito: asi se puede cachear para siempre y al subir de version cambia la direccion.

Al actualizar htmx: cambiar el fichero, su licencia, el nombre en `plantillas/base.html` y la fila de esta tabla. El test `TestHtmxVaVendorizadoYNoPorCDN` se pone rojo si la pagina referencia algo que no esta embebido.

De `pkcs7` no se ha vendorizado el modulo entero: solo `ber.go`, `pkcs7.go`, `verify.go` y `sign.go`, y los tres ultimos recortados. `encrypt.go` y `decrypt.go` se quedan fuera porque este adaptador no cifra ni descifra nada, y porque una copia integra deja la puerta de `gosec` en rojo con once salidas (tres `G405` y dos `G502` por `crypto/des`, una `G505` por `crypto/sha1`, cinco `G115`). **La procedencia fichero a fichero, con el `sha256` de aguas arriba y el de aqui, esta en `adaptadores/tsa/internal/pkcs7/LEEME.md`**, junto con el porque de la version elegida y el procedimiento concreto para seguir los arreglos ajenos. La tabla de ese LEEME no es prosa: `TestElVendorizadoEsElQueDiceLaProcedencia` recalcula los hashes en cada `go test` y se pone rojo si alguien toca codigo ajeno sin anotarlo.

Al actualizar el `pkcs7` vendorizado: seguir los cuatro pasos de su `LEEME.md`, portar el cambio a mano, actualizar la tabla de hashes en el mismo commit y pasar `go test ./adaptadores/tsa/... -count=1`. Cuatro recortes tocan seguridad y están numerados en la cabecera de `verify.go`; el cuarto (`opts.Roots` obligatorio) lo añadió la revisión hostil, porque sin él los dos primeros no cerraban nada: `VerifyWithOpts` con el almacén a nil comprobaba la firma y se saltaba la cadena entera, y un sello de una CA que nadie ha declarado salía válido.

## Objetivo declarado: `github.com/digitorus/pkcs7` sale del grafo de módulos

**De dos dependencias a una.** No es una aspiración: es el estado al que va este adaptador, y se escribe aquí para que no se olvide entre etapas.

### Qué se hizo el 26-08-2026, y qué queda

Hasta esa fecha `adaptadores/tsa` importaba **las dos copias de pkcs7 a la vez**: la vendorizada, para comprobar la firma, y la de aguas arriba, porque `timestamp.Parse` la usa por dentro. Y la que decidía el veredicto (qué se selló y cuándo) era **la de aguas arriba**.

Peor: `timestamp.Parse` llama a `p7.Verify()` cuando el token trae certificados, y `Verify()` es exactamente la función que el recorte 1 quitó de nuestra copia porque su propio comentario aguas arriba dice que inicializa un almacén vacío *"effectively disabling certificate verification"*.

**La salida no era vendorizar `timestamp`. Era encoger.** Vendorizar habría duplicado el deber heredado: dos `LEEME.md`, dos tablas de procedencia y dos canarios en vez de uno.

Lo hecho:

- `timestamp` se queda **sólo como constructor de la consulta**. Construir un `TimeStampReq` no es frontera de confianza: los bytes los ponemos nosotros y quien los lee es la TSA.
- El `TSTInfo` y el `TimeStampResp` se parsean con `encoding/asn1` en `adaptadores/tsa/rfc3161.go`, sobre el contenido que **nuestro** pkcs7 ya extrajo. **Un parser, los mismos bytes.**
- Lo vigila `TestTimestampSoloConstruyeLaPeticion`, que recorre el AST del paquete y falla si alguien vuelve a usar `timestamp` para otra cosa. No basta el comentario: la forma en que esto se deshace no es una decisión, es una línea que alguien escribe un martes porque la dependencia ya estaba importada.

**Lo que queda para cerrar el objetivo**: construir el `TimeStampReq` nosotros, que son unas treinta líneas de ASN.1. Mientras `timestamp` esté importada, `pkcs7` sigue **en el grafo de módulos** y por tanto **en el binario**, aunque ningún código nuestro lo llame. Eso ya no es un riesgo alcanzable, pero sí es lo que un análisis de composición de software le va a señalar al comprador, y *"no lo llamamos"* no es algo que el comprador pueda comprobar sin leerse el código.

### La regla general que sale de aquí

**Vendorizar una librería que otra dependencia también importa no quita el código de en medio: añade una copia.** Antes de vendorizar algo hay que mirar **quién más lo arrastra**. Está también en `docs/pendientes.md`, porque la próxima vez que se vendorice algo hay que releerla antes de empezar y no después.

## Sobre las dos de RFC 3161, que hay que mirar de cerca

Se anotan aquí porque son las únicas dependencias del proyecto que parsean bytes de origen no fiable: el token viaja dentro del expediente, que lo aporta alguien de quien explícitamente no nos fiamos.

- **Ninguna publica versiones semver.** Van fijadas por pseudo-versión, no por tag, así que "actualizar" es elegir un commit a mano y no hay notas de versión que leer.
- **`timestamp` lleva parada desde 2025-05-24**, que es a la vez su último commit y lo que tenemos fijado. No hay nada más nuevo que coger.
- **`pkcs7` sí está viva** (commits de agosto de 2026) y desde la etapa 2 **está vendorizada**, así que ya no es una dependencia que se actualice: es código del repositorio que se actualiza a mano con el procedimiento de su `LEEME.md`. La versión fijada sigue siendo la del **2025-07-29**, y la razón está medida, no supuesta: la rama de cabeza importa `crypto/mldsa`, que no existe en la biblioteca estándar hasta Go 1.27, y **no se sube el mínimo de Go por esto**. Entre las dos no hay ninguna versión intermedia seleccionable, porque los commits que hay en medio vienen de la ancestría de `mozilla-services/pkcs7` y su `go.mod` declara otra ruta de módulo. Y lo que importa de verdad: `ber.go`, el fichero que come los bytes del tercero, es funcionalmente idéntico en las dos, así que no nos estamos dejando ningún arreglo del parser. Todo esto, con los comandos que lo demuestran, en `adaptadores/tsa/internal/pkcs7/LEEME.md`.
- **`pkcs7` no desaparece de `go.mod` por estar vendorizada, y su línea `// indirect` no se puede borrar.** `timestamp` la sigue importando y su propio `go.mod` pide la de 2023, que es la del panic. Sin esa línea, la selección de versión mínima elige la de 2023 y el panic vuelve por la puerta de `timestamp.Parse`. Lo vigila `TestElPkcs7TransitivoNoEsElQueRevienta`, que mira el comportamiento y no el número de versión.
- **`timestamp.Parse` ya no se llama desde ningún sitio** (26-08-2026), y por eso su agujero conocido dejó de importar: sólo comprobaba la firma si el token traía certificados, así que uno sin certificado le pasaba entero sin que se verificara nada. Ahora el token lo lee `adaptadores/tsa/rfc3161.go` y la firma la comprueba `pkcs7.VerifyWithOpts` de la copia vendorizada, que exige firmante, raíces (`ErrSinAnclas`), instante (`ErrSinInstante`) y usos (`ErrSinUsos`), y encadena por emisor y número de serie en vez de por posición. Lo que se fija ahora es la propiedad nuestra, no la ajena: `TestUnTokenSinCertificadoSeEntiendePeroNoVerifica` comprueba que ese token **se parsea bien** (o sea que el rechazo no viene de que no se entienda) y **muere en la firma**, que es donde tiene que morir.

### El panic de `pkcs7`, y la lección

El fuzzing del adaptador encontró que `0x30 0x84` (una SEQUENCE que declara cuatro bytes de longitud y no los trae) salía por `index out of range` en `pkcs7.readObject`. Dos bytes tumbando al verificador, alcanzable con solo mandar un expediente con el token roto.

Lo que enseñó, que importa más que el bug: **estaba arreglado aguas arriba desde el 2025-07-29 y nosotros lo importamos igual**, porque la pseudo-versión que se eligió al añadir la dependencia era de 2023 y nadie volvió a mirar. El fuzzing no encontró un fallo de la librería, encontró un fallo nuestro de mantenimiento. La dependencia está subida y el caso concreto ya no depende de nuestro código.

El `recover` de `VerificarOffline` e `Instante` se queda igualmente, y no como parche: un parser de ASN.1 ajeno colocado justo en la frontera de confianza no puede tener la capacidad de tumbar al verificador, lo arreglen rápido o no. Las semillas están en `adaptadores/tsa/testdata` y corren en cada `go test`.

Qué queda por hacer, por orden: mantener el fuzzing como puerta, no dejar que la pseudo-versión vuelva a envejecer tres años (dependabot no avisa de módulos sin tags, así que esto es revisión manual), y al llegar el QTSP cualificado de la etapa 8 reevaluar si compensa parsear el CMS con `encoding/asn1` propio y quitarse las dos de encima.

### Y lo que vendorizar `pkcs7` NO arregla

Dicho aquí porque una copia en el repositorio da una sensación de control que no siempre se corresponde:

- **`timestamp` sigue fuera y sigue parseando los mismos bytes.** `VerificarOffline` llama a `timestamp.Parse` ANTES que a la copia vendorizada, y esa llamada usa el `pkcs7` de aguas arriba. La mitad de la frontera de confianza sigue siendo de otro. Quitarse `timestamp` de encima es la decisión de la etapa 8.
- **`govulncheck` no mira el directorio vendorizado.** Empareja por ruta de módulo y esto ya no es un módulo. Sigue viendo el `pkcs7` transitivo de `timestamp`, que hoy es la misma versión, así que en la práctica el aviso llegaría igual mientras las dos no se separen. El día que se separen, esta línea es la que hay que releer.
- **El `recover` de `VerificarOffline` sigue haciendo falta**, y ahora por dos motivos en vez de uno: el parser ajeno de `timestamp`, y el nuestro vendorizado, que es ajeno de origen aunque ya sea responsabilidad nuestra.
- **El transcodificador de BER a DER amplifica, y el arreglo no está aquí.** Lo encontró el fuzzing propio del directorio vendorizado, que es la primera vez que alguien lo mira en este proyecto: 331 bytes de entrada producen 159.693 de salida (x482), y la razón se aplana hacia x4.000. Está en la versión fijada y en la de cabeza. Como `timestamp.Parse` corre ANTES y usa el `pkcs7` de aguas arriba, una guarda dentro de la copia vendorizada no llegaría a ejecutarse: la defensa efectiva es un tope al tamaño del token, y está puesta (`maxToken`, 32 KiB, en `adaptadores/tsa/tsa.go`), con su test hostil y su medición clavada por los dos lados. Queda por reportar aguas arriba.
