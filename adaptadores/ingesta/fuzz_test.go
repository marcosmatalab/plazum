package ingesta_test

import (
	"bytes"
	"compress/zlib"
	"testing"

	"github.com/marcosmatalab/plazum/adaptadores/ingesta"
)

// EL FUZZING DEL EXTRACTOR, QUE AQUI NO ES OPCIONAL.
//
// Este paquete parsea ficheros que sube un desconocido. Es, junto al parser de
// corpus, al ledger y al verificador de sellos, uno de los cuatro sitios donde
// entra un byte que no ha escrito nadie de confianza. Un parser de PDF escrito a
// mano sin fuzzing es negligencia con buena letra.
//
// LO QUE SE AFIRMA, y es lo unico que se puede afirmar de un fuzz: que no hay
// entrada que haga PANICAR, colgarse en un bucle o devolver un documento con
// fragmentos vacios. Que el texto extraido sea el correcto no lo puede decir un
// fuzz, y lo dicen los casos de ingesta_test.go.
//
// LA TERCERA FORMA DE LA NADA TAMBIEN SE AFIRMA AQUI, que es lo que hace que
// este fuzz vigile la propiedad del paquete y no solo su robustez: si no hay
// error, entonces o hay fragmentos con texto de verdad, o el fichero estaba
// vacio. Nunca «cero fragmentos y todo bien» sobre un fichero con contenido.
func FuzzLeer(f *testing.F) {
	f.Add("politica.txt", []byte("La direccion aprueba esta politica y la revisa cada ano."))
	f.Add("vacio.txt", []byte(""))
	f.Add("x.pdf", []byte("%PDF-1.4\n1 0 obj\n<< /Type /Page >>\nendobj\n"+
		"2 0 obj\n<< /Length 30 >>\nstream\nBT (hola que tal, esto es texto) Tj ET\nendstream\n"))
	f.Add("cifrado.pdf", []byte("%PDF-1.4\ntrailer\n<< /Encrypt 9 0 R >>\n"))
	f.Add("roto.pdf", []byte("%PDF-1.4\nstream\n"))
	f.Add("parentesis.pdf", []byte("%PDF-1.4\nstream\nBT ((((((((( Tj ET\nendstream\n"))
	f.Add("hex.pdf", []byte("%PDF-1.4\nstream\nBT <4142 Tj ET\nendstream\n"))
	f.Add("octal.pdf", []byte(`%PDF-1.4`+"\nstream\nBT (\\777\\0\\1) Tj ET\nendstream\n"))

	// Un flujo comprimido de verdad, para que el fuzz llegue al inflado en vez
	// de rebotar siempre en el zlib invalido.
	var z bytes.Buffer
	w := zlib.NewWriter(&z)
	_, _ = w.Write([]byte("BT (texto comprimido suficientemente largo para pasar) Tj ET"))
	_ = w.Close()
	f.Add("comprimido.pdf", append([]byte("%PDF-1.4\n<< /Filter /FlateDecode >>\nstream\n"),
		append(z.Bytes(), []byte("\nendstream\n")...)...))

	f.Fuzz(func(t *testing.T, nombre string, datos []byte) {
		doc, err := ingesta.Leer(nombre, datos)
		if err != nil {
			return
		}
		if len(doc.Fragmentos) == 0 {
			// Sin error y sin fragmentos solo puede significar un fichero
			// vacio. Cualquier otra cosa es la tercera forma de la nada leida
			// como la segunda, que es el fallo que este paquete existe para
			// impedir.
			if doc.Formato != ingesta.TextoPlano {
				t.Fatalf("cero fragmentos, sin error y formato %v sobre %d bytes",
					doc.Formato, len(datos))
			}
			return
		}
		for _, fr := range doc.Fragmentos {
			if fr.Texto == "" {
				t.Fatalf("fragmento vacio en la salida: %+v", fr)
			}
			if fr.Orden <= 0 {
				t.Fatalf("fragmento sin orden: %+v", fr)
			}
			if fr.Pagina < 0 {
				t.Fatalf("pagina negativa: %+v", fr)
			}
			if n := len([]rune(fr.Texto)); n > ingesta.LimiteFragmento {
				t.Fatalf("fragmento de %d runas, por encima del limite", n)
			}
		}
		if len(doc.Fragmentos) > ingesta.MaximoFragmentos {
			t.Fatalf("%d fragmentos, por encima del maximo", len(doc.Fragmentos))
		}
	})
}
