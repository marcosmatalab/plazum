# Notarización: diseño, sin construir

> **Estado a 26-08-2026: DISEÑO. No hay una línea de código de esto y no la va a haber hasta que la etapa 3 esté cerrada.** Este documento existe para que la decisión esté tomada por escrito antes de que haya nada que rehacer, y sobre todo para fijar lo que **nunca** se puede decir de este servicio.

---

## La frase exacta, y por qué se escribe antes que nada

Todo lo que este servicio comunique, dentro y fuera del producto, dice esto y sólo esto:

> **sello de tiempo cualificado de un QTSP tercero, más contrafirma y anotación en registro público**

**Lo que NUNCA se dice, en ningún material, ni de pasada:**

- que plazum es un servicio de confianza;
- que plazum es un prestador cualificado;
- que plazum emite sellos cualificados;
- «nuestro sello cualificado», «sellamos con validez eIDAS», «somos QTSP» o cualquier variante.

**Por qué esto va antes que el diseño técnico.** No somos un QTSP y llegar a serlo es un procedimiento de conformidad ante el organismo de supervisión, con auditoría de un organismo de evaluación acreditado y una lista de confianza en la que hay que aparecer. Afirmar la condición sin tenerla, **en un producto de cumplimiento**, no es un exceso de marketing: es exactamente el tipo de afirmación que este producto existe para detectar en otros. El día que un auditor lo mire, no se cae la frase: se cae la tesis entera del producto.

El QTSP es el tercero. Nosotros ponemos el pegamento y el registro.

---

## Qué hace, en cuatro pasos

```
  instancia del cliente                      servicio de notarizacion
  ---------------------                      ------------------------
  1. calcula el hash de la
     cabeza de su cadena
                            --- 32 bytes -->
                                             2. pide sello RFC 3161 a un
                                                QTSP tercero sobre ESE hash
                                             3. contrafirma (hash + sello) con
                                                la clave del servicio
                                             4. anota la entrada en un registro
                                                de SOLO ANADIR
                            <-- sello + ---
                                recibo
  5. guarda sello y recibo
     junto al expediente
```

## Requisito de diseño, no opción: sólo viaja el hash

**La instancia manda 32 bytes y nada más.** Ni una obligación, ni un plazo, ni un nombre, ni un identificador de paquete, ni el tamaño de la cadena, ni cuántas entradas tiene. El hash de la cabeza y punto.

Esto no es una opción de privacidad que se configura: **es la condición sin la cual esta línea no existe.** El motivo está escrito en la decisión D-5 y es el mismo por el que el Cloud sale del camino crítico: un producto autoalojado cuya tesis es *el receptor no se fía del emisor* no puede pedirle al comprador que le mande el mapa de sus incumplimientos. Si mañana el servicio necesita un dato más para funcionar, el servicio está mal diseñado.

Consecuencia práctica que conviene tener presente: **el servicio no puede ayudar a diagnosticar nada.** No sabe qué se ha sellado. Eso es una limitación aceptada, no un defecto que arreglar.

## El registro tiene que salir de nuestras manos

Un registro de solo añadir que custodiamos nosotros vale exactamente lo que nuestra palabra, y la tesis del producto es que la palabra del emisor no basta. **La cabeza del registro se publica periódicamente en un sitio fuera de nuestro control** (un repositorio público, un log de transparencia, o los dos). Sin esa publicación, la notarización es un sello bonito que no cierra nada.

---

## Qué cierra, y qué no

**Cierra el ataque 14 sin romper el offline.** El expediente que verifica hoy sigue verificando exactamente igual sin notarizar: nada de esto entra en el camino de verificación, y una instalación desconectada no pierde ni una propiedad. Lo que la notarización **añade** es la prueba de que el expediente no está antedatado: que la cadena tenía esa forma en un instante que no lo elige el emisor.

**Lo que sigue sin cerrar**, y va a `docs/modelo-de-amenaza.md` con estas palabras: la notarización prueba que **lo notarizado existía en ese momento**. No prueba que lo notarizado sea todo. Un emisor que notariza y luego corta la cola sigue sin ser detectable salvo que alguien compare contra una notarización posterior; lo que la notarización hace es convertir «no detectable» en «detectable si el receptor guarda un recibo», que es una frontera distinta y mejor, pero es una frontera.

---

## Lo que falta decidir, y no se decide aquí

- **Qué QTSP.** Hay que elegir uno de la lista de confianza y leer su precio por sello y su límite de peticiones.
- **Cada cuánto se notariza.** Por cabeza de cadena, por día, por checkpoint. Afecta al coste directamente.
- **Qué pasa cuando el QTSP cae.** El servicio no puede prometer disponibilidad que no controla.
- **Dónde se publica la cabeza.** Repositorio propio, log de transparencia de terceros, o los dos.

Ninguna de estas cuatro bloquea el diseño de arriba, y por eso se dejan abiertas en vez de inventadas.
