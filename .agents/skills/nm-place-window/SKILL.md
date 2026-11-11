---
name: nm-place-window
description: Says which Feature-Sliced Design layer, slice and segment a piece of window frontend code belongs to, and how a slice is shaped. Load it before adding or moving code in apps/desktop/editor, apps/desktop/flashcards or another window (apps/mobile, apps/landing, apps/docs), and when reviewing where a change puts a core call, a type, a command, a composable, a component or a story.
paths:
  - 'apps/**'
---

# Placing window code (`/nm-place-window`)

A window is a thin shell. It composes plugins and libraries, routes requests to the core through `@numen/core-client`, and holds little logic of its own. The business logic of a capability belongs to the core or to a plugin. A small piece of window-specific logic may live in the window, for example where flashcards live, which the flashcards window decides when it picks the vault its tab opens on.

The window is organized as Feature-Sliced Design. This skill says where each kind of code goes and how a slice is shaped. The structural checks refuse some wrong places; the rest is judgement, and the last section says how to make it. The rules are in `.agents/rules/architecture.md` §4 and §10 and in `.agents/rules/coding-style-frontend.md`; the checks are in `tools/lint-layers/`. Sibling files: [`examples.md`](examples.md) holds worked examples of the target shape, [`checks.md`](checks.md) lists each check by what it forbids.

## The layers

Five layers, lowest first. A file imports only from a lower layer. There is no `widgets` layer: a part that two pages use is a feature or an entity.

| Layer      | Holds                                                                                                                                                          | Slices |
| ---------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------ |
| `shared`   | What no business thing owns and at least two slices use: the generated clients bound once, the `Result` type, paths, text ranges, a DOM port, a base component | no     |
| `entities` | A business thing of the window and what is done to it alone: note, file, vault, settings, tab, plugin, command                                                 | yes    |
| `features` | One user scenario, whole: find and run a command, resolve a file conflict, link tabs, move a tab                                                               | yes    |
| `pages`    | One tab, whole: it puts features and entities on a screen and registers its tab kind                                                                           | yes    |
| `app`      | Composition and wiring: create the services, implement the plugin host's contracts, collect the commands of the features, order the start-up                   | no     |

`app` holds nothing a lower layer could hold. A function in `app` that does not create, connect or order slices belongs to a slice: a stateful scenario to a feature, a thing with its own type and rules to an entity, a pure helper to the `lib` of the slice that uses it.

`src/main.ts` is the entry: it mounts `app/` and may import any layer. `src/testing/` holds test helpers and is imported by tests only.

## Slices and segments

A slice is a folder `src/<layer>/<slice>/` named after what it is, in `kebab-case`. It has a public file `index.ts`. Another layer imports the slice through `@/<layer>/<slice>` and only through that file. Inside the slice, imports are relative. A slice imports no sibling slice of its own layer and there is no `@x` cross-import; what two slices share goes to a lower layer.

A slice is split into segments. A segment imports the segments after it in this list and none before it: `ui`, `model`, `api`, `lib`.

| Segment  | Holds                                                                                                                                                                                                           |
| -------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `ui`     | Vue components, their stories and tests, and the `*.types.ts` of a component with more than six props. A component receives props, emits events and holds no logic.                                             |
| `model`  | State and behavior: composables (`use<Feature>`), factories (`create<Thing>`), the ports the slice needs declared as interfaces, the orchestration of a scenario.                                               |
| `api`    | The calls to the core. A file calls a client of `@numen/core-client`, maps the answer field by field into the slice's own type, and turns an error into a `Result`. The `model` of the same slice imports it. |
| `lib`    | Pure TypeScript: types, mappers, rules, formatters. No Vue, no DOM, no clock, no timers.                                                                                                                        |
| `config` | Constants and static tables of the slice.                                                                                                                                                                       |

A `.vue` file lives in `ui/` alone. The words a slice shows are in `words.ts` at the root of the slice. A slice that holds only types and pure functions has `lib/` and `index.ts` and no other segment.

A model never imports the wire. The `model` of a slice imports its own `api` segment directly (`../api`); `api` maps the wire and returns the slice's own types. `provide` and `inject` carry no RPC call: a call to the core is made by the model of the slice that owns it, never reached through the component tree. A component takes props and emits events, and holds no client, no `api` and no injected service. A test of the model runs against a real in-memory transport (`@numen/core-client/testing`), and a spy is only on the outer edge (the network, the filesystem, the navigator).

## I am adding

