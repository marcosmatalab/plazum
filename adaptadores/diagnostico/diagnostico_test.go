package diagnostico

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"plazum/puertos"
	"plazum/puertos/contrato"
)

// Los identificadores de este fichero son SINTETICOS: adaptadores/ esta
// vigilado por TestNingunaNormaCableada y un adaptador no nombra una norma ni en
// sus pruebas.

const paqueteSano = `{
  "urn": "urn:demo:x", "version": "1.0.0", "clase": 4,
  "licencia": "Apache-2.0", "fuente": "sintetico de prueba",
  "vigencia": {"desde": "2026-01-01"},
  "obligaciones": [
    {"id": "demo.uno", "articulo": "1", "cita": "sintetico",
     "vigencia": {"desde": "2026-01-01"}, "clase_e2e": "documental"}
  ]
}`

// paqueteConClaseFueraDeRango es la forma exacta de esquivar la frontera legal
// que el linter de corpus rechaza. Sirve aqui para tener un corpus que EXISTE y
// no CARGA, que es un estado distinto de "no hay corpus" y tiene otro arreglo.
const paqueteRoto = `{
  "urn": "urn:demo:x", "version": "1.0.0", "clase": 9,
  "licencia": "Apache-2.0", "fuente": "sintetico de prueba",
  "vigencia": {"desde": "2026-01-01"},
  "obligaciones": []
}`

func ahoraDePrueba() time.Time { return time.Date(2026, 8, 26, 9, 0, 0, 0, time.UTC) }

func escribirCorpus(t *testing.T, contenido string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "paquetes")
	if err := os.MkdirAll(filepath.Join(dir, "uno"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "uno", "paquete.json"), []byte(contenido), 0o600); err != nil {
		t.Fatal(err)
	}
	return dir
}

// puertoLibre reserva un puerto y lo suelta, para que el doctor pueda abrirlo.
// Comprobar contra 8080 haria el test dependiente de lo que corra en la maquina.
func puertoLibre(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	d := l.Addr().String()
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	return d
}

func opcionesSanas(t *testing.T) Opciones {
	t.Helper()
	return Opciones{
		Ahora:     ahoraDePrueba(),
		Datos:     t.TempDir(),
		Corpus:    escribirCorpus(t, paqueteSano),
		Direccion: puertoLibre(t),
	}
}

func por(cs []puertos.Comprobacion, nombre string) puertos.Comprobacion {
	for _, c := range cs {
		if c.Nombre == nombre {
			return c
		}
	}
	return puertos.Comprobacion{Nombre: "(ausente) " + nombre}
}

// La suite de contrato, con una instalacion DELIBERADAMENTE ROTA.
//
// Se hace asi y no sobre una sana por una razon concreta: la ultima
// comprobacion del contrato es que todo lo que no esta Correcto diga como se
// arregla, y sobre una instalacion sana casi todo sale Correcto, asi que la
// suite pasaria sin ejercitar la unica exigencia fuerte que tiene. Rota, la
// mitad de las comprobaciones caen del lado que importa.
func TestElDoctorCumpleElContratoSobreUnaInstalacionRota(t *testing.T) {
	contrato.Diagnostico(t, func() puertos.Diagnostico {
		ocupado, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = ocupado.Close() })
		return Nuevo(Opciones{
			Ahora:     time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), // reloj atrasado
			Datos:     t.TempDir(),
			Corpus:    escribirCorpus(t, paqueteRoto), // corpus que no carga
			Direccion: ocupado.Addr().String(),        // puerto ocupado
			Keystore:  filepath.Join(t.TempDir(), "no-esta.json"),
		})
	})
}

func TestElDoctorCumpleElContratoSobreUnaInstalacionSana(t *testing.T) {
	contrato.Diagnostico(t, func() puertos.Diagnostico {
		return Nuevo(opcionesSanas(t))
	})
}

