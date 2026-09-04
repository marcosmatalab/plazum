# syntax=docker/dockerfile:1
#
# La imagen de plazum: un binario estatico sobre scratch.
#
# ARRANQUE EN UN COMANDO, que es lo que esta imagen tiene que conseguir:
#
#     docker build -t plazum .
#     docker run --rm plazum
#
# Eso segundo instala la empresa de ejemplo, deriva sus obligaciones y ensena
# sus relojes corriendo. Sin configurar nada, sin red y sin montar nada.
#
# Lo demas, con el corpus y el expediente que ya vienen dentro:
#
#     docker run --rm plazum verify expediente-demo.json contexto-demo.json
#     docker run --rm plazum cobertura paquetes
#     docker run --rm -p 8443:8443 plazum serve --direccion 0.0.0.0:8443
#
# Ninguna lleva rutas raras porque el corpus y el expediente de ejemplo estan en
# el directorio de trabajo, que es donde cada orden los busca por defecto. Para
# usar el corpus propio se monta encima: -v /mi/corpus:/datos/paquetes.
#
# La ultima sirve por http sin TLS y avisa de ello por su salida de errores. Eso
# solo vale detras de un proxy que termine TLS, o en local. Lee docs/tls.md antes
# de dejarlo abierto. --direccion 0.0.0.0:8443 hace falta a proposito: por
# defecto plazum escucha en 127.0.0.1, que dentro de un contenedor es el propio
# contenedor y no lo alcanza nadie.
#
# CON EL SISTEMA DE FICHEROS EN SOLO LECTURA, que es lo primero que prueba quien
# despliega esto en serio. verify, explain y cobertura no escriben nada, asi que
# van tal cual:
#
#     docker run --rm --read-only plazum verify expediente-demo.json contexto-demo.json
#
# El demo SI escribe (instala una empresa de ejemplo), asi que necesita un sitio
# donde hacerlo. Comprobado, no supuesto:
#
#     docker run --rm --read-only --tmpfs /datos/salida:uid=65532,gid=65532 \
#         plazum demo --dir /datos/salida/demo
#
# POR QUE SCRATCH Y NO UNA DISTRIBUCION. El binario es Go puro con CGO_ENABLED=0,
# asi que no enlaza contra ninguna libc y no necesita nada del sistema. Lo que se
# gana no es tamano sino superficie: en la imagen no hay shell, no hay gestor de
# paquetes y no hay ni un binario mas que el nuestro, asi que la lista de CVEs
# que hereda es exactamente la de su propio codigo. Un escaner sobre esta imagen
# habla de plazum y de nada mas.
#
# LO QUE SI HAY QUE METER A MANO, porque scratch no lo trae y sin ello el
# producto miente o se rompe:
#
#   la base de zonas horarias   NO va aqui: viaja dentro del binario, ver
#                               cmd/plazum/zonas.go. Sin ella `plazum verify`
#                               respondia NO VERIFICA sobre un expediente
#                               correcto, o sea acusaba al emisor de un fallo
#                               del receptor
#   las raices de CA            `plazum serve` con OIDC sale a la red por https.
#                               Sin este fichero la autenticacion falla con un
#                               error de certificado que no dice que le falta
#   /etc/passwd y /etc/group    el proceso corre como 65532 y no como root. Sin
#                               estos dos ficheros el usuario existe igual (el
#                               kernel solo mira numeros) pero cualquier
#                               herramienta que resuelva el nombre se queja
#   /tmp                        scratch no lo trae, y un directorio temporal que
#                               no existe se nota tarde y mal
#
# REPRODUCIBILIDAD. La imagen base va fijada POR DIGEST y no por etiqueta: una
# etiqueta la reescribe quien la publica, y entonces "la misma linea" construye
# otra cosa cada mes sin que el diff lo ensene. El binario se compila con
# -trimpath (fuera las rutas de la maquina), -buildid= (fuera el identificador
# que cambia por build) y CGO_ENABLED=0 (fuera el enlazador del sistema). Dos
# construcciones del mismo commit dan el MISMO sha256 del binario, y eso lo
# comprueba el trabajo `imagen` de .github/workflows/etapa2-distribucion.yml
# construyendo dos veces con --no-cache y comparando.
#
# Como refrescar el digest de la imagen base (revision manual, a mano, igual que
# las dependencias sin semver de DEPENDENCIAS.md):
#
#     docker buildx imagetools inspect golang:1.24-alpine
#
# Fijado el 26-08-2026 sobre golang:1.24-alpine.

#
# --platform=$BUILDPLATFORM: la etapa de construccion corre SIEMPRE en la
# arquitectura de la maquina que construye, y el compilador de Go cruza a la de
# destino. Sin eso, `--platform linux/arm64` desde un runner amd64 intentaria
# EJECUTAR el compilador en arm64 emulado con QEMU: diez veces mas lento y una
# dependencia mas (binfmt) que hay que instalar y que a veces no esta.
FROM --platform=$BUILDPLATFORM golang:1.24-alpine@sha256:8bee1901f1e530bfb4a7850aa7a479d17ae3a18beb6e09064ed54cfd245b7191 AS construccion

# TARGETARCH lo pone BuildKit solo. Con el, `docker buildx build --platform
# linux/arm64` cruza sin tocar este fichero.
ARG TARGETARCH=amd64

WORKDIR /src
COPY . .

