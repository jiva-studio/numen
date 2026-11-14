# Numen — Coding Conventions

These rules apply to all code in the numen repository. Every agent (Claude,
Gemini, or any other) must follow them.

## Canonical guides

- **Architecture guidelines**: [`.agents/rules/architecture.md`](.agents/rules/architecture.md)
- **Backend coding style (Go)**: [`.agents/rules/coding-style-backend.md`](.agents/rules/coding-style-backend.md)
- **Frontend coding style (Vue / TypeScript)**: [`.agents/rules/coding-style-frontend.md`](.agents/rules/coding-style-frontend.md)
- **Pull requests, commits and issues**: [`.agents/rules/writing-changes.md`](.agents/rules/writing-changes.md)
- **Unified review skill (`/nm-review`)**: [`.agents/skills/nm-review/SKILL.md`](.agents/skills/nm-review/SKILL.md)
- **Per-module conventions**: `AGENTS.md` inside `libs/core/kernel`, `libs/ui/components`, `libs/core/protocol`, `apps/desktop`.
- **Placement and story skills**: the next section.

## Placement skills

Before adding or moving code, or writing a story, load the skill of the area. Each says which layer, slice or package the code goes to and how it is shaped; the rules it rests on are in `.agents/rules/`.

- [`nm-place-core`](.agents/skills/nm-place-core/SKILL.md): `libs/core/kernel`, `libs/note`, the Go side of `apps/desktop`. Layers of the hexagon, ports, use cases, adapters, container.
- [`nm-place-window`](.agents/skills/nm-place-window/SKILL.md): `apps/desktop/editor`, `apps/desktop/flashcards` and the other windows under `apps/`. FSD layers, slices, segments.
- [`nm-place-plugin`](.agents/skills/nm-place-plugin/SKILL.md): `plugins/`, `libs/plugin/sdk-go`, `libs/plugin/sdk-ts`, `libs/plugin/loader`. Manifest, backend packages, frontend slices, what is copied and what is in the SDK.
- [`nm-place-library`](.agents/skills/nm-place-library/SKILL.md): `libs/ui`, `libs/core/client`, `libs/core/protocol`, `libs/plugin/manager`, `libs/plugin/loader`. Which library shared code belongs to, or whether it stays in a window or a plugin.
- [`nm-write-stories`](.agents/skills/nm-write-stories/SKILL.md): `libs/ui/components` and `libs/ui/workbench`. What a Storybook story shows, how a `play` function and its fixtures are written, how a component takes its clock and viewport, and how it uses design tokens.

A red check is fixed in the code. Never loosen the check, add an exclusion or move code to a place the check does not look at (rule 11 of section 15, and §9 and §10 of the architecture rules).

## Code review protocol

A review is the 3-stage pipeline of [`.agents/skills/nm-review/SKILL.md`](.agents/skills/nm-review/SKILL.md): gatekeeper ([`stages/1-gatekeeper.md`](.agents/skills/nm-review/stages/1-gatekeeper.md)), bug hunter ([`stages/2-bughunter.md`](.agents/skills/nm-review/stages/2-bughunter.md)), adversary ([`stages/3-adversary.md`](.agents/skills/nm-review/stages/3-adversary.md)), and the report templated in the skill. A failing gate stops the pipeline. Go tests run without `-race` locally; the race detector runs in CI only (`GO_RACE=1` turns it on for a make target). One pipeline covers Go and Vue. A reviewer reports findings and does not fix them.

Every pull request follows the draft-first pipeline in [`.agents/rules/process.md`](.agents/rules/process.md): open the pull request as a draft (`gh pr create --draft`), bring CI to green (`gh pr checks --watch`), run `/nm-review` and fix any findings until `/nm-review` approves and CI is green, then mark the pull request ready for review (`gh pr ready`) and request a CodeRabbit review (`gh pr comment --body "@coderabbitai review"`).

---

## 1. Comments state facts and invariants

A comment describes the active behavior, invariant, or constraint. It does not argue, justify rejected alternatives, or narrate history.

Before writing a comment, verify that it provides concrete value that code cannot express directly.

Rewrite on sight of any of these:

- `rather than X`, `instead of X`, `not X but Y`
- `X would have…`, `otherwise somebody would…`, `told only Y, a caller would…`
- `so that we do not…`, `to avoid…`
- `these are those X`, `which is why this exists`
- benchmark numbers, or any comparison with a previous implementation
  (measurements are stored only in `docs/performance.md`)

