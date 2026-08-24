# Desarrollar dutiq con Claude Code

Este workspace está preparado para que Claude Code lo desarrolle entero contigo al mando. Todo el contexto que Claude necesita ya está dentro: `CLAUDE.md` (las reglas), `ETAPAS.md` (el plan con casillas), `docs/guia.md` (la fuente única del plan, con los formatos de E1 y E3 en sus Anexos A y B) y `docs/diseno.md` (el diseño, novena ronda; la guía es la undécima y definitiva).

## Arranque (5 minutos)

```bash
cd dutiq
go test ./...        # todo en verde antes de empezar: es tu línea base
claude               # arrancar Claude Code aquí
```

Primera orden dentro de Claude Code: `/etapa`. Te sitúa: etapa en curso, siguiente casilla, y el plan de ataque de la sesión.

## El bucle de trabajo (así se construye todo)

1. **`/etapa`** al empezar cada sesión: una casilla de ETAPAS.md, no una etapa entera.
2. **Plan mode primero** (Shift+Tab) para lo no trivial: que Claude proponga el plan contra `docs/guia.md` ANTES de tocar código. La regla de oro está en CLAUDE.md: **no re-decidir diseño**; si Claude propone cambiar una decisión de la guía, que lo justifique contra la sección concreta y decide tú.
3. Implementar **con su test-puerta en el mismo cambio**. Sin test no hay casilla.
4. **`/puerta`** antes de commitear: corre todo y propone qué casillas marcar.
5. Commit pequeño con el porqué. Nunca con tests en rojo.
6. **`/clear` entre casillas**: contexto limpio, CLAUDE.md se recarga solo.

## Al cerrar cada etapa (el método que hizo este proyecto)

**`/adversarial`**, o mejor: lanza el subagente `revisor-hostil` con la etapa entera. Este proyecto existe porque siete revisores hostiles encontraron 8 bugs en código que pasaba sus tests, 42 fallos de plan y un 4,5/10 que se convirtió en el diseño final. No declares una etapa cerrada sin su ronda. Los bloqueantes se arreglan antes de seguir; se documenta lo que encontró en el commit de cierre.

## Los comandos y agentes incluidos

| Qué | Cuándo |
|---|---|
| `/etapa` | al abrir sesión: sitúa y propone el ataque |
| `/puerta` | antes de commitear: corre las puertas y propone casillas |
| `/adversarial` | al cerrar etapa o antes de release |
| `/autoria` | para convertir un artículo normativo en obligación con dorados (respeta la frontera legal solo) |
| agente `revisor-hostil` | la ronda de cierre de etapa, en paralelo mientras haces otra cosa |
| agente `autor-corpus` | fábrica de paquetes: dale artículos, te devuelve JSON con dorados y linter en verde |

## Cómo pedir cada etapa (los prompts de arranque)

Copia el que toque, tal cual, como primera orden de la etapa:

- **E1**: "Arranca la etapa 1 según docs/guia.md §3 y Anexo A: el ledger v2 con AEAD con compromiso de clave (struct EntradaV2 del Anexo A; no hay ledger persistido previo, así que sin migración), con sus tests-puerta incluido el control negativo de clave sustituida. Solo esa casilla."
- **E2**: "Arranca la etapa 2 según docs/guia.md §4: el esqueleto de serve con html/template + htmx vendorizado y la puerta de seguridad web (CSRF, rate limit, cabeceras en CI, primer admin por token). Solo esa casilla."
- **E3**: "Etapa 3: primero la casilla de extensión del formato (guia.md Anexo B: clase_e2e, temporalidad, escalado, pruebas/ con su linter). Después completa paquetes/ens con el agente autor-corpus: empieza por los artículos con reloj (31 auditoría, 32 INES, gestión de incidentes) y sigue con el articulado y el Anexo II, con 3 dorados por reloj."
- **E4**: "Etapa 4 según docs/guia.md §6: el objeto Incidente mínimo con timeline bitemporal y el payload de brecha AEPD. Solo esa casilla."
- **E5**: "Etapa 5 según docs/guia.md §7: el verificador de citas por hash como función pura con sus adversariales (cita truncada, de otra versión, inventada). Solo esa casilla."
- **E6**: "Etapa 6 según docs/guia.md §8: el host Extism con el ABI v1 (describe/collect/health) y el sandbox por capacidades, con el test negativo de petición fuera del allowlist."
- **E7**: "Etapa 7 según docs/guia.md §9: el paquete MAGERIT v3 como datos con el agente autor-corpus, y el crosswalk a ENS e ISO 27005."
- **E8**: "Etapa 8 según docs/guia.md §10: la licencia Ed25519 como fichero firmado con activación offline y su verificación, con tests."

## Trucos que pagan

- **Corpus en paralelo**: la autoría de paquetes no toca código; usa un git worktree y el agente `autor-corpus` en una sesión aparte mientras la principal hace código. Es cómo caben las ~150-200 horas de autoría estimadas dentro del calendario (medir las primeras 20 obligaciones y recalibrar).
- **Las 2-3 horas semanales de vigilancia normativa** (desde E3): sesión corta con `/autoria` sobre lo que haya cambiado en BOE/DOUE esa semana.
- **Si Claude sugiere una dependencia nueva**: la respuesta por defecto es no; si de verdad hace falta, fila en DEPENDENCIAS.md primero. En `nucleo/`, jamás (el test lo parará igualmente).
- **Los números no se maquillan**: si una puerta no pasa, la casilla no se marca. El proyecto entero se sostiene sobre esa regla.

## Qué está ya hecho al abrir esto

Núcleo completo en `nucleo/` (motor temporal, Datalog, 8 estados, ledger, expediente, corpus con linter): 100+ tests en verde, cero dependencias. Los 9 puertos hexagonales definidos y compilando en `puertos/`. El paquete ENS sembrado y pasando el linter en `paquetes/ens`. La web del open core en `web/`. El CI en `.github/workflows/ci.yml`. Empiezas en la semana 0 de ETAPAS.md con 7 casillas abiertas (licencia canónica, marca y TU_USUARIO, PI del contrato, vulnerability reporting, endurecer gosec, revisión del CLA) y de ahí a la etapa 1.