# EL ANCLA DEL CORPUS, TAMBIEN AQUI, y hace falta decir por que la imagen la
# necesita si ya lleva el corpus dentro (`COPY paquetes /datos/paquetes` mas
# abajo).
#
# La lleva porque «tener el corpus» y «poder decir que ese es el corpus» son dos
# cosas distintas, y la imagen es justo donde se separan: quien monta el suyo
# encima con `-v /mi/corpus:/datos/paquetes` sustituye el corpus publicado sin
# que nada lo diga. Con el ancla dentro, `docker run plazum corpus` contesta si
# lo que hay montado es lo que se publico o es otra cosa. Sin ella contestaria
# «no lo se» siempre, que en un contenedor es la respuesta permanente.
#
# SE CONSTRUYE DOS VECES A PROPOSITO. La primera pasada da un plazum sin ancla,
# que es todo lo que hace falta para RESUMIR el corpus; con esa huella se
# construye el definitivo. La alternativa era calcular la huella aqui con
# `find | sha256sum`, o sea una segunda implementacion del algoritmo en shell:
# dos implementaciones que se separan y un binario que un dia rechaza su propio
# corpus sin que nadie sepa por que.
#
# -mod=readonly: si el codigo pide un modulo que go.mod no declara, la
# construccion falla en vez de resolverlo sola y meter en la imagen una
# dependencia que nadie ha revisado.
#
# La huella es funcion del corpus y de nada mas (ni fecha, ni maquina, ni
# arquitectura), asi que meterla con -X no rompe la reproducibilidad que
# comprueban -trimpath y -buildid=: dos construcciones del mismo arbol siguen
# dando el mismo sha256.
RUN CGO_ENABLED=0 GOOS=linux go build -mod=readonly -o /tmp/plazum-sin-ancla ./cmd/plazum \
 && ANCLA="$(/tmp/plazum-sin-ancla corpus --huella paquetes)" \
 && echo "ancla del corpus de esta imagen: ${ANCLA}" \
 && CGO_ENABLED=0 GOOS=linux GOARCH="${TARGETARCH}" \
    go build -mod=readonly -trimpath \
    -ldflags="-s -w -buildid= -X main.anclaCorpus=${ANCLA}" -o /salida/plazum ./cmd/plazum \
 && rm -f /tmp/plazum-sin-ancla

# El esqueleto de sistema de la imagen final se prepara aqui, donde SI hay
# shell. En scratch no se puede ejecutar nada, asi que todo lo que haya que
# crear se crea en esta etapa y se copia hecho.
RUN mkdir -p /esqueleto/etc /esqueleto/tmp /esqueleto/datos \
 && printf 'plazum:x:65532:65532:plazum:/datos:/sbin/nologin\n' > /esqueleto/etc/passwd \
 && printf 'plazum:x:65532:\n'                                 > /esqueleto/etc/group \
 && chmod 1777 /esqueleto/tmp \
 && chown -R 65532:65532 /esqueleto/datos

FROM scratch

# Las etiquetas son ESTATICAS a proposito. Una etiqueta con la fecha o el commit
# dentro cambia la configuracion de la imagen en cada construccion, y entonces
# dos construcciones del mismo codigo dan digests distintos y la comprobacion de
# reproducibilidad no puede distinguir eso de un cambio de verdad. Lo variable
# (revision, fecha) lo pone quien publique, con --label.
LABEL org.opencontainers.image.title="plazum" \
      org.opencontainers.image.description="GRC de continuidad de cumplimiento: motor determinista de obligaciones con reloj legal y expediente verificable offline" \
      org.opencontainers.image.licenses="AGPL-3.0-only" \
      org.opencontainers.image.source="https://github.com/marcosmatalab/plazum" \
      org.opencontainers.image.vendor="plazum"

COPY --from=construccion /esqueleto/etc/passwd /esqueleto/etc/group /etc/
COPY --from=construccion --chmod=1777 /esqueleto/tmp /tmp
COPY --from=construccion --chown=65532:65532 /esqueleto/datos /datos
COPY --from=construccion /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=construccion /salida/plazum /plazum

# El corpus y el expediente de ejemplo viajan dentro. Es la diferencia entre una
# imagen que arranca y una imagen que arranca y ENSENA algo: sin el corpus,
# `plazum serve` se niega a levantarse y `plazum verify` no tiene nada que
# recalcular, y el que la ha bajado tiene que buscar en un repositorio que
# todavia no conoce.
#
# Y van DENTRO del directorio de trabajo, no en la raiz, para que cada orden los
# encuentre donde los busca por defecto: `serve` sin --corpus, `cobertura
# paquetes`, `verify expediente-demo.json`. Una ruta absoluta rara en cada
# ejemplo es una cosa mas que el que acaba de bajar la imagen tiene que copiar
# bien.
COPY --chown=65532:65532 paquetes /datos/paquetes
COPY --chown=65532:65532 expediente-demo.json contexto-demo.json /datos/

# Sin privilegios y por numero: un contenedor que corre como root es root del
# host si algo se escapa, y ningun comando de plazum necesita mas que leer su
# corpus y escribir su directorio de datos.
USER 65532:65532
WORKDIR /datos

# El demo por defecto: quien escribe `docker run --rm plazum` a secas casi
# siempre lo acaba de bajar, y una ayuda de seis ordenes sin punto de entrada es
# lo mismo que ninguna. ENTRYPOINT fijo y CMD sustituible, asi que
# `docker run --rm plazum verify ...` cambia la orden sin cambiar el binario.
ENTRYPOINT ["/plazum"]
CMD ["demo"]
