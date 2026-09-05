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
7. The domain does not know what time it is: no `time.Now` or `time.Since` outside `adapter/`, `container/` and an application's `cmd/` — take a clock port. No `fmt.Print*` outside `adapter/cli/` and `cmd/`. No `panic` anywhere but a test. → `.golangci.yml`, `forbidigo` — `make lint-go` fails on what it finds
8. The core writes to no stream of its own. What went wrong in work it carries on past is said through `port.Trouble`; what a call could not answer is that call's error.

## Adapters

9. An adapter takes no other adapter. Its own subpackages are itself; `internal/adapter/x` is not `adapter/x`. → `refused` / `sibling`
10. An adapter never names `container`. It is given what it needs. → `refused`
11. Only a driving adapter imports `usecase/**`. The driving adapters are `adapter/cli`, `adapter/flashcardsui`, `adapter/mcp`, `adapter/webui`, `internal/adapter/theme`. → `driving` in `layers_test.go`
12. An adapter importing the generated schema is a driving adapter and is listed as one. Direction is read off the messages an adapter handles, never off the folder it sits in: `internal/` says only that nothing outside composes it. → `TestEveryAdapterServingTheSchemaIsDriving`
13. `adapter/` holds exactly `agent, cli, flashcardsui, index, mcp, settings, webui`. `internal/adapter/` holds exactly `appstate, embed, filesystem, pdf, proofreading, recognition, theme, transcription, trash`. A new adapter is a line added to the list, which is what makes it a decision. → `TestTheCoresPublicAdaptersAreTheseAndNoOthers`, `TestTheCoresHeldAdaptersAreTheseAndNoOthers`
14. An existing edge that breaks a rule above is an entry in `owed`, and that list only shrinks. Add an entry; never widen a rule.

## Ports

15. An interface the composition root binds an adapter to is declared in `port/`. An interface a single use case needs and nothing binds is declared beside that use case. What has to see it decides where it goes. → [Where a port is declared, and where an adapter stands](docs/adr/0047-where-a-port-is-declared-and-where-an-adapter-stands.md)
16. **The number of callers decides nothing.** A one-caller port is a port. `VaultWatcher`, `IndexMaintenance` and `VectorQueries` are settled cases; do not reopen them.
17. Every interface in `port/` is named as `port.X` somewhere outside `port/`. A port whose last caller went is deleted, not kept. So is every other type `port/` declares: the words a conversation is held in stand beside it, and a word nothing outside says belongs to no conversation. → `TestEveryPortIsAskedForSomewhereElse`, `TestEveryTypePortDeclaresIsNamedSomewhereElse`
18. No adapter writes `var _ port.X = …`. The binding is the composition root's, and naming it in the adapter puts it in two places. → `TestNoAdapterNamesThePortItSatisfies`, and `TestNoApplicationNamesThePortItSatisfies` in `modules/apps/desktop/internal/layers/`
19. A port is named after the need and in the core's own language; an adapter after the technology. The core asks for a `VaultReader`; that the answer is a filesystem is known in `adapter/` and `container/` alone.

## Applications

20. An application under `modules/apps/**` takes `container`, the adapters it serves something through, `domain` and `port`. Everything else of the core is the core's work, and an application doing it is an entry in that module's `owed`. → `TestNoApplicationDoesTheCoresWork` in `modules/apps/desktop/internal/layers/`
21. No application reaches another. What two of them share is a library under `modules/libs/`. → the same test

## The interface library

22. Nothing in `modules/libs/ui` imports `@numen/protocol`, `@numen/desktop-ui` or `modules/apps/**` — not in a component, not in a story, not in a fixture. The dependency runs `modules/apps/*` → `modules/libs/ui`, never back and never sideways. → the package names fail to resolve because `modules/libs/ui/package.json` declares neither; a relative path into `apps/` would typecheck, and nothing refuses it
23. A component's pure core takes the clock, the animation frame and the viewport as parameters. `Date.now`, `new Date()`, `Math.random`, `requestAnimationFrame`, `matchMedia` and `getBoundingClientRect` belong in `lib/clock.ts` and in `.vue` views, not in a pure `.ts`. → nothing refuses this today

## Names in TypeScript

