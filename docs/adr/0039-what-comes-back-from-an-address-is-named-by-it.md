# What comes back from an address is named by the address

- **Status:** Accepted
- **Date:** 2026-09-06
- **Applies to:** `modules/libs/core`
- **Related:** [A book's text is a cache or an artifact](0015-a-books-text-is-a-cache-or-an-artifact.md), [A transcript is WebVTT](0031-a-transcript-is-webvtt.md), [A link is a note that carries an address](0038-a-link-is-a-note-that-carries-an-address.md)

## Context

A reading is named by the hash of the bytes it was made from, so a document renamed or moved keeps it and two copies of one document share it.

A link note's bytes are prose somebody is typing. What was fetched has nothing to do with them: it came from an address the note carries, and a sentence added under it changes the file and not the video.

## Decision

### The name is the hash of the address

The one form every spelling of an address reaches, hashed. Everything one fetch produced stands under it, in the store's own folder for what made the files: `captions` for words a site published, `asr` for words a model here heard, `article` for the prose of a page.

Two notes pointing at one video share what was fetched, and a person typing in either of them keeps it.

### An address is one form

The scheme and the host are lowercased, a default port and a fragment go, and the parameters a site adds to say where a visitor came from go with them. A video is reduced to its identifier, so the share link, the watch page and the embed are one address.

Only `http` and `https` are read. A scheme reaching a file or a socket on this machine is refused where the address is read, so nothing further along has to remember to.

### The sweep asks which notes name an address

`notes.address` holds it, so what still points at a file in the store is a question the index answers. A reading is swept by asking whether any source still names it, and this is the same question asked of the same column.

## Consequences

- Editing a link note costs nothing and loses nothing.
- A vault holding one video twice fetches it once.
- The index grows a column and an index over it, which is one migration.
- What was fetched survives the note being renamed, moved, or deleted and
written again.

## Alternatives considered

**The hash of the note's bytes, as a reading is named.** Rejected: every keystroke would orphan what was fetched, and the person typing about a video is the one who loses it.

**The path of the note.** Rejected: a note that moves loses what was fetched for it, and a hash is what every other artifact here is named by.
