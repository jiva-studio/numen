# ADR-0042: A recording is a source of its own

- **Status:** Accepted
- **Date:** 2026-09-01
- **Applies to:** `modules/libs/core`, `modules/apps/desktop`
- **Related:** ADR-0010, ADR-0015, ADR-0043, ADR-0044

## Context

A vault holds recordings: a lecture, a conversation, a voice note. A recording carries no text a machine can take out of it, and what it says is there once a model has listened. The index files every source under a kind, and the kind is what a question names when it asks about some sources and not others.

`book` already covers a source with text nobody typed here. A recording answers to that description.

## Decision

### A recording is `recording`

`sources.kind` takes a third value. A file is one from its name alone, as every other kind is.

Two things follow from the kind and neither can be had without it.

**A question may ask about recordings.** The kind is read on the search path exactly where the question names the kinds it is about, and there is no way to name recordings while they wear another kind's word.

**A window opens a recording in a player.** Which tab a source opens in is decided from the kind the wire already carries. Under `book` the interface would decide it a second time from the file's name, and one concept would be settled in two places.

### A recording never has a text layer

Nothing reads a recording's own bytes as text. A recording with no transcript is a row with no chunks, found by its name, and that is its ordinary state until a model has listened to it.

## Consequences

- A question can be about recordings, and answers about them are told apart from
answers about books.
- The kinds are a set wherever the scan walks them, and a pass written for one
kind is a pass written for both.
- The wire carries a value older clients do not know, and a client that does not
know it treats the source as unspecified.

## Alternatives considered

**A recording is a `book`.** Rejected: the interface would derive "this is a recording" from the file's extension, which is a decision the core has already made and already sends; and a question could not be about recordings at all.

**A recording is a note whose body is its transcript.** Rejected: the recording is the source, and the words are what a model made of it. A note is what a person typed, and ADR-0001 rests on that difference.
