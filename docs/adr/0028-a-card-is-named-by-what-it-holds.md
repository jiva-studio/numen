# ADR-0028: A card is named by what it holds, and known by a mark

- **Status:** Accepted
- **Date:** 2026-08-28
- **Applies to:** the vault format — every application that reads or writes one
- **Amends:** ADR-0018 (a third and fourth permitted addition)
- **Supersedes:** ADR-0027 — "The first field is the card's name", "A card is addressed by its deck and its heading", and the level count in "The heading carries the structure"
- **Related:** ADR-0019, ADR-0026, ADR-0027

## Context

ADR-0027 put a stencil's first field in the card's heading, and made the heading what the card is called and what it is addressed by. One place to type the question, one line to read it on, nothing to keep in step.

Three things have run into it.

**A first field wants more than one line.** A dialogue, a stanza, a sentence beside its two variants: the field a person is shown first is not always short. A markdown heading is one line, so a field written there can be nothing else.

**A deck outgrows one list.** Two hundred cards is a list a person scrolls, and the cards fall into groups with nowhere to say so.

**A card's history has to survive being edited.** A schedule is computed from a history of answers, and that history is the one thing in a vault nothing can recreate. Addressed by its heading, a card whose question is corrected is a different card, and what was attached to the old text is attached to nothing. ADR-0027 recorded that as a cost to be paid when review had a reason to care.

The three have one answer between them: what a card is called and what a card *is* are two things, and only the second is written for a machine.

## Decision

### A deck has sections

A first-level heading opens a section. The cards under it are its own until the next one. A section is a name and nothing else: no fields, no stencil, no schedule, no mark.

```markdown
---
type: deck
---

# Roots

## Compost, what is it made of ^k7m2xq9fzp

[[Term]]

### Question

Compost, what is it made of

### Answer

Leaves and peelings, turned and left to rot down
```

A deck need not have sections, and cards may stand before the first one. Two sections may carry one name, and an empty section is kept. A section is not carried from place to place; moving one would mean moving its cards, and nothing has asked for that.

**Text between a section's heading and its first card is the section's, and is kept exactly as it stands** — what a card already has between its stencil's wikilink and its first field, one level up. Nothing lays it out and nothing reads it; it is a person writing about their own deck, and a write puts it back where it was.

**A section is made at the end of the deck, and taking one away takes away its heading and nothing else.** Its cards stay where they stand, under whatever heading is above them now, and its own text stays too: a section is a name, so removing it removes a name. Carrying cards out with it would be a second act wearing one name.

**A card is made at the end of the section it was asked for**, and at the end of the cards standing before the first section where it was asked for none. A person asks by pressing the plus that stands in the run they are looking at, so a card lands where they were looking.

**The format now spends three heading levels.** A value may hold a heading of the fourth level and below, and no higher.

### Every field is written under its own heading

A stencil's fields are third-level headings under the card, **all of them, the first included**, and a field holds whatever a person writes under it over as many lines as they like.

The first field is the stencil's first — `fields[0]` — and no other reading of "first" is meant anywhere in this record. It is what a card's heading is read from, so it stays first: a stencil neither moves it nor takes it away, as ADR-0027 already had it.

A card writing one field twice is a problem against the deck, as it was. There is no longer a separate problem for writing the first field twice: it is the same fault and it is reported once.

### The heading is the first field, read back

A card's heading is the first line of its first field, cut to fit one line. It holds nothing of its own: throw it away and the application writes it again from the field.

- The projection is taken over text whose line endings are normalised, and written back in the endings the file keeps.
- It stops at the first line break, and at a hundred and twenty characters, counted as a person counts them and not as bytes.
- It never cuts inside a `[[wikilink]]`, an embed or a run of emphasis: the cut falls before whichever of those it lands in.
- A first field that is empty, or holds only spaces, projects to a heading of nothing, and the card is drawn by its first field's box like any other.

A person editing the file by hand may leave the two disagreeing. The field is what stands; the next write puts the heading back in step, and nothing is reported, because there is nothing for a person to decide.

**A card whose stencil cannot be read is not reprojected.** Nothing can say which field is first, so the heading is left exactly as it stands. This is the one case where a stale heading survives a write.

### A card carries a mark

At the end of a card's heading stands `^` and ten characters of `0123456789abcdefghjkmnpqrstvwxyz`, and that is what the card is for as long as it exists.

```markdown
## Compost, what is it made of ^k7m2xq9fzp
```

It is written when the card is made. A card typed into a deck by hand carries none until the application next writes that file, which is when it is given one.

**A mark is separated from the heading's text by one space, stands last on the line, and is read as a mark only at that length and in that alphabet.** Anything else at the end of a heading is heading text. A malformed mark is not a mark and the card is given one.

