# Hallazgos de censo del frente A: pci-dss, soc2, tisax

Este fichero es del frente A y NO toca `docs/censo-relojes.md`, que en esta
campana lo escribe otro frente. El integrador funde los dos al cerrar.

**Lo que este frente ha hecho, dicho en una linea**: no ha censado nada de esos
tres marcos. Ha ESCRITO rituales de plazum sobre ellos, que es otra cosa y no
mueve ni una casilla de la tabla del censo. La distincion importa porque las dos
se parecen en el resultado (un paquete con relojes) y son opuestas en lo que
afirman: un censo dice *«la norma exige N cosas con reloj»* y un ritual dice
*«plazum propone hacer esto cada tanto, y aqui esta el argumento»*.

## 0. La frontera, aplicada antes de escribir

Los tres son estrato **referencial** (clase 2). El invariante 3 prohibe
transcribir su texto y el linter de prosa prohibe ademas nombrar un marco cerrado
ajeno. De ahi salen dos decisiones que estan en todo lo que sigue:

1. **Ni un identificador de requisito, de criterio ni de control.** Ninguna de
   las 17 obligaciones escritas dice a que punto del marco sirve. La razon es el
   invariante 10, no la pereza: un numero de requisito escrito de memoria tiene la
   FORMA de lo verificable, y esa forma es justo lo que hace que nadie vaya a
   comprobarlo. Verificarlo exigiria leer la copia licenciada, y eso es
   exactamente lo que la frontera prohibe hacer aqui.
   **Cardinal del hueco: 17 de 17 rituales sin anclaje a punto del marco.**
   Lo cierra el cliente en su instancia, con su copia delante, y es una linea por
   ritual.
2. **Ni un numero que venga del marco.** Las 14 cadencias llevan
   `origen_del_intervalo: "propuesto"` con su justificacion, su
   `cuando_cambiarlo` y sus fuentes. Ninguna dice `suelo_legal` ni `fijado`,
   porque para decirlo habria que citar el punto que da el numero, y eso es el
   punto 1 otra vez.

## 1. Los recuentos, antes de escribir

Formula de la casa, por paquete y dicha antes de la primera obligacion.

### pci-dss

**7 contados, +6 identificados y sin escribir.**

Escritos (7): revision de accesos (P3M), analisis de vulnerabilidades (P3M),
prueba de intrusion (P12M), revision de las politicas (P12M, con reapertura por
cambio significativo), formacion de concienciacion (P12M), revision de
proveedores (P12M), y un plazo de remediacion de vulnerabilidad critica (30 dias
naturales).

Identificados y NO escritos (6), con su motivo uno a uno:

| ritual | por que no se escribe |
|---|---|
| revision del conjunto de reglas del cortafuegos | **es el caso documentado de la parafrasis sin nombre**. El unico argumento que sale para su numero se apoya en lo que exige un catalogo de pago, y eso redistribuye su CRITERIO aunque no copie una palabra. Sin un argumento propio que se sostenga solo, el numero no se escribe |
| revision de la segmentacion de red | mismo motivo que el anterior, y ademas su intervalo depende de si hay segmentacion, que es un dato del alcance que este paquete no pregunta |
| rotacion de claves criptograficas | el intervalo defendible depende del algoritmo y del volumen cifrado, no del calendario. Un numero fijo aqui seria un numero inventado con adorno |
| inventario de activos | se solapa entero con el analisis de vulnerabilidades trimestral, que ya recorre lo mismo. Escribirlo duplicaria la ceremonia y le diria al cliente que tiene dos donde tiene una |
| revision de registros y trazas | el unico intervalo honesto es «continua», y una `continua` no admite los tres dorados que el linter exige. Entra cuando se escriba con la primitiva que le toca |
| revision por la direccion | esta escrita, pero en `soc2`, donde el ciclo de gobierno tiene un ancla propia (el periodo de observacion). Repetirla aqui seria la misma ceremonia con dos nombres |

### soc2

**5 contados, +4 identificados y sin escribir.**

Escritos (5): cierre del periodo de observacion (P12M), revision de accesos
(P3M), evaluacion de riesgos (P12M), revision por la direccion (P12M), y un plazo
de remediacion de excepcion (20 dias habiles).

Identificados y NO escritos (4):

| ritual | por que no se escribe |
|---|---|
| prueba de recuperacion | esta escrita en `tisax`, donde la disponibilidad de la cadena de suministro le da un ancla mas concreta. Aqui seria la misma ceremonia con otro nombre |
| revision de la carta de compromisos con el cliente | su ritmo lo pone el contrato, no el marco ni plazum. Un intervalo nuestro encima de un contrato del cliente es una imposicion disfrazada de dato |
| revision de las descripciones de sistema | se mueve con el periodo de observacion, del que ya cuelga el ritual 1. Escribirlo aparte partiria en dos un unico deber |
| pruebas de los controles por el auditor | no es un ritual del obligado: lo ejecuta un tercero contratado. Meterlo en el calendario del cliente le pone una fila que no puede cerrar |

### tisax

**5 contados, +4 identificados y sin escribir.**

Escritos (5): reevaluacion de la etiqueta (P24M desde la obtencion, o sea doce
meses antes de que caduque), revision del alcance (P12M), prueba de recuperacion
(P12M), formacion de concienciacion (P12M), y un plazo de actualizacion del
registro del alcance (10 dias habiles desde el cambio).

