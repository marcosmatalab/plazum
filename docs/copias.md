# Copias y restauración

Qué se copia, qué no se copia, cómo se restaura y cómo se comprueba que lo restaurado sirve para algo.

La regla que ordena todo este documento cabe en una línea. **Una copia que devuelve bytes y deja un ledger que no verifica es peor que no tener copia**, porque da confianza sin darla. Nadie mira un respaldo hasta el día que lo necesita, y ese día ya no queda de dónde sacar otro. Por eso aquí el ensayo de restauración no termina comprobando que el fichero existe, termina verificando la cadena del ledger, sus lápidas y sus supresiones.

## 1. Qué hay en una instalación

| Artefacto | Qué es | Se copia |
|---|---|---|
| la base | la cadena del ledger con sus lápidas y sus checkpoints, y las evidencias dentro | sí, con Litestream |
| `keystore.json` | una clave por entrada y una por evidencia | sí, con réplica **propia** y retención corta |
| `maestra.key` | la clave privada del operador, la que firma lápidas y cierra checkpoints | **no**, va a custodia |
| `paquetes/` | el corpus instalado | **no**, se reinstala desde el canal firmado |

Las evidencias con fichero (el PDF del acta, la captura, el informe del auditor) viven **dentro de la base**, cifradas una a una y direccionadas por el hash de su contenido. La decisión está tomada y está razonada en `docs/guia.md` 3.1, y el motivo es exactamente este documento: hay **un solo fichero que respaldar**, así que Litestream replica la base y con ella viajan las evidencias, `plazum update` hace copia de una cosa, el ensayo restaura una cosa, y no existe una segunda ruta de respaldo que alguien pueda olvidar el día que la necesite.

### Por qué el keystore va aparte

Porque borrar de verdad es destruir una clave, y una clave que viaja en el mismo fichero que la base vuelve con la base.

El ledger no borra entradas, las suprime. La entrada cifrada se queda en la cadena, se destruye su clave y se añade una **lápida firmada con la base legal**. Así la cadena sigue encadenando, las raíces publicadas no cambian, un tercero puede verificarlo todo, y el contenido es irrecuperable. Eso funciona mientras la clave esté destruida en todas partes, respaldos incluidos.

De ahí las dos reglas, que no son de higiene sino de cumplimiento.

1. **El keystore tiene su propia réplica y su propia retención**, hoy 35 días. Cuando se destruye una clave, el borrado es efectivo en el acto para la instancia viva y efectivo para el mundo cuando expira la última generación de réplica que la contenía. Ese plazo se declara en la política de privacidad y consta en la lápida.
2. **La réplica del keystore nunca se restaura por separado de su base.** Restaurar un keystore de anteayer sobre una base de hoy devuelve la clave que el borrado destruyó. No hace falta mala fe, basta con elegir mal la generación en una restauración de urgencia a las tres de la mañana.

**Una restauración que resucita una clave borrada al amparo del artículo 17 del RGPD es un incidente de protección de datos, no un fallo de copia.** Se registra como incidente, se vuelve a destruir la clave sobre la instalación restaurada y se comprueba que no queda ninguna generación dentro del plazo que la conserve.

### Por qué la clave maestra no se replica

Una clave privada que viaja en cada respaldo está en tantos sitios como respaldos haya, y los respaldos se guardan justamente donde es fácil llegar a ellos. Su copia es de **custodia**, la frase de recuperación o el QR impresos, guardados fuera de línea.

Lo que se pierde si se pierde, dicho antes de que pase. **El histórico sigue verificando sin ella**, porque las firmas ya hechas se comprueban con la clave pública. Lo que no se puede hacer hasta reponerla es firmar lápidas nuevas ni cerrar checkpoints nuevos. O sea, se puede demostrar el pasado y no se puede seguir escribiendo, que es el orden correcto de prioridades cuando algo va mal.

### Por qué el corpus no se copia

Los paquetes se reinstalan desde el canal, y el canal los entrega firmados y con su digest. Restaurar el digest esperado y volver a bajar los bytes es más fuerte que restaurar unos bytes de un respaldo, porque comprueba lo que se instala en vez de confiar en lo que se guardó.

## 2. Cómo se configura la réplica

Litestream es una herramienta externa que replica SQLite. **No es una dependencia de Go y no entra en `DEPENDENCIAS.md`**, igual que no entran systemd ni el servidor de correo. Se instala en la máquina, no en el binario.

La base y el keystore se replican con **dos destinos distintos y dos retenciones distintas**, y eso es lo único de esta configuración que no se puede negociar.

```yaml
# /etc/litestream.yml
dbs:
  - path: /var/lib/plazum/plazum.db
    replicas:
      - url: s3://respaldos-plazum/base
        retention: 720h        # 30 dias
```

El keystore no es una base de SQLite, así que no lo replica Litestream. Va con la herramienta de sincronización que use la instalación, con **retención de 35 días**, con cifrado en el destino y con su generación sellada, que es lo que permite después saber si el keystore que se está restaurando es anterior o posterior a un borrado.

Las dos retenciones son distintas a propósito. La de la base puede ser más larga porque la base no contiene nada legible sin claves. La del keystore es la que fija cuándo un borrado pasa a ser efectivo frente al mundo, así que **alargarla es alargar el plazo declarado en la política de privacidad**, y eso se cambia en los dos sitios a la vez o no se cambia.

## 3. Cómo se restaura

1. Se restaura la base al directorio de datos.
2. Se restaura **la generación del keystore que corresponde a esa base**, nunca una anterior. Si la base tiene una lápida del día 20 y el keystore más nuevo que queda es del día 19, ese keystore trae la clave que el borrado destruyó, y hay que tratarlo como incidente en vez de darlo por bueno.
3. Se repone `maestra.key` desde la custodia, si hace falta seguir escribiendo.
4. Se reinstala el corpus desde el canal.
5. **Se verifica lo restaurado antes de darlo por bueno.**

