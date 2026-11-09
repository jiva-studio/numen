# Specification: Reorder and move slot entries by Cmd/Ctrl+drag

**Task Slug:** `1803-slot-layout`
**Date:** `2026-11-09`
**Intent:** `.agents/tasks/1803-slot-layout/intent.md`
**Issue:** jiva-studio/numen#1803. Depends on #1802 (fixed 32px bars, shared chip with a popover that turns upward).

The layout of the slots is a section of the owner `app` in `numen.json`. It applies to every slot `slots.json` declares; today only the title bar and status bar slots are drawn with the SDK attributes, so only they can be dragged. The window reads the layout through the existing settings store, the plugin host applies it when it resolves a slot, and the window runs the drag over the elements that carry the SDK attributes. No protobuf message or RPC changes: the layout travels as JSON in the existing `Setting` of `GetSettings`, `WriteSettings` and `WatchSettings`.

## 1. Target Architecture & Interfaces

### 1.1 Storage in `numen.json` (pending the user's confirmation of the path, see §5 Q1)

```json
{
  "v": 1,
  "app": {
    "layout": {
      "slots": {
        "titleBar.status": ["titleBar.status|md.numen.views.tasks/status", "titleBar.status|md.numen.tools.index.stats/stats"],
        "statusBar.right": ["titleBar.status|md.numen.tools.updater/update"]
      }
    }
  }
}
```

- `app.layout.slots` maps a slot name to a list of entry keys. The example is the tasks and stats chips kept in `titleBar.status` and the updater chip moved into `statusBar.right`.
- An entry key is `<home slot>|<plugin id>/<id>`: the slot of the manifest entry that places the entry, a `|`, the plugin that places it, a `/`, then the view id or the command id exactly as the manifest entry writes it (`chat.open`, `app:note.create`). The application's own entries use the owner `app` (`titleBar.commands|app/app:settings.open`). The home slot is fixed by the manifest and does not change when the user moves the entry, so a key names one manifest entry. A submenu has no key and cannot be moved.
- `|` is the separator because no part of a key can contain it: slot names in `slots.json` and submenu ids are words joined by dots (`submenuIDPattern`), a plugin id matches `[a-zA-Z0-9_-]+` parts joined by dots (`manifestIDPattern`), a view id matches `tabKindPattern` (`[a-z0-9]` words joined by `.`, `:` or `-`), and a command id is a declared `<word>.<word>` or a full id `<owner>:<word>.<word>` (`fullCommandIDPattern`); none admits `|`. `libs/plugin-host-ts` has no id pattern of its own and reads these ids from the manifest.
- A key is unique without any rule: one manifest entry has one home slot, and the same command in two slots has two keys.
- Any slot name that `slots.json` declares is valid in the map. A name it does not declare is refused on write and ignored on read.
- The file is the only store. There is no default value; no `layout` key means the manifest order everywhere.

### 1.2 Wire

No change to any `.proto`.

| Operation | Existing call    | Shape used                                                                                                                                                       |
| :-------- | :--------------- | :--------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Read      | `GetSettings`    | `written` holds `app.layout.slots` like every other app section.                                                                                                 |
| Write     | `WriteSettings`  | One request per drop. One `Setting` per changed slot: `at = ["app", "layout", "slots", "<slot>"]`, `value` = JSON list of keys. Both settings of a move go in the one request, which is written whole or not at all. |
| Follow    | `WatchSettings`  | A change arrives as `at = ["app", "layout", "slots"]`, `value` = the whole map as JSON (or `null` when removed).                                                 |

### 1.3 Slot list (`libs/protocol/slots.json` and its generator)

- `slots.json` has no `isBar` field. `libs/protocol/tools/slots.mjs` emits no `BarSlots`, `IsBarSlot`, `BAR_SLOTS` or `BarSlotName`; the existing slot names and `IsSlot` / `SlotName` are enough.
- The `//` note of `slots.json` no longer mentions bars.
- Removed from the branch: `libs/plugin-host-ts/src/slots/barSlots.ts`, `libs/plugin-sdk-ts/src/slots/barSlots.test.ts` and `libs/plugin-catalog/manifest/slot_bar_test.go` (replaced by the tests of §1.6). Any name for this in the layout code says "slot" (`SlotName`, `listSlots`), not "bar".

