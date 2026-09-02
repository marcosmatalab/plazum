# Pendientes: el registro de P1 y P2

Los hallazgos que no bloquean la casilla en la que salieron, para que no se
pierdan en el cuerpo de un commit.

Clasificacion, la del protocolo de las tres pasadas (`CLAUDE.md`):

- **P0** bloquea la casilla. No entra aqui: se arregla antes de marcar.
- **P1** entra en la etapa. Se arregla dentro de la etapa en curso.
- **P2** a la lista. Se arregla cuando toque o se decide que no.

Cuando algo se cierra, se borra de aqui y consta en el commit que lo cerro.

---

## La otra familia: "sin confiar en el emisor"

Trece instancias. Cada una es un sitio donde el verificador se creia algo que el
EMISOR escribe. Las doce primeras estan en el historial de la etapa 1; la
decimotercera se cerro el 25-08-2026 y era la mas barata de explotar de todas:

> **No hacia falta borrar nada para blanquear un incumplimiento.** El apartado 5b
> del verificador comprobaba `observaciones -> cadena` (que cada observacion
> declarada estuviera anclada) y NUNCA la direccion contraria. Un emisor con un
> control en `fail_en_plazo` no tenia que tocar la cadena, ni destruir una clave,
> ni poner una lapida, ni declarar una supresion: le bastaba con QUITAR esa
> observacion de la lista, dejando su entrada y su clave publicadas e intactas.
> El verificador recalculaba con las que quedaban, salia `pass`, y devolvia
> `Valido=true` con cero discrepancias.

Lo que lo hace la peor de la familia: **toda la maquinaria de borrado legal
estaba defendiendo una puerta con la pared abierta al lado**. Lapidas, keystore,
destruccion de clave, declaracion de supresion y forzado a obsoleto, y ninguna
hacia falta.

Es PREEXISTENTE: reproducido sobre la base, o sea que llevaba ahi desde que
existe el verificador y trece rondas de revision hostil no lo vieron. Lo encontro
una pasada ADVERSARIA sobre el arreglo de otra cosa, buscando refutar una
propiedad que el frente daba por buena.

**El patron, para la decimocuarta**: cuando una comprobacion recorre una lista
para contrastarla con otra, preguntarse SIEMPRE si la direccion contraria tambien
se recorre. La que falta es la que el emisor usa.

## La familia: guardas que no guardaban

**Dieciséis en dos semanas**, y las dieciséis del mismo tipo. La decimoquinta abrió subfamilia propia, al final de esta sección. No son casos borde: son la
forma por defecto en que una comprobacion deja de comprobar sin que nadie se
entere, porque **el sintoma de una guarda rota es exactamente el mismo que el de
una guarda que funciona: verde**.

Con la cuarta y la quinta el patron cambia de sitio y conviene decirlo: las tres
primeras estaban en codigo Go y se cazaron mutando. Las siguientes estan en
**shell dentro de un workflow**, que es tierra sin compilador, sin `go vet` y sin
nadie que lea el rojo si el rojo lleva semanas puesto.

Y la sexta es la mas incomoda de todas, porque esta **dentro del propio aparato
que se construyo para cerrar la tercera**.

| # | La guarda | Que dejaba pasar | Cuanto llevaba asi | Como se cazo |
|---|---|---|---|---|
| 1 | El limite de texto de un paquete referencial | Una `"clase": 9` fuera de rango caia en el `default` del switch y se saltaba el limite entero. La frontera legal, esquivada escribiendo un numero | desde que existia el linter | midiendo los dos casos al lado (clase 2 -> 1 error, clase 9 -> 0 errores) en vez de fiarse de un `contains` |
| 2 | El test AST de "ninguna norma cableada" | Excluia TODOS los `_test.go`. Ocho ficheros de `nucleo/` con normas cableadas, y las reglas de aplicabilidad del ENS escritas en Go dentro de un `progENS` | meses | ampliando el alcance y viendo que se ponia rojo por ocho sitios a la vez |
| 3 | Los pasos de CI con `go test -run` | `go test -run TestQueYaNoSeLlamaAsi` imprime "no tests to run" y **sale con 0**. Un renombrado dejaba la puerta verde sin comprobar nada. Y `go test ./glob/sin/tests/...` hace lo mismo con "no test files" | desconocido | mutando el patron a uno que no casa y viendo que la puerta seguia verde |
| 4 | El job de axe-core entero | La deteccion de la superficie web preguntaba `./plazum 2>&1 \| grep -qw serve`, y `plazum serve` **no estaba en la lista de uso** que imprime el binario. El job caia por "el producto no sabe servir pantallas": **rojo permanente**. No auditaba HTML estatico, no auditaba NADA. Con el, el presupuesto de arranque cronometraba `cobertura paquetes` en vez de `serve`, y el de RAM bajo peticiones no se ejecuto jamas | desde que existia el job | pidiendo que SIRVA en vez de preguntar por una cadena de ayuda, y quitando el camino de respaldo |
| 5 | Un paso de `etapa2-ttfv.yml` | Un bloque abria con `{` y cerraba con `fi`. Error de sintaxis de bash, asi que el paso que comprueba que `doctor` dice como se arregla lo que senala, y que el demo se deshace entero, **nunca se ejecuto** | desconocido | `bash -n` sobre los 32 bloques `run:` de todos los workflows (`TestTodoPasoDeCIEsShellQueBashSabeParsear`) |
| 6 | `.github/puerta.sh` entero | GitHub ejecuta los pasos `bash` con `-e` puesto, y `set -uo pipefail` no lo apaga. Con -e, la linea `salida=$(go test ...)` mata el shell EN EL ACTO: la puerta se ponia roja imprimiendo una sola linea, la del `::group::`, y **el aparato que explica que ha cazado no se ejecutaba nunca**. Todo el trabajo de la tercera, invisible justo cuando hacia falta | desde que se escribio, hace dos dias | un job de windows-latest fallo en `main` y no dejo ni una pista de por que |
| 7 | Las cuatro comparaciones byte a byte del repositorio | No habia `.gitattributes`. El runner de Windows trae `core.autocrlf=true` y convierte a CRLF al hacer checkout; la maquina de desarrollo lo tiene en `input` y deja LF. El generador escribe LF, asi que `TestElDemoPublicadoSaleDeEsteGenerador` comparaba dos ficheros que se diferenciaban en un byte que nadie habia escrito. **Verde en la maquina del autor, rojo en la de cualquier otro.** Y `paquetes/iso27001/paquete.json` llevaba commiteado CRLF de punta a punta, 2206 saltos de linea | desde siempre | la puerta nueva lo caza sola: se escribio y salio roja en el primer intento, senalando el iso27001 |
| 8 | El caso dorado de `nucleo/pantalla` | Al mutar la derivacion, el dorado parecia no inmutarse. No era verdad: el control negativo indexaba `Fuentes[0]` con longitud 0, entraba en **panico**, y un panico aborta el binario de test ENTERO. El dorado no llegaba a ejecutarse y su verde era un verde que no existia | lo que durase esa mutacion | mirando por que una mutacion que TENIA que romper el dorado no lo rompia |
| 9 | La guarda del borrado legal del export a SIEM | La comprobacion que impide que lo suprimido reaparezca en un fichero de texto plano **casaba por INDICE**, y el indice no lo firma nadie. Reordenar o insertar una entrada mueve el emparejamiento entero. Es el ataque 13 otra vez, en codigo escrito el mismo dia | horas | el refutador del propio frente, buscando por que campo casaba |
| 10 | `TestSobreUnaInstalacionSanaNoSeInventaProblemas` | Cableaba `2026-08-26 09:00 UTC` como instante de prueba y lo comparaba con la fecha de un `t.TempDir()` creado **ahora de verdad**. Escrito la tarde del 25 con el instante en el futuro, verde. A las 09:00 UTC del 26, **main en ROJO sin que nadie tocara una linea**, y para siempre | 14 horas, con la mecha encendida desde el primer minuto | amanecio rojo. Ninguna puerta lo caza porque CI solo corre cuando alguien empuja |
| 11 | La puerta de CI del export a SIEM, mientras se escribia | Para probar contra el binario que una entrada con lapida no filtra su contenido, el paso fabricaba un expediente con lapida inyectando `"lapidas": [...]` al principio del objeto `cadena` con `sed`. Pero el expediente **ya trae** `"lapidas": null` (el campo no lleva `omitempty`), asi que el fichero quedaba con DOS claves iguales y en JSON gana la ultima: el expediente de prueba era identico al original. Y la guarda de la propia inyeccion preguntaba `grep -q '"lapidas"'`, que casaba con la que ya estaba | lo que duro escribirla | el paso se ejecuto en un shell con las banderas de GitHub antes de commitearlo, y salio rojo: el centinela aparecia en el fichero. Arreglo: sustituir la linea existente en vez de anadir una clave, y comprobar la inyeccion buscando `entrada_borrada`, que es lo inyectado, y no la clave, que ya existia |
| 12 | La cabecera del cribador de marca | Decia `oficina EUIPO` **cableado**, mirara donde mirara. Con `-oficina ES` la consulta iba de verdad a la OEPM y los registros que salian eran espanoles, pero el rotulo seguia diciendo EUIPO. En una herramienta cuyo unico producto es la PRUEBA, una cabecera que miente sobre el registro consultado deja la prueba sin valor: quien la lea dentro de un ano no sabe donde se busco ni, por tanto, que quedo sin mirar | desde que se escribio la herramienta | pegando la salida en `docs/marca.md` como prueba y leyendola |
| 13 | `adaptadores/tsa`, el certificado de la TSA falsa | Valia `instanteSello +- 24 h`, o sea del 24 al **26-08-2026 a las 10:00 UTC**. A las 10:00:01 de ese dia la libreria de CMS empezo a rechazar la firma con "signing time is outside of certificate validity" y main se puso rojo otra vez, seis horas despues de arreglar la decima. La hora de la firma la pone el reloj de la maquina (atributo `signingTime` del CMS), no el instante cableado, que solo va dentro del TSTInfo | desde que se escribio | trabajando en otra cosa, seis horas despues de que estallara |
| 14 | El invariante de amplificacion del fuzzing del pkcs7 vendorizado | La afirmacion sobre `ber2der` estaba escrita DESPUES del `if err != nil { return }` que abandona cuando `Parse` falla. Una mutacion del codificador de longitudes hizo que `Parse` fallara sobre TODAS las semillas, asi que el bloque no se ejecuto ni una vez y el fuzz salio verde: la mutacion parecia no cazada | lo que duro escribirlo | mutando `lengthLength` para que devolviera siempre 4 y preguntandose por que la unica cosa que tenia que romper era justo la que no rompia |
| 15 | Los recortes 1 y 2 del pkcs7 vendorizado | Se quitaron `Verify()` y `VerifyWithChain(nil)` **porque no verifican la cadena**, y se dejo `VerifyWithOpts`, que es la unica que quedaba exportada y que hace exactamente lo mismo: `verifySignatureAtTime` encadena el certificado solo dentro de un `if opts.Roots != nil`. El **valor cero** de `x509.VerifyOptions` era "acepto cualquier sello". Un token de una CA que nadie ha declarado salia `<nil>`. Se cerraron dos puertas y se dejo abierta la tercera, que era la que se usaba | desde que se vendorizo | eligiendo la propiedad que el trabajo daba por buena ("aqui no se puede verificar sin comprobar de quien es la clave") e intentando tumbarla, en vez de leer el diff. El fuzzer ya afirmaba "ningun token verifica contra un almacen vacio" y usaba `x509.NewCertPool()`: recorria la direccion inocua de "sin raices" y no la que verificaba |
| 16 | La puerta que llegaba a lo que vigila **por un camino que el producto ya no usa** | Nada, todavia: se caza al cambiar el producto. `TestElPkcs7TransitivoNoEsElQueRevienta` comprobaba el pkcs7 de aguas arriba llamando a `timestamp.Parse`, que era el camino del producto. El dia que el producto dejo de usar ese camino, la puerta siguio verde midiendo **el camino y no lo vigilado** | Un dia | Al quitar `timestamp`: la puerta seguia pasando sobre una libreria que ya no decidia nada |

**Lo que tienen en común**, y es lo que hay que buscar en la siguiente:

- **El alcance, no la logica.** Ninguna de las tres tenia mal la comprobacion.
  Las tres miraban al sitio equivocado, o a menos sitios de los que decian.
- **Se cazan por mutacion y solo por mutacion.** Leer el codigo no las encuentra:
  el codigo parece correcto porque *es* correcto sobre lo que mira.
- **La mutacion tiene que ir FUERA de lo que el propio test eligio.** Mutar
  dentro de la lista que el test ya conoce es cazarse a uno mismo. Paso otra vez
  con la lista de rutas de las pantallas: la mutacion anadia un POST a una ruta
  que ya estaba en la lista del test.

**La leccion de la undecima.** El que fabrica el escenario de una puerta puede
fabricarlo mal, y entonces la puerta mide el vacio. Aqui el escenario era un
expediente con lapida construido con `sed`, y la trampa fue de formato: el
fichero **ya tenia** la clave que se pretendia anadir, con la lista vacia. En
JSON, dos claves iguales las resuelve la ultima, asi que la inyeccion se
desintegraba sin dar ningun error. Y la guarda que se puso contra ese mismo
riesgo preguntaba por la existencia de la clave, que ya existia.

La regla que sale: **al comprobar que una mutacion o una inyeccion se aplico, hay
que buscar lo INYECTADO y no el sitio donde se inyecta.** Es la misma familia que
"un `sed` que no casa da verde y parece un hallazgo", una vuelta mas adentro.

**La leccion de la novena, y es la que une a toda la familia.** Nueve de diez
son emparejamientos: dos conjuntos que hay que casar y una eleccion de POR QUE
CAMPO se casan. Cuando ese campo es el indice, la posicion o el orden, la guarda
no guarda, porque nadie firma el orden. La regla esta ahora en `CLAUDE.md` como
invariante de diseño: **toda comprobacion que empareje dos conjuntos lo hace por
una identidad que esta dentro de lo firmado, nunca por indice, posicion ni
orden.** Y no es deuda heredada: la novena aparecio en codigo escrito ese mismo
dia. Es un patron generativo, o sea que hay que preguntarlo en cada
emparejamiento nuevo, no buscarlo en el codigo viejo.

**La leccion de la decima y la decimotercera, que son la misma y por eso se
cuenta dos veces.** Un verde puede CADUCAR, y no es un caso raro: en un solo dia
estallaron DOS, con seis horas de diferencia, y la segunda estaba escrita hacia
semanas. El patron, dicho para poder buscarlo: **cualquier cosa que compare un
valor cableado con algo que ocurre en tiempo real tiene la mecha encendida desde
el minuto en que se escribe.** Un `t.TempDir()` contra un instante fijo. Un
certificado de prueba contra la hora de la firma. Una vigencia de paquete contra
la fecha del expediente. Una raiz de TSA contra su caducidad.

Y el arreglo NUNCA es alargar el plazo o mover el instante: eso aplaza la bomba y
se la deja al siguiente. Es derivar el lado que puede moverse del lado que no, o
fijar el que puede moverse.

**La leccion de la decima.** Un verde puede CADUCAR. No basta con que no dependa
de la maquina (septima), tampoco puede depender del reloj de pared. Un
certificado de prueba, una vigencia de paquete, una raiz de TSA, un plazo del
corpus: todos son bombas con la mecha encendida desde el minuto en que se
escriben. Y la puerta no es un test, es un HORARIO: `ci.yml` corre ahora todos
los dias a las 06:17 UTC, y ahi un verde caducado se cae **solo**, sin nada mas
en el diff. Sin eso, el rojo espera al siguiente empujon y aparece mezclado con
un cambio que no tiene nada que ver, que es la forma mas cara de encontrarlo.

**La leccion de la octava, y es una categoria nueva.** Las siete primeras
fallaban por ALCANCE: la comprobacion miraba al sitio equivocado. La octava
falla porque **la comprobacion no llega a correr**. Un panico en cualquier test
del paquete se lleva por delante a todos los demas, y `go test` lo cuenta como
un fallo, no como veinte tests que no se ejecutaron. Al mutar, eso se lee como
"esta mutacion rompio una cosa" cuando en realidad tapo el resultado de todas
las demas.

Lo que hay que hacer con ella: **cuando una mutacion rompa MENOS de lo que
esperabas, mirar si ha roto de mas.** Un panico, un `t.Fatal` en un `TestMain`,
un `os.Exit`. Y no indexar nunca en un control negativo sin comprobar la
longitud, que es lo que convierte un fallo legible en un panico.

**La leccion de la novena, que es la octava vista desde el otro lado.** La octava
no corria porque algo la mataba antes. La novena no corria porque estaba
**detras de un `return`**: en un cuerpo de fuzz con varias afirmaciones, las que
van despues de un abandono temprano solo se comprueban sobre el subconjunto de
entradas que llega hasta alli, y ese subconjunto lo decide OTRA funcion. La regla
que sale de aqui: **cada afirmacion se coloca lo mas cerca posible de la funcion
sobre la que afirma, y antes de cualquier abandono que dependa de una funcion
distinta.** Y la forma de cazarla es la misma que la de la octava, mirando una
mutacion que rompe menos de lo que deberia; solo que aqui rompia mucho, en otros
sitios, y precisamente por eso era facil darla por cazada.

**La leccion de la septima.** Un verde que depende de la configuracion de la
maquina no es un verde, es una coincidencia. Y la forma en que se manifiesta es
la mas cara de todas: **funciona en la maquina del que lo escribio**. El
`nucleo/pantalla` ya llevaba un ReplaceAll de CRLF a LF
en su comparador de dorados, o sea que alguien SINTIO este problema y lo parcheo
en un sitio en vez de arreglarlo en la raiz. Un parche local a un problema global
es la forma en que un problema global se vuelve invisible.

**La leccion de la sexta, y es de las que valen para todo.** Una puerta se
demuestra **en el shell en el que corre**, no en el del que la escribe. Las cinco
formas de fallo de `puerta.sh` se demostraron a mano, una por una, en un shell
interactivo sin `-e`. La demostracion fue real y aun asi no cubrio el modo en que
el fichero se ejecuta de verdad. Vale igual para un `_test.go` que se prueba con
un `-run` a mano y luego corre dentro de la suite entera, y para un script que se
prueba con `bash x.sh` y luego se ejecuta con `source`.

**La leccion nueva, de la cuarta y la quinta.** Una puerta que depende de
detectar algo con un `grep` sobre la salida de otro programa tiene DOS formas de
fallar, y solo una es la que se vigila. Se vigila que el programa este mal; no se
vigila que el `grep` haya dejado de casar. **Si una puerta tiene camino de
respaldo, el camino de respaldo se convierte en la puerta**, y mide otra cosa sin
decirlo. La forma correcta es exigir la capacidad, no preguntar por una cadena
que la anuncie.

Y el corolario, que es lo que hay que buscar en la sexta: **un rojo permanente es
tan invisible como un verde falso.** Nadie mira un job que lleva semanas rojo, y
un job que lleva semanas rojo no esta midiendo nada.

**Lo que se hizo con la tercera**, y con la quinta:
convertir la convencion en una puerta. `.github/puerta.sh` cuenta los casos
ejecutados y exige un minimo declarado, `puertas_test.go` prohibe que un workflow
invoque `go test` directamente, y la regla queda en `CLAUDE.md`: una puerta que
nunca se ha visto fallar no es una puerta.

### Subfamilia: vendorizar sin mirar quién más lo arrastra

**Vendorizar una librería que otra dependencia también importa no quita el código de en medio: añade una copia.**

Es la forma general del hallazgo 53, y se anota como familia porque la próxima vez que se vendorice algo hay que releerla **antes** de empezar y no después.

**Lo que pasó.** `pkcs7` se vendorizó en la etapa 2 con un motivo correcto: es la única criptografía del proyecto que trabaja sobre bytes de un tercero y su versión fijada envejeció tres años sin que nadie mirara. Lo que no se miró fue **quién más la importaba**. La respuesta era `github.com/digitorus/timestamp`, que este mismo adaptador usa, así que el resultado de vendorizar no fue una copia sino **dos**:

- la vendorizada, que comprueba la firma;
- la de aguas arriba, dentro de `timestamp`, que era **la que decidía el veredicto**.

Y las dos en el binario a la vez.

**Las tres preguntas que hay que hacerse antes de vendorizar algo**, en este orden:

1. **¿Quién más lo arrastra?** `go mod graph | grep <modulo>`. Si alguien más lo importa, vendorizar no lo saca: lo duplica.
2. **¿Cuál de las dos copias va a decidir?** Si la respuesta no es "la nuestra, siempre", vendorizar no ha arreglado nada y ha añadido un deber heredado.
3. **¿Se puede encoger en vez de copiar?** Es la pregunta que faltó. El `TSTInfo` son once campos de ASN.1 y unas doscientas líneas con su fuzzing; la copia entera de otra librería son dos `LEEME.md`, dos tablas de procedencia y dos canarios para siempre.

**Y el corolario incómodo**: vendorizar se siente como reducir dependencias y a veces es lo contrario. La cuenta que importa no es cuántas líneas hay en `go.mod`, es **cuántas copias del mismo parser hay en el binario y cuál de ellas decide**.

**El desenlace, el mismo día**: la respuesta a la tercera pregunta era que sí. El `TSTInfo` son once campos y unas 250 líneas con su fuzzing; el `TimeStampReq`, seis campos y cuarenta líneas. Frente a eso, la copia entera de otra librería habría sido dos `LEEME.md`, dos tablas de procedencia y dos canarios para siempre. **De dos dependencias externas a cero**, y el paso que lo cerró fue el más pequeño de los dos.

### Subfamilia: las dos formas de la nada

El hallazgo 15 (`pkcs7.VerifyWithOpts` encadenaba sólo dentro de un `if opts.Roots != nil`) se anotó primero como "dirección contraria otra vez". Se quedaba corto. Lo que hay debajo es una regla del lenguaje que genera fallos por sí sola, y por eso tiene sección propia y no un número en la tabla de arriba.

**En Go, el valor cero de una estructura de opciones suele significar *permisivo*, y el vacío-pero-presente significa *restrictivo*.**

| forma | qué significa en `crypto/x509` |
|---|---|
| `Roots: nil` | encadenar contra el almacén del sistema, o —en `pkcs7`— **no encadenar** |
| `Roots: x509.NewCertPool()` | no confiar en nadie |
| `KeyUsages: nil` | en `crypto/x509`, `ServerAuth`; en el `pkcs7` vendorizado, **`ExtKeyUsageAny`** |
| `[]string(nil)` como lista blanca | "sin restricción", casi siempre |

**La peligrosa es siempre la `nil`, porque es la que sale por olvidarse.** Nadie escribe `x509.NewCertPool()` sin querer; el valor cero aparece solo, escribiendo la estructura sin pensar en ese campo.

Y de ahí sale el punto ciego del test, que es la mitad que importa: la afirmación 4 del fuzzer de `pkcs7` decía *"ningún token verifica contra un almacén vacío"* y usaba `x509.NewCertPool()`. **Recorría la inocua.** Un test de ausencia que sólo mira una de las dos formas deja la otra abierta y se lee exactamente igual de verde.

**La regla, ya en `CLAUDE.md` como invariante 8**: en una frontera de confianza el valor cero tiene que ser el restrictivo, o estar prohibido explícitamente con centinela; y todo test de ausencia recorre `nil` **y** vacío-presente.

