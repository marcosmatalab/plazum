# iso42001: ISO/IEC 42001:2023, sistema de gestión de la IA

**Paquete referencial. No trae ni una palabra de la norma.** Solo identificadores
de cláusula y de categoría del anexo A, con una etiqueta corta nuestra, y los
rituales de plazum con sus cadencias. El texto lo aporta quien lo use con su
propia copia licenciada de ISO. El linter lo comprueba en cada ejecución: un
paquete referencial con más de 120 bytes de texto normativo no carga.

## Qué trae

- **Las 32 cláusulas** de la 4 a la 10, con la estructura armonizada que comparte
  con ISO/IEC 27001, más lo que la 42001 añade: **6.1.4 y 8.4, la evaluación de
  impacto del sistema de IA**, que no existen en la 27001 y son el puente con el
  AI Act.
- **Las nueve categorías del anexo A** (A.2 a A.10) con su número de controles:
  38 en total.
- **Siete rituales con reloj**, con sus 21 casos dorados ejecutados contra el
  motor en cada `./comprobar.sh`. El detalle, en `RITUALES.md`.

## El hueco declarado: los 38 controles del anexo A, uno a uno

**No están, y no es un olvido.** El paquete declara las nueve categorías del
anexo A y el número de controles de cada una, que es estructura publicada, pero
no los 38 títulos de control individuales.

El motivo es el invariante 3 de este proyecto, y conviene decirlo entero: la
norma es de pago, este paquete se escribió sin una copia licenciada delante, y
**escribir los títulos de memoria sería fabricar el catálogo**. Un título
inventado que se parece al bueno es peor que un hueco, porque un hueco se ve y un
título casi correcto no. Y tampoco se copian de un repositorio de terceros de
GitHub, aunque diga MIT: la licencia de un repositorio no alcanza a contenido que
el subidor no poseía.

**Cómo se cierra**: con la copia licenciada delante, se añaden los 38 controles
como identificador (`A.6.2.4`) y etiqueta corta, exactamente igual que están los
93 del anexo A de la 27001. Es media hora de trabajo y ningún riesgo, una vez se
tiene la copia.

Lo que ese hueco **no** afecta: los siete rituales, que son de plazum y están
completos, y son donde está el valor del paquete. Un catálogo de controles lo
tiene cualquiera; el reloj no lo tiene nadie.

## Por qué este paquete y por qué ahora

La 42001 es la norma de sistema de gestión con la que se certifica el gobierno de
la IA, y es la que un comprador va a pedir en cuanto el AI Act empiece a doler.
Su coste marginal aquí es casi nulo: la estructura armonizada de las cláusulas 4
a 10 es la misma que la de la 27001, que ya estaba, y la aritmética de los
rituales es la misma.

Lo que **no** es marginal es el punto 6.1.4. La evaluación de impacto del sistema
de IA de la 42001 y las obligaciones del AI Act se alimentan del mismo trabajo, y
hoy una organización lo hace dos veces porque nadie le dice que es el mismo. Ese
mapeo tiene casilla propia en `ETAPAS.md` (etapa 3) y **no está construido
todavía**: este paquete es su mitad de arriba.

## Lo que NO hace este paquete

- **No dice si estás conforme.** Dice qué hay que revisar y para cuándo.
- **No trae los ciclos de certificación** (seguimiento anual, recertificación a
  tres años). No salen de la norma sino del esquema de la entidad certificadora,
  y ponerles un número por nuestra cuenta sería inventar.
- **No cubre el disparador por cambio sustancial** (un modelo reentrenado, un
  caso de uso nuevo). Es la familia D del censo y necesita el motor de eventos.
  En un AIMS va a ser el disparador más usado.
- **No sustituye a la copia de la norma.** Sin ella no se puede implantar nada de
  esto, y este paquete no la reemplaza ni lo pretende.

## Las cadencias son un valor de partida, no un dogma

ISO no fija ningún número de meses en ninguna de sus normas de sistema de
gestión. Los doce meses de aquí son de plazum, están justificados por escrito en
`RITUALES.md` y se cambian en la copia del paquete del cliente sin tocar código.
Eso es exactamente el valor del estrato referencial: poner el número y
defenderlo, no copiar el texto.
