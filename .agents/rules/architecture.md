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
│   ├── plugin-sdk-ts/            # TypeScript: what a plugin frontend is built from
│   ├── core-client/              # TypeScript: the client of the core for the windows and the plugin host; private
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
- **`core-client` is the first-party client of the core.** `libs/core-client` (`@numen/core-client`, private, never published) holds the service clients, the transport, the enum codes, the error words, the manifest reader and the plugin catalog client that applications and the plugin host use to talk to the core. It depends on `@numen/protocol` and re-exports the generated types applications name. No plugin imports it (`tools/lint-layers`, `package-matrix`, `core-client-private` and `dependency-graph`).
- **`plugin-host` knows Vue, the plugin SDK and the core client.** `libs/plugin-host-ts` (`@numen/plugin-host`) is the loader of plugin frontends and the build of the host modules an application serves them; it implements the plugin contract over the core client (the agent port, the text links, the texts of a plugin) and reads a manifest through it. Each application that loads plugins implements its `LoaderHost`. It imports nothing of `@numen/ui`, `@numen/shell`, `@numen/protocol` or an application, and no plugin imports it (`tools/lint-layers`, `dependency-graph`).
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
- **Slice isolation**: A slice does not import from a sibling slice on the same layer. Shared code resides in a lower layer. There is no `@x` cross-import; `dependency-graph` fails on one. A slice is reached from above through its `index.ts` alone.
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
  - In Go the rule is structural. `TestTheMachineClockIsReadOnlyByAClockPortImplementation` (`libs/core/container/clock_port_test.go`) loads every module of `go.work` with go/packages, tests included, and fails on any reference to `time.Now`, `time.Since` or `time.Until` outside the body of a method `Now() time.Time` whose receiver implements a clock port (an interface declared in non-test code whose only method is `Now() time.Time`). Generated files are skipped. No path, file or name is listed, forbidigo carries no time pattern, and the test asserts that it finds the real clock implementations. It runs in `make check_layers` (verifier `go-structure`), which `make check_gate` runs whenever a Go file changes.
  - In TypeScript the rule is structural. `no-restricted-syntax` in `tools/lint-layers/base-eslint.mjs` (`createClockSelectors`) bans `Date.now()` and `new Date()` with no arguments in every package, except inside the declaration of the function `getNow` in `libs/ui/core/src/shared/lib/dom.ts`. `performance.now()` is banned the same way, except inside the declaration of `getMonotonicNow` in `libs/ui/core/src/shared/lib/time/clock.ts`. In every `lib/` and `domain/` segment `createPureSegmentConfig` also bans `setTimeout` and `setInterval`; a stateful `model/` module that debounces or polls is not a pure segment and may schedule. Other code takes the time from `getNow` (exported by `@numen/ui`) or from an injected clock, and tests control it with fake timers. `tools/lint-layers/clock.test.mjs` proves a stray `Date.now()` fails and `getNow` passes.
- **Randomness is a port.** `math/rand` and `Math.random()` are passed in or seeded.
- **The same input gives the same output** on any machine on any day. A chunk identified by its text is the same identifier on either platform.
- **No error is swallowed.** An empty `catch {}`, a discarded `_ = fn()` and a floating `void promise` are all forbidden. An error is handled, returned or wrapped.

## 6. One name for one thing

A concept takes the standard term of its field.

## 7. Nothing exists that no decision asked for

For every field parsed, column stored, syntax accepted or option offered there is a decision that wants it. A parsing rule nobody specified was invented by whoever wrote it, and once a vault is full of what it accepts, the guess is permanent. This applies most to things that look free: a tag, an extra frontmatter key, a convenience flag.

Equally: do not abstract before there is a second case, and do not leave a genuinely needed seam missing.

## 8. Size and complexity limits

Cyclomatic complexity, nesting depth and function length are rules. Go, TypeScript and Vue take one set of limits:

| Limit                               | Maximum | Go (`.golangci-structure.yml`) | TypeScript and Vue (`tools/lint-layers/base-eslint.mjs`) |
| ----------------------------------- | ------- | ------------------------------ | -------------------------------------------------------- |
| Cyclomatic complexity of a function | 15      | `cyclop` `max-complexity`      | `complexity`                                             |
| Nesting depth of control blocks     | 4       | revive `max-control-nesting`   | `max-depth`                                              |
| Function length in lines            | 250     | `funlen` `lines`               | `max-lines-per-function`                                 |
| Function length in statements       | 50      | `funlen` `statements`          | `max-statements`                                         |
| Parameters of a function            | 10      | revive `argument-limit`        | `max-params`                                             |

