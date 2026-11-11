---
name: nm-write-stories
description: Says what a Storybook story in libs/ui/components (@numen/ui) and libs/ui/workbench (@numen/workbench) shows, which behavior it exercises with a play function, how its fixtures are made, how a component takes its clock and viewport, and how a component uses design tokens. Load it before writing or changing a component, a story, a story fixture or a unit test under libs/ui, and when reviewing one. Trigger with "/nm-write-stories", "write a story", "add a story", or "story for this component".
paths:
  - libs/ui/**/*.stories.ts
  - libs/ui/**/*.vue
---

# Writing stories (`/nm-write-stories`)

This skill answers what a story in `@numen/ui` and `@numen/workbench` contains. Where a component goes is decided by [`nm-place-library`](../nm-place-library/SKILL.md); this skill starts once the component has its folder. A check can tell that a story exists and imports no domain; it cannot tell whether the story shows the states a caller reaches or whether the `play` function proves the behavior. This skill does.

The rules are in `AGENTS.md` (sections 1, 13 and 14), `.agents/rules/architecture.md` (sections 2, 4 and 5) and `.agents/rules/coding-style-frontend.md` (sections 2, 3, 8, 9 and 12). The user's recorded decisions are stated here as targets: no visual-regression test, no test of a pixel size, a gap, a scale or a contrast ratio; behavior is tested (open, close, flip, move); fixtures are invented content. Where code in `main` differs from a target, the target governs; the section "Where main differs" lists the places.

Files: [`checks.md`](checks.md) lists the automatic checks, existing and proposed, with the violations `main` has for each.

## What a story is for

A story is a component in one state, drawn in a real browser, and optionally a script of what a person does to it. Three uses, one file:

- The development surface: the component is built and looked at in Storybook before any window renders it, so its edge cases are designed and not discovered inside a screen.
- The test: the Vitest addon turns every story into a test and runs it in Chromium and in WebKit. The official page says the addon "transforms your stories into component tests". A story with a `play` function is an interaction test; a story without one still renders, runs the `afterEach` checks of `.storybook/preview.ts` (keyboard walk) and fails on an error thrown while drawing.
- The documentation of the component's states for the next author.

A story does not test pure logic (that is a unit test beside the module), and it does not compare a picture, a size or a colour.

## Anatomy

A component folder holds the story beside the component (`Slider.vue`, `Slider.stories.ts`, `Slider.test.ts`). A slice with parts holds one story file for the public component. Fixtures sit in the `fixtures/` folder beside the component or in `src/shared/fixtures/` when two components use them.

```ts
/** Stories of the select: its choices, its groups and the words on them. */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, fn, userEvent, within } from 'storybook/test'
import Select from './Select.vue'
import { CYRILLIC_CHOICES } from '@/shared/fixtures/prose'

interface Knobs { words: string; chosen: string; isDisabled: boolean }

const meta: Meta<Knobs> = {
  title: 'Controls/Select',          // <Group>/<Name>
  component: Select,
  argTypes: { isDisabled: { control: 'boolean' } },
  args: { words: 'Small\nMedium\nLarge', chosen: 'small', isDisabled: false },
  render: (args) => ({ /* a small harness that holds the state the caller holds */ }),
}
export default meta
type Story = StoryObj<Knobs>

/** Three choices, none of them on a group. */
export const ASelect: Story = {}
```

- One `title` per file, `<Group>/<Name>`: `Controls/…` for a general control, the slice name for a slice, `Workspace/…` for chrome.
- The `render` function is a harness: it holds the state a caller would hold (`ref` for the model, a list the events change) so the component behaves as it does in a window. It imports no domain type; it maps nothing from a wire message.
- A setting a person can change is an `argType` and a control. Data the component can be given in a form that breaks an assumption (empty, one, far too many, another script, far too long, nothing to break at, no text) is a named story, because each is a state that is looked at and run. A second story that differs from another only by a toggle is deleted and the toggle becomes a control.
- Every export has a one-line doc comment stating the state or the behavior. A state story is named for the state (`Empty`, `FarTooMany`, `OtherScripts`); a behavior story is named like a test title, subject and standard verb (`DragsIntoARow`, `TakesAName`). The names follow `AGENTS.md` section 13.
- `tags: ['!dev']` keeps a pure behavior story out of the sidebar; it still runs.

## Which states and behaviors a story exercises

List these before writing the first story, from the component's props, events and slots.

