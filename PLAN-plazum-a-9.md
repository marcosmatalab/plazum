# PLAN plazum a 9

**Estado medido:** 20 de septiembre de 2026, HEAD `648a8c9a6341579c950f1e946802e18756d018e1`, rama `main`.
**Nota de empleabilidad hoy: 6,0. Objetivo: 9,0. Total: 21,75 horas.**

**Qué es plazum, en una frase:** un binario en Go sin dependencias externas que calcula qué normas de cumplimiento te aplican y en qué fecha exacta vence cada obligación, con la cita legal de cada cosa, y deja un expediente que un tercero puede verificar sin red y sin fiarse de quien lo emitió.

**Lo que este plan no es:** no toca el motor, no reescribe el corpus y no cambia una sola decisión de producto. La ingeniería ya mide 8 sobre 10. Lo que está roto es la portada, la procedencia y la señal externa, y eso son 21,75 horas de trabajo casi todo fuera del código Go.

---

## 0. Cómo usar este documento

Ábrelo con Claude Code en la raíz del repo y ve fase por fase. Seis fases, en orden. Cada fase tiene un criterio de terminado que es un comando con una salida esperada: si el comando no da eso, la fase no está cerrada y no se pasa a la siguiente.

Tres reglas para no romper nada por el camino:

1. **Una fase, una rama, un push.** `git switch -c fase-N-nombre`, trabajas, `bash comprobar.sh`, merge a main. Nada de mezclar fases.
2. **`bash comprobar.sh` antes de cada merge.** Tarda 817 segundos medidos. Son casi catorce minutos, así que lánzalo y vete a por café, pero no lo saltes: hay seis tests atados al texto del README y la fase 5 los va a tocar.
3. **Si una puerta se pone roja, no la bajes.** El repo tiene 26 puertas y la regla de la casa es que una puerta que se ablanda deja de ser puerta. Si algo se pone rojo, o el cambio está mal o la puerta hay que moverla explícitamente en el mismo commit y diciendo por qué.

Las horas están estimadas para trabajo con asistencia de Claude Code, no a mano.

---

## 1. Diagnóstico medido

Todo lo de esta tabla está ejecutado sobre el clon en `main`, HEAD `648a8c9`, con Go 1.24.7 en linux/amd64.

| Dimensión | Nota | Cifra medida y su comando | Qué le falta para 9 |
|---|---|---|---|
| **60 segundos** | **2** | `README.md`: 147 líneas, **2.616 palabras**, `grep -c '!\[' README.md` = **0 imágenes**, 0 badges, `grep -oE 'https?://' README.md \| wc -l` = **0 enlaces http**. 22 PNG en `superficies/pantallas/capturas/` sin enlazar en ningún sitio. | Badge de CI verde, dos capturas y enlace a release con binarios, todo en el primer scroll. Es lo más barato y lo más roto. |
| **20 minutos** | **9** | `GOPROXY=off go test ./...` da **EXIT=0 en 144 s**, 64 paquetes ok, 0 FAIL, 7 sin tests. `bash comprobar.sh` da **EXIT=0 en 817 s, 26 de 26 puertas**. Cobertura del núcleo **88,3 %** con `go test ./nucleo/... -coverprofile`. Todo sin claves y sin red. | Solo que `./comprobar.sh` arranque. Hoy da exit 126. Con eso es un 10. |
| **Examen hostil** | **8** | Las nueve cifras del README verificadas una a una, **todas ciertas**, todas derivadas por un test del árbol. `grep -cP '^\s*puerta ' .github/workflows/*.yml` suma **26**, de las que **24 bloquean cada empujón**. | Dos manchas: la frase «13 workflows de CI en verde» y `docs/pendientes.md`, que se contradice a sí mismo. Detalle abajo. |
| **Ingeniería** | **8** | Contratos de capa verificados por AST en CI. `go vet ./...` limpio, `gofmt -l $(git ls-files '*.go')` vacío, **0 marcadores TODO o FIXME reales** en producción. Pero `nucleo/corpus` está a **7.988 de un techo de 8.000** y `paquete.go` a **2.652 de 2.700**. | Partir `paquete.go`. No es cosmético: quedan 12 líneas de margen en el paquete, así que el próximo cambio ahí rompe CI. |
| **Procedencia** | **2** | `GUIA-CLAUDE-CODE.md:3` dice que el workspace está preparado «para que Claude Code lo desarrolle entero». `CLAUDE.md` son 70 KB **enlazados desde el README**. `.claude/` tiene 7 ficheros de agente. `git log --format=%ad --date=short \| sort \| uniq -c` da **609 commits en 18 días activos, pico de 127 el 04-09-2026**. 20 ramas `worktree-agent-*` publicadas. | Declararlo y encuadrarlo tú, con las puertas delante. Hoy el repo hace la peor lectura posible por ti y no ofrece ninguna otra. |
| **Señal externa** | **3** | `git ls-remote --tags origin` confirma que **`v0.1.0-rc1` sí está publicado**, pero es del 04-09-2026 y está **128 commits por detrás de main** (`git rev-list --count v0.1.0-rc1..origin/main`). Sus notas anuncian «33 paquetes, 222 relojes» y hoy `ls -d paquetes/*/ \| wc -l` da **20**. | Etiquetar `v0.1.0` de verdad. `release.yml` ya publica todo lo que hace falta. |
| **Encaje** | **7 backend / 2 AI** | `adaptadores/ia/` más `evals/` suman **1.649 líneas de producción sobre 75.187, o sea el 2,2 %**. La regla del repo es que la IA vive en adaptadores y el núcleo no la conoce, con puerta que corre la suite entera con la IA apagada. | Como backend ya encaja. Como AI no, y no merece la pena forzarlo. Ver ADR-002. |

### Hallazgos con fichero y línea

1. **`./comprobar.sh` no arranca en ningún clon Unix.** `git ls-files -s | grep '\.sh$'` devuelve los **9 ficheros en modo `100644`**. Ejecutarlo da `Permission denied`, exit 126. En un repo cuya tesis entera es «no te fíes de mí, ejecútalo», el comando de ejecutarlo está roto. `bash comprobar.sh` sí funciona y da 26 de 26 en verde.

2. **`GUIA-CLAUDE-CODE.md:3`**: el repo declara por escrito que lo construyó un agente. Textual: «Este workspace está preparado para que Claude Code lo desarrolle entero contigo al mando». Y el README enlaza `CLAUDE.md` como «las reglas».

3. **El historial lo confirma en un comando.** 609 commits en 18 días activos desde el 24-08-2026, 127 de ellos en un solo día, 182k líneas, y 27.686 líneas de mensajes de commit, o sea 45,5 líneas de media por commit. Buena noticia: `git log --format=%B | grep -ic 'co-authored-by: claude'` da **0**. La atribución no se filtró, pero la cadencia sí.

4. **31 de las 38 ramas remotas son borrables hoy.** Medido con `git merge-base --is-ancestor` más `git rev-list --count`: 20 `worktree-agent-*`, 4 `tramo2/*`, 4 `tramo3/*`, 2 `eslabon-*` y `bloque-cinco-puntos` están **todas mergeadas con 0 commits por delante**. Quedan 5 de dependabot sin mergear, de entre el 25-08 y el 08-09, y `actaira-action-evidence` con 2 commits, del 19-09, que es trabajo vivo. Después de la poda quedan **7 ramas**: `main`, las 5 de dependabot y la de trabajo.

5. **22 capturas generadas y 0 enlazadas.** `superficies/pantallas/capturas/` tiene claro y oscuro de once pantallas. El producto tiene seis pantallas accesibles con axe-core en cero violaciones y nadie las ve nunca.

6. **`docs/pendientes.md` se contradice a sí mismo.** 2.856 líneas, 125 cabeceras. Su preámbulo dice que «P0 bloquea la casilla, no entra aquí» y hay `## P0` en las **líneas 58 y 176**. Dice que «cuando algo se cierra, se borra de aquí» y hay un `### CERRADO` en la **línea 674**. Un revisor hostil encuentra eso en un minuto y a partir de ahí desconfía del resto, que es injusto porque el resto aguanta.

