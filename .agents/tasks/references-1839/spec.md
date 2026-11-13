# Specification: Plex shows the links written in a note's text as references

**Task Slug:** `references-1839`
**Date:** `2026-11-13`
**Intent:** `intent.md` in the task folder
**Issue:** jiva-studio/numen#1839

Stacked on #1837 (branch `feat/1837-plex-clusters`, PR #1844): paths and names below are those of that branch, and the PR of this task targets it. The wire shape is the one the owner approved in the issue. Placement skills per area: `libs/core/protocol`, `libs/core/client`, `libs/plugin/sdk-ts`, `libs/ui` tokens, `plugins/themes-presets` -> [`nm-place-library`](../../skills/nm-place-library/SKILL.md) (themes: [`nm-place-plugin`](../../skills/nm-place-plugin/SKILL.md)); `libs/core/kernel`, `libs/note` -> [`nm-place-core`](../../skills/nm-place-core/SKILL.md); `apps/desktop/editor` -> [`nm-place-window`](../../skills/nm-place-window/SKILL.md); `plugins/views-plex-graph` -> [`nm-place-plugin`](../../skills/nm-place-plugin/SKILL.md).

## 1. Where the data comes from

Nothing new is stored. The index already keeps every `[[...]]` of a note body as a link with the role `ref` (`libs/note/markdown/links.go`, `domain.RoleRef`), and a `links:` entry of the frontmatter may carry the role `ref` too.

- Outgoing: `noteport.LinkQueries.GetLinks(vault, focus)` returns the links the focus writes, role `ref` among them. `ListRelatedNotes.getRelatedNotes` reads them today and drops them in `getRelation`.
- Incoming: `GetBacklinks(vault, focus)` returns the links other notes write that resolve to the focus. `getOppositeRole` returns `ref` unchanged, so a backlink of role `ref` is a note that links to the focus.
- Dangling links and links resolving into another vault are skipped by the existing `l.ToVault != cache.vault.ID` test; a link of the focus to itself is removed by `remove(path)`.
- Missing: nothing in the index. What is missing is the relation (`getRelation` has no case for `ref`), the direction (nothing carries which end wrote the link), the wire fields, and the plex edge: `getConnections` keeps only `domain.IsConnectionRole` links (parent, child, jump), so a reference has no `Connection` and the plex has no line for it. The kernel does not add a Connection for it (a Connection is the record pair of a relationship a person draws; a reference is not drawn by a person). The plex draws the line from the related note itself (section 5.3).
- `[text](note.md)` links are not indexed (the issue says they stay out). A `ref` entry written in `links:` frontmatter is indexed identically and shows up as a reference; the kernel cannot tell the two apart and does not try (decision 6).

## 2. Contract (frozen first; owner-approved)

### 2.1 Proto (`libs/core/protocol/proto/numen/v1/note.proto`)

```proto
enum Relation { ...; RELATION_REFERENCE = 5; }

// LinkDirection tells which note wrote a text link.
enum LinkDirection {
  LINK_DIRECTION_UNSPECIFIED = 0;
  LINK_DIRECTION_OUTGOING = 1; // the note in focus links to this note
  LINK_DIRECTION_INCOMING = 2; // this note links to the note in focus
  LINK_DIRECTION_BOTH = 3; // each of the two notes links to the other
}

message RelatedNote {
  ...
  // Set for RELATION_REFERENCE only.
  LinkDirection direction = 8;
}
```

`string related_to = 9` is #1840's and is not added. The comment of `Relation` ("A sibling is written nowhere") gains a sentence: a reference is a link in the text of one of the two notes. Regenerate with `make generate` on mridanga; never hand-edit `gen/` or `src/`; `make generate-check` must show no diff.

### 2.2 Go domain (`libs/note/domain/note_graph.go`)

```go
const RelationReference Relation = "reference"

type LinkDirection string
const (
    DirectionOutgoing LinkDirection = "outgoing"
    DirectionIncoming LinkDirection = "incoming"
    DirectionBoth     LinkDirection = "both"
)

type RelatedNote struct {
    NoteRef
    Relation  Relation
    Direction LinkDirection // set for RelationReference only, empty otherwise
}
```

`GetRelationRank`: parent 0, child 1, jump 2, sibling 3, reference 4, default 5. Reference is last, so any hierarchy position wins over a text link (a sibling also wins, decision 3).

### 2.3 TypeScript (`libs/plugin/sdk-ts/src/workspace.ts`)

