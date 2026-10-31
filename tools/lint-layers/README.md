# Architectural Layer & Boundary Validator (`@numen/lint-layers`)

Automated verification of architectural layer boundaries and invariants across
libraries, plugins, and desktop applications.

## Invariants Checked

Each check has an id, a scope (`libs`, `plugins` or `apps`) and a module; `node check.mjs --list` prints them.

1. **Libraries (`libs`)**:
   - `core-hexagon`: the layer tests of `libs/core/container` (`layers_test.go`): inward dependencies, no machine access, logging or clock reads in the core, the core's own identities, and the ports and adapters it holds.
   - `machine-clock`: the type-aware test `libs/core/container/clock_port_test.go` loads every module of `go.work`, tests included, and fails on a reference to `time.Now`, `time.Since` or `time.Until` outside the body of a method `Now() time.Time` that implements a clock port.
   - `library-matrix`: every `go.mod` under `libs/` belongs to a family, the first directory under `libs/`, and the family is a row of `LIBRARY_DEPENDENCIES` in `checks/libraries.mjs`. The `go.mod` and the Go files of a library, tests included, reach only the modules of their own family, the families their row lists, and nothing outside `libs/`. A library with no row fails, so a new library states its place before it builds.
   - `core-depguard`, `notes-depguard`, `sdk-isolation`: `golangci-lint` with the depguard rules of `libs/core`, `libs/notes` and `libs/plugin-sdk-go`.
   - `core-holds-no-plugin-package`: the core holds no package a domain plugin owns (`usecase/document`, `usecase/recognition`, `usecase/transcription`, `usecase/embedding`, `usecase/plugintool`, `internal/embedding`, `internal/ocrartifact`, `internal/transcript`, `internal/urlfile`, `internal/correction`, `internal/adapter/download`, `adapter/window/flashcards`, the transcriber, proofreader, downloader and tools bridges, `port/transcriptionport`, `port/downloadport`), and no `container` file of the recogniser, the transcriber, the proofreader, the embedder or the downloader.
   - `core-imports-no-domain-proto`: no Go file of `libs/core` names a message, enum or service that `libs/protocol` generates from a domain proto (`transcript`, `ocr`, `article`, `artifact`, `cards`, `presets`, `flashcards`, `transcription`, `download`, `proofreading`). `recording.proto` and `embedding.proto` are the core's own contracts.
   - `notes-names-no-producer`: no Go file of `libs/notes` outside its tests states a plugin word (literal or identifier). The dictionary is every single-token value of the `plugins/*/manifest.json` files and the id segments no other plugin id shares, minus the words the SDKs, the protocol and `libs/core` define, and minus presentation fields (`icon`, `key`, `title`, `description`, `default`, `label`, `unit`, `placeholder`, `name`) and host fields (`targets`).
   - `ui-isolation`: `libs/ui/core/src` names no `@numen/protocol`, `@numen/wire`, `@numen/shell` or `apps`, and `npm run lint` passes in `libs/ui/core`.
   - `ui-holds-no-plugin-view`: `libs/ui/core/src` holds none of the directories of a view a plugin owns and `src/index.ts` exports none of its names (`views-agent-chat`: `screens/agent`, `features/thread`, `features/message-composer`; `Agent`, `Thread`, `MessageComposer`, `useConversation`, `Conversation`, `Turn`, `AgentModelOption`, `AgentPort`, `AgentStep`).
   - `ui-holds-no-shell-part`: `libs/ui/core/src` holds none of the window chrome `@numen/shell` owns (`features/workspace`, `features/palette`, `features/status-bar`, `features/notices`, `features/welcome`, `screens`) and `src/index.ts` exports none of its names (`WorkspaceLayout`, `Palette`, `StatusBar`, `Notices`, `createNoticeQueue`, `WelcomePage`, `getVaultForKey`, `VaultRow`, `WelcomeAction`).
   - `ui-exports-no-editor`: `libs/ui/core/src` holds no `features/editor`, and `src/index.ts` exports nothing from it and no wikilink helper (`Editor`, `createTimeGutter`, `Extension`, `isNoteAddress`, `getWikilinks`). No shipped file of `libs/ui/core/src` names the note link format either: no `isNoteAddress`, `getWikilinks`, `parseAddress`, `writeAddress`, no name holding `wikilink`, no `name://` or `note://` literal. The notes plugin owns the editor; the core parses links.
   - `protocol-holds-no-plugin-contract`: the code of `libs/protocol/proto/numen/v1` names nothing a single plugin owns: `TranscriptService`, `ArticleService`, `BookService`, `DocumentService`, `SOURCE_KIND_BOOK`, `SOURCE_KIND_RECORDING`, `SOURCE_KIND_URL`, `BookFormat`, `FILE_MENU_ACTION_RECOGNISE`, `FILE_MENU_ACTION_PROOFREAD`, `BookProgress`, `RecordingProgress`, `SetVaultInUse` and its messages, and the `mark` and `face` fields of `agent.proto`. Comments are not read.
   - `no-file-kind-outside-plugins`: the shipped code of `libs/core`, `libs/notes`, `libs/plugin-sdk-go`, `libs/plugin-sdk-ts`, `libs/ui/core`, `libs/ui/shell` and `apps/desktop/editor/src` holds no literal `"book"`, `"recording"`, `"url"` or `"document"`, and no name of those kinds (`KindBook`, `TabRecording`, `CapabilityDocument`, `SOURCE_KIND_URL`, `SourceKind.BOOK`, `BookFormat`, `DefaultBookExtensions`, `BookProgress`, `ReadBooks`, `BookServiceName`, and the script constants `BOOK`, `RECORDING`, `DOCUMENT`). A double-quoted string inside a literal, such as a manifest in a Go raw string, is read too. Tests, stories, `testing/`, `testdata/`, `e2e/` and generated code are not read. The core finds a plugin by content type instead.
   - `shell-isolation`: `libs/ui/shell/src` names no `@numen/protocol`, `@numen/wire` or `apps`, and `npm run lint` passes in `libs/ui/shell`; the shell reaches the components through `@numen/ui` alone.
   - `plugin-host-isolation`: `libs/plugin-host-ts/src` imports only `vue` and `@numen/plugin-sdk` besides its own files (its tests also `vitest`, its `vite.ts` also `vite`), so the plugin loader knows nothing of `@numen/ui`, `@numen/shell`, the protocol, the wire or an application; and `npm run lint` passes in `libs/plugin-host-ts`.
   - `notes-hexagon`: Notes Library Internal Hexagonal Layers (layers_test.go).
   - `sdk-consumers`: Plugin SDK Imported Only By Plugins and Applications.
   - `go-no-mutable-globals`: Go No Package-Level Mutable Singletons or os.Exit Outside cmd/.
   - `notes-depguard`: Notes Library Depguard Boundaries.
   - `sdk-isolation`: Plugin SDK Go Isolation (no core or app imports).
   - `sdk-encapsulates-protocol`: Plugin SDK Encapsulates Protocol (no numenv1.* or window IDs in public SDK API).
   - `no-wire-library`: No @numen/wire Library (libs/wire removed; utilities in @numen/plugin-sdk, protocol adapters in host).
   - `contracts-name-no-plugin`: Shared Contracts Name No Plugin (schema, SDKs, plugin host, notes: comments and documents included).
   - `docs-name-existing-paths`: Rule Documents Name Existing Paths (AGENTS.md, .agents/rules, lint-layers README).
   - `docs-name-existing-tests`: Rule Documents Name Declared Go Tests.
   - `docs-list-verifiers`: Layer Validator README Lists Every Registered Verifier.
   - `all-ts-packages-eslint`: Every TypeScript Package Has ESLint Configured.
