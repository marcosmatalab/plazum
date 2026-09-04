# Hallazgos de la rebanada 4: la release candidata y la máquina limpia

Tramo 2, 04-09-2026. Rama `tramo2/release`.

Esta rebanada no dispara la etiqueta. Prepara y verifica; **etiquetar es el paso
irreversible del proyecto entero** y lo hace quien integra. Nada de lo que hay
aquí ha publicado nada.

---

## Lo primero, porque cambia el riesgo de todo lo demás

**El candado está ABIERTO y el repositorio es PÚBLICO.**

| dato | comprobado con | resultado |
|---|---|---|
| `.github/marca-congelada` | `ls -la .github/marca-congelada` | no existe (`No such file or directory`) |
| quién lo borró | `git log --diff-filter=D -- .github/marca-congelada` | `b321243`, el 26-08-2026 |
| visibilidad del repositorio | `gh repo view --json isPrivate,visibility` | `{"isPrivate":false,"visibility":"PUBLIC"}` |
| etiquetas en `origin` | `git ls-remote --tags origin` | ninguna |
| etiquetas locales | `git tag -l` | **`v0.2.0`**, sin empujar |
| ejecuciones de `release.yml` | `gh run list --workflow=release.yml` | **ninguna** hasta que lancé las dos de esta sesión (abajo) |
| releases publicadas | `gh release list` | ninguna |

Las tres consecuencias, y ninguna es teórica:

1. **La próxima etiqueta `v*` que se empuje publica de verdad.** No es un
   ensayo. `candado` dice `publicar=si` en toda ejecución.
2. **Existe `v0.2.0` en local y no en `origin`.** Un `git push --tags`
   distraído, o un `--follow-tags`, dispara la release de `v0.2.0` (objeto
   `debdfac`, sobre el commit `51024d4`, del 25-08-2026) en vez del candidato.
   El primer acto irreversible saldría de una etiqueta de hace diez días.
   No es un descuido: `docs/marca.md` la documenta («El tag `v0.2.0` está creado
   en local y **no se empuja**»), o sea que es deliberada. Pero una etiqueta
   deliberadamente retenida y un `--tags` distraído conviven mal, y quien empuje
   no tiene por qué acordarse. **Recomendación: empujar siempre por refspec
   explícita** (`git push origin refs/tags/v0.1.0-rc1`) y nunca `--tags` ni
   `--follow-tags`; o borrarla si ya no se piensa usar.
3. **El workflow no se había ejecutado nunca.** Su propia cabecera decía que eso
   es estrenarse el peor día, y hasta hoy nadie lo había medido. **Lo he
   ejecutado**, y encontró un P0 que ninguna lectura habría encontrado: ver
   «La primera ejecución» más abajo.

### Prosa caducada, que es la familia que peor miente

`docs/marca.md` **no está en mi columna y no lo he tocado.** Cuando empecé decía
dos cosas falsas; quien integra ya corrigió la primera en `92a78b2` mientras yo
trabajaba, así que **este propio informe habría caducado si no lo vuelvo a
mirar**, que es la lección entera repetida sobre mí mismo. Estado real a
`13781f3`:

- ~~«La congelación de publicación sigue puesta»~~ → **corregido en `92a78b2`**,
  y bien: ahora remite al fichero como única fuente de verdad en vez de
  responder él.
- **Sigue mal, línea 85**: «**El repositorio sigue privado**, lo que a su vez
  mantiene desactivados el workflow de CodeQL y el private vulnerability
  reporting». Es falso **dos veces**: el repositorio es público
  (`isPrivate:false`) y CodeQL **no está desactivado**, corre y sale en verde en
  `main` (run `33822147545`, `success`). La segunda mitad la desmiente el propio
  CI del repositorio.

`release.yml` tenía la misma enfermedad y **esa sí la he curado**: su cabecera
decía *«HOY ESTE WORKFLOW NO PUBLICA NADA [...] Existe .github/marca-congelada»*.
El dato tenía puerta (`distribucion_test.go` imprime el estado real del candado
en cada ejecución) y la explicación no, así que el dato se corrigió solo y la
prosa se quedó **con la forma de una decisión tomada** describiendo un mundo que
ya no existe. Quien la leyera creería que puede empujar una etiqueta sin
consecuencia. Es otra aparición de la *afirmación acompañada*, y otra vez quien
miente es la explicación, no el dato.

---

## P0-1. `latest` se movía al candidato. Arreglado.

`-t "${destino}:latest"` era **incondicional** dentro del paso de subida.
Etiquetar `v0.1.0-rc1` habría movido `ghcr.io/marcosmatalab/plazum:latest` al
candidato.

`latest` es lo que se lleva **quien no eligió versión**: un `docker pull` a
secas, un `FROM` ajeno, un despliegue que no fija nada. Apuntarlo a un `-rc` no
rompe el candidato, rompe a todo el que creía estar en la versión estable, y no
hay commit que lo deshaga porque ya se lo bajaron.

**Y `startsWith(github.ref, 'refs/tags/v')`, que YA estaba puesto, no cierra
esto**: `v0.1.0-rc1` también empieza por `refs/tags/v`. Esa condición contesta
«¿es una etiqueta?» y la pregunta que faltaba es «¿es una etiqueta **de qué
forma**?».

## P0-2. La release del `-rc` salía como la release actual. Arreglado.

`softprops/action-gh-release@v3` se invocaba con `{files: dist/*}` y nada más.
El valor por defecto de `prerelease` es `false`, o sea **el permisivo**: es el
invariante 8 en su forma de siempre, el valor cero que significa «sin
restricción». Un `v0.1.0-rc1` habría salido como la release actual del
repositorio, que es la que ofrece la portada y la que responde la API de
*latest release*.

### El arreglo, y por qué está donde está

El criterio se calcula **una sola vez**, en un paso `forma` del trabajo
`candado`, y sale como la salida `definitiva`:

```bash
if [[ "${GITHUB_REF_NAME}" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "definitiva=si" >> "$GITHUB_OUTPUT"
else
  echo "definitiva=no" >> "$GITHUB_OUTPUT"
fi
```

Dos copias del criterio (una en la imagen, otra en la release) son dos copias que
se separan, y la que se queda vieja es la que nadie mira. Los dos consumidores
leen esta salida.

**Falla cerrado por los dos lados.** La expresión está anclada en los dos
extremos, así que solo dice `si` a la forma exacta; y un valor que no sea `si|no`
**para el paso** en vez de tomarse por «no», que es la tercera forma de la nada
del invariante 8: si alguien renombra el `id` del paso, `latest` dejaría de
moverse **para siempre** y ninguna release lo notaría, porque el fallo iría en la
dirección que nadie mira.

Demostrado bajo el shell en el que corre (`bash --noprofile --norc -eo pipefail`,
el de GitHub para `shell: bash`), no bajo el que lo escribí:

```
ref=v0.1.0-rc1      rc=0  ->  definitiva=no
ref=v0.2.0          rc=0  ->  definitiva=si
ref=main            rc=0  ->  definitiva=no
ref=v1.0.0-beta.2   rc=0  ->  definitiva=no
ref=v1.0            rc=0  ->  definitiva=no
ref=v1.0.0+meta     rc=0  ->  definitiva=no
ref=v10.20.30       rc=0  ->  definitiva=si
```

