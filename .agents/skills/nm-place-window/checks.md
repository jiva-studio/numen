# The checks that bound window code

Each entry names a check or a lint rule and what it forbids. The verifiers run with `node tools/lint-layers/check.mjs` (`--list` prints them); the ESLint rules run with `npm run lint` in the window. `.agents/rules/architecture.md` §10 is the list of the verifiers.

## Layers and slices

- `npm run lint` in the editor, the flashcards window and mobile (ESLint `boundaries/dependencies` from `tools/lint-layers/layers.mjs`): a file imports a layer above its own; a file in a slice layer imports a sibling slice; a layer above a slice layer imports anything of the slice except its `index.ts`; the window imports a plugin.
- `boundaries/no-unknown-files`: a file under `src/` that is in no layer.
- `every-path-has-a-layer`: a module of the repository that is in no layer.
- `dependency-graph`, rule `no-cross-slice-entry`: an `@x` cross-import.
- `dependency-graph`, rule `no-circular`: a cycle of runtime imports.
- `dependency-graph`, rules `segment-lib-imports-no-higher-segment`, `segment-api-imports-no-higher-segment`, `segment-model-imports-no-higher-segment`: `lib` importing `api`, `model` or `ui`; `api` importing `model` or `ui`; `model` importing `ui`.
- `package-matrix`, `protocol-declared-only-by-clients`, `core-client-private`: a window or library declaring a package its row does not allow; a package other than `@numen/core-client` and `@numen/plugin-sdk` declaring `@numen/protocol`; a plugin relying on `@numen/core-client`.
- `core-client-only-in-api`: a window file outside an `api` segment and `main.ts` importing `@numen/core-client`.
- `shared-serves-several-slices`: a file of `shared` that one slice imports.
- `app-is-composition`: a file of `app` over 300 lines, named `use<Name>` or exporting a `use<Name>` composable, calling `watch`, `addEventListener` or a timer, holding a `switch`, more than one decision (`if`, `?:`, loop, `catch`) in a function or more than three in a file.
- `no-component-below-ui`: a `.vue` file in a `lib`, `api` or `model` segment.

## Segments and components

- ESLint on `lib/`: importing Vue, Pinia or `@vue/*`; touching `window`, `document`, `navigator`, `localStorage` or `sessionStorage`; timers (`setTimeout`, `setInterval`).
- ESLint on `model/`: importing from `ui/`; touching `window`, `document` or `navigator` directly.
- ESLint on `.vue`: importing a wire module, `@connectrpc/*` or `@bufbuild/*`; touching `window`, `document` or `navigator`.
- ESLint `numen/max-composable-return-properties`: a composable returning more than eight properties, counted through a returned variable, or returning a spread.
- ESLint `numen/vue-script-sections`, `numen/max-script-lines`, `numen/max-inline-props`, `numen/thin-computed`, `numen/handler-naming`, `numen/boolean-prop-naming`, `numen/enforce-vue-reactivity-in-composable`: a script that departs from the section order and limits of `coding-style-frontend.md`.
- ESLint size rules `complexity`, `max-depth`, `max-lines-per-function`, `max-statements`, `max-params`: a function over the limits of `architecture.md` §8.

## Determinism and ports

- ESLint `no-restricted-syntax` (`createClockSelectors`): `Date.now()`, `new Date()` with no argument and `performance.now()` outside the two declarations that read the clock.
- `createPortConfig`: a clock, a frame request, a measurement or a source of randomness used by code that has to be held still by a test.

## Libraries and styles

- `package-matrix` and the library ESLint configs: `libs/ui/components` importing `@numen/protocol`, a wire type, a domain type or anything under `apps/**`.
- `styles-use-tokens` (stylelint): a color literal, a `--numen-` declaration outside the token and theme files, a `var()` fallback, an undeclared token.
- `plugins-frontend`: `@/**` or `@plugins/**` imported by a plugin frontend.
- `styles-use-tokens` also holds the rule `numen/no-bare-length`: no `px` for a size except a 1px hairline border, no bare length on a property, and a local custom property only in `rem` or `em`.

## Whole repository

- `no-suppressions`: `eslint-disable`, `@ts-ignore`, `@ts-expect-error`, `@ts-nocheck` and `nolint` comments.
- `strict-lint-rules`: a package whose ESLint configuration does not resolve the strict rule set.
- `all-ts-packages-eslint`: a TypeScript package with no ESLint configuration.
- `typecheck-covers-sources`: a `tsconfig` that excludes a tracked file.

## A red check

A red check is fixed in the code, not by changing the check or moving code out of its sight. The fix is the placement that the check points at: a core call moves into the `api` segment of the slice that owns the type, a file used by one slice moves into that slice, a component moves into `ui`, a composable that returns too much is split by responsibility. A core call stays visible to the check: it is in an `api` segment, and `shared/api` is the only place that re-exports the clients. A file is not renamed or relocated to leave a layer.
