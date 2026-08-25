# Conectar tu IdP a plazum: OIDC y SCIM

Esta guía está escrita para quien lo va a hacer, no para quien lo programó. Va
dirigida al administrador de sistemas que un martes por la tarde tiene que
conectar el Entra ID o el Okta de su empresa y quiere haber terminado antes de
irse a casa.

Son dos cosas distintas y conviene no mezclarlas:

- **OIDC** es **entrar**. Quién es esta persona y si puede abrir sesión.
- **SCIM** es **aprovisionar**. Qué personas existen, cuál es su jefe, y quién
  ha dejado de trabajar aquí.

Puedes tener OIDC sin SCIM, y entonces nadie podrá entrar hasta que su cuenta
exista en plazum. Puedes tener SCIM sin OIDC, y entonces tendrás el directorio al
día y nadie entrará por el IdP. Lo normal es querer los dos.

---

## Parte 1: OIDC, para entrar

### Lo que tienes que pegar, y de dónde sale

| Campo en plazum | Qué es | Dónde está en Entra ID | Dónde está en Okta |
|---|---|---|---|
| `emisor` | El `issuer` | `https://login.microsoftonline.com/<id-de-tenant>/v2.0` | `https://<dominio>.okta.com/oauth2/default` |
| `cliente_id` | El identificador de la aplicación | Application (client) ID, en Overview | Client ID, en General |
| `cliente_secreto` | El secreto de la aplicación | Certificates & secrets, valor del secreto | Client Secret, en General |
| `redirect_uri` | A dónde vuelve el navegador | lo registras tú | lo registras tú |

La `redirect_uri` la eliges tú y tiene que ser `https://<tu-plazum>/auth/retorno`.
Cópiala **carácter a carácter** en el IdP, barra final incluida. Es el error más
frecuente de todos y el IdP no te dirá cuál de los dos caracteres sobra.

### Lo que plazum hace por su cuenta

No hay que configurar nada de esto, se dice para que sepas qué esperar:

- Lee el documento de descubrimiento en `<emisor>/.well-known/openid-configuration`
  y comprueba que el `issuer` que declara es exactamente el que pegaste. Si no
  coinciden, no arranca.
- Comprueba que los endpoints de autorización, token y JWKS cuelgan del mismo
  host que el emisor.
- Usa **PKCE con S256 siempre**, también cuando hay secreto de cliente.
- Genera un `state` y un `nonce` por intento, los ata entre sí, y los invalida en
  cuanto se usan una vez.
- Verifica la firma del ID token contra el JWKS, con lista blanca de algoritmos.
  `alg: none` y las familias HMAC no se aceptan nunca.

### Errores que vas a ver, y qué significan de verdad

**`invalid_client`**. El IdP no reconoce las credenciales. Casi siempre es el
secreto. En Entra ID los secretos **caducan**, y el día que caduca deja de
funcionar sin que nadie avise. Ve a Certificates & secrets, mira la fecha, y
genera uno nuevo si ya pasó.

**`invalid_grant`**. El código de autorización no vale. Tres causas, por orden de
frecuencia: alguien refrescó la página de retorno y el código ya se canjeó, el
código caducó, o la `redirect_uri` del canje no es idéntica a la registrada.

**`el emisor del token es X y se esperaba Y`**. Los dos se parecen y no son
iguales. Lo más común es una barra final de más o de menos. Se comparan byte a
byte a propósito: un emisor parecido es un emisor distinto.

**`la audiencia del token no incluye el cliente_id`**. El `cliente_id` pegado no
es el de esta aplicación. Suele pasar cuando hay dos registros de aplicación en
el tenant y se copió el de otro.

**`el token no vale hasta las ...`** de forma repetida. El reloj de la máquina de
plazum y el del IdP no están sincronizados. El arreglo es NTP, no subir el margen
de reloj. El margen está limitado a cinco minutos por diseño: un margen grande
es una caducidad grande, y un token robado seguiría valiendo todo ese tiempo
después de expirar.

