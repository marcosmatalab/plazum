package documentos

// LA CONFIRMACION DE UN CAMPO DE LA FICHA (pieza 7).
//
// # Que se afirma al pulsar este boton, y por que hace falta que lo pulse alguien
//
// `adaptadores/metadatos` propone la ficha de un documento —fecha, alcance,
// firmante, caducidad— leyendo su texto con patrones fijos. Eso es una
// propuesta, no un dato: una fecha de caducidad sacada de un PDF puede ser la
// del documento, la de un anexo o la del pie de una plantilla, y las tres se
// parecen. Aceptarla es una AFIRMACION DE UNA PERSONA sobre su propia
// documentacion, y por eso lo aceptado guarda quien y cuando.
//
// Es el invariante 13 en su forma de entrada: lo que sale de leer un documento
// son hechos sobre el documento; quien decide que significan es alguien que
// responde por ello.
//
// # LO QUE ESTE HANDLER NO HACE, Y ES DELIBERADO
//
// No admite un valor escrito a mano. Lo unico que se puede aceptar es una
// propuesta que el almacen tiene, identificada por su campo, su valor y la
// huella de su documento; si no casa con ninguna, se rechaza. La alternativa
// —dejar escribir el valor en el formulario— convertiria esta ruta en un
// «escribe lo que quieras en el expediente» sin cita que lo sostenga, que es
// justo lo contrario de para lo que existe.
//
// # EL EMPAREJAMIENTO, DICHO (invariante 7)
//
// La propuesta casa por CAMPO + VALOR + HUELLA DEL DOCUMENTO, y ninguno de los
// tres es una posicion. La huella es el sha256 del fichero, o sea contenido: dos
// documentos distintos no pueden compartirla, y reordenar la lista de propuestas
// no mueve ningun emparejamiento. Un indice en el formulario habria sido mas
// corto de escribir y habria aceptado la propuesta equivocada en cuanto la lista
// cambiara entre pintar la pagina y pulsar el boton.

import (
	"net/http"
	"strings"
)

// Los campos del formulario de aceptar.
const (
	CampoDelCampo  = "campo"
	CampoDelValor  = "valor"
	CampoDeHuella  = "huella"
	MaxLargoAcepta = 400
)

// aceptar confirma un campo propuesto.
func (s *Superficie) aceptar(w http.ResponseWriter, r *http.Request) {
	// 1. EL AUTOR. Sin sesion no se pinta un aviso: 401 con la pantalla de sin
	//    sesion, igual que en subir. Y aqui pesa el doble, porque lo que se va a
	//    guardar lleva SU NOMBRE dentro.
	quien := s.quien(r)
	if quien == "" {
		v, _ := s.vista(r)
		v.SinSesion = true
		s.pintar(w, r, v, http.StatusUnauthorized)
		return
	}
	// 2. EL ALMACEN. Llegar con nil es imposible por construccion y se comprueba
	//    igual, por lo mismo que en subir.
	if s.o.Almacen == nil {
		s.conAvisoDe(w, r, "documentos.subir.sin_almacen")
		return
	}
	// EL CUERPO SE ACOTA. Es un formulario de tres campos cortos: lo que llegue
	// por encima de eso no es un formulario.
	r.Body = http.MaxBytesReader(w, r.Body, 8<<10)
	if err := r.ParseForm(); err != nil {
		s.fallo(err)
		s.conAvisoDe(w, r, "documentos.aceptar.no_se_lee")
		return
	}

	// 3. LAS TRES FORMAS DE LA NADA EN LOS CAMPOS OBLIGATORIOS (invariante 8).
	//    Ausente y presente-en-blanco se tratan igual aqui —los dos son «no ha
	//    llegado»— y se distinguen de la tercera, que es la de abajo: presente,
	//    con contenido, y que no casa con ninguna propuesta. Esa NO es una
	//    ausencia y no se puede tratar como tal.
	campo := recortar(r.PostFormValue(CampoDelCampo))
	valor := recortar(r.PostFormValue(CampoDelValor))
	huella := recortar(r.PostFormValue(CampoDeHuella))
	if campo == "" || valor == "" || huella == "" {
		s.conAvisoDe(w, r, "documentos.aceptar.falta_campo")
		return
	}

	// 4. Y TIENE QUE SER UNA PROPUESTA QUE EXISTA. Es la tercera forma de la
	//    nada: hay dato y no se entiende, o sea no casa con nada de lo que esta
	//    cuenta tiene propuesto. Se rechaza; no se guarda «por si acaso».
	//
	//    SE BUSCA EN LAS PROPUESTAS DE ESTA CUENTA, no en las de nadie mas: la
	//    cuenta va en la firma del puerto (invariante 12), asi que aceptar la
	//    propuesta de otro exigiria escribirlo a proposito.
	ps, err := s.o.Almacen.Ficha(r.Context(), quien)
	if err != nil {
		s.fallo(err)
		s.conAvisoDe(w, r, "documentos.aceptar.no_se_guarda")
		return
	}
	var elegida PropuestaDeFicha
	hay := false
	for _, p := range ps {
		if p.Campo == campo && p.Valor == valor && p.Huella == huella {
			elegida, hay = p, true
			break
		}
	}
	if !hay {
		s.conAvisoDe(w, r, "documentos.aceptar.no_casa")
		return
	}

	if err := s.o.Almacen.Aceptar(r.Context(), quien, elegida); err != nil {
		s.fallo(err)
		s.conAvisoDe(w, r, "documentos.aceptar.no_se_guarda")
		return
	}
	// SE REDIRIGE Y NO SE PINTA. Un POST que contesta con la pagina deja el
	// formulario reenviable con F5, y aqui reenviar significa volver a afirmar
	// algo con tu nombre. Es el mismo patron que la subida.
	http.Redirect(w, r, s.o.Base+"/", http.StatusSeeOther)
}

// recortar quita los espacios y acota el largo.
//
// EL TOPE NO ES DECORATIVO: estos tres valores se comparan contra propuestas que
// ya existen, asi que uno de un megabyte no puede casar con nada, pero si puede
// costar memoria y tiempo en la comparacion. Se corta antes.
func recortar(s string) string {
	s = strings.TrimSpace(s)
	if r := []rune(s); len(r) > MaxLargoAcepta {
		return string(r[:MaxLargoAcepta])
	}
	return s
}
