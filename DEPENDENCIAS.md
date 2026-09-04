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

> **A 04-09-2026 el binario se compila con CERO dependencias externas.** `go.mod` no tiene ni una línea `require`, y **`go.sum` ya no existe**: el 04-09-2026 se sacó del índice y se puso en `.gitignore`. Estaba rastreado y **vacío**, que es la cicatriz que dejó la contaminación de cosign: se le quitaron las 529 líneas de sumas ajenas y se le dejó la forma. Un fichero vacío y rastreado no es neutro, es una **invitación**: la próxima herramienta que pase escribiendo sumas lo llena sin CREAR nada, o sea sin aparecer como fichero nuevo para quien mira `git status` por encima. Las cuatro filas de abajo están **planeadas** para etapas futuras y ninguna se ha añadido todavía.
>
> Lo vigilan **dos** puertas, y la segunda nació ese día. `TestElBinarioNoLlevaNingunaDependenciaExterna` mira el grafo del binario. `TestGoSumNoEsUnRecipienteVacio` mira el índice y el ignorado, **y cambia de exigencia con `go.mod`**: sin `require` pide que `go.sum` esté fuera del índice e ignorado; **con `require` exige lo contrario**, porque `go.sum` fija los hashes de lo que se descarga y un `go.sum` ignorado con dependencias dentro es un agujero de suministro, no higiene. Así la línea del `.gitignore` **caduca sola**: el día que entre la primera dependencia hay que quitarla, cambiar el otro test **a propósito** y escribir su fila aquí, los tres en el mismo commit. Que ese día haya que tocar tres cosas es justo lo que se busca.

| Módulo | Dónde | Por qué | Licencia |
|---|---|---|---|
| modernc.org/sqlite | adaptadores/sqlite | SQLite sin cgo: binario único portable | BSD-3 |
| github.com/google/cel-go | adaptadores (predicados) | CEL para predicados de verificación; no Turing completo | Apache-2.0 |
| github.com/extism/go-sdk | adaptadores/wasm | host de conectores WASM sandboxed | BSD-3 |
| golang.org/x/crypto | adaptadores | primitivas fuera de stdlib si hacen falta | BSD-3 |


Decisiones que EVITAN dependencias: los paquetes de corpus se firman con Ed25519 propio (stdlib), no cosign; la distribución es descarga HTTP firmada, no OCI; la búsqueda base es FTS5 de SQLite, no un motor vectorial; htmx va vendorizado como fichero estático, sin npm.

**`golang.org/x/oauth2` estaba planeada para `adaptadores/oidc` y se ha retirado de la tabla.** El adaptador se construyó entero con la biblioteca estándar: descubrimiento OIDC, JWKS y verificación del ID token con `crypto/rsa`, `crypto/ecdsa`, `crypto/sha256`, `crypto/subtle`, `encoding/base64` y `net/http`. Es más código que importar una biblioteca, y es a propósito, porque está en la frontera de confianza: 25 formas de falsificar un ID token comprobadas una a una, cada una con su control negativo. La fila se quita para que nadie la añada creyendo que hacía falta.

## Codigo ajeno vendorizado

No son dependencias del modulo Go: son ficheros que estan en el repo y entran en el binario con `go:embed`. Se anotan aqui igualmente, porque distribuir codigo ajeno obliga a decir cual es y con que licencia, y porque su actualizacion es manual y sin dependabot que avise.

