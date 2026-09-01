package censo

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// opcionesDePrueba son unas opciones completas. Se parte de aqui y cada test
// estropea lo suyo, en vez de escribir el cero y que el fallo sea otro.
func opcionesDePrueba() Opciones {
	return Opciones{
		Sistema:   "erp",
		Fuente:    "export de usuarios",
		Quien:     "u-042",
		Tomada:    time.Date(2026, 9, 1, 9, 0, 0, 0, time.UTC),
		Retencion: "12 meses tras el cierre de la campana",
		Columnas:  ColumnasHabituales(),
	}
}

func tomar(t *testing.T, datos []byte) Instantanea {
	t.Helper()
	ins, err := Tomar(datos, opcionesDePrueba())
	if err != nil {
		t.Fatalf("no carga: %v", err)
	}
	return ins
}

// EL BOM. Es el primer minuto de quien sube un fichero de Excel, y sin esto el
// error que recibe es "falta la columna usuario" SOBRE UN FICHERO QUE LA TIENE.
// Un error que miente cuesta mas que un error feo: manda a quien lo lee a
// arreglar lo que no esta roto.
func TestElBOMNoSeComeElNombreDeLaPrimeraColumna(t *testing.T) {
	con := append([]byte{0xEF, 0xBB, 0xBF}, []byte("usuario;permiso\nana;admin\n")...)
	ins := tomar(t, con)
	if !ins.Notas.QuitadoElBOM {
		t.Error("no se ha dicho en las notas que el fichero traia BOM")
	}
	if len(ins.Filas) != 1 || ins.Filas[0].Cuenta != "ana" {
		t.Fatalf("filas mal leidas: %+v", ins.Filas)
	}
	// Y EL HASH SIGUE SIENDO EL DE LOS BYTES SUBIDOS: quitar el BOM es una
	// decision de lectura, no una edicion del fichero. Si el hash fuera del
	// texto normalizado, dos ficheros distintos tendrian la misma identidad.
	sin := tomar(t, []byte("usuario;permiso\nana;admin\n"))
	if ins.Hash == sin.Hash {
		t.Error("el fichero con BOM y el fichero sin BOM tienen el mismo hash: entonces el hash no " +
			"es de los bytes que subio la persona, y la revision no se puede reproducir")
	}
}

// cp1252. El caso medido: csv NO da ningun error y la cadena invalida viaja
// entera. Cuando el acento esta en el IDENTIFICADOR, lo corrompido en silencio
// es la clave por la que se empareja todo (invariante 7).
func TestUnFicheroEnCP1252NoCorrompeLaIdentidad(t *testing.T) {
	// "usuario;permiso\nmartínez;admin\nO'Brien;lector\n" en cp1252: la i
	// acentuada es 0xED y la comilla tipografica 0x92.
	datos := []byte("usuario;permiso\nmart\xEDnez;admin\nO\x92Brien;lector\n")
	ins := tomar(t, datos)
	if !strings.Contains(ins.Notas.Codificacion, "cp1252") {
		t.Fatalf("no se ha dicho que codificacion se uso: %q", ins.Notas.Codificacion)
	}
	if len(ins.Filas) != 2 {
		t.Fatalf("filas: %+v", ins.Filas)
	}
	if ins.Filas[0].Cuenta != "martínez" {
		t.Errorf("el identificador ha salido %q y tenia que salir %q", ins.Filas[0].Cuenta, "martínez")
	}
	if ins.Filas[1].Cuenta != "O’Brien" {
		t.Errorf("el identificador ha salido %q y tenia que salir %q", ins.Filas[1].Cuenta, "O’Brien")
	}
	// Y LA CLAVE ES UTILIZABLE: si la identidad viajara corrupta, esto seria una
	// cadena que no se puede comparar con la que traiga SCIM manana.
	for _, f := range ins.Filas {
		for _, r := range f.Clave() {
			if r == '\uFFFD' {
				t.Errorf("la clave %q lleva el caracter de reemplazo: la identidad se ha perdido "+
					"por el camino", f.Clave())
			}
		}
	}
}

