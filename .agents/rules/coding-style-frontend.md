# Frontend Coding Style & Component Guidelines

The conventions for Vue 3 and TypeScript in numen: `libs/ui/components` and the windows under `apps/desktop/editor` and `apps/desktop/flashcards`. The architecture these serve is in [`architecture.md`](./architecture.md), _How an interface component is built_, _The component library is shadcn-vue_ and _How this application is tested_.

TypeScript is strict, with `exactOptionalPropertyTypes` and `noUncheckedIndexedAccess`. A component's props are a contract with another module, and an unchecked one is not a contract.

---

## 1. The domain boundary

Nothing in `libs/ui/components` imports `@numen/protocol`, a wire type, a domain type, or anything under `apps/**` — not the components, not the stories, not the fixtures.

A component takes props it defines itself, in a **vocabulary about drawing**, and emits **opaque identifiers**. A prop named after a vault, a note or an RPC message is the boundary crossed even when its type is a string. Whoever renders the component translates the domain into that shape and translates the identifier back.

The dependency runs `apps/*` → `libs/ui/components`. Never back, never sideways.

---

## 2. The pure core and the humble view

Every component is decoupled into logic and presentation:

**Pure core logic.** Given props, pure TypeScript functions compute render properties — coordinates, states, derived flags — as plain data objects with no DOM, Vue, clock, randomness, or DOM measurement dependencies. Pure logic is in standalone TypeScript modules next to the component (`measure.ts`, `sizing.ts`, `model/`), each with dedicated unit tests.

**Presentational views.** Vue components receive these computed values and render elements without performing business logic or calculations in templates. A template must not calculate values directly, and a computed property must not access `window`, `Date.now()`, or element dimensions directly.

**Browser and platform dependencies are ports**: clocks, `requestAnimationFrame`, viewport geometry, and media preferences must be passed as parameters with browser defaults.

A `computed` in a `.vue` file is a thin projection. Loops, arithmetic, string parsing, and data transformations belong in pure modules tested in isolation without mounting components.

---

## 3. Component directories

A component has a directory under `src/`, named after what it draws:

```text
src/plex/
├── Plex.vue              # the humble view
├── Plex.stories.ts       # the stories
├── Plex.test.ts          # what is emitted, and what is not
├── carry.ts              # pure modules, each with its own test next to it
├── carry.test.ts
├── model/                # the concept: the union, and what varies with it
├── render/               # what the view needs to draw the computed values
└── fixtures/             # deliberately irregular data, never taken from the domain
```

What an application may use is re-exported from `src/index.ts`, and that list is narrower than what the module contains: how a component is built is not how it is used.

**A component is split when it holds a second responsibility** — a section with independent state, or a part rendered on its own elsewhere. The length limits of a function and of a block are in [`architecture.md`](architecture.md) §8.

**A polymorphic component dispatches, it does not inline.** Where a component renders distinct variants by kind, the parent is a shallow dispatcher and each variant is its own component; a heavy dependency — an editor, a composer, a form — is never imported into something that renders a high-frequency row.

---

## 4. Section structure

Inside `<script setup lang="ts">`, in this order, with only the sections that exist:

```vue
<script setup lang="ts">
import { computed, ref, onMounted } from 'vue'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{ ... }>()

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{ ... }>()

/* --------------------------------- State ---------------------------------- */
const isOpen = ref(false)

/* --------------------------------- Hooks ---------------------------------- */
onMounted(() => { ... })

/* -------------------------------- Handlers -------------------------------- */
function onSubmit() { ... }

/* -------------------------------- Helpers --------------------------------- */
function formatValue(val: string) { ... }
</script>
```

**Props → Events (or Props & Emits) → State → Hooks → Handlers → Helpers**, checked from the statements by `numen/vue-script-sections`; a banner comment is optional.

---

## 5. Props, emits and state

- Type-based only: `defineProps<Props>()`, with `withDefaults` where a default is wanted. A runtime object declaration is not used.
- Every event a component emits is declared in `defineEmits<Emits>()`.
- Emits are kebab-case in the template (`@item-selected`); the handler in the script is `onItemSelected`. `update:*` for a model.
- **Do not declare sibling events differing only by a flag.** `annotate: [block?: boolean]`, not `annotate` and `annotateBlock`.
- **A second behaviour is a function, never a mode flag** — and only where a second implementation is intended. A flag with one value is dead configuration.
- **A concept is declared once.** A role, a state, a variant: the type union, the layout, the wording and the colour all read from one declaration.
- No component queries the DOM to read state. State flows through props, emits and models.
- No mutation of a prop, and no destructuring of a reactive object or store without `toRefs()` / `storeToRefs()`.
- A `watch` that could be a `computed` is a finding: derived state recomputed by hand goes stale.
- Whatever a component starts, the same component ends — a listener, a timer, an observer, a subscription.

---

## 6. Types and results

- Types and interfaces are singular nouns: `Tab`, `FileEntry`, `MoveResult`, `TranscriptSegment`. Never a gerund, an adjective or a verb.
- Booleans are `is…` / `has…` / `can…`. Fields are noun phrases that say what they hold: `title`, `completedCount`, `failureReason` — not `about`, `doing`, `done`.
- A fallible operation returns a discriminated result:
  ```ts
  type Result<T, E = ErrorCode> = { ok: true; value: T } | { ok: false; error: E }
  ```
  A bare value, an implicit `null`, or a second word for the failure discriminant (`refusal`, `reason`) is a finding. The word is **`error`**.