```ts
export type NoteRelation = 'parent' | 'child' | 'jump' | 'sibling' | 'reference'
// LinkDirection tells which note wrote a text link: the note in focus ('outgoing'), the related note ('incoming'), or each of them ('both').
export type LinkDirection = 'outgoing' | 'incoming' | 'both'
export interface RelatedNote { ...; readonly direction?: LinkDirection }  // present for 'reference' only
```

`LinkDirection` is exported from the SDK index and re-exported by `libs/plugin/loader/src/contract.ts` as `NoteGraph` is. The window mapper (`noteWords.ts`) maps `NOTE_RELATION.REFERENCE` -> `'reference'` and the generated direction enum -> `'outgoing' | 'incoming' | 'both'`; `LINK_DIRECTION_UNSPECIFIED` on a reference is a protocol violation and the note is kept with no `direction` (the plex then draws no arrow). `libs/core/client/src/clients.ts` exports the generated direction enum next to `NOTE_RELATION`.

### 2.4 Go SDK

`libs/plugin/sdk-go` has no related-notes call and no type carrying `Relation` (checked: only `domain.NewRelatedNote` and `CreateInRelation`, both about creating). Parity is therefore met with nothing added: there is no field to keep in step. If the owner wants a Go call it is a separate issue (decision 5).

### 2.5 Plugin setting (`plugins/views-plex-graph/manifest.json`)

```json
{ "key": "references", "type": "enum", "title": "References",
  "description": "Show the notes linked in the text: the ones this note links to, the ones that link to it, both, or none.",
  "default": "both",
  "options": [
    { "value": "off", "title": "Off" },
    { "value": "outgoing", "title": "Outgoing" },
    { "value": "both", "title": "Outgoing and incoming" },
    { "value": "incoming", "title": "Incoming only" } ] }
```

Placed after `noteRoom` (end of the list). Read in `model/look.ts` as `PlexLook.references: Readonly<Ref<ReferenceMode>>` with `type ReferenceMode = 'off' | 'outgoing' | 'incoming' | 'both'`; a stored value that is none of the four falls back to `both`.

## 3. Kernel behavior (`libs/core/kernel/usecase/note/query/related_notes.go`)

