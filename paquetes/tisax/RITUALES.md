# Rituales y cadencias del paquete tisax

**Esto no es el catalogo con el que se evalua.** Ni una palabra de este fichero ni
del `paquete.json` reproduce texto de los cuestionarios. Los cinco relojes de este
paquete son **rituales de plazum**, con un intervalo que ponemos nosotros, el
argumento escrito al lado y las instrucciones para moverlo.

**Y ninguno dice a que punto del catalogo sirve, a proposito.** Es estrato
referencial: el catalogo lo aporta el cliente, y su numeracion no se puede
verificar desde aqui sin abrirlo. El anclaje lo pone el cliente en su instancia.
El hueco esta contado en `docs/hallazgos-censo-a.md`: **5 de 5**.

Este fichero es la **fuente de los casos dorados** de `pruebas/`. Si el motor y un
dorado discrepan, gana el dorado.

## 0. El unico dato externo que este paquete usa, y su verificacion

**La validez de una evaluacion es de tres anos.** No sale del catalogo, que no se
puede leer aqui: sale del proceso publico que ENX describe en su propia pagina
sobre TISAX. **Verificado el 03-09-2026** en `enx.com`, seccion en ingles de
TISAX, donde se describe la validez de tres anos de las evaluaciones reconocidas.

De ese dato, y de nada mas, cuelga el ritual 2.1. Los otros cuatro se sostienen
sobre argumentos propios que no dependen de el.

Lo que ese dato NO autoriza a escribir: una cadencia con `origen_del_intervalo`
distinto de `propuesto`. La validez de tres anos es un hecho del proceso de ENX,
no un intervalo que el catalogo imponga al obligado, y el ritual 2.1 no es esa
validez: es **la decision de plazum de arrancar la reevaluacion doce meses antes
de que caduque**, que es un numero nuestro.

## 1. Regla de computo comun

- **Meses de fecha a fecha, con recorte al ultimo dia del mes destino.** Desde un
  29 de febrero, veinticuatro meses caen en el 28 si el ano destino no es
  bisiesto. Desde un 31 de enero, doce meses caen en un 31 de enero sin recortar.
- **Cierre al final del dia**, 23:59:59 en la zona del calendario.
- **Traslado: ninguno.** Si cae en sabado o domingo, se queda ahi.
- **Dias habiles**, cuando un ritual los use: a partir del dia siguiente al hecho,
  saltando sabados y domingos. En la version 1 del motor el calendario de los
  dorados solo excluye fines de semana.
- **El ciclo arranca en el hecho registrado**, y la ocurrencia n vence a los
  n*intervalo de ese hecho.

## 2. Los cinco rituales

### 2.1 `tisax.ritual.reevaluacion_de_la_etiqueta`, cada 24 meses

Arranca en `obtencion_de_la_etiqueta`. Veinticuatro meses desde la obtencion, o
sea **doce antes de que caduque**, porque la validez publicada es de tres anos y
una reevaluacion se planifica, se contrata y se ejecuta. Arrancar mas tarde deja a
la cadena de suministro sin etiqueta valida durante la transicion; arrancar antes
tira un ano de vigencia ya pagada.

Escalado: responsable de seguridad 90 dias antes, direccion 30 dias antes.

**Ojo con el ciclo largo**: la segunda ocurrencia cae a los 48 meses de la
obtencion, o sea DESPUES de que la etiqueta original caducara. Eso es correcto y
es lo que hace el motor: el ciclo se reinstancia sobre el hecho registrado, y
cuando se obtiene una etiqueta nueva se registra el hecho nuevo y el reloj se
reancla. El dorado de segunda vuelta esta escrito precisamente para dejarlo a la
vista.

### 2.2 `tisax.ritual.revision_del_alcance`, cada 12 meses

Arranca en `ultima_revision_del_alcance`. Doce meses porque el alcance lo forman
sedes y procesos, y esos se mueven con el ejercicio. Una etiqueta que cubre un
perimetro que dejo de existir es peor que no tenerla, porque el cliente de la
cadena se fia de ella.

### 2.3 `tisax.ritual.prueba_de_recuperacion`, cada 12 meses

Arranca en `ultima_prueba_de_recuperacion`. Doce meses porque antes se prueba el
mismo escenario y despues se prueba un plan que ya no describe la infraestructura.

### 2.4 `tisax.ritual.formacion_y_concienciacion`, cada 12 meses

Arranca en `ultima_formacion_de_concienciacion`. Doce meses porque la formacion se
ancla a la plantilla y la plantilla se renueva por ejercicio, y porque la
comprobacion habitual (cruzar la lista de asistentes con la de personal) solo
significa algo si cubre un ejercicio entero.

### 2.5 `tisax.ritual.actualizacion_del_registro_de_alcance`, 10 dias habiles

Es un **plazo**, no una cadencia: arranca en `cambio_en_el_alcance_de_la_etiqueta`
y vence a los 10 dias habiles, contados a partir del dia siguiente, al final del
ultimo dia habil y sin traslado.

Por que 10 dias habiles y no 2 ni 30: dos semanas de trabajo es lo que se tarda en
reunir el dato del cambio (que sede, que proceso, quien responde) sin una persona
dedicada a ello, y es lo bastante corto para que el registro no acumule mas de un
cambio pendiente a la vez. Con dos dias el registro se actualiza a medias y hay
que volver; con treinta, un cambio de sede puede llegar a la revision anual del
alcance sin que nadie lo haya anotado, que es exactamente lo que 2.2 existe para
evitar.

**Esto NO es una notificacion a nadie de fuera.** Es actualizar un registro
propio. Un ritual que sacara un aviso fuera de la organizacion seria una
obligacion `notificatoria`, y una notificatoria exige decir a quien alcanza y de
donde sale que no alcanza a los demas. Eso no se puede contestar sin el catalogo,
asi que no se escribe.

## 3. Lo que NO es un ritual de este paquete

- **La revision de los requisitos de proteccion de prototipos.** Su aplicabilidad
  depende del nivel de proteccion contratado, que es un dato del alcance que este
  paquete todavia no pregunta.
- **La revision de proveedores de la cadena.** Esta escrita en el paquete
  `pci-dss` con el mismo argumento (la evidencia del proveedor es anual), y
  duplicarla aqui no anadiria nada.
- **La revision de la clasificacion de la informacion.** El intervalo defendible
  sale del catalogo, que no se puede leer. Sin argumento propio, no hay numero.
- **La gestion del incidente que afecta a un cliente de la cadena.** Es una
  notificatoria, y su umbral no se puede escribir sin la copia. Un umbral escrito
  de menos no cuesta horas: provoca una actuacion indebida ante un tercero, y eso
  no se deshace.
- **La vigencia de la etiqueta en si.** Es un hecho del proceso de ENX y una fecha
  del cliente, no una cadencia: se modela con el objeto `Certificado` del nucleo,
  con la fecha real de su etiqueta.