7. **Los techos de modularidad están a punto de morder.** `go test . -run TestLosPaquetesGrandesTienenTechoYNoSorpresa -v` imprime `nucleo/corpus: 7988 lineas en 22 ficheros (techo 8000)`. Doce líneas de margen.

8. **Un solo `continue-on-error` en todo el CI**, en `.github/workflows/release.yml:597`, sobre un `download-artifact` del job de ensayo, con cuatro líneas de comentario justificándolo. No es una puerta decorativa, es correcto. Lo digo porque es lo primero que busca un revisor y aquí sale limpio.

---

## 2. Decisiones tomadas, con su porqué y la alternativa descartada

### 2.1 El idioma: el repo se queda en español, y se añade una página en inglés

**Decisión.** El código, los comentarios, los mensajes de commit y la documentación se quedan **enteros en español**. Se traduce **una sola cosa**: un `README.en.md` de una página, menos de 600 palabras.

**Por qué.** El dominio de plazum es derecho español y europeo: BOE, ENS, RD 43/2021, Ley 39/2015, Reglamento 1182/71. Los identificadores del corpus son los de las normas reales y no se traducen sin mentir. Traducir 75.187 líneas de producción y 24.129 de markdown son semanas, y el resultado sería un repo que habla de la Ley 39/2015 en inglés, que es raro para todo el mundo. Para una empresa española de GRC o legaltech, que es el encaje natural, el español es un plus y no un problema.

**La alternativa descartada: traducirlo todo a inglés.** Descartada por coste, por semanas de trabajo, y porque destruye la coherencia del dominio. Pero tiene un coste real que hay que asumir: **hoy un revisor que no hable español no puede evaluar este repo en absoluto**, ni la portada ni un comentario. El `README.en.md` no resuelve eso, solo abre la puerta lo justo para que alguien decida si le interesa.

**Lo que se traduce exactamente:** titular, qué es y para quién, la tabla de las cinco cifras con sus comandos, dos capturas, cómo ejecutarlo en tres comandos, y una nota de una línea diciendo que el dominio es derecho español y europeo y que por eso el resto del repo está en castellano.

### 2.2 El repo se presenta como backend y plataforma, no como AI

**Decisión firme: plazum va como pieza de backend y plataforma.** Cero horas de código nuevo de IA.

**El argumento completo, para defenderlo en una entrevista.** Si en una entrevista te preguntan «¿y dónde está la IA aquí?», esta es la respuesta y es mejor que la que darías si metieras un RAG:

> En plazum la IA está deliberadamente acotada, y esa es una decisión de diseño, no una carencia. El producto calcula fechas de vencimiento legales con consecuencias reales: si el motor se inventa un plazo, alguien incumple. Por eso el núcleo es determinista y no conoce la IA, hay un test que verifica por AST que el núcleo no importa nada de `adaptadores/`, y una puerta de CI que corre la suite entera con la IA apagada y tiene que salir verde. La IA vive en adaptadores: un verificador que comprueba cada cita por hash contra el texto de la norma, un interruptor global, y un arnés de evals con su conjunto de casos. Y hay un linter legal que impide que el paquete de ISO o PCI DSS contenga el texto de la cláusula, así que la IA de este producto literalmente no puede inventarse el contenido de una norma que no tiene delante.

Eso es ingeniería de sistemas con IA y gobierno de lo que el modelo puede tocar, que es más difícil de enseñar y más caro de aprender que montar un RAG. Está construido, medido y con puerta.

**La alternativa descartada: añadir un RAG sobre el corpus para engordar el 2,2 %.** Descartada por tres razones medidas. Primera, choca de frente con lo que hace fuerte al repo: ampliar la superficie de IA debilita el argumento del núcleo determinista, que es el único argumento que plazum tiene y nadie más. Segunda, un RAG creíble con dataset, línea base, métricas y puerta son 25 a 40 horas, no 6, y por debajo de eso es un adaptador con un prompt, que en 2026 no puntúa. Tercera, y es la que decide: ya hay una historia de IA mejor y está sin contar.

**Consecuencia práctica:** si el puesto es de AI, plazum va como **segunda** pieza con el ángulo de arriba, nunca como principal. Para eso hace falta otro repo, y eso está fuera de este plan.

### 2.3 El corpus no se separa del producto

**Decisión.** Un solo repo. El corpus se queda en `paquetes/`.

**Por qué.** Separarlo suena limpio pero rompe tres cosas que hoy funcionan y están medidas. Una: el corpus viaja **dentro** del binario y de la imagen, y hay un test que lo comprueba (`TestLaImagenTraeCorpusYExpedienteParaQueArranqueSola`). Partirlo significa inventar versionado entre dos repos y un mecanismo de descarga, que es exactamente el tipo de dependencia de red que el producto presume de no tener. Dos: hay 808 casos dorados que corren el corpus contra el motor en cada ejecución de `comprobar.sh`; con dos repos eso pasa a ser CI cruzado. Tres: a efectos de portfolio, partirlo deja dos repos mediocres en vez de uno bueno.

**La alternativa descartada: `plazum` y `plazum-corpus` separados**, con el argumento de que las licencias son distintas (código AGPL-3.0, datos Apache-2.0). Es verdad que son distintas, pero eso ya está resuelto con la declaración de licencias por directorio y un linter que la verifica. No hace falta un repo para eso.

### 2.4 Qué se hace con `v0.1.0-rc1`

**El problema medido.** El tag está en origin (`git ls-remote --tags origin` lo confirma), es del 04-09-2026, está **128 commits por detrás de main**, y sus notas dicen «33 paquetes, 222 relojes» cuando hoy el corpus son **20 paquetes**. El que se lo baje hoy se lleva el corpus viejo y unas notas que le mienten. Es el peor activo público del repo: es señal externa negativa.

**Decisión: no se borra, se entierra debajo de una release buena.**

Pasos concretos, en este orden:

1. Etiquetar `v0.1.0` de verdad sobre main. Como no lleva sufijo, `release.yml` activa `make_latest: true` y `prerelease: false`, así que **pasa a ser la release actual del repo automáticamente** y el rc deja de ser lo que la portada ofrece.
2. Editar a mano las notas del rc en GitHub añadiendo arriba una línea: «Obsoleta. El corpus se reorganizó el 10-09-2026 y esta release describe 33 paquetes que hoy son 20. Usa v0.1.0.»
3. **No borrar el tag ni la release.** La firma de `v0.1.0-rc1` está en Rekor, que es un log de solo añadir, y borrar el tag deja la firma apuntando a algo que no existe. Es peor que dejarlo con su aviso.

**La alternativa descartada: borrar el tag y la release.** Descartada porque Rekor es irreversible, porque borrar releases se ve en el historial del repo y porque un rc obsoleto con su aviso escrito es, honestamente, buena señal: demuestra que sabes qué publicaste y cuándo dejó de valer.

### 2.5 El alcance que NO se toca

Esto es tan importante como lo que sí se hace. **No se toca nada de:**

- El motor de plazos, el corpus, el ledger, el expediente, el anclaje RFC 3161. Todo eso está en verde y medido.
- Las seis pantallas, el `serve`, OIDC, SCIM, el export a SIEM.
- Las 26 puertas de CI. Ninguna se baja, ninguna se quita. La única que se toca es el techo de `paquete.go` en la fase 4, y se toca **bajándolo**, no subiéndolo.
- `ETAPAS.md` como plan: se mueve de sitio, no se reescribe ni se marcan casillas que no estén hechas.
- Las 5 ramas de dependabot: se mergean o se cierran, pero eso es trabajo de dependencias, no de este plan.
- `docs/censo-relojes.md` (2.493 líneas): es la medición que sostiene el orden de autoría del corpus. Se queda entera.

---

## 3. Inventario de cambios

### 3.1 Borrar

**Ramas remotas, 31 en total.** Todas verificadas mergeadas con 0 commits por delante:

```bash
# Las 20 de agente, una por una para que quede constancia de cuáles:
git push origin --delete \
  worktree-agent-a0b26ad2598e8d429 worktree-agent-a0d0c24ae2bcfa375 \
  worktree-agent-a230e93799387189f worktree-agent-a25c20ba851ab5c55 \
  worktree-agent-a2e897d9a065aa124 worktree-agent-a5c92884f63f13158 \
  worktree-agent-a81742c7ba6e25535 worktree-agent-a98d0a3f118be953e \
  worktree-agent-a992beee01f155755 worktree-agent-a9b18547d2e55a767 \
  worktree-agent-aa647248ba0d10abe worktree-agent-aa6f43ac074439543 \
  worktree-agent-aa8f6becc4c64b1d7 worktree-agent-aae2e906f926fa784 \
  worktree-agent-aaf99de3c23fc13f1 worktree-agent-ab1714fffe65688ea \
  worktree-agent-abb2b161037763f03 worktree-agent-ac7b9ec938a917f4c \
  worktree-agent-ad3ccc59213937063 worktree-agent-ae5808dc5a68c0a3e

# Las 8 de tramo y eslabón:
git push origin --delete \
  tramo2/nota tramo2/puente tramo2/release tramo2/valores \
  tramo3/corpus tramo3/ia tramo3/release tramo3/visual \
  eslabon-a2-a3 eslabon-de-la-prueba bloque-cinco-puntos
```

Son 20 más 11, igual a **31 nombres**. Verifícalo antes con el comando de la fase 1: tiene que listar exactamente esas 31. Las 7 que quedan después son `main`, las 5 de dependabot y `actaira-action-evidence`, y ninguna de esas se toca aquí.

**Ficheros y directorios:**

| Ruta | Líneas | Por qué se borra | Riesgo |
|---|---|---|---|
| `GUIA-CLAUDE-CODE.md` | 61 | Es el fichero que declara que lo desarrolla un agente. `grep -rn 'GUIA-CLAUDE' .` da **cero referencias** en todo el árbol. | **Ninguno.** Borrado limpio. |
| `.claude/` (7 ficheros) | - | `agents/revisor-hostil.md`, `agents/autor-corpus.md`, 5 comandos. Andamiaje de sesión visible en la raíz. | Ninguno para CI. Guárdalo fuera del repo si lo usas, que es útil. |
| `docs/hallazgos/` (19 ficheros) | **7.866** | Informes de auditorías internas ya cerradas. Nadie externo los lee y hacen que el repo parezca una obra en curso. | Comprobar antes que ningún test los lee. |

**Total de markdown que se va: unas 8.000 líneas de 24.129.**

### 3.2 Modificar

| Fichero y línea | Qué cambia | A qué exactamente |
|---|---|---|
| Los 9 `.sh` | modo `100644` a `100755` | `git update-index --chmod=+x` |
| `README.md:41-45` bloque `ingenieria` | La frase de los workflows | De «**13 workflows de CI** en verde» a «**24 de las 26 puertas de CI** corren en cada empujón y en cada pull request» (texto completo en la sección 8) |
| `README.md` entero | Reescritura a menos de 900 palabras | Estructura y texto literal en la sección 5 |
| `parrafo_de_ingenieria_test.go:58` | `bloqueMarcado(t, "README.md", "ingenieria")` | Se queda igual: el bloque de ingeniería **no se mueve**, es el corazón de la portada |
| `binario_test.go:112` | `leerDoc(t, rutaDelREADME)` | A `leerDoc(t, rutaDelPresupuestoBinario)`, constante nueva `= "docs/presupuesto-binario.md"` |
| `marcos_v1_test.go:50` | `rutaDelREADME = "README.md"` | Añadir `rutaDeCoberturaV1 = "docs/cobertura-v1.md"` y usarla en `:413` y `:534` |
| `instantanea_test.go:120` | `leerDoc(t, rutaDelREADME)` | A `rutaDeCoberturaV1` |
| `coincidencia_casual_test.go:39,76` | `leerFichero(t, "README.md")` | A `"docs/cobertura-v1.md"`, en los dos sitios |
| `cuentas_publicadas_test.go:92-93` | Tabla con `"README.md"` y regex `(\d+) hitos y \d+ casos dorados` | A `"paquetes/CORPUS.md"`, que es donde se va esa frase |
| `modularidad_test.go:94` | `TechoDeNucleoCorpus = 8000` | A `8100` (margen tras el reparto) |
| `modularidad_test.go:98` | `TechoDePaqueteGo = 2700` | A `400` |
| `CLAUDE.md` | Se mueve a `docs/invariantes.md` | Y `sed -i 's/CLAUDE\.md/docs\/invariantes.md/g'` sobre las **22 referencias en prosa** (ver aviso abajo) |
| `docs/pendientes.md` | 2.856 líneas a menos de 400 | Tabla de P1 y P2 abiertos. Lo narrativo a `docs/bitacora/` |
| `ETAPAS.md` | Se mueve a `docs/ETAPAS.md` | Actualizar el enlace del README |

**Aviso sobre `CLAUDE.md`, medido:** hay **22 referencias** en el árbol y **todas son comentarios en prosa** del tipo «invariante 8 de CLAUDE.md». Ningún test abre el fichero. Están en `comprobar.sh:17`, `.github/frontera.sh:73`, `.github/empujar.sh:20` y `:184`, `.github/workflows/etapa2-accesibilidad.yml:15`, y 17 comentarios en ficheros `.go`. Renombrarlo es seguro, pero hay que pasar el `sed` o quedan 22 citas apuntando a un fichero que ya no existe, que es justo el tipo de prosa caducada que este repo persigue.

### 3.3 Crear

| Ruta nueva | Contenido | Viene de |
|---|---|---|
| `README.en.md` | Portada en inglés, menos de 600 palabras | Nuevo |
| `docs/invariantes.md` | Las reglas de ingeniería | `CLAUDE.md`, quitándole el andamiaje de sesión |
| `docs/arquitectura.md` | El corte hexagonal con diagrama y el test que vigila cada flecha | Nuevo |
| `docs/presupuesto-binario.md` | El bloque `binario:inicio`/`fin` entero, con sus marcadores | `README.md:15-27` |
| `docs/cobertura-v1.md` | El bloque `cobertura-v1:inicio`/`fin` entero, con sus marcadores | `README.md:53-68` |
| `docs/incidentes.md` | La familia de notificación de incidente, las 10 normas y los 3 casos raros | `README.md`, párrafo de la familia |
| `docs/ETAPAS.md` | El plan | `ETAPAS.md` movido |
| `docs/bitacora/hallazgos.md` | Una línea por hallazgo cerrado con enlace al commit | Resumen de `docs/hallazgos/` |
| `docs/bitacora/pendientes-historico.md` | Lo narrativo y lo cerrado | `docs/pendientes.md` |
| `docs/demo.gif` | GIF de `plazum demo` y `plazum calendario` | Nuevo, con `vhs` |
| `.github/workflows/` | Sin cambios | - |

Las capturas ya existen en `superficies/pantallas/capturas/`, no hay que generarlas.

---

## 4. Fases de ejecución

### Fase 1: higiene. 0,75 h. Cierra en 6,5

**Objetivo:** que el primer comando del repo arranque y que un recruiter no vea 20 ramas con la palabra «agent».

**Pasos:**

1. Comprueba primero qué ramas son borrables de verdad, no te fíes de la lista. Este es el comando exacto que usé para medirlo, y excluye `main` y `HEAD` y exige 0 commits por delante:
```bash
git fetch --prune
git for-each-ref --format='%(refname:short)' refs/remotes/origin \
  | grep -vE 'origin/HEAD|^origin/main$|^origin$' \
  | while read b; do
      if git merge-base --is-ancestor "$b" origin/main 2>/dev/null; then
        [ "$(git rev-list --count origin/main..$b)" = "0" ] && echo "BORRABLE $b"
      fi
    done
```
Tienen que salir **31 líneas**: 20 `worktree-agent-*` y 11 más. Si sale otra cosa, para y mira por qué antes de borrar nada.

2. Permisos de ejecución:
```bash
git update-index --chmod=+x comprobar.sh \
  .github/frontera.sh .github/esperar-ci.sh .github/presupuesto.sh \
  .github/puerta.sh .github/empujar.sh .github/mutar.sh \
  docs/lanzamiento/maquina-limpia.sh docs/lanzamiento/generar.sh
git commit -m "los nueve .sh son ejecutables, y el primer comando del repo arranca"
```