// LA FILA DESIGUAL. Medido: con el FieldsPerRecord por defecto, ReadAll devuelve
// CERO filas y un error, o sea que una sola fila mala se lleva el fichero
// entero. Aqui es un cubo y las demas siguen.
func TestUnaFilaDesigualNoSeLlevaElFicheroPorDelante(t *testing.T) {
	ins := tomar(t, []byte("usuario;permiso\nana;admin;sobra\nluis;lector\n"))
	if len(ins.Filas) != 2 {
		t.Fatalf("se esperaban las dos filas leidas (la desigual tiene columnas de sobra, no de "+
			"menos): %+v / ilegibles %+v", ins.Filas, ins.Ilegibles)
	}
	if !ins.Cuadra() {
		t.Fatalf("no cuadra: %v", ins.Cuenta())
	}
}

// EL CASO QUE DECIDE EL DISENO, y el que se midio antes de escribir nada: una
// comilla sin cerrar en la linea 3 se lleva por delante la 4 y la 5, el lector
// devuelve un error y luego dice "fin".
//
// Sin el rango del error, esas dos personas no estarian en ningun cubo y nadie
// las echaria de menos. Con el rango, salen dichas: lineas 3 a 5, ilegibles, con
// su motivo.
func TestUnaComillaSinCerrarNoHaceDesaparecerANadieEnSilencio(t *testing.T) {
	datos := []byte("usuario;permiso\nana;admin\n\"luis;lector\nmar;admin\npepa;lector\n")
	ins := tomar(t, datos)

	if len(ins.Filas) != 1 || ins.Filas[0].Cuenta != "ana" {
		t.Fatalf("filas legibles: %+v", ins.Filas)
	}
	if len(ins.Ilegibles) != 1 {
		t.Fatalf("ilegibles: %+v", ins.Ilegibles)
	}
	il := ins.Ilegibles[0]
	if il.Desde != 3 || il.Hasta != 5 {
		t.Errorf("el rango ilegible es %d-%d y las lineas que se traga la comilla son la 3, la 4 y "+
			"la 5. Un rango corto deja personas fuera de todo cubo", il.Desde, il.Hasta)
	}
	if !strings.Contains(il.Motivo, "comilla") {
		t.Errorf("el motivo no dice que fue una comilla: %q", il.Motivo)
	}
	// LA LEY DE CONSERVACION: cuatro lineas de datos, cuatro explicadas.
	if !ins.Cuadra() {
		t.Fatalf("no cuadra y tendria que cuadrar: %v", ins.Cuenta())
	}
	if ins.LineasDeDatos != 4 {
		t.Errorf("lineas de datos = %d, y el fichero tiene 4", ins.LineasDeDatos)
	}
}

// Y CUANDO NO SE PUEDE EXPLICAR UNA LINEA, NO SE CARGA.
//
// Control positivo del descuadre, con dato sintetico: un campo con un salto de
// linea dentro es CSV legitimo, el lector lo lee como UN registro, y el contador
// ciego ve DOS lineas. Nadie puede decir si sobra una linea o falta una persona,
// asi que se para. Es el falso positivo que este diseno elige a proposito, y
// tiene que estar recorrido por un test o no se sabe que existe.
func TestSiUnaLineaNoTieneCuboLaInstantaneaNoSeCrea(t *testing.T) {
	datos := []byte("usuario;permiso\n\"ana\nmaria\";admin\nluis;lector\n")
	_, err := Tomar(datos, opcionesDePrueba())
	if err == nil {
		t.Fatal("ha cargado un fichero en el que una linea no cae en ningun cubo. Ese es " +
			"exactamente el camino por el que una persona desaparece de la revision sin que " +
			"nadie lo note")
	}
	if !errors.Is(err, ErrDescuadre) {
		t.Fatalf("el centinela no es ErrDescuadre: %v", err)
	}
	// Y EL ERROR TIENE QUE SER ACCIONABLE: quien lo recibe no ha escrito este
	// paquete y tiene que saber que mirar.
	for _, quiero := range []string{"comilla sin cerrar", "salto de linea", "NO se ha creado"} {
		if !strings.Contains(err.Error(), quiero) {
			t.Errorf("el error no dice %q:\n%v", quiero, err)
		}
	}
}

