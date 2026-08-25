# Decisiones

> **Para qué sirve este documento.** Las decisiones que cierran una puerta, con el porqué y con el dato que las sostiene. No es un registro de todo lo que se decide: aquí entra lo que alguien va a querer reabrir dentro de seis meses, para que reabrirlo cueste leer un dato en vez de rehacer el razonamiento.
>
> Formato: qué se decide, cuándo, qué lo sostiene, y qué cambia en el repositorio. Si una decisión se revierte, se tacha y se dice por qué, no se borra.

---

## D-1. OSCAL sale del camino crítico

**Fecha:** 25-08-2026.

**Qué se decide:**

1. **No se construye el importador OSCAL.** Deja de ser casilla de etapa.
2. **NIST 800-53 y NIST CSF salen de la autoría de corpus.** Siguen como esqueletos en `paquetes/`, sin trabajo asignado.
3. **OSCAL puede ser adaptador de SALIDA con pérdidas, nunca modelo interno ni formato de entrada.** Es regla de arquitectura y está en `CLAUDE.md`.
4. **El export OSCAL y el Mapping Model se van a etapa 6 o posterior.** Las equivalencias ENS con ISO se hacen en formato propio.
5. **Una sola tubería de ingesta**, el extractor legal desde el ELI del BOE y de EUR-Lex. Reejecutable, porque es también el mecanismo de vigilancia normativa.

### Por qué: el comprador

Mil controles federales estadounidenses no le sirven a un CISO europeo de 20 a 5.000 empleados. NIST 800-53 es el catálogo de un régimen de contratación pública de otro país. Autorizarlo como corpus es trabajo caro que no mueve ni una decisión de compra en el mercado al que se vende esto.

### Por qué: OSCAL vive de obligación, no de adopción

El dato, y es el que zanja la discusión:

- **En 2025 FedRAMP procesó más de 100 autorizaciones Rev5 sin una sola presentación que usara OSCAL.** Ni siquiera los participantes formales del piloto de la Fase 1 de FedRAMP 20x lo usaron para estructurar el material legible por máquina que se les exigía.
- Ahora lo impone por mandato: la RFC-0024 exige paquetes legibles por máquina a partir de **septiembre de 2026**, con un periodo de gracia con notificación pública hasta el **30 de septiembre de 2027**, y **revocación de la certificación FedRAMP** a partir de esa fecha, lo que obliga a una autorización inicial completamente nueva.

Nueve años de formato y la adopción real es cero hasta que aparece la amenaza de revocación. Eso no es un estándar que el mercado quiera, es un estándar que un regulador impone, y encima en un mercado que no es el nuestro.