**`no vino id_token`**. El registro de la aplicación es OAuth 2.0 y no OpenID
Connect, o falta el ámbito `openid`.

### Si el IdP rota sus claves

No hay que hacer nada. Cuando llega un token firmado con una clave que plazum no
conoce, recarga el JWKS y sigue. La recarga está limitada a una por minuto: sin
ese límite, cualquiera podría hacernos bombardear a tu IdP mandando tokens con
identificadores de clave inventados.

---

## Parte 2: SCIM, para aprovisionar

### Lo que tienes que pegar

| Campo en el IdP | Valor |
|---|---|
| Tenant URL / SCIM connector base URL | `https://<tu-plazum>/scim/v2` |
| Secret Token | el token de aprovisionamiento de tu instancia |

El token lo genera plazum. Pégalo **entero y sin la palabra `Bearer` delante**: el
IdP la añade solo. Si lo pegas con `Bearer`, todas las peticiones darán 401 y el
mensaje dirá que la credencial es inválida, que es verdad pero no ayuda mucho.

Sin token no hay servidor SCIM. No es una opción que se pueda dejar vacía: la
instancia no arranca. Este endpoint da de alta y de baja a personas, y sin
credencial sería una puerta trasera con forma de estándar.

### Lo que soporta, dicho sin adornos

Se dice lo que hay porque declarar capacidades que no se tienen es peor que no
declararlas: el IdP se fía, las usa, y el aprovisionamiento falla a mitad de
ciclo.

- **Users**: crear, leer, listar, sustituir (PUT), modificar (PATCH) y borrar.
- **Groups**: crear, leer, listar, modificar miembros y borrar.
- **Extensión enterprise**: `manager`, `department` y `employeeNumber`.
- **Filtros**: solo `atributo eq "valor"` sobre `userName`, `externalId` y
  `displayName`. Cualquier otro filtro devuelve un error explícito. Es
  deliberado: un filtro que se ignora en silencio devuelve la lista entera, y
  entonces el IdP concluye lo contrario de lo que preguntó y empieza a duplicar
  usuarios.
- **No soportado, y así lo declara `/ServiceProviderConfig`**: Bulk, ordenación
  por `sortBy`, y cambio de contraseña.

### Dos atributos que plazum rechaza a propósito

Si tu mapeo de aprovisionamiento los incluye, quítalos, porque las peticiones
que los lleven fallarán enteras.

- **`roles` y `entitlements`**. El rol dentro de plazum se asigna dentro de plazum.
  Si el aprovisionamiento pudiera mandarlos, quien controle el token de SCIM
  controlaría los privilegios, y ese token vale entonces lo que una cuenta de
  administrador.
- **`password`**. plazum no tiene contraseñas propias, la autenticación es OIDC.
  Aceptar una contraseña crearía una segunda vía de entrada que nadie vigila.

### Qué pasa cuando das de baja a alguien

Esto importa más de lo que parece, así que está escrito con detalle.

**Si lo desactivas** (`active: false`): deja de poder entrar inmediatamente. Su
ficha sigue existiendo.

**Si lo borras** (DELETE): deja de devolverse por GET y por LIST, como manda la
especificación, y deja de poder entrar. Su `userName` queda libre.

**En los dos casos, sus obligaciones NO desaparecen.** Quedan visibles marcadas
como huérfanas, con el nombre de quien las tenía y la fecha de la baja. Una
obligación sin responsable es un riesgo, y hacerla invisible convierte un riesgo
en un problema aparentemente resuelto. Junto a cada huérfana verás a quién
escalarla, que sale de la jerarquía.

---

## Parte 3: el atributo `manager`, y qué hacer si tu IdP no lo publica

### Por qué importa

De ahí sale la **jerarquía de escalado**. Cuando una obligación vence y su
responsable no responde, plazum sube el aviso a su jefe. Sin jerarquía, el
escalado es una lista de correos escrita a mano que se queda obsoleta el primer
día.

