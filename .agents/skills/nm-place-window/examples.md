# Worked examples

Each example shows the target shape and cites code in `apps/desktop` or `libs/ui/workbench` that already has it. Paths are relative to the repository root. A difference between the cited file and the target is stated, and the example is the target.

## A call to the core is an `api` file that maps into the slice's type

Adding a query about the appearance of the window. The call, the mapping and the port live in the `api` segment of the entity that owns the result.

`apps/desktop/editor/src/entities/settings/api/theme.ts`:

- `Themes` (line 36) is the port: three methods in the window's own words (`getAppearance`, `readTheme`, `watchThemes`).
- `themes` (line 48) implements it. Each answer of the theme service is mapped field by field into `Appearance`, a type the slice declares in `lib/appearance.ts`; no wire type leaves the file.
- `ships` (line 83) and `WORDED` (line 95) are `Record`s keyed by the schema's enum codes, so a code added to the schema fails to compile until this file answers it.
- The slice's `index.ts` exports `themes` and the types. A page or a feature imports them from `@/entities/settings`.

The shape:

```ts
// entities/<thing>/api/<query>.ts
import { someService } from '@/shared/api/clients'
import type { Thing } from '../lib/thing'

export interface Things {
  getThings(): Promise<readonly Thing[]>
}

export const things: Things = {
  getThings: async () => (await someService.listThings({})).things.map(createThing),
}
```

The cited file imports the clients from `@/shared/clients`; the target path is `@/shared/api/clients`.

## A model receives its port and a feature is a folder of three segments

Adding a scenario with state that writes to the core: linking tabs across windows.

`apps/desktop/editor/src/features/tab-linking/`:

- `model/useTabLinks.ts:30` is `useTabLinks(deps)`. `TabLinksDeps` (line 18) holds the port `write` and the callback `onError`; the model imports no client. It returns eight properties (line 64), the most a composable may return.
- `lib/eligibility.ts` and `lib/windowName.ts` are pure functions with their tests beside them.
- `index.ts` is the public file: the types, `useTabLinks`, the two pure functions and the component.
- In the target the implementation of `write` is `api/tab-links.ts` in the same slice, exported from `index.ts`, and `app/window/` passes it to `useTabLinks`. Today `app/vault/tabLinks.ts` holds that implementation; the target is the slice's `api/`.

The shape:

```ts
// features/<scenario>/model/useScenario.ts
export interface ScenarioDeps {
  readonly write: (change: Change) => Promise<void>
  readonly onError: (error: unknown) => void
}

export function useScenario(deps: ScenarioDeps): ScenarioState {
  /* refs, handlers */
}
```

## A scenario against ports, with no Vue and no client

`apps/desktop/editor/src/features/file-conflict/model/conflicts.ts`:

- `Notes` (line 11) and `ConflictRaiser` (line 20) are the ports the function needs, declared next to it.
- `raiseConflicts(notes, raiser)` (line 25) watches the notes the window holds and raises one conflict for each note whose file changed on disk. It returns the function that stops watching.
- The test passes real implementations of both ports and checks what the raiser received.

A scenario like this is a feature because it is something a person resolves, and it reads the note entity through a port instead of importing a sibling slice.

## Wiring is `app`: hand a slice's state to a host contract

`apps/desktop/editor/src/app/window/createPluginServices.ts` builds the services a plugin is given from slices of the window: it passes `createWindowVaults` of `apps/desktop/editor/src/features/plugin-host/model/window/createWindowVaults.ts` the vault list and the function that opens a vault, and gets the plugin host's `VaultsHost` back. `createWindowVaults(deps)` holds no rule of its own; it connects two lower things to a contract of `@numen/plugin-loader`. The `app` file holds only the choice of what is connected to what.

`createWindowVaults.test.ts` beside it calls the function with a real reactive vault list and checks the state a plugin reads and the vault it opens; a spy is only on `openVault`, the edge.

The vault list comes from the `entities/vault` slice.

## The shape of a slice with a public file and few segments

`libs/ui/workbench/src/features/notices/` is a feature with `lib/` (types, a tally) and `model/` (the queue and its test) and an `index.ts` that exports the creator functions and the types a window needs. It has no `ui/` because it draws nothing, and no `api/` because it calls nothing. A window slice has only the segments it needs.

## What a plugin-facing window part looks like

`apps/desktop/flashcards/src/app/` today holds `host.ts` (the plugin host for the target `desktop.flashcards`), `windowVaults.ts`, `windowCommands.ts` (a `CommandsHost` that lists no command) and `agent.ts`. Each implements one contract of `@numen/plugin-loader` for this window, and none holds a card rule: study, scheduling and card drawing are in `plugins/views-flashcards-studio` and the core. In the target, files that hold a scenario or a thing (the appearance commands, the task stream, the vault list) leave `app/` for their slices, and what remains in `app/plugins/` is the contracts.
