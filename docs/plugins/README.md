# Numen Plugins Developer Guide

Numen features a modular plugin architecture that decouples specialized capabilities (compute engines, document readers, AI providers) into standalone modules. This guide details how to create, test, validate, and package plugins using the developer SDK and CLI scaffolding tooling.

---

## 1. Overview & Architecture

Every plugin runs as an independent subprocess that communicates with the core host over loopback Connect-RPC HTTP/2. No application or library module imports a plugin package: the host discovers plugins on disk from their `manifest.json` and launches them.

A plugin's backend lives in the package `backend/service`, whose `Mount(s *server.Server) error` registers its services. What the plugin holds open is closed with `s.OnClose`, which runs when the host shuts the plugin down.

### The vault, through the host

A plugin never opens the vault folder. A plugin that holds `vault:read` calls `HostVaultService` on the host endpoint (`NUMEN_HOST_PORT`, authenticated by `NUMEN_PLUGIN_TOKEN`) to read a file with its fingerprint, to ask what the index says of notes, links and backlinks, to read its cache, and to watch the vault for changed files. One that holds `vault:write` also writes a file over the fingerprint it read (a file that moved is answered `STALE`), makes a file, holds the vault's writers and brings the index level with the files it changed. The vault is the one the window is showing; the host answers `NO_VAULT` while it shows none. A call that carries the header `Numen-Vault` works the vault of that identity instead (`ListHostVaults` lists them, and an unknown identity is answered not found), which is how a window that shows no vault of its own is served. `EnsureVaultIndexed` starts reading a vault into the index and says where that stands, and `WatchMoves` tells that something moved in some vault. `plugin-sdk-go/host` is the client: `Client.With(id)` works another vault, and `hosttest.Vault` is one held in memory for tests.

### The plugin's folder

Every plugin has a folder of its own in the store of each vault, `.numen/<plugin id>/`, for what it makes and cannot make again. It is served by `HostStoreService` on the host endpoint to every running plugin, without a vault permission, and the host serves a plugin its own folder and no other. A name is a file directly in the folder; one that holds a slash or begins with a dot is answered `OUTSIDE`. `ReadStoreFile` reads a whole file, `AppendStoreFile` adds to the end of one, making it when it is not there, with the content landing whole or not at all, `WriteStoreFile` replaces a whole file the same way, `DeleteStoreFile` removes one, and `ListStoreFiles` lists the files of the folder with their sizes. `Numen-Vault` names the vault as it does for `HostVaultService`. Beside the folder the plugin keeps a cache of its own for each vault (`ReadCache`, `WriteCache` of `HostVaultService`), which belongs to the installation and can be made again.

In Go a plugin works its folder through the `host.Store` interface of `plugin-sdk-go/host`: `Read`, `Append`, `Write`, `Delete`, `List` and `With(id)` for another vault. `host.NewStoreClientFromEnv` is the implementation over the host endpoint, and `hosttest.Vault.Store()` is one held in memory for tests, with `Refuse` to make a file fail and `File` to put a file there as another program would.

### Host Lifecycle & IPC Transport

Plugins communicating over Connect-RPC use loopback TCP:

- The host supervisor launches the plugin executable with `--port=<port>` and `--token=<auth_token>`.
- The plugin process starts an HTTP/2 (`h2c`) server and announces readiness by writing to standard output:
  ```
  NUMEN_READY port=12345
  ```
