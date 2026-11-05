# Numen Plugins Developer Guide

Numen features a modular plugin architecture that decouples specialized capabilities (compute engines, document readers, AI providers) into standalone modules. This guide details how to create, test, validate, and package plugins using the developer SDK and CLI scaffolding tooling.

---

## 1. Overview & Architecture

Every plugin runs as an independent subprocess that communicates with the core host over loopback Connect-RPC HTTP/2. No application or library module imports a plugin package: the host discovers plugins on disk from their `manifest.json` and launches them.

A plugin's backend lives in the package `backend/service`, whose `Mount(s *server.Server) error` registers its services. What the plugin holds open is closed with `s.OnClose`, which runs when the host shuts the plugin down.

### The vault, through the host

A plugin never opens the vault folder. A plugin that holds `vault:read` calls `HostVaultService` on the host endpoint (`NUMEN_HOST_PORT`, authenticated by `NUMEN_PLUGIN_TOKEN`) to read a file with its file stat, to query the index for notes, links and backlinks, to read its cache, and to watch the vault for changed files. One that holds `vault:write` also writes a file over the file stat it read (a call for a file that changed since is rejected with `STALE`), makes a file, holds the vault's writers and updates the index for the files it changed. The vault is the one the window is showing; the host returns `NO_VAULT` while it shows none. A call that carries the header `Numen-Vault` uses the vault of that identity instead (`ListHostVaults` lists them, and an unknown identity returns not found). A window that shows no vault of its own uses this header. `EnsureVaultIndexed` starts reading a vault into the index and reports its progress, and `WatchMoves` reports that something moved in some vault. `plugin-sdk-go/host` is the client: `Client.With(id)` uses another vault, and `hosttest.Vault` is one held in memory for tests.

### The plugin's folder

Every plugin has a folder of its own in the store of each vault, `.numen/<plugin id>/`, for what it creates and cannot create again. It is served by `HostStoreService` on the host endpoint to every running plugin, without a vault permission, and the host serves a plugin its own folder only. A name is a file directly in the folder; one that contains a slash or begins with a dot is rejected with `OUTSIDE`. `ReadStoreFile` reads a whole file, `AppendStoreFile` adds to the end of one, creating it when it does not exist, with the content written whole or not at all, `WriteStoreFile` replaces a whole file the same way, `DeleteStoreFile` removes one, and `ListStoreFiles` lists the files of the folder with their sizes. `Numen-Vault` names the vault as it does for `HostVaultService`. The plugin also has a cache of its own for each vault (`ReadCache`, `WriteCache` of `HostVaultService`), which belongs to the installation and can be made again.

In Go a plugin uses its folder through the `host.Store` interface of `plugin-sdk-go/host`: `Read`, `Append`, `Write`, `Delete`, `List` and `With(id)` for another vault. `host.NewStoreClientFromEnv` is the implementation over the host endpoint, and `hosttest.Vault.Store()` is one held in memory for tests, with `Refuse` to make a file fail and `File` to put a file there as another program would.

### The search index and its floor

A plugin that holds `index:read` or `search:policy` calls `HostSearchService` on the host endpoint. The host keeps one effective floor for each pair of vault and embedding recipe, the similarity a passage must reach to be returned by a search by meaning, and applies it to every such search. The precedence is: a floor the query names, then the effective floor, then the starting floor the plugin of the embedding model reports (`floor_hint`), then 0.30. An embedding plugin embeds and reports its hint, and nothing else. The plugin that holds `search:policy` measures and chooses the floor.

With `index:read`, `GetEmbeddingState` returns the recipe and name of the model in use, its hint, how many chunks of the vault can carry a vector, how many do, and `is_ready` where every chunk does. It is computed from the index at each call, so a plugin that starts after the vectors exist reads the state a running plugin read when they were made. The host also emits the event `onVectorsReady:<vault id>` after each indexing pass that leaves no vector pending (`protocol.GetVectorsReadyEvent`, `host.ReadVectorsReadyEvent`); a plugin that declares `"activationEvents": ["*"]` receives it, and it repeats, so a handler reads the state again and acts on what is not done. `GetIndexStats` returns what the index of the vault holds: the recipe, name and dimension count of the model in use, how many chunks can carry a vector and how many do, how many sources the vault has, the sources and their chunks by file extension, the size of the index database and of the tables that hold vectors. It reads the counts with indexed queries and keeps them for one second, and keeps the sizes for 30 seconds, so a view may ask every few seconds. The Index stats plugin shows it in the status bar. `EmbedTexts` embeds at most 256 texts of at most 8192 bytes each with the model in use. `SampleSimilarities` takes the embedding vectors of a plugin's queries, each with the path of the note it was made from, and returns the cosine similarity of every query with the small chunks of the vault that carry a vector under the recipe and belong to another note. The host takes at most 500 small chunks in the order of their content hashes, so one index yields one sample, and at most 256 queries per call; more is `LIMIT_EXCEEDED`, a query of another dimension count than the index is `INVALID_QUERY`, and a recipe of another model is `RECIPE_MISMATCH`. The response contains one row of similarities per query, in the order of the passages, with the number of passages and of sources sampled; the format of the stored vectors does not leave the host. A plugin reads the titles of the notes it embeds through the vault calls (`vault:read`). `GetBestSimilarity` returns the cosine similarity of the passage nearest a query vector.

With `search:policy`, `GetSearchFloor` returns the floor a search applies to the vault under a recipe with the floor of each source, and `SetSearchFloor` stores it. A floor of zero removes the stored floor, a floor above one or below zero is `INVALID_FLOOR`, and setting the stored floor again changes nothing. A call that names a recipe other than the one of the model in use returns `RECIPE_MISMATCH`, and a call while the installation has no model returns `NO_MODEL`. `Numen-Vault` names the vault as it does for `HostVaultService`. While the plugin that sets the floor is disabled the host keeps applying the last stored floor. The Go client is `host.SearchClient`, behind the `host.Search` interface, and `hosttest.Search` implements the same interface in memory.

