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
│   ├── notes/                    # Go: what a note is and how it is written, shared with the plugins
│   │   ├── domain/               # entities, value objects, identifiers, invariants
│   │   ├── markdown/             # reading and editing a note's file
│   │   ├── textfile/             # the format of a text version's file
│   │   └── ulid/                 # the identifiers a note is given
│   ├── protocol/                 # the schema, and the Go and TypeScript generated from it
│   ├── plugin-sdk-go/            # Go: what a native plugin backend is built from
│   ├── plugin-sdk-ts/            # TypeScript: what a plugin frontend is built from
│   ├── plugin-host-ts/           # TypeScript: the loader of plugin frontends
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
- **`plugin-host` knows only Vue and the plugin SDK.** `libs/plugin-host-ts` (`@numen/plugin-host`) is the loader of plugin frontends and the build of the host modules an application serves them; each application that loads plugins implements its `LoaderHost`. It imports nothing of `@numen/ui`, `@numen/shell`, the protocol or an application, and no plugin imports it (`tools/lint-layers`, `plugin-host-isolation` and `plugins-reach-no-shell`).
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

The frontend is structured into **six FSD layers**:

$$\text{shared (0)} \longrightarrow \text{entities (1)} \longrightarrow \text{features (2)} \longrightarrow \text{widgets (3)} \longrightarrow \text{pages / screens (4)} \longrightarrow \text{app (5)}$$

- **Directionality**: A module may only import from lower architectural layers and never from higher layers.
- **Slice isolation**: A slice must not import directly from sibling slices on the same layer. Shared code must reside in a lower layer or be exposed via an explicit `@x` public interface.
- **Import aliasing**: Imports crossing slice boundaries use the `@/` alias (e.g. `@/entities/note`), while intra-slice imports remain relative (`./model`, `./types`).

Component architecture enforces separation of concerns:

- **Pure logic**: Pure functions compute rendering data (coordinates, states, flags) with no dependencies on DOM, Vue, clock, randomness, or text measurement.
- **Presentational views**: Views receive precomputed data and render elements without executing business calculations in templates.
- **Platform capabilities as ports**: Clocks, `requestAnimationFrame`, viewport geometry, and media preferences must be passed as parameters with browser defaults.
- **Design tokens**: Styling is exclusively driven by design tokens defined in `src/tokens/`.

## 5. Determinism

In anything pure — `domain/`, the model and arithmetic modules beside a component, the text cutting and chunking:

- **The clock is a port.** `time.Now()`, `Date.now()`, `new Date()`, `setTimeout` and `setInterval` are injected, never reached for. Time has a lifetime a test must control.
- **Randomness is a port.** `math/rand` and `Math.random()` are passed in or seeded.
- **The same input gives the same output** on any machine on any day. A chunk identified by its text is the same identifier on either platform.
- **No error is swallowed.** An empty `catch {}` and a discarded `_ = fn()` are both forbidden. An error is handled, returned, wrapped, or ignored with a statement saying it is.

## 6. One name for one thing

A concept takes the standard term of its field.

## 7. Nothing exists that no decision asked for

For every field parsed, column stored, syntax accepted or option offered there is a decision that wants it. A parsing rule nobody specified was invented by whoever wrote it, and once a vault is full of what it accepts, the guess is permanent. This applies most to things that look free: a tag, an extra frontmatter key, a convenience flag.

Equally: do not abstract before there is a second case, and do not leave a genuinely needed seam missing.

## 8. Size is not a rule

A file is split when it holds a second responsibility — a section with its own state, a part rendered on its own elsewhere, a concern that can be exercised separately. Line counts, import counts and block sizes are not limits here and are never a review finding. What is a finding is a folder that collects a kind rather than a thing: every entity in one file, every use case in one package, readable at five and unreadable at fifty.
