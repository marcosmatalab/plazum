# plazum

**Compliance continuity: never fall out of conformity.**

A single Go binary that works out which regulations apply to you, what you have
to do and by which exact date, with the legal citation for every claim. It checks
what can be checked, chases what only a human can do, generates the paperwork,
and leaves an audit file a third party can verify offline and without trusting
you.

No external dependencies: `go.mod` has no `require` line.

> **A note on language,.** plazum's
> domain is Spanish and EU law: the BOE, the DOUE, Regulation 1182/71, Ley
> 39/2015, NIS2, DORA, the AI Act, the CRA. The corpus identifiers are the real
> ones and do not survive translation. So **the repository is in Spanish**, and
> this page exists so you can decide whether it is worth your time. The code is
> ordinary Go; the comments are not.

![The obligation calendar: the next twelve months with their regulation, article and countdown](superficies/pantallas/capturas/calendario-claro.png)

## Run it

```bash
docker build -t plazum . && docker run --rm plazum
```

```bash
go install github.com/marcosmatalab/plazum/cmd/plazum@latest
plazum demo                                             # a sample company and its clocks
plazum demo --serve                                     # and the server on that state
plazum calendario --pais=ES --sector=servicios-digitales --empleados=200
```

![plazum calendario and plazum verify, actually running in a terminal](docs/demo.gif)

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

## AI, deliberately fenced in

plazum computes dates with real consequences, so the core is deterministic and
does not know AI exists. That is not a slogan: one test verifies by AST that
`nucleo/` imports nothing from outside, another that it never reads the system
clock, and a CI gate runs the entire suite with AI switched off. AI lives in the
adapters, and every citation it produces is verified by hash against the real
text before it is ever shown.

---

Full documentation, in Spanish: [`README.md`](README.md). Licence: code
**AGPL-3.0**, corpus data **Apache-2.0**. **None of this is legal advice.**