- `getRelation` returns `RelationReference` for `RoleRef`. `getRelatedNotes` sets `Direction: DirectionOutgoing` for a reference found in `GetLinks`, `DirectionIncoming` for one found in `GetBacklinks`; any other relation has an empty direction.
- Dedup stays in `relatedNoteSet.add` (lowest rank wins; two references to one path merge their directions). Consequences: a note that is also a parent, child or jump keeps that relation and its empty direction; the sibling pass (`addSiblings`) replaces a reference to a sibling by the sibling; a note linked both ways is one reference with direction `both`: `relatedNoteSet.add` merges two references to one path whose directions differ into `DirectionBoth` (equal directions keep the first).
- Order: references are discovered in the order the other links are (the focus' links in declaration order, then backlinks in query order), so the order of `Related` stays total and deterministic.
- `addSiblings` ignores references (it filters on parent and child).
- Connections: unchanged.
- `convertRelationToWire` maps `RelationReference` -> `RELATION_REFERENCE`; a new `convertDirectionToWire` maps the direction, empty -> `UNSPECIFIED`. Both live in `adapter/window/editor/note/notes.go`.

## 4. The direction table (`plex/lib/relation.ts`)

```ts
reference: { grows: 'left', lean: 'up', plural: ['reference', 'references'] }
```

placed between `jump` and `sibling` (the prototype order, #1837 section 1.1: it is the order of the turns, of `RELATION_NAMES`, and of drawing and tab order). `PlexRelation` in `types.ts` gains `'reference'`. `lean: 'up'` against the jump's `lean: 'down'` is what puts references above left and jumps below left; `clusters.ts` needs no change. Nothing else of the layout changes: crowding, admission, spacing and routing are keyed by relation and take the new one as is. `getAxis` gives a reference a horizontal edge (as a jump).

## 5. Plex plugin

### 5.1 Drop (`plex/lib/arrange/edges/drop.ts`)

`creatable` holds connection roles only (`parent`, `child`, `jump`), and `findRole` returns nothing for any other relation, as it does for `sibling`. So a reference is never created by a gesture and the drop is not a new creation kind. What must hold: dragging to the left, the heading both jump and reference grow in, still means a jump. `getRelationTowards` takes `RELATION_NAMES.find(r => options.direction[r] === heading)`, which is `jump` because it precedes `reference`; the spec makes the dependency explicit by choosing the first relation of the heading **that the caller allows** (`getRelationTowards` gets the `relations` list and filters before `find`), so the result no longer hangs on the order of the table. Lower-left is a jump and upper-left a reference only in where the clusters stand; a drag is read in four headings, not eight (decision 2: `left` is not split).

### 5.2 Setting filter (`lib/picture.ts`, `model/usePlexTab.ts`)

`convertNoteGraph(noteGraph, token, references: ReferenceMode)` omits a related note whose relation is `reference` unless the mode admits its direction: `outgoing` admits `outgoing` and `both`; `incoming` admits `incoming` and `both`; `both` admits all three; `off` none; a reference with no direction is admitted by `both` only. The setting is followed live: changing it recomputes the picture of the open tab with no reload (the `look` ref is read inside the `picture` computed). `areNoteGraphsEqual` also compares `direction`.

### 5.3 The line (`lib/picture.ts`)

For each admitted reference the picture adds one edge `{ from: <focus token>, to: <note token>, arrows }` after the edges of the connections: `arrows: ['to']` for `outgoing` (head at the note), `['from']` for `incoming` (head at the focus), `['from', 'to']` for `both`. No label. A reference with no direction has the edge and no arrow. No edge joins a reference to a note other than the focus.

### 5.4 Look

- Hue: token `--numen-relation-reference`, added to `libs/ui/components/src/shared/tokens/tokens.css` (light-dark pair) and to every theme of `plugins/themes-presets/themes/*.css` that defines `--numen-relation-sibling` (12 files), and to the table in `docs/themes.md`. The hue differs from the other four in each theme (taken from the theme's own palette).
- Background: a pill of a reference is filled with the raised surface mixed with its hue (`color-mix(in oklab, var(--numen-raised), var(--relation-hue) 12%)`, hover as the other pills), set in `PlexNodeView.vue` under `.plex__node--reference`. The outline keeps the hue as for every relation. With `showPills` off the dot has the hue as for every relation.
- `apps/landing` demo has its own plex and is not touched.

## 6. Blast radius

| Path | File | Action | Consumers |
|---|---|---|---|
| `libs/core/protocol/proto/numen/v1` | `note.proto` | Modify | generated Go and TS |
| `libs/core/protocol` | `gen/**`, `src/numen/v1/note_pb.ts` | Regenerate | kernel, client |
| `libs/note/domain` | `note_graph.go`, `note_graph_test.go` | Modify | kernel use cases, adapters |
| `libs/core/kernel/usecase/note/query` | `related_notes.go` | Modify | `adapter/window/editor/note` |
| `libs/core/kernel/usecase/note` | `related_notes_test.go`, `related_notes_graph_test.go` | Modify (new cases) | - |
| `libs/core/kernel/adapter/window/editor/note` | `notes.go`, `wire_test.go` | Modify | editor window |
| `libs/core/kernel/adapter/window/editor` | a new `references_test.go` end-to-end through the connect client | Create | - |
| `libs/core/client/src` | `clients.ts` | Modify: export direction enum | `apps/desktop/editor` |
| `libs/plugin/sdk-ts/src` | `workspace.ts`, `index.ts` | Modify | loader, plex |
| `libs/plugin/loader/src` | `contract.ts` | Modify: re-export | plugins |
| `apps/desktop/editor/src/entities/note/api` | `noteWords.ts`, `noteWords.test.ts`, `notes.test.ts` | Modify | plugin host |
| `libs/ui/components/src/shared/tokens` | `tokens.css` | Modify | all windows |
| `plugins/themes-presets/themes` | 12 `*.css` | Modify | themes |
| `docs` | `themes.md` | Modify | - |
| `plugins/views-plex-graph` | `manifest.json` | Modify: setting | settings UI |
| `plugins/views-plex-graph/frontend` | `types.ts`, `model/look.ts`, `lib/picture.ts`, `model/usePlexTab.ts` (reads the mode), `model/setting-keys.test.ts`, `manifest.test.ts`, `lib/picture.test.ts` | Modify | tab |
| `plugins/views-plex-graph/frontend/plex/lib` | `relation.ts`, `arrange/edges/drop.ts` | Modify | arrange, gestures, stories |
| `plugins/views-plex-graph/frontend/plex/ui/render` | `PlexNodeView.vue` | Modify | stories |
| `plugins/views-plex-graph/frontend/plex/fixtures` | `build.ts`, `graphs.ts` | Modify: `reference` count and a `withReferences` fixture | tests |
| `apps/mobile/bind` | `mobile_test.go` | Check only | see 8 |
| `plugins/views-plex-graph/frontend/plex/ui` | `Plex.stories.ts`, `PlexMap.stories.ts` | Modify: story with references | Storybook |
| e2e `tests/e2e/desktop/tests/plugins/views.plex.graph` | new spec | Create (offline, local vault, invented notes) | - |

## 7. Negative invariants

- No new storage, table, index column or migration; no new RPC; `related_to` is not added.
- The plex never writes a reference; `creatable` stays the three connection roles.
- `lib/` stays pure; `Placement` is not touched; the kernel does not know the plugin setting (the filter is in the plugin).
- No Connection is made for a reference in the kernel.
- Defaults reproduce today's look for a note with no text links; no existing test changes an expectation except those listed in section 9.
- Imperative names: `convertDirectionToWire`, `getRelationTowards`, `admitReference` (the filter of 5.2); events/contracts named as the naming rule says; American spelling.
- Tests use invented fixtures, no CSS-text or pixel tests, and do not mock what they verify.

## 8. Risk found

`apps/mobile/bind/mobile_test.go` waits for `len(relations) == len(wanted)` over the four relations. If its fixture vault has a `[[...]]` link beside the hierarchy, a fifth `RELATION_REFERENCE` appears and the wait never ends. The coder checks the fixture and either keeps the links out of it or adds `RELATION_REFERENCE` to `wanted`.

## 9. Acceptance criteria

### AC-1: The kernel returns an outgoing reference
- **Given** a vault where `Focus` has `[[Alpha]]` in its body and no `links:` entry for it
- **Then** `ListRelatedNotes(Focus)` returns Alpha with relation reference and direction outgoing.

### AC-2: The kernel returns an incoming reference
- **Given** `Beta` has `[[Focus]]` in its body and Focus links to nothing
- **Then** Beta comes back with relation reference and direction incoming.

### AC-3: Explicit relations win
- **Given** a note that is a parent, a child, or a jump of the focus and is also linked in the text of the focus and of itself
- **Then** it comes back once, with that relation and no direction.

### AC-4: Both directions of one note are one note
- **Given** Focus links `[[Alpha]]` and Alpha links `[[Focus]]`
- **Then** Alpha comes back once, as a reference with direction both.

### AC-5: Skipped links
- **Given** a link to a note that does not exist, a link of the focus to itself, and a link to a note of another vault
- **Then** none of them comes back.

### AC-6: Siblings are not shadowed wrongly
- **Given** a sibling of the focus that is also linked in the text
- **Then** a note that is a sibling of the focus and linked in the text comes back once, as a sibling.

### AC-7: Order is stable
- **Given** the same vault asked twice
- **Then** the order of `Related` is equal; references follow the focus' own links in text order, then backlinks.

### AC-8: Wire conversion
- **Given** every domain relation and direction
- **Then** `convertRelationToWire` yields `RELATION_REFERENCE` for the reference and `convertDirectionToWire` yields outgoing, incoming, both and unspecified for `outgoing`, `incoming`, `both` and empty (`CheckProduced` style, every value of the generated enum is produced from one domain value).

### AC-9: End to end through the window RPC
- **Given** the editor window served with a real index over a temp vault
- **When** a client calls `ListRelatedNotes` for a note with a text link and a backlink
- **Then** the response has `RELATION_REFERENCE` items with `LINK_DIRECTION_OUTGOING` and `LINK_DIRECTION_INCOMING`, and a non-reference item has `LINK_DIRECTION_UNSPECIFIED`; `connections` has no record for the reference.

### AC-10: The window maps references
- **Given** a response with a reference item per direction (outgoing, incoming, both, and an unspecified one)
- **Then** `mapNoteGraph` returns relation `'reference'` with direction `'outgoing'`, `'incoming'`, `'both'` and no direction.

### AC-11: The setting values
- **Given** the manifest
- **Then** `references` is an enum with the four values, default `both`; `readLook` reads it from the key `references`, follows changes, and falls back to `both` for a value outside the four.

### AC-12: The picture follows the setting
- **Given** a note graph with an outgoing reference, an incoming one, a mutual one (direction both) and a child
- **Then** `convertNoteGraph` with `off` has no reference node, with `outgoing` the outgoing and the mutual one, with `incoming` the incoming and the mutual one, with `both` all three; the child is in every case.

### AC-13: The line carries the arrow
- **Given** `both`
- **Then** the outgoing reference has an edge focus -> note with `arrows: ['to']`, the incoming one an edge focus -> note with `arrows: ['from']`, the mutual one `arrows: ['from', 'to']`, none has labels, and no edge joins two references.

### AC-14: A changed setting redraws the open tab
- **Given** an open plex tab with references shown
- **When** the setting changes to `off`
- **Then** the picture of the tab loses the references without reloading the note graph.

### AC-15: Equal graphs stay equal
- **Given** two note graphs equal except for the direction of one reference
- **Then** `areNoteGraphsEqual` is false; with the same direction it is true.

### AC-16: Clusters above and below left
- **Given** `build('A note', { reference: 6, jump: 6 })` (the builder gains `reference`) in a 1600x1000 window
- **Then** every reference's right edge is left of the focus's left edge, every jump's too, the mean center of the references is above the focus's center and that of the jumps below it; no two boxes overlap.

### AC-17: No overlap with all five relations
- **Given** every fixture of `graphs` plus one with all five relations, with no window and in windows 1200x800, 700x600, 420x320
- **Then** no two boxes overlap, and the overflow counts of the references are reported like the other relations'.

### AC-18: Nothing changes without references
- **Given** every existing fixture (no reference)
- **Then** the arrangement is strictly equal to the one of the base branch (`look.test.ts` and `arrange.test.ts` stay as they are).

### AC-19: The direction table
- **Given** `RELATIONS`
- **Then** `reference` grows `left` with lean `up`, `jump` grows `left` with lean `down`, `RELATION_NAMES` lists reference between jump and sibling, and `formatCount('reference', 1)` is `1 reference`, 2 is `2 references`.

### AC-20: Dragging left still makes a jump
- **Given** `creatable` of parent, child, jump and the default options
- **Then** `getRelationTowards` of a point left of the focus is `jump`, `resolveDrop` to the left on empty space is a create of a jump, and `getDropRelation` for a node dropped in from the left is a jump; with `creatable` of `child` only the left heading is no relation. No drop ever yields `reference`.

### AC-21: The reference in the window
- **Given** `Plex.stories.ts` with a story of five relations (invented titles) in Chromium and WebKit
- **Then** the reference nodes are in the DOM with `data-relation`/class `reference`, are pressable (a press emits the activate event for the note), the status reads "N references not shown" when some are left out in a small window, and one `.plex__edge-arrow` is drawn per reference edge. The look of the fill is not asserted.

### AC-22: The end of the path
- **Given** a vault of invented notes served by the offline e2e harness, one note with `[[...]]` links to two notes and a third note linking to it
- **When** the plex of that note is opened
- **Then** three reference nodes are shown; with the setting `outgoing` two; with `off` none.

## 10. Tests and stories to change

- Go: `libs/note/domain/note_graph_test.go` (rank order lists reference after sibling), `usecase/note/related_notes_test.go` and `related_notes_graph_test.go` (AC-1 to AC-7), `adapter/window/editor/note/wire_test.go` (AC-8), a new window test (AC-9), `apps/mobile/bind/mobile_test.go` (section 8).
- Vitest: `noteWords.test.ts` (AC-10), `setting-keys.test.ts` and `manifest.test.ts` (AC-11; the list of settings gains `references`), `picture.test.ts` (AC-12, 13, 15), `usePlexTab.test.ts` (AC-14), `relation.test.ts` (AC-19), `arrange/edges/drop.test.ts` and `ui/drag.test.ts`, `ui/gesture.test.ts` (AC-20), `arrange.test.ts` or `clusters.test.ts` (AC-16, 17), `look.test.ts` (AC-18, unchanged), fixtures tests `build.test.ts` (the `reference` count).
- Stories: `Plex.stories.ts` (AC-21), `PlexMap.stories.ts` re-run; `PlexNodeView.test.ts` gains the reference in its relation list (it asserts the hue variable by relation name).
- e2e: one spec under `views.plex.graph` (AC-22).
- Nothing existing changes an expectation.

## 11. Decisions of the owner (2026-11-13)

1. `LINK_DIRECTION_BOTH = 3` is added: a note linked both ways is one reference with direction both, arrowheads at both ends, shown in the outgoing, incoming and both modes.
2. A drag to the left stays a jump; a reference is not created by a drag.
3. A sibling wins over a reference.
4. Setting values `off`, `outgoing`, `both`, `incoming`; default `both`.
5. No Go SDK call is added now (a separate issue).
6. `ref` entries of frontmatter `links:` are accepted as references.
7. The mobile test is fixed (section 8); the landing demo is left alone.

## Amendments after review (2026-11-13)

1. The options of the references setting are titled Off, Outgoing, Incoming and Both, in that order; the values are `off`, `outgoing`, `incoming` and `both`, `both` by default.
2. The line from the focus to a reference is no connection: a click on it, or its context menu, opens no link inspector, because no record joins the two notes. Every other line is unchanged.
3. `docs/themes.md` counts the tokens a theme sets as forty-five, the number of tokens its tables list, all declared in `tokens.css`; `--numen-relation-reference` is one of them.
