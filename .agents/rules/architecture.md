# Architecture & Repository Guidelines

The architecture of numen is written down here. This file is the map: it says where things live and which way the arrows point. When this file and the code disagree, this file is the thing to fix.

---

## 1. Monorepo organization

```text
numen/
├── Makefile                      # every gate and build; `make help` lists them
├── go.work                       # the Go modules below, used together in development
├── tests/                        # the one committed fixture vault, read-only and awkward; end-to-end suites
├── tools/                        # lint-layers, lint-quality, lint-narrative, CI scripts
├── libs/
│   ├── core/                     # Go: the hexagon
│   │   ├── port/                 # interfaces the use cases require
│   │   ├── usecase/              # one business scenario each, orchestrating ports
│   │   ├── adapter/              # cli, index, settings, window: the adapters the core exposes
│   │   ├── container/            # the composition root
│   │   └── internal/             # what nothing outside core may name; adapter/plugin is the plugin adapter
│   ├── notes/                    # Go: what a note is and how it is written; plugins reach it through the plugin SDK
│   │   ├── domain/               # entities, value objects, identifiers, invariants
│   │   ├── markdown/             # reading and editing a note's file
│   │   ├── textfile/             # the format of a text version's file
│   │   └── ulid/                 # the identifiers a note is given
│   ├── protocol/                 # the schema, and the Go and TypeScript generated from it
│   ├── plugin-sdk-go/            # Go: what a native plugin backend is built from
│   ├── plugin-sdk-ts/            # TypeScript: what a plugin frontend is built from; published
│   ├── core-client/              # TypeScript: the first-party client of the core, for applications and the plugin host; private
│   ├── plugin-host-ts/           # TypeScript: the loader of plugin frontends
│   ├── plugin-manager/           # Vue: the plugin list and the plugin page, data by props and answers by events
│   └── ui/                       # Vue 3: `core` holds components that know nothing about the domain, `shell` the window chrome
├── plugins/                      # one folder per plugin: manifest.json, backend/ (Go), frontend/ (Vue)
└── apps/
    ├── desktop/                  # the windows: Go host in cmd/ and internal/, pages in editor/ and flashcards/
    ├── mobile/                   # the phone build
    ├── landing/                  # the page the product is read about on
    └── docs/                     # the published documentation site
```

`go.work` joins the Go modules for development. Each Go module is built and tested from its own directory, which is what the Makefile targets do.

---

## 2. Dependency flow

- **Libraries never depend on applications.** `libs/*` is what `apps/*` is built from, never the reverse and never sideways.
- **`core` points inward.** `domain/`, `port/` and `usecase/` import nothing from `adapter/`, and nothing that is a driver, a framework or a transport. The violation is always visible in an import block.
- **`ui` knows nothing about the domain.** No component, story or fixture in `libs/ui/core` imports `@numen/protocol`, a wire type, a domain type, or anything under `apps/**`. A fixture taken from the domain is how the dependency comes back in through the door marked "tests".
- **`core-client` is the first-party client of the core.** `libs/core-client` (`@numen/core-client`, private, never published) holds the service clients, the transport, the enum codes, the error words, the manifest reader and the plugin catalog client that applications and the plugin host use to talk to the core. It depends on `@numen/protocol` and re-exports the generated types applications name. No plugin imports it (`tools/lint-layers`, `package-matrix`, `core-client-private` and `plugins-no-protocol-or-wire`).
- **`plugin-host` knows Vue, the plugin SDK and the core client.** `libs/plugin-host-ts` (`@numen/plugin-host`) is the loader of plugin frontends and the build of the host modules an application serves them; it implements the plugin contract over the core client (the agent port, the text links, the texts of a plugin) and reads a manifest through it. Each application that loads plugins implements its `LoaderHost`. It imports nothing of `@numen/ui`, `@numen/shell`, `@numen/protocol` or an application, and no plugin imports it (`tools/lint-layers`, `plugin-host-isolation` and `plugins-reach-no-shell`).
- **`plugin-manager` is a component library.** `libs/plugin-manager` (`@numen/plugin-manager`) draws the plugin list, the plugin page and the filters over view-model types it defines (`PluginInfo`, `PluginListing`, `ManagedPlugin`) and answers with events; it imports `@numen/ui` and nothing else. The application reads the catalog through `@numen/core-client` and maps the entries to those types (`tools/lint-layers`, `package-matrix`).
- **`protocol` is generated, not written.** What crosses between the Go host and a window is the schema's shape; `make generate-check` fails when what is committed is out of date.
- **An application is where the two sides meet.** `apps/desktop` translates the domain into the drawing vocabulary a component takes, and translates the identifier a component emits back.

