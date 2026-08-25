# dutiq

GRC open source de continuidad de cumplimiento. Motor determinista de obligaciones con reloj legal, corpus normativo como paquetes de datos, expediente verificable offline. Go puro. AGPL-3.0. Una persona lo construye por etapas: el plan vive en `ETAPAS.md` y el detalle completo en `docs/guia.md` (fuente única del plan) y `docs/diseno.md`.

## Comandos

```bash
go build ./...          # compilar todo
go test ./...           # los tests; TODOS en verde siempre
go test . -v            # los tests de raíz: arquitectura (AST), extensibilidad, linter de paquetes
gofmt -l $(git ls-files '*.go')   # vacío siempre; `gofmt -l .` entra en los worktrees de .claude/
go vet ./...            # limpio siempre
```

No hay npm, no hay Makefile, no hay generadores en el producto. El CI sí puede usar herramientas de node (axe-core) sin que eso contradiga lo anterior. Si un comando nuevo hace falta, se documenta aquí.

## Invariantes de arquitectura (vigiladas por tests, no negociables)

1. **`nucleo/` no importa NADA externo ni llama a `time.Now()`.** El instante entra siempre como dato. Lo vigila `arquitectura_test.go` leyendo el AST. Si un cambio lo rompe, el cambio está mal, no el test.
2. **Ninguna norma está cableada en el código.** Los identificadores tipo `ens@`, `iso27001@`, `cis@` en un literal de cadena rompen el build (`extensibilidad_test.go`). Toda norma vive en su paquete de datos bajo `paquetes/` o no vive.
3. **Todo paquete publicado pasa el linter** (`paquetes_test.go` ejecuta `corpus.Cargar("paquetes")`). El linter es la frontera legal: un paquete referencial con más de 120 caracteres de texto normativo NO carga. Jamás pegar texto de ISO, PCI DSS, SOC 2, TISAX o CIS: identificador y título corto como máximo. BOE y DOUE sí se transcriben (art. 13 TRLPI, Decisión 2011/833/UE), siempre con `fuente` enlazada. **Y no se copia contenido normativo de repositorios de terceros de GitHub, aunque digan MIT o Apache: la licencia de un repositorio no alcanza a contenido que el subidor no poseía.** Solo fuente primaria: NIST, EUR-Lex, BOE.
4. **La IA solo entra por el puerto `Asistente`** y su única salida es `puertos.Propuesta` con cita verificada por hash. Los adaptadores de LLM viven fuera de proceso y JAMÁS importan escritura de estado o ledger.
5. **Dependencias**: lista cerrada en `DEPENDENCIAS.md`. Añadir una exige una línea allí con su porqué y su licencia. En `nucleo/`, cero, para siempre.
6. **OSCAL puede ser adaptador de SALIDA con pérdidas, nunca modelo interno ni formato de entrada.** Su modelo (`catalog > group > control > part`) no tiene campo para un plazo: es el mismo agujero que el `RequirementNode` de CISO Assistant. Doblar nuestro modelo para hacer ida y vuelta con OSCAL borra el diferenciador entero, que es el reloj legal. Exportar pierde los plazos y se dice que los pierde; importar nos obligaría a no tenerlos. El porqué completo, con el dato de adopción, en `docs/decisiones.md` D-1.

## Convenciones

- Dominio en español (`Obligacion`, `ventana`, `Calcular`), infraestructura en inglés donde sea idiomático. Comentarios en español, sin tildes en identificadores.
- Todo test de una propiedad de seguridad o legal lleva **control negativo**: se demuestra que el test falla cuando debe (patrón de `TestNingunaNormaCableada`).
- **Una puerta que nunca se ha visto fallar no es una puerta.** Toda puerta nace con su fallo demostrado y anotado en el commit: se rompe a propósito lo que vigila y se pega la salida roja. Sin eso no se sabe si vigila o si acompaña. Vale igual para un test, para un paso de CI y para un linter.
- **Ninguna invocación de `go test` en un workflow.** `go test` sale con código 0 cuando el patrón `-run` no casa con nada y cuando el glob de paquetes no tiene tests: los dos son verdes indistinguibles de un verde de verdad. Las puertas de CI pasan por `.github/puerta.sh`, que cuenta los casos ejecutados y exige un mínimo declarado. Lo vigila `puertas_test.go`.
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
superficies/   serve, api, portal, export (por construir; el CLI está en cmd/dutiq)
cmd/dutiq     verify, explain, estado (construido)
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

   **Elige una propiedad que el código da por buena e intenta tumbarla.** Leer el diff encuentra lo que el autor hizo mal; refutar una propiedad encuentra lo que el autor no pensó, que es donde vive la familia entera de "sin confiar en el emisor". Así salió el ataque 13, que trece rondas de revisión de diff no habían visto. Patrón concreto: cuando una comprobación recorre una lista para contrastarla con otra, preguntar SIEMPRE si la dirección contraria también se recorre. La que falta es la que el emisor usa.

   Dos trampas de la mutación, las dos cometidas ya: una mutación que **no compila** no produce líneas `--- FAIL`, así que un fallo de build parece una mutación no cazada. Comprueba el estado de compilación aparte, y con `go build ./...` entero, no con un grep de `cannot|undefined`: `imported and not used` no contiene ninguna de esas palabras. Y comprueba con `git diff --stat` que la mutación se aplicó de verdad antes de leer el resultado: un `sed` que no casa da verde y parece un hallazgo.

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