| I am adding                                                          | It goes to                                                                                                                                                                                                                 |
| -------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| A call to the core                                                   | `api/` of the slice that owns the type the answer becomes. The clients are bound once in `shared/api/clients.ts`; the error text helpers of `@numen/core-client` are reached through `shared/api/errors.ts`.               |
| A piece of state of one scenario                                     | `model/` of the feature, or of the entity when the state is the thing's own (the open notes, the open tabs)                                                                                                                |
| A pure rule, mapper or formatter                                     | `lib/` of the slice that uses it                                                                                                                                                                                           |
| A screen that is one tab                                             | `pages/<tab>/ui/`, with its state in `model/` and the registration of its tab kind in `model/kind.ts`                                                                                                                      |
| A part of a screen that has state or is drawn on two pages           | A feature (a scenario) or an entity (a thing), imported by the pages                                                                                                                                                       |
| A command (palette, key, menu)                                       | The feature that owns it: `features/<scenario>/model/commands.ts` declares the identifier, the words, the availability and the handler; `entities/command` holds only the `Command` type. `app` collects the commands of the features and gives them to the palette feature. |
| A command that a plugin declares                                     | Nothing in the window: the plugin's manifest declares it, and `app/plugins/` passes the declared entries to the palette                                                                                                    |
| A plugin slot entry or a view a plugin draws                         | Nothing in the window: the plugin registers it and the slot row of `@numen/workbench` draws it. `app/plugins/` binds the slot to the shell.                                                                                    |
| The order of the entries of a slot, or a move between slots          | The core keeps the layout in `numen.json` under `app.layout.slots`, and the shared slot row takes the gesture. The window adds no layout state.                                                                            |
| A tab kind of a capability (a viewer, an editor of a file type)      | The plugin's frontend registers it. A tab kind the window owns (welcome, settings) is a page.                                                                                                                              |
| A type used by two slices                                            | A lower layer; see "Types used by two slices"                                                                                                                                                                              |
| A view-model (the shape a component draws)                           | `lib/` of the slice: the type and the pure function that builds it from the entity. The component takes it as props.                                                                                                       |
| A composable                                                         | `model/<useName>.ts` of the slice: one responsibility, at most eight returned properties, the file named as the composable                                                                                                 |
| A factory of a service                                               | `model/create<Thing>.ts`; its ports are parameters                                                                                                                                                                         |
| A platform capability (clock, scheduler, viewport, storage)          | A port: a parameter of the function that needs it, with the browser default supplied by `app`                                                                                                                              |
| A text the user reads                                                | `words.ts` of the slice that shows it                                                                                                                                                                                      |
| A component                                                          | `ui/<Name>/<Name>.vue` of the slice, with `<Name>.stories.ts` and `<Name>.test.ts` beside it                                                                                                                               |
| A story                                                              | Next to its component. The story of a whole page is in `pages/<tab>/ui/`; the story of the composed shell is in `app/`.                                                                                                |
| A test                                                               | Next to the file it tests, `<name>.test.ts`. Helpers shared by tests are in `src/testing/`.                                                                                                                                |
| A generic component with no domain word                              | `libs/ui/components`, not the window                                                                                                                                                                                             |
| Window chrome (title bar, palette, workspace, notices, welcome page) | `libs/ui/workbench`, not the window; the window supplies data and handles events                                                                                                                                               |

## Target layout of the editor

```text
apps/desktop/editor/src/
├── main.ts                       entry: creates the window with app/ and mounts it
├── app/                          composition and wiring only
│   ├── App.vue                   the root: places the shell, the pages and the notices
│   ├── window/                   createWindow(): creates the services (each model imports its own api), returns the few objects App.vue reads
│   ├── plugins/                  the window's LoaderHost: binds slots, tab kinds, commands, vaults, agent and buffers of the plugin host to slice services
│   └── bootstrap/                the start-up order and the screen shown when it fails
├── pages/
│   ├── welcome/                  the screen of a window with no tab: its rows and actions
│   └── settings/                 the settings tab: ui/, model/ (kind.ts, drafts), lib/
├── features/
│   ├── command-palette/          find and run a command: ui/, model/ (palette, search, handlers), lib/
│   ├── file-conflict/            keep or take a file that changed on disk while it was open
│   ├── tab-linking/              link tabs across windows: ui/, model/, api/ (workspace service), lib/
│   ├── tab-move/                 send a tab to another window: model/, api/, lib/
│   ├── window-display/           interface scale, text scale, theme and mode of the window
│   └── hotkeys/                  bind keys to commands and keep the binding in sync with the settings
├── entities/
│   ├── note/                     a note: api/ (note service), model/ (open notes, buffer), lib/ (types, mappers)
│   ├── file/                     a file of the vault: api/ (file service), lib/
│   ├── vault/                    the vaults of the installation and the one open: api/, model/, lib/
│   ├── settings/                 numen.json as the window reads it: api/ (settings and theme services), model/, lib/
│   ├── tab/                      the open tabs: api/ (saved tabs), model/ (useTabs, useTabGroups, useTabPanes, useRestoredTabs), ui/ (the panes of an unavailable tab), lib/
│   ├── plugin/                   the plugin catalog and what plugins declare: api/, model/
│   └── command/                  the Command type and the command target that features declare and the palette runs: lib/
└── shared/
    ├── api/                      clients.ts: the generated clients bound once; errors.ts: the error text helpers
    ├── lib/                      Result, paths, text ranges, a DOM port: what two or more slices use
    └── ui/                       a base component that two or more slices draw
```

