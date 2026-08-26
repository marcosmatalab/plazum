# El latido: que tu planificador no se muera en silencio

Un producto que vende "no pierdas nunca la conformidad" no puede morir callado.
El fallo que hay que cazar no es que el servidor se apague, porque eso se nota.
Es que el planificador deje de correr ciclos mientras todo lo demas sigue en
pie: la pantalla abre, el corpus esta instalado, y los plazos vencen sin que
nadie los mire. Desde fuera, una instalacion muerta y una instalacion sin nada
que vencer se ven exactamente igual.

Esto se arregla en tres capas. Solo la tercera nos toca a nosotros, y es la
menos importante.

## Capa 1, la pantalla Hoy

Arriba del todo de Hoy esta el estado del planificador, con tres estados
posibles.

- **Late.** Su ultimo ciclo termino hace menos de 24 horas.
- **Callado.** Lleva 24 horas o mas sin correr. La pantalla lo dice en rojo y
  con estas palabras, mientras siga parado los plazos vencen sin que nadie los
  mire.
- **No ha corrido nunca.** Recien instalado es lo normal. Si lleva dias
  instalado, no lo es.

Se calcula con el reloj de tu maquina y con la marca del ultimo ciclo, y no
consulta nada por la red. El limite de esta capa es evidente, hay que entrar a
mirarla.

## Capa 2, el comando y su codigo de salida

Esta es la que de verdad avisa.

```
plazum latido
```

Imprime el mismo veredicto que la pantalla y **termina con codigo 1 cuando el
planificador lleva mas de 24 horas callado**. Eso se engancha al monitor que ya
tengas, a un cron o a un temporizador de systemd, y **no depende de que
nosotros estemos vivos**. No sale a la red.

Quien corre los ciclos apunta que ha corrido:

```
plazum latido ciclo
```

Esa orden escribe la marca del ultimo ciclo y, si tienes el pulso encendido, lo
manda. Sale con codigo 0 aunque el pulso falle, y es a proposito, ver mas abajo.

El ciclo conviene tenerlo cada hora. **El pulso NO sale en cada ciclo, sale una
vez al dia**, que es lo que dice la casilla y lo que hay que mandar: el ciclo es
tuyo y cuanto mas a menudo mejor, el pulso sale de tu maquina y cuanto menos,
mejor. Si un pulso no llega, el siguiente ciclo lo reintenta sin esperar el dia
entero, porque lo que se compara es contra el ultimo pulso ACEPTADO.

### En systemd

```ini
# /etc/systemd/system/plazum-ciclo.service
[Unit]
Description=Ciclo del planificador de plazum

[Service]
Type=oneshot
User=plazum
WorkingDirectory=/var/lib/plazum
ExecStart=/usr/local/bin/plazum latido ciclo --datos /var/lib/plazum
```

```ini
# /etc/systemd/system/plazum-ciclo.timer
[Unit]
Description=Corre el ciclo de plazum cada hora

[Timer]
OnCalendar=hourly
Persistent=true

[Install]
WantedBy=timers.target
```

Y la vigilancia, que es lo que te avisa a ti:

```ini
# /etc/systemd/system/plazum-vigilancia.service
[Unit]
Description=Comprueba que el planificador de plazum sigue vivo
OnFailure=status-email@%n.service

[Service]
Type=oneshot
User=plazum
ExecStart=/usr/local/bin/plazum latido --datos /var/lib/plazum
```

Con `OnFailure` apuntando a tu unidad de correo, el dia que el planificador
lleve 24 horas parado recibes un correo tuyo, de tu maquina, sin nosotros en
medio.

### En cron

```cron
0 * * * *  plazum latido ciclo --datos /var/lib/plazum
15 8 * * * plazum latido --datos /var/lib/plazum
```

La segunda linea no imprime nada relevante mientras todo va bien, y el dia que
falla sale con codigo 1 y cron te manda su salida por correo.

## Capa 3, el pulso (opt-in, apagado de fabrica)

Cubre el unico caso que las dos anteriores no pueden cubrir, que la maquina
entera muera y no quede nadie que corra el cron.

**Viene apagado y solo se enciende a mano.** Se enciende en dos pasos, igual
que `plazum update` no actualiza sin `--aplicar`:

```
plazum latido activar             # ensena lo que se mandaria, y no manda nada
plazum latido activar --acepto    # lo enciende y registra el consentimiento
```

El consentimiento queda escrito en `<datos>/latido.json`, con la fecha y con el
texto literal que aceptaste, para que puedas releerlo dentro de un ano.

### Que se manda, entero

    Un identificador aleatorio de esta instalacion, que generamos aqui y no se
    deriva de nada tuyo, y el instante del pulso.
    Nada mas: ni tu nombre, ni el de tu organizacion, ni tu direccion, ni que
    paquetes normativos tienes, ni nada de tu estado de cumplimiento.