**States.** One story per state a caller can reach:

- the ordinary content;
- each value of a prop that changes what is drawn or what can be done (`isDisabled`, `isOpen`, `isBusy`, a marker);
- the data edge cases of [coding-style-frontend section 9](../../rules/coding-style-frontend.md): empty, one, far too many, text that is not Latin, text that runs right to left, text far too long, text with nothing to break at, no text;
- a value in force that is none of the choices offered;
- a state that is entered and left (opening, loading, failing).

**Behaviors.** One `play` function per behavior the component exists for, stated as what a person does and what follows:

- pointer: click, right click, press and drag, hover onto a submenu;
- keyboard: arrows, Enter, Space, Escape, Tab and Shift+Tab, typing a name, typing to jump;
- focus: where focus lands when the component opens, where it returns when it closes, that it does not leave a modal;
- what is emitted and with which identifier, and what is not emitted;
- what is drawn after the event: a row opened, a menu closed, a value changed, an item reordered;
- placement that is behavior: a menu that flips at the edge stays inside the window, and a menu opened inside a box that clips is not clipped. The assertion is that the thing is whole and on screen, never a fixed number of pixels.

A behavior that depends on layout, focus-visible, paint order or pointer geometry is proven in a story, because jsdom performs no layout. A behavior that is only "emits X on Y" is proven in jsdom in `Component.test.ts`, which is also where the negatives go (what is not emitted and not drawn, which fails silently in a story).

### How a `play` function drives a component

The official page: "use a play function to simulate user behavior like clicks, typing, and submitting a form". The tools come from `storybook/test`: `userEvent`, `expect`, `fn`, `within`, `waitFor`.

- Strike what a person strikes: `userEvent.click`, `userEvent.keyboard('{ArrowDown}')`, `userEvent.type`, `userEvent.pointer([...])` for a drag. Never call a component method, set a ref or dispatch a hand-built event to make the state change; an event the component needs from a person is produced the way a person produces it. Where one browser-level gesture is needed (a pointer sweep, the system colour scheme), the commands in `.storybook/browser.ts` exist.
- Find elements the way a screen reader does: `within(canvas).getByRole('menuitem', { name: 'Open' })`. A query by role and name fails when the control has no role or no name, which a person using assistive technology would also miss. A `data-*` hook or a class is the second choice and is for an element that has no role.
- Spy only on the outer edge. The component's emitted events are the outer edge: pass `fn()` as the `onX` arg and assert `toHaveBeenCalledWith(id)`. The official page: "you will use `fn` as an `arg` value when writing your story". Everything between the event and the edge is real. A story mocks no module, no component, no `getBoundingClientRect`.
- Wait for what takes a frame: `await waitFor(() => expect(...))`. A fixed sleep is a defect.
- Group a long script with `step('…', async () => {})`; the official page: "group sets of related interactions together using the step function".
- A `play` function ends in an assertion about what a person would see or what the caller received. "It rendered" and "the nodes were arranged" pass under almost any implementation: break the rule once and check that the story fails (coding-style-frontend section 9).

## Fixtures

A fixture is invented. Its words are general: Work, Plans, Notes, Friday; a garden shed; seeds and spades. It is never copied from the user's vault, never a real note title, path, book or link, and never a value built from a wire type.

- Text with awkward shape lives in `src/shared/fixtures/prose.ts` (`DEVANAGARI`, `ARABIC`, `RUSSIAN`, `UNBREAKABLE`, `LINK`, `LONG`, `EMPTY`) and is imported by name. A new awkward shape is added there once, with the assumption it breaks in the comment of the constant.
- Data for one component lives in `fixtures/` beside it (`features/tree/fixtures/rows.ts`), typed with the component's own types.
- The `ListItem`, `Row` and `MenuItem` types are the component's own: opaque `id` strings, `text`, flags in the drawing vocabulary. A fixture carries no `vaultId`, `noteId`, RPC message, enum from the protocol or path under `apps/`.
- A fixture generating bulk (`Array.from({ length: 2000 }, …)`) is deterministic: no `Math.random`, no `Date.now`, no `new Date()`.
- The domain words in a text a story shows are examples of text, not types: "Delete the note?" is a string. A prop, an event or a type named for a domain term is the boundary crossed ([`nm-place-library`](../nm-place-library/SKILL.md)).

## Clock and viewport

A component that depends on time or on the size of the window takes it as a parameter with a browser default, and a story passes its own only when the time or the size is what the story is about.

