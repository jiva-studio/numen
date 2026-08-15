# ADR-0010: An attachment is a link, not a place on disk

- **Status:** Accepted
- **Date:** 2026-08-15
- **Applies to:** the vault format — every application that reads or writes one
- **Related:** ADR-0003, ADR-0011

## Context

A node in the graph shows not only a note but the documents pinned to it: PDFs,
images, audio, links. The obvious implementation expresses that pinning through
the filesystem — the attachment sits in the note's folder — and that is how most
note applications do it.

It is wrong for this model, and the reason is not aesthetic.

## Decision

**Attachment is a link role. It is stored as an edge; the filesystem carries no
meaning.**

```yaml
---
links:
  - to: "[[Thermodynamics]]"
    role: parent
  - to: "asset://a1b2c3d4"
    role: attachment
    note: "Jaynes 1957, the primary paper"
  - to: "https://example.org/paper"
    role: attachment
---
```

The target is always one address (ADR-0011): a note, an asset by content hash, or
a URL. There is no separate mechanism for attachments, and the attachments panel
is a query for `role='attachment'` rather than a directory listing.

### What this buys

- **One file on several nodes.** A PDF attached to both *Entropy* and *Shannon*.
  With folders this is impossible — a file is in one place — and for a model with
  multiple parents (ADR-0003) that is not a nicety but a requirement.
- **Moving is free.** An asset is addressed by the hash of its content, so the
  user rearranging their folders breaks nothing.
- **A URL is an attachment.** `to: "https://…"` with the same role, so bookmarks
  and clipped pages arrive in the same panel with no second mechanism.
- **Provenance is one graph.** A phrase found in a PDF, the chunk it came from,
  the card made from it, and the note it is attached to are all connected.

### Rules

- **What is an asset:** any file in the vault that is not a note, minus obvious
  rubbish (`.DS_Store`, `Thumbs.db`, `desktop.ini`, the service folder,
  zero-length files). There is no rule about location: the hash is computed for
  any file and from then on it is addressable. Expensive processing — text
  extraction, transcription, chunking — is triggered by type and on demand, not
  by discovery.
- **An asset with no attachment edges is normal**, not an error: a file was
  dropped into the vault and not yet pinned. It is indexed, findable, and listed
  under "unattached".
- **Deleting a note does not delete its assets** if anything else references
  them. The check is against the edges, not against folder membership.
- **An asset has two natures**: a node in the graph, and a source for search.
  These are separated in the schema — the asset is the thing; chunks and word
  coordinates are derived from it.

### Where new attachments are put

Layout carries no meaning, so the user organises the vault as they like. The
application configures only where *newly added* attachments are written:
`assets/` at the vault root by default, or beside the note, or in a folder named
after the note for people who want colocation. Existing assets are indexed by
hash from anywhere; the application never moves a file.

## Consequences

**Positive** — as above: one file on many nodes, moves that do not break, URLs
without a second mechanism, one graph for provenance.

**Negative**

- "Where is my file?" cannot be answered from the graph, so the interface must
  always show the resolved path. A user who cannot find their own PDF on disk is
  worse off than one with a rigid folder convention.
- Hashing every non-note file in the vault is real work on a large library, and
  has to be incremental on the same `(path, size, modification time)` key as
  everything else.

## Alternatives considered

**A note as a folder** (`Entropy/index.md` with attachments beside it). Rejected
on five counts: `[[Entropy]]` becomes ambiguous between the folder and the file
inside it; block anchors become three-level addresses; tree traversal gets more
expensive; one file can no longer be attached to several notes — the decisive
one; and other markdown editors see a folder of files rather than a note.
Colocation, the only genuine benefit, is available as a setting for where new
attachments are written, without changing the model.

**A dedicated attachments table.** Rejected: it duplicates the edge model, and
then needs its own resolution, its own dangling-reference handling and its own
query path, to express what edges already express.
