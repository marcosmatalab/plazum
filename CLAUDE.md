# plazum

GRC open source de continuidad de cumplimiento. Motor determinista de obligaciones con reloj legal, corpus normativo como paquetes de datos, expediente verificable offline. Go puro. AGPL-3.0. Una persona lo construye por etapas: el plan vive en `ETAPAS.md` y el detalle completo en `docs/guia.md` (fuente única del plan) y `docs/diseno.md`.

## Comandos

```bash
./comprobar.sh          # EL OBJETIVO ÚNICO: formato, vet, build y las 21 puertas con su recuento
go build ./...          # compilar todo
go test ./...           # los tests; TODOS en verde siempre. Para depurar, NO para afirmar
go test . -v            # los tests de raíz: arquitectura (AST), extensibilidad, linter de paquetes
gofmt -l $(git ls-files '*.go')   # vacío siempre; `gofmt -l .` entra en los worktrees de .claude/
go vet ./...            # limpio siempre
GOPROXY=off go test ./...  # la suite entera sin acceso a red (también es puerta de CI)
```

No hay npm, no hay Makefile, no hay generadores en el producto. El CI sí puede usar herramientas de node (axe-core) sin que eso contradiga lo anterior. Si un comando nuevo hace falta, se documenta aquí.

## Invariantes de arquitectura (vigiladas por tests, no negociables)

