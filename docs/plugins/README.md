# Numen Plugins Developer Guide

Numen features a modular plugin architecture that decouples specialized capabilities (compute engines, document readers, AI providers) into standalone modules. This guide details how to create, test, validate, and package plugins using the developer SDK and CLI scaffolding tooling.

---

## 1. Overview & Architecture

Every plugin runs as an independent subprocess that communicates with the core host over loopback Connect-RPC HTTP/2. No application or library module imports a plugin package: the host discovers plugins on disk from their `manifest.json` and launches them.

A plugin's backend lives in the package `backend/service`, whose `Mount(s *server.Server) error` registers its services. What the plugin holds open is closed with `s.OnClose`, which runs when the host shuts the plugin down.

### The vault, through the host

A plugin never opens the vault folder. A plugin that holds `vault:read` calls `HostVaultService` on the host endpoint (`NUMEN_HOST_PORT`, authenticated by `NUMEN_PLUGIN_TOKEN`) to read a file with its fingerprint, to ask what the index says of notes, links and backlinks, to read its data folder and cache, and to watch the vault for changed files. One that holds `vault:write` also writes a file over the fingerprint it read (a file that moved is answered `STALE`), makes a file, holds the vault's writers and brings the index level with the files it changed. The vault is the one the window is showing; the host answers `NO_VAULT` while it shows none. A call that carries the header `Numen-Vault` works the vault of that identity instead (`ListHostVaults` lists them, and an unknown identity is answered not found), which is how a window that shows no vault of its own is served. `EnsureVaultIndexed` starts reading a vault into the index and says where that stands, and `WatchMoves` tells that something moved in some vault. `plugin-sdk-go/host` is the client: `Client.With(id)` works another vault, and `hosttest.Vault` is one held in memory for tests.

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
  "activationEvents": ["onFile:*.sample"],
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
- **`capabilities`**: Core capabilities provided (`document`, `book`, `recognition`, `transcription`, `embedding`, `downloader`, `proofreading`, `tools`, `agent`, and the marker `ai.agent.provider`, which names an agent plugin that serves tasks and lists the programs it runs).
- **`targets`**: Allowed client environments (`desktop.editor`, `desktop.flashcards`).
- **`activationEvents`**: The events the plugin subscribes to. On one of them the host activates the plugin and, when it serves `numen.v1.PluginEventService`, hands it the event through `DeliverEvent`; a plugin built on the Go SDK serves it when it passes `server.WithEventHandler`. A plugin that rejects an event answers with `error` set to `PLUGIN_EVENT_ERROR_CODE_REFUSED` and its `reason`, and the host reports the rejection with the event and the plugin id. A plugin is handed only the events it subscribes to:
  - `onFile:<glob>` (e.g. `onFile:*.pdf`, `onFile:*.epub`)
  - `onCommand:<id>` (e.g. `onCommand:ocr.scan`)
  - `*` (always activated on startup)
- **`permissions`**: Whitelisted resource access: `vault:read`, `vault:write`, `network:outbound` (`network` is accepted as an alias) and `ai:complete` (ask the host for a model completion). Any other value is rejected. Permissions are checked by the host when a plugin calls it; the host does not sandbox the plugin process, so a plugin runs with the user rights of the host.
- **`runtimes`**: Libraries the host provisions for the plugin. The only value is `onnxruntime`; any other value, or one declared twice, is rejected. A plugin that declares it is launched with `NUMEN_ONNX_RUNTIME_LIB` (the absolute path of the library) and `NUMEN_ONNX_PROVIDER` (`auto`, `cpu`, `coreml`, `cuda`, `directml` or `rocm`) in its environment, and opens its sessions on them. It is also launched with the variables of the host that say where the loader finds shared libraries and which accelerator devices it may use (`LD_LIBRARY_PATH`, `CUDA_VISIBLE_DEVICES`, `CUDA_PATH` and `CUDA_PATH_V*`, `CUDA_HOME`, `NVIDIA_VISIBLE_DEVICES`, `NVIDIA_DRIVER_CAPABILITIES`, `ROCR_VISIBLE_DEVICES`, `HIP_VISIBLE_DEVICES`, and `DYLD_LIBRARY_PATH` and `DYLD_FALLBACK_LIBRARY_PATH` on macOS), which a plugin without the runtime does not get; `PATH` reaches every plugin. If the host cannot provision the library the plugin is not launched, and the activation fails with the reason.
- **`lifecycle`**:
  - `idleTimeoutSeconds`: Seconds of inactivity before process eviction; it cannot be negative.
  - `keepAliveWhileVisible`: Keep alive while a tab displaying the plugin is open.
  - `memoryLimitBytes` (or `memoryLimitMB`): The resident memory the plugin process may hold before the host evicts it. A plugin with a backend declares it, and declares `idleTimeoutSeconds` too (`0` keeps it running). The host sets no CPU limit on a plugin process.
