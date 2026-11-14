# Worked examples

Each example follows the target in `SKILL.md`. Line numbers are those of the commit this skill was written at.

## A general control: the slider

`libs/ui/components/src/shared/ui/slider/` holds `Slider.vue`, `track.ts`, `index.ts`, `Slider.test.ts`, `Slider.outside.test.ts`, `track.test.ts` and `Slider.stories.ts`.

- The props are `min`, `max`, `step`, `isDisabled` and a bound value, all numbers and booleans in the control's own terms (`Slider.vue:19-32`). The event is `commit` with the value (`Slider.vue:34-36`).
- The arithmetic of keys and bounds is pure and lives in `track.ts` next to the component (`track.ts:9-40`), imported by the component (`Slider.vue:12-14`). The `.vue` file wires; the file beside it decides.
- The folder exports its one component through `index.ts`, and `src/index.ts` re-exports it.
- The test mounts the component and asserts what a screen reader receives and what is emitted (`Slider.test.ts:40-50`). The story sets knobs and runs in both engines (`Slider.stories.ts:19-50`).

Pattern for a new control: a folder in `shared/ui/`, the component, the types file when props exceed a few lines, the pure helper beside it, an `index.ts`, a test, a story.

## Clock and viewport as parameters: the menu

`libs/ui/components/src/shared/ui/menu/Menu.types.ts:4-16` declares the props `items`, `at`, `isOpen`, `viewport` and `clock`. `Menu.vue:17-28` gives `viewport` and `clock` browser defaults. The event `choose` returns the identifier the caller gave the item (`Menu.types.ts:22`). The menu knows no domain: the caller maps notes, tabs or cards to `MenuItem`s and maps the returned identifier back.

The clock parameter takes the shape `Clock` from `shared/lib/time/clock.ts` (`now`, `schedule`, `cancel`), whose `browserClock` in `shared/api/clock.ts` is the default; `getNow()` and `getMonotonicNow()` are the only functions that read the machine clock, and the lint selectors allow the reads there alone. A test of the menu passes its own clock and decides when time passes.

## A feature slice: the tree

`libs/ui/components/src/features/tree/` is a slice: `index.ts` exports `Tree` and two types; `ui/` holds `Tree.vue`, `TreeRow.vue` and `tree-field/`; `model/` holds drag, gestures, layout, rows, selection and the virtual window; `lib/` holds pure row, drop, step and virtualization functions; `fixtures/rows.ts` holds invented rows.

- Props and events are in `ui/Tree.types.ts:7-30`: `rows` with opaque `RowId`s, `open`, `selected`, a `clock`, and events `open`, `close`, `select`, `rename`, `move`, `remove`, each carrying identifiers.
- The direction inside the slice is `ui` to `model` to `lib`, and nothing outside the slice imports a file of it except through `index.ts`.
- The tests sit beside what they test (`model/*.test.ts`, `lib/*.test.ts`, `ui/Tree.test.ts`), and `ui/Tree.stories.ts` runs the behavior in the browser projects.

Pattern for a component with its own state: a slice with the three segments, one public `index.ts`, identifiers in and out.

## A domain list kept out of the component library: the plugin list

The list of plugins names a domain, so it is not in `@numen/ui`. It is its own library, `libs/plugin/manager`, which imports `@numen/ui` and nothing else.

- The library defines the shapes it draws: `PluginListing` (`src/plugins.ts:7`) and `ManagedPlugin` (`src/plugins.ts:22`), and takes them by props (`src/index.ts` exports `PluginsList`, `PluginDetail`, `PluginSettingsCard`).
- The query parser and filter are pure functions in `src/filter.ts` (`filter.ts:1-50`) with `src/filter.test.ts`.
- The window maps the catalog to those shapes: `apps/desktop/editor/src/pages/settings/lib/pluginListing.ts:14` turns a `CatalogEntry` of the window's `entities/plugin` into a `PluginListing`; the entity maps the entry and its state from `@numen/core-client` in `api/catalog.ts`.
- `@numen/core-client` owns the catalog client and its entry types (`libs/core/client/src/catalog.ts:18-40`), over the generated messages.

Pattern for a domain view: the library defines its own view-model and takes it by props; the window owns the mapping from the wire; the client library owns the call.

## Window chrome in the shell: the palette

`libs/ui/workbench/src/features/palette/` is a slice with `ui/`, `model/`, `lib/`, `fixtures/`, `types.ts` and `index.ts`. It takes `PaletteItem`s and `PaletteGroup`s and emits identifiers; `src/index.ts` exports `Palette`, `commandKeyChord` and the types. It builds on `@numen/ui` (`keyChord`, `stepTo`, `Spinner`, `runInBackground`) and imports no wire type. The window supplies the commands and runs what the palette returns.

## Where the code in `main` differs from the target

The code below does not follow the target. It is not a model.

- Doc comments in `libs/ui/components/src/index.ts` name domain things: a "moment of a recording" for `clock`, a "manifest" for `NamedIcon`, "both windows" for the number formatters. The exports are general; the comments name their first consumer.
- `libs/ui/workbench/src/features/welcome` exposes `VaultRow` and `getVaultForKey`; the shell's other slices (workspace, palette, notices, window frame) are chrome.
- `Menu.types.ts:16` takes `clock?: () => number` while `Tree.types.ts:15` takes `clock?: Clock`. The target is `Clock`.
- `libs/core/client/src/errors/codes.ts:19` words a code with the domain term "vault". The target is sentences stated once as data. Clients are created from a transport parameter (`create*Client(transport)`); no client is created at import time.