### 1.4 Settings adapter (Go, `libs/core/adapter/settings`)

```go
// Layout is the user's arrangement of the slots. Slots maps a slot to the keys of its
// entries in drawing order; a key is <home slot>|<plugin id>/<view or command id>.
//
//numen:settings
type Layout struct {
	Slots map[string][]string `json:"slots"`
}
```

- `File`, `appObject` and `appSections` get `Layout Layout \`json:"layout"\``.
- `isOutsideDeclaration` also returns true for `layout`: the section is not part of `app.json`, is not shown on the settings page, and is not checked by the declared-setting check.
- `readApp` checks the section on read (`readLayout`). A slot name that `slots.json` does not declare, a value that is not a list of strings, and a key that does not split at its first `|` into a non-empty home slot and `<owner>/<id>`, with a valid owner (`app` or `manifest.IsValidID`) and a non-empty id, add a notice `numen.json: "app.layout.slots.<slot>": ...; the value is ignored` and are left out of the value in force. The file is not rewritten. Keys of plugins that are not installed are valid.
- `CheckWrite` checks a write under `["app", "layout", ...]` with the same rules (`checkLayoutWrite`): path `["app","layout","slots","<slot>"]` takes a list of valid keys with no key twice; `["app","layout","slots"]` takes an object of such lists; `["app","layout"]` takes an object with only `slots`. Anything else is `ErrUndeclared` or `ErrWrongType`, which the use case reports as `ErrNotASetting`; nothing is written.

### 1.5 Watch (Go, `libs/core/container/settings/settings.go`)

- `listSettingValues` adds `["app", "layout", "slots"]` with the stored map when the file holds one. `WatchSettings` then reports a change of the layout from a window or from a hand edit.

### 1.6 Manifest validation (Go, `libs/plugin-catalog/manifest/slot.go`)

`listSlotIssues` gets one check that applies to every slot: a view and a command of one plugin with the same id in the same slot are refused with `%w: view and command %q share one key in %q`. `listSlotIssues` has no check that depends on the slot kind. The functions `listBarIssues` and `barEntries.add` and the issues "placed in the bar slots" and "view and command share one key in the bar slots" are deleted (the same-key check above replaces the second). The check on this branch's base stays: the same command placed twice in one slot is refused with `%w: %s %q is placed twice in %q`.

A view or a command may be placed in several slots, and a command in menus and in any other slot; each placement has its own key (§1.1).

### 1.7 Plugin host (TypeScript, `libs/plugin-host-ts/src/slots`)

New file `slotLayout.ts` (pure functions, no Vue):

```ts
import type { SlotName } from '@numen/plugin-sdk'

/** The user's arrangement of the slots: per slot, the keys of its entries in drawing order. */
export type SlotLayout = Readonly<Partial<Record<SlotName, readonly string[]>>>

/** The lists of the slots a drop changes, each the whole new list of its slot. */
export type SlotLayoutChange = Readonly<Partial<Record<SlotName, readonly string[]>>>

/** One drop: the entry, the slot it is drawn in, the slot it is dropped in, and the key it is dropped before. */
export interface SlotMove {
  readonly key: string
  readonly from: SlotName
  readonly to: SlotName
  readonly before?: string | undefined
}

/** The key of an entry in the layout, `<home slot>|<plugin id>/<view or command id>`; undefined for a submenu. */
export function getLayoutKey(entry: DeclaredSlotEntry): string | undefined

/** The layout a settings value holds. A slot that `slots.json` does not declare and a list that is not a list of strings are left out. */
export function readSlotLayout(value: unknown): SlotLayout

/** The lists a drop writes; empty when the drop leaves every slot as it is drawn. */
export function moveSlotEntry(
  layout: SlotLayout,
  drawn: Readonly<Record<SlotName, readonly string[]>>,
  move: SlotMove,
): SlotLayoutChange
```

Rules of `moveSlotEntry` for each slot it writes (`from`, and `to` when it differs):