2. **Plugins (`plugins`)**:
   - `plugins-independence`: the `Plugin` tests of `libs/core/container`.
   - `plugins-leakage`: every `manifest.json` has an id, a name and a version; there is no shared plugin folder; no plugin imports `libs/core`; no plugin imports another, by a Go import, by a relative TypeScript or Vue import that resolves into another plugin's directory, or by another plugin's package name.
   - `plugins-reach-no-app`: no plugin reaches `apps/` by a relative path, an alias or a Go import.
   - `plugins-reach-no-shell`: no plugin names `@numen/shell` or `@numen/plugin-host`, or reaches `libs/ui/shell` or `libs/plugin-host-ts` by a relative path; a plugin takes components from `@numen/ui` and is loaded by the host, never loads itself.
   - `plugins-draw-no-notices`: no plugin imports `Notices`, `Notice`, `createNotice`, `createNoticeQueue` or `NoticeQueue` from a package; a plugin shows notices through `ctx.notices` and the window draws them in one list.
   - `plugins-name-no-window`: the shipped code of a plugin holds no window id literal (`"editor"`, `"flashcards"`) and names no `WindowService` or `NewWindowServiceClient`; a plugin reaches the window through the SDK.
   - `plugins-write-data-through-host`: a shipped Go file of a plugin that names the `.numen` folder calls no `os` function that changes the file system (`os.WriteFile`, `os.MkdirAll`, `os.Rename`, `os.Remove`, …); a plugin writes its data through the host store.
   - `apps-plugins-boundary`: an application compiles nothing from `plugins/`, and a plugin nothing from `apps/`.
   - `plugins-frontend`: ESLint over `plugins/`: no `@/**` or `@plugins/**` import.
   - `plugin-config-golden-parity`: Plugin Configs Match Golden Example plugins/.example/ & Root .golangci.yml.
   - `plugin-backend-layers`: Plugin Backend Internal Hexagonal Layer Direction.
   - `plugins-no-protocol-or-wire`: Plugins Depend Only on SDK (no libs/protocol, numen/v1/*.proto, @numen/protocol or @numen/wire).
   - `plugins-no-cross-plugin-knowledge`: Plugins Know Nothing of Other Plugins (no cross-plugin IDs, tab kinds, text kinds or command IDs).
   - `plugins-standalone-repo`: Plugins Are Standalone-Repo Ready (no ../ escapes, no vite.shared, no go.mod replace ../, no libs/notes, no plugin .proto).
3. **Applications (`apps`)**:
   - `apps-reach-no-plugin`: no application reaches `plugins/` by a relative path, an alias, an `import.meta.glob`, a tsconfig path or a Go import. A path is resolved from the file and from every package root above it.
   - `window-names-no-plugin-kind` (manual: runs only when named, because `apps/desktop/editor/src/entities/tab/model/unavailable.ts` still holds tables of plugin tab kinds until the plugin SDK states them): the shipped TypeScript and Vue sources of `apps/*`, `libs/ui/shell`, `libs/plugin-host-ts` and `libs/ui/core` state no plugin word of the dictionary above as a literal, object key, enum member, `case` label or identifier, read through the TypeScript AST.
   - `desktop-editor-fsd`, `desktop-flashcards-fsd`: the Feature-Sliced Design layer hierarchy (`shared` → `entities` → `features` → `widgets` → `pages` → `app`), by `npm run lint` in each window.
   - `frontend-segments-and-entities`: Frontend Segment Direction (ui -> model -> api -> lib) & No @x Entity Cross-Imports.
   - `vue-style-and-template`: Vue <style> Tokens & <template> Purity.
   - `apps-no-direct-protocol`: Applications Talk to Core Through SDK (no direct @numen/protocol in apps/*/src).
   - `desktop-flashcards-fsd`: Desktop Flashcards FSD Layer Hierarchy.