The frontend grew a house dialect — `Plexing`, `Filing`, `Drawn`, `Asked`, `About` — because agents were told to take a word the repository already used. Where the repository's vocabulary is itself wrong, that rule spreads the fault. It is inverted here.

24. **A name comes from what the field calls the thing, not from what this repository already calls it.** The repository's word counts only where it is also the standard one; where the two differ, the standard wins. [`docs/glossary.md`](docs/glossary.md) already says this of domain concepts — "a concept takes the name its field already gives it" — and it holds for identifiers too. This rule outranks 25–30: they are the standard as of writing, and a better-sourced standard replaces them.
25. **A type or interface is a noun or noun phrase.** Not a gerund (`Filing`), not a participle (`Drawn`), not a third-person verb (`Says`), not a preposition (`About`). Verb phrases name functions and methods; nouns name the things they act on. → Martin, *Clean Code* ch. 2 ("Class Names"): classes take noun or noun phrase names, "a class name should not be a verb"; Cwalina & Abrams, [*Framework Design Guidelines*](https://learn.microsoft.com/en-us/dotnet/standard/design-guidelines/names-of-classes-structs-and-interfaces): "DO name classes and structs with nouns or noun phrases … This distinguishes type names from methods, which are named with verb phrases."
26. **No `I` prefix and no `Interface` suffix**, and nothing else in a name that only restates the type. → [microsoft/TypeScript coding guidelines](https://github.com/microsoft/TypeScript/wiki/Coding-guidelines): "Do not use `I` as a prefix for interface names"; [Google TypeScript style guide](https://google.github.io/styleguide/tsguide.html): "TypeScript expresses information in types, so names should not be decorated with information that is included in the type."
27. **A name is descriptive and spelled in whole words.** → Google: "Names must be descriptive and clear to a new reader"; TypeScript guidelines: "Use whole words in names when possible"; [Vue style guide](https://vuejs.org/style-guide/rules-strongly-recommended.html): "Component names should prefer full words over abbreviations."
28. **The recurring shapes take the suffix the field gives them**, so a reader knows the kind of thing before reading the body. A component's inputs are `Props` ([Vue](https://vuejs.org/guide/typescript/composition-api.html) names the interface `Props`); an optional argument bag is `Options`; a required collaborator set passed in is `Deps`; a thing a composable holds is `State`; an address to something is `Ref`; the imperative surface a component exposes is a `Handle`; a payload carried by an event is an `Event`. Named after what they belong to: `PlexTabDeps`, `MenuRequest`, `NoteRef`.
29. **`UpperCamelCase` for types, `lowerCamelCase` for values, `CONSTANT_CASE` for global constants.** A composable is a function named `useX`. → [Google](https://google.github.io/styleguide/tsguide.html); [Vue composables](https://vuejs.org/guide/reusability/composables.html): "composable functions … camelCase names that start with `use`".
30. **Do not rename to a synonym.** A rename earns its churn only where a stranger who has not read the body is plainly better off. Names that already read as things — `Vault`, `Entry`, `Passage`, `Transcript` — stay. → Ousterhout, *A Philosophy of Software Design* ch. 14: a name must "create an image" and be precise; a name that resists this is a sign the thing itself is unclear.

31. **A component name is multi-word; a data type may be a single noun.** That is what keeps them apart in `modules/libs/ui/src/index.ts`, where one barrel exports both. The bare noun belongs to the type — `Stencil`, `Deck`, `Card` — and the component says what it draws or does with one: `StencilEditor`, not `StencilView`. A `View` suffix added only to dodge the barrel is the collision showing through, and rule 26 already refuses it. → [Vue style guide, essential rules](https://vuejs.org/style-guide/rules-essential.html): "User component names should always be multi-word, except for root `App` components. This prevents conflicts with existing and future HTML elements, since all HTML elements are a single word."

Nothing refuses any of 24–31. They are read by a person and by a reviewer.

## The reviewer

`.claude/agents/go-reviewer.md` reviews Go changes against the same records, afterwards. The one place the two could have disagreed is the number of a port's callers: the reviewer's rule against abstracting before a second case predates the rule above, and both files now draw the line the same way. It holds for an ordinary Go interface; inside `port/` it does not, and "only one caller" is not a finding there.