1. The new drawn order is the current drawn order of the slot with the key removed (`from`) or inserted before `before`, or at the end when `before` is undefined (`to`).
2. The new list holds every key of the new drawn order, so the slot becomes one flat user order.
3. A key of the stored list that is not drawn now (disabled or removed plugin, `when` false) stays in the list, right after the stored key that preceded it, or first when none did.
4. A slot whose new list equals its stored list is left out of the change.

Changes to `slotRegistry.ts`:

- `SlotRegistryDeps` gets `readonly getLayout?: () => SlotLayout | undefined`, read reactively.
- `createSlotRegistry` returns `SlotRegistry`:

```ts
/** The slots of the window with the lookups the drag of the slot entries needs. */
export interface SlotRegistry extends SlotHost {
  /** The layout key of a drawn item, by its `SlotItem.key`; undefined for a separator, a submenu or an unknown key. */
  findLayoutKey(itemKey: string): string | undefined
  /** The layout keys of the commands and views drawn in a slot, in drawing order. */
  listLayoutKeys(slot: SlotName): readonly string[]
}
```

- Resolving a slot `P`:
  1. A key belongs to the first slot, in `slots.json` order, whose list holds it. Keys are matched in every slot, menus included.
  2. The entries drawn in `P` are those whose key belongs to `P`, in the order of the first position of their key in the list of `P`, followed by the entries whose manifest slot is `P` and whose key belongs to no slot, sorted by the current `compareInSlot` (group, then order, then plugin id).
  3. `keepFirstOfEachCommand` still applies.
  4. A separator is drawn between two neighbours whose groups (`group ?? ''`) differ. Its key is `<slot>/separator/<n>`, `n` counting the separators of the slot from 0. With no layout this gives the same order and separators as today.
- A slot without a list resolves as today.

`hosting/types.ts` `PluginHostDeps` gets `readonly getSlotLayout?: () => SlotLayout | undefined`; `hosting/host.ts` passes it to the registry and returns `slots` typed as `SlotRegistry`. `index.ts` exports `SlotLayout`, `SlotLayoutChange`, `SlotMove`, `moveSlotEntry`, `readSlotLayout`.

### 1.8 Plugin SDK (TypeScript, `libs/plugin-sdk-ts/src/slots`)