Fuentes: [RFC-0024, FedRAMP/community](https://github.com/FedRAMP/community/discussions/114), [resumen del mandato y sus fechas](https://quzara.com/fedramp/oscal).

### Por qué: el modelo de OSCAL no tiene dónde poner un plazo

Esta es la razón técnica, y es la que hace que la decisión no sea reversible por conveniencia.

El modelo de OSCAL es `catalog > group > control > part`. **No hay campo para un plazo.** Un control es un enunciado con partes de texto; no tiene fecha límite, ni periodicidad, ni evento disparador, ni régimen de cómputo.

Es exactamente el mismo agujero que tiene el `RequirementNode` de CISO Assistant, y es el agujero donde vive nuestro diferenciador entero: el reloj legal. Un motor de obligaciones con reloj no es un catálogo de controles con fechas pegadas encima, es otro modelo.

**Hacer ida y vuelta con OSCAL obliga a doblar nuestro modelo hasta que quepa en el suyo.** Y un modelo que cabe en OSCAL es un modelo sin plazos, o sea el producto de todos los demás. Por eso salida con pérdidas sí, y entrada no: exportar pierde los plazos y se dice que los pierde; importar nos obligaría a no tenerlos.

### Qué cambia en el repositorio

- `CLAUDE.md`: regla de arquitectura sobre OSCAL.
- `ETAPAS.md`: la casilla de equivalencias en OSCAL Mapping Model se mueve a etapa 6 o posterior; las equivalencias de etapa 3 se hacen en formato propio.
- `paquetes/CORPUS.md`: NIST 800-53 y CSF dejan de tener importador previsto.
- `docs/censo-relojes.md`: NIST 800-53 y CSF no entran en el orden de autoría.

---

## D-2. La capa probatoria queda cerrada

**Fecha:** 25-08-2026.

**Qué se decide:** del ataque 14 en adelante, los hallazgos de la familia "el emisor mete la mano en el expediente" **se documentan en `docs/modelo-de-amenaza.md`, no se arreglan**. Única excepción: que el hallazgo rompa la promesa escrita en ese fichero.

**Por qué:** coste de oportunidad. La capa probatoria está en 9,0 sobre 10 y puntúa 4,0 en decisión de compra. Las dimensiones que deciden la compra (D11 tiempo hasta el valor, D3 corpus, D17 experiencia) suman 20 de peso y están entre 3,0 y 4,5. Seguir puliendo el expediente es seguir puliendo lo que ya gana.

**Lo que va con la decisión:** el modelo de amenaza tenía que existir antes de cerrarla. Cerrar sin escribir qué se defiende y qué no habría dejado la promesa en comentarios de código, que es donde nadie la puede contrastar. Incluye explícitamente el **truncado de cola**, que no es detectable sin un testigo externo publicado y que no se va a montar porque rompe el autoalojado y el offline.

---

## D-3. Los frentes se ordenan por peso de compra, no por afinidad técnica

**Fecha:** 25-08-2026.

**Qué se decide:** delante van tiempo hasta el valor con accesibilidad en CI, censo de relojes, matrix de build con Docker, demo y doctor. Detrás van Litestream, export a SIEM, latido y vendorizar pkcs7.

**Por qué:** las tres dimensiones que deciden la compra suman 20 de peso y están entre 3,0 y 4,5. La capa probatoria está en 9,0 y puntúa 4,0 en decisión de compra. El orden sigue al comprador, no a lo que apetece construir.

---

## D-4. El nombre es PLAZUM

**Fecha:** 26-08-2026.

**Qué se decide:** el producto se llama **plazum**. **Implantado el mismo día**, entero: módulo Go, CLI, marca, documentos, web, dominio de compromiso del ledger (`plazum/commit/v1`) y expediente de demostración regenerado y resellado contra una TSA real. El candado de publicación se queda puesto, pero por otra razón: el nombre ya no bloquea, bloquea que publicar es irreversible y esa decisión es del dueño del proyecto.

**Cómo se llegó:** `vencia` y `preceptum`, los dos propuestos, salieron rojos con la misma forma que costó DUTIQ, una marca ajena ocupando la mayor parte del signo en clases idénticas: AVENCIA al 86% y PRECEPT (de Polestar) al 78%. Se generaron veinte más, cinco salieron limpias en EUIPO y las cinco siguen limpias en OEPM. La criba entera, con los umbrales y los números, en `docs/marca.md`.

**El hallazgo que cambió la elección:** `deontia` era mejor signo que `plazum` (más distintiva, y *lógica deóntica* es literalmente la lógica de la obligación) y salió limpia en los dos registros. Está descartada porque existe **Deontic** (deontic.ai, Lovaina, 2022), plataforma de IA para cumplimiento regulatorio: mismo sector, una letra.

**Lo que eso enseña, y que vale más que el nombre:** el cribador mira registros de marcas y sólo eso. No sabe de empresas en activo que operan sin registrar, y el uso anterior no registrado crea derechos. Es la misma clase de fallo que UTIQ en otro registro: buscar sólo donde es cómodo buscar. La herramienta lo dice ahora en su propia salida y hay un paso manual obligatorio después de cribar. No se automatiza porque no hay fuente gratuita y fiable de razones sociales de la Unión, y una automatización a medias daría el falso verde que la herramienta existe para no dar.

**Lo único abierto**, y es de dictamen, no de criba: `plazo` es descriptiva del servicio en español y el artículo 7.1.c del Reglamento de Marca de la Unión rechaza los signos descriptivos. `Plazum` no es una palabra española y exige un paso mental, que es la zona en la que un signo sugestivo sí se registra. Lo resuelve el agente de la propiedad industrial, no este documento.

