---
name: nm-place-plugin
description: Tells where the code of a Numen plugin goes and how a plugin is shaped inside (manifest, Go backend packages, frontend slices and segments, permissions, what is copied and what belongs in the SDK). Load it before adding or moving code under plugins/, libs/plugin/sdk-go, libs/plugin/sdk-ts or libs/plugin/loader, when scaffolding a plugin, and when reviewing where a plugin change put its code. Trigger with "/nm-place-plugin", "where does this go in the plugin", "new plugin", "cut the plugin backend".
paths:
  - 'plugins/**'
  - 'libs/plugin/sdk-go/**'
  - 'libs/plugin/sdk-ts/**'
  - 'libs/plugin/loader/**'
---

# Placing code in a plugin (`/nm-place-plugin`)

A plugin is a folder under `plugins/` that reaches the rest of the repository only through the plugin SDK. It does not move to a repository of its own yet: the Go SDK requires `libs/core/protocol` and `libs/note` through `replace` lines to folders, which the build of a consumer ignores, so a plugin outside this repository builds only after the SDK is published. It depends on the plugin SDK (`libs/plugin/sdk-go`, `@numen/plugin-sdk`), on `@numen/ui` in its frontend, and on third-party code. Its backend and its frontend do not import each other. Its tooling configuration files are its own copies of `plugins/.example/`. Code that two plugins need is written in both.

This skill covers what the checks cannot decide: how a plugin is cut, where a piece of code goes, what is copied. The rules the checks enforce are in `.agents/rules/architecture.md` §10 and §11 and are listed by effect in [`checks.md`](checks.md). The manifest fields, the host calls and the SDK calls are described in `docs/plugins/README.md`; this skill points to them and does not repeat them. Worked examples are in [`examples.md`](examples.md). Where the code in `plugins/` differs from what is written here, this skill is the target and the code is brought to it.

## The shape of a plugin

```text
plugins/<name>/
├── manifest.json            # identity, permissions, what the plugin contributes
├── manifest.go              # embeds manifest.json for a host that links the plugin in (plugins with a backend)
├── go.mod, go.sum           # present when there is a backend
├── package.json             # present when there is a frontend
├── Makefile, .golangci.yml, .golangci-structure.yml,
│   eslint.config.js, numen-rules.mjs, tsconfig.json   # copies of the files in plugins/.example/
├── README.md
├── testdata/api/            # JSON fixtures of the messages between frontend and backend; the tests of both sides read them
├── backend/                 # Go; package main in backend/main.go
└── frontend/                # Vue and TypeScript; entry frontend/index.ts
```

A plugin has a backend, a frontend or both. A frontend-only plugin (`plugins/notes`) has no `go.mod` and declares no `backend`. A backend-only plugin (`plugins/engines-embeddings`) declares no `frontend`.

### What the host gives a plugin

- The manifest is the contract. The core reads it without starting the plugin and knows no plugin by name. Everything a plugin contributes is a manifest entry first: `contributes.tabKinds`, `readers`, `commands`, `slots`, `handlers`, `triggers`, `themes`, `textKinds`, `noteTypes`, `listeners`, `settingsViews`, and the top-level `settings`, `secrets`, `permissions`, `dependencies`, `services`, `activationEvents`. The fields are in `docs/plugins/README.md`, "What a manifest contributes" and §3.
- The frontend receives a `PluginContext` in `activate`. It registers what the manifest declares (`register.reader`, `register.tabKind`, `register.command`, `register.slotView`, `register.settingsView`), reads `settings.get`, runs `ui.commands.run`, and uses the host services the context offers (`vault.notes`, `vault.files`, `vault.texts`, `ui.notices`, `work.tasks`, `ui.window`, `vault.vaults`). A registration for an id the manifest does not declare fails the activation.
- The backend is a process the host starts with `server.Run(service.Mount)`. `Mount(s *server.Server) error` registers its JSON methods (`s.NewAPI(service)` with `server.Handle` and `server.Stream`), its commands (`command.Mount`), its settings (`SetUpdateConfig`), its event handler (`WithEventHandler`) and what it closes (`s.OnClose`). It reaches the vault, its own store, the search index, texts, the queue and the window through the clients of `plugin/sdk-go/host` (`host.NewClientFromEnv`, `NewStoreClientFromEnv`, `NewTextClientFromEnv`, `NewQueueClientFromEnv`, and the matching `FromVars`). It reads the directories the host grants through `cache.GetPluginDir`, `GetDataDir`, `GetSharedDir` and `GetSharedDataDir`.
- The frontend reaches its backend with `createApi(service)` of `@numen/plugin-sdk`. The service name is the plugin id or starts with it, and the manifest lists it under `services`. A message is JSON whose type each side defines for itself.
- The core routes every call. A plugin never calls another plugin, a window, `libs/core/client` or the protocol. A plugin never opens the vault folder; it calls the host.
- A plugin start gets 30 seconds and is retried at its next use. The window draws at 3 seconds with the plugins loaded by then. A plugin adds no retry, backoff or failure counter of its own for its start.