// EL SEPARADOR SE DEDUCE Y SE DICE. Medido: un fichero de ';' leido con ',' no
// da ningun error, da UNA columna llamada "usuario;permiso". Una deduccion que
// no se cuenta es una suposicion que nadie puede revisar.
func TestElSeparadorSeDeduceYSeDiceEnLasNotas(t *testing.T) {
	casos := []struct {
		nombre, datos, separador string
	}{
		{"punto y coma", "usuario;permiso\nana;admin\n", "punto y coma"},
		{"coma", "usuario,permiso\nana,admin\n", "coma"},
		{"tabulador", "usuario\tpermiso\nana\tadmin\n", "tabulador"},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			ins := tomar(t, []byte(c.datos))
			if ins.Notas.Separador != c.separador {
				t.Fatalf("separador deducido %q, esperado %q", ins.Notas.Separador, c.separador)
			}
			if len(ins.Filas) != 1 || ins.Filas[0].Cuenta != "ana" || ins.Filas[0].Permiso != "admin" {
				t.Fatalf("filas: %+v", ins.Filas)
			}
		})
	}
}

// UN EMPATE ENTRE SEPARADORES SE DICE. Control positivo con dato sintetico: sin
// una cabecera que empate de verdad, la rama del aviso no la recorre nadie y una
// mutacion la deja verde porque no hay nada que romper (es lo de M47).
//
// El caso: una cabecera con un ";" y una "," de verdad. Se elige el punto y coma
// por el orden de la lista, que es una eleccion, y una eleccion se cuenta.
func TestUnEmpateEntreSeparadoresSeDiceEnLasNotas(t *testing.T) {
	ins := tomar(t, []byte("usuario;permiso,nivel\nana;admin,alto\n"))
	if ins.Notas.Separador != "punto y coma" {
		t.Fatalf("separador: %q", ins.Notas.Separador)
	}
	hay := false
	for _, a := range ins.Notas.Avisos {
		if strings.Contains(a, "tantos punto y coma como coma") {
			hay = true
		}
	}
	if !hay {
		t.Fatalf("el empate no se cuenta en ningun sitio: %v.\n"+
			"  Una deduccion que gana por el orden de una lista es una eleccion, y una eleccion "+
			"que nadie ve no se puede revisar", ins.Notas.Avisos)
	}
}

// DOS COLUMNAS PARA EL MISMO PAPEL SE PARA, no se elige. Adivinar la columna de
// identidad es adivinar a quien se esta revisando.
func TestDosColumnasParaElMismoPapelSeParanEnVezDeAdivinar(t *testing.T) {
	_, err := Tomar([]byte("usuario;login;permiso\nana;ana@x;admin\n"), opcionesDePrueba())
	if err == nil {
		t.Fatal("ha elegido una de las dos columnas de identidad por su cuenta")
	}
	if !errors.Is(err, ErrColumnaAmbigua) {
		t.Fatalf("centinela: %v", err)
	}
	if !strings.Contains(err.Error(), "usuario") || !strings.Contains(err.Error(), "login") {
		t.Errorf("el error no nombra las dos columnas que compiten:\n%v", err)
	}
}

// SIN COLUMNA DE IDENTIDAD NO HAY NADA QUE REVISAR, y el error dice que trae el
// fichero y que se acepta. Un "columna no encontrada" a secas obliga a leer el
// codigo fuente para saber que nombres valen.
func TestSinColumnaDeIdentidadElErrorDiceQueHayYQueSeAcepta(t *testing.T) {
	_, err := Tomar([]byte("empleado;permiso\nana;admin\n"), opcionesDePrueba())
	if !errors.Is(err, ErrSinColumnaDeCuenta) {
		t.Fatalf("centinela: %v", err)
	}
	for _, quiero := range []string{`"empleado"`, `"usuario"`, "Columnas.Cuenta"} {
		if !strings.Contains(err.Error(), quiero) {
			t.Errorf("el error no dice %q:\n%v", quiero, err)
		}
	}
}