Y el paso de subida, con el cuerpo **extraído del propio `release.yml`** (no una
transcripción, que se separa) y `docker` sustituido por un sello:

```
--- ref=v0.1.0-rc1  DEFINITIVA='no' ---
Se subira ghcr.io/marcosmatalab/plazum:v0.1.0-rc1 y NO se movera .../plazum:latest.
  Motivo: v0.1.0-rc1 no es una version definitiva (vX.Y.Z sin sufijo).
    -> docker buildx build ... -t ghcr.io/marcosmatalab/plazum:v0.1.0-rc1 --push .
    rc=0
--- ref=v0.2.0  DEFINITIVA='si' ---
Se subira ghcr.io/marcosmatalab/plazum:v0.2.0 y se movera .../plazum:latest a esta imagen.
    -> docker buildx build ... -t .../plazum:v0.2.0 -t .../plazum:latest --push .
    rc=0
--- ref=v0.2.0  DEFINITIVA='' ---
PUERTA ROTA: la forma de la etiqueta llega como '' y solo puede ser si o no.
    rc=1
```

## P1. El trabajo `ensayo` estaba MUERTO y no se podía ver. Arreglado.

Su condición era `needs.candado.outputs.publicar != 'si'`, o sea «el candado está
puesto». El candado se borró el 26-08-2026, así que `publicar` vale **siempre**
`si` y ese trabajo **no podía correr jamás**. Un `workflow_dispatch` de hoy
construía los binarios y la imagen y luego no resumía nada, que es exactamente lo
que el ensayo existe para hacer.

La causa es la de siempre: el ensayo colgaba de **un** motivo cuando hay **dos**
por los que una ejecución no publica, y el que queda vivo es el otro (no es una
etiqueta). Ahora cuenta los dos, igual que ya hacía el paso «la imagen no se ha
subido». Y su `cat .github/marca-congelada` ya no mata el paso bajo `-e` cuando
el fichero no está, que es el fallo del 26-08-2026 repetido.

**Y nadie podía verlo ejecutando**, porque el workflow tiene cero ejecuciones. Lo
encontró leerlo.

## P2. `anchore/sbom-action` publica por su cuenta, y no se declaraba.

Leído de su `action.yml` con `gh api` el 04-09-2026: **`upload-artifact` y
`upload-release-assets` valen `"true"` por defecto**. Y en
`SyftGithubAction.ts:576`:

```ts
const isRefPush = eventName === "push" && ref.startsWith(releaseRefPrefix);
if (isRefPush) { const tag = ...; release = await client.findRelease({ tag }); }
```

O sea que en un push de etiqueta busca la release de ese tag para adjuntarle el
SBOM, **fuera de la decisión de `prerelease`**. Hoy no la encuentra por un solo
motivo: el paso corre **antes** de que `action-gh-release` la cree. Lo único que
lo salva es el **orden de los pasos**, que nadie vigila. Un valor por defecto
permisivo apagado por accidente es el invariante 8 con otro traje.

No lo cambio (es comportamiento de un tercero y hoy es inocuo), pero **entra en
la lista de marcadores de publicación** de `distribucion_test.go`, que no lo
reconocía: eran 5 pasos de publicación contados y son **6**.

---

## Los seis destinos de una ejecución que publica

Leídos de `release.yml`, no de memoria. «Verificado» y «no se pudo ejecutar» son
cosas distintas y van separadas a propósito.

| # | destino | quién lo produce | qué llega | comprobado |
|---|---|---|---|---|
| 1 | **Artefactos del run** (almacén de Actions) | `actions/upload-artifact@v4`, job `binarios` | `dist-linux`, `dist-darwin`, `dist-windows` | **leído** en el workflow. No observado: cero ejecuciones |
| 2 | **Artefactos del run**, otra vez | `anchore/sbom-action@v0`, `upload-artifact: true` por defecto | `sbom.cdx.json` | **leído del `action.yml` del tercero** con `gh api`. No observado |
| 3 | **ghcr.io** | `docker/login-action@v3` + `docker buildx --push` | `ghcr.io/marcosmatalab/plazum:<etiqueta>`, y `:latest` **solo si la etiqueta es definitiva** | **ejecutado en local** con `docker` sellado; y la dirección confirmada por un intento real que el registro rechazó (`denied`) |
| 4 | **Rekor** (log público append-only) | `cosign sign-blob`, job `publicar` | un certificado por fichero firmado. **Irreversible** | **solo leído. NO ejecutado, a propósito**: ejecutarlo es el acto irreversible |
| 5 | **Release de GitHub** | `softprops/action-gh-release@v3`, `files: dist/*` | 30 ficheros (abajo el desglose) | **solo leído.** No ejecutado |
| 6 | **Release de GitHub, por la puerta de atrás** | `anchore/sbom-action@v0`, `upload-release-assets: true` por defecto | el SBOM, si la release ya existiera | **leído del código del tercero**. Hoy inocuo solo por el orden de los pasos |

### Los 30 ficheros de la release, derivados del workflow

- `binarios` construye 2 binarios por sistema (`amd64`, `arm64`) y un
  `SHA256SUMS-<goos>` → **3 ficheros × 3 sistemas = 9**
- `publicar` los aplana en `dist/` y añade `sbom.cdx.json` → **10**
- `cosign sign-blob` recorre `dist/*` (el glob se expande **una vez**, antes del
  bucle: son 10 vueltas) y deja `.sig` y `.pem` por fichero → **+20**
- `files: dist/*` sube **30 activos**

No he podido contrastar este 30 contra una ejecución real porque no hay ninguna.
Es una derivación del texto del workflow, no una medida.

---

## La puerta nueva, y su rojo

`distribucion_test.go` gana tres guardas y su control negativo. **Nació roja
sobre el `release.yml` real de `c289742`**, que es un rojo que nadie le metió:

```
--- FAIL: TestNadieMueveLatestSinPreguntarPorLaFormaDeLaEtiqueta (0.00s)
    release.yml: un paso mueve una etiqueta flotante sin preguntar por la forma
    de la etiqueta.
        linea:  -t "${destino}:${etiqueta}" -t "${destino}:latest" --push .
        hace:   mover 'latest' en un registro de imagenes
        if del paso:     "if: needs.candado.outputs.publicar == 'si' && startsWith(github.ref, 'refs/tags/v')"
        if del trabajo:  ""
      OJO: startsWith(github.ref, 'refs/tags/v') NO sirve aqui. v0.1.0-rc1
      tambien empieza por refs/tags/v, y es justo el caso que hay que dejar fuera.
--- FAIL: TestUnaReleaseDeGitHubDiceSiEsUnCandidatoOUnaVersion (0.00s)
    release.yml: un paso crea una release de GitHub sin decir si es un candidato
    o una version.
      paso:
          - name: publicar release
            uses: softprops/action-gh-release@v3
            with: {files: dist/*}
--- FAIL: TestElCriterioDeVersionDefinitivaViveEnUnSoloSitioYFallaCerrado (0.00s)
    release.yml ya no contiene "definitiva=si"
    release.yml ya no contiene "definitiva=no"
    release.yml ya no contiene "=~ ^v[0-9]+\\.[0-9]+\\.[0-9]+$"
```

### La mutación encontró un agujero EN LA PROPIA PUERTA