- All plugins implement the `numen.v1.PluginLifecycleService` providing `Ping` and `Shutdown` endpoints. `Ping` answers with the protocol version the plugin speaks; the Go SDK fills it in.
- After the handshake the host calls `Ping`. A plugin whose protocol differs from its manifest's `protocol` or from the host's is stopped and reported failed at once.
- A plugin whose `lifecycle.idleTimeoutSeconds` is above zero is terminated after that many seconds without active clients.

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
  "version": "0.1.0",
  "protocol": 1,
  "description": "Custom document viewer plugin for Numen",
  "capabilities": ["document"],
  "targets": ["desktop.editor"],
  "activationEvents": [],
  "contentTypes": [{ "type": "application/x-sample", "globs": ["*.sample"] }],
  "readers": [{ "id": "sample-viewer", "contentTypes": ["application/x-sample"], "rank": "default" }],
  "permissions": ["vault:read"],
  "lifecycle": {
    "idleTimeoutSeconds": 180,
    "keepAliveWhileVisible": true,
    "memoryLimitBytes": 2147483648
  },
  "backend": {
    "type": "process",
    "entrypoint": "bin/sample",
    "serviceName": "numen.v1.DocumentService"
  },
  "frontend": {
    "entrypoint": "dist/index.js",
    "styles": "dist/styles.css"
  },
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
- **`version`**: Semantic versioning string (`X.Y.Z`, optional pre-release and build parts).
- **`protocol`**: The plugin protocol version the plugin was written against. The host speaks protocol 1 and rejects a manifest that names a newer one. Absent means 1.
- **`capabilities`**: Core capabilities provided (`document`, `book`, `embedding`, `tools`, `agent`, and the marker `ai.agent.provider`, which names an agent plugin that serves tasks and lists the programs it runs).
- **`targets`**: Allowed client environments (`desktop.editor`, `desktop.flashcards`).
- **`activationEvents`**: The events the plugin subscribes to. On one of them the host activates the plugin and, when it serves `numen.v1.PluginEventService`, hands it the event through `DeliverEvent`; a plugin built on the Go SDK serves it when it passes `server.WithEventHandler`. A plugin that rejects an event answers with `error` set to `PLUGIN_EVENT_ERROR_CODE_REFUSED` and its `reason`, and the host reports the rejection with the event and the plugin id. A plugin is handed only the events it subscribes to, and the event `onFile:<extension>` of a file a window opens whose content type one of its `readers` or its `textInput` takes:
  - `onCommand:<id>` (e.g. `onCommand:ocr.scan`)
  - `*` (always activated on startup)
- **`contentTypes`**: The types of file the plugin brings that the core does not know, each `{ "type", "globs" }`: a MIME type and the patterns of the names of its files, each `*.` and a lowercase extension (`{ "type": "application/x-sample", "globs": ["*.sample"] }`). The core knows every other type from a fixed snapshot of the `globs` table of the freedesktop.org shared-mime-info database, by the name of the file, with the extension compared without regard to case; a file whose name the table does not know is `application/octet-stream`. A plugin declares only the types it brings. Wherever a manifest names a content type it may name a mask instead: `audio/*` is every audio type, `*/*` every type.
- **`readers`**: The tabs of the frontend that open files, each `{ "id", "contentTypes", "rank" }`: the id the frontend registers the reader under with `registerReader`, the types it opens, and `default` or `alternate`. A file opens in the `default` reader of its type; where two plugins declare one, the reader of the narrower type wins (`audio/mpeg` over `audio/*`), then the plugin of the lower id. An `alternate` reader never opens a file on its own. A plugin with readers has a `frontend`.
- **`permissions`**: Whitelisted resource access: `vault:read`, `vault:write`, `network:outbound` (`network` is accepted as an alias) and `ai:complete` (ask the host for a model completion). Any other value is rejected. Permissions are checked by the host when a plugin calls it; the host does not sandbox the plugin process, so a plugin runs with the user rights of the host.
- **`runtimes`**: Libraries the host provisions for the plugin. The only value is `onnxruntime`; any other value, or one declared twice, is rejected. A plugin that declares it is launched with `NUMEN_ONNX_RUNTIME_LIB` (the absolute path of the library) and `NUMEN_ONNX_PROVIDER` (`auto`, `cpu`, `coreml`, `cuda`, `directml` or `rocm`) in its environment, and opens its sessions on them. It is also launched with the variables of the host that say where the loader finds shared libraries and which accelerator devices it may use (`LD_LIBRARY_PATH`, `CUDA_VISIBLE_DEVICES`, `CUDA_PATH` and `CUDA_PATH_V*`, `CUDA_HOME`, `NVIDIA_VISIBLE_DEVICES`, `NVIDIA_DRIVER_CAPABILITIES`, `ROCR_VISIBLE_DEVICES`, `HIP_VISIBLE_DEVICES`, and `DYLD_LIBRARY_PATH` and `DYLD_FALLBACK_LIBRARY_PATH` on macOS), which a plugin without the runtime does not get; `PATH` reaches every plugin. If the host cannot provision the library the plugin is not launched, and the activation fails with the reason.
- **`lifecycle`**:
  - `idleTimeoutSeconds`: Seconds of inactivity before process eviction; it cannot be negative.
  - `keepAliveWhileVisible`: Keep alive while a tab displaying the plugin is open.
  - `memoryLimitBytes` (or `memoryLimitMB`): The resident memory the plugin process may hold before the host evicts it. A plugin with a backend declares it, and declares `idleTimeoutSeconds` too (`0` keeps it running). The host sets no CPU limit on a plugin process.
