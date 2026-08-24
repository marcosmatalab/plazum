# obligo

GRC open source de continuidad de cumplimiento. Motor determinista de obligaciones con reloj legal, corpus normativo como paquetes de datos, expediente verificable offline. Go puro. AGPL-3.0. Una persona lo construye por etapas: el plan vive en `ETAPAS.md` y el detalle completo en `docs/guia.md` (fuente única del plan) y `docs/diseno.md`.

## Comandos

```bash
go build ./...          # compilar todo
go test ./...           # los tests; TODOS en verde siempre
go test . -v            # los tests de raíz: arquitectura (AST), extensibilidad, linter de paquetes
gofmt -l .              # debe devolver vacío
go vet ./...            # limpio siempre
```

No hay npm, no hay Makefile, no hay generadores en el producto. El CI sí puede usar herramientas de node (axe-core) sin que eso contradiga lo anterior. Si un comando nuevo hace falta, se documenta aquí.

## Invariantes de arquitectura (vigiladas por tests, no negociables)

1. **`nucleo/` no importa NADA externo ni llama a `time.Now()`.** El instante entra siempre como dato. Lo vigila `arquitectura_test.go` leyendo el AST. Si un cambio lo rompe, el cambio está mal, no el test.
2. **Ninguna norma está cableada en el código.** Los identificadores tipo `ens@`, `iso27001@`, `cis@` en un literal de cadena rompen el build (`extensibilidad_test.go`). Toda norma vive en su paquete de datos bajo `paquetes/` o no vive.
3. **Todo paquete publicado pasa el linter** (`paquetes_test.go` ejecuta `corpus.Cargar("paquetes")`). El linter es la frontera legal: un paquete referencial con más de 120 caracteres de texto normativo NO carga. Jamás pegar texto de ISO, PCI DSS, SOC 2, TISAX o CIS: identificador y título corto como máximo. BOE y DOUE sí se transcriben (art. 13 TRLPI, Decisión 2011/833/UE), siempre con `fuente` enlazada.
4. **La IA solo entra por el puerto `Asistente`** y su única salida es `puertos.Propuesta` con cita verificada por hash. Los adaptadores de LLM viven fuera de proceso y JAMÁS importan escritura de estado o ledger.
5. **Dependencias**: lista cerrada en `DEPENDENCIAS.md`. Añadir una exige una línea allí con su porqué y su licencia. En `nucleo/`, cero, para siempre.

## Convenciones

- Dominio en español (`Obligacion`, `ventana`, `Calcular`), infraestructura en inglés donde sea idiomático. Comentarios en español, sin tildes en identificadores.
- Todo test de una propiedad de seguridad o legal lleva **control negativo**: se demuestra que el test falla cuando debe (patrón de `TestNingunaNormaCableada`).
- Todo reloj del corpus lleva **caso dorado** en `pruebas/` del paquete (formato en docs/guia.md Anexo B), derivado del texto legal, no de la implementación. Si motor y caso discrepan, gana el caso.
- Errores accionables: causa, arreglo, y cita si es del dominio. Nada de "error inesperado".
- En documentos para el usuario final: sin guiones largos, comas o dos puntos en su lugar.
- Commits pequeños con el porqué en el cuerpo. Nunca commitear con tests en rojo.

## Estructura

```
nucleo/        ventana, aplicabilidad, estado, ledger (v1+v2 comprometido), blobs,
               historia (bitemporal), certificado, perimetro, expediente, corpus
               (con clase e2e, temporalidad, dorados y su ejecutor) - construido, 0 deps
puertos/       las 9 interfaces hexagonales (compilan, documentadas)
adaptadores/   por construir, etapa a etapa (ver doc.go)
superficies/   serve, api, portal, export (por construir; el CLI está en cmd/obligo)
cmd/obligo     verify, explain, estado (construido)
paquetes/      el corpus: ens (semilla), demo-empresa; cada norma su directorio
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

## Seguridad

`SECURITY.md` define la divulgación coordinada. Secretos jamás en el repo ni en logs. Los tests de fuzzing (parser de corpus, ledger, verificador) son puerta de la etapa 1.
