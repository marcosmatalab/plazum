# pkcs7 vendorizado

Copia en el repositorio de la parte de `github.com/digitorus/pkcs7` que verifica
la firma CMS de un sello de tiempo RFC 3161. No es una dependencia del modulo
Go: es codigo ajeno que vive aqui dentro y entra en el binario como el nuestro.

## Por que esta aqui y no en go.mod

Esta libreria hace criptografia en la frontera de confianza. El token que
verifica lo trae el expediente, o sea alguien de quien explicitamente no nos
fiamos, y una vez nos costo un panico alcanzable con dos bytes: `0x30 0x84`, una
SEQUENCE que declara cuatro bytes de longitud y no los trae, salia por
`index out of range` en `readObject`.

Lo que enseño ese fallo no fue el bug, fue de quien era la culpa. **Estaba
arreglado aguas arriba trece meses antes de que lo encontraramos.** La libreria
esta viva y bien mantenida; lo que fallo fue nuestra vigilancia: no publica tags
de semver, asi que dependabot no tiene con que comparar, se queda callado, y la
pseudo-version que alguien eligio en 2023 envejecio tres años sin que nadie
mirara.

Vendorizar cambia esa vigilancia silenciosa por dos cosas que si se ven: codigo
en el repositorio, y fuzzing propio sobre el en cada CI (`fuzz_test.go`).

El coste, y hay que decirlo: **heredamos seguir sus arreglos a mano.** Como se
hace esta escrito mas abajo, con el comando concreto.

## Procedencia

| Dato | Valor |
|---|---|
| Origen | https://github.com/digitorus/pkcs7 |
| Commit | `57bd227bfa2f32afb86ec739a0330be8d5584378` (pseudo-version `v0.0.0-20250729175123-57bd227bfa2f`) |
| Asunto del commit | "Fix CI workflow and address code quality improvements" |
| Fecha del commit | 2025-07-29 17:51:23 UTC |
| Licencia | MIT, copia integra en `LICENSE`, en este mismo directorio |
| Autor original | Copyright (c) 2015 Andrew Smith, y los contribuidores de digitorus/pkcs7 |
| Vendorizado el | 2026-08-26 |

El `sha256` de aguas arriba se puede comprobar sin fiarse de esta tabla:

```
go mod download -x github.com/digitorus/pkcs7@v0.0.0-20250729175123-57bd227bfa2f
sha256sum "$(go env GOMODCACHE)/github.com/digitorus/pkcs7@v0.0.0-20250729175123-57bd227bfa2f/ber.go"
```

| Fichero | sha256 aguas arriba | sha256 aqui | Estado |
|---|---|---|---|
| `ber.go` | `2c93a570b68b10db2ab7d9bdc245e7bccae89275c0f93f6a8df827aaf1608bc3` | `22193c0743ba4d3144b6c6cf1477b4259cb47eda8408c0226b2dcb3861feeb39` | verbatim, mas cabecera y tres `#nosec G115` |
| `pkcs7.go` | `8a9110f5688ce01d0b4c24ebeb58ee8e189fb8de970829f0c10333654034947b` | `6a5a0d50dd9c61681c345bca960b1a1ba767f10fa9e75d6076b44179ea248981` | recortado, ver abajo |
| `verify.go` | `f6e2123e957c17b770ce721d6b54513ae2a68ddf93f703a9c0be6088b22ec986` | `2386bc409471e0280e3d877fce5bb1972142b5eb91500ea90406be95e1b2c6a8` | recortado, ver abajo |
| `sign.go` | `0bdbba5bfb4e6400e836e0f7792a62d1896453069d136a8bcd935c9fb3213403` | `f97679291c7e1c1242d6b5f76a64bf31898f7f2a5560ac28bd0d241a8cd57884` | recortado a las estructuras ASN.1 |
| `LICENSE` | `d01c6d371866b3c7a1a7e20994d88d2ce83f22974ec6d3596a6125b44495813d` | `d01c6d371866b3c7a1a7e20994d88d2ce83f22974ec6d3596a6125b44495813d` | verbatim |

`encrypt.go` y `decrypt.go` de aguas arriba NO estan aqui, a proposito.

