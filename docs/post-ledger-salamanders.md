# El ledger que no se puede contar dos veces

> **BORRADOR SIN PUBLICAR.** Escrito para acompañar a la v0.2. No sale hasta que el repositorio sea público, que a su vez espera a la comprobación de UTIQ en TMview (ver ETAPAS.md, semana 0).

Dutiq promete una cosa concreta sobre su expediente de cumplimiento, y la promesa es incómoda a propósito: **un tercero tiene que poder verificarlo entero, sin red, en su máquina, y sin fiarse de quien se lo dio**. No "confía en nuestra firma". No "nuestro SaaS lo garantiza". Recalcularlo desde cero y que cuadre, o no vale.

Esa frase parece marketing hasta que te sientas a construirla. Entonces descubres que la mitad de las primitivas que ibas a usar asumen justo lo contrario, que el que cifra y el que verifica están del mismo lado. Este post va de una de esas veces.

## El punto de partida

El ledger de dutiq es una cadena de entradas encadenadas por hash, con raíces Merkle publicadas y selladas contra una TSA externa. Cada entrada va cifrada, y la clave de cada entrada vive en un keystore separado de la cadena.

Lo de cifrar por entrada con clave separada no es paranoia decorativa, resuelve el borrado legal. Cuando llega un ejercicio del artículo 17 del RGPD, o el plazo de retención vence, borrar significa **destruir la clave de esa entrada** y añadir una lápida firmada con la base legal. La cadena no se toca. Las raíces ya publicadas siguen siendo válidas. El verificador informa "esta entrada está suprimida con base legal X" en vez de gritar "cadena manipulada". Es la única forma que encontré de que "cumplir con el derecho de supresión" y "el histórico sigue siendo verificable" no se contradigan.

El cifrado era AES-256-GCM. Es lo que uno pone. Es lo que está en todas partes.

## El problema

AES-GCM no es *key-committing*.

Dicho sin jerga: un mismo texto cifrado puede descifrar **correctamente, con el tag válido y sin error**, bajo dos claves distintas, dando dos textos claros distintos. No es un fallo de implementación ni una colisión que ocurre una vez cada mil años. Es una propiedad del modo, y se puede fabricar a voluntad. La construcción se conoce como *invisible salamanders*.

Y ahora súmalo a lo de arriba.

En dutiq, **el emisor del expediente controla las claves**. Es su instancia, su keystore. La promesa era que el receptor no tiene que fiarse de él. Pues bien, con GCM a secas, un emisor malicioso puede:

1. Fabricar una entrada cuyo cifrado descifra a "el control de MFA falló el 3 de marzo" bajo la clave A, y a "el control de MFA pasó el 3 de marzo" bajo la clave B.
2. Meterla en la cadena. El hash se calcula sobre la envoltura cifrada, así que la cadena es una y solo una.
3. Sellar la raíz Merkle contra la TSA. Ese sello es genuino y prueba la fecha.
4. Enseñarle al auditor la clave B. Todo verifica.
5. Enseñarle al juzgado la clave A. Todo verifica **igual de bien, contra la misma raíz sellada por el mismo tercero**.

El sello de tiempo no le ayuda a nadie: es auténtico en los dos casos. La cadena de hashes tampoco: solo hay una cadena. El tag de GCM tampoco: valida en los dos. El verificador dice "correcto" las dos veces y es cierto las dos veces.

El expediente se convierte en lo contrario de lo que dice ser. No es una prueba, es una superposición de dos historias, y el emisor elige cuál colapsa según quién pregunte.

## Por qué esto muerde aquí y no en tu chat

La no-committing de GCM lleva años en los papers y en casi ningún sitio importa, porque en casi ningún sitio el que cifra es el sospechoso. Si cifras tu propia base de datos para protegerla de un ladrón, la propiedad no te hace falta: nadie va a presentar una clave alternativa, y si lo hiciera, sería la tuya.

Aquí es distinto por dos razones que se refuerzan.

