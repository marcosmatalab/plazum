# plazum

**El GRC de continuidad: no pierdas nunca la conformidad.**

Un solo binario en Go que sabe qué normas te aplican, qué tienes que hacer y para qué fecha exacta, con la cita de cada cosa. Comprueba solo lo comprobable, agenda y reclama lo humano, genera los documentos, escala si nadie atiende, y lo deja todo en un expediente que un auditor puede verificar sin fiarse de ti.

**Cero dependencias externas.** `go.mod` no tiene ni una línea `require`. Se comprueba con un comando, no con una promesa:

```bash
go list -deps -f '{{if not .Standard}}{{.ImportPath}}{{end}}' ./cmd/plazum | grep -v '^plazum/'
```

No imprime nada: todo lo que entra en el binario es biblioteca estándar o código de este repositorio.

Y la consecuencia que importa si tu datacenter está cerrado: **la suite entera compila y pasa sin acceso a red.**

```bash
GOPROXY=off go test ./...
```

No es lo mismo que cero dependencias. Cero dependencias es una propiedad del `go.mod`; esto es que puedes verificar el producto entero **dentro de tu perímetro**, sin abrir una salida a un proxy de módulos ni confiar en que siga estando el día que audites. Es una puerta de CI, no una promesa: la suite completa corre con `GOPROXY=off` en cada empujón.

## Estado: en construcción, por etapas y en público

**Etapa 1 (núcleo probatorio) y etapa 2 (serve, UI y autoservicio) cerradas. Etapa 3 (corpus) abierta.**

Lo medido hoy, no lo prometido: **1.022 casos de test** escritos (1.390 ejecutados, contando subtests) con fuzzing y detector de carreras, **81,3 % de cobertura**, ~32.000 líneas de producción y ~36.000 de test, **9 workflows de CI** en verde. Lo que falta y cuándo, en [`ETAPAS.md`](ETAPAS.md).

**El núcleo determinista**, completo: motor de plazos multi-régimen (días hábiles, calendarios combinables, cierre y traslado, suspensiones y prórrogas, hitos encadenados, límites por categoría, plazos que corren hacia atrás), aplicabilidad Datalog, 8 estados, ledger v1 con Merkle y v2 con AEAD comprometido y borrado legal con lápidas, blobs cifrados content-addressed, historia bitemporal, certificados con sus dorados, perímetros multi-entidad y anclaje RFC 3161 con verificación offline.

**La superficie**, construida en la etapa 2: `plazum serve` con seis pantallas accesibles (axe-core en cero violaciones sobre 16 auditorías), sesiones y CSRF, OIDC y SCIM 2.0 para aprovisionar personas desde el IdP, export a SIEM, actualizador con punto de retorno, `plazum doctor`, ensayo de copias y restauración que corre nueve veces en CI (una sana y **ocho copias rotas**, cada una con el mensaje que tiene que salir), y distribución en matriz Linux/macOS/Windows con imagen Docker reproducible.

**El corpus**: **33 paquetes** con su estrato legal ([`paquetes/CORPUS.md`](paquetes/CORPUS.md)), de los cuales **21 con relojes reales: 218 hitos y 626 casos dorados** que se ejecutan contra el motor en cada ejecución de `./comprobar.sh`. El resto son esqueletos honestos con la transcripción planificada. La medición que decide el orden de autoría (**310 obligaciones con reloj censadas en 31 de los 33 paquetes**, tras el barrido de disyunción del 02-09-2026 y su corrección) está en [`docs/censo-relojes.md`](docs/censo-relojes.md).

<!-- cobertura-v1:inicio -->
**Cuánto del corpus de la v1 está escrito, computado por un test y no a mano.** Los quince paquetes que forman los doce marcos de la v1 están declarados como dato en [`paquetes/marcos-v1.json`](paquetes/marcos-v1.json), con el motivo de cada uno y el de cada exclusión. Sobre ellos: **84,3 %** de los relojes censados están escritos (145 de 172). Ese número **bajó desde un 80,5 % que estaba mal en dos sitios a la vez**, y los dos favorecían la cifra: el numerador sumaba los rituales propios de un paquete referencial (`iso27001` aportaba 6 arriba y 0 abajo, así que escribir relojes que la norma no pide subía la cobertura) y el denominador arrastraba dos censos corregidos hace un día y no propagados. Se dice porque un número que se equivoca hacia arriba es peor que ninguno. El cálculo lo hace una puerta que compara este número con el árbol y se pone roja si se separan, porque un número sin puerta se mueve solo: la lista nació de que dos recuentos del mismo corpus dieran 123 y 124 y la diferencia fuera un paquete de frontera que nadie había escrito en ningún sitio.

**Y 5 de los quince quedan fuera de ese porcentaje**, con su motivo escrito: son referenciales que no se pueden censar sin la norma delante (invariante 3), más el RD 43/2021, al que el censo no le ha dado fila con las tres columnas. No se estiman: un denominador estimado hace que el porcentaje mida la estimación.
<!-- cobertura-v1:fin -->

**Los relojes ahora se ven.** `plazum calendario` saca los próximos doce meses con su artículo, agrupados por mes, con las lecturas divergentes señaladas y la cuenta entera al pie. Y con `--ics` te los llevas al Outlook, al Google Calendar o al Apple Calendar:

```bash
plazum calendario --alcance mis-respuestas.json          # lo que te aplica de verdad
plazum calendario --pais=ES --sector=servicios-digitales --empleados=200
plazum calendario --alcance mis-respuestas.json --ics > obligaciones.ics
```