La columna de la derecha la comprueba `TestElVendorizadoEsElQueDiceLaProcedencia`
en cada `go test`. Si alguien toca un fichero vendorizado y no actualiza esta
tabla, se pone rojo. Lo que ese test prueba, dicho con precision: que nadie ha
editado el codigo ajeno sin dejarlo escrito aqui. No prueba que la columna de la
izquierda sea cierta; eso se comprueba con el comando de arriba, contra el
proxy de modulos, que trae ademas la firma de la suma de comprobacion.

## Por que ESTA version y no la de cabeza

La decision estaba marcada como el sitio donde esto se puede torcer, asi que va
con la comprobacion hecha, no con la intencion.

**La rama de cabeza no compila con nuestro Go.** El `go.mod` de aguas arriba
declara hoy `go 1.27` y el nuestro declara `go 1.24`. Eso por si solo seria
discutible, porque un `go.mod` puede pedir mas de lo que el codigo usa. Medido:

```
$ go build ./pruebacabeza/        # ficheros de 05f79448fa77, 2026-08-21
pruebacabeza\pkcs7.go:10:2: package crypto/mldsa is not in std
```

`crypto/mldsa` (firmas ML-DSA, post-cuanticas) no existe en la biblioteca
estandar hasta Go 1.27. No es una etiqueta en un fichero, es un import que no
resuelve. **Y no se sube el minimo de Go por esto.**

**No hay ninguna version intermedia que sirva.** Entre 2025-07-29 y hoy, en
`digitorus/pkcs7`, los unicos commits que no son de la rama de Go 1.27 vienen de
la ancestria de `mozilla-services/pkcs7`, y esos no se pueden seleccionar como
version de este modulo:

```
$ go get github.com/digitorus/pkcs7@0b6e698e92f2bad3d8000ce746f598907cecf2e7
go: parsing go.mod:
        module declares its path as: go.mozilla.org/pkcs7
                but was required as: github.com/digitorus/pkcs7
```

Entraron en `master` de digitorus el 2026-08-21, en el mismo lote que el salto a
Go 1.27.

**Y lo que importa de verdad: no nos estamos dejando ningun arreglo del
parser.** `ber.go`, que es el fichero que come los bytes del tercero, es
funcionalmente identico en la version fijada y en la de cabeza. El diff entero
son cambios de estilo (`(int)(x)` a `int(x)`) y una reescritura equivalente de
`isIndefiniteTermination`, que pasa de `bytes.Index(ber[offset:], []byte{0,0}) == 0`
a `ber[offset] == 0 && ber[offset+1] == 0`; las dos dicen lo mismo y las dos van
detras de la misma guarda `len(ber)-offset < 2`. Comprobado con:

```
diff -u "$(go env GOMODCACHE)/github.com/digitorus/pkcs7@v0.0.0-20250729175123-57bd227bfa2f/ber.go" \
        "$(go env GOMODCACHE)/github.com/digitorus/pkcs7@v0.0.0-20260821105541-05f79448fa77/ber.go"
```

Lo que si trae la cabeza y aqui no esta: ML-DSA, ML-KEM (RFC 9629), aceptar
`id-ecPublicKey` en `SignerInfo.signatureAlgorithm` por compatibilidad con
Windows CNG, y **quitar del todo la verificacion de firmas DSA**. Ese ultimo es
un endurecimiento que a esta copia le vendria bien y que se puede portar el dia
que haga falta; no se ha portado hoy porque cambiar la politica de algoritmos no
es lo que pedia esta casilla.

## Que se ha quitado, y por que

No es estetica. `gosec` es puerta **bloqueante** de este repositorio y corre
sobre `./...`, asi que en cuanto el codigo esta aqui dentro es codigo nuestro
para el analizador. Una copia integra de los seis ficheros deja el CI asi:

```
Files : 9   Lines : 2639   Issues : 11
  encrypt.go:169  G405 Use of weak cryptographic primitive  des.NewCipher(key)
  decrypt.go:93   G405 Use of weak cryptographic primitive  des.NewCipher(key)
  decrypt.go:95   G405 Use of weak cryptographic primitive  des.NewTripleDESCipher(key)
  encrypt.go:7    G502 Blocklisted import crypto/des
  decrypt.go:8    G502 Blocklisted import crypto/des
  pkcs7.go:18     G505 Blocklisted import crypto/sha1
  ber.go:74,110,119, encrypt.go:394, decrypt.go:162   G115 integer overflow int -> byte
exit status 1
```

