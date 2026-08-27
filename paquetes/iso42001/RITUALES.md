# Rituales y cadencias del paquete iso42001

**Esto no es ISO.** ISO/IEC 42001:2023 exige auditar, revisar, apreciar riesgos y
evaluar el impacto de los sistemas de IA de forma planificada, y deja el
intervalo concreto en manos de la organización. No hay ningún número de meses en
la norma. Los números de este fichero los pone plazum, son un valor de partida
razonable y auditable, y el cliente los cambia en su copia del paquete sin tocar
una línea de código.

Ningún párrafo de este fichero reproduce texto de la norma. Los títulos que
aparecen en `paquete.json` son identificadores y etiquetas cortas nuestras para
que se sepa de qué cláusula o de qué categoría del anexo A se habla, nunca el
enunciado normativo.

Este fichero es la **fuente de los casos dorados** de `pruebas/`. Cada dorado
cita la sección de aquí de la que sale su fecha esperada. Si el motor y un dorado
discrepan, gana el dorado: se arregla el motor, no el caso.

## 1. Regla de cómputo común

Es la misma que la del paquete `iso27001`, y lo es a propósito: quien lleva los
dos sistemas de gestión no debería tener que aprender dos aritméticas.

- **Meses de fecha a fecha, con recorte al último día del mes destino.** Si el
  ciclo arranca un 29 de febrero y el año destino no es bisiesto, vence el 28. El
  recorte solo actúa cuando el día de origen no existe en el mes destino, y
  **solo baja, nunca sube**: un ciclo que arranca el 28 vence el 28 aunque el año
  destino sea bisiesto.
- **El recorte no se arrastra.** La ocurrencia n se cuenta desde el arranque
  registrado, no desde la ocurrencia anterior ya recortada. Arrancando el 29 de
  febrero de 2024, la ocurrencia 1 cae el 28 de febrero de 2025 y la ocurrencia 4
  vuelve al 29 de febrero de 2028, que sí es bisiesto. Hay un caso dorado para
  esto, porque un motor que encadenara ocurrencia sobre ocurrencia dejaría el
  recorte pegado para siempre.
- **Cierre al final del día** (23:59:59 en la zona del calendario). Un ritual
  vence el día entero, no en el minuto en que se cerró el anterior.
- **Traslado: ninguno.** Si el vencimiento cae en sábado, domingo o festivo, se
  queda ahí. Un ritual interno no es un plazo administrativo, así que no hereda
  la regla de traslado del artículo 30.5 de la Ley 39/2015.
- **Días hábiles**, cuando un ritual los use: se cuentan a partir del día
  siguiente al hecho, saltando sábados, domingos y los festivos del calendario
  aplicable. En la versión 1 del motor, el calendario de los dorados solo excluye
  fines de semana; los calendarios de festivos llegan con su propia pieza del
  corpus.
- **El ciclo arranca en el hecho, no en la fecha teórica.** La ocurrencia
  siguiente se cuenta desde lo que se hizo de verdad y quedó registrado, no desde
  cuándo tocaba hacerlo.

## 2. Los siete rituales

### 2.1 `iso42001.ritual.evaluacion_impacto_ia`, cada 12 meses

**Este es el que no tiene equivalente en la 27001, y es el que importa.** Sirve a
las cláusulas 6.1.4 y 8.4 (evaluación de impacto del sistema de IA). Arranca en
`ultima_evaluacion_impacto_ia`, la fecha de cierre de la anterior. Vence a los 12
meses, al final del día.

Escalado: aviso al responsable de IA 90 días antes y a dirección 30 días antes.
Noventa y no sesenta porque una evaluación de impacto de un sistema de IA no la
cierra una persona en una tarde: hay que reunir a quien conoce el modelo, a quien
conoce el dato y a quien responde por el efecto sobre las personas.

Por qué 12 meses: es el ciclo que hace que la evaluación llegue fresca a la
revisión por la dirección, y es el intervalo que un auditor del AIMS espera ver
sin preguntar. Un sistema que cambia más deprisa lo baja en su copia del paquete.

### 2.2 `iso42001.ritual.apreciacion_riesgos_ia`, cada 12 meses

Sirve a las cláusulas 6.1.2 y 8.2. Arranca en `ultima_apreciacion_riesgos_ia`.
Escalado: responsable de IA 60 días antes.

Va separada de la evaluación de impacto **a propósito**, y es una distinción que
la 42001 hace y que se pierde si se juntan: el riesgo mira a la organización, el
impacto mira a las personas y a la sociedad. Fundirlas en un solo ritual es el
atajo que hace que la segunda no se haga nunca.

### 2.3 `iso42001.ritual.auditoria_interna`, cada 12 meses

Sirve a la cláusula 9.2.2 (programa de auditoría interna). Arranca en
`ultima_auditoria_interna_aims`, la fecha de cierre de la anterior. Escalado:
responsable de IA 60 días antes, dirección 30 días antes.

### 2.4 `iso42001.ritual.revision_direccion`, cada 12 meses

Sirve a la cláusula 9.3.1. Arranca en `ultima_revision_direccion_aims`, la fecha
del acta. Escalado: dirección 30 días antes.

### 2.5 `iso42001.ritual.revision_declaracion_aplicabilidad`, cada 12 meses

Sirve a la cláusula 6.1.3 (declaración de aplicabilidad del AIMS). Arranca en
`ultima_revision_soa_aims`. Escalado: responsable de IA 30 días antes.

### 2.6 `iso42001.ritual.revision_politica_ia`, cada 12 meses

Sirve a la cláusula 5.2 (política de IA). Arranca en
`ultima_revision_politica_ia`. Escalado: dirección 30 días antes.

Existe como ritual propio y no como parte de la revisión por la dirección porque
la política de IA es el documento que un cliente, un regulador o un comprador
pide primero, y el que más se queda viejo sin que nadie lo note.

### 2.7 `iso42001.ritual.plan_accion_no_conformidad`, 10 días hábiles

Sirve a la cláusula 10.2 (no conformidad y acción correctiva), que exige
reaccionar y no dice en cuánto. Arranca en `deteccion_no_conformidad_aims` y
vence a los 10 días hábiles, al final del día. Escalado: responsable de IA a los
5 días.

Es el único ritual del paquete que cuenta en días hábiles, y trae el caso dorado
que enseña por qué el calendario es un dato del corpus y no una constante del
código: una no conformidad detectada el 24 de diciembre vence el 7 de enero con
el calendario de hoy, que solo excluye fines de semana, y se moverá en cuanto
entre un calendario de festivos.

## 3. Lo que estos rituales NO cubren

- **La certificación y sus ciclos de vigilancia** (auditoría de seguimiento anual
  y recertificación a los tres años). No están aquí porque no salen de la norma
  sino del esquema de acreditación de cada entidad de certificación, y ponerles
  un número por nuestra cuenta sería inventar. Llegan con la pieza de
  certificados del núcleo, que ya existe.
- **El disparador por cambio sustancial** (un modelo reentrenado, un caso de uso
  nuevo, un proveedor de modelo distinto). Es la familia D del censo y necesita
  el motor de eventos, no el de ventana. Es, con diferencia, el disparador que
  más se va a usar en un AIMS.
