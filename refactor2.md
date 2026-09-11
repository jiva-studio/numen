# Second-pass refactoring plan

Analysis of the codebase after the two commits on `feat/a-book-made-for-a-screen-is-read`
(`89709a65` and `b5d86cb7`). Everything in round one has been done. This file
records what remains.

---

## Issue 1 — Past-participle and third-person function names (Rule 2)

### `flashcards-deck-tab/mutations.ts`

All exported functions are past participles. The rule requires imperative verb
phrases.

| Current | Fix |
|---|---|
| `added` | `addCard` |
| `removed` | `removeCard` |
| `dropped` | `dropCard` |
| `filled` | `fillCard` |
| `sectionAdded` | `addSection` |
| `sectionNamed` | `renameSection` |
| `sectionGone` | `removeSection` |

### `flashcards-deck-tab/drawn.ts`

| Current | Fix |
|---|---|
| `headed` | `applyHead` |
| `named` | `applyName` |

### `flashcards-preset-tab/curve.ts`

| Current | Fix |
|---|---|
| `held` | `clamp` |
| `clearing` | `clearBacklog` |
| `producing` | `produceSchedule` |
| `steers` (3rd-person) | `steer` |

### `flashcards-preset-tab/plot.ts`

| Current | Fix |
|---|---|
| `naming` | `positionLabel` |
| `walked` | `walkGrid` |

### `shared/media/cues.ts`

| Current | Fix |
|---|---|
| `spoken` | `getText` |
| `spanning` | `spanCues` |
| `cued` | `applyCues` |

### `window/vault/clients.ts`

| Current | Fix |
|---|---|
| `finding` (SearchService client) | `search` |

### `window/editing.ts`

| Current | Fix |
|---|---|
| `openEditing` | `useEditing` (returns reactive state via composables) |

---

## Issue 2 — `refusal` field names in wire adapters and domain types (Rule 9)

The glossary maps `refusal` -> `error` / `errorCode`. Remaining occurrences in
non-test, non-protocol code:

### `window/vault/words.ts`
Lines 192, 203, 253, 259: `refusal?:` fields in wire-adapter input shapes;
line 159-160: `turnedDown` reads `.refusal`.

Fix: rename the local input-shape fields to `error`; update `turnedDown` to
read `.error`. The wire shape coming from protobuf still uses `refusal` — map
it in the one place it is read.

### `window/vault/files.ts`
Lines 22, 32, 39: result objects carry both `error` and `refusal: error`.
Remove the `refusal` alias; callers that read `refusal` are in
`flashcards-deck-tab/deckTabs.ts` and `flashcards-preset-tab/core.ts` — fix
those callers.

### `window/vault/notes.ts`
Lines 48, 61: same pattern — result carries `refusal: error`. Remove alias;
fix callers.

### `flashcards-deck-tab/answers.ts`
Lines 47, 67: `readonly refusal: RefusalReason | null` — rename to `error`.
Lines 55, 75: `answer.refusal` — update to `answer.error`.
Lines 83-86: parameter name `refusal` -> `errorCode`; comment on line 83
names the term without saying the rule — delete.

### `flashcards-deck-tab/deckTabs.ts`
Lines 58, 62, 63, 77, 80, 137: reads `.refusal` from the store answer.
Fix after `answers.ts` is fixed.

### `flashcards-preset-tab/core.ts`
Lines 51, 59, 67, 78, 83: `refusal` fields in local result shapes.
Rename to `error`; update callers inside `core.ts`.

### `flashcards-deck-tab/scheduler.ts`
Line 112: reads `answer.refusal`. Fix after `deckTabs.ts` is fixed.

---

## Issue 3 — `make*` factory names in TS/Vue (Rules 7, 9)

The glossary says `make` (frontend) -> `create`. The following are exposed in
public interfaces and called from multiple places.

### `shared/flashcards/types.ts`
```
makeDeck(title, folder): Promise<MakeResult>     -> createDeck
makeStencil(title, folder, fields): ...          -> createStencil
```

### `shared/tabs/makers.ts`
```
makeURL?(address, folder): ...                   -> createUrl
```
Lines 40-46 already prefer `create*` with `make*` as fallbacks.
Remove the `make*` fallbacks once implementations are renamed.

### `shared/core.ts`
```
makeFolder(path): ...    -> createFolder
makeURL(url, folder): ... -> createUrl
```

### `window/vault/files.ts`
Line 35: `makeFolder` exported method -> `createFolder`.
Line 36: `makeURL` -> `createUrl`.

### `window/commands.ts`
Line 93: `makesFolder` -> `createFolder`.

