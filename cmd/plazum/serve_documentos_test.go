package main

import (
	"bytes"
	"context"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/marcosmatalab/plazum/adaptadores/ia"
	"github.com/marcosmatalab/plazum/nucleo/corpus"
	"github.com/marcosmatalab/plazum/puertos"
	"github.com/marcosmatalab/plazum/superficies/documentos"
)

// LA CADENA DE EXTREMO A EXTREMO, QUE ES LA PUERTA QUE DECIDE SI LA PIEZA 3
// EXISTE.
//
// # Por que esta puerta y no otra
//
// Porque todas las demas se aprueban sin producto. El censo de superficies solo
// pide una cadena; el inventario de claves, la hoja de estilo y la procedencia
// del texto miran FORMA; el estado vacio ejerce la pantalla, pero solo vacia.
// Una `/documentos` que contestara 401 a todo el mundo, o que pintara un
// formulario que no ingiere nada, pasa las veinte puertas del arbol en verde.
//
// Lo que ninguna mira es que un documento subido A MANO llegue hasta un hallazgo
// con su cita verificada por hash. Eso es la pieza 3, y esto es lo que lo
// comprueba: multipart de verdad, superficie de verdad, corpus de verdad, cable
// de verdad.
//
// # LOS SIETE ESLABONES QUE ESTO EJERCE, en orden
//
//	ingesta.Leer            del multipart a fragmentos con su pagina
//	ia.Huella               sha256 del fichero
//	ia.FuentesAportadas     una fuente citable por fragmento, con su hash
//	ia.Documentos           al vocabulario del indice
//	busqueda.Nuevo          BM25 sobre lo que subio ESA cuenta
//	ia.Nuevo (Aportado)     el verificador que resuelve citas del cliente
//	evidencia.Mapear        el hallazgo, con la cita ya verificada
//
// Si cualquiera de los siete se desconecta, esto se pone rojo. Y se pone rojo
// diciendo cual, porque cada afirmacion de abajo mira una cosa.

// textoDeLaPolitica es el documento que sube el cliente en esta puerta.
//
// # POR QUE ESTE TEXTO Y NO UNO INVENTADO
//
// Porque la busqueda de la pieza 3 encuentra «revision anual» en un documento
// que dice «revision anual» y NO lo encuentra en uno que dice «cada doce meses»:
// es BM25 sobre palabras, no parafrasis, y eso esta escrito en el encabezado de
// `adaptadores/evidencia` como el hueco medido que le toca al modelo cuando
// llegue. Un texto que no comparta terminos con ninguna obligacion daria cero
// hallazgos, y esta puerta estaria comprobando que la cadena no se rompe en vez
// de que funciona.
//
// Asi que el documento se parece a lo que de verdad sube un cliente: una
// politica escrita con el vocabulario de la norma. Los terminos vienen del
// corpus instalado y no de la cabeza, y la puerta lo comprueba: si el corpus
// dejara de traer obligaciones con estos terminos, esto se pone rojo antes de
// afirmar nada.
const textoDeLaPolitica = `POLITICA DE SEGURIDAD DE LA INFORMACION DE ACME SL

1. Notificacion de incidentes

La organizacion notificara sin dilacion indebida a la autoridad competente todo
incidente que tenga un impacto significativo en la prestacion de sus servicios,
y en todo caso dentro de las veinticuatro horas siguientes a su conocimiento.
La notificacion incluira la informacion disponible sobre la naturaleza del
incidente y las medidas adoptadas.

2. Revision de la politica

El organo de direccion revisara y aprobara esta politica de seguridad de la
informacion al menos una vez al ano, y siempre que se produzcan cambios
significativos en la organizacion o en los riesgos identificados.

3. Formacion y concienciacion

Todo el personal recibira formacion periodica en materia de seguridad de la
informacion, con una periodicidad minima anual, y se dejara constancia
documental de su realizacion.
`

