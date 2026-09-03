# nis2-ue: la Directiva NIS2

**Estado: 12 obligaciones, 16 hitos y 24 casos dorados.** La cadena de
notificacion del art. 23.4 con sus cinco hitos, sus dos hermanas sin cifra del
mismo apartado (la letra c) y la letra e), esta ultima con numero propio), el
aviso a los destinatarios del servicio de los arts. 23.1 y 23.2, las medidas
correctoras del 21.4, los DOS relojes de cambios en la informacion (art. 3.4,
dos semanas, y art. 27.3, tres meses), los dos del registro de nombres de
dominio (art. 28.4 y art. 28.5, este con 72 horas) y los dos de los mecanismos
de intercambio del art. 29.4. Los plazos y su computo, en `COMPUTO.md`.

**Y lo primero, antes que el reloj**: esto es una DIRECTIVA. Los plazos de aqui
no se le pueden ensenar a una empresa espanola como exigibles hoy. Se transcriben
porque son los que las transposiciones van a llevar dentro y porque en otros
Estados miembros ya vinculan. Lo que vincula hoy en Espana es el RD 43/2021,
paquete `nis1-es`.

## Que vincula de verdad, y por que esto importa antes de leer nada mas

**Esta directiva no te obliga a ti.** Una directiva obliga a los Estados
miembros, no a las entidades: su texto esta redactado como "los Estados miembros
velaran por que las entidades...". Lo que obliga a una empresa espanola es la
**norma de transposicion**, y sus plazos son los de esa norma, no los de aqui.

**En Espana esa norma no existe todavia.** A la fecha del censo, el Anteproyecto
de Ley de Coordinacion y Gobernanza de la Ciberseguridad, aprobado en Consejo de
Ministros el 14-01-2025, seguia en tramitacion, y no se ha localizado publicacion
en el BOE consolidado.

Consecuencia practica, y se dice en vez de disimularla:

- El `identificador` de este paquete apunta a la directiva porque es el unico texto que
  hay. No apunta a lo que vincula, porque lo que vincula no esta publicado.
- Las cifras de la directiva (24 horas, 72 horas, un mes) son las que el
  legislador espanol **va a transponer**, y lo normal es que las respete, pero
  hasta que se publiquen no son un plazo exigible en Espana.
- Antes de escribir este paquete hay que comprobar el BOE **del dia**. Si la ley
  ha salido, los plazos que vinculan son los suyos, y este paquete pasa a ser la
  referencia europea de un paquete espanol, no la norma.

## Los tres relojes de "cambio" y de "peticion", que se parecen y no son el mismo

Es la trampa que mas caro sale de este paquete, y ninguno de los tres estaba
escrito hasta el 03-09-2026.

| articulo | que cambia | plazo | a quien alcanza |
|---|---|---|---|
| 3.4, parrafo segundo | la informacion de la **lista nacional** del art. 3.3 | **dos semanas** desde que se produjo el cambio | toda entidad esencial o importante |
| 27.3 | la informacion del **registro de ENISA** del art. 27.2 | **tres meses** desde que se produjo el cambio | solo la lista cerrada del art. 27.1 |
| 28.5 | nada: es responder a una **solicitud de acceso** a los datos de dominios | **72 horas** desde la recepcion | solo registros de dominios de primer nivel y prestadores de servicios de registro |

Un proveedor de servicios de computacion en nube esta en los dos primeros a la
vez, y sobre datos que se solapan (nombre, direccion, contacto, Estados miembros
donde presta servicios). **Manda el corto**: creerse los tres meses del art. 27.3
para un cambio de domicilio social es llegar diez semanas tarde a la autoridad
competente. Y el tercero va en **horas**, asi que no se traslada al lunes cuando
cae en sabado, al reves que los otros dos.

## Donde estan los relojes, segun el censo

**Y el censo se queda corto, medido el 03-09-2026 leyendo el articulado.** La
ficha del censo cuenta **9 puntos unicos** para este marco y el paquete tiene ya
**12 obligaciones**, porque el censo lee CITAS y el paquete se escribe leyendo el
ARTICULO. Los cinco que el censo no tenia: **art. 3.4 parrafo segundo** (dos
semanas), **art. 23.4 letra e)** (un mes desde la gestion del incidente), **art.
28.4**, **art. 28.5** (72 horas) y **art. 29.4**, que ademas son dos. La
correccion de la fila esta escrita en `docs/hallazgos-cra-nis2.md`;
`docs/censo-relojes.md` no lo toca este paquete.

