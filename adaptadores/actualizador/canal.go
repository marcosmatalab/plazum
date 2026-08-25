package actualizador

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Canal es de donde salen las versiones. Se separa del actualizador a proposito:
// la logica que importa (punto de retorno, verificacion, vuelta atras) es la
// misma tanto si la release viene de un directorio montado, de un HTTP firmado o
// de un espejo interno del cliente, y esa logica es la que hay que probar. Un
// canal es una tarde; un rollback mal hecho es un incumplimiento.
type Canal interface {
	// Catalogo devuelve las versiones que ofrece, de la mas nueva a la mas
	// vieja. La ordenacion la decide el canal, no el actualizador: el canal es
	// quien sabe que significa "mas nueva" en su esquema de versiones.
	Catalogo(ctx context.Context) ([]Version, error)
	// Traer devuelve el contenido de un fichero de una version. Los bytes
	// llegan SIN verificar: el actualizador compara su digest contra el que
	// declara el catalogo, y por eso el canal no puede colar otra cosa.
	Traer(ctx context.Context, version, fichero string) ([]byte, error)
}

// Version es una release ofrecida por el canal.
type Version struct {
	// Version es el identificador, tal cual se teclea en `dutiq update`.
	Version string `json:"version"`
	// Notas es lo que cambia, en texto plano. Se ensena antes de aplicar:
	// nadie deberia actualizar un producto que vigila plazos legales sin
	// poder leer que cambia.
	Notas string `json:"notas"`
	// Ficheros mapea la ruta RELATIVA dentro de la instalacion al digest
	// SHA-256 en hexadecimal de su contenido.
	//
	// Va por fichero y no un digest del conjunto porque asi el punto de
	// retorno puede verificarse fichero a fichero, y una copia truncada por
	// disco lleno se caza en el fichero concreto en vez de como "algo no
	// cuadra".
	Ficheros map[string]string `json:"ficheros"`
}

// Errores del canal, como centinelas.
var (
	// ErrVersionDesconocida: la version pedida no esta en el catalogo.
	ErrVersionDesconocida = errors.New("la version pedida no existe en el canal")
	// ErrCatalogoIlegible: el canal contesta algo que no es un catalogo.
	ErrCatalogoIlegible = errors.New("el catalogo del canal no se puede leer")
	// ErrRutaInsegura: una version declara un fichero fuera de la instalacion.
	ErrRutaInsegura = errors.New("ruta de fichero insegura en el catalogo")
)

// NombreDelCatalogo es el fichero que un canal de directorio tiene que traer.
const NombreDelCatalogo = "catalogo.json"

// CanalDirectorio es un canal servido desde un directorio local.
//
// Es el canal de arranque y el que se usa en las pruebas, y no es un juguete:
// es exactamente la forma que tiene una actualizacion en una instalacion sin
// salida a internet, que en este mercado son mas de las que parece. Un canal
// HTTP firmado se anade en su etapa implementando esta misma interfaz, sin
// tocar nada de la vuelta atras.
//
//	<Dir>/catalogo.json          lista de Version, de mas nueva a mas vieja
//	<Dir>/<version>/<fichero>    el contenido de cada fichero de esa version
type CanalDirectorio struct {
	Dir string
}

// Catalogo lee <Dir>/catalogo.json.
func (c CanalDirectorio) Catalogo(ctx context.Context) ([]Version, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ruta := filepath.Join(c.Dir, NombreDelCatalogo)
	b, err := os.ReadFile(ruta) // #nosec G304 -- el directorio del canal lo elige el operador
	if err != nil {
		return nil, fmt.Errorf("%w: no puedo leer %s: %w; un canal de directorio es un "+
			"directorio con %s dentro y un subdirectorio por version",
			ErrCatalogoIlegible, ruta, err, NombreDelCatalogo)
	}
	var vs []Version
	if err := json.Unmarshal(b, &vs); err != nil {
		return nil, fmt.Errorf("%w: %s no es una lista de versiones en JSON: %w",
			ErrCatalogoIlegible, ruta, err)
	}
	for i, v := range vs {
		if strings.TrimSpace(v.Version) == "" {
			return nil, fmt.Errorf("%w: la entrada %d de %s no tiene version",
				ErrCatalogoIlegible, i, ruta)
		}
		if err := ComprobarNombreDeVersion(v.Version); err != nil {
			return nil, fmt.Errorf("%w: %w", ErrCatalogoIlegible, err)
		}
		for f := range v.Ficheros {
			if err := comprobarRutaRelativa(f); err != nil {
				return nil, fmt.Errorf("%w: version %s: %w", ErrRutaInsegura, v.Version, err)
			}
		}
	}
	return vs, nil
}

// Traer lee <Dir>/<version>/<fichero>.
func (c CanalDirectorio) Traer(ctx context.Context, version, fichero string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := ComprobarNombreDeVersion(version); err != nil {
		return nil, err
	}
	if err := comprobarRutaRelativa(fichero); err != nil {
		return nil, err
	}
	ruta := filepath.Join(c.Dir, version, filepath.FromSlash(fichero))
	b, err := os.ReadFile(ruta) // #nosec G304 -- version y fichero pasan por las comprobaciones de arriba
	if err != nil {
		return nil, fmt.Errorf("no puedo traer %s de la version %s: %w. El catalogo la declara, "+
			"asi que el canal esta incompleto: revisa %s", fichero, version, err, c.Dir)
	}
	return b, nil
}

// EscribirCatalogo deja un catalogo en el directorio. Existe para construir un
// canal desde una prueba o desde una release, sin tener que serializar a mano.
func EscribirCatalogo(dir string, vs []Version) error {
	b, err := json.MarshalIndent(vs, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, NombreDelCatalogo), b, 0o600)
}

// masNueva devuelve la primera version del catalogo, que por contrato es la mas
// nueva. Aqui NO se reordena: comparar versiones es cosa del canal, que es
// quien conoce su esquema, y un actualizador que se pusiera a interpretar
// "v0.10" contra "v0.9" acabaria instalando la vieja creyendo que es la nueva.
func masNueva(vs []Version) (Version, bool) {
	if len(vs) == 0 {
		return Version{}, false
	}
	return vs[0], true
}

// buscarVersion localiza una version por nombre y, si no esta, devuelve un error
// que dice cuales hay. Un "no existe" a secas obliga al operador a adivinar.
func buscarVersion(vs []Version, nombre string) (Version, error) {
	for _, v := range vs {
		if v.Version == nombre {
			return v, nil
		}
	}
	nombres := make([]string, 0, len(vs))
	for _, v := range vs {
		nombres = append(nombres, v.Version)
	}
	sort.Strings(nombres)
	if len(nombres) == 0 {
		return Version{}, fmt.Errorf("%w: %q, y el canal no ofrece ninguna version. "+
			"Comprueba que apuntas al canal correcto", ErrVersionDesconocida, nombre)
	}
	return Version{}, fmt.Errorf("%w: %q. El canal ofrece %v", ErrVersionDesconocida, nombre, nombres)
}