**Y una vuelta de tuerca del 28-08-2026, del mismo tipo pero un piso más arriba: un comentario que afirma protección es una CLAIM, y las claims se verifican.** El test `TestUnaVigenciaAbiertaNoCesaNunca` decía en su cabecera que protegía el booleano de `FinDeVigencia` (el invariante 8 otra vez: devolver el cero de `time.Time` en vez de un `bool`). La mutación lo desmintió: **el test seguía verde** con el booleano roto, porque quien llama comprueba además `fin.After(ahora)` y el año 1 no está después de hoy. El booleano **no era load-bearing para el único caller que hay**, y el test que decía protegerlo protegía otra cosa. Es una guarda que no guardaba con el agravante de que **su comentario afirmaba lo contrario**, o sea que la siguiente persona que lo leyera habría dado el contrato por cubierto. Se arregló comprobando el contrato **donde se declara** (`TestFinDeVigenciaDistingueLaAbiertaDeLaQueAcaba`, que sí se pone rojo) y corrigiendo el comentario para que diga lo que guarda de verdad. **Regla: un comentario de test que dice "esto protege X" hay que romper X y ver el rojo, igual que la propia guarda.**

#### El barrido, campo a campo

Recorridas todas las estructuras de opciones, contexto y confianza que cruzan una frontera de confianza. Por cada campo puntero, slice, mapa, interfaz o función, qué significa su valor cero:

| estructura | campo | valor cero | veredicto |
|---|---|---|---|
| `nucleo/ledger.Confianza` | `ClavesConfiables []string` | `len == 0` → `ErrSinClavesConfiables` | **restrictivo** |
| | `VerificarSello func` | `nil` → error, "un anclaje que nadie verifica no es un anclaje" | **restrictivo** |
| | `ClaveOperador ed25519.PublicKey` | exigida por longitud **si hay lápidas** | **restrictivo y proporcionado**: sin lápidas no hay nada que verificar |
| `nucleo/expediente.ContextoReceptor` | `Anclas map[string]string` | `len == 0` → falla, "sin ellas la verificación del corpus sería circular" | **restrictivo** |
| | los otros tres | se delegan tal cual en `ledger.Confianza` | **restrictivo** |
| `adaptadores/tsa/internal/pkcs7` | `opts.Roots` | `nil` → `ErrSinAnclas` (recorte 4) | **arreglado**, era permisivo |
| | `opts.CurrentTime` | cero → `ErrSinInstante` (recorte 3) | **restrictivo** |
| | `opts.KeyUsages` | `len == 0` → `ErrSinUsos` (recorte 5) | **arreglado**, era permisivo Y ensanchado a mano |
| `adaptadores/oidc.Configuracion` | `Algoritmos []string` | `len == 0` → `AlgoritmosPorDefecto`, que es lista blanca | **restrictivo**, y trata igual `nil` y vacío |
| `superficies/scim.Opciones` | `Token string` | `""` → error; menos de 32 caracteres → error | **restrictivo** |
| | `MaxCuerpo int64` | `<= 0` → 1 MiB | **restrictivo** |
| `superficies/serve.Config` | `MaxCuerpo int64` | `<= 0` → 4 MiB | **restrictivo** |
| | `CSP string` | `""` → `CSPPorDefecto`, y se valida | **restrictivo** |
| | `CookieInsegura bool` | `false` = cookie segura | **restrictivo**, y el nombre en negativo es lo que lo consigue |
| | `CSPDebilitadaAProposito bool` | `false` = se valida la CSP | **restrictivo**, mismo truco |
| | `ProxiesDeConfianza int` | `0` → `X-Forwarded-For` se ignora entero | **restrictivo** |
| | `HostsPermitidos []string` | `len == 0` → **acepta cualquier `Host`** | **PERMISIVO**, ver abajo |
| `adaptadores/diagnostico.Opciones` | `RaicesTSA []byte` | vacío → se juzgan las que trae el binario | **no es frontera**: `doctor` informa, no verifica un expediente |
| `adaptadores/actualizador.Opciones` | `Canal Canal` | `nil` → no es permisivo, es un pánico diferido | **P2 nuevo**, ver abajo |

**Diecisiete campos, dos hallazgos.** Los dos únicos permisivos de `pkcs7` ya están arreglados en este mismo bloque. De lo demás sale una cosa que conviene decir en voz alta porque no es casualidad: `CookieInsegura` y `CSPDebilitadaAProposito` están nombrados **en negativo a propósito**, y por eso su valor cero es el seguro. Nombrar el campo por lo que se relaja, y no por lo que se protege, convierte el olvido en la opción segura. Es más barato que un centinela y no se puede olvidar.

**P2 nuevo (a).** `serve.Config.HostsPermitidos` vacío acepta cualquier cabecera `Host`. Está documentado, y el control compensatorio también: *"en este paquete no se construyen URL absolutas a partir de `r.Host`"*. El problema es que **ese control compensatorio es una frase, no una puerta**: el día que alguien añada un correo de escalado con un enlace, o una redirección absoluta, no se pone rojo nada. La comprobación de origen CSRF sí compara contra `r.Host`, y ahí es correcto (un navegador no deja falsificar `Host` en una petición entre sitios), pero eso no cubre la generación de enlaces. **Arreglo: un test que recorra el AST de `superficies/serve` y falle si `r.Host` se usa fuera de `hostPermitido` y `origenAceptable`.**

**P2 nuevo (b).** `actualizador.Nuevo` acepta `Opciones` con `Canal` a `nil` y lo guarda. No es un permiso, es un pánico diferido: revienta cuando alguien actualiza, que es el peor momento. Los otros dos campos sí tienen defecto (`Raiz` vacío → `.`, `Ahora` cero → el reloj). **Arreglo: `Canal` nil es un error de construcción, con el mismo criterio que `ErrSinAnclas`.**

### Subfamilia: alcanzabilidad, no existencia

La versión corpus de la familia, y la que va a reaparecer. **Una regla que existe no es una regla que alguien pueda satisfacer.** El linter comprobaba que un paquete con relojes declarase reglas de aplicabilidad; los trece paquetes con reloj lo pasaban. Lo que no comprobaba era si alguna de esas reglas **alcanzaba** al reloj, y ahí había **siete relojes muertos en cuatro paquetes**: `dora`, `nis1-es` y `psd2-es` declaraban `en_ambito(E) :- designado(E, "...")` y ni una sola `aplica`, más tres de `ens`. Como `aplica(O, S)` es el único predicado por el que una obligación llega a un sujeto, esos relojes no se encendían para nadie: ni expediente, ni calendario, ni explain.

**Lo que hace a esta subfamilia distinta de las demás es el síntoma.** Una guarda rota da verde; ésta da **silencio**, y el silencio en un producto de cumplimiento se lee como *"no me toca"*, que es la respuesta más cara que se puede dar mal.

**La forma general, para reconocerla en el siguiente sitio:** cuando una comprobación verifique que *existe* un mecanismo, preguntar además si existe **un camino que lo active**. Dónde va a volver:

| dónde | la pregunta que hay que hacerse |
|---|---|
| escalado | la obligación declara escalones, ¿hay **alguien a quien escalar** en algún alcance posible? |
| conectores | el entregable pide una evidencia, ¿hay **algún conector que la produzca**? |
| plantillas | el campo declara `origen`, ¿hay **algún camino que rellene ese origen**? |
| preguntas | la pregunta `desbloquea` una obligación, ¿la alcanza **alguna regla**? |

**Cerrado** para relojes, con sus dos granos y su control negativo (`ErrRelojSinAplicabilidad`, `ErrRelojQueNadieEnciende`).

**Y cerrada la fila de PLANTILLAS el 29-08-2026**, que es la que se predijo aquí y volvió por donde se dijo. El linter comprobaba que `origen` no estuviera vacío; `iso27001.anexo_a.*` tampoco lo estaba, y no casaba con ninguna de las 93 obligaciones del anexo, que se llaman `iso27001.a.5.1`. **Seis campos de cuatro paquetes apuntaban a nada**, y el síntoma es el de la familia entera: el entregable habría salido con el hueco **sin decir que lo tenía**.

Tres cosas que quedan de ahí y valen para las dos filas que siguen abiertas (escalado y conectores):

- **La gramática se declara o no existe.** No había ninguna escrita, así que cada paquete inventó la suya: trece sufijos distintos. Ahora es vocabulario cerrado, y el sufijo se elige por el más largo recorriendo una **lista ordenada** y no el mapa — un orden de recorrido aleatorio convierte una gramática en una lotería.
- **La medición se hizo dos veces a propósito.** La primera dio *39 origenes colgando* y era falsa: contaba como rotos los sufijos y los globs, que son gramática que la comprobación no sabía leer. Un hallazgo de 39 habría sido tan inútil como uno de cero. Los seis reales salieron de buscar, por cada origen, **el prefijo más largo que sí es un id real** y mirar qué quedaba detrás. *Antes de contar un hallazgo, descartar que sea una gramática propia mal leída.*
- **Derivable o aportado, y dicho.** Un campo o lo calcula plazum o lo escribe una persona, y las dos cosas son legítimas. Lo que no lo es: que se escriban igual, porque entonces **un hueco que nadie va a rellenar tiene el mismo aspecto que un dato que sale solo**. El mensaje del propio linter ya decía *"un entregable no puede tener huecos que rellene un humano sin trazabilidad"*, y lo que faltaba era exactamente eso.

**El séptimo caso estaba en un FIXTURE, no en el corpus**, y es el que más conviene recordar: `paqueteDemo` generaba plantillas con `entidad:x.y` cableado mientras parametrizaba todo lo demás — un placeholder que no apuntaba a nada, dentro del generador que tres tests usan para demostrar que una norma nueva no toca código. Una puerta nueva encuentra primero la basura de los andamios.

### Subfamilia: el descarte silencioso

`Derivar12Meses` tenía `if !vigente { continue }`. Un `continue` mudo, en una función **cuya propia cabecera promete** que *"lo que no produce fecha NO desaparece: sale en `SinFecha` con el motivo"*. Era la única rama que incumplía la promesa del fichero en el que estaba escrita, y se llevaba entera del calendario cualquier obligación que empieza a obligar **dentro** de la ventana que se está mirando: las dos notificaciones del art. 14 del CRA, quince días antes de aplicarse.

Es la guarda-que-no-guarda **en forma de producto**: no deja pasar algo malo, deja de enseñar algo bueno. Y no la caza ningún test de los que había, porque todos preguntaban por lo que sale y ninguno por lo que se cae.

**El barrido, y es barato:** recorrer todos los `continue` y descartes de las derivaciones **de cara al usuario** (`nucleo/pantalla`, `superficies/`, `cmd/plazum`) y exigir que **cada uno diga a qué cubo va lo que descarta**. Si no hay cubo, es el bug de hoy con otro nombre. La regla, dicha para que se pueda aplicar sin pensarla: *en una derivación que el usuario ve, un elemento sólo desaparece si desaparecer es la respuesta, y entonces se cuenta.*

**El barrido ya está hecho para `Derivar12Meses`** (28-08-2026): nueve descartes, siete con cubo o de puro flujo, **dos sin cubo**.

| descarte | cubo | veredicto |
|---|---|---|
| `Temporalidad == nil` | ninguno | correcto: no es un reloj, y esto es un calendario de relojes |
| vigencia ilegible | `SinFecha` + motivo | ✓ |
| primitiva sin ejecutor | `SinFecha` + motivo | ✓ |
| vencimiento fuera de los doce meses | `FueraDeLaVentana` | ✓ |
| pendiente de hecho / sin plazo legal | `SinFecha` + motivo | ✓ |
| **derogada** | **ninguno** | **hueco.** `corpus.VigentesEn` documenta en su propia cabecera que *"quien la use para pintar una pantalla tiene además que DECIR qué ha pasado: una obligación que desaparece de la lista sin explicación se lee como un fallo del producto"*. El calendario no lo dice. Es el mismo fallo que el estreno, en la otra dirección del tiempo |
| **no alcanzado por la aplicabilidad** | **ninguno** | **sin decidir por escrito.** Contarlo sería casi todo el corpus y no ayuda; no decir nada es lo que hay hoy. Decidir y anotar, no dejarlo por omisión |

**CERRADOS los dos el 28-08-2026, en `docs/decisiones.md` D-13.** La derogada gana cubo propio (`Cese`, espejo exacto del `Estreno`: *deja de obligarte dentro de esta ventana*), y lo no alcanzado gana una linea en la cuenta con su puerta (`--todos-los-relojes`). Ni enumerar ni callar: contador con puerta.

**Y con ellos entra lo que impide que la familia vuelva: la contabilidad quedo CERRADA y se comprueba SUMANDO.** Cada hito instalado cae en exactamente un cubo de la particion por tiempo (en vigor, estrena, ya ceso, empieza despues, vigencia ilegible), y lo que esta en vigor cae en exactamente uno de la de alcance. Un test lo suma. **Es la unica forma de test de esta familia que crece sola**: el dia que alguien anada una rama a la derivacion y se olvide de contarla, la suma se rompe sin que nadie tenga que acordarse de escribir el caso. Es justo lo que faltaba cuando el `continue` mudo paso trece revisiones. El barrido queda pendiente para las OTRAS derivaciones de cara al usuario (`superficies/pantallas`), donde la ley de conservacion todavia no existe.

### Doctrina: el estreno es un tipo aparte, y va a todas las superficies

`Estreno` no es una `Fecha`, y la distinción es de producto, no de modelado. Una `Fecha` es un **vencimiento**: algo que tienes que haber hecho antes de esa hora. Un estreno es lo contrario, **el día en que empieza la cuenta**. En la misma lista, la fila diría *"entrega esto el 11-09-2026"*, que es falso y además alarmante.

*"Empieza a obligarte dentro de esta ventana, y hoy no has incumplido nada"* es la frase que ningún competidor dice, porque para decirla hay que tener el reloj legal y hay que estar dispuesto a enseñar un cero. **Es doctrina, no detalle de `calendario`.**

**Pendiente:** cuando la pantalla **Hoy** tenga datos reales, los estrenos **y los ceses** van ahí también. Hoy sólo los pinta `plazum calendario`. La doctrina, con el porqué de que un cese sea buena noticia y no una fila de mantenimiento, en `docs/decisiones.md` D-13.


### Familia B, tramo 2: las 34 cadencias sin número de 2024/2690, propuestas y NO escritas (28-08-2026)

Las 34 propuestas de intervalo existen, con su justificación, su hito, su hecho y su título, y pasaron una revisión de coherencia adversaria. **No se escribieron en el paquete**, y el motivo es que la revisión encontró **cuatro problemas bloqueantes** y ninguno es cosmético. Escribirlas igual habría metido en el corpus 34 obligaciones con argumentos que se contradicen entre sí, que es peor que no tenerlas: un CISO que lea dos fichas seguidas encuentra la contradicción en un minuto y deja de fiarse del resto.

Las propuestas y la revisión entera están en el journal del run `wf_024e6e58-5f6`.

**Los cuatro bloqueantes:**

| # | qué pasa | arreglo |
|---|---|---|
| **3.3.2 vs 8.1.3** | 3.3.2 justifica su P12M diciendo que se cuelga del ciclo anual de concienciación, y 8.1.3 (que ES el programa de concienciación) está propuesto a P6M. La premisa la desmiente el propio conjunto | mover 3.3.2 a P6M como módulo dentro de la entrega de 8.1.3, o argumentar el P12M sin apoyarse en 8.1.3 |
| **6.9.2** | el intervalo cuelga del verbo equivocado: leído literal dice que el antimalware **se actualiza** cada tres meses, cuando el punto pide comprobar cobertura. **Autoriza un infracumplimiento** | el hito pasa a `comprobacion_cobertura_deteccion_malware` y el título a comprobar que la flota lo tiene instalado, activo y al día |
| **3.6.3 vs 2.2.1** | 3.6.3 se ancla a un informe trimestral a la dirección que 2.2.1 fija en semestral y **rechaza expresamente** el trimestre | anclar 3.6.3 al control de cumplimiento de 2.2.3, que sí es trimestral |
| **12.2.3 vs 12.3.3** | **texto legal idéntico palabra por palabra**, puntos adyacentes de la misma sección, y salen a P24M y P12M | 12.2.3 a P12M, y escribir UNA vez la doctrina: P24M se reserva a puntos cuyo contenido lo fija una norma ajena a la seguridad (10.2.3 y 10.4.2, derecho laboral) |

**Y dos hallazgos que valen más que las 34, porque no son de este tramo:**

**1. Una justificación puede meter por la puerta de atrás lo que el invariante 3 prohíbe por la principal.** El P6M de 6.7.3 se apoyaba en *"el sector de medios de pago lleva años exigiendo la revisión del conjunto de reglas de cortafuegos cada seis meses"*, que es **contenido de PCI DSS parafraseado**. No es texto pegado, así que el linter no lo ve: el limitador de caracteres mira longitud, no procedencia. **Es un vector nuevo de la frontera legal y va a reaparecer cada vez que plazum ponga un número**, porque la forma natural de justificar un intervalo es apoyarse en la práctica reconocida, y media práctica reconocida vive en catálogos privativos.

Regla que queda: **una justificación se apoya en fuente primaria (NIST, ENISA, BOE, DOUE) o en el reloj propio del punto, nunca en lo que exige un catálogo de pago.** Citar que PCI DSS existe es una cosa; sostener nuestro número sobre su contenido es redistribuir su criterio.

**2. Nadie sumó el año.** 7 puntos a P3M, 9 a P6M, 14 a P12M y 4 a P24M dan **unas 62 citas fechadas al año, y sólo de este marco**. Es más de una ceremonia de cumplimiento por semana para el CISO de 200 empleados de la tercera pasada, antes de sumar ENS, ISO o lo suyo propio. **Un calendario que nadie puede cumplir no es un calendario, es un reproche semanal**, y el producto que lo genera se cierra al segundo mes.

Queda pendiente **el criterio de acceso al trimestre**, y la propuesta de la revisión es buena: P3M se reserva a controles cuya evidencia **la produce una máquina** (el escáner de 6.10.2, la consola de agentes de 6.9.2, el inventario de 12.4.3, el sistema de tiques de 3.6.3), no a los que exigen que una persona se siente a revisar. Sin ese criterio escrito, cada autor futuro pondrá P3M a lo que le parezca urgente y la suma crecerá sola.

**Lo que se comprobó y NO es un problema:** la revisión proponía convertir en puerta la regla `hecho == "ultima_" + hito`. Se midió antes de proponerla: **7 de las 23 cadencias del corpus ya publicado no la cumplen, y las desviaciones son correctas**. `iso42001` sufija `_aims` (`ultima_auditoria_interna_aims`) justamente para no chocar con los hechos de `iso27001` en un sujeto que tenga los dos sistemas de gestión. Esa puerta habría roto datos buenos. Queda anotado para que no se vuelva a proponer sin medir.

**Lo demás de la revisión** (10 fichas sin `cuando_cambiarlo`, el argumento comodín de la *fatiga de firma* repetido en nueve puntos, referencias colgadas al lote del tipo *"el único de los cinco"* que el lector de una obligación suelta no puede resolver, y `6.10.2` llamando *bajar* a lo que es *alargar*) es de redacción y se arregla en la misma pasada.

#### Escritas: eran 44 y no 34 (28-08-2026)

**Los cuatro bloqueantes se cerraron y las once que quedaban también.** Las cadencias sin número del anexo de 2024/2690 están escritas, con sus 132 dorados, en `paquetes/nis2-tecnica`.

**Y el censo estaba corto: son 44, no 34.** El recuento anterior sólo miraba los puntos de **tres niveles** (`N.M.K`), y las secciones 7 (evaluación de la eficacia) y 9 (criptografía) del anexo **se numeran a dos** (`7.3`, `9.3`). Además se le pasaron ocho puntos de tres niveles cuyo verbo periódico no encajaba en el patrón buscado: 4.2.3, 4.2.6, 4.3.4, 8.2.1, 12.1.3, 13.1.3, 13.2.3 y 13.3.3. Y uno de los **numerados** también faltaba: el punto **3.4.2, letra b)**, que dice *«la evaluación trimestral de la existencia de incidentes recurrentes»* — son cuatro con número, no tres.

Es la tercera vez que el censo de este anexo se corrige (61 → 57 → 44+4), y las tres por lo mismo: **contar frases en vez de contar deberes**. La regla que queda: un censo de relojes se cierra recorriendo los marcadores de punto **en orden de documento y con el nivel de numeración que cada sección use**, no con una expresión regular de forma fija.

**Lo que sí resolvió cada una de las once:**

| # | punto | cómo quedó |
|---|---|---|
| 1 | 12.2.3 vs 12.4.3 | no se contradicen: los objetos son distintos y ahora se nombran. 12.2.3 es **la política** de gestión de activos (P12M, proceso); 12.4.3 es **el inventario** (P3M, dato). Las dos justificaciones se citan la una a la otra |
| 2 | 12.2.3 renombraba el punto | el título sale ahora de las palabras del anexo: *«Revisar la política para la correcta gestión de los activos»*, que es el objeto del punto 12.2.1. La cita lo dice expresamente |
| 3 | 3.3.2 | ya no se apoya en 8.1.3. Se ancla en el punto 3.3.1 (*el mecanismo será SENCILLO*) y en el 8.1.2.a), que es la vía por la que un nuevo empleado lo aprende sin esperar al ciclo |
| 4 | 6.9.2 | el hito cuelga expresamente de **comprobar la cobertura**, no de actualizar. La justificación lo dice en voz alta, porque la propia norma delega la frecuencia de la actualización en la evaluación de riesgos y en los contratos con proveedores |
| 5 | 2.2.1 | el punto lleva **dos deberes periódicos** y ahora se dice: el control del cumplimiento se escribe en 2.2.3 (donde la norma añade los disparadores) y el informe al órgano de dirección en 2.2.1. No se fabrican dos ceremonias para una actividad |
| 6 | 5.1.7 | sólo la **letra a)** es periódica. El título, el hito y la cita lo dicen; b), c) y d) quedan fuera del número |
| 7 | 6.5.3 | corregido el hecho: 6.5.3 **no trae disparador por evento**, y eso es lo que sostiene su intervalo. Lo mismo se comprobó y se escribió en 6.2.4, 9.3, 10.2.3 y 11.6.4 |
| 8 | las cinco cruzadas | escritas en el mismo commit y comprobados los cinco valores: 3.6.3 sobre 2.2.3 (P6M), 6.3.3 sobre 6.10.2 (P3M), 11.3.3 sobre 11.2.3 (la mitad, P3M), 11.5.4 sobre 11.2.3 (igual, P6M), 6.7.3 sobre 6.7.2 (sin apoyarse en ningún número nuestro) |
| 9 | 6.1.3 | fuera la Directiva 2014/24/UE. Se ancla en el punto 6.1.1, que ata los procedimientos a la evaluación de riesgos del 2.1, que es anual por ley |
| 10 | 8.1.3 | fuera la curva de tasa de clic. Se ancla en el 8.1.2.a) (*incluirá a los nuevos empleados*) y en el ciclo anual de riesgos |
| 11 | 6.3.3 | fuera la guía del fabricante. Se ancla en el 6.3.2.b) (todo el ciclo de vida) y **declara** que se apoya en la cadencia del 6.10.2, que también es nuestra |