// EL VALOR CERO DE Opciones ESTA PROHIBIDO (invariante 8), y se recorren LAS DOS
// FORMAS DE LA NADA: nil y vacio-presente. En Go son distintas y la que sale por
// descuido es la nil.
func TestElValorCeroDeOpcionesEstaProhibidoEnSusDosFormas(t *testing.T) {
	datos := []byte("usuario;permiso\nana;admin\n")
	casos := []struct {
		nombre string
		o      Opciones
		falta  string
	}{
		{"el cero entero", Opciones{}, "Sistema"},
		{"sin quien lo sube", conSinQuien(), "Quien"},
		{"sin instante", conSinInstante(), "Tomada"},
		{"sin retencion declarada", conSinRetencion(), "Retencion"},
		{"columnas nil", conColumnas(Columnas{}), "Columnas.Cuenta"},
		{"columnas vacias pero presentes", conColumnas(Columnas{Cuenta: []string{}}), "Columnas.Cuenta"},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			_, err := Tomar(datos, c.o)
			if err == nil {
				t.Fatalf("se ha sellado una instantanea sin %s", c.falta)
			}
			if !errors.Is(err, ErrOpciones) {
				t.Fatalf("centinela: %v", err)
			}
			if !strings.Contains(err.Error(), c.falta) {
				t.Errorf("el error no dice que falta %s:\n%v", c.falta, err)
			}
		})
	}
}

func conSinQuien() Opciones     { o := opcionesDePrueba(); o.Quien = "  "; return o }
func conSinInstante() Opciones  { o := opcionesDePrueba(); o.Tomada = time.Time{}; return o }
func conSinRetencion() Opciones { o := opcionesDePrueba(); o.Retencion = ""; return o }
func conColumnas(c Columnas) Opciones {
	o := opcionesDePrueba()
	o.Columnas = c
	return o
}

// EL HASH ES DE LOS BYTES. Es lo que permite a un tercero repetir la lectura y
// comprobar que la revision se hizo sobre ese fichero y no sobre otro.
func TestElHashEsDeLosBytesYNoDeLoQueElParserEntendio(t *testing.T) {
	a := tomar(t, []byte("usuario;permiso\nana;admin\n"))
	b := tomar(t, []byte("usuario;permiso\nana;admin\n"))
	if a.Hash != b.Hash {
		t.Fatal("dos lecturas de los mismos bytes dan hashes distintos")
	}
	// Un cambio que NO cambia las filas leidas si cambia el hash: un espacio de
	// mas al final. Si el hash fuera del resultado, dos ficheros distintos
	// tendrian la misma identidad y "revisado sobre este fichero" seria falso.
	c := tomar(t, []byte("usuario;permiso\nana;admin \n"))
	if c.Filas[0].Permiso != "admin" {
		t.Fatalf("el espacio no se recorta: %q", c.Filas[0].Permiso)
	}
	if a.Hash == c.Hash {
		t.Error("dos ficheros distintos comparten hash")
	}
	// Y LA LECTURA ES DETERMINISTA: mismo hash, mismas filas y en el mismo
	// orden. Sin esto, el hash no es un asa de nada.
	for i := range a.Filas {
		if a.Filas[i] != b.Filas[i] {
			t.Fatalf("fila %d distinta entre dos lecturas iguales: %+v / %+v", i, a.Filas[i], b.Filas[i])
		}
	}
}