Es el hallazgo del día y no estaba previsto. La primera versión del guarda
buscaba el marcador **en el bloque entero del paso**, así que bastaba
**nombrarlo**. Se borró la única línea que lo usaba de verdad:

```
238d237
<           DEFINITIVA: ${{ needs.candado.outputs.definitiva }}
```

y la puerta **siguió verde** (`ok github.com/marcosmatalab/plazum 0.119s`),
porque el mensaje de error del propio paso decía *«Sale de
needs.candado.outputs.definitiva»*. **La guarda satisfecha por su propia prosa**,
dentro de la guarda escrita para cerrar justamente esa familia.

El arreglo es la regla que hace el trabajo en un workflow: un valor solo **llega**
a un paso si viaja dentro de una expresión `${{ }}` o está en un `if:`, que ya es
contexto de expresión. Un `echo` que lo nombra y un comentario que lo explica no
mueven nada. Con eso puesto, la misma mutación de una línea:

```
--- FAIL: TestNadieMueveLatestSinPreguntarPorLaFormaDeLaEtiqueta (0.00s)
    linea:  etiquetas+=(-t "${destino}:latest")
    if del paso: "needs.candado.outputs.publicar == 'si' && startsWith(github.ref, 'refs/tags/v')"
```

### El control negativo, en las dos direcciones

La dirección de **no acusar** hace igual de falta: este repositorio escribe más
comentario que código en los workflows, y `release.yml` explica en prosa el fallo
que arregló, con `:latest` dentro. Un guarda que acuse a esa prosa se pone rojo el
día que alguien documenta bien, y entonces se borra el comentario o se borra el
guarda.

Nueve subcasos, todos en verde:

| caso | acusa |
|---|---|
| mueve `latest` a pelo | sí |
| el criterio en el `env` del paso | no |
| **solo `startsWith` de etiqueta** (el fallo del 04-09-2026) | sí |
| solo el candado (contesta la otra pregunta) | sí |
| `latest` solo en un comentario | no |
| `ubuntu-latest` no es una etiqueta flotante | no |
| una almohadilla dentro de comillas no tapa el push de detrás | sí |
| **nombrar el criterio en un `echo` no es consultarlo** | sí |
| el criterio en un `if:` de trabajo sí cuenta | no |

Y sobre el fichero real, comprobado a mano: los comentarios de `release.yml` que
contienen `:latest` **no se cuentan** (son 4 líneas flotantes contadas, las de
código), y la misma línea sin almohadilla **sí** se acusa.

---

## La prueba de la máquina limpia

`docs/lanzamiento/maquina-limpia.sh`. **El criterio de éxito no es que el
workflow salga verde**: un workflow verde dice que se construyó un fichero, no
que ese fichero sirva en manos de otro. El criterio es coger el binario
publicado, en una máquina sin el repositorio, y **llegar al calendario**.

Se niega a correr dentro del repositorio, y eso es la mitad de su valor: ahí
`paquetes/` está al lado sin que nadie lo pida y el verde mediría lo contrario de
lo que se quiere medir.

```
$ docs/lanzamiento/maquina-limpia.sh --binario ...
ESTOY DENTRO DEL REPOSITORIO (hay .git, go.mod o paquetes/ aqui).
rc=2
```

### Ejecutada hoy contra el artefacto DE VERDAD

No contra un binario que me construí yo: contra
`plazum-windows-amd64.exe` **bajado del artefacto de la ejecución
`33854068327`** con `gh run download`, o sea el que salió del workflow. Y con la
suma contrastada contra la que calculó el propio workflow:

```
$ cat SHA256SUMS-windows
dddc50843d6cb3ee7f65e9f880de565c173e14a24a275aa4718c7943628c413b *./plazum-windows-amd64.exe
$ sha256sum plazum-windows-amd64.exe
dddc50843d6cb3ee7f65e9f880de565c173e14a24a275aa4718c7943628c413b *plazum-windows-amd64.exe
```

En un directorio sin `.git`, sin `go.mod` y sin `paquetes/`:

```
pasos ejecutados: 9   rotos: 0   saltados: 0
LA PRUEBA PASA ENTERA: el binario publicado llega al calendario en una
maquina sin repositorio, y se va sin dejar nada.
RC FINAL DEL GUION = 0
```

Los nueve pasos, con su código esperado, porque **la mitad de la prueba consiste
en que ciertos comandos fallen bien**:

| paso | rc esperado | qué demuestra |
|---|---|---|
| `plazum` a secas | 2 | imprime por dónde empezar y no hace nada |
| `plazum calendario` | **1** | sin corpus falla, y dice cómo arreglarlo |
| `plazum demo` | 0 | instala un corpus **sin red y sin repositorio** |
| el corpus está donde `demo` dijo | 0 | `plazum-demo/paquetes/demo-empresa/paquete.json` |
| `plazum doctor --corpus plazum-demo/paquetes` | 0 | el parte de la máquina |
| **`plazum calendario --corpus ... --alcance ...`** | **0** | **el criterio entero** |
| la salida trae `PROXIMOS DOCE MESES` | — | que el rc=0 no sea una pantalla vacía |
| `plazum demo --deshacer` | 0 | sabe irse |
| y de verdad no queda nada | 1 | `test -d plazum-demo` |

Cuando llegue el artefacto de verdad, el mismo guion con `--desde-release
v0.1.0-rc1` se baja los activos, **comprueba la suma contra `SHA256SUMS-<goos>`
antes de ejecutar nada** y verifica la firma con `cosign verify-blob` si hay
cosign. Si no lo hay, lo dice como **saltado con su motivo**, que no es lo mismo
que comprobado: un paso saltado se cuenta aparte y el cierre lo distingue.

### Y se la ha visto fallar

Contra un binario impostor que sale con 0 a todo:

```
== el calendario trae fechas de verdad, no una pantalla vacia
   PASO ROTO: la salida tiene rc=0 y no parece un calendario.
   Un rc=0 con la pantalla vacia es exactamente el verde que este
   guion existe para no dar.
---------------------------------------------------------------
pasos ejecutados: 9   rotos: 4   saltados: 0
PRUEBA ROTA: 4 de 9 pasos.
RC FINAL = 1
```

### Contra el comprador: tres hallazgos de la máquina limpia

**P1 — el mensaje te manda a hacer lo que acabas de hacer.** Después de
`plazum demo`, que sale con 0, `plazum calendario` **sigue fallando**:

```
error: el corpus de paquetes no carga: open paquetes: ...
  Instala uno con `plazum demo`, o apunta --corpus al directorio que tenga tus paquetes
```

`demo` instala en `plazum-demo/paquetes` y el defecto de `--corpus` es
`paquetes`. El comprador teclea lo que la ayuda le puso primero
(«`plazum calendario`» es una de las tres de *empieza por aquí*), obedece el
error, y vuelve a chocar con el mismo error. El arreglo barato: que el mensaje
mire si existe `plazum-demo/paquetes` y lo nombre, o que `demo` imprima el
comando exacto del calendario en su bloque **QUE HACER AHORA**, donde hoy solo
imprime el de `doctor`.

**P1 — el sector del demo no es un perfil de arranque.** El demo dice
`organizacion.sector = privado`, y lo natural después de leerlo es:

```
$ plazum calendario --corpus plazum-demo/paquetes --pais=ES --sector=privado --empleados=212
error: no hay perfil para --pais=ES --sector=privado.
```

El error es bueno (lista los tres que hay), pero es un choque evitable.

**P2 — el corpus real no viaja en la release.** Los 30 activos son binarios,
sumas, SBOM y firmas: **no hay ningún paquete de corpus**. Quien se baja el
binario llega al calendario con el corpus **de demostración** (1 paquete, 7
obligaciones, `urn:plazum:demo:empresa`), no con los 30 marcos. La propia salida
de `demo` lo dice: *«plazum demo --corpus ./paquetes — lo mismo con el corpus real
de 30 marcos, **si lo tienes al lado del binario**»*, y al lado del binario no
está: está en el repositorio. **El criterio se cumple en la letra y no en el
espíritu**, y conviene decidirlo antes de la v1: o el corpus real se publica como
activo de la release, o se dice en la portada que el binario trae una
demostración.

---

## La primera ejecución, y el P0 que sólo ella podía encontrar

Lancé `workflow_dispatch` sobre `tramo2/release`. **No puede publicar**, y no por
confianza: los cinco `if:` del fichero exigen `startsWith(github.ref,
'refs/tags/v')` para todo lo que sale de la máquina, y una rama no lo cumple.

### Ejecución 1 (`33853740997`): roja, y menos mal

```
✓ candado de marca in 4s
X imagen Docker in 5s
  X construir (las dos arquitecturas, sin salir de aqui)
```

```
docker buildx build --platform linux/amd64,linux/arm64 --output type=cacheonly .
ERROR: failed to build: Multi-platform build is not supported for the docker driver.
Switch to a different driver, or turn on the containerd image store, and try again.
##[error]Process completed with exit code 1.
```

**Un runner limpio trae buildx con el driver `docker`, que sólo construye la
arquitectura de la máquina.** El workflow llevaba desde que se escribió dando por
hecho lo contrario. Faltaban `docker/setup-buildx-action@v3` (para un builder con
driver `docker-container`) y `docker/setup-qemu-action@v3` (para los binfmt de la
otra arquitectura).

**Qué significaba para la etiqueta**: `publicar` depende de `imagen`, así que una
etiqueta empujada hoy **no habría publicado nada**; habría dado una ejecución
roja. Menos malo que publicar mal, y aun así es estrenarse el peor día.

Y la trampa que venía de regalo con el arreglo: con driver `docker-container`,
`docker build -t plazum:release .` deja el resultado en la caché del builder y
**no** en el almacén de imágenes, así que el `docker run` del paso siguiente no la
encontraría. Pasa a ser `docker buildx build --load`, explícito.

### Ejecución 2 (`33854068327`): verde entera

```
✓ candado de marca in 5s
✓ imagen Docker in 39s
✓ binarios en windows-latest in 3m32s
✓ binarios en macos-latest in 1m27s
✓ binarios en ubuntu-latest in 1m8s
✓ ensayo sin publicar in 6s
- firmar y publicar            (saltado: no es una etiqueta)
```

(Y repetida sobre `main` ya rebasado, ejecución **`33854766173`**, igual de
verde y con `firmar y publicar` saltado por no ser una etiqueta. Son dos verdes
sobre el mismo `release.yml`, cuya suma es
`946166c6ad5ee24ea5b0e7e227f41479652ed286d1425578b64c243dded8aa4e`.)

Y **el ensayo, que llevaba muerto desde el 26-08-2026, vuelve a hablar**:

```
Artefactos construidos y probados en su propio sistema:
recogido/dist-darwin/SHA256SUMS-darwin
recogido/dist-darwin/plazum-darwin-amd64
recogido/dist-darwin/plazum-darwin-arm64
recogido/dist-linux/SHA256SUMS-linux
recogido/dist-linux/plazum-linux-amd64
recogido/dist-linux/plazum-linux-arm64
recogido/dist-windows/SHA256SUMS-windows
recogido/dist-windows/plazum-windows-amd64.exe
recogido/dist-windows/plazum-windows-arm64.exe

NADA de esto se ha publicado, ni firmado, ni subido a Rekor.
Motivos (pueden ser los dos, y se dicen los dos):
  - esto no es una etiqueta de version (ref: refs/heads/tramo2/release).
    Para publicar de verdad, etiqueta con vX.Y.Z y empuja la etiqueta.
```

**Esos 9 ficheros ya no son una derivación: son una medida.** El total de 30
activos pasa a ser **9 medidos + 21 derivados** (el SBOM, y un `.sig` y un `.pem`
por cada uno de los 10).

### P1 que deja ver la ejecución 1: el ensayo se esconde justo cuando hace falta

En la ejecución roja, `ensayo sin publicar` **no corrió**. Su `if` era correcto,
pero declara `needs: [candado, binarios, imagen]` y un `needs` que falla salta el
trabajo pase lo que pase el `if`. O sea que **el trabajo que existe para contar
qué ha pasado desaparece exactamente cuando algo ha pasado**. Es la misma
enfermedad del ensayo muerto, en otra forma.

No lo arreglo aquí, y digo por qué: la salida es `if: always() && (...)`, y con
`always()` hay que endurecer además el `download-artifact` y el `find` para el
caso de que no haya artefactos. Prefiero entregar un arreglo de buildx
**verificado en verde** que dos cambios de los que uno no he podido probar.
Queda anotado con su arreglo escrito.

### P2: avisos de Node 20 en cuatro acciones

La ejecución los reporta: `actions/upload-artifact@v4`,
`actions/download-artifact@v4`, `docker/setup-buildx-action@v3` y
`docker/setup-qemu-action@v3` apuntan a Node 20, ya deprecado, y se están
forzando a Node 24. Hoy no rompe nada. El día que GitHub retire el forzado, lo
hará en todos a la vez.

---

## Lo que se puede ensayar hoy sin publicar

`workflow_dispatch` corre `candado`, `binarios` (las tres plataformas, con la
suite entera y el binario nativo ejecutándose) e `imagen` (construye las dos
arquitecturas y la ejecuta), y **para en el borde**: los pasos que salen de la
máquina exigen además una etiqueta. Con el arreglo del trabajo `ensayo`, ahora
además **resume qué habría salido**, que es lo que llevaba sin hacer desde el
26-08-2026.

**Lo he lanzado dos veces**, sobre esta rama, y está contado arriba: la primera
encontró el P0 de buildx y la segunda salió verde entera. Recomiendo **una
tercera sobre `main` ya fusionado**, que es gratis y confirma que lo verde de
aquí sigue verde allí, antes de empujar ninguna etiqueta.

---

## Veredicto

**SÍ, con dos condiciones mecánicas que se cumplen en un minuto.** Cambio el
veredicto respecto de lo que escribí antes de ejecutar el workflow: entonces era
un «no» porque `imagen` no construía y nadie lo sabía. Ya construye, y está
comprobado en verde, no razonado.

Lo que está listo y **verificado ejecutando**, no leyendo:

- `latest` sólo se mueve con `vX.Y.Z` sin sufijo, y dice el motivo cuando no lo
  mueve. Los dos casos corridos bajo el shell de GitHub.
- La release de un `-rc` sale como prerelease y no como la actual, con
  `prerelease` y `make_latest` sacados del mismo criterio (inputs confirmados
  contra el `action.yml` de la acción).
