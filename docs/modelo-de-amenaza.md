# Modelo de amenaza del expediente

> **Para qué sirve este documento.** El expediente verificable es la tesis del producto, así que su promesa tiene que estar escrita en un sitio donde se pueda contrastar, no repartida por comentarios de código. Aquí está qué demuestra, qué no demuestra, contra qué emisor, y bajo qué supuestos.
>
> Está escrito para que un receptor hostil lo use en contra. Si encuentras algo que la promesa de aquí no cubre y el producto insinúa que sí, eso es un fallo del documento y hay que arreglarlo. Si encuentras algo que la promesa ya excluye, eso no es un fallo, es este documento haciendo su trabajo.

## La promesa, en una frase

Un tercero con el fichero y el binario, **sin red** y **sin confiar en quien lo emitió**, recalcula desde cero la aplicabilidad, los plazos y los estados de control que el expediente afirma, y obtiene exactamente lo mismo o le dice dónde no coincide.

Nótese lo que la frase dice y lo que no. Dice que **el cálculo** es reproducible. No dice que los **hechos de partida** sean ciertos, ni que el expediente contenga **todo** lo que existe.

## El emisor contra el que esto se defiende

Tres emisores, en orden de hostilidad. La frontera del producto pasa entre el segundo y el tercero, y pasa ahí a propósito.

### Emisor A, el descuidado

No miente. Se equivoca: usa una versión vieja del corpus, aplica una regla que no le tocaba, calcula mal un plazo, arrastra un estado que ya caducó.

**Cubierto entero.** Es el caso para el que sirve la recomputación, y es además el caso que ocurre de verdad todos los días.

### Emisor B, el interesado

No fabrica criptografía. Quiere que el expediente salga mejor de lo que es tocando lo que cree que nadie mira: retirar una observación que falla, declarar unas anclas de corpus que le cuadren, apoyarse en un borrado legal para limpiar un incumplimiento, evaluar una obligación fuera de la vigencia de su norma.

**Cubierto**, y cada una de esas cuatro cosas costó un ataque encontrarla. La sección siguiente dice con qué.

### Emisor C, el que controla su instalación entera

Ejecuta el producto en su propio hierro. Puede parar el ledger cuando quiera, puede llevar dos, puede alimentar el sistema con hechos falsos desde el primer día, puede enseñar el expediente que le conviene.

**No cubierto, y por decisión.** Contra un emisor así no hay artefacto offline que valga: cualquier defensa exige un testigo externo que observe la instalación, y ese testigo rompe la promesa de autoalojado y offline. La sección "Truncado de cola" lo desarrolla, porque es la forma concreta en que este emisor gana.

Lo que sí se sostiene contra el emisor C es más modesto y no es poco: **no puede mentir barato**. Falsear el expediente le exige mantener una instalación paralela coherente en el tiempo, no cambiar un número en un PDF. Y todo lo que sí enseñe es recomputable, o sea que la mentira tiene que ser consistente con las normas reales.

## Qué demuestra el expediente

Cada punto es una capa de `Verificar` en `nucleo/expediente`, con el ataque que la puso ahí.

| Qué se demuestra | Cómo | Qué lo puso ahí |
|---|---|---|
| La cadena está encadenada, su raíz Merkle es la publicada y su checkpoint lleva firma y sello válidos | capa 1, `ledger.CadenaV2.Verificar` con la confianza **que aporta el receptor** | ataques 1 a 9 |
| El corpus usado es el corpus publicado, y no uno que el emisor se escribió | capa 2.b, el digest se **recalcula** desde el contenido y se contrasta con el ancla **del receptor** | ataque 10 |
| Ninguna obligación se evalúa antes de la vigencia de su norma | capa 3 | revisión de temporalidad |
| La aplicabilidad declarada sale de reejecutar el Datalog de los paquetes | capa 3, motor de aplicabilidad | etapa 1 |
| Cada vencimiento declarado sale de recalcularlo con el motor temporal | capa 4 | etapa 1 |
| Toda observación declarada está anclada en la cadena | capa 5b, dirección directa | ataque 11 |
| **Toda entrada viva de la cadena aparece declarada como observación** | capa 5b, dirección inversa | **ataque 13** |
| Un borrado legal no puede mejorar la postura de nadie | capa 5c, la prueba pasa a `obsoleto` y escala al auditor | P1 10 |
| Los estados de control salen de recalcularlos con la misma función pura | capa 6 | etapa 1 |

**La pieza que hace que nada de esto sea circular** es que las anclas de corpus, las claves de operador y el verificador del sello **entran por parámetro desde el receptor**, no desde el fichero. Cuando vivían dentro del expediente, el emisor se escribía sus propias anclas y la verificación se demostraba a sí misma. Sin anclas del receptor, `Verificar` no da un aviso, da una discrepancia y el expediente sale inválido.

