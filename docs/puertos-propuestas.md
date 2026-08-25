# Propuestas de cambio en los puertos

Los puertos de `puertos/` están congelados: varios frentes compilan contra ellos a
la vez y un cambio de firma rompe a los demás en silencio. Lo vigila
`puertos/congelacion_test.go`.

**Un frente que necesita cambiar un puerto no para y no lo cambia.** Escribe la
propuesta aquí, sigue contra el interfaz actual con un `TODO` que enlace a esta
entrada, y continúa. Las decisiones se resuelven en lote, no una a una: parar
cinco frentes para decidir una firma cuesta más que el rodeo.

## Formato de una entrada

```
### <puerto>.<método>: <qué falta en una línea>
- Frente: (a) serve / (b) pantallas / ...
- Qué se necesita: la firma o el método nuevo, escrito.
- Por qué el interfaz actual no da: el caso concreto que no se puede expresar.
- Rodeo mientras tanto: qué se ha hecho para no parar.
- Coste de no cambiarlo: qué queda peor si se decide que no.
```

## Decididas

### 2026-08-25 · `UIGenerada`: retirado como puerto
La derivación de `corpus.EsquemaUI` a modelo de pantalla no es una frontera de
sustitución, es una función pura del corpus. Vive en `nucleo/pantalla`: cero
dependencias, determinista, con casos dorados y bajo el test AST del núcleo. El
render sigue por el puerto `Plantilla`. Motivo: un puerto existe para poder
cambiar la implementación; aquí no queremos dos derivaciones distintas del mismo
corpus, queremos una y comprobable.

### 2026-08-25 · `Seguridad`: retirado como puerto
Nada de capa de seguridad enchufable en un producto cuya tesis es que el receptor
no se fía. Las cabeceras y el rate limit son middleware con puerta de CI que
arranca el servidor y las comprueba contra peticiones reales. El CSRF se emite en
`Sesion` y se aplica en middleware, con un test que enumera las rutas del router y
falla si alguna ruta mutante no pasa por él.

### 2026-08-25 · `Secretos`: puerto nuevo
Generación de tokens y aleatoriedad. `crypto/rand` en producción, fuente
determinista en tests. Es el único puerto añadido sobre la lista pedida.

### 2026-08-25 · `Catalogo`: se queda con ese nombre
Regla en su godoc: **el catálogo nunca transporta `texto_legal`**. El idioma del
corpus va por paquete. Traducir texto transcrito del BOE crea obra derivada y se
sale de la estratificación de licencias.

## Abiertas

(ninguna)