A plugin that holds `window:focus` calls `HostWindowService`. `FocusNote` makes a note the current one of the main window, `FocusPassages` opens a file there at the first of at most 8 byte ranges of its text version and highlights all of them (no range opens it at its beginning), and `ReadOpenTabs` returns the tabs the window last reported with the active one. A host with no window answers `NO_WINDOW`, a path the vault does not hold `NOT_FOUND`, and an empty path or an invalid range is an invalid argument. The Go client is `host.Client`, behind the `host.Window` interface, and `hosttest.Window` implements it in memory.

### Directories the host grants

The host creates two directories for each plugin before it starts the process, readable by the user alone and shared with no other plugin, and two that every plugin shares. It sets them in the environment: `NUMEN_PLUGIN_CACHE_DIR` for what the plugin can compute again, `NUMEN_PLUGIN_DATA_DIR` for what it cannot and the user would lose, `NUMEN_SHARED_CACHE_DIR` for what every plugin can compute again (compiled CoreML models), and `NUMEN_SHARED_DATA_DIR` for what every plugin downloads or makes and cannot compute again (models). The private ones are `<cache root>/plugins/<plugin id>` and `<data root>/plugins/<plugin id>` under the cache and data folders of the platform; the shared ones are `<cache root>/shared` and `<data root>/shared`. Models are in the data directory because the operating system may purge a cache and a model is gigabytes. The host removes a plugin's cache directory when the plugin is uninstalled or its version changes, keeps its data directory, and never deletes either shared directory. A plugin never computes a platform path: in Go it calls `cache.GetPluginDir()`, `cache.GetDataDir()`, `cache.GetSharedDir()` and `cache.GetSharedDataDir()` of `plugin-sdk-go`, which return an error where the host gave no directory, `cache.GetSharedDirOrPluginDir()` and `cache.GetSharedDataDirOrPluginDataDir()` for a host too old to share one, and `plugintest.GrantStorage(t)` grants all four in a test.

The shared directories hold one copy of what several plugins use, so a model of 8 GB is stored once and not once for each plugin. The rules are those of a content-addressed store: an entry is named by its digest (`models/` in the data directory for the models the host provisions, `coreml/<model digest>/` in the cache directory for the compiled form CoreML makes of a model); a writer writes a temporary file in the same directory, verifies the digest and renames it into place, and a reader that finds the digest uses it without hashing it again; two writers of one entry take `shared.Lock(dir)` of `plugin-sdk-go/shared`; no plugin deletes from either. The models folder is `<data root>/shared/models`, and `NUMEN_MODELS_DIR` set in the environment of the application moves it elsewhere; the host passes the folder it resolved to every plugin as `NUMEN_MODELS_DIR`. Nothing is read from an earlier location: a model in `~/.numen/dependencies` or `<config>/numen/models` is ignored and downloaded again.

### Host Lifecycle & IPC Transport

Plugins communicating over Connect-RPC use loopback TCP:

- The host supervisor launches the plugin executable with `--port=<port>` and `--token=<auth_token>`.
- The plugin process starts an HTTP/2 (`h2c`) server and announces readiness by writing to standard output:
  ```
  NUMEN_READY port=12345
  ```
- All plugins implement the `numen.v1.PluginLifecycleService` providing `Ping` and `Shutdown` endpoints. `Ping` returns the protocol version the plugin uses; the Go SDK fills it in.
- After the handshake the host calls `Ping`. A plugin whose protocol differs from its manifest's `protocol` or from the host's is stopped at once and reported as failed.
- The host stops every plugin process after 180 seconds without a lease. A lease is held for each call into the plugin that has not returned, for each open stream, and for each of the plugin's own pending jobs: a plugin stays running by reporting its work to the host as a task (`HostQueueService.ReportTask`, until the task counts to its total, fails, or `FinishTask` ends it) or by holding a queue turn (`HostQueueService.EnqueueWork`, until the stream ends). Each open task id holds one lease. The host drops the leases of a plugin whose process exits. The host sets no CPU or memory limit on a plugin process.

---

## What a plugin may depend on

- The core routes every call. A window of the application (editor, flashcards, mobile) calls the core through `@numen/core-client`. A plugin's frontend calls it through `@numen/plugin-sdk`, and a plugin's backend through `plugin-sdk-go`. A window or a plugin frontend calls a plugin's service through the core, and the core forwards the call to the backend whose manifest declares the service. A backend calls the host endpoint, where the core checks the permissions of its manifest. Windows and plugins never call each other directly.
- A plugin depends on the plugin SDK (`plugin-sdk-go`, `@numen/plugin-sdk`) and third-party code, and its frontend also on `@numen/ui`. A plugin never imports or requires `libs/notes`, `libs/protocol`, `@numen/core-client`, `@connectrpc` or another library of this repository. An `// indirect` line for `libs/notes` or `libs/protocol` in a plugin's `go.mod` comes from Go, because the SDK depends on them.
- The SDK contains what plugins use, plus the contracts of capabilities (agent provider, embedding). It contains no concrete agent, engine or runtime, and no application-side API (catalog client, manifest parser, transport, raw core clients that only windows use). Those are in `@numen/core-client` (`libs/core-client`), which is private.
- The Go SDK exposes no generated protocol type. A type of the SDK may be an alias of a `libs/notes` type, with one name per type. The SDK builds the Connect handlers and clients itself and converts between its types and the wire messages in unexported code. `server.NewLifecycleClient`, `embedding.NewClient` and `tools.NewClient` are the clients for plugin tests.
- A plugin has no `.proto` file and no protobuf code. Its backend serves its frontend with JSON methods (see `services` in §3).
- Code that several plugins use may be duplicated in each of them. No check requires moving code into the SDK or `@numen/ui`.
- Logic that depends on time takes a clock as a port, so a test controls the time. A Go plugin takes `clock.System` and `clock.Fake` of `plugin-sdk-go/clock` or declares its own clock.
- Plugins are trusted code. The core runs no plugin in an iframe or a sandbox. The token the core gives a backend process identifies the calling plugin to the core.
- Nothing is published for now. Each plugin is built as if it had a repository of its own. Publishing, API compatibility tooling and iframe isolation come later, when the SDK is released.