- The type is `Clock` (`now`, `schedule`, `cancel`) from `src/shared/lib/time/clock.ts`; `browserClock` is the default; the component reads time only through the parameter (`getNow()` and `getMonotonicNow()` are the sole readers of the machine clock).
- Real time is the default in a story. A story of a behavior that waits (a long press, a hover delay, a debounce) passes the clock from `src/shared/fixtures/clock.ts` (`stubClock`) and advances it with `tick(to)`, so the story does not sleep and does not flake. The same clock is used in the jsdom test of the component.
- The viewport is a `Size` prop with a default measured from the element (`useViewport`). To place a component at an edge, give the harness a container of the size the story needs or pass `viewport`; do not resize the browser page. The frame is 1200 by 900 (`.storybook/browser.ts`).
- Randomness and identifiers are parameters in the same way. A story passes a seeded or fixed source.
- The toolbar globals `theme`, `interface` and `font` set `color-scheme` and the two scale variables on the root. A story that must run in the dark theme sets `globals: DARK` from `src/shared/fixtures/theme.ts`, and its `play` asserts a behavior.

## Focus and names

No accessibility addon and no axe run is part of the story projects. What a person looks at, and what a `play` function asserts when it can:

- whether the accessible name says the right thing ("Delete note" and not "Button"); a query by role and name in the `play` function fails when the control has none;
- whether the tab order follows the reading order and the visual order;
- whether a focus ring is visible and where focus goes after a dialog closes (the `play` function asserts the destination);
- whether the component announces a state change a screen reader user needs (busy, selected, expanded);
- whether motion respects the reduced-motion preference;
- whether the component works at 200 percent of the interface scale (the toolbar switch).

Contrast is not tested: the user decided against contrast tests. The token pairs are chosen by a person in the theme files.

## Tokens

A component draws with design tokens and declares none.

- Colours, radii, durations, fonts and the spacing steps that exist as tokens are used through the token: a Tailwind name from `theme.css` (`bg-surface`, `text-ink`) or `var(--numen-surface)` in a scoped block. No hex, no `rgb()`, no named colour, no `var()` fallback, no Tailwind palette class, no `dark:` variant.
- Tokens are declared in `src/shared/tokens/tokens.css` and nowhere else; a component never declares a `--numen-` property. A new value that two components use is a new token, added with its Tailwind name in `theme.css`.
- What scales with the interface is in `rem` or `em`. A `rem` or `em` literal sits directly on the property (`padding: 0.375rem`); `px` only for a 1px hairline border.
- A dimension that belongs to one component only (a row height, an indent) may be a local custom property named for the part (`--row`, `--indent`) in `rem` or `em`, declared in that component's block. It is not a token until a second component needs it.
- The theme is chosen by `color-scheme` on the root: the tokens are `light-dark()` pairs. The component root carries the class `numen`.
- Pure geometry and arithmetic are computed in a `.ts` module beside the component, not in the template or in a `computed`.

## Which test project runs what

| Project | Engine | Runs |
| --- | --- | --- |
| `unit` | jsdom | `*.test.ts`: pure functions, models, emitted events and the negatives, with `@vue/test-utils` |
| `stories (chromium)` | Playwright Chromium | every story and its `play` function |
| `stories (webkit)` | Playwright WebKit | the same stories; the window is WebKit on macOS and Linux |

Geometry, focus, pointer and keyboard are proven in the story projects in both engines. jsdom decides logic. Use `getBoundingClientRect` in a story to prove that something is whole, inside or ordered, never to pin a size.

In `libs/ui/components` and `libs/ui/workbench` take the projects one at a time; `npm test` starts both story instances at once and the run dies before a test executes:

```bash
npm run test:unit
npx vitest run --project 'stories (chromium)'
npx vitest run --project 'stories (webkit)'
```

`libs/ui/workbench` imports `@numen/ui` through its build: run `npm run build` in `libs/ui/components` first. Heavy runs belong to the verification machine, not to editing; while writing, run `npm run typecheck` and `npm run lint`.

## Worked examples

Each follows the target.

