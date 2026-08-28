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

**Un hospital es entidad esencial de NIS2 por el anexo I de la Directiva y no es ninguno de los once.** Los requisitos técnicos del anexo de 2024/2690 **no le alcanzan**. Enseñárselos no es un matiz: son 61 relojes y un anexo de 153 puntos de trabajo que no le tocan.

**Qué se decide.** Que esto **no es un detalle de la transcripción de un paquete, sino la forma de trabajar con todo marco derivado**:

1. **El ámbito de un acto de ejecución o delegado se lee en SU artículo de ámbito, nunca se hereda del acto base.** Un reglamento de ejecución puede alcanzar a menos que su directiva, y normalmente alcanza a menos.
2. **Toda regla de aplicabilidad se prueba en las DOS direcciones** (ya está en `CLAUDE.md`), y la dirección que hay que escribir con más cuidado es la negativa, con el artículo de la exclusión al lado.
3. **La dirección negativa lleva su propio control de que no se cumple sola.** El test comprueba además que el sujeto excluido **derive alguna otra obligación**: sin eso, un motor que no derivara nada pasaría la comprobación de exclusión y no habría comprobado nada. Es la trampa del test de ausencia, la misma familia que las dos formas de la nada del invariante 8.

**Por qué está en decisiones y no sólo en el código.** Porque es la diferencia entre **transcribir y entender**, y es demostrable delante de un comprador: cualquiera puede abrir el art. 1, contar once tipos, buscar "hospital" y no encontrarlo. Un competidor con corpus en hoja de cálculo no puede enseñar esa comprobación porque su modelo no tiene dónde ponerla: en una columna llamada *"NIS2"* no cabe la frase *"salvo que no seas ninguno de estos once"*.