## 3. Ports and adapters

- A **port is named after the need**, an **adapter after the technology**. The core asks for somewhere to read a vault from; that the answer is a filesystem, and that the index is SQLite, is knowledge confined to `adapter/` and `container/`.
- **Interfaces belong to whoever needs them.** The consumer declares the interface; the implementation never names it. An adapter importing the port package to announce it satisfies it is ceremony.
- **Driving and driven adapters are both adapters.** A CLI, a window and an MCP server drive the core; a database, a filesystem and a clock are driven by it. The core must not be able to tell which is on the other side.
- **The plugin adapter (`libs/core/internal/adapter/plugin`) has two sides, told apart by name.** A package ending in `bridge` calls a plugin for the core (`agentbridge`, `textbridge`, `agenttoolsbridge`); a package starting with `host` answers what a plugin calls on the core (`hostendpoint`, `hosttext`, `hostvault`). `bridge`, `transport` and `manifestfile` are shared. Neither side imports the other, bridges import no other bridge, no package below `internal/adapter/plugin` imports the root `plugin` package, and nothing in `internal/adapter/plugin` imports a sibling adapter. `.golangci.yml` (`plugin-*` depguard rules) and `container/plugin_adapter_layers_test.go` enforce this.
- **A use case is one business scenario**, and it owns the boundary of one unit of work — which is where a transaction belongs.
- **A repository is a collection**: put an aggregate in, take one out, remove one. Searches and counts are separate queries, not methods on the repository.
- **Entities hold rules, not the world.** No I/O, no clock, no framework tags, no knowledge of how they are stored.

## 4. The interface hexagon & Feature-Sliced Design (FSD)

The window frontends (`apps/desktop/editor`, `apps/desktop/flashcards`) are structured into **five FSD layers**. There is no `widgets` layer; a part that two pages use is a feature or an entity.

$$\text{shared (0)} \longrightarrow \text{entities (1)} \longrightarrow \text{features (2)} \longrightarrow \text{pages (3)} \longrightarrow \text{app (4)}$$

- **Directionality**: A module may only import from lower architectural layers and never from higher layers (`desktop-editor-fsd`, `desktop-flashcards-fsd`).
- **Slice isolation**: A slice does not import from a sibling slice on the same layer. Shared code resides in a lower layer. There is no `@x` cross-import; `frontend-segments-and-entities` fails on one. A slice is reached from above through its `index.ts` alone.
- **Import aliasing**: Imports crossing slice boundaries use the `@/` alias (e.g. `@/entities/note`), while intra-slice imports remain relative (`./model`, `./types`).
- **Segments**: Inside a slice the direction is `ui` → `model` → `api` → `lib`.

Component architecture enforces separation of concerns:

- **Pure logic**: Pure functions compute rendering data (coordinates, states, flags) with no dependencies on DOM, Vue, clock, randomness, or text measurement.
- **Presentational views**: Views receive precomputed data and render elements without executing business calculations in templates.
- **Platform capabilities as ports**: Clocks, `requestAnimationFrame`, viewport geometry, and media preferences are taken as parameters (§5), with browser defaults supplied by the caller.
- **Design tokens**: Styling is exclusively driven by design tokens defined in `src/tokens/`.

## 5. Determinism

In anything pure — `domain/`, the model and arithmetic modules beside a component, the text cutting and chunking:

- **The clock is a port.** `time.Now()`, `Date.now()`, `new Date()`, `setTimeout` and `setInterval` are injected as a clock the caller supplies, and the module that wraps the real one is the only place they appear. Time has a lifetime a test must control.
  - In Go the rule is structural. `TestTheMachineClockIsReadOnlyByAClockPortImplementation` (`libs/core/container/clock_port_test.go`) loads every module of `go.work` with go/packages, tests included, and fails on any reference to `time.Now`, `time.Since` or `time.Until` outside the body of a method `Now() time.Time` whose receiver implements a clock port (an interface declared in non-test code whose only method is `Now() time.Time`). Generated files are skipped. No path, file or name is listed, forbidigo carries no time pattern, and the test asserts that it finds the real clock implementations. It runs in `make check_layers` (verifier `machine-clock`), which `make check_gate` runs whenever a Go file changes.
  - In TypeScript the rule is structural. `no-restricted-syntax` in `tools/lint-layers/base-eslint.mjs` (`createClockSelectors`) bans `Date.now()` and `new Date()` with no arguments in every package, except inside the declaration of the function `getNow` in `libs/ui/core/src/shared/lib/dom.ts`. `performance.now()` is banned the same way, except inside the declaration of `getMonotonicNow` in `libs/ui/core/src/shared/lib/time/clock.ts`. In every `lib/` and `domain/` segment `createPureSegmentConfig` also bans `setTimeout` and `setInterval`; a stateful `model/` module that debounces or polls is not a pure segment and may schedule. Other code takes the time from `getNow` (exported by `@numen/ui`) or from an injected clock, and tests control it with fake timers. `tools/lint-layers/clock.test.mjs` proves a stray `Date.now()` fails and `getNow` passes.