- **`backend`**: `type` is `process` (the only backend type) and `entrypoint` is required. A plugin without a backend is never started: its manifest declares no `activationEvents`, `commands`, `services`, `secrets`, `runtimes`, `textInput` or `playback`, and the host refuses one that does when it reads the manifest.
- **`frontend`**: `entrypoint` is required, `styles` is optional. The window loads both from `/api/plugins/<id>/` and compiles nothing of a plugin into itself. A frontend bundle leaves `vue`, `@numen/plugin-sdk` and `@numen/ui` out and imports them from the paths `HOST_MODULES` in `@numen/plugin-sdk/host-modules` names (`/sdk/vue.js`, `/sdk/plugin-sdk.js`, `/sdk/ui.js`), where the window serves its own instances. The loader of plugin frontends and the build of those host modules are `@numen/plugin-host` (`libs/plugin-host-ts`), which every application that loads plugins uses. A window loads only the plugins that declare a frontend.
- **`settings`**: Declarative settings schema automatically rendered in host settings and stored by the core. The host hands the settings to the plugin in `Configure`, once it is running and again on every change; they are not in its environment. A setting has a `key`, a `type` (`boolean`, `string`, `number`, `enum`, `select`, `range`, `agent`, `file` or `folder`), a `title`, a `description` and a `default`; `enum` and `select` take `options`, `range` and `number` take `min`, `max` and `step`. An option is a value or an object `{ "value", "title", "detail", "dependencies" }`: the title is what the person reads, the detail is said under the setting while the option is in force, and `dependencies` names dependencies of the manifest the host provisions only while the option is in force (the plugin is stopped when the set changes, and launched with the new set on its next call). An `enum` takes `optionsBy: { "setting": "<key>", "options": { "<value>": [options] } }` in place of `options` to offer, under each value of an earlier enum setting, its own list; the first option of a list is the default under that value, and a stored value that is not in the list gives way to it. A setting of one option is shown fixed, with no choice. `when: { "setting": "<key>", "is": ["<value>"] }` shows a setting only while an earlier setting holds one of the values; a setting whose condition fails keeps its stored value and is not handed to the plugin. A `file` setting takes `accept` (extensions with their dot) and a `folder` setting a folder; the window offers the machine's own picker, and the host refuses a path that does not exist, is of the other kind, or has another extension. The settings page says of an enum whether its value is the default, the default of the setting it follows, or the person's choice. Neither `storedSecrets` nor `aiConsent` may be a key: the host keeps both.
  - **`agent`**: which agent program answers the plugin's completions, and with which model. A plugin that holds `ai:complete` may declare one, with no `default`: the person chooses it in the plugin's settings page, from "Default (<the agent answering now>)" and the programs the agent plugins list, then from "Agent default" and the models that program lists in `GetAgentPrograms`. Its value is `{ "agent": "<program name>", "model": "<model id>" }`; an empty `agent` is the default of the settings (`ai.default`), an empty `model` the model `agent.<program>.model` names in the settings, else the program's own. The core stores it with the plugin's other settings, the settings window alone writes it, and the host routes by it: nothing in a request names an agent. The plugin is handed it in `Configure` like any setting, and reads nothing from it to route.