- **`backend`**: `type` is `process` (the only backend type) and `entrypoint` is required.
- **`frontend`**: `entrypoint` is required, `styles` is optional. The window loads both from `/api/plugins/<id>/` and compiles nothing of a plugin into itself. A frontend bundle leaves `vue`, `@numen/plugin-sdk` and `@numen/ui` out and imports them from the paths `HOST_MODULES` in `@numen/plugin-sdk/host-modules` names (`/sdk/vue.js`, `/sdk/plugin-sdk.js`, `/sdk/ui.js`), where the window serves its own instances.
- **`settings`**: Declarative settings schema automatically rendered in host settings and stored by the core. The host hands the settings to the plugin in `Configure`, once it is running and again on every change; they are not in its environment. A setting has a `key`, a `type` (`boolean`, `string`, `number`, `enum`, `select`, `range` or `agent`), a `title`, a `description` and a `default`; `enum` and `select` take `options`, `range` and `number` take `min`, `max` and `step`. Neither `storedSecrets` nor `aiConsent` may be a key: the host keeps both.
  - **`agent`**: which agent program answers the plugin's completions, and with which model. A plugin that holds `ai:complete` may declare one, with no `default`: the person chooses it in the plugin's settings page, from "Default (<the agent answering now>)" and the programs the agent plugins list, then from "Agent default" and the models that program lists in `GetAgentPrograms`. Its value is `{ "agent": "<program name>", "model": "<model id>" }`; an empty `agent` is the default of the settings (`ai.default`), an empty `model` the model `agent.<program>.model` names in the settings, else the program's own. The core stores it with the plugin's other settings, the settings window alone writes it, and the host routes by it: nothing in a request names an agent. The plugin is handed it in `Configure` like any setting, and reads nothing from it to route.
- **`services`**: The services the plugin serves to the windows, by their full names (`numen.v1.CardsService`). A window that receives a call for one forwards it to the plugin with the token the plugin knows the host by, whole and unchanged, streams included; the host activates the plugin for the length of the call. A service the host serves itself is never forwarded.
- **`data`**: The folder, inside the store the host keeps in each vault, where the plugin keeps what it cannot make again (letters, digits, `_` and `-`). A plugin that holds `vault:read` or `vault:write` reads, appends to and lists the files of that folder through `HostVaultService`, and the host answers for no other folder. Beside it the plugin keeps a cache of its own for each vault (`ReadCache`, `WriteCache`), which belongs to the installation and can be made again.
- **`secrets`**: The names of the secrets, such as an API key, the plugin asks the host to keep (letters, digits, `_` and `-`; a name is not also a setting). The host keeps them in the system keychain, never in the settings file, and hands each to the plugin as a `Configure` value under its name: an empty value when no key is stored, which clears the key the plugin holds. A secret the host cannot read is left out of that `Configure` call, and the plugin keeps the key it holds. A secret is never in the environment or on the command line. The host records the names of the secrets that have a key under `storedSecrets` in the plugin's settings (names only; neither a setting nor a secret may take that name) and reads the keychain only for those, so a plugin whose key was never stored causes no keychain access. On a machine without a keychain the plugin is configured with the secret left out, and the host reports it once. A plugin reads a secret from the `Configure` values only, with `server.ReadSecret` of `plugin-sdk-go` (`server.SecretAPIKey` names the `apiKey` secret), and reads no key from its environment.

