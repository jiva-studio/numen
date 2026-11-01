# Architectural Layer & Boundary Validator (`@numen/lint-layers`)

Automated verification of architectural layer boundaries and invariants across
libraries, plugins, and desktop applications.

## Invariants Checked

Each check has an id, a scope (`libs`, `plugins` or `apps`) and a module; `node check.mjs --list` prints them.

1. **Libraries (`libs`)**:
   - `go-structure`: every test of `libs/core/container`, the package that holds the structural Go tests. They load the modules of `go.work` with go/packages, go/types, the AST and golang.org/x/mod/modfile: the layers of the core and the direction of its imports, the machine clock read only by a clock port implementation, the plugin dependency rule over the type-checked import graph and over the direct requirements of each `plugins/*/go.mod`, and the plugin SDK encapsulating the whole `libs/protocol` module. It runs on any change to a Go file, a `go.mod` or `go.work`.
   - `library-matrix`: every `go.mod` under `libs/` belongs to a family, the first directory under `libs/`, and the family is a row of `LIBRARY_DEPENDENCIES` in `checks/libraries.mjs`. The `go.mod` and the Go files of a library, tests included, reach only the modules of their own family, the families their row lists, and nothing outside `libs/`. A library with no row fails, so a new library states its place before it builds. The rows are `core-client` to `protocol`; `plugin-sdk` to `protocol` and `ui` (its Vite helpers); `plugin-host` to `plugin-sdk` and `core-client`; `plugin-manager` to `ui`.
   - `protocol-declared-only-by-clients`: only `@numen/plugin-sdk` and `@numen/core-client` declare `@numen/protocol` in a `package.json` under `apps`, `libs`, `plugins` or `tools`; the imports are `dependency-graph`'s. It fails when it reads fewer than 15 packages or finds the two clients declaring the schema fewer than twice.
   - `core-client-private`: the package named `@numen/core-client` is `"private": true`, and it is not among the packages plugins rely on: the first-party packages a `package.json` under `plugins/` declares, and those they bring in through their `dependencies`, `peerDependencies` and `optionalDependencies`. It fails when that set is empty or it reads fewer than 15 packages.
   - `package-matrix`: every `package.json` under `libs/` is a row of `PACKAGE_DEPENDENCIES` in `checks/libraries.mjs`, keyed by package name, and declares only the first-party `@numen/*` packages its row lists. A library with no row fails, so a new library states its place before it builds. `dependency-graph` holds the imports of each library to the same row.
   - `core-depguard`, `notes-depguard`, `sdk-isolation`: `golangci-lint` with the depguard rules of `libs/core`, `libs/notes` and `libs/plugin-sdk-go`.
   - `dependency-graph`: dependency-cruiser (`dependency-cruiser.mjs`, rules built in `checks/dependencies.mjs` from the packages under `libs`, `apps` and `plugins`) over the `src` of every library and application and the `frontend` of every plugin, type imports included. A plugin reaches only `@numen/plugin-sdk` and `@numen/ui` of the first-party packages and nothing of `@connectrpc`; an application reaches every library but `@numen/plugin-sdk` and `@numen/protocol`; a library reaches the row of the package matrix; no package reaches a file outside its directory by a relative path; the plugin host reaches Vue and Vite alone of third-party code; a `lib`, `api` or `model` segment imports no higher segment of its slice outside tests and stories; nothing imports through `@x`; and no files form a cycle of runtime imports. A package whose `tsconfig.json` declares `paths` is cruised alone with it, so its aliases resolve. It fails when it reads fewer than 1000 modules.
   - `styles-use-tokens`: stylelint (`stylelint.config.mjs` at the root, rules of its own in `stylelint-plugin.mjs`) over every tracked `.css` file and the `<style>` blocks of every `.vue` file: no hex colour, named colour or colour function outside a token file (a `tokens` directory) and a theme file (a `theme` or `themes` directory); no `--numen-` declaration outside those; no `var()` with a fallback; and no `var(--numen-…)` that no token file declares, the declared set read from the syntax trees of the token files. It fails when it reads fewer than 150 files.
   - `ui-lint`: `npm run lint` passes in `libs/ui/core`.
   - `shell-lint`: `npm run lint` passes in `libs/ui/shell`.
   - `plugin-host-lint`: `npm run lint` passes in `libs/plugin-host-ts`.
   - `notes-hexagon`: every test of the root package of `libs/notes` (`layers_test.go`, `purity_test.go`).
   - `notes-depguard`: Notes Library Depguard Boundaries.
   - `sdk-isolation`: Plugin SDK Go Isolation (no core or app imports).
   - `sdk-ts-encapsulates-protocol`: `checks/sdkProtocol.mjs` builds a TypeScript program from every script entry of the `exports` of `libs/plugin-sdk-ts` (a condition object is read by its `types`, `import` or `default` target, a `*` pattern by the files it matches) and fails when the type of an exported declaration reaches a declaration located in `libs/protocol`: a re-export, a signature, a member, a type argument, a base type or a constraint, found through the type checker. It fails when it reads fewer than 2 entries, 100 exported declarations, 15 SDK source files or no file of the schema.
   - `all-ts-packages-eslint`: every directory under `libs` or `apps` that holds a `tsconfig.json` and a `package.json` has an `eslint.config.js`; it fails when it reads fewer than 8 packages.
   - `strict-lint-rules`: every directory with an `eslint.config.js` or `eslint.config.mjs`, and every plugin with a `package.json`, resolves (`eslint --print-config` on a source file and a test file) `no-void`, `no-empty` (`allowEmptyCatch: false`) and the typed `@typescript-eslint` rules `no-empty-function`, `no-floating-promises` (`ignoreVoid: false`), `no-misused-promises`, `await-thenable` and `switch-exhaustiveness-check` as errors. A package of scripts only (`tools/eslint.config.mjs`) resolves the untyped three (`no-void`, `no-empty`, `no-empty-function`) on a source script and a test script, from `createStrictScriptRules`. The shared definition is `createStrictRules` in `base-eslint.mjs`; `plugins/.example/eslint.config.js` carries the same block because a plugin config imports nothing outside its directory.