Las salidas de `crypto/des` son de codigo que este adaptador **no llama nunca**:
un sello de tiempo es un `SignedData`, no hay nada que cifrar ni que descifrar.
Silenciar la puerta con `-exclude-dir` habria dejado la unica criptografia
vendorizada del proyecto sin analisis estatico, que es exactamente la vigilancia
silenciosa que veniamos a quitar. Asi que se quita el codigo:

1. **`encrypt.go` y `decrypt.go` enteros.** Con ellos se van `crypto/des`,
   `crypto/aes`, el relleno PKCS#7 y el descifrado con clave RSA.
2. **`Parse` solo acepta `SignedData`.** `EnvelopedData` y `EncryptedData`
   devuelven ahora un error que dice por que, con `ErrUnsupportedContentType`
   como centinela. De paso, dos `asn1.Unmarshal` menos sobre bytes del tercero.
3. **`sign.go` recortado a las estructuras ASN.1.** De 500 lineas quedan las
   seis declaraciones que el camino de verificacion necesita para deserializar
   un `SignedData` y la funcion que vuelve a serializar los atributos
   autenticados. Con el resto se va `crypto/dsa`, que ademas esta obsoleto.
4. **En `pkcs7.go` se van los ayudantes de firmar** (`GetDigestOIDForSignatureAlgorithm`,
   `getOIDForEncryptionAlgorithm`) y la maquinaria de ordenar atributos para
   marshalling, que solo la usaba el firmante.

Lo que queda de `G115` y `G505` no se puede quitar sin cambiar el
comportamiento, asi que lleva `#nosec` **con el motivo escrito en la propia
linea**, uno por uno. Son cuatro y estan todos en el codificador de longitudes
DER y en el registro de SHA-1 para `crypto.SHA1.New()`.

## Que se ha cambiado, y esto es lo unico que toca una decision de seguridad

En `verify.go`:

1. **Fuera `Verify()`.** Aguas arriba es `VerifyWithChain(nil)` y su propio
   comentario dice "effectively disabling certificate verification when
   validating a signature". Un metodo que se llama Verify y no verifica la
   cadena no puede estar al alcance de la mano en el paquete que decide si un
   sello es de fiar.
2. **Fuera `VerifyWithChain` y `VerifyWithChainAtTime`.** Azucar sobre
   `VerifyWithOpts`, y la primera acepta un almacen `nil`, o sea el caso 1 otra
   vez por otra puerta.
3. **`VerifyWithOpts` exige `opts.CurrentTime`.** Aguas arriba, si viene a cero,
   comprueba la validez del certificado contra `time.Now()`. Esa rama rompe la
   promesa entera de este adaptador, que es que dados el mismo hash, el mismo
   token y las mismas anclas, dos maquinas cualesquiera dan el mismo veredicto
   hoy y dentro de cinco años. Un sello de 2026 verificado en 2031 con el reloj
   de 2031 sale invalido porque el certificado ya caduco, y eso es justo lo que
   un expediente no puede hacer. Ahora falta el instante y sale
   `ErrSinInstante`, con el arreglo escrito en el error.

4. **`VerifyWithOpts` exige `opts.Roots`**, y esto lo anadio la revision hostil
   del vendorizado porque **sin ello los puntos 1 y 2 eran teatro**. Quitar
   `Verify()` y `VerifyWithChain(nil)` cerro dos puertas y dejo abierta la
   tercera, que ademas era la unica que quedaba exportada: aguas arriba,
   `verifySignatureAtTime` encadena el certificado solo dentro de un
   `if opts.Roots != nil`. Con el almacen a nil se comprueba la firma del token
   y no se comprueba **de quien** es la clave que lo firma, que es literalmente
   lo que decia el comentario de aguas arriba citado para justificar el punto 1.

   Medido, no supuesto. Un token sellado por una CA que nadie ha declarado:

   ```
   VerifyWithOpts con Roots nil               -> <nil>
   VerifyWithOpts con Roots = anclas legitimas -> pkcs7: failed to verify certificate
                                                  chain: x509: certificate signed by
                                                  unknown authority
   ```

   O sea que el **valor cero** de `x509.VerifyOptions`, que es el que sale de
   escribir la estructura sin pensar, significaba "acepto cualquier sello".
   Ahora falta el almacen y sale `ErrSinAnclas`, simetrico con `ErrSinInstante`:
   los dos campos que deciden si un sello es de fiar son obligatorios los dos.

   El producto no era explotable por esto **hoy**, porque `Cadena.verificar`
   comprueba `c.Anclas == nil` antes de llamar, pero esa es una guarda del
   llamante y el llamante puede cambiar; la copia vendorizada es la que decide
   si un sello es de fiar y era ella la que aceptaba.

   Por que la afirmacion 4 del fuzzer no lo cazo, que es la parte que hay que
   recordar: afirmaba que ningun token verifica contra un almacen **vacio**
   (`x509.NewCertPool()`), que no es nil. Las dos formas de "sin raices" no son
   la misma y solo se recorria la inocua. Ahora se recorren las dos.

   El `if opts.Roots != nil` de `verifySignatureAtTime` se deja como esta,
   aunque ya no pueda ser falso: es texto de aguas arriba y quitarlo solo haria
   mas ruidoso el diff del deber heredado.

