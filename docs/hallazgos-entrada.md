# Hallazgos del frente de la entrada

Frente A de la campana del 03-09-2026. Columna: `adaptadores/usuarios/`,
`superficies/serve/`, `cmd/plazum/`, `adaptadores/catalogo/cadenas/` y este
fichero.

El P0 era este: **hoy nadie puede entrar en plazum**. Este documento anota lo
que se encontro al arreglarlo, lo que quedo fuera con su cardinal, y las dos
cosas que hay que tocar en columnas ajenas.

---

## 1. El P0, medido antes y despues

Sobre el binario de verdad (`go build ./cmd/plazum`), arrancado en un directorio
vacio con `plazum serve --corpus paquetes`:

| ruta | antes | despues, sin instalar | despues, instalado |
|---|---|---|---|
| `/` | 303 | 303 a `/primer-admin` | 200 |
| `/alcance` | 200 | 303 a `/primer-admin` | 200 |
| `/calendario/` | 200 | 303 a `/primer-admin` | 200 |
| `/controles` | 200 | 303 a `/primer-admin` | 200 |
| `/acta/` | **401** | 303 a `/primer-admin` | 200 |
| `/uar/` | **401** | 303 a `/primer-admin` | 200 |
| `/escalado/` | **401** | 303 a `/primer-admin` | 200 |
| `/primer-admin` | **503** | 200, con formulario | 303 a `/entrar` |
| `/entrar` | formulario inutil | 303 a `/primer-admin` | 200 |

La causa era exacta y de una linea: `cmd/plazum/serve.go` construia
`serve.Config` sin `Autenticar`, sin `HayAdmin` y sin `CrearAdmin`, y el valor
cero de esos tres campos es **denegar y decirlo**. El mecanismo de entrada
estaba entero y probado en `superficies/serve` desde semanas antes.

**Ninguna puerta lo veia porque cada mitad pasaba la suya.** Es la misma familia
que la primitiva `maximo` encendida en el motor y apagada para el corpus, y que
`superficies/acta` construida y sin cablear.

---

## 2. Lo que se construyo

### `adaptadores/usuarios`

Almacen de cuentas en un fichero JSON con version, sal por usuario y derivacion
lenta.

**Cero dependencias nuevas, y es una decision, no una casualidad.**
`DEPENDENCIAS.md` dice que el binario se compila con CERO dependencias externas
y que el dia que entre la primera hay que cambiar
`TestElBinarioNoLlevaNingunaDependenciaExterna` a proposito, en el mismo commit
que su fila. `golang.org/x/crypto` esta en la tabla como **planeada**, no como
puesta. No hizo falta: **Go 1.24 subio PBKDF2 a la biblioteca estandar**
(`crypto/pbkdf2`, RFC 8018), y `crypto/sha256` y `crypto/subtle` ya estaban.

Los parametros: PBKDF2-HMAC-SHA256, **600.000 iteraciones** (la recomendacion de
OWASP para esa combinacion), sal de 16 bytes por usuario, clave de 32.
Medido en esta maquina: **79 ms por derivacion**.

**Lo que PBKDF2 no da, dicho en voz alta**: dureza frente a hardware dedicado.
Eso lo dan scrypt y argon2, que son memory-hard, y ninguno esta en la biblioteca
estandar. El dia que el producto quiera esa propiedad, la decision es anadir la
dependencia con su fila en `DEPENDENCIAS.md` y su cambio del test, no fingir que
ya la tiene.

**El coste va dentro del fichero, con suelo.** Cada cuenta guarda su algoritmo y
sus iteraciones, para poder subir el coste sin invalidar las cuentas viejas. Y
`Abrir` rechaza cualquier cuenta por debajo de `IteracionesMinimas` (210.000):
sin ese suelo, bajar el numero a mano seria un downgrade silencioso.

### Las tres formas de la nada (invariante 8)

| forma | que significa | que hace |
|---|---|---|
| fichero **ausente** | instalacion nueva | almacen vacio, sin error |
| fichero **presente y vacio** | fichero roto | `ErrAlmacenVacio`, plazum no arranca |
| fichero **presente y no interpretable** | dato que hay y no se entiende | error, nunca el valor por defecto |

**El caso del medio es el que importa y no es obvio.** Un fichero de cero bytes
donde deberia haber cuentas NO es una instalacion nueva: es una escritura que se
corto, un disco lleno, o un `> usuarios.json` de alguien. Leerlo como «no hay
administrador» **reabre la ventana del primer administrador en un sistema que ya
estaba instalado**, o sea que convierte un fichero truncado en una toma de
control. El valor cero en esa frontera tiene que ser el restrictivo, y el
restrictivo es negarse.