## Structural checks

Every check below derives its subject from the repository (the manifests, `go.work`, the directories) and names no plugin, package or file. Each declares a minimum count of what it examines, so a rule over nothing fails.

   - `plugins-isolation-capture`: one rule over the capture `plugins/(*)/`: a file of plugin X references plugin Y only when X equals Y. It reads Go imports, relative TypeScript and Vue imports, package names, package dependencies, paths and aliases. The Go import graph is checked by `libs/core/container/plugin_isolation_packages_test.go`, over the type-checked packages of every plugin module.
   - `no-suppressions`: no `nolint`, `eslint-disable`, `eslint-enable`, `@ts-expect-error`, `@ts-ignore` or `@ts-nocheck` comment; no `exclusions`, `exclude-functions`, `exclude-rules`, `exclude-dirs`, `exclude-files` key in a golangci configuration; no source file named in an eslint `ignores` or `files` entry; no constant of a check that lists repository files. Generated code is found by its `Code generated ... DO NOT EDIT` header and is not read.
   - `plugins-share-no-frontend-file`: no TypeScript or Vue file of one plugin repeats a file of another, found by identical tokens or, for files of one name, by token-shingle containment of 0.8 and above, which catches a copy that drifted. Tests, fixtures and the golden configuration files are not read.
   - `plugins-share-no-backend-file`: the same rule over the Go files of the plugins, test files included. Generated code and fixtures are not read. Code that two plugins need lives in `libs/plugin-sdk-go`, and test helpers they share live in its `agenttest` and `plugintest` packages.
   - `frontend-no-import-cycle`: the shipped TypeScript and Vue files of `apps`, `libs` and the plugin frontends form no cycle of runtime imports (relative and `@/` imports; `import type` is erased and not followed).
   - `every-path-has-a-layer`: every Go module and npm package lies under `libs`, `plugins`, `apps` or an auxiliary root (`tools`, `tests`, `docs`); no file lies directly in a layer root; every plugin directory has a `manifest.json`.
   - `workspace-lists-every-module`: `go.work` uses exactly the directories that hold a `go.mod`.
   - Before a verifier that runs `golangci-lint` starts, the linter must be built with a Go no older than the `go` line of `go.work`.

The checks are tested with violation fixtures: `npm test` in this directory.

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