Con 1, 2 y 3 fuera desaparece la unica llamada a `time.Now()` de todo el codigo
vendorizado.

## El triaje del 26-08-2026: los 40 commits de aguas arriba

El canario de `vigilancia.yml` dijo, la primera vez que se ejecutó, que
`digitorus/pkcs7` va **40 commits por delante** de la versión fijada y que los
**cuatro** ficheros vendorizados han cambiado, `ber.go` incluido (+8/-6). Esto
es el triaje de esos 40, con el resultado escrito.

### El número miente, y esa es la primera conclusión

De los 40 commits, **20 tocan alguno de los cuatro ficheros vendorizados**, y de
esos 20 la mitad son *merges* que cuentan el mismo cambio dos veces.

Pero el problema no es el conteo: es que **contar commits no mide nada**. Buena
parte de esos 40 vienen de `d75a4a2076bb Merge mozilla-services/pkcs7 master
ancestry` (21-08-2026), que **reintroduce historia cuyo cambio de código ya
estaba** en la versión que tenemos fijada.

Lo que sí mide es comparar el **código**. Función a función, con `go/printer` y
sin comentarios, contra la punta de `master`:

| fichero | funciones vendorizadas | idénticas a la punta | distintas |
|---|---|---|---|
| `ber.go` | 9 | 7 | 2 |
| `pkcs7.go` | 4 | 3 | 1 |
| `verify.go` | 9 | 7 | 2 |
| `sign.go` | 1 | 1 | 0 |
| **total** | **23** | **18** | **5** |

### El arreglo de seguridad que parecía faltar, y no falta

`3562fcf934a0 "Fix out-of-bounds panic in ber2der on malformed BER input"`
(CWE-125 / CWE-193), fechado el **21-07-2026**, o sea **posterior** a la versión
fijada de esta copia (29-07-2025). Parecía un P0: el parser que come los bytes
del tercero, un pánico alcanzable, y nosotros del lado viejo.

**Medido, no supuesto: no lo es.** Las dos guardas que añade el parche
(`offset >= berLen` en la rama de etiqueta de número alto, y
`offset+numberOfBytes > berLen` antes de leer los octetos de longitud larga) **ya
están en esta copia**, y las cuatro entradas que aguas arriba dice que hacían
pánico devuelven error:

```
1F 80     -> ber2der: cannot move offset forward, end of ber data reached
1F 05     -> ber2der: cannot move offset forward, end of ber data reached
30 81     -> ber2der: cannot move offset forward, end of ber data reached
30 84 01  -> ber2der: cannot move offset forward, end of ber data reached
```

El commit viene de la ascendencia de mozilla-services. Los cuatro casos quedan
clavados en `regresion_arriba_test.go` para que la conclusión no haya que volver
a sacarla.

### Las cinco funciones que difieren, clasificadas

| función | qué cambia aguas arriba | clase | ¿se porta? |
|---|---|---|---|
| `ber.go: readObject` | `(int)(x)` → `int(x)`, cinco veces | **cosmético** | no hace falta |
| `ber.go: isIndefiniteTermination` | `bytes.Index(...) == 0` → comparación directa de dos bytes | **complejidad algorítmica**, semántica idéntica | **no hoy**, ver abajo |
| `pkcs7.go: Parse` | añade las ramas ML-KEM (RFC 9629) | no aplica: aquí no se descifra nada | no |
| `verify.go: VerifyWithOpts` | añade ML-DSA | **aguas arriba SIGUE sin arreglar lo que arreglamos** | no |
| `verify.go: getSignatureAlgorithm` | (a) ML-DSA, (b) `id-ecPublicKey`, (c) DSA fuera | mixto | **(c) sí, recorte 6** |