La primera es que el modelo de amenaza tiene al emisor **dentro**. Todo el producto se sostiene sobre "no te fíes de quien te da el expediente".

La segunda es más fea, y tardé en verla: **el borrado por destrucción de clave le da al ataque una coartada perfecta**. Si alguien pregunta por qué la clave A ya no está o por qué aparece otra, la respuesta legítima existe y es aburrida: retención, supresión, rotación. El mecanismo que necesito para cumplir el RGPD es el mismo que le da naturalidad a presentar claves distintas en momentos distintos.

## El arreglo

Comprometerse con la clave. Junto a cada entrada se guarda:

```
Compromiso = HMAC-SHA256(clave, "dutiq/commit/v1" || nonce)
```

Y al abrir, el orden importa:

```go
func AbrirComprometido(clave, nonce, cifrado, compromiso []byte) ([]byte, error) {
	if !hmac.Equal(compromisoDe(clave, nonce), compromiso) {
		return nil, ...   // se rechaza ANTES de descifrar
	}
	...
	return gcm.Open(nil, nonce, cifrado, []byte(etiquetaCompromiso))
}
```

El compromiso se comprueba **antes** de descifrar, no después. Una clave que no cuadra con el compromiso no llega a tocar el texto cifrado. Como el compromiso entra en el hash de la entrada, y el hash entra en la cadena, y la cadena entra en la raíz sellada, la clave queda atada a la raíz que firmó la TSA. Una entrada, una clave, un texto claro. La segunda historia deja de existir.

Dos detalles que no son adorno:

- **La etiqueta de dominio** (`"dutiq/commit/v1"`) va dentro del HMAC y además como datos autenticados adicionales del GCM. Sin dominio separado, un compromiso calculado para un propósito se puede reutilizar en otro. Va versionada porque el día que cambie el esquema hay que poder distinguir cuál se usó.
- **El nonce entra en el HMAC.** Comprometerse solo con la clave dejaría el compromiso reutilizable entre entradas.

Es HMAC de más por entrada. A cambio, la propiedad que el producto vende deja de ser falsa.

## Lo que no arregla

Conviene decirlo, porque un post que solo cuenta victorias no es un post técnico.

El compromiso ata **una clave a un texto cifrado**. No dice nada sobre si lo que se escribió era verdad. Si el emisor observa mal, o miente al escribir, el ledger registra fielmente una mentira y todo verifica. Contra eso no hay criptografía, hay procedencia de la evidencia, corroboración y conectores que recogen el dato sin que pase por las manos del interesado. Eso vive en otra parte del sistema y es un problema distinto.

Tampoco arregla que te roben la clave maestra del operador, ni que la TSA firme lo que no debe. Son otros supuestos, con otras respuestas.

## Cómo sé que está arreglado

Hay un test, y va con control negativo, que en este proyecto es obligatorio para toda propiedad de seguridad: **se demuestra que el test falla cuando debe**. No basta con que pase, hay que enseñar que sabe fallar. Si quito la comprobación del compromiso, `TestClaveSustituidaSeRechazaPorCompromiso` se pone rojo. Si no se pusiera, el test no estaría probando nada y sería peor que no tenerlo, porque daría tranquilidad falsa.

El verificador, además, corre bajo fuzzing nativo de Go en cada CI, con las semillas guardadas en el repositorio.

---

## Referencias

- Dodis, Grubbs, Ristenpart y Woodage, *Fast Message Franking: From Invisible Salamanders to Encryptment*, CRYPTO 2018 (eprint 2019/016). De aquí sale el término.
- Grubbs, *Hunting Invisible Salamanders: Cryptographic Insecurity with Attacker-Controlled Keys*, Black Hat USA 2020. La versión accesible, y la que mejor explica por qué "claves controladas por el atacante" es el supuesto que lo cambia todo.

Las mismas dos citas están en `nucleo/ledger/v2.go`, donde vive el arreglo.
