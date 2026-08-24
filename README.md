# obligo

**El GRC de continuidad: no pierdas nunca la conformidad.**

Un solo binario en Go que sabe qué normas te aplican, qué tienes que hacer y para qué fecha exacta, con la cita de cada cosa. Comprueba solo lo comprobable, agenda y reclama lo humano, genera los documentos, escala si nadie atiende, y lo deja todo en un expediente que un auditor puede verificar sin fiarse de ti.

**Estado: en construcción, por etapas y en público.** Lo que ya existe y está medido: el núcleo determinista completo: motor de plazos multi-régimen, aplicabilidad Datalog, 8 estados, ledger v1 con Merkle y v2 con AEAD comprometido y borrado legal con lápidas, blobs cifrados content-addressed, historia bitemporal, el objeto Certificado con sus dorados (ISO trienal, ENS bienal, SOC 2 solapadas), perímetros multi-entidad, y el corpus con clase e2e, relojes declarados y casos dorados que se ejecutan contra el motor. 107 casos de test en verde (mas semillas de fuzzing), cero dependencias. Lo que falta y cuándo: `ETAPAS.md`. El diseño (novena ronda) y la guía definitiva (undécima): `docs/`.

## Probar lo que hay hoy

```bash
go build -o obligo ./cmd/obligo
./obligo verify expediente-demo.json   # recalcula el expediente demo, sin red
./obligo cobertura paquetes            # la cobertura honesta del corpus instalado
go test ./...
```

## Los tres pilares

1. **Obligaciones con reloj legal de verdad**: días hábiles, calendarios combinables, cierre y traslado según el Rgto. 1182/71, suspensiones, prórrogas, y las dos lecturas cuando la doctrina discrepa, cada una con su cita.
2. **Expediente verificable offline**: cadena de hashes, Merkle RFC 6962, sellado RFC 3161; un tercero lo recalcula entero sin red y sin confiar en el emisor.
3. **El corpus es datos, no código**: cada norma es un paquete con sus obligaciones, relojes, preguntas y plantillas. Añadir la norma 31 no toca una línea de código, y hay un test que lo demuestra.

## Licencia y modelo

Código AGPL-3.0, completo (SSO incluido). Los datos, abiertos e inmediatos para todos. De pago: la instancia gestionada en la UE y el contrato de servicio sobre el corpus (plazo objetivo con histórico público, changelog sellado, avisos). Ver `web/`.

Soporte: Discussions, sin SLA. Vulnerabilidades: `SECURITY.md`.