A configuration may set a lower value than the table and never a higher one. The TypeScript configuration sets complexity 12, depth 3 and parameters 4 today; the Go configuration also runs `gocognit` (15), `nestif` (4) and `maintidx`. `max-control-nesting`, `argument-limit`, `max-lines-per-function` and `max-statements` are not configured yet. A Vue file also has limits for its blocks: the script block 100 lines, the template 140 lines, template depth 4, 6 inline props, 3 statements in a `computed`, and 8 properties a composable returns. Test files (`_test.go`, `*.test.ts`, `*.stories.ts`) are outside these limits. A function over a limit is split by responsibility.

File length and import counts have no limit. A file is split when it holds a second responsibility: a section with its own state, a part rendered on its own elsewhere, a concern that can be exercised separately. A folder that collects a kind rather than a thing is a finding: every entity in one file, every use case in one package, readable at five and unreadable at fifty.

## 9. Zero exceptions

A rule has no exception. There is no `nolint`, `eslint-disable`, `@ts-ignore`, `@ts-expect-error` or `@ts-nocheck` comment, no `exclusions` or `exclude-functions` in a linter configuration, no file named in an `ignores` or `files` entry, and no list of files in a check. What a rule rejects is fixed. A scope that differs is stated by a shared property: `_test.go` files are in scope, and generated code is found by its `Code generated ... DO NOT EDIT` header. `no-suppressions` in `tools/lint-layers` fails on each marker.

## 10. Rules that hold the boundaries

A check decides by structure. Where an off-the-shelf tool exists, the check is that tool with a configuration: dependency-cruiser for package boundaries and import cycles, depguard and gomodguard for Go imports and `go.mod` requires, golangci-lint linters, stylelint for style sheets. A hand-written check works only on syntax trees, import graphs and type information. A check does not match a word, a regular expression or a list of names against code or comments.

`tools/lint-layers` states each rule once, over a capture or a set derived from the repository, and declares a minimum of what it examines, so a check over nothing fails. Each rule has a violation fixture in `tools/lint-layers/checks.test.mjs`. `node tools/lint-layers/check.mjs --list` prints the registered verifiers:

- `go-structure`: every test of `libs/core/container`, the package of the structural Go tests: the layers of the core, the machine clock, the plugin dependency rule of §11 and the encapsulation of the protocol by the Go SDK.
- `notes-hexagon`: the layer tests of the root package of `libs/notes`.
- `core-depguard`: `.golangci.yml` depguard rules for the domain and use cases of `libs/core`.
- `notes-depguard`: depguard rules of `libs/notes`, which depends on nothing of this repository.
- `sdk-isolation`: the Go SDK imports no core and no application.
- `sdk-ts-encapsulates-protocol`: no exported declaration of an entry in the `exports` of `@numen/plugin-sdk` reaches a declaration of `libs/protocol`.
- `library-matrix`: every `go.mod` under `libs/` depends only on the families its row allows.
- `package-matrix`: every `package.json` under `libs/` declares only the libraries its row allows.
- `protocol-declared-only-by-clients`: only `@numen/plugin-sdk` and `@numen/core-client` declare `@numen/protocol`.
- `core-client-private`: `@numen/core-client` is private and no plugin relies on it.
- `dependency-graph`: dependency-cruiser decides package boundaries, segment direction and import cycles for libraries, applications and plugin frontends.
- `styles-use-tokens`: stylelint allows no colour literal, no `--numen-` declaration outside token and theme files, no `var()` fallback and no undeclared token.
- `all-ts-packages-eslint`: every TypeScript package has ESLint configured.
- `ui-lint`: `npm run lint` passes in `libs/ui/core`.
- `shell-lint`: `npm run lint` passes in `libs/ui/shell`.
- `plugin-host-lint`: `npm run lint` passes in `libs/plugin-host-ts`.
- `strict-lint-rules`: every package resolves `no-void`, `no-empty`, `no-empty-function`, `no-floating-promises` and `no-misused-promises`.
- `no-suppressions`: §9.
- `every-path-has-a-layer`: fails on a module no layer owns.
- `workspace-lists-every-module`: `go.work` lists every `go.mod`.
- `plugin-config-golden-parity`: plugin lint configurations match `plugins/.example/` and the root `.golangci.yml`.
- `plugin-backend-layers`: the layers inside a plugin backend point inward.
- `plugins-independence`: the plugin tests of `libs/core/container` apply the dependency rule of §11 to the type-checked import graph on 14 platform targets.
- `plugins-leakage`: manifest schema and static non-leakage.
- `plugins-use-main-entry`: the SDK and the UI library are imported through their declared entries.
- `plugins-frontend`: no `@/**` and no `@plugins/**` in a plugin frontend.
- `plugins-isolation-capture`: a file under `plugins/(X)/` references no other plugin.
- `plugins-hold-no-schema`: no `.proto` file under `plugins/(*)/`.
- `desktop-editor-fsd`: the FSD layer hierarchy of the editor (§4).
- `desktop-flashcards-fsd`: the FSD layer hierarchy of the flashcards window (§4).