El paso 5 es el que existe este documento para que nadie se salte.

```
ensayocopia verificar -dir /var/lib/plazum -confianza /ruta/fuera/confianza.json
```

El fichero de confianza es el que aporta quien verifica, con la clave pública del operador. Es el mismo fichero de contexto que se le pasa a `plazum verify`. **Tiene que vivir fuera del directorio de datos y fuera de la réplica**, y la herramienta se niega a trabajar si está dentro. El motivo no es de orden: si la clave con la que se comprueba una firma viaja dentro de lo que se restaura, entonces quien pueda escribir en la copia se escribe también esa clave y la verificación se compara consigo misma. Es el mismo agujero que tuvo el expediente en la etapa 1, aplicado a los respaldos.

Si sale bien, la salida dice cuántas entradas verifican, cuántas se abren con las claves restauradas, cuántas evidencias abren y comprueban contra su dirección, y **qué supresiones siguen siendo supresiones, con la base legal de cada una**.

Si sale mal, sale con código 3 y con el arreglo escrito.

## 4. El ensayo de restauración

El ensayo completo no espera a que haya un desastre. Siembra una instalación, la copia, **destruye el original**, la restaura y la verifica.

```
ensayocopia ensayo
ensayocopia ensayo -expediente expediente-demo.json
ensayocopia modos
```

Destruir el original no es teatro. Sin ese paso, nada garantiza que lo que se verifica después no sea el original de siempre, y el ensayo daría verde aunque la copia estuviera vacía.

Corre en cada cambio, en `.github/workflows/etapa2-copias.yml`, y corre **nueve veces**. Una con la copia sana, y ocho rompiéndola de ocho maneras distintas, cada una con el mensaje que tiene que salir.

| Se rompe | Lo caza |
|---|---|
| la réplica no trae el keystore | la restauración se niega y dice qué se ha perdido |
| un byte del cifrado de una entrada | el encadenado, con el verificador del núcleo |
| la base legal de la lápida | la verificación de lápidas del núcleo |
| la clave de la entrada suprimida, repuesta | la comprobación de que un borrado sobrevive a la restauración |
| la generación del keystore, retrasada | el contraste de generaciones entre base y keystore |
| el contenido de una evidencia, bajo su misma dirección | el direccionamiento por contenido |
| el fichero de confianza, metido dentro de lo restaurado | la negativa a comprobar una firma con una clave que viaja en la copia |
| el manifiesto, con un artefacto que sale del directorio | la negativa a restaurar un nombre con `..` o con separadores |

Una puerta que nunca se ha visto fallar no es una puerta. Esas ocho son las que hacen que la primera sirva para algo.

La última merece una línea aparte, porque no la puso el plan sino la pasada adversaria. Restaurar copia lo que el manifiesto **declara**, así que un manifiesto manipulado con un nombre tipo `../../algo` hacía que la restauración escribiera fuera del destino, con los permisos de quien restaura. Y una réplica vive en un bucket o en un disco que se lleva alguien, que es justo donde no se puede dar por hecho que nadie escribe. Ahora un artefacto tiene que ser un fichero suelto del directorio de la réplica, y el keystore tiene que estar **en el manifiesto** y no solo en el disco, porque restaurar sobre la instalación que se quiere reparar dejaba el keystore viejo donde estaba.

## 5. Lo que esto no garantiza

Escrito aquí y no escondido, porque un respaldo del que se cree más de lo que da es la forma más cara de no tener respaldo.

- **No prueba que no exista una copia de una clave destruida en otro sitio.** El ensayo mira la instalación restaurada. Que la clave no siga viva en una generación anterior de la réplica, en un disco externo o en el portátil de alguien es una propiedad de **retención**, no de criptografía, y se sostiene con el plazo de 35 días y con la política, no con un programa.
- **El manifiesto de la copia no es integridad frente a un adversario.** Quien pueda escribir en la réplica puede reescribir también el manifiesto. Sirve contra la copia a medias, el disco que miente y la sincronización interrumpida, que es lo que de verdad pasa. La integridad frente a alguien que quiere engañar la da la cadena, y esa se comprueba contra claves que aporta quien verifica y que no viajan en la copia.
- **Hoy el ensayo no ejercita Litestream.** El adaptador de almacén todavía no existe, así que la base todavía no es un fichero de SQLite. El ensayo monta la instalación con los tipos definitivos del núcleo y mide la propiedad que importa, que la copia devuelve algo que verifica, y esa propiedad no depende del formato. Lo que queda sin cubrir hasta que llegue el almacén es la herramienta de replicación en sí. Está anotado en `docs/pendientes.md`.
- **El anclaje temporal de los checkpoints se comprueba en el otro tramo del ensayo**, el que pasa el expediente emitido por la copia y lo verifica con `plazum verify`. La instalación sembrada no lleva checkpoints porque sellarlos exige salir a una autoridad de sellado, y un ensayo de respaldo que necesita red no se puede correr el día que no hay red.

## 6. Antes de necesitarlo

Tres cosas, y las tres se hacen una vez.

1. Comprobar que la réplica del keystore tiene un destino distinto del de la base y una retención de 35 días.
2. Guardar la frase de recuperación de la clave maestra fuera de línea, y probar una vez que se puede leer.
3. Correr `ensayocopia ensayo` sobre la instalación real, no solo en CI. Un ensayo que solo ha corrido en la máquina del que lo escribió mide la máquina del que lo escribió.
