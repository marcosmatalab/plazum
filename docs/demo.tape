# El GIF de la portada, regenerable y determinista.
#
# SE REGENERA CON docs/lanzamiento/gif.sh, que construye el binario de Linux,
# levanta la imagen con ttyd, ffmpeg y chromium, graba esto y codifica. Ahi
# estan escritas las tres razones por las que no basta `vhs docs/demo.tape`, y
# las tres dan un fallo silencioso.
#
# POR QUE ESTE FICHERO PIDE `Output frames/` Y NO UN .gif: porque `vhs` en esta
# version CAPTURA bien y NO CODIFICA. Sale con codigo 0, dice que esta creando
# el GIF, y no escribe nada ni llega a invocar a ffmpeg. La codificacion la hace
# el script, que es lo que `vhs` haria.
#
# POR QUE UN .tape VERSIONADO Y NO UN GIF GRABADO A MANO. Porque un GIF grabado
# a mano es una salida que nadie puede recomputar: ensena algo y no hay forma de
# saber si es de hoy. Con el guion al lado, cualquiera vuelve a generarlo y
# compara.

Output frames/

Set Shell bash
Set FontSize 16
Set Width 1400
Set Height 800
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