| Fichero | Version | Donde | Por que | Licencia |
|---|---|---|---|---|
| `superficies/pantallas/estatico/htmx-2.0.10.min.js` | 2.0.10 | superficies/pantallas | interactividad de las seis pantallas sin paso de construccion ni npm; por CDN convertiria a un tercero en autor de la pagina donde el operador decide si cumple la ley, y obligaria a tener salida a internet en redes que pueden no tenerla | 0BSD |
| `superficies/pantallas/estatico/inter-var-latin.woff2` | Inter Variable, subconjunto latino, de `@fontsource-variable/inter@5.2.5` (`cdn.jsdelivr.net`, descargado el 02-09-2026; `sha256 f052ee44c3728dfd23aba8a4567150bc314d23903026fbb6ad089422c2df56af`) | superficies/pantallas | la tipografía de las cuatro superficies con pantalla. Va autoalojada por lo mismo que htmx: por CDN, un tercero se entera de quién está mirando su cumplimiento, y la página no se pinta en una red sin salida a internet, que es justo la red del comprador que más paga. Es el corte **variable**, así que son 48 KB para todos los pesos en vez de cuatro ficheros | OFL-1.1 |
| `adaptadores/tsa/internal/pkcs7/` (4 ficheros) | commit `57bd227bfa2f32afb86ec739a0330be8d5584378`, 2025-07-29 | adaptadores/tsa | verificar la firma CMS del token RFC 3161 **y extraer su contenido, que desde el 26-08-2026 es de donde sale tambien el TSTInfo**: y encadenarla a las anclas de confianza. Es la unica criptografia del proyecto que trabaja sobre bytes de un tercero, y su version fijada envejecio tres años sin que nadie mirara porque no hay semver que comparar. Vendorizada, la vigila nuestro fuzzing en cada CI en vez de una alerta que no llega | MIT |

La licencia viaja al lado del fichero (`htmx-LICENSE.txt`, `inter-LICENSE.txt`) y se sirve como un estatico mas. El nombre del fichero lleva la version dentro a proposito: asi se puede cachear para siempre y al subir de version cambia la direccion.

Al actualizar htmx: cambiar el fichero, su licencia, el nombre en `plantillas/base.html` y la fila de esta tabla. El test `TestHtmxVaVendorizadoYNoPorCDN` se pone rojo si la pagina referencia algo que no esta embebido.

De `pkcs7` no se ha vendorizado el modulo entero: solo `ber.go`, `pkcs7.go`, `verify.go` y `sign.go`, y los tres ultimos recortados. `encrypt.go` y `decrypt.go` se quedan fuera porque este adaptador no cifra ni descifra nada, y porque una copia integra deja la puerta de `gosec` en rojo con once salidas (tres `G405` y dos `G502` por `crypto/des`, una `G505` por `crypto/sha1`, cinco `G115`). **La procedencia fichero a fichero, con el `sha256` de aguas arriba y el de aqui, esta en `adaptadores/tsa/internal/pkcs7/LEEME.md`**, junto con el porque de la version elegida y el procedimiento concreto para seguir los arreglos ajenos. La tabla de ese LEEME no es prosa: `TestElVendorizadoEsElQueDiceLaProcedencia` recalcula los hashes en cada `go test` y se pone rojo si alguien toca codigo ajeno sin anotarlo.

Al actualizar el `pkcs7` vendorizado: seguir los cuatro pasos de su `LEEME.md`, portar el cambio a mano, actualizar la tabla de hashes en el mismo commit y pasar `go test ./adaptadores/tsa/... -count=1`. Cuatro recortes tocan seguridad y están numerados en la cabecera de `verify.go`; el cuarto (`opts.Roots` obligatorio) lo añadió la revisión hostil, porque sin él los dos primeros no cerraban nada: `VerifyWithOpts` con el almacén a nil comprobaba la firma y se saltaba la cadena entera, y un sello de una CA que nadie ha declarado salía válido.

## Cerrado el 26-08-2026: `digitorus` sale del binario, no sólo del camino de ejecución

Era objetivo declarado por la mañana y quedó cerrado por la tarde. **De dos dependencias a cero.**

### Cómo se llegó

Hasta esa fecha `adaptadores/tsa` importaba **las dos copias de pkcs7 a la vez**: la vendorizada, para comprobar la firma, y la de aguas arriba, porque `timestamp.Parse` la usa por dentro. Y la que decidía el veredicto (qué se selló y cuándo) era **la de aguas arriba**.

Peor: `timestamp.Parse` llama a `p7.Verify()` cuando el token trae certificados, y `Verify()` es exactamente la función que el recorte 1 quitó de nuestra copia porque su propio comentario aguas arriba dice que inicializa un almacén vacío *"effectively disabling certificate verification"*.