**Una desviación deliberada, dicha para que no sea silenciosa** (que es lo que se le reprochó a la anterior): el hito del 6.9.2 se llama `comprobacion_de_la_cobertura_antimalware` y la nota de la rederivación decía `comprobacion_cobertura_antimalware`. El fondo del arreglo se respeta (cuelga de COMPROBAR); la forma sigue el estilo del resto del paquete, `<accion>_de_la_<objeto>`.

**Lo que queda de este anexo:** los **disparadores por evento** de casi todos los puntos de revisión (*«o cuando se produzcan incidentes significativos o cambios significativos»*), que son `observacion` y no `periodica`; el punto **3.4.2.b)** con su trimestre; y los **artículos 3 a 14**, que son los umbrales de incidente significativo por tipo de entidad.

#### Cerrado: el P1 del comprador, y lo que había detrás (29-08-2026)

Las dos mitades, en el orden en que tenían que ir.

**1. La agrupación**, que es el punto 3 de D-15 y está hecha: `LAS SENTADAS`, delante del listado por meses. El detalle y los números, en `docs/decisiones.md` D-15.

**2. El hecho en el perfil**, después: `papel_nis2_tecnica(E, "entidad_pertinente")` en `es-servicios-digitales`, con **confianza media** y un `porque` que enumera los once tipos cerrados del artículo 1 y dice que si no eres uno de ellos hay que quitarlo.

**Y detrás había un P0 que nadie había visto.** Al enchufar el reglamento técnico en el perfil, la contabilidad decía:

```
55 alcanzados por la aplicabilidad
 6 fechas en los proximos doce meses
 3 hitos sin fecha, con su motivo arriba
```

**46 relojes en ningún sitio.** `corpus.VencimientosDe` hacía, para una `periodica` cuyo hecho de arranque no consta, un `return nil, nil`: la obligación no producía fecha, no producía fila de «sin fecha» y no se contaba en ningún cubo. Desaparecía del calendario, del expediente y de `explain`.

**Lo peor no es el fallo, es dónde estaba.** La contabilidad del calendario tiene dos particiones exhaustivas (por tiempo y por aplicabilidad) y **las dos cuadraban**: el hueco estaba *después*, entre «te alcanza» y «sale en pantalla», que era el único tramo sin ley. Y la rama equivalente de `Plazo` ya llevaba escrito, desde hace semanas, el aviso exacto: *«una lista vacía se leería como "nada que hacer", que es el peor error posible aquí»*. El aviso estaba; la otra rama no lo había leído.

**La ley que faltaba**, y ahora existe: *toda obligación en vigor que la aplicabilidad alcanza aparece en `Fechas` o en `SinFecha`*. Las dos son respuestas; la nada no lo es. La mutación que devuelve el `return nil, nil` la pone roja con **67 de 83** desaparecidas.

**Lo que esto enseña sobre la familia del descarte silencioso:** un `continue` mudo se busca leyendo el código, y así salieron los tres cubos del calendario. Éste no era un `continue`: era un **`return` de éxito con la respuesta vacía**, que ni parece un descarte ni lo parece al leerlo. La forma general de la regla, entonces, no es «vigila los `continue`» sino **«por cada camino que devuelve la nada, pregunta si la nada es una respuesta»**.

#### El tercer descarte silencioso del mismo día: lo ya vencido (29-08-2026)

Salió al preparar los disparadores por evento, y es de la misma familia que el `return nil, nil` de la mañana. La derivación hacía:

```go
if v.Vence.Before(ahora) || !v.Vence.Before(hasta) { cal.FueraDeLaVentana++; continue }
```

**Un vencimiento pasado y uno posterior a la ventana caían en el mismo cubo**, y ese cubo se imprimía con la etiqueta *«fechas más allá de los doce meses»*. Medido: una obligación anual cuya última ejecución consta en 2022-01-15, mirada el 2026-08-27, daba **«0 fechas en los próximos doce meses»** y **«4 fechas más allá de los doce meses»**. Las cuatro estaban *detrás*, no delante.

Para un producto de continuidad de cumplimiento, *«llevas cuatro ciclos sin hacer esto»* es la fila más importante que puede imprimir, y era la única que no imprimía.

**Y al imprimirla salió el peor falso positivo posible.** La primera versión acusaba de incumplir el 2023-01-15 una norma en vigor desde el **2024-11-07**. El ancla de una cadencia es un hecho del operador (*«la última vez que lo hice»*) y puede ser muy anterior a la norma: quien revisó su política en 2022 no incumplía en 2023 un reglamento que no existía. Guardado, con su propio cubo contado.

**Tres decisiones de presentación que valen para toda esta clase:**

- **Una fila por obligación, no una por ocurrencia.** Cuatro años de incumplimiento anual son cuatro vencimientos y **una** noticia. Se da el más antiguo (*«¿desde cuándo?»*, que es la pregunta de un inspector) y el número de ciclos.
- **Va arriba del todo**, antes incluso de las sentadas: de todo lo que el calendario dice, un incumplimiento en curso es lo único que no admite planificación.
- **Con su descargo, y sin él no se publica.** *«Esto NO dice que se haya incumplido: dice que en tus respuestas no consta que se hiciera»*. Sin esa frase, una ausencia de dato se lee como una acusación, y la primera reunión en la que eso pase se lleva por delante la confianza en la pantalla entera. Hay un test que exige la frase.

**Lo que los tres del mismo día enseñan juntos.** Un `continue` mudo se encuentra leyendo el código. Un `return nil, nil` no parece un descarte. Y éste no era ni una cosa ni la otra: era **un cubo bien contado con la etiqueta equivocada**, que es la forma más difícil de ver de todas, porque la contabilidad cuadra y el número está a la vista. La regla, en su forma general: **por cada camino que devuelve la nada, pregunta si la nada es una respuesta; y por cada cubo, si su etiqueta describe todo lo que cae dentro.**

#### Hechos: los disparadores por evento del anexo de 2024/2690 (29-08-2026)

**22 de las 47**, y la decisión de modelo es lo que importa. Casi todo punto de revisión dice *«a intervalos planificados **o cuando se produzcan incidentes significativos o cambios significativos**»*. **Eso no crea un segundo deber: crea un segundo disparador del mismo deber.**

Escribirlos como obligaciones aparte habría dado **69 obligaciones donde hay 47**, y le habría dicho al cliente que tiene el doble de ceremonias — el día después de escribir la sección que existe para no empeorar exactamente eso (D-15). Van como campo `reabre_por` de la obligación.

**Los hechos se derivaron del texto de cada punto**, no de una lista escrita a mano, y por eso salen desiguales y correctos: el 6.1.3 sólo reabre por incidente (su texto no menciona los cambios) y el 10.4.2 por cambio significativo y por **cambio jurídico**, que es lo que mueve un procedimiento disciplinario.

**Qué pasa al reabrirse, y qué no se hace.** La revisión pierde su fecha y sale como *«sin plazo legal»*, con la derivación entera al lado (qué hecho, de qué fecha, y cómo se cierra). **No se inventa una fecha límite**: la norma dice *cuándo* hay que revisar (al ocurrir el hecho) y no da plazo para hacerlo. Es el mismo trato que ya reciben las tres obligaciones sin número del corpus.

**Dos decisiones de borde que evitan bucles:**

- **El empate no reabre.** Si la revisión consta el mismo día del incidente, se hizo *después* de él (por eso consta). Tratar el empate como reapertura pediría repetirla para siempre: cada vez que se registrara la nueva revisión, el incidente volvería a empatar. No daría error; daría una obligación que no se puede cerrar nunca.
- **El linter rechaza la reapertura que no puede dispararse jamás**: la entrada vacía, la repetida, y sobre todo la que nombra **el mismo hecho del que arranca el ciclo** — nada es posterior a sí mismo, así que esa reapertura no salta nunca y el paquete afirmaría una protección que no tiene.

**Y el control negativo hizo su trabajo por segunda vez en el día.** `TestLaComprobacionDeLaTraduccionSaltaCuandoDebe` desplaza cada esperado una hora y exige que fallen todos; se puso rojo diciendo *«tenía que fallar en los 314 casos y sólo falló en 292»*. Los 22 que faltaban eran los nuevos: **un esperado sin fechas no se puede desplazar una hora**. La mutación tuvo que ganar una segunda forma (estropear el estado esperado). Un control negativo que no sabe mutar una clase de caso deja esa clase sin demostrar, **y lo dijo él solo**.

### La ley de conservación se hace de extremo a extremo (29-08-2026)

**Queda como puerta permanente**, y el porqué es la lección del P0 de los 46 relojes.

Cuando se perdieron, **las dos leyes de conservación que ya existían dieron verde durante todo el episodio**. No fallaron: ninguna miraba ahí.

| ley | qué cuadraba | dónde no llegaba |
|---|---|---|
| partición por **tiempo** | en vigor + estrenan + ya cesados + empiezan después + ilegibles = instalados | no dice nada de lo que pasa después de «en vigor» |
| partición por **alcance** | alcanzados + no alcanzados = en vigor | termina en «te alcanza» |

El hueco vivía **exactamente entre las dos**: entre *«te alcanza»* y *«sale en pantalla»*. **Dos particiones que se equilibran cada una por su lado no componen una ley: componen dos leyes con una junta sin vigilar, y la junta es donde se rompe.**

**La ley nueva es una sola cadena**, de `corpus.Cargar` hasta el cubo impreso: *todo reloj instalado acaba en exactamente un destino, y los destinos son o una fila que el usuario ve o una ausencia con nombre y motivo*. Vocabulario cerrado de nueve destinos, en `nucleo/pantalla/destinos.go`.

**Y hace DOS preguntas, no una.** La segunda es la que cierra la junta: **¿el destino que promete una fila TIENE esa fila?** Etiquetar es barato; sin esa mitad, una etiqueta puesta sin fila detrás dejaría la ley en verde y el reloj sin verse, que es literalmente el fallo que persigue.

**No es aritmética, y no puede serlo.** Los contadores de arriba cuentan **hitos** y los de abajo cuentan **vencimientos**, que una periódica multiplica: una suma que mezclara los dos cuadraría por casualidad o no cuadraría nunca. Es un **etiquetado**, y por eso no se puede equilibrar solo.

**El detalle que lo hace familia, y es el más incómodo:** la rama equivalente de `ventana.Plazo` **llevaba el aviso escrito desde semanas antes**, con estas palabras — *«una lista vacía se leería como "nada que hacer", que es el peor error posible aquí»*. Estaba en un comentario, a treinta líneas de la rama que lo incumplía, y no sirvió de nada.

> **Un aviso en un comentario no viaja; una ley en un test sí.**

La mutación M34 (devolver el `return nil, nil` original) deja **67 relojes sin destino** y la puerta lo dice con sus nombres.

### Clase nueva: el falso positivo hacia el supervisor (29-08-2026)

**Lo nombra el umbral de MiCA, y es la peor de las tres formas de equivocarse con un reloj.**

| forma | qué produce | quién lo paga |
|---|---|---|
| el reloj **falta** | el cliente incumple sin saberlo | el cliente, ante la autoridad |
| el reloj **aprieta de más** sobre una ceremonia interna | reuniones que la norma no pide | el cliente, en horas |
| el reloj **alcanza a quien no obliga** y su destinatario está **fuera** | una actuación indebida ante el supervisor | el cliente, ante la autoridad, **por culpa de plazum** |

La tercera no es *"un coste más"*. El art. 22.1 de MiCA pide una comunicación trimestral a la autoridad competente, y sólo a los emisores de fichas referenciadas a activos que emitan **por encima de 100 000 000 EUR**. Escrito sin el umbral, **todo emisor de ART manda a su autoridad, cada trimestre, una comunicación que no le corresponde**. No hay error, no hay rojo y no hay queja: hay una casilla verde, un deber dado por cumplido y un supervisor recibiendo lo que nadie le pidió. Y a diferencia de los otros dos, **no se puede deshacer**: el acto ya salió de la organización.

**Por qué es clase y no caso: las dos direcciones de un umbral no cuestan lo mismo.** Umbral **de menos** (el reloj alcanza a más gente de la que la norma obliga) es un error silencioso y **hacia fuera**. Umbral **de más** (alcanza a menos) es el error clásico, un deber que no se enseña, y lo caza el cliente el día que su asesor se lo dice. Las comprobaciones tienen que recorrer **las dos** — es el invariante 7 otra vez, en su forma legal — y por eso el test del umbral se escribió con sus dos direcciones y con el apartado de la exclusión en el mensaje (M38).

**El conjunto es enumerable, y ése es el hallazgo aprovechable.** La clase `notificatoria` de `clase_e2e` marca exactamente las obligaciones cuyo entregable **sale de la organización**. Hoy son 21 y se listan con un comando. **Barrido hecho el 29-08-2026:**

- **19** no tienen umbral de entidad porque su umbral **es el evento**: son notificaciones de incidente, y lo que decide si se notifica no es el tamaño del obligado sino **la clasificación del incidente**.
- **2** tienen umbral de organización: `mica.art22_1` (los 100 M EUR, escrito con él) y `dora.art28_3`, que no lleva ninguno porque el apartado alcanza a toda entidad financiera — comprobado en el texto, no supuesto.

**Y ese 19 de 21 mueve la familia entera a la etapa 4.** Si en casi todas el umbral es la clasificación del incidente, entonces **los criterios de clasificación SON el umbral**, y equivocarlos en la dirección ancha es exactamente este fallo con otro nombre: notificar a la autoridad un incidente que no era significativo. Los arts. 3 a 14 del Reglamento de Ejecución (UE) 2024/2690 no son un detalle del objeto `Incidente`: son **el sitio donde esta clase vuelve, con el destinatario más caro que hay**.

**La pregunta fija**, para toda obligación `notificatoria`: **¿a quién alcanza, y de dónde sale que no alcanza a los demás?** Se contesta con el apartado que excluye, o no se contesta.

**Dónde vuelve, con el dato que hace de umbral:**

| marco | el umbral |
|---|---|
| NIS2 | el tamaño (mediana empresa, Recomendación 2003/361) y las excepciones del art. 2 |
| Ley 2/2023 | los 50 trabajadores del sistema interno de información |
| DORA | la microempresa (escrito, con su exclusión) y el marco simplificado del art. 16 |
| MiCA | los 100 M EUR (escrito) |
| **2024/2690, arts. 3-14** | **la clasificación del incidente**, y es el que va a la etapa 4 |

**Lo que NO es lintable, dicho para que no se confunda con una garantía.** No hay regla mecánica que diga *"esta notificatoria necesita umbral"*: el art. 33 del RGPD alcanza a todo responsable y está bien que lo haga. Lo que la clase `notificatoria` da es **el conjunto sobre el que hay que hacer la pregunta**, que es distinto de contestarla. Misma frontera que `fuentes_del_intervalo`: la máquina acota, la pasada humana decide.

### Familia: una URL de configuración es una credencial que no se puede reconocer (30-08-2026)

**Salió de aplicar al vecino la regla que se acababa de escribir para Teams.** El adaptador de sellado metía `a.URL` entera en dos errores. Hoy es inofensivo — FreeTSA y Certum son públicas — pero **`Cadena.Autoridades` es configuración**, y una TSA de pago pone el token en la consulta.

**La regla, y su porqué, que es lo único que la hace aplicable:** *cuáles de las URL que configura el operador son credenciales **no es una propiedad que el código pueda saber***. El webhook de Teams basta tenerlo para escribir en el canal; el destino de un *dead man's switch* lleva el identificador secreto **en la ruta** (`hc-ping.com/<uuid>`); una TSA de pago lo lleva en la consulta. Las tres se ven igual que una URL pública. Así que la regla no puede ser *«redacta las secretas»*, que obliga a acertar, sino **«no sale ninguna entera»**, que no obliga a nada.

Y lo que lo hace urgente es **dónde acaban estos errores**: en el log, en la pantalla, y sobre todo en el bloque copiable de `plazum doctor --issue`, **que existe para pegarlo en un issue público**. Un token que llegue ahí lo publica quien pide ayuda.

**El barrido: 13 sitios en 4 ficheros de 3 adaptadores.** Y lo interesante no es el número: es que **hicieron falta tres métodos y cada uno vio lo que los otros dos no podían**.

| método | encontró | por qué era ciego a lo demás |
|---|---|---|
| **grep por nombre** | 4 | **lee líneas, y Go parte las llamadas largas**. Los cinco mensajes de `ComprobarDestino` llevan `destino` en la línea *siguiente* a la del `Errorf`, así que no salieron |
| **test de centinela** | +2, y de ahí +5 | ve lo que de verdad viaja, incluido **el error envuelto de `http.Client`, que lleva la URL entera dentro** y ningún análisis de nombres puede ver. Ciego a los caminos que el test no ejecuta |
| **puerta de AST** | +3 | ve la llamada entera aunque ocupe cinco líneas, y ve los caminos que nadie ejecuta. Ciega al error envuelto |

**Por eso la guarda tiene dos mitades y las dos están escritas**: `credenciales_test.go` recorre el AST de `adaptadores/` y `superficies/`, y cada adaptador que habla con el exterior tiene su test de centinela. La puerta **dice en su propia cabecera lo que no caza**, que es la mitad del problema.

**Dos detalles que valen más que el recuento:**

- **`url.Redacted()` no redacta.** Sólo sustituye la contraseña del *userinfo* y deja intactos ruta, consulta y fragmento — justo donde los servicios de webhook y de ping ponen su secreto. Estaba usado en `latido` **creyendo que redactaba**, que es peor que no usar nada porque parece hecho.
- **La ironía que lo resume.** `ComprobarDestino` rechaza que el destino lleve parte de consulta **porque *«ahí es donde se cuela un identificador sin que nadie lo note»***… e imprimía esa consulta en su propio mensaje de error. La comprobación que existe para que no haya secretos en la URL publicaba el secreto al dispararse.

**Y una exención, escrita con su condición y no en silencio:** la ruta de una petición **entrante** que se registra al entrar un handler en pánico. Es otra clase (no es configuración) y registrarla es lo que hace operable un servidor. Se eximió **tras comprobar el 30-08-2026 que ninguna ruta de este servidor lleva un secreto en su camino** — no hay enlaces mágicos, la sesión va por cookie y SCIM por cabecera. El día que alguna lo lleve, la exención deja de valer y se quita, no se amplía.

### Subfamilia del invariante 8: el valor fuera de dominio que da un no-op en silencio (30-08-2026)

**La cara nueva la enseña M55.** `ventana.Sumar` avanza días hábiles con `for restantes > 0`. Con un número **negativo** el bucle no entra ni una vez y devuelve **la fecha de entrada, sin error**. O sea que calcular *«sesenta días hábiles antes»* como una suma de duración negada da un aviso que llega **el mismo día del vencimiento**, y no hay nada rojo en ningún sitio.

Es el invariante 8 con otro disfraz: allí el valor cero permisivo, aquí **el valor fuera de dominio que no se rechaza**. Y la diferencia con un fallo normal es la de siempre: no da error, **da un no-op**, y un no-op se parece muchísimo a un cálculo correcto.

**Barrido barato, y es la regla:** todo bucle de transformación de la forma `for x > 0` cuyo contador **pueda llegar a ser negativo** o **rechaza el dominio** (que es lo que hace `Restar`, existiendo aparte) o **lo maneja explícito**. No vale confiar en que nadie pase un negativo: el caso que lo trajo lo habría pasado el código nuestro.

### El lazo local decía 24/24 mientras CI rechazaba el commit (30-08-2026)

**Un rojo en `main`, y el fallo de proceso importa más que el hallazgo.** `./comprobar.sh` imprimió *«24 puertas, todas en verde»*, ese número viajó a un informe, y CI rechazó el commit por un `G304` de gosec.

**El hueco:** `comprobar.sh` corre las puertas que los workflows declaran con `puerta()`, y **el paso de gosec no es una `puerta()`** — es un `run:` normal, bloqueante, dentro del job de seguridad. Así que el lazo local nunca lo ejecutaba, y su verde **no significaba lo que parecía**. Es la misma familia que las tres puertas de `-race` saltadas por falta de gcc, con la diferencia de que aquéllas **se dicen** y ésta no existía.

> **Un lazo local que no cubre un paso bloqueante de CI produce un verde que acaba en un informe.**

**Cerrado**: `comprobar.sh` ejecuta gosec al final, distinguiendo *«no se pudo descargar»* de *«encontró algo»* — confundirlos haría que una máquina sin red se leyera como una máquina limpia — y lo dice cuando lo salta, igual que hace con `-race`. **M77** (devolver la anotación mala) lo pone en rojo.

**Y el hallazgo en sí, que es pequeño y enseña algo:** la anotación que escribí era `//nolint:gosec`, que es la directiva de **golangci-lint**. La puerta corre **gosec**, que lee `// #nosec Gxxx -- motivo`. O sea: **escribí una supresión que no suprimía nada y que tiene exactamente el mismo aspecto que una puesta**. El repositorio ya tenía 58 `#nosec` bien escritos, así que la convención existía y la ignoré.

Es hermana de *la rama que nunca se ejecuta*: una directiva dirigida a una herramienta que no es la que corre es una anotación que no vigila a nadie. Y hay una segunda en el árbol, benigna: un `//nolint:staticcheck` en `adaptadores/tsa/tsa_test.go`, que no molesta porque **staticcheck no se ejecuta en CI** — pero que le dice a quien lo lea que algo lo está silenciando.

**La regla:** antes de escribir una supresión, mirar **qué herramienta corre de verdad** en el workflow y usar SU sintaxis. Una supresión dirigida a la herramienta equivocada es peor que ninguna, porque parece hecha.

### Familia: el parser de la biblioteca estándar se come filas y no lo dice (30-08-2026)

**Salió de medir `encoding/csv` antes de escribir la ingesta manual, y no de leerlo después.** Cuatro resultados, y el cuarto decide el diseño entero:

| lo que se probó | qué hace `encoding/csv` | por qué importa |
|---|---|---|
| BOM de Excel delante | lo mete **dentro** del nombre de la primera columna | el error que recibe quien sube es *«falta la columna usuario»* **sobre un fichero que la tiene** |
| bytes cp1252 (`Martínez`) | **ningún error**: devuelve una cadena que no es UTF-8 válido | si el acento está en el identificador, lo corrompido en silencio es la clave por la que se empareja todo |
| una fila con una columna de más | `ReadAll` devuelve **cero filas** y un error | un censo vacío se lee igual que un censo limpio |
| **una comilla sin cerrar** | **se come las filas siguientes y luego dice «fin»** | **tres personas desaparecen de la revisión y no hay ni un error por fila perdida** |

El cuarto se midió así: fichero de cinco líneas, comilla abierta en la 3, el lector devuelve la 1 y la 2, un error, y EOF. Y **leer fila a fila no lo arregla**: el lector consume el resto buscando el cierre.

> **Un parser que devuelve menos filas de las que hay, sin un error por cada una, convierte «he revisado todo» en una afirmación que nadie puede comprobar.**

**La guarda que sirve, y por qué la obvia no sirve.** La ley de conservación sobre LÍNEAS: las que cuenta un camino independiente tienen que cuadrar con las que explican los cubos (una por legible, **el rango entero** por cada ilegible, una por duplicada). Lo que hace que funcione es que **el contador no sabe de comillas**. Si las respetara como el parser, se tragaría exactamente las mismas líneas y cuadraría siempre: sería una guarda que confirma al vigilado. Es la disciplina de los tres métodos ciegos del barrido de URL, aplicada dentro de un fichero.