1. **`nucleo/` no importa NADA externo ni llama a `time.Now()`.** El instante entra siempre como dato. Lo vigila `arquitectura_test.go` leyendo el AST. Si un cambio lo rompe, el cambio está mal, no el test.
2. **Ninguna norma está cableada en el código.** Los identificadores tipo `ens@`, `iso27001@`, `cis@` en un literal de cadena rompen el build (`extensibilidad_test.go`). Toda norma vive en su paquete de datos bajo `paquetes/` o no vive.
3. **Todo paquete publicado pasa el linter** (`paquetes_test.go` ejecuta `corpus.Cargar("paquetes")`). El linter es la frontera legal: un paquete referencial con más de 120 caracteres de texto normativo NO carga. Jamás pegar texto de ISO, PCI DSS, SOC 2, TISAX o CIS: identificador y título corto como máximo. BOE y DOUE sí se transcriben (art. 13 TRLPI, Decisión 2011/833/UE), siempre con `fuente` enlazada. **Y no se copia contenido normativo de repositorios de terceros de GitHub, aunque digan MIT o Apache: la licencia de un repositorio no alcanza a contenido que el subidor no poseía.** Solo fuente primaria: NIST, EUR-Lex, BOE.
4. **La IA solo entra por el puerto `Asistente`** y su única salida es `puertos.Propuesta` con cita verificada por hash. Los adaptadores de LLM viven fuera de proceso y JAMÁS importan escritura de estado o ledger.
5. **Dependencias**: lista cerrada en `DEPENDENCIAS.md`. Añadir una exige una línea allí con su porqué y su licencia. En `nucleo/`, cero, para siempre.
6. **OSCAL puede ser adaptador de SALIDA con pérdidas, nunca modelo interno ni formato de entrada.** Su modelo (`catalog > group > control > part`) no tiene campo para un plazo: es el mismo agujero que el `RequirementNode` de CISO Assistant. Doblar nuestro modelo para hacer ida y vuelta con OSCAL borra el diferenciador entero, que es el reloj legal. Exportar pierde los plazos y se dice que los pierde; importar nos obligaría a no tenerlos. El porqué completo, con el dato de adopción, en `docs/decisiones.md` D-1.
7. **Toda comprobación que empareje dos conjuntos lo hace por una identidad que está DENTRO de lo firmado. Nunca por índice, posición ni orden.** Nadie firma el orden: reordenar o insertar mueve el emparejamiento entero sin romper ninguna firma. Es el fallo del ataque 13 (la cadena y las observaciones) y el de la guarda del borrado legal del export a SIEM, que son el mismo. En la pasada 2, por cada emparejamiento NUEVO hay que decir en voz alta **por qué campo casa y si ese campo está firmado**. La familia entera, en `docs/pendientes.md`.
8. **En una frontera de confianza, el valor cero de una estructura de opciones tiene que ser el RESTRICTIVO, o estar prohibido explícitamente. Todo test de ausencia recorre las DOS formas: `nil` y vacío-presente.** En Go el valor cero de unas opciones suele significar *permisivo*: `x509.VerifyOptions.Roots` a `nil` acepta cualquier CA, `KeyUsages` a `nil` es `ExtKeyUsageAny`, un slice a `nil` es "sin restricción". El vacío-pero-presente significa lo contrario: `x509.NewCertPool()` no confía en nadie. **Las dos formas de la nada no son la misma, y la peligrosa es siempre la `nil`, porque es la que sale por olvidarse.** Es el hallazgo 15 (`pkcs7.VerifyWithOpts` encadenaba sólo dentro de un `if opts.Roots != nil`) y la razón de que su fuzzing no lo cazara: la afirmación recorría el almacén vacío, que es el inocuo. Se prohíbe con centinela (`ErrSinAnclas`) o se documenta por qué no hace falta. La subfamilia entera, con el barrido campo a campo, en `docs/pendientes.md`.
9. **La IA vive en adaptadores y superficies. `nucleo/` no conoce el puerto de IA y no lo importa NUNCA.** El cumplimiento es determinista; la IA sólo entra donde hoy hay fricción, y **su única salida es una `puertos.Propuesta` cuya cita se verifica por hash ANTES de enseñarla**: si no resuelve a texto real, la propuesta se descarta, no se muestra. Es la puerta antialucinación, y es mecánica, no estadística. Dos puertas lo vigilan: `TestElNucleoNoConoceLaIA` (el núcleo no importa el puerto **y ni siquiera lo nombra**, porque copiar el interfaz cumple la letra rompiendo el fondo) y un paso de CI que corre la suite entera con `PLAZUM_SIN_IA=1`, que convierte "el núcleo es determinista" de eslogan en hecho comprobable en dos minutos. La doctrina entera, con el arnés y dónde entra la IA por punto de fricción, en `docs/ia.md`.
10. **Un dato normativo que llegue del estratega, de un informe, de un artículo o de una sesión anterior es una PISTA, nunca una fuente.** Al corpus solo entra lo verificado contra fuente primaria (EUR-Lex, BOE, DOUE, NIST) **en el momento de escribirlo**, y la verificación se anota con su fecha en el cuerpo del commit: qué se miró, dónde, y qué día. Sin esa línea el dato no entra, aunque sea correcto: un acierto que nadie puede reauditar es indistinguible de un acierto por suerte. Vale para números de reglamento, fechas de publicación, de entrada en vigor y de aplicación, y para la numeración de artículos que un ómnibus renumera. Salió de un P0 real, y la regla se cobró su primera pieza al escribirla: la fecha del reglamento ómnibus llegó por una nota de estrategia y era falsa, y la corrección que la sustituyó traía dentro otra conflación. El dato verificado contra EUR-Lex (ELI `reg/2026/1744/oj`, consultado el 27-08-2026) es: Reglamento (UE) 2026/1744 **de 8 de julio de 2026**, que es la fecha DEL ACTO; **publicado en el DOUE el 24 de julio**; en vigor **el tercer día siguiente al de su publicación**, o sea el **27 de julio de 2026**. «De 8 de julio» y «publicado el 8 de julio» no son lo mismo, y la distancia entre las dos son dieciséis días de reloj legal. Las tres fechas de una norma de la UE (acto, publicación, entrada en vigor) se copian por separado o no se copian. Es hermano del invariante 3: aquel dice qué texto se puede copiar, este dice de dónde tiene que venir el dato para que se pueda escribir.


## Convenciones

