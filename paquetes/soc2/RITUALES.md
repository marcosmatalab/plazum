# Rituales y cadencias del paquete soc2

**Esto no son los criterios.** Ni una palabra de este fichero ni del
`paquete.json` reproduce texto del marco. Los cinco relojes de este paquete son
**rituales de plazum**, con un intervalo que ponemos nosotros, el argumento
escrito al lado y las instrucciones para moverlo. El cliente los cambia en su
copia del paquete sin tocar una linea de codigo.

**Y ninguno dice a que criterio sirve, a proposito.** Es estrato referencial: el
texto lo aporta el cliente con su copia licenciada, y la numeracion de los
criterios no se puede verificar desde aqui sin abrirla. Escribir uno de memoria
produciria un dato con la FORMA de lo verificable, que es justo lo que hace que
nadie vaya a comprobarlo. El anclaje lo pone el cliente en su instancia. El hueco
esta contado en `docs/hallazgos-censo-a.md`: **5 de 5**.

Este fichero es la **fuente de los casos dorados** de `pruebas/`. Si el motor y un
dorado discrepan, gana el dorado.

## 0. De donde sale el ritmo de este paquete

De una sola cosa, y conviene decirla antes que los numeros: **el periodo de
observacion**. Un informe de tipo II no describe un instante, describe un periodo
durante el cual los controles estuvieron operando, y quien lo examina muestrea
evidencia repartida por todo ese periodo. De ahi salen los dos unicos argumentos
que este paquete usa para poner un numero:

1. **Encadenamiento.** Dos informes consecutivos no pueden dejar hueco entre
   ellos, porque el cliente que los pide cubre ejercicios completos.
2. **Densidad de evidencia.** Un control que se ejecuta n veces dentro del periodo
   deja n puntos de muestreo, y un fallo sobre n puntos duele menos cuanto mayor
   es n.

Los dos son estructurales y se sostienen solos. Por eso dos de los cuatro
intervalos de este paquete **no declaran `fuentes_del_intervalo`**, y no es un
olvido: la pregunta de la pasada de coherencia es *«por que este argumento no
tiene fuente»*, y la respuesta aqui es que el argumento sale de la forma del
propio informe y no del criterio de nadie. Los otros dos si la declaran, porque
hablan de disciplinas con literatura publica.

## 1. Regla de computo comun

- **Meses de fecha a fecha, con recorte al ultimo dia del mes destino.** Desde un
  29 de febrero, doce meses caen en el 28 si el ano destino no es bisiesto. Desde
  un 31 de agosto, doce meses caen en un 31 de agosto sin recortar nada.
- **Cierre al final del dia**, 23:59:59 en la zona del calendario.
- **Traslado: ninguno.** Si cae en sabado o domingo, se queda ahi.
- **Dias habiles**, cuando un ritual los use: a partir del dia siguiente al hecho,
  saltando sabados y domingos. En la version 1 del motor el calendario de los
  dorados solo excluye fines de semana; los festivos llegan con su propia pieza.
- **El ciclo arranca en el hecho registrado**, y la ocurrencia n vence a los
  n*intervalo de ese hecho, no a un intervalo de la ocurrencia anterior.

## 2. Los cinco rituales

### 2.1 `soc2.ritual.cierre_del_periodo_de_observacion`, cada 12 meses

Arranca en `fin_del_ultimo_periodo_de_observacion`. Doce meses por el argumento 1
de la seccion 0: con doce, un solo informe cubre el ejercicio entero; con menos
hacen falta dos para decir lo mismo; con mas, el informe llega describiendo un
periodo mas viejo que el ejercicio que el cliente esta auditando cuando lo pide.

Sin `fuentes_del_intervalo`: el argumento es el encadenamiento, que es una
propiedad de la forma del informe.

Escalado: responsable de cumplimiento 60 dias antes, direccion 30 dias antes.

### 2.2 `soc2.ritual.revision_de_accesos`, cada 3 meses

Arranca en `ultima_revision_de_accesos`. Tres meses por el argumento 2: cuatro
puntos de evidencia en un periodo anual. Con uno solo, un fallo deja el periodo
entero sin evidencia y el informe sale con una excepcion.

### 2.3 `soc2.ritual.evaluacion_de_riesgos`, cada 12 meses

Arranca en `ultima_evaluacion_de_riesgos`. Doce meses porque es la entrada de 2.4
y de la delimitacion del periodo de 2.1, y las tres tienen que ir al mismo ritmo o
llega tarde a las otras dos.

### 2.4 `soc2.ritual.revision_por_la_direccion`, cada 12 meses

Arranca en `ultima_revision_por_la_direccion`. Doce meses porque se alimenta de
2.1 y de 2.3. Sin `fuentes_del_intervalo`, por lo mismo que 2.1: el argumento es
que sus entradas son anuales.

### 2.5 `soc2.ritual.remediacion_de_excepcion`, 20 dias habiles

Es un **plazo**, no una cadencia: arranca en `excepcion_identificada` y vence a
los 20 dias habiles, contados a partir del dia siguiente, al final del ultimo dia
habil y sin traslado.

Por que 20 dias habiles y no 10 ni 60: veinte habiles son cuatro semanas de
trabajo, que es lo que separa el hallazgo de la siguiente reunion mensual de
gobierno en casi cualquier organizacion, y lo que hace falta para determinar la
causa, acordar la correccion y dejarla documentada. Con diez, el plan se escribe
sin causa raiz y hay que rehacerlo. Con sesenta, la excepcion sobrevive al
siguiente muestreo del periodo y aparece dos veces en el mismo informe.

Por que habiles y no naturales: el trabajo lo hacen personas en horario de
oficina, y la cuenta tiene que significar lo mismo en agosto que en marzo.

## 3. Lo que NO es un ritual de este paquete

- **La prueba de recuperacion.** Esta escrita en el paquete `tisax`, donde la
  disponibilidad para la cadena de suministro le da un ancla mas concreta.
  Repetirla aqui seria la misma ceremonia con dos nombres.
- **La revision de la carta de compromisos con el cliente.** Su ritmo lo pone el
  contrato, no el marco ni plazum. Un intervalo nuestro encima del contrato de un
  cliente es una imposicion disfrazada de dato.
- **Las pruebas que ejecuta el examinador.** No son un ritual del obligado: las
  hace un tercero contratado. Meterlas en el calendario del cliente le pone una
  fila que no puede cerrar, y una fila que no se cierra ensena a ignorar el
  calendario.
- **La emision del informe.** Es un entregable de un tercero con su propio
  calendario contractual. Se modela con el objeto `Certificado` del nucleo, con
  las fechas reales, no con una cadencia inventada aqui.
