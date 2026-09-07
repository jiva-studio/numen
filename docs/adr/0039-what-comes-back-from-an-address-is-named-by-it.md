# What comes back from an address is named by the address

- **Status:** Accepted
- **Date:** 2026-09-06
- **Applies to:** `modules/libs/core`
- **Related:** [A book's text is a cache or an artifact](0015-a-books-text-is-a-cache-or-an-artifact.md), [A transcript is WebVTT](0031-a-transcript-is-webvtt.md), [A url is a source of its own](0038-a-url-is-a-source-of-its-own.md)

## Context

A reading is named by the hash of the bytes it was made from, so a document renamed or moved keeps it and two copies of one document share it.

A url's bytes are the address itself. What was fetched has nothing to do with them: it came from that address, and renaming the file changes the file and not the video.

## Decision

### The name is the hash of the address

The one form every spelling of an address reaches, hashed. Everything one fetch produced stands under it, in the store's own folder for what the files are: `transcript` for words with the times they were said at, `article` for the prose of a page, `copy` for the video itself.

Words are a transcript whoever wrote them down, and the producer stands in the file's name: `transcript/<hash>.captions.vtt` for words a site published with a video, `transcript/<hash>.asr.vtt` for words a model here heard. Both are read, cut and put right by the same code. The prose of a page is an article: it carries no times, and no places on pages either, which is what separates it from a reading.

Two notes pointing at one video share what was fetched, and a person typing in either of them keeps it.

### An address is one form

The scheme and the host are lowercased, a default port and a fragment go, and the parameters a site adds to say where a visitor came from go with them. A video is reduced to its identifier, so the share link, the watch page and the embed are one address.

Only `http` and `https` are read, and only away from this machine. A scheme reaching a file, and a host that is this machine, are refused where the address is read, so nothing further along has to remember to; a site that redirects onto this machine is refused as it lands. An index, a window's own socket and whatever else is listening here answer nobody's paste.

### The sweep asks which sources name it

The fingerprint of the address is the url source's `hash`, as the fingerprint of a document's bytes is a book's. So what still stands on a file in the store is the question a reading is swept by, asked of the same column: whether any source still names it.

The walk that finds a file gone asks it, and everything kept for a hash nothing names is taken out — the words, the prose, the copy. Two urls on one video hold it between them while either stands.

## Consequences

- Renaming a url costs nothing and loses nothing.
- A vault holding one video twice fetches it once.
- Nothing is added to the index: a url stands under its address where a book stands under its bytes.
- What was fetched survives the note being renamed, moved, or deleted and
written again.

## Alternatives considered

**The hash of the note's bytes, as a reading is named.** Rejected: every keystroke would orphan what was fetched, and the person typing about a video is the one who loses it.

**The path of the note.** Rejected: a note that moves loses what was fetched for it, and a hash is what every other artifact here is named by.
