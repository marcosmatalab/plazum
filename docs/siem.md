# El log de auditoría hacia tu SIEM

Esta guía es para la persona que conecta plazum con Splunk, Elastic, Sentinel o
Loki. Va al grano, con el comando que hay que pegar, los campos que salen y, lo
que casi nadie escribe, **lo que NO sale y por qué**.

## En una línea

```bash
plazum export expediente.json
```

Escribe un evento por línea, en JSON, por la salida estándar. El resumen sale
por el canal de errores, así que la tubería lleva solo eventos.

```bash
plazum export expediente.json | nc -q0 siem.interno 5514
plazum export expediente.json --salida /var/log/plazum/auditoria.jsonl
```

Con `--salida` el fichero se crea con permisos 0600. Lleva el rastro de
auditoría de tu organización en texto plano, así que no se deja legible para
cualquier cuenta de la máquina.

## Por qué JSON líneas

Una línea, un evento, sin envoltura y sin comas. Es lo único que traga cualquier
SIEM sin escribir un transformador. Un array envuelto obliga al receptor a leer
el fichero entero antes de la primera línea, y un fichero que crece mientras se
lee no se puede envolver.

## Qué es un evento aquí, y qué no

Esto no es un volcado del ledger. El ledger es la capa probatoria de plazum, sus
entradas van cifradas con una clave por entrada, y su valor está en que un
tercero las recalcule, no en que las lea un panel. El SIEM es lo contrario, un
receptor externo con retención propia al que le mandas texto en claro para que
dispare alertas.

De ahí salen las cinco acciones que existen.

| `event.action` | Qué es | Para qué alertas |
|---|---|---|
| `ledger.entrada` | Se encadenó una entrada de evidencia | Un recolector que deja de escribir |
| `ledger.punto_de_control` | Se cerró y selló un punto de control | `plazum.sellado` a falso significa que ese cierre no está anclado todavía |
| `ledger.supresion_legal` | Se ejerció un borrado con base legal | Un pico de supresiones, o una supresión sin expediente asociado |
| `control.estado` | El estado declarado de un control | `event.outcome` a `failure` con `plazum.escala_al_auditor` a verdadero |
| `obligacion.vencimiento` | Un plazo legal calculado | `plazum.vencido` a verdadero, que es un plazo que ya pasó |

## Lo que NO viaja, y por qué

Esta es la parte que conviene leer antes de firmar el contrato con el SIEM.

**1. Lo que un borrado legal borró no reaparece.** Cuando una entrada se suprime
con base legal, plazum destruye su clave y firma una lápida dentro de la cadena.
El export publica el hecho de la supresión, su índice y su base legal, y **nunca
el contenido**. La comprobación pregunta por la lápida y no por la clave, y lo
pregunta antes de tocar el material cifrado. La diferencia importa porque los dos
actos viven en almacenes distintos con retenciones distintas, así que pueden
discrepar, por ejemplo cuando se restaura una copia del almacén de claves dentro
de su ventana de retención. **En esa discrepancia manda la lápida.**

El motivo es práctico y no ceremonial. Tu SIEM es un tercero que retiene meses o
años. Lo que entra ahí ya no lo alcanza ninguna orden de supresión que ejecutes
en plazum.

Por eso vale también un borrado a medias. Suprimir son dos escrituras, la lápida
firmada dentro de la cadena y la atribución en el expediente que dice qué control
se quedó sin evidencia. Si solo aparece la segunda, el expediente no verifica,
pero el export tampoco publica el contenido, y lo dice. Ante la duda no sale,
porque retener de más es un evento con menos campos y filtrar de más no tiene
vuelta atrás.

**2. Lo que nadie ha revisado no sale.** De dentro de una entrada solo salen los
campos de una lista cerrada. No es una lista de prohibidos sino de permitidos, así
que un recolector que algún día escriba una cabecera de autorización dentro de su
carga no filtra nada por este camino. Cuántos campos se quedaron fuera se dice en
`plazum.campos_omitidos`, para que no sea un descarte silencioso.

**3. El texto de un error de recolección no sale nunca.** Ese campo lo escribe un
tercero y es donde acaba una URL firmada o una cabecera con credencial cuando un
recolector falla. Lo que tu SIEM necesita de ahí, que hubo error, sale como
`plazum.error_de_recoleccion`.

**4. Las claves por entrada no salen.** Viajan dentro del expediente porque es su
función, que el receptor pueda abrir la cadena. Un log de texto plano hacia un
tercero no es ese sitio.

## Los campos

Los nombres son los de ECS (Elastic Common Schema) para lo que todo SIEM ya sabe
mapear solo, y con prefijo `plazum.` para lo que es de este dominio.