## Qué NO demuestra el expediente

Esta es la mitad que suele faltar, y la que un receptor tiene que leer antes de decidir qué peso le da al documento.

### 1. Truncado de cola

**Un emisor que corta la cadena por el final no es detectable, y no lo va a ser.**

La cadena es de sólo añadir por construcción del encadenamiento de hashes: nadie puede insertar, borrar ni reordenar en medio sin romper el enlace. Lo que ningún artefacto offline puede forzar es que el emisor enseñe la cadena **entera**.

En concreto: si el emisor cierra un checkpoint en la entrada N y las entradas N+1 a M le resultan incómodas, presenta el expediente en el checkpoint N. La cadena es internamente coherente, la raíz Merkle cuadra, el sello RFC 3161 verifica, la recomputación da lo que dice. Todo sale verde. Lo que falta no está en el fichero: falta alguien que diga "el día T esa cadena ya medía M".

**Ese alguien es un testigo externo publicado**, un log de transparencia o un difusor de checkpoints, y no lo vamos a montar. Las tres razones, por orden de peso:

1. **Rompe el offline.** Verificar pasaría a exigir red, y el argumento entero del producto es que el receptor recomputa sin depender de nadie, ni siquiera de nosotros.
2. **Rompe el autoalojado y filtra metadatos.** Publicar checkpoints publica la existencia, el tamaño y el ritmo de crecimiento del ledger de cumplimiento de un cliente. Un CISO no acepta eso, y tiene razón en no aceptarlo.
3. **Crea una dependencia de servicio.** Si el testigo desaparece, los expedientes ya emitidos dejan de verificar. Un artefacto probatorio que caduca porque un tercero cerró un servidor no es un artefacto probatorio.

**Lo que sí se puede hacer sin testigo**, y que conviene que el receptor sepa:

- **Mirar la fecha del sello.** El sello RFC 3161 del checkpoint acota por arriba cuándo se cerró. Un emisor puede esconder entradas recientes, pero no puede presentar un checkpoint viejo como si fuera de hoy: la fecha está en el informe y es de un tercero. Esto no detecta el truncado, acota su antigüedad, que es lo único gratis que hay aquí.
- **Que el receptor sea el testigo.** Quien exige el expediente puede exigir además que cada entrega **extienda** a la anterior. Dos expedientes del mismo emisor cuyo `Hasta` no crece, o cuya raíz no es coherente con la que ya se guardó, es exactamente la señal que falta, y la guarda el receptor sin coste ni red. Es un control operativo de la contraparte, no una propiedad del formato, y como tal se dice.

  **Hueco conocido, y aquí queda escrito:** hoy el receptor puede comparar que `Hasta` no retrocede, pero **no hay prueba de consistencia entre dos checkpoints** (RFC 6962, apartado 2.1.2). O sea que dos raíces sucesivas se pueden comparar por número de entradas, pero no demostrar que la segunda extiende a la primera. Es una casilla identificada, no implementada, y no bloquea nada de lo que el producto promete hoy.

### 2. Que los hechos de partida sean ciertos

El expediente demuestra que **del insumo declarado sale el resultado declarado**. No demuestra que el insumo describa el mundo.

Si el emisor declara que hizo la copia de seguridad el día 3, el motor calcula correctamente que estaba en plazo. Que la copia existiera es un problema de evidencia, no de cálculo, y lo acota el blob con su digest, que prueba que **ese fichero** es el que se declaró, no que ese fichero diga la verdad.

Basura dentro, basura firmada y encadenada fuera. Ninguna capa criptográfica arregla eso, y ningún GRC del mercado lo arregla tampoco.

### 3. Que la supresión declarada sea la que ocurrió

Está escrito también en el código, en `SupresionDeEvidencia`, y se repite aquí porque es un límite de la promesa y no un detalle de implementación: **la entrada de la cadena no se compromete con la prueba que ancla**, y el contenido suprimido es irrecuperable por construcción, porque la clave se destruye. Así que ningún verificador puede atribuir un borrado a una prueba concreta.

Lo que sí impide el diseño es que ese borrado **beneficie** a nadie: de `pass` y de `fail` se sale igual a `obsoleto`, y `obsoleto` escala al auditor.

### 4. Que el alcance esté completo

El expediente no demuestra que el perímetro declarado sea el perímetro real de la organización. Un emisor puede declarar tres filiales de cinco.

Esto no es un descuido: **no existe el dato contra el que contrastarlo** dentro del artefacto. La comprobación es externa, del receptor, y es la misma que hace hoy con cualquier certificación: mirar si el alcance declarado se parece a lo que sabe de la empresa.

