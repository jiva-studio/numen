# Flashcards Studio Plugin

Flashcards Studio provides the workspace views for editing flashcard decks, designing card stencils, and configuring SRS review presets. The plugin owns the interface of all three: their tab kinds, models, components, stories and tests live in `frontend/`, and the window names none of them.

## Overview

- **Deck Editor (`deck`)**: Grid-based card manager for viewing, adding, and ordering flashcards.
- **Stencil Editor (`stencil`)**: Card stencil designer for declaring fields, front/back layouts, and templates.
- **Preset Editor (`preset`)**: Spaced repetition scheduler settings and target forgetting curve simulator.

## Layout

- `frontend/deck`, `frontend/stencil`, `frontend/preset`: one tab kind each, with its state, components and tests.
- `frontend/cards`: the decks, stencils and presets as the plugin's `CardsService` and `PresetsService` answer them, and in `frontend/cards/wire` the mapping of a preset's settings (`parseSettings`, `formatSettings`, `goalOf`), the wording of a stop reason (`formatStopReasonMessage`) and of the plugin's error codes (`formatErrorCodeMessage`).
- `proto/numen/flashcards/v1`: the plugin's schemas, package `numen.flashcards.v1`: `CardsService`, `PresetsService`, `FlashcardsService` and the plugin's `ErrorCode`. `buf generate`, run in the plugin directory with the buf workspace of the repository root `buf.yaml`, writes the Go code into `gen/` and the TypeScript into `frontend/gen/`; the generated code is committed.
- `frontend/ui`: the prompt a deck or a stencil puts when its file moved under it.

## What the plugin asks of the window

Each kind registers through `registerTabKind` with the hooks the window asks of a kind: how a tab opens, what it is called, the mark it carries, whether it owes the person an answer about its file (`isAsking`), writing what it owes out (`flushPending`), making the file it edits (`createFile`) and, for the preset, where the window's `preset` command over a file leads (`resolveCommand`).

The open decks and stencils are `FileBuffers` handed to `context.buffers.track`, so the window writes them out when it quits and puts a file that moved under a tab to the person. The presets' pending writes go through `context.buffers.addFlushHandler`. Messages are written through `context.workspace.getMessageWriter`, and the review day is `context.workspace.getToday`.

## Activation

The plugin activates automatically on startup or when opening a deck, stencil, or review preset tab.

## Permissions

`vault:read` and `vault:write`: the plugin reads decks, stencils and presets and writes them back, and the window holds `context.buffers` to `vault:write`.