- Dominio en español (`Obligacion`, `ventana`, `Calcular`), infraestructura en inglés donde sea idiomático. Comentarios en español, sin tildes en identificadores.
- Todo test de una propiedad de seguridad o legal lleva **control negativo**: se demuestra que el test falla cuando debe (patrón de `TestNingunaNormaCableada`).
- **Una puerta que nunca se ha visto fallar no es una puerta.** Toda puerta nace con su fallo demostrado y anotado en el commit: se rompe a propósito lo que vigila y se pega la salida roja. Sin eso no se sabe si vigila o si acompaña. Vale igual para un test, para un paso de CI y para un linter.
- **Ninguna invocación de `go test` en un workflow.** `go test` sale con código 0 cuando el patrón `-run` no casa con nada y cuando el glob de paquetes no tiene tests: los dos son verdes indistinguibles de un verde de verdad. Las puertas de CI pasan por `.github/puerta.sh`, que cuenta los casos ejecutados y exige un mínimo declarado. Lo vigila `puertas_test.go`.
- **Ningún resultado de test cuenta en un informe si no salió de la puerta.** El lazo local también, no solo CI: se ejecuta `./comprobar.sh`, que lee las puertas de `.github/workflows/*.yml` (no las declara: una segunda lista es una lista que se queda vieja) y las corre con su recuento. `go test` con `-run` a mano queda para **depurar**, nunca para **afirmar**. La tercera mordida de esta trampa no fue en CI: fue un `go test . -run "Paquetes"` que salió `ok` porque el patrón no casaba con el test que importaba, y ese `ok` viajó a un informe. Lo vigila `comprobar_test.go`, que exige que `PUERTAS_ESPERADAS` cuadre con las puertas que hay en CI **en los dos sentidos**.
- **Una puerta se demuestra en el shell en el que CORRE, no en el que la escribes.** GitHub ejecuta los pasos `bash` con `-e`, y `set -uo pipefail` no lo apaga: bajo `-e`, `salida=$(go test ...)` mata el shell antes de imprimir nada. Las cinco formas de fallo de `puerta.sh` se demostraron a mano en un shell sin `-e`, y por eso la sexta sobrevivio a la demostracion. Vale igual para un test que se prueba con `-run` suelto y luego corre dentro de la suite.
- **Antes de marcar una casilla, mirar que CI está en verde en `main`** (`gh run list --branch main`). Un rojo permanente es tan invisible como un verde falso: el bloqueante de `gosec` estuvo rojo cinco commits seguidos sin que nadie lo leyera.
- Todo reloj del corpus lleva **caso dorado** en `pruebas/` del paquete (formato en docs/guia.md Anexo B), derivado del texto legal, no de la implementación. Si motor y caso discrepan, gana el caso.
- Las **reglas de aplicabilidad las declara el paquete**, no el código, en el dialecto Datalog del Anexo C de `docs/guia.md`. Un paquete con reglas se prueba ejecutándolas contra el motor con las dos direcciones comprobadas (lo que aplica y lo que no, con el artículo de cada exclusión): el linter solo dice que la regla se parsea.
- Errores accionables: causa, arreglo, y cita si es del dominio. Nada de "error inesperado".
- En documentos para el usuario final: sin guiones largos, comas o dos puntos en su lugar.
- **La capa probatoria está cerrada.** Del ataque 14 en adelante, los hallazgos de la familia "el emisor mete la mano en el expediente" se DOCUMENTAN en `docs/modelo-de-amenaza.md`, no se arreglan. Única excepción: que el hallazgo rompa la promesa escrita en ese fichero. El porqué, en `docs/decisiones.md` D-2.
- Commits pequeños con el porqué en el cuerpo. Nunca commitear con tests en rojo.

## Estructura

```
nucleo/        ventana, aplicabilidad, estado, ledger (v1+v2 comprometido), blobs,
               historia (bitemporal), certificado, perimetro, expediente, corpus
               (con clase e2e, temporalidad, dorados y su ejecutor) - construido, 0 deps
puertos/       las 9 interfaces hexagonales (compilan, documentadas)
adaptadores/   por construir, etapa a etapa (ver doc.go)
superficies/   serve, api, portal, export (por construir; el CLI está en cmd/plazum)
cmd/plazum     verify, explain, estado (construido)
paquetes/      el corpus: los 30 marcos montados con su estrato legal (ver paquetes/CORPUS.md);
               ens, rgpd y cra con relojes reales y 12 dorados en verde; el resto, esqueletos
evals/         conjuntos dorados de IA (etapa 5)
docs/          diseno.md y guia.md: TODO el contexto del proyecto está ahí
web/           la web del open core (estática, sin build)
```

## Flujo de trabajo por etapa

1. Abrir `ETAPAS.md`, localizar la etapa en curso y su primera casilla sin marcar.
2. Leer la sección correspondiente de `docs/guia.md` (tiene los tipos, formatos y decisiones ya tomadas: **no re-decidir diseño**).
3. Plan mode para lo no trivial. Implementar con su test-puerta. `go test ./...` en verde.
4. Marcar la casilla en `ETAPAS.md` y commitear.
5. Al cerrar una etapa: pasar el comando `/adversarial` (revisión hostil) antes de declararla cerrada.

### Las tres pasadas (obligatorias antes de marcar cualquier casilla)

Una pasada que dice "todo correcto" sin enumerar qué intentó romper es una pasada fallida: se repite con otro ángulo.