func TestSobreUnaInstalacionSanaNoSeInventaProblemas(t *testing.T) {
	o := opcionesSanas(t)
	// El keystore se crea con permisos de dueno para que no salte el aviso.
	ks := filepath.Join(o.Datos, "keystore.json")
	if err := os.WriteFile(ks, []byte(`{"claves":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	o.Keystore = ks
	cs := Nuevo(o).Comprobar(context.Background())

	for _, nombre := range []string{"reloj", "escritura", "corpus", "keystore", "raices-tsa", "puerto"} {
		c := por(cs, nombre)
		if c.Estado != puertos.Correcto {
			t.Errorf("sobre una instalacion sana, %q sale en %s: %s (arreglo: %s). "+
				"Un doctor que da falsos positivos se deja de mirar",
				nombre, c.Estado, c.Detalle, c.Arreglo)
		}
	}
}

func TestElDoctorCazaUnRelojAtrasado(t *testing.T) {
	o := opcionesSanas(t)
	o.Ahora = FechaDeReferencia.AddDate(0, 0, -1)

	// BARRIDO DE MUTACION, y este test paso de adorno a puerta por esto: la
	// comprobacion del reloj tiene DOS senales, y el directorio de datos que
	// crea t.TempDir() lleva la fecha real de hoy, que es posterior al Ahora
	// que se fija aqui. O sea que la segunda senal saltaba sola y el test daba
	// verde con la primera BORRADA. Se envejece el directorio para que solo
	// pueda saltar la que se quiere probar.
	viejo := o.Ahora.AddDate(0, 0, -30)
	if err := os.Chtimes(o.Datos, viejo, viejo); err != nil {
		t.Fatalf("no puedo envejecer el directorio de datos, y sin eso este test no aisla "+
			"la senal que quiere probar: %v", err)
	}

	c := por(Nuevo(o).Comprobar(context.Background()), "reloj")
	if c.Estado != puertos.Roto {
		t.Fatalf("un reloj anterior a la construccion del binario tenia que salir roto y salio %s: %s",
			c.Estado, c.Detalle)
	}
	if !strings.Contains(c.Detalle, "construyo este binario") {
		t.Errorf("ha saltado otra senal distinta de la que este test prueba: %s", c.Detalle)
	}
	if c.Arreglo == "" {
		t.Fatal("sin arreglo escrito, el operador tiene que buscar como se sincroniza un reloj")
	}
}

// El otro lado del reloj: un fichero nuestro con fecha posterior a "ahora"
// significa que el reloj se ha movido hacia atras desde la ultima ejecucion.
// Esta senal si es local del todo, no depende de la fecha de construccion.
func TestElDoctorCazaUnRelojQueRetrocedioDesdeLaUltimaEjecucion(t *testing.T) {
	o := opcionesSanas(t)
	futuro := o.Ahora.Add(48 * time.Hour)
	if err := os.Chtimes(o.Datos, futuro, futuro); err != nil {
		t.Skipf("este sistema de ficheros no deja fijar la fecha de un directorio: %v", err)
	}
	c := por(Nuevo(o).Comprobar(context.Background()), "reloj")
	if c.Estado != puertos.Roto {
		t.Fatalf("con el directorio de datos modificado en el futuro, el reloj tenia que salir "+
			"roto y salio %s: %s", c.Estado, c.Detalle)
	}
}

func TestElDoctorAvisaDeUnBinarioMuyViejoOUnRelojMuyAdelantado(t *testing.T) {
	o := opcionesSanas(t)
	o.Ahora = FechaDeReferencia.AddDate(AnosDeVidaUtil+1, 0, 0)
	c := por(Nuevo(o).Comprobar(context.Background()), "reloj")
	if c.Estado == puertos.Correcto {
		t.Fatalf("un reloj %d anos por delante de la construccion tenia que avisar y salio %s: %s",
			AnosDeVidaUtil+1, c.Estado, c.Detalle)
	}
	if c.Arreglo == "" {
		t.Fatal("el aviso no dice que hacer")
	}
	// El arreglo tiene que cubrir LAS DOS causas, porque desde aqui no se
	// distinguen sin red y quedarse con una manda al operador por el camino
	// equivocado la mitad de las veces.
	if !strings.Contains(c.Arreglo, "update") {
		t.Error("el arreglo no menciona actualizar, que es una de las dos causas posibles")
	}
}

// La raiz de una TSA caduca en una fecha conocida, y el dia que caduque los
// sellos NUEVOS dejaran de verificar mientras la TSA sigue respondiendo 200.
// Es la forma mas silenciosa de romperse, y por eso hay que avisar antes.
func TestElDoctorAvisaAntesDeQueCaduqueUnaRaizDeTSA(t *testing.T) {
	if len(raicesDelBinario) == 0 {
		t.Fatal("no hay ninguna raiz declarada: la comprobacion no puede probar nada")
	}
	primera := raicesDelBinario[0]
	for _, r := range raicesDelBinario {
		if r.Caduca.Before(primera.Caduca) {
			primera = r
		}
	}

	t.Run("mucho antes, callado", func(t *testing.T) {
		o := opcionesSanas(t)
		o.Ahora = primera.Caduca.Add(-2 * MargenDeAviso)
		c := por(Nuevo(o).Comprobar(context.Background()), "raices-tsa")
		if c.Estado != puertos.Correcto {
			t.Errorf("faltando el doble del margen, no hay nada que avisar y salio %s: %s",
				c.Estado, c.Detalle)
		}
	})
	t.Run("dentro del margen, avisa", func(t *testing.T) {
		o := opcionesSanas(t)
		o.Ahora = primera.Caduca.Add(-MargenDeAviso / 2)
		c := por(Nuevo(o).Comprobar(context.Background()), "raices-tsa")
		if c.Estado != puertos.Aviso {
			t.Errorf("a %v de caducar tenia que avisar y salio %s: %s",
				MargenDeAviso/2, c.Estado, c.Detalle)
		}
		if !strings.Contains(c.Detalle, primera.Nombre) {
			t.Errorf("el aviso no dice cual caduca: %s", c.Detalle)
		}
		if c.Arreglo == "" {
			t.Error("el aviso no dice que hacer")
		}
	})
	t.Run("ya caducada, no es solo un aviso de estilo", func(t *testing.T) {
		o := opcionesSanas(t)
		o.Ahora = primera.Caduca.AddDate(0, 0, 1)
		c := por(Nuevo(o).Comprobar(context.Background()), "raices-tsa")
		if c.Estado == puertos.Correcto {
			t.Fatalf("con una raiz caducada la comprobacion salio correcta: %s", c.Detalle)
		}
		if !strings.Contains(c.Detalle, primera.Nombre) {
			t.Errorf("no dice cual ha caducado: %s", c.Detalle)
		}
	})
	t.Run("todas caducadas, roto", func(t *testing.T) {
		ultima := raicesDelBinario[0]
		for _, r := range raicesDelBinario {
			if r.Caduca.After(ultima.Caduca) {
				ultima = r
			}
		}
		o := opcionesSanas(t)
		o.Ahora = ultima.Caduca.AddDate(0, 0, 1)
		c := por(Nuevo(o).Comprobar(context.Background()), "raices-tsa")
		if c.Estado != puertos.Roto {
			t.Fatalf("sin ninguna raiz viva, la comprobacion tenia que salir rota y salio %s: %s",
				c.Estado, c.Detalle)
		}
	})
}

// Las raices que aporta el operador SI se leen de verdad: llegan en PEM y traen
// su NotAfter dentro, asi que aqui no hay tabla declarada que pueda envejecer.
func TestElDoctorLeeDeVerdadLasRaicesQueDeclaraElOperador(t *testing.T) {
	ahora := ahoraDePrueba()
	casos := []struct {
		nombre  string
		caduca  time.Time
		estado  puertos.EstadoComprobacion
		enDetal string
	}{
		{"vigente", ahora.AddDate(5, 0, 0), puertos.Correcto, "ancla-de-prueba"},
		{"a punto de caducar", ahora.Add(MargenDeAviso / 2), puertos.Aviso, "ancla-de-prueba"},
		{"caducada", ahora.AddDate(0, 0, -1), puertos.Roto, "ancla-de-prueba"},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			o := opcionesSanas(t)
			o.RaicesTSA = certificadoPEM(t, "ancla-de-prueba", ahora.AddDate(-1, 0, 0), caso.caduca)
			c := por(Nuevo(o).Comprobar(context.Background()), "raices-tsa")
			if c.Estado != caso.estado {
				t.Fatalf("esperaba %s y salio %s: %s", caso.estado, c.Estado, c.Detalle)
			}
			if !strings.Contains(c.Detalle, caso.enDetal) {
				t.Errorf("el detalle no nombra el certificado: %s", c.Detalle)
			}
			if c.Estado != puertos.Correcto && c.Arreglo == "" {
				t.Error("no dice como se arregla")
			}
		})
	}
	t.Run("PEM que no trae certificados", func(t *testing.T) {
		o := opcionesSanas(t)
		o.RaicesTSA = []byte("esto no es un PEM\n")
		c := por(Nuevo(o).Comprobar(context.Background()), "raices-tsa")
		if c.Estado != puertos.Roto {
			t.Fatalf("declarar raices y no poner ninguna tenia que salir roto y salio %s: %s",
				c.Estado, c.Detalle)
		}
	})
	t.Run("PEM con un certificado ilegible", func(t *testing.T) {
		o := opcionesSanas(t)
		o.RaicesTSA = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: []byte{0x30, 0x84}})
		c := por(Nuevo(o).Comprobar(context.Background()), "raices-tsa")
		if c.Estado != puertos.Roto {
			t.Fatalf("un certificado ilegible tenia que salir roto y salio %s: %s", c.Estado, c.Detalle)
		}
	})
}

// Corpus ausente y corpus roto son dos problemas distintos con dos arreglos
// distintos, y confundirlos es una llamada de soporte.
func TestElDoctorDistingueCorpusAusenteDeCorpusQueNoCarga(t *testing.T) {
	t.Run("ausente", func(t *testing.T) {
		o := opcionesSanas(t)
		o.Corpus = filepath.Join(t.TempDir(), "no-existe")
		c := por(Nuevo(o).Comprobar(context.Background()), "corpus")
		if c.Estado != puertos.Aviso {
			t.Fatalf("un corpus sin instalar todavia no es un fallo, es un aviso, y salio %s: %s",
				c.Estado, c.Detalle)
		}
		if !strings.Contains(c.Arreglo, "demo") {
			t.Errorf("el arreglo de un corpus ausente tiene que llevar al operador al demo, "+
				"que es el camino mas corto a ver algo: %s", c.Arreglo)
		}
	})
	t.Run("vacio", func(t *testing.T) {
		o := opcionesSanas(t)
		o.Corpus = t.TempDir()
		c := por(Nuevo(o).Comprobar(context.Background()), "corpus")
		if c.Estado != puertos.Aviso || c.Arreglo == "" {
			t.Fatalf("un directorio de corpus vacio tenia que avisar con arreglo y salio %s: %s",
				c.Estado, c.Detalle)
		}
	})
	t.Run("no carga", func(t *testing.T) {
		o := opcionesSanas(t)
		o.Corpus = escribirCorpus(t, paqueteRoto)
		c := por(Nuevo(o).Comprobar(context.Background()), "corpus")
		if c.Estado != puertos.Roto {
			t.Fatalf("un corpus que el linter rechaza tenia que salir roto y salio %s: %s",
				c.Estado, c.Detalle)
		}
		if !strings.Contains(c.Detalle, "uno") {
			t.Errorf("el detalle no dice que paquete lo rompe, asi que con treinta instalados "+
				"hay que ir uno a uno: %s", c.Detalle)
		}
	})
}

func TestElDoctorCazaUnPuertoOcupado(t *testing.T) {
	ocupado, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ocupado.Close() }()

	o := opcionesSanas(t)
	o.Direccion = ocupado.Addr().String()
	c := por(Nuevo(o).Comprobar(context.Background()), "puerto")
	if c.Estado != puertos.Roto {
		t.Fatalf("un puerto ocupado tenia que salir roto y salio %s: %s", c.Estado, c.Detalle)
	}
	if !strings.Contains(c.Arreglo, o.Direccion) {
		t.Errorf("el arreglo no dice que direccion mirar: %s", c.Arreglo)
	}
}

func TestElDoctorCazaUnDirectorioDeDatosEnElQueNoSePuedeEscribir(t *testing.T) {
	// Portable: se pone un FICHERO donde tenia que ir el directorio. MkdirAll
	// falla en todos los sistemas, y no depende de chmod, que en Windows no
	// significa lo mismo.
	base := t.TempDir()
	estorbo := filepath.Join(base, "datos")
	if err := os.WriteFile(estorbo, []byte("no soy un directorio"), 0o600); err != nil {
		t.Fatal(err)
	}
	o := opcionesSanas(t)
	o.Datos = filepath.Join(estorbo, "dentro")
	c := por(Nuevo(o).Comprobar(context.Background()), "escritura")
	if c.Estado != puertos.Roto {
		t.Fatalf("un directorio de datos que no se puede crear tenia que salir roto y salio %s: %s",
			c.Estado, c.Detalle)
	}
	if c.Arreglo == "" {
		t.Fatal("no dice como se arregla")
	}
}

func TestElDoctorDistingueUnKeystoreAusenteDeUnoIlegible(t *testing.T) {
	t.Run("ausente es aviso, no fallo", func(t *testing.T) {
		o := opcionesSanas(t)
		o.Keystore = filepath.Join(o.Datos, "todavia-no.json")
		c := por(Nuevo(o).Comprobar(context.Background()), "keystore")
		if c.Estado != puertos.Aviso {
			t.Fatalf("una instalacion recien hecha no tiene keystore y eso no es un fallo; salio %s: %s",
				c.Estado, c.Detalle)
		}
	})
	t.Run("un directorio donde iba el fichero", func(t *testing.T) {
		o := opcionesSanas(t)
		o.Keystore = filepath.Join(o.Datos, "keystore.json")
		if err := os.MkdirAll(o.Keystore, 0o750); err != nil {
			t.Fatal(err)
		}
		c := por(Nuevo(o).Comprobar(context.Background()), "keystore")
		if c.Estado != puertos.Roto {
			t.Fatalf("un directorio donde iba el fichero de claves tenia que salir roto y salio %s: %s",
				c.Estado, c.Detalle)
		}
	})
	t.Run("permisos abiertos", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("en Windows el modo que devuelve Stat es sintetico y juzgarlo seria mentir")
		}
		o := opcionesSanas(t)
		o.Keystore = filepath.Join(o.Datos, "keystore.json")
		if err := os.WriteFile(o.Keystore, []byte("{}"), 0o644); err != nil {
			t.Fatal(err)
		}
		c := por(Nuevo(o).Comprobar(context.Background()), "keystore")
		if c.Estado != puertos.Aviso {
			t.Fatalf("un keystore legible por toda la maquina tenia que avisar y salio %s: %s",
				c.Estado, c.Detalle)
		}
		if !strings.Contains(c.Arreglo, "chmod") {
			t.Errorf("el arreglo no dice el comando: %s", c.Arreglo)
		}
	})
}

// Un contexto ya cancelado no puede acabar en una pantalla vacia que parezca un
// diagnostico limpio.
func TestUnDiagnosticoCortadoLoDiceEnVezDeSalirVacio(t *testing.T) {
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	cs := Nuevo(opcionesSanas(t)).Comprobar(ctx)
	if len(cs) == 0 {
		t.Fatal("un diagnostico cortado devolvio cero comprobaciones, que se lee como que " +
			"no hay nada que comprobar")
	}
	ultima := cs[len(cs)-1]
	if ultima.Estado == puertos.Correcto || ultima.Arreglo == "" {
		t.Fatalf("el corte no se cuenta como tal: %+v", ultima)
	}
}

// certificadoPEM fabrica un certificado autofirmado con las fechas que se le
// pidan, para poder comprobar de verdad la lectura de NotAfter.
func certificadoPEM(t *testing.T, nombre string, desde, hasta time.Time) []byte {
	t.Helper()
	clave, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	plantilla := x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: nombre},
		NotBefore:             desde,
		NotAfter:              hasta,
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign,
	}
	der, err := x509.CreateCertificate(rand.Reader, &plantilla, &plantilla, &clave.PublicKey, clave)
	if err != nil {
		t.Fatal(err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}
