# Specification: One fixed bar height and one shared chip

**Task Slug:** `1802-bar-chip`
**Date:** `2026-11-09`
**Intent:** `.agents/tasks/1802-bar-chip/intent.md`
**Issue:** jiva-studio/numen#1802

## 0. Current state

| Part | Today |
| :--- | :---- |
| Title bar height | `32px` literal in `plugins/views-title-bar/frontend/ui/titleBar.css`; `--numen-title-bar-height: 32px` in `libs/ui/shell/src/features/window-frame/tokens/frame.css`; `TITLE_BAR_HEIGHT = 32` in `libs/ui/shell/src/features/window-frame/lib/sizes.ts`; `frame.BarHeight = 32` in `apps/desktop/internal/frame/frame.go`. |
| Status bar height | `1.375rem` in `plugins/views-status-bar/frontend/ui/statusBar.css`; font size `var(--numen-text-1)` (`0.625rem`), entry padding `0 0.5rem`, entry gap `0.25rem`, separator `0.5rem`. All follow the interface scale. |
| Gap between chips | Each plugin wraps its chip in an element with `padding-inline: calc(var(--numen-inset) / 4)` (`2px` each side at scale 1, so `4px` between two chips). Follows the interface scale. |
| Chip | `--chip-height: 20px`, `--chip-text: 12px`, `--chip-pad: 8px`, `--chip-mark: 12px` copied into three stylesheets. Grounds: updater `--numen-accent` with `--numen-accent-ink`; index stats and tasks `--numen-surface`, hover `color-mix(in oklab, var(--numen-surface), var(--numen-ink) 10%)`; tasks with a failed task `--numen-alarm-bg` / `--numen-alarm`. |
| Popover | Each plugin keeps `isOpen`, calls `useMenuDismissal` from `@numen/ui`, and places the panel with `position: absolute; top: calc(100% + 4px); right: 0`. No upward placement. Index stats passes `isResize` so a resize does not close it. |
| Popover panel | Each plugin draws its own frame: `--numen-raised` ground, `1px solid var(--numen-panel-border)`, `--numen-radius-popup`, `--numen-panel-shadow`, `z-index: 50`, and its own width in rem. |
| reka-ui | `reka-ui ^2.10.3` is a dependency of `libs/ui/core` (2.11.0 installed). No component of `@numen/ui` uses its Popover yet. |

The three chips are slot views of `titleBar.status` (manifests of `tools-updater`, `tools-index-stats`, `views-tasks`). The title bar wraps every view in `.title-bar__view`; the status bar draws a view without a wrapper.

## 1. Target Architecture & Interfaces

### 1.1 Bar height

- One token holds the bar height in px: `--numen-title-bar-height: 32px`. The title bar and the status bar read it for `height` and `min-height`. No `rem` and no `--numen-interface-scale` in either bar's height.
- `frame.BarHeight` (Go) and `TITLE_BAR_HEIGHT` (shell) keep the same number, 32.
- Where the token is declared is open question Q1: a plugin frontend may import only `@numen/plugin-sdk` and `@numen/ui`, and the plugin Storybook (`libs/plugin-sdk-ts/stories/.storybook/preview.ts`) loads `@numen/ui/styles.css` and not the shell. The status bar reading a token that only the shell declares has no height in its own story. Proposed: move the declaration from `libs/ui/shell/.../tokens/frame.css` into `libs/ui/core/src/shared/tokens/tokens.css`, keep its name, and point the comments of `frame.go`, `sizes.ts` and `bar_height_test.go` at the new file. Decided: move (section 5).

### 1.2 Bar text and spacing in px

`plugins/views-status-bar/frontend/ui/statusBar.css` takes px values equal to today's values at scale 1:

| Property | Today | New |
| :------- | :---- | :-- |
| `.status-bar` `height`, `min-height` | `1.375rem` | `var(--numen-title-bar-height)` |
| `.status-bar` `font-size` | `var(--numen-text-1)` | `13px` |
| `.status-bar__entry` `padding` | `0 0.5rem` | `0 8px` |
| `.status-bar__entry` `gap` | `0.25rem` | `4px` |
| `.status-bar__gap` `inline-size` | `0.5rem` | `8px` |

`titleBar.css` already uses px; its `32px` height and `line-height` read `var(--numen-title-bar-height)`.

### 1.3 Gap between entries

