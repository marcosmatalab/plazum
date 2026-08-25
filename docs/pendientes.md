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
