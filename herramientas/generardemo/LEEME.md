# generardemo

Construye `expediente-demo.json` y `contexto-demo.json`, los dos ficheros de la
raiz que verifica `plazum verify` recien instalado.

```
go run ./herramientas/generardemo             # compara y no escribe nada
go run ./herramientas/generardemo -escribir   # dice que cambia y lo escribe
```

Sin `-escribir` es una puerta, sale con codigo 1 y con el diff si lo publicado
no es lo que sale del escenario. Nunca reescribe en silencio. La misma
comparacion corre en CI, en `main_test.go`, asi que editar el demo a mano pone
la puerta roja.

## Que hay aqui y por que

`escenario.json` es el escenario, y ahi si se nombran normas reales, el ENS y el
RGPD y el CRA con sus articulos. Ese es el valor del demo, un demo que enseña
identificadores inventados no le demuestra nada a quien lo abre.

`main.go` no puede nombrarlas. `TestNingunaNormaCableada` lee el AST de todo el
codigo del repositorio y una norma en un literal de cadena rompe el build. Un
fichero de datos no lo mira, y de ahi la separacion.

La frontera entre los dos:

- **datos**, lo que el emisor afirma, su corpus (paquetes, reglas de
  aplicabilidad en el dialecto, obligaciones), sus hechos, sus relojes con su
  calendario, sus pruebas y observaciones, y lo que declara haber calculado.
- **codigo**, como se sella y se hace reproducible el artefacto, la clave del
  operador del demo, la derivacion de clave y nonce de cada entrada, el
  checkpoint Merkle y el sello RFC 3161.

Las reglas de `escenario.json` van en el mismo formato que las de un paquete
publicado bajo `paquetes/` y las carga el mismo codigo, o sea que el demo no
puede enseñar una forma de declarar reglas que el producto no acepte.

## El sello no se regenera aqui

El token RFC 3161 sale a la red y lo repone `herramientas/sellardemo`, a mano y
nunca en CI. Esta herramienta lo lee de
`nucleo/expediente/testdata/sello-demo.bin` y se niega a publicar nada si el
expediente que sale no verifica con las raices embebidas.

La raiz Merkle cubre las ENTRADAS de la cadena, o sea las observaciones. Si
cambias una observacion, el sello viejo deja de cuadrar y la generacion falla
diciendolo. El orden entonces es:

```
go run ./herramientas/sellardemo
go run ./herramientas/generardemo -escribir
```

Cambiar la organizacion, una regla, un reloj o un paquete no obliga a resellar.

## Que declara el escenario y que se recalcula

La seccion `declarado` (aplicables, reclamaciones, estados, denominadores) es lo
que el emisor dice haber calculado, y va como dato a proposito. El demo existe
para enseñar que un tercero RECALCULA eso y lo contrasta; si se generara desde el
mismo motor que luego verifica, no probaria nada. Una declaracion equivocada en
el escenario no se publica, la verificacion se hace antes de escribir.
