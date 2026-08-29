# ADR-0029: A deck and a stencil are not searched by their text

- **Status:** Accepted
- **Date:** 2026-08-28
- **Applies to:** `modules/libs/core` — the index
- **Amends:** ADR-0006 (what the index stores)
- **Related:** ADR-0011, ADR-0013, ADR-0027, ADR-0028

## Context

A deck and a stencil are notes, so everything the index does to a note it does to them: the body is cut into chunks, every chunk is searched by its words and embedded as a vector, and every heading is stored and searched by name.

That was never asked for, and it is expensive in a way that grows with use.

A deck of two hundred cards cut by a stencil of four fields carries about a thousand headings. Three of them are the sections a person made. Two hundred are the cards. **Eight hundred are four distinct strings, each repeated two hundred times** — `Question`, `Answer`, and whatever else the stencil declares, written out under every card. They crowd the name search, and each becomes the part a chunk is filed under, so a passage found inside a card is announced as being under *Answer*.

A stencil's headings are its faces and their two sides. There is nothing there a person looks for.

The chunks are worse than the headings. A deck's body is a few hundred disjoint fragments a person wrote to be recalled one at a time, not prose to be read; a stencil's body is templates full of `{{Field}}`. Both are cut, stored, searched and **embedded** — the one expensive thing the index does.

## Decision

### A deck and a stencil are not cut into chunks

A note of `type: deck` or `type: stencil` contributes no chunk, and therefore no vector. The full search does not reach inside a card, and the semantic search does not either.

A card is found by its heading, which is its question; a stencil is found by its title, which is its file name. That is what a person looks for, and it is what the name search already answers.

### Only what a person wrote is kept as a heading

- **A deck** keeps its sections and its cards — the first and second levels, and nothing below them. Its third level is the stencil's field names written out under every card; deeper than that is a heading standing inside a value, which is a person's own writing about one card and not a division of the deck.
- **A stencil** keeps no heading at all.
- **Every other note** is unchanged.

A card's heading reaches the index without the mark it carries (ADR-0028): the mark is written for the file, not for a person reading a list.

### What is unchanged

A deck and a stencil are still notes in every other way: a row of their own, a title, a type, an identifier, their links resolved and their backlinks answered, a node in the plex with their headings hanging under it, and every rule about writing them.

## Consequences

- **The words inside a card are not findable.** A person looking for a card looks for its question. This is the one thing given up, and it is given up knowingly.
- **A deck stops being the largest thing in the chunk table.** Under the deck's own bound of 8 MiB one file could hold more chunks than a shelf of books, all of them embedded.
- **The name search stops being crowded by one deck.** The four strings a stencil declares no longer stand in it two hundred times each.
- **A passage found in a note is announced under a heading somebody wrote**, because the headings a deck keeps are the ones a person made.
- An index built before this holds chunks, vectors and headings for decks and stencils. They are cleared by the migration that brings this in, and the notes are read again by the next scan. **A vector is addressed by the text it was made from, not by the chunk that asked for it**, so one shared with an ordinary note stays: what is cleared is what nothing else is standing on.

## Alternatives considered

**Keeping the chunks and dropping the vectors.** Rejected: the full search over a deck answers with a fragment out of the middle of a card, which is neither the question nor the answer, and the chunk table carries it for that.

**Keeping the headings and dropping only the chunks.** Rejected: the field names are the largest part of what a deck contributes and the least of what it means. They are the stencil's vocabulary, and the stencil is where they are already written once.

**Storing a card as one chunk, so a card is found whole.** Rejected: it is a second way to search cards, beside the review that exists to show them, and nothing has asked for it. Where it turns out to be wanted, a deck's cards are a thing to search on purpose — not the by-product of treating a deck as prose.

**Leaving a stencil searchable, since there are few of them.** Rejected: a stencil holds `{{Field}}` and two words per face. Few of them is a reason it costs little, not a reason it earns anything.
