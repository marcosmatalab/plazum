package main

import (
	"bytes"
	"os"
	"os/exec"
	"runtime/debug"
	"strings"
	"testing"
)

func TestLaVersionSaleDeLaReleaseDeGoOEsDevel(t *testing.T) {
	vcs := []debug.BuildSetting{
		{Key: "vcs.revision", Value: "0123456789abcdef0123456789abcdef01234567"},
		{Key: "vcs.time", Value: "2026-09-23T12:00:00Z"},
		{Key: "vcs.modified", Value: "false"},
	}
	casos := []struct {
		nombre    string
		inyectada string
		bi        *debug.BuildInfo
		quiero    string
	}{
		{
			nombre: "sin informacion de construccion ni release: devel, y ningun commit inventado",
			quiero: "plazum (devel)\n",
		},
		{
			nombre: "go install desde un tag: la version que grabo Go, sin commit",
			bi:     &debug.BuildInfo{GoVersion: "go1.24.7", Main: debug.Module{Version: "v0.2.0"}},
			quiero: "plazum v0.2.0\n  go       go1.24.7\n",
		},
		{
			nombre:    "release: manda lo inyectado aunque Go diga devel",
			inyectada: "v0.2.0",
			bi:        &debug.BuildInfo{GoVersion: "go1.24.7", Main: debug.Module{Version: "(devel)"}, Settings: vcs},
			quiero: "plazum v0.2.0\n" +
				"  commit   0123456789abcdef0123456789abcdef01234567 (2026-09-23T12:00:00Z)\n" +
				"  go       go1.24.7\n",
		},
		{
			nombre: "compilado con cambios sin commitear: se dice",
			bi: &debug.BuildInfo{GoVersion: "go1.24.7", Main: debug.Module{Version: "(devel)"},
				Settings: []debug.BuildSetting{{Key: "vcs.revision", Value: "abc"}, {Key: "vcs.modified", Value: "true"}}},
			quiero: "plazum (devel)\n  commit   abc, con cambios sin commitear\n  go       go1.24.7\n",
		},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			var b bytes.Buffer
			escribirVersion(&b, leerVersion(c.inyectada, c.bi))
			got := b.String()
			// La linea del sistema depende de la maquina que corre el test:
			// se comprueba que esta y que es la ultima, no su valor.
			i := strings.LastIndex(got, "  sistema  ")
			if i < 0 || !strings.HasSuffix(got, "\n") || strings.Count(got[i:], "\n") != 1 {
				t.Fatalf("la salida no acaba en la linea del sistema:\n%s", got)
			}
			if c.bi == nil {
				// Sin informacion de construccion, la version de Go es la del
				// runtime; se comprueba la forma y no el valor.
				if !strings.HasPrefix(got, c.quiero) || !strings.Contains(got, "  go       go") {
					t.Errorf("salida:\n%s\nquiero que empiece por:\n%s", got, c.quiero)
				}
				if strings.Contains(got, "commit") {
					t.Errorf("sin informacion de construccion ha salido un commit:\n%s", got)
				}
				return
			}
			if got[:i] != c.quiero {
				t.Errorf("salida:\n%s\nquiero:\n%s", got[:i], c.quiero)
			}
		})
	}
}

// LA ORDEN DE VERDAD, por la linea de ordenes. Hasta el 23-09-2026
// `plazum --version` caia en la ayuda y salia con codigo 2: esto lo comprueba
// relanzando el binario de test como si fuera plazum.
func TestVersionYGuionGuionVersionContestanLaVersion(t *testing.T) {
	if os.Getenv("PLAZUM_TEST_COMO_MAIN") == "1" {
		os.Args = append([]string{"plazum"}, strings.Fields(os.Getenv("PLAZUM_TEST_ARGS"))...)
		main()
		return
	}
	for _, args := range []string{"version", "--version"} {
		cmd := exec.Command(os.Args[0], "-test.run=^TestVersionYGuionGuionVersionContestanLaVersion$") // #nosec G204 -- relanza el propio binario de test
		cmd.Env = append(os.Environ(), "PLAZUM_TEST_COMO_MAIN=1", "PLAZUM_TEST_ARGS="+args)
		var salida, errores bytes.Buffer
		cmd.Stdout, cmd.Stderr = &salida, &errores
		if err := cmd.Run(); err != nil {
			t.Errorf("plazum %s ha salido con error (%v).\nsalida:\n%s\nerrores:\n%s",
				args, err, salida.String(), errores.String())
			continue
		}
		primera, _, _ := strings.Cut(salida.String(), "\n")
		if !strings.HasPrefix(primera, "plazum ") || strings.Contains(salida.String(), "empieza por aqui") {
			t.Errorf("plazum %s no ha contestado la version. Primera linea: %q", args, primera)
		}
	}
}

func TestVersionNoAdmiteArgumentos(t *testing.T) {
	var salida, errores bytes.Buffer
	if c := cmdVersion([]string{"extra"}, &salida, &errores); c != 2 {
		t.Errorf("version con un argumento de mas sale con %d y tiene que ser 2", c)
	}
	if c := cmdVersion(nil, &salida, &errores); c != 0 {
		t.Errorf("version sin argumentos sale con %d: %s", c, errores.String())
	}
}