- No `any`. No `as` casting away a type the code could state. No non-null assertion standing in for a check.
- No empty `catch {}`. Every catch handles the error, rethrows it, or has a comment saying why it is safely ignored.

---

## 7. Naming and placement

- **Functions and methods** are imperative verb phrases (`getTranscript`, `createTab`, `openVault`) or predicates. Never third-person singular (`carries`, `reads`), never a past participle (`listened`, `spoken`), never a literary metaphor where a standard term exists. A verb with one bare participle or adjective and no object (`getActive`, `getEmitted`, `createResponded`) says nothing about what it returns: name the object (`getActiveTab`, `getEmittedEvents`) or use a predicate. The names check refuses it.
- **Handlers** are `on<Action>`.
- **Composables** are named `use<Feature>` — never a gerund, never a bare noun. The file they live in matches the composable name (`useAppHotkeys.ts`, `useWorkspaceTabs.ts`, `useVaultStatus.ts`).
- **A composable has one coherent responsibility.** A composable that takes an entire component's script — every piece of state, every handler, a long list of exports — is that component undecomposed. Split the component instead.
- **Factories are `create…`.** The word `make` is not used for a factory in frontend code.
- **Files**: directories `kebab-case`, components `PascalCase`, other TypeScript `camelCase`.
- **Domain code stays in its domain and layer.** A file in desktop UI belongs to one of five layers (`shared` → `entities` → `features` → `pages` → `app`). A slice imports lower layers only, through the `index.ts` of the slice, and never a sibling slice of its layer; there is no `@x` cross-import. Imports use the `@/` alias. Placement is the skill [`nm-place-window`](../skills/nm-place-window/SKILL.md).

---

## 8. Styling

- **Every colour, size, radius and duration is a design token.** A literal hex, a raw `rgb()`, a hardcoded duration or a Tailwind palette class (`bg-slate-800`) in a component is a finding.
- **Tokens are declared on the root alone**, in `src/tokens/tokens.css`. A custom property declared on an element beats the same property inherited from the root, so a component that declares its own is no longer changed by the theme.
- **`theme.css` defines Tailwind's names from the tokens, never the reverse**, so `bg-surface` and `var(--numen-surface)` resolve to one value.
- **Text on a ground is readable on it: 4.5:1 for text, 3:1 for large text and for marks.** Text on the accent takes `--numen-accent-ink`, never the inherited colour, `--numen-surface` or a dimmed `opacity`; a hover moves a ground away from its text, not toward it. A person chooses the token pairs in the theme files; no story or test measures a ratio.
- **No `dark:` variant anywhere in the module**, including in a component copied in from the shadcn-vue registry — the tokens are `light-dark()` pairs and `color-scheme` chooses.
- **The root's font size is the interface multiplier.** What scales with the interface is in `rem`; a hairline, a border, a focus ring and the stroke of a handle are one physical line and stay in `px`.
- A component copied in from the registry is edited when it is copied in: its palette, its `dark:` variants and its types are maintained in this module from then on, because the copied source belongs to this module.

---

## 9. Stories and tests

- **A component is built in Storybook**, in isolation, before any application renders it. One built inside a screen has only the edge cases that screen happens to contain.
- **The stories cover the edge cases with deliberately irregular data**: empty, one, far too many, text that is not Latin, text far too long, text with nothing to break at, and no text at all.
- **A setting is a control.** Anything a control can change is an `argType`, and a second story that differs only by the value of one is deleted.
- **Three levels of test.** The assertions that matter are on the pure core as plain functions. The component in jsdom covers what is emitted and what is drawn — and **the negatives belong there**, because what is _not_ emitted and _not_ drawn fails silently and looks right in every screenshot. The stories run as tests in a browser.
- **Both engines.** The window is WebKit on Linux and macOS, Chromium on Windows, so every story renders in both through Playwright. A run in one engine does not test the product.
- **A test must be able to fail.** Break the rule and check that the test fails: "the nodes were arranged" and "the component rendered" pass under almost any implementation.

---

## 10. Code formatting (Prettier & gofmt)

_TypeScript, Vue, JSON and CSS are formatted with Prettier._ Prettier is configured in `prettier.config.mjs`. Format with `npm run format` at the repository root; the check is `npm run format:check`. Prettier's use is mandatory, and a change that is not Prettier-clean is rejected. Today `format:check` runs by hand and through the pre-commit hook, which runs `npx prettier --check` on the staged files; no CI workflow and no make target runs it.

Go code is formatted with `gofmt` (`gofmt -l .`).

- `.editorconfig` — encoding, line endings, the final newline, trailing whitespace, the indent.
- **ESLint, with the boundary and code rules in `eslint.config.js`** — what is semantically wrong: an empty catch, a `ref` mishandled, a module importing what it may not import.
- **A generated file is changed only by its generator.** Nothing under the protocol's generated output is hand-formatted or edited.

---

A comment describes the active behavior or invariant. It does not argue, justify rejected alternatives, or narrate history. See Rule 1 and Rule 13 of `AGENTS.md`.

---

## 12. Verification

```bash
cd libs/ui/components
npm run typecheck
npm run test:unit
npm run build                                   # the windows import @numen/ui from its build
npx vitest run --project 'stories (chromium)'
npx vitest run --project 'stories (webkit)'
```

Take the `libs/ui/components` suites **one at a time**. `npm test` there starts both story instances at once and the run dies before any test executes.

Then the windows, which need the library built first:

```bash
cd apps/desktop/editor         && npm run typecheck && npm test
cd apps/desktop/flashcards && npm run typecheck && npm test
```
