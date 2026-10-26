# Architectural Layer & Boundary Validator (`@numen/lint-layers`)

Automated verification of architectural layer boundaries and invariants across
libraries, plugins, and desktop applications.

## Invariants Checked

Each check has an id, a scope (`libs`, `plugins` or `apps`) and a module; `node check.mjs --list` prints them.

1. **Libraries (`libs`)**:
   - `core-hexagon`: the layer tests of `libs/core/container` (`layers_test.go`): inward dependencies, no machine access, logging or clock reads in the core, the core's own identities, and the ports and adapters it holds.
   - `core-depguard`, `notes-depguard`, `sdk-isolation`: `golangci-lint` with the depguard rules of `libs/core`, `libs/notes` and `libs/plugin-sdk-go`.
   - `core-holds-no-plugin-package`: the core holds no package a domain plugin owns (`usecase/document`, `usecase/recognition`, `usecase/transcription`, `usecase/embedding`, `usecase/plugintool`, `internal/embedding`, `internal/ocrartifact`, `internal/transcript`, `internal/urlfile`, `internal/correction`, `internal/adapter/download`, `adapter/window/flashcards`, the transcriber, proofreader, downloader and tools bridges, `port/transcriptionport`, `port/downloadport`), and no `container` file of the recogniser, the transcriber, the proofreader, the embedder or the downloader.
   - `core-imports-no-domain-proto`: no Go file of `libs/core` names a message, enum or service that `libs/protocol` generates from a domain proto (`transcript`, `ocr`, `article`, `artifact`, `cards`, `presets`, `flashcards`, `transcription`, `download`, `proofreading`). `recording.proto` and `embedding.proto` are the core's own contracts.
   - `notes-names-no-producer`: no Go file of `libs/notes` outside its tests holds the string literal of a producer (`asr`, `ocr`, `captions`, `article`, `copy`, `review-log`).
   - `ui-isolation`: `libs/ui/core/src` names no `@numen/protocol`, `@numen/wire` or `apps`, and `npm run lint` passes in `libs/ui/core`.
   - `shell-isolation`: the same for `libs/ui/shell/src`, and `npm run lint` passes in `libs/ui/shell`.
2. **Plugins (`plugins`)**:
   - `plugins-independence`: the `Plugin` tests of `libs/core/container`.
   - `plugins-leakage`: every `manifest.json` has an id, a name and a version; there is no `plugins/shared`; no plugin imports `libs/core`; no plugin imports another.
   - `plugins-reach-no-app`: no plugin reaches `apps/` by a relative path, an alias or a Go import.
   - `plugins-reach-no-shell`: no plugin names `@numen/shell` or reaches `libs/ui/shell` by a relative path; a plugin takes components from `@numen/ui`.
   - `apps-plugins-boundary`: an application compiles nothing from `plugins/`, and a plugin nothing from `apps/`.
   - `plugins-frontend`: ESLint over `plugins/`: no `@/**` or `@plugins/**` import.
3. **Applications (`apps`)**:
   - `apps-reach-no-plugin`: no application reaches `plugins/` by a relative path, an alias, an `import.meta.glob`, a tsconfig path or a Go import. A path is resolved from the file and from every package root above it.
   - `window-names-no-plugin-kind`: the editor window's sources name no plugin kind (`deck`, `stencil`, `preset`, `plex`, `agent`, `files`) as a literal, outside its tests and stories and the files listed in `CORE_NAME_FILES` (the settings entity and page and the plugin context).
   - `desktop-editor-fsd`, `desktop-flashcards-fsd`: the Feature-Sliced Design layer hierarchy (`shared` → `entities` → `features` → `widgets` → `pages` → `app`), by `npm run lint` in each window.

## Usage

```bash
# Via Makefile:
make check_layers                     # all layers across monorepo
make check_layers target=libs         # library layers only
make check_layers target=plugins      # plugins isolation and boundaries
make check_layers target=apps         # application FSD layers
make check_layers target=core         # specific module

# Directly:
node check.mjs [target...]
node check.mjs --list
node check.mjs --help
```