### 5. Que el corpus sea jurídicamente correcto

El motor recomputa fielmente lo que el paquete dice. Que lo que el paquete dice sea lo que dice el BOE lo sostienen los **casos dorados** derivados del texto legal, y en última instancia lo sostiene una persona leyendo la norma. Es una garantía de proceso, no criptográfica, y se declara como tal.

Un digest que cuadra prueba que el corpus no se manipuló después de publicarse. No prueba que estuviera bien el día que se publicó.

### 6. El instante

`ComoEstaba` lo declara el emisor. El reloj legal entra siempre como dato porque `nucleo/` no llama a `time.Now()`, y eso, que es lo que hace el motor determinista y verificable, significa exactamente que **el instante es un insumo más**. El sello RFC 3161 acota el instante de un **checkpoint**, no el de cada entrada.

## Supuestos, dichos en voz alta

Si alguno de estos no se cumple, la promesa de arriba no se sostiene, y no es un fallo del producto sino un supuesto roto.

1. **El receptor aporta anclas, claves de operador y verificador de sello.** Sin ellos la verificación sería circular, y el código lo trata como discrepancia, no como aviso.
2. **El binario del verificador es el auténtico.** Un verificador manipulado dice lo que quiera. Es lo que resuelven la reproducibilidad del build y la firma de la release, fuera de este documento.
3. **La destrucción de clave es efectiva.** El borrado legal descansa en que la clave de la entrada suprimida deja de existir de verdad, copias y respaldos incluidos.
4. **Ed25519, SHA-256 y el sellado RFC 3161 no están rotos.**
5. **El registro de corpus del receptor es fiable.** Las anclas valen lo que valga su origen.

## Regla permanente: del ataque 14 en adelante

**La capa probatoria está cerrada.** Los ataques 1 a 13 se buscaron, se encontraron y se arreglaron. A partir de aquí, un hallazgo de esta familia **se documenta en este fichero, no se arregla**.

La única excepción: que el hallazgo **rompa la promesa escrita arriba**. Si aparece algo que hace falso el "recalcula y obtiene exactamente lo mismo", eso no es un límite conocido, es un defecto, y se arregla.

El motivo es de coste de oportunidad, y conviene dejarlo por escrito para no reabrirlo cada vez que aparezca un ataque bonito. La capa probatoria de este producto ya está muy por delante de lo que hay en el mercado y puntúa bajo en decisión de compra: nadie elige un GRC por su modelo de amenaza, lo elige porque llega al valor la primera mañana. Seguir puliendo aquí es seguir puliendo lo que ya gana.

## Método: cómo salió el ataque 13, y cómo salen los siguientes

**El ataque 13 no salió de revisar el diff. Salió de coger una propiedad que el código daba por buena e intentar tumbarla.**

La propiedad era "toda observación del expediente está anclada en la cadena". El código la comprobaba, con su test y su control negativo. La pregunta que la tumbó no fue "¿está bien implementada?", que era que sí, sino **"¿y la dirección contraria quién la mira?"**. Nadie. Un emisor podía retirar de la lista una observación que fallaba, dejando su entrada y su clave publicadas e intactas, sin tocar la cadena, sin destruir ninguna clave y sin poner ninguna lápida. Válido, cero discrepancias, y toda la maquinaria de borrado legal defendiendo una puerta con la pared abierta al lado.

De ahí, dos cosas que valen para todas las pasadas adversarias que vengan:

1. **Toda pasada adversaria elige una propiedad que el código da por buena e intenta tumbarla.** Leer el diff encuentra lo que el autor hizo mal. Refutar una propiedad encuentra lo que el autor **no pensó**, que es donde vive esta familia entera.
2. **Cuando una comprobación recorre una lista para contrastarla con otra, preguntar SIEMPRE si la dirección contraria también se recorre.** La que falta es la que el emisor usa. Ese es el patrón del ataque 13, y la forma más probable del 14.

Y una trampa del propio arreglo, que costó un test de más: el escenario base no tenía lápidas, así que una excusa colectiva del tipo "hubo un borrado legal, por eso falta" pasaba en verde. Hizo falta un segundo test con un borrado legal honrado **más** una retirada encubierta de otra observación para cerrar la puerta de verdad.

## Los trece ataques, para el que venga

Están en `nucleo/expediente/hostil_test.go` y `expediente_test.go`, cada uno con su nombre y su comentario. No se resumen aquí a propósito: un resumen se queda viejo y el test no.

Lo que sí conviene saber antes de leerlos: **cuatro de ellos pasaban la verificación** cuando se escribieron. Ese es el rendimiento real de una revisión hostil bien hecha, y es la razón de que la etapa no se cierre sin ella.