// MINIMIZACION: lo que no esta declarado no se guarda, Y SE DICE CUAL. Un
// descarte mudo de columnas es el mismo fallo que un descarte mudo de filas
// mirado desde el otro eje.
func TestLasColumnasNoDeclaradasSeDescartanYSeDicen(t *testing.T) {
	datos := []byte("usuario;permiso;dni;telefono;departamento\nana;admin;12345678Z;600000000;IT\n")
	ins := tomar(t, datos)
	if len(ins.Filas) != 1 {
		t.Fatalf("filas: %+v", ins.Filas)
	}
	f := ins.Filas[0]
	// El DNI y el telefono NO estan en ningun campo de la fila. Es lo que hace
	// que esto sea minimizacion y no una promesa.
	for _, prohibido := range []string{"12345678Z", "600000000"} {
		for _, campo := range []string{f.Cuenta, f.Permiso, f.Rotulo, f.Clave()} {
			if strings.Contains(campo, prohibido) {
				t.Errorf("%q ha entrado en la instantanea a traves de %q", prohibido, campo)
			}
		}
	}
	quiero := map[string]bool{"dni": true, "telefono": true, "departamento": true}
	for _, c := range ins.Notas.ColumnasIgnoradas {
		delete(quiero, c)
	}
	if len(quiero) != 0 {
		t.Errorf("no se dice que se han descartado %v; descartadas: %v", quiero, ins.Notas.ColumnasIgnoradas)
	}
}

// EL ROTULO NO IDENTIFICA. Dos personas se llaman igual y una persona cambia de
// apellido: si la identidad colgara del nombre, la decision de una campana se
// mudaria de cuenta sola.
func TestElRotuloNoEntraEnLaIdentidad(t *testing.T) {
	ins := tomar(t, []byte("usuario;nombre;permiso\nu1;Ana Perez;admin\nu2;Ana Perez;admin\n"))
	if len(ins.Filas) != 2 {
		t.Fatalf("dos cuentas distintas con el mismo nombre tienen que ser dos filas: %+v", ins.Filas)
	}
	if ins.Filas[0].Clave() == ins.Filas[1].Clave() {
		t.Fatal("dos cuentas distintas comparten clave")
	}
	// Y al reves: el mismo identificador con otro rotulo es la misma fila.
	a := Fila{Sistema: "erp", Cuenta: "u1", Permiso: "admin", Rotulo: "Ana Perez"}
	b := Fila{Sistema: "erp", Cuenta: "u1", Permiso: "admin", Rotulo: "Ana Perez-Gomez"}
	if a.Clave() != b.Clave() {
		t.Error("cambiar el rotulo ha cambiado la identidad de la fila")
	}
}

// UN CARACTER DE CONTROL EN EL IDENTIFICADOR NO ES UN IDENTIFICADOR. Medido: csv
// deja pasar un byte nulo sin decir nada, y esa cadena acaba en la clave por la
// que se empareja todo, en los logs y en la pantalla.
func TestUnCaracterDeControlEnElIdentificadorEsUnaFilaIlegible(t *testing.T) {
	ins := tomar(t, []byte("usuario;permiso\nan\x00a;admin\nluis;lector\n"))
	if len(ins.Filas) != 1 || ins.Filas[0].Cuenta != "luis" {
		t.Fatalf("filas: %+v", ins.Filas)
	}
	if len(ins.Ilegibles) != 1 || !strings.Contains(ins.Ilegibles[0].Motivo, "control") {
		t.Fatalf("ilegibles: %+v", ins.Ilegibles)
	}
	if !ins.Cuadra() {
		t.Fatalf("no cuadra: %v", ins.Cuenta())
	}
}

// LA FILA REPETIDA ES UN CUBO PROPIO. No se descarta en silencio (perderia la
// cuenta) ni se revisa dos veces (dos decisiones contrarias sobre el mismo
// acceso es como se consigue un expediente que se contradice).
func TestLaFilaRepetidaSeCuentaYNoSeRevisaDosVeces(t *testing.T) {
	ins := tomar(t, []byte("usuario;permiso\nana;admin\nana;admin\nana;lector\n"))
	if len(ins.Filas) != 2 {
		t.Fatalf("filas unicas: %+v", ins.Filas)
	}
	if len(ins.Duplicadas) != 1 {
		t.Fatalf("duplicadas: %+v", ins.Duplicadas)
	}
	if ins.Duplicadas[0].Linea != 3 || ins.Duplicadas[0].Primera != 2 {
		t.Errorf("la duplicada tiene que decir donde esta y donde estaba la primera: %+v",
			ins.Duplicadas[0])
	}
	if !ins.Cuadra() {
		t.Fatalf("no cuadra: %v", ins.Cuenta())
	}
}