### `window/kinds.ts`
Lines 179-183: `makeDeck`, `makeStencil`, `makeURL` calls -> `createDeck`,
`createStencil`, `createUrl`.

### `shared/flashcards/cards.ts`
Lines 23, 27: `makeDeck`, `makeStencil` function bodies -> `createDeck`,
`createStencil`.

---

## Issue 4 — Oversized files that exceed the ~200-line limit (Rule 11)

| File | Lines | What to split |
|---|---|---|
| `note-tab/notes.ts` | 329 | Extract save-queue logic into `note-tab/queue.ts` (~80 lines) and external-conflict handling into `note-tab/conflict.ts` (~60 lines). `notes.ts` keeps the open-note map and coordinator (~190 lines). |
| `shared/command/search.ts` | 429 | Extract scoring into `shared/command/score.ts` and note-lookup into `shared/command/lookup.ts`. `search.ts` keeps only the `useCommandSearch` composable. |
| `shared/command/palette.ts` | 401 | Extract step-navigation logic into `shared/command/steps.ts`. `palette.ts` wires steps, search, and commands. |
| `flashcards-stencil-tab/stencilTabs.ts` | 359 | Extract field-editing state into `stencilTabs.fields.ts` and wire-read logic into `stencilTabs.wire.ts`. |
| `flashcards-deck-tab/deckTabs.ts` | 320 | Extract timer and scheduling wiring into `deckTabs.schedule.ts`; keep `deckTabs.ts` as the tab-state map. |
| `window/showing.ts` | 285 | Extract the four `follows(...)` loops into `window/streams.ts`. `showing.ts` keeps only state declarations and `start`. |
| `window/vault/words.ts` | 269 | Split by domain: `window/vault/noteWords.ts`, `window/vault/fileWords.ts`, `window/vault/vaultWords.ts`. |
| `shared/command/handlers.ts` | 252 | Extract note-creation helpers into `shared/command/noteHandlers.ts`. |
| `shared/command/commands.ts` | 268 | Extract command-list construction into `shared/command/list.ts`. |

---

## Issue 5 — Boolean fields without `is`/`has`/`can` prefix (Rule 6)

### `shared/command/palette.ts`
- `working` (line 65) -> `isWorking`

### `window/showing.ts`
- `indexing` (line 53) -> `isIndexing`
- `holds` (line 63) -> `hasNote`
- `embedding` (line 80) -> `isEmbedding`

---

## Issue 6 — `address` field name survives in domain types (Rule 9)

- `shared/note.ts` line 101: `address?: Address` -> `link`
- `shared/command/target.ts` line 251: `readonly address: string` -> `url`
- `note-tab/types.ts` lines 32-33: `followLink(address)` param -> `url`; `follows(address)` -> `followUrl(url)`
- `note-tab/open.ts` line 40: `followLink(address)` param -> `url`
- `note-tab/NoteTab.vue` line 44: `onOpen(address)` param -> `url`
- `window/vault/words.ts` line 206: remove `address: link` backward-compat alias
- `shared/media/player.ts` lines 30, 40, 41, 44, 109: `address` field and params -> `url`

---

## Issue 7 — Comment violations (Rule 1)

- `window/showing.ts` lines 3-8: block comment argues for a placement decision
  instead of stating a rule. Delete lines 3-8; keep line 1 only.
- `flashcards-deck-tab/answers.ts` line 83: narration about the absence of
  words — delete.
- `note-tab/notes.ts` line 5: check for "rather than" / "instead of" phrasing.

---

## Issue 8 — `minting` gerund name (Rule 2)

`flashcards-deck-tab/mutations.ts` line 116 default `mint: IdMaker = minting`.
`minting` is a gerund literary metaphor.
Fix: rename the source symbol to `generateId` (wherever it is defined in `ids.ts`
or equivalent).

---

## Recommended order

1. **Issues 1 + 8** — function renames; no interface or type changes. Low risk.
2. **Issue 2** — `refusal` field cleanup. Touch wire-adapter shapes; update
   callers in deckTabs, scheduler, and preset core. Run lint and tests after each file.
3. **Issue 3** — `make*` -> `create*`. Rename interface methods; update call
   sites; remove `make*` fallback aliases.
4. **Issues 5 + 6** — boolean field prefixes and `address` -> `link`/`url`.
   Small targeted renames; update every consumer.
5. **Issue 4** — file splits. One file at a time. Each split is an
   extract-then-re-export; callers need no changes if the old module re-exports.
6. **Issue 7** — comment cleanup. Last, since it touches many lines but changes
   no behaviour.
