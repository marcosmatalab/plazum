# Decisiones

> **Para qué sirve este documento.** Las decisiones que cierran una puerta, con el porqué y con el dato que las sostiene. No es un registro de todo lo que se decide: aquí entra lo que alguien va a querer reabrir dentro de seis meses, para que reabrirlo cueste leer un dato en vez de rehacer el razonamiento.
>
> Formato: qué se decide, cuándo, qué lo sostiene, y qué cambia en el repositorio. Si una decisión se revierte, se tacha y se dice por qué, no se borra.

---

## D-1. OSCAL sale del camino crítico

**Fecha:** 25-08-2026.

**Qué se decide:**

1. **No se construye el importador OSCAL.** Deja de ser casilla de etapa.
2. **NIST 800-53 y NIST CSF salen de la autoría de corpus.** Siguen como esqueletos en `paquetes/`, sin trabajo asignado.
3. **OSCAL puede ser adaptador de SALIDA con pérdidas, nunca modelo interno ni formato de entrada.** Es regla de arquitectura y está en `CLAUDE.md`.
4. **El export OSCAL y el Mapping Model se van a etapa 6 o posterior.** Las equivalencias ENS con ISO se hacen en formato propio.
5. **Una sola tubería de ingesta**, el extractor legal desde el ELI del BOE y de EUR-Lex. Reejecutable, porque es también el mecanismo de vigilancia normativa.

### Por qué: el comprador

Mil controles federales estadounidenses no le sirven a un CISO europeo de 20 a 5.000 empleados. NIST 800-53 es el catálogo de un régimen de contratación pública de otro país. Autorizarlo como corpus es trabajo caro que no mueve ni una decisión de compra en el mercado al que se vende esto.

### Por qué: OSCAL vive de obligación, no de adopción

El dato, y es el que zanja la discusión:

- **En 2025 FedRAMP procesó más de 100 autorizaciones Rev5 sin una sola presentación que usara OSCAL.** Ni siquiera los participantes formales del piloto de la Fase 1 de FedRAMP 20x lo usaron para estructurar el material legible por máquina que se les exigía.
- Ahora lo impone por mandato: la RFC-0024 exige paquetes legibles por máquina a partir de **septiembre de 2026**, con un periodo de gracia con notificación pública hasta el **30 de septiembre de 2027**, y **revocación de la certificación FedRAMP** a partir de esa fecha, lo que obliga a una autorización inicial completamente nueva.

Nueve años de formato y la adopción real es cero hasta que aparece la amenaza de revocación. Eso no es un estándar que el mercado quiera, es un estándar que un regulador impone, y encima en un mercado que no es el nuestro.