- Plazo (5): art. 23.4.a (alerta temprana, 24 horas), art. 23.4.b (notificacion
  del incidente, 72 horas), art. 23.4.d (informe final, un mes), art. 27.3
  (cambios en la informacion registrada, tres meses) y art. 27.2 (fecha limite de
  registro para proveedores de DNS, nube y centros de datos, 17-01-2025). El art.
  23.4.c, el informe intermedio, no lleva numero y **ya esta escrito**, con su
  hito sin cifra. **Escritos: cuatro de cinco**, y el que falta es el 27.2, con
  su motivo mas abajo.
- Periodicidad (1, sin numero): art. 20.2. **La ficha del censo lo llamaba
  "formacion periodica de los organos de direccion" y eso no es lo que dice el
  apartado**, comprobado contra Cellar el 02-09-2026. El art. 20.2 tiene dos
  verbos y el adverbio cuelga del segundo: los Estados miembros *garantizaran*
  que los miembros de los organos de direccion "deban asistir a formaciones" (sin
  ritmo y sin numero) y *alentaran* a las entidades a que ofrezcan formaciones
  similares "a sus empleados **periodicamente**". O sea que lo periodico es la
  formacion de los EMPLEADOS, y ademas cuelga de "alentaran", que no es una
  obligacion de resultado sobre la entidad. **No se ha escrito**: escribirlo como
  una cadencia del organo de direccion seria ponerle ritmo a un deber al que la
  norma no se lo pone, y colgarlo del verbo equivocado.
- Evento (5): incidente significativo, ciberamenaza significativa, constatar el
  incumplimiento de las medidas del 21.2, cambio en la informacion registrada y
  el conocimiento del incidente. **Escritos los cinco**, y el primero resulto ser
  dos: el art. 23.1 manda notificar al CSIRT (que es la cadena del apartado 4) y,
  en su segunda frase, avisar A LOS DESTINATARIOS DEL SERVICIO, que es otro
  destinatario y no hereda las 24 horas de la letra a).

Detalle completo en `docs/censo-relojes.md`.

## El art. 27.3 no alcanza a toda entidad esencial o importante

Es la trampa que este paquete tiene dentro y que se prueba en las dos
direcciones. El art. 23.4 obliga a **toda** entidad esencial o importante; el
art. 27.3 dice "las entidades a que se refiere el apartado 1", y el art. 27.1 es
una **lista cerrada** de infraestructura digital: proveedores de servicios de
DNS, registros de nombres de dominio de primer nivel, entidades que prestan
servicios de registro de nombres de dominio, proveedores de servicios de
computacion en nube, de servicios de centro de datos, de redes de distribucion de
contenidos, de servicios gestionados y de servicios de seguridad gestionados, y
proveedores de mercados en linea, de motores de busqueda en linea y de
plataformas de servicios de redes sociales.

Un hospital o una electrica del anexo I estan en el ambito de la Directiva y
**no** estan en ese registro. Por eso el art. 27.3 cuelga de un papel propio,
`papel_nis2_registro(E, "entidad_del_art_27_1")`, y no de
`designado(E, "entidad_esencial_o_importante")`. Importa mas que un coste de mas:
es una obligacion `notificatoria`, su entregable **sale de la organizacion**, y
escrita ancha provoca una presentacion ante la autoridad competente que nadie
pidio y que no se deshace.

**Y lleva vigencia propia, no la del paquete.** El art. 41.1 manda aplicar las
disposiciones nacionales a partir del 18-10-2024 y el art. 27.2 no exige remitir
la informacion de registro hasta el 17-01-2025: un deber de notificar cambios en
una informacion que todavia no habia que remitir no puede vincular desde la
entrada en vigor de la Directiva (16-01-2023, art. 45).

**El art. 27.2 no se ha escrito, y el motivo se dice.** Es una fecha fija
(17-01-2025) que ya paso, y en Espana no hay transposicion, asi que ensenarla
como vencida le diria a un proveedor de nube espanol que incumplio un registro
que nadie le ha pedido todavia. Entra el dia que exista la norma de
transposicion, con la fecha de esa norma.

## Derechos

Texto del DOUE. La Decision 2011/833/UE autoriza la reutilizacion **con
atribucion**, y el aviso literal viaja en el campo `atribucion` del paquete y
sale en la pantalla del producto.

## Aviso

Esto no es asesoramiento juridico.