### What a plugin may import

- Go: `libs/plugin/sdk-go` and its packages (`host`, `connection`, `searchsource`, `health`, `tools`, `wasm`) and the contracts (`contracts/llm`, `contracts/acp`), the standard library, third-party modules. The note types come from `plugin/sdk-go/host/domain`, `host/markdown`, `host/edit` and `host/ulid`. `go.mod` requires nothing else directly and has no `replace` to a folder; `go.work` resolves the SDK in this repository.
- TypeScript: `@numen/plugin-sdk` and `@numen/ui` through their main entries (and the entries their `exports` declare: `@numen/plugin-sdk/testing`, `/vite`, `/host-modules`, `/contracts/llm`, `/contracts/acp`), `vue`, third-party packages, and files of the plugin by relative path.
- A frontend file imports nothing from `backend/`, and a backend file imports nothing from `frontend/`. The two sides meet only in the JSON fixtures of `testdata/api/`.
- The rest of the repository stays out of a plugin: `libs/note`, `libs/core/protocol`, `@numen/core-client`, `@numen/plugin-loader`, `@numen/workbench`, `@connectrpc/*`, `@bufbuild/*`, another plugin, the core, an application, the `@/` and `@plugins/` aliases, and any `.proto` file.

## Where X goes

| I am adding                                                                          | It goes in                                                                                                                                                                                                                                |
| ------------------------------------------------------------------------------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| A capability the host resolves (`search.source`, `tools`)                            | `capabilities` in the manifest; the backend serves `numen.v1.SearchSourceService` through `plugin/sdk-go/searchsource` or `numen.v1.PluginToolService` through `plugin/sdk-go/tools` |
| An operation other plugins call (`llm.chat`, `acp.task`)                             | A connection type in `contributes.connectionTypes` that lists the operation, served through `connection.Mount` with the typed adapter of its contract (`llm.Provide`, `acp.Provide`); a new operation shape is a contract in `plugin/sdk-go/contracts/<name>` and `sdk-ts/src/contracts/<name>`, never a message of the core protocol |
| A state that stops the plugin from working (no connection, a missing model)          | A health item through `Report` of a `health.Client` (Go) or `context.health.report` (TypeScript), with an action such as `health.OpenSettings("source")`; a task reports progress only |
| A command                                                                            | One entry in `contributes.commands` (id `object.action` in camelCase, title, icon, `when`, `arguments`). The handler is `register.command` in the frontend or an entry of a `command.Registry` mounted with `command.Mount` in the backend |
| A handler of a shared command of the application                                     | `contributes.handlers` with a `when`; the handler is in the backend                                                                                                                                                                       |
| An entry in a bar, a menu or a context menu                                          | `contributes.slots` (`command`, `view` or `submenu`); a view is a Vue component registered with `register.slotView` under the same id                                                                                                      |
| A tab kind                                                                           | `contributes.tabKinds` and `register.tabKind` in `frontend/index.ts`; the tab is a slice of the frontend                                                                                                                                   |
| A reader of file types                                                               | `contributes.readers` (`id`, `contentTypes`, `rank`) and `register.reader` under the same id                                                                                                                                               |
| A setting                                                                            | `settings` in the manifest. The backend reads it in the callback of `SetUpdateConfig`, the frontend with `context.settings.get`. A setting nothing reads is removed                                                                         |
| A custom component in the settings page                                              | `settingsViews` plus `register.settingsView`                                                                                                                                                                                               |
| A key or an address of a service | A connection type in `contributes.connectionTypes` (a plugin that provides the service) or a `connections` setting (a plugin that uses one); the person keeps the key on the Connections page and the plugin never reads it. The plugin calls the service through `plugin/sdk-go/connection` or `context.work.models`; no permission is declared for it |
| A permission                                                                         | `permissions` in the manifest, for the host calls the plugin makes; added in the change that adds the first such call                                                                                                                     |
| A call of the host from the backend                                                  | A client of `plugin/sdk-go/host` built in `service.Mount` and handed to the use cases as an interface the use case declares                                                                                                               |
| A call of the host from the frontend                                                 | A member of `PluginContext` used from the `api` segment of a slice, behind a small interface of the slice so a test replaces it                                                                                                           |
| A new host call or a new member of the context                                       | The core, both SDKs and `plugin/loader`, in one change with the plugin that is its first consumer; look for an existing call first. See the SDK section                                                                                  |
| A dependency on a binary, a runtime, a model or a dataset                            | `dependencies` in the manifest (`kind`, `sourceUrl`, `checksumSha256`); the host fetches and verifies it. A model lives in the shared models directory                                                                                    |
| A background job                                                                     | A task reported to the host from the backend (`host.Queue`: `ReportTask`, `EnqueueWork`), started by a command, a trigger or an event; the host keeps the plugin running while it holds the task                                          |
| Something that runs when an event happens                                            | `activationEvents` plus `server.WithEventHandler` for host events; `contributes.triggers` for a command that runs on a file event                                                                                                         |
| A helper two plugins both need                                                       | A copy in each plugin, in the package or segment that uses it                                                                                                                                                                             |
| A helper one package uses                                                            | That package. It moves to a package of its own when a second package of the plugin imports it                                                                                                                                             |
| Pure logic (parsing, formatting, scheduling, geometry)                               | Backend: a package named after what it computes, importing nothing of the plugin. Frontend: the `lib` segment of the slice                                                                                                                |
| A message between frontend and backend                                               | A struct in `backend/api`, a type in the `api` segment of the frontend, and a JSON fixture in `testdata/api/` that the tests of both sides read                                                                                           |
| A clock, the process environment, a network or file access | An adapter in `backend/internal/adapter/<technology>`, a method of a type that implements an interface the consuming package declares                                                                                                     |
| A stylesheet                                                                         | `frontend/styles.css` using the tokens of `@numen/ui`; no literal colour or duration; sizes as `rem`/`em` literals or tokens, no `px` except a 1px hairline                                                                                                                                              |
| A text shown to a person                                                             | `frontend/words.ts` (or the words module of the slice); a component receives text, it does not compose it                                                                                                                                 |