**La salida no era vendorizar `timestamp`. Era encoger.** Vendorizar habría duplicado el deber heredado: dos `LEEME.md`, dos tablas de procedencia y dos canarios en vez de uno.

Lo hecho:

- `timestamp` se queda **sólo como constructor de la consulta**. Construir un `TimeStampReq` no es frontera de confianza: los bytes los ponemos nosotros y quien los lee es la TSA.
- El `TSTInfo` y el `TimeStampResp` se parsean con `encoding/asn1` en `adaptadores/tsa/rfc3161.go`, sobre el contenido que **nuestro** pkcs7 ya extrajo. **Un parser, los mismos bytes.**
- Lo vigila `TestTimestampSoloConstruyeLaPeticion`, que recorre el AST del paquete y falla si alguien vuelve a usar `timestamp` para otra cosa. No basta el comentario: la forma en que esto se deshace no es una decisión, es una línea que alguien escribe un martes porque la dependencia ya estaba importada.

**Y el paso que faltaba, dado el mismo día**: el `TimeStampReq` se construye en `adaptadores/tsa/rfc3161_peticion.go`, cuarenta líneas de ASN.1 sobre una estructura de seis campos. Se traía de fuera porque *"el ASN.1 a mano son semanas"*, que era cierto del CMS entero y falso de esto.

Con `timestamp` fuera salió `pkcs7`, que era quien lo arrastraba. **La diferencia no es cosmética en un producto de seguridad**: *"no lo llamamos"* es cierto y el comprador no lo puede comprobar sin leerse el código; *"no está"* se comprueba con un comando.

```
$ go list -deps ./cmd/plazum | grep digitorus
$ echo $?
1
```

Lo vigilan tres puertas en la raíz (`dependencias_test.go`): que el binario no lleve nada de `digitorus`, que no vuelva por la puerta de atrás de una dependencia sólo-de-tests, y que no lleve **ninguna** dependencia externa.

**Lo que costó, dicho para que se sepa el precio**: la TSA de mentira de los tests la armaba `timestamp.CreateResponse`, así que hubo que construir el CMS SignedData nosotros (`adaptadores/tsa/tsafalsa_test.go`). Y eso resultó ser una mejora, no un peaje: con la respuesta armada aquí se puede emitir un token con el tipo de contenido equivocado o sin atributos firmados, que con la librería no se podía pedir.

### La regla general que sale de aquí

**Vendorizar una librería que otra dependencia también importa no quita el código de en medio: añade una copia.** Antes de vendorizar algo hay que mirar **quién más lo arrastra**. Está también en `docs/pendientes.md`, porque la próxima vez que se vendorice algo hay que releerla antes de empezar y no después.

## Sobre RFC 3161, que era donde estaban las dos dependencias

Se anota aquí porque eran las únicas dependencias del proyecto que parseaban bytes de origen no fiable: el token viaja dentro del expediente, que lo aporta alguien de quien explícitamente no nos fiamos. **Ya no existen ninguna de las dos**, y lo que queda es código de este repositorio con su fuzzing.

- **Ninguna publicaba versiones semver**, así que iban fijadas por pseudo-versión y "actualizar" era elegir un commit a mano. Ésa fue la razón de fondo de todo lo que vino después: una dependencia que dependabot no puede vigilar envejece sin que salte nada, y aquí envejeció tres años.
- **`timestamp` llevaba parada desde 2025-05-24**, que era a la vez su último commit y lo que teníamos fijado.
- **`pkcs7` sí está viva** y está **vendorizada** desde la etapa 2: es código del repositorio que se actualiza a mano con el procedimiento de su `LEEME.md`, y el cotejo función a función con la punta lo hace el canario mensual (`herramientas/cotejapkcs7`).
- **El agujero que tenía `timestamp.Parse`**, y que fue lo que empujó a quitarla: sólo comprobaba la firma si el token traía certificados, así que uno sin certificado le pasaba entero sin que se verificara nada. Y llamaba a `p7.Verify()`, que es exactamente la función que el recorte 1 quitó de nuestra copia porque desactiva la verificación de cadena.

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
