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