// UTF-16 SE RECHAZA CON INSTRUCCIONES, no se adivina. Es la opcion "Texto
// Unicode" de Excel, que ademas separa por tabuladores: dos suposiciones
// encadenadas dan un censo plausible y equivocado.
func TestUTF16SeRechazaDiciendoQueHacer(t *testing.T) {
	// "usuario" en UTF-16LE con su BOM.
	datos := []byte{0xFF, 0xFE, 'u', 0, 's', 0, 'u', 0, 'a', 0, 'r', 0, 'i', 0, 'o', 0}
	_, err := Tomar(datos, opcionesDePrueba())
	if !errors.Is(err, ErrCodificacion) {
		t.Fatalf("centinela: %v", err)
	}
	if !strings.Contains(err.Error(), "CSV UTF-8") {
		t.Errorf("el error no dice como arreglarlo:\n%v", err)
	}
}

// VARIOS PERMISOS EN UNA CELDA: SE AVISA Y NO SE PARTE. Partirlo seria adivinar,
// y un permiso llamado "lectura, escritura" existe. Pero callarlo dejaria un
// acceso que no se puede revocar por separado.
func TestVariosPermisosEnUnaCeldaSeAvisanYNoSePartenSolos(t *testing.T) {
	ins := tomar(t, []byte("usuario;permiso\nana;admin,lector\n"))
	if len(ins.Filas) != 1 || ins.Filas[0].Permiso != "admin,lector" {
		t.Fatalf("se ha partido el permiso por su cuenta: %+v", ins.Filas)
	}
	if len(ins.Notas.Avisos) != 1 || !strings.Contains(ins.Notas.Avisos[0], "cada uno tiene que ir") {
		t.Fatalf("no se avisa de la celda con varios permisos: %v", ins.Notas.Avisos)
	}
}

// LAS LINEAS EN BLANCO. Este test existe porque el codigo estaba mal y el verde
// no lo decia: el contador independiente contaba los saltos de linea, el lector
// de CSV se salta las lineas vacias en silencio (medido), y un fichero acabado
// en una linea en blanco (o sea, casi cualquiera que haya pasado por un editor)
// daba descuadre y no cargaba.
//
// Una guarda que salta con lo normal se acaba quitando, y entonces deja de
// guardar lo raro, que era su trabajo.
func TestUnaLineaEnBlancoNoDescuadraElRecuento(t *testing.T) {
	casos := []struct{ nombre, datos string }{
		{"al final", "usuario;permiso\nana;admin\n\n"},
		{"en medio", "usuario;permiso\nana;admin\n\nluis;lector\n"},
		{"varias seguidas", "usuario;permiso\nana;admin\n\n\n\nluis;lector\n"},
		{"con CRLF", "usuario;permiso\r\nana;admin\r\n\r\n"},
		{"sin salto final", "usuario;permiso\nana;admin"},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			ins, err := Tomar([]byte(c.datos), opcionesDePrueba())
			if err != nil {
				t.Fatalf("no carga: %v", err)
			}
			if !ins.Cuadra() {
				t.Fatalf("descuadre por una linea en blanco: %v", ins.Cuenta())
			}
			if len(ins.Filas) == 0 {
				t.Fatal("no ha salido ninguna fila")
			}
		})
	}
}