## The backend

`backend/main.go` holds `server.Run(service.Mount)` and nothing else. Every other package is named after what it holds, never after a kind (`utils`, `common`, `helpers`, `types`, `models`), and has one responsibility.

```text
backend/
├── main.go                         # server.Run(service.Mount)
├── service/                        # the composition root and the wire
│   ├── mount.go                    # Mount: reads the environment, builds adapters and use cases, registers API, commands, settings
│   └── <method>.go                 # one handler per JSON method: decode, call a use case, encode
├── wire/<name>/                    # converts between domain or use case values and the `api` messages, and binds SDK commands and tools to use cases; imports api, domain, usecase and ports, never a machine
├── api/                            # the JSON messages of the plugin's service: structs, method names, state words; no logic
├── <domain>/                       # what the plugin knows: entities, value objects, rules (release, format, review)
├── <usecase>/                      # one scenario family per package, orchestrating ports (cards, flashcards)
├── ports/                          # interfaces the use cases require, in the vocabulary of the use case
└── internal/adapter/<technology>/  # what touches the machine or the host: host, process, http, clock, filesystem
```

A small plugin is `main.go`, `service/` and one package named after what the plugin computes. It has no `ports/`, `<usecase>/` or `<domain>/` package until a second consumer needs one. This tier is the start of every backend; the layout above is what it grows into. The rule for machine access is the same in both tiers: exec, environment, files, network and the clock are touched only in `internal/adapter/<technology>`.

The imports point inward and nowhere else.