- `SlotName` stays the generated union of the slot names of `slots.json` (§1.3); there is no bar subset.
- The attributes a plugin puts on the elements of a slot, so the window finds them (pending the user's approval, see §5 Q3):

```ts
// SLOT_ATTRIBUTE names the element that draws a slot; its value is the slot name. SLOT_ENTRY_ATTRIBUTE
// names the element of one command or view of the slot; its value is SlotItem.key.
export const SLOT_ATTRIBUTE = 'data-numen-slot'
export const SLOT_ENTRY_ATTRIBUTE = 'data-numen-slot-entry'
```

The Go SDK has no slot API; nothing is added there.

### 1.9 Slot drawing

Drop targets and draggable entries are the slots and entries that carry `SLOT_ATTRIBUTE` and `SLOT_ENTRY_ATTRIBUTE`. The shared slot row sets them, so today the title bar and status bar slots are drawn that way. Menus are not, so they have no drag until their drawing sets the attributes. No compatibility rule between slots exists: a drop moves an entry into any slot that carries the attribute.

- `plugins/views-title-bar/frontend/ui/TitleBarSlot.vue`: the slot element gets `SLOT_ATTRIBUTE` = `name`; each command button, command-with-items button and view wrapper gets `SLOT_ENTRY_ATTRIBUTE` = `entry.key`. Existing `data-title-bar-*` attributes stay.
- `plugins/views-status-bar/frontend/ui/SlotRow.vue`: the slot element gets `SLOT_ATTRIBUTE` = `place`; each command button and command-with-items button gets `SLOT_ENTRY_ATTRIBUTE`; a view is wrapped in an element with `SLOT_ENTRY_ATTRIBUTE` (views are drawn bare today).
- While the root element of the document has `data-numen-slot-dragging`, an empty slot element that carries the attribute keeps a minimum width of one chip (the #1802 chip height, 32px bar) so it can take a drop. No other drag code lives in the drawing plugins.

### 1.10 Window (TypeScript, `apps/desktop/editor`)

- `features/slot-layout/model/slotDrag.ts` — `createSlotDrag(deps)`, built on `usePressDrag` of `@numen/ui`:

```ts
export interface SlotDragDeps {
  readonly isMac: () => boolean
  readonly findLayoutKey: (itemKey: string) => string | undefined
  readonly listLayoutKeys: (slot: SlotName) => readonly string[]
  readonly getLayout: () => SlotLayout
  readonly writeLayout: (change: SlotLayoutChange) => Promise<void>
}
```

  - A `pointerdown` listener on the document in the capture phase. It acts only for the primary button with `metaKey` on macOS or `ctrlKey` elsewhere, on an element inside `[SLOT_ENTRY_ATTRIBUTE]` inside a `[SLOT_ATTRIBUTE]` whose item key has a layout key. Then it calls `preventDefault` and `stopPropagation` and starts the press. Every other press is not touched.
  - Past the `usePressDrag` threshold the drag starts and `data-numen-slot-dragging` is set on the document root. The drop target is the element with `SLOT_ATTRIBUTE` under the pointer and the entry before which the pointer stands, by the horizontal midpoints of the entry elements of that slot (hidden entries of the title bar fit excluded).
  - Drop on a target: `moveSlotEntry` with the current layout and `listLayoutKeys` of every slot that carries `SLOT_ATTRIBUTE`; a non-empty change is written with one `writeLayout` call. The `click` that follows a drag is stopped in the capture phase.
  - Escape, or a drop outside the elements with `SLOT_ATTRIBUTE`, ends the drag with no write.
  - A press that ends before the threshold leaves the following `click` to the entry.
- `features/slot-layout/ui/SlotDragLayer.vue` — draws `DragPreview` of `@numen/ui` with the title of the entry and an insertion marker at the drop position. Props only.
- `app/window/useWindow.ts` wiring:
  - `getSlotLayout` = `readSlotLayout(settings.rest.getSetting(['app', 'layout', 'slots']))` as a computed, passed into the plugin host deps (through `windowSlots.ts` / `useWindowPlugins.ts`).
  - `writeLayout(change)` = one `settings.rest.writeSettings` call with one edit per slot in `change` at `['app', 'layout', 'slots', slot]`. A failed write is reported by the store as other setting writes are, and the store's re-read draws the stored layout again.
  - `SlotDragLayer` is mounted once in the window root.
- Until the window has read the settings, slots draw the manifest order.

## 2. Blast Radius Matrix

| Package / Path                   | File                                                                 | Action          | Downstream Consumers                                                   |
| :------------------------------- | :------------------------------------------------------------------- | :-------------- | :--------------------------------------------------------------------- |
| `libs/protocol`                  | `slots.json` (drop `isBar`), `tools/slots.mjs`                       | Modify          | generated Go and TS slot lists                                         |
| `libs/plugin-catalog/manifest`   | `slots.go` (generated), `slot_bar_test.go`                           | Modify / Delete | `slot.go`, `libs/core/adapter/settings`                                |
| `libs/plugin-catalog/manifest`   | `slot.go`, `slot_test.go`                                            | Modify          | plugin install and load; **overlaps open PR #1795**                    |
| `libs/core/adapter/settings`     | `document.go`, `owner.go`, new `layout.go`                           | Modify / Create | `GetSettings.written`, `WriteSettings`, settings reference docs        |
| `libs/core/container/settings`   | `settings.go`                                                        | Modify          | `WatchSettings`                                                        |
| `libs/plugin-sdk-ts`             | `src/slots/slotNames.ts` (generated), `src/slots/slots.ts`, `src/slots/barSlots.test.ts`, `index` | Modify / Delete | title bar, status bar, plugin host; **#1795 also edits `slots.ts`**    |
| `libs/plugin-host-ts`            | `src/slots/slotRegistry.ts`, new `src/slots/slotLayout.ts`, `src/slots/barSlots.ts` (delete), `src/hosting/types.ts`, `src/hosting/host.ts`, `src/index.ts` | Modify / Create | editor window; **#1795 also edits `slotRegistry.ts` (item keys with values, `compareInSlot`)** |
| `plugins/views-title-bar`        | `frontend/ui/TitleBarSlot.vue`, `frontend/ui/titleBar.css`, `frontend/lib/frame.ts` and `index.ts` where they use the bar list | Modify | title bar stories and tests |
| `plugins/views-status-bar`       | `frontend/ui/SlotRow.vue`, `frontend/ui/statusBar.css`               | Modify          | status bar tests                                                       |
| `apps/desktop/editor`            | new `src/features/slot-layout/`, `src/app/window/useWindow.ts`, `src/app/window/windowSlots.ts`, `src/app/plugins/useWindowPlugins.ts` | Create / Modify | window root                                                            |
| `docs/plugins/README.md`         | slot section                                                         | Modify          | plugin authors; **#1795 also edits it**                                |
| user docs / settings reference   | `app.layout.slots`                                                   | Modify          | `apps/docs/tools/manual.mjs` reads `//numen:settings`                  |
| `tests/e2e/desktop`              | new `tests/shell/window/slot-layout.spec.ts`                         | Create          | e2e shards                                                             |

PR #1795 (feat/slot-entry-values) lets two entries of one command with different `values` be two items with keys `<plugin id>/command/<id>?<name>=<value>`. Its check that refuses a command twice in one slot stays, and the home slot in the key (§1.1) keeps two placements of one command apart. Whichever PR merges second rebases `slot.go`, `slotRegistry.ts`, `slots.ts` and `docs/plugins/README.md`.

## 3. Negative Invariants (Architectural Prohibitions)

- No new `.proto` message, field or RPC. The layout uses `Setting` as it is.
- No plugin id appears in `libs/core`, `apps/*` or `libs/ui` (§11 g). Keys are data from the file.
- Plugins that draw a slot do not import `@numen/plugin-host`, `@numen/core-client` or each other. They only mark elements with the SDK attributes.
- The plugin host does not read or write settings itself; the window passes `getSlotLayout` in and writes through its settings store.
- No module-level mutable state in the window; the drag state lives in the composable.
- A `.vue` file uses no client; `SlotDragLayer.vue` takes props only.
- No reset, hide or per-vault layout, no palette command and no settings page row.
- No `isBar` field, no bar subset of the slots, no compatibility rule between slots.
- Menus (`fileTree.context`, `tabStrip.context`, `plexGraph.context`, `palette.searchResult`, submenus) are not drawn with the SDK attributes in this task and have no drag.
- No backward compatibility: no migration of an older layout and no reader of another shape.
- No test reads CSS text.

## 4. Acceptance Criteria

### AC-1: A listed entry is drawn in the listed slot, in listed order

- **Given**: tasks `status` and index stats `stats` both in `titleBar.status` by manifest, and a layout `titleBar.status: ["titleBar.status|md.numen.views.tasks/status", "titleBar.status|md.numen.tools.index.stats/stats"]`
- **When**: the registry resolves `titleBar.status`
- **Then**: tasks comes before stats.

### AC-2: A listed entry leaves its manifest slot

- **Given**: the updater view in `titleBar.status` by manifest, and a layout `statusBar.right: ["titleBar.status|md.numen.tools.updater/update"]`
- **When**: the registry resolves `titleBar.status` and `statusBar.right`
- **Then**: the updater is drawn in `statusBar.right` only.

### AC-3: Unlisted entries follow the listed ones

- **Given**: a layout that lists one of three entries of `titleBar.commands`
- **When**: the slot is resolved
- **Then**: the listed entry is first, and the other two follow in manifest order (group, order, plugin id).

### AC-4: No layout keeps today's order

- **Given**: no `app.layout`
- **When**: each slot is resolved
- **Then**: the items and separators are in the same order as before this change.

### AC-5: A separator stands where neighbour groups differ

- **Given**: a listed `titleBar.commands` order whose groups are `navigate, new, navigate`
- **When**: the slot is resolved
- **Then**: a separator stands between each pair of neighbours with different groups, none first, none last, none adjacent, and all separator keys are distinct.

### AC-6: Keys that draw nothing are skipped

- **Given**: a list with a key of a disabled plugin, a key of a plugin that is not installed, and a key whose entry has a `when` that is false
- **When**: the slot is resolved
- **Then**: the other entries are drawn in their listed order and nothing is drawn for those keys.

### AC-7: A key listed twice is drawn once

- **Given**: one key in the lists of `titleBar.status` and `statusBar.right`, and another key twice in one list
- **When**: the slots are resolved
- **Then**: the first is drawn in `titleBar.status` only (first in `slots.json` order), and the second once, at its first position.

### AC-8: A menu slot takes the layout

- **Given**: a command placed by manifest in `fileTree.context` and in `titleBar.commands`, and a layout that lists the key of the `fileTree.context` placement in `statusBar.left`
- **When**: `fileTree.context`, `statusBar.left` and `titleBar.commands` are resolved
- **Then**: the command is drawn in `statusBar.left` (for the `fileTree.context` placement) and in `titleBar.commands`, and not in `fileTree.context`. With no layout, `fileTree.context` resolves as before.

### AC-9: A reorder writes one slot

- **Given**: `titleBar.status` drawn as `[stats, tasks]` with no layout
- **When**: `moveSlotEntry` moves tasks before stats in the same slot
- **Then**: the change holds only `titleBar.status: [tasks, stats]`.

### AC-10: A move writes both slots

- **Given**: the updater drawn in `titleBar.status` and `statusBar.right` drawn empty
- **When**: `moveSlotEntry` moves the updater to `statusBar.right`
- **Then**: the change holds `titleBar.status` without the updater and `statusBar.right: [updater]`.

### AC-11: A move keeps keys that are not drawn

- **Given**: a stored list `[a, x, b]` where `x` belongs to a disabled plugin, drawn as `[a, b]`
- **When**: `b` is moved before `a`
- **Then**: the new list is `[b, a, x]`.

### AC-12: A drop at the same place writes nothing

- **Given**: any drawn layout
- **When**: an entry is dropped before the key that already follows it, or at the end of its own slot when it is last
- **Then**: the change is empty and the window makes no write.

### AC-13: The layout is written, read back and reported

- **Given**: a settings file without `app.layout`
- **When**: one `WriteSettings` holds `["app","layout","slots","titleBar.status"]` and `["app","layout","slots","statusBar.right"]` with lists of keys
- **Then**: both lists are in the file, `GetSettings.written` holds them under `app.layout.slots`, and one `WatchSettings` response reports `["app","layout","slots"]` with the whole map.

### AC-14: A wrong layout write is refused

- **Given**: a settings file
- **When**: a write puts a list under `app.layout.slots.fileTree.context`, a string instead of a list, a list with a number, a key with no `/`, or a key twice in one list
- **Then**: the write fails with `ErrNotASetting` and the file is unchanged.

### AC-15: A wrong layout in the file is ignored with a notice

- **Given**: a hand-written file whose `app.layout.slots` holds a slot name that `slots.json` does not declare and one valid slot list
- **When**: the settings are read
- **Then**: a notice names the undeclared slot, the valid list is in force, and the file is not rewritten.

### AC-16: A hand edit reaches the window

- **Given**: a running core with a subscriber of `WatchSettings`
- **When**: `app.layout.slots` is changed in the file by hand
- **Then**: the subscriber receives `["app","layout","slots"]` with the new map.

### AC-17: The same command twice in one slot is refused

- **Given**: a manifest that places one command twice in `titleBar.commands`
- **When**: the manifest is validated
- **Then**: it is refused with `ErrInvalidSlotEntry` naming the slot.

### AC-18: One entry may stand in several slots

- **Given**: a manifest that places view `status` in `titleBar.status` and in `statusBar.right`, and one that places a command in `titleBar.commands` and `statusBar.left`
- **When**: each manifest is validated
- **Then**: it is accepted, and the two placements have the keys `titleBar.status|<plugin id>/status` and `statusBar.right|<plugin id>/status`.

### AC-19: A command may stand in a menu and in another slot

- **Given**: a manifest that places one command in `titleBar.commands` and in `fileTree.context`
- **When**: the manifest is validated
- **Then**: it is accepted.

### AC-28: A view and a command with one id in one slot are refused

- **Given**: a manifest that places view `open.it` and command `open.it` in `titleBar.commands`; and one that places them in two different slots; and two plugins that each place an entry with the id `open.it` in one slot
- **When**: each manifest is validated
- **Then**: the first is refused with `ErrInvalidSlotEntry`, because both entries would have the layout key `titleBar.commands|<plugin id>/open.it`; the other two are accepted.

### AC-20: The slot row marks its slots and entries

- **Given**: the title bar and the status bar, whose slots are drawn by the shared slot row, with commands, a command with items and a view
- **When**: they are rendered
- **Then**: each slot element carries `data-numen-slot` with its slot name, and each command and view element carries `data-numen-slot-entry` with its item key.

### AC-21: Cmd/Ctrl+drag moves an entry and writes once

- **Given**: a window on Linux with two entries in `titleBar.status`
- **When**: the user presses Ctrl on the second entry, moves past the threshold, and releases over the first entry's left half
- **Then**: one `writeSettings` call is made with one edit at `["app","layout","slots","titleBar.status"]`, and the slot draws the new order. With `metaKey` instead of `ctrlKey` on Linux, no drag starts; on macOS `metaKey` starts it.

### AC-22: A plain press is left alone

- **Given**: an entry of a slot that carries the attributes
- **When**: the user presses it without the modifier, or with the modifier and releases before the threshold
- **Then**: the host does not prevent the event, no drag starts, and the click reaches the entry, which runs its command.

### AC-23: The click after a drag does nothing

- **Given**: a Cmd/Ctrl+drag that ends with a drop
- **When**: the browser sends the `click` that follows
- **Then**: the entry's command does not run.

### AC-24: A drag can be cancelled

- **Given**: a running drag
- **When**: the user presses Escape, or releases outside the elements with `data-numen-slot`
- **Then**: no write is made and the slots are as before.

### AC-25: An empty slot takes a drop

- **Given**: `statusBar.left` with no entries
- **When**: the user drags an entry over it and releases
- **Then**: the entry is drawn in `statusBar.left` and one write holds both lists.

### AC-26: A failed write restores the slots

- **Given**: a write of the layout that fails
- **When**: the drop ends
- **Then**: the window reports the failure as other setting writes do, and the slots draw the layout the file holds.

### AC-27 (e2e): Reorder, move, restart

- **Given**: a desktop window with the updater showing a new release (stand-in), index stats and tasks in the title bar, no `app.layout` in `numen.json`
- **When**: the user Ctrl+drags (Cmd+drag on macOS) the tasks chip before the index stats chip, then Ctrl+drags the updater chip into the right end of the status bar, then the application restarts
- **Then**: after the restart the title bar shows tasks before stats, the status bar's right end shows the updater, the title bar does not, `numen.json` holds `app.layout.slots` with both lists, and clicking the updater chip opens its popover above the chip (the popover's bottom edge is at or above the chip's top edge).

## 5. Decisions (2026-11-09)

1. **Place in the file:** `app.layout.slots`, as proposed; the first level of `numen.json` takes only `v`, `app` and plugin ids.
2. **Wire:** no protobuf change; the layout goes through `WriteSettings`, `GetSettings` and `WatchSettings`.
3. **Attributes:** `data-numen-slot` and `data-numen-slot-entry` are exported from the TS SDK as proposed. They are DOM attributes of the TS host and have no Go counterpart.
4. **Cmd/Ctrl press without movement:** accepted as proposed (a plain button runs, a menu button that opens on press does not).
5. **Unlisted entry with a false `when` at drop time:** not added to the list, as proposed.
6. **Key in two lists (hand-edited file):** drawn in the first slot in `slots.json` order, both copies kept in the file, as proposed.
7. **Two plugins placing the same full command in one slot:** the first is drawn, the second is not movable, as proposed.
8. **Decision 2026-11-09: no `isBar`, slots are generic; the key includes the home slot.**
9. **Decision 2026-11-09: same view/command id twice in one slot of one plugin is refused, so the key stays unique; no kind marker in the key.**

PR #1795 (`feat/slot-entry-values`) changes `slot.go`, `slotRegistry.ts`, the SDK `slots.ts` and `docs/plugins/README.md`; the branch that merges second rebases onto the first.
