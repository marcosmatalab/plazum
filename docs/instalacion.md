# Instalar plazum

De cero a tu primer calendario de obligaciones. No hace falta clonar nada, ni
compilar, ni tener Go instalado, ni conexion despues de la descarga.

plazum calcula que obligaciones legales aplican a tu organizacion y cuando
vencen, a partir del texto de las normas. Este documento te lleva hasta la
primera pantalla que sirve para algo.

## Lo que vas a tardar

Medido el 04-09-2026 en un portatil normal, contando solo lo que tarda la
maquina y no lo que tardas tu en leer.

| paso | tiempo |
|---|---|
| descargar los dos ficheros (12,4 MiB + 354 KiB) | segun tu linea |
| `plazum corpus --instalar` | 1,1 s |
| `plazum calendario` | 0,2 s |

De la descarga al primer calendario son dos ordenes y menos de dos segundos de
computo. La parte lenta es la descarga y la marca tu red, no plazum.

## Antes de nada, que te vas a bajar y por que son dos ficheros

De la pagina de releases salen dos cosas y hacen falta las dos.

**El programa.** `plazum-linux-amd64`, `plazum-darwin-arm64`,
`plazum-windows-amd64.exe` y los demas. Es un solo fichero, sin instalador y sin
dependencias.

**El corpus.** `plazum-corpus.tar.gz`. Son las normas, unos 350 KiB. Van aparte
del programa a proposito, y el motivo te importa a ti, no a nosotros, las normas
cambian cuando cambia el BOE y el programa cambia cuando cambia el programa. Si
el corpus viajara dentro del ejecutable, cada vez que un reglamento moviera una
fecha tendrias que bajarte otra vez el programa entero. Asi te bajas 350 KiB.

Tambien encontraras `plazum-corpus.huella`, `SHA256SUMS-*`, y un `.sig` y un
`.pem` por cada fichero. Sirven para comprobar que lo que te has bajado es lo que
publicamos, y se explican mas abajo.

**Si el corpus no esta, no instales el programa y avisanos.** Un plazum sin
corpus arranca y no sabe ninguna norma.

## Camino corto

Con Linux o macOS, en un directorio vacio, con los dos ficheros dentro.

```bash
chmod +x plazum-linux-amd64
mv plazum-linux-amd64 plazum

./plazum corpus --instalar plazum-corpus.tar.gz
./plazum calendario --pais=ES --sector=fabricante-software --empleados=200
```

En Windows, con PowerShell, lo mismo sin el `chmod` y con `.\plazum.exe`.

Eso es todo. La segunda orden te imprime las fechas de los proximos doce meses
con el articulo de cada una.

### Que hace la primera orden, porque no es solo descomprimir

`corpus --instalar` calcula la huella de lo que le das y la compara con la que el
programa lleva dentro, **antes** de escribir nada en tu disco. Si no cuadra, no
instala y te dice las dos huellas.

Esto importa porque un corpus son fechas legales en las que vas a confiar. Un
fichero de normas que entra sin comprobar es peor que no tener ninguno.

Si sale bien veras algo asi.

```text
  Corpus instalado en paquetes: 33 paquetes, 300 ficheros.
  huella e5e3b2dc4fcb9638304becc5b70152daee31935099855878cbbcc3c7337cf3e0
  Comprobada contra el ancla que este binario lleva dentro.
```

### Los sectores que hay

Si te equivocas de sector, plazum te lista los que conoce. Hoy son estos.

```text
--pais=ES --sector=fabricante-software     Fabricante de software establecido en Espana
--pais=ES --sector=sector-publico          Entidad del sector publico espanol
--pais=ES --sector=servicios-digitales     Prestador privado de servicios digitales en Espana
```

## Lo que acabas de ver, y lo que NO es

Ese calendario sale de un **perfil de arranque**, no de tus respuestas. Cada fila
sale marcada con `[supuesto]` y la pantalla te dice, una por una, que ha supuesto
y con que confianza.

No es tu cumplimiento. Es lo que le pasaria a una organizacion del perfil que has
pedido. Sirve para ver en un minuto si esto te vale, no para presentarlo a nadie.

Para lo tuyo de verdad tienes que contestar la entrevista.

```bash
./plazum serve
```

Levanta la interfaz web en tu maquina. Contestas, y luego.

```bash
./plazum alcance --cuenta tu-cuenta --sujeto miorg --organizacion "Tu SL"
./plazum calendario --alcance alcance.json
```

