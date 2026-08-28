# nis2-tecnica — Reglamento de Ejecución (UE) 2024/2690

**Qué es.** El reglamento de ejecución que aterriza el artículo 21.2 de NIS2 en requisitos técnicos y metodológicos concretos. Es texto del DOUE, así que se **transcribe entero** (estrato transcrito, Decisión 2011/833/UE).

**A quién alcanza, y esto es lo primero que hay que leer.** NO es el ámbito entero de NIS2. Su artículo 1 fija una lista **cerrada de once tipos** de entidad: proveedores de servicios de DNS, registros de nombres de dominio de primer nivel, proveedores de servicios de computación en nube, de servicios de centros de datos, de redes de distribución de contenidos, de servicios gestionados, de servicios de seguridad gestionados, de mercados en línea, de motores de búsqueda en línea y de plataformas de servicios de redes sociales, y prestadores de servicios de confianza.

**Una entidad esencial o importante que no sea de esos once tipos no entra aquí.** Es el hecho `papel_nis2_tecnica(E, "entidad_pertinente")`, y sin él este paquete no enciende ni un reloj. El porqué de tratarlo así, en `docs/decisiones.md` D-14: un mapeo no es un ámbito.

**Desde cuándo.** 7 de noviembre de 2024.

## Los 47 relojes, y de quién es cada número

Esta es la pregunta que decide si se puede discutir una fecha con un inspector, así que va antes que nada:

| origen | cuántos | qué significa |
|---|---|---|
| `suelo_legal` | **3** | la norma pone un mínimo de frecuencia y lo dice con palabras (*«al menos una vez al año»*, *«como mínimo anualmente»*). El número es de la norma; sólo se puede apretar |
| `propuesto` | **44** | el anexo dice *«a intervalos planificados»*, *«periódicamente»* o *«de forma periódica`* y **no da número**. El número lo pone plazum, y viaja con su justificación, sus fuentes y sus instrucciones de uso |

Los tres con número son los puntos **1.1.2** (política de seguridad, revisión por el órgano de dirección), **2.1.4** (evaluación de riesgos y plan de tratamiento) y **10.1.3** (asignación de personal a roles). Los tres dicen «al menos», los tres son anuales, y **los tres son el ancla del resto**: buena parte de las 44 justificaciones se apoyan en uno de ellos, porque son los únicos ritmos que la norma fija por sí misma.

Cada una de las 44 trae, en su `temporalidad`:

- `justificacion_del_intervalo`: por qué ese número y no otro, **empezando por de qué verbo cuelga**. Un punto con tres verbos admite tres relojes distintos y sólo uno es el que la norma exige.
- `fuentes_del_intervalo`: los puntos del propio anexo en los que se apoya el argumento. Si un intervalo se apoya en otro intervalo **nuestro**, la justificación lo dice y nombra el punto.
- `cuando_cambiarlo`: **una condición para acortarlo y una para alargarlo**, cada una con el supuesto que la hace cierta. Es el campo que convierte un defecto en un defecto adaptable.

## 47 obligaciones, 5 fechas al año

El número de obligaciones asusta y es el de la norma, no el nuestro: sumarlas es justamente lo que nadie más hace. Pero **las obligaciones no son ceremonias**. Agrupadas por cadencia quedan así:

| cadencia | obligaciones | qué es, en la práctica |
|---|---|---|
| **P1M** | 1 | verificar la integridad de las copias (4.2.3). Automatizable: en cuanto lo hace la herramienta de respaldo, deja de ser una cita |
| **P3M** | 6 | el trimestre operativo: tendencias en registros, informes de nivel de servicio, cobertura antimalware, exploración de vulnerabilidades, cuentas privilegiadas e inventario de activos |
| **P6M** | 10 | el semestre de control: cumplimiento y su informe al órgano de dirección, prueba de respuesta ante incidentes, revisiones posincidente, configuraciones, seguridad de red, sensibilización, derechos de acceso, identidades y acceso físico |
| **P12M** | **28** (25 propuestos + los 3 con número) | la temporada anual. Aquí caen las tres que fija la norma y veinticinco más |
| **P24M** | 2 | revisión independiente (2.3.4) y procedimientos disciplinarios (10.4.2) |

**Veintiocho obligaciones de nueve capítulos distintos vencen en el mismo ciclo anual**, y varias comparten literalmente el ejercicio: la prueba del plan de continuidad (4.1.4), la de recuperación de copias (4.2.6) y la de los planes de crisis (4.3.4) son **una temporada de pruebas, no tres**, y así está escrito en las tres justificaciones. Lo mismo pasa con la revisión de identidades (11.5.4) y la de derechos de acceso (11.2.3), que son la misma gente mirando la misma lista.

Ése es el trabajo que este paquete hace y que un listado de controles no hace: **decir cuántas veces al año hay que sentarse, no cuántas casillas hay**.

## Lo que este paquete NO hace todavía

- **Los disparadores por evento.** Casi todos los puntos de revisión traen además *«o cuando se produzcan incidentes significativos o cambios significativos en las operaciones o los riesgos»*. Eso es un reloj distinto (una `observacion`, no una `periodica`) y **no está escrito**. Mientras no lo esté, el calendario enseña la cita periódica y no la reapertura por evento, y las justificaciones lo dicen cuando se apoyan en ella.
- **El artículo 4 y la evaluación trimestral de incidentes recurrentes** del punto 3.4.2, letra b), que es el **cuarto** punto del anexo con número (`trimestral`) y va con los artículos 3 a 14, no con el anexo.
- **Los artículos 3 a 14**, que son los umbrales de incidente significativo por tipo de entidad. Son el otro medio paquete.
- **El punto 3.2.2** queda deliberadamente fuera de los relojes: dice que la supervisión se llevará a cabo *«bien de forma continua bien a intervalos periódicos, en función de las capacidades operativas»*. La norma delega expresamente el modo en la capacidad de cada entidad, así que poner ahí un número sería inventarse una obligación que el texto no impone.

## Cómo se escribió, y cómo se comprueba

El `texto_legal` de las 47 **no se ha tecleado**: se extrae de la instantánea con huella de Cellar (`corpus-vigilancia/ue-32024r2690`, CELEX 32024R2690) y se verifica contra EUR-Lex. Por eso el punto 2.1.4 dice «riegos» donde debería decir «riesgos»: **es la errata del DOUE y se transcribe tal cual**, porque el paquete transcribe la norma y no la corrige.

Cada reloj lleva **tres casos dorados** (ciclo normal, borde de calendario con recorte real al último día del mes, y segunda vuelta) que se ejecutan contra el motor en cada `./comprobar.sh`. Si el motor y el caso discrepan, **gana el caso**.
