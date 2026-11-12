# Specification: rename GetNeighborhood to ListRelatedNotes

**Task slug:** `list-related-notes`
**Date:** 2026-11-12
**Intent:** `intent.md` in the task folder
**Issue:** jiva-studio/numen#1838

A rename with no change of behavior: the core call that answers the notes related to one note, its messages, the kernel types behind it, the TypeScript SDK call and types, and every caller take names that say what the data is. The wire keeps every field name and number.

## 1. Target names

Placement does not change: every file stays in its layer. Areas and their placement skills: `libs/core/protocol`, `libs/plugin/sdk-ts`, `libs/plugin/loader`, `libs/core/client` → [`nm-place-library`](../../skills/nm-place-library/SKILL.md); `libs/core/kernel`, `libs/note` → [`nm-place-core`](../../skills/nm-place-core/SKILL.md); `apps/desktop/editor` → [`nm-place-window`](../../skills/nm-place-window/SKILL.md); `plugins/views-plex-graph` → [`nm-place-plugin`](../../skills/nm-place-plugin/SKILL.md).

### 1.1 Proto (`libs/core/protocol/proto/numen/v1/note.proto`)

| Old | New |
| :-- | :-- |
| `rpc GetNeighborhood(GetNeighborhoodRequest) returns (GetNeighborhoodResponse)` | `rpc ListRelatedNotes(ListRelatedNotesRequest) returns (ListRelatedNotesResponse)` |
| `message GetNeighborhoodRequest` | `message ListRelatedNotesRequest` |
| `message GetNeighborhoodResponse` | `message ListRelatedNotesResponse` |
| `message Neighbor` | `message RelatedNote` |

Fields and numbers unchanged (`path = 1`; `focus = 1`, `repeated RelatedNote related = 2`, `focus_type = 3`, `connections = 4`, `focus_tab_kind = 5`; `note = 1`, `relation = 2`, `type = 6`, `tab_kind = 7`). Comments that name the data move with it: the file header ("its neighborhood" → "its related notes"), the rpc comment ("ListRelatedNotes lists the notes related to one note, and the note itself."), the `connections` comment ("the focus and its neighbors" → "the focus and its related notes"), the `RelatedNote` comment (pointing at `ListRelatedNotesResponse.connections`). `proto/numen/v1` has no `RelatedNote` message today (checked).

Regenerate, never hand-edit: `make generate` from the repo root (runs `buf generate` in `libs/core/protocol` plus the slot/when/command/event scripts). It rewrites `gen/numen/v1/note.pb.go`, `gen/numen/v1/numenv1connect/note.connect.go`, `src/numen/v1/note_pb.ts`. `make generate-check` must then show no diff. buf and the protoc plugins are on mridanga's PATH, not the laptop's.

Generated names that follow: Go `v1.ListRelatedNotesRequest`, `v1.ListRelatedNotesResponse`, `v1.RelatedNote`, `numenv1connect.NoteServiceListRelatedNotesProcedure`, handler/client method `ListRelatedNotes`; TS `ListRelatedNotesRequestSchema`, `ListRelatedNotesResponseSchema`, `RelatedNoteSchema`, types `ListRelatedNotesResponse`, `RelatedNote`, client method `notes.listRelatedNotes(...)`.

### 1.2 Go kernel and note domain

