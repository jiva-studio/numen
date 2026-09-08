# Architecture decisions

A file here records **one decision**: something that could have been settled another way, that constrains how the code is built, and that is expensive to revisit. One decision, one file. A file holding two decisions is split.

**What is not here.** How the product behaves for a person, and what the words of the domain mean, are specifications rather than decisions. They live beside this folder as plain pages, listed at the end. Measurements live in [performance.md](../performance.md) and nowhere else — an ADR may state a target, never a number it was measured at.

**Every ADR says what it applies to.** This repository holds several applications, and a decision about one of them is not a decision about the product. The header of each says which.

**A rule is written once, in the present tense, as it now stands.** No record narrates its own drafting, and git holds the drafts.

**A decision that was wrong is rewritten where it stands.** The record says what holds now, and git holds what it said before. A rule that turned out to be a mistake is taken out, not left under a banner and worked around by a second record: two records disagreeing is how a reader ends up following the wrong one.

**A decision that has been narrowed by a later one keeps its text, under a banner.** Where a record still holds except in the case another settles differently, the section that no longer holds opens with a line naming that record and saying what holds now, and that record names this one in its `Supersedes:` or `Amends:` header. A decision that moved to a record of its own leaves nothing behind: nothing was settled differently.

**A decision is in the record once it is on the default branch.** Until then it is its pull request's draft, and a draft is edited in place.

## Reading order

The numbers run without gaps, and the whole corpus is renumbered when one closes: a number is a position in the sequence, not an identity, and it changes. A record is named by its title. This list is the reading order, which is not the numbering.

### What is kept, and where

- [Files on disk are the source of truth](0001-files-are-the-source-of-truth.md)
- [One database for all vaults, outside them](0002-one-database-for-all-vaults.md)
- [A vault carries its identity, and application state lives with the application](0003-a-vault-carries-its-identity.md)

### The shape of the code

- [A hexagonal core in Go](0004-a-hexagonal-core-in-go.md)
- [Where a port is declared, and where an adapter stands](0035-where-a-port-is-declared-and-where-an-adapter-stands.md)
- [A client is generated from the protocol](0005-a-client-is-generated-from-the-protocol.md)
- [One service to a subject](0034-one-service-to-a-subject.md)
- [Nothing is logged, and a person is told where they are](0037-nothing-is-logged.md)
- [How this application is tested](0025-how-this-application-is-tested.md)

### The index

- [What the index stores](0006-what-the-index-stores.md)
- [A schema change is a numbered migration](0007-a-schema-change-is-a-numbered-migration.md)
- [A vault is scanned in the background](0008-a-vault-is-scanned-in-the-background.md)
- [The vault is watched](0009-the-vault-is-watched.md)

### Search

- [A source is text in one table](0010-a-source-is-text-in-one-table.md)
- [Text is cut twice](0011-text-is-cut-twice.md)
- [A chunk is identified by its text](0012-a-chunk-is-identified-by-its-text.md)
- [The vector index stays inside SQLite](0013-the-vector-index-stays-inside-sqlite.md)
- [One search, three rankings, merged by rank](0014-one-search-three-rankings.md)

### Books and passages

- [A book's text is a cache or an artifact](0015-a-books-text-is-a-cache-or-an-artifact.md)
- [A passage is a range of bytes](0016-a-passage-is-a-range-of-bytes.md)

### Recordings

- [A recording is a source of its own](0030-a-recording-is-a-source-of-its-own.md)
- [A transcript is WebVTT](0031-a-transcript-is-webvtt.md)
- [A recording is transcribed without being asked](0032-a-recording-is-transcribed-without-being-asked.md)

### Addresses

- [A url is a source of its own](0038-a-url-is-a-source-of-its-own.md)
- [What comes back from an address is named by the address](0039-what-comes-back-from-an-address-is-named-by-it.md)
- [The window may frame the hosts a video plays from](0040-the-window-may-frame-the-hosts-a-video-plays-from.md)
- [Downloading what is at an address](0041-downloading-what-is-at-an-address.md)

### The vault, written

- [The application writes to the vault](0017-the-application-writes-to-the-vault.md)
- [A path is judged by where it lands](0036-a-path-is-judged-by-where-it-lands.md)
- [The note file](0018-the-note-file.md)
- [A note is identified by a ULID in its frontmatter](0019-a-note-is-identified-by-a-ulid.md)
- [One process, one lifetime](0020-one-process-one-lifetime.md)

### Cards

- [The stencil, the deck and the card](0026-the-stencil-and-the-deck.md)
- [An answer is an artifact, a schedule is a cache](0028-an-answer-is-an-artifact-a-schedule-is-a-cache.md)
- [A preset is a note, and one arithmetic schedules it](0029-the-preset.md)
- [Review is an application of its own](0027-review-is-an-application-of-its-own.md)
- [Both windows open a vault through one path](0033-both-windows-open-a-vault-through-one-path.md)

### Agents

- [An agent reaches the vault through tools](0021-an-agent-reaches-the-vault-through-tools.md)
- [The agent this application starts is a port](0022-the-agent-this-application-starts-is-a-port.md)

### The interface

- [How an interface component is built](0023-how-an-interface-component-is-built.md)
- [The component library is shadcn-vue on Tailwind](0024-the-component-library-is-shadcn-vue.md)

## The specifications

What the product does, and what its words mean.

- [glossary.md](../glossary.md) — the ubiquitous language, term by term
- [note-format.md](../note-format.md) — the note file, key by key
- [links.md](../links.md) — the link record, and how a name resolves
- [cards.md](../cards.md) — the stencil and the deck, field by field
- [flashcards.md](../flashcards.md) — running the cards: what is due, and what an answer is
- [editing.md](../editing.md) — a note in a tab: saving, renaming, removing
- [vaults.md](../vaults.md) — several vaults, one window
- [reading.md](../reading.md) — how a book is read
- [transcribing.md](../transcribing.md) — how a recording is heard
- [importing.md](../importing.md) — how what a url points at is fetched
- [proofreading.md](../proofreading.md) — how a reading and a transcript are put right
- [agents.md](../agents.md) — what the panel's agent can reach
- [publishing.md](../publishing.md) — how a build reaches a person
- [starting.md](../starting.md) — what the application says when it cannot start
- [settings.md](../settings.md) — every setting
- [themes.md](../themes.md) — what a theme is
- [performance.md](../performance.md) — every measurement, dated
