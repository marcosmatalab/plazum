# El presupuesto del binario

plazum se distribuye como **un solo binario sin dependencias**, asi que su tamano
es una promesa al usuario y no un detalle: quien lo mete en una imagen o lo baja
a mano quiere saber cuanto ocupa y por que ha cambiado.

El numero de aqui abajo **no esta escrito a mano**. Lo ata al arbol
`TestElTamanoPublicadoDelBinarioEsElDeHoy`, que construye el binario de Linux con
estas mismas banderas y compara. Si se separan, CI se pone rojo.

<!-- binario:inicio -->
**Y lo que ocupa, que casi nadie publica.** El binario de Linux, con `-s -w -trimpath`, mide **12,1 MB** contra un presupuesto declarado de **25 MB**. Se construye así:

```bash
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -trimpath -o plazum ./cmd/plazum
```

**La cifra es la de `linux/amd64` nativo, que es la máquina que construye la release**, y eso hay que decirlo porque el tamaño no depende sólo del código: cruzando esa misma orden desde Windows salen **0,5 MB más**. Un número de tamaño sin la máquina al lado es medio número, y aquí se aprendió en rojo.

**Ha subido tres veces y las tres se dicen por qué**, porque un binario que engorda sin explicación es lo que hace que nadie se crea el resto. La primera, **0,4 MB el 03-09-2026**: la release **lleva el corpus dentro**, y eso es lo que convierte una máquina recién instalada de 3 relojes en 263, sin red y sin pasos extra. La segunda, **168 KiB el 10-09-2026**, al entrar la lectura de los documentos que subes: extracción de texto, índice de búsqueda y verificación de citas por hash. La tercera, **76 KiB el 11-09-2026**, al entrar la lectura de la cabecera de esos documentos, que es lo que propone su fecha, su alcance, quién los firma y hasta cuándo valen. El presupuesto no se ha movido para acomodar ninguna de las tres.

Las tres cifras salen del **mismo banco**, que es lo que permite restarlas: `go1.24.13` sobre `linux/amd64` nativo, con las banderas de arriba. Dos medidas de máquinas distintas no se restan nunca, y aquí también se aprendió en rojo.
<!-- binario:fin -->