El precio se acepta y se dice: un campo con un salto de línea dentro, que es CSV legítimo, no carga. Cuesta mirar el fichero. El fallo contrario cuesta certificar de más, y eso lo firma una persona.

**Y medir encontró dos fallos míos con el verde puesto**, que es el motivo de que esta entrada valga más que sus hallazgos:

- el contador contaba **todos** los saltos de línea, y el lector **se salta las líneas vacías sin decir nada**. Un fichero acabado en línea en blanco (casi cualquiera que haya pasado por un editor) daba descuadre y no cargaba. *Una guarda que salta con lo normal se acaba quitando, y entonces deja de guardar lo raro, que era su trabajo.*
- el número de línea salía de un `linea++` propio, que **se va corriendo desde la primera línea saltada**. Apuntaba mal justo donde importa: en la fila ilegible que alguien va a arreglar y en la duplicada que dice *«ya salió en la N»*. Lo arregla preguntárselo al lector (`FieldPos`), que es quien sabe dónde está.

Ninguno de los dos lo veía la suite: **pasaba igual antes y después**. Los tests que los vigilan se escribieron después de medir, no antes, y ese orden es el hallazgo de método.

**De la pasada 2, la pregunta de siempre:** por qué campo casa el emparejamiento y si ese campo está firmado. La clave de fila es `sistema|cuenta|permiso`; `cuenta` y `permiso` salen de los bytes que cubre el hash, pero **`sistema` lo declara quien sube**. El mismo fichero subido dos veces con dos sistemas distintos da el **mismo hash** y **claves distintas**. Así que hay dos identidades y se publican las dos: `Hash` (el fichero, que un tercero recalcula) y `Sello` (la lectura entera, de la que cuelga la conclusión).

### Una afirmación que no afirmaba, escrita por mí, el mismo día (30-08-2026)

En el test de la delegación:

```go
if e := c.EstadoDe(clave); e != Delegada { ... }
if e.Termina() { t.Fatal("Delegada no puede ser terminal") }
```

El segundo `e` **no es el local**: el del `if` muere con su bloque. Resolvía a un `var e Estado` del paquete que yo mismo había puesto arriba, con valor cero, así que `e.Termina()` era falso siempre y la afirmación no comprobaba nada. `staticcheck` no la ve (la variable **está** usada) y la suite estaba verde.

**Y M95 se le parece:** la mutación que hacía contar las delegaciones como decididas salió **verde**, porque el caso tenía además un acceso sin tocar y el cierre fallaba por el otro motivo. La afirmación sobre las delegaciones viajaba de gorra en un fallo ajeno.

> **Una afirmación que no falla sola no está probada: está acompañada.**

Las dos son la misma familia que M47 y que la rama que nunca se ejecuta, y las tres se cazan igual: **por cada propiedad, un caso cuyo dato la aísle**, aunque haya que fabricarlo. Aquí fue una campaña con todo decidido menos una delegación.

**Y la tercera hermana, que salió de la verificación externa del acta (02-09-2026): la afirmación que compara una frase con su propia constante.** El test que exige que la UAR enseñe el descargo lo busca así:

```go
frase := cat(t).Traducir("es", "uar.no_consta")
if !strings.Contains(conPendientes, frase) { ... }
```

Los dos lados salen del mismo sitio. Si mañana alguien cambia esa cadena del catálogo por *«has incumplido»*, la pantalla enseña la acusación, el test la busca, la encuentra, y **sale verde**. La mutación tampoco lo caza, por el mismo motivo: mutar la cadena mueve el test con ella. Es hermano de M47, donde cambiar el descargo por la acusación no puso nada rojo, y de la rama que nunca se ejecuta.

> **Una identidad que se mueve en bloque no afirma nada sobre lo que dice: solo afirma que se pinta lo que hay en la tabla.**

Y hay que decir la otra mitad, porque si no la conclusión es que estos tests sobran y no sobran: **lo que sí afirman es que la frase LLEGA a la pantalla**, que es una propiedad real y la que se rompe cuando alguien cambia una plantilla. Lo que no pueden afirmar es su CONTENIDO. Eso lo afirma el test de **forma** que va al lado (`TestNingunDescargoDelActaAcusaEnNingunIdioma`, que exige las dos mitades del patrón, *«esto NO dice…: dice que…»*) y, en castellano, la constante de núcleo. Los dos juntos sí cierran: uno dice que llega, el otro dice qué forma tiene lo que llega.

La regla, para las pantallas que faltan: **un test que busca una cadena en el HTML nunca es la puerta del contenido de esa cadena, aunque lo parezca.** Por cada frase que no puede acusar hacen falta las dos, la de presencia y la de forma, y la de forma tiene que mirar algo que NO salga de la misma tabla que lo pintado.

Para `uar.no_consta` queda cerrado hoy y por un camino que conviene decir, porque no es el obvio: la clave no está entre los 13 descargos que declara `nucleo/acta`, así que el test de forma no la miraba. Lo que la cubre es que ahora está soldada a `acta.descargo.no_revisado` en los dos idiomas (`TestUnaFraseQueViveEnDosClavesSeDiceIgualEnLosDosIdiomas`), y esa sí pasa por la forma. **La cobertura es transitiva, o sea que se pierde el día que alguien deshaga el par.** Por eso el par se declara en una lista con nombre y no en un comentario.


### El barrido de vigencias: lo que se cerro, y lo que la migracion NO verifico (02-09-2026)

Hecho en el mismo dia que la entrada de abajo, y hay que leer las dos juntas.

**Lo que se cerro, con puerta:**

- `vigencias_test.go` contrasta cada `vigencia.desde` de paquete contra las TRES
  fechas de su instantanea y **distingue los dos fallos por su nombre**: cuadrar
  con la de publicacion es la conflacion del invariante 10, cuadrar con la del
  acto es otra cosa. Nacio rojo sobre dato real: cazo `eni`, que llevaba
  `2010-01-29` (publicacion del RD 4/2010) cuando su disposicion final tercera
  dice *«el dia siguiente al de su publicacion»*, o sea `2010-01-30`. Corregido.
- **La herencia silenciosa deja de ser posible**: `vigencia.origen` es
  obligatorio en toda obligacion con reloj, con vocabulario cerrado
  (`heredada` / `propia`) y el valor cero prohibido, porque el lado permisivo
  aqui es justamente el que salia solo.

**Y lo que NO se hizo, que es lo que hay que leer:**

> **La migracion de las 120 obligaciones declaro lo que el dato YA decia. No
> verifico ni una fecha.** Se puso `heredada` donde la fecha coincidia con la del
> paquete (94) y `propia` donde no (26). Eso cierra el futuro —una obligacion
> nueva no puede heredar sin decirlo— y **no dice nada del pasado**.

**Los tres huecos que quedan, con nombre:**

**Y en este orden, que no es el que tenían escrito** (decisión del 02-09-2026): **Cellar va antes que las ITS**. No por dificultad, por alcance: de los 12 marcos del escaparate **cinco son reglamentos de la UE** (AI Act, CRA, DORA, NIS2-UE, RGPD) y es ahí donde vive el error que ya mordió dos veces; las ITS del ENS son cinco Resoluciones y afectan a **un** marco. Y lo primero de esa pieza no es escribir código, es **averiguar si el XML de Cellar trae las tres fechas como dato igual que el BOE o si hay que derivarlas del articulado**: si hay que derivarlas, es otra pieza y se dice con su precio antes de empezarla.

1. **Las 10 instantaneas de Cellar no traen las tres fechas como dato** (las 4
   del BOE si). Viven en la prosa del articulo de entrada en vigor, asi que el
   contraste mecanico **no alcanza a ninguna norma de la UE**. El segundo test
   lo mide y lo publica con un techo declarado, para que el lado ciego no crezca
   en silencio. Arreglo real: que el ingestor de Cellar saque las tres fechas.
2. **Las cinco fechas de las ITS del ENS** (`2016-11-22`, `2018-04-20` x2,
   `2016-11-03` x2) son de instrumentos propios (Resoluciones del BOE) que **no
   estan ingeridos**. Sus citas nombran el BOE-A correcto, y **la fecha en si
   sigue sin poderse reauditar**. Es el caso peligroso al reves, el que la
   verificacion externa nombro: *un acierto que nadie puede reauditar es
   indistinguible de un acierto por suerte*. Ruta concreta: ingerir
   BOE-A-2016-10108, BOE-A-2016-10109 y BOE-A-2018-5370.
3. **Las 26 `propia` no tienen todavia una cita que de SU fecha.** La cita del
   articulo esta, la del instrumento esta, y la del apartado que difiere la
   vigencia no. Es el siguiente escalon de la misma puerta.

### El invariante 10, violado DESPUES de escribirlo, por quien lo escribio (02-09-2026)

El invariante 10 lleva dentro, con estas palabras, el caso que lo trajo:

> «De 8 de julio» y «publicado el 8 de julio» no son lo mismo, y la distancia
> entre las dos son dieciséis días de reloj legal. Las tres fechas de una norma
> de la UE (acto, publicación, entrada en vigor) se copian por separado o no se
> copian.

**Dos semanas despues de escribirlo, `paquetes/ai-act` entro al corpus con la
obligacion del art. 111.4 fechada el `2026-07-24`, que es la fecha de PUBLICACION
del omnibus en el DOUE.** Su art. 4 dice «a los tres dias de su publicacion»: la
vigencia era el `2026-07-27`. Exactamente el error que el invariante nombra,
cometido dentro de la casa, con el invariante ya escrito, por quien lo escribio.

> **Escribir una regla no la implanta.** Es lo que esta entrada demuestra y la
> doctrina sola no puede: un invariante sin barrido es una intencion, y la
> intencion no revisa las fechas de nadie. La diferencia entre el invariante 3 y
> este no es la importancia, es que el 3 tiene linter y el 10 tiene memoria.

**Consecuencia mecanica, y es lo que hay que hacer, no lo que hay que sentir:
BARRIDO PENDIENTE DE TODAS LAS FECHAS DE VIGENCIA YA ESCRITAS.** Los 16 paquetes
con reloj, cada `vigencia.desde` contrastado contra la fecha de ENTRADA EN VIGOR
de su norma, no contra la de publicacion ni contra la del acto. **Si el error
salio una vez por descuido, salio probablemente mas veces**: nadie ha mirado esas
fechas con esta pregunta en la mano.

Precio bajo (son datos ya ingeridos, con sus instantaneas y sus huellas) y va
**antes de escribir mas fechas**, porque cada paquete nuevo aumenta lo que habra
que rebarrer. Puesto en la cola detras de los ocho eventos del CRA.

**Y la forma final es un test, no un barrido:** un barrido a mano encuentra lo de
hoy y no impide lo de manana, que es lo que acaba de pasar con el invariante.

### La primitiva construida y apagada: tres de ocho siguen sin cablear (02-09-2026)

`ventana.Maximo` llevaba semanas construido, con sus tres ramas y sus dorados en
verde, y **no se podía usar desde un paquete**: `nucleo/corpus/dorados.go` sólo
instanciaba cuatro primitivas. Ocho retenciones del CRA lo estaban esperando y
nadie lo había notado, porque **por el lado del motor todo estaba verde**.

> **Una primitiva que sólo se enciende tocando Go es lo contrario del invariante 2.**
> El invariante lo dice desde el primer día («toda norma vive en su paquete de
> datos o no vive») y nadie lo había aplicado a este eje: se aplicaba a los
> identificadores de norma, no a las capacidades del motor.

**Lo que hace este fallo difícil de ver es que no se parece a un fallo.** No hay
test rojo, no hay rama sin ejecutar, no hay número mal: hay una pieza terminada
al lado de otra pieza terminada, sin el cable. Y la pregunta que lo destapa no
está en ninguna de las tres pasadas hasta hoy, porque las tres miran lo que se
acaba de escribir y esto es una ausencia entre dos cosas viejas.

**El barrido de las ocho, medido el 02-09-2026:**

| primitiva | motor | corpus | precio de no tenerla |
|---|---|---|---|
| `puntual` | sí | **sí** | — |
| `periodica` | sí | **sí** | — |
| `continua` | sí | **sí** | — |
| `plazo` (con `Tope` y hitos encadenados) | sí | **sí** | — |
| `maximo` | sí | **sí, desde hoy** | eran las 8 retenciones del CRA |
| **`preaviso`** | sí, con dorados | **NO** | **7 relojes contados y parados**: `psd2` arts. 54.1, 55.1 y 55.3, `mica` arts. 65.4 y 67.4.b, `mdr` art. 75.3 y `data-act` art. 25.2.d. Es la familia G entera del censo |
| **`secuencia`** | sí | **NO** | ninguno hoy. La familia A la midió y la **descartó** a favor de `plazo` con hitos encadenados, así que hoy no hay reloj contado que la pida |
| **`observacion`** | sí | **NO** | sin medir. La usa el ejemplo de SOC 2 tipo II (muestreo sobre ventana), que es estrato referencial y no tiene relojes contados |

**Cómo se lee esa tabla, que es lo que la hace útil**: las tres apagadas no son
el mismo caso y no cuestan lo mismo. `preaviso` tiene demanda contada y esperando,
así que es deuda que ya vence. `secuencia` está apagada **con razón** y lo que
falta ahí no es cable, es decidir si se borra. `observacion` no tiene demanda
contada porque su familia vive en los referenciales, que es justo donde el censo
no puede contar (frontera legal), así que su precio es desconocido y decirlo es
la respuesta.

**No se arregla ahora**, por decisión: `preaviso` entra con la familia G y con la
lección aprendida en `maximo`, que abarató el cableado a esquema + linter +
ejecutor. Lo que sí entra hoy es **la pregunta fija en la pasada 1**, que es lo
que impide que la cuarta primitiva apagada se descubra igual de tarde.

### La guarda que vivía en el navegador: la excusa de la UAR (01-09-2026)

**P1 de una revisión adversaria sobre una casilla ya marcada, y los tres casos se midieron aceptados en verde antes de tocar nada.** `Campana.Excusar` validaba quién, motivo y cuándo, y **del rango no miraba nada**. La única validación real vivía en un `min="1" required` de la plantilla.

> **Una validación que solo vive en el cliente no es una validación, es una sugerencia.** Un `curl` la ignora, y lo que hay detrás es un registro append-only.

Tres capas, medidas:

| capa | qué hacía |
|---|---|
| `superficies/uar` | `desde, _ := strconv.Atoi(...)`: el error se traga y **ausente, vacío e ilegible se vuelven el mismo cero** |
| `nucleo/accesos` | `Excusar` no miraba ni que las líneas existieran, ni que fueran ilegibles, ni cota superior |
| `uar.html` | `min="1" required` — o sea, el navegador |

**Las dos consecuencias, y la segunda es la cara:**

- Una excusa `{0,0}` nacida de un `Atoi` tragado **entra al ledger append-only para siempre** como un hecho que no excusa nada. Y es la contradicción de la casa: en `decidir` el código se cuidaba de registrar **antes** de anotar, con el comentario *«una decisión mal formada quedaría en un registro para siempre»*, y esa misma guarda no existía para las excusas.
- `desde=1&hasta=999999` con **un** motivo cubría todas las líneas ilegibles de la campaña de una tacada y desbloqueaba el cierre. **Cumplía la letra** de *«quien la excusa y por qué»* **y derrotaba lo que esa letra protege**, que es la responsabilidad *por línea*.

**La regla:** toda línea del rango tiene que ser ilegible en el censo sellado, y el rango tiene que tocar al menos una que no estuviera ya excusada. **Una excusa que no excusa nada no es un hecho.** Y en la superficie, `numeroObligatorio` y `numeroOpcional` son **dos funciones y no una con valor por defecto**, porque son dos preguntas: en un campo obligatorio la nada es un error; en uno opcional la nada es el defecto. Meterlas en una es por donde se cuela el cero degenerado.

**Lo que esto añade al método, y es lo que vale:** las dos formas de la nada del invariante 8 tenían una tercera hermana que no estaba nombrada. `strconv.Atoi` con el error descartado convierte **tres** cosas distintas en el mismo cero:

| forma | qué es | qué debe pasar |
|---|---|---|
| campo ausente | la nada | según el campo: defecto o error |
| campo presente y vacío | la otra nada | lo mismo que la anterior, y **dicho** |
| **campo presente y no interpretable** | **un dato que hay y no se entiende** | **siempre error, nunca la nada** |

> **Confundir «no lo has puesto» con «lo has puesto y no lo entiendo» es el invariante 8 en la frontera de entrada.** La primera es una decisión de diseño; la segunda es un dato que se está inventando.

Y una consecuencia de proceso: **la casilla estaba marcada**. La reviso una pasada adversaria externa el mismo día. Una casilla marcada no es una casilla cerrada a la revisión.

### La familia de la directiva inerte, cerrada con puerta (30-08-2026)

**Continuación directa del apunte de arriba, y aquí es donde se cierra.** La regla que quedó escrita —*«antes de escribir una supresión, mirar qué herramienta corre de verdad»*— es una regla de memoria, y una regla de memoria no es una puerta: se cumple mientras alguien se acuerde. Y el propio apunte dejaba la prueba de que la memoria no basta, porque al lado del `//nolint:gosec` que costó el rojo había una **segunda directiva inerte**, un `//nolint:staticcheck` escrito meses antes que nadie había vuelto a mirar.

**Lo que la hace peor que un comentario cualquiera:** una directiva de supresión no sólo no silencia nada, además **afirma** que alguien miró el hallazgo y decidió que era un falso positivo. Quien la lea da por revisado lo que no se revisó. Es *la rama que nunca se ejecuta*, escrita en un comentario.

**La regla, mecánica:** este árbol sólo admite directivas de supresión **de herramientas que CI ejecuta**. Hoy son tres, y la lista **no se declara**: `supresiones_test.go` la lee de los workflows, por el mismo motivo por el que `comprobar.sh` lee las puertas.

**La decisión sobre la segunda directiva no se dejó pendiente: o entra la herramienta, o sale la anotación.** Entró staticcheck, que es barato (un paso de workflow) y es el linter que más defectos reales caza en Go. Su primera pasada encontró **16 cosas**, y el reparto dice bastante:

| qué | cuántas | qué era de verdad |
|---|---|---|
| símbolos muertos (U1000) | 6 | **tres con un comentario que describía una guarda que nadie invocaba** |
| bucles que son un `append` (S1011) | 2 | ruido, arreglado |
| invisibles crudos en literales (ST1018) | 2 | mejor con `\u202e`: es un test **sobre** caracteres invisibles |
| deprecación (SA1019) | 1 | la directiva inerte, ahora en la sintaxis que sí se lee |
| comparación consigo misma (SA4000) | 1 | deliberada (comprobar que `UIDDe` es pura), ahora dicha |
| estilo de mensajes de error (ST1005) | 5 | choca con la convención de la casa: se resta en `staticcheck.conf` con su porqué |

**Los tres símbolos muertos con comentario son el hallazgo, no las 16.** Un helper que decía existir *«para que un fallo al armar los certificados de la PKI de prueba no se leyera como un fallo de verificación»* y al que nadie llamaba. Un `tirar()` del acumulador de ingesta, documentado como *«se usa al entrar en una zona cuyo texto no es del artículo»*, y ninguna zona lo usaba. Y una constante de procedencia cuyo comentario decía *«se escribe una vez y la usan las dos puertas de este fichero»* mientras la ruta se repetía **a mano** doce líneas más abajo, que es exactamente lo que la constante existía para impedir.

> **Un símbolo cuyo comentario describe una guarda que nadie invoca es una guarda que no existe.** Es la misma frase que la rama muerta y que la directiva inerte, dicha en el tercer sitio donde aparece.

Y una cuarta pieza, de otra familia: `cmd/plazum` tenía una **segunda implementación, muerta, de la conversión de régimen legal** (`naturales`/`hábiles` → `ventana.Regimen`). La viva es `corpus.RegimenDe`. Dos traducciones de la misma regla de cómputo pueden separarse sin que nada se ponga rojo, y una de ellas ni siquiera se ejecutaba para notarlo.

**La segunda mitad, que es la que cierra el agujero de proceso de verdad.** El arreglo del día anterior fue escribir `gosec` a mano en `comprobar.sh`, y **duró un día**: la siguiente herramienta llegó esa misma tarde. El hueco nunca fue gosec, era *«cualquier paso bloqueante de CI que no sea una `puerta()`»*. Así que `comprobar.sh` ya no nombra ninguna: **extrae de `ci.yml` toda invocación `go run <módulo>@<versión>` y las corre todas**, con `HERRAMIENTAS_ESPERADAS` cuadrando en los dos sentidos igual que `PUERTAS_ESPERADAS`. Si CI gana una herramienta, el lazo local la corre sin tocarlo.

> **Nombrar la herramienta que falló deja el agujero abierto para la siguiente.**

Y el arreglo trajo de regalo lo que faltaba: `govulncheck`, que también es bloqueante y que el lazo local tampoco corría.

**Las puertas y sus rojos demostrados**, todos sobre copia y con el estado de compilación comprobado aparte:

- **M78** devuelve el `//nolint:staticcheck` inerte → `adaptadores/tsa/tsa_test.go:687 tiene una directiva de golangci-lint, que CI NO EJECUTA`.
- **M79** saca staticcheck de `ci.yml` → las dos `//lint:ignore` legítimas se quedan huérfanas y salen las dos, **y además** el lazo local se queja de que ahora hay 2 herramientas y declara 3.
- **M80** deja un `#nosec` sin regla y sin motivo → rojo. Sin regla se callan **todas** las comprobaciones de esa línea, no la que molestaba.
- **M81** nombra gosec a mano dentro de `comprobar.sh` → rojo: la invocación sale de `ci.yml` o no sale.
- **M82** deja `HERRAMIENTAS_ESPERADAS` vieja → rojo en el test y en el propio script.
- **M83**, la que importa, **en el shell donde corre**: un `var muertaDePrueba = 1` que sólo ve staticcheck, y `./comprobar.sh` entero termina en `COMPROBACION EN ROJO` nombrando la herramienta y su sintaxis.

**Y el control negativo va en las dos direcciones**, porque el fallo probable de este detector no es dejar pasar una directiva: es **acusar a la prosa que las menciona**. En este árbol hay tres comentarios que hablan de `#nosec` sin serlo, así que el detector exige que el texto **empiece** por la directiva, y los tres están en la tabla del control negativo copiados tal cual. Un detector con falsos positivos se acaba desactivando, que es la única forma segura de perder una puerta.

### El demo escribía un fichero que el producto luego rechazaba (30-08-2026)

**Salió al enchufar `plazum escalado` al alcance del demo, y es de los primeros cinco minutos del comprador.** `paquetes/demo-empresa/alcance.json` — el fichero que **`plazum demo` escribe** y que la ayuda de `cmd/plazum/alcance.go` **nombra por su ruta** — no cargaba: traía `notas_de_las_fechas`, un campo que nadie había declarado en la estructura, y la decodificación estricta lo rechazaba entero.

O sea: el producto producía un fichero que el propio producto se negaba a leer, y el camino era `plazum demo` → `plazum calendario --alcance ...` → error duro. Nadie lo había recorrido desde que la decodificación estricta entró.

**El campo era legítimo**, y ése es el matiz: JSON no tiene comentarios, y una nota dentro del fichero explicando que las fechas son desplazamientos (`-45d`) y no fechas fijas es exactamente lo que quiere leer quien lo abre. Así que el arreglo no fue borrarla: fue **declararla y además imprimirla**, que son las dos mitades. Con campo y sin imprimir habría sido un huérfano de la segunda forma.