| Campo | Qué es |
|---|---|
| `@timestamp` | UTC, con milisegundos y ancho fijo |
| `event.kind` | siempre `event` |
| `event.dataset` | siempre `plazum.auditoria` |
| `event.action` | una de las cinco de la tabla de arriba |
| `event.sequence` | 1..N dentro del fichero, sin huecos |
| `event.id` | identificador estable del hecho, para deduplicar |
| `event.outcome` | `success`, `failure` o `unknown` |
| `message` | la línea legible, que es la que ve el analista |
| `organization.name` | la organización del expediente |
| `observer.product` | siempre `plazum` |
| `user.name` | el actor, cuando la entrada lo declara |
| `plazum.esquema` | versión del formato de evento |
| `plazum.instante_es` | de dónde sale `@timestamp`, ver abajo |

El resto de campos `plazum.` son opcionales y dependen de la acción. La lista
completa está en el godoc de `superficies/export`.

## De dónde sale la marca de tiempo

Esto conviene entenderlo antes de mirar un panel, porque plazum no inventa
precisión y otros productos sí.

La cadena de plazum **no fecha sus entradas**. Lo único fechado y anclado contra
una autoridad de sellado es el punto de control, que acota por arriba el instante
de todo lo que cubre. Así que cada evento dice de dónde sale su marca de tiempo,
en `plazum.instante_es`.

| Valor | Significa |
|---|---|
| `observacion_recolectada` | el instante real en que se recogió la evidencia |
| `checkpoint` | el instante del punto de control |
| `cota_superior_del_checkpoint` | no sabemos cuándo pasó, solo que fue antes de ese cierre |
| `lapida` | el instante que firma la lápida |
| `vencimiento_declarado` | la fecha en que vence el plazo |
| `emision_del_expediente` | la fecha a la que el expediente retrata la organización |

Si construyes una alerta sobre latencia o sobre orden de sucesos, filtra por
`plazum.instante_es` primero. Una cota superior no es un instante.

## Determinismo y deduplicación

Dos ejecuciones sobre el mismo expediente dan el mismo fichero byte a byte. Eso
te da dos cosas.

- **Huecos detectables.** `event.sequence` va de 1 a N sin saltos, así que un
  hueco en el índice significa que algo se perdió en el camino, no que plazum
  dejó de escribir.
- **Deduplicación.** `event.id` es estable para el mismo hecho, y no depende del
  contenido. Dos exportaciones que se solapan no cuentan el mismo hecho dos
  veces, y una entrada no cambia de identificador al suprimirse su contenido.

## Cómo se conecta cada SIEM

**Splunk.** Fichero monitorizado con `sourcetype = _json`, o entrada HEC. Los
campos con punto se indexan tal cual.

```
[monitor:///var/log/plazum/auditoria.jsonl]
sourcetype = _json
index = seguridad
```

**Elastic.** Filebeat con el parseo de ndjson, que es el formato nativo de ECS.

```yaml
filebeat.inputs:
  - type: filestream
    paths: ["/var/log/plazum/auditoria.jsonl"]
    parsers:
      - ndjson: {target: "", overwrite_keys: true}
```

**Microsoft Sentinel.** Agente de Azure Monitor con una regla de recolección de
datos sobre texto, formato `json/stream`, y una tabla propia. Los nombres con
punto llegan como columnas.

**Grafana Loki.** Promtail, con una etapa `json` que extrae `event.action` y
`organization.name` como etiquetas. No pongas `event.id` de etiqueta, que es de
cardinalidad alta y te hace explotar el índice.

## Retención, y una decisión que es tuya

plazum puede ejecutar un borrado con base legal sobre su propio ledger. **Sobre
tu SIEM no puede.** Antes de conectar esto, decide y escribe:

- cuánto retiene el SIEM estos eventos;
- quién los puede leer;
- qué haces con ellos cuando alguien ejerce su derecho de supresión.

Lo que este export te da es que la respuesta a la tercera pregunta sea corta,
porque el contenido suprimible nunca llegó a salir. Lo que quedan son metadatos
de cadena, identificadores de control y fechas.

## Automatizarlo

Un temporizador de systemd es suficiente. Nada de esto necesita red saliente
salvo la que va a tu propio SIEM.

```ini
[Unit]
Description=Export del log de auditoria de plazum al SIEM

[Service]
Type=oneshot
User=plazum
ExecStart=/usr/local/bin/plazum export /var/lib/plazum/expediente.json --salida /var/log/plazum/auditoria.jsonl
```

```ini
[Timer]
OnCalendar=hourly
Persistent=true
```

Como el fichero es determinista, reescribirlo entero cada hora no genera eventos
nuevos en el SIEM si tienes la deduplicación por `event.id` puesta.

---

plazum no presta asesoramiento jurídico. Lo que ves aquí es lo que dicen los
paquetes normativos que tienes instalados, con su cita, para que puedas
comprobarlo tú.