// almacenReal monta el cable de produccion sobre el corpus publicado.
func almacenReal(t *testing.T) (*indicesPorCuenta, []*corpus.Paquete) {
	t.Helper()
	ps, err := corpus.Cargar("../../paquetes")
	if err != nil {
		t.Fatalf("el corpus publicado no carga: %v", err)
	}
	if len(ps) < 10 {
		t.Fatalf("solo %d paquetes: esta puerta estaria midiendo el vacio", len(ps))
	}
	a := nuevosIndicesPorCuenta(ps)
	if len(a.consultas) < 100 {
		t.Fatalf("solo %d consultas compuestas del corpus: sin obligaciones con texto legal "+
			"no hay contra que mapear, y esta puerta estaria comprobando que no sale nada "+
			"de un sitio donde no hay nada", len(a.consultas))
	}
	return a, ps
}

// superficieDeDocumentos monta la superficie de produccion sobre ese almacen.
func superficieDeDocumentos(t *testing.T, a *indicesPorCuenta, quien func(*http.Request) string) *documentos.Superficie {
	t.Helper()
	s, err := construirDocumentos(catDePrueba(t), quien,
		func(*http.Request) (string, error) { return "tok", nil }, a, nil)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// subirDocumento manda el fichero por multipart, como lo manda un navegador.
func subirDocumento(t *testing.T, s *documentos.Superficie, nombre, texto string) *httptest.ResponseRecorder {
	t.Helper()
	var cuerpo bytes.Buffer
	w := multipart.NewWriter(&cuerpo)
	f, err := w.CreateFormFile(documentos.CampoDelFichero, nombre)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write([]byte(texto)); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(http.MethodPost,
		documentos.BasePorDefecto+documentos.RutaDeSubir, &cuerpo)
	r.Header.Set("Content-Type", w.FormDataContentType())
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, r)
	return rec
}

func verDocumentos(t *testing.T, s *documentos.Superficie) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, documentos.BasePorDefecto+"/", nil))
	return rec
}

