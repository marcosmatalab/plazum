# psd2-es: de dónde sale cada fecha

## 1. El artículo 67.1: obliga y no se puede calcular

> Los proveedores de servicios de pago notificarán al Banco de España, **de forma
> inmediata y en la forma que este determine**, los incidentes operativos o de
> seguridad graves.

No hay número, y no se inventa. El hito `notificacion_al_banco_de_espana` se
declara con límite `indeterminado`, sale como *sin plazo legal* y el motor mide
el tiempo transcurrido desde el conocimiento del incidente. El escalado sí lleva
números (aviso al responsable de seguridad de pagos a las 2 horas y a dirección a
las 4), y esos son de plazum, no de la norma: son la respuesta operativa a un
plazo que no se puede calcular.

**Esta obligación es la que cambió el linter.** Hasta el 27-08-2026, toda
obligación con reloj tenía que traer tres casos dorados, y de un plazo sin número
no sale ninguna fecha esperada. La regla empujaba a quitarle el reloj a esta
obligación para que el paquete cargara, y entonces el producto dejaba de enseñar
el cronómetro. Ahora los tres dorados se exigen cuando hay algún límite
computable, y la exención se paga con la nota del hito.

## 2. El artículo 66.2: cadencia con suelo legal

> Los proveedores de servicios de pago proporcionarán al Banco de España, con la
> periodicidad y forma que éste determine, **al menos una vez al año**, una
> evaluación actualizada y completa de los riesgos operativos y de seguridad

Doce meses de fecha a fecha desde la última evaluación registrada, al final del
día, sin traslado. El suelo es legal: el Banco de España puede exigir más
frecuencia y el obligado no puede espaciarla.

Las cuentas de los tres dorados:

- Última evaluación el 10 de marzo de 2026: la siguiente vence el 10 de marzo de
  2027, al final del día.
- Borde: última evaluación el **29 de febrero de 2024**. 2025 no es bisiesto, así
  que el 29 no existe y el ciclo cierra el 28. Es el recorte al último día del
  mes destino, que en Derecho español sale del artículo 5.1 del Código Civil.
- Segunda vuelta: la ocurrencia n vence a los 12·n meses de la última evaluación
  registrada. La cadencia no es un acto único.

## 3. Régimen

Para el artículo 66.2: cómputo natural, cierre a fin de día, **traslado
ninguno**. Una evaluación interna no es un plazo administrativo del artículo 30.5
de la Ley 39/2015, así que no hereda su regla de traslado. Si el Banco de España
fija una fecha concreta de remisión, esa sí sería administrativa y llevaría
traslado; no está en el real decreto-ley.
