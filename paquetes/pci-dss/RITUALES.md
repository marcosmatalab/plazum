# Rituales y cadencias del paquete pci-dss

**Esto no es PCI DSS.** Ni una palabra de este fichero ni del `paquete.json`
reproduce texto del marco. Los siete relojes de este paquete son **rituales de
plazum**: ceremonias que un CISO reconoce, con un intervalo que ponemos nosotros,
con el argumento escrito al lado y con instrucciones de uso para moverlo. El
cliente los cambia en su copia del paquete sin tocar una linea de codigo.

**Y ninguno dice a que requisito sirve, a proposito.** PCI DSS es estrato
referencial: su texto lo aporta el cliente con su copia licenciada, y la
numeracion de sus requisitos no se puede verificar desde aqui sin abrirla.
Escribir un numero de requisito de memoria produciria un dato con la FORMA de lo
verificable, que es justo lo que hace que nadie vaya a comprobarlo. Ese anclaje
lo pone el cliente en su instancia, con su copia delante, y es una linea por
ritual. El hueco esta contado en `docs/hallazgos-censo-a.md`: **7 de 7**.

Este fichero es la **fuente de los casos dorados** de `pruebas/`. Cada dorado cita
la seccion de aqui de la que sale su fecha esperada. Si el motor y un dorado
discrepan, gana el dorado: se arregla el motor, no el caso.

## 1. Regla de computo comun

Salvo que un ritual diga otra cosa:

- **Meses de fecha a fecha, con recorte al ultimo dia del mes destino.** Si el
  ciclo arranca un 30 de noviembre y el mes destino es febrero, vence el 28 (o el
  29 en bisiesto). El recorte solo actua cuando el dia de origen no existe en el
  mes destino: desde un 31 de mayo, tres meses caen en un 31 de agosto sin
  recortar nada.
- **Cierre al final del dia**, 23:59:59 en la zona del calendario. Un ritual vence
  el dia entero.
- **Traslado: ninguno.** Si el vencimiento cae en sabado, domingo o festivo, se
  queda ahi. Un ritual interno no es un plazo administrativo, asi que no hereda
  ninguna regla de traslado.
- **Dias naturales**, cuando un ritual los use: se cuentan a partir del dia
  siguiente al hecho, sin saltar nada, y vence al final del ultimo.
- **El ciclo arranca en el hecho, no en la fecha teorica.** La ocurrencia
  siguiente se cuenta desde lo que se hizo de verdad y quedo registrado.
- **La ocurrencia n vence a los n*intervalo del hecho**, no a un intervalo de la
  ocurrencia anterior. Con un ciclo trimestral desde un 31 de mayo, la segunda
  ocurrencia son seis meses desde ese 31 de mayo.

## 2. Los siete rituales

### 2.1 `pcidss.ritual.revision_de_accesos`, cada 3 meses

Arranca en `ultima_revision_de_accesos`. Por que tres meses: el intervalo de la
revision ES la ventana durante la cual un permiso que sobra sigue vivo, asi que un
trimestre acota a 90 dias lo que se le escapa al proceso de altas y bajas. Doce
meses la convierten en un inventario historico y uno en un tramite que se firma
sin mirar.

Escalado: aviso al responsable de seguridad 30 dias antes.

### 2.2 `pcidss.ritual.analisis_de_vulnerabilidades`, cada 3 meses

Arranca en `ultimo_analisis_de_vulnerabilidades`. Por que tres meses: el intervalo
entre analisis es el tiempo maximo que una vulnerabilidad ya publicada puede vivir
sin que nadie la vea. Va al mismo ritmo que 2.1 a proposito, para que las dos
caigan en la misma ceremonia trimestral.

### 2.3 `pcidss.ritual.prueba_de_intrusion`, cada 12 meses

Arranca en `ultima_prueba_de_intrusion`. Por que doce meses: la prueba necesita un
objetivo estable y un presupuesto, y las dos cosas van por ejercicio.

### 2.4 `pcidss.ritual.revision_de_politicas`, cada 12 meses, con reapertura

Arranca en `ultima_revision_de_politicas` y **reabre por**
`cambio_significativo_en_el_entorno_de_pago`. La reapertura no es una segunda
obligacion: es un segundo disparador de la misma. Cuando el hecho consta y es
posterior a la ultima revision registrada, el ciclo anual deja de mandar y el
reloj sale **sin plazo legal**, porque el ritual dice cuando hay que revisar y no
da plazo para hacerlo. Lo que se mide entonces es el tiempo transcurrido desde el
hecho, no una fecha limite inventada.

### 2.5 `pcidss.ritual.formacion_y_concienciacion`, cada 12 meses

Arranca en `ultima_formacion_de_concienciacion`. Por que doce meses: la formacion
se ancla a la plantilla y la plantilla se renueva por ejercicio. Por debajo se
repite publico; por encima hay personas con un ano de acceso sin formacion.

### 2.6 `pcidss.ritual.revision_de_proveedores`, cada 12 meses

Arranca en `ultima_revision_de_proveedores`. Por que doce meses: la evidencia que
un proveedor produce (su informe, su certificado, su cuestionario) es anual, y
pedirla dos veces al ano devuelve la misma dos veces.

### 2.7 `pcidss.ritual.remediacion_de_vulnerabilidad_critica`, 30 dias naturales

Es un **plazo**, no una cadencia: arranca en
`deteccion_de_vulnerabilidad_critica` y vence a los 30 dias naturales, contados a
partir del dia siguiente, al final del ultimo dia y sin traslado.

Por que 30 dias naturales y no 15 ni 90: 30 es lo que deja cerrar el ciclo
completo de un parche (prueba, ventana de cambio, despliegue y comprobacion) sin
saltarse el control de cambios, y es lo que hace que la remediacion caiga siempre
dentro del mismo trimestre del analisis que la detecto. Con 15 el equipo salta el
control de cambios o incumple; con 90 la vulnerabilidad sobrevive a dos analisis
seguidos y el trimestral deja de significar nada.

Por que naturales y no habiles: una vulnerabilidad critica no descansa el fin de
semana. Los dias habiles se reservan para los rituales cuyo trabajo lo hacen
personas en horario de oficina.

Escalado: aviso al responsable de seguridad a los 15 dias y a direccion a los 25.

## 3. Lo que NO es un ritual de este paquete

- **La revision del conjunto de reglas del cortafuegos y la de la segmentacion de
  red.** Se identificaron y no se escriben. El unico argumento que sale para su
  intervalo se apoya en lo que exige un catalogo de pago, y apoyarse en eso
  redistribuye su CRITERIO aunque no copie una palabra. Sin argumento propio que
  se sostenga solo, el numero no se escribe.
- **La rotacion de claves criptograficas.** Su intervalo defendible depende del
  algoritmo y del volumen cifrado, no del calendario.
- **La revision de registros y trazas.** El unico intervalo honesto es continuo, y
  eso se escribe con la primitiva `continua`, no con una cadencia.
- **El ciclo de validacion anual del cliente ante su adquirente o su marca.** No
  sale del marco tecnico sino del contrato de cada entidad, y se modela con el
  objeto `Certificado` del nucleo con las fechas reales del cliente, no con una
  cadencia inventada aqui.
