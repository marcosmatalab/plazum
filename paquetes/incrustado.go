// Package paquetes lleva el corpus publicado dentro del binario.
//
// Es el corpus por defecto de plazum: el que usa cualquier orden cuando no se
// le pasa --corpus y no hay un directorio paquetes/ donde se ejecuta. Un corpus
// en disco, sea el del repositorio, el de la imagen o uno instalado con
// `plazum corpus --instalar`, manda siempre sobre este. El porque y lo que
// cuesta, en docs/decisiones.md D-30.
//
// Va aqui y no bajo cmd/ porque go:embed no puede salir del directorio de su
// paquete. Este fichero es el unico .go de paquetes/ que no es un test, y como
// todo lo que vive en este directorio se distribuye bajo Apache-2.0.
package paquetes

import "embed"

// Ficheros es el arbol de paquetes/ tal y como lo empaqueta la release: todo
// menos los .go. La lista de patrones es explicita a proposito; que cubra
// exactamente ese arbol lo vigila TestElCorpusIncrustadoEsElArbolPublicado,
// que compara la huella de esto con la del directorio.
//
//go:embed CORPUS.md marcos-v1.json */paquete.json */*.md */pruebas
var Ficheros embed.FS