**Sobre `VerifyWithOpts`, que conviene decir en voz alta**: la punta de `master`
sigue teniendo hoy, textual, `if opts.KeyUsages == nil { opts.KeyUsages =
[]x509.ExtKeyUsage{x509.ExtKeyUsageAny} }` y sigue encadenando el certificado
sólo dentro de un `if opts.Roots != nil`. Los recortes 3, 4 y 5 no son un retraso
respecto de aguas arriba: son cosa nuestra y no existen allí.

### Lo que se porta: recorte 6, DSA fuera

De `1390b412643f "Remove DSA signature verification functionality"`. Va en la
misma dirección que los recortes 3, 4 y 5: **más restrictivo**.

No es un arreglo de vulnerabilidad. `crypto/x509` marca `DSAWithSHA1` y
`DSAWithSHA256` como `// Unsupported.`, así que el sello acababa rechazado
igualmente, pero con `x509: cannot verify signature: algorithm unimplemented`,
que sale de dentro de `x509` y no dice ni que el algoritmo era DSA ni qué hacer.
**Un verificador que rechaza por el motivo equivocado obliga a quien recibe el
error a averiguarlo por su cuenta**, y este repositorio tiene escrito que los
errores llevan causa y arreglo.

Con su control negativo: RSA y ECDSA siguen verificándose.

### Lo que NO se porta, y por qué cada uno

- **ML-DSA y ML-KEM** (`4c6eb7c03148`, `ce121b2797a5`). Exigen **Go 1.27**, porque
  importan `crypto/mldsa`. Es el mismo motivo por el que la versión fijada es la
  que es, y no ha cambiado.
- **`id-ecPublicKey` en `SignerInfo.signatureAlgorithm`** (`cda87bbfe2e9`,
  `cdd9cd6ccb2a`). Es una **ampliación de compatibilidad**: acepta una forma que
  RFC 5753 no exige, que emiten Windows CNG y versiones históricas de NSS. Aguas
  arriba argumenta que es segura, y probablemente lo sea. **No se porta porque
  ENSANCHA lo que verifica**, y ensanchar una frontera de confianza es la
  dirección que hay que justificar, no la que se hace por defecto.

  **Cuándo hay que volver aquí, dicho concreto para que no se pierda**: si una
  autoridad de sellado de la lista de anclas emite el OID de clave pública EC en
  `signatureAlgorithm`, su sello **fallará** con
  `pkcs7: unsupported algorithm`. Ese día se porta, con su caso de prueba y su
  token real. Hasta entonces, se falla en cerrado y se sabe por qué.
- **El "excess walk"** (`b023b759d93e`, `6684f57921be`). Ver abajo.

### El "excess walk", medido antes de decidir

`isIndefiniteTermination` hace `bytes.Index(ber[offset:], []byte{0x0, 0x0}) == 0`,
o sea **recorre todo lo que queda del búfer** para responder a una pregunta sobre
dos bytes. Aguas arriba lo cambió por la comparación directa. Es **semánticamente
idéntico** (`Index == 0` es exactamente "empieza por"), y la guarda
`len(ber)-offset < 2` que ya hay delante hace segura la indexación.

Pero se llama una vez por objeto dentro de una secuencia de longitud indefinida,
así que el conjunto es **cuadrático sobre entrada que elige el atacante**.
Medido, con una secuencia indefinida llena de objetos vacíos:

```
  4096 bytes ->   573 µs
  8192 bytes ->  1,03 ms
 16384 bytes ->  3,63 ms
 32768 bytes -> 15,76 ms
```

Cuatro veces por cada duplicación, que es lo que se espera de un cuadrático.

**Y aquí se cierra un lazo**: el tope `maxToken` = 32 KiB, que se puso por la
amplificación de `ber2der` (×482 medido), es también lo que acota esto. Sin el
tope, un token de 1 MB serían ~16 segundos de CPU por petición. Con él, 16 ms.

**No se porta hoy**, y el motivo no es el coste: es que `isIndefiniteTermination`
está en el **camino de derivación del contenido**, y portarla abriría un recorte
declarado justo ahí, que es donde la puerta de los dos parsers existe para que no
haya ninguno. Se porta el día que se mueva la versión fijada, y entonces se
mueven las dos a la vez.

### La coartada del pendiente 53, desmontada