2. **Plugins (`plugins`)**:
   - `plugins-independence`: the `Plugin` tests of `libs/core/container`.
   - `plugins-leakage`: every `manifest.json` has an id, a name and a version; there is no shared plugin folder; no plugin imports `libs/core`.
   - `plugins-hold-no-schema`: no tracked file under `plugins/(*)/` is a `.proto` schema; a plugin reaches the core through the SDK.
   - `plugins-frontend`: ESLint over `plugins/`: no `@/**` or `@plugins/**` import.
   - `plugin-config-golden-parity`: Plugin Configs Match Golden Example plugins/.example/ & Root .golangci.yml.
   - `plugin-backend-layers`: Plugin Backend Internal Hexagonal Layer Direction.
   - `plugins-use-main-entry`: a plugin names `@numen/plugin-sdk` and `@numen/ui` through the main entry. A deep path must be an entry the package declares in `exports`, and it passes only as the `testing` entry, as a stylesheet, or from a build configuration file (`*.config.ts`). The exports are read from the `package.json` of each package, and the rule fails when it reads fewer than 100 references.
3. **Applications (`apps`)**:
   - `desktop-editor-fsd`, `desktop-flashcards-fsd`: the Feature-Sliced Design layer hierarchy (`shared` → `entities` → `features` → `widgets` → `pages` → `app`), by `npm run lint` in each window.
   - `desktop-flashcards-fsd`: Desktop Flashcards FSD Layer Hierarchy.

## Structural checks

Every check below derives its subject from the repository (the manifests, `go.work`, the directories) and names no plugin, package or file. Each declares a minimum count of what it examines, so a rule over nothing fails.

   - `plugins-isolation-capture`: one rule over the capture `plugins/(*)/`: a Go import or a package dependency of plugin X names plugin Y only when X equals Y. The imports of scripts are `dependency-graph`'s, and the Go import graph is checked by `libs/core/container/plugin_isolation_packages_test.go`, over the type-checked packages of every plugin module.
   - `no-suppressions`: no `nolint`, `eslint-disable`, `eslint-enable`, `@ts-expect-error`, `@ts-ignore` or `@ts-nocheck` comment; no `exclusions`, `exclude-functions`, `exclude-rules`, `exclude-dirs`, `exclude-files` key in a golangci configuration; no source file named in an eslint `ignores` or `files` entry; no array of repository files, in a constant or inline, in a script of `tools/lint-layers` or `stylelint.config.mjs`. Generated code is found by its `Code generated ... DO NOT EDIT` header and is not read. It reads comments with `typescript` and `@vue/compiler-sfc`, which `package.json` declares.
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
make check_dependencies               # dependency-cruiser alone (dependency-graph)
make check_styles                     # stylelint alone (styles-use-tokens)

# Directly:
node check.mjs [target...]
node check.mjs --list
node check.mjs --help
```