**Lo que deja como método:** la decodificación estricta convierte un campo huérfano en un fallo duro, que es lo que se quería — pero **el barrido de huérfanos que se hizo entonces miró los modelos de pantalla y los perfiles, y no los ficheros de datos que el propio producto genera**. Un fichero que escribe el binario es una entrada más, y hay que recorrerla como tal. La comprobación es de una línea y ahora existe: un test carga el alcance del demo y exige que la nota se imprima.

### El campo declarado que nadie interpreta (30-08-2026)

**`Escalon.Tras` llevaba desde el primer día en el formato y el linter sólo comprobaba que no estuviera vacío.** Nadie parseaba su valor. Los 53 escalones del corpus resultan estar todos bien escritos — se midió antes de escribir el parser, no se supuso — pero **eso era suerte, no una propiedad**: el primer `P60D_ants` habría salido el día del incidente, que es el único momento en que nadie quiere descubrir un error de formato.

**Es una dimensión nueva del barrido de huérfanos**, y conviene verlas juntas porque son la misma pregunta hecha en tres sitios:

| forma | qué pasa | quién la caza |
|---|---|---|
| el dato no llega al tipo | `encoding/json` lo descarta | cerrada: `nucleo/estricto` |
| el tipo lo tiene y nadie lo pinta | viaja hasta la pantalla y muere ahí | barrido único, hecho |
| **el campo se lee y su VALOR no lo interpreta nadie** | **se acepta cualquier cosa y se descubre al usarlo** | **ésta** |

La regla: **un campo cuyo valor nadie parsea es un campo sin semántica vigilada**. Si el formato dice que ahí va una duración, una fecha o un identificador de una forma concreta, eso se comprueba **al cargar** y no al usarlo. Es la misma doctrina de *comprobación al nacer* que el escalado hacia una figura que nadie ocupa.

### Subfamilia: la rama que nunca se ejecuta (30-08-2026)

**La nombra M47, que salió VERDE la primera vez.** La mutación cambiaba el descargo de un campo pendiente — *«esto NO dice que no exista: dice que en tus respuestas no aparece»* — por una acusación en toda regla, y la suite no se inmutó.

No era un fallo del código: **era que ninguna entrada llegaba a esa rama**. Con las plantillas del corpus, los tres campos que pide el incidente los contesta el incidente, así que no había ni un campo en estado *«no consta»* y la frase del descargo no se construía nunca. La mutación no lo ve porque **no hay nada que romper**.

> **Un descargo que ninguna entrada alcanza es un descargo que no existe.**

**La regla, para que se pueda aplicar sin pensarla:** *toda rama de acusación o de descargo exige un **control positivo** que la recorra, con dato sintético si el corpus no la alcanza.* En el caso: una plantilla sintética que le pide al incidente **todo** su vocabulario, y un incidente sin clasificar, para que `clasificacion_vigente` salga como *no consta*. Y con ella, la afirmación que faltaba: *si ningún campo salió como «no consta», el test falla* — porque cero campos en esa rama no se puede seguir leyendo como cero fallos.

**Es la vacuidad con otra cara.** La familia ya conocida es *«el patrón `-run` no casó con nada y `go test` salió 0»*; ésta es *«el caso existía y ninguna entrada lo activaba»*. Y va a reaparecer **en cada pantalla que distinga «consta / no consta»**: el acta, el expediente, la UAR y la línea de tiempo del incidente tienen todas esa bifurcación, y en todas la rama cara es la que menos datos naturales tiene.

### Hábito: un hallazgo grande se mide dos veces antes de creérselo (30-08-2026)

Del hallazgo de los orígenes de plantilla. La primera medición dio **39 orígenes colgando** y era falsa: contaba como rotos los sufijos (`.ultimo_hito`) y los globs, que son **gramática propia mal leída**, no defectos. La segunda, buscando para cada origen *el prefijo más largo que sí es un id real*, dio **6**, que eran los de verdad.

**Un 39 falso habría sido tan inútil como un cero**, y peor: habría gastado el crédito del siguiente hallazgo. La regla: cuando el número sorprenda por lo alto, **la primera hipótesis es que la comprobación no entiende el formato**, no que el corpus esté roto. Se mide otra vez por otro camino, y sólo entonces se cuenta.

### La predicción del escalado, comprobada antes de construirlo (29-08-2026)

**Confirmada, con números.** La tabla de *alcanzabilidad, no existencia* preguntaba: *«escalado | la obligación declara escalones, ¿hay alguien a quien escalar en algún alcance posible?»*. Se midió antes de escribir una línea del envío, que es el orden que pedía la propia entrada — **comprobación al nacer, no al fallar**.

El corpus tiene **53 escalones** en diez paquetes, y sus destinatarios son **14 nombres de rol distintos**. No hay vocabulario de roles en ninguna parte: ni en el formato del paquete, ni en el código, ni en SCIM, que tiene usuarios, grupos y jerarquía de mando pero ningún concepto de *«el responsable de seguridad»*. El linter comprueba que `a` no esté vacío y nada más.

**Y entre los 14 hay tres nombres para lo que parecía el mismo señor:**

| nombre | paquetes |
|---|---|
| `responsable_seguridad` | eidas2, ens, iso27001, nis2-ue (14 escalones) |
| `responsable_de_seguridad` | demo-empresa (1) |
| `responsable_seguridad_informacion` | nis1-es (1) |

> **CORRECCIÓN del 30-08-2026, y es del fondo, no de la forma.** Esta entrada decía *«al menos dos que son el mismo señor»*. **Al ir a las fuentes, no lo son.** El RD 43/2021 dedica su art. 7 entero a definir el **Responsable de la seguridad de la información**, que es una figura suya y no la del ENS; el propio art. 7.4 dice que sus funciones *«podrán compatibilizarse»* con las del Responsable de Seguridad del ENS, lo cual sólo tiene sentido **si son dos**. El único alias de verdad era el de `demo-empresa`, que es un paquete de demostración y no tiene norma detrás.
>
> La lección va con la entrada porque es la que la regla nueva existe para evitar: **el parecido de dos nombres no es evidencia de nada**, y yo mismo lo di por bueno el día anterior, en la misma nota que decía que hacía falta una cita para afirmarlo. Un análisis por el nombre habría fundido dos figuras que la ley separa.

Los otros dos parecidos tampoco lo son: `responsable_seguridad_producto` (CRA) y `responsable_de_seguridad_de_pagos` (PSD2) son de otros sujetos, y colapsarlas sería el error contrario.

**Qué pasa el día que el escalado envíe.** Un correo a `responsable_de_seguridad` no llega a nadie, porque la figura que la organización asignó se llamaba de otra manera. **No da error**: da un escalón que no escala, que es la forma exacta de la familia — la regla existe, se ve, y nadie puede satisfacerla. Y en un producto de continuidad, un aviso que no llega se parece mucho a no tener aviso.

**El arreglo, y por qué NO se hace en el código.** La tentación es una lista cerrada en Go, como `clase_e2e` o `TipoRecurso`. No vale: aquellas son vocabularios **del producto** (cuánta profundidad tiene una obligación, qué recurso se observa) y un rol es una **figura de la norma** — `responsable_de_la_ac` es de eIDAS2 y `delegado_proteccion_datos` del RGPD. Cerrarlos en Go pondría normas en el código, que es el invariante 2 al revés.

Va donde van las entidades y las preguntas: **el paquete declara sus roles**, cada uno con su descripción y **su cita**, y el linter exige que todo `escalado.a` sea uno de ellos. Con eso el escalón hacia un nombre que nadie ha declarado se cae al cargar, y la organización tiene una lista finita de figuras a las que poner nombre en vez de catorce.

**Lo que bloquea hacerlo hoy, dicho para que no parezca pereza:** son **doce citas** de ocho normas distintas (ENS art. 13, RGPD art. 37, DORA, CRA, eIDAS2, MDR, PSD2 y el AI Act), y el invariante 10 exige verificarlas contra fuente primaria **en el momento de escribirlas**, con su fecha. Inventarlas para que el linter pase sería exactamente lo que ese invariante existe para impedir. Es trabajo de corpus, no de motor, y entra como tal.

**CERRADO el 30-08-2026**, con las citas verificadas contra instantáneas con huella ingestadas ese mismo día (ENS, RD 43/2021, RGPD, DORA, NIS2, MDR, CRA, eIDAS, RDL 19/2018 y el AI Act). El resultado, y es más interesante que el arreglo:

| | |
|---|---|
| **27 figuras** declaradas, **53 escalones** reescritos | |
| **10** las nombra la norma | ENS art. 13.2 (tres), RGPD art. 37.1, DORA arts. 5.2 y 6.4, NIS2 art. 20.1, RD 43/2021 arts. 6 y 7, MDR art. 15 |
| **17** las propone plazum | porque **sus normas no nombran a nadie**: el CRA, eIDAS, el AI Act y el RDL 19/2018 no traen ninguna figura interna, y los dos paquetes referenciales no tienen texto que citar |

**Dos cosas que sólo se ven al mirar las fuentes una por una:**

- **NIS2 no define ningún responsable de seguridad.** Barrido de sus 49 artículos: cero coincidencias. El paquete escalaba a `responsable_seguridad`, **un nombre tomado del ENS**, que es exactamente la unificación sin fuente que esto venía a impedir. Ahora es `nis2.responsable_de_seguridad`, propuesta, y su justificación lo dice.
- **El MDR sí la define, y no se llamaba como el paquete decía.** El art. 15 lleva por título *«Persona responsable del cumplimiento de la normativa»* y su apartado 3.d) le encarga las notificaciones del art. 87. El paquete la llamaba *«responsable de vigilancia»*: un título que no salía del artículo, que es la familia de la prosa otra vez.

**Y la fuente primaria contestó ella sola a la pregunta de si el mismo señor puede tener dos figuras.** El art. 7.4 del RD 43/2021 dice que las funciones del responsable de la seguridad de la información *«podrán compatibilizarse con las señaladas para (...) el Responsable de Seguridad del Esquema Nacional de Seguridad»*. **Compatibilizar dos figuras en una persona no es fundirlas en una figura**, y por eso las dos se declaran aparte y el mapeo persona→figuras es del cliente.

Cuatro mutaciones, todas sobre el corpus: el nombre de antes del vocabulario (`M50`), un paquete escalando a la figura de otra norma (`M51`), una propuesta que se queda sin justificación (`M52`) y una figura que nadie usa (`M53`). Las cuatro rojas.

**Y las dos puertas viejas cazaron el fichero de test nuevo**, que es la mejor señal de que siguen despiertas: el invariante 2 por escribir `ens.` en un caso, y el invariante 9 porque `FiguraPropuesta` contiene `Propuesta`, que es un nombre de la IA. Al renombrar por lo segundo salió mejor de lo que estaba: el campo se llama `origen`, que es masculino, así que sus valores concuerdan igual que los de `origen_del_intervalo` — **`propuesto` en los dos sitios, para la misma idea**.

### Doctrina: un resultado negativo es un entregable (29-08-2026)

**El ENI lo estrena y la regla vale para todo el corpus.** El Real Decreto 4/2010 se recorrió entero, sus 42 artículos, buscando lenguaje temporal. **No tiene ni un reloj periódico**, y eso no es una ausencia de trabajo: es el trabajo.

Un paquete vacío y un paquete que dice *"aquí no hay relojes, y éste es el barrido"* se leen como cosas distintas. El primero dice **"falta por hacer"** y hace que alguien vuelva a buscar dentro dentro de seis meses; el segundo **cierra la pregunta**.

**La forma del entregable, para repetirla:** qué se buscó (las palabras), dónde (el ámbito exacto: los 42 artículos), qué hay **en su lugar** (la tabla: deberes permanentes, deberes de establecimiento, un plazo transitorio agotado en 2011, deberes del Estado sobre el propio Esquema) y qué se decidió no escribir y por qué (la DT única, las NTI).

**Es el mismo principio que las marcas de honestidad del censo**, y conviene decirlo una vez en general: lo que un corpus informa no es sólo lo que contiene, es también **lo que afirma no contener**. Un corpus que sólo sabe decir *"esto está"* deja a su lector distinguiendo entre *"no lo hay"* y *"no lo he mirado"*, y esa distinción es la que decide si se fía.

**Cuándo se aplica:** cuando el barrido de una norma o de un tramo termina con menos relojes de los que su tamaño hacía esperar. Entonces el resultado se escribe en el `LEEME.md` del paquete con las cuatro partes de arriba. **Un paquete cuyo LEEME dice sólo "esqueleto" no ha contestado.**

### Familia: todo campo de prosa libre es una puerta de atrás de la frontera legal

**El caso, y no es teórico.** Al justificar el intervalo del punto 6.7.3 del anexo de 2024/2690, el argumento propuesto fue *"el sector de medios de pago lleva años exigiendo la revisión del conjunto de reglas de cortafuegos cada seis meses"*. Eso es **criterio de PCI DSS**, y el linter no lo veía: el límite de la frontera legal mide **longitud, no procedencia**. Un campo de 200 caracteres pasa igual si lleva dentro un razonamiento propio o el criterio de un catálogo de pago.

**Por qué es familia y no un caso.** Porque la forma natural de justificar un número es apoyarse en la práctica reconocida, y **media práctica reconocida vive en catálogos privativos**. Va a reaparecer cada vez que plazum ponga un intervalo, y también fuera de los intervalos: en la `ayuda` de un atributo (*"esto es lo que pide el control 5.35"*), en el `titulo` de una plantilla, en la `nota` de un hito, y **sobre todo en la IA cuando llegue**, porque un modelo generando la justificación de un número irá derecho a la práctica que mejor conozca, que es la de los catálogos más difundidos.

**Las dos capas, y hacen falta las dos:**

| capa | qué cierra | qué NO cierra |
|---|---|---|
| **lintable** (`nucleo/corpus/frontera_prosa.go`, hecha el 28-08-2026) | la prosa de un paquete **no nombra** un marco de estrato referencial o delegado ajeno. Lista negra derivada del corpus por la CLASE de cada paquete, no a mano | la **paráfrasis anónima**, que es justo el caso de arriba: no nombra PCI DSS, dice *"el sector de medios de pago"* |
| **humana** | la pasada de coherencia pregunta, por cada número, si el argumento **se sostiene sin el apoyo fantasma**: quitando la frase que remite a la práctica ajena, ¿queda un argumento? | nada mecánico; es lectura, y por eso está declarada en vez de supuesta |

**LAS TRES ENTRADAS VISTAS HASTA HOY**, y la lista importa más que cualquiera de ellas: se creía que la puerta era la justificación, y no era una puerta, era una familia de puertas.

| # | entrada | ejemplar | quién la cierra |
|---|---|---|---|
| 1 | **la justificación** | 6.7.3: *"el sector de medios de pago lleva años exigiendo la revisión cada seis meses"* | linter de prosa (si nombra) + pasada humana (si parafrasea) |
| 2 | **el TÍTULO** | 12.2.3: el anexo titula el 12.2 *Gestión de activos* y la ficha lo llamó *"política de manipulación de activos"*, que es el nombre del control homónimo de un catálogo privativo | **sólo la pasada humana**: el linter no lo ve porque no nombra el marco |
| 3 | **el apoyo sin nombre** | 8.1.3 (una curva de decaimiento de tasa de clic sin origen) y 6.3.3 (*"el fabricante reedita su guía"*, sin decir cuál) | `fuentes_del_intervalo` + la pregunta de la pasada |

Y quedan las que aún no se han visto y esta lista predice: la `ayuda` de un atributo, el `titulo` de una plantilla, la `nota` de un hito, y **sobre todo la IA cuando llegue**.

**REGLA NUEVA DE LA PASADA HUMANA, por la entrada 2:** en un paquete de estrato **transcrito**, el título de una obligación **se deriva de las palabras del propio anexo**; apartarse de ellas exige justificación explícita. Un título es el nombre con el que el cliente va a hablar de la obligación durante años, así que un nombre prestado de un catálogo privativo se instala más hondo que una frase de justificación: acaba en su plan de acción, en su acta de comité y en su hoja de auditoría.

**Y NO SE HA HECHO LINTABLE, con la medida delante.** Se midió sobre las 31 obligaciones de paquetes transcritos del corpus: **25 con solape claro entre el título y su `texto_legal`, 6 flojas o sin solape**. Y las 6 son **legítimas**, una por una: `ens.its_incidentes.notificacion_al_ccn` ("Notificar el incidente al CCN") tiene cero palabras en común con su texto legal y es un resumen perfecto; los dos de eIDAS2 anteponen *"Prestador CUALIFICADO / NO cualificado"*, que es precisamente lo que distingue las dos obligaciones. Un linter con **19 % de falsos positivos se desactiva solo la primera semana**, y este repositorio ya tiene escrito por qué (`nombraA` y su lista de genéricos). La regla se queda en la pasada humana, con su pregunta fija: **¿de dónde salen las palabras de este título?**

**Hecho el 28-08-2026, capa humana instrumentada:** el campo `fuentes_del_intervalo` existe en `Temporalidad`, es opcional, y cada entrada tiene que **identificar un documento**: mínimo 12 caracteres y al menos un número (de norma, de publicación, de artículo o el año). *"NIST"* es una organización y no pasa; *"NIST SP 800-92, sección 4.3"* sí. Los dos apoyos fantasma de arriba caen los dos por esa regla. Se comprueba **en los tres orígenes del intervalo**, no sólo en `propuesto`: la mutación que lo restringió a `propuesto` compila y deja `suelo_legal` y `fijado` en verde, que es el camino que nadie recorre y por eso el que se acaba usando.

Lo que la regla mecánica **no** caza, dicho para que no se confunda con una garantía: una referencia inventada con pinta de real. Contra eso no hay linter; hay pasada humana. El campo sólo quita del camino lo que ni siquiera pretende serlo, y sobre todo **cambia la pregunta**: de *"¿de dónde sale esto?"*, que obliga a leer cada frase buscando un eco, a *"¿por qué este argumento no tiene fuente?"*, que se contesta mirando.

**Y la frontera que esta familia NO cruza nunca**, dicha aquí porque es donde se buscará: **`texto_legal` no se mira**. Ahí va transcrito lo que dice el boletín, y el boletín remite a normas privadas continuamente (el ENS remite a ISO/IEC 27001). Aplicar la regla al texto legal sería **censurar la ley** para cumplir una regla nuestra.

### Subfamilia: el número cuelga de la acción obligada, nunca de otro verbo de la frase

**No es lintable, y por eso va aquí con su ejemplar.** El punto 6.9.2 del anexo de 2024/2690 dice que las entidades aplicarán medidas para detectar o impedir el uso de programas maliciosos y que velarán, cuando proceda, por que **se actualicen**. El intervalo propuesto colgó del verbo *actualizar*, y así leído la obligación dice que **el antimalware se actualiza cada tres meses**, que autoriza un infracumplimiento con cara de control: lo que el punto pide es **comprobar la cobertura**, y la actualización de firmas es continua y no trimestral.

**La pregunta fija para la pasada de coherencia**, que es lo que queda de esto: **¿de qué verbo cuelga este número?** Un punto con tres verbos (*aplicarán*, *velarán*, *se actualicen*) admite tres relojes distintos y sólo uno es el que la norma exige. Escoger mal no da error en ningún sitio: da una obligación que se cumple sola y una casilla verde.

**Tercera aparición, y con ella pasa a pregunta fija (29-08-2026).** Las tres, por orden:

| dónde | los verbos | de cuál colgaba el número |
|---|---|---|
| **6.9.2** de 2024/2690 | *aplicarán* / *velarán* / *se actualicen* | del último: decía que el antimalware **se actualiza** cada tres meses |
| **art. 28.3** de DORA | *mantendrán el registro* / *comunicarán a las autoridades* | cazado al escribirlo: el año cuelga de **comunicar**, y mantener el registro es continuo |
| **arts. 35.1 y 67.1** de MiCA | *dispondrán en todo momento* / *revisarán el importe* | la forma más fina: los dos verbos están en **apartados distintos del mismo artículo** |

El de MiCA añade el matiz que faltaba: **la trampa tiene dos caras y la segunda se olvida**. Una es colgar el número del verbo que no toca — *"basta con tener fondos propios una vez al año"*, que es autorizar un infracumplimiento con cara de control. La otra es **dejar fuera del corpus el deber sin número** porque no tiene fecha: lo permanente no se calla, sale como `continua`. Es lo que decide D-17 y lo que hacen ya el art. 72.2 del AI Act y el art. 9.1 del ENI.

Es hermana de *alcanzabilidad, no existencia*: las dos producen una obligación que **existe, se ve y no sirve**.

### El ejemplar de «toda puerta nace con su fallo demostrado» (28-08-2026)

**Queda como el caso de referencia**, porque es el único en el que se ha podido MEDIR lo que la regla evita en vez de argumentarlo.

Al renombrar el módulo de `plazum` a `github.com/marcosmatalab/plazum`, cinco puertas llevaban la ruta vieja escrita a mano. **Dos se quedaron verdes vigilando el vacío**, y no es una hipótesis: se comprobó devolviendo `esElPuertoDeIA` a la ruta cableada, con el fichero compilando:

```
TestElNucleoNoConoceLaIA            --- PASS   <- la PUERTA, verde y ciega
TestElDetectorDeImportDeIAFunciona  --- FAIL   <- su control negativo
```

La puerta del invariante 9 pasa tranquilamente porque su detector ya no reconoce ningún import: **no hay nada que detectar cuando se busca una cadena que ya no existe**. Igual la frontera del latido.

**Lo único que las cazó fue su control negativo.** No fue una formalidad de la primera escritura: fue el único testigo. La lección operativa es que un control negativo no vale sólo el día que se escribe la puerta — vale sobre todo el día, meses después, en que el mundo se mueve por debajo y la puerta deja de mirar sin dejar de pasar.

### Cómo se responde a un escáner (28-08-2026)

**Queda como estándar**, del primer hallazgo de CodeQL sobre el repositorio ya público (`go/bad-redirect-check`, medium):

1. **Arreglarlo.** Un hallazgo abierto en la pestaña de Security es la misma clase de ruido normalizado que un rojo permanente en CI, y este repositorio ya sabe lo que cuesta (gosec, cinco commits).
2. **No inflarlo.** Se escribió en el commit y en el código: *«`Base` la pone el operador en su propio arranque, no llega de una petición, así que no hay atacante remoto. Quien escribe `--base=//evil.com` se lo hace a sí mismo, y decir otra cosa sería inflar el hallazgo»*. Lo que sí era: **el mensaje de error prometía un contrato que el código no verificaba**, que es M14 con otra cara.
3. **Convertir la lista en propiedad.** El test tiene dos mitades y sólo la segunda envejece bien: la primera recorre doce formas de escribirlo (y una lista al lado del test sólo prueba lo que su autor pensó); la segunda afirma que **de todo `Base` que el constructor acepte, el `Location` de la raíz sale local**. La mutación puso en rojo las dos, y que la propiedad se rompa por su cuenta es lo que hace que el test no dependa de que su lista esté completa.

### Clase nueva: el campo huérfano (28-08-2026)

**El caso.** Los tres perfiles de arranque traían un `fechas_porque` — el texto que dice cuáles fechas son de ejemplo y cuáles las fija la norma — y **ese campo no existía en la estructura**. Estaba escrito, revisado y commiteado, y no lo había leído nunca nadie. Lo sacó la decodificación estricta en su primera ejecución.

