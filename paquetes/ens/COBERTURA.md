# Cobertura del paquete ens: lo que hay y lo que falta, con nombre y apellidos

Un porcentaje no dice nada. Esto dice que se transcribio, que no, y por que.
Version del paquete: 0.2.0. Texto contrastado con el consolidado del BOE el
25 de agosto de 2026.

## Transcrito, con obligacion propia y cita al apartado exacto

| Bloque | Obligaciones |
|---|---|
| Capitulo I, arts. 2 y 3 | 3 |
| Capitulo III, arts. 12 a 28 | 32 |
| Capitulo IV, arts. 31 y 33 | 6 |
| Capitulo V, arts. 36 a 38 | 4 |
| Capitulo VII, arts. 40 y 41 | 3 |
| Anexo I, reevaluacion de la categoria | 1 |
| Anexo II, las 73 medidas de seguridad | 73 |
| Disposicion transitoria unica | 1 |
| ITS de Informe del Estado de la Seguridad | 2 |
| ITS de Notificacion de Incidentes | 3 |
| ITS de Conformidad con el ENS | 4 |
| **Total** | **132** |

Ocho de esas 132 llevan reloj, y cada reloj lleva tres casos dorados derivados
del texto legal, con su cita: 24 dorados que se recalculan contra el motor en
cada ejecucion de los tests.

El anexo II esta completo en su **catalogo de medidas**: las 73, cada una con su
identificador (`org.1` a `mp.s.4`), su numeracion del anexo y el texto exacto de
sus requisitos base. Se transcriben del XML del BOE troceando por medida, no a
mano.

## Sin mapear, nominalmente

### Los refuerzos del anexo II

Ninguno de los refuerzos (R1, R2, R3...) esta transcrito. Cada medida los lleva
y se activan segun el nivel de la dimension de seguridad afectada. El formato de
paquete de hoy no tiene donde declarar "esta obligacion solo aplica si la
dimension de confidencialidad es ALTA", asi que transcribirlos produciria un
listado que se le ensena entero a todo el mundo, incluida una entidad de
categoria BASICA a la que no le alcanzan. Eso es peor que no tenerlos. Entran
cuando entren las reglas de aplicabilidad en el paquete.

Cada obligacion de medida lo dice en su cita: **requisitos base sin refuerzos**.

### La tabla de aplicacion por nivel del anexo II, apartado 2.4

Sin mapear, por lo mismo. Es la tabla que dice, para cada medida, si aplica y con
que refuerzos en cada nivel de cada dimension. Es el corazon operativo del ENS y
hoy no cabe en el formato.

### Anexo III y anexo IV

Sin mapear. El anexo III (auditoria de la seguridad) es metodologia de auditoria
mas que obligacion con verbo exigible propio, y el anexo IV es el glosario.

### Articulos sin obligacion propia

Arts. 1, 4, 5 a 11, 19.2, 19.3, 29, 30, 31.2, 31.3, 31.6, 31.7, 32, 33.1, 33.3 a
33.6, 34, 35 y 39, y las disposiciones adicionales primera a tercera, las
disposiciones finales y la derogatoria.

Motivo, por grupos:

- **Dirigidos a otro sujeto**: los arts. 29 a 35 en su mayor parte, el 39 y las
  adicionales imponen deberes al CCN, a la Secretaria General de Administracion
  Digital o a la Comision Sectorial, no a la entidad titular del sistema.
  Meterlos como obligaciones del cliente seria mentirle. El deber operativo que
  si le alcanza por el art. 32 esta transcrito, pero desde su fuente real, que
  es la ITS de Informe del Estado de la Seguridad.
- **Sin verbo exigible**: los arts. 1, 4, 5 a 11 son objeto, definiciones y
  principios basicos. Los principios se implantan a traves de los requisitos
  minimos del capitulo III, que si estan transcritos uno a uno.
- **Facultades, no deberes**: art. 31.6 (suspender el tratamiento en categoria
  ALTA) y art. 31.7 (requerir los informes) son potestades.

### Tres obligaciones sin recurso observable

`ens.art27.mejora_continua`, `ens.art36.ciclo_de_vida` y
`ens.art37.mecanismos_de_control`. No hay conector que las mida: se cierran con
evidencia documental o con un ritual. Estan contadas como no automatizadas en la
salida de cobertura del motor, no escondidas en un porcentaje.

## Lo que falta y no es texto

1. **Reglas de aplicabilidad por categoria y por nivel.** El nucleo tiene el
   motor (Datalog con agregacion y cierre transitivo, probado) pero `Paquete` no
   tiene donde declarar sus reglas. Sin esto, un sistema de categoria BASICA ve
   las 73 medidas y las dos vias de conformidad. Es el hueco numero uno de este
   paquete.
2. **La prorroga de tres meses del art. 31.1 y el reinicio del computo por
   auditoria extraordinaria.** Detalle en `COMPUTO.md`, apartado 5.
3. **Las cadencias que el ENS delega en la organizacion.** Varias medidas piden
   hacer algo "de forma periodica" sin decir cada cuanto (op.acc.4.4 revision de
   permisos, op.cont.3 pruebas del plan de continuidad, mp.info.6.1 copias de
   seguridad, op.exp.3 verificacion de la configuracion). No son relojes legales
   y no se han inventado: se declaran como obligaciones sin reloj y el periodo lo
   pone la normativa interna del cliente, como dice la propia norma.
4. **Equivalencias con ISO/IEC 27001** en formato OSCAL con su lista de huecos.
   Es la casilla siguiente de la etapa 3.
5. **Revision juridica externa.** Este paquete lo ha escrito una sola persona
   contra el texto del BOE. La revision por despacho o consultor esta en la
   etapa 3 y hasta entonces esto no esta firmado por nadie mas.

## Criterio de autoria que conviene poder discutir

- **Una obligacion por medida del anexo II**, no una por cada requisito
  numerado dentro de ella. Es como se construye la Declaracion de Aplicabilidad
  del art. 28.2, que es el documento que un auditor pide primero.
- **La clase de implantacion y los recursos observables de cada medida son
  criterio de plazum**, no del BOE. Estan a la vista en el `paquete.json`, medida
  a medida, para poder discutirlos uno a uno.
- **El art. 31.1 se parte en dos obligaciones**: la ordinaria (parrafos 1 y 3,
  la regla bienal y su extension) y la extraordinaria (parrafo 2), porque son dos
  verbos exigibles con disparadores distintos. La cita de cada una dice de que
  parrafos sale su texto.