Identificados y NO escritos (4):

| ritual | por que no se escribe |
|---|---|
| revision de los requisitos de proteccion de prototipos | su aplicabilidad depende del nivel de proteccion contratado, que es un dato del alcance que este paquete no pregunta todavia |
| revision de proveedores de la cadena | esta escrita en `pci-dss`. El argumento del intervalo es el mismo (la evidencia del proveedor es anual) y duplicarlo no anade nada |
| revision de la clasificacion de la informacion | el intervalo defendible sale del catalogo, que no se puede leer. Sin argumento propio, no hay numero |
| gestion del incidente que afecta a un cliente de la cadena | es una notificatoria, y una notificatoria exige decir A QUIEN alcanza y de donde sale que no alcanza a los demas. Eso no se puede contestar sin la copia, y un umbral escrito de menos provoca una actuacion indebida ante un tercero |

## 2. Lo que estos tres paquetes NO cambian del censo

`docs/censo-relojes.md` deja los tres en «no verificado», y **sigue siendo el
resultado correcto**. Nada de lo escrito aqui lo mueve:

- Para `pci-dss` el censo dice que el marco SI fija cadencias numericas propias y
  que contarlas exige la copia. Eso no ha cambiado. Los 7 relojes escritos son
  nuestros y no son los suyos.
- Para `tisax` lo unico verificable sin el catalogo es la vigencia de la etiqueta,
  que es proceso publico de ENX. Verificado el 03-09-2026 en la pagina publica de
  ENX sobre TISAX: la validez de una evaluacion es de tres anos. De ahi cuelga el
  ritual 1, y de nada mas.
- Para `soc2` el censo dice que la periodicidad la pone el contrato de atestacion
  y no el marco. Eso es lo que hace que el cero sea defendible, y es la propuesta
  de `censados` de mas abajo.

## 3. Lo que se propone para `paquetes/marcos-v1.json`

**Este frente NO ha tocado ese fichero.** Lo escribe el integrador. La propuesta,
con su motivo:

| paquete | `censados` propuesto | por que |
|---|---|---|
| `pci-dss` | **sigue `null`**, con el `sin_verificar_porque` actualizado | el censo afirma que el marco SI fija cadencias numericas propias y que no se pueden contar sin la copia licenciada. Un `0` aqui seria FALSO y cualquier otro numero seria inventado. `null` deja el paquete fuera del numerador y del denominador, que es exactamente lo que el fichero dice que significa |
| `tisax` | **sigue `null`** | la vigencia de la etiqueta es publica y esta verificada, pero el recuento de puntos del catalogo con cadencia exige el catalogo. Un denominador de 1 seria contar lo unico que se pudo mirar y llamarlo el total |
| `soc2` | **sigue `null`** | es el unico de los tres donde el `0` defendido de `iso27001` seria defendible, y aun asi no se propone. Ver el parrafo de abajo: hoy un `0` con relojes escritos SUBE el porcentaje, y eso convierte una buena noticia en un numero peor |

Texto propuesto para el `sin_verificar_porque` de los tres, para que deje de
leerse como *«aqui no hay nada»*: **«referencial: el recuento de los relojes DEL
MARCO exige la copia licenciada y sigue sin hacerse. El paquete si trae N
rituales de plazum con intervalo propuesto, que no son suyos y por eso no entran
en este denominador.»** (N = 7, 5 y 5.)

### El hallazgo del propio contador, que es lo que hay que decidir antes

**Un paquete con `censados: 0` y relojes escritos SUBE el porcentaje de cobertura
de la v1, y ese es el unico caso en el que el numero miente hacia arriba.**
`coberturaDeLaV1` suma `escritos` de todo marco cuyo `censados` no sea `null`, y
suma `censados` al denominador. Con `censados: 0`, la fila aporta N al numerador y
cero al denominador: el porcentaje no mide cobertura, mide cuantos rituales
propios se han escrito.

Esto **ya pasa hoy** con `iso27001` (0 censados, 6 relojes escritos), asi que no
lo trae este frente: lo destapa. Y por eso no se propone `censados: 0` para
`soc2`: seria anadir 5 al numerador de un cociente que no los tiene en el
denominador, o sea empeorar la medida justo mientras se dice que se mejora la
cobertura.

**Efecto real de este frente sobre el porcentaje: NINGUNO.** Con los tres en
`null`, ni el numerador ni el denominador se mueven, y
`TestElPorcentajeDeLaV1LoComputaUnTestYNoUnaPersona` sigue en verde. La puerta que
SI se pone roja es otra, y es correcta:
`TestLosNumerosDelCorpusEnElREADMESalenDelArbol`, porque el README declara cuantos
paquetes traen relojes, cuantos hitos y cuantos dorados hay, y los tres suben. El
README esta fuera de la frontera de este frente: los numeros nuevos van en el
informe para que los ponga el integrador.

**Lo que se propone decidir en lote** (no lo decide un worktree): o el numerador
solo suma relojes de origen `suelo_legal` y `fijado`, que son los que el
denominador cuenta, o los rituales propios salen del cociente y se publican como
segunda cifra. Las dos son honestas; la de hoy no lo es.