Y encaja con lo que `superficies/serve` ya hacia: `anunciar` se niega a arrancar
si `HayAdmin` devuelve error, con el comentario «imprimir un token de instalacion
sin estar seguro abriria una puerta en un sistema que a lo mejor ya esta
instalado». El centinela llega ahi solo.

### Las dos puertas nuevas (`cmd/plazum/entrada_test.go`)

- **`TestPlazumServeCableaTodaDecisionDeIdentidadDeServeConfig`** (estructural).
  Enumera las decisiones de identidad **leyendo el AST de `serve.Config`**: los
  campos cuyo tipo es una funcion cuyo primer parametro es un `context.Context`.
  Hoy son exactamente `Autenticar`, `HayAdmin` y `CrearAdmin`; `Reloj` es
  `func() time.Time` y `Salida` es un `io.Writer`, asi que no entran. Cruza ese
  conjunto con los literales `serve.Config{...}` de `cmd/plazum` y exige que
  esten todos, y que ninguno vaya escrito a `nil`.

  **No lleva lista.** Un cuarto gancho de identidad que se anada manana entra
  solo en la puerta. El criterio es la FIRMA y no el comentario, porque un
  comentario se reescribe sin querer.

- **`TestUnaInstalacionNuevaRecorreLosSeisPasosDelCamino`** (conductual). Levanta
  `plazum serve` sobre un directorio vacio, lee el token del recuadro que imprime
  el arranque, crea el administrador por HTTP como haria el navegador y recorre
  los seis pasos de `camino.Canonico()`. Los pasos se enumeran del producto, no
  de una lista.

- **`TestSinSesionLasPantallasConNombresDePersonasSiguenSinServirse`** es el
  control positivo del descargo: con la aplicacion ya instalada, `/acta/`, `/uar/`
  y `/escalado/` no pueden salir con 200 sin cookie. Sin el, el arreglo mas comodo
  para poner verde la puerta de arriba seria abrir las tres pantallas que llevan
  nombres de personas, y nada se pondria rojo.

**Por que hacen falta las dos**: la estructural no sabe si las funciones que se
pasan sirven de algo; la conductual no puede decir cual de los tres ganchos falta
cuando falla.

---

## 3. Un verde falso que llevaba tiempo puesto, y como salio

`TestPlazumServeLevantaLaInterfazYResponde` y
`TestLaInterfazSaleConTextoYNoConClavesEnCrudo` pedian las pantallas con un
`http.Client` por defecto, **que sigue redirecciones**. Al cablear la entrada,
las seis pantallas pasaron a contestar 303 a `/primer-admin`... y los dos tests
**siguieron en verde**: el cliente iba detras del 303, encontraba el 200 de
`/primer-admin`, veia su `<html>` y su `<h1>` y daba por servida una pantalla que
nadie estaba sirviendo.

O sea que durante ese rato la suite de `plazum serve` habria dado por bueno un
producto en el que ninguna pantalla se servia. Arreglado: los clientes de
`arrancarServe` **no siguen redirecciones**, ninguno de los dos, y el test mira
el codigo que de verdad contesta el servidor.

Es la misma familia que la trampa de `go test -run` que no casa: un cliente que
va detras del error convierte cualquier respuesta en un 200.

---

## 4. LO QUE HAY QUE TOCAR EN OTRA COLUMNA

### 4.1 `.github/workflows/etapa2-accesibilidad.yml` — SE VA A PONER ROJO

**Este es el punto que hay que leer antes de fusionar.** El fichero no esta en la
columna de este frente y no se ha tocado.

El paso «puerta de accesibilidad sobre las pantallas que sirve el producto»
arranca `plazum serve` sobre el repositorio (sin cuentas), pide `/` y saca las
pantallas del `<nav id="navegacion">`. Desde este commit, `/` en una instalacion
sin administrador **redirige a `/primer-admin`**, que no tiene ese `<nav>`. La
extraccion daria cero pantallas y el paso se pondria rojo con «el menu del
producto ofrece 0 pantallas y la etapa 2 declara 6».

El propio workflow ya lo tenia anotado como trabajo pendiente:

> Para cubrirlas hace falta que este paso ENTRE: leer el token de primer
> administrador que imprime el arranque, crear la cuenta, guardar la cookie y
> pasarsela a puppeteer. Es trabajo real y va apuntado, no disimulado.

**Ya no es opcional.** El arreglo es este, y las ordenes estan copiadas de un
recorrido que se ejecuto de verdad contra el binario (no escritas de memoria):