- A dialog whose first story asserts focus by role and name: `apps/desktop/editor/src/pages/settings/ui/confirm-dialog/ConfirmDialog.stories.ts:34-41`. The state stories below it (`:44-54`) are a question that is far too long and a question in another script, with no `play`.
- Choosing emits the identifier the caller gave, asserted through an `fn()` arg, and the menu closes: `libs/ui/components/src/shared/ui/menu/Menu.stories.ts:146-158`. The `args` that carry `fn()` are `:133-134`.
- Keyboard focus lands on the first item and returns to the opener on Escape, asserted by role: `Menu.stories.ts:186-205` (read the body; the export name is not a model).
- A tree driven with real pointer events and the keyboard, over invented rows: `libs/ui/components/src/features/tree/ui/Tree.stories.ts:305-356` (`dragTo`, `WalksWithTheKeyboard`, `DragsIntoARow`), rows in `libs/ui/components/src/features/tree/fixtures/rows.ts:1-18`, renaming typed in another script at `Tree.stories.ts:466-480`.
- Data edge cases as named states with shared awkward text: `libs/ui/components/src/shared/ui/select/Select.stories.ts:102-138`, text in `libs/ui/components/src/shared/fixtures/prose.ts:1-61`.
- A modal that keeps the keyboard, driven by Tab and Shift+Tab: `libs/ui/workbench/src/features/palette/ui/Palette.stories.ts:264-284`; a choice with Enter and Shift+Enter through `fn()` arg: `:247-258`.
- A space bar toggles a switch (focus, then key, then state): `libs/ui/components/src/shared/ui/switch/Switch.stories.ts:211-225`.
- A clock as a parameter with a browser default: `libs/ui/components/src/features/tree/ui/Tree.types.ts:15` (`clock?: Clock`) and the stub that tests pass: `libs/ui/components/src/shared/fixtures/clock.ts:1-40`, used at `libs/ui/components/src/features/tree/ui/Tree.test.ts:436-437`.

## Where main differs

These places do not follow the target and are not models.

- `Switch.stories.ts` (`BusyUntilTheAnswer`, `TheThumbSlides`) measures the thumb position per frame. Behavior is the `aria-checked` change and the busy state.
- `Menu.types.ts:16` takes `clock?: () => number`; the target is `Clock`.
- No story passes `viewport`; the stories place a menu with `at` coordinates.
- `Select.stories.ts` and `Tree.stories.ts` find elements by class (`.menu__item`, `.tree__field`); the target is role and name.
- `Tree.test.ts:44` replaces `Element.prototype.getBoundingClientRect` in jsdom. A test of layout is a story.
- `Prose.stories.ts` has no `play` for a component that emits `follow`; `DragPreview` has no story.
- `libs/ui/components/README.md` lists a layout (`panel/`, `composer/`, `thread/`) that no longer exists.
- Export names such as `Lit`, `GivingItBack` and `NothingBelowIt` are narrative; the target is a standard verb phrase for the behavior.

## What the checks enforce

Today, in `main`:

- `npm run lint` in `libs/ui/components` and `libs/ui/workbench`: an import of a domain or wire package in a component, a story or a fixture, `Date.now()`, `new Date()` and `performance.now()` outside the clock function, `Math.random()`, `setTimeout` and `setInterval` in a pure segment, a layer that imports a higher layer.
- `styles-use-tokens` (stylelint): a colour literal, a `--numen-` declaration outside the token and theme files, a `var()` with a fallback, an undeclared `--numen-` token.
- `no-suppressions`: `eslint-disable`, `@ts-ignore` and the like.
- `numen/no-style-declaration-read` (ESLint, typed): a read of `getComputedStyle(...)` or `element.style` in a story, a test or a fixture.
- The keyboard walk after every story: a tab stop that cannot be seen, a tab stop without a name, a component that keeps Tab.

Proposed, in [`checks.md`](checks.md): every public component has a story; every component that emits has a story with a `play` function; a component declares no raw length or font value. Three of these are built by Package F25 of the refactor plan: a story for every public component, a `play` story for every emitting one, and `eslint-plugin-storybook`. The rest of the list in `checks.md` stays as candidates.

## What no check decides

- Whether the stories show the states a caller reaches: the list of states and behaviors comes from the props, events and slots, and a person reads the component against it.
- Whether a `play` function proves the behavior: break the behavior and see the story fail.
- Whether a fixture is invented and general: a check can see a wire import, not a real name.
- Whether the accessible name is the right words, and the items under "Focus and names".
- Whether a component is general enough for `@numen/ui` ([`nm-place-library`](../nm-place-library/SKILL.md)).
- Whether the design is right: spacing steps, hierarchy, density, motion. The owner looks at the story in both themes and at 200 percent.
