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

**Once en dos semanas**, y las once del mismo tipo. No son casos borde: son la
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

## P2

### De la higiene legal del corpus (26-08-2026)

- **La fuente oficial se pinta como texto, no como enlace.** El pie del producto
  ensena la atribucion de cada paquete con la direccion completa de su fuente,
  pero sin `href`. Son dos razones y las dos son de esa superficie:
  `TestHtmxVaVendorizadoYNoPorCDN` prohibe que la pagina apunte a nada de fuera,
  y htmx va con `selfRequestsOnly`, asi que un enlace externo dentro del cuerpo
  con `hx-boost` no navegaria. La condicion de reutilizacion es citar la fuente,
  y la direccion a la vista la cita, asi que no es un incumplimiento; es peor
  experiencia. Arreglo cuando esa superficie decida su politica de enlaces
  externos: acotar la prohibicion a `src=` y a `<link>`, que es lo que de verdad
  vigila, y sacar el enlace del boost.
- **El `urn` de `eidas2` y de `csrd` sigue nombrando al acto modificativo.** La
  `fuente` ya apunta al instrumento donde viven las obligaciones y el `LEEME.md`
  de cada uno lo dice, pero el identificador no se ha tocado: cambiarlo cambia la
  identidad del paquete en el expediente y en las equivalencias. Es decision de
  la autoria del corpus, no de higiene.
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
   importar `plazum/nucleo/...`, asi que no puede delegar la lectura en otro
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

36. **Dos `fuente` caducadas, encontradas siguiendo los 31 enlaces.** `soc2`
    apunta a `aicpa-cima.com/topic/audit-assurance/...` y redirige a
    `/resources/landing/system-and-organization-controls-soc-suite-of-services`;
    `stig` apunta a `public.cyber.mil/stigs/` y redirige a `www.cyber.mil/stigs/`,
    que es un cambio de anfitrion. Ninguna de las dos esta rota hoy, pero un
    redirect es un aviso de mudanza y las dos son del estrato referencial o
    delegado, donde `fuente` es lo UNICO que el paquete aporta.

37. **No hay forma barata de vigilar que los enlaces sigan vivos.** Se intento y
    no sirve: EUR-Lex e ISO responden **403 a una peticion con guion**, asi que
    un comprobador automatico no distingue "el enlace se ha muerto" de "me han
    tomado por un robot", y una puerta que confunde esas dos cosas es peor que
    no tenerla. Lo que si funcionaria es lo que ya descubrio el frente del
    extractor: **para EUR-Lex, ir por Cellar** (`publications.europa.eu/resource/celex/...`
    con negociacion de contenido) en vez de por el portal. O sea que la vigilancia
    de enlaces del DOUE es gratis reusando `herramientas/ingestanorma`, y la de
    ISO no lo es. Encaja con la casilla de "canario diario fuera del pipeline de
    PR" de la etapa 6, no con una puerta de CI.

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