- `service` imports everything below it, its own `internal/adapter/*` packages included, and is imported by `main` only. It is the one package that knows both the SDK `server` package and the plugin's adapters. It decodes a request, calls a use case and encodes the answer; the conversion between `api` messages and domain values is in `wire`.
- `wire` is the layer between `service` and `api`, the domain, the use cases and `ports`: the service imports it, it imports them, and nothing below it imports it. A `wire` package imports no `service`, no `engine`, no adapter and touches no machine. Scenario logic is not wire: a package that decides something goes to `usecase`.
- A use case imports `ports` and the domain. A port file imports the domain and the SDK's domain types.
- A domain package imports the standard library and `plugin/sdk-go/host/domain`. It reads no file, opens no socket, and takes no clock.
- An adapter imports the domain and, when it names the interface it satisfies, the package that declares it. It imports no `service`, no `wire` and no other adapter. An engine that runs a model is an adapter and owns its runtime as a package below it (`internal/adapter/engine/onnx`); a clock reaches an adapter as the SDK `clock` port or a parameter.
- `api` imports nothing of the plugin.
- A package named `service` or `engine` is outer; inner packages never import it.

The consumer declares an interface: one to five methods, in the vocabulary of the use case, accepted as a parameter and satisfied by a concrete type that a constructor returns. Time and identifiers arrive as parameters. The machine clock is read only in a method `Now() time.Time` of a type implementing a declared clock interface (`clock.System` of the SDK is one). Cancellation and `ctx` follow `.agents/rules/coding-style-backend.md`.

A backend with an engine of its own (a PDF, OCR or ONNX runtime) keeps the engine in a package named after what it computes (`pdf`, `recognition`), outside `service`; `service` calls it through an interface the service declares. The engine's helpers are packages named after what they compute, below or beside it.

## The frontend

```text
frontend/
├── index.ts          # activate: builds the real parts, registers what the manifest declares
├── words.ts          # the texts shown
├── types.ts          # types shared by the slices of this plugin
├── styles.css
└── <slice>/          # a thing the plugin shows or does: deck, preset, reader
    ├── index.ts      # what other files may import from the slice
    ├── ui/           # Vue components: humble views, props in, events out
    ├── model/        # composables and stateful logic: use<Feature>, create<Thing>
    ├── api/          # message types and the wire adapter over createApi or the context
    └── lib/          # pure TypeScript: parsing, formatting, derived data
```

A small frontend keeps the segments at the root of `frontend/` (`plugins/tools-updater/frontend/{api,model,lib}`, the view beside them). A larger one has a slice for each thing it shows, with the segments inside.

Inside a slice the imports run `ui` to `model` to `api` to `lib`: a segment imports the segments after it and none before it.

- `ui/` holds every `.vue` file. A component is a humble view over values it receives: the template calculates nothing, the script keeps within 100 lines and 6 inline props and follows the order Props, Emits, State, Hooks, Handlers, Helpers. It touches no `window`, `document` or `navigator`, and imports no wire file or `@connectrpc`.
- `model/` holds composables (`useDeckTabs`) and factories (`createUpdates`) that own state and call `api`. It takes the clock and measurements as parameters, and imports no component.
- `api/` holds the message types of the backend service and the wire adapter, `createApi(SERVICE)` behind an interface the model declares. It is the segment that touches host services for I/O.
- `lib/` holds pure functions over plain values. It imports no Vue, Pinia, `@vue/*`, component, or file of `ui`, `model` or `api` at any depth, and uses no `window`, `document`, `navigator`, `localStorage`, `Date.now()`, `new Date()`, `Math.random()`, `setTimeout` or `setInterval`. A type that `lib` and `api` both use is declared in `lib` or in `types.ts`, and `api` imports it.
- `frontend/index.ts` builds the real parts (the wire adapter) and passes them to the model, so a test passes fakes. `activate` registers and returns.

Two slices do not import each other, and a slice is imported only through its `index.ts`. A value two slices need is a pure function in `frontend/lib/`, a call to the backend in `frontend/api/`, a type in `frontend/types.ts`, a word in `frontend/words.ts`, or is written again in each slice. A component that two slices draw is written again in each, or lies in `@numen/ui`. What composes slices (a tab that draws a thread and a composer) lies in the root `ui/` and `model/` of the frontend, which reach a slice through its `index.ts`; a slice reaches neither. A slice holds its `index.ts` and the segments `ui`, `model`, `api`, `lib` and nothing else; `frontend/` holds `index.ts`, `words.ts`, `types.ts`, `styles.css`, the slices and the root `api`, `lib`, `model`, `ui`. The `boundaries/dependencies` rule of `plugins/.example/eslint.config.js` and the check `plugin-frontend-slices` of `tools/lint-layers` enforce this.

