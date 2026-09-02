# rgpd — Reglamento (UE) 2016/679

Texto del DOUE, transcrito (estrato transcrito, Decisión 2011/833/UE). Fuente: instantánea con huella de Cellar, CELEX 32016R0679.

**A quién alcanza.** El paquete entra en ámbito con el hecho `trata_datos_personales(E)`, que cubre al **responsable y al encargado** del tratamiento. Dentro, cada obligación dice de quién es: el art. 32.1.d obliga a **los dos** (así arranca el 32.1), y los arts. 12.3, 12.4, 33.1 y 34.1 obligan sólo al **responsable**. Al encargado el art. 28.3.e le da un deber distinto, asistir al responsable, y ese no está escrito todavía.

## Los seis relojes

| art. | qué | reloj | de quién es el número |
|---|---|---|---|
| 12.3 | facilitar al interesado la información sobre las actuaciones seguidas a su solicitud | plazo de **un mes** desde la recepción de la solicitud | de la norma |
| 12.3, 3.ª frase | informar al interesado de la prórroga y de sus motivos | plazo de **un mes** desde la **recepción**, no desde la decisión de prorrogar | de la norma |
| 12.4 | informar de las razones de la no actuación y de las vías de reclamación | plazo de **un mes** desde la recepción de la solicitud | de la norma |
| 32.1.d | verificar, evaluar y valorar la eficacia de las medidas | **P12M** | **de plazum** (`propuesto`) |
| 33.1 | notificar una violación de seguridad a la autoridad de control | plazo de **72 h** desde que se tiene constancia | de la norma |
| 34.1 | comunicar la violación al **interesado** cuando entrañe alto riesgo | **obliga y no tiene número**: «sin dilación indebida» | no hay número que dar |

### Los tres relojes del art. 12, y las cuatro trampas que traen

**Un apartado con tres verbos no son tres relojes iguales.** El art. 12.3 dice tres cosas: facilitar la información sobre las actuaciones en un mes, **poder** prorrogar otros dos meses, e informar de esa prórroga en un mes. La segunda es una **potestad** y no es un reloj: nadie puede incumplir una prórroga que no ha pedido. Las otras dos son deberes distintos y se incumplen por separado, así que son dos obligaciones.

**La prórroga no alarga la fecha que enseña plazum, y se dice.** Los dos meses adicionales sólo existen si se comunican al interesado dentro del primer mes, así que el único plazo que vincula sin ningún acto previo es el mes. Si has prorrogado en plazo, tu fecha exigible son tres meses desde la recepción y la que ves se queda corta, que es el lado seguro.

**El punto de partida es la recepción, siempre.** Ni el aviso de la prórroga ni la información de la no actuación cuentan desde el día en que se decide: los dos cuentan desde que llegó la solicitud. Decidir el día 25 no da un mes, da cinco días. Es lo que fija el segundo caso dorado de cada uno.

**Y el cómputo no es el del art. 33.1.** Un plazo en meses vence al final del día que lleva la misma fecha en el mes de destino, se recorta al último día si esa fecha no existe (31 de marzo → 30 de abril) y **se traslada al hábil siguiente** si cae en sábado, domingo o festivo: Reglamento (CEE, Euratom) n.º 1182/71, arts. 3.2.c y 3.4. Un plazo en **horas** como el del art. 33.1 no traslada nunca, por el mismo art. 3.4. Los tres relojes del art. 12 traen un caso dorado para cada regla.

### El art. 34.1: obliga y no tiene número

El art. 34.1 dice *«sin dilación indebida»* y **no da ninguna cifra**. Copiarle las 72 horas al art. 33.1 sería inventar un plazo: aquellas son para la **autoridad de control** y esta comunicación es para el **interesado**. Sale como *sin plazo legal* y el motor mide el tiempo transcurrido desde la constancia.

Cuelga del **mismo hecho** `constancia` que el art. 33.1, a propósito: una sola violación registrada enciende los dos relojes, que es lo que pasa de verdad.

Sólo sale con estado cuando consta la apreciación de **alto riesgo**, que la hace el responsable y va como hecho con su instante. Sin ella el hito queda *pendiente de hecho*, que no es lo mismo que decir que no hay nada que hacer. Las tres exclusiones del **art. 34.3** (cifrado y medidas equivalentes, medidas ulteriores que eliminan la probabilidad, y esfuerzo desproporcionado con comunicación pública en su lugar) **no están modeladas como hechos**: quien las aplique no declara el alto riesgo, y el reloj no sale con fecha.

### El art. 32.1.d y la lectura que hay detrás

El apartado 32.1 enumera cuatro medidas y **sólo la d) tiene ritmo**: seudonimización y cifrado, confidencialidad e integridad permanentes y capacidad de restaurar son **capacidades**, no ceremonias. La d) es *«un proceso de verificación, evaluación y valoración **regulares** de la eficacia»*.

