# Architectural Layer & Boundary Validator (`@numen/lint-layers`)

Automated verification of architectural layer boundaries and invariants across
libraries, plugins, and desktop applications.

## Invariants Checked

Each check has an id, a scope (`libs`, `plugins` or `apps`) and a module; `node check.mjs --list` prints them.

1. **Libraries (`libs`)**:
   - `core-hexagon`: the layer tests of `libs/core/container` (`layers_test.go`): inward dependencies, no machine access, logging or clock reads in the core, the core's own identities, and the ports and adapters it holds.
   - `machine-clock`: the type-aware test `libs/core/container/clock_port_test.go` loads every module of `go.work`, tests included, and fails on a reference to `time.Now`, `time.Since` or `time.Until` outside the body of a method `Now() time.Time` that implements a clock port.
   - `library-matrix`: every `go.mod` under `libs/` belongs to a family, the first directory under `libs/`, and the family is a row of `LIBRARY_DEPENDENCIES` in `checks/libraries.mjs`. The `go.mod` and the Go files of a library, tests included, reach only the modules of their own family, the families their row lists, and nothing outside `libs/`. A library with no row fails, so a new library states its place before it builds. The rows are `core-client` to `protocol`; `plugin-sdk` to `protocol` and `ui` (its Vite helpers); `plugin-host` to `plugin-sdk` and `core-client`; `plugin-manager` to `ui`.
   - `protocol-imported-only-by-clients`: Applications Reach the Core Only Through the Core Client. Only `@numen/plugin-sdk` and `@numen/core-client` import or declare `@numen/protocol`; the rule reads every `package.json` and every TypeScript and Vue file (tests included) under `apps`, `libs`, `plugins` and `tools`, outside `libs/protocol`, and names the owner of each reference. It fails when it reads fewer than 15 packages, 500 source files or 10 references of the two clients.
   - `core-client-private`: `libs/core-client/package.json` is named `@numen/core-client` and `"private": true`, and no workflow under `.github/workflows` that runs a publish command (`npm publish`, `pnpm publish`, `yarn publish`, `bun publish`, `changeset publish`) names `core-client`. It fails when it reads fewer than 10 workflows.
   - `package-matrix`: every `package.json` under `libs/` is a row of `PACKAGE_DEPENDENCIES` in `checks/libraries.mjs`, keyed by package name. The package declares, and its shipped `src` imports, only the first-party `@numen/*` packages its row lists. A library with no row fails, so a new library states its place before it builds.
   - `manifest-parsed-in-one-place`: a Go file or script of `libs`, `apps` or `tools`, tests included, or a shipped one of `plugins`, that names `manifest.json` or decodes a value named for the manifest (`json.Unmarshal(manifest, …)`) decodes no JSON of its own (`json.Unmarshal`, `json.NewDecoder`, `protojson`, `JSON.parse`, `.json()`) unless it decodes through the generated schema: Go reads `numen.v1.Manifest` with `protojson` in `libs/plugin-catalog/manifestjson`, and the scripts call `fromJson(ManifestSchema, …)` in `tools/shared/manifests.mjs` and in the SDK parser of `libs/plugin-sdk-ts`. The checks, `tools/ci-commands` and `tools/plugin-versions` read manifests through `tools/shared/manifests.mjs`.
   - `core-depguard`, `notes-depguard`, `sdk-isolation`: `golangci-lint` with the depguard rules of `libs/core`, `libs/notes` and `libs/plugin-sdk-go`.
   - `ui-isolation`: `libs/ui/core/src` names no `@numen/protocol`, `@numen/wire`, `@numen/shell` or `apps`, and `npm run lint` passes in `libs/ui/core`.
   - `shell-isolation`: `libs/ui/shell/src` names no `@numen/protocol`, `@numen/wire` or `apps`, and `npm run lint` passes in `libs/ui/shell`; the shell reaches the components through `@numen/ui` alone.
   - `plugin-host-isolation`: `libs/plugin-host-ts/src` imports only `vue`, `@numen/plugin-sdk` and `@numen/core-client` besides its own files (its tests also `vitest`, its `vite.ts` also `vite`), so the plugin loader knows nothing of `@numen/ui`, `@numen/shell`, the schema, the wire or an application; it fails when it reads fewer than 20 imports; and `npm run lint` passes in `libs/plugin-host-ts`.
   - `notes-hexagon`: Notes Library Internal Hexagonal Layers (layers_test.go).
   - `sdk-consumers`: Plugin SDK Imported Only By Plugins and Applications.
   - `go-no-mutable-globals`: Go No Package-Level Mutable Singletons or os.Exit Outside cmd/.
   - `notes-depguard`: Notes Library Depguard Boundaries.
   - `sdk-isolation`: Plugin SDK Go Isolation (no core or app imports).
   - `sdk-encapsulates-protocol`: Plugin SDK Encapsulates Protocol (no numenv1.* or window IDs in public SDK API).
   - `no-wire-library`: No @numen/wire Library (libs/wire removed; utilities in @numen/plugin-sdk, protocol adapters in host).
   - `docs-name-existing-paths`: Rule Documents Name Existing Paths (AGENTS.md, .agents/rules, lint-layers README).
   - `docs-name-existing-tests`: Rule Documents Name Declared Go Tests.
   - `docs-list-verifiers`: Layer Validator README Lists Every Registered Verifier.
   - `all-ts-packages-eslint`: Every TypeScript Package Has ESLint Configured.
   - `strict-lint-rules`: every directory with an `eslint.config.js` or `eslint.config.mjs`, and every plugin with a `package.json`, resolves (`eslint --print-config` on a source file and a test file) `no-void`, `no-empty` (`allowEmptyCatch: false`) and the typed `@typescript-eslint` rules `no-empty-function`, `no-floating-promises` (`ignoreVoid: false`), `no-misused-promises`, `await-thenable` and `switch-exhaustiveness-check` as errors. A package of scripts only (`tools/eslint.config.mjs`) resolves the untyped three (`no-void`, `no-empty`, `no-empty-function`) on a source script and a test script, from `createStrictScriptRules`. The shared definition is `createStrictRules` in `base-eslint.mjs`; `plugins/.example/eslint.config.js` carries the same block because a plugin config imports nothing outside its directory.