Comments must be concise, factual, and strictly technical engineering English. Avoid storytelling and literary cadence.

### Where a comment goes

| Place                                                              | Comment                                                                   |
| ------------------------------------------------------------------ | ------------------------------------------------------------------------- |
| Package (`doc.go`), file header of a module entry                  | yes, what the package or module is for                                    |
| Type, interface, class                                             | yes, what it represents and its contract                                  |
| Exported function and method, interface method                     | yes, one sentence of contract: result, errors, side effects               |
| Struct field, interface or type property, class property, Vue prop | no                                                                        |
| Unexported function, private method                                | no                                                                        |
| Inside a function body                                             | only a non-obvious why: an outside constraint, a workaround, an invariant |
| `.proto` messages and fields                                       | yes, the protocol is a contract between languages                         |

A field states its meaning through its name and type. A fact the name cannot carry goes to the type's comment or into the type itself:

```go
// Limits bounds one import. A zero MaxFiles imports every file.
type Limits struct {
	MaxFiles int
	Timeout  time.Duration
	state    struct {
		mu    sync.Mutex
		count int
	}
}
```

A Go type whose field comments are documentation for someone else carries a marker line at the end of its doc comment, and its exported fields keep their comments:

```go
// Spec describes a tool: its name, its purpose and where it is offered.
//
//numen:api
type Spec struct {
	// Name is the name the agent calls the tool by.
	Name string
}
```

- `//numen:api`: a type of the plugin SDK; plugin authors read the field comments in the SDK documentation.
- `//numen:settings`: a settings type; `apps/docs/tools/manual.mjs` writes the field comments into the settings reference and refuses a settings type without the marker.

`tools/checks/style/comments/check.mjs` rejects the rest in Go, TypeScript and Vue: on staged files in the pre-commit hook, on every file in CI (`make lint_comments`). `--fix` removes the reported comments.

---

## 2. Function and method naming (all languages)

Functions and methods take **imperative verb phrases** in the base form
(`get…`, `read…`, `write…`, `fetch…`, `create…`, `open…`, `delete…`) or
**predicate phrases** (`is…`, `has…`, `can…`).

Forbidden patterns:

| Pattern               | Example                       | Fix                                    |
| --------------------- | ----------------------------- | -------------------------------------- |
| Third-person singular | `carries`, `attends`, `reads` | `carry`, `attend`, `read`              |
| Past participle       | `listened`, `spoken`, `cued`  | `getTranscript`, `transcribe`          |
| Bare noun / adjective | `highlight`, `book`           | `getHighlight`, `createBook`           |
| Literary metaphor     | `minting`, `held`, `cold`     | `getTitle`, `getActiveTab`, `getTitle` |

A method states **what operation is performed** using standard engineering
terms, never literary third-person narrative.

---

## 3. Event handler naming (frontend — TypeScript / Vue)

Event handlers are named `on<Action>`:

```ts
// ✅ correct
;(onSubmit, onClose, onClick, onSelect, onToggle)

// ❌ wrong
;(cold, submitted, handle, doThing)
```

In Vue `<template>`, emits use kebab-case (`@item-selected`); the
corresponding handler in `<script>` is `onItemSelected`.

---

## 4. Vue component structure (`<script setup>` / `<script>`)

All Vue components order their `<script>` block in sections: props and events, state, hooks, handlers, helpers. A section banner is optional, and a banner is the 80-character form:

```vue
<script setup lang="ts">
/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{ ... }>()

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{ ... }>()

// Or when combined:
/* ----------------------------- Props & Emits ------------------------------ */

/* --------------------------------- State ---------------------------------- */
const isOpen = ref(false)
const title = computed(() => ...)

/* --------------------------------- Hooks ---------------------------------- */
onMounted(() => { ... })

/* -------------------------------- Handlers -------------------------------- */
function onSubmit() { ... }
function onClose() { ... }

/* -------------------------------- Helpers --------------------------------- */
function formatValue(val: string) { ... }
</script>
```

Rules:

1. Always follow the order: **Props → Events (or Props & Emits) → State → Hooks → Handlers → Helpers**. `numen/vue-script-sections` reads the order from the statements: macros, then state (variables, composable calls), then hooks (`watch`, `on...` lifecycle calls), then handlers (`on<Action>` functions), then other functions.
2. A function a template binds to an event is named `on<Action>`, and no function of a script starts with `handle` (`numen/handler-naming`).
3. A section with independent state, or a part rendered on its own elsewhere, is extracted into its own component.
4. Business logic does not live in a `.vue` file — it is a composable or a domain model.

---

## 5. Composables naming (`use<Feature>`)

Composables and reactive state factories in Vue must be named with the `use…` prefix:

```ts
// ✅ correct
;(useBookTab, useDocumentTab, useAgentChat, useFileManager)

// ❌ wrong (gerunds, verbs, literary words)
;(booking, documenting, talking, opening)
```

---

## 6. Type and interface naming (all languages)

Types and interfaces are **singular nouns** or noun phrases:

```ts
// ✅
;(Tab, FileEntry, MoveResult, TranscriptSegment)

// ❌ gerunds, adjectives, verbs
;(Opening, Doing, Failed)
```

### Boolean fields

Boolean fields and properties use `is…` / `has…` / `can…` prefix:

```ts
// ✅
isUserRequested: boolean
hasTranscript: boolean

// ❌
userRequested: boolean // reads as past tense, not boolean
failed: string // "failed" looks boolean but is a string — rename to failureReason
```

### Struct / interface fields

Fields are **noun phrases** that say what they hold:

```ts
// ✅
;(title, completedCount, totalCount, failureReason, progressPercent)

// ❌ cryptic or ambiguous
;(about, doing, done, total, counting)
```

---

## 7. Factory and constructor functions

### Frontend (TypeScript / Vue)

Factory functions use `create…`:

```ts
// ✅
;(createTab, createBookKind, createDocumentTab)

// ❌
;(makeTab, bookKind, newTab)
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

A concept is named with the standard term of its field (W3C, LSP, MIME, gRPC codes).

Specific term rules:

| Wrong                | Correct              | Context                          |
| -------------------- | -------------------- | -------------------------------- |
| `refusal`, `refused` | `error`, `errorCode` | operation error discriminants    |
| `spoken`, `spokenBy` | `transcript`         | audio/video transcription output |
| `make` (frontend)    | `create`             | factory functions in TS/Vue      |

### Vault and link formatting rules

- **Books and passages:** `[Passage description](numen:<path>?start=<offset>&length=<len>)` (using the `uri` field from `library_search`).
- **Notes:** `[[Note Title]]` or `[[notes/path.md]]`.
- **Prohibited:** Never format vault notes, books, or passages as `file://` URLs. The `file://` scheme is only for repository source code files.

---

## 10. File placement — domain code stays in its domain

Code that serves one domain is in the slice of that domain: in a window under `apps/desktop/<window>/src/<layer>/<slice>/` (note types in `entities/note`, file entries in `entities/file`), or in its plugin under `plugins/`. What two or more slices use is in a lower layer, at the bottom `shared`, under the word for what it is. A type only one domain reads never moves down. Layers, slices and segments are in [`nm-place-window`](.agents/skills/nm-place-window/SKILL.md) and [`.agents/rules/architecture.md`](.agents/rules/architecture.md) §4.

---

## 11. Single Responsibility — no god objects

Do not build a single type or composable that handles all variants of a
concept. Split by capability:

```ts
// ❌ one ArtifactRunner that can transcribe, correct, fetch for all artifact types
class ArtifactRunner {
  transcribe()
  correct()
  fetch()
}

// ✅ separate by capability, composed via generics or mixins
interface Transcribable {
  getTranscript(): Transcript
}
interface Correctable {
  applyCorrection(c: Correction): void
}
```

A file is split when it holds a second responsibility. The unit of organisation
is the thing, not the kind of thing: a folder collecting every type, every
handler or every use case is unstructured at fifty entries and was already unstructured at
five.

## 12. Naming conventions across formats and files

Each surface of the repository takes one casing convention:

| Surface | Convention | Examples |
| --- | --- | --- |
| **JSON keys** (settings files, plugin manifests, plugin RPC payloads, tool argument and result schemas, index and cache payloads, test fixtures) | `camelCase` | `maxTokens`, `floorHint`, `pathPrefix`, `taskId` |
| **Protobuf `.proto` fields** | `lower_snake_case` (proto3 JSON mapping is `camelCase`) | `max_steps`, `vault_id` |
| **Note frontmatter keys** | Obsidian convention (`lowercase`) | `id`, `type`, `title`, `tags`, `aliases`, `cssclasses` |
| **MCP tool names** | `snake_case` (`subject_action`) | `note_search`, `card_create` |
| **MCP tool arguments and answer keys** | `camelCase` | `pathPrefix`, `taskId` |
| **Directories** | `kebab-case` (Go package directories: `lowercase` with no separator) | `status-corner/`, `command-palette/`, `appstate/` |
| **Go files** | `snake_case` | `count_reviews.go`, `one_schema_test.go` |
| **TypeScript / JavaScript files** | `camelCase` | `useWorkspaceTabs.ts`, `useFileRouter.ts`, `transport.ts` |
| **Vue components** | `PascalCase` | `App.vue`, `NoteTab.vue`, `Field.vue` |
| **SQL tables and columns** | `snake_case` | `open_tabs`, `source_id`, `folded_name` |
| **Environment variables** | `UPPER_SNAKE` | `NUMEN_MODELS_DIR`, `NUMEN_PLUGIN_DATA_DIR` |
| **CLI flags, CSS classes and variables, URL paths** | `kebab-case` | `--output-format`, `--numen-text-color` |
| **Plugin ids** | reverse-domain with dots | `md.numen.engines.embeddings`, `md.numen.views.flashcards-studio` |

## 13. Absolute ban on narrative slop, participles, and Anglish metaphors (all languages)

To keep code readable, pseudo-literary prose, Anglish idioms, irregular past participles as identifiers, and bare prepositions are strictly forbidden across TypeScript, Vue, and Go.

### Forbidden variable & identifier patterns:

1. **Irregular past participles as identifiers**: `held`, `said`, `asked`, `drawn`, `done`, `written`, `kept`, `shown`, `seen`, `told`, `settled`, `cut`, `put`, `taken`, `left`, `answered`.
   - _Fix_: Use standard engineering nouns: `instance`, `wrapper`, `binding`, `response`, `query`, `request`, `result`, `count`, `target`, `item`.
2. **Bare prepositions as identifiers**: `beside`, `under`, `over`, `behind`, `along`, `near`, `around`, `across`.
   - _Fix_: Use concrete nouns: `adjacentPath`, `sidePane`, `parent`, `delta`, `range`.
3. **Literary metaphors & bare adjectives**: `bare`, `wide`, `stale`, `fresh`, `plain`, `shut`, `loose`, `quiet`, `warm`, `unseen`, `cold`, `minting`.
   - _Fix_: Use descriptive identifiers: `rawText`, `isExpanded`, `isAvailable`, `isClosed`.
4. **Third-person singular verbs as function names**: `carries`, `attends`, `reads`, `names`, `answers`, `stands`.
   - _Fix_: Use imperative verb phrases: `carry`, `attend`, `readText`, `getName`, `respond`.
5. **Fake turn casts**: Never write `voice: 'said'`, `state: 'done'`, or `as unknown as Turn`. Adhere strictly to the domain types `voice: 'asked' | 'answered'` and `TurnState: 'settled' | 'arriving' | 'failed'`.
6. **Forbidden error terminology**: Never use `refused` or `refusal`. Always use `error` or `errorCode`.
7. **Narrative comments & docstrings**: Do not write biblical or Saxon idioms (`hands back`, `comes to`, `stands on`, `where it sits`, `what holds here`). Write plain, technical engineering English.
8. **Personification and metaphor in prose a person reads**: A task title, notice, error, manifest `title` or `description`, Markdown document or test title states the operation or the fact in standard engineering terms. A window does not `wear` a theme, a plugin does not `speak` a protocol, a page does not `hand over` a value; a task is titled `Apply theme`, not `Wearing a theme`, and its detail reads `Theme not found: dracula`. `tools/checks/style/names/check.mjs` rejects these families (`wear`/`dress`, `hand over`/`hand back`, `speaks`, `breathe`/`eat`, `lit up`, `bring level`, `stands`/`sits` at a place, `nowhere to put`) in Markdown, comments, test titles, Go and TypeScript string literals and `manifest.json`; words inside backticks are quoted, not checked.
9. **Test titles and comments**: name the subject, a standard verb and concrete nouns. `tools/checks/style/names/check.mjs` category 24 checks the lines a change adds. Examples:
   - `it('releases pointer capture when the scope is disposed')`
   - `it('clears the interval on dispose')`
   - `it('keeps its position when the list has one item')`
   - `it('cancels the drag when no listener is registered')`
   - `it('reports an error when the core rejects the tab report')`
   - `/** Converts a viewport point to screen coordinates. */`