**The mark does not know which deck it is in.** A card moved to another deck, or to another vault, is the same card: the same mark, the same history. Nothing recomputes and nothing is lost.

**Two cards carrying one mark** — a card copied by hand — is a problem against the deck holding them. **Both are read and both are shown, each marked**: nothing a person wrote goes missing from the screen, and which of the two is meant is a thing only they know. Neither is given a new mark, because a machine choosing would be choosing which card keeps the history. Across two decks it is a question for whatever keeps a history, and this record does not answer it.

Ten characters is 1.1 × 10¹⁵ marks. A vault of a hundred thousand cards meets a collision about once in two hundred thousand vaults, which is why the check exists and why it is not a design constraint.

### The mark is not shown

A card's heading reaches the index as a heading and a part, and that text is what a search answers with and what the plex hangs under a deck. The mark is taken off before the heading is recorded: it is written for the file, not for a person reading a list.

### Where a card is made whole

Two of the rules above are the application's to keep, not the caller's: minting a mark for a card that carries none, and putting a stale heading back in step. Neither can be done where a deck's body is composed — one needs a generator, the other needs the card's stencil.

**A deck is made whole once, in the use case that writes it**, before the bytes reach the vault. Every caller — the window, the tools an agent uses — therefore gets the same file, and neither has to know the rules.

### The additions this format makes, named

ADR-0018 permits the application two additions to a note file. A deck and a stencil **are notes** — that is what the `type` key is for — and they carry two more:

- `{{Field}}` in a stencil's face, where a card's value is laid out.
- `^` and a card's mark, at the end of a card's heading in a deck.

Both render as plain text in an editor that never heard of this application, which is the test ADR-0018 sets. ADR-0018 is amended to say so, and `docs/note-format.md` carries them in the table of what may be added.

## Consequences

- **A card can be reordered, renamed and rewritten, and stays itself.** Nothing in a heading is load-bearing.
- **The first field is still the one that stands first**, and ADR-0027's rule that it is neither moved nor removed holds. It is what every card's heading is read from, so putting another field there rewrites the heading of every card in every deck that stencil cuts.
- **Renaming the first field now writes to every deck that stencil cuts**, as renaming any other field already did. It is one more field, not a new kind of cost.
- **The format reports less.** A card with no name and a first field written twice are no longer faults; two cards of one heading is no longer a fault; two cards of one mark is.
- **A deck with no stencil is fully readable.** Every field stands under its own name, so nothing is lost with the stencil — under ADR-0027 the first field's name was unrecoverable.
- **A first field of many lines makes a short heading.** What an outline shows is its first line, which is what an outline is for.
- **Two cards may carry one heading**, where two first fields begin alike. Their marks tell them apart, and nothing else needed them to differ.
- **`[[Deck#^k7m2xq9fzp]]` reaches the deck**, as every fragment does today. Reaching the card itself is work for whoever wants it.
- ADR-0027's rules on how a field is renamed, on the deck's size bound, on what a problem is and on the schedule living outside the file are untouched.

## Alternatives considered

**A name of the card's own, typed by a person.** Rejected: five hundred words is five hundred names nobody wants to invent, and what a person would write is `Card 1`, `Card 2` — a column of identifiers wearing the costume of names.

**The card's place in the deck, written in the heading.** Rejected: a number a reorder rewrites reads as an order, and a number a reorder leaves alone reads as a broken one. The order of the cards is the order of the file.

**Keeping the heading as the field, and a first field of more than one line written out below it as well.** Rejected: the value stands in two places with nothing keeping them in step, which is the fault the format already reports as a field written twice.

**Refusing a first field of more than one line**, as Anki's sort field and Mochi's title field do, and asking a stencil to put something short first. Rejected: it is a rule about how a person must think of their own cards, made to suit the file.

**A mark written at first review**, so a deck nobody has begun to learn holds nothing machine-written. Rejected: the editor writes the deck in the first place, so there is no untouched file to keep clean, and a card with no mark would need a second way to be addressed and a rule for when each applies.

**A ULID**, as a note carries (ADR-0019). Rejected: twenty-six characters at the end of every heading, to buy uniqueness across every vault that will ever exist, where the question is whether two cards in one vault collide. Creation order goes with it, and nothing here asks for creation order.

**A namespace on the mark, `^card/…`.** Rejected: it was needed where a card was a line in an ordinary note among a person's own anchors. It buys nothing where a card is a heading in a deck.

**The mark on the line below, beside the stencil's wikilink.** Rejected: a card's mark and a person's anchor to that card are the same want, and markdown puts an anchor at the end of the block's own line.