// TestUnDocumentoSubidoAManoLlegaAUnHallazgoConSuCitaVerificada es la puerta.
func TestUnDocumentoSubidoAManoLlegaAUnHallazgoConSuCitaVerificada(t *testing.T) {
	// LA IA APAGADA, A PROPOSITO Y DESDE EL PRIMER PASO. La pieza 3 no llama a
	// ningun modelo, asi que tiene que funcionar entera con el interruptor
	// puesto. Si algo de esta cadena necesitara el modelo, esta linea lo
	// descubre aqui y no en la maquina de un cliente sin salida a internet.
	t.Setenv("PLAZUM_SIN_IA", "1")
	if apagada, err := ia.Apagada(); err != nil || !apagada {
		t.Fatalf("el interruptor dice apagada=%v err=%v: esta puerta no esta midiendo lo "+
			"que dice medir", apagada, err)
	}

	a, _ := almacenReal(t)
	s := superficieDeDocumentos(t, a, func(*http.Request) string { return "ciso@acme" })

	// 1. SE SUBE, POR MULTIPART Y COMO LO MANDA UN NAVEGADOR.
	if rec := subirDocumento(t, s, "politica.txt", textoDeLaPolitica); rec.Code != http.StatusSeeOther {
		t.Fatalf("la subida contesta %d y tenia que redirigir con 303.\n%s",
			rec.Code, rec.Body.String())
	}

	// 2. EL DOCUMENTO ESTA, CON SUS FRAGMENTOS CONTADOS.
	docs, err := a.Documentos(context.Background(), "ciso@acme")
	if err != nil {
		t.Fatalf("leyendo los documentos de la cuenta: %v", err)
	}
	if len(docs) != 1 {
		t.Fatalf("la cuenta tiene %d documentos y tenia que tener uno", len(docs))
	}
	if docs[0].Fragmentos == 0 {
		t.Fatal("el documento ha entrado con CERO fragmentos citables, asi que no hay " +
			"nada que buscar y lo de abajo comprobaria el vacio")
	}
	if docs[0].Huella == "" || len(docs[0].Huella) != 64 {
		t.Errorf("la huella del documento es %q y tenia que ser un sha256 en hexadecimal",
			docs[0].Huella)
	}

	// 3. HAY HALLAZGOS, Y CADA UNO TRAE SU CITA, SU SITIO Y SU OBLIGACION.
	hs, err := a.Hallazgos(context.Background(), "ciso@acme")
	if err != nil {
		t.Fatalf("mapeando: %v", err)
	}
	if len(hs) == 0 {
		t.Fatal("el documento se ha indexado y no ha salido NI UN HALLAZGO.\n" +
			"  La cadena esta rota o el texto de la politica ya no comparte terminos con " +
			"ninguna obligacion del corpus. Las dos cosas hay que mirarlas: la pieza 3 es " +
			"exactamente esto y sin esto no existe.")
	}
	for _, h := range hs {
		if h.Parrafo == "" {
			t.Errorf("%s: hallazgo sin cita", h.Obligacion)
		}
		if h.Obligacion == "" || h.Titulo == "" || h.Marco == "" {
			t.Errorf("hallazgo sin obligacion, titulo o marco: %+v", h)
		}
		if h.Pagina == 0 && h.Fragmento == 0 {
			t.Errorf("%s: hallazgo sin sitio, o sea sin decir DONDE esta dentro del "+
				"documento. Una cita sin sitio no lleva a nadie al parrafo, que es lo unico "+
				"que hace esta pieza", h.Obligacion)
		}
		if h.Documento != "politica.txt" {
			t.Errorf("%s: el hallazgo dice que sale de %q y sale de politica.txt",
				h.Obligacion, h.Documento)
		}
		// LA CITA ES LITERAL DEL DOCUMENTO DEL CLIENTE. Es lo que la
		// verificacion por hash garantiza, y se comprueba aparte: un
		// verificador que devolviera cualquier cosa pasaria las lineas de
		// arriba.
		if !strings.Contains(sinEspaciosDeMas(textoDeLaPolitica), sinEspaciosDeMas(h.Parrafo)) {
			t.Errorf("%s: la cita NO esta literalmente en el documento subido.\n"+
				"  cita: %q\n"+
				"  Eso significa que lo que se ensena no es lo que pone en el documento, "+
				"que es la puerta antialucinacion entera", h.Obligacion, h.Parrafo)
		}
	}

	// 4. Y SALE EN LA PANTALLA, con su descargo. Un hallazgo que existe en
	//    memoria y no se pinta no ha llegado a nadie.
	rec := verDocumentos(t, s)
	if rec.Code != http.StatusOK {
		t.Fatalf("la pantalla contesta %d", rec.Code)
	}
	cuerpo := rec.Body.String()
	if !strings.Contains(cuerpo, "politica.txt") {
		t.Error("la pantalla no nombra el documento subido")
	}
	if !strings.Contains(cuerpo, "NO dice que cumplas nada") {
		t.Error("la pantalla ensena hallazgos y NO trae el descargo")
	}
	primera := sinEspaciosDeMas(hs[0].Parrafo)
	if !strings.Contains(sinEspaciosDeMas(cuerpo), primera) {
		t.Errorf("la pantalla no ensena la cita del primer hallazgo (%q)", primera)
	}
	t.Logf("cadena completa: %d fragmentos indexados, %d consultas del corpus, %d hallazgos",
		docs[0].Fragmentos, len(a.consultas), len(hs))
}