| File (old → new path) | Old | New |
| :-- | :-- | :-- |
| `libs/note/domain/neighborhood.go` → `related_notes.go` | `type Neighbor` | `type RelatedNote` *(the creation input became `NewRelatedNote`)* |
| same | `type Neighborhood` | `type NoteGraph` |
| same | `func (n *Neighborhood) AddNeighbor(note NoteRef, neighbor Neighbor)` | `func (n *NoteGraph) AddRelatedNote(note NoteRef, related RelatedNote)` |
| `libs/note/domain/neighborhood_test.go` → `related_notes_test.go` | callers | follow |
| `libs/core/kernel/usecase/note/query/neighborhood.go` → `related_notes.go` | `type ShowNeighborhood`, `NewShowNeighborhood` | `type ListRelatedNotes`, `NewListRelatedNotes` |
| same | `getNeighbors`, `neighborSet`, locals `neighbor` | `getRelatedNotes`, `relatedNoteSet`, `related` |
| `libs/core/kernel/usecase/note/query/outline.go` | `type NeighborhoodView` (embeds `domain.Neighborhood`) | `type RelatedNotesView` (embeds `domain.NoteGraph`) |
| same | `type GetNeighborhoodView`, `NewGetNeighborhoodView`, field `neighborhood` | `type GetRelatedNotesView`, `NewGetRelatedNotesView`, field `relatedNotes` |
| `libs/core/kernel/adapter/window/editor/note/calls.go` | `outlineCalls.GetNeighborhood` | `outlineCalls.ListRelatedNotes` |
| `libs/core/kernel/adapter/window/editor/note/notes.go` | `(h *Outline) GetNeighborhood`, `&v1.Neighbor{...}` | `(h *Outline) ListRelatedNotes`, `&v1.RelatedNote{...}` |
| `libs/core/kernel/adapter/window/editor/note/handler.go` | field `getNeighborhood *query.GetNeighborhoodView`, `NewOutline(..., neighborhood, ...)` | field `getRelatedNotes *query.GetRelatedNotesView`, param `relatedNotes` |
| `libs/core/kernel/container/wire_window_routes.go` | `query.NewGetNeighborhoodView(...)` | `query.NewGetRelatedNotesView(...)` |
| `libs/core/kernel/adapter/window/editor/export_test.go`, `serve_test.go` | `NoteUseCases.Neighborhood` | `NoteUseCases.RelatedNotes` |
| `libs/core/kernel/usecase/note/neighborhood_test.go` → `related_notes_test.go`; `neighborhood_graph_test.go` → `related_notes_graph_test.go`; `connection_test.go` | helpers `neighborhoodOf`, test names with `Neighborhood` | `relatedNotesOf`, test names with `RelatedNotes` (e.g. `TestANeighborhoodStaysInsideItsVault` → `TestRelatedNotesStayInsideTheirVault`) |
| `libs/core/kernel/adapter/window/editor/{changes,connections,rename,vault_session}_test.go` | `client.GetNeighborhood`, `v1.GetNeighborhoodRequest`, locals `neighborhood` | `client.ListRelatedNotes`, `v1.ListRelatedNotesRequest`, `related` |
| `apps/mobile/bind/mobile_test.go:122` | `ask("NoteService", "GetNeighborhood", ...)` | `ask("NoteService", "ListRelatedNotes", ...)` |

`domain.Relation`, `GetRelationRank`, `NoteRef`, `RefQueries`, `TabKinds` and the field `Related` keep their names. Doc comments that say "neighborhood"/"neighbor" for this data say "related notes"/"related note".

### 1.3 TypeScript SDK, loader, client, desktop window

| File | Old | New |
| :-- | :-- | :-- |
| `libs/plugin/sdk-ts/src/workspace.ts` | `interface NoteNeighbor` | `interface RelatedNote` |
| same | `interface NoteNeighborhood` | `interface NoteGraph` |
| same | `VaultNotes.getNeighborhood(path): Promise<NoteNeighborhood>` | `VaultNotes.listRelatedNotes(path): Promise<NoteGraph>` |
| same | comments "a neighboring note", "to a neighbor, or two neighbors" | "a related note", "to a related note, or two related notes" |
| `libs/plugin/loader/src/contract.ts` | re-export `NoteNeighborhood` | re-export `NoteGraph` |
| `libs/plugin/loader/src/hosting/guards/servicePermissions.ts` | key `getNeighborhood` | key `listRelatedNotes` |
| `libs/plugin/loader/src/hosting/contextServices.test.ts` | `getNeighborhood` | `listRelatedNotes` |
| `libs/core/client/src/protocol.ts` | `GetNeighborhoodResponseSchema` | `ListRelatedNotesResponseSchema` |
| `apps/desktop/editor/src/entities/note/api/notes.ts` | `getNeighborhood`, `notes.getNeighborhood({ path })` | `listRelatedNotes`, `notes.listRelatedNotes({ path })` |
| `apps/desktop/editor/src/entities/note/api/noteWords.ts` | `NeighborhoodWire`, `mapNeighborhood` | `NoteGraphWire`, `mapNoteGraph` |
| `apps/desktop/editor/src/entities/note/lib/note.ts` | `type Neighborhood = NoteNeighborhood` | `NoteGraph` imported from `@numen/plugin-loader` (re-export, or import under the same name and export it) |
| `apps/desktop/editor/src/entities/note/index.ts` | export `getNeighborhood` | export `listRelatedNotes` |
| `apps/desktop/editor/src/features/plugin-host/model/pluginServices.ts` | `getNeighborhood: (path) => getNeighborhood(path)` | `listRelatedNotes: (path) => listRelatedNotes(path)` |
| `apps/desktop/editor/src/testing/mocks/note.ts` | mock key `getNeighborhood` | `listRelatedNotes` |
| `apps/desktop/editor/src/entities/note/api/{notes,noteWords}.test.ts`, `features/plugin-host/model/pluginServices.test.ts` | callers and `describe('mapNeighborhood')` | follow |