3. Borrar las ramas con los comandos de 3.1.

4. `git rm GUIA-CLAUDE-CODE.md && git rm -r .claude/`

**Criterio de terminado:**
```bash
git ls-files -s | grep '\.sh$' | grep -c 100755    # tiene que dar 9
./comprobar.sh --ayuda 2>&1 | head -1              # NO puede decir "Permission denied"
git branch -r | grep -v HEAD | wc -l               # tiene que dar 7
ls GUIA-CLAUDE-CODE.md .claude 2>&1                # "No such file or directory"
```

### Fase 2: la release. 1 h. Cierra en 7,0

**Objetivo:** que haya binarios descargables de hoy y que el rc obsoleto deje de ser lo que la portada ofrece.

**`release.yml` ya lo hace todo**, verificado job por job: matriz de 3 plataformas (`:233-247`), `SHA256SUMS-<goos>` (`:339-342`), SBOM CycloneDX con `anchore/sbom-action@v0` (`:539-541`), firma cosign keyless a Rekor (`:543-552`), imagen a ghcr.io (job `imagen`), y publicación con `softprops/action-gh-release@v3` (`:561-565`) con la lógica de `prerelease` y `make_latest` derivada de la forma del tag. No le falta nada. Lo que falta es el tag.

**Pasos:**

1. Escribe el mensaje del tag anotado con las cifras de HOY, no las del rc. Las cifras buenas: 20 paquetes, 88,3 % de cobertura del núcleo, 3.265 casos ejecutados, 26 puertas, binario de 12,1 MiB.
2. `git tag -a v0.1.0 -F notas-v0.1.0.txt && git push origin v0.1.0`
3. Vigila la ejecución. El job `candado` decide si publica, `ensayo` cuenta qué habría salido si algo falla.
4. Cuando termine, edita las notas de `v0.1.0-rc1` en GitHub y añade arriba el aviso de obsolescencia de 2.4.

**Criterio de terminado:** la release `v0.1.0` aparece como **Latest** en GitHub, con al menos 3 binarios, 3 ficheros `SHA256SUMS-*`, `sbom.cdx.json` y las firmas `.sig` y `.pem`, todo descargable. Y el rc aparece marcado como pre-release con su aviso.

**Lo que las horas no garantizan aquí:** el workflow se ha ejecutado de verdad una sola vez en su vida, el 04-09-2026, y aquella vez **falló el job `imagen`** por multiarquitectura en un runner limpio. Reserva margen por si vuelve a pasar. Si falla, el job `ensayo` te dice exactamente qué habría salido.

### Fase 3: procedencia. 2,5 h. Cierra en 7,5

**Objetivo:** que la historia de cómo se construyó la cuentes tú, con las puertas delante, y no la deduzca el revisor del `git log`.

**Pasos:**

1. `git mv CLAUDE.md docs/invariantes.md`
2. Quítale a `docs/invariantes.md` el andamiaje de sesión: comandos de Claude Code, referencias a `/etapa` y `/puerta`, el bucle de trabajo. Deja las reglas de ingeniería, que son las buenas: la puerta que nunca se ha visto fallar no es una puerta, el emparejamiento por identidad firmada, el valor cero restrictivo en frontera de confianza, la IA en adaptadores, la frontera legal.
3. Arregla las 22 citas:
```bash
grep -rl 'CLAUDE\.md' --include='*.go' --include='*.sh' --include='*.yml' . \
  | xargs sed -i 's|CLAUDE\.md|docs/invariantes.md|g'
```
4. Añade al README la sección de procedencia con el texto literal de la sección 5.4.

**Criterio de terminado:**
```bash
grep -rn 'CLAUDE\.md' . --include='*.go' --include='*.sh' --include='*.yml' | wc -l   # 0
ls CLAUDE.md 2>&1                                                                     # no existe
bash comprobar.sh                                                                     # EXIT=0, 26/26
```

### Fase 4: partir `paquete.go`. 3 h. Cierra en 8,0

**Objetivo:** quitar la bomba de relojería de los 12 puntos de margen, y que el fichero mayor del repo deje de ser un argumento en contra.

Esto **no es cosmético**, y el propio repo ya lo diagnosticó. El mensaje de error de `modularidad_test.go:210` dice textualmente que el fichero «lleva dentro el esquema entero, buena parte del linter y la clasificación de campos de texto» y que hay que mirar «si lo que ha crecido es una de esas tres y puede irse a su fichero». El corte sale de ahí.

**Pasos:** crear cuatro ficheros en `nucleo/corpus/`, mismo paquete, cero cambios de API:

| Fichero | Qué se lleva | Líneas origen |
|---|---|---|
| `clase.go` | `Clase`, `LicenciaFuente`, `licenciasPorClase`, `licenciasProhibidas`, los límites referenciales | 31 a 219 |
| `vigencia.go` | `Vigencia`, `rango`, `LecturaVigencia`, `fechaDeVigencia`, `interpretar`, `cubre`, `interseccion`, `VigenteEn`, `EnVigor`, `VigentesEn` | 271 a 511 |
| `esquema.go` | `Atributo`, `TipoEntidad`, `Pregunta`, `Plantilla`, `Temporalidad`, `HitoSpec`, `TopeSpec`, `LecturaSpec`, `RegimenSpec`, `Escalon`, `Dorado`, `EsperadoDorado`, `Obligacion`, `Paquete` | 512 a 1167 |
| `validar.go` | `campoTexto`, `camposDeTexto` (381 líneas), `validarFronteraLegal`, `validarLicenciaFuente`, `validarVigencias`, `Validar` (338 líneas), `validarEsperadoDeDorado` | 1168 a 2652 |

Y en el **mismo commit**, bajar los techos en `modularidad_test.go`: `TechoDePaqueteGo` de 2700 a **400**, `TechoDeNucleoCorpus` de 8000 a **8100**.

**Criterio de terminado:**
```bash
wc -l nucleo/corpus/*.go | sort -rn | head -5     # ninguno por encima de 900
go build ./... && go vet ./...                     # limpios
go test ./nucleo/... -coverprofile=cover.out && go tool cover -func=cover.out | tail -1
# la cobertura no puede bajar de 88,3 %
```

**Opcional dentro de esta fase, 2 h más:** extraer de `cmdServe` (`cmd/plazum/serve.go:124`, 472 líneas) los tres bloques nombrados a `construirAlmacenDeSesiones`, `construirLectorDeLatido` y `decidirCookieSegura`. Lo marco opcional con datos: de esas 472 líneas, **190 son comentario**, y tiene **32 `if`, 0 bucles y 0 `switch`**. Es una raíz de composición plana, no código complejo. Bajarla a 200 líneas queda mejor pero no cambia la nota. Si vas justo de tiempo, sáltalo.

### Fase 5: la portada. 4 h. Cierra en 8,5. **Esta es la fase delicada**

**Objetivo:** que alguien entienda qué es plazum en 60 segundos y vea una prueba sin clonar.

**El peligro, y hay que decirlo antes de los pasos.** El README tiene tres bloques marcados con comentarios HTML y **cada uno está vigilado por un test que lo lee con una expresión regular**. Si mueves un bloque sin mover su test, pasan dos cosas malas a la vez: la portada queda limpia y **el CI se pone rojo**, y encima el rojo no dice «has movido un bloque», dice que el README ya no trae la frase esperada. Medido, el mapa es este:

| Bloque | Líneas | Test que lo vigila | Cómo lo lee |
|---|---|---|---|
| `binario:inicio`/`fin` | 15 a 27 | `binario_test.go:112` | `leerDoc(t, rutaDelREADME)` más `reTamanoDelREADME` (`:93`), que exige la frase literal `mide **X,Y MB** contra un presupuesto` |
| `ingenieria:inicio`/`fin` | 41 a 45 | `parrafo_de_ingenieria_test.go:58` | `bloqueMarcado(t, "README.md", "ingenieria")`, con la ruta **escrita a mano** en la llamada |
| `cobertura-v1:inicio`/`fin` | 53 a 68 | `marcos_v1_test.go:413` y `instantanea_test.go:71` | Regex `(?s)<!-- cobertura-v1:inicio -->.*?\*\*([0-9]+(?:,[0-9]+)?) %\*\*.*?<!-- cobertura-v1:fin -->` |