- **`services`**: The services the plugin serves to the windows, by their full names (`numen.flashcards.v1.CardsService`). A window that receives a call for one forwards it to the plugin with the token the plugin knows the host by, whole and unchanged, streams included; the host activates the plugin for the length of the call. A service the host serves itself is never forwarded. A plugin that defines its own services keeps their schemas in its `proto/` directory under a package of its own (`plugins/views-flashcards-studio/proto/numen/flashcards/v1/`, package `numen.flashcards.v1`), imports the shared types of `libs/protocol` through the buf workspace of the repository root `buf.yaml`, and generates them with `buf generate` run in the plugin directory: Go into `gen/`, TypeScript into `frontend/gen/`.
- **`secrets`**: The secrets, such as an API key, the plugin asks the host to keep, each `{ "name", "title", "description", "when" }`: the name (letters, digits, `_` and `-`; a name is not also a setting) is what the plugin reads the key under, the title what the person reads, and `when` shows the secret, and hands it to the plugin, under the same condition a setting takes. The host keeps them in the system keychain, never in the settings file, and hands each to the plugin as a `Configure` value under its name: an empty value when no key is stored, which clears the key the plugin holds. A secret the host cannot read is left out of that `Configure` call, and the plugin keeps the key it holds. A secret is never in the environment or on the command line. The host records the names of the secrets that have a key under `storedSecrets` in the plugin's settings (names only; neither a setting nor a secret may take that name) and reads the keychain only for those, so a plugin whose key was never stored causes no keychain access. On a machine without a keychain the plugin is configured with the secret left out, and the host reports it once. A plugin reads a secret from the `Configure` values only, with `server.ReadSecret` of `plugin-sdk-go` (`server.SecretAPIKey` names the `apiKey` secret), and reads no key from its environment.
- **`commands`**: The commands the plugin's backend runs, which the palette of the window and `numen-cli` both offer (see [Commands](#commands)). A command has an `id` and argument `name`s of lowercase words joined by hyphens (`create-deck`), a `title`, `contentTypes` (types or masks, `*/*` for a file of any type) when it runs over a file, and `arguments`, each with a `name`, a `title`, a `type` (`string`, `number` or `boolean`) and `required`. `hiddenInPalette` keeps a command out of the palette where the window offers the same thing another way; `numen-cli` still offers it. A plugin that declares commands has a `backend`.
- **`playback`**: `files` names the file types, by extension with its dot, the plugin keeps a copy to play of (`{ "files": [".url"] }`). The copy is a file of the plugin's store folder named by the content hash of the file and the extension of its container (`<sha256>.mp4`; `.mp4`, `.m4a`, `.webm`, `.mp3`, `.ogg`, `.wav` and `.flac` are played). `RecordingService.GetRecording` of such a file answers the address the copy is played from, and the file itself where no copy is kept.

---

## 4. Frontend Plugin SDK (`@numen/plugin-sdk`)

The `@numen/plugin-sdk` library provides strict TypeScript typing, lifecycle hooks, and mock testing harnesses.

### Plugin Lifecycle