A component of general use belongs in `@numen/ui` (what plugins need goes into `@numen/ui`; what only applications need goes into `@numen/workbench`, which a plugin never imports), and is added there with the plugin that uses it first.

## When something belongs in the SDK, is copied, or stays in the core

- A helper that several plugins need is copied into each. The number of plugins that use it moves nothing into the SDK, `@numen/ui` or a shared package.
- The shared SDK holds the contract between a plugin and the host: a host call and its types, the lifecycle of the server, the generic connection, search source, health and tool clients, a testing harness. Its exports serve plugins of more than one family, each has a consumer outside tests (`go/dead-exports`), and it holds no engine, agent program or runtime.
- The shape of an operation is a contract in `plugin/sdk-go/contracts/<name>` and `sdk-ts/src/contracts/<name>`: protobuf messages, a typed client and a typed provider adapter. Plugins and applications import a contract; the core, the core protocol and the rest of the SDK never do (`ts/contracts-imports`).
- The SDK lacks a call when the plugin reads an environment variable the host sets without an SDK function, builds a path the host owns, calls a host service without an SDK client, or parses a format the host defines (a note, a manifest, a vault path). The plugin needs that call from the SDK; it does not copy the parsing.
- A change to the SDK reaches both SDKs: a call or field in `libs/plugin/sdk-go` is in `libs/plugin/sdk-ts` too, with the mapping in `libs/plugin/loader`. Event names are verb-object (`run-command`), booleans are `is…`, lists are plural nouns, spelling is American.
- A mechanism lands in the same change as the first plugin that uses it, and reuses an existing service, route or host call before a new one is added.
- The core keeps what is not the domain of any plugin: notes, vault, the index and all its storage, search over participants, the plugin host, the work queue, texts, settings, the generic connection mechanism, the default theme. Agents, models, calibration, update checks, readers, engines and preset themes are plugins. The core names no plugin id.
- Plugins reach each other through the core: a shared command with handlers, a capability, a connection operation on the rows of the caller's settings. A plugin does not import, start or call another.

## Which domain does it know

Ask of every new package, type, RPC or export of the core, the core protocol (`libs/core/protocol`) or the shared SDK (`libs/plugin/sdk-go`, `libs/plugin/sdk-ts` outside `contracts/`): "Which product domain does this know, and why is it not in a plugin or a contract?" The core holds notes, vaults, the index and its storage, search, text versions, windows, settings, the plugin host and the generic connection mechanism; an agent, a model, an engine or the feature of one plugin is a plugin or a contract ([`architecture.md`](../../rules/architecture.md) §11 (l)). Examples from the microkernel change:

- The list of agent programs was cached in the settings adapter of the core. An agent program is a connection type of agents-acp that serves `acp.task`; the core keeps the connection rows only.
- The core held the model fields of a search source (pooling, handling, the model it came from, the token limit). A source now describes itself by an opaque recipe, a vector width and a chunk size; the model stays in engines-embeddings.
- The shared command catalog `libs/core/protocol/commands.json` held the `app:link.*` commands of the media importer. They are commands of that plugin's manifest now.

`go/dead-exports` fails on an RPC or an SDK export that nothing outside tests uses, and `ts/contracts-imports` fails on a library outside `contracts/` that imports a contract.

## Permissions

A plugin declares in `permissions` each host capability it uses: `vault:read`, `vault:write`, `vault:manage`, `link:open` (opening external links only), `index:read`, `search:policy`, `window:focus`. It declares the permission for the calls it makes and no more. Plugins are trusted code: permissions are consent and API hygiene, not a sandbox, and `link:open` gates only `openExternal`. A manifest test states the permissions a plugin must hold when they are not obvious from its calls (`plugins/agents-acp/manifest_permissions_test.go`).

The host checks both sides. For a backend, the host endpoint authenticates the bearer token (`NUMEN_PLUGIN_TOKEN`) as a plugin id and checks the permission of that plugin's manifest on each call (`authorize` in `libs/core/kernel/adapter/plugin/hostendpoint/host_endpoint.go`; `libs/core/kernel/internal/adapter/plugin/host_permission_test.go` tests it for any client). For a frontend, `libs/plugin/loader/src/hosting/context.ts` wraps each service of the context so that a call without the permission throws (`requirePerm`).

