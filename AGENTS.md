# Constraints

The layering and port rules of this repository, as a list to check code against before it is written. Each is a decision already recorded in [`docs/adr/`](docs/adr/README.md), and where a machine refuses a violation the line names it. Where a line and an ADR disagree, the ADR is right and this file is wrong.

This is not the whole of the architecture. It is the part a generator gets wrong. Commit format, labels and platform runs are in [`CONTRIBUTING.md`](CONTRIBUTING.md).

Paths below are relative to `modules/libs/core/` unless they say otherwise. The two guards are `container/layers_test.go`, which parses import blocks, and `container/direction_test.go`, which reads the transitive `Deps` out of `go list -json` for four platforms.

## The core

1. `domain/`, `port/`, `usecase/**`, and every other package of the core, are compiled from no `adapter/**`, no `internal/adapter/**`, no `container` and no `modules/libs/protocol` — at any remove, not only in the import block. → `TestNothingTheCoreIsCompiledFromReachesOutward`
2. `internal/wire` is the one package that may name the generated schema, because two driving adapters both put its types on the wire. → `owedWire` in `direction_test.go`
3. `container` may name `domain`, `port`, `usecase/**` and the adapters, and nothing else of the core. It binds; it does none of the core's work. → `refused` / `assembling`
4. `container` may not name the generated schema. It is the one package answering for what it names rather than for everything it is built from. → `answering`
5. `domain`, `flashcards`, `markdown`, `internal/cardid` and `internal/ulid` import only each other. A pure package reaching a sibling of the core takes on its goroutines, its channels and its schema. → `pure` in `layers_test.go`
6. A test of a pure package imports no adapter. → `TestNoPurePackageIsTestedThroughAnAdapter`
7. The domain does not know what time it is: no `time.Now` or `time.Since` outside `adapter/`, `container/` and an application's `cmd/` — take a clock port. No `fmt.Print*` outside `adapter/cli/` and `cmd/`. No `panic` anywhere but a test. → `.golangci.yml`, `forbidigo` — reported today, not failed: `make lint-go` runs the core and the desktop under `--issues-exit-code=0` while the backlog is triaged
8. The core writes to no stream of its own. What went wrong in work it carries on past is said through `port.Trouble`; what a call could not answer is that call's error.

## Adapters

9. An adapter takes no other adapter. Its own subpackages are itself; `internal/adapter/x` is not `adapter/x`. → `refused` / `sibling`
10. An adapter never names `container`. It is given what it needs. → `refused`
11. Only a driving adapter imports `usecase/**`. The driving adapters are `adapter/cli`, `adapter/flashcardsui`, `adapter/mcp`, `adapter/webui`, `internal/adapter/theme`. → `driving` in `layers_test.go`
12. An adapter importing the generated schema is a driving adapter and is listed as one. Direction is read off the messages an adapter handles, never off the folder it sits in: `internal/` says only that nothing outside composes it. → `TestEveryAdapterServingTheSchemaIsDriving`
13. `adapter/` holds exactly `agent, cli, flashcardsui, index, mcp, settings, webui`. `internal/adapter/` holds exactly `appstate, embed, filesystem, pdf, proofreading, recognition, theme, transcription, trash`. A new adapter is a line added to the list, which is what makes it a decision. → `TestTheCoresPublicAdaptersAreTheseAndNoOthers`, `TestTheCoresHeldAdaptersAreTheseAndNoOthers`
14. An existing edge that breaks a rule above is an entry in `owed`, and that list only shrinks. Add an entry; never widen a rule.

## Ports

15. An interface the composition root binds an adapter to is declared in `port/`. An interface a single use case needs and nothing binds is declared beside that use case. What has to see it decides where it goes. → ADR-0047
16. **The number of callers decides nothing.** A one-caller port is a port. `VaultWatcher`, `IndexMaintenance` and `VectorQueries` are settled cases; do not reopen them.
17. Every interface in `port/` is named as `port.X` somewhere outside `port/`. A port whose last caller went is deleted, not kept. → `TestEveryPortIsAskedForSomewhereElse`
18. No adapter writes `var _ port.X = …`. The binding is the composition root's, and naming it in the adapter puts it in two places. → `TestNoAdapterNamesThePortItSatisfies`, and `TestNoApplicationNamesThePortItSatisfies` in `modules/apps/desktop/internal/layers/`
19. A port is named after the need and in the core's own language; an adapter after the technology. The core asks for a `VaultReader`; that the answer is a filesystem is known in `adapter/` and `container/` alone.

## Applications

20. An application under `modules/apps/**` takes `container`, the adapters it serves something through, `domain` and `port`. Everything else of the core is the core's work, and an application doing it is an entry in that module's `owed`. → `TestNoApplicationDoesTheCoresWork` in `modules/apps/desktop/internal/layers/`
21. No application reaches another. What two of them share is a library under `modules/libs/`. → the same test

## The interface library

22. Nothing in `modules/libs/ui` imports `@numen/protocol`, `@numen/desktop-ui` or `modules/apps/**` — not in a component, not in a story, not in a fixture. The dependency runs `modules/apps/*` → `modules/libs/ui`, never back and never sideways. → the package names fail to resolve because `modules/libs/ui/package.json` declares neither; a relative path into `apps/` would typecheck, and nothing refuses it
23. A component's pure core takes the clock, the animation frame and the viewport as parameters. `Date.now`, `new Date()`, `Math.random`, `requestAnimationFrame`, `matchMedia` and `getBoundingClientRect` belong in `lib/clock.ts` and in `.vue` views, not in a pure `.ts`. → nothing refuses this today

## The reviewer

`.claude/agents/go-reviewer.md` reviews Go changes against the same records, afterwards. The one place the two could have disagreed is the number of a port's callers: the reviewer's rule against abstracting before a second case predates ADR-0047, and both files now draw the line the same way. It holds for an ordinary Go interface; inside `port/` it does not, and "only one caller" is not a finding there.