Más dos tests que leen el README sin marcadores: `coincidencia_casual_test.go:39` y `:76`, que cruza la cifra del README con la de `ETAPAS.md`, y `cuentas_publicadas_test.go:92-93`, con la regex `(\d+) hitos y \d+ casos dorados`.

La constante compartida es `rutaDelREADME = "README.md"` en `marcos_v1_test.go:50`.

**Procedimiento, bloque a bloque. Uno cada vez, con su test en verde antes de pasar al siguiente:**

1. **Mueve el bloque `binario` primero, que es el más simple.**
   - Crea `docs/presupuesto-binario.md` y pega el bloque **con sus dos marcadores HTML intactos**, de `<!-- binario:inicio -->` a `<!-- binario:fin -->`. Si te dejas un marcador, la regex no casa.
   - En `marcos_v1_test.go`, junto a `rutaDelREADME`, añade `rutaDelPresupuestoBinario = "docs/presupuesto-binario.md"`.
   - En `binario_test.go:112`, cambia `leerDoc(t, rutaDelREADME)` por `leerDoc(t, rutaDelPresupuestoBinario)`.
   - Verifica **solo ese test** antes de seguir:
     ```bash
     go test . -run TestElTamanoPublicadoDelBinarioEsElDeHoy -v
     ```
     Tiene que imprimir la línea `linux/amd64 nativo: NNNNNNNN bytes (12,1 MB), exacto contra el README` y PASS. Si dice que no encuentra el bloque, te dejaste un marcador.
   - Borra el bloque del README y deja en su sitio una línea con enlace: «El binario mide 12,1 MiB contra un presupuesto de 25 MB. Cómo se mide y por qué ha subido tres veces, en `docs/presupuesto-binario.md`.»

2. **Mueve el bloque `cobertura-v1`.** Mismo procedimiento, pero **son cuatro tests**, no uno: `marcos_v1_test.go:413` y `:534`, `instantanea_test.go:120`, y `coincidencia_casual_test.go:39` y `:76`. Añade `rutaDeCoberturaV1 = "docs/cobertura-v1.md"` y cámbialos los cuatro. Verifica:
   ```bash
   go test . -run 'Cobertura|CoincidenQue|MarcosV1|Instantanea' -v
   ```

3. **El bloque `ingenieria` NO se mueve.** Es el corazón de la portada: las cifras de casos, líneas y puertas. Se queda en el README y solo se le cambia la frase de los workflows (sección 8). Como `parrafo_de_ingenieria_test.go:58` lleva la ruta escrita a mano, no hay que tocar nada.

4. **La frase de hitos y dorados** se va a `paquetes/CORPUS.md`. Cambia en `cuentas_publicadas_test.go:92-93` el `"README.md"` de la tabla por `"paquetes/CORPUS.md"` y comprueba que la frase con la forma `N hitos y M casos dorados` está en el destino.

5. **Ahora sí, reescribe el README** con la estructura de la sección 5.

6. **Comprobación final de la fase, completa:**
   ```bash
   bash comprobar.sh        # 817 s, EXIT=0, 26/26
   wc -w README.md          # menos de 900
   grep -c '!\[' README.md  # 3 o más
   ```

### Fase 6: documentación y presentación. 6,5 h. Cierra en 9,0

**Objetivo:** que el aparato documental deje de leerse como «esto no cierra nada» y que haya algo en movimiento que ver.

**Pasos:**

1. **Podar `docs/pendientes.md`, 3 h.** Hoy son 2.856 líneas y 125 cabeceras, y se contradice a sí mismo en las líneas 58, 176 y 674. Se parte en dos: `docs/pendientes.md` queda como **tabla** de P1 y P2 abiertos, una línea por entrada, menos de 400 líneas. Todo lo narrativo y todo lo cerrado se va a `docs/bitacora/pendientes-historico.md`. Los dos `## P0` se arreglan o se degradan a P1 con fecha y motivo, porque el propio preámbulo del fichero dice que un P0 no vive ahí.
2. **Fusionar `docs/hallazgos/`, dentro de esas 3 h.** 19 ficheros y 7.866 líneas pasan a un `docs/bitacora/hallazgos.md` con una línea por hallazgo y enlace al commit que lo cerró.
3. **Mover `ETAPAS.md` a `docs/ETAPAS.md`.** Actualiza los enlaces. En la portada queda **una línea**: «Etapas 1 y 2 cerradas, etapa 3 al 54 %, 78 de 144 casillas». Hoy hay 66 casillas sin marcar visibles en la raíz y eso es lo que se lee como obra parada, aunque el 54 % esté hecho.
   Cuidado: **ocho tests leen `ETAPAS.md`** (`casillas_test.go`, `instantanea_test.go`, `ttfv_camino_test.go`, `coincidencia_casual_test.go`, `puertas_test.go`, `estado_del_plan_test.go`, `cmd/plazum/ia_en_linea_test.go`, `adaptadores/evidencia/entrevista_test.go`). Busca la ruta en cada uno y cámbiala antes de mover.
4. **El GIF, 1,5 h.** Con `vhs`, grabando `plazum demo` y `plazum calendario --pais=ES --sector=servicios-digitales --empleados=200`. A `docs/demo.gif`, menos de 3 MB y menos de 25 segundos.
5. **`README.en.md`, 2,5 h.** Lo de 2.1.
6. **`docs/arquitectura.md`, 1,5 h.** Una página con el corte hexagonal y, en cada flecha, **el test que la vigila**: `arquitectura_test.go:105` para el núcleo que no importa el exterior, `:250` para el núcleo que no lee el reloj, `modularidad_test.go:162` para el fan-out de `cmd/plazum`, `:179` para los techos de paquete.

**Criterio de terminado:**
```bash
wc -l docs/pendientes.md                                   # menos de 400
grep -cE '^## P0|^### CERRADO' docs/pendientes.md          # 0
wc -l $(git ls-files '*.md') | tail -1                     # menos de 12.000
ls README.en.md docs/arquitectura.md docs/demo.gif         # los tres existen
bash comprobar.sh                                          # EXIT=0, 26/26
```

---

## 5. El README final

Menos de 900 palabras hasta el final. Todo lo largo, a `docs/`.

### 5.1 Estructura

```
[badges: CI · cobertura · licencia · release · Go]

# plazum
[titular, 3 líneas]

![captura ancha: calendario-claro.png]

## Pruébalo en 30 segundos
[3 bloques de comandos]

## Las cinco cifras, con el comando que las produce
[tabla]

## Qué hace, en tres pilares
[los tres actuales, recortados a 2 frases cada uno]

![dos capturas en fila: hoy-claro.png y acta-claro.png]

## Cómo está construido
[6 líneas + enlace a docs/arquitectura.md]

## La IA, acotada a propósito
[párrafo literal en 5.5]

## Cómo se construyó
[párrafo literal en 5.4]

## Estado, licencia y aviso legal
[5 líneas]
```

### 5.2 El titular, literal

```markdown
# plazum

**El GRC de continuidad: no pierdas nunca la conformidad.**

Un solo binario en Go que sabe qué normas te aplican, qué tienes que hacer y para
qué fecha exacta, con la cita de cada cosa. Comprueba solo lo comprobable, agenda
y reclama lo humano, genera los documentos, y lo deja todo en un expediente que un
auditor puede verificar sin red y sin fiarse de ti.

Cero dependencias externas: `go.mod` no tiene ni una línea `require`.
```

### 5.3 Pruébalo en 30 segundos, literal

```markdown
## Pruébalo en 30 segundos

Con Docker, sin instalar nada:

```bash
docker run --rm ghcr.io/marcosmatalab/plazum
```

Con Go:

```bash
go install github.com/marcosmatalab/plazum/cmd/plazum@latest
plazum demo
```

O bájate el binario de tu plataforma en [la última release](https://github.com/marcosmatalab/plazum/releases/latest),
con sus sumas SHA256, su SBOM y su firma en Rekor.

Y para ver los relojes de una empresa tipo, sin configurar nada:

```bash
plazum calendario --pais=ES --sector=servicios-digitales --empleados=200
```

Cada fila sale marcada `[supuesto]`: es lo que le pasaría a una empresa de ese
perfil, no una conclusión sobre la tuya.
```