Outside `tools/lint-layers`, `host_permission_test.go` holds the permission rule: the core refuses a call whose permission the plugin's manifest does not declare, for any client of the host endpoint.

## 11. Plugin model

These rules bind every change to `libs/plugin-sdk-go`, `libs/plugin-sdk-ts`, `libs/core-client`, `libs/ui`, `libs/plugin-host-ts`, `libs/plugin-manager`, `libs/core`, `apps/*` and `plugins/`.

**(a) The core routes every call.**

```mermaid
graph LR
    Window["Window (editor, flashcards, mobile)"] -->|"@numen/core-client"| Core
    Frontend["Plugin frontend"] -->|"@numen/plugin-sdk"| Core
    Core -->|"service declared in the manifest"| Backend["Plugin backend"]
    Backend -->|"libs/plugin-sdk-go, host endpoint"| Core
```

- A window of the application (editor, flashcards, mobile) reaches the core only through `@numen/core-client`.
- A plugin's frontend is part of the plugin. It reaches the core only through `@numen/plugin-sdk`.
- A plugin's backend is a process the core starts. It reaches the core only through `libs/plugin-sdk-go`.
- A window or a plugin frontend calls a plugin's service through the core. The core forwards the call to the backend whose manifest declares the service.
- A backend calls the host endpoint of the core. The core checks the call against the permissions of the plugin's manifest.
- Windows and plugins never call each other directly.
- A plugin's own service is named after its plugin id. A shared service, such as `numen.v1.AgentService`, is declared by the core, and a plugin implements it.
- Plugins are trusted code. The core runs no plugin in an iframe or a sandbox and gives a frontend no capability token. The token the core gives a backend process tells the core which plugin calls; it is not a sandbox.

**(b) A plugin depends on the SDK only.** A plugin depends on the plugin SDK (Go: `libs/plugin-sdk-go`; TypeScript: `@numen/plugin-sdk`) and on third-party code. Its frontend also depends on `@numen/ui`. A plugin never imports or requires `libs/notes`, `libs/protocol`, `@numen/core-client`, `@connectrpc` or another library of this repository. The rule is about direct requires and imports: an `// indirect` line for `libs/notes` or `libs/protocol` in a plugin's `go.mod` comes from Go, because the SDK depends on them.

**(c) The SDK contains what plugins use.** The SDK holds only code that at least one plugin imports, plus the contracts of capabilities (agent provider, embedding). It holds no concrete agent, engine or runtime (for example ONNX), and no application-side API: the catalog client, the manifest parser, the transport and the raw core clients that only windows use live in `@numen/core-client`. A type of the SDK may be an alias of a `libs/notes` type. Each type has one name in the SDK; there is no second set of names for the same types.

**(d) No protocol inside a plugin.** A plugin has no `.proto` file and no protobuf code. Its backend answers its own frontend with JSON methods (`server.Handle` and `server.Stream` of the Go SDK, `createApi` of `@numen/plugin-sdk`).

**(e) Code may be duplicated between plugins.** Code that several plugins use may be copied into each of them. No check pushes code into the SDK or `@numen/ui`; a check that does so is wrong and is corrected.

**(f) Who may use `@numen/protocol` and `@numen/core-client`.** Plugins rely on the SDK and `@numen/ui`. `@numen/core-client` and the protocol are for the windows and the SDK:

- Only `@numen/plugin-sdk` and `@numen/core-client` import or declare `@numen/protocol`. The package matrix gives `@numen/core-client` the row `[@numen/protocol]` and `@numen/plugin-host` the row `[@numen/plugin-sdk, @numen/core-client]`.
- `@numen/core-client` is private. Applications (`apps/*`) and the plugin host (`libs/plugin-host-ts`) use it; no plugin does.
- `libs/plugin-manager` is a component library. It depends on `@numen/ui` only, takes data by props and emits events. It owns the view-model logic (filters, query parsing, states) and declares a port interface for the gateway; the application implements the gateway.

**(g) The core knows no plugin by name.** Plugins are discovered through their manifests. A manifest declares what it contributes (`contributes`: `tabKinds`, `settingsViews`, `statusViews`, `themes`, `readers`, and the rest listed in `docs/plugins/README.md`). A plugin id appears nowhere in `libs/core`, `apps/*` or `libs/ui`. The plugin index is generated from the manifests at build time.

