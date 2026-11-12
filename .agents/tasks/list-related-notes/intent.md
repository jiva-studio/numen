# Intent: rename GetNeighborhood to ListRelatedNotes

**Task slug:** `list-related-notes`
**Issue:** jiva-studio/numen#1838
**Date:** 2026-11-12

## Job

A client asks the core for the notes related to one note. The call is named `GetNeighborhood` and its items `Neighbor`, words taken from the picture the plex draws. The names should say what comes back: a list of notes related to one note. After this change a reader of the proto, the kernel, the SDKs or a plugin sees `ListRelatedNotes` and `RelatedNote` and knows what the data is without knowing the plex.

## Change

Rename, with no change of behavior:

- proto: `NoteService.GetNeighborhood` → `ListRelatedNotes`, `GetNeighborhoodRequest` / `GetNeighborhoodResponse` → `ListRelatedNotesRequest` / `ListRelatedNotesResponse`, message `Neighbor` → `RelatedNote`; generated Go and TS regenerated with `make generate`.
- Go kernel: the handler, the use cases and the domain types that carry this data.
- TypeScript SDK: `VaultNotes.getNeighborhood(path)` → `listRelatedNotes(path)`, `NoteNeighbor` → `RelatedNote`, `NoteNeighborhood` → `RelatedNotes`.
- Go SDK: it has no call for this today (checked: nothing in `libs/plugin/sdk-go` calls `NoteService`); nothing to rename there.
- every caller: desktop window, plugin loader, plex plugin, mobile test, Go tests.

## Limits

- Wire field names and numbers stay (`focus`, `related`, `focus_type`, `connections`, `focus_tab_kind`; `note`, `relation`, `type`, `tab_kind`). Only service method and message names change.
- No backward compatibility: no alias, no old method kept, no deprecated export.
- What the person sees stays: the plex group's accessible name "Neighborhood", every label, every order of notes.

## Non-goals

- References, and related notes of related notes (separate issues, built on this name).
- The flashcards studio's own deck call `GetDeckNeighborhood` / `DeckNeighbor` and package `usecase/neighborhood`: a plugin-local RPC about a deck, not this core call.
- The plex's drawing types (`PlexNeighborhood`, `assertNeighborhood`, `plex/fixtures/neighborhoods.ts`): they name the picture, not the data (proposal, see spec §1.4).
- Adding a Go SDK call for related notes.
- Comments that name the plex view as "the neighborhood view" (`FocusNote` in `hostwindow.proto`, `sdk-go/host/window.go`, `agents-tools-window` focus tool text, `WatchFocusResponse.path` comment).
- Landing page copy ("its direct neighbors", demo line "note neighborhood").

## Pre-mortem

- A generated file edited by hand drifts from the schema → `make generate-check` fails in CI. Regenerate only.
- A structural type in the window matches the generated client by field name; a missed rename there compiles in one place and fails in another → the desktop typecheck must run.
- The mobile bind test calls the method by its string name `"GetNeighborhood"` → it fails only at run time. Listed explicitly.
- The existing domain type `domain.RelatedNote` (the input for creating a note in relation) already takes the name the issue wants → a blind rename of `domain.Neighbor` to `RelatedNote` does not compile. The spec picks a different Go domain name; owner confirms.

## Invariants

- Every existing suite stays green with only renames in it: no test changes an expectation.
- No old name from the rename table remains outside the listed exceptions.

## Open questions for the owner

1. Go domain name. `libs/note/domain` already has `RelatedNote` (a note to create next to an existing one, aliased in the Go SDK as `host/domain.RelatedNote` and used by `CreateInRelation`). The spec proposes `domain.Neighbor` → `domain.RelatedNoteRef` and leaves the creation input alone. Alternative: rename the creation input (for example to `NewRelatedNote`) and give `RelatedNote` to the related-note item, which widens the change to the Go SDK and `CreateInRelation` callers. Which one?
2. Container names. The spec proposes `domain.Neighborhood` → `domain.RelatedNotes` and TS SDK `NoteNeighborhood` → `RelatedNotes`. Both are an object (focus, related notes, connections), not a list, under a plural name. Accept, or another name?
3. Plex drawing types. The spec keeps `PlexNeighborhood` and the plex-internal drawing names, and renames only what holds the SDK data. Should `PlexNeighborhood` also move (for example to `PlexGraph`) here or in a follow-up issue?
4. Flashcards studio. `GetDeckNeighborhood` / `DeckNeighbor` / `usecase/neighborhood` are left out. Rename them to the same words in a separate issue?
5. Go SDK parity. The TS SDK has `notes.listRelatedNotes`, the Go SDK has none. Add a Go SDK call in a separate issue, or leave it until a Go plugin needs it?
6. The accessible name "Neighborhood" of the plex group (`PlexView.vue`, used by e2e `support/workspace.ts` and `perf.spec.ts`) stays, since it is what the person hears. Confirm.