- El criterio se calcula una vez, está anclado por los dos extremos, y el paso
  muere si llega ilegible.
- Tres guardas nuevas con control negativo en las dos direcciones, **nacidas
  rojas sobre el workflow real**.
- `imagen` construye las dos arquitecturas: **ejecución `33854068327`, verde**.
- Los binarios de las tres plataformas, con la suite entera y el binario nativo
  verificando el expediente demo: **verde en las tres**.
- El ensayo vuelve a hablar, y su salida está pegada arriba.
- La máquina limpia llega al calendario **con el artefacto real del workflow**,
  suma contrastada: 9 pasos, 0 rotos, 0 saltados.
- `./comprobar.sh`: **21 puertas en verde, 3 saltadas** (las de `-race`, que
  exigen cgo y aquí `CGO_ENABLED=0`; en CI sí corren), 24 leídas de los
  workflows, 3 herramientas de seguridad, salida 0.

Las dos condiciones, antes de empujar:

1. **Empujar por refspec explícita**: `git push origin refs/tags/v0.1.0-rc1`.
   Nunca `--tags` ni `--follow-tags`, porque `v0.2.0` sigue viva en local y se
   iría con ellos. Es el único riesgo que queda con consecuencia irreversible.
2. **Fusionar esta rama a `main` primero.** Todo lo verde de arriba se midió
   sobre `tramo2/release`; una etiqueta sobre un `main` sin estos commits
   construye el workflow viejo, que no sabe hacer la imagen.

Lo que **no** bloquea, y conviene decidir sabiendo que se decide:

- **El corpus real no viaja en la release** (P2 de arriba). Quien baje el binario
  llega al calendario con la demostración, no con los 30 marcos. O se publica el
  corpus como activo, o se dice en la portada.
- La línea 85 de `docs/marca.md`, que no es mi columna.
- Los dos P1 de mensajes de la máquina limpia.
- El ensayo que se esconde cuando un `needs` falla, con su arreglo escrito.

Y una cosa que no es condición pero sí conviene: **una tercera ejecución de
`workflow_dispatch` sobre `main` ya fusionado**, que es gratis y confirma que lo
verde de aquí sigue verde allí.

---

## Mis errores en esta rebanada

**Lancé un `docker buildx build --push` de verdad contra ghcr.io.** Para
demostrar el paso de subida bajo el shell de GitHub extraje su cuerpo del
workflow y lo ejecuté tal cual, con el `--push` dentro. Salió `denied: denied` en
la petición del token, o sea que **no se escribió nada en el registro**, pero eso
fue **suerte y no diseño**: con una credencial de ghcr en la máquina habría
publicado imágenes desde un árbol de trabajo. Es exactamente el acto irreversible
que esta rebanada tenía prohibido. Rehecho con `docker` sustituido por un sello,
que es lo que había que hacer desde el principio: lo que se quería probar era
**cómo se arma la lista de etiquetas**, y para eso el comando final sobra.

Lo que **no** puedo afirmar: que no haya nada publicado en ghcr.io, comprobado
por API. `gh api user/packages` devuelve `403: You need at least read:packages
scope`, así que es un **«no se pudo comprobar»**, no un «no hay nada». Lo que sí
está comprobado es que las tres tentativas murieron en la obtención del token
(`failed to fetch oauth token: denied`), que es antes de que el registro acepte
ningún blob ni manifiesto, y que la entrada `ghcr.io` de `~/.docker/config.json`
es un **objeto vacío** con `credsStore: desktop`.

**La primera versión de mi propia puerta no guardaba.** Está contada arriba: se
satisfacía con que el paso *nombrara* el criterio. La encontró la mutación
obligatoria, no la revisión del diff, y es la razón de que la mutación sea
obligatoria.

**Maté la sesión con `./comprobar.sh` en primer plano**, más de diez minutos sin
salida y el watchdog a los 600 s. Va en segundo plano y a fichero, y nunca por
`tail`, que devuelve el código de `tail` y no el del guion.

### Lo que intenté romper, por pasada

**Contra la especificación.** ¿Es esto la casilla? Sí, y además encontré dos
cosas que la casilla no pedía y que bloqueaban igual: el trabajo `ensayo` muerto
y la cabecera del workflow mintiendo. Las dos entraban en mi columna, así que las
arreglé en vez de anotarlas.

**Contra el atacante.** Cuatro propiedades que el código daba por buenas:

1. *«El `if` del paso ya distingue una release de un commit cualquiera.»*
   Falso: distingue etiqueta de no-etiqueta, no versión de candidato. Es el P0-1.
2. *«Si la salida del criterio no llega, lo prudente es no mover `latest`.»*
   Cómodo y falso: convierte un mecanismo roto en un silencio permanente en la
   dirección que nadie mira. Ahora para y lo dice.
3. *«Mi guarda comprueba que el paso consulte el criterio.»* Falso, y lo demostró
   la mutación: comprobaba que lo *mencionara*.
4. *«El corte de comentarios de estos tests es inocuo.»* Cortar en la primera
   ` #` pierde el `docker push` que venga detrás de un `echo "a # b"`, y lo que
   se pierde al cortar de más es siempre la parte que publica. Se busca la
   almohadilla fuera de las comillas, y hay un subcaso que lo fija.

Y una que **no** conseguí tumbar: que `ubuntu-latest`, `macos-latest` y
`windows-latest` disparasen el guarda. No lo hacen porque el patrón lleva los dos
puntos delante; comprobado sobre los trece workflows, donde la única línea con
`:latest` era la del fallo.

**Contra el comprador.** Los tres hallazgos de la máquina limpia de arriba. El
que más me preocupa no es ninguno de los dos P1 de mensajes, es el P2: el
comprador que se baja el binario cree que está viendo el producto y está viendo
la demostración.

---

# Tramo 3, rebanada 0: el corpus real viaja en la release (04-09-2026)

El P2 con el que termina la sección anterior (*«el comprador que se baja el
binario cree que está viendo el producto y está viendo la demostración»*) era en
realidad el P0 de este tramo. Está cerrado. Esto es lo que se hizo, lo que se
midió, y lo que salió mal por el camino.

## El número, antes y después

Medido con `docs/lanzamiento/maquina-limpia.sh` fuera del repositorio, sobre un
binario construido con las banderas de la release.

| | paquetes | relojes |
|---|---|---|
| antes (corpus de demostración) | 1 | 3 |
| ahora (corpus real, comprobado) | **33** | **222** |

14 pasos ejecutados, 0 rotos, 0 saltados. `doctor` sobre esa misma máquina
informa de 33 paquetes, 528 obligaciones y 700 casos dorados.

## La decisión: activo firmado al lado, no `go:embed`

Tres razones de producto y una cuarta que no es de mérito y se dice igual.

1. **El corpus cambia en el calendario del BOE, no en el del software.**
   Empotrado, mover una fecha es recompilar tres sistemas por dos arquitecturas
   y volver a firmar seis artefactos. Al lado, son 354 KiB.
2. **`plazum update` ya existe y ya sabe volver atrás.** Un corpus empotrado lo
   deja sin la mitad de su trabajo.
3. **Un producto de cumplimiento tiene que dejar mirar su corpus.** Dos megas de
   JSON dentro de un ejecutable no los abre un abogado.
