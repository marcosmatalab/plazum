# nis1-es: RD 43/2021, la notificación de incidentes que vincula HOY en España

**URN:** `urn:es:rd:2021:43` · **Clase 1** (texto legal transcribible) · **En vigor desde 2021-01-29**

Real Decreto 43/2021, de 26 de enero, por el que se desarrolla el Real Decreto-ley 12/2018, de 7 de septiembre, de seguridad de las redes y sistemas de información. Fuente: BOE-A-2021-1192, [texto consolidado](https://www.boe.es/eli/es/rd/2021/01/26/43/con).

## Por qué este paquete va el primero de la familia A

Lo decidió el censo (`docs/censo-relojes.md`), no una preferencia, y las tres razones van en orden:

1. **Vincula hoy.** NIS2 sigue sin transponer en España, comprobado contra el índice de legislación consolidada del BOE y no contra la prensa. `nis2-ue` no obliga por sí misma a nadie; esto sí. Frente a `nis2-ue`, que es lo que hoy encabeza la venta española, es **la misma venta con la ventaja de ser cierta**.
2. **Es exactamente la primitiva de la familia**: tres hitos escalonados sobre un disparador de conocimiento.
3. **No caduca con la transposición.** Cuando salga, cambiarán los límites sobre una forma ya construida.

## El reloj, y por qué obligó a construir dos cosas en el motor

Todo sale de la **tabla 3 del anexo**, apartado 6 («Ventana temporal de reporte»):

| Nivel de peligrosidad o impacto | Notificación inicial | Notificación intermedia | Notificación final |
|---|---|---|---|
| CRÍTICO | Inmediata | 24/48 horas | 20 días |
| MUY ALTO | Inmediata | 72 horas | 40 días |
| ALTO | Inmediata | – | – |
| MEDIO | – | – | – |
| BAJO | – | – | – |

Y la nota al pie de la propia tabla, que es la que manda:

> Los tiempos reflejados en la tabla 3 para la «notificación intermedia» y la «notificación final» tienen como referencia el momento de remisión de la «notificación inicial». La «notificación inicial» tiene como referencia de tiempo el momento de tener conocimiento del incidente.

Esa nota y esa columna de niveles son, literalmente, **las dos cosas que se midieron contra el motor antes de escribir una línea de este paquete**:

- **Hitos encadenados.** La intermedia cuenta desde la **remisión de la inicial**, no desde el incidente. Un operador que tarda once horas en remitir la inicial tiene sus 24 horas **desde ahí**; contarlas desde el incidente le quitaría esas once, y el producto le estaría dando una fecha peor que la que la norma le da. Ya lo hacía `Hito.DesdeHito`.
- **El límite lo decide una categoría que asigna el propio obligado.** El mismo artículo da 24 h o 72 h según el nivel, y **el nivel lo pone quien sufre el incidente**. No lo pueden decidir las reglas de aplicabilidad del paquete, porque la aplicabilidad habla de la organización y esto es de **cada incidente**: la misma empresa tiene incidentes de niveles distintos el mismo mes. **No existía**; se construyó como `Hito.Clase`.

## Tres decisiones de lectura, dichas en voz alta

**«Inmediata» no es cero.** La norma impone la notificación inicial y **no fija plazo**. El motor lo dice con `sin plazo legal` en vez de inventarse un número. Devolver «cero horas» sería fabricar un vencimiento que la norma no da, y en un producto cuya promesa es el reloj legal eso es peor que no decir nada.

**«24/48 horas» son dos cifras y la norma no dice cuál rige.** No se elige en silencio: rige **la más corta** (24 h), que es la lectura conservadora, y la de 48 h viaja como **divergencia declarada con su cita**. El operador ve las dos y sabe por qué ve esa fecha:

```
notificacion_intermedia_critico    determinado    2026-03-03T20:00:00Z
   DIVERGENCIA 48h -> 2026-03-04T20:00:00Z (24h)
      RD 43/2021, anexo, tabla 3: para nivel CRITICO la notificacion
      intermedia dice «24/48 horas» y la norma no precisa cual de las dos rige.
```

**Un incidente sin clasificar no se calla.** Mientras el operador no asigne nivel, los hitos que dependen de él salen `pendiente de hecho` diciendo que rigen *si* se clasifica como X. Una lista vacía se leería como «nada que hacer», cuando lo que pasa es que falta un dato que pone él.

## Lo que este paquete NO trae, y hay que saberlo

- **ALTO, MEDIO y BAJO no tienen reloj aquí.** La tabla 3 sólo pone «Inmediata» para ALTO y nada para MEDIO y BAJO. El art. 9.1 obliga a notificar los de impacto crítico, muy alto o alto; los plazos escalonados sólo existen para los dos primeros.
- **Los umbrales que deciden el nivel** (apartados 3 y 4 del anexo: peligrosidad e impacto) **no están transcritos todavía**. El paquete toma el nivel como hecho declarado por el operador. Transcribir esas tablas es trabajo aparte y va cuando la familia A esté escrita entera.
- **El art. 9.2 permite a las autoridades competentes fijar umbrales sectoriales distintos.** Un operador supervisado por una autoridad que haya ejercido esa potestad tiene otros plazos, y este paquete no los conoce.
- **No sustituye a la notificación del art. 33 del RGPD.** Lo dice el propio art. 9.3: son independientes. Un incidente con datos personales dispara los dos relojes.

## Casos dorados

En `pruebas/tabla3.json`, cinco, **derivados del texto legal y no de la implementación**: si el motor y el caso discrepan, gana el caso.

Los dos que importan de verdad son el de la remisión tardía (que separa «cuenta desde la inicial» de «cuenta desde el incidente», y son once horas de diferencia) y el de **reclasificación**: un incidente que escala de MUY ALTO a CRÍTICO pasa a tener el plazo corto. Si siguiera rigiendo el nivel viejo, el operador vería 72 horas cuando le quedan 24.

## Atribución

Fuente de los datos: Agencia Estatal Boletín Oficial del Estado, BOE (https://www.boe.es). Texto consolidado de carácter meramente informativo. Esta reutilización no tiene carácter oficial.

Texto legal reproducible por el art. 13 del TRLPI (las disposiciones legales no son objeto de propiedad intelectual) y las condiciones de reutilización de la Agencia Estatal BOE.

**Nada de esto es asesoramiento jurídico.**