```ts
import { definePlugin, type PluginContext } from '@numen/plugin-sdk'
import SampleViewer from './Viewer.vue'

export default definePlugin({
  activate(context: PluginContext) {
    // Register the reader the manifest declares under `readers`
    context.registerReader({
      id: 'sample-viewer',
      component: SampleViewer,
      title: 'Sample Document Viewer',
    })

    // Register a palette command
    context.registerCommand({
      id: 'sample.inspect',
      title: 'Sample: Inspect Current Document',
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

Verify `manifest.json` with the same validator the host runs at discovery (identity, protocol, lifecycle, permissions, settings), and ensure backend/frontend entrypoints resolve:

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
2. **Build, lint, test**: every plugin's `Makefile` answers the same four targets, `build`, `lint` (Go vet and golangci-lint, ESLint and the frontend's typecheck), `test` and `clean`, over the backend and the frontend it has. The root targets `plugins_build`, `plugins_lint`, `plugins_test` and `plugins_clean` run them in every directory under `plugins/` that holds a `manifest.json`.
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

A plugin declares its commands in `commands` of the manifest and runs them in its backend; the host builds the palette and the command line from the declarations, so neither knows any command by name. A command that only changes the window, such as opening a tab, is registered with `registerCommand` in the plugin's frontend instead, and the command line never offers it.

```json
"commands": [
  {
    "id": "create-deck",
    "title": "New deck",
    "arguments": [{ "name": "title", "title": "What is it called", "type": "string", "required": true }]
  }
]
```

The host lists the commands of the enabled plugins from their manifests without starting any. Before it runs one it checks the call against the declaration: every required argument is given, no undeclared one is, a `number` parses as a decimal and a `boolean` as true or false (written on to the plugin as `true` or `false`), and a command with `contentTypes` is given a file of one of them while one without is given none. A call that fails the check is answered with the reason and never reaches the plugin. Otherwise the host starts the plugin where it is not running and calls `numen.v1.PluginCommandService.RunCommand` with the command id, the file as a path relative to the vault and the arguments as text by name; the plugin answers with a message a person reads and the file a window opens once the command ran (`opened_path`), either of them empty, or with an error code (`UNKNOWN_COMMAND`, `BAD_ARGUMENTS`, `FAILED`) with the message saying why.

In Go, a backend implements `command.Handler` of `plugin-sdk-go` and serves it with `command.Mount(s, handler)`; `command.Registry` is a handler of functions added by id. A run answers a `command.Result` of the message and the opened path, and reads its arguments, as text, from `command.Arguments`. The handler knows nothing of the transport, and RPC is the one the host uses now.

The window lists the commands through `numen.v1.CommandService.ListCommands` as it starts and each time the palette opens. A command over files is offered over a file of one of its extensions, and one over none anywhere; it asks for its arguments one step each, in the order declared, and an optional one is left out by choosing to leave it blank. `RunPluginCommand` runs it; a message the plugin answered is written in the corner of the window, and the file it names is opened in the editor of its kind. The flashcards plugin makes its decks, stencils and presets this way (`create-deck`, `create-stencil`, `create-preset`, shown as New deck, New stencil and New preset).

`numen-cli` offers each command as a subcommand under its id, after its own commands, which a plugin cannot take. The vault comes first, then the file for a command over files, then one flag for each argument (`--title Verbs`; a `boolean` flag alone is true):

```bash
numen-cli create-deck notebook --title "Spanish verbs"
```

It starts the plugin host without a window, lets the plugins work the vault through the host, prints what the plugin answered and the file it names, and `numen-cli help` lists the commands the enabled plugins declare. Of two plugins that declare one id, the command line runs that of the lower plugin id; the palette offers both.

## Asking the host

The host serves a loopback endpoint and sets `NUMEN_HOST_PORT` in the environment of every plugin process it launches. A plugin that declares `ai:complete` calls `HostService.Complete` there with `NUMEN_PLUGIN_TOKEN` as its bearer token, which is how the host knows which plugin asks; a call without a token the host gave a running plugin is rejected as unauthenticated, and one from a plugin without the permission as permission denied. The host answers from the agent and the model of the calling plugin's `agent` setting, else from `ai.default` (see [Settings](../settings.md)), and sends the question to that program as an `AgentService.AskAgent` task that names no tool, no tool declaration and no conversation, which an agent answers with no tools available to it; the answer is the text the agent said. A program that is not installed or that no plugin lists is answered with `NO_ROUTE`, and no other program answers in its place; one that cannot be reached, stops without an answer or says nothing with `UNAVAILABLE`; a model the program does not offer is the program's error. A plugin the person has not let ask a model is answered `NO_CONSENT`, and its first such call puts the question to the person in the settings page; the SDK returns `ai.ErrNoConsent`, which the plugin shows as the reason nothing was answered. The helper is the `ai` package of `plugin-sdk-go`.

## Models

A plugin that runs a model keeps it in the models folder every plugin shares: `NUMEN_MODELS_DIR`, or `numen/models` under the application data directory. A plugin that downloads a model declares `network:outbound`, fetches it at a pinned revision, and checks the sha256 of the bytes against the sum pinned in its code before the file is put in place. `NUMEN_OFFLINE=1` forbids every download and `NUMEN_MODELS_QUOTA` caps the bytes the folder holds (32 GiB by default).

## The ONNX Runtime

The host finds the library once per plugin folder, in this order: the path in `NUMEN_ONNX_RUNTIME_LIB` (or the older `NUMEN_ONNX_RUNTIME`) in the environment of the host, used as given; the first library that opens and serves the API the plugins call, of every one beside the plugin or the host (and in their `models` folders), in the models folder, in the folders shared libraries live in (`LD_LIBRARY_PATH`, the Nix profile, `/usr/lib` and the like), in the Nix store or in the further folders the platform loader searches (the folders `/etc/ld.so.conf` names, `DYLD_LIBRARY_PATH` and `DYLD_FALLBACK_LIBRARY_PATH` on macOS, `System32` and `PATH` on Windows); and last the release pinned by sha256 for the operating system and architecture, downloaded over https into `onnxruntime/` under the models folder, unpacked and checked again against the sha256 of the library. `NUMEN_OFFLINE=1` forbids the download. A `runtime` setting of a plugin still wins over what the host gives it.

The execution provider is `NUMEN_ONNX_PROVIDER` in the environment of the host: `auto` (the default; the embeddings plugin enables the best accelerator it can and falls back to the CPU, and the recognition and transcription plugins run on the CPU), `cpu`, `coreml`, `cuda`, `directml` or `rocm`. The host passes it to the plugin, which enables it and fails if it cannot; a plugin whose model names a provider uses that one. The pinned release is the CPU build, so a GPU provider (`cuda`, `directml`, `rocm`) with only that release is an error that says so: install a GPU build of the library and name it in `NUMEN_ONNX_RUNTIME_LIB`. A failed download or an unknown provider stops the plugin from launching. The model runs in the plugin process, so the process memory limit of the manifest applies to separately launched plugins, and idle eviction applies to it.