```bash
# despues de arrancar plazum y antes de descubrir el menu
TOKEN=$(grep -oE '^ {5}[0-9a-f]{64}$' serve.log | tr -d ' ')
[ -n "$TOKEN" ] || { echo "PUERTA ROTA: plazum no imprimio token de instalacion"; exit 1; }

curl -s -c galletas.txt -o form.html http://127.0.0.1:8099/primer-admin
CSRF=$(grep -oE 'name="csrf" value="[0-9a-f]+"' form.html \
       | head -1 | grep -oE '[0-9a-f]{32,}')
codigo=$(curl -s -o /dev/null -w '%{http_code}' -b galletas.txt -c galletas.txt \
  -d "csrf=$CSRF" -d "token=$TOKEN" -d "usuario=ciso" -d "secreto=contrasena-de-ci-1" \
  http://127.0.0.1:8099/primer-admin)
[ "$codigo" = 303 ] || { echo "PUERTA ROTA: no se pudo crear el administrador ($codigo)"; exit 1; }

# y a partir de aqui, todas las peticiones con -b galletas.txt
```

Con eso ademas se puede cerrar la deuda que el propio fichero declara: auditar
`/uar/` y `/acta/`, que hoy quedan fuera porque exigen sesion. Puppeteer necesita
la cookie: `pagina.setCookie({name, value, domain: "127.0.0.1"})` con lo que
quede en `galletas.txt`. **Y hay que auditar tambien `/primer-admin`**, que desde
hoy es la PRIMERA pantalla que ve cualquiera en una instalacion nueva, y no esta
en la lista.

Los otros dos pasos del mismo fichero **no se rompen**: el de arranque solo mira
que el puerto acepte, y el de RAM usa `curl -fsSL`, que sigue el 303 y sigue
contando 200 respuestas.
`etapa2-distribucion.yml` tampoco: acepta 303 o 200 en la raiz.
`etapa2-seguridad-web.yml` tampoco: levanta su propio `servidorprueba`.

### 4.2 `ttfv_camino_test.go` — FRONTERA ROTA, a proposito y dicho

**Es del frente C.** `.github/frontera.sh` (tras el rebase sobre `main`) declara
`frente_C="superficies/pantallas/ nucleo/corpus/ ttfv_camino_test.go
docs/hallazgos-entrevista.md"`, y `.github/frontera.sh A main <rama>` lo saca
por su nombre: **un fichero fuera de la columna, y es este**.

Se toco igualmente, y la decision se pone aqui entera para que el integrador
pueda rebatirla:

1. **Se rompia solo, y no hay forma de que no se rompa.** Su modelo mide
   `plazum serve` sobre una instalacion nueva pidiendo las rutas con
   `http.Get` a pelo. Con la entrada cableada, `/alcance` deja de contestar 200
   sin sesion y el test cae en `entreMain` con un Fatal. Las dos unicas salidas
   eran tocarlo o **commitear con un test en rojo**, y esa segunda esta prohibida
   sin matices en CLAUDE.md.
2. **Su propio texto pedia este cambio**: *«Si han bajado, alguien ha cableado la
   entrada y tiene que bajar el numero aqui»*. Todo el fichero esta escrito
   alrededor del hueco que este commit cierra, incluida su seccion «EL RESULTADO,
   DICHO ANTES DE MEDIR NADA», que describia el P0 de este frente.
3. **Por que no va en un commit aparte**, que seria lo comodo para el
   integrador: un commit con solo el resto dejaria esa puerta en rojo, y un
   commit intermedio en rojo esta prohibido igual. Va en el mismo commit, y se
   avisa aqui.

**El choque probable con el frente C, y como se resuelve.** El frente C trabaja
en `superficies/pantallas/` y en la entrevista, o sea en las **41 preguntas de
`/alcance` que son el 59% de este TTFV**. Si bajan las preguntas, el total baja y
puede cruzar el presupuesto de 15 minutos, y entonces salta la rama que obliga a
**borrar `TechoDeclaradoTTFV`** en ese mismo commit. La resolucion semantica del
conflicto, si lo hay, es **quedarse con las dos cosas**: la instalacion de este
frente (sin ella el fichero no compila contra el producto nuevo) y la cuenta de
preguntas del frente C.

Que se cambio: el recorrido ahora **instala** (lee el token, crea el
administrador) y pide los seis pasos con la sesion abierta.

- `PasosQueExigenSesion` **sigue valiendo 3**, porque lo que cuenta no ha
  cambiado: `/acta/`, `/uar/` y `/escalado/` siguen sin servirse sin sesion.
  Lo que cambio es que ahora se puede abrir una. Se anadio ademas el control
  positivo dentro del propio recorrido.
