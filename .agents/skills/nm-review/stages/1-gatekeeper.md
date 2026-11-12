---
name: gatekeeper
description: Stage 1 of the numen review pipeline. Runs the automated gate, then audits the diff against the conventions in AGENTS.md and the rules in .agents/rules/. Reports findings; does not change files.
---

# Stage 1 — Gatekeeper & Architecture

You report findings. You do not edit files, and you do not fix what you find — someone else decides what to do with a finding, and a reviewer who silently rewrites code is not a reviewer.

---

## Level 1: The automated gate

From `REPO_ROOT`, run **only the modular targets matching the touched modules**:

```bash
# For Go core changes:
make lib_core_lint lib_core_test

# For protocol changes:
make lib_protocol_lint lib_protocol_test lib_protocol_generate_check

# For UI changes (@numen/ui):
make lib_ui_components_lint lib_ui_components_test

# For desktop changes:
make app_desktop_lint app_desktop_test

# For mobile changes:
make app_mobile_lint app_mobile_test
```

When the change touches TypeScript, Vue, JSON or CSS, also run the formatter check, which no make target covers:

```bash
npm run format:check
```

Never run blanket checks over untouched parts of the monorepo.

**Auto-rejection criteria**

- `gofmt -l` prints a file → **REJECT** (unformatted Go).
- `go vet`, `golangci-lint` or `govulncheck` fails → **REJECT**.
- `npm run lint` fails → **REJECT**. A house rule rejects code that confuses a reader, and it is never a preference about where a line breaks.
- `npm run format:check` at the repository root reports a file → **REJECT** (not Prettier-clean). TypeScript, Vue, JSON and CSS are formatted with Prettier (`npm run format`), and its use is mandatory.
- A formatter or a linter has touched generated output — anything under `protocol/gen`, `protocol/src` or another generator's target → **REJECT.** A generated file is changed only by its generator: not hand-formatted, not linted for layout, not read for style.
- `go test` fails, or the `-race` job of CI reports a data race → **REJECT**. The race detector runs in CI only (`GO_RACE=1` turns it on for a make target); a local gate runs without it.
- `buf lint` fails → **REJECT** (schema violation).
- `generate-check` fails → **REJECT** (the committed generated code is out of date; _A client is generated from the protocol_).
- `vue-tsc` fails → **REJECT** (type errors).
- `vitest` fails, unit or stories → **REJECT**.
- The story suites did not run in **both** engines → **REJECT the gate as incomplete.** The window is WebKit on this platform and Chromium on Windows; a review in one engine does not cover the product.
- The library was not rebuilt before the windows' suites → the failure is `Cannot find module '@numen/ui'`, which means the gate was run incorrectly. Rebuild and rerun; do not report it as a finding.

`usecase/flashcards` and `adapter/flashcardsui` run close to the ten-minute per-package limit under the CI `-race` job. A timeout there is the machine's load, not the code — rerun before calling it a failure.

`node_modules` symlinked from the primary checkout resolves `@numen/protocol` to whatever branch that checkout has checked out. A generated symbol missing there is that, not the branch under test. Install for real when the answer matters.

---

## Level 2: Architectural inspection

Read `AGENTS.md` at the root and in the touched module, and the rules summarized in [`../../../rules/architecture.md`](../../../rules/architecture.md), [`../../../rules/coding-style-backend.md`](../../../rules/coding-style-backend.md) and [`../../../rules/coding-style-frontend.md`](../../../rules/coding-style-frontend.md). Quote the rule when a change contradicts it. What follows is the short form.

### Placement

Load the placement skill of each area the diff touches ([`nm-place-core`](../../nm-place-core/SKILL.md), [`nm-place-window`](../../nm-place-window/SKILL.md), [`nm-place-plugin`](../../nm-place-plugin/SKILL.md), [`nm-place-library`](../../nm-place-library/SKILL.md)) and check where the diff put its code against it: layer, slice, segment, package and library. Quote the skill when a file is in the wrong place.

### The core knows no plugin domain

The core, the core protocol and the shared SDK are a microkernel ([`architecture.md`](../../../rules/architecture.md) §11 (l)). For every package, type, RPC, message, field or export the diff adds to `libs/core/kernel`, `libs/core/protocol`, `libs/core/client`, `libs/plugin/sdk-go` or `libs/plugin/sdk-ts` outside `contracts/`, ask: "Which product domain does this know, and why is it not in a plugin or a contract?" An answer other than "none" is a finding, with the plugin or the contract it belongs in. Examples from the microkernel change:

- The list of agent programs cached in the settings adapter of the core belongs to the agent plugins, whose programs are connection types of `acp.task`.
- Model pooling, handling, the source model and the token limit in the core belong to engines-embeddings; the core knows a search source by its opaque recipe, width and chunk size.
- The `app:link.*` commands of the media importer in the shared command catalog `libs/core/protocol/commands.json` belong to the manifest of that plugin.

The same holds for an operation of connections: its name, rows, priority, queue, cancel, outcome and frames are the core's; its payload shape is a contract in `libs/plugin/sdk-go/contracts/<name>` and `libs/plugin/sdk-ts/src/contracts/<name>`. `go/dead-exports` and `ts/contracts-imports` catch the cases a structure shows; the question above catches a field or a name that one plugin needs inside a type the core already has.

### Go — core, adapters, the desktop host

- **Dependencies point inward, and the compiler proves it.** `domain/`, `port/` and `usecase/` import nothing from `adapter/` and nothing that is a driver, a framework or a transport. An import going the wrong way is a finding whatever it is for; it is always visible in an import block.
- **A port is named after the need, an adapter after the technology.** The core requires a place to read a vault from. That the place is a filesystem and that the index is SQLite is known only in `adapter/` and `container/`.
- **The consumer of an interface declares it.** The implementation never names the interface. An interface declared next to its single implementation, or an adapter importing the port package to declare that it satisfies the port, adds nothing.
- **A use case is one business scenario, named as one**, and it defines the boundary of one unit of work, which is the transaction boundary. Logic that cannot be exercised without its delivery mechanism is in a handler.
- **Driving and driven adapters are both adapters.** The core must not depend on which kind of adapter is used. A use case formatting output for a terminal, or an entity that uses a column name, violates the boundary.
- **Entities contain rules only**: no I/O, no clock, no framework tags, no knowledge of how they are stored.
- **A repository is a collection, not a service.** Put an aggregate in, take one out, remove one. A long list of methods is the finding; searches and counts are separate queries.
- **Constructors take their collaborators and return the concrete type.** No validation of collaborators, no error return, no `With*` method that changes a use case after construction. `New…` for constructors, `make` for built-in allocation.
- **Nothing is logged.** No logging library in the core; an error is returned, reported through `port.Trouble`, or streamed to the interface.
- **Everything exists because a decision requires it.** For every field parsed, column stored, syntax accepted or option offered there is a decision that requires it. A parsing rule nobody specified was invented by whoever wrote it, and once a vault is full of what it accepts the guess is permanent.
- **Do not abstract before there is a second case**, and say so equally when a genuinely needed seam is missing.

### Vue and TypeScript — the library and the windows

- **`@numen/ui` does not depend on the domain.** No component, story or fixture in `libs/ui/components` imports `@numen/protocol`, a wire type, a domain type, or anything under `apps/**`. A fixture taken from the domain brings the dependency back through test code. A prop named after a vault, a note or an RPC message is the boundary crossed even when its type is a string. The dependency runs `apps/*` → `libs/ui/components`; never back, never sideways.
- **The core is pure and the view is humble.** What is drawn is computed from the props as plain values — every coordinate, state and derived flag — with no DOM, no Vue, no clock, no randomness and no measurement of text. A number in a template that the pure part did not produce is a broken split, and so is a `computed` that reads `window` or `Date.now`.
- **Time and platform capabilities are ports**: the clock, `requestAnimationFrame`, the viewport, the motion preference. Each is a parameter with a default that uses the browser API.
- **A concept is declared once.** The type union, the layout, the wording and the colour of a role or a variant all read from one declaration.
- **A second behaviour is a function, never a mode flag** — and only where a second implementation is intended.
- **Styling is design tokens, and only design tokens.** A literal hex, a raw `rgb()`, a hardcoded duration or a Tailwind palette class in a component is a finding. Custom properties are declared on the root alone. No `dark:` variant anywhere in the module, including in a component copied in from the registry. A hairline, a border, a focus ring and the stroke of a handle stay in `px`; what scales with the interface multiplier is in `rem`.
- **Business logic is not in a `.vue` file.** The test is whether it can be exercised without mounting anything.
- **A component is split when it holds a second responsibility** — a section with its own state, or a part rendered on its own elsewhere.
- **Section banners** in the order the module's `AGENTS.md` states, with only the sections that exist.
- **Naming**: handlers `on<Action>`, emits kebab-case in the template and `onItemSelected` in the script, composables `use<Feature>` in a file of the same name. Directories `kebab-case`, components `PascalCase`, other TypeScript `camelCase`.
- **No `any`, and no `as` casting away a type the code could state.** A non-null assertion standing in for a check is the same finding.
- **A fallible operation returns `{ ok: true; value: T } | { ok: false; error: E }`.** A bare value, an implicit `null`, or a second word for the failure discriminant is a finding.
- **Code serving a single tab is in that tab's folder.** `shared/` is for what two or more domains genuinely use.

