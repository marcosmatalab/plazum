# La doctrina de IA de plazum

> **Estado a 26-08-2026: ESPECIFICACIÓN. No hay adaptador de IA y no lo va a haber hasta después de la Familia A del corpus.** Lo que sí existe ya es el **invariante 9 con sus dos puertas**, escritas antes que el adaptador a propósito: la única forma de que un invariante aguante es que esté puesto antes de que haya presión para saltárselo.

**La frase corta:** mucha más IA, con arnés duro, para **implantación y remediación**. El cumplimiento sigue siendo determinista y la IA sólo entra donde hoy hay fricción.

---

## 1. El invariante 9, que es lo que hace esto verdad y no marketing

> **La IA vive en adaptadores y superficies. `nucleo/` no conoce el puerto de IA y no lo importa nunca.**

Dos puertas mecánicas, y las dos existen hoy:

**Puerta 1 — el AST.** `TestElNucleoNoConoceLaIA` comprueba dos cosas, porque la primera sola no basta: que `nucleo/` no importa `plazum/puertos`, y que **ni siquiera nombra** `Asistente`, `Propuesta`, `LLM` ni `Ollama`. Sin la segunda, alguien copia el interfaz al núcleo para no importar el paquete y cumple la letra rompiendo el fondo.

Una nota honesta sobre esa puerta: la rama del import **no se puede demostrar con una mutación**, porque `plazum/puertos` importa `plazum/nucleo/corpus` y añadir el import al núcleo da un ciclo que el compilador caza antes que el test. Es la trampa que este repositorio ya tiene escrita ("una mutación que no compila no produce `--- FAIL`") con cara nueva: aquí la mutación no es que se me olvidara hacerla compilar, es que **no puede** compilar. La comprobación se queda porque el ciclo existe hoy por una razón que puede desaparecer, y se demuestra sobre un fichero sintético.

**Puerta 2 — la suite entera con la IA desactivada.** Un paso de CI corre `go test ./...` con `PLAZUM_SIN_IA=1`. **Hoy es casi vacía** y se escribe igual: el interruptor tiene que existir antes que el adaptador, o el adaptador se escribe sin pensar en poder apagarlo, y entonces el "modo sin IA" no es un modo, es una casilla que no hace nada.

**Lo que esa segunda puerta consigue**, y es la mitad del argumento de venta: convierte *"el núcleo es determinista"* de eslogan en **hecho comprobable por cualquiera en dos minutos**. Si algún test necesita la IA para estar verde, la IA ha entrado en el camino del cumplimiento y hay que sacarla.

---

## 2. El arnés, con dientes

1. **Toda salida de modelo es una PROPUESTA con cita, nunca un hecho.** Confirmarla es un acto humano, y **quién la confirmó y cuándo va al ledger**.
2. **La cita se verifica determinísticamente ANTES de enseñar la propuesta.** Si no resuelve a texto real del corpus o del documento que subió el cliente, la propuesta **se descarta, no se enseña**. Es el verificador de citas por hash que ya estaba planeado (`puertos.Propuesta.HashFuente`), y es **la puerta antialucinación: mecánica, no estadística**.
3. **Acciones tipadas de un conjunto cerrado.** Un agente emite propuestas de tipos declarados. Nunca acción libre, nunca escritura directa.
4. **Presupuesto por tarea**, en tokens y en euros, aplicado y **visible para el operador**.
5. **Transcript cifrado al ledger**: qué se preguntó, qué contestó, qué hizo la persona.
6. **Local por defecto. Ollama de serie.** El modelo en la nube es *opt-in* con pantalla de consentimiento explícita, y **el consentimiento se anota en el ledger**. Los incumplimientos de un CISO saliendo hacia la API de un tercero es justo lo que ese CISO no va a firmar, y es lo que nos separa de todos los de la nube.
7. **El modo sin IA sigue siendo completo.** Con la IA apagada, el producto hace todo lo que promete el README. Si no, la premisa es falsa.
8. **Evals publicados en cada release**, con modelo y versión fijados. El hito sigue siendo *el primer GRC que publica la precisión de su IA*.

---

## 3. La restricción legal que se convierte en argumento de venta

**La frontera legal decide qué puede explicar la IA**, y no es una limitación que haya que disimular: es lo que nos distingue.

- Sobre **estrato transcrito** (ENS, RGPD, NIS2, CRA, AI Act, y ahora `nis1-es`) la IA **puede explicar el texto**, porque lo tenemos y es reutilizable.
- Sobre **estrato referencial** (ISO, PCI DSS, SOC 2, TISAX) **no hay texto**, así que la IA **no explica el texto y lo dice**: sólo trabaja sobre el ritual y la cadencia declarados.