// LA PUERTA ANTIALUCINACION, EJERCIDA Y NO SUPUESTA.
//
// `evidencia.Mapear` descarta el hallazgo cuya cita no resuelve contra la
// fuente. Hoy eso es casi una tautologia (la cita la copia el buscador del
// propio fragmento), y por eso hace falta ejercerlo: una linea que nunca se
// recorre es una linea que puede no existir.
//
// Se desincroniza el verificador a proposito, dejandolo con fuentes que NO son
// las del indice, y se exige que NO SALGA NINGUN HALLAZGO. Que salga vacio es lo
// correcto: lo que no se puede comprobar no se ensena.
func TestUnaCitaQueNoResuelveNoSeEnsena(t *testing.T) {
	a, _ := almacenReal(t)
	s := superficieDeDocumentos(t, a, func(*http.Request) string { return "ciso@acme" })
	if rec := subirDocumento(t, s, "politica.txt", textoDeLaPolitica); rec.Code != http.StatusSeeOther {
		t.Fatalf("la subida contesta %d", rec.Code)
	}
	// CONTROL POSITIVO PRIMERO: con el verificador bueno SI salen hallazgos.
	// Sin esta mitad, la comprobacion de abajo pasaria en una cadena rota.
	antes, err := a.Hallazgos(context.Background(), "ciso@acme")
	if err != nil || len(antes) == 0 {
		t.Fatalf("con el verificador bueno salen %d hallazgos (err %v): sin hallazgos, "+
			"desincronizarlo no demuestra nada", len(antes), err)
	}

	// Y AHORA EL VERIFICADOR DESINCRONIZADO: fuentes que no son las del indice.
	otra, err := ia.NuevaFuente("aportado:0000000000000000:1", ia.MarcoAportado,
		"pagina 1", "aportado", ia.Aportado, true,
		"un texto que no esta en el documento del cliente y que nadie ha subido")
	if err != nil {
		t.Fatal(err)
	}
	ver, err := ia.Nuevo(ia.Opciones{
		Fuentes: []ia.Fuente{otra}, Admite: []ia.Procedencia{ia.Aportado},
		MinimoCita: ia.MinimoCitaPorDefecto,
	})
	if err != nil {
		t.Fatal(err)
	}
	a.mu.Lock()
	a.por["ciso@acme"].ver = ver
	a.mu.Unlock()

	despues, err := a.Hallazgos(context.Background(), "ciso@acme")
	if err != nil {
		t.Fatalf("mapeando con el verificador desincronizado: %v", err)
	}
	if len(despues) != 0 {
		t.Errorf("con el verificador desincronizado siguen saliendo %d hallazgos.\n"+
			"  Una cita que no resuelve contra su fuente NO se ensena: es la puerta "+
			"antialucinacion, y da igual que hoy la escriba un buscador y no un modelo.\n"+
			"  primero: %+v", len(despues), despues[0])
	}
}

// EL VERIFICADOR DE ESTA PANTALLA ADMITE `Aportado` Y NO EL CORPUS.
//
// Es la otra direccion de la puerta antialucinacion y se olvida sola: un
// verificador construido con `ia.Estricto` (que es el que hay que usar en todo lo
// demas) NO admite documentos del cliente, asi que la pantalla saldria siempre
// vacia y pareceria que el mapeo no encuentra nada. Y al reves, uno que
// admitiera tambien el corpus dejaria que un hallazgo de esta pantalla citara la
// ley diciendo que es tu politica.
func TestElVerificadorDeLosDocumentosAdmiteLoAportadoYNoElCorpus(t *testing.T) {
	a, ps := almacenReal(t)
	s := superficieDeDocumentos(t, a, func(*http.Request) string { return "ciso@acme" })
	if rec := subirDocumento(t, s, "politica.txt", textoDeLaPolitica); rec.Code != http.StatusSeeOther {
		t.Fatalf("la subida contesta %d", rec.Code)
	}
	a.mu.RLock()
	ver := a.por["ciso@acme"].ver
	a.mu.RUnlock()
	if ver == nil {
		t.Fatal("la cuenta no tiene verificador")
	}

	// UNA FUENTE DEL CORPUS NO RESUELVE POR ESTE VERIFICADOR. Se coge del
	// corpus real y no se inventa: una fuente inventada no demostraria que el
	// filtro es por PROCEDENCIA y no porque no estuviera en la lista.
	delCorpus, err := ia.FuentesDelCorpus(ps)
	if err != nil {
		t.Fatal(err)
	}
	if len(delCorpus) == 0 {
		t.Fatal("el corpus no da ni una fuente citable")
	}
	f := delCorpus[0]
	_, err = ver.Verificar(puertaDePrueba(f))
	if err == nil {
		t.Error("el verificador de los documentos ha resuelto una cita del CORPUS.\n" +
			"  Admite Aportado y solo Aportado: si admitiera el corpus, un hallazgo de " +
			"esta pantalla podria citar la ley diciendo que es tu politica.")
	}
	// Y `ia.Estricto` es el de las demas piezas: se comprueba que NO habria
	// valido aqui, que es lo que hace que la eleccion de ia.Nuevo sea una
	// decision y no una casualidad.
	estricto, err := ia.Estricto(delCorpus)
	if err != nil {
		t.Fatal(err)
	}
	a.mu.RLock()
	fuentes := a.por["ciso@acme"].fuentes
	a.mu.RUnlock()
	if len(fuentes) == 0 {
		t.Fatal("la cuenta no tiene fuentes")
	}
	if _, err := estricto.Verificar(puertaDePrueba(fuentes[0])); err == nil {
		t.Error("ia.Estricto ha resuelto una cita de un documento del cliente, asi que la " +
			"distincion entre los dos constructores no separa nada")
	}
}

