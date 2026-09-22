// El VOCABULARIO DE ERRORES del formato, por identidad y no por texto.
//
// Estan juntos y aparte a proposito: un centinela nuevo se anade aqui, donde se
// ve la lista entera, y no al lado de la comprobacion que lo lanza. Un test que
// buscara "clase" con strings.Contains lo encontraria dentro de "clase_e2e" y
// daria verde con el fallo delante, y eso ya paso aqui.

package corpus

import (
	"errors"
)

// Los errores del formato que se comprueban por identidad, no por el texto del
// mensaje. Un test que busque "clase" con strings.Contains lo encuentra dentro
// de "clase_e2e" y da verde con el fallo delante: eso ya paso aqui.
var (
	// ErrTextoRedistribuido: un campo de PROSA pasa del limite en un paquete
	// que no puede redistribuir texto de un tercero. Es la frontera legal.
	ErrTextoRedistribuido = errors.New("texto de un tercero por encima del limite de la clase")
	// ErrCitaDesbordada: un campo de REFERENCIA o de DERIVACION pasa de su
	// techo. No es necesariamente texto normativo, pero ya no es un localizador.
	ErrCitaDesbordada = errors.New("campo de referencia por encima del limite de la clase")
	// ErrVigenciaIlegible: una fecha de vigencia que no se puede leer. Viene de
	// un fichero de datos de un tercero, asi que no es una rareza teorica.
	ErrVigenciaIlegible = errors.New("fecha de vigencia ilegible")
	// ErrVigenciaInvertida: desde posterior a hasta. Una obligacion asi no esta
	// vigente NUNCA, y casi siempre es un error de tecleo, no una derogacion.
	ErrVigenciaInvertida = errors.New("vigencia con desde posterior a hasta")
	// ErrVigenciaSinDesde: no hay fecha de inicio y no hay de donde heredarla.
	ErrVigenciaSinDesde = errors.New("vigencia sin fecha de inicio")
	// ErrLecturaVigenciaSinCita: una lectura divergente de la vigencia sin la
	// cita de donde sale. Sin cita no es una lectura, es una opinion, y el
	// producto entero se apoya en que cada fecha se pueda seguir hasta su
	// fuente.
	ErrLecturaVigenciaSinCita = errors.New("lectura divergente de vigencia sin cita")
	// ErrLecturaVigenciaVacia: una lectura que no mueve ni el desde ni el
	// hasta. No dice nada y ocupa el sitio de la que si diria algo.
	ErrLecturaVigenciaVacia = errors.New("lectura divergente de vigencia sin fecha")
	// ErrLecturaVigenciaQueNoDiverge: una lectura identica a la declarada. Se
	// leeria como un desacuerdo donde no lo hay, que es peor que no decir nada.
	ErrLecturaVigenciaQueNoDiverge = errors.New("lectura divergente de vigencia que no diverge")
	// ErrObligacionSinID: una obligacion sin identificador no se puede citar,
	// ni referenciar desde una pregunta, ni seguir en el expediente.
	ErrObligacionSinID = errors.New("obligacion sin id")
	// ErrSinLicenciaFuente: el paquete no declara de que regimen sale su
	// contenido. Sin eso no se sabe si se puede redistribuir ni a quien hay
	// que atribuirlo, y las dos cosas son obligaciones, no metadatos.
	ErrSinLicenciaFuente = errors.New("paquete sin licencia_fuente")
	// ErrLicenciaFuenteDesconocida: un regimen que no esta en el vocabulario.
	ErrLicenciaFuenteDesconocida = errors.New("licencia_fuente fuera del vocabulario")
	// ErrLicenciaProhibida: un regimen de la lista negra. No es "todavia no".
	ErrLicenciaProhibida = errors.New("licencia_fuente prohibida en este proyecto")
	// ErrLicenciaFuenteIncoherente: el regimen declarado no es de los que
	// admite la clase. Uno de los dos campos miente y no se sabe cual.
	ErrLicenciaFuenteIncoherente = errors.New("licencia_fuente incoherente con la clase")
	// ErrSinAtribucion: el paquete no trae el aviso que hay que ensenar. Un
	// LICENCIAS.md en el repositorio no cumple una obligacion de atribucion
	// hacia quien USA el producto: el aviso tiene que viajar con el paquete.
	ErrSinAtribucion = errors.New("paquete sin atribucion")
)