- **Randomness is a port.** `math/rand` and `Math.random()` are passed in or seeded.
- **The same input gives the same output** on any machine on any day. A chunk identified by its text is the same identifier on either platform.
- **No error is swallowed.** An empty `catch {}`, a discarded `_ = fn()` and a floating `void promise` are all forbidden. An error is handled, returned or wrapped.

## 6. One name for one thing

A concept takes the standard term of its field.

## 7. Nothing exists that no decision asked for

For every field parsed, column stored, syntax accepted or option offered there is a decision that wants it. A parsing rule nobody specified was invented by whoever wrote it, and once a vault is full of what it accepts, the guess is permanent. This applies most to things that look free: a tag, an extra frontmatter key, a convenience flag.

Equally: do not abstract before there is a second case, and do not leave a genuinely needed seam missing.

## 8. Size is not a rule

File length, import counts and block sizes are not limits here and are never a review finding. The limits lint enforces measure branching, not size: cyclomatic complexity (12), nesting depth (3), parameter count (4) and the properties a composable returns (8). A function over one of them is split by responsibility.

A file is split when it holds a second responsibility — a section with its own state, a part rendered on its own elsewhere, a concern that can be exercised separately. What is a finding is a folder that collects a kind rather than a thing: every entity in one file, every use case in one package, readable at five and unreadable at fifty.

## 9. Zero exceptions

A rule has no exception. There is no `nolint`, `eslint-disable`, `@ts-ignore`, `@ts-expect-error` or `@ts-nocheck` comment, no `exclusions` or `exclude-functions` in a linter configuration, no file named in an `ignores` or `files` entry, and no list of files in a check. What a rule rejects is fixed. A scope that differs is stated by a shared property: `_test.go` files are in scope, and generated code is found by its `Code generated ... DO NOT EDIT` header. `no-suppressions` in `tools/lint-layers` fails on each marker.

## 10. Rules that hold the boundaries

`tools/lint-layers` states each rule once, over a capture or a set derived from the repository, and declares a minimum of what it examines, so a check over nothing fails. Each rule has a violation fixture in `tools/lint-layers/checks.test.mjs`.

- **Plugins** (`plugins-isolation-capture`, `plugins-leakage`, `plugins-no-protocol-or-wire`, `plugins-use-main-entry`, `protocol-imported-only-by-clients`, `core-client-private`, `plugins-standalone-repo`, `plugins-reach-no-app`, `plugins-reach-no-shell`, `plugins-share-no-frontend-file`): a file under `plugins/(X)/` references no other plugin; a plugin depends on the plugin SDKs, the UI library, the Go protocol module and third-party code, never on `libs/core`, `libs/notes`, the shell, the plugin host or an application, and reaches notes through the SDK. `libs/core/container/plugin_independence_test.go` states the Go dependency rule and `plugin_isolation_packages_test.go` applies it to the type-checked import graph. No file repeats a file of another plugin; shared code lives in `@numen/ui` or the plugin SDK.
- **Permissions** (`host_permission_test.go`): the core refuses a call whose permission the plugin's manifest does not declare, for any client of the host endpoint.
- **Notes** (`notes-hexagon`, `notes-depguard`, `library-matrix`): `libs/notes` depends on nothing of this repository.
- **UI** (`ui-isolation`, `shell-isolation`, `plugin-host-isolation`): `libs/ui/core` knows no domain, protocol or application; the shell reaches components through `@numen/ui`; the plugin host knows Vue, the plugin SDK and the core client only.
- **Applications** (`apps-reach-no-plugin`): an application reaches no plugin.
- **Contracts** (`protocol-holds-only-host-contracts`): every service, message and enum of `numen.v1` is used by the core, a client, an SDK, the plugin host, the UI or an application. A rule decides by the import graph, the folder layout or the shape of the public surface, and does not match a word or a list of names against code or comments.
- **Fail-closed**: `every-path-has-a-layer` and `workspace-lists-every-module` fail on a module no layer owns and on a `go.mod` that `go.work` omits.

