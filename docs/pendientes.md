# Pendientes: el registro de P1 y P2

Los hallazgos que no bloquean la casilla en la que salieron, para que no se
pierdan en el cuerpo de un commit.

Clasificacion, la del protocolo de las tres pasadas (`CLAUDE.md`):

- **P0** bloquea la casilla. No entra aqui: se arregla antes de marcar.
- **P1** entra en la etapa. Se arregla dentro de la etapa en curso.
- **P2** a la lista. Se arregla cuando toque o se decide que no.

Cuando algo se cierra, se borra de aqui y consta en el commit que lo cerro.

---

## P1

### Del corpus (frente de autoria, 25-08-2026)

1. **El limite de texto solo vigila `texto_legal`.** La frontera legal (120
   caracteres en un paquete referencial) no mira `ayuda`, `cita` ni el titulo de
   una obligacion. Se puede meter texto de ISO por cualquiera de esos tres y el
   linter no dice nada. Es el mismo agujero que la clase fuera de rango, por
   otra puerta.
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
5. **No hay forma de ver un paquete.** Ni un `dutiq corpus ver <urn>`. Para
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

### Del frente de identidad, OIDC y SCIM (25-08-2026)

16. **No existe `dutiq scim token`.** El servidor SCIM exige un token de
    aprovisionamiento de al menos 32 caracteres y no hay forma de generarlo con
    el producto: el operador tiene que inventarselo. El mensaje de error ya NO
    nombra el comando (nombrar uno que no existe quema la confianza en el resto
    de los mensajes), pero el hueco sigue. Es de `cmd/dutiq`, que es de otro
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

## P2

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
   importar `dutiq/nucleo/...`, asi que no puede delegar la lectura en otro
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
    proxy con TLS delante y no se lo dijo a dutiq. Si un escaner de conformidad
    de un comprador lo marca, se condiciona a `X-Forwarded-Proto`.
17. **La cookie usa `SameSite=Lax` y no `Strict`.** Con Strict, llegar desde el
    enlace de un correo de escalado ensena la pantalla como si no hubieras
    entrado. En la etapa 4, cuando esos correos existan, medir si compensa
    Strict con una pagina puente.
18. **El diagnostico no ve todavia el estado del servidor.**
    `Limitador.Vaciados()` y `Sesion.Vivas()` existen y nadie los lee. Al
    construir `dutiq doctor`, conectarlos como dos comprobaciones con su arreglo.
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
22. **`dutiq serve` no existe todavia como orden.** Este frente entrega el
    servidor como biblioteca y un binario de pruebas bajo
    `superficies/serve/internal/servidorprueba` que solo usa la puerta de CI.
    Quien instala dutiq hoy no puede arrancarlo: el cableado de `cmd/dutiq` es
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
   descubrimiento y no se usa: cerrar sesion en dutiq no cierra la del IdP.
26. **`meta.version` se emite y `/ServiceProviderConfig` declara `etag` no
   soportado.** No es contradictorio (el ETag de SCIM es la cabecera, no el
   campo), pero es confuso de leer y algun IdP podria intentar usarlo. O se
   implementa el control de concurrencia optimista o se quita el campo.
27. **No hay SAML.** Apuntado para el ano 2 y dicho en `docs/identidad.md` para
   que nadie lo busque.

