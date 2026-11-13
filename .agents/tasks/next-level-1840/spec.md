# Specification: Plex shows the names of the notes related to each related note

**Task Slug:** `next-level-1840`
**Date:** `2026-11-13`
**Intent:** `.agents/tasks/next-level-1840/intent.md`
**Issue:** #1840. Base: `feat/1837-plex-clusters` (PR #1844), stacked on it. Placement skills: [`nm-place-core`](../../skills/nm-place-core/SKILL.md) (proto, use case, adapter), [`nm-place-plugin`](../../skills/nm-place-plugin/SKILL.md) (plugin, SDKs, manifest), [`nm-place-window`](../../skills/nm-place-window/SKILL.md) (the editor's plugin service), [`nm-place-library`](../../skills/nm-place-library/SKILL.md) (protocol, client, UI library).

The kernel answers `ListRelatedNotes` for one step or two. The views-plex-graph plugin asks for two, orders and limits the second step into names per note, and the Plex draws them in small grey text over and under each related note, keeping room for them in the arrangement. Visuals follow the prototype (the lab folder of the plex frontend) of branch `proto/plex-clusters` (`nextGeneration.ts`, `layout.ts` `getFootprint`, `PlexLab.vue`), whose words `generation`, `level` and `via` are not carried over.

## 1. Target Architecture & Interfaces

### 1.1 Wire (`libs/core/protocol/proto/numen/v1/note.proto`; frozen first, regenerated Go and TS in the same commit)

```proto
message ListRelatedNotesRequest {
  string path = 1;
  // 1: the notes related to the note in focus. 2: also the notes related to each of those.
  // 0 reads as 1. Any other value is invalid.
  uint32 depth = 2;
}

message RelatedNote {
  …
  // The path of the related note this item belongs to, set for a note found by depth 2.
  // `relation` is then relative to that note. Empty for a note related to the focus.
  string related_to = 9;
}
```

`RelatedNote` field number 9 is free on this base. #1839 adds `RELATION_REFERENCE` to the enum and `direction = 8` to `LinkDirection`; this change touches neither.

The response keeps `focus`, `related` (first the notes related to the focus, in the order they have today, then, for depth 2, the items of each of them, in the order of the notes they belong to), `connections`, `focus_type`, `focus_tab_kind`. `connections` joins only the focus and the notes related to it, as today: an item with `related_to` set is never an end of a connection.

### 1.2 Kernel (`libs/core/kernel`, placement: nm-place-core)

- `libs/note/domain/note_graph.go`: `RelatedNote` gains `RelatedTo string` (path of the related note an item belongs to; empty for a first-step note).
- `usecase/note/query/related_notes.go`: `ListRelatedNotes.Execute(ctx, v, path string, depth domain.RelationDepth)`. The wire value goes through `domain.ParseRelationDepth`: 0 and 1 are `DepthNear`, 2 is `DepthFar`, above 2 is `ErrInvalidRelationDepth`. `DepthNear` is today's answer. Depth 2 adds, for every note related to the focus (siblings included), the notes it is related to: parents, children and jumps read from the same links and backlinks as the first step with the same `getRelation`, never siblings (a sibling is derived from a shared parent and is not written anywhere). Items are `RelatedNote{Relation: <relative to the note>, RelatedTo: <its path>}`. The focus is never an item (`AddRelatedNote` already drops it), and a note is not an item of itself. Every other note related to an owner is one of its items, also a note already related to the focus: the common parent of the focus and a reference is an item of the reference, and the other children of the parent, the siblings of the focus included, are items of the parent. A note may be an item of several owners and of one owner only once. No limit and no ordering by relation. The links of a note are read once through the existing `linkCache`; the notes are queried once after every link has been followed (`GetNotes` batch, as now). An item whose note `GetNotes` does not return is dropped, and so are the items of a first-step note it does not return.
- `GetRelatedNotesView.Execute` passes `depth` through; `Types` and `TabKinds` cover every returned path.
- `adapter/window/editor/note/notes.go`: `Outline.ListRelatedNotes` reads `r.Msg.GetDepth()`; 0 and 1 give depth 1, 2 gives depth 2, anything else returns `connect.CodeInvalidArgument`. `convertRelationToWire` stays; the item is written with `RelatedTo`.
- No other adapter changes (`apps/mobile/bind` calls the same handler).