### 5.4 La tabla de las cinco cifras, literal

```markdown
## Las cinco cifras, con el comando que las produce

Ninguna de estas cifras está escrita a mano: las deriva un test del árbol y CI se
pone rojo si se separan. Puedes reproducirlas todas en tu máquina, sin red.

| Lo que se afirma | Comando | Lo que tiene que salir |
|---|---|---|
| Cero dependencias externas | `go list -deps -f '{{if not .Standard}}{{.ImportPath}}{{end}}' ./cmd/plazum \| grep -v '^github.com/marcosmatalab/plazum/'` | Nada. Ni una línea. |
| El binario mide 12,1 MiB, con 25 MB de presupuesto | `GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -trimpath -o plazum ./cmd/plazum && ls -l plazum` | 12.7 MB, o sea 12,1 MiB |
| La suite entera pasa sin salir a la red | `GOPROXY=off go test ./...` | `ok` en 64 paquetes, unos 145 s |
| El núcleo está al 88,3 % de cobertura, con puerta dura al 85 % | `go test ./nucleo/... -coverprofile=c.out && go tool cover -func=c.out \| tail -1` | `total: (statements) 88.3%` |
| 24 de las 26 puertas de CI corren en cada empujón | `bash comprobar.sh` | `26 puertas leidas, 26 ejecutadas`, unos 817 s |
```

### 5.5 El párrafo de la IA, literal

```markdown
## La IA, acotada a propósito

plazum calcula fechas de vencimiento legales con consecuencias reales, así que el
núcleo es determinista y no conoce la IA. No es una frase: hay un test que verifica
por AST que `nucleo/` no importa nada de `adaptadores/`, otro que comprueba que no
lee el reloj del sistema, y una puerta de CI que corre la suite entera con la IA
apagada y tiene que salir verde.

La IA vive en los adaptadores: un verificador que comprueba cada cita por hash
contra el texto de la norma, un interruptor global y un arnés de evals con su
conjunto de casos. Y hay un linter legal que impide que un paquete de ISO o PCI DSS
lleve dentro el texto de la cláusula, así que la IA de este producto no puede
inventarse el contenido de una norma: no lo tiene delante.

La doctrina completa, en [`docs/ia.md`](docs/ia.md).
```

### 5.6 El párrafo de procedencia, literal

```markdown
## Cómo se construyó

Escrito por una persona con asistencia intensiva de IA, bajo un régimen que no
admite que nada se dé por bueno sin comprobarlo: toda comprobación nace con su
fallo demostrado y la salida roja pegada en el commit, las cifras que se publican
las deriva un test del árbol y no se escriben a mano, y 24 de las 26 puertas corren
en cada empujón.

El historial es denso porque el ciclo fue corto. Lo que sostiene el resultado no es
el número de commits: es que `bash comprobar.sh` sale en verde con 26 puertas en 817
segundos, y que `GOPROXY=off go test ./...` pasa en 144 sin tocar la red.

Las reglas que sostienen todo esto están en [`docs/invariantes.md`](docs/invariantes.md).
```

### 5.7 Qué se va a dónde, con su test

| Bloque actual del README | Líneas | Destino | Test que hay que retargetear |
|---|---|---|---|
| Bloque `binario:inicio`/`fin` | 15 a 27 | `docs/presupuesto-binario.md` | `binario_test.go:112` |
| Bloque `ingenieria:inicio`/`fin` | 41 a 45 | **Se queda en el README** | `parrafo_de_ingenieria_test.go:58`, sin tocar |
| Bloque `cobertura-v1:inicio`/`fin` | 53 a 68 | `docs/cobertura-v1.md` | `marcos_v1_test.go:413` y `:534`, `instantanea_test.go:120`, `coincidencia_casual_test.go:39` y `:76` |
| Frase «N hitos y M casos dorados» | párrafo del corpus | `paquetes/CORPUS.md` | `cuentas_publicadas_test.go:92-93` |
| Historia del corpus: 33 a 20, esqueletos, censo | párrafo del corpus | `paquetes/CORPUS.md` | ninguno |
| Familia de notificación de incidente | párrafo final | `docs/incidentes.md` | ninguno |
| «Cómo se construye esto», las 5 reglas | sección entera | `docs/invariantes.md` | ninguno |

---

## 6. Estética y presentación

### Badges, en este orden

Cinco, ni uno más. En una sola línea, justo debajo del `# plazum`:

1. **CI** (`ci.yml`, rama main). Es el único que importa de verdad.
2. **Cobertura del núcleo: 88,3 %**. Badge estático de shields.io, porque no hay Codecov y no hace falta montarlo.
3. **Licencia: AGPL-3.0**.
4. **Última release**, enlazando a `/releases/latest`.
5. **Go 1.24**.

Nada de badges de «made with», ni contadores de visitas, ni «PRs welcome». Restan.

### Las capturas: 3 de las 22

De las 22 en `superficies/pantallas/capturas/`, usa las de tema **claro**, que se ven bien en GitHub con cualquier tema:

| Captura | Dónde | Por qué esa |
|---|---|---|
| `calendario-claro.png` | Ancha, justo debajo del titular | Es la que se entiende sin contexto: fechas, normas, cuenta atrás. Si alguien solo ve una imagen, que sea esta. |
| `hoy-claro.png` | En fila, tras los tres pilares | Enseña el producto en uso, no una lista. |
| `acta-claro.png` | Al lado de la anterior | Es la prueba visible del expediente verificable, que es el pilar 2. |

Las 19 restantes se quedan donde están. No se borran: las genera un test y son artefacto legítimo.

### El GIF

Uno solo, `docs/demo.gif`, en la sección de «Pruébalo en 30 segundos». Graba dos comandos seguidos: `plazum demo` y `plazum calendario --pais=ES ...`. Menos de 25 segundos y menos de 3 MB. Con `vhs` sale determinista y se puede regenerar.

### La marca

**No la toques.** `superficies/pantallas/estatico/plazum-marca.svg` existe, son 1.101 bytes, `viewBox="0 0 32 32"`, con `role="img"` y `<title>`, y dibuja un reloj cuya esfera está cortada por una línea vertical, con su porqué escrito dentro del fichero. Está bien hecha y es accesible. Úsala como avatar de la organización en GitHub y como favicon, que ya lo es. Cero horas.

### About y topics de GitHub

**About**, una frase, que es lo que sale en las búsquedas:

> Motor determinista de obligaciones de cumplimiento con reloj legal. Un binario en Go, cero dependencias, expediente verificable offline.

**Topics**, ocho: `go`, `golang`, `compliance`, `grc`, `regtech`, `nis2`, `dora`, `zero-dependencies`.

Y marca el enlace a la release en el panel de la derecha.

### El tono

Castellano directo, como escribes ya. **Sin guiones largos**: donde te pida un inciso, usa coma o dos puntos. Frases cortas. Cada afirmación con su comando al lado, que es lo que hace distinto a este repo. Y quita los superlativos que quedan en el README actual («que casi nadie publica», «la peor»): cuando la cifra es verificable, el superlativo resta credibilidad en vez de sumarla.

---

## 7. La interfaz: veredicto

El repo tiene seis pantallas accesibles, con axe-core en cero violaciones sobre 28 auditorías, un `serve` con sesiones, CSRF, OIDC y SCIM, y **nadie lo ve nunca**. Tres opciones:

| Opción | Coste al mes | Coste en horas | Qué resuelve | Qué rompe |
|---|---|---|---|---|
| **A. Demo pública desplegada** | 5 a 12 euros en un VPS pequeño, o 0 en free tier con arranque en frío de 30 s | 6 a 10 h de despliegue, TLS, dominio, y vigilancia continua | Máximo impacto: se toca el producto | Superficie de ataque real con OIDC y sesiones, datos de ejemplo que hay que mantener, y una demo caída es peor que ninguna demo |
| **B. Solo capturas y GIF** | 0 | Ya está en la fase 6 | Cubre los 60 segundos | No deja tocar nada |
| **C. Modo demo local en un comando** | 0 | 2 a 3 h | Se toca el producto de verdad, en la máquina de quien evalúa | Hay que clonar o instalar |

