# plazum

**Compliance continuity: never fall out of conformity.**

A single Go binary that works out which regulations apply to you, what you have
to do and by which exact date, with the legal citation for every claim. It checks
what can be checked, chases what only a human can do, generates the paperwork,
and leaves an audit file a third party can verify offline and without trusting
you.

No external dependencies: `go.mod` has no `require` line.

> **A note on language.** plazum's
> domain is Spanish and EU law: the BOE, the DOUE, Regulation 1182/71, Ley
> 39/2015, NIS2, DORA, the AI Act, the CRA. The corpus identifiers are the real
> ones and do not survive translation. So **the repository is in Spanish**, and
> this page exists so you can decide whether it is worth your time. The code is
> ordinary Go; the comments are not.

![The obligation calendar: the next twelve months with their regulation, article and countdown](superficies/pantallas/capturas/calendario-claro.png)

## Run it

```bash
docker run --rm ghcr.io/marcosmatalab/plazum:v0.1.0   # no Go, no clone
```

```bash
go install github.com/marcosmatalab/plazum/cmd/plazum@latest
plazum demo                                             # a sample company and its clocks
plazum demo --serve                                     # and the server on that state
plazum calendario --pais=ES --sector=servicios-digitales --empleados=200
```

![plazum calendario and plazum verify, actually running in a terminal](docs/demo.gif)

Installing, or building the image yourself: [`docs/instalacion.md`](docs/instalacion.md), in Spanish.

Every calendar row is tagged `[supuesto]`: it is what would happen to a company
with that profile, not a conclusion about yours.

## Five claims and their commands

None of these numbers is hand written. A test derives each from the tree and CI
turns red if document and tree drift apart, **in either direction**.

| Claim | Command | What you get |
|---|---|---|
| No external dependencies | `go list -deps -f '{{if not .Standard}}{{.ImportPath}}{{end}}' ./cmd/plazum \| grep -v '^github.com/marcosmatalab/plazum/'` | nothing |
| The binary stays under its declared budget | `GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -trimpath -o plazum ./cmd/plazum` | the figure in [`docs/presupuesto-binario.md`](docs/presupuesto-binario.md) |
| The whole suite passes with no network access | `GOPROXY=off go test ./...` | `ok` |
| The core stays above its hard coverage floor | `go test ./nucleo/... -coverprofile=c.out && go tool cover -func=c.out` | above the floor CI enforces |
| Every CI gate also runs on your machine | `./comprobar.sh` | `26 puertas leidas, 26 ejecutadas` |

## What makes it different from a controls catalogue

1. **A real legal clock.** Working days, combinable national and regional
   calendars, closing and carry-over rules, suspensions and extensions. When
   legal doctrine disagrees, both readings are computed and the divergence is
   shown with its citation: the engine never picks one silently.
2. **An audit file anyone can verify offline.** Hash chain, RFC 6962 Merkle tree,
   RFC 3161 timestamping. A third party recomputes the whole thing without
   network access and without trusting the issuer.
3. **The corpus is data, not code.** Adding a regulation touches no Go at all,
   and a test breaks the build if anyone hard codes a regulation identifier.

## Where the model is fenced in, and how that is enforced

plazum emits dates with legal consequences, so the engine is deterministic by
contract and no model runs inside it. The boundary is executable, not editorial:

| Guarantee | Enforced by |
|---|---|
| The engine depends on no model | AST test: `nucleo/` imports nothing from outside |
| The computation is reproducible | AST test: the instant is an input, never the system clock |
| The product works with the model off | the whole suite runs with `PLAZUM_SIN_IA`, as a CI gate |
| No citation is shown unverified | resolved by hash against the regulation text, or discarded |
| The model cannot invent a regulation it does not hold | the legal-boundary linter keeps third-party clause text out of the corpus |

Citation accuracy is published, not promised: **28 golden cases over 8 sources**
in `evals/citas/`, each with its adversarial corpus and the reason for every
rejection.

## What each decision costs

| Decision | Buys | Costs |
|---|---|---|
| Zero dependencies | audit with no network, no supply-chain surface | PKCS#7 and RFC 3161 vendored, with SHA-256 provenance |
| The instant is an input | an eight-month-old audit file reverifies identically today | the instant crosses the whole core API |
| Corpus as data | adding a regulation touches no Go | a format and a linter to maintain, 808 golden cases to run |
| OSCAL as lossy output only | the internal model is not bent to fit one with no notion of a deadline | no round trip, and it says so |

**Operating budgets**, each a promise with a gate: binary **12,1 MB** against a
25 MB ceiling, core coverage **88,3 %** against a hard 85 % floor, cold start
under 3 s, resident memory under 256 MB.

---

Full documentation, in Spanish: [`README.md`](README.md). Licence: code
**AGPL-3.0**, corpus data **Apache-2.0**. **None of this is legal advice.**
