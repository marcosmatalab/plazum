# Cuanto del corpus de la v1 esta escrito

Esta es la cifra que dice si plazum sirve todavia para poco o ya para bastante, y
por eso es la que mas facil seria inflar. No esta escrita a mano: la computa
`TestElPorcentajeDeLaV1LoComputaUnTestYNoUnaPersona` desde
[`paquetes/marcos-v1.json`](../paquetes/marcos-v1.json) y el corpus, y CI se pone
rojo si el documento y el arbol se separan **en cualquiera de los dos sentidos**.

<!-- cobertura-v1:inicio -->
**Cuanto del corpus de la v1 esta escrito, computado por un test y no a mano.** Los quince paquetes que forman los doce marcos de la v1 estan declarados como dato en [`paquetes/marcos-v1.json`](../paquetes/marcos-v1.json), con el motivo de cada uno y el de cada exclusion. Sobre ellos, y separando quien escribe cada numero:

- **54,2 %** de cobertura estricta: 78 relojes **cuyo intervalo lo escribe la norma**, sobre 144 puntos que el censo ha verificado. **Baja desde el 56,7 % del 08-09-2026 y la bajada es la noticia buena**: el ENS sale del calculo entero (numerador y denominador) porque su fila del censo quedo refutada por su propio paquete al entrar los tres relojes de la ITS de Auditoria. Es la tercera correccion de esta cifra y la tercera hacia abajo, que es lo que pasa cuando una medida deja de contar a favor.

  **Este «78 sobre 144» y el «79 de 146 casillas» de `docs/ETAPAS.md` NO son la misma cifra, y que se parezcan tanto no significa nada.** Aqui el numerador son RELOJES cuyo intervalo escribe la norma y el denominador son PUNTOS CENSADOS, que salen de sumar las siete filas con denominador de `paquetes/marcos-v1.json`. Alli se cuentan CASILLAS DEL PLAN, derivadas del arbol de ese fichero. **Hasta el 22-09-2026 las dos fracciones eran identicas, y al marcar dos casillas se separaron en uno: eso las hace mas confundibles, no menos**, porque una diferencia de uno se lee como un descuadre que alguien deberia arreglar, y cuadrarlas corrompe una en silencio. Lo vigila `TestLasDosCifrasQueCoincidenPorCasualidadLoDicen`, que desde ese dia exige el descargo SIEMPRE y no solo mientras las cifras coincidan.
- **+68 rituales de plazum** sobre esos mismos marcos: puntos que obligan a una cadencia y no dan cifra, donde plazum propone el intervalo, lo justifica y el cliente lo cambia (D-12). Estan escritos y no cuentan arriba.

**Son dos numeros y no uno porque sumarlos permite subir la cobertura escribiendo relojes nuestros**, que es justo el incentivo que no queremos. La cifra estricta mide algo mas duro que "cuanto hay escrito": `nis2-tecnica` tiene sus 48 puntos escritos y aporta 4 de 48, porque en 44 de ellos el anexo impone la cadencia sin dar el numero.

**Este porcentaje se ha corregido tres veces y las tres correcciones lo BAJARON**, que es lo que hay que saber de el: no es mala suerte tres veces, es que la metrica tenia tres formas distintas de inflarse. Cada una se lee abajo con su mecanismo, y los tres mecanismos quitan del numerador o topan una fraccion: ninguno puede subir el numero, que es como se comprueba esta frase sin creersela. Primero salio del numerador un paquete referencial que aportaba 6 arriba y 0 abajo. Despues salieron los rituales de todos. Y la tercera la encontro una puerta nueva: **dos paquetes tenian mas relojes con cita escritos que puntos contaba su censo**, o sea una fraccion por encima de uno, y en un agregado eso sube el total sin que nada lo nombre. Una cifra cuyo fallo probable es favorecerte necesita puerta en las dos direcciones, y ahora la tiene: un test la computa del arbol, se pone rojo si se separa de esta linea en cualquier sentido, y ademas rechaza que un paquete aporte mas arriba que abajo.

Y **8 de los 15 marcos** quedan **fuera de ese porcentaje**, con su motivo escrito: cuatro referenciales que no se pueden censar sin la norma delante (invariante 3), el RD 43/2021, al que el censo no le ha dado fila con las tres columnas, y los TRES cuyo censo quedo refutado por su propio paquete, que desde el 10-09-2026 incluyen al ENS. Para ellos la cifra honesta es **sin denominador, 30 rituales y 56 relojes escritos**, nunca un cero: un cero se lee como medido y vacio, y no estan medidos.

**Y lo que queda arriba tampoco es un techo.** El denominador de `ai-act` es un suelo declarado (sube a 29 como minimo cuando se recuente el Reglamento (UE) 2026/1744), y de los dos paquetes refutados hay **46 relojes identificados y sin escribir** que ningun censo cuenta todavia. Un denominador que va a crecer es un porcentaje que va a bajar, y se dice antes de que baje.
<!-- cobertura-v1:fin -->
