# rgpd — Reglamento (UE) 2016/679

Texto del DOUE, transcrito (estrato transcrito, Decisión 2011/833/UE). Fuente: instantánea con huella de Cellar, CELEX 32016R0679.

**A quién alcanza.** Al **responsable y al encargado** del tratamiento, los dos. Es el hecho `trata_datos_personales(E)`.

## Los dos relojes

| art. | qué | reloj | de quién es el número |
|---|---|---|---|
| 33.1 | notificar una violación de seguridad a la autoridad | plazo de **72 h** desde que se tiene constancia | de la norma (`fijado`) |
| 32.1.d | verificar, evaluar y valorar la eficacia de las medidas | **P12M** | **de plazum** (`propuesto`) |

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

## Lo que este paquete NO hace todavía

- **El resto del RGPD.** Están los dos relojes de arriba y nada más: no hay registro de actividades (art. 30), ni evaluación de impacto (art. 35), ni derechos del interesado (arts. 15 a 22), ni el plazo de un mes del art. 12.3.
- **La notificación al interesado** del art. 34, que es otro reloj (*«sin dilación indebida»*, sin número) y va con el 33.
