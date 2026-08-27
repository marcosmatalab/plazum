// Package perfiles trae los perfiles de arranque empotrados en el binario.
//
// UN PERFIL NO ES UNA RESPUESTA. Es un conjunto de hechos SUPUESTOS a partir de
// tres datos gruesos (pais, sector, empleados), para que alguien que acaba de
// descargar el binario vea algo en diez segundos sin configurar nada. Cada hecho
// lleva escrito POR QUE se supone, y todo lo que se derive de un perfil sale
// marcado como supuesto de punta a punta.
//
// POR QUE SON DATOS Y NO CODIGO. Un mapa en Go que dijera "sector salud implica
// tal norma" rompe el invariante 2 y el build (extensibilidad_test.go). Y por
// debajo del invariante hay una razon mejor: un supuesto sobre a quien alcanza
// una norma es contenido juridico, se equivoca, y tiene que poder corregirlo
// quien lo sufre sin recompilar nada.
//
// LO QUE UN PERFIL NO HACE, y es la mitad del diseno: no afirma hechos que solo
// la organizacion puede saber. Ser operador de servicios esenciales, entidad
// financiera o proveedor de servicios de pago son DESIGNACIONES formales, no
// consecuencias de un sector. Un perfil que las supusiera le ensenaria a una
// consultora de veinte personas los plazos de notificacion de un banco.
package perfiles

import "embed"

// Ficheros son los perfiles, empotrados para que `plazum calendario --pais=...`
// funcione sin red, sin instalar nada y sin directorio de trabajo.
//
//go:embed *.json
var Ficheros embed.FS
