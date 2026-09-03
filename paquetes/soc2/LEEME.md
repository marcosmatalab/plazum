# soc2: que trae este paquete y que no

**Estrato: referencial. Aqui NO estan los criterios y no van a estar.** El marco
lo publica su emisor con sus propias condiciones, y este proyecto no redistribuye
su contenido. Lo que hay aqui son **rituales de plazum**: cinco ceremonias con
reloj, con el intervalo puesto por nosotros y el argumento escrito al lado.

## Lo primero, porque es lo que mas confunde

**Ninguno de estos cinco relojes es un criterio del marco.** Sus identificadores
empiezan por `soc2.ritual.` y su campo `articulo` dice literalmente
`ritual plazum sobre ...`. Si alguien te pregunta de donde sale una de estas
fechas, la respuesta es *«la pone plazum, aqui esta el porque, y la puedo mover»*,
no *«lo exige el marco»*.

**Y ninguno dice a que criterio sirve.** Ese anclaje exige la copia licenciada
delante. Lo pones tu en tu instancia: es una linea por ritual. El hueco esta
contado, no escondido: **5 de 5**, en `docs/hallazgos-censo-a.md`.

## Que hay dentro

| Ritual | Reloj | Clase |
|---|---|---|
| Cierre del periodo de observacion | cada 12 meses | documental |
| Revision de accesos | cada 3 meses | procedimental |
| Evaluacion de riesgos | cada 12 meses | procedimental |
| Revision por la direccion | cada 12 meses | documental |
| Plan de remediacion de una excepcion | 20 dias habiles desde la excepcion | remediacion |

Los cuatro primeros son cadencias con `origen_del_intervalo: "propuesto"`: **el
numero es nuestro y lo puedes mover en las dos direcciones**, y cada uno trae en
el propio paquete su `justificacion_del_intervalo` y su `cuando_cambiarlo`. El
quinto es un plazo.

Cada reloj trae tres casos dorados en `pruebas/`, derivados de `RITUALES.md` y no
de la implementacion. Si el motor y un caso discrepan, gana el caso.

## De donde sale el ritmo

De una sola cosa: el **periodo de observacion**. Un informe de tipo II describe un
periodo, no un instante, y quien lo examina muestrea evidencia repartida por todo
el. De ahi salen los dos argumentos que este paquete usa para poner numeros
(encadenamiento sin huecos entre informes, y densidad de puntos de evidencia
dentro del periodo), explicados en `RITUALES.md` seccion 0.

**Dos de los cuatro intervalos no declaran fuentes, y es deliberado**: su
argumento sale de la forma del propio informe y no del criterio de nadie. Los
otros dos si las declaran.

## A quien alcanza

A quien declare que adopta el marco. Las reglas de `aplicabilidad` derivan
`aplica(<ritual>, E)` desde `adopta(E, "soc2")` y desde nada mas. Este paquete no
afirma nada sobre tu sector ni sobre las categorias que hayas incluido en el
alcance, porque eso esta en el marco y el marco no se puede leer desde aqui.

## Lo que este paquete NO hace

- **No sustituye a los criterios.** Sin tu copia, aqui hay rituales, no criterios.
- **No emite ni prepara el informe.** Eso lo hace un examinador independiente.
- **No mapea ritual a criterio.** Ver arriba: 5 de 5 sin anclar, a proposito.
- **No cubre la prueba de recuperacion.** Esta escrita en el paquete `tisax`, con
  su motivo en `RITUALES.md` seccion 3.

## Aviso

Esto no es asesoramiento juridico ni una interpretacion autorizada del marco. Las
cadencias de `RITUALES.md` son criterio de plazum.
