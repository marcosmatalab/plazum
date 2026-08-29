# mica — Reglamento (UE) 2023/1114

Texto del DOUE, transcrito (estrato transcrito, Decisión 2011/833/UE). Fuente: instantánea con huella de Cellar, CELEX 32023R1114.

**A quién alcanza.** MiCA reparte por **papel**, y el papel lo declara la entidad:

| papel | qué es |
|---|---|
| `oferente` | ofrece criptoactivos distintos de fichas referenciadas a activos y de dinero electrónico |
| `emisor_de_fichas_referenciadas_a_activos` | emite ART |
| `proveedor_de_servicios_de_criptoactivos` | presta servicios de criptoactivos (CASP) |

## Los seis relojes con número

| art. | cadencia | quién | qué |
|---|---|---|---|
| 10.2 | **P1M** | oferente | publicar en el sitio web las unidades en circulación |
| 22.1 | **P3M** | emisor de ART **> 100 M EUR** | comunicar a la autoridad competente |
| 30.1 | **P1M** | emisor de ART | actualizar la información pública de la reserva |
| 35.1 | **P12M** | emisor de ART | revisar el importe de los gastos fijos generales |
| 67.1 | **P12M** | CASP | revisar el importe de los gastos fijos generales |
| 72.4 | **P12M** | CASP | evaluar y revisar la política de conflictos de intereses |

### El umbral de los 100 millones

El art. 22.1 **no alcanza a todo emisor de ART**: sólo a los que emitan por encima de **100 000 000 EUR**. El valor de emisión lo conoce la entidad y no plazum, así que se declara como hecho.

Un umbral mal escrito no da error: da **una comunicación trimestral a la autoridad que la entidad no debe hacer**, que no es sólo un coste de más. Se comprueba en las dos direcciones en `TestMicaRepartePorPapelYPorElUmbralDeCienMillones`, y la mutación que quita el umbral lo pone en rojo.

### «En todo momento» no es un reloj

Los arts. 35.1 y 67.1 exigen fondos propios o salvaguardias prudenciales **«en todo momento»**. Eso es una **condición permanente**, no una cadencia. Lo periódico es una sola cosa y está en el párrafo final de cada uno: **la revisión anual del importe** de la cuarta parte de los gastos fijos generales del año anterior.

Colgar el número de *«disponer de fondos propios»* diría que basta con tenerlos una vez al año, que es **autorizar un infracumplimiento con cara de control**. Es la misma trampa del verbo que en el punto 6.9.2 del anexo de 2024/2690.

### La lectura de «trimestralmente» y «anualmente»

Los arts. 22.1, 35.1 y 67.1 **no dicen «al menos»**. Aun así se leen como **intervalo máximo** y no como fecha exacta: hacerlo antes no puede incumplir un deber de hacerlo, y la lectura de número exacto obligaría a *no hacerlo* durante el periodo, que es absurda. Es la misma lectura razonada del anexo I.1 del ENS, escrita allí con su porqué. Va en la cita de cada intervalo, para que un jurista pueda discutirla.

## Lo que este paquete NO hace todavía

- **El art. 81.12** (revisar la evaluación de idoneidad **por cliente**, al menos cada dos años). Es un reloj **por objeto**, no por organización: la norma cuenta los dos años desde la evaluación inicial de *cada cliente*. Modelarlo como un ciclo de la organización diría algo parecido y distinto. Va con el objeto por cliente, igual que los criterios de incidente significativo de 2024/2690 van con el objeto `Incidente` de la etapa 4.
- **Los relojes de las autoridades** (AEVM, ABE, Comisión): informes anuales, registros y notificaciones de sanciones. Quedan fuera por la regla del censo — el corpus recoge lo que obliga al sujeto, no lo que obliga a quien lo supervisa.
- **Los plazos únicos**: el resultado de la oferta pública a los veinte días hábiles (art. 10.1), y los plazos de autorización y de libro blanco.
- **Las fichas de dinero electrónico** (título IV) y los emisores significativos (arts. 43 y siguientes).
