# Los cuadernos de hallazgos

> **Qué son.** El informe de cada campaña: lo que se construyó, lo que se decidió apartándose de la casilla, lo que salió mal y lo que quedó abierto con su cardinal. Uno por frente y por tramo, con la fecha en la cabecera.
>
> **Por qué viven aparte desde el 09-09-2026.** Eran 18 ficheros y **454.200 bytes**, el 32,8 % de `docs/`, y estaban mezclados en el primer nivel con los documentos que sí se leen para trabajar (`guia.md`, `diseno.md`, `decisiones.md`, `pendientes.md`, `censo-relojes.md`). Quien abría `docs/` veía 42 ficheros y no podía distinguir el plan de la bitácora. Ahora el primer nivel tiene **25** y la bitácora tiene su sitio.
>
> **Lo que NO se ha hecho: resumir, fundir ni borrar.** Ni un byte. Un cuaderno de hallazgos es la única forma que tiene este repositorio de decir *«esto se intentó, salió así y costó esto»*, y consolidarlos convertiría dieciocho relatos fechados en un resumen sin autor. Lo único que ha cambiado es la carpeta, y las **45 referencias** que apuntaban a ellos se reescribieron en el mismo commit.

## Índice

| cuaderno | fecha | de qué va |
|---|---|---|
| [ai-act](ai-act.md) | 03-09-2026 | los relojes del AI Act, frente A |
| [barrido](barrido.md) | 04-09-2026 | el barrido de las 58 casillas **cerradas**, el que miraba hacia el lado que faltaba |
| [censo-a](censo-a.md) | 03-09-2026 | censo de `pci-dss`, `soc2` y `tisax` |
| [censo-b](censo-b.md) | 03-09-2026 | los huecos de siete paquetes ya escritos |
| [conservacion](conservacion.md) | 03-09-2026 | la ley de conservación del calendario |
| [corpus-t3](corpus-t3.md) | 04-09-2026 | la rebanada de corpus del tramo 3 |
| [cra-nis2](cra-nis2.md) | 03-09-2026 | `cra` y `nis2-ue` |
| [d11](d11.md) | 04-09-2026 | D11 en el calendario, el acta y el escalado |
| [entrada](entrada.md) | 03-09-2026 | el frente de la entrada: usuarios y sesión |
| [entrevista](entrevista.md) | 03-09-2026 | la revelación progresiva de la entrevista de alcance |
| [ia](ia.md) | 04-09-2026 | los cuatro cimientos de la IA: búsqueda, verificador, evals e interruptor |
| [pantallas](pantallas.md) | 04-09-2026 | la puerta D11-a, cero formaciones |
| [persistencia](persistencia.md) | 04-09-2026 | la persistencia del alcance |
| [preaviso](preaviso.md) | 03-09-2026 | encender la primitiva `preaviso` para el corpus |
| [prueba](prueba.md) | 07-09-2026 | el eslabón de la prueba (D-22, A1) |
| [puente](puente.md) | 02-09-2026 | el puente de la entrevista al motor, dos tramos |
| [release](release.md) | 04-09-2026 | la release candidata y la máquina limpia |
| [vigencias](vigencias.md) | 03-09-2026 | las tres fechas de una norma |

**Y el índice va con puerta en las dos direcciones** (`hallazgos_test.go`): todo cuaderno del directorio está en la tabla, y toda fila de la tabla tiene su cuaderno. Sin la segunda mitad, la tabla se queda citando ficheros que ya no existen; sin la primera, un cuaderno nuevo entra sin que nadie lo vea, que es como este directorio volvería a ser un montón.
