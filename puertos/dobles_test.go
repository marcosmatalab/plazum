package puertos_test

import (
	"context"
	"io"
	"time"

	"github.com/marcosmatalab/plazum/puertos"
)

// Dobles vacios de los puertos de la etapa 2.
//
// No son implementaciones y no hacen nada: la etapa 2 arranca con los puertos
// congelados y cero codigo detras, a proposito. Lo que aportan es una
// comprobacion en tiempo de COMPILACION de que cada interfaz se puede
// satisfacer. Una interfaz que nadie ha intentado implementar suele tener un
// metodo imposible o un tipo que no existe, y es mucho mejor descubrirlo aqui
// que en tres frentes a la vez.
//
// Sirven ademas de esqueleto: quien empiece un adaptador puede copiar la firma
// de aqui y no equivocarse.

var (
	_ puertos.Servidor     = (*servidorNulo)(nil)
	_ puertos.Sesion       = (*sesionNula)(nil)
	_ puertos.Plantilla    = (*plantillaNula)(nil)
	_ puertos.Catalogo     = (*catalogoNulo)(nil)
	_ puertos.Actualizador = (*actualizadorNulo)(nil)
	_ puertos.Diagnostico  = (*diagnosticoNulo)(nil)
	_ puertos.Secretos     = (*secretosNulos)(nil)
)

type servidorNulo struct{}

func (*servidorNulo) Arrancar(context.Context, string) error { return nil }
func (*servidorNulo) Parar(context.Context) error            { return nil }

type sesionNula struct{}

func (*sesionNula) Abrir(context.Context, string, time.Duration) (string, error) { return "", nil }
func (*sesionNula) Leer(context.Context, string) (string, error)                 { return "", nil }
func (*sesionNula) Cerrar(context.Context, string) error                         { return nil }
func (*sesionNula) TokenCSRF(context.Context, string) (string, error)            { return "", nil }
func (*sesionNula) ComprobarCSRF(context.Context, string, string) error          { return nil }

type plantillaNula struct{}

func (*plantillaNula) Render(io.Writer, string, any, string) error { return nil }

type catalogoNulo struct{}

func (*catalogoNulo) Traducir(_, clave string, _ ...any) string { return clave }
func (*catalogoNulo) Idiomas() []string                         { return nil }
func (*catalogoNulo) Faltantes(string) []string                 { return nil }

type actualizadorNulo struct{}

func (*actualizadorNulo) Disponible(context.Context) (string, string, error) { return "", "", nil }
func (*actualizadorNulo) Aplicar(context.Context, string) (string, error)    { return "", nil }
func (*actualizadorNulo) Deshacer(context.Context, string) error             { return nil }

type diagnosticoNulo struct{}

func (*diagnosticoNulo) Comprobar(context.Context) []puertos.Comprobacion { return nil }

type secretosNulos struct{}

func (*secretosNulos) Token(int) (string, error) { return "", nil }
func (*secretosNulos) Bytes([]byte) error        { return nil }