### Everywhere

- **A function is named by an imperative verb phrase or a predicate**, and the reader checks it. The reader checks for a gerund, a participle, a lone preposition and a third-person verb, and also the word that is neither: a bare noun (`paper`, `room`, `ceiling`), an adjective or an adverb (`plainly`, `quietest`, `harder`), and a metaphor standing where an engineering term exists (`seal`, `retire`, `wake`). Say what the operation is: `measurePage`, `hasRoom`, `getPlainRune`.
- **One name for one thing.** A concept is named with the standard term of its field. A concept under two names in two packages, or a field renamed across a boundary, is a finding.
- **A comment states the rule that holds, and stops.** Flag a comment that restates the code or argues — `rather than X`, `instead of X`, `so that we do not…`, `otherwise somebody would…` — and any comparison with a previous implementation. Benchmark numbers are stored in `docs/performance.md`.
- **Anything that makes the same fact true in two places**, and any new dependency inside the core.

### Read the checks themselves

A green gate proves only what the checks look at. Read the checks the diff relies on: the checks under `tools/checks` and the registry `tools/checks/rules.json`, `tools/lint-layers`, the ESLint configurations and `.golangci*.yml`. A check is sound when:

- no exemption, allow-list, named path or name list narrows what it covers;
- each threshold is the value the rules state, not a value just above the largest current object, so the code is held to the rule and not to its present size;
- it asserts that it found something to check, so a scan of an empty set or of a folder it does not walk fails;
- it decides by structure (syntax tree, import graph, type information) and covers every package, test files and generated files included, as the rules say.

A check that fails one of these is a finding on the check, reported with the rule it should enforce.

Then judge each object by what it does, not by the interfaces it is cut into. For every struct, package and file the diff adds or grows, count the methods, fields and lines against the bounds in `architecture.md` §8, and ask what the object does: one technology for an adapter, one scenario for a use case, one wire and composition for a `service` package. An object that does several jobs is a finding even when each interface it implements is small and every check is green.

### Checks are passed by changing the code

A change that makes a check pass without fixing what it reports is a finding. Look in the diff for:

- a package of type aliases (`type T = otherpkg.T`) that re-exports types of another package;
- an interface split into embedded pieces, each under the size limit, that together exceed it;
- a file or a package added to an `exclude`, `ignores` or similar entry of a lint or `tsconfig` configuration;
- a name, path or file added to a list of exceptions in a check;
- logic moved into a folder a check does not walk, such as a sub-package of `container/`.

---

## What not to report

- Anything `gofmt`, `go vet`, `golangci-lint`, `buf lint`, `vue-tsc`, Prettier or a house ESLint rule already enforces; they all run in the gate above.
- **An import count.** A file is split by responsibility; its length and the size of a package are bounded by the rules and are findings when over the bound.
- **Where a line breaks** in TypeScript, Vue, JSON or CSS. Prettier decides it, and `npm run format:check` is the check.
- Preferences with no consequence: a helper you would have written differently, an early return you find prettier, a name that is merely not your choice.
- Speculative structure. "This should be an interface in case we swap it" is a finding only when the second implementation is in sight.
- The same point twice in different words.

---

## Output integration

A failed gate goes into the gates line of the report in [`../SKILL.md`](../SKILL.md), and a rule violation becomes a finding there. Do not output a standalone report.
