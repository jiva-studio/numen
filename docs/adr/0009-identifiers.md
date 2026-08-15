# ADR-0009: Identifiers for notes and blocks

- **Status:** Accepted
- **Date:** 2026-08-15
- **Applies to:** the vault format — every application that reads or writes one
- **Related:** ADR-0001, ADR-0003, ADR-0011, ADR-0012

## Context

Two different things need addressing: a **note**, which is a file, and a
**block**, which is a line inside one. They are not the same kind of thing, and
giving them one syntax produces an immediate contradiction — a `^id` at the top
of a file is the identifier of the first block, not of the note.

Obsidian needs no note identifier because there, identity *is* the path, and
renaming is handled by rewriting `[[links]]` in the other files. That suffices
when nothing but links is attached to a note. Here, unrecoverable data is
attached — a review log, hand-written link markup — and renaming can happen
outside the application, where a disappearance and an appearance are
indistinguishable and attachments break in silence.

## Decision

| Level | Where | Format | Example |
| --- | --- | --- | --- |
| Note | frontmatter, key `id` | ULID | `id: 01J8F3K2M9QRSTVWXYZ012` |
| Block | anchor at end of a line | 8 characters, base32 without confusable glyphs | `^a7f3d21e` |

**ULID for a note** — its lexicographic order is its chronological order, which
gives creation order for free and is useful the moment anything is sorted or
merged.

**A short identifier for a block** — it appears in text a person reads every
day. The alphabet excludes `0`/`O` and `1`/`l`/`I`; 32⁸ is about 10¹², so a
collision inside one file is not a concern.

### An anchor marks a place, not a thing

The identifier encodes no type. One line may carry a card, an annotated link and
something not yet invented, all at once; what is attached is known by the row
that references the anchor. Encoding the type would mean changing the anchor to
attach a second thing to a line — and an anchor must never change.

Block addressing is the ordinary one: `[[Note#^a7f3d21e]]`. No separate syntax
for "a link to a card" is introduced.

An anchor is unique only within its file, so it is addressed as the pair
`<note id>/<block id>`. Two files each containing `^definition` is normal.

**User-named anchors are allowed**: `^my-definition`, matching `[a-z0-9-]{1,32}`.
The pattern that matches a generated one — exactly eight characters from the
generator's alphabet — is refused, so that "a person named this" stays
distinguishable from "the machine generated this".

### When identifiers are written

**A block anchor is written lazily**, when something is first attached to that
line. The user may rename one; the application updates the references within
that file, which is a local and cheap operation.

**A note identifier is written when the note is created, and never
retroactively.** This is the principle, not an optimisation:

> The application never writes an identifier into a file the user is not
> editing.

Two things follow.

A note created by the application has an identifier from the start.

A note created outside it — vim, `touch`, a script — has none, and **is indexed
in full**: text, headings, search, links by name. It simply cannot be a stable
*target*: nothing can point at it with `note://` and nothing can be attached to
it. The identifier is written the moment the application itself edits that file,
because the user opened and changed it, made a link in it, or made a card in it.

**Cards in a note without an identifier are not indexed.** The line is about
which data survives by being re-derived and which survives only by being
recognised: everything indexed about a note is rebuilt from the file on every
scan, so a rename costs a reindex and nothing else. A card anchors a review log,
which cannot be recomputed from anything — creating one without a stable
identifier means being unable to recognise it later, and the history burns.

Cards sitting in untracked notes are invisible without being reported. They
belong in the "vault problems" view — *N notes contain cards but are not
tracked* — rather than being silently dropped.

**Consequence for the content hash:** because identifiers are never backfilled,
the application never produces a change to a file by itself, so the hash is
taken over the whole file and needs no split between "the user's content" and
"our metadata".

## Consequences

**Positive**

- An external rename or move breaks nothing for a note that has an identifier.
- A vault stays clean for someone who writes mostly outside the application: no
  identifiers appear in files they never opened in it.
- One anchor can carry many attachments over time without ever changing.

**Negative**

- Two classes of note exist, and the interface has to explain why some of them
  cannot be a link target or hold a card. That is a real cost, accepted in
  exchange for never touching files the user did not edit.
- `id:` is visible in frontmatter. Whether to hide it in a properties editor is
  a question for the interface, not for the format.

## Alternatives considered

**Identity by path**, the Obsidian model. Rejected for the reason above:
unrecoverable data is attached, and an external rename breaks it silently.

**Rename detection by content hash.** Not a replacement but a complement: it
works for files that have no identifier yet, and it breaks on the very common
sequence "renamed and immediately edited".

**One identifier scheme for both levels.** Rejected: an identifier at the top of
a file is the first block's, and the levels are genuinely different.

**Backfilling identifiers into every file on first scan.** Rejected: it rewrites
the user's entire vault on first launch, which is both alarming and hostile to
their version history.
