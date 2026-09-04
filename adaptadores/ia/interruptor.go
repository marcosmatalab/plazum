package ia

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

// Variable es el interruptor general de la IA. Puesta, ningun adaptador de
// modelo se construye y ninguna peticion sale de la maquina.
//
// POR QUE ES UNA VARIABLE DE ENTORNO Y NO UNA OPCION. Tiene que poder apagarse
// SIN TOCAR LA CONFIGURACION del producto, porque su primer usuario no es el
// operador: es la puerta de CI que corre la suite entera con la IA apagada, y
// es cualquiera que clone el repositorio y quiera comprobar en dos minutos que
// el nucleo es determinista. Una opcion en un fichero exige saber donde esta el
// fichero; esto es un prefijo delante del comando.
const Variable = "PLAZUM_SIN_IA"

var (
	// ErrIADesactivada es lo que devuelve cualquier constructor de adaptador de
	// modelo cuando el interruptor esta puesto. NO es un fallo: es el modo sin
	// IA funcionando.
	ErrIADesactivada = errors.New("ia: la IA esta desactivada por " + Variable)
	// ErrInterruptorIlegible es la TERCERA FORMA DE LA NADA aplicada al propio
	// interruptor.
	ErrInterruptorIlegible = errors.New("ia: el valor de " + Variable + " no se entiende")
)

// Apagada dice si el interruptor esta puesto.
//
// LAS TRES FORMAS, Y POR QUE AQUI DUELE MAS QUE EN NINGUN SITIO. Lo obvio es
// `os.Getenv(Variable) != ""`, y esta mal de la forma mas cara posible:
//
//	sin poner        la IA funciona. Es el defecto del producto.
//	PLAZUM_SIN_IA=1  la IA se apaga. Bien.
//	PLAZUM_SIN_IA=0  con `!= ""` LA IA SE APAGA IGUAL, cuando quien lo escribio
//	                 estaba diciendo lo contrario.
//	PLAZUM_SIN_IA=si con `!= ""` se apaga por casualidad, no porque se entienda.
//	PLAZUM_SIN_IA=quiza  presente y no interpretable. Con `!= ""` se apaga; con
//	                 un `strconv.ParseBool` cuyo error se tira, se ENCIENDE.
//
// Las dos ultimas son el mismo fallo del invariante 8 mirando a lados
// opuestos, y la peligrosa es la que enciende: un operador que escribio algo
// raro creyendo que apagaba la IA, y la IA hablando con la red. Asi que un
// valor presente que no se entiende es ERROR, nunca un defecto: se para y se
// dice, que es lo unico que no puede sorprender a nadie.
//
// Devuelve (apagada, error). Un error aqui NO se puede tratar como "encendida":
// quien llama tiene que propagarlo.
func Apagada() (bool, error) {
	crudo, presente := os.LookupEnv(Variable)
	if !presente {
		return false, nil
	}
	switch strings.ToLower(strings.TrimSpace(crudo)) {
	case "":
		// Presente y VACIA. Es la nada de la forma "vacio-presente", y aqui
		// significa lo mismo que ausente: quien exporta una variable vacia no
		// esta pidiendo apagar nada. Se distingue del caso siguiente a
		// proposito.
		return false, nil
	case "1", "true", "si", "sí", "yes", "on":
		return true, nil
	case "0", "false", "no", "off":
		return false, nil
	}
	return false, fmt.Errorf("%w: se ha puesto un valor que no es ni afirmativo ni "+
		"negativo. Arreglo: %s=1 para apagar la IA, %s=0 o sin poner para dejarla. "+
		"No se toma un defecto: un valor presente que no se entiende es un dato que hay "+
		"y no se entiende, y darlo por bueno seria decidir por ti si tus datos salen de "+
		"la maquina.", ErrInterruptorIlegible, Variable, Variable)
}

// ExigeEncendida es el guardian que todo constructor de adaptador de modelo
// llama en su primera linea.
//
// Va en una funcion suya y no copiado en cada adaptador PARA QUE HAYA UN SOLO
// SITIO: un interruptor que cada adaptador interpreta a su manera es un
// interruptor que apaga tres de los cuatro.
func ExigeEncendida() error {
	apagada, err := Apagada()
	if err != nil {
		return err
	}
	if apagada {
		return fmt.Errorf("%w. Esto no es un fallo: es el modo sin IA. Con la IA apagada "+
			"el producto hace todo lo que promete, y esta comprobacion es lo que lo "+
			"convierte en un hecho y no en un eslogan", ErrIADesactivada)
	}
	return nil
}