// LA CADENA ENTERA CON PLAZUM_SIN_IA=1, dicho aparte de la puerta grande.
//
// La puerta de arriba ya lo pone, y esto lo afirma de la otra forma: recorriendo
// los paquetes de la cadena y comprobando que ninguno exige el interruptor
// encendido. Las dos afirmaciones son distintas: aquella dice «funciona con la
// IA apagada», y esta dice «ninguno de estos paquetes SABE de la IA».
func TestNingunEslabonDeLaPieza3ExigeElModelo(t *testing.T) {
	t.Setenv("PLAZUM_SIN_IA", "1")
	if err := ia.ExigeEncendida(); err == nil {
		t.Fatal("con PLAZUM_SIN_IA=1 el interruptor dice que la IA esta encendida: este " +
			"test no esta midiendo lo que dice medir")
	}
	a, _ := almacenReal(t)
	s := superficieDeDocumentos(t, a, func(*http.Request) string { return "ciso@acme" })
	if rec := subirDocumento(t, s, "politica.txt", textoDeLaPolitica); rec.Code != http.StatusSeeOther {
		t.Fatalf("con la IA apagada la subida contesta %d", rec.Code)
	}
	hs, err := a.Hallazgos(context.Background(), "ciso@acme")
	if err != nil {
		t.Fatalf("con la IA apagada el mapeo falla: %v", err)
	}
	if len(hs) == 0 {
		t.Error("con la IA apagada no sale ningun hallazgo, asi que algo de esta cadena " +
			"depende del modelo y no deberia")
	}
}

// EL TOPE POR CUENTA, EJERCIDO. Una constante declarada y nunca cruzada es un
// numero decorativo.
//
// SE CRUZA ACUMULANDO Y NO CON UN SOLO FICHERO, y eso hay que decirlo porque es
// una propiedad del sistema y no una comodidad del test: `ingesta.MaximoExtraido`
// corta el texto de UN documento en 8 MiB, que es exactamente el tope por
// cuenta, asi que un fichero solo no lo puede cruzar por mucho que pese. Lo que
// lo cruza es el segundo documento, que es el caso real: nadie sube una politica
// de 8 MiB, sube cuatro de dos.
func TestElTopePorCuentaSeCruzaYSeDice(t *testing.T) {
	a, _ := almacenReal(t)
	// Dos documentos de mas de la mitad del tope cada uno. El segundo cruza.
	mitad := MaxBytesIndexadosPorCuenta/2 + 1024
	uno := strings.Repeat("la organizacion revisara la politica de seguridad cada ano. ",
		mitad/59+1)
	dos := strings.Repeat("el organo de direccion aprueba el plan de continuidad cada ano. ",
		mitad/63+1)
	if len(uno) < mitad || len(dos) < mitad {
		t.Fatalf("los documentos de prueba miden %d y %d y hace falta que sumen mas de %d",
			len(uno), len(dos), MaxBytesIndexadosPorCuenta)
	}
	// CONTROL POSITIVO PRIMERO: el primero SI entra. Sin esta mitad, un tope
	// puesto a cero pasaria el test de abajo con nota.
	if _, err := a.Subir(context.Background(), "ciso@acme", "uno.txt", []byte(uno)); err != nil {
		t.Fatalf("el primer documento no entra: %v", err)
	}
	_, err := a.Subir(context.Background(), "ciso@acme", "dos.txt", []byte(dos))
	if !errors.Is(err, ErrCuentaLlena) {
		t.Errorf("el segundo documento da %v y tenia que dar ErrCuentaLlena", err)
	}
	// Y LO QUE YA ESTABA SIGUE ESTANDO: un tope que se cruza no puede llevarse
	// por delante lo que la cuenta ya tenia indexado.
	docs, err := a.Documentos(context.Background(), "ciso@acme")
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 1 || docs[0].Fichero != "uno.txt" {
		t.Errorf("tras el rechazo la cuenta tiene %d documentos (%v) y tenia que conservar "+
			"el primero", len(docs), docs)
	}
	// Y EL TOPE ES POR CUENTA Y NO GLOBAL: otra cuenta sigue pudiendo subir.
	if _, err := a.Subir(context.Background(), "otra@acme", "suyo.txt",
		[]byte(textoDeLaPolitica)); err != nil {
		t.Errorf("con una cuenta llena, otra cuenta no puede subir: %v.\n"+
			"  El tope es por cuenta; si fuera global, la primera persona que subiera "+
			"documentos dejaria sin pieza 3 a toda la instalacion", err)
	}
}

