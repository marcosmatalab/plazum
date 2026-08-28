package scim

import (
	"fmt"
	"time"

	"github.com/marcosmatalab/plazum/puertos"
)

// SilencioSospechoso es cuanto puede callar el IdP antes de que se avise.
//
// Los ciclos de aprovisionamiento son de 40 minutos en Entra ID y de una hora
// en Okta. Veinticinco horas deja pasar un dia entero de holgura y sigue
// avisando al segundo dia, que es cuando un aprovisionamiento parado empieza a
// significar que alguien despedido conserva el acceso.
const SilencioSospechoso = 25 * time.Hour

// Comprobaciones responde, en el vocabulario de `plazum doctor`, a la pregunta
// del comprador: ¿esta funcionando el SCIM o solo lo espero?
//
// Devuelve []puertos.Comprobacion y no implementa puertos.Diagnostico a
// proposito: el doctor completo lo compone otro frente, y este trozo tiene que
// poder enchufarse ahi sin que dos implementaciones se peleen por el puerto.
//
// Cada comprobacion que no esta correcta dice COMO SE ARREGLA, que es la
// exigencia de la suite de contrato de ese puerto y la razon de que doctor
// exista: un problema sin arreglo escrito le pasa el trabajo al operador.
func (s *Servidor) Comprobaciones(ahora time.Time) []puertos.Comprobacion {
	act := s.Actividad()
	var out []puertos.Comprobacion

	switch {
	case act.UltimaPeticion.IsZero():
		out = append(out, puertos.Comprobacion{
			Nombre:  "scim-conexion",
			Estado:  puertos.Aviso,
			Detalle: "el aprovisionamiento SCIM esta configurado y todavia no ha llegado ninguna peticion",
			Arreglo: "en el IdP, pulsa Probar conexion (Entra ID: Aplicaciones empresariales, " +
				"Aprovisionamiento; Okta: Provisioning, Integration) con la URL " +
				s.base + " y el token de aprovisionamiento de esta instancia. " +
				"Si la prueba falla con 401, el token pegado no es el de aqui",
		})
	case ahora.Sub(act.UltimaCorrecta) > SilencioSospechoso:
		ultima := "nunca"
		if !act.UltimaCorrecta.IsZero() {
			ultima = act.UltimaCorrecta.Format(time.RFC3339)
		}
		out = append(out, puertos.Comprobacion{
			Nombre: "scim-conexion",
			Estado: puertos.Roto,
			Detalle: fmt.Sprintf("el IdP no completa una peticion correcta desde %s "+
				"(%d rechazos, el ultimo: %s)", ultima, act.Rechazos, act.UltimoRechazo),
			Arreglo: "mira el registro de aprovisionamiento del IdP. Un aprovisionamiento " +
				"parado significa que quien salio de la empresa conserva el acceso aqui, asi " +
				"que no es un aviso, es una baja sin ejecutar",
		})
	default:
		out = append(out, puertos.Comprobacion{
			Nombre: "scim-conexion",
			Estado: puertos.Correcto,
			Detalle: fmt.Sprintf("ultima peticion correcta del IdP: %s (%d peticiones, %d rechazos)",
				act.UltimaCorrecta.Format(time.RFC3339), act.Peticiones, act.Rechazos),
		})
	}

	if act.RechazosDeCredencial > 0 {
		out = append(out, puertos.Comprobacion{
			Nombre: "scim-credencial",
			Estado: puertos.Aviso,
			Detalle: fmt.Sprintf("%d peticiones rechazadas por credencial invalida",
				act.RechazosDeCredencial),
			Arreglo: "si son unas pocas y el aprovisionamiento funciona, es el token viejo de " +
				"una configuracion anterior del IdP. Si son muchas y ninguna peticion pasa, el " +
				"token pegado en el IdP no es el de esta instancia: generalo otra vez y " +
				"pegalo entero, sin la palabra Bearer delante",
		})
	}

	// La jerarquia: es lo que sostiene el escalado, y su fallo es silencioso.
	j, err := s.dir.Jerarquia()
	if err != nil {
		out = append(out, puertos.Comprobacion{
			Nombre:  "scim-jerarquia",
			Estado:  puertos.Roto,
			Detalle: err.Error(),
			Arreglo: "corrige el atributo `manager` en el IdP para uno de los implicados, o " +
				"declara su jefe a mano en Personas. Mientras haya ciclo, el escalado de una " +
				"obligacion vencida no avisa a nadie",
		})
		return out
	}
	sin := j.SinManager()
	total := len(j.Relaciones()) + len(sin)
	switch {
	case total == 0:
		out = append(out, puertos.Comprobacion{
			Nombre:  "scim-jerarquia",
			Estado:  puertos.Aviso,
			Detalle: "no hay ningun usuario activo en el directorio",
			Arreglo: "asigna la aplicacion a usuarios en el IdP y espera al primer ciclo de " +
				"aprovisionamiento",
		})
	case len(sin) == total:
		out = append(out, puertos.Comprobacion{
			Nombre: "scim-jerarquia",
			Estado: puertos.Aviso,
			Detalle: fmt.Sprintf("ninguno de los %d usuarios trae el atributo `manager`: el "+
				"escalado no tiene a donde subir", total),
			Arreglo: "publica `manager` en el mapeo de atributos del aprovisionamiento del " +
				"IdP (extension enterprise), o declara la jerarquia a mano en Personas. Las " +
				"dos vias valen y conviven: lo declarado a mano se marca como tal",
		})
	case len(sin) > 0:
		out = append(out, puertos.Comprobacion{
			Nombre:  "scim-jerarquia",
			Estado:  puertos.Aviso,
			Detalle: fmt.Sprintf("%d de %d usuarios activos no tienen jefe conocido", len(sin), total),
			Arreglo: "para esos, el escalado de una obligacion vencida no sube a nadie. " +
				"Declara su jefe a mano en Personas, o revisa por que el IdP no publica su " +
				"`manager`",
		})
	default:
		out = append(out, puertos.Comprobacion{
			Nombre:  "scim-jerarquia",
			Estado:  puertos.Correcto,
			Detalle: fmt.Sprintf("%d usuarios activos, todos con jefe conocido", total),
		})
	}

	if n := len(j.Conflictos()); n > 0 {
		out = append(out, puertos.Comprobacion{
			Nombre: "scim-jerarquia-conflictos",
			Estado: puertos.Aviso,
			Detalle: fmt.Sprintf("%d usuarios tienen un jefe declarado a mano distinto del que "+
				"publica el IdP; manda el del IdP", n),
			Arreglo: "revisa la lista en Personas. Manda el IdP porque es la fuente " +
				"autoritativa mientras hable; si el bueno es el declarado a mano, corrige el " +
				"IdP o retira el mapeo manual para que deje de confundir",
		})
	}
	if n := len(j.Rotas()); n > 0 {
		out = append(out, puertos.Comprobacion{
			Nombre:  "scim-jerarquia-rotas",
			Estado:  puertos.Aviso,
			Detalle: fmt.Sprintf("%d relaciones apuntan a un jefe que ya no esta en el directorio", n),
			Arreglo: "esos usuarios se quedaron sin cadena de escalado cuando su jefe salio " +
				"de la empresa. Actualiza su `manager` en el IdP o declaralo a mano",
		})
	}
	return out
}
