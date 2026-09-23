---
name: gatekeeper
description: Stage 1 of the numen review pipeline. Runs the automated gate, then audits the diff against the architecture decisions in docs/adr and the conventions in AGENTS.md. Reports findings; does not change files.
---

# Stage 1 — Gatekeeper & Architecture

You report findings. You do not edit files, and you do not fix what you find — someone else decides what to do with a finding, and a reviewer who silently rewrites code is not a reviewer.

---

## Level 1: The automated gate

From `REPO_ROOT`:

```bash
make lint    # the migration self-test, gofmt, go vet, golangci-lint, govulncheck,
             # buf lint, and `npm run lint` + `npm run typecheck` in every module
make test    # go test -race in core, desktop and mobile, then the interface suites
```

`npm run lint` is ESLint carrying the house rules in `modules/tools/lint` — the
gerund, narrator and verb ratchets, the composable and filename rules, empty catches,
unreachable style rules, `ref` handling, cross-module reach, dead exports. Those
rules have their own tests (`node --test modules/tools/lint/*.test.mjs`), and so
do the story rules in `modules/tools/stories`; `modules/tools/depgraph/check.mjs`
refuses a dependency that points the wrong way.

`make test` runs `npm test` in `modules/libs/ui`, which starts both story instances at once and dies before any test executes. Run that module's suites one at a time instead:

```bash
cd modules/libs/ui
npm run test:unit
npm run build                                   # the windows reach @numen/ui through its build
vitest run --project 'stories (chromium)'
vitest run --project 'stories (webkit)'
```

Then the rest: `modules/libs/wire`, `modules/apps/desktop/editor`, `modules/apps/desktop/flashcards`, `modules/apps/mobile`.

**Auto-rejection criteria**

- `gofmt -l` prints a file → **REJECT** (unformatted Go).
- `go vet`, `golangci-lint` or `govulncheck` fails → **REJECT**.
- `npm run lint` fails → **REJECT**. A house rule refuses a thing a reader would trip over, and it is never a preference about where a line breaks.
- A pull request adds Prettier, any other formatter, or a `format` script → **REJECT.** *TypeScript and Vue are formatted by hand* forbids it, and adding one is a change to that record, taken before the code. `gofmt` is not an exception to this: it is part of Go, and `gofmt -l` printing a name is still a failure.
- A formatter or a linter has touched generated output — anything under `protocol/gen`, `protocol/src` or another generator's target → **REJECT.** A generated file is whatever generated it: not hand-formatted, not linted for layout, not read for style.
- `go test -race` fails → **REJECT** (a broken test or a data race).
- `buf lint` fails → **REJECT** (schema violation).
- `generate-check` fails → **REJECT** (the committed generated code is out of date; *A client is generated from the protocol*).
- `vue-tsc` fails → **REJECT** (type errors).
- `vitest` fails, unit or stories → **REJECT**.
- The story suites did not run in **both** engines → **REJECT the gate as incomplete.** The window is WebKit on this platform and Chromium on Windows; a review that saw one engine did not see the product (*How this application is tested*).
- The library was not rebuilt before the windows' suites → the failure is `Cannot find module '@numen/ui'`, which is the gate's own mistake. Rebuild and rerun; do not report it as a finding.

`usecase/flashcards` and `adapter/flashcardsui` run close to the ten-minute per-package limit under `-race`. A timeout there is the machine's load, not the code — rerun before calling it a failure.

`node_modules` symlinked from the primary checkout resolves `@numen/protocol` to whatever branch that checkout is parked on. A generated symbol missing there is that, not the branch under test. Install for real when the answer matters.

---

## Level 2: Architectural inspection

Read `docs/adr/README.md`, then the ADRs the touched paths relate to. Quote the ADR when a change contradicts it. The conventions are distilled in [`../../../rules/architecture.md`](../../../rules/architecture.md), [`../../../rules/coding-style-backend.md`](../../../rules/coding-style-backend.md) and [`../../../rules/coding-style-frontend.md`](../../../rules/coding-style-frontend.md); what follows is the short form.

### Go — core, adapters, the desktop host

Related: *A hexagonal core in Go*, *A client is generated from the protocol*, *One process, one lifetime*, and the glossary preamble.

- **Dependencies point inward, and the compiler proves it.** `domain/`, `port/` and `usecase/` import nothing from `adapter/` and nothing that is a driver, a framework or a transport. An import going the wrong way is a finding whatever it is for; it is always visible in an import block.
- **A port is named after the need, an adapter after the technology.** The core asks for somewhere to read a vault from; that the answer is a filesystem, and that the index is SQLite, is knowledge confined to `adapter/` and `container/`.
- **Interfaces belong to whoever needs them.** The consumer declares the interface, the implementation never names it. An interface declared next to its single implementation, or an adapter importing the port package to announce it satisfies it, is ceremony.
- **A use case is one business scenario, named as one**, and it owns the boundary of one unit of work — which is where a transaction belongs. Logic that cannot be exercised without its delivery mechanism has escaped into a handler.
- **Driving and driven adapters are both adapters.** The core must not be able to tell which is on the other side. A use case formatting output for a terminal, or an entity that knows a column name, has crossed the line.
- **Entities hold rules, not the world**: no I/O, no clock, no framework tags, no knowledge of how they are stored.
- **A repository is a collection, not a service.** Put an aggregate in, take one out, remove one. A method list that reads like a menu is the finding; searches and counts are queries beside it.
- **Constructors validate.** Sealed `New*` with required dependencies checked. `New…` for constructors, `make` for built-in allocation.
- **Nothing is logged.** No logging library in the core; an error is returned, reported through `port.Trouble`, or streamed to the interface.
- **Nothing exists that no decision asked for.** For every field parsed, column stored, syntax accepted or option offered there is a decision that wants it. A parsing rule nobody specified was invented by whoever wrote it, and once a vault is full of what it accepts the guess is permanent.
- **Do not abstract before there is a second case**, and say so equally when a genuinely needed seam is missing.