// Y EL NUMERO DE LINEA SOBREVIVE A LAS LINEAS EN BLANCO. Segundo fallo que el
// verde no veia: con un contador propio (`linea++` por vuelta) el numero se va
// corriendo desde la primera linea saltada, y apunta mal justo donde importa: en
// la fila ilegible que alguien tiene que ir a arreglar y en la duplicada que
// dice "esta ya salio en la linea N".
//
// Lo arregla preguntarselo al lector (FieldPos), que es quien sabe donde esta.
func TestElNumeroDeLineaEsElDelFicheroYNoUnContadorPropio(t *testing.T) {
	// La linea 4 esta en blanco: "luis" esta en la 5 del fichero.
	ins := tomar(t, []byte("usuario;permiso\nana;admin\n\n\nluis;lector\n"))
	if len(ins.Filas) != 2 {
		t.Fatalf("filas: %+v", ins.Filas)
	}
	if ins.Filas[0].Linea != 2 {
		t.Errorf("ana esta en la linea 2 del fichero y se ha guardado la %d", ins.Filas[0].Linea)
	}
	if ins.Filas[1].Linea != 5 {
		t.Errorf("luis esta en la linea 5 del fichero y se ha guardado la %d.\n"+
			"  Quien abra el fichero para arreglar esa fila mirara la linea equivocada",
			ins.Filas[1].Linea)
	}
	// Y con una ilegible detras de las blancas, el rango tambien apunta bien.
	con := tomar(t, []byte("usuario;permiso\nana;admin\n\n;lector\nluis;lector\n"))
	if len(con.Ilegibles) != 1 {
		t.Fatalf("ilegibles: %+v", con.Ilegibles)
	}
	if con.Ilegibles[0].Desde != 4 {
		t.Errorf("la fila sin identificador esta en la linea 4 y el ilegible dice la %d",
			con.Ilegibles[0].Desde)
	}
}

// EL SELLO CUBRE LA LECTURA ENTERA, no solo el fichero.
//
// SALIO DE LA PASADA 2, preguntando lo de siempre: por que campo casa el
// emparejamiento y si ese campo esta firmado. La clave de una fila es
// sistema|cuenta|permiso, y `cuenta` y `permiso` salen de los bytes que cubre el
// hash, pero **`sistema` lo declara quien sube**. O sea que el mismo fichero
// subido dos veces con dos sistemas distintos da el MISMO hash y claves
// DISTINTAS, y "la conclusion cuelga del hash exacto" se queda corto.
func TestElSelloDistingueDosLecturasDelMismoFichero(t *testing.T) {
	datos := []byte("usuario;permiso\nana;admin\n")
	a := tomar(t, datos)

	o := opcionesDePrueba()
	o.Sistema = "crm"
	b, err := Tomar(datos, o)
	if err != nil {
		t.Fatal(err)
	}
	if a.Hash != b.Hash {
		t.Fatal("el hash tiene que ser el mismo: es el del fichero, y el fichero es el mismo")
	}
	if a.Filas[0].Clave() == b.Filas[0].Clave() {
		t.Fatal("las claves tenian que ser distintas: el sistema entra en la identidad de la fila")
	}
	if a.Sello() == b.Sello() {
		t.Error("dos lecturas que producen claves distintas comparten sello. Entonces una " +
			"conclusion atada al sello no dice sobre que se reviso, que es lo unico que el " +
			"sello existe para decir")
	}
	// Y el sello es estable: misma lectura, mismo sello.
	if a.Sello() != tomar(t, datos).Sello() {
		t.Error("el sello no es estable entre dos lecturas iguales")
	}
}

// LA CUENTA SE IMPRIME ENTERA. La ley de conservacion no vale si el resultado no
// se puede leer: quien firma la campana tiene que ver los cubos y su suma.
func TestLaCuentaDiceTodosLosCubosYSuma(t *testing.T) {
	ins := tomar(t, []byte("usuario;permiso\nana;admin\nana;admin\n;lector\nluis;lector\n"))
	c := ins.Cuenta()
	for _, clave := range []string{"legibles", "ilegibles", "duplicadas", "lineas cubiertas", "lineas de datos"} {
		if _, hay := c[clave]; !hay {
			t.Errorf("la cuenta no trae el cubo %q: %v", clave, c)
		}
	}
	if c["lineas cubiertas"] != c["lineas de datos"] {
		t.Errorf("los cubos no cubren el fichero: %v", c)
	}
	if c["legibles"] != 2 || c["duplicadas"] != 1 || c["ilegibles"] != 1 {
		t.Errorf("cubos mal repartidos: %v", c)
	}
}