### 1.3 SDKs and host (placement: nm-place-plugin, nm-place-window, nm-place-library)

- `libs/plugin/sdk-ts/src/workspace.ts`: `RelatedNote` gains `readonly relatedTo?: string` (path of the related note it belongs to; absent for a note related to the focus). `VaultNotes.listRelatedNotes(path: string, depth?: 1 | 2): Promise<NoteGraph>`; omitted depth is 1.
- `libs/core/client/src/protocol.ts` and the editor entity `apps/desktop/editor/src/entities/note/api/notes.ts`: `listRelatedNotes(path, depth)` sends `{ path, depth }`; `mapNoteGraph` maps `relatedTo` (empty string becomes absent).
- `apps/desktop/editor/src/features/plugin-host/model/pluginServices.ts`: passes `depth` through. The plugin loader context (`libs/plugin/loader/src/hosting/context.ts`) forwards the argument unchanged.
- Go SDK (`libs/plugin/sdk-go/host`): unchanged. It has no related-notes read today and none is added here (decision 1, separate issue).

### 1.4 Plugin (`plugins/views-plex-graph`, placement: nm-place-plugin)

- `manifest.json`: two settings after `headingLimit`:
  - `showNames`: boolean, title "Show notes related to related notes", description "Under each related note, the names of the notes it is related to.", default `true`, `palette: true`, `icon: "waypoints"`.
  - `namesPerNote`: range, title "Names per note", description "Most names shown under and over a related note.", default `6`, `min` 1, `max` 12, `step` 1.
- `frontend/model/names.ts` (create, beside `nesting.ts`): `SHOW_NAMES = true`, `NAMES_PER_NOTE = 6`, `readNames(context): { showNames: Ref<boolean>; namesPerNote: Ref<number> }`, keys read in camelCase as `readNesting` does. A value outside min/max or not a number falls back to the default.
- `frontend/model/usePlexView.ts`: `RelatedNoteQueries.listRelatedNotes(path, depth)`; the tab asks for depth 2 when `showNames` is on, depth 1 otherwise, and asks again when the setting changes. `areNoteGraphsEqual` compares `relatedTo` too.
- `frontend/lib/picture.ts` and `frontend/lib/names.ts`:
  - `convertNoteGraph` (`picture.ts`) skips every `related` item that has `relatedTo`: they are never nodes, and `getNoteTypes` / `getTabKinds` keep reading all items (a name may be drawn with its type later; no UI uses it now).
  - `getNames(noteGraph, limit, token, references): ReadonlyMap<string, PlexNames>` (`lib/names.ts`) where `PlexNames = { above: PlexName[]; below: PlexName[] }` and `PlexName = { id: string; title: string }` (`id` is `name:<path>`, made by `getNameToken` and read back by `getNamePath`, both in `lib/names.ts`; a name has an id of its own so that a click reaches its note whether or not that note is a node). For each related note: its items ordered parents, children, jumps, references (arrival order inside a relation), cut to `limit` in that order, then split: parents into `above`, the rest into `below`. A note with none has no entry.
- `frontend/types.ts`: `PlexName`, `PlexNames` (plex-side types, no note concept).

### 1.5 Plex (`plugins/views-plex-graph/frontend/plex`, placement: nm-place-plugin, nm-place-window for the shared `@numen/ui` parts)

