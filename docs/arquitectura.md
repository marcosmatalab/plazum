# La arquitectura, y el test que vigila cada flecha

plazum es un hexágono: un núcleo determinista que no conoce el mundo, unos
puertos que declaran lo que necesita, y unos adaptadores y superficies que lo
conectan con lo que hay fuera.

Eso lo dicen todos los proyectos. **Lo que hace distinto a éste es que cada
flecha de aquí abajo tiene un test que la vigila leyendo el AST**, y que esos
tests corren en cada empujón. Una capa no es una capa porque esté en un
directorio: es una capa cuando cruzarla rompe el build.

```
                        ┌──────────────────────────────────────────┐
   lo que hay fuera     │  superficies/   (10)   adaptadores/ (19) │
   (HTTP, IdP, TSA,     │  serve, pantallas,     tsa, oidc, scim,  │
   ficheros, LLM)       │  calendario, acta,     ia, evidencia,    │
                        │  uar, export, scim     latido, catalogo  │
                        └────────────────┬─────────────────────────┘
                                         │ implementan
                        ┌────────────────▼─────────────────────────┐
                        │  puertos/   16 interfaces, cero lógica   │
                        └────────────────┬─────────────────────────┘
                                         │ el núcleo los declara
                        ┌────────────────▼─────────────────────────┐
                        │  nucleo/   (20 paquetes, CERO imports    │
                        │  externos, CERO time.Now())              │
                        │  ventana, aplicabilidad, estado, ledger, │
                        │  expediente, historia, corpus, acta...   │
                        └────────────────▲─────────────────────────┘
                                         │ se cargan como DATOS
                        ┌────────────────┴─────────────────────────┐
                        │  paquetes/   20 normas, JSON + dorados   │
                        └──────────────────────────────────────────┘
```

## Cada flecha, con su puerta

| La afirmación | Quién la vigila | Cómo |
|---|---|---|
| `nucleo/` no importa nada de fuera del núcleo, ni de terceros | `TestElNucleoNoImportaElExterior` | recorre el AST de cada fichero y mira sus `import` |
| ...y el detector de verdad detecta | `TestElDetectorDeImportsSaltaCuandoDebe` | control negativo sobre fuentes sintéticas |
| ...y no se deja ni un fichero sin mirar | `TestElRecorridoDelNucleoVeTodoElArbol` | suelo sobre los ficheros recorridos: un recorrido vacío daría verde |
| `nucleo/` no llama a `time.Now()`: el instante entra como dato | `TestElNucleoNoLeeElRelojDelSistema` | AST otra vez, con su control negativo |
| Ninguna norma está cableada en el código | `TestNingunaNormaCableada` | busca identificadores de norma en literales de cadena |
| ...y distingue la norma de una palabra que la contiene | `TestElDetectorDistingueLaNormaDeUnaPalabraQueLaContiene` | control negativo en las dos direcciones |
| Añadir una norma no toca código | `TestNormaNuevaNoTocaCodigo` | mete un paquete sintético y comprueba que carga |
| El núcleo no conoce la IA, **y ni siquiera la nombra** | `TestElNucleoNoConoceLaIA` | copiar el interfaz cumpliría la letra rompiendo el fondo |
| La IA sólo vive en adaptadores y superficies | `TestLaIASoloViveEnAdaptadoresYSuperficies` | AST sobre el árbol entero |
| El producto entero funciona con la IA apagada | `TestLaSuiteCorreConLaIADesactivada` | y además un paso de CI con `PLAZUM_SIN_IA=1` |
| `cmd/plazum` no engorda en silencio | `TestElFanOutDeCmdPlazumNoCreceEnSilencio` | igualdad exacta sobre los paquetes del módulo que importa |
| Ningún paquete ni fichero crece sin que alguien lo decida | `TestLosPaquetesGrandesTienenTechoYNoSorpresa` | techo por paquete y techo del fichero mayor del árbol |
| El binario no lleva ni una dependencia externa | `TestElBinarioNoLlevaNingunaDependenciaExterna` | `go list -deps` sobre el binario de verdad |

## Las tres decisiones que explican la forma

**1. El instante entra como dato.** `nucleo/` no puede preguntar la hora. Eso
suena a purismo hasta que hay que verificar un expediente emitido hace ocho
meses: el verificador vuelve a calcular con el instante que trae el caso, no con
el de hoy, y por eso el resultado es el mismo en cualquier máquina y en
cualquier fecha. Un motor que lee el reloj no se puede auditar.

**2. El corpus es dato, no código.** Cada norma es un directorio bajo
`paquetes/` con su JSON, sus obligaciones, sus relojes, sus preguntas, sus
plantillas y sus casos dorados. La consecuencia medible: añadir la norma 21 no
toca una línea de Go, y si alguien intenta cablear un identificador de norma el
build se rompe. La otra consecuencia, menos obvia, es que **el linter del corpus
es una frontera legal**: un paquete referencial con más de 120 caracteres de
texto normativo no carga, así que el producto no puede redistribuir lo que no
puede redistribuir.

**3. Los puertos están congelados.** Las 16 interfaces de `puertos/` no tienen
lógica y no cambian por conveniencia de un adaptador. Un frente que necesita
cambiar un puerto escribe la propuesta en
[`puertos-propuestas.md`](puertos-propuestas.md) y sigue contra el interfaz
actual. Es lo que permite construir varios frentes a la vez sin que se pisen.

## Lo que esta página no dice

No dice que la arquitectura sea buena: dice que es **la que está**, y que
cruzarla rompe el build. El diseño con su porqué está en
[`diseno.md`](diseno.md), las decisiones con su alternativa descartada en
[`decisiones.md`](decisiones.md), y las reglas que sostienen todo esto, cada una
con la fecha del día en que algo se rompió por no tenerla, en
[`invariantes.md`](invariantes.md).