**Recomendación firme: B más C. Nada de desplegar.**

El porqué, con los datos del repo. Una demo pública de plazum no es una landing estática: es un servidor con sesiones, CSRF, OIDC y SCIM, o sea **credenciales y superficie de ataque real, mantenida por una persona que va justa de presupuesto**. El día que se caiga, o que alguien le meta datos raros, la demo pasa de activo a pasivo, y encima en un repo cuya tesis es la fiabilidad. El riesgo reputacional es asimétrico: funcionando suma poco más que un GIF, caída resta mucho.

Y la opción C es casi gratis porque **ya existe**: `plazum demo` funciona hoy, lo he ejecutado, instala una empresa de ejemplo, deriva sus obligaciones y enseña los relojes. Las 2 a 3 horas son solo para añadir un `plazum demo --serve` que, después de instalar el ejemplo, levante las seis pantallas en `localhost:8443` con ese estado ya cargado. Eso convierte «tengo seis pantallas» en «ejecuta esto y las ves», con cero coste mensual y cero superficie expuesta.

En la portada queda así: el GIF para los 60 segundos, y debajo una línea que diga «¿quieres tocar las pantallas? `plazum demo --serve` y abre localhost:8443».

**Revisa esta decisión** si en algún momento pagas ya un VPS para otra cosa. Entonces el coste marginal baja y la opción A se vuelve razonable.

---

## 8. Puertas de CI finales

Las 26, tal como salieron de la ejecución real de `bash comprobar.sh` (EXIT=0, 817 s). La columna de casos es la que imprime cada puerta.

| # | Puerta | Qué caza | Bloqueante | Casos (mínimo) | Workflow |
|---|---|---|---|---|---|
| 1 | suite completa | Todo | Sí | 3265 (2200) | `ci.yml` |
| 2 | suite completa con detector de carreras | Carreras de datos | Sí | 3265 (2200) | `ci.yml` |
| 3 | suite completa sin IA | Que el producto funcione con la IA apagada | Sí | 3265 (2200) | `ci.yml` |
| 4 | cobertura del núcleo | Cobertura por debajo del 85 % | Sí | 949 (200) | `ci.yml` |
| 5 | ingesta legal y vigilancia normativa | Ingesta de normas | Sí | 107 (70) | `ci.yml` |
| 6 | anclaje, RFC 3161 y pkcs7 vendorizado | Sellado de tiempo, con fuzzing | Sí | 123 (112) | `ci.yml` |
| 7 | SCIM, directorio y superficie HTTP | Aprovisionamiento desde el IdP | Sí | 79 (74) | `ci.yml` |
| 8 | cribador de marcas | Frontera de marcas | Sí | 29 (26) | `ci.yml` |
| 9 | verificador, búsqueda, ingesta, evidencia y evals | La capa de IA y evidencia | Sí | 213 (186) | `ci.yml` |
| 10 | huecos, claves muertas y descuadres | Texto de interfaz sin clave | Sí | 3 (3) | `etapa2-accesibilidad.yml` |
| 11 | frontera legal del catálogo | Que ISO y PCI no lleven texto de cláusula | Sí | 24 (20) | `etapa2-accesibilidad.yml` |
| 12 | inventario contra lo que pide la interfaz | Cadenas declaradas y usadas | Sí | 2 (2) | `etapa2-accesibilidad.yml` |
| 13 | adaptador de catálogo | El catálogo de cadenas | Sí | 72 (60) | `etapa2-accesibilidad.yml` |
| 14 | ensayo de copias y restauración | 1 copia sana y 9 rotas, cada una con su mensaje | Sí | 43 (38) | `etapa2-copias.yml` |
| 15 | suite completa en local | Build de distribución | Sí | 3265 (2200) | `etapa2-distribucion.yml` |
| 16 | superficies y secretos con detector de carreras | Concurrencia en las pantallas | Sí | 575 (150) | `etapa2-seguridad-web.yml` |
| 17 | export a SIEM | Formato de export | Sí | 30 (25) | `etapa2-siem.yml` |
| 18 | la orden plazum export | La CLI de export | Sí | 331 (50) | `etapa2-siem.yml` |
| 19 | autoservicio: actualizador, diagnóstico y CLI | `plazum doctor` y el actualizador | Sí | 397 (108) | `etapa2-ttfv.yml` |
| 20 | sintaxis de los pasos de CI, con control negativo | Que los workflows no llamen a `go test` a pelo | Sí | 2 (2) | `etapa2-ttfv.yml` |
| 21 | actualizador con detector de carreras | Carreras en el actualizador | Sí | 27 (25) | `etapa2-ttfv.yml` |
| 22 | derivación del calendario | El motor de 12 meses | Sí | 70 (47) | `etapa3-calendario.yml` |
| 23 | la orden plazum calendario | La CLI del calendario | Sí | 29 (18) | `etapa3-calendario.yml` |
| 24 | iCalendar: plegado, escapado, CRLF y UID estable | El `.ics` byte a byte | Sí | 49 (10) | `etapa3-calendario.yml` |
| 25 | frescura de las notas del marcador | Documentos envejecidos | **No**, cron diario | 11 (11) | `frescura.yml` |
| 26 | suite completa antes de empaquetar | Antes de publicar | **No**, solo en tag `v*` | 3265 (2200) | `release.yml` |

Más tres herramientas de seguridad bloqueantes en `ci.yml` que no pasan por `puerta()`: **govulncheck** (`:274`), **gosec** (`:276`) y **staticcheck** (`:298`). Más `gofmt` (`:44`), `go vet` (`:46`) y CodeQL en su propio workflow.

**Reparto medido**, con `grep -cP '^\s*puerta ' .github/workflows/*.yml`: `ci.yml` 9, `etapa2-accesibilidad` 4, `etapa2-ttfv` 3, `etapa3-calendario` 3, `etapa2-siem` 2, `etapa2-copias` 1, `etapa2-distribucion` 1, `etapa2-seguridad-web` 1, `frescura` 1, `release` 1. Total 26, de las que **24 bloquean**.

### La frase exacta y cierta para el README

Sustituye en el bloque `ingenieria` la frase «**13 workflows de CI** en verde» por esta:

> **24 de las 26 puertas de CI corren en cada empujón y en cada pull request**, repartidas en 9 de los 13 workflows. Las otras dos: una en cron diario, que vigila que la documentación no envejezca, y una en la etiqueta de release.

Es más larga y es cierta. La anterior no era falsa del todo, pero inducía a pensar que los 13 workflows pintan el estado de cada commit, y cuatro no lo hacen: `frescura.yml`, `vigilancia.yml` y `vigilancia-corpus.yml` corren por cron, y `release.yml` solo con un tag.

---

## 9. Guion de aceptación

Esto lo ejecuta un tercero en un clon limpio, sin claves y sin configurar nada. Los tiempos son los **medidos** en una máquina de 2 núcleos con Go 1.24.7; en un portátil moderno serán menores.