- `lib/names.ts` (create; pure, from the prototype `nextGeneration.ts`, renamed): 
  - `getNamesRoom(names: PlexNames, request: { fontSize: number; width: number; measure: (title: string) => number }): NamesRoom` with `NamesRoom = { above: Row[]; below: Row[]; rowHeight: number; spacing: number; width: number; top: number; bottom: number }`. Names wrap into rows no wider than `request.width` (the node's `maxWidth` setting); `spacing = fontSize × 0.9`, `rowHeight = fontSize × 1.3`, `top`/`bottom` = rows × `rowHeight` + `rowHeight × 0.15` when there are rows, else 0; `width` = the widest row.
  - `placeNames(room: NamesRoom, pill: { x; y; height }): PlacedName[]`, centered on the pill, under it (`below`, row 1 nearest) and over it (`above`, the last row nearest); a `PlacedName` is `{ id; title; x; y; width; height }`. A name wider than `request.width` is cut with an ellipsis to the width (the existing title-cut helper).
- `lib/arrange/arrange.ts` / `layout/placement.ts`: `ArrangeInput.room?: (node: PlexNode) => NodeRoom` with `NodeRoom = { top: number; bottom: number; width: number }`, the extra space of a note over and under its pill and the widest row, supplied by the caller as numbers like `measure`. Absent, or all zero, nothing changes. `PlacementContext` gains `room: (node: PlexNode) => NodeRoom`.
  - The footprint of a node is `{ width: max(pill width, room.width), height: pill height + room.top + room.bottom }` plus the existing `room` setting (#1836, `applyRoom`) as `noteMargin` of the prototype. `clusters` and `findFreeSpot` place, test `isFree`, index and measure the footprint, so boxes of different heights are supported: the cell, `reach` and `getReach` use the footprint of the node (and of the first of its relation), not `options.nodeSize.height`.
  - The returned `PlacedNode` is the pill: `width` and `height` as today, `y` = footprint center + `(room.top − room.bottom)/2`. The frame, edges (they end at the pill), drag, drop and `getExtent` read pills as before.
  - Admission and spacing test the footprint (pill plus names) against the window; crowding is measured without the names. A frame that has no room reservations is byte-identical to the one of #1837.
- `ui`: `Plex.vue` takes `names?: (id: string) => PlexNames` beside `headings` and `showNames`, measures names with the existing text measuring (`model/sizing/measure.ts`) at a font size smaller than a title (`NAME_SCALE = 0.75` of the title size, a constant), builds `room` for `arrangePlex`, and emits `chooseName(id)` which `usePlexTab` handles as `activate` does for a node (move the focus to that note). Names render as `PlexNodeNames.vue` (create, `ui/render/`): plain text in `--numen-ink-faint` (or the closest existing token), each name a button with no pill, no outline, no heading list; opacity `NAME_OPACITY = 0.7` at rest, `1` while the pointer is over its note's pill or any of its names (the "brought forward"), `cursor: pointer`. Names follow the pill during the move animation (same transition as the pill). The focus size scales the focus pill only; names keep their size on every note.
- The line to a note may cross its names: no routing change; names carry `pointer-events` only on the text.
- `Plex.stories.ts`, `PlexNodeView.stories.ts`: a story with names (`WithNames`) and the `names` / `showNames` args.

## 2. Blast Radius Matrix

| Path | File | Action | Downstream consumers |
|---|---|---|---|
| `libs/core/protocol/proto/numen/v1` | `note.proto` | Modify (`depth = 2`, `related_to = 9`) | all generated code below |
| `libs/core/protocol/gen`, `libs/core/protocol/src` | `note.pb.go`, `note_pb.ts` (generated) | Regenerate | kernel, client, mobile bind |
| `libs/note/domain` | `note_graph.go`, `note_graph_test.go` | Modify (`RelatedTo`) | `usecase/note/query`, adapters |
| `libs/core/kernel/usecase/note/query` | `related_notes.go`, `outline.go`, tests | Modify (`depth`) | `adapter/window/editor/note` |
| `libs/core/kernel/adapter/window/editor/note` | `notes.go`, `calls.go` unchanged signature | Modify | editor window, `rename_test`, `changes_test`, `connections_test`, `vault_session_test` (callers of the use case/handler) |
| `apps/mobile/bind` | `mobile_test.go` | Re-run | mobile |
| `libs/core/client/src` | `protocol.ts` | Modify only if the request type is listed there | editor |
| `libs/plugin/sdk-ts/src` | `workspace.ts` | Modify (`relatedTo`, `depth?`) | loader, plugins, `libs/plugin/sdk-ts` tests |
| `libs/plugin/loader/src/hosting` | `context.ts`, `contextServices.test.ts` | Modify (forward `depth`) | all plugins |
| `apps/desktop/editor/src/entities/note/api` | `notes.ts`, `notes.test.ts`, `noteWords.test.ts` | Modify | plugin-host, `testing/mocks/note.ts` |
| `apps/desktop/editor/src/features/plugin-host/model` | `pluginServices.ts`, `pluginServices.test.ts` | Modify | plugins |
| `plugins/views-plex-graph` | `manifest.json`, `README.md` | Modify (two settings) | settings UI, `frontend/manifest.test.ts`, `frontend/index.test.ts` |
| `plugins/views-plex-graph/frontend/model` | `names.ts` (+test), `usePlexView.ts`, `usePlexTab.ts`, `plexOptions.ts` untouched | Create / Modify | `ui` |
| `plugins/views-plex-graph/frontend/lib` | `picture.ts`, `picture.test.ts`, `names.ts`, `names.test.ts` | Modify | `usePlexView`, `usePlexTab` |
| `plugins/views-plex-graph/frontend` | `types.ts`, `index.ts` | Modify | all of the plugin |
| `plugins/views-plex-graph/frontend/plex/lib` | `names.ts` (+test) | Create | `Plex.vue` |
| `plugins/views-plex-graph/frontend/plex/lib/arrange` | `arrange.ts`, `layout/placement.ts`, `layout/clusters.ts`, `layout/freeSpot.ts`, `layout/admission.ts`, `layout/crowding.ts`, `layout/spacing.ts` | Modify (footprint) | `fixtures/ring.ts`, `arrange.test.ts`, stories |
| `plugins/views-plex-graph/frontend/plex/ui` | `Plex.vue`, `props.ts`, `render/PlexNodeNames.vue` (create), `render/PlexNodeView.vue` | Modify / Create | `PlexView.vue`, stories |
| `apps/docs/src/content/docs`, `docs/settings.md` | `plex.mdx`, `reference.md`, `settings.md` | Modify (two settings) | users |
| `features/editor/plex` | `plex-names.feature` | Create | acceptance run |

`apps/landing` keeps its own plex demo and is not touched. The prototype lab folder stays until its own change.

## 3. Negative Invariants

- The kernel applies no limit and no order by relation; the plugin orders and limits. No second RPC and no per-note call to fetch names.
- No name is ever a node, an edge end or a connection; `connections` is unchanged for depth 2.
- No plugin proto and no new wire message beyond the two fields; `generation`, `level`, `via` and boolean fields are not used (`showNames` is a manifest key, not a wire field).
- `lib/` of the plex has no Vue, DOM, clock or text measuring: widths and row heights come in as numbers. `arrangePlex` stays pure and deterministic.
- The `Placement` stays a coordinates strategy: it knows footprints, not names.
- No camera zoom, no change of pill sizes by the names, no "+N" name.
- With `showNames` off, or a graph with no names, every frame, edge and test of #1837 is unchanged.
- Names have no heading list, no menu, no drag; the heading list opens on the pill only.
- No Go SDK read is added (decision 1); the TS SDK carries `depth` and `relatedTo`.
- Imperative names (`getNames`, `getNamesRoom`, `placeNames`, `readNames`); American spelling; comments only for what the code cannot show; files <= 350 lines, functions <= 80 lines, complexity <= 10, <= 4 parameters.
- Tests use invented fixtures, assert behavior (counts, order, relative positions, events), never CSS text or pixels of the rendering, and do not mock the use case, the placement or the arrangement they verify.

## 4. Acceptance Criteria

### Kernel and wire

**AC-1: Depth 1 is unchanged.** Given a vault where `Entropy.md` has a parent, two children, a jump, and the child has a child of its own, the request `{path, depth: 0}` and `{path, depth: 1}` return the same `related` and `connections` as the request without depth did before, and no item has `related_to`.

**AC-2: Depth 2 adds the notes related to each related note.** Given the vault of AC-1 with depth 2, the response holds every note of depth 1 unchanged and, for the child, an item for its child with `related_to` = the child's path and `relation` = child; for the parent, its own parent, its other children (the siblings of the focus among them) and its jumps, each with `related_to` = the parent's path and relation relative to the parent. The focus itself is not an item of any note, and a note related to the focus is an item of every other note it is related to.

**AC-3: Siblings are related notes too.** Given a sibling of the focus that has a child of its own, depth 2 returns that child with `related_to` = the sibling; the sibling's parent (the shared parent) is returned as a parent of the sibling.

**AC-4: Siblings are never an item.** Given a parent with three children, depth 2 returns no item with `relation` = sibling and `related_to` set.

**AC-5: The focus is never an item.** Given a child that links back to the focus as its parent, depth 2 returns no item whose path is the focus.

**AC-6: A note can be an item of several notes.** Given two children that both have the same child, depth 2 returns two items for that path, one for each `related_to`.

**AC-7: No limit.** Given a child with 60 children, depth 2 returns all 60 items.

**AC-8: Connections do not grow.** For the vault of AC-2, `connections` at depth 2 equals `connections` at depth 1.

**AC-9: A note that is not found drops its items.** Given a related note the index does not return, depth 2 returns neither it nor items with `related_to` = its path; an item whose own note is not found is dropped and the others stay.

**AC-10: Invalid depth is refused.** `depth: 3` returns `CodeInvalidArgument`; `depth: 0` and `1` answer.

**AC-11: The order of `related`.** Items without `related_to` come first in today's order; items of depth 2 follow, grouped by `related_to` in the order of the first group.

**AC-12: Types and tab kinds cover the items.** An item for a note of a plugin-declared type carries its `type` and `tab_kind`.

### SDKs and host

**AC-13: The TS call carries the depth.** `listRelatedNotes(path)` sends depth 1; `listRelatedNotes(path, 2)` sends 2; `relatedTo` is mapped from `related_to`, absent when empty (entity test and plugin-service test against a real mapper).

**AC-14: The loader forwards the argument.** A plugin calling `workspace.notes.listRelatedNotes('a.md', 2)` reaches the editor service with depth 2 (`contextServices.test.ts`).

**AC-15: The Go SDK is unchanged.** `libs/plugin/sdk-go` has no diff in this change.

### Plugin

**AC-16: The settings.** `manifest.json` declares `showNames` (boolean, default true) and `namesPerNote` (range, default 6, min 1, max 12); `readNames` reads both from camelCase keys, ignores snake_case keys, and falls back to the defaults for an out-of-range value.

**AC-17: Names are ordered and limited.** Given a related note with 3 parents, 4 children, 2 jumps and (when #1839 has landed) 2 references, `getNames` with limit 6 returns the 3 parents in `above` and the first 3 children in `below`; limit 12 returns all, in the order parents, children, jumps, references; limit 1 returns the first parent only.

**AC-18: Names are not nodes.** `convertNoteGraph` of a response with items of depth 2 returns the same nodes and edges as for depth 1; no node or edge has an id of an item that is not otherwise related to the focus.

**AC-19: A name is the id of its note.** `getNames` ids equal `getNameToken(path)` (`name:<path>`), and `getNamePath` gives the path back, so a click resolves to the note.

**AC-20: The depth follows the setting.** With `showNames` on the tab asks for depth 2; turning it off asks again with depth 1 and the names go; turning it on shows them again; a change of `namesPerNote` changes the names without a new request.

**AC-21: Equal graphs.** `areNoteGraphsEqual` is false for two graphs that differ only in an item's `relatedTo`.

### Plex arrangement (through `arrangePlex`)

**AC-22: No room, no change.** Every fixture of `graphs` arranged with no `room`, and with a `room` of all zeros, equals strictly the frame of #1837.

**AC-23: Names reserve room.** Given `build('A note', { child: 9 })` where every child has 4 names below it and a window large enough, no footprint (pill plus room) of two nodes overlaps, and the distance between the centers of two children in one column is at least pill height plus `room.top + room.bottom`.

**AC-24: Parents' names stand over the parents.** Given `build('A note', { parent: 4 })` with 2 names above each parent, the footprint of each parent is taller on top by `room.top` and the pill is `(room.top − room.bottom)/2` off the footprint center; no other node lies in the strip over the pill.

**AC-25: A group never covers a name.** For every fixture with names in windows 2400×1400, 1200×800, 700×600, no pill of one node overlaps the name strip (top and bottom room) of another.

**AC-26: Names count against the window.** Given the same graph with and without names in a 700×600 window, no more notes are shown with names than without, and `overflow` + shown = arrived in both; every footprint of a shown node lies inside the window less `margin`.

**AC-27: The pill stays the pill.** With names, `width` and `height` of every returned node equal those without names; edges end at the pill.

**AC-28: Same input, same picture.** Every fixture with names arranged twice is strictly equal.

**AC-29: Mixed heights in a group.** Given one relation of 8 nodes where half have 4 names and half none, no overlap, and the nodes without names do not leave the gap of a named one (some pair of centers is closer than the tallest footprint).

### Window

**AC-30: Names are drawn.** `Plex.stories.ts` `WithNames` in Chromium and WebKit: each related note with names shows them as buttons in small grey text, parents over it, the rest under it; the text height of a name is below the title's; the focus pill, with focus size 2, grows and its names (it has none) and the names of other notes keep their size.

**AC-31: A click on a name moves the focus.** Clicking a name emits `chooseName` with its note's id, and the tab then shows that note in the middle.

**AC-32: Hover brings names forward.** Hovering a pill or a name raises the names of that note to full opacity; moving away returns them. The hover of a name opens no heading list; the hover of the pill still opens it.

**AC-33: Names off.** With `showNames` off the story shows no names and the arrangement equals the one without names.

**AC-34: Limit.** With `namesPerNote` 2 a note with five relations shows two names.

**AC-35: Acceptance.** In the Gherkin acceptance feature `features/editor/plex/plex-names.feature` (offline, local vault; it replaces the e2e spec): the plex of a note with a child that has a child shows the grandchild's name under the child; clicking it moves the focus.

## 5. Tests and Stories to Change

- `libs/note/domain/note_graph_test.go`: `RelatedTo` kept by `AddRelatedNote`.
- `libs/core/kernel/usecase/note/related_notes_test.go`, `related_notes_graph_test.go`, `connection_test.go`: add AC-1 to AC-9, AC-11, AC-12 cases; existing calls gain `depth` 1.
- `libs/core/kernel/adapter/window/editor/note` (and `rename_test.go`, `changes_test.go`, `connections_test.go`, `vault_session_test.go`): callers updated; AC-10 and the wire mapping of `related_to`.
- `apps/mobile/bind/mobile_test.go`: re-run only.
- `apps/desktop/editor/src/entities/note/api/notes.test.ts`, `noteWords.test.ts`, `testing/mocks/note.ts`, `features/plugin-host/model/pluginServices.test.ts`: AC-13.
- `libs/plugin/loader/src/hosting/contextServices.test.ts`, `host.bridge.test.ts`: AC-14.
- `plugins/views-plex-graph/frontend/lib/picture.test.ts`: AC-17, AC-18, AC-19, AC-21 (the cases of `convertNoteGraph` that count nodes keep passing).
- `plugins/views-plex-graph/frontend/model/names.test.ts` (new), `setting-keys.test.ts`, `manifest.test.ts`, `index.test.ts`, `usePlexView.test.ts`, `usePlexTab.test.ts`: AC-16, AC-20.
- `plugins/views-plex-graph/frontend/plex/lib/names.test.ts` (new): wrapping, order of rows, room, placement over and under the pill, cut names.
- `plex/lib/arrange/arrange.test.ts`, `layout/clusters.test.ts`, `layout/freeSpot.test.ts`, `layout/admission.test.ts`, `layout/crowding.test.ts`, `layout/spacing.test.ts`, `clusters.acceptance.test.ts`: AC-22 to AC-29 (new cases); existing cases must stay unchanged (AC-22). `fixtures/build.ts` gains a way to give nodes names and `fixtures/ring.ts` implements the new `PlacementContext.room`.
- `plex/ui/Plex.stories.ts`, `PlexMap.stories.ts`, `render/PlexNodeView.stories.ts`, `PlexNodeView.test.ts`, `Plex.test.ts`: AC-30 to AC-34. Run in Chromium and WebKit, one project at a time. No story asserts a pixel or a color value.
- `features/editor/plex/*.feature` and `tests/acceptance`: AC-35 (the Gherkin acceptance features replace the e2e specs of the plex).
- Docs: `apps/docs/src/content/docs/plex.mdx`, `reference.md`, `docs/settings.md`: the two settings.

## 6. Order of Work

1. Contract: proto, regenerate Go and TS, `domain.RelatedTo` (frozen; the owner has approved it).
2. Tester: red tests for AC-1 to AC-35.
3. Coder: kernel, adapter, SDKs, loader, editor service; then plugin model and `picture.ts`; then `names.ts`, footprints in the arrangement; then the UI.
4. Heavy runs only on mridanga at the end; the prototype `lab` is deleted afterwards in its own change.

## 7. Decisions of the Owner (formerly open questions)

1. No Go SDK `ListRelatedNotes` is added; that is a separate issue. Go keeps only what exists. The TS SDK carries `depth` and `relatedTo`.
2. Both settings stay: **Show notes related to related notes** (`showNames`, default on) and **Names per note** (`namesPerNote`, default 6). The boolean key is `showNames` to match `showPills` in the same manifest.
3. Siblings get names too.
4. A name may repeat a note already drawn; only the focus is left out.
5. The limit counts all names of a note together, parents first.
6. Admission tests the footprint (pill plus names).
7. Name text size is `NAME_SCALE = 0.75` of the title size and the opacity of a name is two settings, `namesOpacity` (its note not in focus, default 0.7) and `focusedNamesOpacity` (the pointer over its note or over a name, default 1), range 0 to 1, 0 hiding the names; the size constant is taken from the lab Playground value the owner approved (`levelScale`); the default of 6 is the lab `nextGenerationLimit`.
8. A name wider than the node is cut with an ellipsis, no tooltip.

## 8. Amendments after the owner ran the plugin (2026-11-13)

1. A measured title is never cut by packing: a crowded window closes gaps and leaves notes out, and a box keeps the width its title needs up to the widest box of the settings (`createWidthLookup`). Unmeasured boxes still narrow with the packing.
2. The dot before a title without a pill is removed in every mode, with its option `dotSize`, its custom property and its width reserve.
3. Names opacity: two range settings, see decision 7.
4. The headings that open under a note are drawn over the names and lines of all notes: the names of every note are drawn in one layer under all the nodes, and the headings reserve no layout room.
5. Every number setting with a range is declared `range`, which the settings page draws as a slider with its value.
6. Focus size scales the title text, icon, gap and padding of the focus note together with its pill, with the pill on and off. Its description reads "its pill, title and icon".
7. The manifest lists the settings in this order: references, nestHeadings, headingLimit, showNames, namesPerNote, namesOpacity, focusedNamesOpacity, showPills, pillPaddingSides, pillPaddingEnds, focusSize, lineThickness, outlineThickness, lineCurvature, highlightLinks, animationLength, noteRoom. Groups were not built: only the order; the settings page has no section headers.
8. Names are for the pointer alone. A name has no `tabindex`, no key handler, no focus handler and no focus outline, so the Tab key never reaches one and no key chooses one. `role="button"` and `aria-label` stay on a name as the way tests find it; no screen reader support is built (supersedes AC-30 "buttons" for the keyboard, and AC-31 for a key).
9. A name is clickable only while it is forward. A click on a name whose note is not attended to emits nothing. The pointer over the note, or over the room of its names (the row of each name, drawn or not), brings the names forward, and then a click chooses. The layout keeps the room for names at a resting opacity of 0; they appear on hover (AC-32).
10. `focusedNamesOpacity` is never 0: its range is 0.05 to 1, step 0.05, default 1, and a stored value below 0.05 is read as 0.05. `namesOpacity` keeps 0 to 1, default 0.7, and 0 hides the names at rest (decision 7 is changed for the focused setting only).
11. The names of a note that left the arrangement stay with its pill until the pill is gone, and fade with it. Attention is dropped with a node: a node and its names report that the pointer left when they are removed, a frame without the attended node resets it, and choosing a name puts its note's names back to rest. A note is named once under one owner, in the first relation of the order parents, children, jumps, references; the limit counts distinct notes.
12. `namesPerNote` is clamped to 1 to 12 and rounded to an integer like the other number settings; only a value that is no number takes the default 6 (replaces "A value outside min/max or not a number falls back to the default" of 1.4).
13. A graph and its arrangement input that change in one tick are arranged once. A text is cut between characters as a reader sees them (grapheme clusters), so an emoji or a flag is kept whole or dropped; the cut helper lives in `plex/lib/cutToFit.ts`.
14. The footprint rule is stated once (`getFootprint`, `getPillCenterY` in `layout/placement.ts`) and read by the placement and by the test of a box against the window. The middle of a group is counted in footprints, as its cost is.
15. Frames without names: the typical and the crowded fixture, in no window and in 1200×800 and 700×600, drawn at a fixed and at a measured width, equal the numbers the plex clusters (#1844) produced (`plex/fixtures/clusterFrames.ts`), with no `room` and with a room of nothing (AC-22; the same numbers are the frames without references of #1839 AC-18).
16. The ghost shown while a drag creates a note has the size of a text field, whatever size a note has (`fieldSize.ts`, `PlexView` `ghostSize`), and the line a drag draws from a note or in from outside has the curvature of a connection (`curvature.test.ts`). The fixture generator `build.ts` keeps producing graphs the plex accepts when walked (`build.test.ts`).
17. `libs/plugin/sdk-ts/src/workspace.ts` is changed by this fix in its doc comments only; the types `relatedTo` and `depth` are those of 1.3. `libs/plugin/sdk-go` has no diff (AC-15).
18. `docs/settings.md`, `apps/docs` `plex.mdx` and `reference.md` state the references setting, the two opacity ranges and that names are for the pointer.

19. Setting `highlightLinks` (title "Highlight links of a hovered note", boolean, default on, in the lines group after `lineCurvature`): while the pointer is over a note, every edge of the frame that joins it is drawn in the raised state of a hovered edge (`PlexView` prop `highlightLinks`, `useEdgeLines` parameter `attention`). The owner chose the default on. No edge is raised while the setting is off, while a connection is dragged out of a note, while a note is dragged onto the plex from outside, or while the plex moves (`Plex` passes `highlightLinks && !isMoving`). The raised state is removed when the pointer leaves the note, when the note is not in the frame any more and when the note is unmounted (the attention reset). Edges of names are not raised.

20. Decisions of the owner, final (2026-11-13):
    - The names of the second level include notes already drawn; only the focus is left out. A sibling owns names (its parents, children, jumps and references) but has no siblings of its own.
    - Names are for the pointer alone. `role="button"` and `aria-label` stay on a name as the way tests find it. A name of a note that is not forward cannot be clicked.
    - `focusedNamesOpacity` has a minimum of 0.05; `namesOpacity` may be 0. The room of the names is reserved in the layout at opacity 0, as at any other opacity.
    - A click on the line to a reference opens no link inspector: no record joins the two notes.
    - Hovering a note highlights the lines that join it by its own setting, `highlightLinks`, on by default (19).
    - The headings that open under a note take the text size of a note at rest and do not follow the focus size; their backdrop is nearly opaque; they are drawn over the lifted lines (4). One note has its headings open at a time: a note is closed as soon as another is attended.
    - When the pill is off the handle lies outside the text of the note, beside its box, and not on its edge.
    - The menu of a note offers "Open preset" only on a preset note (`plugins/views-flashcards-studio/manifest.json`, condition on the `plexGraph.context` entry).
