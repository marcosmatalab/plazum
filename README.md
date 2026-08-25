# plazum

**El GRC de continuidad: no pierdas nunca la conformidad.**

Un solo binario en Go que sabe qué normas te aplican, qué tienes que hacer y para qué fecha exacta, con la cita de cada cosa. Comprueba solo lo comprobable, agenda y reclama lo humano, genera los documentos, escala si nadie atiende, y lo deja todo en un expediente que un auditor puede verificar sin fiarse de ti.

**Estado: en construcción, por etapas y en público.** Lo que ya existe y está medido: el núcleo determinista completo: motor de plazos multi-régimen, aplicabilidad Datalog, 8 estados, ledger v1 con Merkle y v2 con AEAD comprometido y borrado legal con lápidas, blobs cifrados content-addressed, historia bitemporal, el objeto Certificado con sus dorados (ISO trienal, ENS bienal, SOC 2 solapadas), perímetros multi-entidad, y el corpus con clase e2e, relojes declarados y casos dorados que se ejecutan contra el motor. 108 casos de test en verde (mas fuzzing y detector de carreras), cero dependencias. El corpus trae los 30 marcos montados con su estrato legal (paquetes/CORPUS.md): tres con relojes reales y 12 casos dorados que se ejecutan contra el motor; el resto, esqueletos honestos con la transcripción planificada en ETAPAS.md. Y un test de integración recorre el ciclo entero: del paquete al expediente verificado offline, pasando por el reloj, el estado, la historia y el borrado legal. Lo que falta y cuándo: `ETAPAS.md`. El diseño (novena ronda) y la guía definitiva (undécima): `docs/`.

## Probar lo que hay hoy

Con Docker, sin instalar Go y en un comando:

```bash
docker build -t plazum .
docker run --rm plazum
```

Eso instala una empresa de ejemplo, deriva sus obligaciones y enseña sus relojes corriendo. El corpus y el expediente de ejemplo viajan dentro de la imagen, así que lo demás también funciona sin montar nada:

```bash
docker run --rm plazum verify expediente-demo.json contexto-demo.json
docker run --rm plazum explain expediente-demo.json
docker run --rm -p 8443:8443 plazum serve --direccion 0.0.0.0:8443
```

La imagen es un binario estático sobre `scratch`, corre sin privilegios y no trae intérprete de órdenes. Dos construcciones del mismo commit dan el mismo binario, y eso se comprueba en CI.

Con Go instalado:

```bash
go build -o plazum ./cmd/plazum
./plazum demo                                                # el mismo ejemplo, sin Docker
./plazum verify expediente-demo.json contexto-demo.json      # recalcula el expediente demo, sin red
./plazum cobertura paquetes                                  # la cobertura honesta del corpus instalado
go test ./...
```

El contexto de verificación lo aporta el receptor, no el expediente. Verificar un expediente con los datos que trae el propio expediente sería comparar al emisor consigo mismo.

## Los tres pilares

1. **Obligaciones con reloj legal de verdad**: días hábiles, calendarios combinables, cierre y traslado según el Rgto. 1182/71, suspensiones, prórrogas, y las dos lecturas cuando la doctrina discrepa, cada una con su cita.
2. **Expediente verificable offline**: cadena de hashes, Merkle RFC 6962, sellado RFC 3161; un tercero lo recalcula entero sin red y sin confiar en el emisor.
3. **El corpus es datos, no código**: cada norma es un paquete con sus obligaciones, relojes, preguntas y plantillas. Añadir la norma 31 no toca una línea de código, y hay un test que lo demuestra.

## Licencia y modelo

Código AGPL-3.0, completo (SSO incluido). Los datos, abiertos e inmediatos para todos. De pago: la instancia gestionada en la UE y el contrato de servicio sobre el corpus (plazo objetivo con histórico público, changelog sellado, avisos). Ver `web/`.

Soporte: Discussions, sin SLA. Vulnerabilidades: `SECURITY.md`.