```bash
# 0. Clon limpio.                                              ~10 s
git clone https://github.com/marcosmatalab/plazum && cd plazum

# 1. Los scripts son ejecutables.                              instantáneo
git ls-files -s | grep '\.sh$' | grep -c 100755
# ESPERADO: 9

# 2. Cero dependencias externas.                               ~3 s
go list -deps -f '{{if not .Standard}}{{.ImportPath}}{{end}}' ./cmd/plazum \
  | grep -v '^github.com/marcosmatalab/plazum/'
# ESPERADO: ninguna salida

# 3. La suite entera, sin red.                                 144 s MEDIDOS
GOPROXY=off go test ./...
# ESPERADO: exit 0, "ok" en 64 paquetes, 0 FAIL, 7 sin tests

# 4. El binario y su tamaño.                                   ~34 s MEDIDOS
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -trimpath -o plazum ./cmd/plazum
ls -l plazum
# ESPERADO: 12.718.372 bytes aprox, o sea 12,1 MiB, bajo el presupuesto de 25 MB

# 5. El producto funciona.                                     ~2 s
./plazum demo
# ESPERADO: exit 0, "Ferretera Meridional SL", 212 empleados,
#           "Corpus cargado: 1 paquete(s), 7 obligaciones"

# 6. El expediente se verifica sin red y sin fiarse del emisor. ~1 s
./plazum verify expediente-demo.json contexto-demo.json
# ESPERADO: exit 0, ocho líneas "ok" y
#           "VERIFICADO. Recalculado desde cero sin red y sin confiar en el emisor."

# 7. La cobertura del núcleo.                                  ~40 s
go test ./nucleo/... -coverprofile=c.out >/dev/null && go tool cover -func=c.out | tail -1
# ESPERADO: total: (statements) 88.3%   (la puerta dura exige 85 %)

# 8. Las 26 puertas.                                           817 s MEDIDOS (13 min 37 s)
./comprobar.sh
# ESPERADO: exit 0, y al final:
#   COMPROBACION EN VERDE: 26 puertas leidas de los workflows,
#   26 ejecutadas aqui, mas formato, vet y build,
#   mas 3 herramientas de seguridad leidas de ci.yml

# 9. El corpus.                                                instantáneo
ls -d paquetes/*/ | wc -l
# ESPERADO: 20

# 10. Las pantallas, si quiere verlas.                          ~2 s
./plazum demo --serve    # (existirá tras la fase 7 opción C)
# ESPERADO: seis pantallas en localhost:8443 con el estado del demo cargado
```

**Tiempo total del guion: unos 17 minutos**, de los que 13 y medio son el paso 8. Quien solo tenga cinco minutos hace los pasos 1 a 6 y ya ha verificado lo principal.

---

## 10. Lo que este plan no arregla

Con horas no se compran estas cinco cosas, y conviene saberlo antes de empezar para no frustrarse.

1. **El historial es inmutable.** 609 commits en 18 días activos, 127 en un solo día, 182k líneas en 27 días. Reescribirlo sería peor y además detectable. La única jugada es la de la fase 3: contarlo tú primero con las puertas delante. Después de eso **sigue siendo lo primero que pregunta un entrevistador**, y ahí la respuesta no la da el repo, la das tú. Prepárala: qué decidiste, qué rechazaste del agente, qué puerta escribiste porque no te fiabas de lo que había salido. Esa conversación la ganas o la pierdes tú, no el README.

2. **Adopción de terceros.** Cero estrellas ajenas, cero issues externas, cero contribuidores fuera de los alias del propio autor: `git log --format='%an' | sort | uniq -c` da 599 de Marcos Mata, 4 de `obligo`, 4 de dependabot y 2 de `dutiq`. Eso es lo que separa el 9 del 10 y **no se compra con horas**: se compra publicando donde vivan los responsables de cumplimiento españoles y esperando. Meses, no horas.

3. **La estrechez del dominio.** El corpus es derecho español y europeo. Para una empresa española de GRC es el mejor argumento posible. Para un producto internacional en Madrid es un repo que el revisor no puede juzgar por el contenido, solo por la ingeniería. El `README.en.md` abre la puerta, no resuelve el problema.

4. **Autoría en solitario.** No hay ni una revisión de código ajena en 609 commits. Ninguna cantidad de puertas demuestra que sepas trabajar en equipo, y eso es justo lo que quiere ver un responsable de plataforma. Lo compensa en parte la calidad de los mensajes de commit, que sí se leen como comunicación de ingeniero, pero es un sustituto, no una prueba.

5. **La señal de vida.** Del 11 al 20 de septiembre hubo 9 días sin commits, y hoy uno solo. Si plazum va a ser la pieza principal, la cadencia tiene que ser regular durante semanas. Eso no está en las 21,75 horas.

**Sobre el compromiso de la nota.** Las 21,75 horas dejan el repo en 9 sobre la vara de las siete dimensiones. Lo que no garantizan es la fase 2: `release.yml` se ha ejecutado de verdad una sola vez, el 04-09-2026, y aquella vez falló el job de la imagen. Si vuelve a fallar, resolverlo puede costar 2 o 3 horas más. Es el único punto del plan con incertidumbre real de ejecución.

---

## 11. Registro de decisiones

### ADR-001: el repo se queda en español, con una página en inglés

**Contexto.** 75.187 líneas de producción, 24.129 de markdown y 609 mensajes de commit, todo en castellano. El dominio es derecho español y europeo: BOE, ENS, Ley 39/2015. Hoy un revisor que no hable español no puede evaluar nada.

**Alternativa descartada.** Traducir el repo entero a inglés. Semanas de trabajo, y el resultado habla de la Ley 39/2015 en inglés, que no le sirve a nadie.

**Consecuencia.** Se traduce solo `README.en.md`, menos de 600 palabras. El repo encaja bien en empresas españolas de GRC y legaltech, y queda en desventaja en producto internacional. Se asume a cambio de 2,5 horas en vez de semanas.

### ADR-002: plazum se presenta como backend y plataforma, no como AI

**Contexto.** `adaptadores/ia/` más `evals/` suman 1.649 líneas sobre 75.187, el 2,2 %. La regla del repo es que la IA vive en adaptadores y el núcleo no la conoce, con una puerta de CI que corre la suite con la IA apagada.

**Alternativa descartada.** Añadir un RAG sobre el corpus para engordar ese porcentaje. Entre 25 y 40 horas para que sea creíble, y debilita el único argumento que plazum tiene y nadie más: que el motor no puede inventarse una fecha porque es determinista y está verificado por AST.

**Consecuencia.** Cero horas de código nuevo de IA. Se cuenta la historia que ya existe, que es de gobierno de IA y acotación de superficie, y que es más difícil de enseñar que un RAG. Si el puesto es de AI, plazum va como segunda pieza, y hace falta otro repo para la primera.

### ADR-003: un solo repo, el corpus no se separa

**Contexto.** El corpus son 20 paquetes en `paquetes/`, viaja dentro del binario y de la imagen, y hay 808 casos dorados que lo corren contra el motor en cada `comprobar.sh`. Las licencias son distintas: código AGPL-3.0, datos Apache-2.0.

**Alternativa descartada.** Partir en `plazum` y `plazum-corpus`. Obliga a inventar versionado entre repos y descarga por red, que es la dependencia que el producto presume de no tener, y convierte 808 casos dorados en CI cruzado.

**Consecuencia.** Un repo. La diferencia de licencias se sigue resolviendo con la declaración por directorio y su linter, que ya existe y funciona.

### ADR-004: `v0.1.0-rc1` no se borra, se entierra

**Contexto.** El rc está publicado, es del 04-09-2026, está 128 commits por detrás y sus notas anuncian 33 paquetes cuando hoy son 20. Su firma está en Rekor, que es append-only.

**Alternativa descartada.** Borrar el tag y la release. Deja la firma de Rekor apuntando a algo que no existe, y borrar releases se ve.

**Consecuencia.** Se publica `v0.1.0`, que por no llevar sufijo activa `make_latest` y pasa a ser la release actual automáticamente. Al rc se le añade un aviso de obsolescencia escrito. Un rc viejo con su aviso es buena señal: demuestra que sabes qué publicaste y cuándo dejó de valer.

### ADR-005: no se despliega demo pública, se hace modo demo local

**Contexto.** Seis pantallas accesibles con axe-core en cero violaciones, un `serve` con sesiones, CSRF, OIDC y SCIM, y nadie lo ve. Presupuesto ajustado.

**Alternativa descartada.** Desplegar una demo pública. Entre 5 y 12 euros al mes, de 6 a 10 horas de montaje, y una superficie de ataque real con credenciales mantenida por una sola persona. El riesgo es asimétrico: funcionando suma poco más que un GIF, caída resta mucho en un repo cuya tesis es la fiabilidad.

**Consecuencia.** GIF y tres capturas para los 60 segundos, más `plazum demo --serve` en 2 o 3 horas, que levanta las seis pantallas en local con el estado del demo ya cargado. Coste mensual cero, superficie expuesta cero. Se revisa la decisión si algún día ya se paga un VPS para otra cosa.