---

## 4. Frontend Plugin SDK (`@numen/plugin-sdk`)

The `@numen/plugin-sdk` library provides strict TypeScript typing, lifecycle hooks, and mock testing harnesses.

### Plugin Lifecycle

```ts
import { definePlugin, type PluginContext } from '@numen/plugin-sdk'
import SampleViewer from './Viewer.vue'

export default definePlugin({
  activate(context: PluginContext) {
    // Register document viewer for .sample files
    context.registerReader({
      id: 'sample-viewer',
      fileExtensions: ['.sample'],
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
    expect(readers[0].fileExtensions).toContain('.sample')
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

## Asking the host

The host serves a loopback endpoint and sets `NUMEN_HOST_PORT` in the environment of every plugin process it launches. A plugin that declares `ai:complete` calls `HostService.Complete` there with `NUMEN_PLUGIN_TOKEN` as its bearer token, which is how the host knows which plugin asks; a call without a token the host gave a running plugin is rejected as unauthenticated, and one from a plugin without the permission as permission denied. The host answers from the agent and the model of the calling plugin's `agent` setting, else from `ai.default` (see [Settings](../settings.md)), and sends the question to that program as an `AgentService.AskAgent` task that names no tool, no tool declaration and no conversation, which an agent answers with no tools available to it; the answer is the text the agent said. A program that is not installed or that no plugin lists is answered with `NO_ROUTE`, and no other program answers in its place; one that cannot be reached, stops without an answer or says nothing with `UNAVAILABLE`; a model the program does not offer is the program's error. A plugin the person has not let ask a model is answered `NO_CONSENT`, and its first such call puts the question to the person in the settings page; the SDK returns `ai.ErrNoConsent`, which the plugin shows as the reason nothing was answered. The helper is the `ai` package of `plugin-sdk-go`.

## Models

A plugin that runs a model keeps it in the models folder every plugin shares: `NUMEN_MODELS_DIR`, or `numen/models` under the application data directory. A plugin that downloads a model declares `network:outbound`, fetches it at a pinned revision, and checks the sha256 of the bytes against the sum pinned in its code before the file is put in place. `NUMEN_OFFLINE=1` forbids every download and `NUMEN_MODELS_QUOTA` caps the bytes the folder holds (32 GiB by default).

## The ONNX Runtime

The host finds the library once per plugin folder, in this order: the path in `NUMEN_ONNX_RUNTIME_LIB` (or the older `NUMEN_ONNX_RUNTIME` and `NUMEN_OCR_RUNTIME`) in the environment of the host, used as given; the first library that opens and serves the API the plugins call, of every one beside the plugin or the host (and in their `models` folders), in the models folder, in the folders shared libraries live in (`LD_LIBRARY_PATH`, the Nix profile, `/usr/lib` and the like), in the Nix store or in the further folders the platform loader searches (the folders `/etc/ld.so.conf` names, `DYLD_LIBRARY_PATH` and `DYLD_FALLBACK_LIBRARY_PATH` on macOS, `System32` and `PATH` on Windows); and last the release pinned by sha256 for the operating system and architecture, downloaded over https into `onnxruntime/` under the models folder, unpacked and checked again against the sha256 of the library. `NUMEN_OFFLINE=1` forbids the download. A `runtime` setting of a plugin still wins over what the host gives it.

The execution provider is `NUMEN_ONNX_PROVIDER` in the environment of the host: `auto` (the default; the embeddings plugin enables the best accelerator it can and falls back to the CPU, and the recognition and transcription plugins run on the CPU), `cpu`, `coreml`, `cuda`, `directml` or `rocm`. The host passes it to the plugin, which enables it and fails if it cannot; a plugin whose model names a provider uses that one. The pinned release is the CPU build, so a GPU provider (`cuda`, `directml`, `rocm`) with only that release is an error that says so: install a GPU build of the library and name it in `NUMEN_ONNX_RUNTIME_LIB`. A failed download or an unknown provider stops the plugin from launching. The model runs in the plugin process, so the process memory limit of the manifest applies to separately launched plugins, and idle eviction applies to it.