Son literalmente dos campos, `instancia` e `instante`:

```json
{"instancia":"9f2c...","instante":"2026-08-26T09:00:00Z"}
```

Si eso deja de ser cierto, el build se pone rojo. Hay un test que compara lo
que sale contra una **lista blanca** (la pregunta no es si va algo prohibido,
es si va exactamente eso y nada mas), y otro que comprueba que el paquete del
latido **no puede** leer el corpus ni tu estado de cumplimiento, porque ni
siquiera los importa.

El identificador es aleatorio y se genera en tu maquina. No sale del nombre del
equipo, ni de una MAC, ni de nada tuyo. Al desactivar se borra, asi que si algun
dia vuelves a encenderlo sera con otro distinto y nadie puede enlazar el antes
con el despues.

### Lo que el pulso NO hace, dicho claro

**No te avisa a ti.** No tenemos tu correo y no lo queremos, asi que si tu
instancia deja de pulsar hacia nosotros, nosotros no tenemos forma de decirtelo.
Eso no es un descuido, es la consecuencia de no pedirte datos.

Si quieres que el pulso te avise, apuntalo a tu propio monitor:

```
plazum latido activar --destino https://monitor.interno.example/plazum --acepto
```

Cualquier receptor que acepte un POST de JSON sirve, incluido un
"dead man's switch" de los que avisan cuando el pulso deja de llegar. Entonces
el aviso es tuyo, de tu sistema, y nosotros no estamos en el camino.

## La direccion del aviso, que es lo unico que importa de todo esto

**Nuestra caida no puede leerse como la tuya.** Si el veredicto del
planificador dependiera de que nuestro receptor conteste, cada vez que se nos
cayera un servidor tu pantalla se pondria en rojo, tu no encontrarias nada roto
en tu maquina, y en dos semanas habrias aprendido a ignorar el rojo. Un aviso
que se ignora es peor que no tener aviso, porque ademas da tranquilidad.

Por eso, y esta escrito en tests que se ponen rojos si alguien lo cambia:

- el veredicto del planificador **no mira el canal**, ni una sola de sus reglas;
- el estado del canal se informa aparte y **nunca en rojo**, como mucho en
  amarillo, y siempre con la frase que dice que lo que calla es el canal hacia
  nosotros y no tu planificador;
- `plazum latido ciclo` **apunta el ciclo aunque el pulso falle**, porque si un
  fallo de red hacia nosotros dejara sin escribir esa marca, al dia siguiente tu
  pantalla te diria que tu planificador lleva 24 horas muerto;
- `plazum latido ciclo` sale con **codigo 0 aunque el pulso falle**, porque si
  saliera con 1 tu cron te mandaria un correo cada vez que a nosotros nos vaya
  mal, y esos correos acaban en una regla de filtrado junto con el que si
  importaba.

## El smoke test del canal

```
plazum latido probar
```

Manda un pulso de verdad, por el canal de verdad, y te dice si el destino lo
acepta. Aqui si sale con codigo 1 cuando el canal no entrega, porque has
preguntado por el canal. El resultado queda apuntado y la pantalla Hoy lo
ensena, asi que no hay que esperar 24 horas para ver que el canal esta roto.

Se manda un pulso normal y no uno "de prueba" a proposito, un smoke test que usa
un camino distinto del real prueba el camino distinto.

## El destino, y lo que se rechaza

Por defecto `https://plazum.dev/latido` (dominio provisional). Se rechaza, al
activar y no en el primer pulso de madrugada:

- **http contra un tercero.** El pulso sale de tu red, va por https. Se admite
  http solo contra localhost, para que puedas probar tu propio receptor.
- **Una parte de consulta** (`?loquesea=`). Ahi es justo donde se cuela un
  identificador sin que nadie lo note, y ademas acaba en los logs de cada
  intermediario del camino.
- **Usuario y contrasena en la direccion.** Es un secreto en un fichero de
  configuracion y en los logs.
- **Un fragmento** (`#loquesea`). No significa nada en una peticion.

Y el canal **no sigue redirecciones** y **no guarda cookies**. Una redireccion
mueve el pulso a otra maquina, o le anade una parte de consulta, sin que tu te
enteres, y un receptor que pone una cookie convierte pulsos anonimos en una
sesion.

## Ficheros

`<datos>/latido.json`, con permisos 0600. Dentro: si esta activado, el
identificador, el destino, el consentimiento con su fecha y su texto, la marca
del ultimo ciclo y la del ultimo pulso. Si lo borras, vuelves al estado de
fabrica, apagado.
