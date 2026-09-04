package plazum

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// UNA FOTO VIEJA ES HONESTA SI NADIE LA USA COMO DATO VIVO.
//
// # El caso, con su cardinal
//
// `docs/instantanea.md` declara «2.331 casos de test» y la puerta ejecuta 2.516.
// Caduco durante el tramo que la escribio. No se parchea la celda, y no por
// pereza: lo prohibe el propio fichero, que lleva escrita la regla de
// `docs/diseno.md` §14, «se vuelve a hacer entera o no se hace; retocarle una
// celda la convertiria en una foto que finge estar viva».
//
// La decision (04-09-2026) es rehacerla entera en el tramo 4, con su fecha y su
// commit dentro. Y mientras tanto, ESTA PUERTA, que es lo que hace que la
// espera sea honesta: una instantanea vieja no hace dano si nadie la cita; hace
// dano cuando su numero viaja a un README, a un informe o a una web y ahi se lee
// como el dato de hoy.
//
// # Que prohibe exactamente, y que no
//
// Prohibe que un numero que solo vive en la instantanea aparezca en OTRO
// documento del repositorio. No prohibe que la instantanea lo tenga (es su
// oficio) ni que otro documento tenga el numero de HOY computado por su propia
// puerta: `230 relojes` sale de `estado_del_plan_test.go` y `51,4 %` de
// `marcos_v1_test.go`, y esos numeros estan vivos porque alguien los vigila.
//
// La diferencia es de donde sale el numero, no cual es. Por eso la lista de
// abajo son los que NADIE computa: si alguno pasa a tener puerta, sale de aqui
// en el mismo commit y se dice quien lo vigila ahora.
//
// # Por que se enumeran a mano y no se extraen del fichero
//
// Porque extraer «todos los numeros» de un documento de prosa da cientos de
// falsos: fechas, articulos, porcentajes de otra cosa. Una puerta que grita por
// todo se apaga. Se enumeran los que importan, con su valor, y si la
// instantanea cambia un valor sin actualizar esta lista la puerta lo dice: es
// el mismo trato que el bloque del marcador.

// cifrasSoloDeLaInstantanea son numeros que HOY solo viven ahi, con el motivo
// por el que citarlos fuera seria citar una foto vieja como dato de hoy.
var cifrasSoloDeLaInstantanea = []struct {
	Cifra  string
	PorQue string
}{
	{"2.331", "casos de test: la puerta ejecuta 2.516 desde el tramo 2, asi que " +
		"este numero ya es falso"},
	{"1.199", "casos de test de la foto ANTERIOR, citada dentro de la instantanea " +
		"para explicar su propio movimiento"},
}

// documentosQueNoPuedenCitarla: todo .md del repositorio menos la propia
// instantanea. Se recorre el arbol y no una lista escrita, porque una lista de
// documentos es exactamente lo que se queda vieja.
func documentosQueNoPuedenCitarla(t *testing.T) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.Walk(".", func(ruta string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			// Los worktrees de agentes y el directorio de git no son el arbol
			// del producto: contarlos daria hallazgos de trabajo ajeno.
			if n := info.Name(); n == ".git" || n == ".claude" || n == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(ruta, ".md") {
			return nil
		}
		limpia := filepath.ToSlash(ruta)
		if limpia == "docs/instantanea.md" {
			return nil
		}
		b, err := os.ReadFile(ruta) // #nosec G304 -- recorrido del propio repositorio
		if err != nil {
			return err
		}
		out[limpia] = string(b)
		return nil
	})
	if err != nil {
		t.Fatalf("recorriendo el arbol: %v", err)
	}
	if len(out) < 10 {
		t.Fatalf("solo he encontrado %d documentos .md fuera de la instantanea.\n"+
			"  El repositorio tiene decenas, asi que el recorrido esta roto y esta puerta "+
			"estaria dando verde sobre casi nada.", len(out))
	}
	return out
}

func TestNingunNumeroDeLaInstantaneaViejaSeCitaComoDatoVivo(t *testing.T) {
	// PRIMERO: QUE LA INSTANTANEA SIGA TENIENDO ESOS NUMEROS. Si los ha
	// perdido, esta puerta estaria prohibiendo citar algo que ya no existe, o
	// sea vigilando el vacio, y su verde no significaria nada.
	b, err := os.ReadFile("docs/instantanea.md") // #nosec G304 -- ruta constante
	if err != nil {
		t.Fatalf("no puedo leer la instantanea: %v", err)
	}
	foto := string(b)
	for _, c := range cifrasSoloDeLaInstantanea {
		if !strings.Contains(foto, c.Cifra) {
			t.Errorf("la instantanea ya no trae %q.\n"+
				"  O se ha rehecho entera (y entonces esta lista se rehace con ella, en el "+
				"mismo commit, diciendo que numeros la sustituyen), o alguien le ha "+
				"retocado una celda, que es justo lo que su propia regla prohibe.", c.Cifra)
		}
	}

	docs := documentosQueNoPuedenCitarla(t)
	for ruta, cuerpo := range docs {
		for _, c := range cifrasSoloDeLaInstantanea {
			// Se busca la cifra como PALABRA, para que 2.331 no case dentro de
			// 12.3311 ni de una fecha.
			re := regexp.MustCompile(`(^|[^0-9.,])` + regexp.QuoteMeta(c.Cifra) + `([^0-9]|$)`)
			if !re.MatchString(cuerpo) {
				continue
			}
			t.Errorf("%s cita %q, que es un numero de docs/instantanea.md.\n"+
				"  %s\n"+
				"  UNA FOTO VIEJA ES HONESTA SI NADIE LA USA COMO DATO VIVO. La instantanea "+
				"se rehace ENTERA (su propia regla lo dice) y hasta entonces sus numeros no "+
				"salen de ella.\n"+
				"  Si lo que quieres es el numero de HOY, sacalo de la puerta que lo "+
				"computa, no de la foto.", ruta, c.Cifra, c.PorQue)
		}
	}
	t.Logf("%d documentos revisados; %d cifras de la instantanea vigiladas",
		len(docs), len(cifrasSoloDeLaInstantanea))
}
