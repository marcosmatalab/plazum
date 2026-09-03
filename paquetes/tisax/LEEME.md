# tisax: que trae este paquete y que no

**Estrato: referencial. Aqui NO esta el catalogo con el que se evalua y no va a
estarlo.** Lo que hay son **rituales de plazum**: cinco ceremonias con reloj, con
el intervalo puesto por nosotros y el argumento escrito al lado.

## Lo primero, porque es lo que mas confunde

**Ninguno de estos cinco relojes es un requisito del catalogo.** Sus
identificadores empiezan por `tisax.ritual.` y su campo `articulo` dice
literalmente `ritual plazum sobre ...`.

**Y ninguno dice a que punto del catalogo sirve.** Ese anclaje exige tu copia
delante. El hueco esta contado, no escondido: **5 de 5**, en
`docs/hallazgos-censo-a.md`.

## El unico dato externo que se usa, con su verificacion

**La validez de una evaluacion es de tres anos.** No sale del catalogo: sale del
proceso publico que ENX describe en su propia pagina sobre TISAX, verificada el
03-09-2026. De ahi cuelga el primer ritual, y de nada mas.

Lo que ese dato **no** autoriza es a escribir una cadencia que no sea propuesta:
la validez de tres anos es un hecho del proceso, y el ritual es la decision de
plazum de **arrancar la reevaluacion doce meses antes de que caduque**, que es un
numero nuestro y lo puedes mover.

## Que hay dentro

| Ritual | Reloj | Clase |
|---|---|---|
| Arranque de la reevaluacion | cada 24 meses desde la obtencion | procedimental |
| Revision del alcance (sedes y procesos) | cada 12 meses | documental |
| Prueba de recuperacion | cada 12 meses | procedimental |
| Formacion de concienciacion | cada 12 meses | procedimental |
| Actualizacion del registro del alcance | 10 dias habiles desde el cambio | documental |

Los cuatro primeros son cadencias con `origen_del_intervalo: "propuesto"`, cada
una con su `justificacion_del_intervalo` y su `cuando_cambiarlo` dentro del propio
paquete. El quinto es un plazo.

Cada reloj trae tres casos dorados en `pruebas/`, derivados de `RITUALES.md`. Si
el motor y un caso discrepan, gana el caso.

## A quien alcanza

A quien declare que adopta el marco. Las reglas derivan `aplica(<ritual>, E)`
desde `adopta(E, "tisax")` y desde nada mas. Este paquete no afirma nada sobre el
nivel de proteccion que hayas contratado ni sobre tu posicion en la cadena,
porque eso esta en el catalogo.

## Lo que este paquete NO hace

- **No sustituye al catalogo.** Sin tu copia, aqui hay rituales.
- **No te evalua ni te da la etiqueta.** Eso lo hace un proveedor de evaluacion.
- **No mapea ritual a punto del catalogo.** Ver arriba: 5 de 5 sin anclar.
- **No trae la notificacion de incidente a un cliente de la cadena.** Es una
  notificatoria y su umbral no se puede escribir sin la copia; escribirlo de menos
  provocaria una actuacion indebida ante un tercero, y eso no se deshace. El
  motivo, en `RITUALES.md` seccion 3.

## Aviso

Esto no es asesoramiento juridico ni una interpretacion autorizada del marco ni
del proceso de evaluacion. Las cadencias de `RITUALES.md` son criterio de plazum.