**(h) The kernel keeps defaults only.** The kernel keeps the default theme, the effective search floor and its default value. Calibration, update checks and preset themes are plugins.

**(i) Time, randomness and identifiers are ports.** Logic that depends on time takes a clock as a port, so a test controls the time. In Go a plugin takes the clock of the SDK package `clock` or declares its own; the check in §5 accepts a read of the machine clock inside any method `Now() time.Time` that implements a declared clock port. In TypeScript the time comes from `getNow()` of `@numen/ui` or from an injected clock.

**(j) Nothing is published for now.** The SDK, `@numen/ui` and `libs/notes` are built inside this repository. Each plugin is built as if it had a repository of its own (`go-structure` and `dependency-graph`). Later, when the SDK is released: publishing of the SDK and `@numen/ui`, tooling for API compatibility, and iframe isolation of plugin frontends.

**(k) Plugin kinds and storage.** A plugin has a backend kind, and the kind decides who runs it.

- `process` is the kind today: the core starts the backend as an operating-system process (`backend.type` in the manifest).
- `wasm` is a later kind: the plugin is a WebAssembly module that the core runs in its own runtime, with one compilation cache for the host, keyed by the hash of the module, that plugins neither see nor write. The manifest does not change for it yet and the core builds no support for it; `libs/plugin-sdk-go/wasm` holds the cache a plugin that embeds a WebAssembly module of its own (PDFium) keeps in its granted directory.
- Directories are granted by the host, four of them. Two are private to a plugin: a cache directory and a data directory, created before the plugin starts, readable by the user alone and never shared between plugins, at `<cache root>/plugins/<plugin id>` and `<data root>/plugins/<plugin id>`. Two are shared by every plugin: `<cache root>/shared` for what can be computed again, and `<data root>/shared` for what is downloaded or made and cannot be computed again. The application chooses the two roots from the platform's own folders.
- A plugin reads `NUMEN_PLUGIN_CACHE_DIR`, `NUMEN_PLUGIN_DATA_DIR`, `NUMEN_SHARED_CACHE_DIR` and `NUMEN_SHARED_DATA_DIR` (`cache.GetPluginDir`, `cache.GetDataDir`, `cache.GetSharedDir`, `cache.GetSharedDataDir` of the SDK; `cache.GetSharedDirOrPluginDir` and `cache.GetSharedDataDirOrPluginDataDir` fall back to the private directory when an older host grants no shared one) and never computes a platform path. A plugin that gets no directory keeps its cache in memory.
- The private cache directory holds what the plugin can compute again; the core removes it when the plugin is uninstalled or its version changes. The private data directory holds what the plugin cannot compute again and stays across updates.
- The shared directories exist so that one model is one copy on the machine, however many plugins run it. Models are in the data side, since the operating system may purge a cache and a model is gigabytes (`<data root>/shared/models`, which `NUMEN_MODELS_DIR` set by the user moves, and which the host passes to every plugin as `NUMEN_MODELS_DIR`); compiled CoreML models are in the cache side (`coreml/<model digest>/`). Entries are named by their digest. A writer puts the content in a temporary file in the same directory, checks its digest and only then renames it into place; it does not check a file again on every load. Two writers of one entry take the lock `shared.Lock` of `libs/plugin-sdk-go/shared`. A plugin never deletes what is in a shared directory and the core never deletes one on a version change; a size limit and eviction are the core's, later. The models the core provisions (`NUMEN_DEPENDENCY_*`) are kept there too. The core reads no earlier location: nothing is migrated.

### Comparison with established architectures

- **Clean Architecture dependency rule.** Source code dependencies point inward. Here the plugins are the outermost ring. The SDK is the interface between the core and the plugins. The core loads plugins by manifest and never imports them. Composition roots (`libs/core/container`, the application entry points) wire the concrete parts together.
- **VS Code extension API.** VS Code exposes a curated API surface and runs extensions in a separate extension host process. New API is added through a deliberate review. The equivalent rule here: every SDK export needs a plugin that uses it, and each plugin backend runs as its own process.

The VS Code points come from its public documentation, and the Clean Architecture points from the book and its commentary. Both are secondary sources for this repository; the rules above are binding, the comparison is context.

### Why duplication

A shared implementation library couples the plugins that use it: a change for one plugin forces a rebuild and a re-test of every other plugin that imports it. A copy changes with its plugin only. The cost of a copy is bounded by the plugin; the cost of shared-code coupling grows with every plugin added.
