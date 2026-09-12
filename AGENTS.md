# Numen — Coding Conventions

These rules apply to all code in the numen repository. Every agent (Claude,
Gemini, or any other) must follow them.

---

## 1. Comments state the rule, and stop

A comment says what holds. It never names an alternative that is not in the
code, and never cites an ADR number — restate the rule inline instead.

Before writing a comment, delete each clause in turn. If what remains still
describes the code, the deleted clause was argument and does not belong.

Rewrite on sight of any of these:

- `rather than X`, `instead of X`, `not X but Y`
- `X would have…`, `otherwise somebody would…`, `told only Y, a caller would…`
- `so that we do not…`, `to avoid…`
- `these are those X`, `which is why this exists`
- benchmark numbers, or any comparison with a previous implementation
  (measurements live only in `docs/performance.md`)

Two sentences is usually the ceiling. The same holds in ADRs: a passage that
would read as true with nothing decided is narration.

---

## 2. Function and method naming (all languages)

Functions and methods take **imperative verb phrases** in the base form
(`get…`, `read…`, `write…`, `fetch…`, `create…`, `open…`, `delete…`) or
**predicate phrases** (`is…`, `has…`, `can…`).

Forbidden patterns:

| Pattern | Example | Fix |
|---|---|---|
| Third-person singular | `carries`, `attends`, `reads` | `carry`, `attend`, `read` |
| Past participle | `listened`, `spoken`, `cued` | `getTranscript`, `transcribe` |
| Bare noun / adjective | `highlight`, `book` | `getHighlight`, `createBook` |
| Literary metaphor | `minting`, `held`, `cold` | `getTitle`, `getActiveTab`, `getTitle` |

A method states **what operation is performed** using standard engineering
terms, never literary third-person narrative.

---

## 3. Event handler naming (frontend — TypeScript / Vue)

Event handlers are named `on<Action>`:

```ts
// ✅ correct
onSubmit, onClose, onClick, onSelect, onToggle

// ❌ wrong
cold, submitted, handle, doThing
```

In Vue `<template>`, emits use kebab-case (`@item-selected`); the
corresponding handler in `<script>` is `onItemSelected`.

---

## 4. Vue component structure (`<script setup>` / `<script>`)

All Vue components organize their `<script>` block into four standard sections
separated by comment section dividers:

```vue
<script setup lang="ts">
// --- Props & Emits ---
const props = defineProps<{ ... }>()
const emit = defineEmits<{ ... }>()

// --- State ---
const isOpen = ref(false)
const title = computed(() => ...)

// --- Handlers ---
function onSubmit() { ... }
function onClose() { ... }

// --- Helpers ---
function formatValue(val: string) { ... }
</script>
```

Rules:
1. Always follow the order: **Props & Emits $\rightarrow$ State $\rightarrow$ Handlers $\rightarrow$ Helpers**.
2. Large templates or sections with independent state must be extracted into separate child components.
3. Complex business logic must not live in `.vue` files — extract it into composables or domain models.

---

## 5. Composables naming (`use<Feature>`)

Composables and reactive state factories in Vue must be named with the `use…` prefix:

```ts
// ✅ correct
useBookTab, useDocumentTab, useAgentChat, useFileManager

// ❌ wrong (gerunds, verbs, literary words)
booking, documenting, talking, opening
```

---

## 6. Type and interface naming (all languages)

Types and interfaces are **singular nouns** or noun phrases:

```ts
// ✅
Tab, FileEntry, MoveResult, TranscriptSegment

// ❌ gerunds, adjectives, verbs
Opening, Doing, Failed
```

### Boolean fields

Boolean fields and properties use `is…` / `has…` / `can…` prefix:

```ts
// ✅
isUserRequested: boolean
hasTranscript: boolean

// ❌
userRequested: boolean   // reads as past tense, not boolean
failed: string           // "failed" looks boolean but is a string — rename to failureReason
```

### Struct / interface fields

Fields are **noun phrases** that say what they hold:

```ts
// ✅
title, completedCount, totalCount, failureReason, progressPercent

// ❌ cryptic or ambiguous
about, doing, done, total, counting
```

---

## 7. Factory and constructor functions

### Frontend (TypeScript / Vue)

Factory functions use `create…`:

```ts
// ✅
createTab, createBookKind, createConfiguration

// ❌
makeTab, bookKind, newTab
```

The word `make` is **not used** in frontend code for factories.

### Go

Go follows its own idioms: `New…` for constructors, `make` for built-in
allocation. Both are allowed.

```go
// ✅
NewServer(), NewReading(), make(map[string]string)
```

---

## 8. Result pattern for fallible operations (frontend)

Functions that can fail return a **discriminated result**, never a bare value,
implicit `null`, or custom `refusal`:

```ts
type Result<T, E = ErrorCode> = { ok: true; value: T } | { ok: false; error: E }

// ✅
function openVault(id: string): Result<Vault, ErrorCode>

// ❌ caller cannot tell success from failure, or uses custom discriminant
function openVault(id: string): Vault | ErrorCode | undefined
function openVault(id: string): { ok: boolean; refusal?: RefusalReason }
```

Pick **one** word for the failure discriminant: **`error`** (never `refusal`, `reason`, or `refusalReason`).

---

## 9. Domain terminology

Use the **project glossary** (`docs/glossary.md`) as the single source of
truth for domain terms. When the glossary and code disagree, fix the code.

Specific term rules:

| Wrong | Correct | Context |
|---|---|---|
| `refusal`, `refused` | `error`, `errorCode` | operation error discriminants |
| `readings`, `reading` | `recognition`, `recognizedText` | OCR extraction output |
| `stretch` | `span` (`{ from, to }`) | text ranges (unify on Span) |
| `address` (for link destination) | `link` (`note:...`, `asset:...`) | links between files |
| `overtaken` | `stale` | file modified externally on disk |
| `spoken`, `spokenBy` | `transcript` | audio/video transcription output |
| `reflow` for EPUB | `book` | EPUB = book; PDF = document |
| `make` (frontend) | `create` | factory functions in TS/Vue |

The glossary itself must contain **only project-specific terms** — terms that
have a meaning in this project different from or more specific than their
general meaning. Obvious industry terms (tree, drag, chunk, window, theme) do not belong.

---

## 10. The roles, and where each tree's own rules are written

This file holds what every language here shares. What is true of one tree only
— its layers, what each layer holds, which way an import may point, the checks
that prove it — is written in the role for that tree. Read the role before
touching the tree.

| Role | Covers | File |
|---|---|---|
| frontend-engineer | writes TypeScript and Vue: `modules/apps/desktop/**`, `modules/libs/ui` | [`.claude/agents/frontend-engineer.md`](.claude/agents/frontend-engineer.md) |
| go-engineer | writes Go: `modules/libs/core`, `modules/apps/**` | [`.claude/agents/go-engineer.md`](.claude/agents/go-engineer.md) |
| architecture-reviewer | reviews where code stands and which way it reaches, both languages | [`.claude/agents/architecture-reviewer.md`](.claude/agents/architecture-reviewer.md) |
| frontend-reviewer | reviews TypeScript and Vue against these conventions | [`.claude/agents/frontend-reviewer.md`](.claude/agents/frontend-reviewer.md) |
| go-reviewer | reviews Go against the decision records and Go practice | [`.claude/agents/go-reviewer.md`](.claude/agents/go-reviewer.md) |
| naming-reviewer | reviews the names of functions, types, files and folders | [`.claude/agents/naming-reviewer.md`](.claude/agents/naming-reviewer.md) |

The two engineer roles are where the layers are written down. The four reviewer
roles judge against them and change nothing.

A rule that stands in the way is not worked around. The edge goes into the
`baseline` of the check that refused it, with a line saying why. Those lists
only shrink.

What checks all of it: `npm run check --prefix modules/tools/depgraph`,
`node --test modules/tools/lint/*.test.mjs`, and `go test ./container/...` in
`modules/libs/core`.

---

## 11. Single Responsibility — no god objects

Do not build a single type or composable that handles all variants of a
concept. Split by capability:

```ts
// ❌ one ArtifactRunner that can transcribe, correct, fetch for all artifact types
class ArtifactRunner { transcribe(); correct(); fetch(); }

// ✅ separate by capability, composed via generics or mixins
interface Transcribable { getTranscript(): Transcript }
interface Correctable  { applyCorrection(c: Correction): void }
```

When a file exceeds ~200 lines or accumulates more than ~8 imports, consider
splitting it into smaller focused modules.

---

## 12. Known gotchas

- **Every worktree shares one `git stash` stack.** Two agents stashing at once
  cross: one `pop` takes the other's entry into the wrong tree. For a clean
  tree, commit first and `git checkout HEAD~1 -- <path>`, or copy the file
  aside.
- **The windows reach `@numen/ui` through its build.** Run `npm run build` in
  `modules/libs/ui` before the suites in `modules/apps/desktop/ui` and
  `flashcards`, or they fail on `Cannot find module '@numen/ui'`.
- **`node_modules` symlinked from the primary checkout resolves
  `@numen/protocol` to whatever branch that checkout is parked on.** A
  generated symbol missing there is that, not the branch under test. Install
  for real when the answer matters.
- **`usecase/flashcards` and `adapter/flashcardsui` run close to the 10-minute
  per-package limit under `-race`.** A timeout there is the machine's load, not
  the code.
- **`npm test` in `modules/libs/ui` starts both story instances at once**, and
  the run dies before any test executes. Take them one at a time:
  `npm run test:unit`,
  `vitest run --project 'stories (chromium)'`,
  `vitest run --project 'stories (webkit)'`. WebKit does run on this machine.