### 1.4 Plex plugin (`plugins/views-plex-graph/frontend`)

Rule for the coder: a name whose value is the SDK data (`NoteGraph`, `RelatedNote`, the call) moves; a name whose value is the drawn graph (`PlexGraph`, `PlexNode`, `PlexEdge`) takes the word `graph`.

| File | Old | New |
| :-- | :-- | :-- |
| `types.ts` | `type Neighbor = NoteNeighbor` | `type RelatedNote` re-exported from `@numen/plugin-sdk` |
| same | `interface Neighborhood` | `interface NoteGraph` |
| same | `interface Neighbors { neighborhood(path) }` | `interface RelatedNoteQueries { listRelatedNotes(path) }` |
| same | `PlexView.neighborhood: Ref<Neighborhood \| null>` | `PlexView.noteGraph: Ref<NoteGraph \| null>` |
| `model/usePlexView.ts` | its own `Neighbors` interface, ref `neighborhood`, `core.neighborhood(path)` | `RelatedNoteQueries`, ref `noteGraph`, `core.listRelatedNotes(path)` |
| `index.ts` | `usePlexView({ neighborhood: (path) => notes.getNeighborhood(path) })` | `usePlexView({ listRelatedNotes: (path) => notes.listRelatedNotes(path) })` |
| `lib/picture.ts` | `convertNeighborhood`, `areNeighborhoodsEqual`, params `neighborhood` | `convertNoteGraph`, `areNoteGraphsEqual`, params `noteGraph` |
| `model/usePlexTab.ts`, `model/kind.ts` | `view.neighborhood`, locals `neighborhood` | `view.noteGraph`, `relatedNotes` |
| `fixtures/index.ts` | `createNeighborhood`, `RelatedNeighborSpec`, `NeighborSpec`, params `neighbors` | `createNoteGraph`, `RelatedNoteSpec`, `RelatedNoteInput`, `relatedNotes` |
| `fixtures/host.ts` | `getNeighborhood:` | `listRelatedNotes:` |
| `ui/PlexGraphView.stories.ts` | `NeighborSpec`, param `neighbors` | `RelatedNoteInput`, `relatedNotes` |
| `words.ts` | comment "The neighborhood is being read" | "The related notes are being read" |
| tests (`lib/picture.test.ts`, `model/usePlexTab.test.ts`, `model/usePlexView.test.ts`, `model/kind.test.ts`) | callers | follow |

Moves with it: `PlexNeighborhood` → `PlexGraph`, the `neighborhood` prop of `Plex.vue` / `plex/ui/props.ts` → `graph`, `PlexGraphView.vue`'s `picture: neighborhood` → `graph`, `plex/lib/neighborhood.ts` (`assertNeighborhood`) → `plex/lib/graph.ts` (`assertGraph`), `plex/fixtures/neighborhoods.ts` → `graphs.ts`, `walk.ts` `createNeighborhood` → `createGraph`, and the arrange/transition parameters. The `aria-label` of the plex group in `plex/ui/render/PlexView.vue` becomes "Related notes".

## 2. Blast radius

| Package / path | Files | Action | Downstream |
| :-- | :-- | :-- | :-- |
| `libs/core/protocol` | `proto/numen/v1/note.proto`; generated `gen/numen/v1/note.pb.go`, `gen/numen/v1/numenv1connect/note.connect.go`, `src/numen/v1/note_pb.ts` | Modify / regenerate | every Go and TS user of `NoteService` |
| `libs/note` | `domain/neighborhood.go` → `domain/related_notes.go`, its test | Rename + modify | kernel `usecase/note/query`, kernel adapter |
| `libs/core/kernel` | `usecase/note/query/{neighborhood→related_notes,outline}.go`; `adapter/window/editor/note/{calls,notes,handler}.go`; `container/wire_window_routes.go`; tests in `usecase/note/` and `adapter/window/editor/` | Rename + modify | desktop app binary, mobile bind |
| `apps/mobile` | `bind/mobile_test.go` | Modify | — |
| `libs/core/client` | `src/protocol.ts` | Modify | desktop window |
| `libs/plugin/sdk-ts` | `src/workspace.ts` | Modify | loader, desktop window, every plugin frontend using `vault.notes` |
| `libs/plugin/loader` | `src/contract.ts`, `src/hosting/guards/servicePermissions.ts`, `src/hosting/contextServices.test.ts` | Modify | desktop window |
| `apps/desktop/editor` | `entities/note/{api/notes.ts,api/noteWords.ts,lib/note.ts,index.ts}`, `features/plugin-host/model/pluginServices.ts`, `testing/mocks/note.ts`, three test files | Modify | plugin host |
| `plugins/views-plex-graph` | files in §1.4 | Modify | — |
| `libs/plugin/sdk-go` | none (no related-notes call exists) | — | — |

