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
- TypeScript SDK: `VaultNotes.getNeighborhood(path)` → `listRelatedNotes(path)`, `NoteNeighbor` → `RelatedNote`, `NoteNeighborhood` → `NoteGraph`.
- Go SDK: it has no call for this today (checked: nothing in `libs/plugin/sdk-go` calls `NoteService`); nothing to rename there.
- every caller: desktop window, plugin loader, plex plugin, mobile test, Go tests.

## Limits

- Wire field names and numbers stay (`focus`, `related`, `focus_type`, `connections`, `focus_tab_kind`; `note`, `relation`, `type`, `tab_kind`). Only service method and message names change.
- No backward compatibility: no alias, no old method kept, no deprecated export.
- What the person sees stays: every label, every order of notes. The accessible name of the plex group changes from "Neighborhood" to "Related notes".

## Non-goals

- References, and related notes of related notes (separate issues, built on this name).
- The flashcards studio's own deck call `GetDeckNeighborhood` / `DeckNeighbor` and package `usecase/neighborhood`: a plugin-local RPC about a deck, not this core call.
- Adding a Go SDK call for related notes.
- Comments that name the plex view as "the neighborhood view" (`FocusNote` in `hostwindow.proto`, `sdk-go/host/window.go`, `agents-tools-window` focus tool text, `WatchFocusResponse.path` comment).
- Landing page copy ("its direct neighbors", demo line "note neighborhood").

## Pre-mortem

- A generated file edited by hand drifts from the schema → `make generate-check` fails in CI. Regenerate only.
- A structural type in the window matches the generated client by field name; a missed rename there compiles in one place and fails in another → the desktop typecheck must run.
- The mobile bind test calls the method by its string name `"GetNeighborhood"` → it fails only at run time. Listed explicitly.
- The existing domain type `domain.RelatedNote` (the input for creating a note in relation) already takes the name the issue wants → a blind rename of `domain.Neighbor` to `RelatedNote` does not compile. The creation input is renamed to `NewRelatedNote`.

## Invariants

- Every existing suite stays green with only renames in it: no test changes an expectation.
- No old name from the rename table remains outside the listed exceptions.

## Decisions

1. Go domain: `domain.RelatedNote` is the related-note item; the creation input is `domain.NewRelatedNote`.
2. Container: `domain.NoteGraph` and TS `NoteGraph`.
3. `PlexNeighborhood` is `PlexGraph`.
4. The flashcards studio is left out.
5. No Go SDK call here.
6. The accessible name of the plex group is "Related notes".