El apunte decía: *"hoy no es explotable, y está medido: `ber.go` es byte a byte
el mismo en las dos copias"*. Tres cosas de este triaje la desmontan.

**1. La verificación del veredicto no es la nuestra, y es la permisiva.**
`timestamp.Parse`, de donde salen `HashedMessage`, `HashAlgorithm` y `Time`, hace
esto:

```go
p7, err := pkcs7.Parse(bytes)          // el pkcs7 de AGUAS ARRIBA
if len(p7.Certificates) > 0 {
    if err = p7.Verify(); err != nil { // <-- Verify(), no VerifyWithOpts
        return nil, err
    }
}
```

`Verify()` es **exactamente la función que el recorte 1 quitó de esta copia**,
porque su propio comentario aguas arriba dice que inicializa un almacén de
confianza vacío *"effectively disabling certificate verification"*. O sea que no
es sólo que haya dos parsers: **el parser del que sale el veredicto ejecuta una
verificación que esta copia se niega a exponer**.

**2. La distancia no se puede cerrar.** La punta exige Go 1.27 y entre medias no
hay versión intermedia seleccionable. La versión fijada **no se puede mover**, así
que la distancia entre las dos copias sólo crece, y cada arreglo de aguas arriba
la ensancha un poco más.

**3. La coartada tiene fecha de caducidad, y ahora se ve.** Hoy el camino del
contenido sigue siendo el mismo código, y hay una puerta que lo comprueba función
a función. Deja de serlo **el día que se porte algo a `ber.go` o se mueva la
versión fijada**, y las dos cosas van a pasar.

**La conclusión, que cambia el arreglo:** no es "portar con cuidado". Es **dejar
de derivar el veredicto de un parser distinto del que comprueba la firma**, que es
lo que la etapa 8 hace al quitarse `timestamp` de encima. El pendiente 53 sube de
**P2 a P1**.

---

## El deber heredado: como se sigue aguas arriba

Esto es lo que hemos comprado al vendorizar, y una intencion no sirve. **Se hace
al cerrar cada etapa, junto al resto de puertas**, que es la regla que ya tiene
`DEPENDENCIAS.md` para los modulos sin semver. Cuesta un minuto:

```bash
# 1. Traer la historia de aguas arriba (sin blobs, es rapido)
git clone --filter=blob:none https://github.com/digitorus/pkcs7 /tmp/pkcs7-arriba
cd /tmp/pkcs7-arriba

# 2. Que ha cambiado desde nuestro commit, SOLO en los cuatro ficheros que
#    tenemos vendorizados. Si no imprime nada, no hay nada que hacer.
git log --oneline 57bd227bfa2f..origin/master -- ber.go pkcs7.go verify.go sign.go

# 3. El diff, que es lo que hay que leer y portar a mano
git diff 57bd227bfa2f..origin/master -- ber.go pkcs7.go verify.go sign.go

# 4. Si se porta algo: actualizar la tabla de arriba con los sha256 nuevos
cd <repo>/adaptadores/tsa/internal/pkcs7
sha256sum ber.go pkcs7.go verify.go sign.go LICENSE
go test ./... -count=1
```

Tres cosas que hay que mirar en el paso 3 y que no saltan solas:

- **Un arreglo en `ber.go` es urgente.** Es el unico fichero que parsea bytes
  que no controlamos. Los otros tres trabajan sobre estructuras ya
  deserializadas.
- **Un cambio en `getSignatureAlgorithm` o en `verifySignatureAtTime` es una
  decision de politica de algoritmos**, no una correccion mecanica. Se lee
  entero antes de portarlo.
- **Un arreglo puede llegar en un fichero que aqui no esta.** Si el commit toca
  `encrypt.go` o `decrypt.go`, no nos afecta; si toca `pkcs7.go` en la parte de
  firmar, tampoco. Comprobarlo, no suponerlo.

Y el aviso que sigue vigente: **dependabot no va a decir nada de esto, ni antes
ni ahora.** No hay semver que comparar.

## El fuzzing, y que afirma

`fuzz_test.go`. Corre en cada `go test` con su corpus semilla, y el motor
completo se lanza a mano con:

```bash
go test ./adaptadores/tsa/internal/pkcs7/ -run FuzzParseNoSeFiaDeNingunToken \
  -fuzz FuzzParseNoSeFiaDeNingunToken -fuzztime 5m
```