## 3. Negative invariants

- No hand edit of `*.pb.go`, `*.connect.go`, `*_pb.ts`; they come from `make generate`.
- No wire field renamed or renumbered; no new field, no removed field.
- No alias, shim or deprecated export with an old name (no backward compatibility).
- No test expectation changes: only identifiers, file names and test names. A test that needs a different expected value is a bug in the rename.
- No user-visible text changes (labels, accessible names, tool descriptions).
- Out of scope stays untouched: `plugins/views-flashcards-studio` (deck neighborhood), `plugins/agents-tools-*`, `apps/landing`, `apps/docs`, `docs/`, the "neighborhood view" comments on `FocusNote` and `WatchFocusResponse`.

## 4. Acceptance criteria

No new acceptance test file: the behavior is covered by the existing suites. The red check before the change is the grep in AC-1 (it lists hits today).

### AC-1: no old name remains

- **Given**: the branch after the change
- **When**: from the repo root, run
  ```sh
  git grep -n -E 'GetNeighborhood|getNeighborhood|NoteNeighbor|ShowNeighborhood|NeighborhoodView|AddNeighbor|neighborSet|getNeighbors|NeighborSpec|NeighborhoodWire|mapNeighborhood|convertNeighborhood|areNeighborhoodsEqual|neighborhoodOf|PlexNeighborhood|assertNeighborhood|createNeighborhood' -- ':!plugins/views-flashcards-studio' ':!.agents/tasks'
  git grep -n -i -E 'Neighborhoods?' -- 'libs/note' 'libs/core' 'libs/plugin' 'apps/desktop' 'apps/mobile' 'plugins/views-plex-graph' ':!.agents/tasks'
  ```
- **Then**: the first command prints nothing; the second prints only the "neighborhood view" prose in `hostwindow.proto`, `workspace.proto` and their generated files, and `sdk-go/host/window.go`, which is out of scope.

### AC-2: the generated code matches the schema

- **Given**: the edited `note.proto`
- **When**: `make generate-check` on mridanga
- **Then**: it exits 0, and `gen/` and `src/` carry `ListRelatedNotes`, `ListRelatedNotesRequest`, `ListRelatedNotesResponse`, `RelatedNote` and no `Neighbor`.

### AC-3: the core answers the same related notes under the new call

- **Given**: the kernel suites that call the method (`changes_test`, `connections_test`, `rename_test`, `vault_session_test`, the `usecase/note` related-notes tests)
- **When**: they run against `ListRelatedNotes`
- **Then**: they pass with their expectations unchanged.

### AC-4: plugins reach the call as `notes.listRelatedNotes`

- **Given**: the loader permission guard, the desktop plugin services and the plex plugin
- **When**: their unit suites and the plex e2e specs run
- **Then**: they pass with expectations unchanged; a plugin without the read permission is refused `listRelatedNotes` as it was refused `getNeighborhood`.

## 5. Test plan (all on mridanga, once, after the rename is complete)

1. `make generate-check`.
2. Go: `go test ./domain/...` in `libs/note`; `go test ./usecase/note/... ./adapter/window/editor/... ./container/...` in `libs/core/kernel`; `go build ./...` in `libs/core/protocol` and `libs/plugin/sdk-go`; `go test ./bind/...` in `apps/mobile`; `make compile` for every module that imports `numenv1`; `make lint` targets for `libs/note` and `libs/core/kernel`.
3. TS typecheck: `libs/core/protocol`, `libs/core/client`, `libs/plugin/sdk-ts`, `libs/plugin/loader`, `apps/desktop/editor` (after `npm run build` in `libs/ui/core`, then `libs/ui/shell`), `plugins/views-plex-graph`.
4. TS unit: `libs/plugin/loader` and `apps/desktop/editor` `npm run test:unit`; `plugins/views-plex-graph` `npm run test:unit` (covers its stories: run chromium and webkit projects one at a time if it has them).
5. e2e (offline): `tests/e2e/desktop/tests/plugins/views.plex.graph/` (`graph`, `hierarchy`, `links`, `search-menu`), `tests/e2e/desktop/tests/shell/lifecycle/perf.spec.ts`.
6. AC-1 grep.
