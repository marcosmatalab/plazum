# Pendientes: el registro de P0 abiertos, P1 y P2

Los hallazgos que no se pierden en el cuerpo de un commit. **Esto es el índice;
el relato de cada familia, entero y sin resumir, está en
[`bitacora/pendientes-historico.md`](bitacora/pendientes-historico.md).**

Clasificación, la del protocolo de las tres pasadas
([`invariantes.md`](invariantes.md)):

- **P0** bloquea la casilla. Se arregla antes de marcarla, **y mientras siga sin
  arreglar vive aquí con su casilla sin marcar en [`ETAPAS.md`](ETAPAS.md)**. Un
  P0 abierto que no esté escrito en ningún sitio es un P0 que se olvida, y ese
  es peor negocio que la incomodidad de verlo en la lista.
- **P1** entra en la etapa. Se arregla dentro de la etapa en curso.
- **P2** a la lista. Se arregla cuando toque o se decide que no.

Cuando algo se cierra, **se muda a la bitácora** y consta en el commit que lo
cerró. No se borra: un relato fechado es lo único que permite reconocer la
séptima aparición de una familia.

> **Nota de 22-09-2026, y va aquí porque este fichero se contradecía a sí
> mismo.** El preámbulo decía «un P0 no entra aquí» y había dos; decía «cuando
> algo se cierra se borra de aquí» y había secciones marcadas `CERRADO`. Las dos
> reglas se han corregido para que digan lo que de verdad pasa, en vez de
> corregir la realidad para que quepa en la regla: **bajar los dos P0 a P1 para
> cumplir el preámbulo habría sido maquillaje**, y borrar lo cerrado habría
> tirado el material con el que se reconocen las familias.

## Abierto ahora

**2 P0, 29 P1 y 49 P2.** Los tres cardinales se derivan del archivo, no se
escriben a mano: los cuenta
`TestElIndiceDePendientesYElArchivoSeApuntanEnLasDosDirecciones`.

| Prioridad | Dónde | Qué queda |
|---|---|---|
| **P2** | [P2: la precision real de la pieza 3, medida y no supuesta (10-09-2026)](bitacora/pendientes-historico.md#p2-la-precision-real-de-la-pieza-3-medida-y-no-supuesta-10-09-2026) | La precisión de la pieza 3 está supuesta y no medida. |
| **P0** | [P0 de la pieza 1: BM25 no ve la negacion, y la entrevista asistida se queda abierta (10-09-2026)](bitacora/pendientes-historico.md#p0-de-la-pieza-1-bm25-no-ve-la-negacion-y-la-entrevista-asistida-se-queda-abierta-10-09-2026) | La entrevista asistida no distingue el «sí» del «no». Bloquea la casilla de la pieza 1, que sigue sin marcar. |
| **P0** | [P0 del tramo 4: el arreglo del TTFV tiene una pieza que nadie habia costeado](bitacora/pendientes-historico.md#p0-del-tramo-4-el-arreglo-del-ttfv-tiene-una-pieza-que-nadie-habia-costeado) | D11-e, la fila que decide la fecha de la v1, llevaba dentro una pieza sin costear. 35 de 68. |
| **P1** | [P1](bitacora/pendientes-historico.md#p1) | **29 abiertos** de 30. Numerados, y hay código que cita el número: no se renumeran en bloque. |
| **P2** | [P2](bitacora/pendientes-historico.md#p2) | **49 abiertos** de 54. |

## El archivo, por familias

Lo que ya está cerrado o es doctrina. Se guarda porque **reconocer la séptima
aparición de una familia es lo que ha cerrado casi todas**, y eso no se puede
hacer sobre un resumen. Los cuadernos de auditoría por campaña están al lado,
en [`bitacora/`](bitacora/LEEME.md).

| Familia |
|---|
| [La otra familia: "sin confiar en el emisor"](bitacora/pendientes-historico.md#la-otra-familia-sin-confiar-en-el-emisor) |
| [La familia: guardas que no guardaban](bitacora/pendientes-historico.md#la-familia-guardas-que-no-guardaban) |
| [La familia: piezas terminadas sin el cable (02-09-2026)](bitacora/pendientes-historico.md#la-familia-piezas-terminadas-sin-el-cable-02-09-2026) |
| [El puente entre la entrevista y el motor: piloto hecho, y lo que queda (02-09-2026)](bitacora/pendientes-historico.md#el-puente-entre-la-entrevista-y-el-motor-piloto-hecho-y-lo-que-queda-02-09-2026) |
| [La capa visual: lo que entró y lo que quedó fuera, con su cardinal (02-09-2026)](bitacora/pendientes-historico.md#la-capa-visual-lo-que-entró-y-lo-que-quedó-fuera-con-su-cardinal-02-09-2026) |
| [El armazón llega a las cuatro superficies (03-09-2026)](bitacora/pendientes-historico.md#el-armazón-llega-a-las-cuatro-superficies-03-09-2026) |
| [Familia B del censo: lo que quedó abierto al cerrar `eni` y `mica` (08-09-2026)](bitacora/pendientes-historico.md#familia-b-del-censo-lo-que-quedó-abierto-al-cerrar-eni-y-mica-08-09-2026) |
| [Lo que queda del `<li>` de /alcance al cerrar la absolución ciega (10-09-2026)](bitacora/pendientes-historico.md#lo-que-queda-del-li-de-alcance-al-cerrar-la-absolución-ciega-10-09-2026) |
| [El aviso de directiva llega a la pantalla, y lo que queda fuera (10-09-2026)](bitacora/pendientes-historico.md#el-aviso-de-directiva-llega-a-la-pantalla-y-lo-que-queda-fuera-10-09-2026) |
| [Los cardinales muertos del tablero: la mitad viva queda atada, la congelada queda contada (10-09-2026)](bitacora/pendientes-historico.md#los-cardinales-muertos-del-tablero-la-mitad-viva-queda-atada-la-congelada-queda-contada-10-09-2026) |
| [Las piezas 4 y 7, y la escalada del godoc (11-09-2026)](bitacora/pendientes-historico.md#las-piezas-4-y-7-y-la-escalada-del-godoc-11-09-2026) |
| [La ruta de subida y la pieza 3 (10-09-2026)](bitacora/pendientes-historico.md#la-ruta-de-subida-y-la-pieza-3-10-09-2026) |