### Vue and TypeScript — the library and the windows

Related: *How an interface component is built*, *The component library is shadcn-vue*, *How this application is tested*, the glossary, *A client is generated from the protocol*.

- **`@numen/ui` knows nothing about the domain.** No component, story or fixture in `modules/libs/ui` imports `@numen/protocol`, a wire type, a domain type, or anything under `modules/apps/**`. A fixture taken from the domain is how the dependency comes back in through the door marked "tests". A prop named after a vault, a note or an RPC message is the boundary crossed even when its type is a string. The dependency runs `modules/apps/*` → `modules/libs/ui`; never back, never sideways.
- **The core is pure and the view is humble.** What is drawn is computed from the props as plain values — every coordinate, state and derived flag — with no DOM, no Vue, no clock, no randomness and no measurement of text. A number in a template that the pure part did not produce is a broken split, and so is a `computed` reaching for `window` or `Date.now`.
- **Anything with a lifetime is a port**: the clock, `requestAnimationFrame`, the viewport, the motion preference. Each is a parameter with a browser-shaped default.
- **A concept is declared once.** The type union, the layout, the wording and the colour of a role or a variant all read from one declaration.
- **A second behaviour is a function, never a mode flag** — and only where a second implementation is intended.
- **Styling is design tokens, and only design tokens.** A literal hex, a raw `rgb()`, a hardcoded duration or a Tailwind palette class in a component is a finding. Custom properties are declared on the root alone. No `dark:` variant anywhere in the module, including in a component copied in from the registry. A hairline, a border, a focus ring and the stroke of a handle stay in `px`; what follows the interface multiplier is in `rem`.
- **Business logic does not live in a `.vue` file.** The test is whether it can be exercised without mounting anything.
- **A component is split when it holds a second responsibility** — a section with its own state, or a part rendered on its own elsewhere.
- **Section banners** in the order the module's `AGENTS.md` states, with only the sections that exist.
- **Naming**: handlers `on<Action>`, emits kebab-case in the template and `onItemSelected` in the script, composables `use<Feature>` in a file of the same name. Directories `kebab-case`, components `PascalCase`, other TypeScript `camelCase`.
- **No `any`, and no `as` casting away a type the code could state.** A non-null assertion standing in for a check is the same finding.
- **A fallible operation returns `{ ok: true; value: T } | { ok: false; error: E }`.** A bare value, an implicit `null`, or a second word for the failure discriminant is a finding.
- **Code serving a single tab lives in that tab's folder.** `shared/` is for what two or more domains genuinely use.

### Everywhere

- **A function is named by an imperative verb phrase or a predicate**, and the reader is what checks the part no machine does. `modules/tools/lint` refuses a gerund, a participle, a lone preposition and a third-person verb, in both languages. What is left to a person is the word that is neither: a bare noun (`paper`, `room`, `ceiling`), an adjective or an adverb (`plainly`, `quietest`, `harder`), and a metaphor standing where an engineering term exists (`seal`, `retire`, `wake`). The rule is not written for a machine because telling `open` the imperative from `paper` the noun needs a dictionary of every English verb, and that is the reader's to hold. Say what the operation is: `measurePage`, `hasRoom`, `getPlainRune`.
- **One name for one thing** (the glossary preamble). A concept under a name nobody wrote down, or under two names in two packages, is the point at which the code and the documents start describing different systems. A field renamed across a boundary is a defect unless the glossary lists the rename.
- **A comment states the rule that holds, and stops.** Flag a comment that restates the code, cites an ADR number, or argues — `rather than X`, `instead of X`, `so that we do not…`, `otherwise somebody would…` — and any comparison with a previous implementation. Benchmark numbers live in `docs/performance.md`.
- **Anything that makes the same fact true in two places**, and any new dependency inside the core.

---

## What not to report

- Anything `gofmt`, `go vet`, `golangci-lint`, `buf lint`, `vue-tsc` or a house ESLint rule already enforces; they all run in the gate above. Naming in particular is largely machine-refused — check `modules/tools/lint` before writing a naming finding by hand.
- **A file's length, an import count, or a block size.** A file is split by responsibility.
- **Where a line breaks.** No formatter runs over TypeScript, Vue or Markdown, and that is a decision, not an omission. Layout is read like the rest of the code: a finding about it says what a reader trips over, never that a tool would have done it differently.
- Preferences with no consequence: a helper you would have written differently, an early return you find prettier, a name that is merely not your choice.
- Speculative structure. "This should be an interface in case we swap it" is a finding only when the second implementation is in sight.
- The same point twice in different words.

---

## Output integration

Every finding from this stage goes into **Stage 1** of the Unified Report in [`../SKILL.md`](../SKILL.md). Do not output a standalone report.