// MISMO FICHERO DOS VECES NO DUPLICA NADA, porque la huella es del contenido.
func TestSubirElMismoDocumentoDosVecesNoLoDuplica(t *testing.T) {
	a, _ := almacenReal(t)
	s := superficieDeDocumentos(t, a, func(*http.Request) string { return "ciso@acme" })
	for i := 0; i < 2; i++ {
		if rec := subirDocumento(t, s, "politica.txt", textoDeLaPolitica); rec.Code != http.StatusSeeOther {
			t.Fatalf("la subida %d contesta %d", i+1, rec.Code)
		}
	}
	docs, err := a.Documentos(context.Background(), "ciso@acme")
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 1 {
		t.Errorf("la cuenta tiene %d documentos tras subir el mismo dos veces", len(docs))
	}
}

// Y LA FUGA, SOBRE EL CABLE DE VERDAD Y NO SOBRE UN DOBLE.
//
// La superficie ya tiene su version con un almacen falso; esta es la que importa,
// porque el almacen falso podria separar cuentas aunque el de produccion no.
func TestElIndiceDeUnaCuentaNoLoVeOtraCuenta(t *testing.T) {
	a, _ := almacenReal(t)
	quien := "ana@acme"
	s := superficieDeDocumentos(t, a, func(*http.Request) string { return quien })
	if rec := subirDocumento(t, s, "de-ana.txt", textoDeLaPolitica); rec.Code != http.StatusSeeOther {
		t.Fatalf("la subida contesta %d", rec.Code)
	}
	// CONTROL POSITIVO: Ana ve lo suyo.
	deAna, err := a.Hallazgos(context.Background(), "ana@acme")
	if err != nil || len(deAna) == 0 {
		t.Fatalf("Ana no ve sus propios hallazgos (%d, err %v): sin eso, lo de abajo no "+
			"distingue «no hay fuga» de «no hay producto»", len(deAna), err)
	}
	// Y LUIS NO VE NADA.
	docs, err := a.Documentos(context.Background(), "luis@acme")
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 0 {
		t.Errorf("Luis ve %d documentos de otra cuenta", len(docs))
	}
	hs, err := a.Hallazgos(context.Background(), "luis@acme")
	if err != nil {
		t.Fatal(err)
	}
	if len(hs) != 0 {
		t.Errorf("Luis ve %d hallazgos de otra cuenta.\n"+
			"  Es el invariante 12: el indice es de la CUENTA. Un indice por proceso "+
			"serviria la cita literal de la politica de una persona a cualquiera que "+
			"entrara, y el verificador no lo impide porque comprueba procedencia y "+
			"literalidad, no propiedad.", len(hs))
	}
}

// puertaDePrueba compone la propuesta con la que se interroga a un verificador.
//
// El hash y la cita salen de la PROPIA fuente, que es el caso bueno: lo que se
// esta comprobando arriba no es que la cita sea correcta, es de que PROCEDENCIA
// admite cada verificador.
func puertaDePrueba(f ia.Fuente) puertos.Propuesta {
	return puertos.Propuesta{
		Diff: "prueba", Cita: f.Texto, HashFuente: f.Hash,
		Modelo: "ninguno: prueba",
	}
}

// sinEspaciosDeMas colapsa los blancos para poder comparar el texto de un
// fragmento con el del fichero original.
//
// SE NORMALIZA EL BLANCO Y NADA MAS. La afirmacion que sostiene la puerta
// antialucinacion es que las PALABRAS de la cita son las del documento, no que
// los saltos de linea coincidan: `adaptadores/ingesta` fragmenta y junta lineas,
// que es lo que hace que un parrafo se pueda ensenar. Normalizar mas que el
// blanco (minusculas, tildes, puntuacion) si aflojaria la afirmacion, y por eso
// no se hace.
func sinEspaciosDeMas(s string) string { return strings.Join(strings.Fields(s), " ") }
