# Vue 3 review of the refactoring plan

Reviewed:

- `.claude/worktrees/book/review.md`
- `.claude/worktrees/book/refactoring-plan.md`
- implementation under `modules/apps/desktop/editor/src`

## Decision

Keep the terminology cleanup and the decomposition work. Replace the naming
section with the concrete renames below. Do not present every local convention
as a Vue requirement: Vue documents `use...` for composables, camelCase props
in script, kebab-case props and event listeners in templates, and PascalCase
multi-word SFC components. It does not require the four script sections or
`on...` handler names; those remain useful repository rules from `AGENTS.md`.

Official references:

- [Vue composables](https://vuejs.org/guide/reusability/composables)
- [Vue props](https://vuejs.org/guide/components/props)
- [Vue component events](https://vuejs.org/guide/components/events)
- [Vue style guide](https://vuejs.org/style-guide/rules-strongly-recommended.html)

## Required corrections to the plan

### 1. Rename every reactive state factory, not only `booking` and `documenting`

Vue calls a function that encapsulates and returns reactive state a composable;
its documented convention is a camelCase `use...` name. The current plan misses
the two factories that create the reader state itself.

| Location | Current | Proposed |
| --- | --- | --- |
| `book-tab/kind.ts:78` | `booking` | `useBookTab` |
| `book-tab/open.ts:168` | `openBook` | `useBookReader` |
| `document-tab/kind.ts:59` | `documenting` | `useDocumentTab` |
| `document-tab/open.ts:79` | `openDocument` | `useDocumentReader` |
| `agent-tab/kind.ts:55` | `talking` | `useAgentConversation` |
| `files-tab/kind.ts:136` | `filing` | `useFilesTab` |

Update derived state aliases with the factory: `OpenBookState` to
`BookReaderState`, `OpenDocumentState` to `DocumentReaderState`, and the
`ReturnType` references for the renamed factories. Do not call a plain,
stateless converter `use...`; it would falsely imply Composition API state.

### 2. Apply the event-handler convention at the component boundary

`on<Action>` is a repository convention, not an official Vue requirement, but
it makes template callbacks discoverable. `FilesTab.vue` currently mixes
ambiguous local names with inline handlers. Move each callback to script and
use these names:

| Current binding or local | Proposed handler |
| --- | --- |
| `over` | `onDragOver` |
| `dropped` | `onDrop` |
| `@open` | `onOpenEntry` |
| `@close` | `onCloseEntry` |
| `@select` | `onSelectEntries` |
| `@activate` | `onActivateEntry` |
| `@rename` | `onRenameEntry` |
| `@move` | `onMoveEntries` |
| `@drag` | `onDragEntries` |
| `@drop` | `onDropEntries` |
| `@remove` | `onRemoveEntries` |
| `@menu` | `onOpenMenu` |
| `@choose` | `onChooseMenuItem` |
| `@dismiss` | `onDismissMenu` |

Apply the same extraction to `DeckTab.vue`: `asks` to `onOpenScheduleMenu`,
`chose` to `onChooseSchedule`, and create `onAddCard`, `onRemoveCard`,
`onMoveCard`, `onWriteCard`, `onAddSection`, `onRenameSection`,
`onRemoveSection`, and `onDismissScheduleMenu`. Keep emitted event names in
kebab-case in templates, such as `@add-section`; Vue recommends kebab-case for
event listeners there.

### 3. Rename state-object methods by their operation

These public methods are part of each state object and should be imperative or
predicate phrases under the repository convention. Include all call sites and
tests in the same change.

| Location | Current | Proposed |
| --- | --- | --- |
| `agent-tab/kind.ts` | `writing` | `setQuestion` |
|  | `opensTurn` | `openTurnSource` |
|  | `followed` | `followLink` |
| `book-tab/kind.ts` | `holdsBook` | `setBookHandle` |
|  | `holdsTab` | `setTabElement` |
|  | `takes` | `focusTab` |
|  | `pressed` | `handleKeyPress` |
| `document-tab/kind.ts` | `drew` | `setPageHandle` |
| `files-tab/kind.ts` | `renames` | `setRenamingPath` |
|  | `asks` | `openMenu` |
|  | `chose` | `chooseMenuItem` |
| `flashcards-deck-tab/deckTabs.ts` | `decking` | `useDeckTabs` |

Also remove redundant aliases such as `asks: askQuestion` in `agentKind` and
`called: getTitle` beside `getTitle`. A public API should have one name per
operation.

### 4. Correct the proposed component split

`DeckTab.vue` is 195 lines including styles and has a small, cohesive template.
The plan's `DeckCardRow.vue` is not evidenced by the file: no card-row
implementation exists there; `DeckEditor` owns the grid. Extracting it would
create a wrapper without an independent responsibility. Keep `DeckTab.vue`
whole, but apply the script sections and handler extraction above.

`FilesTab.vue` has enough template adaptation and drag/drop behavior to justify
extraction. Keep it as a view component and move tree projection, external URL
drop handling, and menu-item derivation into `useFilesTabView` only if that
logic is used by another component; otherwise private helpers in the component
are clearer than a one-use composable.

### 5. Decompose by a real responsibility

`book-tab/open.ts` (327 lines) has two separable units: pure book navigation
(`pageAt`, `documentAt`, `contentsOf`, and book shape types) and reactive
loading/rendering. Move pure types and functions to `book-tab/book.ts`; retain
the async reactive state in `useBookReader.ts`. Preserve the domain folder.

`document-tab/open.ts` is 219 lines. Extract pure page/highlight mapping only
when it becomes independently testable. Its main problem is naming, not a
demonstrated god object. Do not split merely to meet an arbitrary line count.

### 6. Do not make a protocol-breaking rename look mechanical

`Refusal` and `refusal` occur in public protobuf schemas. Renaming them to
`ErrorCode` and `error` changes generated API names for every client. Make this
a versioned protocol migration: inventory generated consumers, update schemas
and adapters together, regenerate bindings, and run compatibility checks. The
frontend `Result<T, ErrorCode>` shape is appropriate at the UI boundary, but it
does not by itself prescribe the Go domain-error representation.

### 7. Expand the audit with the omissions already visible in scope

The plan says repository-wide but lists only three gerund factories. Include at
least `listing` to `useFileTree`, `decking` to `useDeckTabs`, and the other
stateful factories found by the audit. Exclude test-local fixture helpers unless
they are production conventions. Audit exported/public state APIs first, then
private helpers whose names obscure an operation.

## Vue naming rules to record in the plan

1. Stateful Composition API factories use `useFeature` in camelCase and return
   a plain object of refs and actions.
2. SFC component files, imports, and tags use a consistent PascalCase style;
   user components have multi-word names.
3. Props are camelCase in TypeScript declarations. Use one consistent template
   style; this project can use kebab-case for component props and events.
4. Declare component emits with `defineEmits`; listen in the parent with
   kebab-case events. `onAction` is reserved for the local function receiving
   the event.
5. Keep views thin. Extract stateful reusable logic into composables and
   independently meaningful visual regions into child components.

## Suggested implementation order

1. Commit the Vue convention and concrete rename map.
2. Rename one domain at a time, updating implementation, template, and tests
   atomically.
3. Split `book-tab/open.ts` after its names stabilize.
4. Perform the protobuf error migration as a separately versioned change.
5. Add lint checks only for conventions that can be recognized reliably; keep
   semantic decisions such as component extraction in review.

## Repository-wide audit addendum

The first review covered the files named in `review.md`. This addendum covers
all 508 production `.ts` and `.vue` files under `modules/`, excluding tests,
stories, and fixtures from the actionable rename list. Generated protobuf
TypeScript is included only to measure protocol blast radius; change it only by
editing `.proto` sources and regenerating it.

### A. Stateful factories outside the original scope

These functions use Vue reactivity and return state plus operations. Apply the
repository `use...` convention to all of them, grouped into small commits by
domain. The proposed names describe the state they own.

| Location | Current | Proposed |
| --- | --- | --- |
| `libs/ui/src/shared/lib/stream.ts` | `following` | `useStream` |
| `libs/ui/src/features/thread/conversation.ts` | `conversation` | `useConversation` |
| `apps/desktop/flashcards/src/notices.ts` | `raising` | `useNotices` |
| `apps/desktop/flashcards/src/counting.ts` | `counting` | `useReviewCounter` |
| `apps/desktop/flashcards/src/decks/presets.ts` | `vaultPresets` | `useVaultPresets` |
| `apps/desktop/flashcards/src/session/notes/core.ts` | `reviewed` | `useReviewedNotes` |
| `apps/desktop/flashcards/src/session/NotesPanel.vue` | `notesPanel` | `useNotesPanel` |
| `apps/desktop/flashcards/src/session/panel.ts` | `session` | `useReviewSession` |
| `apps/desktop/editor/src/plex-tab/view.ts` | `view` | `usePlexView` |
| `apps/desktop/flashcards/src/session/panel.ts` | `agentPanel` | `useAgentPanel` |
| `apps/desktop/editor/src/plex-tab/kind.ts` | `plexing` | `usePlexTab` |
| `apps/desktop/editor/src/plex-tab/tickets.ts` | `ticketing` | `useTickets` |
| `apps/desktop/editor/src/files-tab/listing.ts` | `listing` | `useFileTree` |
| `apps/desktop/editor/src/flashcards-deck-tab/scheduler.ts` | `scheduler` | `useDeckSchedule` |
| `apps/desktop/editor/src/flashcards-deck-tab/deckTabs.ts` | `decking` | `useDeckTabs` |
| `apps/desktop/editor/src/flashcards-preset-tab/kind.ts` | `presetting` | `usePresetTab` |
| `apps/desktop/editor/src/flashcards-stencil-tab/stencilTabs.ts` | `stencilling` | `useStencilTabs` |
| `apps/desktop/editor/src/note-tab/kind.ts` | `noting` | `useNoteTab` |
| `apps/desktop/editor/src/note-tab/drawing.ts` | `drawing` | `useNoteDrawing` |
| `apps/desktop/editor/src/settings-file-tab/kind.ts` | `holding` | `useSettingsFileTab` |
| `apps/desktop/editor/src/shared/settings/hanging.ts` | `hanging` | `useHangingSetting` |
| `apps/desktop/editor/src/shared/saving/flushing.ts` | `flushing` | `useFileFlush` |
| `apps/desktop/editor/src/window/showing.ts` | `showing` | `useWindowShowing` |
| `apps/desktop/editor/src/shared/command/palette.ts` | `commandPalette` | `useCommandPalette` |
| `apps/desktop/editor/src/shared/command/search.ts` | `search` | `useSearch` |
| `apps/desktop/editor/src/shared/media/transcript.ts` | `transcript` | `useTranscript` |
| `apps/desktop/editor/src/shared/tabs/windowTabs.ts` | `windowTabs` | `useWindowTabs` |

`book-tab`, `document-tab`, `agent-tab`, and `files-tab` retain the six
renames in the earlier table. For functions that are deliberately independent
of a component instance, this is a repository naming rule, not an assertion
that they use lifecycle APIs.

### B. Public APIs with opaque or narrative operation names

Prioritize the exported/public methods first; they propagate ambiguity through
templates, tests, and domain boundaries.

| Location | Current | Proposed |
| --- | --- | --- |
| `apps/desktop/editor/src/plex-tab/kind.ts` | `turns` | `openPlex` |
| `apps/desktop/editor/src/files-tab/kind.ts` | `reveals` | `revealPath` |
|  | `changed` | `refreshChangedPaths` |
| `apps/desktop/editor/src/note-tab/notes.ts` | `openNotes` | `openNotesForPath` |
| `apps/desktop/editor/src/note-tab/kind.ts` | `attends` | `getAttention` |
| `apps/desktop/editor/src/settings-file-tab/kind.ts` | `editingSettingsFile` | `createSettingsFileTabKind` |
| `apps/desktop/editor/src/shared/saving/conflicts.ts` | `raisesConflicts` | `raiseConflicts` |
| `apps/desktop/editor/src/shared/media/transcript.ts` | `transcribed` | `useTranscriptTab` |
| `apps/desktop/editor/src/shared/media/player.ts` | `playable` | `createMediaTypeProbe` |
| `apps/desktop/editor/src/shared/command/palette.ts` | `shows` | `setOpen` |
| `apps/desktop/editor/src/shared/command/search.ts` | `shows` | `setOpen` |

Do not perform blind suffix stripping. For example, `pageAt`, `documentAt`,
and `contentsOf` are pure selectors, so the existing names are acceptable;
they are not composables. Review each candidate's responsibility before
renaming it.

### C. Terminology migration reaches multiple applications

The current plan is too narrow. The following are production-code occurrences
by source-file count, after excluding tests, stories, fixtures, and generated
bindings:

| Deprecated term | Files | Migration boundary |
| --- | ---: | --- |
| `refusal` / `RefusalReason` | 54 / 15 | protocol schemas, wire adapters, editor, desktop flashcards, mobile, and UI |
| `stretch` / `Stretch` | 25 | protocol schema plus editor, UI book/thread/editor, and search/openers |
| `address` / `Address` | 66 | split only link-destination meanings; retain standard network/DOM address meanings |
| `overtaken` | 16 | editor save/conflict state and dependent UI wording |
| `spoken` | 9 | media/transcript code and UI labels |
| `reading` / `readings` | 81 / 4 | migrate only OCR extraction meanings; retain ordinary reading/book meanings |

The counts are an inventory, not a search-and-replace instruction. In
particular, `address` is legitimate for a browser URL or DOM address and
`reading` is legitimate for a person reading a book. Rename only the glossary
meaning: a Numen link destination becomes `link`, OCR output becomes
`recognition` or `recognizedText`, and an external-disk conflict becomes
`stale`.

Concrete high-impact migration roots:

- `modules/libs/protocol/proto/numen/v1/shared.proto`: `Refusal` to
  `ErrorCode`; regenerate TS and Go.
- `apps/desktop/editor/src/shared/core.ts`, `shared/file.ts`, `shared/note.ts`,
  and `shared/answers.ts`: replace `RefusalReason` in public frontend ports.
- `apps/desktop/editor/src/note-tab/tab.ts`: `overtaken` to `isStale`.
- `apps/desktop/editor/src/shared/command/search.ts` and
  `shared/tabs/openers.ts`: replace `stretch` only where it represents a text
  range with `span: { from, to }`.
- `apps/desktop/editor/src/shared/media/transcript.ts` and `shared/media/cues.ts`:
  replace transcript-domain `spoken` names with `transcript` names.

### D. Boolean fields outside the original review

The editor contains many boolean properties that violate the repository's
`is...` / `has...` / `can...` contract. Rename the public data fields and keep
the conversion at protocol edges:

| Location | Current | Proposed |
| --- | --- | --- |
| `note-tab/tab.ts` | `owed`, `gone`, `overtaken` | `isPending`, `isDeleted`, `isStale` |
| `shared/core.ts` | `ready`, `failed`, `unwatched`, `embedding`, `reload` | `isReady`, `failureReason`, `unwatchedPath`, `isEmbedding`, `shouldReload` |
| `shared/note.ts` | `done`, `mutual`, `frontmatter`, `changed` | `isComplete`, `isMutual`, `hasFrontmatter`, `hasChanged` |
| `shared/settings/theme.ts` | `shipped`, `pinned` | `isBuiltIn`, `isPinned` |
| `shared/media/transcript.ts` | `editable` | `isEditable` |
| `flashcards-preset-tab/kind.ts` | `writing`, `wanted`, `told`, `drawing`, `drawAgain`, `real` | `isWriting`, `isSelected`, `hasMessage`, `isDrawing`, `shouldDrawAgain`, `isReal` |
| `flashcards-preset-tab/core.ts` | `evenLoad`, `changed`, `enough`, `honest` | `hasEvenLoad`, `hasChanged`, `isSufficient`, `isValid` |

### E. Vue component work must be prioritized, not applied mechanically

The four-section script layout is absent from most SFCs because it is a new
repository rule. Add it while a component is otherwise being changed; avoid a
repository-wide formatting-only commit. The highest-value extraction candidates
are `CurveSlider.vue` (673 lines), `SettingsTab.vue` (539), `Palette.vue`
(524), `Tree.vue` (499), `PlexNodeView.vue` (498), and `DeckEditor.vue` (323).
Each needs a separate responsibility review before extraction. File length
alone does not prove a child component boundary.

### F. Revised implementation sequence

1. Add lint rules for cheap, unambiguous violations: `use...` state factories,
   `on...` local event handlers, boolean field prefixes, and `make...` TS
   factories.
2. Rename UI/editor composables by domain, updating their tests and imports in
   one commit per domain.
3. Migrate booleans and glossary terms at their public boundaries.
4. Version and regenerate the protobuf `ErrorCode` change.
5. Split only the large SFCs whose state, visual layout, or tests establish a
   clear independent responsibility.

### G. Go and protocol audit

The repository also contains 519 production Go files. This is outside Vue
conventions but inside the supplied repository naming and glossary rules, so it
must be a workstream of the same refactor. Go source has `Refusal` in 23 files,
`refusal` in 21, `stretch` in 35, `Address` in 20, `address` in 84, `spoken` in
14, and `reading` in 143. As above, classify ordinary English uses before
renaming.

The Go function-name scan finds widespread third-person and participle names.
Start at public interfaces and constructors, then work inward. Representative
high-value corrections are:

| Location | Current | Proposed |
| --- | --- | --- |
| `libs/core/container/opening.go` | `Reading()` | `IsReading()` |
|  | `Refreshing()` | `GetRefresh()` |
|  | `Scanning()` | `GetScan()` |
| `libs/core/container/notes.go` | `Following(...)` | `WithFollowing(...)` |
|  | `Drawing(...)` | `WithDrawing(...)` |
| `libs/core/adapter/agent/config.go` | `Serving()` | `IsServing()` |
| `libs/core/flashcards/review/day.go` | `Ends(...)`, `Opens(...)` | `GetEnd(...)`, `GetStart(...)` |
| `apps/desktop/cmd/numen/refusing.go` | `refusal` type and helpers | `errorCode` / `createErrorCode` |

Do not rename Go methods that implement an external interface until the
interface is migrated in the same commit. Run `go test ./...` for every Go or
protobuf migration; the TypeScript/Vue suite cannot validate those boundaries.

### H. Systematic modeling and architectural patterns from `review.md`

The review comments in `review.md` identify five systemic design flaws spanning across all desktop tab implementations:

#### 1. Redundant / Over-fragmented wrapper types
Single-use artificial wrapper types (`HighlightedPage`, `PendingWrite`, `NoteBaseline`) clutter the type system.
- `document-tab/open.ts:22`: `HighlightedPage` `{ page: number, rects: Rect[] }` $\rightarrow$ represent highlights as `pageHighlights: Map<number, readonly Rect[]>` or optional `rects` on layout pages.
- `note-tab/tab.ts:51`: `PendingWrite` `{ readonly body: string }` $\rightarrow$ eliminate single-field wrapper; use string or clean write request.
- `note-tab/tab.ts:45`: `NoteBaseline` `{ prose: string, at: FilePath }` $\rightarrow$ simplify to `{ content: string, path: string }`.
- `book-tab/open.ts:22, 39`: `SpineDocument`, `PrintedPage` $\rightarrow$ flatten and clean up fields.
- `plex-tab/kind.ts:35` and `files-tab/kind.ts:28`: duplicate `MenuRequest` $\rightarrow$ unify or inline coordinates.

#### 2. Cryptic & literary domain types and fields
Non-standard or literary terms violate Rules 2, 6, and 9:
- `document-tab/open.ts:44`: `interface Shape { pages, at }` $\rightarrow$ `DocumentLayout` with `path: string` instead of `at`.
- `book-tab/open.ts:66`: `Book.at: string` $\rightarrow$ `Book.path: string`.
- `book-tab/open.ts:33`: `BookPart.at: number` $\rightarrow$ `BookPart.offset: number`.
- `note-tab/tab.ts:47, 65`: `at: FilePath` $\rightarrow$ `path: string`.
- `note-tab/tab.ts:68`: `flight: PendingWrite | null` $\rightarrow$ `pendingWrite`.
- `note-tab/tab.ts:70`: `owed: boolean` $\rightarrow$ `hasPendingWrite`.
- `note-tab/tab.ts:75`: `gone: boolean` $\rightarrow$ `isDeleted`.
- `note-tab/tab.ts:77`: `overtaken: boolean` $\rightarrow$ `isStale`.
- `flashcards-preset-tab/kind.ts:99, 101, 117`: `material` $\rightarrow$ `counts`, `place` $\rightarrow$ `sliderValue`, `saying` $\rightarrow$ `errorMessage`.

#### 3. God composables with bloated return surfaces & mixed naming
Composables returning 15–20 mixed properties violate Rule 11 (Single Responsibility) and Rules 2/3/6 (imperative verbs, predicate booleans, on* handlers):
- `document-tab/open.ts`: `useDocumentReader` returns 18 properties with mixed semantics (`highlighted` vs `highlightedOn` vs `highlight` vs `also` vs `alsoOn`).
  - Decompose into focused composables:
    - `useDocumentNavigation` (`currentPage`, `pageCount`, `goToPage`, `nextPage`, `previousPage`).
    - `useDocumentViewport` (`pageImageUrl`, `setViewportWidth`).
    - `useDocumentHighlights` (`primaryHighlights`, `secondaryHighlights`, `setHighlights`, `getHighlightsForPage`).
- `book-tab/open.ts`: `useBookReader` returns 18 properties (`elsewhere`, `reading`, `drawn`, `at`, `go`, `reach`, `follow`). Decompose into navigation, spine loading, and highlights.
- `NoteTabState`, `PresetTabState`, `StencilTabState`, `DeckTabState`: replace 3rd-person singular and past participle methods (`moves`, `adds`, `settles`, `shuts`, `typed`, `drew`) with standard imperative verbs (`move`, `add`, `save`, `close`, `setBody`).

#### 4. Type dumps in implementation files (`kind.ts`, `open.ts`)
Domain ports, DTOs, and component state interfaces are dumped into implementation files alongside runtime code.
- Extract domain interfaces and contracts into dedicated `types.ts` per tab domain (`book-tab/types.ts`, `document-tab/types.ts`, `note-tab/types.ts`, `flashcards-preset-tab/types.ts`).

#### 5. Narrative / philosophical comments
Comments containing literary narration and justification violate Rule 1 ("Comments state the rule, and stop").
- Remove narrative essays from headers in `AgentTab.vue`, `agent-tab/kind.ts`, `book-tab/open.ts`, `document-tab/open.ts`, and flashcard tabs.

