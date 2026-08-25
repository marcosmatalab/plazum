# Raíces de confianza de las TSAs por defecto

Certificados públicos y redistribuibles de las dos TSAs gratuitas que trae
`tsa.PorDefecto()`. Van embebidos en el binario con `go:embed` porque **un
verificador que no trae raíces no es usable offline, y offline es su razón de
existir**: el auditor abre el expediente en su máquina, sin red, y tiene que
poder comprobar el sello.

Son un juego POR DEFECTO, no una imposición: el receptor puede sustituirlas por
las suyas con `raices_tsa` en su fichero de contexto, y entonces solo valen esas.

| Fichero | Sujeto | Emisor | Caduca |
|---|---|---|---|
| `freetsa.pem` | `www.freetsa.org` (Free TSA) | ella misma, autofirmada | 2041-03-07 |
| `certum.pem` | `Certum Trusted Network CA 2` (Unizeto Technologies S.A.) | `Certum Trusted Network CA` | 2029-09-17 |

Obtenidos el 25-08-2026 de la propia cadena que devuelve cada TSA en su
respuesta RFC 3161, no de una descarga aparte: es la forma de que el
certificado que se empotra sea exactamente el que firma.

## Cuándo hay que tocarlos

- **Antes de 2029-09-17**, o el sello de Certum dejará de verificar para sellos
  nuevos. Los sellos ya emitidos siguen valiendo, porque la verificación
  comprueba el certificado en el instante del sello y no hoy.
- Si una TSA rota su cadena. Se detecta porque los sellos nuevos dejan de
  verificar mientras la TSA responde 200.

Para regenerarlos, la sonda está en el historial de git: pedir un sello real,
sacar el último certificado de la cadena del token y guardarlo aquí.
