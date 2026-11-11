# What the checks enforce

Each entry names what the check forbids. The registered checks are printed by `make checks LIST=1` and run with `make checks TAGS=library`; the rules are in `tools/checks/`, `tools/lint-layers/` and the `eslint.config.js` of each package.

A red check is fixed in the code, not by changing the check or moving code out of its sight.

## Package boundaries

- `package-matrix`: a `package.json` under `libs/` that declares a first-party package its row does not list, and a library with no row.
- `dependency-graph` (dependency-cruiser): an import outside the row of the package matrix, a plugin import of anything first-party but `@numen/plugin-sdk` and `@numen/ui` or of `@connectrpc`, an application import of `@numen/plugin-sdk` or `@numen/protocol`, a relative import out of the package's own directory, third-party code in the plugin host other than Vue and Vite, a `lib`, `api` or `model` segment that imports a higher segment of its slice, an `@x` cross-import, and a cycle of runtime imports.
- `protocol-declared-only-by-clients`: a `package.json` other than `@numen/plugin-sdk` and `@numen/core-client` that declares `@numen/protocol`.
- `core-client-private`: a public `@numen/core-client`, and a plugin that relies on it.
- `sdk-ts-encapsulates-protocol`: an exported declaration of `@numen/plugin-sdk` that reaches a declaration of `libs/core/protocol`.
- `plugins-use-main-entry`: a plugin that imports a deep path of the SDK or `@numen/ui` that is not a declared entry.
- `plugins-frontend`, `plugins-isolation-capture`, `plugins-hold-no-schema`, `plugins-leakage`: a plugin frontend that imports through `@/**` or `@plugins/**`, one plugin that names another, a `.proto` file inside a plugin, a plugin that imports `libs/core/kernel`.
- `every-path-has-a-layer`, `workspace-lists-every-module`, `all-ts-packages-eslint`: a module in no layer, a Go module missing from `go.work`, a TypeScript package with no ESLint configuration.
- `npm run lint` in `apps/desktop/editor`, `apps/desktop/flashcards` and `apps/mobile` (ESLint `boundaries/dependencies`): a window module that imports a higher FSD layer, a slice that imports a sibling slice, a slice reached other than through its `index.ts`.

## Inside `@numen/ui` and `@numen/workbench` (`npm run lint` in `libs/ui/components` and `libs/ui/workbench`)

- An import of any first-party package but `@numen/ui`, or of a path under `apps` or `libs`, in the sources and tests of the library. Stories and fixtures stand outside the layer order and carry the same import ban in a block of their own (`createStoryImportConfig` in `tools/lint-layers/layers.mjs`, self-tested by `layers.test.mjs`).
- A layer that imports a higher layer, a slice that imports another slice, and an import of a slice by any file but its `index.ts`.
- `Date.now()`, `new Date()` with no argument and `performance.now()` outside the one function that is the clock; `Math.random()`; `getBoundingClientRect` and `matchMedia` in source files; `setTimeout` and `setInterval` in `lib` and `domain` segments; the other globals with a lifetime (`requestAnimationFrame` and the like) outside a port.

## Styles

- `styles-use-tokens`: a colour literal outside a token or theme file, a `--numen-` declaration outside those files, a `var()` with a fallback, and a `var(--numen-…)` that no token file declares.
- `styles-use-tokens` also holds the rule `numen/no-bare-length`: no `px` for a size except a 1px hairline border; a `rem` or `em` literal may stand on a property; a `rem` or `em` literal may also stand in any custom property.

## Shape and discipline

- `strict-lint-rules`: an empty `catch`, an empty function, a floating or misused promise, `void`, an unawaited thenable, a non-exhaustive `switch` over a union.
- `no-suppressions`: `eslint-disable`, `@ts-ignore`, `@ts-expect-error`, `@ts-nocheck`, `nolint`, an `exclude` entry in a linter configuration, a file named in an `ignores` or `files` entry.
- Complexity limits of function and Vue block (`complexity`, `max-depth`, `max-lines-per-function`, `max-statements`, `max-params`, and the Vue block limits) apply to all source; tests and stories are outside them.
- `lint_comments`: a comment that argues, narrates history or repeats the code; a comment on a field, prop or private function.
- `lint_narrative`: identifiers, test titles, comments and commit subjects in narrative English.
- `json-keys-camelcase`: a JSON key that is not camelCase in a manifest, API fixture or TypeScript JSON type.
- `make generate-check`: generated code that is out of date with the schema.

## Placement checks of windows and plugin frontends

- `core-client-only-in-api`: a window file other than an `api` segment and the window entry that imports `@numen/core-client`.
- `no-component-below-ui`: a component in a `lib`, `api` or `model` segment.
- `shared-serves-several-slices`: a file in a window's `shared` layer that only one slice imports.
- `numen/max-composable-return-properties`: a composable that returns more than eight properties, counted through a returned variable, returns a spread, a group of one key, or a group named for what it wraps (`base`, `parts`, `services`, `actions`, `frame`, `api`, `methods`, `handlers`, `helpers`, `utils`, `deps`); the names a group holds count toward the eight.
- `typecheck-covers-sources`: a `tsconfig` that excludes a tracked file.
- A plugin frontend `lib` segment that imports a `ui`, `model` or `api` segment or a component, and a plugin component that breaks the inline-props, script-sections and script-length limits.
