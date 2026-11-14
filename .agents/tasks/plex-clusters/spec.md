# Specification: Plex places each relation as a compact cluster

**Task Slug:** `plex-clusters`
**Date:** `2026-11-12`
**Intent:** `intent.md` in the task folder (three open questions, section 6; the spec takes the proposed answer for each and marks where it would change)

All paths are under `plugins/views-plex-graph/frontend/plex/`. Placement skill: [`nm-place-plugin`](../../skills/nm-place-plugin/SKILL.md). Everything new is in `lib/arrange/` (pure, no Vue, no DOM, no clock). The behavior to match is `arrangeClusters`, `placeNear`, `interleave`, `getDirection` and `Room.fits` of the prototype `plex/lab/layout.ts` on branch `proto/plex-clusters` (not on main; nothing of it is copied or deleted here).

## 1. The pipeline after the change

`arrangePlex` (`lib/arrange/arrange.ts`) keeps its order and its `Placement` strategy seam. What changes is how much is admitted and how crowding is measured: both now ask the placement itself, through the window, instead of the per-line limits.

```mermaid
graph TD
  A[resolveOptions → applyRoom] --> R["requestNodes: group by relation, cap at maxPerCluster when no window"]
  R --> C["measureCrowding(options, canHoldAll)"]
  C --> W[createWidthLookup at the packed options]
  W --> D["admitNodes: place, drop the first box outside the window and the rest of its relation, place again"]
  D --> S["findSpacing (unchanged): open focusGap, lineGap, gap while every box stays inside"]
  S --> P["placement.place (clusters by default)"]
  P --> E["routeEdges, separateTitles (unchanged)"]
```

### 1.1 Placement: `clusters` (new, `lib/arrange/layout/clusters.ts`)

`export const clusters: Placement = { name: 'clusters', place: placeClusters }`. It replaces `rowsAndColumns` as the default of `ArrangeInput.placement`.

`placeClusters(nodesByRelation, focus, context)` lays the groups in radial sectors around the focus (radial layout as in Yee et al. 2001 and Brandes & Pich; ego networks drawn in sectors):

1. **Sectors.** `getSectors(counts): Sector[]` (`layout/sectors.ts`). The groups that have notes take sectors clockwise in the order parents, siblings, children, jumps, references; the parents' sector is centered at the top. Each such group gets `MIN_SECTOR_DEGREES = 36` and the rest of the 360 degrees is shared by note count. References come in only when the references setting shows them.
2. **Rings.** Each sector is filled ring by ring outward from the focus. The first ring lies half the focus height plus `focusGap` plus half the tallest footprint of the group from the center; each next ring is one footprint height plus `lineGap` further out. The rings stop where they leave the window (half its diagonal) or at 400 rings without a window.
3. **One ring.** `layRing(index, request)` walks the arc of the sector in 3 px steps, 4 px inside each border. When the arc cannot hold all the notes still to place, it takes as many as fit, packed one after another. When it can, the notes are spread evenly: each starts at the middle of its share of the arc and takes the first free spot from there. Odd rings start half a slot later, so notes of two rings do not stand on one ray. A spot is free when it is inside the window and keeps `gap` and `lineGap` from every box placed (`SpotIndex.isFree`, `layout/spotIndex.ts`).
4. **Lines.** Where a spot would make a line run along another line or through another note (`layout/lines.ts`), a free spot a little along the same arc is taken instead, if one has clear lines. This is a preference only: a note never leaves its sector and is never left out for its lines.
5. **Left out.** A note stays in its sector. The notes that no ring inside the window can hold are set outside the window in the middle of their sector, and `arrangePlex` counts them as not shown.
6. **Free.** `isClear(spot, other, gaps)`: `|Δx| ≥ (w₁ + w₂)/2 + gap − TOLERANCE` or `|Δy| ≥ (h₁ + h₂)/2 + lineGap − TOLERANCE`, `TOLERANCE = 0.5`, so a box built exactly one gap from a neighbor counts as free despite floating-point error.
7. **Output.** Each node keeps its relation, identity and title; `width = context.width(node)`, `height = options.nodeSize.height`, `opacity = 1`, `order` = its index within its relation. The returned array is grouped by relation in `RELATION_NAMES` order, each relation in arrival order — the order today's placement returns, so the drawing order and the tab order of nodes do not change.