- New token in `libs/ui/core/src/shared/tokens/tokens.css`: `--numen-bar-gap: 4px` (today's distance between two chips at scale 1).
- `.title-bar__slot` and `.status-bar__slot` set `gap: var(--numen-bar-gap)`. The gap is between adjacent entries of one slot; none before the first or after the last.
- The gap applies between adjacent view entries only; command buttons keep touching (decision Q2). The separator keeps its `8px`.

### 1.4 Chip component

Placement: `libs/ui/core/src/shared/ui/bar-chip/` (shared layer), re-exported from `libs/ui/core/src/index.ts`.

```text
libs/ui/core/src/shared/ui/bar-chip/
├── BarChip.vue
├── BarChip.stories.ts     # tester
├── BarChip.test.ts        # tester (jsdom: what is emitted and drawn, and the negatives)
├── index.ts
└── fixtures/              # invented chip faces and popover contents
```

Contract:

```ts
/** The ground of a chip. */
export type BarChipTone = 'neutral' | 'accent' | 'alarm'

interface BarChipProps {
  /** Accessible name of the trigger. */
  label: string
  /** Text of the native tooltip; none when absent. */
  tooltip?: string | undefined
  tone?: BarChipTone // default 'neutral'
  /** Accessible name of the popover dialog (Q5). */
  popoverLabel?: string | undefined
}

// v-model:open — whether the popover is open. Default false.
const isOpen = defineModel<boolean>('open', { default: false })

// Slots
// default  — the chip face (text, a mark). Required.
// popover  — the popover content. Optional; without it a press opens nothing.
```

`BarChipTone` is new (Q4): the three plugins draw three grounds today, and the chip owns its ground, so it takes the tone as a prop.

Behavior:

- The trigger is a `<button type="button" tabindex="-1">` with `aria-label` = `label`, `title` = `tooltip`, `aria-haspopup="dialog"` and `aria-expanded` = open state. `tabindex="-1"` matches today's three chips.
- Size in px, from tokens in `tokens.css`: `--numen-chip-height: 20px`, `--numen-chip-text: 12px`, `--numen-chip-inset: 8px` (inline padding), `--numen-chip-mark: 12px` (the size a plugin gives its mark), face gap `calc(var(--numen-chip-inset) / 2)`. `font-variant-numeric: tabular-nums`, `line-height: 1`, `white-space: nowrap`, `border-radius: var(--numen-radius)`, no border.
- Grounds: `neutral` = `--numen-surface` / `--numen-ink`, hover and open `color-mix(in oklab, var(--numen-surface), var(--numen-ink) 10%)`; `accent` = `--numen-accent` / `--numen-accent-ink`, hover and open `filter: brightness(1.08)`; `alarm` = `--numen-alarm-bg` / `--numen-alarm` in every state.
- Every child of the trigger has `pointer-events: none` (WebKit sends no click when the node under a press is replaced before the release; today in `tasksStatus.css`).
- Popover: reka-ui `PopoverRoot` (`v-model:open`), `PopoverTrigger as-child`, `PopoverPortal`, `PopoverContent` with `side="bottom"`, `align="end"`, `side-offset` = (bar height − chip height) / 2 + 4px, read from `--numen-title-bar-height` and `--numen-chip-height` (the popover starts 4px below the bar, or ends 4px above it after the flip), `avoid-collisions` (flips to the top when there is no room below). Escape and a press outside close it (reka-ui defaults). A resize does not close it.
- The chip draws the panel frame: `--numen-raised` ground, `--numen-ink` text, `1px solid var(--numen-panel-border)`, `--numen-radius-popup`, `--numen-panel-shadow`, `overflow: hidden`, stacked above both bars. The plugin's content sets its own width and inner layout in rem.
- `PopoverContent` carries `role="dialog"`; the plugin content drops its own `role="dialog"` and `aria-modal`. The dialog's `aria-label` is `popoverLabel` (e.g. "Index stats") — see Q5.
- No business logic in the view; the component has no pure module because it computes nothing.

### 1.5 Plugins on the chip

Each plugin keeps its state (`isOpen` ref bound with `v-model:open`) and its popover component, and closes the popover from its own actions by setting `isOpen` to false.

| Plugin | Face (default slot) | Tone | Popover slot |
| :----- | :------------------ | :--- | :----------- |
| `tools-updater` | `v{{ update.version }}` | `accent` | `UpdatePopover`; the chip is not drawn when `update` is null (as today). |
| `tools-index-stats` | `MARK` and the figure (`min-width: 7ch`, `text-align: end`) | `neutral` | `StatsPopover` only when `stats` is not null. |
| `views-tasks` | the ring (`--numen-chip-mark` square) and the summary (`min-width: 8ch`) | `alarm` when a task failed, else `neutral` | `TaskPopover`. `data-tasks-status` and the `--active` modifier stay on the plugin's root. |

Removed from the plugins: the wrappers `.update-indicator`, `.index-stats`, `.tasks-status` padding (`padding-inline: calc(var(--numen-inset) / 4)`), the `--chip-*` values and the trigger rules, `.update-popover`/`.index-popover`/`.tasks-status__popover-container` positioning (`position`, `top`, `right`, `z-index`) and frame, the `useMenuDismissal` calls and the `isResize` filter. `plugins/tools-updater/frontend/styles.test.ts` and `plugins/tools-index-stats/frontend/styles.test.ts` read CSS text and assert the removed positioning; they are deleted.

`useMenuDismissal` stays in `@numen/ui` (`Menu`, `SearchableListPopup` use it).

## 2. Blast Radius Matrix

| Package / Path | File | Action | Downstream Consumers |
| :------------- | :--- | :----- | :------------------- |
| `libs/ui/core` | `src/shared/ui/bar-chip/BarChip.vue`, `index.ts` | Create | three plugins below |
| `libs/ui/core` | `src/shared/ui/bar-chip/BarChip.stories.ts`, `BarChip.test.ts`, `fixtures/` | Create (tester) | — |
| `libs/ui/core` | `src/index.ts` | Modify: export `BarChip`, `BarChipTone` | every plugin frontend |
| `libs/ui/core` | `src/shared/tokens/tokens.css` | Modify: `--numen-chip-*`, `--numen-bar-gap`; `--numen-title-bar-height` if Q1 = move | everything that loads `@numen/ui/styles.css` |
| `libs/ui/core` | `src/shared/ui/contrast/Contrast.stories.ts` | Modify (tester): cases for the three tones in both token sets | — |
| `libs/ui/shell` | `src/features/window-frame/tokens/frame.css`, `lib/sizes.ts`, `frame.test.ts` | Modify only if Q1 = move | `PaneStrip.vue`, `apps/desktop/flashcards/src/app/App.vue` read the token |
| `apps/desktop` | `internal/frame/frame.go`, `bar_height_test.go` | Modify only if Q1 = move (comment, path of the file read) | — |
| `plugins/views-title-bar` | `frontend/ui/titleBar.css` | Modify: height from the token, slot gap | — |
| `plugins/views-title-bar` | `frontend/ui/TitleBarScale.stories.ts` | Modify (tester): chip gap and chip size at every scale | — |
| `plugins/views-status-bar` | `frontend/ui/statusBar.css` | Modify: height from the token, px values, slot gap | — |
| `plugins/views-status-bar` | `frontend/ui/StatusBarScale.stories.ts` | Create (tester) | — |
| `plugins/tools-updater` | `frontend/UpdateIndicator.vue`, `UpdatePopover.vue`, `styles.css` | Modify | — |
| `plugins/tools-updater` | `frontend/styles.test.ts` | Delete | — |
| `plugins/tools-updater` | `frontend/UpdateIndicator.stories.ts` | Modify (tester) | — |
| `plugins/tools-index-stats` | `frontend/StatsIndicator.vue`, `StatsPopover.vue`, `styles.css` | Modify | — |
| `plugins/tools-index-stats` | `frontend/styles.test.ts` | Delete | — |
| `plugins/tools-index-stats` | `frontend/StatsIndicator.test.ts`, `StatsIndicator.stories.ts` | Modify (tester): the popover is in a portal, `.index-popover` is not inside the wrapper | — |
| `plugins/views-tasks` | `frontend/ui/TasksStatus.vue`, `TaskPopover.vue`, `tasksStatus.css` | Modify | — |
| `plugins/views-tasks` | `frontend/ui/TasksStatus.test.ts`, `TasksStatus.stories.ts` | Modify (tester) | — |
| five plugins above | `manifest.json`, `package.json` | Modify: version bump (`node tools/plugin-versions/check.mjs --write`) | — |

Overlap: branch `fix/tasks-chip-debounce` (worktree `../numen-wt-chip`, no PR) changes `plugins/views-tasks/frontend/ui/TasksStatus.vue`, `TasksStatus.test.ts`, `TasksStatus.stories.ts`, `useChipDisplay.ts`, `lib/chipDisplay.ts`, `index.test.ts`, `manifest.json` and `package.json` (version `2026.11.3`). Whichever merges second resolves the conflict in `TasksStatus.vue` (the template moves into the chip's slots; the timing logic in the script is untouched by this task) and takes the next plugin version.

Jsdom tests: `PopoverPortal` teleports the content to `document.body`. A test that finds the popover inside the mounted wrapper finds nothing; it queries `document.body` and mounts with `attachTo`.

## 3. Negative Invariants (Architectural Prohibitions)

- No bar height, bar font size, bar padding, chip size or bar gap in `rem` or derived from `--numen-interface-scale`, `--numen-inset` or `--numen-text-*`.
- No plugin declares chip sizes, chip grounds, popover position or popover frame. No plugin pads its own slot view.
- No plugin imports `reka-ui`, `@numen/shell` or anything outside `@numen/plugin-sdk` and `@numen/ui`.
- `BarChip` takes no domain type: its props are `label`, `tooltip`, `tone`, `popoverLabel`, `open`.
- Tokens are declared on `:root` in `tokens.css` only; the chip does not declare custom properties on its own element.
- Popover content keeps its rem sizes.
- No test reads CSS text. Looks are checked by stories in Chromium and WebKit.
- No new dependency. reka-ui is already a dependency of `libs/ui/core`.

## 4. Acceptance Criteria

Fixture data is invented. Sizes, spacing, scale behavior and contrast are not tested (see the decision in §5).

### AC-9: A press on the chip opens the popover below it

- **Given**: a `BarChip` with popover content near the top of the viewport
- **When**: the chip is pressed
- **Then**: a dialog appears, the trigger has `aria-expanded="true"`, and the dialog's top edge is 4px below the bottom edge of the bar.

### AC-10: The popover opens upward when there is no room below

- **Given**: a `BarChip` with popover content at the bottom edge of the viewport
- **When**: the chip is pressed
- **Then**: the dialog's bottom edge is at or above the chip's top edge, and the dialog is inside the viewport.

### AC-12: Escape closes the popover

- **Given**: an open popover
- **When**: Escape is pressed
- **Then**: the dialog is gone and the trigger has `aria-expanded="false"`; `update:open` was emitted with `false`.

### AC-13: A press outside closes the popover

- **Given**: an open popover
- **When**: the pointer is pressed on the document outside the chip and the dialog
- **Then**: the dialog is gone.

### AC-14: A press inside the popover keeps it open

- **Given**: an open popover with an invented button in its content
- **When**: the button is pressed
- **Then**: the dialog stays open.

### AC-15: A second press on the chip closes the popover

- **Given**: an open popover
- **When**: the chip is pressed again
- **Then**: the dialog is gone.

### AC-16: A resize keeps the popover open

- **Given**: an open popover
- **When**: the window sends a `resize` event
- **Then**: the dialog stays open.

### AC-17: A chip without popover content opens nothing

- **Given**: a `BarChip` with no `popover` slot
- **When**: the chip is pressed
- **Then**: no dialog appears and no `update:open` with `true` is emitted.

### AC-18: Setting `open` from the parent closes the popover

- **Given**: an open popover bound with `v-model:open`
- **When**: the parent sets `open` to false (an action in the content)
- **Then**: the dialog is gone.

### AC-19: The chip names its trigger and its tooltip

- **Given**: a `BarChip` with an invented `label` and `tooltip`
- **When**: it is drawn
- **Then**: the trigger is a button named by `label`, with `title` = `tooltip`, `aria-haspopup="dialog"`, and out of the tab order (`tabindex="-1"`). Without `tooltip` the trigger has no `title`.

### AC-20: A press survives a face replaced between press and release

- **Given**: a `BarChip` whose face text node is replaced on `pointerdown`
- **When**: the chip is pressed and released in WebKit
- **Then**: the popover opens.

### AC-23: The update notice uses the shared chip

- **Given**: the updater view with an invented newer release
- **When**: it is drawn in the title bar slot and pressed
- **Then**: its face reads `v<version>` and the release dialog opens; the download, release notes and dismiss actions close it.

### AC-24: The index stats use the shared chip

- **Given**: the index stats view with invented stats
- **When**: it is drawn and pressed
- **Then**: the "Index stats" dialog opens with its rows, the close button closes it, and a resize keeps it open. With no stats yet, a press opens no dialog.

### AC-25: The background tasks use the shared chip

- **Given**: the tasks view with invented running and failed tasks
- **When**: it is drawn and pressed
- **Then**: the task list dialog opens; its close button closes it.

## 5. Decisions (answered 2026-11-09)

- **Q1, token location:** move. `--numen-title-bar-height` is declared in `libs/ui/core/src/shared/tokens/tokens.css` under the same name; `frame.css` no longer declares it; the comments of `frame.go` and `sizes.ts` and the path in `bar_height_test.go` point at the new file.
- **Q2, command buttons:** unchanged. The title bar command buttons keep touching as today; the 4px gap applies between view entries (chips) only. The title bar does not change its look.
- **Q3, status bar text:** 13px, the title bar's text size, so both bars read the same.
- **Q4, `tone`:** accepted as proposed; each chip keeps today's ground.
- **Q5, `popoverLabel`:** accepted as proposed.
- **Chip size:** unchanged from today's title bar chip: 20px high, 12px text, 8px padding, in both bars.
- **Gap:** 4px in px; it does not follow the interface scale.
- **Decision 2026-11-09:** no visual tests; the sizes in §1 are implemented but not tested.
