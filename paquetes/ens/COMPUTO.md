# Como se cuentan los plazos del ENS en este paquete

Este fichero es la **fuente de los casos dorados** de `pruebas/`. Cada dorado
cita el apartado de aqui del que sale su fecha esperada, junto con el texto legal
que fija el periodo. Si el motor y un dorado discrepan, gana el dorado: se
arregla el motor, no el caso.

El texto legal dice cada cuanto. No siempre dice a que hora vence ni que pasa si
el vencimiento cae en domingo. Esas dos decisiones estan aqui, escritas, con la
norma en la que se apoyan, para que se puedan discutir en vez de quedar
escondidas en el codigo.

## 1. Periodos en meses o anos: de fecha a fecha

Los plazos de este paquete expresados en meses o anos se cuentan **de fecha a
fecha**. Cuando en el mes de vencimiento no existe el dia inicial, el plazo
expira el ultimo dia de ese mes. Un ciclo que arranca un 29 de febrero vence el
28 de febrero si el ano destino no es bisiesto.

Fuente: **articulo 5.1 del Codigo Civil**, que fija el computo civil general
("si los plazos estuviesen fijados por meses o anos, se computaran de fecha a
fecha" y "cuando en el mes del vencimiento no hubiera dia equivalente al inicial
del computo, se entendera que el plazo expira el ultimo del mes"). La regla
paralela para procedimientos administrativos esta en el articulo 30.4 de la Ley
39/2015.

## 2. Cierre al final del dia

Un plazo fijado en meses o anos vence **al final del dia** que le corresponde,
a las 23:59:59 de la zona del calendario aplicable, no a la hora en que se
cumplio el hito anterior.

Fuente: el computo por meses y anos opera sobre dias completos, no sobre
instantes, asi que el ultimo dia se agota entero. **Esta es una decision de
dutiq, no una cita**: el RD 311/2022 no dice a que hora vence la auditoria
bienal. La alternativa (cierre en el instante exacto del hito anterior) produce
vencimientos no monotonos, que fue un hallazgo del motor de ventana y esta
documentado en `nucleo/ventana/calendario.go`.

Un plazo fijado en horas si vence en el **instante exacto**: las horas son
tiempo absoluto transcurrido.

## 3. Los dias inhabiles no mueven estos plazos

Si un vencimiento de este paquete cae en sabado, domingo o festivo, **se queda
ahi**. No se traslada al siguiente dia habil.

Fuente: **articulo 5.2 del Codigo Civil** ("en el computo civil de los plazos no
se excluyen los dias inhabiles"). La regla contraria, el articulo 30.5 de la Ley
39/2015, alcanza a los plazos de los procedimientos administrativos frente a un
interesado, y las obligaciones de este paquete no lo son: son deberes propios de
la entidad titular del sistema, no tramites en un expediente.

Consecuencia comprobable: el plazo de veinticuatro meses de la disposicion
transitoria unica vencio el **domingo 5 de mayo de 2024**, y ese es el dia que
calcula el motor.

Si tu asesoria juridica sostiene la lectura contraria, se cambia
`regimen.traslado` a `siguiente_habil` en el paquete y los dorados lo reflejan.
No hay que tocar codigo.

## 4. La notificacion inmediata al CCN

La ITS de Notificacion de Incidentes, apartado IV.3, ordena notificar los
incidentes de impacto ALTO, MUY ALTO o CRITICO **en el momento en que se
produzcan**. No concede ninguna demora, asi que en este paquete se declara como
un plazo de **cero horas** desde la calificacion del incidente: vence en el
mismo instante en que se sabe que es ALTO o superior.

Contraste que conviene ver junto: el mismo incidente, si afecta a datos
personales, dispara ademas el articulo 33 del RGPD, que si concede 72 horas.
Dos relojes, dos fuentes, el mismo hecho. La propia ITS lo dice en su apartado
X.2, anadido con efectos desde el 25 de mayo de 2018.

Si prefieres la lectura de que un plazo sin cifra es un plazo indeterminado, el
formato admite `"limite": "indeterminado"` y el motor lo marca como *sin plazo
legal*, midiendo el tiempo transcurrido en vez de una fecha. Hoy el ejecutor de
dorados no sabe comprobar esa variante, asi que el paquete usa la lectura de
cero horas y lo dice aqui en vez de esconderlo.

## 4 bis. Las cadencias que el ENS NO fija

Varias medidas del anexo II ordenan hacer algo "de forma periodica" sin decir
cada cuanto: la revision de permisos de acceso (op.acc.4.4), las pruebas del plan
de continuidad (op.cont.3), la periodicidad de las copias de seguridad
(mp.info.6.1, que remite expresamente a la normativa interna de la organizacion)
o la verificacion de la configuracion (op.exp.3.r1.2).

**Esas obligaciones se declaran sin reloj, a proposito.** Inventar un numero
seria fabricar derecho. El periodo lo pone la politica del cliente, que es lo que
la norma dice, y entonces el reloj vive en el paquete propio del cliente y no
aqui. Un reloj de dutiq con una cifra elegida por dutiq es legitimo cuando se
llama ritual y se dice (asi funciona el paquete `iso27001`), pero no cuando se
disfraza de plazo legal dentro de un paquete transcrito.

## 5. Lo que este paquete todavia NO puede expresar

Tres reglas del ENS existen, se han leido y **no caben en el formato de paquete
de hoy**. No estan simuladas ni maquilladas: estan nombradas aqui.

1. **La extension de tres meses por fuerza mayor** del articulo 31.1, parrafo 3.
   El texto esta transcrito dentro de la obligacion
   `ens.art31.auditoria_ordinaria`, pero el reloj no la aplica. El motor sabe
   hacerlo (`ventana.Prorroga`), lo que falta es un campo en `Temporalidad` para
   declararla desde el paquete.
2. **El reinicio del computo por auditoria extraordinaria** del articulo 31.1,
   parrafo 2: la auditoria extraordinaria fija la nueva fecha de computo de los
   dos anos. Hoy se resuelve en la practica registrando la extraordinaria como
   `fecha_ultima_auditoria`, que es exactamente lo que dice la norma, pero el
   paquete no puede declarar esa regla: la tiene que conocer quien registra el
   hecho.
3. **La aplicabilidad por categoria.** El ENS entero se modula por la categoria
   del sistema: la autoevaluacion bienal alcanza a BASICA y la certificacion
   bienal a MEDIA y ALTA. El nucleo tiene un motor de aplicabilidad en Datalog
   con agregacion, probado, pero `Paquete` no tiene donde declarar sus reglas,
   asi que hoy las dos obligaciones se muestran a todo el mundo y es el texto
   transcrito el que dice a quien alcanza cada una.

## 6. Una nota de vigencia que conviene leer

Las tres instrucciones tecnicas de seguridad que este paquete transcribe se
dictaron bajo el **RD 3/2010**, derogado por el RD 311/2022. Siguen **vigentes**
(el BOE no las marca como derogadas) porque la disposicion derogatoria unica del
RD 311/2022 solo alcanza a lo que se le oponga, y no se les opone. Lo que si
esta desfasado son sus **remisiones internas**: cuando la ITS de Conformidad
habla del "articulo 34 y el anexo III del Real Decreto 3/2010", hoy hay que leer
el articulo 31 y el anexo III del RD 311/2022. El paquete transcribe el texto tal
y como esta publicado, sin corregirlo, y avisa aqui de la lectura.

Fechas de entrada en vigor usadas en el paquete, tomadas del BOE:

| Norma | BOE | Vigor |
|---|---|---|
| RD 311/2022, ENS | BOE-A-2022-7191 | 5 de mayo de 2022 |
| ITS de Conformidad con el ENS | BOE-A-2016-10109 | 3 de noviembre de 2016 |
| ITS de Informe del Estado de la Seguridad | BOE-A-2016-10108 | 22 de noviembre de 2016 |
| ITS de Notificacion de Incidentes | BOE-A-2018-5370 | 20 de abril de 2018 |