## 11. Plugin boundary

These rules bind every change to `libs/plugin-sdk-go`, `libs/plugin-sdk-ts`, `libs/core-client`, `libs/ui`, `libs/plugin-host-ts`, `libs/plugin-manager`, `libs/core` and `plugins/`.

**(a) A plugin depends on two libraries.** A plugin depends on the plugin SDK (Go: `libs/plugin-sdk-go`; TypeScript: `@numen/plugin-sdk`) and on `@numen/ui`, plus third-party code. Plugins live in other repositories and may not use any other shared library. There are no "kits": no third shared library for plugins exists, and none is proposed. The Go SDK may depend on `libs/notes` and passes the note types, the note parser, the editor and the identifiers on to plugins through `host/domain`, `host/markdown`, `host/edit` and `host/ulid`; `libs/notes` is published and tagged together with the SDK. A plugin never imports `libs/notes` and never requires it directly in its `go.mod`; an `// indirect` line that appears because the SDK requires it is allowed (`plugins-standalone-repo`).

**(b) The SDK contains what plugins use.** The SDK holds only code that at least one plugin imports, plus the contracts of capabilities (agent provider, embedding). It knows nothing about implementations:

- no concrete agents, engines or runtimes (for example ONNX);
- no application-side API: the catalog client, the manifest parser, the transport and the raw core clients that only windows use. Those live in `@numen/core-client`.

The SDK is for plugins and is published. `@numen/core-client` is the internal first-party client of the core for applications and the plugin host; it is private and never published. A curated plugin API kept apart from the application's own access to the core is how VS Code, Obsidian and JetBrains separate extensions from the application, which is a first-party client of its core. Every SDK export has a plugin that uses it, or is the contract of a capability an application implements.

**(c) Implementation code is duplicated between plugins.** Code of one implementation family that several plugins use is copied into each of those plugins. Duplication between plugins is preferred over growth of the SDK. No check may push implementation code into a shared library; a check that does so is wrong and is corrected.

**(d) Who may use `@numen/protocol`.**

- Only `@numen/plugin-sdk` and `@numen/core-client` import or declare `@numen/protocol`. Applications (`apps/*`) and host libraries (`libs/plugin-host-ts`) reach the core through `@numen/core-client`; plugins reach it through `@numen/plugin-sdk`. The package matrix gives `@numen/core-client` the row `[@numen/protocol]` and `@numen/plugin-host` the row `[@numen/plugin-sdk, @numen/core-client]`, and no plugin imports `@numen/core-client`.
- `libs/plugin-manager` is a component library. It depends on `@numen/ui` only, takes data by props and emits events. It owns the view-model logic (filters, query parsing, states) and declares a port interface for the gateway; the application implements the gateway.

**(e) The core knows no plugin by name.** Plugins are discovered through their manifests. A manifest declares what it contributes (`contributes`: `tabKinds`, `settingsViews`, `statusViews`, `themes`, `readers`, and the rest listed in `docs/plugins/README.md`). A plugin id appears nowhere in `libs/core`, `apps/*` or `libs/ui`. The plugin index is generated from the manifests at build time.

**(f) The kernel keeps defaults only.** The kernel keeps the default theme, the effective search floor and its default value. Calibration, update checks and preset themes are plugins.

**(g) Time, randomness and identifiers are ports.** There is one sanctioned clock per language. In Go it is a method `Now() time.Time` that implements a declared clock port. In TypeScript it is `getNow()`. The structural checks in §5 enforce this.

### Comparison with established architectures

- **Clean Architecture dependency rule.** Source code dependencies point inward. Here the plugins are the outermost ring. The SDK is the published interface between the core and the plugins. The core loads plugins by manifest and never imports them. Composition roots (`libs/core/container`, the application entry points) wire the concrete parts together.
- **VS Code extension API.** VS Code exposes a curated, stable API surface and runs extensions in a separate extension host process. New API is added through a deliberate review. The equivalent rule here: every SDK export needs a plugin that uses it, and each plugin runs as its own process.

The VS Code points come from its public documentation, and the Clean Architecture points from the book and its commentary. Both are secondary sources for this repository; the rules above are binding, the comparison is context.

### Why duplication

Plugins are independent deployables in separate repositories. A shared implementation library couples their release cycles and their dependencies: a change for one plugin forces a rebuild and a re-test of every other plugin that imports it. A copy changes with its plugin only. The cost of a copy is bounded by the plugin; the cost of shared-code coupling grows with every plugin added.