Fuentes: [RFC-0024, FedRAMP/community](https://github.com/FedRAMP/community/discussions/114), [resumen del mandato y sus fechas](https://quzara.com/fedramp/oscal).

### Por qué: el modelo de OSCAL no tiene dónde poner un plazo

Esta es la razón técnica, y es la que hace que la decisión no sea reversible por conveniencia.

El modelo de OSCAL es `catalog > group > control > part`. **No hay campo para un plazo.** Un control es un enunciado con partes de texto; no tiene fecha límite, ni periodicidad, ni evento disparador, ni régimen de cómputo.

Es exactamente el mismo agujero que tiene el `RequirementNode` de CISO Assistant, y es el agujero donde vive nuestro diferenciador entero: el reloj legal. Un motor de obligaciones con reloj no es un catálogo de controles con fechas pegadas encima, es otro modelo.

**Hacer ida y vuelta con OSCAL obliga a doblar nuestro modelo hasta que quepa en el suyo.** Y un modelo que cabe en OSCAL es un modelo sin plazos, o sea el producto de todos los demás. Por eso salida con pérdidas sí, y entrada no: exportar pierde los plazos y se dice que los pierde; importar nos obligaría a no tenerlos.

### Qué cambia en el repositorio

- `CLAUDE.md`: regla de arquitectura sobre OSCAL.
- `ETAPAS.md`: la casilla de equivalencias en OSCAL Mapping Model se mueve a etapa 6 o posterior; las equivalencias de etapa 3 se hacen en formato propio.
- `paquetes/CORPUS.md`: NIST 800-53 y CSF dejan de tener importador previsto.
- `docs/censo-relojes.md`: NIST 800-53 y CSF no entran en el orden de autoría.

---

## D-2. La capa probatoria queda cerrada

**Fecha:** 25-08-2026.

**Qué se decide:** del ataque 14 en adelante, los hallazgos de la familia "el emisor mete la mano en el expediente" **se documentan en `docs/modelo-de-amenaza.md`, no se arreglan**. Única excepción: que el hallazgo rompa la promesa escrita en ese fichero.

**Por qué:** coste de oportunidad. La capa probatoria está en 9,0 sobre 10 y puntúa 4,0 en decisión de compra. Las dimensiones que deciden la compra (D11 tiempo hasta el valor, D3 corpus, D17 experiencia) suman 20 de peso y están entre 3,0 y 4,5. Seguir puliendo el expediente es seguir puliendo lo que ya gana.

**Lo que va con la decisión:** el modelo de amenaza tenía que existir antes de cerrarla. Cerrar sin escribir qué se defiende y qué no habría dejado la promesa en comentarios de código, que es donde nadie la puede contrastar. Incluye explícitamente el **truncado de cola**, que no es detectable sin un testigo externo publicado y que no se va a montar porque rompe el autoalojado y el offline.

---

## D-3. Los frentes se ordenan por peso de compra, no por afinidad técnica

**Fecha:** 25-08-2026.

**Qué se decide:** delante van tiempo hasta el valor con accesibilidad en CI, censo de relojes, matrix de build con Docker, demo y doctor. Detrás van Litestream, export a SIEM, latido y vendorizar pkcs7.

**Por qué:** las tres dimensiones que deciden la compra suman 20 de peso y están entre 3,0 y 4,5. La capa probatoria está en 9,0 y puntúa 4,0 en decisión de compra. El orden sigue al comprador, no a lo que apetece construir.

---

## D-4. El nombre es PLAZUM

**Fecha:** 26-08-2026.

**Qué se decide:** el producto se llama **plazum**. **Implantado el mismo día**, entero: módulo Go, CLI, marca, documentos, web, dominio de compromiso del ledger (`plazum/commit/v1`) y expediente de demostración regenerado y resellado contra una TSA real. El candado de publicación se queda puesto, pero por otra razón: el nombre ya no bloquea, bloquea que publicar es irreversible y esa decisión es del dueño del proyecto.

**Cómo se llegó:** `vencia` y `preceptum`, los dos propuestos, salieron rojos con la misma forma que costó DUTIQ, una marca ajena ocupando la mayor parte del signo en clases idénticas: AVENCIA al 86% y PRECEPT (de Polestar) al 78%. Se generaron veinte más, cinco salieron limpias en EUIPO y las cinco siguen limpias en OEPM. La criba entera, con los umbrales y los números, en `docs/marca.md`.

**El hallazgo que cambió la elección:** `deontia` era mejor signo que `plazum` (más distintiva, y *lógica deóntica* es literalmente la lógica de la obligación) y salió limpia en los dos registros. Está descartada porque existe **Deontic** (deontic.ai, Lovaina, 2022), plataforma de IA para cumplimiento regulatorio: mismo sector, una letra.

**Lo que eso enseña, y que vale más que el nombre:** el cribador mira registros de marcas y sólo eso. No sabe de empresas en activo que operan sin registrar, y el uso anterior no registrado crea derechos. Es la misma clase de fallo que UTIQ en otro registro: buscar sólo donde es cómodo buscar. La herramienta lo dice ahora en su propia salida y hay un paso manual obligatorio después de cribar. No se automatiza porque no hay fuente gratuita y fiable de razones sociales de la Unión, y una automatización a medias daría el falso verde que la herramienta existe para no dar.

**Lo único abierto**, y es de dictamen, no de criba: `plazo` es descriptiva del servicio en español y el artículo 7.1.c del Reglamento de Marca de la Unión rechaza los signos descriptivos. `Plazum` no es una palabra española y exige un paso mental, que es la zona en la que un signo sugestivo sí se registra. Lo resuelve el agente de la propiedad industrial, no este documento.

---

## D-5. El Cloud sale del camino crítico

**Fecha:** 26-08-2026.

**Qué se decide:** el Cloud gestionado **deja de ser requisito para facturar**. Pasa a "opcional, sólo si los clientes tiran de él". No se construye, no se planifica y no se promete.

**Qué sale del año 1 con esta decisión**, y es lo que hace que valga la pena tomarla: la SL, los DPA con subencargados, el pentest externo, la bóveda multiinquilino y las ocho piezas del runbook de operación. Es el bloque de trabajo más grande que no toca el producto.

**Por qué, en una frase que conviene poder repetir delante de un comprador:** un producto autoalojado cuya tesis es que **el receptor no se fía del emisor** no puede pedirle al comprador que le mande el mapa de sus incumplimientos. La contradicción no es de imagen, es de arquitectura: todo lo que hace valioso al expediente (verificable offline, sin depender del emisor, sin llamar a casa) se debilita en el momento en que el emisor y el proveedor son la misma empresa y los datos viven ahí.

**Y el segundo motivo, que es el que de verdad decide:** una persona sola no puede sostener la postura de seguridad que exige alojar datos de cumplimiento de terceros. No es cuestión de esfuerzo; es que la respuesta a incidentes, la rotación de claves, la disponibilidad y el DPA con subencargados no se hacen a ratos. Prometerlo y no poder cumplirlo es peor que no ofrecerlo.

**Lo que NO se decide aquí:** que no vaya a existir nunca. Si varios clientes lo piden con dinero delante, se replantea con la SL ya constituida y con alguien más. Lo que se decide es que **no está en el camino crítico** y que ningún hito depende de él.

---

## D-6. `plazo`, el motor temporal, sale como módulo aparte con licencia permisiva

**Fecha:** 26-08-2026.

**Qué se decide:** el motor temporal se extrae a un módulo Go independiente llamado `plazo`, con licencia **MIT o Apache-2.0, nunca AGPL**, más su CLI y un playground WASM.

**Por qué la licencia va aparte de la del producto, que es lo importante:** son dos decisiones distintas y mezclarlas mata una de las dos. El producto va AGPL porque queremos que quien lo despliegue como servicio devuelva sus cambios. **Una librería AGPL no la adopta nadie**: entra en un binario ajeno y le impone la licencia entera, así que ningún equipo la mete. Y una librería que nadie adopta no es un embudo de desarrolladores, es un directorio más en el repositorio.

**Frente a lo que ya existe.** `deadlines.bdamokos.org` es una web sin código. `plazo` es librería **y** CLI, hace la capa de **procedimiento administrativo nacional** (art. 30.6 de la Ley 39/2015, con calendarios estatal, autonómico y local combinables) y, cuando la doctrina discrepa sobre cómo se cuenta un plazo, **da las lecturas divergentes con su cita en vez de elegir una en silencio**. Esa última es la diferencia de fondo: una librería de plazos que elige por ti y no te dice que ha elegido es una librería que te hace equivocarte con confianza.

**Lo que se les copia, y se dice:** la exportación a `.ics`. Es buena idea y no hay ningún motivo para no tenerla.

---

## D-7. La notarización nunca se presenta como servicio de confianza

**Fecha:** 26-08-2026.

**Qué se decide:** la línea de notarización se diseña (`docs/notarizacion.md`) y no se construye hasta después de la etapa 3. Y su comunicación queda fijada por escrito **antes** que su técnica, con una redacción exacta y obligatoria:

> sello de tiempo cualificado de un QTSP tercero, más contrafirma y anotación en registro público

**Lo que nunca se dice:** que plazum es un servicio de confianza, que es un prestador cualificado, o que emite sellos cualificados. No lo somos.

**Por qué esto es una decisión y no una nota de estilo:** afirmar la condición de prestador cualificado sin tenerla, **en un producto de cumplimiento**, es exactamente el tipo de afirmación que este producto existe para detectar en otros. El día que un auditor lo mire no se cae la frase, se cae la tesis del producto. Es de los fallos que matan una empresa, y cuesta cero evitarlo si la redacción está decidida antes de que exista la primera diapositiva.

**Y el requisito de diseño que va con ello:** la instancia manda **sólo el hash de la cabeza de la cadena**, 32 bytes. Ni un dato de cliente. No es una opción de privacidad configurable, es la condición sin la cual esta línea no existe, y por el mismo motivo que D-5.

---

## D-8. El agujero de fondo: el producto sirve a quien ya llegó, no a quien empieza

**Fecha:** 26-08-2026.

**Qué se reconoce**, y va antes de cualquier decisión sobre IA porque la explica: **"GRC de continuidad" presupone haber llegado una vez al estado bueno.**

El producto sirve al CISO que **ya tiene su cumplimiento en orden y quiere no perderlo**. No sirve al que empieza desde el caos, **que es el de casi toda empresa de 200 personas**, o sea el comprador que el propio plan declara como objetivo.

**Los primeros treinta días son la otra mitad del producto y hoy no están diseñados.** No es una carencia de una etapa lejana: es la mitad que decide si el comprador llega a usar la otra.

**Dónde se ve el agujero, concreto:** la entrevista de alcance se contesta a mano y sin ayuda, el corpus se carga vacío, y el resultado de contestarla bien es **una pantalla con 130 obligaciones en rojo**. Ese es el muro, y ahí es exactamente donde se abandona un producto de cumplimiento.

**Qué NO se decide aquí:** cómo se cierra. Eso es D-9 y `docs/ia.md`.

---

## D-9. La IA planeada servía a quien ya adoptó; se reordena por punto de fricción

**Fecha:** 26-08-2026.

**Qué se constata.** Ordenadas las siete piezas de IA de la etapa 5 por **a quién sirven**, salen las siete iguales: contradicciones, huecos de evidencia, cuestionarios, búsqueda, MCP, evals. **Todas presuponen corpus cargado, entrevista contestada y evidencia dentro. Ni una ayuda el día que se instala.**

Es la misma forma que D-8 con otra ropa: el plan entero estaba escrito para el usuario que ya había cruzado el muro.

**Qué se decide.** **Mucha más IA, con arnés duro, para implantación y remediación**, y el cumplimiento sigue siendo determinista. La doctrina entera está en `docs/ia.md`; aquí sólo lo que es decisión y no diseño:

1. **El invariante 9 entra en `CLAUDE.md` con sus dos puertas escritas ANTES que el adaptador.** El motivo no es ceremonia: la única forma de que un invariante aguante es que esté puesto antes de que haya presión para saltárselo. La segunda puerta (la suite entera con la IA apagada) es la que convierte *"el núcleo es determinista"* de eslogan en hecho comprobable en dos minutos por cualquiera que clone el repositorio.
2. **Local por defecto, Ollama de serie**, y la nube como *opt-in* con consentimiento anotado en el ledger. Los incumplimientos de un CISO saliendo hacia la API de un tercero es justo lo que ese CISO no va a firmar.
3. **La restricción legal se vende como propiedad, no se disimula.** Sobre estrato referencial (ISO, PCI DSS, SOC 2, TISAX) no hay texto, así que la IA no lo explica y lo dice. La de los competidores se va a inventar el texto de una cláusula de ISO; **la nuestra no puede, porque no lo tiene**. Y no es una promesa sobre el comportamiento del modelo: es consecuencia mecánica de que la cita se verifica por hash antes de enseñar la propuesta.
4. **Nada de pestaña de chat.** La IA va en línea, en el punto de fricción, con su cita visible y dos botones. Si hay que abrir un sitio aparte para usarla, está mal puesta.
5. **Y una que es de nuestro coste, no de la experiencia del cliente, y decide si el modelo de negocio aguanta**: las **notas de alcance de la vigilancia normativa**. El §11 vende "changelog curado con notas de alcance", y hoy el coste marginal de producirlas son fines de semana. Con la IA redactando el borrador desde el diff del BOE y una persona verificándolo, el nivel Respaldado pasa de compromiso caro a **sostenible por una persona**. Es lo que decide si esto escala a 30 clientes.

**Calendario:** se especifica ahora, **se construye después de la Familia A** por dependencia dura (no se puede mapear documentos contra obligaciones que no están escritas), y cuando entre **absorbe la etapa 5**: los agentes de análisis bajan por debajo de las doce piezas de adopción y operación, porque sirven a quien ya adoptó y éstas consiguen que adopte.

---

## D-10. El motor de riesgos es neutral y los catálogos son paquetes por país o metodología, nunca por framework

**Fecha:** 27-08-2026.

**Qué se decide.** El módulo de riesgos se parte en dos piezas con una frontera dura, y la frontera es la decisión entera:

- **El motor de riesgos es neutral.** No conoce ninguna metodología ni ningún catálogo. Activo, amenaza, vulnerabilidad, salvaguarda, probabilidad, impacto y riesgo residual son suyos; qué amenazas existen y cómo se llaman, no.
- **Los catálogos son paquetes de datos**, con la misma frontera legal, el mismo linter y el mismo estrato que el resto del corpus, y se organizan **por país o por metodología**. Jamás por framework.

**Por qué no por framework, que es lo que hace todo el mercado.** Porque los marcos **exigen el análisis y no traen catálogo**. El ENS lo exige (art. 3.2 y art. 6 del RD 311/2022), la ISO 27001 lo exige (cláusula 6.1.2), la ISO 42001 lo exige para IA, NIS2 y DORA lo exigen. Ninguno dice qué amenazas hay. Un catálogo "de ISO 27001" y otro "del ENS" serían **el mismo contenido duplicado con dos etiquetas**, y el cliente que tiene los dos marcos haría el análisis dos veces para satisfacer una obligación que es una sola.

**Y ahí está el diferenciador, que es de producto y no de arquitectura:** con el catálogo separado del marco, **un solo análisis de riesgos satisface la obligación de riesgos de todos los marcos del cliente a la vez**, y el expediente lo demuestra apuntando la misma evidencia desde cada obligación. Eso es *cross-framework* de verdad, y no lo que el mercado llama así, que es un mapeo de controles entre catálogos. Un mapeo ahorra lectura; esto ahorra **el trabajo**.

**MAGERIT es el primero por dos razones, y ninguna es técnica.** Una, es el único catálogo **redistribuible verificado** que tenemos hoy. Dos, cabeza de playa: es el vocabulario que el comprador público español ya habla. **No es una dependencia del ENS ni sale del ENS**: si mañana MAGERIT desapareciera, el ENS seguiría exigiendo el análisis exactamente igual y el motor seguiría funcionando con otro catálogo. Escribirlo como si el ENS lo trajera sería precisamente el acoplamiento que esta decisión prohíbe.

**ENISA es el paneuropeo**, y es el que despega la cabeza de playa: en cuanto el catálogo deja de ser español, el módulo vale para el cliente que nunca va a oír hablar de MAGERIT.

**Apuntados como candidatos de año 2, y apuntados es todo lo que están:** **EBIOS RM** (ANSSI) y las **amenazas elementales del IT-Grundschutz** (BSI). Los dos con la **licencia por verificar contra fuente primaria antes de comprometer nada**, exactamente como se hizo con CIS y con SCF. No se anuncian, no se planifican y no entran en ninguna lista de "lo que traerá el producto" hasta que la verificación exista y esté fechada. Vale aquí el invariante 10 entero: que un catálogo se publique en abierto no dice qué se puede redistribuir, y la licencia de un repositorio no alcanza a contenido que el subidor no poseía.

---

## D-11. El corpus de derecho de la UE es multiidioma por transcripción, nunca por traducción

**Fecha:** 27-08-2026.

**Qué se decide.**

- **Derecho de la UE: multiidioma por transcripción de las versiones oficiales.** Cada versión lingüística publicada en el Diario Oficial es **auténtica**, no es una traducción de otra. Se descargan de **Cellar** las que hagan falta y cada una entra en el corpus como lo que es: texto oficial, con su ELI, su huella y su cita.
- **Traducción automática: nunca.** Ni con revisión, ni "solo para la ayuda", ni marcada como provisional. Un plazo mal traducido es una fecha mal calculada, y el producto entero se vende sobre esa fecha.
- **Derecho nacional: solo existe en su idioma.** El RD 311/2022 está en español y en el corpus está en español. No hay versión inglesa del ENS porque no la hay en el BOE. La interfaz se traduce; el texto normativo no.

**Por qué.** Porque es la única forma de que el multiidioma no cueste credibilidad. Un competidor que traduce tiene que poner un descargo diciendo que el texto no es fiable; nosotros no ponemos descargo porque **cada versión la publicó el legislador**. Es la misma mecánica que la puerta antialucinación del invariante 9: no es una promesa sobre la calidad de un proceso, es que la vía por la que entraría el error no existe.

**Lo que va con la decisión:** la interfaz ya separa texto de catálogo (las claves `ui.*` del calendario y de las pantallas), así que traducir el producto no toca el corpus. Y un paquete nacional que solo existe en un idioma **no es un paquete incompleto**: se dice en su ficha y se acabó, igual que se dice de un estrato referencial que no trae texto.

---

## D-12. La cadencia sin número: plazum propone, la norma pone el suelo, el cliente aprieta

**Fecha:** 28-08-2026.

**La pregunta que estaba abierta**, escrita en la casilla de la Familia B de `ETAPAS.md`: 38 de los 61 puntos del Reglamento de Ejecución (UE) 2024/2690 mandan revisar *"a intervalos planificados"* o *"periódicamente"* **sin dar ningún número**. ¿Vale ahí el patrón de `iso27001` (un ritual de plazum con su intervalo justificado), o hay que decir *"sin plazo legal"* y limitarse a medir el tiempo transcurrido?

**Qué se decide.** Vale el patrón, **con tres piezas obligatorias y una distinción que hasta hoy no estaba escrita**:

1. **plazum propone el intervalo y lo justifica POR ESCRITO**, en la cita de la obligación. Un número sin argumento es un número inventado, y el proyecto entero se sostiene sobre que eso no pasa.
2. **La obligación dice de quién es el número.** Un punto transcrito que trae número propio se cita como lo que es (`anexo, punto 1.1.2`, y el número es de la norma); un intervalo puesto por plazum se cita como ritual (`ritual plazum sobre <punto>`, y el número es nuestro). **Las dos cosas no se mezclan nunca en la misma obligación**, porque el cliente tiene derecho a saber cuál de las dos fechas le puede discutir un inspector.
3. **El suelo manda, y es lo que decide qué puede tocar el cliente:**

| lo que dice la norma | quién pone el número | qué puede hacer el cliente |
|---|---|---|
| *"al menos una vez al año"*, *"como mínimo anualmente"* | **la norma**: es un **suelo legal**, el intervalo máximo permitido | **apretarlo, nunca aflojarlo.** Revisar cada seis meses cumple; cada dieciocho, no |
| *"a intervalos planificados"*, *"periódicamente"*, sin número | **plazum**, y lo dice | moverlo en las dos direcciones: es un defecto, no un límite |

**Por qué el suelo importa tanto como el número.** Porque son la misma frase para un lector distraído y obligaciones opuestas para un inspector. Los tres puntos con número de 2024/2690 (1.1.2, 2.1.4 y 10.1.3) **tienen suelo**: `P12M` ahí no es una propuesta de plazum, es el máximo que la norma tolera, y un producto que dejara aflojarlo estaría ayudando a incumplir. Los otros 38 no tienen suelo y su número es nuestro.

**Lo que va con la decisión:** el paquete tiene que poder decir la diferencia, no sólo la cita. Hoy se distingue leyendo el `articulo` (`anexo, punto N` contra `ritual plazum sobre N`), que funciona pero es convención y no dato. **Pendiente P1**: un campo que lo diga, y con él una guarda que impida aflojar un intervalo con suelo legal. Hasta que exista, la convención se respeta y se dice aquí.

---

## D-13. Lo que el calendario descarta se cuenta; nunca se enumera y nunca se calla

**Fecha:** 28-08-2026.

**El porqué está en un bug, y conviene que se lea con el bug delante.** `Derivar12Meses` tenía `if !vigente { continue }`: un `continue` mudo, en una función cuya propia cabecera promete que *"lo que no produce fecha NO desaparece: sale en `SinFecha` con el motivo"*. Con él se iba entera del calendario cualquier obligación que empieza a obligar **dentro de la ventana que estás mirando** — las dos notificaciones del art. 14 del CRA, quince días antes de aplicarse, en el perfil escrito ese mismo día para enseñarlas. **Ningún test lo vio, y no por descuido: todos preguntaban por lo que sale y ninguno por lo que se cae.**

**Qué se decide, y vale para toda derivación de cara al usuario:**

> En una derivación que el usuario ve, un elemento sólo desaparece si **desaparecer es la respuesta**, y entonces **se cuenta**.

Ni enumerar ni callar. Las dos alternativas son malas y por razones distintas: enumerar las trescientas obligaciones que no te alcanzan no informa, entierra; callarlas deja al operador sin saber si el producto ha mirado el corpus entero o sólo un trozo, que es exactamente la duda que hace que no se fíe. **Un contador, y una puerta para verlos si quiere** (`--todos-los-relojes`, que ya existía).

**Los dos casos que quedaban abiertos del barrido, resueltos así:**

| caso | qué se hace |
|---|---|
| **la derogada** | Cubo propio, `Cese`, **espejo exacto del `Estreno`**: *"deja de obligarte dentro de esta ventana"*. `corpus.VigentesEn` ya documentaba en su cabecera que hay que decirlo; ahora se dice. La que cesó **antes** de la ventana no se pinta (no es una transición de estos doce meses) y **se cuenta**, que es lo que la distingue de un descarte mudo |
| **lo no alcanzado por la aplicabilidad** | Una línea en la cuenta: *"N instalados que NO te alcanzan según tus respuestas"*, con la puerta al lado |

**Y el `Cese` es doctrina, no una fila más.** Es la única sección del calendario que **quita** trabajo en vez de ponerlo. Un producto de cumplimiento en el que las obligaciones sólo crecen enseña al operador que la herramienta nunca le libera de nada; decirle *"esto deja de obligarte el 15 de marzo, puedes parar"* es la mitad del trabajo que nadie hace, y la que se gana la confianza. Como el estreno, **no es una `Fecha`**: ese día no hay nada que entregar.

**Lo que hace que esto no se pudra:** la contabilidad quedó **cerrada y se comprueba sumando**. Cada hito instalado cae en exactamente un cubo de la partición por tiempo (en vigor, estrena, ya cesó, empieza después, vigencia ilegible), y lo que está en vigor cae en exactamente uno de la partición por alcance. Un test lo suma. **Es la única forma de test que crece sola**: el día que alguien añada una rama a la derivación y se olvide de contarla, la suma se rompe sin que nadie tenga que acordarse de escribir el caso. Es lo que faltaba cuando el `continue` mudo pasó trece revisiones.

---

## D-14. Un mapeo no es un ámbito: el caso del art. 1 del Reglamento de Ejecución (UE) 2024/2690

**Fecha:** 28-08-2026.

**El caso, y es ejemplar porque el error que evita lo comete todo el mercado.** El Reglamento de Ejecución (UE) 2024/2690 desarrolla los requisitos técnicos del art. 21.2 de la Directiva NIS2. Un catálogo de controles lo etiqueta *"NIS2"* y se lo aplica a cualquier entidad NIS2, porque en una hoja de cálculo el marco es una columna.

**No es así.** Su art. 1 abre con una lista **cerrada de once tipos** a los que llama *entidades pertinentes*: proveedores de servicios de DNS, registros de nombres de dominio de primer nivel, proveedores de servicios de computación en nube, de servicios de centros de datos, de redes de distribución de contenidos, de servicios gestionados, de servicios de seguridad gestionados, de mercados en línea, de motores de búsqueda en línea y de plataformas de servicios de redes sociales, y prestadores de servicios de confianza.

**Un hospital es entidad esencial de NIS2 por el anexo I de la Directiva y no es ninguno de los once.** Los requisitos técnicos del anexo de 2024/2690 **no le alcanzan**. Enseñárselos no es un matiz: son 57 relojes y un anexo de 153 puntos de trabajo que no le tocan.

**Qué se decide.** Que esto **no es un detalle de la transcripción de un paquete, sino la forma de trabajar con todo marco derivado**:

1. **El ámbito de un acto de ejecución o delegado se lee en SU artículo de ámbito, nunca se hereda del acto base.** Un reglamento de ejecución puede alcanzar a menos que su directiva, y normalmente alcanza a menos.
2. **Toda regla de aplicabilidad se prueba en las DOS direcciones** (ya está en `CLAUDE.md`), y la dirección que hay que escribir con más cuidado es la negativa, con el artículo de la exclusión al lado.
3. **La dirección negativa lleva su propio control de que no se cumple sola.** El test comprueba además que el sujeto excluido **derive alguna otra obligación**: sin eso, un motor que no derivara nada pasaría la comprobación de exclusión y no habría comprobado nada. Es la trampa del test de ausencia, la misma familia que las dos formas de la nada del invariante 8.

**Por qué está en decisiones y no sólo en el código.** Porque es la diferencia entre **transcribir y entender**, y es demostrable delante de un comprador: cualquiera puede abrir el art. 1, contar once tipos, buscar "hospital" y no encontrarlo. Un competidor con corpus en hoja de cálculo no puede enseñar esa comprobación porque su modelo no tiene dónde ponerla: en una columna llamada *"NIS2"* no cabe la frase *"salvo que no seas ninguno de estos once"*.

---

## D-15. Las 62 ceremonias: el número se enseña siempre, y se enseña el colapso a continuación

**Fecha:** 28-08-2026.

**El dato que lo abre.** Los intervalos propuestos para las 34 cadencias sin número del anexo de 2024/2690 suman **unas 62 citas fechadas al año, y sólo de ese marco**: 7 a P3M, 9 a P6M, 14 a P12M y 4 a P24M. Antes de sumar el ENS, la ISO que el cliente tenga, o lo suyo propio. Es **más de una ceremonia de cumplimiento por semana** para el CISO de 200 empleados de la tercera pasada, el que abre esto a las nueve de la mañana sin documentación.

**Un calendario que nadie puede cumplir no es un calendario, es un reproche semanal**, y el producto que lo genera se cierra al segundo mes.

**Qué se decide, y son tres cosas en este orden:**

**1. El número se enseña, siempre.** Es de la norma, no nuestro, y **sumarlo es exactamente lo que nadie más hace**: un catálogo de controles no sabe cuántas veces al año hay que hacer nada, porque no tiene reloj. Esconder el total para que la pantalla no asuste sería maquillar, y sería además romper la promesa que sostiene el resto (la contabilidad honesta del calendario, que ya enseña lo que no puede derivar).

**2. E inmediatamente se enseña el colapso.** La composición cross-framework existe en el diseño **exactamente para esto**, y hasta ahora se había argumentado como ahorro de trabajo de implantación; su primer uso real es aquí. **Una revisión por la dirección puede satisfacer a la vez** la cláusula 9.3 de ISO/IEC 27001, el ritual equivalente del AIMS y el punto 1.1.2 del anexo de 2024/2690, **si el entregable encaja**. Sesenta y dos ceremonias en el papel son muchas menos reuniones reales, y decir sólo la primera mitad es tan deshonesto como esconderla.

**3. El primer paso es barato y va en `calendario`:** agrupar por **ritual y entregable compartido**, con la línea *"esta ceremonia cubre M obligaciones de N marcos"* y la lista debajo. **No necesita el álgebra de composición**, que sigue en su etapa: necesita datos que ya existen en el corpus (el `entregable` de cada obligación y su hito). Cuando Familia B esté escrita, esa agrupación es **la diferencia entre un calendario que asusta y uno que ordena**.

**Es la primera pieza de D16 que se adelanta, y esta cuenta de 62 es su caso de negocio.** No se adelanta por elegancia: se adelanta porque sin ella el corpus completo hace el producto peor en vez de mejor, que es el único motivo válido para mover algo de etapa.

### Hecho el 29-08-2026: el punto 3, con sus números

La agrupación está en `plazum calendario`, delante del listado por meses, y en `nucleo/pantalla/ciclos.go`. **Se derivó, no se escribió a mano.** Con el perfil de servicios digitales y el reglamento técnico de NIS2 encendido:

```
LAS SENTADAS: 52 obligaciones periodicas de 2 marcos en 5 sentadas al ano

  ciclo anual (P12M): 32 obligaciones de 2 marcos
      4 con fecha, en 4 sentadas
      28 esperando un dato tuyo (la ultima vez que lo hiciste)
      se pueden juntar: 32 de las 32 se pueden adelantar.
```

**Tres cosas de la implementación merecen quedar escritas, porque no eran obvias:**

**El consejo de agrupar necesitaba un dato que ya existía por otra razón.** Juntar dos fechas significa **adelantar** una, y adelantar no siempre se puede. `origen_del_intervalo` se escribió el 28-08 para que el cliente supiera de quién es cada número; al día siguiente resultó ser lo que hace que este consejo no sea irresponsable: con `suelo_legal` apretar siempre cumple, con `propuesto` el número es nuestro, y con `fijado` **no se toca**. Sin esa distinción, *"junta estas doce en una sesión"* sería proponerle al cliente que incumpla una de ellas.

**Un ciclo existe aunque su primera fecha no se pueda calcular.** La primera versión agrupaba sólo lo que tenía fecha, y el día uno de un cliente **no hay ninguna**: todas las cadencias esperan la fecha de la última vez que se hizo. Y ése es justamente el día en que más falta hace saber cuántas veces al año habrá que sentarse. Se cuentan las dos cosas por separado y se dicen las dos.

**La cifra de 62 era del papel; la de la pantalla es otra y más útil.** No porque haya menos trabajo, sino porque *ceremonia* y *obligación* nunca fueron la misma unidad: una trimestral produce cuatro citas al año y sigue siendo una obligación. El cálculo cuenta **deberes distintos**, no vencimientos, y ahí está la mitad del colapso.

**Lo que va con la decisión:** el criterio de acceso al trimestre, que hoy no está escrito y por eso cada autor pone P3M a lo que le parece urgente. **P3M se reserva a controles cuya evidencia la produce una máquina** (un escáner, la consola de agentes, el inventario, el sistema de tiques), no a los que exigen que una persona se siente a revisar. Sin ese criterio, la suma crece sola y ninguna agrupación la salva.

## D-16. Una disyuntiva de disparadores no multiplica deberes

**Fecha:** 29-08-2026.

**El texto que lo abre.** Veintidós de los cuarenta y siete puntos del anexo del Reglamento de Ejecución (UE) 2024/2690 dicen la misma frase: *«revisarán y, cuando proceda, actualizarán X **a intervalos planificados o cuando se produzcan incidentes significativos o cambios significativos** en las operaciones o los riesgos»*.

**La decisión: eso es un segundo DISPARADOR del mismo deber, no un segundo deber.** Va como campo `reabre_por` de la obligación periódica, no como obligación aparte.

**Por qué importa tanto como parece poco.** Escribirlos separados habría dado **69 obligaciones donde hay 47**, y le habría dicho al cliente que tiene el doble de ceremonias de las que tiene — el día siguiente de publicar la sección que existe precisamente para no empeorar eso (D-15). El error habría sido invisible: cada obligación, por separado, estaría bien escrita y bien citada. Sólo la suma mentiría.

**Y hay una prueba de que la lectura es la correcta**, no una comodidad: **cerrar el deber una vez lo cierra por los dos caminos**. Quien revisa la política tras un incidente significativo ha cumplido también la revisión periódica, y el ciclo se reinicia desde esa fecha. Dos deberes distintos no se cierran con un solo acto; dos disparadores del mismo deber, sí.

**Lo que va con la decisión, y son tres cosas:**

**1. La reapertura no inventa plazo.** La norma dice *cuándo* hay que revisar (al ocurrir el hecho) y **no da plazo para hacerlo**. Así que el reloj reabierto sale como `sin plazo legal`, con la derivación entera al lado (qué hecho, de qué fecha, y cómo se cierra), y el motor mide el tiempo transcurrido. Es el mismo trato que ya recibían las tres obligaciones sin número del corpus, y por la misma razón: **el corpus no se inventa números que el texto no da**, ni siquiera cuando la pantalla quedaría más ordenada con uno.

**2. El empate no reabre.** Si la revisión consta el mismo día del hecho, se hizo *después* de él — por eso consta. Tratar el empate como reapertura pediría repetirla para siempre: cada vez que se registrara la nueva revisión, el hecho volvería a empatar con ella. No daría error, daría **una obligación imposible de cerrar**, y eso es peor que una sin plazo. Una obligación sin plazo se puede cumplir; una que no se puede cerrar sólo se puede abandonar, y con ella el resto de la pantalla.

**3. Los hechos se derivan del texto de cada punto, nunca de una lista escrita al lado.** Por eso salen desiguales y correctos: el 6.1.3 reabre sólo por incidente (su texto no menciona los cambios) y el 10.4.2 por cambio significativo y por **cambio jurídico**, que es lo que de verdad mueve un procedimiento disciplinario. Una lista escrita a mano habría puesto los mismos dos hechos en los veintidós.

**Es doctrina de corpus, no del anexo de 2024/2690.** La disyuntiva *«periódicamente o cuando ocurra X»* está en el ENS, en DORA, en el RGPD y en la ISO; va a reaparecer en cada marco con un «o cuando», y la respuesta ya está decidida.

## D-17. Un deber permanente no vence, y decirlo es una respuesta

**Fecha:** 29-08-2026.

**El texto que lo abre.** El artículo 72.2 del Reglamento (UE) 2024/1689 obliga al proveedor de un sistema de IA de alto riesgo a recopilar, documentar y analizar los datos de funcionamiento *«de manera activa y **sistemática**»* durante toda su vida útil. No dice «periódicamente», no da cadencia y no da plazo.

**Había tres salidas y dos eran malas.** Ponerle un trimestre habría sido inventar un número que el texto no da — lo que este corpus lleva un año evitando. Dejarlo fuera habría sido callar un deber que existe. La tercera es decir lo que es: **un deber permanente, sin fecha, con su motivo**, igual que las tres obligaciones sin número que el corpus ya traía.

**Y había una cuarta salida peor, que es la que estaba.** La primitiva `continua` llevaba declarada en el formato desde el primer día y el motor **no la sabía ejecutar**: salía como *«esta primitiva todavía no tiene ejecutor»*. Eso es una fila sobre **plazum**, no sobre la norma. Quien la lee entiende que al producto le falta algo, cuando lo que pasa es que a la obligación no le falta nada.

**La diferencia entre las dos frases es la decisión entera:**

| lo que salía | lo que sale |
|---|---|
| *esta primitiva todavía no tiene ejecutor* | *obliga y la norma no da número* |
| una queja sobre el producto | una respuesta sobre la norma |

**Lo que va con la decisión: `continua` no puede ser la salida barata.** Marcar `continua` una obligación que sí tiene cadencia libraría de escribir los tres casos dorados. Dos guardas, y la segunda es la que muerde:

1. tiene que traer **al menos un** dorado — el que afirma que no vence. Sin él, nadie ha comprobado nunca que el motor diga eso.
2. **su propio texto legal no puede decir que es periódica.** Si el boletín dice «periódicamente» y el paquete dice `continua`, uno de los dos miente, y no es el boletín.

La exención de los tres dorados es estrecha a propósito: si la `continua` declara `en` (una fecha de fin), sí produce fecha y vuelven a exigirse los tres.

## D-18. El reloj por objeto: el paquete nombra la forma del hecho, el objeto aporta la instancia

**Fecha:** 29-08-2026. **Medido antes de escribir una línea de la etapa 4**, en `nucleo/corpus/por_objeto_test.go`.

**El problema, con sus dos consumidores.** Dos relojes del corpus piden la misma forma y ninguno existe todavía:

| dónde | qué pide |
|---|---|
| art. 81.12 de MiCA | revisar la evaluación de idoneidad *«para cada cliente, al menos cada dos años después de la evaluación inicial»* |
| art. 33 del RGPD y las otras dieciocho notificatorias de incidente | las 72 horas corren **por incidente**, desde el primer conocimiento de **ese** incidente |

Son el mismo reloj: **N instancias de UNA obligación para UN sujeto**, cada una anclada en su propio hecho.

**Por qué se midió antes en vez de razonarlo.** La medición equivalente de la familia A pidió los hitos encadenados y el `Tope`, y descartó `Secuencia` por llevar el arranque cableado en la estructura. Escribir treinta y tres relojes contra una primitiva que no hace lo que hace falta es escribirlos dos veces.

**Lo medido, y las cuatro mediciones están en verde como test:**

| eslabón | ¿admite objeto? | qué se midió |
|---|---|---|
| `historia` | **sí** | su clave es una cadena que elige quien llama: dos incidentes con dos `PrimerConocimiento` distintos, sin tocar nada |
| primitivas de `ventana` | **sí** | `Plazo` toma el ancla en cada llamada: tres incidentes, tres fechas correctas |
| **el hecho** | **no** | el disparador es UNA cadena que escribe el paquete, y un paquete no conoce los objetos del cliente |
| **el hito** | **no** | uno por obligación, y hay **dos guardas** independientes escritas para que no se repita |

**Las dos formas en que se rompe el hecho, y la segunda es la cara.** Si el alcance trae las fechas por objeto, el disparador del paquete no casa con ninguna y el motor contesta *«el reloj no ha arrancado»* con tres incidentes abiertos. Si el alcance usa la clave que el paquete nombra, **el segundo incidente pisa al primero**: `ventana.Hechos` es un mapa, una fecha por nombre. Medido: dos incidentes, un vencimiento, y el que desaparece vencía **nueve días antes**. Es el descarte silencioso en su versión más cara, porque lo que se pierde no es una fila de calendario, es una notificación a la autoridad.

**La decisión.** No hace falta una primitiva nueva. Falta **una sola cosa, la identidad del objeto**, y se parte en dos mitades que ya no se pueden confundir:

- **el nombre lo escribe el PAQUETE**, y es la forma del hecho (`conocimiento_del_incidente`);
- **la instancia la aporta el ALCANCE**, y es el objeto (`incidente/2026-014`).

**Y viaja la cadena entera, no media.** Cada eslabón empareja por identidad — hecho → vencimiento → fila de dorado → destino de la ley de conservación — y hoy ninguno la lleva. Meter el objeto en la mitad de la cadena produce lo de siempre: **una junta sin vigilar**, que es exactamente donde se rompió la ley de conservación la semana pasada.

**Lo que la decisión prohíbe, con puerta.** El atajo que va a tentar a alguien es meter el identificador del objeto **dentro del nombre del hecho** (`constancia/2026-014`). Eso obliga al paquete a conocer los objetos del cliente, que es lo único que un paquete de corpus no puede hacer. `TestNingunPaqueteNombraUnObjetoDentroDeUnHecho` recorre los 106 relojes del corpus y lo rechaza; **M39** (ponérselo al disparador del art. 33 del RGPD) lo pone rojo con el nombre de la obligación.

---

## D-19. La v1 sale con 12 marcos, y sale como plataforma

**Fecha:** 01-09-2026. **Decisión de Marcos**, no consulta. Recorta **anchura de corpus**, no profundidad de nada: la vara sigue siendo más de 9,7.

### Los 12 del escaparate

ISO 27001, RGPD + LOPDGDD, ENS, NIS2 (`nis2-ue` + `nis2-tecnica` + `nis1-es`), DORA, AI Act, ISO 42001, CRA, Ley 2/2023, SOC 2, PCI DSS y TISAX. Doce marcos repartidos en **quince directorios de paquete**.

**El criterio**: lo que se pide hoy y lo que van a pedir, con **la gobernanza de IA entera dentro** (AI Act + ISO 42001) por exigencia explícita. TISAX entra **por demanda** — automoción, y se pide ya en España — aunque hoy sea un esqueleto.

**Es escaparate, no borrado, y esto importa más que la lista.** Lo construido de MiCA, eIDAS2, MDR, PSD2-ES y ENI **se queda en el corpus** como extra, y el resto del censo pasa a **autoría continua post-v1**. Nada se tira. Un marco fuera del escaparate sigue cargando, sigue pasando el linter y sigue teniendo sus dorados en verde: lo único que cambia es que no se promete en la portada.

### Los números, medidos hoy y no copiados

Contados desde los JSON el 01-09-2026, no traídos de un informe (invariante 10 aplicado a los datos propios: un dato que llega de un informe es una pista):

| | obligaciones escritas | relojes escritos | ficheros de dorados |
|---|---|---|---|
| dentro de los 12 | 386 | **90** | 90 |
| fuera de los 12 | 21 | 16 | 15 |
| **corpus entero** | **407** | **106** | **105** |

**Los 90 relojes escritos del corpus están TODOS dentro de los 12.** De los 16 de fuera, tres son de `demo-empresa`, que no es una norma; los **trece** normativos son exactamente los que el escaparate conserva como extra: `mica` 6, `eidas2` 3, `psd2-es` 2, `mdr` 1, `eni` 1. Dicho de otro modo: **el recorte no deja fuera ni un reloj ya escrito.**

### La corrección al censo que sale de esto

El censo daba **57** a `nis2-tecnica` y el paquete trae **48/48**. La diferencia de nueve **no es autoría pendiente ni consolidación**: es que el censo contó de menos y luego contó dos veces.

Medido dos veces contra la instantánea ingerida de Cellar (CELEX `32024R2690`, `corpus-vigilancia/ue-32024r2690`), troceando el anexo por puntos numerados:

- El censo buscó el adverbio exacto *«periódicamente»* (7 puntos) y se le escaparon **las variantes morfológicas de la misma cadencia**: *de forma periódica*, *de manera periódica*, *revisiones periódicas*, *pruebas periódicas*, *formación periódica*, *regularmente*. Contando puntos del anexo con **cualquier** forma de cadencia salen **48**, que es exactamente lo que el paquete tiene. **Cobertura 48/48, cero puntos con cadencia fuera del paquete.**
- Los **21** puntos que llevan la fórmula de disparador (*«así como cuando se produzcan incidentes significativos o cambios significativos»*) **son los mismos** que ya llevan cadencia: **cero puntos sólo-evento**. Así que sumar 37 cadencias + 19 eventos contaba **dos veces el mismo apartado**, y la unidad que el censo declara es la obligación, no el reloj.

> **`nis2-tecnica` son 48 obligaciones, no 57, y están las 48 escritas.** Lo que falta son los ~20 relojes de EVENTO, que son **segundos relojes sobre puntos ya escritos**, no puntos nuevos.

**Consecuencia en el total del escaparate: de ~195 obligaciones legales a ~186.** Y la primera medición de la primera pasada de este mismo recuento dio *«3 puntos con cadencia que faltan en el paquete»*, y era falsa: mi patrón amplio casaba *«indicará»* dentro de *«periódica»*. Segunda medición por otro camino: cero. Es la regla de *un hallazgo grande se mide dos veces* cobrándose otra pieza.

### Qué es la v1, o sea qué bloquea la salida y nada más

**La plataforma guiada de punta a punta sobre `plazum serve`**: alta guiada por las preguntas de aplicabilidad, calendario, derivación, acta 9.3, UAR y escalado, todo dentro del mismo camino. Español e inglés. Corpus versionado con su test N-1. Licencia Ed25519 y entrega firmada. OIDC, SCIM y export SIEM operativos.

**E6 (conectores) y E7 (riesgos) pasan a post-v1.** Siguen dentro del 9,7 verificado de enero-marzo; dejan de bloquear la salida.

### Las puertas nuevas de D11, porque «intuitivo» sin puerta es un eslogan

Cinco, y todas medibles:

1. **Cero formaciones.** Si una pantalla necesita explicación externa para llegar al valor, es hallazgo de la pasada 3 con prioridad. El producto se explica solo, paso a paso.
2. **Todo estado vacío trae su siguiente paso, con test.** Una pantalla vacía sin verbo es un callejón.
3. **Cada número clicable hasta su derivación.** La pantalla de derivación ya existe; la puerta es que **ninguna cifra quede huérfana de enlace**.
4. **El camino guiado es determinista.** La IA propone detrás, jamás delante (invariante 9).
5. **TTFV medido por debajo de 15 minutos.**

Y **las familias de guardas del núcleo alcanzan a las pantallas igual que al núcleo**: valor cero restrictivo (invariante 8), descartes con su hueco explicado (D-13), y el descargo *«esto NO dice que se haya incumplido: dice que no consta»* en **toda** pantalla que enseñe pasado, **con su control positivo** — porque una rama de descargo que ninguna entrada recorre es una rama que no existe (M47).

### Inglés, por la doctrina de `docs/traducir.md`, sin excepciones

El catálogo de cadenas de **interfaz** sale en ES y EN. El **derecho de la UE** se sirve en inglés **transcrito de la versión oficial inglesa vía Cellar**, como paquete o variante con su propia fuente, **jamás traducido por nosotros** (D-11). El **derecho nacional** (ENS, LOPDGDD, Ley 2/2023, RD 43/2021) **queda en español**, y la interfaz en inglés **lo dice honestamente**, no lo disimula: un paquete español dentro de una interfaz inglesa es un hecho del producto, no una carencia que tapar.

### Qué cambia en el repositorio

`ETAPAS.md` se recorta con esta decisión: sección propia de **v1** con lo que bloquea la salida, E6 y E7 marcadas **post-v1**, y las casillas de corpus de la etapa 3 acotadas **a los 12**. El orden de autoría del corpus deja de ser por marco y pasa a ser **por familia de reloj**, empezando por los dos que tienen fecha encima, que el propio corpus ya trae verificados con su cita: **AI Act art. 111.4, con límite 02-12-2026** (lo fija el apartado, añadido por el Reglamento (UE) 2026/1744) y **CRA art. 14, aplicable desde el 11-09-2026**.

---

## D-20. El sistema es el producto, el corpus es combustible

**Fecha:** 01-09-2026. **Decisión de Marcos**, no consulta. Es la hermana de D-19 y va en la otra dirección: D-19 recortó **anchura de corpus**, esta mueve **dónde está el valor**. El corpus deja de ser el negocio sin dejar de ser el diferenciador, y lo que se vende pasa a ser el sistema: el GRC que asiste y que **hace cosas dentro del entorno del cliente**.

La vara no baja. Sigue siendo más de 9,7, y el punto (e) es el que impide que ese 9,7 se consiga moviendo pesos.

### (a) Los 12 marcos se terminan igual, pero como corpus community-grade

Gratis, sin garantía, con la doctrina del descargo cargando la honestidad. Se escriben los doce con el mismo cuidado, los mismos dorados y el mismo linter legal; lo que cambia es que **no son lo que se cobra**.

No es una rebaja del corpus, es dejar de apoyar el modelo de negocio en lo único que un competidor puede replicar generando datos. El corpus sigue siendo lo que hace creíble todo lo demás.

### (b) La revisión jurídica externa y los design partners salen del camino del 9,7

De **puertas** pasan a **aceleradores opcionales**: suman cuando lleguen, no bloquean mientras no estén. Un plan cuya puerta es que un tercero firme es un plan que no depende de quien lo ejecuta.

Consecuencia en puertas, **sin rebajarlas**, que es la parte que hay que mirar con lupa:

- **D15 (legalidad del corpus)** se verifica con **estratos ejecutables** (el linter, que ya corre) + **fuentes primarias con el invariante 10** (cada dato con qué se miró, dónde y qué día) + **el descargo**. Sin firma externa.
- **D14 (open core self-serve)** se verifica con **tres meses de medición real de uso** + **checkout operando**. Sin referencias de partners.

Las dos puertas nuevas son **más caras de falsificar** que las que sustituyen: una firma es un PDF, tres meses de uso medido no lo son.

### (c) La IA de adopción entra en la v1

Las piezas **1, 2, 3, 4 y 7** de `docs/ia.md`: entrevista asistida desde los documentos del cliente, la pregunta con su consecuencia al lado, mapeo de la evidencia que ya tiene, plan de los primeros 30 días, y extracción de metadatos de la evidencia.

Con ellas se **adelantan de E5** las dos piezas que necesitan: **FTS5 (BM25)** y el **verificador de citas por hash**. No es una excepción al orden: sin búsqueda no hay dónde buscar la cita, y sin verificador la propuesta no se puede enseñar.

Las condiciones no son nuevas, son las que ya están escritas y aquí se recuerdan porque ahora tienen fecha:

1. Toda salida es una **`puertos.Propuesta` con cita verificada por hash ANTES de enseñarse** (invariante 9). Si no resuelve a texto real, se descarta, no se muestra.
2. **En línea, en el punto de fricción. Jamás una pestaña de chat** (`docs/ia.md` §5). Si hay que abrir un sitio aparte para usarla, está mal puesta.
3. **El camino completo en verde con `PLAZUM_SIN_IA=1`**. La puerta ya existe y hoy es casi vacía; con esto deja de serlo, y es lo que convierte «el núcleo es determinista» en comprobable en dos minutos.
4. **Los evals adversariales del subconjunto, dentro de la v1**, y con la inyección vía documento incluida: **un PDF que sube el cliente es entrada adversaria**. Es el invariante 8 en su tercera forma, la de la frontera de entrada, aplicada a la ingesta: de un PDF del que no se extrae nada interpretable **no se saca un valor por defecto**, se saca un error.

**El resto de E5 queda detrás**: runtime de agentes, los tres agentes de análisis, MCP y cuestionarios entrantes. Sirven a quien ya adoptó; éstas consiguen que adopte.

### (d) El tier de 1.490 se reencuadra como suscripción de vigilancia

Lo que se paga es **plazo objetivo de actualización, changelog curado y aviso proactivo**, que es exactamente lo que la web ya lista. **Sin lenguaje de garantía jurídica en ninguna parte**: ni «respaldado», ni «revisión jurídica publicada» como argumento de venta, ni «contractual» pegado a un plazo.

La **pieza 12 de `docs/ia.md`** (notas de alcance de la vigilancia normativa, redactadas por IA desde el diff del BOE y verificadas por una persona) **sube de prioridad**, porque es lo que hace que esa suscripción sea sostenible por una persona. Es IA aplicada a nuestro coste de producción, no a la experiencia del cliente, y decide si el modelo aguanta.

**El SSO sigue gratis, y se dice alto.** Cobrar el SSO es el peaje que todo el sector cobra y es el que hace que un CISO desconfíe del resto del precio.

### (e) La regla de honestidad del ponderado

Al bajar el peso de D3, **el 9,7 global sólo sale si D5, D12 y D8 suben de nota REAL** antes de la pasada de verificación. E6 y E7 siguen dentro del 9,7, detrás de la v1.

> **Una nota que sube por reponderación sin que suba nada real se dice en voz alta.**

Y aquí la regla se cobró su primera pieza al medirla, que es la mejor forma de que no sea un eslogan. Con el movimiento de pesos de esta decisión (D3 8→6, D5 6→7, D8 6→5, D12 6→8, la suma sigue siendo 109):

| | antes | después | movimiento |
|---|---|---|---|
| global sobre las notas de **diseño** | 9,5945 | 9,5963 | **+0,0018** |
| global sobre las notas **reales** de hoy | 6,1761 | **6,1257** | **−0,0505** |

**La reponderación no regala media décima: cuesta media décima.** El motivo es que las dos dimensiones que ganan peso (D5 y D12) son hoy de las más vacías del tablero (2,0 y 1,5), así que darles peso empeora el número honesto. Es lo que tiene que pasar cuando una decisión mueve la promesa hacia lo que todavía no está construido, y es la prueba de que este movimiento de pesos no se hizo para que saliera una nota.

**El movimiento se decidió con una regla falsable, no con un argumento** (02-09-2026). La primera propuesta bajaba D3 a 4 y subía D8 a 7, y costaba −0,1055. La verificación externa la rebatió: D-20 mueve **dónde está el dinero, no dónde está el foso**, y D8 es al revés, la dimensión más commoditizada del GRC. Su contrapropuesta (D3 a 6, D8 a 5) venía con su propia condición de rechazo escrita antes de calcular: **si dejaba el global honesto por encima de 6,1761, estaba regalando nota**. Queda en **6,1257**, por debajo, así que se acepta. La aritmética de las tres variantes, en `docs/diseno.md` §14.

**Y una consecuencia medida que hay que decir aunque incomode**: con los pesos nuevos, cada décima de D12 vale ahora **8/109 = 0,073** puntos de global en vez de 6/109 = 0,055, un tercio más. O sea que esta decisión **no hace el 9,7 más fácil, lo hace más caro**, porque lo cuelga sobre todo de D5 y D12, que están a 2,0 y 1,5.

### Qué cambia en el repositorio

1. **`ETAPAS.md`**: el bloque IA de adopción entra en la sección de v1, **E5 se parte** en lo adelantado (FTS5 y verificador de citas, que suben a v1 con las piezas) y lo que queda detrás, y se recuenta.
2. **`docs/diseno.md` §14**: pesos nuevos con la aritmética a la vista, no sólo el resultado.
3. **`docs/guia.md` §11, `web/index.html`, `docs/ia.md` y `docs/diseno.md`**: barrido de lenguaje. Donde «respaldado» prometía garantía o revisión jurídica, se reescribe.
4. **Lo que NO cambia**: los invariantes 8, 9 y 10, la frontera legal del corpus (invariante 3) y la capa probatoria cerrada (D-2). Esta decisión mueve dónde está el valor, no lo que se puede afirmar.

## D-21. La capa visual entra en la v1, con las reglas de la casa intactas

**Fecha:** 02-09-2026. **Decisión de Marcos**, tomada sobre una medición del árbol, no sobre una impresión.

**Lo que se midió.** El frontend de hoy es excelente de ingeniería y corto de producto: **454 líneas de CSS y 7 plantillas**, CSP estricta, sin build, dos temas, contrastes con puerta axe, catálogo en dos idiomas y navegable sin JavaScript. Y a la vez: **sin app shell, sin panel de inicio, sin una sola visualización, sin identidad y sin hoja de impresión**. Las 21 casillas de la sección v1 de `ETAPAS.md` son todas mecánica del camino; **ninguna dice nada de esto**.

**Por qué es una decisión y no un adorno.** La tesis de D-20 es que el sistema es lo que vale el dinero. Si la v1 sale funcionando y con aspecto de intranet vieja, esa tesis se desmiente sola delante del comprador, y se desmiente **antes de la primera pregunta**. Es media venta perdida en la pantalla de entrada, que es el único sitio donde no hay forma de recuperarla.

### Lo que entra

- **App shell** con barra lateral persistente: los seis pasos del camino y dónde estás. Los pasos salen de `camino.Canonico()`, nunca de una lista escrita en una plantilla: el trinquete de alcanzabilidad vigila justo eso.
- **Hoy convertida en panel de inicio**: cifras grandes arriba (vence esta semana, sin constancia, marcos activos) y **cada cifra clicable hasta su derivación**. Es la puerta D11-c cumpliéndose donde más se ve.
- **Escala tipográfica y densidad de producto.**
- **Tablas con cabecera fija, filtros como fichas, estados como pastillas con su rótulo escrito.**
- **Estados vacíos con dibujo además del verbo.** El verbo ya está (D11-b) y no se quita: el dibujo se suma.
- **Hoja de impresión** para el acta y el board pack, que son documentos que alguien va a imprimir para un consejo.
- **Identidad**: marca, favicon y un color propio.

### Las restricciones, que no se negocian

Sin npm, sin build, sin CDN. **Un solo CSS servido del binario.** Una fuente variable **auto-alojada** en `estatico/`, con `font-src 'self'` y sin relajar ninguna otra directiva de la CSP. htmx como está, y ninguna dependencia de JS nueva. **Todo tiene que seguir navegándose sin JavaScript.**

Y las puertas que ya existen **no se relajan ni una**: axe sigue siendo bloqueante, y los contrastes se miden **en los dos temas**. Si un color no pasa el contraste, el que cambia es el color.

### El riesgo que trae, dicho antes de construirlo

Un panel de inicio con cifras grandes es **exactamente el sitio donde se rompe la regla del descargo**. Una cifra grande que diga «14 sin constancia» se lee como acusación, y lo no constatado no es un incumplimiento: es una ausencia de dato, y plazum no sabe distinguirlos. El descargo va **con el dato**, no en una nota al pie, y toda rama de descargo necesita su control positivo (M47). Acusar en falso es el único error que un producto de cumplimiento no puede cometer ni una vez.
