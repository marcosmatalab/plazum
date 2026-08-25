# TLS en plazum

Esta guía es para la persona que instala plazum, no para quien administra servidores todos los días. Va al grano, con la configuración que hay que pegar y las dos o tres cosas que se hacen mal siempre.

**Lo que hay que saber antes de nada.** plazum no funciona por http salvo en `localhost`. No es una manía, es cómo se comporta el navegador. La cookie de sesión de plazum lleva el atributo `Secure`, que le dice al navegador que no la devuelva por una conexión sin cifrar. Si entras por `http://grc.tuempresa.local:8443`, el navegador acepta la respuesta, se queda la cookie y no la vuelve a mandar nunca. El resultado es que introduces la contraseña, la pantalla vuelve al formulario de entrada, y no hay ningún mensaje de error en ningún sitio. plazum detecta ese caso concreto y te lo dice con letra, en vez de dejarte dando vueltas, pero la solución es la de esta guía.

Así que la primera decisión es una sola, y hay dos respuestas válidas.

---

## Camino recomendado, un proxy inverso delante

Es el camino recomendado porque el certificado se renueva solo, porque el proxy ya está en el inventario de la mayoría de organizaciones y porque separa dos cosas que envejecen a ritmos distintos, la configuración de TLS y la aplicación.

### Con Caddy, que es lo más corto que existe

Un fichero, `/etc/caddy/Caddyfile`, con esto dentro:

```
grc.tuempresa.es {
    reverse_proxy 127.0.0.1:8443
}
```

Caddy pide el certificado a Let's Encrypt solo, lo renueva solo y redirige http a https solo. Necesita que el puerto 443 y el 80 lleguen a esa máquina desde internet, porque así es como Let's Encrypt comprueba que el dominio es tuyo.

### Con nginx, si ya lo tienes

```nginx
server {
    listen 443 ssl;
    http2 on;
    server_name grc.tuempresa.es;

    ssl_certificate     /etc/letsencrypt/live/grc.tuempresa.es/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/grc.tuempresa.es/privkey.pem;
    ssl_protocols       TLSv1.2 TLSv1.3;

    location / {
        proxy_pass         http://127.0.0.1:8443;
        proxy_set_header   Host              $host;
        proxy_set_header   X-Forwarded-For   $proxy_add_x_forwarded_for;
        proxy_set_header   X-Forwarded-Proto $scheme;
        proxy_read_timeout 120s;
    }
}

server {
    listen 80;
    server_name grc.tuempresa.es;
    return 301 https://$host$request_uri;
}
```

Las tres cabeceras `proxy_set_header` no son decorativas. Sin `X-Forwarded-Proto`, plazum no sabe que delante hay TLS. Sin `X-Forwarded-For`, el límite de intentos de autenticación cuenta todas las peticiones como si vinieran del proxy, o sea de una sola dirección, y el primero que se equivoca de contraseña deja fuera a toda la oficina.

### Arrancar plazum detrás del proxy

Dos cosas que hay que decirle a plazum y que no puede adivinar.

**Cuántos proxies hay delante.** Uno, si el esquema es el de arriba. plazum ignora `X-Forwarded-For` por completo mientras no se lo digas, y eso es a propósito: esa cabecera la escribe cualquiera que mande una petición, así que creérsela por defecto sería regalar el límite de intentos. Basta con inventarse una cabecera distinta en cada intento y cada intento estrena contador. Contando los proxies, plazum lee la entrada que puso tu proxy y descarta las que pudo inventar el cliente.

Si tienes dos capas, por ejemplo un balanceador de la nube y detrás nginx, entonces son dos. Cuenta los que añaden la cabecera, no los que hay.

**Por qué nombres se entra.** La lista de hosts permitidos. La cabecera `Host` también la escribe el cliente, y una instalación que la acepta sin mirar es una instalación en la que un tercero puede envenenar lo que se genere a partir de ese nombre. Pon el nombre real y, si accedes también por la dirección interna, ponla al lado.

El servidor escucha solo en `127.0.0.1`, no en todas las interfaces. Con el proxy delante no hace falta más, y así nadie llega a plazum saltándose el proxy.

### Comprobar que ha quedado bien

Tres órdenes, un minuto.

```bash
curl -sI https://grc.tuempresa.es/salud | grep -i 'strict-transport\|content-security\|x-frame'
curl -sI http://grc.tuempresa.es/salud | head -1        # tiene que ser un 301 a https
curl -s  https://grc.tuempresa.es/salud                 # tiene que decir ok
```

Si la primera no devuelve las tres cabeceras, el proxy las está quitando. Algunos las filtran por configuración heredada. plazum las manda siempre.

---

## Alternativa, plazum termina TLS él mismo

Para cuando no hay ningún proxy y no lo va a haber. Funciona, pero el certificado lo renuevas tú.

Se le pasan dos rutas, el certificado y su clave, las dos en PEM. Si vienen de certbot, el certificado es `fullchain.pem` y la clave es `privkey.pem`. plazum comprueba que el par existe, que se puede leer y que la clave es la de ese certificado **antes** de ponerse a escuchar, así que un error aquí sale en el arranque y no en la primera visita.

Con el certificado puesto, plazum negocia TLS 1.2 como mínimo y deja los cifrados en manos de la biblioteca estándar de Go, que ya excluye los rotos y se actualiza con cada versión. No hay una lista de cifrados que mantener, y es deliberado, una lista escrita hoy es una lista envejecida dentro de dos años.

**plazum no pide certificados solo.** No hay ACME ni Let's Encrypt dentro del binario. La librería que hace eso es una dependencia nueva, y en este proyecto añadir una dependencia se decide por escrito y no de pasada. El rodeo, mientras tanto, es certbot por fuera:

```bash
certbot certonly --standalone -d grc.tuempresa.es
# y plazum apuntando a

#   /etc/letsencrypt/live/grc.tuempresa.es/fullchain.pem

#   /etc/letsencrypt/live/grc.tuempresa.es/privkey.pem

```

Certbot renueva cada 60 días con su propio temporizador. plazum lee el certificado al arrancar, así que después de una renovación hay que reiniciar el servicio. Un `--deploy-hook` con `systemctl restart plazum` lo resuelve.

Puerto: plazum no corre como root, así que no puede escuchar en el 443. Escucha en un puerto alto y redirige el 443 con el cortafuegos, o usa `AmbientCapabilities=CAP_NET_BIND_SERVICE` en la unidad de systemd. Correr el binario como root para ganar un puerto es un mal negocio.

---

## Lo que NO hay que hacer

**No pongas un certificado autofirmado y le digas a la gente que acepte el aviso del navegador.** Es el error más común y el más caro. Enseña a toda la organización a hacer clic en "entiendo el riesgo, continuar", que es exactamente el reflejo que un atacante necesita el día que el aviso sea de verdad. Y no ahorra trabajo: montar una autoridad interna cuesta lo mismo que poner Caddy delante, y Let's Encrypt es gratis. Si el nombre es interno y no resuelve desde internet, usa el desafío DNS de Let's Encrypt, que no necesita que la máquina sea accesible.

**No desactives el atributo Secure de la cookie para salir del paso.** plazum tiene una opción para hacerlo, existe para una prueba de media hora en una red de confianza, y avisa cada vez que arranca. Dejarla puesta significa que la sesión de la persona que aprueba los certificados de tu ISMS viaja en claro por la red de la oficina. Es el mismo trabajo que poner el proxy, y el resultado es peor.

**No termines TLS en el proxy y lo dejes escuchando en todas las interfaces.** Si plazum escucha en `0.0.0.0`, cualquiera dentro de la red llega al puerto alto directamente, sin pasar por el proxy, sin TLS y sin el límite de intentos que el proxy pudiera aportar. Escucha en `127.0.0.1` cuando el proxy está en la misma máquina, o cierra el puerto en el cortafuegos cuando no lo está.

**No pongas `preload` en la cabecera HSTS sin pensarlo dos veces.** plazum manda HSTS con dos años y subdominios incluidos, que es lo correcto, y deliberadamente sin `preload`. Entrar en la lista de precarga de los navegadores es sencillo y salir de ella tarda meses, y afecta a todos los subdominios del dominio, incluidos los que todavía no existen. Esa decisión es de la organización, no de una herramienta.

**No mandes tú las cabeceras de seguridad desde el proxy además de plazum.** Acabarás con dos `Content-Security-Policy` en la respuesta, y cuando hay dos el navegador aplica la intersección de las dos, que casi nunca es lo que nadie quiso. plazum ya las manda todas.

---

## El token del primer administrador, y dónde acaba

Cuando plazum arranca sin ningún administrador, imprime por la salida estándar un token de un solo uso. Con él, y una sola vez, se crea el primer administrador desde `/primer-admin`. Caduca en una hora.

**Si lo pierdes, para plazum y arráncalo otra vez.** Imprimirá uno nuevo y el anterior dejará de valer en el mismo momento. No hay forma de recuperarlo y es a propósito: un token recuperable es un token que alguien más puede recuperar.

**Y aquí va la parte incómoda, que preferimos decir a que la descubras.** Si plazum corre como servicio de systemd, su salida estándar la recoge el journal, que es persistente y lo lee cualquiera del grupo `systemd-journal`. O sea que el token queda escrito en un sitio del que no se borra solo. plazum detecta que su salida no es un terminal y lo avisa en la propia impresión, pero el aviso no lo arregla.

Lo que sí lo arregla, por orden de comodidad:

1. Arranca plazum **la primera vez a mano, en una terminal**, crea el administrador, párate ahí, y ya después instálalo como servicio. El token nunca llega al journal.
2. Si ya está como servicio, léelo con `journalctl -u plazum -n 50`, crea el administrador enseguida y después borra ese trozo de journal (`journalctl --rotate` y `journalctl --vacuum-time=1s` es la manera brusca; la fina es tener el journal en volátil durante la instalación).
3. En cualquier caso, en cuanto crees el administrador el token queda quemado. Aunque alguien lo lea después, ya no sirve para nada.

---

## Si hoy no puedes tener TLS

Ocurre. El certificado depende de un tercero que tarda tres semanas, y la evaluación es esta semana.

Mientras tanto, entra por `http://localhost:8443` **desde la propia máquina** con un túnel SSH:

```bash
ssh -L 8443:127.0.0.1:8443 tu-usuario@la-maquina
```

Y luego abre `http://localhost:8443` en tu portátil. El tráfico va cifrado por SSH, el navegador trata `localhost` como contexto seguro y devuelve la cookie, y no has tenido que tocar nada de la configuración de plazum. Es la forma correcta de evaluar el producto sin abrir un agujero, y para una prueba de una semana es suficiente.

---

## Resumen para pegar en el ticket

| Situación | Qué hacer |
|---|---|
| Hay o puede haber un proxy | Caddy o nginx delante, plazum escuchando en `127.0.0.1`, un proxy de confianza declarado y la lista de hosts puesta |
| No hay proxy y no lo habrá | certbot por fuera, las dos rutas PEM a plazum, reinicio en el `--deploy-hook` |
| Evaluación de una semana | Túnel SSH y `http://localhost` |
| Prueba de media hora en red de confianza | La cookie sin `Secure`, y se quita el mismo día |
| Certificado autofirmado | No |