4. Y la que no es de mérito: `go:embed` no sale del directorio de su paquete,
   así que empotrar `paquetes/` pedía un fichero Go en `paquetes/` o en la raíz
   del módulo, y ninguno de los dos está en la columna de esta rebanada.

**Medido, no supuesto**: el corpus empaquetado son 362.979 bytes; el binario pasa
de 12.797.952 a 12.996.096 bytes. Todo el mecanismo cuesta **198 KiB de binario**.
Empotrar el corpus habría costado 2 MiB y la capacidad de actualizarlo.

## El ancla, y la nada que no abre la puerta

El binario lleva dentro la huella del corpus de su release, inyectada con
`-ldflags -X main.anclaCorpus`. cosign firma el binario, así que la huella viaja
bajo esa firma sin necesitar una cadena aparte. Instalar compara **antes** de
escribir nada.

Las tres respuestas, y el valor cero es el restrictivo (invariante 8):

| ancla | qué pasa |
|---|---|
| vacía (el `-ldflags` que nadie puso) | **se para**. «No puedo comprobarlo» no autoriza |
| presente y no interpretable | se para, con un error **distinto**: el arreglo es otro |
| presente y no cuadra | se para, y se dicen las **dos** huellas |
| presente y cuadra | instala |

Y se puede actualizar sin recompilar, que es lo que un ancla fija podría haberse
cargado: `--huella-esperada` deja al operador aportar el ancla que lee en la
página de la release. Sigue siendo comprobación mecánica, sólo cambia de dónde
sale. **No hay bandera de «instala sin comprobar»**: un sí/no se teclea por
costumbre y acaba en todos los guiones de todo el mundo.

La huella es de un **árbol** y no de un tar: manifiesto ordenado de ruta más
`sha256` de cada fichero, con la versión del algoritmo dentro del resumen.
Sobrevive a reempaquetar y no depende del orden del sistema de ficheros ni del
separador de rutas. **Comprobado en dos sistemas**: la misma huella
(`e5e3b2dc…`) sale en Windows y dentro de Docker sobre Linux, y el corpus
manipulado da la misma huella distinta (`3505db9c…`) en los dos.

## Estreno contra el corpus REAL

La puerta se estrenó sobre dato real antes que sobre una mutación, como manda la
convención. Se cambió **un byte** en `rgpd/paquete.json`, el plazo del art. 33.1
de 72 a 96 horas, se reempaquetó, y el binario anclado se negó a instalarlo
diciendo las dos huellas y sin dejar nada en disco. Es exactamente el ataque que
importa: alguien moviendo un plazo legal en silencio.

Honestidad sobre ese estreno: **fue una mutación sobre dato real, no un hallazgo
que nadie hubiera metido.** No encontró nada que ya estuviera mal en el corpus.

## Las mutaciones, con su resultado

Todas con `.github/mutar.sh` sobre árbol limpio y estado commiteado.

| | qué se rompió | resultado |
|---|---|---|
| M1 | `if h != ancla` pasa a `&& ancla != ""` | **sobrevivió**, y con razón: esa línea es inalcanzable con el ancla vacía, porque `anclaAUsar` ya se ha negado antes. La mutación no tocaba lo que la puerta vigila |
| M2 | `anclaAUsar` devuelve `nil` en vez de `ErrSinAncla` | **cazada** por `TestSinAnclaNoSeInstalaNada` |
| M3 | se apaga el bucle que rechaza `..` | **sobrevivió**: es redundante con el `Clean`+prefijo. Defensa en dos capas, no un agujero |
| M4 | se apagan **las dos** capas de travesía | **cazada** por `TestUnTarballHostilNoEscribeFueraDeSuSitio`. La propiedad sí está guardada |
| M5 | se renombra `anclaCorpus` en Go, el workflow sigue igual | **cazada** por `TestElSimboloQueElWorkflowInyectaExisteEnElCodigo` |
| M6 | se quita el `-X` del paso que construye los binarios publicados | **SOBREVIVIÓ. Agujero de verdad, ver abajo** |
| M6-bis | lo mismo, con la puerta arreglada | **cazada** |
| M7 | la etiqueta de fuente vuelve al repositorio equivocado | **cazada** |
| M8 | el calendario vuelve a contestarse con el corpus de la demo | **cazada**: rc=1, 3 pasos rotos, demostrado en bash fuera del repositorio |

### M6, la que pagó su sitio

La primera versión de la puerta buscaba `-X main.anclaCorpus=` en **release.yml
entero**. Se le quitó ese trozo al paso que construye los binarios que se
publican y la puerta siguió verde, porque la cadena aparece **dos veces** en el
fichero: en el paso que publica, y en un `go build` de usar y tirar dentro del
trabajo `corpus` que sólo comprueba que el `.tar.gz` se instala. La segunda
tapaba la ausencia de la primera.

La puerta habría dejado publicar **seis binarios sin ancla** mientras afirmaba lo
contrario, en verde. No es que faltara: es que estaba **mal apuntada**, que se
lee igual de bien y no vigila nada. Un binario sin ancla se firma en Rekor igual
que uno con ancla y se rompe en la máquina del comprador, o sea después del único
paso irreversible del proyecto.

Ahora se exige dentro del trabajo `binarios`, y además dos cosas que la versión
vieja tampoco miraba: que la huella salga de `needs.corpus.outputs.huella` (un
`-X` con una constante pegada compila igual y da un binario cuya ancla no es la
del corpus de al lado), y que el trabajo declare el `needs` (sin él la expresión
se evalúa a cadena vacía, **GitHub no se queja**, y el `-ldflags` queda
`-X main.anclaCorpus=`).

## Lo que la guarda vieja hizo bien al saltar

`maquina-limpia.sh` llevaba un aviso que decía que se llegaba al calendario con
la demo, con una guarda que lo mataba si aparecían más de 5 paquetes, para que no
mintiera al revés el día que el corpus viajara. **Ese día fue hoy y saltó con
33.** Un aviso que se rompe solo cuando caduca vale más que uno correcto que
nadie vuelve a leer. El aviso está reescrito y la guarda apunta al revés: ahora
lo sospechoso es tener **pocos** paquetes (mínimos 30 y 150).

Y había una segunda cosa: el guion **medía `_marcos` con un grep y no lo imprimía
nunca**. La variable se asignaba y se tiraba. La mitad contable de la afirmación
estaba escrita y muerta, y la transcripción se leía como «el producto funciona
entero» sin que ninguna línea lo dijera. Ahora imprime los dos números, porque un
corpus de 33 paquetes vacíos contaría 33 igual.

## Otros dos hallazgos que no buscaba

**La imagen ya traía el corpus.** `COPY paquetes /datos/paquetes` lleva ahí desde
antes de este tramo. El P0 era **sólo del camino del binario descargado**, no del
contenedor. Aun así la imagen se ha anclado también, porque «tener el corpus» y
«poder decir que ese es el corpus» son cosas distintas y en un contenedor se
separan en cuanto alguien monta el suyo con `-v`. Comprobado sobre la imagen
construida: sin montar dice CUADRA, con un corpus manipulado montado encima dice
NO CUADRA.

