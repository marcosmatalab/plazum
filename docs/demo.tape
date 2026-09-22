# El GIF de la portada, regenerable y determinista.
#
# ESTADO, Y VA PRIMERO PORQUE UN GUION SIN SALIDA ES UNA PROMESA SUELTA: el
# GIF NO esta generado. El motivo es de maquina y no de decision, y esta
# contado con su cardinal en docs/pendientes.md (P2 55): vhs necesita ttyd,
# que no existe en Windows, y su contenedor oficial contesta `denied` desde
# esta maquina. En cualquier Linux con Docker, el comando de abajo lo cierra.
#
# POR QUE UN .tape VERSIONADO Y NO UN GIF GRABADO A MANO. Porque un GIF grabado
# a mano es un numero publicado que nadie puede recomputar: ensena una salida y
# no hay forma de saber si es la de hoy. Con el guion al lado, cualquiera vuelve
# a generarlo y compara.
#
# COMO SE REGENERA, y no hace falta instalar nada mas que Docker:
#
#   GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -trimpath \
#       -o docs/lanzamiento/plazum-linux ./cmd/plazum
#   docker run --rm -v "$PWD":/vhs ghcr.io/charmbracelet/vhs /vhs/docs/demo.tape
#
# vhs necesita ttyd, que no existe en Windows; por eso va por contenedor y no
# por instalacion local. El binario se construye FUERA del contenedor para que
# lo que se grabe sea el mismo binario que mide la puerta del presupuesto.

Output docs/demo.gif

Set Shell bash
Set FontSize 16
Set Width 1200
Set Height 700
Set Padding 20
Set Theme "Catppuccin Mocha"
Set TypingSpeed 30ms

Hide
Type "cd /vhs && export PATH=/vhs/docs/lanzamiento:$PATH && clear" Enter
Sleep 500ms
Show

Type "plazum-linux calendario --pais=ES --sector=servicios-digitales --empleados=200"
Sleep 600ms
Enter
Sleep 6s

Type "clear" Enter
Sleep 300ms

Type "plazum-linux verify expediente-demo.json contexto-demo.json"
Sleep 600ms
Enter
Sleep 5s

Sleep 1s
