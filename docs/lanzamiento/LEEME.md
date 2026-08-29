# Material de lanzamiento

Las tres capturas que protagonizan el post, **generadas del binario de verdad** y reproducibles en un comando:

```bash
./docs/lanzamiento/generar.sh
```

Cada fichero lleva en cabecera el commit del que salió y el instante cableado (`--ahora`). **Sin el instante cableado no serían reproducibles**: la captura del estreno del CRA deja de tener sentido en cuanto pasa el 11 de septiembre de 2026, y nadie podría distinguir un post que mentía de un mundo que avanzó.

## 1. `1-estreno-del-cra.txt` — la fila que nadie más imprime

```
EMPIEZA A OBLIGARTE DENTRO DE ESTA VENTANA (7 hitos en 2 obligaciones)

    desde el 2026-09-11  Notificar un incidente grave en la seguridad del producto: alerta, notificacion e informe final  [supuesto]
              urn:eu:reg:2024:2847  art. 14.3 y 14.4
              hoy todavia no obliga: no hay nada que entregar y tampoco nada que hayas incumplido
```

**Un fabricante de software español, dos banderas, diez segundos.** El artículo 14 del Reglamento de Ciberresiliencia empieza a aplicarse el **11 de septiembre de 2026**, y ese día un fabricante pasa a tener veinticuatro horas para alertar de una vulnerabilidad aprovechada activamente.

Lo que hace singular la fila no es la fecha: es que **la obligación todavía no existe**. Un catálogo de controles no la tiene (no ha nacido), y un calendario que sólo enseña vencimientos tampoco (no vence nada). Sale porque el producto distingue *estrenar* de *vencer*, y esa distinción se escribió justamente porque sin ella estas dos filas se caían en silencio.

## 2. `2-las-sentadas.txt` — la composición entre marcos, computada

```
  ciclo bienal (P24M): 3 obligaciones de 2 marcos
      2 con fecha, en 1 sentada
      1 esperando un dato tuyo (la ultima vez que lo hiciste)
      se pueden juntar: 3 de las 3 se pueden adelantar.
      abril de 2027: 2 fechas de 2 marcos
          19  Revision independiente de la seguridad de la informacion  [supuesto]
                urn:iso-iec:27001:2022  art. ritual plazum sobre A.5.35  (adelantable)
          19  Encargar la revision independiente de la seguridad de la informacion y las redes  [supuesto]
                urn:eu:reg-ejec:2024:2690  art. ritual plazum sobre el anexo, punto 2.3.4  (adelantable)
```

**Dos marcos, una sesión.** No porque alguien haya escrito a mano que se parecen, sino porque **arrancan del mismo hecho registrado** y caen el mismo mes. Es lo que un listado de controles no puede decir: no tiene reloj.

Y el consejo de agrupar sabe lo que puede mover y lo que no. Cada fecha dice `(adelantable)` o `NO se mueve`, y eso sale del `origen_del_intervalo` del corpus: con **suelo legal** apretar siempre cumple, con un **número de plazum** se puede mover, y con un **número exacto de la norma** no se toca. Sin esa distinción, *«junta estas doce en una sesión»* sería proponerle al cliente que incumpla una de ellas.

## 3. `3-lo-vencido-y-la-cuenta.txt` — el descargo y la cuenta entera

```
YA VENCIDO Y SIN CONSTANCIA DE QUE SE HAYA HECHO (1 obligacion, 1 vencimiento)

    vencio el 2026-01-15  Revisar en el organo de direccion los roles...
    Esto NO dice que se haya incumplido: dice que en tus respuestas no
    consta que se hiciera. Si se hizo, registra la fecha y desaparece.
```

**El descargo va con el dato, no en una nota al pie.** plazum no sabe si la obligación se cumplió: sabe que no consta. Acusar en falso es el único error que un producto de cumplimiento no puede cometer ni una vez, y hay un test que exige esa frase.

Y debajo, la cuenta:

```
LA CUENTA, ENTERA
    106 hitos de reloj instalados en paquetes
     98 en vigor el 2026-09-01
     48 alcanzados por la aplicabilidad
      2 fechas en los proximos doce meses (un hito periodico da varias)
      ...
     50 instalados que NO te alcanzan segun tus respuestas (verlos: --todos-los-relojes)
```

**Ningún reloj desaparece.** Todo lo instalado acaba en una fila o en un cubo con nombre y motivo, y eso no es una promesa del post: es una ley con su test (`TestTodoRelojInstaladoAcabaEnExactamenteUnDestino`), escrita después de perder 46 relojes en silencio y encontrarlos.

## Lo que estas capturas NO son

**No son una demo preparada.** Salen del corpus publicado, con los perfiles publicados y el binario del commit que dice la cabecera. Lo que enseñan es lo que le sale a cualquiera que ejecute el mismo comando.

**Y no son la prueba de nada por sí solas.** Las líneas que sostienen el argumento están vigiladas por tests con nombre, que es lo que las hace verificables cuando el corpus crezca: `TestAlgunaSentadaCubreMasDeUnMarco`, `TestLoVencidoSaleArribaDelTodoYConSuDescargo`, `TestElDetalleDiceQueSePuedeAdelantarYQueNo`. Una captura envejece; un test no.
