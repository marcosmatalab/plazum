# iso27001: que trae este paquete y que no

**Estrato: referencial. Aqui NO esta el texto de la norma y no va a estarlo.**
ISO/IEC 27001:2022 es una norma de pago y su licencia no permite redistribuir su
contenido. Este paquete trae el identificador de cada requisito y de cada control
mas una etiqueta corta nuestra para reconocerlo, nada mas. El enunciado normativo
lo pone tu copia licenciada, comprada a ISO, a UNE o a tu organismo nacional.

El linter del proyecto lo hace cumplir por ti: una obligacion de un paquete
referencial con mas de 120 caracteres de texto normativo no carga, y el paquete
entero se rechaza. La etiqueta mas larga de este paquete tiene 86 caracteres.

## Que obliga y desde cuando

- **Desde el 25 de octubre de 2022**, fecha de publicacion de ISO/IEC 27001:2022,
  que es la vigencia declarada de todas las obligaciones del paquete.
- Certificarse es **voluntario**. ISO no es derecho: nada de esto obliga por si
  mismo. Obliga cuando tu lo asumes en un contrato, en un pliego o al pedir la
  certificacion. Si lo que buscas es una obligacion legal espanola equivalente,
  mira el paquete `ens`.
- Si vienes de ISO/IEC 27001:2013, el periodo de transicion a la edicion de 2022
  cerro el 31 de octubre de 2025. Un certificado de 2013 ya no es valido.

## Que hay dentro

| Bloque | Obligaciones |
|---|---|
| Clausulas 4 a 10, los requisitos del sistema de gestion | 30 |
| Anexo A tema 5, controles organizativos | 37 |
| Anexo A tema 6, controles de personas | 8 |
| Anexo A tema 7, controles fisicos | 14 |
| Anexo A tema 8, controles tecnologicos | 34 |
| Rituales de plazum con reloj | 6 |
| **Total** | **129** |

Por clase de implantacion: 53 procedimentales, 43 observables, 28 documentales,
4 de remediacion y 1 notificatoria.

## Los relojes son NUESTROS, no de ISO

ISO no fija ningun periodo numerico: pide planificar los intervalos y deja el
numero a la organizacion. Los seis relojes de este paquete son rituales de plazum
con una cadencia de partida, explicada y justificada uno a uno en `RITUALES.md`,
y cambiable sin tocar codigo. Sus identificadores empiezan por
`iso27001.ritual.` precisamente para que nadie los confunda con un requisito de
la norma.

| Ritual | Cadencia | Sirve a |
|---|---|---|
| Auditoria interna | 12 meses | 9.2.2 |
| Revision por la direccion | 12 meses | 9.3.1 |
| Apreciacion de riesgos | 12 meses | 8.2 |
| Revision de la declaracion de aplicabilidad | 12 meses | 6.1.3 |
| Revision independiente | 24 meses | A.5.35 |
| Plan de accion tras no conformidad | 10 dias habiles | 10.2 |

Cada uno trae tres casos dorados en `pruebas/`, derivados de `RITUALES.md` y no
de la implementacion, y se recalculan contra el motor en cada ejecucion de los
tests. Si el motor y un caso discrepan, gana el caso.

## Que se pregunta antes de ensenarte nada

El paquete declara ocho preguntas de alcance, ordenadas por cuantas obligaciones
desbloquea cada una. La primera pregunta desbloquea catorce obligaciones (los
controles fisicos) y la segunda nueve (los de desarrollo). Nadie tiene que leer
un catalogo de 93 controles en frio para empezar.

## Lo que este paquete NO hace

- **No sustituye a la norma.** Sin tu copia licenciada, aqui hay identificadores
  y etiquetas, no el enunciado de lo que hay que hacer.
- **No te certifica.** Certifica una entidad acreditada; esto ordena el trabajo y
  deja rastro verificable de que se hizo y cuando.
- **No cubre ISO/IEC 27002.** Las guias de implantacion de cada control estan en
  27002, que es otra norma de pago y otro paquete, hoy esqueleto.
- **No trae equivalencias con el ENS todavia.** El mapeo ENS a ISO en formato
  OSCAL, con su lista de huecos computada, es la casilla siguiente de la etapa 3.
- **29 de las 129 obligaciones no declaran recurso observable**, o sea que hoy no
  hay conector que las mida solas: se cierran con evidencia documental o con un
  ritual. Estan nombradas una a una en la salida de cobertura, no escondidas
  detras de un porcentaje.

## Aviso

Esto no es asesoramiento juridico ni una interpretacion autorizada de ISO/IEC
27001. Las cadencias de `RITUALES.md` son criterio de plazum y no proceden de la
norma.