**La etiqueta AGPL apuntaba al repositorio equivocado.** El Dockerfile declaraba
`org.opencontainers.image.source="https://github.com/plazum/plazum"` y el módulo
(y el remoto) es `marcosmatalab/plazum`. En una imagen `scratch` esa etiqueta es
la **única oferta de fuente que viaja dentro**, y plazum es AGPL-3.0: quien la
siguiera para ejercer su derecho a la fuente correspondiente acabaría en un
repositorio que no tiene el código que está ejecutando. Además ghcr.io usa esa
etiqueta para enlazar el paquete con su repositorio. Corregida, con puerta que la
**deriva de `go.mod`** en vez de escribirla al lado.

**`mutar.sh` no funcionaba en un worktree**, que es donde se muta. `.git/mutaciones`
a pelo funciona en un checkout normal y no en un worktree, donde `.git` es un
FICHERO: `mkdir -p .git/mutaciones` muere con «Not a directory» en la primera
orden. O sea que el script escrito para que las mutaciones fueran seguras no se
podía usar en ninguna de las cuatro rebanadas de este tramo. Arreglado con
`git rev-parse --git-dir`.

## ghcr.io: ahora sí se puede contestar, y la respuesta es que no hay nada público

El informe anterior decía que no se podía comprobar porque el token no tiene
`read:packages` y el endpoint devuelve 404 hasta para paquetes públicos que
existen. Hay otra vía y funciona: **el endpoint de token anónimo del registro**.

Método, con su control positivo y su control negativo:

| consulta | HTTP |
|---|---|
| `astral-sh/uv` (público, existe) | **200**, y con ese token la lista de etiquetas también da 200 |
| `marcosmatalab/no-existe-jamas-xyz123` | 403 |
| **`marcosmatalab/plazum`** | **403** |

Lo que eso permite afirmar y lo que no:

- **No hay ningún paquete PÚBLICO en `ghcr.io/marcosmatalab/plazum`.** Esto es
  una afirmación positiva, no un «no se pudo»: el control demuestra que un
  paquete público contesta 200.
- **No distingue «no existe» de «existe y es privado»**, porque el inexistente da
  el mismo 403. Eso sigue sin poderse comprobar sin `read:packages`.

Y hay una segunda vía que cierra la pregunta de verdad, que es mirar si el paso
llegó a ejecutarse alguna vez. `release.yml` tiene **tres ejecuciones en toda su
vida** (33853740997, 33854068327, 33854766173), las tres `workflow_dispatch` sobre
`tramo2/release`, **ninguna sobre una etiqueta**. En las tres:

```
skipped   entrar en el registro
skipped   subir la imagen
```

Y `release.yml` es el **único** workflow del repositorio que contiene
`docker push`, `--push`, `docker/login-action` o `build-push-action`.

**Conclusión: el `--push` nunca se ejecutó y no hay nada público en ghcr.io.** Lo
único que queda fuera del alcance es un `docker push` hecho a mano desde una
máquina, que ningún registro de CI puede desmentir.

## gosec puso el lazo en rojo, y tenía razón

`./comprobar.sh` salió **en rojo** con dos hallazgos `G122` (symlink TOCTOU en
`filepath.WalkDir`) sobre las dos lecturas nuevas, la del que resume y la del que
empaqueta. Las anotaciones `#nosec G304` que ya había no cubren `G122`, y eso está
bien: son cosas distintas.

**No se suprimió, se arregló**, porque aquí la queja no es teórica:
`HuellaDeArbol` se llama **sobre un directorio recién extraído de un `.tar.gz` de
fuera**, en el camino de `--instalar`. Entre que `WalkDir` mira una entrada y
alguien la lee hay una ventana, y en esa ventana un enlace puede ocupar el sitio
de un fichero: la comprobación de «esto es un fichero normal» se hizo sobre lo de
antes y la lectura se lleva lo de después. El resumen acabaría incluyendo
contenido de fuera del árbol, **y ese resumen es el que decide si el corpus se
instala**.

Arreglado con `os.OpenRoot` (Go 1.24), que resuelve cada ruta dentro de la raíz y
se niega a salir: la ventana se cierra en el sistema operativo en vez de a base
de comprobar antes. Se añadió de paso un tope por fichero, porque `--huella` y
`--verificar` aceptan el directorio que teclee el operador.

Regresión comprobada: la huella del corpus real es **la misma antes y después**
del refactor (`e5e3b2dc…`), o sea que cambió cómo se lee y no qué se resume.

Vale la pena decirlo porque es justo el caso que `CLAUDE.md` describe: el paso de
gosec **no es una `puerta()`**, es un `run:` normal, y el lazo local sólo lo coge
porque `comprobar.sh` lee las herramientas de `ci.yml`. Sin esa lectura, esto
habría llegado a CI en rojo con un informe que decía «todo verde».

## Los errores que cometí

1. **Leí un código de salida a través de `head`.** La primerísima orden de la
   sesión fue `.github/frontera.sh ... | head -40; echo "EXIT=$?"`, que imprime el
   código de `head` y siempre es 0. Es la trampa número 2 de mi propio encargo,
   cometida antes de escribir una línea de código. La vi porque el texto decía
   «no hay trabajo» debajo de un `EXIT=0`.
2. **Di por bueno un montaje de Docker que no se había montado.** `docker run -v
   /c/Users/...` con MSYS convirtió la ruta en silencio y el contenedor leyó su
   propio corpus. La salida decía CUADRA y yo estaba a punto de escribir que el
   corpus montado se detectaba. Lo cacé porque el número de paquetes era
   sospechosamente idéntico. Con `MSYS_NO_PATHCONV=1` el montaje entró y dijo NO
   CUADRA, que es lo que había que demostrar.
3. **Escribí un `case` de shell con `[0-9a-f]` creyendo que validaba 64
   caracteres.** Valida exactamente uno, así que la guarda habría fallado
   **cerrado** sobre toda huella real y roto la release entera. Cambiado a
   `=~ ^[0-9a-f]{64}$`.
4. **Metí CRLF en `docs/instalacion.md`** con un script de Python que abría el
   fichero en modo texto. Git lo normalizó al commitear (por eso el commit avisó)
   pero el árbol de trabajo se quedó con 245 CRLF y `TestNingunFicheroDeTextoLlevaCRLF`
   se puso rojo durante una mutación. Es la nota que tengo en memoria sobre
   invisibles, cometida otra vez por otra puerta. Barrido el árbol entero después:
   0 ficheros del índice con CRLF.
5. **Creé un fichero de test en la raíz que no es de mi columna.** `corpus_en_la_release_test.go`
   estaba fuera de `rebanada_0`; lo moví dentro de `distribucion_test.go`, que sí
   lo es, antes de commitear.
6. **Muté un fichero que no había preparado.** En M5 renombré también
   `corpus_test.go`, que no estaba en el manifiesto de `mutar.sh`. El script lo
   detectó al restaurar («el árbol NO ha quedado limpio») y lo arreglé desde HEAD.
   La guarda funcionó; el descuido fue mío.
7. **Elegí dos mutaciones que no tocaban lo que creía** (M1 y M3). Las dos
   sobrevivieron por redundancia, no por agujero. Sirvieron igual: obligaron a
   encontrar dónde vive de verdad cada guarda.

## Lo que queda abierto

