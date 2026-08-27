# vigilancia: los hechos de fuera que obligan a revisar el corpus

Un **item de vigilancia** es un hecho que todavía no ha ocurrido, o que ocurrió y
costó algo, y del que cuelga una lectura del corpus. Tiene nombre, dueño en el
corpus y una forma mecánica de detectarse.

## Por qué existe, y lo que costó

El 27-08-2026 el paquete `ai-act` afirmaba que dos fechas del AI Act «no vinculan
porque el ómnibus digital no está publicado en el DOUE». Llevaban publicadas
**treinta y cuatro días**: Reglamento (UE) 2026/1744, de 8 de julio de 2026. Lo
encontró una revisión, no una puerta.

Ese es el fallo que este directorio existe para que no vuelva a depender de que
alguien mire. Una lectura divergente es, por definición, una apuesta sobre algo
que aún no ha pasado; sin decir **de qué cuelga**, nadie sabe cuándo dejó de ser
cierta. Y una divergencia que envejece mal es peor que no tenerla, porque se le
enseña al cliente al lado de la fecha que sí vincula.

## Qué es y qué NO es

**Es** una lista de datos, no de código. Los identificadores de norma viven aquí
y en `paquetes/`, nunca en un literal de Go: eso lo rompe `extensibilidad_test.go`.

**No es** un mecanismo que toque el corpus. Un item que se dispara **abre un
issue**; mover una fecha de `alternativas` al campo `desde` es autoría, la hace
una persona y se commitea con su cita. Automatizar eso sería fabricar derecho,
que es exactamente lo contrario de este producto.

**No falla, avisa.** Un rojo mensual permanente es tan invisible como un verde
falso: en este repositorio el bloqueante de `gosec` estuvo rojo cinco commits sin
que nadie lo leyera. Por eso la detección vive en un workflow con horario, no en
un test, y por eso un test que compare una fecha con el reloj de pared está
prohibido aquí (sería una bomba con la mecha encendida).

## Las dos clases de item

| Clase | Qué se vigila | Cómo se detecta |
|---|---|---|
| `publicacion` | que un instrumento cambie en la fuente oficial | `ingestanorma` reejecutado sobre su identificador, comparado con la instantánea de `corpus-vigilancia/` |
| `fecha` | que llegue una fecha escrita en la propia norma | el workflow mensual la compara con el día en que corre, y avisa dentro de la ventana declarada |

Un item de clase `fecha` **no** se puede vigilar con un test: la comparación con
el reloj de pared solo es honesta en un trabajo con horario, que es lo que la
hace reproducible y no sorpresiva.

## El formato

```json
{
  "id": "identificador-en-kebab",
  "nombre": "una linea que un humano entiende",
  "clase": "publicacion | fecha",
  "porque": "que se rompe si esto pasa y nadie se entera",
  "vigila": [{"jurisdiccion": "ue", "identificador": "32024R1689", "que": "que se mira"}],
  "cuando": "AAAA-MM-DD",
  "ventana_de_aviso_dias": 90,
  "cuelga_de": [{"paquete": "urn:...", "obligacion": "id", "lectura": "id-de-la-alternativa"}],
  "al_dispararse": "que tiene que hacer una persona",
  "ocurrido": {"cuando": "AAAA-MM-DD", "que": "...", "coste": "..."}
}
```

`cuelga_de` es lo que ata el item al corpus, y **se comprueba en las dos
direcciones** (`vigilancia_test.go`): toda lectura con `espera` nombra un item
que existe, y todo item nombra lecturas que existen. La dirección que se olvida
es la segunda: un item huérfano parece vigilancia y no vigila nada.

`ocurrido` se rellena cuando el item dispara, y **no se borra el item**. El
historial de lo que se vio venir y de lo que no es lo único que un competidor no
puede fingir.
