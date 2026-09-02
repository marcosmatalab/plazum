package pantallas

import (
	"fmt"
	"math"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// LA PUERTA DEL CONTRASTE, medida sobre los DOS temas.
//
// # Por que hace falta, si axe-core ya es puerta bloqueante
//
// axe mide lo que la pagina PINTA en el momento de auditarla, y el auditor de
// .github/workflows/etapa2-accesibilidad.yml corre con el tema que trae el
// navegador sin configurar, que es el claro. O sea que el tema OSCURO no lo
// mira nadie: un par de colores que solo existe en el bloque de
// prefers-color-scheme: dark puede estar por debajo de 4.5:1 y la puerta de
// accesibilidad seguir en verde para siempre.
//
// Y hay una segunda mitad que axe tampoco alcanza en ninguno de los dos temas:
// un par declarado que NINGUNA pagina de la auditoria llega a pintar (el estado
// que solo sale con cierto corpus, el fondo de un aviso que solo aparece cuando
// algo se rompe). Ese par existe, se pintara algun dia en casa de alguien, y
// hasta entonces nadie lo ha medido.
//
// Esto mide los TOKENS, que es donde vive la decision, y los mide en los dos
// temas a la vez. No sustituye a axe: axe ve la pagina y esto ve la paleta.
//
// # La formula
//
// WCAG 2.1, criterio 1.4.3 (AA): (L1+0.05)/(L2+0.05) con L la luminancia
// relativa. El minimo es 4.5:1 para texto y 3:1 para componentes de interfaz
// (1.4.11). Cada par dice cual de las dos cosas es, porque un par sin decir
// para que se mide es un numero sin afirmacion.
const (
	umbralTexto = 4.5
	umbralUI    = 3.0
)

// par es una pareja de tokens que de verdad se pinta uno sobre el otro, con el
// sitio donde pasa. El "donde" no es documentacion: es lo que permite ir a
// mirarlo cuando el numero sale corto, y lo que obliga a borrar el par el dia
// que ese sitio deje de existir.
type par struct {
	frente, fondo string
	umbral        float64
	donde         string
}

// paresDeclarados son los pares que las reglas de plazum.css producen.
//
// SE ESCRIBEN A MANO Y NO SE DEDUCEN DE LA HOJA, y conviene decir por que: para
// deducirlos habria que resolver la cascada entera de CSS, o sea escribir medio
// navegador. Lo que si se comprueba mecanicamente es que todo token nombrado
// aqui EXISTE en la hoja y que todo token de color de la hoja aparece en algun
// par, en los dos sentidos: asi la lista no puede quedarse corta en silencio,
// que es la unica forma en que una lista escrita a mano hace dano.
var paresDeclarados = []par{
	{"tinta", "fondo", umbralTexto, "el cuerpo de la pagina"},
	{"tinta", "fondo-2", umbralTexto, "las tarjetas, la barra lateral y el pie"},
	{"tinta", "fondo-3", umbralTexto, "la cabecera de tabla y las fichas de filtro"},
	{"tinta-2", "fondo", umbralTexto, "las intros y las pistas"},
	{"tinta-2", "fondo-2", umbralTexto, "la atribucion del pie y las procedencias"},
	{"tinta-2", "fondo-3", umbralTexto, "las pastillas de procedencia"},
	{"acento", "fondo", umbralTexto, "los enlaces sobre el cuerpo"},
	{"acento", "fondo-2", umbralTexto, "los botones y los enlaces sobre tarjeta"},
	{"acento-tinta", "acento", umbralTexto, "el texto del boton elegido y del paso actual"},
	{"tinta", "acento-2", umbralTexto, "la entrada de menu activa"},
	{"acento", "acento-2", umbralTexto, "el boton bajo el puntero"},
	{"aplica", "aplica-fondo", umbralTexto, "la pastilla de estado 'te aplica'"},
	{"aplica", "fondo-2", umbralTexto, "la cifra grande de 'te aplica'"},
	{"pendiente", "pendiente-fondo", umbralTexto, "la pastilla de 'sin decidir'"},
	{"pendiente", "fondo-2", umbralTexto, "la cifra grande de 'sin decidir'"},
	{"no-aplica", "no-aplica-fondo", umbralTexto, "la pastilla de 'no te aplica'"},
	{"no-aplica", "fondo-2", umbralTexto, "la cifra grande de 'no te aplica'"},
	{"alerta", "alerta-fondo", umbralTexto, "el aviso de respuesta contradictoria"},
	{"alerta", "fondo-2", umbralTexto, "el veredicto roto del planificador"},
	{"borde", "fondo-2", umbralUI, "el contorno de la ficha de filtro y del numero del paso"},
	{"borde", "fondo-3", umbralUI, "la ficha de filtro bajo el puntero"},
}

// sinUmbralWCAG son los tokens de color que NO tienen un minimo de contraste que
// cumplir, con el motivo de cada uno escrito con nombre y apellidos.
//
// EXISTE PARA QUE LA EXCEPCION SE VEA, no para tapar nada: sin este mapa, un
// token que nadie mide se distingue de uno que nadie ha llegado a medir
// exactamente en nada. Es el mismo patron que clavesPropias en el inventario del
// catalogo.
//
// La regla que se aplica es la de WCAG 2.1 SC 1.4.11: el minimo de 3:1 alcanza a
// "la informacion visual necesaria para identificar componentes de interfaz y
// sus estados" y a "los objetos graficos necesarios para entender el contenido".
// Una linea que solo separa dos bloques de texto que se leen igual sin ella no
// es ninguna de las dos cosas. El contorno de una FICHA DE FILTRO si lo es, y por
// eso ese usa --borde y no --borde-suave.
var sinUmbralWCAG = map[string]string{
	"borde-suave": "es la linea de un pelo que separa tarjetas, filas de tabla y el pie. " +
		"No delimita ningun componente con el que se interactue ni sostiene ninguna " +
		"informacion: el texto de dentro se lee igual sin verla. Lo que SI delimita un " +
		"componente (la ficha de filtro, el numero del paso, la cabecera de tabla) usa " +
		"--borde, que si esta medido a 3:1. Separarlos en dos tokens fue la respuesta a " +
		"que esta puerta naciera roja: la alternativa era subir todas las lineas a 3:1, " +
		"que en una pagina con veinte tarjetas es una reja.",
}

func TestElContrasteSeCumpleEnLosDosTemas(t *testing.T) {
	temas := leerTemas(t)
	if len(temas) != 2 {
		t.Fatalf("se han leido %d temas de plazum.css y son dos, el claro y el oscuro. "+
			"Sin los dos, esta puerta mediria la mitad y no lo diria", len(temas))
	}
	medidos := 0
	for _, nombre := range []string{"claro", "oscuro"} {
		tema := temas[nombre]
		for _, p := range paresDeclarados {
			f, hayF := tema[p.frente]
			b, hayB := tema[p.fondo]
			if !hayF || !hayB {
				t.Errorf("tema %s: el par (--%s sobre --%s) nombra un token que la hoja no "+
					"declara. Donde se ve: %s", nombre, p.frente, p.fondo, p.donde)
				continue
			}
			medidos++
			if r := contraste(f, b); r < p.umbral {
				t.Errorf("tema %s: --%s sobre --%s da %.2f:1 y el minimo es %.1f:1.\n"+
					"  donde se ve: %s\n"+
					"  colores: %s sobre %s\n"+
					"  arreglo: el que cambia es el COLOR, nunca el umbral",
					nombre, p.frente, p.fondo, r, p.umbral, p.donde, hex(f), hex(b))
			}
		}
	}
	// Suelo: si el lector de la hoja dejara de encontrar tokens, el bucle de
	// arriba no mediria nada y saldria verde.
	if quiero := 2 * len(paresDeclarados); medidos != quiero {
		t.Fatalf("se han medido %d pares y habia %d que medir: esta puerta no ha llegado a "+
			"comprobar lo que dice", medidos, quiero)
	}
}

// Ningun token de color se queda sin medir. Es la mitad que evita que la lista
// escrita a mano se quede corta: un token nuevo entra en la hoja, no aparece en
// ningun par, y esto se pone rojo diciendo cual.
func TestNingunTokenDeColorSeQuedaSinMedir(t *testing.T) {
	temas := leerTemas(t)
	usados := map[string]bool{}
	for _, p := range paresDeclarados {
		usados[p.frente], usados[p.fondo] = true, true
	}
	var sueltos []string
	for token := range temas["claro"] {
		if !usados[token] && sinUmbralWCAG[token] == "" {
			sueltos = append(sueltos, token)
		}
	}
	sort.Strings(sueltos)
	if len(sueltos) > 0 {
		t.Errorf("estos tokens de color no aparecen en ningun par de paresDeclarados, asi "+
			"que nadie ha medido su contraste: %v.\n"+
			"  Arreglo: anadir el par con el sitio donde se pinta, o, si de verdad no tiene "+
			"minimo que cumplir, anadirlo a sinUmbralWCAG con el motivo escrito", sueltos)
	}
	// Y la excusa tampoco puede quedarse vieja: un token excusado que ya no
	// existe en la hoja es una excusa que sobrevive a lo que excusaba.
	for token, motivo := range sinUmbralWCAG {
		if motivo == "" {
			t.Errorf("--%s esta en sinUmbralWCAG sin motivo. Una excepcion sin motivo es una "+
				"medida que nadie tomo", token)
		}
		if _, ok := temas["claro"][token]; !ok {
			t.Errorf("--%s se excusa en sinUmbralWCAG y ya no esta en la hoja: sobra la "+
				"excusa", token)
		}
		if usados[token] {
			t.Errorf("--%s se excusa en sinUmbralWCAG y ademas aparece en paresDeclarados. "+
				"O se mide o se excusa, no las dos", token)
		}
	}
	// Y la direccion contraria: los dos temas declaran los MISMOS tokens. Uno
	// que solo exista en el claro se pintaria con el color del claro sobre el
	// fondo del oscuro, que es la peor combinacion posible y ademas invisible
	// para quien no use el tema oscuro.
	for token := range temas["claro"] {
		if _, ok := temas["oscuro"][token]; !ok {
			t.Errorf("--%s se declara en el tema claro y no en el oscuro: en oscuro se "+
				"pintaria con el color del claro sobre un fondo oscuro", token)
		}
	}
	for token := range temas["oscuro"] {
		if _, ok := temas["claro"][token]; !ok {
			t.Errorf("--%s solo se declara en el tema oscuro, asi que en el claro no "+
				"existe y lo que lo use se quedara sin color", token)
		}
	}
}

// CONTROL NEGATIVO: se demuestra que la medida muerde. Un par imposible (gris
// claro sobre blanco) tiene que salir por debajo de los dos umbrales y uno
// seguro (negro sobre blanco) muy por encima. Sin esto, un lector de hoja que
// devolviera siempre el mismo color daria verde en todo.
func TestElMedidorDeContrasteMuerde(t *testing.T) {
	blanco := [3]float64{1, 1, 1}
	grisClaro := [3]float64{0.8, 0.8, 0.8}
	negro := [3]float64{0, 0, 0}
	if r := contraste(grisClaro, blanco); r >= umbralUI {
		t.Errorf("gris claro sobre blanco da %.2f:1 y tenia que estar por debajo de %.1f",
			r, umbralUI)
	}
	if r := contraste(negro, blanco); r < 20 {
		t.Errorf("negro sobre blanco da %.2f:1 y son 21:1", r)
	}
	// Y es simetrico: el orden de los dos colores no cambia la razon.
	if a, b := contraste(negro, blanco), contraste(blanco, negro); math.Abs(a-b) > 1e-9 {
		t.Errorf("la medida no es simetrica: %.4f y %.4f", a, b)
	}
}

// ---------------------------------------------------------------------------
// Lector de la hoja
// ---------------------------------------------------------------------------

var (
	reBloqueRaiz  = regexp.MustCompile(`(?s):root\s*\{(.*?)\}`)
	reBloqueDark  = regexp.MustCompile(`(?s)@media\s*\(prefers-color-scheme:\s*dark\)\s*\{\s*:root\s*\{(.*?)\}`)
	reDeclaracion = regexp.MustCompile(`--([a-z0-9-]+)\s*:\s*(#[0-9a-fA-F]{3,8})\s*;`)
)

// leerTemas saca de plazum.css los tokens de color de los dos temas. El oscuro
// hereda del claro lo que no redefine, igual que hace el navegador.
func leerTemas(t *testing.T) map[string]map[string][3]float64 {
	t.Helper()
	b, err := os.ReadFile("estatico/plazum.css")
	if err != nil {
		t.Fatalf("no puedo leer la hoja de estilo: %v", err)
	}
	hoja := string(b)

	claro := map[string][3]float64{}
	m := reBloqueRaiz.FindStringSubmatch(hoja)
	if m == nil {
		t.Fatal("plazum.css no tiene bloque :root: el lector no ha encontrado el tema claro")
	}
	for _, d := range reDeclaracion.FindAllStringSubmatch(m[1], -1) {
		claro[d[1]] = aRGB(t, d[2])
	}
	if len(claro) < 5 {
		t.Fatalf("el tema claro trae %d colores: el lector no esta leyendo la hoja", len(claro))
	}

	oscuro := map[string][3]float64{}
	for k, v := range claro {
		oscuro[k] = v
	}
	m = reBloqueDark.FindStringSubmatch(hoja)
	if m == nil {
		t.Fatal("plazum.css no declara el tema oscuro (@media prefers-color-scheme: dark)")
	}
	n := 0
	for _, d := range reDeclaracion.FindAllStringSubmatch(m[1], -1) {
		oscuro[d[1]] = aRGB(t, d[2])
		n++
	}
	if n < 5 {
		t.Fatalf("el tema oscuro redefine %d colores: el lector no esta leyendo su bloque", n)
	}
	return map[string]map[string][3]float64{"claro": claro, "oscuro": oscuro}
}

// aRGB pasa un #rrggbb (o #rgb) a componentes 0..1.
func aRGB(t *testing.T, s string) [3]float64 {
	t.Helper()
	h := strings.TrimPrefix(s, "#")
	if len(h) == 3 {
		h = string([]byte{h[0], h[0], h[1], h[1], h[2], h[2]})
	}
	if len(h) != 6 {
		t.Fatalf("color %q: esta puerta solo entiende #rgb y #rrggbb, porque un color con "+
			"alfa se pinta sobre lo que tenga debajo y su contraste no se puede medir "+
			"mirando solo el token", s)
	}
	var out [3]float64
	for i := 0; i < 3; i++ {
		v, err := strconv.ParseUint(h[i*2:i*2+2], 16, 8)
		if err != nil {
			t.Fatalf("color %q: %v", s, err)
		}
		out[i] = float64(v) / 255
	}
	return out
}

func hex(c [3]float64) string {
	return fmt.Sprintf("#%02x%02x%02x", int(math.Round(c[0]*255)),
		int(math.Round(c[1]*255)), int(math.Round(c[2]*255)))
}

// luminancia relativa segun WCAG 2.1.
func luminancia(c [3]float64) float64 {
	lin := func(v float64) float64 {
		if v <= 0.04045 {
			return v / 12.92
		}
		return math.Pow((v+0.055)/1.055, 2.4)
	}
	return 0.2126*lin(c[0]) + 0.7152*lin(c[1]) + 0.0722*lin(c[2])
}

func contraste(a, b [3]float64) float64 {
	la, lb := luminancia(a), luminancia(b)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}
