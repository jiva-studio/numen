# Specification: Reconcile Open Tab When File Kind Changes

**Task Slug:** `fix-156-file-kind-tab-reconcile`  
**Date:** `2026-10-08`  
**Intent:** [.agents/tasks/fix-156-file-kind-tab-reconcile/intent.md](intent.md)

## 1. Target Architecture & Interfaces

### File Openers Dependency Contract (`entities/tab/model/openers.ts`)
```ts
export interface FileOpenerDeps {
  fileKinds(paths: readonly string[]): Promise<ReadonlyMap<string, FileKind>>
  reconcileTab?(path: string, newKind: string): void
}
```

### Window Tabs Reconciler Contract (`entities/tab/model/windowTabs.ts`)
```ts
export interface WindowTabReconciler {
  reconcileTab(path: string, newKind: string): void
}
```
`useWindowTabs` provides `reconcileTab(path: string, newKind: string): void`:
- Iterates over all currently open tabs in `open.value`.
- For each open tab, resolves its path using `getOpenTab(state)?.path` or `getTarget(state)?.path` / `getTarget(state)?.file`.
- If the tab's path matches `path` and its kind does not match `newKind` (and the tab is not a non-file workspace tool kind such as `plex`, `agent`, `files`, or `settings`), closes the tab using `requestClose(id)`.

### Uniform `getOpenTab` and `getTarget` on File Tab Kinds
- `pages/deck-editor/kind.ts`: implements `getOpenTab` and `getTarget` returning the deck file path.
- `pages/stencil-editor/kind.ts`: implements `getOpenTab` and `getTarget` returning the stencil file path.
- `pages/preset-editor/kind.ts`: implements `getOpenTab` and `getTarget` returning the preset file path.

### Wiring in `app/useWindow.ts`
- Connects `held.reconcileTab` into `fileOpeners({ ...core, reconcileTab: held.reconcileTab })`.

## 2. Blast Radius Matrix
| Package / Path | File | Action | Downstream Consumers |
| :--- | :--- | :--- | :--- |
| `modules/apps/desktop/editor` | `src/entities/tab/model/openers.ts` | Modify | `src/app/useWindow.ts`, `src/entities/tab/index.ts` |
| `modules/apps/desktop/editor` | `src/entities/tab/model/windowTabs.ts` | Modify | `src/app/useWindow.ts`, `src/entities/tab/index.ts` |
| `modules/apps/desktop/editor` | `src/pages/deck-editor/kind.ts` | Modify | `src/pages/deck-editor/model/useDeckTabs.ts` |
| `modules/apps/desktop/editor` | `src/pages/stencil-editor/kind.ts` | Modify | `src/pages/stencil-editor/index.ts` |
| `modules/apps/desktop/editor` | `src/pages/preset-editor/kind.ts` | Modify | `src/pages/preset-editor/index.ts` |
| `modules/apps/desktop/editor` | `src/app/useWindow.ts` | Modify | Desktop Window App bootstrap |
| `modules/apps/desktop/editor` | `src/entities/tab/model/openers.test.ts` | Modify | Unit test suite |
| `modules/apps/desktop/editor` | `src/entities/tab/model/windowTabs.test.ts` | Modify | Unit test suite |

## 3. Negative Invariants (Architectural Prohibitions)
- No God Objects: Keep tab management decoupled; `openers.ts` asks for tab reconciliation via dependency injection without importing concrete window stores.
- Preserved Boundaries: Workspace utility panels (`plex`, `agent`, `files`, `settings`) must never be closed or altered by file tab reconciliation.
- Single Responsibility: `reconcileTab` only handles closing tabs of mismatching kinds for the same path; it does not dictate how individual kinds serialize or render.
- No Narrative Comments: Comments must state the rule directly and succinctly without conversational explanations.

## 4. Acceptance Criteria

### AC-1: Tab reconciled when file changes from note to deck
- **Given**: A file `Animals.md` is currently open in a `note` tab.
- **When**: `openFile('Animals.md')` is called after `Animals.md` kind is updated to `deck`.
- **Then**: The previous `note` tab for `Animals.md` is closed, and only a single `deck` tab remains open for `Animals.md`.

### AC-2: Tab reconciled when file changes from deck to stencil
- **Given**: A file `Vocabulary.md` is currently open in a `deck` tab.
- **When**: `openFile('Vocabulary.md')` is called after `Vocabulary.md` kind is updated to `stencil`.
- **Then**: The previous `deck` tab for `Vocabulary.md` is closed, and only a single `stencil` tab remains open for `Vocabulary.md`.

### AC-3: Tab unchanged when file kind is identical on reopen
- **Given**: A file `Entropy.md` is currently open in a `note` tab.
- **When**: `openFile('Entropy.md')` is called while its kind remains `note`.
- **Then**: The existing `note` tab is focused and no tab is closed or duplicated.

### AC-4: Non-file workspace tool panels are not affected by reconciliation
- **Given**: A `plex` tab is open and inspecting `Animals.md`, alongside a `note` tab for `Animals.md`.
- **When**: `openFile('Animals.md')` is called with kind `deck`.
- **Then**: The `note` tab is closed and replaced with `deck`, while the `plex` tab remains open and unaffected.
