# evals/

Los conjuntos dorados de la IA, partidos en **dos cadencias** (`docs/guia.md` §7.1), y esa partición es lo que hace que el primero pueda existir hoy:

| conjunto | cadencia | estado |
|---|---|---|
| **citas** (el verificador antialucinación) | **cada PR**, sin red y sin modelo | **28 casos**, en `citas/dorados.json` |
| extracción de obligaciones | nightly y release, con modelo fijado | por escribir, 50 casos previstos |
| contradicciones | nightly y release, con modelo fijado | por escribir, 20 casos previstos |

**Por qué el primero corre en cada PR y los otros dos no.** El verificador de citas es un `sha256` y una comparación de cadenas: no llama a ningún modelo, así que no cuesta dinero, no expone secretos a un fork y no produce rojos aleatorios. Los otros dos sí llaman, y un eval con modelo como puerta de cada push es un rojo aleatorio con factura.

## El formato

Un conjunto es **un solo fichero JSON con su corpus de mentira dentro**. Va todo junto a propósito: un conjunto dorado que dependa del corpus real cambia de significado cada vez que alguien corrige una coma de un paquete, y entonces un rojo no distingue *«el arnés se ha roto»* de *«el corpus ha cambiado»*.

```json
{
  "nombre": "...",
  "porque": "...",
  "fuentes": [
    {"id": "...", "marco": "...", "articulo": "...",
     "clase": "transcrito", "procedencia": "corpus", "texto": "..."}
  ],
  "casos": [
    {"id": "...", "porque": "que ataque es este y por que importa",
     "cita": "...", "hash_de": "id-de-una-fuente",
     "veredicto": "aceptada"},
    {"id": "...", "porque": "...", "cita": "...", "hash_literal": "deadbeef",
     "veredicto": "descartada", "motivo": "hash_ilegible"}
  ]
}
```

Cuatro reglas del formato, y ninguna es ceremonia:

- **El caso NOMBRA su fuente (`hash_de`) y el arnés DERIVA el hash.** Un `sha256` tecleado a mano tiene la forma de lo verificable, y esa forma es justo lo que hace que nadie vaya a verificarlo. Además, escrito a mano, el día que se cambie una coma del texto de la fuente el caso empezaría a probar *«el hash no existe»* en vez de lo que dice probar, **y seguiría en verde**. `hash_literal` es sólo para los casos cuyo ataque ES el hash.
- **`citable` NO es un campo.** Se deriva de la clase con la misma función que usa el producto (`ia.ClaseCitable`), así que un conjunto no puede declarar que un paquete referencial es citable y medir su propia opinión.
- **Un caso descartado dice POR QUÉ**, de un vocabulario cerrado de 8 motivos. Un caso que sólo exige «que falle» pasa también cuando falla por el motivo equivocado, y entonces el eval mide otra cosa.
- **Cada caso lleva su `porque`**, que sale en el mensaje del fallo. Un caso dorado sin el ataque explicado es un caso que nadie sabe si puede borrar.

## Lo que el arnés se niega a cargar

Un conjunto es un fichero que edita una persona con prisa. Si el arnés tomara valores por defecto ante un campo que falta, un caso mal escrito pasaría **siempre** y el eval publicaría una precisión que no ha medido. Así que paran la carga: veredicto ausente, veredicto que no se entiende, descarte sin motivo, motivo fuera del vocabulario, hash dicho de dos formas, hash sin decir, fuente que no existe, identificadores repetidos, campos que el formato no conoce, conjunto sin casos, procedencia ilegible y clase ilegible.

## Correrlo

```bash
go test ./evals/... -count=1     # sin red, sin modelo, sin GPU
```

Los números se publican en cada release con el modelo y la versión fijados, que es el hito de *«el primer GRC que publica la precisión de su IA»*.
