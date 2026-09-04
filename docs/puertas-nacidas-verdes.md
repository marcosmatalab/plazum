# Puertas nacidas verdes, y su fecha de revisión

> **Para qué es este fichero.** Una puerta que nace verde sobre el árbol real no ha demostrado que vigile: ha demostrado que hoy no hay nada que cazar. Puede ser que vigile poco, puede ser que llegara tarde, y las dos cosas hay que saberlas. Así que se anota, con fecha, y se revisa **a los quince días**.
>
> **La fila la escribe quien escribe la puerta, no quien la revisa.** Si la escribiera el revisor, la lista sólo tendría las puertas que alguien se acordó de mirar, que son justo las que no hacen falta en una lista.
>
> **Qué NO entra aquí.** Una puerta que nace **roja** sobre el árbol y se pone verde al arreglar lo que encontró. Ésa ya ha vigilado algo que nadie le puso delante, que es exactamente lo que este fichero busca comprobar. Se anota en su commit y se acabó.

## La regla de revisión, con sus dos salidas

A los quince días se mira **si la puerta ha estado roja alguna vez sin que su autor la rompiera a propósito**. Se contesta con el historial (`git log`, la salida de CI), no de memoria.

- **Ha cazado algo**: se anota qué, se saca la fila de esta tabla y la puerta pasa a ser una puerta normal.
- **No ha cazado nada**: se decide **explícitamente** entre dos cosas, y no se deja pasar otra quincena. O **vigila poco**, y entonces o se amplía o se dice que cubre menos de lo que su nombre promete. O **llegó tarde**, y entonces está bien que exista y lo que hay que preguntarse es qué otra puerta habría hecho falta ANTES.

## Abiertas

| Puerta | Nació | Se revisa | Por qué nació verde | Qué se sabe hoy |
|---|---|---|---|---|
| `TestGoSumNoEsUnRecipienteVacio` | 04-09-2026 | **19-09-2026** | nació en el mismo commit que quitó `go.sum` del índice, así que el árbol ya estaba arreglado cuando la puerta empezó a mirar | **sólo se ha visto fallar contra mutaciones de su propio autor** (M-A, M-B, M-C). La rama «con dependencias» no la recorre ninguna entrada real y no la va a recorrer hasta que entre la primera dependencia, que puede ser en etapas |
| `TestLaInstantaneaNoPublicaCardinalesQueElArbolYaDesmiente` | 04-09-2026 | **19-09-2026** | nació en el mismo commit que remidió la instantánea | **se estrenó en rojo sobre dato real, pero sobre el commit ANTERIOR** (`80627ed`), con tres contrastes a la vez. O sea que documenta un fallo que ya había ocurrido en vez de haberlo impedido. Es más de lo que puede decir la de arriba y sigue sin ser lo mismo que vigilar |

## Cerradas

Ninguna todavía. La primera revisión toca el 19-09-2026.