Ahora las filas ya no llevan `[supuesto]`, porque salen de lo que has contestado.

## Comprobar que lo que te has bajado es lo nuestro

Recomendado siempre, obligatorio si vas a meter esto en un proceso serio.

**La suma.** Cada plataforma trae su `SHA256SUMS-<sistema>`.

```bash
sha256sum -c SHA256SUMS-linux --ignore-missing
```

**La firma.** Firmamos con cosign sin claves, contra el log publico de Rekor.
Necesitas [cosign](https://github.com/sigstore/cosign) instalado.

```bash
cosign verify-blob \
  --certificate plazum-linux-amd64.pem \
  --signature plazum-linux-amd64.sig \
  --certificate-identity-regexp 'https://github.com/marcosmatalab/plazum/.*' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  plazum-linux-amd64
```

Si esa orden te falla por la identidad y no por la firma, mira que identidad
lleva el certificado de verdad y usa esa.

```bash
openssl x509 -in plazum-linux-amd64.pem -noout -text | grep -A1 "Subject Alternative Name"
```

Y dinoslo, porque entonces el que esta mal es este documento.

**El corpus, cuando ya lo tienes instalado.**

```bash
./plazum corpus
```

Te dice que huella espera el programa, cual tiene el corpus que hay en disco, y
si cuadran.

## Actualizar el corpus sin cambiar de programa

Cuando cambia una norma publicamos un corpus nuevo. Puede ser mas nuevo que tu
programa, y entonces su huella no va a cuadrar con la que tu ejecutable lleva
dentro. Eso es normal y no es un error.

En ese caso, coge la huella de la pagina de la release, viene en
`plazum-corpus.huella`, y pasasela.

```bash
./plazum corpus --instalar plazum-corpus.tar.gz \
  --huella-esperada <la huella de la pagina de la release> --forzar
```

Sigue siendo una comprobacion de verdad. Lo unico que cambia es de donde sale lo
que se compara, de la pagina en vez de del programa. **No hay forma de instalar
un corpus sin comprobarlo**, y es deliberado, un si o no se teclea por costumbre
y acaba en todos los guiones de todo el mundo.

## Con Docker, si lo prefieres

La imagen trae el corpus y un expediente de ejemplo dentro, asi que no hay que
instalar nada.

```bash
docker run --rm ghcr.io/marcosmatalab/plazum calendario \
  --pais=ES --sector=fabricante-software --empleados=200
```

Si quieres usar tu propio corpus, se monta encima.

```bash
docker run --rm -v /mi/corpus:/datos/paquetes ghcr.io/marcosmatalab/plazum corpus
```

Esa ultima orden te dira que el corpus montado no es el que se publico con la
imagen, que es justo lo que quieres saber cuando montas uno tuyo.

## Solo quiero ver que hace, sin instalar normas

```bash
./plazum demo
```

Monta una empresa de ejemplo con sus relojes corriendo, sin red y sin corpus. Es
un paseo de dos minutos con un solo paquete de normas dentro, no es tu
cumplimiento y la propia salida te lo dice. Para quitarlo.

```bash
./plazum demo --deshacer
```

## Cuando algo no funciona

```bash
./plazum doctor
```

Revisa esta maquina y te da el arreglo de cada cosa que encuentre, el reloj, los
permisos de escritura, el corpus y los avisos que se quedarian sin destinatario.

Si vas a abrir una incidencia.

```bash
./plazum doctor --issue
```

Saca un bloque preparado para pegar en un issue publico. No lleva dentro ninguna
direccion completa de las que hayas configurado, solo el nombre del servidor, asi
que se puede pegar sin revisarlo.

## Lo que plazum no hace

Conviene saberlo antes de dedicarle una tarde.

**No es asesoramiento juridico.** Calcula fechas a partir del texto de las
normas y te ensena de que articulo sale cada una para que lo puedas contrastar.

**No te acusa de incumplir.** Si un vencimiento paso y no consta que se hiciera,
lo dice asi, que no consta. plazum no puede distinguir entre algo que no se hizo
y algo que se hizo y no se registro, y no va a fingir que si.

**No trae todas las normas del mundo.** Trae 33 paquetes, y de ellos unos pocos
estan transcritos con sus relojes y sus casos de prueba y el resto son
esqueletos, con los identificadores y la estructura pero sin las obligaciones
todavia. Para ver el estado real de cada uno.

```bash
./plazum cobertura paquetes
```

Te da la cobertura honesta de cada paquete instalado, sin un porcentaje unico
que la maquille.