El segundo es el arranque en diez segundos, sin configurar nada, y **cada fila sale marcada `[supuesto]`**: es lo que le pasaría a una empresa de ese perfil, no una conclusión sobre la tuya. El perfil dice además lo que **no** supone, que es la mitad útil.

**La familia de notificación de incidente está cerrada**: ENS, RD 43/2021 (lo único que vincula hoy en España mientras NIS2 no se transponga), RGPD, CRA, DORA con su Reglamento Delegado, NIS2, eIDAS2, AI Act, MDR y el RDL 19/2018 de servicios de pago. Con los tres casos que un catálogo de controles no sabe expresar: **plazos que se desplazan** (el art. 73.4 del AI Act deja sin efecto al 73.2, no se suma a él), **dos plazos que vinculan a la vez** y manda el que caiga antes (art. 5.1.a del Delegado de DORA), y **obligaciones que obligan sin número**, que se dicen como tales en vez de inventarles una fecha.

## Probar lo que hay hoy

Con Docker, sin instalar Go y en un comando:

```bash
docker build -t plazum .
docker run --rm plazum
```

Eso instala una empresa de ejemplo, deriva sus obligaciones y enseña sus relojes corriendo. El corpus y el expediente de ejemplo viajan dentro de la imagen, así que lo demás también funciona sin montar nada:

```bash
docker run --rm plazum verify expediente-demo.json contexto-demo.json
docker run --rm plazum explain expediente-demo.json
docker run --rm -p 8443:8443 plazum serve --direccion 0.0.0.0:8443
```

La imagen es un binario estático sobre `scratch`, corre sin privilegios y no trae intérprete de órdenes. Dos construcciones del mismo commit dan el mismo binario, y eso se comprueba en CI.

Con Go instalado, y sin clonar nada:

```bash
go install github.com/marcosmatalab/plazum/cmd/plazum@latest
```

O desde el repositorio clonado:

```bash
go build -o plazum ./cmd/plazum
./plazum demo                                                # el mismo ejemplo, sin Docker
./plazum verify expediente-demo.json contexto-demo.json      # recalcula el expediente demo, sin red
./plazum explain expediente-demo.json                        # de dónde sale cada fecha, paso a paso
./plazum cobertura paquetes                                  # la cobertura honesta del corpus instalado
./plazum doctor                                              # por qué no funciona y qué hacer
go test ./...
```

El contexto de verificación lo aporta el receptor, no el expediente. Verificar un expediente con los datos que trae el propio expediente sería comparar al emisor consigo mismo.

## Los tres pilares

1. **Obligaciones con reloj legal de verdad.** Días hábiles, calendarios estatal/autonómico/local combinables, cierre y traslado según el Rgto. 1182/71 y la Ley 39/2015, suspensiones y prórrogas. Y **cuando la doctrina discrepa, se calculan las dos lecturas y se enseña la divergencia con su cita**: el motor no elige en silencio.
2. **Expediente verificable offline.** Cadena de hashes, Merkle RFC 6962, sellado RFC 3161: un tercero lo recalcula entero **sin red y sin confiar en el emisor**. Lo que prueba, y lo que **no** prueba, está escrito en [`docs/modelo-de-amenaza.md`](docs/modelo-de-amenaza.md) con el ataque que puso cada capa ahí.
3. **El corpus es datos, no código.** Cada norma es un paquete con sus obligaciones, relojes, preguntas y plantillas. Añadir la norma 33 no toca una línea de código, y **hay un test que rompe el build si alguien cablea un identificador de norma**.

## Cómo se construye esto

Las reglas están en [`CLAUDE.md`](CLAUDE.md) y no son decorativas: son las que sostienen los tres pilares.

- **Una puerta que nunca se ha visto fallar no es una puerta.** Toda comprobación nace con su fallo demostrado: se rompe a propósito lo que vigila y se pega la salida roja en el commit.
- **Toda comprobación que empareje dos conjuntos lo hace por una identidad firmada, nunca por índice ni por orden.** Nadie firma el orden.
- **En una frontera de confianza, el valor cero de unas opciones tiene que ser el restrictivo**, y todo test de ausencia recorre `nil` **y** vacío-presente: son dos cosas distintas y la peligrosa es la que sale por olvidarse.
- **La IA vive en adaptadores y superficies; el núcleo no la conoce.** La suite entera pasa con la IA desactivada, y eso lo comprueba un paso de CI. La doctrina, en [`docs/ia.md`](docs/ia.md).
- **La frontera legal.** BOE y DOUE se transcriben con su fuente enlazada; ISO, PCI DSS, SOC 2, TISAX y CIS **no**: identificador y título corto como máximo. Un linter rechaza el paquete que se pase, y por eso la IA de este producto no puede inventarse el texto de una cláusula de ISO: no lo tiene.

Lo que se sabe que está mal o a medias no se disimula: está en [`docs/pendientes.md`](docs/pendientes.md), con las familias de fallo que se repiten y por qué.

## Licencia y modelo

Código **AGPL-3.0**, completo (SSO incluido). Los datos del corpus, **Apache-2.0, abiertos e inmediatos para todos**.

De pago es la **vigilancia del contenido**, no el contenido: plazo objetivo de actualización con histórico público, changelog curado con notas de alcance, aviso proactivo de cambio material y sello de cada release. **No se vende garantía jurídica**, y no se llama «respaldado»: se vende que alguien mire el BOE y el DOUE todas las semanas y te avise antes de que te enteres tú. Cualquiera puede generar un corpus libre con una IA, y lo va a poder siempre; lo que no se puede generar es que alguien lo siga vigilando el año que viene.

Soporte: Discussions, sin SLA. Vulnerabilidades: [`SECURITY.md`](SECURITY.md).

**Nada de esto es asesoramiento jurídico.**