**Por qué es clase y no caso.** Un campo huérfano tiene dos formas y las dos son silenciosas:

| forma | qué pasa | quién la caza |
|---|---|---|
| **el dato no llega al tipo** | `encoding/json` lo descarta al cargar | ya cerrada: `nucleo/estricto` |
| **el tipo lo tiene y nadie lo pinta** | el dato viaja entero hasta la pantalla y muere ahí | **abierta** |

La segunda es la que queda. No hay error, no hay descarte, no hay aviso: el campo se rellena con cuidado, viaja por el modelo y ninguna plantilla ni ningún `print` lo consume.

**Barrido único, no puerta permanente.** Por cada campo de los modelos de pantalla (`nucleo/pantalla`) y de los perfiles, se pregunta si alguna plantilla o algún `print` lo lee. Los huérfanos **o se imprimen o se borran**. No se convierte en puerta porque un campo puede estar legítimamente sin pintar durante una etapa (el modelo va por delante de la superficie), y una puerta que hay que silenciar a mano cada dos commits se acaba silenciando siempre.

**Barrido hecho el 29-08-2026.** 18 tipos de `nucleo/pantalla` y de los modelos de perfil y alcance, campo a campo, contra todo `.go` fuera de `nucleo/pantalla` y todas las plantillas. **Un huérfano: `Calendario.HitosConVigenciaIlegible`.** Se contaba en la derivación, se usaba en la ley de conservación y no se imprimía en ningún sitio — o sea que el bloque titulado **«LA CUENTA, ENTERA» tenía un cubo de menos**, que es M14 otra vez: un título que afirma exhaustividad es una claim. Resuelto imprimiéndolo, que era la respuesta correcta de las dos: borrarlo habría roto la conservación.

Que el barrido encontrara exactamente uno es también el argumento de por qué no se hace puerta: el coste de la pasada es de minutos y su cosecha es pequeña y no recurrente.

### El protocolo de los revisores a ciegas, como método

**Queda fijado** para toda clasificación legal discutible: **N revisores independientes reclasifican desde el texto, sin ver lo escrito**, y se contrasta. Estrenado el 28-08-2026 sobre las 23 cadencias del corpus: seis revisores, **23 de 23 coincidiendo**, cero discrepancias.

**Su ejemplar es el anexo I.1 del ENS**, que dice *"Anualmente ... deberá re-evaluarse la categoría"* **sin decir "al menos"**. La clasificación `suelo_legal` se sostiene sobre una lectura razonada — *un deber de re-evaluar con una frecuencia fija un intervalo máximo, porque re-evaluar antes no puede incumplirlo* — que va **escrita en la propia cita**, no supuesta. Ése es el nivel: si la lectura no se puede escribir en una frase que un jurista pueda discutir, la clasificación no está hecha.

Lo que hace útil el protocolo no es la coincidencia, es **dónde no coinciden**: el único caso marcado con confianza media por un revisor fue exactamente el que ya estaba marcado como interpretativo. Un protocolo que sólo confirma no aporta; éste señaló el mismo punto blando por su cuenta.

## P1

> **Los numeros son estables a proposito.** Hay codigo que los cita (`P1 10` en
> `nucleo/expediente/expediente.go`, `P1 12` en su `hostil_test.go`), asi que
> renumerar en bloque rompe referencias que nadie va a ir a arreglar. Cuando dos
> frentes numeran a la vez y chocan, **el bloque que llego despues se mueve al
> final**, y no se toca nada mas. Asi paso con el 16 y el 17 del autoservicio,
> que son ahora el 26 y el 27.

### Del corpus (frente de autoria, 25-08-2026)

1. ~~**El limite de texto solo vigila `texto_legal`.**~~ **CERRADO el
   26-08-2026, y en dos mitades.** La primera se cerro antes: el limite paso a
   mirar los veinte y pico campos de texto del formato, con tres techos
   (`LimiteTextoReferencial`, `LimiteCitaReferencial`,
   `LimiteDerivacionReferencial`). La segunda quedaba abierta y era la mitad
   silenciosa: **`Paquete.Aplicabilidad` estaba fuera del barrido**, con la
   excusa de que las reglas tienen su propio linter. Ese linter comprueba que la
   regla se PARSEA, no cuanto texto lleva dentro, y una regla es una cadena libre
   con literales: `aplica("<aqui cabe un control entero>", S) :- ...`. Cerrado en
   `nucleo/corpus/higiene_test.go` con control negativo por campo, y la excepcion
   quitada del barrido por reflexion, que es lo que impide que vuelva.
2. **`Obligacion.Vigencia` no la usa nadie.** El campo existe, se valida y no
   entra en ningun calculo: una obligacion derogada sigue apareciendo. Con
   normas que se modifican cada pocos anos, esto es una respuesta incorrecta,
   no una funcionalidad ausente.
3. **Falta `Obligacion.Titulo`.** Hoy la unica etiqueta legible de una
   obligacion es su `articulo`, que en el ENS es cosas como "Anexo II 4.2.5
   Mecanismo de autenticacion (usuarios externos) [op.acc.5]". Sirve, pero no es
   un titulo, y la pantalla de Controles va a ensenarlo.
4. **`Temporalidad` no sabe de prorrogas.** Ni de suspension de plazo. Hay
   normas que las tienen y hoy no se pueden expresar.
5. **No hay forma de ver un paquete.** Ni un `plazum corpus ver <urn>`. Para
   saber que hay dentro hay que abrir el JSON.
6. **`rgpd` y `cra` llevan el texto transcrito sin tildes.** `ens` si las tiene.
   Es texto del DOUE reproducido: o se reproduce bien o no se reproduce.

### De la etapa 2, bloque de puertos (25-08-2026)

7. **`corpus.EsquemaUI` pierde citas.** Cuando tres normas piden el mismo dato,
   `Paquetes` dice quienes son pero solo sobrevive la ayuda y la cita de UNA
   (la de URN menor, desde que se arreglo el determinismo). El comprador
   pregunta "por que me piden este dato" y se le responde con un articulo de
   tres. Arreglo probable: `Citas map[string]string` por URN en `CampoUI`. Toca
   la forma que consume `nucleo/pantalla`, asi que cuanto antes mejor.

### Del invariante 2 (25-08-2026)

> El P1 numero 8, el espacio de nombres de los predicados, se cerro el 25-08-2026.
> Ver `nucleo/aplicabilidad/espacio.go`. De paso salio una regla de modelado que
> no estaba escrita: un paquete no redefine un predicado que el sujeto aporta
> como hecho, y ahora se denuncia al evaluar en vez de derivar sobre un predicado
> vacio en silencio.

9. **`paquetes/ens` no tiene entidades `informacion` ni `servicio`.** La regla
   de agregacion del anexo I esta declarada y es correcta, pero los hechos que
   consume (`maneja`, `nivel_dimension`) no los recoge ninguna pregunta del
   paquete: solo se pueden afirmar a mano. Hasta que el modelo de entidades
   crezca, la categoria se declara y no se calcula en el producto. La regla se
   ejerce en `aplicabilidad_corpus_test.go`, no en la interfaz.

### Del frente de expediente (25-08-2026)

11. **El generador del demo vive en `nucleo/` y ya no puede regenerarlo.** El
    expediente de demostracion es un artefacto de PRODUCTO y su valor esta en que
    enseña normas reales (ENS, RGPD, CRA, con sus articulos); el escenario de
    `nucleo/expediente/expediente_test.go` es un artefacto de PRUEBA y no puede
    nombrar normas. Compartian constructor. La regeneracion esta cerrada con un
    mensaje que lo explica (`TestLaRegeneracionDelDemoYaNoViveAqui`), asi que hoy
    no hay mina, pero el demo publicado solo se puede editar a mano. Arreglo:
    sacar el generador a `herramientas/generardemo/`, con el escenario como
    fichero de datos, igual que hace `herramientas/sellardemo` con el sello.
12. **El ataque 10 no lo caza la comprobacion que su comentario promete.**
    `TestHostilElEmisorYaNoSeFabricaSusPropiasAnclas` dice cazar al emisor que se
    escribe el ancla que cuadra, contrastando el contenido recalculado. Apagando
    esa comparacion el test SIGUE VERDE: lo que lo salva es el chequeo de
    `ancla declarada de <urn>`, que el propio codigo comenta como informativo
    ("una diferencia no invalida por si sola, pero el auditor tiene que verla").
    El ataque se detecta, o sea que no es un agujero, pero la cobertura no es la
    que el comentario promete y es justo el patron "tapado" del que este proyecto
    se defiende. Hay que aislar cada capa con su propio test.

### Del barrido de aserciones (25-08-2026)

10. **Tras un borrado legal queda un estado de control huerfano.** Al retirar la
    observacion suprimida, el `EstadoControl` que se sostenia en ella sale como
    discrepancia del expediente. El expediente sigue siendo valido y la supresion
    se informa bien, pero la discrepancia es ruido: probablemente deberia poder
    declararse "sin evidencia por supresion legal". Es decision de diseno.

### Del frente de pantallas (25-08-2026)

13. **El formulario del esquema se pinta en solo lectura.** La pantalla de
    Alcance deriva los campos de `corpus.EsquemaUI` y los ensena con su tipo,
    sus valores admitidos, su cita y que paquetes piden cada dato, pero no deja
    escribirlos: no hay expediente donde guardarlos. Un formulario con boton de
    guardar que no guarda es peor que no tener formulario, y en un producto de
    cumplimiento es de las mentiras caras. Arreglo, cuando exista el estado:
    campos de verdad con POST por el middleware de CSRF de quien construye el
    servidor. Hoy `superficies/pantallas` no tiene ninguna ruta que mute, a
    proposito, y hay un test que lo vigila.
14. **La derivacion de la pantalla no es el motor de aplicabilidad.** Alcance
    cruza las respuestas de la entrevista con `pantalla.Fila.Requiere`, que es
    lo que declara el paquete. `nucleo/aplicabilidad` decide de verdad, con
    Datalog, sobre hechos de las entidades del sujeto, y esos hechos salen del
    expediente. La interfaz lo dice con esas palabras y nunca se presenta como
    dictamen, pero son dos lecturas distintas conviviendo. Arreglo: cuando el
    expediente exista, consultar `aplica/2` y dejar la lectura por `Requiere`
    solo como avance mientras falten hechos.
15. **El texto del corpus se pinta sin declarar su idioma.** `corpus.Paquete` no
    dice en que idioma esta su texto, asi que la plantilla no puede poner
    `lang=` alrededor de lo que viene del paquete, y un lector de pantalla lee
    un articulo en espanol con la fonetica del idioma de la interfaz cuando no
    coinciden. Arreglo: un campo `idioma` en el paquete y un `lang=` en la
    plantilla. Es la misma frontera que impide traducirlo: el idioma es del
    paquete, no de la interfaz.
### Del autoservicio (frente (c) de la etapa 2, 25-08-2026)

26. **`nucleo/corpus` no exporta la traducción de `Temporalidad` a primitiva de
    `ventana`.** Existe dentro de `corpus/dorados.go` sin exportar, y solo sirve
    allí para comparar un dorado con su esperado. Para **enseñar** una fecha
    hace falta la misma traducción y no hay forma de llamarla, así que
    `plazum demo` la tiene escrita otra vez (`VencimientosDe`, en
    `cmd/plazum/demo.go`). La duplicación está guardada por
    `TestLaTraduccionDelRelojReproduceTodosLosDoradosDelCorpus`, que ejecuta la
    del CLI contra **todos** los casos dorados publicados y tiene su control
    negativo, así que hoy no puede desviarse en silencio. Pero el sitio correcto
    es una función exportada de `nucleo/corpus`, y con `serve` y las pantallas
    llegando esto se va a escribir una tercera vez. Es una firma nueva en el
    núcleo, o sea que se decide, no se cuela.
27. **La tabla de caducidades de las raíces de TSA está declarada en
    `adaptadores/diagnostico`, no leída.** `x509.CertPool` no expone los
    certificados que contiene (`Subjects()` está obsoleto y solo devuelve el
    sujeto en DER, sin fechas), así que `doctor` juzga las raíces embebidas
    contra una tabla que es espejo de `adaptadores/tsa/raices/LEEME.md`. Puede
    envejecer sin que nadie se entere, que es exactamente la clase de fallo
    silencioso que `doctor` existe para evitar. Arreglo: que `adaptadores/tsa`
    exporte los certificados parseados. Las raíces que aporta el operador sí se
    leen de verdad, con su `NotAfter`, y esa mitad no tiene el problema.

### Del frente de identidad, OIDC y SCIM (25-08-2026)

16. **No existe `plazum scim token`.** El servidor SCIM exige un token de
    aprovisionamiento de al menos 32 caracteres y no hay forma de generarlo con
    el producto: el operador tiene que inventarselo. El mensaje de error ya NO
    nombra el comando (nombrar uno que no existe quema la confianza en el resto
    de los mensajes), pero el hueco sigue. Es de `cmd/plazum`, que es de otro
    frente.
17. **No hay pantalla de Personas.** El mapeo manual de la jerarquia esta
    completo en el adaptador (`FijarManagerManual`, `Conflictos`, `SinManager`,
    `Rotas`) y varios mensajes accionables mandan a "Personas" a usarlo. La
    casilla de la pantalla es de la etapa 2 y de otro frente; hasta que exista,
    la alternativa al `manager` del IdP solo es alcanzable por codigo. La mitad
    de los clientes no publica `manager`, asi que esto es la mitad del valor de
    la casilla sin superficie.
18. **El directorio SCIM vive en memoria.** Un reinicio pierde usuarios, grupos
    y jerarquia, y el IdP tarda hasta un ciclo entero en reponerlos. La
    persistencia es la casilla del adaptador `sqlite`, que sigue sin construir.
    Mientras tanto, SCIM no es apto para produccion aunque el protocolo si lo
    sea.
19. **El `state`, el `nonce` y el verificador PKCE viven en memoria del
    proceso.** Consecuencia inmediata: un reinicio a mitad de login obliga a
    volver a pulsar entrar (tolerable), y dos instancias detras de un balanceador
    NO comparten los flujos en vuelo, asi que un login que empieza en una y
    vuelve a la otra falla siempre. Hay que decidirlo antes de documentar
    cualquier despliegue con mas de una instancia.
20. **El middleware de seguridad tiene que cubrir `/scim/v2`.** El servidor SCIM
    acota su cuerpo y exige credencial, pero el rate limiting y las cabeceras son
    del frente que construye el servidor. Sin limite de tasa, el endpoint SCIM
    admite fuerza bruta contra el token; con un token de 32 caracteres es
    inviable, pero el limite tiene que existir igual y hay que comprobar en el
    cableado que la ruta pasa por el.

### Del frente de i18n, accesibilidad y presupuestos (25-08-2026)

El 21 es el empalme que queda entre este frente y el de pantallas. Del 22 al 25
salieron de la tercera pasada, la del comprador, que aqui es un CISO de 200
empleados que trabaja en ingles. El otro hallazgo de esa pasada, el `lang=`
alrededor del texto del corpus, ya estaba apuntado por el frente de pantallas en
el 15 de arriba; se anade alli el dato que faltaba: es WCAG 3.1.2 (Language of
Parts, nivel AA) y axe NO lo caza, porque axe no sabe en que idioma esta escrito
un parrafo.

21. **La superficie sigue pintando con su borrador de catalogo.** El catalogo de
    verdad ya existe (`adaptadores/catalogo`), cubre EXACTAMENTE las claves que
    declara `superficies/pantallas.ClavesDeCatalogo()`, esta en castellano y en
    ingles, resuelve el plural y hay un test que lo compara en los dos sentidos.
    La superficie todavia construye su `Superficie` con `catEs` de
    `borrador_catalogo_test.go`, que solo tiene castellano. Cambio, del frente de
    pantallas: pasar `catalogo.Nuevo()` en las `Opciones` y borrar el borrador.
    Hasta entonces el producto tiene la traduccion hecha y no la ensena.
22. **El CLI habla un solo idioma.** `cmd/plazum` no pasa por el catalogo: sus
    mensajes estan cableados en castellano. Un CISO que trabaja en ingles pone
    la interfaz web en ingles, corre `plazum verify` y se encuentra con
    "expediente ilegible". La i18n de la etapa 2 es de la UI, asi que no bloquea
    la casilla, pero el producto son las dos superficies.
23. **Nadie ensena todavia `aviso.idioma_del_corpus`.** La clave existe en los
    dos idiomas y explica por que el texto de las normas sigue en el idioma de
    su fuente. Mientras no se pinte AL LADO del texto del corpus, y no en una
    pagina de ayuda, el usuario ingles lee la decision legal como un producto a
    medio traducir. Va de la mano del 15.
24. **Las fechas no tienen formato acordado, y aqui eso es un riesgo.** Un
    03/04/2026 lo lee un espanol como 3 de abril y un ingles como 4 de marzo.
    En una herramienta cuyo producto son fechas limite legales, eso no es una
    molestia de formato. Propuesta: formato no ambiguo e independiente del
    idioma en toda fecha de vencimiento (ISO 8601, o dia mes-abreviado ano).
25. **No existe todavia la eleccion de idioma.** `Traducir` ya normaliza el
    locale (en-GB es en), que es la mitad de abajo. Falta la de arriba: leer
    Accept-Language, dejar elegir y recordarlo, y poner el `lang` del `<html>`
    en el idioma que se ha renderizado. Es del frente de pantallas.

### De los frentes de TTFV y distribucion (26-08-2026)

28. **Las plantillas de `superficies/serve` llevan `lang="es"` cableado.** El
    resto del producto negocia idioma y `/alcance` responde "Alcance" o "Scope"
    segun la cabecera. Las plantillas base de serve, en cambio, declaran espanol
    pase lo que pase, asi que un usuario en ingles recibe `/entrar` con el
    atributo `lang` mintiendo sobre el idioma del contenido. **axe-core no lo
    caza**, porque el atributo esta presente y es sintacticamente valido: lo que
    esta mal es que sea falso, y eso ninguna herramienta automatica lo sabe. Es
    un fallo de accesibilidad real, no cosmetico: un lector de pantalla elige la
    voz por ahi. Toca `superficies/serve`, no `superficies/pantallas`.

29. **Dentro de un contenedor, `plazum serve` dice una direccion que no sirve.**
    Imprime `Abre http://[::]:8443/`, que es la direccion de escucha, no la que
    el operador tiene que abrir. Con la imagen Docker recien construida, el
    primer mensaje que ve quien arranca el producto le da una URL que no
    funciona. O dice la del puerto publicado, o no dice ninguna y explica como
    averiguarla. Es de `superficies/serve`.

### Del vendorizado de pkcs7 (26-08-2026)

30. **`ber2der` amplifica hasta x482 medido, y el arreglo no esta a nuestro
    alcance hoy.** Lo encontro el fuzzing propio del directorio vendorizado. En
    `readObject`, un objeto construido de longitud DEFINIDA devuelve la longitud
    DECLARADA y no el offset que sus hijos consumieron; un hijo que se pasa de
    largo se traga bytes que el abuelo vuelve a leer como su siguiente hermano, y
    salen dos veces. Anidado, se multiplica. Medido: 331 bytes producen 159.693
    (x482), 631 producen 1.197.909 (x1.898), 931 producen 2.542.305 (x2.731); la
    razon se aplana hacia x4.000. **Esta en la version fijada y tambien en la de
    cabeza**, o sea que es de aguas arriba y hay que reportarlo alli. Arreglarlo
    en la copia vendorizada NO quitaria la exposicion, porque `Cadena.verificar`
    llama primero a `timestamp.Parse`, que parsea el mismo token con el `pkcs7`
    de fuera. Lo que si se ha hecho: un tope de 32 KiB al token antes de
    parsearlo, en las dos puertas de entrada, y un test que clava el numero de la
    amplificacion por los dos lados para que ni empeore ni mejore en silencio.
    Queda: reportar aguas arriba, y decidir si se porta el arreglo a la copia el
    dia que `timestamp` desaparezca (etapa 8). Comprobado de paso que el arreglo
    candidato (devolver el offset consumido) **no rompe el token real de la
    demo**: con el puesto, la suite entera pasa salvo las dos puertas que miden
    la amplificacion.

31. **El motor de fuzzing no corre en CI, en ningun objetivo del repositorio.**
    Lo que corre en cada `go test` es el CORPUS SEMILLA, que es una regresion,
    no una busqueda. El bloqueante es concreto y por eso se anota: `go test
    -fuzz` no imprime lineas `--- PASS`, asi que `.github/puerta.sh`, que cuenta
    exactamente esas lineas, contaria cero y pondria la puerta en rojo. Un paso
    de fuzzing necesita o una funcion nueva en `puerta.sh` que sepa leer la
    salida del motor (`elapsed:`, `execs:`), o una entrada en `exentas` de
    `puertas_test.go` con su motivo. Las dos cosas tocan ficheros compartidos,
    asi que se deja decidido en lote. Mientras tanto, el fuzzing largo se lanza a
    mano con el comando que hay en
    `adaptadores/tsa/internal/pkcs7/LEEME.md`.

## P2

### De la higiene legal del corpus (26-08-2026)

- **Las 132 `cita` del ENS llevan la URL entera dentro.** El campo `fuente` ya
  no guarda direcciones, pero cada `cita` del paquete transcrito termina en
  `... (BOE-A-2022-7191). https://www.boe.es/eli/es/rd/2022/05/03/311/con`. Es la
  misma enfermedad en otro campo: el dia que el BOE mueva esa ruta, 132 citas
  apuntan a la nada. Dos cosas atenuan el riesgo y por eso no se toco aqui: la
  cita YA lleva el identificador estable delante (`BOE-A-2022-7191`), asi que
  sigue identificando la norma sin el enlace; y una `cita` es texto que escribe
  quien autora, no un enlace que el producto derive. El arreglo, cuando toque, es
  de la autoria del corpus: quitar la direccion y dejar el `BOE-A-...`, que ya
  esta. Toca 132 lineas de datos de `paquetes/ens`, ninguna de codigo.
- **`docs/guia.md` y `docs/LICENCIAS.md` siguen describiendo el formato viejo.**
  El Anexo B de la guia dice "BOE/DOUE entero con `fuente` enlazada" y LICENCIAS
  habla de "los dos campos que declara cada paquete" cuando ya son tres. Ninguno
  de los dos es de este frente (la guia es la fuente unica del plan), y el riesgo
  real es bajo porque quien escriba `fuente` se encuentra un error del linter que
  le dice literalmente que escribir en su lugar. Pero son dos documentos que
  mienten sobre el formato, y se corrigen en el proximo paso que toque el plan.
- **ISO obliga a guardar una clave del editor (`identificador.registro`).** Es la
  unica excepcion del vocabulario: el catalogo de ISO no esta indexado por la
  designacion de la norma (`ISO/IEC 27002:2022`) sino por un numero de registro
  propio (75652) que no se deriva de ella. Se guarda ese numero porque es una
  CLAVE, no una direccion (la forma de la pagina la sigue poniendo una sola
  funcion), pero es un dato del editor viviendo en nuestro corpus. Si ISO publica
  algun dia un permalink derivable de la designacion, el campo sobra.