Splitting: `clusters.ts` holds `clusters`, `placeClusters` and the ring loop of a sector; `layout/sectors.ts` holds `getSectors` and `layRing`; `layout/spotIndex.ts` holds `SpotIndex` and `isClear`. Each file under 350 lines, each function under 60 lines, complexity 10 and 3 parameters.

### 1.2 Admission (new, `lib/arrange/layout/admission.ts`)

Replaces `canPlace` in `arrange.ts` and the per-line capacity of `limits.ts`.

- `requestNodes(nodes, options): NodesByRelation` — group by relation in arrival order. Without a viewport each relation is cut to `options.maxPerCluster` (default 20, today's 5 × 4 capacity without a window). With a viewport nothing is cut here: the window decides (open question 1; if the owner wants a cap in a window too, it is applied here and nowhere else).
- `admitNodes(requested, lay, isInside): { nodesByRelation, overflow }` where `lay(nodesByRelation) → PlacedNode[]` places at the packed options and the gaps as set, and `isInside(node)` is the window test of `arrange.ts` (`isInsideViewport` per node, less `margin`). Loop: lay the current set; for each relation, keep the prefix of its nodes (arrival order) up to the first one whose box is not inside, and halve the group when its first box is outside (the middle of a smaller group lies nearer the focus); when nothing was cut, stop. Every pass removes at least one node, so it ends. Then the cut notes are taken back (refill): one note of each relation in turn, while the window still holds every box. A note is dropped only when the window has no free place for it. Without a viewport `isInside` is always true and one pass suffices (skip the lay).
- `overflow[relation]` = number arrived − number admitted, set only when above 0 — the same shape `Plex.vue` already prints ("N children not shown").
- This works for any `Placement` (the `ring` fixture too), so the strategy still does not decide how much is shown.

### 1.3 Fit to the window

- **Crowding** (`layout/crowding.ts`): `measureCrowding(options, canHoldAll: (packed: PlexOptions) => boolean)`. Same binary search over `packOptions` (7 halvings), same early returns (no viewport, `squeeze >= 1`, holds at the settings). `canRelationAll` and the import of `getLimits` go. `arrange.ts` supplies `canHoldAll`: lay every requested node at `packed` with the width lookup of `packed` and the gaps as set, and check every box is inside the window.
- **Spacing** (`layout/spacing.ts`): unchanged. `findSpacing` already measures the arrangement it gets; `arrange.ts` passes the admitted nodes and the clusters placement. It opens `focusGap`, then `lineGap`, then `gap`.
- **Room** (#1836): `applyRoom` still widens all three gaps before crowding.

### 1.4 Options (`lib/arrange/options.ts`)

- `LimitOptions` becomes `{ maxPerCluster: number; maxHeadings: number }`. `maxPerLine` and `maxLines` are removed (no manifest setting uses them; only stories and tests do).
- `direction` stays (drop gestures and the edge axis read it, and it sets the side of each group). `isVertical` stays for routing.
- `lineGap` stays: it is the gap between boxes one above another (open question 2).

### 1.5 Routing

`edges/routing.ts` is not changed. `getAxis` still takes the axis from the side of the relation (`vertical` for parents and children, `horizontal` for jumps and siblings). Sibling edges come from the graph as parent → sibling and are routed as today. Lines that pass behind a box of an inner ring are accepted for this issue (open question 3).

## 2. The direction table (`lib/relation.ts`)

The sectors do not read the table; `grows` is the side a drag from the focus makes a relation on (`drop.ts`).

```ts
export interface RelationDescriptor {
  readonly grows: 'up' | 'down' | 'left' | 'right'
  readonly plural: readonly [one: string, many: string]
}

export const RELATIONS = {
  parent: { grows: 'up', plural: ['parent', 'parents'] },
  child: { grows: 'down', plural: ['child', 'children'] },
  jump: { grows: 'left', plural: ['jump', 'jumps'] },
  reference: { grows: 'left', plural: ['reference', 'references'] },
  sibling: { grows: 'right', plural: ['sibling', 'siblings'] },
} as const satisfies Record<PlexRelation, RelationDescriptor>
```

## 3. What is removed

- `lib/arrange/layout/limits.ts` and `limits.test.ts`: deleted (`getLimits`, `Limits`, `RoleLimits`, `getViewportLimits`, `getWidestRow`, `getRelationLimits`).
- `lib/arrange/layout/placement.ts`: `rowsAndColumns`, `line`, `inRows`, `inColumns`, `lines`, `getReach`, `measureWidestWithin`, `LineContext`, `AxisPlacement` deleted. The file keeps `Placement`, `NodesByRelation`, `Widths` and `PlacementContext`, which loses `limits` (`{ options, width }`).
- `arrange.ts`: `canPlace`, the `getLimits` call and the `limits` passed to the placement.
- `crowding.ts`: `canRelationAll`.
- `options.ts`: `maxPerLine`, `maxLines`.
- `index.ts`: exports `clusters` instead of `rowsAndColumns`.

## 4. Blast radius

| Path | File | Action | Consumers |
|---|---|---|---|
| `lib` | `relation.ts` | Modify | `options.ts`, `clusters.ts`, `drop.ts`, `fixtures/build.ts` |
| `lib/arrange/layout` | `clusters.ts`, `sectors.ts`, `spotIndex.ts`, `admission.ts` | Create | `arrange.ts`, `index.ts` |
| `lib/arrange/layout` | `placement.ts`, `crowding.ts` | Modify | `arrange.ts`, `fixtures/ring.ts` |
| `lib/arrange/layout` | `limits.ts`, `limits.test.ts` | Delete | `arrange.ts`, `crowding.ts`, `crowding.test.ts` |
| `lib/arrange` | `arrange.ts`, `options.ts`, `index.ts` | Modify | `model/transition.ts`, `ui/Plex.vue`, `model/sizing/plexScale.ts` |
| `model` | `transition.ts` | Modify: track `maxPerCluster` instead of `maxPerLine`, `maxLines` | `ui/Plex.vue` |
| `ui` | `Plex.stories.ts` | Modify: `maxPerCluster` control, placement mapping `clusters` / `ring` | Storybook |
| `fixtures` | `ring.ts` | Unchanged (implements `Placement`, never read `limits`) | `arrange.test.ts`, stories |

No proto, wire, manifest or setting change. `apps/landing` has its own plex demo and is not touched.

### Tests and stories that pin rows-and-columns and must change

- `lib/arrange/layout/placement.test.ts`: every `describe` but "what arrives back" pins rows and columns ("a row", "a row too long for one line", "a column", "rows and columns together", "how far a column reaches before it clears the rows"). Replaced by `clusters.test.ts` and `spotIndex.test.ts`; the "what arrives back" cases move to `clusters.test.ts` against `clusters`.
- `lib/arrange/layout/limits.test.ts`: deleted with `limits.ts`; its window cases ("nothing reaches past the edge", "the margin is a setting") become admission cases.
- `lib/arrange/layout/crowding.test.ts`: "the loosest packing that fits everything" uses `getLimits` and `canRelationAll`, and "leaves a window alone… twelve to a line and four lines" counts lines; rewritten against `measureCrowding(options, canHoldAll)` with a `canHoldAll` built from `arrangePlex` placement. "leaves the heights, the counts…" checks `maxPerLine`/`maxLines` → `maxPerCluster`.
- `lib/arrange/arrange.test.ts`: "clears the rows a column runs alongside, and only those"; "leaves a row packed to one gap…", "turns one edge of a column towards the focus", "starts each further column beyond the widest of the one before it"; "the lines of a relation are shared out evenly" (both); "order along a line follows the order given" ("lays a row out left to right", "puts the second line further from the focus than the first"); "counts the nodes past the last line" and "does not let a strategy decide how much is shown" (`maxPerLine × maxLines` → `maxPerCluster`); "lets the caller change how many fit on a line"; "uses rows and columns unless told otherwise" → clusters; "passes a strategy only the relations that have nodes in them" (wraps `rowsAndColumns`); "a window that holds the arrangement exactly" (row-shaped window). "puts every jump clear to the left" stays true and stays.
- `lib/arrange/layout/spacing.test.ts`: "a line runs as long as the window holds" (both), "the gap an edge crosses opens first" (eleven children edge to edge in one row), "a gap opens into the room the window leaves" (`stepAxis` reads the first two children as a row). The `findSpacing` unit cases and "what opens still keeps the window" stay.
- `lib/arrange/look.test.ts` (#1836): "places every neighbor where the plex placed it before, within a pixel" and its `TODAY` positions are removed (positions change by design); the width/height, focus, edges (`TODAY_EDGES` stays) and "is the look the arrangement takes…" cases stay. "moves the boxes apart where no window limits them" groups children by row; it is rewritten to compare the smallest box-to-box distance with and without room. "counts the notes that still do not fit as not shown" stays.
- `model/transition.test.ts`: `rowsAndColumns` → `clusters`; "the nodes allowed on a line" and "the lines allowed" → one case "the notes allowed in a group" (`maxPerCluster`).
- `model/sizing/plexScale.test.ts`: the unscaled-key list and the two `maxPerLine`/`maxLines` expectations → `maxPerCluster`.
- `ui/Plex.stories.ts`: `rowsAndColumns` import, the placement mapping, the `maxPerLine` arg and control, and the args of the story with `maxPerLine: 9`. `MakingOne` and `DraggingThemIn` drag 220 px above the focus to make a parent; that point must still be empty (parents of `typical` stay within 220 px; the tester checks it in the play run). `SmallWindow` and `Overcrowded` are visual only.
- `ui/PlexMap.stories.ts`: no position pinned (menus, headings, line ends); re-run only.
- Acceptance `features/editor/plex/*.feature` (they replace the e2e specs of `views.plex.graph`): no scenario reads a position; they check relations. Run only.

## 5. Negative invariants

- `lib/` has no Vue, DOM, clock or text measuring; widths come in as numbers.
- `arrangePlex` stays pure and deterministic: no randomness, stable sort, ties keep candidate order.
- The `Placement` interface stays a coordinates-only strategy: admission and fitting live in `arrange.ts` / `admission.ts` / `crowding.ts`, not in `clusters`.
- No camera zoom, no change of box sizes by the placement, no "+N" node.
- Files ≤ 350 lines, functions ≤ 60 lines, complexity ≤ 10, ≤ 3 params; imperative names (`placeClusters`, `getSectors`, `layRing`, `isFree`, `requestNodes`, `admitNodes`); comments only for what the code cannot show (the 0.5 tolerance).
- Tests use invented fixtures (`fixtures/build.ts`, `fixtures/graphs.ts`), assert behavior on the frame (positions relative to the focus and each other, counts, edges), never CSS or pixels of the rendering, and do not mock the placement or admission they verify.

## 6. Acceptance criteria

Every criterion runs through `arrangePlex` unless it names a unit. The counts and window sizes below were chosen from the prototype's behavior; if one proves false against a faithful port of it, the tester reports it with the frame before changing it. "Window" is `options.viewport`; "inside" means inside the window less `margin`. Overlap is `isOverlapping` of the boxes (`layout/boxes.ts` `getBox(node, 0)`).

### AC-1: No two boxes overlap
- **Given** every fixture in `graphs`, unmeasured and measured by title, with no window and in windows 2400×1400, 1200×800, 700×600, 420×320, and the `room` setting at 0 and 20
- **Then** no two nodes of the frame overlap, the focus included.

### AC-2: Each relation in its sector
- **Given** `build('A note', { parent: 3, sibling: 40, child: 40, jump: 12, reference: 8 })` in windows 1900×1640 and 1200×900
- **Then** no note is left out, no two boxes overlap, and the center of every note lies inside the sector `getSectors` gives its relation.
- **Given** the same graph with names under the notes (`room: 20`), and in a 700×500 window
- **Then** going once around the focus, the notes of each relation form one run.

### AC-3: The sectors
- **Given** `getSectors` of the counts above
- **Then** the sectors are in the order parents, siblings, children, jumps, references, cover 360 degrees, the parents' sector is centered at the top, every sector is at least 36 degrees, and a larger group has the wider sector. A relation without notes has no sector.

### AC-4: A note is left out only when its sector is full
- **Given** the graph of AC-2 in a 700×500 window
- **Then** notes are left out, and every relation with notes left out has a shown note within one box height plus a gap of the border of the window. A graph of 2/6/6/3/2 notes in the same window shows every note.

### AC-5 to AC-8
Removed with the clusters around a middle: the sectors take their place (AC-2 to AC-4).

### AC-9: Overflow is counted, and only the last notes go
- **Given** `graphs.overcrowded` and `build('A note', { parent: 9, child: 40, jump: 9, sibling: 9 })` in a 400×400 window
- **Then** for each relation the shown nodes are the first ones in arrival order; `overflow[relation]` + shown = arrived; at least one relation has overflow; no box lies outside the window; edges to nodes not shown are dropped.
- **Given** no window and a relation of 30 nodes
- **Then** 20 are shown (`maxPerCluster`) and `overflow` is 10; with `maxPerCluster: 5`, 5 and 25.

### AC-10: Boxes narrow and gaps close before a note is left out
- **Given** `build('A note', { parent: 4, child: 12, jump: 4, sibling: 4 })` in windows 680×800, 640×800, 600×800
- **Then** `measureCrowding` with the clusters `canHoldAll` returns the loosest packing (to a 128th) at which every node is inside, and one step looser does not hold them all; a window that holds all at the settings is returned unchanged.

### AC-11: Gaps open in a roomy window and the arrangement stays inside
- **Given** `build('A node', { child: 3 })` in a 2400×1400 window
- **Then** the distance between neighboring children is larger than at the settings and at most `spread` times the gap.
- **Given** `build('A node', { parent: 6, child: 21, jump: 7, sibling: 7 })` with `spread: 4` in windows 2400×1400, 1970×1440, 1200×800, 700×600, 560×480
- **Then** no box lies outside the window.

### AC-12: Siblings join their parent
- **Given** `typical`
- **Then** every sibling has an edge from a shown parent and no edge to the focus; every other neighbor's edge joins it to the focus — the same pairs as `TODAY_EDGES` in `look.test.ts`.

### AC-13: The same input gives the same picture
- **Given** every fixture arranged twice, and `typical` with its relations listed in another order and the nodes of each relation kept in their order
- **Then** the two arrangements of a fixture are strictly equal, and every node of the reordered `typical` has the position it has in the original.

### AC-14: Free spot unit (`spotIndex.ts`)
- **Given** a taken box and a box of the same size exactly one `gap` to its right (and one `lineGap` below)
- **Then** `isFree` is true; at one `gap` less 1 it is false. The grid index answers as the check against every box does.

### AC-15: Lines on a ring
- **Given** one note on a ring whose middle spot has a line that is not clear
- **Then** `layRing` takes a spot beside it on the same ring; where no spot has clear lines, or the window holds only that spot, it takes the middle spot.

### AC-16: Any placement strategy is admitted by the window
- **Given** the `ring` placement and `overcrowded` in a 700×600 window
- **Then** every shown node is inside the window and overflow + shown = arrived per relation; with no window the counts equal `maxPerCluster` per relation, whatever the strategy returns.

### AC-17: The plex in the window
- **Given** `Plex.stories.ts` `Overcrowded` in Chromium and WebKit
- **Then** the status text reads "N … not shown" for the relations left out, and no two `.plex__box` rectangles intersect.
- **Given** `MakingOne` and `DraggingThemIn`
- **Then** they pass unchanged (a drag above the focus still makes a parent).

## 7. Performance check (reported, not a test)

`admitNodes` and `measureCrowding` lay the arrangement several times. The coder measures `arrangePlex` on `crowded`, `overcrowded` and AC-11's full graph in a 1200×800 window on mridanga and reports the time per call in the PR thread; above 16 ms it is raised with the owner before merge.

Measured on mridanga in a 1200×800 window at the time of #1837 (before the names of #1840 and the refill; not measured again since), per `arrangePlex` call: `crowded` 3.8 ms, AC-11's full graph (6/21/7/7) 3.6 ms, `overcrowded` (200 children, 40 jumps) 12.2 ms, down from 1890 ms. The free-spot search keeps placed boxes in a grid; fit checks (`measureCrowding`, `findSpacing`) stop laying at the first box outside the window and admission stops a pass once every relation has met a box outside or is fully placed (`PlacementContext.shouldStop`). The results are unchanged.

## 8. Amendments after #1840 (2026-11-13)

1. A note is dropped only when the window has no free place for it in its sector.
2. Notes cut by the loop are taken back, one of each relation in turn, while the window still holds every box (refill). The loop halves a group whose first box is outside.
3. A measured title is never cut by packing: a box keeps the width its title needs, up to the widest box of the settings. Unmeasured boxes narrow with the packing.
4. Replaced by the sectors of section 1.1 (2026-11-14): there is no start of a group any more.
5. The numbers of section 7 were measured on the code of #1837. The footprints of #1840 (names) and the refill change them; the limit of 16 ms stands.