---

## 14. Known gotchas

- **Every worktree shares one `git stash` stack.** Two agents stashing at once
  mix up the entries: one `pop` takes the other's entry into the wrong tree. For a clean
  tree, commit first and `git checkout HEAD~1 -- <path>`, or copy the file
  aside.
- **The windows import `@numen/ui` from its build.** Run `npm run build` in
  `libs/ui/components`, then in `libs/ui/workbench` (which imports `@numen/ui` from its build too), before the suites in `apps/desktop/editor` and
  `flashcards`, or they fail on `Cannot find module '@numen/ui'` (or `'@numen/workbench'`).
- **`node_modules` symlinked from the primary checkout resolves
  `@numen/protocol` to whatever branch that checkout has checked out.** A
  generated symbol missing there is that, not the branch under test. Install
  for real when the answer matters.
- **`npm test` in `libs/ui/components` starts both story instances at once**, and
  the run fails before any test executes. Take them one at a time:
  `npm run test:unit`,
  `vitest run --project 'stories (chromium)'`,
  `vitest run --project 'stories (webkit)'`. WebKit does run on this machine.

---

## 15. Rules for agents learned from review

The plugin model is [`.agents/rules/architecture.md`](.agents/rules/architecture.md) §11. The core is a microkernel (§11 (l)): the core, the core protocol and the shared SDK know no plugin domain, connection operations are opaque, their shapes are SDK contracts that only plugins and applications import, the core owns all index storage, and agents are providers of `acp.task`.

The plugin model, the SDK rules and the import rules for windows, plugins and libraries are [`.agents/rules/architecture.md`](.agents/rules/architecture.md) §11 (a) to (l); placement within a plugin is [`nm-place-plugin`](.agents/skills/nm-place-plugin/SKILL.md) and placement of shared code is [`nm-place-library`](.agents/skills/nm-place-library/SKILL.md). Time is a port (§5, §11 (i)). Size and complexity limits are one set for Go, TypeScript and Vue (§8). The numbered rules continue from 10.

10. Checks are structural. Use an off-the-shelf tool with configuration where one exists (dependency-cruiser, depguard and gomodguard, golangci-lint linters, stylelint). Write a check by hand only over syntax trees, import graphs and type information. A check never decides by a regular expression over source text, a list of names or a list of files ([`.agents/rules/architecture.md`](.agents/rules/architecture.md) §10).
11. Checks never get weaker: no exclusion and no workaround that bypasses a rule (such as obtaining the time through `time.After(0)`). Derive sets from structure.
12. Tell the owner a design decision before writing it into a plan file or implementing it. Do not present a guess as agreed.
13. Do the work first. Run the heavy checks (`make checks` in full, tests across modules, linters) once at the end; while working run `make checks CHANGED=1`.
14. Pull requests, commits and issues are written as [`.agents/rules/writing-changes.md`](.agents/rules/writing-changes.md) says.
15. Write comments and docs in plain technical English (§1, §13).
16. A failing check is fixed in the code it reports. The check stays as it is, and the code changes until it passes:
    - A type is named from the package that declares it, not from an alias package that re-exports it.
    - An interface stays within the size limit with all its methods, embedded ones included.
    - Every file and every package is linted and typechecked; a configuration excludes only build output. The duplication check reads code only: declarations (`*.yml`, `go.mod`, `Makefile`, plugin `manifest.json`) repeat by nature and are out of its scope.
    - An exception list stays as it is; the code it would exempt is changed.
    - Logic stays in the folder its layer gives it, where the checks walk.
17. Every rule that a check enforces is stated in [`.agents/rules/architecture.md`](.agents/rules/architecture.md) with the check's name; a rule with no check is a finding for the gatekeeper.
18. After every wave of packages in a refactor, run an independent fresh-eyes review before the next wave: a reviewer who did not write the wave reads the tree and the checks, reports findings, and changes nothing. The findings are fixed before the next wave starts.
