# Architecture decisions

A file here records **one decision**: something that could have been settled another way, that constrains how the code is built, and that is expensive to revisit. One decision, one file. A file holding two decisions is split.

**What is not here.** How the product behaves for a person, and what the words of the domain mean, are specifications rather than decisions. They live beside this folder as plain pages, listed at the end. Measurements live in [performance.md](../performance.md) and nowhere else — an ADR may state a target, never a number it was measured at.

**Every ADR says what it applies to.** This repository holds several applications, and a decision about one of them is not a decision about the product. The header of each says which.

**A rule is written once, in the present tense, as it now stands.** No record narrates its own drafting, and git holds the drafts.

**A decision that has been replaced keeps its text, under a banner.** The section that no longer holds opens with a line naming the record that replaced it and saying what holds now, and that record names this one in its `Supersedes:` or `Amends:` header. A decision that moved to a record of its own leaves nothing behind: nothing was settled differently.

**A decision is in the record once it is on the default branch.** Until then it is its pull request's draft, and a draft is edited in place.

## Reading order

The numbers are identity, not order. This list is the order.

### What is kept, and where

- [ADR-0001 — Files on disk are the source of truth](0001-files-are-the-source-of-truth.md)
- [ADR-0002 — One database for all vaults, outside them](0002-one-database-for-all-vaults.md)
- [ADR-0003 — A vault carries its identity, and application state lives with the application](0003-a-vault-carries-its-identity.md)

### The shape of the code

- [ADR-0004 — A hexagonal core in Go](0004-a-hexagonal-core-in-go.md)
- [ADR-0005 — A client is generated from the protocol](0005-a-client-is-generated-from-the-protocol.md)
- [ADR-0025 — How this application is tested](0025-how-this-application-is-tested.md)
- [ADR-0026 — One name per concept](0026-one-name-per-concept.md)

### The index

- [ADR-0006 — What the index stores](0006-what-the-index-stores.md)
- [ADR-0007 — A schema change is a numbered migration](0007-a-schema-change-is-a-numbered-migration.md)
- [ADR-0008 — A vault is scanned in the background](0008-a-vault-is-scanned-in-the-background.md)
- [ADR-0009 — The vault is watched](0009-the-vault-is-watched.md)

### Search

- [ADR-0010 — A source is text in one table](0010-a-source-is-text-in-one-table.md)
- [ADR-0011 — Text is cut twice](0011-text-is-cut-twice.md)
- [ADR-0012 — A chunk is identified by its text](0012-a-chunk-is-identified-by-its-text.md)
- [ADR-0013 — The vector index stays inside SQLite](0013-the-vector-index-stays-inside-sqlite.md)
- [ADR-0014 — One search, three rankings, merged by rank](0014-one-search-three-rankings.md)

### Books and passages

- [ADR-0015 — A book's text is a cache or an artifact](0015-a-books-text-is-a-cache-or-an-artifact.md)
- [ADR-0016 — A passage is a range of bytes](0016-a-passage-is-a-range-of-bytes.md)

### Recordings

- [ADR-0042 — A recording is a source of its own](0042-a-recording-is-a-source-of-its-own.md)
- [ADR-0043 — A transcript is WebVTT](0043-a-transcript-is-webvtt.md)
- [ADR-0044 — A recording is transcribed without being asked](0044-a-recording-is-transcribed-without-being-asked.md)

### The vault, written

- [ADR-0017 — The application writes to the vault](0017-the-application-writes-to-the-vault.md)
- [ADR-0018 — The note file](0018-the-note-file.md)
- [ADR-0019 — A note is identified by a ULID in its frontmatter](0019-a-note-is-identified-by-a-ulid.md)
- [ADR-0020 — One process, one lifetime](0020-one-process-one-lifetime.md)

### Cards

- [ADR-0027 — The stencil and the deck](0027-the-stencil-and-the-deck.md)
- [ADR-0028 — A card is named by what it holds, and known by a mark](0028-a-card-is-named-by-what-it-holds.md)
- [ADR-0029 — A deck and a stencil are not searched by their text](0029-a-deck-and-a-stencil-are-not-searched-by-their-text.md)
- [ADR-0030 — Review is an application of its own](0030-review-is-an-application-of-its-own.md)
- [ADR-0031 — An answer is an artifact, a schedule is a cache](0031-an-answer-is-an-artifact-a-schedule-is-a-cache.md)
- [ADR-0032 — The reviewer's agent writes only cards](0032-the-reviewers-agent-writes-only-cards.md)
- [ADR-0033 — What the deck is joined to is read beside the card](0033-what-the-deck-is-joined-to-is-read-beside-the-card.md)
- [ADR-0034 — A preset is a note, and a deck points at one](0034-the-preset.md)
- [ADR-0036 — The goal names the budget that closes the day](0036-the-goal-names-the-budget.md)
- [ADR-0037 — A preset says what counts as learned, and a date aims at it](0037-what-counts-as-learned.md)
- [ADR-0038 — What a day's budget is spent on, and in what order](0038-what-a-days-budget-is-spent-on.md)
- [ADR-0039 — A day of the week carries a share of the load](0039-a-day-of-the-week-carries-a-share.md)
- [ADR-0040 — One rule says which day a card lands on](0040-one-rule-says-which-day-a-card-lands-on.md)
- [ADR-0041 — A preset's day is divided over the decks it schedules](0041-a-day-is-divided-over-the-decks.md)
- [ADR-0035 — The review window writes the index](0035-the-review-window-writes-the-index.md)
- [ADR-0045 — The review window reads a vault the index does not carry](0045-the-review-window-reads-a-vault-it-does-not-carry.md)

### Agents

- [ADR-0021 — An agent reaches the vault through tools](0021-an-agent-reaches-the-vault-through-tools.md)
- [ADR-0022 — The agent this application starts is a port](0022-the-agent-this-application-starts-is-a-port.md)

### The interface

- [ADR-0023 — How an interface component is built](0023-how-an-interface-component-is-built.md)
- [ADR-0024 — The component library is shadcn-vue on Tailwind](0024-the-component-library-is-shadcn-vue.md)

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
- [proofreading.md](../proofreading.md) — how a reading and a transcript are put right
- [agents.md](../agents.md) — what the panel's agent can reach
- [starting.md](../starting.md) — what the application says when it cannot start
- [settings.md](../settings.md) — every setting
- [themes.md](../themes.md) — what a theme is
- [performance.md](../performance.md) — every measurement, dated