2. **Plugins (`plugins`)**:
   - `plugins-independence`: the `Plugin` tests of `libs/core/container`.
   - `plugins-leakage`: every `manifest.json` has an id, a name and a version; there is no shared plugin folder; no plugin imports `libs/core`; no plugin imports another, by a Go import, by a relative TypeScript or Vue import that resolves into another plugin's directory, or by another plugin's package name.
   - `plugins-reach-no-app`: no plugin reaches `apps/` by a relative path, an alias or a Go import.
   - `plugins-reach-no-shell`: no plugin names `@numen/shell` or `@numen/plugin-host`, or reaches `libs/ui/shell` or `libs/plugin-host-ts` by a relative path; a plugin takes components from `@numen/ui` and is loaded by the host, never loads itself.
   - `plugins-draw-no-notices`: no plugin imports `Notices`, `Notice`, `createNotice`, `createNoticeQueue` or `NoticeQueue` from a package; a plugin shows notices through `ctx.notices` and the window draws them in one list.
   - `plugins-write-data-through-host`: a shipped Go file of a plugin that names the `.numen` folder calls no `os` function that changes the file system (`os.WriteFile`, `os.MkdirAll`, `os.Rename`, `os.Remove`, …); a plugin writes its data through the host store.
   - `apps-plugins-boundary`: an application compiles nothing from `plugins/`, and a plugin nothing from `apps/`.
   - `plugins-frontend`: ESLint over `plugins/`: no `@/**` or `@plugins/**` import.
   - `plugin-config-golden-parity`: Plugin Configs Match Golden Example plugins/.example/ & Root .golangci.yml.
   - `plugin-backend-layers`: Plugin Backend Internal Hexagonal Layer Direction.
   - `plugins-no-protocol-or-wire`: Plugins Never Import @numen/protocol, @numen/core-client or the Wire (no libs/protocol, libs/core-client, numen/v1/*.proto, @numen/protocol, @numen/core-client or @numen/wire in source, or in a `package.json` dependency field); it fails when it reads fewer than 100 plugin sources.
   - `plugins-use-main-entry`: a plugin names `@numen/plugin-sdk` and `@numen/ui` through the main entry (`@numen/core-client` is not an entry a plugin may name; `plugins-no-protocol-or-wire` reports it). A deep path must be an entry the package declares in `exports`, and it passes only as the `testing` entry, as a stylesheet, or from a build configuration file (`*.config.ts`). The exports are read from the `package.json` of each package, and the rule fails when it reads fewer than 100 references.
   - `plugins-standalone-repo`: Plugins Are Standalone-Repo Ready (no ../ escapes, no vite.shared, no go.mod replace ../, no import or direct go.mod requirement of libs/notes (an `// indirect` line is allowed), no plugin .proto).
3. **Applications (`apps`)**:
   - `apps-reach-no-plugin`: no application reaches `plugins/` by a relative path, an alias, an `import.meta.glob`, a tsconfig path or a Go import. A path is resolved from the file and from every package root above it.
   - `desktop-editor-fsd`, `desktop-flashcards-fsd`: the Feature-Sliced Design layer hierarchy (`shared` → `entities` → `features` → `widgets` → `pages` → `app`), by `npm run lint` in each window.
   - `frontend-segments-and-entities`: Frontend Segment Direction (ui -> model -> api -> lib) & No @x Entity Cross-Imports.
   - `vue-style-and-template`: Vue <style> Tokens & <template> Purity.
   - `desktop-flashcards-fsd`: Desktop Flashcards FSD Layer Hierarchy.

## Structural checks

Every check below derives its subject from the repository (the manifests, `go.work`, the directories) and names no plugin, package or file. Each declares a minimum count of what it examines, so a rule over nothing fails.

   - `protocol-holds-only-host-contracts`: every top-level service, message and enum of `libs/protocol/proto/numen/v1` is used by a shipped Go, TypeScript or Vue source of `libs/core`, `libs/core-client`, the plugin SDKs, `libs/plugin-host-ts`, `libs/plugin-manager`, `libs/plugin-catalog`, `libs/ui` or `apps` (as an identifier, a generated type, a client or a handler constructor), or is referenced by a definition that is. A definition that only plugins use is a contract between plugins and fails. It fails when it reads fewer than 200 definitions or 500 sources. This check replaces the lists of plugin proto symbols, domain protos and source kind names.
   - `plugins-isolation-capture`: one rule over the capture `plugins/(*)/`: a file of plugin X references plugin Y only when X equals Y. It reads Go imports, relative TypeScript and Vue imports, package names, package dependencies, paths and aliases. The Go import graph is checked by `libs/core/container/plugin_isolation_packages_test.go`, over the type-checked packages of every plugin module.
   - `no-suppressions`: no `nolint`, `eslint-disable`, `eslint-enable`, `@ts-expect-error`, `@ts-ignore` or `@ts-nocheck` comment; no `exclusions`, `exclude-functions`, `exclude-rules`, `exclude-dirs`, `exclude-files` key in a golangci configuration; no source file named in an eslint `ignores` or `files` entry; no constant of a check that lists repository files. Generated code is found by its `Code generated ... DO NOT EDIT` header and is not read.
   - `plugins-share-no-frontend-file`: no TypeScript or Vue file of one plugin repeats a file of another, found by identical tokens or, for files of one name, by token-shingle containment of 0.8 and above, which catches a copy that drifted. Tests, fixtures and the golden configuration files are not read.
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