The full rules are in [`.agents/rules/architecture.md`](../../.agents/rules/architecture.md) §11.

### What a manifest contributes

The core discovers plugins from their manifests and has no code that names a plugin. The manifest declares:

| Field                      | Where                  | What it declares                                                                                                                                              |
| -------------------------- | ---------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `contributes.tabKinds`     | `ManifestContributes`  | Kinds of tab the frontend registers: `kind`, `title`, `icon`.                                                                                                 |
| `contributes.feature`      | `ManifestContributes`  | What the plugin provides, as text for the placeholder of an unavailable tab.                                                                                  |
| `contributes.category`     | `ManifestContributes`  | The group the plugin is listed in.                                                                                                                            |
| `contributes.themes`       | `ManifestContributes`  | Stylesheets added to the list of themes: `id`, `name`, `appearance`, `file`.                                                                                  |
| `contributes.statusViews`  | `ManifestContributes`  | Components drawn in the status bar: `id`.                                                                                                                     |
| `contributes.handles`      | `ManifestContributes`  | Capabilities the plugin takes part in for files and notes of some types: `purpose`, `contentTypes`, `noteTypes`.                                              |
| `contributes.readers`      | `ManifestContributes`  | Tabs that open files of some types: `id`, `contentTypes`, `rank`.                                                                                             |
| `contributes.noteTypes`    | `ManifestContributes`  | Types of note the plugin defines: `type`, `chunks`, `headings`, `idHeadingLevels`, `idPattern`.                                                               |
| `contributes.frameOrigins` | `ManifestContributes`  | The https origins the plugin's pages frame.                                                                                                                   |
| `contributes.commands`     | `ManifestContributes`  | Every command the plugin declares, whichever part runs it: `id`, `title`, `icon`, `scope`, `shortcut`, and for a backend command `contentTypes`, `arguments`. |
| `contributes.menus`        | `ManifestContributes`  | Where a command is shown in the window: `location`, `command`, `group`, `order` (see [Menus](#menus)).                                                        |
| `settingsViews`            | `Manifest` (top level) | Components mounted in the plugin's settings form: `id`, `title`.                                                                                              |

The schema is `libs/protocol/proto/numen/v1/manifest.proto`; the Go reader is `libs/plugin-catalog/manifest`.

---

## 2. Plugin Directory Layout

Every plugin follows a standard layout:

```text
plugins/<name>/
├── manifest.json       # Metadata, capabilities, activation events, permissions
├── manifest.go         # manifest.json embedded for Go
├── Makefile            # Build and test recipes
├── README.md           # Documentation
├── backend/            # Native Connect-RPC server (Go)
│   ├── main.go         # Process entrypoint
│   └── service/        # Service implementation and mount registration
│       └── server.go
├── go.mod              # Isolated Go module dependencies
├── go.sum
└── frontend/           # Vue 3 / TypeScript UI (optional)
    ├── index.ts        # Client activation entrypoint
    ├── Viewer.vue      # View component
    ├── styles.css      # CSS styling
    ├── package.json    # Frontend dependencies
    └── vite.config.ts  # Vite build configuration
```

---

## 3. The Plugin Manifest (`manifest.json`)

The manifest declares identity, activation rules, required permissions, and service contracts.

```json
{
  "id": "md.numen.sample",
  "name": "Sample Reader",
  "version": "2026.10.1",
  "protocol": 1,
  "description": "Custom document viewer plugin for Numen",
  "capabilities": [],
  "targets": ["desktop.editor"],
  "contributes": {
    "readers": [{ "id": "sample-viewer", "contentTypes": ["application/pdf"], "rank": "alternate" }]
  },
  "permissions": ["vault:read"],
  "backend": { "entrypoint": "bin/sample" },
  "services": ["md.numen.sample"],
  "frontend": {},
  "settings": [
    {
      "key": "zoomDefault",
      "type": "range",
      "title": "Default Zoom",
      "description": "Initial zoom level for documents",
      "default": 100,
      "min": 50,
      "max": 200,
      "step": 10
    }
  ]
}
```

### Manifest Fields

- **`id`**: Unique identifier of two or more dot-separated segments of letters, digits, `_` or `-`; reverse-DNS (`md.numen.<plugin-name>`).
- **`name`**: User-facing display title.
- **`version`**: `<year>.<month>.<build>` (`2026.10.3`): the year and month of the change, and its number within that month, starting at 1. A change to a plugin raises its version once relative to the merge base with main, however many commits the change has: in the same month `build` goes up by one, in a new month the version starts at `<year>.<month>.1`. A new plugin starts at `<year>.<month>.1` of the month it arrives in, and an unchanged plugin keeps its version; CI fails any other number. `make plugin_versions_bump` writes the next version and CI (`make plugin_versions_check`) fails a changed plugin that keeps its version; changes to markdown and test files do not count. A plugin with a `package.json` has the same version there, and CI fails when the two differ. The host reads the value as semver (`X.Y.Z`, optional pre-release and build parts).
- **`protocol`**: The plugin protocol version the plugin was written against. The host uses protocol 1 and rejects a manifest that names a newer one. Absent means 1.
- **`capabilities`**: The capabilities the host resolves a plugin by (`embedding`, `tools` and `ai.agent.provider`, the marker of an agent plugin that serves tasks and lists the programs it runs). The host refuses any other name when it reads the manifest.
- **`targets`**: Allowed client environments (`desktop.editor`, `desktop.flashcards`).
- **`activationEvents`**: The events the plugin subscribes to; the only value is `*`. With `*` the host activates the plugin on every event it emits and hands it the event through `DeliverEvent` when it serves `numen.v1.PluginEventService`; a plugin built on the Go SDK serves it when it passes `server.WithEventHandler`. A plugin that rejects an event returns `error` set to `PLUGIN_EVENT_ERROR_CODE_REFUSED` and its `reason`, and the host reports the rejection with the event and the plugin id. The host emits `onFile:<extension>` when a window opens a file and `onVectorsReady:<vault id>` after an indexing pass leaves no vector pending. A plugin with a backend that does not declare `*` is handed only `onFile:<extension>`, for a file whose content type one of its `contributes.readers` or `contributes.handles` takes. A manifest with any other value is refused.
  - `*` (every event)
- **`contributes.readers`**: The tabs of the frontend that open files, each `{ "id", "contentTypes", "rank" }`: the id the frontend registers the reader under with `registerReader`, the types it opens, and `default` or `alternate`. A type is a MIME type or a mask: `audio/*` is every audio type, `*/*` every type. The core determines the type of a file from a fixed snapshot of the `globs` table of the freedesktop.org shared-mime-info database, by the name of the file, with the extension compared without regard to case; a file whose name the table does not know is `application/octet-stream`. A file opens in the `default` reader of its type; where two plugins declare one, the reader of the narrower type is used (`audio/mpeg` over `audio/*`), then the plugin with the lower id. An `alternate` reader never opens a file on its own. A plugin with readers has a `frontend`.
- **`settingsViews`**: The components of the frontend that the plugin's settings page mounts inside the plugin's settings form, each `{ "id", "title" }`: the id the frontend registers the view under with `registerSettingsView` and its title. A view whose plugin is disabled or whose frontend did not register it draws nothing. A plugin with settings views has a `frontend`.
- **`contributes.statusViews`**: The components of the frontend that the editor's status bar draws at its trailing end, each `{ "id" }`: the id the frontend registers the view under with `registerStatusView`. A view whose plugin is disabled or whose frontend did not register it draws nothing. A plugin with status views has a `frontend`.
- **`permissions`**: Whitelisted resource access: `vault:read`, `vault:write`, `network:outbound` (`network` is accepted as an alias), `ai:complete` (ask the host for a model completion), `index:read` (read the note titles, the stored vectors and the embedding state of the search index, and embed texts with the model in use), `search:policy` (read and set the floor that a search by meaning applies) and `window:focus` (move the main window of the vault to a note or a file, and read the tabs it has open). Any other value is rejected. Permissions are checked by the host when a plugin calls it; the host does not sandbox the plugin process, so a plugin runs with the user rights of the host.
- **`dependencies`**: What the host provisions for the plugin, each `{ "id", "kind", "version", "sourceUrl", "checksumSha256", "metadata" }`. The `kind` is `runtime`, `model`, `binary` or `dataset`: the host fetches the artifact from `sourceUrl`, verifies it against the checksum, and provides its location to the plugin. A dependency with no `sourceUrl` is one the host provides itself. A dependency of kind `runtime` is a library the plugin loads; the host sets the variables that locate it. Of those, the ONNX Runtime (`{ "id": "onnxruntime", "kind": "runtime" }`) launches the plugin with `NUMEN_ONNX_RUNTIME_LIB` (the absolute path of the library) and `NUMEN_ONNX_PROVIDER` (`auto`, `cpu`, `coreml`, `cuda`, `directml` or `rocm`) in its environment, and it opens its sessions with them. It is also launched with the variables of the host that say where the loader finds shared libraries and which accelerator devices it may use (`LD_LIBRARY_PATH`, `CUDA_VISIBLE_DEVICES`, `CUDA_PATH` and `CUDA_PATH_V*`, `CUDA_HOME`, `NVIDIA_VISIBLE_DEVICES`, `NVIDIA_DRIVER_CAPABILITIES`, `ROCR_VISIBLE_DEVICES`, `HIP_VISIBLE_DEVICES`, and `DYLD_LIBRARY_PATH` and `DYLD_FALLBACK_LIBRARY_PATH` on macOS), which a plugin without a runtime does not get; `PATH` is set for every plugin. If the host cannot provision a dependency the plugin is not launched, and the activation fails with the reason. A dependency an option of a setting names in `dependencies` is provisioned only while that option is in force.
- **`backend`**: `entrypoint` is required: the executable the host starts as a process. A plugin without a backend is never started: its manifest declares no `activationEvents`, `services`, `secrets`, dependencies of kind `runtime` or `contributes.handles`, and the host rejects a manifest that declares them when it reads it.
- **`frontend`**: an empty object, `{}`, that states the plugin has a frontend. The bundle is `dist/index.js` and the stylesheet `dist/styles.css` of the plugin folder, which the plugin build writes. The window loads both from the plugin route `/plugin/<id>/`, which serves a file of the plugin folder where one exists and forwards every other path to the plugin backend, and compiles nothing of a plugin into itself. A frontend bundle leaves `vue`, `@numen/plugin-sdk` and `@numen/ui` out and imports them from the paths `HOST_MODULES` in `@numen/plugin-sdk/host-modules` names (`/sdk/vue.js`, `/sdk/plugin-sdk.js`, `/sdk/ui.js`), where the window serves its own instances. The loader of plugin frontends and the build of those host modules are `@numen/plugin-host` (`libs/plugin-host-ts`), which every application that loads plugins uses. A window loads only the plugins that declare a frontend.
- **`settings`**: Declarative settings schema automatically rendered in host settings and stored by the core. The host passes the settings to the plugin in `Configure`, once it is running and again on every change; they are not in its environment. A setting has a `key`, a `type` (`boolean`, `string`, `number`, `enum`, `range` or `file`), a `title`, a `description` and a `default`; `enum` takes `options` (a `string` setting may take them too, as suggestions the person can also type past), `range` and `number` take `min`, `max` and `step`, each absent until the manifest sets it. An option is a value or an object `{ "value", "title", "detail", "dependencies" }`: the title is what the person reads, the detail is shown under the setting while the option is in force, and `dependencies` names dependencies of the manifest the host provisions only while the option is in force (the plugin is stopped when the set changes, and launched with the new set on its next call). An `enum` takes `optionsBy: { "setting": "<key>", "options": { "<value>": [options] } }` in place of `options` to offer, under each value of an earlier enum setting, its own list; the first option of a list is the default under that value, and a stored value that is not in the list is replaced by it. A setting of one option is shown fixed, with no choice. `when: { "setting": "<key>", "is": ["<value>"] }` shows a setting only while an earlier setting holds one of the values; a setting whose condition fails keeps its stored value and is not handed to the plugin. A `file` setting takes `accept` (extensions with their dot) and `multiple: true`, which makes it hold a list of paths, one per line, that the settings page draws as one row each with an action to add another; the window offers the machine's own picker, and the host refuses a path that does not exist, is a folder, or has another extension. The settings page shows for an enum whether its value is the default, the default of the setting it follows, or the person's choice. `storedSecrets` may not be a key: the host keeps it.
- **`services`**: The service names the plugin serves to the windows, and the only place a manifest names a service: each a dotted name with no slash that is the plugin id or starts with it and a dot (`md.numen.views.flashcards.studio`), or is a service the protocol declares and several plugins implement (`numen.v1.AgentService`); the manifest reader rejects any other name. A name is the first segment of the URL path, so a window that posts to `/md.numen.views.flashcards.studio/CreateDeck` has the host forward the call to the plugin with the token the plugin uses to identify the host, whole and unchanged, streams included; the host activates the plugin for the length of the call, and a service the host serves itself is never forwarded. A plugin mounts the methods of a service with `s.NewAPI("md.numen.views.flashcards.studio")` of `plugin-sdk-go`: `server.Handle` serves a method with JSON (the request and response types are structs in the plugin, with `json` tags in lowerCamelCase), `server.Stream` serves a method as Server-Sent Events, a method fails with `server.NewAPIError(code, message)`, and `server.GetHeader(ctx)` returns the request headers such as `Numen-Vault`. The frontend of the plugin calls them with `createApi("md.numen.views.flashcards.studio")` of `@numen/plugin-sdk`. The plugin has no schema file and no generated code; its request and response types are written in Go and in TypeScript, and a fixture under `testdata/api/` that both a Go test and a TypeScript test read keeps the two consistent.
- **`secrets`**: The secrets, such as an API key, that the host stores for the plugin, each `{ "name", "title", "description", "when" }`: the name (letters, digits, `_` and `-`; a name is not also a setting) is what the plugin reads the key under, the title what the person reads, and `when` shows the secret, and hands it to the plugin, under the same condition a setting takes. The host keeps them in the system keychain, never in the settings file, and passes each to the plugin as a `Configure` value under its name: an empty value when no key is stored, which clears the key the plugin holds. A secret the host cannot read is left out of that `Configure` call, and the plugin keeps the key it holds. A secret is never in the environment or on the command line. The host records the names of the secrets that have a stored key under `storedSecrets` in the plugin's settings (names only; neither a setting nor a secret may take that name) and reads the keychain only for those, so a plugin with no stored key causes no keychain access. On a machine without a keychain the plugin is configured with the secret left out, and the host reports it once. A plugin reads a secret from the `Configure` values only, with `server.ReadSecret` of `plugin-sdk-go` (`server.SecretAPIKey` names the `apiKey` secret), and reads no key from its environment.
- **`contributes.commands`**: Every command the plugin declares, whichever part of the plugin runs it (see [Commands](#commands)). A command has an `id` as an object and an action, each in camelCase, with a dot between them (`cardModel.create`), a `title`, an `icon` (the name of an icon of the Lucide set; a name the set does not have is drawn as a placeholder), `scope` (`note` for a command listed only while a note is in front), `shortcut` (the keystroke the plugin suggests, such as `mod+shift+e`) and, for a command a backend runs, argument `name`s of lowercase words joined by hyphens (`deck-title`), `contentTypes` (types or masks, `*/*` for a file of any type) when it runs over a file, and `arguments`, each with a `name`, a `title`, a `type` (`string`, `number` or `boolean`) and `required`. The palette asks only the required arguments, so an optional one is for a caller that is not a person, such as `numen-cli`; an argument a person must be able to give is declared required. `when` is a list of conditions on the context a window supplies, `[{ "key", "is" }]`: a condition holds in a context that has the key with one of the values in `is` (`*` is any value), and a command is listed over each context in which every condition holds. A command that names conditions runs over any file, names no `contentTypes` or `noteTypes`, and is listed by the menu of the file instead of the palette. The menu of a file supplies one context for each text version of the file, with the keys `version.live` (`true` for the version displayed now, else `false`), `version.kind` (the kind of the version, as the plugin that made it names it) and `version.plugin` (the id of the plugin that made it): `[{ "key": "version.live", "is": ["true"] }]` lists a command over the live version, and `[{ "key": "version.plugin", "is": ["<the plugin's own id>"] }]` over each version the plugin made. The host gives the command the id of the version as its argument `version-id`. A plugin does not use a kind that another plugin defines. A command over a file with no required argument is listed by the menu of the file of one of its `contentTypes`. A plugin with no frontend and no backend may declare commands; they are never listed.
- **`contributes.menus`**: The places the window draws a command of the plugin in, each `{ "location", "command", "group", "order" }` (see [Menus](#menus)). The locations are `titleBar` and `searchResult`; `command` is the short id of a command the manifest declares or the full id of an editor command; `group` and `order` are optional.
- **`contributes.handles`**: The capabilities the plugin takes part in for files and notes of some types, each `{ "purpose", "contentTypes", "noteTypes" }`. The `purpose` is the id of the capability, defined by a part of the application; `contentTypes` are types or masks of files and `noteTypes` the plugin's own note types of notes. A purpose is declared once, and a plugin with handles has a `backend`. The application defines these purposes:
  - `text-provider`: the plugin provides text versions of vault files. `contentTypes`, where it names any, are the files whose text the host requests from the plugin when they have no live text version; the plugin then serves `numen.v1.TextProviderService`. A plugin that makes its text when a command runs declares the handle with no type, and the commands that work its versions declare conditions under `when` (see [Commands](#commands)).
  - `playback-copy`: `contentTypes` names the files for which the plugin keeps a copy to play, and `noteTypes` the notes for which it keeps one (`{ "purpose": "playback-copy", "noteTypes": ["link"] }`); no plugin copies an ordinary note. The copy is a file of the plugin's store folder named by the content hash of the file and an extension of an audio or video type (`<sha256>.mp4`). `MediaService.GetMediaSource` of such a file returns the address the copy is played from, and the file itself where no copy is kept. `contributes.commands[].noteTypes` selects notes by type the same way.
- **`contributes.noteTypes`**: The types of note the plugin defines, each `{ "type", "chunks", "headings", "idHeadingLevels", "idPattern" }`, and how the index keeps the notes of each: `chunks` states whether the prose is split into chunks and searched by meaning (it is, when omitted), `headings` lists the levels of heading the index keeps (every level, when omitted), `idHeadingLevels` the levels whose headings end in a block id, and `idPattern` the pattern a block id matches whole. A note of a type no plugin declares is kept as an ordinary note is.
- **`contributes.frameOrigins`**: The https origins the plugin's pages frame (`["https://www.youtube-nocookie.com"]`). The window's `frame-src` names them while the plugin is enabled. A page that frames a site checking the Referer is loaded from the loopback socket at the address `MediaService.GetPageSource` returns, under a policy that frames the plugin's origins alone.

A key that the schema does not name is not an error: the manifest reader leaves it out and reports a warning that names its path (`unknown key "settings[0].mood"`). The catalog entry of the plugin contains the warnings.

---

## 4. Frontend Plugin SDK (`@numen/plugin-sdk`)

The `@numen/plugin-sdk` library provides strict TypeScript typing, lifecycle hooks, and mock testing harnesses.

### Plugin Lifecycle

```ts
import { definePlugin, type PluginContext } from '@numen/plugin-sdk'
import SampleViewer from './Viewer.vue'

export default definePlugin({
  activate(context: PluginContext) {
    // Register the reader the manifest declares under `contributes.readers`
    context.registerReader({
      id: 'sample-viewer',
      component: SampleViewer,
      title: 'Sample Document Viewer',
    })

    // Bind the handler of the command the manifest declares under `contributes.commands`
    context.registerCommand({
      id: 'document.inspect',
      handler: () => {
        const zoom = context.getSetting<number>('zoomDefault') ?? 100
        console.log('Inspecting with zoom', zoom)
      },
    })
  },

  deactivate() {
    // Cleanup state
  },
})
```

### Settings Views

`context.registerSettingsView({ id, component })` registers the Vue component the settings page mounts for the `settingsViews` entry with the same id. The component takes the props of `SettingsViewProps`: `pluginId` and `values`, the current value of every declared setting by key. It emits `update:setting` with a key and a value, and the host writes that value to the setting `<key>` in the object named by `<pluginId>`. `createMockPluginContext` records registrations for `getRegisteredSettingsViews()`.

### Status Bar Views

`context.registerStatusView({ id, component })` registers the Vue component the editor's status bar draws for the `contributes.statusViews` entry with the same id. The component takes the props of `StatusViewProps`: `pluginId`. The views of all plugins are drawn in the order the plugins registered them. `createMockPluginContext` records registrations for `getRegisteredStatusViews()`.

### The Application and the System Browser

`context.app` is `{ version, build, platform }` of the application build the plugin runs in: the version it calls itself, its build number (0 where it has none) and `darwin`, `windows` or `linux`. `context.openExternal(url)` opens an `http` or `https` address in the system browser and returns a `Result`; any other address fails. `createMockPluginContext` takes `app` and records the addresses opened for `getOpenedAddresses()`.

### Unit Testing with Mock Harness

```ts
import { describe, expect, it } from 'vitest'
import { createMockPluginContext } from '@numen/plugin-sdk/testing'
import plugin from './index'

describe('sample plugin', () => {
  it('registers reader on activation', async () => {
    const ctx = createMockPluginContext({ pluginId: 'md.numen.sample' })
    await plugin.activate(ctx)

    const readers = ctx.getRegisteredReaders()
    expect(readers).toHaveLength(1)
    expect(readers[0].id).toBe('sample-viewer')
  })
})
```

---

## 5. Plugin Developer CLI Tooling

Numen provides CLI commands available via `numen plugin`, `numen-cli plugin`, `numen-plugin`, and `make`:

### 1. Scaffolding (`create`)

Generate a new plugin boilerplate:

```bash
# Using Makefile:
make create_plugin name=sample template=reader

# Or using CLI:
numen plugin create my-plugin --type=native --template=reader
numen plugin create ui-plugin --type=frontend
```

Supported templates:

- `reader`: Complete document viewer with Connect-RPC backend and Vue 3 frontend.
- `compute`: Headless backend computation service (Go Connect-RPC).
- `agent`: AI agent integration with backend service and client controls.

### 2. Validation (`validate`)

Verify `manifest.json` with the same validator the host runs at discovery (identity, protocol, permissions, settings), and ensure backend/frontend entrypoints resolve:

```bash
numen plugin validate plugins/sample
```

Output:

```text
plugin at plugins/sample is valid
```

If validation fails, each rule violation is reported:

```text
validation errors:
  - invalid permission: unrestricted:root
  - plugin version is not valid semver (e.g. 1.0.0): "1.0"
```

### 3. Packaging (`package`)

Bundle the plugin into a distributable archive ready for installation:

```bash
numen plugin package plugins/sample
```

Output:

```text
Bundled 12 files into plugins/sample/dist/md.numen.sample-0.1.0.zip
packaged plugin: plugins/sample/dist/md.numen.sample-0.1.0.zip
```

---

## 6. Developing & Testing Workflow

1. **Scaffold plugin**:
   ```bash
   make create_plugin name=epub-reader template=reader
   ```
2. **Build, lint, test**: every plugin's `Makefile` has the same four targets, `build`, `lint` (Go vet and golangci-lint, ESLint and the frontend's typecheck), `test` and `clean`, over the backend and the frontend it has. The root targets `plugins_build`, `plugins_lint`, `plugins_test` and `plugins_clean` run them in every directory under `plugins/` that holds a `manifest.json`.
   ```bash
   cd plugins/epub-reader
   make build
   make lint
   make test
   ```
3. **Validate manifest**:
   ```bash
   numen plugin validate .
   ```
4. **Package for distribution**:
   ```bash
   numen plugin package .
   ```

## Commands

Every command is filed in the window under `<owner>:<object>.<action>`. The owner is the id of the plugin that provides the command, or `app` for a command of the editor itself (`app:note.open`, `app:tab.close`). A plugin writes only the part after the colon, as `<object>.<action>` with each part in camelCase (`tree.revealNote`), in `contributes.commands` of its manifest and in `registerCommand` of its frontend, and the host adds the owner. A command registered twice under one full id makes the activation of its plugin fail with the full id in the message. The title is a separate imperative phrase that says what the command does (`Create deck`).

A plugin declares every command once, in `contributes.commands` of the manifest, whichever part of the plugin runs it. The declaration has no field that names the part: the application learns that from where the handler is registered. A command the frontend runs has its handler bound in the plugin's frontend with `registerCommand({ id, handler })`, and a command the backend runs is served by the plugin's backend. The title, icon, scope and shortcut are read from the declaration, never from the code.

```json
"contributes": {
  "commands": [
    {
      "id": "tree.open",
      "title": "Open file tree",
      "icon": "folder-tree",
      "shortcut": "mod+shift+e"
    },
    {
      "id": "deck.create",
      "title": "Create deck",
      "icon": "layers",
      "arguments": [{ "name": "title", "title": "Title", "type": "string", "required": true }]
    }
  ]
}
```

A command is listed only when it is declared and something can run it: its frontend registered a handler, or its plugin has a backend and no handler of the frontend takes it. A handler registered for an id the manifest does not declare, and a declared command of a plugin that has a frontend and no backend with no handler once the frontend has started, are not listed and are written to the message log. The same id twice in the list makes the manifest invalid. A command of a plugin whose frontend is still loading is listed, and running it waits for the plugin. A command of a plugin whose backend failed to start is left out. A disabled plugin lists no command and binds no shortcut. A manifest with `commands` at the top level has the key dropped with an `unknown key` warning.

`scope` and `shortcut` belong to a command the frontend runs. A manifest of a plugin that has a backend and no frontend that declares one on a command is invalid; on a command the backend of a plugin with both parts runs they are ignored. Every declared command is listed in the palette; a command has no way to be hidden from it.

A plugin with both parts cannot say which of its commands the frontend registers, so the palette lists its commands through the backend until the frontend has registered them; running one the backend does not know answers an unknown command. The check that the manifest and the code agree (#1523) closes this.

The host lists the commands of the enabled plugins from their manifests without starting any. Before it runs one it checks the call against the declaration: every required argument is given, no undeclared one is, a `number` parses as a decimal and a `boolean` as true or false (written on to the plugin as `true` or `false`), and a command with `contentTypes` is given a file of one of them while one without is given none. A call that fails the check is rejected with the reason and is not passed to the plugin. Otherwise the host starts the plugin where it is not running and calls `numen.v1.PluginCommandService.RunCommand` with the command id, the file as a path relative to the vault and the arguments as text by name; the plugin returns a message a person reads and the file a window opens once the command ran (`opened_path`), either of them empty, or an error code (`UNKNOWN_COMMAND`, `BAD_ARGUMENTS`, `FAILED`) with a message giving the reason.

In Go, a backend implements `command.Handler` of `plugin-sdk-go` and serves it with `command.Mount(s, handler)`; `command.Registry` is a handler of functions added by id. A run returns a `command.Result` of the message and the opened path, and reads its arguments, as text, from `command.Arguments`. The handler does not depend on the transport. The host currently uses RPC.

The window lists the commands through `numen.v1.CommandService.ListCommands` as it starts, each time the palette opens and each time a plugin starts or stops. A command over files is listed for a file of one of its extensions, and a command with no `contentTypes` is listed for no file. The window prompts for the arguments one at a time, in the order declared, and an optional one is skipped by leaving it blank. `RunPluginCommand` runs it; a message the plugin returned is shown in the corner of the window, and the file it names is opened in the editor of its kind. The flashcards plugin makes its decks, card models and presets this way (`deck.create`, `cardModel.create`, `preset.create`, shown as Create deck, Create card model and Create preset).

`numen-cli` provides each command of a plugin that has a backend as a subcommand under its id, after its own commands, which a plugin cannot take. The vault comes first, then the file for a command over files, then one flag for each argument (`--title Verbs`; a `boolean` flag alone is true):

```bash
numen-cli deck.create notebook --title "Spanish verbs"
```

It starts the plugin host without a window, lets the plugins use the vault through the host, prints what the plugin returned and the file it names, and `numen-cli help` lists the commands the enabled plugins declare. Of two plugins that declare one id, the command line runs that of the lower plugin id; the palette offers both.

## Menus

A plugin says in `contributes.menus` which of its commands appears in a place the window draws, in which group and at which position. The window asks the place for its entries and draws them; a command declares nothing about where it is shown, and it stays in the palette whatever the menus say.

```json
"contributes": {
  "menus": [
    { "location": "searchResult", "command": "graph.showNote", "group": "navigate", "order": 20 },
    { "location": "titleBar", "command": "app:note.create", "group": "new", "order": 10 }
  ]
}
```

Each entry has a `location`, the name of the place, and a `command`, either the short id of a command this manifest declares (`graph.showNote`) or the full id of a command of the editor (`app:note.create`); the id of another plugin's command is invalid. `group` is optional: a name that tells the place to draw the entries together, without surrounding space. `order` is an optional whole number. The same command can be placed in several locations, each with its own group and order; the same `(location, command)` twice in one manifest is invalid. The Go reader checks these rules and does not check that the location is known or that the command is declared; the window does, leaves out an entry that fails, and writes one warning per entry to the message log.

There are two locations. `titleBar` is the title bar of the window; an entry there is a button with the icon of its command, shown while a vault is open, and its hover text is the title of the command and its suggested shortcut, if it has one; it needs a command with an `icon`, and a command with scope `note` is left out because the bar has no note. `searchResult` is the action panel (Ctrl/Cmd+K) of a search result that is a note: a note found by its name, a heading and a passage of a note. A result that is another file has no entries. The entry needs no icon, shows the title of its command, and receives `CommandNote` (`path`, `title`, `type`) of the result, not of the tab in front. A command that needs an input or a choice opens the palette at that step, as it does when run from the palette.

The place receives the entries sorted: groups first, in alphabetical order of the name compared as plain text by code unit (capitals before lowercase letters, whatever the language); inside a group by `order`, lowest first, and entries without `order` after those with one; ties by the full id of the command as plain text; entries with no group after all the groups, by `order` and then by id. The result does not depend on the order in which plugins load. The same command placed in one place by two plugins is shown once, by the first in this order. The title, icon and suggested shortcut of an entry are the ones of its command. In a search result "Open note" is built by the palette and is always the first action, so Enter opens the note whatever the plugins place; the first entry is the second action, run by Shift+Enter.

The application places commands of its own in the same places, in the same way and from a list of its own, kept apart from the plugins' placements: the owner of the list is `app`. Today it holds one entry, `app:settings.open` in `titleBar`, in the group `settings`, which sorts after `navigate` and `new`, so the settings button is the last one in the bar. The entries of the application are merged with those of the enabled plugins once these have loaded, and the one sort above serves both.

## Asking the host

The host serves a loopback endpoint and sets `NUMEN_HOST_PORT` in the environment of every plugin process it launches. A plugin that declares `ai:complete` calls `HostService.Complete` there with `NUMEN_PLUGIN_TOKEN` as its bearer token, which identifies the calling plugin to the host; a call without a token the host gave a running plugin is rejected as unauthenticated, and a call from a plugin without the permission is rejected as permission denied. The host uses the program set in `app.ai.default` (see [Settings](../settings.md)) and sends the question to it as an `AgentService.AskAgent` task that names no tool, no tool declaration and no conversation, which the agent handles with no tools available to it; the answer is the text the agent returned. A program that is not installed or that no plugin lists returns `NO_ROUTE`, and no other program is used in its place. A program that cannot be reached, stops without an answer or returns nothing returns `UNAVAILABLE`. a model the program does not offer is the program's error. The helper is the `ai` package of `plugin-sdk-go`.

## Models

A plugin that runs a model keeps it in the models folder every plugin shares: `models` in the shared data directory the host grants, or the folder `NUMEN_MODELS_DIR` names when the user sets it. A plugin that downloads a model declares `network:outbound`, fetches it at a pinned revision, and checks the sha256 of the bytes against the sum pinned in its code before the file is put in place. `NUMEN_OFFLINE=1` forbids every download and `NUMEN_MODELS_QUOTA` caps the bytes the folder holds (32 GiB by default).

A plugin that embeds text keeps what a model needs besides its files, such as the prefixes it expects and the way its output becomes a vector, as data and not in code. The embeddings plugin ships a table of profiles and reads more from `*.json` files in the `profiles` folder of the data directory the host grants it, merged over the table by `id`; a file that cannot be read is reported on the log of the plugin and skipped. The shape of a profile is in the README of the plugin.

## The ONNX Runtime

The host finds the library once per plugin folder, in this order: the path in `NUMEN_ONNX_RUNTIME_LIB` (or the older `NUMEN_ONNX_RUNTIME`) in the environment of the host, used as given; the first library that opens and serves the API the plugins call, of every one beside the plugin or the host (and in their `models` folders), in the models folder, in the folders where shared libraries are installed (`LD_LIBRARY_PATH`, the Nix profile, `/usr/lib` and the like), in the Nix store or in the further folders the platform loader searches (the folders `/etc/ld.so.conf` names, `DYLD_LIBRARY_PATH` and `DYLD_FALLBACK_LIBRARY_PATH` on macOS, `System32` and `PATH` on Windows); and last the release pinned by sha256 for the operating system and architecture, downloaded over https into `onnxruntime/` under the models folder, unpacked and checked again against the sha256 of the library. `NUMEN_OFFLINE=1` forbids the download. A `runtime` setting of a plugin takes precedence over what the host provides.

The execution provider is `NUMEN_ONNX_PROVIDER` in the environment of the host: `auto` (the default; the embeddings plugin enables the best accelerator it can and falls back to the CPU, and the recognition and transcription plugins run on the CPU), `cpu`, `coreml`, `cuda`, `directml` or `rocm`. The host passes it to the plugin, which enables it and fails if it cannot enable it; a plugin whose model names a provider uses that one. The pinned release is the CPU build, so a GPU provider (`cuda`, `directml`, `rocm`) with only that release fails with an error stating this: install a GPU build of the library and name it in `NUMEN_ONNX_RUNTIME_LIB`. A failed download or an unknown provider stops the plugin from launching. The model runs in the plugin process, so the process memory limit of the manifest applies to separately launched plugins, and idle eviction applies to it.
