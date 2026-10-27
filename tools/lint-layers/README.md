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
   - `ui-isolation`: `libs/ui/core/src` names no `@numen/protocol`, `@numen/wire`, `@numen/shell` or `apps`, and `npm run lint` passes in `libs/ui/core`.
   - `ui-holds-no-plugin-view`: `libs/ui/core/src` holds none of the directories of a view a plugin owns and `src/index.ts` exports none of its names (`views-agent-chat`: `screens/agent`, `features/thread`, `features/message-composer`; `Agent`, `Thread`, `MessageComposer`, `useConversation`, `Conversation`, `Turn`, `AgentModelOption`, `AgentPort`, `AgentStep`).
   - `ui-holds-no-shell-part`: `libs/ui/core/src` holds none of the window chrome `@numen/shell` owns (`features/workspace`, `features/palette`, `features/status-bar`, `features/notices`, `features/welcome`, `screens`) and `src/index.ts` exports none of its names (`WorkspaceLayout`, `Palette`, `StatusBar`, `Notices`, `createNoticeQueue`, `WelcomePage`, `getVaultForKey`, `VaultRow`, `WelcomeAction`).
   - `ui-exports-no-editor`: `libs/ui/core/src` holds no `features/editor`, and `src/index.ts` exports nothing from it and no wikilink helper (`Editor`, `createTimeGutter`, `Extension`, `isNoteAddress`, `getWikilinks`). The notes plugin owns the editor; the core parses links.
   - `protocol-holds-no-plugin-contract`: the code of `libs/protocol/proto/numen/v1` names nothing a single plugin owns: `TranscriptService`, `ArticleService`, `BookService`, `DocumentService`, `SOURCE_KIND_BOOK`, `SOURCE_KIND_RECORDING`, `SOURCE_KIND_URL`, `BookFormat`, `FILE_MENU_ACTION_RECOGNISE`, `FILE_MENU_ACTION_PROOFREAD`, `BookProgress`, `RecordingProgress`, `SetVaultInUse` and its messages, and the `mark` and `face` fields of `agent.proto`. Comments are not read.
   - `no-file-kind-outside-plugins`: the shipped code of `libs/core`, `libs/notes`, `libs/plugin-sdk-go`, `libs/plugin-sdk-ts`, `libs/ui/core`, `libs/ui/shell` and `apps/desktop/editor/src` holds no literal of a plugin's file extension (`"pdf"`, `".epub"`, `"onFile:*.pdf"`, `".mp3"`, … in `PLUGIN_EXTENSIONS`), no literal `"book"`, `"recording"`, `"url"` or `"document"`, and no name of those kinds (`KindBook`, `TabRecording`, `CapabilityDocument`, `SOURCE_KIND_URL`, `SourceKind.BOOK`, `BookFormat`, `DefaultBookExtensions`, `BookProgress`, `ReadBooks`, `BookServiceName`, and the script constants `BOOK`, `RECORDING`, `DOCUMENT`). A double-quoted string inside a literal, such as a manifest in a Go raw string, is read too. Tests, stories, `testing/`, `testdata/`, `e2e/` and generated code are not read. The core finds a plugin by content type instead.
   - `shell-isolation`: `libs/ui/shell/src` names no `@numen/protocol`, `@numen/wire` or `apps`, and `npm run lint` passes in `libs/ui/shell`; the shell reaches the components through `@numen/ui` alone.
   - `plugin-host-isolation`: `libs/plugin-host-ts/src` imports only `vue` and `@numen/plugin-sdk` besides its own files (its tests also `vitest`, its `vite.ts` also `vite`), so the plugin loader knows nothing of `@numen/ui`, `@numen/shell`, the protocol, the wire or an application; and `npm run lint` passes in `libs/plugin-host-ts`.
2. **Plugins (`plugins`)**:
   - `plugins-independence`: the `Plugin` tests of `libs/core/container`.
   - `plugins-leakage`: every `manifest.json` has an id, a name and a version; there is no `plugins/shared`; no plugin imports `libs/core`; no plugin imports another, by a Go import, by a relative TypeScript or Vue import that resolves into another plugin's directory, or by another plugin's package name.
   - `plugins-reach-no-app`: no plugin reaches `apps/` by a relative path, an alias or a Go import.
   - `plugins-reach-no-shell`: no plugin names `@numen/shell` or `@numen/plugin-host`, or reaches `libs/ui/shell` or `libs/plugin-host-ts` by a relative path; a plugin takes components from `@numen/ui` and is loaded by the host, never loads itself.
   - `plugins-draw-no-notices`: no plugin imports `Notices`, `Notice`, `createNotice`, `createNoticeQueue` or `NoticeQueue` from a package; a plugin shows notices through `ctx.notices` and the window draws them in one list.
   - `plugins-name-no-window`: the shipped code of a plugin holds no window id literal (`"editor"`, `"flashcards"`) and names no `WindowService` or `NewWindowServiceClient`; a plugin reaches the window through the SDK.
   - `plugins-write-data-through-host`: a shipped Go file of a plugin that names the `.numen` folder calls no `os` function that changes the file system (`os.WriteFile`, `os.MkdirAll`, `os.Rename`, `os.Remove`, …); a plugin writes its data through the host store.
   - `apps-plugins-boundary`: an application compiles nothing from `plugins/`, and a plugin nothing from `apps/`.
   - `plugins-frontend`: ESLint over `plugins/`: no `@/**` or `@plugins/**` import.
3. **Applications (`apps`)**:
   - `apps-reach-no-plugin`: no application reaches `plugins/` by a relative path, an alias, an `import.meta.glob`, a tsconfig path or a Go import. A path is resolved from the file and from every package root above it.
   - `window-names-no-plugin-kind`: the editor window's sources name no plugin kind (`deck`, `stencil`, `preset`, `plex`, `agent`, `files`) as a literal, outside its tests and stories and the files listed in `CORE_NAME_FILES` (the settings entity and page and the plugin context).
   - `desktop-editor-fsd`, `desktop-flashcards-fsd`: the Feature-Sliced Design layer hierarchy (`shared` → `entities` → `features` → `widgets` → `pages` → `app`), by `npm run lint` in each window.

## Baseline

The five checks `protocol-holds-no-plugin-contract`, `no-file-kind-outside-plugins`, `ui-exports-no-editor`, `plugins-name-no-window` and `plugins-write-data-through-host` read `baseline.json`: the violations known when they were added, by check id, as `<file>: <symbol>` and the number of times the symbol occurs in that file. Lines are left out, so an edit above a violation does not move it.

A check fails on a violation the baseline does not hold, and on a baseline entry found fewer times than it lists. A change that removes a violation runs `node check.mjs --update-baseline <check>` and commits the lowered `baseline.json`. `--update-baseline` only lowers counts and removes entries; it never adds one. `--no-baseline` lists every violation with its line.

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
node check.mjs no-file-kind-outside-plugins --no-baseline   # every violation, with its line
node check.mjs --update-baseline                             # drop baseline entries that no longer occur
```