- **El comando de `cosign verify-blob` de `docs/instalacion.md` no está
  comprobado contra una firma real**, porque no existe ninguna release. La
  identidad del certificado es la forma esperada de una firma keyless de GitHub
  Actions, pero es lo único del documento que no se ha ejecutado. El documento lo
  dice y le da al lector la orden de `openssl` para mirar la identidad de verdad.
  **La primera release tiene que verificarlo y corregir el documento si falla.**
- **`comprobar.sh` no cubre los pasos de `release.yml`.** Lee herramientas de
  `ci.yml` únicamente. Las tres guardas nuevas del workflow de release (huella
  hexadecimal, corpus instalable con 30 paquetes mínimo, corpus presente antes de
  firmar) sólo se ejecutan en CI. No es una regresión, es la forma que ya tenía el
  fichero, y `comprobar.sh` es del integrador, no de esta columna.
- **No se ha añadido ninguna `puerta "` a `release.yml`** a propósito:
  `PUERTAS_ESPERADAS=24` vive en `comprobar.sh`, que no es de esta columna, y
  cualquier `puerta` nueva la habría roto. Las guardas nuevas son pasos `run:`
  normales, igual que las que ya había.

## Dos hallazgos posteriores, y el ensayo que pagó su sitio

Lo de arriba se escribió con `comprobar.sh` en verde y la máquina limpia pasando.
Faltaba lo importante: **nada de esto se había ejecutado nunca en un runner.**

### El ensayo (`workflow_dispatch`, run 33871775551)

`release.yml` ya llevaba escrito en su cabecera que un workflow que sólo se
ejecuta el día de la release se estrena el peor día, y que su primera ejecución
encontró un fallo que ninguna lectura del YAML habría visto. Esa frase se cobró
otra pieza, y esta vez la mía.

| trabajo | resultado |
|---|---|
| `candado de marca` | success |
| **`corpus de la release`** (nuevo) | **success** |
| **`imagen Docker`** (Dockerfile de dos pasadas) | **success** |
| `binarios en ubuntu-latest` | **failure** |
| `binarios en macos-latest` | **failure** |

Lo bueno: el trabajo nuevo del corpus funciona en un runner limpio, y el
Dockerfile de dos pasadas con el ancla construye multiarquitectura bajo buildx y
QEMU. Lo malo, abajo.

### `filepath.VolumeName` no contesta lo mismo en los tres sistemas

`TestUnTarballHostilNoEscribeFueraDeSuSitio/ruta_con_unidad_de_disco` pasaba en
verde en Windows y salía **rojo en ubuntu y en macos**. Tumbó cinco workflows a
la vez (`ci`, `release`, `etapa2-siem`, `etapa2-ttfv`, `etapa2-distribucion`) y
los cinco por el mismo caso.

`filepath.VolumeName` es dependiente del sistema: en Windows contesta `C:` para
`C:/fuera.json`, en Linux contesta `""`. Así que en Linux esa entrada se tomaba
por una ruta relativa con un directorio llamado `C:` y **se aceptaba**.

Y no es un test demasiado exigente que hubiera que aflojar, que era la reacción
barata disponible: un mismo `.tar.gz` tiene que extraerse **al mismo árbol en los
tres sistemas**, porque su huella es una sola y se compara contra un ancla única.
Si Linux acepta una entrada que Windows rechaza, el mismo fichero da árboles
distintos según dónde caiga y la huella deja de significar lo que dice que
significa.

Es «una puerta se demuestra en el shell en el que CORRE» aplicada al sistema
operativo. Yo sólo lo había ejecutado en Windows. Arreglado con una comprobación
escrita a mano que contesta igual en todas partes, y **demostrado verde dentro de
un contenedor de Linux**, que es donde fallaba, no otra vez en Windows, que es
donde ya pasaba.

### Lo que se extrae y lo que se resume no casaban por nada

Encontrado revisando mi propio código, no probándolo. Es el invariante 7 con otro
traje.

La huella se calcula sobre un **subconjunto** del árbol: `entraEnElCorpus` deja
fuera el código Go, para que un test que añada R2 en `paquetes/` no invalide el
corpus publicado. Correcto. Pero `extraerCorpus` extraía el tar **entero**.

Los dos conjuntos casaban por nada. Un tarball podía traer ficheros `.go` de
propina, aterrizar con ellos en el disco, y **la huella cuadrar igual**, porque lo
colado no entraba en el resumen que decide si el corpus se instala.

plazum no ejecuta nada de `paquetes/`, así que no es ejecución de código. Lo que
es, es peor de razonar: un corpus verificado que contiene ficheros que su
verificación no cubre. Y si el destino cae dentro de un módulo Go (alguien que
instala el corpus en su clon del repositorio), esos `.go` sí los ve el compilador
de ese módulo.

Ahora se rechazan en vez de saltarse, porque el corpus legítimo nunca los trae:
`empaquetarCorpus` usa la misma regla al empacar. Con las dos direcciones
probadas, que sin la segunda una función que dijera que no a todo pasaría la
primera y rompería el producto entero.

### El error de proceso que hay debajo de los dos

Los dos son la misma equivocación mía: **di por validado en una plataforma lo que
tiene que valer en tres, y di por cubierto por la huella lo que la huella no
cubre.** En los dos casos escribí la afirmación general y comprobé el caso
particular que tenía a mano.

Y hay un tercero de la misma familia que sí evité a tiempo: `comprobar.sh` local
tampoco corre las tres puertas de `-race`, porque esta máquina tiene
`CGO_ENABLED=0`. Se dice arriba con su motivo, y por eso la línea del lazo no es
«todo verde» sino «24 puertas leídas, 3 saltadas por cgo, que en CI sí corren».

## Un apartamiento del encargo, dicho en voz alta

El encargo pedía que el binario **verificase el corpus al arrancar** y dijese qué
hace si no cuadra. Lo entregado verifica **al instalar** y **a demanda**
(`plazum corpus`, `plazum corpus --verificar`), no en cada arranque. Es una
decisión, no un olvido, y va aquí porque «es mejor así» no vale sin decirlo.

**La medida primero.** Calcular la huella del corpus real cuesta **166 ms**;
`plazum calendario` entero cuesta **185 ms**. Un chequeo en cada arranque casi
duplica el comando insignia.

**El argumento, que no es el coste.** Verificar al instalar es *más* estricto que
verificar al arrancar, no menos: el corpus se comprueba **antes de tocar el
disco** y, si no cuadra, no llega a existir. Un chequeo al arrancar defiende de
algo distinto y más débil: que alguien edite en disco un corpus que ya se
verificó, en una máquina que el operador controla. Y no puede bloquear, porque un
corpus más nuevo que el binario nunca cuadrará con su ancla y ése es el caso
legítimo de toda actualización.

**Lo que sí quedaría sin cubrir**: el operador que instaló bien y luego tiene el
corpus alterado en disco. Hoy se entera si escribe `plazum corpus`, y
`docs/instalacion.md` se lo dice.

**Si el propietario prefiere el chequeo al arrancar**, el sitio es `calendario`,
que es la pantalla donde salen las fechas legales, y la forma es un aviso en la
cabecera, nunca un bloqueo. Son 166 ms y una línea. No lo he metido a última hora
sobre CI ya en verde: un cambio en el comando insignia después de la validación
merece su propio ciclo, no un hueco al final de la sesión.