**La lectura, escrita para que se pueda discutir.** El art. 32.1 dice que las medidas *«en su caso»* incluyan estas cuatro, o sea que la lista es **indicativa**. plazum trata la verificación como exigible igualmente, y no por comodidad: los **arts. 5.2 y 24.1** obligan a *poder demostrar* que las medidas siguen siendo apropiadas, y eso no se demuestra sin verificarlas. Si tu análisis concluye que no procede, quita el hecho y el reloj se apaga.

**Y el número lo pone plazum**, porque la norma dice «regulares» y nada más. Doce meses porque este apartado **no trae disparador por evento**: el calendario es su único refresco, y el art. 32.1 ata las medidas al «estado de la técnica» y al riesgo, que cambian de forma continua. La justificación completa, sus fuentes y sus instrucciones de uso (`cuando_cambiarlo`) van dentro de la obligación.

## Por qué está agrupado con el AI Act y con DORA

Tres reglamentos piden lo mismo con otras palabras, y **caen en la misma sesión** si sus fechas coinciden:

| | |
|---|---|
| RGPD art. 32.1.d | verificar la eficacia de las medidas técnicas y organizativas |
| AI Act art. 9.2 | revisar el sistema de gestión de riesgos del sistema de alto riesgo |
| DORA art. 6.5 | revisar el marco de gestión del riesgo relacionado con las TIC |

`plazum calendario --sentadas` lo dice en una línea: **«3 fechas de 3 marcos»**. Hay un test que lo exige (`TestUnaSentadaPuedeCubrirTresReglamentos`), porque una herramienta que trate cada marco por separado no puede escribir esa frase: para escribirla hay que tener los tres relojes a la vez.

## El entregable del art. 33: qué rellena plazum y qué no

El art. 33.1 dice **cuándo** (72 horas desde la constancia) y el **art. 33.3** dice **qué**, en cuatro letras. Son dos deberes distintos y se pueden incumplir por separado — notificar tarde, y notificar sin contenido — así que van como dos obligaciones y el reloj lo lleva sólo el 33.1.

La plantilla `rgpd.notificacion_a_la_autoridad_de_control` reparte sus siete campos en dos mitades, y esa división es el producto:

| lo rellena plazum | lo escribe quien responde |
|---|---|
| el identificador del incidente | la naturaleza de la violación (letra a) |
| el momento de la violación | el contacto del delegado de protección de datos (letra b) |
| **el momento de la constancia** | las posibles consecuencias (letra c) |
| | las medidas adoptadas o propuestas (letra d) |

**Los dos instantes van los dos, y separados.** El plazo cuenta desde la constancia, no desde que ocurrió: una brecha de marzo descubierta en julio no lleva cuatro meses de retraso. Confundirlos en el documento que se manda a la autoridad diría que se supo cuatro meses antes de saberse.

**Los campos que faltan no se rellenan con el vacío ni se presentan como un incumplimiento.** Cada uno sale con lo que es: *lo escribe quien responde*, o *no consta el dato — esto NO dice que no exista, dice que en tus respuestas no aparece*. Hay un test que lo exige palabra por palabra.

**Y lo que este entregable NO es:** una transcripción del formulario web de la AEPD ni el de ninguna otra autoridad de control. Es el **contenido mínimo que el Reglamento exige**. Mapearlo sobre el formulario concreto de una autoridad necesita ese formulario como fuente primaria y es una pieza aparte; llamarlo aquí «el formulario de la AEPD» sería afirmar algo que no se ha mirado.

## Lo que este paquete NO hace todavía

- **El resto del RGPD.** Están los seis relojes de arriba y nada más: no hay registro de actividades (art. 30), ni evaluación de impacto (art. 35), ni el contenido de los derechos del interesado (arts. 15 a 22), que es lo que hay que entregar dentro del mes del art. 12.3.
- **El art. 14.3, letra a)**, que es un plazo de un mes desde que se obtienen los datos cuando no se obtienen del interesado. Está **identificado y no escrito**, y el motivo es del motor, no del texto: las letras b) y c) del mismo apartado fijan límites que sólo pueden **adelantar** esa fecha (el momento de la primera comunicación al interesado, el momento de la primera cesión), y el `tope` del motor admite **uno solo** y deja el hito *pendiente de hecho* cuando su hecho no consta, que es la mayoría de los casos. Escribir sólo la letra a) daría una fecha **más tarde** que la legal siempre que aplique la b) o la c), y esa es la dirección en la que un GRC hace daño. **1 reloj esperando** a que el motor sepa decir «el más temprano de N límites condicionales».
- **El art. 12.5**, que permite cobrar un canon o negarse a actuar ante solicitudes manifiestamente infundadas o excesivas. No es un reloj: es una potestad, y además cambia lo que se puede hacer, no cuándo.
- **El mapeo al formulario de la AEPD**, por lo dicho arriba: hace falta el formulario como fuente primaria.
- **El art. 33.5**, el deber de documentar toda violación. Es permanente y sin cadencia, o sea una `continua`, y entra cuando se escriba con su barrido completo del artículo.