1. **Contra la especificación.** ¿Es exactamente la casilla de `ETAPAS.md` y su sección de `docs/guia.md`? Nada de "es mejor así" sin decirlo en voz alta.
2. **Contra el atacante.** Emisor malicioso, entrada adversaria, reloj que miente, receptor hostil. **Mutación obligatoria**: borra la línea de la comprobación y demuestra que algo se pone rojo. Si no se pone rojo, ese test no existe.

   **La pasada 2 empieza con `git status` limpio: toda mutación se aplica sobre estado commiteado.** Con el árbol limpio, `git checkout <fichero>` restaura siempre y no puede tragarse trabajo; con el árbol sucio se lleva por delante lo que aún no estaba commiteado, y si el fichero es nuevo falla con «pathspec did not match» y **la mutación se queda puesta**. Las dos cosas pasaron el 27-08-2026, en el mismo día. Es la versión de proceso del invariante 8: el estado sin commitear es el valor degenerado del repositorio, y es el que sale por descuido.

   **Elige una propiedad que el código da por buena e intenta tumbarla.** Leer el diff encuentra lo que el autor hizo mal; refutar una propiedad encuentra lo que el autor no pensó, que es donde vive la familia entera de "sin confiar en el emisor". Así salió el ataque 13, que trece rondas de revisión de diff no habían visto. Patrón concreto: cuando una comprobación recorre una lista para contrastarla con otra, preguntar SIEMPRE si la dirección contraria también se recorre. La que falta es la que el emisor usa.

   Dos trampas de la mutación, las dos cometidas ya: una mutación que **no compila** no produce líneas `--- FAIL`, así que un fallo de build parece una mutación no cazada. Comprueba el estado de compilación aparte, y con `go build ./...` entero, no con un grep de `cannot|undefined`: `imported and not used` no contiene ninguna de esas palabras. Y comprueba con `git diff --stat` que la mutación se aplicó de verdad antes de leer el resultado: un `sed` que no casa da verde y parece un hallazgo.

   **Por cada emparejamiento nuevo, di por qué campo casa y si ese campo está firmado** (invariante 7). Nueve de los diez fallos de la familia "guardas que no guardaban" son esto.

   **Y por cada estructura de opciones que cruce una frontera de confianza, di qué significa su valor cero** (invariante 8). Si un test comprueba "sin X", tiene que recorrer `nil` **y** vacío-presente: son dos cosas distintas y la que se olvida es la permisiva.

   Y un verde puede CADUCAR: si el test cablea un instante y lo compara con algo que ocurre en tiempo real, es una bomba con la mecha ya encendida. Lo caza el horario diario de `ci.yml`, no una revisión.

   Y una trampa del test: una mutación que el propio test eligió no demuestra nada. Si el test prueba contra una lista escrita a su lado, muta **fuera** de esa lista.
3. **Contra el comprador.** Un CISO de 200 empleados abre esto a las 9 de la mañana, sin documentación y sin soporte. ¿Llega al valor? ¿Qué no entiende? ¿Dónde tiene que leer código fuente? Cada hallazgo aquí es de D11 y D17 y va con prioridad.

Clasificación y parada: **P0** bloquea la casilla, **P1** entra en la etapa, **P2** a la lista. Solo P0 bloquea. Si en una etapa salen más de 3 P0 seguidos del mismo tipo, parar y avisar: eso es fallo de diseño, no de implementación.

## Trabajo en paralelo (worktrees)

Los frentes que no comparten ficheros se construyen a la vez en worktrees contra interfaces congeladas.

- **Definición de terminado**: commit en su rama propia, y el informe **incluye el SHA**. Sin SHA, "puertas en verde" no es verificable, porque el trabajo puede vivir solo en el árbol de trabajo del worktree y no existir en ninguna rama.
- **Las puertas compartidas caducan lo validado antes.** Si cambia el linter de paquetes, un test de arquitectura o un esquema, todo lo que se validó contra la versión anterior deja de estar validado. Antes de la puerta final: `git rebase` sobre `main`, y **la ejecución que cuenta se hace en `main`**, no en el worktree.
- **Un worktree nunca se añade al índice como repo embebido.** Va en `.gitignore` desde que se crea.
- **Un worktree no cambia un puerto por su cuenta.** Escribe la propuesta en `docs/puertos-propuestas.md`, sigue contra el interfaz actual con un `TODO` y no para. Las decisiones de puertos se resuelven en lote.

## Seguridad

`SECURITY.md` define la divulgación coordinada. Secretos jamás en el repo ni en logs. Los tests de fuzzing (parser de corpus, ledger, verificador) son puerta de la etapa 1.
