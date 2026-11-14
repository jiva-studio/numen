# Numen Plugins Developer Guide

Numen features a modular plugin architecture that decouples specialized capabilities (compute engines, document readers, AI providers) into standalone modules. This guide details how to create, test, validate, and package plugins using the developer SDK and CLI scaffolding tooling.

---

## 1. Overview & Architecture

Every plugin runs as an independent subprocess that communicates with the core host over loopback Connect-RPC HTTP/2. No application or library module imports a plugin package: the host discovers plugins on disk from their `manifest.json` and launches them.

A plugin's backend lives in the package `backend/service`, whose `Mount(s *server.Server) error` registers its services. What the plugin holds open is closed with `s.OnClose`, which runs when the host shuts the plugin down.

### The vault, through the host

A plugin never opens the vault folder. A plugin that has `vault:read` calls `HostVaultService` on the host endpoint (`NUMEN_HOST_PORT`, authenticated by `NUMEN_PLUGIN_TOKEN`) to read a file with its file stat, to query the index for notes, links and backlinks, and to read its cache. One that has `vault:write` also writes a file over the file stat it read (a call for a file that changed since is rejected with `CONFLICT`), creates a file, acquires the vault write lock and updates the index for the files it changed. The vault is the one the window is showing; the host returns `NO_VAULT` while it shows none. A call that carries the header `Numen-Vault` uses the vault of that identity instead (`ListHostVaults` lists them, and an unknown identity returns not found). A window that shows no vault of its own uses this header. `EnsureVaultIndexed` starts reading a vault into the index and reports its progress, and `WatchMoves` reports that something moved in some vault. `plugin/sdk-go/host` is the client: `Client.With(id)` uses another vault, and `hosttest.Vault` is an in-memory implementation for tests.

### The plugin's folder

Every plugin has a folder of its own in the store of each vault, `.numen/<plugin id>/`, for what it creates and cannot create again. It is served by `HostStoreService` on the host endpoint to every running plugin, without a vault permission, and the host serves a plugin its own folder only. A name is a file directly in the folder; one that contains a slash or begins with a dot is rejected with `OUTSIDE`. `ReadStoreFile` reads a whole file, `AppendStoreFile` adds to the end of one, creating it when it does not exist, with the content written whole or not at all, `WriteStoreFile` replaces a whole file the same way, `DeleteStoreFile` removes one, and `ListStoreFiles` lists the files of the folder with their sizes. `Numen-Vault` names the vault as it does for `HostVaultService`. The plugin also has a cache of its own for each vault (`ReadCache`, `WriteCache` of `HostVaultService`), which belongs to the installation and can be made again.

In Go a plugin uses its folder through `host.StoreClient` of `plugin/sdk-go/host` and declares the narrow interface of the calls it needs, `host.StoreReader` (`Read`, `List`) or `host.StoreWriter` (`Append`, `Write`, `Delete`, `Rename`); `With(id)` gives the client for another vault. `host.NewStoreClientFromEnv` is the implementation over the host endpoint, and `hosttest.Vault.Store()` is one held in memory for tests, with `Refuse` to make a file fail and `File` to put a file there as another program would.

### The search index and its floor

A plugin that holds `index:read` or `search:policy` calls `HostSearchService` on the host endpoint. The host keeps one effective floor for each pair of vault and embedding recipe, the similarity a passage must reach to be returned by a search by meaning, and applies it to every such search. The precedence is: a floor the query names, then the effective floor, then the starting floor the plugin of the embedding model reports (`floor_hint`), then 0.30. An embedding plugin embeds and reports its hint, and nothing else. The plugin that holds `search:policy` measures and chooses the floor.

With `index:read`, `GetEmbeddingState` returns the recipe and name of the model in use, its hint, how many chunks of the vault can carry a vector, how many do, and `is_ready` where every chunk does. It is computed from the index at each call, so a plugin that starts after the vectors exist reads the state a running plugin read when they were made. The host also emits the event `onSearchSourceReady:<vault id>` when an indexing pass leaves no vector pending: after the first pass of a window over the vault, and after every later pass that made vectors (`protocol.GetSearchSourceReadyEvent`, `host.ReadSearchSourceReadyEvent`); a plugin that declares `"activationEvents": ["onSearchSourceReady"]` receives it, and it repeats, so a handler reads the state again and acts on what is not done. `GetIndexStats` returns what the index of the vault holds: the recipe, name and dimension count of the model in use, how many chunks can carry a vector and how many do, how many sources the vault has, the sources and their chunks by file extension, the size of the index database and of the tables that hold vectors. It reads the counts with indexed queries and keeps them for one second, and keeps the sizes for 30 seconds, so a view may ask every few seconds. The Index stats plugin shows it in the title bar. `EmbedTexts` embeds at most 256 texts of at most 8192 bytes each with the model in use. `SampleSimilarities` takes the embedding vectors of a plugin's queries, each with the path of the note it was made from, and returns the cosine similarity of every query with the small chunks of the vault that carry a vector under the recipe and belong to another note. The host takes at most 500 small chunks in the order of their content hashes, so one index yields one sample, and at most 256 queries per call; more is `LIMIT_EXCEEDED`, a query of another dimension count than the index is `INVALID_QUERY`, and a recipe of another model is `RECIPE_MISMATCH`. The response contains one row of similarities per query, in the order of the passages, with the number of passages and of sources sampled; the format of the stored vectors does not leave the host. A plugin reads the titles of the notes it embeds through the vault calls (`vault:read`). `GetBestSimilarity` returns the cosine similarity of the passage nearest a query vector.

With `search:policy`, `GetSearchFloor` returns the floor a search applies to the vault under a recipe with the floor of each source, and `SetSearchFloor` stores it. A floor of zero removes the stored floor, a floor above one or below zero is `INVALID_FLOOR`, and setting the stored floor again changes nothing. A call that names a recipe other than the one of the model in use returns `RECIPE_MISMATCH`, and a call while the installation has no model returns `NO_MODEL`. `Numen-Vault` names the vault as it does for `HostVaultService`. While the plugin that sets the floor is disabled the host keeps applying the last stored floor. The Go client is `host.SearchClient`, behind the `host.SearchIndex` and `host.SearchPolicy` interfaces, and `hosttest.Search` implements the same interfaces in memory.

A plugin that has `window:focus` calls `HostWindowService`. `FocusNote` makes a note the current one of the main window, `FocusPassages` opens a file there at the first of at most 8 byte ranges of its text version and highlights all of them (no range opens it at its beginning), and `ReadOpenTabs` returns the tabs the window last reported with the active one. A host with no window returns `NO_WINDOW`, a missing path returns `NOT_FOUND`, and an empty path or an invalid range is an invalid argument. The Go client is `host.Client`, behind the `host.Window` interface, and `hosttest.Window` implements it in memory.

### Directories the host grants

The host creates two directories for each plugin before it starts the process, readable by the user alone and shared with no other plugin, and two that every plugin shares. It sets them in the environment: `NUMEN_PLUGIN_CACHE_DIR` for what the plugin can compute again, `NUMEN_PLUGIN_DATA_DIR` for what it cannot and the user would lose, `NUMEN_SHARED_CACHE_DIR` for what every plugin can compute again (compiled CoreML models), and `NUMEN_SHARED_DATA_DIR` for what every plugin downloads or makes and cannot compute again (models). The private ones are `<cache root>/plugins/<plugin id>` and `<data root>/plugins/<plugin id>` under the cache and data folders of the platform; the shared ones are `<cache root>/shared` and `<data root>/shared`. Models are in the data directory because the operating system may purge a cache and a model is gigabytes. The host removes a plugin's cache directory when the plugin is uninstalled or its version changes, keeps its data directory, and never deletes either shared directory. A plugin never computes a platform path: in Go it calls `cache.GetPluginDir()`, `cache.GetDataDir()`, `cache.GetSharedDir()` and `cache.GetSharedDataDir()` of `plugin/sdk-go`, which return an error where the host gave no directory, `cache.GetSharedDirOrPluginDir()` and `cache.GetSharedDataDirOrPluginDataDir()` for a host too old to share one, and `plugintest.GrantStorage(t)` grants all four in a test.

The shared directories hold one copy of what several plugins use, so a model of 8 GB is stored once and not once for each plugin. The rules are those of a content-addressed store: an entry is named by its digest (`models/` in the data directory for the models the host provisions, `coreml/<model digest>/` in the cache directory for the compiled form CoreML makes of a model); a writer writes a temporary file in the same directory, verifies the digest and renames it into place, and a reader that finds the digest uses it without hashing it again; two writers of one entry take `shared.Lock(dir)` of `plugin/sdk-go/shared`; no plugin deletes from either. The models folder is `<data root>/shared/models`, and `NUMEN_MODELS_DIR` set in the environment of the application moves it elsewhere; the host passes the folder it resolved to every plugin as `NUMEN_MODELS_DIR`. Nothing is read from an earlier location: a model in `~/.numen/dependencies` or `<config>/numen/models` is ignored and downloaded again.

### Host Lifecycle & IPC Transport

Plugins communicating over Connect-RPC use loopback TCP:

- The host supervisor launches the plugin executable with `--port=<port>` and `--token=<auth_token>`.
- The plugin process starts an HTTP/2 (`h2c`) server and announces readiness by writing to standard output:
  ```
  NUMEN_READY port=12345
  ```
- The ready line has one `listener.<name>=<port>` token for each listener the manifest declares under `contributes.listeners`, with the loopback port the plugin serves the listener on: `NUMEN_READY port=12345 listener.mcp=12346`. A plugin whose ready line lacks a declared listener, or has a malformed token (no name, a port that is not a positive integer, a name given twice), fails to activate. In Go, `Server.OpenListener(name)` called before `Serve` binds the port and adds the token.
- All plugins implement the `numen.v1.PluginLifecycleService` providing `Ping` and `Shutdown` endpoints. `Ping` returns the protocol version the plugin uses; the Go SDK fills it in.
- After the handshake the host calls `Ping`. A plugin whose protocol differs from its manifest's `protocol` or from the host's is stopped at once and reported as failed.
- The host stops a plugin process after 180 seconds without a lease, or after the `idleStopSeconds` of its `backend` (see [the manifest fields](#manifest-fields)). It checks every 30 seconds. A lease is held for each call into the plugin that has not returned, for each open stream, and for each of the plugin's own pending jobs: a plugin stays running by reporting its work to the host as a task (`HostQueueService.ReportTask`, until the task counts to its total, fails, or `FinishTask` ends it) or by holding a queue turn (`HostQueueService.EnqueueWork`, until the stream ends). Each open task id holds one lease. The host drops the leases of a plugin whose process exits. The host sets no CPU or memory limit on a plugin process.
- A plugin whose work is done calls `ReleaseProcess` of `PluginService` through the host endpoint: the host stops the process at once where it holds no lease, and else as soon as it returns the last lease. A lease taken after the call cancels it, so the process runs again for its idle limit. The Go SDK calls it with `process.NewClientFromEnv()` and `Release(ctx)`. The provider handler of `connection.Mount` releases the process after it answers `connections.detected` or an option list while no other call of the provider is open.
- A frontend that has `plugins:read` watches the plugins that run through `host.work.processes` of `@numen/plugin-sdk`, which follows `WatchPluginProcesses` of `PluginService`: for each plugin its name, when its process started, when a lease was last taken or returned, the leases it holds, whether it runs inside the core, from when the idle stop takes it (none while it holds a lease; the time of its last use once it is released) and its worst health item, with the time of the core when it took the list. The list is sent at once and again on every change, at most once a second. The Go SDK has no client for the list. The Running plugins plugin shows it in the title bar.

---

## What a plugin may depend on

- The core routes every call. A window of the application (editor, flashcards, mobile) calls the core through `@numen/core-client`. A plugin's frontend calls it through `@numen/plugin-sdk`, and a plugin's backend through `plugin/sdk-go`. A window or a plugin frontend calls a plugin's service through the core, and the core forwards the call to the backend whose manifest declares the service. A backend calls the host endpoint, where the core checks the permissions of its manifest. Windows and plugins never call each other directly.
- A plugin depends on the plugin SDK (`plugin/sdk-go`, `@numen/plugin-sdk`) and third-party code, and its frontend also on `@numen/ui`. A plugin never imports or requires `libs/note`, `libs/core/protocol`, `@numen/core-client`, `@connectrpc` or another library of this repository. An `// indirect` line for `libs/note` or `libs/core/protocol` in a plugin's `go.mod` comes from Go, because the SDK depends on them.
- The SDK contains what plugins use, plus the contracts of the operations of connections (the `llm` and `acp` namespaces of the entry `@numen/plugin-sdk/contracts`). It contains no concrete agent, engine or runtime, and no application-side API (catalog client, manifest parser, transport, raw core clients that only windows use). Those are in `@numen/core-client` (`libs/core/client`), which is private.
- The Go SDK exposes no generated protocol type. A type of the SDK may be an alias of a `libs/note` type, with one name per type. The SDK builds the Connect handlers and clients itself and converts between its types and the wire messages in unexported code. `servertest.NewLifecycleClient`, `embedding.NewClient` and `tools.NewClient` are the clients for plugin tests.
- A plugin has no `.proto` file and no protobuf code. Its backend serves its frontend with JSON methods (see `services` in §3).
- Code that several plugins use may be duplicated in each of them. No check requires moving code into the SDK or `@numen/ui`.
- Logic that depends on time takes a clock as a port, so a test controls the time. A Go plugin takes `clock.System` and `clock.Fake` of `plugin/sdk-go/clock` or declares its own clock.
- Plugins are trusted code. The core runs no plugin in an iframe or a sandbox. The token the core gives a backend process identifies the calling plugin to the core. The permission table of the plugin loader (`SERVICE_RULES`) catches mistakes and states what a plugin declares; it is not a security boundary, because a plugin frontend runs in the window and can call the core endpoints, `fetch` and `import` directly. Any plugin can run any command that some plugin lists in a slot.
- Nothing is published for now. A plugin imports only the SDK, so it can leave this repository once the SDK is published; until then it builds only here, because the Go SDK requires `libs/core/protocol` and `libs/note` through `replace` lines to folders, which the build of a consumer ignores. Publishing, API compatibility tooling and iframe isolation come later, when the SDK is released.

The full rules are in [`.agents/rules/architecture.md`](../../.agents/rules/architecture.md) §11.

### What a manifest contributes

The core discovers plugins from their manifests and has no code that names a plugin. The manifest declares:

| Field                          | Where                  | What it declares                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                   |
| ------------------------------ | ---------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `contributes.tabKinds`         | `ManifestContributes`  | Kinds of tab the frontend registers: `kind`, `title`, `icon`, `when`, `rank`, `isNoteEditor`.                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| `contributes.feature`          | `ManifestContributes`  | What the plugin provides, as text for the placeholder of an unavailable tab.                                                                                                                                                                                                                                                                                                                                                                                                                                       |
| `contributes.category`         | `ManifestContributes`  | The group the plugin is listed in.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                 |
| `contributes.themes`           | `ManifestContributes`  | Stylesheets added to the list of themes: `id`, `name`, `appearance`, `file`.                                                                                                                                                                                                                                                                                                                                                                                                                                       |
| `contributes.handlers`         | `ManifestContributes`  | The plugin's handlers of shared commands: `command`, `title`, `when` (see [Shared commands and handlers](#shared-commands-and-handlers)).                                                                                                                                                                                                                                                                                                                                                                          |
| `contributes.triggers`         | `ManifestContributes`  | Events on which the plugin's own handlers run: `on`, `command`, `when` (see [Triggers](#triggers)).                                                                                                                                                                                                                                                                                                                                                                                                                |
| `contributes.textKinds`        | `ManifestContributes`  | The readable name of each kind of text version the plugin makes: `kind`, `title`. A text version is listed by this name, and by its raw kind where none is declared.                                                                                                                                                                                                                                                                                                                                               |
| `contributes.readers`          | `ManifestContributes`  | Tabs that open files of some types: `id`, `contentTypes`, `rank`.                                                                                                                                                                                                                                                                                                                                                                                                                                                  |
| `contributes.noteTypes`        | `ManifestContributes`  | Types of note the plugin defines: `type`, `chunks`, `headings`, `idHeadingLevels`, `idPattern`.                                                                                                                                                                                                                                                                                                                                                                                                                    |
| `contributes.frameOrigins`     | `ManifestContributes`  | The https origins the plugin's pages frame.                                                                                                                                                                                                                                                                                                                                                                                                                                                                        |
| `contributes.commands`         | `ManifestContributes`  | Every command the plugin declares, whichever part runs it: `id`, `title`, `icon`, `when`, and for a backend command `arguments`.                                                                                                                                                                                                                                                                                                                                                                                   |
| `contributes.keybindings`      | `ManifestContributes`  | Keys bound to commands: `key`, `command`, `values`, `when` (see [Keybindings](#keybindings)).                                                                                                                                                                                                                                                                                                                                                                                                                      |
| `contributes.viewCapabilities` | `ManifestContributes`  | The kinds of view the frontend draws for other plugins to mount: `conversation`, `note-editor`, `vault-summary`, or a name that starts with the id of the plugin and a dot. The frontend registers a view only for a kind listed here, and only one plugin at a time draws a kind (see [Views](#views-other-plugins-mount)).                                                                                                                                                                                       |
| `contributes.dropHandlers`     | `ManifestContributes`  | The kinds of dropped data the frontend takes, as entries `{ "types": ["text/uri-list"] }`. `register.dropHandler` throws for a type not listed here and unless the plugin holds `vault:read`. One plugin takes a type, the first to register it: a second registration of that type throws with both plugin ids. The Go manifest reader rejects an empty or repeated type and entries in a plugin without a frontend.                                                                                              |
| `contributes.slots`            | `ManifestContributes`  | What the plugin places in the slots of a window and in submenus: `slot` with `command`, `view` or `submenu`, and `title`, `icon`, `values`, `group`, `order` (see [Slots](#slots)).                                                                                                                                                                                                                                                                                                                                |
| `contributes.listeners`        | `ManifestContributes`  | TCP ports on `127.0.0.1` the host binds while the plugin is enabled: `name`, `port` or `portSetting` (exactly one), `protocol`.                                                                                                                                                                                                                                                                                                                                                                                    |
| `contributes.connectionTypes`  | `ManifestContributes`  | The kinds of connection the plugin provides, each `{ id, title, description, operations, fields }`; a field is `{ key, type, title, description, default, options, min, max, step, required, isAddress, fixedValue }` with the types `string`, `number`, `boolean`, `enum`, `file` and `secret`. The type is named `<plugin id>/<id>` outside the manifest. The key `parallelRequests` is a reserved number field. A type that lists `connections.detected` reports the connections of the type found on the machine. A plugin with connection types has a `backend` (see [Connections](../settings.md#connections)). |
| `settingsViews`                | `Manifest` (top level) | Components mounted in the plugin's settings form: `id`, `title`.                                                                                                                                                                                                                                                                                                                                                                                                                                                   |

The schema is `libs/core/protocol/proto/numen/v1/manifest.proto`; the Go reader is `libs/plugin/catalog/manifest`.

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
- **`capabilities`**: The capabilities the host resolves a plugin by (`search.source` and `tools`). The host refuses any other name when it reads the manifest.
- **`targets`**: Allowed client environments (`desktop.editor`, `desktop.flashcards`).
- **`isOffByDefault`**: `true` keeps the plugin off until the person enables it in the settings. Absent means the plugin runs once it is installed.
- **`activationEvents`**: The events the plugin subscribes to. When the host emits one of them it activates the plugin and passes it the event through `DeliverEvent` when the plugin serves `numen.v1.PluginEventService`; a plugin built on the Go SDK serves it when it passes `server.WithEventHandler`. A plugin that rejects an event returns `error` set to `PLUGIN_EVENT_ERROR_CODE_REFUSED` and its `reason`, and the host reports the rejection with the event and the plugin id. A plugin that declares no activation event starts when called. An event that no plugin declares starts none, and a manifest with any other value (including `*`) is refused:
  - `onSearchSourceReady` (after the first indexing pass of a window and after every pass that made vectors, when no chunk is left pending for the search source; delivered as `onSearchSourceReady:<vault id>`)
- **`contributes.tabKinds`**: The kinds of tab the frontend registers with `register.tabKind`, each `{ "kind", "title", "icon", "when", "rank" }`. `when` is a condition over a note (see [Conditions](#conditions)), such as `target.noteType == 'deck'` or `has(target.properties.media)`: a note opens in a tab of a kind whose condition holds over it, and a kind without `when` opens no note. `rank` is `default` (when omitted) or `alternate`. `isNoteEditor: true` says the kind edits the text of the note its tab opens on: the window gives its tabs the store of open notes, which saves what is typed. Among the kinds of the enabled plugins whose condition holds, the core chooses a `default` kind before an `alternate` one, then the plugin with the lower id, and names the choice in `ListFileKinds`. A note no kind takes opens in the reader of its content type. A plugin with tab kinds has a `frontend`.
- **`contributes.readers`**: The tabs of the frontend that open files, each `{ "id", "contentTypes", "rank" }`: the id the frontend registers the reader under with `register.reader`, the types it opens, and `default` or `alternate`. A type is a MIME type or a mask: `audio/*` is every audio type, `*/*` every type. The core determines the type of a file from a fixed snapshot of the `globs` table of the freedesktop.org shared-mime-info database, by the name of the file, with the extension compared without regard to case; a file whose name the table does not know is `application/octet-stream`. A file opens in the `default` reader of its type; where two plugins declare one, the reader of the narrower type is used (`audio/mpeg` over `audio/*`), then the plugin with the lower id. An `alternate` reader never opens a file on its own. A plugin with readers has a `frontend`.
- **`settingsViews`**: The components of the frontend that the plugin's settings page mounts inside the plugin's settings form, each `{ "id", "title" }`: the id the frontend registers the view under with `register.settingsView` and its title. A view whose plugin is disabled or whose frontend did not register it draws nothing. A plugin with settings views has a `frontend`.
- **`permissions`**: Whitelisted resource access: `vault:read`, `vault:write`, `vault:manage` (add, rename, remove and open the vaults of the installation), `link:open` (open external links in the system browser; `network` is accepted as an alias), `index:read` (read the note titles, the stored vectors and the embedding state of the search index, and embed texts with the model in use), `search:policy` (read and set the similarity threshold of semantic search), `window:focus` (move the main window of the vault to a note or a file, and read the tabs it has open), `plugins:read` (read which plugins run, for how long, how long each has been idle, when the core stops it for idleness, and its worst health item) and `connections:use:<operation>` (call the operation it names, such as llm.chat, through every connection that serves it, and list those connections and the values of their row fields; one permission for each operation, and none of the settings of rows of the plugin lists the operation). There is no other permission for a model call or for the network: a plugin that uses the connections rows of its settings, and a provider of a connection type, declare no permission, because the rows the person chose are the grant, and a request of a native backend is not gated. Any other value is rejected. Permissions are checked by the host when a plugin calls it; the host does not sandbox the plugin process, so a plugin runs with the user rights of the host. Plugins are trusted code, and permissions are consent and API hygiene: `link:open` gates only `openExternal`, and a `fetch` of the frontend or a request of a native backend is not gated.
- **`dependencies`**: What the host provisions for the plugin, each `{ "id", "kind", "version", "sourceUrl", "checksumSha256", "metadata" }`. The `kind` is `runtime`, `model`, `binary` or `dataset`: the host fetches the artifact from `sourceUrl`, verifies it against the checksum, and provides its location to the plugin. A dependency with no `sourceUrl` is one the host provides itself. A dependency of kind `runtime` is a library the plugin loads; the host sets the variables that locate it. Of those, the ONNX Runtime (`{ "id": "onnxruntime", "kind": "runtime" }`) launches the plugin with `NUMEN_ONNX_RUNTIME_LIB` (the absolute path of the library) and `NUMEN_ONNX_PROVIDER` (`auto`, `cpu`, `coreml`, `cuda`, `directml` or `rocm`) in its environment, and it opens its sessions with them. It is also launched with the variables of the host that say where the loader finds shared libraries and which accelerator devices it may use (`LD_LIBRARY_PATH`, `CUDA_VISIBLE_DEVICES`, `CUDA_PATH` and `CUDA_PATH_V*`, `CUDA_HOME`, `NVIDIA_VISIBLE_DEVICES`, `NVIDIA_DRIVER_CAPABILITIES`, `ROCR_VISIBLE_DEVICES`, `HIP_VISIBLE_DEVICES`, and `DYLD_LIBRARY_PATH` and `DYLD_FALLBACK_LIBRARY_PATH` on macOS), which a plugin without a runtime does not get; `PATH` is set for every plugin. If the host cannot provision a dependency the plugin is not launched, and the activation fails with the reason. A dependency an option of a setting names in `dependencies` is provisioned only while that option is in force.
- **`backend`**: `entrypoint` is required: the executable the host starts as a process. `idleStopSeconds`, from 30 to 3600, is how long the process runs without a lease before the host stops it; without it the limit is 180 seconds. A plugin that loads a model sets a longer one to keep it loaded. A plugin without a backend is never started: its manifest declares no `activationEvents`, `services`, dependencies of kind `runtime` or `contributes.triggers`, and the host rejects a manifest that declares them when it reads it.
- **`frontend`**: an empty object, `{}`, that states the plugin has a frontend. The bundle is `dist/index.js` and the stylesheet `dist/styles.css` of the plugin folder, which the plugin build writes. The window loads both from the plugin route `/plugin/<id>/`, which serves a file of the plugin folder where one exists and forwards every other path to the plugin backend, and compiles nothing of a plugin into itself. A frontend bundle leaves `vue`, `@numen/plugin-sdk` and `@numen/ui` out and imports them from the paths `HOST_MODULES` in `@numen/plugin-sdk/host-modules` names (`/sdk/vue.js`, `/sdk/plugin-sdk.js`, `/sdk/ui.js`), where the window serves its own instances. The loader of plugin frontends and the build of those host modules are `@numen/plugin-loader` (`libs/plugin/loader`), which every application that loads plugins uses. A window loads only the plugins that declare a frontend.
- **`settings`**: Declarative settings schema automatically rendered in host settings and stored by the core. The host passes the settings to the plugin in `Configure`, once it is running and again on every change; they are not in its environment. A setting has a `key`, a `type` (`boolean`, `string`, `number`, `enum`, `range`, `file` or `connections`), a `title`, a `description`, an optional `details` and a `default`. The `description` is the one line the settings row shows under the title, at most 64 characters, and the manifest check rejects a longer one; `details` is the full text, which the row opens from an info button beside the title. `enum` takes `options` (a `string` setting may take them too, as suggestions the person can also type past), `range` and `number` take `min`, `max` and `step`, each absent until the manifest sets it. An option is a value or an object `{ "value", "title", "detail", "dependencies", "when" }`: the title is what the person reads, the detail is shown under the setting while the option is in force, and `dependencies` names dependencies of the manifest the host provisions only while the option is in force (the plugin is stopped when the set changes, and launched with the new set on its next call). The `when` of an option is a condition over the variables below, such as `app.platform != 'darwin'`; the settings page draws an option whose condition does not hold disabled, and the manifest check rejects a condition it cannot read. An `enum` takes `optionsBy: { "setting": "<key>", "options": { "<value>": [options] } }` in place of `options` to offer, under each value of an earlier enum setting, its own list; the first option of a list is the default under that value, and a stored value that is not in the list is replaced by it. A setting of one option is shown fixed, with no choice. `when: { "setting": "<key>", "is": ["<value>"] }` shows a setting only while an earlier setting holds one of the values; a setting whose condition fails keeps its stored value and is not handed to the plugin. A `file` setting takes `accept` (extensions with their dot) and `multiple: true`, which makes it hold a list of paths, one per line, that the settings page draws as one row each with an action to add another; the window offers the machine's own picker, and the host refuses a path that does not exist, is a folder, or has another extension. A `string` setting without a `default` takes `generate: "token"`: when the host loads the settings of the enabled plugin and the setting holds no value or `""`, it stores 32 random bytes in hex in it through the same write as the person's, once; a value already held is kept. The settings page shows for an enum whether its value is the default, the default of the setting it follows, or the person's choice. A `connections` setting holds the connections a plugin tries for one operation (see [Rows of connections](connection-rows.md)); it takes `operations` and `max` and none of `default`, `options`, `optionsBy`, `optionsFrom`, `min`, `step`, `accept`, `multiple`, `preview` or `generate`.
- **`services`**: The service names the plugin serves to the windows, and the only place a manifest names a service: each a dotted name with no slash that is the plugin id or starts with it and a dot (`md.numen.views.flashcards.studio`), or is one of the protocol services that plugins implement (`numen.v1.AgentService`, `numen.v1.PluginToolService`, `numen.v1.ProviderService`, `numen.v1.SearchSourceService`); the manifest reader rejects any other name, a service the core serves (`numen.v1.NoteService`) included. A name is the first segment of the URL path, so a window that posts to `/md.numen.views.flashcards.studio/CreateDeck` has the host forward the call to the plugin with the token the plugin uses to identify the host, whole and unchanged, streams included; the host activates the plugin for the length of the call, and a service the host serves itself is never forwarded. A plugin mounts the methods of a service with `s.NewAPI("md.numen.views.flashcards.studio")` of `plugin/sdk-go`: `server.Handle` serves a method with JSON (the request and response types are structs in the plugin, with `json` tags in lowerCamelCase), `server.Stream` serves a method as Server-Sent Events, a method fails with `server.NewAPIError(code, message)`, and `server.GetHeader(ctx)` returns the request headers such as `Numen-Vault`. The frontend of the plugin calls them with `createApi("md.numen.views.flashcards.studio")` of `@numen/plugin-sdk`. The plugin has no schema file and no generated code; its request and response types are written in Go and in TypeScript, and a fixture under `testdata/api/` that both a Go test and a TypeScript test read keeps the two consistent.
- **`contributes.commands`**: Every command the plugin declares, whichever part of the plugin runs it (see [Commands](#commands)). A command has an `id` as an object and an action, each in camelCase, with a dot between them (`cardModel.create`), a `title`, an `icon` (the name of an icon of the Lucide set; a name the set does not have is drawn as a placeholder), `when` (the condition under which it is available at all, see [Conditions](#conditions)) and, for a command a backend runs, `arguments`, each with a `name` of lowercase words joined by hyphens (`deck-title`), a `title`, a `type` (`string`, `number` or `boolean`) and `required`. The palette asks only the required arguments, so an optional one is for a caller that is not a person, such as `numen-cli`; an argument a person must be able to give is declared required. A command whose `when` names `version.*` has items: the text versions of a file over which its condition holds. A place draws it as a submenu of them and the palette as a pick list, and the host gives the command the id of the chosen version as its argument `version-id`. A plugin with no frontend and no backend may declare commands; they are never listed.
- **`contributes.slots`**: The entries the plugin places in the slots of a window and in submenus, each `{ "slot", "command" }`, `{ "slot", "view" }` or `{ "slot", "submenu", "title", "icon" }` with an optional `group` and `order` (see [Slots](#slots)). A plugin that places a view has a frontend.
- **`contributes.handlers`** and **`contributes.triggers`**: the plugin's handlers of the shared commands of the application, and the events on which its own handlers run (see [Shared commands and handlers](#shared-commands-and-handlers) and [Triggers](#triggers)).
- **`contributes.noteTypes`**: The types of note the plugin defines, each `{ "type", "chunks", "headings", "idHeadingLevels", "idPattern" }`, and how the index keeps the notes of each: `chunks` states whether the prose is split into chunks and searched by meaning (it is, when omitted), `headings` lists the levels of heading the index keeps (every level, when omitted), `idHeadingLevels` the levels whose headings end in a block id, and `idPattern` the pattern a block id matches whole. A note of a type no plugin declares is kept as an ordinary note is.
- **`contributes.frameOrigins`**: The https origins the plugin's pages frame (`["https://www.youtube-nocookie.com"]`). The window's `frame-src` names them while the plugin is enabled. A page of the plugin's folder that frames a site checking the Referer is loaded from the loopback socket at the address `resolveAddress` returns for it (see [Addresses](#addresses)), under a policy that frames the plugin's origins alone.
- **`contributes.listeners`**: TCP ports on `127.0.0.1` the host binds for the plugin, each `{ "name", "port", "portSetting", "protocol" }`: `name` is lowercase words joined by hyphens and unique in the manifest, `protocol` is `tcp`, and the port is given by exactly one of `port`, a fixed port from 1024 to 65535, and `portSetting`, the key of a `number` setting of the plugin. The host binds the value in force of that setting (the stored value, else its default); a value that is not an integer from 1024 to 65535 marks the plugin failed. When the setting changes, the host binds the new port, then closes the old one and the connections on it. While the plugin is enabled and the host is not in safe mode, the host binds each port. A connection starts the plugin, keeps the plugin active until the connection closes, and is forwarded to the loopback port the plugin reports for the listener in its ready line; the host reads nothing above TCP. A port that cannot be bound marks the plugin failed with the reason and closes its ports. The ports of a disabled plugin are closed. A plugin with listeners has a `backend`.
- **`contributes.connectionTypes`**: the kinds of connection the plugin provides, each `{ "id", "title", "description", "operations", "fields" }`. `id` is lowercase words joined by hyphens and unique in the manifest; the registry names the type `<plugin id>/<id>`. `operations` is a non-empty list of lowercase words with no repeat, and `submit`, `status` and `fetch` are not operations. A field has a `key` of letters and digits starting with a letter that is unique in the type and is not `type`, `name` or `id`, a `type` of `string`, `number`, `boolean`, `enum` (with `options`), `file` or `secret`, a `title`, and optionally `description`, `default`, `min`, `max`, `step`, `required`, `isAddress` and `fixedValue`. A `secret` field is kept in the key store and has no `default`, no `fixedValue`, no `options` and is not the address. `fixedValue` is held by every connection of the type, is not stored, and is allowed only on an optional `string` field with no `default`. `isAddress` marks the one `string` field whose change removes the secrets of the connection. The key `parallelRequests` is a `number` field with a `min` of at least 1, a `step` of 1 and a whole `default` that is not below `min`. A type that lists the operation `connections.detected` reports the connections of the type found on the machine (see [Writing a provider](providers.md#declare-the-types)). A plugin with connection types has a `backend` and carries out their operations as a provider (see [Writing a provider](providers.md)).

A key that the schema does not name is not an error: the manifest reader leaves it out and reports a warning that names its path (`unknown key "settings[0].mood"`). The catalog entry of the plugin contains the warnings.

---

## 4. Frontend Plugin SDK (`@numen/plugin-sdk`)

The `@numen/plugin-sdk` library provides strict TypeScript typing, lifecycle hooks, and mock testing harnesses.

### Plugin Lifecycle

`activate(plugin, host)` takes two objects. The plugin context `plugin` (`PluginContext`) holds what the plugin owns: `identity` (`id`, `target`, `app`, `subscriptions`, `openExternal`, `hasPermission`), `register` (`reader`, `settingsView`, `slotView`, `tabKind`, `command`, `view`, `dropHandler`), `settings` (`get`, `onChange`) and `health` where the window offers it. The host services `host` (`HostServices`) hold what the host lends it: `vault` (`notes`, `relations`, `files`, `texts`, `buffers`, `vaults`), `tabs` (`open`, `navigate`, `close`), `ui` (`workspace`, `window`, `commands`, `slots`, `notices`, `drops`, `mountView`), `work` (`connections`, `tasks`, `backgroundTasks`, `processes`) and `events` (`onVaultChange`, `onFileSelect`, `emitVaultChange`, `emitFileSelect`). What the plugin is handed through either object is kept in `identity.subscriptions` and taken back with the plugin. `tabs.close`, `ui.slots` and `ui.notices` are always there; any other service the window does not offer is absent from its part. `register.reader`, `settingsView`, `tabKind`, `command` and `dropHandler` throw for an id or drop type the manifest does not declare (`contributes.readers`, `settingsViews`, `contributes.tabKinds`, `contributes.dropHandlers`, `contributes.commands`, or `contributes.handlers` for a shared command), `register.reader` and `register.tabKind` throw for an id that an enabled plugin registers already, the same plugin included (disposing the registration frees the id; the Go manifest reader refuses a plugin that declares a tab kind or a reader an earlier plugin declares); the activation of the plugin then fails with the plugin id, the kind and the id. `identity.openExternal` rejects unless the plugin holds `link:open`. `ui.mountView` takes a capability the SDK names (`ViewContexts`) and the context that capability is passed. A plugin draws a view for others only for a capability its manifest lists under `contributes.viewCapabilities` (the Go manifest reader accepts `conversation`, `note-editor`, `vault-summary` or a name that starts with the plugin id and a dot); `register.view` throws for any other, throws for a capability another plugin already draws, and throws unless the plugin holds the permissions the context of that kind hands its view (none for `conversation`, whose view runs tasks on the connections the mounting plugin hands it, `vault:read` and `vault:write` for `note-editor`, `vault:read` for `vault-summary`, the table in `libs/plugin/loader/src/hosting/guards/viewPermissions.ts`); the plugin that mounts a `note-editor` needs `vault:read` and `vault:write`. Every member of a service is checked against one table in `libs/plugin/loader/src/hosting/guards/servicePermissions.ts`, and a member the table does not list is not on the service a plugin gets. The table asks `vault:read` for what reaches vault data (the notes, registering a drop handler, the files, the tabs, the vault list, the commands' items, resolving the links of a text), `vault:write` for what changes the vault or acts for the user (writes, running a command, handing a drop to a plugin, cancelling a background task, closing a tab, emitting a vault change or a file selection). `ui.commands.run` also refuses, by rejecting, a command that the plugin does not own, that no slot of any manifest lists and that is not `app:vault.create` or `app:settings.open`: a plugin runs only what the user is shown a row for or what it declared itself and `vault:manage` for `vaults.switchTo`; the window frame, the notices, the slots and the plugin's own settings need none. `identity.subscriptions` takes `push` and has a `length`; only the host removes from it. The `workspace` the window gives a plugin's tabs has the parts `window`, `dragged`, `files`, `messages` and `icons`, and the method `getNoteInFocus`.

```ts
import { definePlugin, type PluginContext } from '@numen/plugin-sdk'
import SampleViewer from './Viewer.vue'

export default definePlugin({
  activate(plugin: PluginContext) {
    // Register the reader the manifest declares under `contributes.readers`
    plugin.register.reader({
      id: 'sample-viewer',
      component: SampleViewer,
      title: 'Sample Document Viewer',
    })

    // Bind the handler of the command the manifest declares under `contributes.commands`
    plugin.register.command({
      id: 'document.inspect',
      handler: () => {
        const zoom = plugin.settings.get<number>('zoomDefault') ?? 100
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

`plugin.register.settingsView({ id, component })` registers the Vue component the settings page mounts for the `settingsViews` entry with the same id. The component takes the props `pluginId` and `values`, the current value of every declared setting by key. It emits `update:setting` with a key and a value, and the host writes that value to the setting `<key>` in the object named by `<pluginId>`. `createMockPluginContext` records registrations for `getRegisteredSettingsViews()`.

### Views in Slots

`plugin.register.slotView({ id, component })` registers the Vue component of a view the manifest places with an entry `{ "slot", "view" }` of `contributes.slots` under the same id; it is drawn in every slot the manifest places it in, and an id the manifest places nowhere throws, so the activation fails. The window passes a slot view no props; it reads what it draws from the host services of its plugin. `createMockPluginContext` records registrations for `getRegisteredSlotViews()`.

### The Title Bar

The title bar of a window is a view a plugin draws in the slot `titleBar`. The manifest places it with an entry `{ "slot": "titleBar", "view" }` of `contributes.slots`, and the frontend registers the component under the same id with `plugin.register.slotView({ id, component })`; an id the manifest places nowhere throws, and the activation fails. The Title bar plugin (`plugins/views-title-bar`, `md.numen.views.titleBar`) draws the bar of the application this way, in the editor and in the flashcards window.

```json
"contributes": {
  "slots": [{ "slot": "titleBar", "view": "bar" }]
}
```

```ts
import { definePlugin } from '@numen/plugin-sdk'
import OrchardBar from './OrchardBar.vue'

export default definePlugin({
  activate(plugin) {
    plugin.register.slotView({ id: 'bar', component: OrchardBar })
  },
})
```

The view draws the whole bar: the strip, the entries of its slots, the title, the window buttons, the area that moves the window and the room the macOS buttons take. The window draws nothing of its own in it and adds nothing beside it, and the shell offers no components for it: a bar plugin draws its own. The window passes the view no props and answers no events of it. The view reads what it draws from the plugin context and the host services of its plugin:

| Source                | What the bar reads and does                                                                                                                                                                                              |
| --------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `host.ui.slots`       | The entries of `titleBar.commands` and `titleBar.status` through `watchEntries`, in the order they are drawn. An entry the window cannot run is left out.                                                                |
| `host.ui.commands`    | `run` runs the command of an entry, with no target in a bar; `getIds` and `watch` tell which commands the window runs now, and the list of vaults ends in a row that creates a vault while they hold `app:vault.create`. |
| `host.vault.vaults`   | The vaults the window can show and the one it shows (`null` while it shows none) through `watch`; `switchTo` shows another one.                                                                                          |
| `host.ui.window`      | Whether the window is maximized or fills the screen through `watch`; `minimize`, `toggleMaximize` and `close` are the window buttons.                                                                                    |
| `plugin.identity.app` | `name`, the name of the application, and `platform`: `darwin`, `windows` or `linux`. On macOS the system draws the window buttons at the left, and the bar keeps room for them.                                          |

A window that does not offer `slots`, `commands`, `vaults` or `window` leaves that field out of the host services. `host.ui.commands.run` runs any command of the window and needs no permission.

When two enabled plugins place a view in `titleBar`, the window draws the one of the plugin first by plugin id, compared as plain text, and writes an error for the other plugin to the message log. The view is drawn in the window, not in a pane, and the window adds the stylesheet of the plugin while its view is drawn.

The Wails runtime reads CSS custom properties of the page. A bar marks the area that moves the window with `--wails-draggable: drag`, and every button, field or link inside it with `--wails-draggable: no-drag`. On Windows it marks its window buttons with `--wails-non-client-region` (`minimize`, `maximize`, `close`), so the system treats them as its own buttons and the maximize button opens Snap Layouts. The macOS buttons take 80 pixels at the left, and sit inside a bar 32 pixels high. The shell sets `--wails-resize: all` on the page, so a window on Windows and Linux is resized by its edges whichever plugin draws the bar.

#### Without a title bar

While no enabled plugin draws the bar, the window draws no strip, no window buttons and no area of its own at the top: the tabs start at the top of the window. The strips of tabs of the panes that touch the top edge move the window by their free space; a tab and its close button do not move it, and a strip that the tabs fill has no free space. A double click on the free space does nothing of the application's; on macOS the system zooms the window. The strip in the top-left corner keeps room for the macOS buttons, except in full screen; on Windows and Linux the window has no buttons, and it is closed and moved by the system (Alt+F4, the system menu). The strip of a pane below another pane does not move the window. The window finds these panes from its split tree after each change of the layout. A window with no vault open has no strip and nothing at the top. Turning the plugin on or off needs no restart.

#### The start of a window

The window waits for the plugins that fill a slot of the window itself: those whose manifest places an entry in `titleBar` or `statusBar`. The loader reads the manifests of the enabled plugins and starts these plugins before every other plugin. While they start, the window shows only a loading indicator; once all of them have started, it draws the whole window at once. The window waits for them no longer than 3 seconds (`INTERFACE_WAIT_MS` of `@numen/plugin-loader`): it is then drawn with what has started. A plugin of a window slot that is not ready by then goes on starting and adds its view, tab kinds, menu entries, commands and styles when it is; the plugins after it in the order start without waiting for it. A plugin, of a window slot or not, that has not started within 30 seconds (`PLUGIN_START_LIMIT_MS`, the limit of the backend activation too) counts as failed with "the plugin did not start within 30 s", and its start is cancelled: what it has registered is taken back at once, what it registers afterwards is taken back too, and that start never makes it active. Its backend is left as it is. A start that throws fails the same way, with its error as the reason. The failure is shown until a start succeeds. Turning the plugin on, a new window and the Retry button on its settings page start it again, with a new 30 seconds. A backend that does not start within 30 seconds fails the same way and is started again at once by its next start: there is no wait between two starts and no count of failures. A command of a plugin that is still starting waits for it up to the same 30 seconds. A tab of a kind whose plugin is still starting shows that the plugin is starting until that plugin has started or failed, whatever the other plugins do. A new vault places its tabs once the enabled plugins whose manifests declare a kind of tab have started or failed.

### The Application and the System Browser

`plugin.identity.app` is `{ version, build, platform }` of the application build the plugin runs in: the version it calls itself, its build number (0 where it has none) and `darwin`, `windows` or `linux`. `plugin.identity.openExternal(url)` opens an `http` or `https` address in the system browser and returns a `Result`; any other address fails. `createMockPluginContext` takes `app` and records the addresses opened for `getOpenedAddresses()`; `createMockHostServices` builds the host services of a test apart from it.

### Unit Testing with Mock Harness

```ts
import { describe, expect, it } from 'vitest'
import { createMockHostServices, createMockPluginContext } from '@numen/plugin-sdk/testing'
import plugin from './index'

describe('sample plugin', () => {
  it('registers reader on activation', async () => {
    const ctx = createMockPluginContext({ pluginId: 'md.numen.sample' })
    await plugin.activate(ctx, createMockHostServices())

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

Every command is filed in the window under `<owner>:<object>.<action>`. The owner is the id of the plugin that provides the command, or `app` for a command of the editor itself (`app:file.open`, `app:tab.close`). A plugin writes only the part after the colon, as `<object>.<action>` with each part in camelCase (`tree.renameFile`), in `contributes.commands` of its manifest and in `register.command` of its frontend, and the host adds the owner. A command registered twice under one full id makes the activation of its plugin fail with the full id in the message. The title is a separate imperative phrase that says what the command does (`Create deck`).

A plugin declares every command once, in `contributes.commands` of the manifest, whichever part of the plugin runs it. The declaration has no field that names the part: the application learns that from where the handler is registered. A command the frontend runs has its handler bound in the plugin's frontend with `register.command({ id, handler })`, and a command the backend runs is served by the plugin's backend. The title, icon and condition are read from the declaration, never from the code.

A handler gets a `CommandNote`: the `CommandTarget` of the run (`path`, `title`, `isFolder`, `others` the paths selected with it, and `tab` where a tab menu runs it) and the `type` of the note. A command may offer items over a target, as "Move to window" offers one per open window: `register.command({ id, handler, listItems })`, where `listItems(target)` returns `{ id, title }` rows now or as a promise. A place draws such a command as a submenu of its items and the palette as a pick list; the handler gets the id of the item picked as its second parameter (`itemId`). Its third parameter (`values`) holds the values of the arguments of the run by the names the manifest declares, as a read-only record of strings that is empty when none were given: `handler: (note, itemId, values) => {}`. A plugin runs any command, its own or another's, with `host.ui.commands.run(command, target?, itemId?)`, and reads the items with `host.ui.commands.listItems(command, target)`.

```json
"contributes": {
  "commands": [
    {
      "id": "tree.open",
      "title": "Open file tree",
      "icon": "folder-tree"
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

A command is listed only when it is declared and something can run it: its frontend registered a handler, or its plugin has a backend and no handler of the frontend takes it. A handler registered for an id the manifest does not declare, and a declared command of a plugin that has a frontend and no backend with no handler once the frontend has started, are not listed and are written to the message log. The same id twice in the list makes the manifest invalid. A command of a plugin whose frontend is still loading is listed, and running it waits for the plugin. A command of a plugin whose backend failed to start is left out. A disabled plugin lists no command and binds no key. A manifest with `commands` at the top level has the key dropped with an `unknown key` warning.

Every declared command is listed in the palette; a command has no way to be hidden from it.

A plugin with both parts cannot say which of its commands the frontend registers, so the palette lists its commands through the backend until the frontend has registered them; running one the backend does not know answers an unknown command. The check that the manifest and the code agree (#1523) closes this.

The host lists the commands of the enabled plugins from their manifests without starting any. Before it runs one it checks the call against the declaration: every required argument is given, no undeclared one is, a `number` parses as a decimal and a `boolean` as true or false (written on to the plugin as `true` or `false`), and the condition of the command holds over the file it is given. The host knows the content type and the note type of the file, not the window or the settings, so a condition that depends on those lets the call through. A call that fails the check is rejected with the reason and is not passed to the plugin. Otherwise the host starts the plugin where it is not running and calls `numen.v1.PluginCommandService.RunCommand` with the command id, the file as a path relative to the vault and the arguments as text by name; the plugin returns a message a person reads and the file a window opens once the command ran (`opened_path`), either of them empty, or an error code (`UNKNOWN_COMMAND`, `BAD_ARGUMENTS`, `FAILED`) with a message giving the reason.

In Go, a backend implements `command.Handler` of `plugin/sdk-go` and serves it with `command.Mount(s, handler)`; `command.Registry` is a handler of functions added by id. A run returns a `command.Result` of the message and the opened path, and reads its arguments, as text, from `command.Arguments`. The handler does not depend on the transport. The host currently uses RPC.

The window lists the commands through `numen.v1.CommandService.ListCommands` as it starts, each time the palette opens and each time a plugin starts or stops. A command is listed where its condition holds; the window derives the content type of a file from the table of extensions `ListCommands` sends. The window prompts for the arguments one at a time, in the order declared, and an optional one is skipped by leaving it blank. A command with items is marked `has_items`; `ListCommandItems` lists the items of several commands over one file in one call, one list with its error for each (for a command whose `when` names `version.*`, the text versions over which it holds, titled by their kind), and `RunPluginCommand` takes the id of the item picked as `argument`, which the core checks against the items before the run. `RunPluginCommand` runs it; a message the plugin returned is shown in the corner of the window, and the file it names is opened in the editor of its kind. The flashcards plugin makes its decks, card models and presets this way (`deck.create`, `cardModel.create`, `preset.create`, shown as Create deck, Create card model and Create preset).

`numen-cli` provides each command of a plugin that has a backend as a subcommand under its id, after its own commands, which a plugin cannot take. The vault comes first, then the file for a command whose condition reads its target, then one flag for each argument (`--title Verbs`; a `boolean` flag alone is true):

```bash
numen-cli deck.create notebook --title "Spanish verbs"
```

It starts the plugin host without a window, lets the plugins use the vault through the host, prints what the plugin returned and the file it names, and `numen-cli help` lists the commands the enabled plugins declare. Of two plugins that declare one id, the command line runs that of the lower plugin id; the palette offers both.

A shared command is a subcommand under its full id, run through `RunPluginCommand` with the id of the shared command, as the window runs it. The command line runs the one backend handler whose condition holds over the file; where several hold, `--plugin <id>` picks one, and without it nothing runs. A handler that offers items is not run from the command line:

```bash
numen-cli app:file.createText notebook scans/mill.pdf --plugin md.numen.engines.ocr
```

## Shared commands and handlers

A shared command is a command of the application that does nothing by itself: plugins handle it. The shared commands are declared once, in `libs/core/protocol/commands.json`, which the manifest check and the SDK (`SHARED_COMMANDS`) read:

| Command               | Title       | What a handler does               |
| --------------------- | ----------- | --------------------------------- |
| `app:file.createText` | Create text | Creates a text version of a file. |

A plugin handles one in `contributes.handlers` with `{ "command", "title", "when"? }`: the full id of the shared command, what a person reads the handler as, and the condition over which targets it handles it. Its code registers the handler under the full id of the command: the frontend with `register.command({ id: 'app:file.createText', handler, listItems? })`, the backend by adding a run under that id to its `command.Registry`. A handler the frontend registers runs there; any other runs in the backend. The manifest check rejects a handler of a command that is not shared, a command handled twice in one manifest, a handler without a title, a `when` that does not compile, and handlers of a plugin with neither a frontend nor a backend.

```json
"contributes": {
  "handlers": [
    { "command": "app:file.createText", "title": "Recognize text", "when": "target.contentType == 'image/png'" }
  ],
  "slots": [{ "slot": "fileTree.context", "command": "app:file.createText", "group": "3-text" }]
}
```

Run over a target, a shared command offers the handlers whose condition holds there as its items (`host.ui.commands.listItems(command, target)`), each with the plugin id as its id and the handler's title; a place draws a single one as a row of that title and several as a submenu. Where those handlers offer items of their own, the command offers theirs instead, in one list, each with the plugin id as its `group`, and a place draws a separator between two groups. A handler offers items of its own with `listItems` of its registration; a backend handler whose `when` names `version.*` offers the text versions of the file over which it holds, listed by the core, as `has_items` of the handler in `ListCommands` tells the window. `host.ui.commands.run(command, target, itemId)` runs the handler of the plugin the item id names, or with `<plugin id>:<item id>` that handler with its item; without an item id it runs the one handler whose condition holds, and nothing where several or none do. `numen.v1.CommandService.ListCommands` lists the handlers of the enabled plugins as `handlers`, and `RunPluginCommand` and `ListCommandItems` take a handler by its plugin id and the full id of its shared command.

The palette asks for the required `arguments` a shared command declares in `commands.json`, one step each, as for a command of a manifest, and passes the values to the handler's backend as the `arguments` of `RunPluginCommand`; the core checks them against `commands.json` for every handler.

A shared command placed in a slot by several plugins shows once. A command with items whose list is empty is hidden in every place: the menus, the bars' dropdowns and the palette.

## Triggers

A trigger runs the plugin's own handler of a shared command over a file of the vault on an event: `{ "event", "command", "when"? }` in `contributes.triggers`. The events are declared once, in `libs/core/protocol/events.json`:

| Event          | When it happens                                                        |
| -------------- | ---------------------------------------------------------------------- |
| `file.added`   | A file is added to the vault, or the core finds it for the first time. |
| `file.changed` | The content of a file of the vault changes.                            |

The core runs a trigger in the backend of its plugin, with the file as the target, over each file where its `when` and the `when` of the handler hold. It runs the triggers over every file when a vault opens, when the host has applied a change of the settings (a plugin turned on or off, or a setting changed) and when a plugin recovers, as after a Retry, and over the changed files alone on a file event; both events run in those passes. The condition reads the target, including `target.noteType` and `target.properties` of a note and `target.versions`, and the settings as `config` (`config.app.<key>`, `config.<plugin id split at dots>.<key>`). Nothing is recorded: whether the work is done is read from the data, so a trigger says `!target.versions.exists(v, v.kind == 'recognition' && v.isValid)`, and a version removed with Delete text is made again on the next pass. The core reads a note only where the path and the settings leave the condition undecided, and the versions, which read the file for its hash, only where the note leaves it undecided. A file whose run fails is tried again once in each pass; once a plugin cannot be reached, the pass skips its remaining files. The manifest check rejects a trigger on an event that is not declared, of a command the plugin does not handle, or with a `when` that does not compile, and a plugin with triggers has a `backend`.

```json
"contributes": {
  "handlers": [{ "command": "app:file.createText", "title": "Extract text layer", "when": "target.contentType == 'application/pdf'" }],
  "triggers": [{ "event": "file.added", "command": "app:file.createText", "when": "target.contentType == 'application/pdf' && !target.versions.exists(v, v.kind == 'page-text' && v.isValid)" }]
}
```

## Addresses

A plugin keeps what it downloads or makes in its folder of the vault's store, never in the vault. An address `plugin://<plugin id>/<name>` names a file of that folder; any other address is a path of the vault. A plugin writes such addresses into a note where another part reads them, as the media importer writes `media:` and `page:` into a link note, and the media viewer reads them with `host.vault.notes.readProperties(path)`, the frontmatter of a note by its keys. `resolveAddress(address)` of the SDK, `numen.v1.MediaService.ResolveAddress`, returns where a window loads the bytes from, an http address on the loopback socket, and their media type. A file of a plugin's folder is served there with the frame origins of its plugin (`contributes.frameOrigins`) and the Referrer-Policy `strict-origin-when-cross-origin`, so a page saved there frames a player that checks the Referer. The address names the size of the file, and a file written again with another size is a different address.

## Keybindings

`contributes.keybindings` is a list of `{ "key", "command", "values"?, "when"? }`. `key` is a chord of the modifiers `mod`, `ctrl`, `alt`, `shift` and `meta` joined by `+` and one key name, such as `mod+shift+p`, or a single letter such as `n`. `command` is the id of a command of the manifest or the full id of another command, such as `app:file.open`. `values` maps argument names to text and is checked as the `values` of a slot entry. `when` is a condition, see [Conditions](#conditions); two entries with the same `key` and the same `when` make the manifest invalid. The keys of the application are the `keybindings` list of `libs/core/protocol/commands.json`; there `command` may also be the id of a row of the palette that runs a command, such as `app:file.open#note`. The core lists the keybindings of every enabled plugin with its commands, so a key works before its plugin starts, and the window matches every key against them in one handler.

## Slots

A window has a fixed list of slots, declared once in `libs/core/protocol/slots.json` and known to the core and to the SDK (`SLOT`, `SlotName`). A plugin declares no slot; it fills slots of the list with the entries of `contributes.slots`, and it may declare submenus, which become places of their own. An entry in a slot that is neither in the list nor a submenu some plugin declares is drawn nowhere, and the window writes a warning to the message log of the plugin once every manifest is read.

| Slot                   | What it is                                                     | Who draws it                 |
| ---------------------- | -------------------------------------------------------------- | ---------------------------- |
| `titleBar`             | The bar at the top of a window; one view.                      | The window                   |
| `statusBar`            | The bar at the bottom of a window; one view.                   | The window                   |
| `titleBar.commands`    | The buttons of the title bar.                                  | The plugin of the title bar  |
| `titleBar.status`      | The views of the title bar, after its title.                   | The plugin of the title bar  |
| `statusBar.left`       | The start of the status bar.                                   | The plugin of the status bar |
| `statusBar.right`      | The end of the status bar.                                     | The plugin of the status bar |
| `fileTree.context`     | The menu of a row of the file tree, over its file or folder.   | The plugin of the file tree  |
| `tabStrip.context`     | The menu of a tab, over the file of the tab.                   | The tab strip of the window  |
| `plexGraph.context`    | The menu of a node of the plex graph, over its note.           | The plugin of the plex graph |
| `palette.searchResult` | The actions of a search result that is a note, over that note. | The palette                  |
| `palette.note`         | The commands of the palette over the note in front.            | The palette                  |

`titleBar` and `statusBar` are the slots of the window itself: each holds one view, the plugin of that bar, which draws the whole bar and the slots inside it. When two enabled plugins place a view in the same window slot, the window draws the view of the plugin first by plugin id, compared as plain text, and writes an error for the other plugin to its message log. The window waits for the plugins with an entry in a window slot before it draws itself (see [The start of a window](#the-start-of-a-window)).

```json
"contributes": {
  "commands": [{ "id": "ledger.open", "title": "Open ledger", "icon": "book-open" }],
  "slots": [
    { "slot": "titleBar.commands", "command": "ledger.open", "group": "navigate", "order": 40 },
    { "slot": "titleBar.status", "view": "balance" },
    { "slot": "fileTree.context", "submenu": "ledger.new", "title": "New ledger", "icon": "plus", "group": "0-new" },
    { "slot": "ledger.new", "command": "app:note.create" }
  ]
}
```

An entry places exactly one thing:

- `{ "slot", "command", "title"?, "icon"?, "values"?, "group"?, "order"?, "when"? }`: a command, drawn by the owner of the slot with the title and icon of the command; a button in a bar, a row in a menu. An entry `title` and `icon` replace the title and the icon of the command in this place. `values` maps argument names to text, the values of the arguments of the command; the host passes them when the entry is chosen, the way a run of the command takes `values` (`host.ui.commands.run(command, target, itemId, values)`), and a place passes the values of the entry it runs. Two entries of one command with different `values` are two items, with the key `<plugin id>/command/<id>?<name>=<value>&...`. The manifest check refuses `values` on a view or a submenu; a name the command does not declare in `arguments`; for an app command listed in `libs/core/protocol/commands.json` under `appCommands` (`app:file.open` with `placement` as `current` or `split`, `app:file.reveal` with `kind` as the kind of any tab) a name it does not list or a value that is not among the `options` of its `enum` argument; and for any `app:` command a name that is neither listed there nor an argument of the shared command of that id, so an `app:` command that has none takes no `values`. The arguments of a command of another plugin are not known to the check. `command` is the id of a command this manifest declares (`ledger.open`), or the full id of a command of another owner (`app:note.create`, `md.numen.views.file.tree:tree.open`). The plugin writes no code to draw it.
- `{ "slot", "view", "group"?, "order"?, "when"? }`: a component the frontend registers with `plugin.register.slotView({ id, component })` under the same id, and which draws itself. Any slot takes a view; a menu draws it as a row that holds the component. A view that has not been registered yet is not drawn.
- `{ "slot", "submenu", "title", "icon"?, "group"?, "order"?, "when"? }`: a submenu with its title and the Lucide name of its icon. Its id, words joined by dots such as `ledger.new`, is a place other entries name as their `slot`, those of any plugin. A submenu id is unique: when a second plugin declares an id another plugin declares, the submenu of the plugin first by id is drawn and the other gets an error. A submenu that holds itself, directly or through other submenus, is an error and is drawn nowhere. A submenu whose entries are all hidden is not drawn.

Any entry may go into any slot of the list; the name of a slot says what it is for, and a component that does not fit its slot is the matter of the plugin that placed it. The manifest check rejects an entry that places nothing or more than one thing, the same command, view or submenu twice in one slot, a command the manifest neither declares nor names by its full id, a submenu id declared twice in one manifest or named as a slot of the list, a submenu without a title, a command entry with a blank title or an icon that is not lowercase words joined by hyphens, values the command does not take, two views in one window slot, a group with surrounding space, an order written as text and a `when` that names an unknown key or is malformed.

An entry is identified by its plugin and what it places, as the key `<plugin id>/<command|view|submenu>/<id>` the SDK gives it (a command entry with `values` adds `?<name>=<value>` pairs, sorted by name and URL-encoded), so that a setting can later move it to another slot or change its order.

The entries of a slot are sorted the same way in every slot and in every submenu. `group` is a string such as `0-new` or `1-edit`; groups are sorted alphanumerically, compared as plain text by code unit, so a leading digit sets their order, and an entry without a group is in a last group. A plugin joins the group of another plugin by naming it. Inside a group, entries are sorted by `order`, lowest first, entries without an `order` after those with one, then by plugin id. A separator divides two groups: a line in a menu, a gap in a bar. A group whose entries are all hidden leaves no separator, so a slot never draws two separators in a row, nor one first or last.

An entry is hidden while its command is not available: its plugin does not run in this window (`targets`), or the condition of the command does not hold over the target of the slot. An entry of a plugin that is disabled, or whose last start failed, is not listed at all.

An entry with `when` is drawn only while its condition holds in that place (see [Conditions](#conditions)). `"when": "target.isFolder && !target.isMultiple"` places an entry in the menu of one folder of the file tree, and not when several rows are selected. The `when` of a command decides whether it is available at all, in every place; the `when` of an entry decides whether it shows in its slot.

The owner of a slot only draws it. It reads the entries with `host.ui.slots.getEntries(slot, target)`, or `host.ui.slots.watchEntries(slot, handler, target)` to be told again on every change; the target is what the slot is over (the clicked file or folder, the file of a tab, the note of a search result; a bar passes none, and the window runs its entries over the file of the tab in front), and the entries come in drawing order as `SlotCommand`, `SlotView`, `SlotSubmenu` and `SlotSeparator` items. It runs the command of an entry with `host.ui.commands.run(command, target)`, the one way a plugin runs a command, which runs it as the palette runs it, over the target; a command that needs an input opens the palette at that step. An entry whose command has items (`hasItems`) is drawn as a submenu of the items `host.ui.commands.listItems(command, target)` reads when the menu opens. `useSlotMenu` of the SDK runs the whole open: `open` reads the entries and the target, then the item lists, and shows the menu once every list is read, or 150 ms (`MENU_HOLD_MS`) after the open with a loading row for each list still being read; Escape or a pointer press from the start of the open until the menu shows cancels it. A menu with no rows does not open, and a list that fails to read goes to `onError`. `pick` returns the command, item and values of a row with the target and closes the menu; the place runs it with `host.ui.commands.run(command, target, itemId, values)`. In a test, `createMockHostServices()` gives `host.ui.slots` as a stand-in whose entries a test sets with `setEntries(slot, entries)`.

In `titleBar.commands` an entry is a button with the icon of its command, shown while a vault is open, and its hover text is the title of the command and its key, if it has one. In `palette.searchResult` an entry is an action of a search result that is a note: a note found by its name, a heading and a passage of a note; a result that is another file has none. "Open note" is built by the palette and is always the first action, so Enter opens the note whatever the plugins place; the first entry is the second action, run by Shift+Enter, over the note of the result, not of the tab in front.

The application places commands of its own in the same slots, from a list of its own whose owner is `app`. `app:settings.open` is in `titleBar.commands`, in the group `settings`, which sorts after `navigate` and `new`, so the settings button is the last one in the bar. The commands of the window over a row (`app:file.open`, `app:file.copyPath`, `app:file.remove`) are in `fileTree.context`, and those over a tab (`app:tab.moveToWindow`, `app:tab.detach`, `app:tab.link`, `app:tab.unlink`) in `tabStrip.context`.

## Conditions

A condition, `when`, is an expression of [CEL](https://cel.dev), as a when clause of VS Code: `target.isNote`, `target.contentType in ['application/pdf', 'image/png']`, `target.contentType.startsWith('audio/')`, `version.isLive && version.kind != ''`. The manifest check compiles it against the variables below and rejects one that is malformed, names another variable or gives no true or false. An empty or absent `when` always holds. The variables are listed once, in `libs/core/protocol/when.json`, which the Go check of the manifest and the window both read; a plugin writes no code for them.

| Variable              | Type   | What it is                                                                                                                                                                                                 |
| --------------------- | ------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `target.isRoot`       | bool   | The target is the root folder of the vault.                                                                                                                                                                |
| `target.isFolder`     | bool   | The target is a folder.                                                                                                                                                                                    |
| `target.isMultiple`   | bool   | The target is one of several paths selected together.                                                                                                                                                      |
| `target.isNote`       | bool   | The target is a note.                                                                                                                                                                                      |
| `target.isFile`       | bool   | The target is a file that is not a note.                                                                                                                                                                   |
| `target.contentType`  | string | The content type of the target, such as `application/pdf`; empty for a folder.                                                                                                                             |
| `target.noteType`     | string | The type of the note, such as `link`; empty where the target is no note.                                                                                                                                   |
| `target.properties`   | map    | The frontmatter of the note by its keys, such as `target.properties.media`; empty where the target is no note.                                                                                             |
| `target.versions`     | list   | The text versions of the file, each with `kind`, `pluginId`, `isValid` (it still matches the file) and `isLive` (the newest valid one): `target.versions.exists(v, v.kind == 'recognition' && v.isValid)`. |
| `version.kind`        | string | The kind of a text version of the file, as the plugin that made it names it.                                                                                                                               |
| `version.isLive`      | bool   | The text version is the one shown and indexed now.                                                                                                                                                         |
| `window.isVaultOpen`  | bool   | A vault is open in the window.                                                                                                                                                                             |
| `window.isVaultReady` | bool   | The vault of the window is open and accepts operations.                                                                                                                                                    |
| `window.isTabOpen`    | bool   | A tab is in front.                                                                                                                                                                                         |
| `app.platform`        | string | The operating system of the window: `darwin`, `windows` or `linux`, as `plugin.identity.app.platform`.                                                                                                     |
| `config.<key>`        | any    | A setting by its dotted key: `config.app.<key>` for the application, `config.<plugin id>.<key>` for a plugin.                                                                                              |

`target` is what the command runs over, as its place gives it: the clicked row in the file tree, the file of the tab in the tab menu, the result in the palette search, and in the palette the note or file in front. An entry of a bar (title bar or status bar) runs over the file of the tab in front, as a VS Code `editor/title` command receives the active editor; with no tab in front every target variable is false or empty. The window derives `target.contentType`, `target.isNote` and `target.isFile` from the path; the menus of the file tree and of a tab read the note type and the properties of the row before they open (`FileService.ListFileKinds` returns both). `version.*` is set only over a text version of a file, in its menu.

## Asking the host

The host serves a loopback endpoint and sets `NUMEN_HOST_PORT` in the environment of every plugin process it launches. A plugin calls it with `NUMEN_PLUGIN_TOKEN` as its bearer token, which identifies the calling plugin to the host; a call without a token the host gave a running plugin is rejected as unauthenticated, and a call that needs a permission the plugin does not hold is rejected as permission denied. A plugin that needs a model asks for an operation on the connections of its own rows (see [Rows of connections](connection-rows.md)).

## Models

A plugin that runs a model keeps it in the models folder every plugin shares: `models` in the shared data directory the host grants, or the folder `NUMEN_MODELS_DIR` names when the user sets it. A plugin that downloads a model declares `link:open` as consent, fetches it at a pinned revision, and checks the sha256 of the bytes against the sum pinned in its code before the file is put in place. `NUMEN_OFFLINE=1` forbids every download.

A plugin that embeds text keeps what a model needs besides its files, such as the prefixes it expects and the way its output becomes a vector, as data and not in code. The embeddings plugin ships a table of profiles and reads more from `*.json` files in the `profiles` folder of the data directory the host grants it, merged over the table by `id`; a file that cannot be read is reported on the log of the plugin and skipped. The shape of a profile is in the README of the plugin.

## The ONNX Runtime

The host finds the library once per plugin folder, in this order: the path in `NUMEN_ONNX_RUNTIME_LIB` (or the older `NUMEN_ONNX_RUNTIME`) in the environment of the host, used as given; the first library that opens and serves the API the plugins call, of every one beside the plugin or the host (and in their `models` folders), in the models folder, in the folders where shared libraries are installed (`LD_LIBRARY_PATH`, the Nix profile, `/usr/lib` and the like), in the Nix store or in the further folders the platform loader searches (the folders `/etc/ld.so.conf` names, `DYLD_LIBRARY_PATH` and `DYLD_FALLBACK_LIBRARY_PATH` on macOS, `System32` and `PATH` on Windows); and last the release pinned by sha256 for the operating system and architecture, downloaded over https into `onnxruntime/` under the models folder, unpacked and checked again against the sha256 of the library. `NUMEN_OFFLINE=1` forbids the download. A `runtime` setting of a plugin takes precedence over what the host provides.

The execution provider is `NUMEN_ONNX_PROVIDER` in the environment of the host: `auto` (the default; the embeddings plugin enables the best accelerator it can and falls back to the CPU, and the recognition and transcription plugins run on the CPU), `cpu`, `coreml`, `cuda`, `directml` or `rocm`. The host passes it to the plugin, which enables it and fails if it cannot enable it; a plugin whose model names a provider uses that one. The pinned release is the CPU build, so a GPU provider (`cuda`, `directml`, `rocm`) with only that release fails with an error stating this: install a GPU build of the library and name it in `NUMEN_ONNX_RUNTIME_LIB`. A failed download or an unknown provider stops the plugin from launching. The model runs in the plugin process, so the process memory limit of the manifest applies to separately launched plugins, and idle eviction applies to it.