### Si tu IdP lo publica

En **Entra ID**: en Provisioning, Attribute Mapping, añade el mapeo de `manager`
a `urn:ietf:params:scim:schemas:extension:enterprise:2.0:User:manager`. Ojo con
el orden: el jefe tiene que estar aprovisionado antes que la persona que le
reporta. Si no lo está, ese usuario dará un 400 con un mensaje que lo explica, y
el IdP lo reintentará solo en el ciclo siguiente. No hay que hacer nada.

En **Okta**: en Provisioning, To App, comprueba que `managerId` está mapeado.

### Si tu IdP no lo publica

Es la mitad de los casos, y está previsto. Declara la jerarquía a mano desde la
pantalla de Personas. Lo declarado a mano:

- Pasa por **las mismas comprobaciones** que lo que llega del IdP. No es un
  sistema paralelo, es el mismo con otro origen.
- Queda **marcado como manual**, con quién lo declaró y desde cuándo. Podrás
  distinguir siempre lo que dice tu IdP de lo que escribiste tú.
- **Cede ante el IdP.** Si más adelante el IdP empieza a publicar `manager` para
  esa persona, manda el IdP, y el caso aparece en la lista de conflictos para
  que lo veas en lugar de enterarte dentro de seis meses.

### Ciclos

Si intentas poner a alguien como jefe de su propio jefe, la petición se rechaza
con un 400 que dibuja el ciclo entero. También se rechaza que alguien sea su
propio jefe. No es purismo: un ciclo cuelga el escalado, y una obligación
vencida que sube en círculo no avisa a nadie.

---

## Parte 4: comprobar que está funcionando de verdad

Un aprovisionamiento que nunca llegó a conectarse y uno que funciona bien se
parecen mucho desde fuera: en los dos casos no pasa nada raro. Por eso hay
comprobaciones explícitas en `plazum doctor` y en la pantalla de estado:

| Comprobación | Qué te dice |
|---|---|
| `scim-conexion` | Si el IdP ha llegado a llamar alguna vez, y cuándo fue la última vez que completó una petición correcta. Si lleva más de 25 horas sin conseguirlo, se pone en rojo: un aprovisionamiento parado significa que quien salió de la empresa conserva el acceso |
| `scim-credencial` | Cuántas peticiones se han rechazado por credencial inválida. Un goteo suele ser un token viejo de una configuración anterior; un chorro con nada pasando significa que el token pegado no es el de esta instancia |
| `scim-jerarquia` | Cuántas personas activas no tienen jefe conocido. Para esas, el escalado no sube a nadie |
| `scim-jerarquia-conflictos` | Personas cuyo jefe declarado a mano no coincide con el que publica el IdP |
| `scim-jerarquia-rotas` | Personas que se quedaron sin cadena de escalado porque su jefe salió de la empresa |

La prueba rápida, en este orden:

1. Pulsa **Probar conexión** en el IdP. Si da 401, el token no es el de aquí.
2. Asigna la aplicación a una persona y espera al ciclo (40 minutos en Entra ID,
   una hora en Okta) o lanza la sincronización a mano.
3. Mira `plazum doctor`. `scim-conexion` tiene que estar en verde.
4. Entra tú por OIDC. Si no te deja, el mensaje te dirá si es que tu cuenta no
   está aprovisionada, si está desactivada, o si es un problema del token.

---

## Lo que todavía no hay

Se dice para que no lo busques:

- **SAML**. Está apuntado para el año 2. Hoy solo OIDC.
- **Filtros compuestos de SCIM** (`and`, `or`, `co`, `sw`). Solo `eq` sobre tres
  atributos.
- **Bulk de SCIM**. Ningún IdP de los que importan lo usa para aprovisionar.
- **Endpoint `/Me`**.

---

Este documento no es asesoramiento jurídico.
