# Numen Plugins Developer Guide

Numen features a modular plugin architecture that decouples specialized capabilities (compute engines, document readers, AI providers) into standalone modules. This guide details how to create, test, validate, and package plugins using the developer SDK and CLI scaffolding tooling.

---

## 1. Overview & Architecture

Numen plugins follow a dual-mode architectural contract:
1. **In-tree Monolithic Build**: Plugins compile directly into the application binary for development without IPC overhead.
2. **Out-of-Process Modular Build**: Plugins execute as independent subprocesses communicating with the core host over loopback Connect-RPC HTTP/2.

### Host Lifecycle & IPC Transport

Plugins communicating over Connect-RPC use loopback TCP:
- The host supervisor launches the plugin executable with `--port=<port>` and `--token=<auth_token>`.
- The plugin process starts an HTTP/2 (`h2c`) server and announces readiness by writing to standard output:
  ```
  NUMEN_READY port=12345
  ```
- All plugins implement the `numen.v1.PluginLifecycleService` providing `Ping` and `Shutdown` endpoints.
- Plugins configured with `mode: "on-demand"` are automatically terminated after their configured idle timeout (e.g. 180 seconds without active clients).

---

## 2. Plugin Directory Layout

Every plugin follows a standard layout:

```text
modules/plugins/<name>/
├── manifest.json       # Metadata, capabilities, activation events, permissions
├── Makefile            # Build and test recipes
├── README.md           # Documentation
├── backend/            # Native Connect-RPC server (Go)
│   └── main.go
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
  "description": "Custom document viewer plugin for Numen",
  "capabilities": ["document"],
  "targets": ["desktop.editor"],
  "activationEvents": ["onFile:*.sample"],
  "permissions": ["vault:read"],
  "lifecycle": {
    "mode": "on-demand",
    "idleTimeoutSeconds": 180,
    "keepAliveWhileVisible": true
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

- **`id`**: Unique reverse-DNS identifier (e.g., `md.numen.<plugin-name>`).
- **`name`**: User-facing display title.
- **`version`**: Semantic versioning string (`X.Y.Z`).
- **`capabilities`**: Core capabilities provided (`document`, `compute`, `agent`, `recognition`, `transcription`, `proofreading`).
- **`targets`**: Allowed client environments (`desktop.editor`, `desktop.flashcards`).
- **`activationEvents`**: Triggers for host activation:
  - `onFile:<glob>` (e.g. `onFile:*.pdf`, `onFile:*.epub`)
  - `onCommand:<id>` (e.g. `onCommand:ocr.scan`)
  - `*` (always activated on startup)
- **`permissions`**: Whitelisted resource access (`vault:read`, `vault:write`, `network`, `shell`, `notifications`, `clipboard`).
- **`lifecycle`**:
  - `mode`: `"on-demand"`, `"persistent"`, or `"manual"`.
  - `idleTimeoutSeconds`: Seconds of inactivity before process eviction.
  - `keepAliveWhileVisible`: Keep alive while a tab displaying the plugin is open.
- **`settings`**: Declarative settings schema automatically rendered in host settings and stored by the core.

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

Verify `manifest.json`, check permissions against whitelist, and ensure backend/frontend entrypoints resolve:

```bash
numen plugin validate modules/plugins/sample
```

Output:
```text
plugin at modules/plugins/sample is valid
```

If validation fails, each rule violation is reported:
```text
validation errors:
  - field 'id' "sample" must use reverse-DNS notation (e.g. md.numen.sample)
  - permission "unrestricted:root" is unknown; allowed permissions: [vault:read vault:write network shell notifications clipboard]
```

### 3. Packaging (`package`)

Bundle the plugin into a distributable archive ready for installation:

```bash
numen plugin package modules/plugins/sample
```

Output:
```text
Bundled 12 files into modules/plugins/sample/dist/md.numen.sample-0.1.0.zip
packaged plugin: modules/plugins/sample/dist/md.numen.sample-0.1.0.zip
```

---

## 6. Developing & Testing Workflow

1. **Scaffold plugin**:
   ```bash
   make create_plugin name=epub-reader template=reader
   ```
2. **Build backend**:
   ```bash
   cd modules/plugins/epub-reader
   make build
   ```
3. **Run unit tests**:
   ```bash
   make test
   ```
4. **Validate manifest**:
   ```bash
   numen plugin validate .
   ```
5. **Package for distribution**:
   ```bash
   numen plugin package .
   ```

## Models

A plugin that runs a model keeps it in the models folder every plugin shares: `NUMEN_MODELS_DIR`, or `numen/models` under the platform cache directory. A plugin that downloads a model declares `network:outbound`, fetches it at a pinned revision, and checks the sha256 of the bytes against the sum pinned in its code before the file is put in place. `NUMEN_OFFLINE=1` forbids every download and `NUMEN_MODELS_QUOTA` caps the bytes the folder holds (32 GiB by default). The helpers are the `models` package of `plugin-sdk-go`.