**Escrito como propiedad del producto:** la IA de los competidores va a inventarse alegremente el texto de una cláusula de ISO. **La nuestra no puede, porque no lo tiene, y avisa.**

Eso no es una promesa de comportamiento del modelo: es una consecuencia mecánica de la regla 2 del arnés. Sin texto en el corpus no hay cita que resuelva, y sin cita que resuelva la propuesta se descarta antes de enseñarse.

**Test con control negativo, cuando exista el adaptador:** pedirle que explique una cláusula referencial devuelve la negativa con el motivo, nunca un párrafo.

---

## 4. Dónde entra la IA, por punto de fricción

Ordenado por lo que le duele al CISO, no por lo que es bonito de construir.

### 4.1. Adopción: los primeros treinta días (lo que no existía)

Es la mitad del producto que no estaba diseñada (`docs/decisiones.md` D-8).

1. **Entrevista asistida.** Suelta su política, su inventario, su informe de auditoría y el Excel de controles que tiene todo CISO, y el sistema **propone cada respuesta con la cita del documento y la página** donde la leyó. Él confirma o corrige. Es **la mayor mejora de tiempo hasta el primer valor de todo el producto**.
2. **Explicar la pregunta y su consecuencia, al lado de la pregunta**: *"si contestas que sí, se te activan estas nueve obligaciones"*. **El abandono se produce exactamente ahí.**
3. **Mapeo de la evidencia que ya tiene**: propone qué documento suyo satisface qué obligación, con cita.
4. **Plan de ataque de los primeros treinta días.** 130 obligaciones en rojo es un muro y es donde se abandona el producto. La IA **ordena y agrupa por trabajo**: *"estas catorce se cierran con una sola auditoría"*. Es la composición cross-framework ya diseñada, presentada como plan, en la pantalla **Hoy**.
5. **Importador del corpus del cliente**: su Excel se convierte en paquete propio en vez de tirarse.

### 4.2. Operación diaria

6. **Explicar la obligación en cristiano**, anclado al texto transcrito con su cita. Es el "guiado" del diseño, y respeta el apartado 3.
7. **Extracción de metadatos de la evidencia**: sube un PDF y se proponen fecha, alcance, firmante y caducidad. **La tarea repetitiva que más veces al año hace un CISO.**
8. **Redactor del entregable documental.** Hoy la clase documental sólo exige que el documento **exista**. Redactarlo desde plantilla y contexto es la diferencia entre **señalar y resolver**.
9. **Generador de propuestas de remediación.** Hoy `remediacion` es sólo una etiqueta que valida el linter (`nucleo/corpus/paquete.go`): no hay motor. Propuesta como código, revisión por trozos, **nunca se aplica sola**.
10. **Cuestionarios de seguridad entrantes**: **sube de prioridad**. Cada uno es un día entero del CISO y es el favor que hace que te recomienden.
11. **Borrador del acta 9.3 y del board pack** desde los datos deterministas, ya planeado en la etapa 4.

### 4.3. Y la que no está en ninguna lista y es la más rentable

12. **Notas de alcance de la vigilancia normativa.** El §11 de `docs/guia.md` vende *"changelog curado con notas de alcance"*. **Hoy el coste marginal de producir esas notas son tus fines de semana, y por eso la suscripción no escala.** Con la IA redactando el borrador desde el diff del BOE y tú verificándolo, el nivel **Respaldado** pasa de compromiso caro a sostenible por una persona.

Es **IA aplicada a nuestro coste de producción, no a la experiencia del cliente**, y decide si el modelo aguanta a 30 clientes.

---

## 5. Cómo se presenta, que es la mitad del valor

**Nada de una pestaña de chat.** Un CISO que no sabe qué preguntar delante de una caja vacía se va.

La IA va **en línea, en el punto de fricción**: al lado de la pregunta de la entrevista, al lado de la obligación en rojo, al lado del botón de subir evidencia. Cada propuesta con **su cita visible** y **dos botones, confirmar o corregir**.

**Si hay que abrir un sitio aparte para usar la IA, está mal puesta.**

---

## 6. Calendario

- **Se especifica ahora**, en este documento, con el invariante 9 y sus dos puertas escritas aunque el adaptador no exista.
- **Se construye después de la Familia A**, por dependencia dura: no se puede mapear documentos contra obligaciones que no están escritas, ni explicar textos que no están transcritos.
- **Cuando entre, absorbe la etapa 5**: los agentes de análisis (contradicciones, huecos de evidencia) **bajan por debajo de las doce de arriba**, porque sirven a quien ya adoptó y éstas consiguen que adopte.