A folder in this tree that does not exist yet is created when its first code arrives, together with its first consumer. A file in `shared/` that one slice uses moves into that slice.

## Target layout of flashcards

The flashcards window is the plugin host plus a vault chooser: the plugins of its target draw the cards, so the window holds no card logic.

```text
apps/desktop/flashcards/src/
├── main.ts                       entry: mounts app/
├── app/
│   ├── App.vue                   the root: the chooser, or the plugin view for the chosen vault
│   ├── window/                   createWindow(): creates the notices, the tabs and the services in the order they depend on each other, then the host
│   ├── plugins/                  the window's LoaderHost for the target desktop.flashcards: binds the tabs, slots, vaults, commands (none) and agent to the plugins
│   └── app.css                   the entry style sheet
├── pages/
│   └── vault-chooser/            the screen with the vaults to choose from: ui/ (VaultChooser.vue, MountedView.vue: renders a view a plugin declares into an element), lib/ (rows, the key that opens a vault)
├── features/
│   ├── app-start/                start the plugins the core lists and tell when the window can be drawn: model/               choose a theme or mode from the palette and apply it: api/ (theme service), model/, lib/
│   ├── window-tasks/             show the background tasks of the core as notices: api/ (window service stream), model/
│   └── external-links/           open an address that leads outside the window in the system browser: model/
├── entities/
│   ├── vault/                    the vaults of the installation: api/ (vaults service), lib/ (VaultRow)
│   ├── tab/                      the tab kinds the plugins register and the one tab the window shows, a plugin view opened on a vault: model/ (open and close), lib/
│   ├── plugin/                   the installed plugins and the slots their manifests declare: api/ (plugin catalog), model/
│   └── agent/                    whether the agent is reachable: api/ (agent service), model/
└── shared/
    └── api/                      clients.ts: the generated clients bound once; errors.ts: the error text helpers
```

`words.ts` and the version string belong to the slice that shows them; a string that two slices show is in `shared/config/`.

## The line between window, plugin and core

The core is the hexagon: it holds every rule about notes, the vault, the index, search, text versions, settings and the plugin host, behind use cases. A window is one of its driving adapters, as the CLI and the MCP server are. A plugin holds one capability whole: its logic, its storage under `.numen/<plugin id>/`, its services, its frontend and its commands.

The window holds:

- Composition: which plugins its target loads, which slots and tab kinds it offers them, how it implements the plugin host's contracts.
- Routing: a call from a screen to a client of `@numen/core-client`, with the answer mapped into the window's own type and an error shown as text through `formatErrorMessage` or `formatErrorCodeMessage`.
- Window state: the open tabs, the query and step of the palette, hotkeys, notices, focus, the draft of a form before it is saved.
- A small piece of window-specific logic, such as the place the flashcards window opens its tab.

The window hands to the core every rule that a second client would need: reading or writing a note or a file, finding a link, searching, scheduling, validating a setting, resolving a conflict by content, permissions. It hands to a plugin every capability a person could turn off: a reader, a graph, a media player, proofreading, a chat with an agent, flashcards study. A window calls a plugin's service through the core and imports nothing from `plugins/`; a plugin never calls the window.

The test for logic that grows in a window: would the CLI, the phone or an MCP client need the same answer? If yes, it is a use case in the core. Would a person turn it off with its plugin? If yes, it is in the plugin.

## Types used by two slices

A type that two slices use lives in the lowest layer that makes sense:

- A business thing (note, vault, tab, command, plugin) used by slices of one layer belongs to `entities/<thing>/lib`, exported by its `index.ts`.
- A type that names no business thing (a path rename, a text range, a result) belongs to `shared/lib`.
- A type that a feature declares and a page reads stays in the feature, because a page may import a feature.
- A type that only one slice reads stays in that slice, however general it looks.

## Other windows

`apps/mobile` is one Capacitor window over the same core client. Its code takes the same layers, segments and rules as the editor: a new call to the core is in an `api/` segment, a screen is a page, a scenario is a feature. `apps/landing` and `apps/docs` are Astro sites with components, layouts and content; they hold no vault domain and no core call. The rules that apply to them are the design tokens, the naming, the size limits and the plain English of the text.

## Size and role

A structure is judged by what one object does, not by the interfaces it is cut into.

- A source file has at most 500 lines. A file that holds a second responsibility is split into a slice, a segment or a module named after the thing it holds.
- A window holds composition, routing, window state and small window-specific logic. A scenario another client would need is a use case in the core; a capability a person could turn off is a plugin.
- Shared code is an entity, a `shared/lib` module named for what it computes, or a library chosen by the decision list; never a folder collected for reuse.
- A slice is imported through its `index.ts` only.
- A value of unknown type is parsed once, at the `api` boundary, into the window's own type. Screens, models and libs receive the named type.
- A `catch` shows the failure as text or returns a `Result` with the error; it never discards it.
- One host concern has one port: the clock, the viewport, the storage and the window API each enter through a single parameter or composable.
- Production code is not written for a test double. The test mounts the real component or model and replaces only the transport at the outer edge.

## What the checks enforce

Each structural check and lint rule is listed by what it forbids in [`checks.md`](checks.md). In short, they forbid an import of a higher layer, an import between slices of one layer, an import of a slice other than through its `index.ts`, a segment importing a segment before it in the list above, a `@numen/core-client` import outside an `api` segment, a component outside `ui`, a file in `shared` that one slice uses, a composable that returns more than eight properties, a `.vue` file that calls a client, a plugin imported by a window, a wire type in `libs/ui/components`, a color literal in a style, and the machine clock outside a port. An RPC call carried by `provide`/`inject`, and a command declared outside the owning feature's `model/commands.ts`, are refused in review.

A red check is fixed in the code, not by changing the check or moving code out of its sight.

## What no check decides

A check sees imports and syntax. These calls are made by the person writing and the person reviewing, and each has a test to apply.

- Entity or feature. An entity is a thing with a name in the product that exists when no one acts on it, with its state and rules; a feature is something a person does, with a start and an end. "The open notes" is an entity; "resolve the conflict of an open note" is a feature that reads the entity. When the code is mostly verbs it is a feature.
- Feature or page. A page is a tab a person opens and closes as a whole. A part of a screen with its own state and scenario, or one drawn on two pages, is a feature.
- Does a type go to a lower layer. When two slices of one layer need the same type, it moves to the layer below: to the entity that owns the concept when there is one, to `shared/lib` otherwise. One slice does not import the other and the type is not copied.
- Window, plugin or core. Apply the test in "The line between window, plugin and core" before adding logic to a window.
- Model or lib. Code that creates a ref, watches, schedules or holds state is model. A function of its arguments is lib.
- Is a split needed. A slice or a file is split when it holds a second responsibility: a part with state of its own, a part drawn alone elsewhere, a concern exercised separately. A folder is named after the thing it holds, never after a kind (`utils`, `helpers`, `types`).
- Is it app. If a lower layer could hold it, it is not app. App creates, connects and orders.
- A new mechanism. It lands with its first consumer, reusing the existing service, route and host call; a mechanism without a consumer stays on its branch.

## Style the placed code follows

- Functions are imperative (`getTab`, `findGroup`, `createTabLinks`, `isEligible`); handlers are `on<Action>`; composables are `use<Feature>`; factories are `create<Thing>`; booleans are `is`, `has`, `can`. Types are singular nouns.
- A fallible operation returns `Result`, and its failure key is `error`.
- Lengths: a local custom property in `rem` or `em` (`--row: 1.5rem`) may name a dimension that belongs to one component. No `px` for a size, with one exception, a 1px hairline border. No bare length on a property (`padding: 0.375rem`): a property takes a token, a local custom property or `0`.
- A comment states a fact the code cannot show. Prose is plain technical English.
- A test checks behavior through the real path and fails when the behavior breaks. None reads CSS text or measures a look. A scenario uses the words of the product from `words.ts`.
- A moved file is moved. Nothing keeps the old path or the old layout working.
