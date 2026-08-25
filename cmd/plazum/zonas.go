package main

// La base de zonas horarias viaja DENTRO del binario.
//
// POR QUE. Un expediente declara la zona de su calendario legal ("Europe/Madrid"
// en el demo) y el verificador la resuelve con time.LoadLocation, que en Linux y
// en macOS lee /usr/share/zoneinfo del sistema. Una imagen minima no trae ese
// directorio: en scratch y en distroless-static NO existe.
//
// Lo que pasaba sin esta linea, comprobado en una imagen scratch de verdad y no
// supuesto:
//
//	DISCREPA   reloj de rgpd.art33.notificacion
//	           declarado:   declaracion valida
//	           recalculado: zona horaria "Europe/Madrid": unknown time zone Europe/Madrid
//	NO VERIFICA: 6 discrepancia(s).
//
// Y ese es el peor fallo posible en este producto. `plazum verify` no reventaba:
// respondia NO VERIFICA, o sea acusaba al emisor de haber falseado su expediente
// cuando el que estaba roto era el receptor. Un verificador que dice "mientes"
// porque a su contenedor le falta un fichero destruye exactamente la confianza
// que el expediente existe para crear.
//
// El coste: unos 450 KB de tabla en el binario. time/tzdata es biblioteca
// ESTANDAR, asi que no toca DEPENDENCIAS.md, y su tabla solo se usa cuando el
// sistema no tiene la suya, asi que en una maquina normal no cambia nada: sigue
// mandando el zoneinfo del sistema, que es el que el operador actualiza.
//
// La puerta que vigila esta linea es doble, y hace falta que sea doble porque
// ninguna de las dos mitades vale sola:
//
//	TestElCLITraeSuPropiaBaseDeZonasHorarias   lee el AST de este paquete y
//	  exige el import. Es estatico a proposito: en una maquina que SI tiene
//	  zoneinfo (la de desarrollo, el runner de CI) una comprobacion en
//	  ejecucion pasa igual de verde con la linea puesta que quitada, asi que
//	  no vigilaria nada.
//	el trabajo `imagen` de .github/workflows/etapa2-distribucion.yml   ejecuta
//	  `plazum verify` DENTRO de la imagen scratch y exige VERIFICADO. Esa es la
//	  que comprueba el efecto y no la intencion.
import _ "time/tzdata"
