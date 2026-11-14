# What the checks enforce

Each entry names a check or a lint rule by what it forbids. `make checks LIST=1` prints the registered checks and `make checks TAGS=plugin` runs them; the rules are in `.agents/rules/architecture.md` §5, §8, §9, §10 and §11. A red check is fixed in the code, not by changing the check or moving code out of its sight.

## Dependencies of a plugin

- `go.mod` of a plugin (`make checks ONLY=go/plugin-gomod`, tests `TestNoPluginModuleRequiresMoreThanTheSDK`, `TestWhatThePluginGoModRulesReject`): the module path is the folder's; it requires directly only the plugin SDK modules and third-party code; it names no `libs/note`, `libs/core/protocol`, connect or protobuf module, no other plugin and no core package; it has no `replace` to a folder.
- The import graph of the plugins on fourteen platform targets (`make checks ONLY=go/plugin-boundary,go/plugin-isolation-packages`): a plugin reaches no other plugin, no core package, no application.
- `go/plugin-copies-agree` (`make checks ONLY=go/plugin-copies-agree`): the ONNX runtime package `backend/internal/adapter/engine/onnx` of `engines-embeddings`, `engines-ocr` and `engines-transcription` has the same files with the same content in each plugin, apart from the files the rule names as intended differences (none). A change to one copy is made in all three.
- `plugins-isolation-capture`: a file under `plugins/<name>/` references no other plugin.
- `package.json` of a plugin (`package-matrix`, `protocol-declared-only-by-clients`, `core-client-private`): it declares no `@numen/protocol` and no `@numen/core-client`, and among the packages of this repository only `@numen/plugin-sdk` and `@numen/ui`.
- `plugins-use-main-entry`: the SDK and the UI library are imported through the entries their `exports` declare.
- `make plugins_lint` (ESLint with the golden `eslint.config.js`): no `@/**` and no `@plugins/**` import, and no `@numen/*` import other than `@numen/plugin-sdk` and `@numen/ui`.
- `plugins-hold-no-schema`: no `.proto` file under `plugins/`.
- `sdk-isolation` and `sdk-ts-encapsulates-protocol`: the SDKs import no core and no application, and no exported SDK declaration exposes a generated protocol type.
- The no-reexport rule of the core: a package declares no alias of a type of another package. Types a plugin uses come from the SDK under one name each.

## The manifest and the host

- `plugins-leakage`: the manifest schema, and static non-leakage of the core into plugins.
- `json-keys-camelcase`: every JSON key of a manifest, fixture and JSON type is camelCase.
- Manifest tests of `libs/core/kernel/container`: every manifest passes the validator; a plugin that registers tabs declares them; no wildcard activation event; every setting declared is read by the plugin; every plugin has a README; the plugin names follow the naming rule; the core names no plugin id.
- `host_permission_test.go`: the host refuses a call whose permission the plugin's manifest does not declare, for any client of the host endpoint.

## Tooling configuration

- `plugin-config-golden-parity`: `Makefile`, `.golangci.yml`, `eslint.config.js` and `tsconfig.json` of a plugin (and `numen-rules.mjs`) are the files of `plugins/.example/` byte for byte, and the golden Go linters equal those of the root `.golangci.yml`.

## Inside a Go backend

- `plugin-backend-layers`: a package under `backend/` other than `cmd`, `service`, `engine`, `wire` and `testsupport` imports none of `backend/service`, `backend/engine` and `backend/wire`; a package under `backend/wire` imports none of `backend/service`, `backend/engine` and `backend/internal/adapter/*`.
- `plugin-service-size`: the non-test code of `backend/service` is at most 600 lines, and every Go package of a plugin backend has non-test files of at most 500 lines and at most 3000 lines in all; a service over its limit fails and is told which of the `domain`, `usecase` and `ports` packages it lacks. The rule fails when it reads no service.
- `plugin-service-imports`: a non-test file of `backend/service` imports only the allow list: `bytes`, `cmp`, `context`, `encoding/json`, `errors`, `fmt`, `maps`, `math`, `net/http`, `path`, `regexp`, `slices`, `sort`, `strconv`, `strings`, `time`, `unicode`, `unicode/utf8`, the packages of `libs/plugin/sdk-go`, the `api`, `domain`, `usecase`, `ports` and `wire` packages of its own plugin and its own `internal/adapter/*` packages; machine access, state and the other packages live behind a use case. The rule fails when it reads no file.
- The Go lint configuration of the plugin (`.golangci.yml`, `.golangci-structure.yml`): cyclomatic complexity, cognitive complexity, nesting, function length, parameter count, an interface of at most five methods, exhaustive switches and struct literals, no naked returns, no package-level mutable state and no `init`, no discarded errors, `errors.Is`/`errors.As`, US spelling in comments; depguard keeps the domain free of adapters, `os` and `net`.
- The machine clock (`time.Now`, `time.Since`, `time.Until`) is read only inside a method `Now() time.Time` of a type that implements a declared clock interface. Tests are in scope.
- No function of a port calls a method of an interface value; an interface's full method set, embedded methods included, stays within the limit.
- No package lies below the composition root of the core.

## Inside a frontend

- Golden `eslint.config.js` for every plugin frontend: the strict promise and `void` rules; `complexity`, `max-depth`, `max-params`; no `console`; no `any`; no `Date.now()`, `new Date()`, `Math.random()`, `getBoundingClientRect` or `matchMedia` outside tests and stories.
- `lib/` segments: no import of Vue, Pinia or `@vue/*`; no import of a `ui`, `model` or `api` segment at any depth and no `.vue` file; no `window`, `document`, `navigator`, `localStorage`, `sessionStorage`; no `setTimeout` or `setInterval`.
- `model/` segments: no import from `ui`; no `window`, `document`, `navigator`.
- `.vue` files: no import of a wire file, `@connectrpc/*` or `@bufbuild/*`; no `window`, `document`, `navigator`.
- Components (`numen-rules.mjs`): at most six inline props (`numen/max-inline-props`), the statement order props and events, state, hooks, handlers, helpers (`numen/vue-script-sections`), `on<Action>` names for template event handlers (`numen/handler-naming`), at most 100 script lines (`numen/max-script-lines`); the template, depth and composable-return limits of §8.
- `no-component-below-ui`: no `.vue` file in a `lib`, `api` or `model` segment.
- `styles-use-tokens`: no colour literal, no `--numen-` declaration outside the token files, no `var()` fallback and no undeclared token in a style sheet.
- Every package with a `package.json` or an ESLint configuration is linted with the strict rule set, and no `tsconfig` excludes a tracked file.

## Everywhere

- `no-suppressions`: no `nolint`, `eslint-disable`, `@ts-ignore`, `@ts-expect-error` or `@ts-nocheck`; no `exclusions` or `exclude-functions` in a linter configuration; no file named in an `ignores` entry to avoid a rule.
- `style/names` (`make lint_narrative`): identifiers, test titles, comments and commit subjects are written in plain technical English.
- Version of a changed plugin: `<year>.<month>.<build>` raised once relative to the merge base with main.