A plugin reads and writes the vault through `host.Client`, which implements the capability interfaces (`host.FileReader`, `host.FileWriter`, `host.NoteLookup`, `host.LinkLookup` and the others), and each plugin declares the narrow one it needs. A plugin keeps its own files through `host.StoreReader` and `host.StoreWriter`, or in the directories `NUMEN_PLUGIN_DATA_DIR` and `NUMEN_PLUGIN_CACHE_DIR` name. A frontend that shows notes holds `vault:read`, and one that edits them also `vault:write`.

## Size and role

A structure is judged by what one object does, not by the interfaces it is cut into.

- The `service` package of a backend is the composition root and the wire: it builds adapters and use cases and holds one handler per method. A rule of the plugin is in a domain or use case package.
- A backend struct has the bounds of [`architecture.md`](../../rules/architecture.md) §8: adapters at most 20 methods and 15 fields, use cases at most 12 and 12. A file has at most 500 lines.
- An adapter owns one technology (host, process, http, filesystem) and holds no business rule.
- Shared code inside a plugin is a domain, a port, a use case or a package named after what it computes. A plugin does not collect reuse in a folder named by kind.
- A frontend is a set of slices. The rest of the plugin imports a slice through its `index.ts` only.
- A permission is a row in the manifest and a deny-by-default table in the host: a call with no row is refused. The plugin declares each permission its calls use.
- A value of unknown type, such as a JSON body or a host reply, is parsed once at the edge of `api` or the host client into a named type.
- One host concern has one runtime port in the plugin: the clock, the filesystem and the process each enter through one interface.
- A `catch` or an error branch handles the failure: it returns a typed error or reports it.
- Production code carries no seam that only a test mock uses.

## What the checks enforce

The complete list, with the verifier of each, is in [`checks.md`](checks.md). In short:

- A plugin's `go.mod` requires only the SDK modules and third-party code; its `package.json` names only `@numen/plugin-sdk` and `@numen/ui` among the packages of the repository; the import graph of every plugin, on fourteen platform targets, reaches no other plugin, no core, no application, no `libs/note` and no `libs/core/protocol`.
- No plugin holds a `.proto` file, a `@/` or `@plugins/` import, or an import past the main entry of the SDK or the UI library.
- The tooling configuration of a plugin is byte-identical to `plugins/.example/`.
- Inside a Go backend an inner package does not import `service` or `engine`. Inside a frontend `lib/` imports no `ui`, `model` or `api`, no Vue and no browser global; a component stays within the size limits; no component lies in `lib`, `api` or `model`.
- Functions stay within the complexity, depth, length and parameter limits; an interface has at most 5 methods; the machine clock appears only inside a method of a clock port implementation; no lint suppression comment exists.
- The manifest passes the validator; JSON keys are camelCase; every setting a manifest declares is read by the plugin; every plugin has a README.

A red check is fixed in the code, not by changing the check or moving code out of its sight.

## What no check decides

- How a backend is cut into packages. Start from the things the backend knows (a release, a card, a page, a model) and name a package after each. Put the scenarios that use them in packages of their own and the interfaces the scenarios require in `ports`. Test the cut on the import block of each package: a package that imports half of its siblings holds two responsibilities, and a package that all the others import holds a kind and is split by what it computes. Create a package when a second responsibility appears in one; a function stays in its package until another package imports it.
- When a frontend gets a slice. A slice is a thing a person sees or does that has its own state and its own backend calls. A component that shares state with its neighbour stays in the slice; a part with independent state, drawn elsewhere too, becomes a slice.
- Whether a helper is copied or the SDK lacks a call. Copy when the helper computes something of this plugin (formatting, a parser of the plugin's own format). Ask for an SDK call when the helper reproduces something the host owns (the signs are in the SDK section). Whether to add the call is decided by the user, not by the author of the plugin: write the need and the first consumer, and ask.
- A default of a setting, a shortcut, a model or a product behaviour. Report the choice; the user decides it.

Write identifiers, tests and comments in short plain technical English: functions are imperative (`getEndpoint`, `findDeck`, `createUpdates`) or predicates (`isConfigured`), tests read `TestEvaluateRejectsUnknownPlatform` or `it('opens the created file')`, and a comment states a fact the code cannot show.
