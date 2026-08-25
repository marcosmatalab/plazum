# Rituales y cadencias del paquete iso27001

**Esto no es ISO.** ISO/IEC 27001:2022 exige auditar, revisar y apreciar riesgos
de forma planificada, y deja el intervalo concreto en manos de la organizacion.
No hay ningun numero de meses en la norma. Los numeros de este fichero los pone
dutiq, son un valor de partida razonable y auditable, y el cliente los cambia en
su copia del paquete sin tocar una linea de codigo.

Ningun parrafo de este fichero reproduce texto de la norma. Los titulos que
aparecen en `paquete.json` son identificadores y etiquetas cortas nuestras para
que se sepa de que control se habla, nunca el enunciado del control.

Este fichero es la **fuente de los casos dorados** de `pruebas/`. Cada dorado
cita la seccion de aqui de la que sale su fecha esperada. Si el motor y un
dorado discrepan, gana el dorado: se arregla el motor, no el caso.

## 1. Regla de computo comun

Salvo que un ritual diga otra cosa:

- **Meses de fecha a fecha, con recorte al ultimo dia del mes destino.** Si el
  ciclo arranca un 29 de febrero y el ano destino no es bisiesto, vence el 28.
  El recorte solo actua cuando el dia de origen no existe en el mes destino.
- **Cierre al final del dia** (23:59:59 en la zona del calendario). Un ritual
  vence el dia entero, no en el minuto en que se cerro el anterior.
- **Traslado: ninguno.** Si el vencimiento cae en sabado, domingo o festivo, se
  queda ahi. Un ritual interno no es un plazo administrativo del articulo 30.5
  de la Ley 39/2015, asi que no hereda su regla de traslado.
- **Dias habiles**, cuando un ritual los use: se cuentan a partir del dia
  siguiente al hecho, saltando sabados, domingos y los festivos del calendario
  aplicable. En la version 1 del motor el calendario de los dorados solo excluye
  fines de semana; los calendarios de festivos llegan con su propia pieza del
  corpus.
- **El ciclo arranca en el hecho, no en la fecha teorica.** La ocurrencia
  siguiente se cuenta desde lo que se hizo de verdad y quedo registrado, no
  desde cuando tocaba hacerlo.

## 2. Los seis rituales

### 2.1 `iso27001.ritual.auditoria_interna`, cada 12 meses

Sirve al requisito 9.2.2 (programa de auditoria interna). Arranca en
`ultima_auditoria_interna`, la fecha de cierre de la auditoria anterior.
Vence a los 12 meses, al final del dia. La ocurrencia n vence a los 12*n meses.

Escalado: aviso al responsable de seguridad 60 dias antes y a direccion 30 dias
antes. Por que 12 meses: es el intervalo que espera cualquier entidad de
certificacion para cubrir el alcance completo entre auditorias externas, y el
que hace que la revision por la direccion tenga entradas frescas.

### 2.2 `iso27001.ritual.revision_direccion`, cada 12 meses

Sirve al requisito 9.3.1. Arranca en `ultima_revision_direccion`, la fecha del
acta anterior. Vence a los 12 meses, al final del dia.

### 2.3 `iso27001.ritual.apreciacion_riesgos`, cada 12 meses

Sirve al requisito 8.2. Arranca en `ultima_apreciacion_riesgos`. Vence a los 12
meses, al final del dia. ISO pide ademas repetirla cuando hay cambios
significativos: ese disparador por evento no es una cadencia y no esta en este
reloj, se declara como obligacion aparte cuando el motor de eventos lo soporte.

### 2.4 `iso27001.ritual.revision_declaracion_aplicabilidad`, cada 12 meses

Sirve al requisito 6.1.3. Arranca en `ultima_revision_soa`. Vence a los 12
meses, al final del dia. La declaracion de aplicabilidad es el documento que un
auditor pide primero, y el que mas rapido envejece cuando cambia el alcance.

### 2.5 `iso27001.ritual.revision_independiente`, cada 24 meses

Sirve al control A.5.35. Arranca en `ultima_revision_independiente`. Vence a los
24 meses, al final del dia. Por que 24 y no 12: la revision independiente es mas
cara que la interna y encaja con el ciclo de tres anos de la certificacion, de
forma que cae una vez entre la certificacion inicial y la recertificacion.

### 2.6 `iso27001.ritual.plan_accion_no_conformidad`, 10 dias habiles

Sirve al requisito 10.2. Es un plazo, no una cadencia: arranca en
`deteccion_no_conformidad` y vence a los 10 dias habiles, al final del ultimo
dia habil. Por que 10 dias habiles: es el plazo que las entidades de
certificacion suelen conceder para presentar el plan de accion tras una no
conformidad, asi que un cliente que lo cumpla internamente llega a tiempo a la
peticion externa.

## 3. Lo que NO es un ritual de este paquete

- El ciclo de certificacion (inicial, vigilancia anual, recertificacion a los
  tres anos) no sale de ISO/IEC 27001 sino de las reglas de acreditacion de la
  entidad certificadora. Se modela con el objeto `Certificado` del nucleo, con
  las fechas reales del certificado del cliente, y no con una cadencia inventada
  aqui.
- Las cadencias tecnicas (revision de accesos, pruebas de restauracion, analisis
  de vulnerabilidades) no estan en este paquete porque ISO no las cuantifica y
  porque su periodo depende del riesgo. Se declaran en el paquete propio del
  cliente.