- **El `identificador` no entra en el digest del paquete.** `DigestPaquete`
  resume reglas y obligaciones, asi que la procedencia queda fuera, igual que
  quedaba el campo `fuente` que sustituye. Consecuencia: alguien que distribuya
  un corpus manipulado puede cambiar a que fuente apunta un paquete sin cambiar
  su digest ni romper ningun ancla. NO llega al expediente (`expediente.Paquete`
  lleva urn, version, digest, clase y vigencia, y ninguna direccion), asi que
  esto no toca la capa probatoria ni la promesa de `docs/modelo-de-amenaza.md`:
  el unico enganado posible es quien ya esta ejecutando con un corpus adulterado
  en su propia maquina. Se anota porque el cambio de formato es el momento en el
  que alguien podria creer que ahora si viaja firmado, y no viaja.
- **PCI DSS deriva la biblioteca, no el documento.** PCI SSC sirve todas las
  versiones desde `document_library/` y no publica una direccion por version, asi
  que el identificador (`4.0`) no entra en la URL: entra en la pantalla, que es lo
  que le dice al lector que documento coger. El dia que PCI publique una direccion
  por version, es una linea de `corpus.Identificador.Enlace`.

- **La fuente oficial se pinta como texto, no como enlace.** El pie del producto
  ensena la atribucion de cada paquete con su identificador estable y con la
  direccion derivada de el, pero sin `href`. Son dos razones y las dos son de esa
  superficie: `TestHtmxVaVendorizadoYNoPorCDN` prohibe que la pagina apunte a
  nada de fuera, y htmx va con `selfRequestsOnly`, asi que un enlace externo
  dentro del cuerpo con `hx-boost` no navegaria. La condicion de reutilizacion es
  citar la fuente, y la direccion a la vista la cita, asi que no es un
  incumplimiento; es peor experiencia. Arreglo cuando esa superficie decida su
  politica de enlaces externos: acotar la prohibicion a `src=` y a `<link>`, que
  es lo que de verdad vigila, y sacar el enlace del boost.

  **Al 26-08-2026 es mas barato que antes y sigue siendo de esa superficie.** El
  enlace ya no sale de un campo de datos sino de una funcion, asi que ponerle
  `href` es tocar una plantilla y una linea de un test de esa superficie, no 31
  ficheros. Sigue sin hacerse aqui porque decidir la politica de enlaces externos
  de una superficie es de esa superficie, y esa prohibicion la escribio otro
  frente por un motivo de seguridad que no es de este cambio. Es la respuesta al
  "¿llega a la norma de un clic?" del comprador: hoy no, se copia y se pega.
- **El `urn` de `eidas2` y de `csrd` sigue nombrando al acto modificativo.** El
  `identificador` ya apunta al instrumento donde viven las obligaciones y el
  `LEEME.md` de cada uno lo dice, pero el `urn` no se ha tocado: cambiarlo cambia
  la identidad del paquete en el expediente y en las equivalencias. Es decision
  de la autoria del corpus, no de higiene.
- **El RDL 19/2018 no esta censado.** Es lo que vincula en Espana en lugar de la
  Directiva 2015/2366, y hasta que se cense no se puede decidir si es un marco
  propio o una capa de `psd2`. Ya consta como hueco en `docs/censo-relojes.md`;
  se repite aqui porque ahora el `LEEME.md` del paquete se lo promete al lector.
- **El pie repite el mismo aviso veintiuna veces.** Con el corpus publicado, el
  pie ocupa 15,5 KB de una pagina de 170 KB, y 21 de sus 31 lineas llevan el
  MISMO parrafo de la Decision 2011/833/UE porque 21 paquetes vienen del DOUE.
  Legalmente esta bien (cada obra atribuida) y la practica habitual es un aviso
  con la lista de obras debajo. Agrupar por texto de aviso es decidir QUE se
  ensena, o sea que es de `nucleo/pantalla` y cambia la forma del caso dorado:
  no se hace de pasada.
- **El pie identifica cada paquete por su URN.** `urn:eu:reg:2016:679` no es un
  nombre que un CISO reconozca, y el formato de paquete no tiene campo de nombre
  legible. Se cruza con el P1 numero 5 (no hay forma de ver un paquete): el
  arreglo es el mismo campo.
- **`Campo.Peticiones` no pasa por el saneado de la superficie.**
  `sanearPantallas` recorre preguntas, campos, filas y fuentes, y deja fuera la
  lista de peticiones de cada campo, que tambien trae cita y ayuda del corpus.
  Es preexistente y no lo introduce este frente. No hay test de exhaustividad del
  saneado, que es lo que lo habria cazado: hoy la lista se escribe a mano.

### Alcance declarado del autoservicio (frente (c) de la etapa 2, 25-08-2026)

Lo que se ha dejado fuera a propósito, para que no se confunda con lo que falla.

- **El canal de actualización es solo de directorio.** `CanalDirectorio` cubre
  la instalación sin salida a internet, que en este mercado son más de las que
  parece, y es la forma en la que se prueba toda la vuelta atrás. El canal HTTP
  firmado va con la entrega del corpus de la etapa 3 e implementa la misma
  interfaz sin tocar nada del rollback.
- **`plazum update` no migra la base de datos ni reinicia el servicio.** Lo
  primero llega con el adaptador de almacén; lo segundo es de systemd o de quien
  arranque. Está dicho en el godoc del paquete para que no se confunda con lo
  que sí hace.
- **El cerrojo del actualizador no caduca.** Un proceso que muere sin soltarlo
  deja un cerrojo huérfano que hay que borrar a mano. El error dice la ruta
  exacta y el pid que lo dejó, así que es un minuto, pero un cerrojo con marca
  de tiempo y expiración sería mejor. No se ha hecho porque expirar un cerrojo
  mal es peor que no tenerlo.
- **El demo con `--corpus` enseña las obligaciones reales como no aplicables.**
  Es correcto (nadie ha respondido el alcance de esos paquetes) y se explica en
  pantalla, pero se lee peor de lo que es. Cuando exista la pantalla de Alcance
  de `serve`, el demo debería poder precargar un alcance real y enseñar
  obligaciones de verdad derivadas.
- **El demo no encadena con `plazum verify`.** El paso de "veo mis obligaciones"
  a "un tercero puede recalcular mi expediente sin fiarse de mí" es la promesa
  más fuerte del producto, y hoy solo se ofrece si `expediente-demo.json` está
  al lado del binario, porque no viaja empotrado. Empotrarlo son ~25 KB y lo
  cerraría; toca `adaptadores/tsa` y el demo del expediente, que son de otro
  frente.

1. **`nombresDeConfianza` es una lista cerrada.** El detector de
   `confianza_test.go` caza por nomenclatura, asi que un campo llamado
   `RaicesAceptadas` o `ClavePublicaDelOperador` pasaria. Es inherente al
   metodo; la red de verdad para esa clase es la revision hostil.
2. **`ledger`: la clave publica malformada no tiene centinela.** Tiene dientes
   (sin la guarda de tamano, `ed25519.Verify` hace panic), pero el test lo
   comprueba por `recover` y no por identidad del error. Darle centinela obliga
   a reescribir el mensaje accionable.
3. **Un `paquete.json` corrupto y uno ausente se tratan distinto.** El ausente ya
   esta cerrado (`TestNingunPaqueteSeCaeDelCorpusEnSilencio`); el corrupto da
   error y pone la puerta roja, que es lo correcto. Queda apuntado por si algun
   dia hace falta un directorio bajo `paquetes/` que no sea un paquete: la
   excepcion tendra que escribirse a mano en `directoriosPublicados`.
4. **Lectura del reloj por via indirecta.** `//go:linkname` a `runtime.nanotime`
   no se detecta directamente. Se cierra por el otro lado: `syscall`, `unsafe` y
   `plugin` estan prohibidos como imports del nucleo, y `nucleo/` solo puede
   importar `<modulo>/nucleo/...`, asi que no puede delegar la lectura en otro
   paquete del repo.
5. **`time.Now()` en los `_test.go` de `nucleo/`** no se vigila, a proposito. Un
   test que lee el reloj es fragil, pero no rompe la reproducibilidad del
   expediente, que es la propiedad que el invariante defiende. Hoy no hay
   ninguno.

### Del frente de pantallas (25-08-2026)

6. **El linter del corpus no acota la longitud de etiqueta ni de ayuda.** Una
   etiqueta de 100 KB no rompe la pagina (hay test) pero la deja inservible. El
   limite de 120 caracteres del referencial es frontera legal, no de
   presentacion, y no cubre esto. Arreglo: un aviso del linter, no un rechazo.
7. **Las paginas no llevan cache HTTP.** Cada clic en una respuesta re-renderiza
   la pagina entera. Con corpus grande esta paginado y acotado, pero no hay
   `ETag` ni `Last-Modified`. La pagina es funcion pura de (corpus, consulta,
   idioma), asi que un `ETag` sobre el hash de esa terna es directo.
8. **La accesibilidad esta cuidada a mano, no verificada por herramienta.** Hay
   puntos de referencia, enlace de salto, `aria-current`, tablas con `scope` y
   `caption`, contraste elegido por encima de 4.5:1 en los dos temas y estados
   que no dependen solo del color. Nada de eso esta comprobado con axe-core, que
   es puerta de CI de esta etapa y necesita node.
9. **Las formas del plural las tiene que resolver el catalogo.** Las claves con
   contador (`alcance.derivacion.aplican`, `menu.aplican`,
   `alcance.pregunta.desbloquea`) pasan el numero como argumento, que es lo
   unico correcto porque la forma plural depende del idioma. El borrador de
   catalogo de `superficies/pantallas/borrador_catalogo_test.go` no las
   resuelve, asi que hoy se lee "decide 1 obligaciones". Arreglo en el frente de
   i18n: que `Traducir` elija forma segun el primer argumento numerico.
   ESTA HECHO en `adaptadores/catalogo` (25-08-2026): las formas van separadas
   por barra vertical en el fichero de cadenas y las elige `elegirForma`. Lo que
   queda es de este frente: que la superficie use el catalogo de verdad en vez
   del borrador, y entonces esto se borra de aqui.
10. **No se resalta que cambio con la ultima respuesta.** El panel de la
    derivacion ensena el estado actual y lo que desbloquea la siguiente
    pregunta, pero no marca que se movio con el ultimo clic. Con corpus grande
    eso obliga a comparar de memoria.
11. **No hay siguiente paso al terminar la entrevista.** Cuando se responden
    todas las preguntas, el panel se queda ensenando lo que aplica y no propone
    que hacer despues. Lo siguiente natural es Certificados, y esta en el menu,
    pero no se sugiere.


### Del frente de serve, sesiones y seguridad web (25-08-2026)

12. **El limitador se vacia entero al llegar al techo de claves.**
    `superficies/serve/middleware.go`, `Limitador.Permitir`. Con mas de 200.000
    claves vivas se purgan las caducadas y, si aun asi no baja, se tira el mapa
    completo y se cuenta en `Vaciados()`. Es fallar abierto a proposito: fallar
    cerrado convertiria una inundacion en una caida total. Lo correcto seria
    expulsar las mas antiguas, que pide un monticulo. Con la clave por direccion
    de conexion hace falta una botnet para llegar al techo, y quien tiene una
    botnet no necesita reiniciar contadores. Toca cuando el limitador se
    persista o se comparta entre instancias.
13. **Las sesiones viven en memoria y reiniciar echa a todo el mundo.** Es una
    decision, no un descuido: para un producto que se instala una vez es
    aceptable, y ademas es la vuelta atras mas barata ante una sospecha de
    sesion robada. Al construir el adaptador de `Almacen`, decidir si se
    persisten conservando las propiedades de hoy (identificador guardado en
    hash, caducidad comprobada en cada lectura, tokens atados a la sesion).
14. **El limite de intentos de autenticacion es por direccion, no por cuenta.**
    Un ataque repartido entre muchas direcciones contra una sola cuenta no lo
    frena el cubo actual. Al existir el almacen de usuarios, un segundo cubo por
    sujeto, con cuidado: un cubo por cuenta lo puede usar un tercero para dejar
    fuera a una persona concreta.
15. **`Origin` ausente no se rechaza.** Cuando el navegador lo manda y no
    coincide, se rechaza; cuando no lo manda, la proteccion la da el token, que
    es lo que un tercero no puede leer. Es lo correcto hoy porque un cliente de
    linea de ordenes legitimo tampoco lo manda. Revisar al aparecer la API con
    token portador.
16. **HSTS se manda tambien sobre http, y RFC 6797 §7.2 dice que no.** Se hace a
    sabiendas y esta anotado en el codigo: el navegador la ignora ahi (§8.1),
    asi que no cuesta nada, y el operador que mas la necesita es el que puso un
    proxy con TLS delante y no se lo dijo a plazum. Si un escaner de conformidad
    de un comprador lo marca, se condiciona a `X-Forwarded-Proto`.
17. **La cookie usa `SameSite=Lax` y no `Strict`.** Con Strict, llegar desde el
    enlace de un correo de escalado ensena la pantalla como si no hubieras
    entrado. En la etapa 4, cuando esos correos existan, medir si compensa
    Strict con una pagina puente.
18. **El diagnostico no ve todavia el estado del servidor.**
    `Limitador.Vaciados()` y `Sesion.Vivas()` existen y nadie los lee. Al
    construir `plazum doctor`, conectarlos como dos comprobaciones con su arreglo.
19. **La politica de contrasena del primer administrador es una longitud
    minima.** 12 caracteres y nada mas. Al existir el almacen de usuarios,
    decidir donde vive la politica, probablemente en el adaptador y no en la
    superficie web, porque la superficie no es la unica puerta.
20. **El cierre ordenado bajo senal solo se comprueba en Linux.** La parte de Go
    esta cubierta por test (`TestArrancarSirveDeVerdadYElContextoLoCierra`
    cancela el contexto y exige nil); lo que solo se ejercita en CI es el enlace
    entre la senal del sistema y ese contexto, que lo hace `signal.NotifyContext`
    de la biblioteca estandar. En Windows `kill -TERM` termina el proceso sin
    senal y el paso no se puede reproducir en local.
21. **Las dos pantallas de arranque van sin estilo.** Entrar y crear el primer
    administrador se pintan con HTML plano, sin hoja de estilos y sin depender
    de ningun estatico. Es deliberado (tienen que funcionar antes de que exista
    interfaz, con la CSP mas estrecha posible), pero la primera pantalla que ve
    un comprador es esa. Engancharlas a la hoja del frente de pantallas sin
    meter nada inline y sin que dejen de funcionar si el estatico no carga.
22. **`plazum serve` no existe todavia como orden.** Este frente entrega el
    servidor como biblioteca y un binario de pruebas bajo
    `superficies/serve/internal/servidorprueba` que solo usa la puerta de CI.
    Quien instala plazum hoy no puede arrancarlo: el cableado de `cmd/plazum` es
    de otro frente y depende ademas del almacen de usuarios, que no existe.
23. **Los estaticos no traen `ETag` ni contenido precomprimido.** Van con cacheo
    largo e inmutable, que resuelve la segunda visita, pero la primera baja el
    fichero entero sin comprimir. Junto al presupuesto de tamano de la etapa 2,
    si se mide y sale caro. Se solapa con el numero 7 de arriba, del frente de
    pantallas.
### Del frente de identidad, OIDC y SCIM (25-08-2026)

24. **El filtro de un atributo multivaluado en la ruta de un PATCH se ignora.**
   `emails[type eq "work"].value` se normaliza a `emails` y la operacion se
   aplica a la coleccion entera. Para lo unico multivaluado que se guarda
   (correos) el resultado que le importa al producto no cambia, pero deja de ser
   SCIM estricto y esta dicho en el godoc del paquete.
25. **No hay cierre de sesion federado.** El `end_session_endpoint` se lee del
   descubrimiento y no se usa: cerrar sesion en plazum no cierra la del IdP.
26. **`meta.version` se emite y `/ServiceProviderConfig` declara `etag` no
   soportado.** No es contradictorio (el ETag de SCIM es la cabecera, no el
   campo), pero es confuso de leer y algun IdP podria intentar usarlo. O se
   implementa el control de concurrencia optimista o se quita el campo.
27. **No hay SAML.** Apuntado para el ano 2 y dicho en `docs/identidad.md` para
   que nadie lo busque.

### De i18n, accesibilidad y presupuestos (25-08-2026)

28. **`web/index.html` tiene una violacion de axe y no entra en la puerta.** Es
    `region` (moderada, de la familia best-practice, no de WCAG): el contenido
    no esta dentro de ningun landmark. Se arregla envolviendo el cuerpo en
    `<main>`. Medido con axe-core 4.13 sobre el fichero de hoy. La puerta de
    accesibilidad apunta a las pantallas de la aplicacion y no a la web publica,
    que tiene otro dueno y otro ciclo, asi que el hallazgo se apunta y no se
    cuela en el CI de otro.