- Se anadio `CosteDeInstalar = 120 s`, que antes no estaba en el modelo porque
  no habia nada que contar.
- **`TechoDeclaradoTTFV` sube de 20 a 25 minutos, y el numero empeora de 18m56s
  a 23m11s.** Hay que decirlo sin adornos. La diferencia se explica entera: tres
  lecturas de pantalla que antes no se hacian (2m15s) y la instalacion (2m0s).
  **No es la trampa que ese techo persigue** (esa es subirlo porque la entrevista
  ha crecido): el recorrido medido ya no es el mismo, porque antes se median tres
  sextos y ahora seis. Un techo en 20 minutos obligaria a esconder medio camino
  para pasar.

El cuello de botella no ha cambiado: **41 preguntas en la entrevista de
`/alcance`, 13m40s, casi tres quintas partes del total.**

### 4.3 Una trampa del `.github/frontera.sh`, para quien venga detras

Este frente arranco de una base en la que `.github/frontera.sh` todavia tenia la
matriz de la campana ANTERIOR (`frente_A="paquetes/ai-act/ ..."`). Comprobarse
contra ella habria dado **frontera rota en todo**, o sea un rojo que no significa
nada, que cuesta lo mismo que un verde que no significa nada. La matriz de este
tramo entro en `main` a mitad del trabajo (commit «la matriz de fronteras del
tramo del P0»).

La leccion, que va a repetirse en cada campana: **la comprobacion de frontera se
hace despues del `git rebase` sobre `main`, no antes**, porque la matriz vive en
`main` y puede llegar mas tarde que el frente que la necesita.

---

## 5. Lo que NO se hizo, con su cardinal

1. **Las 2 pantallas de arranque (`/entrar` y `/primer-admin`) siguen en espanol
   fijo, fuera del catalogo.** `superficies/serve` no recibe catalogo y darselo
   significa un campo nuevo en `serve.Config` y claves nuevas en
   `adaptadores/catalogo/cadenas/{es,en}.json`. Es un cambio de puerto de hecho, y
   la puerta «inventario contra lo que pide la interfaz» compara las claves con lo
   que la interfaz pide: claves que nadie pide la rompen. No es una regresion de
   este commit (ya estaba asi), pero ahora **importa mas**, porque `/primer-admin`
   ha pasado de ser inalcanzable a ser la primera pantalla de toda instalacion.
   Un comprador que lea en ingles ve la instalacion entera en espanol.

2. **0 ordenes de terminal para crear el administrador.** Hoy la unica via es el
   navegador. Una instalacion headless (Docker, Ansible, un runner de CI) tiene
   que hacer el baile de curl del punto 4.1. Una orden
   `plazum admin crear --usuarios F --usuario X` leyendo el secreto de la entrada
   estandar cerraria eso y haria trivial el arreglo del workflow. Se dejo fuera a
   proposito: es una segunda via de creacion de credenciales y merece su propia
   pasada adversaria, no un anadido al final de este tramo.

3. **1 sola cuenta por instalacion, en la practica.** El almacen soporta varias
   en su formato, y solo hay una forma de crear la primera. No hay alta de mas
   usuarios, ni cambio de contrasena, ni bloqueo por intentos fallidos por cuenta
   (el rate limit de `serve` es por cliente, no por usuario). Para una v1 con
   `adaptadores/oidc` como camino recomendado esto es coherente, y hay que
   decirlo: **plazum con cuentas locales es de una persona**.

4. **Los permisos 0600 del fichero de cuentas solo significan algo fuera de
   Windows.** `os.Chmod` en Windows solo mueve el bit de solo lectura, asi que el
   test los comprueba solo donde valen. En Windows el fichero queda con la ACL
   que herede del directorio.

5. **0 cambios en `adaptadores/catalogo/cadenas/`.** Estaba en la columna de este
   frente por si traia pantallas nuevas; no las trae. La unica pantalla nueva de
   verdad es `/primer-admin`, que ya existia y que vive fuera del catalogo (punto
   1 de esta lista).

---

## 6. Recorrido del comprador, cronometrado

De un directorio vacio a las seis pantallas, con el binario ya compilado:

```text
arranque hasta que /salud contesta        678 ms
lectura del token + GET /primer-admin      ~
POST /primer-admin (crea la cuenta)       incluido
los seis pasos con la sesion              todos 200
TOTAL de maquina                        2.760 ms
```

Lo que cuesta de verdad es lo humano: **41 preguntas de entrevista**. El TTFV
completo del camino, medido por su puerta, es **23m11s** contra un presupuesto de
15 minutos.