El corpus semilla lleva un token de una TSA de verdad, el del expediente de
demostracion (`testdata/token-real.der`), y prefijos suyos cortados por sitios
incomodos. Sin eso el fuzzer se pasa el rato rebotando en "input data is empty"
y no llega nunca a la parte interesante, que es la que se recorre cuando la
estructura es casi valida.

Lo que se afirma sobre CUALQUIER cadena de bytes no es solo que no reviente. Un
parser que no entra en panico y devuelve un token dado por bueno es peor que uno
que revienta:

1. no entra en panico;
2. `Parse` devuelve token o error, nunca las dos cosas ni ninguna;
3. es determinista: dos parseos de los mismos bytes dan el mismo veredicto;
4. **ningun token verifica sin raices que lo respalden**, y "sin raices" tiene
   DOS formas que no son la misma: el almacen vacio (`x509.NewCertPool()`) y el
   almacen **nil**. Esta es la de verdad: si el fuzzer encontrara una cadena que
   sale valida sin una sola raiz detras, el anclaje del expediente no probaria
   nada. La segunda forma la anadio la revision hostil y era justo la que
   verificaba: ver el recorte 4;
5. **sin instante no verifica nada**, ni el token bueno, que es lo que hace que
   el veredicto no dependa del reloj de quien verifica;
6. la transcodificacion de BER a DER es **idempotente**.

La que NO esta, y el hueco es el hallazgo de este trabajo: que no amplifique. Se
intento afirmar x2, x4 y x64, y el fuzzer tumbo las tres. Ver la seccion de
abajo.

## Lo que esto NO arregla

Honestidad sobre el alcance, porque una copia vendorizada da una sensacion de
control que no siempre se corresponde:

- **`github.com/digitorus/timestamp` sigue siendo dependencia externa, y sigue
  importando el `pkcs7` de aguas arriba.** `timestamp.Parse`, que
  `VerificarOffline` llama antes que a esta copia, parsea los mismos bytes con
  la libreria de fuera. La frontera de confianza no es solo nuestra. Por eso
  `go.mod` conserva la linea `require github.com/digitorus/pkcs7 ... // indirect`
  fijada a la version de 2025-07-29: sin ella, MVS elegiria la que pide el `go.mod` de
  `timestamp`, que es la de 2023, la del panico. Lo vigila
  `TestElPkcs7TransitivoNoEsElQueRevienta`.
- **`govulncheck` no ve este directorio.** Empareja vulnerabilidades por ruta de
  modulo, y esto ya no es un modulo. Sigue viendo el `pkcs7` transitivo de
  `timestamp`, que es la misma version, asi que en la practica el aviso llegaria
  igual mientras las dos no se separen.
- **`ber2der` amplifica, y mucho.** Lo encontro el fuzzing de este mismo
  directorio, que es la primera vez que alguien lo mira aqui. Un objeto
  construido de longitud definida devuelve la longitud DECLARADA y no la que sus
  hijos consumieron de verdad, asi que los bytes que un hijo se traga de mas los
  vuelve a leer el abuelo como su siguiente hermano y salen dos veces. Medido:
  331 bytes producen 159.693 (x482), 631 producen 1.197.909 (x1.898), 931
  producen 2.542.305 (x2.731); la razon se aplana hacia x4.000. Esta en la
  version fijada y tambien en la de cabeza. **No se ha arreglado aqui**, y por
  una razon concreta: `Cadena.verificar` llama PRIMERO a `timestamp.Parse`, que
  parsea el mismo token con el `pkcs7` de aguas arriba, asi que una guarda
  dentro de esta copia no llegaria a ejecutarse. La defensa que si funciona esta
  en el llamante: `VerificarOffline` e `Instante` rechazan un token de mas de 32
  KiB antes de parsearlo. El numero de la amplificacion lo clava por los dos
  lados `TestBer2derAmplificaYPorEsoElTokenLlevaTope`, para que ni empeore ni
  mejore en silencio. Queda por reportar aguas arriba.
- **La firma del token se sigue aceptando con SHA-1** si el token lo declara
  (`getSignatureAlgorithm` mapea a `x509.SHA1WithRSA` y `CheckSignature` lo
  admite). Ninguna TSA seria firma asi hoy, y `VerificarOffline` ya exige
  SHA-256 para el `messageImprint`, que es lo que ata el sello al contenido.
  Queda anotado como P2, no arreglado: cambiar la politica de algoritmos de la
  copia es una decision aparte de vendorizarla.