29. **El contraste con el corpus no ve una parafrasis.** Compara trozos de seis
    palabras normalizadas, asi que caza la copia literal y no caza a quien
    reescribe el articulo con sus palabras dentro de una cadena de interfaz.
    Bajar la ventana a cuatro o cinco devuelve falsos positivos ("en el plazo
    de"), y un detector que grita por todo se acaba desactivando. Contra la
    parafrasis quedan la frontera del cargador y la revision humana.
30. **La web publica solo esta en castellano.** El producto habla ingles desde
    esta casilla y la pagina que lo vende, no. No es de esta casilla, pero el
    comprador llega antes a la web que al producto.
31. **El formateo de duraciones y cantidades no es del catalogo.** Hoy
    `error.limite_peticiones` recibe una duracion ya formateada por quien llama.
    Cuando haya que declinar unidades por idioma, eso pide una decision (ICU
    MessageFormat o equivalente) que hoy seria prematura. El plural con contador,
    que es el caso urgente y lo pide el 9 de arriba, ya esta resuelto en
    `Traducir`.

### De los frentes de TTFV y distribucion (26-08-2026)

32. **`/primer-admin` no entra en la auditoria de accesibilidad.** Con
    `plazum serve` a secas no hay almacen de usuarios y la ruta responde 503, asi
    que axe no la ve. Es la primera pantalla que toca quien instala esto, o sea
    la peor para tener sin auditar. Hace falta levantarla con almacen en el job.

33. **Los tres presupuestos viven en un fichero que se llama
    `etapa2-accesibilidad.yml`.** Tamano de binario, arranque y RAM no son
    accesibilidad. No se renombro para no romper los checks requeridos de las
    ramas; hay referencias cruzadas en las cabeceras. Se arregla cuando se toque
    la proteccion de rama.

34. **`plazum --help` sale con codigo 2.** Cae en el camino de "orden
    desconocida". Pedir ayuda no es un error y un script que compruebe el codigo
    de salida se lleva una sorpresa. Preexistente.

35. **La puerta de reproducibilidad de la imagen no prueba `-trimpath`.** Compara
    dos construcciones desde la misma ruta, asi que caza indeterminacion real
    (marcas de tiempo, orden de mapas) pero no rutas empotradas. Eso hoy lo
    vigila un test estatico que lee el `Dockerfile`. La comprobacion fuerte,
    construir fuera de la imagen y comparar, exige fijar la version exacta de Go
    en los dos sitios. Anotado en el propio workflow.

### De la pasada adversaria sobre la higiene legal (26-08-2026)

36. ~~**Dos `fuente` caducadas, encontradas siguiendo los 31 enlaces.**~~
    **RESUELTO el 26-08-2026.** `soc2` apuntaba a
    `aicpa-cima.com/topic/audit-assurance/...` y `stig` a `public.cyber.mil/stigs/`,
    que redirigia cambiando de anfitrion. Las dos apuntan ya a su destino actual,
    y las dos declaran `identificador.tipo: sin-identificador` con el motivo
    escrito, porque ni AICPA ni DISA publican un identificador citable.

37. ~~**No hay forma barata de vigilar que los enlaces sigan vivos.**~~
    **RETIRADO POR DISENO el 26-08-2026: la vigilancia ya no hace falta.**
    El diagnostico seguia siendo correcto (EUR-Lex e ISO responden 403 a una
    peticion automatica, asi que un comprobador no distingue "muerto" de "me han
    tomado por un robot"), pero la pregunta era la equivocada. Con la URL como
    dato habia que vigilar que la PAGINA siguiera en su sitio; con el
    identificador como dato lo que se vigila es que el IDENTIFICADOR siga
    existiendo, y un identificador no se mueve: se deroga, y eso ya lo mira la
    vigilancia de normas de `herramientas/ingestanorma`. Si manana un editor
    reorganiza su sitio, el sintoma es un enlace roto en pantalla y el arreglo es
    una funcion (`corpus.Identificador.Enlace`), no treinta y un ficheros de
    datos ni una puerta que no se puede escribir.

    **Lo que esto retira, dicho para que conste:** la idea de una casilla de
    "canario de enlaces". En `ETAPAS.md` NO habia tal casilla que retirar; la
    unica casilla de canario de la etapa 6 (linea 111, "Canario diario contra
    cuentas sandbox reales") es la de los CONECTORES contra cuentas de los cuatro
    proveedores, que no tiene nada que ver con esto y sigue en pie. Lo que se
    retira es el parrafo de aqui que la reclamaba para los enlaces del corpus.

    Queda vivo el hallazgo tecnico por si sirve en otro sitio: para EUR-Lex, ir
    por Cellar (`publications.europa.eu/resource/celex/...` con negociacion de
    contenido) esquiva el 403 del portal.

### Del frente del latido (26-08-2026)

38. **El planificador de la etapa 2 es el cron del operador.** No hay todavia un
    proceso propio que corra ciclos: quien apunta que ha corrido es
    `plazum latido ciclo`, programado en un temporizador. El vigilante ya esta
    entero y no cambia cuando exista el planificador de verdad, solo cambia
    quien escribe la marca. Lo que hay que revisar ese dia: que el planificador
    escriba la marca al TERMINAR el ciclo y no al empezarlo, porque uno que se
    cuelga a mitad seguiria dando senales de vida.

39. **`plazum doctor` no comprueba el planificador.** El comprador que teclea
    doctor porque algo no va no se entera de que su planificador lleva dos dias
    parado; tiene que teclear `plazum latido`. Son dos comandos para una sola
    pregunta ("¿esto funciona?"). El arreglo es una comprobacion mas en
    `adaptadores/diagnostico`, que llama a `pantalla.Vigilar` igual que la
    pantalla y la terminal. No se hizo aqui porque ese paquete es de otro
    frente.

40. **"Su ultimo ciclo termino hace 0 horas".** Durante la primera hora despues
    de un ciclo, el contador en horas enteras da cero y la frase suena rara. Se
    arregla con una forma mas en el catalogo ("hace menos de una hora"), que hoy
    `elegirForma` no sabe elegir: solo distingue singular de plural por el
    contador. Es el pendiente 31 de esta misma lista con otra cara.

41. **`latido.json` se escribe sin candado.** Dos ciclos solapados (un
    temporizador que se dispara mientras el anterior sigue) pueden pisarse la
    escritura. El dano maximo es perder una marca de pulso y mandar un pulso de
    mas; la marca del ciclo la reescribe el siguiente. Un fichero de bloqueo
    aqui es barato y no se ha puesto porque hoy nadie lo escribe en paralelo.

42. **El receptor del pulso no existe.** `plazum.dev/latido` es el destino por
    defecto y todavia no hay nada al otro lado, asi que quien active el pulso
    contra el de por defecto vera el canal en amarillo. La pantalla dice, con
    esas palabras, que eso no toca sus plazos, pero el amarillo se ve. Va con la
    casilla del dominio y con la decision de hacer publico el repo.

### Del frente de copias y restauracion (26-08-2026)

43. **Litestream no se ejercita en CI, y por eso la casilla dice "documentado" y
    no "probado".** El adaptador de almacen no existe, asi que la base todavia
    no es un fichero de SQLite y no hay nada que Litestream pueda replicar. El
    ensayo monta la instalacion con los tipos definitivos del nucleo y mide la
    propiedad que importa (que la copia devuelve algo que verifica), y esa
    propiedad no depende del formato. Lo que queda sin puerta hasta que llegue
    el almacen es la herramienta de replicacion en si: su configuracion, su
    retencion y su `litestream restore`. Cuando llegue, el ensayo cambia de
    fuente de bytes y no de comprobaciones. **P1 de la etapa del almacen.**

44. **La retencion de 35 dias es una politica, no una comprobacion.** El ensayo
    caza que la instalacion RESTAURADA no traiga una clave suprimida, y caza que
    el keystore restaurado no sea anterior a un borrado. Lo que nadie mide es
    que las generaciones ANTERIORES de la replica, las que si contienen esa
    clave, hayan expirado dentro del plazo declarado. Mientras eso sea manual,
    "el borrado es efectivo para el mundo a los 35 dias" es una afirmacion sin
    respaldo ejecutable, y es la afirmacion que va escrita en la politica de
    privacidad y en la lapida. Lo que haria falta: que la replica del keystore
    exponga sus generaciones con su instante y que el ensayo compruebe que
    ninguna anterior al ultimo borrado sigue viva pasado el plazo. **P1.**

45. **El keystore del ensayo se escribe en claro.** `docs/guia.md` dice que el
    keystore va "cifrado con la clave maestra". El ensayo no lo cifra, y es una
    desviacion consciente: elegir hoy la derivacion de clave y el formato de ese
    fichero seria decidir, desde una herramienta de prueba, algo que la guia
    asigna al adaptador de almacen, que despues tendria que heredarlo o
    romperlo. Consecuencia que conviene ver antes de implementarlo: si el
    keystore va cifrado con la maestra, y la maestra NO viaja en la copia (que
    es lo correcto), entonces restaurar exige reponer antes la maestra desde la
    custodia, y el paso 3 del procedimiento de `docs/copias.md` pasa a ser
    obligatorio en vez de opcional. **P1.**

46. **`plazum doctor` no sabe verificar una restauracion.** Quien restaura a las
    tres de la manana tiene el binario del producto, no `herramientas/`. La
    comprobacion de "lo restaurado prueba algo" vive hoy en `ensayocopia`, que
    no se distribuye. Su sitio natural es `plazum doctor`, junto a la
    comprobacion de keystore que ya existe alli. No se hizo en esta casilla
    porque `cmd/plazum/` es de otro frente. **P1 de la pasada del comprador.**

47. **La cadena que siembra el ensayo no lleva checkpoints.** Cerrar uno exige
    un sello de una autoridad de sellado, o sea red, y un ensayo de respaldo que
    necesita red no se puede correr el dia que no hay red. El anclaje temporal
    se ejercita en el otro tramo, el que pasa el expediente publicado por la
    copia y lo verifica con `plazum verify`. Queda sin cubrir un caso concreto:
    una base restaurada cuyo CHECKPOINT venga manipulado y cuyo expediente no
    viaje en la copia. **P2.**

48. **El manifiesto de la copia no es integridad frente a un adversario.** Quien
    pueda escribir en la replica reescribe tambien el manifiesto. Sirve contra
    la copia a medias y el disco que miente, que es lo que pasa de verdad, y asi
    esta escrito en `docs/copias.md` y en el godoc. La integridad frente a
    alguien que quiere enganar la da la cadena, que se comprueba contra claves
    que aporta el receptor. No se arregla: firmar el manifiesto con la maestra
    lo mejoraria, y la maestra no viaja en la copia a proposito. **P2, cerrado
    por diseno.**

### Del vendorizado de pkcs7 (26-08-2026)

49. **La copia vendorizada sigue aceptando SHA-1 para la firma del token.**
    `getSignatureAlgorithm` mapea a `x509.SHA1WithRSA` y `CheckSignature` lo
    admite. `VerificarOffline` ya exige SHA-256 para el `messageImprint`, que es
    lo que ata el sello al contenido, asi que el riesgo real es bajo y ninguna
    TSA seria firma asi hoy. No se ha tocado porque cambiar la politica de
    algoritmos de una copia recien vendorizada es una decision aparte de
    vendorizarla, y porque el primer sello legitimo que se rechace por esto
    costaria mas que el ataque que evita.

50. **La rama de cabeza de `pkcs7` quita del todo la verificacion de firmas
    DSA**, y a esta copia le vendria bien. No se ha portado por lo mismo que el
    49. Es un endurecimiento de una linea cuando se decida.

51. **`govulncheck` no ve el directorio vendorizado.** Empareja por ruta de
    modulo y esto ya no es un modulo. Hoy da igual, porque el mismo `pkcs7` sigue
    entrando como transitiva de `timestamp` y es la misma version; el dia que las
    dos se separen, esta linea es la que hay que releer.

### De la revision hostil del vendorizado (26-08-2026)

52. ~~**`opts.KeyUsages` a nil sigue queriendo decir `ExtKeyUsageAny`**.~~
    **CERRADO el 26-08-2026, recorte 5.** El motivo que lo dejó abierto ("el
    único llamante pasa siempre `ExtKeyUsageTimeStamping`") era **literalmente
    el mismo** que se había rechazado una ronda antes para `opts.Roots`: es una
    guarda del llamante, y vendorizar existe para no depender de que el de
    arriba se porte bien. Depender de que el de abajo se porte bien es el mismo
    error mirando al otro lado.

    Y era peor de lo que decía este apunte: no se heredaba un valor cero
    permisivo, **la copia lo ensanchaba** con cuatro líneas propias.
    `crypto/x509` con la lista vacía usa `ExtKeyUsageServerAuth` y **rechaza**
    un sello de tiempo diciéndolo; el código vendorizado lo convertía en "sirve
    para cualquier cosa", en silencio.

    Centinela `ErrSinUsos`, exigido **por longitud y no por nil**, que es lo que
    pide el invariante 8. Mutación demostrada por las dos caras: borrar la
    guarda pone rojo el test hostil y el fuzzer, y cambiarla por `== nil`
    también, que es lo que demuestra que recorrer las dos formas no es adorno.

53. ~~**El TSTInfo del que sale el veredicto y el contenido cuya firma se
    comprueba se emparejan por NADA.**~~ **MUERTO, y su rastro tambien
    (26-08-2026).**

    Por la manana subio de P2 a P1 con el triaje de aguas arriba. Por la tarde
    dejo de existir: `timestamp` se quedo solo como constructor de la consulta y
    el TSTInfo pasó a leerse con `encoding/asn1` sobre el contenido de la copia
    vendorizada. Un parser, los mismos bytes.

    **Y por la noche se quitó también el constructor**, que eran cuarenta líneas
    de ASN.1 sobre una estructura de seis campos. Con `timestamp` fuera salió
    `github.com/digitorus/pkcs7`, que era quien lo arrastraba, y el binario pasó
    a compilarse con **cero dependencias externas**.

    La diferencia importa en un producto de seguridad: *"no lo llamamos"* es
    cierto y el comprador no lo puede comprobar sin leerse el código; *"no
    está"* se comprueba con `go list -deps ./cmd/plazum | grep digitorus`.

    **Una consecuencia que no estaba en el plan y conviene dejar escrita**: el
    "excess walk" de `ber.go` (un `bytes.Index` que hacía cuadrático el parseo
    de secuencias de longitud indefinida) llevaba desde el triaje sin portarse, y
    el motivo era que abriría un recorte declarado en el camino de derivación
    del contenido, que es donde la puerta de los dos parsers existía para que no
    hubiera ninguno. Al morir esa puerta, el port se desbloqueó. **Encoger
    desbloquea arreglos que duplicar bloqueaba.**

54. ~~**El deber heredado sigue siendo un procedimiento sin puerta.**~~
    **CERRADO el 26-08-2026** con `.github/workflows/vigilancia.yml`: mensual,
    fuera del pipeline de PR, y **sin comparar ninguna fecha con el reloj**, que
    era la objeción correcta de este apunte. No falla: abre un issue, porque un
    rojo mensual permanente es tan invisible como un verde falso.

    **Y no es un adorno: al ejecutarlo a mano contra el repositorio real, aguas
    arriba va 40 commits por delante y los CUATRO ficheros vendorizados han
    cambiado, `ber.go` incluido (+8/-6).** Eso es exactamente lo que llevaba sin
    vigilancia desde que se vendorizó.

    Dos cosas lo cazaron mientras se escribía, y ninguna leyendo el código:

    - `TestTodoPasoDeCIEsShellQueBashSabeParsear`: el terminador de un heredoc
      va en la columna 0, y **la columna 0 dentro de un `run: |` está fuera del
      bloque**. El heredoc no rompía el shell, rompía el YAML, y el paso dejaba
      de ser el que se había escrito.
    - la guarda de "no he podido preguntar" decía `-z` y no valía: **`gh api`
      escribe el error en stdout** y sale con 1, así que con un repositorio
      inventado devuelve 118 caracteres de JSON que no están vacíos. Se
      compararían con el sha vendorizado, saldrían distintos, y se abriría un
      issue diciendo que aguas arriba se ha movido cuando lo que pasa es que no
      se ha llegado a preguntar. Ahora se exige la **forma** del sha y el estado
      de salida. Demostrado contra un repositorio que no existe.

---

## La familia: piezas terminadas sin el cable (02-09-2026)

Cuatro casos en cuatro días, y **ninguno puso roja una puerta**. No son cuatro
despistes: es un agujero estructural del sistema de puertas. Había puertas que
verifican **piezas** y ninguna que verificara **juntas**, y cada mitad pasaba la
suya.

| # | pieza | la mitad que faltaba | ¿lo vio alguna puerta? |
|---|---|---|---|
| 1 | primitiva `maximo` | encendida en el motor, apagada para el corpus, 8 relojes del CRA esperando | no |
| 2 | `superficies/acta` | entera y con tests verdes, sin una sola dirección del producto que llevara a ella | no |
| 3 | el camino guiado | el enlace a la entrevista existía y viajaba **vacío**: se comía las respuestas | no |
| 4 | la entrevista y el motor | hablan idiomas distintos y nadie comprobaba que el primero pueda alimentar al segundo | no |

**Cerrado con el trinquete de alcanzabilidad**, en tres mitades, todas
enumerando **desde el árbol** y no de una lista escrita al lado:

- `cmd/plazum/alcanzabilidad.go` — todo paquete de `superficies/` que declare
  `ServeHTTP` con la firma de `http.Handler` está en el censo, con vocabulario
  cerrado y **valor cero prohibido**. El emparejamiento con el montaje real es
  **por la ruta de import del handler montado**, sacada por reflexión: la
  primera versión emparejaba por una cadena escrita a mano y `scim` podía
  declararse montada en el paso de la UAR y heredar su cable (invariante 7,
  décimo caso de la familia de guardas que no guardaban).
- `nucleo/corpus/primitivas_encendidas.go` — por cada primitiva del motor, o hay
  un paquete que la usa o hay una línea con su motivo y su cardinal. Si el
  corpus la sabe construir se le pregunta **a `VencimientosDe`** con el
  centinela `ErrPrimitivaSinEjecutor`, no a una copia de su `switch`.
- `entrevista_alcanza_al_motor_test.go` — el hueco entre la entrevista y el
  motor, congelado por igualdad exacta.

### Lo que el trinquete encontró al nacer, con su cardinal

1. **`observacion` es candidata a borrado. Cardinal 0.** Cumple hoy las **tres**
   condiciones enteras por las que se borró `Secuencia` el 02-09-2026:
   `VencimientosDe` no la construye (ningún `paquete.json` puede declararla,
   invariante 2), ningún reloj contado la pide, y su `Ventana` es un `Intervalo`
   fijado en la estructura, que es el mismo defecto de diseño que descartó a
   `Secuencia` (un paquete de corpus no puede saber entre qué dos fechas observa
   un cliente concreto). **Decisión de producto pendiente: borrarla o cablearla.**
2. **`preaviso` está apagada. Cardinal 7** (censo, familia G). Está cableada
   desde el 02-09-2026 y **cero paquetes** la declaran. No es deuda de código: un
   paquete puede usarla hoy sin tocar Go. Los siete relojes están **fuera** de
   los 12 marcos de la v1 (`psd2`, `mica`, `mdr`, `data-act`), así que no bloquea
   la v1.
3. **`superficies/scim` no la monta nadie, y está bien.** Su constructor exige
   token y colgarla del servidor de la interfaz sería una puerta trasera con
   forma de estándar. Lo que faltaba era que **constara**: sin esa línea era
   indistinguible del caso del acta.

### P0: `plazum serve` no puede exportar un `alcance.json` fiel. Cardinal 36 de 41

**Bloquea la casilla tal y como estaba escrita**, y el hueco no es el botón de
descarga.

- la entrevista de `superficies/pantallas` recoge **sí o no** sobre un id de
  pregunta, y nada más: `pantallas.De` sólo lee los parámetros `si` y `no`;
- el motor de aplicabilidad consume **hechos tipados**, con su valor dentro:
  `ambito(sistema, publico)`, `categoria(sistema, alta)`.

Medido con el parser de verdad sobre el corpus instalado: de las **41** preguntas,
**5** se traducen desde un sí/no (su atributo lo usa alguna regla como predicado
unario), **16** tienen su atributo usado **siempre** con dos o más argumentos (la
entrevista no pregunta el valor, y afirmarlas exigiría inventárselo) y **20**
tienen un atributo que **no usa ninguna regla**.

Un `alcance.json` exportado hoy cargaría **sin error** y llevaría dentro mucho
menos de lo que aparenta: un fichero con la **forma** de la respuesta completa.
Eso es lo peor que este producto puede producir, así que el exportador no se
escribe hasta que se decida el puente. **Las dos salidas, y son de producto:**

- **A**: la entrevista aprende a llevar valores (un parámetro `val=id:valor`
  además de `si`/`no`), con su pantalla. Toca la pantalla principal del producto.
- **B**: el paquete **declara** cómo se traduce una pregunta a un hecho. Toca el
  esquema del corpus y por tanto los 30 paquetes.

**No se midió cuántas obligaciones derivaría** porque para eso hay que decir qué
hecho produce un «sí», y **esa traducción no existe en ninguna parte**: cualquier
número saldría de una regla inventada para medir. Se intentó y dio dos resultados
incompatibles (0 de 26 mirando cuerpo a cuerpo, 16 de 171 por punto fijo, porque
los predicados se comparten entre paquetes); ninguno auditable, fuera los dos.

### P1: el acta sólo puede leer 1 de sus 3 fuentes

`cmd/plazum/serve_acta.go` cablea la que se puede y **dice cuál no**:

| fuente | ¿se lee de disco? | por qué |
|---|---|---|
| campaña de accesos | **sí** | `censo.Tomar` + `accesos.Reconstruir`, ya probados, reutilizando `campanaEnFichero` |
| registro de incidentes | **sí, desde el 02-09-2026** | `nucleo/incidente.Reconstruir` lee el fichero y lo **replica por `Abrir` y `Registrar`**: un incidente leído de disco pasa por las mismas reglas que uno creado a mano. Se conecta con `--acta-incidentes` |
| programa de auditoría | **no. Cardinal: 1 de 3 fuentes pendiente** | `nucleo/auditoria` no tiene formato en disco ni reconstrucción: un `Programa` se construye llamando a `Auditar`, `Diferir`, `Anotar` y `Cerrar`, y **ninguna orden de plazum escribe esos hechos**. Es la última que falta |

**El registro de incidentes se cerró replicando por los constructores, nunca
rellenando campos privados.** Es la única forma honesta de leer un objeto cuyo
valor está en sus reglas: un fichero corrupto no produce un objeto a medias,
produce un error. Hay test que lo demuestra, con su control positivo.

**Y la distinción que justifica el campo `HayRegistroDeIncidentes` ahora se
recorre en las dos direcciones**, con dato real:

- **sin `--acta-incidentes`**: la sección dice que su fuente no está conectada.
  plazum no sabe si hubo incidentes.
- **con la bandera y el fichero vacío**: la sección dice que no hubo incidentes en
  el periodo. Es una **afirmación**, y plazum sólo la puede hacer porque alguien
  le ha dado el registro.

Sin la segunda rama, «cero incidentes» sería una rama que ninguna entrada alcanza
y la mutación la dejaría verde porque no hay nada que romper (M47).

**Lo que queda es el programa de auditoría**, y no es «el mismo patrón ya
probado»: es una pieza que no existe. Falta un formato en disco y una orden que
lo escriba.

---

## El puente entre la entrevista y el motor: piloto hecho, y lo que queda (02-09-2026)

**Decidido por Marcos: B primero, A detrás.** El razonamiento es del invariante
2, no de gusto: que «¿tratas datos personales?» produzca
`trata_datos_personales(E)` es **conocimiento normativo**, y el conocimiento
normativo vive en el paquete o no vive. Si esa tabla la escribe Go, se ha
cableado una norma por la puerta de atrás.

### B: hecho, y el piloto mueve el número

El esquema es `nucleo/corpus/puente.go`: cada atributo de entidad declara un
bloque `hecho` con **vocabulario cerrado** de tres formas.

| forma | qué significa | tipo de atributo |
|---|---|---|
| `afirma_si` | un «sí» afirma `predicado(instancia)` | booleano |
| `con_valor` | la respuesta afirma `predicado(instancia, valor)` | enumerado, texto, entero, fecha |
| `no_llega_al_motor` | este atributo no alimenta ninguna regla, **con su motivo obligatorio** | cualquiera |

**Un «no» no afirma nada**, y es deliberado: en este motor la ausencia de un
hecho no es su negación (hay negación explícita para quien la quiera), así que
hacer que un «no» afirmara algo metería en el expediente una afirmación que el
operador no ha hecho. Tiene control positivo propio, porque el escenario del
piloto contesta que sí a todo y sin ese caso la rama no la recorre nadie (M47).

El linter empareja **por el nombre del predicado**, y las dos puntas están dentro
del mismo paquete firmado (el `hecho` del atributo y la regla que lo lee, en el
mismo `paquete.json`): no hay índice ni posición por medio (invariante 7). Y
comprueba las dos direcciones: que el predicado lo use alguna regla, y que lo use
**con la aridad que le toca por el tipo del atributo**, no por lo que el autor
creyera.

**La medida del piloto (`ens`, 17 preguntas, el que más tiene de los 12): 25
obligaciones derivadas** desde 17 hechos, con el escenario declarado en el test.
Antes de esto el número no se podía dar: había que decir qué hecho produce cada
respuesta, y esa traducción no existía en ninguna parte.

Y el 25 dice más de lo que parece: **no son 25 obligaciones de `ens`**. Son 25 de
todo el corpus, porque los predicados se comparten entre paquetes. Contestar que
el sistema trata datos personales enciende obligaciones del RGPD, y describir la
información que maneja enciende una de interoperabilidad. Eso es el corpus
funcionando como se diseñó, y un recuento por paquete no lo habría visto.

**El bloque es opcional mientras dure el piloto**, a propósito: se valida cuando
está, y no se han tocado los otros 29 paquetes. Cuando se decida que el diseño
sirve, pasa a obligatorio y el valor cero (no declararlo) será el que rompa, como
en `origen_del_intervalo`.

### A: pendiente, y ahora se sabe exactamente qué cuesta

La pantalla tiene que aprender a preguntar por valores, porque **16 preguntas
tienen su atributo usado siempre con aridad ≥2 y un sí/no no tiene dónde meter el
valor**. `pantallas.De` sólo lee `si` y `no`. La función que traduce respuestas a
hechos ya está escrita y medida en `puente_piloto_test.go`: cuando la pantalla
aprenda a llevar valores, lo que hará es exactamente eso.

### El hallazgo aparte, con su cardinal: 20 preguntas huérfanas

**20 de las 41 preguntas del corpus tienen un atributo que no usa ninguna
regla.** No es lo mismo que el problema del valor: éstas no llegarían al motor ni
llevando el valor puesto. Cada una es una de dos cosas, y las dos son deuda con
nombre:

- **le falta la regla**: la pregunta es buena y nadie escribió la regla que la
  lee. Se arregla escribiendo la regla.
- **la pregunta sobra**: no condiciona nada y se recoge por costumbre. Se arregla
  borrándola, o declarándola `no_llega_al_motor` con su motivo si vale para
  documentar el alcance aunque no derive.

Reparto medido: `iso27001` 8, `iso42001` 3, `demo-empresa` 3 (más 3 fechas), y
una cada uno en `ai-act`, `dora`, `mdr`, `nis1-es` y `nis2-ue`. El mecanismo para
declararlas ya existe (`no_llega_al_motor`); lo que falta es ir marco por marco
decidiendo cuál de las dos cosas es cada una, y eso **no se hace en bloque**:
cada una exige leer la norma.

## La capa visual: lo que entró y lo que quedó fuera, con su cardinal (02-09-2026)

El armazón, la identidad, el panel de inicio y la hoja de impresión entraron. Lo
que sigue está **medido**, no estimado, y es lo que el frente visual **no** hizo.

### P1: tres de las cuatro superficies con pantalla no tienen armazón

La barra lateral con el camino guiado la pintan **las seis pantallas** de
`superficies/pantallas`. **El camino, el acta y la revisión de accesos siguen sin
ella**: cada una arma su propio `<body>` y su propio `Vista`, y su única salida
es el enlace `camino-vuelta`, que ya existía. Son **3 superficies** y **4
plantillas** (`camino.html`, `acta.html`, `uar.html`, y la pantalla de derivación
del acta va dentro de la suya).

No se hizo porque darles el armazón exige que las tres importen
`superficies/camino`, ganen un campo `Tira` en su vista y reordenen su plantilla:
es un cambio en el Go de tres superficies que otros frentes tocan, y este bloque
se cerró contra interfaces congeladas. **Lo que sí comparten ya**: la hoja de
estilo entera, la tipografía, la paleta medida, el icono de pestaña y la hoja de
impresión, que es la que de verdad le importa al acta.

### P2: «312 obligaciones que te alcanzan» antes de responder nada

Con los 30 paquetes instalados y la entrevista **sin responder**, el panel de
inicio dice **312**. No es un fallo del panel: es la semántica de la derivación,
donde una obligación sin preguntas de aplicabilidad aplica sin condiciones
(`derivacion.sin_condiciones`), y la mayoría de los paquetes son esqueletos que
todavía no declaran preguntas. La cifra lleva su matiz escrito al lado («todavía
no has respondido nada, así que aquí solo están las que alcanzan a todo el
mundo») y abre a `/controles?f=aplica`, donde cada fila dice por qué.

**El número baja escribiendo reglas de aplicabilidad en los paquetes, no tocando
la pantalla.** Se anota aquí porque es el primer número que ve un comprador y
porque la tentación barata sería esconderlo.

### P2: la hoja de impresión no tiene puerta

`@media print` cubre el acta y el board pack: se va la navegación, se queda todo
el contenido con su descargo y su atribución, las filas no se parten entre
páginas y los enlaces internos se imprimen con su dirección detrás. **No hay
ninguna puerta que lo compruebe**: axe audita la pantalla, no el papel, y no hay
forma barata de afirmar «esto sale bien impreso» sin renderizar a PDF y
compararlo. Queda como revisión manual, dicho para que no se dé por vigilado.
